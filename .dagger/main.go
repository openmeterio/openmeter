package main

import (
	"github.com/openmeterio/openmeter/.dagger/internal/dagger"
)

type Openmeter struct {
	// Project source directory
	// This will become useful once pulling from remote becomes available
	//
	// +private
	Source *dagger.Directory
}

func New(
	// Project source directory.
	//
	// +defaultPath="/"
	// +ignore=[".devenv", ".direnv", ".github", ".vscode", "api/client/node/node_modules", "tmp", "go.work", "go.work.sum"]
	source *dagger.Directory,
) *Openmeter {
	return &Openmeter{
		Source: source,
	}
}
