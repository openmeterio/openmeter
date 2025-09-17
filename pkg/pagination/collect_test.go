package pagination

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestCollectAll_MultiplePages(t *testing.T) {
	ctx := context.Background()

	// total items = 25, pageSize = 10 -> 3 pages (10, 10, 5)
	total := 25
	p := NewPaginator[int](func(ctx context.Context, page Page) (Result[int], error) {
		start := (page.PageNumber - 1) * page.PageSize
		end := start + page.PageSize
		if end > total {
			end = total
		}
		items := make([]int, 0, max(0, end-start))
		for i := start; i < end; i++ {
			items = append(items, i)
		}
		return Result[int]{
			Page:  page,
			Items: items,
		}, nil
	})

	items, err := CollectAll[int](ctx, p, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) != total {
		t.Fatalf("expected %d items, got %d", total, len(items))
	}

	for i := 0; i < total; i++ {
		if items[i] != i {
			t.Fatalf("expected items[%d]==%d, got %d", i, i, items[i])
		}
	}
}

func TestCollectAll_EmptyFirstPage(t *testing.T) {
	ctx := context.Background()

	p := NewPaginator[int](func(ctx context.Context, page Page) (Result[int], error) {
		// Always return empty result
		return Result[int]{
			Page:  page,
			Items: nil,
		}, nil
	})

	items, err := CollectAll[int](ctx, p, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(items))
	}
}

func TestCollectAll_ErrorMidway(t *testing.T) {
	ctx := context.Background()

	wantErr := errors.New("boom")
	p := NewPaginator[int](func(ctx context.Context, page Page) (Result[int], error) {
		if page.PageNumber == 2 {
			return Result[int]{}, wantErr
		}
		// return a full page for page 1 so we attempt page 2 next
		items := make([]int, page.PageSize)
		for i := 0; i < page.PageSize; i++ {
			items[i] = i
		}
		return Result[int]{
			Page:  page,
			Items: items,
		}, nil
	})

	items, err := CollectAll[int](ctx, p, 10)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err.Error() != wantErr.Error() {
		t.Fatalf("expected error %q, got %q", wantErr.Error(), err.Error())
	}
	if items != nil {
		t.Fatalf("expected nil items on error, got %v", items)
	}
}

func TestCollectAll_MaxSafeIter(t *testing.T) {
	ctx := context.Background()

	// Paginator always returns a full page to force iteration until MAX_SAFE_ITER is exceeded
	p := NewPaginator[int](func(ctx context.Context, page Page) (Result[int], error) {
		items := make([]int, page.PageSize)
		for i := 0; i < page.PageSize; i++ {
			items[i] = i
		}
		return Result[int]{
			Page:  page,
			Items: items,
		}, nil
	})

	items, err := CollectAll[int](ctx, p, 1) // keep memory small; will iterate > MAX_SAFE_ITER
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if want := fmt.Sprintf("max safe iter reached: %d", MAX_SAFE_ITER+1); err.Error() != want {
		t.Fatalf("expected error %q, got %q", want, err.Error())
	}
	if items != nil {
		t.Fatalf("expected nil items on error, got %v", items)
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
