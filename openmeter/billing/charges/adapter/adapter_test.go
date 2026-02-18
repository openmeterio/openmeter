package adapter_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	chargesadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/adapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

type ChargesAdapterTestSuite struct {
	billingtest.BaseSuite

	adapter charges.Adapter
}

func TestChargesAdapter(t *testing.T) {
	suite.Run(t, new(ChargesAdapterTestSuite))
}

func (s *ChargesAdapterTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()

	a, err := chargesadapter.New(chargesadapter.Config{
		Client: s.DBClient,
		Logger: slog.Default(),
	})
	s.NoError(err)
	s.adapter = a
}

func (s *ChargesAdapterTestSuite) TestCreateAndGetFlatFeeCharge() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("charges-adapter-flatfee")

	customer := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(customer.ID)

	now := time.Now().UTC().Truncate(time.Millisecond)
	periodStart := now.Add(-30 * 24 * time.Hour)
	periodEnd := now

	input := charges.NewCharge(charges.FlatFeeCharge{
		ManagedResource: newManagedResource(ns, "Flat Fee Charge"),
		Status:          charges.ChargeStatusActive,
		Intent: charges.FlatFeeIntent{
			IntentMeta:            newIntentMeta(customer.ID, periodStart, periodEnd),
			InvoiceAt:             periodEnd,
			SettlementMode:        productcatalog.InvoiceOnlySettlementMode,
			PaymentTerm:           productcatalog.InArrearsPaymentTerm,
			AmountBeforeProration: alpacadecimal.NewFromFloat(100),
			AmountAfterProration:  alpacadecimal.NewFromFloat(100),
			ProRating: productcatalog.ProRatingConfig{
				Enabled: false,
				Mode:    productcatalog.ProRatingModeProratePrices,
			},
		},
		State: charges.FlatFeeState{},
	})

	created, err := s.adapter.CreateCharges(ctx, []charges.Charge{input})
	s.NoError(err)
	s.Len(created, 1)

	gc, err := created[0].AsGenericCharge()
	s.NoError(err)
	s.NotEmpty(gc.GetManagedResource().ID)

	fetched, err := s.adapter.GetChargesByIDs(ctx, ns, []string{gc.GetManagedResource().ID})
	s.NoError(err)
	s.Len(fetched, 1)

	s.Equal(created[0], fetched[0])
}

func (s *ChargesAdapterTestSuite) TestCreateAndGetUsageBasedCharge() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("charges-adapter-usagebased")

	customer := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(customer.ID)

	now := time.Now().UTC().Truncate(time.Millisecond)
	periodStart := now.Add(-30 * 24 * time.Hour)
	periodEnd := now

	price := productcatalog.NewPriceFrom(productcatalog.FlatPrice{
		Amount:      alpacadecimal.NewFromFloat(10),
		PaymentTerm: productcatalog.InArrearsPaymentTerm,
	})

	input := charges.NewCharge(charges.UsageBasedCharge{
		ManagedResource: newManagedResource(ns, "Usage Based Charge"),
		Status:          charges.ChargeStatusActive,
		Intent: charges.UsageBasedIntent{
			IntentMeta:     newIntentMeta(customer.ID, periodStart, periodEnd),
			Price:          *price,
			FeatureKey:     "api-requests",
			InvoiceAt:      periodEnd,
			SettlementMode: productcatalog.InvoiceOnlySettlementMode,
		},
		State: charges.UsageBasedState{},
	})

	created, err := s.adapter.CreateCharges(ctx, []charges.Charge{input})
	s.NoError(err)
	s.Len(created, 1)

	gc, err := created[0].AsGenericCharge()
	s.NoError(err)
	s.NotEmpty(gc.GetManagedResource().ID)

	fetched, err := s.adapter.GetChargesByIDs(ctx, ns, []string{gc.GetManagedResource().ID})
	s.NoError(err)
	s.Len(fetched, 1)

	s.Equal(created[0], fetched[0])
}

func (s *ChargesAdapterTestSuite) TestCreateAndGetCreditPurchaseCharge() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("charges-adapter-creditpurchase")

	customer := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(customer.ID)

	now := time.Now().UTC().Truncate(time.Millisecond)
	periodStart := now.Add(-30 * 24 * time.Hour)
	periodEnd := now

	settlement := charges.NewCreditPurchaseSettlement(charges.InvoiceCreditPurchaseSettlement{
		GenericCreditPurchaseSettlement: charges.GenericCreditPurchaseSettlement{
			SettlementCurrency: currencyx.Code(currency.USD),
			CostBasis:          alpacadecimal.NewFromFloat(50),
		},
	})

	input := charges.NewCharge(charges.CreditPurchaseCharge{
		ManagedResource: newManagedResource(ns, "Credit Purchase Charge"),
		Status:          charges.ChargeStatusActive,
		Intent: charges.CreditPurchaseIntent{
			IntentMeta:   newIntentMeta(customer.ID, periodStart, periodEnd),
			Currency:     currencyx.Code(currency.USD),
			CreditAmount: alpacadecimal.NewFromFloat(100),
			Settlement:   settlement,
		},
		State: charges.CreditPurchaseState{
			Status: charges.AuthorizedPaymentSettlementStatus,
		},
	})

	created, err := s.adapter.CreateCharges(ctx, []charges.Charge{input})
	s.NoError(err)
	s.Len(created, 1)

	gc, err := created[0].AsGenericCharge()
	s.NoError(err)
	s.NotEmpty(gc.GetManagedResource().ID)

	fetched, err := s.adapter.GetChargesByIDs(ctx, ns, []string{gc.GetManagedResource().ID})
	s.NoError(err)
	s.Len(fetched, 1)

	s.Equal(created[0], fetched[0])
}

func newManagedResource(ns, name string) models.ManagedResource {
	mr := models.ManagedResource{}
	mr.Namespace = ns
	mr.Name = name

	return mr
}

func newIntentMeta(customerID string, periodStart, periodEnd time.Time) charges.IntentMeta {
	return charges.IntentMeta{
		ManagedBy:  billing.ManuallyManagedLine,
		CustomerID: customerID,
		Currency:   currencyx.Code(currency.USD),
		ServicePeriod: timeutil.ClosedPeriod{
			From: periodStart,
			To:   periodEnd,
		},
		FullServicePeriod: timeutil.ClosedPeriod{
			From: periodStart,
			To:   periodEnd,
		},
		BillingPeriod: timeutil.ClosedPeriod{
			From: periodStart,
			To:   periodEnd,
		},
	}
}
