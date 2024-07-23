package main

import (
	"context"
	"strings"
	"time"

	"github.com/sourcegraph/conc/pool"

	"github.com/openmeterio/openmeter/ci/internal/dagger"
)

// Build individual artifacts. (Useful for testing and development)
func (m *Ci) Build() *Build {
	return &Build{
		Source: m.Source,
	}
}

type Build struct {
	// +private
	Source *dagger.Directory
}

func (m *Build) All(
	ctx context.Context,

	// Target platform in "[os]/[platform]/[version]" format (e.g., "darwin/arm64/v7", "windows/amd64", "linux/arm64").
	// +optional
	platform dagger.Platform,
) error {
	p := pool.New().WithErrors().WithContext(ctx)

	p.Go(syncFunc(m.ContainerImage(platform)))
	p.Go(syncFunc(m.HelmChart("openmeter", "")))
	p.Go(syncFunc(m.HelmChart("benthos-collector", "")))

	return p.Wait()
}

func (m *Build) containerImages(version string) []*dagger.Container {
	platforms := []dagger.Platform{
		"linux/amd64",
		"linux/arm64",
	}

	variants := make([]*dagger.Container, 0, len(platforms))

	for _, platform := range platforms {
		variants = append(variants, m.containerImage(platform, version))
	}

	return variants
}

// Build a container image.
func (m *Build) ContainerImage(
	// Target platform in "[os]/[platform]/[version]" format (e.g., "darwin/arm64/v7", "windows/amd64", "linux/arm64").
	// +optional
	platform dagger.Platform,
) *dagger.Container {
	return m.containerImage(platform, "")
}

func (m *Build) containerImage(platform dagger.Platform, version string) *dagger.Container {
	return dag.Container(dagger.ContainerOpts{Platform: platform}).
		From(alpineBaseImage).
		WithExec([]string{"apk", "add", "--update", "--no-cache", "ca-certificates", "tzdata", "bash"}).
		WithLabel("org.opencontainers.image.title", "openmeter").
		WithLabel("org.opencontainers.image.description", "Cloud Metering for AI, Billing and FinOps. Collect and aggregate millions of usage events in real-time.").
		WithLabel("org.opencontainers.image.url", "https://github.com/openmeterio/openmeter").
		WithLabel("org.opencontainers.image.source", "https://github.com/openmeterio/openmeter").
		WithLabel("org.opencontainers.image.licenses", "Apache-2.0").
		With(func(c *dagger.Container) *dagger.Container {
			if version != "" {
				c = c.WithLabel("org.opencontainers.image.version", version)
			}

			return c
		}).
		WithFile("/usr/local/bin/openmeter", m.Binary().api(platform, version)).
		WithFile("/usr/local/bin/openmeter-sink-worker", m.Binary().sinkWorker(platform, version)).
		WithLabel("org.opencontainers.image.created", time.Now().String()) // TODO: embed commit timestamp
}

// Build binaries.
func (m *Build) Binary() *Binary {
	return &Binary{
		Source: m.Source,
	}
}

type Binary struct {
	// +private
	Source *dagger.Directory
}

// Build all binaries.
func (m *Binary) All(
	ctx context.Context,

	// Target platform in "[os]/[platform]/[version]" format (e.g., "darwin/arm64/v7", "windows/amd64", "linux/arm64").
	// +optional
	platform dagger.Platform,
) error {
	p := pool.New().WithErrors().WithContext(ctx)

	p.Go(syncFunc(m.Api(platform)))
	p.Go(syncFunc(m.SinkWorker(platform)))
	p.Go(syncFunc(m.BenthosCollector(platform)))

	return p.Wait()
}

// Build the API server binary.
func (m *Binary) Api(
	// Target platform in "[os]/[platform]/[version]" format (e.g., "darwin/arm64/v7", "windows/amd64", "linux/arm64").
	// +optional
	platform dagger.Platform,
) *dagger.File {
	return m.api(platform, "")
}

func (m *Binary) api(platform dagger.Platform, version string) *dagger.File {
	return m.buildCross(platform, version, "./cmd/server").WithName("server")
}

// Build the sink worker binary.
func (m *Binary) SinkWorker(
	// Target platform in "[os]/[platform]/[version]" format (e.g., "darwin/arm64/v7", "windows/amd64", "linux/arm64").
	// +optional
	platform dagger.Platform,
) *dagger.File {
	return m.sinkWorker(platform, "")
}

