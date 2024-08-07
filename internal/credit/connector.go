package credit

import (
	"log/slog"
	"time"

	"github.com/openmeterio/openmeter/internal/credit/balance"
	"github.com/openmeterio/openmeter/internal/credit/grant"
	"github.com/openmeterio/openmeter/internal/streaming"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
)

type CreditConnector interface {
	BalanceConnector
	GrantConnector
}

type connector struct {
	// grants and balance snapshots are managed in this same package
	grantRepo           grant.Repo
	balanceSnapshotRepo balance.SnapshotRepo
	// external dependencies
	publisher          eventbus.Publisher
	ownerConnector     grant.OwnerConnector
	streamingConnector streaming.Connector
	logger             *slog.Logger
	// configuration
	snapshotGracePeriod time.Duration
	granularity         time.Duration
}

func NewCreditConnector(
	grantRepo grant.Repo,
	balanceSnapshotRepo balance.SnapshotRepo,
	ownerConnector grant.OwnerConnector,
	streamingConnector streaming.Connector,
	logger *slog.Logger,
	granularity time.Duration,
	publisher eventbus.Publisher,
) CreditConnector {
	return &connector{
		grantRepo:           grantRepo,
		balanceSnapshotRepo: balanceSnapshotRepo,
		ownerConnector:      ownerConnector,
		streamingConnector:  streamingConnector,
		logger:              logger,

		publisher: publisher,

		// TODO: make configurable
		granularity:         granularity,
		snapshotGracePeriod: time.Hour,
	}
}
