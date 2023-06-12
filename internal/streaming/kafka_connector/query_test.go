package kafka_connector

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/internal/models"
	. "github.com/openmeterio/openmeter/internal/streaming"
)

func TestDetectedEventsTableQuery(t *testing.T) {
	tests := []struct {
		data detectedEventsTableQueryData
		want string
	}{
		{
			data: detectedEventsTableQueryData{
				Retention:  32,
				Partitions: 100,
			},
			want: "CREATE TABLE IF NOT EXISTS OM_DETECTED_EVENTS WITH ( KAFKA_TOPIC = 'om_detected_events', KEY_FORMAT = 'JSON_SR', VALUE_FORMAT = 'JSON_SR', PARTITIONS = 100 ) AS SELECT ID AS KEY1, SOURCE AS KEY2, AS_VALUE(ID) AS ID, AS_VALUE(SOURCE) AS SOURCE, EARLIEST_BY_OFFSET(TIME) AS TIME, EARLIEST_BY_OFFSET(TYPE) AS TYPE, EARLIEST_BY_OFFSET(SUBJECT) AS SUBJECT, EARLIEST_BY_OFFSET(TIME) AS STRING, EARLIEST_BY_OFFSET(DATA) AS DATA, COUNT(ID) as ID_COUNT FROM OM_EVENTS WINDOW TUMBLING ( SIZE 32 DAYS, RETENTION 32 DAYS ) GROUP BY ID, SOURCE;",
		},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got, err := Execute(detectedEventsTableQueryTemplate, tt.data)
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
				Partitions:    1,
				KeySchemaId:   1,
				ValueSchemaId: 1,
			},
			want: "CREATE STREAM IF NOT EXISTS OM_EVENTS WITH ( KAFKA_TOPIC = 'om_events', KEY_FORMAT = 'JSON_SR', VALUE_FORMAT = 'JSON_SR', PARTITIONS = 1, KEY_SCHEMA_ID = 1, VALUE_SCHEMA_ID = 1 );",
		},
		{
			data: cloudEventsStreamQueryData{
				Topic:         "foo",
				Partitions:    2,
				KeySchemaId:   2,
				ValueSchemaId: 2,
			},
			want: "CREATE STREAM IF NOT EXISTS OM_EVENTS WITH ( KAFKA_TOPIC = 'foo', KEY_FORMAT = 'JSON_SR', VALUE_FORMAT = 'JSON_SR', PARTITIONS = 2, KEY_SCHEMA_ID = 2, VALUE_SCHEMA_ID = 2 );",
		},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got, err := Execute(cloudEventsStreamQueryTemplate, tt.data)
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
					ID:            "meter1",
					Name:          "API Network Traffic",
					ValueProperty: "$.bytes",
					Type:          "api-calls",
					Aggregation:   models.MeterAggregationSum,
					GroupBy:       []string{"$.path"},
					WindowSize:    models.WindowSizeHour,
				},
				WindowRetention: "365 DAYS",
				Partitions:      1,
			},
			want: "CREATE TABLE IF NOT EXISTS `OM_METER_METER1` WITH ( KAFKA_TOPIC = 'om_meter_meter1', KEY_FORMAT = 'JSON_SR', VALUE_FORMAT = 'JSON_SR', PARTITIONS = 1 ) AS SELECT SUBJECT AS KEY1, AS_VALUE(SUBJECT) AS SUBJECT, WINDOWSTART AS WINDOWSTART_TS, WINDOWEND AS WINDOWEND_TS, COALESCE(EXTRACTJSONFIELD(data, '$.path'), '') AS `$.path_KEY`, AS_VALUE(COALESCE(EXTRACTJSONFIELD(data, '$.path'), '')) AS `$.path`, SUM(CAST(EXTRACTJSONFIELD(data, '$.bytes') AS DECIMAL(12, 4))) AS VALUE FROM OM_DETECTED_EVENTS_STREAM WINDOW TUMBLING ( SIZE 1 HOUR, RETENTION 365 DAYS ) WHERE ID_COUNT = 1 AND TYPE = 'api-calls' GROUP BY SUBJECT, COALESCE(EXTRACTJSONFIELD(data, '$.path'), '') EMIT CHANGES;",
		},
		{
			data: meterTableQueryData{
				Meter: &models.Meter{
					ID:          "meter2",
					Name:        "API Calls",
					Type:        "api-calls",
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
					ID:            "meter2",
					Name:          "API Calls",
					Type:          "api-calls",
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
					ID:            "meter3",
					Name:          "API call count by path",
					Type:          "api-calls",
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
		t.Run("", func(t *testing.T) {
			got, err := Execute(meterTableQueryTemplate, tt.data)
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
					ID: "meter1",
				},
				GetValuesParams: &GetValuesParams{
					Subject: &subject,
				},
			},
			want: "SELECT * FROM `OM_METER_METER1` WHERE SUBJECT = 'subject1';",
		},
		{
			data: meterValuesData{
				Meter: &models.Meter{
					ID: "meter2",
				},
				GetValuesParams: &GetValuesParams{
					Subject: &subject,
					From:    &from,
				},
			},
			want: "SELECT * FROM `OM_METER_METER2` WHERE SUBJECT = 'subject1' AND WINDOWSTART >= 1609459200001;",
		},
		{
			data: meterValuesData{
				Meter: &models.Meter{
					ID: "meter3",
				},
				GetValuesParams: &GetValuesParams{},
			},
			want: "SELECT * FROM `OM_METER_METER3`;",
		},
		{
			data: meterValuesData{
				Meter: &models.Meter{
					ID: "meter4",
				},
				GetValuesParams: &GetValuesParams{
					Subject: &subject,
					To:      &to,
				},
			},
			want: "SELECT * FROM `OM_METER_METER4` WHERE SUBJECT = 'subject1' AND WINDOWEND <= 1609545600000;",
		},
		{
			data: meterValuesData{
				Meter: &models.Meter{
					ID: "meter5",
				},
				GetValuesParams: &GetValuesParams{
					Subject: &subject,
					From:    &from,
					To:      &to,
				},
			},
			want: "SELECT * FROM `OM_METER_METER5` WHERE SUBJECT = 'subject1' AND WINDOWSTART >= 1609459200001 AND WINDOWEND <= 1609545600000;",
		},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got, err := Execute(meterValuesTemplate, tt.data)
			if err != nil {
				t.Error(err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}
