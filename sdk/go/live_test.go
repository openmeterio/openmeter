package openmeter_test

import (
	"context"
	"os"
	"testing"
	"time"

	openmeter "github.com/openmeterio/openmeter/sdk/go"
)

// TestLive exercises the SDK against a real OpenMeter server. It is skipped
// unless OPENMETER_BASE_URL is set, so it never runs during a normal `go test`.
//
//	OPENMETER_BASE_URL=http://127.0.0.1:8888/api/v3 \
//	  go test -run TestLive -v ./...
//
// Set OPENMETER_TOKEN to send a bearer token when the target requires auth.
func TestLive(t *testing.T) {
	baseURL := os.Getenv("OPENMETER_BASE_URL")
	if baseURL == "" {
		t.Skip("set OPENMETER_BASE_URL to run the live test")
	}

	opts := []openmeter.Option{}
	if token := os.Getenv("OPENMETER_TOKEN"); token != "" {
		opts = append(opts, openmeter.WithToken(token))
	}

	client, err := openmeter.New(baseURL, opts...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx := context.Background()

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

	first := page.Data[0]
	t.Logf("first meter: id=%s key=%s aggregation=%s", first.ID, first.Key, first.Aggregation)

	// Get: round-trips a single meter by ID.
	got, err := client.Meters.Get(ctx, first.ID)
	if err != nil {
		t.Fatalf("Get(%s): %v", first.ID, err)
	}
	if got.ID != first.ID {
		t.Fatalf("Get returned id %s, want %s", got.ID, first.ID)
	}

	// Query: POST body + JSON result.
	from := time.Now().Add(-30 * 24 * time.Hour)
	day := openmeter.MeterQueryGranularityDay
	res, err := client.Meters.Query(ctx, first.ID, openmeter.MeterQueryRequest{
		From:        &from,
		Granularity: &day,
	})
	if err != nil {
		t.Fatalf("Query(%s): %v", first.ID, err)
	}
	t.Logf("Query returned %d rows", len(res.Data))

	// QueryCSV: same query, CSV content negotiation.
	csv, err := client.Meters.QueryCSV(ctx, first.ID, openmeter.MeterQueryRequest{
		From:        &from,
		Granularity: &day,
	})
	if err != nil {
		t.Fatalf("QueryCSV(%s): %v", first.ID, err)
	}
	t.Logf("QueryCSV returned %d bytes:\n%s", len(csv), string(csv))
}
