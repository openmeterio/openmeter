package common

import (
	"github.com/google/wire"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	accountadapter "github.com/openmeterio/openmeter/openmeter/ledger/account/adapter"
	accountservice "github.com/openmeterio/openmeter/openmeter/ledger/account/service"
	historical "github.com/openmeterio/openmeter/openmeter/ledger/historical"
	historicaladapter "github.com/openmeterio/openmeter/openmeter/ledger/historical/adapter"
	"github.com/openmeterio/openmeter/openmeter/ledger/resolvers"
	resolversadapter "github.com/openmeterio/openmeter/openmeter/ledger/resolvers/adapter"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
)

// LedgerStack is the full provider set for the ledger stack.
// Callers must provide *entdb.Client and *lockr.Locker (e.g. via common.Lockr).
var LedgerStack = wire.NewSet(
	NewLedgerAccountRepo,
	NewLedgerHistoricalRepo,
	NewLedgerResolversRepo,
	NewLedgerAccountLiveServices,
	NewLedgerAccountService,
	NewLedgerHistoricalLedger,
	NewLedgerResolversService,
	wire.Bind(new(ledger.Ledger), new(*historical.Ledger)),
	wire.Bind(new(ledger.AccountResolver), new(*resolvers.AccountResolver)),
)

func NewLedgerAccountRepo(db *entdb.Client) ledgeraccount.Repo {
	return accountadapter.NewRepo(db)
}

func NewLedgerHistoricalRepo(db *entdb.Client) historical.Repo {
	return historicaladapter.NewRepo(db)
}

func NewLedgerResolversRepo(db *entdb.Client) resolvers.CustomerAccountRepo {
	return resolversadapter.NewRepo(db)
}

// NewLedgerAccountLiveServices builds AccountLiveServices with the given locker.
// SubAccountService is always self-wired by NewLedgerAccountService; Querier is
// intentionally left nil (only required for GetBalance, not the commit path).
func NewLedgerAccountLiveServices(locker *lockr.Locker) ledgeraccount.AccountLiveServices {
	return ledgeraccount.AccountLiveServices{
		Locker: locker,
	}
}

func NewLedgerAccountService(
	repo ledgeraccount.Repo,
	live ledgeraccount.AccountLiveServices,
) ledgeraccount.Service {
	return accountservice.New(repo, live)
}

func NewLedgerHistoricalLedger(
	repo historical.Repo,
	accountSvc ledgeraccount.Service,
	locker *lockr.Locker,
	routingValidator ledger.RoutingValidator,
) *historical.Ledger {
	return historical.NewLedger(repo, accountSvc, locker, routingValidator)
}

func NewLedgerResolversService(
	accountSvc ledgeraccount.Service,
	repo resolvers.CustomerAccountRepo,
) *resolvers.AccountResolver {
	return resolvers.NewAccountResolver(resolvers.AccountResolverConfig{
		AccountService: accountSvc,
		Repo:           repo,
	})
}
