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
	"bytes"
	"context"
	"strings"

	"github.com/dave/jennifer/jen"
	"github.com/elliotchance/orderedmap/v2"
	"github.com/google/go-github/v63/github"

	"github.com/openmeterio/openmeter/.dagger/internal/dagger"
)

func (m *Openmeter) Dev() *Dev {
	return &Dev{
		Source: m.Source,
		CI:     m,
	}
}

type Dev struct {
	Source *dagger.Directory
	CI     *Openmeter
}

// Update dependency versions used in CI.
func (m *Dev) UpdateVersions(
	ctx context.Context,

	// +optional
	githubToken *dagger.Secret,
) (*dagger.File, error) {
	githubClient := github.NewClient(nil)

	if githubToken != nil {
		token, err := githubToken.Plaintext(ctx)
		if err != nil {
			return nil, err
		}

		githubClient = githubClient.WithAuthToken(token)
	}

	versions := orderedmap.NewOrderedMap[string, string]()

	// GolangCI Lint
	{
		release, _, err := githubClient.Repositories.GetLatestRelease(ctx, "golangci", "golangci-lint")
		if err != nil {
			return nil, err
		}

		versions.Set("golangciLint", release.GetTagName())
	}

	// Helm
	{
		release, _, err := githubClient.Repositories.GetLatestRelease(ctx, "helm", "helm")
		if err != nil {
			return nil, err
		}

		versions.Set("helm", strings.TrimPrefix(release.GetTagName(), "v"))
	}

	// Helm docs
	{
		release, _, err := githubClient.Repositories.GetLatestRelease(ctx, "norwoodj", "helm-docs")
		if err != nil {
			return nil, err
		}

		versions.Set("helmDocs", release.GetTagName())
	}

	// Spectral
	{
		release, _, err := githubClient.Repositories.GetLatestRelease(ctx, "stoplightio", "spectral")
		if err != nil {
			return nil, err
		}

		versions.Set("spectral", strings.TrimPrefix(release.GetTagName(), "v"))
	}

	f := jen.NewFile("main")

	f.Const().DefsFunc(func(g *jen.Group) {
		for _, tool := range versions.Keys() {
			version, _ := versions.Get(tool)

			g.Id(tool + "Version").Op("=").Lit(version)
		}
	})

	var buf bytes.Buffer

	err := f.Render(&buf)
	if err != nil {
		return nil, err
	}

	return dag.Directory().WithNewFile("versions.go", buf.String()).File("versions.go"), nil
}

// Check OpenAPI changes between the "old" hand-written version and the "new" TypeSpec generated one.
func (m *Dev) OpenapiChanges() *dagger.Service {
	old := m.Source.File("api/openapi.yaml")
	new := m.CI.Generate().Openapi()

	return dag.OpenapiChanges().Diff(old, new).HTML().Serve()
}
