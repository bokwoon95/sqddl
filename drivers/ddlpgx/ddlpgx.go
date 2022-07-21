package ddlpgx

import (
	"errors"
	"net/url"
	"strings"

	"github.com/bokwoon95/sq"
	"github.com/bokwoon95/sqddl/ddl"
	"github.com/jackc/pgconn"
	_ "github.com/jackc/pgx/v4/stdlib"
)

// Register registers a ddl.Driver for Postgres using
// github.com/jackc/pgx/v4/stdlib.
func Register() {
	ddl.Register(ddl.Driver{
		Dialect:    sq.DialectPostgres,
		DriverName: "pgx",
		IsLockTimeout: func(err error) bool {
			var pgerr *pgconn.PgError
			if !errors.As(err, &pgerr) {
				return false
			}
			return pgerr.Code == "55P03" // lock_not_available
		},
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
}
