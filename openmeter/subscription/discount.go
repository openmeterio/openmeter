package subscription

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

type AppliedDiscount struct {
	// Priority is the order in which the discount is applied. Lower values are applied first.
	Priority int `json:"priority"`
	// AppliesToKeys is a list of SubscriptionItem keys that this discount applies to.
	AppliesToKeys []string `json:"appliesToKeys"`

	Discount productcatalog.Discount
}

type SubscriptionPhaseDiscount struct {
	models.NamespacedID
	models.ManagedModel

	// SubscriptionPhaseId is the ID of the phase this Discount belongs to.
	SubscriptionPhaseId string `json:"subscriptionPhaseId"`
}

type AddSubscriptionPhaseDiscountInput struct {
	SubscriptionPhaseId models.NamespacedID `json:"subscriptionPhaseId"`
	AppliedDiscount
}

type DiscountAdapter interface {
	Add(ctx context.Context, input AddSubscriptionPhaseDiscountInput) (SubscriptionPhaseDiscount, error)
	Remove(ctx context.Context, id models.NamespacedID) error
	GetForPhase(ctx context.Context, phaseId models.NamespacedID) ([]SubscriptionPhaseDiscount, error)
}
