package ddl

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bokwoon95/sq"
)

// Subsetter is used to dump a referentially-intact subset of the database.
type Subsetter struct {
	db           sq.DB
	dialect      string
	inMemory     bool
	suffix       string
	catalog      *Catalog
	cache        *CatalogCache
	results      []queryResult
	referencedBy map[*Table][]*Constraint
	tableResults map[[2]string][]int
	tableJoins   map[[2]string][]join
}

// queryResult holds the query results of a subset query (in memory).
type queryResult struct {
	columns     []string
	columnTypes []string
	records     [][]string
}

// TODO: export this once you get the temp table version working.
func newSubsetter(dialect string, db sq.DB, filter Filter) (*Subsetter, error) {
	ss := &Subsetter{
		dialect:      dialect,
		db:           db,
		suffix:       time.Now().UTC().Format("20060102150405"),
		catalog:      &Catalog{},
		referencedBy: make(map[*Table][]*Constraint),
		tableResults: make(map[[2]string][]int),
		tableJoins:   make(map[[2]string][]join),
	}
	dbi := NewDatabaseIntrospector(dialect, db)
	dbi.Filter = filter
	dbi.ObjectTypes = []string{"TABLES"}
	dbi.ConstraintTypes = []string{PRIMARY_KEY, FOREIGN_KEY}
	err := dbi.WriteCatalog(ss.catalog)
	if err != nil {
		return nil, err
	}
	ss.cache = NewCatalogCache(ss.catalog)
	for i := range ss.catalog.Schemas {
		schema := &ss.catalog.Schemas[i]
		for j := range schema.Tables {
			table := &schema.Tables[j]
			for k := range table.Constraints {
				constraint := &table.Constraints[k]
				if constraint.ConstraintType != FOREIGN_KEY {
					continue
				}
				referencedSchema := ss.cache.GetSchema(ss.catalog, constraint.ReferencesSchema)
				referencedTable := ss.cache.GetTable(referencedSchema, constraint.ReferencesTable)
				ss.referencedBy[referencedTable] = append(ss.referencedBy[referencedTable], constraint)
			}
		}
	}
	return ss, nil
}

// NewInMemorySubsetter creates an in-memory subsetter that holds the results
// of each subset query in-memory.
func NewInMemorySubsetter(dialect string, db sq.DB, filter Filter) (*Subsetter, error) {
	ss, err := newSubsetter(dialect, db, filter)
	if err != nil {
		return nil, err
	}
	ss.inMemory = true
	return ss, nil
}

// Subset adds a new subset query to the subsetter.
func (ss *Subsetter) Subset(query string) error {
	return ss.subset(query, false)
}

// ExtendedSubset adds a new extended subset query to the subsetter.
func (ss *Subsetter) ExtendedSubset(query string) error {
	return ss.subset(query, true)
}

