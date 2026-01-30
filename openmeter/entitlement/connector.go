package entitlement

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/models"
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
	IDs              []string
	Namespaces       []string
	SubjectKeys      []string
	CustomerIDs      []string
	CustomerKeys     []string
	FeatureIDs       []string
	FeatureKeys      []string
	FeatureIDsOrKeys []string
	EntitlementTypes []EntitlementType
	OrderBy          ListEntitlementsOrderBy
	Order            sortx.Order
	// TODO[galexi]: We should clean up how these 4 fields are used together.
	IncludeDeleted      bool
	IncludeDeletedAfter time.Time
	ExcludeInactive     bool
	ActiveAt            *time.Time

	Page pagination.Page
	// will be deprecated
	Limit int
	// will be deprecated
	Offset int
}

type Service interface {
	models.ServiceHooks[Entitlement]

	// Meant for API use primarily
	CreateEntitlement(ctx context.Context, input CreateEntitlementInputs, grants []CreateEntitlementGrantInputs) (*Entitlement, error)
	// OverrideEntitlement replaces a currently active entitlement with a new one.
	OverrideEntitlement(ctx context.Context, customerID string, entitlementIdOrFeatureKey string, input CreateEntitlementInputs, grants []CreateEntitlementGrantInputs) (*Entitlement, error)

	ScheduleEntitlement(ctx context.Context, input CreateEntitlementInputs) (*Entitlement, error)
	// SupersedeEntitlement replaces an entitlement by scheduling a new one
	SupersedeEntitlement(ctx context.Context, entitlementId string, input CreateEntitlementInputs) (*Entitlement, error)

	GetEntitlement(ctx context.Context, namespace string, id string) (*Entitlement, error)
	GetEntitlementWithCustomer(ctx context.Context, namespace string, id string) (*EntitlementWithCustomer, error)
	DeleteEntitlement(ctx context.Context, namespace string, id string, at time.Time) error

	GetEntitlementValue(ctx context.Context, namespace string, customerID string, idOrFeatureKey string, at time.Time) (EntitlementValue, error)

	GetEntitlementsOfCustomer(ctx context.Context, namespace string, customerId string, at time.Time) ([]Entitlement, error)
	ListEntitlements(ctx context.Context, params ListEntitlementsParams) (pagination.Result[Entitlement], error)
	ListEntitlementsWithCustomer(ctx context.Context, params ListEntitlementsParams) (ListEntitlementsWithCustomerResult, error)

	// Attempts to get the entitlement in an ambiguous situation where it's unclear if the entitlement is referenced by ID or FeatureKey + CustomerID.
	// First attempts to resolve by ID, then by FeatureKey + CustomerID.
	//
	// For consistency, it is forbidden for entitlements to be created for featueres the keys of which could be mistaken for entitlement IDs.
	GetEntitlementOfCustomerAt(ctx context.Context, namespace string, customerID string, idOrFeatureKey string, at time.Time) (*Entitlement, error)

	// GetAccess returns the access of a customer.
	// It returns a map of featureKey to entitlement value + ID.
	GetAccess(ctx context.Context, namespace string, customerID string) (Access, error)
}

type ListEntitlementsWithCustomerResult struct {
	Entitlements  pagination.Result[Entitlement]
	CustomersByID map[models.NamespacedID]*customer.Customer
}

type EntitlementWithCustomer struct {
	Entitlement
	Customer customer.Customer
}
