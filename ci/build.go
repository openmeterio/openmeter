package main

import (
	"context"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
)

// Build individual artifacts. (Useful for testing and development)
func (m *Ci) Build() *Build {
	return &Build{
		Source: m.Source,
	}
}

type Build struct {
	// +private
	Source *Directory
}

func (m *Build) All(
	ctx context.Context,

	// Target platform in "[os]/[platform]/[version]" format (e.g., "darwin/arm64/v7", "windows/amd64", "linux/arm64").
	// +optional
	platform Platform,
) error {
	var group errgroup.Group

	group.Go(func() error {
		_, err := m.ContainerImage(platform).Sync(ctx)
		if err != nil {
			return err
		}

		return nil
	})

	group.Go(func() error {
		_, err := m.HelmChart("").Sync(ctx)
		if err != nil {
			return err
		}

		return nil
	})

	return group.Wait()
}

func (m *Build) containerImages(version string) []*Container {
	platforms := []Platform{
		"linux/amd64",
		"linux/arm64",
	}

	variants := make([]*Container, 0, len(platforms))

	for _, platform := range platforms {
		variants = append(variants, m.containerImage(platform, version))
	}

	return variants
}

// Build a container image.
func (m *Build) ContainerImage(
	// Target platform in "[os]/[platform]/[version]" format (e.g., "darwin/arm64/v7", "windows/amd64", "linux/arm64").
	// +optional
	platform Platform,
) *Container {
	return m.containerImage(platform, "")
}

func (m *Build) containerImage(platform Platform, version string) *Container {
	return dag.Container(ContainerOpts{Platform: platform}).
		From(alpineBaseImage).
		WithLabel("org.opencontainers.image.title", "openmeter").
		WithLabel("org.opencontainers.image.description", "Cloud Metering for AI, Billing and FinOps. Collect and aggregate millions of usage events in real-time.").
		WithLabel("org.opencontainers.image.url", "https://github.com/openmeterio/openmeter").
		WithLabel("org.opencontainers.image.created", time.Now().String()). // TODO: embed commit timestamp
		WithLabel("org.opencontainers.image.source", "https://github.com/openmeterio/openmeter").
		WithLabel("org.opencontainers.image.licenses", "Apache-2.0").
		With(func(c *Container) *Container {
			if version != "" {
				c = c.WithLabel("org.opencontainers.image.version", version)
			}

			return c
		}).
		WithExec([]string{"apk", "add", "--update", "--no-cache", "ca-certificates", "tzdata", "bash"}).
		WithFile("/usr/local/bin/openmeter", m.Binary().api(platform, version)).
		WithFile("/usr/local/bin/openmeter-sink-worker", m.Binary().sinkWorker(platform, version))
}

// Build binaries.
func (m *Build) Binary() *Binary {
	return &Binary{
		Source: m.Source,
	}
}

type Binary struct {
	// +private
	Source *Directory
}

// Build all binaries.
func (m *Binary) All(
	ctx context.Context,

	// Target platform in "[os]/[platform]/[version]" format (e.g., "darwin/arm64/v7", "windows/amd64", "linux/arm64").
	// +optional
	platform Platform,
) error {
	var group errgroup.Group

	group.Go(func() error {
		_, err := m.Api(platform).Sync(ctx)
		if err != nil {
			return err
		}

		return nil
	})

	group.Go(func() error {
		_, err := m.SinkWorker(platform).Sync(ctx)
		if err != nil {
			return err
		}

		return nil
	})

	return group.Wait()
}

// Build the API server binary.
func (m *Binary) Api(
	// Target platform in "[os]/[platform]/[version]" format (e.g., "darwin/arm64/v7", "windows/amd64", "linux/arm64").
	// +optional
	platform Platform,
) *File {
	return m.api(platform, "")
}

func (m *Binary) api(platform Platform, version string) *File {
	return m.build(platform, version, "./cmd/server")
}

// Build the sink worker binary.
func (m *Binary) SinkWorker(
	// Target platform in "[os]/[platform]/[version]" format (e.g., "darwin/arm64/v7", "windows/amd64", "linux/arm64").
	// +optional
	platform Platform,
) *File {
	return m.sinkWorker(platform, "")
}

func (m *Binary) sinkWorker(platform Platform, version string) *File {
	return m.build(platform, version, "./cmd/sink-worker")
}

func (m *Binary) build(platform Platform, version string, pkg string) *File {
	if version == "" {
		version = "unknown"
	}

	goModule := buildContainer(platform)

	binary := goModule.
		WithSource(m.Source).
		Build(GoWithSourceBuildOpts{
			Pkg:      pkg,
			Trimpath: true,
			Tags:     []string{"musl"},
			RawArgs: []string{
				"-ldflags",
				"-s -w -linkmode external -extldflags \"-static\" -X main.version=" + version,
			},
		})

	return goModule.
		Container().
		WithFile("/out/binary", binary).
		WithExec([]string{"xx-verify", "/out/binary"}).
		File("/out/binary")
}

func buildContainer(platform Platform) *Go {
	return dag.Go(GoOpts{
		Container: dag.Go(GoOpts{Version: goVersion}).
			WithEnvVariable("TARGETPLATFORM", string(platform)).
			WithCgoEnabled().
			Container().
			WithDirectory("/", dag.Container().From(xxBaseImage).Rootfs()).
			WithExec([]string{"apk", "add", "--update", "--no-cache", "ca-certificates", "make", "git", "curl", "clang", "lld"}).
			WithExec([]string{"xx-apk", "add", "--update", "--no-cache", "musl-dev", "gcc"}).
			WithExec([]string{"xx-go", "--wrap"}),
	})
}

func (m *Build) HelmChart(
	// Release version.
	// +optional
	version string,
) *File {
	chart := helmChartDir(m.Source)

	opts := HelmPackageOpts{
		DependencyUpdate: true,
	}

	if version != "" {
		opts.Version = strings.TrimPrefix(version, "v")
		opts.AppVersion = version
	}

	return dag.Helm(HelmOpts{Version: helmVersion}).Package(chart, opts)
}

func helmChartDir(source *Directory) *Directory {
	chart := source.Directory("deploy/charts/openmeter")

	readme := dag.HelmDocs(HelmDocsOpts{Version: helmDocsVersion}).Generate(chart, HelmDocsGenerateOpts{
		Templates: []*File{
			source.File("deploy/charts/template.md"),
			source.File("deploy/charts/openmeter/README.tmpl.md"),
		},
		SortValuesOrder: "file",
	})

	return chart.WithFile("README.md", readme)
}
