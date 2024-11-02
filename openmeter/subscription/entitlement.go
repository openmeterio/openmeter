package subscription

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/models"
)

type SubscriptionEntitlement struct {
	Entitlement entitlement.Entitlement
	Cadence     models.CadencedModel
	ItemRef     SubscriptionItemRef
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

func (s SubscriptionEntitlement) AsSpec() *SubscriptionEntitlementSpec {
	return &SubscriptionEntitlementSpec{
		EntitlementInputs: s.Entitlement.AsCreateEntitlementInputs(),
		Cadence:           s.Cadence,
		ItemRef:           s.ItemRef,
	}
}

type SubscriptionEntitlementSpec struct {
	EntitlementInputs entitlement.CreateEntitlementInputs
	Cadence           models.CadencedModel
	ItemRef           SubscriptionItemRef
}

func (s SubscriptionEntitlementSpec) Self() SubscriptionEntitlementSpec {
	return s
}

func (s SubscriptionEntitlementSpec) Equal(other SubscriptionEntitlementSpec) bool {
	return reflect.DeepEqual(s, other)
}

func (s SubscriptionEntitlementSpec) Validate() error {
	if s.EntitlementInputs.ActiveFrom == nil {
		return fmt.Errorf("entitlement active from is nil")
	}
	if !s.Cadence.ActiveFrom.Equal(*s.EntitlementInputs.ActiveFrom) {
		return fmt.Errorf("entitlement active from %v does not match cadence active from %v", s.EntitlementInputs.ActiveFrom, s.Cadence.ActiveFrom)
	}
	if s.EntitlementInputs.ActiveTo == nil {
		if s.Cadence.ActiveTo != nil {
			return fmt.Errorf("entitlement active to is nil, but cadence active to is %v", s.Cadence.ActiveTo)
		}
	} else {
		if s.Cadence.ActiveTo == nil {
			return fmt.Errorf("entitlement active to is %v, but cadence active to is nil", s.EntitlementInputs.ActiveTo)
		}
		if !s.EntitlementInputs.ActiveTo.Equal(*s.Cadence.ActiveTo) {
			return fmt.Errorf("entitlement active to %v does not match cadence active to %v", s.EntitlementInputs.ActiveTo, s.Cadence.ActiveTo)
		}
	}
	return nil
}

type EntitlementAdapter interface {
	ScheduleEntitlement(ctx context.Context, ref SubscriptionItemRef, input entitlement.CreateEntitlementInputs) (*SubscriptionEntitlement, error)
	// // At refers to a point in time for which we're querying the system state, meaning:
	// if t1 < t2 < t3, and some entitlement was deleted effective at t2, then
	// with at = t1 the entitlement will be returned, while with at = t3 it won't.
	//
	// As SubscriptionItemRef is a stable ref while the underlying entitlement might change,
	// logically changed entitlemnets have to be deleted.
	GetForItem(ctx context.Context, namespace string, ref SubscriptionItemRef, at time.Time) (*SubscriptionEntitlement, error)
	// At refers to a point in time for which we're querying the system state, meaning:
	// if t1 < t2 < t3, and some entitlement was deleted effective at t2, then
	// with at = t1 the entitlement will be returned, while with at = t3 it won't.
	GetForSubscription(ctx context.Context, subscriptionID models.NamespacedID, at time.Time) ([]SubscriptionEntitlement, error)

	Delete(ctx context.Context, namespace string, ref SubscriptionItemRef) error
}
