package ddloracle

import (
	"errors"

	"github.com/bokwoon95/sqddl/ddl"
	_ "github.com/sijms/go-ora/v2"
	"github.com/sijms/go-ora/v2/network"
)

func Register() {
	ddl.Register(ddl.Driver{
		Dialect:    "oracle",
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
