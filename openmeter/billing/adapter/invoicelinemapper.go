package billingadapter

import (
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func (a *adapter) mapInvoiceLineFromDB(dbLines []*db.BillingInvoiceLine) ([]*billing.Line, error) {
	lines := make([]*billing.Line, 0, len(dbLines))

	for _, dbLine := range dbLines {
		line, err := a.mapInvoiceLineWithoutReferences(dbLine)
		if err != nil {
			return nil, fmt.Errorf("mapping line [id=%s]: %w", dbLine.ID, err)
		}

		// Let's map any detailed lines
		detailedLines := make([]*billing.Line, 0, len(dbLine.Edges.DetailedLines))
		for _, dbDetailedLine := range dbLine.Edges.DetailedLines {
			detailedLine, err := a.mapInvoiceDetailedLineWithoutReferences(dbDetailedLine)
			if err != nil {
				return nil, fmt.Errorf("mapping detailed line [parentID=%s,id=%s]: %w", dbLine.ID, dbDetailedLine.ID, err)
			}

			detailedLine.SaveDBSnapshot()

			detailedLines = append(detailedLines, detailedLine)
		}

		line.Children = billing.NewLineChildren(detailedLines)

		line.SaveDBSnapshot()

		lines = append(lines, line)
	}

	return lines, nil
}

func (a *adapter) mapInvoiceLineWithoutReferences(dbLine *db.BillingInvoiceLine) (*billing.Line, error) {
	invoiceLine := &billing.Line{
		LineBase: billing.LineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace:   dbLine.Namespace,
				ID:          dbLine.ID,
				CreatedAt:   dbLine.CreatedAt.In(time.UTC),
				UpdatedAt:   dbLine.UpdatedAt.In(time.UTC),
				DeletedAt:   convert.TimePtrIn(dbLine.DeletedAt, time.UTC),
				Name:        dbLine.Name,
				Description: dbLine.Description,
			}),

			Metadata:    dbLine.Metadata,
			Annotations: dbLine.Annotations,
			InvoiceID:   dbLine.InvoiceID,
			Status:      dbLine.Status,
			ManagedBy:   dbLine.ManagedBy,

			Period: billing.Period{
				Start: dbLine.PeriodStart.In(time.UTC),
				End:   dbLine.PeriodEnd.In(time.UTC),
			},

			ParentLineID:           dbLine.ParentLineID,
			SplitLineGroupID:       dbLine.SplitLineGroupID,
			ChildUniqueReferenceID: dbLine.ChildUniqueReferenceID,

			InvoiceAt: dbLine.InvoiceAt.In(time.UTC),

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
			BillingPeriod: timeutil.ClosedPeriod{
				From: lo.FromPtr(dbLine.SubscriptionBillingPeriodFrom).In(time.UTC),
				To:   lo.FromPtr(dbLine.SubscriptionBillingPeriodTo).In(time.UTC),
			},
		}
	}

	if dbLine.Type != billing.InvoiceLineTypeUsageBased {
		return invoiceLine, fmt.Errorf("only usage based lines can be top level lines [line_id=%s]", dbLine.ID)
	}

	ubpLine := dbLine.Edges.UsageBasedLine
	if ubpLine == nil {
		return nil, fmt.Errorf("manual usage based line is missing")
	}

	invoiceLine.UsageBased = &billing.UsageBasedLine{
		ConfigID:                     ubpLine.ID,
		FeatureKey:                   lo.FromPtr(ubpLine.FeatureKey),
		Price:                        ubpLine.Price,
		Quantity:                     dbLine.Quantity,
		MeteredQuantity:              ubpLine.MeteredQuantity,
		PreLinePeriodQuantity:        ubpLine.PreLinePeriodQuantity,
		MeteredPreLinePeriodQuantity: ubpLine.MeteredPreLinePeriodQuantity,
	}

	if len(dbLine.Edges.LineUsageDiscounts) > 0 {
		discounts, err := slicesx.MapWithErr(dbLine.Edges.LineUsageDiscounts, a.mapInvoiceLineUsageDiscountFromDB)
		if err != nil {
			return nil, fmt.Errorf("mapping invoice line usage discounts[%s] failed: %w", dbLine.ID, err)
		}

		invoiceLine.Discounts.Usage = discounts
	}

	if len(dbLine.Edges.LineAmountDiscounts) > 0 {
		discounts, err := slicesx.MapWithErr(dbLine.Edges.LineAmountDiscounts, a.mapInvoiceLineAmountDiscountFromDB)
		if err != nil {
			return nil, fmt.Errorf("mapping invoice line amount discounts[%s] failed: %w", dbLine.ID, err)
		}
		invoiceLine.Discounts.Amount = discounts
	}

	return invoiceLine, nil
}

func (a *adapter) mapInvoiceDetailedLineWithoutReferences(dbLine *db.BillingInvoiceLine) (*billing.DetailedLine, error) {
	invoiceLine := &billing.DetailedLine{
		LineBase: billing.LineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace:   dbLine.Namespace,
				ID:          dbLine.ID,
				CreatedAt:   dbLine.CreatedAt.In(time.UTC),
				UpdatedAt:   dbLine.UpdatedAt.In(time.UTC),
				DeletedAt:   convert.TimePtrIn(dbLine.DeletedAt, time.UTC),
				Name:        dbLine.Name,
				Description: dbLine.Description,
			}),

			Metadata:    dbLine.Metadata,
			Annotations: dbLine.Annotations,
			InvoiceID:   dbLine.InvoiceID,
			Status:      dbLine.Status,
			ManagedBy:   dbLine.ManagedBy,

			Period: billing.Period{
				Start: dbLine.PeriodStart.In(time.UTC),
				End:   dbLine.PeriodEnd.In(time.UTC),
			},

			ParentLineID:           dbLine.ParentLineID,
			SplitLineGroupID:       dbLine.SplitLineGroupID,
			ChildUniqueReferenceID: dbLine.ChildUniqueReferenceID,

			InvoiceAt: dbLine.InvoiceAt.In(time.UTC),

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
			BillingPeriod: timeutil.ClosedPeriod{
				From: lo.FromPtr(dbLine.SubscriptionBillingPeriodFrom).In(time.UTC),
				To:   lo.FromPtr(dbLine.SubscriptionBillingPeriodTo).In(time.UTC),
			},
		}
	}

	if dbLine.Type != billing.InvoiceLineTypeFee {
		return nil, fmt.Errorf("only fee typed lines can be detailed lines [line_id=%s]", dbLine.ID)
	}

	invoiceLine.FlatFee = &billing.FlatFeeLine{
		ConfigID:      dbLine.Edges.FlatFeeLine.ID,
		PerUnitAmount: dbLine.Edges.FlatFeeLine.PerUnitAmount,
		Quantity:      lo.FromPtr(dbLine.Quantity),
		Category:      dbLine.Edges.FlatFeeLine.Category,
		PaymentTerm:   dbLine.Edges.FlatFeeLine.PaymentTerm,
		Index:         dbLine.Edges.FlatFeeLine.Index,
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
