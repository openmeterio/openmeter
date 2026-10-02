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
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
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

	s.Run("when authorization fails after the first line writes", func() {
		t := s.T()

		failed = s.authorizeInvoiceWithSecondLineFailure(invoice, s.FlatFeeSvc.GetLineEngine())
		require.Equal(t, billing.StandardInvoiceStatusPaymentProcessingBookingAuthorizedFailed, failed.Status)
		require.True(t, failed.HasCriticalValidationIssues())
		firstBeforeRetry = s.requireFlatFeeInvoiceRun(lines[0])
		secondBeforeRetry = s.requireFlatFeeInvoiceRun(lines[1])
		require.Nil(t, firstBeforeRetry.Payment)
		require.Nil(t, secondBeforeRetry.Payment)
		beforeTransactionIDs = s.paymentAuthorizationTransactionIDs(ns)
		require.Len(t, beforeTransactionIDs, 2)
	})

	s.Run("given a historical first-line authorization committed before retry", func() {
		// This partial booking represents an old database state; new attempts are atomic.
		// The setup is only needed to verify recovery from data written before that guarantee.
		t := s.T()
		ctx := t.Context()

		require.NoError(t, s.FlatFeeSvc.GetLineEngine().OnPaymentAuthorized(ctx, billing.OnPaymentAuthorizedInput{
			Invoice: failed,
			Lines:   billing.StandardLines{lines[0]},
		}))
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

	s.Run("when authorization fails after the first line writes", func() {
		t := s.T()

		failed = s.authorizeInvoiceWithSecondLineFailure(invoice, s.UsageBasedSvc.GetLineEngine())
		require.Equal(t, billing.StandardInvoiceStatusPaymentProcessingBookingAuthorizedFailed, failed.Status)
		require.True(t, failed.HasCriticalValidationIssues())
		firstBeforeRetry = s.requireUsageBasedInvoiceRun(lines[0])
		secondBeforeRetry = s.requireUsageBasedInvoiceRun(lines[1])
		require.Nil(t, firstBeforeRetry.Payment)
		require.Nil(t, secondBeforeRetry.Payment)
		beforeTransactionIDs = s.paymentAuthorizationTransactionIDs(ns)
		require.Len(t, beforeTransactionIDs, 2)
	})

	s.Run("given a historical first-line authorization committed before retry", func() {
		// This partial booking represents an old database state; new attempts are atomic.
		// The setup is only needed to verify recovery from data written before that guarantee.
		t := s.T()
		ctx := t.Context()

		require.NoError(t, s.UsageBasedSvc.GetLineEngine().OnPaymentAuthorized(ctx, billing.OnPaymentAuthorizedInput{
			Invoice: failed,
			Lines:   billing.StandardLines{lines[0]},
		}))
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

	s.Run("when authorization fails after the first line writes", func() {
		t := s.T()

		failed = s.authorizeInvoiceWithSecondLineFailure(invoice, s.CreditPurchaseSvc.GetLineEngine())
		require.Equal(t, billing.StandardInvoiceStatusPaymentProcessingBookingAuthorizedFailed, failed.Status)
		require.True(t, failed.HasCriticalValidationIssues())
		firstBeforeRetry = s.requireInvoiceFundedCreditPurchase(lines[0])
		secondBeforeRetry = s.requireInvoiceFundedCreditPurchase(lines[1])
		require.Nil(t, firstBeforeRetry.Realizations.InvoiceSettlement)
		require.Nil(t, secondBeforeRetry.Realizations.InvoiceSettlement)
		beforeTransactionIDs = s.paymentAuthorizationTransactionIDs(ns)
		require.Len(t, beforeTransactionIDs, 2)
	})

	s.Run("given a historical first-line authorization committed before retry", func() {
		// This partial booking represents an old database state; new attempts are atomic.
		// The setup is only needed to verify recovery from data written before that guarantee.
		t := s.T()
		ctx := t.Context()

		require.NoError(t, s.CreditPurchaseSvc.GetLineEngine().OnPaymentAuthorized(ctx, billing.OnPaymentAuthorizedInput{
			Invoice: failed,
			Lines:   billing.StandardLines{lines[0]},
		}))
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

func (s *CreditThenInvoiceTestSuite) TestFlatFeePaymentSettlementFailureRollsBackEveryLine() {
	clock.FreezeTime(time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC))
	defer clock.UnFreeze()

	t := s.T()
	ctx := t.Context()
	ns := s.GetUniqueNamespace("flat-fee-payment-settlement-atomic")
	invoice := s.setupTwoFlatFeePaymentInvoice(ns)
	lines := invoice.Lines.OrEmpty()
	require.Len(t, lines, 2)

	var err error
	invoice, err = s.BillingService.PaymentAuthorized(ctx, invoice.GetInvoiceID())
	require.NoError(t, err)
	require.Equal(t, billing.StandardInvoiceStatusPaymentProcessingAuthorized, invoice.Status)
	for _, line := range lines {
		run := s.requireFlatFeeInvoiceRun(line)
		s.requireInvoiceAuthorizationBooking(line, run.Payment)
	}
	beforeSettlementTransactionIDs := s.paymentAuthorizationTransactionIDs(ns)

	engine := s.FlatFeeSvc.GetLineEngine()
	fault := &failOncePaymentSettlementLineEngine{LineEngine: engine, FailLineID: lines[1].ID}
	s.replaceLineEngine(t, engine, fault)

	require.NoError(t, s.BillingService.TriggerInvoice(ctx, billing.InvoiceTriggerServiceInput{
		InvoiceTriggerInput: billing.InvoiceTriggerInput{Invoice: invoice.GetInvoiceID(), Trigger: billing.TriggerPaid},
		AppType:             app.AppTypeCustomInvoicing,
		Capability:          app.CapabilityTypeCollectPayments,
	}))
	require.True(t, fault.Failed)

	failed, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{Invoice: invoice.GetInvoiceID()})
	require.NoError(t, err)
	require.Equal(t, billing.StandardInvoiceStatusPaymentProcessingBookingSettledFailed, failed.Status)
	require.True(t, failed.HasCriticalValidationIssues())
	for _, line := range lines {
		run := s.requireFlatFeeInvoiceRun(line)
		s.requireInvoiceAuthorizationBooking(line, run.Payment)
	}
	require.ElementsMatch(t, beforeSettlementTransactionIDs, s.paymentAuthorizationTransactionIDs(ns))
}

func (s *CreditThenInvoiceTestSuite) TestFlatFeeDirectPaymentFailureRollsBackAuthorizationAndSettlement() {
	clock.FreezeTime(time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC))
	defer clock.UnFreeze()

	t := s.T()
	ctx := t.Context()
	ns := s.GetUniqueNamespace("flat-fee-direct-payment-atomic")
	invoice := s.setupTwoFlatFeePaymentInvoice(ns)
	lines := invoice.Lines.OrEmpty()
	require.Len(t, lines, 2)
	beforePaymentTransactionIDs := s.paymentAuthorizationTransactionIDs(ns)

	engine := s.FlatFeeSvc.GetLineEngine()
	fault := &failOncePaymentSettlementLineEngine{LineEngine: engine, FailLineID: lines[1].ID}
	s.replaceLineEngine(t, engine, fault)

	require.NoError(t, s.BillingService.TriggerInvoice(ctx, billing.InvoiceTriggerServiceInput{
		InvoiceTriggerInput: billing.InvoiceTriggerInput{Invoice: invoice.GetInvoiceID(), Trigger: billing.TriggerPaid},
		AppType:             app.AppTypeCustomInvoicing,
		Capability:          app.CapabilityTypeCollectPayments,
	}))
	require.True(t, fault.Failed)

	failed, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{Invoice: invoice.GetInvoiceID()})
	require.NoError(t, err)
	require.Equal(t, billing.StandardInvoiceStatusPaymentProcessingBookingAuthorizedAndSettledFailed, failed.Status)
	require.True(t, failed.HasCriticalValidationIssues())
	for _, line := range lines {
		run := s.requireFlatFeeInvoiceRun(line)
		require.Nil(t, run.Payment)
	}
	require.ElementsMatch(t, beforePaymentTransactionIDs, s.paymentAuthorizationTransactionIDs(ns))
}

func (s *CreditThenInvoiceTestSuite) TestPaymentAuthorizationFailureRollsBackAcrossLineEngines() {
	clock.FreezeTime(time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC))
	defer clock.UnFreeze()
	defer s.MockStreamingConnector.Reset()

	t := s.T()
	ctx := t.Context()
	ns := s.GetUniqueNamespace("payment-authorization-cross-engine-atomic")
	invoice := s.setupMixedPaymentInvoice(ns)
	clock.FreezeTime(invoice.DefaultCollectionAtForStandardInvoice())
	defer clock.UnFreeze()

	var err error
	invoice, err = s.BillingService.AdvanceInvoice(ctx, invoice.GetInvoiceID())
	require.NoError(t, err)
	invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
	require.NoError(t, err)
	require.Equal(t, billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)
	require.Len(t, invoice.Lines.OrEmpty(), 2)
	beforeAuthorizationTransactionIDs := s.paymentAuthorizationTransactionIDs(ns)

	state := &failSecondPaymentAuthorizationState{}
	flatFeeEngine := s.FlatFeeSvc.GetLineEngine()
	usageBasedEngine := s.UsageBasedSvc.GetLineEngine()
	s.replaceLineEngine(t, flatFeeEngine, &failSecondPaymentAuthorizationLineEngine{LineEngine: flatFeeEngine, State: state})
	s.replaceLineEngine(t, usageBasedEngine, &failSecondPaymentAuthorizationLineEngine{LineEngine: usageBasedEngine, State: state})

	require.NoError(t, s.BillingService.TriggerInvoice(ctx, billing.InvoiceTriggerServiceInput{
		InvoiceTriggerInput: billing.InvoiceTriggerInput{Invoice: invoice.GetInvoiceID(), Trigger: billing.TriggerAuthorized},
		AppType:             app.AppTypeCustomInvoicing,
		Capability:          app.CapabilityTypeCollectPayments,
	}))
	require.True(t, state.Failed)
	require.Equal(t, 2, state.Calls)

	failed, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{Invoice: invoice.GetInvoiceID()})
	require.NoError(t, err)
	require.Equal(t, billing.StandardInvoiceStatusPaymentProcessingBookingAuthorizedFailed, failed.Status)
	require.True(t, failed.HasCriticalValidationIssues())
	for _, line := range invoice.Lines.OrEmpty() {
		switch line.Engine {
		case billing.LineEngineTypeChargeFlatFee:
			run := s.requireFlatFeeInvoiceRun(line)
			require.Nil(t, run.Payment)
		case billing.LineEngineTypeChargeUsageBased:
			run := s.requireUsageBasedInvoiceRun(line)
			require.Nil(t, run.Payment)
		default:
			require.Failf(t, "unexpected line engine", "engine=%s", line.Engine)
		}
	}
	require.ElementsMatch(t, beforeAuthorizationTransactionIDs, s.paymentAuthorizationTransactionIDs(ns))
}

