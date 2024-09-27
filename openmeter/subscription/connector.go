package subscription

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/subscription/override"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	modelref "github.com/openmeterio/openmeter/pkg/models/ref"
)

type NewSubscriptionRequest struct {
	Namespace  string
	ActiveFrom time.Time
	CustomerID string

	// Reference to the Plan that the subscription is based on.
	TemplatingPlanRef modelref.VersionedKeyRef // TODO: maybe just use IDs instead of versioned keys?

	// Phase overrides are applied to the Plan's phases in order.
	// These overrides are divergences from what's defined in the Plan that the user can make
	// when starting a subscription. There are two main use csases for this:
	//
	// 1. To edit the contents of the plan, i.e. manually override some aspects (like price)
	// 2. To fill in missing information that the plan doesn't have (like mandated custom pricing for enterprise customers)
	PhaseOverrides [][]override.Override[RateCard]
}

type OverrideSubscriptionPhaseRequest struct {
	rateCardOverrides []override.Override[RateCard]
	at                time.Time
	// PhaseId is required instead of index as the number of phases in the current subscription changes with each edit
	phaseId string
}

type Connector interface {
	// EndAt ends a subscription effective at the provided time.
	EndAt(ctx context.Context, subscriptionID string, at time.Time) (Subscription, error)

	// StartNew attempts to start a new subscription for a customer effective at the provided time.
	StartNew(ctx context.Context, customerID string, req NewSubscriptionRequest) (Subscription, error)
}

type connector struct {
	customerSubscriptionRepo    CustomerSubscriptionRepo
	subscriptionRepo            SubscriptionRepo
	subscriptionEntitlementRepo SubscriptionEntitlementRepo

	lifecycleManager LifecycleManager

	planAdapter          PlanAdapter
	billingAdapter       BillingAdapter
	entitlementConnector entitlement.Connector
	customerService      customer.Service
	transactionManager   transaction.Creator
}

var _ Connector = (*connector)(nil)

// EndAt ends a subscription effective at the provided time.
func (c *connector) EndAt(ctx context.Context, subscriptionID string, at time.Time) (Subscription, error) {
	sub, err := c.subscriptionRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return Subscription{}, err
	}

	// Lets validate that the request makes sense

	if sub.ActiveTo != nil {
		return Subscription{}, &models.GenericUserError{Message: "Subscription is already ended at a different time."}
	}

	if at.Before(sub.ActiveFrom) {
		return Subscription{}, &models.GenericUserError{Message: "End time is before start time."}
	}

	// We need to validate that the change doesnt affect already closed invoices
	cus, err := c.customerService.GetCustomer(ctx, customer.GetCustomerInput{
		ID:        sub.CustomerId,
		Namespace: sub.Namespace,
	})
	if err != nil {
		return Subscription{}, fmt.Errorf("failed to get customer: %w", err)
	}

	lastInvoicedAt, err := c.billingAdapter.LastInvoicedAt(ctx, cus.ID)
	if err != nil {
		return Subscription{}, fmt.Errorf("failed to get last invoiced at for customer %v: %w", cus.ID, err)
	}

	if !lastInvoicedAt.Before(at) {
		return Subscription{}, &models.GenericUserError{Message: "End time is after last invoiced time."}
	}

	return transaction.Run(ctx, c.transactionManager, func(ctx context.Context) (Subscription, error) {
		// Q: Do we invoice first or do we close the subscription first?
		// If we close the subscripiton first, the invoice sees a closed subscription it has to invoice.
		// If we invoice first, the subscription can still be used through invoicing which might result in data drift.
		// For that, we'll do the former.

		sub, err := c.subscriptionRepo.UpdateCadence(ctx, subscriptionID, models.CadencedModel{
			ActiveFrom: sub.ActiveFrom,
			ActiveTo:   &at,
		})
		if err != nil {
			return Subscription{}, fmt.Errorf("failed to end subscription: %w", err)
		}

		err = c.billingAdapter.TriggerInvoicing(ctx, cus.ID, sub.ID)
		return sub, err
	})
}

// StartNew attempts to start a new subscription for a customer based on the provided templating plan.
func (c *connector) StartNew(ctx context.Context, customerID string, req NewSubscriptionRequest) (Subscription, error) {
	prevCustomerSubs, err := c.customerSubscriptionRepo.GetAll(ctx, customerID, CustomerSubscriptionRepoParams{})
	if err != nil {
		return Subscription{}, err
	}

	templatingPlan, err := c.planAdapter.GetVersion(ctx, req.TemplatingPlanRef)
	if err != nil {
		return Subscription{}, err
	}

	// We need to validate that the new subscription meets lifecycle rules
	err = c.lifecycleManager.CanStartNew(ctx, customerID, prevCustomerSubs, templatingPlan)
	if err != nil {
		return Subscription{}, err
	}

	return transaction.Run(ctx, c.transactionManager, func(ctx context.Context) (Subscription, error) {
		// Fetch the Plan contents and apply overrides
		phases, err := c.planAdapter.GetPhases(ctx, modelref.VersionedKeyRef{
			Key:     templatingPlan.Key,
			Version: templatingPlan.Version,
		})
		if err != nil {
			return Subscription{}, err
		}

		if len(phases) != len(req.PhaseOverrides) {
			return Subscription{}, &models.GenericUserError{Message: "PhaseOverrides must have the same length as the Plan's phases"}
		}

		// Lets create the subscription
		sub, err := c.subscriptionRepo.Create(ctx, SubscriptionCreateInput{
			NamespacedModel:   models.NamespacedModel{Namespace: req.Namespace},
			CadencedModel:     models.CadencedModel{ActiveFrom: req.ActiveFrom},
			TemplatingPlanRef: req.TemplatingPlanRef,
		})
		if err != nil {
			return Subscription{}, err
		}

		startOfPhase := req.ActiveFrom

		for phaseIdx, phase := range phases {
			// Lets find any overrides for ratecards in this phase
			overrides := req.PhaseOverrides[phaseIdx]

			rateCards := override.ApplyOverrides(phase.RateCards(), overrides)
			// TODO: persist stuff
			for _, rateCard := range rateCards {
				err := c.createRateCardEntitlement(ctx, rateCard)
				if err != nil {
					return Subscription{}, fmt.Errorf("failed to insert phase: %w", err)
				}
			}

			startOfPhase = startOfPhase.Add(phase.Duration())
		}

		// We initiate invoicing for the new subscription
		err = c.billingAdapter.StartNewInvoice(ctx, customerID, sub.ID)

		return sub, err
	})
}
