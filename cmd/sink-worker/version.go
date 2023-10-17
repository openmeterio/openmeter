package main

import "runtime/debug"

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

	buildInfo, _ := debug.ReadBuildInfo()

	revision = "unknown"
	revisionDate = "unknown"

	for _, setting := range buildInfo.Settings {
		if setting.Key == "vcs.revision" {
			revision = setting.Value
		}

		if setting.Key == "vcs.time" {
			revisionDate = setting.Value
		}
	}
}
