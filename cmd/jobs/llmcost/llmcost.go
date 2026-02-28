package llmcost

import (
	"github.com/spf13/cobra"

	"github.com/openmeterio/openmeter/cmd/jobs/internal"
)

var Cmd = &cobra.Command{
	Use:   "llm-cost",
	Short: "LLM cost database operations",
}

func init() {
	Cmd.AddCommand(syncCmd())
}

func syncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Sync LLM cost prices from external sources",
		RunE: func(cmd *cobra.Command, args []string) error {
			return internal.App.LLMCostSyncJob.Run(cmd.Context())
		},
	}
}
