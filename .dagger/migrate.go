// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"

	"github.com/sourcegraph/conc/pool"

	"github.com/openmeterio/openmeter/.dagger/internal/dagger"
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
		postgres := dag.Postgres(dagger.PostgresOpts{
			Version: postgresVersion,
			Name:    "no-diff",
		})

		result := atlas.
			WithServiceBinding("postgres", postgres.Service()).
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
			WithServiceBinding("postgres", dag.Postgres(dagger.PostgresOpts{
				Version: postgresVersion,
				Name:    "last-10",
			}).Service()).
			WithExec([]string{"atlas", "migrate", "--env", "ci", "lint", "--latest", "10"}),
	))

	// Validate checksum is intact
	p.Go(syncFunc(
		atlas.
			WithServiceBinding("postgres", dag.Postgres(dagger.PostgresOpts{
				Version: postgresVersion,
				Name:    "validate",
			}).Service()).
			WithExec([]string{"atlas", "migrate", "--env", "ci", "validate"}),
	))

	return p.Wait()
}
