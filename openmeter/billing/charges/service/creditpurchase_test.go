package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

type CreditPurchaseTestSuite struct {
	BaseSuite
}

func TestCreditPurchase(t *testing.T) {
	suite.Run(t, new(CreditPurchaseTestSuite))
}

func (s *CreditPurchaseTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()
}

func (s *CreditPurchaseTestSuite) TearDownTest() {
	s.BaseSuite.TearDownTest()
}

func (s *CreditPurchaseTestSuite) TestPromotionalCreditPurchase() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("charges-service-promotional-credit-purchase")

	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	intent := CreateCreditPurchaseIntent(s.T(),
		createCreditPurchaseIntentInput{
			customer: cust.GetID(),
			currency: USD,
			amount:   alpacadecimal.NewFromFloat(100),
			servicePeriod: timeutil.ClosedPeriod{
				From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
				To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
			},
			settlement: creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{}),
		},
	)

	promotionalCallback := newCountedLedgerTransactionCallback[creditpurchase.Charge]()
	s.CreditPurchaseTestHandler.onPromotionalCreditPurchase = promotionalCallback.Handler(s.T(), func(t *testing.T, charge creditpurchase.Charge) {
		assert.Equal(t, charge.Intent.Settlement.Type(), creditpurchase.SettlementTypePromotional)
		assert.Nil(t, charge.State.CreditGrantRealization, "credit grant realization should not be set")
		assert.Nil(t, charge.State.ExternalPaymentSettlement, "external payment settlement should not be set")
	})

	res, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			intent,
		},
	})
	s.NoError(err)
	s.Len(res, 1)
	s.Equal(meta.ChargeTypeCreditPurchase, res[0].Type())

	s.Equal(1, promotionalCallback.nrInvocations)
	cpCharge, err := res[0].AsCreditPurchaseCharge()
	s.NoError(err)
	s.NotNil(cpCharge.State.CreditGrantRealization)
	s.Equal(promotionalCallback.id, cpCharge.State.CreditGrantRealization.GroupReference.TransactionGroupID)
	s.Equal(meta.ChargeStatusFinal, cpCharge.Status)

	charge := s.mustGetChargeByID(cpCharge.GetChargeID())
	updatedCPCharge, err := charge.AsCreditPurchaseCharge()
	s.NoError(err)
	s.Equal(promotionalCallback.id, updatedCPCharge.State.CreditGrantRealization.GroupReference.TransactionGroupID)
	s.Equal(meta.ChargeStatusFinal, updatedCPCharge.Status)
}

type createCreditPurchaseIntentInput struct {
	customer      customer.CustomerID
	currency      currencyx.Code
	amount        alpacadecimal.Decimal
	servicePeriod timeutil.ClosedPeriod
	settlement    creditpurchase.Settlement
}

func (i createCreditPurchaseIntentInput) Validate() error {
	if err := i.customer.Validate(); err != nil {
		return fmt.Errorf("customer: %w", err)
	}

	if i.currency == "" {
		return errors.New("currency is required")
	}

	if !i.amount.IsPositive() {
		return errors.New("amount must be positive")
	}

	if err := i.servicePeriod.Validate(); err != nil {
		return fmt.Errorf("service period: %w", err)
	}

	if err := i.settlement.Validate(); err != nil {
		return fmt.Errorf("settlement: %w", err)
	}

	return nil
}

func CreateCreditPurchaseIntent(t *testing.T, input createCreditPurchaseIntentInput) charges.ChargeIntent {
	t.Helper()
	require.NoError(t, input.Validate())

	return charges.NewChargeIntent(creditpurchase.Intent{
		Intent: meta.Intent{
			Name:              "Credit Purchase",
			ManagedBy:         billing.ManuallyManagedLine,
			CustomerID:        input.customer.ID,
			Currency:          input.currency,
			ServicePeriod:     input.servicePeriod,
			BillingPeriod:     input.servicePeriod,
			FullServicePeriod: input.servicePeriod,
		},
		CreditAmount: input.amount,
		Settlement:   input.settlement,
	})
}

