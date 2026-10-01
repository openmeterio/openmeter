package credits

import (
	"context"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func (s *CreditThenInvoiceTestSuite) TestPaidCallbackClearsPreviousPaymentFailureBeforeBooking() {
	t := s.T()
	ctx := t.Context()
	clock.FreezeTime(time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC))
	defer clock.UnFreeze()

	// given an issued charge whose provider reported a payment failure
	fixture := s.setupPaymentBookingInvoice()
	// Stripe reports collection errors against initiatePayment. Use the billing
	// app boundary because the custom-invoicing convenience method rolls back critical issues.
	require.NoError(t, s.BillingService.TriggerInvoice(ctx, billing.InvoiceTriggerServiceInput{
		InvoiceTriggerInput: billing.InvoiceTriggerInput{
			Invoice: fixture.Invoice.GetInvoiceID(),
			Trigger: billing.TriggerFailed,
			ValidationErrors: &billing.InvoiceTriggerValidationInput{
				Operation: billing.StandardInvoiceOpInitiatePayment,
				Errors: []error{billing.ValidationIssue{
					Severity: billing.ValidationIssueSeverityCritical,
					Code:     "test_payment_declined",
					Message:  "provider reported a declined payment",
				}},
			},
		},
		AppType:    app.AppTypeCustomInvoicing,
		Capability: app.CapabilityTypeCollectPayments,
	}))
	failed, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{Invoice: fixture.Invoice.GetInvoiceID()})
	require.NoError(t, err)
	require.Equal(t, billing.StandardInvoiceStatusPaymentProcessingFailed, failed.Status)
	require.Len(t, failed.ValidationIssues, 1)
	require.Equal(t, "test_payment_declined", failed.ValidationIssues[0].Code)
	require.Equal(t, billing.AppTypeCapabilityToComponent(app.AppTypeCustomInvoicing, app.CapabilityTypeCollectPayments, string(billing.StandardInvoiceOpInitiatePayment)), failed.ValidationIssues[0].Component)
	require.True(t, failed.HasCriticalValidationIssues())
	charge := s.RequireFlatFeeChargeStatus(fixture.Charge, flatfee.StatusActiveAwaitingPaymentSettlement)
	require.NotNil(t, charge.Realizations.CurrentRun)
	require.Nil(t, charge.Realizations.CurrentRun.Payment)

	// when the provider subsequently reports paid through the same service entry point
	require.NoError(t, s.BillingService.TriggerInvoice(ctx, billing.InvoiceTriggerServiceInput{
		InvoiceTriggerInput: billing.InvoiceTriggerInput{Invoice: fixture.Invoice.GetInvoiceID(), Trigger: billing.TriggerPaid},
		AppType:             app.AppTypeCustomInvoicing,
		Capability:          app.CapabilityTypeCollectPayments,
	}))
	invoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{Invoice: fixture.Invoice.GetInvoiceID()})
	require.NoError(t, err)

	// then successful accounting and invoice state agree despite the earlier provider error
	s.requireSettledPaymentBookings(fixture)
	s.Equal(billing.StandardInvoiceStatusPaid, invoice.Status)
	s.False(invoice.StatusDetails.Failed)
	s.False(invoice.HasCriticalValidationIssues(), "issues: %+v", invoice.ValidationIssues)
}

