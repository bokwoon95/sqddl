package ddl

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bokwoon95/sq"
)

// TouchCmd implements the `sqddl touch` subcommand.
type TouchCmd struct {
	// (Required) DB is the database.
	DB *sql.DB

	// (Required) Dialect is the database dialect.
	Dialect string

	// (Required) DirFS is the migration directory.
	DirFS fs.FS

	// Filenames specifies the list of files to be touched in the history
	// table. If empty, all migrations in the migration directory will be
	// touched.
	Filenames []string

	// Stderr specifies the command's standard error. If nil, the command
	// writes to os.Stderr.
	Stderr io.Writer

	// HistoryTable is the name of the migration history table. If empty, the
	// default history table name will be "sqddl_history".
	HistoryTable string

	db  string        // -db flag.
	buf *bytes.Buffer // Reusable buffer. Make sure to Reset() before use.
}

// TouchCommand creates a new TouchCmd from the given arguments.
//
//   sqddl touch -db <DATABASE_URL> -dir <MIGRATION_DIR> [FILENAMES...]
//
//   TouchCommand(
//       "-db", "postgres://user:pass@localhost:5432/sakila",
//       "-dir", "./migrations",
//       "02_sakila.sql",
//       "04_extras.sql",
//   )
func TouchCommand(args ...string) (*TouchCmd, error) {
	var cmd TouchCmd
	var dir string
	flagset := flag.NewFlagSet("", flag.ContinueOnError)
	flagset.StringVar(&cmd.db, "db", "", "(required) Database URL/DSN.")
	flagset.StringVar(&dir, "dir", "", "(required) Migration directory.")
	flagset.StringVar(&cmd.HistoryTable, "history-table", "sqddl_history", "Name of migration history table.")
	flagset.Usage = func() {
		fmt.Fprint(flagset.Output(), `Usage:
  sqddl touch -db <DATABASE_URL> -dir <MIGRATION_DIR> [FILENAMES...]
  sqddl touch -db 'postgres://user:pass@localhost:5432/sakila' -dir ./migrations
  sqddl touch -db 'postgres://user:pass@localhost:5432/sakila' -dir ./migrations 01_init.sql 02_data.sql
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
	var driverName, dsn string
	cmd.Dialect, driverName, dsn = NormalizeDSN(cmd.db)
	if cmd.Dialect == "" {
		return nil, fmt.Errorf("could not identity dialect for -db %q", cmd.db)
	}
	cmd.DB, err = sql.Open(driverName, dsn)
	if err != nil {
		return nil, err
	}
	cmd.DirFS = dirFS(dir)
	cmd.Filenames = flagset.Args()
	for i, filename := range cmd.Filenames {
		cmd.Filenames[i] = normalizeFilename(filename, dir)
	}
	return &cmd, nil
}

// Run runs the TouchCmd.
func (cmd *TouchCmd) Run() error {
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
	if cmd.db != "" {
		defer cmd.DB.Close()
	}
	cmd.buf = bufpool.Get().(*bytes.Buffer)
	cmd.buf.Reset()
	defer bufpool.Put(cmd.buf)

	err := ensureHistoryTableExists(cmd.Dialect, cmd.DB, cmd.HistoryTable)
	if err != nil {
		return err
	}
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

	// Upsert the rowvalues.
	bi := BatchInsert{
		Dialect:    cmd.Dialect,
		TableName:  cmd.HistoryTable,
		Columns:    []string{"filename", "checksum", "started_at", "time_taken_ns", "success"},
		KeyColumns: []string{"filename"},
	}
	tx, err := cmd.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	i := 0
	startedAt := time.Now()
	rowsAffected, err := bi.ExecContext(context.Background(), tx, func(row []any) error {
		if i >= len(cmd.Filenames) {
			return io.EOF
		}
		filename := cmd.Filenames[i] // filename
		row[0] = filename
		if strings.HasPrefix(filename, "repeatable/") {
			f, err := cmd.DirFS.Open(filename)
			if err != nil {
				return err
			}
			defer f.Close()
			cmd.buf.Reset()
			_, err = cmd.buf.ReadFrom(f)
			if err != nil {
				return err
			}
			hash := sha256.Sum256(bytes.ReplaceAll(cmd.buf.Bytes(), []byte("\r\n"), []byte("\n")))
			row[1] = hex.EncodeToString(hash[:]) // checksum
		}
		row[2] = startedAt // started_at
		row[3] = 0         // time_taken_ns
		row[4] = true      // success
		i++
		return nil
	})
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	if rowsAffected == 1 {
		fmt.Fprintln(cmd.Stderr, "1 row affected")
	} else {
		fmt.Fprintln(cmd.Stderr, strconv.FormatInt(rowsAffected, 10)+" rows affected")
	}
	return nil
}

func historyTableExists(dialect string, db *sql.DB, tableName string) (bool, error) {
	var exists bool
	switch dialect {
	case DialectSQLite:
		err := db.QueryRow(`SELECT EXISTS`+
			` (SELECT 1`+
			` FROM sqlite_schema`+
			` WHERE type = 'table'`+
			` AND tbl_name = $1)`, tableName,
		).Scan(&exists)
		if err != nil {
			return false, err
		}
	case DialectPostgres:
		err := db.QueryRow(`SELECT EXISTS`+
			` (SELECT 1`+
			` FROM pg_class AS tables`+
			` JOIN pg_namespace AS schemas ON schemas.oid = tables.relnamespace`+
			` WHERE tables.relkind = 'r'`+
			` AND schemas.nspname = current_schema()`+
			` AND tables.relname = $1)`, tableName,
		).Scan(&exists)
		if err != nil {
			return false, err
		}
	case DialectMySQL:
		err := db.QueryRow(`SELECT EXISTS`+
			` (SELECT 1`+
			` FROM information_schema.tables`+
			` WHERE table_type = 'BASE TABLE'`+
			` AND table_schema = database()`+
			` AND table_name = ?)`, tableName,
		).Scan(&exists)
		if err != nil {
			return false, err
		}
	case DialectSQLServer:
		err := db.QueryRow(`IF EXISTS`+
			` (SELECT 1`+
			` FROM sys.tables`+
			` WHERE SCHEMA_NAME(schema_id) = SCHEMA_NAME()`+
			` AND name = @p1) SELECT 1 ELSE SELECT 0`, tableName,
		).Scan(&exists)
		if err != nil {
			return false, err
		}
	default:
		return false, fmt.Errorf("unsupported dialect: %q", dialect)
	}
	return exists, nil
}

func ensureHistoryTableExists(dialect string, db *sql.DB, historyTable string) error {
	exists, err := historyTableExists(dialect, db, historyTable)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	pkeyDefinition := "CONSTRAINT sqddl_filename_pkey PRIMARY KEY (filename)"
	timeType := "DATETIME"
	boolType := "BOOLEAN"
	switch dialect {
	case DialectPostgres:
		timeType = "TIMESTAMPTZ"
	case DialectMySQL:
		pkeyDefinition = "PRIMARY KEY (filename)"
	case DialectSQLServer:
		timeType = "DATETIMEOFFSET"
		boolType = "BIT"
	}
	query := "CREATE TABLE " + sq.QuoteIdentifier(dialect, historyTable) + " (" +
		"\n    filename VARCHAR(255) NOT NULL" +
		"\n    ,checksum VARCHAR(64)" +
		"\n    ,started_at " + timeType +
		"\n    ,time_taken_ns BIGINT" +
		"\n    ,success " + boolType +
		"\n" +
		"\n    ," + pkeyDefinition +
		"\n)"
	_, err = db.Exec(query)
	if err != nil {
		return err
	}
	return nil
}

func walkDir(fsys fs.FS) (filenames []string, err error) {
	err = fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path == "." || path == "repeatable" || strings.HasPrefix(path, "repeatable/") {
				return nil
			}
			return fs.SkipDir
		}
		filenames = append(filenames, path)
		return nil
	})
	return filenames, err
}

func validateFilesExist(fsys fs.FS, filenames []string) error {
	for _, filename := range filenames {
		file, err := fsys.Open(filename)
		if err != nil {
			return err
		}
		file.Close()
	}
	return nil
}

func normalizeFilename(filename, dir string) string {
	filename = filepath.ToSlash(filename)
	dir = filepath.ToSlash(dir)
	if dir == "" {
		return filename
	}
	if strings.HasPrefix(filename, ".") {
		filename = strings.TrimPrefix(strings.TrimPrefix(filename, "."), "/")
	}
	if strings.HasPrefix(dir, ".") {
		dir = strings.TrimPrefix(strings.TrimPrefix(dir, "."), "/")
	}
	return strings.TrimPrefix(strings.TrimPrefix(filename, dir), "/")
}

func sortAndFilterFilenames(filenames []string) []string {
	result := make([]string, 0, len(filenames))
	repeatable := make([]string, 0, len(filenames))
	for _, filename := range filenames {
		filename = filepath.ToSlash(filename)
		if !strings.HasSuffix(filename, ".sql") || strings.HasSuffix(filename, ".undo.sql") {
			continue
		}
		basename := filepath.Base(filename)
		if basename == "schema.sql" || basename == "indexes.sql" || basename == "constraints.sql" {
			continue
		}
		if strings.HasPrefix(filename, "repeatable/") {
			repeatable = append(repeatable, filename)
		} else {
			result = append(result, filename)
		}
	}
	result = append(result, repeatable...)
	return result
}
