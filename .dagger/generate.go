package main

import (
	"github.com/openmeterio/openmeter/.dagger/internal/dagger"
)

// Generate various artifacts.
func (m *Openmeter) Generate() *Generate {
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
		From(NODEJS_CONTAINER_IMAGE).
		WithExec([]string{"apt", "update"}).
		WithExec([]string{"apt", "install", "-y", "python3", "python3-pip", "python3-venv"}).
		WithExec([]string{"npm", "install", "-g", "autorest"}).
		WithDirectory("/work", m.Source.Directory("api")).
		WithWorkdir("/work/client/python").
		WithExec([]string{"autorest", "config.yaml"}).
		Directory("/work/client/python")
}
