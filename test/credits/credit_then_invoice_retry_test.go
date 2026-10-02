package credits

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

func (s *CreditThenInvoiceTestSuite) TestFlatFeePaymentAuthorizationRetryPreservesCompletedLineBooking() {
	clock.FreezeTime(time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC))
	defer clock.UnFreeze()

	ns := s.GetUniqueNamespace("flat-fee-payment-authorization-retry")
	var (
		cust                                *customer.Customer
		invoice, failed, retried            billing.StandardInvoice
		lines                               billing.StandardLines
		firstBeforeRetry, secondBeforeRetry flatfee.RealizationRun
		beforeTransactionIDs                []string
	)

	s.Run("given $5 and $10 flat-fee charges are invoiced", func() {
		t := s.T()
		ctx := t.Context()

		s.ProvisionDefaultTaxCodes(ctx, ns)
		invoicing := s.SetupCustomInvoicing(ns)
		cust = s.CreateLedgerBackedCustomer(ns, "test-subject")
		s.ProvisionBillingProfile(ctx, ns, invoicing.App.GetID())
		period := timeutil.ClosedPeriod{From: clock.Now(), To: clock.Now().AddDate(0, 1, 0)}
		intents := make([]charges.ChargeIntent, 0, 2)
		for i, amount := range []int64{5, 10} {
			intents = append(intents, s.CreateMockChargeIntent(CreateMockChargeIntentInput{
				Customer:       cust.GetID(),
				Currency:       USD,
				ServicePeriod:  period,
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromInt(amount),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				ProRating:         productcatalog.ProRatingConfig{Enabled: true, Mode: productcatalog.ProRatingModeProratePrices},
				Name:              fmt.Sprintf("charge %d", i),
				ManagedBy:         billing.SubscriptionManagedLine,
				UniqueReferenceID: fmt.Sprintf("charge-%d", i),
			}))
		}
		created, err := s.Charges.Create(ctx, charges.CreateInput{Namespace: ns, Intents: charges.NewCreateChargeIntents(intents...)})
		require.NoError(t, err)
		require.Len(t, created, 2)
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(), AsOf: lo.ToPtr(clock.Now()),
		})
		require.NoError(t, err)
		require.Len(t, invoices, 1)
		invoice = invoices[0]
	})

	s.Run("given the invoice is approved for payment", func() {
		t := s.T()
		ctx := t.Context()
		var err error

		invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
		require.NoError(t, err)
		require.Equal(t, billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)
		require.Equal(t, float64(15), invoice.Totals.Total.InexactFloat64())
		lines = invoice.Lines.OrEmpty()
		require.Len(t, lines, 2)
		require.ElementsMatch(t, []float64{5, 10}, []float64{
			lines[0].Totals.Total.InexactFloat64(),
			lines[1].Totals.Total.InexactFloat64(),
		})
	})

	s.Run("given authorization fails after the first line commits", func() {
		t := s.T()

		failed = s.authorizeInvoiceWithSecondLineFailure(invoice, s.FlatFeeSvc.GetLineEngine())
		require.Equal(t, billing.StandardInvoiceStatusPaymentProcessingBookingAuthorizedFailed, failed.Status)
		require.True(t, failed.HasCriticalValidationIssues())
		firstBeforeRetry = s.requireFlatFeeInvoiceRun(lines[0])
		secondBeforeRetry = s.requireFlatFeeInvoiceRun(lines[1])
		s.requireInvoiceAuthorizationBooking(lines[0], firstBeforeRetry.Payment)
		require.Nil(t, secondBeforeRetry.Payment)
		beforeTransactionIDs = s.paymentAuthorizationTransactionIDs(ns)
		require.Len(t, beforeTransactionIDs, 3)
	})

	s.Run("when invoice authorization is retried", func() {
		t := s.T()
		ctx := t.Context()
		var err error

		retried, err = s.BillingService.RetryInvoice(ctx, failed.GetInvoiceID())
		require.NoError(t, err)
	})

	s.Run("then the first booking is preserved and both lines are authorized", func() {
		t := s.T()

		require.Equal(t, billing.StandardInvoiceStatusPaymentProcessingAuthorized, retried.Status, "issues: %+v", retried.ValidationIssues)
		require.False(t, retried.HasCriticalValidationIssues())
		firstAfterRetry := s.requireFlatFeeInvoiceRun(lines[0])
		secondAfterRetry := s.requireFlatFeeInvoiceRun(lines[1])
		require.Equal(t, firstBeforeRetry, firstAfterRetry)
		require.Equal(t, secondBeforeRetry.ID, secondAfterRetry.ID)
		require.Equal(t, secondBeforeRetry.AccruedUsage, secondAfterRetry.AccruedUsage)
		s.requireInvoiceAuthorizationBooking(lines[0], firstAfterRetry.Payment)
		s.requireInvoiceAuthorizationBooking(lines[1], secondAfterRetry.Payment)
	})

	s.Run("then only the missing authorization is added to the ledger", func() {
		t := s.T()

		afterTransactionIDs := s.paymentAuthorizationTransactionIDs(ns)
		require.Subset(t, afterTransactionIDs, beforeTransactionIDs)
		require.Len(t, afterTransactionIDs, 4)
		s.AssertLedgerSnapshotEqual(LedgerSnapshot{
			Accrued:              alpacadecimal.NewFromInt(15),
			AuthorizedReceivable: alpacadecimal.NewFromInt(-15),
		}, s.CreateLedgerSnapshot(LedgerSnapshotInput{
			Namespace: ns,
			Customer:  cust.GetID(),
			Currency:  USD,
			CostBasis: mo.None[*alpacadecimal.Decimal](),
		}))
	})
}

