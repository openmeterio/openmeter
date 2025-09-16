package pagination

import (
	"context"

	"github.com/samber/lo"
)

const MAX_SAFE_ITER = 10_000

// CollectAll collects all items from the paginator with cursoring.
func CollectAll[T any](ctx context.Context, paginator Paginator[T], cursor *Cursor) ([]T, error) {
	var all []T

	// Let's make a local copy of the cursor
	var c *Cursor

	if cursor != nil {
		c = lo.ToPtr(lo.FromPtr(cursor))
	}

	for i := 0; i < MAX_SAFE_ITER; i++ {
		res, err := paginator.Paginate(ctx, c)
		if err != nil {
			return all, err
		}

		all = append(all, res.Items...)

		if res.NextCursor == nil {
			break
		}

		c = res.NextCursor
	}

	return all, nil
}
