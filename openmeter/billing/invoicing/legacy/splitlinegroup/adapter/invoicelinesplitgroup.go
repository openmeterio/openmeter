package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/invoicing/legacy/splitlinegroup"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoice"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceline"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoicesplitlinegroup"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceusagebasedlineconfig"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

var _ splitlinegroup.Adapter = (*adapter)(nil)

func (a *adapter) splitLineGroupLinesWithInvoice(namespace string) func(*db.BillingInvoiceLineQuery) {
	return func(q *db.BillingInvoiceLineQuery) {
		q.Where(billinginvoiceline.Namespace(namespace))
		q.WithUsageBasedLine(func(q *db.BillingInvoiceUsageBasedLineConfigQuery) {
			q.Where(billinginvoiceusagebasedlineconfig.Namespace(namespace))
		})
		q.WithBillingInvoice(func(q *db.BillingInvoiceQuery) {
			q.Where(billinginvoice.Namespace(namespace))
		})
	}
}

func (a *adapter) GetSplitLineGroupsForSubscription(ctx context.Context, in billing.GetLinesForSubscriptionInput) ([]splitlinegroup.SplitLineHierarchy, error) {
	if err := in.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]splitlinegroup.SplitLineHierarchy, error) {
		dbGroups, err := tx.db.BillingInvoiceSplitLineGroup.Query().
			Where(billinginvoicesplitlinegroup.Namespace(in.Namespace)).
			Where(billinginvoicesplitlinegroup.SubscriptionID(in.SubscriptionID)).
			WithBillingInvoiceLines(tx.splitLineGroupLinesWithInvoice(in.Namespace)).
			Where(billinginvoicesplitlinegroup.DeletedAtIsNil()).
			All(ctx)
		if err != nil {
			return nil, fmt.Errorf("fetching split line groups: %w", err)
		}

		groups, err := slicesx.MapWithErr(dbGroups, func(dbGroup *db.BillingInvoiceSplitLineGroup) (splitlinegroup.SplitLineHierarchy, error) {
			return tx.mapSplitLineHierarchyFromDB(ctx, dbGroup)
		})
		if err != nil {
			return nil, fmt.Errorf("mapping split line groups: %w", err)
		}

		return groups, nil
	})
}

func (a *adapter) CreateSplitLineGroup(ctx context.Context, input splitlinegroup.CreateSplitLineGroupAdapterInput) (splitlinegroup.SplitLineGroup, error) {
	if err := input.Validate(); err != nil {
		return splitlinegroup.SplitLineGroup{}, billing.ValidationError{
			Err: err,
		}
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (splitlinegroup.SplitLineGroup, error) {
		create := tx.db.BillingInvoiceSplitLineGroup.Create().
			SetNamespace(input.Namespace).
			SetNillableUniqueReferenceID(input.UniqueReferenceID).
			SetName(input.Name).
			SetNillableDescription(input.Description).
			SetMetadata(input.Metadata).
			SetServicePeriodStart(input.ServicePeriod.From.UTC()).
			SetServicePeriodEnd(input.ServicePeriod.To.UTC()).
			SetCurrency(input.Currency).
			SetRatecardDiscounts(&input.RatecardDiscounts).
			SetPrice(input.Price).
			SetNillableFeatureKey(input.FeatureKey)

		if input.Subscription != nil {
			create = create.SetSubscriptionID(input.Subscription.SubscriptionID).
				SetSubscriptionPhaseID(input.Subscription.PhaseID).
				SetSubscriptionItemID(input.Subscription.ItemID).
				SetSubscriptionBillingPeriodFrom(input.Subscription.BillingPeriod.From.In(time.UTC)).
				SetSubscriptionBillingPeriodTo(input.Subscription.BillingPeriod.To.In(time.UTC))
		}

		dbSplitLineGroup, err := create.Save(ctx)
		if err != nil {
			return splitlinegroup.SplitLineGroup{}, err
		}

		return tx.mapSplitLineGroupFromDB(dbSplitLineGroup)
	})
}

func (a *adapter) UpdateSplitLineGroup(ctx context.Context, input splitlinegroup.UpdateSplitLineGroupInput) (splitlinegroup.SplitLineGroup, error) {
	if err := input.Validate(); err != nil {
		return splitlinegroup.SplitLineGroup{}, billing.ValidationError{
			Err: err,
		}
	}

	// TODO[later]: we should consider creating a batch endpoint, but updates for split line groups are rare (e.g. subscription cancellation)
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (splitlinegroup.SplitLineGroup, error) {
		updateQuery := tx.db.BillingInvoiceSplitLineGroup.UpdateOneID(input.ID).
			SetName(input.Name).
			SetOrClearDescription(input.Description).
			SetMetadata(input.Metadata).
			SetServicePeriodStart(input.ServicePeriod.From.UTC()).
			SetServicePeriodEnd(input.ServicePeriod.To.UTC()).
			SetRatecardDiscounts(&input.RatecardDiscounts).
			Where(
				billinginvoicesplitlinegroup.Namespace(input.Namespace),
			)

		dbSplitLineGroup, err := updateQuery.Save(ctx)
		if err != nil {
			return splitlinegroup.SplitLineGroup{}, err
		}

		return tx.mapSplitLineGroupFromDB(dbSplitLineGroup)
	})
}