func (s *CreditThenInvoiceTestSuite) TestUsageBasedPaymentAuthorizationRetryPreservesCompletedLineBooking() {
	clock.FreezeTime(time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC))
	defer clock.UnFreeze()
	defer s.MockStreamingConnector.Reset()

	ns := s.GetUniqueNamespace("usage-based-payment-authorization-retry")
	var (
		cust                                *customer.Customer
		invoice, failed, retried            billing.StandardInvoice
		lines                               billing.StandardLines
		firstBeforeRetry, secondBeforeRetry usagebased.RealizationRun
		beforeTransactionIDs                []string
	)

	s.Run("given $5 and $10 usage-based charges are invoiced", func() {
		t := s.T()
		ctx := t.Context()

		s.ProvisionDefaultTaxCodes(ctx, ns)
		invoicing := s.SetupCustomInvoicing(ns)
		cust = s.CreateLedgerBackedCustomer(ns, "test-subject")
		s.ProvisionBillingProfile(ctx, ns, invoicing.App.GetID())
		feature := s.SetupApiRequestsTotalFeature(ctx, ns)
		period := timeutil.ClosedPeriod{From: clock.Now().AddDate(0, -1, 0), To: clock.Now()}
		s.MockStreamingConnector.AddSimpleEvent(feature.Feature.Key, 5, period.From.Add(15*24*time.Hour))
		intents := make([]charges.ChargeIntent, 0, 2)
		for i, unitPrice := range []int64{1, 2} {
			intents = append(intents, s.CreateMockChargeIntent(CreateMockChargeIntentInput{
				Customer:          cust.GetID(),
				Currency:          USD,
				ServicePeriod:     period,
				SettlementMode:    productcatalog.CreditThenInvoiceSettlementMode,
				Price:             productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(unitPrice)}),
				ProRating:         productcatalog.ProRatingConfig{Enabled: true, Mode: productcatalog.ProRatingModeProratePrices},
				FeatureKey:        feature.Feature.Key,
				Name:              fmt.Sprintf("charge %d", i),
				ManagedBy:         billing.SubscriptionManagedLine,
				UniqueReferenceID: fmt.Sprintf("charge-%d", i),
			}))
		}
		created, err := s.Charges.Create(ctx, charges.CreateInput{Namespace: ns, Intents: charges.NewCreateChargeIntents(intents...)})
		require.NoError(t, err)
		require.Len(t, created, 2)
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(), AsOf: lo.ToPtr(clock.Now()),
		})
		require.NoError(t, err)
		require.Len(t, invoices, 1)
		invoice = invoices[0]
	})

	clock.FreezeTime(invoice.DefaultCollectionAtForStandardInvoice())
	defer clock.UnFreeze()

	s.Run("given the invoice is approved for payment", func() {
		t := s.T()
		ctx := t.Context()
		var err error

		invoice, err = s.BillingService.AdvanceInvoice(ctx, invoice.GetInvoiceID())
		require.NoError(t, err)
		invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
		require.NoError(t, err)
		require.Equal(t, billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)
		require.Equal(t, float64(15), invoice.Totals.Total.InexactFloat64())
		lines = invoice.Lines.OrEmpty()
		require.Len(t, lines, 2)
		require.ElementsMatch(t, []float64{5, 10}, []float64{
			lines[0].Totals.Total.InexactFloat64(),
			lines[1].Totals.Total.InexactFloat64(),
		})
	})

	s.Run("given authorization fails after the first line commits", func() {
		t := s.T()

		failed = s.authorizeInvoiceWithSecondLineFailure(invoice, s.UsageBasedSvc.GetLineEngine())
		require.Equal(t, billing.StandardInvoiceStatusPaymentProcessingBookingAuthorizedFailed, failed.Status)
		require.True(t, failed.HasCriticalValidationIssues())
		firstBeforeRetry = s.requireUsageBasedInvoiceRun(lines[0])
		secondBeforeRetry = s.requireUsageBasedInvoiceRun(lines[1])
		s.requireInvoiceAuthorizationBooking(lines[0], firstBeforeRetry.Payment)
		require.Nil(t, secondBeforeRetry.Payment)
		beforeTransactionIDs = s.paymentAuthorizationTransactionIDs(ns)
		require.Len(t, beforeTransactionIDs, 3)
	})

	s.Run("when invoice authorization is retried", func() {
		t := s.T()
		ctx := t.Context()
		var err error

		retried, err = s.BillingService.RetryInvoice(ctx, failed.GetInvoiceID())
		require.NoError(t, err)
	})

	s.Run("then the first booking is preserved and both lines are authorized", func() {
		t := s.T()

		require.Equal(t, billing.StandardInvoiceStatusPaymentProcessingAuthorized, retried.Status, "issues: %+v", retried.ValidationIssues)
		require.False(t, retried.HasCriticalValidationIssues())
		firstAfterRetry := s.requireUsageBasedInvoiceRun(lines[0])
		secondAfterRetry := s.requireUsageBasedInvoiceRun(lines[1])
		require.Equal(t, firstBeforeRetry, firstAfterRetry)
		require.Equal(t, secondBeforeRetry.ID, secondAfterRetry.ID)
		require.Equal(t, secondBeforeRetry.InvoiceUsage, secondAfterRetry.InvoiceUsage)
		s.requireInvoiceAuthorizationBooking(lines[0], firstAfterRetry.Payment)
		s.requireInvoiceAuthorizationBooking(lines[1], secondAfterRetry.Payment)
	})

	s.Run("then only the missing authorization is added to the ledger", func() {
		t := s.T()

		afterTransactionIDs := s.paymentAuthorizationTransactionIDs(ns)
		require.Subset(t, afterTransactionIDs, beforeTransactionIDs)
		require.Len(t, afterTransactionIDs, 4)
		s.AssertLedgerSnapshotEqual(LedgerSnapshot{
			Accrued:              alpacadecimal.NewFromInt(15),
			AuthorizedReceivable: alpacadecimal.NewFromInt(-15),
		}, s.CreateLedgerSnapshot(LedgerSnapshotInput{
			Namespace: ns,
			Customer:  cust.GetID(),
			Currency:  USD,
			CostBasis: mo.None[*alpacadecimal.Decimal](),
		}))
	})
}

