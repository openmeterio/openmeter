package subscription

import (
	"context"
	"fmt"

	"github.com/qmuntal/stateless"

	"github.com/openmeterio/openmeter/pkg/models"
)

type SubscriptionStatus string

const (
	// Active means the subscription is active and the customer is being billed
	SubscriptionStatusActive SubscriptionStatus = "active"
	// Canceled means the subscription has already been canceled but is still active
	SubscriptionStatusCanceled SubscriptionStatus = "canceled"
	// Inactive means the subscription is inactive (might have been previously active) and the customer is not being billed
	SubscriptionStatusInactive SubscriptionStatus = "inactive"
	// Scheduled means the subscription is scheduled to be active in the future
	SubscriptionStatusScheduled SubscriptionStatus = "scheduled"
)

func (s SubscriptionStatus) Validate() error {
	switch s {
	case SubscriptionStatusActive, SubscriptionStatusCanceled, SubscriptionStatusInactive, SubscriptionStatusScheduled:
		return nil
	default:
		return fmt.Errorf("invalid subscription status: %s", s)
	}
}

type SubscriptionAction string

const (
	SubscriptionActionCreate       SubscriptionAction = "create"
	SubscriptionActionUpdate       SubscriptionAction = "update"
	SubscriptionActionCancel       SubscriptionAction = "cancel"
	SubscriptionActionContinue     SubscriptionAction = "continue"
	SubscriptionActionDelete       SubscriptionAction = "delete"
	SubscriptionActionChangeAddons SubscriptionAction = "change_addons"
)

// SubscriptionStateMachine is a very simple state machine that determines what actions can be taken on a Subscription
type SubscriptionStateMachine struct {
	sm *stateless.StateMachine
}

func (sm SubscriptionStateMachine) CanTransitionOrErr(ctx context.Context, action SubscriptionAction) error {
	can, err := sm.sm.CanFireCtx(ctx, action)
	// If there was an error, let's just log it and return false
	if err != nil {
		return fmt.Errorf("failed to check if transition is possible: %w", err)
	}

	state, err := sm.sm.State(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current state: %w", err)
	}

	status, ok := state.(SubscriptionStatus)
	if !ok {
		return fmt.Errorf("failed to cast state to SubscriptionStatus, got %T %v", state, state)
	}

	if err := status.Validate(); err != nil {
		return fmt.Errorf("invalid state: %w", err)
	}

	if !can {
		return models.NewGenericForbiddenError(
			fmt.Errorf("transition %s in state %s not allowed", action, state),
		)
	}

	return nil
}

func NewStateMachine(status SubscriptionStatus) SubscriptionStateMachine {
	sm := stateless.NewStateMachine(status)

	sm.Configure(SubscriptionStatusInactive).
		Permit(SubscriptionActionCreate, SubscriptionStatusActive)

	sm.Configure(SubscriptionStatusActive).
		PermitReentry(SubscriptionActionUpdate).
		PermitReentry(SubscriptionActionChangeAddons).
		Permit(SubscriptionActionCancel, SubscriptionStatusCanceled)

	sm.Configure(SubscriptionStatusCanceled).
		Permit(SubscriptionActionContinue, SubscriptionStatusActive)

	sm.Configure(SubscriptionStatusScheduled).
		Permit(SubscriptionActionDelete, nil) // Delete deletes the state too

	return SubscriptionStateMachine{
		sm: sm,
	}
}
