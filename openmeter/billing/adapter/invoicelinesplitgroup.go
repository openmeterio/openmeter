package billingadapter

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoicesplitlinegroup"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

var _ billing.InvoiceSplitLineGroupAdapter = (*adapter)(nil)

func (a *adapter) CreateSplitLineGroup(ctx context.Context, input billing.CreateSplitLineGroupAdapterInput) (billing.SplitLineGroup, error) {
	if err := input.Validate(); err != nil {
		return billing.SplitLineGroup{}, billing.ValidationError{
			Err: err,
		}
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (billing.SplitLineGroup, error) {
		create := tx.db.BillingInvoiceSplitLineGroup.Create().
			SetNamespace(input.Namespace).
			SetNillableUniqueReferenceID(input.UniqueReferenceID).
			SetName(input.Name).
			SetNillableDescription(input.Description).
			SetMetadata(input.Metadata).
			SetServicePeriodStart(input.ServicePeriod.Start.UTC()).
			SetServicePeriodEnd(input.ServicePeriod.End.UTC()).
			SetCurrency(input.Currency).
			SetRatecardDiscounts(&input.RatecardDiscounts).
			SetPrice(&input.Price).
			SetNillableTaxConfig(input.TaxConfig).
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
			return billing.SplitLineGroup{}, err
		}

		return tx.mapSplitLineGroupFromDB(dbSplitLineGroup)
	})
}

func (a *adapter) UpdateSplitLineGroup(ctx context.Context, input billing.UpdateSplitLineGroupInput) (billing.SplitLineGroup, error) {
	if err := input.Validate(); err != nil {
		return billing.SplitLineGroup{}, billing.ValidationError{
			Err: err,
		}
	}

	// TODO[later]: we should consider creating a batch endpoint, but updates for split line groups are rare (e.g. subscription cancellation)
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (billing.SplitLineGroup, error) {
		updateQuery := tx.db.BillingInvoiceSplitLineGroup.UpdateOneID(input.ID).
			SetName(input.Name).
			SetOrClearDescription(input.Description).
			SetMetadata(input.Metadata).
			SetServicePeriodStart(input.ServicePeriod.Start.UTC()).
			SetServicePeriodEnd(input.ServicePeriod.End.UTC()).
			SetRatecardDiscounts(&input.RatecardDiscounts).
			SetOrClearTaxConfig(input.TaxConfig).
			Where(
				billinginvoicesplitlinegroup.Namespace(input.Namespace),
			)

		dbSplitLineGroup, err := updateQuery.Save(ctx)
		if err != nil {
			return billing.SplitLineGroup{}, err
		}

		return tx.mapSplitLineGroupFromDB(dbSplitLineGroup)
	})
}

func (a *adapter) DeleteSplitLineGroup(ctx context.Context, input billing.DeleteSplitLineGroupInput) error {
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

func (a *adapter) GetSplitLineGroup(ctx context.Context, input billing.GetSplitLineGroupInput) (billing.SplitLineHierarchy, error) {
	if err := input.Validate(); err != nil {
		return billing.SplitLineHierarchy{}, billing.ValidationError{
			Err: err,
		}
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (billing.SplitLineHierarchy, error) {
		dbSplitLineGroup, err := tx.db.BillingInvoiceSplitLineGroup.Query().
			Where(
				billinginvoicesplitlinegroup.Namespace(input.Namespace),
				billinginvoicesplitlinegroup.ID(input.ID),
			).
			WithBillingInvoiceLines(func(q *db.BillingInvoiceLineQuery) {
				a.expandLineItems(q)
				q.WithBillingInvoice(func(q *db.BillingInvoiceQuery) {
					q.WithBillingWorkflowConfig()
				})
			}).
			First(ctx)
		if err != nil {
			if db.IsNotFound(err) {
				return billing.SplitLineHierarchy{}, billing.NotFoundError{
					Err: fmt.Errorf("split line group not found [id=%s]", input.ID),
				}
			}

			return billing.SplitLineHierarchy{}, err
		}

		return a.mapSplitLineHierarchyFromDB(ctx, dbSplitLineGroup)
	})
}

func (a *adapter) mapSplitLineGroupFromDB(dbSplitLineGroup *db.BillingInvoiceSplitLineGroup) (billing.SplitLineGroup, error) {
	if dbSplitLineGroup.Price == nil {
		return billing.SplitLineGroup{}, fmt.Errorf("price is required")
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
			return billing.SplitLineGroup{}, err
		}
	}

	if dbSplitLineGroup.Price == nil {
		return billing.SplitLineGroup{}, fmt.Errorf("price is required")
	}

	return billing.SplitLineGroup{
		NamespacedID: models.NamespacedID{
			Namespace: dbSplitLineGroup.Namespace,
			ID:        dbSplitLineGroup.ID,
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: dbSplitLineGroup.CreatedAt,
			UpdatedAt: dbSplitLineGroup.UpdatedAt,
			DeletedAt: dbSplitLineGroup.DeletedAt,
		},
		SplitLineGroupMutableFields: billing.SplitLineGroupMutableFields{
			Name:        dbSplitLineGroup.Name,
			Description: dbSplitLineGroup.Description,
			Metadata:    dbSplitLineGroup.Metadata,

			ServicePeriod: billing.Period{
				Start: dbSplitLineGroup.ServicePeriodStart.UTC(),
				End:   dbSplitLineGroup.ServicePeriodEnd.UTC(),
			},

			RatecardDiscounts: lo.FromPtr(dbSplitLineGroup.RatecardDiscounts),

			TaxConfig: lo.EmptyableToPtr(dbSplitLineGroup.TaxConfig),
		},
		UniqueReferenceID: dbSplitLineGroup.UniqueReferenceID,

		Currency:     dbSplitLineGroup.Currency,
		Price:        *dbSplitLineGroup.Price,
		FeatureKey:   dbSplitLineGroup.FeatureKey,
		Subscription: subscriptionRef,
	}, nil
}

