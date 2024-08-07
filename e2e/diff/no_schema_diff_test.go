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

var pgUrl string

func TestMain(m *testing.M) {
	pgUrl = os.Getenv("POSTGRES_URL")

	os.Exit(m.Run())
}

func setup(t *testing.T) {
	t.Helper()
	if pgUrl == "" {
		t.Skip("POSTGRES_URL not set")
	}
}

func TestNoSchemaDiffOnMigrate(t *testing.T) {
	setup(t)
	driver, err := entutils.GetPGDriver(pgUrl)
	if err != nil {
		t.Fatalf("failed to get pg driver %s", err)
	}

	tmpDirPath, err := os.MkdirTemp("", "migrate")
	if err != nil {
		t.Fatalf("failed to create temp dir %s", err)
	}
	defer os.RemoveAll(tmpDirPath)

	dir, err := migrate.NewLocalDir(tmpDirPath)
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
