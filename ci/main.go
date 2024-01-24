package main

import (
	"context"
	"fmt"
	"path"

	"golang.org/x/sync/errgroup"

	"github.com/sourcegraph/conc/pool"
)

const (
	// Alpine is required for our current build (due to Kafka and CGO), but it doesn't seem to work well with golangci-lint
	goVersion      = "1.21.5"
	goBuildVersion = goVersion + "-alpine3.18@sha256:d8b99943fb0587b79658af03d4d4e8b57769b21dcf08a8401352a9f2a7228754"

	golangciLintVersion = "v1.55.2"
	spectralVersion     = "6.11"
	kafkaVersion        = "3.6"
	clickhouseVersion   = "23.3.9.55"
	redisVersion        = "7.0.12"

	helmDocsVersion = "v1.11.3"
	helmVersion     = "3.13.2"

	alpineBaseImage = "alpine:3.19.0@sha256:51b67269f354137895d43f3b3d810bfacd3945438e94dc5ac55fdac340352f48"
	xxBaseImage     = "tonistiigi/xx:1.3.0@sha256:904fe94f236d36d65aeb5a2462f88f2c537b8360475f6342e7599194f291fb7e"
)

type Ci struct {
	// Project source directory
	// This will become useful once pulling from remote becomes available
	//
	// +private
	Source *Directory
}

func New(
	// Checkout the repository (at the designated ref) and use it as the source directory instead of the local one.
	// +optional
	checkout string,
) *Ci {
	var source *Directory

	if checkout != "" {
		source = dag.Git("https://github.com/openmeterio/openmeter.git", GitOpts{
			KeepGitDir: true,
		}).Branch(checkout).Tree()
	} else {
		source = projectDir()
	}
	return &Ci{
		Source: source,
	}
}

func (m *Ci) Ci(ctx context.Context) error {
	p := pool.New().WithErrors().WithContext(ctx)

	p.Go(syncFunc(m.Test()))
	p.Go(m.Lint().All)

	// TODO: run trivy scan on container(s?)
	// TODO: version should be the commit hash (if any?)?
	p.Go(func(ctx context.Context) error {
		images := m.Build().containerImages("ci")

		for _, image := range images {
			_, err := image.Sync(ctx)
			if err != nil {
				return err
			}
		}

		return nil
	})

	// TODO: run trivy scan on helm chart
	p.Go(syncFunc(m.Build().HelmChart("openmeter", "0.0.0")))
	p.Go(syncFunc(m.Build().HelmChart("benthos-collector", "0.0.0")))

	return p.Wait()
}

func (m *Ci) Test() *Container {
	return dag.Go().
		WithSource(m.Source).
		Exec([]string{"go", "test", "-v", "./..."})
}

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

func (m *Ci) Etoe(test Optional[string]) *Container {
	image := m.Build().ContainerImage("").
		WithExposedPort(10000).
		WithMountedFile("/etc/openmeter/config.yaml", dag.Host().File(path.Join(root(), "e2e", "config.yaml"))).
		WithServiceBinding("kafka", dag.Kafka(KafkaOpts{Version: kafkaVersion}).Service()).
		WithServiceBinding("clickhouse", clickhouse())

	api := image.
		WithExposedPort(8080).
		WithExec([]string{"openmeter", "--config", "/etc/openmeter/config.yaml"}).
		AsService()

	sinkWorker := image.
		WithServiceBinding("redis", redis()).
		WithServiceBinding("api", api). // Make sure api is up before starting sink worker
		WithExec([]string{"openmeter-sink-worker", "--config", "/etc/openmeter/config.yaml"}).
		AsService()

	args := []string{"go", "test", "-v"}

	if t, ok := test.Get(); ok {
		args = append(args, "-run", fmt.Sprintf("Test%s", t))
	}

	args = append(args, "./e2e/...")

	return dag.Go(GoOpts{
		Container: dag.Go(GoOpts{Version: goVersion}).
			WithSource(m.Source).
			Container().
			WithServiceBinding("api", api).
			WithServiceBinding("sink-worker", sinkWorker).
			WithEnvVariable("OPENMETER_ADDRESS", "http://api:8080"),
	}).
		Exec(args)
}

func clickhouse() *Service {
	return dag.Container().
		From(fmt.Sprintf("clickhouse/clickhouse-server:%s-alpine", clickhouseVersion)).
		WithEnvVariable("CLICKHOUSE_USER", "default").
		WithEnvVariable("CLICKHOUSE_PASSWORD", "default").
		WithEnvVariable("CLICKHOUSE_DB", "openmeter").
		WithExposedPort(9000).
		WithExposedPort(9009).
		WithExposedPort(8123).
		AsService()
}

func redis() *Service {
	return dag.Container().
		From(fmt.Sprintf("redis:%s-alpine", redisVersion)).
		WithExposedPort(6379).
		AsService()
}

func (m *Ci) Release(ctx context.Context, version string, githubActor string, githubToken *Secret) error {
	var group errgroup.Group

	group.Go(func() error {
		chart := m.Build().HelmChart("openmeter", version)

		_, err := dag.Helm(HelmOpts{Version: helmVersion}).
			Login("ghcr.io", githubActor, githubToken).
			Push(chart, "oci://ghcr.io/openmeterio/helm-charts").
			Sync(ctx)
		if err != nil {
			return err
		}

		return nil
	})

	group.Go(func() error {
		chart := m.Build().HelmChart("benthos-collector", version)

		_, err := dag.Helm(HelmOpts{Version: helmVersion}).
			Login("ghcr.io", githubActor, githubToken).
			Push(chart, "oci://ghcr.io/openmeterio/helm-charts").
			Sync(ctx)
		if err != nil {
			return err
		}

		return nil
	})

	return group.Wait()
}
