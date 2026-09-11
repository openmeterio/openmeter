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
	MigrateToPlan(ctx context.Context, input MigrateSubscriptionWorkflowInput) (subscription.SubscriptionView, error)
	ChangeToPlan(ctx context.Context, subscriptionID models.NamespacedID, inp ChangeSubscriptionWorkflowInput, plan subscription.Plan) (current subscription.Subscription, new subscription.SubscriptionView, err error)
	Restore(ctx context.Context, subscriptionID models.NamespacedID) (subscription.Subscription, error)

	AddAddon(ctx context.Context, subscriptionID models.NamespacedID, addonInp AddAddonWorkflowInput) (subscription.SubscriptionView, subscriptionaddon.SubscriptionAddon, error)
	ChangeAddonQuantity(ctx context.Context, subscriptionID models.NamespacedID, changeInp ChangeAddonQuantityWorkflowInput) (subscription.SubscriptionView, subscriptionaddon.SubscriptionAddon, error)
}

type MigrateSubscriptionWorkflowInput struct {
	SubscriptionID models.NamespacedID
	Plan           subscription.Plan
	Timing         subscription.Timing
}

func (i MigrateSubscriptionWorkflowInput) Validate() error {
	var errs []error
	if i.SubscriptionID.Namespace == "" || i.SubscriptionID.ID == "" {
		errs = append(errs, errors.New("subscription namespace and ID are required"))
	}
	if i.Plan == nil || i.Plan.ToCreateSubscriptionPlanInput().Plan == nil {
		errs = append(errs, errors.New("a catalog plan is required"))
	}
	if err := i.Timing.Validate(); err != nil {
		errs = append(errs, err)
	}
	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type CreateSubscriptionWorkflowInput struct {
	ChangeSubscriptionWorkflowInput
	Namespace  string
	CustomerID string

	BillingAnchor *time.Time `json:"billingAnchor,omitempty"`
	Annotations   models.Annotations
}

type ChangeSubscriptionWorkflowInput struct {
	subscription.Timing
	models.MetadataModel
	Name          string
	Description   *string
	CostBasisMode subscription.CostBasisMode

	BillingAnchor *time.Time `json:"billingAnchor,omitempty"`
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
