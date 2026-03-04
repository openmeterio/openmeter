package credits

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/adapter"
	chargesservice "github.com/openmeterio/openmeter/openmeter/billing/charges/service"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

const USD = currencyx.Code(currency.USD)

type CreditsTestSuite struct {
	billingtest.BaseSuite

	Charges charges.Service
	Ledger  *MockLedger
}

func TestCreditsTestSuite(t *testing.T) {
	suite.Run(t, new(CreditsTestSuite))
}

func (s *CreditsTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()

	chargesAdapter, err := adapter.New(adapter.Config{
		Client: s.DBClient,
		Logger: slog.Default(),
	})
	s.NoError(err)

	s.Ledger = newMockLedger()

	chargesService, err := chargesservice.New(chargesservice.Config{
		Adapter:        chargesAdapter,
		BillingService: s.BillingService,
		Handlers: chargesservice.Handlers{
			FlatFee:        s.Ledger,
			CreditPurchase: s.Ledger,
		},
	})
	s.NoError(err)
	s.Charges = chargesService
}

func (s *CreditsTestSuite) TeardownTest() {
	s.Ledger.Reset()
}

func (s *CreditsTestSuite) TestFlatFeePartialCreditRealizations() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("charges-sanity-test")

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

	setupAt := datetime.MustParseTimeInLocation(s.T(), "2025-12-31T00:00:00Z", time.UTC).AsTime()

	clock.SetTime(setupAt)

	s.Run("the customer receives a promotional credit grant", func() {
		intent := s.createCreditPurchaseIntent(createCreditPurchaseIntentInput{
			customer: cust.GetID(),
			currency: USD,
			amount:   alpacadecimal.NewFromFloat(30),
			servicePeriod: timeutil.ClosedPeriod{
				From: setupAt,
				To:   setupAt,
			},
			settlement: charges.NewCreditPurchaseSettlement(charges.PromotionalCreditPurchaseSettlement{}),
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
		cpCharge, err := res[0].AsCreditPurchaseCharge()
		s.NoError(err)

		// This should match the ledger's transaction group ID
		s.NotEmpty(cpCharge.State.CreditGrantRealization.TransactionGroupID)

		// LEDGER[galexi]:
		// - OnPromotionalCreditPurchase is called
		// - At this point the customer must have 30 USD promotional credits

		// Validate balances
		s.Equal(float64(30), s.Ledger.customerPromotionalCredits)
		s.Equal(float64(0), s.Ledger.customerCredits)
	})

	var externalCreditPurchaseChargeID charges.ChargeID
	s.Run("and customer purchases 50 USD credits as 0.5 costbasis", func() {
		intent := s.createCreditPurchaseIntent(createCreditPurchaseIntentInput{
			customer: cust.GetID(),
			currency: USD,
			amount:   alpacadecimal.NewFromFloat(50),
			servicePeriod: timeutil.ClosedPeriod{
				From: setupAt,
				To:   setupAt,
			},
			settlement: charges.NewCreditPurchaseSettlement(charges.ExternalCreditPurchaseSettlement{
				GenericCreditPurchaseSettlement: charges.GenericCreditPurchaseSettlement{
					SettlementCurrency: USD,
					CostBasis:          alpacadecimal.NewFromFloat(0.5),
				},
				InitialStatus: charges.CreatedInitialCreditPurchasePaymentSettlementStatus,
			}),
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
		cpCharge, err := res[0].AsCreditPurchaseCharge()
		s.NoError(err)

		// This should match the ledger's transaction group ID
		s.NotEmpty(cpCharge.State.CreditGrantRealization.TransactionGroupID)

		// LEDGER[galexi]:
		// - OnCreditPurchaseInitiated is called
		// - At this point the customer must have 50 USD credits cost basis of 0.5

		// Validate balances
		s.Equal(float64(50), s.Ledger.customerCredits)
		s.Equal(float64(25), s.Ledger.receivables)

		externalCreditPurchaseChargeID = cpCharge.GetChargeID()
	})

	s.Run("the customer pays for the credit purchase - authorized", func() {
		updatedCharge, err := s.Charges.UpdateExternalCreditPurchasePaymentState(ctx, charges.UpdateExternalCreditPurchasePaymentStateInput{
			ChargeID:           externalCreditPurchaseChargeID,
			TargetPaymentState: charges.PaymentSettlementStatusAuthorized,
		})
		s.NoError(err)

		// LEDGER[galexi]:
		// - OnCreditPurchasePaymentAuthorized is called

		s.Equal(charges.PaymentSettlementStatusAuthorized, updatedCharge.State.ExternalPaymentSettlement.Status)
		s.Equal(float64(25), s.Ledger.receivables)
	})

	s.Run("the customer settles the credit purchase payment", func() {
		updatedCharge, err := s.Charges.UpdateExternalCreditPurchasePaymentState(ctx, charges.UpdateExternalCreditPurchasePaymentStateInput{
			ChargeID:           externalCreditPurchaseChargeID,
			TargetPaymentState: charges.PaymentSettlementStatusSettled,
		})
		s.NoError(err)

		// LEDGER[galexi]:
		// - OnCreditPurchasePaymentSettled is called

		s.Equal(charges.PaymentSettlementStatusSettled, updatedCharge.State.ExternalPaymentSettlement.Status)
		s.Equal(float64(25), s.Ledger.receivables)
	})

	// TOTAL credits balance: 30 + 50 = 80 USD

	var flatFeeChargeID charges.ChargeID

	s.Run("create new upcoming charge for flat fee", func() {
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

		// LEDGER[galexi]:
		// - This is a noop as this is in the future, so it just creates the charge + gathering line

		flatFeeChargeID = flatFeeCharge.GetChargeID()
	})

	var stdInvoiceID billing.InvoiceID
	var stdLineID billing.LineID
	clock.SetTime(servicePeriod.From)
	s.Run("invoice the charge", func() {
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

		charge := s.mustGetChargeByID(flatFeeChargeID)
		updatedFlatFeeCharge, err := charge.AsFlatFeeCharge()
		s.NoError(err)

		s.Equal(flatFeeChargeID.ID, updatedFlatFeeCharge.ID)

		// LEDGER[galexi]:
		// - OnFlatFeeAssignedToInvoice is called with the pre tax total amount of USD 100
		// - Two credit realizations should happen for the two different credit types

		// Validate the credit realizations
		// The charge should have $80 realized as credits
		s.Len(updatedFlatFeeCharge.State.CreditRealizations, 2)
		promotionalCreditRealization := updatedFlatFeeCharge.State.CreditRealizations[0]
		s.Equal(float64(30), promotionalCreditRealization.Amount.InexactFloat64())

		customerCreditRealization := updatedFlatFeeCharge.State.CreditRealizations[1]
		s.Equal(float64(50), customerCreditRealization.Amount.InexactFloat64())

		stdInvoiceID = invoice.GetInvoiceID()
		s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, invoice.Status)
	})

	s.Run("advance the invoice and authorize payment", func() {
		invoice, err := s.BillingService.ApproveInvoice(ctx, stdInvoiceID)
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)

		charge := s.mustGetChargeByID(flatFeeChargeID)
		updatedFlatFeeCharge, err := charge.AsFlatFeeCharge()
		s.NoError(err)

		// LEDGER[galexi]:
		// - OnFlatFeeStandardInvoiceUsageAccrued is called with the service period and totals of USD 20 to be represented
		//   on the ledger
		// - OnFlatFeePaymentAuthorized is called (I cannot make this a two step process without creating a new app) with the USD 20

		// Invoice usage accrued callback should have been invoked
		accruedUsage := updatedFlatFeeCharge.State.AccruedUsage
		s.NotNil(accruedUsage)
		s.Equal(servicePeriod, accruedUsage.ServicePeriod, "service period should be the same as the input")
		s.False(accruedUsage.Mutable, "accrued usage should not be mutable")
		s.NotNil(accruedUsage.LineID, "line ID should be set")
		s.Equal(stdLineID.ID, *accruedUsage.LineID, "line ID should be the same as the standard line")
		s.Equal(float64(20), accruedUsage.Totals.Total.InexactFloat64(), "totals should be the same as the input")
		s.Equal(float64(70), accruedUsage.Totals.CreditsTotal.InexactFloat64(), "totals should be the same as the input")
	})

	s.Run("payment is settled", func() {
		invoice, err := customInvoicing.Service.HandlePaymentTrigger(ctx, appcustominvoicing.HandlePaymentTriggerInput{
			InvoiceID: stdInvoiceID,
			Trigger:   billing.TriggerPaid,
		})
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaid, invoice.Status)

		// LEDGER[galexi]:
		// - OnFlatFeePaymentSettled is called with the USD 20

		charge := s.mustGetChargeByID(flatFeeChargeID)
		updatedFlatFeeCharge, err := charge.AsFlatFeeCharge()
		s.NoError(err)
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

func (s *CreditsTestSuite) createMockChargeIntent(input createMockChargeIntentInput) charges.ChargeIntent {
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

func (s *CreditsTestSuite) mustGetChargeByID(chargeID charges.ChargeID) charges.Charge {
	s.T().Helper()
	charge, err := s.Charges.GetChargeByID(s.T().Context(), charges.GetChargeByIDInput{
		ChargeID: chargeID,
		Expands:  charges.Expands{charges.ExpandRealizations},
	})
	s.NoError(err)
	return charge
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

func (s *CreditsTestSuite) createCreditPurchaseIntent(input createCreditPurchaseIntentInput) charges.ChargeIntent {
	s.T().Helper()
	s.NoError(input.Validate())

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
