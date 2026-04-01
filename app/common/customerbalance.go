package common

import (
	"context"
	"log/slog"

	"github.com/google/wire"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	chargeadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/adapter"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	flatfeeadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee/adapter"
	flatfeeservice "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee/service"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	metaadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/meta/adapter"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	usagebasedadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/adapter"
	usagebasedservice "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	ledgerchargeadapter "github.com/openmeterio/openmeter/openmeter/ledger/chargeadapter"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var CustomerBalance = wire.NewSet(
	NewCustomerBalanceService,
	NewCustomerBalanceFacade,
)

func NewCustomerBalanceService(
	logger *slog.Logger,
	db *entdb.Client,
	locker *lockr.Locker,
	historicalLedger ledger.Ledger,
	accountResolver ledger.AccountResolver,
	accountService ledgeraccount.Service,
	billingService billing.Service,
	featureConnector feature.FeatureConnector,
	ratingService rating.Service,
	streamingConnector streaming.Connector,
) (*customerbalance.Service, error) {
	metaAdapter, err := metaadapter.New(metaadapter.Config{
		Client: db,
		Logger: logger,
	})
	if err != nil {
		return nil, err
	}

	searchAdapter, err := chargeadapter.New(chargeadapter.Config{
		Client: db,
		Logger: logger,
	})
	if err != nil {
		return nil, err
	}

	flatFeeAdapter, err := flatfeeadapter.New(flatfeeadapter.Config{
		Client:      db,
		Logger:      logger,
		MetaAdapter: metaAdapter,
	})
	if err != nil {
		return nil, err
	}

	flatFeeService, err := flatfeeservice.New(flatfeeservice.Config{
		Adapter:     flatFeeAdapter,
		Handler:     ledgerchargeadapter.NewFlatFeeHandler(historicalLedger, accountResolver, accountService),
		MetaAdapter: metaAdapter,
		Locker:      locker,
	})
	if err != nil {
		return nil, err
	}

	usageAdapter, err := usagebasedadapter.New(usagebasedadapter.Config{
		Client:      db,
		Logger:      logger,
		MetaAdapter: metaAdapter,
	})
	if err != nil {
		return nil, err
	}

	usageService, err := usagebasedservice.New(usagebasedservice.Config{
		Adapter:                 usageAdapter,
		Handler:                 usagebased.UnimplementedHandler{},
		Locker:                  locker,
		MetaAdapter:             metaAdapter,
		CustomerOverrideService: billingService,
		FeatureService:          featureConnector,
		RatingService:           ratingService,
		StreamingConnector:      streamingConnector,
	})
	if err != nil {
		return nil, err
	}

	return customerbalance.New(customerbalance.Config{
		AccountResolver:   accountResolver,
		SubAccountService: accountService,
		ChargesService:    customerBalanceChargeStore{search: searchAdapter, flatFeeService: flatFeeService, usageBasedService: usageService},
		UsageBasedService: usageService,
	})
}

func NewCustomerBalanceFacade(service *customerbalance.Service) (*customerbalance.Facade, error) {
	return customerbalance.NewFacade(service)
}

type customerBalanceChargeStore struct {
	search            charges.ChargesSearchAdapter
	flatFeeService    flatfee.Service
	usageBasedService usagebased.Service
}

func (s customerBalanceChargeStore) ListCharges(ctx context.Context, input charges.ListChargesInput) (pagination.Result[charges.Charge], error) {
	searchResult, err := s.search.ListCharges(ctx, input)
	if err != nil {
		return pagination.Result[charges.Charge]{}, err
	}

	flatFeeIDs := make([]string, 0, len(searchResult.Items))
	usageBasedIDs := make([]string, 0, len(searchResult.Items))

	for _, item := range searchResult.Items {
		switch item.Type {
		case meta.ChargeTypeFlatFee:
			flatFeeIDs = append(flatFeeIDs, item.ID)
		case meta.ChargeTypeUsageBased:
			usageBasedIDs = append(usageBasedIDs, item.ID)
		}
	}

	flatFeeCharges, err := s.flatFeeService.GetByIDs(ctx, flatfee.GetByIDsInput{
		Namespace: input.Namespace,
		IDs:       flatFeeIDs,
		Expands:   input.Expands,
	})
	if err != nil {
		return pagination.Result[charges.Charge]{}, err
	}

	usageBasedCharges, err := s.usageBasedService.GetByIDs(ctx, usagebased.GetByIDsInput{
		Namespace: input.Namespace,
		IDs:       usageBasedIDs,
		Expands:   input.Expands,
	})
	if err != nil {
		return pagination.Result[charges.Charge]{}, err
	}

	chargesByID := make(map[string]charges.Charge, len(flatFeeCharges)+len(usageBasedCharges))

	for _, charge := range flatFeeCharges {
		chargesByID[charge.ID] = charges.NewCharge(charge)
	}

	for _, charge := range usageBasedCharges {
		chargesByID[charge.ID] = charges.NewCharge(charge)
	}

	items := make([]charges.Charge, 0, len(searchResult.Items))
	for _, item := range searchResult.Items {
		charge, ok := chargesByID[item.ID]
		if !ok {
			continue
		}

		items = append(items, charge)
	}

	return pagination.Result[charges.Charge]{
		Page:       searchResult.Page,
		TotalCount: searchResult.TotalCount,
		Items:      items,
	}, nil
}
