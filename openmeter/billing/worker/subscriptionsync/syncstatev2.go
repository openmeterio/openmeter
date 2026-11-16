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
	UpsertSyncStateV2(context.Context, UpsertSyncStateV2Input) error
}

type SyncStateV2State string

const (
	// SyncStateV2StateSubscriptionDeleted records successful reconciliation against
	// an empty billing target because the subscription has a non-nil DeletedAt.
	SyncStateV2StateSubscriptionDeleted SyncStateV2State = "subscription_deleted"
	// SyncStateV2StateCustomerDeletedBefore records successful reconciliation against
	// an empty billing target because the customer was deleted at or before the
	// non-deleted subscription's ActiveFrom.
	SyncStateV2StateCustomerDeletedBefore SyncStateV2State = "customer_deleted_before"
)

func (SyncStateV2State) Values() []string {
	return []string{
		string(SyncStateV2StateSubscriptionDeleted),
		string(SyncStateV2StateCustomerDeletedBefore),
	}
}

func (s SyncStateV2State) Validate() error {
	if !slices.Contains(s.Values(), string(s)) {
		return fmt.Errorf("invalid sync state v2 state: %q", s)
	}

	return nil
}

// SyncStateV2 describes completed reconciliation, not pending or failed work.
type SyncStateV2 struct {
	SubscriptionID   models.NamespacedID
	LastSuccessState SyncStateV2State
	LastSuccessAt    time.Time
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

type UpsertSyncStateV2Input = SyncStateV2

var _ models.Validator = UpsertSyncStateV2Input{}

func (i SyncStateV2) Validate() error {
	var errs []error
	if err := i.SubscriptionID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("subscription ID: %w", err))
	}

	if err := i.LastSuccessState.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("last success state: %w", err))
	}

	if i.LastSuccessAt.IsZero() {
		errs = append(errs, errors.New("last success at is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
