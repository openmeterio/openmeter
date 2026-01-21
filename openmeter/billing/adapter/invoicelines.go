package billingadapter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoice"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceflatfeelineconfig"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceline"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoicelinediscount"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoicelineusagediscount"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoicesplitlinegroup"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceusagebasedlineconfig"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingstandardinvoicedetailedline"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingstandardinvoicedetailedlineamountdiscount"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/entitydiff"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

var _ billing.InvoiceLineAdapter = (*adapter)(nil)

func (a *adapter) UpsertInvoiceLines(ctx context.Context, inputIn billing.UpsertInvoiceLinesAdapterInput) ([]*billing.Line, error) {
	// Given that the input's content is spread across multiple tables, we need to
	// handle the upserting of the data in a more complex way. We will first upsert
	// all items that yield an ID into their parent structs then we will create the
	// parents.

	if err := inputIn.Validate(); err != nil {
		return nil, err
	}

	// Validate for missing functionality (this is put here, as we should remove them from here,
	// once we have the functionality)

	input := &billing.UpsertInvoiceLinesAdapterInput{
		Namespace: inputIn.Namespace,
		Lines: lo.Map(inputIn.Lines, func(line *billing.Line, _ int) *billing.Line {
			return line.Clone()
		}),
		SchemaLevel: inputIn.SchemaLevel,
		InvoiceID:   inputIn.InvoiceID,
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]*billing.Line, error) {
		// Let's genereate the line diffs first
		lineDiffs, err := diffInvoiceLines(input.Lines)
		if err != nil {
			return nil, fmt.Errorf("generating line diffs: %w", err)
		}

		if input.SchemaLevel == 1 {
			// Step 1: Let's create/upsert the line configs first
			if err = tx.upsertFeeLineConfig(ctx, lineDiffs.DetailedLine); err != nil {
				return nil, fmt.Errorf("upserting fee line configs: %w", err)
			}
		}

		if err := tx.upsertUsageBasedConfig(ctx, lineDiffs.Line); err != nil {
			return nil, fmt.Errorf("upserting usage based line configs: %w", err)
		}

		// Step 2: Let's create the lines, but not their detailed lines
		invoiceLineUpsertConfig := upsertInput[*billing.Line, *db.BillingInvoiceLineCreate]{
			Create: func(tx *db.Client, line *billing.Line) (*db.BillingInvoiceLineCreate, error) {
				if line.ID == "" {
					line.ID = ulid.Make().String()
				}

				create := tx.BillingInvoiceLine.Create().
					SetID(line.ID).
					SetNamespace(line.Namespace).
					SetInvoiceID(line.InvoiceID).
					SetPeriodStart(line.Period.Start.In(time.UTC)).
					SetPeriodEnd(line.Period.End.In(time.UTC)).
					SetNillableParentLineID(line.ParentLineID).
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
					SetAmount(line.Totals.Amount).
					SetChargesTotal(line.Totals.ChargesTotal).
					SetDiscountsTotal(line.Totals.DiscountsTotal).
					SetTaxesTotal(line.Totals.TaxesTotal).
					SetTaxesInclusiveTotal(line.Totals.TaxesInclusiveTotal).
					SetTaxesExclusiveTotal(line.Totals.TaxesExclusiveTotal).
					SetTotal(line.Totals.Total).
					// ExternalIDs
					SetNillableInvoicingAppExternalID(lo.EmptyableToPtr(line.ExternalIDs.Invoicing))

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
					SetNillableQuantity(line.UsageBased.Quantity).
					SetUsageBasedLineID(line.UsageBased.ConfigID).
					SetNillableFlatFeeLineID(nil)

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
					// TODO[OM-1416]: all nillable fileds must be listed explicitly
					UpdateQuantity().
					UpdateChildUniqueReferenceID().
					Exec(ctx)
			},
			MarkDeleted: func(ctx context.Context, line *billing.Line) (*billing.Line, error) {
				line.DeletedAt = lo.ToPtr(clock.Now().In(time.UTC))
				return line, nil
			},
		}

		if err := upsertWithOptions(ctx, tx.db, lineDiffs.Line, invoiceLineUpsertConfig); err != nil {
			return nil, fmt.Errorf("creating lines: %w", err)
		}

		// Step 3: Let's create the detailed lines
		if input.SchemaLevel == 1 {
			if err := tx.upsertDetailedLines(ctx, lineDiffs.DetailedLine); err != nil {
				return nil, fmt.Errorf("upserting detailed lines: %w", err)
			}
			// detailed line amount discounts
			err = tx.upsertDetailedLineAmountDiscounts(ctx, lineDiffs.DetailedLineAmountDiscounts)
			if err != nil {
				return nil, fmt.Errorf("upserting detailed line amount discounts: %w", err)
			}
		} else {
			if err := tx.upsertDetailedLinesV2(ctx, lineDiffs.DetailedLine); err != nil {
				return nil, fmt.Errorf("upserting detailed lines: %w", err)
			}
			// detailed line amount discounts
			err = tx.upsertDetailedLineAmountDiscountsV2(ctx, lineDiffs.DetailedLineAmountDiscounts)
			if err != nil {
				return nil, fmt.Errorf("upserting detailed line amount discounts: %w", err)
			}
		}

		// Step 4: Let's upsert anything else, that doesn't have strict ID requirements

		// Step 4a: Line Discounts
		err = upsertWithOptions(ctx, tx.db, lineDiffs.UsageDiscounts, upsertInput[usageLineDiscountManagedWithLine, *db.BillingInvoiceLineUsageDiscountCreate]{
			Create: func(tx *db.Client, d usageLineDiscountManagedWithLine) (*db.BillingInvoiceLineUsageDiscountCreate, error) {
				discount := d.Entity

				if discount.ID == "" {
					discount.ID = ulid.Make().String()
				}

				create := tx.BillingInvoiceLineUsageDiscount.Create().
					SetID(discount.ID).
					SetNamespace(d.Parent.GetNamespace()).
					SetLineID(d.Parent.GetID()).
					SetReason(discount.Reason.Type()).
					SetReasonDetails(lo.ToPtr(discount.Reason)).
					SetQuantity(discount.Quantity).
					SetNillablePreLinePeriodQuantity(discount.PreLinePeriodQuantity).
					SetNillableDeletedAt(discount.DeletedAt).
					SetNillableChildUniqueReferenceID(discount.ChildUniqueReferenceID).
					SetNillableDescription(discount.Description).
					// ExternalIDs
					SetNillableInvoicingAppExternalID(lo.EmptyableToPtr(discount.ExternalIDs.Invoicing))

				return create, nil
			},
			UpsertItems: func(ctx context.Context, tx *db.Client, items []*db.BillingInvoiceLineUsageDiscountCreate) error {
				return tx.BillingInvoiceLineUsageDiscount.
					CreateBulk(items...).
					OnConflict(
						sql.ConflictColumns(billinginvoicelineusagediscount.FieldID),
						sql.ResolveWithNewValues(),
						sql.ResolveWith(func(u *sql.UpdateSet) {
							u.SetIgnore(billinginvoicelineusagediscount.FieldCreatedAt)
						}),
					).Exec(ctx)
			},
			MarkDeleted: func(ctx context.Context, d usageLineDiscountManagedWithLine) (usageLineDiscountManagedWithLine, error) {
				d.Entity.DeletedAt = lo.ToPtr(clock.Now().In(time.UTC))

				return d, nil
			},
		})
		if err != nil {
			return nil, fmt.Errorf("upserting usage discounts: %w", err)
		}

		err = upsertWithOptions(ctx, tx.db, lineDiffs.AmountDiscounts, upsertInput[amountLineDiscountManagedWithLine, *db.BillingInvoiceLineDiscountCreate]{
			Create: func(tx *db.Client, d amountLineDiscountManagedWithLine) (*db.BillingInvoiceLineDiscountCreate, error) {
				discount := d.Entity

				if discount.ID == "" {
					discount.ID = ulid.Make().String()
				}

				create := tx.BillingInvoiceLineDiscount.Create().
					SetID(discount.ID).
					SetNamespace(d.Parent.GetNamespace()).
					SetLineID(d.Parent.GetID()).
					SetReason(discount.Reason.Type()).
					SetSourceDiscount(lo.ToPtr(discount.Reason)).
					SetAmount(discount.Amount).
					SetNillableRoundingAmount(lo.EmptyableToPtr(discount.RoundingAmount)).
					SetNillableDeletedAt(discount.DeletedAt).
					SetNillableChildUniqueReferenceID(discount.ChildUniqueReferenceID).
					SetNillableDescription(discount.Description).
					// ExternalIDs
					SetNillableInvoicingAppExternalID(lo.EmptyableToPtr(discount.ExternalIDs.Invoicing))

				return create, nil
			},
			UpsertItems: func(ctx context.Context, tx *db.Client, items []*db.BillingInvoiceLineDiscountCreate) error {
				return tx.BillingInvoiceLineDiscount.
					CreateBulk(items...).
					OnConflict(
						sql.ConflictColumns(billinginvoicelinediscount.FieldID),
						sql.ResolveWithNewValues(),
						sql.ResolveWith(func(u *sql.UpdateSet) {
							u.SetIgnore(billinginvoicelinediscount.FieldCreatedAt)
						}),
					).Exec(ctx)
			},
			MarkDeleted: func(ctx context.Context, d amountLineDiscountManagedWithLine) (amountLineDiscountManagedWithLine, error) {
				d.Entity.DeletedAt = lo.ToPtr(clock.Now().In(time.UTC))

				return d, nil
			},
		})
		if err != nil {
			return nil, fmt.Errorf("upserting amount discounts: %w", err)
		}

		// Step 4b: Taxes (TODO[later]: implement)

		// Step 5: Update updated_at for all the affected lines
		if !lineDiffs.AffectedLineIDs.IsEmpty() {
			err := tx.db.BillingInvoiceLine.Update().
				SetUpdatedAt(clock.Now().In(time.UTC)).
				Where(billinginvoiceline.IDIn(lineDiffs.AffectedLineIDs.AsSlice()...)).
				Exec(ctx)
			if err != nil {
				return nil, fmt.Errorf("updating updated_at for lines: %w", err)
			}
		}

		// Step 6: Refetch the lines, as due to the upserts we doesn't have a full view of the data

		// We will include deleted lines, as we need to return all the lines even if the edit function marked them as deleted.
		return tx.refetchInvoiceLines(ctx, refetchInvoiceLinesInput{
			Namespace: input.Namespace,
			LineIDs: lo.Map(input.Lines, func(line *billing.Line, _ int) string {
				return line.ID
			}),
			IncludeDeleted: true,
			SchemaLevel:    input.SchemaLevel,
			InvoiceID:      input.InvoiceID,
		})
	})
}

