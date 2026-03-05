package ledger

import (
	"context"
	"fmt"
	"strconv"

	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/pkg/models"
)

// ----------------------------------------------------------------------------
// Dimension type definitions
// ----------------------------------------------------------------------------

// Dimension is a generic key-value pair that can be used to filter and roll-up balance of sub-accounts.
// Dimension lifecycle is externally owned; ledger stores local references used for
// transactional routing and referential integrity.
type Dimension[V any] interface {
	models.Equaler[Dimension[any]]

	Key() DimensionKey

	DimensionValue[V]
}

type DimensionValue[V any] interface {
	Value() V
	DisplayValue() string
}

type DimensionKey string

var (
	DimensionKeyCurrency       DimensionKey = "currency"
	DimensionKeyFeature        DimensionKey = "feature"
	DimensionKeyCreditPriority DimensionKey = "credit_priority"
	DimensionKeyTaxCode        DimensionKey = "tax_code"
)

func (d DimensionKey) Validate() error {
	switch d {
	case DimensionKeyCurrency, DimensionKeyCreditPriority:
		return nil
	default:
		return ErrInvalidDimensionKey.WithAttrs(models.Attributes{
			"dimension_key": d,
		})
	}
}

// ----------------------------------------------------------------------------
// Dimension types
// ----------------------------------------------------------------------------

// List of all known dimension types
type (
	DimensionCurrency       Dimension[string]
	DimensionTaxCode        Dimension[string]
	DimensionFeature        Dimension[[]string]
	DimensionCreditPriority Dimension[int]
)

type DimensionResolver interface {
	GetCurrencyDimension(ctx context.Context, value string) (DimensionCurrency, error)
	GetCreditPriorityDimension(ctx context.Context, value int) (DimensionCreditPriority, error)
}

func ParseCreditPriority(raw string) (int, error) {
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("parse credit priority: %w", err)
	}
	if value < 1 {
		return 0, fmt.Errorf("credit priority must be a positive integer")
	}
	return value, nil
}

// ----------------------------------------------------------------------------
// SubAccountDimensions
// ----------------------------------------------------------------------------

// SubAccountDimensions is a set of all known dimensins for a generic sub-account
type SubAccountDimensions struct {
	Currency       DimensionCurrency
	TaxCode        mo.Option[DimensionTaxCode]
	CreditPriority mo.Option[DimensionCreditPriority]
	Feature        mo.Option[DimensionFeature]
}
