package testutils

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/legacylineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
)

// LineageWithAllocations retains the allocation references supplied by direct
// ledger/handler tests, which do not persist charge runs. Lineage state itself
// still uses the real database-backed service. Full lifecycle tests load these
// references from the charge allocation tables.
type LineageWithAllocations struct {
	legacylineage.Service
	allocations map[string]creditrealization.Realization
}

func (s *LineageWithAllocations) CreateInitialLineages(ctx context.Context, input legacylineage.CreateInitialLineagesInput) error {
	if err := s.Service.CreateInitialLineages(ctx, input); err != nil {
		return err
	}
	if s.allocations == nil {
		s.allocations = make(map[string]creditrealization.Realization)
	}
	for _, realization := range input.Realizations {
		s.allocations[realization.ID] = realization
	}
	return nil
}

func (s *LineageWithAllocations) LoadLineagesByCustomer(ctx context.Context, input legacylineage.LoadLineagesByCustomerInput) ([]legacylineage.Lineage, error) {
	roots, err := s.Service.LoadLineagesByCustomer(ctx, input)
	if err != nil {
		return nil, err
	}
	for i := range roots {
		if allocation, ok := s.allocations[roots[i].RootRealizationID]; ok {
			roots[i].OriginalTransactionGroupID = allocation.LedgerTransaction.TransactionGroupID
			roots[i].OriginalAllocationSortHint = allocation.SortHint
		}
	}
	return roots, nil
}
