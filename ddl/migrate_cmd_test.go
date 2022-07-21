package ddl

import (
	"database/sql"
	"io"
	"testing"
	"testing/fstest"

	"github.com/bokwoon95/sq"
	"github.com/bokwoon95/sqddl/internal/testutil"
)

func TestMigrateCmd(t *testing.T) {
	type historyTableEntry struct {
		filename string
		success  bool
	}

	assertHistoryTable := func(t *testing.T, db sq.DB, wantEntries []historyTableEntry) {
		gotEntries, err := sq.FetchAll(db, sq.
			Queryf("SELECT {*} FROM sqddl_history ORDER BY filename"),
			func(row *sq.Row) (historyTableEntry, error) {
				return historyTableEntry{
					filename: row.String("filename"),
					success:  row.Bool("success"),
				}, nil
			},
		)
		if err != nil {
			t.Fatal(testutil.Callers(), err)
		}
		if diff := testutil.Diff(gotEntries, wantEntries); diff != "" {
			t.Fatal(testutil.Callers(), diff)
		}
	}

	assertTables := func(t *testing.T, db sq.DB, wantTables []string) {
		gotTables, err := sq.FetchAll(db, sq.
			Queryf("SELECT {*} FROM sqlite_schema WHERE type = 'table' ORDER BY tbl_name"),
			func(row *sq.Row) (string, error) {
				return row.String("tbl_name"), nil
			},
		)
		if err != nil {
			t.Fatal(testutil.Callers(), err)
		}
		if diff := testutil.Diff(gotTables, wantTables); diff != "" {
			t.Fatal(testutil.Callers(), diff)
		}
	}

	t.Run("migrate all", func(t *testing.T) {
		t.Parallel()
		migrateCmd, err := MigrateCommand(
			"-db", "sqlite:file/"+t.Name()+"?vfs=memdb&_foreign_keys=true",
			"-dir", "sqlite_migrations",
		)
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
		assertHistoryTable(t, migrateCmd.DB, []historyTableEntry{
			{"01_sakila.sql", true},
			{"02_employee.sql", true},
			{"repeatable/fts/film_text.sql", true},
			{"repeatable/triggers/actor.sql", true},
			{"repeatable/triggers/address.sql", true},
			{"repeatable/triggers/category.sql", true},
			{"repeatable/triggers/city.sql", true},
			{"repeatable/triggers/country.sql", true},
			{"repeatable/triggers/customer.sql", true},
			{"repeatable/triggers/film.sql", true},
			{"repeatable/triggers/film_actor.sql", true},
			{"repeatable/triggers/film_category.sql", true},
			{"repeatable/triggers/inventory.sql", true},
			{"repeatable/triggers/language.sql", true},
			{"repeatable/triggers/rental.sql", true},
			{"repeatable/triggers/staff.sql", true},
			{"repeatable/triggers/store.sql", true},
			{"repeatable/views/actor_info.sql", true},
			{"repeatable/views/customer_list.sql", true},
			{"repeatable/views/film_list.sql", true},
			{"repeatable/views/full_address.sql", true},
			{"repeatable/views/nicer_but_slower_film_list.sql", true},
			{"repeatable/views/sales_by_film_category.sql", true},
			{"repeatable/views/sales_by_store.sql", true},
			{"repeatable/views/staff_list.sql", true},
		})
	})

	t.Run("migrate some", func(t *testing.T) {
		t.Parallel()
		dsn := "sqlite:file:/" + t.Name() + "?vfs=memdb&_foreign_keys=true"
		migrateCmd, err := MigrateCommand(
			"-db", dsn,
			"-dir", "sqlite_migrations",
			"01_sakila.sql",
			"sqlite_migrations/02_employee.sql",
			"repeatable/views/actor_info.sql",
		)
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
		assertHistoryTable(t, migrateCmd.DB, []historyTableEntry{
			{"01_sakila.sql", true},
			{"02_employee.sql", true},
			{"repeatable/views/actor_info.sql", true},
		})
	})

	t.Run("migration failure (transactional)", func(t *testing.T) {
		t.Parallel()
		db, err := sql.Open("sqlite3", ":memory:?_foreign_keys=true")
		if err != nil {
			t.Fatal(testutil.Callers(), err)
		}
		migrateCmd := &MigrateCmd{
			Dialect: "sqlite",
			DB:      db,
			DirFS: fstest.MapFS{
				"01_table1.sql": &fstest.MapFile{
					Data: []byte("CREATE TABLE table1 ( id INT );"),
				},
				"02_table2.sql": &fstest.MapFile{
					// 02_table2.sql is primed to fail.
					Data: []byte("CREATE TABLE table2 ( id INT ); fail_here"),
				},
				"03_table3.sql": &fstest.MapFile{
					Data: []byte("CREATE TABLE table3 ( id INT );"),
				},
			},
			Stderr: io.Discard,
		}
		err = migrateCmd.Run()
		if err == nil {
			t.Error(testutil.Callers(), "expected error but got nil")
		}
		// Make sure only 01_table1.sql and 02_table2.sql entries exist, and
		// that both are marked as failed.
		assertHistoryTable(t, db, []historyTableEntry{
			{"01_table1.sql", false},
			{"02_table2.sql", false},
		})
		// Make sure all tables in the transaction were rolled back, no tables
		// should exist except for the history table.
		assertTables(t, db, []string{
			"sqddl_history",
		})
	})

	t.Run("migration failure (non-transactional)", func(t *testing.T) {
		t.Parallel()
		db, err := sql.Open("sqlite3", ":memory:?_foreign_keys=true")
		if err != nil {
			t.Fatal(testutil.Callers(), err)
		}
		migrateCmd := &MigrateCmd{
			Dialect: "sqlite",
			DB:      db,
			DirFS: fstest.MapFS{
				"01_table1.txoff.sql": &fstest.MapFile{
					Data: []byte("CREATE TABLE table1 ( id INT );"),
				},
				"01_table1.undo.sql": &fstest.MapFile{
					Data: []byte("DROP TABLE IF EXISTS table1;"),
				},
				"02_table2.txoff.sql": &fstest.MapFile{
					// 02_table2.txoff.sql is primed to fail.
					Data: []byte("CREATE TABLE table2 ( id INT ); fail_here"),
				},
				"02_table2.undo.sql": &fstest.MapFile{
					Data: []byte("DROP TABLE IF EXISTS table2;"),
				},
				"03_table3.txoff.sql": &fstest.MapFile{
					Data: []byte("CREATE TABLE table3 ( id INT );"),
				},
				"03_table3.undo.sql": &fstest.MapFile{
					Data: []byte("DROP TABLE IF EXISTS table3;"),
				},
			},
			Stderr: io.Discard,
		}
		err = migrateCmd.Run()
		if err == nil {
			t.Error(testutil.Callers(), "expected error but got nil")
		}
		// Make sure only 01_table1.txoff.sql and 02_table2.txoff.sql entries
		// exist, and 01_table1.txoff.sql succeeded while 02_table2.txoff.sql
		// failed.
		assertHistoryTable(t, db, []historyTableEntry{
			{"01_table1.txoff.sql", true},
			{"02_table2.txoff.sql", false},
		})
		// Make sure 02_table2.txoff.sql was successfully undone by
		// 02_table2.undo.sql, so only table1 (and the history table) exists.
		assertTables(t, db, []string{
			"sqddl_history",
			"table1",
		})
	})
}
