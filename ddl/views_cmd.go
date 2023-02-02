package ddl

import (
	"database/sql"
	"flag"
	"fmt"
	"go/format"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ViewsCmd implements the `sqddl views` subcommand.
type ViewsCmd struct {
	// (Required) DB is the database.
	DB *sql.DB

	// (Required) Dialect is the database dialect.
	Dialect string

	// PackageName is the name of the package for the generated go code. Leave
	// blank to use "_".
	PackageName string

	// Filename is the name of the file to write the table structs into. Leave
	// blank to write to Stdout instead.
	Filename string

	// Stdout specifies the command's standard out to write to if no Filename
	// is provided. If nil, the command writes to os.Stdout.
	Stdout io.Writer

	// Schemas is the list of schemas that will be included.
	Schemas []string

	// ExcludeSchemas is the list of schemas that will be excluded.
	ExcludeSchemas []string

	// Views is the list of tables that will be included.
	Views []string

	// ExcludeViews is the list of tables that will be excluded.
	ExcludeViews []string

	db string // -db flag.
}

func ViewsCommand(args ...string) (*ViewsCmd, error) {
	var cmd ViewsCmd
	var historyTable, schemas, excludeSchemas, views, excludeViews string
	flagset := flag.NewFlagSet("", flag.ContinueOnError)
	flagset.StringVar(&cmd.db, "db", "", "(required) Database URL/DSN.")
	flagset.StringVar(&historyTable, "history-table", "", "(ignored)")
	flagset.StringVar(&cmd.PackageName, "pkg", "_", "Package name used in the generated code.")
	flagset.StringVar(&cmd.Filename, "file", "", "Name of the file to write the generated code into. Leave blank to write to stdout.")
	flagset.StringVar(&schemas, "schemas", "", "Comma-separated list of schemas to be included.")
	flagset.StringVar(&excludeSchemas, "exclude-schemas", "", "Comma-separated list of schemas to be excluded.")
	flagset.StringVar(&views, "views", "", "Comma-separated list of views to be included.")
	flagset.StringVar(&excludeViews, "exclude-views", "", "Comma-separated list of views to be excluded.")
	flagset.Usage = func() {
		fmt.Fprint(flagset.Output(), `Usage:
  sqddl views -db <DATABASE_URL> [FLAGS]
  sqddl views -db 'postgres://user:pass@localhost:5432/sakila'
  sqddl views -db 'postgres://user:pass@localhost:5432/sakila' -pkg tables -file tables/views.go
  sqddl views -db 'postgres://user:pass@localhost:5432/sakila' -schemas schema1,schema2,schema3 -exclude-views view1,view2,view3
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
	if schemas != "" {
		cmd.Schemas = strings.Split(schemas, ",")
	}
	if excludeSchemas != "" {
		cmd.ExcludeSchemas = strings.Split(excludeSchemas, ",")
	}
	if views != "" {
		cmd.Views = strings.Split(views, ",")
	}
	if excludeViews != "" {
		cmd.ExcludeViews = strings.Split(excludeViews, ",")
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
	return &cmd, nil
}

func (cmd *ViewsCmd) Run() error {
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

	viewStructs, err := NewViewStructs(cmd.Dialect, cmd.DB, Filter{
		Schemas:        cmd.Schemas,
		ExcludeSchemas: cmd.ExcludeSchemas,
		Views:          cmd.Views,
		ExcludeViews:   cmd.ExcludeViews,
	})
	if err != nil {
		return err
	}
	text, err := viewStructs.MarshalText()
	if err != nil {
		return err
	}
	if len(text) == 0 {
		return nil
	}
	if cmd.PackageName == "" && cmd.Filename != "" {
		err = os.MkdirAll(filepath.Dir(cmd.Filename), 0755)
		if err != nil {
			return err
		}
		dirname := filepath.Base(filepath.Dir(cmd.Filename))
		if dirname != "" && dirname != "." {
			cmd.PackageName = strings.ReplaceAll(dirname, " ", "_")
		}
	}
	if cmd.PackageName == "" {
		cmd.PackageName = "_"
	}
	text, err = format.Source(text)
	if err != nil {
		return err
	}

	out := cmd.Stdout
	if cmd.Filename != "" {
		file, err := os.OpenFile(cmd.Filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer file.Close()
		out = file
	}
	_, err = io.WriteString(out, "package "+cmd.PackageName+"\n\nimport \"github.com/bokwoon95/sq\"\n\n")
	if err != nil {
		return err
	}
	_, err = out.Write(text)
	if err != nil {
		return err
	}
	return nil
}
