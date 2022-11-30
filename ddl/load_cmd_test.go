package ddl

import (
	"database/sql"
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"testing"

	"github.com/bokwoon95/sq"
	"github.com/bokwoon95/sqddl/internal/testutil"
)

func TestLoadCmd(t *testing.T) {
	wantCatalog := &Catalog{}
	b, err := os.ReadFile("testdata/sqlite/schema.json")
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	err = json.Unmarshal(b, &wantCatalog)
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	wantCatalog.CatalogName = ""

	assertTables := func(t *testing.T, db *sql.DB) {
		gotCatalog := &Catalog{}
		err = NewDatabaseIntrospector("sqlite", db).WriteCatalog(gotCatalog)
		if err != nil {
			t.Fatal(testutil.Callers(), err)
		}
		cache := NewCatalogCache(gotCatalog)
		for _, wantSchema := range wantCatalog.Schemas {
			for _, wantTable := range wantSchema.Tables {
				gotTable := cache.GetTable(cache.GetSchema(gotCatalog, wantSchema.SchemaName), wantTable.TableName)
				if gotTable == nil {
					t.Errorf("table %q doesn't exist", wantTable.TableName)
					continue
				}
				if diff := testutil.Diff(gotTable.Columns, wantTable.Columns); diff != "" {
					t.Error(testutil.Callers(), diff)
				}
				if diff := testutil.Diff(gotTable.Constraints, wantTable.Constraints); diff != "" {
					t.Error(testutil.Callers(), diff)
				}
				if diff := testutil.Diff(gotTable.Indexes, wantTable.Indexes); diff != "" {
					t.Error(testutil.Callers(), diff)
				}
			}
		}
	}

	t.Run("dir", func(t *testing.T) {
		t.Parallel()
		loadCmd, err := LoadCommand(
			"-db", "sqlite:file:/"+t.Name()+"?vfs=memdb&_foreign_keys=true",
			"-verbose",
			"testdata/sqlite/schema.sql",
			"csv_testdata",
			"testdata/sqlite/indexes.sql",
			"testdata/sqlite/constraints.sql",
		)
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
		assertTables(t, loadCmd.DB)
	})

	t.Run("zip", func(t *testing.T) {
		t.Parallel()
		db, err := sql.Open("sqlite3", "file:/"+t.Name()+"?vfs=memdb&_foreign_keys=true")
		if err != nil {
			t.Fatal(testutil.Callers(), err)
		}
		loadCmd := &LoadCmd{
			DB:        db,
			Dialect:   "sqlite",
			DirFS:     os.DirFS("testdata/sqlite"),
			Filenames: []string{"dump.zip"},
			Stderr:    io.Discard,
		}
		err = loadCmd.Run()
		if err != nil {
			t.Fatal(testutil.Callers(), err)
		}
		assertTables(t, db)
	})

	t.Run("tgz", func(t *testing.T) {
		t.Parallel()
		db, err := sql.Open("sqlite3", "file:/"+t.Name()+"?vfs=memdb&_foreign_keys=true")
		if err != nil {
			t.Fatal(testutil.Callers(), err)
		}
		fsys, err := fs.Sub(testFS, "testdata/sqlite")
		if err != nil {
			t.Fatal(testutil.Callers(), err)
		}
		loadCmd := &LoadCmd{
			DB:        db,
			Dialect:   "sqlite",
			DirFS:     fsys,
			Filenames: []string{"dump.tgz"},
			Stderr:    io.Discard,
		}
		err = loadCmd.Run()
		if err != nil {
			t.Fatal(testutil.Callers(), err)
		}
		assertTables(t, db)
	})

	t.Run("individual files", func(t *testing.T) {
		t.Parallel()
		loadCmd, err := LoadCommand(
			"-db", "sqlite:file:/"+t.Name()+"?vfs=memdb&_foreign_keys=true",
			"testdata/sqlite/schema.sql",
			"csv_testdata/actor.csv",
			"testdata/sqlite/indexes.sql",
			"sqlite_migrations/repeatable/fts/film_text.sql",
			"sqlite_migrations/repeatable/views/actor_info.sql",
		)
		if err != nil {
			t.Fatal(testutil.Callers(), err)
		}
		loadCmd.Stderr = io.Discard
		err = loadCmd.Run()
		if err != nil {
			t.Fatal(testutil.Callers(), err)
		}
	})

	assertOrderedCSV := func(t *testing.T, db *sql.DB) {
		wantA := []int{1, 2, 3}
		gotA, err := sq.FetchAll(db, sq.Queryf("SELECT {*} FROM a"), func(row *sq.Row) int {
			return row.Int("a")
		})
		if err != nil {
			t.Fatal(testutil.Callers(), err)
		}
		if diff := testutil.Diff(gotA, wantA); diff != "" {
			t.Fatal(testutil.Callers(), diff)
		}

		wantB := [][2]int{{1, 1}, {2, 2}, {3, 3}}
		gotB, err := sq.FetchAll(db, sq.Queryf("SELECT {*} FROM b"), func(row *sq.Row) [2]int {
			return [2]int{row.Int("b"), row.Int("a")}
		})
		if err != nil {
			t.Fatal(testutil.Callers(), err)
		}
		if diff := testutil.Diff(gotB, wantB); diff != "" {
			t.Fatal(testutil.Callers(), diff)
		}

		wantC := [][2]int{{1, 1}, {2, 2}, {3, 3}}
		gotC, err := sq.FetchAll(db, sq.Queryf("SELECT {*} FROM c"), func(row *sq.Row) [2]int {
			return [2]int{row.Int("c"), row.Int("b")}
		})
		if err != nil {
			t.Fatal(testutil.Callers(), err)
		}
		if diff := testutil.Diff(gotC, wantC); diff != "" {
			t.Fatal(testutil.Callers(), diff)
		}
	}

	t.Run("ordered csv (dir)", func(t *testing.T) {
		t.Parallel()
		loadCmd, err := LoadCommand(
			"-db", "sqlite:file:/"+t.Name()+"?vfs=memdb&_foreign_keys=true",
			"testdata/ordered_csv",
		)
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
		assertOrderedCSV(t, loadCmd.DB)
	})

	t.Run("ordered csv (zip)", func(t *testing.T) {
		t.Parallel()
		loadCmd, err := LoadCommand(
			"-db", "sqlite:file:/"+t.Name()+"?vfs=memdb&_foreign_keys=true",
			"testdata/ordered_csv.zip",
		)
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
		assertOrderedCSV(t, loadCmd.DB)
	})

	t.Run("ordered csv (tar gz)", func(t *testing.T) {
		t.Parallel()
		loadCmd, err := LoadCommand(
			"-db", "sqlite:file:/"+t.Name()+"?vfs=memdb&_foreign_keys=true",
			"testdata/ordered_csv.tar.gz",
		)
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
		assertOrderedCSV(t, loadCmd.DB)
	})
}
