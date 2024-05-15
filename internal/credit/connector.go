package credit

import (
	"context"
	"time"

	"github.com/oklog/ulid/v2"
)

type ListGrantsParams struct {
	Namespace         string
	LedgerIDs         []ulid.ULID
	From              *time.Time
	To                *time.Time
	FromHighWatermark bool
	IncludeVoid       bool
	Limit             int
}

type ListFeaturesParams struct {
	Namespace       string
	IncludeArchived bool
}

type ListLedgersParams struct {
	Namespace string
	Subjects  []string
	Offset    int
	Limit     int
}

type NamespacedID struct {
	Namespace string
	ID        ulid.ULID
}

func NewNamespacedID(ns string, id ulid.ULID) NamespacedID {
	return NamespacedID{
		Namespace: ns,
		ID:        id,
	}
}

type Connector interface {
	// Ledger
	CreateLedger(ctx context.Context, ledger Ledger) (Ledger, error)
	ListLedgers(ctx context.Context, params ListLedgersParams) ([]Ledger, error)

	// Grant
	CreateGrant(ctx context.Context, grant Grant) (Grant, error)
	VoidGrant(ctx context.Context, grant Grant) (Grant, error)
	ListGrants(ctx context.Context, params ListGrantsParams) ([]Grant, error)
	GetGrant(ctx context.Context, grantID NamespacedID) (Grant, error)

	// Credit
	GetBalance(ctx context.Context, ledgerID NamespacedID, cutline time.Time) (Balance, error)
	GetHistory(ctx context.Context, ledgerID NamespacedID, from time.Time, to time.Time, limit int) (LedgerEntryList, error)
	GetHighWatermark(ctx context.Context, ledgerID NamespacedID) (HighWatermark, error)
	Reset(ctx context.Context, reset Reset) (Reset, []Grant, error)

	// Feature
	CreateFeature(ctx context.Context, feature Feature) (Feature, error)
	DeleteFeature(ctx context.Context, featureID NamespacedID) error
	ListFeatures(ctx context.Context, params ListFeaturesParams) ([]Feature, error)
	GetFeature(ctx context.Context, featureID NamespacedID) (Feature, error)
}
