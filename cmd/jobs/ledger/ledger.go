package ledger

import (
	"github.com/spf13/cobra"

	"github.com/openmeterio/openmeter/cmd/jobs/ledger/backfillaccounts"
)

var Cmd = &cobra.Command{
	Use:   "ledger",
	Short: "Ledger operations",
}

func init() {
	Cmd.AddCommand(backfillaccounts.Cmd)
}
