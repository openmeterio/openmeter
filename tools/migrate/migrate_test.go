package migrate_test

import (
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/tools/migrate"
)

// The main test runner
func TestUpDownUp(t *testing.T) {
	runner{}.Test(t)
}

// Helpers

const (
	directionUp int = iota
	directionDown
)

type stop struct {
	// The migration version AT WHICH the break occurs (after applied = inclusive)
	version   uint
	direction int
	// We have to use the raw SQL connection as any ORM would only use the latest schema version
	action func(t *testing.T, db *sql.DB)
}

type stops []stop

func (s *stops) add(stops stops) {
	n := append(*s, stops...)
	*s = n
}

func TestAdd(t *testing.T) {
	b := stops{}

	b.add(stops{{
		version:   1,
		direction: directionUp,
	}})

	require.Equal(t, 1, len(b))
}

func (s stops) ups() stops {
	var ups stops
	for _, stop := range s {
		if stop.direction == directionUp {
			ups = append(ups, stop)
		}
	}

	slices.SortStableFunc(ups, func(i, j stop) int {
		return int(i.version) - int(j.version)
	})

	return ups
}

func (s stops) downs() stops {
	var downs stops
	for _, stop := range s {
		if stop.direction == directionDown {
			downs = append(downs, stop)
		}
	}

	slices.SortStableFunc(downs, func(i, j stop) int {
		return int(j.version) - int(i.version)
	})

	return downs
}

type runner struct {
	stops stops
}

func (r runner) Test(t *testing.T) {
	testDB := testutils.InitPostgresDB(t)
	defer testDB.PGDriver.Close()

	migrator, err := migrate.New(migrate.MigrateOptions{
		ConnectionString: testDB.URL,
		Migrations:       migrate.OMMigrationsConfig,
		Logger:           testutils.NewLogger(t),
	})
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		err1, err2 := migrator.Close()
		err := errors.Join(err1, err2)
		if err != nil {
			t.Fatal(err)
		}
	}()

	t.Logf("Running migrations with %d stops", len(r.stops))

	for _, stop := range r.stops.ups() {
		if err := migrator.Migrate(stop.version); err != nil {
			t.Fatal(err)
		}

		stop.action(t, testDB.PGDriver.DB())
	}

	// We go till the very end either way
	if err := migrator.Up(); err != nil && err.Error() != "no change" {
		t.Fatal(err)
	}

	for _, stop := range r.stops.downs() {
		if err := migrator.Migrate(stop.version); err != nil {
			if err != migrate.ErrNoChange {
				t.Fatal(err)
			}
		}

		stop.action(t, testDB.PGDriver.DB())
	}

	// After all stops, let's purge the db
	r.purgeDB(t, testDB)

	// Now let's run the rest of the migrations
	if err := migrator.Down(); err != nil {
		t.Fatal(err)
	}

	// Then let's go up again to make sure nothing's bricked
	if err := migrator.Up(); err != nil {
		t.Fatal(err)
	}
}

// PurgeDB truncates all tables in the database
// This is needed after data migrations as unfortunately either
// - data migrations are missing for some previous schema changes
// - proper data migrations are not possible as the schemas are not compatible
func (r runner) purgeDB(t *testing.T, db *testutils.TestDB) {
	// First, get the current search path to identify the schema
	var searchPath string
	err := db.PGDriver.DB().QueryRow(`SHOW search_path`).Scan(&searchPath)
	require.NoError(t, err)

	// Get the first schema from the search path (typically the one that's used by default)
	// The search path is usually in the format: "$user", public
	schemas := strings.Split(searchPath, ",")
	var schema string
	for _, s := range schemas {
		s = strings.TrimSpace(s)
		if s != "" && s != `"$user"` && !strings.Contains(s, "$") {
			schema = strings.Trim(s, `"`)
			break
		}
	}

	// If no valid schema found, default to public as a fallback
	if schema == "" {
		schema = "public"
	}

	t.Logf("Purging database tables in schema: %s", schema)

	// Get all tables in the identified schema
	rows, err := db.PGDriver.DB().Query(`
		SELECT tablename
		FROM pg_tables
		WHERE schemaname = $1
		AND tablename != $2
	`, schema, migrate.OMMigrationsConfig.StateTableName)
	require.NoError(t, err)
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		err := rows.Scan(&tableName)
		require.NoError(t, err)
		tables = append(tables, tableName)
	}

	if len(tables) == 0 {
		t.Logf("No tables found in schema %s", schema)
		return
	}

	// Truncate all tables in one transaction
	tx, err := db.PGDriver.DB().Begin()
	require.NoError(t, err)

	// Disable foreign key constraints during the truncation
	_, err = tx.Exec("SET CONSTRAINTS ALL DEFERRED")
	require.NoError(t, err)

	for _, table := range tables {
		// Quote the table name and schema for safety
		quotedTable := fmt.Sprintf(`"%s"."%s"`, schema, table)
		// Use TRUNCATE with CASCADE to remove dependent data
		_, err = tx.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", quotedTable))
		require.NoError(t, err)
	}

	err = tx.Commit()
	require.NoError(t, err)
}
