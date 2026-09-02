package service

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	flatfeeservice "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee/service"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	usagebasedservice "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service"
	billingratingservice "github.com/openmeterio/openmeter/openmeter/billing/rating/service"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

func TestCustomerChargeCustomCurrencyList(t *testing.T) {
	suite.Run(t, new(CustomerChargeCustomCurrencyListTestSuite))
}

// CustomerChargeCustomCurrencyListTestSuite proves that a custom-currency
// charge, once invoiced, is listed through the customer-charge facade with
// the charge/run side denominated in the custom currency while the booked
// invoice and its lines stay denominated in the invoice's fiat currency.
type CustomerChargeCustomCurrencyListTestSuite struct {
	BaseSuite
}

func (s *CustomerChargeCustomCurrencyListTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()
}

func (s *CustomerChargeCustomCurrencyListTestSuite) TearDownTest() {
	s.BaseSuite.TearDownTest()
}

// enableFlatFeeCustomCurrencyService swaps in a flat-fee service built with a
// mocked lineage dependency, mirroring
// InvoicableChargesTestSuite.enableFlatFeeCustomCurrenciesWithMockLineage,
// which is defined on a different suite type and cannot be called here.
func (s *CustomerChargeCustomCurrencyListTestSuite) enableFlatFeeCustomCurrencyService() {
	s.T().Helper()

	lineageMock := &mockLineageService{Service: s.LineageService}
	lineageMock.On("CreateInitialLineages", mock.Anything, mock.Anything).Return(nil).Maybe()
	lineageMock.On("PersistCorrectionLineageSegments", mock.Anything, mock.Anything).Return(nil).Maybe()
	lineageMock.On("BackfillAdvanceLineageSegments", mock.Anything, mock.Anything).Return(nil).Maybe()

	customCurrencyFlatFeeService, err := flatfeeservice.New(flatfeeservice.Config{
		Adapter:       s.FlatFeeAdapter,
		Handler:       s.FlatFeeTestHandler,
		Lineage:       lineageMock,
		MetaAdapter:   s.MetaAdapter,
		Locker:        s.Locker,
		RatingService: billingratingservice.New(billingratingservice.Config{UnitConfigEnabled: s.UnitConfigEnabled}),
		Currencies:    s.CurrencyService,
	})
	s.Require().NoError(err)

	originalFlatFeeService := s.Charges.flatFeeService
	s.Charges.flatFeeService = customCurrencyFlatFeeService
	s.Require().NoError(s.BillingService.DeregisterLineEngine(billing.LineEngineTypeChargeFlatFee))
	s.Require().NoError(s.BillingService.RegisterLineEngine(customCurrencyFlatFeeService.GetLineEngine()))
	s.T().Cleanup(func() {
		s.Charges.flatFeeService = originalFlatFeeService
		s.Require().NoError(s.BillingService.DeregisterLineEngine(billing.LineEngineTypeChargeFlatFee))
		s.Require().NoError(s.BillingService.RegisterLineEngine(originalFlatFeeService.GetLineEngine()))
	})
}

// enableUsageBasedCustomCurrencyService swaps in a usage-based service built
// with a mocked lineage dependency, mirroring
// UsageBasedChargesTestSuite.useCustomCurrencyUsageBasedServiceWithMockedLineage,
// which is defined on a different suite type and cannot be called here.
func (s *CustomerChargeCustomCurrencyListTestSuite) enableUsageBasedCustomCurrencyService() {
	s.T().Helper()

	lineageMock := &mockLineageService{Service: s.LineageService}
	lineageMock.On("CreateInitialLineages", mock.Anything, mock.Anything).Return(nil).Maybe()
	lineageMock.On("PersistCorrectionLineageSegments", mock.Anything, mock.Anything).Return(nil).Maybe()
	lineageMock.On("BackfillAdvanceLineageSegments", mock.Anything, mock.Anything).Return(nil).Maybe()

	customCurrencyUsageBasedService, err := usagebasedservice.New(usagebasedservice.Config{
		Adapter:                 s.UsageBasedAdapter,
		Handler:                 s.UsageBasedTestHandler,
		Lineage:                 lineageMock,
		Locker:                  s.Locker,
		MetaAdapter:             s.MetaAdapter,
		InvoiceUpdater:          s.InvoiceUpdater,
		CustomerOverrideService: s.BillingService,
		FeatureMeterResolver:    s.FeatureMeterResolver,
		RatingService:           billingratingservice.New(billingratingservice.Config{UnitConfigEnabled: s.UnitConfigEnabled}),
		Currencies:              s.CurrencyService,
		StreamingConnector:      s.MockStreamingConnector,
	})
	s.Require().NoError(err)

	originalUsageBasedService := s.Charges.usageBasedService
	s.Charges.usageBasedService = customCurrencyUsageBasedService
	s.Require().NoError(s.BillingService.DeregisterLineEngine(billing.LineEngineTypeChargeUsageBased))
	s.Require().NoError(s.BillingService.RegisterLineEngine(customCurrencyUsageBasedService.GetLineEngine()))
	s.T().Cleanup(func() {
		s.Charges.usageBasedService = originalUsageBasedService
		s.Require().NoError(s.BillingService.DeregisterLineEngine(billing.LineEngineTypeChargeUsageBased))
		s.Require().NoError(s.BillingService.RegisterLineEngine(originalUsageBasedService.GetLineEngine()))
	})
}