func (a *adapter) DeleteSplitLineGroup(ctx context.Context, input splitlinegroup.DeleteSplitLineGroupInput) error {
	if err := input.Validate(); err != nil {
		return billing.ValidationError{
			Err: err,
		}
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		nr, err := tx.db.BillingInvoiceSplitLineGroup.Delete().
			Where(
				billinginvoicesplitlinegroup.Namespace(input.Namespace),
				billinginvoicesplitlinegroup.ID(input.ID),
			).Exec(ctx)
		if err != nil {
			return err
		}

		if nr != 1 {
			return billing.NotFoundError{
				Err: fmt.Errorf("split line group not found [id=%s]", input.ID),
			}
		}

		return nil
	})
}

func (a *adapter) GetSplitLineGroup(ctx context.Context, input splitlinegroup.GetSplitLineGroupInput) (splitlinegroup.SplitLineHierarchy, error) {
	if err := input.Validate(); err != nil {
		return splitlinegroup.SplitLineHierarchy{}, billing.ValidationError{
			Err: err,
		}
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (splitlinegroup.SplitLineHierarchy, error) {
		dbSplitLineGroup, err := tx.db.BillingInvoiceSplitLineGroup.Query().
			Where(
				billinginvoicesplitlinegroup.Namespace(input.Namespace),
				billinginvoicesplitlinegroup.ID(input.ID),
			).
			WithBillingInvoiceLines(tx.splitLineGroupLinesWithInvoice(input.Namespace)).
			First(ctx)
		if err != nil {
			if db.IsNotFound(err) {
				return splitlinegroup.SplitLineHierarchy{}, billing.NotFoundError{
					Err: fmt.Errorf("split line group not found [id=%s]", input.ID),
				}
			}

			return splitlinegroup.SplitLineHierarchy{}, err
		}

		return a.mapSplitLineHierarchyFromDB(ctx, dbSplitLineGroup)
	})
}

func (a *adapter) mapSplitLineGroupFromDB(dbSplitLineGroup *db.BillingInvoiceSplitLineGroup) (splitlinegroup.SplitLineGroup, error) {
	if dbSplitLineGroup.Price == nil {
		return splitlinegroup.SplitLineGroup{}, fmt.Errorf("price is required")
	}

	var subscriptionRef *billing.SubscriptionReference
	if dbSplitLineGroup.SubscriptionID != nil || dbSplitLineGroup.SubscriptionPhaseID != nil || dbSplitLineGroup.SubscriptionItemID != nil {
		subscriptionRef = &billing.SubscriptionReference{
			SubscriptionID: lo.FromPtr(dbSplitLineGroup.SubscriptionID),
			PhaseID:        lo.FromPtr(dbSplitLineGroup.SubscriptionPhaseID),
			ItemID:         lo.FromPtr(dbSplitLineGroup.SubscriptionItemID),
			BillingPeriod: timeutil.ClosedPeriod{
				From: lo.FromPtr(dbSplitLineGroup.SubscriptionBillingPeriodFrom).In(time.UTC),
				To:   lo.FromPtr(dbSplitLineGroup.SubscriptionBillingPeriodTo).In(time.UTC),
			},
		}

		if err := subscriptionRef.Validate(); err != nil {
			return splitlinegroup.SplitLineGroup{}, err
		}
	}

	return splitlinegroup.SplitLineGroup{
		NamespacedID: models.NamespacedID{
			Namespace: dbSplitLineGroup.Namespace,
			ID:        dbSplitLineGroup.ID,
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: dbSplitLineGroup.CreatedAt,
			UpdatedAt: dbSplitLineGroup.UpdatedAt,
			DeletedAt: dbSplitLineGroup.DeletedAt,
		},
		SplitLineGroupMutableFields: splitlinegroup.SplitLineGroupMutableFields{
			Name:        dbSplitLineGroup.Name,
			Description: dbSplitLineGroup.Description,
			Metadata:    dbSplitLineGroup.Metadata,

			ServicePeriod: timeutil.ClosedPeriod{
				From: dbSplitLineGroup.ServicePeriodStart.UTC(),
				To:   dbSplitLineGroup.ServicePeriodEnd.UTC(),
			},

			RatecardDiscounts: lo.FromPtr(dbSplitLineGroup.RatecardDiscounts),
		},
		UniqueReferenceID: dbSplitLineGroup.UniqueReferenceID,

		Currency:     dbSplitLineGroup.Currency,
		Price:        dbSplitLineGroup.Price,
		FeatureKey:   dbSplitLineGroup.FeatureKey,
		Subscription: subscriptionRef,
	}, nil
}

