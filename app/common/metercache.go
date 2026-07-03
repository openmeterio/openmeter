package common

import (
	"fmt"
	"log/slog"
	"math/rand/v2"

	"github.com/google/wire"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/meter"
	clickhouseconnector "github.com/openmeterio/openmeter/openmeter/streaming/clickhouse"
	"github.com/openmeterio/openmeter/openmeter/streaming/clickhouse/metercache"
	"github.com/openmeterio/openmeter/pkg/framework/pgdriver"
	"github.com/openmeterio/openmeter/pkg/pglockx"
)

var MeterCache = wire.NewSet(
	NewMeterCacheReconciler,
)

// NewMeterCacheReconciler provides the meter cache lifecycle reconciler. It is always
// constructed — with the cache disabled it idles until closed — so the server's run.Group
// wiring stays unconditional, mirroring the notification event handler.
func NewMeterCacheReconciler(
	conf config.AggregationConfiguration,
	logger *slog.Logger,
	connector *clickhouseconnector.Connector,
	meterService meter.Service,
	driver *pgdriver.Driver,
) (*metercache.Reconciler, error) {
	if !conf.Cache.Enabled {
		return metercache.New(metercache.Config{
			Logger: logger,
		})
	}

	lockConfig := pglockx.Config{
		Owner:             fmt.Sprintf("metercache.reconciler-%v", rand.Int()),
		HeartbeatInterval: pglockx.DefaultHeartbeatInterval,
		LeaseTime:         pglockx.DefaultLeaseTime,
	}

	logger.Debug("initializing meter cache reconciler lock client",
		"lock.leaseTime", lockConfig.LeaseTime, "lock.heartbeatInterval", lockConfig.HeartbeatInterval, "lock.owner", lockConfig.Owner)

	lockClient, err := pglockx.New(driver.DB(), lockConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize meter cache reconciler lock client: %w", err)
	}

	reconciler, err := metercache.New(metercache.Config{
		Enabled:    true,
		Logger:     logger,
		Connector:  connector,
		Meters:     meterService,
		LockClient: lockClient,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize meter cache reconciler: %w", err)
	}

	return reconciler, nil
}
