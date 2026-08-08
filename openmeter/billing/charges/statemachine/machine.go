package statemachine

import (
	"context"
	"errors"
	"fmt"

	"github.com/qmuntal/stateless"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/pkg/models"
)

var (
	ErrUnsupportedOperation    = models.NewGenericPreConditionFailedError(fmt.Errorf("unsupported operation"))
	ErrUnhandledInvoicePatches = errors.New("unhandled invoice patches")
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
	// AdvanceUntilInvoicePatchesOrStable fires lifecycle continuation
	// transitions until the machine emits invoice patches or has no next
	// transition. Returned patches are removed from the machine; the caller must
	// settle them before advancing a freshly loaded charge again.
	AdvanceUntilInvoicePatchesOrStable(ctx context.Context) (invoiceupdater.Patches, error)
	// AdvanceUntilStable fires lifecycle continuation transitions until the
	// machine has no next transition. It fails with ErrUnhandledInvoicePatches
	// when a transition emits invoice patches instead of advancing past them.
	AdvanceUntilStable(ctx context.Context) error
	// CanFire reports whether the trigger is permitted from the current charge
	// state without mutating or persisting the charge.
	CanFire(ctx context.Context, trigger meta.Trigger) (bool, error)
	// FireAndAdvanceUntilInvoicePatchesOrStable validates and fires the explicit
	// trigger, persists the resulting charge state, then fires continuation
	// transitions until invoice patches are emitted or the machine has no next
	// transition. Returned patches are removed from the machine; the caller must
	// settle them before advancing a freshly loaded charge again.
	FireAndAdvanceUntilInvoicePatchesOrStable(ctx context.Context, trigger meta.Trigger, args ...models.Validator) (invoiceupdater.Patches, error)
	// FireAndAdvanceUntilStable validates and fires the explicit trigger, then
	// advances until the machine has no next transition. It fails with
	// ErrUnhandledInvoicePatches rather than advancing past emitted patches.
	FireAndAdvanceUntilStable(ctx context.Context, trigger meta.Trigger, args ...models.Validator) error
	// GetCharge returns the machine's current in-memory charge, including state
	// and base values returned by persistence during completed transitions.
	GetCharge() CHARGE
	// RefetchCharge replaces the current in-memory charge with its latest
	// persisted representation.
	RefetchCharge(ctx context.Context) error
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
	Charge         CHARGE
	stateMachine   *stateless.StateMachine
	config         Config[CHARGE, BASE, STATUS]
	invoicePatches invoiceupdater.Patches
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

func (m *Machine[CHARGE, BASE, STATUS]) drainInvoicePatches() invoiceupdater.Patches {
	patches := m.invoicePatches
	m.invoicePatches = nil
	return patches
}

func (m *Machine[CHARGE, BASE, STATUS]) AddInvoicePatch(patches ...invoiceupdater.Patch) {
	m.invoicePatches = append(m.invoicePatches, patches...)
}

func (m *Machine[CHARGE, BASE, STATUS]) fireAndActivate(ctx context.Context, trigger meta.Trigger, args ...models.Validator) error {
	if len(m.invoicePatches) > 0 {
		return fmt.Errorf("%w: cannot fire trigger %v with %d pending invoice patches", ErrUnhandledInvoicePatches, trigger, len(m.invoicePatches))
	}

	var validationErrors []error
	if trigger == nil {
		validationErrors = append(validationErrors, errors.New("trigger is required"))
	}

	validationErrors = append(validationErrors, lo.Map(args, func(argument models.Validator, index int) error {
		if argument == nil {
			return fmt.Errorf("arguments[%d]: argument is required", index)
		}

		if err := argument.Validate(); err != nil {
			return fmt.Errorf("arguments[%d]: %w", index, err)
		}

		return nil
	})...)

	if err := models.NewNillableGenericValidationError(errors.Join(validationErrors...)); err != nil {
		return fmt.Errorf("trigger %v input: %w", trigger, err)
	}

	fireArgs := lo.Map(args, func(argument models.Validator, _ int) any {
		return argument
	})

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

	if err := m.stateMachine.FireCtx(ctx, trigger, fireArgs...); err != nil {
		return err
	}

	if err := m.stateMachine.ActivateCtx(ctx); err != nil {
		return err
	}

	updatedBase, err := m.config.Persistence.UpdateBase(ctx, m.Charge.GetBase())
	if err != nil {
		return fmt.Errorf("persist charge: %w", err)
	}

	m.Charge = m.Charge.WithBase(updatedBase)

	return nil
}

// FireAndAdvanceUntilInvoicePatchesOrStable applies an explicit lifecycle
// trigger and continues through next transitions until invoice patches are
// emitted or the machine becomes stable. Returned patches are removed from the
// machine and must be settled before the charge is advanced again.
func (m *Machine[CHARGE, BASE, STATUS]) FireAndAdvanceUntilInvoicePatchesOrStable(ctx context.Context, trigger meta.Trigger, args ...models.Validator) (invoiceupdater.Patches, error) {
	if err := m.fireAndActivate(ctx, trigger, args...); err != nil {
		return nil, fmt.Errorf("fire trigger %v: %w", trigger, err)
	}

	patches, err := m.advanceUntilInvoicePatchesOrStable(ctx)
	if err != nil {
		return nil, fmt.Errorf("advance after trigger %v: %w", trigger, err)
	}

	return patches, nil
}

// AdvanceUntilInvoicePatchesOrStable continues through next transitions until
// invoice patches are emitted or the machine becomes stable. Returned patches
// are removed from the machine and must be settled before the charge is
// advanced again.
func (m *Machine[CHARGE, BASE, STATUS]) AdvanceUntilInvoicePatchesOrStable(ctx context.Context) (invoiceupdater.Patches, error) {
	return m.advanceUntilInvoicePatchesOrStable(ctx)
}

func (m *Machine[CHARGE, BASE, STATUS]) advanceUntilInvoicePatchesOrStable(ctx context.Context) (invoiceupdater.Patches, error) {
	for {
		if len(m.invoicePatches) > 0 {
			return m.drainInvoicePatches(), nil
		}

		canFire, err := m.CanFire(ctx, meta.TriggerNext)
		if err != nil {
			return nil, err
		}
		if !canFire {
			return nil, nil
		}

		currentStatus := m.Charge.GetStatus()
		if err := m.fireAndActivate(ctx, meta.TriggerNext); err != nil {
			return nil, fmt.Errorf("cannot transition to the next status [current_status=%s]: %w", currentStatus, err)
		}
	}
}

// FireAndAdvanceUntilStable applies an explicit lifecycle trigger and advances
// until the machine is stable. Invoice patches are rejected because callers
// that own them must use FireAndAdvanceUntilInvoicePatchesOrStable.
func (m *Machine[CHARGE, BASE, STATUS]) FireAndAdvanceUntilStable(ctx context.Context, trigger meta.Trigger, args ...models.Validator) error {
	patches, err := m.FireAndAdvanceUntilInvoicePatchesOrStable(ctx, trigger, args...)
	if err != nil {
		return err
	}
	if len(patches) > 0 {
		return fmt.Errorf("%w: trigger %v produced %d invoice patches", ErrUnhandledInvoicePatches, trigger, len(patches))
	}

	return nil
}

// AdvanceUntilStable advances until the machine is stable. Invoice patches are
// rejected because callers that own them must use
// AdvanceUntilInvoicePatchesOrStable.
func (m *Machine[CHARGE, BASE, STATUS]) AdvanceUntilStable(ctx context.Context) error {
	patches, err := m.advanceUntilInvoicePatchesOrStable(ctx)
	if err != nil {
		return err
	}
	if len(patches) > 0 {
		return fmt.Errorf("%w: transition produced %d invoice patches", ErrUnhandledInvoicePatches, len(patches))
	}

	return nil
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
