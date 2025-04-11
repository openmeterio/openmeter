package billingadapter

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceline"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type mapInvoiceLineFromDBInput struct {
	lines          []*db.BillingInvoiceLine
	includeDeleted bool
}

func (a *adapter) mapInvoiceLineFromDB(ctx context.Context, in mapInvoiceLineFromDBInput) ([]*billing.Line, error) {
	pendingParentIDs := make([]string, 0, len(in.lines))
	resolvedChildrenOfIDs := make(map[string]struct{}, len(in.lines))

	for _, line := range in.lines {
		if line.ParentLineID != nil {
			pendingParentIDs = append(pendingParentIDs, *line.ParentLineID)
		}

		if line.Status != billing.InvoiceLineStatusDetailed {
			resolvedChildrenOfIDs[line.ID] = struct{}{}
		}
	}

	// NOTE: Given that the invoice lines can be in parent-child relationship we might fetch
	// duplicate lines, so we need to deduplicate them.
	//
	// We cannot get around this limitation, as a parent line might have more children than the ones we have
	// saved.
	references, err := a.fetchInvoiceLineNewReferences(ctx, pendingParentIDs, lo.Keys(resolvedChildrenOfIDs), in.includeDeleted)
	if err != nil {
		return nil, err
	}

	references = append(references, in.lines...)

	mappedEntities := make(map[string]*billing.Line, len(references))

	for _, dbLine := range references {
		if _, ok := mappedEntities[dbLine.ID]; ok {
			continue
		}

		entity, err := a.mapInvoiceLineWithoutReferences(dbLine)
		if err != nil {
			return nil, err
		}

		mappedEntities[dbLine.ID] = &entity
	}

	for _, entity := range mappedEntities {
		if entity.ParentLineID != nil {
			parent, ok := mappedEntities[*entity.ParentLineID]
			if !ok {
				// We don't care about the parent if it's not loaded as it might be too deep
				continue
			}

			entity.ParentLine = parent
			// We only add children references if we know that those has been properly resolved
			if _, ok := resolvedChildrenOfIDs[parent.ID]; ok {
				parent.Children.Append(entity)
			}
		}
	}

	result := make([]*billing.Line, 0, len(mappedEntities))
	for _, dbEntity := range in.lines {
		entity, ok := mappedEntities[dbEntity.ID]
		if !ok {
			return nil, fmt.Errorf("missing entity[%s]", dbEntity.ID)
		}

		// Given that we did the one level deep resolution, all the children should be resolved
		// if it's not present it means that the line has no children, but before we only rely on Append
		// to set the Present flag implicitly.
		if !entity.Children.IsPresent() {
			entity.Children = billing.NewLineChildren(nil)
		}

		entity.SaveDBSnapshot()

		result = append(result, entity)
	}

	return result, nil
}

func (a *adapter) fetchInvoiceLineNewReferences(ctx context.Context, parentIDs []string, childrenOf []string, includeDeletedLines bool) ([]*db.BillingInvoiceLine, error) {
	if len(parentIDs) == 0 && len(childrenOf) == 0 {
		return nil, nil
	}

	query := a.db.BillingInvoiceLine.Query()

	query = a.expandLineItems(query)

	predicates := make([]predicate.BillingInvoiceLine, 0, 2)
	if len(parentIDs) > 0 {
		predicates = append(predicates, billinginvoiceline.IDIn(lo.Uniq(parentIDs)...))
	}

	if len(childrenOf) > 0 {
		predicates = append(predicates, billinginvoiceline.ParentLineIDIn(lo.Uniq(childrenOf)...))
	}

	query = query.Where(billinginvoiceline.Or(predicates...))

	if !includeDeletedLines {
		query = query.Where(billinginvoiceline.DeletedAtIsNil())
	}

	return query.All(ctx)
}