func (a *adapter) upsertFeeLineConfig(ctx context.Context, in detailedLineDiff) error {
	return upsertWithOptions(ctx, a.db, in, upsertInput[detailedLineWithParent, *db.BillingInvoiceFlatFeeLineConfigCreate]{
		Create: func(tx *db.Client, lineWithParent detailedLineWithParent) (*db.BillingInvoiceFlatFeeLineConfigCreate, error) {
			line := lineWithParent.Entity

			if line.FeeLineConfigID == "" {
				line.FeeLineConfigID = ulid.Make().String()
			}

			create := tx.BillingInvoiceFlatFeeLineConfig.Create().
				SetNamespace(line.Namespace).
				SetPerUnitAmount(line.PerUnitAmount).
				SetCategory(line.Category).
				SetPaymentTerm(line.PaymentTerm).
				SetID(line.FeeLineConfigID).
				SetNillableIndex(line.Index)
			return create, nil
		},
		UpsertItems: func(ctx context.Context, tx *db.Client, items []*db.BillingInvoiceFlatFeeLineConfigCreate) error {
			return tx.BillingInvoiceFlatFeeLineConfig.
				CreateBulk(items...).
				OnConflict(
					sql.ConflictColumns(billinginvoiceflatfeelineconfig.FieldID),
					sql.ResolveWithNewValues(),
				).
				UpdateIndex().
				Exec(ctx)
		},
	})
}

