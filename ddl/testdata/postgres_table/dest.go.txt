package _

import "github.com/bokwoon95/sq"

type ACTORS struct {
	sq.TableStruct
	ACTOR_ID sq.NumberField `ddl:"primarykey identity"`
	NAME     sq.StringField
}

type MOVIES struct {
	sq.TableStruct
	MOVIE_ID sq.NumberField `ddl:"primarykey identity"`
	TITLE    sq.StringField `ddl:"index"`
	SYNOPSIS sq.StringField
}

type MOVIE_AWARDS struct {
	sq.TableStruct
	MOVIE_ID                sq.NumberField `ddl:"references=movies.movie_id"`
	BEST_ACTOR              sq.NumberField `ddl:"references=actors.actor_id"`
	BEST_SUPPORTING_ACTOR   sq.NumberField `ddl:"references=actors.actor_id"`
	BEST_ACTRESS            sq.NumberField `ddl:"references=actors.actor_id"`
	BEST_SUPPORTING_ACTRESS sq.NumberField `ddl:"references=actors.actor_id"`
}
