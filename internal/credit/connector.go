package credit

import (
	"context"
	"time"
)

type ListGrantsParams struct {
	Namespace         string
	LedgerIDs         []LedgerID
	From              *time.Time
	To                *time.Time
	FromHighWatermark bool
	IncludeVoid       bool
	Limit             int
}

type FeatureOrderBy string

const (
	FeatureOrderByCreatedAt FeatureOrderBy = "created_at"
	FeatureOrderByUpdatedAt FeatureOrderBy = "updated_at"
	FeatureOrderByID        FeatureOrderBy = "id"
)

type ListFeaturesParams struct {
	Namespace       string
	IncludeArchived bool
	Offset          int
	Limit           int
	OrderBy         FeatureOrderBy
}

type LedgerOrderBy string

const (
	LedgerOrderByCreatedAt LedgerOrderBy = "created_at"
	LedgerOrderBySubject   LedgerOrderBy = "subject"
	LedgerOrderByID        LedgerOrderBy = "id"
)

type ListLedgersParams struct {
	Namespace   string
	Subjects    []string
	SubjectLike string
	Offset      int
	Limit       int
	OrderBy     LedgerOrderBy
}

type Connector interface {
	// Ledger
	CreateLedger(ctx context.Context, ledger Ledger) (Ledger, error)
	ListLedgers(ctx context.Context, params ListLedgersParams) ([]Ledger, error)

	// Grant
	CreateGrant(ctx context.Context, grant Grant) (Grant, error)
	VoidGrant(ctx context.Context, grant Grant) (Grant, error)
	ListGrants(ctx context.Context, params ListGrantsParams) ([]Grant, error)
	GetGrant(ctx context.Context, grantID NamespacedGrantID) (Grant, error)

	// Credit
	GetBalance(ctx context.Context, ledgerID NamespacedLedgerID, cutline time.Time) (Balance, error)
	GetHistory(ctx context.Context, ledgerID NamespacedLedgerID, from time.Time, to time.Time, limit int) (LedgerEntryList, error)
	GetHighWatermark(ctx context.Context, ledgerID NamespacedLedgerID) (HighWatermark, error)
	Reset(ctx context.Context, reset Reset) (Reset, []Grant, error)

	// Feature
	CreateFeature(ctx context.Context, feature Feature) (Feature, error)
	DeleteFeature(ctx context.Context, featureID NamespacedFeatureID) error
	ListFeatures(ctx context.Context, params ListFeaturesParams) ([]Feature, error)
	GetFeature(ctx context.Context, featureID NamespacedFeatureID) (Feature, error)
}
