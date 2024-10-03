package clickhouse_connector

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cloudevents/sdk-go/v2/event"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/pkg/models"
)

type MeterView struct {
	Slug        string
	Aggregation models.MeterAggregation
	GroupBy     []string
}

type ScannedEventRow struct {
	id              string
	eventType       string
	subject         string
	source          string
	eventTime       time.Time
	dataStr         string
	validationError string
	ingestedAt      time.Time
	storedAt        time.Time
}

func parseEventRow(row ScannedEventRow) (api.IngestedEvent, error) {
	var data interface{}
	err := json.Unmarshal([]byte(row.dataStr), &data)
	if err != nil {
		return api.IngestedEvent{}, fmt.Errorf("query events parse data: %w", err)
	}

	event := event.New()
	event.SetID(row.id)
	event.SetType(row.eventType)
	event.SetSubject(row.subject)
	event.SetSource(row.source)
	event.SetTime(row.eventTime)
	err = event.SetData("application/json", data)
	if err != nil {
		return api.IngestedEvent{}, fmt.Errorf("query events set data: %w", err)
	}

	ingestedEvent := api.IngestedEvent{
		Event: event,
	}

	if row.validationError != "" {
		ingestedEvent.ValidationError = &row.validationError
	}

	ingestedEvent.IngestedAt = row.ingestedAt
	ingestedEvent.StoredAt = row.storedAt

	return ingestedEvent, nil
}