func (s *CreditThenInvoiceTestSuite) TestCreditPurchasePaymentAuthorizationRetryPreservesCompletedLineBooking() {
	clock.FreezeTime(time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC))
	defer clock.UnFreeze()

	ns := s.GetUniqueNamespace("credit-purchase-payment-authorization-retry")
	var (
		cust                                *customer.Customer
		invoice, failed, retried            billing.StandardInvoice
		lines                               billing.StandardLines
		firstBeforeRetry, secondBeforeRetry creditpurchase.Charge
		beforeTransactionIDs                []string
	)

	s.Run("given $5 and $10 credit-purchase charges are invoiced", func() {
		t := s.T()
		ctx := t.Context()

		defaults := s.ProvisionDefaultTaxCodes(ctx, ns)
		invoicing := s.SetupCustomInvoicing(ns)
		cust = s.CreateLedgerBackedCustomer(ns, "test-subject")
		s.ProvisionBillingProfile(ctx, ns, invoicing.App.GetID())
		period := timeutil.ClosedPeriod{From: clock.Now(), To: clock.Now().AddDate(0, 1, 0)}
		for _, amount := range []int64{5, 10} {
			intent, err := s.CreateCreditPurchaseIntent(CreateCreditPurchaseIntentInput{
				Customer:      cust.GetID(),
				Currency:      USD,
				Amount:        alpacadecimal.NewFromInt(amount),
				ServicePeriod: period,
				Settlement:    creditpurchase.NewInvoiceSettlement(),
				CostBasis:     newFiatCreditPurchaseCostBasis(alpacadecimal.NewFromInt(1)),
				TaxConfig:     productcatalog.TaxCodeConfig{TaxCodeID: defaults.InvoicingTaxCodeID},
			}).AsCreditPurchaseIntent()
			require.NoError(t, err)
			created, err := s.CreditPurchaseSvc.Create(ctx, creditpurchase.CreateInput{Namespace: ns, Intent: intent})
			require.NoError(t, err)
			require.NotNil(t, created.GatheringLineToCreate)
			_, err = s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
				Customer: cust.GetID(),
				Currency: created.GatheringLineToCreate.Currency,
				Lines:    billing.CreatePendingInvoiceLines{{GatheringLine: *created.GatheringLineToCreate}},
			})
			require.NoError(t, err)
		}
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(), AsOf: lo.ToPtr(clock.Now()),
		})
		require.NoError(t, err)
		require.Len(t, invoices, 1)
		invoice = invoices[0]
	})

	s.Run("given the invoice is approved for payment", func() {
		t := s.T()
		ctx := t.Context()
		var err error

		invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
		require.NoError(t, err)
		require.Equal(t, billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)
		require.Equal(t, float64(15), invoice.Totals.Total.InexactFloat64())
		lines = invoice.Lines.OrEmpty()
		require.Len(t, lines, 2)
		require.ElementsMatch(t, []float64{5, 10}, []float64{
			lines[0].Totals.Total.InexactFloat64(),
			lines[1].Totals.Total.InexactFloat64(),
		})
	})

	s.Run("given authorization fails after the first line commits", func() {
		t := s.T()

		failed = s.authorizeInvoiceWithSecondLineFailure(invoice, s.CreditPurchaseSvc.GetLineEngine())
		require.Equal(t, billing.StandardInvoiceStatusPaymentProcessingBookingAuthorizedFailed, failed.Status)
		require.True(t, failed.HasCriticalValidationIssues())
		firstBeforeRetry = s.requireInvoiceFundedCreditPurchase(lines[0])
		secondBeforeRetry = s.requireInvoiceFundedCreditPurchase(lines[1])
		s.requireInvoiceAuthorizationBooking(lines[0], firstBeforeRetry.Realizations.InvoiceSettlement)
		require.Nil(t, secondBeforeRetry.Realizations.InvoiceSettlement)
		beforeTransactionIDs = s.paymentAuthorizationTransactionIDs(ns)
		require.Len(t, beforeTransactionIDs, 3)
	})

	s.Run("when invoice authorization is retried", func() {
		t := s.T()
		ctx := t.Context()
		var err error

		retried, err = s.BillingService.RetryInvoice(ctx, failed.GetInvoiceID())
		require.NoError(t, err)
	})

	s.Run("then the first booking is preserved and both lines are authorized", func() {
		t := s.T()

		require.Equal(t, billing.StandardInvoiceStatusPaymentProcessingAuthorized, retried.Status, "issues: %+v", retried.ValidationIssues)
		require.False(t, retried.HasCriticalValidationIssues())
		firstAfterRetry := s.requireInvoiceFundedCreditPurchase(lines[0])
		secondAfterRetry := s.requireInvoiceFundedCreditPurchase(lines[1])
		require.Equal(t, firstBeforeRetry.Realizations, firstAfterRetry.Realizations)
		require.Equal(t, secondBeforeRetry.Realizations.CreditGrantRealization, secondAfterRetry.Realizations.CreditGrantRealization)
		require.Equal(t, creditpurchase.StatusActivePaymentAuthorized, firstAfterRetry.Status)
		require.Equal(t, creditpurchase.StatusActivePaymentAuthorized, secondAfterRetry.Status)
		s.requireInvoiceAuthorizationBooking(lines[0], firstAfterRetry.Realizations.InvoiceSettlement)
		s.requireInvoiceAuthorizationBooking(lines[1], secondAfterRetry.Realizations.InvoiceSettlement)
	})

	s.Run("then only the missing authorization is added to the ledger", func() {
		t := s.T()

		afterTransactionIDs := s.paymentAuthorizationTransactionIDs(ns)
		require.Subset(t, afterTransactionIDs, beforeTransactionIDs)
		require.Len(t, afterTransactionIDs, 4)
		s.AssertLedgerSnapshotEqual(LedgerSnapshot{
			FBO:                  alpacadecimal.NewFromInt(15),
			AuthorizedReceivable: alpacadecimal.NewFromInt(-15),
		}, s.CreateLedgerSnapshot(LedgerSnapshotInput{
			Namespace: ns,
			Customer:  cust.GetID(),
			Currency:  USD,
			CostBasis: mo.None[*alpacadecimal.Decimal](),
		}))
	})
}