func (ss *Subsetter) subset(query string, extended bool) error {
	const (
		OPEN_BRACE  = '{'
		CLOSE_BRACE = '}'
	)
	starStart, starEnd := -1, -1
	tableStart, tableEnd := -1, -1
	var result queryResult
	str := query
	for i := strings.IndexByte(str, OPEN_BRACE); i >= 0; i = strings.IndexByte(str, OPEN_BRACE) {
		if i+1 <= len(str) && str[i+1] == OPEN_BRACE {
			str = str[i+2:]
			continue
		}
		str = str[i:]
		j := strings.IndexByte(str, CLOSE_BRACE)
		if j < 0 {
			return fmt.Errorf("no %q found", CLOSE_BRACE)
		}
		if str[:j+1] == "{*}" {
			if starStart > 0 {
				return fmt.Errorf("more than one {*}")
			}
			starStart = len(query) - len(str)
			starEnd = starStart + j + 1
			str = str[j+1:]
			continue
		}
		if tableStart > 0 {
			return fmt.Errorf("more than one {tableName}")
		}
		tableStart = len(query) - len(str)
		tableEnd = tableStart + j + 1
		str = str[j+1:]
	}
	var tableSchema string
	tableName := query[tableStart+1 : tableEnd-1]
	if i := strings.Index(tableName, "."); i >= 0 {
		tableSchema, tableName = tableName[:i], tableName[i+1:]
	}
	if tableSchema == "" {
		tableSchema = ss.catalog.CurrentSchema
	}
	baseTable := ss.cache.GetTable(ss.cache.GetSchema(ss.catalog, tableSchema), tableName)
	if baseTable == nil {
		return fmt.Errorf("no table found for %s", query[tableStart:tableEnd])
	}
	pkey := ss.cache.GetPrimaryKey(baseTable)
	if pkey == nil {
		return fmt.Errorf("table %q has no primary key", tableName)
	}
	result.columns = pkey.Columns
	result.columnTypes = make([]string, 0, len(pkey.Columns))
	scanDest := make([]any, 0, len(pkey.Columns))
	var b strings.Builder
	b.Grow(len(query) + len(tableSchema) + len(tableName) + 20)
	b.WriteString(query[:starStart])
	for i, columnName := range pkey.Columns {
		if i > 0 {
			b.WriteString(", ")
		}
		column := ss.cache.GetColumn(baseTable, columnName)
		if baseTable.TableSchema != "" {
			b.WriteString(sq.QuoteIdentifier(ss.dialect, baseTable.TableSchema) + ".")
		}
		b.WriteString(sq.QuoteIdentifier(ss.dialect, baseTable.TableName) + "." + sq.QuoteIdentifier(ss.dialect, columnName))
		columnType, _, _ := normalizeColumnType(ss.dialect, column.ColumnType)
		if ss.dialect == sq.DialectMySQL {
			if strings.HasSuffix(columnType, " UNSIGNED") {
				columnType = strings.TrimSuffix(columnType, " UNSIGNED")
			} else {
				columnType = strings.TrimSuffix(columnType, " SIGNED")
			}
		}
		result.columnTypes = append(result.columnTypes, columnType)
		if ss.dialect == sq.DialectSQLite {
			scanDest = append(scanDest, &sql.NullString{})
			continue
		}
		switch columnType {
		case "BYTEA", "BINARY", "VARBINARY", "TINYBLOB", "BLOB", "MEDIUMBLOB", "LONGBLOB", "VARBIT":
			scanDest = append(scanDest, &[]byte{})
		case "BOOLEAN", "BIT":
			scanDest = append(scanDest, &sql.NullBool{})
		case "NUMERIC", "FLOAT", "REAL", "DOUBLE PRECISION":
			scanDest = append(scanDest, &sql.NullFloat64{})
		case "TINYINT", "SMALLINT", "MEDIUMINT", "INT", "INTEGER", "BIGINT":
			scanDest = append(scanDest, &sql.NullInt64{})
		case "TINYTEXT", "TEXT", "MEDIUMTEXT", "LONGTEXT", "CHAR", "VARCHAR", "NVARCHAR", "UUIDField", "UNIQUEIDENTIFIER", "JSON", "JSONB":
			scanDest = append(scanDest, &sql.NullString{})
		case "DATE", "TIME", "TIMETZ", "DATETIME", "DATETIME2", "SMALLDATETIME", "DATETIMEOFFSET", "TIMESTAMP", "TIMESTAMPTZ":
			scanDest = append(scanDest, &sq.Timestamp{})
		default:
			scanDest = append(scanDest, &sql.NullString{})
		}
	}
	b.WriteString(query[starEnd:tableStart])
	if baseTable.TableSchema != "" {
		b.WriteString(sq.QuoteIdentifier(ss.dialect, baseTable.TableSchema) + ".")
	}
	b.WriteString(sq.QuoteIdentifier(ss.dialect, baseTable.TableName) + query[tableEnd:])
	rows, err := ss.db.QueryContext(context.Background(), b.String())
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(scanDest...)
		if err != nil {
			return err
		}
		record := make([]string, len(scanDest))
		for i, value := range scanDest {
			switch value := value.(type) {
			case *[]byte:
				if *value == nil {
					record[i] = "NULL"
					continue
				}
				switch ss.dialect {
				case sq.DialectPostgres:
					record[i] = "'\\x" + hex.EncodeToString(*value) + "'"
				case sq.DialectSQLServer:
					record[i] = "0x" + hex.EncodeToString(*value)
				default:
					record[i] = "x'" + hex.EncodeToString(*value) + "'"
				}
			case *sql.NullBool:
				if !value.Valid {
					record[i] = "NULL"
					continue
				}
				switch ss.dialect {
				case sq.DialectSQLServer:
					if value.Bool {
						record[i] = "1"
					} else {
						record[i] = "0"
					}
				default:
					if value.Bool {
						record[i] = "TRUE"
					} else {
						record[i] = "FALSE"
					}
				}
			case *sql.NullFloat64:
				if !value.Valid {
					record[i] = "NULL"
					continue
				}
				record[i] = strconv.FormatFloat(value.Float64, 'f', -1, 64)
			case *sql.NullInt64:
				if !value.Valid {
					record[i] = "NULL"
					continue
				}
				record[i] = strconv.FormatInt(value.Int64, 10)
			case *sql.NullString:
				if !value.Valid {
					record[i] = "NULL"
					continue
				}
				if ss.dialect == sq.DialectSQLite {
					columnType := result.columnTypes[i]
					switch columnType {
					case "BINARY", "VARBINARY", "TINYBLOB", "BLOB", "MEDIUMBLOB", "LONGBLOB":
						record[i] = "x'" + hex.EncodeToString([]byte(value.String)) + "'"
						continue
					case "BOOLEAN", "BIT",
						"NUMERIC", "FLOAT", "REAL", "DOUBLE PRECISION",
						"TINYINT", "SMALLINT", "MEDIUMINT", "INT", "INTEGER", "BIGINT":
						record[i] = value.String
						continue
					case "UUID":
						if len(value.String) == 16 {
							record[i] = "x'" + hex.EncodeToString([]byte(value.String)) + "'"
							continue
						}
					case "DATE", "TIME", "TIMETZ", "DATETIME", "DATETIME2", "SMALLDATETIME", "DATETIMEOFFSET", "TIMESTAMP", "TIMESTAMPTZ":
						if _, err := strconv.ParseInt(value.String, 10, 64); err == nil {
							record[i] = value.String
							continue
						} else if _, err = strconv.ParseFloat(value.String, 64); err == nil {
							record[i] = value.String
							continue
						}
					}
				}
				record[i] = "'" + sq.EscapeQuote(value.String, '\'') + "'"
			case *sq.Timestamp:
				if !value.Valid {
					record[i] = "NULL"
					continue
				}
				if value.Time.Sub(value.Time.Round(time.Millisecond)) != 0 {
					record[i] = "'" + value.Time.UTC().Format("2006-01-02 15:04:05.000") + "'"
				} else {
					record[i] = "'" + value.Time.UTC().Format("2006-01-02 15:04:05") + "'"
				}
			default:
				panic("unreachable")
			}
		}
		result.records = append(result.records, record)
	}
	tableID := [2]string{baseTable.TableSchema, baseTable.TableName}
	ss.results = append(ss.results, result)
	baseFkey := &Constraint{
		ConstraintType:    FOREIGN_KEY,
		Columns:           pkey.Columns,
		ReferencesSchema:  baseTable.TableSchema,
		ReferencesTable:   baseTable.TableName,
		ReferencesColumns: pkey.Columns,
	}
	if ss.inMemory {
		baseFkey.TableName = "query" + strconv.Itoa(len(ss.results)) + "_"
	} else {
		baseFkey.TableSchema = tableSchema + "_" + ss.suffix
		baseFkey.TableName = tableName
	}
	tableJoins := map[[2]string][]join{
		tableID: {{
			tableIDs: [][2]string{tableID},
			fkeys:    []*Constraint{baseFkey},
		}},
	}
	var toVisit []*Constraint
	visited := make(map[*Constraint]struct{}) // In order to break cycles, never visit the same fkey twice.
	if !extended {
		for i := range baseTable.Constraints {
			toVisit = append(toVisit, &baseTable.Constraints[i])
		}
	} else {
		toVisit = append(toVisit, ss.referencedBy[baseTable]...)
		for len(toVisit) > 0 {
			constraint := toVisit[len(toVisit)-1]
			toVisit = toVisit[:len(toVisit)-1]
			if constraint.ConstraintType != FOREIGN_KEY {
				continue
			}
			if _, ok := visited[constraint]; ok {
				continue
			}
			visited[constraint] = struct{}{}
			tableID := [2]string{constraint.TableSchema, constraint.TableName}
			referencedTableID := [2]string{constraint.ReferencesSchema, constraint.ReferencesTable}
			baseJoins := tableJoins[referencedTableID]
			newJoins := make([]join, len(baseJoins))
			i := 0
			for _, baseJoin := range baseJoins {
				duplicateTable := false
				for j := range baseJoin.tableIDs {
					if tableID == baseJoin.tableIDs[j] {
						duplicateTable = true
						break
					}
				}
				if duplicateTable {
					continue
				}
				newJoins[i].fkeys = make([]*Constraint, len(baseJoin.fkeys), len(baseJoin.fkeys)+1)
				newJoins[i].tableIDs = make([][2]string, len(baseJoin.tableIDs), len(baseJoin.tableIDs)+1)
				copy(newJoins[i].fkeys, baseJoin.fkeys)
				copy(newJoins[i].tableIDs, baseJoin.tableIDs)
				newJoins[i].fkeys = append(newJoins[i].fkeys, constraint)
				newJoins[i].tableIDs = append(newJoins[i].tableIDs, tableID)
				i++
			}
			newJoins = newJoins[:i]
			tableJoins[tableID] = append(tableJoins[tableID], newJoins...)
			table := ss.cache.GetTable(ss.cache.GetSchema(ss.catalog, constraint.TableSchema), constraint.TableName)
			toVisit = append(toVisit, ss.referencedBy[table]...)
		}
		tableIDs := make([][2]string, 0, len(tableJoins))
		for tableID := range tableJoins {
			tableIDs = append(tableIDs, tableID)
		}
		sort.Slice(tableIDs, func(i, j int) bool {
			for n := 0; n < 2; n++ {
				if tableIDs[i][n] < tableIDs[j][n] {
					return true
				}
				if tableIDs[i][n] > tableIDs[j][n] {
					return false
				}
			}
			return false
		})
		for _, tableID := range tableIDs {
			table := ss.cache.GetTable(ss.cache.GetSchema(ss.catalog, tableID[0]), tableID[1])
			for i := range table.Constraints {
				toVisit = append(toVisit, &table.Constraints[i])
			}
		}
	}
	for len(toVisit) > 0 {
		constraint := toVisit[len(toVisit)-1]
		toVisit = toVisit[:len(toVisit)-1]
		if constraint.ConstraintType != FOREIGN_KEY {
			continue
		}
		if _, ok := visited[constraint]; ok {
			continue
		}
		visited[constraint] = struct{}{}
		tableID := [2]string{constraint.TableSchema, constraint.TableName}
		referencedTableID := [2]string{constraint.ReferencesSchema, constraint.ReferencesTable}
		baseJoins := tableJoins[tableID]
		newJoins := make([]join, len(baseJoins))
		i := 0
		for _, baseJoin := range baseJoins {
			duplicateTable := false
			for j := range baseJoin.tableIDs {
				if referencedTableID == baseJoin.tableIDs[j] {
					duplicateTable = true
					break
				}
			}
			if duplicateTable {
				continue
			}
			newJoins[i].fkeys = make([]*Constraint, len(baseJoin.fkeys), len(baseJoin.fkeys)+1)
			newJoins[i].tableIDs = make([][2]string, len(baseJoin.tableIDs), len(baseJoin.tableIDs)+1)
			copy(newJoins[i].fkeys, baseJoin.fkeys)
			copy(newJoins[i].tableIDs, baseJoin.tableIDs)
			newJoins[i].fkeys = append(newJoins[i].fkeys, constraint)
			newJoins[i].tableIDs = append(newJoins[i].tableIDs, referencedTableID)
			i++
		}
		newJoins = newJoins[:i]
		tableJoins[referencedTableID] = append(tableJoins[referencedTableID], newJoins...)
		table := ss.cache.GetTable(ss.cache.GetSchema(ss.catalog, constraint.ReferencesSchema), constraint.ReferencesTable)
		for i := range table.Constraints {
			toVisit = append(toVisit, &table.Constraints[i])
		}
	}
	for tableID, joins := range tableJoins {
		ss.tableResults[tableID] = append(ss.tableResults[tableID], len(ss.results)-1)
		ss.tableJoins[tableID] = append(ss.tableJoins[tableID], joins...)
	}
	return nil
}

