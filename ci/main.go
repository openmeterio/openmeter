package main

import (
	"context"
	"os"
	"path/filepath"

	"golang.org/x/sync/errgroup"
)

const (
	goVersion           = "1.21.0"
	golangciLintVersion = "v1.54.2"
	spectralVersion     = "6.11"
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
	return dag.Go().FromVersion(goVersion).
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

func root() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return filepath.Join(wd, "..")
}

func projectDir() *Directory {
	return dag.Host().Directory(root(), HostDirectoryOpts{
		Exclude: []string{
			".direnv",
			".devenv",
			"api/client/node/node_modules",
		},
	})
}
