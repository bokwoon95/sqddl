package ddl

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// The various keyword constants used by ddl.
const (
	PRIMARY_KEY = "PRIMARY KEY"
	FOREIGN_KEY = "FOREIGN KEY"
	UNIQUE      = "UNIQUE"
	CHECK       = "CHECK"
	EXCLUDE     = "EXCLUDE"
	INDEX       = "INDEX"

	DEFAULT_IDENTITY = "GENERATED BY DEFAULT AS IDENTITY"
	ALWAYS_IDENTITY  = "GENERATED ALWAYS AS IDENTITY"
	IDENTITY         = "IDENTITY"

	RESTRICT    = "RESTRICT"
	CASCADE     = "CASCADE"
	NO_ACTION   = "NO ACTION"
	SET_NULL    = "SET NULL"
	SET_DEFAULT = "SET DEFAULT"
)

// The dialects supported by ddl.
const (
	DialectSQLite    = "sqlite"
	DialectPostgres  = "postgres"
	DialectMySQL     = "mysql"
	DialectSQLServer = "sqlserver"
	DialectOracle    = "oracle"
)

var bufpool = sync.Pool{
	New: func() any { return &bytes.Buffer{} },
}

// Catalog represents a database catalog i.e. a database instance.
type Catalog struct {
	// Dialect is the dialect of the database. Possible values: "sqlite",
	// "postgres", "mysql", "sqlserver".
	Dialect string `json:",omitempty"`

	// VersionNums holds the database's version numbers.
	//
	// Example: Postgres 14.2 would be represented as []int{14, 2}.
	VersionNums VersionNums `json:",omitempty"`

	// Database name.
	CatalogName string `json:",omitempty"`

	// CurrentSchema is the current schema of the database. For Postgres it
	// is usually "public", for MySQL this is the database name, for SQL Server
	// it is usually "dbo". It is always empty for SQLite.
	CurrentSchema string `json:",omitempty"`

	// DefaultCollation is the default collation of the database.
	DefaultCollation string `json:",omitempty"`

	// If DefaultCollationValid is false, the database's default collation is
	// unknown.
	DefaultCollationValid bool `json:",omitempty"`

	// The extensions in the database. Postgres only.
	Extensions []string `json:",omitempty"`

	// If ExtensionsValid is false, the database's extensions are unknown.
	ExtensionsValid bool `json:",omitempty"`

	// The list of schemas within the database.
	Schemas []Schema `json:",omitempty"`
}

// Schema represents a database schema.
type Schema struct {
	// SchemaName is the name of the schema.
	SchemaName string `json:",omitempty"`

	// Tables is the list of tables within the schema.
	Tables []Table `json:",omitempty"`

	// Views is the list of views within the schema.
	Views []View `json:",omitempty"`

	// If ViewsValid is false, the schema's views are unknown.
	ViewsValid bool `json:",omitempty"`

	// Routines is the list of routines (stored procedures and functions)
	// within the schema.
	Routines []Routine `json:",omitempty"`

	// If RoutinesValid is false, the schema's routines are unknown.
	RoutinesValid bool `json:",omitempty"`

	// The list of enum types within the schema. Postgres only.
	Enums []Enum `json:",omitempty"`

	// If EnumsValid is false, the schema's enum types are unknown.
	EnumsValid bool `json:",omitempty"`

	// The list of domain types within the schema. Postgres only.
	Domains []Domain `json:",omitempty"`

	// If DomainsValid is false, the schema's domain types are unknown.
	DomainsValid bool `json:",omitempty"`

	// Comment stores the comment on the schema object.
	Comment string `json:",omitempty"`

	// If Ignore is true, the schema should be treated like it doesn't exist (a
	// soft delete flag).
	Ignore bool `json:",omitempty"`
}

// Enum represents a database enum type. Postgres only.
type Enum struct {
	// EnumSchema is the name of schema that the enum type belongs to.
	EnumSchema string `json:",omitempty"`

	// EnumName is the name of the enum type.
	EnumName string `json:",omitempty"`

	// EnumLabels contains the list of labels associated with the enum type.
	EnumLabels []string `json:",omitempty"`

	// Comment stores the comment on the enum type.
	Comment string `json:",omitempty"`

	// If Ignore is true, the enum type should be treated like it doesn't
	// exist (a soft delete flag).
	Ignore bool `json:",omitempty"`
}