func (a *adapter) mapSplitLineHierarchyFromDB(ctx context.Context, dbSplitLineGroup *db.BillingInvoiceSplitLineGroup) (splitlinegroup.SplitLineHierarchy, error) {
	empty := splitlinegroup.SplitLineHierarchy{}

	group, err := a.mapSplitLineGroupFromDB(dbSplitLineGroup)
	if err != nil {
		return empty, err
	}

	mappedLines, err := a.mapSplitLineHierarchyLinesFromDB(ctx, dbSplitLineGroup.ID, dbSplitLineGroup.Edges.BillingInvoiceLines)
	if err != nil {
		return empty, err
	}

	return splitlinegroup.SplitLineHierarchy{
		Group:         group,
		StandardLines: mappedLines.StandardLines,
		GatheringLine: mappedLines.GatheringLine,
	}, nil
}

type mappedSplitLineHierarchyLines struct {
	StandardLines []splitlinegroup.StandardLine
	GatheringLine *splitlinegroup.GatheringLine
}

func (a *adapter) mapSplitLineHierarchyLinesFromDB(ctx context.Context, groupID string, dbLines []*db.BillingInvoiceLine) (mappedSplitLineHierarchyLines, error) {
	out := mappedSplitLineHierarchyLines{
		StandardLines: make([]splitlinegroup.StandardLine, 0, len(dbLines)),
	}

	for _, dbLine := range dbLines {
		if dbLine.Edges.BillingInvoice == nil {
			return mappedSplitLineHierarchyLines{}, fmt.Errorf("billing invoice must be expanded when mapping split line hierarchy lines [line_id=%s]", dbLine.ID)
		}

		switch dbLine.Edges.BillingInvoice.Status {
		case billing.StandardInvoiceStatusGathering:
			if out.GatheringLine != nil {
				return mappedSplitLineHierarchyLines{}, fmt.Errorf("multiple gathering lines found for split line group [group_id=%s]", groupID)
			}

			out.GatheringLine = lo.ToPtr(a.mapSplitLineHierarchyGatheringLineFromDB(ctx, dbLine))
		default:
			out.StandardLines = append(out.StandardLines, a.mapSplitLineHierarchyStandardLineFromDB(ctx, dbLine))
		}
	}

	return out, nil
}

func (a *adapter) mapSplitLineHierarchyStandardLineFromDB(ctx context.Context, dbLine *db.BillingInvoiceLine) splitlinegroup.StandardLine {
	var subscriptionRef *billing.SubscriptionReference

	if dbLine.SubscriptionID != nil && dbLine.SubscriptionPhaseID != nil && dbLine.SubscriptionItemID != nil {
		subscriptionRef = &billing.SubscriptionReference{
			SubscriptionID: lo.FromPtr(dbLine.SubscriptionID),
			PhaseID:        lo.FromPtr(dbLine.SubscriptionPhaseID),
			ItemID:         lo.FromPtr(dbLine.SubscriptionItemID),
			BillingPeriod: timeutil.ClosedPeriod{
				From: lo.FromPtr(dbLine.SubscriptionBillingPeriodFrom),
				To:   lo.FromPtr(dbLine.SubscriptionBillingPeriodTo),
			},
		}
	}

	return splitlinegroup.StandardLine{
		ID: billing.LineID{
			Namespace: dbLine.Namespace,
			ID:        dbLine.ID,
		},
		DeletedAt:   dbLine.DeletedAt,
		Annotations: dbLine.Annotations,
		ManagedBy:   dbLine.ManagedBy,
		Invoice: splitlinegroup.InvoiceHeader{
			ID:        dbLine.Edges.BillingInvoice.ID,
			DeletedAt: dbLine.Edges.BillingInvoice.DeletedAt,
		},
		ServicePeriod: timeutil.ClosedPeriod{
			From: dbLine.PeriodStart.UTC(),
			To:   dbLine.PeriodEnd.UTC(),
		},
		Totals:       totals.FromDB(dbLine),
		Subscription: subscriptionRef,
	}
}

