package ddloracle

import (
	"errors"

	"github.com/bokwoon95/sqddl/ddl"
	_ "github.com/sijms/go-ora/v2"
	"github.com/sijms/go-ora/v2/network"
)

// Register registers a ddl.Driver for Oracle using github.com/sijms/go-ora.
func Register() {
	ddl.Register(ddl.Driver{
		Dialect:    ddl.DialectOracle,
		DriverName: "oracle",
		IsLockTimeout: func(err error) bool {
			var oracleErr *network.OracleError
			if !errors.As(err, &oracleErr) {
				return false
			}
			return oracleErr.ErrCode == 4021 // ORA-04021: TIMEOUT OCCURRED WHILE WAITING TO LOCK OBJECT
		},
		PreprocessDSN: func(dsn string) string {
			return dsn
		},
	})
}
