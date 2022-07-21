package ddl

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"embed"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/bokwoon95/sq"
	"golang.org/x/sync/errgroup"
)

// LoadCmd implements the `sqddl load` subcommand.
type LoadCmd struct {
	// (Required) DB is the database to apply migrations to.
	DB *sql.DB

	// (Required) Dialect is the database dialect.
	Dialect string

	// (Required) DirFS is where the Filenames will be sourced from.
	DirFS fs.FS

	// Filenames specifies the list of files (sql, csv, subdirectory, zip or
	// tgz) to be loaded from the DirFS.
	Filenames []string

	// Stderr specifies the command's standard error. If nil, the command
	// writes to os.Stderr.
	Stderr io.Writer

	// HistoryTable is the name of the migration history table. If empty, the
	// default history table name will be "sqddl_history".
	HistoryTable string

	// Batchsize controls the batch size of a single INSERT
	// statement. If 0, a default batch size of 1000 is used.
	Batchsize int

	// Nullstring specifies the string that is used in CSV to
	// represent NULL. If empty, `\N` is used.
	Nullstring string

	// Binaryprefix specifies the string prefix that is used in CSV to denote a
	// hexadecimal binary literal. If empty, `0x` is used.
	Binaryprefix string

	// NoNullstring specifies that the Nullstring should not be used
	// when reading CSV files.
	NoNullstring bool

	// NoBinaryprefix specifies that the Binaryprefix should not be
	// used when reading CSV files.
	NoBinaryprefix bool

	// Log start and end timestamps for each file loaded.
	Verbose bool

	// (SQLite only) Load timestamp strings as unix timestamp integers for
	// TIMESTAMP, DATETIME and DATE columns.
	TimestampAsInteger bool

	// Ctx is the command's context.
	Ctx context.Context

	db string // -db flag.
}

