package openmeter

import (
	"strings"
	"testing"
)

func TestPaginate_BackstopStopsNonTerminatingServer(t *testing.T) {
	// A server that always returns a full page while reporting a non-positive
	// total disables the total guard and would otherwise loop forever. The
	// backstop must stop it with an error after maxPages fetches.
	calls := 0
	it := paginate(nil, func(page, size int) ([]int, int, error) {
		calls++
		return []int{page}, 0, nil
	})

	yielded := 0
	var lastErr error
	for _, err := range it {
		if err != nil {
			lastErr = err
			break
		}
		yielded++
	}

	if lastErr == nil {
		t.Fatal("expected a backstop error, got nil")
	}
	if !strings.Contains(lastErr.Error(), "did not terminate") {
		t.Fatalf("error = %v, want a pagination-did-not-terminate error", lastErr)
	}
	if calls != maxPages {
		t.Fatalf("fetch calls = %d, want %d", calls, maxPages)
	}
	if yielded != maxPages {
		t.Fatalf("yielded %d items, want %d", yielded, maxPages)
	}
}
