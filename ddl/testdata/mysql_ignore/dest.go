package _

import "github.com/bokwoon95/sq"

type FILM struct {
	sq.TableStruct
	FILM_ID     sq.NumberField `ddl:"notnull informix:primarykey informix:index"`
	TITLE       sq.StringField `ddl:"notnull len=255"`
	FULLTEXT    sq.AnyField `ddl:"dialect=informix type=TSVECTOR index={. using=gin}"`
}

type FILM_TEXT struct {
	sq.TableStruct `ddl:"dialect=informix"`
	FILM_ID        sq.NumberField `ddl:"dialect=informix primarykey"`
	TITLE          sq.StringField
	DESCRIPTION    sq.StringField `ddl:"type=TEXT"`
	_              struct{}       `ddl:"index={title,description using=fulltext}"`
}
