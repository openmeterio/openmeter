package invoicesync

import "context"

var _ Service = (*NoopService)(nil)

// NoopService is a no-op implementation of Service for tests that need the
// dependency but don't exercise sync plan logic.
type NoopService struct{}

func (n NoopService) CreateDraftSyncPlan(ctx context.Context, input CreateSyncPlanInput) error {
	return nil
}

func (n NoopService) CreateIssuingSyncPlan(ctx context.Context, input CreateSyncPlanInput) error {
	return nil
}

func (n NoopService) CreateDeleteSyncPlan(ctx context.Context, input CreateSyncPlanInput) error {
	return nil
}

func (n NoopService) CancelActivePlan(ctx context.Context, namespace, invoiceID string, phase SyncPlanPhase) error {
	return nil
}
