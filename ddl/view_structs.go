package ddl

import (
	"bytes"
	"database/sql"
	"strconv"
	"strings"
)

type ViewStructs []ViewStruct

type ViewStruct struct {
	Name   string
	Fields []StructField
}

// NewViewStructs introspects a database connection and returns a slice of
// ViewStructs, each ViewStruct corresponding to a view in the database. You
// may narrow down the list of views by filling in the Schemas, ExcludeSchemas,
// Views and ExcludeViews fields of the Filter struct. The Filter.ObjectTypes
// field will always be set to []string{"VIEWS"}.
func NewViewStructs(dialect string, db *sql.DB, filter Filter) (ViewStructs, error) {
	var viewStructs ViewStructs
	var catalog Catalog
	dbi := &DatabaseIntrospector{
		Filter:  filter,
		Dialect: dialect,
		DB:      db,
	}
	dbi.ObjectTypes = []string{"VIEWS"}
	err := dbi.WriteCatalog(&catalog)
	if err != nil {
		return nil, err
	}
	err = viewStructs.ReadCatalog(&catalog)
	if err != nil {
		return nil, err
	}
	return viewStructs, nil
}

// ReadCatalog reads from a catalog and populates the ViewStructs accordingly.
func (s *ViewStructs) ReadCatalog(catalog *Catalog) error {
	for _, schema := range catalog.Schemas {
		for _, view := range schema.Views {
			viewStruct := ViewStruct{
				Name:   strings.ToUpper(strings.ReplaceAll(view.ViewName, " ", "_")),
				Fields: make([]StructField, 0, len(view.Columns)+1),
			}
			viewStruct.Fields = append(viewStruct.Fields, StructField{Type: "sq.ViewStruct"})
			if (view.ViewSchema != "" && view.ViewSchema != catalog.CurrentSchema) || needsQuoting(view.ViewName) {
				if view.ViewSchema != "" {
					viewStruct.Fields[0].NameTag = view.ViewSchema + "." + view.ViewName
				} else {
					viewStruct.Fields[0].NameTag = view.ViewName
				}
			}
			isEnum := make(map[string]bool)
			for _, column := range view.EnumColumns {
				isEnum[column] = true
			}
			for i, column := range view.Columns {
				columnType := ""
				if i < len(view.ColumnTypes) {
					columnType = view.ColumnTypes[i]
				}
				structField := StructField{
					Name: strings.ToUpper(strings.ReplaceAll(column, " ", "_")),
					Type: getFieldType(catalog.Dialect, &Column{
						ColumnType: columnType,
						IsEnum:     isEnum[column],
					}),
				}
				viewStruct.Fields = append(viewStruct.Fields, structField)
			}
			*s = append(*s, viewStruct)
		}
	}
	return nil
}

// MarshalText converts the ViewStructs into Go source code.
func (s *ViewStructs) MarshalText() (text []byte, err error) {
	buf := bufpool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufpool.Put(buf)
	for _, viewStruct := range *s {
		if buf.Len() > 0 {
			buf.WriteString("\n")
		}
		buf.WriteString("type " + viewStruct.Name + " struct {")
		for _, structField := range viewStruct.Fields {
			if structField.Name != "" {
				buf.WriteString("\n\t" + structField.Name + " " + structField.Type)
			} else {
				buf.WriteString("\n\t" + structField.Type)
			}
			if structField.NameTag == "" {
				continue
			}
			buf.WriteString(" `")
			if structField.NameTag != "" {
				buf.WriteString(`sq:` + strconv.Quote(structField.NameTag))
			}
			buf.WriteString("`")
		}
		buf.WriteString("\n}\n")
	}
	b := make([]byte, buf.Len())
	copy(b, buf.Bytes())
	return b, nil
}
