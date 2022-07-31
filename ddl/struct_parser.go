package ddl

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"reflect"
	"strconv"
	"strings"
	"unicode"

	"github.com/bokwoon95/sq"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// StructParser is used to parse Go source code into TableStructs.
type StructParser struct {
	TableStructs       TableStructs
	parserDiagnostics  *parserDiagnostics
	dialect            string
	locations          map[[2]string]location
	columnExplicitType map[[3]string]struct{}
	cache              *CatalogCache
}

// NewStructParser creates a new StructParser. An existing token.Fileset can be
// passed in. If not, passing in nil is fine and a new token.FileSet will be
// instantiated.
func NewStructParser(fset *token.FileSet) *StructParser {
	if fset == nil {
		fset = token.NewFileSet()
	}
	return &StructParser{parserDiagnostics: &parserDiagnostics{
		fset: fset,
	}}
}

// VisitNode is a callback function that populates the TableStructs when passed
// to ast.Inspect().
func (p *StructParser) VisitNode(node ast.Node) bool {
	switch node.(type) {
	case *ast.File, *ast.GenDecl:
		return true
	}
	p.VisitStruct(node)
	return false
}

// VisitStruct is a callback function that populates the TableStructs when
// passed to inspect.Inspector.Preorder(). It expects the node to be of type
// *ast.TypeSpec.
func (p *StructParser) VisitStruct(node ast.Node) {
	// Is it a type declaration?
	typeSpec, ok := node.(*ast.TypeSpec)
	if !ok {
		return
	}
	// Is it a type declaration for a struct?
	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		return
	}
	// Does the struct have fields?
	if structType.Fields == nil {
		return
	}
	tableStruct := TableStruct{
		Name:   typeSpec.Name.Name,
		Fields: make([]StructField, 0, len(structType.Fields.List)),
	}
	for i, astField := range structType.Fields.List {
		var structField StructField
		// Name
		if len(astField.Names) > 0 && astField.Names[0] != nil {
			structField.Name = astField.Names[0].Name
		}
		// Type
		if typ, ok := astField.Type.(*ast.SelectorExpr); ok {
			if x, ok := typ.X.(*ast.Ident); ok {
				structField.Type = x.Name + "." + typ.Sel.Name
			}
		} else if typ, ok := astField.Type.(*ast.Ident); ok {
			structField.Type = typ.Name
		}
		// Tag
		if astField.Tag != nil {
			structField.tagPos = astField.Tag.Pos()
			if tag, err := strconv.Unquote(astField.Tag.Value); err == nil {
				structField.NameTag = reflect.StructTag(tag).Get("sq")
				structField.Modifiers, err = NewModifiers(reflect.StructTag(tag).Get("ddl"))
				if err != nil {
					loc := location{
						pos:        structField.tagPos,
						structName: tableStruct.Name,
						fieldName:  structField.Name,
					}
					p.report(loc, err.Error())
					continue
				}
			}
		}
		// If the first field is not sq.TableStruct, skip this struct entirely.
		if i == 0 {
			if structField.Name != "" || structField.Type != "sq.TableStruct" {
				structNameIsUppercase := true
				for _, char := range tableStruct.Name {
					if !unicode.IsUpper(char) {
						structNameIsUppercase = false
						break
					}
				}
				if structNameIsUppercase && structField.Type != "sq.TableStruct" {
					loc := location{pos: structField.tagPos}
					p.report(loc, "struct "+tableStruct.Name+" is all uppercase but no sq.TableStruct field was found")
				}
				return
			}
		}
		tableStruct.Fields = append(tableStruct.Fields, structField)
	}
	p.TableStructs = append(p.TableStructs, tableStruct)
}

// ParseFile parses an fs.File containing Go source code and populates the
// TableStructs.
func (p *StructParser) ParseFile(f fs.File) error {
	fileinfo, err := f.Stat()
	if err != nil {
		return err
	}
	file, err := parser.ParseFile(p.parserDiagnostics.fset, fileinfo.Name(), f, 0)
	if err != nil {
		return err
	}
	ast.Inspect(file, p.VisitNode)
	return p.Error()
}

