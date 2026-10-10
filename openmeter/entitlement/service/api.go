package service

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/credit"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func (c *service) GetCustomerEntitlementAccess(ctx context.Context, input entitlement.GetCustomerEntitlementAccessInput) (entitlement.CustomerEntitlementAccess, error) {
	if err := input.Validate(); err != nil {
		return entitlement.CustomerEntitlementAccess{}, err
	}

	cus, err := c.getActiveCustomer(ctx, input.CustomerID)
	if err != nil {
		return entitlement.CustomerEntitlementAccess{}, err
	}

	ent, err := c.getAccessEntitlement(ctx, cus, input)
	if err != nil {
		return entitlement.CustomerEntitlementAccess{}, err
	}

	if ent == nil {
		return entitlement.CustomerEntitlementAccess{
			FeatureKey: input.FeatureKey,
			Value:      &entitlement.NoAccessValue{},
		}, nil
	}

	value, err := c.getEntitlementValueAt(ctx, ent, input.At)
	if err != nil {
		// The metered engine resolves the entitlement against the current time, so a
		// feature-key lookup at a time before a metered entitlement's deletion cannot
		// be evaluated and grants no access.
		if _, ok := lo.ErrorsAs[*entitlement.NotFoundError](err); ok {
			return entitlement.CustomerEntitlementAccess{
				FeatureKey: ent.FeatureKey,
				Type:       ent.EntitlementType,
				Value:      &entitlement.NoAccessValue{},
			}, nil
		}

		return entitlement.CustomerEntitlementAccess{}, err
	}

	return entitlement.CustomerEntitlementAccess{
		FeatureKey: ent.FeatureKey,
		Type:       ent.EntitlementType,
		Value:      value,
	}, nil
}

// getAccessEntitlement returns nil without an error when the feature key has no
// active entitlement, since missing access is a valid answer for a feature lookup
// but not for an entitlement ID.
func (c *service) getAccessEntitlement(ctx context.Context, cus *customer.Customer, input entitlement.GetCustomerEntitlementAccessInput) (*entitlement.Entitlement, error) {
	if input.EntitlementID == "" {
		ent, err := c.entitlementRepo.GetActiveEntitlementOfCustomerAt(ctx, cus.Namespace, cus.ID, input.FeatureKey, input.At)
		if _, ok := lo.ErrorsAs[*entitlement.NotFoundError](err); ok {
			return nil, nil
		}

		return ent, err
	}

	ent, err := c.getCustomerEntitlement(ctx, cus, input.EntitlementID)
	if err != nil {
		if _, ok := lo.ErrorsAs[*entitlement.NotFoundError](err); ok {
			return nil, models.NewGenericNotFoundError(err)
		}

		return nil, err
	}

	return ent, nil
}

func (c *service) ListCustomerEntitlementAccess(ctx context.Context, input entitlement.ListCustomerEntitlementAccessInput) ([]entitlement.CustomerEntitlementAccess, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	cus, err := c.getActiveCustomer(ctx, input.CustomerID)
	if err != nil {
		return nil, err
	}

	access, err := c.GetAccess(ctx, cus.Namespace, cus.ID)
	if err != nil {
		return nil, err
	}

	items := make([]entitlement.CustomerEntitlementAccess, 0, len(access.Entitlements))
	for featureKey, ent := range access.Entitlements {
		if _, ok := ent.Value.(*entitlement.NoAccessValue); ok {
			continue
		}

		items = append(items, entitlement.CustomerEntitlementAccess{
			FeatureKey: featureKey,
			Type:       ent.Type,
			Value:      ent.Value,
		})
	}

	slices.SortFunc(items, func(a, b entitlement.CustomerEntitlementAccess) int {
		return strings.Compare(a.FeatureKey, b.FeatureKey)
	})

	return items, nil
}

func (c *service) CreateCustomerEntitlement(ctx context.Context, input entitlement.CreateCustomerEntitlementInput) (*entitlement.Entitlement, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	cus, err := c.getActiveCustomer(ctx, input.CustomerID)
	if err != nil {
		return nil, err
	}

	createInput := input.Entitlement
	createInput.Namespace = cus.Namespace
	createInput.UsageAttribution = cus.GetUsageAttribution()

	return c.CreateEntitlement(ctx, createInput, input.Grants)
}

