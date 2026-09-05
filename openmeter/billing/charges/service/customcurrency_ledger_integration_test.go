package service

import (
	"errors"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

type CustomCurrencyLedgerIntegrationTestSuite struct {
	BaseSuite
}

func TestCustomCurrencyLedgerIntegration(t *testing.T) {
	suite.Run(t, new(CustomCurrencyLedgerIntegrationTestSuite))
}

func (s *CustomCurrencyLedgerIntegrationTestSuite) SetupSuite() {
	s.UseRealLedgerHandlers = true
	s.BaseSuite.SetupSuite()
}

func (s *CustomCurrencyLedgerIntegrationTestSuite) TearDownTest() {
	s.BaseSuite.TearDownTest()
}

func (s *CustomCurrencyLedgerIntegrationTestSuite) TestFlatFeeFinalizationBooksAndCoversCustomCurrencyOverage() {
	createAt := datetime.MustParseTimeInLocation(s.T(), "2024-12-01T00:00:00Z", time.UTC).AsTime()
	clock.FreezeTime(createAt)
	defer clock.UnFreeze()

	// given: a draft 10 TOKENS flat-fee invoice with 3 USD of paid credits
	fixture := s.prepareFlatFeeCustomCurrencyLedgerFixture(createAt)
	mockApp := s.SandboxApp.EnableMock(s.T())
	defer s.SandboxApp.DisableMock()
	invoiceSync := &failFirstInvoiceSync{}
	mockApp.OnUpsertStandardInvoice(invoiceSync.Upsert)
	mockApp.OnFinalizeStandardInvoice(nil)

	// when: finalization prepares the ledger, invoice sync fails, then retry succeeds
	s.finalizeFlatFeeCustomCurrencyInvoice(&fixture)
	s.Equal(1, invoiceSync.Attempts())
	s.retryCustomCurrencyInvoice(&fixture)
	s.Equal(2, invoiceSync.Attempts())
	s.Equal(1, mockApp.FinalizeInvoiceCallCount())

	// then: retry reuses preparation and payment settles only the remaining 2 USD
	s.requireFlatFeeCustomCurrencyPreparationReused(fixture)
	s.payCustomCurrencyInvoice(&fixture)
	s.requireFlatFeeCustomCurrencyPaymentSettled(fixture)
	mockApp.AssertExpectations(s.T())
}

func (s *CustomCurrencyLedgerIntegrationTestSuite) TestFlatFeeDeletionCorrectsCustomCurrencyOverage() {
	createAt := datetime.MustParseTimeInLocation(s.T(), "2024-12-01T00:00:00Z", time.UTC).AsTime()
	clock.FreezeTime(createAt)
	defer clock.UnFreeze()

	// given: a draft 10 TOKENS flat-fee invoice with 3 USD of paid credits
	fixture := s.prepareFlatFeeCustomCurrencyLedgerFixture(createAt)
	mockApp := s.SandboxApp.EnableMock(s.T())
	defer s.SandboxApp.DisableMock()
	invoiceSync := &failFirstInvoiceSync{}
	mockApp.OnUpsertStandardInvoice(invoiceSync.Upsert)
	mockApp.OnDeleteStandardInvoice(nil)

	// when: finalization prepares the ledger, invoice sync fails, then the invoice is deleted
	s.finalizeFlatFeeCustomCurrencyInvoice(&fixture)
	s.Equal(1, invoiceSync.Attempts())
	s.deleteCustomCurrencyInvoice(&fixture)

	// then: the persisted run and every prepared ledger effect are corrected
	s.requireFlatFeeCustomCurrencyDeletionCorrected(fixture)
	mockApp.AssertExpectations(s.T())
}

func (s *CustomCurrencyLedgerIntegrationTestSuite) TestUsageBasedFinalizationBooksAndCoversCustomCurrencyOverage() {
	createAt := datetime.MustParseTimeInLocation(s.T(), "2024-12-01T00:00:00Z", time.UTC).AsTime()
	clock.FreezeTime(createAt)
	defer clock.UnFreeze()

	// given: a draft 10 TOKENS usage invoice with 3 USD of paid credits
	fixture := s.prepareUsageBasedCustomCurrencyLedgerFixture(createAt)
	mockApp := s.SandboxApp.EnableMock(s.T())
	defer s.SandboxApp.DisableMock()
	invoiceSync := &failFirstInvoiceSync{}
	mockApp.OnUpsertStandardInvoice(invoiceSync.Upsert)
	mockApp.OnFinalizeStandardInvoice(nil)

	// when: finalization prepares the ledger, invoice sync fails, then retry succeeds
	s.finalizeUsageBasedCustomCurrencyInvoice(&fixture)
	s.Equal(1, invoiceSync.Attempts())
	s.retryCustomCurrencyInvoice(&fixture)
	s.Equal(2, invoiceSync.Attempts())
	s.Equal(1, mockApp.FinalizeInvoiceCallCount())

	// then: retry reuses preparation and payment settles only the remaining 2 USD
	s.requireUsageBasedCustomCurrencyPreparationReused(fixture)
	s.payCustomCurrencyInvoice(&fixture)
	s.requireUsageBasedCustomCurrencyPaymentSettled(fixture)
	mockApp.AssertExpectations(s.T())
}

func (s *CustomCurrencyLedgerIntegrationTestSuite) TestUsageBasedDeletionCorrectsCustomCurrencyOverage() {
	createAt := datetime.MustParseTimeInLocation(s.T(), "2024-12-01T00:00:00Z", time.UTC).AsTime()
	clock.FreezeTime(createAt)
	defer clock.UnFreeze()

	// given: a draft 10 TOKENS usage invoice with 3 USD of paid credits
	fixture := s.prepareUsageBasedCustomCurrencyLedgerFixture(createAt)
	mockApp := s.SandboxApp.EnableMock(s.T())
	defer s.SandboxApp.DisableMock()
	invoiceSync := &failFirstInvoiceSync{}
	mockApp.OnUpsertStandardInvoice(invoiceSync.Upsert)
	mockApp.OnDeleteStandardInvoice(nil)

	// when: finalization prepares the ledger, invoice sync fails, then the invoice is deleted
	s.finalizeUsageBasedCustomCurrencyInvoice(&fixture)
	s.Equal(1, invoiceSync.Attempts())
	s.deleteCustomCurrencyInvoice(&fixture)

	// then: the persisted run and every prepared ledger effect are corrected
	s.requireUsageBasedCustomCurrencyDeletionCorrected(fixture)
	mockApp.AssertExpectations(s.T())
}

type customCurrencyLedgerFixture struct {
	Namespace             string
	CustomerID            customer.CustomerID
	ChargeID              meta.ChargeID
	CustomCurrency        currencies.Currency
	FiatCurrency          currencyx.Currency
	SourceChargeID        string
	Invoice               billing.StandardInvoice
	RunID                 string
	GrossTransactionGroup string
	FiatCreditRealization creditrealization.Realization
}

type failFirstInvoiceSync struct {
	attempts int
}

func (s *failFirstInvoiceSync) Upsert(billing.StandardInvoice) (*billing.UpsertStandardInvoiceResult, error) {
	s.attempts++
	if s.attempts == 1 {
		return nil, errors.New("simulated invoice sync failure")
	}

	return billing.NewUpsertStandardInvoiceResult(), nil
}

func (s *failFirstInvoiceSync) Attempts() int {
	return s.attempts
}

func (s *CustomCurrencyLedgerIntegrationTestSuite) prepareFlatFeeCustomCurrencyLedgerFixture(createAt time.Time) customCurrencyLedgerFixture {
	s.T().Helper()
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-flatfee-real-ledger")
	defaults := s.ProvisionDefaultTaxCodes(ctx, ns)
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	customer := s.CreateTestCustomer(ns, "customer-c1")
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID(), billingtest.WithManualApproval())

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2025-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2025-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	customCurrency := s.createTestCustomCurrency(ctx, ns)
	fiatCurrency := s.newFiatCurrency(USD)
	costBasisIntent := costbasis.NewIntent(costbasis.ManualIntent{
		FiatCurrency: fiatCurrency,
		Rate:         alpacadecimal.NewFromFloat(0.5),
	})
	sourceChargeID := ulid.Make().String()
	s.fundFiatCredits(fundFiatCreditsInput{
		CustomerID:     customer.GetID(),
		At:             createAt,
		Amount:         alpacadecimal.NewFromInt(3),
		SourceChargeID: sourceChargeID,
	})

	// given: a 10 TOKENS flat fee with no custom credits and 3 USD of existing credits
	created, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			charges.NewChargeIntent(flatfee.Intent{
				Intent: meta.Intent{
					ManagedBy:         billing.SubscriptionManagedLine,
					UniqueReferenceID: lo.ToPtr("flat-fee-real-ledger"),
					CustomerID:        customer.ID,
					Currency:          customCurrency,
					TaxConfig: productcatalog.TaxCodeConfig{
						TaxCodeID: defaults.InvoicingTaxCodeID,
					},
				},
				IntentMutableFields: flatfee.IntentMutableFields{
					IntentMutableFields: meta.IntentMutableFields{
						Name:              "flat-fee-real-ledger",
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
	charge, err := created[0].AsFlatFeeCharge()
	s.Require().NoError(err)

	clock.FreezeTime(servicePeriod.To)
	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: customer.GetID(),
		AsOf:     lo.ToPtr(servicePeriod.To),
	})
	s.Require().NoError(err)
	s.Require().Len(invoices, 1)
	invoice := invoices[0]
	s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, invoice.Status)

	return customCurrencyLedgerFixture{
		Namespace:      ns,
		CustomerID:     customer.GetID(),
		ChargeID:       charge.GetChargeID(),
		CustomCurrency: customCurrency,
		FiatCurrency:   fiatCurrency,
		SourceChargeID: sourceChargeID,
		Invoice:        invoice,
	}
}

func (s *CustomCurrencyLedgerIntegrationTestSuite) prepareUsageBasedCustomCurrencyLedgerFixture(createAt time.Time) customCurrencyLedgerFixture {
	s.T().Helper()
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-usage-based-real-ledger")
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

	invoiceAt := datetime.MustParseTimeInLocation(s.T(), "2025-02-01T00:00:00Z", time.UTC).AsTime()
	collectionAt := datetime.MustParseTimeInLocation(s.T(), "2025-02-02T00:01:00Z", time.UTC).AsTime()
	usageAt := datetime.MustParseTimeInLocation(s.T(), "2025-01-15T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2025-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   invoiceAt,
	}
	feature := s.SetupApiRequestsTotalFeature(ctx, ns)
	s.T().Cleanup(feature.Cleanup)
	customCurrency := s.createTestCustomCurrency(ctx, ns)
	fiatCurrency := s.newFiatCurrency(USD)
	costBasisIntent := costbasis.NewIntent(costbasis.ManualIntent{
		FiatCurrency: fiatCurrency,
		Rate:         alpacadecimal.NewFromFloat(0.5),
	})
	sourceChargeID := ulid.Make().String()
	s.fundFiatCredits(fundFiatCreditsInput{
		CustomerID:     customer.GetID(),
		At:             createAt,
		Amount:         alpacadecimal.NewFromInt(3),
		SourceChargeID: sourceChargeID,
	})

	// given: 5 metered units priced at 2 TOKENS with no custom credits and 3 USD of existing credits
	created, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			charges.NewChargeIntent(usagebased.Intent{
				Intent: meta.Intent{
					ManagedBy:         billing.SubscriptionManagedLine,
					UniqueReferenceID: lo.ToPtr("usage-based-real-ledger"),
					CustomerID:        customer.ID,
					Currency:          customCurrency,
					TaxConfig: productcatalog.TaxCodeConfig{
						TaxCodeID: defaults.InvoicingTaxCodeID,
					},
				},
				IntentMutableFields: usagebased.IntentMutableFields{
					IntentMutableFields: meta.IntentMutableFields{
						Name:              "usage-based-real-ledger",
						ServicePeriod:     servicePeriod,
						FullServicePeriod: servicePeriod,
						BillingPeriod:     servicePeriod,
					},
					InvoiceAt: invoiceAt,
					Price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromInt(2),
					}),
				},
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				FeatureKey:     feature.Feature.Key,
				CostBasis:      &costBasisIntent,
			}),
		},
	})
	s.Require().NoError(err)
	s.Require().Len(created, 1)
	charge, err := created[0].AsUsageBasedCharge()
	s.Require().NoError(err)

	s.MockStreamingConnector.AddSimpleEvent(
		feature.Feature.Key,
		5,
		usageAt,
		streamingtestutils.WithStoredAt(usageAt),
	)
	clock.FreezeTime(invoiceAt)
	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: customer.GetID(),
		AsOf:     lo.ToPtr(invoiceAt),
	})
	s.Require().NoError(err)
	s.Require().Len(invoices, 1)
	invoice := invoices[0]

	clock.FreezeTime(collectionAt)
	invoice, err = s.BillingService.AdvanceInvoice(ctx, invoice.GetInvoiceID())
	s.Require().NoError(err)
	s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, invoice.Status)

	return customCurrencyLedgerFixture{
		Namespace:      ns,
		CustomerID:     customer.GetID(),
		ChargeID:       charge.GetChargeID(),
		CustomCurrency: customCurrency,
		FiatCurrency:   fiatCurrency,
		SourceChargeID: sourceChargeID,
		Invoice:        invoice,
	}
}

