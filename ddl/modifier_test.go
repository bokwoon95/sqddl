package ddl

import (
	"testing"

	"github.com/bokwoon95/sqddl/internal/testutil"
)

func Test_popRawValue(t *testing.T) {
	type TT struct {
		description   string
		str           string
		wantValue     string
		wantRemainder string
	}

	tests := []TT{{
		description: "empty",
	}, {
		description:   "one",
		str:           "primarykey",
		wantValue:     "primarykey",
		wantRemainder: "",
	}, {
		description:   "basic",
		str:           "primarykey auto_increment identity",
		wantValue:     "primarykey",
		wantRemainder: " auto_increment identity",
	}, {
		description:   "brace quoted",
		str:           "{one two three} four five",
		wantValue:     "one two three",
		wantRemainder: " four five",
	}, {
		description:   "entirely brace quoted",
		str:           "{one two three}",
		wantValue:     "one two three",
		wantRemainder: "",
	}, {
		description:   "braces in the middle",
		str:           "abc{one two three}d efg hij",
		wantValue:     "abc{one",
		wantRemainder: " two three}d efg hij",
	}, {
		description:   "preceding space",
		str:           "  one  two  three",
		wantValue:     "one",
		wantRemainder: "  two  three",
	}}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.description, func(t *testing.T) {
			gotValue, gotRemainder, err := popValue(tt.str)
			if err != nil {
				t.Fatal(testutil.Callers(), err)
			}
			if diff := testutil.Diff(gotValue, tt.wantValue); diff != "" {
				t.Error(testutil.Callers(), diff)
			}
			if diff := testutil.Diff(gotRemainder, tt.wantRemainder); diff != "" {
				t.Error(testutil.Callers(), diff)
			}
		})
	}

	t.Run("unclosed brace", func(t *testing.T) {
		_, _, err := popValue("{one")
		if err == nil {
			t.Error(testutil.Callers(), "expected error but got nil")
		}
		_, _, err = popValue("{one {two} {{{three")
		if err == nil {
			t.Error(testutil.Callers(), "expected error but got nil")
		}
	})
}

func Test_popModifier(t *testing.T) {
	type TT struct {
		description   string
		str           string
		wantModifier  Modifier
		wantRemainder string
	}

	tests := []TT{{
		description: "empty",
	}, {
		description:   "one",
		str:           "primarykey",
		wantModifier:  Modifier{Name: "primarykey"},
		wantRemainder: "",
	}, {
		description:   "basic",
		str:           "primarykey auto_increment identity",
		wantModifier:  Modifier{Name: "primarykey"},
		wantRemainder: " auto_increment identity",
	}, {
		description:   "brace quoted",
		str:           "{one two three} four five",
		wantModifier:  Modifier{Name: "{one"},
		wantRemainder: " two three} four five",
	}, {
		description:   "entirely brace quoted",
		str:           "{one two three}",
		wantModifier:  Modifier{Name: "{one"},
		wantRemainder: " two three}",
	}, {
		description:   "braces in the middle",
		str:           "abc{one two three}d efg hij",
		wantModifier:  Modifier{Name: "abc{one"},
		wantRemainder: " two three}d efg hij",
	}, {
		description:   "preceding space",
		str:           "  one  two  three",
		wantModifier:  Modifier{Name: "one"},
		wantRemainder: "  two  three",
	}, {
		description:   "name and empty value",
		str:           "index= primarykey auto_increment",
		wantModifier:  Modifier{Name: "index"},
		wantRemainder: " primarykey auto_increment",
	}, {
		description:   "name and rawvalue",
		str:           "index=actor_id primarykey auto_increment",
		wantModifier:  Modifier{Name: "index", RawValue: "actor_id"},
		wantRemainder: " primarykey auto_increment",
	}, {
		description:   "name and brace quoted rawvalue",
		str:           "index={actor_id unique using=HASH} primarykey auto_increment",
		wantModifier:  Modifier{Name: "index", RawValue: "actor_id unique using=HASH"},
		wantRemainder: " primarykey auto_increment",
	}}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.description, func(t *testing.T) {
			gotModifier, gotRemainder, err := popModifier(tt.str)
			if err != nil {
				t.Fatal(testutil.Callers(), err)
			}
			if diff := testutil.Diff(gotModifier, tt.wantModifier); diff != "" {
				t.Error(testutil.Callers(), diff)
			}
			if diff := testutil.Diff(gotModifier, tt.wantModifier); diff != "" {
				t.Error(testutil.Callers(), diff)
			}
			if diff := testutil.Diff(gotRemainder, tt.wantRemainder); diff != "" {
				t.Error(testutil.Callers(), diff)
			}
		})
	}
}

