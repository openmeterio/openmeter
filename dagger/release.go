package main

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/openmeterio/openmeter/dagger/internal/dagger"
)

func (m *Openmeter) Release(ctx context.Context, version string, githubActor string, githubToken *dagger.Secret, pypiToken *dagger.Secret, npmToken *dagger.Secret) error {
	p := newPipeline(ctx)

	p.addJobs(
		func(ctx context.Context) error {
			return m.pushHelmChart(ctx, "openmeter", version, githubActor, githubToken)
		},

		func(ctx context.Context) error {
			return m.pushHelmChart(ctx, "benthos-collector", version, githubActor, githubToken)
		},

		func(ctx context.Context) error {
			if githubToken == nil {
				return errors.New("GitHub token is required to publish a release")
			}

			releaseAssets := m.releaseAssets(version)

			return dag.Gh(dagger.GhOpts{
				Token: githubToken,
				Repo:  "openmeterio/openmeter",
			}).Release().Create(ctx, version, version, dagger.GhReleaseCreateOpts{
				Files:         releaseAssets,
				GenerateNotes: true,
				Latest:        true,
				VerifyTag:     true,
			})
		},

		func(ctx context.Context) error {
			return m.publishPythonSdk(ctx, version, pypiToken)
		},
		func(ctx context.Context) error {
			return m.publishNodeSdk(ctx, version, npmToken)
		},
		func(ctx context.Context) error {
			return m.publishWebSdk(ctx, version, npmToken)
		},
	)

	return p.wait()
}

func (m *Openmeter) pushHelmChart(ctx context.Context, name string, version string, githubActor string, githubToken *dagger.Secret) error {
	return m.Build().
		helmChart(name, version).
		WithRegistryAuth("ghcr.io", githubActor, githubToken).
		Publish(ctx, "oci://ghcr.io/openmeterio/helm-charts")
}

func (m *Openmeter) releaseAssets(version string) []*dagger.File {
	binaryArchives := m.binaryArchives(version)
	checksums := dag.Checksum().Sha256().Calculate(binaryArchives)

	return append(binaryArchives, checksums)
}

func (m *Openmeter) binaryArchives(version string) []*dagger.File {
	platforms := []dagger.Platform{
		"linux/amd64",
		"linux/arm64",

		"darwin/amd64",
		"darwin/arm64",
	}

	archives := make([]*dagger.File, 0, len(platforms))

	for _, platform := range platforms {
		archives = append(archives, m.binaryArchive(version, platform))
	}

	return archives
}

func (m *Openmeter) binaryArchive(version string, platform dagger.Platform) *dagger.File {
	var archiver interface {
		Archive(name string, source *dagger.Directory) *dagger.File
	} = dag.Archivist().TarGz()

	if strings.HasPrefix(string(platform), "windows/") {
		archiver = dag.Archivist().Zip()
	}

	return archiver.Archive(
		fmt.Sprintf("benthos-collector_%s", strings.ReplaceAll(string(platform), "/", "_")),
		dag.Directory().
			WithFile("", m.Build().Binary().benthosCollector(platform, version)).
			WithFile("", m.Source.File("README.md")).
			WithFile("", m.Source.File("LICENSE")),
	)
}

func (m *Openmeter) publishPythonSdk(ctx context.Context, version string, pypiToken *dagger.Secret) error {
	_, err := dag.Python(dagger.PythonOpts{
		Container: dag.Python(dagger.PythonOpts{Container: dag.Container().From("pypy:3.10-7.3.16-slim")}).
			WithPipCache(cacheVolume("pip")).
			Container().
			WithExec([]string{"pip", "--disable-pip-version-check", "install", "pipx"}).
			WithEnvVariable("PATH", "${PATH}:/root/.local/bin", dagger.ContainerWithEnvVariableOpts{Expand: true}).
			WithExec([]string{"pipx", "install", "poetry"}),
	}).
		WithSource(m.Source.Directory("api/client/python")). // TODO: generate SDK on the fly?
		Container().
		WithExec([]string{"poetry", "install"}).
		WithExec([]string{"poetry", "version", version}).
		WithSecretVariable("POETRY_PYPI_TOKEN_PYPI", pypiToken).
		WithEnvVariable("CACHE_BUSTER", time.Now().Format(time.RFC3339Nano)).
		WithExec([]string{"poetry", "publish", "--build"}).
		Sync(ctx)

	return err
}

func (m *Openmeter) publishNodeSdk(ctx context.Context, version string, npmToken *dagger.Secret) error {
	// TODO: generate SDK on the fly?
	return m.publishToNpm(ctx, "node", version, npmToken)
}

func (m *Openmeter) publishWebSdk(ctx context.Context, version string, npmToken *dagger.Secret) error {
	// TODO: generate SDK on the fly?
	return m.publishToNpm(ctx, "web", version, npmToken)
}

func (m *Openmeter) publishToNpm(ctx context.Context, pkg string, version string, npmToken *dagger.Secret) error {
	_, err := dag.Container().
		From("node:20.15.1-alpine3.20").
		WithExec([]string{"npm", "install", "-g", "pnpm"}).
		WithExec([]string{"sh", "-c", "echo '//registry.npmjs.org/:_authToken=${NPM_TOKEN}' > /root/.npmrc"}).
		WithSecretVariable("NPM_TOKEN", npmToken).
		WithDirectory("/work", m.Source.Directory("api")).
		WithWorkdir(path.Join("/work/client", pkg)).
		WithExec([]string{"pnpm", "install", "--frozen-lockfile"}).
		WithExec([]string{"pnpm", "version", version, "--no-git-tag-version"}).
		WithEnvVariable("CACHE_BUSTER", time.Now().Format(time.RFC3339Nano)).
		WithExec([]string{"pnpm", "publish", "--access=public", "--no-git-checks"}).
		Sync(ctx)

	return err
}
