package credit

import (
	"log/slog"
	"time"

	"github.com/openmeterio/openmeter/internal/credit/balance"
	"github.com/openmeterio/openmeter/internal/credit/grant"
	"github.com/openmeterio/openmeter/internal/event/publisher"
	"github.com/openmeterio/openmeter/internal/streaming"
)

type CreditConnector interface {
	BalanceConnector
	GrantConnector
}

type connector struct {
	// grants and balance snapshots are managed in this same package
	grantRepo           grant.GrantRepo
	balanceSnapshotRepo balance.BalanceSnapshotRepo
	// external dependencies
	publisher          publisher.TopicPublisher
	ownerConnector     grant.OwnerConnector
	streamingConnector streaming.Connector
	logger             *slog.Logger
	// configuration
	snapshotGracePeriod time.Duration
	granularity         time.Duration
}

func NewCreditConnector(
	grantRepo grant.GrantRepo,
	balanceSnapshotRepo balance.BalanceSnapshotRepo,
	ownerConnector grant.OwnerConnector,
	streamingConnector streaming.Connector,
	logger *slog.Logger,
	granularity time.Duration,
	publisher publisher.TopicPublisher,
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