// Domain represents a database domain type. Postgres only.
type Domain struct {
	// DomainSchema is the name of schema that the domain type belongs to.
	DomainSchema string `json:",omitempty"`

	// DomainName is the name of the domain type.
	DomainName string `json:",omitempty"`

	// UnderlyingType is the underlying type of the domain.
	UnderlyingType string `json:",omitempty"`

	// CollationName is the collation of the domain type.
	CollationName string `json:",omitempty"`

	// IsNotNull indicates if the domain type is NOT NULL.
	IsNotNull bool `json:",omitempty"`

	// ColumnDefault is the default value of the domain type.
	ColumnDefault string `json:",omitempty"`

	// CheckNames is the list of check constraint names on the domain type.
	CheckNames []string `json:",omitempty"`

	// CheckExprs is the list of check constraints expressions on the domain
	// type.
	CheckExprs []string `json:",omitempty"`

	// Comment stores the comment on the domain type.
	Comment string `json:",omitempty"`

	// If Ignore is true, the domain type should be treated like it doesn't
	// exist (a soft delete flag).
	Ignore bool `json:",omitempty"`
}

// Routine represents a database routine (either a stored procedure or a
// function).
type Routine struct {
	// RoutineSchema is the name of schema that the routine belongs to.
	RoutineSchema string `json:",omitempty"`

	// RoutineName is the name of the routine.
	RoutineName string `json:",omitempty"`

	// IdentityArguments is a string containing the identity arguments that
	// uniquely identify routines sharing the same name. Postgres only.
	IdentityArguments string `json:",omitempty"`

	// RoutineType identifies the type of the routine. Possible values:
	// "PROCEDURE", "FUNCTION".
	RoutineType string `json:",omitempty"`

	// SQL is the SQL definition of the routine.
	SQL string `json:",omitempty"`

	// Attrs stores additional metadata about the routine.
	Attrs map[string]string `json:",omitempty"`

	// Comment stores the comment on the routine.
	Comment string `json:",omitempty"`

	// If Ignore is true, the routine should be treated like it doesn't exist
	// (a soft delete flag).
	Ignore bool `json:",omitempty"`
}

// View represents a database view.
type View struct {
	// ViewSchema is the name of schema that the view belongs to.
	ViewSchema string `json:",omitempty"`

	// ViewName is the name of the view.
	ViewName string `json:",omitempty"`

	// IsMaterialized indicates if the view is a materialized view.
	IsMaterialized bool `json:",omitempty"`

	// SQL is the SQL definition of the view.
	SQL string `json:",omitempty"`

	// Columns is the list of columns in the view.
	Columns []string `json:",omitempty"`

	// ColumnTypes is the list of column types in the view.
	ColumnTypes []string `json:",omitempty"`

	// EnumColumns is the list of columns in the view whose column type is an
	// enum.
	EnumColumns []string `json:",omitempty"`

	// Indexes is the list of indexes belonging to the view.
	Indexes []Index `json:",omitempty"`

	// Triggers is the list of triggers belonging to the view.
	Triggers []Trigger `json:",omitempty"`

	// Comment stores the comment on the view.
	Comment string `json:",omitempty"`

	// If Ignore is true, the view should be treated like it doesn't exist (a
	// soft delete flag).
	Ignore bool `json:",omitempty"`
}

// Table represents a database table.
type Table struct {
	// TableSchema is the name of schema that the table belongs to.
	TableSchema string `json:",omitempty"`

	// TableName is the name of the table.
	TableName string `json:",omitempty"`

	// SQL is the SQL definition of the table.
	SQL string `json:",omitempty"`

	// IsVirtual indicates if the table is a virtual table. SQLite only.
	IsVirtual bool `json:",omitempty"`

	// Columns is the list of columns within the table.
	Columns []Column `json:",omitempty"`

	// Constraints is the list of constraints within the table.
	Constraints []Constraint `json:",omitempty"`

	// Indexes is the list of indexes within the table.
	Indexes []Index `json:",omitempty"`

	// Triggers is the list of triggers within the table.
	Triggers []Trigger `json:",omitempty"`

	// Comment stores the comment on the table.
	Comment string `json:",omitempty"`

	// If Ignore is true, the table should be treated like it doesn't exist (a
	// soft delete flag).
	Ignore bool `json:",omitempty"`
}