func (a *adapter) mapInvoiceLineWithoutReferences(dbLine *db.BillingInvoiceLine) (billing.Line, error) {
	invoiceLine := billing.Line{
		LineBase: billing.LineBase{
			Namespace: dbLine.Namespace,
			ID:        dbLine.ID,

			CreatedAt: dbLine.CreatedAt.In(time.UTC),
			UpdatedAt: dbLine.UpdatedAt.In(time.UTC),
			DeletedAt: convert.TimePtrIn(dbLine.DeletedAt, time.UTC),

			Metadata:  dbLine.Metadata,
			InvoiceID: dbLine.InvoiceID,
			Status:    dbLine.Status,
			ManagedBy: dbLine.ManagedBy,

			Period: billing.Period{
				Start: dbLine.PeriodStart.In(time.UTC),
				End:   dbLine.PeriodEnd.In(time.UTC),
			},

			ParentLineID:           dbLine.ParentLineID,
			ChildUniqueReferenceID: dbLine.ChildUniqueReferenceID,

			InvoiceAt: dbLine.InvoiceAt.In(time.UTC),

			Name:        dbLine.Name,
			Description: dbLine.Description,

			Type:     dbLine.Type,
			Currency: dbLine.Currency,

			TaxConfig:         lo.EmptyableToPtr(dbLine.TaxConfig),
			RateCardDiscounts: lo.FromPtr(dbLine.RatecardDiscounts),
			Totals: billing.Totals{
				Amount:              dbLine.Amount,
				ChargesTotal:        dbLine.ChargesTotal,
				DiscountsTotal:      dbLine.DiscountsTotal,
				TaxesInclusiveTotal: dbLine.TaxesInclusiveTotal,
				TaxesExclusiveTotal: dbLine.TaxesExclusiveTotal,
				TaxesTotal:          dbLine.TaxesTotal,
				Total:               dbLine.Total,
			},
			ExternalIDs: billing.LineExternalIDs{
				Invoicing: lo.FromPtr(dbLine.InvoicingAppExternalID),
			},
		},
	}

	if dbLine.SubscriptionID != nil && dbLine.SubscriptionPhaseID != nil && dbLine.SubscriptionItemID != nil {
		invoiceLine.Subscription = &billing.SubscriptionReference{
			SubscriptionID: *dbLine.SubscriptionID,
			PhaseID:        *dbLine.SubscriptionPhaseID,
			ItemID:         *dbLine.SubscriptionItemID,
		}
	}

	switch dbLine.Type {
	case billing.InvoiceLineTypeFee:
		invoiceLine.FlatFee = &billing.FlatFeeLine{
			ConfigID:      dbLine.Edges.FlatFeeLine.ID,
			PerUnitAmount: dbLine.Edges.FlatFeeLine.PerUnitAmount,
			Quantity:      lo.FromPtr(dbLine.Quantity),
			Category:      dbLine.Edges.FlatFeeLine.Category,
			PaymentTerm:   dbLine.Edges.FlatFeeLine.PaymentTerm,
		}
	case billing.InvoiceLineTypeUsageBased:
		ubpLine := dbLine.Edges.UsageBasedLine
		if ubpLine == nil {
			return invoiceLine, fmt.Errorf("manual usage based line is missing")
		}
		invoiceLine.UsageBased = &billing.UsageBasedLine{
			ConfigID:                     ubpLine.ID,
			FeatureKey:                   ubpLine.FeatureKey,
			Price:                        ubpLine.Price,
			Quantity:                     dbLine.Quantity,
			MeteredQuantity:              ubpLine.MeteredQuantity,
			PreLinePeriodQuantity:        ubpLine.PreLinePeriodQuantity,
			MeteredPreLinePeriodQuantity: ubpLine.MeteredPreLinePeriodQuantity,
		}

		// Before usage discounts were introduced, we didn't have the metered quantity
		if ubpLine.MeteredQuantity == nil && dbLine.Quantity != nil {
			invoiceLine.UsageBased.MeteredQuantity = lo.ToPtr(dbLine.Quantity.Copy())
		}

		// Before usage discounts were introduced, we didn't have the metered pre-line period quantity
		if ubpLine.MeteredPreLinePeriodQuantity == nil && ubpLine.PreLinePeriodQuantity != nil {
			invoiceLine.UsageBased.MeteredPreLinePeriodQuantity = lo.ToPtr(ubpLine.PreLinePeriodQuantity.Copy())
		}

	default:
		return invoiceLine, fmt.Errorf("unsupported line type[%s]: %s", dbLine.ID, dbLine.Type)
	}

	if len(dbLine.Edges.LineUsageDiscounts) > 0 {
		discounts, err := slicesx.MapWithErr(dbLine.Edges.LineUsageDiscounts, a.mapInvoiceLineUsageDiscountFromDB)
		if err != nil {
			return invoiceLine, fmt.Errorf("mapping invoice line usage discounts[%s] failed: %w", dbLine.ID, err)
		}

		invoiceLine.Discounts.Usage = discounts
	}

	if len(dbLine.Edges.LineAmountDiscounts) > 0 {
		discounts, err := slicesx.MapWithErr(dbLine.Edges.LineAmountDiscounts, a.mapInvoiceLineAmountDiscountFromDB)
		if err != nil {
			return invoiceLine, fmt.Errorf("mapping invoice line amount discounts[%s] failed: %w", dbLine.ID, err)
		}
		invoiceLine.Discounts.Amount = discounts
	}

	return invoiceLine, nil
}

