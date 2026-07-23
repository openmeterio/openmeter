package costbasis

import "github.com/openmeterio/openmeter/pkg/models"

type CostBasis struct {
	models.NamespacedID
	models.ManagedModel

	CurrencyID string `json:"currencyID"`

	Intent Intent `json:"intent"`
	State  *State `json:"state,omitempty"`
}
