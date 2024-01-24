package main

import (
	"context"

	"golang.org/x/sync/errgroup"
)

func (m *Ci) Release(ctx context.Context, version string, githubActor string, githubToken *Secret) error {
	var group errgroup.Group

	group.Go(func() error {
		return m.pushHelmChart(ctx, "openmeter", version, githubActor, githubToken)
	})

	group.Go(func() error {
		return m.pushHelmChart(ctx, "benthos-collector", version, githubActor, githubToken)
	})

	return group.Wait()
}

func (m *Ci) pushHelmChart(ctx context.Context, name string, version string, githubActor string, githubToken *Secret) error {
	chart := m.Build().HelmChart(name, version)

	_, err := dag.Helm(HelmOpts{Version: helmVersion}).
		Login("ghcr.io", githubActor, githubToken).
		Push(chart, "oci://ghcr.io/openmeterio/helm-charts").
		Sync(ctx)
	if err != nil {
		return err
	}

	return nil
}
