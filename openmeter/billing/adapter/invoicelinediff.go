package billingadapter

import (
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/entitydiff"
	"github.com/openmeterio/openmeter/pkg/set"
)

type (
	usageLineDiscountManagedWithLine  = entitydiff.EqualerNestedEntity[billing.UsageLineDiscountManaged, *billing.Line]
	amountLineDiscountManagedWithLine = entitydiff.EqualerNestedEntity[billing.AmountLineDiscountManaged, *billing.Line]

	detailedLineWithParent               = entitydiff.NestedEntity[*billing.DetailedLine, *billing.Line]
	detailedLineAmountDiscountWithParent = entitydiff.EqualerNestedEntity[billing.AmountLineDiscountManaged, *billing.Line]
	detailedLineDiff                     = entitydiff.Diff[detailedLineWithParent]
)

type invoiceLineDiff struct {
	Line entitydiff.Diff[*billing.Line]

	// Dependant entities
	UsageDiscounts  entitydiff.Diff[usageLineDiscountManagedWithLine]
	AmountDiscounts entitydiff.Diff[amountLineDiscountManagedWithLine]

	// AffectedLineIDs contains the list of line IDs that are affected by the diff, even if they
	// are not updated. We can use this to update the UpdatedAt of the lines if any of the dependant
	// entities are updated.
	AffectedLineIDs *set.Set[string]

	// ChildrenDiff contains the diff for the children of the line, we need to make this two-staged
	// as first we need to make sure that the parent line IDs of the children are correct, and then
	// we can update the children themselves.

	DetailedLine                detailedLineDiff
	DetailedLineAmountDiscounts entitydiff.Diff[detailedLineAmountDiscountWithParent]
	DetailedLineAffectedLineIDs *set.Set[string]
}

func diffInvoiceLines(lines []*billing.Line) (invoiceLineDiff, error) {
	diff := invoiceLineDiff{
		AffectedLineIDs:             set.New[string](),
		DetailedLineAffectedLineIDs: set.New[string](),
	}

	// For now we are handling the dbState on a per line basis so that we don't have to make operations
	// against the invoice itself. Going forward we can consider moving this to the invoice level, as this
	// only makes sense for gathering invoices.
	dbState := []*billing.Line{}
	for _, line := range lines {
		if line.DBState != nil {
			dbState = append(dbState, line.DBState)
		}
	}

	// Handle top level line diffs
	err := entitydiff.DiffByID(entitydiff.DiffByIDInput[*billing.Line]{
		DBState:       dbState,
		ExpectedState: lines,
		HandleDelete:  diff.DeleteLine,
		HandleCreate:  diff.CreateLine,
		HandleUpdate: func(item entitydiff.DiffUpdate[*billing.Line]) error {
			if item.ExpectedState.UsageBased == nil {
				return fmt.Errorf("expected state usage based is nil")
			}

			if item.DBState.UsageBased == nil {
				return fmt.Errorf("db state usage based is nil")
			}

			if !item.ExpectedState.LineBase.Equal(item.DBState.LineBase) || !item.ExpectedState.UsageBased.Equal(item.DBState.UsageBased) {
				diff.Line.NeedsUpdate(item)
			}

			// Dependant entities

			diff.UsageDiscounts = diff.UsageDiscounts.Append(entitydiff.DiffByIDEqualer(
				entitydiff.NewEqualersWithParent(item.ExpectedState.Discounts.Usage, item.ExpectedState),
				entitydiff.NewEqualersWithParent(item.DBState.Discounts.Usage, item.DBState),
			))

			diff.AmountDiscounts = diff.AmountDiscounts.Append(entitydiff.DiffByIDEqualer(
				entitydiff.NewEqualersWithParent(item.ExpectedState.Discounts.Amount, item.ExpectedState),
				entitydiff.NewEqualersWithParent(item.DBState.Discounts.Amount, item.DBState),
			))

			// Detailed line diffs
			err := entitydiff.DiffByID(entitydiff.DiffByIDInput[*billing.Line]{
				DBState:       item.DBState.Children,
				ExpectedState: item.ExpectedState.Children,
				HandleDelete: func(detailedLine *billing.Line) error {
					if !item.DBState.IsDeleted() {
						diff.AffectedLineIDs.Add(item.DBState.GetID())
					}

					return diff.DeleteDetailedLine(detailedLine, item.DBState)
				},
				HandleCreate: func(detailedLine *billing.Line) error {
					return diff.CreateDetailedLine(detailedLine, item.ExpectedState)
				},
				HandleUpdate: func(detailedLine entitydiff.DiffUpdate[*billing.Line]) error {
					if detailedLine.ExpectedState == nil || detailedLine.ExpectedState.FlatFee == nil {
						return fmt.Errorf("detailed line expected state is nil or flat fee is nil")
					}

					if detailedLine.DBState == nil || detailedLine.DBState.FlatFee == nil {
						return fmt.Errorf("detailed line db state is nil or flat fee is nil")
					}

					if !detailedLine.ExpectedState.LineBase.Equal(detailedLine.DBState.LineBase) || !detailedLine.ExpectedState.FlatFee.Equal(detailedLine.DBState.FlatFee) {
						diff.DetailedLine.NeedsUpdate(entitydiff.DiffUpdate[detailedLineWithParent]{
							DBState: detailedLineWithParent{
								Entity: detailedLine.DBState,
								Parent: item.DBState,
							},
							ExpectedState: detailedLineWithParent{
								Entity: detailedLine.ExpectedState,
								Parent: item.DBState,
							},
						})

						if !item.ExpectedState.IsDeleted() {
							diff.AffectedLineIDs.Add(item.DBState.ID)
						}
					}

					discountChanges := entitydiff.DiffByIDEqualer(
						entitydiff.NewEqualersWithParent(detailedLine.ExpectedState.Discounts.Amount, detailedLine.ExpectedState),
						entitydiff.NewEqualersWithParent(detailedLine.DBState.Discounts.Amount, detailedLine.DBState),
					)

					diff.DetailedLineAmountDiscounts = diff.DetailedLineAmountDiscounts.Append(discountChanges)

					if !discountChanges.IsEmpty() {
						if !item.ExpectedState.IsDeleted() {
							diff.AffectedLineIDs.Add(item.DBState.ID)
						}

						if !detailedLine.ExpectedState.IsDeleted() {
							diff.DetailedLineAffectedLineIDs.Add(detailedLine.DBState.ID)
						}
					}

					return nil
				},
			})
			if err != nil {
				return err
			}

			return nil
		},
	})
	if err != nil {
		return diff, err
	}

	return diff, nil
}

