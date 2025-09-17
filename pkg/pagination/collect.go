package pagination

import (
	"context"
	"fmt"
)

const MAX_SAFE_ITER = 10_000

// CollectAll collects all items from the paginator going page by page.
func CollectAll[T any](ctx context.Context, paginator Paginator[T], pageSize int) ([]T, error) {
	var all []T

	page := Page{
		PageSize:   pageSize,
		PageNumber: 1,
	}

	for {
		res, err := paginator.Paginate(ctx, page)
		if err != nil {
			return nil, err
		}

		all = append(all, res.Items...)

		if len(res.Items) < pageSize {
			break
		}

		page.PageNumber += 1

		if page.PageNumber > MAX_SAFE_ITER {
			return nil, fmt.Errorf("max safe iter reached: %d", page.PageNumber)
		}
	}

	return all, nil
}
