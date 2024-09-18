package main

import (
	"context"

	"github.com/openmeterio/openmeter/.dagger/internal/dagger"
)

func (m *Openmeter) Snapshot(ctx context.Context, stainlessToken *dagger.Secret) error {
	p := newPipeline(ctx)

	p.addJobs(func(ctx context.Context) error {
		return nil
		// return m.uploadOpenAPISpecToStainless(ctx, stainlessToken)
	})

	return p.wait()
}

func (m *Openmeter) uploadOpenAPISpecToStainless(ctx context.Context, stainlessToken *dagger.Secret) error {
	_, err := dag.Stainless(stainlessToken).UploadSpec("openmeter", m.Source.File("api/openapi.yaml")).Sync(ctx)

	return err
}
