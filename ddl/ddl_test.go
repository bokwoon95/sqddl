package ddl

import (
	"embed"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/bokwoon95/sqddl/internal/testutil"

	_ "github.com/denisenkom/go-mssqldb"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed all:sqlite_migrations testdata csv_testdata
var testFS embed.FS

func Test_generateName(t *testing.T) {
	type TT struct {
		description string
		nameType    string
		tableName   string
		columnNames []string
		wantName    string
	}

	tests := []TT{{
		description: "primary key",
		nameType:    PRIMARY_KEY, tableName: "tbl", columnNames: []string{"col"},
		wantName: "tbl_col_pkey",
	}, {
		description: "foreign key",
		nameType:    FOREIGN_KEY, tableName: "tbl", columnNames: []string{"col"},
		wantName: "tbl_col_fkey",
	}, {
		description: "unique",
		nameType:    UNIQUE, tableName: "tbl", columnNames: []string{"col 1", "col 2"},
		wantName: "tbl_col_1_col_2_key",
	}, {
		description: "check",
		nameType:    CHECK, tableName: "my tbl", columnNames: []string{"col"},
		wantName: "my_tbl_col_check",
	}, {
		description: "exclude",
		nameType:    EXCLUDE, tableName: "my tbl", columnNames: []string{"col"},
		wantName: "my_tbl_col_excl",
	}, {
		description: "index",
		nameType:    INDEX, tableName: "tbl", columnNames: []string{"col1", "col2"},
		wantName: "tbl_col1_col2_idx",
	}, {
		description: "name truncated",
		nameType:    PRIMARY_KEY,
		tableName:   "pm_url_role_capability",
		columnNames: []string{"site_id", "urlpath", "plugin", "role", "capability"},
		wantName:    "pm_url_role_capability_site_id_urlpath_plugin_role_capabil_pkey",
	}}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.description, func(t *testing.T) {
			gotName := GenerateName(tt.nameType, tt.tableName, tt.columnNames)
			if diff := testutil.Diff(gotName, tt.wantName); diff != "" {
				t.Error(testutil.Callers(), diff)
			}
		})
	}
}

func Test_isLiteral_wrappedInBrackets(t *testing.T) {
	type TT struct {
		description string
		s           string
		isTrue      bool
	}

	isLiteralTests := []TT{{
		description: "empty", s: "", isTrue: false,
	}, {
		description: "string", s: "'lorem ipsum'", isTrue: true,
	}, {
		description: "known literal", s: "current_timestamp", isTrue: true,
	}, {
		description: "int", s: "123", isTrue: true,
	}, {
		description: "float", s: "3.14", isTrue: true,
	}, {
		description: "expression", s: "strftime('%s', 'now')", isTrue: false,
	}}

	for _, tt := range isLiteralTests {
		tt := tt
		t.Run(tt.description, func(t *testing.T) {
			isTrue := isLiteral(tt.s)
			if diff := testutil.Diff(isTrue, tt.isTrue); diff != "" {
				t.Error(testutil.Callers(), diff)
			}
		})
	}

	wrappedInBracketsTests := []TT{{
		description: "empty", s: "", isTrue: false,
	}, {
		description: "brackets", s: "(strftime('%s', 'now'))", isTrue: true,
	}, {
		description: "no brackets", s: "1 + 1", isTrue: false,
	}, {
		description: "partial brackets", s: "strftime('%s', 'now')", isTrue: false,
	}}

	for _, tt := range wrappedInBracketsTests {
		tt := tt
		t.Run(tt.description, func(t *testing.T) {
			isTrue := wrappedInBrackets(tt.s)
			if diff := testutil.Diff(isTrue, tt.isTrue); diff != "" {
				t.Error(testutil.Callers(), diff)
			}
		})
	}
}