func (a *adapter) upsertDetailedLines(ctx context.Context, in detailedLineDiff) error {
	detailedLineUpsertConfig := upsertInput[detailedLineWithParent, *db.BillingInvoiceLineCreate]{
		Create: func(tx *db.Client, lineWithParent detailedLineWithParent) (*db.BillingInvoiceLineCreate, error) {
			line := lineWithParent.Entity

			if line.ID == "" {
				line.ID = ulid.Make().String()
			}

			create := tx.BillingInvoiceLine.Create().
				SetID(line.ID).
				SetNamespace(line.Namespace).
				SetInvoiceID(line.InvoiceID).
				SetPeriodStart(line.ServicePeriod.Start.In(time.UTC)).
				SetPeriodEnd(line.ServicePeriod.End.In(time.UTC)).
				SetParentLineID(lineWithParent.Parent.ID).
				SetInvoiceAt(lineWithParent.Parent.InvoiceAt.In(time.UTC)).
				SetNillableDeletedAt(line.DeletedAt).
				SetStatus(billing.InvoiceLineStatusDetailed).
				SetManagedBy(billing.SystemManagedLine).
				SetType(billing.InvoiceLineTypeFee).
				SetName(line.Name).
				SetNillableDescription(line.Description).
				SetCurrency(line.Currency).
				SetNillableChildUniqueReferenceID(line.ChildUniqueReferenceID).
				// Totals
				SetAmount(line.Totals.Amount).
				SetChargesTotal(line.Totals.ChargesTotal).
				SetDiscountsTotal(line.Totals.DiscountsTotal).
				SetTaxesTotal(line.Totals.TaxesTotal).
				SetTaxesInclusiveTotal(line.Totals.TaxesInclusiveTotal).
				SetTaxesExclusiveTotal(line.Totals.TaxesExclusiveTotal).
				SetTotal(line.Totals.Total).
				// ExternalIDs
				SetNillableInvoicingAppExternalID(lo.EmptyableToPtr(line.ExternalIDs.Invoicing))

			if line.TaxConfig != nil {
				create = create.SetTaxConfig(*line.TaxConfig)
			}

			create = create.SetQuantity(line.Quantity).
				SetFlatFeeLineID(line.FeeLineConfigID).
				SetNillableUsageBasedLineID(nil)

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
				UpdateQuantity().
				UpdateChildUniqueReferenceID().
				Exec(ctx)
		},
		MarkDeleted: func(ctx context.Context, line detailedLineWithParent) (detailedLineWithParent, error) {
			line.Entity.DeletedAt = lo.ToPtr(clock.Now().In(time.UTC))
			return line, nil
		},
	}

	return upsertWithOptions(ctx, a.db, in, detailedLineUpsertConfig)
}

