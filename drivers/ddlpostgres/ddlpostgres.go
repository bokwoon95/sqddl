package ddlpostgres

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/bokwoon95/sq"
	"github.com/bokwoon95/sqddl/ddl"
	"github.com/lib/pq"
)

// Register registers a ddl.Driver for Postgres using
// github.com/lib/pq.
func Register() {
	ddl.Register(ddl.Driver{
		Dialect:    sq.DialectPostgres,
		DriverName: "postgres",
		IsLockTimeout: func(err error) bool {
			var pqerr *pq.Error
			if !errors.As(err, &pqerr) {
				return false
			}
			return pqerr.Code == "55P03" // lock_not_available
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
		AnnotateError: func(originalErr error, query string) error {
			var pqerr *pq.Error
			if !errors.As(originalErr, &pqerr) {
				return originalErr
			}
			line := 1
			pos, posErr := strconv.Atoi(pqerr.Position)
			if posErr == nil {
				line = strings.Count(query[:pos], "\n") + 1
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
}
