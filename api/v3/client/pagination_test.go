// Hand-written wire tests for the generated OpenMeter Go SDK. The generator's
// output cleaner preserves *_test.go files, so these survive regeneration.
//
// This file is an internal (package openmeter) test so it can drive the
// unexported paginate/paginateCursor iterators directly; the maxPages guard in
// particular would need thousands of HTTP round trips to exercise black-box.
package openmeter

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestPaginateWalksPagesUntilTotal(t *testing.T) {
	t.Parallel()

	var pagesFetched []int
	seq := paginate(&PageParams{Size: Int(2)}, func(page, size int) ([]int, int, error) {
		if size != 2 {
			t.Errorf("fetch size = %d, want 2", size)
		}
		pagesFetched = append(pagesFetched, page)
		switch page {
		case 1:
			return []int{1, 2}, 5, nil
		case 2:
			return []int{3, 4}, 5, nil
		case 3:
			return []int{5}, 5, nil
		default:
			return nil, 5, fmt.Errorf("unexpected fetch for page %d", page)
		}
	})

	var got []int
	for item, err := range seq {
		if err != nil {
			t.Fatalf("iteration error: %v", err)
		}
		got = append(got, item)
	}

	if want := []int{1, 2, 3, 4, 5}; fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("items = %v, want %v", got, want)
	}
	if want := []int{1, 2, 3}; fmt.Sprint(pagesFetched) != fmt.Sprint(want) {
		t.Errorf("pages fetched = %v, want %v", pagesFetched, want)
	}
}

func TestPaginateStopsOnEmptyPage(t *testing.T) {
	t.Parallel()

	// The server reports no total, so only an empty page terminates the walk.
	fetches := 0
	seq := paginate(nil, func(page, size int) ([]int, int, error) {
		fetches++
		if page == 1 {
			return []int{1, 2}, 0, nil
		}
		return nil, 0, nil
	})

	var got []int
	for item, err := range seq {
		if err != nil {
			t.Fatalf("iteration error: %v", err)
		}
		got = append(got, item)
	}

	if len(got) != 2 {
		t.Errorf("items = %v, want 2 items", got)
	}
	if fetches != 2 {
		t.Errorf("fetches = %d, want 2 (stop right after the empty page)", fetches)
	}
}

func TestPaginateStartsFromRequestedPage(t *testing.T) {
	t.Parallel()

	var first int
	seq := paginate(&PageParams{Number: Int(3)}, func(page, size int) ([]int, int, error) {
		if first == 0 {
			first = page
		}
		return nil, 0, nil
	})
	for _, err := range seq {
		if err != nil {
			t.Fatalf("iteration error: %v", err)
		}
	}

	if first != 3 {
		t.Errorf("first fetched page = %d, want 3", first)
	}
}

func TestPaginateMaxPagesGuard(t *testing.T) {
	t.Parallel()

	// A server that always returns data and never a total would loop forever
	// without the guard.
	fetches := 0
	seq := paginate(nil, func(page, size int) ([]int, int, error) {
		fetches++
		return []int{page}, 0, nil
	})

	items := 0
	var finalErr error
	for _, err := range seq {
		if err != nil {
			finalErr = err
			break
		}
		items++
	}

	if finalErr == nil {
		t.Fatal("pagination terminated without error, want the maxPages guard to fire")
	}
	if !strings.Contains(finalErr.Error(), "did not terminate within") {
		t.Errorf("guard error = %q, want it to mention non-termination", finalErr)
	}
	if fetches != maxPages {
		t.Errorf("fetches = %d, want exactly maxPages (%d)", fetches, maxPages)
	}
	if items != maxPages {
		t.Errorf("items = %d, want %d", items, maxPages)
	}
}

func TestPaginateCursorForwardFollowsNext(t *testing.T) {
	t.Parallel()

	var afters []*string
	seq := paginateCursor(nil, func(after, before *string, size int) ([]string, *string, *string, error) {
		if before != nil {
			t.Errorf("forward paging sent before=%q, want nil", *before)
		}
		afters = append(afters, after)
		if after == nil {
			return []string{"a", "b"}, String("c2"), nil, nil
		}
		if *after == "c2" {
			return []string{"c"}, nil, String("c1"), nil
		}
		return nil, nil, nil, fmt.Errorf("unexpected after cursor %q", *after)
	})

	var got []string
	for item, err := range seq {
		if err != nil {
			t.Fatalf("iteration error: %v", err)
		}
		got = append(got, item)
	}

	if want := "a b c"; strings.Join(got, " ") != want {
		t.Errorf("items = %v, want %q", got, want)
	}
	if len(afters) != 2 || afters[0] != nil || afters[1] == nil || *afters[1] != "c2" {
		t.Errorf("after cursors sent = %v, want [nil c2]", afters)
	}
}