// Column represents a database column.
type Column struct {
	// TableSchema is the name of the schema that the table and column belong to.
	TableSchema string `json:",omitempty"`

	// TableName is the name of the table that the column belongs to.
	TableName string `json:",omitempty"`

	// ColumnName is the name of the column.
	ColumnName string `json:",omitempty"`

	// ColumnType is the type of the column.
	ColumnType string `json:",omitempty"`

	// CharacterLength stores the character length of the column (as a string)
	// if applicable.
	CharacterLength string `json:",omitempty"`

	// NumericPrecision stores the numeric precision of the column (as a
	// string) if applicable.
	NumericPrecision string `json:",omitempty"`

	// NumericScale stores the numeric scale of the column (as a string) if
	// applicable.
	NumericScale string `json:",omitempty"`

	// DomainName stores the name of the domain if the column is a domain type.
	// In which case the ColumnType of the column is the underlying type of the
	// domain. Postgres only.
	DomainName string `json:",omitempty"`

	// IsEnum indicates if the column is an enum type. If true, the ColumnType
	// of the column is the name of the enum. Postgres only.
	IsEnum bool `json:",omitempty"`

	// IsNotNull indicates if the column is NOT NULL.
	IsNotNull bool `json:",omitempty"`

	// IsPrimaryKey indicates if the column is the primary key. It is true only
	// if the column is the only column participating in the primary key
	// constraint.
	IsPrimaryKey bool `json:",omitempty"`

	// IsUnique indicates if the column is unique. It is true only if the
	// column is the only column participating in the unique constraint.
	IsUnique bool `json:",omitempty"`

	// IsAutoincrement indicates if the column is AUTO_INCREMENT (MySQL) or
	// AUTOINCREMENT (SQLite).
	IsAutoincrement bool `json:",omitempty"`

	// ReferencesSchema stores the name of the referenced schema if the column
	// is a foreign key. It is filled in only if the column is the only column
	// participating in the foreign key constraint.
	ReferencesSchema string `json:",omitempty"`

	// ReferencesTable stores the name of the referenced table if the column is
	// a foreign key. It is filled in only if the column is the only column
	// participating in the foreign key constraint.
	ReferencesTable string `json:",omitempty"`

	// ReferencesTable stores the name of the referenced column if the column
	// is a foreign key. It is filled in only if the column is the only column
	// participating in the foreign key constraint.
	ReferencesColumn string `json:",omitempty"`

	// UpdateRule stores the ON UPDATE rule of the column's foreign key (if
	// applicable). Possible values: "RESTRICT", "CASCADE", "NO ACTION", "SET
	// NULL", "SET DEFAULT".
	UpdateRule string `json:",omitempty"`

	// DeleteRule stores the ON DELETE of the column's foreign key (if
	// applicable). Possible values: "RESTRICT", "CASCADE", "NO ACTION", "SET
	// NULL", "SET DEFAULT".
	DeleteRule string `json:",omitempty"`

	// IsDeferrable indicates if the column's foreign key is deferrable (if
	// applicable). Postgres only.
	IsDeferrable bool `json:",omitempty"`

	// IsInitiallyDeferred indicates if the column's foreign key is initially
	// deferred (if applicable). Postgres only.
	IsInitiallyDeferred bool `json:",omitempty"`

	// ColumnIdentity stores the identity definition of the column. Possible
	// values: "GENERATED BY DEFAULT AS IDENTITY" (Postgres), "GENERATED ALWAYS
	// AS IDENTITY" (Postgres), "IDENTITY" (SQLServer).
	ColumnIdentity string `json:",omitempty"`

	// ColumnDefault stores the default value of the column as it is literally
	// represented in SQL. So if the default value is a string, the value
	// should be surrounded by 'single quotes'. If the default value is a
	// function call, the default value should be surrounded by brackets e.g.
	// (uuid()).
	ColumnDefault string `json:",omitempty"`

	// OnUpdateCurrentTimestamp indicates if the column is updated with the
	// CURRENT_TIMESTAMP whenever the row is updated. MySQL only.
	OnUpdateCurrentTimestamp bool `json:",omitempty"`

	// IsGenerated indicates if the column is a generated column. It does not
	// have be set to true if the GeneratedExpr field is already non-empty.
	IsGenerated bool `json:",omitempty"`

	// GeneratedExpr holds the generated expression of the column if the column
	// is generated.
	GeneratedExpr string `json:",omitempty"`

	// GeneratedExprStored indicates if the generated column is STORED. If
	// false, the generated column is assumed to be VIRTUAL.
	GeneratedExprStored bool `json:",omitempty"`

	// CollationName stores the collation of the column. If empty, the column
	// collation is assumed to follow the DefaultCollation of the Catalog.
	CollationName string `json:",omitempty"`

	// Comment stores the comment on the column.
	Comment string `json:",omitempty"`

	// If Ignore is true, the column should be treated like it doesn't exist (a
	// soft delete flag).
	Ignore bool `json:",omitempty"`
}

