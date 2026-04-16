package common

import (
	"github.com/google/wire"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
)

var CustomerBalance = wire.NewSet(
	NewCustomerBalanceService,
	NewCustomerBalanceFacade,
)

func NewCustomerBalanceService(
	creditsConfig config.CreditsConfiguration,
	historicalLedger ledger.Ledger,
	accountResolver ledger.AccountResolver,
	accountService ledgeraccount.Service,
	billingRegistry BillingRegistry,
) (customerbalance.FacadeService, error) {
	if !creditsConfig.Enabled {
		return customerbalance.NewNoopService(), nil
	}

	return customerbalance.New(customerbalance.Config{
		AccountResolver:   accountResolver,
		SubAccountService: accountService,
		ChargesService:    billingRegistry.Charges.Service,
		UsageBasedService: billingRegistry.Charges.UsageBasedService,
		Ledger:            historicalLedger,
	})
}

func NewCustomerBalanceFacade(service customerbalance.FacadeService) (*customerbalance.Facade, error) {
	return customerbalance.NewFacade(service)
}