// Query returns the query needed to dump a subset of a table according to the
// subset queries added to the subsetter thus far.
func (ss *Subsetter) Query(tableSchema, tableName string) string {
	if tableSchema == "" {
		tableSchema = ss.catalog.CurrentSchema
	}
	table := ss.cache.GetTable(ss.cache.GetSchema(ss.catalog, tableSchema), tableName)
	if table == nil {
		return ""
	}
	fullTableName := sq.QuoteIdentifier(ss.dialect, table.TableName)
	if table.TableSchema != "" {
		fullTableName = sq.QuoteIdentifier(ss.dialect, table.TableSchema) + "." + fullTableName
	}
	pkey := ss.cache.GetPrimaryKey(table)
	if pkey == nil {
		return ""
	}
	var pkeyColumns string
	{
		var b strings.Builder
		for i, column := range pkey.Columns {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(sq.QuoteIdentifier(ss.dialect, column))
		}
		pkeyColumns = b.String()
	}
	var qualifiedPkeyColumns string
	{
		var b strings.Builder
		for i, column := range pkey.Columns {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(fullTableName + "." + sq.QuoteIdentifier(ss.dialect, column))
		}
		qualifiedPkeyColumns = b.String()
	}
	buf := bufpool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufpool.Put(buf)
	// if ss.inMemory
	nums := ss.tableResults[[2]string{tableSchema, tableName}]
	if len(nums) == 0 {
		return ""
	}
	for _, num := range nums {
		result := ss.results[num]
		if buf.Len() == 0 {
			buf.WriteString("WITH ")
		} else {
			buf.WriteString(",")
		}
		var resultColumns string
		{
			var b strings.Builder
			for j, column := range result.columns {
				if j > 0 {
					b.WriteString(", ")
				}
				b.WriteString(sq.QuoteIdentifier(ss.dialect, column))
			}
			resultColumns = b.String()
		}
		buf.WriteString("query" + strconv.Itoa(num+1) + "_ ")
		if ss.dialect == sq.DialectSQLServer {
			buf.WriteString(" AS (\n    SELECT * FROM (VALUES ")
		} else {
			buf.WriteString("(" + resultColumns + ") AS (\n    VALUES ")
		}
		n := buf.Len()
		for j, record := range result.records {
			if buf.Len()-n > 80 {
				n = buf.Len()
				buf.WriteString("\n       ")
			}
			if j > 0 {
				buf.WriteString(",")
			}
			if ss.dialect == sq.DialectMySQL {
				buf.WriteString("ROW")
			}
			buf.WriteString("(")
			for k, field := range record {
				if k > 0 {
					buf.WriteString(", ")
				}
				buf.WriteString(field)
			}
			buf.WriteString(")")
		}
		if ss.dialect == sq.DialectSQLServer {
			buf.WriteString(") AS tmp (" + resultColumns + ")")
		}
		buf.WriteString("\n)\n")
	}
	if buf.Len() == 0 {
		buf.WriteString("WITH ")
	} else {
		buf.WriteString(",")
	}
	name := "pkey_"
	buf.WriteString("pkey_ (" + pkeyColumns + ") AS (")
	joins := ss.tableJoins[[2]string{tableSchema, tableName}]
	for i, join := range joins {
		if i == 0 {
			buf.WriteString("\n    ")
		} else {
			buf.WriteString("\n    UNION\n    ")
		}
		buf.WriteString("SELECT DISTINCT " + qualifiedPkeyColumns + "\n    FROM " + join.toSQL(ss.dialect))
	}
	buf.WriteString("\n)\nSELECT ")
	written := false
	for _, column := range table.Columns {
		if column.GeneratedExpr != "" || column.IsGenerated {
			continue
		}
		if !written {
			written = true
		} else {
			buf.WriteString("\n    ,")
		}
		buf.WriteString(fullTableName + "." + sq.QuoteIdentifier(ss.dialect, column.ColumnName))
	}
	buf.WriteString("\nFROM " + fullTableName + "\nJOIN " + name + " ON ")
	for i, column := range pkey.Columns {
		if i > 0 {
			buf.WriteString(" AND ")
		}
		columnName := sq.QuoteIdentifier(ss.dialect, column)
		buf.WriteString(name + "." + columnName + " = " + fullTableName + "." + columnName)
	}
	if len(pkey.Columns) > 0 {
		buf.WriteString("\nORDER BY ")
		for i, column := range pkey.Columns {
			if i > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(fullTableName + "." + sq.QuoteIdentifier(ss.dialect, column))
		}
	}
	buf.WriteString("\n;")
	return buf.String()
}