func (a *adapter) mapInvoiceLineUsageDiscountFromDB(dbDiscount *db.BillingInvoiceLineUsageDiscount) (billing.UsageLineDiscountManaged, error) {
	base := billing.LineDiscountBase{
		Description:            dbDiscount.Description,
		ChildUniqueReferenceID: dbDiscount.ChildUniqueReferenceID,
		ExternalIDs: billing.LineExternalIDs{
			Invoicing: lo.FromPtr(dbDiscount.InvoicingAppExternalID),
		},
	}

	if dbDiscount.Reason == billing.MaximumSpendDiscountReason && dbDiscount.ReasonDetails == nil {
		// Old (maximum spend) discounts do not have reason details
		base.Reason = billing.NewDiscountReasonFrom(billing.MaximumSpendDiscount{})
	} else {
		if dbDiscount.ReasonDetails == nil {
			return billing.UsageLineDiscountManaged{}, fmt.Errorf("mapping invoice line discount[%s] failed: reason details is nil", dbDiscount.ID)
		}
		base.Reason = *dbDiscount.ReasonDetails
	}

	managed := models.ManagedModelWithID{
		ID: dbDiscount.ID,
		ManagedModel: models.ManagedModel{
			CreatedAt: dbDiscount.CreatedAt.In(time.UTC),
			UpdatedAt: dbDiscount.UpdatedAt.In(time.UTC),
			DeletedAt: convert.TimePtrIn(dbDiscount.DeletedAt, time.UTC),
		},
	}

	return billing.UsageLineDiscountManaged{
		ManagedModelWithID: managed,
		UsageLineDiscount: billing.UsageLineDiscount{
			LineDiscountBase:      base,
			Quantity:              dbDiscount.Quantity,
			PreLinePeriodQuantity: dbDiscount.PreLinePeriodQuantity,
		},
	}, nil
}

func (a *adapter) mapInvoiceLineAmountDiscountFromDB(dbDiscount *db.BillingInvoiceLineDiscount) (billing.AmountLineDiscountManaged, error) {
	base := billing.LineDiscountBase{
		Description:            dbDiscount.Description,
		ChildUniqueReferenceID: dbDiscount.ChildUniqueReferenceID,
		ExternalIDs: billing.LineExternalIDs{
			Invoicing: lo.FromPtr(dbDiscount.InvoicingAppExternalID),
		},
	}

	if dbDiscount.Reason == billing.MaximumSpendDiscountReason && dbDiscount.SourceDiscount == nil {
		// Old (maximum spend) discounts do not have reason details
		base.Reason = billing.NewDiscountReasonFrom(billing.MaximumSpendDiscount{})
	} else {
		if dbDiscount.SourceDiscount == nil {
			return billing.AmountLineDiscountManaged{}, fmt.Errorf("mapping invoice line discount[%s] failed: reason details is nil", dbDiscount.ID)
		}
		base.Reason = *dbDiscount.SourceDiscount
	}

	managed := models.ManagedModelWithID{
		ID: dbDiscount.ID,
		ManagedModel: models.ManagedModel{
			CreatedAt: dbDiscount.CreatedAt.In(time.UTC),
			UpdatedAt: dbDiscount.UpdatedAt.In(time.UTC),
			DeletedAt: convert.TimePtrIn(dbDiscount.DeletedAt, time.UTC),
		},
	}

	return billing.AmountLineDiscountManaged{
		ManagedModelWithID: managed,
		AmountLineDiscount: billing.AmountLineDiscount{
			LineDiscountBase: base,
			Amount:           dbDiscount.Amount,
			RoundingAmount:   lo.FromPtr(dbDiscount.RoundingAmount),
		},
	}, nil
}
