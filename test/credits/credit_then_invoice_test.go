package credits

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"

	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	billingrating "github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

func (s *CreditsTestSuite) TestUsageBasedCreditThenInvoiceProgressiveBillingCreditAllocation() {
	t := s.T()
	ctx := t.Context()
	ns := s.GetUniqueNamespace("charges-credits-usagebased-progressive-credit-then-invoice")

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.createLedgerBackedCustomer(ns, "test-subject")

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithProgressiveBilling(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(t, "P2D")),
		billingtest.WithManualApproval(),
	)

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)

	setupAt := datetime.MustParseTimeInLocation(t, "2025-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(t, "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(t, "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	midPeriodInvoiceAt := datetime.MustParseTimeInLocation(t, "2026-01-16T00:00:00Z", time.UTC).AsTime()
	costBasis := alpacadecimal.NewFromInt(1)

	clock.FreezeTime(setupAt)
	defer clock.UnFreeze()

	var (
		usageBasedChargeID meta.ChargeID
		partialInvoice     billing.StandardInvoice
		finalInvoice       billing.StandardInvoice
	)

	t.Run("given settled purchased credits and a credit-then-invoice usage charge", func(t *testing.T) {
		// given:
		// - a ledger-backed customer with progressive billing enabled
		// - the customer buys and settles 7 USD credits
		// when:
		// - a unit-priced credit-then-invoice usage charge is created
		// then:
		// - the purchased credits are available and the usage charge has no realizations yet
		creditPurchaseIntent := s.createCreditPurchaseIntent(createCreditPurchaseIntentInput{
			customer:      cust.GetID(),
			currency:      USD,
			amount:        alpacadecimal.NewFromInt(7),
			servicePeriod: timeutil.ClosedPeriod{From: setupAt, To: setupAt},
			settlement: creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{
				GenericSettlement: creditpurchase.GenericSettlement{
					Currency:  USD,
					CostBasis: costBasis,
				},
				InitialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus,
			}),
		})

		creditPurchaseRes, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				creditPurchaseIntent,
			},
		})
		s.NoError(err)
		s.Len(creditPurchaseRes, 1)

		creditPurchaseCharge, err := creditPurchaseRes[0].AsCreditPurchaseCharge()
		s.NoError(err)
		s.mustSettleExternalCreditPurchase(ctx, creditPurchaseCharge.GetChargeID())

		usageChargeRes, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				s.createMockChargeIntent(createMockChargeIntentInput{
					customer:       cust.GetID(),
					currency:       USD,
					servicePeriod:  servicePeriod,
					settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromInt(1),
					}),
					name:              "usage-based-progressive-credit-then-invoice",
					managedBy:         billing.SubscriptionManagedLine,
					uniqueReferenceID: "usage-based-progressive-credit-then-invoice",
					featureKey:        apiRequestsTotal.Feature.Key,
				}),
			},
		})
		s.NoError(err)
		s.Len(usageChargeRes, 1)

		usageBasedCharge, err := usageChargeRes[0].AsUsageBasedCharge()
		s.NoError(err)
		usageBasedChargeID = usageBasedCharge.GetChargeID()
		s.Equal(usagebased.RatingEngineDelta, usageBasedCharge.State.RatingEngine)

		s.Equal(float64(7), s.mustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&costBasis)).InexactFloat64())
		s.Equal(float64(0), s.mustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&costBasis), ledger.TransactionAuthorizationStatusOpen).InexactFloat64())
		charge, err := s.mustGetChargeByID(usageBasedChargeID).AsUsageBasedCharge()
		s.NoError(err)
		s.Equal(usagebased.RatingEngineDelta, charge.State.RatingEngine)
		s.Empty(charge.Realizations)
	})

	t.Run("when the first progressive invoice is approved and covered by credits", func(t *testing.T) {
		// given:
		// - 5 units of usage are visible before the mid-period invoice cutoff
		// when:
		// - billing creates, collects, and approves the progressive invoice
		// then:
		// - the invoice totals 5 USD, is fully credited, reaches paid state, and leaves 2 USD credits available
		s.MockStreamingConnector.AddSimpleEvent(
			apiRequestsTotal.Feature.Key,
			5,
			datetime.MustParseTimeInLocation(t, "2026-01-10T00:00:00Z", time.UTC).AsTime(),
		)

		clock.FreezeTime(midPeriodInvoiceAt)
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(midPeriodInvoiceAt),
		})
		s.NoError(err)
		s.Len(invoices, 1)
		partialInvoice = invoices[0]

		clock.FreezeTime(partialInvoice.DefaultCollectionAtForStandardInvoice())
		partialInvoice, err = s.BillingService.AdvanceInvoice(ctx, partialInvoice.GetInvoiceID())
		s.NoError(err)
		s.Len(partialInvoice.Lines.OrEmpty(), 1)

		partialLine := partialInvoice.Lines.OrEmpty()[0]
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       5,
			CreditsTotal: 5,
			Total:        0,
		}, partialLine.Totals)
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       5,
			CreditsTotal: 5,
			Total:        0,
		}, partialInvoice.Totals)
		s.Equal(float64(5), partialLine.CreditsApplied.SumAmount(lo.Must(USD.Calculator())).InexactFloat64())
		s.Equal(float64(5), lo.FromPtr(partialLine.UsageBased.Quantity).InexactFloat64())
		s.Equal(float64(5), lo.FromPtr(partialLine.UsageBased.MeteredQuantity).InexactFloat64())
		s.Equal(float64(0), lo.FromPtr(partialLine.UsageBased.PreLinePeriodQuantity).InexactFloat64())
		s.Equal(float64(0), lo.FromPtr(partialLine.UsageBased.MeteredPreLinePeriodQuantity).InexactFloat64())
		s.Len(partialLine.DetailedLines, 1)
		s.Equal(billingrating.UnitPriceUsageChildUniqueReferenceID, partialLine.DetailedLines[0].ChildUniqueReferenceID)
		s.Equal(float64(5), partialLine.DetailedLines.SumTotals().CreditsTotal.InexactFloat64())

		chargeWithDetails := s.mustGetUsageBasedChargeByIDWithDetailedLines(usageBasedChargeID)
		s.Equal(usagebased.RatingEngineDelta, chargeWithDetails.State.RatingEngine)
		currentRun, err := chargeWithDetails.GetCurrentRealizationRun()
		s.NoError(err)
		s.Equal(usagebased.RealizationRunTypePartialInvoice, currentRun.Type)
		s.Equal(float64(5), currentRun.MeteredQuantity.InexactFloat64())
		s.Equal(float64(5), currentRun.DetailedLines.OrEmpty().SumTotals().Amount.InexactFloat64())

		partialInvoice, err = s.BillingService.ApproveInvoice(ctx, partialInvoice.GetInvoiceID())
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, partialInvoice.Status)

		charge, err := s.mustGetChargeByID(usageBasedChargeID).AsUsageBasedCharge()
		s.NoError(err)
		s.Equal(usagebased.StatusActive, charge.Status)
		s.Len(charge.Realizations, 1)
		s.True(charge.Realizations[0].NoFiatTransactionRequired)
		s.NotNil(charge.Realizations[0].InvoiceUsage)
		s.Nil(charge.Realizations[0].Payment)

		partialInvoice, err = s.CustomInvoicingService.HandlePaymentTrigger(ctx, appcustominvoicing.HandlePaymentTriggerInput{
			InvoiceID: partialInvoice.GetInvoiceID(),
			Trigger:   billing.TriggerPaid,
		})
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaid, partialInvoice.Status)

		charge, err = s.mustGetChargeByID(usageBasedChargeID).AsUsageBasedCharge()
		s.NoError(err)
		s.Equal(usagebased.StatusActive, charge.Status)
		s.Len(charge.Realizations, 1)
		s.True(charge.Realizations[0].NoFiatTransactionRequired)
		s.Nil(charge.Realizations[0].Payment)

		s.Equal(float64(2), s.mustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&costBasis)).InexactFloat64())
		s.Equal(float64(0), s.mustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&costBasis), ledger.TransactionAuthorizationStatusOpen).InexactFloat64())
		s.Equal(float64(5), s.mustCustomerAccruedBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal]()).InexactFloat64())
	})

	t.Run("when the final invoice is approved after additional usage", func(t *testing.T) {
		// given:
		// - the first progressive invoice is fully covered by credits
		// - 15 more units become visible before the final service-period cutoff
		// when:
		// - billing creates, collects, and approves the final invoice
		// then:
		// - the final invoice bills only the 15 USD delta, consumes the remaining 2 USD credits, and leaves 13 USD due
		s.MockStreamingConnector.AddSimpleEvent(
			apiRequestsTotal.Feature.Key,
			15,
			datetime.MustParseTimeInLocation(t, "2026-01-25T00:00:00Z", time.UTC).AsTime(),
		)

		clock.FreezeTime(servicePeriod.To)
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(servicePeriod.To),
		})
		s.NoError(err)
		s.Len(invoices, 1)
		finalInvoice = invoices[0]

		clock.FreezeTime(finalInvoice.DefaultCollectionAtForStandardInvoice())
		finalInvoice, err = s.BillingService.AdvanceInvoice(ctx, finalInvoice.GetInvoiceID())
		s.NoError(err)
		s.Len(finalInvoice.Lines.OrEmpty(), 1)

		finalLine := finalInvoice.Lines.OrEmpty()[0]
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       15,
			CreditsTotal: 2,
			Total:        13,
		}, finalLine.Totals)
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       15,
			CreditsTotal: 2,
			Total:        13,
		}, finalInvoice.Totals)
		s.Equal(float64(2), finalLine.CreditsApplied.SumAmount(lo.Must(USD.Calculator())).InexactFloat64())
		s.Equal(float64(15), lo.FromPtr(finalLine.UsageBased.Quantity).InexactFloat64())
		s.Equal(float64(15), lo.FromPtr(finalLine.UsageBased.MeteredQuantity).InexactFloat64())
		s.Equal(float64(5), lo.FromPtr(finalLine.UsageBased.PreLinePeriodQuantity).InexactFloat64())
		s.Equal(float64(5), lo.FromPtr(finalLine.UsageBased.MeteredPreLinePeriodQuantity).InexactFloat64())
		s.Len(finalLine.DetailedLines, 1)
		s.Equal(billingrating.UnitPriceUsageChildUniqueReferenceID, finalLine.DetailedLines[0].ChildUniqueReferenceID)
		s.Equal(float64(2), finalLine.DetailedLines.SumTotals().CreditsTotal.InexactFloat64())

		chargeWithDetails := s.mustGetUsageBasedChargeByIDWithDetailedLines(usageBasedChargeID)
		s.Equal(usagebased.RatingEngineDelta, chargeWithDetails.State.RatingEngine)
		currentRun, err := chargeWithDetails.GetCurrentRealizationRun()
		s.NoError(err)
		s.Equal(usagebased.RealizationRunTypeFinalRealization, currentRun.Type)
		s.Equal(float64(20), currentRun.MeteredQuantity.InexactFloat64())
		s.Equal(float64(15), currentRun.DetailedLines.OrEmpty().SumTotals().Amount.InexactFloat64())

		finalInvoice, err = s.BillingService.ApproveInvoice(ctx, finalInvoice.GetInvoiceID())
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, finalInvoice.Status)

		charge, err := s.mustGetChargeByID(usageBasedChargeID).AsUsageBasedCharge()
		s.NoError(err)
		s.Equal(usagebased.StatusActiveAwaitingPaymentSettlement, charge.Status)
		s.Len(charge.Realizations, 2)
		s.True(charge.Realizations[0].NoFiatTransactionRequired)
		s.Nil(charge.Realizations[0].Payment)
		s.False(charge.Realizations[1].NoFiatTransactionRequired)

		s.Equal(float64(0), s.mustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&costBasis)).InexactFloat64())
		s.Equal(float64(-13), s.mustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&costBasis), ledger.TransactionAuthorizationStatusOpen).InexactFloat64())
		s.Equal(float64(-13), s.mustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen).InexactFloat64())
		s.Equal(float64(20), s.mustCustomerAccruedBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal]()).InexactFloat64())
		s.Equal(float64(-7), s.mustWashBalance(ns, USD, mo.None[*alpacadecimal.Decimal]()).InexactFloat64())
	})

	t.Run("when the final invoice payment is authorized and settled", func(t *testing.T) {
		// given:
		// - the final invoice has 13 USD due after credits
		// when:
		// - the invoice payment is authorized and then settled
		// then:
		// - all invoice receivables are closed and the usage-based charge reaches final
		var err error
		finalInvoice, err = s.BillingService.PaymentAuthorized(ctx, finalInvoice.GetInvoiceID())
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingAuthorized, finalInvoice.Status)
		s.Equal(float64(-13), s.mustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&costBasis), ledger.TransactionAuthorizationStatusAuthorized).InexactFloat64())

		finalInvoice, err = s.CustomInvoicingService.HandlePaymentTrigger(ctx, appcustominvoicing.HandlePaymentTriggerInput{
			InvoiceID: finalInvoice.GetInvoiceID(),
			Trigger:   billing.TriggerPaid,
		})
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaid, finalInvoice.Status)

		charge, err := s.mustGetChargeByID(usageBasedChargeID).AsUsageBasedCharge()
		s.NoError(err)
		s.Equal(usagebased.StatusFinal, charge.Status)
		s.Len(charge.Realizations, 2)
		s.True(charge.Realizations[0].NoFiatTransactionRequired)
		s.Nil(charge.Realizations[0].Payment)
		s.False(charge.Realizations[1].NoFiatTransactionRequired)
		s.NotNil(charge.Realizations[1].Payment)
		s.Equal(payment.StatusSettled, charge.Realizations[1].Payment.Status)

		s.Equal(float64(0), s.mustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&costBasis), ledger.TransactionAuthorizationStatusOpen).InexactFloat64())
		s.Equal(float64(0), s.mustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&costBasis), ledger.TransactionAuthorizationStatusAuthorized).InexactFloat64())
		s.Equal(float64(0), s.mustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen).InexactFloat64())
		s.Equal(float64(0), s.mustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusAuthorized).InexactFloat64())
	})
}

