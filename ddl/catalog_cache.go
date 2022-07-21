package ddl

// CatalogCache is used for querying and modifying a Catalog's nested
// objects without the need to loop over all the tables, columns,
// constraints etc just to find what you need. It does so by maintaining an
// internal cache of where everything is kept. All changes to a Catalog should
// be made through the CatalogCache it in order to keep the internal cache
// up-to-date.
type CatalogCache struct {
	dialect     string
	extensions  map[string]int
	schemas     map[string]int
	enums       map[[2]string]int
	domains     map[[2]string]int
	routines    map[[3]string]int
	views       map[[2]string]int
	tables      map[[2]string]int
	columns     map[[3]string]int
	constraints map[[3]string]int
	indexes     map[[3]string]int
	triggers    map[[3]string]int
	pkeys       map[[2]string]int
	fkeys       map[[2]string][]int
}

// NewCatalogCache creates a new CatalogCache from a given Catalog.
func NewCatalogCache(catalog *Catalog) *CatalogCache {
	cache := &CatalogCache{
		dialect:     catalog.Dialect,
		extensions:  make(map[string]int),
		schemas:     make(map[string]int),
		enums:       make(map[[2]string]int),
		domains:     make(map[[2]string]int),
		routines:    make(map[[3]string]int),
		views:       make(map[[2]string]int),
		tables:      make(map[[2]string]int),
		columns:     make(map[[3]string]int),
		constraints: make(map[[3]string]int),
		indexes:     make(map[[3]string]int),
		triggers:    make(map[[3]string]int),
		pkeys:       make(map[[2]string]int),
		fkeys:       make(map[[2]string][]int),
	}
	for i, extension := range catalog.Extensions {
		cache.extensions[extension] = i
	}
	for i, schema := range catalog.Schemas {
		cache.schemas[schema.SchemaName] = i
		for j, enum := range schema.Enums {
			enumID := [2]string{schema.SchemaName, enum.EnumName}
			cache.enums[enumID] = j
		}
		for j, domain := range schema.Domains {
			domainID := [2]string{schema.SchemaName, domain.DomainName}
			cache.domains[domainID] = j
		}
		for j, routine := range schema.Routines {
			identityArguments := ""
			if cache.dialect == "postgres" {
				identityArguments = routine.IdentityArguments
			}
			routineID := [3]string{schema.SchemaName, routine.RoutineName, identityArguments}
			cache.routines[routineID] = j
		}
		for j, view := range schema.Views {
			viewID := [2]string{schema.SchemaName, view.ViewName}
			cache.views[viewID] = j
			if view.IsMaterialized {
				for k, index := range view.Indexes {
					indexID := [3]string{schema.SchemaName, view.ViewName, index.IndexName}
					cache.indexes[indexID] = k
				}
			} else {
				for k, trigger := range view.Triggers {
					triggerID := [3]string{schema.SchemaName, view.ViewName, trigger.TriggerName}
					cache.triggers[triggerID] = k
				}
			}
		}
		for j, table := range schema.Tables {
			tableID := [2]string{schema.SchemaName, table.TableName}
			cache.tables[tableID] = j
			for k, column := range table.Columns {
				columnID := [3]string{schema.SchemaName, table.TableName, column.ColumnName}
				cache.columns[columnID] = k
			}
			for k, constraint := range table.Constraints {
				switch constraint.ConstraintType {
				case PRIMARY_KEY:
					cache.pkeys[tableID] = k
				case FOREIGN_KEY:
					cache.fkeys[tableID] = append(cache.fkeys[tableID], k)
				}
				if constraint.ConstraintName == "" { // SQLite constraints will have no names, skip
					continue
				}
				constraintID := [3]string{schema.SchemaName, table.TableName, constraint.ConstraintName}
				cache.constraints[constraintID] = k
			}
			for k, index := range table.Indexes {
				indexID := [3]string{schema.SchemaName, table.TableName, index.IndexName}
				cache.indexes[indexID] = k
			}
			for k, trigger := range table.Triggers {
				triggerID := [3]string{schema.SchemaName, table.TableName, trigger.TriggerName}
				cache.triggers[triggerID] = k
			}
		}
	}
	return cache
}

// GetSchema gets a Schema with the given schemaName from the Catalog, or
// returns nil if it doesn't exist. If a nil catalog is passed in, GetSchema
// returns nil.
//
// The returning Schema pointer is valid as long as no new Schema is added to
// the Catalog; if a new Schema is added, the pointer may now be pointing at a
// stale Schema. Call GetSchema again in order to get the new pointer.
func (c *CatalogCache) GetSchema(catalog *Catalog, schemaName string) *Schema {
	if catalog == nil {
		return nil
	}
	i, ok := c.schemas[schemaName]
	if ok && !catalog.Schemas[i].Ignore {
		return &catalog.Schemas[i]
	}
	return nil
}

