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

type Command interface {
	Create(ctx context.Context, req NewSubscriptionRequest) (Subscription, error)
	Edit(ctx context.Context, subscriptionID string, patches []Patch) (Subscription, error)
	Cancel(ctx context.Context, subscriptionID string, at time.Time) (Subscription, error)
}

type command struct {
	repo Repository
	// adapters
	billingAdapter BillingAdapter
	planAdapter    PlanAdapter
}

func NewConnector() Command {
	return &command{}
}

func (c *command) Create(ctx context.Context, req NewSubscriptionRequest) (Subscription, error) {
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

	// Check that all customizations are valid
	for _, patch := range req.ItemCustomization {
		if err := patch.Path().Validate(); err != nil {
			return def, err
		}
	}

	// Get the default spec based on the Plan
	spec, err := SpecFromPlan(plan, CreateSubscriptionCustomerInput{
		Currency:   req.Currency,
		CustomerId: req.CustomerID,
		ActiveFrom: req.ActiveFrom,
	})
	if err != nil {
		return def, fmt.Errorf("failed to create subscription spec from plan: %w", err)
	}

	// Apply customizations
	err = spec.ApplyPatches(lo.Map(req.ItemCustomization, ToApplies), ApplyContext{
		Operation:   SpecOperationCreate,
		CurrentTime: clock.Now(),
	})
	if err != nil {
		return def, fmt.Errorf("failed to apply customizations: %w", err)
	}

	// Create subscription entity
	_, err = c.repo.CreateSubscription(ctx, spec.GetCreateInput())
	if err != nil {
		return def, err
	}

	// Iterate through each phase & create phases and items
	for _, phase := range spec.Phases {
		err := c.createPhase(ctx, spec, phase.PhaseKey)
		if err != nil {
			return def, err
		}
	}
	// Return sub reference
	panic("implement me")
}

func (c *command) createPhase(ctx context.Context, spec *SubscriptionSpec, phaseKey string) error {
	if spec == nil {
		return fmt.Errorf("spec is nil")
	}

	_, exists := spec.Phases[phaseKey]
	if !exists {
		return fmt.Errorf("phase %s does not exist", phaseKey)
	}

	panic("implement me")
}

func (c *command) Edit(ctx context.Context, subscriptionID string, patches []Patch) (Subscription, error) {
	panic("implement me")
}

func (c *command) Cancel(ctx context.Context, subscriptionID string, at time.Time) (Subscription, error) {
	panic("implement me")
}

type Query interface {
	Get(ctx context.Context, subscriptionID string) (Subscription, error)
	Expand(ctx context.Context, subscriptionID string) (SubscriptionView, error)
}

type query struct {
	repo Repository

	planAdapter PlanAdapter
}

func NewQuery() Query {
	return &query{}
}

func (q *query) Get(ctx context.Context, subscriptionID string) (Subscription, error) {
	sub, err := q.repo.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return Subscription{}, err
	}
	return sub, nil
}

func (q *query) Expand(ctx context.Context, subscriptionID string) (SubscriptionView, error) {
	// sub, err := q.Get(ctx, subscriptionID)
	// if err != nil {
	// 	return nil, err
	// }

	// patches, err := q.repo.GetPatches(ctx, subscriptionID)
	// if err != nil {
	// 	return nil, err
	// }

	// plan, err := q.planAdapter.GetVersion(ctx, sub.Plan.Key, sub.Plan.Version)
	// if err != nil {
	// 	return nil, err
	// }

	// // Get the default spec based on the Plan
	// spec, err := SpecFromPlan(plan, CreateSubscriptionCustomerInput{
	// 	Currency:   sub.Currency,
	// 	CustomerId: sub.CustomerId,
	// 	ActiveFrom: sub.ActiveFrom,
	// })
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to create subscription spec from plan: %w", err)
	// }

	// // Apply customizations
	// err = spec.ApplyPatches(lo.Map(patches, func(p SubscriptionPatch, i int) Applies  {

	// }), ApplyContext{
	// 	Operation:   SpecOperationCreate,
	// 	CurrentTime: clock.Now(),
	// })
	// if err != nil {
	// 	return def, fmt.Errorf("failed to apply customizations: %w", err)
	// }

	panic("implement me")
}
