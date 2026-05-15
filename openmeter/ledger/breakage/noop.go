package breakage

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/ledger"
)

// NoopService disables breakage while keeping constructors explicit. It is
// useful for legacy tests and deployments that wire credit purchases without
// expiration support.
type NoopService struct{}

var _ Service = NoopService{}

func (NoopService) PlanIssuance(context.Context, PlanIssuanceInput) ([]ledger.TransactionInput, []PendingRecord, error) {
	return nil, nil, nil
}

func (NoopService) ReleasePlan(context.Context, ReleasePlanInput) (ledger.TransactionInput, PendingRecord, error) {
	return nil, PendingRecord{}, nil
}

func (NoopService) ReopenRelease(context.Context, ReopenReleaseInput) (ledger.TransactionInput, PendingRecord, error) {
	return nil, PendingRecord{}, nil
}

func (NoopService) ListPlans(context.Context, ListPlansInput) ([]Plan, error) {
	return nil, nil
}

func (NoopService) ListReleases(context.Context, ListReleasesInput) ([]Release, error) {
	return nil, nil
}

func (NoopService) ListExpiredRecords(context.Context, ListExpiredRecordsInput) ([]Record, error) {
	return nil, nil
}

func (NoopService) PersistCommittedRecords(context.Context, []PendingRecord, ledger.TransactionGroup) error {
	return nil
}

// NewNoopService returns a breakage service that never plans or releases
// breakage.
func NewNoopService() Service {
	return NoopService{}
}
