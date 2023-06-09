package _

import "github.com/bokwoon95/sq"

type PUBLIC_ACTOR struct {
	sq.TableStruct
	ACTOR_ID           sq.NumberField `ddl:"type=int notnull primarykey identity"`
	FIRST_NAME         sq.StringField `ddl:"type=varchar(45) notnull"`
	LAST_NAME          sq.StringField `ddl:"type=varchar(45) notnull index"`
	FULL_NAME          sq.StringField `ddl:"type=text generated"`
	FULL_NAME_REVERSED sq.StringField `ddl:"type=text generated"`
	LAST_UPDATE        sq.TimeField   `ddl:"type=timestamptz notnull default=CURRENT_TIMESTAMP"`
}

type PUBLIC_ADDRESS struct {
	sq.TableStruct
	ADDRESS_ID  sq.NumberField `ddl:"type=int notnull primarykey identity"`
	ADDRESS     sq.StringField `ddl:"type=varchar(50) notnull"`
	ADDRESS2    sq.StringField `ddl:"type=varchar(50)"`
	DISTRICT    sq.StringField `ddl:"type=varchar(20) notnull"`
	CITY_ID     sq.NumberField `ddl:"type=int notnull references={city onupdate=cascade ondelete=restrict deferrable index}"`
	POSTAL_CODE sq.StringField `ddl:"type=varchar(10)"`
	PHONE       sq.StringField `ddl:"type=text notnull"`
	LAST_UPDATE sq.TimeField   `ddl:"type=timestamptz notnull default=CURRENT_TIMESTAMP"`
}

type PUBLIC_CATEGORY struct {
	sq.TableStruct
	CATEGORY_ID sq.NumberField `ddl:"type=int notnull primarykey identity"`
	NAME        sq.StringField `ddl:"type=varchar(45) notnull"`
	LAST_UPDATE sq.TimeField   `ddl:"type=timestamptz notnull default=CURRENT_TIMESTAMP"`
}

type PUBLIC_CITY struct {
	sq.TableStruct
	CITY_ID     sq.NumberField `ddl:"type=int notnull primarykey identity"`
	CITY        sq.StringField `ddl:"type=varchar(50) notnull"`
	COUNTRY_ID  sq.NumberField `ddl:"type=int notnull references={country onupdate=cascade ondelete=restrict deferrable index}"`
	LAST_UPDATE sq.TimeField   `ddl:"type=timestamptz notnull default=CURRENT_TIMESTAMP"`
}

type PUBLIC_COUNTRY struct {
	sq.TableStruct
	COUNTRY_ID  sq.NumberField `ddl:"type=int notnull primarykey identity"`
	COUNTRY     sq.StringField `ddl:"type=varchar(50) notnull"`
	LAST_UPDATE sq.TimeField   `ddl:"type=timestamptz notnull default=CURRENT_TIMESTAMP"`
}

type PUBLIC_CUSTOMER struct {
	sq.TableStruct
	CUSTOMER_ID sq.NumberField  `ddl:"type=int notnull primarykey identity"`
	STORE_ID    sq.NumberField  `ddl:"type=int notnull references={store onupdate=cascade ondelete=restrict deferrable index}"`
	FIRST_NAME  sq.StringField  `ddl:"type=varchar(45) notnull"`
	LAST_NAME   sq.StringField  `ddl:"type=varchar(45) notnull index"`
	EMAIL       sq.StringField  `ddl:"type=varchar(50) unique"`
	ADDRESS_ID  sq.NumberField  `ddl:"type=int notnull references={address onupdate=cascade ondelete=restrict deferrable index}"`
	ACTIVE      sq.BooleanField `ddl:"type=boolean notnull default=true"`
	CREATE_DATE sq.TimeField    `ddl:"type=timestamptz notnull default=CURRENT_TIMESTAMP"`
	LAST_UPDATE sq.TimeField    `ddl:"type=timestamptz notnull default=CURRENT_TIMESTAMP"`
	_           struct{}        `ddl:"unique=email,first_name,last_name"`
}

type PUBLIC_DEPARTMENT struct {
	sq.TableStruct
	DEPARTMENT_ID sq.UUIDField   `ddl:"type=uuid notnull primarykey"`
	NAME          sq.StringField `ddl:"type=varchar(255) notnull"`
}

