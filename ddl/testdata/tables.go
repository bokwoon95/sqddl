package _

import "github.com/bokwoon95/sq"

type ACTOR struct {
	sq.TableStruct
	ACTOR_ID           sq.NumberField `ddl:"primarykey auto_increment identity"`
	FIRST_NAME         sq.StringField `ddl:"notnull len=45"`
	LAST_NAME          sq.StringField `ddl:"notnull len=45 index"`
	FULL_NAME          sq.StringField `ddl:"generated"`
	FULL_NAME_REVERSED sq.StringField `ddl:"generated"`
	LAST_UPDATE        sq.TimeField   `ddl:"notnull default=CURRENT_TIMESTAMP sqlite:default=unixepoch() onupdatecurrenttimestamp"`
}

type ADDRESS struct {
	sq.TableStruct
	ADDRESS_ID  sq.NumberField `ddl:"primarykey auto_increment identity"`
	ADDRESS     sq.StringField `ddl:"notnull len=50"`
	ADDRESS2    sq.StringField `ddl:"len=50"`
	DISTRICT    sq.StringField `ddl:"notnull len=20"`
	CITY_ID     sq.NumberField `ddl:"notnull references={city onupdate=cascade ondelete=restrict index}"`
	POSTAL_CODE sq.StringField `ddl:"len=10"`
	PHONE       sq.StringField `ddl:"notnull len=20"`
	LAST_UPDATE sq.TimeField   `ddl:"notnull default=CURRENT_TIMESTAMP sqlite:default=unixepoch() onupdatecurrenttimestamp"`
}

type CATEGORY struct {
	sq.TableStruct
	CATEGORY_ID sq.NumberField `ddl:"primarykey auto_increment identity"`
	NAME        sq.StringField `ddl:"notnull len=45"`
	LAST_UPDATE sq.TimeField   `ddl:"notnull default=CURRENT_TIMESTAMP sqlite:default=unixepoch() onupdatecurrenttimestamp"`
}

type CITY struct {
	sq.TableStruct
	CITY_ID     sq.NumberField `ddl:"primarykey auto_increment identity"`
	CITY        sq.StringField `ddl:"notnull len=50"`
	COUNTRY_ID  sq.NumberField `ddl:"notnull references={country onupdate=cascade ondelete=restrict index}"`
	LAST_UPDATE sq.TimeField   `ddl:"notnull default=CURRENT_TIMESTAMP sqlite:default=unixepoch() onupdatecurrenttimestamp"`
}

type COUNTRY struct {
	sq.TableStruct
	COUNTRY_ID  sq.NumberField `ddl:"primarykey auto_increment identity"`
	COUNTRY     sq.StringField `ddl:"notnull len=50"`
	LAST_UPDATE sq.TimeField   `ddl:"notnull default=CURRENT_TIMESTAMP sqlite:default=unixepoch() onupdatecurrenttimestamp"`
}

type CUSTOMER struct {
	sq.TableStruct
	CUSTOMER_ID sq.NumberField `ddl:"primarykey auto_increment identity"`
	STORE_ID    sq.NumberField `ddl:"notnull references={store onupdate=cascade ondelete=restrict index}"`
	FIRST_NAME  sq.StringField `ddl:"notnull len=45 "`
	LAST_NAME   sq.StringField `ddl:"notnull len=45 index"`
	EMAIL       sq.StringField `ddl:"unique len=50"`
	ADDRESS_ID  sq.NumberField `ddl:"notnull references={address onupdate=cascade ondelete=restrict index}"`
	ACTIVE      sq.NumberField `ddl:"notnull default=TRUE"`
	CREATE_DATE sq.TimeField   `ddl:"notnull default=CURRENT_TIMESTAMP sqlite:default=unixepoch()"`
	LAST_UPDATE sq.TimeField   `ddl:"notnull default=CURRENT_TIMESTAMP sqlite:default=unixepoch() onupdatecurrenttimestamp"`
	_           struct{}       `ddl:"unique=email,first_name,last_name"`
}

type DEPARTMENT struct {
	sq.TableStruct
	DEPARTMENT_ID sq.UUIDField   `ddl:"notnull primarykey"`
	NAME          sq.StringField `ddl:"notnull len=255"`
}

