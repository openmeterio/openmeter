package currencies

import (
	"time"

	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type CostBasis struct {
	models.ManagedModel
	models.NamespacedID
	currencyx.CostBasis

	CurrencyID string `json:"currency_id"`

	// CustomCurrency is included only if the CostBasis is expanded
	CustomCurrency *Currency `json:"-"`
}

// IsEffectiveAt reports whether the cost basis can be used at the provided
// time. EffectiveFrom is inclusive and EffectiveTo is exclusive.
func (c CostBasis) IsEffectiveAt(at time.Time) bool {
	return !c.EffectiveFrom.After(at) &&
		(c.EffectiveTo == nil || c.EffectiveTo.After(at)) &&
		(c.DeletedAt == nil || c.DeletedAt.After(at))
}
