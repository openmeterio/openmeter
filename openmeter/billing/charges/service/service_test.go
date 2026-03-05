package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/adapter"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

const USD = currencyx.Code(currency.USD)

type ChargesServiceTestSuite struct {
	billingtest.BaseSuite

	Charges                   *service
	FlatFeeTestHandler        *flatFeeTestHandler
	CreditPurchaseTestHandler *creditPurchaseTestHandler
}

func TestChargesService(t *testing.T) {
	suite.Run(t, new(ChargesServiceTestSuite))
}

func (s *ChargesServiceTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()

	chargesAdapter, err := adapter.New(adapter.Config{
		Client: s.DBClient,
		Logger: slog.Default(),
	})
	s.NoError(err)

	s.FlatFeeTestHandler = newFlatFeeTestHandler()
	s.CreditPurchaseTestHandler = newCreditPurchaseTestHandler()

	chargesService, err := New(Config{
		Adapter:        chargesAdapter,
		BillingService: s.BillingService,
		Handlers: Handlers{
			FlatFee:        s.FlatFeeTestHandler,
			CreditPurchase: s.CreditPurchaseTestHandler,
		},
	})
	s.NoError(err)
	s.Charges = chargesService
}

func (s *ChargesServiceTestSuite) TeardownTest() {
	s.FlatFeeTestHandler.Reset()
	s.CreditPurchaseTestHandler.Reset()
}

