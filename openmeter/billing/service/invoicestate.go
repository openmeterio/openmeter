package billingservice

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/qmuntal/stateless"

	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	"github.com/openmeterio/openmeter/pkg/clock"
)

type InvoiceStateMachine struct {
	Invoice      billingentity.Invoice
	Calculator   InvoiceCalculator
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

	// TODO[later]: we should have a triggerAsyncNext to signify that a transition should be done asynchronously (
	// e.g. the invoice needs to be synced to an external system such as stripe)
)

var invoiceStateMachineCache = sync.Pool{
	New: func() interface{} {
		return allocateStateMachine()
	},
}

// TODO: this can panic let's validate that upon init somehow
func allocateStateMachine() *InvoiceStateMachine {
	out := &InvoiceStateMachine{}

	// TODO[later]: Tax is not captured here for now, as it would require the DB schema too
	// TODO[later]: Delete invoice is not implemented yet

	stateMachine := stateless.NewStateMachineWithExternalStorage(
		func(ctx context.Context) (stateless.State, error) {
			return out.Invoice.Status, nil
		},
		func(ctx context.Context, state stateless.State) error {
			invState, ok := state.(billingentity.InvoiceStatus)
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

	stateMachine.Configure(billingentity.InvoiceStatusDraftCreated).
		Permit(triggerNext, billingentity.InvoiceStatusDraftValidating).
		Permit(triggerUpdated, billingentity.InvoiceStatusDraftValidating)

	stateMachine.Configure(billingentity.InvoiceStatusDraftValidating).
		Permit(
			triggerNext,
			billingentity.InvoiceStatusDraftSyncing,
			boolFn(out.noCriticalValidationErrors),
		).
		Permit(triggerFailed, billingentity.InvoiceStatusDraftInvalid).
		// NOTE: we should permit update here, but stateless doesn't allow transitions to the same state
		Permit(triggerUpdated, billingentity.InvoiceStatusDraftCreated).
		OnActive(allOf(
			out.calculateInvoice,
			out.validateDraftInvoice),
		)

	stateMachine.Configure(billingentity.InvoiceStatusDraftInvalid).
		Permit(triggerRetry, billingentity.InvoiceStatusDraftValidating).
		Permit(triggerUpdated, billingentity.InvoiceStatusDraftValidating)

	stateMachine.Configure(billingentity.InvoiceStatusDraftSyncing).
		Permit(
			triggerNext,
			billingentity.InvoiceStatusDraftManualApprovalNeeded,
			boolFn(not(out.isAutoAdvanceEnabled)),
			boolFn(out.noCriticalValidationErrors),
		).
		Permit(
			triggerNext,
			billingentity.InvoiceStatusDraftWaitingAutoApproval,
			boolFn(out.isAutoAdvanceEnabled),
			boolFn(out.noCriticalValidationErrors),
		).
		Permit(triggerFailed, billingentity.InvoiceStatusDraftSyncFailed).
		OnActive(out.syncDraftInvoice)

	stateMachine.Configure(billingentity.InvoiceStatusDraftSyncFailed).
		Permit(triggerRetry, billingentity.InvoiceStatusDraftValidating).
		Permit(triggerUpdated, billingentity.InvoiceStatusDraftValidating)

	stateMachine.Configure(billingentity.InvoiceStatusDraftReadyToIssue).
		Permit(triggerNext, billingentity.InvoiceStatusIssuing).
		Permit(triggerUpdated, billingentity.InvoiceStatusDraftValidating)

	// Automatic and manual approvals
	stateMachine.Configure(billingentity.InvoiceStatusDraftWaitingAutoApproval).
		// Manual approval forces the draft invoice to be issued regardless of the review period
		Permit(triggerApprove, billingentity.InvoiceStatusDraftReadyToIssue).
		Permit(triggerUpdated, billingentity.InvoiceStatusDraftValidating).
		Permit(triggerNext,
			billingentity.InvoiceStatusDraftReadyToIssue,
			boolFn(out.shouldAutoAdvance),
			boolFn(out.noCriticalValidationErrors),
		)

	// This state is a pre-issuing state where we can halt the execution and execute issuing in the background
	// if needed
	stateMachine.Configure(billingentity.InvoiceStatusDraftManualApprovalNeeded).
		Permit(triggerApprove,
			billingentity.InvoiceStatusDraftReadyToIssue,
			boolFn(out.noCriticalValidationErrors),
		).
		Permit(triggerUpdated, billingentity.InvoiceStatusDraftValidating)

	// Issuing state

	stateMachine.Configure(billingentity.InvoiceStatusIssuing).
		Permit(triggerNext, billingentity.InvoiceStatusIssued).
		Permit(triggerFailed, billingentity.InvoiceStatusIssuingSyncFailed).
		OnActive(out.issueInvoice)

	stateMachine.Configure(billingentity.InvoiceStatusIssuingSyncFailed).
		Permit(triggerRetry, billingentity.InvoiceStatusIssuing)

	// Issued state (final)
	stateMachine.Configure(billingentity.InvoiceStatusIssued)

	out.StateMachine = stateMachine

	return out
}

type InvoiceStateMachineCallback func(context.Context, *InvoiceStateMachine) error

func WithInvoiceStateMachine(ctx context.Context, invoice billingentity.Invoice, calc InvoiceCalculator, cb InvoiceStateMachineCallback) (billingentity.Invoice, error) {
	sm := invoiceStateMachineCache.Get().(*InvoiceStateMachine)

	// Stateless doesn't store any state in the state machine, so it's fine to reuse the state machine itself
	sm.Invoice = invoice
	sm.Calculator = calc

	defer func() {
		sm.Invoice = billingentity.Invoice{}
		sm.Calculator = nil
		invoiceStateMachineCache.Put(sm)
	}()

	if err := cb(ctx, sm); err != nil {
		return billingentity.Invoice{}, err
	}

	return sm.Invoice, nil
}

func (m *InvoiceStateMachine) StatusDetails(ctx context.Context) (billingentity.InvoiceStatusDetails, error) {
	if m.Invoice.Status == billingentity.InvoiceStatusGathering {
		// Gathering is a special state that is not part of the state machine, due to
		// cross invoice operations
		return billingentity.InvoiceStatusDetails{
			Immutable: false,
		}, nil
	}

	var outErr error
	actions := make([]billingentity.InvoiceAction, 0, 4)

	ok, err := m.StateMachine.CanFireCtx(ctx, triggerNext)
	if err != nil {
		outErr = errors.Join(outErr, err)
	} else if ok {
		actions = append(actions, billingentity.InvoiceActionAdvance)
	}

	ok, err = m.StateMachine.CanFireCtx(ctx, triggerRetry)
	if err != nil {
		outErr = errors.Join(outErr, err)
	} else if ok {
		actions = append(actions, billingentity.InvoiceActionRetry)
	}

	ok, err = m.StateMachine.CanFireCtx(ctx, triggerApprove)
	if err != nil {
		outErr = errors.Join(outErr, err)
	} else if ok {
		actions = append(actions, billingentity.InvoiceActionApprove)
	}

	mutable, err := m.StateMachine.CanFireCtx(ctx, triggerUpdated)
	if err != nil {
		outErr = errors.Join(outErr, err)
	}

	// TODO[later]: add more actions (void, delete, etc.)

	return billingentity.InvoiceStatusDetails{
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
			return nil
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

		if err := m.StateMachine.ActivateCtx(ctx); err != nil {
			return fmt.Errorf("failed to activate failed state: %w", err)
		}

		return activationError
	}

	return nil
}

// validateDraftInvoice validates the draft invoice using the apps referenced in the invoice.
func (m *InvoiceStateMachine) validateDraftInvoice(ctx context.Context) error {
	invocingBase := m.Invoice.Workflow.Apps.Invoicing
	invoicingApp, ok := invocingBase.(billingentity.InvoicingApp)
	if !ok {
		// If this happens we are rolling back the state transition (as we are not wrapping this into a validation issue)
		return fmt.Errorf("app [type=%s, id=%s] does not implement the invoicing interface",
			m.Invoice.Workflow.Apps.Invoicing.GetType(),
			m.Invoice.Workflow.Apps.Invoicing.GetID().ID)
	}

	component := billingentity.AppTypeCapabilityToComponent(invocingBase.GetType(), appentitybase.CapabilityTypeInvoiceCustomers)

	// Anything returned by the validation is considered a validation issue, thus in case of an error
	// we wouldn't roll back the state transitions.
	return m.Invoice.MergeValidationIssues(
		billingentity.ValidationWithComponent(
			component,
			invoicingApp.ValidateInvoice(ctx, m.Invoice),
		),
		component,
	)
}

func (m *InvoiceStateMachine) calculateInvoice(ctx context.Context) error {
	return m.Calculator.Calculate(&m.Invoice)
}

// syncDraftInvoice syncs the draft invoice with the external system.
func (m *InvoiceStateMachine) syncDraftInvoice(ctx context.Context) error {
	return nil
}

// issueInvoice issues the invoice using the invoicing app
func (m *InvoiceStateMachine) issueInvoice(ctx context.Context) error {
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
