package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	chargestatemachine "github.com/openmeterio/openmeter/openmeter/billing/charges/statemachine"
)

type stateMachine struct {
	*chargestatemachine.Machine[creditpurchase.Charge, creditpurchase.ChargeBase, creditpurchase.Status]

	Adapter creditpurchase.Adapter
	// Realizations *realizations.Service
	Service *service

	CreditNotesSupported bool
}

type StateMachine = chargestatemachine.StateMachine[creditpurchase.Charge]

type StateMachineConfig struct {
	Charge creditpurchase.Charge

	Adapter creditpurchase.Adapter
	// Realizations *realizations.Service
	Service *service

	CreditNotesSupported bool
}

func (c StateMachineConfig) Validate() error {
	if c.Charge.ID == "" {
		return fmt.Errorf("charge ID is required")
	}
	if c.Adapter == nil {
		return fmt.Errorf("adapter is required")
	}
	return nil
}

func newStateMachineBase(config StateMachineConfig) (*stateMachine, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	out := &stateMachine{
		Adapter:              config.Adapter,
		Service:              config.Service,
		CreditNotesSupported: config.CreditNotesSupported,
	}

	machine, err := chargestatemachine.New(chargestatemachine.Config[creditpurchase.Charge, creditpurchase.ChargeBase, creditpurchase.Status]{
		Charge: config.Charge,
		Persistence: chargestatemachine.Persistence[creditpurchase.Charge, creditpurchase.ChargeBase]{
			UpdateBase: func(ctx context.Context, base creditpurchase.ChargeBase) (creditpurchase.ChargeBase, error) {
				return out.Adapter.UpdateCharge(ctx, base)
			},
			Refetch: func(ctx context.Context, chargeID meta.ChargeID) (creditpurchase.Charge, error) {
				return out.Adapter.GetByID(ctx, creditpurchase.GetByIDInput{
					ChargeID: chargeID,
					Expands:  meta.Expands{meta.ExpandRealizations},
				})
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create state machine: %w", err)
	}

	out.Machine = machine

	return out, nil
}
