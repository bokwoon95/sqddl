package ddl

import (
	"bytes"
	"go/token"
	"strconv"
	"strings"

	"github.com/bokwoon95/sq"
)

// TableStructs is a slice of TableStructs.
type TableStructs []TableStruct

// TableStruct represents a table struct.
type TableStruct struct {
	// Name is the name of the table struct.
	Name string

	// Fields are the table struct fields.
	Fields []StructField
}

// StructField represents a struct field within a table struct.
type StructField struct {
	// Name is the name of the struct field.
	Name string

	// Type is the type of the struct field.
	Type string

	// NameTag is the value for the "sq" struct tag.
	NameTag string

	// Modifiers are the parsed modifiers for the "ddl" struct tag.
	Modifiers Modifiers

	// tagPos tracks where in the source code the struct tag appeared in. Used
	// for error reporting.
	tagPos token.Pos
}

// ReadCatalog reads from a catalog and populates the TableStructs accordingly.
func (s *TableStructs) ReadCatalog(catalog *Catalog) error {
	buf := bufpool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufpool.Put(buf)
	for _, schema := range catalog.Schemas {
		for _, table := range schema.Tables {
			tableStruct := TableStruct{
				Name:   strings.ToUpper(strings.ReplaceAll(table.TableName, " ", "_")),
				Fields: make([]StructField, 0, len(table.Columns)+1),
			}
			tableStruct.Fields = append(tableStruct.Fields, StructField{Type: "sq.TableStruct"})
			firstField := &tableStruct.Fields[0]
			if (table.TableSchema != "" && table.TableSchema != catalog.CurrentSchema) || needsQuoting(table.TableName) {
				if table.TableSchema != "" {
					firstField.NameTag = table.TableSchema + "." + table.TableName
				} else {
					firstField.NameTag = table.TableName
				}
			}
			if catalog.Dialect == sq.DialectSQLite && isVirtualTable(&table) {
				firstField.Modifiers = append(firstField.Modifiers, Modifier{Name: "virtual"})
			}
			constraintModifierList := make([]*Modifier, 0, len(table.Constraints))
			indexModifierList := make([]*Modifier, 0, len(table.Indexes))
			var primarykeyModifier *Modifier
			uniqueModifiers := make(map[string]*Modifier)
			foreignkeyModifiers := make(map[string]*Modifier)
			indexModifiers := make(map[string]*Modifier)
			addedModifier := make(map[*Modifier]bool)
			for _, constraint := range table.Constraints {
				if constraint.Ignore {
					continue
				}
				columnNames := strings.Join(constraint.Columns, ",")
				m := &Modifier{Value: columnNames}
				switch constraint.ConstraintType {
				case PRIMARY_KEY:
					m.Name = "primarykey"
					primarykeyModifier = m
				case UNIQUE:
					m.Name = "unique"
					uniqueModifiers[columnNames] = m
				case FOREIGN_KEY:
					m.Name = "foreignkey"
					foreignkeyModifiers[columnNames] = m
					buf.Reset()
					if constraint.ReferencesSchema != "" && constraint.ReferencesSchema != catalog.CurrentSchema {
						buf.WriteString(constraint.ReferencesSchema + ".")
					}
					buf.WriteString(constraint.ReferencesTable)
					columnsEqual := true
					for i := range constraint.Columns {
						if i >= len(constraint.ReferencesColumns) {
							columnsEqual = false
							break
						}
						if constraint.Columns[i] != constraint.ReferencesColumns[i] {
							columnsEqual = false
							break
						}
					}
					if !columnsEqual {
						buf.WriteString("." + strings.Join(constraint.ReferencesColumns, ","))
					}
					// references
					m.Submodifiers = append(m.Submodifiers, Modifier{Name: "references", RawValue: buf.String()})
					// onupdate
					if constraint.UpdateRule != "" && constraint.UpdateRule != NO_ACTION {
						m.Submodifiers = append(m.Submodifiers, Modifier{
							Name:  "onupdate",
							Value: strings.ToLower(strings.ReplaceAll(constraint.UpdateRule, " ", "")),
						})
					}
					// ondelete
					if constraint.DeleteRule != "" && constraint.DeleteRule != NO_ACTION {
						m.Submodifiers = append(m.Submodifiers, Modifier{
							Name:  "ondelete",
							Value: strings.ToLower(strings.ReplaceAll(constraint.DeleteRule, " ", "")),
						})
					}
				default:
					continue
				}
				// deferred deferrable
				if constraint.IsDeferrable {
					if constraint.IsInitiallyDeferred {
						m.Submodifiers = append(m.Submodifiers, Modifier{Name: "deferred"})
					} else {
						m.Submodifiers = append(m.Submodifiers, Modifier{Name: "deferrable"})
					}
				}
				constraintModifierList = append(constraintModifierList, m)
			}
			for _, index := range table.Indexes {
				if index.Ignore || !isSimpleIndex(index) {
					continue
				}
				columnNames := strings.Join(index.Columns, ",")
				m := &Modifier{Name: "index", Value: columnNames}
				indexModifiers[columnNames] = m
				// unique
				if index.IsUnique {
					m.Submodifiers = append(m.Submodifiers, Modifier{Name: "unique"})
				}
				// using
				if index.IndexType != "" && !strings.EqualFold(index.IndexType, "BTREE") {
					m.Submodifiers = append(m.Submodifiers, Modifier{Name: "using", RawValue: index.IndexType})
				}
				// foreignkey.index
				if foreignkeyModifier := foreignkeyModifiers[columnNames]; foreignkeyModifier != nil {
					addedModifier[m] = true
					foreignkeyModifier.Submodifiers = append(foreignkeyModifier.Submodifiers, *m)
					foreignkeyModifier.Submodifiers[len(foreignkeyModifier.Submodifiers)-1].Value = ""
				}
				indexModifierList = append(indexModifierList, m)
			}
			for i, column := range table.Columns {
				if column.Ignore {
					continue
				}
				structField := StructField{
					Name: strings.ToUpper(strings.ReplaceAll(column.ColumnName, " ", "_")),
					Type: getFieldType(catalog.Dialect, &table.Columns[i]),
				}
				if needsQuoting(column.ColumnName) {
					structField.NameTag = column.ColumnName
				}
				var defaultColumnType string
				switch structField.Type {
				case "sq.BinaryField":
					switch catalog.Dialect {
					case sq.DialectSQLite:
						defaultColumnType = "BLOB"
					case sq.DialectPostgres:
						defaultColumnType = "BYTEA"
					case sq.DialectMySQL:
						defaultColumnType = "MEDIUMBLOB"
					case sq.DialectSQLServer:
						defaultColumnType = "VARBINARY(MAX)"
					default:
						defaultColumnType = "BINARY"
					}
				case "sq.BooleanField":
					switch catalog.Dialect {
					case sq.DialectSQLServer:
						defaultColumnType = "BIT"
					default:
						defaultColumnType = "BOOLEAN"
					}
				case "sq.EnumField":
					switch catalog.Dialect {
					case sq.DialectSQLite, sq.DialectPostgres:
						defaultColumnType = "TEXT"
					case sq.DialectSQLServer:
						defaultColumnType = "NVARCHAR(255)"
					default:
						defaultColumnType = "VARCHAR(255)"
					}
				case "sq.JSONField":
					switch catalog.Dialect {
					case sq.DialectSQLite, sq.DialectMySQL:
						defaultColumnType = "JSON"
					case sq.DialectPostgres:
						defaultColumnType = "JSONB"
					case sq.DialectSQLServer:
						defaultColumnType = "NVARCHAR(MAX)"
					default:
						defaultColumnType = "VARCHAR(255)"
					}
				case "sq.NumberField":
					defaultColumnType = "INT"
				case "sq.StringField":
					switch catalog.Dialect {
					case sq.DialectSQLite, sq.DialectPostgres:
						defaultColumnType = "TEXT"
					case sq.DialectSQLServer:
						defaultColumnType = "NVARCHAR(255)"
					default:
						defaultColumnType = "VARCHAR(255)"
					}
				case "sq.TimeField":
					switch catalog.Dialect {
					case sq.DialectPostgres:
						defaultColumnType = "TIMESTAMPTZ"
					case sq.DialectSQLServer:
						defaultColumnType = "DATETIMEOFFSET"
					default:
						defaultColumnType = "TIMESTAMP"
					}
				case "sq.UUIDField":
					switch catalog.Dialect {
					case sq.DialectSQLite, sq.DialectPostgres:
						defaultColumnType = "UUID"
					default:
						defaultColumnType = "BINARY(16)"
					}
				}
				// type
				if column.DomainName != "" {
					structField.Modifiers = append(structField.Modifiers, Modifier{Name: "type", RawValue: column.DomainName})
				} else if column.ColumnType != "" && column.ColumnType != defaultColumnType {
					isSQLiteRowid := catalog.Dialect == sq.DialectSQLite &&
						primarykeyModifier != nil &&
						primarykeyModifier.Value == column.ColumnName &&
						strings.EqualFold(column.ColumnType, "INTEGER")
					if !isSQLiteRowid {
						structField.Modifiers = append(structField.Modifiers, Modifier{Name: "type", RawValue: column.ColumnType})
					}
				}
				// notnull
				if column.IsNotNull {
					structField.Modifiers = append(structField.Modifiers, Modifier{Name: "notnull"})
				}
				// primarykey
				if primarykeyModifier != nil && primarykeyModifier.Value == column.ColumnName {
					addedModifier[primarykeyModifier] = true
					structField.Modifiers = append(structField.Modifiers, *primarykeyModifier)
					structField.Modifiers[len(structField.Modifiers)-1].Value = ""
				}
				// unique
				if uniqueModifier := uniqueModifiers[column.ColumnName]; uniqueModifier != nil {
					addedModifier[uniqueModifier] = true
					structField.Modifiers = append(structField.Modifiers, *uniqueModifier)
					structField.Modifiers[len(structField.Modifiers)-1].Value = ""
				}
				// references
				if foreignkeyModifier := foreignkeyModifiers[column.ColumnName]; foreignkeyModifier != nil {
					addedModifier[foreignkeyModifier] = true
					structField.Modifiers = append(structField.Modifiers, *foreignkeyModifier)
					i := len(structField.Modifiers) - 1
					structField.Modifiers[i].Name = "references"
					structField.Modifiers[i].Value = structField.Modifiers[i].Submodifiers[0].RawValue
					structField.Modifiers[i].Submodifiers = structField.Modifiers[i].Submodifiers[1:]
				}
				// autoincrement
				if column.IsAutoincrement {
					switch catalog.Dialect {
					case sq.DialectSQLite:
						structField.Modifiers = append(structField.Modifiers, Modifier{Name: "autoincrement"})
					case sq.DialectMySQL:
						structField.Modifiers = append(structField.Modifiers, Modifier{Name: "auto_increment"})
					}
				}
				// identity
				if column.ColumnIdentity != "" {
					switch catalog.Dialect {
					case sq.DialectPostgres:
						if column.ColumnIdentity == DEFAULT_IDENTITY {
							structField.Modifiers = append(structField.Modifiers, Modifier{Name: "identity"})
						} else if column.ColumnIdentity == ALWAYS_IDENTITY {
							structField.Modifiers = append(structField.Modifiers, Modifier{Name: "alwaysidentity"})
						}
					case sq.DialectSQLServer:
						if column.ColumnIdentity == IDENTITY {
							structField.Modifiers = append(structField.Modifiers, Modifier{Name: "identity"})
						}
					}
				}
				// default
				if column.ColumnDefault != "" && !strings.ContainsRune(column.ColumnDefault, '`') {
					structField.Modifiers = append(structField.Modifiers, Modifier{Name: "default", RawValue: unwrapBrackets(column.ColumnDefault)})
				}
				// onupdatecurrenttimestamp
				if column.OnUpdateCurrentTimestamp {
					structField.Modifiers = append(structField.Modifiers, Modifier{Name: "onupdatecurrenttimestamp"})
				}
				// collate
				if column.CollationName != "" && column.CollationName != catalog.DefaultCollation {
					structField.Modifiers = append(structField.Modifiers, Modifier{Name: "collate", RawValue: column.CollationName})
				}
				// index
				if indexModifier := indexModifiers[column.ColumnName]; indexModifier != nil {
					if !addedModifier[indexModifier] {
						addedModifier[indexModifier] = true
						structField.Modifiers = append(structField.Modifiers, *indexModifier)
						structField.Modifiers[len(structField.Modifiers)-1].Value = ""
					}
				}
				// generated
				if column.IsGenerated || column.GeneratedExpr != "" {
					structField.Modifiers = append(structField.Modifiers, Modifier{Name: "generated"})
				}
				tableStruct.Fields = append(tableStruct.Fields, structField)
			}
			if primarykeyModifier != nil && !addedModifier[primarykeyModifier] {
				addedModifier[primarykeyModifier] = true
				tableStruct.Fields[0].Modifiers = Modifiers{*primarykeyModifier}
			}
			for _, constraintModifier := range constraintModifierList {
				if addedModifier[constraintModifier] {
					continue
				}
				tableStruct.Fields = append(tableStruct.Fields, StructField{
					Name:      "_",
					Type:      "struct{}",
					Modifiers: Modifiers{*constraintModifier},
				})
			}
			for _, indexModifier := range indexModifierList {
				if addedModifier[indexModifier] {
					continue
				}
				tableStruct.Fields = append(tableStruct.Fields, StructField{
					Name:      "_",
					Type:      "struct{}",
					Modifiers: Modifiers{*indexModifier},
				})
			}
			*s = append(*s, tableStruct)
		}
	}
	return nil
}

