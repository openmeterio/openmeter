package subscription

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	modelref "github.com/openmeterio/openmeter/pkg/models/ref"
	"github.com/samber/lo"
)

type NewSubscriptionRequest struct {
	Namespace         string
	ActiveFrom        time.Time
	CustomerID        string
	TemplatingPlanRef modelref.VersionedKeyRef // TODO: maybe just use IDs instead of versioned keys?

	// Phase overrides are applied to the Plan's phases in order
	PhaseOverrides [][]Override[RateCard]
}

type OverrideSubscriptionPhaseRequest struct {
	rateCardOverrides []Override[RateCard]
	at                time.Time
	// PhaseId is required instead of index as the number of phases in the current subscription changes with each edit
	phaseId string
}

type Connector interface {
	// EndAt ends a subscription effective at the provided time.
	EndAt(ctx context.Context, subscriptionID string, at time.Time) (Subscription, error)

	// StartNew attempts to start a new subscription for a customer effective at the provided time.
	StartNew(ctx context.Context, customerID string, req NewSubscriptionRequest) (Subscription, error)

	// OverridePhase overrides the rate cards for a phase in a subscription.
	OverridePhase(ctx context.Context, subscriptionID models.NamespacedID, req OverrideSubscriptionPhaseRequest) (SubscriptionPhase, error)
}