func (c *service) OverrideCustomerEntitlement(ctx context.Context, input entitlement.OverrideCustomerEntitlementInput) (*entitlement.Entitlement, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	cus, err := c.getActiveCustomer(ctx, input.CustomerID)
	if err != nil {
		return nil, err
	}

	entitlementID := models.NamespacedID{Namespace: cus.Namespace, ID: input.EntitlementID}

	ent, err := c.entitlementRepo.GetEntitlement(ctx, entitlementID)
	if err != nil {
		return nil, err
	}

	if ent.CustomerID != cus.ID {
		return nil, &entitlement.NotFoundError{EntitlementID: entitlementID}
	}

	createInput := input.Entitlement
	createInput.Namespace = cus.Namespace
	createInput.UsageAttribution = cus.GetUsageAttribution()

	return c.OverrideEntitlement(ctx, cus.ID, ent.ID, createInput, input.Grants)
}

func (c *service) GetCustomerEntitlement(ctx context.Context, input entitlement.GetCustomerEntitlementInput) (*entitlement.Entitlement, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	cus, err := c.getActiveCustomer(ctx, input.CustomerID)
	if err != nil {
		return nil, err
	}

	entitlementID := models.NamespacedID{Namespace: cus.Namespace, ID: input.EntitlementID}

	ent, err := c.entitlementRepo.GetEntitlement(ctx, entitlementID)
	if err != nil {
		return nil, err
	}

	// The entitlement is addressed through the customer, so one owned by another
	// customer must not be revealed.
	if ent.CustomerID != cus.ID {
		return nil, &entitlement.NotFoundError{EntitlementID: entitlementID}
	}

	return ent, nil
}

func (c *service) ListCustomerEntitlements(ctx context.Context, input entitlement.ListCustomerEntitlementsInput) (pagination.Result[entitlement.Entitlement], error) {
	if err := input.Validate(); err != nil {
		return pagination.Result[entitlement.Entitlement]{}, err
	}

	cus, err := c.getActiveCustomer(ctx, input.CustomerID)
	if err != nil {
		return pagination.Result[entitlement.Entitlement]{}, err
	}

	now := clock.Now()

	return c.ListEntitlements(ctx, entitlement.ListEntitlementsParams{
		Namespaces:          []string{cus.Namespace},
		CustomerIDs:         []string{cus.ID},
		FeatureID:           input.FeatureID,
		FeatureKey:          input.FeatureKey,
		EntitlementType:     input.Type,
		OrderBy:             lo.CoalesceOrEmpty(input.OrderBy, entitlement.ListEntitlementsOrderByCreatedAt),
		Order:               input.Order,
		Page:                input.Page,
		ActiveAt:            &now,
		IncludeDeletedAfter: now,
	})
}

func (c *service) DeleteCustomerEntitlement(ctx context.Context, input entitlement.DeleteCustomerEntitlementInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	cus, err := c.getActiveCustomer(ctx, input.CustomerID)
	if err != nil {
		return err
	}

	entitlementID := models.NamespacedID{Namespace: cus.Namespace, ID: input.EntitlementID}

	ent, err := c.entitlementRepo.GetEntitlement(ctx, entitlementID)
	if err != nil {
		return err
	}

	// The entitlement is addressed through the customer, so one owned by another
	// customer must not be revealed.
	if ent.CustomerID != cus.ID {
		return &entitlement.NotFoundError{EntitlementID: entitlementID}
	}

	return c.DeleteEntitlement(ctx, cus.Namespace, ent.ID, clock.Now())
}

func (c *service) GetEntitlementByID(ctx context.Context, input entitlement.GetEntitlementByIDInput) (*entitlement.Entitlement, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return c.entitlementRepo.GetEntitlement(ctx, models.NamespacedID{Namespace: input.Namespace, ID: input.EntitlementID})
}

