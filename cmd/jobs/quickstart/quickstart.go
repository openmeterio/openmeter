package quickstart

import (
	"github.com/spf13/cobra"

	"github.com/openmeterio/openmeter/cmd/jobs/billing/subscriptionsync"
)

var Cmd = &cobra.Command{
	Use:   "quickstart",
	Short: "Quickstart operations",
	Long:  "Helpers for docker-compose based quickstart setup. Should not be used in production systems.",
}

func init() {
	Cmd.AddCommand(subscriptionsync.Cmd)
}
