package adapter

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/creditrealizationlineage"
	"github.com/openmeterio/openmeter/openmeter/ent/db/creditrealizationlineagesegment"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

type lineageRepo struct {
	db *entdb.Client
}

func (r *lineageRepo) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	txCtx, rawConfig, eDriver, err := r.db.HijackTx(ctx, &sql.TxOptions{
		ReadOnly: false,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hijack transaction: %w", err)
	}

	return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil
}

func (r *lineageRepo) WithTx(ctx context.Context, tx *entutils.TxDriver) *lineageRepo {
	txDB := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig())

	return &lineageRepo{
		db: txDB.Client(),
	}
}

func (r *lineageRepo) Self() *lineageRepo {
	return r
}

func LoadActiveLineageSegments(
	ctx context.Context,
	db *entdb.Client,
	namespace string,
	realizationIDs []string,
) (map[string][]creditrealization.ActiveLineageSegment, error) {
	return entutils.TransactingRepo(ctx, &lineageRepo{db: db}, func(ctx context.Context, tx *lineageRepo) (map[string][]creditrealization.ActiveLineageSegment, error) {
		if len(realizationIDs) == 0 {
			return map[string][]creditrealization.ActiveLineageSegment{}, nil
		}

		lineages, err := tx.db.CreditRealizationLineage.Query().
			Where(
				creditrealizationlineage.Namespace(namespace),
				creditrealizationlineage.RootRealizationIDIn(realizationIDs...),
			).
			WithSegments(func(q *entdb.CreditRealizationLineageSegmentQuery) {
				q.Where(creditrealizationlineagesegment.ClosedAtIsNil()).
					Order(creditrealizationlineagesegment.ByCreatedAt())
			}).
			All(ctx)
		if err != nil {
			return nil, err
		}

		return lo.SliceToMap(lineages, func(lineage *entdb.CreditRealizationLineage) (string, []creditrealization.ActiveLineageSegment) {
			return lineage.RootRealizationID, lo.Map(lineage.Edges.Segments, func(segment *entdb.CreditRealizationLineageSegment, _ int) creditrealization.ActiveLineageSegment {
				return creditrealization.ActiveLineageSegment{
					ID:                        segment.ID,
					Amount:                    segment.Amount,
					State:                     segment.State,
					BackingTransactionGroupID: segment.BackingTransactionGroupID,
				}
			})
		}), nil
	})
}

func AttachInitialActiveLineageSegments(realizations creditrealization.Realizations) {
	for idx := range realizations {
		originKind, err := creditrealization.LineageOriginKindFromAnnotations(realizations[idx].Annotations)
		if err != nil {
			continue
		}

		initialState := creditrealization.InitialLineageSegmentState(originKind)
		if err := initialState.Validate(); err != nil {
			continue
		}

		realizations[idx].ActiveLineageSegments = []creditrealization.ActiveLineageSegment{
			{
				Amount: realizations[idx].Amount,
				State:  initialState,
			},
		}
	}
}