func (s *ChargesServiceTestSuite) TestFlatFeePartialCreditRealizations() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("charges-service-flatfee-partial-credit-realizations")

	customInvoicing := s.SetupCustomInvoicing(ns)

	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithProgressiveBilling(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "PT1H")),
		billingtest.WithManualApproval(),
	)

	const (
		flatFeeName = "flat-fee"
	)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	clock.SetTime(servicePeriod.From)

	flatFeeChargeID := charges.ChargeID{}

	s.Run("create new upcoming charges", func() {
		res, err := s.Charges.CreateCharges(ctx, charges.CreateChargeInputs{
			Namespace: ns,
			Intents: []charges.ChargeIntent{
				s.createMockChargeIntent(createMockChargeIntentInput{
					customer:       cust.GetID(),
					currency:       USD,
					servicePeriod:  servicePeriod,
					settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      alpacadecimal.NewFromFloat(100),
						PaymentTerm: productcatalog.InAdvancePaymentTerm,
					}),
					name:              flatFeeName,
					managedBy:         billing.SubscriptionManagedLine,
					uniqueReferenceID: flatFeeName,
				}),
			},
		})
		s.NoError(err)

		s.Len(res, 1)
		s.Equal(res[0].Type(), charges.ChargeTypeFlatFee)
		flatFeeCharge, err := res[0].AsFlatFeeCharge()
		s.NoError(err)

		gatheringInvoices, err := s.BillingService.ListGatheringInvoices(ctx, billing.ListGatheringInvoicesInput{
			Namespaces: []string{ns},
			Customers:  []string{cust.ID},
			Currencies: []currencyx.Code{currencyx.Code(currency.USD)},
			Expand:     []billing.GatheringInvoiceExpand{billing.GatheringInvoiceExpandLines},
		})
		s.NoError(err)
		s.Len(gatheringInvoices.Items, 1)
		gatheringInvoice := gatheringInvoices.Items[0]

		lines := gatheringInvoice.Lines.OrEmpty()
		s.Len(lines, 1)
		gatheringLine := lines[0]

		s.Equal(flatFeeCharge.ID, *gatheringLine.ChargeID)

		// TODO: validate periods, price, etc.

		flatFeeChargeID = flatFeeCharge.GetChargeID()
	})

	var stdInvoiceID billing.InvoiceID
	var stdLineID billing.LineID
	s.Run("invoice the charge", func() {
		defer s.FlatFeeTestHandler.Reset()

		testTrnsGroupID := ulid.Make().String()
		creditRealizationCallbackInvocations := 0
		s.FlatFeeTestHandler.onFlatFeeAssignedToInvoice = func(ctx context.Context, input charges.OnFlatFeeAssignedToInvoiceInput) ([]charges.CreditRealizationCreateInput, error) {
			creditRealizationCallbackInvocations++

			return []charges.CreditRealizationCreateInput{
				{
					ServicePeriod: input.ServicePeriod,
					Amount:        input.PreTaxTotalAmount.Mul(alpacadecimal.NewFromFloat(0.3)), // 30% as credits
					LedgerTransaction: charges.LedgerTransactionGroupReference{
						TransactionGroupID: testTrnsGroupID,
					},
				},
			}, nil
		}

		invoices, err := s.Charges.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(servicePeriod.From),
		})
		s.NoError(err)
		s.Len(invoices, 1)
		invoice := invoices[0]
		s.DebugDumpStandardInvoice("invoice after invoice pending lines", invoice)

		s.Len(invoice.Lines.OrEmpty(), 1)
		stdLine := invoice.Lines.OrEmpty()[0]

		s.Equal(flatFeeChargeID.ID, *stdLine.ChargeID)
		stdLineID = stdLine.GetLineID()

		s.Equal(1, creditRealizationCallbackInvocations)

		charge := s.mustGetChargeByID(flatFeeChargeID)
		updatedFlatFeeCharge, err := charge.AsFlatFeeCharge()
		s.NoError(err)

		s.Equal(flatFeeChargeID.ID, updatedFlatFeeCharge.ID)

		// Validate the credit realizations
		// The charge should have $30 realized as credits
		s.Len(updatedFlatFeeCharge.State.CreditRealizations, 1)
		creditRealization := updatedFlatFeeCharge.State.CreditRealizations[0]
		s.Equal(testTrnsGroupID, creditRealization.LedgerTransaction.TransactionGroupID)
		s.Equal(servicePeriod.From, creditRealization.ServicePeriod.From)
		s.Equal(servicePeriod.To, creditRealization.ServicePeriod.To)
		s.Equal(float64(30), creditRealization.Amount.InexactFloat64())

		// Validate the standard invoice's contents
		// Invoice totals should be $70
		s.Equal(float64(70), invoice.Totals.Total.InexactFloat64())
		s.Equal(float64(30), invoice.Totals.CreditsTotal.InexactFloat64())

		// Validate the standard line's contents
		// Line totals should be $70
		s.Equal(float64(30), stdLine.Totals.CreditsTotal.InexactFloat64())
		s.Equal(float64(70), stdLine.Totals.Total.InexactFloat64())

		// The line should have a credit realization intent
		s.Len(stdLine.CreditsApplied, 1)
		creditRealizationIntent := stdLine.CreditsApplied[0]
		s.Equal(float64(30), creditRealizationIntent.Amount.InexactFloat64())
		s.Equal(creditRealization.ID, creditRealizationIntent.CreditRealizationID)

		// The line should have a single detailed line
		s.Len(stdLine.DetailedLines, 1)
		detailedLine := stdLine.DetailedLines[0]
		s.Equal(float64(70), detailedLine.Totals.Total.InexactFloat64())
		s.Equal(float64(30), detailedLine.Totals.CreditsTotal.InexactFloat64())

		// The detailed line should have a credit realization intent
		s.Len(detailedLine.CreditsApplied, 1)
		creditRealizationDetail := detailedLine.CreditsApplied[0]
		s.Equal(float64(30), creditRealizationDetail.Amount.InexactFloat64())
		s.Equal(creditRealization.ID, creditRealizationDetail.CreditRealizationID)

		stdInvoiceID = invoice.GetInvoiceID()
		s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, invoice.Status)
	})

	s.Run("advance the invoice and authorize payment", func() {
		defer s.FlatFeeTestHandler.Reset()

		authorizedCallback := newCountedLedgerTransactionCallback[charges.FlatFeeCharge]()
		s.FlatFeeTestHandler.onFlatFeePaymentAuthorized = authorizedCallback.Handler(s.T())

		invoiceUsageAccruedCallback := newCountedLedgerTransactionCallback[charges.OnFlatFeeStandardInvoiceUsageAccruedInput]()
		s.FlatFeeTestHandler.onFlatFeeStandardInvoiceUsageAccrued = invoiceUsageAccruedCallback.Handler(s.T())

		invoice, err := s.BillingService.ApproveInvoice(ctx, stdInvoiceID)
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)

		s.Equal(1, authorizedCallback.nrInvocations)
		s.Equal(1, invoiceUsageAccruedCallback.nrInvocations)

		charge := s.mustGetChargeByID(flatFeeChargeID)
		updatedFlatFeeCharge, err := charge.AsFlatFeeCharge()
		s.NoError(err)

		// Invoice usage accrued callback should have been invoked
		accruedUsage := updatedFlatFeeCharge.State.AccruedUsage
		s.NotNil(accruedUsage)
		s.Equal(invoiceUsageAccruedCallback.id, accruedUsage.LedgerTransaction.TransactionGroupID, "ledger transaction gets recorded")
		s.Equal(servicePeriod, accruedUsage.ServicePeriod, "service period should be the same as the input")
		s.False(accruedUsage.Mutable, "accrued usage should not be mutable")
		s.NotNil(accruedUsage.LineID, "line ID should be set")
		s.Equal(stdLineID.ID, *accruedUsage.LineID, "line ID should be the same as the standard line")
		s.Equal(float64(70), accruedUsage.Totals.Total.InexactFloat64(), "totals should be the same as the input")
		s.Equal(float64(30), accruedUsage.Totals.CreditsTotal.InexactFloat64(), "totals should be the same as the input")

		// Authorization callback should have been invoked
		s.Equal(authorizedCallback.id, updatedFlatFeeCharge.State.Payment.Authorized.TransactionGroupID)
		s.Equal(charges.ChargeStatusActive, updatedFlatFeeCharge.Status)
	})

	s.Run("payment is settled", func() {
		defer s.FlatFeeTestHandler.Reset()

		settledCallback := newCountedLedgerTransactionCallback[charges.FlatFeeCharge]()
		s.FlatFeeTestHandler.onFlatFeePaymentSettled = settledCallback.Handler(s.T())

		invoice, err := customInvoicing.Service.HandlePaymentTrigger(ctx, appcustominvoicing.HandlePaymentTriggerInput{
			InvoiceID: stdInvoiceID,
			Trigger:   billing.TriggerPaid,
		})
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaid, invoice.Status)

		charge := s.mustGetChargeByID(flatFeeChargeID)
		updatedFlatFeeCharge, err := charge.AsFlatFeeCharge()
		s.NoError(err)
		s.Equal(settledCallback.id, updatedFlatFeeCharge.State.Payment.Settled.TransactionGroupID)
		s.Equal(charges.ChargeStatusFinal, updatedFlatFeeCharge.Status)
	})
}

