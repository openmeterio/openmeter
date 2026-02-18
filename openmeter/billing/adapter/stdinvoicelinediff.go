package billingadapter

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/entitydiff"
	"github.com/openmeterio/openmeter/pkg/set"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type (
	usageLineDiscountManagedWithLine  = entitydiff.EqualerNestedEntity[billing.UsageLineDiscountManaged, *billing.StandardLine]
	amountLineDiscountManagedWithLine = entitydiff.EqualerNestedEntity[billing.AmountLineDiscountManaged, *billing.StandardLine]

	detailedLineWithParent               = entitydiff.NestedEntity[*billing.DetailedLine, *billing.StandardLine]
	detailedLineDiff                     = entitydiff.Diff[detailedLineWithParent]
	detailedLineAmountDiscountWithParent = entitydiff.EqualerNestedEntity[billing.AmountLineDiscountManaged, *billing.DetailedLine]
	detailedLineAmountDiscountDiff       = entitydiff.Diff[detailedLineAmountDiscountWithParent]
)

type invoiceLineDiff struct {
	Line entitydiff.Diff[*billing.StandardLine]

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
	DetailedLineAmountDiscounts detailedLineAmountDiscountDiff
	DetailedLineAffectedLineIDs *set.Set[string]
}

func diffInvoiceLines(lines []*billing.StandardLine) (invoiceLineDiff, error) {
	diff := invoiceLineDiff{
		AffectedLineIDs:             set.New[string](),
		DetailedLineAffectedLineIDs: set.New[string](),
	}

	// For now we are handling the dbState on a per line basis so that we don't have to make operations
	// against the invoice itself. Going forward we can consider moving this to the invoice level, as this
	// only makes sense for gathering invoices.
	dbState := []*billing.StandardLine{}
	for _, line := range lines {
		if line.DBState != nil {
			dbState = append(dbState, line.DBState)
		}
	}

	// Handle top level line diffs
	err := entitydiff.DiffByID(entitydiff.DiffByIDInput[*billing.StandardLine]{
		DBState:       dbState,
		ExpectedState: lines,
		HandleDelete:  diff.DeleteLine,
		HandleCreate:  diff.CreateLine,
		HandleUpdate: func(item entitydiff.DiffUpdate[*billing.StandardLine]) error {
			if !item.ExpectedState.StandardLineBase.Equal(item.PersistedState.StandardLineBase) {
				diff.Line.NeedsUpdate(item)
			}

			// Dependant entities

			diff.UsageDiscounts = diff.UsageDiscounts.Append(entitydiff.DiffByIDEqualer(
				entitydiff.NewEqualersWithParent(item.ExpectedState.Discounts.Usage, item.ExpectedState),
				entitydiff.NewEqualersWithParent(item.PersistedState.Discounts.Usage, item.PersistedState),
			))

			diff.AmountDiscounts = diff.AmountDiscounts.Append(entitydiff.DiffByIDEqualer(
				entitydiff.NewEqualersWithParent(item.ExpectedState.Discounts.Amount, item.ExpectedState),
				entitydiff.NewEqualersWithParent(item.PersistedState.Discounts.Amount, item.PersistedState),
			))

			// Detailed line diffs
			err := entitydiff.DiffByID(entitydiff.DiffByIDInput[*billing.DetailedLine]{
				DBState:       slicesx.SliceToPtrSlice(item.PersistedState.DetailedLines),
				ExpectedState: slicesx.SliceToPtrSlice(item.ExpectedState.DetailedLines),
				HandleDelete: func(detailedLine *billing.DetailedLine) error {
					if !item.PersistedState.IsDeleted() {
						diff.AffectedLineIDs.Add(item.PersistedState.GetID())
					}

					return diff.DeleteDetailedLine(detailedLine, item.PersistedState)
				},
				HandleCreate: func(detailedLine *billing.DetailedLine) error {
					return diff.CreateDetailedLine(detailedLine, item.ExpectedState)
				},
				HandleUpdate: func(detailedLine entitydiff.DiffUpdate[*billing.DetailedLine]) error {
					if detailedLine.ExpectedState == nil {
						return fmt.Errorf("detailed line expected state is nil or flat fee is nil")
					}

					if detailedLine.PersistedState == nil {
						return fmt.Errorf("detailed line db state is nil or flat fee is nil")
					}

					if !detailedLine.ExpectedState.DetailedLineBase.Equal(detailedLine.PersistedState.DetailedLineBase) {
						diff.DetailedLine.NeedsUpdate(entitydiff.DiffUpdate[detailedLineWithParent]{
							PersistedState: detailedLineWithParent{
								Entity: detailedLine.PersistedState,
								Parent: item.PersistedState,
							},
							ExpectedState: detailedLineWithParent{
								Entity: detailedLine.ExpectedState,
								Parent: item.ExpectedState,
							},
						})

						if !item.ExpectedState.IsDeleted() {
							diff.AffectedLineIDs.Add(item.PersistedState.ID)
						}
					}

					discountChanges := entitydiff.DiffByIDEqualer(
						entitydiff.NewEqualersWithParent(detailedLine.ExpectedState.AmountDiscounts, detailedLine.ExpectedState),
						entitydiff.NewEqualersWithParent(detailedLine.PersistedState.AmountDiscounts, detailedLine.PersistedState),
					)

					diff.DetailedLineAmountDiscounts = diff.DetailedLineAmountDiscounts.Append(discountChanges)

					if !discountChanges.IsEmpty() {
						if !item.ExpectedState.IsDeleted() {
							diff.AffectedLineIDs.Add(item.PersistedState.ID)
						}

						if !detailedLine.ExpectedState.IsDeleted() {
							diff.DetailedLineAffectedLineIDs.Add(detailedLine.PersistedState.ID)
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

func (d *invoiceLineDiff) DeleteLine(item *billing.StandardLine) error {
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

	for idx := range item.DetailedLines {
		if err := d.DeleteDetailedLine(&item.DetailedLines[idx], item); err != nil {
			return err
		}
	}

	return nil
}

func (d *invoiceLineDiff) CreateLine(item *billing.StandardLine) error {
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

	for idx := range item.DetailedLines {
		child := &item.DetailedLines[idx]
		d.DetailedLine.NeedsCreate(detailedLineWithParent{
			Entity: child,
			Parent: item,
		})

		for _, discount := range child.AmountDiscounts {
			d.DetailedLineAmountDiscounts.NeedsCreate(detailedLineAmountDiscountWithParent{
				Entity: discount,
				Parent: child,
			})
		}
	}

	return nil
}

func (d *invoiceLineDiff) DeleteDetailedLine(item *billing.DetailedLine, parent *billing.StandardLine) error {
	d.DetailedLine.NeedsDelete(detailedLineWithParent{
		Entity: item,
		Parent: parent,
	})

	for _, discount := range item.AmountDiscounts {
		d.DetailedLineAmountDiscounts.NeedsDelete(detailedLineAmountDiscountWithParent{
			Entity: discount,
			Parent: item,
		})
	}

	return nil
}

func (d *invoiceLineDiff) CreateDetailedLine(item *billing.DetailedLine, parent *billing.StandardLine) error {
	d.DetailedLine.NeedsCreate(detailedLineWithParent{
		Entity: item,
		Parent: parent,
	})

	for _, discount := range item.AmountDiscounts {
		d.DetailedLineAmountDiscounts.NeedsCreate(detailedLineAmountDiscountWithParent{
			Entity: discount,
			Parent: item,
		})
	}

	return nil
}
