package billingservice

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/qmuntal/stateless"
	"github.com/samber/lo"

	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/service/invoicecalc"
	"github.com/openmeterio/openmeter/pkg/clock"
)

type InvoiceStateMachine struct {
	Invoice      billing.Invoice
	Calculator   invoicecalc.Calculator
	StateMachine *stateless.StateMachine
}

var (
	// triggerRetry is used to retry a state transition that failed, used by the end user to invoke it manually
	triggerRetry stateless.Trigger = "trigger_retry"
	// triggerApprove is used to approve a state manually
	triggerApprove stateless.Trigger = "trigger_approve"
	// triggerNext is used to advance the invoice to the next state if automatically possible
	triggerNext stateless.Trigger = "trigger_next"
	// triggerFailed is used to trigger the failure state transition associated with the current state
	triggerFailed stateless.Trigger = "trigger_failed"
	// triggerUpdated is used to trigger a change in the invoice (we are using this to calculate the immutable states
	// and trigger re-validation)
	triggerUpdated stateless.Trigger = "trigger_updated"
	// triggerDelete is used to delete the invoice
	triggerDelete stateless.Trigger = "trigger_delete"

	// TODO[OM-989]: we should have a triggerAsyncNext to signify that a transition should be done asynchronously (
	// e.g. the invoice needs to be synced to an external system such as stripe)
)

const (
	opValidate = "validate"
	opSync     = "sync"
	opDelete   = "delete"
	opFinalize = "finalize"
)

var invoiceStateMachineCache = sync.Pool{
	New: func() interface{} {
		return allocateStateMachine()
	},
}

