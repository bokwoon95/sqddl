package ddl

import (
	"bytes"
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"strconv"
	"strings"
	"text/template"

	"github.com/bokwoon95/sq"
)

//go:embed introspection_scripts
var embedFS embed.FS

var templates = template.New("").Funcs(template.FuncMap{"mklist": mklist})

// DatabaseIntrospector is used to introspect a database.
type DatabaseIntrospector struct {
	// Filter is a struct used by the DatabaseIntrospector in order to narrow
	// down its search.
	Filter

	// The dialect of the database being introspected. Possible values:
	// "sqlite", "postgres", "mysql", "sqlserver".
	Dialect string

	// DB is the database connection used to introspect the database.
	DB sq.DB
}

// NewDatabaseIntrospector creates a new DatabaseIntrospector.
func NewDatabaseIntrospector(dialect string, db sq.DB) *DatabaseIntrospector {
	return &DatabaseIntrospector{Dialect: dialect, DB: db}
}

// WriteCatalog populates the Catalog by introspecting the database.
func (dbi *DatabaseIntrospector) WriteCatalog(catalog *Catalog) error {
	if catalog.Dialect == "" {
		catalog.Dialect = dbi.Dialect
	}
	var err error
	cache := NewCatalogCache(catalog)
	objectTypes := make(map[string]struct{})
	for _, objectType := range dbi.Filter.ObjectTypes {
		objectTypes[objectType] = struct{}{}
	}
	includeObjectType := func(objectType string) bool {
		if len(objectTypes) == 0 {
			return true
		}
		_, ok := objectTypes[objectType]
		return ok
	}

	catalog.VersionNums, err = dbi.GetVersionNums()
	if err != nil {
		return err
	}
	dbi.Filter.VersionNums = catalog.VersionNums

	catalog.CatalogName, err = dbi.GetDatabaseName()
	if err != nil {
		return err
	}

	catalog.CurrentSchema, err = dbi.GetCurrentSchema()
	if err != nil {
		return err
	}

	catalog.DefaultCollation, err = dbi.GetDefaultCollation()
	if err != nil {
		return err
	}
	catalog.DefaultCollationValid = true

	if includeObjectType("EXTENSIONS") {
		extensions, err := dbi.GetExtensions()
		if err != nil {
			return err
		}
		if len(extensions) > 0 {
			catalog.Extensions = append(catalog.Extensions, extensions...)
			catalog.ExtensionsValid = true
		}
	}

	if includeObjectType("ENUMS") {
		enums, err := dbi.GetEnums()
		if err != nil {
			return err
		}
		for _, enum := range enums {
			schema := cache.GetOrCreateSchema(catalog, enum.EnumSchema)
			schema.EnumsValid = true
			cache.AddOrUpdateEnum(schema, enum)
		}
	}

	if includeObjectType("DOMAINS") {
		domains, err := dbi.GetDomains()
		if err != nil {
			return err
		}
		for _, domain := range domains {
			schema := cache.GetOrCreateSchema(catalog, domain.DomainSchema)
			schema.DomainsValid = true
			cache.AddOrUpdateDomain(schema, domain)
		}
	}

	if includeObjectType("ROUTINES") {
		routines, err := dbi.GetRoutines()
		if err != nil {
			return err
		}
		for _, routine := range routines {
			schema := cache.GetOrCreateSchema(catalog, routine.RoutineSchema)
			schema.RoutinesValid = true
			cache.AddOrUpdateRoutine(schema, routine)
		}
	}

	if includeObjectType("VIEWS") {
		views, err := dbi.GetViews()
		if err != nil {
			return err
		}
		for _, view := range views {
			schema := cache.GetOrCreateSchema(catalog, view.ViewSchema)
			schema.ViewsValid = true
			// view.TriggersValid = true
			cache.AddOrUpdateView(schema, view)
		}
	}

	if includeObjectType("TABLES") {
		tables, err := dbi.GetTables()
		if err != nil {
			return err
		}
		for _, table := range tables {
			schema := cache.GetOrCreateSchema(catalog, table.TableSchema)
			cache.AddOrUpdateTable(schema, table)
		}

		columns, err := dbi.GetColumns()
		if err != nil {
			return err
		}
		for _, column := range columns {
			schema := cache.GetSchema(catalog, column.TableSchema)
			if schema == nil {
				continue
			}
			table := cache.GetTable(schema, column.TableName)
			if table == nil {
				continue
			}
			// table.TriggersValid = true
			cache.AddOrUpdateColumn(table, column)
		}

		constraints, err := dbi.GetConstraints()
		if err != nil {
			return err
		}
		for _, constraint := range constraints {
			schema := cache.GetSchema(catalog, constraint.TableSchema)
			if schema == nil {
				continue
			}
			table := cache.GetOrCreateTable(schema, constraint.TableName)
			if table == nil {
				continue
			}
			cache.AddOrUpdateConstraint(table, constraint)
		}
	}

	if includeObjectType("TABLES") || includeObjectType("VIEWS") {
		indexes, err := dbi.GetIndexes()
		if err != nil {
			return err
		}
		for _, index := range indexes {
			schema := cache.GetSchema(catalog, index.TableSchema)
			if schema == nil {
				continue
			}
			if index.IsViewIndex {
				view := cache.GetView(schema, index.TableName)
				if view == nil {
					continue
				}
				cache.AddOrUpdateViewIndex(view, index)
			} else {
				table := cache.GetTable(schema, index.TableName)
				if table == nil {
					continue
				}
				cache.AddOrUpdateIndex(table, index)
			}
		}

		triggers, err := dbi.GetTriggers()
		if err != nil {
			return err
		}
		for _, trigger := range triggers {
			schema := cache.GetSchema(catalog, trigger.TableSchema)
			if schema == nil {
				continue
			}
			if trigger.IsViewTrigger {
				view := cache.GetView(schema, trigger.TableName)
				if view == nil {
					continue
				}
				cache.AddOrUpdateViewTrigger(view, trigger)
			} else {
				table := cache.GetTable(schema, trigger.TableName)
				if table == nil {
					continue
				}
				cache.AddOrUpdateTrigger(table, trigger)
			}
		}
	}

	// Set IsPrimaryKey and IsUnique fields for each primary key or unique
	// column.
	for _, schema := range catalog.Schemas {
		for _, table := range schema.Tables {
			for _, constraint := range table.Constraints {
				if len(constraint.Columns) != 1 || (constraint.ConstraintType != PRIMARY_KEY && constraint.ConstraintType != UNIQUE && constraint.ConstraintType != FOREIGN_KEY) {
					continue
				}
				column := cache.GetColumn(&table, constraint.Columns[0])
				switch constraint.ConstraintType {
				case PRIMARY_KEY:
					column.IsPrimaryKey = true
				case UNIQUE:
					column.IsUnique = true
				case FOREIGN_KEY:
					column.ReferencesSchema = constraint.ReferencesSchema
					column.ReferencesTable = constraint.ReferencesTable
					column.ReferencesColumn = constraint.ReferencesColumns[0]
					column.UpdateRule = constraint.UpdateRule
					column.DeleteRule = constraint.DeleteRule
					column.IsDeferrable = constraint.IsDeferrable
					column.IsInitiallyDeferred = constraint.IsInitiallyDeferred
				}
			}
		}
	}
	return nil
}

