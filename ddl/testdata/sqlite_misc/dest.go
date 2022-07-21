package _

import "github.com/bokwoon95/sq"

type AUTHOR struct {
	sq.TableStruct
	AUTHOR_ID sq.NumberField `ddl:"primarykey"`
	NAME      sq.StringField `ddl:"index"`
	EMAIL     sq.StringField `ddl:"unique notnull"`
	IS_ACTIVE sq.BooleanField
}

type POST struct {
	sq.TableStruct
	POST_ID   sq.NumberField `ddl:"primarykey"`
	CONTENTS  sq.StringField
	TAGS      sq.StringField
	AUTHOR_ID sq.NumberField `ddl:"references={author onupdate=cascade index}"`
}

type COUNTRY struct {
	sq.TableStruct
	COUNTRY_ID sq.NumberField `ddl:"primarykey"`
	COUNTRY    sq.StringField
}

type CITY struct {
	sq.TableStruct
	CITY_ID    sq.NumberField `ddl:"primarykey"`
	CITY       sq.StringField
	COUNTRY_ID sq.NumberField `ddl:"references={country onupdate=cascade index}"`
}

type ADDRESS struct {
	sq.TableStruct
	ADDRESS_ID sq.NumberField `ddl:"primarykey"`
	ADDRESS    sq.StringField
	CITY_ID    sq.NumberField `ddl:"references={city onupdate=cascade index}"`
}