// Tables returns the tables that are involved in the subset dump.
func (ss *Subsetter) Tables() []*Table {
	tableIDs := make([][2]string, 0, len(ss.tableJoins))
	for tableID := range ss.tableJoins {
		tableIDs = append(tableIDs, tableID)
	}
	sort.Slice(tableIDs, func(i, j int) bool {
		for n := 0; n < 2; n++ {
			if tableIDs[i][n] < tableIDs[j][n] {
				return true
			}
			if tableIDs[i][n] > tableIDs[j][n] {
				return false
			}
		}
		return false
	})
	tables := make([]*Table, 0, len(tableIDs))
	for _, tableID := range tableIDs {
		table := ss.cache.GetTable(ss.cache.GetSchema(ss.catalog, tableID[0]), tableID[1])
		if ss.cache.GetPrimaryKey(table) == nil {
			continue
		}
		tables = append(tables, table)
	}
	return tables
}

// TODO: export this once you get the temp table version working.
// tempSchemas returns the list of temporary schemas created to hold the subset
// query results.
func (ss *Subsetter) tempSchemas() []string {
	if ss.inMemory {
		return nil
	}
	schemasMap := make(map[string]struct{})
	for tableID := range ss.tableJoins {
		schemasMap[tableID[0]] = struct{}{}
	}
	schemas := make([]string, 0, len(schemasMap))
	for schema := range schemasMap {
		schemas = append(schemas, schema)
	}
	sort.Strings(schemas)
	tempDir := os.TempDir()
	for i, schema := range schemas {
		if ss.dialect == sq.DialectSQLite {
			schemas[i] = tempDir + string(filepath.Separator) + schema + "_" + ss.suffix + ".sqlite3"
		} else {
			schemas[i] = schema + "_" + ss.suffix
		}
	}
	return schemas
}