// GetVersionNums returns the version numbers of the database.
func (dbi *DatabaseIntrospector) GetVersionNums() (versionNums VersionNums, err error) {
	ctx := context.Background()
	var rows *sql.Rows
	switch dbi.Dialect {
	case DialectSQLite:
		rows, err = dbi.DB.QueryContext(ctx, "SELECT sqlite_version()")
		if err != nil {
			return nil, err
		}
	case DialectPostgres:
		rows, err = dbi.DB.QueryContext(ctx, "SELECT current_setting('server_version')") // alternatively, "SHOW server_version"
		if err != nil {
			return nil, err
		}
	case DialectMySQL:
		rows, err = dbi.DB.QueryContext(ctx, "SELECT version()")
		if err != nil {
			return nil, err
		}
	case DialectSQLServer:
		rows, err = dbi.DB.QueryContext(ctx, "SELECT SERVERPROPERTY('ProductVersion')")
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported dialect: %s", dbi.Dialect)
	}
	defer rows.Close()
	var versionStr string
	if rows.Next() {
		err = rows.Scan(&versionStr)
		if err != nil {
			return nil, fmt.Errorf("scanning versionString: %w", err)
		}
		if dbi.Dialect == DialectPostgres {
			versionStr, _, _ = strings.Cut(versionStr, " ")
		}
	}
	versionStrs := strings.Split(versionStr, ".")
	versionNums = make([]int, len(versionStrs))
	for i, str := range versionStrs {
		versionNum, err := strconv.Atoi(str)
		if err != nil {
			return versionNums, fmt.Errorf("version %s: cannot convert %s to integer: %w", versionStr, str, err)
		}
		versionNums[i] = versionNum
	}
	return versionNums, closeRows(rows)
}

// GetDatabaseName returns the database name.
func (dbi *DatabaseIntrospector) GetDatabaseName() (databaseName string, err error) {
	ctx := context.Background()
	var rows *sql.Rows
	switch dbi.Dialect {
	case DialectSQLite:
		return "", nil
	case DialectPostgres:
		rows, err = dbi.DB.QueryContext(ctx, "SELECT current_database()")
		if err != nil {
			return "", err
		}
	case DialectMySQL:
		rows, err = dbi.DB.QueryContext(ctx, "SELECT database()")
		if err != nil {
			return "", err
		}
	case DialectSQLServer:
		rows, err = dbi.DB.QueryContext(ctx, "SELECT DB_NAME()")
		if err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("unsupported dialect: %s", dbi.Dialect)
	}
	defer rows.Close()
	if rows.Next() {
		err = rows.Scan(&databaseName)
		if err != nil {
			return "", fmt.Errorf("scanning databaseName: %w", err)
		}
	}
	return databaseName, closeRows(rows)
}

// GetCurrentSchema returns the current schema of the database.
func (dbi *DatabaseIntrospector) GetCurrentSchema() (currentSchema string, err error) {
	ctx := context.Background()
	var rows *sql.Rows
	switch dbi.Dialect {
	case DialectSQLite:
		return "", nil
	case DialectPostgres:
		rows, err = dbi.DB.QueryContext(ctx, "SELECT current_schema()")
		if err != nil {
			return "", err
		}
	case DialectMySQL:
		rows, err = dbi.DB.QueryContext(ctx, "SELECT database()")
		if err != nil {
			return "", err
		}
	case DialectSQLServer:
		rows, err = dbi.DB.QueryContext(ctx, "SELECT SCHEMA_NAME()")
		if err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("unsupported dialect: %s", dbi.Dialect)
	}
	defer rows.Close()
	if rows.Next() {
		err = rows.Scan(&currentSchema)
		if err != nil {
			return "", fmt.Errorf("scanning currentSchema: %w", err)
		}
	}
	return currentSchema, closeRows(rows)
}

// GetDefaultCollation returns the default collation of the database.
func (dbi *DatabaseIntrospector) GetDefaultCollation() (defaultCollation string, err error) {
	ctx := context.Background()
	var rows *sql.Rows
	switch dbi.Dialect {
	case DialectSQLite:
		return "", nil
	case DialectPostgres:
		rows, err = dbi.DB.QueryContext(ctx, "SELECT datcollate AS collation FROM pg_database WHERE datname = current_database()")
		if err != nil {
			return "", err
		}
	case DialectMySQL:
		rows, err = dbi.DB.QueryContext(ctx, "SELECT @@collation_database")
		if err != nil {
			return "", err
		}
	case DialectSQLServer:
		rows, err = dbi.DB.QueryContext(ctx, "SELECT SERVERPROPERTY('collation')")
		if err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("unsupported dialect: %s", dbi.Dialect)
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&defaultCollation)
		if err != nil {
			return "", fmt.Errorf("scanning defaultCollation: %w", err)
		}
		break
	}
	return defaultCollation, closeRows(rows)
}

