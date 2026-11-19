package appstripe

import (
	"errors"
	"net/http"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stripe/stripe-go/v80"

	stripeapp "github.com/openmeterio/openmeter/openmeter/app/stripe"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	stdinvoicesanity "github.com/openmeterio/openmeter/openmeter/billing/stdinvoice/sanity"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	omtestutils "github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

// TestMonetaryAuditSplitCreditPreservesDistinctProviderItems exercises a real
// tiered charge and settled purchased credits: five units at USD 1 plus five
// at USD 2 = USD 15 gross; one USD 7 realization covers USD 5 of the first
// detail and USD 2 of the second, leaving USD 8 cash due.
// This is a single-invoice, mixed-detail counterexample; it does not simulate
// three progressive invoices, which are covered by separate billing audits.
//
// This split is intentional: billing/models/stddetailedline.Bases.WithCreditsApplied
// copies the same realization ID onto each covered detail, and invoice sanity
// checks conserve the realization's aggregate value across those details.
// Initial synchronization through the real Stripe SDK correctly creates two
// distinct credit items totaling USD 7 and a provider subtotal of USD 8.
//
// The next app-level update indexes credit items only by realization ID in
// appinvoice.go:updateInvoice. One item overwrites the other in that map, so
// both detail allocations target the same provider item. The unselected credit
// item is also left in the removal set. Applying that plan would discard one
// allocation and overstate the cash due; the last update cannot represent both
// USD 5 and USD 2. Credit-item identity must retain its detailed-line allocation
// identity, or aggregation must preserve the full value and tax treatment.
//
// This test captures the update plan before the HTTP boundary and deliberately
// returns a sentinel error. It does not accept il_ IDs on invoice-item endpoints
// or claim the invoice finalized: TestStripeFinalizationMonetaryAudit separately
// demonstrates that existing il_/ii_ confusion blocks actual approval first.
// Thus this is an independent mapping defect masked by that collection blocker.
// Stripe invoice-item amounts and negative deductions:
// https://docs.stripe.com/api/invoiceitems/create?api-version=2024-06-20
//
// Reproduce: POSTGRES_HOST=127.0.0.1 go test -tags=dynamic ./test/app/stripe
// -run TestStripeInvoicing/TestMonetaryAuditSplitCreditPreservesDistinctProviderItems -count=1 -v
func (s *StripeInvoiceTestSuite) TestMonetaryAuditSplitCreditPreservesDistinctProviderItems() {
	// Given one purchased credit realization covering two actual rated details.
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("stripe-split-credit-monetary-audit")
	setupAt := time.Date(2025, time.December, 1, 0, 0, 0, 0, time.UTC)
	period := timeutil.ClosedPeriod{
		From: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC),
	}
	clock.FreezeTime(setupAt)
	defer clock.UnFreeze()
	defer s.MockStreamingConnector.Reset()

	s.TaxCodeEnv.ProvisionDefaultTaxCodes(s.T(), namespace)
	customInvoicing := s.SetupCustomInvoicing(namespace)
	s.ProvisionBillingProfile(ctx, namespace, customInvoicing.App.GetID(), billingtest.WithManualApproval())
	customerEntity := s.createStripeLedgerBackedCustomer(ctx, namespace, "synthetic-split-credit-subject")
	installedApp, err := s.Fixture.setupApp(ctx, namespace)
	s.Require().NoError(err)
	_, err = s.Fixture.setupAppCustomerData(ctx, installedApp, customerEntity)
	s.Require().NoError(err)
	installedStripeApp, ok := installedApp.(stripeapp.App)
	s.Require().True(ok)

	creditCharges, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: namespace,
		Intents: charges.NewCreateChargeIntents(s.createCreditPurchaseIntent(createCreditPurchaseIntentInput{
			customer:      customerEntity.GetID(),
			currency:      "USD",
			amount:        alpacadecimal.NewFromInt(7),
			servicePeriod: timeutil.ClosedPeriod{From: setupAt, To: setupAt},
			settlement: creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{
				InitialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus,
			}),
			costBasis: creditpurchase.NewCostBasis(creditpurchase.FiatCostBasis{
				Rate: alpacadecimal.NewFromInt(1),
			}),
		})),
	})
	s.Require().NoError(err)
	s.Require().Len(creditCharges, 1)
	creditCharge, err := creditCharges[0].AsCreditPurchaseCharge()
	s.Require().NoError(err)
	s.mustSettleExternalCreditPurchase(ctx, creditCharge.GetChargeID())

	usageFeature := s.SetupApiRequestsTotalFeature(ctx, namespace)
	defer usageFeature.Cleanup()
	_, err = s.Charges.Create(ctx, charges.CreateInput{
		Namespace: namespace,
		Intents: charges.NewCreateChargeIntents(s.createMockChargeIntent(createMockChargeIntentInput{
			customer:          customerEntity.GetID(),
			currency:          "USD",
			servicePeriod:     period,
			settlementMode:    productcatalog.CreditThenInvoiceSettlementMode,
			name:              "Synthetic tiered service",
			managedBy:         billing.SubscriptionManagedLine,
			uniqueReferenceID: "synthetic-split-credit",
			featureKey:        usageFeature.Feature.Key,
			price: productcatalog.NewPriceFrom(productcatalog.TieredPrice{
				Mode: productcatalog.GraduatedTieredPrice,
				Tiers: []productcatalog.PriceTier{
					{
						UpToAmount: lo.ToPtr(alpacadecimal.NewFromInt(5)),
						UnitPrice:  &productcatalog.PriceTierUnitPrice{Amount: alpacadecimal.NewFromInt(1)},
					},
					{
						UnitPrice: &productcatalog.PriceTierUnitPrice{Amount: alpacadecimal.NewFromInt(2)},
					},
				},
			}),
		})),
	})
	s.Require().NoError(err)
	s.MockStreamingConnector.AddSimpleEvent(usageFeature.Feature.Key, 10, period.From.Add(time.Hour))

	clock.FreezeTime(period.To.Add(time.Hour))
	defer clock.UnFreeze()
	collectedInvoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: customerEntity.GetID(),
		AsOf:     lo.ToPtr(period.To),
	})
	s.Require().NoError(err)
	s.Require().Len(collectedInvoices, 1)
	persistedInvoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
		Invoice: collectedInvoices[0].GetInvoiceID(),
		Expand:  billing.StandardInvoiceExpandAll,
	})
	s.Require().NoError(err)
	s.Require().NoError(stdinvoicesanity.Validate(persistedInvoice))
	s.Require().Equal(15.0, persistedInvoice.Totals.Amount.InexactFloat64())
	s.Require().Equal(7.0, persistedInvoice.Totals.CreditsTotal.InexactFloat64())
	s.Require().Equal(8.0, persistedInvoice.Totals.Total.InexactFloat64())
	s.Require().Len(persistedInvoice.Lines.OrEmpty(), 1)
	standardLine := persistedInvoice.Lines.OrEmpty()[0]
	s.Require().Len(standardLine.CreditsApplied, 1)
	s.Require().Len(standardLine.DetailedLines, 2)
	creditRealizationID := standardLine.CreditsApplied[0].CreditRealizationID
	allocatedAmounts := make([]float64, 0, len(standardLine.DetailedLines))
	for _, detail := range standardLine.DetailedLines {
		s.Require().Len(detail.CreditsApplied, 1)
		s.Require().Equal(creditRealizationID, detail.CreditsApplied[0].CreditRealizationID)
		allocatedAmounts = append(allocatedAmounts, detail.CreditsApplied[0].Amount.InexactFloat64())
	}

	s.Require().ElementsMatch([]float64{5, 2}, allocatedAmounts)

	provider := &stripeMonetaryAuditTransport{
		items:     make(map[string][]*stripe.InvoiceItem),
		responses: make(map[string][]byte),
	}
	previousHTTPClient := stripe.GetBackendWithConfig(stripe.APIBackend, &stripe.BackendConfig{}).(*stripe.BackendImplementation).HTTPClient
	stripe.SetHTTPClient(&http.Client{Transport: provider})
	defer stripe.SetHTTPClient(previousHTTPClient)
	realStripeApp, err := stripeapp.New(installedStripeApp.Meta, stripeapp.AppConfig{
		Logger:                 omtestutils.NewDiscardLogger(s.T()),
		AppService:             s.AppService,
		BillingService:         s.BillingService,
		StripeAppClientFactory: stripeclient.NewStripeAppClient,
		StripeAppService:       s.AppStripeService,
		SecretService:          s.SecretService,
	})
	s.Require().NoError(err)
	stripeInvoice, err := persistedInvoice.Clone()
	s.Require().NoError(err)
	createdResult, err := realStripeApp.UpsertStandardInvoice(ctx, stripeInvoice)
	s.Require().NoError(err)
	s.Require().NoError(createdResult.MergeIntoInvoice(&stripeInvoice))
	s.Require().Equal(int64(800), provider.invoiceTotal(stripeInvoice.ExternalIDs.Invoicing))

	providerLines := lo.Map(provider.items[stripeInvoice.ExternalIDs.Invoicing], func(item *stripe.InvoiceItem, _ int) *stripe.InvoiceLineItem {
		return &stripe.InvoiceLineItem{
			ID:          "il_" + item.ID,
			InvoiceItem: item,
			Amount:      item.Amount,
			Metadata:    item.Metadata,
		}
	})
	createdCreditItems := lo.Filter(providerLines, func(line *stripe.InvoiceLineItem, _ int) bool {
		return line.Metadata["om_line_type"] == "credit"
	})
	s.Require().Len(createdCreditItems, 2)
	s.Require().NotEqual(createdCreditItems[0].ID, createdCreditItems[1].ID)

	// When the unchanged snapshot is mapped for update, stop before execution.
	stopBeforeProviderMutation := errors.New("synthetic capture of update plan before provider execution")
	var plannedUpdates []*stripeclient.StripeInvoiceItemWithID
	s.StripeAppClient.On("UpdateInvoice", mock.Anything).
		Once().Return(&stripe.Invoice{ID: stripeInvoice.ExternalIDs.Invoicing}, nil)
	s.StripeAppClient.On("ListInvoiceLineItems", stripeInvoice.ExternalIDs.Invoicing).
		Once().Return(providerLines, nil)
	s.StripeAppClient.On("UpdateInvoiceLines", mock.Anything).
		Once().Run(func(args mock.Arguments) {
		plannedUpdates = args.Get(0).(stripeclient.UpdateInvoiceLinesInput).Lines
	}).Return([]*stripe.InvoiceItem(nil), stopBeforeProviderMutation)
	_, err = installedStripeApp.UpsertStandardInvoice(ctx, stripeInvoice)
	s.Require().ErrorIs(err, stopBeforeProviderMutation)
	s.StripeAppClient.AssertExpectations(s.T())

	// Then each distinct detail allocation must retain a distinct update target.
	creditUpdates := lo.Filter(plannedUpdates, func(item *stripeclient.StripeInvoiceItemWithID, _ int) bool {
		return item.Metadata["om_line_type"] == "credit"
	})
	s.Require().Len(creditUpdates, 2)
	updateIDs := lo.Map(creditUpdates, func(item *stripeclient.StripeInvoiceItemWithID, _ int) string {
		s.T().Logf("credit realization=%s target=%s amount=%d", item.Metadata["om_line_id"], item.ID, lo.FromPtr(item.Amount))
		return item.ID
	})
	s.Require().Equal(int64(-700), lo.SumBy(creditUpdates, func(item *stripeclient.StripeInvoiceItemWithID) int64 {
		return lo.FromPtr(item.Amount)
	}))
	s.Equal(2, len(lo.Uniq(updateIDs)), "USD 5 and USD 2 allocations cannot overwrite the same provider item")
	for _, detail := range standardLine.DetailedLines {
		creditAllocation := detail.CreditsApplied[0]
		expectedProviderLine, found := lo.Find(createdCreditItems, func(line *stripe.InvoiceLineItem) bool {
			return line.Amount == -creditAllocation.Amount.Mul(alpacadecimal.NewFromInt(100)).IntPart()
		})
		s.Require().True(found, "initial sync must include detail=%s realization=%s", detail.ID, creditAllocation.CreditRealizationID)
		plannedUpdate, found := lo.Find(creditUpdates, func(item *stripeclient.StripeInvoiceItemWithID) bool {
			return lo.FromPtr(item.Amount) == expectedProviderLine.Amount && item.Metadata["om_line_id"] == creditAllocation.CreditRealizationID
		})
		s.Require().True(found, "update must retain detail=%s realization=%s", detail.ID, creditAllocation.CreditRealizationID)

		// Resolve the target's identity independently of the known il_/ii_
		// endpoint defect; no provider mutation is executed by this test.
		actualProviderLine, found := lo.Find(providerLines, func(line *stripe.InvoiceLineItem) bool {
			return plannedUpdate.ID == line.ID || plannedUpdate.ID == line.InvoiceItem.ID
		})
		s.Require().True(found)
		s.Equal(expectedProviderLine.InvoiceItem.ID, actualProviderLine.InvoiceItem.ID,
			"detail=%s realization=%s must keep its own originally created credit item", detail.ID, creditAllocation.CreditRealizationID)
	}
}