func (s *CreditsTestSuite) TestUsageBasedCreditThenInvoiceLateCollectionUsageRequiresFiatPayment() {
	t := s.T()
	ctx := t.Context()
	ns := s.GetUniqueNamespace("charges-credits-usagebased-credit-then-invoice-late-usage")

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.createLedgerBackedCustomer(ns, "test-subject")

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(t, "P2D")),
		billingtest.WithManualApproval(),
	)

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)

	setupAt := datetime.MustParseTimeInLocation(t, "2025-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(t, "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(t, "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	costBasis := alpacadecimal.NewFromInt(1)

	clock.FreezeTime(setupAt)
	defer clock.UnFreeze()

	var (
		usageBasedChargeID meta.ChargeID
		invoice            billing.StandardInvoice
		collectionAt       time.Time
	)

	t.Run("given prepaid credits and a credit-then-invoice usage charge", func(t *testing.T) {
		// given:
		// - a ledger-backed customer has 5 USD prepaid credits
		// - the billing profile has a two-day collection period and manual approval
		// when:
		// - a unit-priced credit-then-invoice usage charge is created
		// then:
		// - the credits are available and the charge is ready for final invoicing
		creditPurchaseIntent := s.createCreditPurchaseIntent(createCreditPurchaseIntentInput{
			customer:      cust.GetID(),
			currency:      USD,
			amount:        alpacadecimal.NewFromInt(5),
			servicePeriod: timeutil.ClosedPeriod{From: setupAt, To: setupAt},
			settlement: creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{
				GenericSettlement: creditpurchase.GenericSettlement{
					Currency:  USD,
					CostBasis: costBasis,
				},
				InitialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus,
			}),
		})

		creditPurchaseRes, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				creditPurchaseIntent,
			},
		})
		s.NoError(err)
		s.Len(creditPurchaseRes, 1)

		creditPurchaseCharge, err := creditPurchaseRes[0].AsCreditPurchaseCharge()
		s.NoError(err)
		s.mustSettleExternalCreditPurchase(ctx, creditPurchaseCharge.GetChargeID())

		usageChargeRes, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				s.createMockChargeIntent(createMockChargeIntentInput{
					customer:       cust.GetID(),
					currency:       USD,
					servicePeriod:  servicePeriod,
					settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromInt(1),
					}),
					name:              "usage-based-credit-then-invoice-late-usage",
					managedBy:         billing.SubscriptionManagedLine,
					uniqueReferenceID: "usage-based-credit-then-invoice-late-usage",
					featureKey:        apiRequestsTotal.Feature.Key,
				}),
			},
		})
		s.NoError(err)
		s.Len(usageChargeRes, 1)

		usageBasedCharge, err := usageChargeRes[0].AsUsageBasedCharge()
		s.NoError(err)
		usageBasedChargeID = usageBasedCharge.GetChargeID()
		s.Equal(float64(5), s.mustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&costBasis)).InexactFloat64())
	})

	t.Run("when invoice creation sees usage fully covered by credits", func(t *testing.T) {
		// given:
		// - 5 USD of usage is visible before the invoice is created
		// when:
		// - final invoicing starts and creates an invoice waiting for collection
		// then:
		// - no fiat payment is required for the current run
		s.MockStreamingConnector.AddSimpleEvent(
			apiRequestsTotal.Feature.Key,
			5,
			servicePeriod.From.Add(10*24*time.Hour),
			streamingtestutils.WithStoredAt(servicePeriod.To.Add(-time.Hour)),
		)

		clock.FreezeTime(servicePeriod.To)
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(servicePeriod.To),
		})
		s.NoError(err)
		s.Len(invoices, 1)
		invoice = invoices[0]
		collectionAt = invoice.DefaultCollectionAtForStandardInvoice()
		s.Equal(billing.StandardInvoiceStatusDraftWaitingForCollection, invoice.Status)
		s.Len(invoice.Lines.OrEmpty(), 1)
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       5,
			CreditsTotal: 5,
			Total:        0,
		}, invoice.Totals)

		charge, err := s.mustGetChargeByID(usageBasedChargeID).AsUsageBasedCharge()
		s.NoError(err)
		currentRun, err := charge.GetCurrentRealizationRun()
		s.NoError(err)
		s.True(currentRun.NoFiatTransactionRequired)
		s.Nil(currentRun.Payment)
	})

	t.Run("when a late event arrives during collection", func(t *testing.T) {
		// given:
		// - the draft invoice initially had no fiat total
		// - one additional usage event is stored during the collection window
		// when:
		// - collection completes and the invoice snapshots usage again
		// then:
		// - the invoice has 1 USD due and the run requires a fiat payment
		s.MockStreamingConnector.AddSimpleEvent(
			apiRequestsTotal.Feature.Key,
			1,
			servicePeriod.To.Add(-time.Hour),
			streamingtestutils.WithStoredAt(servicePeriod.To.Add(12*time.Hour)),
		)

		clock.FreezeTime(collectionAt)
		var err error
		invoice, err = s.BillingService.AdvanceInvoice(ctx, invoice.GetInvoiceID())
		s.NoError(err)
		s.Len(invoice.Lines.OrEmpty(), 1)
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       6,
			CreditsTotal: 5,
			Total:        1,
		}, invoice.Totals)

		line := invoice.Lines.OrEmpty()[0]
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       6,
			CreditsTotal: 5,
			Total:        1,
		}, line.Totals)
		s.Equal(float64(6), lo.FromPtr(line.UsageBased.Quantity).InexactFloat64())
		s.Equal(float64(6), lo.FromPtr(line.UsageBased.MeteredQuantity).InexactFloat64())

		charge, err := s.mustGetChargeByID(usageBasedChargeID).AsUsageBasedCharge()
		s.NoError(err)
		currentRun, err := charge.GetCurrentRealizationRun()
		s.NoError(err)
		s.False(currentRun.NoFiatTransactionRequired)
		s.Nil(currentRun.Payment)
	})

	t.Run("when the invoice is approved and paid", func(t *testing.T) {
		// given:
		// - the invoice has 1 USD due after consumed credits
		// when:
		// - the invoice is approved, payment is authorized, and then paid
		// then:
		// - usage-based payment booking records a settled payment and the charge reaches final
		var err error
		invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)

		charge, err := s.mustGetChargeByID(usageBasedChargeID).AsUsageBasedCharge()
		s.NoError(err)
		s.Equal(usagebased.StatusActiveAwaitingPaymentSettlement, charge.Status)
		s.Len(charge.Realizations, 1)
		s.False(charge.Realizations[0].NoFiatTransactionRequired)
		s.NotNil(charge.Realizations[0].InvoiceUsage)
		s.Nil(charge.Realizations[0].Payment)
		s.Equal(float64(-1), s.mustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&costBasis), ledger.TransactionAuthorizationStatusOpen).InexactFloat64())

		invoice, err = s.BillingService.PaymentAuthorized(ctx, invoice.GetInvoiceID())
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingAuthorized, invoice.Status)
		s.Equal(float64(-1), s.mustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&costBasis), ledger.TransactionAuthorizationStatusAuthorized).InexactFloat64())

		invoice, err = s.CustomInvoicingService.HandlePaymentTrigger(ctx, appcustominvoicing.HandlePaymentTriggerInput{
			InvoiceID: invoice.GetInvoiceID(),
			Trigger:   billing.TriggerPaid,
		})
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaid, invoice.Status)

		charge, err = s.mustGetChargeByID(usageBasedChargeID).AsUsageBasedCharge()
		s.NoError(err)
		s.Equal(usagebased.StatusFinal, charge.Status)
		s.Len(charge.Realizations, 1)
		s.False(charge.Realizations[0].NoFiatTransactionRequired)
		s.NotNil(charge.Realizations[0].Payment)
		s.Equal(payment.StatusSettled, charge.Realizations[0].Payment.Status)
		s.Equal(float64(0), s.mustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&costBasis), ledger.TransactionAuthorizationStatusOpen).InexactFloat64())
		s.Equal(float64(0), s.mustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&costBasis), ledger.TransactionAuthorizationStatusAuthorized).InexactFloat64())
	})
}

func (s *CreditsTestSuite) mustGetUsageBasedChargeByIDWithDetailedLines(chargeID meta.ChargeID) usagebased.Charge {
	s.T().Helper()

	charge, err := s.Charges.GetByID(s.T().Context(), charges.GetByIDInput{
		ChargeID: chargeID,
		Expands: meta.Expands{
			meta.ExpandRealizations,
			meta.ExpandDetailedLines,
		},
	})
	s.NoError(err)

	usageBasedCharge, err := charge.AsUsageBasedCharge()
	s.NoError(err)

	return usageBasedCharge
}
