package sakila

import "github.com/bokwoon95/sq"

type ACTOR struct {
	sq.TableStruct
	ACTOR_ID           sq.NumberField `ddl:"primarykey"`
	FIRST_NAME         sq.StringField `ddl:"notnull"`
	LAST_NAME          sq.StringField `ddl:"notnull index"`
	FULL_NAME          sq.StringField `ddl:"generated"`
	FULL_NAME_REVERSED sq.StringField `ddl:"generated"`
	LAST_UPDATE        sq.TimeField   `ddl:"type=DATETIME notnull default=unixepoch()"`
}

type ADDRESS struct {
	sq.TableStruct
	ADDRESS_ID  sq.NumberField `ddl:"primarykey"`
	ADDRESS     sq.StringField `ddl:"notnull"`
	ADDRESS2    sq.StringField
	DISTRICT    sq.StringField `ddl:"notnull"`
	CITY_ID     sq.NumberField `ddl:"notnull references={city onupdate=cascade ondelete=restrict index}"`
	POSTAL_CODE sq.StringField
	PHONE       sq.StringField `ddl:"notnull"`
	LAST_UPDATE sq.TimeField   `ddl:"type=DATETIME notnull default=unixepoch()"`
}

type CATEGORY struct {
	sq.TableStruct
	CATEGORY_ID sq.NumberField `ddl:"primarykey"`
	NAME        sq.StringField `ddl:"notnull"`
	LAST_UPDATE sq.TimeField   `ddl:"type=DATETIME notnull default=unixepoch()"`
}

type CITY struct {
	sq.TableStruct
	CITY_ID     sq.NumberField `ddl:"primarykey"`
	CITY        sq.StringField `ddl:"notnull"`
	COUNTRY_ID  sq.NumberField `ddl:"notnull references={country onupdate=cascade ondelete=restrict index}"`
	LAST_UPDATE sq.TimeField   `ddl:"type=DATETIME notnull default=unixepoch()"`
}

type COUNTRY struct {
	sq.TableStruct
	COUNTRY_ID  sq.NumberField `ddl:"primarykey"`
	COUNTRY     sq.StringField `ddl:"notnull"`
	LAST_UPDATE sq.TimeField   `ddl:"type=DATETIME notnull default=unixepoch()"`
}

type CUSTOMER struct {
	sq.TableStruct
	CUSTOMER_ID sq.NumberField `ddl:"primarykey"`
	STORE_ID    sq.NumberField `ddl:"notnull references={store onupdate=cascade ondelete=restrict index}"`
	FIRST_NAME  sq.StringField `ddl:"notnull"`
	LAST_NAME   sq.StringField `ddl:"notnull index"`
	EMAIL       sq.StringField `ddl:"unique"`
	ADDRESS_ID  sq.NumberField `ddl:"notnull references={address onupdate=cascade ondelete=restrict index}"`
	ACTIVE      sq.NumberField `ddl:"notnull default=TRUE"`
	CREATE_DATE sq.TimeField   `ddl:"type=DATETIME notnull default=unixepoch()"`
	LAST_UPDATE sq.TimeField   `ddl:"type=DATETIME notnull default=unixepoch()"`
	_           struct{}       `ddl:"unique=email,first_name,last_name"`
}

type DEPARTMENT struct {
	sq.TableStruct
	DEPARTMENT_ID sq.UUIDField   `ddl:"notnull primarykey"`
	NAME          sq.StringField `ddl:"notnull"`
}

type EMPLOYEE struct {
	sq.TableStruct
	EMPLOYEE_ID sq.UUIDField   `ddl:"notnull primarykey"`
	NAME        sq.StringField `ddl:"notnull"`
	TITLE       sq.StringField `ddl:"notnull"`
	MANAGER_ID  sq.UUIDField   `ddl:"references={employee.employee_id index}"`
}

type EMPLOYEE_DEPARTMENT struct {
	sq.TableStruct `ddl:"primarykey=employee_id,department_id"`
	EMPLOYEE_ID    sq.UUIDField `ddl:"notnull references={employee index}"`
	DEPARTMENT_ID  sq.UUIDField `ddl:"notnull references={department index}"`
}

type FILM struct {
	sq.TableStruct
	FILM_ID              sq.NumberField `ddl:"primarykey"`
	TITLE                sq.StringField `ddl:"notnull index"`
	DESCRIPTION          sq.StringField
	RELEASE_YEAR         sq.NumberField
	LANGUAGE_ID          sq.NumberField `ddl:"notnull references={language onupdate=cascade ondelete=restrict index}"`
	ORIGINAL_LANGUAGE_ID sq.NumberField `ddl:"references={language.language_id onupdate=cascade ondelete=restrict index}"`
	RENTAL_DURATION      sq.NumberField `ddl:"notnull default=3"`
	RENTAL_RATE          sq.NumberField `ddl:"type=REAL notnull default=4.99"`
	LENGTH               sq.NumberField
	REPLACEMENT_COST     sq.NumberField `ddl:"type=REAL notnull default=19.99"`
	RATING               sq.StringField `ddl:"default='G'"`
	SPECIAL_FEATURES     sq.JSONField
	LAST_UPDATE          sq.TimeField `ddl:"type=DATETIME notnull default=unixepoch()"`
}

