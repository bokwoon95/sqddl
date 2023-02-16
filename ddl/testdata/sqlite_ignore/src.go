package _

import "github.com/bokwoon95/sq"

type FILM struct {
	sq.TableStruct
	FILM_ID     sq.NumberField `ddl:"primarykey auto_increment identity"`
	TITLE       sq.StringField `ddl:"notnull len=255 index"`
	DESCRIPTION sq.StringField
}
