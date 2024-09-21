// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"

	"github.com/openmeterio/openmeter/.dagger/internal/dagger"
)

func (m *Openmeter) Ci(ctx context.Context) (*dagger.Directory, error) {
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
		WithFile("", m.Generate().Openapi()).
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
