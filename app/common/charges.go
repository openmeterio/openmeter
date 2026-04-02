package common

import (
	"fmt"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/billing"
	chargesadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/adapter"
	creditpurchaseadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase/adapter"
	creditpurchaseservice "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase/service"
	flatfeeadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee/adapter"
	flatfeeservice "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee/service"
	metaadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/meta/adapter"
	chargesservice "github.com/openmeterio/openmeter/openmeter/billing/charges/service"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	usagebasedadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/adapter"
	usagebasedservice "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	ledgerchargeadapter "github.com/openmeterio/openmeter/openmeter/ledger/chargeadapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
)

// newChargesRegistry constructs the full charges stack.
// Private: must be initialized via NewBillingRegistry.
func newChargesRegistry(
	db *entdb.Client,
	logger *slog.Logger,
	locker *lockr.Locker,
	billingService billing.Service,
	ratingService rating.Service,
	featureService feature.FeatureConnector,
	streamingConnector streaming.Connector,
	ledgerService ledger.Ledger,
	accountResolver ledger.AccountResolver,
	accountService ledgeraccount.Service,
) (*ChargesRegistry, error) {
	metaAdapter, err := metaadapter.New(metaadapter.Config{
		Client: db,
		Logger: logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create charges meta adapter: %w", err)
	}

	// Ledger-backed handlers
	flatFeeHandler := ledgerchargeadapter.NewFlatFeeHandler(ledgerService, accountResolver, accountService)
	creditPurchaseHandler := ledgerchargeadapter.NewCreditPurchaseHandler(ledgerService, accountResolver, accountService)
	usageBasedHandler := usagebased.UnimplementedHandler{}

	// Flat fee
	flatFeeAdapter, err := flatfeeadapter.New(flatfeeadapter.Config{
		Client:      db,
		Logger:      logger,
		MetaAdapter: metaAdapter,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create charges flat fee adapter: %w", err)
	}

	flatFeeSvc, err := flatfeeservice.New(flatfeeservice.Config{
		Adapter:     flatFeeAdapter,
		Handler:     flatFeeHandler,
		MetaAdapter: metaAdapter,
		Locker:      locker,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create charges flat fee service: %w", err)
	}

	// Usage based
	usageBasedAdapter, err := usagebasedadapter.New(usagebasedadapter.Config{
		Client:      db,
		Logger:      logger,
		MetaAdapter: metaAdapter,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create charges usage based adapter: %w", err)
	}

	usageBasedSvc, err := usagebasedservice.New(usagebasedservice.Config{
		Adapter:                 usageBasedAdapter,
		Handler:                 usageBasedHandler,
		Locker:                  locker,
		MetaAdapter:             metaAdapter,
		CustomerOverrideService: billingService,
		FeatureService:          featureService,
		RatingService:           ratingService,
		StreamingConnector:      streamingConnector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create charges usage based service: %w", err)
	}

	// Credit purchase
	creditPurchaseAdapter, err := creditpurchaseadapter.New(creditpurchaseadapter.Config{
		Client:      db,
		Logger:      logger,
		MetaAdapter: metaAdapter,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create charges credit purchase adapter: %w", err)
	}

	creditPurchaseSvc, err := creditpurchaseservice.New(creditpurchaseservice.Config{
		Adapter:     creditPurchaseAdapter,
		Handler:     creditPurchaseHandler,
		MetaAdapter: metaAdapter,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create charges credit purchase service: %w", err)
	}

	// Root charges service
	rootAdapter, err := chargesadapter.New(chargesadapter.Config{
		Client: db,
		Logger: logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create charges adapter: %w", err)
	}

	chargesSvc, err := chargesservice.New(chargesservice.Config{
		Adapter:               rootAdapter,
		MetaAdapter:           metaAdapter,
		FeatureService:        featureService,
		FlatFeeService:        flatFeeSvc,
		CreditPurchaseService: creditPurchaseSvc,
		UsageBasedService:     usageBasedSvc,
		BillingService:        billingService,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create charges service: %w", err)
	}

	return &ChargesRegistry{
		Service:               chargesSvc,
		FlatFeeService:        flatFeeSvc,
		UsageBasedService:     usageBasedSvc,
		CreditPurchaseService: creditPurchaseSvc,
	}, nil
}
