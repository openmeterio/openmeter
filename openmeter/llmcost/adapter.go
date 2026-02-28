package llmcost

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// Adapter provides persistence for LLM cost prices.
type Adapter interface {
	entutils.TxCreator

	// Canonical prices (global + overrides)
	ListPrices(ctx context.Context, input ListPricesInput) (pagination.Result[Price], error)
	GetPrice(ctx context.Context, input GetPriceInput) (Price, error)
	ResolvePrice(ctx context.Context, input ResolvePriceInput) (Price, error)

	// Per-namespace overrides
	CreateOverride(ctx context.Context, input CreateOverrideInput) (Price, error)
	UpdateOverride(ctx context.Context, input UpdateOverrideInput) (Price, error)
	DeleteOverride(ctx context.Context, input DeleteOverrideInput) error
	ListOverrides(ctx context.Context, input ListOverridesInput) (pagination.Result[Price], error)

	// Reconciled global prices
	UpsertGlobalPrice(ctx context.Context, price Price) error
}
