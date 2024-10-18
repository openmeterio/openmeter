package subscription

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/customer"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/subscription/price"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/samber/lo"
)

type NewSubscriptionRequest struct {
	Namespace  string
	ActiveFrom time.Time
	CustomerID string
	Currency   currencyx.Code

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
	// connectors
	priceConnector  price.Connector
	customerService customer.Service
	// adapters
	billingAdapter     BillingAdapter
	planAdapter        PlanAdapter
	entitlementAdapter EntitlementAdapter
	// framework
	transactionManager transaction.Creator
}

func NewCommand() Command {
	return &command{}
}

func (c *command) Create(ctx context.Context, req NewSubscriptionRequest) (Subscription, error) {
	def := Subscription{}
	currentTime := clock.Now()

	// Fetch the customer
	cust, err := c.customerService.GetCustomer(ctx, customerentity.GetCustomerInput{
		Namespace: req.Namespace,
		ID:        req.CustomerID,
	})
	if err != nil {
		return def, err
	}

	if cust == nil {
		return def, fmt.Errorf("customer is nil")
	}

	// If user has a plan right now return an error
	_, err = c.repo.GetCustomerSubscription(ctx, models.NamespacedID{
		ID:        req.CustomerID,
		Namespace: req.Namespace,
	})
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

	return transaction.Run(ctx, c.transactionManager, func(ctx context.Context) (Subscription, error) {
		// Create subscription entity
		sub, err := c.repo.CreateSubscription(ctx, req.Namespace, spec.GetCreateInput())
		if err != nil {
			return def, err
		}

		// Iterate through each phase & create phases and items
		for _, phase := range spec.Phases {
			err := c.createPhase(ctx, sub, *cust, spec, phase.PhaseKey)
			if err != nil {
				return def, fmt.Errorf("failed to create phase %s: %w", phase.PhaseKey, err)
			}
		}

		// Once everything is succesful, lets save the patches
		patchInputs, err := TransformPatchesForRepository(req.ItemCustomization, currentTime)
		if err != nil {
			return def, fmt.Errorf("failed to transform patches for repository: %w", err)
		}
		_, err = c.repo.CreateSubscriptionPatches(ctx, models.NamespacedID{
			ID:        sub.ID,
			Namespace: sub.Namespace,
		}, patchInputs)
		if err != nil {
			return def, fmt.Errorf("failed to create subscription patches: %w", err)
		}
		// Return sub reference
		return sub, nil
	})
}

func (c *command) createPhase(ctx context.Context, sub Subscription, cust customerentity.Customer, spec *SubscriptionSpec, phaseKey string) error {
	if spec == nil {
		return fmt.Errorf("spec is nil")
	}

	phase, exists := spec.Phases[phaseKey]
	if !exists {
		return fmt.Errorf("phase %s does not exist", phaseKey)
	}

	phaseStartTime, _ := phase.StartAfter.AddTo(sub.ActiveFrom)
	var phaseEndTime *time.Time

	// Find the next phase if any
	sortedPhases := spec.GetSortedPhases()
	for i, p := range sortedPhases {
		if p.PhaseKey == phaseKey && i+1 < len(sortedPhases) {
			nextPhase := sortedPhases[i+1]
			et, _ := nextPhase.StartAfter.AddTo(sub.ActiveFrom)
			phaseEndTime = &et
			break
		}
	}

	cadence := models.CadencedModel{
		ActiveFrom: phaseStartTime,
		ActiveTo:   phaseEndTime,
	}

	return transaction.RunWithNoValue(ctx, c.transactionManager, func(ctx context.Context) error {
		for _, item := range phase.Items {
			// Create Entitlement
			if item.CreateEntitlementInput != nil {
				if len(cust.UsageAttribution.SubjectKeys) == 0 {
					return fmt.Errorf("customer has no subject keys")
				}
				customerSubject := cust.UsageAttribution.SubjectKeys[0]

				if item.FeatureKey == nil {
					return fmt.Errorf("item %s has no feature key, cannot create entitlement", item.ItemKey)
				}

				input, err := item.CreateEntitlementInput.ToCreateEntitlementInput(
					sub.Namespace,
					*item.FeatureKey,
					customerSubject,
					cadence,
				)
				if err != nil {
					return fmt.Errorf("failed to create entitlement input for item %s: %w", item.ItemKey, err)
				}
				if input == nil {
					return fmt.Errorf("entitlement input is nil")
				}

				// FIXME: this will fail if there's already an entitlement present
				_, err = c.entitlementAdapter.ScheduleEntitlement(ctx, SubscriptionItemRef{
					SubscriptionId: sub.ID,
					PhaseKey:       phaseKey,
					ItemKey:        item.ItemKey,
				}, *input)
				if err != nil {
					return fmt.Errorf("failed to create entitlement for item %s: %w", item.ItemKey, err)
				}
			}
			// Create Price
			if item.CreatePriceInput != nil {
				// TODO: link price to Item & Phase
				_, err := c.priceConnector.Create(ctx, price.CreateInput{
					SubscriptionId: models.NamespacedID{
						Namespace: sub.Namespace,
						ID:        sub.ID,
					},
					Spec:          *item.CreatePriceInput,
					CadencedModel: cadence,
				})
				if err != nil {
					return fmt.Errorf("failed to create price for item %s: %w", item.ItemKey, err)
				}
			}
		}

		// TODO: Write discounts!
		return nil
	})
}

