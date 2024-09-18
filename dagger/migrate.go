package main

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/dagger/internal/dagger"
	"github.com/sourcegraph/conc/pool"
)

func (m *Openmeter) Migrate() *Migrate {
	return &Migrate{
		Source: m.Source,
	}
}

type Migrate struct {
	Source *dagger.Directory
}

func (m *Migrate) Check(ctx context.Context) error {
	app := goModuleCross("").
		WithSource(m.Source).
		Container().
		WithEnvVariable("GOFLAGS", "-tags=musl")

	bin := dag.Container(dagger.ContainerOpts{
		Platform: "linux/amd64",
	}).From(atlasImage).File("atlas")

	atlas := app.
		WithFile("/bin/atlas", bin).
		WithDirectory("openmeter/ent", m.Source.Directory("openmeter/ent")).
		WithDirectory("tools/migrate/migrations", m.Source.Directory("tools/migrate/migrations")).
		WithFile("atlas.hcl", m.Source.File("atlas.hcl"))

	p := pool.New().WithErrors().WithContext(ctx)

	// Always validate schema is generated
	p.Go(func(ctx context.Context) error {
		result := app.
			WithExec([]string{"go", "generate", "-x", "./openmeter/ent/..."}).
			Directory("openmeter/ent")

		source := m.Source.Directory("openmeter/ent")

		err := diff(ctx, source, result)
		if err != nil {
			return fmt.Errorf("schema is not in sync with generated code")
		}
		return nil
	})

	// Always validate migrations are in sync with schema
	p.Go(func(ctx context.Context) error {
		result := atlas.
			WithServiceBinding("postgres", postgresNamed("no-diff")).
			WithExec([]string{"atlas", "migrate", "--env", "ci", "diff", "test"}).
			Directory("tools/migrate/migrations")

		source := m.Source.Directory("tools/migrate/migrations")
		err := diff(ctx, source, result)
		if err != nil {
			return fmt.Errorf("migrations are not in sync with schema")
		}

		return nil
	})

	// Always lint last 10 migrations
	p.Go(syncFunc(
		atlas.
			WithServiceBinding("postgres", postgresNamed("last-10")).
			WithExec([]string{"atlas", "migrate", "--env", "ci", "lint", "--latest", "10"}),
	))

	// Validate checksum is intact
	p.Go(syncFunc(
		atlas.
			WithServiceBinding("postgres", postgresNamed("validate")).
			WithExec([]string{"atlas", "migrate", "--env", "ci", "validate"}),
	))

	return p.Wait()
}