func (s *CreditThenInvoiceTestSuite) setupTwoFlatFeePaymentInvoice(namespace string) billing.StandardInvoice {
	t := s.T()
	t.Helper()
	ctx := t.Context()

	s.ProvisionDefaultTaxCodes(ctx, namespace)
	invoicing := s.SetupCustomInvoicing(namespace)
	cust := s.CreateLedgerBackedCustomer(namespace, "test-subject")
	s.ProvisionBillingProfile(ctx, namespace, invoicing.App.GetID())
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
	created, err := s.Charges.Create(ctx, charges.CreateInput{Namespace: namespace, Intents: charges.NewCreateChargeIntents(intents...)})
	require.NoError(t, err)
	require.Len(t, created, 2)
	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: cust.GetID(),
		AsOf:     lo.ToPtr(clock.Now()),
	})
	require.NoError(t, err)
	require.Len(t, invoices, 1)

	invoice, err := s.BillingService.ApproveInvoice(ctx, invoices[0].GetInvoiceID())
	require.NoError(t, err)
	require.Equal(t, billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)
	require.Equal(t, float64(15), invoice.Totals.Total.InexactFloat64())

	return invoice
}

func (s *CreditThenInvoiceTestSuite) setupMixedPaymentInvoice(namespace string) billing.StandardInvoice {
	t := s.T()
	t.Helper()
	ctx := t.Context()

	s.ProvisionDefaultTaxCodes(ctx, namespace)
	invoicing := s.SetupCustomInvoicing(namespace)
	cust := s.CreateLedgerBackedCustomer(namespace, "test-subject")
	s.ProvisionBillingProfile(ctx, namespace, invoicing.App.GetID())
	feature := s.SetupApiRequestsTotalFeature(ctx, namespace)
	period := timeutil.ClosedPeriod{From: clock.Now().AddDate(0, -1, 0), To: clock.Now()}
	s.MockStreamingConnector.AddSimpleEvent(feature.Feature.Key, 5, period.From.Add(15*24*time.Hour))
	created, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: namespace,
		Intents: charges.NewCreateChargeIntents(
			s.CreateMockChargeIntent(CreateMockChargeIntentInput{
				Customer:       cust.GetID(),
				Currency:       USD,
				ServicePeriod:  period,
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromInt(5),
					PaymentTerm: productcatalog.InArrearsPaymentTerm,
				}),
				ProRating:         productcatalog.ProRatingConfig{Enabled: true, Mode: productcatalog.ProRatingModeProratePrices},
				Name:              "flat fee charge",
				ManagedBy:         billing.SubscriptionManagedLine,
				UniqueReferenceID: "flat-fee-charge",
			}),
			s.CreateMockChargeIntent(CreateMockChargeIntentInput{
				Customer:          cust.GetID(),
				Currency:          USD,
				ServicePeriod:     period,
				SettlementMode:    productcatalog.CreditThenInvoiceSettlementMode,
				Price:             productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(2)}),
				ProRating:         productcatalog.ProRatingConfig{Enabled: true, Mode: productcatalog.ProRatingModeProratePrices},
				FeatureKey:        feature.Feature.Key,
				Name:              "usage based charge",
				ManagedBy:         billing.SubscriptionManagedLine,
				UniqueReferenceID: "usage-based-charge",
			}),
		),
	})
	require.NoError(t, err)
	require.Len(t, created, 2)
	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: cust.GetID(),
		AsOf:     lo.ToPtr(clock.Now()),
	})
	require.NoError(t, err)
	require.Len(t, invoices, 1)
	require.Equal(t, billing.StandardInvoiceStatusDraftWaitingForCollection, invoices[0].Status)

	return invoices[0]
}