// MarshalText converts the TableStructs into Go source code.
func (s *TableStructs) MarshalText() (text []byte, err error) {
	buf := bufpool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufpool.Put(buf)
	for _, tableStruct := range *s {
		hasColumn := false
		for i := len(tableStruct.Fields) - 1; i >= 0; i-- {
			if tableStruct.Fields[i].Name != "" && tableStruct.Fields[i].Name != "_" {
				hasColumn = true
				break
			}
		}
		if !hasColumn {
			continue
		}
		if buf.Len() > 0 {
			buf.WriteString("\n")
		}
		buf.WriteString("type " + tableStruct.Name + " struct {")
		for _, structField := range tableStruct.Fields {
			if structField.Name != "" {
				buf.WriteString("\n\t" + structField.Name + " " + structField.Type)
			} else {
				buf.WriteString("\n\t" + structField.Type)
			}
			ddlTag := structField.Modifiers.String()
			if structField.NameTag == "" && ddlTag == "" {
				continue
			}
			buf.WriteString(" `")
			written := false
			if structField.NameTag != "" {
				if written {
					buf.WriteString(" ")
				}
				written = true
				buf.WriteString(`sq:` + strconv.Quote(structField.NameTag))
			}
			if ddlTag != "" {
				if written {
					buf.WriteString(" ")
				}
				written = true
				buf.WriteString(`ddl:` + strconv.Quote(ddlTag))
			}
			buf.WriteString("`")
		}
		buf.WriteString("\n}\n")
	}
	b := make([]byte, buf.Len())
	copy(b, buf.Bytes())
	return b, nil
}

