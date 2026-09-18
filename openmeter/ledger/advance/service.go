package advance

import "context"

type Service interface {
	// PlanBackfill selects advances in collection order and builds their attribution
	// templates. The caller must hold the customer's posting locks in the database
	// transaction that will commit these templates and persist legacy allocations.
	PlanBackfill(ctx context.Context, input BackfillInput) (BackfillPlan, error)
}
