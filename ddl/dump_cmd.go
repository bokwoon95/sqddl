package ddl

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bokwoon95/sq"
	"github.com/bokwoon95/sqddl/internal/pqarray"
	"golang.org/x/sync/errgroup"
)

// DumpCmd implements the `sqddl dump` subcommand.
type DumpCmd struct {
	// (Required) DB is the database to dump.
	DB *sql.DB

	// (Required) Dialect is the database dialect.
	Dialect string

	// OutputDir is the output directory where the files will be created. If empty,
	// the command creates files in the current working directory.
	OutputDir string

	// Stderr specifies the command's standard error. If nil, the command
	// writes to os.Stderr.
	Stderr io.Writer

	// HistoryTable is the name of the migration history table. If empty, the
	// default history table name will be "sqddl_history".
	HistoryTable string

	// If non-empty, Zip specifies the name of the zip file to dump the
	// contents into. The .zip suffix is optional.
	Zip string

	// If non-empty, Tgz specifies the name of the tar gzip file to dump the
	// contents into. The .tgz suffix is optional.
	Tgz string

	// If SchemaOnly is true, DumpCmd will only dump the schema.
	SchemaOnly bool

	// If DataOnly is true, DumpCmd will only dump table data.
	DataOnly bool

	// Nullstring specifies the string that is used in CSV to represent NULL.
	// Leave blank to use `\N`.
	Nullstring string

	// Binaryprefix specifies the prefix that is used in CSV to denote a
	// hexadecimal binary literal (e.g. 0xa55cfae). Leave blank to use `0x`.
	Binaryprefix string

	// Schemas is the list of schemas that will be included in the dump.
	Schemas []string

	// ExcludeSchemas is the list of schemas that will be excluded from the
	// dump.
	ExcludeSchemas []string

	// Tables is the list of tables that will be included in the dump.
	Tables []string

	// ExcludeTables is the list of tables that will be excluded from the dump.
	ExcludeTables []string

	// SubsetQueries holds the initial subset queries.
	SubsetQueries []string

	// ExtendedSubsetQueries holds the initial extended subset queries.
	ExtendedSubsetQueries []string

	// (Postgres only) Dump arrays as JSON arrays.
	ArrayAsJSON bool

	// (Postgres only) Dump UUIDs as bytes (in hexadecimal form e.g. 0x267f4bdb50a041399704c26a16f8f019).
	UUIDAsBytes bool

	// Ctx is the command's context.
	Ctx context.Context

	db           string // -db flag.
	catalog      *Catalog
	cache        *CatalogCache
	tableQueries []tableQuery
}

// TODO: data masking with -mask flag.

