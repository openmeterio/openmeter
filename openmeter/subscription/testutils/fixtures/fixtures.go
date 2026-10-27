// Package fixtures creates persisted subscription scenarios using the existing
// test services. Each scenario caches its defaults; Create methods make fresh rows.
package fixtures

import (
	"fmt"
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/do/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Scenario owns lazy defaults in the existing harness's ExampleNamespace. It
// does not own the harness or isolate database state. Returned resources are
// creation snapshots; reload through the services when asserting current state.
type Scenario struct {
	t        *testing.T
	deps     subscriptiontestutils.SubscriptionDependencies
	injector do.Injector
}

func New(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) *Scenario {
	t.Helper()

	s := &Scenario{t: t, deps: deps, injector: do.New()}
	do.Provide(s.injector, func(do.Injector) (*customer.Customer, error) {
		return s.createCustomer(customer.CustomerMutate{})
	})
	do.Provide(s.injector, func(do.Injector) (*feature.Feature, error) {
		f, err := deps.FeatureConnector.CreateFeature(t.Context(), feature.CreateFeatureInputs{
			Namespace: subscriptiontestutils.ExampleNamespace,
			Key:       "fixture-feature-" + ulid.Make().String(),
			Name:      "Fixture feature",
		})
		if err != nil {
			return nil, fmt.Errorf("create default feature: %w", err)
		}

		return &f, nil
	})
	do.Provide(s.injector, func(i do.Injector) (subscription.Plan, error) {
		f, err := do.Invoke[*feature.Feature](i)
		if err != nil {
			return nil, fmt.Errorf("resolve default plan feature: %w", err)
		}

		input := subscriptiontestutils.BuildTestPlanInput(t).AddPhase(nil, &productcatalog.FlatFeeRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Key:                 f.Key,
				Name:                "Fixture access",
				Feature:             productcatalog.NewFeatureReference(&f.ID, &f.Key),
				EntitlementTemplate: productcatalog.NewEntitlementTemplateFrom(productcatalog.BooleanEntitlementTemplate{}),
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromInt(100),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
			},
			BillingCadence: lo.ToPtr(subscriptiontestutils.ISOMonth),
		}).Build()
		input.Plan.Key = "fixture-plan-" + ulid.Make().String()

		return s.createPlan(input)
	})

	return s
}

func (s *Scenario) Customer() *customer.Customer {
	s.t.Helper()
	c, err := do.Invoke[*customer.Customer](s.injector)
	require.NoError(s.t, err, "resolve default customer")

	return c
}

func (s *Scenario) Feature() *feature.Feature {
	s.t.Helper()
	f, err := do.Invoke[*feature.Feature](s.injector)
	require.NoError(s.t, err, "resolve default feature")

	return f
}

func (s *Scenario) Plan() subscription.Plan {
	s.t.Helper()
	p, err := do.Invoke[subscription.Plan](s.injector)
	require.NoError(s.t, err, "resolve default plan")

	return p
}

// CreateCustomer fills omitted identity and currency fields, preserving supplied
// usage attribution so callers can share or vary their own prerequisites.
func (s *Scenario) CreateCustomer(input customer.CustomerMutate) *customer.Customer {
	s.t.Helper()
	c, err := s.createCustomer(input)
	require.NoError(s.t, err, "create customer")

	return c
}

func (s *Scenario) createCustomer(input customer.CustomerMutate) (*customer.Customer, error) {
	key := "fixture-customer-" + ulid.Make().String()
	if input.Name == "" {
		input.Name = key
	}

	if input.Key == nil {
		input.Key = &key
	}

	if input.Currency == nil {
		input.Currency = lo.ToPtr(currencyx.Code("USD"))
	}

	if input.UsageAttribution == nil {
		input.UsageAttribution = &customer.CustomerUsageAttribution{SubjectKeys: []string{*input.Key}}
	}

	return s.deps.CustomerService.CreateCustomer(s.t.Context(), customer.CreateCustomerInput{
		Namespace:      subscriptiontestutils.ExampleNamespace,
		CustomerMutate: input,
	})
}

// CreatePlan persists and publishes the supplied plan from the current test
// clock onward. The caller creates any features or other prerequisites it uses.
func (s *Scenario) CreatePlan(input plan.CreatePlanInput) subscription.Plan {
	s.t.Helper()
	p, err := s.createPlan(input)
	require.NoError(s.t, err, "create and publish plan %q", input.Plan.Key)

	return p
}

func (s *Scenario) createPlan(input plan.CreatePlanInput) (subscription.Plan, error) {
	p, err := s.deps.PlanService.CreatePlan(s.t.Context(), input)
	if err != nil {
		return nil, fmt.Errorf("create plan %q: %w", input.Plan.Key, err)
	}

	p, err = s.deps.PlanService.PublishPlan(s.t.Context(), plan.PublishPlanInput{
		NamespacedID: p.NamespacedID,
		EffectivePeriod: productcatalog.EffectivePeriod{
			EffectiveFrom: lo.ToPtr(clock.Now()),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("publish plan %q: %w", input.Plan.Key, err)
	}

	return &plansubscription.Plan{Plan: p.AsProductCatalogPlan(), Ref: &p.NamespacedID}, nil
}

type CreateSubscriptionInput struct {
	Customer *customer.Customer
	Plan     subscription.Plan
	Change   subscriptionworkflow.ChangeSubscriptionWorkflowInput
}

func (i CreateSubscriptionInput) Validate() error {
	if i.Customer != nil && i.Customer.Namespace != subscriptiontestutils.ExampleNamespace {
		return models.NewGenericValidationError(fmt.Errorf("customer must belong to scenario namespace %q", subscriptiontestutils.ExampleNamespace))
	}

	return nil
}

// CreateSubscription always creates a fresh subscription, resolving only omitted
// dependencies. Lifecycle actions, including cancellation, remain explicit.
func (s *Scenario) CreateSubscription(input CreateSubscriptionInput) subscription.SubscriptionView {
	s.t.Helper()
	require.NoError(s.t, input.Validate(), "validate subscription fixture input")
	if input.Customer == nil {
		input.Customer = s.Customer()
	}

	if input.Plan == nil {
		input.Plan = s.Plan()
	}

	if input.Change.Timing == (subscription.Timing{}) {
		input.Change.Timing = subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)}
	}

	view, err := s.deps.WorkflowService.CreateFromPlan(s.t.Context(), subscriptionworkflow.CreateSubscriptionWorkflowInput{
		ChangeSubscriptionWorkflowInput: input.Change,
		Namespace:                       input.Customer.Namespace,
		CustomerID:                      input.Customer.ID,
		BillingAnchor:                   input.Change.BillingAnchor,
	}, input.Plan)
	require.NoError(s.t, err, "create subscription for customer %q from plan %q", input.Customer.ID, input.Plan.GetName())

	return view
}
