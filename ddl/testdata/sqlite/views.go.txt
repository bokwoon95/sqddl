package sakila

import "github.com/bokwoon95/sq"

type ACTOR_INFO struct {
	sq.ViewStruct
	ACTOR_ID   sq.NumberField
	FIRST_NAME sq.StringField
	LAST_NAME  sq.StringField
	FILM_INFO  sq.AnyField
}

type CUSTOMER_LIST struct {
	sq.ViewStruct
	ID       sq.NumberField
	NAME     sq.AnyField
	ADDRESS  sq.StringField
	ZIP_CODE sq.StringField
	PHONE    sq.StringField
	CITY     sq.StringField
	COUNTRY  sq.StringField
	NOTES    sq.AnyField
	SID      sq.NumberField
}

type FILM_LIST struct {
	sq.ViewStruct
	FID         sq.NumberField
	TITLE       sq.StringField
	DESCRIPTION sq.StringField
	CATEGORY    sq.StringField
	PRICE       sq.NumberField
	LENGTH      sq.NumberField
	RATING      sq.StringField
	ACTORS      sq.AnyField
}

type FULL_ADDRESS struct {
	sq.ViewStruct
	COUNTRY_ID  sq.NumberField
	CITY_ID     sq.NumberField
	ADDRESS_ID  sq.NumberField
	COUNTRY     sq.StringField
	CITY        sq.StringField
	ADDRESS     sq.StringField
	ADDRESS2    sq.StringField
	DISTRICT    sq.StringField
	POSTAL_CODE sq.StringField
	PHONE       sq.StringField
	LAST_UPDATE sq.TimeField
}

type NICER_BUT_SLOWER_FILM_LIST struct {
	sq.ViewStruct
	FID         sq.NumberField
	TITLE       sq.StringField
	DESCRIPTION sq.StringField
	CATEGORY    sq.StringField
	PRICE       sq.NumberField
	LENGTH      sq.NumberField
	RATING      sq.StringField
	ACTORS      sq.AnyField
}

type SALES_BY_FILM_CATEGORY struct {
	sq.ViewStruct
	CATEGORY    sq.StringField
	TOTAL_SALES sq.AnyField
}

type SALES_BY_STORE struct {
	sq.ViewStruct
	STORE       sq.AnyField
	MANAGER     sq.AnyField
	TOTAL_SALES sq.AnyField
}

type STAFF_LIST struct {
	sq.ViewStruct
	ID       sq.NumberField
	NAME     sq.AnyField
	ADDRESS  sq.StringField
	ZIP_CODE sq.StringField
	PHONE    sq.StringField
	CITY     sq.StringField
	COUNTRY  sq.StringField
	SID      sq.NumberField
}
