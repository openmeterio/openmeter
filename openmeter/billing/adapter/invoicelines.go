package billingadapter

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoice"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceflatfeelineconfig"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceline"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoicelinediscount"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoicelineusagediscount"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceusagebasedlineconfig"
	"github.com/openmeterio/openmeter/pkg/clock"
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

	// TODO[OM-1015]: Updating split line's children is not supported (yet)
	for _, line := range inputIn.Lines {
		if line.Status == billing.InvoiceLineStatusSplit &&
			line.Children.IsPresent() {
			return nil, fmt.Errorf("updating split line's detailed lines is not supported")
		}
	}

	input := &billing.UpsertInvoiceLinesAdapterInput{
		Namespace: inputIn.Namespace,
		Lines: lo.Map(inputIn.Lines, func(line *billing.Line, _ int) *billing.Line {
			return line.Clone()
		}),
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]*billing.Line, error) {
		// Let's genereate the line diffs first
		lineDiffs, err := diffInvoiceLines(input.Lines)
		if err != nil {
			return nil, fmt.Errorf("generating line diffs: %w", err)
		}

		// Step 1: Let's create/upsert the line configs first
		if err = tx.upsertFeeLineConfig(ctx,
			unionOfDiffs(lineDiffs.FlatFee, lineDiffs.ChildrenDiff.FlatFee)); err != nil {
			return nil, fmt.Errorf("upserting fee line configs: %w", err)
		}

		if err := tx.upsertUsageBasedConfig(ctx,
			unionOfDiffs(lineDiffs.UsageBased, lineDiffs.ChildrenDiff.UsageBased)); err != nil {
			return nil, fmt.Errorf("upserting usage based line configs: %w", err)
		}

		// Step 2: Let's create the lines, but not their detailed lines
		lineUpsertConfig := upsertInput[*billing.Line, *db.BillingInvoiceLineCreate]{
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
					SetNillableDeletedAt(line.DeletedAt).
					SetInvoiceAt(line.InvoiceAt.In(time.UTC)).
					SetStatus(line.Status).
					SetManagedBy(line.ManagedBy).
					SetType(line.Type).
					SetName(line.Name).
					SetNillableDescription(line.Description).
					SetCurrency(line.Currency).
					SetMetadata(line.Metadata).
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
						SetSubscriptionItemID(line.Subscription.ItemID)
				}

				if line.TaxConfig != nil {
					create = create.SetTaxConfig(*line.TaxConfig)
				}

				if !line.RateCardDiscounts.IsEmpty() {
					create = create.SetRatecardDiscounts(lo.ToPtr(line.RateCardDiscounts))
				}

				switch line.Type {
				case billing.InvoiceLineTypeFee:
					create = create.SetQuantity(line.FlatFee.Quantity).
						SetFlatFeeLineID(line.FlatFee.ConfigID).
						SetNillableUsageBasedLineID(nil)
				case billing.InvoiceLineTypeUsageBased:
					create = create.
						SetNillableQuantity(line.UsageBased.Quantity).
						SetUsageBasedLineID(line.UsageBased.ConfigID).
						SetNillableFlatFeeLineID(nil)

				default:
					return nil, fmt.Errorf("unsupported type: %s", line.Type)
				}

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
					Exec(ctx)
			},
			MarkDeleted: func(ctx context.Context, line *billing.Line) (*billing.Line, error) {
				line.DeletedAt = lo.ToPtr(clock.Now().In(time.UTC))
				return line, nil
			},
		}

		if err := upsertWithOptions(ctx, tx.db, lineDiffs.LineBase, lineUpsertConfig); err != nil {
			return nil, fmt.Errorf("creating lines: %w", err)
		}

		// Step 3: Let's create the detailed lines
		flattenedDetailedLines := lo.FlatMap(input.Lines, func(_ *billing.Line, idx int) []*billing.Line {
			return input.Lines[idx].Children.OrEmpty()
		})

		if len(flattenedDetailedLines) > 0 {
			// Let's restore the parent <-> child relationship in terms of the ParentLineID field
			for _, line := range input.Lines {
				for _, child := range line.Children.OrEmpty() {
					child.ParentLineID = lo.ToPtr(line.ID)
				}
			}

			if err := upsertWithOptions(ctx, tx.db, lineDiffs.ChildrenDiff.LineBase, lineUpsertConfig); err != nil {
				return nil, fmt.Errorf("[children] creating lines: %w", err)
			}
		}

		// Step 4: Let's upsert anything else, that doesn't have strict ID requirements

		// Step 4a: Line Discounts

		allUsageDiscountDiffs := unionOfDiffs(lineDiffs.UsageDiscounts, lineDiffs.ChildrenDiff.UsageDiscounts)
		err = upsertWithOptions(ctx, tx.db, allUsageDiscountDiffs, upsertInput[usageLineDiscountMangedWithLine, *db.BillingInvoiceLineUsageDiscountCreate]{
			Create: func(tx *db.Client, d usageLineDiscountMangedWithLine) (*db.BillingInvoiceLineUsageDiscountCreate, error) {
				discount := d.Discount

				if discount.ID == "" {
					discount.ID = ulid.Make().String()
				}

				create := tx.BillingInvoiceLineUsageDiscount.Create().
					SetID(discount.ID).
					SetNamespace(d.Line.Namespace).
					SetLineID(d.Line.ID).
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
			MarkDeleted: func(ctx context.Context, d usageLineDiscountMangedWithLine) (usageLineDiscountMangedWithLine, error) {
				d.Discount.DeletedAt = lo.ToPtr(clock.Now().In(time.UTC))

				return d, nil
			},
		})
		if err != nil {
			return nil, fmt.Errorf("upserting usage discounts: %w", err)
		}

		allAmountDiscountDiffs := unionOfDiffs(lineDiffs.AmountDiscounts, lineDiffs.ChildrenDiff.AmountDiscounts)
		err = upsertWithOptions(ctx, tx.db, allAmountDiscountDiffs, upsertInput[amountLineDiscountMangedWithLine, *db.BillingInvoiceLineDiscountCreate]{
			Create: func(tx *db.Client, d amountLineDiscountMangedWithLine) (*db.BillingInvoiceLineDiscountCreate, error) {
				discount := d.Discount

				if discount.ID == "" {
					discount.ID = ulid.Make().String()
				}

				create := tx.BillingInvoiceLineDiscount.Create().
					SetID(discount.ID).
					SetNamespace(d.Line.Namespace).
					SetLineID(d.Line.ID).
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
			MarkDeleted: func(ctx context.Context, d amountLineDiscountMangedWithLine) (amountLineDiscountMangedWithLine, error) {
				d.Discount.DeletedAt = lo.ToPtr(clock.Now().In(time.UTC))

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
		return tx.fetchLines(ctx, input.Namespace, lo.Map(input.Lines, func(line *billing.Line, _ int) string {
			return line.ID
		}))
	})
}

func (a *adapter) upsertFeeLineConfig(ctx context.Context, in diff[*billing.Line]) error {
	return upsertWithOptions(ctx, a.db, in, upsertInput[*billing.Line, *db.BillingInvoiceFlatFeeLineConfigCreate]{
		Create: func(tx *db.Client, line *billing.Line) (*db.BillingInvoiceFlatFeeLineConfigCreate, error) {
			if line.FlatFee.ConfigID == "" {
				line.FlatFee.ConfigID = ulid.Make().String()
			}

			create := tx.BillingInvoiceFlatFeeLineConfig.Create().
				SetNamespace(line.Namespace).
				SetPerUnitAmount(line.FlatFee.PerUnitAmount).
				SetCategory(line.FlatFee.Category).
				SetPaymentTerm(line.FlatFee.PaymentTerm).
				SetID(line.FlatFee.ConfigID)

			if line.Status == billing.InvoiceLineStatusDetailed {
				// TODO[later]: Detailed lines must be a separate entity, so that we don't need these hacks (like line config or type specific sets)
				create = create.SetNillableIndex(line.FlatFee.Index)
			}

			return create, nil
		},
		UpsertItems: func(ctx context.Context, tx *db.Client, items []*db.BillingInvoiceFlatFeeLineConfigCreate) error {
			return tx.BillingInvoiceFlatFeeLineConfig.
				CreateBulk(items...).
				OnConflict(
					sql.ConflictColumns(billinginvoiceflatfeelineconfig.FieldID),
					sql.ResolveWithNewValues(),
				).Exec(ctx)
		},
	})
}

func (a *adapter) upsertUsageBasedConfig(ctx context.Context, lineDiffs diff[*billing.Line]) error {
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

			if input.InvoiceAtBefore != nil {
				q = q.Where(billinginvoiceline.InvoiceAtLT(*input.InvoiceAtBefore))
			}

			if !input.IncludeDeleted {
				q = q.Where(billinginvoiceline.DeletedAtIsNil())
			}

			if len(input.ParentLineIDs) > 0 {
				if input.ParentLineIDsIncludeParent {
					q = q.Where(
						billinginvoiceline.Or(
							billinginvoiceline.ParentLineIDIn(input.ParentLineIDs...),
							billinginvoiceline.IDIn(input.ParentLineIDs...),
						),
					)
				} else {
					q = q.Where(billinginvoiceline.ParentLineIDIn(input.ParentLineIDs...))
				}
			}

			if len(input.Statuses) > 0 {
				q = q.Where(billinginvoiceline.StatusIn(input.Statuses...))
			}

			tx.expandLineItems(q)
		})

		dbInvoices, err := query.All(ctx)
		if err != nil {
			return nil, err
		}

		lines := lo.FlatMap(dbInvoices, func(dbInvoice *db.BillingInvoice, _ int) []*db.BillingInvoiceLine {
			return dbInvoice.Edges.BillingInvoiceLines
		})

		return tx.mapInvoiceLineFromDB(ctx, mapInvoiceLineFromDBInput{
			lines:          lines,
			includeDeleted: input.IncludeDeleted,
		})
	})
}

