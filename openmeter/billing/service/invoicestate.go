package billingservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/qmuntal/stateless"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/service/invoicecalc"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/clock"
)

type InvoiceStateMachine struct {
	Invoice             billing.Invoice
	Calculator          invoicecalc.Calculator
	StateMachine        *stateless.StateMachine
	Logger              *slog.Logger
	Publisher           eventbus.Publisher
	Service             *Service
	FSNamespaceLockdown []string
}

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

			previousStatus := out.Invoice.Status
			out.Invoice.Status = invState

			if invState == billing.InvoiceStatusPaymentProcessingPending &&
				previousStatus != billing.InvoiceStatusPaymentProcessingPending &&
				out.Invoice.PaymentProcessingEnteredAt == nil {
				now := clock.Now().UTC()
				out.Invoice.PaymentProcessingEnteredAt = &now
			}

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
	// e.g. allowing billing.TriggerNext on the "superstate" causes all substates to have billing.TriggerNext).

	stateMachine.Configure(billing.InvoiceStatusDraftCreated).
		Permit(billing.TriggerNext, billing.InvoiceStatusDraftWaitingForCollection).
		Permit(billing.TriggerFailed, billing.InvoiceStatusDraftInvalid).
		Permit(billing.TriggerDelete, billing.InvoiceStatusDeleteInProgress).
		Permit(billing.TriggerUpdated, billing.InvoiceStatusDraftUpdating).
		OnActive(out.calculateInvoice)

	stateMachine.Configure(billing.InvoiceStatusDraftWaitingForCollection).
		Permit(
			billing.TriggerNext,
			billing.InvoiceStatusDraftCollecting,
			boolFn(out.isReadyForCollection),
		).
		Permit(billing.TriggerDelete, billing.InvoiceStatusDeleteInProgress).
		Permit(billing.TriggerUpdated, billing.InvoiceStatusDraftUpdating).
		Permit(billing.TriggerSnapshotQuantities, billing.InvoiceStatusDraftCollecting)

	stateMachine.Configure(billing.InvoiceStatusDraftCollecting).
		Permit(billing.TriggerNext, billing.InvoiceStatusDraftValidating).
		Permit(billing.TriggerDelete, billing.InvoiceStatusDeleteInProgress).
		Permit(billing.TriggerFailed, billing.InvoiceStatusDraftInvalid).
		Permit(billing.TriggerUpdated, billing.InvoiceStatusDraftUpdating).
		OnActive(
			allOf(
				out.snapshotQuantityAsNeeded,
				out.calculateInvoice,
			),
		)

	// Invoice is edited
	stateMachine.Configure(billing.InvoiceStatusDraftUpdating).
		Permit(billing.TriggerNext, billing.InvoiceStatusDraftWaitingForCollection).
		Permit(billing.TriggerDelete, billing.InvoiceStatusDeleteInProgress).
		Permit(billing.TriggerFailed, billing.InvoiceStatusDraftInvalid).
		OnActive(
			allOf(
				out.calculateInvoice,
				out.validateDraftInvoice,
			),
		)

	stateMachine.Configure(billing.InvoiceStatusDraftValidating).
		Permit(
			billing.TriggerNext,
			billing.InvoiceStatusDraftSyncing,
			boolFn(out.noCriticalValidationErrors),
		).
		Permit(billing.TriggerFailed, billing.InvoiceStatusDraftInvalid).
		Permit(billing.TriggerDelete, billing.InvoiceStatusDeleteInProgress).
		// NOTE: we should permit update here, but stateless doesn't allow transitions to the same state
		Permit(billing.TriggerUpdated, billing.InvoiceStatusDraftUpdating).
		OnActive(allOf(
			out.calculateInvoice,
			out.validateDraftInvoice,
		))

	stateMachine.Configure(billing.InvoiceStatusDraftInvalid).
		Permit(billing.TriggerRetry, billing.InvoiceStatusDraftValidating).
		Permit(billing.TriggerDelete, billing.InvoiceStatusDeleteInProgress).
		Permit(billing.TriggerUpdated, billing.InvoiceStatusDraftUpdating)

	stateMachine.Configure(billing.InvoiceStatusDraftSyncing).
		Permit(
			billing.TriggerNext,
			billing.InvoiceStatusDraftManualApprovalNeeded,
			boolFn(not(out.isAutoAdvanceEnabled)),
			boolFn(out.noCriticalValidationErrors),
			boolFn(out.canDraftSyncAdvance),
		).
		Permit(
			billing.TriggerNext,
			billing.InvoiceStatusDraftWaitingAutoApproval,
			boolFn(out.isAutoAdvanceEnabled),
			boolFn(out.noCriticalValidationErrors),
			boolFn(out.canDraftSyncAdvance),
		).
		Permit(billing.TriggerDelete, billing.InvoiceStatusDeleteInProgress).
		Permit(billing.TriggerFailed, billing.InvoiceStatusDraftSyncFailed).
		OnActive(out.syncDraftInvoice)

	stateMachine.Configure(billing.InvoiceStatusDraftSyncFailed).
		Permit(billing.TriggerRetry, billing.InvoiceStatusDraftValidating).
		Permit(billing.TriggerDelete, billing.InvoiceStatusDeleteInProgress).
		Permit(billing.TriggerUpdated, billing.InvoiceStatusDraftUpdating)

	stateMachine.Configure(billing.InvoiceStatusDraftReadyToIssue).
		Permit(billing.TriggerNext, billing.InvoiceStatusIssuingSyncing).
		Permit(billing.TriggerDelete, billing.InvoiceStatusDeleteInProgress).
		Permit(billing.TriggerUpdated, billing.InvoiceStatusDraftUpdating)

	// Automatic and manual approvals
	stateMachine.Configure(billing.InvoiceStatusDraftWaitingAutoApproval).
		// Manual approval forces the draft invoice to be issued regardless of the review period
		Permit(billing.TriggerApprove, billing.InvoiceStatusDraftReadyToIssue).
		Permit(billing.TriggerUpdated, billing.InvoiceStatusDraftUpdating).
		Permit(billing.TriggerDelete, billing.InvoiceStatusDeleteInProgress).
		Permit(billing.TriggerNext,
			billing.InvoiceStatusDraftReadyToIssue,
			boolFn(out.shouldAutoAdvance),
			boolFn(out.noCriticalValidationErrors),
		)

	// This state is a pre-issuing state where we can halt the execution and execute issuing in the background
	// if needed
	stateMachine.Configure(billing.InvoiceStatusDraftManualApprovalNeeded).
		Permit(billing.TriggerApprove,
			billing.InvoiceStatusDraftReadyToIssue,
			boolFn(out.noCriticalValidationErrors),
		).
		Permit(billing.TriggerUpdated, billing.InvoiceStatusDraftUpdating).
		Permit(billing.TriggerDelete, billing.InvoiceStatusDeleteInProgress)

	// Deletion state
	stateMachine.Configure(billing.InvoiceStatusDeleteInProgress).
		Permit(billing.TriggerNext, billing.InvoiceStatusDeleteSyncing).
		Permit(billing.TriggerFailed, billing.InvoiceStatusDeleteFailed).
		OnActive(out.deleteInvoice)

	stateMachine.Configure(billing.InvoiceStatusDeleteSyncing).
		Permit(billing.TriggerNext, billing.InvoiceStatusDeleted).
		Permit(billing.TriggerFailed, billing.InvoiceStatusDeleteFailed).
		OnActive(out.syncDeletedInvoice)

	stateMachine.Configure(billing.InvoiceStatusDeleteFailed).
		Permit(billing.TriggerRetry, billing.InvoiceStatusDeleteInProgress)

	stateMachine.Configure(billing.InvoiceStatusDeleted)

	// Issuing state

	stateMachine.Configure(billing.InvoiceStatusIssuingSyncing).
		Permit(billing.TriggerNext,
			billing.InvoiceStatusIssued,
			boolFn(out.canIssuingSyncAdvance),
		).
		Permit(billing.TriggerFailed, billing.InvoiceStatusIssuingSyncFailed).
		Permit(billing.TriggerDelete, billing.InvoiceStatusDeleteInProgress).
		OnActive(out.finalizeInvoice)

	stateMachine.Configure(billing.InvoiceStatusIssuingSyncFailed).
		Permit(billing.TriggerDelete, billing.InvoiceStatusDeleteInProgress).
		Permit(billing.TriggerRetry, billing.InvoiceStatusIssuingSyncing)

	// Issued state
	stateMachine.Configure(billing.InvoiceStatusIssued).
		Permit(billing.TriggerNext, billing.InvoiceStatusPaymentProcessingPending).
		Permit(billing.TriggerVoid, billing.InvoiceStatusVoided)

	// Payment states
	stateMachine.Configure(billing.InvoiceStatusPaymentProcessingPending).
		Permit(billing.TriggerPaid, billing.InvoiceStatusPaid).
		Permit(billing.TriggerFailed, billing.InvoiceStatusPaymentProcessingFailed).
		Permit(billing.TriggerPaymentUncollectible, billing.InvoiceStatusUncollectible).
		Permit(billing.TriggerPaymentOverdue, billing.InvoiceStatusOverdue).
		Permit(billing.TriggerActionRequired, billing.InvoiceStatusPaymentProcessingActionRequired).
		Permit(billing.TriggerVoid, billing.InvoiceStatusVoided)

	stateMachine.Configure(billing.InvoiceStatusPaymentProcessingFailed).
		Permit(billing.TriggerPaid, billing.InvoiceStatusPaid).
		Permit(billing.TriggerRetry, billing.InvoiceStatusPaymentProcessingPending).
		Permit(billing.TriggerPaymentOverdue, billing.InvoiceStatusOverdue).
		Permit(billing.TriggerPaymentUncollectible, billing.InvoiceStatusUncollectible).
		Permit(billing.TriggerActionRequired, billing.InvoiceStatusPaymentProcessingActionRequired).
		Permit(billing.TriggerVoid, billing.InvoiceStatusVoided)

	stateMachine.Configure(billing.InvoiceStatusPaymentProcessingActionRequired).
		Permit(billing.TriggerPaid, billing.InvoiceStatusPaid).
		Permit(billing.TriggerFailed, billing.InvoiceStatusPaymentProcessingFailed).
		Permit(billing.TriggerRetry, billing.InvoiceStatusPaymentProcessingPending).
		Permit(billing.TriggerPaymentOverdue, billing.InvoiceStatusOverdue).
		Permit(billing.TriggerPaymentUncollectible, billing.InvoiceStatusUncollectible).
		Permit(billing.TriggerVoid, billing.InvoiceStatusVoided)

	// Payment overdue state

	stateMachine.Configure(billing.InvoiceStatusOverdue).
		Permit(billing.TriggerPaid, billing.InvoiceStatusPaid).
		Permit(billing.TriggerFailed, billing.InvoiceStatusPaymentProcessingFailed).
		Permit(billing.TriggerRetry, billing.InvoiceStatusPaymentProcessingPending).
		Permit(billing.TriggerPaymentUncollectible, billing.InvoiceStatusUncollectible).
		Permit(billing.TriggerActionRequired, billing.InvoiceStatusPaymentProcessingActionRequired)

	stateMachine.Configure(billing.InvoiceStatusUncollectible).
		Permit(billing.TriggerVoid, billing.InvoiceStatusVoided).
		Permit(billing.TriggerPaid, billing.InvoiceStatusPaid)

	// Final payment states
	stateMachine.Configure(billing.InvoiceStatusPaid)

	stateMachine.Configure(billing.InvoiceStatusVoided)

	out.StateMachine = stateMachine

	return out
}

