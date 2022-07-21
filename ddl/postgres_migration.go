package ddl

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/bokwoon95/sq"
)

type postgresMigration struct {
	versionNums      VersionNums
	currentSchema    string
	defaultCollation string

	// 1. Drop the foreign keys.
	dropFkeys [][]*Constraint

	// 2. Execute all DROP SCHEMA + CREATE SCHEMA + DROP TABLE + CREATE TABLE in one transaction.
	dropSchemas   []string
	createSchemas []string
	dropTables    []*Table
	createTables  []*Table

	// 3. Execute each ALTER TABLE.
	alterTables []postgresAlterTable

	// 4. Add the foreign keys for new tables. This should be fast because the
	// new tables are empty (so no rows have to be validated).
	addFastFkeys [][]*Constraint

	// 5. Add the foreign keys for existing tables.
	addFkeys [][]*Constraint
}

type postgresAlterTable struct {
	tableSchema string
	tableName   string

	// Do these in one transaction.
	dropIndexes      []*Index
	dropConstraints  []*Constraint
	dropColumns      []*Column
	addColumns       []*Column
	alterColumns     [][2]*Column
	alterConstraints [][2]*Constraint

	// Validate NOT NULL check constraints in a separate transaction.
	validateNotNull []*Column

	// Create indexes concurrently outside a transaction.
	createIndexesConcurrently []*Index

	// Add constraints concurrently outside a transaction.
	addConstraintsConcurrently []*Constraint
}

