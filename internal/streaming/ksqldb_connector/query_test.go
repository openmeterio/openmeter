package ksqldb_connector

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/internal/streaming"
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
				Format:     "JSON_SR",
				Namespace:  "default",
				Topic:      "om_default_detected_events",
				Retention:  32,
				Partitions: 100,
			},
			want: "CREATE TABLE IF NOT EXISTS OM_DEFAULT_DETECTED_EVENTS WITH ( KAFKA_TOPIC = 'om_default_detected_events', KEY_FORMAT = 'JSON_SR', VALUE_FORMAT = 'JSON_SR', PARTITIONS = 100 ) AS SELECT `id` AS `key1`, `source` AS `key2`, AS_VALUE(`id`) AS `id`, EARLIEST_BY_OFFSET(`type`) AS `type`, AS_VALUE(`source`) AS `source`, EARLIEST_BY_OFFSET(`subject`) AS `subject`, EARLIEST_BY_OFFSET(`time`) AS `time`, EARLIEST_BY_OFFSET(`data`) AS `data`, COUNT(`id`) as `id_count` FROM OM_DEFAULT_EVENTS WINDOW TUMBLING ( SIZE 32 DAYS, RETENTION 32 DAYS ) GROUP BY `id`, `source`;",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("", func(t *testing.T) {
			got, err := streaming.TemplateQuery(detectedEventsTableQueryTemplate, tt.data)
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
				Format:    "JSON_SR",
				Namespace: "default",
				Topic:     "om_default_detected_events",
			},
			want: "CREATE STREAM IF NOT EXISTS OM_DEFAULT_DETECTED_EVENTS_STREAM ( `key1` STRING KEY, `key2` STRING KEY, `id` STRING, `id_count` BIGINT, `type` STRING, `source` STRING, `subject` STRING, `time` STRING, `data` STRING ) WITH ( KAFKA_TOPIC = 'om_default_detected_events', KEY_FORMAT = 'JSON_SR', VALUE_FORMAT = 'JSON_SR' );",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("", func(t *testing.T) {
			got, err := streaming.TemplateQuery(detectedEventsStreamQueryTemplate, tt.data)
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
				Format:        "JSON_SR",
				Namespace:     "default",
				Topic:         "om_default_events",
				KeySchemaId:   1,
				ValueSchemaId: 1,
			},
			want: "CREATE STREAM IF NOT EXISTS OM_DEFAULT_EVENTS WITH ( KAFKA_TOPIC = 'om_default_events', KEY_FORMAT = 'JSON_SR', VALUE_FORMAT = 'JSON_SR', KEY_SCHEMA_ID = 1, VALUE_SCHEMA_ID = 1 );",
		},
		{
			data: cloudEventsStreamQueryData{
				Format:        "JSON_SR",
				Namespace:     "default",
				Topic:         "foo",
				KeySchemaId:   2,
				ValueSchemaId: 2,
			},
			want: "CREATE STREAM IF NOT EXISTS OM_DEFAULT_EVENTS WITH ( KAFKA_TOPIC = 'foo', KEY_FORMAT = 'JSON_SR', VALUE_FORMAT = 'JSON_SR', KEY_SCHEMA_ID = 2, VALUE_SCHEMA_ID = 2 );",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("", func(t *testing.T) {
			got, err := streaming.TemplateQuery(cloudEventsStreamQueryTemplate, tt.data)
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
				Format:    "JSON_SR",
				Namespace: "default",
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
			want: "CREATE TABLE IF NOT EXISTS `OM_DEFAULT_METER_METER1` WITH ( KAFKA_TOPIC = 'om_default_meter_meter1', KEY_FORMAT = 'JSON_SR', VALUE_FORMAT = 'JSON_SR', PARTITIONS = 1 ) AS SELECT `subject` AS `key1`, AS_VALUE(`subject`) AS `subject`, windowstart AS `windowstart_ts`, windowend AS `windowend_ts`, COALESCE(EXTRACTJSONFIELD(`data`, '$.path'), '') AS `path_key`, AS_VALUE(COALESCE(EXTRACTJSONFIELD(`data`, '$.path'), '')) AS `path`, SUM(CAST(EXTRACTJSONFIELD(`data`, '$.bytes') AS DECIMAL(12, 4))) AS `value` FROM OM_DEFAULT_DETECTED_EVENTS_STREAM WINDOW TUMBLING ( SIZE 1 HOUR, RETENTION 365 DAYS ) WHERE `id_count` = 1 AND `type` = 'api-calls' GROUP BY `subject`, COALESCE(EXTRACTJSONFIELD(`data`, '$.path'), '') EMIT CHANGES;",
		},
		{
			data: meterTableQueryData{
				Format:    "JSON_SR",
				Namespace: "default",
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
			want: "CREATE TABLE IF NOT EXISTS `OM_DEFAULT_METER_METER2` WITH ( KAFKA_TOPIC = 'om_default_meter_meter2', KEY_FORMAT = 'JSON_SR', VALUE_FORMAT = 'JSON_SR', PARTITIONS = 1 ) AS SELECT `subject` AS `key1`, AS_VALUE(`subject`) AS `subject`, windowstart AS `windowstart_ts`, windowend AS `windowend_ts`, COUNT(*) AS `value` FROM OM_DEFAULT_DETECTED_EVENTS_STREAM WINDOW TUMBLING ( SIZE 1 HOUR, RETENTION 365 DAYS ) WHERE `id_count` = 1 AND `type` = 'api-calls' GROUP BY `subject` EMIT CHANGES;",
		},
		{
			data: meterTableQueryData{
				Format:    "JSON_SR",
				Namespace: "default",
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
			want: "CREATE TABLE IF NOT EXISTS `OM_DEFAULT_METER_METER2` WITH ( KAFKA_TOPIC = 'om_default_meter_meter2', KEY_FORMAT = 'JSON_SR', VALUE_FORMAT = 'JSON_SR', PARTITIONS = 1 ) AS SELECT `subject` AS `key1`, AS_VALUE(`subject`) AS `subject`, windowstart AS `windowstart_ts`, windowend AS `windowend_ts`, COUNT(EXTRACTJSONFIELD(`data`, '$.duration_ms')) AS `value` FROM OM_DEFAULT_DETECTED_EVENTS_STREAM WINDOW TUMBLING ( SIZE 1 HOUR, RETENTION 365 DAYS ) WHERE `id_count` = 1 AND `type` = 'api-calls' GROUP BY `subject` EMIT CHANGES;",
		},
		{
			data: meterTableQueryData{
				Format:    "JSON_SR",
				Namespace: "default",
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
			want: "CREATE TABLE IF NOT EXISTS `OM_DEFAULT_METER_METER3` WITH ( KAFKA_TOPIC = 'om_default_meter_meter3', KEY_FORMAT = 'JSON_SR', VALUE_FORMAT = 'JSON_SR', PARTITIONS = 1 ) AS SELECT `subject` AS `key1`, AS_VALUE(`subject`) AS `subject`, windowstart AS `windowstart_ts`, windowend AS `windowend_ts`, AVG(CAST(EXTRACTJSONFIELD(`data`, '$.duration_ms') AS DECIMAL(12, 4))) AS `value` FROM OM_DEFAULT_DETECTED_EVENTS_STREAM WINDOW TUMBLING ( SIZE 1 MINUTE, RETENTION 365 DAYS ) WHERE `id_count` = 1 AND `type` = 'api-calls' GROUP BY `subject` EMIT CHANGES;",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("", func(t *testing.T) {
			got, err := streaming.TemplateQuery(meterTableQueryTemplate, tt.data)
			if err != nil {
				t.Error(err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDeleteMeterTableQuery(t *testing.T) {
	tests := []struct {
		data deleteMeterTableQueryData
		want string
	}{
		{
			data: deleteMeterTableQueryData{
				Slug:      "meter1",
				Namespace: "default",
			},
			want: "DROP TABLE `OM_DEFAULT_METER_METER1` DELETE TOPIC;",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("", func(t *testing.T) {
			got, err := streaming.TemplateQuery(deleteMeterTableQueryTemplate, tt.data)
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
				Namespace: "default",
				Slug:      "meter1",
				GroupBy:   []string{},
				QueryParams: &QueryParams{
					Subject: &subject,
				},
			},
			want: "SELECT `subject`, `value`, windowstart as `windowstart`, windowend as `windowend` FROM `OM_DEFAULT_METER_METER1` WHERE subject = 'subject1';",
		},
		{
			data: meterValuesData{
				Namespace: "default",
				Slug:      "meter2",
				GroupBy:   []string{},
				QueryParams: &QueryParams{
					Subject: &subject,
					From:    &from,
				},
			},
			want: "SELECT `subject`, `value`, windowstart as `windowstart`, windowend as `windowend` FROM `OM_DEFAULT_METER_METER2` WHERE subject = 'subject1' AND windowstart >= 1609459200001;",
		},
		{
			data: meterValuesData{
				Namespace:   "default",
				Slug:        "meter3",
				GroupBy:     []string{},
				QueryParams: &QueryParams{},
			},
			want: "SELECT `subject`, `value`, windowstart as `windowstart`, windowend as `windowend` FROM `OM_DEFAULT_METER_METER3`;",
		},
		{
			data: meterValuesData{
				Namespace: "default",
				Slug:      "meter4",
				QueryParams: &QueryParams{
					Subject: &subject,
					To:      &to,
				},
			},
			want: "SELECT `subject`, `value`, windowstart as `windowstart`, windowend as `windowend` FROM `OM_DEFAULT_METER_METER4` WHERE subject = 'subject1' AND windowend <= 1609545600000;",
		},
		{
			data: meterValuesData{
				Namespace: "default",
				Slug:      "meter5",
				GroupBy:   []string{},
				QueryParams: &QueryParams{
					Subject: &subject,
					From:    &from,
					To:      &to,
				},
			},
			want: "SELECT `subject`, `value`, windowstart as `windowstart`, windowend as `windowend` FROM `OM_DEFAULT_METER_METER5` WHERE subject = 'subject1' AND windowstart >= 1609459200001 AND windowend <= 1609545600000;",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("", func(t *testing.T) {
			got, err := streaming.TemplateQuery(meterValuesTemplate, tt.data)
			if err != nil {
				t.Error(err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}