// GetColumns returns the columns in the database.
//
// To narrow down your search to a specific schema and table, pass the schema
// and table names into the DatabaseIntrospector.Filter.Schemas slice and
// the DatabaseIntrospector.Filter.Tables slice respectively.
func (dbi *DatabaseIntrospector) GetColumns() ([]Column, error) {
	ctx := context.Background()
	var err error
	var rows *sql.Rows
	switch dbi.Dialect {
	case DialectSQLite:
		rows, err = dbi.queryContext(ctx, "introspection_scripts/sqlite_columns.sql")
		if err != nil {
			return nil, err
		}
	case DialectPostgres:
		rows, err = dbi.queryContext(ctx, "introspection_scripts/postgres_columns.sql")
		if err != nil {
			return nil, err
		}
	case DialectMySQL:
		rows, err = dbi.queryContext(ctx, "introspection_scripts/mysql_columns.sql")
		if err != nil {
			return nil, err
		}
	case DialectSQLServer:
		rows, err = dbi.queryContext(ctx, "introspection_scripts/sqlserver_columns.sql")
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported dialect: %s", dbi.Dialect)
	}
	defer rows.Close()
	var columns []Column
	for rows.Next() {
		var column Column
		switch dbi.Dialect {
		case DialectSQLite:
			err = rows.Scan(
				&column.TableName,
				&column.ColumnName,
				&column.ColumnType,
				&column.IsNotNull,
				&column.IsGenerated,
				&column.ColumnDefault,
			)
			if err != nil {
				return nil, fmt.Errorf("scanning Column: %w", err)
			}
			if strings.HasSuffix(column.ColumnType, " GENERATED ALWAYS") {
				column.ColumnType = strings.TrimSuffix(column.ColumnType, " GENERATED ALWAYS")
				column.IsGenerated = true
			}
			if column.ColumnDefault != "" {
				if !isLiteral(column.ColumnDefault) {
					column.ColumnDefault = wrapBrackets(column.ColumnDefault)
				}
			}
		case DialectPostgres:
			err = rows.Scan(
				&column.TableSchema,
				&column.TableName,
				&column.ColumnName,
				&column.ColumnType,
				&column.IsEnum,
				&column.DomainName,
				&column.NumericPrecision,
				&column.NumericScale,
				&column.ColumnIdentity,
				&column.IsNotNull,
				&column.GeneratedExpr,
				&column.GeneratedExprStored,
				&column.CollationName,
				&column.ColumnDefault,
				&column.Comment,
			)
			if err != nil {
				return nil, fmt.Errorf("scanning Column: %w", err)
			}
			normalizedType, arg1, arg2 := normalizeColumnType(dbi.Dialect, column.ColumnType)
			switch normalizedType {
			case "VARCHAR", "CHAR":
				if column.CharacterLength == "" {
					column.CharacterLength = arg1
				}
			case "TIMESTAMPTZ":
				if arg1 != "" {
					column.ColumnType = "timestamptz(" + arg1 + ")"
				} else {
					column.ColumnType = "timestamptz"
				}
			case "NUMERIC":
				if column.NumericPrecision == "" {
					column.NumericPrecision = arg1
				}
				if column.NumericScale == "" {
					column.NumericScale = arg2
				}
			case "FLOAT":
				if arg1 != "" {
					column.ColumnType = "float(" + arg1 + ")"
				}
			default:
				column.CharacterLength, column.NumericPrecision, column.NumericScale = "", "", ""
			}
			if column.IsEnum {
				column.ColumnType = strings.ToLower(column.ColumnType)
			} else if arg1 != "" && arg2 != "" {
				column.ColumnType = strings.ToLower(normalizedType) + "(" + arg1 + "," + arg2 + ")"
			} else if arg1 != "" {
				column.ColumnType = strings.ToLower(normalizedType) + "(" + arg1 + ")"
			} else {
				column.ColumnType = strings.ToLower(normalizedType)
			}
		case DialectMySQL:
			err = rows.Scan(
				&column.TableSchema,
				&column.TableName,
				&column.ColumnName,
				&column.ColumnType,
				&column.CharacterLength,
				&column.NumericPrecision,
				&column.NumericScale,
				&column.IsAutoincrement,
				&column.IsNotNull,
				&column.OnUpdateCurrentTimestamp,
				&column.GeneratedExpr,
				&column.GeneratedExprStored,
				&column.CollationName,
				&column.ColumnDefault,
				&column.Comment,
			)
			normalizedType, arg1, arg2 := normalizeColumnType(dbi.Dialect, column.ColumnType)
			switch normalizedType {
			case "VARCHAR", "CHAR":
				if column.CharacterLength == "" {
					column.CharacterLength = arg1
				}
				column.NumericPrecision, column.NumericScale = "", ""
			case "NUMERIC":
				if column.NumericPrecision == "" {
					column.NumericPrecision = arg1
				}
				if column.NumericScale == "" {
					column.NumericScale = arg2
				}
				column.CharacterLength = ""
			default:
				column.CharacterLength, column.NumericPrecision, column.NumericScale = "", "", ""
			}
			if err != nil {
				return nil, fmt.Errorf("scanning Column: %w", err)
			}
			if column.GeneratedExpr != "" {
				column.GeneratedExpr = strings.ReplaceAll(column.GeneratedExpr, `\'`, `'`)
			}
			if column.ColumnDefault != "" && !isLiteral(column.ColumnDefault) && !wrappedInBrackets(column.ColumnDefault) {
				column.ColumnDefault = `'` + sq.EscapeQuote(column.ColumnDefault, '\'') + `'`
			}
		case DialectSQLServer:
			err = rows.Scan(
				&column.TableSchema,
				&column.TableName,
				&column.ColumnName,
				&column.ColumnType,
				&column.CharacterLength,
				&column.NumericPrecision,
				&column.NumericScale,
				&column.ColumnIdentity,
				&column.IsNotNull,
				&column.GeneratedExpr,
				&column.GeneratedExprStored,
				&column.CollationName,
				&column.ColumnDefault,
			)
			if err != nil {
				return nil, fmt.Errorf("scanning Column: %w", err)
			}
			normalizedType, _, _ := normalizeColumnType(dbi.Dialect, column.ColumnType)
			switch normalizedType {
			case "VARBINARY", "BINARY", "NVARCHAR", "VARCHAR", "CHAR":
				if column.CharacterLength == "-1" {
					column.CharacterLength = "MAX"
				}
				column.ColumnType = column.ColumnType + "(" + column.CharacterLength + ")"
				column.NumericPrecision, column.NumericScale = "", ""
			case "NUMERIC":
				if column.NumericPrecision != "" && column.NumericScale != "" {
					column.ColumnType = column.ColumnType + "(" + column.NumericPrecision + "," + column.NumericScale + ")"
				} else if column.NumericPrecision != "" {
					column.ColumnType = column.ColumnType + "(" + column.NumericPrecision + ")"
				}
				column.CharacterLength = ""
			case "FLOAT":
				column.CharacterLength, column.NumericScale = "", ""
			default:
				column.CharacterLength, column.NumericPrecision, column.NumericScale = "", "", ""
			}
			column.ColumnDefault = unwrapBrackets(unwrapBrackets(column.ColumnDefault))
			if !isLiteral(column.ColumnDefault) {
				column.ColumnDefault = wrapBrackets(column.ColumnDefault)
			}
		}
		columns = append(columns, column)
	}
	return columns, closeRows(rows)
}

