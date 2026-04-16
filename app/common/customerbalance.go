package common

import (
	"log/slog"

	"github.com/google/wire"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
)

var CustomerBalance = wire.NewSet(
	NewCustomerBalanceService,
	NewCustomerBalanceFacade,
)

func NewCustomerBalanceService(
	creditsConfig config.CreditsConfiguration,
	logger *slog.Logger,
	db *entdb.Client,
	locker *lockr.Locker,
	historicalLedger ledger.Ledger,
	accountResolver ledger.AccountResolver,
	accountService ledgeraccount.Service,
	billingRegistry BillingRegistry,
	featureConnector feature.FeatureConnector,
	ratingService rating.Service,
	streamingConnector streaming.Connector,
) (customerbalance.FacadeService, error) {
	if !creditsConfig.Enabled {
		return customerbalance.NewNoopService(), nil
	}

	return customerbalance.New(customerbalance.Config{
		AccountResolver:   accountResolver,
		SubAccountService: accountService,
		ChargesService:    billingRegistry.Charges.Service,
		UsageBasedService: billingRegistry.Charges.UsageBasedService,
	})
}

func NewCustomerBalanceFacade(service customerbalance.FacadeService) (*customerbalance.Facade, error) {
	return customerbalance.NewFacade(service)
}
