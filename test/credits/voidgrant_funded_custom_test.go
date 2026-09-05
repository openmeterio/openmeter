package credits

import (
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/creditgrant"
	"github.com/openmeterio/openmeter/openmeter/ent/db/ledgerentry"
	"github.com/openmeterio/openmeter/openmeter/ent/db/ledgersubaccount"
	"github.com/openmeterio/openmeter/openmeter/ent/db/ledgersubaccountroute"
	"github.com/openmeterio/openmeter/openmeter/ent/db/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgerbreakage "github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
)

func (s *VoidGrantTestSuite) TestVoidFundedCustomCurrencyGrantAtPaymentStages() {
	for _, initial := range []creditpurchase.InitialPaymentSettlementStatus{
		creditpurchase.CreatedInitialPaymentSettlementStatus,
		creditpurchase.AuthorizedInitialPaymentSettlementStatus,
		creditpurchase.SettledInitialPaymentSettlementStatus,
	} {
		s.Run(string(initial), func() {
			// given: expiring custom credits funded at each external payment stage
			ctx := s.T().Context()
			ns := s.GetUniqueNamespace("funded-custom-void-" + string(initial))
			cust := s.setupVoidTestCustomer(ctx, ns)
			tokens := s.CreateCustomCurrency(ns, "TOKENS")
			at := time.Date(2026, 4, 17, 11, 0, 0, 0, time.UTC)
			clock.FreezeTime(at)
			defer clock.UnFreeze()
			rate := alpacadecimal.RequireFromString("0.5")
			grant, err := s.CreditGrantService.Create(ctx, creditgrant.CreateInput{
				Namespace: ns, CustomerID: cust.ID, Name: "funded custom grant",
				Currency: tokens.GetCode(), Amount: alpacadecimal.NewFromInt(100),
				ExpiresAfter:  lo.ToPtr(datetime.MustParseDuration(s.T(), "P1D")),
				FundingMethod: creditgrant.FundingMethodExternal,
				Purchase:      &creditgrant.PurchaseTerms{Currency: USD, PerUnitCostBasis: &rate, AvailabilityPolicy: &initial},
			})
			s.Require().NoError(err)
			route := ledger.RouteFilter{Currency: tokens.Reference(), CostBasisCurrency: mo.Some(lo.ToPtr(USD)), CostBasis: mo.Some(&rate)}
			s.Equal(float64(100), s.MustCustomerFBOBalanceByRouteAsOf(cust.GetID(), route, lo.ToPtr(clock.Now())).InexactFloat64())
			plansInput := ledgerbreakage.ListPlansInput{CustomerID: cust.GetID(), Currency: tokens.GetCode(), AsOf: at}
			plans, err := s.BreakageService.ListPlans(ctx, plansInput)
			s.Require().NoError(err)
			s.Require().Len(plans, 1)
			fiatEntries := s.DBClient.LedgerEntry.Query().Where(
				ledgerentry.NamespaceEQ(ns),
				ledgerentry.HasSubAccountWith(ledgersubaccount.HasRouteWith(ledgersubaccountroute.CurrencyEQ(string(USD)))),
			)
			paymentEntriesBefore, err := fiatEntries.Clone().IDs(ctx)
			s.Require().NoError(err)

			// when: the grant is voided and retried before its original expiry
			clock.FreezeTime(at.Add(time.Hour))
			voidInput := creditgrant.VoidInput{Namespace: ns, CustomerID: cust.ID, ChargeID: grant.ID}
			voided, err := s.CreditGrantService.Void(ctx, voidInput)
			s.Require().NoError(err)
			s.Require().NotNil(voided.State.VoidedAt)
			postings := s.DBClient.LedgerTransaction.Query().Where(ledgertransaction.NamespaceEQ(ns))
			postingCount, err := postings.Clone().Count(ctx)
			s.Require().NoError(err)
			retried, err := s.CreditGrantService.Void(ctx, voidInput)
			s.Require().NoError(err)

			// then: credit and expiry are removed once, without refunding payment
			s.Equal(voided.State.VoidedAt, retried.State.VoidedAt)
			s.Equal(grant.Realizations.ExternalPaymentSettlement, voided.Realizations.ExternalPaymentSettlement)
			s.Equal(grant.Realizations.ExternalPaymentSettlement, retried.Realizations.ExternalPaymentSettlement)
			s.Equal(grant.Status, retried.Status)
			paymentEntriesAfter, err := fiatEntries.Clone().IDs(ctx)
			s.Require().NoError(err)
			s.ElementsMatch(paymentEntriesBefore, paymentEntriesAfter, "void must not change fiat payment ledger state")
			retriedPostingCount, err := postings.Clone().Count(ctx)
			s.Require().NoError(err)
			s.Equal(postingCount, retriedPostingCount, "retry must not post another correction")
			s.Equal(float64(0), s.MustCustomerFBOBalanceByRouteAsOf(cust.GetID(), route, lo.ToPtr(clock.Now())).InexactFloat64())
			plansInput.AsOf = clock.Now()
			plans, err = s.BreakageService.ListPlans(ctx, plansInput)
			s.Require().NoError(err)
			s.Empty(plans, "void must release the planned expiry")

			clock.FreezeTime(at.Add(25 * time.Hour))
			s.Equal(float64(0), s.MustCustomerFBOBalanceByRouteAsOf(cust.GetID(), route, lo.ToPtr(clock.Now())).InexactFloat64(), "expiry must not withdraw credit twice")
			businessAccounts, err := s.LedgerResolver.GetBusinessAccounts(ctx, ns)
			s.Require().NoError(err)
			breakage, err := s.BalanceQuerier.GetAccountBalance(ctx, businessAccounts.BreakageAccount, route, ledger.BalanceQuery{AsOf: lo.ToPtr(clock.Now())})
			s.Require().NoError(err)
			s.Equal(float64(0), breakage.InexactFloat64())
		})
	}
}