func WritebackCorrectionLineageSegments(
	ctx context.Context,
	db *entdb.Client,
	namespace string,
	realizations creditrealization.Realizations,
) error {
	return entutils.TransactingRepoWithNoValue(ctx, &lineageRepo{db: db}, func(ctx context.Context, tx *lineageRepo) error {
		if _, err := entutils.GetDriverFromContext(ctx); err != nil {
			return fmt.Errorf("write back correction lineage segments must be called in a transaction: %w", err)
		}

		correctionAmountsByRealizationID := make(map[string]alpacadecimal.Decimal, len(realizations))
		correctionOrder := make([]string, 0)

		for _, realization := range realizations {
			if realization.Type != creditrealization.TypeCorrection || realization.CorrectsRealizationID == nil {
				continue
			}

			correctsRealizationID := *realization.CorrectsRealizationID
			if _, ok := correctionAmountsByRealizationID[correctsRealizationID]; !ok {
				correctionOrder = append(correctionOrder, correctsRealizationID)
			}

			correctionAmountsByRealizationID[correctsRealizationID] = correctionAmountsByRealizationID[correctsRealizationID].Add(realization.Amount.Abs())
		}

		if len(correctionOrder) == 0 {
			return nil
		}

		lineages, err := tx.db.CreditRealizationLineage.Query().
			Where(
				creditrealizationlineage.Namespace(namespace),
				creditrealizationlineage.RootRealizationIDIn(correctionOrder...),
			).
			WithSegments(func(q *entdb.CreditRealizationLineageSegmentQuery) {
				q.Where(creditrealizationlineagesegment.ClosedAtIsNil()).
					Order(creditrealizationlineagesegment.ByCreatedAt())
			}).
			Order(creditrealizationlineage.ByCreatedAt()).
			ForUpdate().
			All(ctx)
		if err != nil {
			return fmt.Errorf("lock lineages for correction writeback: %w", err)
		}

		lineagesByRealizationID := lo.SliceToMap(lineages, func(lineage *entdb.CreditRealizationLineage) (string, *entdb.CreditRealizationLineage) {
			return lineage.RootRealizationID, lineage
		})

		now := clock.Now().Truncate(time.Microsecond)

		for _, realizationID := range correctionOrder {
			lineage, ok := lineagesByRealizationID[realizationID]
			if !ok {
				continue
			}

			remaining := correctionAmountsByRealizationID[realizationID]
			for _, segment := range sortCorrectionWritebackSegments(lineage.Edges.Segments) {
				if !remaining.IsPositive() {
					break
				}

				consumedAmount := minCorrectionWritebackAmount(segment.Amount, remaining)
				if !consumedAmount.IsPositive() {
					continue
				}

				if _, err := tx.db.CreditRealizationLineageSegment.UpdateOneID(segment.ID).
					SetClosedAt(now).
					Save(ctx); err != nil {
					return fmt.Errorf("close active lineage segment %s: %w", segment.ID, err)
				}

				remainder := segment.Amount.Sub(consumedAmount)
				if remainder.IsPositive() {
					create := tx.db.CreditRealizationLineageSegment.Create().
						SetID(ulid.Make().String()).
						SetLineageID(segment.LineageID).
						SetAmount(remainder).
						SetState(segment.State)

					if segment.BackingTransactionGroupID != nil {
						create = create.SetBackingTransactionGroupID(*segment.BackingTransactionGroupID)
					}

					if _, err := create.Save(ctx); err != nil {
						return fmt.Errorf("create lineage segment remainder for %s: %w", segment.ID, err)
					}
				}

				remaining = remaining.Sub(consumedAmount)
			}

			if remaining.IsPositive() {
				return fmt.Errorf("correction amount %s exceeds active lineage coverage for realization %s", remaining.String(), realizationID)
			}
		}

		return nil
	})
}

func LockAdvanceLineagesForBackfill(
	ctx context.Context,
	db *entdb.Client,
	namespace string,
	customerID string,
	currency currencyx.Code,
) ([]*entdb.CreditRealizationLineage, error) {
	return entutils.TransactingRepo(ctx, &lineageRepo{db: db}, func(ctx context.Context, tx *lineageRepo) ([]*entdb.CreditRealizationLineage, error) {
		if _, err := entutils.GetDriverFromContext(ctx); err != nil {
			return nil, fmt.Errorf("lock advance lineages for backfill must be called in a transaction: %w", err)
		}

		return tx.db.CreditRealizationLineage.Query().
			Where(
				creditrealizationlineage.Namespace(namespace),
				creditrealizationlineage.CustomerIDEQ(customerID),
				creditrealizationlineage.CurrencyEQ(currency),
				creditrealizationlineage.HasSegmentsWith(
					creditrealizationlineagesegment.ClosedAtIsNil(),
					creditrealizationlineagesegment.StateEQ(creditrealization.LineageSegmentStateAdvanceUncovered),
				),
			).
			Order(creditrealizationlineage.ByCreatedAt()).
			ForUpdate().
			All(ctx)
	})
}

func sortCorrectionWritebackSegments(segments []*entdb.CreditRealizationLineageSegment) []*entdb.CreditRealizationLineageSegment {
	sorted := append([]*entdb.CreditRealizationLineageSegment(nil), segments...)
	sort.SliceStable(sorted, func(i, j int) bool {
		precedence := func(state creditrealization.LineageSegmentState) int {
			switch state {
			case creditrealization.LineageSegmentStateAdvanceBackfilled:
				return 0
			case creditrealization.LineageSegmentStateAdvanceUncovered:
				return 1
			case creditrealization.LineageSegmentStateRealCredit:
				return 2
			default:
				return 3
			}
		}

		return precedence(sorted[i].State) < precedence(sorted[j].State)
	})

	return sorted
}

func minCorrectionWritebackAmount(a, b alpacadecimal.Decimal) alpacadecimal.Decimal {
	if a.GreaterThan(b) {
		return b
	}

	return a
}
