package pagination

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"
)

func TestCollectAllV2_MultiplePages(t *testing.T) {
	ctx := context.Background()

	total := 25
	pageSize := 10

	p := NewPaginator[int](func(ctx context.Context, cursor *Cursor) (Result[int], error) {
		start := 0
		if cursor != nil {
			// use provided cursor to determine start offset
			var err error
			start, err = strconv.Atoi(cursor.ID)
			if err != nil {
				return Result[int]{}, err
			}
		}

		end := start + pageSize
		if end > total {
			end = total
		}

		items := make([]int, 0, max(0, end-start))
		for i := start; i < end; i++ {
			items = append(items, i)
		}

		var next *Cursor
		if end < total {
			next = loCursor(end)
		}

		return Result[int]{
			Items:      items,
			NextCursor: next,
		}, nil
	})

	items, err := CollectAll[int](ctx, p, nil)
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

func TestCollectAllV2_RespectsInitialCursor(t *testing.T) {
	ctx := context.Background()

	total := 25
	pageSize := 10
	startOffset := 10

	p := NewPaginator[int](func(ctx context.Context, cursor *Cursor) (Result[int], error) {
		start := 0
		if cursor != nil {
			var err error
			start, err = strconv.Atoi(cursor.ID)
			if err != nil {
				return Result[int]{}, err
			}
		}

		end := start + pageSize
		if end > total {
			end = total
		}
		items := make([]int, 0, max(0, end-start))
		for i := start; i < end; i++ {
			items = append(items, i)
		}
		var next *Cursor
		if end < total {
			next = loCursor(end)
		}
		return Result[int]{Items: items, NextCursor: next}, nil
	})

	items, err := CollectAll[int](ctx, p, loCursor(startOffset))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := total - startOffset
	if len(items) != expected {
		t.Fatalf("expected %d items, got %d", expected, len(items))
	}
	for i := 0; i < expected; i++ {
		want := i + startOffset
		if items[i] != want {
			t.Fatalf("expected items[%d]==%d, got %d", i, want, items[i])
		}
	}
}

func TestCollectAllV2_EmptyFirstPage(t *testing.T) {
	ctx := context.Background()

	p := NewPaginator[int](func(ctx context.Context, cursor *Cursor) (Result[int], error) {
		return Result[int]{Items: nil, NextCursor: nil}, nil
	})

	items, err := CollectAll[int](ctx, p, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(items))
	}
}

func TestCollectAllV2_ErrorMidway(t *testing.T) {
	ctx := context.Background()

	wantErr := errors.New("boom")
	call := 0

	p := NewPaginator[int](func(ctx context.Context, cursor *Cursor) (Result[int], error) {
		call++
		if call == 2 {
			return Result[int]{}, wantErr
		}
		// First call returns a full page of 10 with a next cursor
		items := make([]int, 10)
		for i := 0; i < 10; i++ {
			items[i] = i
		}
		return Result[int]{Items: items, NextCursor: loCursor(10)}, nil
	})

	items, err := CollectAll[int](ctx, p, nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err.Error() != wantErr.Error() {
		t.Fatalf("expected error %q, got %q", wantErr.Error(), err.Error())
	}
	// v2 returns the collected items so far along with the error
	if len(items) != 10 {
		t.Fatalf("expected 10 items, got %d", len(items))
	}
}

func TestCollectAllV2_MaxSafeIterCap(t *testing.T) {
	ctx := context.Background()

	// Always return 1 item and a next cursor to force hitting the cap
	call := 0
	p := NewPaginator[int](func(ctx context.Context, cursor *Cursor) (Result[int], error) {
		// create a single item per page
		item := call
		call++
		return Result[int]{
			Items:      []int{item},
			NextCursor: loCursor(call),
		}, nil
	})

	items, err := CollectAll[int](ctx, p, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != MAX_SAFE_ITER {
		t.Fatalf("expected %d items (cap), got %d", MAX_SAFE_ITER, len(items))
	}
}

// helpers
func loCursor(offset int) *Cursor {
	c := NewCursor(time.Now(), strconv.Itoa(offset))
	return &c
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
