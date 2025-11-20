package cogspgx

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openmeterio/openmeter/pkg/framework/cogs"
	"github.com/openmeterio/openmeter/pkg/framework/pgdriver"
)

type NetworkTimeObserverConfig struct {
	WaitTimeBetweenPings time.Duration
}

// ObserveNetworkTime registers a callback that pings the database.
// The pings are registered by the interceptor on the cogs.postgres.ping_round_trip_ms metric.
// We use the interceptor to avoid double counting the time and use the same code-path for both measurements.
// This is useful to estimate the network time of the database and deduct it from the total round-trip time of the request.
func NewNetworkTimeObserver(metrics *cogs.Metrics, logger *slog.Logger, config NetworkTimeObserverConfig) *NetworkTimeObserver {
	return &NetworkTimeObserver{metrics: metrics, logger: logger, conf: config}
}

type NetworkTimeObserver struct {
	metrics *cogs.Metrics
	logger  *slog.Logger

	conf NetworkTimeObserverConfig

	stop chan struct{}
	done chan struct{}
}

var _ pgdriver.Observer = &NetworkTimeObserver{}

func (o *NetworkTimeObserver) Stop() {
	close(o.stop)
	<-o.done
}

func (o *NetworkTimeObserver) ObservePool(pool *pgxpool.Pool) error {
	// We'll run the ping in a separate goroutine
	o.logger.Info("starting network time observer")

	go func() {
		defer close(o.done)
		for {
			select {
			case <-o.stop:
				o.logger.Info("stopping network time observer")
				return
			default:
				func() {
					ctx := context.WithValue(context.Background(), contextKeyObserverPing, true)

					// We need to use .Query() instead of .Ping() as Tracer doesn't support .Ping()
					r, err := pool.Query(ctx, "SELECT 1")
					if err != nil {
						o.logger.ErrorContext(ctx, "failed to ping database", "error", err)
						return
					}
					defer r.Close()

					for r.Next() {
						// no-op
					}

					o.logger.DebugContext(ctx, "pinged database")

					time.Sleep(o.conf.WaitTimeBetweenPings)
				}()
			}
		}
	}()

	return nil
}
