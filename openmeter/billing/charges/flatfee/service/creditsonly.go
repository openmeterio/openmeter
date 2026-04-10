package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/qmuntal/stateless"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/statemachine"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/statelessx"
)

var _ statemachine.StateMutator[flatfee.Charge] = (*StateMutator)(nil)

type StateMutator struct {
	adapter flatfee.Adapter
}

func (s StateMutator) GetState(_ context.Context, charge flatfee.Charge) (stateless.State, error) {
	return charge.Status, nil
}

func (s StateMutator) SetState(_ context.Context, charge *flatfee.Charge, newState stateless.State) error {
	newStatus := newState.(meta.ChargeStatus)
	if err := newStatus.Validate(); err != nil {
		return fmt.Errorf("invalid status: %w", err)
	}

	charge.Status = newStatus
	return nil
}

func (s StateMutator) PersistChargeBase(ctx context.Context, charge flatfee.Charge) (flatfee.Charge, error) {
	updatedChargeBase, err := s.adapter.UpdateCharge(ctx, charge.ChargeBase)
	if err != nil {
		return flatfee.Charge{}, fmt.Errorf("persist charge base: %w", err)
	}

	charge.ChargeBase = updatedChargeBase
	return charge, nil
}

func (s StateMutator) SetOrClearAdvanceAfter(charge flatfee.Charge, advanceAfter *time.Time) flatfee.Charge {
	charge.State.AdvanceAfter = advanceAfter
	return charge
}

type CreditsOnlyStateMachine struct {
	*statemachine.Base[flatfee.Charge]

	Service *service
	Adapter flatfee.Adapter
}

type CreditsOnlyStateMachineConfig struct {
	Charge  flatfee.Charge
	Service *service
}

func (c CreditsOnlyStateMachineConfig) Validate() error {
	if err := c.Charge.Validate(); err != nil {
		return fmt.Errorf("charge: %w", err)
	}

	if c.Service == nil {
		return fmt.Errorf("service is required")
	}

	return nil
}