// GetOrCreateSchema gets a Schema with the given schemaName from the Catalog,
// or creates it if it doesn't exist.
//
// The returning Schema pointer is valid as long as no new Schema is added to
// the Catalog; if a new Schema is added, the pointer may now be pointing at a
// stale Schema. Call GetSchema again in order to get the new pointer.
func (c *CatalogCache) GetOrCreateSchema(catalog *Catalog, schemaName string) *Schema {
	i, ok := c.schemas[schemaName]
	if ok && !catalog.Schemas[i].Ignore {
		return &catalog.Schemas[i]
	}
	catalog.Schemas = append(catalog.Schemas, Schema{
		SchemaName: schemaName,
	})
	i = len(catalog.Schemas) - 1
	c.schemas[schemaName] = i
	return &catalog.Schemas[i]
}

// AddOrUpdateSchema adds the given Schema to the Catalog, or updates it if
// it already exists.
func (c *CatalogCache) AddOrUpdateSchema(catalog *Catalog, schema Schema) {
	i, ok := c.schemas[schema.SchemaName]
	if ok && !catalog.Schemas[i].Ignore {
		catalog.Schemas[i] = schema
		return
	}
	catalog.Schemas = append(catalog.Schemas, schema)
	i = len(catalog.Schemas) - 1
	c.schemas[schema.SchemaName] = i
}

// GetEnum gets a Enum with the given enumName from the Schema, or returns nil
// if it doesn't exist. If a nil schema is passed in, GetEnum returns nil.
//
// The returning Enum pointer is valid as long as no new Enum is added to
// the Schema; if a new Enum is added, the pointer may now be pointing at a
// stale Enum. Call GetEnum again in order to get the new pointer.
func (c *CatalogCache) GetEnum(schema *Schema, enumName string) *Enum {
	if schema == nil {
		return nil
	}
	i, ok := c.enums[[2]string{schema.SchemaName, enumName}]
	if ok && !schema.Enums[i].Ignore {
		return &schema.Enums[i]
	}
	return nil
}

// GetOrCreateEnum gets a Enum with the given enumName from the Schema,
// or creates it if it doesn't exist.
//
// The returning Enum pointer is valid as long as no new Enum is added to
// the Schema; if a new Enum is added, the pointer may now be pointing at a
// stale Enum. Call GetEnum again in order to get the new pointer.
func (c *CatalogCache) GetOrCreateEnum(schema *Schema, enumName string) *Enum {
	i, ok := c.enums[[2]string{schema.SchemaName, enumName}]
	if ok && !schema.Enums[i].Ignore {
		return &schema.Enums[i]
	}
	schema.Enums = append(schema.Enums, Enum{
		EnumSchema: schema.SchemaName,
		EnumName:   enumName,
	})
	i = len(schema.Enums) - 1
	c.enums[[2]string{schema.SchemaName, enumName}] = i
	return &schema.Enums[i]
}

// AddOrUpdateEnum adds the given Enum to the Schema, or updates it if
// it already exists.
func (c *CatalogCache) AddOrUpdateEnum(schema *Schema, enum Enum) {
	i, ok := c.enums[[2]string{schema.SchemaName, enum.EnumName}]
	if ok && !schema.Enums[i].Ignore {
		schema.Enums[i] = enum
		return
	}
	schema.Enums = append(schema.Enums, enum)
	i = len(schema.Enums) - 1
	c.enums[[2]string{schema.SchemaName, enum.EnumName}] = i
}

// GetDomain gets a Domain with the given domainName from the Schema, or
// returns nil if it doesn't exist. If a nil schema is passed in, GetDomain
// returns nil.
//
// The returning Domain pointer is valid as long as no new Domain is added to
// the Schema; if a new Domain is added, the pointer may now be pointing at a
// stale Domain. Call GetDomain again in order to get the new pointer.
func (c *CatalogCache) GetDomain(schema *Schema, domainName string) *Domain {
	if schema == nil {
		return nil
	}
	i, ok := c.domains[[2]string{schema.SchemaName, domainName}]
	if ok && !schema.Domains[i].Ignore {
		return &schema.Domains[i]
	}
	return nil
}

// GetOrCreateDomain gets a Domain with the given domainName from the Schema,
// or creates it if it doesn't exist.
//
// The returning Domain pointer is valid as long as no new Domain is added to
// the Schema; if a new Domain is added, the pointer may now be pointing at a
// stale Domain. Call GetDomain again in order to get the new pointer.
func (c *CatalogCache) GetOrCreateDomain(schema *Schema, domainName string) *Domain {
	i, ok := c.domains[[2]string{schema.SchemaName, domainName}]
	if ok && !schema.Domains[i].Ignore {
		return &schema.Domains[i]
	}
	schema.Domains = append(schema.Domains, Domain{
		DomainSchema: schema.SchemaName,
		DomainName:   domainName,
	})
	i = len(schema.Domains) - 1
	c.domains[[2]string{schema.SchemaName, domainName}] = i
	return &schema.Domains[i]
}

