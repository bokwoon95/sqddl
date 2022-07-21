package _

import "github.com/bokwoon95/sq"

type CATEGORY struct {
	sq.TableStruct `sq:"public.category"`
	CATEGORY_ID    sq.NumberField `ddl:"primarykey identity"`
	CATEGORY       sq.StringField `ddl:"unique"`
}

type MOVIE struct {
	sq.TableStruct `sq:"public.movie"`
	MOVIE_ID       sq.NumberField `ddl:"primarykey identity"`
	TITLE          sq.StringField `ddl:"unique"`
	CATEGORY       sq.StringField `ddl:"references=public.category.category index"`
	SUBCATEGORY    sq.StringField `ddl:"references=public.category.category index"`
	METADATA       sq.JSONField
}
