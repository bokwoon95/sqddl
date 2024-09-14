package ddl

import (
	"bytes"
	"context"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"testing/fstest"
	"time"
)

// AutomigrateCmd implements the `sqddl automigrate` subcommand.
type AutomigrateCmd struct {
	// (Required) DB is the database to apply migrations to.
	DB *sql.DB

	// (Required) Dialect is the database dialect.
	Dialect string

	// DestCatalog is the destination catalog that you want to migrate to.
	DestCatalog *Catalog

	// DirFS is where the Filenames will be sourced from.
	DirFS fs.FS

	// Filenames specifies the list of files (loaded from the DirFS field) used
	// to build the DestCatalog. It will be ignored if the DestCatalog is
	// already non-nil.
	Filenames []string

	// Stdout is the command's standard out. If nil, the command writes to
	// os.Stdout.
	Stdout io.Writer

	// Stderr specifies the command's standard error. If nil, the command
	// writes to os.Stderr.
	Stderr io.Writer

	// HistoryTable is the name of the migration history table. If empty, the
	// default history table name will be "sqddl_history".
	HistoryTable string

	// DropObjects controls whether statements like DROP TABLE, DROP COLUMN
	// will be generated.
	DropObjects bool

	// AcceptWarnings will accept warnings when generating migrations.
	AcceptWarnings bool

	// If DryRun is true, the SQL queries will be written to Stdout instead of
	// being run against the database.
	DryRun bool

	// LockTimeout specifies how long to wait to acquire a lock on a table
	// before bailing out. If empty, 1*time.Second is used.
	LockTimeout time.Duration

	// Maximum number of retries on lock timeout.
	MaxAttempts int

	// Maximum delay between retries.
	MaxDelay time.Duration

	// Base delay between retries.
	BaseDelay time.Duration

	// Ctx is the command's context.
	Ctx context.Context

	db string // -db flag.
}

