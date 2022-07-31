# sqddl

## Introduction to sqddl #introduction

Github link: [https://github.com/bokwoon95/sqddl](https://github.com/bokwoon95/sqddl)

sqddl is a zero-configuration database migration tool for Go. It can [generate migrations](#generate) based on a [declarative schema (defined as Go structs)](#table-structs).

Notable features:

- Works on SQLite, Postgres, MySQL and SQL Server.
- No up and down migrations. No migration versions.
    - [Migrations that have not been run will be run, and migrations that have already been run will be skipped](#migrate).
    - That's basically it.
    - How do you roll back to a previous state? [Update your schema definition, generate a new migration from it](#rollback).
- Embed migrations into your application binary (via `//go:embed`) [and run them automatically on startup](#running-embedded-migrations-on-startup).
- [Supports transactional migrations](#transactional-migrations).
    - Multiple migrations can run [in a single transaction](#transactional-migrations).
    - Or a migration can be run [in its own transaction](#tx).
    - Or a migration can explicitly [opt out of transactions](#txoff).
- [Supports undo migrations](#undo-migrations).
    - If a non-transactional migration fails, a corresponding undo migration can optionally be defined to clean up the effects of the failed migration.
- [Supports repeatable migrations](#repeatable-migrations) ([concept taken from Flyway](https://flywaydb.org/documentation/tutorials/repeatable)).
    - Repeatable migrations are migrations that are re-run whenever the contents of their file changes. Useful for views, stored procedures.
- [Generates safe migrations](#safe-migrations) by default.
- [Dump a database](#dump) and [restore it later](#load) as easy test fixtures.

## Installation #installation

```shell
$ go install -tags=fts5 github.com/bokwoon95/sqddl/sqddl@latest
```

## Subcommands #subcommands

sqddl has 12 subcommands. Click on each of them to find out more.

- [migrate](#migrate) - Run pending migrations and add them to the history table.
- [ls](#ls) - Show pending migrations.
- [touch](#touch) - Upsert migrations into to the history table. Does not run them.
- [rm](#rm) - Remove migrations from the history table.
- [mv](#mv) - Rename migrations in the history table.
- [tables](#tables) - Generate table structs from database.
- [views](#views) - Generate view structs from database.
- [generate](#generate) - Generate migrations from a declarative schema (defined as [table structs](#table-structs)).
- [wipe](#wipe) - Wipe a database of all views, tables, routines, enums, domains and extensions.
- [dump](#dump) - Dump the database schema as SQL scripts and CSV files.
- [load](#load) - Load SQL scripts and CSV files into a database.
- [automigrate](#automigrate) - Automatically migrate a database based on a declarative schema (defined as [table structs](#table-structs)).

## Global flags #global-flags

There are 2 global flags that all [subcommands](#subcommands) accept: -db and -history-table. You can pass these flags to the sqddl command and they will get forwarded to the subcommand. This allows you to press up in the shell and backspace a little bit to change the subcommand (reusing the -db and -history-table flags) rather than having to navigate all the way back in the line just to change the subcommand.

```shell
# The two commands below are equivalent
$ sqddl ls -db 'postgres://user:pass@localhost:5432/sakila'
#       ^ subcommand
$ sqddl -db 'postgres://user:pass@localhost:5432/sakila' ls
#                                                        ^ subcommand
```

### -db flag #db-flag

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

### -db flag with file #db-flag-file

If you don't want to include the database URL directly in the command, you can pass in a filename (prefixed with `file:`) where the file contains the database URL.

```shell
# file:<filepath>
file:relative/path/to/file
file:./relative/path/to/file
file:/absolute/path/to/file.txt
```

```shell
$ sqddl migrate -db file:/absolute/path/to/database_url.txt
```

The file should contain the database URL as-is.

```shell
$ cat /absolute/path/to/database_url.txt
postgres://user:pass@localhost:5432/sakila
```

`file:<filepath>` may also reference an SQLite database. The first [16 bytes of the file](https://www.sqlite.org/fileformat.html#the_database_header) are inspected to differentiate between an SQLite database and a plaintext file.

### -history-table flag #history-table-flag

The -history-table flag indicates the name of the history table to be used by the subcommand. By default it is "sqddl\_history". Try to avoid using a custom history table name unless you really have a table name that conflicts with the default.

## migrate #migrate

The migrate [subcommand](#subcommands) runs pending migrations in some directory (specified with -dir). No output means no pending migrations. Once a migration has been run, it will be recorded in a history table so that it doesn't get run again.

Any top-level \*.sql file in the migration directory is considered a migration. You are free to use any naming convention for your migrations, but keep in mind that they will be run in alphabetical order.

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

This is the history table schema in case you are interested.

```sql
CREATE TABLE sqddl_history (
    filename VARCHAR(255) PRIMARY KEY NOT NULL,
    checksum VARCHAR(64),
    started_at DATETIME, -- postgres uses TIMESTAMPTZ, sqlserver uses DATETIMEOFFSET
    time_taken_ns BIGINT,
    success BOOLEAN -- sqlserver uses BIT
);
```

### schema.sql, indexes.sql and constraints.sql #schema-indexes-constraints-sql

The [migrate](#migrate) subcommand ignores any SQL file named "schema.sql", "indexes.sql" or "constraints.sql". Those files are reserved by the [dump](#dump) and [load](#load) subcommands to contain the overall schema definition of the database.

### There are no up and down migrations? How do I rollback to a past working state? #rollback

There are no up and down migrations, nor is there the concept of a "migration version" that you can roll back to. All that matters is whether a migration has or has not been applied. To revert the database schema back to a previous state, you have to create a new migration to undo the changes. Because schema definitions are declarative ([using table structs](#table-structs)), you just have to update your schema definition and let sqddl [generate the new migrations](#generate).

```shell
# revert the file tables/tables.go to its previous working state in commit c5f567
$ git checkout c5f567 -- tables/tables.go

# generate a new migration from tables/tables.go
$ sqddl generate \
    -src postgres://user:pass@localhost:5432/sakila \
    -dest tables/tables.go \
    -output-dir ./migrations
```

### Data migrations are ok #data-migrations

Because migrations are meant to be run only once, it is perfectly fine to include data modifying commands like INSERT or UPDATE inside a migration.

### Cleaning up old migrations #cleaning-old-migrations

Because migrations are meant to be run only once, once they are run you can delete them from the migrations directory. No other ceremony is needed. This keeps the size of the migration directory from growing indefinitely. Don't worry about losing schema history, because sqddl is able to [introspect your database and produce a schema.sql from it](#dump). By regularly updating that schema.sql (and its associated schema.json) in version control you can get track your schema history.

### Running specific migrations #running-specific-migrations

Instead of running all pending migrations, you can run specific migrations by passing them in as arguments.

```shell
# sqddl migrate -db <DATABASE_URL> -dir <MIGRATION_DIR> [FLAGS] [FILENAMES...]
$ sqddl migrate \
    -db 'postgres://user:pass@localhost:5432/sakila' \
    -dir ./migrations \
    02_sakila.sql 04_extras.sql # only run 02_sakila.sql and 04_extras.sql
```

A migration has already been run will not be run again, even if it was explicitly passed in as an argument.

#### Globbing filenames #migrate-glob

You can use globbing to pass in multiple filenames matching a certain pattern. Filenames can optionally be prefixed with the name of the migration directory, allowing you to glob on files in the migration directory.

```shell
# sqddl migrate -db <DATABASE_URL> -dir <MIGRATION_DIR> [FLAGS] [FILENAMES...]
$ sqddl migrate \
    -db 'postgres://user:pass@localhost:5432/sakila' \
    -dir ./migrations \
    ./migrations/2022*.sql # only run migrations that start with 2022
```

### Transactional migrations #transactional-migrations

All migrations are run inside a single transaction until a [\*.tx.sql](#tx) or [\*.txoff.sql](#txoff) migration is encountered. At which point the \*.tx.sql or \*.txoff.sql migrations are run separately, then subsequent migrations will continue running inside a single transaction again following the same logic. The following diagram illustrates the point:

```shell
01.sql       # ┐
02.sql       # ├── transaction 1
03.sql       # ┘
04.tx.sql    # ─── transaction 2
05.tx.sql    # ─── transaction 3
06.sql       # ┐
07.sql       # ├── transaction 4
08.sql       # ┘
09.txoff.sql # ─── no transaction
10.sql       # ┐
11.sql       # ├── transaction 5
12.sql       # ┘
```

MySQL is an exception because it does not support transactional DDL. For MySQL, each migration is run outside a transaction.

#### Explicitly running a migration in a transaction #tx

If a migration filename ends in \*.tx.sq, it will be run inside a new transaction. This applies even to MySQL (for example if you have some INSERT or UPDATE migrations that you want to run inside a transaction).

```shell
01.sql       # ┐
02.sql       # ├── transaction 1
03.sql       # ┘
04.tx.sql    # ─── transaction 2
05.tx.sql    # ─── transaction 3
```

#### Explicitly opting out of a transaction #txoff

If a migration filename ends in \*.txoff.sql, it will be run outside a transaction. This is already the default for MySQL, but you might want to do this for something like Postgres if the migration contains a CREATE INDEX CONCURRENTLY command (which has to be run outside a transaction).

```shell
01.sql       # ┐
02.sql       # ├── transaction 1
03.sql       # ┘
04.txoff.sql # ─── no transaction
05.txoff.sql # ─── no transaction
```

**CAVEAT: If you have multiple SQL statements in a file, this doesn't actually work (for Postgres and SQL Server).** Due to a Postgres and SQL Server driver limitation, if you run more than one statement in an `Exec` the entire thing gets implicitly wrapped in a transaction. The SQLite and MySQL drivers do not have this limitation.

- [https://github.com/amacneil/dbmate/issues/285](https://github.com/amacneil/dbmate/issues/285)
- [https://github.com/golang-migrate/migrate/issues/284](https://github.com/golang-migrate/migrate/issues/284)

As a workaround, if you need to disable transactions for Postgres and SQL Server make sure there is only one statement in the \*.txoff.sql file. In practice this is not really a big deal, as the only time you need to disable transactional DDL is if you run CREATE INDEX CONCURRENTLY in Postgres.

### Undo migrations #undo-migrations

When a non-transactional migration fails, a corresponding undo migration (if one exists) will be called to cleanup the effects of the failed migration. An undo migration is identified by &lt;name&gt;.undo.sql, where &lt;name&gt; is obtained from either &lt;name&gt;.sql or &lt;name&gt;.txoff.sql.

```shell
# This example is for MySQL, which doesn't have transactional migrations and so
# undo migrations must be defined for rollback on failure.
01_init.sql
01_init.undo.sql
02_create_index.txoff.sql
02_create_index.undo.sql
03_misc.sql
03_misc.undo.sql
```

Undo migrations are not down migrations. They are run only when a migration fails. Furthermore they are run only for non-transactional migrations, because if a transactional migration fails its effects will be rolled back cleanly (so no undo migration is needed).

### Repeatable migrations #repeatable-migrations

Any migration inside a special `repeatable` subdirectory is considered a repeatable migration. They are re-run whenever the contents of their file changes (based on a SHA256 checksum stored in the history table). Unlike normal migrations, repeatable migrations are sourced recursively inside the `repeatable` subdirectory. This allows you to organize your repeatable migrations into further subdirectories as you wish.

```shell
01_extensions_types.sql    # ┐
02_sakila.sql              # ├─ migrations
03_webpage.sql             # │
04_extras.sql              # ┘
repeatable/                # ┐
├── functions/             # │
│   ├── film_in_stock.sql  # ├─ repeatable migrations
│   └── rewards_report.sql # │
├── actor_info.sql         # │
└── staff_list.sql         # ┘
```

### Lock timeouts and automatic retries #lock-timeout-retries

By default, an aggressive table lock timeout of 1 second is applied when running migrations. This means if an ALTER TABLE command cannot acquire a lock within 1 second it will fail. That is for your own good, as ALTER TABLE commands are extremely dangerous if they are left waiting for a lock (it will freeze all SQL queries running against the table).

<blockquote><em>(NOTE: an ALTER TABLE is allowed to run for more than 1 second, it is just not allowed to wait for more than 1 second for some other query to finish touching the table)</em></blockquote>

To mitigate this, retryable migrations that time out waiting for a lock are automatically retried (up to 10 times). A migration is considered retryable if it is a [transactional migration](#transactional-migrations) or consists of a single SQL statement (which is naturally transactional). An exponentially-increasing random delay (up to 5 minutes) is inserted between attempts to maximize the chances of successfully acquiring a lock with the minimum number of attempts. If a retryable migration fails for any other reason not due to a lock timeout, it will fail normally and no further retries will be made.

You can increase the timeout duration by supplying a -lock-timeout flag.

```shell
# sqddl migrate -db <DATABASE_URL> -dir <MIGRATION_DIR> [FLAGS] [FILENAMES...]

$ sqddl migrate \
    -db 'postgres://user:pass@localhost:5432/sakila' \
    -dir ./migrations \
    -lock-timeout '10s' # lock timeout duration of 10 seconds

$ sqddl migrate \
    -db 'postgres://user:pass@localhost:5432/sakila' \
    -dir ./migrations \
    -lock-timeout '2m30s' # lock timeout duration of 2 minutes and 30 seconds
```

### Calling migrate from Go code #migrate-cmd

You can call migrate from Go code almost as if you were [calling it from the command line](#migrate).

```go
migrateCmd, err := ddl.MigrateCommand("-db", "postgres://user:pass@localhost:5432/sakila", "-dir", "./migrations")
err = migrateCmd.Run()
```

You can also instantiate the struct fields directly (instead of passing in string arguments).

```go
db, err := sql.Open("postgres", "postgres://user:pass@localhost:5432/sakila")
migrateCmd := &ddl.MigrateCmd{
    Dialect: "postgres",
    DB:      db,
    DirFS:   os.DirFS("./migrations"),
}
err = migrateCmd.Run()
```

Note that ddl library by itself doesn't have [automatic lock timeout retries](#lock-timeout-retries) because the detection of lock timeout errors is driver-specific and the ddl library avoids importing any drivers. That behaviour has to be registered by [importing the following helper packages and calling their Register() function](https://github.com/bokwoon95/sqddl/blob/main/main.go#L18-L23):

- ddlsqlite3.Register() if you use [mattn/go-sqlite3](https://github.com/mattn/go-sqlite3).
- ddlpostgres.Register() if you use [lib/pq](https://github.com/lib/pq).
- ddlpgx.Register() if you use [jackc/pgx](https://github.com/jackc/pgx).
- ddlmysql.Register() if you use [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql).
- ddlsqlserver.Register() if you use [denisenkom/go-mssqldb](https://github.com/denisenkom/go-mssqldb).
- For any other drivers, please call the ddl.Register() function directly (follow the code template inside [ddlpostgres](https://github.com/bokwoon95/sqddl/blob/main/drivers/ddlpostgres/ddlpostgres.go), [ddlpgx](https://github.com/bokwoon95/sqddl/blob/main/drivers/ddlpgx/ddlpgx.go), [ddlmysql](https://github.com/bokwoon95/sqddl/blob/main/drivers/ddlmysql/ddlmysql.go) or [ddlsqlserver](https://github.com/bokwoon95/sqddl/blob/main/drivers/ddlsqlserver/ddlsqlserver.go)).

#### Running migrations from a //go:embed directory on startup #running-embedded-migrations-on-startup

To run embedded migrations (using `//go:embed`) on startup, create a [MigrateCmd](#migrate-cmd) as normal and assign an embed.FS to the DirFS field. The DirFS field accepts anything that implements the fs.FS interface.

```go
//go:embed migrations
var rootDir embed.FS

db, err := sql.Open("postgres", "postgres://user:pass@localhost:5432/sakila")
migrationsDir, err := fs.Sub(rootDir, "migrations")
cmd := &ddl.MigrateCmd{
    Dialect: "postgres",
    DB:      db,
    Dir:     migrationsDir,
}
err = cmd.Run()
```

## ls #ls

The ls [subcommand](#subcommands) shows the pending migrations to be run. No output means no pending migrations.

```shell
# sqddl ls -db <DATABASE_URL> -dir <MIGRATION_DIR> [FLAGS]
$ sqddl ls -db 'postgres://user:pass@localhost:5432/sakila' -dir ./migrations
[pending] 01_extensions_types.sql
[pending] 02_sakila.sql
[pending] 03_webpage.sql
[pending] 04_extras.sql
```

## touch #touch

The touch [subcommand](#subcommands) upserts migrations into the history table without running them.

- The SHA256 checksum for [repeatable migrations](#repeatable-migrations) will be updated.
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

### Why would I want to add migrations to the history table without running them? #why-use-touch

If you are introducing sqddl to an existing project, your database already has a bunch of migrations applied to it which you do not want to run again. So you add them to the history table in order to prevent them from being run. Alternatively, you can just [delete the migrations from the directory](#cleaning-old-migrations).

### Globbing filenames #touch-glob

You can use globbing to pass in multiple filenames matching a certain pattern. Filenames can optionally be prefixed with the name of the migration directory, allowing you to glob on files in the migration directory.

```shell
# sqddl touch -db <DATABASE_URL> -dir <MIGRATION_DIR> [FILENAMES...]
$ sqddl touch \
    -db 'postgres://user:pass@localhost:5432/sakila' \
    -dir ./migrations \
    ./migrations/2022*.sql # add all migrations starting with 2022*
```

## rm #rm

The rm [subcommand](#subcommands) removes migrations from the history table (it does not remove the actual migration files from the directory). This is useful if you accidentally added migrations to the history table using [touch](#touch), or if you want to deregister the migration from the history table so that [migrate](#migrate) will run it again.

```shell
# sqddl rm -db <DATABASE_URL> [FILENAMES...]
$ sqddl rm \
    -db 'postgres://user:pass@localhost:5432/sakila' \
    -dir ./migrations \
    02_sakila.sql 04_extras.sql # remove 02_sakila.sql and 04_extras.sql from the history table
2 rows affected
```

### Globbing filenames #rm-glob

You can use globbing to pass in multiple filenames matching a certain pattern. Filenames can optionally be prefixed with the name of the migration directory, allowing you to glob on files in the migration directory.

```shell
# sqddl rm -db <DATABASE_URL> [FILENAMES...]
$ sqddl rm \
    -db 'postgres://user:pass@localhost:5432/sakila' \
    -dir ./migrations \
    ./migrations/2022*.sql # remove all migrations starting with 2022*
```

## mv #mv

The mv [subcommand](#subcommands) renames migrations in the history table. This is useful if you manually renamed the filename of a migration that was already run (for example a repeatable migration) and you want to update its entry in the history table.

```shell
# sqddl mv -db <DATABASE_URL> <OLD_FILENAME> <NEW_FILENAME>
$ sqddl mv \
    -db 'postgres://user:pass@localhost:5432/sakila' \
    -dir ./migrations \
    old_name.sql new_name.sql # renames old_name.sql to new_name.sql in the history table
1 row affected
```

## tables #tables

The tables [subcommand](#subcommands) generates table structs from the database.

```shell
# sqddl tables -db <DATABASE_URL> [FLAGS]
$ sqddl tables -db 'postgres://user:pass@localhost:5432/sakila' -pkg tables
```

You can include and exclude specific schemas and tables in the output. The [history table](#history-table-flag) (default "sqddl\_history") is always excluded.

```shell
# sqddl tables -db <DATABASE_URL> [FLAGS]

# Include all tables in the 'public' schema.
$ sqddl tables \
    -db 'postgres://user:pass@localhost:5432/sakila' \
    -schemas public

# Include all tables called 'actor' or 'film'.
$ sqddl tables \
    -db 'postgres://user:pass@localhost:5432/sakila' \
    -tables actor,film

# Include the table called 'actor' in the 'public' schema.
$ sqddl tables \
    -db 'postgres://user:pass@localhost:5432/sakila' \
    -schemas public \
    -tables actor

# Exclude all tables in the 'customer1', 'customer2' and 'customer3' schemas.
$ sqddl tables \
    -db 'postgres://user:pass@localhost:5432/sakila' \
    -exclude-schemas customer1,customer2,customer3

# Exclude all tables called 'country', 'city' or 'address'.
$ sqddl tables \
    -db 'postgres://user:pass@localhost:5432/sakila' \
    -exclude-tables country,city,address
```

## views #views

The views [subcommand](#subcommands) generates view structs from the database.

```shell
# sqddl views -db <DATABASE_URL> [FLAGS]
$ sqddl views -db 'postgres://user:pass@localhost:5432/sakila' -pkg tables
```

You can include and exclude specific schemas and views in the output.

```shell
# sqddl views -db <DATABASE_URL> [FLAGS]

# Include all views in the 'public' schema.
$ sqddl views \
    -db 'postgres://user:pass@localhost:5432/sakila' \
    -schemas public

# Include all views called 'actor_info' or 'staff_list'.
$ sqddl views \
    -db 'postgres://user:pass@localhost:5432/sakila' \
    -views actor_info,staff_list

# Include the view called 'actor_info' in the 'public' schema.
$ sqddl views \
    -db 'postgres://user:pass@localhost:5432/sakila' \
    -schemas public \
    -views actor_info

# Exclude all views in the 'customer1', 'customer2' and 'customer3' schemas.
$ sqddl views \
    -db 'postgres://user:pass@localhost:5432/sakila' \
    -exclude-schemas customer1,customer2,customer3

# Exclude all views called 'customer_list', 'film_list' or 'staff_list'.
$ sqddl views \
    -db 'postgres://user:pass@localhost:5432/sakila' \
    -exclude-views customer_list,film_list,staff_list
```

## generate #generate

The generate [subcommand](#subcommands) generates migrations needed to get from a source schema to a destination schema. The source is typically a database URL/DSN ([same as the -db flag](#db-flag)), while the destination is typically a Go source file containing [table structs](#table-structs). No output means no migrations were generated.

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

### Accepted -src and -dest values #src-dest-values

There are three types of values are accepted by the -src and -dest flags.

1. Database URLs/DSNs ([refer to the -db flag](#db-flag)).
    - e.g. `postgres://user:pass@localhost:5432/sakila`
2. Go files (containing [table structs](#table-structs))
    - e.g. `tables/tables.go`
3. JSON files (as created by the [dump command](#dump))
    - e.g. `schema.json`
    - Effectively this lets you store a snapshot of the prod database's schema for the purpose of [generating migrations](#generate) without actually needing to connect to it. Perfect for development!
    - directories and [.zip/.tgz/.tar.gzip archives](#dump-zip-tgz-archive) are also accepted as long as they contain a top-level schema.json file inside
    - e.g. `sakila.zip`, `sakila.tgz`

### DDL limitations #ddl-limitations

The [generate](#generate) subcommand is only able to generate a subset of DDL:

- CREATE SCHEMA
- DROP SCHEMA
- CREATE TABLE
- DROP TABLE
- CREATE INDEX
- DROP INDEX
- ALTER TABLE
    - ADD COLUMN
    - DROP COLUMN
    - ALTER COLUMN
    - ADD CONSTRAINT
    - DROP CONSTRAINT

Any DDL statement not supported here has to be added as a migration manually. CHECK and EXCLUDE constraints are also not supported, you will have to add them manually.

### Safe migrations #safe-migrations

[Generated migrations](#generate) are safe by default i.e. they can be run against a database without blocking normal DML (SELECT, INSERT, UPDATE, DELETE) for too long ([no longer than 1s](#lock-timeout-retries)). If there is anything potentially unsafe, a [warning](#migration-warnings) will be generated.

- SQLite only allows one writer at a time, so most of this is not applicable.
    - Furthermore, SQLite doesn't support ALTER COLUMN or ADD/DROP CONSTRAINT which has to be worked around by building an entirely new table, copying data over, then renaming the new table into the old table. Expect downtime in that case.

- CREATE TABLE is always safe because no existing code could possibly be referencing the table (since it doesn't yet exist). Same for CREATE SCHEMA.

- DROP TABLE is always safe because existing code should no longer be referencing the table (you should have already removed them). Same for DROP SCHEMA.

- CREATE INDEX is always created CONCURRENTLY for Postgres.
    - For MySQL, CREATE INDEX is safe out of the box.
    - For SQL Server, you need to buy the most expensive license they have (Enterprise Edition) in order to CREATE INDEX without locking the table, so that feature has been excluded from the generated output. If you do have the Enterprise Edition, you will have to add `WITH (ONLINE = ON)` yourself.

- DROP INDEX is a fast operation, so setting a [low lock timeout value](#lock-timeout-retries) should be enough to ensure it is safe.

- ALTER TABLE ADD COLUMN is a fast operation, so setting a [low lock timeout value](#lock-timeout-retries) should be enough to ensure it is safe.
    - Adding a column with NOT NULL without providing a default is prohibited. A [warning](#migration-warnings) will be issued.
    - (Postgres 10 and below) Adding a column with DEFAULT or IDENTITY is unsafe. A [warning](#migration-warnings) will be issued.

- ALTER TABLE DROP COLUMN is a fast operation, so setting a [low lock timeout value](#lock-timeout-retries) should be enough to ensure it is safe.

- ALTER TABLE ALTER COLUMN is usually unsafe, if unsafe a [warning will be explicitly printed](#migration-warnings).
    - (Postgres 12+) Adding NOT NULL to an existing column is done by adding a CHECK (column IS NOT NULL) NOT VALID, validating the CHECK constraint in a separate transaction then setting NOT NULL and dropping the constraint ([https://dba.stackexchange.com/a/268128](https://dba.stackexchange.com/a/268128)).

- For ALTER TABLE ADD CONSTRAINT only PRIMARY KEY, FOREIGN KEY and UNIQUE constraints are supported.
    - (Postgres) PRIMARY KEY and UNIQUE constraints are always created by first creating the underlying index CONCURRENTLY, then creating the constraint using that index.
    - (Postgres) FOREIGN KEY constraints are initially created as NOT VALID, then validated in a separate transaction.
    - (MySQL) Adding constraints seems to be safe out of the box.
    - (SQL Server) You will need the Enterprise license ($$) in order to use `WITH (ONLINE = ON)` so it will not be generated. You should add that into the migration yourself if you have the Enterprise Edition.

- ALTER TABLE DROP CONSTRAINT is a fast operation, so setting a [low lock timeout value](#lock-timeout-retries) should be enough to ensure it is safe.

### Migration warnings #migration-warnings

Some of the migrations generated may be unsafe and will generate warnings. You can choose to proceed anyway despite these warnings by passing in the -accept-warnings flag.

```shell
# sqddl generate -src <SRC_SCHEMA> -dest <DEST_SCHEMA> [FLAGS]
$ sqddl generate \
    -src 'postgres://user:pass@localhost:5432/mydatabase' \
    -dest tables/tables.go \
    -output-dir ./migrations \
    -accept-warnings
users: column "user_id" changing type from "TEXT" to "INT" may be unsafe
users: column "email" changing type from "TEXT" to "VARCHAR(255)" is unsafe
users: setting NOT NULL on column "email" is unsafe for large tables. Upgrade to Postgres 12+ to avoid this issue
users: column "bio" decreasing limit from "VARCHAR(1000)" to "VARCHAR(255)" is unsafe
users: column "height_meters" changing scale from "NUMERIC(1,2)" to "NUMERIC(1,4)" is unsafe
users: column "weight_kilos" decreasing precision from "NUMERIC(5,2)" to "NUMERIC(3,2)" is unsafe
users: adding column "is_active" with DEFAULT is unsafe for large tables. Upgrade to Postgres 11+ to avoid this issue. If not, you should add a column without the default first, backfill the default values, then set the column default
./migrations/20060102150405_01_alter_user.tx.sql
./migrations/20060102150405_02_validate_user_not_null.tx.sql
./migrations/20060102150405_03_alter_country.tx.sql
```

### Safe migration references #safe-migration-references

- [https://medium.com/paypal-tech/postgresql-at-scale-database-schema-changes-without-downtime-20d3749ed680](https://medium.com/paypal-tech/postgresql-at-scale-database-schema-changes-without-downtime-20d3749ed680)
- [https://postgres.ai/blog/20210923-zero-downtime-postgres-schema-migrations-lock-timeout-and-retries](https://postgres.ai/blog/20210923-zero-downtime-postgres-schema-migrations-lock-timeout-and-retries)
- [https://www.braintreepayments.com/blog/safe-operations-for-high-volume-postgresql/](https://www.braintreepayments.com/blog/safe-operations-for-high-volume-postgresql/)
- [https://gocardless.com/blog/zero-downtime-postgres-migrations-the-hard-parts/](https://gocardless.com/blog/zero-downtime-postgres-migrations-the-hard-parts/)
- [https://github.com/ankane/strong_migrations](https://github.com/ankane/strong_migrations)
- [https://github.com/fatkodima/online_migrations](https://github.com/fatkodima/online_migrations)
- [https://mydbops.wordpress.com/2020/03/04/an-overview-of-ddl-algorithms-in-mysql-covers-mysql-8/](https://mydbops.wordpress.com/2020/03/04/an-overview-of-ddl-algorithms-in-mysql-covers-mysql-8/)
- [https://www.citusdata.com/blog/2018/02/15/when-postgresql-blocks/](https://www.citusdata.com/blog/2018/02/15/when-postgresql-blocks/)
- [https://medium.com/doctolib/adding-a-not-null-constraint-on-pg-faster-with-minimal-locking-38b2c00c4d1c](https://medium.com/doctolib/adding-a-not-null-constraint-on-pg-faster-with-minimal-locking-38b2c00c4d1c)
- [https://dev.mysql.com/doc/refman/8.0/en/innodb-online-ddl-operations.html](https://dev.mysql.com/doc/refman/8.0/en/innodb-online-ddl-operations.html)
- [https://littlekendra.com/2016/12/15/limiting-downtime-for-schema-changes-dear-sql-dba-episode-25](https://littlekendra.com/2016/12/15/limiting-downtime-for-schema-changes-dear-sql-dba-episode-25)

## wipe #wipe

The wipe [subcommand](#subcommands) wipes a database of all views, tables, routines, enums, domains and extensions.

```shell
# sqddl wipe -db <DATABASE_URL> [FLAGS]
$ sqddl wipe -db 'postgres://user:pass@localhost:5432/sakila'
```

To view the SQL commands first without running them, pass in the -dry-run flag.

```shell
# sqddl wipe -db <DATABASE_URL> [FLAGS]
$ sqddl wipe -db 'postgres://user:pass@localhost:5432/sakila' -dry-run
DROP TABLE IF EXISTS actor CASCADE;

DROP TABLE IF EXISTS address CASCADE;

DROP TABLE IF EXISTS category CASCADE;

DROP TABLE IF EXISTS city CASCADE;

DROP TABLE IF EXISTS country CASCADE;

DROP TABLE IF EXISTS customer CASCADE;

DROP TABLE IF EXISTS data CASCADE;

DROP TABLE IF EXISTS film CASCADE;

DROP TABLE IF EXISTS film_actor CASCADE;

DROP TABLE IF EXISTS film_category CASCADE;

DROP TABLE IF EXISTS inventory CASCADE;

DROP TABLE IF EXISTS language CASCADE;

DROP TABLE IF EXISTS payment CASCADE;

DROP TABLE IF EXISTS rental CASCADE;

DROP TABLE IF EXISTS staff CASCADE;

DROP TABLE IF EXISTS store CASCADE;

DROP TABLE IF EXISTS template CASCADE;

DROP TABLE IF EXISTS webpage CASCADE;

DROP TABLE IF EXISTS webpage_data CASCADE;

DROP TYPE IF EXISTS mpaa_rating CASCADE;

DROP DOMAIN IF EXISTS year CASCADE;

DROP EXTENSION IF EXISTS btree_gist CASCADE;

DROP EXTENSION IF EXISTS "uuid-ossp" CASCADE;
```

### Calling wipe from Go code #wipe-cmd

You can call wipe from Go code almost as if you were [calling it from the command line](#wipe).

```go
wipeCmd, err := ddl.WipeCommand("-db", "postgres://user:pass@localhost:5432/sakila", "-dry-run")
err = wipeCmd.Run()
```

You can also instantiate the struct fields directly (instead of passing in string arguments).

```go
db, err := sql.Open("postgres", "postgres://user:pass@localhost:5432/sakila")
wipeCmd := &ddl.WipeCmd{
    Dialect: "postgres",
    DB:      db,
    DryRun:  true,
}
err = wipeCmd.Run()
```

## dump #dump

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

### Dumping the schema only #dump-schema-only

Pass in the -schema-only flag. Only schema.json, schema.sql, indexes.sql and constraints.sql will be dumped.

```shell
# sqddl dump -db <DATABASE_URL> [FLAGS]
$ sqddl dump -db 'postgres://user:pass@localhost:5432/sakila' -output-dir ./db -schema-only
./db/schema.json
./db/schema.sql
./db/indexes.sql
./db/constraints.sql
```

### Dumping the data only #dump-data-only

Pass in the -data-only flag. Only the CSV files will be dumped.

```shell
# sqddl dump -db <DATABASE_URL> [FLAGS]
$ sqddl dump -db 'postgres://user:pass@localhost:5432/sakila' -output-dir ./db -data-only
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

### Dumping specific tables #dump-tables

Pass in the -tables flag. You can additionally pass in -schemas to restrict it to a particular schema.

```shell
# sqddl dump -db <DATABASE_URL> [FLAGS]
$ sqddl dump \
    -db 'postgres://user:pass@localhost:5432/sakila' \
    -output-dir ./db \
    -tables actor,language,film
./db/actor.csv
./db/language.csv
./db/film.csv
```

### Dumping into a compressed archive #dump-zip-tgz-archive

You may choose to [dump](#dump) the contents directly into a .zip or .tgz archive instead by passing in the -zip or -tgz flag.

```shell
# sqddl dump -db <DATABASE_URL> [FLAGS]

$ sqddl dump -db 'postgres://user:pass@localhost:5432/sakila' -output-dir ./db -zip sakila.zip
./db/sakila.zip

$ sqddl dump -db 'postgres://user:pass@localhost:5432/sakila' -output-dir ./db -tgz sakila.tgz
./db/sakila.tgz
```

The downside is that this will be slower than dumping into a directory because CSV files are written one at a time (as compared to dumping into a directory where CSV files are written concurrently). If you have many tables to dump and you want to compress the result, consider dumping the files into a directory first then manually compressing it.

### Dumping a referentially-intact subset of the database #data-subsetting

You can dump a referentially-intact subset of the database by passing in the -subset flag followed by a query that pulls in the data subset you want. -subset may be passed in multiple times. Each subset query must contain a "`{*}`" as a placeholder for the columns to be selected, and the table being dumped must be wrapped in curly braces e.g. "`{film}`". The table name may optionally be prefixed by the schema e.g. "`{public.film}`".

```shell
# sqddl dump -db <DATABASE_URL> [FLAGS]
$ sqddl dump \
    -db 'postgres://user:pass@localhost:5432/sakila' \
    -output-dir ./db \
    -data-only \
    -subset 'SELECT {*} FROM {film} ORDER BY film_id LIMIT 10' # dump the first 10 films
    -subset 'SELECT {*} FROM {actor}'                          # dump all actors
./db/actor.csv
./db/film.csv
./db/language.csv # language.csv is included because the film table references the language table
```

Rows from other tables will be dumped accordingly to keep the dumped data referentially intact. However because the subsetting algorithm depends on primary keys to deduplicate rows, tables without primary keys will not be dumped. If you want to dump a table that does not have a primary key, it is suggested that you [dump the entire table](#dump-tables) separately (without any subsetting).

A -subset dump only includes the target table and any tables that the target table depends on directly or indirectly. This will likely not involve every table in the database. If you want a single subset query to pull in every table, you should use the -extended-subset flag instead. It includes *any table that directly or indirectly depends on the target table*, followed by the rest of the tables that are directly or indirectly depended on. In practice this means every table that can be joined to the target table in some way will be involved.

```shell
# sqddl dump -db <DATABASE_URL> [FLAGS]
$ sqddl dump \
    -db 'postgres://user:pass@localhost:5432/sakila' \
    -output-dir ./db \
    -data-only \
    -extended-subset 'SELECT {*} FROM {film} ORDER BY film_id LIMIT 10'
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

A current limitation is that your subset queries cannot pull in too many rows, because their values will be materialized into the query used to dump each table. If your subset query pulls in one million rows, there will be one million lines in the resultant query which is likely going to cause it to fail. There is an upcoming feature to add a -temp-table flag that dumps values into temporary tables first, to avoid materializing so many values into the query. That is a work in progress.

## load #load

The load [subcommand](#subcommands) loads SQL scripts and CSV files into a database. It can also load [directories](#dump) and [zip/tar gzip archives](#dump-zip-tgz-archive) created by the [dump](#dump) subcommand.

```shell
# sqddl load -db <DATABASE_URL> [FLAGS] [FILENAMES...]
$ sqddl load \
    -db 'postgres://user:pass@localhost:5432/sakila' \
    ./db/schema.sql ./db/actor.csv ./db/language.csv ./db/indexes.sql ./db/constraints.sql

$ sqddl load -db 'postgres://user:pass@localhost:5432/sakila' ./db

$ sqddl load -db 'postgres://user:pass@localhost:5432/sakila' ./db/sakila.zip

$ sqddl load -db 'postgres://user:pass@localhost:5432/sakila' ./db/sakila.tgz
```

**If the filename passed in is an SQL file**, it will be run as a normal SQL script.

**If the filename passed in is a CSV file**, which table the data will be loaded into depends on the filename.
- If the filename starts with a number followed by an underscore, the number and underscore is discarded. This allows you to control the order of the CSV files being loaded by prefixing them with a number e.g. `01_actor.csv`.
- If there is a dot, everything before the first dot will be the table schema and everthing after the first dot will be the table name (excluding the .csv suffix).
- If there is no dot, the entire filename taken to be the table name (excluding the .csv suffix).

```text
actor.csv           => INSERT INTO actor
01_actor.csv        => INSERT INTO actor
02_country.csv      => INSERT INTO country
03_category.csv     => INSERT INTO category
public.actor.csv    => INSERT INTO public.actor
public.my table.csv => INSERT INTO public."my table"
```

The first line must include the column headers. The CSV format follows the same rules as Go's [csv package](https://pkg.go.dev/encoding/csv):

- Records are delimited by "\n" (LF) or "\r\n" (CRLF).
- Fields are delimited by commas.
- If a field contains commas or newlines, it must be wrapped in double quotes.
    - Strings do not need to be wrapped in double quotes unless they contain commas or newlines.
- If a double quoted field contains double quotes internally, those internal double quotes must be escaped by doubling up on them.

```text
film_id,title,cost,special_features
1,ACADEMY DINOSAUR,20.99,"[""Deleted Scenes"", ""Behind the Scenes""]"
2,ACE GOLDFINGER,12.99,"[""Trailers"", ""Deleted Scenes""]"
3,ADAPTATION HOLES,18.99,"[""Trailers"", ""Deleted Scenes""]"
4,AFFAIR PREJUDICE,26.99,"[""Commentaries"", ""Behind the Scenes""]"
```

If the CSV includes primary key columns, the CSV data will be upserted based on the primary key. So, it is safe to load CSV files containing lines of duplicate data.

**If the filename passed in is a directory or a .zip/.tgz/.tar.gzip archive**, files inside are loaded in a specific order:
- First run schema.sql if it exists.
- Then load all top-level CSV files.
    - If it is a directory or a .zip archive, the CSV files are loaded concurrently.
    - If it is a .tgz/.tar.gzip archive, CSV files are loaded one at a time.
- Then run indexes.sql if it exists.
- Then run constraints.sql if it exists.

### Load data type coercion #load-data-type-coercion

[load](#load) will coerce strings into a more suitable format to fit the column type (where it would otherwise fail).

- (Postgres) Binary literals of form `0x267f4bdb50a041399704c26a16f8f019` (length must be 32) are converted to UUID strings if the column type is UUID. So you can use strings like `0x267f4bdb50a041399704c26a16f8f019` to represent UUIDs across all databases.

- (Postgres) JSON arrays are converted to Postgres arrays if the column type is an array (e.g. TEXT[], INT[], etc). So you can use JSON arrays to represent arrays across all databases.

### Calling load from Go code #load-cmd

You can call load from Go code almost as if you were [calling it from the command line](#load).

```go
loadCmd, err := ddl.LoadCommand(
    "-db", "postgres://user:pass@localhost:5432/sakila",
    "-dir", "./db",
    "schema.sql",
    "actor.csv",
    "language.csv",
    "indexes.sql",
    "constraints.sql",
)
err = loadCmd.Run()
```

You can also instantiate the struct fields directly (instead of passing in string arguments).

```go
db, err := sql.Open("postgres", "postgres://user:pass@localhost:5432/sakila")
loadCmd := &ddl.LoadCmd{
    Dialect:   "postgres",
    DB:        db,
    DirFS:     os.DirFS("./db"),
    Filenames: []string{
        "schema.sql",
        "actor.csv",
        "language.csv",
        "indexes.sql",
        "constraints.sql",
    },
}
err = loadCmd.Run()
```

#### Loading files from a //go:embed directory #loading-embedded-files

To load embedded files (using `//go:embed`), just assign the embed.FS to the DirFS field of the [LoadCmd](#load-cmd) and run it normally. The DirFS field accepts anything that implements the fs.FS interface.

```go
//go:embed db
var rootDir embed.FS

db, err := sql.Open("postgres", "postgres://user:pass@localhost:5432/sakila")
dbDir, err := fs.Sub(rootDir, "db")
loadCmd := &ddl.LoadCmd{
    Dialect:   "postgres",
    DB:        db,
    DirFS:     dbDir,
    Filenames: []string{
        "schema.sql",
        "actor.csv",
        "language.csv",
        "indexes.sql",
        "constraints.sql",
    },
}
err = loadCmd.Run()
```

For greater space savings, you can embed an entire .tgz archive and load that instead. This is particularly useful for test fixtures. Each test fixture can be encapsulated into its own .tgz archive (created from the [dump command](#dump-zip-tgz-archive)) and loaded for each test that spins up a new database instance e.g. an in-memory SQLite database.

```go
//go:embed sakila.tgz
var rootDir embed.FS

db, err := sql.Open("postgres", "postgres://user:pass@localhost:5432/sakila")
loadCmd := &ddl.LoadCmd{
    Dialect:   "postgres",
    DB:        db,
    DirFS:     rootDir,
    Filenames: []string{"sakila.tgz"},
}
err = loadCmd.Run()
```

## automigrate #automigrate

The automigrate [subcommand](#subcommands) automatically migrates a database based on a declarative schema ([defined as table structs](#table-structs)). It is equivalent to running [generate](#generate) followed by [migrate](#migrate), except the generated migrations are created in-memory and will not be added to the history table.

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

## Table structs #table-structs

Your [table structs](#tables) serve as a declarative schema for your tables. Each table struct maps to an SQL table, and [encodes CREATE TABLE information in ddl struct tags](#ddl-struct-tags).

```go
type ACTOR struct {
    sq.TableStruct // sq.TableStruct must be the first field to mark this as a table struct.
    ACTOR_ID    sq.NumberField `ddl:"notnull primarykey identity"`
    FIRST_NAME  sq.StringField `ddl:"type=VARCHAR(45) notnull"`
    LAST_NAME   sq.StringField `ddl:"type=VARCHAR(45) notnull index"`
    LAST_UPDATE sq.TimeField   `ddl:"notnull default=CURRENT_TIMESTAMP"`
}
```

A table struct is meant to be directly usable by [the sq query builder](https://bokwoon.neocities.org/sq.html#table-structs) so no code generation is necessary. You maintain the schema definitions (the structs) by hand and can use them as a source of truth when [generating migrations](#generate).

### Table and column name translation #table-column-name-translation

The table name and column names are translated from the struct name and struct field names by lowercasing them. So a struct `ACTOR` will be translated to a table called `actor`, and a field `ACTOR_ID` will be translated to a column called `actor_id`.

If you wish to use a different naming convention (for example in PascalCase), you can explicitly specify the name inside an sq struct tag.

```go
type ACTOR struct {
    sq.TableStruct `sq:"Actor"`
    ACTOR_ID       sq.NumberField `sq:"ActorID" ddl:"notnull primarykey identity"`
    FIRST_NAME     sq.StringField `sq:"FirstName" ddl:"type=VARCHAR(45) notnull"`
    LAST_NAME      sq.StringField `sq:"LastName" ddl:"type=VARCHAR(45) notnull index"`
    LAST_UPDATE    sq.TimeField   `sq:"LastUpdate" ddl:"notnull default=CURRENT_TIMESTAMP"`
}
```

### Available Field types #field-types

There are 10 available field types that you can use in your [table structs](#table-structs). Each field is associated with a default SQL type which will be used in the CREATE TABLE command if you don't [explicitly mention its type](#type-modifier).

You will need to import the [sq package](https://github.com/bokwoon95/sq) in order to use these fields.

<table>
<tr>
<th>Field</th>
<th>Default SQL Type</th>
</tr>
<tr>
<td><strong>sq.NumberField</strong></td>
<td>INT</td>
</tr>
<tr>
<td><strong>sq.StringField</strong></td>
<td>
<p>SQLite, Postgres - TEXT</p>
<p>MySQL - VARCHAR(255)</p>
<p>SQL Server - NVARCHAR(255)</p>
</td>
</tr>
<tr>
<td><strong>sq.TimeField</strong></td>
<td>
<p>SQLite, MySQL - DATETIME</p>
<p>Postgres - TIMESTAMPTZ</p>
<p>SQL Server - DATETIMEOFFSET</p>
</td>
</tr>
<tr>
<td><strong>sq.BooleanField</strong></td>
<td>
<p>SQLite, Postgres, MySQL - BOOLEAN</p>
<p>SQL Server - BIT</p>
</td>
</tr>
<tr>
<td><strong>sq.BinaryField</strong></td>
<td>
<p>SQLite - BLOB</p>
<p>Postgres - BYTEA</p>
<p>MySQL - MEDIUMBLOB</p>
<p>SQL Server - VARBINARY(MAX)</p>
</td>
</tr>
<tr>
<td><strong>sq.ArrayField</strong></td>
<td>
<p>SQLite, MySQL - JSON</p>
<p>Postgres - TEXT[]</p>
<p>SQL Server - NVARCHAR(MAX)</p>
</td>
</tr>
<tr>
<td><strong>sq.EnumField</strong></td>
<td>
<p>SQLite, Postgres - TEXT</p>
<p>MySQL - VARCHAR(255)</p>
<p>SQL Server - NVARCHAR(255)</p>
</td>
</tr>
<tr>
<td><strong>sq.JSONField</strong></td>
<td>
<p>SQLite, MySQL - JSON</p>
<p>Postgres - JSONB</p>
<p>SQL Server - NVARCHAR(MAX)</p>
</td>
</tr>
<tr>
<td><strong>sq.UUIDField</strong></td>
<td>
<p>SQLite, Postgres - UUID</p>
<p>MySQL, SQL Server - BINARY(16)</p>
</td>
</tr>
<tr>
<td><strong>sq.AnyField</strong></td>
<td>
A catch-all field type that can substitute as any of the 9 other field types.
Use this to represent types like TSVECTOR that don't have a corresponding Field representation.
There is no default SQL type, so a <a href="#type-modifier">type</a> always has to be specified.
</td>
</tr>
</table>

## DDL struct tags #ddl-struct-tags

<blockquote>NOTE: If you already have an existing database, you should <a href="#tables">generate your table structs</a> rather than manually create the table structs and struct tags. That will give you a feel of what kind of struct tag modifiers there are and how to use them.</blockquote>

A `ddl` struct tag consists of one or more modifiers. Modifiers are delimited by spaces.

```go
type ACTOR struct {
    sq.TableStruct
    ACTOR_ID sq.NumberField `ddl:"notnull primarykey identity"`
    //                            └─────┘ └────────┘ └──────┘
    //                           modifier  modifier   modifier
}
```
```sql
CREATE TABLE actor (
    actor_id INT NOT NULL GENERATED BY DEFAULT AS IDENTITY

    ,CONSTRAINT actor_actor_id_pkey PRIMARY KEY (actor_id)
);
```

To see all available modifiers, check out the [modifier list](#modifier-list).

### Modifier values #modifier-values

Modifiers may have values associated with them on the right hand side of an equals '=' sign. No spaces are allowed around the '=' sign, since a space would start a new modifier.
In the example below, the modifier value `DATETIME('now')` has no spaces so no [{brace quoting}](#brace-quoting) is necessary.

```go
type ACTOR struct {
    sq.TableStruct
    //                                modifier value               modifier value
    //                                  ┌───────┐                 ┌─────────────┐
    LAST_UPDATE sq.TimeField `ddl:"type=TIMESTAMP notnull default=DATETIME('now')"`
    //                             └────────────┘ └─────┘ └─────────────────────┘
    //                                modifier    modifier       modifier
}
```
```sql
CREATE TABLE actor (
    last_update TIMESTAMP NOT NULL DEFAULT (DATETIME('now'))
);
```

### Brace quoting #brace-quoting

If a [modifier value](#modifier-values) does contain spaces, the entire value has to be {brace quoted} to ensure it remains a single unit.

```go
type FILM_ACTOR struct {
    sq.TableStruct
    ACTOR_ID    sq.NumberField `ddl:"notnull references=actor.actor_id"`
    //                                             brace quoted because of whitespace
    //                                               ┌────────────────────────────┐
    LAST_UPDATE sq.TimeField   `ddl:"notnull default={DATETIME('now', 'localtime')}"`
    //                               └─────┘ └────────────────────────────────────┘
    //                               modifier               modifier
}
```
```sql
CREATE TABLE film_actor (
    actor_id INT NOT NULL
    ,last_update DATETIME NOT NULL DEFAULT (DATETIME('now', 'localtime'))
    --                                                     ↑
    --                                                 whitespace

    ,CONSTRAINT film_actor_actor_id_fkey FOREIGN KEY (actor_id) REFERENCES actor (actor_id)
);
```

### Submodifiers #submodifiers

A modifier may have additional submodifiers. They are expressed as a [{brace quoted}](#brace-quoting) raw value that contains a value and additional space-delimited submodifiers.

```text
                        <modifier>
┌────────────────────────────────────────────────────────────┐
<name>={<value> <submodifier> <submodifier> <submodifier> ...}
└────┘ └─────────────────────────────────────────────────────┘
<name>                     <raw value>
```

```go
type FILM_ACTOR struct {//                                              modifier
    sq.TableStruct      //                   ┌────────────────────────────────────────────────────────────┐
    //                              modifier │               value        submodifier      submodifier    │
    //                               ┌─────┐ │           ┌────────────┐ ┌──────────────┐ ┌───────────────┐│
    ACTOR_ID    sq.NumberField `ddl:"notnull references={actor.actor_id onupdate=cascade ondelete=restrict}"`
    //                                       └────────┘ └─────────────────────────────────────────────────┘
    //                                          name                         raw value
    LAST_UPDATE sq.TimeField   `ddl:"notnull default=CURRENT_TIMESTAMP"`
    //                                       └─────┘ └───────────────┘
    //                                        name       raw value
}
```
```sql
CREATE TABLE actor (
    actor_id INT NOT NULL
    ,last_update DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP

    ,CONSTRAINT film_actor_actor_id_fkey FOREIGN KEY (actor_id) REFERENCES actor (actor_id) ON UPDATE CASCADE ON DELETE RESTRICT
);
```

### Column-level modifiers and table-level modifiers #column-table-modifiers

Struct tag modifiers defined on one of the [10 field types](#field-types) are considered **column-level modifiers**. Modifiers defined on a column implicitly take that column as their argument, unless explicitly specified otherwise.

```go
type ACTOR struct {
    sq.TableStruct             // implicit: PRIMARY KEY (actor_id)
    ACTOR_ID   sq.NumberField `ddl:"primarykey"`
    FIRST_NAME sq.StringField
}
```

```go
type ACTOR struct {
    sq.TableStruct             // explicit: PRIMARY KEY (first_name)
    ACTOR_ID   sq.NumberField `ddl:"primarykey=first_name"`
    FIRST_NAME sq.StringField
}
```

Struct tag modifiers defined on an `sq.TableStruct` are considered **table-level modifiers**. There is no implicit column attached to the context, so column arguments must always be defined.

```go
type ACTOR struct {
    sq.TableStruct `ddl:"primarykey"` // Error: no column provided
    ACTOR_ID       sq.NumberField
    FIRST_NAME     sq.StringField
}
```

```go
type ACTOR struct {
    sq.TableStruct `ddl:"primarykey=actor_id"` // PRIMARY KEY (actor_id)
    ACTOR_ID       sq.NumberField
    FIRST_NAME     sq.StringField
}
```

Sometimes table-level modifiers can get really long and there's limited space in an `sq.TableStruct` struct tag and struct tags [cannot be broken into multiple lines](https://github.com/golang/go/issues/42182), so as a workaround you can define table-level modifiers on additional unnamed `_` struct fields of `struct{}` type.

```go
type ACTOR struct {
    sq.TableStruct `ddl:"primarykey=actor_id"` // PRIMARY KEY (actor_id)
    ACTOR_ID       sq.NumberField
    FIRST_NAME     sq.StringField
    LAST_NAME      sq.StringField
    LATEST_FILM_ID sq.NumberField
    // CREATE UNIQUE INDEX ON actor (first_name, last_name)
    _ struct{} `ddl:"index={first_name,last_name unique}"`
    // FOREIGN KEY (latest_film_id) REFERENCES film (film_id) ON UPDATE CASCADE ON DELETE RESTRICT DEFERRABLE INITIALLY DEFERRED
    _ struct{} `ddl:"foreignkey={latest_film_id references=film.film_id onupdate=cascade ondelete=restrict deferred}"`
}
```

### Dialect-specific modifiers #dialect-specific-modifiers

A modifier may be prefixed by one or more SQL dialects like this: `<dialect>,<dialect>,...:<modifier>`. This indicates that the modifier is only applicable for those dialects. This is used for defining table structs that use different DDL definitions depending on the dialect.

The currently valid dialect prefixes are: `sqlite`, `postgres`, `mysql` and `sqlserver`.

Modifiers are evaluated left-to-right, and so putting a dialectless modifier at the end will always override any dialect-specific modifier defined earlier.

Some modifiers are already dialect-specific e.g. `auto_increment` only applies to MySQL, `identity` only applies to Postgres and SQL Server. In such cases no dialect prefix is needed, `ddl` will automatically ignore the modifier if it is not applicable for the current dialect.

**Dialect prefix example**

```go
type FILM struct {
    sq.TableStruct
    FILM_ID          sq.NumberField `ddl:"sqlite:type=INTEGER primarykey auto_increment identity"`
    TITLE            sq.StringField `ddl:"mysql,sqlserver:type=VARCHAR(50)"`
    SPECIAL_FEATURES sq.ArrayField  `ddl:"type=JSON postgres:type=TEXT[] sqlserver:type=NVARCHAR(MAX)"`
}
```

The above table struct will generate different CREATE TABLE statements for each dialect.

```sql
-- sqlite
CREATE TABLE film (
    film_id INTEGER PRIMARY KEY
    ,title TEXT
    ,special_features JSON
);
```

```sql
-- postgres
CREATE TABLE film (
    film_id INT GENERATED BY DEFAULT AS IDENTITY
    ,title TEXT
    ,special_features TEXT[]

    ,CONSTRAINT film_film_id_pkey PRIMARY KEY (film_id)
);
```

```sql
-- mysql
CREATE TABLE film (
    film_id INT AUTO_INCREMENT
    ,title VARCHAR(50)
    ,special_features JSON

    ,PRIMARY KEY (film_id)
);
```

```sql
-- sqlserver
CREATE TABLE film (
    film_id INT IDENTITY
    ,title VARCHAR(50)
    ,special_features NVARCHAR(MAX)

    ,CONSTRAINT film_film_id_pkey PRIMARY KEY (film_id)
);
```

## Modifier list #modifier-list

### type #type-modifier

*Column-level modifier.*

Accepts a value representing the column type. The value is literally passed to the database, spaces and all (make sure to use [{brace quoting}](#brace-quoting)).

```go
type FILM struct {
    sq.TableStruct
    TITLE       sq.StringField `ddl:"type=VARCHAR(50)"`
    LAST_UPDATE sq.TimeField   `ddl:"type={TIMESTAMP WITH TIME ZONE}"`
}
```
```sql
CREATE TABLE film (
    title VARCHAR(50)
    ,last_update TIMESTAMP WITH TIME ZONE
);
```

This is also where you can define special types like SERIAL or ENUM.

```go
type FILM struct {
    sq.TableStruct
    FILM_ID     sq.NumberField `ddl:"type=SERIAL primarykey"`
    FILM_RATING sq.StringField `ddl:"type={ENUM('G', 'PG', 'PG-13', 'R', 'NC-17')} default='G'"`
}
```
```sql
CREATE TABLE film (
    film_id SERIAL PRIMARY KEY
    ,film_rating ENUM('G', 'PG', 'PG-13', 'R', 'NC-17') DEFAULT 'G'
);
```

NOTE: if your Postgres version is 10 or higher, [you should not be using SERIAL](https://stackoverflow.com/a/55300741/3030828). Instead, use [`identity`](#identity-modifier).

### len #len-modifier

*Column-level modifier.*

Accepts a value representing the column's character limit. Also sets the column type to VARCHAR (for Postgres and MySQL) or NVARCHAR (for SQL Server). It has no effect on SQLite.

```go
type FILM struct {
    sq.TableStruct
    TITLE       sq.StringField `ddl:"len=50"`
    DESCRIPTION sq.StringField `ddl:"len=5000"`
}
```
```sql
-- Postgres, MySQL
CREATE TABLE film (
    title VARCHAR(50)
    ,description VARCHAR(5000)
);
```
```sql
-- SQL Server
CREATE TABLE film (
    title NVARCHAR(50)
    ,description NVARCHAR(5000)
);
```

### auto_increment #auto_increment-modifier

*Column-level modifier. Only valid for MySQL, ignored otherwise.*

Sets the column to be `AUTO_INCREMENT`.

```go
type FILM struct {
    sq.TableStruct
    FILM_ID sq.NumberField `ddl:"primarykey auto_increment"`
}
```
```sql
-- MySQL
CREATE TABLE film (
    film_id INT AUTO_INCREMENT

    ,PRIMARY KEY (film_id)
);
```

### autoincrement #autoincrement-modifier

*Column-level modifier. Only valid for SQLite, ignored otherwise.*

Sets the column to be `AUTOINCREMENT`.

```go
type FILM struct {
    sq.TableStruct
    FILM_ID sq.NumberField `ddl:"type=INTEGER primarykey autoincrement"`
}
```

```sql
-- SQLite
CREATE TABLE film (
    film_id INTEGER PRIMARY KEY AUTOINCREMENT
);
```

### identity #identity-modifier

*Column-level modifier. Only valid for Postgres or SQL Server, ignored otherwise.*

(Postgres) Sets the column to `GENERATED BY DEFAULT AS IDENTITY`.

(SQL Server) Sets the column to `IDENTITY`.

```go
type FILM struct {
    sq.TableStruct
    FILM_ID sq.NumberField `ddl:"primarykey identity"`
}
```

```sql
-- Postgres
CREATE TABLE film (
    film_id INT GENERATED BY DEFAULT AS IDENTITY

    ,CONSTRAINT film_film_id_pkey PRIMARY KEY (film_id)
);
```

```sql
-- SQL Server
CREATE TABLE film (
    film_id INT IDENTITY

    ,CONSTRAINT film_film_id_pkey PRIMARY KEY (film_id)
);
```

### alwaysidentity #alwaysidentity-modifier

*Column-level modifier. Only valid for Postgres or SQL Server, ignored otherwise.*

(Postgres) Sets the column to `GENERATED ALWAYS AS IDENTITY`.

(SQL Server) Sets the column to `IDENTITY`.

```go
type FILM struct {
    sq.TableStruct
    FILM_ID sq.NumberField `ddl:"primarykey alwaysidentity"`
}
```

```sql
-- Postgres
CREATE TABLE film (
    film_id INT GENERATED ALWAYS AS IDENTITY

    ,CONSTRAINT film_film_id_pkey PRIMARY KEY (film_id)
);
```

```sql
-- SQL Server
CREATE TABLE film (
    film_id INT IDENTITY

    ,CONSTRAINT film_film_id_pkey PRIMARY KEY (film_id)
);
```

### notnull #notnull-modifier

*Column-level modifier.*

Sets the column to be `NOT NULL`.

```go
type ACTOR struct {
    sq.TableStruct
    FIRST_NAME sq.StringField `ddl:"type=VARCHAR(255) notnull"`
}
```

```sql
CREATE TABLE actor (
    first_name VARCHAR(255) NOT NULL
);
```

### onupdatecurrenttimestamp #onupdatecurrenttimestamp-modifier

*Column-level modifier. Only valid for MySQL, ignored otherwise.*

Enables `ON UPDATE CURRENT_TIMESTAMP` for the column.

```go
type ACTOR struct {
    sq.TableStruct
    LAST_UPDATE sq.TimeField `ddl:"default=CURRENT_TIMESTAMP onupdatecurrenttimestamp"`
}
```

```sql
-- MySQL
CREATE TABLE actor (
    last_update DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
```

### collate #collate-modifier

*Column-level modifier.*

Accepts a value representing the column collation.

```go
type FILM_ACTOR_REVIEW struct {
    sq.TableStruct
    REVIEW_BODY sq.StringField `ddl:"collate=C"`
}
```

```sql
CREATE TABLE film_actor_review (
    review_body TEXT COLLATE "C"
);
```

### default #default-modifier

*Column-level modifier.*

Accepts a value representing the column default.

If the column default is anything other than a string, number, `TRUE`, `FALSE`, `CURRENT_DATE`, `CURRENT_TIME` or `CURRENT_TIMESTAMP`, it will be considered an SQL expression. Default SQL expressions are automatically wrapped in brackets (unless the dialect is Postgres).

For SQL Server, `TRUE` and `FALSE` are automatically converted to `1` and `0` so you can use `TRUE` and `FALSE` as default values.

```go
type FILM struct {
    sq.TableStruct
    TITLE       sq.StringField `ddl:"default=''"`
    RENTAL_RATE sq.NumberField `ddl:"type=DECIMAL(4,2) default=4.99"`
    RATING      sq.StringField `ddl:"default='G'"`
    LAST_UPDATE sq.NumberField `ddl:"default=DATETIME('now')"`
}
```

```sql
CREATE TABLE film (
    title TEXT DEFAULT ''
    rental_rate DECIMAL(4,2) DEFAULT 4.99
    rating TEXT DEFAULT 'G'
    last_update DATETIME DEFAULT (DATETIME('now'))
);
```

### generated #generated-modifier

*Column-level modifier.*

Indicates that the column is a generated column.

Defining the generated expression inside the struct tag itself is not supported. You should add generated columns manually via a migration.

```go
type ACTOR struct {
    sq.TableStruct
    ACTOR_ID   sq.NumberField `ddl:"primarykey"`
    FIRST_NAME sq.StringField
    LAST_NAME  sq.StringField
    FULL_NAME  sq.StringField `ddl:"generated"`
}
```

```sql
CREATE TABLE actor (
    actor_id INT PRIMARY KEY
    ,first_name TEXT
    ,last_name TEXT
);
```

Added manually via a migration:

```sql
ALTER TABLE actor ADD COLUMN full_name TEXT GENERATED ALWAYS AS first_name || ' ' || last_name;
```

### dialect #dialect-modifier

*Column-level and table-level modifier.*

Accepts a comma-separated list of dialects. The table or column will only be applicable for those dialects. The dialect value cannot be blank.

```go
type FILM struct {
    sq.TableStruct
    FILM_ID  sq.NumberField `ddl"primarykey"`
    TITLE    sq.StringField `ddl:"len=50"`
    FULLTEXT sq.AnyField    `ddl:"dialect=postgres type=TSVECTOR index={fulltext using=GIN}"`
}

type FILM_TEXT struct {
    sq.TableStruct `ddl:"dialect=mysql"`
    FILM_ID        sq.NumberField
    TITLE          sq.StringField `ddl:"index={title using=FULLTEXT}"`
}
```

```sql
-- Postgres
CREATE TABLE film (
    film_id INT
    ,title VARCHAR(50)
    ,fulltext TSVECTOR

    ,CONSTRAINT film_film_id_pkey PRIMARY KEY (film_id)
);

CREATE INDEX film_fulltext_idx ON film USING gin (fulltext);
```

```sql
-- MySQL
CREATE TABLE film (
    film_id INT
    ,title VARCHAR(50)

    ,PRIMARY KEY (film_id)
);

CREATE TABLE film_text (
    film_id INT
    ,title TEXT
);

CREATE FULLTEXT INDEX film_text_title_idx ON film_text (title);
```

If dialect appears as a column-level modifier, it sets the [dialect prefix](#dialect-specific-modifiers) for the rest of the modifiers on the right (modifiers are evaluated left-to-right).

```go
type FILM struct {
    sq.TableStruct
    FULLTEXT sq.AnyField `ddl:"notnull dialect=postgres type=TSVECTOR index={fulltext using=GIN}"`
    //                                         ^ dialect modifier
}

/* is equivalent to */

type FILM struct {
    sq.TableStruct
    FULLTEXT sq.AnyField `ddl:"notnull postgres:type=TSVECTOR postgres:index={fulltext using=GIN}"`
    //                                 ^ dialect prefix       ^ dialect prefix
}
```

### index #index-modifier

*Column-level and table-level modifier.*

Accepts a value and additional [submodifiers](#submodifiers). The value is the comma-separated list of columns in the index.

```go
type EMPLOYEE_DEPARTMENT struct {
    sq.TableStruct `index=employee_id,department_id`
    EMPLOYEE_ID    sq.NumberField
    DEPARTMENT_ID  sq.NumberField
}
```

```sql
CREATE TABLE employee_department (
    employee_id INT
    ,department_id INT
);

CREATE INDEX employee_department_employee_id_department_id_idx ON employee_department (employee_id, department_id);
```

The value can be omitted if the column being indexed is the same column the struct tag is declared on.

```go
type CUSTOMER struct {
    sq.TableStruct
    EMAIL sq.StringField `ddl:"index"`
}
```

```sql
CREATE TABLE customer (
    email TEXT
);

CREATE INDEX customer_email_idx ON customer (email);
```

Additional [submodifiers](#submodifiers) can be specified after the value, delimited by spaces.

```go
type EMPLOYEE_DEPARTMENT struct {
    sq.TableStruct `index={employee_id,department_id unique}`
    EMPLOYEE_ID    sq.NumberField
    DEPARTMENT_ID  sq.NumberField
}
```

```sql
CREATE TABLE employee_department (
    employee_id INT
    ,department_id INT
);

CREATE UNIQUE INDEX employee_department_employee_id_department_id_idx ON employee_department (employee_id, department_id);
```

If submodifiers are present, the value always has to be specified (or the submodifier will be mistaken as a value).

As a shortcut, a dot '.' can be used to represent the same column the struct tag is declared on.

```go
// WRONG
type CUSTOMER struct {
    sq.TableStruct
    EMAIL sq.StringField `ddl:"index={unique}"` // Error: no such column "unique"
}
```

```go
// RIGHT
type CUSTOMER struct {
    sq.TableStruct
    EMAIL sq.StringField `ddl:"index={email unique}"`
}

/* is equivalent to */

type CUSTOMER struct {
    sq.TableStruct
    EMAIL sq.StringField `ddl:"index={. unique}"`
}
```

```sql
CREATE TABLE customer (
    email TEXT
);

CREATE UNIQUE INDEX customer_email_idx ON customer (email);
```

#### index.unique #index-unique-submodifier

*[`index`](#index-modifier) submodifier.*

Marks the index as `UNIQUE`.

```go
type CUSTOMER struct {
    sq.TableStruct
    EMAIL sq.StringField `ddl:"index={email unique}"`
}
```

```sql
CREATE TABLE customer (
    email TEXT
);

CREATE UNIQUE INDEX customer_email_idx ON customer (email);
```

#### index.using #index-using-submodifier

*[`index`](#index-modifier) submodifier. Only valid for Postgres or MySQL, ignored otherwise.*

Accepts a value representing the index type. Possible values:
- (Postgres) "BTREE", "HASH", "GIN", "GIST", "BRIN", etc
- (MySQL) "BTREE", "HASH", "FULLTEXT", "SPATIAL"

Most of the time you don't have to specify the index type because "BTREE" is the default (which is what you will be using most of the time).

```go
type FILM struct {
    sq.TableStruct `mysql:index={title,description using=FULLTEXT}`
    TITLE          sq.StringField
    DESCRIPTION    sq.StringField
    FULLTEXT       sq.CustomField `ddl:"dialect=postgres type=TSVECTOR index={fulltext using=GIN}"`
}
```

```sql
-- Postgres
CREATE TABLE film (
    title TEXT
    ,description TEXT
    ,fulltext TSVECTOR
);

CREATE INDEX film_fulltext_idx ON film USING GIST (fulltext);
```

```sql
-- MySQL
CREATE TABLE film (
    ,title VARCHAR(255)
    ,description VARCHAR(255)
);

CREATE FULLTEXT INDEX film_title_description_idx ON film (title, description);
```

### primarykey #primarykey-modifier

*Column-level and table-level modifier.*

Accepts a value and additional [submodifiers](#submodifiers). The value is the comma-separated list of columns in the primary key.

```go
type EMPLOYEE_DEPARTMENT struct {
    sq.TableStruct `primarykey=employee_id,department_id`
    EMPLOYEE_ID    sq.NumberField
    DEPARTMENT_ID  sq.NumberField
}
```

```sql
CREATE TABLE employee_department (
    employee_id INT
    ,department_id INT

    ,CONSTRAINT employee_department_employee_id_department_id_pkey PRIMARY KEY (employee_id, department_id)
);
```

The value can be omitted if the primary key column is the same column the struct tag is declared on.

```go
type CUSTOMER struct {
    sq.TableStruct
    EMAIL sq.StringField `ddl:"primarykey"`
}
```

```sql
CREATE TABLE customer (
    email TEXT

    ,CONSTRAINT customer_email_pkey PRIMARY KEY (email)
);
```

Additional [submodifiers](#submodifiers) can be specified after the value, delimited by spaces.

```go
type EMPLOYEE_DEPARTMENT struct {
    sq.TableStruct `primarykey={employee_id,department_id deferrable}`
    EMPLOYEE_ID    sq.NumberField
    DEPARTMENT_ID  sq.NumberField
}
```

```sql
CREATE TABLE employee_department (
    employee_id INT
    ,department_id INT

    ,CONSTRAINT employee_department_employee_id_department_id_pkey PRIMARY KEY (employee_id, department_id) DEFERRABLE
);
```

If submodifiers are present, the value always has to be specified (or the submodifier will be mistaken as a value).

As a shortcut, a dot '.' can be used to represent the same column the struct tag is declared on.

```go
// WRONG
type CUSTOMER struct {
    sq.TableStruct
    EMAIL sq.StringField `ddl:"primarykey={deferrable}"` // Error: no such column "deferrable"
}
```

```go
// RIGHT
type CUSTOMER struct {
    sq.TableStruct
    EMAIL sq.StringField `ddl:"primarykey={email deferrable}"`
}

/* is equivalent to */

type CUSTOMER struct {
    sq.TableStruct
    EMAIL sq.StringField `ddl:"primarykey={. deferrable}"`
}
```

```sql
CREATE TABLE customer (
    email TEXT

    ,CONSTRAINT customer_email_pkey PRIMARY KEY (email) DEFERRABLE
);
```

#### primarykey.deferrable #primarykey-deferrable-submodifier

*[`primarykey`](#primarykey-modifier) submodifier. Only valid for Postgres, ignored otherwise.*

Sets the primary key constraint to `DEFERRABLE`.

```go
type ACTOR struct {
    sq.TableStruct
    ACTOR_ID sq.NumberField `ddl:"primarykey={actor_id deferrable}"`
}
```
```sql
CREATE TABLE actor (
    actor_id INT

    ,CONSTRAINT actor_actor_id_pkey PRIMARY KEY (actor_id) DEFERRABLE
);
```

#### primarykey.deferred #primarykey-deferred-submodifier

*[`primarykey`](#primarykey-modifier) submodifier. Only valid for Postgres, ignored otherwise.*

Sets the primary key constraint to `DEFERRABLE INITIALLY DEFERRED`.

```go
type ACTOR struct {
    sq.TableStruct
    ACTOR_ID sq.NumberField `ddl:"primarykey={actor_id deferred}"`
}
```
```sql
CREATE TABLE actor (
    actor_id INT

    ,CONSTRAINT actor_actor_id_pkey PRIMARY KEY (actor_id) DEFERRABLE INITIALLY DEFERRED
);
```

### references #references-modifier

*Column-level modifier.*

Accepts a value and additional [submodifiers](#submodifiers). The value is the column being referenced by the foreign key and can take one of three forms:

1. `<table>` (if the columns have the same name)
    - e.g. `film`
2. `<table>.<column>`
    - e.g. `film.film_id`
3. `<schema>.<table>.<column>` (if the foreign key points at another schema)
    - e.g. `public.film.film_id`

```go
type FILM_ACTOR struct {
    sq.TableStruct
    FILM_ID       sq.NumberField `ddl:"references=film"`
    ACTOR_ID      sq.NumberField `ddl:"references=actor.actor_id"`
    CHARACTER_ID  sq.NumberField `ddl:"references=schema1.characters.character_id"`
}
```

```sql
CREATE TABLE film_actor (
    film_id INT
    ,actor_id INT
    ,character_id INT

    ,CONSTRAINT film_actor_film_id_fkey FOREIGN KEY (film_id) REFERENCES film (film_id)
    ,CONSTRAINT film_actor_actor_id_fkey FOREIGN KEY (actor_id) REFERENCES actor (actor_id)
    ,CONSTRAINT film_actor_character_id_fkey FOREIGN KEY (character_id) REFERENCES schema1.characters (character_id)
);
```

The `references` modifier only supports foreign keys containing one column. To support foreign keys containing multiple columns, use the [`foreignkey`](#foreignkey-modifier) modifier instead.

#### references.index #references-index-submodifier

*[`references`](#references-modifier) submodifier.*

Creates an index on the foreign key column.

```go
type FILM struct {
    sq.TableStruct
    LANGUAGE_ID sq.NumberField `ddl:"references={language.language_id index}"`
}
```

```sql
CREATE TABLE film (
    language_id INT

    ,CONSTRAINT film_language_id_fkey FOREIGN KEY (language_id) REFERENCES language (language_id)
);

CREATE INDEX film_language_id_idx ON film (language_id);
```

#### references.onupdate #references-onupdate-submodifier

*[`references`](#references-modifier) submodifier.*

Accepts a value for the `ON UPDATE` action of the foreign key. Possible values: `cascade`, `restrict`, `noaction`, `setnull` or `setdefault`.

```go
type FILM struct {
    sq.TableStruct
    LANGUAGE_ID sq.NumberField `ddl:"references={language.language_id onupdate=cascade}"`
}
```

```sql
CREATE TABLE film (
    language_id INT

    ,CONSTRAINT film_language_id_fkey FOREIGN KEY (language_id) REFERENCES language (language_id) ON UPDATE CASCADE
);
```

#### references.ondelete #references-ondelete-submodifier

*[`references`](#references-modifier) submodifier.*

Accepts a value for the `ON DELETE` action of the foreign key. Possible values: `cascade`, `restrict`, `noaction`, `setnull` or `setdefault`.

```go
type FILM struct {
    sq.TableStruct
    LANGUAGE_ID sq.NumberField `ddl:"references={language.language_id ondelete=restrict}"`
}
```

```sql
CREATE TABLE film (
    language_id INT

    ,CONSTRAINT film_language_id_fkey FOREIGN KEY (language_id) REFERENCES language (language_id) ON DELETE RESTRICT
);
```

#### references.deferrable #references-deferrable-submodifier

*[`references`](#references-modifier) submodifier. Only valid for SQLite or Postgres, ignored otherwise.*

Sets the foreign key constraint to `DEFERRABLE`.

```go
type FILM struct {
    sq.TableStruct
    LANGUAGE_ID sq.NumberField `ddl:"references={language.language_id deferrable}"`
}
```

```sql
CREATE TABLE film (
    language_id INT

    ,CONSTRAINT film_language_id_fkey FOREIGN KEY (language_id) REFERENCES language (language_id) DEFERRABLE
);
```

#### references.deferred #references-deferred-submodifier

*[`references`](#references-modifier) submodifier. Only valid for SQLite or Postgres, ignored otherwise.*

Sets the foreign key constraint to `DEFERRABLE INITIALLY DEFERRED`.

```go
type FILM struct {
    sq.TableStruct
    LANGUAGE_ID sq.NumberField `ddl:"references={language.language_id deferred}"`
}
```

```sql
CREATE TABLE film (
    language_id INT

    ,CONSTRAINT film_language_id_fkey FOREIGN KEY (language_id) REFERENCES language (language_id) DEFERRABLE INITIALLY DEFERRED
);
```

### foreignkey #foreignkey-modifier

*Table-level modifier.*

Accepts a value and additional [submodifiers](#submodifiers). The value is the comma-separated list of columns in the foreign key. The `references` submodifier value must always be provided (its format is the same as the [`references`](#references-modifier) modifier).

```go
type FILM_ACTOR struct {
    sq.TableStruct
    FILM_ID       sq.NumberField `ddl:"foreignkey={film_id references=film}"`
    ACTOR_ID      sq.NumberField `ddl:"foreignkey={actor_id references=actor.actor_id}"`
    CHARACTER_ID  sq.NumberField `ddl:"foreignkey={character_id references=schema1.characters.character_id}"`
}
```

```sql
CREATE TABLE film_actor (
    film_id INT
    ,actor_id INT
    ,character_id INT

    ,CONSTRAINT film_actor_film_id_fkey FOREIGN KEY (film_id) REFERENCES film (film_id)
    ,CONSTRAINT film_actor_actor_id_fkey FOREIGN KEY (actor_id) REFERENCES actor (actor_id)
    ,CONSTRAINT film_actor_character_id_fkey FOREIGN KEY (character_id) REFERENCES schema1.characters (character_id)
);
```

Here is how to define a foreign key with multiple columns.

```go
type TASK struct {
    sq.TableStruct
    EMPLOYEE_ID   sq.NumberField
    DEPARTMENT_ID sq.NumberField

    _ struct{} `ddl:"foreignkey={employee_id,department_id references=employee_department.employee_id,department_id onupdate=cascade}"`
}

/* is equivalent to */

type TASK struct {
    sq.TableStruct
    EMPLOYEE_ID   sq.NumberField
    DEPARTMENT_ID sq.NumberField

    _ struct{} `ddl:"foreignkey={employee_id,department_id references=employee_department onupdate=cascade}"`
}
```

```sql
CREATE TABLE task (
    employee_id INT
    ,department_id INT

    ,CONSTRAINT task_employee_id_department_id_fkey FOREIGN KEY (employee_id, department_id) REFERENCES employee_department (employee_id, department_id) ON UPDATE CASCADE
);
```

#### foreignkey.references #foreignkey-references-submodifier

*[`foreignkey`](#foreignkey-modifier) submodifier.*

Accepts a value representing the column(s) being referenced by the foreign key. Must always be provided for the `foreignkey` modifier. Refer to the [`foreignkey`](#foreignkey-modifier) modifier for an example.

#### foreignkey.index #foreignkey-index-submodifier

*[`foreignkey`](#foreignkey-modifier) submodifier.*

Creates an index on the foreign key column(s).

```go
type TASK struct {
    sq.TableStruct
    EMPLOYEE_ID   sq.NumberField
    DEPARTMENT_ID sq.NumberField

    _ struct{} `ddl:"foreignkey={employee_id,department_id references=employee_department index}"`
}
```

```sql
CREATE TABLE task (
    employee_id INT
    ,department_id INT

    ,CONSTRAINT task_employee_id_department_id_fkey FOREIGN KEY (employee_id, department_id) REFERENCES employee_department (employee_id, department_id)
);

CREATE INDEX task_employee_id_department_id_idx ON task (employee_id, department_id);
```

#### foreignkey.onupdate #foreignkey-onupdate-submodifier

*[`foreignkey`](#foreignkey-modifier) submodifier.*

Accepts a value for the `ON UPDATE` action of the foreign key. Possible values: `cascade`, `restrict`, `noaction`, `setnull` or `setdefault`.

```go
type FILM struct {
    sq.TableStruct
    LANGUAGE_ID sq.NumberField `ddl:"foreignkey={language_id references=language.language_id onupdate=cascade}"`
}
```

```sql
CREATE TABLE film (
    language_id INT

    ,CONSTRAINT film_language_id_fkey FOREIGN KEY (language_id) REFERENCES language (language_id) ON UPDATE CASCADE
);
```

#### foreignkey.ondelete #foreignkey-ondelete-submodifier

*[`foreignkey`](#foreignkey-modifier) submodifier.*

Accepts a value for the `ON DELETE` action of the foreign key. Possible values: `cascade`, `restrict`, `noaction`, `setnull` or `setdefault`.

```go
type FILM struct {
    sq.TableStruct
    LANGUAGE_ID sq.NumberField `ddl:"foreignkey={language_id references=language.language_id ondelete=restrict}"`
}
```

```sql
CREATE TABLE film (
    language_id INT

    ,CONSTRAINT film_language_id_fkey FOREIGN KEY (language_id) REFERENCES language (language_id) ON DELETE RESTRICT
);
```

#### foreignkey.deferrable #foreignkey-deferrable-submodifier

*[`foreignkey`](#foreignkey-modifier) submodifier. Only valid for SQLite or Postgres, ignored otherwise.*

Sets the foreign key constraint to `DEFERRABLE`.

```go
type FILM struct {
    sq.TableStruct
    LANGUAGE_ID sq.NumberField `ddl:"foreignkey={language_id references=language.language_id deferrable}"`
}
```

```sql
CREATE TABLE film (
    language_id INT

    ,CONSTRAINT film_language_id_fkey FOREIGN KEY (language_id) REFERENCES language (language_id) DEFERRABLE
);
```

#### foreignkey.deferred #foreignkey-deferred-submodifier

*[`foreignkey`](#foreignkey-modifier) submodifier. Only valid for SQLite or Postgres, ignored otherwise.*

Sets the foreign key constraint to `DEFERRABLE INITIALLY DEFERRED`.

```go
type FILM struct {
    sq.TableStruct
    LANGUAGE_ID sq.NumberField `ddl:"foreignkey={language_id references=language.language_id deferred}"`
}
```

```sql
CREATE TABLE film (
    language_id INT

    ,CONSTRAINT film_language_id_fkey FOREIGN KEY (language_id) REFERENCES language (language_id) DEFERRABLE INITIALLY DEFERRED
);
```

### unique #unique-modifier

*Column-level and table-level modifier.*

Accepts a value and additional [submodifiers](#submodifiers). The value is the comma-separated list of columns in the unique constraint.

```go
type EMPLOYEE_DEPARTMENT struct {
    sq.TableStruct `unique=employee_id,department_id`
    EMPLOYEE_ID    sq.NumberField
    DEPARTMENT_ID  sq.NumberField
}
```

```sql
CREATE TABLE employee_department (
    employee_id INT
    ,department_id INT

    ,CONSTRAINT employee_department_employee_id_department_id_key UNIQUE (employee_id, department_id)
);
```

The value can be omitted if the primary key column is the same column the struct tag is declared on.

```go
type CUSTOMER struct {
    sq.TableStruct
    EMAIL sq.StringField `ddl:"unique"`
}
```

```sql
CREATE TABLE customer (
    email TEXT

    ,CONSTRAINT customer_email_key UNIQUE (email)
);
```

Additional [submodifiers](#submodifiers) can be specified after the value, delimited by spaces.

```go
type EMPLOYEE_DEPARTMENT struct {
    sq.TableStruct `unique={employee_id,department_id deferrable}`
    EMPLOYEE_ID    sq.NumberField
    DEPARTMENT_ID  sq.NumberField
}
```

```sql
CREATE TABLE employee_department (
    employee_id INT
    ,department_id INT

    ,CONSTRAINT employee_department_employee_id_department_id_key UNIQUE (employee_id, department_id) DEFERRABLE
);
```

If submodifiers are present, the value always has to be specified (or the submodifier will be mistaken as a value).

As a shortcut, a dot '.' can be used to represent the same column the struct tag is declared on.

```go
// WRONG
type CUSTOMER struct {
    sq.TableStruct
    EMAIL sq.StringField `ddl:"unique={deferrable}"` // Error: no such column "deferrable"
}
```

```go
// RIGHT
type CUSTOMER struct {
    sq.TableStruct
    EMAIL sq.StringField `ddl:"unique={email deferrable}"`
}

/* is equivalent to */

type CUSTOMER struct {
    sq.TableStruct
    EMAIL sq.StringField `ddl:"unique={. deferrable}"`
}
```

```sql
CREATE TABLE customer (
    email TEXT

    ,CONSTRAINT customer_email_pkey PRIMARY KEY (email) DEFERRABLE
);
```

#### unique.deferrable #unique-deferrable-submodifier

*[`unique`](#unique-modifier) submodifier. Only valid for Postgres, ignored otherwise.*

Sets the unique constraint to `DEFERRABLE`.

```go
type ACTOR struct {
    sq.TableStruct
    ACTOR_ID sq.NumberField `ddl:"unique={actor_id deferrable}"`
}
```
```sql
CREATE TABLE actor (
    actor_id INT

    ,CONSTRAINT actor_actor_id_key UNIQUE (actor_id) DEFERRABLE
);
```

#### unique.deferred #unique-deferred-submodifier

*[`unique`](#unique-modifier) submodifier. Only valid for Postgres, ignored otherwise.*

Sets the unique constraint to `DEFERRABLE INITIALLY DEFERRED`.

```go
type ACTOR struct {
    sq.TableStruct
    ACTOR_ID sq.NumberField `ddl:"unique={actor_id deferrable}"`
}
```
```sql
CREATE TABLE actor (
    actor_id INT

    ,CONSTRAINT actor_actor_id_key UNIQUE (actor_id) DEFERRABLE
);
```