type InvoiceStateMachineCallback func(context.Context, *InvoiceStateMachine) error

func (s *Service) WithInvoiceStateMachine(ctx context.Context, invoice billing.Invoice, cb InvoiceStateMachineCallback) (billing.Invoice, error) {
	sm := invoiceStateMachineCache.Get().(*InvoiceStateMachine)
	sm.Logger = s.logger
	sm.Publisher = s.publisher
	sm.FSNamespaceLockdown = s.fsNamespaceLockdown
	// Stateless doesn't store any state in the state machine, so it's fine to reuse the state machine itself
	sm.Invoice = invoice
	sm.Calculator = s.invoiceCalculator
	sm.Service = s

	defer func() {
		sm.Invoice = billing.Invoice{}
		sm.Calculator = nil
		sm.Service = nil
		sm.Logger = nil
		sm.Publisher = nil
		sm.FSNamespaceLockdown = nil

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
		// cross invoice operations, for now the sugar around grathering invoices will handle
		// the status details.
		return billing.InvoiceStatusDetails{}, nil
	}

	var outErr, err error
	availableActions := billing.InvoiceAvailableActions{}

	if availableActions.Advance, err = m.calculateAvailableActionDetails(ctx, billing.TriggerNext); err != nil {
		outErr = errors.Join(outErr, err)
	}

	if availableActions.Delete, err = m.calculateAvailableActionDetails(ctx, billing.TriggerDelete); err != nil {
		outErr = errors.Join(outErr, err)
	}

	if availableActions.Retry, err = m.calculateAvailableActionDetails(ctx, billing.TriggerRetry); err != nil {
		outErr = errors.Join(outErr, err)
	}

	if availableActions.Approve, err = m.calculateAvailableActionDetails(ctx, billing.TriggerApprove); err != nil {
		outErr = errors.Join(outErr, err)
	}

	if availableActions.SnapshotQuantities, err = m.calculateAvailableActionDetails(ctx, billing.TriggerSnapshotQuantities); err != nil {
		outErr = errors.Join(outErr, err)
	}

	mutable, err := m.StateMachine.CanFireCtx(ctx, billing.TriggerUpdated)
	if err != nil {
		outErr = errors.Join(outErr, err)
	}

	// TODO[OM-988]: add more actions (void, delete, etc.)

	return billing.InvoiceStatusDetails{
		Immutable:        !mutable,
		Failed:           m.Invoice.Status.IsFailed(),
		AvailableActions: availableActions,
	}, outErr
}

