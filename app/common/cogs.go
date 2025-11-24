package common

import (
	"log/slog"

	"github.com/google/wire"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/pkg/framework/cogs"
	cogspgx "github.com/openmeterio/openmeter/pkg/framework/cogs/pgx"
)

var COGS = wire.NewSet(
	NewMetrics,
	NewNetworkTimeObserver,
)

func NewMetrics(meter metric.Meter) (*cogs.Metrics, error) {
	return cogs.NewMetrics(meter)
}

func NewNetworkTimeObserver(metrics *cogs.Metrics, logger *slog.Logger, config config.COGSConfiguration) *cogspgx.NetworkTimeObserver {
	return cogspgx.NewNetworkTimeObserver(metrics, logger, cogspgx.NetworkTimeObserverConfig{
		WaitTimeBetweenPings: config.PGPollingInterval,
	})
}
