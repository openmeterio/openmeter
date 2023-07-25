package kafka_connector

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	. "github.com/openmeterio/openmeter/internal/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestDetectedEventsTableQuery(t *testing.T) {
	tests := []struct {
		data detectedEventsTableQueryData
		want string
	}{
		{
			data: detectedEventsTableQueryData{
				Topic:      "om_detected_events",
				Retention:  32,
				Partitions: 100,
			},
			want: "CREATE TABLE IF NOT EXISTS OM_DETECTED_EVENTS WITH ( KAFKA_TOPIC = 'om_detected_events', KEY_FORMAT = 'JSON_SR', VALUE_FORMAT = 'JSON_SR', PARTITIONS = 100 ) AS SELECT ID AS KEY1, SOURCE AS KEY2, AS_VALUE(ID) AS ID, EARLIEST_BY_OFFSET(TYPE) AS TYPE, AS_VALUE(SOURCE) AS SOURCE, EARLIEST_BY_OFFSET(SUBJECT) AS SUBJECT, EARLIEST_BY_OFFSET(TIME) AS TIME, EARLIEST_BY_OFFSET(DATA) AS DATA, COUNT(ID) as ID_COUNT FROM OM_EVENTS WINDOW TUMBLING ( SIZE 32 DAYS, RETENTION 32 DAYS ) GROUP BY ID, SOURCE;",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("", func(t *testing.T) {
			got, err := templateQuery(detectedEventsTableQueryTemplate, tt.data)
			if err != nil {
				t.Error(err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDetectedEventsStreamQuery(t *testing.T) {
	tests := []struct {
		data detectedEventsStreamQueryData
		want string
	}{
		{
			data: detectedEventsStreamQueryData{
				Topic: "om_detected_events",
			},
			want: "CREATE STREAM IF NOT EXISTS OM_DETECTED_EVENTS_STREAM WITH ( KAFKA_TOPIC = 'om_detected_events', KEY_FORMAT = 'JSON_SR', VALUE_FORMAT = 'JSON_SR' );",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("", func(t *testing.T) {
			got, err := templateQuery(detectedEventsStreamQueryTemplate, tt.data)
			if err != nil {
				t.Error(err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCloudEventsStreamQuery(t *testing.T) {
	tests := []struct {
		data cloudEventsStreamQueryData
		want string
	}{
		{
			data: cloudEventsStreamQueryData{
				Topic:         "om_events",
				KeySchemaId:   1,
				ValueSchemaId: 1,
			},
			want: "CREATE STREAM IF NOT EXISTS OM_EVENTS WITH ( KAFKA_TOPIC = 'om_events', KEY_FORMAT = 'JSON_SR', VALUE_FORMAT = 'JSON_SR', KEY_SCHEMA_ID = 1, VALUE_SCHEMA_ID = 1 );",
		},
		{
			data: cloudEventsStreamQueryData{
				Topic:         "foo",
				KeySchemaId:   2,
				ValueSchemaId: 2,
			},
			want: "CREATE STREAM IF NOT EXISTS OM_EVENTS WITH ( KAFKA_TOPIC = 'foo', KEY_FORMAT = 'JSON_SR', VALUE_FORMAT = 'JSON_SR', KEY_SCHEMA_ID = 2, VALUE_SCHEMA_ID = 2 );",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("", func(t *testing.T) {
			got, err := templateQuery(cloudEventsStreamQueryTemplate, tt.data)
			if err != nil {
				t.Error(err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMeterTableQuery(t *testing.T) {
	tests := []struct {
		data meterTableQueryData
		want string
	}{
		{
			data: meterTableQueryData{
				Meter: &models.Meter{
					Slug:          "meter1",
					Description:   "API Network Traffic",
					ValueProperty: "$.bytes",
					EventType:     "api-calls",
					Aggregation:   models.MeterAggregationSum,
					GroupBy:       map[string]string{"path": "$.path"},
					WindowSize:    models.WindowSizeHour,
				},
				WindowRetention: "365 DAYS",
				Partitions:      1,
			},
			want: "CREATE TABLE IF NOT EXISTS `OM_METER_METER1` WITH ( KAFKA_TOPIC = 'om_meter_meter1', KEY_FORMAT = 'JSON_SR', VALUE_FORMAT = 'JSON_SR', PARTITIONS = 1 ) AS SELECT SUBJECT AS KEY1, AS_VALUE(SUBJECT) AS SUBJECT, WINDOWSTART AS WINDOWSTART_TS, WINDOWEND AS WINDOWEND_TS, COALESCE(EXTRACTJSONFIELD(data, '$.path'), '') AS `path_KEY`, AS_VALUE(COALESCE(EXTRACTJSONFIELD(data, '$.path'), '')) AS `path`, SUM(CAST(EXTRACTJSONFIELD(data, '$.bytes') AS DECIMAL(12, 4))) AS VALUE FROM OM_DETECTED_EVENTS_STREAM WINDOW TUMBLING ( SIZE 1 HOUR, RETENTION 365 DAYS ) WHERE ID_COUNT = 1 AND TYPE = 'api-calls' GROUP BY SUBJECT, COALESCE(EXTRACTJSONFIELD(data, '$.path'), '') EMIT CHANGES;",
		},
		{
			data: meterTableQueryData{
				Meter: &models.Meter{
					Slug:        "meter2",
					Description: "API Calls",
					EventType:   "api-calls",
					Aggregation: models.MeterAggregationCount,
					WindowSize:  models.WindowSizeHour,
				},
				WindowRetention: "365 DAYS",
				Partitions:      1,
			},
			want: "CREATE TABLE IF NOT EXISTS `OM_METER_METER2` WITH ( KAFKA_TOPIC = 'om_meter_meter2', KEY_FORMAT = 'JSON_SR', VALUE_FORMAT = 'JSON_SR', PARTITIONS = 1 ) AS SELECT SUBJECT AS KEY1, AS_VALUE(SUBJECT) AS SUBJECT, WINDOWSTART AS WINDOWSTART_TS, WINDOWEND AS WINDOWEND_TS, COUNT(*) AS VALUE FROM OM_DETECTED_EVENTS_STREAM WINDOW TUMBLING ( SIZE 1 HOUR, RETENTION 365 DAYS ) WHERE ID_COUNT = 1 AND TYPE = 'api-calls' GROUP BY SUBJECT EMIT CHANGES;",
		},
		{
			data: meterTableQueryData{
				Meter: &models.Meter{
					Slug:          "meter2",
					Description:   "API Calls",
					EventType:     "api-calls",
					ValueProperty: "$.duration_ms",
					Aggregation:   models.MeterAggregationCount,
					WindowSize:    models.WindowSizeHour,
				},
				WindowRetention: "365 DAYS",
				Partitions:      1,
			},
			want: "CREATE TABLE IF NOT EXISTS `OM_METER_METER2` WITH ( KAFKA_TOPIC = 'om_meter_meter2', KEY_FORMAT = 'JSON_SR', VALUE_FORMAT = 'JSON_SR', PARTITIONS = 1 ) AS SELECT SUBJECT AS KEY1, AS_VALUE(SUBJECT) AS SUBJECT, WINDOWSTART AS WINDOWSTART_TS, WINDOWEND AS WINDOWEND_TS, COUNT(EXTRACTJSONFIELD(data, '$.duration_ms')) AS VALUE FROM OM_DETECTED_EVENTS_STREAM WINDOW TUMBLING ( SIZE 1 HOUR, RETENTION 365 DAYS ) WHERE ID_COUNT = 1 AND TYPE = 'api-calls' GROUP BY SUBJECT EMIT CHANGES;",
		},
		{
			data: meterTableQueryData{
				Meter: &models.Meter{
					Slug:          "meter3",
					Description:   "API call count by path",
					EventType:     "api-calls",
					Aggregation:   models.MeterAggregationAvg,
					ValueProperty: "$.duration_ms",
					WindowSize:    models.WindowSizeMinute,
				},
				WindowRetention: "365 DAYS",
				Partitions:      1,
			},
			want: "CREATE TABLE IF NOT EXISTS `OM_METER_METER3` WITH ( KAFKA_TOPIC = 'om_meter_meter3', KEY_FORMAT = 'JSON_SR', VALUE_FORMAT = 'JSON_SR', PARTITIONS = 1 ) AS SELECT SUBJECT AS KEY1, AS_VALUE(SUBJECT) AS SUBJECT, WINDOWSTART AS WINDOWSTART_TS, WINDOWEND AS WINDOWEND_TS, AVG(CAST(EXTRACTJSONFIELD(data, '$.duration_ms') AS DECIMAL(12, 4))) AS VALUE FROM OM_DETECTED_EVENTS_STREAM WINDOW TUMBLING ( SIZE 1 MINUTE, RETENTION 365 DAYS ) WHERE ID_COUNT = 1 AND TYPE = 'api-calls' GROUP BY SUBJECT EMIT CHANGES;",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("", func(t *testing.T) {
			got, err := templateQuery(meterTableQueryTemplate, tt.data)
			if err != nil {
				t.Error(err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValuesSelectQuery(t *testing.T) {
	subject := "subject1"
	from, _ := time.Parse(time.RFC3339, "2021-01-01T00:00:00.001Z")
	to, _ := time.Parse(time.RFC3339, "2021-01-02T00:00:00Z")
	tests := []struct {
		data meterValuesData
		want string
	}{
		{
			data: meterValuesData{
				Meter: &models.Meter{
					Slug: "meter1",
				},
				GetValuesParams: &GetValuesParams{
					Subject: &subject,
				},
			},
			want: "SELECT SUBJECT, VALUE, WINDOWSTART, WINDOWEND FROM `OM_METER_METER1` WHERE SUBJECT = 'subject1';",
		},
		{
			data: meterValuesData{
				Meter: &models.Meter{
					Slug: "meter2",
				},
				GetValuesParams: &GetValuesParams{
					Subject: &subject,
					From:    &from,
				},
			},
			want: "SELECT SUBJECT, VALUE, WINDOWSTART, WINDOWEND FROM `OM_METER_METER2` WHERE SUBJECT = 'subject1' AND WINDOWSTART >= 1609459200001;",
		},
		{
			data: meterValuesData{
				Meter: &models.Meter{
					Slug: "meter3",
				},
				GetValuesParams: &GetValuesParams{},
			},
			want: "SELECT SUBJECT, VALUE, WINDOWSTART, WINDOWEND FROM `OM_METER_METER3`;",
		},
		{
			data: meterValuesData{
				Meter: &models.Meter{
					Slug: "meter4",
				},
				GetValuesParams: &GetValuesParams{
					Subject: &subject,
					To:      &to,
				},
			},
			want: "SELECT SUBJECT, VALUE, WINDOWSTART, WINDOWEND FROM `OM_METER_METER4` WHERE SUBJECT = 'subject1' AND WINDOWEND <= 1609545600000;",
		},
		{
			data: meterValuesData{
				Meter: &models.Meter{
					Slug: "meter5",
				},
				GetValuesParams: &GetValuesParams{
					Subject: &subject,
					From:    &from,
					To:      &to,
				},
			},
			want: "SELECT SUBJECT, VALUE, WINDOWSTART, WINDOWEND FROM `OM_METER_METER5` WHERE SUBJECT = 'subject1' AND WINDOWSTART >= 1609459200001 AND WINDOWEND <= 1609545600000;",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("", func(t *testing.T) {
			got, err := templateQuery(meterValuesTemplate, tt.data)
			if err != nil {
				t.Error(err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}
