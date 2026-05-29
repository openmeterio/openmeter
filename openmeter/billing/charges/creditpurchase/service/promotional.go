package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/pkg/clock"
)

func (s *service) grantPromotionalCredit(ctx context.Context, charge creditpurchase.Charge) (creditpurchase.Charge, error) {
	ledgerTransactionGroupReference, err := s.handler.OnPromotionalCreditPurchase(ctx, charge)
	if err != nil {
		return creditpurchase.Charge{}, err
	}

	grantRealization, err := s.adapter.CreateCreditGrant(ctx, charge.GetChargeID(), creditpurchase.CreateCreditGrantInput{
		TransactionGroupID: ledgerTransactionGroupReference.TransactionGroupID,
		GrantedAt:          clock.Now(),
	})
	if err != nil {
		return creditpurchase.Charge{}, err
	}

	charge.Realizations.CreditGrantRealization = &grantRealization

	if ledgerTransactionGroupReference.TransactionGroupID != "" {
		if err := s.lineage.BackfillAdvanceLineageSegments(ctx, lineage.BackfillAdvanceLineageSegmentsInput{
			Namespace:                 charge.Namespace,
			CustomerID:                charge.Intent.CustomerID,
			Currency:                  charge.Intent.Currency,
			Amount:                    charge.Intent.CreditAmount,
			BackingTransactionGroupID: ledgerTransactionGroupReference.TransactionGroupID,
		}); err != nil {
			return creditpurchase.Charge{}, err
		}
	}

	return charge, nil
}

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
		OnEntry(func(ctx context.Context, _ ...any) error {
			return s.GrantPromotionalCredit(ctx)
		})
}

func (s *PromotionalCreditpurchaseStateMachine) GrantPromotionalCredit(ctx context.Context) error {
	charge, err := s.Service.grantPromotionalCredit(ctx, s.Charge)
	if err != nil {
		return fmt.Errorf("grant promotional credit: %w", err)
	}

	s.Charge = charge
	return nil
}
