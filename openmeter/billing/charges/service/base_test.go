package service

import (
	"errors"
	"log/slog"

	"github.com/invopop/gobl/currency"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/adapter"
	creditpurchaseadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase/adapter"
	creditpurchaseservice "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase/service"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	flatfeeadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee/adapter"
	flatfeeservice "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee/service"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	metaadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/meta/adapter"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
	"github.com/samber/lo"
)

const USD = currencyx.Code(currency.USD)

type BaseSuite struct {
	billingtest.BaseSuite

	Charges                   *service
	FlatFeeTestHandler        *flatFeeTestHandler
	CreditPurchaseTestHandler *creditPurchaseTestHandler
}

func (s *BaseSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()

	s.FlatFeeTestHandler = newFlatFeeTestHandler()
	s.CreditPurchaseTestHandler = newCreditPurchaseTestHandler()

	metaAdapter, err := metaadapter.New(metaadapter.Config{
		Client: s.DBClient,
		Logger: slog.Default(),
	})
	s.NoError(err)

	flatFeeAdapter, err := flatfeeadapter.New(flatfeeadapter.Config{
		Client:      s.DBClient,
		Logger:      slog.Default(),
		MetaAdapter: metaAdapter,
	})
	s.NoError(err)

	flatFeeService, err := flatfeeservice.New(flatfeeservice.Config{
		Adapter:     flatFeeAdapter,
		Handler:     s.FlatFeeTestHandler,
		MetaAdapter: metaAdapter,
	})
	s.NoError(err)

	creditPurchaseAdapter, err := creditpurchaseadapter.New(creditpurchaseadapter.Config{
		Client:      s.DBClient,
		Logger:      slog.Default(),
		MetaAdapter: metaAdapter,
	})
	s.NoError(err)

	creditPurchaseService, err := creditpurchaseservice.New(creditpurchaseservice.Config{
		Adapter:     creditPurchaseAdapter,
		Handler:     s.CreditPurchaseTestHandler,
		MetaAdapter: metaAdapter,
	})
	s.NoError(err)

	chargesAdapter, err := adapter.New(adapter.Config{
		Client: s.DBClient,
		Logger: slog.Default(),
	})
	s.NoError(err)

	chargesService, err := New(Config{
		Adapter: chargesAdapter,

		MetaAdapter:           metaAdapter,
		FlatFeeService:        flatFeeService,
		CreditPurchaseService: creditPurchaseService,

		BillingService: s.BillingService,
	})
	s.NoError(err)
	s.Charges = chargesService
}

func (s *BaseSuite) TeardownTest() {
	s.FlatFeeTestHandler.Reset()
	s.CreditPurchaseTestHandler.Reset()
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

func (s *BaseSuite) createMockChargeIntent(input createMockChargeIntentInput) charges.ChargeIntent {
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

	intentMeta := meta.Intent{
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

		flatFeeIntent := flatfee.Intent{
			Intent:         intentMeta,
			PaymentTerm:    price.PaymentTerm,
			FeatureKey:     input.featureKey,
			InvoiceAt:      invoiceAt,
			SettlementMode: lo.CoalesceOrEmpty(input.settlementMode, productcatalog.InvoiceOnlySettlementMode),

			AmountBeforeProration: price.Amount,
			AmountAfterProration:  price.Amount,
		}
		return charges.NewChargeIntent(flatFeeIntent)
	}

	s.FailNow("not implemented: usage based intents")

	return charges.ChargeIntent{}
}

func (s *BaseSuite) mustGetChargeByID(chargeID meta.ChargeID) charges.Charge {
	s.T().Helper()
	charge, err := s.Charges.GetByID(s.T().Context(), charges.GetByIDInput{
		ChargeID: chargeID,
		Expands: meta.Expands{
			meta.ExpandRealizations,
		},
	})
	s.NoError(err)
	return charge
}
