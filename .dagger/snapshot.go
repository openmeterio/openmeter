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
