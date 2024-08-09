package entitlement

import "github.com/spf13/cobra"

func RootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "entitlement",
		Short: "Entitlement related jobs",
	}

	cmd.AddCommand(NewRecalculateBalanceSnapshotsCommand())

	return cmd
}