func (s *CustomCurrencyLedgerIntegrationTestSuite) finalizeFlatFeeCustomCurrencyInvoice(fixture *customCurrencyLedgerFixture) {
	s.T().Helper()

	invoice, err := s.BillingService.ApproveInvoice(s.T().Context(), fixture.Invoice.GetInvoiceID())
	s.Require().NoError(err)
	s.Equal(billing.StandardInvoiceStatusIssuingSyncFailed, invoice.Status)
	fixture.Invoice = invoice

	persisted := s.mustGetChargeByID(fixture.ChargeID)
	charge, err := persisted.AsFlatFeeCharge()
	s.Require().NoError(err)
	s.Require().NotNil(charge.Realizations.CurrentRun)
	run := charge.Realizations.CurrentRun
	s.Require().NotNil(run.AccruedUsage)
	s.Require().NotNil(run.AccruedUsage.LedgerTransaction)
	s.Require().Len(run.FiatOverageCreditRealizations, 1)
	fixture.RunID = run.ID.ID
	fixture.GrossTransactionGroup = run.AccruedUsage.LedgerTransaction.TransactionGroupID
	fixture.FiatCreditRealization = run.FiatOverageCreditRealizations[0]

	s.requirePreparedCustomCurrencyLedgerOutcome(*fixture)
}