// WriteCatalog populates the Catalog using the StructParser's TableStructs.
func (p *StructParser) WriteCatalog(catalog *Catalog) error {
	p.dialect = catalog.Dialect
	p.locations = make(map[[2]string]location)
	p.columnExplicitType = make(map[[3]string]struct{})
	p.cache = NewCatalogCache(catalog)

	for _, tableStruct := range p.TableStructs {
		var tableSchema string
		tableName := strings.ToLower(tableStruct.Name)
		if tableStruct.Fields[0].NameTag != "" {
			tableName = tableStruct.Fields[0].NameTag
		}
		if i := strings.IndexByte(tableName, '.'); i >= 0 {
			tableSchema, tableName = tableName[:i], tableName[i+1:]
		}
		if tableSchema == "" && catalog.CurrentSchema != "" {
			tableSchema = catalog.CurrentSchema
		}

		schema := p.cache.GetOrCreateSchema(catalog, tableSchema)
		table := p.cache.GetOrCreateTable(schema, tableName)

		// The main loop.
		for _, structField := range tableStruct.Fields {
			loc := location{
				pos:        structField.tagPos,
				structName: tableStruct.Name,
				fieldName:  structField.Name,
			}
			if (structField.Name == "" && structField.Type == "sq.TableStruct") || (structField.Name == "_" && structField.Type == "struct{}") {
				p.parseTableModifiers(table, loc, structField.Modifiers)
				continue
			}
			columnName := strings.ToLower(structField.Name)
			if structField.NameTag != "" {
				columnName = structField.NameTag
			}
			var columnType, characterLength string
			switch structField.Type {
			case "sq.AnyField":
			case "sq.ArrayField":
				switch p.dialect {
				case sq.DialectSQLite, sq.DialectMySQL:
					columnType = "JSON"
				case sq.DialectPostgres:
					columnType = "TEXT[]"
				case sq.DialectSQLServer:
					columnType, characterLength = "NVARCHAR(MAX)", "MAX"
				default:
					columnType, characterLength = "VARCHAR(255)", "255"
				}
			case "sq.BinaryField":
				switch p.dialect {
				case sq.DialectSQLite:
					columnType = "BLOB"
				case sq.DialectPostgres:
					columnType = "BYTEA"
				case sq.DialectMySQL:
					columnType = "MEDIUMBLOB"
				case sq.DialectSQLServer:
					columnType, characterLength = "VARBINARY(MAX)", "MAX"
				default:
					columnType = "BINARY"
				}
			case "sq.BooleanField":
				switch p.dialect {
				case sq.DialectSQLServer:
					columnType = "BIT"
				default:
					columnType = "BOOLEAN"
				}
			case "sq.EnumField":
				switch p.dialect {
				case sq.DialectSQLite, sq.DialectPostgres:
					columnType = "TEXT"
				case sq.DialectSQLServer:
					columnType, characterLength = "NVARCHAR(255)", "255"
				default:
					columnType, characterLength = "VARCHAR(255)", "255"
				}
			case "sq.JSONField":
				switch p.dialect {
				case sq.DialectSQLite, sq.DialectMySQL:
					columnType = "JSON"
				case sq.DialectPostgres:
					columnType = "JSONB"
				case sq.DialectSQLServer:
					columnType, characterLength = "NVARCHAR(MAX)", "MAX"
				default:
					columnType = "VARCHAR(255)"
				}
			case "sq.NumberField":
				columnType = "INT"
			case "sq.StringField":
				switch p.dialect {
				case sq.DialectSQLite, sq.DialectPostgres:
					columnType = "TEXT"
				case sq.DialectSQLServer:
					columnType, characterLength = "NVARCHAR(255)", "255"
				default:
					columnType, characterLength = "VARCHAR(255)", "255"
				}
			case "sq.TimeField":
				switch p.dialect {
				case sq.DialectPostgres:
					columnType = "TIMESTAMPTZ"
				case sq.DialectSQLServer:
					columnType = "DATETIMEOFFSET"
				default:
					columnType = "DATETIME"
				}
			case "sq.UUIDField":
				switch p.dialect {
				case sq.DialectSQLite, sq.DialectPostgres:
					columnType = "UUID"
				default:
					columnType = "BINARY(16)"
				}
			default:
				continue
			}
			if characterLength != "" {
				column := p.cache.GetOrCreateColumn(table, columnName, columnType)
				column.CharacterLength = characterLength
			}
			p.parseColumnModifiers(table, columnName, columnType, loc, structField.Modifiers)
		}

		// Validate column existence for PRIMARY KEY and UNIQUE constraints.
		for _, constraint := range table.Constraints {
			if constraint.Ignore {
				continue
			}
			hasInvalidColumn := false
			for _, columnName := range constraint.Columns {
				column := p.cache.GetColumn(table, columnName)
				if column == nil {
					hasInvalidColumn = true
					loc := p.locations[[2]string{table.TableSchema, constraint.ConstraintName}]
					p.report(loc, strings.Join(constraint.Columns, ",")+": "+columnName+" does not exist in the table")
				}
			}
			if hasInvalidColumn {
				continue
			}
			// set IsUnique and IsPrimaryKey for the corresponding columns
			if len(constraint.Columns) != 1 || (constraint.ConstraintType != PRIMARY_KEY && constraint.ConstraintType != UNIQUE && constraint.ConstraintType != FOREIGN_KEY) {
				continue
			}
			columnName := constraint.Columns[0]
			column := p.cache.GetColumn(table, columnName)
			switch constraint.ConstraintType {
			case PRIMARY_KEY:
				column.IsPrimaryKey = true
				if catalog.Dialect == sq.DialectSQLite && strings.EqualFold(column.ColumnType, "INT") {
					if _, ok := p.columnExplicitType[[3]string{table.TableSchema, table.TableName, columnName}]; !ok {
						column.ColumnType = "INTEGER"
					}
				}
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

		// Validate column existence for indexes.
		for _, index := range table.Indexes {
			if index.Ignore {
				continue
			}
			for _, columnName := range index.Columns {
				column := p.cache.GetColumn(table, columnName)
				if column == nil {
					loc := p.locations[[2]string{table.TableSchema, index.IndexName}]
					p.report(loc, strings.Join(index.Columns, ",")+": "+columnName+" does not exist in the table")
				}
			}
		}

		// Set PRIMARY KEY columns to NOT NULL.
		pkey := p.cache.GetPrimaryKey(table)
		if pkey != nil && !pkey.Ignore {
			for _, columnName := range pkey.Columns {
				column := p.cache.GetColumn(table, columnName)
				if column == nil {
					continue
				}
				if catalog.Dialect == sq.DialectSQLite && column.ColumnType == "INTEGER" && column.IsPrimaryKey {
					// SQLite forbids INTEGER PRIMARY KEY (alias for ROWID) columns
					// from being marked as NOT NULL (since they can never be NULL).
					continue
				}
				column.IsNotNull = true
			}
		}
	}

	// Validate column existence for FOREIGN KEY constraints.
	for _, schema := range catalog.Schemas {
		for _, table := range schema.Tables {
			for _, constraint := range table.Constraints {
				if constraint.Ignore || constraint.ConstraintType != FOREIGN_KEY {
					continue
				}
				// We need the location of the failing constraint so that we
				// can inform the user where it is. If we can't find it,
				// continue.
				loc, ok := p.locations[[2]string{table.TableSchema, constraint.ConstraintName}]
				if !ok {
					continue
				}
				schemaName := constraint.ReferencesSchema
				if schemaName == "" {
					schemaName = catalog.CurrentSchema
				}
				refschema := p.cache.GetSchema(catalog, schemaName)
				if refschema == nil {
					p.report(loc, fmt.Sprintf("schema %s does not exist", schemaName))
					continue
				}
				reftable := p.cache.GetTable(refschema, constraint.ReferencesTable)
				tableName := constraint.ReferencesTable
				if reftable == nil {
					if schemaName != "" {
						tableName = schemaName + "." + tableName
					}
					p.report(loc, fmt.Sprintf("table %s does not exist", tableName))
					continue
				}
				for _, columnName := range constraint.ReferencesColumns {
					refcolumn := p.cache.GetColumn(reftable, columnName)
					if refcolumn == nil {
						columnName = tableName + "." + columnName
						p.report(loc, fmt.Sprintf("column %s does not exist", columnName))
						continue
					}
				}
			}
		}
	}
	return p.Error()
}

func (p *StructParser) parseIndexModifier(table *Table, columnNames []string, loc location, m *Modifier) {
	err := m.ParseRawValue()
	if err != nil {
		p.report(loc, err.Error())
		return
	}
	if m.Value != "" && m.Value != "." {
		columnNames = strings.Split(m.Value, ",")
	}
	if len(columnNames) == 0 {
		p.report(loc, "no column provided")
	}
	indexName := generateName(INDEX, table.TableName, columnNames)
	p.locations[[2]string{table.TableSchema, indexName}] = loc
	index := p.cache.GetOrCreateIndex(table, indexName, columnNames)
	index.TableSchema = table.TableSchema
	index.TableName = table.TableName
	index.Ignore = m.ExcludesDialect(p.dialect)
	for i := range m.Submodifiers {
		submodifier := &m.Submodifiers[i]
		if submodifier.ExcludesDialect(p.dialect) {
			continue
		}
		switch submodifier.Name {
		case "unique":
			index.IsUnique = true
		case "using":
			if p.dialect != sq.DialectPostgres && p.dialect != sq.DialectMySQL {
				continue
			}
			index.IndexType = submodifier.RawValue
		default:
			p.report(loc, "unknown modifier "+strconv.Quote(submodifier.Name))
		}
	}
}

func (p *StructParser) parsePrimaryKeyUniqueModifier(table *Table, columnNames []string, loc location, m *Modifier) {
	err := m.ParseRawValue()
	if err != nil {
		p.report(loc, err.Error())
		return
	}
	constraintType := PRIMARY_KEY
	if strings.EqualFold(UNIQUE, m.Name) {
		constraintType = UNIQUE
	}
	if m.Value != "" && m.Value != "." {
		columnNames = strings.Split(m.Value, ",")
	}
	if len(columnNames) == 0 {
		p.report(loc, "no column provided")
	}
	constraintName := generateName(constraintType, table.TableName, columnNames)
	if p.dialect == sq.DialectMySQL && constraintType == PRIMARY_KEY {
		constraintName = "PRIMARY"
	}
	p.locations[[2]string{table.TableSchema, constraintName}] = loc
	constraint := p.cache.GetOrCreateConstraint(table, constraintName, constraintType, columnNames)
	constraint.TableSchema = table.TableSchema
	constraint.TableName = table.TableName
	constraint.Ignore = m.ExcludesDialect(p.dialect)
	for i := range m.Submodifiers {
		submodifier := &m.Submodifiers[i]
		if submodifier.ExcludesDialect(p.dialect) {
			continue
		}
		switch submodifier.Name {
		case "deferrable":
			if p.dialect != sq.DialectSQLite && p.dialect != sq.DialectPostgres {
				continue
			}
			constraint.IsDeferrable = true
		case "deferred":
			if p.dialect != sq.DialectSQLite && p.dialect != sq.DialectPostgres {
				continue
			}
			constraint.IsDeferrable = true
			constraint.IsInitiallyDeferred = true
		default:
			p.report(loc, "unknown modifier "+strconv.Quote(submodifier.Name))
		}
	}
}

func (p *StructParser) parseForeignKeyModifier(table *Table, loc location, m *Modifier) {
	err := m.ParseRawValue()
	if err != nil {
		p.report(loc, err.Error())
		return
	}
	if m.Value == "" || m.Value == "." {
		p.report(loc, "no column(s) provided")
		return
	}
	if len(m.Submodifiers) == 0 {
		p.report(loc, "no referenced column(s) provided")
		return
	}
	columnNames := strings.Split(m.Value, ",")
	constraintName := generateName(FOREIGN_KEY, table.TableName, columnNames)
	p.locations[[2]string{table.TableSchema, constraintName}] = loc
	constraint := p.cache.GetOrCreateConstraint(table, constraintName, FOREIGN_KEY, columnNames)
	constraint.TableSchema = table.TableSchema
	constraint.TableName = table.TableName
	constraint.Ignore = m.ExcludesDialect(p.dialect)
	for i := range m.Submodifiers {
		submodifier := &m.Submodifiers[i]
		if submodifier.ExcludesDialect(p.dialect) {
			continue
		}
		switch submodifier.Name {
		case "references":
			switch parts := strings.SplitN(submodifier.RawValue, ".", 3); len(parts) {
			case 1:
				constraint.ReferencesTable = parts[0]
				constraint.ReferencesColumns = columnNames
			case 2:
				constraint.ReferencesTable = parts[0]
				constraint.ReferencesColumns = strings.Split(parts[1], ",")
			case 3:
				constraint.ReferencesSchema = parts[0]
				constraint.ReferencesTable = parts[1]
				constraint.ReferencesColumns = strings.Split(parts[2], ",")
			}
		case "onupdate":
			switch submodifier.RawValue {
			case "cascade":
				constraint.UpdateRule = CASCADE
			case "restrict":
				if p.dialect == sq.DialectSQLServer {
					constraint.UpdateRule = NO_ACTION
				} else {
					constraint.UpdateRule = RESTRICT
				}
			case "noaction":
				constraint.UpdateRule = NO_ACTION
			case "setnull":
				constraint.UpdateRule = SET_NULL
			case "setdefault":
				constraint.UpdateRule = SET_DEFAULT
			case "":
				constraint.UpdateRule = ""
			default:
				loc.keys = append(loc.keys, submodifier.Name)
				p.report(loc, "unknown value "+strconv.Quote(submodifier.RawValue))
			}
		case "ondelete":
			switch submodifier.RawValue {
			case "cascade":
				constraint.DeleteRule = CASCADE
			case "restrict":
				if p.dialect == sq.DialectSQLServer {
					constraint.DeleteRule = NO_ACTION
				} else {
					constraint.DeleteRule = RESTRICT
				}
			case "noaction":
				constraint.DeleteRule = NO_ACTION
			case "setnull":
				constraint.DeleteRule = SET_NULL
			case "setdefault":
				constraint.DeleteRule = SET_DEFAULT
			case "":
				constraint.DeleteRule = ""
			default:
				loc.keys = append(loc.keys, submodifier.Name)
				p.report(loc, "unknown value "+strconv.Quote(submodifier.RawValue))
			}
		case "deferrable":
			if p.dialect != sq.DialectSQLite && p.dialect != sq.DialectPostgres {
				continue
			}
			constraint.IsDeferrable = true
		case "deferred":
			if p.dialect != sq.DialectSQLite && p.dialect != sq.DialectPostgres {
				continue
			}
			constraint.IsDeferrable = true
			constraint.IsInitiallyDeferred = true
		case "index":
			loc.keys = append(loc.keys, submodifier.Name)
			p.parseIndexModifier(table, columnNames, loc, submodifier)
		default:
			p.report(loc, "unknown modifier "+strconv.Quote(submodifier.Name))
		}
	}
}

func (p *StructParser) parseReferencesModifier(table *Table, columnName string, loc location, m *Modifier) {
	err := m.ParseRawValue()
	if err != nil {
		p.report(loc, err.Error())
		return
	}
	if columnName == "" {
		p.report(loc, "no column provided")
	}
	constraintName := generateName(FOREIGN_KEY, table.TableName, []string{columnName})
	p.locations[[2]string{table.TableSchema, constraintName}] = loc

	constraint := p.cache.GetOrCreateConstraint(table, constraintName, FOREIGN_KEY, []string{columnName})
	constraint.TableSchema = table.TableSchema
	constraint.TableName = table.TableName
	constraint.Ignore = m.ExcludesDialect(p.dialect)
	switch parts := strings.SplitN(m.Value, ".", 3); len(parts) {
	case 1:
		constraint.ReferencesTable = parts[0]
		constraint.ReferencesColumns = []string{columnName}
	case 2:
		constraint.ReferencesTable = parts[0]
		constraint.ReferencesColumns = strings.Split(parts[1], ",")
	case 3:
		constraint.ReferencesSchema = parts[0]
		constraint.ReferencesTable = parts[1]
		constraint.ReferencesColumns = strings.Split(parts[2], ",")
	}

	for i, submodifier := range m.Submodifiers {
		if submodifier.ExcludesDialect(p.dialect) {
			continue
		}
		switch submodifier.Name {
		case "onupdate":
			switch submodifier.RawValue {
			case "cascade":
				constraint.UpdateRule = CASCADE
			case "restrict":
				if p.dialect == sq.DialectSQLServer {
					constraint.UpdateRule = NO_ACTION
				} else {
					constraint.UpdateRule = RESTRICT
				}
			case "noaction":
				constraint.UpdateRule = NO_ACTION
			case "setnull":
				constraint.UpdateRule = SET_NULL
			case "setdefault":
				constraint.UpdateRule = SET_DEFAULT
			case "":
				constraint.UpdateRule = ""
			default:
				loc.keys = append(loc.keys, submodifier.Name)
				p.report(loc, "unknown value "+strconv.Quote(submodifier.RawValue))
			}
		case "ondelete":
			switch submodifier.RawValue {
			case "cascade":
				constraint.DeleteRule = CASCADE
			case "restrict":
				if p.dialect == sq.DialectSQLServer {
					constraint.DeleteRule = NO_ACTION
				} else {
					constraint.DeleteRule = RESTRICT
				}
			case "noaction":
				constraint.DeleteRule = NO_ACTION
			case "setnull":
				constraint.DeleteRule = SET_NULL
			case "setdefault":
				constraint.DeleteRule = SET_DEFAULT
			case "":
				constraint.DeleteRule = ""
			default:
				loc.keys = append(loc.keys, submodifier.Name)
				p.report(loc, "unknown value "+strconv.Quote(submodifier.RawValue))
			}
		case "deferrable":
			if p.dialect == sq.DialectSQLite || p.dialect == sq.DialectPostgres {
				constraint.IsDeferrable = true
			}
		case "deferred":
			if p.dialect == sq.DialectSQLite || p.dialect == sq.DialectPostgres {
				constraint.IsDeferrable = true
				constraint.IsInitiallyDeferred = true
			}
		case "index":
			loc.keys = append(loc.keys, submodifier.Name)
			p.parseIndexModifier(table, []string{columnName}, loc, &m.Submodifiers[i])
		default:
			p.report(loc, "unknown modifier "+strconv.Quote(submodifier.Name))
		}
	}
}

func (p *StructParser) parseColumnModifiers(table *Table, columnName, columnType string, loc location, modifiers Modifiers) {
	column := p.cache.GetOrCreateColumn(table, columnName, columnType)
	column.TableSchema = table.TableSchema
	column.TableName = table.TableName
	column.IsEnum = columnType == "sq.EnumField"

	var dialects []string
	for i := range modifiers {
		modifier := &modifiers[i]
		if len(modifier.Dialects) == 0 {
			modifier.Dialects = dialects
		}
		if modifier.ExcludesDialect(p.dialect) {
			continue
		}
		switch modifier.Name {
		case "type":
			p.columnExplicitType[[3]string{table.TableSchema, table.TableName, columnName}] = struct{}{}
			column.ColumnType = modifier.RawValue
			normalizedType, arg1, arg2 := normalizeColumnType(p.dialect, column.ColumnType)
			switch normalizedType {
			case "VARBINARY", "BINARY", "NVARCHAR", "VARCHAR", "CHAR":
				if arg1 != "" {
					column.CharacterLength = arg1
				} else if column.CharacterLength != "" {
					column.ColumnType = column.ColumnType + "(" + column.CharacterLength + ")"
				}
				column.NumericPrecision, column.NumericScale = "", ""
			case "NUMERIC":
				if arg1 != "" {
					column.NumericPrecision = arg1
				}
				if arg2 != "" {
					column.NumericScale = arg2
				}
				column.CharacterLength = ""
			default:
				column.CharacterLength, column.NumericPrecision, column.NumericScale = "", "", ""
			}
		case "len":
			column.CharacterLength = modifier.RawValue
			if column.CharacterLength != "" {
				switch p.dialect {
				case sq.DialectPostgres, sq.DialectMySQL:
					column.ColumnType = "VARCHAR(" + column.CharacterLength + ")"
				case sq.DialectSQLServer:
					column.ColumnType = "NVARCHAR(" + column.CharacterLength + ")"
				}
			}
		case "auto_increment":
			if p.dialect != sq.DialectMySQL {
				continue
			}
			column.IsAutoincrement = true
		case "autoincrement":
			if p.dialect != sq.DialectSQLite {
				continue
			}
			column.IsAutoincrement = true
		case "identity":
			switch p.dialect {
			case sq.DialectPostgres:
				column.ColumnIdentity = DEFAULT_IDENTITY
			case sq.DialectSQLServer:
				column.ColumnIdentity = IDENTITY
			}
		case "alwaysidentity":
			switch p.dialect {
			case sq.DialectPostgres:
				column.ColumnIdentity = ALWAYS_IDENTITY
			case sq.DialectSQLServer:
				column.ColumnIdentity = IDENTITY
			}
		case "notnull":
			column.IsNotNull = true
		case "onupdatecurrenttimestamp":
			if p.dialect != sq.DialectMySQL {
				continue
			}
			column.OnUpdateCurrentTimestamp = true
		case "collate":
			column.CollationName = modifier.RawValue
		case "default":
			if p.dialect != sq.DialectPostgres && !isLiteral(modifier.RawValue) {
				column.ColumnDefault = wrapBrackets(modifier.RawValue)
				continue
			}
			column.ColumnDefault = modifier.RawValue
			if p.dialect == sq.DialectSQLServer {
				if strings.EqualFold(column.ColumnDefault, "TRUE") {
					column.ColumnDefault = "1"
				} else if strings.EqualFold(column.ColumnDefault, "FALSE") {
					column.ColumnDefault = "0"
				}
			}
		case "generated":
			column.IsGenerated = true
		case "dialect":
			if modifier.RawValue == "" {
				loc.keys = []string{modifier.Name}
				p.report(loc, "dialect value cannot be blank")
				continue
			}
			column.Ignore = true
			dialects = strings.Split(modifier.RawValue, ",")
			for _, dialect := range dialects {
				if p.dialect == dialect {
					column.Ignore = false
					break
				}
			}
		case "index":
			loc.keys = []string{modifier.Name}
			p.parseIndexModifier(table, []string{columnName}, loc, modifier)
		case "primarykey", "unique":
			loc.keys = []string{modifier.Name}
			p.parsePrimaryKeyUniqueModifier(table, []string{columnName}, loc, modifier)
		case "foreignkey":
			loc.keys = []string{modifier.Name}
			p.parseForeignKeyModifier(table, loc, modifier)
		case "references":
			loc.keys = []string{modifier.Name}
			p.parseReferencesModifier(table, columnName, loc, modifier)
		default:
			p.report(loc, "unknown modifier "+strconv.Quote(modifier.Name))
		}
	}
}

func (p *StructParser) parseTableModifiers(table *Table, loc location, modifiers Modifiers) {
	var dialects []string
	for i := range modifiers {
		modifier := &modifiers[i]
		if len(modifier.Dialects) == 0 {
			modifier.Dialects = dialects
		}
		switch modifier.Name {
		case "dialect":
			if modifier.RawValue == "" {
				loc.keys = []string{modifier.Name}
				p.report(loc, "dialect value cannot be blank")
				continue
			}
			table.Ignore = true
			dialects = strings.Split(modifier.RawValue, ",")
			for _, dialect := range dialects {
				if p.dialect == dialect {
					table.Ignore = false
					break
				}
			}
		case "index":
			loc.keys = []string{modifier.Name}
			p.parseIndexModifier(table, nil, loc, modifier)
		case "primarykey", "unique":
			loc.keys = []string{modifier.Name}
			p.parsePrimaryKeyUniqueModifier(table, nil, loc, modifier)
		case "foreignkey":
			loc.keys = []string{modifier.Name}
			p.parseForeignKeyModifier(table, loc, modifier)
		case "virtual":
			if p.dialect != sq.DialectSQLite {
				continue
			}
			table.IsVirtual = true
		default:
			p.report(loc, "unknown modifier "+strconv.Quote(modifier.Name))
		}
	}
}

func (p *StructParser) report(loc location, msg string) {
	p.parserDiagnostics.locs = append(p.parserDiagnostics.locs, loc)
	p.parserDiagnostics.msgs = append(p.parserDiagnostics.msgs, msg)
}

type location struct {
	pos        token.Pos
	structName string
	fieldName  string
	keys       []string
}

type parserDiagnostics struct {
	fset *token.FileSet
	locs []location
	msgs []string
}

// Diagnostics returns the errors encountered after calling WriteCatalog in a
// structured format.
func (p *StructParser) Diagnostics() ([]token.Pos, []string) {
	positions := make([]token.Pos, 0, len(p.parserDiagnostics.msgs))
	msgs := make([]string, 0, len(p.parserDiagnostics.msgs))
	for i, msg := range p.parserDiagnostics.msgs {
		var loc location
		if i < len(p.parserDiagnostics.locs) {
			loc = p.parserDiagnostics.locs[i]
		}
		n := len(loc.structName) + len(".") + len(loc.fieldName) + len(": ")
		for _, key := range loc.keys {
			n += len(".") + len(key)
		}
		n += len(": ") + len(msg)
		var b strings.Builder
		b.Grow(n)
		if loc.structName != "" && loc.fieldName != "" {
			b.WriteString(loc.structName + "." + loc.fieldName + ": ")
		}
		for j, key := range loc.keys {
			if j > 0 {
				b.WriteString(".")
			}
			b.WriteString(key)
		}
		b.WriteString(": " + msg)
		positions = append(positions, loc.pos)
		msgs = append(msgs, b.String())
	}
	return positions, msgs
}

// Error returns the errors encountered after calling WriteCatalog.
func (p *StructParser) Error() error {
	if len(p.parserDiagnostics.msgs) > 0 {
		return p.parserDiagnostics
	}
	return nil
}

// Error implements the error interface.
func (d *parserDiagnostics) Error() string {
	buf := bufpool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufpool.Put(buf)
	for i, msg := range d.msgs {
		if i > 0 {
			buf.WriteString("\n")
		}
		if i >= len(d.locs) {
			buf.WriteString(msg)
			continue
		}
		loc := d.locs[i]
		if d.fset != nil {
			pos := d.fset.Position(loc.pos)
			if pos.IsValid() {
				buf.WriteString(pos.String() + ": ")
			}
		}
		if loc.structName != "" && loc.fieldName != "" {
			buf.WriteString(loc.structName + "." + loc.fieldName + ": ")
		}
		if len(loc.keys) > 0 {
			for j, key := range loc.keys {
				if j > 0 {
					buf.WriteByte('.')
				}
				buf.WriteString(key)
			}
			buf.WriteString(": ")
		}
		buf.WriteString(msg)
	}
	return buf.String()
}

// Analyzer is an &analysis.Analyzer which can be used in a custom linter.
var Analyzer = &analysis.Analyzer{
	Name:     "ddl",
	Doc:      "validates ddl structs",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run: func(pass *analysis.Pass) (any, error) {
		inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
		nodeFilter := []ast.Node{(*ast.TypeSpec)(nil)}
		p := NewStructParser(pass.Fset)
		inspect.Preorder(nodeFilter, p.VisitStruct)
		var catalog Catalog
		_ = p.WriteCatalog(&catalog)
		positions, msgs := p.Diagnostics()
		if len(msgs) == 0 {
			return nil, nil
		}
		for i, msg := range msgs {
			var pos token.Pos
			if i < len(positions) {
				pos = positions[i]
			}
			pass.Reportf(pos, msg)
		}
		return nil, nil
	},
}