// TODO[OM-990]: this can panic let's validate that upon init somehow
func allocateStateMachine() *InvoiceStateMachine {
	out := &InvoiceStateMachine{}

	// TODO[OM-979]: Tax is not captured here for now, as it would require the DB schema too
	// TODO[OM-988]: Delete invoice is not implemented yet

	stateMachine := stateless.NewStateMachineWithExternalStorage(
		func(ctx context.Context) (stateless.State, error) {
			return out.Invoice.Status, nil
		},
		func(ctx context.Context, state stateless.State) error {
			invState, ok := state.(billing.InvoiceStatus)
			if !ok {
				return fmt.Errorf("invalid state type: %v", state)
			}

			out.Invoice.Status = invState

			sd, err := out.StatusDetails(ctx)
			if err != nil {
				return err
			}

			out.Invoice.StatusDetails = sd

			return nil
		},
		stateless.FiringImmediate,
	)

	// Draft states

	// NOTE: we are not using the substate support of stateless for now, as the
	// substate inherits all the parent's state transitions resulting in unexpected behavior (
	// e.g. allowing triggerNext on the "superstate" causes all substates to have triggerNext).

	stateMachine.Configure(billing.InvoiceStatusDraftCreated).
		Permit(triggerNext, billing.InvoiceStatusDraftValidating).
		Permit(triggerDelete, billing.InvoiceStatusDeleteInProgress).
		Permit(triggerUpdated, billing.InvoiceStatusDraftUpdating).
		OnActive(out.calculateInvoice)

	stateMachine.Configure(billing.InvoiceStatusDraftUpdating).
		Permit(triggerNext, billing.InvoiceStatusDraftValidating).
		Permit(triggerDelete, billing.InvoiceStatusDeleteInProgress).
		OnActive(
			allOf(
				out.calculateInvoice,
				out.validateDraftInvoice,
			),
		)

	stateMachine.Configure(billing.InvoiceStatusDraftValidating).
		Permit(
			triggerNext,
			billing.InvoiceStatusDraftSyncing,
			boolFn(out.noCriticalValidationErrors),
		).
		Permit(triggerFailed, billing.InvoiceStatusDraftInvalid).
		Permit(triggerDelete, billing.InvoiceStatusDeleteInProgress).
		// NOTE: we should permit update here, but stateless doesn't allow transitions to the same state
		Permit(triggerUpdated, billing.InvoiceStatusDraftUpdating).
		OnActive(allOf(
			out.calculateInvoice,
			out.validateDraftInvoice,
		))

	stateMachine.Configure(billing.InvoiceStatusDraftInvalid).
		Permit(triggerRetry, billing.InvoiceStatusDraftValidating).
		Permit(triggerDelete, billing.InvoiceStatusDeleteInProgress).
		Permit(triggerUpdated, billing.InvoiceStatusDraftUpdating)

	stateMachine.Configure(billing.InvoiceStatusDraftSyncing).
		Permit(
			triggerNext,
			billing.InvoiceStatusDraftManualApprovalNeeded,
			boolFn(not(out.isAutoAdvanceEnabled)),
			boolFn(out.noCriticalValidationErrors),
		).
		Permit(
			triggerNext,
			billing.InvoiceStatusDraftWaitingAutoApproval,
			boolFn(out.isAutoAdvanceEnabled),
			boolFn(out.noCriticalValidationErrors),
		).
		Permit(triggerDelete, billing.InvoiceStatusDeleteInProgress).
		Permit(triggerFailed, billing.InvoiceStatusDraftSyncFailed).
		OnActive(out.syncDraftInvoice)

	stateMachine.Configure(billing.InvoiceStatusDraftSyncFailed).
		Permit(triggerRetry, billing.InvoiceStatusDraftValidating).
		Permit(triggerDelete, billing.InvoiceStatusDeleteInProgress).
		Permit(triggerUpdated, billing.InvoiceStatusDraftUpdating)

	stateMachine.Configure(billing.InvoiceStatusDraftReadyToIssue).
		Permit(triggerNext, billing.InvoiceStatusIssuing).
		Permit(triggerDelete, billing.InvoiceStatusDeleteInProgress).
		Permit(triggerUpdated, billing.InvoiceStatusDraftUpdating)

	// Automatic and manual approvals
	stateMachine.Configure(billing.InvoiceStatusDraftWaitingAutoApproval).
		// Manual approval forces the draft invoice to be issued regardless of the review period
		Permit(triggerApprove, billing.InvoiceStatusDraftReadyToIssue).
		Permit(triggerUpdated, billing.InvoiceStatusDraftUpdating).
		Permit(triggerDelete, billing.InvoiceStatusDeleteInProgress).
		Permit(triggerNext,
			billing.InvoiceStatusDraftReadyToIssue,
			boolFn(out.shouldAutoAdvance),
			boolFn(out.noCriticalValidationErrors),
		)

	// This state is a pre-issuing state where we can halt the execution and execute issuing in the background
	// if needed
	stateMachine.Configure(billing.InvoiceStatusDraftManualApprovalNeeded).
		Permit(triggerApprove,
			billing.InvoiceStatusDraftReadyToIssue,
			boolFn(out.noCriticalValidationErrors),
		).
		Permit(triggerUpdated, billing.InvoiceStatusDraftUpdating)

	// Deletion state
	stateMachine.Configure(billing.InvoiceStatusDeleteInProgress).
		Permit(triggerNext, billing.InvoiceStatusDeleteSyncing).
		Permit(triggerFailed, billing.InvoiceStatusDeleteFailed).
		OnActive(out.deleteInvoice)

	stateMachine.Configure(billing.InvoiceStatusDeleteSyncing).
		Permit(triggerNext, billing.InvoiceStatusDeleted).
		Permit(triggerFailed, billing.InvoiceStatusDeleteFailed).
		OnActive(out.syncDeletedInvoice)

	stateMachine.Configure(billing.InvoiceStatusDeleteFailed).
		Permit(triggerRetry, billing.InvoiceStatusDeleteInProgress)

	stateMachine.Configure(billing.InvoiceStatusDeleted)

	// Issuing state

	stateMachine.Configure(billing.InvoiceStatusIssuing).
		Permit(triggerNext, billing.InvoiceStatusIssued).
		Permit(triggerFailed, billing.InvoiceStatusIssuingSyncFailed).
		Permit(triggerDelete, billing.InvoiceStatusDeleteInProgress).
		OnActive(out.finalizeInvoice)

	stateMachine.Configure(billing.InvoiceStatusIssuingSyncFailed).
		Permit(triggerDelete, billing.InvoiceStatusDeleteInProgress).
		Permit(triggerRetry, billing.InvoiceStatusIssuing)

	// Issued state (final)
	stateMachine.Configure(billing.InvoiceStatusIssued)

	out.StateMachine = stateMachine

	return out
}

type InvoiceStateMachineCallback func(context.Context, *InvoiceStateMachine) error

func (s *Service) WithInvoiceStateMachine(ctx context.Context, invoice billing.Invoice, cb InvoiceStateMachineCallback) (billing.Invoice, error) {
	sm := invoiceStateMachineCache.Get().(*InvoiceStateMachine)

	// Stateless doesn't store any state in the state machine, so it's fine to reuse the state machine itself
	sm.Invoice = invoice
	sm.Calculator = s.invoiceCalculator

	defer func() {
		sm.Invoice = billing.Invoice{}
		sm.Calculator = nil
		invoiceStateMachineCache.Put(sm)
	}()

	if err := cb(ctx, sm); err != nil {
		return billing.Invoice{}, err
	}

	sd, err := sm.StatusDetails(ctx)
	if err != nil {
		return sm.Invoice, fmt.Errorf("error resolving status details: %w", err)
	}

	sm.Invoice.StatusDetails = sd

	return sm.Invoice, nil
}