func (m *InvoiceStateMachine) calculateAvailableActionDetails(ctx context.Context, baseTrigger billing.InvoiceTrigger) (*billing.InvoiceAvailableActionDetails, error) {
	ok, err := m.StateMachine.CanFireCtx(ctx, baseTrigger)
	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, nil
	}

	// Given we don't have access to the underlying graph we need to emulate the state transitions without any side-effects.
	// To achieve this, we are temporary modifying the invoice object, but never invoke the
	// ActiveCtx to prevent any callbacks from being executed.

	originalState := m.Invoice.Status
	originalValidationErrors := m.Invoice.ValidationIssues
	originalPaymentProcessingEnteredAt := m.Invoice.PaymentProcessingEnteredAt
	m.Invoice.ValidationIssues = nil

	if err := m.StateMachine.FireCtx(ctx, baseTrigger); err != nil {
		return nil, err
	}

	for {
		canFire, err := m.StateMachine.CanFireCtx(ctx, billing.TriggerNext)
		if err != nil {
			return nil, err
		}

		if !canFire {
			break
		}

		if err := m.StateMachine.FireCtx(ctx, billing.TriggerNext); err != nil {
			return nil, err
		}
	}

	resultingState := m.Invoice.Status
	m.Invoice.Status = originalState
	m.Invoice.PaymentProcessingEnteredAt = originalPaymentProcessingEnteredAt
	m.Invoice.ValidationIssues = originalValidationErrors

	return &billing.InvoiceAvailableActionDetails{
		ResultingState: resultingState,
	}, nil
}

