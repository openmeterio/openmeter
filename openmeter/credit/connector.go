package credit

import (
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

type CreditConnector interface {
	BalanceConnector
	GrantConnector
}

type connector struct {
	CreditConnectorConfig
}

type CreditConnectorConfig struct {
	// services
	GrantRepo              grant.Repo
	BalanceSnapshotService balance.SnapshotService
	OwnerConnector         grant.OwnerConnector
	StreamingConnector     streaming.Connector
	Logger                 *slog.Logger
	Tracer                 trace.Tracer
	Publisher              eventbus.Publisher
	TransactionManager     transaction.Creator
	// configuration
	Granularity         time.Duration
	SnapshotGracePeriod datetime.ISODuration
}

func NewCreditConnector(
	cfg CreditConnectorConfig,
) CreditConnector {
	return &connector{
		CreditConnectorConfig: cfg,
	}
}

// balance can be snapshotted if we have a snapshottable value either
// - before the current usage period (period start inclusive)
// - or before the defined grace period
func (c *connector) getSnapshotNotAfter(lastResetAt, at time.Time) time.Time {
	t, _ := c.SnapshotGracePeriod.Negate().AddTo(at)

	if lastResetAt.After(t) {
		t = lastResetAt
	}

	return t
}
