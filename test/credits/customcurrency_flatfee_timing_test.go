package credits

import (
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

func (s *CustomCurrencyCreditsSuite) TestFlatFeeCreditThenInvoiceSelectsCreditsAtServiceStart() {
	for _, tc := range []struct {
		name              string
		grantAmount       int64
		grantAfterStart   bool
		expectedAllocated float64
		expectedFiatTotal float64
	}{
		{name: "mid-period credit covers the fee", grantAmount: 20, expectedAllocated: 10},
		{name: "mid-period credit covers part of the fee", grantAmount: 4, expectedAllocated: 4, expectedFiatTotal: 3},
		{name: "later credit cannot fund earlier service", grantAmount: 20, grantAfterStart: true, expectedFiatTotal: 5},
	} {
		s.Run(tc.name, func() {
			t := s.T()
			ctx := t.Context()
			billingStart := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
			servicePeriod := timeutil.ClosedPeriod{From: billingStart.AddDate(0, 0, 14), To: billingStart.AddDate(0, 1, 0)}
			grantAt := servicePeriod.From.Add(-24 * time.Hour)
			if tc.grantAfterStart {
				grantAt = servicePeriod.From.Add(24 * time.Hour)
			}
			clock.FreezeTime(grantAt)
			defer clock.UnFreeze()

			// given: custom credits and a mid-period fee whose invoice_at remains
			// at the billing-period start, as for an in-advance addon change.
			ns := s.GetUniqueNamespace("flat-fee-credit-booking-time")
			defaults := s.ProvisionDefaultTaxCodes(ctx, ns)
			invoicing := s.SetupCustomInvoicing(ns)
			customer := s.CreateLedgerBackedCustomer(ns, "test-subject")
			_ = s.ProvisionBillingProfile(ctx, ns, invoicing.App.GetID(), billingtest.WithManualApproval())
			tokens := s.createCustomCurrency(ns, "TOKENS")
			s.createCustomCurrencyCreditPurchase(ctx, customCurrencyCreditPurchaseInput{
				Namespace:  ns,
				Customer:   customer.GetID(),
				Currency:   tokens,
				Amount:     alpacadecimal.NewFromInt(tc.grantAmount),
				At:         grantAt,
				Name:       "TOKENS grant",
				Settlement: creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{}),
				TaxConfig:  productcatalog.TaxCodeConfig{TaxCodeID: defaults.CreditGrantTaxCodeID},
			})

			collectionAt := servicePeriod.From.Add(48 * time.Hour)
			clock.FreezeTime(collectionAt)
			costBasis := s.newManualCostBasis(alpacadecimal.NewFromFloat(0.5))
			charge := s.createCustomCurrencyFlatFeeCharge(ctx, customCurrencyFlatFeeChargeInput{
				Namespace:      ns,
				Customer:       customer.GetID(),
				Currency:       tokens,
				ServicePeriod:  servicePeriod,
				InvoiceAt:      billingStart,
				Amount:         alpacadecimal.NewFromInt(10),
				PaymentTerm:    productcatalog.InAdvancePaymentTerm,
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				CostBasis:      &costBasis,
				Name:           "Mid-period TOKENS fee",
				TaxConfig:      productcatalog.TaxCodeConfig{TaxCodeID: defaults.InvoicingTaxCodeID},
			})

			// when: collection runs after both the service start and the grant.
			invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
				Customer: customer.GetID(),
				AsOf:     lo.ToPtr(collectionAt),
			})
			s.Require().NoError(err)
			s.Require().Len(invoices, 1)

			// then: only credit effective at the service start is consumed,
			// and only uncovered custom currency becomes invoiceable fiat.
			realized, err := s.MustGetChargeByID(charge.GetChargeID()).AsFlatFeeCharge()
			s.Require().NoError(err)
			s.Require().NotNil(realized.Realizations.CurrentRun)
			s.Equal(tc.expectedAllocated, realized.Realizations.CurrentRun.CreditRealizations.Sum().InexactFloat64())
			s.Equal(tc.expectedFiatTotal, invoices[0].Totals.Total.InexactFloat64())
			accounts := s.mustCustomerAccounts(customer.GetID())
			s.requireAccountBalance(accounts.FBOAccount, ledger.RouteFilter{Currency: tokens.Reference()}, float64(tc.grantAmount)-tc.expectedAllocated, "remaining TOKENS")
		})
	}
}
