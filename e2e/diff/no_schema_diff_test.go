package e2e

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"ariga.io/atlas/sql/migrate"
	"entgo.io/ent/dialect/sql/schema"
	entMigrate "github.com/openmeterio/openmeter/internal/ent/db/migrate"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

var PG_URL string

func TestMain(m *testing.M) {
	PG_URL = os.Getenv("POSTGRES_URL")
	if PG_URL == "" {
		PG_URL = "postgres://postgres:postgres@localhost:5432/postgres"
	}

	os.Exit(m.Run())
}

const (
	DIR = "./tmp/"
)

func TestNoSchemaDiffOnMigrate(t *testing.T) {
	driver, err := entutils.GetPGDriver(PG_URL)
	if err != nil {
		t.Fatalf("failed to get pg driver %s", err)
	}

	// OpenMemDir doesn't work from inside dagger
	// dir := migrate.OpenMemDir("migrations")
	err = os.RemoveAll(DIR)
	if err != nil {
		t.Fatalf("failed to remove dir %s", err)
	}
	err = os.MkdirAll(DIR, os.ModePerm)
	if err != nil {
		t.Fatalf("failed to create dir %s", err)
	}
	dir, err := migrate.NewLocalDir(DIR)
	if err != nil {
		t.Fatalf("failed to open local dir %s", err)
	}

	migrate, err := schema.NewMigrate(driver, schema.WithDir(dir))
	if err != nil {
		t.Fatalf("failed to create migrate %s", err)
	}
	err = migrate.Diff(context.Background(), entMigrate.Tables...)
	if err != nil {
		t.Fatalf("failed to diff schema %s", err)
	}

	files, err := dir.Files()
	if err != nil {
		t.Fatalf("failed to list files %s", err)
	}

	var filename string
	for _, file := range files {
		// there's only a single pair of up & down sql
		if strings.Contains(file.Name(), "up.sql") {
			filename = file.Name()
			break
		}
	}

	// If there's no file then there's no diff. If we have a file then we want to display its content and fail
	if filename != "" {
		file, err := dir.Open(filename)
		if err != nil {
			t.Fatalf("failed to open diff file %s", err)
		}
		defer file.Close()

		data, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("failed to read diff file %s", err)
		}

		// sanity check
		if len(data) != 0 {
			t.Fatalf("schema diff found: %s", string(data))
		}
	}
}
