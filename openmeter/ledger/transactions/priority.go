package transactions

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/ledger"
)

func resolveCustomerFBOPriorityDimension(ctx context.Context, deps ResolverDependencies, configured *int) (ledger.DimensionCreditPriority, error) {
	priority := ledger.DefaultCustomerFBOPriority
	if configured != nil {
		priority = *configured
	}
	if priority < 1 {
		return nil, fmt.Errorf("credit priority must be a positive integer")
	}

	dim, err := deps.DimensionService.GetCreditPriorityDimension(ctx, priority)
	if err != nil {
		return nil, fmt.Errorf("failed to get credit priority dimension %d: %w", priority, err)
	}

	return dim, nil
}
