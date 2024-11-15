package billingadapter

import (
	"errors"
	"fmt"

	"github.com/samber/lo"

	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
)

type operation string

const (
	operationCreate operation = "create"
	operationUpdate operation = "update"
	operationDelete operation = "delete"
)

type diff[T any] struct {
	ToDelete []T
	ToUpdate []T
	ToCreate []T
}

func (d *diff[T]) NeedsUpdate(item ...T) {
	d.ToUpdate = append(d.ToUpdate, item...)
}

func (d *diff[T]) NeedsCreate(item ...T) {
	d.ToCreate = append(d.ToCreate, item...)
}

func (d *diff[T]) NeedsDelete(item ...T) {
	d.ToDelete = append(d.ToDelete, item...)
}

func unionOfDiffs[T any](a, b diff[T]) diff[T] {
	out := diff[T]{
		ToDelete: make([]T, 0, len(a.ToDelete)+len(b.ToDelete)),
		ToUpdate: make([]T, 0, len(a.ToUpdate)+len(b.ToUpdate)),
		ToCreate: make([]T, 0, len(a.ToCreate)+len(b.ToCreate)),
	}

	out.ToDelete = append(out.ToDelete, a.ToDelete...)
	out.ToDelete = append(out.ToDelete, b.ToDelete...)

	out.ToUpdate = append(out.ToUpdate, a.ToUpdate...)
	out.ToUpdate = append(out.ToUpdate, b.ToUpdate...)

	out.ToCreate = append(out.ToCreate, a.ToCreate...)
	out.ToCreate = append(out.ToCreate, b.ToCreate...)

	return out
}

type discountWithLine struct {
	Discount billingentity.LineDiscount // Note: no pointer here, as we are referencing the object at the end and there are no dependencies
	Line     *billingentity.Line
}

type invoiceLineDiff struct {
	LineBase   diff[*billingentity.Line]
	FlatFee    diff[*billingentity.Line]
	UsageBased diff[*billingentity.Line]

	// Dependant entities
	Discounts diff[discountWithLine]

	// ChildrenDiff contains the diff for the children of the line, we need to make this two-staged
	// as first we need to make sure that the parent line IDs of the children are correct, and then
	// we can update the children themselves.
	ChildrenDiff *invoiceLineDiff
}

func (d *invoiceLineDiff) NeedsCreate(item ...*billingentity.Line) {
	d.LineBase.NeedsCreate(item...)
	switch item[0].Type {
	case billingentity.InvoiceLineTypeFee:
		d.FlatFee.NeedsCreate(item...)
	case billingentity.InvoiceLineTypeUsageBased:
		d.UsageBased.NeedsCreate(item...)
	}
}

func diffInvoiceLines(lines []*billingentity.Line) (*invoiceLineDiff, error) {
	var outErr error
	diff := invoiceLineDiff{
		ChildrenDiff: &invoiceLineDiff{},
	}

	workItems := lo.Map(lines, func(l *billingentity.Line, _ int) *billingentity.Line {
		return l
	})

	remaining := make([]*billingentity.Line, 0, len(lines))

	// childUpdates contain the list of lines for which we would need to validate if we
	// need to update the children
	childUpdates := make([]*billingentity.Line, 0, len(lines))

	// Let's try to match items by ID first and figure out what we need to be updated
	for _, workItem := range workItems {
		if workItem.DBState != nil {
			if err := diffLineBaseEntities(workItem, &diff); err != nil {
				outErr = errors.Join(outErr, err)
				continue
			}

			childUpdates = append(childUpdates, workItem)
			continue
		}

		remaining = append(remaining, workItem)
	}

	// Items without a DBState are new items => let's create them
	for _, workItem := range remaining {
		diff.LineBase.NeedsCreate(workItem)
		switch workItem.Type {
		case billingentity.InvoiceLineTypeFee:
			diff.FlatFee.NeedsCreate(workItem)
		case billingentity.InvoiceLineTypeUsageBased:
			diff.UsageBased.NeedsCreate(workItem)
		}

		if err := handleLineDependantEntities(workItem, operationCreate, &diff); err != nil {
			outErr = errors.Join(outErr, err)
		}

		// Any child of a new item is also new => let's create them
		for _, child := range workItem.Children.Get() {
			diff.ChildrenDiff.LineBase.NeedsCreate(child)
			switch child.Type {
			case billingentity.InvoiceLineTypeFee:
				diff.ChildrenDiff.FlatFee.NeedsCreate(child)
			case billingentity.InvoiceLineTypeUsageBased:
				diff.ChildrenDiff.UsageBased.NeedsCreate(child)
			}

			if err := handleLineDependantEntities(child, operationCreate, diff.ChildrenDiff); err != nil {
				outErr = errors.Join(outErr, fmt.Errorf("handling children entries: %w", err))
			}
		}
	}

	// Let's figure out what we need to do about child lines
	for _, childUpdate := range childUpdates {
		// If the children are not present, we don't need to do anything (a.k.a. do not touch)
		if !childUpdate.Children.IsPresent() {
			continue
		}

		if err := getChildrenActions(
			childUpdate.DBState.Children.Get(),
			childUpdate.Children.Get(),
			diff.ChildrenDiff,
		); err != nil {
			outErr = errors.Join(outErr, err)
		}
	}

	if outErr != nil {
		return nil, outErr
	}

	return &diff, nil
}