func (s *CustomCurrencyLedgerIntegrationTestSuite) finalizeUsageBasedCustomCurrencyInvoice(fixture *customCurrencyLedgerFixture) {
	s.T().Helper()

	invoice, err := s.BillingService.ApproveInvoice(s.T().Context(), fixture.Invoice.GetInvoiceID())
	s.Require().NoError(err)
	s.Equal(billing.StandardInvoiceStatusIssuingSyncFailed, invoice.Status)
	fixture.Invoice = invoice

	persisted := s.mustGetChargeByID(fixture.ChargeID)
	charge, err := persisted.AsUsageBasedCharge()
	s.Require().NoError(err)
	run, err := charge.GetCurrentRealizationRun()
	s.Require().NoError(err)
	s.Require().NotNil(run.InvoiceUsage)
	s.Require().NotNil(run.InvoiceUsage.LedgerTransaction)
	s.Require().Len(run.FiatOverageCreditRealizations, 1)
	fixture.RunID = run.ID.ID
	fixture.GrossTransactionGroup = run.InvoiceUsage.LedgerTransaction.TransactionGroupID
	fixture.FiatCreditRealization = run.FiatOverageCreditRealizations[0]

	s.requirePreparedCustomCurrencyLedgerOutcome(*fixture)
}

func (s *CustomCurrencyLedgerIntegrationTestSuite) requirePreparedCustomCurrencyLedgerOutcome(fixture customCurrencyLedgerFixture) {
	s.T().Helper()

	s.requireCustomCurrencyInvoice(fixture.Invoice, fixture.FiatCurrency)
	s.requireCustomCurrencyLedgerOutcome(requireCustomCurrencyLedgerOutcomeInput{
		Namespace:             fixture.Namespace,
		CustomerID:            fixture.CustomerID,
		ChargeID:              fixture.ChargeID.ID,
		SourceChargeID:        fixture.SourceChargeID,
		CustomCurrency:        fixture.CustomCurrency,
		GrossTransactionGroup: fixture.GrossTransactionGroup,
		FiatCreditRealization: fixture.FiatCreditRealization,
	})
}

