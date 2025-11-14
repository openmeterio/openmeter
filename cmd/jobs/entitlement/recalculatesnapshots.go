package entitlement

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/openmeterio/openmeter/cmd/jobs/internal"
	"github.com/openmeterio/openmeter/openmeter/entitlement/balanceworker"
)

func NewRecalculateBalanceSnapshotsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "recalculate-balance-snapshots",
		Short: "Recalculate balance snapshots and send the resulting events into the eventbus",
		RunE: func(cmd *cobra.Command, args []string) error {
			recalculator, err := balanceworker.NewRecalculator(balanceworker.RecalculatorOptions{
				Entitlement:            internal.App.EntitlementRegistry,
				EventBus:               internal.App.EventPublisher,
				MetricMeter:            internal.App.Meter,
				NotificationService:    internal.App.NotificationService,
				HighWatermarkCacheSize: 100_000,
				Logger:                 internal.App.Logger,
				Customer:               internal.App.Customer,
				Subject:                internal.App.Subject,
			})
			if err != nil {
				return err
			}

			return recalculator.Recalculate(cmd.Context(), "default", time.Now())
		},
	}
}