type createMockChargeIntentInput struct {
	customer          customer.CustomerID
	currency          currencyx.Code
	servicePeriod     timeutil.ClosedPeriod
	price             *productcatalog.Price
	featureKey        string
	name              string
	settlementMode    productcatalog.SettlementMode
	managedBy         billing.InvoiceLineManagedBy
	uniqueReferenceID string
}

func (i *createMockChargeIntentInput) Validate() error {
	if i.price == nil {
		return errors.New("price is required")
	}

	if i.customer.Namespace == "" {
		return errors.New("customer namespace is required")
	}

	if i.customer.ID == "" {
		return errors.New("customer id is required")
	}

	if i.currency == "" {
		return errors.New("currency is required")
	}

	return nil
}

func (s *ChargesServiceTestSuite) createMockChargeIntent(input createMockChargeIntentInput) charges.ChargeIntent {
	s.T().Helper()
	s.NoError(input.Validate())

	isFlatFee := input.price.Type() == productcatalog.FlatPriceType
	invoiceAt := input.servicePeriod.To

	if isFlatFee {
		price, err := input.price.AsFlat()
		s.NoError(err)

		switch price.PaymentTerm {
		case productcatalog.InAdvancePaymentTerm:
			invoiceAt = input.servicePeriod.From
		case productcatalog.InArrearsPaymentTerm:
			invoiceAt = input.servicePeriod.To
		default:
			s.T().Fatalf("invalid payment term: %s", price.PaymentTerm)
		}
	}

	intentMeta := charges.IntentMeta{
		Name:              input.name,
		ManagedBy:         input.managedBy,
		ServicePeriod:     input.servicePeriod,
		FullServicePeriod: input.servicePeriod,
		BillingPeriod:     input.servicePeriod,
		UniqueReferenceID: lo.EmptyableToPtr(input.uniqueReferenceID),
		CustomerID:        input.customer.ID,
		Currency:          input.currency,
	}

	if isFlatFee {
		price, err := input.price.AsFlat()
		s.NoError(err)

		flatFeeIntent := charges.FlatFeeIntent{
			IntentMeta:     intentMeta,
			PaymentTerm:    price.PaymentTerm,
			FeatureKey:     input.featureKey,
			InvoiceAt:      invoiceAt,
			SettlementMode: lo.CoalesceOrEmpty(input.settlementMode, productcatalog.InvoiceOnlySettlementMode),

			AmountBeforeProration: price.Amount,
			AmountAfterProration:  price.Amount,
		}
		return charges.NewChargeIntent(flatFeeIntent)
	}

	usageBasedIntent := charges.UsageBasedIntent{
		IntentMeta:     intentMeta,
		Price:          *input.price,
		InvoiceAt:      invoiceAt,
		SettlementMode: lo.CoalesceOrEmpty(input.settlementMode, productcatalog.InvoiceOnlySettlementMode),
		FeatureKey:     input.featureKey,
	}

	return charges.NewChargeIntent(usageBasedIntent)
}

