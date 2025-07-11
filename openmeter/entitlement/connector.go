package entitlement

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type ListEntitlementsOrderBy string

const (
	ListEntitlementsOrderByCreatedAt ListEntitlementsOrderBy = "created_at"
	ListEntitlementsOrderByUpdatedAt ListEntitlementsOrderBy = "updated_at"
)

func (o ListEntitlementsOrderBy) Values() []ListEntitlementsOrderBy {
	return []ListEntitlementsOrderBy{
		ListEntitlementsOrderByCreatedAt,
		ListEntitlementsOrderByUpdatedAt,
	}
}

func (o ListEntitlementsOrderBy) StrValues() []string {
	return slicesx.Map(o.Values(), func(v ListEntitlementsOrderBy) string {
		return string(v)
	})
}

type ListEntitlementsParams struct {
	IDs                 []string
	Namespaces          []string
	SubjectKeys         []string
	FeatureIDs          []string
	FeatureKeys         []string
	FeatureIDsOrKeys    []string
	EntitlementTypes    []EntitlementType
	OrderBy             ListEntitlementsOrderBy
	Order               sortx.Order
	IncludeDeleted      bool
	IncludeDeletedAfter time.Time
	ExcludeInactive     bool
	Page                pagination.Page
	// will be deprecated
	Limit int
	// will be deprecated
	Offset int
}

type Connector interface {
	CreateEntitlement(ctx context.Context, input CreateEntitlementInputs) (*Entitlement, error)
	ScheduleEntitlement(ctx context.Context, input CreateEntitlementInputs) (*Entitlement, error)
	// OverrideEntitlement replaces a currently active entitlement with a new one.
	OverrideEntitlement(ctx context.Context, subject string, entitlementIdOrFeatureKey string, input CreateEntitlementInputs) (*Entitlement, error)
	// SupersedeEntitlement replaces an entitlement by scheduling a new one
	SupersedeEntitlement(ctx context.Context, entitlementId string, input CreateEntitlementInputs) (*Entitlement, error)

	GetEntitlement(ctx context.Context, namespace string, id string) (*Entitlement, error)
	DeleteEntitlement(ctx context.Context, namespace string, id string, at time.Time) error

	GetEntitlementValue(ctx context.Context, namespace string, subjectKey string, idOrFeatureKey string, at time.Time) (EntitlementValue, error)

	GetEntitlementsOfSubject(ctx context.Context, namespace string, subjectKey string, at time.Time) ([]Entitlement, error)
	ListEntitlements(ctx context.Context, params ListEntitlementsParams) (pagination.PagedResponse[Entitlement], error)

	// Attempts to get the entitlement in an ambiguous situation where it's unclear if the entitlement is referenced by ID or FeatureKey + SubjectKey.
	// First attempts to resolve by ID, then by FeatureKey + SubjectKey.
	//
	// For consistency, it is forbidden for entitlements to be created for featueres the keys of which could be mistaken for entitlement IDs.
	GetEntitlementOfSubjectAt(ctx context.Context, namespace string, subjectKey string, idOrFeatureKey string, at time.Time) (*Entitlement, error)

	// GetAccess returns the access of a subject for a given namespace.
	// It returns a map of featureKey to entitlement value + ID.
	GetAccess(ctx context.Context, namespace string, subjectKey string) (Access, error)
}