func (m *InvoiceStateMachine) AdvanceUntilStateStable(ctx context.Context) error {
	for {
		preAdvanceState, err := billing.NewEventInvoice(m.Invoice)
		if err != nil {
			return err
		}

		canFire, err := m.StateMachine.CanFireCtx(ctx, billing.TriggerNext)
		if err != nil {
			return err
		}

		// We have reached a state that requires either manual intervention or that is final
		if !canFire {
			if err := m.triggerPostAdvanceHooks(ctx); err != nil {
				return err
			}

			return m.Invoice.ValidationIssues.AsError()
		}

		if err := m.FireAndActivate(ctx, billing.TriggerNext); err != nil {
			return fmt.Errorf("cannot transition to the next status [current_status=%s]: %w", m.Invoice.Status, err)
		}

		// Let's emit an event for the transition
		event, err := billing.NewInvoiceUpdatedEvent(m.Invoice, preAdvanceState)
		if err != nil {
			return fmt.Errorf("error creating invoice updated event: %w", err)
		}

		if err := m.Publisher.Publish(ctx, event); err != nil {
			return fmt.Errorf("error emitting invoice updated event: %w", err)
		}
	}
}

func (m *InvoiceStateMachine) CanFire(ctx context.Context, trigger billing.InvoiceTrigger) (bool, error) {
	return m.StateMachine.CanFireCtx(ctx, trigger)
}