func (a *adapter) mapSplitLineHierarchyFromDB(ctx context.Context, dbSplitLineGroup *db.BillingInvoiceSplitLineGroup) (billing.SplitLineHierarchy, error) {
	empty := billing.SplitLineHierarchy{}

	group, err := a.mapSplitLineGroupFromDB(dbSplitLineGroup)
	if err != nil {
		return empty, err
	}

	mappedLines, err := a.mapSplitLineHierarchyLinesFromDB(ctx, dbSplitLineGroup.Edges.BillingInvoiceLines)
	if err != nil {
		return empty, err
	}

	return billing.SplitLineHierarchy{
		Group: group,
		Lines: mappedLines,
	}, nil
}

func (a *adapter) mapSplitLineHierarchyLinesFromDB(ctx context.Context, dbLines []*db.BillingInvoiceLine) ([]billing.LineWithInvoiceHeader, error) {
	return slicesx.MapWithErr(dbLines, func(dbLine *db.BillingInvoiceLine) (billing.LineWithInvoiceHeader, error) {
		if dbLine.Edges.BillingInvoice == nil {
			return billing.LineWithInvoiceHeader{}, fmt.Errorf("billing invoice must be expanded when mapping split line hierarchy lines [id=%s]", dbLine.ID)
		}

		switch dbLine.Edges.BillingInvoice.Status {
		case billing.StandardInvoiceStatusGathering:
			return a.mapSplitLineHierarchyGatheringLineFromDB(ctx, dbLine)
		default:
			return a.mapSplitLineHierarchyStandardLineFromDB(ctx, dbLine)
		}
	})
}

func (a *adapter) mapSplitLineHierarchyStandardLineFromDB(ctx context.Context, dbLine *db.BillingInvoiceLine) (billing.LineWithInvoiceHeader, error) {
	line, err := a.mapStandardInvoiceLineWithoutReferences(dbLine)
	if err != nil {
		return billing.LineWithInvoiceHeader{}, err
	}

	invoice, err := a.mapStandardInvoiceFromDB(ctx, dbLine.Edges.BillingInvoice, billing.StandardInvoiceExpands{})
	if err != nil {
		return billing.LineWithInvoiceHeader{}, err
	}

	return billing.NewLineWithInvoiceHeader(billing.StandardLineWithInvoiceHeader{
		Line:    line,
		Invoice: invoice,
	}), nil
}

