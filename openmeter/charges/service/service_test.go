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
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/charges"
	"github.com/openmeterio/openmeter/openmeter/charges/adapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

type ChargesServiceTestSuite struct {
	billingtest.BaseSuite

	Charges *service
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

	chargesService, err := New(Config{
		Adapter:            chargesAdapter,
		BillingService:     s.BillingService,
		Handler:            charges.NewNoopHandlerRouter(),
		CustomerService:    s.CustomerService,
		StreamingConnector: s.MockStreamingConnector,
		FeatureService:     s.FeatureService,
	})
	s.NoError(err)
	s.Charges = chargesService
}

func (s *ChargesServiceTestSuite) TeardownTest() {
	s.Charges.handler = charges.NewNoopHandlerRouter()
}

func (s *ChargesServiceTestSuite) SetupMockHandler() *MockHandler {
	mockHandler := &MockHandler{}
	s.Charges.handler = mockHandler

	return mockHandler
}

func (s *ChargesServiceTestSuite) SetupRecordingHandler() *RecordingHandler {
	recordingHandler := &RecordingHandler{}
	s.Charges.handler = recordingHandler

	return recordingHandler
}

func (s *ChargesServiceTestSuite) TestChargeInvoiceOnlyFlow() {
	namespace := "ns-charges-service"
	ctx := context.Background()
	defer clock.ResetTime()

	customInvoicing := s.SetupCustomInvoicing(namespace)

	cust := s.CreateTestCustomer(namespace, "test")

	_ = s.ProvisionBillingProfile(ctx, namespace, customInvoicing.App.GetID(),
		billingtest.WithProgressiveBilling(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "PT1H")),
		billingtest.WithManualApproval(),
	)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	clock.SetTime(servicePeriod.From)

	const (
		flatFeeName    = "test-flat-fee"
		usageBasedName = "test-usage-based"
	)

	feature := s.SetupApiRequestsTotalFeature(ctx, namespace)
	var (
		flatFeeChargeID    charges.ChargeID
		usageBasedChargeID charges.ChargeID
	)

	recordingHandler := s.SetupRecordingHandler()

	s.Run("create new upcoming charges", func() {
		res, err := s.Charges.CreateCharges(ctx, charges.CreateChargeInput{
			Customer: cust.GetID(),
			Currency: currencyx.Code(currency.USD),
			Intents: []charges.CreateChargeIntentInput{
				s.createMockChargeIntent(createMockChargeIntentInput{
					servicePeriod: servicePeriod,
					price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      alpacadecimal.NewFromFloat(100),
						PaymentTerm: productcatalog.InAdvancePaymentTerm,
					}),
					name:              flatFeeName,
					managedBy:         billing.SubscriptionManagedLine,
					uniqueReferenceID: flatFeeName,
				}),
				s.createMockChargeIntent(createMockChargeIntentInput{
					servicePeriod: servicePeriod,
					price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromFloat(2),
					}),
					name:              usageBasedName,
					featureKey:        feature.Feature.Key,
					managedBy:         billing.SubscriptionManagedLine,
					uniqueReferenceID: usageBasedName,
				}),
			},
		})
		s.NoError(err)

		s.Len(res, 2)
		s.Equal(res[0].Intent.IntentType, charges.IntentTypeFlatFee)
		s.Equal(res[1].Intent.IntentType, charges.IntentTypeUsageBased)

		// TODO: more checks (service period, invoice at, etc.)
		s.NotEmpty(res[0].Expanded.GatheringLines[0].Invoice.ID)
		s.Equal(res[0].Expanded.GatheringLines[0].Invoice.ID, res[1].Expanded.GatheringLines[0].Invoice.ID)

		// Line price types
		s.Equal(res[0].Expanded.GatheringLines[0].Line.Price.Type(), productcatalog.FlatPriceType)
		s.Equal(res[1].Expanded.GatheringLines[0].Line.Price.Type(), productcatalog.UnitPriceType)

		flatFeeChargeID = res[0].GetChargeID()
		usageBasedChargeID = res[1].GetChargeID()
	})

	var stdInvoice billing.StandardInvoice
	s.Run("create mid-period progressively billed invoice", func() {
		asOf := datetime.MustParseTimeInLocation(s.T(), "2026-01-16T00:00:00Z", time.UTC).AsTime()
		clock.SetTime(asOf)

		out, err := s.Charges.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     &asOf,
			// TODO: this should not be needed, check why?
			ProgressiveBillingOverride: lo.ToPtr(true),
		})
		s.NoError(err)

		s.Len(out, 1)
		stdInvoice = out[0]
		s.Equal(stdInvoice.Status, billing.StandardInvoiceStatusDraftManualApprovalNeeded)

		lines := stdInvoice.Lines.OrEmpty()
		s.Len(lines, 2)
		stdLineFlatFee := s.getStandardLineByName(lines, flatFeeName)
		stdLineUsageBased := s.getStandardLineByName(lines, usageBasedName)

		s.Nil(stdLineFlatFee.SplitLineGroupID, "split line group ID should not be set for flat fee line")
		s.NotNil(stdLineFlatFee.ChargeID, "charge ID should be set for flat fee line")
		s.Equal(*stdLineFlatFee.ChargeID, flatFeeChargeID.ID, "charge ID should match")

		s.NotNil(stdLineUsageBased.SplitLineGroupID, "split line group ID should be set for usage based line")
		s.NotNil(stdLineUsageBased.ChargeID, "charge ID should be set for usage based line")
		s.Equal(*stdLineUsageBased.ChargeID, usageBasedChargeID.ID, "charge ID should match")

		flatFeeCharge := s.mustGetCharge(flatFeeChargeID)
		flatFeeRealization, found := flatFeeCharge.Realizations.StandardInvoice.GetByLineID(stdLineFlatFee.ID)
		s.True(found, "flat fee realization should be found")
		s.Equal(flatFeeRealization.Status, charges.StandardInvoiceRealizationStatusDraft)
		s.Equal(flatFeeRealization.ServicePeriod, stdLineFlatFee.Period.ToClosedPeriod())
		s.Equal(flatFeeRealization.Totals, stdLineFlatFee.Totals)

		usageBasedCharge := s.mustGetCharge(usageBasedChargeID)
		usageBasedRealization, found := usageBasedCharge.Realizations.StandardInvoice.GetByLineID(stdLineUsageBased.ID)
		s.True(found, "usage based realization should be found")
		s.Equal(usageBasedRealization.Status, charges.StandardInvoiceRealizationStatusDraft)
		s.Equal(usageBasedRealization.ServicePeriod, stdLineUsageBased.Period.ToClosedPeriod())
		s.Equal(usageBasedRealization.Totals, stdLineUsageBased.Totals)

		recordingHandler.Expect(s.T(), recordingHandlerExpectation{
			standardInvoiceRealizationCreated: []recordingHandlerExpectationItem{
				{
					chargeID:      flatFeeChargeID.ID,
					realizationID: flatFeeRealization.ID,
					status:        charges.StandardInvoiceRealizationStatusDraft,
				},
				{
					chargeID:      usageBasedChargeID.ID,
					realizationID: usageBasedRealization.ID,
					status:        charges.StandardInvoiceRealizationStatusDraft,
				},
			},
		})

		recordingHandler.Reset()
	})

	s.Run("approve invoice, payment is initiated", func() {
		asOf := datetime.MustParseTimeInLocation(s.T(), "2026-01-16T12:00:00Z", time.UTC).AsTime()
		clock.SetTime(asOf)

		var err error
		stdInvoice, err = s.BillingService.ApproveInvoice(ctx, stdInvoice.GetInvoiceID())
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, stdInvoice.Status)

		lines := stdInvoice.Lines.OrEmpty()
		s.Len(lines, 2)
		stdLineFlatFee := s.getStandardLineByName(lines, flatFeeName)
		stdLineUsageBased := s.getStandardLineByName(lines, usageBasedName)

		flatFeeCharge := s.mustGetCharge(flatFeeChargeID)
		flatFeeRealization, found := flatFeeCharge.Realizations.StandardInvoice.GetByLineID(stdLineFlatFee.ID)
		s.True(found, "flat fee realization should be found")
		s.Equal(flatFeeRealization.Status, charges.StandardInvoiceRealizationStatusAuthorized)

		usageBasedCharge := s.mustGetCharge(usageBasedChargeID)
		usageBasedRealization, found := usageBasedCharge.Realizations.StandardInvoice.GetByLineID(stdLineUsageBased.ID)
		s.True(found, "usage based realization should be found")
		s.Equal(usageBasedRealization.Status, charges.StandardInvoiceRealizationStatusAuthorized)

		recordingHandler.Expect(s.T(), recordingHandlerExpectation{
			standardInvoiceRealizationAuthorized: []recordingHandlerExpectationItem{
				{
					chargeID:      flatFeeChargeID.ID,
					realizationID: flatFeeRealization.ID,
					status:        charges.StandardInvoiceRealizationStatusAuthorized,
				},
				{
					chargeID:      usageBasedChargeID.ID,
					realizationID: usageBasedRealization.ID,
					status:        charges.StandardInvoiceRealizationStatusAuthorized,
				},
			},
		})

		recordingHandler.Reset()
	})

	s.Run("settle invoice, payment is settled", func() {
		asOf := datetime.MustParseTimeInLocation(s.T(), "2026-01-16T18:00:00Z", time.UTC).AsTime()
		clock.SetTime(asOf)

		_, err := customInvoicing.Service.HandlePaymentTrigger(ctx, appcustominvoicing.HandlePaymentTriggerInput{
			InvoiceID: stdInvoice.GetInvoiceID(),
			Trigger:   billing.TriggerPaid,
		})
		s.NoError(err)

		// Let's refetch the invoice as the HandlePaymentTrigger only returns the invoice without expanded lines
		stdInvoice, err = s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
			Invoice: stdInvoice.GetInvoiceID(),
			Expand: billing.StandardInvoiceExpands{
				billing.StandardInvoiceExpandLines,
			},
		})
		s.NoError(err)

		s.Equal(billing.StandardInvoiceStatusPaid, stdInvoice.Status)

		lines := stdInvoice.Lines.OrEmpty()
		s.Len(lines, 2)
		stdLineFlatFee := s.getStandardLineByName(lines, flatFeeName)
		stdLineUsageBased := s.getStandardLineByName(lines, usageBasedName)

		flatFeeCharge := s.mustGetCharge(flatFeeChargeID)
		flatFeeRealization, found := flatFeeCharge.Realizations.StandardInvoice.GetByLineID(stdLineFlatFee.ID)
		s.True(found, "flat fee realization should be found")
		s.Equal(flatFeeRealization.Status, charges.StandardInvoiceRealizationStatusSettled)

		usageBasedCharge := s.mustGetCharge(usageBasedChargeID)
		usageBasedRealization, found := usageBasedCharge.Realizations.StandardInvoice.GetByLineID(stdLineUsageBased.ID)
		s.True(found, "usage based realization should be found")
		s.Equal(usageBasedRealization.Status, charges.StandardInvoiceRealizationStatusSettled)

		recordingHandler.Expect(s.T(), recordingHandlerExpectation{
			standardInvoiceRealizationSettled: []recordingHandlerExpectationItem{
				{
					chargeID:      flatFeeChargeID.ID,
					realizationID: flatFeeRealization.ID,
					status:        charges.StandardInvoiceRealizationStatusSettled,
				},
				{
					chargeID:      usageBasedChargeID.ID,
					realizationID: usageBasedRealization.ID,
					status:        charges.StandardInvoiceRealizationStatusSettled,
				},
			},
		})

		recordingHandler.Reset()
	})
}

