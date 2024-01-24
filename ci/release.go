package main

import (
	"context"

	"golang.org/x/sync/errgroup"
)

func (m *Ci) Release(ctx context.Context, version string, githubActor string, githubToken *Secret) error {
	var group errgroup.Group

	group.Go(func() error {
		chart := m.Build().HelmChart("openmeter", version)

		_, err := dag.Helm(HelmOpts{Version: helmVersion}).
			Login("ghcr.io", githubActor, githubToken).
			Push(chart, "oci://ghcr.io/openmeterio/helm-charts").
			Sync(ctx)
		if err != nil {
			return err
		}

		return nil
	})

	group.Go(func() error {
		chart := m.Build().HelmChart("benthos-collector", version)

		_, err := dag.Helm(HelmOpts{Version: helmVersion}).
			Login("ghcr.io", githubActor, githubToken).
			Push(chart, "oci://ghcr.io/openmeterio/helm-charts").
			Sync(ctx)
		if err != nil {
			return err
		}

		return nil
	})

	return group.Wait()
}
