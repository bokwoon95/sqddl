package ddl

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// GenerateCmd implements the `sqddl generate` subcommand.
type GenerateCmd struct {
	// SrcCatalog is the source catalog that you want to migrate from.
	SrcCatalog *Catalog

	// DestCatalog is the destination catalog that you want to migrate to.
	DestCatalog *Catalog

	// DirFS is where the Filenames will be sourced from.
	DirFS fs.FS

	// Filenames specifies the list of files (loaded from the Dir) used to
	// build the DestCatalog. It will be ignored if the DestCatalog is already
	// non-nil.
	Filenames []string

	// OutputDir is where the migration scripts will be created.
	// Leave blank to use the current working directory.
	OutputDir string

	// Stderr specifies the command's standard error. If nil, the command
	// writes to os.Stderr.
	Stderr io.Writer

	// HistoryTable is the name of the migration history table. If empty, the
	// default history table name will be "sqddl_history".
	HistoryTable string

	// Prefix is filename prefix for the migration(s). If empty, the current
	// timestamp is used.
	Prefix string

	// DropObjects controls whether statements like DROP TABLE, DROP COLUMN
	// will be generated.
	DropObjects bool

	// Dialect is the sql dialect used. This will override whatever dialect is
	// set inside the SrcCatalog and DestCatalog.
	Dialect string

	// AcceptWarnings will accept warnings when generating migrations.
	AcceptWarnings bool
}

// GenerateCommand creates a new GenerateCmd with the given arguments. E.g.
//
//   sqddl generate -src <SRC_SCHEMA> -dest <DEST_SCHEMA> [FLAGS]
//
//   GenerateCommand(
//     "-src", "postgres://user:pass@localhost:5432/mydatabase",
//     "-dest", "tables/tables.go",
//     "-output-dir", "./migrations",
//   )
func GenerateCommand(args ...string) (*GenerateCmd, error) {
	var cmd GenerateCmd
	var db, src, dest string
	flagset := flag.NewFlagSet("", flag.ContinueOnError)
	flagset.StringVar(&db, "db", "", "Database URL/DSN. Used as the source schema if -src is not provided.")
	flagset.StringVar(&src, "src", "", "Comma-separated list of source schemas.")
	flagset.StringVar(&dest, "dest", "", "Comma-separated list of destination schemas.")
	flagset.StringVar(&cmd.OutputDir, "output-dir", "", "Output directory. Leave blank to use the current working directory.")
	flagset.StringVar(&cmd.HistoryTable, "history-table", "sqddl_history", "Name of migration history table.")
	flagset.StringVar(&cmd.Prefix, "prefix", "", "Migration filename prefix. Leave blank to use the current timestamp.")
	flagset.BoolVar(&cmd.DropObjects, "drop-objects", false, "Whether statements like DROP TABLE, DROP COLUMN, etc should be generated.")
	flagset.StringVar(&cmd.Dialect, "dialect", "", "The database dialect used. Not needed if the database dialect can be inferred from the source schema's database URL.")
	flagset.BoolVar(&cmd.AcceptWarnings, "accept-warnings", false, "Accept warnings when generating migrations.")
	flagset.Usage = func() {
		fmt.Fprint(flagset.Output(), `Usage:
  sqddl generate -src <SRC_SCHEMA> -dest <DEST_SCHEMA> [FLAGS]
  sqddl generate -src 'postgres://username:password@localhost:5432/sakila' -dest tables/tables.go
  sqddl generate -src 'postgres://username:password@localhost:5432/sakila' -dest file1.go,file2.go,file3.go
  sqddl generate -src schema.json -dest tables/tables.go
  sqddl generate -src dump/       -dest tables/tables.go # dump/ must contain schema.json
  sqddl generate -src dump.zip    -dest tables/tables.go # dump.zip must contain schema.json
Flags:
`)
		flagset.PrintDefaults()
	}
	err := flagset.Parse(args)
	if err != nil {
		return nil, err
	}
	cmd.SrcCatalog = &Catalog{}
	cmd.DestCatalog = &Catalog{}
	if cmd.Dialect != "" {
		cmd.SrcCatalog.Dialect = cmd.Dialect
		cmd.DestCatalog.Dialect = cmd.Dialect
	}
	if src != "" {
		for _, s := range strings.Split(src, ",") {
			err = writeCatalog(cmd.SrcCatalog, os.DirFS("."), cmd.HistoryTable, s)
			if err != nil {
				return nil, err
			}
		}
	} else if db != "" {
		err = writeCatalog(cmd.SrcCatalog, os.DirFS("."), cmd.HistoryTable, db)
		if err != nil {
			return nil, err
		}
	}
	if cmd.Dialect == "" && cmd.SrcCatalog.Dialect != "" {
		cmd.Dialect = cmd.SrcCatalog.Dialect
	}
	if dest != "" {
		for _, s := range strings.Split(dest, ",") {
			err = writeCatalog(cmd.DestCatalog, os.DirFS("."), cmd.HistoryTable, s)
			if err != nil {
				return nil, err
			}
		}
	}
	if cmd.Dialect == "" && cmd.DestCatalog.Dialect != "" {
		cmd.Dialect = cmd.DestCatalog.Dialect
	}
	return &cmd, nil
}

