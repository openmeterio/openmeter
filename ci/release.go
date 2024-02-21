package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

func (m *Ci) Release(ctx context.Context, version string, githubActor string, githubToken *Secret) error {
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
				Version: ghVersion,
				Token:   githubToken,
				Repo:    "openmeterio/openmeter",
			}).Release().Create(version, version, GhReleaseCreateOpts{
				Files:         releaseAssets,
				GenerateNotes: true,
				Latest:        true,
				VerifyTag:     true,
			}).Sync(ctx)

			return err
		},
	)

	return p.wait()
}

func (m *Ci) pushHelmChart(ctx context.Context, name string, version string, githubActor string, githubToken *Secret) error {
	chart := m.Build().HelmChart(name, version)

	_, err := dag.Helm(HelmOpts{Version: helmVersion}).
		Login("ghcr.io", githubActor, githubToken).
		Push(chart, "oci://ghcr.io/openmeterio/helm-charts").
		Sync(ctx)
	if err != nil {
		return err
	}

	return nil
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
