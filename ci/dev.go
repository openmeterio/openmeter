package main

import (
	"bytes"
	"context"
	"strings"

	"github.com/dave/jennifer/jen"
	"github.com/google/go-github/v63/github"

	"github.com/openmeterio/openmeter/ci/internal/dagger"
)

func (m *Ci) Dev() *Dev {
	return &Dev{
		Source: m.Source,
	}
}

type Dev struct {
	Source *dagger.Directory
}

// Udate dependency versions used in CI.
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

	versions := make(map[string]string)

	// GolangCI Lint
	{
		release, _, err := githubClient.Repositories.GetLatestRelease(ctx, "golangci", "golangci-lint")
		if err != nil {
			return nil, err
		}

		versions["golangciLint"] = release.GetTagName()
	}

	// Helm
	{
		release, _, err := githubClient.Repositories.GetLatestRelease(ctx, "helm", "helm")
		if err != nil {
			return nil, err
		}

		versions["helm"] = strings.TrimPrefix(release.GetTagName(), "v")
	}

	// Helm docs
	{
		release, _, err := githubClient.Repositories.GetLatestRelease(ctx, "norwoodj", "helm-docs")
		if err != nil {
			return nil, err
		}

		versions["helmDocs"] = release.GetTagName()
	}

	// Spectral
	{
		release, _, err := githubClient.Repositories.GetLatestRelease(ctx, "stoplightio", "spectral")
		if err != nil {
			return nil, err
		}

		versions["spectral"] = strings.TrimPrefix(release.GetTagName(), "v")
	}

	f := jen.NewFile("main")

	f.Const().DefsFunc(func(g *jen.Group) {
		for tool, version := range versions {
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
