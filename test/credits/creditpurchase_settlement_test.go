package credits

import (
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func (s *CreditPurchaseCostBasisSuite) TestDynamicInvoicePurchaseRoundingToZeroIsAtomic() {
	// given an unresolved dynamic purchase that will round below one fiat minor unit
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("dynamic-zero-purchase")
	clock.FreezeTime(time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC))
	defer clock.UnFreeze()
	invoicing := s.SetupCustomInvoicing(ns)
	s.ProvisionBillingProfile(ctx, ns, invoicing.App.GetID())
	fixture := s.provisionDynamicCreditPurchaseFixture(ns)
	fixture.creditAmount = alpacadecimal.RequireFromString("0.001")
	clock.FreezeTime(fixture.servicePeriod.From)

	created, err := s.creditPurchaseService.Create(ctx, creditpurchase.CreateInput{
		Namespace: ns, Intent: fixture.newIntent(creditpurchase.NewInvoiceSettlement()),
	})
	s.Require().NoError(err)
	s.Require().NotNil(created.GatheringLineToCreate)
	s.Nil(created.Charge.State.ResolvedCostBasis)
	_, err = s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
		Customer: fixture.customerID, Currency: currencyx.FiatCode(fixture.settlementCurrency),
		Lines: []billing.GatheringLine{*created.GatheringLineToCreate},
	})
	s.Require().NoError(err)

	// when invoicing resolves the actual purchase price
	asOf := clock.Now()
	_, err = s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: fixture.customerID, AsOf: &asOf,
	})

	// then the failed activation leaves the pending intent unpriced and grants no credit
	s.Require().ErrorContains(err, "purchase amount must be positive after rounding")
	charge, err := s.MustGetChargeByID(created.Charge.GetChargeID()).AsCreditPurchaseCharge()
	s.Require().NoError(err)
	s.Equal(creditpurchase.StatusCreated, charge.Status)
	s.Nil(charge.State.ResolvedCostBasis)
	s.Nil(charge.Realizations.CreditGrantRealization)
	s.Equal(float64(0), s.mustAccountBalance(fixture.customerAccounts.FBOAccount, ledger.RouteFilter{
		Currency: fixture.customCurrency.Reference(), CostBasisCurrency: mo.Some(&fixture.settlementCurrency),
		CostBasis: mo.Some(&fixture.resolvedRate), CreditPriority: lo.ToPtr(ledger.DefaultCustomerFBOPriority),
	}).InexactFloat64())
	invoices, err := s.BillingService.ListStandardInvoices(ctx, billing.ListStandardInvoicesInput{Namespace: ns})
	s.Require().NoError(err)
	s.Empty(invoices.Items)
}