// DumpCommand creates a new DumpCmd with the given arguments. E.g.
//
//   sqddl dump -db <DATABASE_URL> [FLAGS]
//
//   DumpCommand("-db", "postgres://user:pass@localhost:5432/sakila", "-output-dir", "./db")
func DumpCommand(args ...string) (*DumpCmd, error) {
	var cmd DumpCmd
	var schemas, excludeSchemas, tables, excludeTables string
	var subsetQueries, extendedSubsetQueries strslice
	flagset := flag.NewFlagSet("", flag.ContinueOnError)
	flagset.StringVar(&cmd.db, "db", "", "(required) Database URL/DSN.")
	flagset.StringVar(&cmd.OutputDir, "output-dir", "", "Output directory. Leave blank to use the current working directory.")
	flagset.StringVar(&cmd.HistoryTable, "history-table", "sqddl_history", "Name of migration history table.")
	flagset.StringVar(&cmd.Zip, "zip", "", "Name of the zip file to dump the contents into. The .zip suffix is optional.")
	flagset.StringVar(&cmd.Tgz, "tgz", "", "Name of the tgz file to dump the contents into. The .tgz suffix is optional.")
	flagset.BoolVar(&cmd.SchemaOnly, "schema-only", false, "Only dump the schema.")
	flagset.BoolVar(&cmd.DataOnly, "data-only", false, "Only dump the data.")
	flagset.StringVar(&cmd.Nullstring, "nullstring", "\\N", "The string used in CSV to represent NULL.")
	flagset.StringVar(&cmd.Binaryprefix, "binaryprefix", "0x", "The string used in CSV to prefix a hexadecimal binary literal (e.g. 0xa55cfae)")
	flagset.StringVar(&schemas, "schemas", "", "Comma separated list of schema names to include (if empty all schema names will be included).")
	flagset.StringVar(&excludeSchemas, "exclude-schemas", "", "Comma separated list of schema names to exclude.")
	flagset.StringVar(&tables, "tables", "", "Comma separated list of table names to include (if empty all table names will be included).")
	flagset.StringVar(&excludeTables, "exclude-tables", "", "Comma seprated list of table names to exclude.")
	flagset.Var(&subsetQueries, "subset", "A query that pulls in a subset of a table which the rest of the dump will be derived from. This flag can be specified multiple times.")
	flagset.Var(&extendedSubsetQueries, "extended-subset", "A query that pulls in a subset of a table which the rest of the dump will be derived from. This flag can be specified multiple times.")
	flagset.BoolVar(&cmd.ArrayAsJSON, "array-as-json", false, "(Postgres only) Dump arrays as JSON arrays.")
	flagset.BoolVar(&cmd.UUIDAsBytes, "uuid-as-bytes", false, "(Postgres only) Dump UUIDs as bytes (in hexadecimal form e.g. 0x267f4bdb50a041399704c26a16f8f019).")
	flagset.Usage = func() {
		fmt.Fprint(flagset.Output(), `Usage:
  sqddl dump -db <DATABASE_URL> [FLAGS]
  sqddl dump -db 'postgres://username:password@localhost:5432/sakila' -dir dump
  sqddl dump -db 'postgres://username:password@localhost:5432/sakila' -zip dump.zip
  sqddl dump -db 'postgres://username:password@localhost:5432/sakila' -tgz dump.tgz
  sqddl dump \
    -db 'postgres://username:password@localhost:5432/sakila' \
    -dir dump \
    -extended-subset 'SELECT {*} FROM {film} ORDER BY film_id LIMIT 10' \
    -subset 'SELECT {*} FROM {actor}'
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
	if schemas != "" {
		cmd.Schemas = strings.Split(schemas, ",")
	}
	if excludeSchemas != "" {
		cmd.ExcludeSchemas = strings.Split(excludeSchemas, ",")
	}
	if tables != "" {
		cmd.Tables = strings.Split(tables, ",")
	}
	if excludeTables != "" {
		cmd.ExcludeTables = strings.Split(excludeTables, ",")
	}
	cmd.SubsetQueries = subsetQueries
	cmd.ExtendedSubsetQueries = extendedSubsetQueries
	return &cmd, nil
}

// Run runs the DumpCmd.
func (cmd *DumpCmd) Run() error {
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
	if cmd.Nullstring == "" {
		cmd.Nullstring = `\N`
	}
	if cmd.Binaryprefix == "" {
		cmd.Binaryprefix = `0x`
	}
	if cmd.db != "" {
		defer cmd.DB.Close()
	}
	if cmd.Ctx == nil {
		cmd.Ctx = context.Background()
	}
	buf := bufpool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufpool.Put(buf)

	dbi := NewDatabaseIntrospector(cmd.Dialect, cmd.DB)
	dbi.Schemas = cmd.Schemas
	dbi.ExcludeSchemas = cmd.ExcludeSchemas
	dbi.Tables = cmd.Tables
	dbi.ExcludeTables = append(cmd.ExcludeTables, cmd.HistoryTable)
	cmd.catalog = &Catalog{}
	err := dbi.WriteCatalog(cmd.catalog)
	if err != nil {
		return err
	}
	cmd.cache = NewCatalogCache(cmd.catalog)
	if cmd.OutputDir != "" {
		err = os.MkdirAll(cmd.OutputDir, 0755)
		if err != nil {
			return err
		}
	}

	if !cmd.SchemaOnly {
		if len(cmd.SubsetQueries) > 0 || len(cmd.ExtendedSubsetQueries) > 0 {
			subsetter, err := NewInMemorySubsetter(cmd.Dialect, cmd.DB, dbi.Filter)
			if err != nil {
				return err
			}
			for _, query := range cmd.SubsetQueries {
				err = subsetter.Subset(query)
				if err != nil {
					return err
				}
			}
			for _, query := range cmd.ExtendedSubsetQueries {
				err = subsetter.ExtendedSubset(query)
				if err != nil {
					return err
				}
			}
			for _, table := range subsetter.Tables() {
				cmd.tableQueries = append(cmd.tableQueries, tableQuery{
					table: table,
					query: subsetter.Query(table.TableSchema, table.TableName),
				})
			}
		} else {
			for i := range cmd.catalog.Schemas {
				schema := &cmd.catalog.Schemas[i]
				for j := range schema.Tables {
					table := &schema.Tables[j]
					if cmd.Dialect == DialectSQLite && isVirtualTable(table) {
						continue
					}
					buf.Reset()
					buf.WriteString("SELECT ")
					written := false
					for _, column := range table.Columns {
						if column.GeneratedExpr != "" || column.IsGenerated {
							continue
						}
						if !written {
							written = true
						} else {
							buf.WriteString(", ")
						}
						buf.WriteString(sq.QuoteIdentifier(cmd.Dialect, column.ColumnName))
					}
					buf.WriteString(" FROM ")
					if table.TableSchema != "" && table.TableSchema != cmd.catalog.CurrentSchema {
						buf.WriteString(sq.QuoteIdentifier(cmd.Dialect, table.TableSchema) + ".")
					}
					buf.WriteString(sq.QuoteIdentifier(cmd.Dialect, table.TableName))
					pkey := cmd.cache.GetPrimaryKey(table)
					if pkey != nil {
						buf.WriteString(" ORDER BY ")
						for j, column := range pkey.Columns {
							if j > 0 {
								buf.WriteString(", ")
							}
							buf.WriteString(sq.QuoteIdentifier(cmd.Dialect, column))
						}
					}
					cmd.tableQueries = append(cmd.tableQueries, tableQuery{
						table: table,
						query: buf.String(),
					})
				}
			}
		}
	}

	if cmd.Zip != "" {
		err = cmd.dumpZip(cmd.Zip)
		if err != nil {
			return err
		}
	} else if cmd.Tgz != "" {
		err = cmd.dumpTgz(cmd.Tgz)
		if err != nil {
			return err
		}
	} else {
		err = cmd.dumpDir()
		if err != nil {
			return err
		}
	}
	return nil
}

func (cmd *DumpCmd) dumpZip(name string) error {
	zipName := filepath.Join(cmd.OutputDir, strings.TrimSuffix(name, ".zip")+".zip")
	zipFile, err := os.OpenFile(zipName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer zipFile.Close()
	fmt.Fprintln(cmd.Stderr, zipName)
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()
	dumpSchema, dumpData := true, true
	if cmd.SchemaOnly {
		dumpData = false
	} else if cmd.DataOnly {
		dumpSchema = false
	}
	if dumpSchema {
		// schema.json
		filename := "schema.json"
		file, err := zipWriter.Create(filename)
		if err != nil {
			return err
		}
		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		err = encoder.Encode(cmd.catalog)
		if err != nil {
			return err
		}
		// schema.sql
		filename = "schema.sql"
		file, err = zipWriter.Create(filename)
		if err != nil {
			return err
		}
		err = cmd.dumpSchema(file)
		if err != nil {
			return err
		}
		// indexes.sql
		filename = "indexes.sql"
		file, err = zipWriter.Create(filename)
		if err != nil {
			return err
		}
		err = cmd.dumpIndexes(file)
		if err != nil {
			return err
		}
		// constraints.sql
		filename = "constraints.sql"
		file, err = zipWriter.Create(filename)
		if err != nil {
			return err
		}
		err = cmd.dumpConstraints(file)
		if err != nil {
			return err
		}
	}
	if dumpData {
		for _, q := range cmd.tableQueries {
			filename := q.table.TableName + ".csv"
			if q.table.TableSchema != "" && q.table.TableSchema != cmd.catalog.CurrentSchema {
				filename = q.table.TableSchema + "." + filename
			}
			file, err := zipWriter.Create(filename)
			if err != nil {
				return err
			}
			err = cmd.dumpCSV(cmd.Ctx, file, q.table, q.query)
			if err != nil {
				return err
			}
		}
	}
	err = zipWriter.Close()
	if err != nil {
		return err
	}
	err = zipFile.Close()
	if err != nil {
		return err
	}
	return nil
}

func (cmd *DumpCmd) dumpTgz(name string) error {
	tgzName := filepath.Join(cmd.OutputDir, strings.TrimSuffix(name, ".tgz")+".tgz")
	tgzFile, err := os.OpenFile(tgzName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer tgzFile.Close()
	fmt.Fprintln(cmd.Stderr, tgzName)
	gzipWriter, _ := gzip.NewWriterLevel(tgzFile, gzip.BestCompression)
	defer gzipWriter.Close()
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()
	buf := bufpool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufpool.Put(buf)
	dumpSchema, dumpData := true, true
	if cmd.SchemaOnly {
		dumpData = false
	} else if cmd.DataOnly {
		dumpSchema = false
	}
	if dumpSchema {
		// schema.json
		buf.Reset()
		encoder := json.NewEncoder(buf)
		encoder.SetIndent("", "  ")
		err = encoder.Encode(cmd.catalog)
		if err != nil {
			return err
		}
		filename := "schema.json"
		err = tarWriter.WriteHeader(&tar.Header{Name: filename, Mode: 0644, Size: int64(buf.Len())})
		if err != nil {
			return err
		}
		_, err = buf.WriteTo(tarWriter)
		if err != nil {
			return err
		}
		// schema.sql
		buf.Reset()
		err = cmd.dumpSchema(buf)
		if err != nil {
			return err
		}
		filename = "schema.sql"
		err = tarWriter.WriteHeader(&tar.Header{Name: filename, Mode: 0644, Size: int64(buf.Len())})
		if err != nil {
			return err
		}
		_, err = buf.WriteTo(tarWriter)
		if err != nil {
			return err
		}
		// indexes.sql
		buf.Reset()
		err = cmd.dumpIndexes(buf)
		if err != nil {
			return err
		}
		filename = "indexes.sql"
		err = tarWriter.WriteHeader(&tar.Header{Name: filename, Mode: 0644, Size: int64(buf.Len())})
		if err != nil {
			return err
		}
		_, err = buf.WriteTo(tarWriter)
		if err != nil {
			return err
		}
		// constraints.sql
		buf.Reset()
		err = cmd.dumpConstraints(buf)
		if err != nil {
			return err
		}
		filename = "constraints.sql"
		err = tarWriter.WriteHeader(&tar.Header{Name: filename, Mode: 0644, Size: int64(buf.Len())})
		if err != nil {
			return err
		}
		_, err = buf.WriteTo(tarWriter)
		if err != nil {
			return err
		}
	}
	if dumpData {
		for _, q := range cmd.tableQueries {
			filename := q.table.TableName + ".csv"
			if q.table.TableSchema != "" && q.table.TableSchema != cmd.catalog.CurrentSchema {
				filename = q.table.TableSchema + "." + filename
			}
			buf.Reset()
			err = cmd.dumpCSV(cmd.Ctx, buf, q.table, q.query)
			if err != nil {
				return err
			}
			err = tarWriter.WriteHeader(&tar.Header{Name: filename, Mode: 0644, Size: int64(buf.Len())})
			if err != nil {
				return err
			}
			_, err = buf.WriteTo(tarWriter)
			if err != nil {
				return err
			}
		}
	}
	err = tarWriter.Close()
	if err != nil {
		return err
	}
	err = gzipWriter.Close()
	if err != nil {
		return err
	}
	err = tgzFile.Close()
	if err != nil {
		return err
	}
	return nil
}

func (cmd *DumpCmd) dumpDir() error {
	dumpSchema, dumpData := true, true
	if cmd.SchemaOnly {
		dumpData = false
	} else if cmd.DataOnly {
		dumpSchema = false
	}
	if dumpSchema {
		// schema.json
		filename := filepath.Join(cmd.OutputDir, "schema.json")
		file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		err = encoder.Encode(cmd.catalog)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.Stderr, filename)
		err = file.Close()
		if err != nil {
			return err
		}
		// schema.sql
		filename = filepath.Join(cmd.OutputDir, "schema.sql")
		file, err = os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		err = cmd.dumpSchema(file)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.Stderr, filename)
		err = file.Close()
		if err != nil {
			return err
		}
		// indexes.sql
		filename = filepath.Join(cmd.OutputDir, "indexes.sql")
		file, err = os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		err = cmd.dumpIndexes(file)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.Stderr, filename)
		err = file.Close()
		if err != nil {
			return err
		}
		// constraints.sql
		filename = filepath.Join(cmd.OutputDir, "constraints.sql")
		file, err = os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		err = cmd.dumpConstraints(file)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.Stderr, filename)
		err = file.Close()
		if err != nil {
			return err
		}
	}
	if dumpData {
		g, ctx := errgroup.WithContext(cmd.Ctx)
		for _, q := range cmd.tableQueries {
			q := q
			g.Go(func() error {
				filename := filepath.Join(cmd.OutputDir, q.table.TableName+".csv")
				if q.table.TableSchema != "" && q.table.TableSchema != cmd.catalog.CurrentSchema {
					filename = filepath.Join(cmd.OutputDir, q.table.TableSchema+"."+q.table.TableName+".csv")
				}
				file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
				if err != nil {
					return err
				}
				err = cmd.dumpCSV(ctx, file, q.table, q.query)
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.Stderr, filename)
				return file.Close()
			})
		}
		err := g.Wait()
		if err != nil {
			return err
		}
	}
	return nil
}

func (cmd *DumpCmd) dumpSchema(w io.Writer) error {
	buf, isBuffer := w.(*bytes.Buffer)
	if !isBuffer {
		buf = bufpool.Get().(*bytes.Buffer)
		buf.Reset()
		defer bufpool.Put(buf)
	}

	// CREATE EXTENSION.
	for _, extension := range cmd.catalog.Extensions {
		if buf.Len() > 0 {
			buf.WriteString("\n")
		}
		buf.WriteString("CREATE EXTENSION IF NOT EXISTS " + sq.QuoteIdentifier(cmd.Dialect, extension) + ";\n")
	}

	// CREATE SCHEMA.
	for _, schema := range cmd.catalog.Schemas {
		if schema.SchemaName != "" {
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			writeCreateSchema(cmd.Dialect, buf, schema.SchemaName)
		}
		var schemaName string
		if schema.SchemaName != "" && schema.SchemaName != cmd.catalog.CurrentSchema {
			schemaName = sq.QuoteIdentifier(cmd.Dialect, schema.SchemaName)
		}

		// CREATE ENUM.
		for _, enum := range schema.Enums {
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			enumName := sq.QuoteIdentifier(cmd.Dialect, enum.EnumName)
			if schemaName != "" {
				enumName = schemaName + "." + enumName
			}
			buf.WriteString("CREATE TYPE " + enumName + " AS ENUM (")
			for i, label := range enum.EnumLabels {
				if i > 0 {
					buf.WriteString(", ")
				}
				buf.WriteString("'" + sq.EscapeQuote(label, '\'') + "'")
			}
			buf.WriteString(");\n")
		}

		// CREATE DOMAIN.
		for _, domain := range schema.Domains {
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			domainName := sq.QuoteIdentifier(cmd.Dialect, domain.DomainName)
			if schemaName != "" {
				domainName = schemaName + "." + domainName
			}
			buf.WriteString("CREATE DOMAIN " + domainName + " AS " + domain.UnderlyingType)
			if domain.IsNotNull {
				buf.WriteString(" NOT NULL")
			}
			if domain.CollationName != "" && domain.CollationName != cmd.catalog.DefaultCollation {
				buf.WriteString(` COLLATE "` + sq.EscapeQuote(domain.CollationName, '"') + `"`)
			}
			if domain.ColumnDefault != "" {
				buf.WriteString(" DEFAULT " + domain.ColumnDefault)
			}
			if len(domain.CheckExprs) > 0 {
				for i, checkExpr := range domain.CheckExprs {
					var constraintName string
					if i < len(domain.CheckNames) {
						constraintName = domain.CheckNames[i]
					}
					if constraintName != "" {
						buf.WriteString(" CONSTRAINT " + sq.QuoteIdentifier(cmd.Dialect, constraintName))
					}
					buf.WriteString(" CHECK " + wrapBrackets(checkExpr))
				}
			}
			buf.WriteString(";\n")
		}
	}

	for i := range cmd.catalog.Schemas {
		schema := &cmd.catalog.Schemas[i]
		// CREATE TABLE.
		for j := range schema.Tables {
			table := &schema.Tables[j]
			if table.IsVirtual && table.SQL == "" {
				continue
			}
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			writeCreateTable(cmd.Dialect, buf, cmd.catalog.CurrentSchema, cmd.catalog.DefaultCollation, table, false)
		}
	}

	if isBuffer {
		return nil
	}
	_, err := buf.WriteTo(w)
	if err != nil {
		return err
	}
	return nil
}

