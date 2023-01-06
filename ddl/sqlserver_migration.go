package ddl

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/bokwoon95/sq"
)

type sqlserverMigration struct {
	versionNums      VersionNums
	currentSchema    string
	defaultCollation string
	warnings         []string

	// 1. Drop the foreign keys.
	dropFkeys [][]*Constraint

	// 2. DROP SCHEMA (plus all objects in it one by one).
	dropSchemas []*Schema

	// 3. Execute all CREATE SCHEMA + DROP TABLE + CREATE TABLE in one transaction.
	createSchemas []string
	dropTables    []*Table
	createTables  []*Table

	// 4. Execute each ALTER TABLE.
	alterTables []sqlserverAlterTable

	// 5. Add the foreign keys for new tables. This should be fast because the
	// new tables are empty (so no rows have to be validated).
	addFastFkeys [][]*Constraint

	// 6. Add foreign keys for existing tables.
	addFkeys [][]*Constraint
}

type sqlserverAlterTable struct {
	tableSchema string
	tableName   string
	columnTypes map[string]string
	pkey        *Constraint

	// Do these in one transaction.
	dropIndexes     []*Index
	dropConstraints []*Constraint
	dropColumns     []*Column
	addColumns      []*Column
	alterColumns    [][2]*Column

	// Create indexes individually outside a transaction.
	createIndexes []*Index

	// Add PRIMARY KEY and UNIQUE constraints individually outside a transaction.
	addConstraints []*Constraint
}

