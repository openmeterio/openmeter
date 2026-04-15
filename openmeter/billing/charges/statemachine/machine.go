package statemachine

import (
	"context"
	"errors"
	"fmt"

	"github.com/qmuntal/stateless"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Status interface {
	~string
	Validate() error
}

type ChargeLike[CHARGE any, BASE any, STATUS Status] interface {
	GetChargeID() meta.ChargeID
	GetStatus() STATUS
	WithStatus(STATUS) CHARGE
	GetBase() BASE
	WithBase(BASE) CHARGE
}

type Persistence[CHARGE any, BASE any] struct {
	UpdateBase func(ctx context.Context, base BASE) (BASE, error)
	Refetch    func(ctx context.Context, chargeID meta.ChargeID) (CHARGE, error)
}

type Config[CHARGE ChargeLike[CHARGE, BASE, STATUS], BASE any, STATUS Status] struct {
	Charge      CHARGE
	Persistence Persistence[CHARGE, BASE]
}

type StateMachine[CHARGE any] interface {
	AdvanceUntilStateStable(ctx context.Context) (*CHARGE, error)
	CanFire(ctx context.Context, trigger meta.Trigger) (bool, error)
	FireAndActivate(ctx context.Context, trigger meta.Trigger, args ...any) error
	GetCharge() CHARGE
}

func (c Config[CHARGE, BASE, STATUS]) Validate() error {
	var errs []error

	if c.Persistence.UpdateBase == nil {
		errs = append(errs, errors.New("persistence.update base is required"))
	}

	if c.Persistence.Refetch == nil {
		errs = append(errs, errors.New("persistence.refetch is required"))
	}

	return errors.Join(errs...)
}

type Machine[CHARGE ChargeLike[CHARGE, BASE, STATUS], BASE any, STATUS Status] struct {
	Charge       CHARGE
	stateMachine *stateless.StateMachine
	config       Config[CHARGE, BASE, STATUS]
}

func New[CHARGE ChargeLike[CHARGE, BASE, STATUS], BASE any, STATUS Status](config Config[CHARGE, BASE, STATUS]) (*Machine[CHARGE, BASE, STATUS], error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	out := &Machine[CHARGE, BASE, STATUS]{
		Charge: config.Charge,
		config: config,
	}

	out.stateMachine = stateless.NewStateMachineWithExternalStorage(
		func(ctx context.Context) (stateless.State, error) {
			return out.Charge.GetStatus(), nil
		},
		func(ctx context.Context, state stateless.State) error {
			newStatus := state.(STATUS)
			if err := newStatus.Validate(); err != nil {
				return fmt.Errorf("invalid status: %w", err)
			}

			out.Charge = out.Charge.WithStatus(newStatus)

			return nil
		},
		stateless.FiringImmediate,
	)

	return out, nil
}

func (m *Machine[CHARGE, BASE, STATUS]) Configure(state STATUS) *stateless.StateConfiguration {
	return m.stateMachine.Configure(state)
}

func (m *Machine[CHARGE, BASE, STATUS]) CanFire(ctx context.Context, trigger meta.Trigger) (bool, error) {
	return m.stateMachine.CanFireCtx(ctx, trigger)
}

func (m *Machine[CHARGE, BASE, STATUS]) GetCharge() CHARGE {
	return m.Charge
}

var ErrUnsupportedOperation = models.NewGenericPreConditionFailedError(fmt.Errorf("unsupported operation"))

func (m *Machine[CHARGE, BASE, STATUS]) FireAndActivate(ctx context.Context, trigger meta.Trigger, args ...any) error {
	canFire, err := m.CanFire(ctx, trigger)
	if err != nil {
		return err
	}

	if !canFire {
		return fmt.Errorf(
			"%w: %s [status=%s,id=%s]",
			ErrUnsupportedOperation,
			trigger,
			m.Charge.GetStatus(),
			m.Charge.GetChargeID().ID,
		)
	}

	if err := m.stateMachine.FireCtx(ctx, trigger, args...); err != nil {
		return err
	}

	return m.stateMachine.ActivateCtx(ctx)
}

func (m *Machine[CHARGE, BASE, STATUS]) AdvanceUntilStateStable(ctx context.Context) (*CHARGE, error) {
	var advanced bool

	for {
		canFire, err := m.CanFire(ctx, meta.TriggerNext)
		if err != nil {
			return nil, err
		}

		if !canFire {
			if !advanced {
				return nil, nil
			}

			charge := m.Charge
			return &charge, nil
		}

		currentStatus := m.Charge.GetStatus()

		if err := m.FireAndActivate(ctx, meta.TriggerNext); err != nil {
			return nil, fmt.Errorf("cannot transition to the next status [current_status=%s]: %w", currentStatus, err)
		}

		updatedBase, err := m.config.Persistence.UpdateBase(ctx, m.Charge.GetBase())
		if err != nil {
			return nil, fmt.Errorf("persist charge: %w", err)
		}

		m.Charge = m.Charge.WithBase(updatedBase)

		advanced = true
	}
}

func (m *Machine[CHARGE, BASE, STATUS]) RefetchCharge(ctx context.Context) error {
	chargeID := m.Charge.GetChargeID()

	charge, err := m.config.Persistence.Refetch(ctx, chargeID)
	if err != nil {
		return err
	}

	m.Charge = charge
	return nil
}
