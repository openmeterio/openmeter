package billingservice

import (
	"context"
	"fmt"
	"time"

	"github.com/qmuntal/stateless"

	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	"github.com/openmeterio/openmeter/pkg/clock"
)

type InvoiceStateMachine struct {
	Invoice *billingentity.Invoice
	FSM     *stateless.StateMachine
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

	// TODO[later]: we should have a triggerAsyncNext to signify that a transition should be done asynchronously (
	// e.g. the invoice needs to be synced to an external system such as stripe)
)

func NewInvoiceStateMachine(invoice *billingentity.Invoice) *InvoiceStateMachine {
	out := &InvoiceStateMachine{
		Invoice: invoice,
	}

	// TODO[later]: Tax is not captured here for now, as it would require the DB schema too
	// TODO[later]: Delete invoice is not implemented yet
	// TODO[optimization]: The state machine can be added to sync.Pool to avoid allocations (state is stored in the Invoice entity)

	fsm := stateless.NewStateMachineWithExternalStorage(
		func(ctx context.Context) (stateless.State, error) {
			return out.Invoice.Status, nil
		},
		func(ctx context.Context, state stateless.State) error {
			invState, ok := state.(billingentity.InvoiceStatus)
			if !ok {
				return fmt.Errorf("invalid state type: %v", state)
			}

			out.Invoice.Status = invState
			out.Invoice.StatusDetails = out.StatusDetails(ctx)
			return nil
		},
		stateless.FiringImmediate,
	)

	// Draft states

	// NOTE: we are not using the substate support of stateless for now, as the
	// substate inherits all the parent's state transitions resulting in unexpected behavior.

	fsm.Configure(billingentity.InvoiceStatusDraftCreated).
		Permit(triggerNext, billingentity.InvoiceStatusDraftValidating)

	fsm.Configure(billingentity.InvoiceStatusDraftValidating).
		Permit(triggerNext, billingentity.InvoiceStatusDraftSyncing).
		Permit(triggerFailed, billingentity.InvoiceStatusDraftInvalid).
		OnActive(out.validateDraftInvoice)

	fsm.Configure(billingentity.InvoiceStatusDraftInvalid).
		Permit(triggerRetry, billingentity.InvoiceStatusDraftValidating)

	fsm.Configure(billingentity.InvoiceStatusDraftSyncing).
		Permit(triggerNext,
			billingentity.InvoiceStatusDraftManualApprovalNeeded,
			boolFn(not(out.isAutoAdvanceEnabled))).
		Permit(triggerNext,
			billingentity.InvoiceStatusDraftWaitingAutoApproval,
			boolFn(out.isAutoAdvanceEnabled)).
		Permit(triggerFailed, billingentity.InvoiceStatusDraftSyncFailed).
		OnActive(out.syncDraftInvoice)

	fsm.Configure(billingentity.InvoiceStatusDraftSyncFailed).
		Permit(triggerRetry, billingentity.InvoiceStatusDraftValidating)

	fsm.Configure(billingentity.InvoiceStatusDraftReadyToIssue).
		Permit(triggerNext, billingentity.InvoiceStatusIssuing)

	// Automatic and manual approvals
	fsm.Configure(billingentity.InvoiceStatusDraftWaitingAutoApproval).
		// Manual approval forces the draft invoice to be issued regardless of the review period
		Permit(triggerApprove, billingentity.InvoiceStatusDraftReadyToIssue).
		Permit(triggerNext,
			billingentity.InvoiceStatusDraftReadyToIssue,
			boolFn(out.shouldAutoAdvance),
		)

	// This state is a pre-issuing state where we can halt the execution and execute issuing in the background
	// if needed
	fsm.Configure(billingentity.InvoiceStatusDraftManualApprovalNeeded).
		Permit(triggerApprove, billingentity.InvoiceStatusDraftReadyToIssue)

	// Issuing state

	fsm.Configure(billingentity.InvoiceStatusIssuing).
		Permit(triggerNext, billingentity.InvoiceStatusIssued).
		Permit(triggerFailed, billingentity.InvoiceStatusIssuingSyncFailed).
		OnActive(out.issueInvoice)

	fsm.Configure(billingentity.InvoiceStatusIssuingSyncFailed).
		Permit(triggerRetry, billingentity.InvoiceStatusIssuing)

	// Issued state (final)
	fsm.Configure(billingentity.InvoiceStatusIssued)

	out.FSM = fsm

	return out
}

func (m *InvoiceStateMachine) StatusDetails(ctx context.Context) billingentity.InvoiceStatusDetails {
	actions := make([]billingentity.InvoiceAction, 0, 4)

	if ok, err := m.FSM.CanFireCtx(ctx, triggerNext); err == nil && ok {
		actions = append(actions, billingentity.InvoiceActionAdvance)
	}

	if ok, err := m.FSM.CanFireCtx(ctx, triggerRetry); err == nil && ok {
		actions = append(actions, billingentity.InvoiceActionRetry)
	}

	if ok, err := m.FSM.CanFireCtx(ctx, triggerApprove); err == nil && ok {
		actions = append(actions, billingentity.InvoiceActionApprove)
	}

	// TODO[later]: add more actions (void, delete, etc.)

	return billingentity.InvoiceStatusDetails{
		Immutable:        !m.Invoice.Status.IsMutable(),
		Failed:           m.Invoice.Status.IsFailed(),
		AvailableActions: actions,
	}
}

func (m *InvoiceStateMachine) ActivateUntilStateStable(ctx context.Context) error {
	for {
		canFire, err := m.FSM.CanFireCtx(ctx, triggerNext)
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
	return m.FSM.CanFireCtx(ctx, trigger)
}

// FireAndActivate fires the trigger and activates the new state, if activation fails it automatically
// transitions to the failed state and activates that.
func (m *InvoiceStateMachine) FireAndActivate(ctx context.Context, trigger stateless.Trigger) error {
	if err := m.FSM.FireCtx(ctx, trigger); err != nil {
		return err
	}

	err := m.FSM.ActivateCtx(ctx)
	if err != nil {
		// There was an error activating the state, we should trigger a transition to the failed state
		activationError := err

		// TODO[later]: depending on the final implementation, we might want to make this a special error
		// that signals that the invoice is in an inconsistent state
		if err := m.FSM.FireCtx(ctx, triggerFailed); err != nil {
			return fmt.Errorf("failed to transition to failed state: %w", err)
		}

		if err := m.FSM.ActivateCtx(ctx); err != nil {
			return fmt.Errorf("failed to activate failed state: %w", err)
		}

		return activationError
	}

	return nil
}

// validateDraftInvoice validates the draft invoice using the apps referenced in the invoice.
func (m *InvoiceStateMachine) validateDraftInvoice(ctx context.Context) error {
	return nil
}

// syncDraftInvoice syncs the draft invoice with the external system.
func (m *InvoiceStateMachine) syncDraftInvoice(ctx context.Context) error {
	return nil
}

// issueInvoice issues the invoice using the invoicing app
func (m *InvoiceStateMachine) issueInvoice(ctx context.Context) error {
	return nil
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
