package credit

import (
	"log/slog"
	"time"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/event/publisher"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

// TODO: adapters have to be exported here

func NewBalanceConnector(
	gc GrantRepo,
	bsc BalanceSnapshotRepo,
	oc OwnerConnector,
	sc streaming.Connector,
	log *slog.Logger,
) BalanceConnector {
	return credit.NewBalanceConnector(gc, bsc, oc, sc, log)
}

func NewGrantConnector(
	oc OwnerConnector,
	db GrantRepo,
	bsdb BalanceSnapshotRepo,
	granularity time.Duration,
	publisher publisher.TopicPublisher,
) GrantConnector {
	return credit.NewGrantConnector(oc, db, bsdb, granularity, publisher)
}
