package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	chargestatemachine "github.com/openmeterio/openmeter/openmeter/billing/charges/statemachine"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (s *service) onExternalCreditPurchase(ctx context.Context, charge creditpurchase.Charge) (creditpurchase.Charge, error) {
	externalCreditPurchaseSettlement, err := charge.Intent.Settlement.AsExternalSettlement()
	if err != nil {
		return creditpurchase.Charge{}, err
	}

	trigger, err := externalInitialPaymentTrigger(externalCreditPurchaseSettlement.InitialStatus)
	if err != nil {
		return creditpurchase.Charge{}, err
	}

	stateMachine, err := s.newExternalCreditPurchaseStateMachine(charge)
	if err != nil {
		return creditpurchase.Charge{}, fmt.Errorf("new external state machine: %w", err)
	}

	advancedCharge, err := stateMachine.AdvanceUntilStateStable(ctx)
	if err != nil {
		return creditpurchase.Charge{}, fmt.Errorf("advance external state machine: %w", err)
	}

	charge = lo.FromPtrOr(advancedCharge, charge)

	if trigger == "" {
		return charge, nil
	}

	charge, err = stateMachine.handleExternalPaymentLifecycleTrigger(ctx, trigger)
	if err != nil {
		return creditpurchase.Charge{}, fmt.Errorf("fire external payment trigger %s: %w", trigger, err)
	}

	return charge, nil
}

func externalInitialPaymentTrigger(status creditpurchase.InitialPaymentSettlementStatus) (meta.Trigger, error) {
	switch status {
	case creditpurchase.CreatedInitialPaymentSettlementStatus:
		return "", nil
	case creditpurchase.AuthorizedInitialPaymentSettlementStatus:
		return billing.TriggerAuthorized, nil
	case creditpurchase.SettledInitialPaymentSettlementStatus:
		return billing.TriggerPaid, nil
	default:
		return "", fmt.Errorf("invalid initial payment settlement status: %s", status)
	}
}

type ExternalCreditPurchaseStateMachine struct {
	*stateMachine
}

func NewExternalCreditPurchaseStateMachine(config StateMachineConfig) (*ExternalCreditPurchaseStateMachine, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	if config.Realizations == nil {
		return nil, fmt.Errorf("realizations service is required")
	}

	if config.Charge.Intent.Settlement.Type() != creditpurchase.SettlementTypeExternal {
		return nil, fmt.Errorf("charge %s is not external", config.Charge.ID)
	}

	stateMachine, err := newStateMachineBase(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create external credit purchase state machine: %w", err)
	}

	out := &ExternalCreditPurchaseStateMachine{
		stateMachine: stateMachine,
	}
	out.configureStates()

	return out, nil
}

func (s *ExternalCreditPurchaseStateMachine) configureStates() {
	s.Configure(creditpurchase.StatusCreated).
		Permit(meta.TriggerNext, creditpurchase.StatusActivePaymentPending)

	s.Configure(creditpurchase.StatusActive).
		Permit(meta.TriggerNext, creditpurchase.StatusActivePaymentPending)

	s.Configure(creditpurchase.StatusActiveInitialCreditGrant).
		Permit(meta.TriggerNext, creditpurchase.StatusActivePaymentPending).
		OnActive(s.InitiateExternalCreditPurchaseFromCreated)

	s.Configure(creditpurchase.StatusActivePaymentPending).
		Permit(billing.TriggerAuthorized, creditpurchase.StatusActivePaymentAuthorized).
		Permit(billing.TriggerPaid, creditpurchase.StatusActivePaymentPaidAndAuthorized)

	s.Configure(creditpurchase.StatusActivePaymentAuthorized).
		OnActive(s.AuthorizeExternalPayment).
		Permit(billing.TriggerPaid, creditpurchase.StatusActivePaymentSettled)

	s.Configure(creditpurchase.StatusActivePaymentPaidAndAuthorized).
		Permit(billing.TriggerNext, creditpurchase.StatusActivePaymentSettled).
		OnActive(s.AuthorizeExternalPayment)

	s.Configure(creditpurchase.StatusActivePaymentSettled).
		Permit(billing.TriggerNext, creditpurchase.StatusFinal).
		OnActive(s.SettleExternalPayment)

	s.Configure(creditpurchase.StatusFinal)
}

func (s *ExternalCreditPurchaseStateMachine) AdvanceUntilStateStable(ctx context.Context) (*creditpurchase.Charge, error) {
	hadGrant := hasExternalCreditGrant(s.Charge)

	advancedCharge, err := s.stateMachine.Machine.AdvanceUntilStateStable(ctx)
	if err != nil {
		return nil, err
	}

	switch s.Charge.Status {
	case creditpurchase.StatusActivePaymentPending, creditpurchase.StatusActivePaymentAuthorized:
		if err := s.EnsureExternalCreditPurchaseInitiated(ctx); err != nil {
			return nil, err
		}
	}

	if advancedCharge != nil || (!hadGrant && hasExternalCreditGrant(s.Charge)) {
		charge := s.GetCharge()
		return &charge, nil
	}

	return nil, nil
}