type PUBLIC_EMPLOYEE struct {
	sq.TableStruct
	EMPLOYEE_ID sq.UUIDField   `ddl:"type=uuid notnull primarykey"`
	NAME        sq.StringField `ddl:"type=varchar(255) notnull"`
	TITLE       sq.StringField `ddl:"type=varchar(255) notnull"`
	MANAGER_ID  sq.UUIDField   `ddl:"type=uuid references={employee.employee_id index}"`
}

type PUBLIC_EMPLOYEE_DEPARTMENT struct {
	sq.TableStruct `ddl:"primarykey=employee_id,department_id"`
	EMPLOYEE_ID    sq.UUIDField `ddl:"type=uuid notnull references={employee index}"`
	DEPARTMENT_ID  sq.UUIDField `ddl:"type=uuid notnull references={department index}"`
}

type PUBLIC_FILM struct {
	sq.TableStruct
	FILM_ID              sq.NumberField `ddl:"type=int notnull primarykey identity"`
	TITLE                sq.StringField `ddl:"type=text notnull index"`
	DESCRIPTION          sq.StringField `ddl:"type=text"`
	RELEASE_YEAR         sq.NumberField `ddl:"type=year"`
	LANGUAGE_ID          sq.NumberField `ddl:"type=int notnull references={language onupdate=cascade ondelete=restrict deferrable index}"`
	ORIGINAL_LANGUAGE_ID sq.NumberField `ddl:"type=int references={language.language_id onupdate=cascade ondelete=restrict deferrable index}"`
	RENTAL_DURATION      sq.NumberField `ddl:"type=int notnull default=3"`
	RENTAL_RATE          sq.NumberField `ddl:"type=numeric(4,2) notnull default=4.99"`
	LENGTH               sq.NumberField `ddl:"type=int"`
	REPLACEMENT_COST     sq.NumberField `ddl:"type=numeric(5,2) notnull default=19.99"`
	RATING               sq.EnumField   `ddl:"type=mpaa_rating default='G'::mpaa_rating"`
	SPECIAL_FEATURES     sq.ArrayField  `ddl:"type=text[]"`
	LAST_UPDATE          sq.TimeField   `ddl:"type=timestamptz notnull default=CURRENT_TIMESTAMP"`
	FULLTEXT             sq.AnyField    `ddl:"type=tsvector index={. using=gin}"`
}

type PUBLIC_FILM_ACTOR struct {
	sq.TableStruct `ddl:"primarykey=actor_id,film_id"`
	ACTOR_ID       sq.NumberField `ddl:"type=int notnull references={actor onupdate=cascade ondelete=restrict deferrable}"`
	FILM_ID        sq.NumberField `ddl:"type=int notnull references={film onupdate=cascade ondelete=restrict deferrable index}"`
	LAST_UPDATE    sq.TimeField   `ddl:"type=timestamptz notnull default=CURRENT_TIMESTAMP"`
}

type PUBLIC_FILM_CATEGORY struct {
	sq.TableStruct `ddl:"primarykey=film_id,category_id"`
	FILM_ID        sq.NumberField `ddl:"type=int notnull references={film onupdate=cascade ondelete=restrict deferrable}"`
	CATEGORY_ID    sq.NumberField `ddl:"type=int notnull references={category onupdate=cascade ondelete=restrict deferrable}"`
	LAST_UPDATE    sq.TimeField   `ddl:"type=timestamptz notnull default=CURRENT_TIMESTAMP"`
}

type PUBLIC_INVENTORY struct {
	sq.TableStruct
	INVENTORY_ID sq.NumberField `ddl:"type=int notnull primarykey identity"`
	FILM_ID      sq.NumberField `ddl:"type=int notnull references={film onupdate=cascade ondelete=restrict deferrable index}"`
	STORE_ID     sq.NumberField `ddl:"type=int notnull references={store onupdate=cascade ondelete=restrict deferrable}"`
	LAST_UPDATE  sq.TimeField   `ddl:"type=timestamptz notnull default=CURRENT_TIMESTAMP"`
	_            struct{}       `ddl:"index=store_id,film_id"`
}

type PUBLIC_LANGUAGE struct {
	sq.TableStruct
	LANGUAGE_ID sq.NumberField `ddl:"type=int notnull primarykey identity"`
	NAME        sq.StringField `ddl:"type=text notnull"`
	LAST_UPDATE sq.TimeField   `ddl:"type=timestamptz notnull default=CURRENT_TIMESTAMP"`
}

