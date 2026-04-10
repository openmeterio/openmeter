package statemachine

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/qmuntal/stateless"
)

type StateMutator[T any] interface {
	GetState(_ context.Context) (stateless.State, error)
	SetState(_ context.Context, newState stateless.State) error
	PersistChange(_ context.Context) error

	GetEntity() T
}

type Base[T any] interface {
	FireAndActivate(ctx context.Context, trigger meta.Trigger, args ...any) error
	AdvanceUntilStateStable(ctx context.Context) (*T, error)
}

type base[T any] struct {
	*stateless.StateMachine

	stateMutator StateMutator[T]
}

func New[T any](stateMutator StateMutator[T]) (*base[T], error) {
	if stateMutator == nil {
		return nil, fmt.Errorf("state mutator is required")
	}

	out := &base[T]{
		stateMutator: stateMutator,
	}

	stateMachine := stateless.NewStateMachineWithExternalStorage(
		func(ctx context.Context) (stateless.State, error) {
			return out.stateMutator.GetState(ctx)
		},
		func(ctx context.Context, state stateless.State) error {
			return out.stateMutator.SetState(ctx, state)
		},
		stateless.FiringImmediate,
	)

	out.StateMachine = stateMachine

	return out, nil
}

func (s *base[T]) FireAndActivate(ctx context.Context, trigger meta.Trigger, args ...any) error {
	if err := s.StateMachine.FireCtx(ctx, trigger, args...); err != nil {
		return err
	}

	return s.StateMachine.ActivateCtx(ctx)
}

func (s *base[T]) AdvanceUntilStateStable(ctx context.Context) (*T, error) {
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

			entity := s.stateMutator.GetEntity()
			return &entity, nil
		}

		if err := s.FireAndActivate(ctx, meta.TriggerNext); err != nil {
			return nil, fmt.Errorf("cannot transition to the next status [current_status=%s]: %w", s.stateOrUnknown(ctx), err)
		}

		if err := s.stateMutator.PersistChange(ctx); err != nil {
			return nil, fmt.Errorf("persist change: %w", err)
		}

		advanced = true
	}
}

func (s *base[T]) stateOrUnknown(ctx context.Context) stateless.State {
	state, err := s.stateMutator.GetState(ctx)
	if err != nil {
		state = "unknown"
	}

	return state
}