func (s *ChargesServiceTestSuite) mustGetChargeByID(chargeID charges.ChargeID) charges.Charge {
	s.T().Helper()
	charge, err := s.Charges.GetChargeByID(s.T().Context(), charges.GetChargeByIDInput{
		ChargeID: chargeID,
		Expands:  charges.Expands{charges.ExpandRealizations},
	})
	s.NoError(err)
	return charge
}

func (s *ChargesServiceTestSuite) TestPromotionalCreditPurchase() {
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
			settlement: charges.NewCreditPurchaseSettlement(charges.PromotionalCreditPurchaseSettlement{}),
		},
	)

	promotionalCallback := newCountedLedgerTransactionCallback[charges.CreditPurchaseCharge]()
	s.CreditPurchaseTestHandler.onPromotionalCreditPurchase = promotionalCallback.Handler(s.T(), func(t *testing.T, charge charges.CreditPurchaseCharge) {
		assert.Equal(t, charge.Intent.Settlement.Type(), charges.CreditPurchaseSettlementTypePromotional)
		assert.Nil(t, charge.State.CreditGrantRealization, "credit grant realization should not be set")
		assert.Nil(t, charge.State.ExternalPaymentSettlement, "external payment settlement should not be set")
	})

	res, err := s.Charges.CreateCharges(ctx, charges.CreateChargeInputs{
		Namespace: ns,
		Intents: []charges.ChargeIntent{
			intent,
		},
	})
	s.NoError(err)
	s.Len(res, 1)
	s.Equal(charges.ChargeTypeCreditPurchase, res[0].Type())

	s.Equal(1, promotionalCallback.nrInvocations)
	cpCharge, err := res[0].AsCreditPurchaseCharge()
	s.NoError(err)
	s.NotNil(cpCharge.State.CreditGrantRealization)
	s.Equal(promotionalCallback.id, cpCharge.State.CreditGrantRealization.LedgerTransactionGroupReference.TransactionGroupID)
	s.Equal(charges.ChargeStatusFinal, cpCharge.Status)

	charge := s.mustGetChargeByID(cpCharge.GetChargeID())
	updatedCPCharge, err := charge.AsCreditPurchaseCharge()
	s.NoError(err)
	s.Equal(promotionalCallback.id, updatedCPCharge.State.CreditGrantRealization.LedgerTransactionGroupReference.TransactionGroupID)
	s.Equal(charges.ChargeStatusFinal, updatedCPCharge.Status)
}

