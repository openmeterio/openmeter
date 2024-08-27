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

func (m *Ci) Ci(ctx context.Context) (*dagger.Directory, error) {
	p := newPipeline(ctx)

	trivy := dag.Trivy(dagger.TrivyOpts{
		Cache:             cacheVolume("trivy"),
		WarmDatabaseCache: true,
	})

	containerImages := m.Build().containerImages("ci")

	helmChartOpenMeter := m.Build().helmChart("openmeter", "0.0.0").File()
	helmChartBenthosCollector := m.Build().helmChart("benthos-collector", "0.0.0").File()
	helmCharts := dag.Directory().WithFiles("", []*dagger.File{helmChartOpenMeter, helmChartBenthosCollector})

	releaseAssets := dag.Directory().WithFiles("", m.releaseAssets("ci"))

	generated := dag.Directory().
		WithDirectory("sdk/python", m.Generate().PythonSdk()).
		WithDirectory("sdk/node", m.Generate().NodeSdk()).
		WithDirectory("sdk/web", m.Generate().WebSdk())

	dir := dag.Directory().
		WithFile("scans/image.sarif", trivy.Container(containerImages[0]).Report("sarif")).
		WithFile("scans/helm-openmeter.sarif", trivy.HelmChart(helmChartOpenMeter).Report("sarif")).
		WithFile("scans/helm-benthos-collector.sarif", trivy.HelmChart(helmChartBenthosCollector).Report("sarif")).
		WithDirectory("charts/", helmCharts).
		WithDirectory("release/", releaseAssets).
		WithDirectory("generated/", generated)

	p.addJobs(
		m.Generate().Check,
		wrapSyncable(m.Test()),
		m.Lint().All,

		// TODO: version should be the commit hash (if any?)?
		wrapSyncables(m.Build().containerImages("ci")),

		wrapSyncable(dir),
	)

	return dir, p.wait()
}

func (m *Ci) Test() *dagger.Container {
	return goModuleCross("").
		WithSource(m.Source).
		WithEnvVariable("POSTGRES_HOST", "postgres").
		WithEnvVariable("SVIX_HOST", "svix").
		WithEnvVariable("SVIX_JWT_SECRET", SvixJWTSingingSecret).
		WithServiceBinding("postgres", postgres()).
		WithServiceBinding("svix", svix()).
		Exec([]string{"go", "test", "-tags", "musl", "-v", "./..."})
}

func (m *Ci) QuickstartTest(
	service *dagger.Service,

	// +default=8888
	port int,
) *dagger.Container {
	return goModule().
		WithModuleCache(cacheVolume("go-mod-quickstart")).
		WithBuildCache(cacheVolume("go-build-quickstart")).
		WithSource(m.Source).
		WithEnvVariable("OPENMETER_ADDRESS", fmt.Sprintf("http://openmeter:%d", port)).
		WithServiceBinding("openmeter", service).
		Exec([]string{"go", "test", "-v", "./quickstart/"})
}
