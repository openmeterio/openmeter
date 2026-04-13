package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/qmuntal/stateless"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	flatfeerealizations "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee/service/realizations"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/pkg/models"
)

type stateMachine struct {
	*stateless.StateMachine

	Charge flatfee.Charge

	Adapter      flatfee.Adapter
	Realizations *flatfeerealizations.Service
}

type StateMachine interface {
	AdvanceUntilStateStable(ctx context.Context) (*flatfee.Charge, error)
	CanFire(ctx context.Context, trigger meta.Trigger) (bool, error)
	FireAndActivate(ctx context.Context, trigger meta.Trigger, args ...any) error
	GetCharge() flatfee.Charge
}

type StateMachineConfig struct {
	Charge flatfee.Charge

	Adapter      flatfee.Adapter
	Realizations *flatfeerealizations.Service
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

	return errors.Join(errs...)
}

func newStateMachineBase(config StateMachineConfig) (*stateMachine, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	out := &stateMachine{
		Charge:       config.Charge,
		Adapter:      config.Adapter,
		Realizations: config.Realizations,
	}

	stateMachine := stateless.NewStateMachineWithExternalStorage(
		func(ctx context.Context) (stateless.State, error) {
			return out.Charge.Status, nil
		},
		func(ctx context.Context, state stateless.State) error {
			newStatus := state.(flatfee.Status)
			if err := newStatus.Validate(); err != nil {
				return fmt.Errorf("invalid status: %w", err)
			}

			out.Charge.Status = newStatus
			return nil
		},
		stateless.FiringImmediate,
	)

	out.StateMachine = stateMachine

	return out, nil
}

var ErrUnsupportedOperation = models.NewGenericPreConditionFailedError(fmt.Errorf("unsupported operation"))

func (s *stateMachine) CanFire(ctx context.Context, trigger meta.Trigger) (bool, error) {
	return s.StateMachine.CanFireCtx(ctx, trigger)
}

func (s *stateMachine) FireAndActivate(ctx context.Context, trigger meta.Trigger, args ...any) error {
	canFire, err := s.CanFire(ctx, trigger)
	if err != nil {
		return err
	}

	if !canFire {
		return fmt.Errorf("%w: %s [status=%s,id=%s]", ErrUnsupportedOperation, trigger, s.Charge.Status, s.Charge.ID)
	}

	if err := s.StateMachine.FireCtx(ctx, trigger, args...); err != nil {
		return err
	}

	return s.StateMachine.ActivateCtx(ctx)
}

func (s *stateMachine) AdvanceUntilStateStable(ctx context.Context) (*flatfee.Charge, error) {
	var advanced bool

	for {
		canFire, err := s.StateMachine.CanFireCtx(ctx, meta.TriggerNext)
		if err != nil {
			return nil, err
		}

		if !canFire {
			if !advanced {
				return nil, nil
			}

			charge := s.Charge
			return &charge, nil
		}

		if err := s.FireAndActivate(ctx, meta.TriggerNext); err != nil {
			return nil, fmt.Errorf("cannot transition to the next status [current_status=%s]: %w", s.Charge.Status, err)
		}

		updatedChargeBase, err := s.Adapter.UpdateCharge(ctx, s.Charge.ChargeBase)
		if err != nil {
			return nil, fmt.Errorf("persist charge: %w", err)
		}

		s.Charge.ChargeBase = updatedChargeBase

		advanced = true
	}
}

func (s *stateMachine) refetchCharge(ctx context.Context) error {
	charge, err := s.Adapter.GetByID(ctx, flatfee.GetByIDInput{
		ChargeID: s.Charge.GetChargeID(),
	})
	if err != nil {
		return fmt.Errorf("get charge: %w", err)
	}

	s.Charge = charge
	return nil
}

func (s *stateMachine) GetCharge() flatfee.Charge {
	return s.Charge
}
