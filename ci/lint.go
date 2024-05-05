package main

import (
	"context"

	"github.com/sourcegraph/conc/pool"
)

func (m *Ci) Lint() *Lint {
	return &Lint{
		Source: m.Source,
	}
}

type Lint struct {
	Source *Directory
}

func (m *Lint) All(ctx context.Context) error {
	p := pool.New().WithErrors().WithContext(ctx)

	p.Go(syncFunc(m.Go()))
	p.Go(syncFunc(m.Openapi()))
	p.Go(m.Helm)

	return p.Wait()
}

func (m *Lint) Go() *Container {
	return dag.GolangciLint(GolangciLintOpts{
		Version:   golangciLintVersion,
		GoVersion: goVersion,
	}).
		Run(m.Source, GolangciLintRunOpts{
			Verbose: true,
		})
}

func (m *Lint) Openapi() *Container {
	return dag.Spectral(SpectralOpts{Version: spectralVersion}).
		Lint([]*File{m.Source.File("api/openapi.yaml")}, m.Source.File(".spectral.yaml"))
}

func (m *Lint) Helm(ctx context.Context) error {
	p := pool.New().WithErrors().WithContext(ctx)

	p.Go(syncFunc(helmChart(m.Source, "openmeter").Lint()))
	p.Go(syncFunc(helmChart(m.Source, "benthos-collector").Lint()))

	return p.Wait()
}
