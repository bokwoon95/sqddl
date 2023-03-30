package ddl

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/bokwoon95/sqddl/internal/testutil"
	mssql "github.com/denisenkom/go-mssqldb"
	"github.com/go-sql-driver/mysql"
	"github.com/lib/pq"
)

var (
	sqliteDSN    = "sakila.sqlite3"
	postgresDSN  = flag.String("postgres", "", "")
	mysqlDSN     = flag.String("mysql", "", "")
	sqlserverDSN = flag.String("sqlserver", "", "")
)

func TestSQLite(t *testing.T) {
	const (
		dialect = "sqlite"
		driver  = "sqlite3"
	)
	t.Parallel()
	testMigrateIntrospect(t, dialect, driver, sqliteDSN)
	testAutomigrate(t, dialect, sqliteDSN,
		"testdata/sqlite_create_schema",
		"testdata/sqlite_drop_schema",
		"testdata/sqlite_empty",
		"testdata/sqlite_misc",
		"testdata/sqlite_ignore",
	)
	testLoadDump(t, dialect, sqliteDSN, nil)
}

func TestPostgres(t *testing.T) {
	const (
		dialect = "postgres"
		driver  = "postgres"
	)
	if *postgresDSN == "" {
		return
	}
	t.Parallel()
	testMigrateIntrospect(t, dialect, driver, *postgresDSN)
	testAutomigrate(t, dialect, *postgresDSN,
		"testdata/postgres_add",
		"testdata/postgres_alter",
		"testdata/postgres_drop",
		"testdata/postgres_schema",
		"testdata/postgres_table",
		"testdata/postgres_ignore",
	)
	testLoadDump(t, dialect, *postgresDSN, map[string]func([]string) []string{
		"film.csv": func(record []string) []string {
			// Remove the last column, the fulltext column (which only Postgres
			// has).
			return record[:len(record)-1]
		},
	})
}

func TestMySQL(t *testing.T) {
	const (
		dialect = "mysql"
		driver  = "mysql"
	)
	if *mysqlDSN == "" {
		return
	}
	db, err := sql.Open(driver, *mysqlDSN)
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	dbi := NewDatabaseIntrospector(dialect, db)
	version, err := dbi.GetVersion()
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	if strings.Contains(version, "MariaDB") {
		t.Skip("skipping integration tests for MariaDB because it doesn't support indexed expressions (which are present in the mysql migration scripts)")
	}
	t.Parallel()
	testMigrateIntrospect(t, dialect, driver, *mysqlDSN)
	testAutomigrate(t, dialect, *mysqlDSN,
		"testdata/mysql_add",
		"testdata/mysql_alter",
		"testdata/mysql_drop",
		"testdata/mysql_schema",
		"testdata/mysql_table",
		"testdata/mysql_ignore",
	)
	testLoadDump(t, dialect, *mysqlDSN, map[string]func([]string) []string{
		"film.csv": func(record []string) []string {
			// MySQL inserts a trailing space after each comma in a JSON array,
			// we unmarshal and marshal the JSOn string using Go's json library
			// to remove those trailing spaces.
			// [1, 2, 3, 4] => [1,2,3,4]
			var s []string
			err := json.Unmarshal([]byte(record[11]), &s)
			if err != nil {
				return record
			}
			b, err := json.Marshal(s)
			if err != nil {
				return record
			}
			record[11] = string(b)
			return record
		},
	})
}

func TestSQLServer(t *testing.T) {
	const (
		dialect = "sqlserver"
		driver  = "sqlserver"
	)
	if *sqlserverDSN == "" {
		return
	}
	t.Parallel()
	testMigrateIntrospect(t, dialect, driver, *sqlserverDSN)
	testAutomigrate(t, dialect, *sqlserverDSN,
		"testdata/sqlserver_add",
		"testdata/sqlserver_alter",
		"testdata/sqlserver_drop",
		"testdata/sqlserver_schema",
		"testdata/sqlserver_table",
		"testdata/sqlserver_ignore",
	)
	testLoadDump(t, dialect, *sqlserverDSN, map[string]func([]string) []string{
		"language.csv": func(record []string) []string {
			record[1] = strings.TrimSpace(record[1])
			return record
		},
	})
}

func testMigrateIntrospect(t *testing.T, dialect string, driver string, dsn string) {
	wipeCmd, err := WipeCommand("-db", dsn)
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	err = wipeCmd.Run()
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}

	migrateCmd, err := MigrateCommand("-db", dsn, "-dir", dialect+"_migrations")
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	migrateCmd.Stderr = io.Discard
	err = migrateCmd.Run()
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}

	wantCatalog := &Catalog{}
	b, err := os.ReadFile(filepath.Join("testdata", dialect, "schema.json"))
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	err = json.Unmarshal(b, &wantCatalog)
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	gotCatalog := &Catalog{}
	db, err := sql.Open(driver, dsn)
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	dbi := NewDatabaseIntrospector(dialect, db)
	dbi.ExcludeTables = []string{"sqddl_history"}
	err = dbi.WriteCatalog(gotCatalog)
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	// Don't compare VersionNums.
	gotCatalog.VersionNums, wantCatalog.VersionNums = nil, nil
	// Don't compare DefaultCollation.
	gotCatalog.DefaultCollation, wantCatalog.DefaultCollation = "", ""
	if diff := testutil.Diff(gotCatalog, wantCatalog); diff != "" {
		t.Error(testutil.Callers(), diff)
	}
}