func (s *CreditThenInvoiceTestSuite) authorizeInvoiceWithSecondLineFailure(invoice billing.StandardInvoice, engine billing.LineEngine) billing.StandardInvoice {
	t := s.T()
	t.Helper()
	ctx := t.Context()
	fault := &failOncePaymentAuthorizationLineEngine{LineEngine: engine, FailLineID: invoice.Lines.OrEmpty()[1].ID}
	require.NoError(t, s.BillingService.DeregisterLineEngine(engine.GetLineEngineType()))
	require.NoError(t, s.BillingService.RegisterLineEngine(fault))
	restoration := paymentAuthorizationLineEngineRestoration{T: t, Billing: s.BillingService, Engine: engine}
	t.Cleanup(restoration.restore)

	// The app boundary persists validation failures with prior line progress;
	// the custom-invoicing convenience method would roll the transaction back.
	require.NoError(t, s.BillingService.TriggerInvoice(ctx, billing.InvoiceTriggerServiceInput{
		InvoiceTriggerInput: billing.InvoiceTriggerInput{Invoice: invoice.GetInvoiceID(), Trigger: billing.TriggerAuthorized},
		AppType:             app.AppTypeCustomInvoicing,
		Capability:          app.CapabilityTypeCollectPayments,
	}))
	require.True(t, fault.Failed)
	failed, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{Invoice: invoice.GetInvoiceID()})
	require.NoError(t, err)
	return failed
}

type paymentAuthorizationLineEngineRestoration struct {
	T       *testing.T
	Billing billing.Service
	Engine  billing.LineEngine
}

func (r paymentAuthorizationLineEngineRestoration) restore() {
	require.NoError(r.T, r.Billing.DeregisterLineEngine(r.Engine.GetLineEngineType()))
	require.NoError(r.T, r.Billing.RegisterLineEngine(r.Engine))
}