func newPostgresMigration(srcCatalog, destCatalog *Catalog, dropObjects bool) postgresMigration {
	const dialect = sq.DialectPostgres
	m := postgresMigration{
		versionNums:      srcCatalog.VersionNums,
		currentSchema:    srcCatalog.CurrentSchema,
		defaultCollation: srcCatalog.DefaultCollation,
	}
	srcCache, destCache := NewCatalogCache(srcCatalog), NewCatalogCache(destCatalog)
	dropFkeysPos := make(map[[4]string]int)
	addFastFkeysPos := make(map[[4]string]int)
	addFkeysPos := make(map[[4]string]int)
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
					m.dropSchemas = append(m.dropSchemas, srcSchema.SchemaName)
				}
				for j := range srcSchema.Tables {
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
			alterTable := postgresAlterTable{
				tableSchema: destTable.TableSchema,
				tableName:   destTable.TableName,
			}
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
					}
				}
				for k := range srcTable.Indexes {
					srcIndex := &srcTable.Indexes[k]
					destIndex := destCache.GetIndex(destTable, srcIndex.IndexName)
					if destIndex == nil {
						// DROP INDEX.
						alterTable.dropIndexes = append(alterTable.dropIndexes, srcIndex)
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
				if srcColumn == nil {
					// ADD COLUMN.
					alterTable.addColumns = append(alterTable.addColumns, destColumn)
					continue
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
					if srcColumn.IsNotNull != destColumn.IsNotNull && !destColumn.IsPrimaryKey {
						return true
					}
					if (srcColumn.ColumnIdentity != "" && destColumn.ColumnIdentity == "") || (srcColumn.ColumnIdentity == "" && destColumn.ColumnIdentity != "") {
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
					if !srcColumn.IsNotNull && destColumn.IsNotNull && !m.versionNums.LowerThan(12) {
						alterTable.validateNotNull = append(alterTable.validateNotNull, destColumn)
					}
				}
			}
			for k := range destTable.Indexes {
				destIndex := &destTable.Indexes[k]
				srcIndex := srcCache.GetIndex(srcTable, destIndex.IndexName)
				if srcIndex == nil {
					// CREATE INDEX CONCURRENTLY.
					alterTable.createIndexesConcurrently = append(alterTable.createIndexesConcurrently, destIndex)
				}
			}
			addingPrimaryKey := false
			for k := range destTable.Constraints {
				destConstraint := &destTable.Constraints[k]
				srcConstraint := srcCache.GetConstraint(srcTable, destConstraint.ConstraintName)
				if srcConstraint == nil {
					switch destConstraint.ConstraintType {
					case PRIMARY_KEY:
						// ADD PRIMARY KEY CONCURRENTLY.
						addingPrimaryKey = true
						alterTable.addConstraintsConcurrently = append(alterTable.addConstraintsConcurrently, destConstraint)
					case UNIQUE:
						// ADD UNIQUE CONCURRENTLY.
						alterTable.addConstraintsConcurrently = append(alterTable.addConstraintsConcurrently, destConstraint)
					case FOREIGN_KEY:
						// ADD FOREIGN KEY + VALIDATE FOREIGN KEY.
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
				// ALTER CONSTRAINT.
				if srcConstraint.IsDeferrable != destConstraint.IsDeferrable || srcConstraint.IsInitiallyDeferred != destConstraint.IsInitiallyDeferred {
					alterTable.alterConstraints = append(alterTable.alterConstraints, [2]*Constraint{srcConstraint, destConstraint})
				}
			}
			// If we aren't configured to drop constraints, we have to manually
			// drop the existing primary key if a new primary key is being
			// added because there can only be one primary key at a time.
			if addingPrimaryKey && !dropObjects {
				srcPkey := srcCache.GetPrimaryKey(srcTable)
				if srcPkey != nil {
					alterTable.dropConstraints = append(alterTable.dropConstraints, srcPkey)
				}
			}
			if len(alterTable.dropConstraints) > 0 ||
				len(alterTable.dropIndexes) > 0 ||
				len(alterTable.addColumns) > 0 ||
				len(alterTable.alterColumns) > 0 ||
				len(alterTable.alterConstraints) > 0 ||
				len(alterTable.createIndexesConcurrently) > 0 ||
				len(alterTable.addConstraintsConcurrently) > 0 {
				m.alterTables = append(m.alterTables, alterTable)
			}
		}
	}
	return m
}

func (m *postgresMigration) sql(prefix string) (filenames []string, bufs []*bytes.Buffer, warnings []string) {
	const dialect = sq.DialectPostgres
	n := 0

	// DROP FOREIGN KEY.
	for _, fkeys := range m.dropFkeys {
		n++
		fk := *fkeys[0]
		name := strings.ReplaceAll(fk.TableName, " ", "_") + "_"
		tableName := sq.QuoteIdentifier(dialect, fk.TableName)
		if fk.TableSchema != "" && fk.TableSchema != m.currentSchema {
			name = strings.ReplaceAll(fk.TableSchema, " ", "_") + "_" + name
			tableName = sq.QuoteIdentifier(dialect, fk.TableSchema) + "." + tableName
		}
		if fk.ReferencesSchema != "" && fk.ReferencesSchema != m.currentSchema {
			name += strings.ReplaceAll(fk.ReferencesSchema, " ", "_") + "_"
		}
		name += strings.ReplaceAll(fk.ReferencesTable, " ", "_")
		// ${prefix}_${n}_drop_${table1}_${table2}_fkeys.tx.sql
		filenames = append(filenames, prefix+"_"+fmt.Sprintf("%02d", n)+"_drop_"+name+"_fkeys.tx.sql")
		buf := bufpool.Get().(*bytes.Buffer)
		buf.Reset()
		bufs = append(bufs, buf)
		for _, fkey := range fkeys {
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			constraintName := sq.QuoteIdentifier(dialect, fkey.ConstraintName)
			buf.WriteString("ALTER TABLE " + tableName + " DROP CONSTRAINT IF EXISTS " + constraintName + ";\n")
		}
	}

	// DROP SCHEMA + CREATE SCHEMA.
	if len(m.dropSchemas) > 0 || len(m.createSchemas) > 0 {
		n++
		// ${prefix}_${n}_schemas.sql
		filenames = append(filenames, prefix+"_"+fmt.Sprintf("%02d", n)+"_schemas.sql")
		buf := bufpool.Get().(*bytes.Buffer)
		buf.Reset()
		bufs = append(bufs, buf)
		for _, schemaName := range m.dropSchemas {
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			buf.WriteString("DROP SCHEMA IF EXISTS " + sq.QuoteIdentifier(dialect, schemaName) + " CASCADE;\n")
		}
		for _, schemaName := range m.createSchemas {
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			buf.WriteString("CREATE SCHEMA IF NOT EXISTS " + sq.QuoteIdentifier(dialect, schemaName) + ";\n")
		}
	}

	// DROP TABLE + CREATE TABLE.
	if len(m.dropTables) > 0 || len(m.createTables) > 0 {
		n++
		// ${prefix}_${n}_tables.sql
		filenames = append(filenames, prefix+"_"+fmt.Sprintf("%02d", n)+"_tables.sql")
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
			buf.WriteString("DROP TABLE IF EXISTS " + tableName + ";\n")
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
		n++
		name := strings.ReplaceAll(alterTable.tableName, " ", "_")
		tableName := sq.QuoteIdentifier(dialect, alterTable.tableName)
		if alterTable.tableSchema != "" && alterTable.tableSchema != m.currentSchema {
			name = strings.ReplaceAll(alterTable.tableSchema, " ", "_") + "_" + name
			tableName = sq.QuoteIdentifier(dialect, alterTable.tableSchema) + "." + tableName
		}
		// ${prefix}_${n}_alter_${table}.tx.sql
		filenames = append(filenames, prefix+"_"+fmt.Sprintf("%02d", n)+"_alter_"+name+".tx.sql")
		buf := bufpool.Get().(*bytes.Buffer)
		buf.Reset()
		bufs = append(bufs, buf)
		// DROP INDEX.
		for _, index := range alterTable.dropIndexes {
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			indexName := sq.QuoteIdentifier(dialect, index.IndexName)
			if index.TableSchema != "" && index.TableSchema != m.currentSchema {
				indexName = sq.QuoteIdentifier(dialect, index.TableSchema) + "." + indexName
			}
			buf.WriteString("DROP INDEX IF EXISTS " + indexName + ";\n")
		}
		// DROP CONSTRAINT.
		for _, constraint := range alterTable.dropConstraints {
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			constraintName := sq.QuoteIdentifier(dialect, constraint.ConstraintName)
			buf.WriteString("ALTER TABLE " + tableName + " DROP CONSTRAINT IF EXISTS " + constraintName + ";\n")
		}
		// DROP COLUMN.
		for _, column := range alterTable.dropColumns {
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			columnName := sq.QuoteIdentifier(dialect, column.ColumnName)
			buf.WriteString("ALTER TABLE " + tableName + " DROP COLUMN IF EXISTS " + columnName + ";\n")
		}
		// ADD COLUMN.
		for _, column := range alterTable.addColumns {
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			columnName := sq.QuoteIdentifier(dialect, column.ColumnName)
			if column.ColumnDefault != "" && m.versionNums.LowerThan(11) {
				warnings = append(warnings, fmt.Sprintf("%s: adding column %q with DEFAULT is unsafe for large tables. Upgrade to Postgres 11+ to avoid this issue. If not, you should add a column without the default first, backfill the default values, then set the column default", tableName, columnName))
			} else if column.ColumnIdentity != "" && m.versionNums.LowerThan(11) {
				warnings = append(warnings, fmt.Sprintf("%s: adding column %q with %q is unsafe for large tables. Upgrade to Postgres 11+ to avoid this issue. If not, you should add a column without the identity first, backfill the identity values, then add the identity to the column and finally update the identity sequence", tableName, column.ColumnIdentity, columnName))
			}
			buf.WriteString("ALTER TABLE " + tableName + " ADD COLUMN ")
			writeColumnDefinition(dialect, buf, m.defaultCollation, column, false)
			buf.WriteString(";\n")
		}
		// ALTER COLUMN.
		for _, columns := range alterTable.alterColumns {
			srcColumn, destColumn := columns[0], columns[1]
			columnName := destColumn.ColumnName
			srcType, srcArg1, srcArg2 := normalizeColumnType(dialect, srcColumn.ColumnType)
			destType, destArg1, destArg2 := normalizeColumnType(dialect, destColumn.ColumnType)
			// Do we need to ALTER TYPE?
			if [3]string{srcType, srcArg1, srcArg2} != [3]string{destType, destArg1, destArg2} {
				// I'm going to ignore CITEXT because using it doesn't seem to
				// be a good idea. "Personally, I stay away from citext after
				// mixed experiences."
				// (https://dba.stackexchange.com/a/230688). Besides,
				// accounting for CITEXT would complicate code here because
				// ALTER TYPE safety depends on whether the CITEXT column is
				// indexed. I'll just treat all CITEXT conversions as unsafe
				// and let the user decide if it is safe. Reference:
				// https://github.com/ankane/strong_migrations#changing-the-type-of-a-column.
				switch [2]string{srcType, destType} {
				case [2]string{"CIDR", "INET"}, [2]string{"VARCHAR", "TEXT"}:
					// Always ok.
				case [2]string{"TEXT", "VARCHAR"}:
					if destArg1 != "" {
						warnings = append(warnings, fmt.Sprintf("%s: column %q changing type from \"TEXT\" to %q is unsafe", tableName, columnName, destColumn.ColumnType))
					}
				case [2]string{"VARCHAR", "VARCHAR"}:
					srcLimit, _ := strconv.Atoi(srcArg1)
					destLimit, _ := strconv.Atoi(destArg1)
					if srcLimit > 0 && destLimit > 0 && destLimit < srcLimit {
						warnings = append(warnings, fmt.Sprintf("%s: column %q decreasing limit from %q to %q is unsafe", tableName, columnName, srcColumn.ColumnType, destColumn.ColumnType))
					}
				case [2]string{"NUMERIC", "NUMERIC"}:
					srcScale, _ := strconv.Atoi(srcArg2)
					destScale, _ := strconv.Atoi(destArg2)
					if srcScale != destScale && destScale != 0 {
						warnings = append(warnings, fmt.Sprintf("%s: column %q changing scale from %q to %q is unsafe", tableName, columnName, srcColumn.ColumnType, destColumn.ColumnType))
					}
					srcPrecision, _ := strconv.Atoi(srcArg1)
					destPrecision, _ := strconv.Atoi(destArg1)
					if srcPrecision > 0 && destPrecision > 0 && destPrecision < srcPrecision {
						warnings = append(warnings, fmt.Sprintf("%s: column %q decreasing precision from %q to %q is unsafe", tableName, columnName, srcColumn.ColumnType, destColumn.ColumnType))
					}
				default:
					warnings = append(warnings, fmt.Sprintf("%s: column %q changing type from %q to %q may be unsafe", tableName, columnName, srcColumn.ColumnType, destColumn.ColumnType))
				}
				if buf.Len() > 0 {
					buf.WriteString("\n")
				}
				buf.WriteString("ALTER TABLE " + tableName + " ALTER COLUMN " + columnName + " TYPE " + destColumn.ColumnType)
				if destColumn.CollationName != "" && destColumn.CollationName != m.defaultCollation {
					buf.WriteString(` COLLATE "` + sq.EscapeQuote(destColumn.CollationName, '"') + `"`)
				}
				buf.WriteString(";\n")
			} else {
				// Do we need to add ALTER TYPE ... COLLATE?
				srcCollation := srcColumn.CollationName
				if srcCollation == "" {
					srcCollation = m.defaultCollation
				}
				destCollation := destColumn.CollationName
				if destCollation == "" {
					destCollation = m.defaultCollation
				}
				if srcCollation != destCollation {
					if buf.Len() > 0 {
						buf.WriteString("\n")
					}
					buf.WriteString("ALTER TABLE " + tableName + " ALTER COLUMN " + columnName + " TYPE " + destColumn.ColumnType + ` COLLATE "` + sq.EscapeQuote(destColumn.CollationName, '"') + `"` + ";\n")
				}
			}
			srcDefault := normalizeColumnDefault(dialect, srcColumn.ColumnDefault)
			destDefault := normalizeColumnDefault(dialect, destColumn.ColumnDefault)
			// Do we need to remove DEFAULT? Or do we need to add/change it?
			if srcDefault != "" && destDefault == "" {
				if buf.Len() > 0 {
					buf.WriteString("\n")
				}
				buf.WriteString("ALTER TABLE " + tableName + " ALTER COLUMN " + columnName + " DROP DEFAULT;\n")
			} else if (srcDefault == "" && destDefault != "") || (isLiteral(srcDefault) && isLiteral(destDefault) && srcDefault != destDefault) {
				if buf.Len() > 0 {
					buf.WriteString("\n")
				}
				buf.WriteString("ALTER TABLE " + tableName + " ALTER COLUMN " + columnName + " SET DEFAULT " + destDefault + ";\n")
			}
			// Do we need to remove NOT NULL? Or do we need to add it?
			if srcColumn.IsNotNull && !destColumn.IsNotNull {
				if buf.Len() > 0 {
					buf.WriteString("\n")
				}
				buf.WriteString("ALTER TABLE " + tableName + " ALTER COLUMN " + columnName + " DROP NOT NULL;\n")
			} else if !srcColumn.IsNotNull && destColumn.IsNotNull {
				if buf.Len() > 0 {
					buf.WriteString("\n")
				}
				if m.versionNums.LowerThan(12) {
					warnings = append(warnings, fmt.Sprintf("%s: setting NOT NULL on column %q is unsafe for large tables. Upgrade to Postgres 12+ to avoid this issue", tableName, columnName))
					buf.WriteString("ALTER TABLE " + tableName + " ALTER COLUMN " + columnName + " SET NOT NULL;\n")
				} else {
					constraintName := sq.QuoteIdentifier(dialect, destColumn.TableName+"_"+destColumn.ColumnName+"_not_null_check")
					buf.WriteString("ALTER TABLE " + tableName + " ADD CONSTRAINT " + constraintName + " CHECK (" + columnName + " IS NOT NULL) NOT VALID;\n")
				}
			}
			// Do we need to remove IDENTITY? Or do we need to add it?
			if srcColumn.ColumnIdentity != "" && destColumn.ColumnIdentity == "" {
				if buf.Len() > 0 {
					buf.WriteString("\n")
				}
				buf.WriteString("ALTER TABLE " + tableName + " ALTER COLUMN " + columnName + " DROP IDENTITY;\n")
			} else if srcColumn.ColumnIdentity == "" && destColumn.ColumnIdentity != "" {
				if buf.Len() > 0 {
					buf.WriteString("\n")
				}
				buf.WriteString("ALTER TABLE " + tableName + " ALTER COLUMN " + columnName + " ADD " + destColumn.ColumnIdentity + ";\n")
			}
		}
		// ALTER CONSTRAINT.
		for _, constraint := range alterTable.alterConstraints {
			destConstraint := constraint[1]
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			constraintName := sq.QuoteIdentifier(dialect, destConstraint.ConstraintName)
			buf.WriteString("ALTER TABLE " + tableName + " ALTER CONSTRAINT " + constraintName)
			if destConstraint.IsDeferrable {
				buf.WriteString(" DEFERRABLE")
			} else {
				buf.WriteString(" NOT DEFERRABLE")
			}
			if destConstraint.IsInitiallyDeferred {
				buf.WriteString(" INITIALLY DEFERRED")
			}
			buf.WriteString(";\n")
		}

		// VALIDATE NOT NULL CHECK.
		if len(alterTable.validateNotNull) > 0 {
			n++
			// ${prefix}_${n}_validate_${table}_not_null.tx.sql
			filenames = append(filenames, prefix+"_"+fmt.Sprintf("%02d", n)+"_validate_"+name+"_not_null.tx.sql")
			buf := bufpool.Get().(*bytes.Buffer)
			buf.Reset()
			bufs = append(bufs, buf)
			for _, column := range alterTable.validateNotNull {
				if buf.Len() > 0 {
					buf.WriteString("\n")
				}
				constraintName := sq.QuoteIdentifier(dialect, column.TableName+"_"+column.ColumnName+"_not_null_check")
				columnName := sq.QuoteIdentifier(dialect, column.ColumnName)
				buf.WriteString("ALTER TABLE " + tableName + " VALIDATE CONSTRAINT " + constraintName + ";\n")
				buf.WriteString("ALTER TABLE " + tableName + " ALTER COLUMN " + columnName + " SET NOT NULL;\n")
				buf.WriteString("ALTER TABLE " + tableName + " DROP CONSTRAINT " + constraintName + ";\n")
			}
		}

		// CREATE INDEX CONCURRENTLY.
		for _, index := range alterTable.createIndexesConcurrently {
			n++
			name := strings.ReplaceAll(index.IndexName, " ", "_")
			num := fmt.Sprintf("%02d", n)
			// ${prefix}_${n}_create_${index}.txoff.sql
			filenames = append(filenames, prefix+"_"+num+"_create_"+name+".txoff.sql")
			buf := bufpool.Get().(*bytes.Buffer)
			buf.Reset()
			bufs = append(bufs, buf)
			writeCreateIndex(dialect, buf, m.currentSchema, index, true)
			// ${prefix}_${n}_create_${index}.undo.sql
			filenames = append(filenames, prefix+"_"+num+"_create_"+name+".undo.sql")
			buf = bufpool.Get().(*bytes.Buffer)
			buf.Reset()
			bufs = append(bufs, buf)
			indexName := sq.QuoteIdentifier(dialect, index.IndexName)
			if index.TableSchema != "" && index.TableSchema != m.currentSchema {
				indexName = sq.QuoteIdentifier(dialect, index.TableSchema) + "." + indexName
			}
			buf.WriteString("DROP INDEX IF EXISTS " + indexName + ";\n")
		}

		// ADD CONSTRAINT CONCURRENTLY.
		for _, addKeyConstraint := range alterTable.addConstraintsConcurrently {
			n++
			name := strings.ReplaceAll(addKeyConstraint.ConstraintName, " ", "_")
			constraintName := sq.QuoteIdentifier(dialect, addKeyConstraint.ConstraintName)
			// ${prefix}_${n}_create_${index}.txoff.sql
			filenames = append(filenames, fmt.Sprintf("%s_%02d_create_%s.txoff.sql", prefix, n, name))
			buf := bufpool.Get().(*bytes.Buffer)
			buf.Reset()
			bufs = append(bufs, buf)
			createIndex := &Index{
				TableSchema: addKeyConstraint.TableSchema,
				TableName:   addKeyConstraint.TableName,
				IndexName:   addKeyConstraint.ConstraintName,
				IsUnique:    true,
				Columns:     addKeyConstraint.Columns,
			}
			writeCreateIndex(dialect, buf, m.currentSchema, createIndex, true)
			// ${prefix}_${n}_create_${index}.undo.sql
			filenames = append(filenames, fmt.Sprintf("%s_%02d_create_%s.undo.sql", prefix, n, name))
			buf = bufpool.Get().(*bytes.Buffer)
			buf.Reset()
			bufs = append(bufs, buf)
			indexName := sq.QuoteIdentifier(dialect, createIndex.IndexName)
			if createIndex.TableSchema != "" && createIndex.TableSchema != m.currentSchema {
				indexName = sq.QuoteIdentifier(dialect, createIndex.TableSchema) + "." + indexName
			}
			buf.WriteString("DROP INDEX IF EXISTS " + indexName + ";\n")
			// ${prefix}_${n}_add_${constraint}.tx.sql
			n++
			filenames = append(filenames, fmt.Sprintf("%s_%02d_add_%s.tx.sql", prefix, n, name))
			buf = bufpool.Get().(*bytes.Buffer)
			buf.Reset()
			bufs = append(bufs, buf)
			buf.WriteString("ALTER TABLE " + tableName + " ADD CONSTRAINT " + constraintName + " " + addKeyConstraint.ConstraintType + " USING INDEX " + constraintName + ";\n")
		}
	}

	// ADD FOREIGN KEY.
	for _, fkeys := range m.addFastFkeys {
		n++
		fk := *fkeys[0]
		name := strings.ReplaceAll(fk.TableName, " ", "_") + "_"
		tableName := sq.QuoteIdentifier(dialect, fk.TableName)
		if fk.TableSchema != "" && fk.TableSchema != m.currentSchema {
			name = strings.ReplaceAll(fk.TableSchema, " ", "_") + "_" + name
			tableName = sq.QuoteIdentifier(dialect, fk.TableSchema) + "." + tableName
		}
		if fk.ReferencesSchema != "" && fk.ReferencesSchema != m.currentSchema {
			name += strings.ReplaceAll(fk.ReferencesSchema, " ", "_") + "_"
		}
		name += strings.ReplaceAll(fk.ReferencesTable, " ", "_")
		// ${prefix}_${n}_add_${table1}_${table2}_fkeys.tx.sql
		filenames = append(filenames, prefix+"_"+fmt.Sprintf("%02d", n)+"_add_"+name+"_fkeys.tx.sql")
		buf := bufpool.Get().(*bytes.Buffer)
		buf.Reset()
		bufs = append(bufs, buf)
		for _, fkey := range fkeys {
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			buf.WriteString("ALTER TABLE " + tableName + " ADD ")
			writeConstraintDefinition(dialect, buf, m.currentSchema, fkey)
			buf.WriteString(";\n")
		}
	}

	// ADD FOREIGN KEY NOT VALID + VALIDATE FOREIGN KEY.
	for _, fkeys := range m.addFkeys {
		n++
		fk := *fkeys[0]
		name := strings.ReplaceAll(fk.TableName, " ", "_") + "_"
		tableName := sq.QuoteIdentifier(dialect, fk.TableName)
		if fk.TableSchema != "" && fk.TableSchema != m.currentSchema {
			name = strings.ReplaceAll(fk.TableSchema, " ", "_") + "_" + name
			tableName = sq.QuoteIdentifier(dialect, fk.TableSchema) + "." + tableName
		}
		if fk.ReferencesSchema != "" && fk.ReferencesSchema != m.currentSchema {
			name += strings.ReplaceAll(fk.ReferencesSchema, " ", "_") + "_"
		}
		name += strings.ReplaceAll(fk.ReferencesTable, " ", "_")
		// ${prefix}_${n}_add_${table1}_${table2}_fkeys.tx.sql
		filenames = append(filenames, prefix+"_"+fmt.Sprintf("%02d", n)+"_add_"+name+"_fkeys.tx.sql")
		buf := bufpool.Get().(*bytes.Buffer)
		buf.Reset()
		bufs = append(bufs, buf)
		for _, fkey := range fkeys {
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			buf.WriteString("ALTER TABLE " + tableName + " ADD ")
			writeConstraintDefinition(dialect, buf, m.currentSchema, fkey)
			buf.WriteString(" NOT VALID;\n")
		}
		n++
		// ${prefix}_${n}_validate_${table1}_${table2}_fkeys.tx.sql
		filenames = append(filenames, prefix+"_"+fmt.Sprintf("%02d", n)+"_validate_"+name+"_fkeys.tx.sql")
		buf = bufpool.Get().(*bytes.Buffer)
		buf.Reset()
		bufs = append(bufs, buf)
		for _, fkey := range fkeys {
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			constraintName := sq.QuoteIdentifier(dialect, fkey.ConstraintName)
			buf.WriteString("ALTER TABLE " + tableName + " VALIDATE CONSTRAINT " + constraintName + ";\n")
		}
	}

	return filenames, bufs, warnings
}
