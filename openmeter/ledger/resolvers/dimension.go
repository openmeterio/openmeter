package resolvers

import (
	"context"
	"strconv"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
)

// DimensionLookup is the minimal dependency required by DimensionResolver.
type DimensionLookup interface {
	GetDimensionByKeyAndValue(ctx context.Context, namespace string, key ledger.DimensionKey, value string) (*ledgeraccount.DimensionData, error)
}

// DimensionResolver implements ledger.DimensionResolver for a given namespace.
// Dimension lifecycle is externally owned; ledger only resolves local dimension
// references used for routing and transactional integrity.
type DimensionResolver struct {
	Namespace string
	Lookup    DimensionLookup
}

var _ ledger.DimensionResolver = (*DimensionResolver)(nil)

func (r *DimensionResolver) GetCurrencyDimension(ctx context.Context, value string) (ledger.DimensionCurrency, error) {
	dim, err := r.Lookup.GetDimensionByKeyAndValue(ctx, r.Namespace, ledger.DimensionKeyCurrency, value)
	if err != nil {
		return nil, err
	}

	return dim.AsCurrencyDimension()
}

func (r *DimensionResolver) GetCreditPriorityDimension(ctx context.Context, value int) (ledger.DimensionCreditPriority, error) {
	dim, err := r.Lookup.GetDimensionByKeyAndValue(ctx, r.Namespace, ledger.DimensionKeyCreditPriority, strconv.Itoa(value))
	if err != nil {
		return nil, err
	}

	return dim.AsCreditPriorityDimension()
}