func (cmd *DumpCmd) dumpIndexes(w io.Writer) error {
	buf, isBuffer := w.(*bytes.Buffer)
	if !isBuffer {
		buf = bufpool.Get().(*bytes.Buffer)
		buf.Reset()
		defer bufpool.Put(buf)
	}
	for i := range cmd.catalog.Schemas {
		schema := &cmd.catalog.Schemas[i]
		for j := range schema.Tables {
			table := &schema.Tables[j]
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			buf.WriteString("-- ")
			if table.TableSchema != "" && table.TableSchema != cmd.catalog.CurrentSchema {
				buf.WriteString(table.TableSchema + ".")
			}
			buf.WriteString(table.TableName + "\n")
			for k := range table.Indexes {
				index := &table.Indexes[k]
				writeCreateIndex(cmd.Dialect, buf, cmd.catalog.CurrentSchema, index, false)
			}
		}
	}
	if isBuffer {
		return nil
	}
	_, err := buf.WriteTo(w)
	if err != nil {
		return err
	}
	return nil
}

func (cmd *DumpCmd) dumpConstraints(w io.Writer) error {
	buf, isBuffer := w.(*bytes.Buffer)
	if !isBuffer {
		buf = bufpool.Get().(*bytes.Buffer)
		buf.Reset()
		defer bufpool.Put(buf)
	}
	// SQLite doesn't support ALTER TABLE ADD CONSTRAINT.
	if cmd.Dialect == DialectSQLite {
		return nil
	}
	for i := range cmd.catalog.Schemas {
		schema := &cmd.catalog.Schemas[i]
		// PRIMARY KEY, UNIQUE, CHECK, EXCLUDE
		for j := range schema.Tables {
			table := &schema.Tables[j]
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			buf.WriteString("-- ")
			if table.TableSchema != "" && table.TableSchema != cmd.catalog.CurrentSchema {
				buf.WriteString(table.TableSchema + ".")
			}
			buf.WriteString(table.TableName + "\n")
			for k := range table.Constraints {
				constraint := &table.Constraints[k]
				if constraint.ConstraintType == PRIMARY_KEY && (cmd.Dialect == DialectMySQL || cmd.Dialect == DialectSQLServer) {
					continue
				}
				if constraint.ConstraintType == FOREIGN_KEY {
					continue
				}
				tableName := sq.QuoteIdentifier(cmd.Dialect, constraint.TableName)
				if constraint.TableSchema != "" && constraint.TableSchema != cmd.catalog.CurrentSchema {
					tableName = sq.QuoteIdentifier(cmd.Dialect, constraint.TableSchema) + "." + tableName
				}
				buf.WriteString("ALTER TABLE " + tableName + " ADD ")
				writeConstraintDefinition(cmd.Dialect, buf, cmd.catalog.CurrentSchema, constraint)
				buf.WriteString(";\n")
			}
		}
		// FOREIGN KEY
		// Foreign keys are added at the very end in order to avoid problems
		// with circular references.
		for j := range schema.Tables {
			table := &schema.Tables[j]
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			buf.WriteString("-- ")
			if table.TableSchema != "" && table.TableSchema != cmd.catalog.CurrentSchema {
				buf.WriteString(table.TableSchema + ".")
			}
			buf.WriteString(table.TableName + "\n")
			fkeys := cmd.cache.GetForeignKeys(table)
			for _, fkey := range fkeys {
				tableName := sq.QuoteIdentifier(cmd.Dialect, fkey.TableName)
				if fkey.TableSchema != "" && fkey.TableSchema != cmd.catalog.CurrentSchema {
					tableName = sq.QuoteIdentifier(cmd.Dialect, fkey.TableSchema) + "." + tableName
				}
				buf.WriteString("ALTER TABLE " + tableName + " ADD ")
				writeConstraintDefinition(cmd.Dialect, buf, cmd.catalog.CurrentSchema, fkey)
				buf.WriteString(";\n")
			}
		}
	}
	if isBuffer {
		return nil
	}
	_, err := buf.WriteTo(w)
	if err != nil {
		return err
	}
	return nil
}