type createCreditPurchaseIntentInput struct {
	customer      customer.CustomerID
	currency      currencyx.Code
	amount        alpacadecimal.Decimal
	servicePeriod timeutil.ClosedPeriod
	settlement    charges.CreditPurchaseSettlement
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

	return charges.NewChargeIntent(charges.CreditPurchaseIntent{
		IntentMeta: charges.IntentMeta{
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

func (s *ChargesServiceTestSuite) TestExternalAuthorizedCreditPurchaseAutoSettled() {
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
			settlement: charges.NewCreditPurchaseSettlement(charges.ExternalCreditPurchaseSettlement{
				InitialStatus: charges.SettledInitialCreditPurchasePaymentSettlementStatus,
				GenericCreditPurchaseSettlement: charges.GenericCreditPurchaseSettlement{
					SettlementCurrency: USD,
					CostBasis:          alpacadecimal.NewFromFloat(0.5),
				},
			}),
		},
	)

	// First the initiated callback should be called, without any grant realizations or payment settlements
	initiatedCallback := newCountedLedgerTransactionCallback[charges.CreditPurchaseCharge]()
	s.CreditPurchaseTestHandler.onCreditPurchaseInitiated = initiatedCallback.Handler(s.T(), func(t *testing.T, charge charges.CreditPurchaseCharge) {
		assert.Equal(t, charge.Intent.Settlement.Type(), charges.CreditPurchaseSettlementTypeExternal)
		assert.Nil(t, charge.State.CreditGrantRealization, "credit grant realization should not be set")
		assert.Nil(t, charge.State.ExternalPaymentSettlement, "external payment settlement should not be set")
	})

	// Then the authorized callback should be called, with a grant realization and no payment settlement
	authorizedCallback := newCountedLedgerTransactionCallback[charges.CreditPurchaseCharge]()
	s.CreditPurchaseTestHandler.onCreditPurchasePaymentAuthorized = authorizedCallback.Handler(s.T(), func(t *testing.T, charge charges.CreditPurchaseCharge) {
		assert.Equal(t, charge.Intent.Settlement.Type(), charges.CreditPurchaseSettlementTypeExternal)
		assert.NotNil(t, charge.State.CreditGrantRealization, "credit grant realization should be set")
		assert.Equal(t, initiatedCallback.id, charge.State.CreditGrantRealization.LedgerTransactionGroupReference.TransactionGroupID)
		assert.Nil(t, charge.State.ExternalPaymentSettlement)
		assert.Equal(t, charges.ChargeStatusActive, charge.Status, "charge status should be active")
	})

	// Then the settled callback should be called, with a grant realization and a payment settlement
	settledCallback := newCountedLedgerTransactionCallback[charges.CreditPurchaseCharge]()
	s.CreditPurchaseTestHandler.onCreditPurchasePaymentSettled = settledCallback.Handler(s.T(), func(t *testing.T, charge charges.CreditPurchaseCharge) {
		assert.Equal(t, charge.Intent.Settlement.Type(), charges.CreditPurchaseSettlementTypeExternal)
		assert.NotNil(t, charge.State.ExternalPaymentSettlement, "external payment settlement should be set")

		// Authorized transaction group ID should be set
		assert.Equal(t, authorizedCallback.id, charge.State.ExternalPaymentSettlement.Authorized.TransactionGroupID)
		assert.Equal(t, charges.PaymentSettlementStatusAuthorized, charge.State.ExternalPaymentSettlement.Status)
		assert.Equal(t, charges.ChargeStatusActive, charge.Status, "charge status should be active")
	})
	res, err := s.Charges.CreateCharges(ctx, charges.CreateChargeInputs{
		Namespace: ns,
		Intents: []charges.ChargeIntent{
			intent,
		},
	})
	s.NoError(err)
	s.Len(res, 1)
	s.Equal(charges.ChargeTypeCreditPurchase, res[0].Type())

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
			s.Equal(initiatedCallback.id, creditPurchaseCharge.State.CreditGrantRealization.LedgerTransactionGroupReference.TransactionGroupID)

			// Payment settlement should be set
			s.NotNil(creditPurchaseCharge.State.ExternalPaymentSettlement, "external payment settlement should be set")
			s.Equal(authorizedCallback.id, creditPurchaseCharge.State.ExternalPaymentSettlement.Authorized.TransactionGroupID, "authorized transaction group ID should be set")
			s.Equal(settledCallback.id, creditPurchaseCharge.State.ExternalPaymentSettlement.Settled.TransactionGroupID, "settled transaction group ID should be set")

			// The charge should be final
			s.Equal(charges.ChargeStatusFinal, creditPurchaseCharge.Status)
		})
	}
}

