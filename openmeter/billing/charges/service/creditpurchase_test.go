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

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/timeutil"
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

func (s *CreditPurchaseTestSuite) TeardownTest() {
	s.BaseSuite.TeardownTest()
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

		// Invoice settlement should be set
		s.NotNil(creditPurchaseCharge.State.InvoiceSettlement)
		lineID := creditPurchaseCharge.State.InvoiceSettlement.LineID
		s.NotEmpty(lineID)

		invoiceID = billing.InvoiceID{
			Namespace: ns,
			ID:        creditPurchaseCharge.State.InvoiceSettlement.InvoiceID,
		}
		s.NotEmpty(invoiceID)

		invoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
			Invoice: invoiceID,
			Expand:  billing.StandardInvoiceExpandAll,
		})
		s.NoError(err)
		s.Equal(invoiceID, invoice.GetInvoiceID())

		s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, invoice.Status)
		// TODO[mark]:
		// - validate the standard line  invoice.Lines
		// - validate the detailed line invoice.Lines.DetailedLines
		// - validate totals (invoice, lines, detailed lines)
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
			assert.NotNil(t, charge.State.InvoiceSettlement)
			assert.Equal(t, invoiceID, charge.State.InvoiceSettlement.InvoiceID)
			assert.Equal(t, meta.ChargeStatusActive, charge.Status, "charge status should be active")
		})

		res, err := s.BillingService.ApproveInvoice(ctx, invoiceID)
		s.NoError(err)

		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, res.Status)

		charge := s.mustGetChargeByID(chargeID)
		creditPurchaseCharge, err := charge.AsCreditPurchaseCharge()
		s.NoError(err)

		s.Equal(1, authorizedCallback.nrInvocations)
		s.Equal(authorizedCallback.id, creditPurchaseCharge.State.CreditGrantRealization.GroupReference.TransactionGroupID)
		s.Equal(meta.ChargeStatusActive, creditPurchaseCharge.Status, "charge status should be active")
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

		res, err := customInvoicing.Service.HandlePaymentTrigger(ctx, appcustominvoicing.HandlePaymentTriggerInput{
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
		s.Equal(meta.ChargeStatusFinal, res.Status)
	})
}
