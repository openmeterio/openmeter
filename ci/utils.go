package main

import (
	"os"
	"path/filepath"
)

func root() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return filepath.Join(wd, "..")
}

func projectDir() *Directory {
	return dag.Host().Directory(root(), HostDirectoryOpts{
		Exclude: []string{
			".direnv",
			".devenv",
			"api/client/node/node_modules",
			"ci",
		},
	})
}
