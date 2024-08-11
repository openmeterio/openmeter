package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/ci/internal/dagger"
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
	return goModuleCross("").
		WithSource(m.Source).
		Container().
		WithServiceBinding("postgres", postgres()).
		WithEnvVariable("POSTGRES_HOST", "postgres").
		WithExec([]string{"go", "test", "-tags", "musl", "-v", "./..."})
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
