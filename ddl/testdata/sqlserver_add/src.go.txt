package _

import "github.com/bokwoon95/sq"

type CATEGORY struct {
	sq.TableStruct
	CATEGORY sq.StringField `ddl:"primarykey"`
}

type MOVIE struct {
	sq.TableStruct
	MOVIE_ID    sq.NumberField
	TITLE       sq.StringField
	CATEGORY    sq.StringField
	SUBCATEGORY sq.StringField
}

type CINEMA struct {
	sq.TableStruct
	CINEMA_ID sq.NumberField `ddl:"primarykey"`
}
