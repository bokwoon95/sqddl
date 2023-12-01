package ddl

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

type mysqlMigration struct {
	versionNums      VersionNums
	currentSchema    string
	defaultCollation string
	dropFkeys        []*Constraint
	dropSchemas      []string
	createSchemas    []string
	dropTables       []*Table
	createTables     []*Table
	alterTables      []mysqlAlterTable
	addFkeys         []*Constraint
}

type mysqlAlterTable struct {
	tableSchema     string
	tableName       string
	dropConstraints []*Constraint
	dropIndexes     []*Index
	dropColumns     []*Column
	addColumns      []*Column
	alterColumns    [][2]*Column
	createIndexes   []*Index
	addConstraints  []*Constraint
}

func newMySQLMigration(srcCatalog, destCatalog *Catalog, dropObjects bool) mysqlMigration {
	const dialect = DialectMySQL
	m := mysqlMigration{
		versionNums:      srcCatalog.VersionNums,
		currentSchema:    srcCatalog.CurrentSchema,
		defaultCollation: srcCatalog.DefaultCollation,
	}
	srcCache, destCache := NewCatalogCache(srcCatalog), NewCatalogCache(destCatalog)
	if dropObjects {
		for i := range srcCatalog.Schemas {
			srcSchema := &srcCatalog.Schemas[i]
			if srcSchema.Ignore {
				continue
			}
			destSchema := destCache.GetSchema(destCatalog, srcSchema.SchemaName)
			if destSchema == nil {
				// DROP SCHEMA.
				if srcSchema.SchemaName != "" {
					m.dropSchemas = append(m.dropSchemas, srcSchema.SchemaName)
				}
				for j := range srcSchema.Tables {
					srcTable := &srcSchema.Tables[j]
					if srcTable.Ignore {
						continue
					}
					// DROP FOREIGN KEY.
					srcFkeys := srcCache.GetForeignKeys(srcTable)
					m.dropFkeys = append(m.dropFkeys, srcFkeys...)
				}
				continue
			}
		}
	}
	for i := range destCatalog.Schemas {
		destSchema := &destCatalog.Schemas[i]
		if destSchema.Ignore {
			continue
		}
		srcSchema := srcCache.GetSchema(srcCatalog, destSchema.SchemaName)
		if srcSchema == nil {
			// CREATE SCHEMA.
			if destSchema.SchemaName != "" {
				m.createSchemas = append(m.createSchemas, destSchema.SchemaName)
			}
			for j := range destSchema.Tables {
				destTable := &destSchema.Tables[j]
				if destTable.Ignore {
					continue
				}
				// CREATE TABLE.
				m.createTables = append(m.createTables, destTable)
				// ADD FOREIGN KEY.
				destFkeys := destCache.GetForeignKeys(destTable)
				m.addFkeys = append(m.addFkeys, destFkeys...)
			}
			continue
		}
		if dropObjects {
			for j := range srcSchema.Tables {
				srcTable := &srcSchema.Tables[j]
				if srcTable.Ignore {
					continue
				}
				destTable := destCache.GetTable(destSchema, srcTable.TableName)
				if destTable == nil {
					// DROP TABLE.
					m.dropTables = append(m.dropTables, srcTable)
					// DROP FOREIGN KEY.
					srcFkeys := srcCache.GetForeignKeys(srcTable)
					m.dropFkeys = append(m.dropFkeys, srcFkeys...)
				}
			}
		}
		for j := range destSchema.Tables {
			destTable := &destSchema.Tables[j]
			if destTable.Ignore {
				continue
			}
			srcTable := srcCache.GetTable(srcSchema, destTable.TableName)
			if srcTable == nil {
				// CREATE TABLE.
				m.createTables = append(m.createTables, destTable)
				// ADD FOREIGN KEY.
				destFkeys := destCache.GetForeignKeys(destTable)
				m.addFkeys = append(m.addFkeys, destFkeys...)
				continue
			}
			// ALTER TABLE.
			alterTable := mysqlAlterTable{
				tableSchema: destTable.TableSchema,
				tableName:   destTable.TableName,
			}
			if dropObjects {
				for k := range srcTable.Constraints {
					srcConstraint := &srcTable.Constraints[k]
					if srcConstraint.Ignore {
						continue
					}
					destConstraint := destCache.GetConstraint(destTable, srcConstraint.ConstraintName)
					if destConstraint == nil {
						switch srcConstraint.ConstraintType {
						case PRIMARY_KEY, UNIQUE:
							// DROP PRIMARY KEY, DROP UNIQUE.
							alterTable.dropConstraints = append(alterTable.dropConstraints, srcConstraint)
						case FOREIGN_KEY:
							// DROP FOREIGN KEY.
							m.dropFkeys = append(m.dropFkeys, srcConstraint)
						}
						continue
					}
				}
				for k := range srcTable.Indexes {
					srcIndex := &srcTable.Indexes[k]
					if srcIndex.Ignore {
						continue
					}
					destIndex := destCache.GetIndex(destTable, srcIndex.IndexName)
					if destIndex == nil {
						// DROP INDEX.
						alterTable.dropIndexes = append(alterTable.dropIndexes, srcIndex)
					}
				}
				for k := range srcTable.Columns {
					srcColumn := &srcTable.Columns[k]
					if srcColumn.Ignore {
						continue
					}
					destColumn := destCache.GetColumn(destTable, srcColumn.ColumnName)
					if destColumn == nil {
						// DROP COLUMN.
						alterTable.dropColumns = append(alterTable.dropColumns, srcColumn)
					}
				}
			}
			for k := range destTable.Columns {
				destColumn := &destTable.Columns[k]
				if destColumn.Ignore {
					continue
				}
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
				}
			}
			for k := range destTable.Indexes {
				destIndex := &destTable.Indexes[k]
				if destIndex.Ignore {
					continue
				}
				srcIndex := srcCache.GetIndex(srcTable, destIndex.IndexName)
				if srcIndex == nil {
					// CREATE INDEX.
					alterTable.createIndexes = append(alterTable.createIndexes, destIndex)
				}
			}
			for k := range destTable.Constraints {
				destConstraint := &destTable.Constraints[k]
				if destConstraint.Ignore {
					continue
				}
				srcConstraint := srcCache.GetConstraint(srcTable, destConstraint.ConstraintName)
				if srcConstraint == nil {
					switch destConstraint.ConstraintType {
					case PRIMARY_KEY, UNIQUE:
						// ADD PRIMARY KEY | ADD UNIQUE.
						alterTable.addConstraints = append(alterTable.addConstraints, destConstraint)
					case FOREIGN_KEY:
						// ADD FOREIGN KEY.
						m.addFkeys = append(m.addFkeys, destConstraint)
					}
					continue
				}
				// MySQL primary keys are always called `PRIMARY` so we cannot
				// rely on the constraint name as their identity. Instead we
				// have to manually check if their columns are the same. If the
				// columns are not the same, we need to drop the old primary
				// key and add the new primary key.
				if destConstraint.ConstraintType == PRIMARY_KEY {
					srcColumns := strings.Join(srcConstraint.Columns, "\x00")
					destColumns := strings.Join(destConstraint.Columns, "\x00")
					if srcColumns != destColumns {
						alterTable.dropConstraints = append(alterTable.dropConstraints, srcConstraint)
						alterTable.addConstraints = append(alterTable.addConstraints, destConstraint)
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

func (m *mysqlMigration) sql(prefix string) (filenames []string, bufs []*bytes.Buffer, warnings []string) {
	const dialect = DialectMySQL
	n := 0

	// DROP FOREIGN KEY.
	for _, fkey := range m.dropFkeys {
		n++
		name := strings.ReplaceAll(fkey.ConstraintName, " ", "_")
		// ${prefix}_${n}_drop_${constraint}.sql
		filenames = append(filenames, prefix+"_"+fmt.Sprintf("%02d", n)+"_drop_"+name+".sql")
		buf := bufpool.Get().(*bytes.Buffer)
		buf.Reset()
		bufs = append(bufs, buf)
		tableName := QuoteIdentifier(dialect, fkey.TableName)
		if fkey.TableSchema != "" && fkey.TableSchema != m.currentSchema {
			tableName = QuoteIdentifier(dialect, fkey.TableSchema) + "." + tableName
		}
		constraintName := QuoteIdentifier(dialect, fkey.ConstraintName)
		buf.WriteString("ALTER TABLE " + tableName + " DROP CONSTRAINT " + constraintName + ";\n")
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
			buf.WriteString("DROP SCHEMA IF EXISTS " + QuoteIdentifier(dialect, schemaName) + ";\n")
		}
		if len(m.createSchemas) > 0 {
			// ${prefix}_${n}_schemas.undo.sql
			filenames = append(filenames, prefix+"_"+fmt.Sprintf("%02d", n)+"_schemas.undo.sql")
			undobuf := bufpool.Get().(*bytes.Buffer)
			undobuf.Reset()
			bufs = append(bufs, undobuf)
			for _, schemaName := range m.createSchemas {
				if buf.Len() > 0 {
					buf.WriteString("\n")
				}
				buf.WriteString("CREATE SCHEMA IF NOT EXISTS " + QuoteIdentifier(dialect, schemaName) + ";\n")
				if undobuf.Len() > 0 {
					undobuf.WriteString("\n")
				}
				undobuf.WriteString("DROP SCHEMA IF EXISTS " + QuoteIdentifier(dialect, schemaName) + ";\n")
			}
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
			tableName := QuoteIdentifier(dialect, table.TableName)
			if table.TableSchema != "" && table.TableSchema != m.currentSchema {
				tableName = QuoteIdentifier(dialect, table.TableSchema) + "." + tableName
			}
			buf.WriteString("DROP TABLE IF EXISTS " + tableName + ";\n")
		}
		if len(m.createTables) > 0 {
			// ${prefix}_${n}_tables.undo.sql
			filenames = append(filenames, prefix+"_"+fmt.Sprintf("%02d", n)+"_tables.undo.sql")
			undobuf := bufpool.Get().(*bytes.Buffer)
			undobuf.Reset()
			bufs = append(bufs, undobuf)
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
				if undobuf.Len() > 0 {
					undobuf.WriteString("\n")
				}
				tableName := QuoteIdentifier(dialect, table.TableName)
				if table.TableSchema != "" && table.TableSchema != m.currentSchema {
					tableName = QuoteIdentifier(dialect, table.TableSchema) + "." + tableName
				}
				undobuf.WriteString("DROP TABLE IF EXISTS " + tableName + ";\n")
			}
		}
	}

	// ALTER TABLE.
	for _, alterTable := range m.alterTables {
		n++
		name := strings.ReplaceAll(alterTable.tableName, " ", "_")
		tableName := QuoteIdentifier(dialect, alterTable.tableName)
		if alterTable.tableSchema != "" && alterTable.tableSchema != m.currentSchema {
			name = strings.ReplaceAll(alterTable.tableSchema, " ", "_") + "_" + name
			tableName = QuoteIdentifier(dialect, alterTable.tableSchema) + "." + tableName
		}
		// ${prefix}_${n}_alter_${table}.sql
		filenames = append(filenames, prefix+"_"+fmt.Sprintf("%02d", n)+"_alter_"+name+".sql")
		buf := bufpool.Get().(*bytes.Buffer)
		buf.Reset()
		bufs = append(bufs, buf)
		buf.WriteString("ALTER TABLE " + tableName)
		written := false
		for _, constraint := range alterTable.dropConstraints {
			buf.WriteString("\n    ")
			if written {
				buf.WriteString(",")
			}
			written = true
			constraintName := QuoteIdentifier(dialect, constraint.ConstraintName)
			buf.WriteString("DROP CONSTRAINT " + constraintName)
		}
		for _, index := range alterTable.dropIndexes {
			buf.WriteString("\n    ")
			if written {
				buf.WriteString(",")
			}
			written = true
			indexName := QuoteIdentifier(dialect, index.IndexName)
			buf.WriteString("DROP INDEX " + indexName)
		}
		for _, column := range alterTable.dropColumns {
			buf.WriteString("\n    ")
			if written {
				buf.WriteString(",")
			}
			written = true
			columnName := QuoteIdentifier(dialect, column.ColumnName)
			buf.WriteString("DROP COLUMN " + columnName)
		}
		for _, column := range alterTable.addColumns {
			buf.WriteString("\n    ")
			if written {
				buf.WriteString(",")
			}
			written = true
			buf.WriteString("ADD COLUMN ")
			writeColumnDefinition(dialect, buf, m.defaultCollation, column, false)
		}
		for _, columns := range alterTable.alterColumns {
			srcColumn, destColumn := columns[0], columns[1]
			columnName := destColumn.ColumnName
			srcType, srcArg1, srcArg2 := normalizeColumnType(dialect, srcColumn.ColumnType)
			destType, destArg1, destArg2 := normalizeColumnType(dialect, destColumn.ColumnType)
			if [3]string{srcType, srcArg1, srcArg2} != [3]string{destType, destArg1, destArg2} {
				switch [2]string{srcType, destType} {
				case [2]string{"VARCHAR", "VARCHAR"}:
					srcLimit, _ := strconv.Atoi(srcArg1)
					destLimit, _ := strconv.Atoi(destArg1)
					if srcLimit > 0 && destLimit > 0 {
						if srcLimit <= 255 && destLimit > 255 {
							warnings = append(warnings, fmt.Sprintf("%s: column %q changing type from %q to %q is unsafe (cannot increase limit from less than or equal to 255 to greater than 255)", tableName, columnName, srcColumn.ColumnType, destColumn.ColumnType))
						} else if srcLimit > 255 && destLimit <= 255 {
							warnings = append(warnings, fmt.Sprintf("%s: column %q changing type from %q to %q is unsafe (cannot decrease limit from greater than 255 to less than or equal to 255)", tableName, columnName, srcColumn.ColumnType, destColumn.ColumnType))
						}
					}
				default:
					warnings = append(warnings, fmt.Sprintf("%s: column %q changing type from %q to %q may be unsafe", tableName, columnName, srcColumn.ColumnType, destColumn.ColumnType))
				}
			}
			buf.WriteString("\n    ")
			if written {
				buf.WriteString(",")
			}
			written = true
			buf.WriteString("MODIFY COLUMN ")
			writeColumnDefinition(dialect, buf, m.defaultCollation, destColumn, false)
		}
		for _, index := range alterTable.createIndexes {
			buf.WriteString("\n    ")
			if written {
				buf.WriteString(",")
			}
			written = true
			buf.WriteString("ADD ")
			writeIndexDefinition(dialect, buf, m.currentSchema, index, false, true)
		}
		for _, constraint := range alterTable.addConstraints {
			buf.WriteString("\n    ")
			if written {
				buf.WriteString(",")
			}
			written = true
			buf.WriteString("ADD ")
			writeConstraintDefinition(dialect, buf, m.currentSchema, constraint)
		}
		buf.WriteString("\n;\n")
	}

	// ADD FOREIGN KEY.
	for _, fkey := range m.addFkeys {
		n++
		name := strings.ReplaceAll(fkey.ConstraintName, " ", "_")
		// ${prefix}_${n}_add_${constraint}.sql
		filenames = append(filenames, prefix+"_"+fmt.Sprintf("%02d", n)+"_add_"+name+".sql")
		buf := bufpool.Get().(*bytes.Buffer)
		buf.Reset()
		bufs = append(bufs, buf)
		tableName := QuoteIdentifier(dialect, fkey.TableName)
		if fkey.TableSchema != "" && fkey.TableSchema != m.currentSchema {
			tableName = QuoteIdentifier(dialect, fkey.TableSchema) + "." + tableName
		}
		buf.WriteString("ALTER TABLE " + tableName + " ADD ")
		writeConstraintDefinition(dialect, buf, m.currentSchema, fkey)
		buf.WriteString(";\n")
	}

	return filenames, bufs, warnings
}