func diffLineBaseEntities(line *billingentity.Line, out *invoiceLineDiff) error {
	if line.DBState.ID == "" {
		// This should not happen, as we fill the DBState after the DB fetch, it's more
		// like a safeguard against future changes/manual DB state manipulation
		return errors.New("line: db ID cannot be empty")
	}

	if line.ID != line.DBState.ID {
		return fmt.Errorf("line[%s]: id change is not allowed", line.ID)
	}

	baseNeedsUpdate := false
	if !line.DBState.LineBase.Equal(line.LineBase) {
		baseNeedsUpdate = true
	}

	switch line.Type {
	case billingentity.InvoiceLineTypeFee:
		// TODO: any dependant object (such as discounts should also update the line's UpdatedAt)
		if !line.DBState.FlatFee.Equal(line.FlatFee) {
			// Due to UpdatedAt + QTY
			baseNeedsUpdate = true

			out.FlatFee.NeedsUpdate(line)
		}
	case billingentity.InvoiceLineTypeUsageBased:
		if !line.DBState.UsageBased.Equal(line.UsageBased) {
			baseNeedsUpdate = true

			out.UsageBased.NeedsUpdate(line)
		}
	}

	if baseNeedsUpdate {
		out.LineBase.NeedsUpdate(line)
	}

	return handleLineDependantEntities(line, operationUpdate, out)
}

func getChildrenActions(dbSave []*billingentity.Line, current []*billingentity.Line, out *invoiceLineDiff) error {
	currentByID := lo.GroupBy(
		lo.Filter(current, func(l *billingentity.Line, _ int) bool {
			return l.ID != ""
		}),
		func(l *billingentity.Line) string {
			return l.ID
		})

	for _, dbLine := range dbSave {
		if _, ok := currentByID[dbLine.ID]; !ok {
			// We don't have this line in the current list, so we need to delete it
			// from the DB
			out.LineBase.NeedsDelete(dbLine)
			if err := handleLineDependantEntities(dbLine, operationDelete, out); err != nil {
				return err
			}
		}
	}

	dbSaveByID := lo.GroupBy(dbSave, func(l *billingentity.Line) string {
		return l.ID
	})

	for _, currentLine := range current {
		if currentLine.ID == "" {
			// We don't have an ID for this line, so we need to create it
			out.NeedsCreate(currentLine)
			if err := handleLineDependantEntities(currentLine, operationCreate, out); err != nil {
				return err
			}
			continue
		}

		dbLine, ok := dbSaveByID[currentLine.ID]
		if !ok {
			// Maybe we have a fake ID, let's throw an error
			return fmt.Errorf("line[%s]: not found in DB", currentLine.ID)
		}

		currentLine.DBState = dbLine[0]

		if err := diffLineBaseEntities(currentLine, out); err != nil {
			return err
		}
	}

	return nil
}

func handleLineDependantEntities(line *billingentity.Line, lineOperation operation, out *invoiceLineDiff) error {
	return handleLineDiscounts(line, lineOperation, out)
}

func handleLineDiscounts(line *billingentity.Line, lineOperation operation, out *invoiceLineDiff) error {
	switch lineOperation {
	case operationCreate:
		for _, discount := range line.Discounts.Get() {
			out.Discounts.NeedsCreate(discountWithLine{
				Discount: discount,
				Line:     line,
			})
		}
	case operationDelete:
		for _, discount := range line.Discounts.Get() {
			out.Discounts.NeedsDelete(discountWithLine{
				Discount: discount,
				Line:     line,
			})
		}
	case operationUpdate:
		return handleLineDiscountUpdate(line, out)
	}

	return nil
}

func handleLineDiscountUpdate(line *billingentity.Line, out *invoiceLineDiff) error {
	// We need to figure out what we need to update
	currentDiscountIDs := lo.GroupBy(
		lo.Filter(
			line.Discounts.Get(),
			func(d billingentity.LineDiscount, _ int) bool {
				return d.ID != ""
			},
		),
		func(d billingentity.LineDiscount) string {
			return d.ID
		},
	)

	dbDiscountIDs := lo.GroupBy(line.DBState.Discounts.Get(), func(d billingentity.LineDiscount) string {
		return d.ID
	})

	for _, dbDiscount := range line.DBState.Discounts.Get() {
		if _, ok := currentDiscountIDs[dbDiscount.ID]; !ok {
			// We need to delete this discount
			out.Discounts.NeedsDelete(discountWithLine{
				Discount: dbDiscount,
				Line:     line,
			})
		}
	}

	for _, currentDiscount := range line.Discounts.Get() {
		if currentDiscount.ID == "" {
			// We need to create this discount
			out.Discounts.NeedsCreate(discountWithLine{
				Discount: currentDiscount,
				Line:     line,
			})
			continue
		}

		dbDiscount, ok := dbDiscountIDs[currentDiscount.ID]
		if !ok {
			return fmt.Errorf("discount[%s]: not found in DB", currentDiscount.ID)
		}

		dbItem := dbDiscount[0]
		currentDiscount.ID = dbItem.ID
		currentDiscount.CreatedAt = dbItem.CreatedAt
		currentDiscount.UpdatedAt = dbItem.UpdatedAt

		if !dbItem.Equal(currentDiscount) {
			out.Discounts.NeedsUpdate(discountWithLine{
				Discount: currentDiscount,
				Line:     line,
			})
		}
	}

	return nil
}