// GetConstraints returns the constraints in the database.
//
// To search for specific constraint types, add the constraint types into the
// Databaseintrospector.Filter.ConstraintTypes slice. An empty ConstraintTypes
// slice means all constraint types will be included. The possible constraint
// types are: "PRIMARY KEY", "UNIQUE", "FOREIGN KEY", "CHECK" and "EXCLUDE".
//
// To narrow down your search to a specific schema and table, pass the schema
// and table names into the DatabaseIntrospector.Filter.Schemas slice and
// the DatabaseIntrospector.Filter.Tables slice respectively.
func (dbi *DatabaseIntrospector) GetConstraints() ([]Constraint, error) {
	ctx := context.Background()
	var err error
	var rows *sql.Rows
	switch dbi.Dialect {
	case DialectSQLite:
		rows, err = dbi.queryContext(ctx, "introspection_scripts/sqlite_constraints.sql")
		if err != nil {
			return nil, err
		}
	case DialectPostgres:
		rows, err = dbi.queryContext(ctx, "introspection_scripts/postgres_constraints.sql")
		if err != nil {
			return nil, err
		}
	case DialectMySQL:
		rows, err = dbi.queryContext(ctx, "introspection_scripts/mysql_constraints.sql")
		if err != nil {
			return nil, err
		}
	case DialectSQLServer:
		rows, err = dbi.queryContext(ctx, "introspection_scripts/sqlserver_constraints.sql")
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported dialect: %s", dbi.Dialect)
	}
	defer rows.Close()
	var constraints []Constraint
	for rows.Next() {
		var constraint Constraint
		switch dbi.Dialect {
		case DialectSQLite:
			var columns, referencesColumns string
			err = rows.Scan(
				&constraint.TableName,
				&constraint.ConstraintType,
				&columns,
				&constraint.ReferencesTable,
				&referencesColumns,
				&constraint.UpdateRule,
				&constraint.DeleteRule,
			)
			if err != nil {
				return nil, fmt.Errorf("scanning Constraint: %w", err)
			}
			if columns != "" {
				constraint.Columns = strings.Split(columns, ",")
			}
			if referencesColumns != "" {
				constraint.ReferencesColumns = strings.Split(referencesColumns, ",")
			}
		case DialectPostgres:
			var columns, referencesColumns, operators []byte
			err = rows.Scan(
				&constraint.TableSchema,
				&constraint.TableName,
				&constraint.ConstraintName,
				&constraint.ConstraintType,
				&columns,
				&constraint.ReferencesSchema,
				&constraint.ReferencesTable,
				&referencesColumns,
				&constraint.UpdateRule,
				&constraint.DeleteRule,
				&constraint.MatchOption,
				&constraint.CheckExpr,
				&operators,
				&constraint.ExclusionIndexType,
				&constraint.ExclusionPredicate,
				&constraint.IsDeferrable,
				&constraint.IsInitiallyDeferred,
				&constraint.IsNotValid,
			)
			if err != nil {
				return nil, fmt.Errorf("scanning Constraint: %w", err)
			}
			err = json.Unmarshal(columns, &constraint.Columns)
			if err != nil {
				return nil, fmt.Errorf("unmarshaling %s into %T: %w", columns, constraint.Columns, err)
			}
			err = json.Unmarshal(referencesColumns, &constraint.ReferencesColumns)
			if err != nil {
				return nil, fmt.Errorf("unmarshaling %s into %T: %w", referencesColumns, constraint.ReferencesColumns, err)
			}
			err = json.Unmarshal(operators, &constraint.ExclusionOperators)
			if err != nil {
				return nil, fmt.Errorf("unmarshaling %s into %T: %w", operators, constraint.ExclusionOperators, err)
			}
			constraint.CheckExpr = unwrapBrackets(strings.TrimPrefix(constraint.CheckExpr, "CHECK "))
		case DialectMySQL:
			var columns, referencesColumns string
			err = rows.Scan(
				&constraint.TableSchema,
				&constraint.TableName,
				&constraint.ConstraintName,
				&constraint.ConstraintType,
				&columns,
				&constraint.ReferencesSchema,
				&constraint.ReferencesTable,
				&referencesColumns,
				&constraint.UpdateRule,
				&constraint.DeleteRule,
				&constraint.MatchOption,
				&constraint.CheckExpr,
			)
			if err != nil {
				return nil, fmt.Errorf("scanning Constraint: %w", err)
			}
			if columns != "" {
				constraint.Columns = strings.Split(columns, ",")
			}
			if referencesColumns != "" {
				constraint.ReferencesColumns = strings.Split(referencesColumns, ",")
			}
			constraint.CheckExpr = unwrapBrackets(constraint.CheckExpr)
		case DialectSQLServer:
			var columns, referencesColumns string
			err = rows.Scan(
				&constraint.TableSchema,
				&constraint.TableName,
				&constraint.ConstraintName,
				&constraint.ConstraintType,
				&columns,
				&constraint.ReferencesSchema,
				&constraint.ReferencesTable,
				&referencesColumns,
				&constraint.UpdateRule,
				&constraint.DeleteRule,
				&constraint.CheckExpr,
				&constraint.IsClustered,
				&constraint.IsNotValid,
			)
			if err != nil {
				return nil, fmt.Errorf("scanning Constraint: %w", err)
			}
			if columns != "" {
				constraint.Columns = strings.Split(columns, ",")
			}
			if referencesColumns != "" {
				constraint.ReferencesColumns = strings.Split(referencesColumns, ",")
			}
			constraint.UpdateRule = strings.ReplaceAll(constraint.UpdateRule, "_", " ")
			constraint.DeleteRule = strings.ReplaceAll(constraint.DeleteRule, "_", " ")
			constraint.CheckExpr = unwrapBrackets(constraint.CheckExpr)
		}
		// When deserializing a Catalog from JSON, empty slices are set as nil
		// (because of omitempty). If we want to follow that precedent we have
		// to set empty slices to nil here as well.
		if len(constraint.Columns) == 0 {
			constraint.Columns = nil
		}
		if len(constraint.ReferencesColumns) == 0 {
			constraint.ReferencesColumns = nil
		}
		if len(constraint.ExclusionOperators) == 0 {
			constraint.ExclusionOperators = nil
		}
		if constraint.UpdateRule == NO_ACTION {
			constraint.UpdateRule = ""
		}
		if constraint.DeleteRule == NO_ACTION {
			constraint.DeleteRule = ""
		}
		constraints = append(constraints, constraint)
	}
	return constraints, closeRows(rows)
}

