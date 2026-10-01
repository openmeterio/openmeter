package credits

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

func (s *CreditThenInvoiceTestSuite) TestIssuingSkipsDeletedChargeLine() {
	for _, chargeType := range []meta.ChargeType{meta.ChargeTypeFlatFee, meta.ChargeTypeUsageBased} {
		s.Run(string(chargeType), func() {
			t := s.T()
			ctx := t.Context()
			clock.FreezeTime(time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC))
			defer clock.UnFreeze()

			// given a two-charge draft whose automatic approval deadline has passed
			fixture := s.setupChargeBookingInvoice(chargeType)
			clock.FreezeTime(fixture.Invoice.DraftUntil.Add(time.Second))
			defer clock.UnFreeze()

			// when deleting the first charge causes the remaining invoice to issue
			s.MustRefundCharge(ctx, fixture.Customer, fixture.Charges[0])
			invoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
				Invoice: fixture.Invoice.GetInvoiceID(),
				Expand: billing.StandardInvoiceExpands{
					billing.StandardInvoiceExpandLines,
					billing.StandardInvoiceExpandDeletedLines,
				},
			})
			require.NoError(t, err)
			require.Len(t, invoice.Lines.OrEmpty(), 2)
			deletedLine := invoice.Lines.GetByID(fixture.Invoice.Lines.OrEmpty()[0].ID)
			require.NotNil(t, deletedLine)
			require.NotNil(t, deletedLine.DeletedAt)
			s.RequireChargeStatus(fixture.Charges[0], meta.ChargeStatusDeleted)

			// then the deleted charge does not block booking the retained line
			s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)
			s.False(invoice.StatusDetails.Failed)
			s.False(invoice.HasCriticalValidationIssues())
			s.RequireTotals(billingtest.ExpectedTotals{Amount: 5, Total: 5}, invoice.Totals)
		})
	}
}

func (s *CreditThenInvoiceTestSuite) TestIssuingRetryPreservesCompletedChargeBooking() {
	for _, chargeType := range []meta.ChargeType{meta.ChargeTypeFlatFee, meta.ChargeTypeUsageBased} {
		s.Run(string(chargeType), func() {
			t := s.T()
			ctx := t.Context()
			clock.FreezeTime(time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC))
			defer clock.UnFreeze()

			// given issuance books the first charge before the second callback fails once
			fixture := s.setupChargeBookingInvoice(chargeType)
			clock.FreezeTime(fixture.Invoice.DraftUntil.Add(time.Second))
			defer clock.UnFreeze()
			fault := s.failIssuingAfterFirstBooking(fixture)

			failedInvoice, err := s.BillingService.AdvanceInvoice(ctx, fixture.Invoice.GetInvoiceID())
			require.NoError(t, err)
			require.Equal(t, billing.StandardInvoiceStatusIssuingChargeBookingFailed, failedInvoice.Status)
			require.Len(t, failedInvoice.Lines.OrEmpty(), 2)
			require.True(t, fault.Failed)
			require.True(t, failedInvoice.HasCriticalValidationIssues())
			booked := s.requireCompletedChargeBooking(fixture)
			ledgerInput := LedgerSnapshotInput{
				Namespace: fixture.Customer.Namespace,
				Customer:  fixture.Customer,
				Currency:  USD,
				CostBasis: mo.None[*alpacadecimal.Decimal](),
			}
			bookedLedger := s.CreateLedgerSnapshot(ledgerInput)
			require.Equal(t, float64(5), bookedLedger.Accrued.InexactFloat64())

			// when retry encounters the first booking again after the transient failure is gone
			invoice, err := s.BillingService.RetryInvoice(ctx, failedInvoice.GetInvoiceID())
			require.NoError(t, err)

			// then retry preserves the first booking and books the second charge exactly once
			s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)
			s.False(invoice.StatusDetails.Failed)
			s.False(invoice.HasCriticalValidationIssues())
			s.Equal(booked, s.requireCompletedChargeBooking(fixture))
			s.AssertLedgerSnapshotEqual(LedgerSnapshot{
				FBO:                  bookedLedger.FBO,
				Accrued:              alpacadecimal.NewFromInt(10),
				OpenReceivable:       alpacadecimal.NewFromInt(-10),
				AuthorizedReceivable: bookedLedger.AuthorizedReceivable,
				Wash:                 bookedLedger.Wash,
				Earnings:             bookedLedger.Earnings,
			}, s.CreateLedgerSnapshot(ledgerInput))
		})
	}
}