func (m *InvoiceStateMachine) TriggerFailed(ctx context.Context) error {
	if err := m.StateMachine.FireCtx(ctx, billing.TriggerFailed); err != nil {
		return err
	}

	activationError := m.StateMachine.ActivateCtx(ctx)
	if activationError != nil {
		return activationError
	}

	return nil
}

// FireAndActivate fires the trigger and activates the new state, if activation fails it automatically
// transitions to the failed state and activates that.
// In addition to the activation a calculation is always performed to ensure that the invoice is up to date.
func (m *InvoiceStateMachine) FireAndActivate(ctx context.Context, trigger billing.InvoiceTrigger) error {
	if err := m.StateMachine.FireCtx(ctx, trigger); err != nil {
		return err
	}

	activationError := m.StateMachine.ActivateCtx(ctx)
	if activationError != nil || m.Invoice.HasCriticalValidationIssues() {
		validationIssues := m.Invoice.ValidationIssues.Clone()

		// There was an error activating the state, we should trigger a transition to the failed state
		canFire, err := m.StateMachine.CanFireCtx(ctx, billing.TriggerFailed)
		if err != nil {
			return fmt.Errorf("failed to check if we can transition to failed state: %w", err)
		}

		if !canFire {
			return fmt.Errorf("cannot move into failed state: %w", activationError)
		}

		if err := m.StateMachine.FireCtx(ctx, billing.TriggerFailed); err != nil {
			return fmt.Errorf("failed to transition to failed state: %w", err)
		}

		if activationError != nil {
			return activationError
		}

		return validationIssues.AsError()
	}

	return nil
}

