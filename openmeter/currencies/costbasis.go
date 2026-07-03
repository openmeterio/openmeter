package currencies

import (
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/pkg/models"
)

type CostBasis struct {
	models.ManagedModel
	models.NamespacedID
	CurrencyID    string                `json:"currency_id"`
	FiatCode      string                `json:"fiat_code"`
	Rate          alpacadecimal.Decimal `json:"rate"`
	EffectiveFrom time.Time             `json:"effective_from"`
	EffectiveTo   *time.Time            `json:"effective_to,omitempty"`
}
