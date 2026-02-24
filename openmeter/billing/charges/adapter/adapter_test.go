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

	feeIntent := charges.FlatFeeIntent{
		IntentMeta:            newIntentMeta(customer.ID, periodStart, periodEnd, "Flat Fee Charge"),
		InvoiceAt:             periodEnd,
		SettlementMode:        productcatalog.InvoiceOnlySettlementMode,
		PaymentTerm:           productcatalog.InArrearsPaymentTerm,
		AmountBeforeProration: alpacadecimal.NewFromFloat(100),
		AmountAfterProration:  alpacadecimal.NewFromFloat(100),
		ProRating: productcatalog.ProRatingConfig{
			Enabled: false,
			Mode:    productcatalog.ProRatingModeProratePrices,
		},
	}

	created, err := s.adapter.CreateCharges(ctx, charges.CreateChargeInputs{
		Namespace: ns,
		Intents:   charges.ChargeIntents{charges.NewChargeIntent(feeIntent)},
	})
	s.NoError(err)
	s.Len(created, 1)

	ffCreated, err := created[0].AsFlatFeeCharge()
	s.NoError(err)
	s.NotEmpty(ffCreated.ManagedResource.ID)

	fetched, err := s.adapter.GetChargesByIDs(ctx, ns, []string{ffCreated.ManagedResource.ID})
	s.NoError(err)
	s.Len(fetched, 1)
	ffFetched, err := fetched[0].AsFlatFeeCharge()
	s.NoError(err)

	s.Equal(ffCreated.Intent, ffFetched.Intent)
	s.Equal(charges.ChargeStatusActive, ffFetched.Status)
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

	usageIntent := charges.UsageBasedIntent{
		IntentMeta:     newIntentMeta(customer.ID, periodStart, periodEnd, "Usage Based Charge"),
		Price:          *price,
		FeatureKey:     "api-requests",
		InvoiceAt:      periodEnd,
		SettlementMode: productcatalog.InvoiceOnlySettlementMode,
	}

	created, err := s.adapter.CreateCharges(ctx, charges.CreateChargeInputs{
		Namespace: ns,
		Intents:   charges.ChargeIntents{charges.NewChargeIntent(usageIntent)},
	})
	s.NoError(err)
	s.Len(created, 1)

	ubCreated, err := created[0].AsUsageBasedCharge()
	s.NoError(err)
	s.NotEmpty(ubCreated.ManagedResource.ID)

	fetched, err := s.adapter.GetChargesByIDs(ctx, ns, []string{ubCreated.ManagedResource.ID})
	s.NoError(err)
	s.Len(fetched, 1)
	ubFetched, err := fetched[0].AsUsageBasedCharge()
	s.NoError(err)

	s.Equal(ubCreated.Intent, ubFetched.Intent)
	s.Equal(charges.ChargeStatusActive, ubFetched.Status)
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

	cpIntent := charges.CreditPurchaseIntent{
		IntentMeta:   newIntentMeta(customer.ID, periodStart, periodEnd, "Credit Purchase Charge"),
		CreditAmount: alpacadecimal.NewFromFloat(100),
		Settlement:   settlement,
	}

	created, err := s.adapter.CreateCharges(ctx, charges.CreateChargeInputs{
		Namespace: ns,
		Intents:   charges.ChargeIntents{charges.NewChargeIntent(cpIntent)},
	})
	s.NoError(err)
	s.Len(created, 1)

	cpCreated, err := created[0].AsCreditPurchaseCharge()
	s.NoError(err)
	s.NotEmpty(cpCreated.ManagedResource.ID)

	fetched, err := s.adapter.GetChargesByIDs(ctx, ns, []string{cpCreated.ManagedResource.ID})
	s.NoError(err)
	s.Len(fetched, 1)
	cpFetched, err := fetched[0].AsCreditPurchaseCharge()
	s.NoError(err)

	s.Equal(cpCreated.Intent, cpFetched.Intent)
	s.Equal(charges.ChargeStatusActive, cpFetched.Status)
	s.Equal(cpCreated.State, cpFetched.State)
}

func newManagedResource(ns string) charges.ManagedResource {
	mr := charges.ManagedResource{}
	mr.Namespace = ns

	return mr
}

func newIntentMeta(customerID string, periodStart, periodEnd time.Time, name string) charges.IntentMeta {
	return charges.IntentMeta{
		Name:       name,
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
