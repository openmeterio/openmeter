package service

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

func TestUsageBasedCharges(t *testing.T) {
	suite.Run(t, new(UsageBasedChargesTestSuite))
}

type UsageBasedChargesTestSuite struct {
	BaseSuite
}

func (s *UsageBasedChargesTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()
}

func (s *UsageBasedChargesTestSuite) TearDownTest() {
	s.BaseSuite.TearDownTest()
}

func (s *UsageBasedChargesTestSuite) TestUsageBasedCreditThenInvoicePartialInvoiceLifecycle() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-usage-based-partial-invoice-lifecycle")

	customInvoicing := s.SetupCustomInvoicing(ns)

	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithProgressiveBilling(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
		billingtest.WithManualApproval(),
	)

	createAt := datetime.MustParseTimeInLocation(s.T(), "2025-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	midPeriodInvoiceAt := datetime.MustParseTimeInLocation(s.T(), "2026-01-16T00:00:00Z", time.UTC).AsTime()
	secondPartialAttemptAt := datetime.MustParseTimeInLocation(s.T(), "2026-01-21T00:00:00Z", time.UTC).AsTime()

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)
	meterSlug := apiRequestsTotal.Feature.Key

	graduatedTieredPrice := productcatalog.NewPriceFrom(productcatalog.TieredPrice{
		Mode: productcatalog.GraduatedTieredPrice,
		Tiers: []productcatalog.PriceTier{
			{
				UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(10)),
				UnitPrice: &productcatalog.PriceTierUnitPrice{
					Amount: alpacadecimal.NewFromFloat(1),
				},
			},
			{
				UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(20)),
				UnitPrice: &productcatalog.PriceTierUnitPrice{
					Amount: alpacadecimal.NewFromFloat(0.5),
				},
			},
			{
				UnitPrice: &productcatalog.PriceTierUnitPrice{
					Amount: alpacadecimal.NewFromFloat(0.25),
				},
			},
		},
	})

	clock.FreezeTime(createAt)
	defer clock.UnFreeze()
	defer s.UsageBasedTestHandler.Reset()

	s.UsageBasedTestHandler.onCreditsOnlyUsageAccrued, _ = newCappedCreditAllocator(0)

	var (
		usageBasedChargeID meta.ChargeID
		partialInvoice     billing.StandardInvoice
		finalInvoice       billing.StandardInvoice
		partialRunID       string
	)

	s.Run("given a graduated tiered usage-based charge", func() {
		// given:
		// - a credit-then-invoice usage-based charge with graduated tiered pricing
		// when:
		// - the charge is created
		// then:
		// - it starts in created status without realization runs
		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: []charges.ChargeIntent{
				s.createMockChargeIntent(createMockChargeIntentInput{
					customer:          cust.GetID(),
					currency:          USD,
					servicePeriod:     servicePeriod,
					settlementMode:    productcatalog.CreditThenInvoiceSettlementMode,
					price:             graduatedTieredPrice,
					name:              "usage-based-partial-invoice",
					managedBy:         billing.SubscriptionManagedLine,
					uniqueReferenceID: "usage-based-partial-invoice",
					featureKey:        meterSlug,
				}),
			},
		})
		s.NoError(err)
		s.Len(res, 1)

		usageBasedCharge, err := res[0].AsUsageBasedCharge()
		s.NoError(err)
		usageBasedChargeID = usageBasedCharge.GetChargeID()

		// then
		fetched := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(meta.ChargeStatusCreated, meta.ChargeStatus(fetched.Status))
		s.Empty(fetched.Realizations)
	})

	s.Run("when partially invoiced at service period start", func() {
		// given:
		// - the usage-based charge exists at the exact service period start
		// when:
		// - billing tries to invoice pending lines immediately
		// then:
		// - no invoice is created and the charge remains uninvoiced
		clock.FreezeTime(servicePeriod.From)

		// when
		_, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(servicePeriod.From),
		})

		// then
		s.Error(err)
		s.ErrorAs(err, &billing.ValidationError{})
		s.ErrorIs(err, billing.ErrInvoiceCreateNoLines)

		charge := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Empty(charge.Realizations)
		s.Equal(usagebased.StatusCreated, charge.Status)
	})

	s.Run("when partially invoiced mid period", func() {
		// given:
		// - mid-period usage exists for the charge
		// when:
		// - billing invoices pending lines mid period
		// then:
		// - a partial invoice is created and a partial realization run becomes active
		s.MockStreamingConnector.AddSimpleEvent(
			meterSlug,
			15,
			datetime.MustParseTimeInLocation(s.T(), "2026-01-15T00:00:00Z", time.UTC).AsTime(),
		)
		clock.FreezeTime(midPeriodInvoiceAt)

		// when
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(midPeriodInvoiceAt),
		})

		// then
		s.Require().NoError(err)
		s.Len(invoices, 1)

		partialInvoice = invoices[0]
		s.Require().Len(partialInvoice.Lines.OrEmpty(), 1)

		stdLine := partialInvoice.Lines.OrEmpty()[0]
		expectedPartialCollectionEnd := midPeriodInvoiceAt.Add(usagebased.InternalCollectionPeriod)
		s.Require().NotNil(stdLine.OverrideCollectionPeriodEnd)
		s.True(expectedPartialCollectionEnd.Equal(*stdLine.OverrideCollectionPeriodEnd))
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount: 12.5,
			Total:  12.5,
		}, stdLine.Totals)
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount: 12.5,
			Total:  12.5,
		}, partialInvoice.Totals)

		charge := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(usagebased.StatusActivePartialInvoiceWaitingForCollection, charge.Status)
		s.Len(charge.Realizations, 1)

		currentRun, err := charge.GetCurrentRealizationRun()
		s.Require().NoError(err)
		s.Equal(usagebased.RealizationRunTypePartialInvoice, currentRun.Type)
		s.Require().NotNil(currentRun.LineID)
		s.Equal(stdLine.ID, *currentRun.LineID)
		s.Require().NotNil(currentRun.InvoiceID)
		s.Equal(partialInvoice.ID, *currentRun.InvoiceID)
		s.True(midPeriodInvoiceAt.Equal(currentRun.ServicePeriodTo))
		s.True(midPeriodInvoiceAt.Equal(currentRun.StoredAtLT))
		s.Require().NotNil(partialInvoice.CollectionAt)
		s.True(expectedPartialCollectionEnd.Equal(*partialInvoice.CollectionAt))

		partialRunID = currentRun.ID.ID

		invoicesResult, err := s.BillingService.ListStandardInvoices(ctx, billing.ListStandardInvoicesInput{
			Namespaces: []string{ns},
		})
		s.NoError(err)
		s.Len(invoicesResult.Items, 1)
	})

	s.Run("when partially invoiced again before the first mid period invoice is issued", func() {
		// given:
		// - a partial realization run is already active for the charge
		// when:
		// - billing tries to invoice pending lines again before the first invoice is issued
		// then:
		// - the request is rejected and no additional run or invoice is created
		s.MockStreamingConnector.AddSimpleEvent(
			meterSlug,
			5,
			datetime.MustParseTimeInLocation(s.T(), "2026-01-20T00:00:00Z", time.UTC).AsTime(),
		)
		clock.FreezeTime(secondPartialAttemptAt)

		// when
		_, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(secondPartialAttemptAt),
		})

		// then
		s.Error(err)
		s.ErrorAs(err, &billing.ValidationError{})
		s.ErrorIs(err, usagebased.ErrActiveRealizationRunAlreadyExists)

		charge := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(usagebased.StatusActivePartialInvoiceWaitingForCollection, charge.Status)
		s.Len(charge.Realizations, 1)

		currentRun, runErr := charge.GetCurrentRealizationRun()
		s.NoError(runErr)
		s.Equal(partialRunID, currentRun.ID.ID)

		invoicesResult, listErr := s.BillingService.ListStandardInvoices(ctx, billing.ListStandardInvoicesInput{
			Namespaces: []string{ns},
		})
		s.NoError(listErr)
		s.Len(invoicesResult.Items, 1)
		s.Equal(partialInvoice.ID, invoicesResult.Items[0].ID)
	})

	s.Run("when the first partial invoice is advanced and approved", func() {
		// given:
		// - the first partial invoice is ready to be collected and manually approved
		// when:
		// - the invoice is advanced and then approved
		// then:
		// - the run waits in processing until issuance, then accrues invoice usage and returns to active
		defer s.UsageBasedTestHandler.Reset()

		clock.FreezeTime(partialInvoice.DefaultCollectionAtForStandardInvoice())

		// when
		invoice, err := s.BillingService.AdvanceInvoice(ctx, partialInvoice.GetInvoiceID())

		// then
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, invoice.Status)

		charge := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(usagebased.StatusActivePartialInvoiceProcessing, charge.Status)

		invoiceUsageAccruedCallback := newCountedLedgerTransactionCallback[usagebased.OnInvoiceUsageAccruedInput]()
		s.UsageBasedTestHandler.onInvoiceUsageAccrued = invoiceUsageAccruedCallback.Handler(s.T())

		// when
		invoice, err = s.BillingService.ApproveInvoice(ctx, partialInvoice.GetInvoiceID())

		// then
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)
		s.Equal(1, invoiceUsageAccruedCallback.nrInvocations)

		charge = s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(usagebased.StatusActive, charge.Status)
	})

	s.Run("when the final invoice is created and the final realization completes after the service period", func() {
		// given:
		// - more usage arrives and the earlier partial invoice is already approved
		// when:
		// - billing invoices again after the service period and later advances collection
		// then:
		// - a final realization run is created and reaches processing before issuance
		defer s.UsageBasedTestHandler.Reset()

		s.UsageBasedTestHandler.onCreditsOnlyUsageAccrued, _ = newCappedCreditAllocator(0)
		s.MockStreamingConnector.AddSimpleEvent(
			meterSlug,
			10,
			datetime.MustParseTimeInLocation(s.T(), "2026-01-25T00:00:00Z", time.UTC).AsTime(),
		)
		clock.FreezeTime(servicePeriod.To)

		// when
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(servicePeriod.To),
		})

		// then
		s.NoError(err)
		s.Len(invoices, 1)
		finalInvoice = invoices[0]
		// TODO[rating]: totals are off due to rating not yet supporting progressive billing via charges

		charge := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(usagebased.StatusActiveFinalRealizationWaitingForCollection, charge.Status)
		s.Len(charge.Realizations, 2)

		currentRun, runErr := charge.GetCurrentRealizationRun()
		s.NoError(runErr)
		s.Equal(usagebased.RealizationRunTypeFinalRealization, currentRun.Type)
		s.Require().NotNil(currentRun.InvoiceID)
		s.Equal(finalInvoice.ID, *currentRun.InvoiceID)

		// given
		clock.FreezeTime(finalInvoice.DefaultCollectionAtForStandardInvoice())

		// when
		finalInvoice, err = s.BillingService.AdvanceInvoice(ctx, finalInvoice.GetInvoiceID())

		// then
		s.NoError(err)

		charge = s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(usagebased.StatusActiveFinalRealizationProcessing, charge.Status)

		currentRun, runErr = charge.GetCurrentRealizationRun()
		s.NoError(runErr)
		s.Equal(usagebased.RealizationRunTypeFinalRealization, currentRun.Type)
	})

	s.Run("when the final invoice is approved while the partial invoice is still unpaid", func() {
		// given:
		// - the final realization run is processing and the earlier partial invoice payment is still unsettled
		// when:
		// - the final invoice is approved
		// then:
		// - the final run accrues invoice usage and the charge waits for payment settlement
		defer s.UsageBasedTestHandler.Reset()

		invoiceUsageAccruedCallback := newCountedLedgerTransactionCallback[usagebased.OnInvoiceUsageAccruedInput]()
		s.UsageBasedTestHandler.onInvoiceUsageAccrued = invoiceUsageAccruedCallback.Handler(s.T())

		// when
		approvedInvoice, err := s.BillingService.ApproveInvoice(ctx, finalInvoice.GetInvoiceID())

		// then
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, approvedInvoice.Status)
		s.Equal(1, invoiceUsageAccruedCallback.nrInvocations)

		finalInvoice = approvedInvoice

		charge := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(usagebased.StatusActiveAwaitingPaymentSettlement, charge.Status)
		s.Nil(charge.State.CurrentRealizationRunID)
		s.Len(charge.Realizations, 2)
	})

	s.Run("when the final invoice is paid before the partial invoice is settled", func() {
		// given:
		// - the charge is awaiting payment settlement with the partial invoice still unpaid
		// when:
		// - the final invoice is paid
		// then:
		// - the charge keeps waiting because not all invoiced runs are settled yet
		defer s.UsageBasedTestHandler.Reset()

		authorizedCallback := newCountedLedgerTransactionCallback[usagebased.OnPaymentAuthorizedInput]()
		s.UsageBasedTestHandler.onPaymentAuthorized = authorizedCallback.Handler(s.T())
		settledCallback := newCountedLedgerTransactionCallback[usagebased.OnPaymentSettledInput]()
		s.UsageBasedTestHandler.onPaymentSettled = settledCallback.Handler(s.T())

		// when
		paidInvoice, err := s.CustomInvoicingService.HandlePaymentTrigger(ctx, appcustominvoicing.HandlePaymentTriggerInput{
			InvoiceID: finalInvoice.GetInvoiceID(),
			Trigger:   billing.TriggerPaid,
		})

		// then
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaid, paidInvoice.Status)
		s.Equal(1, authorizedCallback.nrInvocations)
		s.Equal(1, settledCallback.nrInvocations)

		finalInvoice = paidInvoice

		charge := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(usagebased.StatusActiveAwaitingPaymentSettlement, charge.Status)
		s.Len(charge.Realizations, 2)
	})

	s.Run("when the outstanding partial invoice is finally paid", func() {
		// given:
		// - the final invoice is already settled but the earlier partial invoice is still unpaid
		// when:
		// - the partial invoice is paid
		// then:
		// - all invoiced runs are settled and the charge reaches final
		defer s.UsageBasedTestHandler.Reset()

		authorizedCallback := newCountedLedgerTransactionCallback[usagebased.OnPaymentAuthorizedInput]()
		s.UsageBasedTestHandler.onPaymentAuthorized = authorizedCallback.Handler(s.T())
		settledCallback := newCountedLedgerTransactionCallback[usagebased.OnPaymentSettledInput]()
		s.UsageBasedTestHandler.onPaymentSettled = settledCallback.Handler(s.T())

		// when
		paidInvoice, err := s.CustomInvoicingService.HandlePaymentTrigger(ctx, appcustominvoicing.HandlePaymentTriggerInput{
			InvoiceID: partialInvoice.GetInvoiceID(),
			Trigger:   billing.TriggerPaid,
		})

		// then
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaid, paidInvoice.Status)
		s.Equal(1, authorizedCallback.nrInvocations)
		s.Equal(1, settledCallback.nrInvocations)

		partialInvoice = paidInvoice

		charge := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(usagebased.StatusFinal, charge.Status)
		s.Len(charge.Realizations, 2)
		s.Nil(charge.State.CurrentRealizationRunID)
	})
}

