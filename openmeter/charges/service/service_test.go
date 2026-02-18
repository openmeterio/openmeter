package service

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/charges"
	"github.com/openmeterio/openmeter/openmeter/charges/adapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
	"github.com/stretchr/testify/suite"
)

type ChargesServiceTestSuite struct {
	billingtest.BaseSuite

	Charges charges.Service
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
		Adapter:        chargesAdapter,
		BillingService: s.BillingService,
	})
	s.NoError(err)
	s.Charges = chargesService
}

func (s *ChargesServiceTestSuite) TestChargeInvoiceOnlyFlow() {
	namespace := "ns-charges-service"
	ctx := context.Background()

	sandboxApp := s.InstallSandboxApp(s.T(), namespace)

	cust := s.CreateTestCustomer(namespace, "test")

	_ = s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID(),
		billingtest.WithProgressiveBilling(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "PT1H")),
		billingtest.WithManualApproval(),
	)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}

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
					name:      "test-flat-fee",
					managedBy: billing.SubscriptionManagedLine,
				}),
				s.createMockChargeIntent(createMockChargeIntentInput{
					servicePeriod: servicePeriod,
					price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromFloat(2),
					}),
					name:       "test-usage-based",
					featureKey: "test-usage-based",
					managedBy:  billing.SubscriptionManagedLine,
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
	})
}

type createMockChargeIntentInput struct {
	servicePeriod timeutil.ClosedPeriod
	price         *productcatalog.Price
	featureKey    string
	name          string
	managedBy     billing.InvoiceLineManagedBy
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
