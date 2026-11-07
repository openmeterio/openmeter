package internal

import "github.com/openmeterio/openmeter/pkg/buildinfo"

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

	revision, revisionDate = buildinfo.Revision(revision)
}

func Version() (string, string, string) {
	return version, revision, revisionDate
}
