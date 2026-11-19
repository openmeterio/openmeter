package subscriptionsync

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/openmeterio/openmeter/cmd/jobs/internal"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/reconciler"
)

var (
	namespaces  []string
	customerIDs []string
	lookback    time.Duration
)

var Cmd = &cobra.Command{
	Use:   "subscriptionsync",
	Short: "Subscription sync operations",
}

func init() {
	Cmd.AddCommand(ListCmd())
	Cmd.AddCommand(AllCmd())
}

var ListCmd = func() *cobra.Command {
	var forceSync bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List subscriptions which need to be synced",
		RunE: func(cmd *cobra.Command, args []string) error {
			subs, err := internal.App.BillingSubscriptionReconciler.ListSubscriptions(cmd.Context(), reconciler.ReconcilerListSubscriptionsInput{
				Namespaces: namespaces,
				Customers:  customerIDs,
				Lookback:   lookback,
				ForceSync:  forceSync,
			})
			if err != nil {
				return err
			}

			for _, sub := range subs {
				fmt.Printf("Namespace: %s ID: %s CustomerID: %s\n", sub.Namespace, sub.ID, sub.CustomerID)
			}

			return nil
		},
	}

	cmd.PersistentFlags().StringSliceVar(&namespaces, "n", nil, "filter by namespaces")
	cmd.PersistentFlags().StringSliceVar(&customerIDs, "c", nil, "filter by customer ids")
	cmd.PersistentFlags().DurationVar(&lookback, "l", reconciler.DefaultLookback, "lookback period")
	cmd.PersistentFlags().BoolVar(&forceSync, "f", false, "list what a forced run would select: every subscription in the lookback regardless of its sync state; a subscription that ended more than 30 days ago is only reconciled by all --force-all; deleted subscriptions with completed cleanup stay excluded")

	return cmd
}

var AllCmd = func() *cobra.Command {
	var force bool
	var forceAll bool

	cmd := &cobra.Command{
		Use:   "all",
		Short: "Sync all subscriptions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return internal.App.BillingSubscriptionReconciler.All(cmd.Context(), reconciler.ReconcilerAllInput{
				ReconcilerListSubscriptionsInput: reconciler.ReconcilerListSubscriptionsInput{
					Namespaces: namespaces,
					Customers:  customerIDs,
					Lookback:   lookback,
					ForceSync:  force || forceAll,
				},
				ForceResyncOldSubscriptions: forceAll,
			})
		},
	}

	cmd.PersistentFlags().StringSliceVar(&namespaces, "n", nil, "filter by namespaces")
	cmd.PersistentFlags().StringSliceVar(&customerIDs, "c", nil, "filter by customer ids")
	cmd.PersistentFlags().DurationVar(&lookback, "l", reconciler.DefaultLookback, "lookback period")
	cmd.PersistentFlags().BoolVar(&force, "f", false, "force reconciliation of every subscription in the lookback regardless of its sync state; skips deleted subscriptions with completed cleanup (delete their v2 sync-state entry to retry)")
	cmd.PersistentFlags().BoolVar(&forceAll, "force-all", false, "force reconciliation of every subscription in the lookback regardless of its sync state or subscription end age; lookback still limits selection; skips deleted subscriptions with completed cleanup (delete their v2 sync-state entry to retry)")

	return cmd
}
