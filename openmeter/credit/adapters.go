package credit

import (
	"log/slog"
	"time"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

// TODO: adapters have to be exported here

func NewBalanceConnector(
	gc GrantDBConnector,
	bsc BalanceSnapshotDBConnector,
	oc OwnerConnector,
	sc streaming.Connector,
	log *slog.Logger,
) BalanceConnector {
	return credit.NewBalanceConnector(gc, bsc, oc, sc, log)
}

func NewGrantConnector(
	oc OwnerConnector,
	db GrantDBConnector,
	bsdb BalanceSnapshotDBConnector,
	granularity time.Duration,
) GrantConnector {
	return credit.NewGrantConnector(oc, db, bsdb, granularity)
}