func (s *CustomerChargeCustomCurrencyListTestSuite) TestFlatFeeCustomCurrencyChargeListsInvoiceCurrency() {
	// given:
	// - a 10 TOKENS flat fee settled through credit then invoice with a
	//   manual 0.5 USD cost basis, invoiced and paid with no credit or fiat
	//   overage coverage
	// when:
	// - the customer's charges are listed with the invoice and detailed-line
	//   expands
	// then:
	// - the charge/run side carries TOKENS while the booked invoice and its
	//   lines carry the invoice's USD currency
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-custom-currency-list-flatfee")

	s.enableFlatFeeCustomCurrencyService()

	defaults := s.ProvisionDefaultTaxCodes(ctx, ns)
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	customer := s.CreateTestCustomer(ns, "customer-c1")
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID(), billingtest.WithManualApproval())

	customCurrency := s.createTestCustomCurrency(ctx, ns)
	fiatCurrency, err := currencyx.NewFiatCurrency(USD)
	s.Require().NoError(err)

	createAt := datetime.MustParseTimeInLocation(s.T(), "2024-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2025-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2025-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	clock.FreezeTime(createAt)
	defer clock.UnFreeze()

	costBasisIntent := costbasis.NewIntent(costbasis.ManualIntent{
		FiatCurrency: fiatCurrency,
		Rate:         alpacadecimal.NewFromFloat(0.5),
	})

	fiatOverageCreditsHandler := &flatFeeFiatOverageCreditsHandler{available: alpacadecimal.Zero}
	s.FlatFeeTestHandler.onAllocateFiatOverageCredits = fiatOverageCreditsHandler.Allocate
	s.FlatFeeTestHandler.onCorrectFiatOverageCreditAllocations = fiatOverageCreditsHandler.Correct

	allocationCallback := newCountedCreditAllocationCallback[flatfee.OnAllocateCreditsInput]()
	s.FlatFeeTestHandler.onAllocateCredits = allocationCallback.Handler(
		s.T(),
		func(flatfee.OnAllocateCreditsInput, ledgertransaction.GroupReference) creditrealization.CreateAllocationInputs {
			return nil
		},
	)

	created, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: []charges.ChargeIntent{
			charges.NewChargeIntent(flatfee.Intent{
				Intent: meta.Intent{
					ManagedBy:         billing.SubscriptionManagedLine,
					UniqueReferenceID: lo.ToPtr("custom-currency-list-flatfee"),
					CustomerID:        customer.ID,
					Currency:          customCurrency,
					TaxConfig: productcatalog.TaxCodeConfig{
						TaxCodeID: defaults.InvoicingTaxCodeID,
					},
				},
				IntentMutableFields: flatfee.IntentMutableFields{
					IntentMutableFields: meta.IntentMutableFields{
						Name:              "custom-currency-list-flatfee",
						ServicePeriod:     servicePeriod,
						FullServicePeriod: servicePeriod,
						BillingPeriod:     servicePeriod,
					},
					InvoiceAt:             servicePeriod.To,
					PaymentTerm:           productcatalog.InArrearsPaymentTerm,
					AmountBeforeProration: alpacadecimal.NewFromInt(10),
				},
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				CostBasis:      &costBasisIntent,
			}),
		},
	})
	s.Require().NoError(err)
	s.Require().Len(created, 1)

	clock.FreezeTime(servicePeriod.To)
	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: customer.GetID(),
		AsOf:     lo.ToPtr(servicePeriod.To),
	})
	s.Require().NoError(err)
	s.Require().Len(invoices, 1)
	invoice := invoices[0]
	s.Require().Len(invoice.Lines.OrEmpty(), 1)

	s.FlatFeeTestHandler.onCustomCurrencyOverageAccrued = func(_ context.Context, _ flatfee.OnCustomCurrencyOverageAccruedInput) (flatfee.OnCustomCurrencyOverageAccruedResult, error) {
		return flatfee.OnCustomCurrencyOverageAccruedResult{
			TransactionGroup: ledgertransaction.GroupReference{TransactionGroupID: ulid.Make().String()},
			TotalFiatAmount:  alpacadecimal.NewFromFloat(5),
		}, nil
	}
	s.FlatFeeTestHandler.onPaymentAuthorized = newCountedLedgerTransactionCallback[flatfee.OnPaymentAuthorizedInput]().Handler(s.T())
	s.FlatFeeTestHandler.onPaymentSettled = newCountedLedgerTransactionCallback[flatfee.OnPaymentSettledInput]().Handler(s.T())

	invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
	s.Require().NoError(err)
	s.Equal(billing.StandardInvoiceStatusPaid, invoice.Status)

	// when: the customer's charges are listed with the invoice and
	// detailed-line expands
	result, err := s.Charges.ListCustomerCharges(ctx, charges.ListCustomerChargesInput{
		ListChargesInput: charges.ListChargesInput{
			Page:        pagination.NewPage(1, 10),
			Namespace:   ns,
			CustomerIDs: []string{customer.ID},
			ChargeTypes: []meta.ChargeType{meta.ChargeTypeFlatFee},
			Expands: meta.Expands{
				meta.ExpandRealizationInvoice,
				meta.ExpandDetailedLines,
			},
		},
	})
	s.Require().NoError(err)
	s.Require().Len(result.Charges.Items, 1)
	item := result.Charges.Items[0]

	ff, err := item.AsFlatFeeCharge()
	s.Require().NoError(err)

	// then: the charge/run side carries TOKENS ...
	s.True(ff.Intent.GetCurrency().IsCustom())
	s.Equal(customCurrency.ID, ff.Intent.GetCurrency().ID)
	s.Equal(currencyx.Code("TOKENS"), ff.Intent.GetCurrency().GetCode())

	s.Require().NotNil(item.ResolvedCostBasis)
	s.Equal(float64(0.5), item.ResolvedCostBasis.CostBasis.InexactFloat64())

	costBasisFiatCurrency, err := ff.Intent.GetCostBasisIntent().GetFiatCurrency()
	s.Require().NoError(err)
	s.Equal(USD, costBasisFiatCurrency.Details().Code)

	s.Require().Len(item.FlatFeeRealizations, 1)
	entry := item.FlatFeeRealizations[0]
	s.Require().NotNil(entry.Run)
	s.True(entry.Run.DetailedLines.IsPresent())
	s.Require().Len(entry.Run.DetailedLines.OrEmpty(), 1)
	s.RequireTotals(billingtest.ExpectedTotals{Amount: 10, Total: 10}, entry.Run.DetailedLines.OrEmpty()[0].Totals)

	// ... while the booked invoice carries USD
	s.Require().NotNil(entry.Invoice)
	s.Equal(currencyx.FiatCode(USD), entry.Invoice.Currency)

	// and the invoice line reloaded with the lines expand carries USD too
	reloadedInvoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
		Invoice: invoice.GetInvoiceID(),
		Expand:  billing.StandardInvoiceExpandAll,
	})
	s.Require().NoError(err)
	s.Require().Len(reloadedInvoice.Lines.OrEmpty(), 1)
	s.requireCustomCurrencyOverageLine(requireCustomCurrencyOverageLineInput{
		line:               reloadedInvoice.Lines.OrEmpty()[0],
		expectTokenOverage: 10,
		expectCostBasis:    0.5,
		expectFiatTotals:   billingtest.ExpectedTotals{Amount: 5, Total: 5},
	})
}