func (s *ChargesServiceTestSuite) TestExternalAuthorizedCreditPurchaseManuallySettled() {
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
			settlement: charges.NewCreditPurchaseSettlement(charges.ExternalCreditPurchaseSettlement{
				InitialStatus: charges.CreatedInitialCreditPurchasePaymentSettlementStatus,
				GenericCreditPurchaseSettlement: charges.GenericCreditPurchaseSettlement{
					SettlementCurrency: USD,
					CostBasis:          alpacadecimal.NewFromFloat(0.5),
				},
			}),
		},
	)

	var chargeID charges.ChargeID
	var initatedTrnsID string

	s.Run("initiated", func() {
		defer s.CreditPurchaseTestHandler.Reset()

		// First the initiated callback should be called, without any grant realizations or payment settlements
		initatedCallback := newCountedLedgerTransactionCallback[charges.CreditPurchaseCharge]()
		s.CreditPurchaseTestHandler.onCreditPurchaseInitiated = initatedCallback.Handler(s.T(), func(t *testing.T, charge charges.CreditPurchaseCharge) {
			assert.Equal(t, charge.Intent.Settlement.Type(), charges.CreditPurchaseSettlementTypeExternal)
			assert.Nil(t, charge.State.CreditGrantRealization, "credit grant realization should not be set")
			assert.Nil(t, charge.State.ExternalPaymentSettlement, "external payment settlement should not be set")
		})

		res, err := s.Charges.CreateCharges(ctx, charges.CreateChargeInputs{
			Namespace: ns,
			Intents: []charges.ChargeIntent{
				intent,
			},
		})
		s.NoError(err)
		s.Len(res, 1)
		s.Equal(charges.ChargeTypeCreditPurchase, res[0].Type())

		creditPurchaseCharge, err := res[0].AsCreditPurchaseCharge()
		s.NoError(err)
		s.Equal(1, initatedCallback.nrInvocations)
		s.Equal(initatedCallback.id, creditPurchaseCharge.State.CreditGrantRealization.LedgerTransactionGroupReference.TransactionGroupID)
		s.Equal(charges.ChargeStatusActive, creditPurchaseCharge.Status)

		chargeID = creditPurchaseCharge.GetChargeID()
		initatedTrnsID = initatedCallback.id
	})

	var authorizedTrnsID string
	s.Run("authorized", func() {
		defer s.CreditPurchaseTestHandler.Reset()

		// Then the authorized callback should be called, with a grant realization and no payment settlement
		authorizedCallback := newCountedLedgerTransactionCallback[charges.CreditPurchaseCharge]()
		s.CreditPurchaseTestHandler.onCreditPurchasePaymentAuthorized = authorizedCallback.Handler(s.T(), func(t *testing.T, charge charges.CreditPurchaseCharge) {
			assert.Equal(t, charge.Intent.Settlement.Type(), charges.CreditPurchaseSettlementTypeExternal)
			assert.NotNil(t, charge.State.CreditGrantRealization, "credit grant realization should be set")
			assert.Equal(t, initatedTrnsID, charge.State.CreditGrantRealization.LedgerTransactionGroupReference.TransactionGroupID)
			assert.Nil(t, charge.State.ExternalPaymentSettlement)
			assert.Equal(t, charges.ChargeStatusActive, charge.Status, "charge status should be active")
		})

		res, err := s.Charges.UpdateExternalCreditPurchasePaymentState(ctx, charges.UpdateExternalCreditPurchasePaymentStateInput{
			ChargeID:           chargeID,
			TargetPaymentState: charges.PaymentSettlementStatusAuthorized,
		})
		s.NoError(err)

		s.Equal(1, authorizedCallback.nrInvocations)
		s.Equal(authorizedCallback.id, res.State.ExternalPaymentSettlement.Authorized.TransactionGroupID)
		s.Equal(charges.PaymentSettlementStatusAuthorized, res.State.ExternalPaymentSettlement.Status)
		s.Equal(charges.ChargeStatusActive, res.Status)

		authorizedTrnsID = authorizedCallback.id
	})

	s.Run("settled", func() {
		defer s.CreditPurchaseTestHandler.Reset()

		// Then the settled callback should be called, with a grant realization and a payment settlement
		settledCallback := newCountedLedgerTransactionCallback[charges.CreditPurchaseCharge]()
		s.CreditPurchaseTestHandler.onCreditPurchasePaymentSettled = settledCallback.Handler(s.T(), func(t *testing.T, charge charges.CreditPurchaseCharge) {
			assert.Equal(t, charge.Intent.Settlement.Type(), charges.CreditPurchaseSettlementTypeExternal)
			assert.NotNil(t, charge.State.ExternalPaymentSettlement, "external payment settlement should be set")

			// Authorized transaction group ID should be set
			assert.Equal(t, authorizedTrnsID, charge.State.ExternalPaymentSettlement.Authorized.TransactionGroupID)
			assert.Equal(t, charges.PaymentSettlementStatusAuthorized, charge.State.ExternalPaymentSettlement.Status)
			assert.Equal(t, charges.ChargeStatusActive, charge.Status, "charge status should be active")
		})
		res, err := s.Charges.UpdateExternalCreditPurchasePaymentState(ctx, charges.UpdateExternalCreditPurchasePaymentStateInput{
			ChargeID:           chargeID,
			TargetPaymentState: charges.PaymentSettlementStatusSettled,
		})
		s.NoError(err)

		s.Equal(1, settledCallback.nrInvocations)
		s.Equal(settledCallback.id, res.State.ExternalPaymentSettlement.Settled.TransactionGroupID)
		s.Equal(charges.PaymentSettlementStatusSettled, res.State.ExternalPaymentSettlement.Status)
		s.Equal(charges.ChargeStatusFinal, res.Status)
	})
}