func (m *Binary) sinkWorker(platform dagger.Platform, version string) *dagger.File {
	return m.buildCross(platform, version, "./cmd/sink-worker").WithName("sink-worker")
}

func (m *Binary) buildCross(platform dagger.Platform, version string, pkg string) *dagger.File {
	if version == "" {
		version = "unknown"
	}

	goMod := goModuleCross(platform)

	binary := goMod.
		WithSource(m.Source).
		Build(dagger.GoWithSourceBuildOpts{
			Pkg:      pkg,
			Trimpath: true,
			Tags:     []string{"musl"},
			Ldflags: []string{
				"-s", "-w",
				"-linkmode", "external",
				"-extldflags", `"-static"`,
				"-X", "main.version=" + version,
			},
		})

	return goMod.
		Container().
		WithFile("/out/binary", binary).
		WithExec([]string{"xx-verify", "/out/binary"}).
		File("/out/binary")
}

// Build the sink worker binary.
func (m *Binary) BenthosCollector(
	// Target platform in "[os]/[platform]/[version]" format (e.g., "darwin/arm64/v7", "windows/amd64", "linux/arm64").
	// +optional
	platform dagger.Platform,
) *dagger.File {
	return m.benthosCollector(platform, "")
}

func (m *Binary) benthosCollector(platform dagger.Platform, version string) *dagger.File {
	return m.build(platform, version, "./cmd/benthos-collector").WithName("benthos")
}

func (m *Binary) build(platform dagger.Platform, version string, pkg string) *dagger.File {
	if version == "" {
		version = "unknown"
	}

	return goModule().
		WithSource(m.Source).
		WithPlatform(platform).
		Build(dagger.GoWithSourceBuildOpts{
			Pkg:      pkg,
			Trimpath: true,
			Ldflags: []string{
				"-s", "-w",
				"-X", "main.version=" + version,
			},
		})
}

func goModule() *dagger.Go {
	return dag.Go(dagger.GoOpts{Version: goBuildVersion}).
		WithModuleCache(dag.CacheVolume("openmeter-go-mod-v2")).
		WithBuildCache(dag.CacheVolume("openmeter-go-build-v2"))
}

func goModuleCross(platform dagger.Platform) *dagger.Go {
	container := goModule().
		WithCgoEnabled(). // TODO: set env var instead?
		Container().
		With(func(c *dagger.Container) *dagger.Container {
			if platform != "" {
				c = c.WithEnvVariable("TARGETPLATFORM", string(platform))
			}

			return c
		}).
		WithDirectory("/", dag.Container().From(xxBaseImage).Rootfs()).
		WithExec([]string{"apk", "add", "--update", "--no-cache", "ca-certificates", "make", "git", "curl", "clang", "lld"}).
		WithExec([]string{"xx-apk", "add", "--update", "--no-cache", "musl-dev", "gcc"}).
		WithExec([]string{"xx-go", "--wrap"})

	return dag.Go(dagger.GoOpts{Container: container})
}

func (m *Build) HelmChart(
	// Name of the chart to build.
	name string,

	// Release version.
	// +optional
	version string,
) *dagger.File {
	return m.helmChart(name, version).File()
}

func (m *Build) helmChart(name string, version string) *dagger.HelmPackage {
	opts := dagger.HelmChartPackageOpts{
		DependencyUpdate: true,
	}

	if version != "" {
		opts.Version = strings.TrimPrefix(version, "v")
		opts.AppVersion = version
	}

	return helmChart(m.Source, name).Package(opts)
}

func helmChart(source *dagger.Directory, name string) *dagger.HelmChart {
	chart := source.Directory("deploy/charts").Directory(name)

	readme := dag.HelmDocs(dagger.HelmDocsOpts{Version: helmDocsVersion}).Generate(chart, dagger.HelmDocsGenerateOpts{
		Templates: []*dagger.File{
			source.File("deploy/charts/template.md"),
			chart.File("README.tmpl.md"),
		},
		SortValuesOrder: "file",
	})

	chart = chart.WithFile("README.md", readme)

	return dag.Helm(dagger.HelmOpts{Version: helmVersion}).Chart(chart)
}
