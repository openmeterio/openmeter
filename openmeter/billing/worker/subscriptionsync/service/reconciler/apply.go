package reconciler

import (
	"context"
	"fmt"
)

func (s *Service) Apply(ctx context.Context, input ApplyInput) error {
	patches := make([]Patch, 0, len(input.Plan.SemanticPatches))

	for _, semanticPatch := range input.Plan.SemanticPatches {
		expanded, err := semanticPatch.Expand(ctx, ExpandInput{
			Subscription: input.Subscription,
			Currency:     input.Currency,
			Invoices:     input.Invoices,
		})
		if err != nil {
			return fmt.Errorf("expanding semantic patch[%s/%s]: %w", semanticPatch.Operation(), semanticPatch.UniqueReferenceID(), err)
		}

		patches = append(patches, expanded...)
	}

	invoiceUpdater := NewInvoiceUpdater(s.billingService, s.logger)
	if err := invoiceUpdater.ApplyPatches(ctx, input.Customer, patches); err != nil {
		return fmt.Errorf("updating invoices: %w", err)
	}

	return nil
}
