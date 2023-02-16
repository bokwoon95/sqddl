package _

import "github.com/bokwoon95/sq"

type FILM struct {
	sq.TableStruct
	FILM_ID     sq.NumberField `ddl:"notnull"`
	TITLE       sq.StringField `ddl:"notnull len=255 informix:index informix:unique"`
	DESCRIPTION sq.StringField `ddl:"dialect=informix"`
}

type COUNTRY struct {
	sq.TableStruct `ddl:"dialect=informix"`
	COUNTRY_ID     sq.NumberField `ddl:"primarykey auto_increment identity"`
	COUNTRY        sq.StringField `ddl:"notnull len=50"`
	LAST_UPDATE    sq.TimeField   `ddl:"notnull default=CURRENT_TIMESTAMP sqlite:default=unixepoch() onupdatecurrenttimestamp"`
}