func (cmd *DumpCmd) dumpCSV(ctx context.Context, w io.Writer, table *Table, query string) error {
	headers := make([]string, 0, len(table.Columns))
	columnTypes := make([]string, 0, len(table.Columns))
	scanDest := make([]any, 0, len(table.Columns))
	for _, column := range table.Columns {
		if column.GeneratedExpr != "" || column.IsGenerated {
			continue
		}
		headers = append(headers, column.ColumnName)
		columnType, _, _ := normalizeColumnType(cmd.Dialect, column.ColumnType)
		if cmd.Dialect == DialectMySQL {
			if strings.HasSuffix(columnType, " UNSIGNED") {
				columnType = strings.TrimSuffix(columnType, " UNSIGNED")
			} else {
				columnType = strings.TrimSuffix(columnType, " SIGNED")
			}
		}
		columnTypes = append(columnTypes, columnType)
		if cmd.Dialect == DialectSQLite {
			switch columnType {
			case "DATETIME", "TIMESTAMP":
				scanDest = append(scanDest, &sql.NullTime{})
			default:
				scanDest = append(scanDest, &sql.NullString{})
			}
			continue
		}
		if cmd.Dialect == DialectPostgres && cmd.ArrayAsJSON {
			matched := true
			switch columnType {
			case "BOOLEAN[]":
				scanDest = append(scanDest, &pqarray.BoolArray{})
			case "SMALLINT[]", "INT[]", "INTEGER[]", "BIGINT[]":
				scanDest = append(scanDest, &pqarray.Int64Array{})
			case "NUMERIC[]", "REAL[]", "DOUBLE PRECISION[]":
				scanDest = append(scanDest, &pqarray.Float64Array{})
			case "VARCHAR[]", "CHAR[]", "TEXT[]":
				scanDest = append(scanDest, &pqarray.StringArray{})
			default:
				matched = false
			}
			if matched {
				continue
			}
		}
		switch columnType {
		case "BYTEA", "BINARY", "VARBINARY", "TINYBLOB", "BLOB", "MEDIUMBLOB", "LONGBLOB", "VARBIT":
			scanDest = append(scanDest, &[]byte{})
		case "BOOLEAN", "BIT":
			scanDest = append(scanDest, &sql.NullBool{})
		case "NUMERIC", "FLOAT", "REAL", "DOUBLE PRECISION":
			scanDest = append(scanDest, &sql.NullFloat64{})
		case "TINYINT", "SMALLINT", "MEDIUMINT", "INT", "INTEGER", "BIGINT":
			scanDest = append(scanDest, &sql.NullInt64{})
		case "TINYTEXT", "TEXT", "MEDIUMTEXT", "LONGTEXT", "CHAR", "VARCHAR", "NVARCHAR", "UUID", "UNIQUEIDENTIFIER", "JSON", "JSONB":
			scanDest = append(scanDest, &sql.NullString{})
		case "DATE", "TIME", "TIMETZ", "DATETIME", "DATETIME2", "SMALLDATETIME", "DATETIMEOFFSET", "TIMESTAMP", "TIMESTAMPTZ":
			scanDest = append(scanDest, &sql.NullTime{})
		default:
			scanDest = append(scanDest, &sql.NullString{})
		}
	}
	rows, err := cmd.DB.Query(query)
	if err != nil {
		return fmt.Errorf("%w\n"+query, err)
	}
	defer rows.Close()
	csvWriter := csv.NewWriter(w)
	err = csvWriter.Write(headers)
	if err != nil {
		return err
	}
	record := make([]string, len(scanDest))
	for rows.Next() {
		err = rows.Scan(scanDest...)
		if err != nil {
			return err
		}
		for i, value := range scanDest {
			switch value := value.(type) {
			case *[]byte:
				if len(*value) == 0 {
					record[i] = cmd.Nullstring
					continue
				}
				record[i] = cmd.Binaryprefix + hex.EncodeToString(*value)
			case *sql.NullBool:
				if !value.Valid {
					record[i] = cmd.Nullstring
					continue
				}
				if value.Bool {
					record[i] = "1"
					continue
				}
				record[i] = "0"
			case *sql.NullFloat64:
				if !value.Valid {
					record[i] = cmd.Nullstring
					continue
				}
				record[i] = strconv.FormatFloat(value.Float64, 'f', -1, 64)
			case *sql.NullInt64:
				if !value.Valid {
					record[i] = cmd.Nullstring
					continue
				}
				record[i] = strconv.FormatInt(value.Int64, 10)
			case *sql.NullString:
				if !value.Valid {
					record[i] = cmd.Nullstring
					continue
				}
				columnType := columnTypes[i]
				switch columnType {
				case "BINARY", "VARBINARY", "TINYBLOB", "BLOB", "MEDIUMBLOB", "LONGBLOB":
					record[i] = cmd.Binaryprefix + hex.EncodeToString([]byte(value.String))
					continue
				case "UUID":
					if cmd.Dialect == DialectSQLite && len(value.String) == 16 {
						record[i] = cmd.Binaryprefix + hex.EncodeToString([]byte(value.String))
						continue
					}
					if cmd.Dialect == DialectPostgres && cmd.UUIDAsBytes {
						record[i] = cmd.Binaryprefix + strings.ReplaceAll(value.String, "-", "")
						continue
					}
				}
				record[i] = value.String
			case *sql.NullTime:
				if !value.Valid {
					record[i] = cmd.Nullstring
					continue
				}
				record[i] = value.Time.UTC().Format("2006-01-02 15:04:05")
			case *pqarray.BoolArray, *pqarray.Int64Array, *pqarray.Float64Array, *pqarray.StringArray:
				// If we reach here it means cmd.ArrayAsJSON is true. If
				// cmd.ArrayAsJSON is false, then the value would have been
				// scanned as an *sql.NullString instead.
				b, err := json.Marshal(value)
				if err != nil {
					return err
				}
				record[i] = string(b)
			default:
				panic("unreachable")
			}
		}
		err = csvWriter.Write(record)
		if err != nil {
			return err
		}
	}
	csvWriter.Flush()
	err = csvWriter.Error()
	if err != nil {
		return err
	}
	return nil
}

