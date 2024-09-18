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
