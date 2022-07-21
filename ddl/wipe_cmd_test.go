package ddl

import (
	"database/sql"
	"io"
	"strings"
	"testing"

	"github.com/bokwoon95/sqddl/internal/testutil"
)

func TestWipeCmd(t *testing.T) {
	t.Parallel()
	dsn := "sqlite:file:/" + t.Name() + "?vfs=memdb&_foreign_keys=true"
	db, err := sql.Open("sqlite3", strings.TrimPrefix(dsn, "sqlite:"))
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	defer db.Close()

	// Load dump.zip.
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

	// Wipe db.
	wipeCmd, err := WipeCommand("-db", dsn)
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	wipeCmd.db = "" // Keep database open after running command.
	defer wipeCmd.DB.Close()
	err = wipeCmd.Run()
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}

	// Assert empty db catalog.
	wantCatalog := &Catalog{}
	gotCatalog := &Catalog{}
	err = NewDatabaseIntrospector("sqlite", db).WriteCatalog(gotCatalog)
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	gotCatalog.Dialect = ""
	gotCatalog.VersionNums = nil
	gotCatalog.DefaultCollationValid = false
	if diff := testutil.Diff(gotCatalog, wantCatalog); diff != "" {
		t.Fatal(testutil.Callers(), diff)
	}
}
