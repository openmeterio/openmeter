package openmeter_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/openmeterio/openmeter/sdk/go/openmeter"
)

func Example() {
	client, err := openmeter.New("https://openmeter.cloud", openmeter.WithToken("om_..."))
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// List meters, filtered and sorted, one page at a time.
	page, err := client.Meters.List(ctx, openmeter.MeterListParams{
		Page:   &openmeter.PageParams{Size: openmeter.Int(20), Number: openmeter.Int(1)},
		Sort:   []string{"created_at desc"},
		Filter: &openmeter.MeterFilter{Key: &openmeter.StringFilter{Contains: openmeter.String("tokens")}},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Query the first meter's usage grouped by day.
	if len(page.Data) > 0 {
		gran := openmeter.MeterQueryGranularityDay
		from := time.Now().Add(-7 * 24 * time.Hour)
		result, err := client.Meters.Query(ctx, page.Data[0].ID, openmeter.MeterQueryRequest{
			From:        &from,
			Granularity: &gran,
		})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("rows: %d\n", len(result.Data))
	}
}
