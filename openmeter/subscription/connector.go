package subscription

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/samber/lo"
)

type NewSubscriptionRequest struct {
	Namespace  string
	ActiveFrom time.Time
	CustomerID string
	Currency   models.CurrencyCode

	Plan struct {
		Key     string
		Version int
	}

	// The SubscriptionItem customizations compared to the plan
	ItemCustomization []Patch

	// TODO: Add discounts, either separately or as part of the patch language
}

type Connector interface {
	Create(ctx context.Context, req NewSubscriptionRequest) (Subscription, error)
	Edit(ctx context.Context, subscriptionID string, patches []Patch) (Subscription, error)
	Cancel(ctx context.Context, subscriptionID string, at time.Time) (Subscription, error)
}

type connector struct {
	repo Repository
	// adapters
	billingAdapter BillingAdapter
	planAdapter    PlanAdapter
}

func NewConnector() Connector {
	return &connector{}
}

func (c *connector) Create(ctx context.Context, req NewSubscriptionRequest) (Subscription, error) {
	def := Subscription{}

	// If user has a plan right now return an error
	_, err := c.repo.GetCustomerSubscription(ctx, req.CustomerID)
	if err != nil {
		if _, ok := lo.ErrorsAs[*NotFoundError](err); !ok {
			return def, err
		}
	} else {
		return def, &models.GenericUserError{Message: "customer already has a subscription"}
	}
	// Fetch the plan, check if it exists
	plan, err := c.planAdapter.GetVersion(ctx, req.Plan.Key, req.Plan.Version)
	if err != nil {
		return def, err
	}
	// Check that all customizations are valid, only RateCard overrides are allowed
	for _, patch := range req.ItemCustomization {
		if err := patch.Path().Validate(); err != nil {
			return def, err
		}
	}
	// Apply customizations
	spec, err := SpecFromPlan(plan, CreateSubscriptionCustomerInput{
		Currency:   req.Currency,
		CustomerId: req.CustomerID,
		ActiveFrom: req.ActiveFrom,
	})
	if err != nil {
		return def, fmt.Errorf("failed to create subscription spec from plan: %w", err)
	}

	err = spec.ApplyPatches(lo.Map(req.ItemCustomization, ToApplies), ApplyContext{
		Operation:   SpecOperationCreate,
		CurrentTime: clock.Now(),
	})
	if err != nil {
		return def, fmt.Errorf("failed to apply customizations: %w", err)
	}

	// Iterate through each phase & create phases and items
	// Return sub reference
	panic("implement me")
}

func (c *connector) Edit(ctx context.Context, subscriptionID string, patches []Patch) (Subscription, error) {
	panic("implement me")
}

func (c *connector) Cancel(ctx context.Context, subscriptionID string, at time.Time) (Subscription, error) {
	panic("implement me")
}
