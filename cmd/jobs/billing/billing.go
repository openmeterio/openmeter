package billing

import (
	"github.com/spf13/cobra"

	"github.com/openmeterio/openmeter/cmd/jobs/billing/advance"
	"github.com/openmeterio/openmeter/cmd/jobs/billing/collect"
)

var Cmd = &cobra.Command{
	Use:   "billing",
	Short: "Billing operations",
}

func init() {
	Cmd.AddCommand(advance.Cmd)
	Cmd.AddCommand(collect.Cmd)
}
