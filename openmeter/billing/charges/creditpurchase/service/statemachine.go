package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	creditpurchaserealizations "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase/service/realizations"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	chargestatemachine "github.com/openmeterio/openmeter/openmeter/billing/charges/statemachine"
	"github.com/openmeterio/openmeter/pkg/models"
)

type stateMachine struct {
	*chargestatemachine.Machine[creditpurchase.Charge, creditpurchase.ChargeBase, creditpurchase.Status]

	Adapter      creditpurchase.Adapter
	Realizations *creditpurchaserealizations.Service
	Service      *service

	CreditNotesSupported bool
}

type StateMachine = chargestatemachine.StateMachine[creditpurchase.Charge]

type StateMachineConfig struct {
	Charge creditpurchase.Charge

	Adapter      creditpurchase.Adapter
	Realizations *creditpurchaserealizations.Service
	Service      *service

	CreditNotesSupported bool
}

func (c StateMachineConfig) Validate() error {
	var errs []error

	if c.Charge.ID == "" {
		errs = append(errs, errors.New("charge ID is required"))
	}

	if c.Adapter == nil {
		errs = append(errs, errors.New("adapter is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func newStateMachineBase(config StateMachineConfig) (*stateMachine, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	out := &stateMachine{
		Adapter:              config.Adapter,
		Realizations:         config.Realizations,
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

func (s *stateMachine) FireAndAdvanceUntilStateStable(ctx context.Context, trigger meta.Trigger) (creditpurchase.Charge, error) {
	if err := s.FireAndActivate(ctx, trigger); err != nil {
		return creditpurchase.Charge{}, err
	}

	advancedCharge, err := s.AdvanceUntilStateStable(ctx)
	if err != nil {
		return creditpurchase.Charge{}, err
	}

	if advancedCharge != nil {
		return *advancedCharge, nil
	}

	return s.GetCharge(), nil
}
