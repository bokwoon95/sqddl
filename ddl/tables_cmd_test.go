package ddl

import (
	"bytes"
	"database/sql"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/bokwoon95/sqddl/internal/testutil"
)

func TestTablesCmd(t *testing.T) {
	t.Parallel()
	dsn := "sqlite:file:/" + t.Name() + "?vfs=memdb&_foreign_keys=true"
	db, err := sql.Open("sqlite3", strings.TrimPrefix(dsn, "sqlite:"))
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	defer db.Close()

	migrateCmd, err := MigrateCommand("-db", dsn, "-dir", "sqlite_migrations")
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	migrateCmd.Stderr = io.Discard
	migrateCmd.db = "" // Keep database open after running command.
	defer migrateCmd.DB.Close()
	err = migrateCmd.Run()
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}

	buf := &bytes.Buffer{}
	tablesCmd, err := TablesCommand("-db", dsn, "-pkg", "sakila")
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	tablesCmd.Stdout = buf
	tablesCmd.db = "" // Keep database open after running command.
	defer tablesCmd.DB.Close()
	err = tablesCmd.Run()
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	b, err := os.ReadFile("testdata/sqlite/tables.go")
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	wantOutput := strings.ReplaceAll(string(b), "\r\n", "\n")
	gotOutput := buf.String()
	if diff := testutil.Diff(gotOutput, wantOutput); diff != "" {
		t.Error(testutil.Callers(), diff)
	}
}
