package ddl

import (
	"bytes"
	"context"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/bokwoon95/sq"
)

// WipeCmd implements the `sqddl wipe` subcommand.
type WipeCmd struct {
	// (Required) DB is the database to wipe.
	DB *sql.DB

	// (Required) Dialect is the database dialect.
	Dialect string

	// If DryRun is true, the SQL queries will be written to Stdout instead of
	// being run against the database.
	DryRun bool

	// Stdout is the command's standard out. If nil, the command writes to
	// os.Stdout.
	Stdout io.Writer

	// Ctx is the command's context.
	Ctx context.Context

	db  string        // -db flag.
	buf *bytes.Buffer // Reusable buffer. Make sure to Reset() before use.
}

// WipeCommand creates a new WipeCmd with the given arguments. E.g.
//   sqddl wipe -db <DATABASE_URL> [FLAGS]
//
//   WipeCommand("-db", "postgres://user:pass@localhost:5432/sakila")
func WipeCommand(args ...string) (*WipeCmd, error) {
	var cmd WipeCmd
	var historyTable string
	flagset := flag.NewFlagSet("", flag.ContinueOnError)
	flagset.StringVar(&cmd.db, "db", "", "(required) Database URL/DSN.")
	flagset.StringVar(&historyTable, "history-table", "", "(ignored)")
	flagset.BoolVar(&cmd.DryRun, "dry-run", false, "Print the generated SQL statements without running them.")
	flagset.Usage = func() {
		fmt.Fprint(flagset.Output(), `Usage:
  sqddl wipe -db <DATABASE_URL> [FLAGS]
  sqddl wipe -db 'postgres://username:password@localhost:5432/sakila' -dry-run
  sqddl wipe -db 'postgres://username:password@localhost:5432/sakila'
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
	return &cmd, nil
}

// Run runs the WipeCmd.
func (cmd *WipeCmd) Run() error {
	if cmd.DB == nil {
		return fmt.Errorf("nil DB")
	}
	if cmd.Dialect == "" {
		return fmt.Errorf("empty Dialect")
	}
	if cmd.Stdout == nil {
		cmd.Stdout = os.Stdout
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

	dbi := NewDatabaseIntrospector(cmd.Dialect, cmd.DB)
	currentSchema, err := dbi.GetCurrentSchema()
	if err != nil {
		return err
	}

	// DROP VIEW.
	views, err := dbi.GetViews()
	if err != nil {
		return err
	}
	for _, view := range views {
		if cmd.buf.Len() > 0 {
			cmd.buf.WriteString("\n")
		}
		cmd.buf.WriteString("DROP ")
		if view.IsMaterialized {
			cmd.buf.WriteString("MATERIALIZED ")
		}
		cmd.buf.WriteString("VIEW IF EXISTS ")
		if view.ViewSchema != "" && view.ViewSchema != currentSchema {
			cmd.buf.WriteString(sq.QuoteIdentifier(cmd.Dialect, view.ViewSchema) + ".")
		}
		cmd.buf.WriteString(sq.QuoteIdentifier(cmd.Dialect, view.ViewName))
		if cmd.Dialect == DialectPostgres || cmd.Dialect == DialectMySQL {
			cmd.buf.WriteString(" CASCADE")
		}
		cmd.buf.WriteString(";\n")
	}

	// ALTER TABLE DROP CONSTRAINT (Foreign keys). We drop all foreign keys
	// first before dropping tables so that we don't get tripped by by circular
	// table dependencies later.
	if cmd.Dialect != DialectSQLite {
		dbi.ConstraintTypes = []string{FOREIGN_KEY}
		constraints, err := dbi.GetConstraints()
		if err != nil {
			return err
		}
		for _, constraint := range constraints {
			if cmd.buf.Len() > 0 {
				cmd.buf.WriteString("\n")
			}
			cmd.buf.WriteString("ALTER TABLE ")
			if constraint.TableSchema != "" && constraint.TableSchema != currentSchema {
				cmd.buf.WriteString(sq.QuoteIdentifier(cmd.Dialect, constraint.TableSchema) + ".")
			}
			cmd.buf.WriteString(sq.QuoteIdentifier(cmd.Dialect, constraint.TableName) + " DROP CONSTRAINT " + sq.QuoteIdentifier(cmd.Dialect, constraint.ConstraintName))
			if cmd.Dialect == DialectPostgres {
				cmd.buf.WriteString(" CASCADE")
			}
			cmd.buf.WriteString(";\n")
		}
	}

	// DROP TABLE.
	tables, err := dbi.GetTables()
	if err != nil {
		return err
	}
	if cmd.Dialect == DialectSQLite && len(tables) > 0 {
		if cmd.buf.Len() > 0 {
			cmd.buf.WriteString("\n")
		}
		cmd.buf.WriteString("PRAGMA defer_foreign_keys = 1;\n")
	}
	for _, table := range tables {
		if cmd.buf.Len() > 0 {
			cmd.buf.WriteString("\n")
		}
		cmd.buf.WriteString("DROP TABLE IF EXISTS ")
		if table.TableSchema != "" && table.TableSchema != currentSchema {
			cmd.buf.WriteString(sq.QuoteIdentifier(cmd.Dialect, table.TableSchema) + ".")
		}
		cmd.buf.WriteString(sq.QuoteIdentifier(cmd.Dialect, table.TableName))
		if cmd.Dialect == DialectPostgres || cmd.Dialect == DialectMySQL {
			cmd.buf.WriteString(" CASCADE")
		}
		cmd.buf.WriteString(";\n")
	}

	// DROP PROCEDURE and DROP FUNCTION.
	if cmd.Dialect != DialectSQLite {
		routines, err := dbi.GetRoutines()
		if err != nil {
			return err
		}
		for _, routine := range routines {
			if cmd.buf.Len() > 0 {
				cmd.buf.WriteString("\n")
			}
			cmd.buf.WriteString("DROP ")
			if routine.RoutineType == "PROCEDURE" {
				cmd.buf.WriteString("PROCEDURE ")
			} else {
				cmd.buf.WriteString("FUNCTION ")
			}
			cmd.buf.WriteString("IF EXISTS ")
			if routine.RoutineSchema != "" && routine.RoutineSchema != currentSchema {
				cmd.buf.WriteString(sq.QuoteIdentifier(cmd.Dialect, routine.RoutineSchema) + ".")
			}
			cmd.buf.WriteString(sq.QuoteIdentifier(cmd.Dialect, routine.RoutineName))
			if cmd.Dialect == DialectPostgres {
				cmd.buf.WriteString(" CASCADE")
			}
			cmd.buf.WriteString(";\n")
		}
	}

	if cmd.Dialect == DialectPostgres {
		// DROP TYPE.
		enums, err := dbi.GetEnums()
		if err != nil {
			return err
		}
		for _, enum := range enums {
			if cmd.buf.Len() > 0 {
				cmd.buf.WriteString("\n")
			}
			cmd.buf.WriteString("DROP TYPE IF EXISTS ")
			if enum.EnumSchema != "" && enum.EnumSchema != currentSchema {
				cmd.buf.WriteString(sq.QuoteIdentifier(cmd.Dialect, enum.EnumSchema) + ".")
			}
			cmd.buf.WriteString(sq.QuoteIdentifier(cmd.Dialect, enum.EnumName) + " CASCADE;\n")
		}

		// DROP DOMAIN.
		domains, err := dbi.GetDomains()
		if err != nil {
			return err
		}
		for _, domain := range domains {
			if cmd.buf.Len() > 0 {
				cmd.buf.WriteString("\n")
			}
			cmd.buf.WriteString("DROP DOMAIN IF EXISTS ")
			if domain.DomainSchema != "" && domain.DomainSchema != currentSchema {
				cmd.buf.WriteString(sq.QuoteIdentifier(cmd.Dialect, domain.DomainSchema) + ".")
			}
			cmd.buf.WriteString(sq.QuoteIdentifier(cmd.Dialect, domain.DomainName) + " CASCADE;\n")
		}

		// DROP EXTENSION.
		extensions, err := dbi.GetExtensions()
		if err != nil {
			return err
		}
		for _, extension := range extensions {
			if extension == "plpgsql" {
				continue
			}
			if cmd.buf.Len() > 0 {
				cmd.buf.WriteString("\n")
			}
			cmd.buf.WriteString("DROP EXTENSION IF EXISTS " + sq.QuoteIdentifier(cmd.Dialect, extension) + " CASCADE;\n")
		}
	}

	// If DryRun, write the SQL queries to Stdout instead.
	if cmd.DryRun {
		_, err = cmd.buf.WriteTo(cmd.Stdout)
		if err != nil {
			return err
		}
		return nil
	}

	queries := cmd.buf.String()
	if queries == "" {
		return nil
	}

	// If MySQL, run the queries using DB.Exec because MySQL doesn't support
	// running DDL in transactions.
	if cmd.Dialect == DialectMySQL {
		_, err = cmd.DB.ExecContext(cmd.Ctx, queries)
		if err != nil {
			return err
		}
		return nil
	}

	// Otherwise, run the queries in a transaction.
	tx, err := cmd.DB.BeginTx(cmd.Ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(cmd.Ctx, queries)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}
