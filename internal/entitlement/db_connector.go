package entitlement

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

type EntitlementRepoCreateEntitlementInputs struct {
	Namespace        string
	FeatureID        string                  `json:"featureId"`
	MeasureUsageFrom time.Time               `json:"measureUsageFrom,omitempty"`
	SubjectKey       string                  `json:"subjectKey"`
	UsagePeriod      RecurrenceWithNextReset `json:"usagePeriod"`
}

type EntitlementRepo interface {
	// Entitlement Management
	GetEntitlementsOfSubject(ctx context.Context, namespace string, subjectKey models.SubjectKey) ([]Entitlement, error)
	CreateEntitlement(ctx context.Context, entitlement EntitlementRepoCreateEntitlementInputs) (*Entitlement, error)
	UpdateEntitlementUsagePeriod(ctx context.Context, entitlementID models.NamespacedID, newAnchor *time.Time, nextReset time.Time) error
	GetEntitlement(ctx context.Context, entitlementID models.NamespacedID) (*Entitlement, error)

	ListEntitlements(ctx context.Context, params ListEntitlementsParams) ([]Entitlement, error)

	//FIXME: This is a terrbile hack
	LockEntitlementForTx(ctx context.Context, entitlementID models.NamespacedID) error

	entutils.TxCreator
	entutils.TxUser[EntitlementRepo]
}

type UsageResetRepo interface {
	Save(ctx context.Context, usageResetTime UsageResetTime) error
	GetLastAt(ctx context.Context, entitlementID models.NamespacedID, at time.Time) (*UsageResetTime, error)
	GetBetween(ctx context.Context, entitlementID models.NamespacedID, from time.Time, to time.Time) ([]UsageResetTime, error)

	entutils.TxCreator
	entutils.TxUser[UsageResetRepo]
}

type UsageResetNotFoundError struct {
	EntitlementID models.NamespacedID
}

func (e UsageResetNotFoundError) Error() string {
	return fmt.Sprintf("usage reset not found for entitlement %s in namespace %s", e.EntitlementID.ID, e.EntitlementID.Namespace)
}
