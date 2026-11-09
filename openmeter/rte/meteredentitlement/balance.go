package meteredentitlement

import (
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Key identifies the current calculation for an entitlement.
type Key struct {
	EntitlementID models.NamespacedID
}

func (k Key) Validate() error {
	return models.NewNillableGenericValidationError(k.EntitlementID.Validate())
}

type Balance = meteredentitlement.EntitlementBalance
