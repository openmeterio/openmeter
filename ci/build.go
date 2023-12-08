package main

import (
	"context"

	"golang.org/x/sync/errgroup"
)

func (m *Ci) Build() *Build {
	return &Build{}
}

type Build struct{}

func (m *Build) All(ctx context.Context) error {
	var group errgroup.Group

	group.Go(func() error {
		err := m.Binary().All(ctx)
		if err != nil {
			return err
		}

		return nil
	})

	group.Go(func() error {
		_, err := m.ContainerImage().Sync(ctx)
		if err != nil {
			return err
		}

		return nil
	})

	return group.Wait()
}

func (m *Build) ContainerImage() *Container {
	return dag.Container().
		From("alpine:3.18.5@sha256:34871e7290500828b39e22294660bee86d966bc0017544e848dd9a255cdf59e0").
		WithExec([]string{"apk", "add", "--update", "--no-cache", "ca-certificates", "tzdata", "bash"}). // TODO: use apko instead?
		WithMountedFile("/usr/local/bin/openmeter", m.Binary().Api()).
		WithMountedFile("/usr/local/bin/openmeter-sink-worker", m.Binary().SinkWorker())
}

func (m *Build) Binary() *Binary {
	return &Binary{}
}

type Binary struct{}

func (m *Binary) All(ctx context.Context) error {
	var group errgroup.Group

	group.Go(func() error {
		_, err := m.Api().Sync(ctx)
		if err != nil {
			return err
		}

		return nil
	})

	group.Go(func() error {
		_, err := m.SinkWorker().Sync(ctx)
		if err != nil {
			return err
		}

		return nil
	})

	return group.Wait()
}

func (m *Binary) Api() *File {
	return buildDir().DockerBuild().File("/usr/local/bin/openmeter")
}

func (m *Binary) SinkWorker() *File {
	return buildDir().DockerBuild().File("/usr/local/bin/openmeter-sink-worker")
}
