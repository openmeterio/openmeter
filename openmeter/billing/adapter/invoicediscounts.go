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
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoicediscount"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type upsertInvoiceDiscountsInput struct {
	OriginalState []billing.InvoiceDiscount
	TargetState   []billing.InvoiceDiscount
}

func (a *adapter) upsertInvoiceDiscounts(ctx context.Context, input upsertInvoiceDiscountsInput) error {
	discountUpsertConfig := upsertInput[billing.InvoiceDiscount, *db.BillingInvoiceDiscountCreate]{
		Create: func(tx *db.Client, discount billing.InvoiceDiscount) (*db.BillingInvoiceDiscountCreate, error) {
			base, err := discount.DiscountBase()
			if err != nil {
				return nil, err
			}

			if base.ID == "" {
				// TODO: do we need a pointer input here or are we refetching either ways?!
				base.ID = ulid.Make().String()
			}

			create := tx.BillingInvoiceDiscount.Create().
				SetID(base.ID).
				SetNamespace(base.Namespace).
				SetCreatedAt(base.CreatedAt).
				SetUpdatedAt(base.UpdatedAt).
				SetNillableDeletedAt(base.DeletedAt).
				SetName(base.Name).
				SetNillableDescription(base.Description).
				SetInvoiceID(base.InvoiceID).
				SetLineIds(base.LineIDs).
				SetType(base.Type)

			switch discount.Type() {
			case billing.PercentageDiscountType:
				percentage, err := discount.AsPercentage()
				if err != nil {
					return nil, err
				}

				create = create.SetAmount(percentage.Percentage)
			default:
				return nil, fmt.Errorf("invalid invoice discount type: %s", discount.Type())
			}

			return create, nil
		},
		UpsertItems: func(ctx context.Context, tx *db.Client, items []*db.BillingInvoiceDiscountCreate) error {
			return tx.BillingInvoiceDiscount.
				CreateBulk(items...).
				OnConflict(sql.ConflictColumns(billinginvoicediscount.FieldID),
					sql.ResolveWithNewValues(),
					sql.ResolveWith(func(u *sql.UpdateSet) {
						u.SetIgnore(billinginvoicediscount.FieldCreatedAt)
					})).
				UpdateDescription().
				Exec(ctx)
		},
		MarkDeleted: func(ctx context.Context, disc billing.InvoiceDiscount) (billing.InvoiceDiscount, error) {
			switch disc.Type() {
			case billing.PercentageDiscountType:
				percentage, err := disc.AsPercentage()
				if err != nil {
					return billing.InvoiceDiscount{}, err
				}

				percentage.DeletedAt = lo.ToPtr(clock.Now().In(time.UTC))
				return billing.NewInvoiceDiscountFrom(percentage), nil
			default:
				return billing.InvoiceDiscount{}, fmt.Errorf("invalid invoice discount type: %s", disc.Type())
			}
		},
	}

	diff, err := a.diffInvoiceDiscounts(input)
	if err != nil {
		return fmt.Errorf("failed to diff invoice discounts: %w", err)
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		if err := upsertWithOptions(ctx, tx.db, diff, discountUpsertConfig); err != nil {
			return fmt.Errorf("failed to upsert invoice discounts: %w", err)
		}

		return nil
	})
}

