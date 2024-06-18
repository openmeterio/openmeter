package entitlement

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

type CreateEntitlementDBInputs struct {
	Namespace        string
	FeatureID        string    `json:"featureId"`
	MeasureUsageFrom time.Time `json:"measureUsageFrom,omitempty"`
	SubjectKey       string    `json:"subjectKey"`
}

type EntitlementDBConnector interface {
	// Entitlement Management
	GetEntitlementsOfSubject(ctx context.Context, namespace string, subjectKey models.SubjectKey) ([]Entitlement, error)
	CreateEntitlement(ctx context.Context, entitlement CreateEntitlementDBInputs) (*Entitlement, error)
	GetEntitlement(ctx context.Context, entitlementID models.NamespacedID) (*Entitlement, error)

	//FIXME: This is a terrbile hack
	LockEntitlementForTx(ctx context.Context, entitlementID models.NamespacedID) error

	entutils.TxCreator
	entutils.TxUser[EntitlementDBConnector]
}

type UsageResetDBConnector interface {
	Save(ctx context.Context, usageResetTime UsageResetTime) error
	GetLastAt(ctx context.Context, entitlementID models.NamespacedID, at time.Time) (*UsageResetTime, error)
	GetBetween(ctx context.Context, entitlementID models.NamespacedID, from time.Time, to time.Time) ([]UsageResetTime, error)

	entutils.TxCreator
	entutils.TxUser[UsageResetDBConnector]
}

type UsageResetNotFoundError struct {
	EntitlementID models.NamespacedID
}

func (e UsageResetNotFoundError) Error() string {
	return fmt.Sprintf("usage reset not found for entitlement %s in namespace %s", e.EntitlementID.ID, e.EntitlementID.Namespace)
}
