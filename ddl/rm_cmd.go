package ddl

import (
	"database/sql"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/bokwoon95/sq"
)

// RmCmd implements the `sqddl rm` subcommand.
type RmCmd struct {
	// (Required) DB is the database.
	DB *sql.DB

	// (Required) Dialect is the database dialect.
	Dialect string

	// Filenames specifies the list of files to be removed from the
	// history table.
	Filenames []string

	// Stderr specifies the command's standard error. If nil, the command
	// writes to os.Stderr.
	Stderr io.Writer

	// HistoryTable is the name of the migration history table. If empty, the
	// default history table name will be "sqddl_history".
	HistoryTable string

	db string // -db flag.
}

// RmCommand creates a new RmCmd with the given arguments.
//
//   sqddl rm -db <DATABASE_URL> [FILENAMES...]
//
//   RmCommand(
//       "-db", "postgres://user:pass@localhost:5432/sakila",
//       "-dir", "./migrations",
//       "02_sakila.sql", 04_extras.sql",
//   )
func RmCommand(args ...string) (*RmCmd, error) {
	var cmd RmCmd
	var dir string
	flagset := flag.NewFlagSet("", flag.ContinueOnError)
	flagset.StringVar(&cmd.db, "db", "", "(required) Database URL/DSN.")
	flagset.StringVar(&dir, "dir", "", "Migration directory.")
	flagset.StringVar(&cmd.HistoryTable, "history-table", "sqddl_history", "Name of migration history table.")
	flagset.Usage = func() {
		fmt.Fprint(flagset.Output(), `Usage:
  sqddl rm -db <DATABASE_URL> [FILENAMES...]
  sqddl rm -db 'postgres://user:pass@localhost:5432/sakila'
  sqddl rm -db 'postgres://user:pass@localhost:5432/sakila' 01_init.sql 02_data.sql
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
	cmd.Dialect, driverName, dsn = normalizeDSN(cmd.db)
	if cmd.Dialect == "" {
		return nil, fmt.Errorf("could not identity dialect for -db %q", cmd.db)
	}
	cmd.DB, err = sql.Open(driverName, dsn)
	if err != nil {
		return nil, err
	}
	cmd.Filenames = flagset.Args()
	for i, filename := range cmd.Filenames {
		cmd.Filenames[i] = normalizeFilename(filename, dir)
	}
	return &cmd, nil
}

// Run runs the RmCmd.
func (cmd *RmCmd) Run() error {
	if cmd.DB == nil {
		return fmt.Errorf("nil DB")
	}
	if cmd.Dialect == "" {
		return fmt.Errorf("empty Dialect")
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

	if len(cmd.Filenames) == 0 {
		fmt.Fprintln(cmd.Stderr, "no filenames specified")
		return nil
	}
	exists, err := historyTableExists(cmd.Dialect, cmd.DB, cmd.HistoryTable)
	if err != nil {
		return err
	}
	if !exists {
		fmt.Fprintln(cmd.Stderr, "0 rows affected")
		return nil
	}
	historyTable := sq.Identifier(cmd.HistoryTable)
	result, err := sq.Exec(cmd.DB, sq.
		Queryf(`DELETE FROM {} WHERE filename IN ({})`, historyTable, cmd.Filenames).
		SetDialect(cmd.Dialect),
	)
	if err != nil {
		return err
	}
	if result.RowsAffected == 1 {
		fmt.Fprintln(cmd.Stderr, "1 row affected")
	} else {
		fmt.Fprintln(cmd.Stderr, strconv.FormatInt(result.RowsAffected, 10)+" rows affected")
	}
	return nil
}
