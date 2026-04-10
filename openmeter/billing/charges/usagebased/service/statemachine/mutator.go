package statemachine

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/statemachine"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/qmuntal/stateless"
)

var _ statemachine.StateMutator[usagebased.Charge] = (*StateMutator)(nil)

type StateMutator struct {
	adapter usagebased.Adapter
}

func (s StateMutator) GetState(_ context.Context, charge usagebased.Charge) (stateless.State, error) {
	return charge.Status, nil
}

func (s StateMutator) SetState(_ context.Context, charge *usagebased.Charge, newState stateless.State) error {
	newStatus := newState.(usagebased.Status)
	if err := newStatus.Validate(); err != nil {
		return fmt.Errorf("invalid status: %w", err)
	}

	charge.Status = newStatus
	return nil
}

func (s StateMutator) PersistChargeBase(ctx context.Context, charge usagebased.Charge) (usagebased.Charge, error) {
	updatedChargeBase, err := s.adapter.UpdateCharge(ctx, charge.ChargeBase)
	if err != nil {
		return usagebased.Charge{}, fmt.Errorf("persist charge base: %w", err)
	}

	charge.ChargeBase = updatedChargeBase
	return charge, nil
}

func (s StateMutator) SetOrClearAdvanceAfter(charge usagebased.Charge, advanceAfter *time.Time) usagebased.Charge {
	charge.State.AdvanceAfter = advanceAfter
	return charge
}
