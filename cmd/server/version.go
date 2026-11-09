package main

import "github.com/openmeterio/openmeter/pkg/buildinfo"

// Provisioned by ldflags.
var version string

//nolint:gochecknoglobals
var (
	revision     string
	revisionDate string
)

//nolint:gochecknoinits,goconst
func init() {
	if version == "" {
		version = "unknown"
	}

	revision, revisionDate = buildinfo.Revision(revision)
}
