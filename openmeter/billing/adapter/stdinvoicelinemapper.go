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

func (a *adapter) mapStandardInvoiceLinesFromDB(schemaLevelByInvoiceID map[string]int, dbLines []*db.BillingInvoiceLine) ([]*billing.StandardLine, error) {
	lines := make([]*billing.StandardLine, 0, len(dbLines))

	for _, dbLine := range dbLines {
		line, err := a.mapStandardInvoiceLineWithoutReferences(dbLine)
		if err != nil {
			return nil, fmt.Errorf("mapping line [id=%s]: %w", dbLine.ID, err)
		}

		schemaLevel, found := schemaLevelByInvoiceID[dbLine.InvoiceID]
		if !found {
			return nil, fmt.Errorf("schema level not found for invoice [id=%s]", dbLine.InvoiceID)
		}

		if schemaLevel == 1 {
			// Let's map any detailed lines
			line.DetailedLines, err = slicesx.MapWithErr(dbLine.Edges.DetailedLines, a.mapStandardInvoiceDetailedLineFromDB)
			if err != nil {
				return nil, fmt.Errorf("mapping detailed lines [parentID=%s,id=%s]: %w", lo.FromPtr(dbLine.ParentLineID), dbLine.ID, err)
			}
		} else {
			line.DetailedLines, err = slicesx.MapWithErr(dbLine.Edges.DetailedLinesV2, a.mapStandardInvoiceDetailedLineV2FromDB)
			if err != nil {
				return nil, fmt.Errorf("mapping detailed lines [parentID=%s,id=%s]: %w", lo.FromPtr(dbLine.ParentLineID), dbLine.ID, err)
			}
		}

		if err := line.SaveDBSnapshot(); err != nil {
			return nil, fmt.Errorf("saving DB snapshot [id=%s]: %w", line.GetID(), err)
		}

		lines = append(lines, line)
	}

	return lines, nil
}

func (a *adapter) mapStandardInvoiceLineWithoutReferences(dbLine *db.BillingInvoiceLine) (*billing.StandardLine, error) {
	invoiceLine := &billing.StandardLine{
		StandardLineBase: billing.StandardLineBase{
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
			ManagedBy:   dbLine.ManagedBy,

			Period: billing.Period{
				Start: dbLine.PeriodStart.In(time.UTC),
				End:   dbLine.PeriodEnd.In(time.UTC),
			},

			ParentLineID:           dbLine.ParentLineID,
			SplitLineGroupID:       dbLine.SplitLineGroupID,
			ChildUniqueReferenceID: dbLine.ChildUniqueReferenceID,

			InvoiceAt: dbLine.InvoiceAt.In(time.UTC),

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

	if dbLine.Type != billing.InvoiceLineAdapterTypeUsageBased {
		return nil, fmt.Errorf("only usage based lines can be top level lines [line_id=%s]", dbLine.ID)
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
		discounts, err := slicesx.MapWithErr(dbLine.Edges.LineUsageDiscounts, a.mapStandardInvoiceLineUsageDiscountFromDB)
		if err != nil {
			return nil, fmt.Errorf("mapping invoice line usage discounts[%s] failed: %w", dbLine.ID, err)
		}

		invoiceLine.Discounts.Usage = discounts
	}

	if len(dbLine.Edges.LineAmountDiscounts) > 0 {
		discounts, err := slicesx.MapWithErr(dbLine.Edges.LineAmountDiscounts, a.mapStandardInvoiceLineAmountDiscountFromDB)
		if err != nil {
			return nil, fmt.Errorf("mapping invoice line amount discounts[%s] failed: %w", dbLine.ID, err)
		}
		invoiceLine.Discounts.Amount = discounts
	}

	return invoiceLine, nil
}

func (a *adapter) mapStandardInvoiceDetailedLineFromDB(dbLine *db.BillingInvoiceLine) (billing.DetailedLine, error) {
	// TODO: Once we move into a separate table we can get rid of these assertions
	if dbLine.ParentLineID == nil {
		return billing.DetailedLine{}, fmt.Errorf("detailed line parent line ID is required [detailed_line_id=%s]", dbLine.ID)
	}

	detailedLineBase := billing.DetailedLineBase{
		ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
			Namespace:   dbLine.Namespace,
			ID:          dbLine.ID,
			CreatedAt:   dbLine.CreatedAt.In(time.UTC),
			UpdatedAt:   dbLine.UpdatedAt.In(time.UTC),
			DeletedAt:   convert.TimePtrIn(dbLine.DeletedAt, time.UTC),
			Name:        dbLine.Name,
			Description: dbLine.Description,
		}),

		InvoiceID:              dbLine.InvoiceID,
		ChildUniqueReferenceID: dbLine.ChildUniqueReferenceID,
		FeeLineConfigID:        dbLine.Edges.FlatFeeLine.ID,

		ServicePeriod: billing.Period{
			Start: dbLine.PeriodStart.In(time.UTC),
			End:   dbLine.PeriodEnd.In(time.UTC),
		},
		PerUnitAmount: dbLine.Edges.FlatFeeLine.PerUnitAmount,
		Quantity:      lo.FromPtr(dbLine.Quantity),
		Category:      dbLine.Edges.FlatFeeLine.Category,
		PaymentTerm:   dbLine.Edges.FlatFeeLine.PaymentTerm,
		Index:         dbLine.Edges.FlatFeeLine.Index,

		Currency: dbLine.Currency,

		TaxConfig: lo.EmptyableToPtr(dbLine.TaxConfig),
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
	}

	discounts, err := slicesx.MapWithErr(dbLine.Edges.LineAmountDiscounts, a.mapStandardInvoiceLineAmountDiscountFromDB)
	if err != nil {
		return billing.DetailedLine{}, fmt.Errorf("mapping invoice line amount discounts[%s] failed: %w", dbLine.ID, err)
	}

	return billing.DetailedLine{
		DetailedLineBase: detailedLineBase,
		AmountDiscounts:  discounts,
	}, nil
}

