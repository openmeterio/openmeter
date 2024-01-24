package main

import (
	"context"
	"fmt"

	"github.com/google/go-github/v58/github"
	"github.com/sourcegraph/conc/pool"
	"golang.org/x/oauth2"
)

const (
	// Alpine is required for our current build (due to Kafka and CGO), but it doesn't seem to work well with golangci-lint
	goVersion      = "1.21.5"
	goBuildVersion = goVersion + "-alpine3.18@sha256:d8b99943fb0587b79658af03d4d4e8b57769b21dcf08a8401352a9f2a7228754"

	golangciLintVersion = "v1.55.2"
	spectralVersion     = "6.11"
	kafkaVersion        = "3.6"
	clickhouseVersion   = "23.3.9.55"
	redisVersion        = "7.0.12"

	helmDocsVersion = "v1.11.3"
	helmVersion     = "3.13.2"

	ghVersion = "2.42.1"

	alpineBaseImage = "alpine:3.19.0@sha256:51b67269f354137895d43f3b3d810bfacd3945438e94dc5ac55fdac340352f48"
	xxBaseImage     = "tonistiigi/xx:1.3.0@sha256:904fe94f236d36d65aeb5a2462f88f2c537b8360475f6342e7599194f291fb7e"
)

type Ci struct {
	// Project source directory
	// This will become useful once pulling from remote becomes available
	//
	// +private
	Source *Directory
}

func New(
	// Checkout the repository (at the designated ref) and use it as the source directory instead of the local one.
	// +optional
	checkout string,
) *Ci {
	var source *Directory

	if checkout != "" {
		source = dag.Git("https://github.com/openmeterio/openmeter.git", GitOpts{
			KeepGitDir: true,
		}).Branch(checkout).Tree()
	} else {
		source = projectDir()
	}
	return &Ci{
		Source: source,
	}
}

func (m *Ci) Ci(ctx context.Context, commit string, githubToken *Secret) error {
	token, err := githubToken.Plaintext(ctx)
	if err != nil {
		return err
	}

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	gh := github.NewClient(tc)

	p := pool.New().WithErrors().WithContext(ctx)

	p.Go(syncFunc(m.Test()))
	p.Go(m.Lint().All)

	// TODO: run trivy scan on container(s?)
	// TODO: version should be the commit hash (if any?)?
	p.Go(func(ctx context.Context) error {
		{
			rs := github.RepoStatus{
				State: github.String("pending"),
				// TargetURL:   github.String(""),
				Context:     github.String("continuous-integration/dagger/container-image"),
				Description: github.String("Starting container image build"),
			}

			_, _, err := gh.Repositories.CreateStatus(ctx, "openmeterio", "openmeter", commit, &rs)
			if err != nil {
				fmt.Println("GITHUB STATUS ERROR")
				fmt.Println(err.Error())
			}
		}

		images := m.Build().containerImages("ci")

		for _, image := range images {
			_, err := image.Sync(ctx)
			if err != nil {

				{
					rs := github.RepoStatus{
						State: github.String("failure"),
						// TargetURL:   github.String(""),
						Context:     github.String("continuous-integration/dagger/container-image"),
						Description: github.String("Image build failed!"),
					}

					_, _, err := gh.Repositories.CreateStatus(ctx, "openmeterio", "openmeter", commit, &rs)
					if err != nil {
						fmt.Println("GITHUB STATUS ERROR")
						fmt.Println(err.Error())
					}
				}
				return err
			}
		}

		{
			rs := github.RepoStatus{
				State: github.String("success"),
				// TargetURL:   github.String(""),
				Context:     github.String("continuous-integration/dagger/container-image"),
				Description: github.String("Image build successful!"),
			}

			_, _, err := gh.Repositories.CreateStatus(ctx, "openmeterio", "openmeter", commit, &rs)
			if err != nil {
				fmt.Println("GITHUB STATUS ERROR")
				fmt.Println(err.Error())
			}
		}

		return nil
	})

	// TODO: run trivy scan on helm chart
	p.Go(syncFunc(m.Build().HelmChart("openmeter", "0.0.0")))
	p.Go(syncFunc(m.Build().HelmChart("benthos-collector", "0.0.0")))

	p.Go(func(ctx context.Context) error {
		files := m.releaseAssets("ci")

		for _, file := range files {
			_, err := file.Sync(ctx)
			if err != nil {
				return err
			}
		}

		return nil
	})

	return p.Wait()
}

func (m *Ci) Test() *Container {
	return dag.Go().
		WithSource(m.Source).
		Exec([]string{"go", "test", "-v", "./..."})
}
