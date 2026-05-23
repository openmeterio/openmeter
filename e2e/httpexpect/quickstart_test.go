package httpexpect_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/assert"

	apiv3 "github.com/openmeterio/openmeter/api/v3"
)

// TestQuickstartV3 exercises the ingest → query flow using the v3 API:
//  1. Create a meter (COUNT, event type "request", dimensions method+route).
//  2. Ingest three CloudEvents via the v3 ingest endpoint.
//  3. Poll the v3 meter query endpoint until ClickHouse has produced the
//     expected two time-window buckets.
func TestQuickstartV3(t *testing.T) {
	e := newV3Expect(t)
	address := openmeterAddress(t)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	// Step 1: create the meter.
	dimensions := map[string]string{
		"method": "$.method",
		"route":  "$.route",
	}
	createReq := apiv3.CreateMeterRequest{
		Key:         "quickstart_v3_" + suffix,
		Name:        "Quickstart V3 Meter",
		Aggregation: apiv3.MeterAggregationCount,
		EventType:   "request",
		Dimensions:  &dimensions,
	}

	var meter apiv3.Meter
	e.POST("/meters").
		WithJSON(createReq).
		Expect().
		Status(http.StatusCreated).
		JSON().Decode(&meter)

	// Step 2: ingest three CloudEvents.
	// Events 1 and 2 share the Jan 1 hour bucket → value "2".
	// Event 3 is Jan 2 → separate bucket, value "1".
	makeEvent := func(id string, ts time.Time) apiv3.MeteringEvent {
		return apiv3.MeteringEvent{
			Specversion:     "1.0",
			Id:              "quickstart-v3-" + suffix + "-" + id,
			Source:          "service-0",
			Type:            "request",
			Subject:         "customer-1",
			Time:            nullable.NewNullableWithValue(ts),
			Datacontenttype: nullable.NewNullableWithValue(apiv3.MeteringEventDatacontenttype("application/json")),
			Data:            nullable.NewNullableWithValue(map[string]any{"method": "GET", "route": "/hello"}),
		}
	}

	for _, ev := range []apiv3.MeteringEvent{
		makeEvent("1", time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
		makeEvent("2", time.Date(2023, 1, 1, 0, 30, 0, 0, time.UTC)),
		makeEvent("3", time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)),
	} {
		e.POST("/events").
			WithHeader("Content-Type", "application/cloudevents+json").
			WithJSON(ev).
			Expect().
			Status(http.StatusAccepted)
	}

	// Step 3: poll until ClickHouse has processed all events.
	granularity := apiv3.MeterQueryGranularity("PT1H")
	queryReq := apiv3.MeterQueryRequest{Granularity: &granularity}

	assert.EventuallyWithT(t, func(ct *assert.CollectT) {
		var result apiv3.MeterQueryResult
		newExpectCollect(ct, address).
			POST("/meters/" + meter.Id + "/query").
			WithJSON(queryReq).
			Expect().
			Status(http.StatusOK).
			JSON().Decode(&result)

		if assert.Len(ct, result.Data, 2) {
			assert.Equal(ct, apiv3.Numeric("2"), result.Data[0].Value)
			assert.Equal(ct, apiv3.Numeric("1"), result.Data[1].Value)
		}
	}, 30*time.Second, time.Second)
}
