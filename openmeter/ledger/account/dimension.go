package account

import (
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
)

type DimensionData struct {
	ID          string
	Namespace   string
	CreatedAt   time.Time
	Annotations models.Annotations
	models.ManagedModel

	DimensionKey          ledger.DimensionKey
	DimensionValue        string // TBD
	DimensionDisplayValue string
}

func (d *DimensionData) AsCurrencyDimension() (*currencyDimension, error) {
	if d.DimensionKey != ledger.DimensionKeyCurrency {
		return nil, fmt.Errorf("dimension is not a currency dimension")
	}

	return &currencyDimension{
		data: *d,
	}, nil
}

type currencyDimension struct {
	data DimensionData
}

var _ ledger.DimensionCurrency = (*currencyDimension)(nil)

func (d *currencyDimension) Equal(other ledger.Dimension[any]) bool {
	return d.Key() == other.Key() && d.Value() == other.Value()
}

func (d *currencyDimension) Key() ledger.DimensionKey {
	return ledger.DimensionKeyCurrency
}

func (d *currencyDimension) Value() string {
	return d.data.DimensionValue
}

func (d *currencyDimension) DisplayValue() string {
	return d.data.DimensionDisplayValue
}

// TODO: Implement other dimension types
func (d *DimensionData) AsTaxCodeDimension() (ledger.DimensionTaxCode, error) {
	return nil, ledger.ErrInvalidDimensionKey
}

func (d *DimensionData) AsFeatureDimension() (ledger.DimensionFeature, error) {
	return nil, ledger.ErrInvalidDimensionKey
}

func (d *DimensionData) AsCreditPriorityDimension() (ledger.DimensionCreditPriority, error) {
	return nil, ledger.ErrInvalidDimensionKey
}
