package entitlement

import (
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/api/types"
	"github.com/openmeterio/openmeter/pkg/models"
)

type CreateEntitlementInputs struct {
	Namespace        string
	FeatureID        string                            `json:"featureId"`
	MeasureUsageFrom time.Time                         `json:"measureUsageFrom,omitempty"`
	SubjectKey       string                            `json:"subjectKey"`
	UsagePeriod      types.RecurringPeriodCreateInputs `json:"usagePeriod,omitempty"`
}

type Entitlement struct {
	models.NamespacedModel
	models.ManagedModel
	ID               string                `json:"id,omitempty"`
	FeatureID        string                `json:"featureId,omitempty"`
	MeasureUsageFrom time.Time             `json:"measureUsageFrom,omitempty"`
	SubjectKey       string                `json:"subjectKey,omitempty"`
	UsagePeriod      types.RecurringPeriod `json:"usagePeriod,omitempty"`
}

type EntitlementAlreadyExistsError struct {
	EntitlementID string
	FeatureID     string
	SubjectKey    string
}

func (e *EntitlementAlreadyExistsError) Error() string {
	return fmt.Sprintf("entitlement with id %s already exists for feature %s and subject %s", e.EntitlementID, e.FeatureID, e.SubjectKey)
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

type EntitlementValue struct {
	HasAccess bool    `json:"hasAccess"`
	Balance   float64 `json:"balance"`
	Usage     float64 `json:"usage"`
	Overage   float64 `json:"overage"`
}
