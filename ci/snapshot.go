package main

import "context"

func (m *Ci) Snapshot(ctx context.Context, stainlessToken *Secret) error {
	p := newPipeline(ctx)

	p.addJobs(func(ctx context.Context) error {
		return m.uploadOpenAPISpecToStainless(ctx, stainlessToken)
	})

	return p.wait()
}

func (m *Ci) uploadOpenAPISpecToStainless(ctx context.Context, stainlessToken *Secret) error {
	_, err := dag.Stainless(stainlessToken).UploadSpec("openmeter", m.Source.File("api/openapi.yaml")).Sync(ctx)

	return err
}