func (s *CreditThenInvoiceTestSuite) TestIssuingFailedInvoicePreservesUnsupportedCorrectionHistory() {
	t := s.T()
	ctx := t.Context()
	clock.FreezeTime(time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC))
	defer clock.UnFreeze()

	// given an immutable invoice failed after booking its retained flat fee
	fixture := s.setupChargeBookingInvoice(meta.ChargeTypeFlatFee)
	clock.FreezeTime(fixture.Invoice.DraftUntil.Add(time.Second))
	defer clock.UnFreeze()
	fault := s.failIssuingAfterFirstBooking(fixture)
	before, err := s.BillingService.AdvanceInvoice(ctx, fixture.Invoice.GetInvoiceID())
	require.NoError(t, err)
	require.True(t, fault.Failed)
	require.Equal(t, billing.StandardInvoiceStatusIssuingChargeBookingFailed, before.Status)
	require.True(t, before.StatusDetails.Immutable)
	require.True(t, before.HasCriticalValidationIssues())
	booked := s.requireCompletedChargeBooking(fixture)
	ledgerInput := LedgerSnapshotInput{
		Namespace: fixture.Customer.Namespace,
		Customer:  fixture.Customer,
		Currency:  USD,
		CostBasis: mo.None[*alpacadecimal.Decimal](),
	}
	bookedLedger := s.CreateLedgerSnapshot(ledgerInput)
	require.Equal(t, float64(5), bookedLedger.Accrued.InexactFloat64())

	// when a later shrink needs a credit note the invoice cannot provide
	period := fixture.Invoice.Lines.OrEmpty()[0].GetServicePeriod()
	shrunkTo := period.From.Add(period.To.Sub(period.From) / 2)
	patch, err := meta.NewPatchShrink(meta.NewPatchShrinkInput{
		ChangeSource:           billing.ChangeSourceSystem,
		NewServicePeriodTo:     shrunkTo,
		NewFullServicePeriodTo: period.To,
		NewBillingPeriodTo:     shrunkTo,
		NewInvoiceAt:           period.From,
	})
	require.NoError(t, err)
	require.NoError(t, s.Charges.ApplyPatches(ctx, charges.ApplyPatchesInput{
		CustomerID: fixture.Customer,
		PatchesByChargeID: map[string]charges.Patch{
			fixture.Charges[0].ID: patch,
		},
	}))

	// then correction drift is recorded without removing issued history or reversing its booking
	charge := s.RequireFlatFeeChargeStatus(fixture.Charges[0], flatfee.StatusActiveAwaitingPaymentSettlement)
	require.NotNil(t, charge.Realizations.CurrentRun)
	s.Equal(flatfee.RealizationRunTypeInvalidDueToUnsupportedCreditNote, charge.Realizations.CurrentRun.Type)
	s.Equal(booked, s.requireCompletedChargeBooking(fixture))
	s.AssertLedgerSnapshotUnchanged(ledgerInput, bookedLedger)
	s.Empty(s.mustGatheringLinesForCharge(fixture.Customer.Namespace, fixture.Customer.ID, fixture.Charges[0].ID, false))
	invoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
		Invoice: fixture.Invoice.GetInvoiceID(),
		Expand:  billing.StandardInvoiceExpandAll,
	})
	require.NoError(t, err)
	s.Equal(billing.StandardInvoiceStatusIssuingChargeBookingFailed, invoice.Status)
	s.True(invoice.StatusDetails.Immutable)
	s.True(invoice.HasCriticalValidationIssues())
	require.Len(t, invoice.Lines.OrEmpty(), 2)
	s.Nil(invoice.Lines.OrEmpty()[0].DeletedAt)
	s.Equal(booked.LineID, invoice.Lines.OrEmpty()[0].ID)
	s.RequireTotals(billingtest.ExpectedTotals{Amount: 10, Total: 10}, invoice.Totals)
	warning, found := lo.Find(invoice.ValidationIssues, func(issue billing.ValidationIssue) bool {
		return issue.Code == billing.ImmutableInvoiceHandlingNotSupportedErrorCode
	})
	require.True(t, found)
	s.Equal(billing.ValidationIssueSeverityWarning, warning.Severity)
	s.Equal(billing.ComponentName("charges.invoiceupdater"), warning.Component)
	s.Equal("lines/"+booked.LineID, warning.Path)
}

