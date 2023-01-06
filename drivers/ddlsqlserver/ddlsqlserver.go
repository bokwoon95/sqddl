package ddlsqlserver

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/bokwoon95/sqddl/ddl"
	mssql "github.com/denisenkom/go-mssqldb"
)

// Register registers a ddl.Driver for SQL Server using
// github.com/denisenkom/go-mssqldb.
func Register() {
	ddl.Register(ddl.Driver{
		Dialect:    ddl.DialectSQLServer,
		DriverName: "sqlserver",
		IsLockTimeout: func(err error) bool {
			var mssqlErr mssql.Error
			if !errors.As(err, &mssqlErr) {
				return false
			}
			return mssqlErr.Number == 1222 // LK_TIMEOUT
		},
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
