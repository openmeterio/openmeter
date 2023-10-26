package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

const (
	goVersion           = "1.21.0"
	golangciLintVersion = "1.54.2"
)

type Ci struct{}

func (m *Ci) Check(ctx context.Context) error {
	test, lint := m.Test(), m.Lint()

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

func (m *Ci) Lint() *Container {
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

func root() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return filepath.Join(wd, "..")
}