func NewCreditsOnlyStateMachine(config CreditsOnlyStateMachineConfig) (*CreditsOnlyStateMachine, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	if config.Charge.Intent.SettlementMode != productcatalog.CreditOnlySettlementMode {
		return nil, fmt.Errorf("charge %s is not credit_only", config.Charge.ID)
	}

	base, err := statemachine.New[flatfee.Charge](config.Charge, StateMutator{
		adapter: config.Service.adapter,
	})
	if err != nil {
		return nil, fmt.Errorf("new base: %w", err)
	}

	if base == nil {
		return nil, errors.New("base is nil")
	}

	out := &CreditsOnlyStateMachine{
		Charge:  config.Charge,
		Service: config.Service,
		Adapter: config.Service.adapter,
	}

	sm := stateless.NewStateMachineWithExternalStorage(
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

	out.StateMachine = sm
	out.configureStates()

	return out, nil
}

func (s *CreditsOnlyStateMachine) configureStates() {
	s.Configure(flatfee.StatusCreated).
		Permit(meta.TriggerNext, flatfee.StatusActive, statelessx.BoolFn(s.IsAfterInvoiceAt)).
		Permit(meta.TriggerDelete, flatfee.StatusDeleted).
		OnActive(
			s.SetAdvanceAfterInvoiceAt,
		)

	s.Configure(flatfee.StatusActive).
		Permit(meta.TriggerNext, flatfee.StatusFinal).
		Permit(meta.TriggerDelete, flatfee.StatusDeleted).
		OnActive(
			s.AllocateCredits,
		)

	s.Configure(flatfee.StatusFinal).
		Permit(meta.TriggerDelete, flatfee.StatusDeleted).
		OnActive(s.ClearAdvanceAfter)

	s.Configure(flatfee.StatusDeleted).
		OnEntry(statelessx.WithParameters(s.DeleteCharge))
}

func (s *CreditsOnlyStateMachine) IsAfterInvoiceAt() bool {
	return !clock.Now().Before(s.Charge.Intent.InvoiceAt)
}

func (s *CreditsOnlyStateMachine) SetAdvanceAfterInvoiceAt(ctx context.Context) error {
	s.Charge.State.AdvanceAfter = lo.ToPtr(meta.NormalizeTimestamp(s.Charge.Intent.InvoiceAt))
	return nil
}

func (s *CreditsOnlyStateMachine) ClearAdvanceAfter(ctx context.Context) error {
	s.Charge.State.AdvanceAfter = nil
	return nil
}

func (s *CreditsOnlyStateMachine) AllocateCredits(ctx context.Context) error {
	currencyCalculator, err := s.Charge.Intent.Currency.Calculator()
	if err != nil {
		return fmt.Errorf("get currency calculator: %w", err)
	}

	amount := currencyCalculator.RoundToPrecision(s.Charge.State.AmountAfterProration)

	if amount.IsNegative() {
		return fmt.Errorf("charge total is negative [charge_id=%s, amount=%s]", s.Charge.ID, amount.String())
	}

	var creditAllocations creditrealization.CreateAllocationInputs
	if !amount.IsZero() {
		input := flatfee.OnCreditsOnlyUsageAccruedInput{
			Charge:           s.Charge,
			AmountToAllocate: amount,
		}

		if err := input.Validate(); err != nil {
			return fmt.Errorf("validate input: %w", err)
		}

		var err error
		creditAllocations, err = s.Service.handler.OnCreditsOnlyUsageAccrued(ctx, input)
		if err != nil {
			return fmt.Errorf("on credits only usage accrued: %w", err)
		}

		if !creditAllocations.Sum().Equal(amount) {
			return models.NewGenericValidationError(
				fmt.Errorf("credit allocations do not match total [charge_id=%s, total=%s, allocations_sum=%s]",
					s.Charge.ID, amount.String(), creditAllocations.Sum().String()),
			)
		}
	}

	if len(creditAllocations) > 0 {
		realizations, err := s.Service.createCreditAllocations(ctx, s.Charge, creditAllocations.AsCreateInputs())
		if err != nil {
			return fmt.Errorf("create credit allocations: %w", err)
		}

		s.Charge.Realizations.CreditRealizations = append(s.Charge.Realizations.CreditRealizations, realizations...)
	}

	return nil
}

// TODO: Move these into some helper base package

var ErrUnsupportedOperation = models.NewGenericPreConditionFailedError(fmt.Errorf("unsupported operation"))

func (s *CreditsOnlyStateMachine) FireAndActivate(ctx context.Context, trigger meta.Trigger, args ...any) error {
	canFire, err := s.StateMachine.CanFireCtx(ctx, trigger)
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

func (s *CreditsOnlyStateMachine) AdvanceUntilStateStable(ctx context.Context) (*flatfee.Charge, error) {
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

func (s *CreditsOnlyStateMachine) DeleteCharge(ctx context.Context, policy meta.PatchDeletePolicy) error {
	if policy.CreditRefundPolicy == meta.CreditRefundPolicyCorrect {
		currencyCalculator, err := s.Charge.Intent.Currency.Calculator()
		if err != nil {
			return fmt.Errorf("get currency calculator: %w", err)
		}

		realizationIDs := lo.Map(s.Charge.Realizations.CreditRealizations, func(realization creditrealization.Realization, _ int) string {
			return realization.ID
		})
		lineageSegmentsByRealization, err := s.Service.lineage.LoadActiveSegmentsByRealizationID(ctx, s.Charge.Namespace, realizationIDs)
		if err != nil {
			return fmt.Errorf("load active lineage segments: %w", err)
		}

		// Let's reverse the credit allocations
		corrections, err := s.Charge.Realizations.CreditRealizations.CorrectAll(currencyCalculator, func(req creditrealization.CorrectionRequest) (creditrealization.CreateCorrectionInputs, error) {
			return s.Service.handler.OnCreditsOnlyUsageAccruedCorrection(ctx, flatfee.CreditsOnlyUsageAccruedCorrectionInput{
				Charge:                       s.Charge,
				AllocateAt:                   clock.Now(),
				Corrections:                  req,
				LineageSegmentsByRealization: lineageSegmentsByRealization,
			})
		})
		if err != nil {
			return fmt.Errorf("correct credits: %w", err)
		}

		if len(corrections) > 0 {
			if _, err := s.Service.createCreditAllocations(ctx, s.Charge, corrections); err != nil {
				return fmt.Errorf("create credit corrections: %w", err)
			}
		}
	}

	if err := s.Adapter.DeleteCharge(ctx, s.Charge); err != nil {
		return fmt.Errorf("delete charge: %w", err)
	}

	return s.refetchCharge(ctx)
}

func (s *CreditsOnlyStateMachine) refetchCharge(ctx context.Context) error {
	charge, err := s.Service.GetByID(ctx, flatfee.GetByIDInput{
		ChargeID: s.Charge.GetChargeID(),
	})
	if err != nil {
		return fmt.Errorf("get charge: %w", err)
	}

	s.Charge = charge
	return nil
}
