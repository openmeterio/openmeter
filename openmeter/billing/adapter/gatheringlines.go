package billingadapter

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceline"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceusagebasedlineconfig"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/entitydiff"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type gatheringLineDiff struct {
	Line entitydiff.Diff[*billing.UpcomingCharge]
}

func diffGatheringInvoiceLines(lines billing.UpcomingCharges) (gatheringLineDiff, error) {
	dbState := []*billing.UpcomingCharge{}
	for _, line := range lines {
		if line.DBState != nil {
			dbState = append(dbState, line.DBState)
		}
	}

	linePtrs := lo.Map(lines, func(_ billing.UpcomingCharge, idx int) *billing.UpcomingCharge {
		return &lines[idx]
	})

	diff := gatheringLineDiff{}

	err := entitydiff.DiffByID(entitydiff.DiffByIDInput[*billing.UpcomingCharge]{
		DBState:       dbState,
		ExpectedState: linePtrs,
		HandleDelete: func(item *billing.UpcomingCharge) error {
			diff.Line.NeedsDelete(item)
			return nil
		},
		HandleCreate: func(item *billing.UpcomingCharge) error {
			diff.Line.NeedsCreate(item)
			return nil
		},
		HandleUpdate: func(item entitydiff.DiffUpdate[*billing.UpcomingCharge]) error {
			diff.Line.NeedsUpdate(item)
			return nil
		},
	})
	if err != nil {
		return gatheringLineDiff{}, err
	}

	return diff, nil
}

func (a *adapter) updateUpcomingCharges(ctx context.Context, lines billing.UpcomingCharges) error {
	diff, err := diffGatheringInvoiceLines(lines)
	if err != nil {
		return err
	}

	err = upsertWithOptions(ctx, a.db, diff.Line, upsertInput[*billing.UpcomingCharge, *db.BillingInvoiceUsageBasedLineConfigCreate]{
		Create: func(tx *db.Client, line *billing.UpcomingCharge) (*db.BillingInvoiceUsageBasedLineConfigCreate, error) {
			if line.UBPConfigID == "" {
				line.UBPConfigID = ulid.Make().String()
			}

			create := tx.BillingInvoiceUsageBasedLineConfig.Create().
				SetNamespace(line.Namespace).
				SetPriceType(line.Price.Type()).
				SetPrice(lo.ToPtr(line.Price)).
				SetFeatureKey(line.FeatureKey).
				SetID(line.UBPConfigID)

			return create, nil
		},
		UpsertItems: func(ctx context.Context, tx *db.Client, items []*db.BillingInvoiceUsageBasedLineConfigCreate) error {
			return tx.BillingInvoiceUsageBasedLineConfig.
				CreateBulk(items...).
				OnConflict(
					sql.ConflictColumns(billinginvoiceusagebasedlineconfig.FieldID),
					sql.ResolveWithNewValues(),
				).Exec(ctx)
		},
	})
	if err != nil {
		return fmt.Errorf("creating usage based line configs: %w", err)
	}

	invoiceLineUpsertConfig := upsertInput[*billing.UpcomingCharge, *db.BillingInvoiceLineCreate]{
		Create: func(tx *db.Client, line *billing.UpcomingCharge) (*db.BillingInvoiceLineCreate, error) {
			if line.ID == "" {
				line.ID = ulid.Make().String()
			}

			create := tx.BillingInvoiceLine.Create().
				SetID(line.ID).
				SetNamespace(line.Namespace).
				SetInvoiceID(line.InvoiceID).
				SetPeriodStart(line.ServicePeriod.From.In(time.UTC)).
				SetPeriodEnd(line.ServicePeriod.To.In(time.UTC)).
				SetNillableSplitLineGroupID(line.SplitLineGroupID).
				SetNillableDeletedAt(line.DeletedAt).
				SetInvoiceAt(line.InvoiceAt.In(time.UTC)).
				SetStatus(billing.InvoiceLineStatusValid).
				SetManagedBy(line.ManagedBy).
				SetType(billing.InvoiceLineTypeUsageBased).
				SetName(line.Name).
				SetNillableDescription(line.Description).
				SetCurrency(line.Currency).
				SetMetadata(line.Metadata).
				SetAnnotations(line.Annotations).
				SetNillableChildUniqueReferenceID(line.ChildUniqueReferenceID).
				// Totals
				SetAmount(alpacadecimal.Zero).
				SetChargesTotal(alpacadecimal.Zero).
				SetDiscountsTotal(alpacadecimal.Zero).
				SetTaxesTotal(alpacadecimal.Zero).
				SetTaxesInclusiveTotal(alpacadecimal.Zero).
				SetTaxesExclusiveTotal(alpacadecimal.Zero).
				SetTotal(alpacadecimal.Zero)

			if line.Subscription != nil {
				create = create.SetSubscriptionID(line.Subscription.SubscriptionID).
					SetSubscriptionPhaseID(line.Subscription.PhaseID).
					SetSubscriptionItemID(line.Subscription.ItemID).
					SetSubscriptionBillingPeriodFrom(line.Subscription.BillingPeriod.From.In(time.UTC)).
					SetSubscriptionBillingPeriodTo(line.Subscription.BillingPeriod.To.In(time.UTC))
			}

			if line.TaxConfig != nil {
				create = create.SetTaxConfig(*line.TaxConfig)
			}

			if !line.RateCardDiscounts.IsEmpty() {
				create = create.SetRatecardDiscounts(lo.ToPtr(line.RateCardDiscounts))
			}

			create = create.
				SetUsageBasedLineID(line.UBPConfigID)

			return create, nil
		},
		UpsertItems: func(ctx context.Context, tx *db.Client, items []*db.BillingInvoiceLineCreate) error {
			return tx.BillingInvoiceLine.
				CreateBulk(items...).
				OnConflict(sql.ConflictColumns(billinginvoiceline.FieldID),
					sql.ResolveWithNewValues(),
					sql.ResolveWith(func(u *sql.UpdateSet) {
						u.SetIgnore(billinginvoiceline.FieldCreatedAt)
					})).
				UpdateChildUniqueReferenceID().
				Exec(ctx)
		},
		MarkDeleted: func(ctx context.Context, line *billing.UpcomingCharge) (*billing.UpcomingCharge, error) {
			line.DeletedAt = lo.ToPtr(clock.Now().In(time.UTC))
			return line, nil
		},
	}

	if err := upsertWithOptions(ctx, a.db, diff.Line, invoiceLineUpsertConfig); err != nil {
		return fmt.Errorf("creating lines: %w", err)
	}

	return nil
}

