package billingadapter

import (
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/externalid"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func (a *adapter) mapStandardInvoiceLinesFromDB(dbLines []*db.BillingInvoiceLine) (billing.StandardLines, error) {
	lines := make([]*billing.StandardLine, 0, len(dbLines))

	for _, dbLine := range dbLines {
		line, err := a.mapStandardInvoiceLineWithoutReferences(dbLine)
		if err != nil {
			return nil, fmt.Errorf("mapping line [id=%s]: %w", dbLine.ID, err)
		}

		line.DetailedLines, err = slicesx.MapWithErr(dbLine.Edges.DetailedLinesV2, a.mapStandardInvoiceDetailedLineFromDB)
		if err != nil {
			return nil, fmt.Errorf("mapping detailed lines [parent_id=%s]: %w", dbLine.ID, err)
		}

		if err := line.SaveDBSnapshot(); err != nil {
			return nil, fmt.Errorf("saving DB snapshot [id=%s]: %w", line.GetID(), err)
		}

		lines = append(lines, line)
	}

	return lines, nil
}

func (a *adapter) mapStandardInvoiceLineWithoutReferences(dbLine *db.BillingInvoiceLine) (*billing.StandardLine, error) {
	creditsApplied := lo.FromPtr(dbLine.CreditsApplied)
	if len(creditsApplied) == 0 {
		creditsApplied = nil
	}

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
			Engine:      dbLine.Engine,

			Period: timeutil.ClosedPeriod{
				From: dbLine.PeriodStart.In(time.UTC),
				To:   dbLine.PeriodEnd.In(time.UTC),
			},

			ParentLineID:           dbLine.ParentLineID,
			SplitLineGroupID:       dbLine.SplitLineGroupID,
			ChargeID:               dbLine.ChargeID,
			ChildUniqueReferenceID: dbLine.ChildUniqueReferenceID,

			InvoiceAt:                   dbLine.InvoiceAt.In(time.UTC),
			OverrideCollectionPeriodEnd: convert.TimePtrIn(dbLine.OverrideCollectionPeriodEnd, time.UTC),

			Currency: dbLine.Currency,

			TaxConfig:           productcatalog.FromDBTaxColumns(dbLine.TaxCodeID, dbLine.TaxBehavior),
			InvoicingAppTaxCode: dbLine.InvoicingAppTaxCode,
			RateCardDiscounts:   lo.FromPtr(dbLine.RatecardDiscounts),
			CreditsApplied:      creditsApplied,
			Totals:              totals.FromDB(dbLine),
			ExternalIDs:         externalid.MapLineExternalIDFromDB(dbLine),
		},
	}

	// Only the stripe code of the legacy JSON is needed, as the billed code of lines that were
	// never stamped.
	if dbLine.TaxConfig.Stripe != nil {
		invoiceLine.LegacyStripeTaxCode = dbLine.TaxConfig.Stripe.Code
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
		UnitConfig:                   ubpLine.UnitConfig,
	}

	if len(dbLine.Edges.LineUsageDiscounts) > 0 {
		discounts, err := slicesx.MapWithErr(dbLine.Edges.LineUsageDiscounts, a.mapStandardInvoiceLineUsageDiscountFromDB)
		if err != nil {
			return nil, fmt.Errorf("mapping invoice line usage discounts[%s] failed: %w", dbLine.ID, err)
		}

		invoiceLine.Discounts.Usage = discounts
	}

	return invoiceLine, nil
}

func (a *adapter) mapStandardInvoiceDetailedLineFromDB(dbLine *db.BillingStandardInvoiceDetailedLine) (billing.DetailedLine, error) {
	detailedLineBase := billing.DetailedLineBase{
		InvoiceID: dbLine.InvoiceID,
		Base:      stddetailedline.FromDB(dbLine),
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
		ExternalIDs:            externalid.MapLineExternalIDFromDB(dbDiscount),
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

func (a *adapter) mapStandardInvoiceDetailedLineAmountDiscountFromDB(dbDiscount *db.BillingStandardInvoiceDetailedLineAmountDiscount) (billing.AmountLineDiscountManaged, error) {
	base := billing.LineDiscountBase{
		Description:            dbDiscount.Description,
		ChildUniqueReferenceID: dbDiscount.ChildUniqueReferenceID,
		ExternalIDs:            externalid.MapLineExternalIDFromDB(dbDiscount),
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
