package entitlement

import (
	"fmt"

	"github.com/openmeterio/openmeter/pkg/models"
)

type AlreadyExistsError struct {
	EntitlementID string
	FeatureID     string
	SubjectKey    string
}

func (e *AlreadyExistsError) Error() string {
	return fmt.Sprintf("entitlement with id %s already exists for feature %s and subject %s", e.EntitlementID, e.FeatureID, e.SubjectKey)
}

type NotFoundError struct {
	EntitlementID models.NamespacedID
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("entitlement not found %s in namespace %s", e.EntitlementID.ID, e.EntitlementID.Namespace)
}

type WrongTypeError struct {
	Expected EntitlementType
	Actual   EntitlementType
}

func (e *WrongTypeError) Error() string {
	return fmt.Sprintf("expected entitlement type %s but got %s", e.Expected, e.Actual)
}

type InvalidValueError struct {
	Message string
	Type    EntitlementType
}

func (e *InvalidValueError) Error() string {
	return fmt.Sprintf("invalid entitlement value for type %s: %s", e.Type, e.Message)
}

type InvalidFeatureError struct {
	FeatureID string
	Message   string
}

func (e *InvalidFeatureError) Error() string {
	return fmt.Sprintf("invalid feature %s: %s", e.FeatureID, e.Message)
}