func (s *CreditThenInvoiceTestSuite) authorizeInvoiceWithSecondLineFailure(invoice billing.StandardInvoice, engine billing.LineEngine) billing.StandardInvoice {
	t := s.T()
	t.Helper()
	ctx := t.Context()
	fault := &failOncePaymentAuthorizationLineEngine{LineEngine: engine, FailLineID: invoice.Lines.OrEmpty()[1].ID}
	s.replaceLineEngine(t, engine, fault)

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

func (s *CreditThenInvoiceTestSuite) replaceLineEngine(t *testing.T, original, replacement billing.LineEngine) {
	t.Helper()
	require.Equal(t, original.GetLineEngineType(), replacement.GetLineEngineType())
	require.NoError(t, s.BillingService.DeregisterLineEngine(original.GetLineEngineType()))
	require.NoError(t, s.BillingService.RegisterLineEngine(replacement))
	restoration := lineEngineRestoration{T: t, Billing: s.BillingService, Engine: original}
	t.Cleanup(restoration.restore)
}

type lineEngineRestoration struct {
	T       *testing.T
	Billing billing.Service
	Engine  billing.LineEngine
}

func (r lineEngineRestoration) restore() {
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

type failOncePaymentSettlementLineEngine struct {
	billing.LineEngine
	FailLineID string
	Failed     bool
}

type failSecondPaymentAuthorizationState struct {
	Calls  int
	Failed bool
}

type failSecondPaymentAuthorizationLineEngine struct {
	billing.LineEngine
	State *failSecondPaymentAuthorizationState
}

func (e *failSecondPaymentAuthorizationLineEngine) OnPaymentAuthorized(ctx context.Context, input billing.OnPaymentAuthorizedInput) error {
	e.State.Calls++
	if e.State.Calls == 2 {
		e.State.Failed = true
		return billing.ValidationIssue{
			Severity: billing.ValidationIssueSeverityCritical,
			Code:     "test_cross_engine_callback_failed",
			Message:  "second line engine callback failed",
		}
	}

	return e.LineEngine.OnPaymentAuthorized(ctx, input)
}

func (e *failOncePaymentSettlementLineEngine) OnPaymentSettled(ctx context.Context, input billing.OnPaymentSettledInput) error {
	if e.Failed {
		return e.LineEngine.OnPaymentSettled(ctx, input)
	}

	for _, line := range input.Lines {
		if line.ID == e.FailLineID {
			e.Failed = true
			return billing.ValidationIssue{
				Severity: billing.ValidationIssueSeverityCritical,
				Code:     "test_transient_line_callback_failed",
				Message:  "transient line callback failure after prior progress",
			}
		}

		one := input
		one.Lines = billing.StandardLines{line}
		if err := e.LineEngine.OnPaymentSettled(ctx, one); err != nil {
			return err
		}
	}

	return nil
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
				Message:  "transient line callback failure after prior progress",
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

const (
	issuingCompletedLineName = "completed charge"
	issuingRemainingLineName = "remaining charge"
)

func (s *CreditThenInvoiceTestSuite) TestFlatFeeIssuingRetryPreservesCompletedChargeBooking() {
	ctx := s.T().Context()
	clock.FreezeTime(time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC))
	defer clock.UnFreeze()

	var fixture issuingRetryInvoice
	var ledgerBeforeAttempt LedgerSnapshot
	s.Run("create an invoice with two flat fee charges", func() {
		// given an invoice with two independently bookable flat fee charges
		t := s.T()
		fixture = s.setupFlatFeeIssuingRetryInvoice()
		require.Equal(t, float64(15), fixture.Invoice.Totals.Total.InexactFloat64())
		ledgerBeforeAttempt = s.CreateLedgerSnapshot(fixture.ledgerSnapshotInput())
	})

	var failedInvoice billing.StandardInvoice
	var completedBeforeRetry completedIssuingBooking
	var ledgerBeforeRetry LedgerSnapshot
	s.Run("fail issuance after the first charge booking writes", func() {
		// when invoice issuance fails after the first charge booking writes
		// then the complete line-engine attempt is rolled back
		t := s.T()
		clock.FreezeTime(fixture.Invoice.DraftUntil.Add(time.Second))
		defer clock.UnFreeze()

		fault := s.failIssuingAfterCompletedChargeBooking(s.FlatFeeSvc.GetLineEngine())
		var err error
		failedInvoice, err = s.BillingService.AdvanceInvoice(ctx, fixture.Invoice.GetInvoiceID())
		require.NoError(t, err)
		require.True(t, fault.Failed)
		require.Equal(t, billing.StandardInvoiceStatusIssuingChargeBookingFailed, failedInvoice.Status)
		require.True(t, failedInvoice.HasCriticalValidationIssues())

		chargeID := fixture.ChargesByLineName[issuingCompletedLineName]
		charge := s.RequireFlatFeeChargeStatus(chargeID, flatfee.StatusActiveRealizationIssuing)
		require.NotNil(t, charge.Realizations.CurrentRun)
		require.Nil(t, charge.Realizations.CurrentRun.AccruedUsage)
		require.Nil(t, charge.Realizations.CurrentRun.Payment)
		s.AssertLedgerSnapshotUnchanged(fixture.ledgerSnapshotInput(), ledgerBeforeAttempt)
	})

	s.Run("record a historical first-charge booking before retry", func() {
		// This partial booking represents an old database state; new attempts are atomic.
		// The setup is only needed to verify recovery from data written before that guarantee.
		t := s.T()
		line := fixture.LinesByName[issuingCompletedLineName]
		require.NotNil(t, line)
		require.NoError(t, s.FlatFeeSvc.GetLineEngine().OnInvoiceIssued(ctx, billing.OnInvoiceIssuedInput{
			Invoice: failedInvoice,
			Lines:   billing.StandardLines{line},
		}))

		completedBeforeRetry = s.requireCompletedFlatFeeIssuingBooking(fixture)
		ledgerBeforeRetry = s.CreateLedgerSnapshot(fixture.ledgerSnapshotInput())
		require.Equal(t, float64(5), ledgerBeforeRetry.Accrued.InexactFloat64())
	})

	s.Run("retry books only the remaining charge", func() {
		// then retry preserves the completed booking and books the remaining charge
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
	var ledgerBeforeAttempt LedgerSnapshot
	s.Run("create an invoice with two usage based charges", func() {
		// given an invoice with two independently bookable usage based charges
		t := s.T()
		fixture = s.setupUsageBasedIssuingRetryInvoice()
		require.Equal(t, float64(15), fixture.Invoice.Totals.Total.InexactFloat64())
		ledgerBeforeAttempt = s.CreateLedgerSnapshot(fixture.ledgerSnapshotInput())
	})

	var failedInvoice billing.StandardInvoice
	var completedBeforeRetry completedIssuingBooking
	var ledgerBeforeRetry LedgerSnapshot
	s.Run("fail issuance after the first charge booking writes", func() {
		// when invoice issuance fails after the first charge booking writes
		// then the complete line-engine attempt is rolled back
		t := s.T()
		clock.FreezeTime(fixture.Invoice.DraftUntil.Add(time.Second))
		defer clock.UnFreeze()

		fault := s.failIssuingAfterCompletedChargeBooking(s.UsageBasedSvc.GetLineEngine())
		var err error
		failedInvoice, err = s.BillingService.AdvanceInvoice(ctx, fixture.Invoice.GetInvoiceID())
		require.NoError(t, err)
		require.True(t, fault.Failed)
		require.Equal(t, billing.StandardInvoiceStatusIssuingChargeBookingFailed, failedInvoice.Status)
		require.True(t, failedInvoice.HasCriticalValidationIssues())

		chargeID := fixture.ChargesByLineName[issuingCompletedLineName]
		charge := s.RequireUsageBasedChargeStatus(chargeID, usagebased.StatusActiveRealizationIssuing)
		require.NotNil(t, charge.State.CurrentRealizationRunID)
		currentRun, err := charge.Realizations.GetByID(*charge.State.CurrentRealizationRunID)
		require.NoError(t, err)
		require.Nil(t, currentRun.InvoiceUsage)
		require.Nil(t, currentRun.Payment)
		s.AssertLedgerSnapshotUnchanged(fixture.ledgerSnapshotInput(), ledgerBeforeAttempt)
	})

	s.Run("record a historical first-charge booking before retry", func() {
		// This partial booking represents an old database state; new attempts are atomic.
		// The setup is only needed to verify recovery from data written before that guarantee.
		t := s.T()
		line := fixture.LinesByName[issuingCompletedLineName]
		require.NotNil(t, line)
		require.NoError(t, s.UsageBasedSvc.GetLineEngine().OnInvoiceIssued(ctx, billing.OnInvoiceIssuedInput{
			Invoice: failedInvoice,
			Lines:   billing.StandardLines{line},
		}))

		completedBeforeRetry = s.requireCompletedUsageBasedIssuingBooking(fixture)
		ledgerBeforeRetry = s.CreateLedgerSnapshot(fixture.ledgerSnapshotInput())
		require.Equal(t, float64(5), ledgerBeforeRetry.Accrued.InexactFloat64())
	})

	s.Run("retry books only the remaining charge", func() {
		// then retry preserves the completed booking and books the remaining charge
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
	Customer          customer.CustomerID
	Invoice           billing.StandardInvoice
	LinesByName       map[string]*billing.StandardLine
	ChargesByLineName map[string]meta.ChargeID
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
	chargeInputs := []struct {
		Name   string
		Amount int64
	}{
		{Name: issuingCompletedLineName, Amount: 5},
		{Name: issuingRemainingLineName, Amount: 10},
	}
	intents := make([]charges.ChargeIntent, 0, len(chargeInputs))
	for _, chargeInput := range chargeInputs {
		intents = append(intents, s.CreateMockChargeIntent(CreateMockChargeIntentInput{
			Customer:       cust.GetID(),
			Currency:       USD,
			ServicePeriod:  period,
			SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromInt(chargeInput.Amount),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			ProRating:         productcatalog.ProRatingConfig{Enabled: true, Mode: productcatalog.ProRatingModeProratePrices},
			Name:              chargeInput.Name,
			ManagedBy:         billing.SubscriptionManagedLine,
			UniqueReferenceID: chargeInput.Name,
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
	fixture := issuingRetryInvoice{
		Customer: cust.GetID(),
		Invoice:  invoice,
		LinesByName: lo.KeyBy(invoice.Lines.OrEmpty(), func(line *billing.StandardLine) string {
			return line.Name
		}),
		ChargesByLineName: make(map[string]meta.ChargeID, 2),
	}
	require.Len(t, fixture.LinesByName, 2)
	for lineName, expectedAmount := range map[string]float64{
		issuingCompletedLineName: 5,
		issuingRemainingLineName: 10,
	} {
		line := fixture.LinesByName[lineName]
		require.NotNil(t, line)
		require.NotNil(t, line.ChargeID)
		chargeID := meta.ChargeID{Namespace: ns, ID: *line.ChargeID}
		fixture.ChargesByLineName[lineName] = chargeID
		s.RequireFlatFeeChargeStatus(chargeID, flatfee.StatusActiveRealizationProcessing)
		s.RequireTotals(billingtest.ExpectedTotals{Amount: expectedAmount, Total: expectedAmount}, line.Totals)
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
	chargeInputs := []struct {
		Name      string
		UnitPrice int64
	}{
		{Name: issuingCompletedLineName, UnitPrice: 1},
		{Name: issuingRemainingLineName, UnitPrice: 2},
	}
	intents := make([]charges.ChargeIntent, 0, len(chargeInputs))
	for _, chargeInput := range chargeInputs {
		intents = append(intents, s.CreateMockChargeIntent(CreateMockChargeIntentInput{
			Customer:          cust.GetID(),
			Currency:          USD,
			ServicePeriod:     period,
			SettlementMode:    productcatalog.CreditThenInvoiceSettlementMode,
			Price:             productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(chargeInput.UnitPrice)}),
			ProRating:         productcatalog.ProRatingConfig{Enabled: true, Mode: productcatalog.ProRatingModeProratePrices},
			FeatureKey:        feature.Feature.Key,
			Name:              chargeInput.Name,
			ManagedBy:         billing.SubscriptionManagedLine,
			UniqueReferenceID: chargeInput.Name,
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
	fixture := issuingRetryInvoice{
		Customer: cust.GetID(),
		Invoice:  invoice,
		LinesByName: lo.KeyBy(invoice.Lines.OrEmpty(), func(line *billing.StandardLine) string {
			return line.Name
		}),
		ChargesByLineName: make(map[string]meta.ChargeID, 2),
	}
	require.Len(t, fixture.LinesByName, 2)
	for lineName, expectedAmount := range map[string]float64{
		issuingCompletedLineName: 5,
		issuingRemainingLineName: 10,
	} {
		line := fixture.LinesByName[lineName]
		require.NotNil(t, line)
		require.NotNil(t, line.ChargeID)
		chargeID := meta.ChargeID{Namespace: ns, ID: *line.ChargeID}
		fixture.ChargesByLineName[lineName] = chargeID
		s.RequireUsageBasedChargeStatus(chargeID, usagebased.StatusActiveRealizationProcessing)
		s.RequireTotals(billingtest.ExpectedTotals{Amount: expectedAmount, Total: expectedAmount}, line.Totals)
	}

	return fixture
}

func (s *CreditThenInvoiceTestSuite) requireCompletedFlatFeeIssuingBooking(fixture issuingRetryInvoice) completedIssuingBooking {
	t := s.T()
	t.Helper()
	chargeID, ok := fixture.ChargesByLineName[issuingCompletedLineName]
	require.True(t, ok)
	charge := s.RequireFlatFeeChargeStatus(chargeID, flatfee.StatusActiveAwaitingPaymentSettlement)
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
	chargeID, ok := fixture.ChargesByLineName[issuingCompletedLineName]
	require.True(t, ok)
	charge := s.RequireUsageBasedChargeStatus(chargeID, usagebased.StatusActiveAwaitingPaymentSettlement)
	require.Nil(t, charge.State.CurrentRealizationRunID)
	completedLine := fixture.LinesByName[issuingCompletedLineName]
	require.NotNil(t, completedLine)
	run, err := charge.Realizations.GetByLineID(completedLine.ID)
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
	completedLine := fixture.LinesByName[issuingCompletedLineName]
	require.NotNil(t, completedLine)
	require.NotEmpty(t, booking.RunID)
	require.Equal(t, completedLine.ID, booking.LineID)
	require.Equal(t, fixture.Invoice.ID, booking.InvoiceID)
	require.NotNil(t, booking.AccruedUsage)
	require.NotNil(t, booking.AccruedUsage.LedgerTransaction)
	require.Equal(t, completedLine.Totals.Total.InexactFloat64(), booking.AccruedUsage.Totals.Total.InexactFloat64())
}

func (s *CreditThenInvoiceTestSuite) failIssuingAfterCompletedChargeBooking(engine billing.LineEngine) *failOnceIssuingLineEngine {
	t := s.T()
	t.Helper()
	fault := &failOnceIssuingLineEngine{
		LineEngine:        engine,
		CompletedLineName: issuingCompletedLineName,
		FailLineName:      issuingRemainingLineName,
	}
	s.replaceLineEngine(t, engine, fault)

	return fault
}

type failOnceIssuingLineEngine struct {
	billing.LineEngine
	CompletedLineName string
	FailLineName      string
	Failed            bool
}

func (e *failOnceIssuingLineEngine) OnInvoiceIssued(ctx context.Context, input billing.OnInvoiceIssuedInput) error {
	if e.Failed {
		return e.LineEngine.OnInvoiceIssued(ctx, input)
	}

	linesByName := lo.KeyBy(input.Lines, func(line *billing.StandardLine) string {
		return line.Name
	})
	completedLine, ok := linesByName[e.CompletedLineName]
	if !ok {
		return fmt.Errorf("completed line %q is missing", e.CompletedLineName)
	}
	if _, ok := linesByName[e.FailLineName]; !ok {
		return fmt.Errorf("failing line %q is missing", e.FailLineName)
	}

	lineInput := input
	lineInput.Lines = billing.StandardLines{completedLine}
	if err := e.LineEngine.OnInvoiceIssued(ctx, lineInput); err != nil {
		return err
	}

	e.Failed = true

	return billing.ValidationIssue{
		Severity: billing.ValidationIssueSeverityCritical,
		Code:     "test_issuing_callback_failed",
		Message:  "transient invoice-issued callback failure",
	}
}

func (s *CreditThenInvoiceTestSuite) TestInvoiceFinalizationRetryPreservesCompletedLinePreparation() {
	for _, chargeType := range []meta.ChargeType{meta.ChargeTypeFlatFee, meta.ChargeTypeUsageBased} {
		s.Run(string(chargeType), func() {
			t := s.T()
			ctx := t.Context()
			clock.FreezeTime(time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC))
			defer clock.UnFreeze()
			defer s.MockStreamingConnector.Reset()

			// given line preparation commits for the first charge before the second callback fails
			fixture := s.setupChargeBookingInvoice(chargeType)
			engine := s.bookingRetryLineEngine(chargeType)
			fault := &failOnceFinalizingLineEngine{LineEngine: engine, FailLineID: fixture.Invoice.Lines.OrEmpty()[1].ID}
			s.replaceBookingRetryLineEngine(fault, engine)
			failed, err := s.BillingService.ApproveInvoice(ctx, fixture.Invoice.GetInvoiceID())
			require.NoError(t, err)
			require.True(t, fault.Failed)
			require.Equal(t, billing.StandardInvoiceStatusIssuingLineFinalizationFailed, failed.Status)
			before := s.bookingRetryProgress(fixture)
			require.Equal(t, "active.realization.issuing", before[0].Status)
			require.Equal(t, "active.realization.processing", before[1].Status)
			require.Empty(t, s.bookingRetryTransactionIDs(fixture))

			// when finalization resumes from the persisted partially prepared charges
			invoice, err := s.BillingService.RetryInvoice(ctx, failed.GetInvoiceID())
			require.NoError(t, err)

			// then both lines finish issuing while the already prepared run keeps its identity
			s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status, "issues: %+v", invoice.ValidationIssues)
			s.False(invoice.HasCriticalValidationIssues())
			after := s.bookingRetryProgress(fixture)
			for i := range before {
				s.Equal(before[i].RunID, after[i].RunID)
				s.Equal("active.awaiting_payment_settlement", after[i].Status)
				s.NotEmpty(after[i].UsageGroupID)
				s.Nil(after[i].Payment)
			}
			s.Len(s.bookingRetryTransactionIDs(fixture), 2)
		})
	}
}

func (s *CreditThenInvoiceTestSuite) TestPaymentSettlementRetryPreservesCompletedLineBooking() {
	for _, chargeType := range []meta.ChargeType{meta.ChargeTypeFlatFee, meta.ChargeTypeUsageBased, meta.ChargeTypeCreditPurchase} {
		s.Run(string(chargeType), func() {
			t := s.T()
			ctx := t.Context()
			clock.FreezeTime(time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC))
			defer clock.UnFreeze()
			defer s.MockStreamingConnector.Reset()

			// given both lines are authorized but only the first settlement commits before failure
			fixture := s.setupPaymentRetryInvoice(chargeType)
			authorized := s.triggerBookingRetryPayment(fixture, billing.TriggerAuthorized)
			require.Equal(t, billing.StandardInvoiceStatusPaymentProcessingAuthorized, authorized.Status)
			engine := s.bookingRetryLineEngine(chargeType)
			fault := &failOncePaymentLineEngine{LineEngine: engine, SettlementLineID: fixture.Invoice.Lines.OrEmpty()[1].ID}
			s.replaceBookingRetryLineEngine(fault, engine)
			failed := s.triggerBookingRetryPayment(fixture, billing.TriggerPaid)
			require.True(t, fault.Failed)
			require.Equal(t, billing.StandardInvoiceStatusPaymentProcessingBookingSettledFailed, failed.Status)
			before := s.bookingRetryProgress(fixture)
			require.NotNil(t, before[0].Payment)
			require.Equal(t, payment.StatusSettled, before[0].Payment.Status)
			require.NotNil(t, before[1].Payment)
			require.Equal(t, payment.StatusAuthorized, before[1].Payment.Status)
			beforeIDs := s.bookingRetryTransactionIDs(fixture)
			require.Len(t, beforeIDs, 5)

			// when the settlement callback resumes from the persisted payments
			invoice, err := s.BillingService.RetryInvoice(ctx, failed.GetInvoiceID())
			require.NoError(t, err)

			// then only the remaining settlement is added and existing references survive
			s.Equal(billing.StandardInvoiceStatusPaid, invoice.Status, "issues: %+v", invoice.ValidationIssues)
			s.False(invoice.HasCriticalValidationIssues())
			after := s.bookingRetryProgress(fixture)
			s.Equal(before[0], after[0])
			for i, line := range after {
				s.Equal(before[i].RunID, line.RunID)
				s.Equal(before[i].UsageGroupID, line.UsageGroupID)
				s.Equal("final", line.Status)
				if s.NotNil(line.Payment) {
					s.Equal(before[i].Payment.ID, line.Payment.ID)
					s.Equal(before[i].Payment.Authorized, line.Payment.Authorized)
					s.Equal(payment.StatusSettled, line.Payment.Status)
					s.NotNil(line.Payment.Settled)
				}
			}
			afterIDs := s.bookingRetryTransactionIDs(fixture)
			s.Subset(afterIDs, beforeIDs)
			s.Len(afterIDs, 6)
		})
	}
}

func (s *CreditThenInvoiceTestSuite) TestCombinedPaymentBookingRetryResumesAfterAuthorization() {
	for _, chargeType := range []meta.ChargeType{meta.ChargeTypeFlatFee, meta.ChargeTypeUsageBased} {
		s.Run(string(chargeType), func() {
			t := s.T()
			ctx := t.Context()
			clock.FreezeTime(time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC))
			defer clock.UnFreeze()
			defer s.MockStreamingConnector.Reset()

			// given combined booking completes authorization before settlement fails without progress
			fixture := s.setupPaymentRetryInvoice(chargeType)
			engine := s.bookingRetryLineEngine(chargeType)
			fault := &failOncePaymentLineEngine{LineEngine: engine, SettlementLineID: fixture.Invoice.Lines.OrEmpty()[0].ID}
			s.replaceBookingRetryLineEngine(fault, engine)
			failed := s.triggerBookingRetryPayment(fixture, billing.TriggerPaid)
			require.True(t, fault.Failed)
			require.Equal(t, billing.StandardInvoiceStatusPaymentProcessingBookingAuthorizedAndSettledFailed, failed.Status)
			before := s.bookingRetryProgress(fixture)
			for _, line := range before {
				require.NotNil(t, line.Payment)
				require.Equal(t, payment.StatusAuthorized, line.Payment.Status)
				require.Nil(t, line.Payment.Settled)
			}
			beforeIDs := s.bookingRetryTransactionIDs(fixture)
			require.Len(t, beforeIDs, 4)

			// when the combined callback is retried after the settlement fault is consumed
			invoice, err := s.BillingService.RetryInvoice(ctx, failed.GetInvoiceID())
			require.NoError(t, err)

			// then settlement resumes without creating replacement authorizations
			s.Equal(billing.StandardInvoiceStatusPaid, invoice.Status, "issues: %+v", invoice.ValidationIssues)
			s.False(invoice.HasCriticalValidationIssues())
			after := s.bookingRetryProgress(fixture)
			for i, line := range after {
				s.Equal(before[i].RunID, line.RunID)
				s.Equal(before[i].UsageGroupID, line.UsageGroupID)
				s.Equal("final", line.Status)
				require.NotNil(t, line.Payment)
				s.Equal(before[i].Payment.ID, line.Payment.ID)
				s.Equal(before[i].Payment.Authorized, line.Payment.Authorized)
				s.Equal(payment.StatusSettled, line.Payment.Status)
				s.NotNil(line.Payment.Settled)
			}
			afterIDs := s.bookingRetryTransactionIDs(fixture)
			s.Subset(afterIDs, beforeIDs)
			s.Len(afterIDs, 6)
		})
	}
}

func (s *CreditThenInvoiceTestSuite) setupPaymentRetryInvoice(chargeType meta.ChargeType) chargeBookingInvoice {
	t := s.T()
	t.Helper()
	ctx := t.Context()
	var fixture chargeBookingInvoice
	if chargeType != meta.ChargeTypeCreditPurchase {
		fixture = s.setupChargeBookingInvoice(chargeType)
	} else {
		ns := s.GetUniqueNamespace("credit-purchase-payment-retry")
		defaults := s.ProvisionDefaultTaxCodes(ctx, ns)
		invoicing := s.SetupCustomInvoicing(ns)
		cust := s.CreateLedgerBackedCustomer(ns, "test-subject")
		s.ProvisionBillingProfile(ctx, ns, invoicing.App.GetID())
		period := timeutil.ClosedPeriod{From: clock.Now(), To: clock.Now().AddDate(0, 1, 0)}
		for range 2 {
			created, err := s.CreditPurchaseSvc.Create(ctx, creditpurchase.CreateInput{
				Namespace: ns,
				Intent: creditpurchase.Intent{
					Intent: meta.Intent{
						CustomerID: cust.ID,
						Currency:   currenciestestutils.NewFiatCurrency(t, USD),
						ManagedBy:  billing.SystemManagedLine,
						TaxConfig:  productcatalog.TaxCodeConfig{TaxCodeID: defaults.InvoicingTaxCodeID},
					},
					IntentMutableFields: creditpurchase.IntentMutableFields{
						IntentMutableFields: meta.IntentMutableFields{Name: "invoice-funded credits", ServicePeriod: period, FullServicePeriod: period, BillingPeriod: period},
						CreditAmount:        alpacadecimal.NewFromInt(5),
						Settlement:          creditpurchase.NewInvoiceSettlement(),
					},
					CostBasis: creditpurchase.NewCostBasis(creditpurchase.FiatCostBasis{Rate: alpacadecimal.NewFromInt(1)}),
				},
			})
			require.NoError(t, err)
			require.NotNil(t, created.GatheringLineToCreate)
			_, err = s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
				Customer: cust.GetID(), Currency: created.GatheringLineToCreate.Currency,
				Lines: billing.CreatePendingInvoiceLines{{GatheringLine: *created.GatheringLineToCreate}},
			})
			require.NoError(t, err)
		}
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{Customer: cust.GetID(), AsOf: lo.ToPtr(clock.Now())})
		require.NoError(t, err)
		require.Len(t, invoices, 1)
		fixture = chargeBookingInvoice{Customer: cust.GetID(), Invoice: invoices[0]}
		require.Len(t, fixture.Invoice.Lines.OrEmpty(), 2)
		for i, line := range fixture.Invoice.Lines.OrEmpty() {
			require.NotNil(t, line.ChargeID)
			fixture.Charges[i] = meta.ChargeID{Namespace: ns, ID: *line.ChargeID}
		}
	}
	invoice, err := s.BillingService.ApproveInvoice(ctx, fixture.Invoice.GetInvoiceID())
	require.NoError(t, err)
	require.Equal(t, billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)
	require.Equal(t, float64(10), invoice.Totals.Total.InexactFloat64())
	fixture.Invoice = invoice
	return fixture
}

func (s *CreditThenInvoiceTestSuite) bookingRetryLineEngine(chargeType meta.ChargeType) billing.LineEngine {
	s.T().Helper()
	switch chargeType {
	case meta.ChargeTypeFlatFee:
		return s.FlatFeeSvc.GetLineEngine()
	case meta.ChargeTypeUsageBased:
		return s.UsageBasedSvc.GetLineEngine()
	case meta.ChargeTypeCreditPurchase:
		return s.CreditPurchaseSvc.GetLineEngine()
	default:
		s.T().Fatalf("unexpected charge type: %s", chargeType)
		return nil
	}
}

func (s *CreditThenInvoiceTestSuite) replaceBookingRetryLineEngine(fault, original billing.LineEngine) {
	t := s.T()
	t.Helper()
	require.NoError(t, s.BillingService.DeregisterLineEngine(original.GetLineEngineType()))
	require.NoError(t, s.BillingService.RegisterLineEngine(fault))
	t.Cleanup(func() {
		require.NoError(t, s.BillingService.DeregisterLineEngine(original.GetLineEngineType()))
		require.NoError(t, s.BillingService.RegisterLineEngine(original))
	})
}

// triggerBookingRetryPayment uses the app boundary, whose validation failures
// commit prior line progress instead of rolling the enclosing transaction back.
func (s *CreditThenInvoiceTestSuite) triggerBookingRetryPayment(fixture chargeBookingInvoice, trigger billing.InvoiceTrigger) billing.StandardInvoice {
	t := s.T()
	t.Helper()
	require.NoError(t, s.BillingService.TriggerInvoice(t.Context(), billing.InvoiceTriggerServiceInput{
		InvoiceTriggerInput: billing.InvoiceTriggerInput{Invoice: fixture.Invoice.GetInvoiceID(), Trigger: trigger},
		AppType:             app.AppTypeCustomInvoicing, Capability: app.CapabilityTypeCollectPayments,
	}))
	invoice, err := s.BillingService.GetStandardInvoiceById(t.Context(), billing.GetStandardInvoiceByIdInput{Invoice: fixture.Invoice.GetInvoiceID()})
	require.NoError(t, err)
	return invoice
}

type bookingRetryLineProgress struct {
	Status       string
	RunID        string
	UsageGroupID string
	Payment      *payment.Invoiced
}

// bookingRetryProgress reloads charge facts and verifies payment references against
// the journal so retries cannot appear successful by changing invoice state alone.
func (s *CreditThenInvoiceTestSuite) bookingRetryProgress(fixture chargeBookingInvoice) [2]bookingRetryLineProgress {
	t := s.T()
	t.Helper()
	var result [2]bookingRetryLineProgress
	for i, id := range fixture.Charges {
		charge := s.MustGetChargeByID(id)
		line := &result[i]
		lineID := fixture.Invoice.Lines.OrEmpty()[i].ID
		switch charge.Type() {
		case meta.ChargeTypeFlatFee:
			value, err := charge.AsFlatFeeCharge()
			require.NoError(t, err)
			run, err := value.Realizations.GetByLineID(lineID)
			require.NoError(t, err)
			line.Status, line.RunID, line.Payment = string(value.Status), run.ID.ID, run.Payment
			if run.AccruedUsage != nil && run.AccruedUsage.LedgerTransaction != nil {
				line.UsageGroupID = run.AccruedUsage.LedgerTransaction.TransactionGroupID
			}
		case meta.ChargeTypeUsageBased:
			value, err := charge.AsUsageBasedCharge()
			require.NoError(t, err)
			run, err := value.Realizations.GetByLineID(lineID)
			require.NoError(t, err)
			line.Status, line.RunID, line.Payment = string(value.Status), run.ID.ID, run.Payment
			if run.InvoiceUsage != nil && run.InvoiceUsage.LedgerTransaction != nil {
				line.UsageGroupID = run.InvoiceUsage.LedgerTransaction.TransactionGroupID
			}
		case meta.ChargeTypeCreditPurchase:
			value, err := charge.AsCreditPurchaseCharge()
			require.NoError(t, err)
			line.Status, line.Payment = string(value.Status), value.Realizations.InvoiceSettlement
		}
		if line.Payment == nil {
			continue
		}
		require.Equal(t, fixture.Invoice.ID, line.Payment.InvoiceID)
		require.Equal(t, lineID, line.Payment.LineID)
		require.Equal(t, float64(5), line.Payment.FiatAmount.InexactFloat64())
		require.Nil(t, line.Payment.DeletedAt)
		require.NotNil(t, line.Payment.Authorized)
		groupIDs := []string{line.Payment.Authorized.TransactionGroupID}
		if line.Payment.Settled != nil {
			groupIDs = append(groupIDs, line.Payment.Settled.TransactionGroupID)
		}
		require.Len(t, lo.Uniq(groupIDs), len(groupIDs))
		for groupIndex, groupID := range groupIDs {
			require.NotEmpty(t, groupID)
			group, err := s.Ledger.GetTransactionGroup(t.Context(), models.NamespacedID{Namespace: id.Namespace, ID: groupID})
			require.NoError(t, err)
			require.Len(t, group.Transactions(), 1)
			transaction := group.Transactions()[0]
			require.Equal(t, group.ID(), transaction.GroupID())
			bookedAt := line.Payment.Authorized.Time
			expectedPostings := map[string]float64{"customer_receivable/open": 5, "customer_receivable/authorized": -5}
			if groupIndex == 1 {
				bookedAt = line.Payment.Settled.Time
				expectedPostings = map[string]float64{"customer_receivable/authorized": 5, "wash": -5}
			}
			require.True(t, bookedAt.Equal(transaction.BookedAt()))
			require.Len(t, transaction.Entries(), 2)
			postings := make(map[string]float64)
			for _, entry := range transaction.Entries() {
				chargeID := entry.Provenance().SpendChargeID
				if charge.Type() == meta.ChargeTypeCreditPurchase {
					chargeID = entry.Provenance().SourceChargeID
				}
				require.Equal(t, lo.ToPtr(id.ID), chargeID)
				address := entry.PostingAddress()
				route := address.Route().Route()
				require.Equal(t, USD, route.Currency.GetCode())
				key := string(address.AccountType())
				if route.TransactionAuthorizationStatus != nil {
					key += "/" + string(*route.TransactionAuthorizationStatus)
				}
				postings[key] += entry.Amount().InexactFloat64()
			}
			require.Equal(t, expectedPostings, postings)
		}
	}
	return result
}

func (s *CreditThenInvoiceTestSuite) bookingRetryTransactionIDs(fixture chargeBookingInvoice) []string {
	t := s.T()
	t.Helper()
	transactions, err := s.Ledger.ListTransactions(t.Context(), ledger.ListTransactionsInput{Namespace: fixture.Customer.Namespace, Limit: 100})
	require.NoError(t, err)
	require.Nil(t, transactions.NextCursor)
	return lo.Map(transactions.Items, func(tx ledger.Transaction, _ int) string { return tx.ID().ID })
}

type failOncePaymentLineEngine struct {
	billing.LineEngine
	AuthorizationLineID string
	SettlementLineID    string
	Failed              bool
}

func (e *failOncePaymentLineEngine) OnPaymentAuthorized(ctx context.Context, input billing.OnPaymentAuthorizedInput) error {
	if e.AuthorizationLineID == "" || e.Failed {
		return e.LineEngine.OnPaymentAuthorized(ctx, input)
	}
	for _, line := range input.Lines {
		if line.ID == e.AuthorizationLineID {
			e.Failed = true
			return bookingRetryCallbackFailure()
		}
		one := input
		one.Lines = billing.StandardLines{line}
		if err := e.LineEngine.OnPaymentAuthorized(ctx, one); err != nil {
			return err
		}
	}
	return nil
}

func (e *failOncePaymentLineEngine) OnPaymentSettled(ctx context.Context, input billing.OnPaymentSettledInput) error {
	if e.SettlementLineID == "" || e.Failed {
		return e.LineEngine.OnPaymentSettled(ctx, input)
	}
	for _, line := range input.Lines {
		if line.ID == e.SettlementLineID {
			e.Failed = true
			return bookingRetryCallbackFailure()
		}
		one := input
		one.Lines = billing.StandardLines{line}
		if err := e.LineEngine.OnPaymentSettled(ctx, one); err != nil {
			return err
		}
	}
	return nil
}

type failOnceFinalizingLineEngine struct {
	billing.LineEngine
	FailLineID string
	Failed     bool
}

func (e *failOnceFinalizingLineEngine) OnInvoiceFinalizing(ctx context.Context, input billing.OnInvoiceFinalizingInput) (billing.StandardLines, error) {
	if e.Failed {
		return e.LineEngine.OnInvoiceFinalizing(ctx, input)
	}
	var prepared billing.StandardLines
	for _, line := range input.Lines {
		if line.ID == e.FailLineID {
			e.Failed = true
			return nil, bookingRetryCallbackFailure()
		}
		one := input
		one.Lines = billing.StandardLines{line}
		lines, err := e.LineEngine.OnInvoiceFinalizing(ctx, one)
		if err != nil {
			return nil, err
		}
		prepared = append(prepared, lines...)
	}
	return prepared, nil
}

func bookingRetryCallbackFailure() error {
	return billing.ValidationIssue{
		Severity: billing.ValidationIssueSeverityCritical,
		Code:     "test_transient_line_callback_failed",
		Message:  "transient line callback failure after prior progress committed",
	}
}