func writeCreateSchema(dialect string, buf *bytes.Buffer, schemaName string) {
	switch dialect {
	case DialectPostgres, DialectMySQL:
		buf.WriteString("CREATE SCHEMA IF NOT EXISTS " + sq.QuoteIdentifier(dialect, schemaName) + ";\n")
	case DialectSQLServer:
		// SQLServer doesn't allow CREATE SCHEMA to exist with other
		// SQL statements so we wrap it in an EXEC() in order to get
		// around that restriction
		// (https://stackoverflow.com/q/5748056).
		buf.WriteString("IF SCHEMA_ID('" + sq.EscapeQuote(schemaName, '\'') + "') IS NULL EXEC('CREATE SCHEMA " + sq.EscapeQuote(sq.QuoteIdentifier(dialect, schemaName), '\'') + "');\n")
	}
}

func writeCreateTable(dialect string, buf *bytes.Buffer, currentSchema, defaultCollation string, table *Table, includeConstraints bool) {
	if table.SQL != "" {
		buf.WriteString(table.SQL + "\n")
		return
	}
	tableName := sq.QuoteIdentifier(dialect, table.TableName)
	if table.TableSchema != "" && table.TableSchema != currentSchema {
		tableName = sq.QuoteIdentifier(dialect, table.TableSchema) + "." + tableName
	}
	buf.WriteString("CREATE TABLE " + tableName + " (")
	columnWritten := false
	for i := range table.Columns {
		column := &table.Columns[i]
		if column.Ignore || (column.IsGenerated && column.GeneratedExpr == "") {
			continue
		}
		if !columnWritten {
			columnWritten = true
			buf.WriteString("\n    ")
		} else {
			buf.WriteString("\n    ,")
		}
		writeColumnDefinition(dialect, buf, defaultCollation, column, false)
	}
	if dialect == DialectSQLite {
		newlineSeparatorWritten := false
		for i := range table.Constraints {
			constraint := &table.Constraints[i]
			if constraint.Ignore {
				continue
			}
			if constraint.ConstraintType == PRIMARY_KEY && len(constraint.Columns) == 1 {
				continue
			}
			if !newlineSeparatorWritten {
				newlineSeparatorWritten = true
				buf.WriteString("\n\n    ,")
			} else {
				buf.WriteString("\n    ,")
			}
			writeConstraintDefinition(dialect, buf, currentSchema, constraint)
		}
		buf.WriteString("\n);\n")
		return
	}
	if !includeConstraints {
		if dialect == DialectMySQL || dialect == DialectSQLServer {
			for i := range table.Constraints {
				constraint := &table.Constraints[i]
				if constraint.ConstraintType == PRIMARY_KEY && !constraint.Ignore {
					buf.WriteString("\n\n    ,")
					writeConstraintDefinition(dialect, buf, currentSchema, constraint)
					break
				}
			}
		}
		buf.WriteString("\n);\n")
		return
	}
	newlineSeparatorWritten := false
	for i := range table.Constraints {
		constraint := &table.Constraints[i]
		if constraint.Ignore {
			continue
		}
		if constraint.ConstraintType == FOREIGN_KEY {
			continue
		}
		if !newlineSeparatorWritten {
			newlineSeparatorWritten = true
			buf.WriteString("\n\n    ,")
		} else {
			buf.WriteString("\n    ,")
		}
		writeConstraintDefinition(dialect, buf, currentSchema, constraint)
	}
	buf.WriteString("\n);\n")
}