func TestPaginateCursorBackwardFollowsPrevious(t *testing.T) {
	t.Parallel()

	// given: iteration starts from a before cursor
	// when: each page reports a previous cursor until the first page
	// then: the walk follows previous (never next) and stops when previous is gone
	var befores []string
	seq := paginateCursor(&CursorPageParams{Before: String("c9")}, func(after, before *string, size int) ([]string, *string, *string, error) {
		if after != nil {
			t.Errorf("backward paging sent after=%q, want nil", *after)
		}
		if before == nil {
			t.Fatal("backward paging sent no before cursor")
		}
		befores = append(befores, *before)
		switch *before {
		case "c9":
			// A next cursor is present too; backward paging must ignore it.
			return []string{"x", "y"}, String("must-not-follow"), String("c8"), nil
		case "c8":
			return []string{"z"}, String("must-not-follow"), nil, nil
		default:
			return nil, nil, nil, fmt.Errorf("unexpected before cursor %q", *before)
		}
	})

	var got []string
	for item, err := range seq {
		if err != nil {
			t.Fatalf("iteration error: %v", err)
		}
		got = append(got, item)
	}

	if want := "x y z"; strings.Join(got, " ") != want {
		t.Errorf("items = %v, want %q", got, want)
	}
	if want := "c9 c8"; strings.Join(befores, " ") != want {
		t.Errorf("before cursors sent = %v, want %q", befores, want)
	}
}

func TestPaginateCursorRejectsBothCursors(t *testing.T) {
	t.Parallel()

	seq := paginateCursor(&CursorPageParams{After: String("a"), Before: String("b")}, func(after, before *string, size int) ([]string, *string, *string, error) {
		t.Error("fetch called despite invalid cursor combination")
		return nil, nil, nil, nil
	})

	var errs []error
	for _, err := range seq {
		errs = append(errs, err)
	}

	if len(errs) != 1 || errs[0] == nil {
		t.Fatalf("yields = %v, want exactly one error", errs)
	}
	if !strings.Contains(errs[0].Error(), "cannot use both after and before") {
		t.Errorf("error = %q, want it to reject the after+before combination", errs[0])
	}
}

func TestPaginateCursorEarlyBreakStopsFetching(t *testing.T) {
	t.Parallel()

	fetches := 0
	seq := paginateCursor(nil, func(after, before *string, size int) ([]string, *string, *string, error) {
		fetches++
		return []string{"a", "b"}, String("next"), nil, nil
	})

	for item, err := range seq {
		if err != nil {
			t.Fatalf("iteration error: %v", err)
		}
		if item == "a" {
			break
		}
	}

	if fetches != 1 {
		t.Errorf("fetches = %d, want 1 (breaking the loop must stop paging)", fetches)
	}
}

func TestPaginateCursorMaxPagesGuard(t *testing.T) {
	t.Parallel()

	seq := paginateCursor(nil, func(after, before *string, size int) ([]string, *string, *string, error) {
		return []string{"item"}, String("again"), nil, nil
	})

	items := 0
	var finalErr error
	for _, err := range seq {
		if err != nil {
			finalErr = err
			break
		}
		items++
	}

	if finalErr == nil || !strings.Contains(finalErr.Error(), "did not terminate within") {
		t.Fatalf("final error = %v, want the maxPages guard to fire", finalErr)
	}
	if items != maxPages {
		t.Errorf("items = %d, want %d", items, maxPages)
	}
}

// newHTTPTestClient wires a Client to an httptest server for the end-to-end
// ListAll tests below.
func newHTTPTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	c, err := New(srv.URL)
	if err != nil {
		t.Fatalf("New(%q): %v", srv.URL, err)
	}
	return c
}

func TestMetersListAllWalksPagesOverHTTP(t *testing.T) {
	t.Parallel()

	var hits atomic.Int32
	om := newHTTPTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if got := r.URL.Query().Get("page[size]"); got != "2" {
			t.Errorf("page[size] = %q, want %q", got, "2")
		}
		w.Header().Set("Content-Type", "application/json")
		switch page := r.URL.Query().Get("page[number]"); page {
		case "1":
			_, _ = io.WriteString(w, `{"data":[{"key":"m1"},{"key":"m2"}],"meta":{"page":{"number":1,"size":2,"total":3}}}`)
		case "2":
			_, _ = io.WriteString(w, `{"data":[{"key":"m3"}],"meta":{"page":{"number":2,"size":2,"total":3}}}`)
		default:
			t.Errorf("unexpected page[number] %q requested", page)
			_, _ = io.WriteString(w, `{"data":[],"meta":{"page":{"number":0,"size":2,"total":3}}}`)
		}
	})

	var keys []string
	params := MeterListParams{Page: &PageParams{Size: Int(2)}}
	for meter, err := range om.Meters.ListAll(t.Context(), params) {
		if err != nil {
			t.Fatalf("ListAll: %v", err)
		}
		keys = append(keys, meter.Key)
	}

	if want := "m1 m2 m3"; strings.Join(keys, " ") != want {
		t.Errorf("meter keys = %v, want %q", keys, want)
	}
	if got := hits.Load(); got != 2 {
		t.Errorf("server hits = %d, want 2", got)
	}
}

