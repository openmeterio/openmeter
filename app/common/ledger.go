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
	"github.com/openmeterio/openmeter/openmeter/ledger/routingrules"
	"github.com/openmeterio/openmeter/openmeter/namespace"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
)

// LedgerStack is the full provider set for the ledger stack.
// Callers must provide *entdb.Client and *lockr.Locker (e.g. via common.Lockr).
var LedgerStack = wire.NewSet(
	NewLedgerRoutingValidator,
	NewLedgerAccountRepo,
	NewLedgerHistoricalRepo,
	NewLedgerResolversRepo,
	NewLedgerAccountService,
	NewLedgerHistoricalLedger,
	NewLedgerNamespaceHandler,
	NewLedgerResolversService,
	wire.Bind(new(ledger.Ledger), new(*historical.Ledger)),
	wire.Bind(new(ledger.Querier), new(*historical.Ledger)),
	wire.Bind(new(ledger.AccountResolver), new(*resolvers.AccountResolver)),
)

func NewLedgerRoutingValidator() ledger.RoutingValidator {
	return routingrules.DefaultValidator
}

func NewLedgerAccountRepo(db *entdb.Client) ledgeraccount.Repo {
	return accountadapter.NewRepo(db)
}

func NewLedgerHistoricalRepo(db *entdb.Client) historical.Repo {
	return historicaladapter.NewRepo(db)
}

func NewLedgerResolversRepo(db *entdb.Client) resolvers.CustomerAccountRepo {
	return resolversadapter.NewRepo(db)
}

func NewLedgerAccountService(
	repo ledgeraccount.Repo,
	locker *lockr.Locker,
	querier ledger.Querier,
) ledgeraccount.Service {
	return accountservice.New(repo, ledgeraccount.AccountLiveServices{
		Locker:  locker,
		Querier: querier,
	})
}

func NewLedgerHistoricalLedger(
	repo historical.Repo,
	accountRepo ledgeraccount.Repo,
	locker *lockr.Locker,
	routingValidator ledger.RoutingValidator,
) *historical.Ledger {
	// TODO: this is a hack
	// package boundary between account and historical ledger is incorrect, dependency resolution is broken
	accountSvc := accountservice.New(accountRepo, ledgeraccount.AccountLiveServices{
		Locker: locker,
		// Querier: nil, // This is the hack
	})

	return historical.NewLedger(repo, accountSvc, locker, routingValidator)
}

func NewLedgerResolversService(
	accountSvc ledgeraccount.Service,
	repo resolvers.CustomerAccountRepo,
	locker *lockr.Locker,
) *resolvers.AccountResolver {
	return resolvers.NewAccountResolver(resolvers.AccountResolverConfig{
		AccountService: accountSvc,
		Repo:           repo,
		Locker:         locker,
	})
}

func NewLedgerNamespaceHandler(accountResolver *resolvers.AccountResolver) namespace.Handler {
	return resolvers.NewNamespaceHandler(accountResolver)
}
