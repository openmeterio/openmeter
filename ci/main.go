package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/ci/internal/dagger"
)

const (
	// Alpine is required for our current build (due to Kafka and CGO), but it doesn't seem to work well with golangci-lint
	goVersion      = "1.22.5"
	goBuildVersion = goVersion + "-alpine3.19@sha256:0642d4f809abf039440540de1f0e83502401686e3946ed8e7398a1d94648aa6d"
	xxBaseImage    = "tonistiigi/xx:1.4.0@sha256:0cd3f05c72d6c9b038eb135f91376ee1169ef3a330d34e418e65e2a5c2e9c0d4"

	golangciLintVersion = "v1.59.1"
	spectralVersion     = "6.11"
	kafkaVersion        = "3.6"
	clickhouseVersion   = "23.3.9.55"
	redisVersion        = "7.0.12"
	postgresVersion     = "15.3"

	helmDocsVersion = "v1.14.2"
	helmVersion     = "3.15.2"

	alpineBaseImage = "alpine:3.20.1@sha256:b89d9c93e9ed3597455c90a0b88a8bbb5cb7188438f70953fede212a0c4394e0"
)

type Ci struct {
	// Project source directory
	// This will become useful once pulling from remote becomes available
	//
	// +private
	Source *dagger.Directory
}

func New(
	// Project source directory.
	// +optional
	source *dagger.Directory,

	// Checkout the repository (at the designated ref) and use it as the source directory instead of the local one.
	// +optional
	ref string,
) (*Ci, error) {
	if source == nil && ref != "" {
		source = dag.Git("https://github.com/openmeterio/openmeter.git", dagger.GitOpts{
			KeepGitDir: true,
		}).Ref(ref).Tree()
	}

	if source == nil {
		return nil, errors.New("either source or ref is required")
	}

	return &Ci{
		Source: source,
	}, nil
}

func (m *Ci) Ci(ctx context.Context) error {
	p := newPipeline(ctx)

	p.addJobs(
		wrapSyncable(m.Test()),
		m.Lint().All,

		// TODO: run trivy scan on container(s?)
		// TODO: version should be the commit hash (if any?)?
		wrapSyncables(m.Build().containerImages("ci")),

		// TODO: run trivy scan on helm chart
		wrapSyncable(m.Build().helmChart("openmeter", "0.0.0").File()),
		wrapSyncable(m.Build().helmChart("benthos-collector", "0.0.0").File()),

		wrapSyncables(m.releaseAssets("ci")),

		wrapSyncable(m.Generate().PythonSdk()),
		wrapSyncable(m.Generate().NodeSdk()),
		wrapSyncable(m.Generate().WebSdk()),
	)

	return p.wait()
}

func (m *Ci) Test() *dagger.Container {
	return dag.Go().
		WithSource(m.Source).
		Container().
		WithServiceBinding("postgres", postgres()).
		WithEnvVariable("POSTGRES_HOST", "postgres").
		WithExec([]string{"go", "test", "-v", "./..."})
}

func (m *Ci) QuickstartTest(
	service *dagger.Service,

	// +default=8888
	port int,
) *dagger.Container {
	return goModule().
		WithSource(m.Source).
		Container().
		WithServiceBinding("openmeter", service).
		WithEnvVariable("OPENMETER_ADDRESS", fmt.Sprintf("http://openmeter:%d", port)).
		WithExec([]string{"go", "test", "-v", "./quickstart/"})
}