type EMPLOYEE struct {
	sq.TableStruct
	EMPLOYEE_ID sq.UUIDField   `ddl:"notnull primarykey"`
	NAME        sq.StringField `ddl:"notnull len=255"`
	TITLE       sq.StringField `ddl:"notnull len=255"`
	MANAGER_ID  sq.UUIDField   `ddl:"references={employee.employee_id index}"`
}

type EMPLOYEE_DEPARTMENT struct {
	sq.TableStruct `ddl:"primarykey=employee_id,department_id"`
	EMPLOYEE_ID    sq.UUIDField `ddl:"notnull references={employee index}"`
	DEPARTMENT_ID  sq.UUIDField `ddl:"notnull references={department index}"`
}

type FILM struct {
	sq.TableStruct
	FILM_ID              sq.NumberField `ddl:"primarykey auto_increment identity"`
	TITLE                sq.StringField `ddl:"notnull len=255 index"`
	DESCRIPTION          sq.StringField
	RELEASE_YEAR         sq.NumberField `ddl:"postgres:type=year"`
	LANGUAGE_ID          sq.NumberField `ddl:"notnull references={language onupdate=cascade ondelete=restrict index}"`
	ORIGINAL_LANGUAGE_ID sq.NumberField `ddl:"references={language.language_id onupdate=cascade ondelete=restrict index}"`
	RENTAL_DURATION      sq.NumberField `ddl:"notnull default=3"`
	RENTAL_RATE          sq.NumberField `ddl:"type=DECIMAL(4,2) sqlite:type=REAL notnull default=4.99"`
	LENGTH               sq.NumberField
	REPLACEMENT_COST     sq.NumberField `ddl:"type=DECIMAL(5,2) sqlite:type=REAL notnull default=19.99"`
	RATING               sq.StringField `ddl:"postgres:type=mpaa_rating default='G'"`
	SPECIAL_FEATURES     sq.ArrayField
	LAST_UPDATE          sq.TimeField `ddl:"notnull default=CURRENT_TIMESTAMP sqlite:default=unixepoch() onupdatecurrenttimestamp"`
	FULLTEXT             sq.AnyField  `ddl:"dialect=postgres type=TSVECTOR index={. using=gin}"`
}

type FILM_ACTOR struct {
	sq.TableStruct `ddl:"primarykey=actor_id,film_id"`
	ACTOR_ID       sq.NumberField `ddl:"notnull references={actor onupdate=cascade ondelete=restrict}"`
	FILM_ID        sq.NumberField `ddl:"notnull references={film onupdate=cascade ondelete=restrict index}"`
	LAST_UPDATE    sq.TimeField   `ddl:"notnull default=CURRENT_TIMESTAMP sqlite:default=unixepoch() onupdatecurrenttimestamp"`
}

type FILM_CATEGORY struct {
	sq.TableStruct `ddl:"primarykey=film_id,category_id"`
	FILM_ID        sq.NumberField `ddl:"notnull references={film onupdate=cascade ondelete=restrict}"`
	CATEGORY_ID    sq.NumberField `ddl:"notnull references={category onupdate=cascade ondelete=restrict}"`
	LAST_UPDATE    sq.TimeField   `ddl:"notnull default=CURRENT_TIMESTAMP sqlite:default=unixepoch() onupdatecurrenttimestamp"`
}

type FILM_TEXT struct {
	sq.TableStruct `ddl:"virtual dialect=mysql,sqlite"`
	FILM_ID        sq.NumberField `ddl:"dialect=mysql primarykey"`
	TITLE          sq.StringField
	DESCRIPTION    sq.StringField `ddl:"mysql:type=TEXT"`
	FILM_TEXT      sq.AnyField    `ddl:"dialect=sqlite"`
	RANK           sq.NumberField `ddl:"dialect=sqlite"`
	_              struct{}       `ddl:"mysql:index={title,description using=fulltext}"`
}

type INVENTORY struct {
	sq.TableStruct
	INVENTORY_ID sq.NumberField `ddl:"primarykey auto_increment identity"`
	FILM_ID      sq.NumberField `ddl:"notnull references={film onupdate=cascade ondelete=restrict index}"`
	STORE_ID     sq.NumberField `ddl:"notnull references={store onupdate=cascade ondelete=restrict}"`
	LAST_UPDATE  sq.TimeField   `ddl:"notnull default=CURRENT_TIMESTAMP sqlite:default=unixepoch() onupdatecurrenttimestamp"`
	_            struct{}       `ddl:"index=store_id,film_id"`
}