// AddOrUpdateDomain adds the given Domain to the Schema, or updates it if
// it already exists.
func (c *CatalogCache) AddOrUpdateDomain(schema *Schema, domain Domain) {
	i, ok := c.domains[[2]string{schema.SchemaName, domain.DomainName}]
	if ok && !schema.Domains[i].Ignore {
		schema.Domains[i] = domain
		return
	}
	schema.Domains = append(schema.Domains, domain)
	i = len(schema.Domains) - 1
	c.domains[[2]string{schema.SchemaName, domain.DomainName}] = i
}

// GetRoutine gets a Routine with the given routineName (and identityArguments)
// from the Schema, or returns nil if it doesn't exist. If a nil schema is
// passed in, GetRoutine returns nil. The identityArguments string only applies
// to Postgres because functions may be overloaded and differ only by their
// identity arguments. For other database dialects, the identityArguments
// should be an empty string.
//
// The returning Routine pointer is valid as long as no new Routine is added to
// the Schema; if a new Routine is added, the pointer may now be pointing at a
// stale Routine. Call GetRoutine again in order to get the new pointer.
func (c *CatalogCache) GetRoutine(schema *Schema, routineName, identityArguments string) *Routine {
	if schema == nil {
		return nil
	}
	if c.dialect != "postgres" {
		identityArguments = ""
	}
	i, ok := c.routines[[3]string{schema.SchemaName, routineName, identityArguments}]
	if ok && !schema.Routines[i].Ignore {
		return &schema.Routines[i]
	}
	return nil
}

// GetOrCreateRoutine gets a Routine with the given routineName and
// identityArguments from the Schema, or creates it if it doesn't exist. The
// identityArguments string only applies to Postgres because functions may be
// overloaded and differ only by their identity arguments. For other database
// dialects, the identityArguments should be an empty string.
//
// The returning Routine pointer is valid as long as no new Routine is added to
// the Schema; if a new Routine is added, the pointer may now be pointing at a
// stale Routine. Call GetRoutine again in order to get the new pointer.
func (c *CatalogCache) GetOrCreateRoutine(schema *Schema, routineName, identityArguments string) *Routine {
	i, ok := c.routines[[3]string{schema.SchemaName, routineName, identityArguments}]
	if ok && !schema.Routines[i].Ignore {
		return &schema.Routines[i]
	}
	schema.Routines = append(schema.Routines, Routine{
		RoutineSchema:     schema.SchemaName,
		RoutineName:       routineName,
		IdentityArguments: identityArguments,
	})
	i = len(schema.Routines) - 1
	c.routines[[3]string{schema.SchemaName, routineName, identityArguments}] = i
	return &schema.Routines[i]
}

// AddOrUpdateRoutine adds the given Routine to the Schema, or updates it if
// it already exists.
func (c *CatalogCache) AddOrUpdateRoutine(schema *Schema, routine Routine) {
	i, ok := c.routines[[3]string{schema.SchemaName, routine.RoutineName, routine.IdentityArguments}]
	if ok && !schema.Routines[i].Ignore {
		schema.Routines[i] = routine
		return
	}
	schema.Routines = append(schema.Routines, routine)
	i = len(schema.Routines) - 1
	c.routines[[3]string{schema.SchemaName, routine.RoutineName, routine.IdentityArguments}] = i
}

// GetView gets a View with the given viewName from the Schema, or returns nil
// if it doesn't exist. If a nil schema is passed in, GetView returns nil.
//
// The returning View pointer is valid as long as no new View is added to
// the Schema; if a new View is added, the pointer may now be pointing at a
// stale View. Call GetView again in order to get the new pointer.
func (c *CatalogCache) GetView(schema *Schema, viewName string) *View {
	if schema == nil {
		return nil
	}
	i, ok := c.views[[2]string{schema.SchemaName, viewName}]
	if ok && !schema.Views[i].Ignore {
		return &schema.Views[i]
	}
	return nil
}

// GetOrCreateView gets a View with the given viewName from the Schema,
// or creates it if it doesn't exist.
//
// The returning View pointer is valid as long as no new View is added to
// the Schema; if a new View is added, the pointer may now be pointing at a
// stale View. Call GetView again in order to get the new pointer.
func (c *CatalogCache) GetOrCreateView(schema *Schema, viewName string) *View {
	i, ok := c.views[[2]string{schema.SchemaName, viewName}]
	if ok && !schema.Views[i].Ignore {
		return &schema.Views[i]
	}
	schema.Views = append(schema.Views, View{
		ViewSchema: schema.SchemaName,
		ViewName:   viewName,
	})
	i = len(schema.Views) - 1
	c.views[[2]string{schema.SchemaName, viewName}] = i
	return &schema.Views[i]
}

