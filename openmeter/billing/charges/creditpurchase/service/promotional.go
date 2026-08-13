package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/pkg/statelessx"
)

type PromotionalCreditpurchaseStateMachine struct {
	*stateMachine
}

func NewPromotionalCreditPurchaseStateMachine(config StateMachineConfig) (*PromotionalCreditpurchaseStateMachine, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	if config.Charge.Intent.Settlement.Type() != creditpurchase.SettlementTypePromotional {
		return nil, fmt.Errorf("charge %s is not promotional", config.Charge.ID)
	}

	stateMachine, err := newStateMachineBase(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create promotional credit purchase state machine: %w", err)
	}

	out := &PromotionalCreditpurchaseStateMachine{
		stateMachine: stateMachine,
	}
	out.configureStates()

	return out, nil
}

func (s *PromotionalCreditpurchaseStateMachine) configureStates() {
	s.Configure(creditpurchase.StatusCreated).
		Permit(meta.TriggerNext, creditpurchase.StatusFinal)

	s.Configure(creditpurchase.StatusActive).
		Permit(meta.TriggerNext, creditpurchase.StatusFinal)

	s.Configure(creditpurchase.StatusFinal).
		OnEntry(statelessx.EntryFunc(s.GrantPromotionalCredit))
}

func (s *PromotionalCreditpurchaseStateMachine) GrantPromotionalCredit(ctx context.Context) error {
	charge, err := s.Realizations.GrantPromotionalCredits(ctx, s.Charge)
	if err != nil {
		return err
	}

	s.Charge = charge
	return nil
}
