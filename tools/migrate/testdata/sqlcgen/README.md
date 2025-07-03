# Migration test data generation

Given migrations depend on many entities (such as customer, billing profile, subscription etc.), it's easier to generate data using tests. This guide shows how we can create testcases based on sql dumps for faster validation.

We cannot use ent here, as the schema evolves, so we need a point-in-time snapshot. SQLC is a great codegenerator to generate ad-hoc queries, that we can persist to the db.

SQLC requires the DB schema to generate the queries, for now, I would just commit those alongside the tests in case we would need to tweak the test to understand what have happened with a faulty migration.

## Quick Start (Automated)

For a quick and automated way to generate SQLC testdata for any migration version, you can use the new make command:

```bash
make generate-sqlc-testdata VERSION=20240826120919
```

This command will:
1. Run the specified migration version against a fresh PostgreSQL database
2. Generate a schema dump using `pg_dump`
3. Create the proper SQLC configuration and placeholder queries
4. Generate Go structs for all database tables using SQLC
5. Export everything to `tools/migrate/testdata/sqlcgen/[VERSION]/`

The generated directory will contain:
- `sqlc.yaml` - SQLC configuration
- `sqlc/db-schema.sql` - Database schema dump
- `sqlc/queries.sql` - Placeholder queries (you can add your own queries here)
- `db/models.go` - Generated Go structs for all tables
- `db/db.go` - Database interface
- `db/queries.sql.go` - Generated query functions

After generation, you can add your specific queries to `sqlc/queries.sql` and run `sqlc generate` from the version directory if needed.

## How to generate a new testcase (Manual)

Write a unit test that generates the required data. Make it fail with `t.FailNow()` so that the database snapshot is retained.

Dump the schema and the data:
- Schema dump: `pg_dump -s 'postgres://pgtdbuser:pgtdbpass@127.0.0.1:5432/testdb_tpl_92d6cc3e2b7979388fd8f7b12aad9c7b_inst_6aaae321?sslmode=disable' > tools/migrate/testdata/sqlcgen/20250605102416/sqlc/db-schema.sql`
  - You can also use the `-n` flag to specifiy individual schemas: `pg_dump -Ox -s -t '*billing*' 'postgres://pgtdbuser:pgtdbpass@127.0.0.1:5432/testdb_tpl_92d6cc3e2b7979388fd8f7b12aad9c7b_inst_4710adae?sslmode=disable'`
- Data dump: `pg_dump --exclude-table-data=schema_om --column-inserts -n public --inserts -a 'postgres://pgtdbuser:pgtdbpass@127.0.0.1:5432/testdb_tpl_92d6cc3e2b7979388fd8f7b12aad9c7b_inst_bf480b8a?sslmode=disable' > tools/migrate/testdata/sqlcgen/20250605102416/fixture.sql`

Create sqlc.yaml:
```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "sqlc/queries.sql"
    schema: "sqlc/db-schema.sql"
    gen:
      go:
        package: "db"
        out: "db"
        omit_unused_structs: true
```

Write any queries that you need into `sqlc/queries.sql` (see existing testcases as examples).

To regenerate the db library, go to the folder containing sqlc.yaml and issue
```sh
sqlc generate
```

We are not adding the `//go:generate` flag to the file, as these files should not change over time.

Then you can write a migration testcase using the following skeleton:

```go
package migrate_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"strings"
	"testing"

	v20250605102416 "github.com/openmeterio/openmeter/tools/migrate/testdata/sqlcgen/20250605102416/db"
)

func TestMigrateFlatFeesToUBPFlatFees(t *testing.T) {
	runner{stops{
		{
			version:   20250605102416,
			direction: directionUp,
			action: func(t *testing.T, db *sql.DB) {
				loadFixture(t, db, "testdata/sqlcgen/20250605102416/fixture.sql")
			},
		},
		{
			version:   20250605131637,
			direction: directionUp,
			action: func(t *testing.T, db *sql.DB) {
				q := v20250605102416.New(db)

        // execute queries
      },
    },
  }}.Test(t)
}
```

## Directory Structure

The directories are named by the migration version number (timestamp) where the database snapshot was taken. For example:
- `20250605102416/` - Contains schema and fixture for migration version 20250605102416
- `20250609172811/` - Contains schema and fixture for migration version 20250609172811

Each directory contains:
- `db/` - Generated Go code from sqlc
- `fixture.sql` - Test data dump
- `sqlc/` - SQLC configuration and queries
- `sqlc.yaml` - SQLC configuration file