func getFieldType(dialect string, column *Column) (fieldType string) {
	if column.IsEnum {
		return "sq.EnumField"
	}
	if strings.HasSuffix(column.ColumnType, "[]") {
		return "sq.ArrayField"
	}
	normalizedType, arg1, _ := normalizeColumnType(dialect, column.ColumnType)
	if normalizedType == "TINYINT" && arg1 == "1" {
		return "sq.BooleanField"
	}
	if normalizedType == "BINARY" && arg1 == "16" {
		return "sq.UUIDField"
	}
	switch normalizedType {
	case "BYTEA", "BINARY", "VARBINARY", "TINYBLOB", "BLOB", "MEDIUMBLOB", "LONGBLOB", "VARBIT":
		return "sq.BinaryField"
	case "BOOLEAN", "BIT":
		return "sq.BooleanField"
	case "JSON", "JSONB":
		return "sq.JSONField"
	case "TINYINT", "SMALLINT", "MEDIUMINT", "INT", "INTEGER", "BIGINT", "NUMERIC", "FLOAT", "REAL", "DOUBLE PRECISION":
		return "sq.NumberField"
	case "TINYTEXT", "TEXT", "MEDIUMTEXT", "LONGTEXT", "CHAR", "VARCHAR", "NVARCHAR":
		return "sq.StringField"
	case "DATE", "TIME", "TIMETZ", "DATETIME", "DATETIME2", "SMALLDATETIME", "DATETIMEOFFSET", "TIMESTAMP", "TIMESTAMPTZ":
		return "sq.TimeField"
	case "UUID", "UNIQUEIDENTIFIER":
		return "sq.UUIDField"
	}
	return "sq.AnyField"
}

func needsQuoting(identifier string) bool {
	for i, char := range identifier {
		if i == 0 && (char >= '0' && char <= '9') {
			return true
		}
		if char == '_' || (char >= '0' && char <= '9') || (char >= 'a' && char <= 'z') {
			continue
		}
		return true
	}
	return false
}

// We only consider simple indexes for table structs because complex indexes
// involving predicates or included columns are harder to diff.
func isSimpleIndex(index Index) bool {
	if len(index.Columns) == 0 {
		return false
	}
	if len(index.IncludeColumns) > 0 {
		return false
	}
	if index.Predicate != "" {
		return false
	}
	for _, isDescending := range index.Descending {
		if isDescending {
			return false
		}
	}
	for _, opclass := range index.Opclasses {
		if strings.Count(opclass, "_") > 1 {
			return false
		}
	}
	for _, column := range index.Columns {
		if strings.HasPrefix(column, "(") {
			return false
		}
	}
	upperSQL := strings.ToUpper(index.SQL)
	if strings.Contains(upperSQL, " WHERE ") {
		return false
	}
	if strings.Contains(upperSQL, " DESC") {
		return false
	}
	if strings.Contains(upperSQL, " INCLUDE ") {
		return false
	}
	return true
}
