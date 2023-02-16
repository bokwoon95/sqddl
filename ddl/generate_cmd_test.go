package ddl

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bokwoon95/sqddl/internal/testutil"
)

func TestGenerateCmd(t *testing.T) {
	type TT struct {
		dialect string
		dir     string
	}
	tests := []TT{
		{dialect: "mysql", dir: "testdata/mysql_add"},
		{dialect: "mysql", dir: "testdata/mysql_alter"},
		{dialect: "mysql", dir: "testdata/mysql_drop"},
		{dialect: "mysql", dir: "testdata/mysql_schema"},
		{dialect: "mysql", dir: "testdata/mysql_table"},
		{dialect: "mysql", dir: "testdata/mysql_ignore"},
		{dialect: "postgres", dir: "testdata/postgres_add"},
		{dialect: "postgres", dir: "testdata/postgres_alter"},
		{dialect: "postgres", dir: "testdata/postgres_drop"},
		{dialect: "postgres", dir: "testdata/postgres_schema"},
		{dialect: "postgres", dir: "testdata/postgres_table"},
		{dialect: "postgres", dir: "testdata/postgres_ignore"},
		{dialect: "sqlite", dir: "testdata/sqlite_create_schema"},
		{dialect: "sqlite", dir: "testdata/sqlite_drop_schema"},
		{dialect: "sqlite", dir: "testdata/sqlite_empty"},
		{dialect: "sqlite", dir: "testdata/sqlite_misc"},
		{dialect: "sqlite", dir: "testdata/sqlite_ignore"},
		{dialect: "sqlserver", dir: "testdata/sqlserver_add"},
		{dialect: "sqlserver", dir: "testdata/sqlserver_alter"},
		{dialect: "sqlserver", dir: "testdata/sqlserver_drop"},
		{dialect: "sqlserver", dir: "testdata/sqlserver_schema"},
		{dialect: "sqlserver", dir: "testdata/sqlserver_table"},
		{dialect: "sqlserver", dir: "testdata/sqlserver_ignore"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.dir, func(t *testing.T) {
			t.Parallel()
			tempDir := t.TempDir()
			generateCmd, err := GenerateCommand(
				"-src", filepath.Join(tt.dir, "src.go"),
				"-dest", filepath.Join(tt.dir, "dest.go"),
				"-output-dir", tempDir,
				"-prefix", strings.TrimPrefix(tt.dir, "testdata/"+tt.dialect+"_"),
				"-drop-objects",
				"-dialect", tt.dialect,
				"-accept-warnings",
			)
			if err != nil {
				t.Fatal(testutil.Callers(), err)
			}
			generateCmd.Stderr = io.Discard
			err = generateCmd.Run()
			if err != nil {
				t.Error(testutil.Callers(), err)
			}
		})
	}
}

func Test_writeCatalog(t *testing.T) {
	dsn := "file:/" + t.Name() + ".db?vfs=memdb&_foreign_keys=true"
	loadCmd, err := LoadCommand("-db", dsn, "-dir", "testdata/sqlite", "dump.zip")
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	loadCmd.Stderr = io.Discard
	loadCmd.db = "" // Keep database open after running command.
	defer loadCmd.DB.Close()
	err = loadCmd.Run()
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	wantCatalog := &Catalog{}
	dbi := NewDatabaseIntrospector("sqlite", loadCmd.DB)
	dbi.ObjectTypes = []string{"TABLES"}
	dbi.ExcludeTables = []string{"sqddl_history"}
	err = dbi.WriteCatalog(wantCatalog)
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}

	sources := []string{
		"testdata/sqlite/schema.json",
		"testdata/sqlite/tables.go",
		"testdata/sqlite/dump.zip",
		"testdata/sqlite/dump.tgz",
		"testdata/sqlite",
		dsn,
	}
	for _, source := range sources {
		source := source
		t.Run(source, func(t *testing.T) {
			gotCatalog := &Catalog{Dialect: "sqlite"}
			err = writeCatalog(gotCatalog, os.DirFS("."), "sqddl_history", source)
			if err != nil {
				t.Fatal(testutil.Callers(), err)
			}
			cache := NewCatalogCache(gotCatalog)
			for _, wantSchema := range wantCatalog.Schemas {
				for _, wantTable := range wantSchema.Tables {
					gotTable := cache.GetTable(cache.GetSchema(gotCatalog, wantSchema.SchemaName), wantTable.TableName)
					if gotTable == nil {
						t.Fatalf(testutil.Callers()+" missing table %q", wantTable.TableName)
					}
					if diff := testutil.Diff(gotTable.Columns, wantTable.Columns); diff != "" {
						t.Error(testutil.Callers(), diff)
					}
					if strings.HasSuffix(source, ".go") {
						continue
					}
					if diff := testutil.Diff(gotTable.Constraints, wantTable.Constraints); diff != "" {
						t.Error(testutil.Callers(), diff)
					}
					if diff := testutil.Diff(gotTable.Indexes, wantTable.Indexes); diff != "" {
						t.Error(testutil.Callers(), diff)
					}
				}
			}
		})
	}
}