// Constraint represents a database constraint.
type Constraint struct {
	// TableSchema is the name of the schema that the table and constraint belong to.
	TableSchema string `json:",omitempty"`

	// TableName is the name of the table that the constraint belongs to.
	TableName string `json:",omitempty"`

	// ConstraintName is the name of the constraint.
	ConstraintName string `json:",omitempty"`

	// ConstraintType is the type of the constraint. Possible values: "PRIMARY
	// KEY", "UNIQUE", "FOREIGN KEY", "CHECK", "EXCLUDE".
	ConstraintType string `json:",omitempty"`

	// Columns holds the name of the columns participating in the constraint.
	Columns []string `json:",omitempty"`

	// ReferencesSchema stores the name of the referenced schema if the constraint
	// is a foreign key.
	ReferencesSchema string `json:",omitempty"`

	// ReferencesSchema stores the name of the referenced table if the constraint
	// is a foreign key.
	ReferencesTable string `json:",omitempty"`

	// ReferencesSchema stores the name of the referenced columns if the constraint
	// is a foreign key.
	ReferencesColumns []string `json:",omitempty"`

	// UpdateRule stores the ON UPDATE rule if the constraint is a foreign key.
	// Possible values: "RESTRICT", "CASCADE", "NO ACTION", "SET NULL", "SET
	// DEFAULT".
	UpdateRule string `json:",omitempty"`

	// DeleteRule stores the ON DELETE rule if the constraint is a foreign key.
	// Possible values: "RESTRICT", "CASCADE", "NO ACTION", "SET NULL", "SET
	// DEFAULT".
	DeleteRule string `json:",omitempty"`

	//MatchOption stores the MATCH option if the constraint is a foreign key.
	MatchOption string `json:",omitempty"`

	// CheckExpr stores the CHECK expression if the constraint is a CHECK constraint.
	CheckExpr string `json:",omitempty"`

	// ExclusionOperators stores the list of exclusion operators if the
	// constraint is an EXCLUDE constraint. Postgres only.
	ExclusionOperators []string `json:",omitempty"`

	// ExclusionIndexType stores the exclusion index type if the constraint is
	// an EXCLUDE constraint. Postgres only.
	ExclusionIndexType string `json:",omitempty"`

	// ExclusionPredicate stores the exclusion predicate if the constraint is
	// an EXCLUDE constraint. Postgres only.
	ExclusionPredicate string `json:",omitempty"`

	// IsDeferrable indicates if the constraint is deferrable. Postgres only.
	IsDeferrable bool `json:",omitempty"`

	// IsDeferrable indicates if the constraint is initially deferred. Postgres
	// only.
	IsInitiallyDeferred bool `json:",omitempty"`

	// IsClustered indicates if the constraint is the clustered index of the
	// table. SQLServer only.
	IsClustered bool `json:",omitempty"`

	// IsNotValid indicates if the constraint exists but is not valid e.g. if
	// it was constructed with the NOT VALID (Postgres) or WITH NOCHECK
	// (SQLServer).
	IsNotValid bool `json:",omitempty"`

	// Comment stores the comment on the constraint.
	Comment string `json:",omitempty"`

	// If Ignore is true, the constraint should be treated like it doesn't
	// exist (a soft delete flag).
	Ignore bool `json:",omitempty"`
}

// Index represents a database index.
type Index struct {
	// TableSchema is the name of the schema that the table and index belong to.
	TableSchema string `json:",omitempty"`

	// TableName is the name of the table (or view) that the index belongs to.
	TableName string `json:",omitempty"`

	// IndexName is the name of the index.
	IndexName string `json:",omitempty"`

	// IndexType is the type of the index.
	IndexType string `json:",omitempty"`

	// IsViewIndex indicates if the index is for a view.
	IsViewIndex bool `json:",omitempty"`

	// IsUnique indicates if the index is a unique index.
	IsUnique bool `json:",omitempty"`

	// Columns holds the names of the columns participating in the index.
	Columns []string `json:",omitempty"`

	// IncludeColumns holds the names of the columns that are included by the
	// index (the INCLUDE clause).
	IncludeColumns []string `json:",omitempty"`

	// Descending indicates if each column of the index is descending.
	Descending []bool `json:",omitempty"`

	// Opclasses holds the opclass of each column of the index. Postgres only.
	Opclasses []string `json:",omitempty"`

	// Predicate stores the index predicate i.e. the index is a partial index.
	Predicate string `json:",omitempty"`

	// SQL is the SQL definition of the index.
	SQL string `json:",omitempty"`

	// Comment stores the comment on the index.
	Comment string `json:",omitempty"`

	// If Ignore is true, the index should be treated like it doesn't exist (a
	// soft delete flag).
	Ignore bool `json:",omitempty"`
}

