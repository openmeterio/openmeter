package service

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/oklog/ulid/v2"
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
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"
)

const USD = currencyx.Code(currency.USD)

type ChargesServiceTestSuite struct {
	billingtest.BaseSuite

	Charges            *service
	FlatFeeTestHandler *flatFeeTestHandler
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

	chargesService, err := New(Config{
		Adapter:        chargesAdapter,
		BillingService: s.BillingService,
		Handlers: Handlers{
			FlatFee: s.FlatFeeTestHandler,
		},
	})
	s.NoError(err)
	s.Charges = chargesService
}

func (s *ChargesServiceTestSuite) TeardownTest() {
	s.FlatFeeTestHandler.Reset()
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
	s.Run("invoice the charge", func() {
		defer s.FlatFeeTestHandler.Reset()

		s.FlatFeeTestHandler.onFlatFeeAssignedToInvoice = func(ctx context.Context, input charges.OnFlatFeeAssignedToInvoiceInput) ([]charges.CreditRealizationCreateInput, error) {
			return []charges.CreditRealizationCreateInput{
				{
					ServicePeriod: input.ServicePeriod,
					Amount:        input.PreTaxTotalAmount.Mul(alpacadecimal.NewFromFloat(0.3)), // 30% as credits
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

		charge, err := s.Charges.GetChargeByID(ctx, flatFeeChargeID)
		s.NoError(err)
		updatedFlatFeeCharge, err := charge.AsFlatFeeCharge()
		s.NoError(err)

		s.Equal(flatFeeChargeID.ID, updatedFlatFeeCharge.ID)

		// Validate the credit realizations
		// The charge should have $30 realized as credits
		s.Len(updatedFlatFeeCharge.State.CreditRealizations, 1)
		creditRealization := updatedFlatFeeCharge.State.CreditRealizations[0]
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

		testTrnsGroupID := ulid.Make().String()

		authorizedCallbackInvocations := 0
		s.FlatFeeTestHandler.onFlatFeePaymentAuthorized = func(ctx context.Context, charge charges.FlatFeeCharge) (charges.LedgerTransactionGroupReference, error) {
			authorizedCallbackInvocations++
			return charges.LedgerTransactionGroupReference{
				TransactionGroupID: testTrnsGroupID,
			}, nil
		}

		invoice, err := s.BillingService.ApproveInvoice(ctx, stdInvoiceID)
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)

		s.Equal(1, authorizedCallbackInvocations)

		charge, err := s.Charges.GetChargeByID(ctx, flatFeeChargeID)
		s.NoError(err)
		updatedFlatFeeCharge, err := charge.AsFlatFeeCharge()
		s.NoError(err)
		s.Equal(testTrnsGroupID, updatedFlatFeeCharge.State.AuthorizedTransaction.TransactionGroupID)
		s.Equal(charges.ChargeStatusActive, updatedFlatFeeCharge.Status)
	})

	s.Run("payment is settled", func() {
		defer s.FlatFeeTestHandler.Reset()

		testTrnsGroupID := ulid.Make().String()

		settledCallbackInvocations := 0
		s.FlatFeeTestHandler.onFlatFeePaymentSettled = func(ctx context.Context, charge charges.FlatFeeCharge) (charges.LedgerTransactionGroupReference, error) {
			settledCallbackInvocations++
			return charges.LedgerTransactionGroupReference{
				TransactionGroupID: testTrnsGroupID,
			}, nil
		}

		invoice, err := customInvoicing.Service.HandlePaymentTrigger(ctx, appcustominvoicing.HandlePaymentTriggerInput{
			InvoiceID: stdInvoiceID,
			Trigger:   billing.TriggerPaid,
		})
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaid, invoice.Status)

		charge, err := s.Charges.GetChargeByID(ctx, flatFeeChargeID)
		s.NoError(err)
		updatedFlatFeeCharge, err := charge.AsFlatFeeCharge()
		s.NoError(err)
		s.Equal(testTrnsGroupID, updatedFlatFeeCharge.State.SettledTransaction.TransactionGroupID)
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
			s.Fail("invalid payment term: %s", price.PaymentTerm)
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
