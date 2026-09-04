package common

import (
	"github.com/google/wire"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	ledgerbreakage "github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	"github.com/openmeterio/openmeter/openmeter/ledger/creditvoid"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
)

var CustomerBalance = wire.NewSet(
	NewCustomerBalanceService,
	NewCustomerBalanceFacade,
)

func NewCustomerBalanceService(
	creditsConfig config.CreditsConfiguration,
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

	return customerbalance.New(customerbalance.Config{
		AccountResolver:   accountResolver,
		SubAccountService: accountService,
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
