package ddl

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"hash/maphash"
	"io"
	"io/fs"
	"math"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bokwoon95/sq"
)

// MigrateCmd implements the `sqddl migrate` subcommand.
type MigrateCmd struct {
	// (Required) DB is the database to apply migrations to.
	DB *sql.DB

	// (Required) Dialect is the database dialect.
	Dialect string

	// (Required) DirFS is the migration directory.
	DirFS fs.FS

	// Filenames specifies the list of migration scripts within the migration
	// directory to be applied. If a provided filename has already been
	// applied, it will not be applied again.
	//
	// If Filenames is empty, all migration scripts in the Dir will be added to
	// the list.
	//
	// The order in which the filenames are provided is honoured, except for
	// repeatable migrations (files inside the "repeatable/" directory) which
	// are always run after the regular migrations.
	Filenames []string

	// Stderr specifies the command's standard error. If nil, the command
	// writes to os.Stderr.
	Stderr io.Writer

	// HistoryTable is the name of the migration history table. If empty, the
	// default history table name will be "sqddl_history".
	HistoryTable string

	// LockTimeout specifies how long to wait to acquire a lock on a table
	// before bailing out. If empty, 1*time.Second is used.
	LockTimeout time.Duration

	// Maximum number of retries on lock timeout.
	MaxAttempts int

	// Maximum delay between retries.
	MaxDelay time.Duration

	// Base delay between retries.
	BaseDelay time.Duration

	// Verbose will include the start and end timestamps of each migration in
	// the log output.
	Verbose bool

	// Run migrations without adding them to the history table.
	SkipHistoryTable bool

	// Ctx is the command's context.
	Ctx context.Context

	db     string        // -db flag.
	buf    *bytes.Buffer // Reusable buffer. Make sure to Reset() before use.
	driver Driver
}

// MigrateCommand creates a new MigrateCmd with the given arguments. E.g.
//
//   sqddl migrate -db <DATABASE_URL> -dir <MIGRATION_DIR> [FLAGS] [FILENAMES...]
//
//   MigrateCommand("-db", "postgres://user:pass@localhost:5432/sakila", "-dir", "./migrations")
func MigrateCommand(args ...string) (*MigrateCmd, error) {
	var cmd MigrateCmd
	var dir, lockTimeout, maxDelay, baseDelay string
	flagset := flag.NewFlagSet("", flag.ContinueOnError)
	flagset.StringVar(&cmd.db, "db", "", "(required) Database URL/DSN.")
	flagset.StringVar(&dir, "dir", "", "(required) Migration directory.")
	flagset.StringVar(&cmd.HistoryTable, "history-table", "sqddl_history", "Name of migration history table.")
	flagset.StringVar(&lockTimeout, "lock-timeout", "1s", "How long to wait to acquire a lock on a table before timing out.")
	flagset.IntVar(&cmd.MaxAttempts, "max-attempts", 10, "Maximum number of retries on lock timeout.")
	flagset.StringVar(&maxDelay, "max-delay", "5m", "Maximum delay between retries.")
	flagset.StringVar(&baseDelay, "base-delay", "1s", "Base delay between retries.")
	flagset.BoolVar(&cmd.Verbose, "verbose", false, "Log start and end timestamps for each migration.")
	flagset.BoolVar(&cmd.SkipHistoryTable, "skip-history-table", false, "Run migrations without adding them to the history table.")
	flagset.Usage = func() {
		fmt.Fprint(flagset.Output(), `Usage:
  sqddl migrate -db <DATABASE_URL> -dir <MIGRATION_DIR> [FLAGS] [FILENAMES...]
  sqddl migrate -db 'postgres://username:password@localhost:5432/sakila' -dir ./migrations
  sqddl migrate -db 'postgres://username:password@localhost:5432/sakila' -dir ./migrations 01_init.sql 02_data.sql
Flags:
`)
		flagset.PrintDefaults()
	}
	err := flagset.Parse(args)
	if err != nil {
		return nil, err
	}
	if cmd.db == "" {
		return nil, fmt.Errorf("-db empty or not provided")
	}
	if dir == "" {
		return nil, fmt.Errorf("-dir empty or not provided")
	}
	var driverName, dsn string
	cmd.Dialect, driverName, dsn = normalizeDSN(cmd.db)
	if cmd.Dialect == "" {
		return nil, fmt.Errorf("could not identity dialect for -db %q", cmd.db)
	}
	cmd.DB, err = sql.Open(driverName, dsn)
	if err != nil {
		return nil, err
	}
	cmd.DirFS = os.DirFS(dir)
	if lockTimeout != "" {
		cmd.LockTimeout, err = time.ParseDuration(lockTimeout)
		if err != nil {
			return nil, err
		}
	}
	if maxDelay != "" {
		cmd.MaxDelay, err = time.ParseDuration(maxDelay)
		if err != nil {
			return nil, err
		}
	}
	if baseDelay != "" {
		cmd.BaseDelay, err = time.ParseDuration(baseDelay)
		if err != nil {
			return nil, err
		}
	}
	cmd.Filenames = flagset.Args()
	for i, filename := range cmd.Filenames {
		cmd.Filenames[i] = normalizeFilename(filename, dir)
	}
	return &cmd, nil
}