func (c *command) Edit(ctx context.Context, subscriptionID string, patches []Patch) (Subscription, error) {
	panic("implement me")
}

func (c *command) Cancel(ctx context.Context, subscriptionID string, at time.Time) (Subscription, error) {
	panic("implement me")
}

type Query interface {
	Get(ctx context.Context, subscriptionID models.NamespacedID) (Subscription, error)
	Expand(ctx context.Context, subscriptionID models.NamespacedID) (SubscriptionView, error)
}

type query struct {
	repo Repository
	// connectors
	priceConnector   price.Connector
	featureConnector feature.FeatureConnector
	// adapters
	planAdapter        PlanAdapter
	entitlementAdapter EntitlementAdapter
}

func NewQuery() Query {
	return &query{}
}

func (q *query) Get(ctx context.Context, subscriptionID models.NamespacedID) (Subscription, error) {
	sub, err := q.repo.GetSubscription(ctx, models.NamespacedID{})
	if err != nil {
		return Subscription{}, err
	}
	return sub, nil
}

func (q *query) Expand(ctx context.Context, subscriptionID models.NamespacedID) (SubscriptionView, error) {
	currentTime := clock.Now()
	sub, err := q.Get(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}

	patches, err := q.repo.GetSubscriptionPatches(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}

	plan, err := q.planAdapter.GetVersion(ctx, sub.Plan.Key, sub.Plan.Version)
	if err != nil {
		return nil, err
	}

	// Get the default spec based on the Plan
	spec, err := SpecFromPlan(plan, CreateSubscriptionCustomerInput{
		Currency:   sub.Currency,
		CustomerId: sub.CustomerId,
		ActiveFrom: sub.ActiveFrom,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription spec from plan: %w", err)
	}

	// Map patches to applies
	applies := make([]Applies, 0, len(patches))
	for _, sp := range patches {
		a, err := sp.AsPatch()
		if err != nil {
			return nil, err
		}
		if p, ok := a.(Applies); !ok {
			return nil, fmt.Errorf("failed to convert patch to applies")
		} else {
			applies = append(applies, p)
		}
	}

	// Apply customizations
	err = spec.ApplyPatches(applies, ApplyContext{
		Operation:   SpecOperationCreate,
		CurrentTime: clock.Now(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to apply customizations: %w", err)
	}

	// Let's fetch all dependent entities in batches and then match them to the spec
	ents, err := q.entitlementAdapter.GetForSubscription(ctx, models.NamespacedID{
		Namespace: sub.Namespace,
		ID:        sub.ID,
	}, currentTime)
	if err != nil {
		return nil, err
	}

	prices, err := q.priceConnector.GetForSubscription(ctx, models.NamespacedID{
		Namespace: sub.Namespace,
		ID:        sub.ID,
	})

	view, err := NewSubscriptionView(sub, spec, ents, prices)

	return view, err
}