// Trigger represents a database trigger.
type Trigger struct {
	// TableSchema is the name of the schema that the table and trigger belong to.
	TableSchema string `json:",omitempty"`

	// TableName is the name of the table (or view) that the trigger belongs to.
	TableName string `json:",omitempty"`

	// TriggerName is the name of the trigger.
	TriggerName string `json:",omitempty"`

	// IsViewTrigger indicates if the trigger belongs to a view.
	IsViewTrigger bool `json:",omitempty"`

	// SQL is the SQL definition of the trigger.
	SQL string `json:",omitempty"`

	// Attrs stores additional metadata about the trigger.
	Attrs map[string]string `json:",omitempty"`

	// Comment stores the comment on the trigger.
	Comment string `json:",omitempty"`

	// If Ignore is true, the trigger should be treated like it doesn't exist
	// (a soft delete flag).
	Ignore bool `json:",omitempty"`
}

// generate name generates the appropriate constraint/index name for a given
// table and columns.
func generateName(nameType string, tableName string, columnNames []string) string {
	var b strings.Builder
	n := len(tableName) + len(nameType)
	for _, columnName := range columnNames {
		n += len(columnName) + 1
	}
	b.Grow(n)
	for _, char := range tableName {
		if char == ' ' {
			char = '_'
		}
		b.WriteRune(char)
	}
	for _, columnName := range columnNames {
		b.WriteString("_")
		for _, char := range columnName {
			if char == ' ' {
				char = '_'
			}
			b.WriteRune(char)
		}
	}
	var suffix string
	switch nameType {
	case PRIMARY_KEY:
		suffix = "_pkey"
	case FOREIGN_KEY:
		suffix = "_fkey"
	case UNIQUE:
		suffix = "_key"
	case INDEX:
		suffix = "_idx"
	case CHECK:
		suffix = "_check"
	case EXCLUDE:
		suffix = "_excl"
	}
	// Cap length to 63 chars (Postgres' limitation).
	excessLength := b.Len() + len(suffix) - 63
	if excessLength > 0 {
		trimmedPrefix := b.String()
		trimmedPrefix = trimmedPrefix[:len(trimmedPrefix)-excessLength]
		return trimmedPrefix + suffix
	}
	b.WriteString(suffix)
	return b.String()
}

func isLiteral(s string) bool {
	// is string literal?
	if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		return true
	}
	// is known literal?
	if strings.EqualFold(s, "TRUE") ||
		strings.EqualFold(s, "FALSE") ||
		strings.EqualFold(s, "CURRENT_DATE") ||
		strings.EqualFold(s, "CURRENT_TIME") ||
		strings.EqualFold(s, "CURRENT_TIMESTAMP") ||
		strings.EqualFold(s, "NULL") {
		return true
	}
	// is int literal?
	if _, err := strconv.ParseInt(s, 10, 64); err == nil {
		return true
	}
	// is float literal?
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return true
	}
	return false
}

func wrapBrackets(s string) string {
	if s == "" {
		return ""
	}
	if s[0] == '(' && s[len(s)-1] == ')' {
		return s
	}
	return "(" + s + ")"
}

func unwrapBrackets(s string) string {
	if s == "" {
		return ""
	}
	last := len(s) - 1
	if s[0] == '(' && s[last] == ')' {
		return s[1:last]
	}
	return s
}

func wrappedInBrackets(s string) bool {
	return s != "" && s[0] == '(' && s[len(s)-1] == ')'
}

// splitArgs works like strings.Split with commas except it ignores commas inside
// 'strings', (brackets) and [square brackets].
func splitArgs(s string) []string {
	if s == "" {
		return nil
	}
	var args []string
	var splitAt, skipCharAt, arrayLevel, bracketLevel int
	var insideString bool
	for {
		splitAt, skipCharAt, arrayLevel, bracketLevel = -1, -1, 0, 0
		insideString = false
		for i, char := range s {
			// do we unconditionally skip the current char?
			if skipCharAt == i {
				continue
			}
			// are we currently inside an array literal?
			if arrayLevel > 0 {
				switch char {
				// does the current char close an array literal?
				case ']':
					arrayLevel--
				// does the current char start a new array literal?
				case '[':
					arrayLevel++
				}
				continue
			}
			// are we currently inside a bracket expression?
			if bracketLevel > 0 {
				switch char {
				// does the current char close a bracket expression?
				case ')':
					bracketLevel--
				// does the current char start a new bracket expression?
				case '(':
					bracketLevel++
				}
				continue
			}
			// are we currently inside a string?
			if insideString {
				nextIndex := i + 1
				// does the current char terminate the current string?
				if char == '\'' {
					// is the next char the same as the current char, which
					// escapes it and prevents it from terminating the current
					// string?
					if nextIndex < len(s) && s[nextIndex] == '\'' {
						skipCharAt = nextIndex
					} else {
						insideString = false
					}
				}
				continue
			}
			// does the current char mark the start of a new array literal?
			if char == '[' {
				arrayLevel++
				continue
			}
			// does the current char mark the start of a new bracket expression?
			if char == '(' {
				bracketLevel++
				continue
			}
			// does the current char mark the start of a new string?
			if char == '\'' {
				insideString = true
				continue
			}
			// is the current char an argument delimiter?
			if char == ',' {
				splitAt = i
				break
			}
		}
		// did we find an argument delimiter?
		if splitAt >= 0 {
			args, s = append(args, s[:splitAt]), s[splitAt+1:]
		} else {
			args = append(args, s)
			break
		}
	}
	return args
}

