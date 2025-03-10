package credit

import (
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/isodate"
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
	SnapshotGracePeriod isodate.Period
}

func NewCreditConnector(
	cfg CreditConnectorConfig,
) CreditConnector {
	return &connector{
		CreditConnectorConfig: cfg,
	}
}

func (c *connector) getSnapshotBefore(at time.Time) time.Time {
	t, _ := c.SnapshotGracePeriod.Negate().AddTo(at)
	return t
}