// GetDomains returns the domains in the database. Postgres only.
//
// To search for specific domains, add the domain names into the
// DatabaseIntrospector.Filter.Domains slice. To exclude specific domains
// from your search, add the domain names into the
// DatabaseIntrospector.Filter.ExcludeDomains slice.
//
// To narrow down your search to a specific schema, pass the schema name
// into the DatabaseIntrospector.Filter.Schemas slice.
func (dbi *DatabaseIntrospector) GetDomains() ([]Domain, error) {
	ctx := context.Background()
	if dbi.Dialect != DialectPostgres {
		return nil, nil
	}
	var domains []Domain
	rows, err := dbi.queryContext(ctx, "introspection_scripts/postgres_domains.sql")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var domain Domain
		var b1, b2 []byte
		err = rows.Scan(
			&domain.DomainSchema,
			&domain.DomainName,
			&domain.UnderlyingType,
			&domain.CollationName,
			&domain.IsNotNull,
			&domain.ColumnDefault,
			&b1,
			&b2,
			&domain.Comment,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning enum: %w", err)
		}
		err = json.Unmarshal(b1, &domain.CheckNames)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling domain constraints: %w", err)
		}
		err = json.Unmarshal(b2, &domain.CheckExprs)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling domain constraints: %w", err)
		}
		for i, checkExpr := range domain.CheckExprs {
			domain.CheckExprs[i] = strings.TrimPrefix(checkExpr, "CHECK ")
		}
		// When deserializing a Catalog from JSON, empty slices are set as
		// nil (because of omitempty). If we want to follow that precedent
		// we have to set empty slices to nil here as well.
		if len(domain.CheckNames) == 0 {
			domain.CheckNames = nil
		}
		if len(domain.CheckExprs) == 0 {
			domain.CheckExprs = nil
		}
		domains = append(domains, domain)
	}
	return domains, closeRows(rows)
}

// GetEnums returns the enums in the database. Postgres only.
//
// To search for specific enums, add the enum names into the
// DatabaseIntrospector.Filter.Enums slice. To exclude specific enums from
// your search, add the enum names into the
// DatabaseIntrospector.Filter.ExcludeEnums slice.
//
// To narrow down your search to a specific schema, pass the schema name into
// the DatabaseIntrospector.Filter.Schemas slice.
func (dbi *DatabaseIntrospector) GetEnums() ([]Enum, error) {
	ctx := context.Background()
	if dbi.Dialect != DialectPostgres {
		return nil, nil
	}
	var enums []Enum
	rows, err := dbi.queryContext(ctx, "introspection_scripts/postgres_enums.sql")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var enum Enum
		var b []byte
		err = rows.Scan(&enum.EnumSchema, &enum.EnumName, &b, &enum.Comment)
		if err != nil {
			return nil, fmt.Errorf("scanning enum: %w", err)
		}
		err = json.Unmarshal(b, &enum.EnumLabels)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling enum labels: %w", err)
		}
		enums = append(enums, enum)
	}
	return enums, closeRows(rows)
}

// GetExtensions returns the extensions in the database. Postgres only.
//
// To search for specific extensions, add the extension names into the
// DatabaseIntrospector.Filter.Extensions slice. To exclude specific
// extensions from your search, add the extension names into the
// DatabaseIntrospector.Filter.ExcludeExtensions slice.
func (dbi *DatabaseIntrospector) GetExtensions() (extensions []string, err error) {
	ctx := context.Background()
	if dbi.Dialect != DialectPostgres {
		return nil, nil
	}
	rows, err := dbi.queryContext(ctx, "introspection_scripts/postgres_extensions.sql")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var extname, extversion string
		err = rows.Scan(&extname, &extversion)
		if err != nil {
			return nil, fmt.Errorf("scanning extension: %w", err)
		}
		extensions = append(extensions, extname)
	}
	return extensions, closeRows(rows)
}