func (s *CustomCurrencyLedgerIntegrationTestSuite) retryCustomCurrencyInvoice(fixture *customCurrencyLedgerFixture) {
	s.T().Helper()

	invoice, err := s.BillingService.RetryInvoice(s.T().Context(), fixture.Invoice.GetInvoiceID())
	s.Require().NoError(err)
	s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)
	fixture.Invoice = invoice
}

func (s *CustomCurrencyLedgerIntegrationTestSuite) requireFlatFeeCustomCurrencyPreparationReused(fixture customCurrencyLedgerFixture) {
	s.T().Helper()

	persisted := s.mustGetChargeByID(fixture.ChargeID)
	charge, err := persisted.AsFlatFeeCharge()
	s.Require().NoError(err)
	s.Equal(flatfee.StatusActiveAwaitingPaymentSettlement, charge.Status)
	s.Require().NotNil(charge.Realizations.CurrentRun)
	run := charge.Realizations.CurrentRun
	s.Equal(fixture.RunID, run.ID.ID)
	s.Require().NotNil(run.AccruedUsage)
	s.Require().NotNil(run.AccruedUsage.LedgerTransaction)
	s.Equal(fixture.GrossTransactionGroup, run.AccruedUsage.LedgerTransaction.TransactionGroupID)
	s.Require().Len(run.FiatOverageCreditRealizations, 1)
	s.Equal(fixture.FiatCreditRealization.ID, run.FiatOverageCreditRealizations[0].ID)
	s.requirePreparedCustomCurrencyLedgerOutcome(fixture)
}

func (s *CustomCurrencyLedgerIntegrationTestSuite) requireUsageBasedCustomCurrencyPreparationReused(fixture customCurrencyLedgerFixture) {
	s.T().Helper()

	persisted := s.mustGetChargeByID(fixture.ChargeID)
	charge, err := persisted.AsUsageBasedCharge()
	s.Require().NoError(err)
	s.Equal(usagebased.StatusActiveAwaitingPaymentSettlement, charge.Status)
	s.Nil(charge.State.CurrentRealizationRunID)
	run, err := charge.Realizations.GetByID(fixture.RunID)
	s.Require().NoError(err)
	s.Require().NotNil(run.InvoiceUsage)
	s.Require().NotNil(run.InvoiceUsage.LedgerTransaction)
	s.Equal(fixture.GrossTransactionGroup, run.InvoiceUsage.LedgerTransaction.TransactionGroupID)
	s.Require().Len(run.FiatOverageCreditRealizations, 1)
	s.Equal(fixture.FiatCreditRealization.ID, run.FiatOverageCreditRealizations[0].ID)
	s.requirePreparedCustomCurrencyLedgerOutcome(fixture)
}

func (s *CustomCurrencyLedgerIntegrationTestSuite) payCustomCurrencyInvoice(fixture *customCurrencyLedgerFixture) {
	s.T().Helper()
	ctx := s.T().Context()

	err := s.BillingService.TriggerInvoice(ctx, billing.InvoiceTriggerServiceInput{
		InvoiceTriggerInput: billing.InvoiceTriggerInput{
			Invoice: fixture.Invoice.GetInvoiceID(),
			Trigger: billing.TriggerPaid,
		},
		AppType:    app.AppTypeSandbox,
		Capability: app.CapabilityTypeCollectPayments,
	})
	s.Require().NoError(err)
	invoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
		Invoice: fixture.Invoice.GetInvoiceID(),
		Expand:  billing.StandardInvoiceExpandAll,
	})
	s.Require().NoError(err)
	s.Equal(billing.StandardInvoiceStatusPaid, invoice.Status)
	fixture.Invoice = invoice
	s.requireCustomCurrencyInvoice(invoice, fixture.FiatCurrency)
}

func (s *CustomCurrencyLedgerIntegrationTestSuite) requireFlatFeeCustomCurrencyPaymentSettled(fixture customCurrencyLedgerFixture) {
	s.T().Helper()

	persisted := s.mustGetChargeByID(fixture.ChargeID)
	charge, err := persisted.AsFlatFeeCharge()
	s.Require().NoError(err)
	s.Equal(flatfee.StatusFinal, charge.Status)
	s.Require().NotNil(charge.Realizations.CurrentRun)
	s.requireSettledCustomCurrencyPayment(requireSettledCustomCurrencyPaymentInput{
		Namespace:      fixture.Namespace,
		CustomerID:     fixture.CustomerID,
		ChargeID:       fixture.ChargeID.ID,
		CustomCurrency: fixture.CustomCurrency,
		Payment:        charge.Realizations.CurrentRun.Payment,
	})
}