func (s *CreditThenInvoiceTestSuite) requireFlatFeeInvoiceRun(line *billing.StandardLine) flatfee.RealizationRun {
	t := s.T()
	t.Helper()
	require.NotNil(t, line.ChargeID)
	charge := s.RequireFlatFeeChargeStatus(meta.ChargeID{Namespace: line.Namespace, ID: *line.ChargeID}, flatfee.StatusActiveAwaitingPaymentSettlement)
	run, err := charge.Realizations.GetByLineID(line.ID)
	require.NoError(t, err)
	require.True(t, run.Immutable)
	require.Nil(t, run.DeletedAt)
	require.NotNil(t, run.AccruedUsage)
	require.NotNil(t, run.AccruedUsage.LedgerTransaction)
	return run
}

func (s *CreditThenInvoiceTestSuite) requireUsageBasedInvoiceRun(line *billing.StandardLine) usagebased.RealizationRun {
	t := s.T()
	t.Helper()
	require.NotNil(t, line.ChargeID)
	charge := s.RequireUsageBasedChargeStatus(meta.ChargeID{Namespace: line.Namespace, ID: *line.ChargeID}, usagebased.StatusActiveAwaitingPaymentSettlement)
	run, err := charge.Realizations.GetByLineID(line.ID)
	require.NoError(t, err)
	require.Nil(t, run.DeletedAt)
	require.NotNil(t, run.InvoiceUsage)
	require.NotNil(t, run.InvoiceUsage.LedgerTransaction)
	return run
}

func (s *CreditThenInvoiceTestSuite) requireInvoiceFundedCreditPurchase(line *billing.StandardLine) creditpurchase.Charge {
	t := s.T()
	t.Helper()
	require.NotNil(t, line.ChargeID)
	charge, err := s.MustGetChargeByID(meta.ChargeID{Namespace: line.Namespace, ID: *line.ChargeID}).AsCreditPurchaseCharge()
	require.NoError(t, err)
	require.NotNil(t, charge.Realizations.CreditGrantRealization)
	return charge
}

func (s *CreditThenInvoiceTestSuite) requireInvoiceAuthorizationBooking(line *billing.StandardLine, booked *payment.Invoiced) {
	t := s.T()
	t.Helper()
	require.NotNil(t, booked)
	require.Nil(t, booked.DeletedAt)
	require.Equal(t, line.InvoiceID, booked.InvoiceID)
	require.Equal(t, line.ID, booked.LineID)
	require.Equal(t, line.Period, booked.ServicePeriod)
	require.Equal(t, line.Totals.Total.InexactFloat64(), booked.FiatAmount.InexactFloat64())
	require.Equal(t, payment.StatusAuthorized, booked.Status)
	require.NotNil(t, booked.Authorized)
	require.Nil(t, booked.Settled)
	groupID := models.NamespacedID{Namespace: line.Namespace, ID: booked.Authorized.TransactionGroupID}
	group, err := s.Ledger.GetTransactionGroup(t.Context(), groupID)
	require.NoError(t, err)
	require.Equal(t, groupID, group.ID())
	require.Len(t, group.Transactions(), 1)
	tx := group.Transactions()[0]
	require.Equal(t, groupID, tx.GroupID())
	require.True(t, booked.Authorized.Time.Equal(tx.BookedAt()))
	require.Len(t, tx.Entries(), 2)
	postings := make(map[string]float64)
	for _, entry := range tx.Entries() {
		if line.Engine == billing.LineEngineTypeChargeCreditPurchase {
			require.Equal(t, line.ChargeID, entry.Provenance().SourceChargeID)
		} else {
			require.Equal(t, line.ChargeID, entry.Provenance().SpendChargeID)
		}
		address := entry.PostingAddress()
		route := address.Route().Route()
		require.Equal(t, USD, route.Currency.GetCode())
		require.NotNil(t, route.TransactionAuthorizationStatus)
		key := string(address.AccountType()) + "/" + string(*route.TransactionAuthorizationStatus)
		postings[key] += entry.Amount().InexactFloat64()
	}
	amount := line.Totals.Total.InexactFloat64()
	require.Equal(t, map[string]float64{
		"customer_receivable/open":       amount,
		"customer_receivable/authorized": -amount,
	}, postings)
}

func (s *CreditThenInvoiceTestSuite) paymentAuthorizationTransactionIDs(namespace string) []string {
	t := s.T()
	t.Helper()
	transactions, err := s.Ledger.ListTransactions(t.Context(), ledger.ListTransactionsInput{Namespace: namespace, Limit: 100})
	require.NoError(t, err)
	require.Nil(t, transactions.NextCursor)
	ids := make([]string, 0, len(transactions.Items))
	for _, transaction := range transactions.Items {
		ids = append(ids, transaction.ID().ID)
	}
	return ids
}

type failOncePaymentAuthorizationLineEngine struct {
	billing.LineEngine
	FailLineID string
	Failed     bool
}

