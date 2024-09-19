package main

import (
	"fmt"

	"github.com/openmeterio/openmeter/.dagger/internal/dagger"
)

func (m *Openmeter) Test() *dagger.Container {
	// TODO: customize user and password
	postgres := dag.Postgres(dagger.PostgresOpts{
		Version: postgresVersion,
	}).WithDatabase("svix")

	svix := dag.Svix(dagger.SvixOpts{
		Version:   svixVersion,
		Postgres:  postgres.AsSvixPostgres(),
		Database:  "svix",
		JwtSecret: dag.SetSecret("svix-jwt-secret", SvixJWTSingingSecret),
	})

	return goModuleCross("").
		WithSource(m.Source).
		WithEnvVariable("POSTGRES_HOST", "postgres").
		WithEnvVariable("SVIX_HOST", "svix").
		WithEnvVariable("SVIX_JWT_SECRET", SvixJWTSingingSecret).
		WithServiceBinding("postgres", postgres.Service()).
		WithServiceBinding("svix", svix.Service()).
		Exec([]string{"go", "test", "-tags", "musl", "-v", "./..."})
}

func (m *Openmeter) QuickstartTest(
	service *dagger.Service,

	// +default=8888
	port int,
) *dagger.Container {
	return goModule().
		WithModuleCache(cacheVolume("go-mod-quickstart")).
		WithBuildCache(cacheVolume("go-build-quickstart")).
		WithSource(m.Source).
		WithEnvVariable("OPENMETER_ADDRESS", fmt.Sprintf("http://openmeter:%d", port)).
		WithServiceBinding("openmeter", service).
		Exec([]string{"go", "test", "-v", "./quickstart/"})
}
