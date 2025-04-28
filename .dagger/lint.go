package main

import (
	"context"

	"github.com/sourcegraph/conc/pool"

	"github.com/openmeterio/openmeter/.dagger/internal/dagger"
)

func (m *Openmeter) Lint() *Lint {
	return &Lint{
		Source: m.Source,
	}
}

type Lint struct {
	// +private
	Source *dagger.Directory
}

func (m *Lint) All(ctx context.Context) error {
	p := pool.New().WithErrors().WithContext(ctx)

	p.Go(syncFunc(m.Go()))
	p.Go(syncFunc(m.Typespec()))
	p.Go(syncFunc(m.Openapi()))
	p.Go(m.Helm)

	return p.Wait()
}

func (m *Lint) Go() *dagger.Container {
	return dag.GolangciLint(dagger.GolangciLintOpts{
		Version:   golangciLintVersion,
		Container: goModuleCross("").Container(),
		Cache:     cacheVolume("golangci-lint"),
	}).
		Run(m.Source, dagger.GolangciLintRunOpts{
			Verbose: true,
		})
}

func (m *Lint) Typespec() *dagger.Container {
	return typespecBase(m.Source.Directory("api/spec")).
		WithExec([]string{"pnpm", "lint"})
}

func (m *Lint) Openapi() *dagger.Container {
	return dag.Spectral(dagger.SpectralOpts{Version: spectralVersion}).
		Lint([]*dagger.File{m.Source.File("api/openapi.yaml")}, m.Source.File(".spectral.yaml"))
}

func (m *Lint) Helm(ctx context.Context) error {
	p := pool.New().WithErrors().WithContext(ctx)

	p.Go(syncFunc(helmChart(m.Source, "openmeter").Lint()))
	p.Go(syncFunc(helmChart(m.Source, "benthos-collector").Lint()))

	return p.Wait()
}
