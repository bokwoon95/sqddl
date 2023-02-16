package ddl

import (
	"bytes"
	"database/sql"
	"flag"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bokwoon95/sq"
)

var generateGoldenFiles = flag.Bool("generate-golden-files", false, "")

func TestMain(m *testing.M) {
	flag.Parse()
	if !*generateGoldenFiles {
		os.Exit(m.Run())
	}

	l := log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)
	db, err := sql.Open("sqlite3", sqliteDSN)
	if err != nil {
		l.Fatalf("error connecting to %q: %s", sqliteDSN, err)
	}
	defer db.Close()

	wipeCmd := &WipeCmd{Dialect: "sqlite", DB: db}
	err = wipeCmd.Run()
	if err != nil {
		l.Fatal(err)
	}

	migrateCmd := &MigrateCmd{Dialect: "sqlite", DB: db, DirFS: os.DirFS("sqlite_migrations")}
	migrateCmd.Stderr = io.Discard
	err = migrateCmd.Run()
	if err != nil {
		l.Fatal(err)
	}

	loadCmd := &LoadCmd{Dialect: "sqlite", DB: db, Filenames: []string{"csv_testdata"}}
	loadCmd.Stderr = io.Discard
	err = loadCmd.Run()
	if err != nil {
		l.Fatal(err)
	}

	dumpCmd := &DumpCmd{Dialect: "sqlite", DB: db, OutputDir: "testdata/sqlite", SchemaOnly: true}
	dumpCmd.Stderr = io.Discard
	err = dumpCmd.Run()
	if err != nil {
		l.Fatal(err)
	}

	dumpCmd = &DumpCmd{Dialect: "sqlite", DB: db, OutputDir: "testdata/sqlite", Zip: "dump.zip"}
	dumpCmd.Stderr = io.Discard
	err = dumpCmd.Run()
	if err != nil {
		l.Fatal(err)
	}

	dumpCmd = &DumpCmd{Dialect: "sqlite", DB: db, OutputDir: "testdata/sqlite", Tgz: "dump.tgz"}
	dumpCmd.Stderr = io.Discard
	err = dumpCmd.Run()
	if err != nil {
		l.Fatal(err)
	}

	tablesCmd := &TablesCmd{Dialect: "sqlite", DB: db, Filename: "testdata/sqlite/tables.go", PackageName: "sakila"}
	err = tablesCmd.Run()
	if err != nil {
		l.Fatal(err)
	}

	viewsCmd := &ViewsCmd{Dialect: "sqlite", DB: db, Filename: "testdata/sqlite/views.go", PackageName: "sakila"}
	err = viewsCmd.Run()
	if err != nil {
		l.Fatal(err)
	}

	err = os.RemoveAll("testdata/subset")
	if err != nil {
		l.Fatal(err)
	}
	dumpCmd = &DumpCmd{
		Dialect: "sqlite", DB: db, OutputDir: "testdata/subset", DataOnly: true,
		SubsetQueries: []string{
			"SELECT {*} FROM {film} ORDER BY film_id LIMIT 10",
			"SELECT {*} FROM {actor}",
		},
	}
	dumpCmd.Stderr = io.Discard
	err = dumpCmd.Run()
	if err != nil {
		l.Fatal(err)
	}

	err = os.RemoveAll("testdata/extended_subset")
	if err != nil {
		l.Fatal(err)
	}
	dumpCmd = &DumpCmd{
		Dialect: "sqlite", DB: db, OutputDir: "testdata/extended_subset", DataOnly: true,
		ExtendedSubsetQueries: []string{
			"SELECT {*} FROM {film} ORDER BY film_id LIMIT 10",
		},
	}
	dumpCmd.Stderr = io.Discard
	err = dumpCmd.Run()
	if err != nil {
		l.Fatal(err)
	}

	p := NewStructParser(nil)
	file, err := os.Open("testdata/tables.go")
	if err != nil {
		l.Fatal(err)
	}
	defer file.Close()
	err = p.ParseFile(file)
	if err != nil {
		l.Fatal(err)
	}
	dialects := []string{"sqlite", "postgres", "mysql", "sqlserver"}
	for _, dialect := range dialects {
		catalog := &Catalog{Dialect: dialect}
		err = p.WriteCatalog(catalog)
		if err != nil {
			l.Fatalf("%s: writing catalog: %s", dialect, err)
		}
		buf := &bytes.Buffer{}
		for i := range catalog.Schemas {
			schema := &catalog.Schemas[i]
			for j := range schema.Tables {
				table := &schema.Tables[j]
				if table.Ignore || (table.IsVirtual && table.SQL == "") {
					continue
				}
				if buf.Len() > 0 {
					buf.WriteString("\n")
				}
				writeCreateTable(dialect, buf, "", "", table, true)
				for k := range table.Indexes {
					index := &table.Indexes[k]
					if buf.Len() > 0 {
						buf.WriteString("\n")
					}
					writeCreateIndex(dialect, buf, "", index, false)
				}
			}
		}
		if dialect != "sqlite" {
			for i := range catalog.Schemas {
				schema := &catalog.Schemas[i]
				for j := range schema.Tables {
					table := &schema.Tables[j]
					if table.Ignore || (table.IsVirtual && table.SQL == "") {
						continue
					}
					for k := range table.Constraints {
						constraint := &table.Constraints[k]
						if constraint.ConstraintType != FOREIGN_KEY {
							continue
						}
						if buf.Len() > 0 {
							buf.WriteString("\n")
						}
						tableName := sq.QuoteIdentifier(dialect, constraint.TableName)
						if constraint.TableSchema != "" {
							tableName = sq.QuoteIdentifier(dialect, constraint.TableSchema) + "." + tableName
						}
						buf.WriteString("ALTER TABLE " + tableName + " ADD ")
						writeConstraintDefinition(dialect, buf, "", constraint)
						buf.WriteString(";\n")
					}
				}
			}
		}
		filename := "testdata/tables." + dialect + ".sql"
		file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			l.Fatalf("%s: opening %s: %s", dialect, filename, err)
		}
		_, err = buf.WriteTo(file)
		if err != nil {
			l.Fatalf("%s: writing to %s: %s", dialect, filename, err)
		}
		err = file.Close()
		if err != nil {
			l.Fatalf("%s: closing %s: %s", dialect, filename, err)
		}
	}

	type testcase struct {
		dir         string
		dropObjects bool
	}

	for _, database := range []struct {
		dialect       string
		driver        string
		dsn           string
		currentSchema string
		testcases     []testcase
	}{
		{
			dialect: "sqlite", driver: "sqlite3", dsn: sqliteDSN,
			testcases: []testcase{
				{"testdata/sqlite_create_schema", true},
				{"testdata/sqlite_drop_schema", true},
				{"testdata/sqlite_empty", true},
				{"testdata/sqlite_misc", true},
				{"testdata/sqlite_ignore", false},
			},
		},
		{
			dialect: "postgres", driver: "postgres", dsn: *postgresDSN,
			currentSchema: "public",
			testcases: []testcase{
				{"testdata/postgres_add", false},
				{"testdata/postgres_alter", false},
				{"testdata/postgres_drop", true},
				{"testdata/postgres_schema", true},
				{"testdata/postgres_table", true},
			},
		},
		{
			dialect: "mysql", driver: "mysql", dsn: *mysqlDSN,
			currentSchema: "sakila",
			testcases: []testcase{
				{"testdata/mysql_add", false},
				{"testdata/mysql_alter", false},
				{"testdata/mysql_drop", true},
				{"testdata/mysql_schema", true},
				{"testdata/mysql_table", true},
			},
		},
		{
			dialect: "sqlserver", driver: "sqlserver", dsn: *sqlserverDSN,
			currentSchema: "dbo",
			testcases: []testcase{
				{"testdata/sqlserver_add", false},
				{"testdata/sqlserver_alter", false},
				{"testdata/sqlserver_drop", true},
				{"testdata/sqlserver_schema", true},
				{"testdata/sqlserver_table", true},
			},
		},
	} {
		if database.dsn == "" {
			continue
		}
		db, err := sql.Open(database.driver, database.dsn)
		if err != nil {
			l.Fatalf("%s: error connecting to %s: %s", database.dialect, sqliteDSN, err)
		}

		wipeCmd := &WipeCmd{Dialect: database.dialect, DB: db}
		err = wipeCmd.Run()
		if err != nil {
			l.Fatalf("%s: %s", database.dialect, err)
		}

		migrateCmd := &MigrateCmd{Dialect: database.dialect, DB: db, DirFS: os.DirFS(database.dialect + "_migrations")}
		migrateCmd.Stderr = io.Discard
		err = migrateCmd.Run()
		if err != nil {
			l.Fatalf("%s: %s", database.dialect, err)
		}

		dumpCmd := &DumpCmd{Dialect: database.dialect, DB: db, OutputDir: "testdata/" + database.dialect, SchemaOnly: true}
		dumpCmd.Stderr = io.Discard
		err = dumpCmd.Run()
		if err != nil {
			l.Fatalf("%s: %s", database.dialect, err)
		}

		for _, tc := range database.testcases {
			srcCatalog := &Catalog{Dialect: database.dialect, CurrentSchema: database.currentSchema}
			err = writeCatalog(srcCatalog, os.DirFS("."), "sqddl_history", filepath.Join(tc.dir, "src.go"))
			if err != nil {
				l.Fatalf("%s: %s: %s", database.dialect, tc.dir, err)
			}
			destCatalog := &Catalog{Dialect: database.dialect, CurrentSchema: database.currentSchema}
			err = writeCatalog(destCatalog, os.DirFS("."), "sqddl_history", filepath.Join(tc.dir, "dest.go"))
			if err != nil {
				l.Fatalf("%s: %s: %s", database.dialect, tc.dir, err)
			}
			var filenames, warnings []string
			var bufs []*bytes.Buffer
			switch database.dialect {
			case "sqlite":
				m := newSQLiteMigration(srcCatalog, destCatalog, tc.dropObjects)
				filenames, bufs, warnings = m.sql(strings.TrimPrefix(filepath.Base(tc.dir), "sqlite_"))
			case "postgres":
				m := newPostgresMigration(srcCatalog, destCatalog, tc.dropObjects)
				filenames, bufs, warnings = m.sql(strings.TrimPrefix(filepath.Base(tc.dir), "postgres_"))
			case "mysql":
				m := newMySQLMigration(srcCatalog, destCatalog, tc.dropObjects)
				filenames, bufs, warnings = m.sql(strings.TrimPrefix(filepath.Base(tc.dir), "mysql_"))
			case "sqlserver":
				m := newSQLServerMigration(srcCatalog, destCatalog, tc.dropObjects)
				filenames, bufs, warnings = m.sql(strings.TrimPrefix(filepath.Base(tc.dir), "sqlserver_"))
			default:
				continue
			}
			err = fs.WalkDir(os.DirFS(tc.dir), ".", func(path string, d fs.DirEntry, err error) error {
				if path != "." && d.IsDir() {
					return fs.SkipDir
				}
				if !strings.HasSuffix(path, ".sql") && !strings.HasSuffix(path, ".txt") {
					return nil
				}
				return os.Remove(filepath.Join(tc.dir, path))
			})
			if err != nil {
				l.Fatalf("%s: %s: %s", database.dialect, tc.dir, err)
			}
			for i, filename := range filenames {
				err := os.WriteFile(filepath.Join(tc.dir, filename), bufs[i].Bytes(), 0644)
				if err != nil {
					l.Fatalf("%s: %s: %s, %s", database.dialect, tc.dir, filename, err)
				}
			}
			if len(warnings) > 0 {
				err := os.WriteFile(filepath.Join(tc.dir, "warnings.txt"), []byte(strings.Join(warnings, "\n")), 0644)
				if err != nil {
					l.Fatalf("%s: %s: %s", database.dialect, tc.dir, err)
				}
			}
		}
	}

	os.Exit(m.Run())
}