func (m *InvoiceStateMachine) StatusDetails(ctx context.Context) (billing.InvoiceStatusDetails, error) {
	if m.Invoice.Status == billing.InvoiceStatusGathering {
		// Gathering is a special state that is not part of the state machine, due to
		// cross invoice operations
		return billing.InvoiceStatusDetails{
			Immutable: false,
		}, nil
	}

	var outErr error
	actions := make([]billing.InvoiceAction, 0, 4)

	ok, err := m.StateMachine.CanFireCtx(ctx, triggerNext)
	if err != nil {
		outErr = errors.Join(outErr, err)
	} else if ok {
		actions = append(actions, billing.InvoiceActionAdvance)
	}

	ok, err = m.StateMachine.CanFireCtx(ctx, triggerRetry)
	if err != nil {
		outErr = errors.Join(outErr, err)
	} else if ok {
		actions = append(actions, billing.InvoiceActionRetry)
	}

	ok, err = m.StateMachine.CanFireCtx(ctx, triggerApprove)
	if err != nil {
		outErr = errors.Join(outErr, err)
	} else if ok {
		actions = append(actions, billing.InvoiceActionApprove)
	}

	mutable, err := m.StateMachine.CanFireCtx(ctx, triggerUpdated)
	if err != nil {
		outErr = errors.Join(outErr, err)
	}

	// TODO[OM-988]: add more actions (void, delete, etc.)

	return billing.InvoiceStatusDetails{
		Immutable:        !mutable,
		Failed:           m.Invoice.Status.IsFailed(),
		AvailableActions: actions,
	}, outErr
}

func (m *InvoiceStateMachine) AdvanceUntilStateStable(ctx context.Context) error {
	for {
		canFire, err := m.StateMachine.CanFireCtx(ctx, triggerNext)
		if err != nil {
			return err
		}

		// We have reached a state that requires either manual intervention or that is final
		if !canFire {
			return m.Invoice.ValidationIssues.AsError()
		}

		if err := m.FireAndActivate(ctx, triggerNext); err != nil {
			return fmt.Errorf("cannot transition to the next status [current_status=%s]: %w", m.Invoice.Status, err)
		}
	}
}

func (m *InvoiceStateMachine) CanFire(ctx context.Context, trigger stateless.Trigger) (bool, error) {
	return m.StateMachine.CanFireCtx(ctx, trigger)
}

// FireAndActivate fires the trigger and activates the new state, if activation fails it automatically
// transitions to the failed state and activates that.
// In addition to the activation a calculation is always performed to ensure that the invoice is up to date.
func (m *InvoiceStateMachine) FireAndActivate(ctx context.Context, trigger stateless.Trigger) error {
	if err := m.StateMachine.FireCtx(ctx, trigger); err != nil {
		return err
	}

	activationError := m.StateMachine.ActivateCtx(ctx)
	if activationError != nil || m.Invoice.HasCriticalValidationIssues() {
		validationIssues := m.Invoice.ValidationIssues.Clone()

		// There was an error activating the state, we should trigger a transition to the failed state
		canFire, err := m.StateMachine.CanFireCtx(ctx, triggerFailed)
		if err != nil {
			return fmt.Errorf("failed to check if we can transition to failed state: %w", err)
		}

		if !canFire {
			return fmt.Errorf("cannot move into failed state: %w", activationError)
		}

		if err := m.StateMachine.FireCtx(ctx, triggerFailed); err != nil {
			return fmt.Errorf("failed to transition to failed state: %w", err)
		}

		if activationError != nil {
			return activationError
		}

		return validationIssues.AsError()
	}

	return nil
}

