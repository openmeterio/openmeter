package credit

import (
	"log/slog"
	"time"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/event/publisher"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

func NewCreditConnector(
	grantRepo GrantRepo,
	balanceSnapshotRepo BalanceSnapshotRepo,
	ownerConnector OwnerConnector,
	streamingConnector streaming.Connector,
	logger *slog.Logger,
	granularity time.Duration,
	publisher publisher.TopicPublisher,
) CreditConnector {
	return credit.NewCreditConnector(
		grantRepo,
		balanceSnapshotRepo,
		ownerConnector,
		streamingConnector,
		logger,
		granularity,
		publisher,
	)
}