func Test_wrapBrackets_unwrapBrackets(t *testing.T) {
	type TT struct {
		description string
		s           string
		wantResult  string
	}

	wrapBracketsTests := []TT{{
		description: "empty", s: "", wantResult: "",
	}, {
		description: "no brackets", s: "lorem ipsum", wantResult: "(lorem ipsum)",
	}, {
		description: "has brackets", s: "(lorem ipsum)", wantResult: "(lorem ipsum)",
	}}

	for _, tt := range wrapBracketsTests {
		tt := tt
		t.Run(tt.description, func(t *testing.T) {
			gotResult := wrapBrackets(tt.s)
			if diff := testutil.Diff(gotResult, tt.wantResult); diff != "" {
				t.Error(testutil.Callers(), diff)
			}
		})
	}

	unwrapBracketsTests := []TT{{
		description: "empty", s: "", wantResult: "",
	}, {
		description: "has brackets", s: "(lorem ipsum)", wantResult: "lorem ipsum",
	}, {
		description: "no brackets", s: "lorem ipsum", wantResult: "lorem ipsum",
	}}

	for _, tt := range unwrapBracketsTests {
		tt := tt
		t.Run(tt.description, func(t *testing.T) {
			gotResult := unwrapBrackets(tt.s)
			if diff := testutil.Diff(gotResult, tt.wantResult); diff != "" {
				t.Error(testutil.Callers(), diff)
			}
		})
	}
}

func Test_splitArgs(t *testing.T) {
	type TT struct {
		description string
		s           string
		wantArgs    []string
	}

	tests := []TT{{
		description: "empty", s: "", wantArgs: nil,
	}, {
		description: "one item", s: "lorem_ipsum", wantArgs: []string{"lorem_ipsum"},
	}, {
		description: "simple", s: "tom, dick, harry", wantArgs: []string{"tom", " dick", " harry"},
	}, {
		description: "string", s: "abcde, '{ab,cd,e}', 'I''m a bee'", wantArgs: []string{"abcde", " '{ab,cd,e}'", " 'I''m a bee'"},
	}, {
		description: "function", s: "lorem, json_extract(data, '$,a,b,c'), ipsum", wantArgs: []string{"lorem", " json_extract(data, '$,a,b,c')", " ipsum"},
	}, {
		description: "array", s: "lorem, ARRAY[1, 2, ARRAY[3, 4, 5]], ipsum", wantArgs: []string{"lorem", " ARRAY[1, 2, ARRAY[3, 4, 5]]", " ipsum"},
	}}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.description, func(t *testing.T) {
			gotArgs := splitArgs(tt.s)
			if diff := testutil.Diff(gotArgs, tt.wantArgs); diff != "" {
				t.Error(testutil.Callers(), diff)
			}
		})
	}
}