// AddOrUpdateView adds the given View to the Schema, or updates it if
// it already exists.
func (c *CatalogCache) AddOrUpdateView(schema *Schema, view View) {
	i, ok := c.views[[2]string{schema.SchemaName, view.ViewName}]
	if ok && !schema.Views[i].Ignore {
		schema.Views[i] = view
		return
	}
	schema.Views = append(schema.Views, view)
	i = len(schema.Views) - 1
	c.views[[2]string{schema.SchemaName, view.ViewName}] = i
}

// GetTable gets a Table with the given tableName from the Schema, or returns
// nil if it doesn't exist. If a nil schema is passed in, GetTable returns nil.
//
// The returning Table pointer is valid as long as no new Table is added to
// the Schema; if a new Table is added, the pointer may now be pointing at a
// stale Table. Call GetTable again in order to get the new pointer.
func (c *CatalogCache) GetTable(schema *Schema, tableName string) *Table {
	if schema == nil {
		return nil
	}
	i, ok := c.tables[[2]string{schema.SchemaName, tableName}]
	if ok && !schema.Tables[i].Ignore {
		return &schema.Tables[i]
	}
	return nil
}

// GetOrCreateTable gets a Table with the given tableName from the Schema,
// or creates it if it doesn't exist.
//
// The returning Table pointer is valid as long as no new Table is added to
// the Schema; if a new Table is added, the pointer may now be pointing at a
// stale Table. Call GetTable again in order to get the new pointer.
func (c *CatalogCache) GetOrCreateTable(schema *Schema, tableName string) *Table {
	i, ok := c.tables[[2]string{schema.SchemaName, tableName}]
	if ok && !schema.Tables[i].Ignore {
		return &schema.Tables[i]
	}
	schema.Tables = append(schema.Tables, Table{
		TableSchema: schema.SchemaName,
		TableName:   tableName,
	})
	i = len(schema.Tables) - 1
	c.tables[[2]string{schema.SchemaName, tableName}] = i
	return &schema.Tables[i]
}

// AddOrUpdateTable adds the given Table to the Schema, or updates it if
// it already exists.
func (c *CatalogCache) AddOrUpdateTable(schema *Schema, table Table) {
	i, ok := c.tables[[2]string{schema.SchemaName, table.TableName}]
	if ok && !schema.Tables[i].Ignore {
		schema.Tables[i] = table
		return
	}
	schema.Tables = append(schema.Tables, table)
	i = len(schema.Tables) - 1
	c.tables[[2]string{schema.SchemaName, table.TableName}] = i
}

// GetColumn gets a Column with the given columnName from the Table, or returns
// nil if it doesn't exist. If a nil table is passed in, GetColumn returns nil.
//
// The returning Column pointer is valid as long as no new Column is added to
// the Table; if a new Column is added, the pointer may now be pointing at a
// stale Column. Call GetColumn again in order to get the new pointer.
func (c *CatalogCache) GetColumn(table *Table, columnName string) *Column {
	if table == nil {
		return nil
	}
	i, ok := c.columns[[3]string{table.TableSchema, table.TableName, columnName}]
	if ok && !table.Columns[i].Ignore {
		return &table.Columns[i]
	}
	return nil
}

// GetOrCreateColumn gets a Column with the given columnName from the Table, or
// creates it if it doesn't exist.
//
// The returning Column pointer is valid as long as no new Column is added to
// the Table; if a new Column is added, the pointer may now be pointing at a
// stale Column. Call GetColumn again in order to get the new pointer.
func (c *CatalogCache) GetOrCreateColumn(table *Table, columnName, columnType string) *Column {
	i, ok := c.columns[[3]string{table.TableSchema, table.TableName, columnName}]
	if ok && !table.Columns[i].Ignore {
		return &table.Columns[i]
	}
	table.Columns = append(table.Columns, Column{
		ColumnName: columnName,
		ColumnType: columnType,
	})
	i = len(table.Columns) - 1
	c.columns[[3]string{table.TableSchema, table.TableName, columnName}] = i
	return &table.Columns[i]
}

// AddOrUpdateColumn adds the given Column to the Table, or updates it if
// it already exists.
func (c *CatalogCache) AddOrUpdateColumn(table *Table, column Column) {
	i, ok := c.columns[[3]string{table.TableSchema, table.TableName, column.ColumnName}]
	if ok && !table.Columns[i].Ignore {
		table.Columns[i] = column
		return
	}
	table.Columns = append(table.Columns, column)
	i = len(table.Columns) - 1
	c.columns[[3]string{table.TableSchema, table.TableName, column.ColumnName}] = i
}

