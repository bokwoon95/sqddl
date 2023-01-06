package ddl

import (
	"database/sql"
	"io"
	"strings"
	"testing"

	"github.com/bokwoon95/sq"
	"github.com/bokwoon95/sqddl/internal/testutil"
)

func TestMvCmd(t *testing.T) {
	t.Parallel()
	dsn := "sqlite:file:/" + t.Name() + "?vfs=memdb&_foreign_keys=true"
	db, err := sql.Open("sqlite3", strings.TrimPrefix(dsn, "sqlite:"))
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	defer db.Close()

	// Touch customer_list.sql.
	touchCmd, err := TouchCommand(
		"-db", dsn,
		"-dir", "sqlite_migrations",
		"repeatable/views/customer_list.sql",
	)
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	touchCmd.Stderr = io.Discard
	touchCmd.db = "" // Keep database open after running command.
	defer touchCmd.DB.Close()
	err = touchCmd.Run()
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}

	// Rename customer_list.sql to customer_list2.sql.
	mvCmd, err := MvCommand(
		"-db", dsn,
		"repeatable/views/customer_list.sql",
		"repeatable/views/customer_list2.sql",
	)
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	mvCmd.Stderr = io.Discard
	mvCmd.db = "" // Keep database open after running command.
	defer mvCmd.DB.Close()
	err = mvCmd.Run()
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}

	// Assert customer_list2.sql.
	wantFilenames := []string{"repeatable/views/customer_list2.sql"}
	gotFilenames, err := sq.FetchAll(db, sq.
		Queryf("SELECT {*} FROM sqddl_history ORDER BY filename"),
		func(row *sq.Row) string {
			return row.String("filename")
		},
	)
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	if diff := testutil.Diff(gotFilenames, wantFilenames); diff != "" {
		t.Fatal(testutil.Callers(), diff)
	}
}