// normalizeColumnType will normalize column types so that they can be
// meaningfully compared.
func normalizeColumnType(dialect string, columnType string) (normalizedType, arg1, arg2 string) {
	columnType = strings.ToUpper(strings.TrimSpace(columnType))
	normalizedType = columnType
	var args, suffix string
	i := strings.Index(columnType, "(")
	j := strings.LastIndex(columnType, ")")
	if j > i {
		normalizedType = strings.TrimSpace(columnType[:i])
		args = strings.TrimSpace(columnType[i+1 : j])
		suffix = strings.TrimSpace(columnType[j+1:])
		k := strings.Index(args, ",")
		if k >= 0 {
			arg1 = strings.TrimSpace(args[:k])
			arg2 = strings.TrimSpace(args[k+1:])
		} else {
			arg1 = strings.TrimSpace(args)
		}
	}
	isPostgresArray := false
	if dialect == DialectPostgres && strings.HasSuffix(normalizedType, "[]") {
		isPostgresArray = true
		normalizedType = normalizedType[:len(normalizedType)-2]
	}
	switch dialect {
	case DialectPostgres:
		// https://www.postgresql.org/docs/current/datatype.html
		switch strings.ReplaceAll(normalizedType, " ", "") {
		// Numeric
		case "INTEGER", "SERIAL", "SERIAL4", "INT4":
			normalizedType = "INT"
		case "BIGSERIAL", "SERIAL8", "INT8":
			normalizedType = "BIGINT"
		case "SMALLSERIAL", "SERIAL2", "INT2":
			normalizedType = "SMALLINT"
		case "DECIMAL":
			normalizedType = "NUMERIC"
		case "FLOAT4":
			normalizedType = "REAL"
		case "FLOAT8":
			normalizedType = "DOUBLE PRECISION"
		// Character
		case "CHARACTERVARYING":
			normalizedType = "VARCHAR"
		case "CHARACTER":
			normalizedType = "CHAR"
		// TimeField
		case "TIMESTAMPWITHOUTTIMEZONE":
			normalizedType = "TIMESTAMP"
		case "TIMESTAMPWITHTIMEZONE":
			normalizedType = "TIMESTAMPTZ"
		case "TIMESTAMP":
			if suffix != "" && strings.ReplaceAll(suffix, " ", "") == "WITHTIMEZONE" {
				normalizedType = "TIMESTAMPTZ"
			}
		case "TIMEWITHOUTTIMEZONE":
			normalizedType = "TIME"
		case "TIMEWITHTIMEZONE":
			normalizedType = "TIMETZ"
		case "TIME":
			if suffix != "" && strings.ReplaceAll(suffix, " ", "") == "WITHTIMEZONE" {
				normalizedType = "TIMETZ"
			}
		// Binary
		case "BITVARYING":
			normalizedType = "VARBIT"
		// Boolean
		case "BOOL":
			normalizedType = "BOOLEAN"
		}
	case DialectMySQL:
		isUnsigned := strings.HasSuffix(normalizedType, " UNSIGNED")
		if isUnsigned {
			normalizedType = strings.TrimSuffix(normalizedType, " UNSIGNED")
		} else {
			normalizedType = strings.TrimSuffix(normalizedType, " SIGNED")
		}
		switch strings.ReplaceAll(normalizedType, " ", "") {
		case "INTEGER":
			normalizedType, arg1, arg2 = "INT", "", ""
		case "DEC", "DECIMAL":
			normalizedType = "NUMERIC"
		case "BOOL", "BOOLEAN":
			normalizedType, arg1, arg2 = "TINYINT", "1", ""
		case "TINYINT":
			if arg1 != "1" {
				arg1 = ""
			}
			arg2 = ""
		case "SMALLINT", "MEDIUMINT", "INT", "BIGINT":
			// MySQL display width is deprecated
			// https://dev.mysql.com/doc/refman/8.0/en/numeric-type-attributes.html
			arg1, arg2 = "", ""
		}
		if isUnsigned {
			normalizedType += " UNSIGNED"
		}
	case DialectSQLServer:
		switch strings.ReplaceAll(normalizedType, " ", "") {
		case "BINARYVARYING":
			normalizedType = "VARBINARY"
		case "INTEGER":
			normalizedType = "INT"
		case "NATIONALCHARACTERVARYING":
			normalizedType = "NVARCHAR"
		case "CHARACTERVARYING":
			normalizedType = "VARCHAR"
		case "CHARACTER":
			normalizedType = "CHAR"
		case "DEC", "DECIMAL":
			normalizedType = "NUMERIC"
		}
	}
	if isPostgresArray {
		normalizedType = normalizedType + "[]"
	}
	return normalizedType, arg1, arg2
}

