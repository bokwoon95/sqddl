package _

import "github.com/bokwoon95/sq"

type ACTOR struct {
	sq.TableStruct
	ACTOR_ID sq.NumberField `ddl:"primarykey"`
	NAME     sq.StringField
}

type MOVIE struct {
	sq.TableStruct
	MOVIE_ID sq.NumberField `ddl:"primarykey"`
	TITLE    sq.StringField `ddl:"index"`
	SYNOPSIS sq.StringField
}

type MOVIE_AWARD struct {
	sq.TableStruct
	MOVIE_ID                sq.NumberField `ddl:"references=movie.movie_id"`
	BEST_ACTOR              sq.NumberField `ddl:"references=actor.actor_id"`
	BEST_SUPPORTING_ACTOR   sq.NumberField `ddl:"references=actor.actor_id"`
	BEST_ACTRESS            sq.NumberField `ddl:"references=actor.actor_id"`
	BEST_SUPPORTING_ACTRESS sq.NumberField `ddl:"references=actor.actor_id"`
}