// join represents a chain of tables joined by foreign keys.
type join struct {
	tableIDs [][2]string
	fkeys    []*Constraint
}

// toSQL converts the join into SQL.
func (j join) toSQL(dialect string) string {
	buf := bufpool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufpool.Put(buf)
	tableID := j.tableIDs[len(j.tableIDs)-1]
	if tableID[0] != "" {
		buf.WriteString(sq.QuoteIdentifier(dialect, tableID[0]) + ".")
	}
	buf.WriteString(sq.QuoteIdentifier(dialect, tableID[1]))
	var schema1, table1 string
	var columns1, columns2 []string
	schema2, table2 := tableID[0], tableID[1]
	for i := len(j.fkeys) - 1; i >= 0; i-- {
		fkey := j.fkeys[i]
		if schema2 == fkey.TableSchema && table2 == fkey.TableName {
			schema1, table1, columns1 = fkey.TableSchema, fkey.TableName, fkey.Columns
			schema2, table2, columns2 = fkey.ReferencesSchema, fkey.ReferencesTable, fkey.ReferencesColumns
		} else if schema2 == fkey.ReferencesSchema && table2 == fkey.ReferencesTable {
			schema1, table1, columns1 = fkey.ReferencesSchema, fkey.ReferencesTable, fkey.ReferencesColumns
			schema2, table2, columns2 = fkey.TableSchema, fkey.TableName, fkey.Columns
		} else {
			panic("broken chain")
		}
		buf.WriteString("\n    JOIN ")
		if schema2 != "" {
			buf.WriteString(sq.QuoteIdentifier(dialect, schema2) + ".")
		}
		buf.WriteString(sq.QuoteIdentifier(dialect, table2) + " ON ")
		for i := range columns1 {
			column1 := columns1[i]
			column2 := columns2[i]
			if i > 0 {
				buf.WriteString(" AND ")
			}
			if schema2 != "" {
				buf.WriteString(sq.QuoteIdentifier(dialect, schema2) + ".")
			}
			buf.WriteString(sq.QuoteIdentifier(dialect, table2) + "." + sq.QuoteIdentifier(dialect, column2) + " = ")
			if schema1 != "" {
				buf.WriteString(sq.QuoteIdentifier(dialect, schema1) + ".")
			}
			buf.WriteString(sq.QuoteIdentifier(dialect, table1) + "." + sq.QuoteIdentifier(dialect, column1))
		}
	}
	return buf.String()
}
