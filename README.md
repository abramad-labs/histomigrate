[![GitHub Workflow Status (branch)](https://img.shields.io/github/actions/workflow/status/abramad-labs/histomigrate/ci.yaml?branch=master)](https://github.com/abramad-labs/histomigrate/actions/workflows/ci.yaml?query=branch%3Amaster)
[![GoDoc](https://pkg.go.dev/badge/github.com/abramad-labs/histomigrate)](https://pkg.go.dev/github.com/abramad-labs/histomigrate)
[![Coverage Status](https://img.shields.io/coveralls/github/abramad-labs/histomigrate/master.svg)](https://coveralls.io/github/abramad-labs/histomigrate?branch=master)
[![packagecloud.io](https://img.shields.io/badge/deb-packagecloud.io-844fec.svg)](https://packagecloud.io/golang-migrate/migrate?filter=debs)
[![Docker Pulls](https://img.shields.io/docker/pulls/migrate/migrate.svg)](https://hub.docker.com/r/migrate/migrate/)
![Supported Go Versions](https://img.shields.io/badge/Go-1.23%2C%201.24-lightgrey.svg)
[![GitHub Release](https://img.shields.io/github/release/abramad-labs/histomigrate.svg)](https://github.com/abramad-labs/histomigrate/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/abramad-labs/histomigrate)](https://goreportcard.com/report/github.com/abramad-labs/histomigrate)

# HistoMigrate

__Database migrations written in Go, now with **out-of-order support**! Use as [CLI](#cli-usage) or import as [library](#use-in-your-go-project).__

* **HistoMigrate** reads migrations from [sources](#migration-sources) and applies them in correct order to a [database](#databases), intelligently handling non-linear application scenarios.
* Drivers are "dumb"; HistoMigrate glues everything together and makes sure the logic is bulletproof (keeping drivers lightweight).
* Database drivers don't assume things or try to correct user input. When in doubt, HistoMigrate fails.

Forked from [golang-migrate/migrate](https://github.com/golang-migrate/migrate)

---

## Out-of-Order Migration Support

HistoMigrate introduces robust support for **out-of-order migrations**, solving common pain points in modern deployment workflows, such as:

* **Cherry-picking Hotfixes:** Apply critical hotfixes that include migrations with higher version numbers without issues, even if lower-version migrations from the main branch are yet to be applied.
* **Non-Linear Development:** Seamlessly manage migration application when features or bug fixes are merged or deployed in a non-sequential order.

This is achieved by transitioning from a single "current version" tracking model to a **history-based approach** where the migration table records every individual migration that has been applied, rather than just the highest timestamp. This ensures your database's migration state is always an accurate reflection of exactly which scripts have run.

---

## Databases

Database drivers run migrations. [Add a new database?](database/driver.go)

* [PostgreSQL](database/postgres)
* [PGX v4](database/pgx)
* [PGX v5](database/pgx/v5)
* [Redshift](database/redshift)
* [Ql](database/ql)
* [Cassandra / ScyllaDB](database/cassandra)
* [SQLite](database/sqlite)
* [SQLite3](database/sqlite3)
* [SQLCipher](database/sqlcipher)
* [MySQL / MariaDB](database/mysql)
* [Neo4j](database/neo4j)
* [MongoDB](database/mongodb)
* [CrateDB](database/crate)
* [Shell](database/shell)
* [Google Cloud Spanner](database/spanner)
* [CockroachDB](database/cockroachdb)
* [YugabyteDB](database/yugabytedb)
* [ClickHouse](database/clickhouse)
* [Firebird](database/firebird)
* [MS SQL Server](database/sqlserver)
* [rqlite](database/rqlite)

### Database URLs

Database connection strings are specified via URLs. The URL format is driver dependent but generally has the form: `dbdriver://username:password@host:port/dbname?param1=true&param2=false`

Any [reserved URL characters](https://en.wikipedia.org/wiki/Percent-encoding#Percent-encoding_reserved_characters) need to be escaped. Note, the `%` character also [needs to be escaped](https://en.wikipedia.org/wiki/Percent-encoding#Percent-encoding_the_percent_character)

Explicitly, the following characters need to be escaped:
`!`, `#`, `$`, `%`, `&`, `'`, `(`, `)`, `*`, `+`, `,`, `/`, `:`, `;`, `=`, `?`, `@`, `[`, `]`

It's easiest to always run the URL parts of your DB connection URL (e.g. username, password, etc) through an URL encoder. See the example Python snippets below:

```bash
$ python3 -c 'import urllib.parse; print(urllib.parse.quote(input("String to encode: "), ""))'
String to encode: FAKEpassword!#$%&'()*+,/:;=?@[]
FAKEpassword%21%23%24%25%26%27%28%29%2A%2B%2C%2F%3A%3B%3D%3F%40%5B%5D
$ python2 -c 'import urllib; print urllib.quote(raw_input("String to encode: "), "")'
String to encode: FAKEpassword!#$%&'()*+,/:;=?@[]
FAKEpassword%21%23%24%25%26%27%28%29%2A%2B%2C%2F%3A%3B%3D%3F%40%5B%5D
$
````

-----

## Migration Sources

Source drivers read migrations from local or remote sources. [Add a new source?](https://www.google.com/search?q=source/driver.go)

  * [Filesystem](https://www.google.com/search?q=source/file) - read from filesystem
  * [io/fs](https://www.google.com/search?q=source/iofs) - read from a Go [io/fs](https://pkg.go.dev/io/fs#FS) for embedding migrations directly into Go binaries.
  * [Go-Bindata](https://www.google.com/search?q=source/go_bindata) - read from embedded binary data ([jteeuwen/go-bindata](https://github.com/jteeuwen/go-bindata))
  * [pkger](https://www.google.com/search?q=source/pkger) - read from embedded binary data ([markbates/pkger](https://github.com/markbates/pkger))
  * [GitHub](https://www.google.com/search?q=source/github) - read from remote GitHub repositories
  * [GitHub Enterprise](https://www.google.com/search?q=source/github_ee) - read from remote GitHub Enterprise repositories
  * [Bitbucket](https://www.google.com/search?q=source/bitbucket) - read from remote Bitbucket repositories
  * [Gitlab](https://www.google.com/search?q=source/gitlab) - read from remote Gitlab repositories
  * [AWS S3](https://www.google.com/search?q=source/aws_s3) - read from Amazon Web Services S3
  * [Google Cloud Storage](https://www.google.com/search?q=source/google_cloud_storage) - read from Google Cloud Platform Storage

-----

## CLI usage

This section details how to use the `migrate` CLI tool for managing your database migrations.

  * Simple wrapper around this library.
  * Handles ctrl+c (SIGINT) gracefully.
  * No config search paths, no config files, no magic ENV var injections.

[CLI Documentation](https://www.google.com/search?q=cmd/migrate) (includes CLI install instructions)

### Basic usage

```bash
$ migrate -source file://path/to/migrations -database postgres://localhost:5432/database up 2
```

### Docker usage

```bash
$ docker run -v {{ migration dir }}:/migrations --network host migrate/migrate \
    -path=/migrations/ -database postgres://localhost:5432/database up 2
```

-----

### **Database Migration Guide using `migrate` CLI**

This guide explains how to use the `migrate` CLI tool for managing database schema migrations with **HistoMigrate's out-of-order capabilities**.

#### Prerequisites

  * [Install `migrate`](https://www.google.com/search?q=https://github.com/abramad-labs/histomigrate/tree/master/cmd/migrate%23installation)
  * Ensure you have a valid `DATABASE_URL` (e.g., `postgres://user:pass@host:port/dbname?sslmode=disable`)
  * Create a `db/migrations` directory or any folder where you store migration files

-----

#### 1\. Create a New Migration File

```bash
migrate create -ext sql -dir db/migrations -tz Local $MIGRATION_TITLE
```

##### Example

```bash
migrate create -ext sql -dir db/migrations -tz Local create_users_table
```

##### Description

Creates two `.sql` files in the `db/migrations` directory with specified names and a current timestamp prefix:

  * `xxxxxx_create_users_table.up.sql`
  * `xxxxxx_create_users_table.down.sql`

Where `xxxxxx` is a UTC timestamp (or local time if `-tz Local` is used).

-----

#### 2\. Apply All Available Migrations

```bash
migrate -path db/migrations -database postgres://user:pass@host:port/dbname?sslmode=disable up
```

##### Description

Applies all `.up.sql` migrations in order, skipping any already applied.

-----

#### 3\. Apply a Limited Number of Migrations

```bash
migrate -path db/migrations -database postgres://user:pass@host:port/dbname?sslmode=disable up 1
```

##### Description

Applies only **1** new migration. Replace `1` with any desired number.

-----

#### 4\. Apply a Specific Migration (`do`)

```bash
migrate -path db/migrations -database postgres://user:pass@host:port/dbname?sslmode=disable do $VERSION
```

##### Example

```bash
migrate -path db/migrations -database $DATABASE_URL do 20250525112233
```

##### Description

Applies the specified migration's `.up.sql` script to the database. This command explicitly applies a migration **regardless of its version order relative to the current head**, allowing for out-of-order application scenarios (e.g., hotfixes).

-----

#### 5\. Revert a Specific Migration (`undo`)

```bash
migrate -path db/migrations -database postgres://user:pass@host:port/dbname?sslmode=disable undo $VERSION
```

##### Description

Rolls back a specific migration using its `.down.sql` script. Use the migration timestamp to undo. This command specifically targets and reverts a previously applied migration, even if it was applied out-of-order.

##### Example

```bash
migrate -path db/migrations -database $DATABASE_URL undo 20250525112233
```

-----

#### 6\. Revert All Migrations (`down`)

```bash
migrate -path db/migrations -database postgres://user:pass@host:port/dbname?sslmode=disable down
```

##### Description

Rolls back **all** applied migrations in reverse order of application.

> ⚠️ **Use with caution in production.** This command will revert the entire schema history, including migrations applied with the `do` command.

-----

#### 7\. Check Current Migration Version

```bash
migrate -path db/migrations -database $DATABASE_URL version
```

##### Description

Shows the last migration timestamp that was applied to the database.

-----

#### Example PostgreSQL URL

```bash
postgres://username:password@localhost:5432/dbname?sslmode=disable
```

-----

#### Common Commands Summary

| Command | Description |
| :---------------------------------------------------------- | :--------------------------------------- |
| `migrate create -ext sql -dir $PATH -tz Local name`         | Create a new migration file              |
| `migrate -path $PATH -database $DATABASE_URL up`            | Apply all new migrations                 |
| `migrate -path $PATH -database $DATABASE_URL up $LIMIT`     | Apply a limited number of new migrations |
| `migrate -path $PATH -database $DATABASE_URL do $VERSION`   | ✅ Apply a specific migration (out-of-order possible) |
| `migrate -path $PATH -database $DATABASE_URL undo $VERSION` | ⬅️ Roll back specified migration         |
| `migrate -path $PATH -database $DATABASE_URL down`          | 🔁 Roll back all migrations              |
| `migrate -path $PATH -database $DATABASE_URL down $LIMIT`   | ⬅️ Roll back limited number of migrations |
| `migrate -path $PATH -database $DATABASE_URL version`       | Show last applied migration timestamp    |

-----

## Use in your Go project

  * API is stable and frozen for this release.
  * Uses [Go modules](https://golang.org/cmd/go/#hdr-Modules__module_versions__and_more) to manage dependencies.
  * To help prevent database corruptions, it supports graceful stops via `GracefulStop chan bool`.
  * Bring your own logger.
  * Uses `io.Reader` streams internally for low memory overhead.
  * Thread-safe and no goroutine leaks.

**[Go Documentation](https://pkg.go.dev/github.com/abramad-labs/histomigrate)**

```go
import (
    "[github.com/abramad-labs/histomigrate](https://github.com/abramad-labs/histomigrate)"
    _ "[github.com/abramad-labs/histomigrate/database/postgres](https://github.com/abramad-labs/histomigrate/database/postgres)"
    _ "[github.com/abramad-labs/histomigrate/source/github](https://github.com/abramad-labs/histomigrate/source/github)"
)

func main() {
    m, err := migrate.New(
        "github://mattes:personal-access-token@mattes/migrate_test",
        "postgres://localhost:5432/database?sslmode=enable")
    m.Steps(2)
}
```

Want to use an existing database client?

```go
import (
    "database/sql"
    _ "[github.com/lib/pq](https://github.com/lib/pq)"
    "[github.com/abramad-labs/histomigrate](https://github.com/abramad-labs/histomigrate)"
    "[github.com/abramad-labs/histomigrate/database/postgres](https://github.com/abramad-labs/histomigrate/database/postgres)"
    _ "[github.com/abramad-labs/histomigrate/source/file](https://github.com/abramad-labs/histomigrate/source/file)"
)

func main() {
    db, err := sql.Open("postgres", "postgres://localhost:5432/database?sslmode=enable")
    driver, err := postgres.WithInstance(db, &postgres.Config{})
    m, err := migrate.NewWithDatabaseInstance(
        "file:///migrations",
        "postgres", driver)
    m.Up() // or m.Steps(2) if you want to explicitly set the number of migrations to run
}
```

-----

## Getting started

Go to [getting started](https://www.google.com/search?q=GETTING_STARTED.md)

-----

## Tutorials

  * [CockroachDB](https://www.google.com/search?q=database/cockroachdb/TUTORIAL.md)
  * [PostgreSQL](https://www.google.com/search?q=database/postgres/TUTORIAL.md)

(more tutorials to come)

-----

## Migration files

Each migration has an up and down migration. [Why?](https://www.google.com/search?q=FAQ.md%23why-two-separate-files-up-and-down-for-a-migration)

```bash
1481574547_create_users_table.up.sql
1481574547_create_users_table.down.sql
```

[Best practices: How to write migrations.](https://www.google.com/search?q=MIGRATIONS.md)

-----

## Coming from another db migration tool?

Check out [migradaptor](https://github.com/musinit/migradaptor/).
*Note: migradaptor is not affiliated or supported by this project*

-----

## Versions

Version | Supported? | Import | Notes
:--------|:------------|:--------|:------
**master** | :white\_check\_mark: | `import "github.com/abramad-labs/histomigrate"` | New features and bug fixes arrive here first |
**v4** | :x: | `import "github.com/golang-migrate/migrate"` (with package manager) | **DO NOT USE** - This refers to the upstream `golang-migrate` v4. |
**v3** | :x: | `import "gopkg.in/golang-migrate/migrate.v3"` | **DO NOT USE** - This refers to the upstream `golang-migrate` v3. |

-----

## Development and Contributing

Yes, please\! [`Makefile`](https://www.google.com/search?q=Makefile) is your friend,
read the [development guide](https://www.google.com/search?q=CONTRIBUTING.md).

Also have a look at the [FAQ](FAQ.md).