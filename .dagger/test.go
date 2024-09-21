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
