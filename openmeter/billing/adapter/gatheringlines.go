package billingadapter

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceline"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceusagebasedlineconfig"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/entitydiff"
	"github.com/samber/lo"
)

type gatheringLineDiff struct {
	Line entitydiff.Diff[*billing.GatheringLine]
}

func diffGatheringInvoiceLines(lines billing.GatheringLines) (gatheringLineDiff, error) {
	dbState := []*billing.GatheringLine{}
	for _, line := range lines {
		if line.DBState != nil {
			dbState = append(dbState, line.DBState)
		}
	}

	linePtrs := lo.Map(lines, func(_ billing.GatheringLine, idx int) *billing.GatheringLine {
		return &lines[idx]
	})

	diff := gatheringLineDiff{}

	err := entitydiff.DiffByID(entitydiff.DiffByIDInput[*billing.GatheringLine]{
		DBState:       dbState,
		ExpectedState: linePtrs,
		HandleDelete: func(item *billing.GatheringLine) error {
			diff.Line.NeedsDelete(item)
			return nil
		},
		HandleCreate: func(item *billing.GatheringLine) error {
			diff.Line.NeedsCreate(item)
			return nil
		},
		HandleUpdate: func(item entitydiff.DiffUpdate[*billing.GatheringLine]) error {
			diff.Line.NeedsUpdate(item)
			return nil
		},
	})
	if err != nil {
		return gatheringLineDiff{}, err
	}

	return diff, nil
}

func (a *adapter) updateGatheringLines(ctx context.Context, lines billing.GatheringLines) error {
	diff, err := diffGatheringInvoiceLines(lines)
	if err != nil {
		return err
	}

	upsertWithOptions(ctx, a.db, diff.Line, upsertInput[*billing.GatheringLine, *db.BillingInvoiceUsageBasedLineConfigCreate]{
		Create: func(tx *db.Client, line *billing.GatheringLine) (*db.BillingInvoiceUsageBasedLineConfigCreate, error) {
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

	invoiceLineUpsertConfig := upsertInput[*billing.GatheringLine, *db.BillingInvoiceLineCreate]{
		Create: func(tx *db.Client, line *billing.GatheringLine) (*db.BillingInvoiceLineCreate, error) {
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
		MarkDeleted: func(ctx context.Context, line *billing.GatheringLine) (*billing.GatheringLine, error) {
			line.DeletedAt = lo.ToPtr(clock.Now().In(time.UTC))
			return line, nil
		},
	}

	if err := upsertWithOptions(ctx, a.db, diff.Line, invoiceLineUpsertConfig); err != nil {
		return fmt.Errorf("creating lines: %w", err)
	}

	return nil
}
