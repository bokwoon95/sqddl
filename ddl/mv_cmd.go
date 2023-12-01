package ddl

import (
	"database/sql"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// MvCmd implements the `sqddl mv` subcommand.
type MvCmd struct {
	// (Required) DB is the database.
	DB *sql.DB

	// (Required) Dialect is the database dialect.
	Dialect string

	// (Required) SrcFilename is the source filename to be renamed from.
	SrcFilename string

	// (Required) DestFilename is the destination filename to be renamed to.
	DestFilename string

	// Stderr specifies the command's standard error. If nil, the command
	// writes to os.Stderr.
	Stderr io.Writer

	// HistoryTable is the name of the migration history table. If empty, the
	// default history table name will be "sqddl_history".
	HistoryTable string

	db string // -db flag.
}

// MvCommand creates a new MvCmd with the given arguments. E.g.
//
//	sqddl mv -db <DATABASE_URL> <OLD_FILENAME> <NEW_FILENAME>
//
//	MvCommand(
//	    "-db", "postgres://user:pass@localhost:5432/sakila",
//	    "-dir", "./migrations",
//	    "old_name.sql", "new_name.sql",
//	)
func MvCommand(args ...string) (*MvCmd, error) {
	var cmd MvCmd
	var dir string
	flagset := flag.NewFlagSet("", flag.ContinueOnError)
	flagset.StringVar(&cmd.db, "db", "", "(required) Database URL/DSN.")
	flagset.StringVar(&dir, "dir", "", "Migration directory.")
	flagset.StringVar(&cmd.HistoryTable, "history-table", "sqddl_history", "Name of migration history table.")
	flagset.Usage = func() {
		fmt.Fprint(flagset.Output(), `Usage:
  sqddl mv -db <DATABASE_URL> <OLD_FILENAME> <NEW_FILENAME>
  sqddl mv -db 'postgres://user:pass@localhost:5432/sakila' old_name.sql new_name.sql
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
	flagArgs := flagset.Args()
	if len(flagArgs) >= 1 {
		cmd.SrcFilename = normalizeFilename(flagArgs[0], dir)
	}
	if len(flagArgs) >= 2 {
		cmd.DestFilename = normalizeFilename(flagArgs[1], dir)
	}
	return &cmd, nil
}

// Run runs the MvCmd.
func (cmd *MvCmd) Run() error {
	if cmd.DB == nil {
		return fmt.Errorf("nil DB")
	}
	if cmd.Dialect == "" {
		return fmt.Errorf("empty Dialect")
	}
	if cmd.SrcFilename == "" {
		return fmt.Errorf("source filename not provided")
	}
	if cmd.DestFilename == "" {
		return fmt.Errorf("destination filename not provided")
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

	exists, err := historyTableExists(cmd.Dialect, cmd.DB, cmd.HistoryTable)
	if err != nil {
		return err
	}
	if !exists {
		fmt.Fprintln(cmd.Stderr, "0 rows affected")
		return nil
	}
	var b strings.Builder
	b.WriteString("UPDATE " + QuoteIdentifier(cmd.Dialect, cmd.HistoryTable))
	b.WriteString(" SET filename = '" + EscapeQuote(cmd.DestFilename, '\'') + "'")
	b.WriteString(" WHERE filename = '" + EscapeQuote(cmd.SrcFilename, '\'') + "'")
	result, err := cmd.DB.Exec(b.String())
	if err != nil {
		return nil
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil
	}
	fmt.Fprintln(cmd.Stderr, strconv.FormatInt(rowsAffected, 10)+" rows affected")
	if err != nil {
		return err
	}
	return nil
}
