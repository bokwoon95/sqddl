package ddlsqlite3

import (
	"net/url"
	"strings"

	"github.com/bokwoon95/sq"
	"github.com/bokwoon95/sqddl/ddl"
	_ "github.com/mattn/go-sqlite3"
)

// Register registers a ddl.Driver for SQLite using
// github.com/mattn/go-sqlite3.
func Register() {
	ddl.Register(ddl.Driver{
		Dialect:    sq.DialectSQLite,
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
}
