package _

import "github.com/bokwoon95/sq"

type PERSON struct {
	sq.TableStruct
	PERSON_ID      sq.NumberField `ddl:"primarykey identity"`
	NAME           sq.StringField `ddl:"type=VARCHAR(1000)"`
	EMAIL          sq.StringField `ddl:"type=VARCHAR(200) collate=latin1_bin notnull"`
	PASSWORD       sq.StringField `ddl:"type=VARCHAR(255)"`
	BIO            sq.StringField `ddl:"type=VARCHAR(255) default={'lorem ipsum'}"`
	NOTES          sq.StringField `ddl:"type=VARCHAR(1000)"`
	HEIGHT_METERS  sq.NumberField `ddl:"type=NUMERIC(3,2)"`
	WEIGHT_KILOS   sq.NumberField `ddl:"type=NUMERIC(3,2)"`
	SALARY_DOLLARS sq.NumberField `ddl:"type=DECIMAL(10,2)"`
	IP_ADDRESS     sq.AnyField    `ddl:"type=VARCHAR(255)"`
	COUNTRY_ID     sq.NumberField `ddl:"references={country deferred}"`
}

type COUNTRY struct {
	sq.TableStruct
	COUNTRY_ID sq.NumberField `ddl:"primarykey"`
	COUNTRY    sq.StringField `ddl:"collate=latin1_bin"`
}
