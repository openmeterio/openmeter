package credit

import (
	"log/slog"
	"time"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/watermill/eventbus"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

func NewCreditConnector(
	grantRepo GrantRepo,
	balanceSnapshotRepo BalanceSnapshotRepo,
	ownerConnector OwnerConnector,
	streamingConnector streaming.Connector,
	logger *slog.Logger,
	granularity time.Duration,
	publisher eventbus.Publisher,
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
