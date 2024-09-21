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
	p.Go(syncFunc(m.Openapi()))
	p.Go(m.Helm)

	return p.Wait()
}

func (m *Lint) Go() *dagger.Container {
	return dag.GolangciLint(dagger.GolangciLintOpts{
		Version:     golangciLintVersion,
		GoContainer: goModuleCross("").Container(),
	}).
		WithLinterCache(cacheVolume("golangci-lint")).
		Run(m.Source, dagger.GolangciLintRunOpts{
			Verbose: true,
		})
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