func (m *InvoiceStateMachine) withInvoicingApp(op billing.InvoiceOperation, cb func(billing.InvoicingApp) (*billing.InvoiceOperation, error)) error {
	invocingBase := m.Invoice.Workflow.Apps.Invoicing
	invoicingApp, ok := invocingBase.(billing.InvoicingApp)
	if !ok {
		// If this happens we are rolling back the state transition (as we are not wrapping this into a validation issue)
		return fmt.Errorf("app [type=%s, id=%s] does not implement the invoicing interface",
			m.Invoice.Workflow.Apps.Invoicing.GetType(),
			m.Invoice.Workflow.Apps.Invoicing.GetID().ID)
	}

	opOverride, result := cb(invoicingApp)
	if opOverride != nil {
		op = *opOverride
		if err := op.Validate(); err != nil {
			return err
		}
	}

	component := billing.AppTypeCapabilityToComponent(invocingBase.GetType(), app.CapabilityTypeInvoiceCustomers, op)

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

func (m *InvoiceStateMachine) triggerPostAdvanceHooks(ctx context.Context) error {
	return m.withInvoicingApp(billing.InvoiceOpPostAdvanceHook, func(app billing.InvoicingApp) (*billing.InvoiceOperation, error) {
		if hook, ok := app.(billing.InvoicingAppPostAdvanceHook); ok {
			res, err := hook.PostAdvanceInvoiceHook(ctx, m.Invoice.Clone())
			if err != nil {
				return nil, err
			}

			if res == nil {
				return nil, nil
			}

			var opOverride *billing.InvoiceOperation
			if trigger := res.GetTriggerToInvoke(); trigger != nil {
				if trigger.ValidationErrors != nil {
					opOverride = &trigger.ValidationErrors.Operation
				}

				return opOverride, m.HandleInvoiceTrigger(ctx, *trigger)
			}

			return opOverride, nil
		}

		return nil, nil
	})
}

func (m *InvoiceStateMachine) HandleInvoiceTrigger(ctx context.Context, trigger billing.InvoiceTriggerInput) error {
	if err := trigger.Validate(); err != nil {
		return err
	}

	if trigger.Invoice != m.Invoice.InvoiceID() {
		return fmt.Errorf("trigger invoice ID does not match the current invoice ID")
	}

	preAdvanceState, err := billing.NewEventInvoice(m.Invoice)
	if err != nil {
		return err
	}

	err = m.FireAndActivate(ctx, trigger.Trigger)
	if err != nil {
		return err
	}

	event, err := billing.NewInvoiceUpdatedEvent(m.Invoice, preAdvanceState)
	if err != nil {
		return err
	}

	if err := m.Publisher.Publish(ctx, event); err != nil {
		return err
	}

	if trigger.ValidationErrors != nil {
		return errors.Join(trigger.ValidationErrors.Errors...)
	}

	return nil
}

func (m *InvoiceStateMachine) mergeUpsertInvoiceResult(result *billing.UpsertInvoiceResult) error {
	return result.MergeIntoInvoice(&m.Invoice)
}

// validateDraftInvoice validates the draft invoice using the apps referenced in the invoice.
func (m *InvoiceStateMachine) validateDraftInvoice(ctx context.Context) error {
	if err := m.validateNamespaceLockdown(); err != nil {
		return err
	}

	return m.withInvoicingApp(billing.InvoiceOpValidate, func(app billing.InvoicingApp) (*billing.InvoiceOperation, error) {
		return nil, app.ValidateInvoice(ctx, m.Invoice.Clone())
	})
}

func (m *InvoiceStateMachine) calculateInvoice(ctx context.Context) error {
	return m.Calculator.Calculate(&m.Invoice)
}

// syncDraftInvoice syncs the draft invoice with the external system.
func (m *InvoiceStateMachine) syncDraftInvoice(ctx context.Context) error {
	if err := m.validateNamespaceLockdown(); err != nil {
		return err
	}

	// Let's save the invoice so that we are sure that all the IDs are available for downstream apps
	return m.withInvoicingApp(billing.InvoiceOpSync, func(app billing.InvoicingApp) (*billing.InvoiceOperation, error) {
		results, err := app.UpsertInvoice(ctx, m.Invoice.Clone())
		if err != nil {
			return nil, err
		}

		if results == nil {
			return nil, nil
		}

		return nil, m.mergeUpsertInvoiceResult(results)
	})
}

// finalizeInvoice finalizes the invoice using the invoicing app and payment app (later).
func (m *InvoiceStateMachine) finalizeInvoice(ctx context.Context) error {
	if err := m.validateNamespaceLockdown(); err != nil {
		return err
	}

	return m.withInvoicingApp(billing.InvoiceOpFinalize, func(app billing.InvoicingApp) (*billing.InvoiceOperation, error) {
		clonedInvoice := m.Invoice.Clone()
		// First we sync the invoice
		upsertResults, err := app.UpsertInvoice(ctx, clonedInvoice)
		if err != nil {
			return nil, err
		}

		if upsertResults != nil {
			if err := m.mergeUpsertInvoiceResult(upsertResults); err != nil {
				return nil, err
			}
		}

		// Let's set the issuedAt now as if the finalization fails we will roll back the state transition
		m.Invoice.IssuedAt = lo.ToPtr(clock.Now().In(time.UTC))

		// Let's update the dueAt now that we know when the invoice was issued (so that downstream apps
		// can use this during the sync)
		if err := invoicecalc.CalculateDueAt(&m.Invoice); err != nil {
			return nil, err
		}

		results, err := app.FinalizeInvoice(ctx, clonedInvoice)
		if err != nil {
			return nil, err
		}

		if results != nil {
			if err := results.MergeIntoInvoice(&m.Invoice); err != nil {
				return nil, err
			}
		}

		return nil, nil
	})
}

// syncDeletedInvoice syncs the deleted invoice with the external system
func (m *InvoiceStateMachine) syncDeletedInvoice(ctx context.Context) error {
	if err := m.validateNamespaceLockdown(); err != nil {
		return err
	}

	return m.withInvoicingApp(billing.InvoiceOpDelete, func(app billing.InvoicingApp) (*billing.InvoiceOperation, error) {
		return nil, app.DeleteInvoice(ctx, m.Invoice.Clone())
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

func (m *InvoiceStateMachine) isReadyForCollection() bool {
	if m.Invoice.CollectionAt == nil {
		m.Logger.Warn("invoice has no collection at set, assuming collection is not required", "invoice", m.Invoice.ID)
		return true
	}

	if clock.Now().Before(*m.Invoice.CollectionAt) {
		return false
	}

	return true
}

func (m *InvoiceStateMachine) snapshotQuantityAsNeeded(ctx context.Context) error {
	// Let's skip the snapshot if we already have the snapshot and it happened after the collection date
	if m.Invoice.QuantitySnapshotedAt != nil && !m.Invoice.QuantitySnapshotedAt.Before(m.Invoice.DefaultCollectionAtForStandardInvoice()) {
		m.Logger.InfoContext(ctx, "skipping snapshot quantity as it already exists and was taken after the collection date",
			"invoice", m.Invoice.ID,
			"quantity_snapshoted_at", m.Invoice.QuantitySnapshotedAt,
			"collection_at", m.Invoice.CollectionAt,
		)
		return nil
	}

	// We don't have the snapshot and the collection date is in the future
	if m.Invoice.QuantitySnapshotedAt == nil && clock.Now().Before(*m.Invoice.CollectionAt) {
		return nil
	}

	lineSvcs, err := m.Service.lineService.FromEntities(m.Invoice.Lines.OrEmpty())
	if err != nil {
		return fmt.Errorf("creating line services: %w", err)
	}

	err = m.Service.snapshotLineQuantitiesInParallel(ctx, m.Invoice.Customer, lineSvcs)
	if err != nil {
		return fmt.Errorf("snapshotting lines: %w", err)
	}

	m.Invoice.QuantitySnapshotedAt = lo.ToPtr(clock.Now().UTC())

	return nil
}

func (m *InvoiceStateMachine) canDraftSyncAdvance() bool {
	if invoicingApp, ok := m.Invoice.Workflow.Apps.Invoicing.(billing.InvoicingAppAsyncSyncer); ok {
		can, err := invoicingApp.CanDraftSyncAdvance(m.Invoice)
		if err != nil {
			m.Logger.Error("error checking if we can advance the draft invoice", "error", err)
			return false
		}
		return can
	}

	return true
}

func (m *InvoiceStateMachine) validateNamespaceLockdown() error {
	if slices.Contains(m.FSNamespaceLockdown, m.Invoice.Namespace) {
		return fmt.Errorf("%w: %s", billing.ErrNamespaceLocked, m.Invoice.Namespace)
	}

	return nil
}

func (m *InvoiceStateMachine) canIssuingSyncAdvance() bool {
	if invoicingApp, ok := m.Invoice.Workflow.Apps.Invoicing.(billing.InvoicingAppAsyncSyncer); ok {
		can, err := invoicingApp.CanIssuingSyncAdvance(m.Invoice)
		if err != nil {
			m.Logger.Error("error checking if we can advance the issuing invoice", "error", err)
			return false
		}
		return can
	}

	return true
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
