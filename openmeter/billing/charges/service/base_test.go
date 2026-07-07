package service

import (
	"context"
	"errors"
	"log/slog"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/adapter"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	creditpurchaseadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase/adapter"
	creditpurchaselineengine "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase/lineengine"
	creditpurchaseservice "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase/service"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	flatfeeadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee/adapter"
	flatfeeservice "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee/service"
	lineageadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/lineage/adapter"
	lineageservice "github.com/openmeterio/openmeter/openmeter/billing/charges/lineage/service"
	chargeslinerouter "github.com/openmeterio/openmeter/openmeter/billing/charges/linerouter"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	metaadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/meta/adapter"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	usagebasedadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/adapter"
	usagebasedservice "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service"
	billingratingservice "github.com/openmeterio/openmeter/openmeter/billing/rating/service"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger/recognizer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/featuregate"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

const USD = currencyx.Code(currency.USD)

type BaseSuite struct {
	billingtest.BaseSuite

	// UnitConfigEnabled toggles the unitConfig.enabled rating flag for the charges
	// stack the suite builds. Defaults to false; a derived suite sets it in its own
	// SetupSuite (before calling BaseSuite.SetupSuite) to exercise unit_config rating.
	UnitConfigEnabled bool

	Charges                   *service
	UsageBasedService         usagebased.Service
	FlatFeeTestHandler        *flatFeeTestHandler
	CreditPurchaseTestHandler *creditPurchaseTestHandler
	UsageBasedTestHandler     *usageBasedTestHandler
}

func (s *BaseSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()

	s.FlatFeeTestHandler = newFlatFeeTestHandler()
	s.CreditPurchaseTestHandler = newCreditPurchaseTestHandler()
	s.UsageBasedTestHandler = newUsageBasedTestHandler()

	metaAdapter, err := metaadapter.New(metaadapter.Config{
		Client: s.DBClient,
		Logger: slog.Default(),
	})
	s.NoError(err)

	locker, err := lockr.NewLocker(&lockr.LockerConfig{
		Logger: slog.Default(),
	})
	s.NoError(err)

	lineageAdapter, err := lineageadapter.New(lineageadapter.Config{
		Client: s.DBClient,
	})
	s.NoError(err)

	lineageService, err := lineageservice.New(lineageservice.Config{
		Adapter: lineageAdapter,
	})
	s.NoError(err)

	flatFeeAdapter, err := flatfeeadapter.New(flatfeeadapter.Config{
		Client:      s.DBClient,
		Logger:      slog.Default(),
		MetaAdapter: metaAdapter,
	})
	s.NoError(err)

	flatFeeService, err := flatfeeservice.New(flatfeeservice.Config{
		Adapter:       flatFeeAdapter,
		Handler:       s.FlatFeeTestHandler,
		Lineage:       lineageService,
		MetaAdapter:   metaAdapter,
		Locker:        locker,
		RatingService: billingratingservice.New(billingratingservice.Config{UnitConfigEnabled: s.UnitConfigEnabled}),
	})
	s.NoError(err)

	err = s.BillingService.RegisterLineEngine(flatFeeService.GetLineEngine())
	s.NoError(err)

	usageBasedAdapter, err := usagebasedadapter.New(usagebasedadapter.Config{
		Client:      s.DBClient,
		Logger:      slog.Default(),
		MetaAdapter: metaAdapter,
	})
	s.NoError(err)

	usageBasedService, err := usagebasedservice.New(usagebasedservice.Config{
		Adapter:                 usageBasedAdapter,
		Handler:                 s.UsageBasedTestHandler,
		Lineage:                 lineageService,
		Locker:                  locker,
		MetaAdapter:             metaAdapter,
		CustomerOverrideService: s.BillingService,
		FeatureService:          s.FeatureService,
		RatingService:           billingratingservice.New(billingratingservice.Config{UnitConfigEnabled: s.UnitConfigEnabled}),
		StreamingConnector:      s.MockStreamingConnector,
	})
	s.NoError(err)
	s.UsageBasedService = usageBasedService

	err = s.BillingService.RegisterLineEngine(usageBasedService.GetLineEngine())
	s.NoError(err)

	creditPurchaseAdapter, err := creditpurchaseadapter.New(creditpurchaseadapter.Config{
		Client:      s.DBClient,
		Logger:      slog.Default(),
		MetaAdapter: metaAdapter,
	})
	s.NoError(err)

	creditPurchaseService, err := creditpurchaseservice.New(creditpurchaseservice.Config{
		Adapter:     creditPurchaseAdapter,
		Handler:     s.CreditPurchaseTestHandler,
		Lineage:     lineageService,
		MetaAdapter: metaAdapter,
	})
	s.NoError(err)

	creditPurchaseLineEngine, err := creditpurchaselineengine.New(creditpurchaselineengine.Config{
		RatingService: billingratingservice.New(billingratingservice.Config{UnitConfigEnabled: s.UnitConfigEnabled}),
	})
	s.NoError(err)

	err = s.BillingService.RegisterLineEngine(creditPurchaseLineEngine)
	s.NoError(err)
	createLineRouter, err := chargeslinerouter.New(chargeslinerouter.Config{
		CreditsEnabled:           true,
		CreditThenInvoiceEnabled: true,
		FeatureGate: featuregate.NewFeatureGateChecker(featuregate.NewNoop(), featuregate.Flags{
			featuregate.CtxKeyCredits: string(featuregate.CtxKeyCredits),
		}, map[featuregate.FeatureFlag]bool{featuregate.CtxKeyCredits: true}),
	})
	s.NoError(err)
	err = s.BillingService.RegisterCreateLineRouter(createLineRouter)
	s.NoError(err)

	chargesAdapter, err := adapter.New(adapter.Config{
		Client: s.DBClient,
		Logger: slog.Default(),
	})
	s.NoError(err)

	chargesService, err := New(Config{
		Logger:  slog.Default(),
		Adapter: chargesAdapter,

		FeatureService:        s.FeatureService,
		MetaAdapter:           metaAdapter,
		FlatFeeService:        flatFeeService,
		CreditPurchaseService: creditPurchaseService,
		UsageBasedService:     usageBasedService,
		RecognizerService:     recognizer.NoopService{},

		BillingService: s.BillingService,
		TaxCodeService: s.TaxCodeService,
	})
	s.NoError(err)
	s.Charges = chargesService
}

func (s *BaseSuite) TearDownTest() {
	s.FlatFeeTestHandler.Reset()
	s.CreditPurchaseTestHandler.Reset()
	s.UsageBasedTestHandler.Reset()
	s.MockStreamingConnector.Reset()
	clock.UnFreeze()
	clock.ResetTime()
}

type createMockChargeIntentInput struct {
	customer          customer.CustomerID
	currency          currencyx.Code
	servicePeriod     timeutil.ClosedPeriod
	price             *productcatalog.Price
	unitConfig        *productcatalog.UnitConfig
	featureKey        string
	name              string
	settlementMode    productcatalog.SettlementMode
	managedBy         billing.InvoiceLineManagedBy
	uniqueReferenceID string
	taxConfig         productcatalog.TaxCodeConfig
	proRating         productcatalog.ProRatingConfig
}

func (i *createMockChargeIntentInput) Validate() error {
	if i.price == nil {
		return errors.New("price is required")
	}

	if i.customer.Namespace == "" {
		return errors.New("customer namespace is required")
	}

	if i.customer.ID == "" {
		return errors.New("customer id is required")
	}

	if i.currency == "" {
		return errors.New("currency is required")
	}

	return nil
}

func (s *BaseSuite) createMockChargeIntent(input createMockChargeIntentInput) charges.ChargeIntent {
	s.T().Helper()
	s.NoError(input.Validate())

	isFlatFee := input.price.Type() == productcatalog.FlatPriceType
	invoiceAt := input.servicePeriod.To

	if isFlatFee {
		price, err := input.price.AsFlat()
		s.NoError(err)

		switch price.PaymentTerm {
		case productcatalog.InAdvancePaymentTerm:
			invoiceAt = input.servicePeriod.From
		case productcatalog.InArrearsPaymentTerm:
			invoiceAt = input.servicePeriod.To
		default:
			s.T().Fatalf("invalid payment term: %s", price.PaymentTerm)
		}
	}

	intentMeta := meta.Intent{
		ManagedBy:         input.managedBy,
		UniqueReferenceID: lo.EmptyableToPtr(input.uniqueReferenceID),
		CustomerID:        input.customer.ID,
		Currency:          input.currency,
		TaxConfig:         input.taxConfig,
	}
	intentMutableFields := meta.IntentMutableFields{
		Name:              input.name,
		ServicePeriod:     input.servicePeriod,
		FullServicePeriod: input.servicePeriod,
		BillingPeriod:     input.servicePeriod,
	}

	if isFlatFee {
		price, err := input.price.AsFlat()
		s.NoError(err)

		flatFeeIntent := flatfee.Intent{
			Intent: intentMeta,
			IntentMutableFields: flatfee.IntentMutableFields{
				IntentMutableFields:   intentMutableFields,
				PaymentTerm:           price.PaymentTerm,
				InvoiceAt:             invoiceAt,
				AmountBeforeProration: price.Amount,
				ProRating:             input.proRating,
			},
			FeatureKey:     lo.EmptyableToPtr(input.featureKey),
			SettlementMode: lo.CoalesceOrEmpty(input.settlementMode, productcatalog.CreditThenInvoiceSettlementMode),
		}
		return charges.NewChargeIntent(flatFeeIntent)
	}

	usageBasedIntent := usagebased.Intent{
		Intent:     intentMeta,
		FeatureKey: input.featureKey,
		IntentMutableFields: usagebased.IntentMutableFields{
			IntentMutableFields: intentMutableFields,
			Price:               lo.FromPtr(input.price),
			UnitConfig:          input.unitConfig,
			InvoiceAt:           invoiceAt,
		},
		SettlementMode: lo.CoalesceOrEmpty(input.settlementMode, productcatalog.CreditThenInvoiceSettlementMode),
	}
	return charges.NewChargeIntent(usageBasedIntent)
}

func (s *BaseSuite) grantPromotionalCredits(ctx context.Context, customerID customer.CustomerID, amount float64) []charges.Charge {
	s.T().Helper()

	now := clock.Now()

	intent := CreateCreditPurchaseIntent(s.T(), createCreditPurchaseIntentInput{
		customer: customerID,
		currency: USD,
		amount:   alpacadecimal.NewFromFloat(amount),
		servicePeriod: timeutil.ClosedPeriod{
			From: now,
			To:   now,
		},
		settlement: creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{}),
	})

	res, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: customerID.Namespace,
		Intents: charges.ChargeIntents{
			intent,
		},
	})
	s.NoError(err)
	s.Len(res, 1)

	return res
}

func (s *BaseSuite) mustGetChargeByID(chargeID meta.ChargeID) charges.Charge {
	s.T().Helper()
	charge, err := s.Charges.GetByID(s.T().Context(), charges.GetByIDInput{
		ChargeID: chargeID,
		Expands: meta.Expands{
			meta.ExpandRealizations,
		},
	})
	s.NoError(err)
	return charge
}