func testAutomigrate(t *testing.T, dialect string, dsn string, dirs ...string) {
	for _, dir := range dirs {
		dir := dir
		t.Run(dir, func(t *testing.T) {
			wipeCmd, err := WipeCommand("-db", dsn)
			if err != nil {
				t.Fatal(testutil.Callers(), err)
			}
			err = wipeCmd.Run()
			if err != nil {
				t.Fatal(testutil.Callers(), err)
			}

			// First do a -dry-run and dump the commands into a buffer, then
			// run the automigrate for real. If anything goes wrong we can
			// display the commands in the buffer.
			buf := &bytes.Buffer{}
			automigrateCmd, err := AutomigrateCommand(
				"-db", dsn,
				"-dest", filepath.Join(dir, "src.go"),
				"-drop-objects",
				"-accept-warnings",
				"-dry-run",
			)
			if err != nil {
				t.Fatal(testutil.Callers(), err)
			}
			buf.Reset()
			automigrateCmd.Stdout = buf
			automigrateCmd.Stderr = io.Discard
			err = automigrateCmd.Run()
			if err != nil {
				t.Fatal(testutil.Callers(), err)
			}
			automigrateCmd, err = AutomigrateCommand(
				"-db", dsn,
				"-dest", filepath.Join(dir, "src.go"),
				"-drop-objects",
				"-accept-warnings",
			)
			if err != nil {
				t.Fatal(testutil.Callers(), err)
			}
			automigrateCmd.Stderr = io.Discard
			err = automigrateCmd.Run()
			if err != nil {
				_, _ = buf.WriteTo(os.Stdout)
				t.Fatal(testutil.Callers(), err)
			}
			writeDatabaseSchemaIntoJSONFile(t, dsn, filepath.Join(dir, "src.json"))

			automigrateCmd, err = AutomigrateCommand(
				"-db", dsn,
				"-dest", filepath.Join(dir, "dest.go"),
				"-drop-objects",
				"-accept-warnings",
				"-dry-run",
			)
			if err != nil {
				t.Fatal(testutil.Callers(), err)
			}
			buf.Reset()
			automigrateCmd.Stdout = buf
			automigrateCmd.Stderr = io.Discard
			err = automigrateCmd.Run()
			if err != nil {
				t.Fatal(testutil.Callers(), err)
			}
			automigrateCmd, err = AutomigrateCommand(
				"-db", dsn,
				"-dest", filepath.Join(dir, "dest.go"),
				"-drop-objects",
				"-accept-warnings",
			)
			if err != nil {
				t.Fatal(testutil.Callers(), err)
			}
			automigrateCmd.Stderr = io.Discard
			err = automigrateCmd.Run()
			if err != nil {
				_, _ = buf.WriteTo(os.Stdout)
				t.Fatal(testutil.Callers(), err)
			}
			writeDatabaseSchemaIntoJSONFile(t, dsn, filepath.Join(dir, "dest.json"))
		})
	}
}