// GetConstraint gets a Constraint with the given constraintName from the
// Table, or returns nil if it doesn't exist. If a nil table is passed in,
// GetConstraint returns nil.
//
// The returning Constraint pointer is valid as long as no new Constraint is added to
// the Table; if a new Constraint is added, the pointer may now be pointing at a
// stale Constraint. Call GetConstraint again in order to get the new pointer.
func (c *CatalogCache) GetConstraint(table *Table, constraintName string) *Constraint {
	if table == nil {
		return nil
	}
	i, ok := c.constraints[[3]string{table.TableSchema, table.TableName, constraintName}]
	if ok && !table.Constraints[i].Ignore {
		return &table.Constraints[i]
	}
	return nil
}

// GetOrCreateConstraint gets a Constraint with the given constraintName from
// the Table, or creates it if it doesn't exist.
//
// The returning Constraint pointer is valid as long as no new Constraint is added to
// the Table; if a new Constraint is added, the pointer may now be pointing at a
// stale Constraint. Call GetConstraint again in order to get the new pointer.
func (c *CatalogCache) GetOrCreateConstraint(table *Table, constraintName, constraintType string, columnNames []string) *Constraint {
	i, ok := c.constraints[[3]string{table.TableSchema, table.TableName, constraintName}]
	if ok && !table.Constraints[i].Ignore {
		return &table.Constraints[i]
	}
	table.Constraints = append(table.Constraints, Constraint{
		ConstraintName: constraintName,
		ConstraintType: constraintType,
		Columns:        columnNames,
	})
	i = len(table.Constraints) - 1
	tableID := [2]string{table.TableSchema, table.TableName}
	switch constraintType {
	case PRIMARY_KEY:
		c.pkeys[tableID] = i
	case FOREIGN_KEY:
		c.fkeys[tableID] = append(c.fkeys[tableID], i)
	}
	c.constraints[[3]string{table.TableSchema, table.TableName, constraintName}] = i
	return &table.Constraints[i]
}

// AddOrUpdateConstraint adds the given Constraint to the Table, or updates it if
// it already exists.
func (c *CatalogCache) AddOrUpdateConstraint(table *Table, constraint Constraint) {
	i, ok := c.constraints[[3]string{table.TableSchema, table.TableName, constraint.ConstraintName}]
	if ok && !table.Constraints[i].Ignore {
		table.Constraints[i] = constraint
		return
	}
	table.Constraints = append(table.Constraints, constraint)
	i = len(table.Constraints) - 1
	tableID := [2]string{table.TableSchema, table.TableName}
	switch constraint.ConstraintType {
	case PRIMARY_KEY:
		c.pkeys[tableID] = i
	case FOREIGN_KEY:
		c.fkeys[tableID] = append(c.fkeys[tableID], i)
	}
	if constraint.ConstraintName != "" { // SQLite constraints will have no name, skip
		c.constraints[[3]string{table.TableSchema, table.TableName, constraint.ConstraintName}] = i
	}
}

// GetIndex gets a Index with the given indexName from the Table, or returns
// nil if it doesn't exist. If a nil table is passed in, GetIndex returns nil.
//
// The returning Index pointer is valid as long as no new Index is added to
// the Table; if a new Index is added, the pointer may now be pointing at a
// stale Index. Call GetIndex again in order to get the new pointer.
func (c *CatalogCache) GetIndex(table *Table, indexName string) *Index {
	if table == nil {
		return nil
	}
	i, ok := c.indexes[[3]string{table.TableSchema, table.TableName, indexName}]
	if ok && !table.Indexes[i].Ignore {
		return &table.Indexes[i]
	}
	return nil
}

// GetOrCreateIndex gets a Index with the given indexName from the Table, or
// creates it if it doesn't exist.
//
// The returning Index pointer is valid as long as no new Index is added to
// the Table; if a new Index is added, the pointer may now be pointing at a
// stale Index. Call GetIndex again in order to get the new pointer.
func (c *CatalogCache) GetOrCreateIndex(table *Table, indexName string, columnNames []string) *Index {
	i, ok := c.indexes[[3]string{table.TableSchema, table.TableName, indexName}]
	if ok && !table.Indexes[i].Ignore {
		return &table.Indexes[i]
	}
	table.Indexes = append(table.Indexes, Index{
		IndexName: indexName,
		Columns:   columnNames,
	})
	i = len(table.Indexes) - 1
	c.indexes[[3]string{table.TableSchema, table.TableName, indexName}] = i
	return &table.Indexes[i]
}

// AddOrUpdateIndex adds the given Index to the Table, or updates it if
// it already exists.
func (c *CatalogCache) AddOrUpdateIndex(table *Table, index Index) {
	i, ok := c.indexes[[3]string{table.TableSchema, table.TableName, index.IndexName}]
	if ok && !table.Indexes[i].Ignore {
		table.Indexes[i] = index
		return
	}
	table.Indexes = append(table.Indexes, index)
	i = len(table.Indexes) - 1
	c.indexes[[3]string{table.TableSchema, table.TableName, index.IndexName}] = i
}

