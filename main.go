package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/bokwoon95/sqddl/ddl"
	"github.com/bokwoon95/sqddl/drivers/ddlmysql"
	"github.com/bokwoon95/sqddl/drivers/ddloracle"
	"github.com/bokwoon95/sqddl/drivers/ddlpgx"
	"github.com/bokwoon95/sqddl/drivers/ddlsqlite3"
	"github.com/bokwoon95/sqddl/drivers/ddlsqlserver"
)

func init() {
	ddlsqlite3.Register()
	ddlpgx.Register()
	ddlmysql.Register()
	ddlsqlserver.Register()
	ddloracle.Register()
}

const helptext = `Usage:
  sqddl migrate     # Run pending migrations.
  sqddl ls          # Show pending migrations.
  sqddl touch       # Upsert migrations into to the history table. Does not run them.
  sqddl rm          # Remove migrations from the history table.
  sqddl mv          # Rename migrations in the history table.
  sqddl tables      # Generate table structs from database.
  sqddl views       # Generate view structs from database.
  sqddl generate    # Generate migrations from table structs.
  sqddl wipe        # Wipe a database of all views, tables, routines, enums, domains and extensions.
  sqddl dump        # Dump a database as sql scripts and csv files.
  sqddl load        # Load database dumps, sql scripts or csv files into a database.
  sqddl automigrate # Automatically migrate a database according to table structs.

To learn more about each subcommand, pass in the -h flag to the subcommand e.g. sqddl migrate -h

Core documentation resides at https://bokwoon.neocities.org/sqddl.html
`

func main() {
	var db, historyTable string
	flagset := flag.NewFlagSet("sqddl", flag.ContinueOnError)
	flagset.StringVar(&db, "db", "", "")
	flagset.StringVar(&historyTable, "history-table", "", "")
	flagset.Usage = func() {
		fmt.Fprint(os.Stderr, helptext)
	}
	err := flagset.Parse(os.Args[1:])
	if errors.Is(err, flag.ErrHelp) {
		os.Exit(0)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, helptext+err.Error())
		os.Exit(1)
	}
	flagArgs := flagset.Args()
	if len(flagArgs) == 0 {
		fmt.Fprintf(os.Stderr, helptext)
		return
	}
	// forward the -db and -history-table flags to the subcommand
	subcmd := flagArgs[0]
	n := len(os.Args[1:]) - len(flagArgs)
	args := make([]string, n+len(flagArgs[1:]))
	copy(args[:n], os.Args[1:n+1])
	copy(args[n:], flagArgs[1:])

	exit := func(subcmd string, err error) {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		var migrationErr *ddl.MigrationError
		if errors.As(err, &migrationErr) {
			fmt.Fprintln(os.Stderr, migrationErr.Contents)
		}
		fmt.Fprintln(os.Stderr, subcmd+": "+err.Error())
		os.Exit(1)
	}

	switch subcmd {
	case "migrate":
		migrateCmd, err := ddl.MigrateCommand(args...)
		if err != nil {
			exit(subcmd, err)
		}
		err = migrateCmd.Run()
		if err != nil {
			exit(subcmd, err)
		}
	case "ls":
		lsCmd, err := ddl.LsCommand(args...)
		if err != nil {
			exit(subcmd, err)
		}
		err = lsCmd.Run()
		if err != nil {
			exit(subcmd, err)
		}
	case "touch":
		touchCmd, err := ddl.TouchCommand(args...)
		if err != nil {
			exit(subcmd, err)
		}
		err = touchCmd.Run()
		if err != nil {
			exit(subcmd, err)
		}
	case "rm":
		rmCmd, err := ddl.RmCommand(args...)
		if err != nil {
			exit(subcmd, err)
		}
		err = rmCmd.Run()
		if err != nil {
			exit(subcmd, err)
		}
	case "mv":
		mvCmd, err := ddl.MvCommand(args...)
		if err != nil {
			exit(subcmd, err)
		}
		err = mvCmd.Run()
		if err != nil {
			exit(subcmd, err)
		}
	case "tables":
		tablesCmd, err := ddl.TablesCommand(args...)
		if err != nil {
			exit(subcmd, err)
		}
		err = tablesCmd.Run()
		if err != nil {
			exit(subcmd, err)
		}
	case "views":
		viewsCmd, err := ddl.ViewsCommand(args...)
		if err != nil {
			exit(subcmd, err)
		}
		err = viewsCmd.Run()
		if err != nil {
			exit(subcmd, err)
		}
	case "generate":
		generateCmd, err := ddl.GenerateCommand(args...)
		if err != nil {
			exit(subcmd, err)
		}
		err = generateCmd.Run()
		if err != nil {
			exit(subcmd, err)
		}
	case "wipe":
		wipeCmd, err := ddl.WipeCommand(args...)
		if err != nil {
			exit(subcmd, err)
		}
		err = wipeCmd.Run()
		if err != nil {
			exit(subcmd, err)
		}
	case "dump":
		dumpCmd, err := ddl.DumpCommand(args...)
		if err != nil {
			exit(subcmd, err)
		}
		err = dumpCmd.Run()
		if err != nil {
			exit(subcmd, err)
		}
	case "load":
		loadCmd, err := ddl.LoadCommand(args...)
		if err != nil {
			exit(subcmd, err)
		}
		err = loadCmd.Run()
		if err != nil {
			exit(subcmd, err)
		}
	case "automigrate":
		automigrateCmd, err := ddl.AutomigrateCommand(args...)
		if err != nil {
			exit(subcmd, err)
		}
		err = automigrateCmd.Run()
		if err != nil {
			exit(subcmd, err)
		}
	default:
		fmt.Fprintf(os.Stderr, "unrecognized subcommand %q\n", subcmd)
		return
	}
}
