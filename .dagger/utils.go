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
	"fmt"

	"github.com/sourcegraph/conc/pool"

	"github.com/openmeterio/openmeter/.dagger/internal/dagger"
)

type syncable[T any] interface {
	Sync(ctx context.Context) (T, error)
}

func sync[T any](ctx context.Context, s syncable[T]) error {
	_, err := s.Sync(ctx)
	if err != nil {
		return err
	}

	return nil
}

func syncFunc[T any](s syncable[T]) func(context.Context) error {
	return func(ctx context.Context) error {
		return sync(ctx, s)
	}
}

func wrapSyncable[T any](s syncable[T]) func(context.Context) error {
	return func(ctx context.Context) error {
		_, err := s.Sync(ctx)
		return err
	}
}

func wrapSyncables[T syncable[T]](ss []T) func(context.Context) error {
	return func(ctx context.Context) error {
		for _, s := range ss {
			_, err := s.Sync(ctx)
			if err != nil {
				return err
			}
		}
		return nil
	}
}

type pipeline struct {
	pool *pool.ContextPool
}

func newPipeline(ctx context.Context) *pipeline {
	return &pipeline{
		pool: pool.New().WithErrors().WithContext(ctx),
	}
}

func (p *pipeline) addJob(job func(context.Context) error) {
	p.pool.Go(job)
}

func (p *pipeline) addJobs(jobs ...func(context.Context) error) {
	for _, job := range jobs {
		p.pool.Go(job)
	}
}

func (p *pipeline) wait() error {
	return p.pool.Wait()
}

func addSyncableStep[T any](p *pipeline, s syncable[T]) {
	p.pool.Go(syncFunc(s))
}

func cacheVolume(name string) *dagger.CacheVolume {
	return dag.CacheVolume(fmt.Sprintf("openmeter-%s", name))
}