func Test_normalizeColumnType(t *testing.T) {
	type TT struct {
		inputs   []string
		wantType string
		wantArg1 string
		wantArg2 string
	}

	runTests := func(dialect string, tests []TT) {
		for _, tt := range tests {
			tt := tt
			for _, input := range tt.inputs {
				input := input
				description := dialect + " " + input
				t.Run(description, func(t *testing.T) {
					gotNormalizedType, gotArg1, gotArg2 := normalizeColumnType(dialect, input)
					if diff := testutil.Diff(gotNormalizedType, tt.wantType); diff != "" {
						t.Error(testutil.Callers(), diff)
					}
					if diff := testutil.Diff(gotArg1, tt.wantArg1); diff != "" {
						t.Error(testutil.Callers(), diff)
					}
					if diff := testutil.Diff(gotArg2, tt.wantArg2); diff != "" {
						t.Error(testutil.Callers(), diff)
					}
				})
			}
		}
	}

	// SQLite
	runTests(DialectSQLite, []TT{{
		inputs:   []string{"lorem ipsum"},
		wantType: "LOREM IPSUM",
	}, {
		inputs:   []string{"NUMERIC(5,2)", "numeric (5, 2)"},
		wantType: "NUMERIC", wantArg1: "5", wantArg2: "2",
	}, {
		inputs:   []string{"VARCHAR(255)", "varchar    (255)"},
		wantType: "VARCHAR", wantArg1: "255",
	}})

	// Postgres
	runTests(DialectPostgres, []TT{{
		inputs:   []string{"integer", "serial", "serial4", "int4", "int"},
		wantType: "INT",
	}, {
		inputs:   []string{"bigserial", "serial8", "int8", "bigint"},
		wantType: "BIGINT",
	}, {
		inputs:   []string{"smallserial", "serial2", "int2", "smallint"},
		wantType: "SMALLINT",
	}, {
		inputs:   []string{"decimal(5,2)", "numeric(5,2)"},
		wantType: "NUMERIC", wantArg1: "5", wantArg2: "2",
	}, {
		inputs:   []string{"float4", "real"},
		wantType: "REAL",
	}, {
		inputs:   []string{"float8", "double precision"},
		wantType: "DOUBLE PRECISION",
	}, {
		inputs:   []string{"character varying (255)", "varchar(255)"},
		wantType: "VARCHAR", wantArg1: "255",
	}, {
		inputs:   []string{"character (255)", "char(255)"},
		wantType: "CHAR", wantArg1: "255",
	}, {
		inputs:   []string{"timestamp with time zone", "timestamptz"},
		wantType: "TIMESTAMPTZ",
	}, {
		inputs:   []string{"timestamp (3) with time zone", "timestamptz(3) "},
		wantType: "TIMESTAMPTZ", wantArg1: "3",
	}, {
		inputs:   []string{"timestamp without time zone", "timestamp"},
		wantType: "TIMESTAMP",
	}, {
		inputs:   []string{"timestamp (3) without time zone", "timestamp(3)"},
		wantType: "TIMESTAMP", wantArg1: "3",
	}, {
		inputs:   []string{"time with time zone", "timetz"},
		wantType: "TIMETZ",
	}, {
		inputs:   []string{"time (3) with time zone", "timetz(3)"},
		wantType: "TIMETZ", wantArg1: "3",
	}, {
		inputs:   []string{"time without time zone", "time"},
		wantType: "TIME",
	}, {
		inputs:   []string{"time (3) without time zone", "time(3)"},
		wantType: "TIME", wantArg1: "3",
	}, {
		inputs:   []string{"bit varying", "varbit"},
		wantType: "VARBIT",
	}, {
		inputs:   []string{"bit varying (255)", "varbit(255)"},
		wantType: "VARBIT", wantArg1: "255",
	}, {
		inputs:   []string{"bool", "boolean"},
		wantType: "BOOLEAN",
	}, {
		inputs: []string{"float"}, wantType: "FLOAT",
	}, {
		inputs: []string{"float(1)"}, wantType: "FLOAT", wantArg1: "1",
	}, {
		inputs: []string{"text"}, wantType: "TEXT",
	}, {
		inputs: []string{"text[]"}, wantType: "TEXT[]",
	}, {
		inputs: []string{"bytea"}, wantType: "BYTEA",
	}, {
		inputs: []string{"uuid"}, wantType: "UUID",
	}, {
		inputs: []string{"json"}, wantType: "JSON",
	}, {
		inputs: []string{"jsonb"}, wantType: "JSONB",
	}, {
		inputs: []string{"date"}, wantType: "DATE",
	}})

	// MySQL
	runTests(DialectMySQL, []TT{{
		inputs:   []string{"integer", "int"},
		wantType: "INT",
	}, {
		inputs:   []string{"integer unsigned", "int unsigned"},
		wantType: "INT UNSIGNED",
	}, {
		inputs:   []string{"integer signed", "int signed"},
		wantType: "INT",
	}, {
		inputs: []string{"tinyint signed"}, wantType: "TINYINT",
	}, {
		inputs: []string{"tinyint unsigned"}, wantType: "TINYINT UNSIGNED",
	}, {
		inputs: []string{"smallint signed"}, wantType: "SMALLINT",
	}, {
		inputs: []string{"smallint unsigned"}, wantType: "SMALLINT UNSIGNED",
	}, {
		inputs: []string{"mediumint signed"}, wantType: "MEDIUMINT",
	}, {
		inputs: []string{"mediumint unsigned"}, wantType: "MEDIUMINT UNSIGNED",
	}, {
		inputs: []string{"bigint signed"}, wantType: "BIGINT",
	}, {
		inputs: []string{"bigint unsigned"}, wantType: "BIGINT UNSIGNED",
	}, {
		inputs:   []string{"dec(5,2)", "decimal(5,2)", "numeric(5, 2)"},
		wantType: "NUMERIC", wantArg1: "5", wantArg2: "2",
	}, {
		inputs:   []string{"bool", "boolean", "tinyint(1)"},
		wantType: "TINYINT", wantArg1: "1",
	}, {
		inputs: []string{"binary(16)"}, wantType: "BINARY", wantArg1: "16",
	}, {
		inputs: []string{"tinyblob"}, wantType: "TINYBLOB",
	}, {
		inputs: []string{"blob"}, wantType: "BLOB",
	}, {
		inputs: []string{"mediumblob"}, wantType: "MEDIUMBLOB",
	}, {
		inputs: []string{"longblob"}, wantType: "LONGBLOB",
	}, {
		inputs: []string{"char(255)"}, wantType: "CHAR", wantArg1: "255",
	}, {
		inputs: []string{"varchar(255)"}, wantType: "VARCHAR", wantArg1: "255",
	}, {
		inputs: []string{"tinytext"}, wantType: "TINYTEXT",
	}, {
		inputs: []string{"text"}, wantType: "TEXT",
	}, {
		inputs: []string{"mediumtext"}, wantType: "MEDIUMTEXT",
	}, {
		inputs: []string{"longtext"}, wantType: "LONGTEXT",
	}, {
		inputs: []string{"json"}, wantType: "JSON",
	}, {
		inputs: []string{"date"}, wantType: "DATE",
	}, {
		inputs: []string{"time"}, wantType: "TIME",
	}, {
		inputs: []string{"datetime"}, wantType: "DATETIME",
	}, {
		inputs: []string{"timestamp"}, wantType: "TIMESTAMP",
	}})

	// SQLServer
	runTests(DialectSQLServer, []TT{{
		inputs:   []string{"binary varying (16)", "varbinary(16)"},
		wantType: "VARBINARY", wantArg1: "16",
	}, {
		inputs:   []string{"integer", "int"},
		wantType: "INT",
	}, {
		inputs:   []string{"national character varying (255)", "nvarchar(255)"},
		wantType: "NVARCHAR", wantArg1: "255",
	}, {
		inputs:   []string{"national character varying (MAX)", "nvarchar(MAX)"},
		wantType: "NVARCHAR", wantArg1: "MAX",
	}, {
		inputs:   []string{"character varying (255)", "varchar(255)"},
		wantType: "VARCHAR", wantArg1: "255",
	}, {
		inputs:   []string{"character (255)", "char(255)"},
		wantType: "CHAR", wantArg1: "255",
	}, {
		inputs:   []string{"dec(5,2)", "decimal(5,2)", "numeric(5, 2)"},
		wantType: "NUMERIC", wantArg1: "5", wantArg2: "2",
	}, {
		inputs: []string{"tinyint"}, wantType: "TINYINT",
	}, {
		inputs: []string{"smallint"}, wantType: "SMALLINT",
	}, {
		inputs: []string{"bit"}, wantType: "BIT",
	}, {
		inputs: []string{"float"}, wantType: "FLOAT",
	}, {
		inputs: []string{"real"}, wantType: "REAL",
	}, {
		inputs: []string{"date"}, wantType: "DATE",
	}, {
		inputs: []string{"datetime"}, wantType: "DATETIME",
	}, {
		inputs: []string{"datetime2"}, wantType: "DATETIME2",
	}, {
		inputs: []string{"datetime2(1)"}, wantType: "DATETIME2", wantArg1: "1",
	}, {
		inputs: []string{"datetimeoffset"}, wantType: "DATETIMEOFFSET",
	}, {
		inputs: []string{"datetimeoffset(1)"}, wantType: "DATETIMEOFFSET", wantArg1: "1",
	}, {
		inputs: []string{"time"}, wantType: "TIME",
	}, {
		inputs: []string{"time(1)"}, wantType: "TIME", wantArg1: "1",
	}, {
		inputs: []string{"binary(16)"}, wantType: "BINARY", wantArg1: "16",
	}, {
		inputs: []string{"uniqueidentifier"}, wantType: "UNIQUEIDENTIFIER",
	}})
}