// GetTrigger gets a Trigger with the given triggerName from the Table, or
// returns nil if it doesn't exist. If a nil table is passed in, GetTrigger
// returns nil.
//
// The returning Trigger pointer is valid as long as no new Trigger is added to
// the Table; if a new Trigger is added, the pointer may now be pointing at a
// stale Trigger. Call GetTrigger again in order to get the new pointer.
func (c *CatalogCache) GetTrigger(table *Table, triggerName string) *Trigger {
	if table == nil {
		return nil
	}
	i, ok := c.triggers[[3]string{table.TableSchema, table.TableName, triggerName}]
	if ok && !table.Triggers[i].Ignore {
		return &table.Triggers[i]
	}
	return nil
}

// GetOrCreateTrigger gets a Trigger with the given triggerName from the Table,
// or creates it if it doesn't exist.
//
// The returning Trigger pointer is valid as long as no new Trigger is added to
// the Table; if a new Trigger is added, the pointer may now be pointing at a
// stale Trigger. Call GetTrigger again in order to get the new pointer.
func (c *CatalogCache) GetOrCreateTrigger(table *Table, triggerName string) *Trigger {
	i, ok := c.triggers[[3]string{table.TableSchema, table.TableName, triggerName}]
	if ok && !table.Triggers[i].Ignore {
		return &table.Triggers[i]
	}
	table.Triggers = append(table.Triggers, Trigger{
		TriggerName: triggerName,
	})
	i = len(table.Triggers) - 1
	c.triggers[[3]string{table.TableSchema, table.TableName, triggerName}] = i
	return &table.Triggers[i]
}

// AddOrUpdateTrigger adds the given Trigger to the Table, or updates it if
// it already exists.
func (c *CatalogCache) AddOrUpdateTrigger(table *Table, trigger Trigger) {
	i, ok := c.triggers[[3]string{table.TableSchema, table.TableName, trigger.TriggerName}]
	if ok && !table.Triggers[i].Ignore {
		table.Triggers[i] = trigger
		return
	}
	table.Triggers = append(table.Triggers, trigger)
	i = len(table.Triggers) - 1
	c.triggers[[3]string{table.TableSchema, table.TableName, trigger.TriggerName}] = i
}

// GetViewIndex gets a Index with the given indexName from the View, or
// returns nil if it doesn't exist.
//
// The returning Index pointer is valid as long as no new Index is added to
// the View; if a new Index is added, the pointer may now be pointing at a
// stale Index. Call GetViewIndex again in order to get the new pointer.
func (c *CatalogCache) GetViewIndex(view *View, indexName string) *Index {
	if view == nil {
		return nil
	}
	i, ok := c.indexes[[3]string{view.ViewSchema, view.ViewName, indexName}]
	if ok && !view.Indexes[i].Ignore {
		return &view.Indexes[i]
	}
	return nil
}

// GetOrCreateViewIndex gets a Index with the given indexName from the View, or
// creates it if it doesn't exist.
//
// The returning Index pointer is valid as long as no new Index is added to
// the View; if a new Index is added, the pointer may now be pointing at a
// stale Index. Call GetIndex again in order to get the new pointer.
func (c *CatalogCache) GetOrCreateViewIndex(view *View, indexName string, columnNames []string) *Index {
	i, ok := c.indexes[[3]string{view.ViewSchema, view.ViewName, indexName}]
	if ok && !view.Indexes[i].Ignore {
		return &view.Indexes[i]
	}
	view.Indexes = append(view.Indexes, Index{
		IndexName: indexName,
		Columns:   columnNames,
	})
	i = len(view.Indexes) - 1
	c.indexes[[3]string{view.ViewSchema, view.ViewName, indexName}] = i
	return &view.Indexes[i]
}

// AddOrUpdateViewIndex adds the given Index to the View, or updates it if
// it already exists.
func (c *CatalogCache) AddOrUpdateViewIndex(view *View, index Index) {
	i, ok := c.indexes[[3]string{view.ViewSchema, view.ViewName, index.IndexName}]
	if ok && !view.Indexes[i].Ignore {
		view.Indexes[i] = index
		return
	}
	view.Indexes = append(view.Indexes, index)
	i = len(view.Indexes) - 1
	c.indexes[[3]string{view.ViewSchema, view.ViewName, index.IndexName}] = i
}

// GetViewTrigger gets a Trigger with the given triggerName from the View, or
// returns nil if it doesn't exist.
//
// The returning Trigger pointer is valid as long as no new Trigger is added to
// the View; if a new Trigger is added, the pointer may now be pointing at a
// stale Trigger. Call GetViewTrigger again in order to get the new pointer.
func (c *CatalogCache) GetViewTrigger(view *View, triggerName string) *Trigger {
	if view == nil {
		return nil
	}
	i, ok := c.triggers[[3]string{view.ViewSchema, view.ViewName, triggerName}]
	if ok && !view.Triggers[i].Ignore {
		return &view.Triggers[i]
	}
	return nil
}