func writeCreateIndex(dialect string, buf *bytes.Buffer, currentSchema string, index *Index, createConcurrently bool) {
	if index.SQL != "" {
		sql := index.SQL
		if createConcurrently && dialect == DialectPostgres {
			indexName := sq.QuoteIdentifier(dialect, index.IndexName)
			sql = strings.Replace(sql, "CREATE INDEX "+indexName, "CREATE INDEX CONCURRENTLY "+indexName, 1)
		}
		buf.WriteString(sql + "\n")
		return
	}
	// CREATE INDEX.
	buf.WriteString("CREATE ")
	writeIndexDefinition(dialect, buf, currentSchema, index, createConcurrently, false)
	buf.WriteString(";\n")
}

func writeIndexDefinition(dialect string, buf *bytes.Buffer, currentSchema string, index *Index, createConcurrently, createInline bool) {
	var isFulltextOrSpatialIndex bool
	if dialect == DialectMySQL {
		if strings.EqualFold(index.IndexType, "FULLTEXT") || strings.EqualFold(index.IndexName, "SPATIAL") {
			isFulltextOrSpatialIndex = true
		}
	}
	indexName := sq.QuoteIdentifier(dialect, index.IndexName)
	if index.IsUnique {
		buf.WriteString("UNIQUE ")
	} else if isFulltextOrSpatialIndex {
		buf.WriteString(index.IndexType + " ")
	}
	buf.WriteString("INDEX ")
	if createConcurrently && dialect == DialectPostgres {
		buf.WriteString("CONCURRENTLY ")
	}
	buf.WriteString(indexName)
	if index.IndexType != "" && dialect == DialectMySQL && !strings.EqualFold(index.IndexType, "BTREE") && !isFulltextOrSpatialIndex {
		if !createInline {
			buf.WriteString(" USING")
		}
		buf.WriteString(" " + index.IndexType)
	}
	if !createInline {
		buf.WriteString(" ON ")
		if index.TableSchema != "" && index.TableSchema != currentSchema {
			buf.WriteString(sq.QuoteIdentifier(dialect, index.TableSchema) + ".")
		}
		buf.WriteString(sq.QuoteIdentifier(dialect, index.TableName))
	}
	if index.IndexType != "" && dialect == DialectPostgres && !strings.EqualFold(index.IndexType, "BTREE") {
		buf.WriteString(" USING " + index.IndexType)
	}
	buf.WriteString(" (")
	for i, column := range index.Columns {
		if i > 0 {
			buf.WriteString(", ")
		}
		if wrappedInBrackets(column) {
			buf.WriteString(column)
		} else {
			buf.WriteString(sq.QuoteIdentifier(dialect, column))
		}
		if i < len(index.Descending) && index.Descending[i] {
			buf.WriteString(" DESC")
		}
	}
	buf.WriteString(")")
	if len(index.IncludeColumns) > 0 && (dialect == DialectPostgres || dialect == DialectSQLServer) {
		buf.WriteString(" INCLUDE (")
		writeColumnNames(dialect, buf, index.IncludeColumns)
		buf.WriteString(")")
	}
	if index.Predicate != "" && (dialect == DialectSQLite || dialect == DialectPostgres || dialect == DialectSQLServer) {
		buf.WriteString(" WHERE " + index.Predicate)
	}
}