func (a *adapter) mapSplitLineHierarchyGatheringLineFromDB(ctx context.Context, dbLine *db.BillingInvoiceLine) splitlinegroup.GatheringLine {
	var subscriptionRef *billing.SubscriptionReference

	if dbLine.SubscriptionID != nil && dbLine.SubscriptionPhaseID != nil && dbLine.SubscriptionItemID != nil {
		subscriptionRef = &billing.SubscriptionReference{
			SubscriptionID: lo.FromPtr(dbLine.SubscriptionID),
			PhaseID:        lo.FromPtr(dbLine.SubscriptionPhaseID),
			ItemID:         lo.FromPtr(dbLine.SubscriptionItemID),
			BillingPeriod: timeutil.ClosedPeriod{
				From: lo.FromPtr(dbLine.SubscriptionBillingPeriodFrom),
				To:   lo.FromPtr(dbLine.SubscriptionBillingPeriodTo),
			},
		}
	}

	return splitlinegroup.GatheringLine{
		ID: billing.LineID{
			Namespace: dbLine.Namespace,
			ID:        dbLine.ID,
		},
		DeletedAt:   dbLine.DeletedAt,
		Annotations: dbLine.Annotations,
		ManagedBy:   dbLine.ManagedBy,

		Invoice: splitlinegroup.InvoiceHeader{
			ID:        dbLine.Edges.BillingInvoice.ID,
			DeletedAt: dbLine.Edges.BillingInvoice.DeletedAt,
		},
		ServicePeriod: timeutil.ClosedPeriod{
			From: dbLine.PeriodStart.UTC(),
			To:   dbLine.PeriodEnd.UTC(),
		},
		InvoiceAt:    dbLine.InvoiceAt.UTC(),
		Subscription: subscriptionRef,
	}
}

type lineIdToSplitLineHierarchy map[string]*splitlinegroup.SplitLineHierarchy

type splitLineSettableLines interface {
	GetSplitLineGroupID() *string
	GetID() string
	SetSplitLineHierarchy(*splitlinegroup.SplitLineHierarchy)
}

func withSplitLineHierarchyForLines[T splitLineSettableLines](lines []T, hierarchyByLineID lineIdToSplitLineHierarchy) ([]T, error) {
	for _, line := range lines {
		if line.GetSplitLineGroupID() == nil {
			continue
		}

		hierarchy, ok := hierarchyByLineID[line.GetID()]
		if !ok {
			return nil, fmt.Errorf("split line group[%s] for line[%s] not found", *line.GetSplitLineGroupID(), line.GetID())
		}

		line.SetSplitLineHierarchy(hierarchy)
	}

	return lines, nil
}

func (a *adapter) fetchAllSplitLineGroups(ctx context.Context, namespace string, splitLineGroupIDs []string) ([]splitlinegroup.SplitLineHierarchy, error) {
	query := a.db.BillingInvoiceSplitLineGroup.Query().
		Where(
			billinginvoicesplitlinegroup.Namespace(namespace),
			billinginvoicesplitlinegroup.IDIn(splitLineGroupIDs...),
		).
		WithBillingInvoiceLines(a.splitLineGroupLinesWithInvoice(namespace))

	dbSplitLineGroups, err := query.All(ctx)
	if err != nil {
		return nil, err
	}

	return slicesx.MapWithErr(dbSplitLineGroups, func(dbSplitLineGroup *db.BillingInvoiceSplitLineGroup) (splitlinegroup.SplitLineHierarchy, error) {
		return a.mapSplitLineHierarchyFromDB(ctx, dbSplitLineGroup)
	})
}

func (a *adapter) GetSplitLineGroupHeaders(ctx context.Context, input splitlinegroup.GetSplitLineGroupHeadersInput) (splitlinegroup.SplitLineGroupHeaders, error) {
	if err := input.Validate(); err != nil {
		return splitlinegroup.SplitLineGroupHeaders{}, billing.ValidationError{
			Err: err,
		}
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (splitlinegroup.SplitLineGroupHeaders, error) {
		dbSplitLineGroups, err := tx.db.BillingInvoiceSplitLineGroup.Query().
			Where(billinginvoicesplitlinegroup.Namespace(input.Namespace)).
			Where(billinginvoicesplitlinegroup.IDIn(input.SplitLineGroupIDs...)).
			All(ctx)
		if err != nil {
			return splitlinegroup.SplitLineGroupHeaders{}, err
		}

		splitLineGroups, err := slicesx.MapWithErr(dbSplitLineGroups, func(dbSplitLineGroup *db.BillingInvoiceSplitLineGroup) (splitlinegroup.SplitLineGroup, error) {
			return a.mapSplitLineGroupFromDB(dbSplitLineGroup)
		})
		if err != nil {
			return splitlinegroup.SplitLineGroupHeaders{}, err
		}

		return splitLineGroups, nil
	})
}