type LANGUAGE struct {
	sq.TableStruct
	LANGUAGE_ID sq.NumberField `ddl:"primarykey auto_increment identity"`
	NAME        sq.StringField `ddl:"notnull len=20"`
	LAST_UPDATE sq.TimeField   `ddl:"notnull default=CURRENT_TIMESTAMP sqlite:default=unixepoch() onupdatecurrenttimestamp"`
}

type PAYMENT struct {
	sq.TableStruct
	PAYMENT_ID   sq.NumberField `ddl:"primarykey auto_increment identity"`
	CUSTOMER_ID  sq.NumberField `ddl:"notnull references={customer onupdate=cascade ondelete=restrict index}"`
	STAFF_ID     sq.NumberField `ddl:"notnull references={staff onupdate=cascade ondelete=restrict index}"`
	RENTAL_ID    sq.NumberField `ddl:"references={rental onupdate=cascade ondelete=setnull index}"`
	AMOUNT       sq.NumberField `ddl:"type=REAL notnull"`
	PAYMENT_DATE sq.TimeField   `ddl:"notnull default=CURRENT_TIMESTAMP sqlite:default=unixepoch()"`
	LAST_UPDATE  sq.TimeField   `ddl:"notnull default=CURRENT_TIMESTAMP sqlite:default=unixepoch() onupdatecurrenttimestamp"`
}

type RENTAL struct {
	sq.TableStruct
	RENTAL_ID    sq.NumberField `ddl:"primarykey auto_increment identity"`
	RENTAL_DATE  sq.TimeField   `ddl:"notnull default=CURRENT_TIMESTAMP sqlite:default=unixepoch() onupdatecurrenttimestamp"`
	INVENTORY_ID sq.NumberField `ddl:"notnull references={inventory onupdate=cascade ondelete=restrict index}"`
	CUSTOMER_ID  sq.NumberField `ddl:"notnull references={customer onupdate=cascade ondelete=restrict index}"`
	RETURN_DATE  sq.TimeField
	STAFF_ID     sq.NumberField `ddl:"notnull references={staff onupdate=cascade ondelete=restrict index}"`
	LAST_UPDATE  sq.TimeField   `ddl:"notnull default=CURRENT_TIMESTAMP sqlite:default=unixepoch() onupdatecurrenttimestamp"`
	_            struct{}       `ddl:"index={inventory_id,customer_id,staff_id unique}"`
}

type STAFF struct {
	sq.TableStruct
	STAFF_ID    sq.NumberField `ddl:"primarykey auto_increment identity"`
	FIRST_NAME  sq.StringField `ddl:"notnull"`
	LAST_NAME   sq.StringField `ddl:"notnull"`
	ADDRESS_ID  sq.NumberField `ddl:"notnull references={address onupdate=cascade ondelete=restrict index}"`
	PICTURE     sq.BinaryField
	EMAIL       sq.StringField `ddl:"unique"`
	STORE_ID    sq.NumberField `ddl:"references={store onupdate=cascade ondelete=restrict index}"`
	ACTIVE      sq.NumberField `ddl:"notnull default=TRUE"`
	USERNAME    sq.StringField `ddl:"notnull"`
	PASSWORD    sq.StringField
	LAST_UPDATE sq.TimeField `ddl:"notnull default=CURRENT_TIMESTAMP sqlite:default=unixepoch() onupdatecurrenttimestamp"`
}

type STORE struct {
	sq.TableStruct
	STORE_ID         sq.NumberField `ddl:"primarykey auto_increment identity"`
	MANAGER_STAFF_ID sq.NumberField `ddl:"notnull references={staff.staff_id onupdate=cascade ondelete=restrict index}"`
	ADDRESS_ID       sq.NumberField `ddl:"notnull references={address onupdate=cascade ondelete=restrict index}"`
	LAST_UPDATE      sq.TimeField   `ddl:"notnull default=CURRENT_TIMESTAMP sqlite:default=unixepoch() onupdatecurrenttimestamp"`
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