func (a *adapter) mapGatheringInvoiceLinesFromDB(schemaLevel int, dbLines []*db.BillingInvoiceLine) ([]billing.UpcomingCharge, error) {
	return slicesx.MapWithErr(dbLines, func(dbLine *db.BillingInvoiceLine) (billing.UpcomingCharge, error) {
		return a.mapGatheringInvoiceLineFromDB(schemaLevel, dbLine)
	})
}

func (a *adapter) mapGatheringInvoiceLineFromDB(schemaLevel int, dbLine *db.BillingInvoiceLine) (billing.UpcomingCharge, error) {
	if dbLine.Type != billing.InvoiceLineTypeUsageBased {
		return billing.UpcomingCharge{}, fmt.Errorf("only usage based lines can be gathering invoice lines [line_id=%s]", dbLine.ID)
	}

	ubpLine := dbLine.Edges.UsageBasedLine
	if ubpLine == nil {
		return billing.UpcomingCharge{}, fmt.Errorf("usage based line data is missing [line_id=%s]", dbLine.ID)
	}

	line := billing.UpcomingCharge{
		UpcomingChargeBase: billing.UpcomingChargeBase{
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

			ServicePeriod: timeutil.ClosedPeriod{
				From: dbLine.PeriodStart.In(time.UTC),
				To:   dbLine.PeriodEnd.In(time.UTC),
			},

			SplitLineGroupID:       dbLine.SplitLineGroupID,
			ChildUniqueReferenceID: dbLine.ChildUniqueReferenceID,

			InvoiceAt: dbLine.InvoiceAt.In(time.UTC),

			Currency: dbLine.Currency,

			TaxConfig:         lo.EmptyableToPtr(dbLine.TaxConfig),
			RateCardDiscounts: lo.FromPtr(dbLine.RatecardDiscounts),

			UBPConfigID: ubpLine.ID,
			FeatureKey:  lo.FromPtr(ubpLine.FeatureKey),
			Price:       lo.FromPtr(ubpLine.Price),
		},
	}

	if dbLine.SubscriptionID != nil && dbLine.SubscriptionPhaseID != nil && dbLine.SubscriptionItemID != nil {
		line.Subscription = &billing.SubscriptionReference{
			SubscriptionID: *dbLine.SubscriptionID,
			PhaseID:        *dbLine.SubscriptionPhaseID,
			ItemID:         *dbLine.SubscriptionItemID,
			BillingPeriod: timeutil.ClosedPeriod{
				From: lo.FromPtr(dbLine.SubscriptionBillingPeriodFrom).In(time.UTC),
				To:   lo.FromPtr(dbLine.SubscriptionBillingPeriodTo).In(time.UTC),
			},
		}
	}

	cloned, err := line.WithoutDBState()
	if err != nil {
		return billing.UpcomingCharge{}, fmt.Errorf("cloning line: %w", err)
	}

	line.DBState = lo.ToPtr(cloned)

	return line, nil
}
