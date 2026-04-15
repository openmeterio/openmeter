package statemachine

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
)

type fakeStatus string

const (
	fakeStatusCreated fakeStatus = "created"
	fakeStatusActive  fakeStatus = "active"
	fakeStatusFinal   fakeStatus = "final"
)

func (s fakeStatus) Validate() error {
	switch s {
	case fakeStatusCreated, fakeStatusActive, fakeStatusFinal:
		return nil
	default:
		return fmt.Errorf("invalid fake status: %s", s)
	}
}

type fakeBase struct {
	Revision int
}

type fakeCharge struct {
	ChargeID meta.ChargeID
	Status   fakeStatus
	Base     fakeBase
	Marker   string
}

func (c fakeCharge) GetChargeID() meta.ChargeID {
	return c.ChargeID
}

func (c fakeCharge) GetStatus() fakeStatus {
	return c.Status
}

func (c fakeCharge) WithStatus(status fakeStatus) fakeCharge {
	c.Status = status
	return c
}

func (c fakeCharge) GetBase() fakeBase {
	return c.Base
}

func (c fakeCharge) WithBase(base fakeBase) fakeCharge {
	c.Base = base
	return c
}

func newFakeCharge(status fakeStatus) fakeCharge {
	return fakeCharge{
		ChargeID: meta.ChargeID{
			Namespace: "test-namespace",
			ID:        "charge-1",
		},
		Status: status,
		Base: fakeBase{
			Revision: 0,
		},
		Marker: "initial",
	}
}

func newTestMachine(
	t *testing.T,
	charge fakeCharge,
	updateBase func(ctx context.Context, base fakeBase) (fakeBase, error),
	refetch func(ctx context.Context, chargeID meta.ChargeID) (fakeCharge, error),
) *Machine[fakeCharge, fakeBase, fakeStatus] {
	t.Helper()

	machine, err := New(Config[fakeCharge, fakeBase, fakeStatus]{
		Charge: charge,
		Persistence: Persistence[fakeCharge, fakeBase]{
			UpdateBase: updateBase,
			Refetch:    refetch,
		},
	})
	require.NoError(t, err)

	return machine
}

func TestMachine_FireAndActivateUpdatesStatus(t *testing.T) {
	// Given:
	// a machine in created state with a next transition to active.
	// When:
	// FireAndActivate is called with the next trigger.
	// Then:
	// the machine updates the in-memory charge status to active.
	machine := newTestMachine(
		t,
		newFakeCharge(fakeStatusCreated),
		func(ctx context.Context, base fakeBase) (fakeBase, error) { return base, nil },
		func(ctx context.Context, chargeID meta.ChargeID) (fakeCharge, error) { return fakeCharge{}, nil },
	)

	machine.Configure(fakeStatusCreated).Permit(meta.TriggerNext, fakeStatusActive)

	err := machine.FireAndActivate(t.Context(), meta.TriggerNext)

	require.NoError(t, err)
	require.Equal(t, fakeStatusActive, machine.GetCharge().GetStatus())
}

func TestMachine_FireAndActivateReturnsUnsupportedOperationWhenTriggerCannotFire(t *testing.T) {
	// Given:
	// a machine in created state without a next transition.
	// When:
	// FireAndActivate is called with the next trigger.
	// Then:
	// the machine returns an unsupported-operation error that includes the trigger, status, and charge id.
	machine := newTestMachine(
		t,
		newFakeCharge(fakeStatusCreated),
		func(ctx context.Context, base fakeBase) (fakeBase, error) { return base, nil },
		func(ctx context.Context, chargeID meta.ChargeID) (fakeCharge, error) { return fakeCharge{}, nil },
	)

	err := machine.FireAndActivate(t.Context(), meta.TriggerNext)

	require.Error(t, err)
	require.ErrorIs(t, err, ErrUnsupportedOperation)
	require.ErrorContains(t, err, fmt.Sprint(meta.TriggerNext))
	require.ErrorContains(t, err, string(fakeStatusCreated))
	require.ErrorContains(t, err, "charge-1")
}

func TestMachine_AdvanceUntilStateStableReturnsNilWhenAlreadyStable(t *testing.T) {
	// Given:
	// a machine already in a stable state with no next transition.
	// When:
	// AdvanceUntilStateStable is called.
	// Then:
	// it returns nil and does not persist the base.
	var updateCalls int

	machine := newTestMachine(
		t,
		newFakeCharge(fakeStatusFinal),
		func(ctx context.Context, base fakeBase) (fakeBase, error) {
			updateCalls++
			return base, nil
		},
		func(ctx context.Context, chargeID meta.ChargeID) (fakeCharge, error) { return fakeCharge{}, nil },
	)

	charge, err := machine.AdvanceUntilStateStable(t.Context())

	require.NoError(t, err)
	require.Nil(t, charge)
	require.Zero(t, updateCalls)
}

func TestMachine_AdvanceUntilStateStableWalksTransitionsAndPersistsReturnedBase(t *testing.T) {
	// Given:
	// a machine that can advance from created to active to final.
	// When:
	// AdvanceUntilStateStable is called.
	// Then:
	// it walks all next transitions and the returned charge contains the persisted base returned by UpdateBase.
	var updateCalls int

	machine := newTestMachine(
		t,
		newFakeCharge(fakeStatusCreated),
		func(ctx context.Context, base fakeBase) (fakeBase, error) {
			updateCalls++
			base.Revision++
			return base, nil
		},
		func(ctx context.Context, chargeID meta.ChargeID) (fakeCharge, error) { return fakeCharge{}, nil },
	)

	machine.Configure(fakeStatusCreated).Permit(meta.TriggerNext, fakeStatusActive)
	machine.Configure(fakeStatusActive).Permit(meta.TriggerNext, fakeStatusFinal)

	charge, err := machine.AdvanceUntilStateStable(t.Context())

	require.NoError(t, err)
	require.NotNil(t, charge)
	require.Equal(t, fakeStatusFinal, charge.GetStatus())
	require.Equal(t, 2, charge.GetBase().Revision)
	require.Equal(t, 2, updateCalls)
}

