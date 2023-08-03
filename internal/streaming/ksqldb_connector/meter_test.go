package ksqldb_connector

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/pkg/models"
)

func TestMeterQueryAssert(t *testing.T) {
	data := meterTableQueryData{
		Format:    "JSON_SR",
		Namespace: "default",
		Meter: &models.Meter{
			Slug:          "meter1",
			Description:   "API Network Traffic",
			ValueProperty: "$.bytes",
			EventType:     "api-calls",
			Aggregation:   models.MeterAggregationSum,
			GroupBy:       map[string]string{"path": "$.path", "method": "$.method"},
			WindowSize:    models.WindowSizeHour,
		},
		WindowRetention: "365 DAYS",
		Partitions:      1,
	}

	tests := []struct {
		name  string
		data  meterTableQueryData
		query string
		match error
	}{
		{
			name:  "should match",
			data:  data,
			query: "CREATE TABLE IF NOT EXISTS `OM_DEFAULT_METER_METER1` WITH ( KAFKA_TOPIC = 'om_default_meter_meter1', KEY_FORMAT = 'JSON_SR', VALUE_FORMAT = 'JSON_SR', PARTITIONS = 1 ) AS SELECT `subject` AS `key`, AS_VALUE(`subject`) AS `subject`, WINDOWSTART AS `windowstart_ts`, WINDOWEND AS `windowend_ts`, COALESCE(EXTRACTJSONFIELD(`data`, '$.path'), '') AS `path_KEY`, AS_VALUE(COALESCE(EXTRACTJSONFIELD(`data`, '$.path'), '')) AS `path`, COALESCE(EXTRACTJSONFIELD(`data`, '$.method'), '') AS `method_KEY`, AS_VALUE(COALESCE(EXTRACTJSONFIELD(`data`, '$.method'), '')) AS `method`, SUM(CAST(EXTRACTJSONFIELD(`data`, '$.bytes') AS DECIMAL(12, 4))) AS `value` FROM OM_DEFAULT_DETECTED_EVENTS_STREAM WINDOW TUMBLING ( SIZE 1 HOUR, RETENTION 365 DAYS ) WHERE ID_COUNT = 1 AND TYPE = 'api-calls' GROUP BY `subject``, COALESCE(EXTRACTJSONFIELD(`data`, '$.path'), ''), COALESCE(EXTRACTJSONFIELD(`data`, '$.method'), '') EMIT CHANGES;",
			match: nil,
		},
		{
			name:  "should not match if value property differs",
			data:  data,
			query: "CREATE TABLE IF NOT EXISTS `OM_DEFAULT_METER_METER1` WITH ( KAFKA_TOPIC = 'om_default_meter_meter1', KEY_FORMAT = 'JSON_SR', VALUE_FORMAT = 'JSON_SR', PARTITIONS = 1 ) AS SELECT `subject` AS `key`, AS_VALUE(`subject`) AS `subject`, WINDOWSTART AS `windowstart_ts`, WINDOWEND AS `windowend_ts`, COALESCE(EXTRACTJSONFIELD(`data`, '$.path'), '') AS `path_KEY`, AS_VALUE(COALESCE(EXTRACTJSONFIELD(`data`, '$.path'), '')) AS `path`, COALESCE(EXTRACTJSONFIELD(`data`, '$.method'), '') AS `method_KEY`, AS_VALUE(COALESCE(EXTRACTJSONFIELD(`data`, '$.method'), '')) AS `method`, SUM(CAST(EXTRACTJSONFIELD(`data`, '$.invalid') AS DECIMAL(12, 4))) AS `value` FROM OM_DEFAULT_DETECTED_EVENTS_STREAM WINDOW TUMBLING ( SIZE 1 HOUR, RETENTION 365 DAYS ) WHERE ID_COUNT = 1 AND TYPE = 'api-calls' GROUP BY `subject``, COALESCE(EXTRACTJSONFIELD(`data`, '$.path'), ''), COALESCE(EXTRACTJSONFIELD(`data`, '$.method'), '') EMIT CHANGES;",
			match: fmt.Errorf("meter value property mismatch, old: $.invalid, new: $.bytes"),
		},
		{
			name:  "should not match if group by length differs",
			data:  data,
			query: "CREATE TABLE IF NOT EXISTS `OM_DEFAULT_METER_METER1` WITH ( KAFKA_TOPIC = 'om_default_meter_meter1', KEY_FORMAT = 'JSON_SR', VALUE_FORMAT = 'JSON_SR', PARTITIONS = 1 ) AS SELECT `subject` AS `key`, AS_VALUE(`subject`) AS `subject`, WINDOWSTART AS `windowstart_ts`, WINDOWEND AS `windowend_ts`, COALESCE(EXTRACTJSONFIELD(`data`, '$.path'), '') AS `path_KEY`, AS_VALUE(COALESCE(EXTRACTJSONFIELD(`data`, '$.path'), '')) AS `path`, SUM(CAST(EXTRACTJSONFIELD(`data`, '$.bytes') AS DECIMAL(12, 4))) AS `value` FROM OM_DEFAULT_DETECTED_EVENTS_STREAM WINDOW TUMBLING ( SIZE 1 HOUR, RETENTION 365 DAYS ) WHERE ID_COUNT = 1 AND TYPE = 'api-calls' GROUP BY `subject``, COALESCE(EXTRACTJSONFIELD(`data`, '$.path'), '') EMIT CHANGES;",
			match: fmt.Errorf("meter group by length mistmatch, old: 1, new: 2"),
		},
		{
			name:  "should not match if group by differs",
			data:  data,
			query: "CREATE TABLE IF NOT EXISTS `OM_DEFAULT_METER_METER1` WITH ( KAFKA_TOPIC = 'om_default_meter_meter1', KEY_FORMAT = 'JSON_SR', VALUE_FORMAT = 'JSON_SR', PARTITIONS = 1 ) AS SELECT `subject` AS `key`, AS_VALUE(`subject`) AS `subject`, WINDOWSTART AS `windowstart_ts`, WINDOWEND AS `windowend_ts`, COALESCE(EXTRACTJSONFIELD(`data`, '$.foo'), '') AS `foo_KEY`, AS_VALUE(COALESCE(EXTRACTJSONFIELD(`data`, '$.foo'), '')) AS `foo`, COALESCE(EXTRACTJSONFIELD(`data`, '$.method'), '') AS `method_KEY`, AS_VALUE(COALESCE(EXTRACTJSONFIELD(`data`, '$.method'), '')) AS `method`, SUM(CAST(EXTRACTJSONFIELD(`data`, '$.bytes') AS DECIMAL(12, 4))) AS `value` FROM OM_DEFAULT_DETECTED_EVENTS_STREAM WINDOW TUMBLING ( SIZE 1 HOUR, RETENTION 365 DAYS ) WHERE ID_COUNT = 1 AND TYPE = 'api-calls' GROUP BY `subject``, COALESCE(EXTRACTJSONFIELD(`data`, '$.invalid'), ''), COALESCE(EXTRACTJSONFIELD(`data`, '$.method'), '') EMIT CHANGES;",
			match: fmt.Errorf("meter group by not found: $.path"),
		},
		{
			name:  "should not match if window size differs",
			data:  data,
			query: "CREATE TABLE IF NOT EXISTS `OM_DEFAULT_METER_METER1` WITH ( KAFKA_TOPIC = 'om_default_meter_meter1', KEY_FORMAT = 'JSON_SR', VALUE_FORMAT = 'JSON_SR', PARTITIONS = 1 ) AS SELECT `subject` AS `key`, AS_VALUE(`subject`) AS `subject`, WINDOWSTART AS `windowstart_ts`, WINDOWEND AS `windowend_ts`, COALESCE(EXTRACTJSONFIELD(`data`, '$.path'), '') AS `path_KEY`, AS_VALUE(COALESCE(EXTRACTJSONFIELD(`data`, '$.path'), '')) AS `path`, COALESCE(EXTRACTJSONFIELD(`data`, '$.method'), '') AS `method_KEY`, AS_VALUE(COALESCE(EXTRACTJSONFIELD(`data`, '$.method'), '')) AS `method`, SUM(CAST(EXTRACTJSONFIELD(`data`, '$.bytes') AS DECIMAL(12, 4))) AS `value` FROM OM_DEFAULT_DETECTED_EVENTS_STREAM WINDOW TUMBLING ( SIZE 2 HOUR, RETENTION 365 DAYS ) WHERE ID_COUNT = 1 AND TYPE = 'api-calls' GROUP BY `subject``, COALESCE(EXTRACTJSONFIELD(`data`, '$.path'), ''), COALESCE(EXTRACTJSONFIELD(`data`, '$.method'), '') EMIT CHANGES;",
			match: fmt.Errorf("meter window size mismatch, old: 2 HOUR, new: 1 HOUR"),
		},
		{
			name:  "should not match if window retention differs",
			data:  data,
			query: "CREATE TABLE IF NOT EXISTS `OM_DEFAULT_METER_METER1` WITH ( KAFKA_TOPIC = 'om_default_meter_meter1', KEY_FORMAT = 'JSON_SR', VALUE_FORMAT = 'JSON_SR', PARTITIONS = 1 ) AS SELECT `subject` AS `key`, AS_VALUE(`subject`) AS `subject`, WINDOWSTART AS `windowstart_ts`, WINDOWEND AS `windowend_ts`, COALESCE(EXTRACTJSONFIELD(`data`, '$.path'), '') AS `path_KEY`, AS_VALUE(COALESCE(EXTRACTJSONFIELD(`data`, '$.path'), '')) AS `path`, COALESCE(EXTRACTJSONFIELD(`data`, '$.method'), '') AS `method_KEY`, AS_VALUE(COALESCE(EXTRACTJSONFIELD(`data`, '$.method'), '')) AS `method`, SUM(CAST(EXTRACTJSONFIELD(`data`, '$.bytes') AS DECIMAL(12, 4))) AS `value` FROM OM_DEFAULT_DETECTED_EVENTS_STREAM WINDOW TUMBLING ( SIZE 1 HOUR, RETENTION 1 DAYS ) WHERE ID_COUNT = 1 AND TYPE = 'api-calls' GROUP BY `subject``, COALESCE(EXTRACTJSONFIELD(`data`, '$.path'), ''), COALESCE(EXTRACTJSONFIELD(`data`, '$.method'), '') EMIT CHANGES;",
			match: fmt.Errorf("meter window retention mismatch, old: 1 DAY, new: 365 DAYS"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := MeterQueryAssert(tt.query, tt.data)

			assert.Equal(t, tt.match, got)
		})
	}
}
