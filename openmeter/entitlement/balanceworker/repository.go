package balanceworker

import "context"

type BalanceWorkerRepository interface {
	ListEntitlementsAffectedByIngestEvents(ctx context.Context, filters []IngestEventQueryFilter) ([]ListAffectedEntitlementsResponse, error)
}

type IngestEventQueryFilter struct {
	Namespace    string
	EventSubject string
	MeterSlugs   []string
}

type ListAffectedEntitlementsResponse struct {
	Namespace     string
	EntitlementID string
	CustomerID    string
	SubjectKey    string
	// not all entitlements have a meter associated
	MeterSlug *string
}