func TestMachine_AdvanceUntilStateStablePersistsPostActivationBase(t *testing.T) {
	// Given:
	// a machine whose activation logic mutates the in-memory base before persistence.
	// When:
	// AdvanceUntilStateStable is called.
	// Then:
	// UpdateBase receives the post-activation base and the returned charge contains the persisted result.
	var observedBase fakeBase

	machine := newTestMachine(
		t,
		newFakeCharge(fakeStatusCreated),
		func(ctx context.Context, base fakeBase) (fakeBase, error) {
			observedBase = base
			base.Revision++
			return base, nil
		},
		func(ctx context.Context, chargeID meta.ChargeID) (fakeCharge, error) { return fakeCharge{}, nil },
	)

	machine.Configure(fakeStatusCreated).
		Permit(meta.TriggerNext, fakeStatusActive)
	machine.Configure(fakeStatusActive).OnActive(func(ctx context.Context) error {
		machine.Charge = machine.Charge.WithBase(fakeBase{Revision: 7})
		return nil
	})

	charge, err := machine.AdvanceUntilStateStable(t.Context())

	require.NoError(t, err)
	require.NotNil(t, charge)
	require.Equal(t, 7, observedBase.Revision)
	require.Equal(t, 8, charge.GetBase().Revision)
}

func TestMachine_FireAndActivatePropagatesActivationErrors(t *testing.T) {
	// Given:
	// a machine whose activation callback fails after a valid transition fires.
	// When:
	// FireAndActivate is called.
	// Then:
	// the activation error is returned to the caller.
	activationErr := errors.New("activation failed")

	machine := newTestMachine(
		t,
		newFakeCharge(fakeStatusCreated),
		func(ctx context.Context, base fakeBase) (fakeBase, error) { return base, nil },
		func(ctx context.Context, chargeID meta.ChargeID) (fakeCharge, error) { return fakeCharge{}, nil },
	)

	machine.Configure(fakeStatusCreated).
		Permit(meta.TriggerNext, fakeStatusActive)
	machine.Configure(fakeStatusActive).OnActive(func(ctx context.Context) error {
		return activationErr
	})

	err := machine.FireAndActivate(t.Context(), meta.TriggerNext)

	require.ErrorIs(t, err, activationErr)
}

func TestMachine_FireAndActivateFailsFastOnInvalidTargetStatus(t *testing.T) {
	// Given:
	// a machine with a transition targeting an invalid status value.
	// When:
	// FireAndActivate is called.
	// Then:
	// it fails before any persistence logic is involved.
	machine := newTestMachine(
		t,
		newFakeCharge(fakeStatusCreated),
		func(ctx context.Context, base fakeBase) (fakeBase, error) {
			t.Fatal("UpdateBase must not be called")
			return fakeBase{}, nil
		},
		func(ctx context.Context, chargeID meta.ChargeID) (fakeCharge, error) { return fakeCharge{}, nil },
	)

	machine.Configure(fakeStatusCreated).Permit(meta.TriggerNext, fakeStatus("broken"))

	err := machine.FireAndActivate(t.Context(), meta.TriggerNext)

	require.Error(t, err)
	require.ErrorContains(t, err, "invalid status")
}

func TestMachine_RefetchChargeReplacesTheInMemoryCharge(t *testing.T) {
	// Given:
	// a machine whose persistence layer can refetch a different copy of the charge.
	// When:
	// RefetchCharge is called.
	// Then:
	// the machine replaces the in-memory charge with the refetched object.
	machine := newTestMachine(
		t,
		newFakeCharge(fakeStatusCreated),
		func(ctx context.Context, base fakeBase) (fakeBase, error) { return base, nil },
		func(ctx context.Context, chargeID meta.ChargeID) (fakeCharge, error) {
			charge := newFakeCharge(fakeStatusFinal)
			charge.Base = fakeBase{Revision: 11}
			charge.Marker = "refetched"
			return charge, nil
		},
	)

	err := machine.RefetchCharge(t.Context())

	require.NoError(t, err)
	require.Equal(t, fakeStatusFinal, machine.GetCharge().GetStatus())
	require.Equal(t, 11, machine.GetCharge().GetBase().Revision)
	require.Equal(t, "refetched", machine.GetCharge().Marker)
}

func TestMachine_AdvanceUntilStateStablePropagatesPersistenceErrors(t *testing.T) {
	// Given:
	// a machine that can advance but fails while persisting the updated base.
	// When:
	// AdvanceUntilStateStable is called.
	// Then:
	// it returns the wrapped persistence error and stops advancing.
	persistErr := errors.New("persist failed")

	machine := newTestMachine(
		t,
		newFakeCharge(fakeStatusCreated),
		func(ctx context.Context, base fakeBase) (fakeBase, error) {
			return fakeBase{}, persistErr
		},
		func(ctx context.Context, chargeID meta.ChargeID) (fakeCharge, error) { return fakeCharge{}, nil },
	)

	machine.Configure(fakeStatusCreated).Permit(meta.TriggerNext, fakeStatusActive)

	charge, err := machine.AdvanceUntilStateStable(t.Context())

	require.Nil(t, charge)
	require.ErrorIs(t, err, persistErr)
	require.ErrorContains(t, err, "persist charge")
}
