package _

import "github.com/bokwoon95/sq"

type PERSON struct {
	sq.TableStruct
	PERSON_ID      sq.NumberField `ddl:"primarykey"`
	NAME           sq.StringField `ddl:"type=VARCHAR(255)"`
	EMAIL          sq.StringField `ddl:"type=TEXT"`
	PASSWORD       sq.StringField `ddl:"type=TEXT default='password'"`
	BIO            sq.StringField `ddl:"type=VARCHAR(1000)"`
	NOTES          sq.StringField `ddl:"type=VARCHAR(255)"`
	HEIGHT_METERS  sq.NumberField `ddl:"type=NUMERIC(3,1)"`
	WEIGHT_KILOS   sq.NumberField `ddl:"type=NUMERIC(5,2)"`
	SALARY_DOLLARS sq.NumberField `ddl:"type=DECIMAL(5,2)"`
	IP_ADDRESS     sq.AnyField    `ddl:"type=VARCHAR(15)"`
	COUNTRY_ID     sq.NumberField `ddl:"references=country"`
}

type COUNTRY struct {
	sq.TableStruct
	COUNTRY_ID sq.NumberField `ddl:"primarykey"`
	COUNTRY    sq.StringField
}
