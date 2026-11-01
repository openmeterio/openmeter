package credits

import (
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func (s *CreditGrantTestSuite) TestListCustomerBalances() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("customerbalance-list")
	s.TaxCodeEnv.ProvisionDefaultTaxCodes(s.T(), ns)
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())

	start := datetime.MustParseTimeInLocation(s.T(), "2026-09-30T10:00:00Z", time.UTC).AsTime()
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	// given: a funded customer, a customer without ledger accounts, and a funded
	// customer deleted after the close.
	funded := s.CreateLedgerBackedCustomer(ns, "funded")
	noLedger := s.CreateTestCustomer(ns, "no-ledger")
	deleted := s.CreateLedgerBackedCustomer(ns, "deleted")
	s.mustCreatePromotionalCreditGrant(ctx, ns, funded.GetID(), "funded-grant", alpacadecimal.NewFromInt(10))
	s.mustCreatePromotionalCreditGrant(ctx, ns, deleted.GetID(), "deleted-grant", alpacadecimal.NewFromInt(20))

	closeAt := start.Add(time.Minute)

	clock.FreezeTime(start.Add(time.Hour))
	defer clock.UnFreeze()
	s.Require().NoError(s.CustomerService.DeleteCustomer(ctx, deleted.GetID()))

	clock.FreezeTime(start.Add(2 * time.Hour))
	defer clock.UnFreeze()

	facade, err := customerbalance.NewFacade(s.CustomerBalanceSvc)
	s.Require().NoError(err)
	lister, err := customerbalance.NewLister(s.CustomerService, facade)
	s.Require().NoError(err)

	booked := func(rows []customerbalance.CustomerBalances) map[string][]float64 {
		return lo.SliceToMap(rows, func(row customerbalance.CustomerBalances) (string, []float64) {
			return row.CustomerID.ID, lo.Map(row.Balances, func(b customerbalance.BalanceByCurrency, _ int) float64 {
				return b.Balance.Booked().InexactFloat64()
			})
		})
	}

	s.Run("a past close keeps the deleted customer and lists the one without ledger accounts as empty", func() {
		// when: the close is re-read at its original time.
		result, err := lister.ListCustomerBalances(ctx, customerbalance.ListCustomerBalancesInput{
			Namespace: ns,
			Page:      pagination.NewPage(1, 100),
			AsOf:      &closeAt,
		})
		s.Require().NoError(err)

		// then: every customer that existed at the close is listed, in ID order.
		s.Equal(3, result.TotalCount)
		s.True(lo.IsSortedByKey(result.Items, func(row customerbalance.CustomerBalances) string { return row.CustomerID.ID }))
		s.Equal(map[string][]float64{
			funded.ID:   {10},
			noLedger.ID: {},
			deleted.ID:  {20},
		}, booked(result.Items))
	})

	s.Run("a live read leaves the deleted customer out", func() {
		result, err := lister.ListCustomerBalances(ctx, customerbalance.ListCustomerBalancesInput{
			Namespace: ns,
			Page:      pagination.NewPage(1, 100),
		})
		s.Require().NoError(err)

		s.Equal(map[string][]float64{funded.ID: {10}, noLedger.ID: {}}, booked(result.Items))
	})

	s.Run("customer IDs limit the listing", func() {
		result, err := lister.ListCustomerBalances(ctx, customerbalance.ListCustomerBalancesInput{
			Namespace:   ns,
			Page:        pagination.NewPage(1, 100),
			CustomerIDs: []string{deleted.ID},
			AsOf:        &closeAt,
		})
		s.Require().NoError(err)

		s.Equal(map[string][]float64{deleted.ID: {20}}, booked(result.Items))
	})
}
