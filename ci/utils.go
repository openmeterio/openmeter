package main

import (
	"os"
	"path/filepath"
	"slices"
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
