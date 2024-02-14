package main

import (
	"context"
	"os"
	"path/filepath"
	"slices"

	"github.com/sourcegraph/conc/pool"
)

func root() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return filepath.Join(wd, "..")
}

// paths to exclude from all contexts
var excludes = []string{
	".direnv",
	".devenv",
	"api/client/node/node_modules",
	"assets",
	"ci",
	"deploy/charts/**/charts",
	"docs",
	"examples",
	"tmp",
}

func exclude(paths ...string) []string {
	return append(slices.Clone(excludes), paths...)
}

func projectDir() *Directory {
	return dag.Host().Directory(root(), HostDirectoryOpts{
		Exclude: exclude(),
	})
}

func buildDir() *Directory {
	return dag.Host().Directory(root(), HostDirectoryOpts{
		Exclude: exclude("e2e"),
	})
}

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

func (p *pipeline) addStep(s func(context.Context) error) {
	p.pool.Go(s)
}

func (p *pipeline) wait() error {
	return p.pool.Wait()
}

func addSyncableStep[T any](p *pipeline, s syncable[T]) {
	p.pool.Go(syncFunc(s))
}
