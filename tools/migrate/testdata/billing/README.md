# Billing migration checks

Given billing depends on many entities (such as customer, billing profile, subscription etc.), it's easier to generate data using tests. This guide shows how we can create testcases based on sql dumps for faster validation.

We cannot use ent here, as the schema evolves, so we need a point-in-time snapshot. SQLC is a great codegenerator to generate ad-hoc queries, that we can persist to the db.

SQLC requires the DB schema to generate the queries, for now, I would just commit those alongside the tests in case we would need to tweak the test to understand what have happened with a faulty migration.

## How to generate a new testcase

Write a unit test that generates the required data. Make it fail with `t.FailNow()` so that the database snapshot is retained.

Dump the schema and the data:
- Schema dump: `pg_dump -s 'postgres://pgtdbuser:pgtdbpass@127.0.0.1:5432/testdb_tpl_92d6cc3e2b7979388fd8f7b12aad9c7b_inst_6aaae321?sslmode=disable' > tools/migrate/testdata/billing/flatfeetoubpflatfee/sqlc/db-schema.sql`
- Data dump: `pg_dump --exclude-table-data=schema_om --column-inserts -n public --inserts -a 'postgres://pgtdbuser:pgtdbpass@127.0.0.1:5432/testdb_tpl_92d6cc3e2b7979388fd8f7b12aad9c7b_inst_bf480b8a?sslmode=disable' > tools/migrate/testdata/billing/flatfeetoubpflatfee/fixture.sql`

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

	flatfeetoubpflatfeedb "github.com/openmeterio/openmeter/tools/migrate/testdata/billing/flatfeetoubpflatfee/db"
)

func TestMigrateFlatFeesToUBPFlatFees(t *testing.T) {
	runner{stops{
		{
			version:   20250527084817,
			direction: directionUp,
			action: func(t *testing.T, db *sql.DB) {
				loadFixture(t, db, "testdata/billing/flatfeetoubpflatfee/fixture.sql")
			},
		},
		{
			version:   20250527123425,
			direction: directionUp,
			action: func(t *testing.T, db *sql.DB) {
				q := flatfeetoubpflatfeedb.New(db)

        // execute queries
      },
    },
  }}.Test(t)
}
```