func (s *UsageBasedChargesTestSuite) TestUsageBasedCreditThenInvoicePendingPartialInvoiceBlocksFinalRealizationUntilApproval() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-usage-based-pending-partial-invoice-blocks-final")

	customInvoicing := s.SetupCustomInvoicing(ns)

	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithProgressiveBilling(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
		billingtest.WithManualApproval(),
	)

	createAt := datetime.MustParseTimeInLocation(s.T(), "2025-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	midPeriodInvoiceAt := datetime.MustParseTimeInLocation(s.T(), "2026-01-16T00:00:00Z", time.UTC).AsTime()

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)
	meterSlug := apiRequestsTotal.Feature.Key

	clock.FreezeTime(createAt)
	defer clock.UnFreeze()
	defer s.UsageBasedTestHandler.Reset()

	s.UsageBasedTestHandler.onCreditsOnlyUsageAccrued, _ = newCappedCreditAllocator(0)

	var (
		usageBasedChargeID meta.ChargeID
		partialInvoice     billing.StandardInvoice
	)

	s.Run("given a credit-then-invoice usage-based charge", func() {
		// given:
		// - a credit-then-invoice usage-based charge with unit pricing
		// when:
		// - the charge is created
		// then:
		// - it starts in created status
		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: []charges.ChargeIntent{
				s.createMockChargeIntent(createMockChargeIntentInput{
					customer:       cust.GetID(),
					currency:       USD,
					servicePeriod:  servicePeriod,
					settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromFloat(1),
					}),
					name:              "usage-based-partial-invoice-pending-blocks-final",
					managedBy:         billing.SubscriptionManagedLine,
					uniqueReferenceID: "usage-based-partial-invoice-pending-blocks-final",
					featureKey:        meterSlug,
				}),
			},
		})
		s.NoError(err)
		s.Len(res, 1)

		usageBasedCharge, err := res[0].AsUsageBasedCharge()
		s.NoError(err)
		usageBasedChargeID = usageBasedCharge.GetChargeID()

		// then
		charge := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(usagebased.StatusCreated, charge.Status)
	})

	s.Run("when a partial invoice is created mid period and left waiting for manual approval", func() {
		// given:
		// - mid-period usage exists for the charge
		// when:
		// - billing creates a partial invoice and the invoice is advanced but not approved
		// then:
		// - the charge remains on the processing partial-invoice branch while waiting for approval
		s.MockStreamingConnector.AddSimpleEvent(
			meterSlug,
			10,
			datetime.MustParseTimeInLocation(s.T(), "2026-01-15T00:00:00Z", time.UTC).AsTime(),
		)
		clock.FreezeTime(midPeriodInvoiceAt)

		// when
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(midPeriodInvoiceAt),
		})

		// then
		s.NoError(err)
		s.Len(invoices, 1)
		partialInvoice = invoices[0]

		// given
		clock.FreezeTime(partialInvoice.DefaultCollectionAtForStandardInvoice())

		// when
		partialInvoice, err = s.BillingService.AdvanceInvoice(ctx, partialInvoice.GetInvoiceID())

		// then
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, partialInvoice.Status)

		charge := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(usagebased.StatusActivePartialInvoiceProcessing, charge.Status)
	})

	s.Run("when the service period ends before the partial invoice is approved", func() {
		// given:
		// - the partial invoice still owns the active realization run while the branch is processing
		// when:
		// - billing tries to invoice pending lines for the final period
		// then:
		// - final realization is blocked by the active-run invariant
		s.MockStreamingConnector.AddSimpleEvent(
			meterSlug,
			5,
			datetime.MustParseTimeInLocation(s.T(), "2026-01-25T00:00:00Z", time.UTC).AsTime(),
		)
		clock.FreezeTime(servicePeriod.To)

		// when
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(servicePeriod.To),
		})

		// then
		s.Error(err)
		s.ErrorAs(err, &billing.ValidationError{})
		s.ErrorIs(err, usagebased.ErrActiveRealizationRunAlreadyExists)
		s.Nil(invoices)

		charge := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(usagebased.StatusActivePartialInvoiceProcessing, charge.Status)

		invoicesResult, listErr := s.BillingService.ListStandardInvoices(ctx, billing.ListStandardInvoicesInput{
			Namespaces: []string{ns},
		})
		s.NoError(listErr)
		s.Len(invoicesResult.Items, 1)
		s.Equal(partialInvoice.ID, invoicesResult.Items[0].ID)
	})

	s.Run("when the pending partial invoice is approved after the service period end", func() {
		// given:
		// - the partial invoice is still pending manual approval after the service period end
		// when:
		// - the invoice is approved
		// then:
		// - invoice usage is accrued and the charge returns to active
		defer s.UsageBasedTestHandler.Reset()

		clock.FreezeTime(servicePeriod.To)
		invoiceUsageAccruedCallback := newCountedLedgerTransactionCallback[usagebased.OnInvoiceUsageAccruedInput]()
		s.UsageBasedTestHandler.onInvoiceUsageAccrued = invoiceUsageAccruedCallback.Handler(s.T())

		// when
		partialInvoice, err := s.BillingService.ApproveInvoice(ctx, partialInvoice.GetInvoiceID())

		// then
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, partialInvoice.Status)
		s.Equal(1, invoiceUsageAccruedCallback.nrInvocations)

		charge := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(usagebased.StatusActive, charge.Status)
	})

	s.Run("when invoice pending lines is retried after the partial invoice approval", func() {
		// given:
		// - the previously blocking partial invoice has been approved
		// when:
		// - billing retries invoice pending lines after the service period
		// then:
		// - final realization can start successfully
		defer s.UsageBasedTestHandler.Reset()

		s.UsageBasedTestHandler.onCreditsOnlyUsageAccrued, _ = newCappedCreditAllocator(0)
		clock.FreezeTime(servicePeriod.To)

		// when
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(servicePeriod.To),
		})

		// then
		s.NoError(err)
		s.Len(invoices, 1)

		charge := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(usagebased.StatusActiveFinalRealizationWaitingForCollection, charge.Status)

		currentRun, runErr := charge.GetCurrentRealizationRun()
		s.NoError(runErr)
		s.Equal(usagebased.RealizationRunTypeFinalRealization, currentRun.Type)
	})
}

func (s *UsageBasedChargesTestSuite) mustGetUsageBasedChargeByID(chargeID meta.ChargeID) usagebased.Charge {
	s.T().Helper()

	charge := s.mustGetChargeByID(chargeID)
	usageBasedCharge, err := charge.AsUsageBasedCharge()
	s.NoError(err)

	return usageBasedCharge
}