func (s *CreditPurchaseTestSuite) TestExternalAuthorizedCreditPurchaseAutoSettled() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("charges-service-external-authorized-credit-purchase-auto-settled")

	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	// Let's buy 100 USD credits for $0.50 each (total cost is $50)
	intent := CreateCreditPurchaseIntent(s.T(),
		createCreditPurchaseIntentInput{
			customer: cust.GetID(),
			currency: USD,
			amount:   alpacadecimal.NewFromFloat(100),
			servicePeriod: timeutil.ClosedPeriod{
				From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
				To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
			},
			settlement: creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{
				InitialStatus: creditpurchase.SettledInitialPaymentSettlementStatus,
				GenericSettlement: creditpurchase.GenericSettlement{
					Currency:  USD,
					CostBasis: alpacadecimal.NewFromFloat(0.5),
				},
			}),
		},
	)

	// First the initiated callback should be called, without any grant realizations or payment settlements
	initiatedCallback := newCountedLedgerTransactionCallback[creditpurchase.Charge]()
	s.CreditPurchaseTestHandler.onCreditPurchaseInitiated = initiatedCallback.Handler(s.T(), func(t *testing.T, charge creditpurchase.Charge) {
		assert.Equal(t, charge.Intent.Settlement.Type(), creditpurchase.SettlementTypeExternal)
		assert.Nil(t, charge.State.CreditGrantRealization, "credit grant realization should not be set")
		assert.Nil(t, charge.State.ExternalPaymentSettlement, "external payment settlement should not be set")
	})

	// Then the authorized callback should be called, with a grant realization and no payment settlement
	authorizedCallback := newCountedLedgerTransactionCallback[creditpurchase.Charge]()
	s.CreditPurchaseTestHandler.onCreditPurchasePaymentAuthorized = authorizedCallback.Handler(s.T(), func(t *testing.T, charge creditpurchase.Charge) {
		assert.Equal(t, charge.Intent.Settlement.Type(), creditpurchase.SettlementTypeExternal)
		assert.NotNil(t, charge.State.CreditGrantRealization, "credit grant realization should be set")
		assert.Equal(t, initiatedCallback.id, charge.State.CreditGrantRealization.GroupReference.TransactionGroupID)
		assert.Nil(t, charge.State.ExternalPaymentSettlement)
		assert.Equal(t, meta.ChargeStatusActive, charge.Status, "charge status should be active")
	})

	// Then the settled callback should be called, with a grant realization and a payment settlement
	settledCallback := newCountedLedgerTransactionCallback[creditpurchase.Charge]()
	s.CreditPurchaseTestHandler.onCreditPurchasePaymentSettled = settledCallback.Handler(s.T(), func(t *testing.T, charge creditpurchase.Charge) {
		assert.Equal(t, charge.Intent.Settlement.Type(), creditpurchase.SettlementTypeExternal)
		assert.NotNil(t, charge.State.ExternalPaymentSettlement, "external payment settlement should be set")

		// Authorized transaction group ID should be set
		assert.Equal(t, authorizedCallback.id, charge.State.ExternalPaymentSettlement.Authorized.TransactionGroupID)
		assert.Equal(t, payment.StatusAuthorized, charge.State.ExternalPaymentSettlement.Status)
		assert.Equal(t, meta.ChargeStatusActive, charge.Status, "charge status should be active")
	})
	res, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			intent,
		},
	})
	s.NoError(err)
	s.Len(res, 1)
	s.Equal(meta.ChargeTypeCreditPurchase, res[0].Type())

	// All callback should have been invoked only once
	s.Equal(1, initiatedCallback.nrInvocations)
	s.Equal(1, authorizedCallback.nrInvocations)
	s.Equal(1, settledCallback.nrInvocations)

	dbCharge := s.mustGetChargeByID(lo.Must(res[0].GetChargeID()))

	// Let's validate both the output from the Create and the DB state
	for _, tc := range []struct {
		name   string
		charge charges.Charge
	}{
		{name: "output", charge: res[0]},
		{name: "db", charge: dbCharge},
	} {
		s.Run(tc.name, func() {
			// The charge should have a grant realization and a payment settlement
			creditPurchaseCharge, err := tc.charge.AsCreditPurchaseCharge()
			s.NoError(err)
			// Credit grant realization should be set
			s.NotNil(creditPurchaseCharge.State.CreditGrantRealization)
			s.Equal(initiatedCallback.id, creditPurchaseCharge.State.CreditGrantRealization.GroupReference.TransactionGroupID)

			// Payment settlement should be set
			s.NotNil(creditPurchaseCharge.State.ExternalPaymentSettlement, "external payment settlement should be set")
			s.Equal(authorizedCallback.id, creditPurchaseCharge.State.ExternalPaymentSettlement.Authorized.TransactionGroupID, "authorized transaction group ID should be set")
			s.Equal(settledCallback.id, creditPurchaseCharge.State.ExternalPaymentSettlement.Settled.TransactionGroupID, "settled transaction group ID should be set")

			// The charge should be final
			s.Equal(meta.ChargeStatusFinal, creditPurchaseCharge.Status)
		})
	}
}

