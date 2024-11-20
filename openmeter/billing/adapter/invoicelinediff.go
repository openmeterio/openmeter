package billingadapter

import (
	"errors"
	"fmt"

	"github.com/samber/lo"

	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	"github.com/openmeterio/openmeter/pkg/set"
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

	// AffectedLineIDs contains the list of line IDs that are affected by the diff, even if they
	// are not updated. We can use this to update the UpdatedAt of the lines if any of the dependant
	// entities are updated.
	AffectedLineIDs set.Set[string]

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
		AffectedLineIDs: set.New[string](),
		ChildrenDiff: &invoiceLineDiff{
			AffectedLineIDs: set.New[string](),
		},
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

	diff.AffectedLineIDs = set.Subtract(
		set.Union(diff.AffectedLineIDs, diff.ChildrenDiff.AffectedLineIDs),
		set.New(lo.Map(diff.LineBase.ToUpdate, func(l *billingentity.Line, _ int) string {
			return l.ID
		})...),
		set.New(lo.Map(diff.ChildrenDiff.LineBase.ToUpdate, func(l *billingentity.Line, _ int) string {
			return l.ID
		})...),
	)

	// Let's make sure we are not leaking any in-progress calculation details
	diff.ChildrenDiff.AffectedLineIDs = nil
	diff.ChildrenDiff.ChildrenDiff = nil

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
		if !line.DBState.FlatFee.Equal(line.FlatFee) {
			// Due to quantity being stored in the base entity, we need to update the base entity
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

		out.AffectedLineIDs.Add(lineParentIDs(line, lineIDExcludeSelf)...)
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

			// Deleting a child is considered an update for the parent
			out.AffectedLineIDs.Add(lineParentIDs(dbLine, lineIDExcludeSelf)...)
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

			// Adding a child is considered an update for the parent even if the db line of parent has not changed
			out.AffectedLineIDs.Add(lineParentIDs(currentLine, lineIDExcludeSelf)...)
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

			out.AffectedLineIDs.Add(lineParentIDs(line, lineIDIncludeSelf)...)
		}
	}

	for _, currentDiscount := range line.Discounts.Get() {
		if currentDiscount.ID == "" {
			// We need to create this discount
			out.Discounts.NeedsCreate(discountWithLine{
				Discount: currentDiscount,
				Line:     line,
			})

			out.AffectedLineIDs.Add(lineParentIDs(line, lineIDIncludeSelf)...)

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

			out.AffectedLineIDs.Add(lineParentIDs(line, lineIDIncludeSelf)...)
		}
	}

	return nil
}

type lineIDIncludeFlag int

const (
	lineIDIncludeSelf lineIDIncludeFlag = iota
	lineIDExcludeSelf
)

func lineParentIDs(line *billingentity.Line, includeLineID lineIDIncludeFlag) []string {
	out := make([]string, 0, 3)

	if line == nil {
		return out
	}

	if includeLineID == lineIDIncludeSelf {
		out = append(out, line.ID)
	}

	currentLine := line
	for currentLine != nil {
		if currentLine.ParentLineID == nil {
			break
		}

		out = append(out, *currentLine.ParentLineID)

		currentLine = currentLine.ParentLine
	}

	return out
}
