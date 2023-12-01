package _

import "github.com/bokwoon95/sq"

type CATEGORY struct {
	sq.TableStruct
	CATEGORY_ID sq.NumberField `ddl:"primarykey identity"`
	CATEGORY    sq.StringField `ddl:"unique"`
}

type MOVIE struct {
	sq.TableStruct
	MOVIE_ID    sq.NumberField `ddl:"primarykey"`
	TITLE       sq.StringField `ddl:"unique"`
	CATEGORY    sq.StringField `ddl:"references=category.category index"`
	SUBCATEGORY sq.StringField `ddl:"references=category.category index"`
	METADATA    sq.JSONField
}

type CINEMA struct {
	sq.TableStruct
	CINEMA_ID sq.NumberField `ddl:"primarykey identity"`
}