func (c *service) ListNamespaceEntitlements(ctx context.Context, input entitlement.ListNamespaceEntitlementsInput) (pagination.Result[entitlement.Entitlement], error) {
	if err := input.Validate(); err != nil {
		return pagination.Result[entitlement.Entitlement]{}, err
	}

	now := clock.Now()

	return c.ListEntitlements(ctx, entitlement.ListEntitlementsParams{
		Namespaces:          []string{input.Namespace},
		CustomerID:          input.CustomerID,
		FeatureID:           input.FeatureID,
		FeatureKey:          input.FeatureKey,
		EntitlementType:     input.Type,
		OrderBy:             lo.CoalesceOrEmpty(input.OrderBy, entitlement.ListEntitlementsOrderByCreatedAt),
		Order:               input.Order,
		Page:                input.Page,
		ActiveAt:            &now,
		IncludeDeletedAfter: now,
	})
}

func (c *service) getActiveCustomer(ctx context.Context, customerID customer.CustomerID) (*customer.Customer, error) {
	cus, err := c.customerService.GetCustomer(ctx, customer.GetCustomerInput{
		CustomerID: &customerID,
	})
	if err != nil {
		return nil, err
	}

	if cus.IsDeleted() {
		return nil, models.NewGenericConflictError(
			fmt.Errorf("customer is deleted [namespace=%s customer.id=%s]", cus.Namespace, cus.ID),
		)
	}

	return cus, nil
}

func (c *service) GetCustomerEntitlementHistory(ctx context.Context, input entitlement.GetCustomerEntitlementHistoryInput) (entitlement.CustomerEntitlementHistory, error) {
	if err := input.Validate(); err != nil {
		return entitlement.CustomerEntitlementHistory{}, err
	}

	windowSize, err := historyWindowSize(input.WindowSize)
	if err != nil {
		return entitlement.CustomerEntitlementHistory{}, err
	}

	cus, err := c.getActiveCustomer(ctx, input.CustomerID)
	if err != nil {
		return entitlement.CustomerEntitlementHistory{}, err
	}

	ent, err := c.getCustomerEntitlement(ctx, cus, input.EntitlementID)
	if err != nil {
		return entitlement.CustomerEntitlementHistory{}, err
	}

	if ent.EntitlementType != entitlement.EntitlementTypeMetered {
		return entitlement.CustomerEntitlementHistory{}, &entitlement.WrongTypeError{
			Expected: entitlement.EntitlementTypeMetered,
			Actual:   ent.EntitlementType,
		}
	}

	windows, burndown, err := c.meteredEntitlementConnector.GetEntitlementBalanceHistory(ctx, models.NamespacedID{
		Namespace: ent.Namespace,
		ID:        ent.ID,
	}, meteredentitlement.BalanceHistoryParams{
		From:           input.From,
		To:             input.To,
		WindowSize:     windowSize,
		WindowTimeZone: *lo.CoalesceOrEmpty(input.TimeZone, time.UTC),
	})
	if err != nil {
		return entitlement.CustomerEntitlementHistory{}, err
	}

	return entitlement.CustomerEntitlementHistory{
		Windows:  windows,
		Burndown: burndown,
	}, nil
}

func (c *service) ListCustomerEntitlementGrants(ctx context.Context, input entitlement.ListCustomerEntitlementGrantsInput) (pagination.Result[grant.Grant], error) {
	if err := input.Validate(); err != nil {
		return pagination.Result[grant.Grant]{}, err
	}

	cus, err := c.getActiveCustomer(ctx, input.CustomerID)
	if err != nil {
		return pagination.Result[grant.Grant]{}, err
	}

	ent, err := c.getCustomerEntitlement(ctx, cus, input.EntitlementID)
	if err != nil {
		return pagination.Result[grant.Grant]{}, err
	}

	// The entitlement is already resolved, so the grant list is addressed by ID; the
	// list itself does not depend on the entitlement type and is empty for the
	// non-metered ones, as grants can only be issued for metered entitlements.
	grants, err := c.meteredEntitlementConnector.ListEntitlementGrants(ctx, ent.Namespace, meteredentitlement.ListEntitlementGrantsParams{
		CustomerID:                ent.CustomerID,
		EntitlementIDOrFeatureKey: ent.ID,
		IncludeDeleted:            input.IncludeDeleted,
		OrderBy:                   input.OrderBy,
		Order:                     input.Order,
		Page:                      input.Page,
	})
	if err != nil {
		return pagination.Result[grant.Grant]{}, err
	}

	return pagination.MapResult(grants, func(g meteredentitlement.EntitlementGrant) grant.Grant {
		return g.Grant
	}), nil
}

