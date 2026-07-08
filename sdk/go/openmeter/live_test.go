package openmeter_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/openmeterio/openmeter/sdk/go/openmeter"
)

// newLiveClient builds a client pointed at a real server from the environment.
// It skips the test unless OPENMETER_BASE_URL is set, so live tests never run
// during a normal `go test`. OPENMETER_TOKEN, when set, is sent as a bearer
// token. The returned context is bounded so a call can't hang against a stuck
// server, and is canceled at test end.
//
//	OPENMETER_BASE_URL=http://127.0.0.1:8888/api/v3 \
//	  go test -run TestLive -v ./...
func newLiveClient(t *testing.T) (*openmeter.Client, context.Context) {
	t.Helper()

	baseURL := os.Getenv("OPENMETER_BASE_URL")
	if baseURL == "" {
		t.Skip("set OPENMETER_BASE_URL to run live tests")
	}

	var opts []openmeter.Option
	if token := os.Getenv("OPENMETER_TOKEN"); token != "" {
		opts = append(opts, openmeter.WithToken(token))
	}

	client, err := openmeter.New(baseURL, opts...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	t.Cleanup(cancel)

	return client, ctx
}

// TestLiveReadOnly exercises the read operations (list, get, filter, paginate,
// query) against a real server. Safe to run against shared environments.
func TestLiveReadOnly(t *testing.T) {
	client, ctx := newLiveClient(t)

	// List: exercises query-string params against a live endpoint.
	page, err := client.Meters.List(ctx, openmeter.MeterListParams{
		Page: &openmeter.PageParams{Size: openmeter.Int(10), Number: openmeter.Int(1)},
		Sort: []string{"created_at"},
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	t.Logf("List returned %d meters (page total %d)", len(page.Data), page.Meta.Page.Total)

	if len(page.Data) == 0 {
		t.Skip("no meters on server; seed one to exercise Get/Query")
	}

	firstMeter := page.Data[0]

	t.Logf("first meter: id=%s key=%s aggregation=%s", firstMeter.ID, firstMeter.Key, firstMeter.Aggregation)

	// Get: round-trips a single meter by ID.
	fetchedMeter, err := client.Meters.Get(ctx, firstMeter.ID)
	if err != nil {
		t.Fatalf("Get(%s): %v", firstMeter.ID, err)
	}

	if fetchedMeter.ID != firstMeter.ID {
		t.Fatalf("Get returned id %s, want %s", fetchedMeter.ID, firstMeter.ID)
	}

	// Filter: exact-match on the first meter's key should return exactly it.
	filtered, err := client.Meters.List(ctx, openmeter.MeterListParams{
		Filter: &openmeter.MeterFilter{Key: &openmeter.StringFilter{Eq: openmeter.String(firstMeter.Key)}},
	})
	if err != nil {
		t.Fatalf("List(filter key eq %q): %v", firstMeter.Key, err)
	}

	if len(filtered.Data) != 1 || filtered.Data[0].Key != firstMeter.Key {
		t.Fatalf("filter key eq %q returned %d meters, want 1 (%q)", firstMeter.Key, len(filtered.Data), firstMeter.Key)
	}

	t.Logf("filter key eq %q matched %d meter", firstMeter.Key, len(filtered.Data))

	// Filter: a key that does not exist should return no meters.
	none, err := client.Meters.List(ctx, openmeter.MeterListParams{
		Filter: &openmeter.MeterFilter{Key: &openmeter.StringFilter{Eq: openmeter.String("no-such-meter-key-xyz")}},
	})
	if err != nil {
		t.Fatalf("List(filter no-match): %v", err)
	}

	if len(none.Data) != 0 {
		t.Fatalf("filter no-match returned %d meters, want 0", len(none.Data))
	}

	// ListAll: iterate every meter across pages. A small page size forces the
	// iterator to fetch multiple pages against a real server.
	iteratedCount := 0
	for meter, err := range client.Meters.ListAll(ctx, openmeter.MeterListParams{
		Page: &openmeter.PageParams{Size: openmeter.Int(2)},
	}) {
		if err != nil {
			t.Fatalf("ListAll: %v", err)
		}
		if meter.ID == "" {
			t.Fatal("ListAll yielded a meter with empty ID")
		}
		iteratedCount++
	}

	t.Logf("ListAll iterated %d meters", iteratedCount)

	if iteratedCount != page.Meta.Page.Total {
		t.Fatalf("ListAll count %d != reported total %d", iteratedCount, page.Meta.Page.Total)
	}

	// Query: POST body + JSON result.
	from := time.Now().Add(-30 * 24 * time.Hour)
	day := openmeter.MeterQueryGranularityDay

	queryResult, err := client.Meters.Query(ctx, firstMeter.ID, openmeter.MeterQueryRequest{
		From:        &from,
		Granularity: &day,
	})
	if err != nil {
		t.Fatalf("Query(%s): %v", firstMeter.ID, err)
	}

	t.Logf("Query returned %d rows", len(queryResult.Data))

	// QueryCSV: same query, CSV content negotiation.
	csvData, err := client.Meters.QueryCSV(ctx, firstMeter.ID, openmeter.MeterQueryRequest{
		From:        &from,
		Granularity: &day,
	})
	if err != nil {
		t.Fatalf("QueryCSV(%s): %v", firstMeter.ID, err)
	}

	t.Logf("QueryCSV returned %d bytes:\n%s", len(csvData), string(csvData))
}

// TestLiveReadWrite exercises the mutating operations (create, update, delete)
// against a real server. These write to the target, so it is additionally gated
// behind OPENMETER_LIVE_MUTATE to avoid mutating shared environments by default.
func TestLiveReadWrite(t *testing.T) {
	client, ctx := newLiveClient(t)

	if os.Getenv("OPENMETER_LIVE_MUTATE") == "" {
		t.Skip("set OPENMETER_LIVE_MUTATE=1 to run the create/update/delete cycle")
	}

	// Create -> Get -> Update -> Delete a throwaway meter.
	created, err := client.Meters.Create(ctx, openmeter.CreateMeterRequest{
		Name:        "SDK baseline smoke test",
		Key:         "sdk_baseline_smoke_test",
		Aggregation: openmeter.MeterAggregationCount,
		EventType:   "sdk_baseline_smoke_test",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Logf("created meter id=%s key=%s", created.ID, created.Key)

	// Best-effort cleanup if a step below fails before the explicit delete runs.
	// Tolerant by design: once the explicit delete succeeds this is a no-op that
	// 404s, which we ignore.
	deleted := false
	defer func() {
		if !deleted {
			_ = client.Meters.Delete(ctx, created.ID)
		}
	}()

	fetchedMeter, err := client.Meters.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get(%s): %v", created.ID, err)
	}
	if fetchedMeter.Key != created.Key {
		t.Fatalf("Get key = %q, want %q", fetchedMeter.Key, created.Key)
	}

	updated, err := client.Meters.Update(ctx, created.ID, openmeter.UpdateMeterRequest{
		Name: openmeter.String("SDK baseline smoke test (renamed)"),
	})
	if err != nil {
		t.Fatalf("Update(%s): %v", created.ID, err)
	}
	t.Logf("updated meter name=%q", updated.Name)

	if err := client.Meters.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete(%s): %v", created.ID, err)
	}
	deleted = true
	t.Logf("deleted meter id=%s", created.ID)
}
