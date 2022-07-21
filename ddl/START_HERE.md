This document describes how the codebase is organized. It is meant for people who are contributing to the codebase (or are just casually browsing).

Files are written in such a way that **each successive file in the list below only depends on files that come before it**. This self-enforced restriction makes deep architectural changes trivial because you can essentially blow away the entire codebase and rewrite it from scratch file-by-file, complete with working tests every step of the way. Please adhere to this file order when submitting pull requests.

- [**ddl.go**](https://github.com/bokwoon95/sqddl/blob/main/ddl/ddl.go)
    - Core data types: Catalog, Schema, Enum, Domain, Routine, View, Table, Column, Constraint, Index and Trigger.
    - Misc utility functions.
- [**catalog_cache.go**](https://github.com/bokwoon95/sqddl/blob/main/ddl/catalog_cache.go)
    - CatalogCache is used for querying and modifying a Catalog's nested objects.
- [**database_introspector.go**](https://github.com/bokwoon95/sqddl/blob/main/ddl/database_introspector.go)
    - DatabaseIntrospector is used to introspect a database.
- [**batch_insert.go**](https://github.com/bokwoon95/sqddl/blob/main/ddl/batch_insert.go)
    - BatchInsert is used to insert data into a table in batches.
- [**load_cmd.go**](https://github.com/bokwoon95/sqddl/blob/main/ddl/load_cmd.go)
    - [`sqddl load`](https://bokwoon.neocities.org/sqddl.html#load)
- [**wipe_cmd.go**](https://github.com/bokwoon95/sqddl/blob/main/ddl/wipe_cmd.go)
    - [`sqddl wipe`](https://bokwoon.neocities.org/sqddl.html#wipe)
- [**touch_cmd.go**](https://github.com/bokwoon95/sqddl/blob/main/ddl/touch_cmd.go)
    - [`sqddl touch`](https://bokwoon.neocities.org/sqddl.html#touch)
- [**migrate_cmd.go**](https://github.com/bokwoon95/sqddl/blob/main/ddl/migrate_cmd.go)
    - [`sqddl migrate`](https://bokwoon.neocities.org/sqddl.html#migrate)
- [**ls_cmd.go**](https://github.com/bokwoon95/sqddl/blob/main/ddl/ls_cmd.go)
    - [`sqddl ls`](https://bokwoon.neocities.org/sqddl.html#ls)
- [**rm_cmd.go**](https://github.com/bokwoon95/sqddl/blob/main/ddl/rm_cmd.go)
    - [`sqddl rm`](https://bokwoon.neocities.org/sqddl.html#rm)
- [**mv_cmd.go**](https://github.com/bokwoon95/sqddl/blob/main/ddl/mv_cmd.go)
    - [`sqddl mv`](https://bokwoon.neocities.org/sqddl.html#mv)
- [**modifier.go**](https://github.com/bokwoon95/sqddl/blob/main/ddl/modifier.go)
    - Modifier represents a modifier in a [ddl struct tag](https://bokwoon.neocities.org/sqddl.html#ddl-struct-tags).
- [**table_structs.go**](https://github.com/bokwoon95/sqddl/blob/main/ddl/table_structs.go)
    - TableStructs are used to represent [table structs](https://bokwoon.neocities.org/sqddl.html#table-structs).
- [**tables_cmd.go**](https://github.com/bokwoon95/sqddl/blob/main/ddl/tables_cmd.go)
    - [`sqddl tables`](https://bokwoon.neocities.org/sqddl.html#tables)
- [**views_cmd.go**](https://github.com/bokwoon95/sqddl/blob/main/ddl/views_cmd.go)
    - [`sqddl views`](https://bokwoon.neocities.org/sqddl.html#views)
- [**struct_parser.go**](https://github.com/bokwoon95/sqddl/blob/main/ddl/struct_parser.go)
    - StructParser is used to parse Go source code into TableStructs.
- [**sqlite_migration.go**](https://github.com/bokwoon95/sqddl/blob/main/ddl/sqlite_migration.go)
    - Code for generating SQLite migrations.
- [**postgres_migration.go**](https://github.com/bokwoon95/sqddl/blob/main/ddl/postgres_migration.go)
    - Code for generating Postgres migrations.
- [**mysql_migration.go**](https://github.com/bokwoon95/sqddl/blob/main/ddl/mysql_migration.go)
    - Code for generating MySQL migrations.
- [**sqlserver_migration.go**](https://github.com/bokwoon95/sqddl/blob/main/ddl/sqlserver_migration.go)
    - Code for generating SQL Server migrations.
- [**subsetter.go**](https://github.com/bokwoon95/sqddl/blob/main/ddl/subsetter.go)
    - Subsetter is used to dump a referentially-intact subset of the database.
- [**dump_cmd.go**](https://github.com/bokwoon95/sqddl/blob/main/ddl/dump_cmd.go)
    - [`sqddl dump`](https://bokwoon.neocities.org/sqddl.html#dump)
- [**generate_cmd.go**](https://github.com/bokwoon95/sqddl/blob/main/ddl/generate_cmd.go)
    - [`sqddl generate`](https://bokwoon.neocities.org/sqddl.html#generate)
- [**automigrate_cmd.go**](https://github.com/bokwoon95/sqddl/blob/main/ddl/automigrate_cmd.go)
    - [`sqddl automigrate`](https://bokwoon.neocities.org/sqddl.html#automigrate)
- [**integration_test.go**](https://github.com/bokwoon95/sqddl/blob/main/ddl/integration_test.go)
    - Tests that interact with a live database i.e. SQLite, Postgres, MySQL and SQL Server.
- [**golden_files_test.go**](https://github.com/bokwoon95/sqddl/blob/main/ddl/golden_files_test.go)
    - Code for generating golden files (files which test output is compared against).

## Testing

Add tests if you add code.

To run tests, use:

```shell
$ go test ./ddl -tags=fts5 # -failfast -shuffle=on -coverprofile=coverage
```

There are tests that require a live database connection. They will only run if you provide the corresponding database URL in the test flags:

```shell
$ go test ./ddl -tags=fts5 -postgres $POSTGRES_URL -mysql $MYSQL_URL -sqlserver $SQLSERVER_URL # -failfast -shuffle=on -coverprofile=coverage
```

If you have docker, you can use the docker-compose.yml (run `docker-compose up`) at the root of this project's directory to spin up Postgres, MySQL and SQL Server databases that are reachable at the following URLs:

```shell
# docker-compose up -d
POSTGRES_URL='postgres://user1:Hunter2!@localhost:5456/sakila?sslmode=disable'
MYSQL_URL='root:Hunter2!@tcp(localhost:3330)/sakila?multiStatements=true&parseTime=true'
SQLSERVER_URL='sqlserver://sa:Hunter2!@localhost:1447'
```

## Golden files

There are certain files in the `ddl/testdata` directory that are designated "golden files", files which test output is compared against. If you make a change that affects test output, you may find it easier to regenerate the golden files rather than modifying the golden files directly by hand. Pass in the `-generate-golden-files` test flag to regenerate golden files:

```shell
$ go test ./ddl -tags=fts5 -generate-golden-files -postgres $POSTGRES_URL -mysql $MYSQL_URL -sqlserver $SQLSERVER_URL # -failfast -shuffle=on -coverprofile=coverage
```

## Documentation

Documentation is contained entirely within [sqddl.md](https://github.com/bokwoon95/sqddl/blob/main/sqddl.md) in the project root directory. You can view the output at [https://bokwoon.neocities.org/sqddl.html](https://bokwoon.neocities.org/sqddl.html). The documentation is regenerated everytime a new commit is pushed to the main branch, so to change the documentation just change sqddl.md and submit a pull request.

You can preview the output of sqddl.md locally by installing [github.com/bokwoon95/mddocs](https://github.com/bokwoon95/mddocs) and running it with sqddl.md as the argument.

```shell
$ go install github/bokwoon95/mddocs@latest
$ mddocs
Usage:
mddocs project.md              # serves project.md on a localhost connection
mddocs project.md project.html # render project.md into project.html

$ mddocs sqddl.md
serving sqddl.md at localhost:6060
```

To add a new section and register it in the table of contents, append a `#headerID` to the end of a header (replace `headerID` with the actual header ID). The header ID should only contain unicode letters, digits, hyphen `-` and underscore `_`.

```text
## This is a header.

## This is a header with a headerID. #header-id <-- added to table of contents
```
