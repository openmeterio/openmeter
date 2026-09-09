package adapter

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/lib/pq"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/legacylineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfeeruncreditallocations"
	"github.com/openmeterio/openmeter/openmeter/ent/db/chargeusagebasedruncreditallocations"
	"github.com/openmeterio/openmeter/openmeter/ent/db/creditrealizationlineage"
	"github.com/openmeterio/openmeter/openmeter/ent/db/creditrealizationlineagesegment"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func LoadActiveSegmentsByRealizationID(
	ctx context.Context,
	db *entdb.Client,
	namespace string,
	realizationIDs []string,
) (legacylineage.ActiveSegmentsByRealizationID, error) {
	repo := &adapter{db: db}

	return entutils.TransactingRepo(ctx, repo, func(ctx context.Context, tx *adapter) (legacylineage.ActiveSegmentsByRealizationID, error) {
		if len(realizationIDs) == 0 {
			return legacylineage.ActiveSegmentsByRealizationID{}, nil
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

		return lo.SliceToMap(lineages, func(entry *entdb.CreditRealizationLineage) (string, []legacylineage.Segment) {
			return entry.RootRealizationID, lo.Map(entry.Edges.Segments, func(segment *entdb.CreditRealizationLineageSegment, _ int) legacylineage.Segment {
				return mapSegment(segment)
			})
		}), nil
	})
}

func (a *adapter) LoadActiveSegmentsByRealizationID(
	ctx context.Context,
	namespace string,
	realizationIDs []string,
) (legacylineage.ActiveSegmentsByRealizationID, error) {
	return LoadActiveSegmentsByRealizationID(ctx, a.db, namespace, realizationIDs)
}

func (a *adapter) CreateLineages(ctx context.Context, input legacylineage.CreateLineagesInput) error {
	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		rootCreates := make([]*entdb.CreditRealizationLineageCreate, 0, len(input.Specs))
		segmentCreates := make([]*entdb.CreditRealizationLineageSegmentCreate, 0, len(input.Specs))

		for _, spec := range input.Specs {
			rootCreates = append(rootCreates, tx.db.CreditRealizationLineage.Create().
				SetID(spec.LineageID).
				SetNamespace(input.Namespace).
				SetChargeID(input.ChargeID).
				SetRootRealizationID(spec.RootRealizationID).
				SetCustomerID(input.CustomerID).
				SetCurrency(input.Currency.Code).
				SetNillableCustomCurrencyID(input.Currency.CustomCurrencyID).
				SetOriginKind(spec.OriginKind).
				SetAdvanceFeatures(pq.StringArray(spec.AdvanceFeatures)),
			)
			segmentCreates = append(segmentCreates, tx.db.CreditRealizationLineageSegment.Create().
				SetLineageID(spec.LineageID).
				SetAmount(spec.Amount).
				SetState(spec.InitialState),
			)
		}

		if _, err := tx.db.CreditRealizationLineage.CreateBulk(rootCreates...).Save(ctx); err != nil {
			return fmt.Errorf("create credit realization lineages: %w", err)
		}
		if _, err := tx.db.CreditRealizationLineageSegment.CreateBulk(segmentCreates...).Save(ctx); err != nil {
			return fmt.Errorf("create initial credit realization lineage segments: %w", err)
		}

		return nil
	})
}

func (a *adapter) LoadLineagesByCustomer(ctx context.Context, input legacylineage.LoadLineagesByCustomerInput) ([]legacylineage.Lineage, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]legacylineage.Lineage, error) {
		activeSegments := []predicate.CreditRealizationLineageSegment{creditrealizationlineagesegment.ClosedAtIsNil()}
		if input.SegmentState != nil {
			activeSegments = append(activeSegments, creditrealizationlineagesegment.StateEQ(*input.SegmentState))
		}
		query := tx.db.CreditRealizationLineage.Query().Where(
			creditrealizationlineage.Namespace(input.Namespace),
			creditrealizationlineage.CustomerIDEQ(input.CustomerID),
			currencyIdentityPredicate(input.Currency),
		)
		if input.OriginKind != nil {
			query.Where(creditrealizationlineage.OriginKindEQ(*input.OriginKind))
		}
		if input.HasActiveSegments || input.SegmentState != nil {
			// Filtering the eager-loaded children alone would still return every
			// historical root, including ones with no matching segments.
			query.Where(creditrealizationlineage.HasSegmentsWith(activeSegments...))
		}
		if len(input.FeatureFilters) > 0 {
			query.Where(advanceFeaturesOverlap(input.FeatureFilters))
		}
		lineages, err := query.WithSegments(func(q *entdb.CreditRealizationLineageSegmentQuery) {
			q.Where(activeSegments...).Order(creditrealizationlineagesegment.ByCreatedAt(), creditrealizationlineagesegment.ByID())
		}).Order(creditrealizationlineage.ByCreatedAt(), creditrealizationlineage.ByID()).All(ctx)
		if err != nil {
			return nil, err
		}

		mapped := lo.Map(lineages, mapLineage)
		if err := tx.loadOriginalAllocations(ctx, input.Namespace, mapped); err != nil {
			return nil, err
		}
		return mapped, nil
	})
}

