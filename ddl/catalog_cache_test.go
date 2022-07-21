package ddl

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/bokwoon95/sqddl/internal/testutil"
)

func Test_Catalog_WriteCatalog(t *testing.T) {
	f, err := os.Open("testdata/postgres/schema.json")
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	defer f.Close()
	c1 := &Catalog{}
	err = json.NewDecoder(f).Decode(c1)
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	c2 := &Catalog{}
	err = c1.WriteCatalog(c2)
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	if diff := testutil.Diff(c2, c1); diff != "" {
		t.Error(testutil.Callers(), diff)
	}
}

func TestCatalogCache(t *testing.T) {
	f, err := os.Open("testdata/postgres/schema.json")
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	defer f.Close()
	catalog := &Catalog{}
	err = json.NewDecoder(f).Decode(catalog)
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	cache := NewCatalogCache(catalog)
	const lorem_ipsum = "lorem ipsum"

	t.Run("schema", func(t *testing.T) {
		// get nonexistent schema
		gotSchema := cache.GetSchema(catalog, lorem_ipsum)
		if diff := testutil.Diff(gotSchema, (*Schema)(nil)); diff != "" {
			t.Fatal(testutil.Callers(), diff)
		}
		// create schema and assert it was created
		wantSchema := Schema{
			SchemaName: lorem_ipsum,
		}
		cache.AddOrUpdateSchema(catalog, wantSchema)
		gotSchema = cache.GetSchema(catalog, lorem_ipsum)
		if diff := testutil.Diff(*gotSchema, wantSchema); diff != "" {
			t.Fatal(testutil.Callers(), diff)
		}
		// modify schema and assert it was modified
		wantSchema.Comment = lorem_ipsum
		cache.AddOrUpdateSchema(catalog, wantSchema)
		gotSchema = cache.GetOrCreateSchema(catalog, lorem_ipsum)
		if diff := testutil.Diff(*gotSchema, wantSchema); diff != "" {
			t.Fatal(testutil.Callers(), diff)
		}
	})

	t.Run("enum", func(t *testing.T) {
		schema := cache.GetOrCreateSchema(catalog, lorem_ipsum)
		// get nonexistent enum
		gotEnum := cache.GetEnum(schema, lorem_ipsum)
		if diff := testutil.Diff(gotEnum, (*Enum)(nil)); diff != "" {
			t.Error(testutil.Callers(), diff)
		}
		// create enum and assert it was created
		wantEnum := Enum{
			EnumName: lorem_ipsum,
		}
		cache.AddOrUpdateEnum(schema, wantEnum)
		gotEnum = cache.GetEnum(schema, lorem_ipsum)
		if diff := testutil.Diff(*gotEnum, wantEnum); diff != "" {
			t.Fatal(testutil.Callers(), diff)
		}
		// modify enum and assert it was modified
		wantEnum.Comment = lorem_ipsum
		cache.AddOrUpdateEnum(schema, wantEnum)
		gotEnum = cache.GetOrCreateEnum(schema, lorem_ipsum)
		if diff := testutil.Diff(*gotEnum, wantEnum); diff != "" {
			t.Fatal(testutil.Callers(), diff)
		}
	})

	t.Run("domain", func(t *testing.T) {
		schema := cache.GetOrCreateSchema(catalog, lorem_ipsum)
		// get nonexistent domain
		gotDomain := cache.GetDomain(schema, lorem_ipsum)
		if diff := testutil.Diff(gotDomain, (*Domain)(nil)); diff != "" {
			t.Error(testutil.Callers(), diff)
		}
		// create domain and assert it was created
		wantDomain := Domain{
			DomainName: lorem_ipsum,
		}
		cache.AddOrUpdateDomain(schema, wantDomain)
		gotDomain = cache.GetDomain(schema, lorem_ipsum)
		if diff := testutil.Diff(*gotDomain, wantDomain); diff != "" {
			t.Fatal(testutil.Callers(), diff)
		}
		// modify domain and assert it was modified
		wantDomain.Comment = lorem_ipsum
		cache.AddOrUpdateDomain(schema, wantDomain)
		gotDomain = cache.GetOrCreateDomain(schema, lorem_ipsum)
		if diff := testutil.Diff(*gotDomain, wantDomain); diff != "" {
			t.Fatal(testutil.Callers(), diff)
		}
	})

	t.Run("routine", func(t *testing.T) {
		schema := cache.GetOrCreateSchema(catalog, lorem_ipsum)
		// get nonexistent routine
		gotRoutine := cache.GetRoutine(schema, lorem_ipsum, "")
		if diff := testutil.Diff(gotRoutine, (*Routine)(nil)); diff != "" {
			t.Error(testutil.Callers(), diff)
		}
		// create routine and assert it was created
		wantRoutine := Routine{
			RoutineName: lorem_ipsum,
		}
		cache.AddOrUpdateRoutine(schema, wantRoutine)
		gotRoutine = cache.GetRoutine(schema, lorem_ipsum, "")
		if diff := testutil.Diff(*gotRoutine, wantRoutine); diff != "" {
			t.Fatal(testutil.Callers(), diff)
		}
		// modify routine and assert it was modified
		wantRoutine.Comment = lorem_ipsum
		cache.AddOrUpdateRoutine(schema, wantRoutine)
		gotRoutine = cache.GetOrCreateRoutine(schema, lorem_ipsum, "")
		if diff := testutil.Diff(*gotRoutine, wantRoutine); diff != "" {
			t.Fatal(testutil.Callers(), diff)
		}
	})

	t.Run("view", func(t *testing.T) {
		schema := cache.GetOrCreateSchema(catalog, lorem_ipsum)
		// get nonexistent view
		gotView := cache.GetView(schema, lorem_ipsum)
		if diff := testutil.Diff(gotView, (*View)(nil)); diff != "" {
			t.Error(testutil.Callers(), diff)
		}
		// create view and assert it was created
		wantView := View{
			ViewName: lorem_ipsum,
		}
		cache.AddOrUpdateView(schema, wantView)
		gotView = cache.GetView(schema, lorem_ipsum)
		if diff := testutil.Diff(*gotView, wantView); diff != "" {
			t.Fatal(testutil.Callers(), diff)
		}
		// modify view and assert it was modified
		wantView.Comment = lorem_ipsum
		cache.AddOrUpdateView(schema, wantView)
		gotView = cache.GetOrCreateView(schema, lorem_ipsum)
		if diff := testutil.Diff(*gotView, wantView); diff != "" {
			t.Fatal(testutil.Callers(), diff)
		}
	})

	t.Run("table", func(t *testing.T) {
		schema := cache.GetOrCreateSchema(catalog, lorem_ipsum)
		// get nonexistent table
		gotTable := cache.GetTable(schema, lorem_ipsum)
		if diff := testutil.Diff(gotTable, (*Table)(nil)); diff != "" {
			t.Error(testutil.Callers(), diff)
		}
		// create table and assert it was created
		wantTable := Table{
			TableName: lorem_ipsum,
		}
		cache.AddOrUpdateTable(schema, wantTable)
		gotTable = cache.GetTable(schema, lorem_ipsum)
		if diff := testutil.Diff(*gotTable, wantTable); diff != "" {
			t.Fatal(testutil.Callers(), diff)
		}
		// modify table and assert it was modified
		wantTable.Comment = lorem_ipsum
		cache.AddOrUpdateTable(schema, wantTable)
		gotTable = cache.GetOrCreateTable(schema, lorem_ipsum)
		if diff := testutil.Diff(*gotTable, wantTable); diff != "" {
			t.Fatal(testutil.Callers(), diff)
		}
	})

	t.Run("column", func(t *testing.T) {
		schema := cache.GetOrCreateSchema(catalog, lorem_ipsum)
		table := cache.GetOrCreateTable(schema, lorem_ipsum)
		// get nonexistent column
		gotColumn := cache.GetColumn(table, lorem_ipsum)
		if diff := testutil.Diff(gotColumn, (*Column)(nil)); diff != "" {
			t.Error(testutil.Callers(), diff)
		}
		// create column and assert it was created
		wantColumn := Column{
			ColumnName: lorem_ipsum,
			ColumnType: lorem_ipsum,
		}
		cache.AddOrUpdateColumn(table, wantColumn)
		gotColumn = cache.GetColumn(table, lorem_ipsum)
		if diff := testutil.Diff(*gotColumn, wantColumn); diff != "" {
			t.Fatal(testutil.Callers(), diff)
		}
		// modify column and assert it was modified
		wantColumn.Comment = lorem_ipsum
		cache.AddOrUpdateColumn(table, wantColumn)
		gotColumn = cache.GetOrCreateColumn(table, lorem_ipsum, lorem_ipsum)
		if diff := testutil.Diff(*gotColumn, wantColumn); diff != "" {
			t.Fatal(testutil.Callers(), diff)
		}
	})

	t.Run("constraints", func(t *testing.T) {
		schema := cache.GetOrCreateSchema(catalog, lorem_ipsum)
		table := cache.GetOrCreateTable(schema, lorem_ipsum)
		// get nonexistent primary key
		gotConstraint := cache.GetConstraint(table, lorem_ipsum)
		if diff := testutil.Diff(gotConstraint, (*Constraint)(nil)); diff != "" {
			t.Error(testutil.Callers(), diff)
		}
		// create primary key and assert it was created
		wantConstraint := Constraint{
			ConstraintName: lorem_ipsum,
			ConstraintType: PRIMARY_KEY,
			Columns:        []string{lorem_ipsum},
		}
		cache.AddOrUpdateConstraint(table, wantConstraint)
		gotConstraint = cache.GetConstraint(table, lorem_ipsum)
		if diff := testutil.Diff(*gotConstraint, wantConstraint); diff != "" {
			t.Fatal(testutil.Callers(), diff)
		}
		gotPkey := cache.GetPrimaryKey(table)
		if diff := testutil.Diff(*gotPkey, wantConstraint); diff != "" {
			t.Fatal(testutil.Callers(), diff)
		}
		// modify primary key and assert it was modified
		wantConstraint.Comment = lorem_ipsum
		cache.AddOrUpdateConstraint(table, wantConstraint)
		gotConstraint = cache.GetOrCreateConstraint(table, lorem_ipsum, PRIMARY_KEY, []string{lorem_ipsum})
		if diff := testutil.Diff(*gotConstraint, wantConstraint); diff != "" {
			t.Fatal(testutil.Callers(), diff)
		}
		gotPkey = cache.GetPrimaryKey(table)
		if diff := testutil.Diff(*gotPkey, wantConstraint); diff != "" {
			t.Fatal(testutil.Callers(), diff)
		}
		// get nonexistent foreign keys
		gotFkeys := cache.GetForeignKeys(table)
		if diff := testutil.Diff(gotFkeys, ([]*Constraint)(nil)); diff != "" {
			t.Error(testutil.Callers(), diff)
		}
		// create foreign keys and assert they were created
		wantFkeys := []Constraint{
			{ConstraintName: "1", ConstraintType: FOREIGN_KEY},
			{ConstraintName: "2", ConstraintType: FOREIGN_KEY},
			{ConstraintName: "3", ConstraintType: FOREIGN_KEY},
		}
		for _, fkey := range wantFkeys {
			cache.AddOrUpdateConstraint(table, fkey)
		}
		gotFkeys = cache.GetForeignKeys(table)
		for i, wantFkey := range wantFkeys {
			gotFkey := *gotFkeys[i]
			if diff := testutil.Diff(gotFkey, wantFkey); diff != "" {
				t.Error(testutil.Callers(), diff)
			}
		}
		// modify foreign keys and assert they were modified
		for i := range wantFkeys {
			wantFkeys[i].Comment = lorem_ipsum
		}
		for _, fkey := range wantFkeys {
			cache.AddOrUpdateConstraint(table, fkey)
		}
		gotFkeys = cache.GetForeignKeys(table)
		for i, wantFkey := range wantFkeys {
			gotFkey := *gotFkeys[i]
			if diff := testutil.Diff(gotFkey, wantFkey); diff != "" {
				t.Error(testutil.Callers(), diff)
			}
		}
	})

	t.Run("index", func(t *testing.T) {
		schema := cache.GetOrCreateSchema(catalog, lorem_ipsum)
		table := cache.GetOrCreateTable(schema, lorem_ipsum)
		// get nonexistent index
		gotIndex := cache.GetIndex(table, lorem_ipsum)
		if diff := testutil.Diff(gotIndex, (*Index)(nil)); diff != "" {
			t.Error(testutil.Callers(), diff)
		}
		// create index and assert it was created
		wantIndex := Index{
			IndexName: lorem_ipsum,
			Columns:   []string{lorem_ipsum},
		}
		cache.AddOrUpdateIndex(table, wantIndex)
		gotIndex = cache.GetIndex(table, lorem_ipsum)
		if diff := testutil.Diff(*gotIndex, wantIndex); diff != "" {
			t.Fatal(testutil.Callers(), diff)
		}
		// modify index and assert it was modified
		wantIndex.Comment = lorem_ipsum
		cache.AddOrUpdateIndex(table, wantIndex)
		gotIndex = cache.GetOrCreateIndex(table, lorem_ipsum, []string{lorem_ipsum})
		if diff := testutil.Diff(*gotIndex, wantIndex); diff != "" {
			t.Fatal(testutil.Callers(), diff)
		}
	})

	t.Run("trigger", func(t *testing.T) {
		schema := cache.GetOrCreateSchema(catalog, lorem_ipsum)
		table := cache.GetOrCreateTable(schema, lorem_ipsum)
		// get nonexistent trigger
		gotTrigger := cache.GetTrigger(table, lorem_ipsum)
		if diff := testutil.Diff(gotTrigger, (*Trigger)(nil)); diff != "" {
			t.Error(testutil.Callers(), diff)
		}
		// create trigger and assert it was created
		wantTrigger := Trigger{
			TriggerName: lorem_ipsum,
		}
		cache.AddOrUpdateTrigger(table, wantTrigger)
		gotTrigger = cache.GetTrigger(table, lorem_ipsum)
		if diff := testutil.Diff(*gotTrigger, wantTrigger); diff != "" {
			t.Fatal(testutil.Callers(), diff)
		}
		// modify trigger and assert it was modified
		wantTrigger.Comment = lorem_ipsum
		cache.AddOrUpdateTrigger(table, wantTrigger)
		gotTrigger = cache.GetOrCreateTrigger(table, lorem_ipsum)
		if diff := testutil.Diff(*gotTrigger, wantTrigger); diff != "" {
			t.Fatal(testutil.Callers(), diff)
		}
	})

	t.Run("view index", func(t *testing.T) {
		const lorem_ipsum = "lorem ipsum view"
		schema := cache.GetOrCreateSchema(catalog, lorem_ipsum)
		view := cache.GetOrCreateView(schema, lorem_ipsum)
		// get nonexistent view index
		gotIndex := cache.GetViewIndex(view, lorem_ipsum)
		if diff := testutil.Diff(gotIndex, (*Index)(nil)); diff != "" {
			t.Error(testutil.Callers(), diff)
		}
		// create view index and assert it was created
		wantIndex := Index{
			IndexName: lorem_ipsum,
			Columns:   []string{lorem_ipsum},
		}
		cache.AddOrUpdateViewIndex(view, wantIndex)
		gotIndex = cache.GetViewIndex(view, lorem_ipsum)
		if diff := testutil.Diff(*gotIndex, wantIndex); diff != "" {
			t.Fatal(testutil.Callers(), diff)
		}
		// modify view index and assert it was modified
		wantIndex.Comment = lorem_ipsum
		cache.AddOrUpdateViewIndex(view, wantIndex)
		gotIndex = cache.GetOrCreateViewIndex(view, lorem_ipsum, []string{lorem_ipsum})
		if diff := testutil.Diff(*gotIndex, wantIndex); diff != "" {
			t.Fatal(testutil.Callers(), diff)
		}
	})

	t.Run("view trigger", func(t *testing.T) {
		const lorem_ipsum = "lorem ipsum view"
		schema := cache.GetOrCreateSchema(catalog, lorem_ipsum)
		view := cache.GetOrCreateView(schema, lorem_ipsum)
		// get nonexistent view trigger
		gotTrigger := cache.GetViewTrigger(view, lorem_ipsum)
		if diff := testutil.Diff(gotTrigger, (*Trigger)(nil)); diff != "" {
			t.Error(testutil.Callers(), diff)
		}
		// create view trigger and assert it was created
		wantTrigger := Trigger{
			TriggerName: lorem_ipsum,
		}
		cache.AddOrUpdateViewTrigger(view, wantTrigger)
		gotTrigger = cache.GetViewTrigger(view, lorem_ipsum)
		if diff := testutil.Diff(*gotTrigger, wantTrigger); diff != "" {
			t.Fatal(testutil.Callers(), diff)
		}
		// modify view trigger and assert it was modified
		wantTrigger.Comment = lorem_ipsum
		cache.AddOrUpdateViewTrigger(view, wantTrigger)
		gotTrigger = cache.GetOrCreateViewTrigger(view, lorem_ipsum)
		if diff := testutil.Diff(*gotTrigger, wantTrigger); diff != "" {
			t.Fatal(testutil.Callers(), diff)
		}
	})
}