func testLoadDump(t *testing.T, dialect string, dsn string, transforms map[string]func([]string) []string) {
	wipeCmd, err := WipeCommand("-db", dsn)
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	err = wipeCmd.Run()
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}

	loadCmd, err := LoadCommand(
		"-db", dsn,
		"-timestamp-as-integer",
		filepath.Join("testdata", dialect, "schema.sql"),
		"csv_testdata",
		filepath.Join("testdata", dialect, "indexes.sql"),
		filepath.Join("testdata", dialect, "constraints.sql"),
	)
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	loadCmd.Stderr = io.Discard
	err = loadCmd.Run()
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}

	tempDir := t.TempDir()
	dumpCmd, err := DumpCommand(
		"-db", dsn,
		"-output-dir", tempDir,
		"-array-as-json",
		"-uuid-as-bytes",
	)
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	dumpCmd.Stderr = io.Discard
	err = dumpCmd.Run()
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	filepairs := [][2]string{
		{filepath.Join(tempDir, "schema.sql"), filepath.Join("testdata", dialect, "schema.sql")},
		{filepath.Join(tempDir, "constraints.sql"), filepath.Join("testdata", dialect, "constraints.sql")},
		{filepath.Join(tempDir, "indexes.sql"), filepath.Join("testdata", dialect, "indexes.sql")},
	}
	assertCSVsAreIdentical(t, tempDir, "csv_testdata", filepairs, transforms)

	tempDir = t.TempDir()
	dumpCmd, err = DumpCommand(
		"-db", dsn,
		"-output-dir", tempDir,
		"-array-as-json",
		"-uuid-as-bytes",
		"-subset", "SELECT {*} FROM {film} ORDER BY film_id LIMIT 10",
		"-subset", "SELECT {*} FROM {actor}",
	)
	if dialect == "sqlserver" {
		dumpCmd.SubsetQueries = []string{
			"SELECT TOP 10 {*} FROM {film} ORDER BY film_id",
			"SELECT {*} FROM {actor}",
		}
	}
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	err = dumpCmd.Run()
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	assertCSVsAreIdentical(t, tempDir, "testdata/subset", nil, transforms)

	tempDir = t.TempDir()
	dumpCmd, err = DumpCommand(
		"-db", dsn,
		"-output-dir", tempDir,
		"-array-as-json",
		"-uuid-as-bytes",
		"-extended-subset", "SELECT {*} FROM {film} ORDER BY film_id LIMIT 10",
	)
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	if dialect == "sqlserver" {
		dumpCmd.ExtendedSubsetQueries = []string{
			"SELECT TOP 10 {*} FROM {film} ORDER BY film_id",
		}
	}
	err = dumpCmd.Run()
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	assertCSVsAreIdentical(t, tempDir, "testdata/extended_subset", nil, transforms)
}

func rewriteCSV(filename string, transform func(record []string) []string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	csvReader := csv.NewReader(file)
	csvReader.FieldsPerRecord = -1
	csvReader.LazyQuotes = true
	buf := bufpool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufpool.Put(buf)
	csvWriter := csv.NewWriter(buf)
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		record = transform(record)
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
	file.Close()
	file, err = os.OpenFile(filename, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	_, err = buf.WriteTo(file)
	if err != nil {
		return err
	}
	err = file.Close()
	if err != nil {
		return err
	}
	return nil
}

func writeDatabaseSchemaIntoJSONFile(t *testing.T, dsn string, filename string) {
	catalog := &Catalog{}
	err := writeCatalog(catalog, os.DirFS("."), "sqddl_history", dsn)
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	catalog.VersionNums = nil
	catalog.DefaultCollation = ""
	catalog.DefaultCollationValid = false
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	err = enc.Encode(catalog)
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	err = file.Close()
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
}

func init() {
	Register(Driver{
		Dialect:    "postgres",
		DriverName: "postgres",
		IsLockTimeout: func(err error) bool {
			var pqerr *pq.Error
			if !errors.As(err, &pqerr) {
				return false
			}
			return pqerr.Code == "55P03" // lock_not_available
		},
		AnnotateError: func(originalErr error, query string) error {
			var pqerr *pq.Error
			if !errors.As(originalErr, &pqerr) {
				return originalErr
			}
			line := 1
			pos, posErr := strconv.Atoi(pqerr.Position)
			if posErr == nil {
				for _, char := range query[:pos] {
					if char == '\n' {
						line++
					}
				}
			}
			if posErr != nil && pqerr.Detail == "" && pqerr.Hint == "" {
				return originalErr
			}
			var b strings.Builder
			if posErr == nil {
				b.WriteString("line " + strconv.Itoa(line) + ": ")
			}
			b.WriteString("%w")
			if pqerr.Detail != "" {
				b.WriteString("\ndetail: " + pqerr.Detail)
			}
			if pqerr.Hint != "" {
				b.WriteString("\nhint: " + pqerr.Hint)
			}
			return fmt.Errorf(b.String(), originalErr)
		},
	})
	Register(Driver{
		Dialect:    "mysql",
		DriverName: "mysql",
		IsLockTimeout: func(err error) bool {
			var mysqlerr *mysql.MySQLError
			if !errors.As(err, &mysqlerr) {
				return false
			}
			return mysqlerr.Number == 1205 // ER_LOCK_WAIT_TIMEOUT
		},
	})
	Register(Driver{
		Dialect:    "sqlserver",
		DriverName: "sqlserver",
		IsLockTimeout: func(err error) bool {
			var mssqlErr mssql.Error
			if !errors.As(err, &mssqlErr) {
				return false
			}
			return mssqlErr.Number == 1222 // LK_TIMEOUT
		},
		AnnotateError: func(originalErr error, query string) error {
			var mssqlErr mssql.Error
			if !errors.As(originalErr, &mssqlErr) {
				return originalErr
			}
			var b strings.Builder
			b.WriteString("line " + strconv.Itoa(int(mssqlErr.LineNo)) + ": %w")
			if len(mssqlErr.All) > 1 {
				for _, err := range mssqlErr.All {
					b.WriteString("\n" + err.Message)
				}
			}
			return fmt.Errorf(b.String(), originalErr)
		},
	})
}