func (e *failOncePaymentAuthorizationLineEngine) OnPaymentAuthorized(ctx context.Context, input billing.OnPaymentAuthorizedInput) error {
	if e.Failed {
		return e.LineEngine.OnPaymentAuthorized(ctx, input)
	}

	for _, line := range input.Lines {
		if line.ID == e.FailLineID {
			e.Failed = true
			return billing.ValidationIssue{
				Severity: billing.ValidationIssueSeverityCritical,
				Code:     "test_transient_line_callback_failed",
				Message:  "transient line callback failure after prior progress committed",
			}
		}

		one := input
		one.Lines = billing.StandardLines{line}
		if err := e.LineEngine.OnPaymentAuthorized(ctx, one); err != nil {
			return err
		}
	}

	return nil
}

func (s *CreditThenInvoiceTestSuite) TestFlatFeeIssuingRetryPreservesCompletedChargeBooking() {
	ctx := s.T().Context()
	clock.FreezeTime(time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC))
	defer clock.UnFreeze()

	var fixture issuingRetryInvoice
	s.Run("create an invoice with two flat fee charges", func() {
		t := s.T()
		fixture = s.setupFlatFeeIssuingRetryInvoice()
		require.Equal(t, float64(15), fixture.Invoice.Totals.Total.InexactFloat64())
	})

	var completedBeforeRetry completedIssuingBooking
	var ledgerBeforeRetry LedgerSnapshot
	s.Run("fail issuance after the first charge booking completes", func() {
		t := s.T()
		clock.FreezeTime(fixture.Invoice.DraftUntil.Add(time.Second))
		defer clock.UnFreeze()

		fault := s.failIssuingAfterFirstBooking(fixture.Invoice, s.FlatFeeSvc.GetLineEngine())
		failedInvoice, err := s.BillingService.AdvanceInvoice(ctx, fixture.Invoice.GetInvoiceID())
		require.NoError(t, err)
		require.True(t, fault.Failed)
		require.Equal(t, billing.StandardInvoiceStatusIssuingChargeBookingFailed, failedInvoice.Status)
		require.True(t, failedInvoice.HasCriticalValidationIssues())

		completedBeforeRetry = s.requireCompletedFlatFeeIssuingBooking(fixture)
		ledgerBeforeRetry = s.CreateLedgerSnapshot(fixture.ledgerSnapshotInput())
		require.Equal(t, float64(5), ledgerBeforeRetry.Accrued.InexactFloat64())
	})

	s.Run("retry books only the remaining charge", func() {
		t := s.T()
		retried, err := s.BillingService.RetryInvoice(ctx, fixture.Invoice.GetInvoiceID())
		require.NoError(t, err)
		require.Equal(t, billing.StandardInvoiceStatusPaymentProcessingPending, retried.Status, "issues: %+v", retried.ValidationIssues)
		require.False(t, retried.HasCriticalValidationIssues())
		require.Equal(t, completedBeforeRetry, s.requireCompletedFlatFeeIssuingBooking(fixture))
		s.AssertLedgerSnapshotEqual(LedgerSnapshot{
			FBO:                  ledgerBeforeRetry.FBO,
			Accrued:              alpacadecimal.NewFromInt(15),
			OpenReceivable:       alpacadecimal.NewFromInt(-15),
			AuthorizedReceivable: ledgerBeforeRetry.AuthorizedReceivable,
			Wash:                 ledgerBeforeRetry.Wash,
			Earnings:             ledgerBeforeRetry.Earnings,
		}, s.CreateLedgerSnapshot(fixture.ledgerSnapshotInput()))
	})
}

func (s *CreditThenInvoiceTestSuite) TestUsageBasedIssuingRetryPreservesCompletedChargeBooking() {
	ctx := s.T().Context()
	clock.FreezeTime(time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC))
	defer clock.UnFreeze()
	defer s.MockStreamingConnector.Reset()

	var fixture issuingRetryInvoice
	s.Run("create an invoice with two usage based charges", func() {
		t := s.T()
		fixture = s.setupUsageBasedIssuingRetryInvoice()
		require.Equal(t, float64(15), fixture.Invoice.Totals.Total.InexactFloat64())
	})

	var completedBeforeRetry completedIssuingBooking
	var ledgerBeforeRetry LedgerSnapshot
	s.Run("fail issuance after the first charge booking completes", func() {
		t := s.T()
		clock.FreezeTime(fixture.Invoice.DraftUntil.Add(time.Second))
		defer clock.UnFreeze()

		fault := s.failIssuingAfterFirstBooking(fixture.Invoice, s.UsageBasedSvc.GetLineEngine())
		failedInvoice, err := s.BillingService.AdvanceInvoice(ctx, fixture.Invoice.GetInvoiceID())
		require.NoError(t, err)
		require.True(t, fault.Failed)
		require.Equal(t, billing.StandardInvoiceStatusIssuingChargeBookingFailed, failedInvoice.Status)
		require.True(t, failedInvoice.HasCriticalValidationIssues())

		completedBeforeRetry = s.requireCompletedUsageBasedIssuingBooking(fixture)
		ledgerBeforeRetry = s.CreateLedgerSnapshot(fixture.ledgerSnapshotInput())
		require.Equal(t, float64(5), ledgerBeforeRetry.Accrued.InexactFloat64())
	})

	s.Run("retry books only the remaining charge", func() {
		t := s.T()
		retried, err := s.BillingService.RetryInvoice(ctx, fixture.Invoice.GetInvoiceID())
		require.NoError(t, err)
		require.Equal(t, billing.StandardInvoiceStatusPaymentProcessingPending, retried.Status, "issues: %+v", retried.ValidationIssues)
		require.False(t, retried.HasCriticalValidationIssues())
		require.Equal(t, completedBeforeRetry, s.requireCompletedUsageBasedIssuingBooking(fixture))
		s.AssertLedgerSnapshotEqual(LedgerSnapshot{
			FBO:                  ledgerBeforeRetry.FBO,
			Accrued:              alpacadecimal.NewFromInt(15),
			OpenReceivable:       alpacadecimal.NewFromInt(-15),
			AuthorizedReceivable: ledgerBeforeRetry.AuthorizedReceivable,
			Wash:                 ledgerBeforeRetry.Wash,
			Earnings:             ledgerBeforeRetry.Earnings,
		}, s.CreateLedgerSnapshot(fixture.ledgerSnapshotInput()))
	})
}