func (a *adapter) mapSplitLineHierarchyGatheringLineFromDB(ctx context.Context, dbLine *db.BillingInvoiceLine) (billing.LineWithInvoiceHeader, error) {
	line, err := a.mapGatheringInvoiceLineFromDB(dbLine.Edges.BillingInvoice.SchemaLevel, dbLine)
	if err != nil {
		return billing.LineWithInvoiceHeader{}, err
	}

	invoice, err := a.mapGatheringInvoiceFromDB(ctx, dbLine.Edges.BillingInvoice, billing.GatheringInvoiceExpands{})
	if err != nil {
		return billing.LineWithInvoiceHeader{}, err
	}

	return billing.NewLineWithInvoiceHeader(billing.GatheringLineWithInvoiceHeader{
		Line:    line,
		Invoice: invoice,
	}), nil
}

type lineIdToSplitLineHierarchy map[string]*billing.SplitLineHierarchy

// expandSplitLineHierarchy expands the given lines with their progressive line hierarchy
// This is done by fetching all the lines that are children of the given lines parent lines and then building
// the hierarchy.
func (a *adapter) expandSplitLineHierarchy(ctx context.Context, namespace string, lines []billing.GenericInvoiceLine) (lineIdToSplitLineHierarchy, error) {
	// Let's collect all the lines with a parent line id set

	lineToGroupIDs := map[string]string{}

	for _, line := range lines {
		if line.GetSplitLineGroupID() != nil {
			lineToGroupIDs[line.GetID()] = *line.GetSplitLineGroupID()
		}
	}

	if len(lineToGroupIDs) == 0 {
		return lineIdToSplitLineHierarchy{}, nil
	}

	splitLineGroups, err := a.fetchAllSplitLineGroups(ctx, namespace, lo.Values(lineToGroupIDs))
	if err != nil {
		return nil, err
	}

	// Let's build the return values
	hierarchyByLineID := map[string]*billing.SplitLineHierarchy{}
	for _, splitLineGroup := range splitLineGroups {
		for _, line := range splitLineGroup.Lines {
			hierarchyByLineID[line.Line.GetID()] = &splitLineGroup
		}
	}

	return hierarchyByLineID, nil
}

type splitLineSettableLines interface {
	GetSplitLineGroupID() *string
	GetID() string
	SetSplitLineHierarchy(*billing.SplitLineHierarchy)
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

func (a *adapter) fetchAllSplitLineGroups(ctx context.Context, namespace string, splitLineGroupIDs []string) ([]billing.SplitLineHierarchy, error) {
	query := a.db.BillingInvoiceSplitLineGroup.Query().
		Where(
			billinginvoicesplitlinegroup.Namespace(namespace),
			billinginvoicesplitlinegroup.IDIn(splitLineGroupIDs...),
		).
		WithBillingInvoiceLines(func(q *db.BillingInvoiceLineQuery) {
			a.expandLineItems(q)
			q.WithBillingInvoice(func(q *db.BillingInvoiceQuery) {
				q.WithBillingWorkflowConfig()
			}) // TODO[later]: we can consider loading this in a separate query, might be more efficient
		})

	dbSplitLineGroups, err := query.All(ctx)
	if err != nil {
		return nil, err
	}

	return slicesx.MapWithErr(dbSplitLineGroups, func(dbSplitLineGroup *db.BillingInvoiceSplitLineGroup) (billing.SplitLineHierarchy, error) {
		return a.mapSplitLineHierarchyFromDB(ctx, dbSplitLineGroup)
	})
}

func (a *adapter) GetSplitLineGroupHeaders(ctx context.Context, input billing.GetSplitLineGroupHeadersInput) (billing.SplitLineGroupHeaders, error) {
	if err := input.Validate(); err != nil {
		return billing.SplitLineGroupHeaders{}, billing.ValidationError{
			Err: err,
		}
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (billing.SplitLineGroupHeaders, error) {
		dbSplitLineGroups, err := tx.db.BillingInvoiceSplitLineGroup.Query().
			Where(billinginvoicesplitlinegroup.Namespace(input.Namespace)).
			Where(billinginvoicesplitlinegroup.IDIn(input.SplitLineGroupIDs...)).
			All(ctx)
		if err != nil {
			return billing.SplitLineGroupHeaders{}, err
		}

		splitLineGroups, err := slicesx.MapWithErr(dbSplitLineGroups, func(dbSplitLineGroup *db.BillingInvoiceSplitLineGroup) (billing.SplitLineGroup, error) {
			return a.mapSplitLineGroupFromDB(dbSplitLineGroup)
		})
		if err != nil {
			return billing.SplitLineGroupHeaders{}, err
		}

		return splitLineGroups, nil
	})
}
