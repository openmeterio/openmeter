package subscriptionsync

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/openmeterio/openmeter/cmd/jobs/internal"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/reconciler"
)

const (
	defaultLookback = 24 * time.Hour
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
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List subscriptions which can be synced",
		RunE: func(cmd *cobra.Command, args []string) error {
			subs, err := internal.App.BillingSubscriptionReconciler.ListSubscriptions(cmd.Context(), reconciler.ReconcilerListSubscriptionsInput{
				Namespaces: namespaces,
				Customers:  customerIDs,
				Lookback:   lookback,
			})
			if err != nil {
				return err
			}

			for _, sub := range subs {
				activeTo := ""
				if sub.ActiveTo != nil {
					activeTo = fmt.Sprintf(" ActiveTo: %s", sub.ActiveTo.Format(time.RFC3339))
				}

				fmt.Printf("Namespace: %s ID: %s CustomerID: %s ActiveFrom: %s%s\n",
					sub.Namespace,
					sub.ID,
					sub.CustomerId,
					sub.ActiveFrom.Format(time.RFC3339),
					activeTo)
			}

			return nil
		},
	}

	cmd.PersistentFlags().StringSliceVar(&namespaces, "n", nil, "filter by namespaces")
	cmd.PersistentFlags().StringSliceVar(&customerIDs, "c", nil, "filter by customer ids")
	cmd.PersistentFlags().DurationVar(&lookback, "l", defaultLookback, "lookback period")

	return cmd
}

var AllCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "all",
		Short: "Sync all subscriptions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return internal.App.BillingSubscriptionReconciler.All(cmd.Context(), reconciler.ReconcilerAllInput{
				Namespaces: namespaces,
				Customers:  customerIDs,
				Lookback:   lookback,
			})
		},
	}

	cmd.PersistentFlags().StringSliceVar(&namespaces, "n", nil, "filter by namespaces")
	cmd.PersistentFlags().StringSliceVar(&customerIDs, "c", nil, "filter by customer ids")
	cmd.PersistentFlags().DurationVar(&lookback, "l", defaultLookback, "lookback period")
	return cmd
}
