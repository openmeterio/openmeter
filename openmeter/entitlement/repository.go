package entitlement

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	paginationv2 "github.com/openmeterio/openmeter/pkg/pagination/v2"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type UpdateEntitlementUsagePeriodParams struct {
	CurrentUsagePeriod timeutil.ClosedPeriod
}

type ListExpiredEntitlementsParams struct {
	Namespaces    []string
	Highwatermark time.Time
	Limit         int
	// Cursor is the ID of the last entitlement in the previous page
	// If not provided, the query will return the first page of results
	Cursor *paginationv2.Cursor
}

type UpsertEntitlementCurrentPeriodElement struct {
	models.NamespacedID
	CurrentUsagePeriod timeutil.ClosedPeriod
}

type EntitlementRepo interface {
	// GetActiveEntitlementOfSubjectAt returns the active entitlement of a customer at a given time by feature key
	GetActiveEntitlementOfCustomerAt(ctx context.Context, namespace string, customerID string, featureKey string, at time.Time) (*Entitlement, error)

	// GetScheduledEntitlements returns all scheduled entitlements for a given customer-feature pair that become inactive after the given time, sorted by the time they become active
	GetScheduledEntitlements(ctx context.Context, namespace string, customerID string, featureKey string, starting time.Time) ([]Entitlement, error)

	// DeactivateEntitlement deactivates an entitlement by setting the activeTo time. If the entitlement is already deactivated, it returns an error.
	DeactivateEntitlement(ctx context.Context, entitlementID models.NamespacedID, at time.Time) error

	CreateEntitlement(ctx context.Context, entitlement CreateEntitlementRepoInputs) (*Entitlement, error)
	GetEntitlement(ctx context.Context, entitlementID models.NamespacedID) (*Entitlement, error)
	DeleteEntitlement(ctx context.Context, entitlementID models.NamespacedID, at time.Time) error

	ListEntitlements(ctx context.Context, params ListEntitlementsParams) (pagination.Result[Entitlement], error)

	// ListNamespacesWithActiveEntitlements returns a list of namespaces that have active entitlements
	//
	// Active in this context means the entitlement is active at any point between now and the given time.
	// If includeDeletedAfter is before the current time, it will include namespaces that have entitlements active at that instance.
	ListNamespacesWithActiveEntitlements(ctx context.Context, includeDeletedAfter time.Time) ([]string, error)

	UpdateEntitlementUsagePeriod(ctx context.Context, entitlementID models.NamespacedID, params UpdateEntitlementUsagePeriodParams) error

	// Methods for entitlement batch reset

	// ListActiveEntitlementsWithExpiredUsagePeriod returns a list of active entitlements with usage period that expired before the highwatermark
	// - Only entitlements active at the highwatermark are considered.
	// - The list is sorted by the current usage period end, then by created at, then by id.
	// - The list is paginated by the cursor & limit.
	// - CurrentUsagePeriod won't be mapped to the calculated values
	ListActiveEntitlementsWithExpiredUsagePeriod(ctx context.Context, params ListExpiredEntitlementsParams) ([]Entitlement, error)

	// UpsertEntitlementCurrentPeriods upserts the current usage period for a list of entitlements
	// - If an entitlement is found, it will be updated
	// - If any update fails, the entire operation will fail
	UpsertEntitlementCurrentPeriods(ctx context.Context, updates []UpsertEntitlementCurrentPeriodElement) error

	LockEntitlementForTx(ctx context.Context, tx *entutils.TxDriver, entitlementID models.NamespacedID) error

	entutils.TxCreator
}

type CreateEntitlementRepoInputs struct {
	Namespace        string                             `json:"namespace"`
	FeatureID        string                             `json:"featureId"`
	FeatureKey       string                             `json:"featureKey"`
	UsageAttribution streaming.CustomerUsageAttribution `json:"usageAttribution"`
	EntitlementType  EntitlementType                    `json:"type"`
	Metadata         map[string]string                  `json:"metadata,omitempty"`
	ActiveFrom       *time.Time                         `json:"activeFrom,omitempty"`
	ActiveTo         *time.Time                         `json:"activeTo,omitempty"`

	Annotations models.Annotations `json:"annotations,omitempty"`

	MeasureUsageFrom        *time.Time             `json:"measureUsageFrom,omitempty"`
	IssueAfterReset         *float64               `json:"issueAfterReset,omitempty"`
	IssueAfterResetPriority *uint8                 `json:"issueAfterResetPriority,omitempty"`
	IsSoftLimit             *bool                  `json:"isSoftLimit,omitempty"`
	Config                  *string                `json:"config,omitempty"`
	UsagePeriod             *UsagePeriodInput      `json:"usagePeriod,omitempty"`
	CurrentUsagePeriod      *timeutil.ClosedPeriod `json:"currentUsagePeriod,omitempty"`
	PreserveOverageAtReset  *bool                  `json:"preserveOverageAtReset,omitempty"`
}
