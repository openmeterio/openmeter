package subscription

import (
	"context"
	"maps"
	"reflect"
	"time"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

type CreateSubscriptionEntityInput struct {
	models.CadencedModel
	models.NamespacedModel
	models.MetadataModel

	Annotations models.Annotations `json:"annotations"`

	Plan        *PlanRef
	Name        string  `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`

	CustomerId string `json:"customerId,omitempty"`
	Currency   currencyx.Code

	// BillingCadence is the default billing cadence for subscriptions.
	BillingCadence datetime.ISODuration `json:"billing_cadence"`

	// ProRatingConfig is the default pro-rating configuration for subscriptions.
	ProRatingConfig productcatalog.ProRatingConfig `json:"pro_rating_config"`

	// BillingAnchor is the time the subscription will be billed.
	BillingAnchor time.Time `json:"billingAnchor"`
}

type SubscriptionRepository interface {
	entutils.TxCreator

	models.CadencedResourceRepo[Subscription]

	// Returns the subscription by ID
	GetByID(ctx context.Context, subscriptionID models.NamespacedID) (Subscription, error)

	// Create a new subscription
	Create(ctx context.Context, input CreateSubscriptionEntityInput) (Subscription, error)

	// Delete a subscription
	Delete(ctx context.Context, id models.NamespacedID) error

	// List subscriptions
	List(ctx context.Context, params ListSubscriptionsInput) (SubscriptionList, error)

	// UpdateAnnotations updates the annotations of a subscription
	UpdateAnnotations(ctx context.Context, id models.NamespacedID, annotations models.Annotations) (*Subscription, error)
}

type CreateSubscriptionPhaseEntityInput struct {
	models.NamespacedModel
	models.MetadataModel

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
	StartAfter datetime.ISODuration `json:"interval"`

	// SortHint
	SortHint *uint8 `json:"sortHint,omitempty"`
}

func (i CreateSubscriptionPhaseEntityInput) Equal(other CreateSubscriptionPhaseEntityInput) bool {
	return reflect.DeepEqual(i, other)
}

type GetForSubscriptionAtInput struct {
	Namespace      string
	SubscriptionID string
	At             time.Time
}

type SubscriptionPhaseRepository interface {
	entutils.TxCreator

	// Returns the phases for a subscription
	GetForSubscriptionAt(ctx context.Context, input GetForSubscriptionAtInput) ([]SubscriptionPhase, error)
	// Returns the phases for a list of subscriptions
	GetForSubscriptionsAt(ctx context.Context, input []GetForSubscriptionAtInput) ([]SubscriptionPhase, error)

	// Create a new subscription phase
	Create(ctx context.Context, input CreateSubscriptionPhaseEntityInput) (SubscriptionPhase, error)
	Delete(ctx context.Context, id models.NamespacedID) error
}

type CreateSubscriptionItemEntityInput struct {
	models.NamespacedModel
	models.MetadataModel

	Annotations models.Annotations `json:"annotations"`

	ActiveFromOverrideRelativeToPhaseStart *datetime.ISODuration
	ActiveToOverrideRelativeToPhaseStart   *datetime.ISODuration

	models.CadencedModel

	BillingBehaviorOverride BillingBehaviorOverride

	// PhaseID is the ID of the phase this item belongs to.
	PhaseID string

	// Key is the unique key of the item in the phase.
	Key string

	RateCard productcatalog.RateCard

	EntitlementID *string
	Name          string  `json:"name,omitempty"`
	Description   *string `json:"description,omitempty"`
}

func (i CreateSubscriptionItemEntityInput) Equal(other CreateSubscriptionItemEntityInput) bool {
	a := i
	a.MetadataModel = models.MetadataModel{}
	a.Annotations = models.Annotations{}
	b := other
	b.MetadataModel = models.MetadataModel{}
	b.Annotations = models.Annotations{}

	// we don't compare annotations
	return reflect.DeepEqual(a, b) && maps.Equal(i.Metadata, other.Metadata)
}

type SubscriptionItemRepository interface {
	entutils.TxCreator

	GetForSubscriptionAt(ctx context.Context, inp GetForSubscriptionAtInput) ([]SubscriptionItem, error)
	GetForSubscriptionsAt(ctx context.Context, input []GetForSubscriptionAtInput) ([]SubscriptionItem, error)

	Create(ctx context.Context, input CreateSubscriptionItemEntityInput) (SubscriptionItem, error)
	Delete(ctx context.Context, id models.NamespacedID) error
	GetByID(ctx context.Context, id models.NamespacedID) (SubscriptionItem, error)
}
