package service

import (
	"errors"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
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
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-flatfee-real-ledger")
	defaults := s.ProvisionDefaultTaxCodes(ctx, ns)
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	customer := s.CreateTestCustomer(ns, "customer-c1")
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID(), billingtest.WithManualApproval())

	createAt := datetime.MustParseTimeInLocation(s.T(), "2024-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2025-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2025-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	clock.FreezeTime(createAt)
	defer clock.UnFreeze()

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

	// when: finalization prepares the invoice, then external synchronization fails
	mockApp := s.SandboxApp.EnableMock(s.T())
	defer s.SandboxApp.DisableMock()
	mockApp.OnUpsertStandardInvoice(func(billing.StandardInvoice) (*billing.UpsertStandardInvoiceResult, error) {
		return nil, errors.New("simulated invoice sync failure")
	})
	invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
	s.Require().NoError(err)
	s.Equal(billing.StandardInvoiceStatusIssuingSyncFailed, invoice.Status)

	persisted := s.mustGetChargeByID(charge.GetChargeID())
	flatFeeCharge, err := persisted.AsFlatFeeCharge()
	s.Require().NoError(err)
	s.Require().NotNil(flatFeeCharge.Realizations.CurrentRun)
	run := flatFeeCharge.Realizations.CurrentRun
	s.Require().NotNil(run.AccruedUsage)
	s.Require().NotNil(run.AccruedUsage.LedgerTransaction)
	s.Require().Len(run.FiatOverageCreditRealizations, 1)

	// then: the invoice records the gross 5 USD purchase, 3 USD credit coverage, and 2 USD due
	s.requireCustomCurrencyInvoice(invoice, fiatCurrency)
	s.requireCustomCurrencyLedgerOutcome(requireCustomCurrencyLedgerOutcomeInput{
		Namespace:             ns,
		CustomerID:            customer.GetID(),
		ChargeID:              charge.ID,
		SourceChargeID:        sourceChargeID,
		CustomCurrency:        customCurrency,
		GrossTransactionGroup: run.AccruedUsage.LedgerTransaction.TransactionGroupID,
		FiatCreditRealization: run.FiatOverageCreditRealizations[0],
	})
	mockApp.AssertExpectations(s.T())
}

func (s *CustomCurrencyLedgerIntegrationTestSuite) TestUsageBasedFinalizationBooksAndCoversCustomCurrencyOverage() {
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

	createAt := datetime.MustParseTimeInLocation(s.T(), "2024-12-01T00:00:00Z", time.UTC).AsTime()
	invoiceAt := datetime.MustParseTimeInLocation(s.T(), "2025-02-01T00:00:00Z", time.UTC).AsTime()
	collectionAt := datetime.MustParseTimeInLocation(s.T(), "2025-02-02T00:01:00Z", time.UTC).AsTime()
	usageAt := datetime.MustParseTimeInLocation(s.T(), "2025-01-15T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2025-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   invoiceAt,
	}
	clock.FreezeTime(createAt)
	defer clock.UnFreeze()

	feature := s.SetupApiRequestsTotalFeature(ctx, ns)
	defer feature.Cleanup()
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

	// when: finalization prepares the invoice, then external synchronization fails
	mockApp := s.SandboxApp.EnableMock(s.T())
	defer s.SandboxApp.DisableMock()
	mockApp.OnUpsertStandardInvoice(func(billing.StandardInvoice) (*billing.UpsertStandardInvoiceResult, error) {
		return nil, errors.New("simulated invoice sync failure")
	})
	invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
	s.Require().NoError(err)
	s.Equal(billing.StandardInvoiceStatusIssuingSyncFailed, invoice.Status)

	persisted := s.mustGetChargeByID(charge.GetChargeID())
	usageBasedCharge, err := persisted.AsUsageBasedCharge()
	s.Require().NoError(err)
	run, err := usageBasedCharge.GetCurrentRealizationRun()
	s.Require().NoError(err)
	s.Require().NotNil(run.InvoiceUsage)
	s.Require().NotNil(run.InvoiceUsage.LedgerTransaction)
	s.Require().Len(run.FiatOverageCreditRealizations, 1)

	// then: FF and UBP reach the same invoice, ledger, provenance, and lineage result
	s.requireCustomCurrencyInvoice(invoice, fiatCurrency)
	s.requireCustomCurrencyLedgerOutcome(requireCustomCurrencyLedgerOutcomeInput{
		Namespace:             ns,
		CustomerID:            customer.GetID(),
		ChargeID:              charge.ID,
		SourceChargeID:        sourceChargeID,
		CustomCurrency:        customCurrency,
		GrossTransactionGroup: run.InvoiceUsage.LedgerTransaction.TransactionGroupID,
		FiatCreditRealization: run.FiatOverageCreditRealizations[0],
	})
	mockApp.AssertExpectations(s.T())
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
	// The 3 USD coverage is one real-credit realization owned by the overage
	// charge. Its lineage lets later correction unwind the allocation's current
	// segment state instead of using the legacy first-order fallback.
	s.Require().Len(lineages, 1)
	s.Equal(input.ChargeID, lineages[0].ChargeID)
	s.Equal(input.FiatCreditRealization.ID, lineages[0].RootRealizationID)
	s.Equal(creditrealization.LineageOriginKindRealCredit, lineages[0].OriginKind)
	s.Require().Len(lineages[0].Segments, 1)
	s.Equal(float64(3), lineages[0].Segments[0].Amount.InexactFloat64())
	s.Equal(creditrealization.LineageSegmentStateRealCredit, lineages[0].Segments[0].State)
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