func (s *CreditPurchaseTestSuite) TestExternalAuthorizedCreditPurchaseManuallySettled() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("charges-service-external-authorized-credit-purchase-manually-settled")

	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	// Let's buy 100 USD credits for $0.50 each (total cost is $50)
	intent := CreateCreditPurchaseIntent(s.T(),
		createCreditPurchaseIntentInput{
			customer: cust.GetID(),
			currency: USD,
			amount:   alpacadecimal.NewFromFloat(100),
			servicePeriod: timeutil.ClosedPeriod{
				From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
				To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
			},
			settlement: creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{
				InitialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus,
				GenericSettlement: creditpurchase.GenericSettlement{
					Currency:  USD,
					CostBasis: alpacadecimal.NewFromFloat(0.5),
				},
			}),
		},
	)

	var chargeID meta.ChargeID
	var initatedTrnsID string

	s.Run("initiated", func() {
		defer s.CreditPurchaseTestHandler.Reset()

		// First the initiated callback should be called, without any grant realizations or payment settlements
		initatedCallback := newCountedLedgerTransactionCallback[creditpurchase.Charge]()
		s.CreditPurchaseTestHandler.onCreditPurchaseInitiated = initatedCallback.Handler(s.T(), func(t *testing.T, charge creditpurchase.Charge) {
			assert.Equal(t, charge.Intent.Settlement.Type(), creditpurchase.SettlementTypeExternal)
			assert.Nil(t, charge.State.CreditGrantRealization, "credit grant realization should not be set")
			assert.Nil(t, charge.State.ExternalPaymentSettlement, "external payment settlement should not be set")
		})

		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				intent,
			},
		})
		s.NoError(err)
		s.Len(res, 1)
		s.Equal(meta.ChargeTypeCreditPurchase, res[0].Type())

		creditPurchaseCharge, err := res[0].AsCreditPurchaseCharge()
		s.NoError(err)
		s.Equal(1, initatedCallback.nrInvocations)
		s.Equal(initatedCallback.id, creditPurchaseCharge.State.CreditGrantRealization.GroupReference.TransactionGroupID)
		s.Equal(meta.ChargeStatusActive, creditPurchaseCharge.Status)

		chargeID = creditPurchaseCharge.GetChargeID()
		initatedTrnsID = initatedCallback.id
	})

	var authorizedTrnsID string
	s.Run("authorized", func() {
		defer s.CreditPurchaseTestHandler.Reset()

		// Then the authorized callback should be called, with a grant realization and no payment settlement
		authorizedCallback := newCountedLedgerTransactionCallback[creditpurchase.Charge]()
		s.CreditPurchaseTestHandler.onCreditPurchasePaymentAuthorized = authorizedCallback.Handler(s.T(), func(t *testing.T, charge creditpurchase.Charge) {
			assert.Equal(t, charge.Intent.Settlement.Type(), creditpurchase.SettlementTypeExternal)
			assert.NotNil(t, charge.State.CreditGrantRealization, "credit grant realization should be set")
			assert.Equal(t, initatedTrnsID, charge.State.CreditGrantRealization.GroupReference.TransactionGroupID)
			assert.Nil(t, charge.State.ExternalPaymentSettlement)
			assert.Equal(t, meta.ChargeStatusActive, charge.Status, "charge status should be active")
		})

		res, err := s.Charges.HandleCreditPurchaseExternalPaymentStateTransition(ctx, charges.HandleCreditPurchaseExternalPaymentStateTransitionInput{
			ChargeID:           chargeID,
			TargetPaymentState: payment.StatusAuthorized,
		})
		s.NoError(err)

		s.Equal(1, authorizedCallback.nrInvocations)
		s.Equal(authorizedCallback.id, res.State.ExternalPaymentSettlement.Authorized.TransactionGroupID)
		s.Equal(payment.StatusAuthorized, res.State.ExternalPaymentSettlement.Status)
		s.Equal(meta.ChargeStatusActive, res.Status)

		authorizedTrnsID = authorizedCallback.id
	})

	s.Run("settled", func() {
		defer s.CreditPurchaseTestHandler.Reset()

		// Then the settled callback should be called, with a grant realization and a payment settlement
		settledCallback := newCountedLedgerTransactionCallback[creditpurchase.Charge]()
		s.CreditPurchaseTestHandler.onCreditPurchasePaymentSettled = settledCallback.Handler(s.T(), func(t *testing.T, charge creditpurchase.Charge) {
			assert.Equal(t, charge.Intent.Settlement.Type(), creditpurchase.SettlementTypeExternal)
			assert.NotNil(t, charge.State.ExternalPaymentSettlement, "external payment settlement should be set")

			// Authorized transaction group ID should be set
			assert.Equal(t, authorizedTrnsID, charge.State.ExternalPaymentSettlement.Authorized.TransactionGroupID)
			assert.Equal(t, payment.StatusAuthorized, charge.State.ExternalPaymentSettlement.Status)
			assert.Equal(t, meta.ChargeStatusActive, charge.Status, "charge status should be active")
		})
		res, err := s.Charges.HandleCreditPurchaseExternalPaymentStateTransition(ctx, charges.HandleCreditPurchaseExternalPaymentStateTransitionInput{
			ChargeID:           chargeID,
			TargetPaymentState: payment.StatusSettled,
		})
		s.NoError(err)

		s.Equal(1, settledCallback.nrInvocations)
		s.Equal(settledCallback.id, res.State.ExternalPaymentSettlement.Settled.TransactionGroupID)
		s.Equal(payment.StatusSettled, res.State.ExternalPaymentSettlement.Status)
		s.Equal(meta.ChargeStatusFinal, res.Status)
	})
}