// normalizeColumnType will normalize column defaults so that they can be
// meaningfully compared.
func normalizeColumnDefault(dialect string, columnDefault string) (normalizedDefault string) {
	columnDefault = strings.TrimSpace(columnDefault)
	if columnDefault == "" {
		return ""
	}
	upperDefault := strings.ToUpper(columnDefault)
	switch upperDefault {
	case "1", "TRUE":
		return "'1'"
	case "0", "FALSE":
		return "'0'"
	case "CURRENT_DATE", "CURRENT_TIME", "CURRENT_TIMESTAMP", "NULL":
		return upperDefault
	}
	switch dialect {
	case DialectSQLite:
		if upperDefault == "DATETIME()" || upperDefault == "DATETIME('NOW')" {
			return "CURRENT_TIMESTAMP"
		}
	case DialectPostgres:
		if upperDefault == "NOW()" {
			return "CURRENT_TIMESTAMP"
		}
		if before, _, found := strings.Cut(columnDefault, "::"); found {
			return before
		}
	case DialectMySQL:
		if upperDefault == "NOW()" {
			return "CURRENT_TIMESTAMP"
		}
	case DialectSQLServer:
		if upperDefault == "GETDATE()" {
			return "CURRENT_TIMESTAMP"
		}
	}
	return columnDefault
}

// dirFS is like os.DirFS without the restriction of banning filenames like
// '../../somefile.sql'.
type dirFS string

var _ fs.FS = (*dirFS)(nil)

// Open implements fs.FS.
func (d dirFS) Open(name string) (fs.File, error) {
	dir := string(d)
	if dir == "." {
		dir = ""
	}
	return os.Open(filepath.Join(dir, name))
}

var (
	driversMu sync.RWMutex
	drivers   = make(map[string]Driver)
)

// Driver represents the capabilities of the underlying database driver for a
// particular dialect. It is not necessary to implement all fields.
type Driver struct {
	// (Required) Dialect is the database dialect. Possible values: "sqlite", "postgres",
	// "mysql", "sqlserver".
	Dialect string

	// (Required) DriverName is the driverName to be used with sql.Open().
	DriverName string

	// If not nil, IsLockTimeout is used to check if an error is a
	// database-specific lock timeout error.
	IsLockTimeout func(error) bool

	// If not nil, PreprocessDSN will be called on a dataSourceName right
	// before it is passed in to sql.Open().
	PreprocessDSN func(string) string

	// If not nil, AnnotateError will be called on an error returned by the
	// database to display to the user. The primary purpose is to annotate the
	// error with useful information like line number where an error occurred.
	AnnotateError func(originalErr error, query string) error
}

// Registers registers a driver for a particular dialect. It is safe to call
// Register for a dialect multiple times, the last one wins.
func Register(driver Driver) {
	driversMu.Lock()
	defer driversMu.Unlock()
	if driver.Dialect == "" {
		panic("ddl: driver dialect cannot be empty")
	}
	drivers[driver.Dialect] = driver
}

func getDriver(dialect string) (driver Driver, ok bool) {
	driversMu.RLock()
	defer driversMu.RUnlock()
	driver, ok = drivers[dialect]
	return driver, ok
}

