package internal

import (
	"runtime/debug"
)

//nolint:gochecknoglobals
var (
	version      string
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

func Version() (string, string, string) {
	return version, revision, revisionDate
}