// expandLineItems is a helper function to expand the line items in the query, given that the mapper
// will handle the parent/child fetching it's fine to only fetch items that we need to reconstruct
// this specific entity.
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

func (a *adapter) AssociateLinesToInvoice(ctx context.Context, input billing.AssociateLinesToInvoiceAdapterInput) ([]*billing.Line, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]*billing.Line, error) {
		nAffected, err := tx.db.BillingInvoiceLine.Update().
			SetInvoiceID(input.Invoice.ID).
			Where(billinginvoiceline.Namespace(input.Invoice.Namespace)).
			Where(billinginvoiceline.IDIn(input.LineIDs...)).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("associating lines: %w", err)
		}

		if nAffected != len(input.LineIDs) {
			return nil, fmt.Errorf("not all lines were associated")
		}

		invoiceLines, err := tx.fetchLines(ctx, input.Invoice.Namespace, input.LineIDs)
		if err != nil {
			return nil, fmt.Errorf("fetching lines: %w", err)
		}

		return invoiceLines, nil
	})
}

func (a *adapter) fetchLines(ctx context.Context, ns string, lineIDs []string) ([]*billing.Line, error) {
	query := a.db.BillingInvoiceLine.Query().
		Where(billinginvoiceline.Namespace(ns)).
		Where(billinginvoiceline.IDIn(lineIDs...))

	query = a.expandLineItems(query)

	dbLines, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching lines: %w", err)
	}

	if len(dbLines) != len(lineIDs) {
		return nil, fmt.Errorf("not all lines were created")
	}

	dbLinesByID := lo.GroupBy(dbLines, func(line *db.BillingInvoiceLine) string {
		return line.ID
	})

	dbLinesInSameOrder, err := slicesx.MapWithErr(lineIDs, func(id string) (*db.BillingInvoiceLine, error) {
		line, ok := dbLinesByID[id]
		if !ok || len(line) < 1 {
			return nil, fmt.Errorf("line not found: %s", id)
		}

		return line[0], nil
	})
	if err != nil {
		return nil, err
	}

	lines, err := a.mapInvoiceLineFromDB(ctx, mapInvoiceLineFromDBInput{
		lines: dbLinesInSameOrder,
	})
	if err != nil {
		return nil, err
	}

	// Let's expand the line hierarchy so that we can have a full view of the invoice during the upcoming calculations
	linesWithHierarchy, err := a.expandProgressiveLineHierarchy(ctx, ns, lines)
	if err != nil {
		return nil, err
	}

	return linesWithHierarchy, nil
}

func (a *adapter) GetLinesForSubscription(ctx context.Context, in billing.GetLinesForSubscriptionInput) ([]*billing.Line, error) {
	if err := in.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]*billing.Line, error) {
		query := tx.db.BillingInvoiceLine.Query().
			Where(billinginvoiceline.Namespace(in.Namespace)).
			Where(billinginvoiceline.SubscriptionID(in.SubscriptionID)).
			Where(billinginvoiceline.ParentLineIDIsNil()) // This one is required so that we are not fetching split line's children directly, the mapper will handle that

		query = tx.expandLineItems(query)

		dbLines, err := query.All(ctx)
		if err != nil {
			return nil, fmt.Errorf("fetching lines: %w", err)
		}

		return tx.mapInvoiceLineFromDB(ctx, mapInvoiceLineFromDBInput{
			lines:          dbLines,
			includeDeleted: true,
		})
	})
}
