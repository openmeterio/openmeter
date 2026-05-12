package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	flatfeerealizations "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee/service/realizations"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/statelessx"
)

type CreditsOnlyStateMachine struct {
	*stateMachine
}

func NewCreditsOnlyStateMachine(config StateMachineConfig) (*CreditsOnlyStateMachine, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	if config.Charge.Intent.SettlementMode != productcatalog.CreditOnlySettlementMode {
		return nil, fmt.Errorf("charge %s is not credit_only", config.Charge.ID)
	}

	stateMachine, err := newStateMachineBase(config)
	if err != nil {
		return nil, fmt.Errorf("new state machine: %w", err)
	}

	out := &CreditsOnlyStateMachine{
		stateMachine: stateMachine,
	}
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
		OnActive(s.AllocateCredits)

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

	if s.Charge.Realizations.CurrentRun == nil {
		runBase, err := s.Adapter.ProvisionCurrentRun(ctx, flatfee.ProvisionCurrentRunInput{
			Charge:                    s.Charge.ChargeBase,
			NoFiatTransactionRequired: true, // We are in credits-only mode
		})
		if err != nil {
			return fmt.Errorf("provision current run: %w", err)
		}

		s.Charge.Realizations.CurrentRun = &flatfee.RealizationRun{
			RealizationRunBase: runBase,
		}
	}

	result, err := s.Realizations.AllocateCreditsOnly(ctx, flatfeerealizations.AllocateCreditsOnlyInput{
		Charge:             s.Charge,
		Amount:             amount,
		CurrencyCalculator: currencyCalculator,
	})
	if err != nil {
		return fmt.Errorf("allocate credits: %w", err)
	}

	s.Charge.Realizations.CurrentRun.CreditRealizations = append(s.Charge.Realizations.CurrentRun.CreditRealizations, result.Realizations...)
	return nil
}

func (s *CreditsOnlyStateMachine) DeleteCharge(ctx context.Context, policy meta.PatchDeletePolicy) error {
	if policy.CreditRefundPolicy == meta.CreditRefundPolicyCorrect && s.Charge.Realizations.CurrentRun != nil {
		currencyCalculator, err := s.Charge.Intent.Currency.Calculator()
		if err != nil {
			return fmt.Errorf("get currency calculator: %w", err)
		}

		if _, err := s.Realizations.CorrectAllCredits(ctx, flatfeerealizations.CorrectAllCreditRealizationsInput{
			Charge:             s.Charge,
			AllocateAt:         clock.Now(),
			CurrencyCalculator: currencyCalculator,
		}); err != nil {
			return fmt.Errorf("correct credits: %w", err)
		}
	}

	if err := s.Adapter.DeleteCharge(ctx, s.Charge); err != nil {
		return fmt.Errorf("delete charge: %w", err)
	}

	if err := s.RefetchCharge(ctx); err != nil {
		return fmt.Errorf("get charge: %w", err)
	}

	return nil
}
