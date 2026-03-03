package account

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/ledger"
)

// DimensionResolver implements ledger.DimensionResolver for a given namespace.
// Dimension lifecycle is externally owned; ledger only resolves local dimension
// references used for routing and transactional integrity.
type DimensionResolver struct {
	Namespace string
	Service   Service
}

var _ ledger.DimensionResolver = (*DimensionResolver)(nil)

func (r *DimensionResolver) GetCurrencyDimension(ctx context.Context, value string) (ledger.DimensionCurrency, error) {
	dim, err := r.Service.GetDimensionByKeyAndValue(ctx, r.Namespace, ledger.DimensionKeyCurrency, value)
	if err != nil {
		return nil, err
	}

	return dim.AsCurrencyDimension()
}