func (s *CustomCurrencyLedgerIntegrationTestSuite) requireUsageBasedCustomCurrencyPaymentSettled(fixture customCurrencyLedgerFixture) {
	s.T().Helper()

	persisted := s.mustGetChargeByID(fixture.ChargeID)
	charge, err := persisted.AsUsageBasedCharge()
	s.Require().NoError(err)
	s.Equal(usagebased.StatusFinal, charge.Status)
	s.Nil(charge.State.CurrentRealizationRunID)
	run, err := charge.Realizations.GetByID(fixture.RunID)
	s.Require().NoError(err)
	s.requireSettledCustomCurrencyPayment(requireSettledCustomCurrencyPaymentInput{
		Namespace:      fixture.Namespace,
		CustomerID:     fixture.CustomerID,
		ChargeID:       fixture.ChargeID.ID,
		CustomCurrency: fixture.CustomCurrency,
		Payment:        run.Payment,
	})
}

func (s *CustomCurrencyLedgerIntegrationTestSuite) deleteCustomCurrencyInvoice(fixture *customCurrencyLedgerFixture) {
	s.T().Helper()

	invoice, err := s.BillingService.DeleteInvoice(s.T().Context(), billing.DeleteInvoiceInput{
		Invoice:        fixture.Invoice.GetInvoiceID(),
		DeletionSource: billing.ChangeSourceAPIRequest,
	})
	s.Require().NoError(err)
	s.Equal(billing.StandardInvoiceStatusDeleted, invoice.Status)
	fixture.Invoice = invoice
}

func (s *CustomCurrencyLedgerIntegrationTestSuite) requireFlatFeeCustomCurrencyDeletionCorrected(fixture customCurrencyLedgerFixture) {
	s.T().Helper()

	persisted := s.mustGetChargeByID(fixture.ChargeID)
	charge, err := persisted.AsFlatFeeCharge()
	s.Require().NoError(err)
	s.Equal(flatfee.StatusDeleted, charge.Status)
	s.Nil(charge.Realizations.CurrentRun)
	s.Require().Len(charge.Realizations.PriorRuns, 1)
	correctedRun := charge.Realizations.PriorRuns[0]
	s.Equal(fixture.RunID, correctedRun.ID.ID)
	s.Require().NotNil(correctedRun.DeletedAt)

	s.requireCustomCurrencyCorrectionOutcome(requireCustomCurrencyCorrectionOutcomeInput{
		Namespace:                 fixture.Namespace,
		CustomerID:                fixture.CustomerID,
		ChargeID:                  fixture.ChargeID.ID,
		CustomCurrency:            fixture.CustomCurrency,
		OriginalFiatRealization:   fixture.FiatCreditRealization,
		CorrectedFiatRealizations: correctedRun.FiatOverageCreditRealizations,
	})
}

func (s *CustomCurrencyLedgerIntegrationTestSuite) requireUsageBasedCustomCurrencyDeletionCorrected(fixture customCurrencyLedgerFixture) {
	s.T().Helper()
	ctx := s.T().Context()

	persisted := s.mustGetChargeByID(fixture.ChargeID)
	charge, err := persisted.AsUsageBasedCharge()
	s.Require().NoError(err)
	s.Equal(usagebased.StatusDeleted, charge.Status)
	s.Nil(charge.State.CurrentRealizationRunID)
	dbRun, err := s.DBClient.ChargeUsageBasedRuns.Get(ctx, fixture.RunID)
	s.Require().NoError(err)
	s.Require().NotNil(dbRun.DeletedAt)
	dbFiatRealizations, err := dbRun.QueryFiatOverageCreditAllocations().All(ctx)
	s.Require().NoError(err)

	s.requireCustomCurrencyCorrectionOutcome(requireCustomCurrencyCorrectionOutcomeInput{
		Namespace:                 fixture.Namespace,
		CustomerID:                fixture.CustomerID,
		ChargeID:                  fixture.ChargeID.ID,
		CustomCurrency:            fixture.CustomCurrency,
		OriginalFiatRealization:   fixture.FiatCreditRealization,
		CorrectedFiatRealizations: creditrealization.FromDBRealizations(dbFiatRealizations),
	})
}

type fundFiatCreditsInput struct {
	CustomerID     customer.CustomerID
	At             time.Time
	Amount         alpacadecimal.Decimal
	SourceChargeID string
}

