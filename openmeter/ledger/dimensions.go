package ledger

import (
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/pkg/models"
)

// ----------------------------------------------------------------------------
// Dimension type definitions
// ----------------------------------------------------------------------------

// Dimension is a generic key-value pair that can be used to filter and roll-up balance of sub-accounts
// SubAccount-s reference Dimensions in their definitions so that the ledger is a self-contained system. In practice Dimensions translate to external resources (e.g. Currency, Features...) and should be managed & provisioned separately.
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
	// For now we only support currency as a dimension
	case DimensionKeyCurrency:
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

// CustomerSubAccountDimensions are dimensions specific to customer sub-accounts.
type CustomerSubAccountDimensions struct {
	Currency       DimensionCurrency
	TaxCode        DimensionTaxCode
	CreditPriority DimensionCreditPriority
	Features       mo.Option[DimensionFeature]
}