// GetIndexes returns the indexes in the database.
//
// To narrow down your search to a specific schema and table, pass the schema
// and table names into the DatabaseIntrospector.Filter.Schemas slice and
// the DatabaseIntrospector.Filter.Tables slice respectively.
func (dbi *DatabaseIntrospector) GetIndexes() ([]Index, error) {
	ctx := context.Background()
	var err error
	var rows *sql.Rows
	switch dbi.Dialect {
	case DialectSQLite:
		rows, err = dbi.queryContext(ctx, "introspection_scripts/sqlite_indexes.sql")
		if err != nil {
			return nil, err
		}
	case DialectPostgres:
		rows, err = dbi.queryContext(ctx, "introspection_scripts/postgres_indexes.sql")
		if err != nil {
			return nil, err
		}
	case DialectMySQL:
		rows, err = dbi.queryContext(ctx, "introspection_scripts/mysql_indexes.sql")
		if err != nil {
			return nil, err
		}
	case DialectSQLServer:
		rows, err = dbi.queryContext(ctx, "introspection_scripts/sqlserver_indexes.sql")
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported dialect: %s", dbi.Dialect)
	}
	defer rows.Close()
	var indexes []Index
	for rows.Next() {
		var index Index
		switch dbi.Dialect {
		case DialectSQLite:
			var columns string
			err = rows.Scan(
				&index.TableName,
				&index.IndexName,
				&index.IsUnique,
				&columns,
				&index.SQL,
			)
			if err != nil {
				return nil, fmt.Errorf("scanning Index: %w", err)
			}
			if columns != "" {
				index.Columns = strings.Split(columns, ",")
			}
		case DialectPostgres:
			var columns, opclasses []byte
			var numKeyColumns int
			err = rows.Scan(
				&index.TableSchema,
				&index.TableName,
				&index.IndexName,
				&index.IndexType,
				&index.IsViewIndex,
				&index.IsUnique,
				&numKeyColumns,
				&columns,
				&opclasses,
				&index.Predicate,
				&index.SQL,
			)
			if err != nil {
				return nil, fmt.Errorf("scanning Index: %w", err)
			}
			err = json.Unmarshal(columns, &index.Columns)
			if err != nil {
				return nil, fmt.Errorf("unmarshaling %s into %T: %w", columns, index.Columns, err)
			}
			err = json.Unmarshal(opclasses, &index.Opclasses)
			if err != nil {
				return nil, fmt.Errorf("unmarshaling %s into %T: %w", opclasses, index.Opclasses, err)
			}
			index.Columns, index.IncludeColumns = index.Columns[:numKeyColumns], index.Columns[numKeyColumns:]
		case DialectMySQL:
			var columns, descending string
			err = rows.Scan(
				&index.TableSchema,
				&index.TableName,
				&index.IndexName,
				&index.IndexType,
				&index.IsUnique,
				&columns,
				&descending,
			)
			if err != nil {
				return nil, fmt.Errorf("scanning Index: %w", err)
			}
			index.Columns = splitArgs(strings.TrimSpace(strings.ReplaceAll(columns, `\'`, `'`)))
			index.Descending = make([]bool, 0, len(index.Columns))
			if descending != "" {
				for _, str := range strings.Split(descending, ",") {
					b, _ := strconv.ParseBool(str)
					index.Descending = append(index.Descending, b)
				}
			}
		case DialectSQLServer:
			var columns, descending, included string
			err = rows.Scan(
				&index.TableSchema,
				&index.TableName,
				&index.IndexName,
				&index.IndexType,
				&index.IsViewIndex,
				&index.IsUnique,
				&columns,
				&descending,
				&included,
				&index.Predicate,
			)
			if err != nil {
				return nil, fmt.Errorf("scanning Index: %w", err)
			}
			include := make([]bool, 0, strings.Count(included, ",")+1)
			if included != "" {
				for _, str := range strings.Split(included, ",") {
					b, _ := strconv.ParseBool(str)
					include = append(include, b)
				}
			}
			index.IncludeColumns = make([]string, 0, strings.Count(included, "1"))
			index.Columns = make([]string, 0, strings.Count(columns, ",")+1-len(index.IncludeColumns))
			index.Descending = make([]bool, 0, len(index.Columns))
			if columns != "" {
				for i, str := range strings.Split(columns, ",") {
					if i <= len(include) && include[i] {
						index.IncludeColumns = append(index.IncludeColumns, str)
					} else {
						index.Columns = append(index.Columns, str)
					}
				}
			}
			if descending != "" {
				for _, str := range strings.Split(descending, ",") {
					b, _ := strconv.ParseBool(str)
					index.Descending = append(index.Descending, b)
				}
				index.Descending = index.Descending[len(index.IncludeColumns):]
			}
		}
		// When deserializing a Catalog from JSON, empty slices are set as nil
		// (because of omitempty). If we want to follow that precedent we have
		// to set empty slices to nil here as well.
		if len(index.Columns) == 0 {
			index.Columns = nil
		}
		if len(index.IncludeColumns) == 0 {
			index.IncludeColumns = nil
		}
		if len(index.Opclasses) == 0 {
			index.Opclasses = nil
		}
		if len(index.Descending) == 0 {
			index.Descending = nil
		}
		index.SQL = strings.ReplaceAll(index.SQL, "\r\n", "\n")
		indexes = append(indexes, index)
	}
	return indexes, closeRows(rows)
}