func (c *service) CreateCustomerEntitlementGrant(ctx context.Context, input entitlement.CreateCustomerEntitlementGrantInput) (grant.Grant, error) {
	if err := input.Validate(); err != nil {
		return grant.Grant{}, err
	}

	cus, err := c.getActiveCustomer(ctx, input.CustomerID)
	if err != nil {
		return grant.Grant{}, err
	}

	ent, err := c.getCustomerEntitlement(ctx, cus, input.EntitlementID)
	if err != nil {
		return grant.Grant{}, err
	}

	created, err := c.meteredEntitlementConnector.CreateGrant(ctx, ent.Namespace, ent.CustomerID, ent.ID, input.Grant)
	if err != nil {
		return grant.Grant{}, err
	}

	return created.Grant, nil
}

func (c *service) ListNamespaceGrants(ctx context.Context, input entitlement.ListNamespaceGrantsInput) (pagination.Result[grant.Grant], error) {
	if err := input.Validate(); err != nil {
		return pagination.Result[grant.Grant]{}, err
	}

	return c.grantRepo.ListGrants(ctx, grant.ListParams{
		Namespace:      input.Namespace,
		IncludeDeleted: input.IncludeDeleted,
		CustomerID:     input.CustomerID,
		FeatureID:      input.FeatureID,
		OrderBy:        lo.CoalesceOrEmpty(input.OrderBy, grant.OrderByCreatedAt),
		Order:          input.Order,
		Page:           input.Page,
	})
}

func (c *service) VoidGrant(ctx context.Context, input entitlement.VoidGrantInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	err := c.grantConnector.VoidGrant(ctx, input.GrantID, input.At)
	if _, ok := lo.ErrorsAs[*credit.GrantNotFoundError](err); ok {
		return models.NewGenericNotFoundError(err)
	}

	return err
}

// historyWindowSize rejects the meter window sizes the balance history cannot be
// calculated with; sub-hour windows are too expensive to compute.
func historyWindowSize(size meter.WindowSize) (meteredentitlement.WindowSize, error) {
	switch size {
	case meter.WindowSizeHour:
		return meteredentitlement.WindowSizeHour, nil
	case meter.WindowSizeDay:
		return meteredentitlement.WindowSizeDay, nil
	default:
		return "", models.NewGenericValidationError(fmt.Errorf("unsupported window size %q", size))
	}
}

func (c *service) ResetCustomerEntitlementUsage(ctx context.Context, input entitlement.ResetCustomerEntitlementUsageInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	cus, err := c.getActiveCustomer(ctx, input.CustomerID)
	if err != nil {
		return err
	}

	ent, err := c.getCustomerEntitlement(ctx, cus, input.EntitlementID)
	if err != nil {
		return err
	}

	if ent.EntitlementType != entitlement.EntitlementTypeMetered {
		return &entitlement.WrongTypeError{
			Expected: entitlement.EntitlementTypeMetered,
			Actual:   ent.EntitlementType,
		}
	}

	_, err = c.meteredEntitlementConnector.ResetEntitlementUsage(ctx, models.NamespacedID{
		Namespace: ent.Namespace,
		ID:        ent.ID,
	}, meteredentitlement.ResetEntitlementUsageParams{
		At:              lo.FromPtrOr(input.EffectiveAt, clock.Now()),
		RetainAnchor:    input.RetainAnchor,
		PreserveOverage: input.PreserveOverage,
	})

	return err
}

// getCustomerEntitlement resolves an entitlement addressed through its customer.
// An entitlement owned by another customer is reported as not found so the
// customer scope does not reveal it.
func (c *service) getCustomerEntitlement(ctx context.Context, cus *customer.Customer, entitlementID string) (*entitlement.Entitlement, error) {
	id := models.NamespacedID{Namespace: cus.Namespace, ID: entitlementID}

	ent, err := c.entitlementRepo.GetEntitlement(ctx, id)
	if err != nil {
		return nil, err
	}

	if ent.CustomerID != cus.ID {
		return nil, &entitlement.NotFoundError{EntitlementID: id}
	}

	return ent, nil
}