type createMockChargeIntentInput struct {
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

	return nil
}

func (s *ChargesServiceTestSuite) createMockChargeIntent(input createMockChargeIntentInput) charges.CreateChargeIntentInput {
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
		ManagedBy:         input.managedBy,
		ServicePeriod:     input.servicePeriod,
		FullServicePeriod: input.servicePeriod,
		BillingPeriod:     input.servicePeriod,
		InvoiceAt:         invoiceAt,
		SettlementMode:    lo.CoalesceOrEmpty(input.settlementMode, productcatalog.InvoiceOnlySettlementMode),
		UniqueReferenceID: lo.EmptyableToPtr(input.uniqueReferenceID),
	}

	var intent charges.Intent
	if isFlatFee {
		price, err := input.price.AsFlat()
		s.NoError(err)

		flatFeeIntent := charges.FlatFeeIntent{
			PaymentTerm: price.PaymentTerm,
			FeatureKey:  input.featureKey,

			AmountBeforeProration: price.Amount,
			AmountAfterProration:  price.Amount,
		}
		intent = charges.NewIntent(intentMeta, flatFeeIntent)
	} else {
		usageBasedIntent := charges.UsageBasedIntent{
			Price:      *input.price,
			FeatureKey: input.featureKey,
		}
		intent = charges.NewIntent(intentMeta, usageBasedIntent)
	}

	return charges.CreateChargeIntentInput{
		Name:   input.name,
		Intent: intent,
	}
}

