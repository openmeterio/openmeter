package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/openmeterio/openmeter/.dagger/internal/dagger"
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
				Latest:        dagger.GhLatestLatestTrue,
				VerifyTag:     true,
			})
		},

		// Disabled for now as we don't have a way to generate the SDK yet
		// func(ctx context.Context) error {
		// 	return m.publishPythonSdk(ctx, version, pypiToken)
		// },
		func(ctx context.Context) error {
			return m.PublishJavascriptSdk(ctx, version, "latest", npmToken)
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

// TODO: keep in sync with api/client/javascript/Makefile for now, if the release process is moved to nix, can be removed
func (m *Openmeter) PublishJavascriptSdk(ctx context.Context, version string, tag string, npmToken *dagger.Secret) error {
	// TODO: generate SDK on the fly?
	_, err := dag.Container().
		From(NODEJS_CONTAINER_IMAGE).
		WithExec([]string{"npm", "install", "-g", fmt.Sprintf("corepack@v%s", COREPACK_VERSION)}).
		WithExec([]string{"corepack", "enable"}).
		WithDirectory("/work", m.Source.Directory("api")).
		WithWorkdir("/work/client/javascript").
		WithExec([]string{"pnpm", "install", "--frozen-lockfile"}).
		WithExec([]string{"pnpm", "version", version, "--no-git-tag-version"}).
		WithEnvVariable("CACHE_BUSTER", time.Now().Format(time.RFC3339Nano)).
		WithExec([]string{"sh", "-c", "echo '//registry.npmjs.org/:_authToken=${NPM_TOKEN}' > /root/.npmrc"}).
		WithSecretVariable("NPM_TOKEN", npmToken).
		WithExec([]string{"pnpm", "publish", "--no-git-checks", "--tag", tag}).
		Sync(ctx)

	return err
}
