package main

import (
	"context"
	"errors"
	"fmt"
)

const (
	// Alpine is required for our current build (due to Kafka and CGO), but it doesn't seem to work well with golangci-lint
	goVersion      = "1.22.2"
	goBuildVersion = goVersion + "-alpine3.19@sha256:cdc86d9f363e8786845bea2040312b4efa321b828acdeb26f393faa864d887b0"
	xxBaseImage    = "tonistiigi/xx:1.4.0@sha256:0cd3f05c72d6c9b038eb135f91376ee1169ef3a330d34e418e65e2a5c2e9c0d4"

	golangciLintVersion = "v1.57.2"
	spectralVersion     = "6.11"
	kafkaVersion        = "3.6"
	clickhouseVersion   = "23.3.9.55"
	redisVersion        = "7.0.12"

	helmDocsVersion = "v1.13.1"
	helmVersion     = "3.14.4"

	alpineBaseImage = "alpine:3.19.1@sha256:c5b1261d6d3e43071626931fc004f70149baeba2c8ec672bd4f27761f8e1ad6b"
)

type Ci struct {
	// Project source directory
	// This will become useful once pulling from remote becomes available
	//
	// +private
	Source *Directory
}

func New(
	// Project source directory.
	// +optional
	source *Directory,

	// Checkout the repository (at the designated ref) and use it as the source directory instead of the local one.
	// +optional
	ref string,
) (*Ci, error) {
	if source == nil && ref != "" {
		source = dag.Git("https://github.com/openmeterio/openmeter.git", GitOpts{
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
	)

	return p.wait()
}

func (m *Ci) Test() *Container {
	return dag.Go().
		WithSource(m.Source).
		Exec([]string{"go", "test", "-v", "./..."})
}

func (m *Ci) QuickstartTest(
	service *Service,

	// +default=8888
	port int,
) *Container {
	return dag.Go().
		WithSource(m.Source).
		Container().
		WithServiceBinding("openmeter", service).
		WithEnvVariable("OPENMETER_ADDRESS", fmt.Sprintf("http://openmeter:%d", port)).
		WithExec([]string{"go", "test", "-v", "./quickstart/"})
}
