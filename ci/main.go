package main

import (
	"context"
	"fmt"
	"path"

	"golang.org/x/sync/errgroup"
)

const (
	goVersion           = "1.21.5"
	golangciLintVersion = "v1.54.2"
	spectralVersion     = "6.11"
	kafkaVersion        = "3.6"
	clickhouseVersion   = "23.3.9.55"
	redisVersion        = "7.0.12"

	helmDocsVersion = "1.11.3"
	helmVersion     = "3.13.2"
)

type Ci struct{}

func (m *Ci) Ci(ctx context.Context) error {
	test, lint := m.Test(), m.Lint().Go()

	_, err := test.Sync(ctx)
	if err != nil {
		return err
	}

	_, err = lint.Sync(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (m *Ci) Test() *Container {
	return dag.Go(GoOpts{Version: goVersion}).
		WithSource(projectDir()).
		Exec([]string{"go", "test", "-v", "./..."})
}

func (m *Ci) Lint() *Lint {
	return &Lint{}
}

type Lint struct{}

func (m *Lint) All(ctx context.Context) error {
	var group errgroup.Group

	group.Go(func() error {
		_, err := m.Go().Sync(ctx)
		if err != nil {
			return err
		}

		return nil
	})

	group.Go(func() error {
		_, err := m.Openapi().Sync(ctx)
		if err != nil {
			return err
		}

		return nil
	})

	return group.Wait()
}

func (m *Lint) Go() *Container {
	return dag.GolangciLint().
		Run(GolangciLintRunOpts{
			Version:   golangciLintVersion,
			GoVersion: goVersion,
			Source:    projectDir(),
			Verbose:   true,
		})
}

func (m *Lint) Openapi() *Container {
	return dag.Spectral().
		FromVersion(spectralVersion).
		WithSource(projectDir()).
		Lint("api/openapi.yaml")
}

func (m *Ci) Etoe(test Optional[string]) *Container {
	image := m.Build().ContainerImage().
		WithExposedPort(10000).
		WithMountedFile("/etc/openmeter/config.yaml", dag.Host().File(path.Join(root(), "e2e", "config.yaml"))).
		WithServiceBinding("kafka", dag.Kafka().FromVersion(kafkaVersion).Service()).
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
			WithSource(projectDir()).
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
		chart := m.Build().HelmChart(Opt(version))

		_, err := dag.Helm().FromVersion(helmVersion).
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