func TestParseRawValue(t *testing.T) {
	type TT struct {
		description      string
		rawvalue         string
		wantValue        string
		wantSubmodifiers Modifiers
	}

	tests := []TT{{
		description: "empty",
	}, {
		description: "basic",
		rawvalue:    "notnull unique index={. unique} name=testing references={inventory onupdate=cascade ondelete=restrict}",
		wantValue:   "notnull",
		wantSubmodifiers: Modifiers{
			{Name: "unique"},
			{Name: "index", RawValue: ". unique"},
			{Name: "name", RawValue: "testing"},
			{Name: "references", RawValue: "inventory onupdate=cascade ondelete=restrict"},
		},
	}, {
		description: "dialects",
		rawvalue:    "primarykey sqlite:notnull mysql,sqlserver:type=VARCHAR(20)",
		wantValue:   "primarykey",
		wantSubmodifiers: Modifiers{
			{Dialects: []string{"sqlite"}, Name: "notnull"},
			{Dialects: []string{"mysql", "sqlserver"}, Name: "type", RawValue: "VARCHAR(20)"},
		},
	}, {
		description: "trailing whitespace",
		rawvalue:    "primarykey auto_increment identity  ",
		wantValue:   "primarykey",
		wantSubmodifiers: Modifiers{
			{Name: "auto_increment"},
			{Name: "identity"},
		},
	}}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.description, func(t *testing.T) {
			m := Modifier{RawValue: tt.rawvalue}
			err := m.ParseRawValue()
			if err != nil {
				t.Fatal(testutil.Callers(), err)
			}
			if diff := testutil.Diff(m.Value, tt.wantValue); diff != "" {
				t.Error(testutil.Callers(), diff)
			}
			if diff := testutil.Diff(m.Submodifiers, tt.wantSubmodifiers); diff != "" {
				t.Error(testutil.Callers(), diff)
			}
		})
	}

	t.Run("unclosed brace in rawvalue", func(t *testing.T) {
		m := Modifier{RawValue: "{testing index=abc"}
		err := m.ParseRawValue()
		if err == nil {
			t.Error(testutil.Callers(), "expected error but got nil")
		}
	})

	t.Run("unclosed brace in submodifiers", func(t *testing.T) {
		m := Modifier{RawValue: "testing index={abc"}
		err := m.ParseRawValue()
		if err == nil {
			t.Error(testutil.Callers(), "expected error but got nil")
		}
	})
}

func TestModifiers_String(t *testing.T) {
	type TT struct {
		description string
		modifiers   Modifiers
		wantString  string
	}

	tests := []TT{{
		description: "empty",
		modifiers: Modifiers{
			{}, {}, {},
			{Name: "primarykey"},
			{},
			{Name: "unique"},
		},
		wantString: "primarykey unique",
	}, {
		description: "basic",
		modifiers: Modifiers{
			{Name: "notnull"},
			{Name: "unique"},
			{Name: "index", RawValue: ". unique"},
			{Name: "name", RawValue: "testing"},
			{Name: "references", RawValue: "inventory onupdate=cascade ondelete=restrict"},
		},
		wantString: "notnull unique index={. unique} name=testing references={inventory onupdate=cascade ondelete=restrict}",
	}, {
		description: "nested",
		modifiers: Modifiers{
			{Name: "notnull"},
			{Name: "unique"},
			{Name: "index", Submodifiers: Modifiers{
				{Name: "unique"},
			}},
			{Name: "name", Value: "testing a b c", Submodifiers: Modifiers{
				{Name: "1"},
				{Name: "2"},
				{Name: "3"},
			}},
			{Name: "check", Value: "col1 = col2"},
			{Name: "references", Value: "inventory", Submodifiers: Modifiers{
				{Name: "onupdate", RawValue: "cascade"},
				{Name: "ondelete", RawValue: "restrict"},
			}},
		},
		wantString: "notnull unique index={. unique} name={{testing a b c} 1 2 3} check={{col1 = col2}} references={inventory onupdate=cascade ondelete=restrict}",
	}}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.description, func(t *testing.T) {
			gotString := tt.modifiers.String()
			if diff := testutil.Diff(gotString, tt.wantString); diff != "" {
				t.Error(testutil.Callers(), diff)
			}
		})
	}
}

func TestModifier_IncludesDialect(t *testing.T) {
	type TT struct {
		description string
		modifier    Modifier
		dialect     string
		wantExclude bool
	}

	tests := []TT{{
		description: "empty",
		modifier:    Modifier{},
		dialect:     "postgres",
		wantExclude: false,
	}, {
		description: "doesn't exclude",
		modifier: Modifier{
			Dialects: []string{"sqlite", "postgres", "mysql"},
		},
		dialect:     "postgres",
		wantExclude: false,
	}, {
		description: "excludes",
		modifier: Modifier{
			Dialects: []string{"sqlite", "postgres", "mysql"},
		},
		dialect:     "sqlserver",
		wantExclude: true,
	}}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.description, func(t *testing.T) {
			gotExclude := tt.modifier.ExcludesDialect(tt.dialect)
			if diff := testutil.Diff(gotExclude, tt.wantExclude); diff != "" {
				t.Error(testutil.Callers(), diff)
			}
		})
	}
}