func (a *adapter) upsertDetailedLineAmountDiscounts(ctx context.Context, in detailedLineAmountDiscountDiff) error {
	return upsertWithOptions(ctx, a.db, in, upsertInput[detailedLineAmountDiscountWithParent, *db.BillingInvoiceLineDiscountCreate]{
		Create: func(tx *db.Client, d detailedLineAmountDiscountWithParent) (*db.BillingInvoiceLineDiscountCreate, error) {
			discount := d.Entity

			if discount.ID == "" {
				discount.ID = ulid.Make().String()
			}

			create := tx.BillingInvoiceLineDiscount.Create().
				SetID(discount.ID).
				SetNamespace(d.Parent.GetNamespace()).
				SetLineID(d.Parent.GetID()).
				SetReason(discount.Reason.Type()).
				SetSourceDiscount(lo.ToPtr(discount.Reason)).
				SetAmount(discount.Amount).
				SetNillableRoundingAmount(lo.EmptyableToPtr(discount.RoundingAmount)).
				SetNillableDeletedAt(discount.DeletedAt).
				SetNillableChildUniqueReferenceID(discount.ChildUniqueReferenceID).
				SetNillableDescription(discount.Description).
				// ExternalIDs
				SetNillableInvoicingAppExternalID(lo.EmptyableToPtr(discount.ExternalIDs.Invoicing))

			return create, nil
		},
		UpsertItems: func(ctx context.Context, tx *db.Client, items []*db.BillingInvoiceLineDiscountCreate) error {
			return tx.BillingInvoiceLineDiscount.
				CreateBulk(items...).
				OnConflict(
					sql.ConflictColumns(billinginvoicelinediscount.FieldID),
					sql.ResolveWithNewValues(),
					sql.ResolveWith(func(u *sql.UpdateSet) {
						u.SetIgnore(billinginvoicelinediscount.FieldCreatedAt)
					}),
				).Exec(ctx)
		},
		MarkDeleted: func(ctx context.Context, d detailedLineAmountDiscountWithParent) (detailedLineAmountDiscountWithParent, error) {
			d.Entity.DeletedAt = lo.ToPtr(clock.Now().In(time.UTC))

			return d, nil
		},
	})
}