func writeColumnDefinition(dialect string, buf *bytes.Buffer, defaultCollation string, column *Column, columnLevelConstraint bool) {
	isSQLServerGeneratedColumn := dialect == DialectSQLServer && column.GeneratedExpr != ""
	// ColumnName
	buf.WriteString(sq.QuoteIdentifier(dialect, column.ColumnName))
	// ColumnType
	if column.DomainName != "" {
		buf.WriteString(" " + column.DomainName)
	} else if column.IsEnum {
		buf.WriteString(" " + column.ColumnType)
	} else if column.ColumnType != "" && !isSQLServerGeneratedColumn {
		buf.WriteString(" " + strings.ToUpper(column.ColumnType))
	}
	// PRIMARY KEY
	if column.IsPrimaryKey && (columnLevelConstraint || dialect == DialectSQLite) {
		buf.WriteString(" PRIMARY KEY")
	}
	// AUTO_INCREMENT
	if column.IsAutoincrement {
		switch dialect {
		case DialectSQLite:
			buf.WriteString(" AUTOINCREMENT")
		case DialectMySQL:
			buf.WriteString(" AUTO_INCREMENT")
		}
	}
	// NOT NULL
	if column.IsNotNull && !isSQLServerGeneratedColumn {
		buf.WriteString(" NOT NULL")
	}
	// UNIQUE
	if column.IsUnique && columnLevelConstraint {
		buf.WriteString(" UNIQUE")
	}
	// IDENTITY
	if column.ColumnIdentity != "" && (dialect == DialectPostgres || dialect == DialectSQLServer) {
		buf.WriteString(" " + column.ColumnIdentity)
	}
	// DEFAULT
	if column.ColumnDefault != "" {
		buf.WriteString(" DEFAULT " + column.ColumnDefault)
	}
	// ON UPDATE CURRENT TIMESTAMP
	if column.OnUpdateCurrentTimestamp && dialect == DialectMySQL {
		buf.WriteString(" ON UPDATE CURRENT_TIMESTAMP")
	}
	// COLLATE
	if column.CollationName != "" && column.CollationName != defaultCollation {
		if dialect == DialectPostgres {
			buf.WriteString(` COLLATE "` + sq.EscapeQuote(column.CollationName, '"') + `"`)
		} else {
			buf.WriteString(` COLLATE ` + column.CollationName)
		}
	}
	// GENERATED AS
	if column.GeneratedExpr != "" {
		generatedExpr := wrapBrackets(column.GeneratedExpr)
		switch dialect {
		case DialectPostgres:
			buf.WriteString(" GENERATED ALWAYS AS " + generatedExpr + " STORED")
		case DialectMySQL:
			buf.WriteString(" AS " + generatedExpr)
			if column.GeneratedExprStored {
				buf.WriteString(" STORED")
			}
		case DialectSQLServer:
			buf.WriteString(" AS " + generatedExpr)
			if column.GeneratedExprStored {
				buf.WriteString(" PERSISTED")
			}
		}
	}
	// REFERENCES
	if columnLevelConstraint && column.ReferencesTable != "" && column.ReferencesColumn != "" {
		buf.WriteString(" REFERENCES ")
		if column.ReferencesSchema != "" && column.ReferencesSchema != column.TableSchema {
			buf.WriteString(sq.QuoteIdentifier(dialect, column.ReferencesSchema) + ".")
		}
		buf.WriteString(sq.QuoteIdentifier(dialect, column.ReferencesTable) + " (" + sq.QuoteIdentifier(dialect, column.ReferencesColumn) + ")")
		if column.UpdateRule != "" && column.UpdateRule != NO_ACTION {
			buf.WriteString(" ON UPDATE " + column.UpdateRule)
		}
		if column.DeleteRule != "" && column.DeleteRule != NO_ACTION {
			buf.WriteString(" ON DELETE " + column.DeleteRule)
		}
		if column.IsDeferrable && (dialect == DialectSQLite || dialect == DialectPostgres) {
			buf.WriteString(" DEFERRABLE")
			if column.IsInitiallyDeferred {
				buf.WriteString(" INITIALLY DEFERRED")
			}
		}
	}
}

