package ddl

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/bokwoon95/sqddl/internal/testutil"
)

func TestStructParser(t *testing.T) {
	file, err := os.Open("testdata/struct_parser/tables.go.txt")
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	defer file.Close()
	p := NewStructParser(nil)
	err = p.ParseFile(file)
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	gotCatalog := &Catalog{}
	err = p.WriteCatalog(gotCatalog)
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	wantCatalog := &Catalog{}
	b, err := os.ReadFile("testdata/struct_parser/tables.json")
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	err = json.Unmarshal(b, wantCatalog)
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	if diff := testutil.Diff(gotCatalog, wantCatalog); diff != "" {
		t.Error(testutil.Callers(), diff)
	}
}