type chargeBookingInvoice struct {
	Customer customer.CustomerID
	Invoice  billing.StandardInvoice
	Charges  [2]meta.ChargeID
}

type completedChargeBooking struct {
	RunID        string
	LineID       string
	InvoiceID    string
	AccruedUsage *invoicedusage.AccruedUsage
}

// requireCompletedChargeBooking checks the durable issued run independently of
// charge scheduling: flat fees retain the current run, while usage charges release it.
func (s *CreditThenInvoiceTestSuite) requireCompletedChargeBooking(fixture chargeBookingInvoice) completedChargeBooking {
	t := s.T()
	t.Helper()
	charge := s.MustGetChargeByID(fixture.Charges[0])
	var booked completedChargeBooking
	switch charge.Type() {
	case meta.ChargeTypeFlatFee:
		flatFeeCharge, err := charge.AsFlatFeeCharge()
		require.NoError(t, err)
		require.Equal(t, flatfee.StatusActiveAwaitingPaymentSettlement, flatFeeCharge.Status)
		run := flatFeeCharge.Realizations.CurrentRun
		require.NotNil(t, run)
		require.True(t, run.Immutable)
		require.Nil(t, run.DeletedAt)
		require.Nil(t, run.Payment)
		booked = completedChargeBooking{RunID: run.ID.ID, LineID: lo.FromPtr(run.LineID), InvoiceID: lo.FromPtr(run.InvoiceID), AccruedUsage: run.AccruedUsage}
	case meta.ChargeTypeUsageBased:
		usageCharge, err := charge.AsUsageBasedCharge()
		require.NoError(t, err)
		require.Equal(t, usagebased.StatusActiveAwaitingPaymentSettlement, usageCharge.Status)
		require.Nil(t, usageCharge.State.CurrentRealizationRunID)
		run, err := usageCharge.Realizations.GetByLineID(fixture.Invoice.Lines.OrEmpty()[0].ID)
		require.NoError(t, err)
		require.True(t, run.Immutable)
		require.Nil(t, run.DeletedAt)
		require.Nil(t, run.Payment)
		booked = completedChargeBooking{RunID: run.ID.ID, LineID: lo.FromPtr(run.LineID), InvoiceID: lo.FromPtr(run.InvoiceID), AccruedUsage: run.InvoiceUsage}
	default:
		t.Fatalf("unexpected charge type: %s", charge.Type())
	}
	require.NotEmpty(t, booked.RunID)
	require.Equal(t, fixture.Invoice.Lines.OrEmpty()[0].ID, booked.LineID)
	require.Equal(t, fixture.Invoice.ID, booked.InvoiceID)
	require.NotNil(t, booked.AccruedUsage)
	return booked
}

