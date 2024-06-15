package entitlement

import (
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

type CreateEntitlementInputs struct {
	Namespace        string
	FeatureID        string
	MeasureUsageFrom time.Time
}

type Entitlement struct {
	models.NamespacedModel
	models.ManagedModel
	ID               string    `json:"id,omitempty"`
	FeatureID        string    `json:"featureId,omitempty"`
	MeasureUsageFrom time.Time `json:"measureUsageFrom,omitempty"`
}

type EntitlementNotFoundError struct {
	EntitlementID models.NamespacedID
}

func (e *EntitlementNotFoundError) Error() string {
	return fmt.Sprintf("entitlement not found %s in namespace %s", e.EntitlementID.ID, e.EntitlementID.Namespace)
}

type UsageResetTime struct {
	models.NamespacedModel
	ResetTime     time.Time
	EntitlementID string
}
