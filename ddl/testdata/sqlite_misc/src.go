package _

import "github.com/bokwoon95/sq"

type AUTHOR struct {
	sq.TableStruct
	AUTHOR_ID sq.NumberField `ddl:"primarykey"`
	NAME      sq.StringField
	EMAIL     sq.StringField
	METADATA  sq.StringField `ddl:"index"`
}

type POST struct {
	sq.TableStruct
	POST_ID  sq.NumberField `ddl:"primarykey"`
	CONTENTS sq.StringField
	METADATA sq.StringField `ddl:"index"`
}

type RESIDENCE struct {
	sq.TableStruct
	RESIDENCE_ID sq.NumberField `ddl:"primarykey"`
	COUNTRY      sq.StringField
	CITY         sq.StringField
	ADDRESS      sq.StringField
}
