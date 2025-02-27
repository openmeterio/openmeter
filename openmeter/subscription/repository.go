package subscription

import (
	"context"
	"maps"
	"reflect"
	"time"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type CreateSubscriptionEntityInput struct {
	models.CadencedModel
	models.NamespacedModel
	models.AnnotatedModel

	productcatalog.Alignment

	Plan        *PlanRef
	Name        string  `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`

	CustomerId string `json:"customerId,omitempty"`
	Currency   currencyx.Code
}

type SubscriptionRepository interface {
	entutils.TxCreator

	models.CadencedResourceRepo[Subscription]

	// Returns all subscriptions active or scheduled after the given timestamp
	GetAllForCustomerSince(ctx context.Context, customerID models.NamespacedID, at time.Time) ([]Subscription, error)

	// Returns the subscription by ID
	GetByID(ctx context.Context, subscriptionID models.NamespacedID) (Subscription, error)

	// Create a new subscription
	Create(ctx context.Context, input CreateSubscriptionEntityInput) (Subscription, error)

	// Delete a subscription
	Delete(ctx context.Context, id models.NamespacedID) error

	// List subscriptions
	List(ctx context.Context, params ListSubscriptionsInput) (pagination.PagedResponse[Subscription], error)
}

type CreateSubscriptionPhaseEntityInput struct {
	models.NamespacedModel
	models.AnnotatedModel

	// ActiveFrom is the time the phase becomes active.
	ActiveFrom time.Time

	// SubscriptionID is the ID of the subscription this phase belongs to.
	SubscriptionID string `json:"subscriptionId"`

	// Key is the unique key for Phase.
	Key string `json:"key"`

	// Name
	Name string `json:"name"`

	// Description
	Description *string `json:"description,omitempty"`

	// StartAfter
	StartAfter isodate.Period `json:"interval"`
}

func (i CreateSubscriptionPhaseEntityInput) Equal(other CreateSubscriptionPhaseEntityInput) bool {
	return reflect.DeepEqual(i, other)
}

type SubscriptionPhaseRepository interface {
	entutils.TxCreator

	// Returns the phases for a subscription
	GetForSubscriptionAt(ctx context.Context, subscriptionID models.NamespacedID, at time.Time) ([]SubscriptionPhase, error)

	// Create a new subscription phase
	Create(ctx context.Context, input CreateSubscriptionPhaseEntityInput) (SubscriptionPhase, error)
	Delete(ctx context.Context, id models.NamespacedID) error
}

type CreateSubscriptionItemEntityInput struct {
	models.NamespacedModel
	models.AnnotatedModel

	ActiveFromOverrideRelativeToPhaseStart *isodate.Period
	ActiveToOverrideRelativeToPhaseStart   *isodate.Period

	models.CadencedModel

	BillingBehaviorOverride BillingBehaviorOverride

	// PhaseID is the ID of the phase this item belongs to.
	PhaseID string

	// Key is the unique key of the item in the phase.
	Key string

	RateCard RateCard

	EntitlementID *string
	Name          string  `json:"name,omitempty"`
	Description   *string `json:"description,omitempty"`
}

func (i CreateSubscriptionItemEntityInput) Equal(other CreateSubscriptionItemEntityInput) bool {
	a := i
	a.AnnotatedModel = models.AnnotatedModel{}
	b := other
	b.AnnotatedModel = models.AnnotatedModel{}

	return reflect.DeepEqual(a, b) && maps.Equal(i.Metadata, other.Metadata)
}

type SubscriptionItemRepository interface {
	entutils.TxCreator

	GetForSubscriptionAt(ctx context.Context, subscriptionID models.NamespacedID, at time.Time) ([]SubscriptionItem, error)

	Create(ctx context.Context, input CreateSubscriptionItemEntityInput) (SubscriptionItem, error)
	Delete(ctx context.Context, id models.NamespacedID) error
	GetByID(ctx context.Context, id models.NamespacedID) (SubscriptionItem, error)
}
