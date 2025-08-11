package main

import (
	"fmt"

	"github.com/openmeterio/openmeter/.dagger/internal/dagger"
)

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