// LoadCommand creates a new LoadCmd with the given arguments. E.g.
//   sqddl load -db <DATABASE_URL> [FLAGS] [FILENAMES...]
//
//   LoadCommand(
//       "-db", "postgres://user:pass@localhost:5432/sakila",
//       "./db/schema.sql",
//       "./db/actor.csv",
//       "./db/language.csv",
//       "./db/indexes.sql",
//       "./db/constraints.sql",
//   )
//   LoadCommand("-db", "postgres://user:pass@localhost:5432/sakila", "./db")
//   LoadCommand("-db", "postgres://user:pass@localhost:5432/sakila", "./db/sakila.zip")
//   LoadCommand("-db", "postgres://user:pass@localhost:5432/sakila", "./db/sakila.tgz")
func LoadCommand(args ...string) (*LoadCmd, error) {
	var cmd LoadCmd
	var dir string
	flagset := flag.NewFlagSet("", flag.ContinueOnError)
	flagset.StringVar(&cmd.db, "db", "", "(required) Database URL/DSN.")
	flagset.StringVar(&dir, "dir", "", "Working directory. Leave blank to use the current working directory.")
	flagset.StringVar(&cmd.HistoryTable, "history-table", "sqddl_history", "Name of migration history table.")
	flagset.StringVar(&cmd.Nullstring, "nullstring", "\\N", "The string used to represent NULL in CSV files.")
	flagset.StringVar(&cmd.Binaryprefix, "binaryprefix", "0x", "The string used to prefix raw bytes (in hexadecimal form) in CSV files.")
	flagset.IntVar(&cmd.Batchsize, "batchsize", 1000, "How many rows to insert per INSERT statement when loading CSV files.")
	flagset.BoolVar(&cmd.NoNullstring, "no-nullstring", false, "Do not recognize any nullstring when loading CSV files (you will be unable to insert NULL values).")
	flagset.BoolVar(&cmd.NoBinaryprefix, "no-binaryprefix", false, "Do not recognize any binaryprefix when loading CSV files (you will be unable to insert BLOB values).")
	flagset.BoolVar(&cmd.Verbose, "verbose", false, "Log start and end timestamps for each file loaded.")
	flagset.BoolVar(&cmd.TimestampAsInteger, "timestamp-as-integer", false, "(SQLite only) Load timestamp strings as unix timestamp integers for TIMESTAMP, DATETIME and DATE columns.")
	flagset.Usage = func() {
		fmt.Fprint(flagset.Output(), `Usage:
  sqddl load -db <DATABASE_URL> [FLAGS] [FILENAMES...]
  sqddl load -db 'postgres://username:password@localhost:5432/sakila' ./db
  sqddl load -db 'postgres://username:password@localhost:5432/sakila' ./db.zip
  sqddl load -db 'postgres://username:password@localhost:5432/sakila' ./db.tgz
  sqddl load -db 'postgres://username:password@localhost:5432/sakila' ./db/schema.sql ./db/*.csv ./db/indexes.sql ./db/constraints.sql
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
	cmd.DirFS = dirFS(dir)
	cmd.Filenames = flagset.Args()
	for i, filename := range cmd.Filenames {
		cmd.Filenames[i] = normalizeFilename(filename, dir)
	}
	return &cmd, nil
}

// Run runs the LoadCmd.
func (cmd *LoadCmd) Run() error {
	if cmd.DB == nil {
		return fmt.Errorf("nil DB")
	}
	if cmd.Dialect == "" {
		return fmt.Errorf("empty Dialect")
	}
	if cmd.DirFS == nil {
		cmd.DirFS = dirFS("")
	}
	if cmd.Stderr == nil {
		cmd.Stderr = os.Stderr
	}
	if cmd.Batchsize == 0 {
		cmd.Batchsize = 1000
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

	// If user provided no filenames, nothing to do but return.
	if len(cmd.Filenames) == 0 {
		return nil
	}

	// Grab a single database connectin from the pool. Each SQL and CSV file
	// should be run in a single session.
	var err error
	conn, err := cmd.DB.Conn(cmd.Ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	restoreSessionValue := func() error { return nil }
	if cmd.Dialect == sq.DialectSQLite {
		restoreSessionValue, err = setSessionValue(cmd.Ctx, conn, "PRAGMA foreign_keys", "PRAGMA foreign_keys = %s", "0")
		if err != nil {
			return err
		}
		defer restoreSessionValue()
	}

	// The main loop.
	for _, filename := range cmd.Filenames {
		// is it an SQL file?
		if strings.HasSuffix(filename, ".sql") {
			contents, err := cmd.DirFS.Open(filename)
			if err != nil {
				return err
			}
			err = cmd.loadSQL(conn, filename, contents)
			if err != nil {
				return fmt.Errorf("%s: %w", filename, err)
			}
			continue
		}
		// is it a CSV file?
		if strings.HasSuffix(filename, ".csv") {
			contents, err := cmd.DirFS.Open(filename)
			if err != nil {
				return err
			}
			tx, err := conn.BeginTx(cmd.Ctx, nil)
			if err != nil {
				return err
			}
			err = cmd.loadCSV(cmd.Ctx, tx, filename, contents)
			if err != nil {
				return fmt.Errorf("%s: %w", filename, err)
			}
			continue
		}
		// is it a zip file?
		if strings.HasSuffix(filename, ".zip") {
			err = cmd.loadZip(conn, filename)
			if err != nil {
				return err
			}
			continue
		}
		// is it a tar gzip file?
		if strings.HasSuffix(filename, ".tgz") || strings.HasSuffix(filename, ".tar.gz") {
			err = cmd.loadTgz(conn, filename)
			if err != nil {
				return err
			}
			continue
		}
		// is it a directory?
		file, err := cmd.DirFS.Open(filename)
		if err != nil {
			return err
		}
		fileinfo, err := file.Stat()
		if err != nil {
			return err
		}
		file.Close()
		if fileinfo.IsDir() {
			err = cmd.loadDir(conn, filename)
			if err != nil {
				return err
			}
		}
	}

	err = restoreSessionValue()
	if err != nil {
		return err
	}
	return nil
}

// loadSQL loads an SQL file (an io.Reader) into the database.
//
// The io.Reader will automatically be closed if it implements io.ReadCloser.
func (cmd *LoadCmd) loadSQL(conn *sql.Conn, filename string, r io.Reader) error {
	if readCloser, ok := r.(io.ReadCloser); ok {
		defer readCloser.Close()
	}
	buf := bufpool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufpool.Put(buf)
	_, err := buf.ReadFrom(r)
	if err != nil {
		return err
	}
	if buf.Len() == 0 {
		return nil
	}
	if cmd.Verbose {
		fmt.Fprintln(cmd.Stderr, timestamp()+"[START] "+filename)
	}
	restoreSessionValue := func() error { return nil }
	if cmd.Dialect == sq.DialectMySQL && filepath.Base(filename) == "constraints.sql" {
		restoreSessionValue, err = setSessionValue(cmd.Ctx, conn, "SELECT @@foreign_key_checks", "SET foreign_key_checks = %s", "0")
		if err != nil {
			return err
		}
		defer restoreSessionValue()
	}
	query := buf.String()
	startedAt := time.Now()
	_, err = conn.ExecContext(cmd.Ctx, query)
	timeTaken := time.Since(startedAt)
	if err != nil {
		if cmd.Verbose {
			fmt.Fprintln(cmd.Stderr, timestamp()+"[FAIL]  "+filename+" ("+timeTaken.String()+")")
		} else {
			fmt.Fprintln(cmd.Stderr, "[FAIL] "+filename+" ("+timeTaken.String()+")")
		}
		return err
	}
	if cmd.Verbose {
		fmt.Fprintln(cmd.Stderr, timestamp()+"[OK]    "+filename+" ("+timeTaken.String()+")")
	} else {
		fmt.Fprintln(cmd.Stderr, "[OK] "+filename+" ("+timeTaken.String()+")")
	}
	err = restoreSessionValue()
	if err != nil {
		return err
	}
	return nil
}

// startsWithNumber is used to trim leading digits from a csv filename (in
// order to determine the name of the table).
var startsWithNumber = regexp.MustCompile(`^\d*_`)

var sqliteTimestampFormats = []string{
	"2006-01-02 15:04:05.999999999-07:00",
	"2006-01-02T15:04:05.999999999-07:00",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02T15:04:05.999999999",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04",
	"2006-01-02T15:04",
	"2006-01-02",
}

// loadCSV loads a CSV file (an io.Reader) into the database. The table name is
// determined by the filename, while the column names are determined by the
// first line of the CSV file (the headers).
//
// The io.Reader will automatically be closed if it implements io.ReadCloser.
func (cmd *LoadCmd) loadCSV(ctx context.Context, tx *sql.Tx, filename string, r io.Reader) error {
	defer tx.Rollback()
	if readCloser, ok := r.(io.ReadCloser); ok {
		defer readCloser.Close()
	}
	bi := BatchInsert{
		Dialect:   cmd.Dialect,
		Batchsize: cmd.Batchsize,
	}

	// Determine the table name.
	var err error
	bi.TableName = filepath.Base(strings.TrimSuffix(filename, ".csv"))
	if startsWithNumber.MatchString(bi.TableName) {
		_, bi.TableName, _ = strings.Cut(bi.TableName, "_")
	}
	if i := strings.Index(bi.TableName, "."); i >= 0 {
		bi.TableSchema, bi.TableName = bi.TableName[:i], bi.TableName[i+1:]
	}
	if bi.TableSchema == "" {
		bi.TableSchema, err = NewDatabaseIntrospector(cmd.Dialect, tx).GetCurrentSchema()
		if err != nil {
			return err
		}
	}

	// Init csvReader.
	csvReader := csv.NewReader(r)
	csvReader.FieldsPerRecord = -1
	csvReader.LazyQuotes = true
	csvReader.ReuseRecord = true

	// Read the first line (the header).
	bi.Columns, err = csvReader.Read()
	if err == io.EOF {
		return fmt.Errorf("filename %s is empty", filename)
	}
	if err != nil {
		return err
	}

	var columnTypes []string
	columnTypes, bi.KeyColumns, bi.IdentityColumns, err = getColumnInfo(ctx, tx, bi.Dialect, bi.TableSchema, bi.TableName, bi.Columns)
	if err != nil {
		return err
	}

	restoreSessionValue := func() error { return nil }
	switch cmd.Dialect {
	case sq.DialectPostgres:
		// If inserting a timezone-less timestamp like '2006-01-02 15:04:05'
		// into a TIMESTAMPTZ column, Postgres will attach the session timezone
		// to the timestamp. That is not what we want. SQLite doesn't do that,
		// MySQL doesn't do that, SQLServer doesn't do that. They treat it like
		// a timestamp in UTC timezone. Make Postgres behave like the other
		// database by temporarily setting the session timezone to UTC.
		restoreSessionValue, err = setSessionValue(ctx, tx, "SHOW timezone", "SET timezone = '%s'", "UTC")
	case sq.DialectMySQL:
		// MySQL inserts are too damn slow if foreign keys are enabled, disable
		// them first before inserting. Assume the data is always valid.
		restoreSessionValue, err = setSessionValue(ctx, tx, "SELECT @@foreign_key_checks", "SET foreign_key_checks = %s", "0")
	}
	if err != nil {
		return err
	}
	defer restoreSessionValue()

	// Run the batch insert.
	start := time.Now()
	if cmd.Verbose {
		fmt.Fprintln(cmd.Stderr, timestamp()+"[START] "+filename)
	}
	var recordNo int
	rowsAffected, err := bi.ExecContext(ctx, tx, func(rowvalue []any) error {
		record, err := csvReader.Read()
		if err == io.EOF {
			return io.EOF
		}
		recordNo++
		if err != nil {
			return fmt.Errorf("record %d %#v: %w", recordNo, record, err)
		}
		if len(record) != len(bi.Columns) {
			return fmt.Errorf("record %d %#v: expected %d fields but got %d", recordNo, record, len(bi.Columns), len(record))
		}
		for i, field := range record {
			columnType := columnTypes[i]
			if !cmd.NoNullstring && field == cmd.Nullstring {
				rowvalue[i] = nil
				continue
			}
			binaryCompatible := false
			switch columnType {
			case "BYTEA", "BINARY", "VARBINARY", "TINYBLOB", "BLOB", "MEDIUMBLOB", "LONGBLOB", "VARBIT", "UUID":
				binaryCompatible = true
			}
			if binaryCompatible && !cmd.NoBinaryprefix && cmd.Binaryprefix != "" && strings.HasPrefix(field, cmd.Binaryprefix) {
				b, err := hex.DecodeString(strings.TrimPrefix(field, cmd.Binaryprefix))
				if err != nil {
					return fmt.Errorf("record %d %#v: %s is not a valid binary literal", recordNo, record, field)
				}
				if cmd.Dialect == sq.DialectPostgres && columnType == "UUID" {
					// (Postgres only) Convert UUID bytes into UUID string if
					// the column type is UUID.
					if len(b) != 16 {
						return fmt.Errorf("record %d %#v: %s are not valid UUID bytes (got %d bytes, want 16)", recordNo, record, field, len(b))
					}
					buf := make([]byte, 36)
					hex.Encode(buf, b[:4])
					buf[8] = '-'
					hex.Encode(buf[9:13], b[4:6])
					buf[13] = '-'
					hex.Encode(buf[14:18], b[6:8])
					buf[18] = '-'
					hex.Encode(buf[19:23], b[8:10])
					buf[23] = '-'
					hex.Encode(buf[24:], b[10:])
					rowvalue[i] = string(buf)
					continue
				}
				rowvalue[i] = b
				continue
			}
			if cmd.Dialect == sq.DialectPostgres && strings.HasSuffix(columnType, "[]") && len(field) > 2 && field[0] == '[' && field[len(field)-1] == ']' {
				// (Postgres only) Convert JSON arrays into Postgres arrays if
				// the column is an array type (e.g. TEXT[], INT[]).
				rowvalue[i] = "{" + field[1:len(field)-1] + "}"
				continue
			}
			if cmd.Dialect == sq.DialectSQLite && cmd.TimestampAsInteger && (columnType == "TIMESTAMP" || columnType == "DATETIME" || columnType == "DATE") {
				s := strings.TrimSuffix(field, "Z")
				var timeVal sql.NullTime
				for _, format := range sqliteTimestampFormats {
					if t, err := time.ParseInLocation(format, s, time.UTC); err == nil {
						timeVal.Time, timeVal.Valid = t, true
						break
					}
				}
				if timeVal.Valid {
					rowvalue[i] = timeVal.Time.UTC().Unix()
					continue
				}
			}
			rowvalue[i] = field
		}
		return nil
	})
	timeTaken := time.Since(start)
	rowsAffectedMsg := "(1 row affected)"
	if rowsAffected != 1 {
		rowsAffectedMsg = "(" + strconv.FormatInt(rowsAffected, 10) + " rows affected)"
	}
	if err != nil {
		batchNo := int(math.Ceil(float64(recordNo) / float64(cmd.Batchsize)))
		if batchNo == 0 {
			batchNo = 1
		}
		offset := (batchNo - 1) * cmd.Batchsize
		start, end := offset+1, offset+cmd.Batchsize // excludes header, add 1 to get true line number in file
		if recordNo < end {
			end = recordNo
		}
		errStatus := "[FAIL] "
		if errors.Is(err, context.Canceled) {
			errStatus = "[CANCELLED] "
		} else if errors.Is(err, context.DeadlineExceeded) {
			errStatus = "[TIMEOUT] "
		}
		if cmd.Verbose {
			fmt.Fprintln(cmd.Stderr, timestamp()+errStatus+filename+" ("+timeTaken.String()+") "+rowsAffectedMsg)
		} else {
			fmt.Fprintln(cmd.Stderr, errStatus+filename+" ("+timeTaken.String()+") "+rowsAffectedMsg)
		}
		if start == end {
			return fmt.Errorf("record %d: %w", start, err)
		}
		return fmt.Errorf("record %d to %d: %w", start, end, err)
	}
	if cmd.Verbose {
		fmt.Fprintln(cmd.Stderr, timestamp()+"[OK]    "+filename+" ("+timeTaken.String()+") "+rowsAffectedMsg)
	} else {
		fmt.Fprintln(cmd.Stderr, "[OK] "+filename+" ("+timeTaken.String()+") "+rowsAffectedMsg)
	}

	err = restoreSessionValue()
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

// loadZip loads a .zip file into the database. The order of files loaded goes
// like this: schema.sql -> CSV files -> indexes.sql -> constraints.sql.
func (cmd *LoadCmd) loadZip(conn *sql.Conn, zipName string) error {
	// Sanity check in case someone tries to load a zip file from an embed.FS.
	// This is potentially a common mistake, so we specially flag it out with a
	// custom error message.
	if _, ok := cmd.DirFS.(embed.FS); ok {
		return fmt.Errorf("cannot read .zip files embedded with embed.FS, please use .tgz instead")
	}

	// Open the zip file.
	file, err := cmd.DirFS.Open(zipName)
	if err != nil {
		return err
	}
	defer file.Close()
	fileinfo, err := file.Stat()
	if err != nil {
		return err
	}
	readerAt, ok := file.(io.ReaderAt)
	if !ok {
		return fmt.Errorf("cannot read %s because it is not of type io.ReaderAt", zipName)
	}
	zipReader, err := zip.NewReader(readerAt, fileinfo.Size())
	if err != nil {
		return err
	}

	// We only want to read from top-level SQL and CSV files, but sometimes the
	// files are located behind a extra top level directory of the same name as
	// the zip file e.g.
	//
	// dump.zip
	// └─ dump
	//    ├─ schema.sql
	//    ├─ actor.csv
	//    ├─ dump
	//    └─ repeatable
	//
	// rather than
	//
	// dump.zip
	// ├─ schema.sql
	// ├─ actor.csv
	// ├─ dump
	// └─ repeatable
	//
	// As a result we want to accept both dump.zip/schema.sql and
	// dump.zip/dump/schema.sql as top level files.
	name := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(zipName), "./"), ".zip")

	// Locate schema.sql, indexes.sql and constraints.sql.
	var schemaFilename, indexesFilename, constraintsFilename string
	var schemaFile, indexesFile, constraintsFile *zip.File
	for _, f := range zipReader.File {
		if strings.HasPrefix(f.Name, "__MACOSX/") {
			// https://superuser.com/questions/104500/what-is-macosx-folder
			continue
		}
		switch f.Name {
		case "schema.sql", name + "/schema.sql":
			schemaFilename = f.Name
			schemaFile = f
		case "indexes.sql", name + "/indexes.sql":
			indexesFilename = f.Name
			indexesFile = f
		case "constraints.sql", name + "/constraints.sql":
			constraintsFilename = f.Name
			constraintsFile = f
		}
		if schemaFile != nil && indexesFile != nil && constraintsFile != nil {
			break
		}
	}

	// Run schema.sql if it exists.
	if schemaFile != nil {
		contents, err := schemaFile.Open()
		if err != nil {
			return err
		}
		filename := filepath.ToSlash(filepath.Join(zipName, schemaFilename))
		err = cmd.loadSQL(conn, filename, contents)
		if err != nil {
			return fmt.Errorf("%s: %w", filename, err)
		}
	}

	// Load CSV files in parallel unless dialect is SQLite (SQLite only allows
	// one writer at a time).
	var g *errgroup.Group
	var ctx context.Context
	if cmd.Dialect != sq.DialectSQLite {
		g, ctx = errgroup.WithContext(cmd.Ctx)
	}
	for _, f := range zipReader.File {
		f := f
		if strings.HasPrefix(f.Name, "__MACOSX/") {
			// https://superuser.com/questions/104500/what-is-macosx-folder
			continue
		}
		if !strings.HasSuffix(f.Name, ".csv") {
			continue
		}
		dir := filepath.Dir(f.Name)
		if dir != "." && dir != name {
			continue
		}
		if g != nil {
			g.Go(func() error {
				contents, err := f.Open()
				if err != nil {
					return err
				}
				tx, err := conn.BeginTx(ctx, nil)
				if err != nil {
					return err
				}
				filename := filepath.ToSlash(filepath.Join(zipName, f.Name))
				err = cmd.loadCSV(ctx, tx, filename, contents)
				if err != nil {
					return fmt.Errorf("%s: %w", filename, err)
				}
				return nil
			})
		} else {
			contents, err := f.Open()
			if err != nil {
				return err
			}
			tx, err := conn.BeginTx(cmd.Ctx, nil)
			if err != nil {
				return err
			}
			filename := filepath.ToSlash(filepath.Join(zipName, f.Name))
			err = cmd.loadCSV(cmd.Ctx, tx, filename, contents)
			if err != nil {
				return fmt.Errorf("%s: %w", filename, err)
			}
		}
	}
	if g != nil {
		err = g.Wait()
		if err != nil {
			return err
		}
	}

	// Run indexes.sql if it exists.
	if indexesFile != nil {
		contents, err := indexesFile.Open()
		if err != nil {
			return err
		}
		filename := filepath.ToSlash(filepath.Join(zipName, indexesFilename))
		err = cmd.loadSQL(conn, filename, contents)
		if err != nil {
			return fmt.Errorf("%s: %w", filename, err)
		}
	}

	// Run constraints.sql if it exists.
	if constraintsFile != nil {
		contents, err := constraintsFile.Open()
		if err != nil {
			return err
		}
		filename := filepath.ToSlash(filepath.Join(zipName, constraintsFilename))
		err = cmd.loadSQL(conn, filename, contents)
		if err != nil {
			return fmt.Errorf("%s: %w", filename, err)
		}
	}
	return nil
}

// loadTgz loads a .tgz or .tar.gz file into the database. The order of files
// loaded goes like this: schema.sql -> CSV files -> indexes.sql ->
// constraints.sql.
func (cmd *LoadCmd) loadTgz(conn *sql.Conn, tgzName string) error {
	// schemaBuf
	var schemaFilename string
	schemaBuf := bufpool.Get().(*bytes.Buffer)
	schemaBuf.Reset()
	defer bufpool.Put(schemaBuf)
	// indexesBuf
	var indexesFilename string
	indexesBuf := bufpool.Get().(*bytes.Buffer)
	indexesBuf.Reset()
	defer bufpool.Put(indexesBuf)
	// constraintsBuf
	var constraintsFilename string
	constraintsBuf := bufpool.Get().(*bytes.Buffer)
	constraintsBuf.Reset()
	defer bufpool.Put(constraintsBuf)

	// Open the tgz file.
	file, err := cmd.DirFS.Open(tgzName)
	if err != nil {
		return err
	}
	defer file.Close()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	tarReader := tar.NewReader(gzipReader)

	// We only want to read from top-level SQL and CSV files, but sometimes the
	// files are located behind a extra top level directory of the same name as
	// the tgz file e.g.
	//
	// dump.tgz
	// └─ dump
	//    ├─ schema.sql
	//    ├─ actor.csv
	//    ├─ dump
	//    └─ repeatable
	//
	// rather than
	//
	// dump.tgz
	// ├─ schema.sql
	// ├─ actor.csv
	// ├─ dump
	// └─ repeatable
	//
	// As a result we want to accept both dump.tgz/schema.sql and
	// dump.tgz/dump/schema.sql as top level files.
	var name string
	if strings.HasSuffix(tgzName, ".tar.gz") {
		name = strings.TrimSuffix(strings.TrimPrefix(filepath.Base(tgzName), "./"), ".tar.gz")
	} else {
		name = strings.TrimSuffix(strings.TrimPrefix(filepath.Base(tgzName), "./"), ".tgz")
	}

	// Open the tgz file. This first pass is solely dedicated to locating and
	// storing the contents of schema.sql, indexes.sql and constraints.sql into
	// memory. This is to get around the limitation that tgz files can only be
	// streamed (random access is not supported).
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		switch hdr.Name {
		case "schema.sql", name + "/schema.sql":
			schemaFilename = hdr.Name
			_, err = schemaBuf.ReadFrom(tarReader)
			if err != nil {
				return err
			}
		case "indexes.sql", name + "/indexes.sql":
			indexesFilename = hdr.Name
			_, err = indexesBuf.ReadFrom(tarReader)
			if err != nil {
				return err
			}
		case "constraints.sql", name + "/constraints.sql":
			constraintsFilename = hdr.Name
			_, err = constraintsBuf.ReadFrom(tarReader)
			if err != nil {
				return err
			}
		}
		if schemaBuf.Len() > 0 && indexesBuf.Len() > 0 && constraintsBuf.Len() > 0 {
			break
		}
	}
	err = file.Close()
	if err != nil {
		return err
	}

	// Run schema.sql if it exists.
	if schemaBuf.Len() > 0 {
		filename := filepath.ToSlash(filepath.Join(tgzName, schemaFilename))
		err = cmd.loadSQL(conn, filename, schemaBuf)
		if err != nil {
			return fmt.Errorf("%s: %w", filename, err)
		}
	}

	// Open the tgz file a second time, this time for reading CSV files.
	file, err = cmd.DirFS.Open(tgzName)
	if err != nil {
		return err
	}
	defer file.Close()
	gzipReader, err = gzip.NewReader(file)
	if err != nil {
		return err
	}
	tarReader = tar.NewReader(gzipReader)

	// Load CSV files.
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if !strings.HasSuffix(hdr.Name, ".csv") {
			continue
		}
		dir := filepath.Dir(hdr.Name)
		if dir != "." && dir != name {
			continue
		}
		tx, err := conn.BeginTx(cmd.Ctx, nil)
		if err != nil {
			return err
		}
		filename := filepath.ToSlash(filepath.Join(tgzName, hdr.Name))
		err = cmd.loadCSV(cmd.Ctx, tx, filename, tarReader)
		if err != nil {
			return fmt.Errorf("%s: %w", filename, err)
		}
	}

	// Run indexes.sql if it exists.
	if indexesBuf.Len() > 0 {
		filename := filepath.ToSlash(filepath.Join(tgzName, indexesFilename))
		err = cmd.loadSQL(conn, filename, indexesBuf)
		if err != nil {
			return fmt.Errorf("%s: %w", filename, err)
		}
	}

	// Run constraints.sql if it exists.
	if constraintsBuf.Len() > 0 {
		filename := filepath.ToSlash(filepath.Join(tgzName, constraintsFilename))
		err = cmd.loadSQL(conn, filename, constraintsBuf)
		if err != nil {
			return fmt.Errorf("%s: %w", filename, err)
		}
	}
	return nil
}

// loadDir loads a directory into the database. The order of files loaded goes
// like this: schema.sql -> csv files -> indexes.sql -> constraints.sql.
func (cmd *LoadCmd) loadDir(conn *sql.Conn, dirname string) error {
	// Run schema.sql if it exists.
	schemaFilename := filepath.ToSlash(filepath.Join(dirname, "schema.sql"))
	schemaFile, err := cmd.DirFS.Open(schemaFilename)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err == nil {
		err = cmd.loadSQL(conn, schemaFilename, schemaFile)
		if err != nil {
			return fmt.Errorf("%s: %w", schemaFilename, err)
		}
	}

	// Walk the directory to find all top level CSV files.
	var csvFilenames []string
	err = fs.WalkDir(cmd.DirFS, dirname, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && path != dirname {
			return fs.SkipDir
		}
		if strings.HasSuffix(path, ".csv") {
			csvFilenames = append(csvFilenames, path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Load CSV files in parallel unless dialect is SQLite (SQLite only allows
	// one writer at a time).
	var g *errgroup.Group
	var ctx context.Context
	if cmd.Dialect != sq.DialectSQLite {
		g, ctx = errgroup.WithContext(cmd.Ctx)
	}
	for _, filename := range csvFilenames {
		filename := filename
		if g != nil {
			g.Go(func() error {
				contents, err := cmd.DirFS.Open(filename)
				if err != nil {
					return err
				}
				tx, err := cmd.DB.BeginTx(ctx, nil)
				if err != nil {
					return err
				}
				err = cmd.loadCSV(ctx, tx, filename, contents)
				if err != nil {
					return fmt.Errorf("%s: %w", filename, err)
				}
				return nil
			})
		} else {
			contents, err := cmd.DirFS.Open(filename)
			if err != nil {
				return err
			}
			tx, err := conn.BeginTx(cmd.Ctx, nil)
			if err != nil {
				return err
			}
			err = cmd.loadCSV(cmd.Ctx, tx, filename, contents)
			if err != nil {
				return fmt.Errorf("%s: %w", filename, err)
			}
		}
	}
	if g != nil {
		err = g.Wait()
		if err != nil {
			return err
		}
	}

	// Run indexes.sql if it exists.
	indexesFilename := filepath.ToSlash(filepath.Join(dirname, "indexes.sql"))
	indexesFile, err := cmd.DirFS.Open(indexesFilename)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err == nil {
		err = cmd.loadSQL(conn, indexesFilename, indexesFile)
		if err != nil {
			return fmt.Errorf("%s: %w", indexesFilename, err)
		}
	}

	// Run constraints.sql if it exists.
	constraintsFilename := filepath.ToSlash(filepath.Join(dirname, "constraints.sql"))
	constraintsFile, err := cmd.DirFS.Open(constraintsFilename)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err == nil {
		err = cmd.loadSQL(conn, constraintsFilename, constraintsFile)
		if err != nil {
			return fmt.Errorf("%s: %w", constraintsFilename, err)
		}
	}
	return nil
}

func getColumnInfo(ctx context.Context, tx *sql.Tx, dialect, tableSchema, tableName string, columns []string) (columnTypes, keyColumns, identityColumns []string, err error) {
	columnIndex := make(map[string]int)
	for i, column := range columns {
		columnIndex[column] = i
	}
	columnTypes = make([]string, len(columns))
	keyColumns = make([]string, 0, len(columns))
	identityColumns = make([]string, 0, len(columns))
	var query string
	switch dialect {
	case sq.DialectSQLite:
		query = "SELECT" +
			" name AS column_name" +
			", type AS column_type" +
			", pk > 0 AS is_key_column" +
			" FROM pragma_table_info($1)" +
			" ORDER BY cid"
	case sq.DialectPostgres:
		query = "SELECT" +
			" columns.attname AS column_name" +
			", format_type(columns.atttypid, columns.atttypmod) AS column_type" +
			", COALESCE(columns.attnum = ANY(pg_constraint.conkey), FALSE) AS is_key_column" +
			", columns.attidentity IN ('d', 'a') AS is_identity_column" +
			" FROM pg_attribute AS columns" +
			" JOIN pg_class AS tables ON tables.relkind = 'r' AND tables.oid = columns.attrelid" +
			" JOIN pg_namespace AS schemas ON schemas.oid = tables.relnamespace" +
			" LEFT JOIN pg_constraint ON pg_constraint.contype = 'p' AND pg_constraint.conrelid = tables.oid" +
			" WHERE columns.attnum > 0 AND schemas.nspname = $1 AND tables.relname = $2" +
			" ORDER BY columns.attnum"
	case sq.DialectMySQL:
		query = "SELECT" +
			" columns.column_name" +
			", columns.column_type" +
			", key_column_usage.column_name IS NOT NULL AS is_key_column" +
			" FROM information_schema.columns" +
			" JOIN information_schema.table_constraints USING (table_schema, table_name)" +
			" LEFT JOIN information_schema.key_column_usage USING (constraint_schema, constraint_name, table_name, column_name)" +
			" WHERE table_constraints.constraint_type = 'PRIMARY KEY' AND columns.table_schema = ? AND columns.table_name = ?" +
			" ORDER BY columns.ordinal_position"
	case sq.DialectSQLServer:
		query = "SELECT" +
			" columns.name AS column_name" +
			", COALESCE(TYPE_NAME(columns.system_type_id), '') AS column_type" +
			", CASE WHEN index_columns.column_id IS NOT NULL THEN 1 ELSE 0 END AS is_key_column" +
			", columns.is_identity AS is_identity_column" +
			" FROM sys.columns" +
			" JOIN sys.tables ON tables.object_id = columns.object_id" +
			" JOIN sys.schemas ON schemas.schema_id = tables.schema_id" +
			" LEFT JOIN sys.indexes ON indexes.object_id = tables.object_id" +
			" LEFT JOIN sys.index_columns ON index_columns.index_id = indexes.index_id AND index_columns.object_id = indexes.object_id AND index_columns.column_id = columns.column_id" +
			" WHERE COALESCE(indexes.is_primary_key, 0) = 1 AND schemas.name = @p1 AND tables.name = @p2" +
			" ORDER BY COLUMNPROPERTY(columns.object_id, columns.name, 'ordinal')"
	default:
		return columnTypes, nil, nil, nil
	}
	var rows *sql.Rows
	if dialect == sq.DialectSQLite {
		rows, err = tx.QueryContext(ctx, query, tableName)
	} else {
		rows, err = tx.QueryContext(ctx, query, tableSchema, tableName)
	}
	if err != nil {
		return nil, nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var columnName, columnType string
		var isKeyColumn, isIdentityColumn bool
		if dialect == sq.DialectPostgres || dialect == sq.DialectSQLServer {
			err = rows.Scan(&columnName, &columnType, &isKeyColumn, &isIdentityColumn)
		} else {
			err = rows.Scan(&columnName, &columnType, &isKeyColumn)
		}
		if err != nil {
			return nil, nil, nil, err
		}
		if i, ok := columnIndex[columnName]; ok {
			columnTypes[i], _, _ = normalizeColumnType(dialect, columnType)
		}
		if isKeyColumn {
			keyColumns = append(keyColumns, columnName)
		}
		if isIdentityColumn {
			identityColumns = append(identityColumns, columnName)
		}
	}
	return columnTypes, keyColumns, identityColumns, closeRows(rows)
}

func setSessionValue(ctx context.Context, db sq.DB, getter, setter, value string) (restoreSessionValue func() error, err error) {
	rows, err := db.QueryContext(ctx, getter)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var oldValue string
	for rows.Next() {
		err = rows.Scan(&oldValue)
		if err != nil {
			return nil, err
		}
		break
	}
	if err = closeRows(rows); err != nil {
		return nil, err
	}
	if oldValue == value {
		return func() error { return nil }, nil
	}
	_, err = db.ExecContext(ctx, strings.ReplaceAll(setter, "%s", sq.EscapeQuote(value, '\'')))
	if err != nil {
		return nil, err
	}
	var done int32
	return func() error {
		if !atomic.CompareAndSwapInt32(&done, 0, 1) {
			return nil
		}
		_, err = db.ExecContext(ctx, strings.ReplaceAll(setter, "%s", sq.EscapeQuote(oldValue, '\'')))
		return err
	}, nil
}
