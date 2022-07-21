package ddlmysql

import (
	"errors"
	"net/url"
	"strings"

	"github.com/bokwoon95/sq"
	"github.com/bokwoon95/sqddl/ddl"
	"github.com/go-sql-driver/mysql"
)

// Register registers a ddl.Driver for MySQL using
// github.com/go-sql-driver/mysql.
func Register() {
	ddl.Register(ddl.Driver{
		Dialect:    sq.DialectMySQL,
		DriverName: "mysql",
		IsLockTimeout: func(err error) bool {
			var mysqlerr *mysql.MySQLError
			if !errors.As(err, &mysqlerr) {
				return false
			}
			return mysqlerr.Number == 1205 // ER_LOCK_WAIT_TIMEOUT
		},
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
}