func Test_normalizeColumnDefault(t *testing.T) {
	type TT struct {
		inputs      []string
		wantDefault string
	}

	runTests := func(dialect string, tests []TT) {
		for _, tt := range tests {
			tt := tt
			for _, input := range tt.inputs {
				input := input
				description := dialect + " " + input
				t.Run(description, func(t *testing.T) {
					gotDefault := normalizeColumnDefault(dialect, input)
					if diff := testutil.Diff(tt.wantDefault, gotDefault); diff != "" {
						t.Error(testutil.Callers(), diff)
					}
				})
			}
		}
	}

	// SQLite
	runTests(DialectSQLite, []TT{{
		inputs:      []string{"1", "'1'", "true"},
		wantDefault: "'1'",
	}, {
		inputs:      []string{"0", "'0'", "false"},
		wantDefault: "'0'",
	}, {
		inputs:      []string{"datetime()", "datetime('now')", "current_timestamp"},
		wantDefault: "CURRENT_TIMESTAMP",
	}})

	// Postgres
	runTests(DialectPostgres, []TT{{
		inputs:      []string{"1", "'1'", "true"},
		wantDefault: "'1'",
	}, {
		inputs:      []string{"0", "'0'", "false"},
		wantDefault: "'0'",
	}, {
		inputs:      []string{"now()", "current_timestamp"},
		wantDefault: "CURRENT_TIMESTAMP",
	}, {
		inputs:      []string{"'G'", "'G'::mpaa_rating"},
		wantDefault: "'G'",
	}})

	// MySQL
	runTests(DialectMySQL, []TT{{
		inputs:      []string{"1", "'1'", "true"},
		wantDefault: "'1'",
	}, {
		inputs:      []string{"0", "'0'", "false"},
		wantDefault: "'0'",
	}, {
		inputs:      []string{"now()", "current_timestamp"},
		wantDefault: "CURRENT_TIMESTAMP",
	}})

	// SQLServer
	runTests(DialectSQLServer, []TT{{
		inputs:      []string{"1", "'1'", "true"},
		wantDefault: "'1'",
	}, {
		inputs:      []string{"0", "'0'", "false"},
		wantDefault: "'0'",
	}, {
		inputs:      []string{"getdate()", "current_timestamp"},
		wantDefault: "CURRENT_TIMESTAMP",
	}})
}

