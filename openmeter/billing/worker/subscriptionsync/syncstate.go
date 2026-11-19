package subscriptionsync

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

type SyncStateAdapter interface {
	GetSyncStates(context.Context, GetSyncStatesInput) ([]SyncStateManaged, error)
	MarkSubscriptionChanged(context.Context, models.NamespacedID) error
	RecordSyncResult(context.Context, RecordSyncResultInput) error
}

type SyncStateState string

const (
	// SyncStateStateNew means that no sync result has been recorded yet,
	// even when the initialized current and last synced revisions are equal.
	SyncStateStateNew SyncStateState = "new"
	// SyncStateStateActive records a successful non-terminal reconciliation,
	// including scheduled subscriptions with billable items.
	SyncStateStateActive SyncStateState = "active"
	// SyncStateStateNoBillables records successful reconciliation of a subscription
	// with no billable items in any phase. No periodic reconciliation is scheduled.
	SyncStateStateNoBillables SyncStateState = "no_billables"
	// SyncStateStateSubscriptionDeleted records successful reconciliation against
	// an empty billing target because the subscription has a non-nil DeletedAt.
	SyncStateStateSubscriptionDeleted SyncStateState = "subscription_deleted"
	// SyncStateStateCustomerDeletedBefore records successful reconciliation against
	// an empty billing target because the customer was deleted at or before the
	// non-deleted subscription's ActiveFrom.
	SyncStateStateCustomerDeletedBefore SyncStateState = "customer_deleted_before"
	// SyncStateStateEnded records successful billing reconciliation at or after
	// the earlier of subscription ActiveTo and customer deletion, with a sync
	// horizon that covers that end.
	SyncStateStateEnded SyncStateState = "ended"
	// SyncStateStateTooOld records an intentional skip of billing
	// reconciliation because ActiveTo is more than 30 days before the worker clock.
	SyncStateStateTooOld SyncStateState = "too_old"
)

// validateNextReconciliationAt keeps the deadline in step with the state: an active
// result without one would silently drop out of reconciliation, and a leftover
// deadline on any other state would schedule work that is not needed.
func (s SyncStateState) validateNextReconciliationAt(at *time.Time) error {
	if s != SyncStateStateActive {
		if at != nil {
			return errors.New("next reconciliation at is only allowed for an active sync result")
		}

		return nil
	}

	if at == nil || at.IsZero() {
		return errors.New("next reconciliation at is required for an active sync result")
	}

	return nil
}

func (SyncStateState) Values() []string {
	return []string{
		string(SyncStateStateNew),
		string(SyncStateStateActive),
		string(SyncStateStateNoBillables),
		string(SyncStateStateSubscriptionDeleted),
		string(SyncStateStateCustomerDeletedBefore),
		string(SyncStateStateEnded),
		string(SyncStateStateTooOld),
	}
}

func (s SyncStateState) Validate() error {
	if !slices.Contains(s.Values(), string(s)) {
		return fmt.Errorf("invalid sync state: %q", s)
	}

	return nil
}

// SyncState keeps subscription changes and the last completed sync outcome,
// including intentional age skips. New rows have no completed outcome yet.
type SyncState struct {
	SubscriptionID   models.NamespacedID
	LastSuccessState SyncStateState
	// LastSuccessAt is nil until a sync result has been recorded.
	LastSuccessAt *time.Time
	// NextReconciliationAt schedules the next reconciliation. It is required for
	// active results and must be nil for every other state.
	NextReconciliationAt *time.Time
	// LastSyncedSubsRevision is initialized on new rows and acknowledged by recorded sync results.
	LastSyncedSubsRevision string
	// CurrentSubsRevision may change while sync is running; recording its result must not overwrite it.
	CurrentSubsRevision string
}

// SyncStateManaged includes the sync-state row's identity and timestamps.
type SyncStateManaged struct {
	SyncState
	models.NamespacedID
	models.ManagedModel
}

func (s SyncStateManaged) Validate() error {
	var errs []error
	if err := s.SyncState.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("sync state: %w", err))
	}

	if err := s.NamespacedID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("sync state ID: %w", err))
	}

	if err := s.ManagedModel.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("managed model: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type GetSyncStatesInput = []models.NamespacedID

type RecordSyncResultInput struct {
	SubscriptionID       models.NamespacedID
	LastSuccessState     SyncStateState
	LastSuccessAt        time.Time
	NextReconciliationAt *time.Time
	// CurrentSubsRevision is deliberately absent: sync can acknowledge its
	// observed revision but cannot replace a newer one.
	LastSyncedSubsRevision string
}

var _ models.Validator = RecordSyncResultInput{}

func (i SyncState) Validate() error {
	var errs []error
	if err := i.SubscriptionID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("subscription ID: %w", err))
	}

	if err := i.LastSuccessState.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("last success state: %w", err))
	}

	if i.LastSuccessState == SyncStateStateNew {
		if i.LastSuccessAt != nil {
			errs = append(errs, errors.New("new sync state must not have a last success at"))
		}
	} else if i.LastSuccessAt == nil || i.LastSuccessAt.IsZero() {
		errs = append(errs, errors.New("last success at is required for a recorded sync result"))
	}

	if err := i.LastSuccessState.validateNextReconciliationAt(i.NextReconciliationAt); err != nil {
		errs = append(errs, err)
	}

	if i.LastSyncedSubsRevision == "" {
		errs = append(errs, errors.New("last synced subscription revision is required"))
	}

	if i.CurrentSubsRevision == "" {
		errs = append(errs, errors.New("current subscription revision is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (i RecordSyncResultInput) Validate() error {
	var errs []error
	if err := i.SubscriptionID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("subscription ID: %w", err))
	}

	if err := i.LastSuccessState.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("last success state: %w", err))
	}

	if i.LastSuccessState == SyncStateStateNew {
		errs = append(errs, errors.New("new sync state must be initialized by a subscription change"))
	}

	if i.LastSuccessAt.IsZero() {
		errs = append(errs, errors.New("last success at is required"))
	}

	if err := i.LastSuccessState.validateNextReconciliationAt(i.NextReconciliationAt); err != nil {
		errs = append(errs, err)
	}

	if i.LastSyncedSubsRevision == "" {
		errs = append(errs, errors.New("last synced subscription revision is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
