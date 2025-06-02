package subscriptionworkflow

import (
	"context"
	"errors"
	"time"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Service interface {
	CreateFromPlan(ctx context.Context, inp CreateSubscriptionWorkflowInput, plan subscription.Plan) (subscription.SubscriptionView, error)
	EditRunning(ctx context.Context, subscriptionID models.NamespacedID, customizations []subscription.Patch, timing subscription.Timing) (subscription.SubscriptionView, error)
	ChangeToPlan(ctx context.Context, subscriptionID models.NamespacedID, inp ChangeSubscriptionWorkflowInput, plan subscription.Plan) (current subscription.Subscription, new subscription.SubscriptionView, err error)
	Restore(ctx context.Context, subscriptionID models.NamespacedID) (subscription.Subscription, error)

	AddAddon(ctx context.Context, subscriptionID models.NamespacedID, addonInp AddAddonWorkflowInput) (subscription.SubscriptionView, subscriptionaddon.SubscriptionAddon, error)
	ChangeAddonQuantity(ctx context.Context, subscriptionID models.NamespacedID, changeInp ChangeAddonQuantityWorkflowInput) (subscription.SubscriptionView, subscriptionaddon.SubscriptionAddon, error)
}

type CreateSubscriptionWorkflowInput struct {
	ChangeSubscriptionWorkflowInput
	Namespace  string
	CustomerID string

	BillingAnchor *time.Time `json:"billingAnchor,omitempty"`
}

type ChangeSubscriptionWorkflowInput struct {
	subscription.Timing
	models.MetadataModel
	Name        string
	Description *string
}

type AddAddonWorkflowInput struct {
	models.MetadataModel

	AddonID string `json:"addonID"`

	InitialQuantity int `json:"initialQuantity"`

	Timing subscription.Timing `json:"timing"`
}

func (i AddAddonWorkflowInput) Validate() error {
	if i.AddonID == "" {
		return errors.New("addonID is required")
	}

	if i.InitialQuantity <= 0 {
		return errors.New("initialQuantity must be greater than 0")
	}

	return nil
}

type ChangeAddonQuantityWorkflowInput struct {
	SubscriptionAddonID models.NamespacedID

	Quantity int `json:"quantity"`

	Timing subscription.Timing `json:"timing"`
}

func (i ChangeAddonQuantityWorkflowInput) Validate() error {
	return nil
}