type FILM_ACTOR struct {
	sq.TableStruct `ddl:"primarykey=actor_id,film_id"`
	ACTOR_ID       sq.NumberField `ddl:"notnull references={actor onupdate=cascade ondelete=restrict}"`
	FILM_ID        sq.NumberField `ddl:"notnull references={film onupdate=cascade ondelete=restrict index}"`
	LAST_UPDATE    sq.TimeField   `ddl:"type=DATETIME notnull default=unixepoch()"`
}

type FILM_CATEGORY struct {
	sq.TableStruct `ddl:"primarykey=film_id,category_id"`
	FILM_ID        sq.NumberField `ddl:"notnull references={film onupdate=cascade ondelete=restrict}"`
	CATEGORY_ID    sq.NumberField `ddl:"notnull references={category onupdate=cascade ondelete=restrict}"`
	LAST_UPDATE    sq.TimeField   `ddl:"type=DATETIME notnull default=unixepoch()"`
}

type FILM_TEXT struct {
	sq.TableStruct `ddl:"virtual"`
	TITLE          sq.AnyField
	DESCRIPTION    sq.AnyField
	FILM_TEXT      sq.AnyField
	RANK           sq.AnyField
}

type INVENTORY struct {
	sq.TableStruct
	INVENTORY_ID sq.NumberField `ddl:"primarykey"`
	FILM_ID      sq.NumberField `ddl:"notnull references={film onupdate=cascade ondelete=restrict index}"`
	STORE_ID     sq.NumberField `ddl:"notnull references={store onupdate=cascade ondelete=restrict}"`
	LAST_UPDATE  sq.TimeField   `ddl:"type=DATETIME notnull default=unixepoch()"`
	_            struct{}       `ddl:"index=store_id,film_id"`
}

type LANGUAGE struct {
	sq.TableStruct
	LANGUAGE_ID sq.NumberField `ddl:"primarykey"`
	NAME        sq.StringField `ddl:"notnull"`
	LAST_UPDATE sq.TimeField   `ddl:"type=DATETIME notnull default=unixepoch()"`
}

type PAYMENT struct {
	sq.TableStruct
	PAYMENT_ID   sq.NumberField `ddl:"primarykey"`
	CUSTOMER_ID  sq.NumberField `ddl:"notnull references={customer onupdate=cascade ondelete=restrict index}"`
	STAFF_ID     sq.NumberField `ddl:"notnull references={staff onupdate=cascade ondelete=restrict index}"`
	RENTAL_ID    sq.NumberField `ddl:"references={rental onupdate=cascade ondelete=setnull index}"`
	AMOUNT       sq.NumberField `ddl:"type=REAL notnull"`
	PAYMENT_DATE sq.TimeField   `ddl:"type=DATETIME notnull default=unixepoch()"`
	LAST_UPDATE  sq.TimeField   `ddl:"type=DATETIME notnull default=unixepoch()"`
}

type RENTAL struct {
	sq.TableStruct
	RENTAL_ID    sq.NumberField `ddl:"primarykey"`
	RENTAL_DATE  sq.TimeField   `ddl:"type=DATETIME notnull default=unixepoch()"`
	INVENTORY_ID sq.NumberField `ddl:"notnull references={inventory onupdate=cascade ondelete=restrict index}"`
	CUSTOMER_ID  sq.NumberField `ddl:"notnull references={customer onupdate=cascade ondelete=restrict index}"`
	RETURN_DATE  sq.TimeField   `ddl:"type=DATETIME"`
	STAFF_ID     sq.NumberField `ddl:"notnull references={staff onupdate=cascade ondelete=restrict index}"`
	LAST_UPDATE  sq.TimeField   `ddl:"type=DATETIME notnull default=unixepoch()"`
	_            struct{}       `ddl:"index={inventory_id,customer_id,staff_id unique}"`
}

type STAFF struct {
	sq.TableStruct
	STAFF_ID    sq.NumberField `ddl:"primarykey"`
	FIRST_NAME  sq.StringField `ddl:"notnull"`
	LAST_NAME   sq.StringField `ddl:"notnull"`
	ADDRESS_ID  sq.NumberField `ddl:"notnull references={address onupdate=cascade ondelete=restrict index}"`
	PICTURE     sq.BinaryField
	EMAIL       sq.StringField `ddl:"unique"`
	STORE_ID    sq.NumberField `ddl:"references={store onupdate=cascade ondelete=restrict index}"`
	ACTIVE      sq.NumberField `ddl:"notnull default=TRUE"`
	USERNAME    sq.StringField `ddl:"notnull"`
	PASSWORD    sq.StringField
	LAST_UPDATE sq.TimeField `ddl:"type=DATETIME notnull default=unixepoch()"`
}

type STORE struct {
	sq.TableStruct
	STORE_ID         sq.NumberField `ddl:"primarykey"`
	MANAGER_STAFF_ID sq.NumberField `ddl:"notnull references={staff.staff_id onupdate=cascade ondelete=restrict index}"`
	ADDRESS_ID       sq.NumberField `ddl:"notnull references={address onupdate=cascade ondelete=restrict index}"`
	LAST_UPDATE      sq.TimeField   `ddl:"type=DATETIME notnull default=unixepoch()"`
}

type TASK struct {
	sq.TableStruct
	TASK_ID       sq.UUIDField   `ddl:"notnull primarykey"`
	EMPLOYEE_ID   sq.UUIDField   `ddl:"notnull"`
	DEPARTMENT_ID sq.UUIDField   `ddl:"notnull"`
	TASK          sq.StringField `ddl:"notnull"`
	DATA          sq.JSONField
	_             struct{} `ddl:"foreignkey={employee_id,department_id references=employee_department index}"`
}
