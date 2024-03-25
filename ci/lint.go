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