func (s *ChargesServiceTestSuite) getStandardLineByName(lines billing.StandardLines, id string) *billing.StandardLine {
	s.T().Helper()

	line, found := lo.Find(lines, func(line *billing.StandardLine) bool {
		return line.Name == id
	})
	if !found {
		s.Failf("line not found", "line with name %s not found", id)
		return nil
	}

	return line
}

func (s *ChargesServiceTestSuite) mustGetCharge(id charges.ChargeID) charges.Charge {
	s.T().Helper()

	charge, err := s.Charges.GetChargeByID(s.T().Context(), id)
	s.NoError(err)

	return charge
}

type testCreditThenInvoiceHandler struct {
	charges.NoOpHandler
	charges charges.Service

	// emulate ledger
	customerAccountBalance     float64
	customerOutstandingBalance float64

	// TODO: let's use the balances instead
	realizePercent float64
}

func (h *testCreditThenInvoiceHandler) OnRealizeUsageBasedCreditChargePeriodically(ctx context.Context, input charges.UsageBasedRealizationInput) ([]charges.CreditRealizationCreateInput, error) {
	realizePercent := alpacadecimal.NewFromFloat(h.realizePercent)

	// Get the realized unsettled amount for the charge and the line's current total usage amount
	realizedUnsettledAmount := input.Charge.Realizations.RealizedUnsettledAmount()
	totalUsageAmount := input.CurrentUsage.Totals.Amount

	// Calculate the realizable amount (amount that can be realized from credits as the difference of the previous two)
	realizableAmount := totalUsageAmount.Sub(realizedUnsettledAmount)
	if realizableAmount.IsNegative() || realizableAmount.IsZero() {
		return nil, nil
	}

	// Simulate ledger balance checks by multiplying the realizable amount by the realization percentage

	amountToRealize := realizableAmount.Mul(realizePercent)

	if amountToRealize.IsNegative() {
		return nil, fmt.Errorf("amount to realize is negative: %s", amountToRealize.String())
	}

	// Let's find out the service period for the credit realization
	servicePeriod := timeutil.ClosedPeriod{
		From: input.Charge.Intent.ServicePeriod.From,
		To:   input.AsOf,
	}

	lastRealizedPeriod := input.Charge.Realizations.Credit.LastRealizedPeriod()
	if lastRealizedPeriod != nil {
		servicePeriod.From = lastRealizedPeriod.To
	}

	h.customerAccountBalance -= amountToRealize.InexactFloat64()

	// Note: we are also realizing the 0 value to make sure that we signify that in that period there was no usage
	// in this setup the ledger transaction should not be created.

	return []charges.CreditRealizationCreateInput{
		{
			Amount:        amountToRealize,
			ServicePeriod: servicePeriod,
		},
	}, nil
}