func (s *ExternalCreditPurchaseStateMachine) InitiateExternalCreditPurchaseFromCreated(ctx context.Context) error {
	return s.applyRealizationUpdate(ctx, s.Realizations.InitiateExternalCreditPurchase)
}

func (s *ExternalCreditPurchaseStateMachine) EnsureExternalCreditPurchaseInitiated(ctx context.Context) error {
	if hasExternalCreditGrant(s.Charge) {
		return nil
	}

	return s.InitiateExternalCreditPurchaseFromCreated(ctx)
}

func (s *ExternalCreditPurchaseStateMachine) AuthorizeExternalPayment(ctx context.Context) error {
	return s.applyRealizationUpdate(ctx, s.Realizations.AuthorizeExternalPayment)
}

func (s *ExternalCreditPurchaseStateMachine) SettleExternalPayment(ctx context.Context) error {
	return s.applyRealizationUpdate(ctx, s.Realizations.SettleExternalPayment)
}

// applyRealizationUpdate records realization-side effects while preserving the
// state-machine-owned base from the current transition.
func (s *ExternalCreditPurchaseStateMachine) applyRealizationUpdate(
	ctx context.Context,
	update func(context.Context, creditpurchase.Charge) (creditpurchase.Charge, error),
) error {
	updatedCharge, err := update(ctx, s.Charge)
	if err != nil {
		return err
	}

	s.Charge = updatedCharge.WithBase(s.Charge.ChargeBase)
	return nil
}

func (s *service) newExternalCreditPurchaseStateMachine(charge creditpurchase.Charge) (*ExternalCreditPurchaseStateMachine, error) {
	return NewExternalCreditPurchaseStateMachine(StateMachineConfig{
		Charge:       charge,
		Adapter:      s.adapter,
		Realizations: s.realizations,
	})
}

func (s *ExternalCreditPurchaseStateMachine) handleExternalPaymentLifecycleTrigger(ctx context.Context, trigger meta.Trigger) (creditpurchase.Charge, error) {
	if _, err := s.AdvanceUntilStateStable(ctx); err != nil {
		return creditpurchase.Charge{}, fmt.Errorf("advance external state machine: %w", err)
	}

	if trigger == billing.TriggerPaid {
		return s.handleSettledExternalPayment(ctx)
	}

	return s.FireAndAdvanceUntilStateStable(ctx, trigger)
}

func hasExternalCreditGrant(charge creditpurchase.Charge) bool {
	return charge.Realizations.CreditGrantRealization != nil &&
		charge.Realizations.CreditGrantRealization.TransactionGroupID != ""
}

// handleSettledExternalPayment handles provider flows that may report paid
// without a separate authorization event, preserving authorization before settlement.
func (s *ExternalCreditPurchaseStateMachine) handleSettledExternalPayment(ctx context.Context) (creditpurchase.Charge, error) {
	if s.Charge.Status == creditpurchase.StatusActivePaymentPending {
		if _, err := s.FireAndAdvanceUntilStateStable(ctx, billing.TriggerAuthorized); err != nil {
			return creditpurchase.Charge{}, err
		}
	}

	if s.Charge.Status != creditpurchase.StatusActivePaymentAuthorized {
		return creditpurchase.Charge{}, fmt.Errorf(
			"%w: %s [status=%s,id=%s]",
			chargestatemachine.ErrUnsupportedOperation,
			billing.TriggerPaid,
			s.Charge.Status,
			s.Charge.GetChargeID().ID,
		)
	}

	if err := s.SettleExternalPayment(ctx); err != nil {
		return creditpurchase.Charge{}, err
	}

	s.Charge = s.Charge.WithStatus(creditpurchase.StatusFinal)
	updatedBase, err := s.Adapter.UpdateCharge(ctx, s.Charge.GetBase())
	if err != nil {
		return creditpurchase.Charge{}, fmt.Errorf("persist charge: %w", err)
	}

	s.Charge = s.Charge.WithBase(updatedBase)

	return s.GetCharge(), nil
}

func (s *service) HandleExternalPaymentAuthorized(ctx context.Context, charge creditpurchase.Charge) (creditpurchase.Charge, error) {
	return s.handleExternalPaymentTrigger(ctx, charge, billing.TriggerAuthorized)
}

func (s *service) HandleExternalPaymentSettled(ctx context.Context, charge creditpurchase.Charge) (creditpurchase.Charge, error) {
	return s.handleExternalPaymentTrigger(ctx, charge, billing.TriggerPaid)
}

func (s *service) handleExternalPaymentTrigger(ctx context.Context, charge creditpurchase.Charge, trigger meta.Trigger) (creditpurchase.Charge, error) {
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (creditpurchase.Charge, error) {
		stateMachine, err := s.newExternalCreditPurchaseStateMachine(charge)
		if err != nil {
			return creditpurchase.Charge{}, err
		}

		return stateMachine.handleExternalPaymentLifecycleTrigger(ctx, trigger)
	})
}
