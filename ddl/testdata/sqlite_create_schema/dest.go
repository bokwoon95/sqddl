package _

import "github.com/bokwoon95/sq"

type ACTOR struct {
	sq.TableStruct
	ACTOR_ID    sq.NumberField `ddl:"primarykey autoincrement"`
	FIRST_NAME  sq.StringField `ddl:"notnull"`
	LAST_NAME   sq.StringField `ddl:"notnull index"`
	LAST_UPDATE sq.TimeField   `ddl:"notnull default=unixepoch()"`
}

type CATEGORY struct {
	sq.TableStruct
	CATEGORY_ID sq.NumberField `ddl:"primarykey"`
	NAME        sq.StringField `ddl:"notnull"`
	LAST_UPDATE sq.TimeField   `ddl:"notnull default=unixepoch()"`
}