func (s *CreditPurchaseTestSuite) TestStandardInvoiceCreditPurchase() {
	defer clock.UnFreeze()
	ctx := context.Background()
	ns := s.GetUniqueNamespace("charges-service-standard-invoice-credit-purchase")

	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	customInvoicing := s.SetupCustomInvoicing(ns)

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithProgressiveBilling(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "PT1H")),
		billingtest.WithManualApproval(),
	)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	clock.SetTime(datetime.MustParseTimeInLocation(s.T(), "2025-12-01T00:00:00Z", time.UTC).AsTime())

	// Let's buy 100 USD credits for $0.50 each (total cost is $50)
	intent := CreateCreditPurchaseIntent(s.T(),
		createCreditPurchaseIntentInput{
			customer:      cust.GetID(),
			currency:      USD,
			amount:        alpacadecimal.NewFromFloat(100),
			servicePeriod: servicePeriod,
			settlement: creditpurchase.NewSettlement(creditpurchase.InvoiceSettlement{
				GenericSettlement: creditpurchase.GenericSettlement{
					Currency:  USD,
					CostBasis: alpacadecimal.NewFromFloat(0.5),
				},
			}),
		},
	)

	var chargeID meta.ChargeID
	var initatedTrnsID string

	var invoiceID billing.InvoiceID

	s.Run("initiated", func() {
		defer s.CreditPurchaseTestHandler.Reset()

		// First the initiated callback should be called, without any grant realizations or payment settlements

		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				intent,
			},
		})
		s.NoError(err)
		s.Len(res, 1)
		s.Equal(meta.ChargeTypeCreditPurchase, res[0].Type())
		chargeID, err = res[0].GetChargeID()
		s.NoError(err)

		charge := s.mustGetChargeByID(chargeID)
		creditPurchaseCharge, err := charge.AsCreditPurchaseCharge()
		s.NoError(err)
		s.Equal(meta.ChargeStatusCreated, creditPurchaseCharge.Status)
		s.Nil(creditPurchaseCharge.State.CreditGrantRealization)
		s.Nil(creditPurchaseCharge.State.InvoiceSettlement)
	})

	s.Run("invoice pending lines", func() {
		defer s.CreditPurchaseTestHandler.Reset()

		initatedCallback := newCountedLedgerTransactionCallback[creditpurchase.Charge]()
		s.CreditPurchaseTestHandler.onCreditPurchaseInitiated = initatedCallback.Handler(s.T(), func(t *testing.T, charge creditpurchase.Charge) {
			assert.Equal(t, charge.Intent.Settlement.Type(), creditpurchase.SettlementTypeInvoice)
			assert.Nil(t, charge.State.CreditGrantRealization, "credit grant realization should not be set")
			assert.Nil(t, charge.State.InvoiceSettlement, "invoice settlement should not be set")
		})

		clock.FreezeTime(datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime())
		now := clock.Now()
		createdInvoices, err := s.Charges.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     &now,
		})

		s.NoError(err)
		s.Len(createdInvoices, 1)
		s.Len(createdInvoices[0].Lines.OrEmpty(), 1)
		invoiceID = createdInvoices[0].GetInvoiceID()
		s.NotEmpty(invoiceID)
		s.NoError(invoiceID.Validate())

		invoicesResult, err := s.BillingService.ListStandardInvoices(ctx, billing.ListStandardInvoicesInput{
			Namespaces: []string{ns},
		})

		s.NoError(err)
		s.Len(invoicesResult.Items, 1)

		updatedCharge := s.mustGetChargeByID(chargeID)

		creditPurchaseCharge, err := updatedCharge.AsCreditPurchaseCharge()
		s.NoError(err)

		s.Equal(1, initatedCallback.nrInvocations)
		s.Equal(initatedCallback.id, creditPurchaseCharge.State.CreditGrantRealization.GroupReference.TransactionGroupID)
		s.Equal(meta.ChargeStatusActive, creditPurchaseCharge.Status)

		chargeID = creditPurchaseCharge.GetChargeID()
		initatedTrnsID = initatedCallback.id

		s.NotEmpty(invoiceID)
	})

	var authorizedTrnsID string
	s.Run("authorized", func() {
		defer s.CreditPurchaseTestHandler.Reset()

		// Then the authorized callback should be called, with a grant realization and no payment settlement
		authorizedCallback := newCountedLedgerTransactionCallback[creditpurchase.Charge]()
		s.CreditPurchaseTestHandler.onCreditPurchasePaymentAuthorized = authorizedCallback.Handler(s.T(), func(t *testing.T, charge creditpurchase.Charge) {
			assert.Equal(t, charge.Intent.Settlement.Type(), creditpurchase.SettlementTypeInvoice)
			assert.NotNil(t, charge.State.CreditGrantRealization, "credit grant realization should be set")
			assert.Equal(t, initatedTrnsID, charge.State.CreditGrantRealization.GroupReference.TransactionGroupID)
			assert.Nil(t, charge.State.InvoiceSettlement)
			assert.Equal(t, meta.ChargeStatusActive, charge.Status, "charge status should be active")
		})

		invoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
			Invoice: invoiceID,
			Expand:  billing.StandardInvoiceExpandAll,
		})
		s.NoError(err)
		s.Equal(invoiceID, invoice.GetInvoiceID())

		s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, invoice.Status)

		res, err := s.BillingService.ApproveInvoice(ctx, invoiceID)
		s.NoError(err)

		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, res.Status)

		invoice, err = s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
			Invoice: invoiceID,
			Expand:  billing.StandardInvoiceExpandAll,
		})
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)

		charge := s.mustGetChargeByID(chargeID)
		creditPurchaseCharge, err := charge.AsCreditPurchaseCharge()
		s.NoError(err)

		// Invoice settlement should be set
		s.NotNil(creditPurchaseCharge.State.InvoiceSettlement)
		lineID := creditPurchaseCharge.State.InvoiceSettlement.LineID
		s.NotEmpty(lineID)

		s.Equal(1, authorizedCallback.nrInvocations)
		s.NotNil(creditPurchaseCharge.State.InvoiceSettlement)
		s.NotNil(creditPurchaseCharge.State.InvoiceSettlement.Authorized)
		s.Equal(authorizedCallback.id, creditPurchaseCharge.State.InvoiceSettlement.Authorized.TransactionGroupID)
		s.Equal(meta.ChargeStatusActive, creditPurchaseCharge.Status, "charge status should be active")

		// validate the standard line
		lines := invoice.Lines.OrEmpty()
		s.Require().Len(lines, 1)

		line := lines[0]
		s.Equal(lineID, line.ID)
		s.Equal(USD, line.Currency)
		s.Equal(billing.Period{
			Start: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
			End:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
		}, line.Period)
		s.Equal(alpacadecimal.NewFromFloat(50), line.Totals.Amount)
		s.Equal(alpacadecimal.NewFromFloat(50), line.Totals.Total)

		// validate the detailed line
		s.Require().Len(line.DetailedLines, 1)

		detailedLine := line.DetailedLines[0]

		s.Equal(USD, detailedLine.Currency)
		s.Equal(alpacadecimal.NewFromFloat(50), detailedLine.PerUnitAmount)
		s.Equal(alpacadecimal.NewFromFloat(1), detailedLine.Quantity)
		s.Equal(alpacadecimal.NewFromFloat(50), detailedLine.Totals.Amount)
		s.Equal(alpacadecimal.NewFromFloat(50), detailedLine.Totals.Total)

		// validate invoice totals
		s.Equal(alpacadecimal.NewFromFloat(50), invoice.Totals.Amount)
		s.Equal(alpacadecimal.NewFromFloat(50), invoice.Totals.Total)

		authorizedTrnsID = authorizedCallback.id
	})

	s.Run("settled", func() {
		defer s.CreditPurchaseTestHandler.Reset()
		// Then the settled callback should be called, with a grant realization and a payment settlement
		settledCallback := newCountedLedgerTransactionCallback[creditpurchase.Charge]()
		s.CreditPurchaseTestHandler.onCreditPurchasePaymentSettled = settledCallback.Handler(s.T(), func(t *testing.T, charge creditpurchase.Charge) {
			assert.Equal(t, charge.Intent.Settlement.Type(), creditpurchase.SettlementTypeInvoice)
			assert.NotNil(t, charge.State.InvoiceSettlement, "invoice settlement should be set")

			// Authorized transaction group ID should still be set from the authorized phase
			assert.Equal(t, authorizedTrnsID, charge.State.InvoiceSettlement.Authorized.TransactionGroupID)
			assert.Equal(t, meta.ChargeStatusActive, charge.Status, "charge status should be active")
		})

		// First verify the invoice is in the expected state
		invoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
			Invoice: invoiceID,
		})

		s.NoError(err)
		s.Equal(invoiceID, invoice.GetInvoiceID())
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)

		res, err := s.CustomInvoicingService.HandlePaymentTrigger(ctx, appcustominvoicing.HandlePaymentTriggerInput{
			InvoiceID: invoiceID,
			Trigger:   billing.TriggerPaid,
		})

		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaid, res.Status)

		s.Equal(1, settledCallback.nrInvocations)

		charge := s.mustGetChargeByID(chargeID)
		creditPurchaseCharge, err := charge.AsCreditPurchaseCharge()
		s.NoError(err)

		s.Equal(settledCallback.id, creditPurchaseCharge.State.InvoiceSettlement.Settled.TransactionGroupID)
		s.Equal(payment.StatusSettled, creditPurchaseCharge.State.InvoiceSettlement.Status)
		s.Equal(meta.ChargeStatusFinal, creditPurchaseCharge.Status)
	})
}