// advanceFeaturesOverlap mirrors purchase eligibility, including excluding
// featureless legacy routes from a feature-restricted purchase.
func advanceFeaturesOverlap(features []string) predicate.CreditRealizationLineage {
	return func(s *sql.Selector) {
		s.Where(sql.P(func(b *sql.Builder) {
			b.Ident(s.C(creditrealizationlineage.FieldAdvanceFeatures)).WriteString(" && ").Arg(pq.StringArray(features))
		}))
	}
}

// currencyIdentityPredicate matches a lineage's persisted currency identity:
// the display code plus, for custom currencies, the exact managed currency
// ID. Distinct managed currencies can reuse a code (e.g. after one is
// soft-deleted), so matching on code alone would conflate their lineages.
func currencyIdentityPredicate(ref currencies.CurrencyReference) predicate.CreditRealizationLineage {
	if ref.CustomCurrencyID == nil {
		return creditrealizationlineage.And(
			creditrealizationlineage.CurrencyEQ(ref.Code),
			creditrealizationlineage.CustomCurrencyIDIsNil(),
		)
	}

	return creditrealizationlineage.And(
		creditrealizationlineage.CurrencyEQ(ref.Code),
		creditrealizationlineage.CustomCurrencyIDEQ(*ref.CustomCurrencyID),
	)
}

func (a *adapter) LockCorrectionLineages(ctx context.Context, namespace string, realizationIDs []string) ([]legacylineage.Lineage, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]legacylineage.Lineage, error) {
		if _, err := entutils.GetDriverFromContext(ctx); err != nil {
			return nil, fmt.Errorf("lock correction lineages must be called in a transaction: %w", err)
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
			Order(creditrealizationlineage.ByCreatedAt()).
			ForUpdate().
			All(ctx)
		if err != nil {
			return nil, err
		}

		return lo.Map(lineages, mapLineage), nil
	})
}

func (a *adapter) LockAdvanceLineagesForBackfill(ctx context.Context, namespace string, customerID string, currency currencies.CurrencyReference) ([]legacylineage.Lineage, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]legacylineage.Lineage, error) {
		if _, err := entutils.GetDriverFromContext(ctx); err != nil {
			return nil, fmt.Errorf("lock advance lineages for backfill must be called in a transaction: %w", err)
		}

		lineages, err := tx.db.CreditRealizationLineage.Query().
			Where(
				creditrealizationlineage.Namespace(namespace),
				creditrealizationlineage.CustomerIDEQ(customerID),
				currencyIdentityPredicate(currency),
				creditrealizationlineage.HasSegmentsWith(
					creditrealizationlineagesegment.ClosedAtIsNil(),
					creditrealizationlineagesegment.StateEQ(creditrealization.LineageSegmentStateAdvanceUncovered),
				),
			).
			WithSegments(func(q *entdb.CreditRealizationLineageSegmentQuery) {
				q.Where(creditrealizationlineagesegment.ClosedAtIsNil(), creditrealizationlineagesegment.StateEQ(creditrealization.LineageSegmentStateAdvanceUncovered)).
					Order(creditrealizationlineagesegment.ByCreatedAt(), creditrealizationlineagesegment.ByID())
			}).
			Order(creditrealizationlineage.ByCreatedAt(), creditrealizationlineage.ByID()).
			ForUpdate().
			All(ctx)
		if err != nil {
			return nil, err
		}

		return lo.Map(lineages, mapLineage), nil
	})
}