func (s *CustomCurrencyLedgerIntegrationTestSuite) fundFiatCredits(input fundFiatCreditsInput) {
	s.T().Helper()

	// Seed the paid FBO balance at the ledger boundary so this fixture exercises
	// charge finalization without taking ownership of credit-purchase behavior.
	_, err := s.LedgerDeps.ResolversService.CreateCustomerAccounts(s.T().Context(), input.CustomerID)
	s.Require().NoError(err)
	_, err = s.LedgerDeps.ResolversService.EnsureBusinessAccounts(s.T().Context(), input.CustomerID.Namespace)
	s.Require().NoError(err)

	resolverDeps := transactions.ResolverDependencies{
		AccountService: s.LedgerDeps.ResolversService,
		AccountCatalog: s.LedgerDeps.AccountService,
		BalanceQuerier: s.LedgerDeps.HistoricalLedger,
	}
	scope := transactions.ResolutionScope{
		CustomerID: input.CustomerID,
		Namespace:  input.CustomerID.Namespace,
	}
	currency := currencies.NewCurrencyReference(USD)

	issue, err := transactions.ResolveTransactions(s.T().Context(), resolverDeps, scope, transactions.IssueCustomerReceivableTemplate{
		At:             input.At,
		Amount:         input.Amount,
		Currency:       currency,
		SourceChargeID: &input.SourceChargeID,
	})
	s.Require().NoError(err)
	_, err = s.LedgerDeps.HistoricalLedger.CommitGroup(s.T().Context(), transactions.GroupInputs(input.CustomerID.Namespace, nil, issue...))
	s.Require().NoError(err)

	settle, err := transactions.ResolveTransactions(
		s.T().Context(),
		resolverDeps,
		scope,
		transactions.AuthorizeCustomerReceivablePaymentTemplate{
			At:             input.At,
			Amount:         input.Amount,
			Currency:       currency,
			SourceChargeID: &input.SourceChargeID,
		},
		transactions.SettleCustomerReceivableFromPaymentTemplate{
			At:             input.At,
			Amount:         input.Amount,
			Currency:       currency,
			SourceChargeID: &input.SourceChargeID,
		},
	)
	s.Require().NoError(err)
	_, err = s.LedgerDeps.HistoricalLedger.CommitGroup(s.T().Context(), transactions.GroupInputs(input.CustomerID.Namespace, nil, settle...))
	s.Require().NoError(err)
}

func (s *CustomCurrencyLedgerIntegrationTestSuite) requireCustomCurrencyInvoice(invoice billing.StandardInvoice, fiatCurrency currencyx.Currency) {
	s.T().Helper()

	s.RequireTotals(billingtest.ExpectedTotals{
		Amount:       5,
		CreditsTotal: 3,
		Total:        2,
	}, invoice.Totals)
	s.Require().Len(invoice.Lines.OrEmpty(), 1)
	s.Equal(float64(3), invoice.Lines.OrEmpty()[0].CreditsApplied.SumAmount(fiatCurrency).InexactFloat64())
}

type requireCustomCurrencyLedgerOutcomeInput struct {
	Namespace             string
	CustomerID            customer.CustomerID
	ChargeID              string
	SourceChargeID        string
	CustomCurrency        currencies.Currency
	GrossTransactionGroup string
	FiatCreditRealization creditrealization.Realization
}

func (s *CustomCurrencyLedgerIntegrationTestSuite) requireCustomCurrencyLedgerOutcome(input requireCustomCurrencyLedgerOutcomeInput) {
	s.T().Helper()
	ctx := s.T().Context()

	// The gross overage is one atomic credit-purchase-equivalent booking:
	// issue the uncovered custom amount, consume it immediately into accrued,
	// then convert its custom receivable into the invoice's fiat receivable.
	s.requireTransactionTemplates(input.Namespace, input.GrossTransactionGroup, []string{
		transactions.TemplateCode(transactions.IssueCustomerReceivableTemplate{}),
		transactions.TemplateCode(transactions.TransferCustomerFBOAdvanceToAccruedTemplate{}),
		transactions.TemplateCode(transactions.ConvertCurrencyTemplate{}),
	})
	// Existing fiat credit settles part of that already-booked receivable in a
	// separate group; it must not replace or shrink the gross overage booking.
	s.requireTransactionTemplates(input.Namespace, input.FiatCreditRealization.LedgerTransaction.TransactionGroupID, []string{
		transactions.TemplateCode(transactions.CoverCustomerReceivableTemplate{}),
	})
	s.requireChargeTransactionTemplates(
		input.Namespace,
		input.ChargeID,
		ledger.TransactionDirectionForward,
		[]string{
			transactions.TemplateCode(transactions.IssueCustomerReceivableTemplate{}),
			transactions.TemplateCode(transactions.TransferCustomerFBOAdvanceToAccruedTemplate{}),
			transactions.TemplateCode(transactions.ConvertCurrencyTemplate{}),
			transactions.TemplateCode(transactions.CoverCustomerReceivableTemplate{}),
		},
	)

	// Receivable coverage preserves both sides of attribution. The original
	// credit-purchase charge remains the value source, while the custom-currency
	// overage charge is the spend that consumed it.
	coverageGroup, err := s.LedgerDeps.HistoricalLedger.GetTransactionGroup(ctx, models.NamespacedID{
		Namespace: input.Namespace,
		ID:        input.FiatCreditRealization.LedgerTransaction.TransactionGroupID,
	})
	s.Require().NoError(err)
	s.Require().Len(coverageGroup.Transactions(), 1)
	for _, entry := range coverageGroup.Transactions()[0].Entries() {
		s.Require().NotNil(entry.SourceChargeID())
		s.Equal(input.SourceChargeID, *entry.SourceChargeID())
		s.Require().NotNil(entry.SpendChargeID())
		s.Equal(input.ChargeID, *entry.SpendChargeID())
	}

	// The synthetic custom purchase leaves no spendable custom balance or open
	// custom receivable: all 10 TOKENS are accrued. The customer's 3 USD credit
	// is fully consumed, reducing the gross -5 USD receivable to -2 USD owed.
	accounts, err := s.LedgerDeps.ResolversService.GetCustomerAccounts(ctx, input.CustomerID)
	s.Require().NoError(err)
	customFilter := ledger.RouteFilter{Currency: input.CustomCurrency.Reference()}
	fiatFilter := ledger.RouteFilter{Currency: currencies.NewCurrencyReference(USD)}
	s.requireAccountBalance(accounts.FBOAccount, customFilter, 0)
	s.requireAccountBalance(accounts.ReceivableAccount, customFilter, 0)
	s.requireAccountBalance(accounts.AccruedAccount, customFilter, 10)
	s.requireAccountBalance(accounts.FBOAccount, fiatFilter, 0)
	s.requireAccountBalance(accounts.ReceivableAccount, fiatFilter, -2)

	lineages, err := s.LineageService.LoadLineagesByCustomer(ctx, lineage.LoadLineagesByCustomerInput{
		Namespace:  input.Namespace,
		CustomerID: input.CustomerID.ID,
		Currency:   currencies.NewCurrencyReference(USD),
	})
	s.Require().NoError(err)
	// New coverage is corrected from its ledger origin, with no lineage side state.
	s.Empty(lineages)
	s.Equal(true, input.FiatCreditRealization.Annotations[ledger.AnnotationOriginTracked])
	for _, tx := range coverageGroup.Transactions() {
		for _, entry := range tx.Entries() {
			s.NotNil(entry.OriginID())
		}
	}
}

