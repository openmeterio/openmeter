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
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
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

	// TODO: Migrate existing invoice credit purchases from the legacy active status
	// to their detailed status with an SQL migration, then remove this runtime mapping.
	mappedCharge, err := mapLegacyInvoiceCreditPurchaseStatus(config.Charge)
	if err != nil {
		return nil, fmt.Errorf("map legacy invoice credit purchase status: %w", err)
	}
	config.Charge = mappedCharge

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

// mapLegacyInvoiceCreditPurchaseStatus maps the generic active status used before
// detailed invoice states from durable realization facts. The mapping is in-memory
// so the next real lifecycle event persists its target without a synthetic transition.
func mapLegacyInvoiceCreditPurchaseStatus(charge creditpurchase.Charge) (creditpurchase.Charge, error) {
	if charge.Status != creditpurchase.StatusActive {
		return charge, nil
	}

	creditGrant := charge.Realizations.CreditGrantRealization
	if creditGrant == nil || creditGrant.TransactionGroupID == "" {
		return creditpurchase.Charge{}, models.NewGenericPreConditionFailedError(
			fmt.Errorf("legacy active invoice credit purchase has no credit grant [charge_id=%s]", charge.ID),
		)
	}

	invoicedPayment := charge.Realizations.InvoiceSettlement
	if invoicedPayment == nil {
		return charge.WithStatus(creditpurchase.StatusActivePaymentPending), nil
	}

	switch invoicedPayment.Status {
	case payment.StatusAuthorized:
		return charge.WithStatus(creditpurchase.StatusActivePaymentAuthorized), nil
	case payment.StatusSettled:
		return charge.WithStatus(creditpurchase.StatusFinal), nil
	default:
		return creditpurchase.Charge{}, fmt.Errorf(
			"unsupported invoiced payment status %q [charge_id=%s]",
			invoicedPayment.Status,
			charge.ID,
		)
	}
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

	if _, err := stateMachine.AdvanceUntilStateStable(ctx); err != nil {
		return creditpurchase.Charge{}, fmt.Errorf("reconcile invoice credit purchase state: %w", err)
	}

	if err := stateMachine.FireAndActivate(ctx, input.Trigger, input.LineWithHeader); err != nil {
		return creditpurchase.Charge{}, fmt.Errorf("fire invoice lifecycle trigger %s: %w", input.Trigger, err)
	}

	advancedCharge, err := stateMachine.AdvanceUntilStateStable(ctx)
	if err != nil {
		return creditpurchase.Charge{}, fmt.Errorf("advance invoice credit purchase after %s: %w", input.Trigger, err)
	}

	if advancedCharge != nil {
		return *advancedCharge, nil
	}

	return stateMachine.GetCharge(), nil
}

// TODO: Move these invoice lifecycle hook adapters to the credit-purchase line engine
// and remove the service entry points when the legacy standard invoice hook routing is retired.
func (s *service) PostInvoiceDraftCreated(ctx context.Context, charge creditpurchase.Charge, lineWithHeader billing.StandardLineWithInvoiceHeader) error {
	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		_, err := s.handleInvoiceLifecycleTrigger(ctx, HandleInvoiceLifecycleTriggerInput{
			Charge:         charge,
			Trigger:        meta.TriggerInvoiceCreated,
			LineWithHeader: lineWithHeader,
		})
		return err
	})
}

// PostInvoicePaymentAuthorized is called when billing has authorized payment.
// Billing invokes the hook inside the invoice lifecycle transaction.
func (s *service) PostInvoicePaymentAuthorized(ctx context.Context, charge creditpurchase.Charge, lineWithHeader billing.StandardLineWithInvoiceHeader) error {
	_, err := s.handleInvoiceLifecycleTrigger(ctx, HandleInvoiceLifecycleTriggerInput{
		Charge:         charge,
		Trigger:        billing.TriggerAuthorized,
		LineWithHeader: lineWithHeader,
	})
	return err
}

// PostInvoicePaymentSettled is called when billing has settled payment.
// Billing invokes the hook inside the invoice lifecycle transaction.
func (s *service) PostInvoicePaymentSettled(ctx context.Context, charge creditpurchase.Charge, lineWithHeader billing.StandardLineWithInvoiceHeader) error {
	_, err := s.handleInvoiceLifecycleTrigger(ctx, HandleInvoiceLifecycleTriggerInput{
		Charge:         charge,
		Trigger:        billing.TriggerPaid,
		LineWithHeader: lineWithHeader,
	})
	return err
}