// GetOrCreateViewTrigger gets a Trigger with the given triggerName from the
// View, or creates it if it doesn't exist.
//
// The returning Trigger pointer is valid as long as no new Trigger is added to
// the View; if a new Trigger is added, the pointer may now be pointing at a
// stale Trigger. Call GetTrigger again in order to get the new pointer.
func (c *CatalogCache) GetOrCreateViewTrigger(view *View, triggerName string) *Trigger {
	i, ok := c.triggers[[3]string{view.ViewSchema, view.ViewName, triggerName}]
	if ok && !view.Triggers[i].Ignore {
		return &view.Triggers[i]
	}
	view.Triggers = append(view.Triggers, Trigger{
		TriggerName: triggerName,
	})
	i = len(view.Triggers) - 1
	c.triggers[[3]string{view.ViewSchema, view.ViewName, triggerName}] = i
	return &view.Triggers[i]
}

// AddOrUpdateViewTrigger adds the given Trigger to the View, or updates it if
// it already exists.
func (c *CatalogCache) AddOrUpdateViewTrigger(view *View, trigger Trigger) {
	i, ok := c.triggers[[3]string{view.ViewSchema, view.ViewName, trigger.TriggerName}]
	if ok && !view.Triggers[i].Ignore {
		view.Triggers[i] = trigger
		return
	}
	view.Triggers = append(view.Triggers, trigger)
	i = len(view.Triggers) - 1
	c.triggers[[3]string{view.ViewSchema, view.ViewName, trigger.TriggerName}] = i
}

// GetPrimaryKey gets the primary key of the Table, or returns nil if it
// doesn't exist.
func (c *CatalogCache) GetPrimaryKey(table *Table) *Constraint {
	if table == nil {
		return nil
	}
	i, ok := c.pkeys[[2]string{table.TableSchema, table.TableName}]
	if ok && !table.Constraints[i].Ignore {
		return &table.Constraints[i]
	}
	return nil
}

// GetForeignKeys gets the foreign keys of the table.
func (c *CatalogCache) GetForeignKeys(table *Table) []*Constraint {
	if table == nil {
		return nil
	}
	nums, ok := c.fkeys[[2]string{table.TableSchema, table.TableName}]
	if !ok {
		return nil
	}
	fkeys := make([]*Constraint, 0, len(nums))
	for _, num := range nums {
		fkey := &table.Constraints[num]
		if fkey.Ignore {
			continue
		}
		fkeys = append(fkeys, fkey)
	}
	return fkeys
}

