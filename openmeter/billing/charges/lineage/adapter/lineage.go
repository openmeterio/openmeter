package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/creditrealizationlineage"
	"github.com/openmeterio/openmeter/openmeter/ent/db/creditrealizationlineagesegment"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func LoadActiveSegmentsByRealizationID(
	ctx context.Context,
	db *entdb.Client,
	namespace string,
	realizationIDs []string,
) (lineage.ActiveSegmentsByRealizationID, error) {
	repo := &adapter{db: db}

	return entutils.TransactingRepo(ctx, repo, func(ctx context.Context, tx *adapter) (lineage.ActiveSegmentsByRealizationID, error) {
		if len(realizationIDs) == 0 {
			return lineage.ActiveSegmentsByRealizationID{}, nil
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

		return lo.SliceToMap(lineages, func(entry *entdb.CreditRealizationLineage) (string, []lineage.Segment) {
			return entry.RootRealizationID, lo.Map(entry.Edges.Segments, func(segment *entdb.CreditRealizationLineageSegment, _ int) lineage.Segment {
				return mapSegment(segment)
			})
		}), nil
	})
}

func (a *adapter) LoadActiveSegmentsByRealizationID(
	ctx context.Context,
	namespace string,
	realizationIDs []string,
) (lineage.ActiveSegmentsByRealizationID, error) {
	return LoadActiveSegmentsByRealizationID(ctx, a.db, namespace, realizationIDs)
}

func (a *adapter) CreateLineages(ctx context.Context, input lineage.CreateLineagesInput) error {
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
				SetCurrency(input.Currency).
				SetOriginKind(spec.OriginKind),
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

func (a *adapter) LoadLineagesByCustomer(ctx context.Context, input lineage.LoadLineagesByCustomerInput) ([]lineage.Lineage, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]lineage.Lineage, error) {
		lineages, err := tx.db.CreditRealizationLineage.Query().
			Where(
				creditrealizationlineage.Namespace(input.Namespace),
				creditrealizationlineage.CustomerIDEQ(input.CustomerID),
				creditrealizationlineage.CurrencyEQ(input.Currency),
			).
			WithSegments(func(q *entdb.CreditRealizationLineageSegmentQuery) {
				q.Where(creditrealizationlineagesegment.ClosedAtIsNil()).
					Order(creditrealizationlineagesegment.ByCreatedAt())
			}).
			Order(creditrealizationlineage.ByCreatedAt()).
			All(ctx)
		if err != nil {
			return nil, err
		}

		return lo.Map(lineages, mapLineage), nil
	})
}

func (a *adapter) LockCorrectionLineages(ctx context.Context, namespace string, realizationIDs []string) ([]lineage.Lineage, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]lineage.Lineage, error) {
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

func (a *adapter) LockAdvanceLineagesForBackfill(ctx context.Context, namespace string, customerID string, currency currencyx.Code) ([]lineage.Lineage, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]lineage.Lineage, error) {
		if _, err := entutils.GetDriverFromContext(ctx); err != nil {
			return nil, fmt.Errorf("lock advance lineages for backfill must be called in a transaction: %w", err)
		}

		lineages, err := tx.db.CreditRealizationLineage.Query().
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
		if err != nil {
			return nil, err
		}

		return lo.Map(lineages, func(entry *entdb.CreditRealizationLineage, _ int) lineage.Lineage {
			return lineage.Lineage{
				ID:                entry.ID,
				ChargeID:          entry.ChargeID,
				RootRealizationID: entry.RootRealizationID,
				CustomerID:        entry.CustomerID,
				Currency:          entry.Currency,
				OriginKind:        entry.OriginKind,
			}
		}), nil
	})
}

func (a *adapter) ListActiveSegments(ctx context.Context, input lineage.ListActiveSegmentsInput) ([]lineage.Segment, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]lineage.Segment, error) {
		query := tx.db.CreditRealizationLineageSegment.Query().
			Where(
				creditrealizationlineagesegment.ClosedAtIsNil(),
				creditrealizationlineagesegment.LineageIDIn(input.LineageIDs...),
			).
			Order(creditrealizationlineagesegment.ByCreatedAt())

		if input.State != nil {
			query = query.Where(creditrealizationlineagesegment.StateEQ(*input.State))
		}

		segments, err := query.All(ctx)
		if err != nil {
			return nil, err
		}

		return lo.Map(segments, func(segment *entdb.CreditRealizationLineageSegment, _ int) lineage.Segment {
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

func (a *adapter) CreateSegment(ctx context.Context, input lineage.CreateSegmentInput) error {
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

func mapLineage(entry *entdb.CreditRealizationLineage, _ int) lineage.Lineage {
	return lineage.Lineage{
		ID:                entry.ID,
		ChargeID:          entry.ChargeID,
		RootRealizationID: entry.RootRealizationID,
		CustomerID:        entry.CustomerID,
		Currency:          entry.Currency,
		OriginKind:        entry.OriginKind,
		Segments: lo.Map(entry.Edges.Segments, func(segment *entdb.CreditRealizationLineageSegment, _ int) lineage.Segment {
			return mapSegment(segment)
		}),
	}
}

func mapSegment(segment *entdb.CreditRealizationLineageSegment) lineage.Segment {
	return lineage.Segment{
		ID:                              segment.ID,
		LineageID:                       segment.LineageID,
		Amount:                          segment.Amount,
		State:                           segment.State,
		BackingTransactionGroupID:       segment.BackingTransactionGroupID,
		SourceState:                     segment.SourceState,
		SourceBackingTransactionGroupID: segment.SourceBackingTransactionGroupID,
	}
}
