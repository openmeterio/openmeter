package main

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/ci/internal/dagger"
)

// Generate various artifacts.
func (m *Ci) Generate() *Generate {
	return &Generate{
		Source: m.Source,
	}
}

type Generate struct {
	// +private
	Source *dagger.Directory
}

// Generate the Python SDK.
func (m *Generate) PythonSdk() *dagger.Directory {
	// We build our image as the official autorest Dockerfile is outdated
	// and not compatible with the latest autorest.
	// More specifically, the latest autorest npm package depends on
	// other Azure packages that require a higher node version.
	// Official image: https://github.com/Azure/autorest/blob/63ffe68961e24ed8aa59a2ca4c16a8019c271e45/docker/base/ubuntu/Dockerfile

	// Autorest is incompatible with latest node version
	return dag.Container().
		From("node:22.5.1-slim").
		WithExec([]string{"apt", "update"}).
		WithExec([]string{"apt", "install", "-y", "python3", "python3-pip", "python3-venv"}).
		WithExec([]string{"npm", "install", "-g", "autorest"}).
		WithDirectory("/work", m.Source.Directory("api")).
		WithWorkdir("/work/client/python").
		WithExec([]string{"autorest", "config.yaml"}).
		Directory("/work/client/python")
}

// Generate the Node SDK.
func (m *Generate) NodeSdk() *dagger.Directory {
	return dag.Container().
		From("node:20.15.1-alpine3.20").
		WithExec([]string{"npm", "install", "-g", "pnpm"}).
		WithDirectory("/work", m.Source.Directory("api")).
		WithWorkdir("/work/client/node").
		WithExec([]string{"pnpm", "install", "--frozen-lockfile"}).
		WithExec([]string{"pnpm", "run", "generate"}).
		WithExec([]string{"pnpm", "build"}).
		WithExec([]string{"pnpm", "test"}).
		Directory("/work/client/node").
		WithoutDirectory("node_modules")
}

// Generate the Web SDK.
func (m *Generate) WebSdk() *dagger.Directory {
	return dag.Container().
		From("node:20.15.1-alpine3.20").
		WithExec([]string{"npm", "install", "-g", "pnpm"}).
		WithDirectory("/work", m.Source.Directory("api")).
		WithWorkdir("/work/client/web").
		WithExec([]string{"pnpm", "install", "--frozen-lockfile"}).
		WithExec([]string{"pnpm", "run", "generate"}).
		Directory("/work/client/web").
		WithoutDirectory("node_modules")
}

func (m *Generate) Check(ctx context.Context) error {
	result := goModuleCross("").
		WithSource(m.Source).
		WithEnvVariable("GOFLAGS", "-tags=musl").
		Exec([]string{"go", "generate", "-x", "./..."}).
		Directory("")

	err := diff(ctx, m.Source, result)
	if err != nil {
		return fmt.Errorf("go generate wasn't run: %w", err)
	}

	return nil
}

func diff(ctx context.Context, d1, d2 *dagger.Directory) error {
	_, err := dag.Container(dagger.ContainerOpts{Platform: ""}).
		From(alpineBaseImage).
		WithDirectory("src", d1).
		WithDirectory("res", d2).
		WithExec([]string{"diff", "-u", "-r", "-q", "src", "res"}).
		Sync(ctx)

	return err
}