func (a *adapter) mapStandardInvoiceDetailedLineV2FromDB(dbLine *db.BillingStandardInvoiceDetailedLine) (billing.DetailedLine, error) {
	detailedLineBase := billing.DetailedLineBase{
		ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
			Namespace:   dbLine.Namespace,
			ID:          dbLine.ID,
			CreatedAt:   dbLine.CreatedAt.In(time.UTC),
			UpdatedAt:   dbLine.UpdatedAt.In(time.UTC),
			DeletedAt:   convert.TimePtrIn(dbLine.DeletedAt, time.UTC),
			Name:        dbLine.Name,
			Description: dbLine.Description,
		}),

		InvoiceID:              dbLine.InvoiceID,
		ChildUniqueReferenceID: dbLine.ChildUniqueReferenceID,

		ServicePeriod: billing.Period{
			Start: dbLine.ServicePeriodStart.In(time.UTC),
			End:   dbLine.ServicePeriodEnd.In(time.UTC),
		},
		PerUnitAmount: dbLine.PerUnitAmount,
		Quantity:      dbLine.Quantity,
		Category:      dbLine.Category,
		PaymentTerm:   dbLine.PaymentTerm,
		Index:         dbLine.Index,

		Currency: dbLine.Currency,

		TaxConfig: lo.EmptyableToPtr(dbLine.TaxConfig),
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
	}

	discounts, err := slicesx.MapWithErr(dbLine.Edges.AmountDiscounts, a.mapStandardInvoiceDetailedLineAmountDiscountFromDB)
	if err != nil {
		return billing.DetailedLine{}, fmt.Errorf("mapping invoice line amount discounts[%s] failed: %w", dbLine.ID, err)
	}

	return billing.DetailedLine{
		DetailedLineBase: detailedLineBase,
		AmountDiscounts:  discounts,
	}, nil
}

func (a *adapter) mapStandardInvoiceLineUsageDiscountFromDB(dbDiscount *db.BillingInvoiceLineUsageDiscount) (billing.UsageLineDiscountManaged, error) {
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

func (a *adapter) mapStandardInvoiceLineAmountDiscountFromDB(dbDiscount *db.BillingInvoiceLineDiscount) (billing.AmountLineDiscountManaged, error) {
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

func (a *adapter) mapStandardInvoiceDetailedLineAmountDiscountFromDB(dbDiscount *db.BillingStandardInvoiceDetailedLineAmountDiscount) (billing.AmountLineDiscountManaged, error) {
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
