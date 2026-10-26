package billingadapter

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoice"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceline"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoicesplitlinegroup"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

var _ billing.InvoiceSplitLineGroupAdapter = (*adapter)(nil)

func (a *adapter) splitLineGroupLinesWithInvoice(namespace string) func(*db.BillingInvoiceLineQuery) {
	return func(q *db.BillingInvoiceLineQuery) {
		q.Where(billinginvoiceline.Namespace(namespace))
		q.WithBillingInvoice(func(q *db.BillingInvoiceQuery) {
			q.Where(billinginvoice.Namespace(namespace))
		})
	}
}

func (a *adapter) GetSplitLineGroupsForSubscription(ctx context.Context, in billing.GetLinesForSubscriptionInput) ([]billing.SplitLineHierarchy, error) {
	if err := in.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]billing.SplitLineHierarchy, error) {
		dbGroups, err := tx.db.BillingInvoiceSplitLineGroup.Query().
			Where(billinginvoicesplitlinegroup.Namespace(in.Namespace)).
			Where(billinginvoicesplitlinegroup.SubscriptionID(in.SubscriptionID)).
			WithBillingInvoiceLines(tx.splitLineGroupLinesWithInvoice(in.Namespace)).
			Where(billinginvoicesplitlinegroup.DeletedAtIsNil()).
			All(ctx)
		if err != nil {
			return nil, fmt.Errorf("fetching split line groups: %w", err)
		}

		groups, err := slicesx.MapWithErr(dbGroups, func(dbGroup *db.BillingInvoiceSplitLineGroup) (billing.SplitLineHierarchy, error) {
			return tx.mapSplitLineHierarchyFromDB(dbGroup)
		})
		if err != nil {
			return nil, fmt.Errorf("mapping split line groups: %w", err)
		}

		return groups, nil
	})
}

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
			SetServicePeriodStart(input.ServicePeriod.From.UTC()).
			SetServicePeriodEnd(input.ServicePeriod.To.UTC()).
			SetRatecardDiscounts(&input.RatecardDiscounts).
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
			WithBillingInvoiceLines(tx.splitLineGroupLinesWithInvoice(input.Namespace)).
			First(ctx)
		if err != nil {
			if db.IsNotFound(err) {
				return billing.SplitLineHierarchy{}, billing.NotFoundError{
					Err: fmt.Errorf("split line group not found [id=%s]", input.ID),
				}
			}

			return billing.SplitLineHierarchy{}, err
		}

		return a.mapSplitLineHierarchyFromDB(dbSplitLineGroup)
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

func (a *adapter) mapSplitLineHierarchyFromDB(dbSplitLineGroup *db.BillingInvoiceSplitLineGroup) (billing.SplitLineHierarchy, error) {
	empty := billing.SplitLineHierarchy{}

	group, err := a.mapSplitLineGroupFromDB(dbSplitLineGroup)
	if err != nil {
		return empty, err
	}

	mappedLines, err := a.mapSplitLineHierarchyLinesFromDB(dbSplitLineGroup.Edges.BillingInvoiceLines)
	if err != nil {
		return empty, err
	}

	return billing.SplitLineHierarchy{
		Group: group,
		Lines: mappedLines,
	}, nil
}

func (a *adapter) mapSplitLineHierarchyLinesFromDB(dbLines []*db.BillingInvoiceLine) ([]billing.SplitLineHierarchyLine, error) {
	return slicesx.MapWithErr(dbLines, func(dbLine *db.BillingInvoiceLine) (billing.SplitLineHierarchyLine, error) {
		dbInvoice := dbLine.Edges.BillingInvoice
		if dbInvoice == nil {
			return billing.SplitLineHierarchyLine{}, fmt.Errorf("billing invoice must be expanded when mapping split line hierarchy lines [id=%s]", dbLine.ID)
		}

		invoiceType := billing.InvoiceTypeStandard
		if dbInvoice.Status == billing.StandardInvoiceStatusGathering {
			invoiceType = billing.InvoiceTypeGathering
		}

		if dbLine.Type != billing.InvoiceLineAdapterTypeUsageBased {
			if invoiceType == billing.InvoiceTypeGathering {
				return billing.SplitLineHierarchyLine{}, fmt.Errorf("only usage based lines can be gathering invoice lines [line_id=%s]", dbLine.ID)
			}

			return billing.SplitLineHierarchyLine{}, fmt.Errorf("only usage based lines can be top level lines [line_id=%s]", dbLine.ID)
		}

		var subscription *billing.SubscriptionReference
		if dbLine.SubscriptionID != nil && dbLine.SubscriptionPhaseID != nil && dbLine.SubscriptionItemID != nil {
			subscription = &billing.SubscriptionReference{
				SubscriptionID: *dbLine.SubscriptionID,
				PhaseID:        *dbLine.SubscriptionPhaseID,
				ItemID:         *dbLine.SubscriptionItemID,
				BillingPeriod: timeutil.ClosedPeriod{
					From: lo.FromPtr(dbLine.SubscriptionBillingPeriodFrom).UTC(),
					To:   lo.FromPtr(dbLine.SubscriptionBillingPeriodTo).UTC(),
				},
			}
		}

		return billing.SplitLineHierarchyLine{
			LineID: billing.LineID{
				Namespace: dbLine.Namespace,
				ID:        dbLine.ID,
			},
			Invoice: billing.SplitLineHierarchyInvoice{
				InvoiceID: billing.InvoiceID{
					Namespace: dbInvoice.Namespace,
					ID:        dbInvoice.ID,
				},
				Type:      invoiceType,
				DeletedAt: convert.TimePtrIn(dbInvoice.DeletedAt, time.UTC),
				Status:    dbInvoice.Status,
			},
			DeletedAt: convert.TimePtrIn(dbLine.DeletedAt, time.UTC),
			ServicePeriod: timeutil.ClosedPeriod{
				From: dbLine.PeriodStart.UTC(),
				To:   dbLine.PeriodEnd.UTC(),
			},
			ManagedBy:    dbLine.ManagedBy,
			Annotations:  dbLine.Annotations,
			Subscription: subscription,
			InvoiceAt:    dbLine.InvoiceAt.UTC(),
			Amount:       dbLine.Amount,
			ChargesTotal: dbLine.ChargesTotal,
		}, nil
	})
}

func (a *adapter) fetchAllSplitLineGroups(ctx context.Context, namespace string, splitLineGroupIDs []string) ([]billing.SplitLineHierarchy, error) {
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

	return slicesx.MapWithErr(dbSplitLineGroups, func(dbSplitLineGroup *db.BillingInvoiceSplitLineGroup) (billing.SplitLineHierarchy, error) {
		return a.mapSplitLineHierarchyFromDB(dbSplitLineGroup)
	})
}

func (a *adapter) GetSplitLineGroups(ctx context.Context, input billing.GetSplitLineGroupsInput) ([]billing.SplitLineHierarchy, error) {
	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{Err: err}
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]billing.SplitLineHierarchy, error) {
		return tx.fetchAllSplitLineGroups(ctx, input.Namespace, input.SplitLineGroupIDs)
	})
}
