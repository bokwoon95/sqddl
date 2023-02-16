package _

import "github.com/bokwoon95/sq"

type FILM struct {
	sq.TableStruct
	FILM_ID     sq.NumberField `ddl:"primarykey auto_increment identity"`
	TITLE       sq.StringField `ddl:"notnull len=255 index"`
	DESCRIPTION sq.StringField
	FULLTEXT    sq.AnyField `ddl:"type=TSVECTOR index={. using=gin}"`
}

type FILM_TEXT struct {
	sq.TableStruct `ddl:"dialect=mysql"`
	FILM_ID        sq.NumberField `ddl:"dialect=mysql primarykey"`
	TITLE          sq.StringField
	DESCRIPTION    sq.StringField `ddl:"type=TEXT"`
	_              struct{}       `ddl:"index={title,description using=fulltext}"`
}
