package balanceworker

import "context"

type BalanceWorkerRepository interface {
	ListAffectedEntitlements(ctx context.Context, filterPairs []IngestEventQueryFilter) ([]IngestEventDataResponse, error)
}

type IngestEventQueryFilter struct {
	Namespace  string
	SubjectKey string
	MeterSlugs []string
}

type IngestEventDataResponse struct {
	Namespace     string
	EntitlementID string
	SubjectKey    string
	// not all entitlements have a meter associated
	MeterSlug *string
}
