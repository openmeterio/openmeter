package customerbalance

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	chargemeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// pageRecordingChargeService serves a fixed charge list with the charges
// search adapter's paging semantics (a zero page returns every row; any other
// page returns one slice and costs a COUNT query) and records each request.
type pageRecordingChargeService struct {
	items charges.Charges
	pages *[]pagination.Page
}

func (s pageRecordingChargeService) GetByIDs(context.Context, charges.GetByIDsInput) (charges.Charges, error) {
	return nil, nil
}

func (s pageRecordingChargeService) ListCharges(_ context.Context, input charges.ListChargesInput) (pagination.Result[charges.Charge], error) {
	*s.pages = append(*s.pages, input.Page)

	if input.Page.IsZero() {
		return pagination.Result[charges.Charge]{Items: s.items, TotalCount: len(s.items)}, nil
	}

	from := min(input.Page.Offset(), len(s.items))
	to := min(from+input.Page.Limit(), len(s.items))

	return pagination.Result[charges.Charge]{Page: input.Page, Items: s.items[from:to], TotalCount: len(s.items)}, nil
}

// Balance reads list every candidate charge, so they must not page: each page
// of a paged list runs COUNT(*) over charges_search_v1s (#5274).
func TestBalanceChargeCandidatesAreListedInOneUnpagedQuery(t *testing.T) {
	customerID := customer.CustomerID{Namespace: "ns", ID: "customer-1"}
	const n = 2*chargeListPageSize + 50

	flatFees := make(charges.Charges, 0, n)
	creditPurchases := make(charges.Charges, 0, n)
	for i := range n {
		flatFees = append(flatFees, charges.NewCharge(flatfee.Charge{
			ChargeBase: flatfee.ChargeBase{
				ManagedResource: chargemeta.ManagedResource{ID: fmt.Sprintf("flat-fee-%d", i)},
			},
		}))
		creditPurchases = append(creditPurchases, charges.NewCharge(creditpurchase.Charge{
			ChargeBase: creditpurchase.ChargeBase{
				ManagedResource: chargemeta.ManagedResource{ID: fmt.Sprintf("credit-purchase-%d", i)},
			},
		}))
	}

	tests := []struct {
		name  string
		items charges.Charges
		list  func(*service) ([]charges.Charge, error)
	}{
		{
			name:  "live balance candidates",
			items: flatFees,
			list: func(s *service) ([]charges.Charge, error) {
				return s.listChargeLiveBalanceCandidates(t.Context(), customerID)
			},
		},
		{
			name:  "pending grant candidates",
			items: creditPurchases,
			list: func(s *service) ([]charges.Charge, error) {
				return s.listPendingGrantCandidateCharges(t.Context(), customerID)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var pages []pagination.Page
			s := &service{ChargesService: pageRecordingChargeService{items: tc.items, pages: &pages}}

			got, err := tc.list(s)
			require.NoError(t, err)

			require.Equal(t, []pagination.Page{{}}, pages, "expected a single unpaged ListCharges call")
			require.Len(t, got, n)
			require.Equal(t, tc.items[n-1], got[n-1])
		})
	}
}
