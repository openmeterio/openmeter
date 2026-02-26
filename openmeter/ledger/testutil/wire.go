//go:build wireinject

package testutil

import (
	"log/slog"

	"github.com/google/wire"

	"github.com/openmeterio/openmeter/app/common"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	historical "github.com/openmeterio/openmeter/openmeter/ledger/historical"
	"github.com/openmeterio/openmeter/openmeter/ledger/resolvers"
)

// Deps holds the wired ledger components needed for integration tests.
type Deps struct {
	AccountService   ledgeraccount.Service
	ResolversService *resolvers.Service
	HistoricalLedger *historical.Ledger
}

// InitDeps builds the full ledger stack from an already-open ent client and logger.
func InitDeps(db *entdb.Client, logger *slog.Logger) (Deps, error) {
	wire.Build(
		common.Lockr,
		common.LedgerStack,
		wire.Struct(new(Deps), "*"),
	)
	return Deps{}, nil
}