func (a *adapter) upsertDetailedLinesV2(ctx context.Context, in detailedLineDiff) error {
	detailedLineUpsertConfig := upsertInput[detailedLineWithParent, *db.BillingStandardInvoiceDetailedLineCreate]{
		Create: func(tx *db.Client, lineWithParent detailedLineWithParent) (*db.BillingStandardInvoiceDetailedLineCreate, error) {
			line := lineWithParent.Entity

			if line.ID == "" {
				line.ID = ulid.Make().String()
			}

			create := tx.BillingStandardInvoiceDetailedLine.Create().
				SetID(line.ID).
				SetNamespace(line.Namespace).
				SetInvoiceID(line.InvoiceID).
				SetServicePeriodStart(line.ServicePeriod.Start.In(time.UTC)).
				SetServicePeriodEnd(line.ServicePeriod.End.In(time.UTC)).
				SetParentLineID(lineWithParent.Parent.ID).
				SetNillableDeletedAt(line.DeletedAt).
				SetName(line.Name).
				SetNillableDescription(line.Description).
				SetCurrency(line.Currency).
				SetNillableChildUniqueReferenceID(line.ChildUniqueReferenceID).
				// Totals
				SetAmount(line.Totals.Amount).
				SetChargesTotal(line.Totals.ChargesTotal).
				SetDiscountsTotal(line.Totals.DiscountsTotal).
				SetTaxesTotal(line.Totals.TaxesTotal).
				SetTaxesInclusiveTotal(line.Totals.TaxesInclusiveTotal).
				SetTaxesExclusiveTotal(line.Totals.TaxesExclusiveTotal).
				SetTotal(line.Totals.Total).
				// ExternalIDs
				SetNillableInvoicingAppExternalID(lo.EmptyableToPtr(line.ExternalIDs.Invoicing))

			if line.TaxConfig != nil {
				create = create.SetTaxConfig(*line.TaxConfig)
			}

			return create, nil
		},
		UpsertItems: func(ctx context.Context, tx *db.Client, items []*db.BillingStandardInvoiceDetailedLineCreate) error {
			return tx.BillingStandardInvoiceDetailedLine.
				CreateBulk(items...).
				OnConflict(sql.ConflictColumns(billingstandardinvoicedetailedline.FieldID),
					sql.ResolveWithNewValues(),
					sql.ResolveWith(func(u *sql.UpdateSet) {
						u.SetIgnore(billingstandardinvoicedetailedline.FieldCreatedAt)
					})).
				UpdateQuantity().
				UpdateChildUniqueReferenceID().
				Exec(ctx)
		},
		MarkDeleted: func(ctx context.Context, line detailedLineWithParent) (detailedLineWithParent, error) {
			line.Entity.DeletedAt = lo.ToPtr(clock.Now().In(time.UTC))
			return line, nil
		},
	}

	return upsertWithOptions(ctx, a.db, in, detailedLineUpsertConfig)
}

func (a *adapter) upsertDetailedLineAmountDiscountsV2(ctx context.Context, in detailedLineAmountDiscountDiff) error {
	return upsertWithOptions(ctx, a.db, in, upsertInput[detailedLineAmountDiscountWithParent, *db.BillingStandardInvoiceDetailedLineAmountDiscountCreate]{
		Create: func(tx *db.Client, d detailedLineAmountDiscountWithParent) (*db.BillingStandardInvoiceDetailedLineAmountDiscountCreate, error) {
			discount := d.Entity

			if discount.ID == "" {
				discount.ID = ulid.Make().String()
			}

			create := tx.BillingStandardInvoiceDetailedLineAmountDiscount.Create().
				SetID(discount.ID).
				SetNamespace(d.Parent.GetNamespace()).
				SetLineID(d.Parent.GetID()).
				SetReason(discount.Reason.Type()).
				SetSourceDiscount(lo.ToPtr(discount.Reason)).
				SetAmount(discount.Amount).
				SetNillableRoundingAmount(lo.EmptyableToPtr(discount.RoundingAmount)).
				SetNillableDeletedAt(discount.DeletedAt).
				SetNillableChildUniqueReferenceID(discount.ChildUniqueReferenceID).
				SetNillableDescription(discount.Description).
				// ExternalIDs
				SetNillableInvoicingAppExternalID(lo.EmptyableToPtr(discount.ExternalIDs.Invoicing))

			return create, nil
		},
		UpsertItems: func(ctx context.Context, tx *db.Client, items []*db.BillingStandardInvoiceDetailedLineAmountDiscountCreate) error {
			return tx.BillingStandardInvoiceDetailedLineAmountDiscount.
				CreateBulk(items...).
				OnConflict(
					sql.ConflictColumns(billingstandardinvoicedetailedlineamountdiscount.FieldID),
					sql.ResolveWithNewValues(),
					sql.ResolveWith(func(u *sql.UpdateSet) {
						u.SetIgnore(billingstandardinvoicedetailedlineamountdiscount.FieldCreatedAt)
					}),
				).Exec(ctx)
		},
		MarkDeleted: func(ctx context.Context, d detailedLineAmountDiscountWithParent) (detailedLineAmountDiscountWithParent, error) {
			d.Entity.DeletedAt = lo.ToPtr(clock.Now().In(time.UTC))

			return d, nil
		},
	})
}

