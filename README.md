[![GoDoc](https://img.shields.io/badge/pkg.go.dev-ddl-blue)](https://pkg.go.dev/github.com/bokwoon95/sqddl/ddl)
[![tests](https://github.com/bokwoon95/sqddl/actions/workflows/tests.yml/badge.svg?branch=main)](https://github.com/bokwoon95/sqddl/actions/workflows/tests.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/bokwoon95/sqddl)](https://goreportcard.com/report/github.com/bokwoon95/sqddl)
[![Coverage Status](https://coveralls.io/repos/github/bokwoon95/sqddl/badge.svg?branch=main)](https://coveralls.io/github/bokwoon95/sqddl?branch=main)

# sqddl

[one-page documentation](https://bokwoon.neocities.org/sqddl.html)

sqddl is a zero-configuration database migration tool for Go. It can [generate migrations](https://bokwoon.neocities.org/sqddl.html#generate) based on a [declarative schema (defined as Go structs)](https://bokwoon.neocities.org/sqddl.html#table-structs).

Notable features:

- Works on SQLite, Postgres, MySQL and SQL Server.
- No up and down migrations. No migration versions.
    - [Migrations that have not been run will be run, and migrations that have already been run will be skipped](https://bokwoon.neocities.org/sqddl.html#migrate).
    - That's basically it.
    - How do you roll back to a previous state? [Update your schema definition, generate a new migration from it](https://bokwoon.neocities.org/sqddl.html#rollback).
- Embed migrations into your application binary (via `//go:embed`) [and run them automatically on startup](https://bokwoon.neocities.org/sqddl.html#running-embedded-migrations-on-startup).
- [Supports transactional migrations](https://bokwoon.neocities.org/sqddl.html#transactional-migrations).
    - Multiple migrations can run [in a single transaction](https://bokwoon.neocities.org/sqddl.html#transactional-migrations).
    - Or a migration can be run [in its own transaction](https://bokwoon.neocities.org/sqddl.html#tx).
    - Or a migration can explicitly [opt out of transactions](https://bokwoon.neocities.org/sqddl.html#txoff).
- [Supports undo migrations](https://bokwoon.neocities.org/sqddl.html#undo-migrations).
    - If a non-transactional migration fails, a corresponding undo migration can optionally be defined to clean up the effects of the failed migration.
- [Supports repeatable migrations](https://bokwoon.neocities.org/sqddl.html#repeatable-migrations) ([concept taken from Flyway](https://flywaydb.org/documentation/tutorials/repeatable)).
    - Repeatable migrations are migrations that are re-run whenever the contents of their file changes. Useful for views, stored procedures.
- [Generates safe migrations](https://bokwoon.neocities.org/sqddl.html#safe-migrations) by default.
- [Dump a database](https://bokwoon.neocities.org/sqddl.html#dump) and [restore it later](https://bokwoon.neocities.org/sqddl.html#load) as easy test fixtures.

## Installation

```shell
$ go install -tags=fts5 github.com/bokwoon95/sqddl@latest
```

## Example migrations.

To view what a sample migration directory would look like, click the following links:

- [sqlite\_migrations](https://github.com/bokwoon95/sqddl/tree/main/ddl/sqlite_migrations)
- [postgres\_migrations](https://github.com/bokwoon95/sqddl/tree/main/ddl/postgres_migrations)
- [mysql\_migrations](https://github.com/bokwoon95/sqddl/tree/main/ddl/mysql_migrations)
- [sqlserver\_migrations](https://github.com/bokwoon95/sqddl/tree/main/ddl/sqlserver_migrations)

## Subcommands

sqddl has 12 subcommands. Click on each of them to find out more.

- [migrate](#migrate) - Run pending migrations and add them to the [history table](#history-table).
- [ls](#ls) - Show pending migrations.
- [touch](#touch) - Upsert migrations into to the [history table](#history-table). Does not run them.
- [rm](#rm) - Remove migrations from the [history table](#history-table).
- [mv](#mv) - Rename migrations in the [history table](#history-table).
- [tables](#tables) - Generate table structs from database.
- [views](#views) - Generate view structs from database.
- [generate](#generate) - Generate migrations from a declarative schema (defined as [table structs](https://bokwoon.neocities.org/sqddl.html#table-structs)).
- [wipe](#wipe) - Wipe a database of all views, tables, routines, enums, domains and extensions.
- [dump](#dump) - Dump the database schema as SQL scripts and data as CSV files.
- [load](#load) - Load SQL scripts and CSV files into a database.
- [automigrate](#automigrate) - Automatically migrate a database based on a declarative schema (defined as [table structs](https://bokwoon.neocities.org/sqddl.html#table-structs)).

## History table

The history table stores the history of the applied migrations. The default name of the table is "sqddl\_history". You can override it with the -history-table flag, but try not to do so unless you really have a table name that conflicts with the default.

This is the schema for the history table.

```sql
CREATE TABLE sqddl_history (
    filename VARCHAR(255) PRIMARY KEY NOT NULL,
    checksum VARCHAR(64),
    started_at DATETIME, -- postgres uses TIMESTAMPTZ, sqlserver uses DATETIMEOFFSET
    time_taken_ns BIGINT,
    success BOOLEAN -- sqlserver uses BIT
);
```

## migrate

Docs: [https://bokwoon.neocities.org/sqddl.html#migrate](https://bokwoon.neocities.org/sqddl.html#migrate).

The migrate [subcommand](#subcommands) runs pending migrations in some directory (specified with -dir). No output means no pending migrations. Once a migration has been run, it will be recorded in a [history table](#history-table) so that it doesn't get run again.

Any top-level \*.sql file in the migration directory is considered a migration. You are free to use any naming convention for your migrations, but keep in mind that they will be run in alphabetical order.

Check out [https://bokwoon.neocities.org/sqddl.html#db-flag](https://bokwoon.neocities.org/sqddl.html#db-flag) for more -db flag examples.

```shell
# sqddl migrate -db <DATABASE_URL> -dir <MIGRATION_DIR> [FLAGS] [FILENAMES...]
$ sqddl migrate -db 'postgres://user:pass@localhost:5432/sakila' -dir ./migrations
BEGIN
[OK] 01_extensions_types.sql (22.4397ms)
[OK] 02_sakila.sql (194.1385ms)
[OK] 03_webpage.sql (29.5218ms)
[OK] 04_extras.sql (20.1678ms)
COMMIT
```

## ls

Docs: [https://bokwoon.neocities.org/sqddl.html#ls](https://bokwoon.neocities.org/sqddl.html#ls).

The ls [subcommand](#subcommands) shows the pending migrations to be run. No output means no pending migrations.

```shell
# sqddl ls -db <DATABASE_URL> -dir <MIGRATION_DIR> [FLAGS]
$ sqddl ls -db 'postgres://user:pass@localhost:5432/sakila' -dir ./migrations
[pending] 01_extensions_types.sql
[pending] 02_sakila.sql
[pending] 03_webpage.sql
[pending] 04_extras.sql
```

## touch

Docs: [https://bokwoon.neocities.org/sqddl.html#touch](https://bokwoon.neocities.org/sqddl.html#touch).

The touch [subcommand](#subcommands) upserts migrations into the [history table](#history-table) without running them.

- The SHA256 checksum for [repeatable migrations](https://bokwoon.neocities.org/sqddl.html#repeatable-migrations) will be updated.
- started\_at will be set to the current time.
- time\_taken\_ns will be set to 0.

```shell
# sqddl touch -db <DATABASE_URL> -dir <MIGRATION_DIR> [FILENAMES...]
$ sqddl touch \
    -db 'postgres://user:pass@localhost:5432/sakila' \
    -dir ./migrations \
    02_sakila.sql 04_extras.sql # add 02_sakila.sql and 04_extras.sql to the history table without running them
2 rows affected
```

## rm

Docs: [https://bokwoon.neocities.org/sqddl.html#rm](https://bokwoon.neocities.org/sqddl.html#rm).

The rm [subcommand](#subcommands) removes migrations from the [history table](#history-table) (it does not remove the actual migration files from the directory). This is useful if you accidentally added migrations to the history table using [touch](#touch), or if you want to deregister the migration from the history table so that [migrate](#migrate) will run it again.

```shell
# sqddl rm -db <DATABASE_URL> [FILENAMES...]
$ sqddl rm \
    -db 'postgres://user:pass@localhost:5432/sakila' \
    -dir ./migrations \
    02_sakila.sql 04_extras.sql # remove 02_sakila.sql and 04_extras.sql from the history table
2 rows affected
```

## mv

Docs: [https://bokwoon.neocities.org/sqddl.html#mv](https://bokwoon.neocities.org/sqddl.html#mv).

The mv [subcommand](#subcommands) renames migrations in the [history table](#history-table). This is useful if you manually renamed the filename of a migration that was already run (for example a repeatable migration) and you want to update its entry in the history table.

```shell
# sqddl mv -db <DATABASE_URL> <OLD_FILENAME> <NEW_FILENAME>
$ sqddl mv \
    -db 'postgres://user:pass@localhost:5432/sakila' \
    -dir ./migrations \
    old_name.sql new_name.sql # renames old_name.sql to new_name.sql in the history table
1 row affected
```

## tables

Docs: [https://bokwoon.neocities.org/sqddl.html#tables](https://bokwoon.neocities.org/sqddl.html#tables).

The tables [subcommand](#subcommands) generates table structs from the database.

```shell
# sqddl tables -db <DATABASE_URL> [FLAGS]
$ sqddl tables -db 'postgres://user:pass@localhost:5432/sakila' -pkg tables -file tables/tables.go
```

## views

Docs: [https://bokwoon.neocities.org/sqddl.html#views](https://bokwoon.neocities.org/sqddl.html#views).

The views [subcommand](#subcommands) generates view structs from the database.

```shell
# sqddl views -db <DATABASE_URL> [FLAGS]
$ sqddl views -db 'postgres://user:pass@localhost:5432/sakila' -pkg tables -file tables/views.go
```

## generate

Docs: [https://bokwoon.neocities.org/sqddl.html#generate](https://bokwoon.neocities.org/sqddl.html#generate).

The generate [subcommand](#subcommands) generates migrations needed to get from a source schema to a destination schema. The source is typically a database URL/DSN ([same as the -db flag](https://bokwoon.neocities.org/sqddl.html#db-flag)), while the destination is typically a Go source file containing [table structs](https://bokwoon.neocities.org/sqddl.html#table-structs). No output means no migrations were generated.

```shell
# sqddl generate -src <SRC_SCHEMA> -dest <DEST_SCHEMA> [FLAGS]
$ sqddl generate \
    -src 'postgres://user:pass@localhost:5432/mydatabase' \
    -dest tables/tables.go \
    -output-dir ./migrations
./migrations/20060102150405_01_schemas.sql
./migrations/20060102150405_02_tables.sql
./migrations/20060102150405_03_add_person_country_fkeys.tx.sql
```

## wipe

Docs: [https://bokwoon.neocities.org/sqddl.html#wipe](https://bokwoon.neocities.org/sqddl.html#wipe).

The wipe [subcommand](#subcommands) wipes a database of all views, tables, routines, enums, domains and extensions.

```shell
# sqddl wipe -db <DATABASE_URL> [FLAGS]
$ sqddl wipe -db 'postgres://user:pass@localhost:5432/sakila'
```

## dump

Docs: [https://bokwoon.neocities.org/sqddl.html#dump](https://bokwoon.neocities.org/sqddl.html#dump).

The dump [subcommand](#subcommands) can dump a database's schema and data.

- The schema is dumped as 4 files:
    - schema.json
    - schema.sql
    - indexes.sql
    - constraints.sql
- The data is dumped as a CSV file per table.
    - e.g. if the table is called `actor`, the CSV file will be called `actor.csv`.

```shell
# sqddl dump -db <DATABASE_URL> [FLAGS]
$ sqddl dump -db 'postgres://user:pass@localhost:5432/sakila' -output-dir ./db
./db/schema.json
./db/schema.sql
./db/indexes.sql
./db/constraints.sql
./db/actor.csv
./db/address.csv
./db/category.csv
./db/city.csv
./db/country.csv
./db/customer.csv
./db/data.csv
./db/film.csv
./db/film_actor.csv
./db/film_category.csv
./db/inventory.csv
./db/language.csv
./db/payment.csv
./db/rental.csv
./db/staff.csv
./db/store.csv
```

## load

Docs: [https://bokwoon.neocities.org/sqddl.html#load](https://bokwoon.neocities.org/sqddl.html#load).

The load [subcommand](#subcommands) loads SQL scripts and CSV files into a database. It can also load [directories](https://bokwoon.neocities.org/sqddl.html#dump) and [zip/tar gzip archives](https://bokwoon.neocities.org/sqddl.html#dump-zip-tgz-archive) created by the [dump](#dump) subcommand.

```shell
# sqddl load -db <DATABASE_URL> [FLAGS] [FILENAMES...]
$ sqddl load \
    -db 'postgres://user:pass@localhost:5432/sakila' \
    ./db/schema.sql ./db/actor.csv ./db/language.csv ./db/indexes.sql ./db/constraints.sql

$ sqddl load -db 'postgres://user:pass@localhost:5432/sakila' ./db

$ sqddl load -db 'postgres://user:pass@localhost:5432/sakila' ./db/sakila.zip

$ sqddl load -db 'postgres://user:pass@localhost:5432/sakila' ./db/sakila.tgz
```

## automigrate

Docs: [https://bokwoon.neocities.org/sqddl.html#automigrate](https://bokwoon.neocities.org/sqddl.html#automigrate).

The automigrate [subcommand](#subcommands) automatically migrates a database based on a declarative schema ([defined as table structs](https://bokwoon.neocities.org/sqddl.html#table-structs)). It is equivalent to running [generate](#generate) followed by [migrate](#migrate), except the generated migrations are created in-memory and will not be added to the [history table](#history-table).

```shell
# sqddl automigrate -db <DATABASE_URL> -dest <DEST_SCHEMA> [FLAGS]
$ sqddl automigrate -db 'postgres://user:pass@localhost:5432/sakila' -dest tables/tables.go
BEGIN
[OK] automigrate_01_schemas.sql (604.834µs)
[OK] automigrate_02_tables.sql (6.896833ms)
COMMIT
BEGIN
[OK] automigrate_03_add_person_country_fkeys.tx.sql (1.40075ms)
COMMIT
```

## Contributing

See [START\_HERE.md](https://github.com/bokwoon95/sqddl/blob/main/ddl/START_HERE.md).
