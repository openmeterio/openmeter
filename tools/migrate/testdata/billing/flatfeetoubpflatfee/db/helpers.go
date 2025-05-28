package db

import (
	"context"
	"fmt"
)

type LineHierarchy struct {
	Line          GetUsageBasedLineByIDRow
	DetailedLines []GetFlatFeeLinesByParentIDRow
}

func (q *Queries) GetLineHierarchyByDetailedLineID(ctx context.Context, detailedLineID string) (LineHierarchy, error) {
	lineParent, err := q.GetParentID(ctx, detailedLineID)
	if err != nil {
		return LineHierarchy{}, err
	}

	if !lineParent.Valid {
		return LineHierarchy{}, fmt.Errorf("line parent not found")
	}

	ubpLine, err := q.GetUsageBasedLineByID(ctx, lineParent.String)
	if err != nil {
		return LineHierarchy{}, err
	}

	if ubpLine.ID != lineParent.String {
		return LineHierarchy{}, fmt.Errorf("ubp line id does not match line parent id")
	}

	detailedLines, err := q.GetFlatFeeLinesByParentID(ctx, lineParent.String)
	if err != nil {
		return LineHierarchy{}, err
	}

	return LineHierarchy{
		Line:          ubpLine,
		DetailedLines: detailedLines,
	}, nil
}
