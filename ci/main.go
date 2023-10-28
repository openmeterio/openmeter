package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sync/errgroup"
)

const (
	goVersion           = "1.21.0"
	golangciLintVersion = "1.54.2"
	spectralVersion     = "6.11"
)

type Ci struct{}

func (m *Ci) Check(ctx context.Context) error {
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
	host := dag.Host()

	return dag.Container().
		From(fmt.Sprintf("golang:%s", goVersion)).
		WithMountedCache("/root/.cache/go-build", dag.CacheVolume("go-build")).
		WithMountedCache("/go/pkg/mod", dag.CacheVolume("go-mod")).
		WithMountedDirectory("/src", host.Directory(root(), HostDirectoryOpts{
			Exclude: []string{".direnv", ".devenv", "api/client/node/node_modules"},
		})).
		WithWorkdir("/src").
		WithExec([]string{"go", "test", "-v", "./..."})
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
	host := dag.Host()

	bin := dag.Container().
		From(fmt.Sprintf("docker.io/golangci/golangci-lint:v%s", golangciLintVersion)).
		File("/usr/bin/golangci-lint")

	return dag.Container().
		From(fmt.Sprintf("golang:%s", goVersion)).
		WithMountedCache("/root/.cache/go-build", dag.CacheVolume("go-build")).
		WithMountedCache("/go/pkg/mod", dag.CacheVolume("go-mod")).
		WithMountedDirectory("/src", host.Directory(root(), HostDirectoryOpts{
			Exclude: []string{".direnv", ".devenv", "api/client/node/node_modules"},
		})).
		WithWorkdir("/src").
		WithFile("/usr/local/bin/golangci-lint", bin).
		WithExec([]string{"golangci-lint", "run", "--verbose"})
}

func (m *Lint) Openapi() *Container {
	host := dag.Host()

	return dag.Spectral().
		WithVersion(spectralVersion).
		WithSource(host.Directory(root(), HostDirectoryOpts{
			Exclude: []string{".direnv", ".devenv", "api/client/node/node_modules"},
		})).
		Lint("api/openapi.yaml")
}

func root() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return filepath.Join(wd, "..")
}