func (a *adapter) upsertUsageBasedConfig(ctx context.Context, lineDiffs entitydiff.Diff[*billing.Line]) error {
	return upsertWithOptions(ctx, a.db, lineDiffs, upsertInput[*billing.Line, *db.BillingInvoiceUsageBasedLineConfigCreate]{
		Create: func(tx *db.Client, line *billing.Line) (*db.BillingInvoiceUsageBasedLineConfigCreate, error) {
			if line.UsageBased.ConfigID == "" {
				line.UsageBased.ConfigID = ulid.Make().String()
			}

			create := tx.BillingInvoiceUsageBasedLineConfig.Create().
				SetNamespace(line.Namespace).
				SetPriceType(line.UsageBased.Price.Type()).
				SetPrice(line.UsageBased.Price).
				SetFeatureKey(line.UsageBased.FeatureKey).
				SetID(line.UsageBased.ConfigID).
				SetNillablePreLinePeriodQuantity(line.UsageBased.PreLinePeriodQuantity).
				SetNillableMeteredQuantity(line.UsageBased.MeteredQuantity).
				SetNillableMeteredPreLinePeriodQuantity(line.UsageBased.MeteredPreLinePeriodQuantity)

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
}

// TODO[OM-982]: Add pagination
func (a *adapter) ListInvoiceLines(ctx context.Context, input billing.ListInvoiceLinesAdapterInput) ([]*billing.Line, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]*billing.Line, error) {
		query := tx.db.BillingInvoice.Query().
			Where(billinginvoice.Namespace(input.Namespace))

		if input.CustomerID != "" {
			query = query.Where(billinginvoice.CustomerID(input.CustomerID))
		}

		if len(input.InvoiceStatuses) > 0 {
			query = query.Where(billinginvoice.StatusIn(input.InvoiceStatuses...))
		}

		query = query.WithBillingInvoiceLines(func(q *db.BillingInvoiceLineQuery) {
			q = q.Where(billinginvoiceline.Namespace(input.Namespace))

			if len(input.LineIDs) > 0 {
				q = q.Where(billinginvoiceline.IDIn(input.LineIDs...))
			}

			if len(input.InvoiceIDs) > 0 {
				q = q.Where(billinginvoiceline.InvoiceIDIn(input.InvoiceIDs...))
			}

			if !input.IncludeDeleted {
				q = q.Where(billinginvoiceline.DeletedAtIsNil())
			}

			if len(input.Statuses) > 0 {
				q = q.Where(billinginvoiceline.StatusIn(input.Statuses...))
			}

			tx.expandLineItemsWithDetailedLines(q)
		})

		dbInvoices, err := query.All(ctx)
		if err != nil {
			return nil, err
		}

		lines := lo.FlatMap(dbInvoices, func(dbInvoice *db.BillingInvoice, _ int) []*db.BillingInvoiceLine {
			return dbInvoice.Edges.BillingInvoiceLines
		})

		schemaLevelByInvoiceID := lo.SliceToMap(dbInvoices, func(dbInvoice *db.BillingInvoice) (string, int) {
			return dbInvoice.ID, dbInvoice.SchemaLevel
		})

		mappedLines, err := tx.mapInvoiceLineFromDB(schemaLevelByInvoiceID, lines)
		if err != nil {
			return nil, err
		}

		// Let's expand the line hierarchy so that we can have a full view of the split line groups
		linesWithHierarchy, err := tx.expandSplitLineHierarchy(ctx, input.Namespace, mappedLines)
		if err != nil {
			return nil, err
		}

		return linesWithHierarchy, nil
	})
}