func (m *InvoiceStateMachine) withInvoicingApp(op string, cb func(billing.InvoicingApp) error) error {
	invocingBase := m.Invoice.Workflow.Apps.Invoicing
	invoicingApp, ok := invocingBase.(billing.InvoicingApp)
	if !ok {
		// If this happens we are rolling back the state transition (as we are not wrapping this into a validation issue)
		return fmt.Errorf("app [type=%s, id=%s] does not implement the invoicing interface",
			m.Invoice.Workflow.Apps.Invoicing.GetType(),
			m.Invoice.Workflow.Apps.Invoicing.GetID().ID)
	}

	component := billing.AppTypeCapabilityToComponent(invocingBase.GetType(), appentitybase.CapabilityTypeInvoiceCustomers, op)
	result := cb(invoicingApp)

	// Anything returned by the validation is considered a validation issue, thus in case of an error
	// we wouldn't roll back the state transitions.
	return m.Invoice.MergeValidationIssues(
		billing.ValidationWithComponent(
			component,
			result,
		),
		component,
	)
}

func (m *InvoiceStateMachine) mergeUpsertInvoiceResult(result *billing.UpsertInvoiceResult) error {
	return billing.MergeUpsertInvoiceResult(&m.Invoice, result)
}

// validateDraftInvoice validates the draft invoice using the apps referenced in the invoice.
func (m *InvoiceStateMachine) validateDraftInvoice(ctx context.Context) error {
	return m.withInvoicingApp(opValidate, func(app billing.InvoicingApp) error {
		return app.ValidateInvoice(ctx, m.Invoice.Clone())
	})
}

func (m *InvoiceStateMachine) calculateInvoice(ctx context.Context) error {
	return m.Calculator.Calculate(&m.Invoice)
}

// syncDraftInvoice syncs the draft invoice with the external system.
func (m *InvoiceStateMachine) syncDraftInvoice(ctx context.Context) error {
	return m.withInvoicingApp(opSync, func(app billing.InvoicingApp) error {
		results, err := app.UpsertInvoice(ctx, m.Invoice.Clone())
		if err != nil {
			return err
		}

		if results == nil {
			return nil
		}

		return m.mergeUpsertInvoiceResult(results)
	})
}

// finalizeInvoice finalizes the invoice using the invoicing app and payment app (later).
func (m *InvoiceStateMachine) finalizeInvoice(ctx context.Context) error {
	return m.withInvoicingApp(opFinalize, func(app billing.InvoicingApp) error {
		clonedInvoice := m.Invoice.Clone()
		// First we sync the invoice
		upsertResults, err := app.UpsertInvoice(ctx, clonedInvoice)
		if err != nil {
			return err
		}

		if upsertResults != nil {
			if err := m.mergeUpsertInvoiceResult(upsertResults); err != nil {
				return err
			}
		}

		results, err := app.FinalizeInvoice(ctx, clonedInvoice)
		if err != nil {
			return err
		}

		if results != nil {
			if paymentExternalID, ok := results.GetPaymentExternalID(); ok {
				m.Invoice.ExternalIDs.Payment = paymentExternalID
			}
		}

		return nil
	})
}

// syncDeletedInvoice syncs the deleted invoice with the external system
func (m *InvoiceStateMachine) syncDeletedInvoice(ctx context.Context) error {
	return m.withInvoicingApp(opDelete, func(app billing.InvoicingApp) error {
		return app.DeleteInvoice(ctx, m.Invoice.Clone())
	})
}

// deleteInvoice deletes the invoice
func (m *InvoiceStateMachine) deleteInvoice(ctx context.Context) error {
	m.Invoice.DeletedAt = lo.ToPtr(clock.Now().In(time.UTC))

	return nil
}

func (m *InvoiceStateMachine) noCriticalValidationErrors() bool {
	return !m.Invoice.HasCriticalValidationIssues()
}

func (m *InvoiceStateMachine) isAutoAdvanceEnabled() bool {
	return m.Invoice.Workflow.Config.Invoicing.AutoAdvance
}

func (m *InvoiceStateMachine) shouldAutoAdvance() bool {
	if !m.isAutoAdvanceEnabled() || m.Invoice.DraftUntil == nil {
		return false
	}

	return !clock.Now().In(time.UTC).Before(*m.Invoice.DraftUntil)
}

func boolFn(fn func() bool) func(context.Context, ...any) bool {
	return func(context.Context, ...any) bool {
		return fn()
	}
}

func not(fn func() bool) func() bool {
	return func() bool {
		return !fn()
	}
}

type actionFn func(context.Context) error

// allOf chains multiple action functions into a single action function, all functions
// will be called, regardless of their error state.
// The reported errors will be joined into a single error object.
func allOf(fn ...actionFn) actionFn {
	return func(ctx context.Context) error {
		var outErr error

		for _, f := range fn {
			if err := f(ctx); err != nil {
				outErr = errors.Join(outErr, err)
			}
		}

		return outErr
	}
}
