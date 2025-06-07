package billingadapter

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoicesplitlinegroup"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

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
			SetPeriodStart(input.Period.Start.UTC()).
			SetPeriodEnd(input.Period.End.UTC()).
			SetCurrency(input.Currency).
			SetRatecardDiscounts(&input.RatecardDiscounts).
			SetPrice(input.Price).
			SetNillableTaxConfig(input.TaxConfig)

		if input.Subscription != nil {
			create = create.SetSubscriptionID(input.Subscription.SubscriptionID).
				SetSubscriptionPhaseID(input.Subscription.PhaseID).
				SetSubscriptionItemID(input.Subscription.ItemID)
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
			SetOrClearDeletedAt(input.DeletedAt).
			SetOrClearUniqueReferenceID(input.UniqueReferenceID).
			SetName(input.Name).
			SetOrClearDescription(input.Description).
			SetPeriodStart(input.Period.Start.UTC()).
			SetPeriodEnd(input.Period.End.UTC()).
			SetRatecardDiscounts(&input.RatecardDiscounts).
			SetPrice(input.Price).
			SetOrClearTaxConfig(input.TaxConfig).
			Where(
				billinginvoicesplitlinegroup.Namespace(input.Namespace),
			)

		if input.Subscription != nil {
			updateQuery = updateQuery.SetSubscriptionID(input.Subscription.SubscriptionID).
				SetSubscriptionPhaseID(input.Subscription.PhaseID).
				SetSubscriptionItemID(input.Subscription.ItemID)
		} else {
			updateQuery = updateQuery.ClearSubscriptionID().
				ClearSubscriptionPhaseID().
				ClearSubscriptionItemID()
		}

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
				q.WithBillingInvoice()
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
		}

		if err := subscriptionRef.Validate(); err != nil {
			return billing.SplitLineGroup{}, err
		}
	}

	return billing.SplitLineGroup{
		ManagedModel: models.ManagedModel{
			CreatedAt: dbSplitLineGroup.CreatedAt,
			UpdatedAt: dbSplitLineGroup.UpdatedAt,
			DeletedAt: dbSplitLineGroup.DeletedAt,
		},
		SplitLineGroupBase: billing.SplitLineGroupBase{
			Namespace:         dbSplitLineGroup.Namespace,
			UniqueReferenceID: dbSplitLineGroup.UniqueReferenceID,
			Name:              dbSplitLineGroup.Name,
			Description:       dbSplitLineGroup.Description,
			Period: billing.Period{
				Start: dbSplitLineGroup.PeriodStart.UTC(),
				End:   dbSplitLineGroup.PeriodEnd.UTC(),
			},
			Currency:          dbSplitLineGroup.Currency,
			RatecardDiscounts: lo.FromPtr(dbSplitLineGroup.RatecardDiscounts),
			Price:             dbSplitLineGroup.Price,
			TaxConfig:         lo.EmptyableToPtr(dbSplitLineGroup.TaxConfig),

			Subscription: subscriptionRef,
		},
		ID: dbSplitLineGroup.ID,
	}, nil
}

func (a *adapter) mapSplitLineHierarchyFromDB(ctx context.Context, dbSplitLineGroup *db.BillingInvoiceSplitLineGroup) (billing.SplitLineHierarchy, error) {
	empty := billing.SplitLineHierarchy{}

	group, err := a.mapSplitLineGroupFromDB(dbSplitLineGroup)
	if err != nil {
		return empty, err
	}

	mappedLines, err := slicesx.MapWithErr(dbSplitLineGroup.Edges.BillingInvoiceLines, func(dbLine *db.BillingInvoiceLine) (billing.LineWithInvoiceHeader, error) {
		line, err := a.mapInvoiceLineWithoutReferences(dbLine)
		if err != nil {
			return billing.LineWithInvoiceHeader{}, err
		}

		return billing.LineWithInvoiceHeader{
			Line:    &line,
			Invoice: a.mapInvoiceBaseFromDB(ctx, dbLine.Edges.BillingInvoice),
		}, nil
	})
	if err != nil {
		return empty, err
	}

	return billing.SplitLineHierarchy{
		Group: group,
		Lines: mappedLines,
	}, nil
}

// expandSplitLineHierarchy expands the given lines with their progressive line hierarchy
// This is done by fetching all the lines that are children of the given lines parent lines and then building
// the hierarchy.
func (a *adapter) expandSplitLineHierarchy(ctx context.Context, namespace string, lines []*billing.Line) ([]*billing.Line, error) {
	// Let's collect all the lines with a parent line id set

	lineToGroupIDs := map[string]string{}

	for _, line := range lines {
		if line.SplitLineGroupID != nil {
			lineToGroupIDs[line.ID] = *line.SplitLineGroupID
		}
	}

	if len(lineToGroupIDs) == 0 {
		return lines, nil
	}

	splitLineGroups, err := a.fetchAllSplitLineGroups(ctx, namespace, lo.Values(lineToGroupIDs))
	if err != nil {
		return nil, err
	}

	// Let's build the return values
	hierarchyByLineID := map[string]*billing.SplitLineHierarchy{}
	for _, splitLineGroup := range splitLineGroups {
		for _, line := range splitLineGroup.Lines {
			hierarchyByLineID[line.Line.ID] = &splitLineGroup
		}
	}

	for _, line := range lines {
		if line.SplitLineGroupID == nil {
			continue
		}

		hierarchy, ok := hierarchyByLineID[line.ID]
		if !ok {
			return nil, fmt.Errorf("split line group[%s] for line[%s] not found", *line.SplitLineGroupID, line.ID)
		}

		line.SplitLineHierarchy = hierarchy
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
			q.WithBillingInvoice() // TODO[later]: we can consider loading this in a separate query, might be more efficient
		})

	dbSplitLineGroups, err := query.All(ctx)
	if err != nil {
		return nil, err
	}

	return slicesx.MapWithErr(dbSplitLineGroups, func(dbSplitLineGroup *db.BillingInvoiceSplitLineGroup) (billing.SplitLineHierarchy, error) {
		return a.mapSplitLineHierarchyFromDB(ctx, dbSplitLineGroup)
	})
}
