package currencies

import (
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type CostBasis struct {
	models.ManagedModel
	models.NamespacedID
	currencyx.CostBasis

	CurrencyID string
}