// AutomigrateCommand creates a new AutomigrateCmd with the given arguments. E.g.
//
//	sqddl automigrate -db <DATABASE_URL> -dest <DEST_SCHEMA> [FLAGS]
//
//	Automigrate("-db", "postgres://user:pass@localhost:5432/sakila", "-dest", "tables/tables.go")
func AutomigrateCommand(args ...string) (*AutomigrateCmd, error) {
	var cmd AutomigrateCmd
	var dest, dir, lockTimeout, maxDelay, baseDelay string
	flagset := flag.NewFlagSet("", flag.ContinueOnError)
	flagset.StringVar(&cmd.db, "db", "", "(required) Database URL/DSN.")
	flagset.StringVar(&dir, "dir", "", "Working directory. Leave blank to use the current working directory.")
	flagset.StringVar(&dest, "dest", "", "Comma-separated list of destination schemas.")
	flagset.StringVar(&cmd.HistoryTable, "history-table", "sqddl_history", "Name of migration history table.")
	flagset.BoolVar(&cmd.DropObjects, "drop-objects", false, "Whether statements like DROP TABLE, DROP COLUMN, etc should be generated.")
	flagset.BoolVar(&cmd.AcceptWarnings, "accept-warnings", false, "Accept warnings when generating migrations.")
	flagset.BoolVar(&cmd.DryRun, "dry-run", false, "Print the generated SQL statements without running them.")
	flagset.StringVar(&lockTimeout, "lock-timeout", "1s", "How long to wait to acquire a lock on a table before timing out.")
	flagset.IntVar(&cmd.MaxAttempts, "max-attempts", 10, "Maximum number of retries on lock timeout.")
	flagset.StringVar(&maxDelay, "max-delay", "5m", "Maximum delay between retries.")
	flagset.StringVar(&baseDelay, "base-delay", "1s", "Base delay between retries.")
	flagset.Usage = func() {
		fmt.Fprint(flagset.Output(), `Usage:
  sqddl automigrate -db <DATABASE_URL> -dest <DEST_SCHEMA> [FLAGS]
  sqddl automigrate -db 'postgres://username:password@localhost:5432/sakila' -dest tables/tables.go
  sqddl automigrate -db 'postgres://username:password@localhost:5432/sakila' -dest file1.go,file2.go,file3.go
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
	if dest != "" {
		dbi := NewDatabaseIntrospector(cmd.Dialect, cmd.DB)
		cmd.DestCatalog = &Catalog{Dialect: cmd.Dialect}
		cmd.DestCatalog.CurrentSchema, err = dbi.GetCurrentSchema()
		if err != nil {
			return nil, err
		}
		for _, s := range strings.Split(dest, ",") {
			err = writeCatalog(cmd.DestCatalog, cmd.DirFS, cmd.HistoryTable, s)
			if err != nil {
				return nil, err
			}
		}
	}
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
	return &cmd, nil
}

// Run runs the AutomigrateCmd.
func (cmd *AutomigrateCmd) Run() error {
	if cmd.DB == nil {
		return fmt.Errorf("nil DB")
	}
	if cmd.Dialect == "" {
		return fmt.Errorf("empty Dialect")
	}
	if cmd.DirFS == nil {
		cmd.DirFS = dirFS(".")
	}
	if cmd.Stdout == nil {
		cmd.Stdout = os.Stdout
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
	if cmd.Ctx == nil {
		cmd.Ctx = context.Background()
	}
	srcCatalog := &Catalog{}
	dbi := NewDatabaseIntrospector(cmd.Dialect, cmd.DB)
	dbi.ObjectTypes = []string{"TABLES"}
	dbi.ExcludeTables = []string{cmd.HistoryTable}
	err := dbi.WriteCatalog(srcCatalog)
	if err != nil {
		return err
	}
	if cmd.DestCatalog == nil {
		cmd.DestCatalog = &Catalog{
			Dialect:       srcCatalog.Dialect,
			CurrentSchema: srcCatalog.CurrentSchema,
		}
		for _, filename := range cmd.Filenames {
			err = writeCatalog(cmd.DestCatalog, cmd.DirFS, cmd.HistoryTable, filename)
			if err != nil {
				return err
			}
		}
	} else {
		if cmd.DestCatalog.Dialect == "" {
			cmd.DestCatalog.Dialect = srcCatalog.Dialect
		}
		if cmd.DestCatalog.CurrentSchema == "" {
			cmd.DestCatalog.CurrentSchema = srcCatalog.CurrentSchema
		}
	}
	var filenames, warnings []string
	var bufs []*bytes.Buffer
	prefix := "automigrate"
	switch cmd.Dialect {
	case DialectSQLite:
		m := newSQLiteMigration(srcCatalog, cmd.DestCatalog, cmd.DropObjects)
		filenames, bufs, warnings = m.sql(prefix)
	case DialectPostgres:
		m := newPostgresMigration(srcCatalog, cmd.DestCatalog, cmd.DropObjects)
		filenames, bufs, warnings = m.sql(prefix)
	case DialectMySQL:
		m := newMySQLMigration(srcCatalog, cmd.DestCatalog, cmd.DropObjects)
		filenames, bufs, warnings = m.sql(prefix)
	case DialectSQLServer:
		m := newSQLServerMigration(srcCatalog, cmd.DestCatalog, cmd.DropObjects)
		filenames, bufs, warnings = m.sql(prefix)
	default:
		return fmt.Errorf("unsupported dialect %q", srcCatalog.Dialect)
	}
	defer func() {
		for _, buf := range bufs {
			bufpool.Put(buf)
		}
	}()
	if len(filenames) == 0 {
		return nil
	}
	if len(warnings) > 0 {
		if !cmd.AcceptWarnings && !cmd.DryRun {
			return fmt.Errorf("warnings present (to proceed despite the warnings, use the -accept-warnings flag):\n" + strings.Join(warnings, "\n"))
		}
		for _, warning := range warnings {
			fmt.Fprintln(cmd.Stderr, warning)
		}
	}
	if cmd.DryRun {
		for i, filename := range filenames {
			if strings.HasSuffix(filename, ".undo.sql") {
				continue
			}
			if i > 0 {
				_, err = io.WriteString(cmd.Stdout, "\n")
				if err != nil {
					return err
				}
			}
			if len(filenames) > 1 {
				_, err = io.WriteString(cmd.Stdout, "-- "+filename+"\n")
				if err != nil {
					return err
				}
			}
			_, err = bufs[i].WriteTo(cmd.Stdout)
			if err != nil {
				return err
			}
		}
		return nil
	}
	migrationDir := make(map[string]*fstest.MapFile)
	for i, filename := range filenames {
		migrationDir[filename] = &fstest.MapFile{
			Data: bufs[i].Bytes(),
			Mode: 0644,
		}
	}
	migrateCmd := &MigrateCmd{
		Dialect:          cmd.Dialect,
		DB:               cmd.DB,
		DirFS:            fstest.MapFS(migrationDir),
		Filenames:        filenames,
		Stderr:           cmd.Stderr,
		HistoryTable:     cmd.HistoryTable,
		LockTimeout:      cmd.LockTimeout,
		MaxAttempts:      cmd.MaxAttempts,
		MaxDelay:         cmd.MaxDelay,
		BaseDelay:        cmd.BaseDelay,
		SkipHistoryTable: true,
		Ctx:              cmd.Ctx,
	}
	err = migrateCmd.Run()
	if err != nil {
		return err
	}
	return nil
}