func newSQLServerMigration(srcCatalog, destCatalog *Catalog, dropObjects bool) sqlserverMigration {
	const dialect = DialectSQLServer
	m := sqlserverMigration{
		versionNums:      srcCatalog.VersionNums,
		currentSchema:    srcCatalog.CurrentSchema,
		defaultCollation: srcCatalog.DefaultCollation,
	}
	srcCache, destCache := NewCatalogCache(srcCatalog), NewCatalogCache(destCatalog)
	dropFkeysPos := make(map[[4]string]int)    // Track tablesID position in m.dropFkeys.
	addFastFkeysPos := make(map[[4]string]int) // Track tablesID position in m.dropFastFkeys.
	addFkeysPos := make(map[[4]string]int)     // Track tablesID position in m.addFkeys.

	// getTablesID returns the tablesID for a given constraint.
	getTablesID := func(constraint *Constraint) [4]string {
		tablesID := [4]string{
			constraint.TableSchema,
			constraint.TableName,
			constraint.ReferencesSchema,
			constraint.ReferencesTable,
		}
		// We want {schema1.table1, schema2.table2} to be the same as
		// {schema2.table2, schema1.table1}, so rearrange them and always put
		// the smaller value in front.
		if tablesID[0] > tablesID[2] || tablesID[1] > tablesID[3] {
			tablesID[0], tablesID[2] = tablesID[2], tablesID[0]
			tablesID[1], tablesID[3] = tablesID[3], tablesID[1]
		}
		return tablesID
	}

	if dropObjects {
		for i := range srcCatalog.Schemas {
			srcSchema := &srcCatalog.Schemas[i]
			destSchema := destCache.GetSchema(destCatalog, srcSchema.SchemaName)
			if destSchema == nil {
				// DROP SCHEMA.
				if srcSchema.SchemaName != "" {
					m.dropSchemas = append(m.dropSchemas, srcSchema)
				}
				for j := range srcSchema.Tables {
					// DROP FOREIGN KEY.
					srcTable := &srcSchema.Tables[j]
					srcFkeys := srcCache.GetForeignKeys(srcTable)
					for _, srcFkey := range srcFkeys {
						tablesID := getTablesID(srcFkey)
						if n, ok := dropFkeysPos[tablesID]; ok {
							m.dropFkeys[n] = append(m.dropFkeys[n], srcFkey)
						} else {
							m.dropFkeys = append(m.dropFkeys, []*Constraint{srcFkey})
							n = len(m.dropFkeys) - 1
							dropFkeysPos[tablesID] = n
						}
					}
				}
				continue
			}
		}
	}

	for i := range destCatalog.Schemas {
		destSchema := &destCatalog.Schemas[i]
		srcSchema := srcCache.GetSchema(srcCatalog, destSchema.SchemaName)
		if srcSchema == nil {
			// CREATE SCHEMA.
			if destSchema.SchemaName != "" {
				m.createSchemas = append(m.createSchemas, destSchema.SchemaName)
			}
			for j := range destSchema.Tables {
				destTable := &destSchema.Tables[j]
				// CREATE TABLE.
				m.createTables = append(m.createTables, destTable)
				// ADD FOREIGN KEY.
				destFkeys := destCache.GetForeignKeys(destTable)
				for _, destFkey := range destFkeys {
					tablesID := getTablesID(destFkey)
					if n, ok := addFastFkeysPos[tablesID]; ok {
						m.addFastFkeys[n] = append(m.addFastFkeys[n], destFkey)
					} else {
						m.addFastFkeys = append(m.addFastFkeys, []*Constraint{destFkey})
						n = len(m.addFastFkeys) - 1
						addFastFkeysPos[tablesID] = n
					}
				}
			}
			continue
		}

		if dropObjects {
			for j := range srcSchema.Tables {
				srcTable := &srcSchema.Tables[j]
				destTable := destCache.GetTable(destSchema, srcTable.TableName)
				if destTable == nil {
					// DROP TABLE.
					m.dropTables = append(m.dropTables, srcTable)
					// DROP FOREIGN KEY.
					srcFkeys := srcCache.GetForeignKeys(srcTable)
					for _, srcFkey := range srcFkeys {
						tablesID := getTablesID(srcFkey)
						if n, ok := dropFkeysPos[tablesID]; ok {
							m.dropFkeys[n] = append(m.dropFkeys[n], srcFkey)
						} else {
							m.dropFkeys = append(m.dropFkeys, []*Constraint{srcFkey})
							n = len(m.dropFkeys) - 1
							dropFkeysPos[tablesID] = n
						}
					}
				}
			}
		}

		for j := range destSchema.Tables {
			destTable := &destSchema.Tables[j]
			srcTable := srcCache.GetTable(srcSchema, destTable.TableName)
			if srcTable == nil {
				// CREATE TABLE.
				m.createTables = append(m.createTables, destTable)
				// ADD FOREIGN KEY.
				destFkeys := destCache.GetForeignKeys(destTable)
				for _, destFkey := range destFkeys {
					tablesID := getTablesID(destFkey)
					if n, ok := addFastFkeysPos[tablesID]; ok {
						m.addFastFkeys[n] = append(m.addFastFkeys[n], destFkey)
					} else {
						m.addFastFkeys = append(m.addFastFkeys, []*Constraint{destFkey})
						n = len(m.addFastFkeys) - 1
						addFastFkeysPos[tablesID] = n
					}
				}
				continue
			}

			// ALTER TABLE.
			alterTable := sqlserverAlterTable{
				tableSchema: destTable.TableSchema,
				tableName:   destTable.TableName,
				columnTypes: make(map[string]string),
			}
			if alterTable.tableSchema == "" {
				alterTable.tableSchema = m.currentSchema
			}
			if alterTable.tableSchema == "" {
				alterTable.tableSchema = "dbo"
			}
			droppedIndex := make(map[*Index]bool)
			droppedConstraint := make(map[*Constraint]bool)

			if dropObjects {
				for k := range srcTable.Constraints {
					srcConstraint := &srcTable.Constraints[k]
					destConstraint := destCache.GetConstraint(destTable, srcConstraint.ConstraintName)
					if destConstraint == nil {
						switch srcConstraint.ConstraintType {
						case PRIMARY_KEY, UNIQUE:
							// DROP PRIMARY KEY, DROP UNIQUE.
							alterTable.dropConstraints = append(alterTable.dropConstraints, srcConstraint)
						case FOREIGN_KEY:
							// DROP FOREIGN KEY.
							tablesID := getTablesID(srcConstraint)
							if n, ok := dropFkeysPos[tablesID]; ok {
								m.dropFkeys[n] = append(m.dropFkeys[n], srcConstraint)
							} else {
								m.dropFkeys = append(m.dropFkeys, []*Constraint{srcConstraint})
								n = len(m.dropFkeys) - 1
								dropFkeysPos[tablesID] = n
							}
						}
						droppedConstraint[srcConstraint] = true
					}
				}
				for k := range srcTable.Indexes {
					srcIndex := &srcTable.Indexes[k]
					destIndex := destCache.GetIndex(destTable, srcIndex.IndexName)
					if destIndex == nil {
						// DROP INDEX.
						alterTable.dropIndexes = append(alterTable.dropIndexes, srcIndex)
						droppedIndex[srcIndex] = true
					}
				}
				for k := range srcTable.Columns {
					srcColumn := &srcTable.Columns[k]
					destColumn := destCache.GetColumn(destTable, srcColumn.ColumnName)
					if destColumn == nil {
						// DROP COLUMN.
						alterTable.dropColumns = append(alterTable.dropColumns, srcColumn)
					}
				}
			}

			for k := range destTable.Columns {
				destColumn := &destTable.Columns[k]
				srcColumn := srcCache.GetColumn(srcTable, destColumn.ColumnName)
				alterTable.columnTypes[destColumn.ColumnName] = destColumn.ColumnType
				if srcColumn == nil {
					// ADD COLUMN.
					alterTable.addColumns = append(alterTable.addColumns, destColumn)
					continue
				}
				if srcColumn.ColumnIdentity == "" && destColumn.ColumnIdentity != "" {
					tableName := sq.QuoteIdentifier(dialect, destTable.TableName)
					if destSchema.SchemaName != "" && destSchema.SchemaName != m.currentSchema {
						tableName = sq.QuoteIdentifier(dialect, destSchema.SchemaName) + "."
					}
					columnName := sq.QuoteIdentifier(dialect, destColumn.ColumnName)
					m.warnings = append(m.warnings, fmt.Sprintf("%s: column %s: identity cannot be added to an existing column, skipping", tableName, columnName))
				}
				columnsAreDifferent := func() bool {
					srcType, srcArg1, srcArg2 := normalizeColumnType(dialect, srcColumn.ColumnType)
					destType, destArg1, destArg2 := normalizeColumnType(dialect, destColumn.ColumnType)
					if [3]string{srcType, srcArg1, srcArg2} != [3]string{destType, destArg1, destArg2} {
						return true
					}
					srcDefault := normalizeColumnDefault(dialect, srcColumn.ColumnDefault)
					destDefault := normalizeColumnDefault(dialect, destColumn.ColumnDefault)
					if srcDefault != destDefault {
						return true
					}
					if srcColumn.IsNotNull != destColumn.IsNotNull {
						return true
					}
					srcCollation := srcColumn.CollationName
					if srcCollation == "" {
						srcCollation = m.defaultCollation
					}
					destCollation := destColumn.CollationName
					if destCollation == "" {
						destCollation = m.defaultCollation
					}
					if srcCollation != destCollation {
						return true
					}
					return false
				}()
				if columnsAreDifferent {
					// ALTER COLUMN.
					alterTable.alterColumns = append(alterTable.alterColumns, [2]*Column{srcColumn, destColumn})
				}
			}

			for k := range destTable.Indexes {
				destIndex := &destTable.Indexes[k]
				srcIndex := srcCache.GetIndex(srcTable, destIndex.IndexName)
				if srcIndex == nil {
					// CREATE INDEX.
					alterTable.createIndexes = append(alterTable.createIndexes, destIndex)
				}
			}

			for k := range destTable.Constraints {
				destConstraint := &destTable.Constraints[k]
				srcConstraint := srcCache.GetConstraint(srcTable, destConstraint.ConstraintName)
				if destConstraint.ConstraintType == PRIMARY_KEY {
					alterTable.pkey = destConstraint
				}
				if srcConstraint == nil {
					switch destConstraint.ConstraintType {
					case PRIMARY_KEY, UNIQUE:
						// ADD PRIMARY KEY | ADD UNIQUE.
						alterTable.addConstraints = append(alterTable.addConstraints, destConstraint)
					case FOREIGN_KEY:
						// ADD FOREIGN KEY.
						tablesID := getTablesID(destConstraint)
						if n, ok := addFkeysPos[tablesID]; ok {
							m.addFkeys[n] = append(m.addFkeys[n], destConstraint)
						} else {
							m.addFkeys = append(m.addFkeys, []*Constraint{destConstraint})
							n = len(m.addFkeys) - 1
							addFkeysPos[tablesID] = n
						}
					}
					continue
				}
			}

			// If we aren't configured to drop constraints, we have to manually
			// drop the existing primary key if a new primary key is being
			// added because there can only be one primary key at a time.
			destPkey := destCache.GetPrimaryKey(destTable)
			srcPkey := srcCache.GetPrimaryKey(srcTable)
			if !dropObjects && srcPkey != nil && destPkey != nil && srcPkey.ConstraintName != destPkey.ConstraintName {
				alterTable.dropConstraints = append(alterTable.dropConstraints, srcPkey)
				droppedConstraint[srcPkey] = true
			}

			// If any of the columns we are altering have indexes or
			// constraints that depend on them, we have to drop and recreate
			// those indexes and constraints.
			columnIndexDependencies := make(map[string][]*Index)
			columnConstraintDependencies := make(map[string][]*Constraint)
			for k := range srcTable.Indexes {
				srcIndex := &srcTable.Indexes[k]
				for _, column := range srcIndex.Columns {
					columnIndexDependencies[column] = append(columnIndexDependencies[column], srcIndex)
				}
			}
			for k := range srcTable.Constraints {
				srcConstraint := &srcTable.Constraints[k]
				for _, column := range srcConstraint.Columns {
					columnConstraintDependencies[column] = append(columnConstraintDependencies[column], srcConstraint)
				}
			}
			for _, columnpair := range alterTable.alterColumns {
				columnName := columnpair[0].ColumnName
				for _, index := range columnIndexDependencies[columnName] {
					if droppedIndex[index] {
						continue
					}
					alterTable.dropIndexes = append(alterTable.dropIndexes, index)
					alterTable.createIndexes = append(alterTable.createIndexes, index)
				}
				for _, constraint := range columnConstraintDependencies[columnName] {
					if droppedConstraint[constraint] {
						continue
					}
					switch constraint.ConstraintType {
					case PRIMARY_KEY, UNIQUE:
						alterTable.dropConstraints = append(alterTable.dropConstraints, constraint)
						alterTable.addConstraints = append(alterTable.addConstraints, constraint)
					case FOREIGN_KEY:
						tablesID := getTablesID(constraint)
						n1, ok1 := dropFkeysPos[tablesID]
						n2, ok2 := addFkeysPos[tablesID]
						if ok1 {
							m.dropFkeys[n1] = append(m.dropFkeys[n1], constraint)
						} else {
							m.dropFkeys = append(m.dropFkeys, []*Constraint{constraint})
						}
						if ok2 {
							m.addFkeys[n2] = append(m.addFastFkeys[n2], constraint)
						} else {
							m.addFkeys = append(m.addFkeys, []*Constraint{constraint})
						}
					}
				}
			}

			if len(alterTable.dropConstraints) > 0 ||
				len(alterTable.dropIndexes) > 0 ||
				len(alterTable.addColumns) > 0 ||
				len(alterTable.alterColumns) > 0 ||
				len(alterTable.createIndexes) > 0 ||
				len(alterTable.addConstraints) > 0 {
				m.alterTables = append(m.alterTables, alterTable)
			}
		}
	}
	return m
}

