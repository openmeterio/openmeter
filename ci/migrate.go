package main

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/ci/internal/dagger"
	"github.com/sourcegraph/conc/pool"
)

func (m *Ci) Migrate() *Migrate {
	return &Migrate{
		Source: m.Source,
	}
}

type Migrate struct {
	Source *dagger.Directory
}

func (m *Migrate) Check(
	ctx context.Context,
	// +optional
	baseRef string,
) error {
	if baseRef == "" {
		baseRef = "main"
	}

	nix := nix(m.Source)
	p := pool.New().WithErrors().WithContext(ctx)

	// Always validate migrations are in sync with schema
	p.Go(syncFunc(nix.
		WithServiceBinding("postgres", postgresNamed("no-diff")).
		WithExec([]string{"sh", "-c", "nix develop --impure .#dagger -c atlas migrate --env ci diff test"}).
		WithExec([]string{"sh", "-c", "if [[ -n $(git status --porcelain ./tools/migrate/migrations) ]]; then echo 'Migrations directory is dirty'; exit 1; fi"}),
	))
	// Always lint last 10 migrations
	p.Go(syncFunc(nix.
		WithServiceBinding("postgres", postgresNamed("last-10")).
		WithExec([]string{"sh", "-c", "nix develop --impure .#dagger -c atlas migrate --env ci lint --latest 10"}),
	))
	// Always compare changes with base branch (main)
	p.Go(syncFunc(nix.
		WithServiceBinding("postgres", postgresNamed("basebranch")).
		WithExec([]string{"sh", "-c", fmt.Sprintf("nix develop --impure .#dagger -c atlas migrate --env ci lint --git-base \"%s\"", baseRef)}),
	))
	// Validate checksum is intact
	p.Go(syncFunc(nix.
		WithServiceBinding("postgres", postgresNamed("validate")).
		WithExec([]string{"sh", "-c", "nix develop --impure .#dagger -c atlas migrate --env ci validate"}),
	))

	return p.Wait()
}

func nix(src *dagger.Directory) *dagger.Container {
	return dag.Nix().SetupNix(dagger.NixSetupNixOpts{
		Src: src,
	}).
		// Note: we have to use `sh -c` as otherwise devenv nix assertion fails: https://github.com/cachix/devenv/blob/b285601679c7686f623791ad93a8e0debc322633/src/modules/top-level.nix#L229
		WithExec([]string{"sh", "-c", "nix develop --impure .#dagger"}) // Prepare environment
}