func (s *CreditThenInvoiceTestSuite) TestPaymentBookingRetryPreservesCompletedSettlement() {
	t := s.T()
	ctx := t.Context()
	clock.FreezeTime(time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC))
	defer clock.UnFreeze()

	// given the real settlement completes before a one-time callback failure is reported
	fixture := s.setupPaymentBookingInvoice()
	engine := s.FlatFeeSvc.GetLineEngine()
	fault := &failOnceAfterPaymentSettlementLineEngine{LineEngine: engine}
	require.NoError(t, s.BillingService.DeregisterLineEngine(engine.GetLineEngineType()))
	require.NoError(t, s.BillingService.RegisterLineEngine(fault))
	t.Cleanup(func() {
		require.NoError(t, s.BillingService.DeregisterLineEngine(engine.GetLineEngineType()))
		require.NoError(t, s.BillingService.RegisterLineEngine(engine))
	})
	require.NoError(t, s.BillingService.TriggerInvoice(ctx, billing.InvoiceTriggerServiceInput{
		InvoiceTriggerInput: billing.InvoiceTriggerInput{Invoice: fixture.Invoice.GetInvoiceID(), Trigger: billing.TriggerPaid},
		AppType:             app.AppTypeCustomInvoicing,
		Capability:          app.CapabilityTypeCollectPayments,
	}))
	require.True(t, fault.Failed)
	failed, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{Invoice: fixture.Invoice.GetInvoiceID()})
	require.NoError(t, err)
	require.Equal(t, billing.StandardInvoiceStatusPaymentProcessingBookingAuthorizedAndSettledFailed, failed.Status)
	require.True(t, failed.HasCriticalValidationIssues())
	require.Len(t, failed.ValidationIssues, 1)
	require.Equal(t, "test_post_settlement_callback_failed", failed.ValidationIssues[0].Code)
	booked := s.requireSettledPaymentBookings(fixture)
	before, err := s.Ledger.ListTransactions(ctx, ledger.ListTransactionsInput{Namespace: fixture.Customer.Namespace, Limit: 100})
	require.NoError(t, err)
	require.Nil(t, before.NextCursor)
	require.Len(t, before.Items, 3)
	beforeIDs := lo.Map(before.Items, func(tx ledger.Transaction, _ int) string { return tx.ID().ID })

	// when retry resumes from the persisted invoice with its completed payment intact
	invoice, err := s.BillingService.RetryInvoice(ctx, failed.GetInvoiceID())
	require.NoError(t, err)

	// then retry reaches paid without replacing references or adding ledger transactions
	s.Equal(billing.StandardInvoiceStatusPaid, invoice.Status, "issues: %+v", invoice.ValidationIssues)
	s.False(invoice.StatusDetails.Failed)
	s.False(invoice.HasCriticalValidationIssues())
	s.Equal(booked, s.requireSettledPaymentBookings(fixture))
	after, err := s.Ledger.ListTransactions(ctx, ledger.ListTransactionsInput{Namespace: fixture.Customer.Namespace, Limit: 100})
	require.NoError(t, err)
	require.Nil(t, after.NextCursor)
	s.ElementsMatch(beforeIDs, lo.Map(after.Items, func(tx ledger.Transaction, _ int) string { return tx.ID().ID }))
}

type paymentBookingInvoice struct {
	Customer customer.CustomerID
	Charge   meta.ChargeID
	Invoice  billing.StandardInvoice
}

func (s *CreditThenInvoiceTestSuite) setupPaymentBookingInvoice() paymentBookingInvoice {
	t := s.T()
	t.Helper()
	ctx := t.Context()
	ns := s.GetUniqueNamespace("payment-booking")
	s.ProvisionDefaultTaxCodes(ctx, ns)
	invoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")
	s.ProvisionBillingProfile(ctx, ns, invoicing.App.GetID())
	created, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.NewCreateChargeIntents(s.CreateMockChargeIntent(CreateMockChargeIntentInput{
			Customer:          cust.GetID(),
			Currency:          USD,
			ServicePeriod:     timeutil.ClosedPeriod{From: clock.Now(), To: clock.Now().AddDate(0, 1, 0)},
			SettlementMode:    productcatalog.CreditThenInvoiceSettlementMode,
			Price:             productcatalog.NewPriceFrom(productcatalog.FlatPrice{Amount: alpacadecimal.NewFromInt(21), PaymentTerm: productcatalog.InAdvancePaymentTerm}),
			Name:              "payment booking",
			ManagedBy:         billing.SubscriptionManagedLine,
			UniqueReferenceID: "payment-booking",
		})),
	})
	require.NoError(t, err)
	require.Len(t, created, 1)
	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{Customer: cust.GetID(), AsOf: lo.ToPtr(clock.Now())})
	require.NoError(t, err)
	require.Len(t, invoices, 1)
	require.Equal(t, billing.StandardInvoiceStatusDraftWaitingAutoApproval, invoices[0].Status)
	invoice, err := s.BillingService.ApproveInvoice(ctx, invoices[0].GetInvoiceID())
	require.NoError(t, err)
	require.Equal(t, billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)
	require.Len(t, invoice.Lines.OrEmpty(), 1)
	require.Equal(t, float64(21), invoice.Totals.Total.InexactFloat64())
	require.NotNil(t, invoice.Lines.OrEmpty()[0].ChargeID)
	return paymentBookingInvoice{
		Customer: cust.GetID(),
		Charge:   meta.ChargeID{Namespace: ns, ID: *invoice.Lines.OrEmpty()[0].ChargeID},
		Invoice:  invoice,
	}
}

