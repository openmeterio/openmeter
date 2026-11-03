package subscription

import (
	"context"
	"errors"
	"fmt"
	"time"

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

type GetForItemsAtInput struct {
	Items []SubscriptionItem
	At    time.Time
}

func (i GetForItemsAtInput) Validate() error {
	var errs []error

	if i.At.IsZero() {
		errs = append(errs, errors.New("at is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type EntitlementAdapter interface {
	ScheduleEntitlement(ctx context.Context, input ScheduleSubscriptionEntitlementInput, annotations models.Annotations) (*SubscriptionEntitlement, error)

	// GetForItemsAt returns the entitlements of the given items as they were at At. It takes the
	// caller's items instead of reading them again so a concurrent edit can't make the two disagree.
	GetForItemsAt(ctx context.Context, input GetForItemsAtInput) ([]SubscriptionEntitlement, error)

	DeleteByItemID(ctx context.Context, itemId models.NamespacedID) error
}
