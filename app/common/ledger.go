package common

import (
	"context"

	"github.com/google/wire"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/customer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	accountadapter "github.com/openmeterio/openmeter/openmeter/ledger/account/adapter"
	accountservice "github.com/openmeterio/openmeter/openmeter/ledger/account/service"
	historical "github.com/openmeterio/openmeter/openmeter/ledger/historical"
	historicaladapter "github.com/openmeterio/openmeter/openmeter/ledger/historical/adapter"
	ledgernoop "github.com/openmeterio/openmeter/openmeter/ledger/noop"
	"github.com/openmeterio/openmeter/openmeter/ledger/resolvers"
	resolversadapter "github.com/openmeterio/openmeter/openmeter/ledger/resolvers/adapter"
	"github.com/openmeterio/openmeter/openmeter/ledger/routingrules"
	"github.com/openmeterio/openmeter/openmeter/namespace"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
)

type ledgerReadWriter interface {
	ledger.Ledger
	ledger.BalanceQuerier
}

type customerLedgerProvisioner interface {
	ledger.AccountResolver
	CreateCustomerAccounts(ctx context.Context, customerID customer.CustomerID) (ledger.CustomerAccounts, error)
}

// LedgerStack is the full provider set for the ledger stack.
// Callers must provide *entdb.Client and *lockr.Locker (e.g. via common.Lockr).
var LedgerStack = wire.NewSet(
	NewLedgerRoutingValidator,
	NewLedgerAccountRepo,
	NewLedgerHistoricalRepo,
	NewLedgerResolversRepo,
	NewLedgerHistoricalLedger,
	NewLedgerAccountService,
	NewLedgerAccountCatalog,
	NewLedgerAccountLocker,
	NewLedgerBalanceQuerier,
	NewLedgerAccountResolver,
	NewLedgerService,
	NewLedgerNamespaceHandler,
	NewLedgerResolversService,
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
	creditsConfig config.CreditsConfiguration,
	repo ledgeraccount.Repo,
	locker *lockr.Locker,
) ledgeraccount.Service {
	if !creditsConfig.Enabled {
		return ledgernoop.AccountService{}
	}

	return accountservice.New(repo, locker)
}

func NewLedgerHistoricalLedger(
	creditsConfig config.CreditsConfiguration,
	repo historical.Repo,
	accountCatalog ledger.AccountCatalog,
	accountLocker ledger.AccountLocker,
	routingValidator ledger.RoutingValidator,
) ledgerReadWriter {
	if !creditsConfig.Enabled {
		return ledgernoop.Ledger{}
	}

	return historical.NewLedger(repo, accountCatalog, accountLocker, routingValidator)
}

func NewLedgerBalanceQuerier(historicalLedger ledgerReadWriter) ledger.BalanceQuerier {
	return historicalLedger
}

func NewLedgerAccountCatalog(accountSvc ledgeraccount.Service) ledger.AccountCatalog {
	return accountSvc
}

func NewLedgerAccountLocker(accountSvc ledgeraccount.Service) ledger.AccountLocker {
	return accountSvc
}

func NewLedgerResolversService(
	creditsConfig config.CreditsConfiguration,
	accountSvc ledgeraccount.Service,
	repo resolvers.CustomerAccountRepo,
	locker *lockr.Locker,
) customerLedgerProvisioner {
	if !creditsConfig.Enabled {
		return ledgernoop.AccountResolver{}
	}

	return resolvers.NewAccountResolver(resolvers.AccountResolverConfig{
		AccountService: accountSvc,
		Repo:           repo,
		Locker:         locker,
	})
}

func NewLedgerAccountResolver(accountResolver customerLedgerProvisioner) ledger.AccountResolver {
	return accountResolver
}

func NewLedgerService(historicalLedger ledgerReadWriter) ledger.Ledger {
	return historicalLedger
}

func NewLedgerNamespaceHandler(accountResolver ledger.AccountResolver) namespace.Handler {
	if _, ok := accountResolver.(ledgernoop.AccountResolver); ok {
		return ledgernoop.NamespaceHandler{}
	}

	return resolvers.NewNamespaceHandler(accountResolver)
}