type connector struct {
	customerSubscriptionRepo CustomerSubscriptionRepo
	subscriptionRepo         SubscriptionRepo
	subscriptionPhaseRepo    SubscriptionPhaseRepo

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

	// FIXME: fix once https://github.com/openmeterio/openmeter/pull/1568 is merged
	subP, err := transaction.Run(ctx, c.transactionManager, func(ctx context.Context) (*Subscription, error) {
		// Q: Do we invoice first or do we close the subscription first?
		// If we close the subscripiton first, the invoice sees a closed subscription it has to invoice.
		// If we invoice first, the subscription can still be used through invoicing which might result in data drift.
		// For that, we'll do the former.

		sub, err := c.subscriptionRepo.UpdateCadence(ctx, subscriptionID, models.CadencedModel{
			ActiveFrom: sub.ActiveFrom,
			ActiveTo:   &at,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to end subscription: %w", err)
		}

		err = c.billingAdapter.TriggerInvoicing(ctx, cus.ID, sub.ID)
		return &sub, err
	})

	if err != nil || subP == nil {
		return Subscription{}, err
	}

	return *subP, nil
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

	// FIXME: fix once https://github.com/openmeterio/openmeter/pull/1568 is merged
	subP, err := transaction.Run(ctx, c.transactionManager, func(ctx context.Context) (*Subscription, error) {
		// Fetch the Plan contents and apply overrides
		phases, err := c.planAdapter.GetPhases(ctx, modelref.VersionedKeyRef{
			Key:     templatingPlan.Key,
			Version: templatingPlan.Version,
		})
		if err != nil {
			return nil, err
		}

		if len(phases) != len(req.PhaseOverrides) {
			return nil, &models.GenericUserError{Message: "PhaseOverrides must have the same length as the Plan's phases"}
		}

		// Lets create the subscription
		sub, err := c.subscriptionRepo.Create(ctx, SubscriptionCreateInput{
			NamespacedModel:   models.NamespacedModel{Namespace: req.Namespace},
			CadencedModel:     models.CadencedModel{ActiveFrom: req.ActiveFrom},
			TemplatingPlanRef: req.TemplatingPlanRef,
		})
		if err != nil {
			return nil, err
		}

		startOfPhase := req.ActiveFrom

		for phaseIdx, phase := range phases {
			// Lets find any overrides for ratecards in this phase
			overrides := req.PhaseOverrides[phaseIdx]

			rateCards := ApplyOverrides(phase.RateCards(), overrides)
			_, err := c.insertPhase(ctx, insertPhaseRequest{
				phaseInput: SubscriptionPhaseCreateInput{
					NamespacedModel: models.NamespacedModel{Namespace: req.Namespace},
					ActiveFrom:      startOfPhase,
					SubscriptionId:  sub.ID,
				},
				rateCards: rateCards,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to insert phase: %w", err)
			}

			startOfPhase = startOfPhase.Add(phase.Duration())
		}

		// We initiate invoicing for the new subscription
		err = c.billingAdapter.StartNewInvoice(ctx, customerID, sub.ID)

		return &sub, err
	})

	if err != nil || subP == nil {
		return Subscription{}, err
	}

	return *subP, nil
}

// OverridePhase overrides (among others) the rate cards for a phase in a subscription.
// This in turn creates a new phase lasting until the last scheduled phase after that with the new configuration.
func (c *connector) OverridePhase(ctx context.Context, subscriptionID models.NamespacedID, req OverrideSubscriptionPhaseRequest) (SubscriptionPhase, error) {
	sub, err := c.subscriptionRepo.GetByID(ctx, subscriptionID.ID)
	if err != nil {
		return SubscriptionPhase{}, err
	}

	phases, err := c.subscriptionPhaseRepo.GetForSub(ctx, subscriptionID.ID)
	if err != nil {
		return SubscriptionPhase{}, err
	}

	phase, phaseIdx, ok := lo.FindIndexOf(phases, func(phase SubscriptionPhase) bool {
		return phase.ID == req.phaseId
	})

	if !ok {
		return SubscriptionPhase{}, &models.GenericUserError{Message: fmt.Sprintf("Phase with id %s not found", req.phaseId)}
	}

	if !phase.ActiveFrom.Before(req.at) {
		return SubscriptionPhase{}, &models.GenericUserError{Message: "Override time must be after the phase start time"}
	}

	if phaseIdx < len(phases)-1 && !req.at.Before(phases[phaseIdx+1].ActiveFrom) {
		return SubscriptionPhase{}, &models.GenericUserError{Message: "Override time must be before the next phase start time"}
	}

	rateCards, err := c.subscriptionPhaseRepo.GetRateCards(ctx, phase.ID)
	if err != nil {
		return SubscriptionPhase{}, fmt.Errorf("failed to get rate cards for phase: %w", err)
	}

	// When overriding, 1st we have to close the old phase, close any downstream resources, and then create the new phase
	// FIXME: fix once https://github.com/openmeterio/openmeter/pull/1568 is merged
	subpP, err := transaction.Run(ctx, c.transactionManager, func(ctx context.Context) (*SubscriptionPhase, error) {
		err := c.subscriptionPhaseRepo.DeleteAt(ctx, phase.ID, req.at)
		if err != nil {
			return nil, fmt.Errorf("failed to delete phase: %w", err)
		}

		// TODO: close and migrate entitlements with values...

		// create new phase
		rateCards := ApplyOverrides(rateCards, req.rateCardOverrides)

		p, errr := c.insertPhase(ctx, insertPhaseRequest{
			phaseInput: SubscriptionPhaseCreateInput{
				NamespacedModel: models.NamespacedModel{Namespace: sub.Namespace},
				ActiveFrom:      req.at,
				SubscriptionId:  sub.ID,
			},
			rateCards: rateCards,
		})
		return &p, errr
	})

	if err != nil || subpP == nil {
		return SubscriptionPhase{}, err
	}

	return *subpP, nil
}

type insertPhaseRequest struct {
	phaseInput SubscriptionPhaseCreateInput
	rateCards  []RateCard
}

// insertPhase inserts a new phase into the subscription if it can.
func (c *connector) insertPhase(ctx context.Context, req insertPhaseRequest) (SubscriptionPhase, error) {
	subPhase, err := c.subscriptionPhaseRepo.Create(ctx, req.phaseInput)
	if err != nil {
		return SubscriptionPhase{}, err
	}

	for _, rateCard := range req.rateCards {
		// Lets create dependent resources
		err := c.createDependentsOfRateCard(ctx, rateCard)
		if err != nil {
			return SubscriptionPhase{}, fmt.Errorf("failed to create dependent resources for rate card: %w", err)
		}
	}

	return subPhase, nil
}
