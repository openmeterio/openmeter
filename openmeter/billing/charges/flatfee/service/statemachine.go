package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	flatfeerealizations "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee/service/realizations"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	chargestatemachine "github.com/openmeterio/openmeter/openmeter/billing/charges/statemachine"
	"github.com/openmeterio/openmeter/pkg/clock"
)

type stateMachine struct {
	*chargestatemachine.Machine[flatfee.Charge, flatfee.ChargeBase, flatfee.Status]

	Adapter      flatfee.Adapter
	Realizations *flatfeerealizations.Service
	Service      *service

	CreditNotesSupported bool
}

type StateMachine = chargestatemachine.StateMachine[flatfee.Charge]

type StateMachineConfig struct {
	Charge flatfee.Charge

	Adapter      flatfee.Adapter
	Realizations *flatfeerealizations.Service
	Service      *service

	CreditNotesSupported bool
}

func (c StateMachineConfig) Validate() error {
	var errs []error

	if err := c.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if c.Adapter == nil {
		errs = append(errs, errors.New("adapter is required"))
	}

	if c.Realizations == nil {
		errs = append(errs, errors.New("realizations service is required"))
	}

	if c.Service == nil {
		errs = append(errs, errors.New("service is required"))
	}

	return errors.Join(errs...)
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

	machine, err := chargestatemachine.New(chargestatemachine.Config[flatfee.Charge, flatfee.ChargeBase, flatfee.Status]{
		Charge: config.Charge,
		Persistence: chargestatemachine.Persistence[flatfee.Charge, flatfee.ChargeBase]{
			UpdateBase: func(ctx context.Context, base flatfee.ChargeBase) (flatfee.ChargeBase, error) {
				return out.Adapter.UpdateCharge(ctx, base)
			},
			Refetch: func(ctx context.Context, chargeID meta.ChargeID) (flatfee.Charge, error) {
				return out.Adapter.GetByID(ctx, flatfee.GetByIDInput{
					ChargeID: chargeID,
					Expands:  meta.Expands{meta.ExpandRealizations},
				})
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("new machine: %w", err)
	}

	out.Machine = machine

	return out, nil
}

func (s *stateMachine) IsInsideServicePeriod() bool {
	return !clock.Now().Before(s.Charge.Intent.ServicePeriod.From)
}

func (s *stateMachine) AdvanceAfterServicePeriodFrom(ctx context.Context) error {
	s.Charge.State.AdvanceAfter = lo.ToPtr(meta.NormalizeTimestamp(s.Charge.Intent.ServicePeriod.From))
	return nil
}

func (s *stateMachine) AdvanceAfterServicePeriodTo(ctx context.Context) error {
	s.Charge.State.AdvanceAfter = lo.ToPtr(meta.NormalizeTimestamp(s.Charge.Intent.ServicePeriod.To))
	return nil
}

func (s *stateMachine) ClearAdvanceAfter(ctx context.Context) error {
	s.Charge.State.AdvanceAfter = nil
	return nil
}
