package main

import (
	"runtime/debug"

	"github.com/spf13/cobra"
)

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

func versionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("%s version %s (%s) built on %s\n", "Open Meter", version, revision, revisionDate)
		},
	}
}