type requireCustomCurrencyCorrectionOutcomeInput struct {
	Namespace                 string
	CustomerID                customer.CustomerID
	ChargeID                  string
	CustomCurrency            currencies.Currency
	OriginalFiatRealization   creditrealization.Realization
	CorrectedFiatRealizations creditrealization.Realizations
}

func (s *CustomCurrencyLedgerIntegrationTestSuite) requireCustomCurrencyCorrectionOutcome(input requireCustomCurrencyCorrectionOutcomeInput) {
	s.T().Helper()
	ctx := s.T().Context()

	// The allocation stays in history and is offset by one persisted correction.
	// Its group reverses the fiat receivable coverage before the gross custom
	// overage group is reversed in dependency order.
	s.Require().Len(input.CorrectedFiatRealizations, 2)
	_, found := lo.Find(input.CorrectedFiatRealizations, func(realization creditrealization.Realization) bool {
		return realization.ID == input.OriginalFiatRealization.ID && realization.Type == creditrealization.TypeAllocation
	})
	s.True(found, "original fiat allocation is missing")
	correction, found := lo.Find(input.CorrectedFiatRealizations, func(realization creditrealization.Realization) bool {
		return realization.Type == creditrealization.TypeCorrection
	})
	s.Require().True(found, "fiat allocation correction is missing")
	s.Equal(float64(-3), correction.Amount.InexactFloat64())
	s.Require().NotNil(correction.CorrectsRealizationID)
	s.Equal(input.OriginalFiatRealization.ID, *correction.CorrectsRealizationID)
	s.Require().NotNil(correction.LedgerTransaction)
	s.requireTransactionTemplates(input.Namespace, correction.LedgerTransaction.TransactionGroupID, []string{
		transactions.TemplateCode(transactions.CoverCustomerReceivableTemplate{}),
	})

	// One correction exists for each prepared transaction: fiat coverage plus
	// the conversion, immediate custom consumption, and custom credit issuance.
	s.requireChargeTransactionTemplates(
		input.Namespace,
		input.ChargeID,
		ledger.TransactionDirectionCorrection,
		[]string{
			transactions.TemplateCode(transactions.CoverCustomerReceivableTemplate{}),
			transactions.TemplateCode(transactions.ConvertCurrencyTemplate{}),
			transactions.TemplateCode(transactions.TransferCustomerFBOAdvanceToAccruedTemplate{}),
			transactions.TemplateCode(transactions.IssueCustomerReceivableTemplate{}),
		},
	)

	// Cancellation removes the synthetic custom purchase and its invoice
	// receivable, while restoring the original 3 USD paid-credit balance.
	accounts, err := s.LedgerDeps.ResolversService.GetCustomerAccounts(ctx, input.CustomerID)
	s.Require().NoError(err)
	customFilter := ledger.RouteFilter{Currency: input.CustomCurrency.Reference()}
	fiatFilter := ledger.RouteFilter{Currency: currencies.NewCurrencyReference(USD)}
	s.requireAccountBalance(accounts.FBOAccount, customFilter, 0)
	s.requireAccountBalance(accounts.ReceivableAccount, customFilter, 0)
	s.requireAccountBalance(accounts.AccruedAccount, customFilter, 0)
	s.requireAccountBalance(accounts.FBOAccount, fiatFilter, 3)
	s.requireAccountBalance(accounts.ReceivableAccount, fiatFilter, 0)

	// The immutable journal retains the origin and its reversal without lineage.
	lineages, err := s.LineageService.LoadLineagesByCustomer(ctx, lineage.LoadLineagesByCustomerInput{
		Namespace:  input.Namespace,
		CustomerID: input.CustomerID.ID,
		Currency:   currencies.NewCurrencyReference(USD),
	})
	s.Require().NoError(err)
	s.Empty(lineages)
	s.Equal(true, correction.Annotations[ledger.AnnotationOriginTracked])
}

type requireSettledCustomCurrencyPaymentInput struct {
	Namespace      string
	CustomerID     customer.CustomerID
	ChargeID       string
	CustomCurrency currencies.Currency
	Payment        *payment.Invoiced
}

