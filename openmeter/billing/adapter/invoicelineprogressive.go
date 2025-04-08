package billingadapter

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceline"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

// expandProgressiveLineHierarchy expands the given lines with their progressive line hierarchy
// This is done by fetching all the lines that are children of the given lines parent lines and then building
// the hierarchy.
func (a *adapter) expandProgressiveLineHierarchy(ctx context.Context, namespace string, lines []*billing.Line) ([]*billing.Line, error) {
	// Let's collect all the lines with a parent line id set

	lineIDsToParentIDs := map[string]string{}

	for _, line := range lines {
		if line.ParentLineID != nil {
			lineIDsToParentIDs[line.ID] = *line.ParentLineID
		}
	}

	if len(lineIDsToParentIDs) == 0 {
		return lines, nil
	}

	inScopeLines, err := a.fetchAllLinesForParentIDs(ctx, namespace, lo.Values(lineIDsToParentIDs))
	if err != nil {
		return nil, err
	}

	// let's build the hierarchy objects
	hierarchyByParentID, err := a.buildProgressiveLineHierarchy(inScopeLines)
	if err != nil {
		return nil, err
	}

	// Let's validate the hierarchy
	for parentID, hierarchy := range hierarchyByParentID {
		if hierarchy.Root.Line == nil {
			return nil, fmt.Errorf("root line for parent line[%s] not found", parentID)
		}

		for _, child := range hierarchy.Children {
			if child.Line == nil {
				return nil, fmt.Errorf("child line for parent line[%s] not found", parentID)
			}

			// This is the only valid state for a child line
			if child.Line.Status != billing.InvoiceLineStatusValid {
				return nil, fmt.Errorf("child line for parent line[%s] is not valid", parentID)
			}
		}
	}

	// let's assign the hierarchy to the already fetched lines
	return slicesx.MapWithErr(lines, func(line *billing.Line) (*billing.Line, error) {
		if line.ParentLineID == nil {
			return line, nil
		}

		hierarchy, ok := hierarchyByParentID[*line.ParentLineID]
		if !ok {
			return nil, fmt.Errorf("parent line for line[%s] not found", line.ID)
		}

		line.ProgressiveLineHierarchy = &hierarchy

		return line, nil
	})
}

func (a *adapter) fetchAllLinesForParentIDs(ctx context.Context, namespace string, parentIDs []string) ([]billing.InvoiceLineWithInvoiceBase, error) {
	query := a.db.BillingInvoiceLine.Query().
		Where(
			billinginvoiceline.Or(
				billinginvoiceline.IDIn(parentIDs...),
				billinginvoiceline.ParentLineIDIn(parentIDs...),
			),
			billinginvoiceline.Namespace(namespace),
		).
		WithFlatFeeLine().
		WithUsageBasedLine().
		WithLineDiscounts().
		WithBillingInvoice() // TODO[later]: we can consider loading this in a separate query, might be more efficient

	dbLines, err := query.All(ctx)
	if err != nil {
		return nil, err
	}

	mappedLines, err := slicesx.MapWithErr(dbLines, func(dbLine *db.BillingInvoiceLine) (billing.InvoiceLineWithInvoiceBase, error) {
		empty := billing.InvoiceLineWithInvoiceBase{}

		line, err := a.mapInvoiceLineWithoutReferences(dbLine)
		if err != nil {
			return empty, err
		}

		return billing.InvoiceLineWithInvoiceBase{
			Line:    &line,
			Invoice: a.mapInvoiceBaseFromDB(ctx, dbLine.Edges.BillingInvoice),
		}, nil
	})
	if err != nil {
		return nil, err
	}

	return mappedLines, nil
}

func (a *adapter) buildProgressiveLineHierarchy(inScopeLines []billing.InvoiceLineWithInvoiceBase) (map[string]billing.InvoiceLineProgressiveHierarchy, error) {
	hierarchyByParentID := map[string]billing.InvoiceLineProgressiveHierarchy{}

	for _, line := range inScopeLines {
		if line.Line.ParentLineID == nil {
			// We have encountered a parent line

			hierarchy, ok := hierarchyByParentID[line.Line.ID]
			if ok {
				if hierarchy.Root.Line != nil {
					return nil, fmt.Errorf("parent line[%s] already exists", line.Line.ID)
				}
			}

			hierarchy.Root = line
			hierarchyByParentID[line.Line.ID] = hierarchy
			continue
		}

		// We have encountered a child line
		parentID := *line.Line.ParentLineID

		hierarchy := hierarchyByParentID[parentID]
		hierarchy.Children = append(hierarchy.Children, line)
		hierarchyByParentID[parentID] = hierarchy
	}

	return hierarchyByParentID, nil
}
