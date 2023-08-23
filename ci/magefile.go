//go:build mage
// +build mage

package main

import (
	"context"
	"fmt"
	"os"

	"dagger.io/dagger"
	"github.com/magefile/mage/mg"
)

const (
	goVersion           = "1.21.0"
	golangciLintVersion = "1.54.2"
)

// Run tests
func Test(ctx context.Context) error {
	var clientOpts []dagger.ClientOpt

	if os.Getenv("DEBUG") == "true" {
		clientOpts = append(clientOpts, dagger.WithLogOutput(os.Stderr))
	}

	client, err := dagger.Connect(ctx, clientOpts...)
	if err != nil {
		return err
	}
	defer client.Close()

	c := client.Container().
		From(fmt.Sprintf("golang:%s", goVersion)).
		WithMountedCache("/root/.cache/go-build", client.CacheVolume("go-build")).
		WithMountedCache("/go/pkg/mod", client.CacheVolume("go-mod")).
		WithMountedDirectory("/src", client.Host().Directory(".")).
		WithWorkdir("/src").
		WithExec([]string{"go", "test", "-v", "./..."})

	err = process(ctx, c)
	if err != nil {
		return err
	}

	return nil
}

// Run linter
func Lint(ctx context.Context) error {
	var clientOpts []dagger.ClientOpt

	if os.Getenv("DEBUG") == "true" {
		clientOpts = append(clientOpts, dagger.WithLogOutput(os.Stderr))
	}

	client, err := dagger.Connect(ctx, clientOpts...)
	if err != nil {
		return err
	}
	defer client.Close()

	bin := client.Container().
		From(fmt.Sprintf("docker.io/golangci/golangci-lint:v%s", golangciLintVersion)).
		File("/usr/bin/golangci-lint")

	c := client.Container().
		From(fmt.Sprintf("golang:%s", goVersion)).
		WithMountedCache("/root/.cache/go-build", client.CacheVolume("go-build")).
		WithMountedCache("/go/pkg/mod", client.CacheVolume("go-mod")).
		WithMountedDirectory("/src", client.Host().Directory(".")).
		WithWorkdir("/src").
		WithFile("/usr/local/bin/golangci-lint", bin).
		WithExec([]string{"golangci-lint", "run", "--verbose"})

	err = process(ctx, c)
	if err != nil {
		return err
	}

	return nil
}

func process(ctx context.Context, container *dagger.Container) error {
	output, err := container.Stdout(ctx)

	fmt.Print(output)

	// if err != nil {
	// 	return err
	// }

	erroutput, err := container.Stderr(ctx)

	fmt.Print(erroutput)

	if err != nil {
		return err
	}

	exit, err := container.ExitCode(ctx)
	if err != nil {
		return err
	}

	if exit > 0 {
		return mg.Fatal(exit)
	}

	return nil
}
