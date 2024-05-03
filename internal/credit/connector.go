package credit

import (
	"context"
	"time"
)

type ListGrantsParams struct {
	Subjects          []string
	From              *time.Time
	To                *time.Time
	FromHighWatermark bool
	IncludeVoid       bool
	Limit             int
}

type ListFeaturesParams struct {
	IncludeArchived bool
}

type Connector interface {
	// Grant
	CreateGrant(ctx context.Context, namespace string, grant Grant) (Grant, error)
	VoidGrant(ctx context.Context, namespace string, grant Grant) (Grant, error)
	ListGrants(ctx context.Context, namespace string, params ListGrantsParams) ([]Grant, error)
	GetGrant(ctx context.Context, namespace string, id string) (Grant, error)

	// Credit
	GetBalance(ctx context.Context, namespace string, subject string, cutline time.Time) (Balance, error)
	GetHistory(ctx context.Context, namespace string, subject string, from time.Time, to time.Time, limit int) (LedgerEntryList, error)
	GetHighWatermark(ctx context.Context, namespace string, subject string) (HighWatermark, error)
	Reset(ctx context.Context, namespace string, reset Reset) (Reset, []Grant, error)

	// Feature
	CreateFeature(ctx context.Context, namespace string, feature Feature) (Feature, error)
	DeleteFeature(ctx context.Context, namespace string, id string) error
	ListFeatures(ctx context.Context, namespace string, params ListFeaturesParams) ([]Feature, error)
	GetFeature(ctx context.Context, namespace string, id string) (Feature, error)
}
