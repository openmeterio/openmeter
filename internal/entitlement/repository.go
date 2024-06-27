package entitlement

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/recurrence"
)

type UpdateEntitlementUsagePeriodParams struct {
	NewAnchor          *time.Time
	CurrentUsagePeriod recurrence.Period
}

type EntitlementRepo interface {
	// Entitlement Management
	GetEntitlementsOfSubject(ctx context.Context, namespace string, subjectKey models.SubjectKey) ([]Entitlement, error)
	CreateEntitlement(ctx context.Context, entitlement CreateEntitlementRepoInputs) (*Entitlement, error)
	GetEntitlement(ctx context.Context, entitlementID models.NamespacedID) (*Entitlement, error)
	GetEntitlementOfSubject(ctx context.Context, namespace string, subjectKey string, id string) (*Entitlement, error)
	DeleteEntitlement(ctx context.Context, entitlementID models.NamespacedID) error

	ListEntitlements(ctx context.Context, params ListEntitlementsParams) ([]Entitlement, error)

	//FIXME: This is a terrbile hack
	LockEntitlementForTx(ctx context.Context, entitlementID models.NamespacedID) error

	UpdateEntitlementUsagePeriod(ctx context.Context, entitlementID models.NamespacedID, params UpdateEntitlementUsagePeriodParams) error
	ListEntitlementsWithExpiredUsagePeriod(ctx context.Context, namespace string, highwatermark time.Time) ([]Entitlement, error)

	entutils.TxCreator
	entutils.TxUser[EntitlementRepo]
}

type CreateEntitlementRepoInputs struct {
	Namespace       string            `json:"namespace"`
	FeatureID       string            `json:"featureId"`
	SubjectKey      string            `json:"subjectKey"`
	EntitlementType EntitlementType   `json:"type"`
	Metadata        map[string]string `json:"metadata,omitempty"`

	MeasureUsageFrom   *time.Time         `json:"measureUsageFrom,omitempty"`
	IssueAfterReset    *float64           `json:"issueAfterReset,omitempty"`
	IsSoftLimit        *bool              `json:"isSoftLimit,omitempty"`
	Config             *string            `json:"config,omitempty"`
	UsagePeriod        *UsagePeriod       `json:"usagePeriod,omitempty"`
	CurrentUsagePeriod *recurrence.Period `json:"currentUsagePeriod,omitempty"`
}
