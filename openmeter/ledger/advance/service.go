package advance

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/ledger"
)

// Service plans advance postings under customer posting locks. The caller commits
// the postings and persists their bookkeeping in the same database transaction.
type Service interface {
	PlanIssue(ctx context.Context, input IssueInput) ([]ledger.TransactionInput, error)
	PlanBackfill(ctx context.Context, input BackfillInput) (BackfillPlan, error)
	PlanCorrection(ctx context.Context, input CorrectionInput) (CorrectionPlan, error)
	PlanLegacyCorrection(ctx context.Context, input LegacyCorrectionInput) (CorrectionPlan, error)
}
