package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

func (m *Ci) Release(ctx context.Context, version string, githubActor string, githubToken *Secret, pypiToken *Secret) error {
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

			_, err := dag.Gh(GhOpts{
				Token: githubToken,
				Repo:  "openmeterio/openmeter",
			}).Release().Create(ctx, version, version, GhReleaseCreateOpts{
				Files:         releaseAssets,
				GenerateNotes: true,
				Latest:        true,
				VerifyTag:     true,
			})

			return err
		},

		func(ctx context.Context) error {
			return m.publishPythonSdk(ctx, version, pypiToken)
		},
	)

	return p.wait()
}

func (m *Ci) pushHelmChart(ctx context.Context, name string, version string, githubActor string, githubToken *Secret) error {
	_, err := m.Build().
		helmChart(name, version).
		WithRegistryAuth("ghcr.io", githubActor, githubToken).
		Publish(ctx, "oci://ghcr.io/openmeterio/helm-charts")

	return err
}

func (m *Ci) releaseAssets(version string) []*File {
	binaryArchives := m.binaryArchives(version)
	checksums := dag.Checksum().Sha256().Calculate(binaryArchives)

	return append(binaryArchives, checksums)
}

func (m *Ci) binaryArchives(version string) []*File {
	platforms := []Platform{
		"linux/amd64",
		"linux/arm64",

		"darwin/amd64",
		"darwin/arm64",
	}

	archives := make([]*File, 0, len(platforms))

	for _, platform := range platforms {
		archives = append(archives, m.binaryArchive(version, platform))
	}

	return archives
}

func (m *Ci) binaryArchive(version string, platform Platform) *File {
	var archiver interface {
		Archive(name string, source *Directory) *File
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

func (m *Ci) publishPythonSdk(ctx context.Context, version string, pypiToken *Secret) error {
	_, err := dag.Python(PythonOpts{
		Container: dag.Python(PythonOpts{Container: dag.Container().From("pypy:3.10-slim")}).
			WithPipCache(dag.CacheVolume("openmeter-pip")).
			Container().
			WithExec([]string{"pip", "--disable-pip-version-check", "install", "pipx"}).
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