type issuingRetryInvoice struct {
	Customer customer.CustomerID
	Invoice  billing.StandardInvoice
	Charges  [2]meta.ChargeID
}

func (i issuingRetryInvoice) ledgerSnapshotInput() LedgerSnapshotInput {
	return LedgerSnapshotInput{
		Namespace: i.Customer.Namespace,
		Customer:  i.Customer,
		Currency:  USD,
		CostBasis: mo.None[*alpacadecimal.Decimal](),
	}
}

type completedIssuingBooking struct {
	RunID        string
	LineID       string
	InvoiceID    string
	AccruedUsage *invoicedusage.AccruedUsage
}

func (s *CreditThenInvoiceTestSuite) setupFlatFeeIssuingRetryInvoice() issuingRetryInvoice {
	t := s.T()
	t.Helper()
	ctx := t.Context()
	ns := s.GetUniqueNamespace("flat-fee-issuing-retry")
	s.ProvisionDefaultTaxCodes(ctx, ns)
	invoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")
	s.ProvisionBillingProfile(ctx, ns, invoicing.App.GetID())
	period := timeutil.ClosedPeriod{From: clock.Now(), To: clock.Now().AddDate(0, 1, 0)}
	intents := make([]charges.ChargeIntent, 0, 2)
	for i, amount := range []int64{5, 10} {
		intents = append(intents, s.CreateMockChargeIntent(CreateMockChargeIntentInput{
			Customer:       cust.GetID(),
			Currency:       USD,
			ServicePeriod:  period,
			SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromInt(amount),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			ProRating:         productcatalog.ProRatingConfig{Enabled: true, Mode: productcatalog.ProRatingModeProratePrices},
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
	require.Equal(t, billing.StandardInvoiceStatusDraftWaitingAutoApproval, invoice.Status)
	require.NotNil(t, invoice.DraftUntil)
	require.Len(t, invoice.Lines.OrEmpty(), 2)
	fixture := issuingRetryInvoice{Customer: cust.GetID(), Invoice: invoice}
	for i, line := range invoice.Lines.OrEmpty() {
		require.NotNil(t, line.ChargeID)
		fixture.Charges[i] = meta.ChargeID{Namespace: ns, ID: *line.ChargeID}
		s.RequireFlatFeeChargeStatus(fixture.Charges[i], flatfee.StatusActiveRealizationProcessing)
		s.RequireTotals(billingtest.ExpectedTotals{Amount: float64((i + 1) * 5), Total: float64((i + 1) * 5)}, line.Totals)
	}

	return fixture
}

func (s *CreditThenInvoiceTestSuite) setupUsageBasedIssuingRetryInvoice() issuingRetryInvoice {
	t := s.T()
	t.Helper()
	ctx := t.Context()
	ns := s.GetUniqueNamespace("usage-based-issuing-retry")
	s.ProvisionDefaultTaxCodes(ctx, ns)
	invoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")
	s.ProvisionBillingProfile(ctx, ns, invoicing.App.GetID())
	feature := s.SetupApiRequestsTotalFeature(ctx, ns)
	period := timeutil.ClosedPeriod{From: clock.Now().AddDate(0, -1, 0), To: clock.Now()}
	s.MockStreamingConnector.AddSimpleEvent(feature.Feature.Key, 5, period.From.Add(15*24*time.Hour))
	intents := make([]charges.ChargeIntent, 0, 2)
	for i, unitPrice := range []int64{1, 2} {
		intents = append(intents, s.CreateMockChargeIntent(CreateMockChargeIntentInput{
			Customer:          cust.GetID(),
			Currency:          USD,
			ServicePeriod:     period,
			SettlementMode:    productcatalog.CreditThenInvoiceSettlementMode,
			Price:             productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(unitPrice)}),
			ProRating:         productcatalog.ProRatingConfig{Enabled: true, Mode: productcatalog.ProRatingModeProratePrices},
			FeatureKey:        feature.Feature.Key,
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
	require.Equal(t, billing.StandardInvoiceStatusDraftWaitingForCollection, invoice.Status)
	clock.FreezeTime(invoice.DefaultCollectionAtForStandardInvoice())
	defer clock.UnFreeze()
	invoice, err = s.BillingService.AdvanceInvoice(ctx, invoice.GetInvoiceID())
	require.NoError(t, err)
	require.Equal(t, billing.StandardInvoiceStatusDraftWaitingAutoApproval, invoice.Status)
	require.NotNil(t, invoice.DraftUntil)
	require.Len(t, invoice.Lines.OrEmpty(), 2)
	fixture := issuingRetryInvoice{Customer: cust.GetID(), Invoice: invoice}
	for i, line := range invoice.Lines.OrEmpty() {
		require.NotNil(t, line.ChargeID)
		fixture.Charges[i] = meta.ChargeID{Namespace: ns, ID: *line.ChargeID}
		s.RequireUsageBasedChargeStatus(fixture.Charges[i], usagebased.StatusActiveRealizationProcessing)
		s.RequireTotals(billingtest.ExpectedTotals{Amount: float64((i + 1) * 5), Total: float64((i + 1) * 5)}, line.Totals)
	}

	return fixture
}

func (s *CreditThenInvoiceTestSuite) requireCompletedFlatFeeIssuingBooking(fixture issuingRetryInvoice) completedIssuingBooking {
	t := s.T()
	t.Helper()
	charge := s.RequireFlatFeeChargeStatus(fixture.Charges[0], flatfee.StatusActiveAwaitingPaymentSettlement)
	run := charge.Realizations.CurrentRun
	require.NotNil(t, run)
	require.True(t, run.Immutable)
	require.Nil(t, run.DeletedAt)
	require.Nil(t, run.Payment)
	booking := completedIssuingBooking{
		RunID:        run.ID.ID,
		LineID:       lo.FromPtr(run.LineID),
		InvoiceID:    lo.FromPtr(run.InvoiceID),
		AccruedUsage: run.AccruedUsage,
	}
	s.requireCompletedIssuingBooking(fixture, booking)

	return booking
}

func (s *CreditThenInvoiceTestSuite) requireCompletedUsageBasedIssuingBooking(fixture issuingRetryInvoice) completedIssuingBooking {
	t := s.T()
	t.Helper()
	charge := s.RequireUsageBasedChargeStatus(fixture.Charges[0], usagebased.StatusActiveAwaitingPaymentSettlement)
	require.Nil(t, charge.State.CurrentRealizationRunID)
	run, err := charge.Realizations.GetByLineID(fixture.Invoice.Lines.OrEmpty()[0].ID)
	require.NoError(t, err)
	require.True(t, run.Immutable)
	require.Nil(t, run.DeletedAt)
	require.Nil(t, run.Payment)
	booking := completedIssuingBooking{
		RunID:        run.ID.ID,
		LineID:       lo.FromPtr(run.LineID),
		InvoiceID:    lo.FromPtr(run.InvoiceID),
		AccruedUsage: run.InvoiceUsage,
	}
	s.requireCompletedIssuingBooking(fixture, booking)

	return booking
}

func (s *CreditThenInvoiceTestSuite) requireCompletedIssuingBooking(fixture issuingRetryInvoice, booking completedIssuingBooking) {
	t := s.T()
	t.Helper()
	firstLine := fixture.Invoice.Lines.OrEmpty()[0]
	require.NotEmpty(t, booking.RunID)
	require.Equal(t, firstLine.ID, booking.LineID)
	require.Equal(t, fixture.Invoice.ID, booking.InvoiceID)
	require.NotNil(t, booking.AccruedUsage)
	require.NotNil(t, booking.AccruedUsage.LedgerTransaction)
	require.Equal(t, firstLine.Totals.Total.InexactFloat64(), booking.AccruedUsage.Totals.Total.InexactFloat64())
}

func (s *CreditThenInvoiceTestSuite) failIssuingAfterFirstBooking(invoice billing.StandardInvoice, engine billing.LineEngine) *failOnceIssuingLineEngine {
	t := s.T()
	t.Helper()
	fault := &failOnceIssuingLineEngine{LineEngine: engine, FailLineID: invoice.Lines.OrEmpty()[1].ID}
	require.NoError(t, s.BillingService.DeregisterLineEngine(engine.GetLineEngineType()))
	require.NoError(t, s.BillingService.RegisterLineEngine(fault))
	restoration := issuingLineEngineRestoration{T: t, Billing: s.BillingService, Engine: engine}
	t.Cleanup(restoration.restore)

	return fault
}

type issuingLineEngineRestoration struct {
	T       *testing.T
	Billing billing.Service
	Engine  billing.LineEngine
}

func (r issuingLineEngineRestoration) restore() {
	require.NoError(r.T, r.Billing.DeregisterLineEngine(r.Engine.GetLineEngineType()))
	require.NoError(r.T, r.Billing.RegisterLineEngine(r.Engine))
}

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
