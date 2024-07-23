package e2e

import (
	"context"
	"io"
	"strings"
	"testing"

	"ariga.io/atlas/sql/migrate"
	"entgo.io/ent/dialect/sql/schema"
	entMigrate "github.com/openmeterio/openmeter/internal/ent/db/migrate"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func TestNoSchemaDiffOnMigrate(t *testing.T) {
	// driver := testutils.InitPostgresDB(t)
	driver, err := entutils.GetPGDriver("postgres://postgres:postgres@localhost:5432/postgres")
	if err != nil {
		t.Fatalf("failed to get pg driver %s", err)
	}
	// initialize client & run migrations

	memDir := migrate.OpenMemDir("migrations")

	migrate, err := schema.NewMigrate(driver, schema.WithDir(memDir))
	if err != nil {
		t.Fatalf("failed to create migrate %s", err)
	}
	err = migrate.Diff(context.Background(), entMigrate.Tables...)
	if err != nil {
		t.Fatalf("failed to diff schema %s", err)
	}

	files, err := memDir.Files()
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
		t.Logf("file: %s", file.Name())
	}

	if filename == "" {
		t.Fatalf("no up.sql found")
	}

	file, err := memDir.Open(filename)
	if err != nil {
		t.Fatalf("failed to open diff file %s", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("failed to read diff file %s", err)
	}

	// check if diff file is empty
	if len(data) != 0 {
		t.Fatalf("schema diff found: %s", string(data))
	}
}