func writeColumnNames(dialect string, buf *bytes.Buffer, columns []string) {
	for i, column := range columns {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(sq.QuoteIdentifier(dialect, column))
	}
}

func writeConstraintDefinition(dialect string, buf *bytes.Buffer, currentSchema string, constraint *Constraint) {
	// Is it a MySQL primary key constraint?
	if constraint.ConstraintType == PRIMARY_KEY && dialect == DialectMySQL {
		buf.WriteString("PRIMARY KEY (")
		writeColumnNames(dialect, buf, constraint.Columns)
		buf.WriteString(")")
		return
	}

	if constraint.ConstraintName != "" {
		buf.WriteString("CONSTRAINT " + sq.QuoteIdentifier(dialect, constraint.ConstraintName) + " ")
	}
	buf.WriteString(constraint.ConstraintType)

	// Is it a check constraint?
	if constraint.ConstraintType == CHECK {
		buf.WriteString(" (" + constraint.CheckExpr + ")")
		return
	}

	// If it is not an EXCLUDE constraint, write the columns.
	if constraint.ConstraintType != EXCLUDE {
		buf.WriteString(" (")
		writeColumnNames(dialect, buf, constraint.Columns)
		buf.WriteString(")")
	}

	// Is it a foreign key constraint?
	if constraint.ConstraintType == FOREIGN_KEY {
		referencesTable := sq.QuoteIdentifier(dialect, constraint.ReferencesTable)
		if constraint.ReferencesSchema != "" && constraint.ReferencesSchema != currentSchema {
			referencesTable = sq.QuoteIdentifier(dialect, constraint.ReferencesSchema) + "." + referencesTable
		} else if dialect == DialectMySQL && constraint.TableSchema != constraint.ReferencesSchema {
			// If dialect is mysql and the foreign key reference crosses schema
			// boundaries, we have to always qualify it with a schema (even if
			// the schema is the current schema). If not MySQL may do the wrong
			// thing and assign the wrong schema to an unqualified table name.
			refschema := constraint.ReferencesSchema
			if refschema == "" {
				refschema = currentSchema
			}
			referencesTable = sq.QuoteIdentifier(dialect, refschema) + "." + referencesTable
		}
		buf.WriteString(" REFERENCES " + referencesTable + " (")
		writeColumnNames(dialect, buf, constraint.ReferencesColumns)
		buf.WriteString(")")
		if constraint.UpdateRule != "" && constraint.UpdateRule != NO_ACTION {
			buf.WriteString(" ON UPDATE " + constraint.UpdateRule)
		}
		if constraint.DeleteRule != "" && constraint.DeleteRule != NO_ACTION {
			buf.WriteString(" ON DELETE " + constraint.DeleteRule)
		}
	}

	// Is it an exclude constraint?
	if constraint.ConstraintType == EXCLUDE && dialect == DialectPostgres {
		if constraint.ExclusionIndexType != "" {
			buf.WriteString(" USING " + constraint.ExclusionIndexType)
		}
		buf.WriteString(" (")
		for i, column := range constraint.Columns {
			if i > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(column + " WITH ")
			if i < len(constraint.ExclusionOperators) {
				buf.WriteString(constraint.ExclusionOperators[i])
			}
		}
		buf.WriteString(")")
		if constraint.ExclusionPredicate != "" {
			buf.WriteString(" WHERE (" + constraint.ExclusionPredicate + ")")
		}
	}

	// Is it a deferrable constraint?
	if constraint.IsDeferrable && (dialect == DialectSQLite || dialect == DialectPostgres) {
		buf.WriteString(" DEFERRABLE")
		if constraint.IsInitiallyDeferred {
			buf.WriteString(" INITIALLY DEFERRED")
		}
	}
}

type tableQuery struct {
	table *Table
	query string
}
