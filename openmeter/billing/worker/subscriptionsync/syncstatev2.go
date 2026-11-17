package subscriptionsync

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

type SyncStateV2Adapter interface {
	GetSyncStatesV2(context.Context, GetSyncStatesV2Input) ([]SyncStateV2Managed, error)
	MarkSubscriptionChangedV2(context.Context, models.NamespacedID) error
	RecordSyncResultV2(context.Context, RecordSyncResultV2Input) error
}

type SyncStateV2State string

const (
	// SyncStateV2StateNew means that no v2 sync result has been recorded yet,
	// even when the initialized current and last synced revisions are equal.
	SyncStateV2StateNew SyncStateV2State = "new"
	// SyncStateV2StateActive records a successful non-terminal reconciliation,
	// including scheduled subscriptions with billable items.
	SyncStateV2StateActive SyncStateV2State = "active"
	// SyncStateV2StateNoBillables records successful reconciliation of a subscription
	// with no billable items in any phase. No periodic reconciliation is scheduled.
	SyncStateV2StateNoBillables SyncStateV2State = "no_billables"
	// SyncStateV2StateSubscriptionDeleted records successful reconciliation against
	// an empty billing target because the subscription has a non-nil DeletedAt.
	SyncStateV2StateSubscriptionDeleted SyncStateV2State = "subscription_deleted"
	// SyncStateV2StateCustomerDeletedBefore records successful reconciliation against
	// an empty billing target because the customer was deleted at or before the
	// non-deleted subscription's ActiveFrom.
	SyncStateV2StateCustomerDeletedBefore SyncStateV2State = "customer_deleted_before"
	// SyncStateV2StateEnded records successful billing reconciliation
	// after ActiveTo, with a sync horizon that covers the subscription end.
	SyncStateV2StateEnded SyncStateV2State = "ended"
	// SyncStateV2StateTooOld records an intentional skip of billing
	// reconciliation because ActiveTo is more than 30 days before the worker clock.
	SyncStateV2StateTooOld SyncStateV2State = "too_old"
)

func (SyncStateV2State) Values() []string {
	return []string{
		string(SyncStateV2StateNew),
		string(SyncStateV2StateActive),
		string(SyncStateV2StateNoBillables),
		string(SyncStateV2StateSubscriptionDeleted),
		string(SyncStateV2StateCustomerDeletedBefore),
		string(SyncStateV2StateEnded),
		string(SyncStateV2StateTooOld),
	}
}

func (s SyncStateV2State) Validate() error {
	if !slices.Contains(s.Values(), string(s)) {
		return fmt.Errorf("invalid sync state v2 state: %q", s)
	}

	return nil
}

// SyncStateV2 keeps subscription changes and the last completed sync outcome,
// including intentional age skips. New rows have no completed outcome yet.
type SyncStateV2 struct {
	SubscriptionID   models.NamespacedID
	LastSuccessState SyncStateV2State
	// LastSuccessAt is nil until a v2 sync result has been recorded.
	LastSuccessAt *time.Time
	// NextReconciliationAt schedules the next v2 reconciliation. It is required for
	// active results and must be nil for every other state.
	NextReconciliationAt *time.Time
	// LastSyncedSubsRevision is initialized on new rows and acknowledged by recorded sync results.
	LastSyncedSubsRevision string
	// CurrentSubsRevision may change while sync is running; recording its result must not overwrite it.
	CurrentSubsRevision string
}

// SyncStateV2Managed includes the sync-state row's identity and timestamps.
type SyncStateV2Managed struct {
	SyncStateV2
	models.NamespacedID
	models.ManagedModel
}

func (s SyncStateV2Managed) Validate() error {
	var errs []error
	if err := s.SyncStateV2.Validate(); err != nil {
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

type GetSyncStatesV2Input = []models.NamespacedID

type RecordSyncResultV2Input struct {
	SubscriptionID       models.NamespacedID
	LastSuccessState     SyncStateV2State
	LastSuccessAt        time.Time
	NextReconciliationAt *time.Time
	// CurrentSubsRevision is deliberately absent: sync can acknowledge its
	// observed revision but cannot replace a newer one.
	LastSyncedSubsRevision string
}

var _ models.Validator = RecordSyncResultV2Input{}

func (i SyncStateV2) Validate() error {
	var errs []error
	if err := i.SubscriptionID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("subscription ID: %w", err))
	}

	if err := i.LastSuccessState.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("last success state: %w", err))
	}

	if i.LastSuccessState == SyncStateV2StateNew {
		if i.LastSuccessAt != nil {
			errs = append(errs, errors.New("new sync state must not have a last success at"))
		}
	} else if i.LastSuccessAt == nil || i.LastSuccessAt.IsZero() {
		errs = append(errs, errors.New("last success at is required for a recorded sync result"))
	}

	if i.LastSuccessState == SyncStateV2StateActive && (i.NextReconciliationAt == nil || i.NextReconciliationAt.IsZero()) {
		errs = append(errs, errors.New("next reconciliation at is required for an active sync result"))
	}

	if i.LastSuccessState != SyncStateV2StateActive && i.NextReconciliationAt != nil {
		errs = append(errs, errors.New("next reconciliation at is only allowed for an active sync result"))
	}

	if i.LastSyncedSubsRevision == "" {
		errs = append(errs, errors.New("last synced subscription revision is required"))
	}

	if i.CurrentSubsRevision == "" {
		errs = append(errs, errors.New("current subscription revision is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (i RecordSyncResultV2Input) Validate() error {
	var errs []error
	if err := i.SubscriptionID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("subscription ID: %w", err))
	}

	if err := i.LastSuccessState.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("last success state: %w", err))
	}

	if i.LastSuccessState == SyncStateV2StateNew {
		errs = append(errs, errors.New("new sync state must be initialized by a subscription change"))
	}

	if i.LastSuccessAt.IsZero() {
		errs = append(errs, errors.New("last success at is required"))
	}

	if i.LastSuccessState == SyncStateV2StateActive && (i.NextReconciliationAt == nil || i.NextReconciliationAt.IsZero()) {
		errs = append(errs, errors.New("next reconciliation at is required for an active sync result"))
	}

	if i.LastSuccessState != SyncStateV2StateActive && i.NextReconciliationAt != nil {
		errs = append(errs, errors.New("next reconciliation at is only allowed for an active sync result"))
	}

	if i.LastSyncedSubsRevision == "" {
		errs = append(errs, errors.New("last synced subscription revision is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
