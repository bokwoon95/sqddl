package ddl

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/bokwoon95/sq"
)

// LsCmd implements the `sqddl ls` subcommand.
type LsCmd struct {
	// (Required) DB is the database.
	DB *sql.DB

	// (Required) Dialect is the database dialect.
	Dialect string

	// (Required) DirFS is the migration directory.
	DirFS fs.FS

	// Stdout specifies the command's standard out. If nil, the command writes
	// to os.Stdout.
	Stdout io.Writer

	// HistoryTable is the name of the migration history table. If empty, the
	// default history table name will be "sqddl_history".
	HistoryTable string

	// Include pending migrations in output.
	IncludePending bool

	// Include applied migrations in output.
	IncludeApplied bool

	// Include failed migrations in output.
	IncludeFailed bool

	// Include missing migrations in output.
	IncludeMissing bool

	// Include all migrations in output.
	IncludeAll bool

	db  string        // -db flag.
	dir string        // -dir flag.
	buf *bytes.Buffer // Reusable buffer. Make sure to Reset() before use.
}

// LsCommand creates a new LsCmd with the given arguments. E.g.
//
//   sqddl ls -db <DATABASE_URL> -dir <MIGRATION_DIR> [FLAGS]
//
//   LsCommand("-db", "postgres://user:pass@localhost:5432/sakila", "-dir", "./migrations")
func LsCommand(args ...string) (*LsCmd, error) {
	var cmd LsCmd
	flagset := flag.NewFlagSet("", flag.ContinueOnError)
	flagset.StringVar(&cmd.db, "db", "", "(required) Database URL/DSN.")
	flagset.StringVar(&cmd.dir, "dir", "", "(required) Migration directory.")
	flagset.StringVar(&cmd.HistoryTable, "history-table", "sqddl_history", "Name of migration history table.")
	flagset.BoolVar(&cmd.IncludePending, "pending", false, "Include pending migrations.")
	flagset.BoolVar(&cmd.IncludeApplied, "applied", false, "Include applied migrations.")
	flagset.BoolVar(&cmd.IncludeFailed, "failed", false, "Include failed migrations.")
	flagset.BoolVar(&cmd.IncludeMissing, "missing", false, "Include missing migrations.")
	flagset.BoolVar(&cmd.IncludeAll, "all", false, "Include all migrations (pending, applied, failed, missing).")
	flagset.Usage = func() {
		fmt.Fprint(flagset.Output(), `Usage:
  sqddl ls -db <DATABASE_URL> -dir <MIGRATION_DIR> [FLAGS]
  sqddl ls -db 'postgres://username:password@localhost:5432/sakila' -dir ./migrations
  sqddl ls -db 'postgres://username:password@localhost:5432/sakila' -dir ./migrations -all
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
	if cmd.dir == "" {
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
	cmd.DirFS = os.DirFS(cmd.dir)
	return &cmd, nil
}

func (cmd *LsCmd) Run() error {
	if cmd.DB == nil {
		return fmt.Errorf("nil DB")
	}
	if cmd.Dialect == "" {
		return fmt.Errorf("empty Dialect")
	}
	if cmd.DirFS == nil {
		return fmt.Errorf("nil Dir")
	}
	if cmd.Stdout == nil {
		cmd.Stdout = os.Stdout
	}
	if cmd.HistoryTable == "" {
		cmd.HistoryTable = "sqddl_history"
	}
	if cmd.IncludeAll {
		cmd.IncludePending = true
		cmd.IncludeApplied = true
		cmd.IncludeFailed = true
		cmd.IncludeMissing = true
	} else if !cmd.IncludePending && !cmd.IncludeApplied && !cmd.IncludeFailed && !cmd.IncludeMissing {
		cmd.IncludePending = true
		cmd.IncludeFailed = true
	}
	if cmd.db != "" {
		defer cmd.DB.Close()
	}
	cmd.buf = bufpool.Get().(*bytes.Buffer)
	cmd.buf.Reset()
	defer bufpool.Put(cmd.buf)

	filenames, err := walkDir(cmd.DirFS)
	if err != nil {
		return err
	}
	filenames = sortAndFilterFilenames(filenames)
	migrations := make([]migration, len(filenames))
	cache := make(map[string]int)
	for i, filename := range filenames {
		migrations[i].filename = filename
		cache[filename] = i
	}
	exists, err := historyTableExists(cmd.Dialect, cmd.DB, cmd.HistoryTable)
	if err != nil {
		return err
	}
	historyTable := sq.Identifier(cmd.HistoryTable)
	if exists {
		cursor, err := sq.FetchCursor(cmd.DB, sq.
			Queryf("SELECT {*} FROM {} WHERE filename IN ({})", historyTable, filenames).
			SetDialect(cmd.Dialect),
			migration{}.rowmapper,
		)
		if err != nil {
			return err
		}
		defer cursor.Close()
		for cursor.Next() {
			script, err := cursor.Result()
			if err != nil {
				return err
			}
			if i, ok := cache[script.filename]; ok {
				migrations[i] = script
			}
		}
	}

	// The main loop.
	for _, script := range migrations {
		status := "[applied]"
		if !script.valid || !script.startedAt.Valid {
			status = "[pending]"
		} else if !script.success {
			status = "[failed]"
		} else if strings.HasPrefix(script.filename, "repeatable/") {
			f, err := cmd.DirFS.Open(script.filename)
			if err != nil {
				return err
			}
			cmd.buf.Reset()
			_, err = cmd.buf.ReadFrom(f)
			f.Close()
			if err != nil {
				return err
			}
			hash := sha256.Sum256(bytes.ReplaceAll(cmd.buf.Bytes(), []byte("\r\n"), []byte("\n")))
			checksum := hex.EncodeToString(hash[:])
			if checksum != script.checksum {
				status = "[pending]"
			}
		}
		if status == "[pending]" && !cmd.IncludePending {
			continue
		} else if status == "[applied]" && !cmd.IncludeApplied {
			continue
		} else if status == "[failed]" && !cmd.IncludeFailed {
			continue
		}
		_, err = io.WriteString(cmd.Stdout, status+" "+script.filename)
		if err != nil {
			return err
		}
		if script.startedAt.Valid {
			_, err = io.WriteString(cmd.Stdout, " ("+script.startedAt.Time.UTC().Format("2006-01-02 15:04:05")+" "+time.Duration(script.timeTakenNs.Int64).String()+")")
			if err != nil {
				return err
			}
		}
		_, err = io.WriteString(cmd.Stdout, "\n")
		if err != nil {
			return err
		}
	}

	// If user wants to include missing scripts in the output, we must do a
	// separate query in the database looking for scripts that exist in the
	// history table but are not in the migration dir.
	if exists && cmd.IncludeMissing {
		cursor, err := sq.FetchCursor(cmd.DB, sq.
			Queryf("SELECT {*} FROM {} WHERE filename NOT IN ({}) ORDER BY CASE WHEN filename LIKE 'repeatable/%' THEN 1 ELSE 0 END, filename", historyTable, filenames).
			SetDialect(cmd.Dialect),
			migration{}.rowmapper,
		)
		if err != nil {
			return err
		}
		defer cursor.Close()
		for cursor.Next() {
			script, err := cursor.Result()
			if err != nil {
				return err
			}
			_, err = io.WriteString(cmd.Stdout, "[missing] "+script.filename+"\n")
			if err != nil {
				return err
			}
		}
	}
	return nil
}
