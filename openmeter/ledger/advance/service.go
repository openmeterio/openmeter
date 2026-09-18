package advance

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/ledger"
)

// Service plans advance postings. The caller owns posting locks, commit and persistence.
type Service interface {
	PlanIssue(ctx context.Context, input IssueInput) ([]ledger.TransactionInput, error)

	// PlanBackfill selects advances in collection order and builds their attribution
	// templates. The caller must hold the customer's posting locks in the database
	// transaction that will commit these templates and persist legacy allocations.
	PlanBackfill(ctx context.Context, input BackfillInput) (BackfillPlan, error)

	PlanCorrection(ctx context.Context, input CorrectionInput) (CorrectionPlan, error)
	PlanLegacyCorrection(ctx context.Context, input LegacyCorrectionInput) (CorrectionPlan, error)
}
