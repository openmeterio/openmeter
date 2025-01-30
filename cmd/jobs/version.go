package main

import (
	"github.com/spf13/cobra"

	"github.com/openmeterio/openmeter/cmd/jobs/internal"
)

func versionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Run: func(cmd *cobra.Command, args []string) {
			version, revision, revisionDate := internal.Version()
			cmd.Printf("%s version %s (%s) built on %s\n", "Open Meter", version, revision, revisionDate)
		},
	}
}