// GetRoutines returns the routines (functions and procedures) in the database.
//
// To search for specific routines, add the routine names into the
// DatabaseIntrospector.Filter.Routines slice. To exclude specific routines
// from your search, add the routine names into the
// DatabaseIntrospector.Filter.ExcludeRoutines slice.
//
// To narrow down your search to a specific schema, pass the schema name into
// the DatabaseIntrospector.Filter.Schemas slice.
func (dbi *DatabaseIntrospector) GetRoutines() ([]Routine, error) {
	ctx := context.Background()
	var err error
	var rows *sql.Rows
	switch dbi.Dialect {
	case DialectSQLite:
		return nil, nil
	case DialectPostgres:
		rows, err = dbi.queryContext(ctx, "introspection_scripts/postgres_routines.sql")
		if err != nil {
			return nil, err
		}
	case DialectMySQL:
		rows, err = dbi.queryContext(ctx, "introspection_scripts/mysql_routines.sql")
		if err != nil {
			return nil, err
		}
	case DialectSQLServer:
		rows, err = dbi.queryContext(ctx, "introspection_scripts/sqlserver_routines.sql")
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported dialect: %s", dbi.Dialect)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var routines []Routine
	for rows.Next() {
		var routine Routine
		switch dbi.Dialect {
		case DialectPostgres:
			var returnType string
			err = rows.Scan(
				&routine.RoutineSchema,
				&routine.RoutineName,
				&routine.RoutineType,
				&routine.IdentityArguments,
				&routine.SQL,
				&returnType,
				&routine.Comment,
			)
			if err != nil {
				return nil, fmt.Errorf("scanning Routine: %w", err)
			}
			routine.Attrs = map[string]string{
				"returnType": returnType,
			}
		case DialectMySQL:
			var returnType, parameters, routineBody, sqlDataAccess, securityType string
			var isDeterministic bool
			err = rows.Scan(
				&routine.RoutineSchema,
				&routine.RoutineName,
				&routine.RoutineType,
				&returnType,
				&parameters,
				&routine.SQL,
				&routine.Comment,
				&routineBody,
				&isDeterministic,
				&sqlDataAccess,
				&securityType,
			)
			routine.Attrs = map[string]string{
				"returnType":      returnType,
				"parameters":      parameters,
				"routineBody":     routineBody,
				"isDeterministic": strconv.FormatBool(isDeterministic),
				"sqlDataAccess":   sqlDataAccess,
				"securityType":    securityType,
			}
			if err != nil {
				return nil, fmt.Errorf("scanning Routine: %w", err)
			}
		case DialectSQLServer:
			err = rows.Scan(
				&routine.RoutineSchema,
				&routine.RoutineName,
				&routine.RoutineType,
				&routine.SQL,
			)
			if err != nil {
				return nil, fmt.Errorf("scanning Routine: %w", err)
			}
		}
		routine.SQL = strings.ReplaceAll(routine.SQL, "\r\n", "\n")
		routines = append(routines, routine)
	}
	return routines, closeRows(rows)
}

// GetTables returns the tables in the database. It does not automatically
// fetch the columns, constraints and indexes for you. You must fetch them
// yourself. Look into CatalogCache for a streamlined way of adding columns
// from different tables into their corresponding table structs.
//
// To search for specific tables, add the table names into the
// DatabaseIntrospector.Filter.Tables slice. To exclude specific tables from
// your search, add the table names into the
// DatabaseIntrospector.Filter.ExcludeTables slice.
//
// To narrow down your search to a specific schema, pass the schema name into
// the DatabaseIntrospector.Filter.Schemas slice.
func (dbi *DatabaseIntrospector) GetTables() ([]Table, error) {
	ctx := context.Background()
	var err error
	var rows *sql.Rows
	switch dbi.Dialect {
	case DialectSQLite:
		rows, err = dbi.queryContext(ctx, "introspection_scripts/sqlite_tables.sql")
		if err != nil {
			return nil, err
		}
	case DialectPostgres:
		rows, err = dbi.queryContext(ctx, "introspection_scripts/postgres_tables.sql")
		if err != nil {
			return nil, err
		}
	case DialectMySQL:
		rows, err = dbi.queryContext(ctx, "introspection_scripts/mysql_tables.sql")
		if err != nil {
			return nil, err
		}
	case DialectSQLServer:
		rows, err = dbi.queryContext(ctx, "introspection_scripts/sqlserver_tables.sql")
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported dialect: %s", dbi.Dialect)
	}
	defer rows.Close()
	var tables []Table
	for rows.Next() {
		var table Table
		switch dbi.Dialect {
		case DialectSQLite:
			err = rows.Scan(&table.TableName, &table.SQL)
			if err != nil {
				return nil, fmt.Errorf("scanning Table: %w", err)
			}
		case DialectPostgres, DialectMySQL:
			err = rows.Scan(&table.TableSchema, &table.TableName, &table.Comment)
			if err != nil {
				return nil, fmt.Errorf("scanning Table: %w", err)
			}
		case DialectSQLServer:
			err = rows.Scan(&table.TableSchema, &table.TableName)
			if err != nil {
				return nil, fmt.Errorf("scanning Table: %w", err)
			}
		}
		table.SQL = strings.ReplaceAll(table.SQL, "\r\n", "\n")
		tables = append(tables, table)
	}
	return tables, closeRows(rows)
}

// GetTriggers returns the triggers in the database.
//
// To narrow down your search to a specific schema and table, pass the schema
// and table names into the DatabaseIntrospector.Filter.Schemas slice and
// the DatabaseIntrospector.Filter.Tables slice respectively.
func (dbi *DatabaseIntrospector) GetTriggers() ([]Trigger, error) {
	ctx := context.Background()
	var err error
	var rows *sql.Rows
	switch dbi.Dialect {
	case DialectSQLite:
		rows, err = dbi.queryContext(ctx, "introspection_scripts/sqlite_triggers.sql")
		if err != nil {
			return nil, err
		}
	case DialectPostgres:
		rows, err = dbi.queryContext(ctx, "introspection_scripts/postgres_triggers.sql")
		if err != nil {
			return nil, err
		}
	case DialectMySQL:
		rows, err = dbi.queryContext(ctx, "introspection_scripts/mysql_triggers.sql")
		if err != nil {
			return nil, err
		}
	case DialectSQLServer:
		rows, err = dbi.queryContext(ctx, "introspection_scripts/sqlserver_triggers.sql")
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported dialect: %s", dbi.Dialect)
	}
	defer rows.Close()
	var triggers []Trigger
	for rows.Next() {
		var trigger Trigger
		switch dbi.Dialect {
		case DialectSQLite:
			err = rows.Scan(&trigger.TableName, &trigger.TriggerName, &trigger.SQL)
			if err != nil {
				return nil, fmt.Errorf("scanning Trigger: %w", err)
			}
		case DialectPostgres:
			err = rows.Scan(
				&trigger.TableSchema,
				&trigger.TableName,
				&trigger.TriggerName,
				&trigger.IsViewTrigger,
				&trigger.SQL,
				&trigger.Comment,
			)
			if err != nil {
				return nil, fmt.Errorf("scanning Trigger: %w", err)
			}
		case DialectSQLServer:
			err = rows.Scan(
				&trigger.TableSchema,
				&trigger.TableName,
				&trigger.TriggerName,
				&trigger.IsViewTrigger,
				&trigger.SQL,
			)
			if err != nil {
				return nil, fmt.Errorf("scanning Trigger: %w", err)
			}
		case DialectMySQL:
			var actionTiming, eventManipulation string
			err = rows.Scan(
				&trigger.TableSchema,
				&trigger.TableName,
				&trigger.TriggerName,
				&trigger.SQL,
				&actionTiming,
				&eventManipulation,
			)
			if err != nil {
				return nil, fmt.Errorf("scanning Trigger: %w", err)
			}
			trigger.Attrs = map[string]string{
				"actionTiming":      actionTiming,
				"eventManipulation": eventManipulation,
			}
		}
		trigger.SQL = strings.ReplaceAll(trigger.SQL, "\r\n", "\n")
		triggers = append(triggers, trigger)
	}
	return triggers, nil
}

// GetViews returns the views in the database.
//
// To search for specific views, add the view names into the
// DatabaseIntrospector.Filter.Views slice. To exclude specific views from
// your search, add the view names into the
// DatabaseIntrospector.Filter.ExcludeViews slice.
//
// To narrow down your search to a specific schema, pass the schema name into
// the DatabaseIntrospector.Filter.Schemas slice.
func (dbi *DatabaseIntrospector) GetViews() ([]View, error) {
	ctx := context.Background()
	var err error
	var rows *sql.Rows
	switch dbi.Dialect {
	case DialectSQLite:
		rows, err = dbi.queryContext(ctx, "introspection_scripts/sqlite_views.sql")
		if err != nil {
			return nil, err
		}
	case DialectPostgres:
		rows, err = dbi.queryContext(ctx, "introspection_scripts/postgres_views.sql")
		if err != nil {
			return nil, err
		}
	case DialectMySQL:
		rows, err = dbi.queryContext(ctx, "introspection_scripts/mysql_views.sql")
		if err != nil {
			return nil, err
		}
	case DialectSQLServer:
		rows, err = dbi.queryContext(ctx, "introspection_scripts/sqlserver_views.sql")
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported dialect: %s", dbi.Dialect)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var views []View
	for rows.Next() {
		var view View
		switch dbi.Dialect {
		case DialectSQLite:
			var s1, s2 string
			err = rows.Scan(&view.ViewName, &view.SQL, &s1, &s2)
			if err != nil {
				return nil, fmt.Errorf("scanning View: %w", err)
			}
			if s1 != "" {
				view.Columns = strings.Split(s1, "|")
			}
			if s2 != "" {
				view.ColumnTypes = strings.Split(s2, "|")
			}
		case DialectPostgres:
			var s1, s2, s3 string
			err = rows.Scan(
				&view.ViewSchema,
				&view.ViewName,
				&view.IsMaterialized,
				&view.SQL,
				&s1,
				&s2,
				&s3,
				&view.Comment,
			)
			if err != nil {
				return nil, fmt.Errorf("scanning View: %w", err)
			}
			if s1 != "" {
				view.Columns = strings.Split(s1, "|")
			}
			if s2 != "" {
				view.ColumnTypes = strings.Split(s2, "|")
			}
			if s3 != "" {
				for i, str := range strings.Split(s3, "|") {
					isEnum, _ := strconv.ParseBool(str)
					if isEnum && i < len(view.Columns) {
						view.EnumColumns = append(view.EnumColumns, view.Columns[i])
					}
				}
			}
		case DialectMySQL, DialectSQLServer:
			var s1, s2 string
			err = rows.Scan(&view.ViewSchema, &view.ViewName, &view.SQL, &s1, &s2)
			if err != nil {
				return nil, fmt.Errorf("scanning View: %w", err)
			}
			if s1 != "" {
				view.Columns = strings.Split(s1, "|")
			}
			if s2 != "" {
				view.ColumnTypes = strings.Split(s2, "|")
			}
		}
		view.SQL = strings.ReplaceAll(view.SQL, "\r\n", "\n")
		views = append(views, view)
	}
	return views, closeRows(rows)
}

// queryContext works like (*sql.DB).QueryContext except instead of taking in a
// query string, it takes in a filename in order to read the contents from and
// convert into a query string.
func (dbi *DatabaseIntrospector) queryContext(ctx context.Context, filename string) (*sql.Rows, error) {
	var err error
	tmpl := templates.Lookup(filename)
	if tmpl == nil {
		b, err := fs.ReadFile(embedFS, filename)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", filename, err)
		}
		tmpl, err = template.
			New(filename).
			Funcs(template.FuncMap{"mklist": mklist}).
			Parse(string(b))
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", filename, err)
		}
		go func() {
			_, _ = templates.AddParseTree(filename, tmpl.Tree)
		}()
	}
	buf := bufpool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufpool.Put(buf)
	err = tmpl.Execute(buf, &dbi.Filter)
	if err != nil {
		return nil, fmt.Errorf("executing %s: %w", filename, err)
	}
	query := buf.String()
	rows, err := dbi.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s:\n%s: %w", filename, query, err)
	}
	return rows, nil
}