func (a *adapter) ListActiveSegments(ctx context.Context, input legacylineage.ListActiveSegmentsInput) ([]legacylineage.Segment, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]legacylineage.Segment, error) {
		query := tx.db.CreditRealizationLineageSegment.Query().
			Where(
				creditrealizationlineagesegment.ClosedAtIsNil(),
				creditrealizationlineagesegment.LineageIDIn(input.LineageIDs...),
			).
			Order(
				creditrealizationlineagesegment.ByLineageField(creditrealizationlineage.FieldCreatedAt),
				creditrealizationlineagesegment.ByLineageID(),
				creditrealizationlineagesegment.ByCreatedAt(),
				creditrealizationlineagesegment.ByID(),
			)

		if input.State != nil {
			query = query.Where(creditrealizationlineagesegment.StateEQ(*input.State))
		}

		segments, err := query.All(ctx)
		if err != nil {
			return nil, err
		}

		return lo.Map(segments, func(segment *entdb.CreditRealizationLineageSegment, _ int) legacylineage.Segment {
			return mapSegment(segment)
		}), nil
	})
}

func (a *adapter) CloseSegment(ctx context.Context, segmentID string, closedAt time.Time) error {
	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		if _, err := tx.db.CreditRealizationLineageSegment.UpdateOneID(segmentID).
			SetClosedAt(closedAt).
			Save(ctx); err != nil {
			return err
		}

		return nil
	})
}

func (a *adapter) CreateSegment(ctx context.Context, input legacylineage.CreateSegmentInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("create lineage segment: %w", err)
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		create := tx.db.CreditRealizationLineageSegment.Create().
			SetID(ulid.Make().String()).
			SetLineageID(input.LineageID).
			SetAmount(input.Amount).
			SetState(input.State).
			SetNillableBackingTransactionGroupID(input.BackingTransactionGroupID).
			SetNillableSourceState(input.SourceState).
			SetNillableSourceBackingTransactionGroupID(input.SourceBackingTransactionGroupID)

		_, err := create.Save(ctx)
		return err
	})
}

func mapLineage(entry *entdb.CreditRealizationLineage, _ int) legacylineage.Lineage {
	return legacylineage.Lineage{
		CreatedAt:         entry.CreatedAt,
		ID:                entry.ID,
		ChargeID:          entry.ChargeID,
		RootRealizationID: entry.RootRealizationID,
		CustomerID:        entry.CustomerID,
		Currency:          currencies.CurrencyReference{Code: entry.Currency, CustomCurrencyID: entry.CustomCurrencyID},
		OriginKind:        entry.OriginKind,
		AdvanceFeatures:   []string(entry.AdvanceFeatures),
		Segments: lo.Map(entry.Edges.Segments, func(segment *entdb.CreditRealizationLineageSegment, _ int) legacylineage.Segment {
			return mapSegment(segment)
		}),
	}
}

func mapSegment(segment *entdb.CreditRealizationLineageSegment) legacylineage.Segment {
	return legacylineage.Segment{
		CreatedAt:                       segment.CreatedAt,
		ID:                              segment.ID,
		LineageID:                       segment.LineageID,
		Amount:                          segment.Amount,
		State:                           segment.State,
		BackingTransactionGroupID:       segment.BackingTransactionGroupID,
		SourceState:                     segment.SourceState,
		SourceBackingTransactionGroupID: segment.SourceBackingTransactionGroupID,
	}
}

// Original allocation references identify the collected source bucket within a
// group, including legacy collections without spend provenance.
func (a *adapter) loadOriginalAllocations(ctx context.Context, namespace string, roots []legacylineage.Lineage) error {
	var ids []string
	for _, root := range roots {
		ids = append(ids, root.RootRealizationID)
	}
	if len(ids) == 0 {
		return nil
	}
	flat, err := a.db.ChargeFlatFeeRunCreditAllocations.Query().Where(chargeflatfeeruncreditallocations.Namespace(namespace), chargeflatfeeruncreditallocations.IDIn(ids...)).All(ctx)
	if err != nil {
		return err
	}
	usage, err := a.db.ChargeUsageBasedRunCreditAllocations.Query().Where(chargeusagebasedruncreditallocations.Namespace(namespace), chargeusagebasedruncreditallocations.IDIn(ids...)).All(ctx)
	if err != nil {
		return err
	}
	groups := make(map[string]string, len(flat)+len(usage))
	sortHints := make(map[string]int, len(flat)+len(usage))
	for _, allocation := range flat {
		groups[allocation.ID] = allocation.LedgerTransactionGroupID
		sortHints[allocation.ID] = allocation.SortHint
	}
	for _, allocation := range usage {
		groups[allocation.ID] = allocation.LedgerTransactionGroupID
		sortHints[allocation.ID] = allocation.SortHint
	}
	for i := range roots {
		roots[i].OriginalTransactionGroupID = groups[roots[i].RootRealizationID]
		roots[i].OriginalAllocationSortHint = sortHints[roots[i].RootRealizationID]
	}
	return nil
}