// Run runs the MigrateCmd.
func (cmd *MigrateCmd) Run() error {
	if cmd.DB == nil {
		return fmt.Errorf("nil DB")
	}
	if cmd.Dialect == "" {
		return fmt.Errorf("empty Dialect")
	}
	if cmd.DirFS == nil {
		return fmt.Errorf("nil Dir")
	}
	if cmd.Stderr == nil {
		cmd.Stderr = os.Stderr
	}
	if cmd.HistoryTable == "" {
		cmd.HistoryTable = "sqddl_history"
	}
	if cmd.LockTimeout == 0 {
		cmd.LockTimeout = time.Second
	}
	if cmd.MaxAttempts == 0 {
		cmd.MaxAttempts = 10
	}
	if cmd.MaxDelay == 0 {
		cmd.MaxDelay = 5 * time.Minute
	}
	if cmd.BaseDelay == 0 {
		cmd.BaseDelay = time.Second
	}
	if cmd.db != "" {
		defer cmd.DB.Close()
	}
	if cmd.Ctx == nil {
		cmd.Ctx = context.Background()
	}
	cmd.buf = bufpool.Get().(*bytes.Buffer)
	cmd.buf.Reset()
	defer bufpool.Put(cmd.buf)
	cmd.driver, _ = getDriver(cmd.Dialect)

	var err error
	if len(cmd.Filenames) == 0 {
		cmd.Filenames, err = walkDir(cmd.DirFS)
		if err != nil {
			return err
		}
	} else {
		err = validateFilesExist(cmd.DirFS, cmd.Filenames)
		if err != nil {
			return err
		}
	}
	cmd.Filenames = sortAndFilterFilenames(cmd.Filenames)
	migrations := make([]migration, len(cmd.Filenames))
	cache := make(map[string]int)
	for i, filename := range cmd.Filenames {
		migrations[i] = migration{filename: filename}
		cache[filename] = i
	}

	if !cmd.SkipHistoryTable {
		err = ensureHistoryTableExists(cmd.Dialect, cmd.DB, cmd.HistoryTable)
		if err != nil {
			return err
		}
		cursor, err := sq.FetchCursor(cmd.DB, sq.
			Queryf(`SELECT {*} FROM {} WHERE filename IN ({})`, sq.Identifier(cmd.HistoryTable), cmd.Filenames).
			SetDialect(cmd.Dialect),
			migration{}.rowmapper,
		)
		if err != nil {
			return err
		}
		defer cursor.Close()
		for cursor.Next() {
			m, err := cursor.Result()
			if err != nil {
				return err
			}
			if i, ok := cache[m.filename]; ok {
				migrations[i] = m
			}
		}
	}

	conn, err := cmd.DB.Conn(cmd.Ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	restoreSessionValue := func() error { return nil }
	seconds := strconv.Itoa(int(math.Ceil(cmd.LockTimeout.Seconds())))
	milliseconds := strconv.FormatInt(cmd.LockTimeout.Milliseconds(), 10)
	switch cmd.Dialect {
	case DialectPostgres:
		restoreSessionValue, err = setSessionValue(cmd.Ctx, conn, "SHOW lock_timeout", "SET lock_timeout = %s", milliseconds)
	case DialectMySQL:
		restoreSessionValue, err = setSessionValue(cmd.Ctx, conn, "SELECT @@lock_wait_timeout", "SET lock_wait_timeout = %s", seconds)
	case DialectSQLServer:
		restoreSessionValue, err = setSessionValue(cmd.Ctx, conn, "SELECT @@LOCK_TIMEOUT", "SET LOCK_TIMEOUT %s", milliseconds)
	}
	if err != nil {
		return err
	}
	defer restoreSessionValue()

	queue := make([]migration, 0, len(migrations))
	for _, m := range migrations {
		isRepeatable := strings.HasPrefix(m.filename, "repeatable/")
		if !isRepeatable && m.valid && m.success {
			continue
		}
		if isRepeatable {
			file, err := cmd.DirFS.Open(m.filename)
			if err != nil {
				return err
			}
			cmd.buf.Reset()
			_, err = cmd.buf.ReadFrom(file)
			file.Close()
			if err != nil {
				return err
			}
			var checksum string
			hash := sha256.Sum256(bytes.ReplaceAll(cmd.buf.Bytes(), []byte("\r\n"), []byte("\n")))
			checksum = hex.EncodeToString(hash[:])
			if checksum == m.checksum && m.valid && m.success {
				continue
			}
			m.checksum = checksum
		}
		if len(queue) == 0 {
			queue = append(queue, m)
			continue
		}
		if cmd.Dialect == DialectMySQL ||
			strings.HasSuffix(queue[0].filename, ".tx.sql") ||
			strings.HasSuffix(queue[0].filename, ".txoff.sql") ||
			strings.HasSuffix(m.filename, ".tx.sql") ||
			strings.HasSuffix(m.filename, ".txoff.sql") {
			err = cmd.runWithRetry(conn, queue)
			if err != nil {
				return err
			}
			queue = queue[:0]
		}
		queue = append(queue, m)
	}
	// Flush the queue.
	if len(queue) > 0 {
		err = cmd.runWithRetry(conn, queue)
		if err != nil {
			return err
		}
	}

	return restoreSessionValue()
}

type migration struct {
	valid       bool          // Indicates whether the migration is in the history table.
	filename    string        // Migration filename.
	checksum    string        // Migration file checksum.
	startedAt   sql.NullTime  // When the migration started at.
	timeTakenNs sql.NullInt64 // How long the migration took (in nanoseconds).
	success     bool          // Whether the migration was successful.
}

func (m migration) rowmapper(row *sq.Row) migration {
	m.valid = true
	m.filename = row.String("filename")
	m.checksum = row.String("checksum")
	m.startedAt = row.NullTime("started_at")
	m.timeTakenNs = row.NullInt64("time_taken_ns")
	m.success = row.Bool("success")
	return m
}

// https://www.reddit.com/r/golang/comments/ntyi7i/what_is_the_reason_go_chose_to_use_a_constant_as/h0w0tu7/
var rng = rand.New(rand.NewSource(int64(new(maphash.Hash).Sum64())))

func (cmd *MigrateCmd) runWithRetry(conn *sql.Conn, queue []migration) error {
	isTx := len(queue) > 1 || (!strings.HasSuffix(queue[0].filename, ".txoff.sql") && cmd.Dialect != DialectMySQL) || strings.HasSuffix(queue[0].filename, ".tx.sql")
	isStmt := false
	if !isTx && len(queue) == 1 {
		file, err := cmd.DirFS.Open(queue[0].filename)
		if err != nil {
			return err
		}
		cmd.buf.Reset()
		_, err = cmd.buf.ReadFrom(file)
		file.Close()
		if err != nil {
			return err
		}
		b := bytes.TrimSpace(cmd.buf.Bytes())
		// If there is only one semicolon in the file and it appears at the
		// very end, consider it to be a single SQL statement.
		if bytes.Count(b, []byte(";")) == 1 && b[len(b)-1] == ';' {
			isStmt = true
		}
	}
	isRetryable := isTx || isStmt

	attempts := 0
	for {
		attempts++
		stoppedAt, migrationErr := cmd.run(conn, queue)
		if migrationErr != nil && isRetryable && cmd.driver.IsLockTimeout != nil && cmd.driver.IsLockTimeout(migrationErr) {
			if attempts >= cmd.MaxAttempts {
				fmt.Fprintf(cmd.Stderr, queue[stoppedAt].filename+": attempt %d/%d timed out\n", attempts, cmd.MaxAttempts)
				return fmt.Errorf("%s: %w", queue[stoppedAt].filename, migrationErr)
			}
			multiplier := int(math.Exp2(float64(attempts)))
			delay := time.Duration(rng.Intn(int(cmd.BaseDelay*time.Duration(multiplier)/time.Second))) * time.Second
			if delay < cmd.BaseDelay {
				delay = cmd.BaseDelay
			} else if delay > cmd.MaxDelay {
				delay = cmd.MaxDelay
			}
			fmt.Fprintf(cmd.Stderr, queue[stoppedAt].filename+": attempt %d/%d timed out, retrying in %s\n", attempts, cmd.MaxAttempts, delay.String())
			time.Sleep(delay)
			continue
		}
		if migrationErr != nil {
			if !isRetryable {
				migrationErr = cmd.undo(conn, queue[0], migrationErr)
			}
			if stoppedAt < 0 {
				return migrationErr
			}
			if cmd.SkipHistoryTable {
				return fmt.Errorf("%s: %w", queue[stoppedAt].filename, migrationErr)
			}
			bi := BatchInsert{
				Dialect:    cmd.Dialect,
				TableName:  cmd.HistoryTable,
				Columns:    []string{"filename", "checksum", "started_at", "time_taken_ns", "success"},
				KeyColumns: []string{"filename"},
			}
			i := 0
			_, err := bi.ExecContext(cmd.Ctx, conn, func(row []any) error {
				if i > stoppedAt {
					return io.EOF
				}
				m := queue[i]
				row[0] = m.filename  // filename
				row[1] = m.checksum  // checksum
				row[2] = m.startedAt // started_at
				if cmd.Dialect == DialectSQLite && m.startedAt.Valid {
					row[2] = m.startedAt.Time.UTC().Format("2006-01-02 15:04:05")
				}
				row[3] = m.timeTakenNs // time_taken_ns
				row[4] = false         // success
				i++
				return nil
			})
			if err != nil {
				fmt.Fprintln(cmd.Stderr, err.Error())
			}
			return fmt.Errorf("%s: %w", queue[stoppedAt].filename, migrationErr)
		}
		return nil
	}
}

func (cmd *MigrateCmd) run(conn *sql.Conn, migrations []migration) (stoppedAt int, err error) {
	var tx *sql.Tx
	if len(migrations) > 1 || (!strings.HasSuffix(migrations[0].filename, ".txoff.sql") && cmd.Dialect != DialectMySQL) || strings.HasSuffix(migrations[0].filename, ".tx.sql") {
		var err error
		if cmd.Verbose {
			fmt.Fprintln(cmd.Stderr, timestamp()+"BEGIN")
		} else {
			fmt.Fprintln(cmd.Stderr, "BEGIN")
		}
		tx, err = conn.BeginTx(cmd.Ctx, nil)
		if err != nil {
			return -1, err
		}
		defer tx.Rollback()
	}
	rollback := func(tx *sql.Tx) {
		if tx == nil {
			return
		}
		if cmd.Verbose {
			fmt.Fprintln(cmd.Stderr, timestamp()+"ROLLBACK")
		} else {
			fmt.Fprintln(cmd.Stderr, "ROLLBACK")
		}
		tx.Rollback()
	}
	var db sq.DB
	if tx != nil {
		db = tx
	} else {
		db = conn
	}
	for i := range migrations {
		m := &migrations[i]
		// Read file contents into buffer.
		file, err := cmd.DirFS.Open(m.filename)
		if err != nil {
			rollback(tx)
			return i, err
		}
		cmd.buf.Reset()
		_, err = cmd.buf.ReadFrom(file)
		file.Close()
		if err != nil {
			rollback(tx)
			return i, err
		}
		// Execute the contents of the script.
		contents := cmd.buf.String()
		if cmd.Verbose {
			fmt.Fprintln(cmd.Stderr, timestamp()+"[START] "+m.filename)
		}
		m.startedAt = sql.NullTime{Time: time.Now(), Valid: true}
		_, migrationErr := db.ExecContext(cmd.Ctx, contents)
		timeTaken := time.Since(m.startedAt.Time)
		m.timeTakenNs = sql.NullInt64{Int64: int64(timeTaken), Valid: true}
		if migrationErr != nil {
			if cmd.driver.AnnotateError != nil {
				migrationErr = cmd.driver.AnnotateError(migrationErr, contents)
			}
			m.success = false
			if cmd.Verbose {
				fmt.Fprintln(cmd.Stderr, timestamp()+"[FAIL]  "+m.filename+" ("+timeTaken.String()+")")
			} else {
				fmt.Fprintln(cmd.Stderr, "[FAIL] "+m.filename+" ("+timeTaken.String()+")")
			}
			rollback(tx)
			return i, migrationErr
		}
		m.success = true
		if cmd.Verbose {
			fmt.Fprintln(cmd.Stderr, timestamp()+"[OK]    "+m.filename+" ("+timeTaken.String()+")")
		} else {
			fmt.Fprintln(cmd.Stderr, "[OK] "+m.filename+" ("+timeTaken.String()+")")
		}
		if cmd.SkipHistoryTable {
			continue
		}
		// Upsert the script status in the history table.
		bi := BatchInsert{
			Dialect:    cmd.Dialect,
			TableName:  cmd.HistoryTable,
			Columns:    []string{"filename", "checksum", "started_at", "time_taken_ns", "success"},
			KeyColumns: []string{"filename"},
		}
		i := 0
		_, err = bi.ExecContext(cmd.Ctx, db, func(row []any) error {
			if i > 0 {
				return io.EOF
			}
			row[0] = m.filename
			row[1] = m.checksum
			row[2] = m.startedAt
			if cmd.Dialect == DialectSQLite && m.startedAt.Valid {
				row[2] = m.startedAt.Time.UTC().Format("2006-01-02 15:04:05")
			}
			row[3] = m.timeTakenNs
			row[4] = m.success
			i++
			return nil
		})
		if err != nil {
			rollback(tx)
			return i, err
		}
	}
	if tx != nil {
		if cmd.Verbose {
			fmt.Fprintln(cmd.Stderr, timestamp()+"COMMIT")
		} else {
			fmt.Fprintln(cmd.Stderr, "COMMIT")
		}
		err = tx.Commit()
		if err != nil {
			return len(migrations) - 1, err
		}
	}
	return len(migrations) - 1, nil
}

func (cmd *MigrateCmd) undo(conn *sql.Conn, m migration, originalErr error) error {
	// Get the undo script filename.
	undofile := strings.TrimSuffix(strings.TrimSuffix(m.filename, ".sql"), ".txoff") + ".undo.sql"
	file, err := cmd.DirFS.Open(undofile)
	// If the undo script doesn't exist, return.
	if errors.Is(err, fs.ErrNotExist) {
		return originalErr
	}
	if cmd.Verbose {
		fmt.Fprintln(cmd.Stderr, timestamp()+"[UNDO]  "+undofile)
	} else {
		fmt.Fprintln(cmd.Stderr, "[UNDO] "+undofile)
	}
	if err != nil {
		return fmt.Errorf("%w\n%s: %s", originalErr, undofile, err.Error())
	}
	defer file.Close()
	// Else read the undo script into a buffer and execute it. Any errors will
	// be printed to cmd.Stderr.
	cmd.buf.Reset()
	_, err = cmd.buf.ReadFrom(file)
	if err != nil {
		return fmt.Errorf("%w\n%s: %s", originalErr, undofile, err.Error())
	}
	_, err = conn.ExecContext(cmd.Ctx, cmd.buf.String())
	if err != nil {
		return fmt.Errorf("%w\n%s: %s", originalErr, undofile, err.Error())
	}
	return originalErr
}