func (s *ChargesServiceTestSuite) TestCreditThenInvoiceFlow() {
	namespace := "ns-credit-then-invoice-flow"
	ctx := context.Background()
	defer clock.ResetTime()

	customInvoicing := s.SetupCustomInvoicing(namespace)

	cust := s.CreateTestCustomer(namespace, "test")

	_ = s.ProvisionBillingProfile(ctx, namespace, customInvoicing.App.GetID(),
		billingtest.WithProgressiveBilling(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "PT1H")),
		billingtest.WithManualApproval(),
	)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	clock.SetTime(servicePeriod.From)

	const (
		flatFeeName    = "test-flat-fee"
		usageBasedName = "test-usage-based"
	)

	feature := s.SetupApiRequestsTotalFeature(ctx, namespace)
	// Let's add some usage before the period to initialize the streaming connector
	s.MockStreamingConnector.AddSimpleEvent(*feature.Feature.MeterSlug, 100, servicePeriod.From.Add(-time.Hour))

	var (
		flatFeeChargeID    charges.ChargeID
		usageBasedChargeID charges.ChargeID
	)

	handler := &testCreditThenInvoiceHandler{
		charges: s.Charges,
	}
	s.Charges.handler = handler

	s.Run("create new upcoming charges", func() {
		res, err := s.Charges.CreateCharges(ctx, charges.CreateChargeInput{
			Customer: cust.GetID(),
			Currency: currencyx.Code(currency.USD),
			Intents: []charges.CreateChargeIntentInput{
				s.createMockChargeIntent(createMockChargeIntentInput{
					servicePeriod: servicePeriod,
					price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromFloat(2),
					}),
					name:              usageBasedName,
					featureKey:        feature.Feature.Key,
					managedBy:         billing.SubscriptionManagedLine,
					uniqueReferenceID: usageBasedName,
					settlementMode:    productcatalog.CreditThenInvoiceSettlementMode,
				}),
			},
		})
		s.NoError(err)

		s.Len(res, 1)
		s.Equal(res[0].Intent.IntentType, charges.IntentTypeUsageBased)
		s.Equal(res[0].Intent.SettlementMode, productcatalog.CreditThenInvoiceSettlementMode)

		usageBasedChargeID = res[0].GetChargeID()
	})

	s.Run("trigger realization for 2026-01-01", func() {
		asOf := datetime.MustParseTimeInLocation(s.T(), "2026-01-02T01:00:00Z", time.UTC).AsTime()
		clock.SetTime(asOf)

		// Let's register 5 units of usage for the period (cost = 10)
		s.MockStreamingConnector.AddSimpleEvent(*feature.Feature.MeterSlug, 5, asOf.Add(-time.Second))

		handler.realizePercent = 0.5 // Let's realize 50% of the usage, we expect an 5 USD credit realization

		err := s.Charges.TriggerPeriodicRealization(ctx, usageBasedChargeID)
		s.NoError(err)

		usageBasedCharge := s.mustGetCharge(usageBasedChargeID)
		s.Len(usageBasedCharge.Realizations.Credit, 1)
		s.Equal(usageBasedCharge.Realizations.Credit[0].Amount, alpacadecimal.NewFromFloat(5))
		s.Equal(usageBasedCharge.Realizations.Credit[0].ServicePeriod, timeutil.ClosedPeriod{
			From: servicePeriod.From,
			To:   datetime.MustParseTimeInLocation(s.T(), "2026-01-02T00:00:00Z", time.UTC).AsTime(),
		})
	})

	s.Run("trigger realization for 2026-01-02", func() {
		asOf := datetime.MustParseTimeInLocation(s.T(), "2026-01-03T01:00:00Z", time.UTC).AsTime()
		clock.SetTime(asOf)

		err := s.Charges.TriggerPeriodicRealization(ctx, usageBasedChargeID)
		s.NoError(err)

		handler.realizePercent = 0 // Let's not realize any credits this time

		s.expectCreditRealizations(usageBasedChargeID, []creditRealizationExpectation{
			{
				periodFrom: "2026-01-01T00:00:00Z",
				periodTo:   "2026-01-02T00:00:00Z",
				amount:     5,
			},
			{
				periodFrom: "2026-01-02T00:00:00Z",
				periodTo:   "2026-01-03T00:00:00Z",
				amount:     0,
			},
		})
	})

	s.Run("after realization we progressively bill the charge's current amount", func() {
		asOf := datetime.MustParseTimeInLocation(s.T(), "2026-01-03T13:00:00Z", time.UTC).AsTime()
		clock.SetTime(asOf)

		// Let's register 2 units of usage for the period (cost = 4), right before the invoice time
		s.MockStreamingConnector.AddSimpleEvent(*feature.Feature.MeterSlug, 2, asOf.Add(-time.Second))

		handler.realizePercent = 0.5 // Let's pay 50% of the usage from credits

		stdInvoices, err := s.Charges.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     &asOf,
			// TODO: is this needed?!
			ProgressiveBillingOverride: lo.ToPtr(true),
		})
		s.NoError(err)

		s.Len(stdInvoices, 1)
		stdInvoice := stdInvoices[0]
		s.Equal(stdInvoice.Status, billing.StandardInvoiceStatusDraftManualApprovalNeeded)

		lines := stdInvoice.Lines.OrEmpty()
		s.Len(lines, 2)
		stdLineUsageBased := s.getStandardLineByName(lines, usageBasedName)
		s.NotNil(stdLineUsageBased)
		stdLineFlatFee := s.getStandardLineByName(lines, flatFeeName)
		s.NotNil(stdLineFlatFee)

		// TODO: add a test for the flat fee line

		s.Equal(stdLineUsageBased.UsageBased.Quantity, alpacadecimal.NewFromFloat(2))
		s.Equal(stdLineUsageBased.UsageBased.MeteredQuantity, alpacadecimal.NewFromFloat(2))
		s.Equal(*stdLineUsageBased.ChargeID, usageBasedChargeID.ID)

		s.expectCreditRealizations(usageBasedChargeID, []creditRealizationExpectation{
			{
				periodFrom: "2026-01-01T00:00:00Z",
				periodTo:   "2026-01-02T00:00:00Z",
				amount:     5,
			},
			{
				periodFrom: "2026-01-02T00:00:00Z",
				periodTo:   "2026-01-03T00:00:00Z",
				amount:     0,
			},
			{
				periodFrom: "2026-01-03T00:00:00Z",
				periodTo:   "2026-01-04T13:00:00Z",
				amount:     4, // TODO
			},
		})
		usageBasedCharge := s.mustGetCharge(usageBasedChargeID)
		s.Len(usageBasedCharge.Realizations.StandardInvoice, 1)
		// TODO: expect the realization's various values, including the totals / credit allocations etc
	})

	s.Failf("not implemented", "not implemented %v %v", flatFeeChargeID, usageBasedChargeID)
}

type creditRealizationExpectation struct {
	periodFrom string
	periodTo   string
	amount     float64
}

func (s *ChargesServiceTestSuite) expectCreditRealizations(chargeID charges.ChargeID, expectations []creditRealizationExpectation) {
	s.T().Helper()

	charge := s.mustGetCharge(chargeID)

	actual := lo.Map(charge.Realizations.Credit, func(realization charges.CreditRealization, _ int) creditRealizationExpectation {
		return creditRealizationExpectation{
			periodFrom: realization.ServicePeriod.From.Format(time.RFC3339),
			periodTo:   realization.ServicePeriod.To.Format(time.RFC3339),
			amount:     realization.Amount.InexactFloat64(),
		}
	})

	s.T().Logf("actual credit realizations for charge %s:", chargeID)
	for _, realization := range actual {
		s.T().Logf("  - [%s..%s]: %f\n", realization.periodFrom, realization.periodTo, realization.amount)
	}

	s.ElementsMatch(expectations, actual)
}
