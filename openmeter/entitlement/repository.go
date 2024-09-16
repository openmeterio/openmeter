package entitlement

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
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
	GetEntitlementOfSubject(ctx context.Context, namespace string, subjectKey string, idOrFeatureKey string) (*Entitlement, error)
	DeleteEntitlement(ctx context.Context, entitlementID models.NamespacedID) error

	ListEntitlements(ctx context.Context, params ListEntitlementsParams) (pagination.PagedResponse[Entitlement], error)
	ListNamespacesWithActiveEntitlements(ctx context.Context, includeDeletedAfter time.Time) ([]string, error)

	HasEntitlementForMeter(ctx context.Context, namespace string, meterSlug string) (bool, error)

	UpdateEntitlementUsagePeriod(ctx context.Context, entitlementID models.NamespacedID, params UpdateEntitlementUsagePeriodParams) error
	ListEntitlementsWithExpiredUsagePeriod(ctx context.Context, namespaces []string, highwatermark time.Time) ([]Entitlement, error)

	LockEntitlementForTx(ctx context.Context, tx *entutils.TxDriver, entitlementID models.NamespacedID) error

	entutils.TxCreator
}

type CreateEntitlementRepoInputs struct {
	Namespace       string            `json:"namespace"`
	FeatureID       string            `json:"featureId"`
	FeatureKey      string            `json:"featureKey"`
	SubjectKey      string            `json:"subjectKey"`
	EntitlementType EntitlementType   `json:"type"`
	Metadata        map[string]string `json:"metadata,omitempty"`

	MeasureUsageFrom        *time.Time         `json:"measureUsageFrom,omitempty"`
	IssueAfterReset         *float64           `json:"issueAfterReset,omitempty"`
	IssueAfterResetPriority *uint8             `json:"issueAfterResetPriority,omitempty"`
	IsSoftLimit             *bool              `json:"isSoftLimit,omitempty"`
	Config                  []byte             `json:"config,omitempty"`
	UsagePeriod             *UsagePeriod       `json:"usagePeriod,omitempty"`
	CurrentUsagePeriod      *recurrence.Period `json:"currentUsagePeriod,omitempty"`
	PreserveOverageAtReset  *bool              `json:"preserveOverageAtReset,omitempty"`
}
