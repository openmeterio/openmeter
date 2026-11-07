// Package buildinfo resolves metadata embedded in Go binaries.
package buildinfo

import "runtime/debug"

// Revision uses the supplied commit SHA when available. Go's build info
// supplies the commit timestamp and the revision for native checkouts.
func Revision(linkedRevision string) (string, string) {
	revision := linkedRevision
	revisionDate := "unknown"

	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				if revision == "" {
					revision = setting.Value
				}
			case "vcs.time":
				revisionDate = setting.Value
			}
		}
	}

	if revision == "" {
		revision = "unknown"
	}

	return revision, revisionDate
}
