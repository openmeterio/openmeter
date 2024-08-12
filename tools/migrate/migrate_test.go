package migrate_test

import (
	"errors"
	"testing"

	"github.com/openmeterio/openmeter/internal/testutils"
	"github.com/openmeterio/openmeter/tools/migrate"
)

func TestUpDownUp(t *testing.T) {
	testDB := testutils.InitPostgresDB(t)

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

	if err := migrator.Up(); err != nil {
		t.Fatal(err)
	}

	if err := migrator.Down(); err != nil {
		t.Fatal(err)
	}

	if err := migrator.Up(); err != nil {
		t.Fatal(err)
	}
}
