package service

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	creditpurchaserealizations "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase/service/realizations"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/statelessx"
)

type InvoiceCreditPurchaseStateMachine struct {
	*stateMachine
}

func NewInvoiceCreditPurchaseStateMachine(config StateMachineConfig) (*InvoiceCreditPurchaseStateMachine, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	if config.Realizations == nil {
		return nil, errors.New("realizations service is required")
	}

	if config.Charge.Intent.Settlement.Type() != creditpurchase.SettlementTypeInvoice {
		return nil, fmt.Errorf("charge %s is not invoice settled", config.Charge.ID)
	}

	stateMachine, err := newStateMachineBase(config)
	if err != nil {
		return nil, fmt.Errorf("new invoice credit purchase state machine: %w", err)
	}

	out := &InvoiceCreditPurchaseStateMachine{
		stateMachine: stateMachine,
	}
	out.configureStates()

	return out, nil
}

func (s *InvoiceCreditPurchaseStateMachine) configureStates() {
	s.Configure(creditpurchase.StatusCreated).
		Permit(meta.TriggerInvoiceCreated, creditpurchase.StatusActiveInitialCreditGrant)

	s.Configure(creditpurchase.StatusActiveInitialCreditGrant).
		Permit(meta.TriggerNext, creditpurchase.StatusActivePaymentPending).
		OnActive(s.GrantCredits)

	s.Configure(creditpurchase.StatusActivePaymentPending).
		Permit(billing.TriggerAuthorized, creditpurchase.StatusActivePaymentAuthorized)

	s.Configure(creditpurchase.StatusActivePaymentAuthorized).
		Permit(billing.TriggerPaid, creditpurchase.StatusActivePaymentSettled).
		OnEntryFrom(
			billing.TriggerAuthorized,
			statelessx.WithParameters(s.AuthorizeInvoicedPayment),
		)

	s.Configure(creditpurchase.StatusActivePaymentSettled).
		Permit(meta.TriggerNext, creditpurchase.StatusFinal).
		OnActive(s.SettleInvoicedPayment)

	s.Configure(creditpurchase.StatusFinal)
}

func (s *InvoiceCreditPurchaseStateMachine) GrantCredits(ctx context.Context) error {
	updatedCharge, err := s.Realizations.GrantCredits(ctx, s.Charge)
	if err != nil {
		return err
	}

	s.Charge = updatedCharge
	return nil
}

func (s *InvoiceCreditPurchaseStateMachine) AuthorizeInvoicedPayment(ctx context.Context, lineWithHeader billing.StandardLineWithInvoiceHeader) error {
	updatedCharge, err := s.Realizations.AuthorizeInvoicedPayment(ctx, creditpurchaserealizations.AuthorizeInvoicedPaymentInput{
		Charge:         s.Charge,
		LineWithHeader: lineWithHeader,
	})
	if err != nil {
		return err
	}

	s.Charge = updatedCharge
	return nil
}

func (s *InvoiceCreditPurchaseStateMachine) SettleInvoicedPayment(ctx context.Context) error {
	updatedCharge, err := s.Realizations.SettleInvoicedPayment(ctx, s.Charge)
	if err != nil {
		return err
	}

	s.Charge = updatedCharge
	return nil
}

type HandleInvoiceLifecycleTriggerInput struct {
	Charge         creditpurchase.Charge
	Trigger        meta.Trigger
	LineWithHeader billing.StandardLineWithInvoiceHeader
}

var _ models.Validator = HandleInvoiceLifecycleTriggerInput{}

func (i HandleInvoiceLifecycleTriggerInput) Validate() error {
	var errs []error

	if err := i.Charge.GetChargeID().Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge ID: %w", err))
	}

	if i.Charge.Intent.Settlement.Type() != creditpurchase.SettlementTypeInvoice {
		errs = append(errs, fmt.Errorf("charge %s is not invoice settled", i.Charge.ID))
	}

	switch trigger := i.Trigger.(type) {
	case nil:
		errs = append(errs, errors.New("trigger is required"))
	case string:
		if !slices.ContainsFunc([]meta.Trigger{
			meta.TriggerInvoiceCreated,
			billing.TriggerAuthorized,
			billing.TriggerPaid,
		}, func(allowedTrigger meta.Trigger) bool {
			return allowedTrigger == trigger
		}) {
			errs = append(errs, fmt.Errorf("unsupported trigger %q", trigger))
		}
	default:
		errs = append(errs, fmt.Errorf("unsupported trigger %v", trigger))
	}

	if err := i.LineWithHeader.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("line with invoice header: %w", err))
	}

	line := i.LineWithHeader.Line
	if i.LineWithHeader.Invoice.Namespace != i.Charge.Namespace {
		errs = append(errs, errors.New("invoice namespace must match charge namespace"))
	}

	if line != nil {
		if line.ChargeID == nil || *line.ChargeID != i.Charge.ID {
			errs = append(errs, errors.New("line charge ID must match charge"))
		}

		if line.Namespace != i.Charge.Namespace {
			errs = append(errs, errors.New("line namespace must match charge namespace"))
		}

		if line.InvoiceID != i.LineWithHeader.Invoice.ID {
			errs = append(errs, errors.New("line invoice ID must match invoice"))
		}
	}

	if invoicedPayment := i.Charge.Realizations.InvoiceSettlement; invoicedPayment != nil {
		if line != nil && invoicedPayment.LineID != line.ID {
			errs = append(errs, errors.New("payment line ID must match line"))
		}

		if invoicedPayment.InvoiceID != i.LineWithHeader.Invoice.ID {
			errs = append(errs, errors.New("payment invoice ID must match invoice"))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (s *service) handleInvoiceLifecycleTrigger(ctx context.Context, input HandleInvoiceLifecycleTriggerInput) (creditpurchase.Charge, error) {
	if err := input.Validate(); err != nil {
		return creditpurchase.Charge{}, fmt.Errorf("validate invoice lifecycle trigger: %w", err)
	}

	stateMachine, err := NewInvoiceCreditPurchaseStateMachine(StateMachineConfig{
		Charge:       input.Charge,
		Adapter:      s.adapter,
		Realizations: s.realizations,
	})
	if err != nil {
		return creditpurchase.Charge{}, err
	}

	if err := stateMachine.AdvanceUntilStable(ctx); err != nil {
		return creditpurchase.Charge{}, fmt.Errorf("reconcile invoice credit purchase state: %w", err)
	}

	if err := stateMachine.FireAndAdvanceUntilStable(ctx, input.Trigger, input.LineWithHeader); err != nil {
		return creditpurchase.Charge{}, fmt.Errorf("fire invoice lifecycle trigger %s: %w", input.Trigger, err)
	}

	return stateMachine.GetCharge(), nil
}