func (m *sqlserverMigration) sql(prefix string) (filenames []string, bufs []*bytes.Buffer, warnings []string) {
	const dialect = DialectSQLServer
	warnings = m.warnings
	n := 0
	getTablesName := func(fkey *Constraint) string {
		var b strings.Builder
		b.Grow(len(fkey.TableSchema) + len(fkey.TableName) + len(fkey.ReferencesSchema) + len(fkey.ReferencesTable) + 4)
		if fkey.TableSchema != "" && fkey.TableSchema != m.currentSchema {
			b.WriteString(fkey.TableSchema + "_")
		}
		b.WriteString(fkey.TableName + "_")
		if fkey.ReferencesSchema != "" && fkey.ReferencesSchema != m.currentSchema {
			b.WriteString(fkey.ReferencesSchema + "_")
		}
		b.WriteString(fkey.ReferencesTable)
		return b.String()
	}

	// DROP FOREIGN KEY.
	for _, fkeys := range m.dropFkeys {
		n++
		name := getTablesName(fkeys[0])
		// ${prefix}_${n}_drop_${table1}_${table2}_fkeys.tx.sql
		filenames = append(filenames, fmt.Sprintf("%s_%02d_drop_%s_fkeys.tx.sql", prefix, n, name))
		buf := bufpool.Get().(*bytes.Buffer)
		buf.Reset()
		bufs = append(bufs, buf)
		for _, fkey := range fkeys {
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			constraintName := sq.QuoteIdentifier(dialect, fkey.ConstraintName)
			tableName := sq.QuoteIdentifier(dialect, fkey.TableName)
			if fkey.TableSchema != "" && fkey.TableSchema != m.currentSchema {
				tableName = sq.QuoteIdentifier(dialect, fkey.TableSchema) + "." + tableName
			}
			buf.WriteString("ALTER TABLE " + tableName + " DROP CONSTRAINT " + constraintName + ";\n")
		}
	}

	// DROP SCHEMA.
	for _, schema := range m.dropSchemas {
		n++
		// ${prefix}_${n}_drop_${schema}.sql
		filenames = append(filenames, fmt.Sprintf("%s_%02d_drop_%s.sql", prefix, n, schema.SchemaName))
		buf := bufpool.Get().(*bytes.Buffer)
		buf.Reset()
		bufs = append(bufs, buf)
		var schemaPrefix string
		if schema.SchemaName != "" && schema.SchemaName != m.currentSchema {
			schemaPrefix = sq.QuoteIdentifier(dialect, schema.SchemaName) + "."
		}
		// DROP VIEW.
		for _, view := range schema.Views {
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			buf.WriteString("DROP VIEW " + schemaPrefix + sq.QuoteIdentifier(dialect, view.ViewName) + ";\n")
		}
		// DROP TABLE.
		for _, table := range schema.Tables {
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			buf.WriteString("DROP TABLE " + schemaPrefix + sq.QuoteIdentifier(dialect, table.TableName) + ";\n")
		}
		// DROP PROCEDURE and DROP FUNCTION.
		for _, routine := range schema.Routines {
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			if routine.RoutineType == "FUNCTION" {
				buf.WriteString("DROP FUNCTION " + schemaPrefix + sq.QuoteIdentifier(dialect, routine.RoutineName) + ";\n")
			} else {
				buf.WriteString("DROP PROCEDURE " + schemaPrefix + sq.QuoteIdentifier(dialect, routine.RoutineName) + ";\n")
			}
		}
		// DROP SCHEMA.
		if buf.Len() > 0 {
			buf.WriteString("\n")
		}
		buf.WriteString("DROP SCHEMA " + sq.QuoteIdentifier(dialect, schema.SchemaName) + ";\n")
	}

	// CREATE SCHEMA.
	if len(m.createSchemas) > 0 {
		n++
		// ${prefix}_${n}_create_schemas.sql
		filenames = append(filenames, fmt.Sprintf("%s_%02d_create_schemas.sql", prefix, n))
		buf := bufpool.Get().(*bytes.Buffer)
		buf.Reset()
		bufs = append(bufs, buf)
		for _, schemaName := range m.createSchemas {
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			buf.WriteString("IF SCHEMA_ID('" + sq.EscapeQuote(schemaName, '\'') + "') IS NULL EXEC('CREATE SCHEMA " + sq.EscapeQuote(sq.QuoteIdentifier(dialect, schemaName), '\'') + "');\n")
		}
	}

	// DROP TABLE + CREATE TABLE.
	if len(m.dropTables) > 0 || len(m.createTables) > 0 {
		n++
		// ${prefix}_${n}_tables.sql
		filenames = append(filenames, fmt.Sprintf("%s_%02d_tables.sql", prefix, n))
		buf := bufpool.Get().(*bytes.Buffer)
		buf.Reset()
		bufs = append(bufs, buf)
		for _, table := range m.dropTables {
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			tableName := sq.QuoteIdentifier(dialect, table.TableName)
			if table.TableSchema != "" && table.TableSchema != m.currentSchema {
				tableName = sq.QuoteIdentifier(dialect, table.TableSchema) + "." + tableName
			}
			buf.WriteString("DROP TABLE " + tableName + ";\n")
		}
		for _, table := range m.createTables {
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			writeCreateTable(dialect, buf, m.currentSchema, m.defaultCollation, table, true)
			for _, index := range table.Indexes {
				if buf.Len() > 0 {
					buf.WriteString("\n")
				}
				writeCreateIndex(dialect, buf, m.currentSchema, &index, false)
			}
		}
	}

	// ALTER TABLE.
	for _, alterTable := range m.alterTables {
		isPkeyColumn := make(map[string]bool)
		if alterTable.pkey != nil {
			for _, column := range alterTable.pkey.Columns {
				isPkeyColumn[column] = true
			}
		}
		n++
		name := alterTable.tableName
		tableName := sq.QuoteIdentifier(dialect, alterTable.tableName)
		if alterTable.tableSchema != "" && alterTable.tableSchema != m.currentSchema {
			name = alterTable.tableSchema + "_" + name
			tableName = sq.QuoteIdentifier(dialect, alterTable.tableSchema) + "." + tableName
		}
		// ${prefix}_${n}_alter_${table}.tx.sql
		filenames = append(filenames, fmt.Sprintf("%s_%02d_alter_%s.tx.sql", prefix, n, name))
		buf := bufpool.Get().(*bytes.Buffer)
		buf.Reset()
		bufs = append(bufs, buf)

		// DROP INDEX.
		for _, index := range alterTable.dropIndexes {
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			indexName := sq.QuoteIdentifier(dialect, index.IndexName)
			tableName := sq.QuoteIdentifier(dialect, index.TableName)
			if index.TableSchema != "" && index.TableSchema != m.currentSchema {
				tableName = sq.QuoteIdentifier(dialect, index.TableSchema) + "." + tableName
			}
			buf.WriteString("DROP INDEX " + indexName + " ON " + tableName + ";\n")
		}

		// DROP CONSTRAINT.
		for _, constraint := range alterTable.dropConstraints {
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			constraintName := sq.QuoteIdentifier(dialect, constraint.ConstraintName)
			buf.WriteString("ALTER TABLE " + tableName + " DROP CONSTRAINT " + constraintName + ";\n")
		}

		// DROP COLUMN.
		for _, column := range alterTable.dropColumns {
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			columnName := sq.QuoteIdentifier(dialect, column.ColumnName)
			buf.WriteString("ALTER TABLE " + tableName + " DROP COLUMN " + columnName + ";\n")
		}

		// ADD COLUMN.
		for _, column := range alterTable.addColumns {
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			buf.WriteString("ALTER TABLE " + tableName + " ADD ")
			if dialect != DialectSQLServer {
				buf.WriteString("COLUMN ")
			}
			writeColumnDefinition(dialect, buf, m.defaultCollation, column, false)
			buf.WriteString(";\n")
		}

		// ALTER COLUMN.
		for _, columns := range alterTable.alterColumns {
			srcColumn, destColumn := columns[0], columns[1]
			columnName := sq.QuoteIdentifier(dialect, destColumn.ColumnName)
			srcType, srcArg1, srcArg2 := normalizeColumnType(dialect, srcColumn.ColumnType)
			destType, destArg1, destArg2 := normalizeColumnType(dialect, destColumn.ColumnType)
			// ALTER TYPE
			alterType := false
			if [3]string{srcType, srcArg1, srcArg2} != [3]string{destType, destArg1, destArg2} {
				alterType = true
			}
			// COLLATE
			collation := ""
			if destColumn.CollationName != "" && destColumn.CollationName != m.defaultCollation {
				collation = sq.EscapeQuote(destColumn.CollationName, '[')
			}
			// NULL | NOT NULL
			nullability := ""
			if isPkeyColumn[destColumn.ColumnName] || destColumn.ColumnIdentity != "" {
				nullability = "NOT NULL"
			} else if srcColumn.IsNotNull && !destColumn.IsNotNull {
				if destColumn.IsNotNull {
					nullability = "NOT NULL"
				} else {
					nullability = "NULL"
				}
			}
			if alterType || collation != "" || nullability != "" {
				if alterType {
					warnings = append(warnings, fmt.Sprintf("%s: column %s changing type from %q to %q may be unsafe", tableName, columnName, srcColumn.ColumnType, destColumn.ColumnType))
				}
				if buf.Len() > 0 {
					buf.WriteString("\n")
				}
				buf.WriteString("ALTER TABLE " + tableName + " ALTER COLUMN " + columnName + " " + destColumn.ColumnType)
				if collation != "" {
					buf.WriteString(" COLLATE " + collation)
				}
				if nullability != "" {
					buf.WriteString(" " + nullability)
				}
				buf.WriteString(";\n")
			}
			srcDefault := normalizeColumnDefault(dialect, srcColumn.ColumnDefault)
			destDefault := normalizeColumnDefault(dialect, destColumn.ColumnDefault)
			// Do we need to remove DEFAULT? Or do we need to add/change it?
			if srcDefault != destDefault {
				if srcDefault != "" {
					if buf.Len() > 0 {
						buf.WriteString("\n")
					}
					// https://stackoverflow.com/a/49393082/3030828
					buf.WriteString(`BEGIN
    DECLARE @name NVARCHAR(255);
    SELECT
        @name = default_constraints.name
    FROM
        sys.default_constraints
        JOIN sys.tables ON tables.object_id = default_constraints.parent_object_id
        JOIN sys.columns ON columns.column_id = default_constraints.parent_column_id
    WHERE
        SCHEMA_NAME(tables.schema_id) = '` + sq.EscapeQuote(alterTable.tableSchema, '\'') + `'
        AND tables.name = '` + sq.EscapeQuote(alterTable.tableName, '\'') + `'
        AND columns.name = '` + sq.EscapeQuote(destColumn.ColumnName, '\'') + `'
    ;
    EXEC('ALTER TABLE ` + sq.EscapeQuote(tableName, '\'') + ` DROP CONSTRAINT ' + @name);
END;
`)
				}
				if destDefault != "" {
					// ADD
					if buf.Len() > 0 {
						buf.WriteString("\n")
					}
					buf.WriteString("ALTER TABLE " + tableName + " ADD DEFAULT " + destDefault + " FOR " + columnName + ";\n")
				}
			}
		}

		// CREATE INDEX.
		for _, index := range alterTable.createIndexes {
			n++
			// ${prefix}_${n}_create_${index}.tx.sql
			filenames = append(filenames, fmt.Sprintf("%s_%02d_create_%s.tx.sql", prefix, n, index.IndexName))
			buf := bufpool.Get().(*bytes.Buffer)
			buf.Reset()
			bufs = append(bufs, buf)
			writeCreateIndex(dialect, buf, m.currentSchema, index, true)
		}

		// ADD CONSTRAINT.
		for _, constraint := range alterTable.addConstraints {
			n++
			// ${prefix}_${n}_add_${constraint}.tx.sql
			filenames = append(filenames, fmt.Sprintf("%s_%02d_add_%s.tx.sql", prefix, n, constraint.ConstraintName))
			buf := bufpool.Get().(*bytes.Buffer)
			buf.Reset()
			bufs = append(bufs, buf)
			buf.WriteString("ALTER TABLE " + tableName + " ADD ")
			writeConstraintDefinition(dialect, buf, m.currentSchema, constraint)
			buf.WriteString(";\n")
		}
	}

	// ADD FOREIGN KEY (fast).
	for _, fkeys := range m.addFastFkeys {
		n++
		name := getTablesName(fkeys[0])
		// ${prefix}_${n}_add_${table1}_${table2}_fkeys.tx.sql
		filenames = append(filenames, fmt.Sprintf("%s_%02d_add_%s_fkeys.tx.sql", prefix, n, name))
		buf := bufpool.Get().(*bytes.Buffer)
		buf.Reset()
		bufs = append(bufs, buf)
		for _, fkey := range fkeys {
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			buf.WriteString("ALTER TABLE ")
			if fkey.TableSchema != "" && fkey.TableSchema != m.currentSchema {
				buf.WriteString(sq.QuoteIdentifier(dialect, fkey.TableSchema) + ".")
			}
			buf.WriteString(sq.QuoteIdentifier(dialect, fkey.TableName) + " ADD ")
			writeConstraintDefinition(dialect, buf, m.currentSchema, fkey)
			buf.WriteString(";\n")
		}
	}

	// ADD FOREIGN KEY (existing tables).
	for _, fkeys := range m.addFkeys {
		n++
		name := getTablesName(fkeys[0])
		// ${prefix}_${n}_add_${table1}_${table2}_fkeys.tx.sql
		filenames = append(filenames, fmt.Sprintf("%s_%02d_add_%s_fkeys.tx.sql", prefix, n, name))
		buf := bufpool.Get().(*bytes.Buffer)
		buf.Reset()
		bufs = append(bufs, buf)
		for _, fkey := range fkeys {
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			buf.WriteString("ALTER TABLE ")
			if fkey.TableSchema != "" && fkey.TableSchema != m.currentSchema {
				buf.WriteString(sq.QuoteIdentifier(dialect, fkey.TableSchema) + ".")
			}
			buf.WriteString(sq.QuoteIdentifier(dialect, fkey.TableName) + " ADD ")
			writeConstraintDefinition(dialect, buf, m.currentSchema, fkey)
			buf.WriteString(";\n")
		}
	}

	return filenames, bufs, warnings
}
