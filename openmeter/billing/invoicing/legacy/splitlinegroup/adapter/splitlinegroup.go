package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/invoicing/legacy/splitlinegroup"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoice"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceline"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoicesplitlinegroup"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func (a *adapter) splitLineGroupLinesWithInvoice(namespace string) func(*db.BillingInvoiceLineQuery) {
	return func(q *db.BillingInvoiceLineQuery) {
		q.Where(billinginvoiceline.Namespace(namespace))
		q.WithBillingInvoice(func(q *db.BillingInvoiceQuery) {
			q.Where(billinginvoice.Namespace(namespace))
		})
	}
}

func (a *adapter) GetSplitLineGroupsForSubscription(ctx context.Context, in splitlinegroup.GetSplitLineGroupsForSubscriptionInput) ([]splitlinegroup.SplitLineHierarchy, error) {
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
			return tx.mapSplitLineHierarchyFromDB(dbGroup)
		})
		if err != nil {
			return nil, fmt.Errorf("mapping split line groups: %w", err)
		}

		return groups, nil
	})
}

func (a *adapter) CreateSplitLineGroup(ctx context.Context, input splitlinegroup.SplitLineGroupCreate) (splitlinegroup.SplitLineGroup, error) {
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

func (a *adapter) UpdateSplitLineGroup(ctx context.Context, input splitlinegroup.SplitLineGroupUpdate) (splitlinegroup.SplitLineGroup, error) {
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

func (a *adapter) DeleteSplitLineGroup(ctx context.Context, input models.NamespacedID) error {
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

func (a *adapter) GetSplitLineGroup(ctx context.Context, input models.NamespacedID) (splitlinegroup.SplitLineHierarchy, error) {
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

		return a.mapSplitLineHierarchyFromDB(dbSplitLineGroup)
	})
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
		return a.mapSplitLineHierarchyFromDB(dbSplitLineGroup)
	})
}

func (a *adapter) GetSplitLineGroups(ctx context.Context, input splitlinegroup.GetSplitLineGroupsInput) ([]splitlinegroup.SplitLineHierarchy, error) {
	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{Err: err}
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]splitlinegroup.SplitLineHierarchy, error) {
		return tx.fetchAllSplitLineGroups(ctx, input.Namespace, input.SplitLineGroupIDs)
	})
}