// expandLineItems is a helper function to expand the line items in the query, detailed lines are not included
func (a *adapter) expandLineItems(q *db.BillingInvoiceLineQuery) *db.BillingInvoiceLineQuery {
	return q.WithFlatFeeLine().
		WithUsageBasedLine().
		WithLineUsageDiscounts(
			func(q *db.BillingInvoiceLineUsageDiscountQuery) {
				q.Where(billinginvoicelineusagediscount.DeletedAtIsNil())
			},
		).
		WithLineAmountDiscounts(
			func(q *db.BillingInvoiceLineDiscountQuery) {
				q.Where(billinginvoicelinediscount.DeletedAtIsNil())
			},
		)
}

// expandLineItemsWithDetailedLines expands the invoice lines and their detailed lines if any exists
func (a *adapter) expandLineItemsWithDetailedLines(q *db.BillingInvoiceLineQuery) *db.BillingInvoiceLineQuery {
	q = a.expandLineItems(q)

	q.WithDetailedLines(func(bilq *db.BillingInvoiceLineQuery) {
		// We never include deleted detailed lines in the query, as we intent to keep them as history.
		//
		// If we want to reuse the deleted lines in ChildrenWithIDReuse, we must make sure that non-deleted lines are
		// prioritized for reuse or we will end up with INSERT conflicts due to the child unique reference id uniqueness constraint.
		bilq = bilq.Where(billinginvoiceline.DeletedAtIsNil())

		a.expandLineItems(bilq)
	})

	q.WithDetailedLinesV2(func(bilq *db.BillingStandardInvoiceDetailedLineQuery) {
		// We never include deleted detailed lines in the query, as we intent to keep them as history.
		//
		// If we want to reuse the deleted lines in ChildrenWithIDReuse, we must make sure that non-deleted lines are
		// prioritized for reuse or we will end up with INSERT conflicts due to the child unique reference id uniqueness constraint.
		bilq.Where(billingstandardinvoicedetailedline.DeletedAtIsNil()).
			WithAmountDiscounts(func(bilq *db.BillingStandardInvoiceDetailedLineAmountDiscountQuery) {
				bilq.Where(billingstandardinvoicedetailedlineamountdiscount.DeletedAtIsNil())
			})
	})

	return q
}

type refetchInvoiceLinesInput struct {
	Namespace      string
	LineIDs        []string
	IncludeDeleted bool
	SchemaLevel    int
	InvoiceID      string
}

func (i refetchInvoiceLinesInput) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if i.SchemaLevel < 1 {
		return errors.New("schema level must be at least 1")
	}

	if i.InvoiceID == "" {
		return errors.New("invoice id is required")
	}

	return nil
}

func (a *adapter) refetchInvoiceLines(ctx context.Context, in refetchInvoiceLinesInput) ([]*billing.Line, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	query := a.db.BillingInvoiceLine.Query().
		Where(billinginvoiceline.Namespace(in.Namespace)).
		Where(billinginvoiceline.IDIn(in.LineIDs...))

	if !in.IncludeDeleted {
		query = query.Where(billinginvoiceline.DeletedAtIsNil())
	}

	query = a.expandLineItemsWithDetailedLines(query)

	dbLines, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching lines: %w", err)
	}

	if len(dbLines) != len(in.LineIDs) {
		return nil, fmt.Errorf("not all lines were created")
	}

	dbLinesByID := lo.GroupBy(dbLines, func(line *db.BillingInvoiceLine) string {
		return line.ID
	})

	dbLinesInSameOrder, err := slicesx.MapWithErr(in.LineIDs, func(id string) (*db.BillingInvoiceLine, error) {
		line, ok := dbLinesByID[id]
		if !ok || len(line) < 1 {
			return nil, fmt.Errorf("line not found: %s", id)
		}

		return line[0], nil
	})
	if err != nil {
		return nil, err
	}

	lines, err := a.mapInvoiceLineFromDB(map[string]int{in.InvoiceID: in.SchemaLevel}, dbLinesInSameOrder)
	if err != nil {
		return nil, err
	}

	// Let's expand the line hierarchy so that we can have a full view of the invoice during the upcoming calculations
	linesWithHierarchy, err := a.expandSplitLineHierarchy(ctx, in.Namespace, lines)
	if err != nil {
		return nil, err
	}

	return linesWithHierarchy, nil
}