type PUBLIC_PAYMENT struct {
	sq.TableStruct
	PAYMENT_ID   sq.NumberField `ddl:"type=int notnull primarykey identity"`
	CUSTOMER_ID  sq.NumberField `ddl:"type=int notnull references={customer onupdate=cascade ondelete=restrict deferrable index}"`
	STAFF_ID     sq.NumberField `ddl:"type=int notnull references={staff onupdate=cascade ondelete=restrict deferrable index}"`
	RENTAL_ID    sq.NumberField `ddl:"type=int references={rental onupdate=cascade ondelete=setnull deferrable index}"`
	AMOUNT       sq.NumberField `ddl:"type=numeric(5,2) notnull"`
	PAYMENT_DATE sq.TimeField   `ddl:"type=timestamptz notnull default=CURRENT_TIMESTAMP"`
	LAST_UPDATE  sq.TimeField   `ddl:"type=timestamptz notnull default=CURRENT_TIMESTAMP"`
}

type PUBLIC_RENTAL struct {
	sq.TableStruct
	RENTAL_ID    sq.NumberField `ddl:"type=int notnull primarykey identity"`
	RENTAL_DATE  sq.TimeField   `ddl:"type=timestamptz notnull default=CURRENT_TIMESTAMP"`
	INVENTORY_ID sq.NumberField `ddl:"type=int notnull references={inventory onupdate=cascade ondelete=restrict deferrable index}"`
	CUSTOMER_ID  sq.NumberField `ddl:"type=int notnull references={customer onupdate=cascade ondelete=restrict deferrable index}"`
	RETURN_DATE  sq.TimeField   `ddl:"type=timestamptz"`
	STAFF_ID     sq.NumberField `ddl:"type=int notnull references={staff onupdate=cascade ondelete=restrict deferrable index}"`
	LAST_UPDATE  sq.TimeField   `ddl:"type=timestamptz notnull default=CURRENT_TIMESTAMP"`
	_            struct{}       `ddl:"index={inventory_id,customer_id,staff_id unique}"`
}

type PUBLIC_STAFF struct {
	sq.TableStruct
	STAFF_ID    sq.NumberField  `ddl:"type=int notnull primarykey identity"`
	FIRST_NAME  sq.StringField  `ddl:"type=varchar(45) notnull"`
	LAST_NAME   sq.StringField  `ddl:"type=varchar(45) notnull"`
	ADDRESS_ID  sq.NumberField  `ddl:"type=int notnull references={address onupdate=cascade ondelete=restrict deferrable index}"`
	PICTURE     sq.BinaryField  `ddl:"type=bytea"`
	EMAIL       sq.StringField  `ddl:"type=varchar(50) unique"`
	STORE_ID    sq.NumberField  `ddl:"type=int references={store deferrable index}"`
	ACTIVE      sq.BooleanField `ddl:"type=boolean notnull default=true"`
	USERNAME    sq.StringField  `ddl:"type=varchar(16) notnull"`
	PASSWORD    sq.StringField  `ddl:"type=varchar(40)"`
	LAST_UPDATE sq.TimeField    `ddl:"type=timestamptz notnull default=CURRENT_TIMESTAMP"`
}

type PUBLIC_STORE struct {
	sq.TableStruct
	STORE_ID         sq.NumberField `ddl:"type=int notnull primarykey identity"`
	MANAGER_STAFF_ID sq.NumberField `ddl:"type=int notnull references={staff.staff_id onupdate=cascade ondelete=restrict deferrable index}"`
	ADDRESS_ID       sq.NumberField `ddl:"type=int notnull references={address onupdate=cascade ondelete=restrict deferrable index}"`
	LAST_UPDATE      sq.TimeField   `ddl:"type=timestamptz notnull default=CURRENT_TIMESTAMP"`
}

type PUBLIC_TASK struct {
	sq.TableStruct
	TASK_ID       sq.UUIDField   `ddl:"type=uuid notnull primarykey"`
	EMPLOYEE_ID   sq.UUIDField   `ddl:"type=uuid notnull"`
	DEPARTMENT_ID sq.UUIDField   `ddl:"type=uuid notnull"`
	TASK          sq.StringField `ddl:"type=varchar(255) notnull"`
	DATA          sq.JSONField   `ddl:"type=jsonb"`
	_             struct{}       `ddl:"foreignkey={employee_id,department_id references=employee_department index}"`
}
