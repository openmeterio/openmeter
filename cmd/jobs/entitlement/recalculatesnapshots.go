package entitlement

import (
	"log/slog"

	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel"

	"github.com/openmeterio/openmeter/cmd/jobs/config"
	"github.com/openmeterio/openmeter/openmeter/entitlement/balanceworker"
)

const (
	otelNameRecalculateBalanceSnapshot = "openmeter.io/jobs/entitlement/recalculate-balance-snapshots"
)

func NewRecalculateBalanceSnapshotsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "recalculate-balance-snapshots",
		Short: "Recalculate balance snapshots and send the resulting events into the eventbus",
		RunE: func(cmd *cobra.Command, args []string) error {
			conf, err := config.GetConfig()
			if err != nil {
				return err
			}

			logger := slog.Default()

			meter := otel.GetMeterProvider().Meter(otelNameRecalculateBalanceSnapshot)

			entitlementConnectors, err := initEntitlements(
				cmd.Context(),
				conf,
				logger,
				meter,
				otelNameRecalculateBalanceSnapshot,
			)
			if err != nil {
				return err
			}

			defer entitlementConnectors.Shutdown()

			recalculator, err := balanceworker.NewRecalculator(balanceworker.RecalculatorOptions{
				Entitlement: entitlementConnectors.Registry,
				EventBus:    entitlementConnectors.EventBus,
				Meter:       meter,
			})
			if err != nil {
				return err
			}

			return recalculator.Recalculate(cmd.Context(), "default")
		},
	}
}