// requireSettledPaymentBookings follows persisted run references to the real journal;
// balances alone cannot detect missing references or balanced duplicate bookings.
func (s *CreditThenInvoiceTestSuite) requireSettledPaymentBookings(fixture paymentBookingInvoice) payment.Invoiced {
	t := s.T()
	t.Helper()
	ctx := t.Context()
	charge := s.RequireFlatFeeChargeStatus(fixture.Charge, flatfee.StatusFinal)
	run := charge.Realizations.CurrentRun
	require.NotNil(t, run)
	require.True(t, run.Immutable)
	require.Nil(t, run.DeletedAt)
	require.NotNil(t, run.AccruedUsage)
	require.NotNil(t, run.AccruedUsage.LedgerTransaction)
	require.NotNil(t, run.Payment)
	booked := run.Payment
	require.Nil(t, booked.DeletedAt)
	require.Equal(t, payment.StatusSettled, booked.Status)
	require.Equal(t, fixture.Invoice.ID, booked.InvoiceID)
	require.Equal(t, fixture.Invoice.Lines.OrEmpty()[0].ID, booked.LineID)
	require.Equal(t, run.ServicePeriod, booked.ServicePeriod)
	require.Equal(t, float64(21), booked.FiatAmount.InexactFloat64())
	require.NotNil(t, booked.Authorized)
	require.NotNil(t, booked.Settled)
	groupIDs := []string{run.AccruedUsage.LedgerTransaction.TransactionGroupID, booked.Authorized.TransactionGroupID, booked.Settled.TransactionGroupID}
	expectedPostings := []map[string]float64{
		{"customer_accrued": 21, "customer_receivable/open": -21},
		{"customer_receivable/open": 21, "customer_receivable/authorized": -21},
		{"customer_receivable/authorized": 21, "wash": -21},
	}
	require.Len(t, lo.Uniq(groupIDs), 3)
	for i, groupID := range groupIDs {
		require.NotEmpty(t, groupID)
		id := models.NamespacedID{Namespace: fixture.Customer.Namespace, ID: groupID}
		group, err := s.Ledger.GetTransactionGroup(ctx, id)
		require.NoError(t, err)
		require.Equal(t, id, group.ID())
		require.Len(t, group.Transactions(), 1)
		tx := group.Transactions()[0]
		require.Equal(t, id, tx.GroupID())
		if i == 1 {
			require.True(t, booked.Authorized.Time.Equal(tx.BookedAt()))
		}

		if i == 2 {
			require.True(t, booked.Settled.Time.Equal(tx.BookedAt()))
		}

		require.Len(t, tx.Entries(), 2)
		sum := alpacadecimal.Zero
		postings := make(map[string]float64)
		for _, entry := range tx.Entries() {
			require.Equal(t, tx.ID(), entry.TransactionID())
			require.NotNil(t, entry.Provenance().SpendChargeID)
			require.Equal(t, fixture.Charge.ID, *entry.Provenance().SpendChargeID)
			address := entry.PostingAddress()
			route := address.Route().Route()
			require.Equal(t, USD, route.Currency.GetCode())
			key := string(address.AccountType())
			if route.TransactionAuthorizationStatus != nil {
				key += "/" + string(*route.TransactionAuthorizationStatus)
			}

			postings[key] += entry.Amount().InexactFloat64()
			sum = sum.Add(entry.Amount())
		}
		require.Equal(t, float64(0), sum.InexactFloat64())
		require.Equal(t, expectedPostings[i], postings)
	}
	s.AssertLedgerSnapshotEqual(LedgerSnapshot{
		Accrued: alpacadecimal.NewFromInt(21),
		Wash:    alpacadecimal.NewFromInt(-21),
	}, s.CreateLedgerSnapshot(LedgerSnapshotInput{
		Namespace: fixture.Customer.Namespace,
		Customer:  fixture.Customer,
		Currency:  USD,
		CostBasis: mo.None[*alpacadecimal.Decimal](),
	}))
	return *booked
}

// The fault is reported after the delegated settlement so billing sees an error
// while the real charge and journal have completed; no persisted state is injected.
type failOnceAfterPaymentSettlementLineEngine struct {
	billing.LineEngine
	Failed bool
}

func (e *failOnceAfterPaymentSettlementLineEngine) OnPaymentSettled(ctx context.Context, input billing.OnPaymentSettledInput) error {
	if err := e.LineEngine.OnPaymentSettled(ctx, input); err != nil {
		return err
	}

	if e.Failed {
		return nil
	}

	e.Failed = true
	return billing.ValidationIssue{
		Severity: billing.ValidationIssueSeverityCritical,
		Code:     "test_post_settlement_callback_failed",
		Message:  "transient callback failure after settlement completed",
	}
}
