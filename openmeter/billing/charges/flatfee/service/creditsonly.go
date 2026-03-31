package service

import (
	"context"
	"fmt"

	"github.com/qmuntal/stateless"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/statelessx"
)

type CreditsOnlyStateMachine struct {
	*stateless.StateMachine

	Charge  flatfee.Charge
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
			newStatus := state.(meta.ChargeStatus)
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
	s.Configure(meta.ChargeStatusCreated).
		Permit(
			flatfee.TriggerNext,
			meta.ChargeStatusActive,
			statelessx.BoolFn(s.IsAfterInvoiceAt),
		).
		OnActive(
			s.SetAdvanceAfterInvoiceAt,
		)

	s.Configure(meta.ChargeStatusActive).
		Permit(
			flatfee.TriggerNext,
			meta.ChargeStatusFinal,
		).
		OnActive(
			s.AllocateCredits,
		)

	s.Configure(meta.ChargeStatusFinal).
		OnActive(s.ClearAdvanceAfter)
}

func (s *CreditsOnlyStateMachine) IsAfterInvoiceAt() bool {
	return !clock.Now().Before(s.Charge.Intent.InvoiceAt)
}

func (s *CreditsOnlyStateMachine) SetAdvanceAfterInvoiceAt(ctx context.Context) error {
	s.Charge.State.AdvanceAfter = lo.ToPtr(s.Charge.Intent.InvoiceAt)
	return nil
}

func (s *CreditsOnlyStateMachine) ClearAdvanceAfter(ctx context.Context) error {
	s.Charge.State.AdvanceAfter = nil
	return nil
}

func (s *CreditsOnlyStateMachine) AllocateCredits(ctx context.Context) error {
	amount := s.Charge.State.AmountAfterProration

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
		realizations, err := s.Adapter.CreateCreditAllocations(ctx, s.Charge.GetChargeID(), creditAllocations.AsCreateInputs())
		if err != nil {
			return fmt.Errorf("create credit allocations: %w", err)
		}

		s.Charge.State.CreditRealizations = append(s.Charge.State.CreditRealizations, realizations...)
	}

	return nil
}

func (s *CreditsOnlyStateMachine) FireAndActivate(ctx context.Context, trigger flatfee.Trigger) error {
	if err := s.StateMachine.FireCtx(ctx, trigger); err != nil {
		return err
	}

	return s.StateMachine.ActivateCtx(ctx)
}

func (s *CreditsOnlyStateMachine) AdvanceUntilStateStable(ctx context.Context) (*flatfee.Charge, error) {
	var advanced bool

	for {
		canFire, err := s.StateMachine.CanFireCtx(ctx, flatfee.TriggerNext)
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

		if err := s.FireAndActivate(ctx, flatfee.TriggerNext); err != nil {
			return nil, fmt.Errorf("cannot transition to the next status [current_status=%s]: %w", s.Charge.Status, err)
		}

		if err := s.Adapter.UpdateCharge(ctx, s.Charge); err != nil {
			return nil, fmt.Errorf("persist charge: %w", err)
		}

		advanced = true
	}
}