func TestNormalizeDSN(t *testing.T) {
	type TT struct {
		dsn               string
		wantDialect       string
		wantDriverName    string
		wantNormalizedDSN string
	}

	// Wipe all drivers first in order to test normalizeDSN without drivers.
	// Restore the drivers later with a defer.
	oldDrivers := drivers
	drivers = make(map[string]Driver)
	defer func() { drivers = oldDrivers }()

	driverlessTests := []TT{{
		dsn:         "file:nonexistent_file", // <-- file doesn't exist and doesn't have .{sqlite,sqlite3,db,db3} suffix
		wantDialect: "", wantDriverName: "",
		wantNormalizedDSN: "",
	}, {
		dsn:         "file:nonexistent_file.db", // <-- file doesn't exist and has .{sqlite,sqlite3,db,db3} suffix
		wantDialect: DialectSQLite, wantDriverName: "sqlite3",
		wantNormalizedDSN: "file:nonexistent_file.db",
	}, {
		dsn:         "file:testdata/sqlite_db", // <-- file is an SQLite database.
		wantDialect: DialectSQLite, wantDriverName: "sqlite3",
		wantNormalizedDSN: "file:testdata/sqlite_db",
	}, {
		dsn:         "file:testdata/database_url.txt", // <-- file contains a Postgres DSN.
		wantDialect: DialectPostgres, wantDriverName: "postgres",
		wantNormalizedDSN: "postgres://user1:Hunter2!@localhost:5456/sakila?sslmode=disable",
	}, {
		dsn:         "sqlite:has_sqlite_prefix",
		wantDialect: DialectSQLite, wantDriverName: "sqlite3",
		wantNormalizedDSN: "has_sqlite_prefix",
	}, {
		dsn:         "sqlite3:has_sqlite3_prefix",
		wantDialect: DialectSQLite, wantDriverName: "sqlite3",
		wantNormalizedDSN: "has_sqlite3_prefix",
	}, {
		dsn:         "postgres://user1:Hunter2!@localhost:5456/sakila",
		wantDialect: DialectPostgres, wantDriverName: "postgres",
		wantNormalizedDSN: "postgres://user1:Hunter2!@localhost:5456/sakila",
	}, {
		dsn:         "mysql://root:Hunter2!@tcp(localhost:3330)/sakila?multiStatements=true&parseTime=true",
		wantDialect: DialectMySQL, wantDriverName: "mysql",
		wantNormalizedDSN: "root:Hunter2!@tcp(localhost:3330)/sakila?multiStatements=true&parseTime=true",
	}, {
		dsn:         "sqlserver://sa:Hunter2!@localhost:1447",
		wantDialect: DialectSQLServer, wantDriverName: "sqlserver",
		wantNormalizedDSN: "sqlserver://sa:Hunter2!@localhost:1447",
	}, {
		dsn:         "root:Hunter2!@tcp(localhost:3330)/sakila?multiStatements=true&parseTime=true",
		wantDialect: DialectMySQL, wantDriverName: "mysql",
		wantNormalizedDSN: "root:Hunter2!@tcp(localhost:3330)/sakila?multiStatements=true&parseTime=true",
	}, {
		dsn:         "test.sqlite",
		wantDialect: DialectSQLite, wantDriverName: "sqlite3",
		wantNormalizedDSN: "test.sqlite",
	}, {
		dsn:         "test.sqlite3",
		wantDialect: DialectSQLite, wantDriverName: "sqlite3",
		wantNormalizedDSN: "test.sqlite3",
	}, {
		dsn:         "test.db",
		wantDialect: DialectSQLite, wantDriverName: "sqlite3",
		wantNormalizedDSN: "test.db",
	}, {
		dsn:         "test.db3",
		wantDialect: DialectSQLite, wantDriverName: "sqlite3",
		wantNormalizedDSN: "test.db3",
	}}

	for _, tt := range driverlessTests {
		tt := tt
		t.Run(tt.dsn, func(t *testing.T) {
			gotDialect, gotDriverName, gotNormalizedDSN := NormalizeDSN(tt.dsn)
			if diff := testutil.Diff(gotDialect, tt.wantDialect); diff != "" {
				t.Error(testutil.Callers(), diff)
			}
			if diff := testutil.Diff(gotDriverName, tt.wantDriverName); diff != "" {
				t.Error(testutil.Callers(), diff)
			}
			if diff := testutil.Diff(gotNormalizedDSN, tt.wantNormalizedDSN); diff != "" {
				t.Error(testutil.Callers(), diff)
			}
		})
	}

	// Register the drivers.
	Register(Driver{
		Dialect:    DialectSQLite,
		DriverName: "sqlite3",
		PreprocessDSN: func(dsn string) string {
			dsn = strings.TrimPrefix(strings.TrimPrefix(dsn, "sqlite:"), "//")
			before, after, _ := strings.Cut(dsn, "?")
			q, err := url.ParseQuery(after)
			if err != nil {
				return dsn
			}
			if !q.Has("_foreign_keys") && !q.Has("_fk") {
				q.Set("_foreign_keys", "true")
			}
			return before + "?" + q.Encode()
		},
	})
	Register(Driver{
		Dialect:    DialectPostgres,
		DriverName: "postgres",
		PreprocessDSN: func(dsn string) string {
			before, after, _ := strings.Cut(dsn, "?")
			q, err := url.ParseQuery(after)
			if err != nil {
				return dsn
			}
			if !q.Has("sslmode") {
				q.Set("sslmode", "disable")
			}
			return before + "?" + q.Encode()
		},
	})
	Register(Driver{
		Dialect:    DialectMySQL,
		DriverName: "mysql",
		PreprocessDSN: func(dsn string) string {
			if strings.HasPrefix(dsn, "mysql://") {
				u, err := url.Parse(dsn)
				if err != nil {
					dsn = strings.TrimPrefix(dsn, "mysql://")
				} else {
					var b strings.Builder
					b.Grow(len(dsn))
					if u.User != nil {
						username := u.User.Username()
						password, ok := u.User.Password()
						b.WriteString(username)
						if ok {
							b.WriteString(":" + password)
						}
					}
					if u.Host != "" {
						if b.Len() > 0 {
							b.WriteString("@")
						}
						b.WriteString("tcp(" + u.Host + ")")
					}
					b.WriteString("/" + strings.TrimPrefix(u.Path, "/"))
					if u.RawQuery != "" {
						b.WriteString("?" + u.RawQuery)
					}
					dsn = b.String()
				}
			}
			before, after, _ := strings.Cut(dsn, "?")
			q, err := url.ParseQuery(after)
			if err != nil {
				return dsn
			}
			if !q.Has("allowAllFiles") {
				q.Set("allowAllFiles", "true")
			}
			if !q.Has("multiStatements") {
				q.Set("multiStatements", "true")
			}
			if !q.Has("parseTime") {
				q.Set("parseTime", "true")
			}
			return before + "?" + q.Encode()
		},
	})
	Register(Driver{
		Dialect:    DialectSQLServer,
		DriverName: "sqlserver",
		PreprocessDSN: func(dsn string) string {
			u, err := url.Parse(dsn)
			if err != nil {
				return dsn
			}
			if u.Path != "" {
				before, after, _ := strings.Cut(dsn, "?")
				q, err := url.ParseQuery(after)
				if err != nil {
					return dsn
				}
				q.Set("database", u.Path[1:])
				dsn = strings.TrimSuffix(before, u.Path) + "?" + q.Encode()
			}
			return dsn
		},
	})

	driverTests := []TT{{
		dsn:         "test.db",
		wantDialect: DialectSQLite, wantDriverName: "sqlite3",
		wantNormalizedDSN: "test.db?_foreign_keys=true",
	}, {
		dsn:         "test.db?_fk=0",
		wantDialect: DialectSQLite, wantDriverName: "sqlite3",
		wantNormalizedDSN: "test.db?_fk=0",
	}, {
		dsn:         "postgres://user1:Hunter2!@localhost:5456/sakila",
		wantDialect: DialectPostgres, wantDriverName: "postgres",
		wantNormalizedDSN: "postgres://user1:Hunter2!@localhost:5456/sakila?sslmode=disable",
	}, {
		dsn:         "postgres://user1:Hunter2!@localhost:5456/sakila?sslmode=verify-full",
		wantDialect: DialectPostgres, wantDriverName: "postgres",
		wantNormalizedDSN: "postgres://user1:Hunter2!@localhost:5456/sakila?sslmode=verify-full",
	}, {
		dsn:         "mysql://root:Hunter2!@localhost:3330/sakila",
		wantDialect: DialectMySQL, wantDriverName: "mysql",
		wantNormalizedDSN: "root:Hunter2!@tcp(localhost:3330)/sakila?allowAllFiles=true&multiStatements=true&parseTime=true",
	}, {
		dsn:         "root:Hunter2!@unix(/tmp/mysql.sock)/sakila?allowAllFiles=false&multiStatements=false&parseTime=false",
		wantDialect: DialectMySQL, wantDriverName: "mysql",
		wantNormalizedDSN: "root:Hunter2!@unix(/tmp/mysql.sock)/sakila?allowAllFiles=false&multiStatements=false&parseTime=false",
	}, {
		dsn:         "sqlserver://sa:Hunter2!@localhost:1447/sakila",
		wantDialect: DialectSQLServer, wantDriverName: "sqlserver",
		wantNormalizedDSN: "sqlserver://sa:Hunter2!@localhost:1447?database=sakila",
	}, {
		dsn:         "sqlserver://sa:Hunter2!@localhost:1447?database=sakila",
		wantDialect: DialectSQLServer, wantDriverName: "sqlserver",
		wantNormalizedDSN: "sqlserver://sa:Hunter2!@localhost:1447?database=sakila",
	}}

	for _, tt := range driverTests {
		tt := tt
		t.Run(tt.dsn, func(t *testing.T) {
			gotDialect, gotDriverName, gotNormalizedDSN := NormalizeDSN(tt.dsn)
			if diff := testutil.Diff(gotDialect, tt.wantDialect); diff != "" {
				t.Error(testutil.Callers(), diff)
			}
			if diff := testutil.Diff(gotDriverName, tt.wantDriverName); diff != "" {
				t.Error(testutil.Callers(), diff)
			}
			if diff := testutil.Diff(gotNormalizedDSN, tt.wantNormalizedDSN); diff != "" {
				t.Error(testutil.Callers(), diff)
			}
		})
	}

	Register(Driver{Dialect: DialectSQLite, DriverName: "sqlite3"})
	gotDialect, gotDriverName, gotNormalizedDSN := NormalizeDSN("sqlite://abcdefg")
	if diff := testutil.Diff(gotDialect, "sqlite"); diff != "" {
		t.Error(testutil.Callers(), diff)
	}
	if diff := testutil.Diff(gotDriverName, "sqlite3"); diff != "" {
		t.Error(testutil.Callers(), diff)
	}
	if diff := testutil.Diff(gotNormalizedDSN, "sqlite://abcdefg"); diff != "" {
		t.Error(testutil.Callers(), diff)
	}
}

func TestVersionNums(t *testing.T) {
	type TT struct {
		v1   VersionNums
		v2   VersionNums
		want bool
	}

	tests := []TT{
		{[]int{14, 2}, []int{}, false},
		{[]int{}, []int{14, 2}, false},
		{[]int{7, 1, 1}, []int{7, 1}, false},
		{[]int{10, 3}, []int{10, 2}, false},
		{[]int{10, 2}, []int{10, 2}, false},
		{[]int{10, 2}, []int{10, 3}, true},
		{[]int{9, 10, 11}, []int{11, 10, 9}, true},
	}
	for i, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("test #%d", i+1), func(t *testing.T) {
			got := tt.v1.LowerThan(tt.v2...)
			if diff := testutil.Diff(got, tt.want); diff != "" {
				t.Error(testutil.Callers(), diff)
			}
			got = tt.v1.GreaterOrEqualTo(tt.v2...)
			if diff := testutil.Diff(got, !tt.want); diff != "" {
				t.Error(testutil.Callers(), diff)
			}
		})
	}
}
