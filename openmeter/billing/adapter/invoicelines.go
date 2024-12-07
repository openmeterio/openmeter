package billingadapter

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoice"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceflatfeelineconfig"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceline"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoicelinediscount"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceusagebasedlineconfig"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

var _ billing.InvoiceLineAdapter = (*adapter)(nil)

func (a *adapter) UpsertInvoiceLines(ctx context.Context, inputIn billing.UpsertInvoiceLinesAdapterInput) ([]*billingentity.Line, error) {
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
		if line.Status == billingentity.InvoiceLineStatusSplit &&
			line.Children.IsPresent() {
			return nil, fmt.Errorf("updating split line's detailed lines is not supported")
		}
	}

	input := &billing.UpsertInvoiceLinesAdapterInput{
		Namespace: inputIn.Namespace,
		Lines: lo.Map(inputIn.Lines, func(line *billingentity.Line, _ int) *billingentity.Line {
			return line.Clone()
		}),
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]*billingentity.Line, error) {
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
		lineUpsertConfig := upsertInput[*billingentity.Line, *db.BillingInvoiceLineCreate]{
			Create: func(tx *db.Client, line *billingentity.Line) (*db.BillingInvoiceLineCreate, error) {
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
					SetInvoiceAt(line.InvoiceAt.In(time.UTC)).
					SetStatus(line.Status).
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

				if line.TaxConfig != nil {
					create = create.SetTaxConfig(*line.TaxConfig)
				}

				switch line.Type {
				case billingentity.InvoiceLineTypeFee:
					create = create.SetQuantity(line.FlatFee.Quantity).
						SetFlatFeeLineID(line.FlatFee.ConfigID).
						SetNillableUsageBasedLineID(nil)
				case billingentity.InvoiceLineTypeUsageBased:
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
					ClearDeletedAt().
					UpdateQuantity().
					Exec(ctx)
			},
			Delete: func(ctx context.Context, tx *db.Client, items []*billingentity.Line) error {
				return tx.BillingInvoiceLine.Update().
					SetDeletedAt(clock.Now().In(time.UTC)).
					Where(billinginvoiceline.IDIn(lo.Map(items, func(line *billingentity.Line, _ int) string {
						return line.ID
					})...)).
					Exec(ctx)
			},
		}

		if err := upsertWithOptions(ctx, tx.db, lineDiffs.LineBase, lineUpsertConfig); err != nil {
			return nil, fmt.Errorf("creating lines: %w", err)
		}

		// Step 3: Let's create the detailed lines
		flattenedDetailedLines := lo.FlatMap(input.Lines, func(_ *billingentity.Line, idx int) []*billingentity.Line {
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

		// Step 4a: Discounts

		allDiscountDiffs := unionOfDiffs(lineDiffs.Discounts, lineDiffs.ChildrenDiff.Discounts)
		err = upsertWithOptions(ctx, tx.db, allDiscountDiffs, upsertInput[discountWithLine, *db.BillingInvoiceLineDiscountCreate]{
			Create: func(tx *db.Client, d discountWithLine) (*db.BillingInvoiceLineDiscountCreate, error) {
				if d.Discount.ID == "" {
					d.Discount.ID = ulid.Make().String()
				}

				create := tx.BillingInvoiceLineDiscount.Create().
					SetID(d.Discount.ID).
					SetNamespace(d.Line.Namespace).
					SetLineID(d.Line.ID).
					SetAmount(d.Discount.Amount).
					SetNillableChildUniqueReferenceID(d.Discount.ChildUniqueReferenceID).
					SetNillableDescription(d.Discount.Description)

				return create, nil
			},
			UpsertItems: func(ctx context.Context, tx *db.Client, items []*db.BillingInvoiceLineDiscountCreate) error {
				return tx.BillingInvoiceLineDiscount.
					CreateBulk(items...).
					OnConflict(
						sql.ConflictColumns(billinginvoicelinediscount.FieldID),
						sql.ResolveWithNewValues(),
						sql.ResolveWith(func(u *sql.UpdateSet) {
							u.SetIgnore(billinginvoiceline.FieldCreatedAt)
						}),
					).Exec(ctx)
			},
			Delete: func(ctx context.Context, tx *db.Client, items []discountWithLine) error {
				return tx.BillingInvoiceLineDiscount.Update().
					SetDeletedAt(clock.Now().In(time.UTC)).
					Where(billinginvoicelinediscount.IDIn(lo.Map(items, func(d discountWithLine, _ int) string {
						return d.Discount.ID
					})...)).
					Exec(ctx)
			},
		})
		if err != nil {
			return nil, fmt.Errorf("upserting discounts: %w", err)
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
		return tx.fetchLines(ctx, input.Namespace, lo.Map(input.Lines, func(line *billingentity.Line, _ int) string {
			return line.ID
		}))
	})
}

func (a *adapter) upsertFeeLineConfig(ctx context.Context, in diff[*billingentity.Line]) error {
	return upsertWithOptions(ctx, a.db, in, upsertInput[*billingentity.Line, *db.BillingInvoiceFlatFeeLineConfigCreate]{
		Create: func(tx *db.Client, line *billingentity.Line) (*db.BillingInvoiceFlatFeeLineConfigCreate, error) {
			if line.FlatFee.ConfigID == "" {
				line.FlatFee.ConfigID = ulid.Make().String()
			}

			create := tx.BillingInvoiceFlatFeeLineConfig.Create().
				SetNamespace(line.Namespace).
				SetPerUnitAmount(line.FlatFee.PerUnitAmount).
				SetID(line.FlatFee.ConfigID)

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

func (a *adapter) upsertUsageBasedConfig(ctx context.Context, lineDiffs diff[*billingentity.Line]) error {
	return upsertWithOptions(ctx, a.db, lineDiffs, upsertInput[*billingentity.Line, *db.BillingInvoiceUsageBasedLineConfigCreate]{
		Create: func(tx *db.Client, line *billingentity.Line) (*db.BillingInvoiceUsageBasedLineConfigCreate, error) {
			if line.UsageBased.ConfigID == "" {
				line.UsageBased.ConfigID = ulid.Make().String()
			}

			create := tx.BillingInvoiceUsageBasedLineConfig.Create().
				SetNamespace(line.Namespace).
				SetPriceType(line.UsageBased.Price.Type()).
				SetPrice(&line.UsageBased.Price).
				SetFeatureKey(line.UsageBased.FeatureKey).
				SetID(line.UsageBased.ConfigID).
				SetNillablePreLinePeriodQuantity(line.UsageBased.PreLinePeriodQuantity)

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
func (a *adapter) ListInvoiceLines(ctx context.Context, input billing.ListInvoiceLinesAdapterInput) ([]*billingentity.Line, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]*billingentity.Line, error) {
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

		return tx.mapInvoiceLineFromDB(ctx, lines)
	})
}

// expandLineItems is a helper function to expand the line items in the query, given that the mapper
// will handle the parent/child fetching it's fine to only fetch items that we need to reconstruct
// this specific entity.
func (a *adapter) expandLineItems(q *db.BillingInvoiceLineQuery) *db.BillingInvoiceLineQuery {
	return q.WithFlatFeeLine().
		WithUsageBasedLine().
		WithLineDiscounts(
			func(q *db.BillingInvoiceLineDiscountQuery) {
				q.Where(billinginvoicelinediscount.DeletedAtIsNil())
			},
		)
}

func (a *adapter) AssociateLinesToInvoice(ctx context.Context, input billing.AssociateLinesToInvoiceAdapterInput) ([]*billingentity.Line, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]*billingentity.Line, error) {
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

		return tx.fetchLines(ctx, input.Invoice.Namespace, input.LineIDs)
	})
}

func (a *adapter) fetchLines(ctx context.Context, ns string, lineIDs []string) ([]*billingentity.Line, error) {
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

	return a.mapInvoiceLineFromDB(ctx, dbLinesInSameOrder)
}

func (a *adapter) GetInvoiceLine(ctx context.Context, input billing.GetInvoiceLineAdapterInput) (*billingentity.Line, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (*billingentity.Line, error) {
		query := tx.db.BillingInvoiceLine.Query().
			Where(billinginvoiceline.Namespace(input.Namespace)).
			Where(billinginvoiceline.ID(input.ID))

		query = tx.expandLineItems(query)

		dbLine, err := query.First(ctx)
		if err != nil {
			return nil, fmt.Errorf("fetching line: %w", err)
		}

		mappedLines, err := tx.mapInvoiceLineFromDB(ctx, []*db.BillingInvoiceLine{dbLine})
		if err != nil {
			return nil, fmt.Errorf("mapping line: %w", err)
		}

		if len(mappedLines) != 1 {
			return nil, billingentity.NotFoundError{
				Err: fmt.Errorf("line not found: %s", input.ID),
			}
		}

		return mappedLines[0], nil
	})
}

func (a *adapter) GetInvoiceLineOwnership(ctx context.Context, in billing.GetInvoiceLineOwnershipAdapterInput) (billing.GetOwnershipAdapterResponse, error) {
	if err := in.Validate(); err != nil {
		return billing.GetOwnershipAdapterResponse{}, billingentity.ValidationError{
			Err: err,
		}
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (billing.GetOwnershipAdapterResponse, error) {
		dbInvoice, err := tx.db.BillingInvoiceLine.Query().
			Where(billinginvoiceline.ID(in.ID)).
			Where(billinginvoiceline.Namespace(in.Namespace)).
			WithBillingInvoice().
			First(ctx)
		if err != nil {
			if db.IsNotFound(err) {
				return billing.GetOwnershipAdapterResponse{}, billingentity.NotFoundError{
					Entity: billingentity.EntityInvoice,
					ID:     in.ID,
					Err:    err,
				}
			}

			return billing.GetOwnershipAdapterResponse{}, err
		}

		return billing.GetOwnershipAdapterResponse{
			Namespace:  dbInvoice.Namespace,
			InvoiceID:  dbInvoice.InvoiceID,
			CustomerID: dbInvoice.Edges.BillingInvoice.CustomerID,
		}, nil
	})
}