// Run runs the GenerateCmd.
func (cmd *GenerateCmd) Run() error {
	files, warnings, err := cmd.Results()
	if err != nil {
		return err
	}
	if len(warnings) > 0 {
		if !cmd.AcceptWarnings {
			return fmt.Errorf("warnings present (to proceed despite the warnings, use the -accept-warnings flag):\n" + strings.Join(warnings, "\n"))
		}
		for _, warning := range warnings {
			fmt.Fprintln(cmd.Stderr, warning)
		}
	}
	defer func() {
		for _, file := range files {
			file.Close()
		}
	}()
	for _, file := range files {
		fileinfo, err := file.Stat()
		if err != nil {
			return err
		}
		name := filepath.Join(cmd.OutputDir, fileinfo.Name())
		outputfile, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		_, err = outputfile.ReadFrom(file)
		if err != nil {
			return err
		}
		err = outputfile.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// Results gets the results of the GenerateCmd.
//
// Each file in the files slice should be closed once read.
func (cmd *GenerateCmd) Results() (files []fs.File, warnings []string, err error) {
	// TODO: dropping a column that is referenced by an existing index or
	// constraint (that hasn't been dropped) is an error.
	// TODO: adding a column with NOT NULL is unsafe. If no DEFAULT is
	// provided, it is straight up an error. If a DEFAULT is provided, it is
	// unsafe (for Postgres <11 and SQL Server non-enterprise edition). Issue a
	// warning telling the user to set it as NULL first, backfilling existing
	// rows, then setting NOT NULL on it.
	// TODO: adding an identity column is an error for sqlserver.
	if cmd.Dialect == "" {
		return nil, nil, fmt.Errorf("empty Dialect")
	}
	prefix := cmd.Prefix
	if prefix == "" {
		prefix = time.Now().UTC().Format("20060102150405")
	}
	if cmd.Stderr == nil {
		cmd.Stderr = os.Stderr
	}
	if cmd.SrcCatalog == nil {
		cmd.SrcCatalog = &Catalog{}
	}
	if cmd.Dialect != "" {
		cmd.SrcCatalog.Dialect = cmd.Dialect
	}
	if cmd.DestCatalog == nil {
		cmd.DestCatalog = &Catalog{Dialect: cmd.SrcCatalog.Dialect}
		for _, filename := range cmd.Filenames {
			err := writeCatalog(cmd.DestCatalog, cmd.DirFS, cmd.HistoryTable, filename)
			if err != nil {
				return nil, nil, err
			}
		}
	}
	var filenames []string
	var bufs []*bytes.Buffer
	switch cmd.Dialect {
	case DialectSQLite:
		m := newSQLiteMigration(cmd.SrcCatalog, cmd.DestCatalog, cmd.DropObjects)
		filenames, bufs, warnings = m.sql(prefix)
	case DialectPostgres:
		m := newPostgresMigration(cmd.SrcCatalog, cmd.DestCatalog, cmd.DropObjects)
		filenames, bufs, warnings = m.sql(prefix)
	case DialectMySQL:
		m := newMySQLMigration(cmd.SrcCatalog, cmd.DestCatalog, cmd.DropObjects)
		filenames, bufs, warnings = m.sql(prefix)
	case DialectSQLServer:
		m := newSQLServerMigration(cmd.SrcCatalog, cmd.DestCatalog, cmd.DropObjects)
		filenames, bufs, warnings = m.sql(prefix)
	default:
		return nil, nil, fmt.Errorf("unsupported dialect %q", cmd.SrcCatalog.Dialect)
	}
	files = make([]fs.File, len(filenames))
	for i, filename := range filenames {
		buf := bufs[i]
		files[i] = &bufferFile{
			name: filename,
			size: int64(buf.Len()),
			buf:  buf,
		}
	}
	return files, warnings, err
}

type bufferFile struct {
	name string
	size int64
	buf  *bytes.Buffer
}

var _ fs.File = (*bufferFile)(nil)

func (bf *bufferFile) Name() string { return bf.name }

func (bf *bufferFile) Size() int64 { return bf.size }

func (bf *bufferFile) Mode() fs.FileMode { return 0644 }

func (bf *bufferFile) ModTime() time.Time { return time.Time{} }

func (bf *bufferFile) IsDir() bool { return false }

func (bf *bufferFile) Sys() any { return nil }

func (bf *bufferFile) Stat() (fs.FileInfo, error) { return bf, nil }

func (bf *bufferFile) Read(b []byte) (int, error) { return bf.buf.Read(b) }

func (bf *bufferFile) Close() error {
	if bf.buf == nil {
		return fmt.Errorf("already closed")
	}
	bufpool.Put(bf.buf)
	bf.buf = nil
	return nil
}

func writeCatalog(catalog *Catalog, fsys fs.FS, historyTable, s string) error {
	s = filepath.ToSlash(s)
	if strings.HasSuffix(s, ".json") {
		file, err := fsys.Open(s)
		if err != nil {
			return err
		}
		defer file.Close()
		err = json.NewDecoder(file).Decode(catalog)
		if err != nil {
			return err
		}
		return nil
	}
	if strings.HasSuffix(s, ".go") {
		file, err := fsys.Open(s)
		if err != nil {
			return err
		}
		defer file.Close()
		p := NewStructParser(nil)
		err = p.ParseFile(file)
		if err != nil {
			return err
		}
		err = p.WriteCatalog(catalog)
		if err != nil {
			return err
		}
		return nil
	}
	if strings.HasSuffix(s, ".zip") {
		return writeCatalogFromZip(catalog, fsys, s)
	}
	if strings.HasSuffix(s, ".tgz") || strings.HasSuffix(s, ".tar.gz") {
		return writeCatalogFromTgz(catalog, fsys, s)
	}
	if isDir(fsys, s) {
		file, err := fsys.Open(s + "/schema.json")
		if err != nil {
			return err
		}
		defer file.Close()
		err = json.NewDecoder(file).Decode(catalog)
		if err != nil {
			return err
		}
		return nil
	}
	dialect, driverName, dsn := normalizeDSN(s)
	if dialect == "" {
		return fmt.Errorf("could not identity dialect for %q", s)
	}
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return err
	}
	dbi := NewDatabaseIntrospector(dialect, db)
	dbi.ObjectTypes = []string{"TABLES"}
	dbi.ExcludeTables = []string{historyTable}
	err = dbi.WriteCatalog(catalog)
	if err != nil {
		return err
	}
	return nil
}

func writeCatalogFromZip(catalog *Catalog, fsys fs.FS, s string) error {
	file, err := fsys.Open(s)
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
		return fmt.Errorf("cannot read %s because it is not of type io.ReaderAt", s)
	}
	r, err := zip.NewReader(readerAt, fileinfo.Size())
	if err != nil {
		return err
	}
	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "__MACOSX/") {
			// https://superuser.com/questions/104500/what-is-macosx-folder
			continue
		}
		if filepath.Base(f.Name) != "schema.json" {
			continue
		}
		schemaFile, err := f.Open()
		if err != nil {
			return err
		}
		defer schemaFile.Close()
		err = json.NewDecoder(schemaFile).Decode(catalog)
		if err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("%s does not contain schema.json", s)
}

func writeCatalogFromTgz(catalog *Catalog, fsys fs.FS, s string) error {
	file, err := fsys.Open(s)
	if err != nil {
		return err
	}
	defer file.Close()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	tarReader := tar.NewReader(gzipReader)
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if filepath.Base(hdr.Name) != "schema.json" {
			continue
		}
		err = json.NewDecoder(tarReader).Decode(catalog)
		if err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("%s does not contain schema.json", s)
}

func isDir(fsys fs.FS, s string) bool {
	file, err := fsys.Open(s)
	if err != nil {
		return false
	}
	defer file.Close()
	fileinfo, err := file.Stat()
	if err != nil {
		return false
	}
	return fileinfo.IsDir()
}
