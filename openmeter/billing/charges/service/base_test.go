package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/adapter"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	creditpurchaseadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase/adapter"
	creditpurchaseservice "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase/service"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	flatfeeadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee/adapter"
	flatfeeservice "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee/service"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	lineageadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/lineage/adapter"
	lineageservice "github.com/openmeterio/openmeter/openmeter/billing/charges/lineage/service"
	chargeslinerouter "github.com/openmeterio/openmeter/openmeter/billing/charges/linerouter"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	metaadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/meta/adapter"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	usagebasedadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/adapter"
	usagebasedservice "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service"
	billingratingservice "github.com/openmeterio/openmeter/openmeter/billing/rating/service"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	currencyadapter "github.com/openmeterio/openmeter/openmeter/currencies/adapter"
	"github.com/openmeterio/openmeter/openmeter/currencies/currencyresolver"
	currencyservice "github.com/openmeterio/openmeter/openmeter/currencies/service"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/customer"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/ledger/recognizer"
	ledgertestutils "github.com/openmeterio/openmeter/openmeter/ledger/testutils"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	featurepkg "github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
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

	// UseRealRecognizer wires a real, ledger-backed recognizer.Service instead of
	// recognizer.NoopService{}. Defaults to false because most suites in this
	// package drive charges through mock ledger handlers that never persist real
	// customer/business ledger accounts; a derived suite that needs to observe
	// actual recognition behavior sets it in its own SetupSuite (before calling
	// BaseSuite.SetupSuite).
	UseRealRecognizer bool

	Charges                   *service
	UsageBasedService         usagebased.Service
	CurrencyService           currencies.Service
	MetaAdapter               meta.Adapter
	LineageService            lineage.Service
	Locker                    *lockr.Locker
	InvoiceUpdater            invoiceupdater.Updater
	FlatFeeAdapter            flatfee.Adapter
	CreditPurchaseAdapter     creditpurchase.Adapter
	UsageBasedAdapter         usagebased.Adapter
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
	s.MetaAdapter = metaAdapter

	currencyAdapter, err := currencyadapter.New(currencyadapter.Config{
		Client: s.DBClient,
	})
	s.NoError(err)
	currencyService, err := currencyservice.New(currencyAdapter)
	s.NoError(err)
	s.CurrencyService = currencyService

	locker, err := lockr.NewLocker(&lockr.LockerConfig{
		Logger: slog.Default(),
	})
	s.NoError(err)
	s.Locker = locker

	lineageAdapter, err := lineageadapter.New(lineageadapter.Config{
		Client: s.DBClient,
	})
	s.NoError(err)

	lineageService, err := lineageservice.New(lineageservice.Config{
		Adapter: lineageAdapter,
	})
	s.NoError(err)
	s.LineageService = lineageService

	var recognizerService recognizer.Service = recognizer.NoopService{}
	if s.UseRealRecognizer {
		ledgerDeps, err := ledgertestutils.InitDeps(s.DBClient, slog.Default())
		s.NoError(err)

		recognizerService, err = recognizer.NewService(recognizer.Config{
			Ledger: ledgerDeps.HistoricalLedger,
			Dependencies: transactions.ResolverDependencies{
				AccountService: ledgerDeps.ResolversService,
				AccountCatalog: ledgerDeps.AccountService,
				BalanceQuerier: ledgerDeps.HistoricalLedger,
			},
			Lineage:            lineageService,
			TransactionManager: enttx.NewCreator(s.DBClient),
		})
		s.NoError(err)
	}

	flatFeeAdapter, err := flatfeeadapter.New(flatfeeadapter.Config{
		Client:      s.DBClient,
		Logger:      slog.Default(),
		MetaAdapter: metaAdapter,
	})
	s.NoError(err)
	s.FlatFeeAdapter = flatFeeAdapter

	flatFeeService, err := flatfeeservice.New(flatfeeservice.Config{
		Adapter:       flatFeeAdapter,
		Handler:       s.FlatFeeTestHandler,
		Lineage:       lineageService,
		MetaAdapter:   metaAdapter,
		Locker:        locker,
		RatingService: billingratingservice.New(billingratingservice.Config{UnitConfigEnabled: s.UnitConfigEnabled}),
		Currencies:    currencyService,
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
	s.UsageBasedAdapter = usageBasedAdapter

	invoiceUpdater, err := invoiceupdater.New(invoiceupdater.Config{
		BillingService: s.BillingService,
		Logger:         slog.Default(),
	})
	s.NoError(err)
	s.InvoiceUpdater = invoiceUpdater

	usageBasedService, err := usagebasedservice.New(usagebasedservice.Config{
		Adapter:                 usageBasedAdapter,
		Handler:                 s.UsageBasedTestHandler,
		Lineage:                 lineageService,
		Locker:                  locker,
		MetaAdapter:             metaAdapter,
		InvoiceUpdater:          invoiceUpdater,
		CustomerOverrideService: s.BillingService,
		FeatureService:          s.FeatureService,
		RatingService:           billingratingservice.New(billingratingservice.Config{UnitConfigEnabled: s.UnitConfigEnabled}),
		Currencies:              currencyService,
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
	s.CreditPurchaseAdapter = creditPurchaseAdapter

	creditPurchaseService, err := creditpurchaseservice.New(creditpurchaseservice.Config{
		Adapter:     creditPurchaseAdapter,
		Handler:     s.CreditPurchaseTestHandler,
		Lineage:     lineageService,
		MetaAdapter: metaAdapter,
	})
	s.NoError(err)

	err = s.BillingService.RegisterLineEngine(creditPurchaseService.GetLineEngine())
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

	currencyResolver, err := currencyresolver.New(currencyService)
	s.NoError(err)

	chargesService, err := New(Config{
		Logger:  slog.Default(),
		Adapter: chargesAdapter,

		FeatureService:        s.FeatureService,
		MetaAdapter:           metaAdapter,
		FlatFeeService:        flatFeeService,
		CreditPurchaseService: creditPurchaseService,
		UsageBasedService:     usageBasedService,
		RecognizerService:     recognizerService,

		BillingService:   s.BillingService,
		TaxCodeService:   s.TaxCodeService,
		CurrencyResolver: currencyResolver,
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

func (s *BaseSuite) createTestCustomCurrency(ctx context.Context, namespace string) currencies.Currency {
	s.T().Helper()

	currency, err := s.CurrencyService.CreateCurrency(ctx, currencies.CreateCurrencyInput{
		Namespace: namespace,
		CurrencyDetails: currencyx.CurrencyDetails{
			Code:               "TOKENS",
			Name:               "Tokens",
			Symbol:             "T",
			Precision:          3,
			DecimalMark:        ".",
			ThousandsSeparator: ",",
		},
	})
	s.Require().NoError(err)

	return currency
}

type requireCustomCurrencyOverageLineInput struct {
	line               *billing.StandardLine
	expectTokenOverage float64
	expectCostBasis    float64
	expectFiatTotals   billingtest.ExpectedTotals
}

func (s *BaseSuite) requireCustomCurrencyOverageLine(in requireCustomCurrencyOverageLineInput) {
	s.T().Helper()

	s.Equal(currencyx.FiatCode(USD), in.line.Currency)
	switch reason := in.line.Annotations[billing.AnnotationKeyReason].(type) {
	case string:
		s.Equal(billing.AnnotationValueReasonOverage, reason)
	case *string:
		s.Require().NotNil(reason)
		s.Equal(billing.AnnotationValueReasonOverage, *reason)
	default:
		s.Fail("overage reason annotation has an unexpected type")
	}

	s.Require().NotNil(in.line.UsageBased)
	s.Require().NotNil(in.line.UsageBased.Price)
	flatPrice, err := in.line.UsageBased.Price.AsFlat()
	s.Require().NoError(err)
	s.Equal(in.expectFiatTotals.Amount, flatPrice.Amount.InexactFloat64())

	s.Require().Len(in.line.DetailedLines, 1)
	detailedLine := in.line.DetailedLines[0]
	s.Equal(in.expectTokenOverage, detailedLine.Quantity.InexactFloat64())
	s.Equal(in.expectCostBasis, detailedLine.PerUnitAmount.InexactFloat64())
	s.RequireTotals(in.expectFiatTotals, detailedLine.Totals)
	s.RequireTotals(in.expectFiatTotals, in.line.Totals)
}

type requireDeletedCustomCurrencyOverageLineInput struct {
	line             *billing.StandardLine
	expectFiatTotals billingtest.ExpectedTotals
}

func (s *BaseSuite) requireDeletedCustomCurrencyOverageLine(in requireDeletedCustomCurrencyOverageLineInput) {
	s.T().Helper()

	s.Require().NotNil(in.line.DeletedAt)
	s.Equal(currencyx.FiatCode(USD), in.line.Currency)
	switch reason := in.line.Annotations[billing.AnnotationKeyReason].(type) {
	case string:
		s.Equal(billing.AnnotationValueReasonOverage, reason)
	case *string:
		s.Require().NotNil(reason)
		s.Equal(billing.AnnotationValueReasonOverage, *reason)
	default:
		s.Fail("overage reason annotation has an unexpected type")
	}
	s.Empty(in.line.DetailedLines)
	s.RequireTotals(in.expectFiatTotals, in.line.Totals)
}

type createMockChargeIntentInput struct {
	customer            customer.CustomerID
	currency            currencyx.Code
	servicePeriod       timeutil.ClosedPeriod
	price               *productcatalog.Price
	unitConfig          *productcatalog.UnitConfig
	featureKey          string
	name                string
	settlementMode      productcatalog.SettlementMode
	managedBy           billing.InvoiceLineManagedBy
	uniqueReferenceID   string
	taxConfig           productcatalog.TaxCodeConfig
	proRating           productcatalog.ProRatingConfig
	percentageDiscounts *billing.PercentageDiscount
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
		Currency:          currenciestestutils.NewFiatCurrency(s.T(), input.currency),
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
				PercentageDiscounts:   input.percentageDiscounts.CloneOrNil(),
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

func (s *BaseSuite) newFiatCurrency(code currencyx.Code) *currencyx.FiatCurrency {
	s.T().Helper()

	fiatCurrency, err := currencyx.NewFiatCurrency(code)
	s.Require().NoError(err)

	return fiatCurrency
}

func (s *BaseSuite) newUsageBasedIntent(
	customerID string,
	currency currencies.Currency,
	taxCodeID string,
	uniqueReferenceID string,
	featureKey string,
	settlementMode productcatalog.SettlementMode,
	costBasis *costbasis.Intent,
) usagebased.Intent {
	s.T().Helper()

	period := timeutil.ClosedPeriod{
		From: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC),
	}

	return usagebased.Intent{
		Intent: meta.Intent{
			ManagedBy:         billing.ManuallyManagedLine,
			CustomerID:        customerID,
			Currency:          currency,
			TaxConfig:         productcatalog.TaxCodeConfig{TaxCodeID: taxCodeID},
			UniqueReferenceID: lo.ToPtr(uniqueReferenceID),
		},
		IntentMutableFields: usagebased.IntentMutableFields{
			IntentMutableFields: meta.IntentMutableFields{
				Name:              uniqueReferenceID,
				ServicePeriod:     period,
				FullServicePeriod: period,
				BillingPeriod:     period,
			},
			InvoiceAt: period.To,
			Price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromInt(1),
			}),
		},
		SettlementMode: settlementMode,
		FeatureKey:     featureKey,
		CostBasis:      costBasis,
	}
}

func (s *BaseSuite) createFeatureMeters(ctx context.Context, namespace, key string) featurepkg.FeatureMeterCollection {
	s.T().Helper()

	testMeter := newTestMeter(namespace, key+"-meter")
	s.Require().NoError(s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{testMeter}))

	feature, err := s.FeatureService.CreateFeature(ctx, featurepkg.CreateFeatureInputs{
		Namespace: namespace,
		Name:      key,
		Key:       key,
		MeterID:   lo.ToPtr(testMeter.ID),
	})
	s.Require().NoError(err)

	featureMeter := featurepkg.FeatureMeter{
		Feature: feature,
		Meter:   &testMeter,
	}

	return featurepkg.FeatureMeterCollection{
		ByKey: map[string]featurepkg.FeatureMeter{feature.Key: featureMeter},
		ByID:  map[string]featurepkg.FeatureMeter{feature.ID: featureMeter},
	}
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