func (s *CustomerChargeCustomCurrencyListTestSuite) TestUsageBasedCustomCurrencyChargeListsInvoiceCurrency() {
	// given:
	// - a metered charge priced at 2 TOKENS per unit, settled through credit
	//   then invoice with a manual 0.5 USD cost basis; 5 metered units
	//   produce a 10 TOKENS overage with no credit or fiat overage coverage
	// when:
	// - the realization run is collected, the invoice is approved, and the
	//   customer's charges are listed with the invoice and detailed-line
	//   expands
	// then:
	// - the charge/run side carries TOKENS while the booked invoice and its
	//   lines carry the invoice's USD currency
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-custom-currency-list-usagebased")

	s.enableUsageBasedCustomCurrencyService()

	defaults := s.ProvisionDefaultTaxCodes(ctx, ns)
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	customer := s.CreateTestCustomer(ns, "customer-c1")
	_ = s.ProvisionBillingProfile(
		ctx,
		ns,
		sandboxApp.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P1D")),
		billingtest.WithManualApproval(),
	)

	feature := s.SetupApiRequestsTotalFeature(ctx, ns)
	defer feature.Cleanup()
	customCurrency := s.createTestCustomCurrency(ctx, ns)
	fiatCurrency, err := currencyx.NewFiatCurrency(USD)
	s.Require().NoError(err)

	invoiceAt := datetime.MustParseTimeInLocation(s.T(), "2025-02-01T00:00:00Z", time.UTC).AsTime()
	collectionEnd := datetime.MustParseTimeInLocation(s.T(), "2025-02-02T00:01:00Z", time.UTC).AsTime()
	usageAt := datetime.MustParseTimeInLocation(s.T(), "2025-01-15T00:00:00Z", time.UTC).AsTime()
	createAt := datetime.MustParseTimeInLocation(s.T(), "2024-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2025-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   invoiceAt,
	}

	clock.FreezeTime(createAt)
	defer clock.UnFreeze()

	costBasisIntent := costbasis.NewIntent(costbasis.ManualIntent{
		FiatCurrency: fiatCurrency,
		Rate:         alpacadecimal.NewFromFloat(0.5),
	})
	price := productcatalog.NewPriceFrom(productcatalog.UnitPrice{
		Amount: alpacadecimal.NewFromInt(2),
	})

	fiatOverageCreditsHandler := &usageBasedFiatOverageCreditsHandler{available: alpacadecimal.Zero}
	s.UsageBasedTestHandler.onAllocateFiatOverageCredits = fiatOverageCreditsHandler.Allocate
	s.UsageBasedTestHandler.onCorrectFiatOverageCreditAllocations = fiatOverageCreditsHandler.Correct
	s.UsageBasedTestHandler.onCreditsOnlyUsageAccrued, _ = newCappedCreditAllocator(0)

	created, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: []charges.ChargeIntent{
			charges.NewChargeIntent(usagebased.Intent{
				Intent: meta.Intent{
					ManagedBy:         billing.SubscriptionManagedLine,
					UniqueReferenceID: lo.ToPtr("custom-currency-list-usagebased"),
					CustomerID:        customer.ID,
					Currency:          customCurrency,
					TaxConfig: productcatalog.TaxCodeConfig{
						TaxCodeID: defaults.InvoicingTaxCodeID,
					},
				},
				IntentMutableFields: usagebased.IntentMutableFields{
					IntentMutableFields: meta.IntentMutableFields{
						Name:              "custom-currency-list-usagebased",
						ServicePeriod:     servicePeriod,
						FullServicePeriod: servicePeriod,
						BillingPeriod:     servicePeriod,
					},
					InvoiceAt: servicePeriod.To,
					Price:     *price,
				},
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				FeatureKey:     feature.Feature.Key,
				CostBasis:      &costBasisIntent,
			}),
		},
	})
	s.Require().NoError(err)
	s.Require().Len(created, 1)

	s.MockStreamingConnector.AddSimpleEvent(feature.Feature.Key, 5, usageAt)

	clock.FreezeTime(invoiceAt)
	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: customer.GetID(),
		AsOf:     lo.ToPtr(invoiceAt),
	})
	s.Require().NoError(err)
	s.Require().Len(invoices, 1)
	invoice := invoices[0]
	s.Require().Len(invoice.Lines.OrEmpty(), 1)

	clock.FreezeTime(collectionEnd)
	invoice, err = s.BillingService.AdvanceInvoice(ctx, invoice.GetInvoiceID())
	s.Require().NoError(err)
	s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, invoice.Status)
	s.Require().Len(invoice.Lines.OrEmpty(), 1)

	s.UsageBasedTestHandler.onCustomCurrencyOverageAccrued = func(_ context.Context, _ usagebased.OnCustomCurrencyOverageAccruedInput) (usagebased.OnCustomCurrencyOverageAccruedResult, error) {
		return usagebased.OnCustomCurrencyOverageAccruedResult{
			TransactionGroup: ledgertransaction.GroupReference{TransactionGroupID: ulid.Make().String()},
			TotalFiatAmount:  alpacadecimal.NewFromFloat(5),
		}, nil
	}
	s.UsageBasedTestHandler.onPaymentAuthorized = newCountedLedgerTransactionCallback[usagebased.OnPaymentAuthorizedInput]().Handler(s.T())
	s.UsageBasedTestHandler.onPaymentSettled = newCountedLedgerTransactionCallback[usagebased.OnPaymentSettledInput]().Handler(s.T())

	invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
	s.Require().NoError(err)
	s.Equal(billing.StandardInvoiceStatusPaid, invoice.Status)

	// when: the customer's charges are listed with the invoice and
	// detailed-line expands
	result, err := s.Charges.ListCustomerCharges(ctx, charges.ListCustomerChargesInput{
		ListChargesInput: charges.ListChargesInput{
			Page:        pagination.NewPage(1, 10),
			Namespace:   ns,
			CustomerIDs: []string{customer.ID},
			ChargeTypes: []meta.ChargeType{meta.ChargeTypeUsageBased},
			Expands: meta.Expands{
				meta.ExpandRealizationInvoice,
				meta.ExpandDetailedLines,
			},
		},
	})
	s.Require().NoError(err)
	s.Require().Len(result.Charges.Items, 1)
	item := result.Charges.Items[0]

	ub, err := item.AsUsageBasedCharge()
	s.Require().NoError(err)

	// then: the charge/run side carries TOKENS ...
	s.True(ub.Intent.GetCurrency().IsCustom())
	s.Equal(customCurrency.ID, ub.Intent.GetCurrency().ID)
	s.Equal(currencyx.Code("TOKENS"), ub.Intent.GetCurrency().GetCode())

	s.Require().NotNil(item.ResolvedCostBasis)
	s.Equal(float64(0.5), item.ResolvedCostBasis.CostBasis.InexactFloat64())

	costBasisFiatCurrency, err := ub.Intent.GetCostBasisIntent().GetFiatCurrency()
	s.Require().NoError(err)
	s.Equal(USD, costBasisFiatCurrency.Details().Code)

	s.Require().Len(item.UsageBasedRealizations, 1)
	entry := item.UsageBasedRealizations[0]
	s.Require().NotNil(entry.Run)
	s.True(entry.Run.DetailedLines.IsPresent())
	s.Require().Len(entry.Run.DetailedLines.OrEmpty(), 1)
	s.RequireTotals(billingtest.ExpectedTotals{Amount: 10, Total: 10}, entry.Run.DetailedLines.OrEmpty()[0].Totals)

	// ... while the booked invoice carries USD
	s.Require().NotNil(entry.Invoice)
	s.Equal(currencyx.FiatCode(USD), entry.Invoice.Currency)

	// and the invoice line reloaded with the lines expand carries USD too
	reloadedInvoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
		Invoice: invoice.GetInvoiceID(),
		Expand:  billing.StandardInvoiceExpandAll,
	})
	s.Require().NoError(err)
	s.Require().Len(reloadedInvoice.Lines.OrEmpty(), 1)
	s.requireCustomCurrencyOverageLine(requireCustomCurrencyOverageLineInput{
		line:               reloadedInvoice.Lines.OrEmpty()[0],
		expectTokenOverage: 10,
		expectCostBasis:    0.5,
		expectFiatTotals:   billingtest.ExpectedTotals{Amount: 5, Total: 5},
	})
}