func (s *CreditThenInvoiceTestSuite) setupChargeBookingInvoice(chargeType meta.ChargeType) chargeBookingInvoice {
	t := s.T()
	t.Helper()
	ctx := t.Context()

	ns := s.GetUniqueNamespace("issuing-charge-booking-" + string(chargeType))
	s.ProvisionDefaultTaxCodes(ctx, ns)
	invoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")
	s.ProvisionBillingProfile(ctx, ns, invoicing.App.GetID())
	period := timeutil.ClosedPeriod{From: clock.Now(), To: clock.Now().AddDate(0, 1, 0)}
	price := productcatalog.NewPriceFrom(productcatalog.FlatPrice{
		Amount:      alpacadecimal.NewFromInt(5),
		PaymentTerm: productcatalog.InAdvancePaymentTerm,
	})
	var featureKey string
	var processingStatus any = flatfee.StatusActiveRealizationProcessing
	if chargeType == meta.ChargeTypeUsageBased {
		processingStatus = usagebased.StatusActiveRealizationProcessing
		feature := s.SetupApiRequestsTotalFeature(ctx, ns)
		featureKey = feature.Feature.Key
		period = timeutil.ClosedPeriod{From: clock.Now().AddDate(0, -1, 0), To: clock.Now()}
		s.MockStreamingConnector.AddSimpleEvent(featureKey, 5, period.From.Add(15*24*time.Hour))
		price = productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)})
	}
	intents := make([]charges.ChargeIntent, 0, 2)
	for i := range 2 {
		intents = append(intents, s.CreateMockChargeIntent(CreateMockChargeIntentInput{
			Customer:       cust.GetID(),
			Currency:       USD,
			ServicePeriod:  period,
			SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
			Price:          price,
			ProRating: productcatalog.ProRatingConfig{
				Enabled: true,
				Mode:    productcatalog.ProRatingModeProratePrices,
			},
			FeatureKey:        featureKey,
			Name:              fmt.Sprintf("charge %d", i),
			ManagedBy:         billing.SubscriptionManagedLine,
			UniqueReferenceID: fmt.Sprintf("charge-%d", i),
		}))
	}
	created, err := s.Charges.Create(ctx, charges.CreateInput{Namespace: ns, Intents: charges.NewCreateChargeIntents(intents...)})
	require.NoError(t, err)
	require.Len(t, created, 2)
	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: cust.GetID(),
		AsOf:     lo.ToPtr(clock.Now()),
	})
	require.NoError(t, err)
	require.Len(t, invoices, 1)
	invoice := invoices[0]
	if chargeType == meta.ChargeTypeUsageBased {
		require.Equal(t, billing.StandardInvoiceStatusDraftWaitingForCollection, invoice.Status)
		clock.FreezeTime(invoice.DefaultCollectionAtForStandardInvoice())
		defer clock.UnFreeze()
		invoice, err = s.BillingService.AdvanceInvoice(ctx, invoice.GetInvoiceID())
		require.NoError(t, err)
	}

	require.Equal(t, billing.StandardInvoiceStatusDraftWaitingAutoApproval, invoice.Status)
	require.NotNil(t, invoice.DraftUntil)
	require.Len(t, invoice.Lines.OrEmpty(), 2)
	fixture := chargeBookingInvoice{Customer: cust.GetID(), Invoice: invoice}
	for i, line := range invoice.Lines.OrEmpty() {
		require.NotNil(t, line.ChargeID)
		fixture.Charges[i] = meta.ChargeID{Namespace: ns, ID: *line.ChargeID}
		s.RequireChargeStatus(fixture.Charges[i], processingStatus)
		s.RequireTotals(billingtest.ExpectedTotals{Amount: 5, Total: 5}, line.Totals)
	}
	return fixture
}

// failIssuingAfterFirstBooking arranges a partially committed invoice callback
// without relying on the deleted-line defect, and restores the real engine afterward.
func (s *CreditThenInvoiceTestSuite) failIssuingAfterFirstBooking(fixture chargeBookingInvoice) *failOnceIssuingLineEngine {
	t := s.T()
	t.Helper()
	var engine billing.LineEngine
	switch fixture.Invoice.Lines.OrEmpty()[0].Engine {
	case billing.LineEngineTypeChargeFlatFee:
		engine = s.FlatFeeSvc.GetLineEngine()
	case billing.LineEngineTypeChargeUsageBased:
		engine = s.UsageBasedSvc.GetLineEngine()
	default:
		t.Fatalf("unexpected line engine: %s", fixture.Invoice.Lines.OrEmpty()[0].Engine)
	}
	fault := &failOnceIssuingLineEngine{LineEngine: engine, FailLineID: fixture.Invoice.Lines.OrEmpty()[1].ID}
	require.NoError(t, s.BillingService.DeregisterLineEngine(engine.GetLineEngineType()))
	require.NoError(t, s.BillingService.RegisterLineEngine(fault))
	t.Cleanup(func() {
		require.NoError(t, s.BillingService.DeregisterLineEngine(engine.GetLineEngineType()))
		require.NoError(t, s.BillingService.RegisterLineEngine(engine))
	})
	return fault
}

// failOnceIssuingLineEngine retains real per-line bookings before a transient
// callback failure, isolating retry safety from deleted-line dispatch.
type failOnceIssuingLineEngine struct {
	billing.LineEngine
	FailLineID string
	Failed     bool
}

func (e *failOnceIssuingLineEngine) OnInvoiceIssued(ctx context.Context, input billing.OnInvoiceIssuedInput) error {
	if e.Failed {
		return e.LineEngine.OnInvoiceIssued(ctx, input)
	}

	for _, line := range input.Lines {
		if line.ID == e.FailLineID {
			e.Failed = true
			return billing.ValidationIssue{
				Severity: billing.ValidationIssueSeverityCritical,
				Code:     "test_issuing_callback_failed",
				Message:  "transient invoice-issued callback failure",
			}
		}

		lineInput := input
		lineInput.Lines = billing.StandardLines{line}
		if err := e.LineEngine.OnInvoiceIssued(ctx, lineInput); err != nil {
			return err
		}
	}

	return nil
}