func (a *adapter) diffInvoiceDiscounts(input upsertInvoiceDiscountsInput) (diff[billing.InvoiceDiscount], error) {
	discountDiff := diff[billing.InvoiceDiscount]{}

	originalLinesByID := make(map[string]billing.InvoiceDiscount, len(input.OriginalState))
	for _, discount := range input.OriginalState {
		base, err := discount.DiscountBase()
		if err != nil {
			return discountDiff, err
		}

		originalLinesByID[base.ID] = discount
	}

	targetLineIDs := make([]string, 0, len(input.TargetState))
	for _, discount := range input.TargetState {
		base, err := discount.DiscountBase()
		if err != nil {
			return discountDiff, err
		}

		// New item
		if base.ID == "" {
			discountDiff.ToCreate = append(discountDiff.ToCreate, discount)
			continue
		}

		// Let's collect the line IDs from the target state
		targetLineIDs = append(targetLineIDs, base.ID)

		originalState, ok := originalLinesByID[base.ID]
		if !ok {
			// Let's not allow adding a discount with a preexisting ID (for now)
			return discountDiff, fmt.Errorf("cannot add discount with preexisting ID: %s", base.ID)
		}

		if !discount.Equals(originalState) {
			if base.DeletedAt != nil {
				discountDiff.ToDelete = append(discountDiff.ToDelete, discount)
			} else {
				discountDiff.ToUpdate = append(discountDiff.ToUpdate, discount)
			}
		}
	}

	// Items that are not in the target state are to be deleted
	missingLineIDsFromTarget, _ := lo.Difference(lo.Keys(originalLinesByID), targetLineIDs)
	for _, id := range missingLineIDsFromTarget {
		discountDiff.ToDelete = append(discountDiff.ToDelete, originalLinesByID[id])
	}

	return discountDiff, nil
}

// GetInvoiceDiscount gets a discount from the database, given its ID. This is not exposed via the adater interface,
// but is used by tests to introspect the database.
func (a *adapter) GetInvoiceDiscount(ctx context.Context, id billing.InvoiceDiscountID) (billing.InvoiceDiscount, error) {
	dbDisc, err := a.db.BillingInvoiceDiscount.Query().
		Where(billinginvoicediscount.ID(id.ID)).
		Where(billinginvoicediscount.Namespace(id.Namespace)).
		Only(ctx)
	if err != nil {
		return billing.InvoiceDiscount{}, fmt.Errorf("failed to get invoice discount: %w", err)
	}

	return a.mapInvoiceDiscountFromDB(dbDisc)
}

func (a *adapter) expandDiscounts(q *db.BillingInvoiceQuery) *db.BillingInvoiceQuery {
	return q.WithInvoiceDiscounts(func(q *db.BillingInvoiceDiscountQuery) {
		q.Where(billinginvoicediscount.DeletedAtIsNil())
	})
}

func (a *adapter) mapInvoiceDiscountsFromDB(dbDisc []*db.BillingInvoiceDiscount) (billing.InvoiceDiscounts, error) {
	discounts, err := slicesx.MapWithErr(dbDisc, a.mapInvoiceDiscountFromDB)
	if err != nil {
		return billing.InvoiceDiscounts{}, err
	}

	return billing.NewInvoiceDiscounts(discounts), nil
}

func (a *adapter) mapInvoiceDiscountFromDB(dbDisc *db.BillingInvoiceDiscount) (billing.InvoiceDiscount, error) {
	switch dbDisc.Type {
	case billing.PercentageDiscountType:
		return mapPercentageInvoiceDisountFromDB(dbDisc)
	default:
		return billing.InvoiceDiscount{}, fmt.Errorf("invalid invoice discount type: %s [id=%s]", dbDisc.Type, dbDisc.ID)
	}
}

func mapPercentageInvoiceDisountFromDB(dbDisc *db.BillingInvoiceDiscount) (billing.InvoiceDiscount, error) {
	return billing.NewInvoiceDiscountFrom(billing.InvoiceDiscountPercentage{
		InvoiceDiscountBase: billing.InvoiceDiscountBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				ID:          dbDisc.ID,
				Namespace:   dbDisc.Namespace,
				CreatedAt:   dbDisc.CreatedAt,
				UpdatedAt:   dbDisc.UpdatedAt,
				DeletedAt:   dbDisc.DeletedAt,
				Name:        dbDisc.Name,
				Description: dbDisc.Description,
			}),

			InvoiceID: dbDisc.InvoiceID,
			Type:      dbDisc.Type,
			LineIDs:   dbDisc.LineIds,
		},
		Percentage: dbDisc.Amount,
	}), nil
}
