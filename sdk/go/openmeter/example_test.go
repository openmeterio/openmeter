package openmeter_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/openmeterio/openmeter/sdk/go/openmeter"
)

// Example constructs a client and runs a create -> get flow.
func Example() {
	client, err := openmeter.New("https://openmeter.cloud/api/v3", openmeter.WithToken("om_..."))
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	meter, err := client.Meters.Create(ctx, openmeter.CreateMeterRequest{
		Name:          "Tokens",
		Key:           "tokens_total",
		Aggregation:   openmeter.MeterAggregationSum,
		EventType:     "prompt",
		ValueProperty: openmeter.String("$.tokens"),
	})
	if err != nil {
		log.Fatal(err)
	}

	fetched, err := client.Meters.Get(ctx, meter.ID)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(fetched.Key)
}

// ExampleMetersService_Update updates a meter; all request fields are optional.
func ExampleMetersService_Update() {
	client, _ := openmeter.New("https://openmeter.cloud/api/v3", openmeter.WithToken("om_..."))

	updated, err := client.Meters.Update(context.Background(), "01ABC", openmeter.UpdateMeterRequest{
		Name: openmeter.String("Tokens (renamed)"),
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(updated.Name)
}

// ExampleMetersService_Delete deletes a meter by ID.
func ExampleMetersService_Delete() {
	client, _ := openmeter.New("https://openmeter.cloud/api/v3", openmeter.WithToken("om_..."))

	if err := client.Meters.Delete(context.Background(), "01ABC"); err != nil {
		log.Fatal(err)
	}
}

// ExampleMetersService_List lists meters with pagination, sort, and filter.
func ExampleMetersService_List() {
	client, _ := openmeter.New("https://openmeter.cloud/api/v3", openmeter.WithToken("om_..."))

	page, err := client.Meters.List(context.Background(), openmeter.MeterListParams{
		Page:   &openmeter.PageParams{Size: openmeter.Int(20), Number: openmeter.Int(1)},
		Sort:   []string{"created_at desc"},
		Filter: &openmeter.MeterFilter{Key: &openmeter.StringFilter{Contains: openmeter.String("tokens")}},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%d of %d meters\n", len(page.Data), page.Meta.Page.Total)
}

// ExampleMetersService_ListAll iterates every meter across pages (Go 1.23+
// range-over-func).
func ExampleMetersService_ListAll() {
	client, _ := openmeter.New("https://openmeter.cloud/api/v3", openmeter.WithToken("om_..."))

	for meter, err := range client.Meters.ListAll(context.Background(), openmeter.MeterListParams{}) {
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(meter.Key)
	}
}

// ExampleMetersService_Query queries a meter for usage; QueryCSV returns the
// same data with Accept: text/csv.
func ExampleMetersService_Query() {
	client, _ := openmeter.New("https://openmeter.cloud/api/v3", openmeter.WithToken("om_..."))

	day := openmeter.MeterQueryGranularityDay
	from := time.Now().Add(-7 * 24 * time.Hour)

	result, err := client.Meters.Query(context.Background(), "01ABC", openmeter.MeterQueryRequest{
		From:        &from,
		Granularity: &day,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%d rows\n", len(result.Data))
}

// ExampleMetersService_QueryCSVStream streams a large CSV export without
// buffering the whole payload.
func ExampleMetersService_QueryCSVStream() {
	client, _ := openmeter.New("https://openmeter.cloud/api/v3", openmeter.WithToken("om_..."))

	stream, err := client.Meters.QueryCSVStream(context.Background(), "01ABC", openmeter.MeterQueryRequest{})
	if err != nil {
		log.Fatal(err)
	}
	defer stream.Close()

	if _, err := io.Copy(io.Discard, stream); err != nil {
		log.Fatal(err)
	}
}

// ExamplePlanAddonsService shows the nested sub-resource shape: operations on
// /plans/{planId}/addons take the parent plan ID as their first argument.
func ExamplePlanAddonsService() {
	client, _ := openmeter.New("https://openmeter.cloud/api/v3", openmeter.WithToken("om_..."))

	ctx := context.Background()
	planID := "01PLAN"

	created, err := client.PlanAddons.Create(ctx, planID, openmeter.CreatePlanAddonRequest{
		Name:          "Pro add-on",
		Addon:         openmeter.AddonReference{ID: "01ADDON"},
		FromPlanPhase: "trial",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(created.ID)

	for addon, err := range client.PlanAddons.ListAll(ctx, planID, openmeter.PlanAddonListParams{}) {
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(addon.Name)
	}
}

// ExampleWithHTTPClient injects a go-retryablehttp client with a custom retry
// policy. Its retry behavior stays hidden behind the standard *http.Client, so
// the SDK's public surface is unchanged.
func ExampleWithHTTPClient() {
	rc := retryablehttp.NewClient()
	rc.RetryMax = 5

	client, err := openmeter.New("https://openmeter.cloud/api/v3",
		openmeter.WithToken("om_..."),
		openmeter.WithHTTPClient(rc.StandardClient()),
	)
	if err != nil {
		log.Fatal(err)
	}

	_ = client
}

// ExampleWithHTTPClient_noRetries injects a plain client with a fixed timeout
// and no retries.
func ExampleWithHTTPClient_noRetries() {
	client, err := openmeter.New("https://openmeter.cloud/api/v3",
		openmeter.WithHTTPClient(&http.Client{Timeout: 10 * time.Second}),
	)
	if err != nil {
		log.Fatal(err)
	}

	_ = client
}

// tracingTransport is a custom http.RoundTripper, e.g. for tracing or injecting
// headers on every request.
type tracingTransport struct {
	base http.RoundTripper
}

func (t tracingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}

	return base.RoundTrip(r)
}

// ExampleWithHTTPClient_customTransport injects a client with a custom
// http.RoundTripper.
func ExampleWithHTTPClient_customTransport() {
	client, err := openmeter.New("https://openmeter.cloud/api/v3",
		openmeter.WithHTTPClient(&http.Client{Transport: tracingTransport{}}),
	)
	if err != nil {
		log.Fatal(err)
	}

	_ = client
}

// ExampleAPIError inspects a typed error returned for a non-2xx response.
func ExampleAPIError() {
	client, _ := openmeter.New("https://openmeter.cloud/api/v3", openmeter.WithToken("om_..."))

	_, err := client.Meters.Get(context.Background(), "missing")

	var apiErr *openmeter.APIError
	if errors.As(err, &apiErr) {
		fmt.Printf("status=%d detail=%s trace=%s\n", apiErr.StatusCode, apiErr.Detail, apiErr.Instance)
	}
}
