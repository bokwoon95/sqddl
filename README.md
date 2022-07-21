[![GoDoc](https://img.shields.io/badge/pkg.go.dev-ddl-blue)](https://pkg.go.dev/github.com/bokwoon95/sqddl/ddl)
![tests](https://github.com/bokwoon95/sqddl/actions/workflows/tests.yml/badge.svg?branch=main)
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
$ go install -tags=fts5 github.com/bokwoon95/sqddl/sqddl@latest
```

## Subcommands

sqddl has 12 subcommands. Click on each of them to find out more.

- [migrate](https://bokwoon.neocities.org/sqddl.html#migrate) - Run pending migrations and add them to the history table.
- [ls](https://bokwoon.neocities.org/sqddl.html#ls) - Show pending migrations.
- [touch](https://bokwoon.neocities.org/sqddl.html#touch) - Upsert migrations into to the history table. Does not run them.
- [rm](https://bokwoon.neocities.org/sqddl.html#rm) - Remove migrations from the history table.
- [mv](https://bokwoon.neocities.org/sqddl.html#mv) - Rename migrations in the history table.
- [tables](https://bokwoon.neocities.org/sqddl.html#tables) - Generate table structs from database.
- [views](https://bokwoon.neocities.org/sqddl.html#views) - Generate view structs from database.
- [generate](https://bokwoon.neocities.org/sqddl.html#generate) - Generate migrations from a declarative schema (defined as [table structs](https://bokwoon.neocities.org/sqddl.html#table-structs)).
- [wipe](https://bokwoon.neocities.org/sqddl.html#wipe) - Wipe a database of all views, tables, routines, enums, domains and extensions.
- [dump](https://bokwoon.neocities.org/sqddl.html#dump) - Dump the database schema as SQL scripts and data as CSV files.
- [load](https://bokwoon.neocities.org/sqddl.html#load) - Load SQL scripts and CSV files into a database.
- [automigrate](https://bokwoon.neocities.org/sqddl.html#automigrate) - Automatically migrate a database based on a declarative schema (defined as [table structs](https://bokwoon.neocities.org/sqddl.html#table-structs)).

## -db flag

-db is the database url needed to connect to your database. For sqlite this is a file path.

**SQLite examples**

```shell
# <filename>.{sqlite,sqlite3,db,db3}
relative/path/to/file.sqlite
./relative/path/to/file.sqlite3
/absolute/path/to/file.db
file:/absolute/path/to/file.db3

# sqlite:<filepath>
sqlite:relative/path/to/file
sqlite:./relative/path/to/file
sqlite:/absolute/path/to/file
sqlite:file:/absolute/path/to/file
```

**Postgres examples**

```shell
# postgres://<username>:<password>@<host>:<port>/<database>
postgres://user:pass@localhost:5432/sakila
postgres://admin1:Hunter2!@127.0.0.1:5433/mydatabase
```

**MySQL examples**

```shell
# <username>:<password>@tcp(<host>:<port>)/<database>
user:pass@tcp(localhost:3306)/sakila
root:Hunter2!@tcp(127.0.0.1:3307)/mydatabase

# mysql://<username>:<password>@<host>:<port>/<database>
mysql://user:pass@localhost:3306/sakila
mysql://root:Hunter2!@127.0.0.1:3307/mydatabase

# mysql://<username>:<password>@tcp(<host>:<port>)/<database>
mysql://user:pass@tcp(localhost:3306)/sakila
mysql://root:Hunter2!@tcp(127.0.0.1:3307)/mydatabase
```

**SQL Server examples**

```shell
# sqlserver://<username>:<password>@<host>:<port>?database=<database>
sqlserver://user:pass@localhost:1433?database=sakila
sqlserver://sa:Hunter2!@127.0.0.1:1434?database=mydatabase

# sqlserver://<username>:<password>@<host>:<port>/<database>
sqlserver://user:pass@localhost:1433/sakila
sqlserver://sa:Hunter2!@127.0.0.1:1434/mydatabase
```

## Contributing

See [START\_HERE.md](https://github.com/bokwoon95/sqddl/blob/main/ddl/START_HERE.md).
