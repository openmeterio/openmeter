package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	"github.com/openmeterio/openmeter/openmeter/subscription/patch"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *service) CreateFromPlan(ctx context.Context, inp subscriptionworkflow.CreateSubscriptionWorkflowInput, plan subscription.Plan) (subscription.SubscriptionView, error) {
	var def subscription.SubscriptionView

	// Let's find the customer
	cust, err := s.CustomerService.GetCustomer(ctx, customer.GetCustomerInput{
		CustomerID: &customer.CustomerID{
			Namespace: inp.Namespace,
			ID:        inp.CustomerID,
		},
	})
	if err != nil {
		return def, fmt.Errorf("failed to fetch customer: %w", err)
	}

	if cust == nil {
		return def, fmt.Errorf("unexpected nil customer")
	}

	if err := inp.Timing.ValidateForAction(subscription.SubscriptionActionCreate, nil); err != nil {
		return def, fmt.Errorf("invalid timing: %w", err)
	}

	activeFrom, err := inp.Timing.Resolve()
	if err != nil {
		return def, fmt.Errorf("failed to resolve active from: %w", err)
	}

	// Let's create the new Spec
	spec, err := subscription.NewSpecFromPlan(plan, subscription.CreateSubscriptionCustomerInput{
		CustomerId:    cust.ID,
		Currency:      plan.Currency(),
		ActiveFrom:    activeFrom,
		MetadataModel: inp.MetadataModel,
		Name:          lo.CoalesceOrEmpty(inp.Name, plan.GetName()),
		Description:   inp.Description,
	})

	if err := subscriptionworkflow.MapSubscriptionErrors(err); err != nil {
		return def, fmt.Errorf("failed to create spec from plan: %w", err)
	}

	// Finally, let's create the subscription
	return transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (subscription.SubscriptionView, error) {
		sub, err := s.Service.Create(ctx, inp.Namespace, spec)
		if err != nil {
			return def, fmt.Errorf("failed to create subscription: %w", err)
		}

		return s.Service.GetView(ctx, sub.NamespacedID)
	})
}

func (s *service) EditRunning(ctx context.Context, subscriptionID models.NamespacedID, customizations []subscription.Patch, timing subscription.Timing) (subscription.SubscriptionView, error) {
	// First, let's fetch the current state of the Subscription
	curr, err := s.Service.GetView(ctx, subscriptionID)
	if err != nil {
		return subscription.SubscriptionView{}, fmt.Errorf("failed to fetch subscription: %w", err)
	}

	adds, err := s.AddonService.List(ctx, subscriptionID.Namespace, subscriptionaddon.ListSubscriptionAddonsInput{
		SubscriptionID: subscriptionID.ID,
	})
	if err != nil {
		return subscription.SubscriptionView{}, fmt.Errorf("failed to list addons: %w", err)
	}

	if hasAddons(curr, adds.Items) {
		return subscription.SubscriptionView{}, models.NewGenericForbiddenError(fmt.Errorf("subscription with addons cannot be edited"))
	}

	// Let's set the owner subsystem
	// TODO: let's refactor, its a bit ad-hoc
	customizations = lo.Map(customizations, func(p subscription.Patch, _ int) subscription.Patch {
		if ap, ok := p.(patch.PatchAddItem); ok {
			if ap.CreateInput.CreateSubscriptionItemInput.Annotations == nil {
				ap.CreateInput.CreateSubscriptionItemInput.Annotations = models.Annotations{}
			}
			subscription.AnnotationParser.AddOwnerSubSystem(ap.CreateInput.CreateSubscriptionItemInput.Annotations, subscription.OwnerSubscriptionSubSystem)

			subscriptionworkflow.AnnotationParser.SetUniquePatchID(ap.CreateInput.CreateSubscriptionItemInput.Annotations)

			return ap
		}

		if ap, ok := p.(*patch.PatchAddItem); ok {
			if ap.CreateInput.CreateSubscriptionItemInput.Annotations == nil {
				ap.CreateInput.CreateSubscriptionItemInput.Annotations = models.Annotations{}
			}
			subscription.AnnotationParser.AddOwnerSubSystem(ap.CreateInput.CreateSubscriptionItemInput.Annotations, subscription.OwnerSubscriptionSubSystem)

			subscriptionworkflow.AnnotationParser.SetUniquePatchID(ap.CreateInput.CreateSubscriptionItemInput.Annotations)

			return ap
		}

		return p
	})

	// Let's validate the patches
	for i, p := range customizations {
		if err := p.Validate(); err != nil {
			return subscription.SubscriptionView{}, models.NewGenericValidationError(fmt.Errorf("invalid patch at index %d: %s", i, err.Error()))
		}
	}

	// Let's try to decode when the subscription should be patched
	if err := timing.ValidateForAction(subscription.SubscriptionActionUpdate, &curr); err != nil {
		return subscription.SubscriptionView{}, models.NewGenericValidationError(fmt.Errorf("invalid timing: %w", err))
	}

	editTime, err := timing.ResolveForSpec(curr.Spec)
	if err != nil {
		return subscription.SubscriptionView{}, fmt.Errorf("failed to resolve timing: %w", err)
	}

	// Let's apply the customizations
	spec := curr.AsSpec()

	err = spec.ApplyMany(lo.Map(customizations, subscription.ToApplies), subscription.ApplyContext{
		CurrentTime: editTime,
	})
	if err := subscriptionworkflow.MapSubscriptionErrors(err); err != nil {
		return subscription.SubscriptionView{}, fmt.Errorf("failed to apply customizations: %w", err)
	}

	// Finally, let's update the subscription
	return transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (subscription.SubscriptionView, error) {
		sub, err := s.Service.Update(ctx, subscriptionID, spec)
		if err != nil {
			return subscription.SubscriptionView{}, fmt.Errorf("failed to update subscription: %w", err)
		}

		return s.Service.GetView(ctx, sub.NamespacedID)
	})
}

