package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
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
		Permit(meta.TriggerNext, creditpurchase.StatusActiveInitialCreditGrant)

	s.Configure(creditpurchase.StatusActive).
		Permit(meta.TriggerNext, creditpurchase.StatusActivePaymentPending)

	s.Configure(creditpurchase.StatusActiveInitialCreditGrant).
		Permit(meta.TriggerNext, creditpurchase.StatusActivePaymentPending).
		OnActive(s.GrantCredits)

	s.Configure(creditpurchase.StatusActivePaymentPending).
		Permit(billing.TriggerAuthorized, creditpurchase.StatusActivePaymentAuthorized).
		Permit(billing.TriggerPaid, creditpurchase.StatusActivePaymentPaidAndAuthorized)

	s.Configure(creditpurchase.StatusActivePaymentAuthorized).
		OnActive(s.AuthorizeExternalPayment).
		Permit(billing.TriggerPaid, creditpurchase.StatusActivePaymentSettled)

	s.Configure(creditpurchase.StatusActivePaymentPaidAndAuthorized).
		Permit(meta.TriggerNext, creditpurchase.StatusActivePaymentSettled).
		OnActive(s.AuthorizeExternalPayment)

	s.Configure(creditpurchase.StatusActivePaymentSettled).
		Permit(meta.TriggerNext, creditpurchase.StatusFinal).
		OnActive(s.SettleExternalPayment)

	s.Configure(creditpurchase.StatusFinal)
}

func (s *ExternalCreditPurchaseStateMachine) GrantCredits(ctx context.Context) error {
	updatedCharge, err := s.Realizations.GrantCredits(ctx, s.Charge)
	if err != nil {
		return err
	}

	s.Charge = updatedCharge
	return nil
}

func (s *ExternalCreditPurchaseStateMachine) AuthorizeExternalPayment(ctx context.Context) error {
	updatedCharge, err := s.Realizations.AuthorizeExternalPayment(ctx, s.Charge)
	if err != nil {
		return err
	}

	s.Charge = updatedCharge
	return nil
}

func (s *ExternalCreditPurchaseStateMachine) SettleExternalPayment(ctx context.Context) error {
	updatedCharge, err := s.Realizations.SettleExternalPayment(ctx, s.Charge)
	if err != nil {
		return err
	}

	s.Charge = updatedCharge
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

	return s.FireAndAdvanceUntilStateStable(ctx, trigger)
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
