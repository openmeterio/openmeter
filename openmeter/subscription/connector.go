package subscription

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/dummy"
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
	OverridePhase(ctx context.Context, subscriptionID models.NamespacedID, req OverrideSubscriptionPhaseRequest) (Subscription, error)
}

type connector struct {
	customerSubscriptionRepo CustomerSubscriptionRepo
	subscriptionRepo         SubscriptionRepo
	subscriptionPhaseRepo    SubscriptionPhaseRepo
	planAdapter              PlanAdapter

	lifecycleManager LifecycleManager

	entitlementConnector entitlement.Connector
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

	// Lets end the subscription
	return c.subscriptionRepo.UpdateCadence(ctx, subscriptionID, models.CadencedModel{
		ActiveFrom: sub.ActiveFrom,
		ActiveTo:   &at,
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

	return dummy.Transaction(ctx, func(ctx context.Context) (Subscription, error) {
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
			// Lets create the subscription phase
			_, err := c.subscriptionPhaseRepo.Create(ctx, SubscriptionPhaseCreateInput{
				NamespacedModel: models.NamespacedModel{Namespace: req.Namespace},
				SubscriptionId:  sub.ID,
				ActiveFrom:      startOfPhase,
			})
			if err != nil {
				return Subscription{}, fmt.Errorf("failed to create subscription phase #%d: %w", phaseIdx, err)
			}

			startOfPhase = startOfPhase.Add(phase.Duration())

			// Lets create dependent resources from the RateCards
			// Lets find any overrides for ratecards in this phase
			overrides := req.PhaseOverrides[phaseIdx]

			rateCards := ApplyOverrides(phase.RateCards(), overrides)

			for _, rateCard := range rateCards {
				// Lets create dependent resources
				err := c.createDependentsOfRateCard(ctx, rateCard)
				if err != nil {
					return Subscription{}, fmt.Errorf("failed to create dependent resources for rate card: %w", err)
				}
			}
		}

		return sub, nil
	})
}

// TODO: should we apply overrides to the original plan definition
// or should we apply them to the latest version of the sub phase?
//
// e.g. incremental diff or full diff
//
// # Currently its incremental
//
// TODO: should we alternatively make ratecards a CadencedModel? (instead of creating new phases...)
func (c *connector) OverridePhase(ctx context.Context, subscriptionID models.NamespacedID, req OverrideSubscriptionPhaseRequest) (Subscription, error) {
	sub, err := c.subscriptionRepo.GetByID(ctx, subscriptionID.ID)
	if err != nil {
		return Subscription{}, err
	}

	phases, err := c.subscriptionPhaseRepo.GetForSub(ctx, subscriptionID.ID)
	if err != nil {
		return Subscription{}, err
	}

	phase, phaseIdx, ok := lo.FindIndexOf(phases, func(phase SubscriptionPhase) bool {
		return phase.ID == req.phaseId
	})

	if !ok {
		return Subscription{}, &models.GenericUserError{Message: fmt.Sprintf("Phase with id %s not found", req.phaseId)}
	}

	if !phase.ActiveFrom.Before(req.at) {
		return Subscription{}, &models.GenericUserError{Message: "Override time must be after the phase start time"}
	}

	if phaseIdx < len(phases)-1 && !req.at.Before(phases[phaseIdx+1].ActiveFrom) {
		return Subscription{}, &models.GenericUserError{Message: "Override time must be before the next phase start time"}
	}

	rateCards, err := c.subscriptionPhaseRepo.GetRateCards(ctx, phase.ID)
	if err != nil {
		return Subscription{}, fmt.Errorf("failed to get rate cards for phase: %w", err)
	}

	// When overriding, 1st we have to close the old phase, close any downstream resources, and then create the new phase
	return dummy.Transaction(ctx, func(ctx context.Context) (Subscription, error) {
		err := c.subscriptionPhaseRepo.DeleteAt(ctx, phase.ID, req.at)
		if err != nil {
			return Subscription{}, fmt.Errorf("failed to delete phase: %w", err)
		}

		// TODO: close and migrate entitlements with values...

		// Create new phase
		_, err = c.subscriptionPhaseRepo.Create(ctx, SubscriptionPhaseCreateInput{
			NamespacedModel: models.NamespacedModel{Namespace: sub.Namespace},
			SubscriptionId:  sub.ID,
			ActiveFrom:      req.at,
		})
		if err != nil {
			return Subscription{}, fmt.Errorf("failed to create new subscription phase: %w", err)
		}

		rateCards := ApplyOverrides(rateCards, req.rateCardOverrides)

		for _, rateCard := range rateCards {
			// Lets create dependent resources
			err := c.createDependentsOfRateCard(ctx, rateCard)
			if err != nil {
				return Subscription{}, fmt.Errorf("failed to create dependent resources for rate card: %w", err)
			}
		}

		return sub, nil
	})
}