func (s *CustomCurrencyLedgerIntegrationTestSuite) requireSettledCustomCurrencyPayment(input requireSettledCustomCurrencyPaymentInput) {
	s.T().Helper()
	ctx := s.T().Context()

	// Existing fiat credits already covered 3 USD, so the invoice payment must
	// authorize and settle only the remaining 2 USD on the same cost-basis route.
	s.Require().NotNil(input.Payment)
	s.Equal(payment.StatusSettled, input.Payment.Status)
	s.Equal(float64(2), input.Payment.FiatAmount.InexactFloat64())
	s.Require().NotNil(input.Payment.Authorized)
	s.Require().NotNil(input.Payment.Settled)
	s.requirePaymentTransaction(
		input.Namespace,
		input.Payment.Authorized.TransactionGroupID,
		transactions.TemplateCode(transactions.AuthorizeCustomerReceivablePaymentTemplate{}),
		input.ChargeID,
	)
	s.requirePaymentTransaction(
		input.Namespace,
		input.Payment.Settled.TransactionGroupID,
		transactions.TemplateCode(transactions.SettleCustomerReceivableFromPaymentTemplate{}),
		input.ChargeID,
	)

	// Payment clears only the remaining fiat receivable. It cannot rebook or
	// otherwise alter the already-consumed 10-unit custom overage.
	accounts, err := s.LedgerDeps.ResolversService.GetCustomerAccounts(ctx, input.CustomerID)
	s.Require().NoError(err)
	customFilter := ledger.RouteFilter{Currency: input.CustomCurrency.Reference()}
	fiatFilter := ledger.RouteFilter{Currency: currencies.NewCurrencyReference(USD)}
	s.requireAccountBalance(accounts.FBOAccount, customFilter, 0)
	s.requireAccountBalance(accounts.ReceivableAccount, customFilter, 0)
	s.requireAccountBalance(accounts.AccruedAccount, customFilter, 10)
	s.requireAccountBalance(accounts.FBOAccount, fiatFilter, 0)
	s.requireAccountBalance(accounts.ReceivableAccount, fiatFilter, 0)

	s.requireChargeTransactionTemplates(
		input.Namespace,
		input.ChargeID,
		ledger.TransactionDirectionForward,
		[]string{
			transactions.TemplateCode(transactions.IssueCustomerReceivableTemplate{}),
			transactions.TemplateCode(transactions.TransferCustomerFBOAdvanceToAccruedTemplate{}),
			transactions.TemplateCode(transactions.ConvertCurrencyTemplate{}),
			transactions.TemplateCode(transactions.CoverCustomerReceivableTemplate{}),
			transactions.TemplateCode(transactions.AuthorizeCustomerReceivablePaymentTemplate{}),
			transactions.TemplateCode(transactions.SettleCustomerReceivableFromPaymentTemplate{}),
		},
	)
}

func (s *CustomCurrencyLedgerIntegrationTestSuite) requirePaymentTransaction(namespace, groupID, expectedTemplateCode, chargeID string) {
	s.T().Helper()

	group, err := s.LedgerDeps.HistoricalLedger.GetTransactionGroup(s.T().Context(), models.NamespacedID{
		Namespace: namespace,
		ID:        groupID,
	})
	s.Require().NoError(err)
	s.Require().Len(group.Transactions(), 1)
	transaction := group.Transactions()[0]
	templateCode, err := ledger.TransactionTemplateCodeFromAnnotations(transaction.Annotations())
	s.Require().NoError(err)
	s.Equal(expectedTemplateCode, templateCode)
	for _, entry := range transaction.Entries() {
		s.Require().NotNil(entry.SourceChargeID())
		s.Equal(chargeID, *entry.SourceChargeID())
		s.Nil(entry.SpendChargeID())
	}
}

func (s *CustomCurrencyLedgerIntegrationTestSuite) requireChargeTransactionTemplates(namespace, chargeID string, direction ledger.TransactionDirection, expected []string) {
	s.T().Helper()

	result, err := s.LedgerDeps.HistoricalLedger.ListTransactions(s.T().Context(), ledger.ListTransactionsInput{
		Namespace: namespace,
		Limit:     20,
		AnnotationFilters: map[string]string{
			ledger.AnnotationChargeID:             chargeID,
			ledger.AnnotationTransactionDirection: string(direction),
		},
	})
	s.Require().NoError(err)
	actual := make([]string, 0, len(result.Items))
	for _, transaction := range result.Items {
		templateCode, err := ledger.TransactionTemplateCodeFromAnnotations(transaction.Annotations())
		s.Require().NoError(err)
		actual = append(actual, templateCode)
	}
	s.ElementsMatch(expected, actual)
}

func (s *CustomCurrencyLedgerIntegrationTestSuite) requireTransactionTemplates(namespace, groupID string, expected []string) {
	s.T().Helper()

	group, err := s.LedgerDeps.HistoricalLedger.GetTransactionGroup(s.T().Context(), models.NamespacedID{
		Namespace: namespace,
		ID:        groupID,
	})
	s.Require().NoError(err)
	actual := make([]string, 0, len(group.Transactions()))
	for _, transaction := range group.Transactions() {
		templateCode, err := ledger.TransactionTemplateCodeFromAnnotations(transaction.Annotations())
		s.Require().NoError(err)
		actual = append(actual, templateCode)
	}
	s.Equal(expected, actual)
}

func (s *CustomCurrencyLedgerIntegrationTestSuite) requireAccountBalance(account ledger.Account, filter ledger.RouteFilter, expected float64) {
	s.T().Helper()

	balance, err := s.LedgerDeps.HistoricalLedger.GetAccountBalance(s.T().Context(), account, filter, ledger.BalanceQuery{})
	s.Require().NoError(err)
	s.Equal(expected, balance.InexactFloat64())
}