func (a *adapter) GetLinesForSubscription(ctx context.Context, in billing.GetLinesForSubscriptionInput) ([]billing.LineOrHierarchy, error) {
	if err := in.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]billing.LineOrHierarchy, error) {
		query := tx.db.BillingInvoiceLine.Query().
			Where(billinginvoiceline.Namespace(in.Namespace)).
			Where(billinginvoiceline.SubscriptionID(in.SubscriptionID)).
			Where(billinginvoiceline.ParentLineIDIsNil()) // This one is required so that we are not fetching split line's children directly, the mapper will handle that

		query = tx.expandLineItems(query)

		dbLines, err := query.All(ctx)
		if err != nil {
			return nil, fmt.Errorf("fetching lines: %w", err)
		}

		invoiceSchemaLevelByID, err := tx.getSchemaLevelPerInvoice(ctx, customer.CustomerID{
			Namespace: in.Namespace,
			ID:        in.CustomerID,
		})
		if err != nil {
			return nil, fmt.Errorf("getting schema level per invoice: %w", err)
		}

		lines, err := tx.mapInvoiceLineFromDB(invoiceSchemaLevelByID, dbLines)
		if err != nil {
			return nil, fmt.Errorf("mapping lines: %w", err)
		}

		dbGroups, err := tx.db.BillingInvoiceSplitLineGroup.Query().
			Where(billinginvoicesplitlinegroup.Namespace(in.Namespace)).
			Where(billinginvoicesplitlinegroup.SubscriptionID(in.SubscriptionID)).
			WithBillingInvoiceLines(func(q *db.BillingInvoiceLineQuery) {
				tx.expandLineItems(q)
				q.WithBillingInvoice()
			}).All(ctx)
		if err != nil {
			return nil, fmt.Errorf("fetching split line groups: %w", err)
		}

		groups, err := slicesx.MapWithErr(dbGroups, func(dbGroup *db.BillingInvoiceSplitLineGroup) (billing.SplitLineHierarchy, error) {
			group, err := tx.mapSplitLineGroupFromDB(dbGroup)
			if err != nil {
				return billing.SplitLineHierarchy{}, err
			}

			lines, err := slicesx.MapWithErr(dbGroup.Edges.BillingInvoiceLines, func(dbLine *db.BillingInvoiceLine) (billing.LineWithInvoiceHeader, error) {
				line, err := tx.mapInvoiceLineWithoutReferences(dbLine)
				if err != nil {
					return billing.LineWithInvoiceHeader{}, err
				}

				return billing.LineWithInvoiceHeader{
					Line:    line,
					Invoice: tx.mapInvoiceBaseFromDB(ctx, dbLine.Edges.BillingInvoice),
				}, nil
			})
			if err != nil {
				return billing.SplitLineHierarchy{}, err
			}

			return billing.SplitLineHierarchy{
				Group: group,
				Lines: lines,
			}, nil
		})
		if err != nil {
			return nil, fmt.Errorf("mapping groups: %w", err)
		}

		// Sanity check: let's make sure that there are no items with overlapping childUniqueReferenceID
		groupUniqueReferenceIDs := lo.Map(
			lo.Filter(
				groups,
				func(group billing.SplitLineHierarchy, _ int) bool {
					return group.Group.UniqueReferenceID != nil
				},
			),
			func(group billing.SplitLineHierarchy, _ int) string {
				return lo.FromPtr(group.Group.UniqueReferenceID)
			},
		)

		lineChildUniqueReferenceIDs := lo.Map(
			lo.Filter( // Lines can have a nil childUniqueReferenceID, when they are part of a split line group (e.g. the group has the unique reference id)
				lines,
				func(line *billing.Line, _ int) bool {
					return line.ChildUniqueReferenceID != nil
				},
			),
			func(line *billing.Line, _ int) string {
				return lo.FromPtr(line.ChildUniqueReferenceID)
			},
		)

		overlappingChildUniqueReferenceIDs := lo.Intersect(groupUniqueReferenceIDs, lineChildUniqueReferenceIDs)

		if len(overlappingChildUniqueReferenceIDs) > 0 {
			return nil, fmt.Errorf("overlapping childUniqueReferenceID: %v", overlappingChildUniqueReferenceIDs)
		}

		// Let's map to the union type
		out := make([]billing.LineOrHierarchy, 0, len(groups)+len(lines))

		out = append(out, lo.Map(groups, func(h billing.SplitLineHierarchy, _ int) billing.LineOrHierarchy {
			return billing.NewLineOrHierarchy(&h)
		})...)

		out = append(out, lo.Map(lines, func(line *billing.Line, _ int) billing.LineOrHierarchy {
			return billing.NewLineOrHierarchy(line)
		})...)

		return out, nil
	})
}
