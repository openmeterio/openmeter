package statemachine

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/statelessx"
)

type CreditThenInvoice struct {
	*Base
}

func NewCreditThenInvoice(config Config) (*CreditThenInvoice, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	if config.Charge.Intent.SettlementMode != productcatalog.CreditThenInvoiceSettlementMode {
		return nil, fmt.Errorf("charge %s is not credit_then_invoice", config.Charge.ID)
	}

	base, err := newBase(config)
	if err != nil {
		return nil, fmt.Errorf("new base: %w", err)
	}

	out := CreditThenInvoice{
		Base: base,
	}

	out.configureStates()

	return &out, nil
}

func (s *CreditThenInvoice) configureStates() {
	s.Configure(usagebased.StatusCreated).
		Permit(
			meta.TriggerNext,
			usagebased.StatusActive,
			statelessx.BoolFn(s.IsInsideServicePeriod),
		).
		Permit(meta.TriggerDelete, usagebased.StatusDeleted).
		OnActive(
			s.AdvanceAfterServicePeriodFrom,
		)

	s.Configure(usagebased.StatusActive).
		Permit(
			meta.TriggerInvoiceCreated,
			usagebased.StatusActiveFinalRealizationStarted,
			statelessx.BoolFn(s.IsAfterServicePeriod),
		).
		Permit(meta.TriggerDelete, usagebased.StatusDeleted).
		OnActive(
			statelessx.AllOf(
				s.SyncFeatureIDFromFeatureMeter,
				s.AdvanceAfterServicePeriodTo,
			),
		)

	s.Configure(usagebased.StatusActiveFinalRealizationStarted).
		Permit(
			meta.TriggerNext,
			usagebased.StatusActiveFinalRealizationWaitingForCollection,
		).
		Permit(meta.TriggerDelete, usagebased.StatusDeleted).
		OnActive(
			s.StartFinalRealizationRun,
		)

	s.Configure(usagebased.StatusActiveFinalRealizationWaitingForCollection).
		Permit(
			meta.TriggerCollectionCompleted,
			usagebased.StatusActiveFinalRealizationProcessing,
		).
		Permit(meta.TriggerDelete, usagebased.StatusDeleted).
		OnActive(s.AdvanceAfterCollectionPeriodEnd)

	s.Configure(usagebased.StatusActiveFinalRealizationProcessing).
		Permit(
			meta.TriggerNext,
			usagebased.StatusActiveFinalRealizationCompleted,
		).
		Permit(meta.TriggerDelete, usagebased.StatusDeleted).
		OnActive(
			s.FinalizeRealizationRun,
		)

	s.Configure(usagebased.StatusActiveFinalRealizationCompleted).
		Permit(meta.TriggerDelete, usagebased.StatusDeleted)

	s.Configure(usagebased.StatusFinal).
		Permit(meta.TriggerDelete, usagebased.StatusDeleted)

	s.Configure(usagebased.StatusDeleted).
		OnEntry(statelessx.WithParameters(s.DeleteCharge))
}

func (s *CreditThenInvoice) DeleteCharge(ctx context.Context, _ meta.PatchDeletePolicy) error {
	if err := s.Adapter.DeleteCharge(ctx, s.Charge); err != nil {
		return fmt.Errorf("delete charge: %w", err)
	}

	return s.refetchCharge(ctx)
}

func (s *CreditThenInvoice) StartFinalRealizationRun(ctx context.Context) error {
	return s.Base.StartRealizationRun(ctx, StartRealizationRunInput{
		Type: usagebased.RealizationRunTypeFinalRealization,
	})
}