func (d *invoiceLineDiff) DeleteLine(item *billing.Line) error {
	d.Line.NeedsDelete(item)

	for _, discount := range item.Discounts.Usage {
		d.UsageDiscounts.NeedsDelete(usageLineDiscountManagedWithLine{
			Entity: discount,
			Parent: item,
		})
	}
	for _, discount := range item.Discounts.Amount {
		d.AmountDiscounts.NeedsDelete(amountLineDiscountManagedWithLine{
			Entity: discount,
			Parent: item,
		})
	}

	for _, detailedLine := range item.Children {
		if err := d.DeleteDetailedLine(detailedLine, item); err != nil {
			return err
		}
	}

	return nil
}

func (d *invoiceLineDiff) CreateLine(item *billing.Line) error {
	d.Line.NeedsCreate(item)

	for _, usageDiscount := range item.Discounts.Usage {
		d.UsageDiscounts.NeedsCreate(usageLineDiscountManagedWithLine{
			Entity: usageDiscount,
			Parent: item,
		})
	}
	for _, amountDiscount := range item.Discounts.Amount {
		d.AmountDiscounts.NeedsCreate(amountLineDiscountManagedWithLine{
			Entity: amountDiscount,
			Parent: item,
		})
	}

	for _, child := range item.Children {
		d.DetailedLine.NeedsCreate(detailedLineWithParent{
			Entity: child,
			Parent: item,
		})

		for _, discount := range child.Discounts.Amount {
			d.DetailedLineAmountDiscounts.NeedsCreate(detailedLineAmountDiscountWithParent{
				Entity: discount,
				Parent: child,
			})
		}
	}

	return nil
}

func (d *invoiceLineDiff) DeleteDetailedLine(item *billing.Line, parent *billing.Line) error {
	d.DetailedLine.NeedsDelete(detailedLineWithParent{
		Entity: item,
		Parent: parent,
	})

	for _, discount := range item.Discounts.Amount {
		d.DetailedLineAmountDiscounts.NeedsDelete(detailedLineAmountDiscountWithParent{
			Entity: discount,
			Parent: item,
		})
	}

	return nil
}

func (d *invoiceLineDiff) CreateDetailedLine(item *billing.Line, parent *billing.Line) error {
	d.DetailedLine.NeedsCreate(detailedLineWithParent{
		Entity: item,
		Parent: parent,
	})

	for _, discount := range item.Discounts.Amount {
		d.DetailedLineAmountDiscounts.NeedsCreate(detailedLineAmountDiscountWithParent{
			Entity: discount,
			Parent: item,
		})
	}

	return nil
}

func (d *invoiceLineDiff) GetDetailedLineDiffWithParentID() entitydiff.Diff[*billing.DetailedLine] {
	return entitydiff.Diff[*billing.DetailedLine]{
		Create: lo.Map(d.DetailedLine.Create, func(item detailedLineWithParent, _ int) *billing.DetailedLine {
			item.Entity.ParentLineID = lo.ToPtr(item.Parent.GetID())

			return item.Entity
		}),
		Delete: lo.Map(d.DetailedLine.Delete, func(item detailedLineWithParent, _ int) *billing.DetailedLine {
			item.Entity.ParentLineID = lo.ToPtr(item.Parent.GetID())

			return item.Entity
		}),
		Update: lo.Map(d.DetailedLine.Update, func(item entitydiff.DiffUpdate[detailedLineWithParent], _ int) entitydiff.DiffUpdate[*billing.DetailedLine] {
			item.ExpectedState.Entity.ParentLineID = lo.ToPtr(item.ExpectedState.Parent.GetID())

			return entitydiff.DiffUpdate[*billing.DetailedLine]{
				DBState:       item.DBState.Entity,
				ExpectedState: item.ExpectedState.Entity,
			}
		}),
	}
}
