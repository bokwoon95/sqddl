package ddl

import (
	"bytes"
	"strings"
)

type sqliteMigration struct {
	dropTables   []*Table
	createTables []*Table
	alterTables  []sqliteAlterTable
}

type sqliteAlterTable struct {
	srcTable        *Table
	destTable       *Table
	dropIndexes     []*Index
	dropConstraints []*Constraint
	dropColumns     []*Column
	addColumns      []*Column
	alterColumns    [][2]*Column
	addConstraints  []*Constraint
	createIndexes   []*Index
	columnIsDropped map[string]bool
	columnIsAdded   map[string]bool
}

func newSQLiteMigration(srcCatalog, destCatalog *Catalog, dropObjects bool) sqliteMigration {
	const dialect = DialectSQLite
	m := sqliteMigration{}
	if len(srcCatalog.Schemas) == 0 && len(destCatalog.Schemas) == 0 {
		return m
	}
	if len(destCatalog.Schemas) == 0 {
		if dropObjects {
			for i := range srcCatalog.Schemas[0].Tables {
				srcTable := &srcCatalog.Schemas[0].Tables[i]
				if isVirtualTable(srcTable) {
					continue
				}
				m.dropTables = append(m.dropTables, srcTable)
			}
		}
		return m
	}
	if len(srcCatalog.Schemas) == 0 {
		for i := range destCatalog.Schemas[0].Tables {
			destTable := &destCatalog.Schemas[0].Tables[i]
			if isVirtualTable(destTable) {
				continue
			}
			m.createTables = append(m.createTables, destTable)
		}
		return m
	}

	// Because SQLite doesn't support constraint names, we have to generate it
	// ourselves (we need constraint names because that's how we identify the
	// existence of a constraint).
	for i := range srcCatalog.Schemas {
		srcSchema := &srcCatalog.Schemas[i]
		for j := range srcSchema.Tables {
			srcTable := &srcSchema.Tables[j]
			if isVirtualTable(srcTable) {
				continue
			}
			for k := range srcTable.Constraints {
				srcConstraint := &srcTable.Constraints[k]
				if srcConstraint.ConstraintType != PRIMARY_KEY && srcConstraint.ConstraintType != UNIQUE && srcConstraint.ConstraintType != FOREIGN_KEY {
					continue
				}
				srcConstraint.ConstraintName = GenerateName(srcConstraint.ConstraintType, srcConstraint.TableName, srcConstraint.Columns)
			}
		}
	}
	for i := range destCatalog.Schemas {
		destSchema := &destCatalog.Schemas[i]
		for j := range destSchema.Tables {
			destTable := &destSchema.Tables[j]
			if isVirtualTable(destTable) {
				continue
			}
			for k := range destTable.Constraints {
				destConstraint := &destTable.Constraints[k]
				if destConstraint.ConstraintType != PRIMARY_KEY && destConstraint.ConstraintType != UNIQUE && destConstraint.ConstraintType != FOREIGN_KEY {
					continue
				}
				destConstraint.ConstraintName = GenerateName(destConstraint.ConstraintType, destConstraint.TableName, destConstraint.Columns)
			}
		}
	}

	srcCache, destCache := NewCatalogCache(srcCatalog), NewCatalogCache(destCatalog)
	srcSchema, destSchema := &srcCatalog.Schemas[0], &destCatalog.Schemas[0]
	if dropObjects {
		for i := range srcSchema.Tables {
			srcTable := &srcSchema.Tables[i]
			if srcTable.Ignore {
				continue
			}
			if isVirtualTable(srcTable) {
				continue
			}
			destTable := destCache.GetTable(destSchema, srcTable.TableName)
			if destTable == nil {
				// DROP TABLE.
				m.dropTables = append(m.dropTables, srcTable)
			}
		}
	}
	for i := range destSchema.Tables {
		destTable := &destSchema.Tables[i]
		if destTable.Ignore {
			continue
		}
		if isVirtualTable(destTable) {
			continue
		}
		srcTable := srcCache.GetTable(srcSchema, destTable.TableName)
		if srcTable == nil {
			// CREATE TABLE.
			m.createTables = append(m.createTables, destTable)
			continue
		}
		// ALTER TABLE.
		alterTable := sqliteAlterTable{
			srcTable:        srcTable,
			destTable:       destTable,
			columnIsDropped: make(map[string]bool),
			columnIsAdded:   make(map[string]bool),
		}
		if dropObjects {
			for j := range srcTable.Constraints {
				srcConstraint := &srcTable.Constraints[j]
				if srcConstraint.Ignore {
					continue
				}
				destConstraint := destCache.GetConstraint(destTable, srcConstraint.ConstraintName)
				if destConstraint == nil {
					// DROP CONSTRAINT.
					alterTable.dropConstraints = append(alterTable.dropConstraints, srcConstraint)
				}
			}
			for j := range srcTable.Indexes {
				srcIndex := &srcTable.Indexes[j]
				if srcIndex.Ignore {
					continue
				}
				destIndex := destCache.GetIndex(destTable, srcIndex.IndexName)
				if destIndex == nil {
					// DROP INDEX.
					alterTable.dropIndexes = append(alterTable.dropIndexes, srcIndex)
				}
			}
			for j := range srcTable.Columns {
				srcColumn := &srcTable.Columns[j]
				if srcColumn.Ignore {
					continue
				}
				destColumn := destCache.GetColumn(destTable, srcColumn.ColumnName)
				if destColumn == nil {
					// DROP COLUMN.
					alterTable.dropColumns = append(alterTable.dropColumns, srcColumn)
					alterTable.columnIsDropped[srcColumn.ColumnName] = true
				}
			}
			for j := range destTable.Columns {
				destColumn := &destTable.Columns[j]
				if destColumn.Ignore {
					continue
				}
				srcColumn := srcCache.GetColumn(srcTable, destColumn.ColumnName)
				if srcColumn == nil {
					// ADD COLUMN.
					alterTable.addColumns = append(alterTable.addColumns, destColumn)
					alterTable.columnIsAdded[destColumn.ColumnName] = true
					continue
				}
				columnsAreDifferent := func() bool {
					if srcColumn.ColumnType != destColumn.ColumnType {
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
					return false
				}()
				if columnsAreDifferent {
					// ALTER COLUMN.
					alterTable.alterColumns = append(alterTable.alterColumns, [2]*Column{srcColumn, destColumn})
				}
			}
			for j := range destTable.Indexes {
				destIndex := &destTable.Indexes[j]
				if destIndex.Ignore {
					continue
				}
				srcIndex := srcCache.GetIndex(srcTable, destIndex.IndexName)
				if srcIndex == nil {
					// CREATE INDEX.
					alterTable.createIndexes = append(alterTable.createIndexes, destIndex)
				}
			}
			for j := range destTable.Constraints {
				destConstraint := &destTable.Constraints[j]
				if destConstraint.Ignore {
					continue
				}
				srcConstraint := srcCache.GetConstraint(srcTable, destConstraint.ConstraintName)
				if srcConstraint == nil {
					// ADD CONSTRAINT.
					alterTable.addConstraints = append(alterTable.addConstraints, destConstraint)
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

func (m *sqliteMigration) sql(prefix string) (filenames []string, bufs []*bytes.Buffer, warnings []string) {
	const (
		dialect          = DialectSQLite
		currentSchema    = ""
		defaultCollation = ""
	)
	// ${prefix}.sql
	filenames = append(filenames, prefix+".sql")
	buf := bufpool.Get().(*bytes.Buffer)
	buf.Reset()
	bufs = append(bufs, buf)

	// Figure out which tables have to be copied.
	copyTable := make([]bool, len(m.alterTables))
	hasCopyTable := false
	for i, alterTable := range m.alterTables {
		if len(alterTable.alterColumns) > 0 {
			copyTable[i], hasCopyTable = true, true
			continue
		}
		for _, constraint := range alterTable.dropConstraints {
			if constraint.ConstraintType != FOREIGN_KEY || len(constraint.Columns) > 1 {
				copyTable[i], hasCopyTable = true, true
				continue
			}
			if !alterTable.columnIsDropped[constraint.Columns[0]] {
				copyTable[i], hasCopyTable = true, true
				continue
			}
		}
		for _, constraint := range alterTable.addConstraints {
			if constraint.ConstraintType != FOREIGN_KEY || len(constraint.Columns) > 1 {
				copyTable[i], hasCopyTable = true, true
				continue
			}
			if !alterTable.columnIsAdded[constraint.Columns[0]] {
				copyTable[i], hasCopyTable = true, true
				continue
			}
		}
	}

	if hasCopyTable {
		if buf.Len() > 0 {
			buf.WriteString("\n")
		}
		buf.WriteString("PRAGMA legacy_alter_table = ON;\n")
	}

	// DROP TABLE.
	for _, table := range m.dropTables {
		if buf.Len() > 0 {
			buf.WriteString("\n")
		}
		tableName := QuoteIdentifier(dialect, table.TableName)
		buf.WriteString("DROP TABLE " + tableName + ";\n")
	}

	// CREATE TABLE.
	for _, table := range m.createTables {
		if buf.Len() > 0 {
			buf.WriteString("\n")
		}
		writeCreateTable(dialect, buf, currentSchema, defaultCollation, table, true)
		for _, index := range table.Indexes {
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			writeCreateIndex(dialect, buf, currentSchema, &index, false)
		}
	}

	// ALTER TABLE | COPY TABLE.
	for i, alterTable := range m.alterTables {
		if copyTable[i] {
			// COPY TABLE.
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			alterTable.copyTable(buf)
			continue
		}
		tableName := QuoteIdentifier(dialect, alterTable.destTable.TableName)
		// DROP INDEX.
		for _, index := range alterTable.dropIndexes {
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			indexName := QuoteIdentifier(dialect, index.IndexName)
			buf.WriteString("DROP INDEX " + indexName + ";\n")
		}
		// DROP COLUMN.
		for _, column := range alterTable.dropColumns {
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			columnName := QuoteIdentifier(dialect, column.ColumnName)
			buf.WriteString("ALTER TABLE " + tableName + " DROP COLUMN " + columnName + ";\n")
		}
		// ADD COLUMN.
		for _, column := range alterTable.addColumns {
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			buf.WriteString("ALTER TABLE " + tableName + " ADD COLUMN ")
			writeColumnDefinition(dialect, buf, defaultCollation, column, true)
			buf.WriteString(";\n")
		}
		// CREATE INDEX.
		for _, index := range alterTable.createIndexes {
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			writeCreateIndex(dialect, buf, currentSchema, index, false)
		}
	}

	if hasCopyTable {
		if buf.Len() > 0 {
			buf.WriteString("\n")
		}
		buf.WriteString("PRAGMA legacy_alter_table = OFF;\n")
	}

	if bufs[0].Len() == 0 {
		filenames = filenames[1:]
	}
	return filenames, bufs, warnings
}

func (alterTable *sqliteAlterTable) copyTable(buf *bytes.Buffer) {
	const (
		dialect          = DialectSQLite
		currentSchema    = ""
		defaultCollation = ""
	)
	tableName := QuoteIdentifier(dialect, alterTable.destTable.TableName)
	tableNameNew := QuoteIdentifier(dialect, alterTable.destTable.TableName+"_new")

	// CREATE "${TableName}_new".
	tableNew := *alterTable.destTable
	tableNew.TableName = alterTable.destTable.TableName + "_new"
	tableNew.SQL = strings.Replace(alterTable.destTable.SQL, "CREATE TABLE "+tableName, "CREATE TABLE "+tableNameNew, 1)
	writeCreateTable(dialect, buf, currentSchema, defaultCollation, &tableNew, true)

	// INSERT INTO "${TableName}_new" SELECT ... FROM "$TableName".
	isDestColumn := make(map[string]bool)
	for _, destColumn := range alterTable.destTable.Columns {
		if destColumn.IsGenerated || destColumn.GeneratedExpr != "" {
			continue
		}
		isDestColumn[destColumn.ColumnName] = true
	}
	insertColumns := make([]string, 0, len(alterTable.srcTable.Columns))
	for _, srcColumn := range alterTable.srcTable.Columns {
		if !isDestColumn[srcColumn.ColumnName] {
			continue
		}
		insertColumns = append(insertColumns, srcColumn.ColumnName)
	}
	buf.WriteString("INSERT INTO " + tableNameNew + "\n    (")
	for i, insertColumn := range insertColumns {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(QuoteIdentifier(dialect, insertColumn))
	}
	buf.WriteString(")\nSELECT\n    ")
	for i, insertColumn := range insertColumns {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(QuoteIdentifier(dialect, insertColumn))
	}
	buf.WriteString("\nFROM\n    " + tableName + "\n;\n")

	// DROP "$TableName".
	buf.WriteString("DROP TABLE " + tableName + ";\n")

	// ALTER "${TableName}_new" RENAME TO "$TableName".
	buf.WriteString("ALTER TABLE " + tableNameNew + " RENAME TO " + tableName + ";\n")

	// CREATE INDEX.
	for i := range alterTable.destTable.Indexes {
		destIndex := &alterTable.destTable.Indexes[i]
		if buf.Len() > 0 {
			buf.WriteString("\n")
		}
		writeCreateIndex(dialect, buf, "", destIndex, false)
	}

	// CREATE TRIGGER.
	triggers := alterTable.destTable.Triggers
	if len(triggers) == 0 {
		// If we cannot find triggers in the destTable, fall back to the
		// srcTable. This will often be the case if destTable was derived from
		// Go structs (which don't support defining triggers) and srcTable was
		// derived from a database.
		triggers = alterTable.srcTable.Triggers
	}
	for _, trigger := range triggers {
		if buf.Len() > 0 {
			buf.WriteString("\n")
		}
		buf.WriteString(trigger.SQL + "\n")
	}
}
