package common

import (
	"log/slog"

	"github.com/google/wire"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	ledgerbreakage "github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	"github.com/openmeterio/openmeter/openmeter/ledger/creditvoid"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	"github.com/openmeterio/openmeter/openmeter/ledger/fbo"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
)

var CustomerBalance = wire.NewSet(
	NewCustomerBalanceService,
	NewCustomerBalanceFacade,
)

func NewCustomerBalanceService(
	creditsConfig config.CreditsConfiguration,
	logger *slog.Logger,
	meter metric.Meter,
	historicalLedger ledger.Ledger,
	balanceQuerier ledger.BalanceQuerier,
	accountResolver ledger.AccountResolver,
	accountService ledgeraccount.Service,
	billingRegistry BillingRegistry,
	currencyService currencies.Service,
	breakageService ledgerbreakage.Service,
	creditVoidService creditvoid.Service,
) (customerbalance.Service, error) {
	if !creditsConfig.Enabled {
		return customerbalance.NewNoopService(), nil
	}

	advanceService, err := NewLedgerAdvanceService(logger, historicalLedger, balanceQuerier, accountResolver, accountService, breakageService)
	if err != nil {
		return nil, err
	}

	fboService, err := fbo.NewService(fbo.Config{
		Logger:        logger,
		Meter:         meter,
		Advance:       advanceService,
		Breakage:      breakageService,
		AccountLocker: accountService,
		Dependencies: transactions.ResolverDependencies{
			AccountService: accountResolver,
			AccountCatalog: accountService,
			BalanceQuerier: balanceQuerier,
		},
	})
	if err != nil {
		return nil, err
	}

	return customerbalance.New(customerbalance.Config{
		AccountResolver:   accountResolver,
		FBO:               fboService,
		ChargesService:    billingRegistry.Charges.Service,
		UsageBasedService: billingRegistry.Charges.UsageBasedService,
		Currencies:        currencyService,
		Ledger:            historicalLedger,
		BalanceQuerier:    balanceQuerier,
		Breakage:          breakageService,
		CreditVoid:        creditVoidService,
	})
}

func NewCustomerBalanceFacade(service customerbalance.Service) (*customerbalance.Facade, error) {
	return customerbalance.NewFacade(service)
}
