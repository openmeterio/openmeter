package common

import (
	"fmt"

	"github.com/google/wire"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/billing/creditgrant"
	creditgrantservice "github.com/openmeterio/openmeter/openmeter/billing/creditgrant/service"
	"github.com/openmeterio/openmeter/openmeter/customer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	ledgerbreakage "github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	"github.com/openmeterio/openmeter/openmeter/ledger/creditvoid"
	creditvoidadapter "github.com/openmeterio/openmeter/openmeter/ledger/creditvoid/adapter"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
)

var CreditGrant = wire.NewSet(
	NewCreditVoidService,
	NewCreditGrantService,
)

func NewCreditVoidService(
	creditsConfig config.CreditsConfiguration,
	db *entdb.Client,
	ledgerService ledger.Ledger,
	balanceQuerier ledger.BalanceQuerier,
	accountResolver ledger.AccountResolver,
	accountService ledgeraccount.Service,
	breakageService ledgerbreakage.Service,
) (creditvoid.Service, error) {
	if !creditsConfig.Enabled {
		return creditvoid.NewNoopService(), nil
	}

	adapter, err := creditvoidadapter.New(creditvoidadapter.Config{
		Client: db,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create credit void adapter: %w", err)
	}

	svc, err := creditvoid.NewService(creditvoid.Config{
		Adapter: adapter,
		Ledger:  ledgerService,
		Dependencies: transactions.ResolverDependencies{
			AccountService: accountResolver,
			AccountCatalog: accountService,
			BalanceQuerier: balanceQuerier,
		},
		Breakage:           breakageService,
		AccountLocker:      accountService,
		TransactionManager: enttx.NewCreator(db),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create credit void service: %w", err)
	}

	return svc, nil
}

func NewCreditGrantService(
	db *entdb.Client,
	billingRegistry BillingRegistry,
	customerService customer.Service,
	creditVoidService creditvoid.Service,
) (creditgrant.Service, error) {
	if billingRegistry.Charges == nil {
		return nil, nil
	}

	svc, err := creditgrantservice.New(creditgrantservice.Config{
		CreditPurchaseService: billingRegistry.Charges.CreditPurchaseService,
		ChargesService:        billingRegistry.Charges.Service,
		BillingService:        billingRegistry.Billing,
		CustomerService:       customerService,
		CreditVoidService:     creditVoidService,
		TransactionManager:    enttx.NewCreator(db),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create credit grant service: %w", err)
	}

	return svc, nil
}