func TestMetersListAllEarlyBreakStopsRequests(t *testing.T) {
	t.Parallel()

	var hits atomic.Int32
	om := newHTTPTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"data":[{"key":"m1"},{"key":"m2"}],"meta":{"page":{"number":1,"size":2,"total":100}}}`)
	})

	for _, err := range om.Meters.ListAll(t.Context(), MeterListParams{Page: &PageParams{Size: Int(2)}}) {
		if err != nil {
			t.Fatalf("ListAll: %v", err)
		}
		break
	}

	if got := hits.Load(); got != 1 {
		t.Errorf("server hits = %d, want 1 (breaking the loop must stop fetching)", got)
	}
}

func TestEventsListAllFollowsNextCursorOverHTTP(t *testing.T) {
	t.Parallel()

	var hits atomic.Int32
	om := newHTTPTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if got := r.URL.Query().Get("page[before]"); got != "" {
			t.Errorf("forward paging sent page[before]=%q, want none", got)
		}
		if got := r.URL.Query().Get("page[size]"); got != "2" {
			t.Errorf("page[size] = %q, want %q", got, "2")
		}
		w.Header().Set("Content-Type", "application/json")
		switch after := r.URL.Query().Get("page[after]"); after {
		case "":
			_, _ = io.WriteString(w, `{"data":[{"event":{"id":"e1"}},{"event":{"id":"e2"}}],"meta":{"page":{"size":2,"next":"cur2","previous":null}}}`)
		case "cur2":
			_, _ = io.WriteString(w, `{"data":[{"event":{"id":"e3"}}],"meta":{"page":{"size":2,"next":null,"previous":"cur1"}}}`)
		default:
			t.Errorf("unexpected page[after] %q requested", after)
			_, _ = io.WriteString(w, `{"data":[],"meta":{"page":{"size":2,"next":null,"previous":null}}}`)
		}
	})

	var ids []string
	params := IngestedEventListParams{Page: &CursorPageParams{Size: Int(2)}}
	for event, err := range om.Events.ListAll(t.Context(), params) {
		if err != nil {
			t.Fatalf("ListAll: %v", err)
		}
		ids = append(ids, event.Event.ID)
	}

	if want := "e1 e2 e3"; strings.Join(ids, " ") != want {
		t.Errorf("event ids = %v, want %q", ids, want)
	}
	if got := hits.Load(); got != 2 {
		t.Errorf("server hits = %d, want 2", got)
	}
}

func TestEventsListAllFollowsPreviousCursorWithBefore(t *testing.T) {
	t.Parallel()

	var hits atomic.Int32
	om := newHTTPTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if got := r.URL.Query().Get("page[after]"); got != "" {
			t.Errorf("backward paging sent page[after]=%q, want none", got)
		}
		w.Header().Set("Content-Type", "application/json")
		switch before := r.URL.Query().Get("page[before]"); before {
		case "cur9":
			_, _ = io.WriteString(w, `{"data":[{"event":{"id":"e9a"}},{"event":{"id":"e9b"}}],"meta":{"page":{"size":2,"next":"cur10","previous":"cur8"}}}`)
		case "cur8":
			_, _ = io.WriteString(w, `{"data":[{"event":{"id":"e8"}}],"meta":{"page":{"size":2,"next":"cur9","previous":null}}}`)
		default:
			t.Errorf("unexpected page[before] %q requested", before)
			_, _ = io.WriteString(w, `{"data":[],"meta":{"page":{"size":2,"next":null,"previous":null}}}`)
		}
	})

	var ids []string
	params := IngestedEventListParams{Page: &CursorPageParams{Before: String("cur9")}}
	for event, err := range om.Events.ListAll(t.Context(), params) {
		if err != nil {
			t.Fatalf("ListAll: %v", err)
		}
		ids = append(ids, event.Event.ID)
	}

	if want := "e9a e9b e8"; strings.Join(ids, " ") != want {
		t.Errorf("event ids = %v, want %q", ids, want)
	}
	if got := hits.Load(); got != 2 {
		t.Errorf("server hits = %d, want 2", got)
	}
}
