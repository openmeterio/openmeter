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
	// GetActiveEntitlementsOfSubject returns all active entitlements of a subject at a given time
	GetActiveEntitlementsOfSubject(ctx context.Context, namespace string, subjectKey models.SubjectKey, at time.Time) ([]Entitlement, error)

	// GetActiveEntitlementOfSubjectAt returns the active entitlement of a subject at a given time by feature key
	GetActiveEntitlementOfSubjectAt(ctx context.Context, namespace string, subjectKey string, featureKey string, at time.Time) (*Entitlement, error)

	// GetScheduledEntitlements returns all scheduled entitlements for a given subject-feature pair that become inactive after the given time, sorted by the time they become active
	GetScheduledEntitlements(ctx context.Context, namespace string, subjectKey models.SubjectKey, featureKey string, starting time.Time) ([]Entitlement, error)

	CreateEntitlement(ctx context.Context, entitlement CreateEntitlementRepoInputs) (*Entitlement, error)
	GetEntitlement(ctx context.Context, entitlementID models.NamespacedID) (*Entitlement, error)
	DeleteEntitlement(ctx context.Context, entitlementID models.NamespacedID) error

	ListEntitlements(ctx context.Context, params ListEntitlementsParams) (pagination.PagedResponse[Entitlement], error)

	// ListNamespacesWithActiveEntitlements returns a list of namespaces that have active entitlements
	//
	// Active in this context means the entitlement is active at any point between now and the given time.
	// If includeDeletedAfter is before the current time, it will include namespaces that have entitlements active at that instance.
	ListNamespacesWithActiveEntitlements(ctx context.Context, includeDeletedAfter time.Time) ([]string, error)

	// HasEntitlementForMeter returns true if the meter has any active or inactive entitlements
	HasEntitlementForMeter(ctx context.Context, namespace string, meterSlug string) (bool, error)

	UpdateEntitlementUsagePeriod(ctx context.Context, entitlementID models.NamespacedID, params UpdateEntitlementUsagePeriodParams) error

	// ListActiveEntitlementsWithExpiredUsagePeriod returns a list of active entitlements with usage period that expired before the highwatermark
	//
	// Only entitlements active at the highwatermark are considered. FIXME: this implementation might be incorrect
	ListActiveEntitlementsWithExpiredUsagePeriod(ctx context.Context, namespaces []string, highwatermark time.Time) ([]Entitlement, error)

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
