package subscriptionsync

import (
	"context"
	"errors"
	"time"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Adapter interface {
	SyncStateAdapter

	entutils.TxCreator
}

type SyncStateAdapter interface {
	InvalidateSyncState(ctx context.Context, input InvalidateSyncStateInput) error
	GetSyncStates(ctx context.Context, input GetSyncStatesInput) ([]SyncState, error)
	UpsertSyncState(ctx context.Context, input UpsertSyncStateInput) error
}

type InvalidateSyncStateInput = models.NamespacedID

type GetSyncStatesInput = []models.NamespacedID

type UpsertSyncStateInput = SyncState

type SyncState struct {
	SubscriptionID models.NamespacedID
	HasBillables   bool
	SyncedAt       time.Time
	NextSyncAfter  *time.Time
}

func (i UpsertSyncStateInput) Validate() error {
	var errs []error

	if err := i.SubscriptionID.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.SyncedAt.IsZero() {
		errs = append(errs, errors.New("synced at is required"))
	}

	if i.HasBillables && i.NextSyncAfter == nil {
		errs = append(errs, errors.New("next sync after is required when the subscription has billables"))
	}

	if i.NextSyncAfter != nil && i.NextSyncAfter.IsZero() {
		errs = append(errs, errors.New("next sync after must not be zero"))
	}

	return errors.Join(errs...)
}