func (s *service) ChangeToPlan(ctx context.Context, subscriptionID models.NamespacedID, inp subscriptionworkflow.ChangeSubscriptionWorkflowInput, plan subscription.Plan) (subscription.Subscription, subscription.SubscriptionView, error) {
	// typing helper
	type res struct {
		curr subscription.Subscription
		new  subscription.SubscriptionView
	}

	// Changing the plan means canceling the current subscription and creating a new one with the provided timestamp
	r, err := transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (res, error) {
		// Second, let's try to cancel the current subscription
		curr, err := s.Service.Cancel(ctx, subscriptionID, inp.Timing)
		if err != nil {
			return res{}, fmt.Errorf("failed to end current subscription: %w", err)
		}

		// Let's create a new timing with the exact value as the create step might not be able resolve it for itself
		verbatumTiming := subscription.Timing{
			Custom: curr.ActiveTo, // We have to make sure we resolve to the exact same timestamp
		}

		inp.Timing = verbatumTiming

		// Third, let's create a new subscription with the new plan
		new, err := s.CreateFromPlan(ctx, subscriptionworkflow.CreateSubscriptionWorkflowInput{
			ChangeSubscriptionWorkflowInput: inp,
			Namespace:                       curr.Namespace,
			CustomerID:                      curr.CustomerId,
		}, plan)
		if err != nil {
			return res{}, fmt.Errorf("failed to create new subscription: %w", err)
		}

		// Let's just return after a great success
		return res{curr, new}, nil
	})

	return r.curr, r.new, err
}

func (s *service) Restore(ctx context.Context, subscriptionID models.NamespacedID) (subscription.Subscription, error) {
	now := clock.Now()

	// Let's fetch the sub
	sub, err := s.Service.GetView(ctx, subscriptionID)
	if err != nil {
		return subscription.Subscription{}, fmt.Errorf("failed to fetch subscription: %w", err)
	}

	// Let's get all subs scheduled afterward
	scheduled, err := s.Service.GetAllForCustomerSince(ctx, models.NamespacedID{
		Namespace: sub.Subscription.Namespace,
		ID:        sub.Subscription.CustomerId,
	}, now)
	if err != nil {
		return subscription.Subscription{}, fmt.Errorf("failed to fetch scheduled subscriptions: %w", err)
	}

	// Let's filter out the current sub if present
	scheduled = lo.Filter(scheduled, func(s subscription.Subscription, _ int) bool {
		return s.NamespacedID != subscriptionID
	})

	return transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (subscription.Subscription, error) {
		// Let's delete all scheduled subs
		for _, sch := range scheduled {
			if err := s.Service.Delete(ctx, sch.NamespacedID); err != nil {
				return subscription.Subscription{}, fmt.Errorf("failed to delete scheduled subscription: %w", err)
			}
		}

		// Let's continue the current sub
		return s.Service.Continue(ctx, subscriptionID)
	})
}
