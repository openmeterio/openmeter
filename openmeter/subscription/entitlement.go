package subscription

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/models"
)

type SubscriptionEntitlement struct {
	Entitlement entitlement.EntitlementWithCustomer
	Cadence     models.CadencedModel
}

func (s SubscriptionEntitlement) Validate() error {
	if s.Entitlement.ActiveFrom == nil {
		return fmt.Errorf("entitlement active from is nil")
	}
	if !s.Cadence.ActiveFrom.Equal(*s.Entitlement.ActiveFrom) {
		return fmt.Errorf("entitlement active from %v does not match cadence active from %v", s.Entitlement.ActiveFrom, s.Cadence.ActiveFrom)
	}
	if s.Entitlement.ActiveTo == nil {
		if s.Cadence.ActiveTo != nil {
			return fmt.Errorf("entitlement active to is nil, but cadence active to is %v", s.Cadence.ActiveTo)
		}
	} else {
		if s.Cadence.ActiveTo == nil {
			return fmt.Errorf("entitlement active to is %v, but cadence active to is nil", s.Entitlement.ActiveTo)
		}
		if !s.Entitlement.ActiveTo.Equal(*s.Cadence.ActiveTo) {
			return fmt.Errorf("entitlement active to %v does not match cadence active to %v", s.Entitlement.ActiveTo, s.Cadence.ActiveTo)
		}
	}
	return nil
}

func (s SubscriptionEntitlement) ToScheduleSubscriptionEntitlementInput() ScheduleSubscriptionEntitlementInput {
	return ScheduleSubscriptionEntitlementInput{
		CreateEntitlementInputs: s.Entitlement.AsCreateEntitlementInputs(s.Entitlement.Customer),
	}
}

type ScheduleSubscriptionEntitlementInput struct {
	entitlement.CreateEntitlementInputs
	Customer customer.Customer
}

func (s ScheduleSubscriptionEntitlementInput) Equal(other ScheduleSubscriptionEntitlementInput) bool {
	return s.CreateEntitlementInputs.Equal(other.CreateEntitlementInputs)
}

func (s ScheduleSubscriptionEntitlementInput) Validate() error {
	if s.CreateEntitlementInputs.ActiveFrom == nil {
		return fmt.Errorf("entitlement active from is nil")
	}

	if err := s.Customer.Validate(); err != nil {
		return fmt.Errorf("customer is invalid: %w", err)
	}

	return nil
}

type EntitlementAdapter interface {
	ScheduleEntitlement(ctx context.Context, input ScheduleSubscriptionEntitlementInput, annotations models.Annotations) (*SubscriptionEntitlement, error)
	// At refers to a point in time for which we're querying the system state, meaning:
	// if t1 < t2 < t3, and some entitlement was deleted effective at t2, then
	// with at = t1 the entitlement will be returned, while with at = t3 it won't.
	GetForSubscriptionAt(ctx context.Context, input GetForSubscriptionAtInput) ([]SubscriptionEntitlement, error)

	GetForSubscriptionsAt(ctx context.Context, input []GetForSubscriptionAtInput) ([]SubscriptionEntitlement, error)

	DeleteByItemID(ctx context.Context, itemId models.NamespacedID) error
}