// WriteCatalog populates the dest Catalog from the src Catalog, doing a deep
// copy in the process (nothing is shared between the src and dest Catalogs).
func (src *Catalog) WriteCatalog(dest *Catalog) error {
	cache := NewCatalogCache(dest)
	dest.Dialect = src.Dialect
	dest.VersionNums = cloneSlice(src.VersionNums)
	dest.CatalogName = src.CatalogName
	dest.CurrentSchema = src.CurrentSchema
	dest.DefaultCollation = src.DefaultCollation
	dest.Extensions = cloneSlice(src.Extensions)
	dest.DefaultCollationValid = src.DefaultCollationValid
	dest.ExtensionsValid = src.ExtensionsValid

	for _, srcSchema := range src.Schemas {
		destSchema := cache.GetOrCreateSchema(dest, srcSchema.SchemaName)
		destSchema.SchemaName = srcSchema.SchemaName
		destSchema.Comment = srcSchema.Comment
		destSchema.Ignore = srcSchema.Ignore
		destSchema.EnumsValid = srcSchema.EnumsValid
		destSchema.DomainsValid = srcSchema.DomainsValid
		destSchema.RoutinesValid = srcSchema.RoutinesValid
		destSchema.ViewsValid = srcSchema.ViewsValid

		for _, srcEnum := range srcSchema.Enums {
			destEnum := cache.GetOrCreateEnum(destSchema, srcEnum.EnumName)
			destEnum.EnumSchema = srcEnum.EnumSchema
			destEnum.EnumName = srcEnum.EnumName
			destEnum.EnumLabels = cloneSlice(srcEnum.EnumLabels)
			destEnum.Comment = srcEnum.Comment
			destEnum.Ignore = srcEnum.Ignore
		}

		for _, srcDomain := range srcSchema.Domains {
			destDomain := cache.GetOrCreateDomain(destSchema, srcDomain.DomainName)
			destDomain.DomainSchema = srcDomain.DomainSchema
			destDomain.DomainName = srcDomain.DomainName
			destDomain.UnderlyingType = srcDomain.UnderlyingType
			destDomain.CollationName = srcDomain.CollationName
			destDomain.IsNotNull = srcDomain.IsNotNull
			destDomain.ColumnDefault = srcDomain.ColumnDefault
			destDomain.CheckNames = cloneSlice(srcDomain.CheckNames)
			destDomain.CheckExprs = cloneSlice(srcDomain.CheckExprs)
			destDomain.Comment = srcDomain.Comment
			destDomain.Ignore = srcDomain.Ignore
		}

		for _, srcRoutine := range srcSchema.Routines {
			destRoutine := cache.GetOrCreateRoutine(destSchema, srcRoutine.RoutineName, srcRoutine.IdentityArguments)
			destRoutine.RoutineType = srcRoutine.RoutineType
			destRoutine.RoutineSchema = srcRoutine.RoutineSchema
			destRoutine.RoutineName = srcRoutine.RoutineName
			destRoutine.IdentityArguments = srcRoutine.IdentityArguments
			destRoutine.SQL = srcRoutine.SQL
			destRoutine.Comment = srcRoutine.Comment
			destRoutine.Ignore = srcRoutine.Ignore
			destRoutine.Attrs = cloneMap(srcRoutine.Attrs)
		}

		for _, srcView := range srcSchema.Views {
			destView := cache.GetOrCreateView(destSchema, srcView.ViewName)
			destView.ViewSchema = srcView.ViewSchema
			destView.ViewName = srcView.ViewName
			destView.IsMaterialized = srcView.IsMaterialized
			destView.SQL = srcView.SQL
			destView.Columns = cloneSlice(srcView.Columns)
			destView.ColumnTypes = cloneSlice(srcView.ColumnTypes)
			destView.EnumColumns = cloneSlice(srcView.EnumColumns)
			destView.Comment = srcView.Comment
			destView.Ignore = srcView.Ignore
			for _, srcIndex := range srcView.Indexes {
				destIndex := cache.GetOrCreateViewIndex(destView, srcIndex.IndexName, srcIndex.Columns)
				*destIndex = srcIndex
				destIndex.Columns = cloneSlice(srcIndex.Columns)
				destIndex.IncludeColumns = cloneSlice(srcIndex.IncludeColumns)
			}
			for _, srcTrigger := range srcView.Triggers {
				destTrigger := cache.GetOrCreateViewTrigger(destView, srcTrigger.TriggerName)
				*destTrigger = srcTrigger
				destTrigger.Attrs = cloneMap(srcTrigger.Attrs)
			}
		}

		for _, srcTable := range srcSchema.Tables {
			destTable := cache.GetOrCreateTable(destSchema, srcTable.TableName)
			destTable.TableSchema = srcTable.TableSchema
			destTable.TableName = srcTable.TableName
			destTable.SQL = srcTable.SQL
			destTable.Comment = srcTable.Comment
			destTable.Ignore = srcTable.Ignore
			for _, srcColumn := range srcTable.Columns {
				destColumn := cache.GetOrCreateColumn(destTable, srcColumn.ColumnName, srcColumn.ColumnType)
				*destColumn = srcColumn
			}
			for _, srcConstraint := range srcTable.Constraints {
				destConstraint := cache.GetOrCreateConstraint(destTable, srcConstraint.ConstraintName, srcConstraint.ConstraintType, srcConstraint.Columns)
				*destConstraint = srcConstraint
				destConstraint.Columns = cloneSlice(srcConstraint.Columns)
				destConstraint.ReferencesColumns = cloneSlice(srcConstraint.ReferencesColumns)
				destConstraint.ExclusionOperators = cloneSlice(srcConstraint.ExclusionOperators)
			}
			for _, srcIndex := range srcTable.Indexes {
				destIndex := cache.GetOrCreateIndex(destTable, srcIndex.IndexName, srcIndex.Columns)
				*destIndex = srcIndex
				destIndex.Columns = cloneSlice(srcIndex.Columns)
				destIndex.Descending = cloneSlice(srcIndex.Descending)
				destIndex.Opclasses = cloneSlice(srcIndex.Opclasses)
				destIndex.IncludeColumns = cloneSlice(srcIndex.IncludeColumns)
			}
			for _, srcTrigger := range srcTable.Triggers {
				destTrigger := cache.GetOrCreateTrigger(destTable, srcTrigger.TriggerName)
				*destTrigger = srcTrigger
				destTrigger.Attrs = cloneMap(srcTrigger.Attrs)
			}
		}
	}
	return nil
}

func cloneSlice[T any](src []T) []T {
	if src == nil {
		return nil
	}
	dest := make([]T, len(src))
	copy(dest, src)
	return dest
}

func cloneMap(src map[string]string) map[string]string {
	if src == nil {
		return nil
	}
	dest := make(map[string]string)
	for k, v := range src {
		dest[k] = v
	}
	return dest
}
