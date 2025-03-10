package credit

import (
	"log/slog"
	"time"

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
	// grants and balance snapshots are managed in this same package
	grantRepo              grant.Repo
	balanceSnapshotService balance.SnapshotService
	// external dependencies
	transactionManager transaction.Creator
	publisher          eventbus.Publisher
	ownerConnector     grant.OwnerConnector
	streamingConnector streaming.Connector
	logger             *slog.Logger
	// configuration
	granularity         time.Duration
	snapshotGracePeriod isodate.Period
}

func NewCreditConnector(
	grantRepo grant.Repo,
	balanceSnapshotService balance.SnapshotService,
	ownerConnector grant.OwnerConnector,
	streamingConnector streaming.Connector,
	logger *slog.Logger,
	granularity time.Duration,
	publisher eventbus.Publisher,
	transactionManager transaction.Creator,
) CreditConnector {
	return &connector{
		grantRepo:              grantRepo,
		balanceSnapshotService: balanceSnapshotService,
		ownerConnector:         ownerConnector,
		streamingConnector:     streamingConnector,
		logger:                 logger,

		transactionManager: transactionManager,

		publisher: publisher,

		// TODO: make configurable
		granularity:         granularity,
		snapshotGracePeriod: isodate.NewPeriod(0, 0, 1, 0, 0, 0, 0),
	}
}

func (c *connector) getSnapshotBefore(at time.Time) time.Time {
	t, _ := c.snapshotGracePeriod.Negate().AddTo(at)
	return t
}