// NormalizeDSN normalizes an input DSN (Data Source Name), using a heuristic
// to detect the dialect of the DSN as well as providing an appropriate
// driverName to be used with sql.Open().
func NormalizeDSN(dsn string) (dialect, driverName, normalizedDSN string) {
	if strings.HasPrefix(dsn, "file:") {
		filename, _, _ := strings.Cut(strings.TrimPrefix(strings.TrimPrefix(dsn, "file:"), "//"), "?")
		file, err := os.Open(filename)
		if errors.Is(err, os.ErrNotExist) && (strings.HasSuffix(filename, ".sqlite") ||
			strings.HasSuffix(filename, ".sqlite3") ||
			strings.HasSuffix(filename, ".db") ||
			strings.HasSuffix(filename, ".db3")) {
			return DialectSQLite, "sqlite3", dsn
		}
		if err != nil {
			return "", "", ""
		}
		defer file.Close()
		r := bufio.NewReader(file)
		// SQLite databases may also start with a 'file:' prefix. Treat the
		// contents of the file as a dsn only if the file isn't already an
		// SQLite database i.e. the first 16 bytes isn't the SQLite file
		// header. https://www.sqlite.org/fileformat.html#the_database_header
		header, err := r.Peek(16)
		if err != nil {
			return "", "", ""
		}
		if string(header) == "SQLite format 3\x00" {
			dsn = "sqlite:" + dsn
		} else {
			var b strings.Builder
			_, err = r.WriteTo(&b)
			if err != nil {
				return "", "", ""
			}
			dsn = strings.TrimSpace(b.String())
		}
	}
	trimmedDSN, _, _ := strings.Cut(dsn, "?")
	if strings.HasPrefix(dsn, "sqlite:") || strings.HasPrefix(dsn, "sqlite3:") {
		dialect = DialectSQLite
	} else if strings.HasPrefix(dsn, "postgres://") {
		dialect = DialectPostgres
	} else if strings.HasPrefix(dsn, "mysql://") {
		dialect = DialectMySQL
	} else if strings.HasPrefix(dsn, "sqlserver://") {
		dialect = DialectSQLServer
	} else if strings.HasPrefix(dsn, "oracle://") {
		dialect = DialectOracle
	} else if strings.Contains(dsn, "@tcp(") || strings.Contains(dsn, "@unix(") {
		dialect = DialectMySQL
	} else if strings.HasSuffix(trimmedDSN, ".sqlite") ||
		strings.HasSuffix(trimmedDSN, ".sqlite3") ||
		strings.HasSuffix(trimmedDSN, ".db") ||
		strings.HasSuffix(trimmedDSN, ".db3") {
		dialect = DialectSQLite
	} else {
		return "", "", ""
	}
	if driver, ok := getDriver(dialect); ok {
		driverName = driver.DriverName
		if driver.PreprocessDSN != nil {
			return dialect, driver.DriverName, driver.PreprocessDSN(dsn)
		}
		return dialect, driver.DriverName, dsn
	}
	switch dialect {
	case DialectSQLite:
		var tmp string
		if strings.HasPrefix(dsn, "sqlite3:") {
			tmp = strings.TrimPrefix(dsn, "sqlite3:")
		} else {
			tmp = strings.TrimPrefix(dsn, "sqlite:")
		}
		return dialect, "sqlite3", strings.TrimPrefix(tmp, "//")
	case DialectPostgres:
		return dialect, "postgres", dsn
	case DialectMySQL:
		return dialect, "mysql", strings.TrimPrefix(dsn, "mysql://")
	case DialectSQLServer:
		return dialect, "sqlserver", dsn
	case DialectOracle:
		return dialect, "oracle", dsn
	}
	return "", "", ""
}

func timestamp() string {
	return time.Now().UTC().Format("2006-01-02 15:04:05 ")
}

type strslice []string

var _ flag.Value = (*strslice)(nil)

func (strs *strslice) String() string { return fmt.Sprint(*strs) }

func (strs *strslice) Set(value string) error {
	*strs = append(*strs, value)
	return nil
}

// Really hacky way to detect virtual tables, switch to a more sophisticated
// method if problems are reported.
func isVirtualTable(table *Table) bool {
	return table != nil && (table.IsVirtual || strings.HasPrefix(table.SQL, "CREATE VIRTUAL TABLE"))
}

// VersionNums holds a database's version.
type VersionNums []int

// LowerThan checks if the database version is lower than the given version
// numbers. You can provide just one number (the major version) or multiple
// numbers (the major, minor and patch versions). E.g.
//
//   version.LowerThan(12)   # $version < 12
//   version.LowerThan(8, 5) # $version < 8.5
func (v VersionNums) LowerThan(nums ...int) bool {
	for i, versionNum := range v {
		if len(nums) <= i || versionNum > nums[i] {
			return false
		}
		if versionNum < nums[i] {
			return true
		}
	}
	return false
}

// GreaterOrEqualTo checks if the database version is greater or equal to the
// given version numbers. You can provide just one number (the major version)
// or multiple numbers (the major, minor and patch versions). E.g.
//
//   version.GreaterOrEqualTo(12)   # $version >= 12
//   version.GreaterOrEqualTo(8, 5) # $version >= 8.5
func (v VersionNums) GreaterOrEqualTo(nums ...int) bool {
	return !v.LowerThan(nums...)
}
