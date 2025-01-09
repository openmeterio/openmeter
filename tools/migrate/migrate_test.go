package migrate_test

import (
	"database/sql"
	"errors"
	"slices"
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
		return int(i.version) - int(j.version)
	})

	return downs
}

type runner struct {
	stops stops
}

func (r runner) Test(t *testing.T) {
	testDB := testutils.InitPostgresDB(t)
	defer testDB.PGDriver.Close()

	migrator, err := migrate.NewMigrate(testDB.URL, migrate.OMMigrations, "migrations")
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
			t.Fatal(err)
		}

		stop.action(t, testDB.PGDriver.DB())
	}

	if err := migrator.Down(); err != nil {
		t.Fatal(err)
	}

	// Then let's go up again to make sure nothing's bricked
	if err := migrator.Up(); err != nil {
		t.Fatal(err)
	}
}