func closeRows(rows *sql.Rows) error {
	err := rows.Close()
	if err != nil {
		return fmt.Errorf("rows.Close: %w", err)
	}
	err = rows.Err()
	if err != nil {
		return fmt.Errorf("rows.Err: %w", err)
	}
	return nil
}

// Filter is a struct used by DatabaseIntrospector in order to narrow down its
// search.
type Filter struct {
	// VersionNums holds the version number of the underlying database
	// connection. This is used by the query templates to output different
	// queries based on the database version. If no version number is found,
	// the highest version number possible is assumed. You can use
	// DatabaseIntrospector.GetVersionNums to populate it.
	VersionNums VersionNums

	// IncludeSystemCatalogs controls whether the DatabaseIntrospector will
	// include the system tables in its search (information_schema, pg_catalog,
	// etc). Default is false.
	IncludeSystemCatalogs bool

	// ConstraintTypes controls what constraint types will be included in the
	// search. An empty slice means all constraint types will be included. The
	// possible constraint types are: "PRIMARY KEY", "UNIQUE", "FOREIGN KEY",
	// "CHECK" and "EXCLUDE".
	ConstraintTypes []string

	// ObjectTypes controls what object types will be included in the search.
	// An empty slice means all object types will be included. The possible
	// object types are: "EXTENSIONS", "ENUMS", "DOMAINS", "ROUTINES", "VIEWS"
	// and "TABLES".
	ObjectTypes []string

	// Tables is the list of tables to be included in the search. If empty, all
	// tables will be included.
	Tables []string

	// Schemas is the list of schemas to include in the search. If empty, all
	// schemas will be included.
	Schemas []string

	// ExcludeSchemas is the list of schemas to exclude from the search.
	ExcludeSchemas []string

	// ExcludeTables is the list of tables to be excluded from the search.
	ExcludeTables []string

	// Views is the list of views to be included in the search. If empty, all
	// views will be included.
	Views []string

	// Routines is the list of routines to be included in the search. If empty,
	// all routines will be included.
	Routines []string

	// ExcludeRoutines is the list of routines to be excluded from the search.
	ExcludeRoutines []string

	// ExcludeViews is the list of views to be excluded from the search.
	ExcludeViews []string

	// Enums is the list of enums to be included in the search. If empty, all
	// enums will be included.
	Enums []string

	// ExcludeEnums is the list of enums to be excluded from the search.
	ExcludeEnums []string

	// Domains is the list of domains to be included in the search. If empty,
	// all domains will be included.
	Domains []string

	// ExcludeDomains is the list of domains to be excluded from the search.
	ExcludeDomains []string

	// Extensions is the list of extensions to include in the search by
	// DatabaseIntrospector.GetExtensions. If empty, all extensions will be
	// included.
	Extensions []string

	// ExcludeExtensions is the list of extensions to exclude from the search.
	ExcludeExtensions []string
}

// IncludeConstraintType returns a bool indicating if the constraintType should
// be included in the search.
func (f *Filter) IncludeConstraintType(constraintType string) bool {
	if len(f.ConstraintTypes) == 0 {
		return true
	}
	for _, t := range f.ConstraintTypes {
		if t == constraintType {
			return true
		}
	}
	return false
}

// mklist converts a slice of strings into a comma separated list of SQL
// strings.
func mklist(strs []string) string {
	// calculate number of bytes needed in final string
	var n int
	for _, str := range strs {
		if str == "" {
			continue
		}
		n += 2                       // 2 wrapper quotes `'` + `'`
		n += 2                       // 2 char delim `, `
		n += strings.Count(str, `'`) // each internal quote `'` is escaped to `''`
		n += len(str)
	}

	var b strings.Builder
	b.Grow(n)
	written := false
	for _, str := range strs {
		if str == "" {
			continue
		}
		if !written {
			written = true
		} else {
			b.WriteString(`, `)
		}
		b.WriteString(`'` + sq.EscapeQuote(str, '\'') + `'`)
	}
	return b.String()
}
