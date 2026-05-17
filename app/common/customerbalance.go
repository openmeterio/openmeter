package common

import (
	"github.com/google/wire"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	ledgerbreakage "github.com/openmeterio/openmeter/openmeter/ledger/breakage"
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
	breakageService ledgerbreakage.Service,
) (customerbalance.Service, error) {
	if !creditsConfig.Enabled {
		return customerbalance.NewNoopService(), nil
	}

	return customerbalance.New(customerbalance.Config{
		AccountResolver:   accountResolver,
		SubAccountService: accountService,
		ChargesService:    billingRegistry.Charges.Service,
		CreditPurchaseSvc: billingRegistry.Charges.CreditPurchaseService,
		UsageBasedService: billingRegistry.Charges.UsageBasedService,
		Ledger:            historicalLedger,
		BalanceQuerier:    balanceQuerier,
		Breakage:          breakageService,
	})
}

func NewCustomerBalanceFacade(service customerbalance.Service) (*customerbalance.Facade, error) {
	return customerbalance.NewFacade(service)
}