func (s *CreditPurchaseTestSuite) TestStandardInvoiceCreditPurchaseDeferred() {
	// This test exercises the deferred invoicing path where InvoiceAt is in the future.
	// In this case, InvoiceSettlement remains nil at Create() time.
	// The gathering line is created but not immediately invoiced.
	ctx := context.Background()
	ns := s.GetUniqueNamespace("charges-service-standard-invoice-credit-purchase-deferred")

	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	customInvoicing := s.SetupCustomInvoicing(ns)

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithProgressiveBilling(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "PT1H")),
		billingtest.WithManualApproval(),
	)

	// Let's buy 100 USD credits for $0.50 each (total cost is $50)
	// Service period is in the FUTURE to exercise the deferred invoicing path
	intent := CreateCreditPurchaseIntent(s.T(),
		createCreditPurchaseIntentInput{
			customer: cust.GetID(),
			currency: USD,
			amount:   alpacadecimal.NewFromFloat(100),
			servicePeriod: timeutil.ClosedPeriod{
				From: datetime.MustParseTimeInLocation(s.T(), "2037-01-01T00:00:00Z", time.UTC).AsTime(),
				To:   datetime.MustParseTimeInLocation(s.T(), "2037-02-01T00:00:00Z", time.UTC).AsTime(),
			},
			settlement: creditpurchase.NewSettlement(creditpurchase.InvoiceSettlement{
				GenericSettlement: creditpurchase.GenericSettlement{
					Currency:  USD,
					CostBasis: alpacadecimal.NewFromFloat(0.5),
				},
			}),
		},
	)

	var chargeID meta.ChargeID

	s.Run("initiated", func() {
		defer s.CreditPurchaseTestHandler.Reset()

		// First the initiated callback should be called, without any grant realizations or payment settlements
		initatedCallback := newCountedLedgerTransactionCallback[creditpurchase.Charge]()
		s.CreditPurchaseTestHandler.onCreditPurchaseInitiated = initatedCallback.Handler(s.T(), func(t *testing.T, charge creditpurchase.Charge) {
			assert.Equal(t, charge.Intent.Settlement.Type(), creditpurchase.SettlementTypeInvoice)
			assert.Nil(t, charge.State.CreditGrantRealization, "credit grant realization should not be set")
			assert.Nil(t, charge.State.InvoiceSettlement, "invoice settlement should not be set for deferred invoicing")
		})

		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				intent,
			},
		})
		s.NoError(err)
		s.Len(res, 1)
		s.Equal(meta.ChargeTypeCreditPurchase, res[0].Type())

		creditPurchaseCharge, err := res[0].AsCreditPurchaseCharge()
		s.NoError(err)
		s.Equal(1, initatedCallback.nrInvocations)
		s.Equal(initatedCallback.id, creditPurchaseCharge.State.CreditGrantRealization.GroupReference.TransactionGroupID)
		s.Equal(meta.ChargeStatusActive, creditPurchaseCharge.Status)

		// For deferred invoicing, InvoiceSettlement should be nil at this point
		// because the invoice hasn't been created yet (InvoiceAt is in the future)
		assert.Nil(s.T(), creditPurchaseCharge.State.InvoiceSettlement, "invoice settlement should be nil for deferred invoicing")

		chargeID = creditPurchaseCharge.GetChargeID()
	})

	s.Run("gathering_line_created", func() {
		// Verify that the gathering line was created but not yet invoiced
		// In production, InvoicePendingLines would be called when InvoiceAt arrives
		charge := s.mustGetChargeByID(chargeID)
		creditPurchaseCharge, err := charge.AsCreditPurchaseCharge()
		s.NoError(err)

		// InvoiceSettlement should still be nil because the gathering line
		// has InvoiceAt in the future and hasn't been invoiced yet
		assert.Nil(s.T(), creditPurchaseCharge.State.InvoiceSettlement, "invoice settlement should remain nil until invoicing")

		// The charge should be active and ready for when the invoice date arrives
		assert.Equal(s.T(), meta.ChargeStatusActive, creditPurchaseCharge.Status)
	})

	// Note: We cannot test the full deferred invoicing flow in a unit test because
	// InvoicePendingLines requires AsOf to be in the past. In production, the deferred
	// path would work as follows:
	// 1. When InvoiceAt arrives (2027-01-01), a background job calls InvoicePendingLines
	// 2. The gathering line is converted to a standard invoice
	// 3. InvoiceSettlement is populated via LinkInvoicedPayment
	// 4. The invoice goes through the normal approve/paid flow
	//
	// The TestStandardInvoiceCreditPurchase test covers steps 2-4 with a past-dated invoice,
	// verifying that InvoiceSettlement is correctly populated and the callbacks fire properly.
}
