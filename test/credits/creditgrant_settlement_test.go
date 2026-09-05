package credits

import (
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/billing/creditgrant"
	"github.com/openmeterio/openmeter/openmeter/ent/db/ledgertransaction"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func (s *CreditGrantTestSuite) TestRejectPurchaseRoundingToZeroBeforeIssuance() {
	for _, code := range []currencyx.Code{"TOKENS", USD} {
		for _, funding := range []creditgrant.FundingMethod{creditgrant.FundingMethodExternal, creditgrant.FundingMethodInvoice} {
			s.Run(string(code)+"/"+string(funding), func() {
				// given a positive purchase price below the settlement currency's precision
				ctx := s.T().Context()
				ns := s.GetUniqueNamespace("rounded-zero-purchase")
				s.ProvisionDefaultTaxCodes(ctx, ns)
				cust := s.CreateLedgerBackedCustomer(ns, "customer")
				if code.IsCustom() {
					s.CreateCustomCurrency(ns, code)
				}
				if funding == creditgrant.FundingMethodInvoice {
					invoicing := s.SetupCustomInvoicing(ns)
					s.ProvisionBillingProfile(ctx, ns, invoicing.App.GetID())
				}
				clock.FreezeTime(time.Date(2026, 4, 17, 11, 0, 0, 0, time.UTC))
				defer clock.UnFreeze()

				// when creating paid credits that would otherwise be available immediately
				_, err := s.CreditGrantService.Create(ctx, creditgrant.CreateInput{
					Namespace: ns, CustomerID: cust.ID, Name: "small purchase", Currency: code,
					Amount: alpacadecimal.NewFromInt(1), FundingMethod: funding,
					Purchase: &creditgrant.PurchaseTerms{
						Currency: USD, PerUnitCostBasis: lo.ToPtr(alpacadecimal.RequireFromString("0.001")),
						AvailabilityPolicy: lo.ToPtr(creditpurchase.CreatedInitialPaymentSettlementStatus),
					},
				})

				// then rejection is atomic: no grant, ledger posting or invoice remains
				s.Require().ErrorContains(err, "purchase amount must be positive after rounding")
				grants, err := s.CreditGrantService.List(ctx, creditgrant.ListInput{Namespace: ns, CustomerID: cust.ID})
				s.Require().NoError(err)
				s.Empty(grants.Items)
				postings, err := s.DBClient.LedgerTransaction.Query().Where(ledgertransaction.NamespaceEQ(ns)).Count(ctx)
				s.Require().NoError(err)
				s.Zero(postings)
				invoices, err := s.BillingService.ListStandardInvoices(ctx, billing.ListStandardInvoicesInput{Namespace: ns})
				s.Require().NoError(err)
				s.Empty(invoices.Items)
			})
		}
	}
}

func (s *CreditGrantTestSuite) TestPurchaseRoundingToMinorUnitCanSettle() {
	// given a fractional price that rounds up to one fiat minor unit
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("rounded-positive-purchase")
	s.ProvisionDefaultTaxCodes(ctx, ns)
	cust := s.CreateLedgerBackedCustomer(ns, "customer")
	s.CreateCustomCurrency(ns, "TOKENS")
	clock.FreezeTime(time.Date(2026, 4, 17, 11, 0, 0, 0, time.UTC))
	defer clock.UnFreeze()

	// when the custom purchase is created and settled
	grant, err := s.CreditGrantService.Create(ctx, creditgrant.CreateInput{
		Namespace: ns, CustomerID: cust.ID, Name: "one cent purchase", Currency: "TOKENS",
		Amount: alpacadecimal.NewFromInt(1), FundingMethod: creditgrant.FundingMethodExternal,
		Purchase: &creditgrant.PurchaseTerms{Currency: USD, PerUnitCostBasis: lo.ToPtr(alpacadecimal.RequireFromString("0.005"))},
	})
	s.Require().NoError(err)
	settled, err := s.CreditGrantService.UpdateExternalSettlement(ctx, creditgrant.UpdateExternalSettlementInput{
		Namespace: ns, CustomerID: cust.ID, ChargeID: grant.ID, TargetStatus: payment.StatusSettled,
	})

	// then valid rounded purchases retain the normal payment lifecycle
	s.Require().NoError(err)
	s.Equal(creditpurchase.StatusFinal, settled.Status)
	s.Require().NotNil(settled.Realizations.ExternalPaymentSettlement)
	s.Equal(0.01, settled.Realizations.ExternalPaymentSettlement.FiatAmount.InexactFloat64())
}
