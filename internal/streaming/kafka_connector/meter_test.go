// Copyright © 2023 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kafka_connector

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/internal/models"
)

func TestMeterQueryAssert(t *testing.T) {
	data := meterTableQueryData{
		Meter: &models.Meter{
			ID:            "meter1",
			Name:          "API Network Traffic",
			ValueProperty: "$.bytes",
			Type:          "api-calls",
			Aggregation:   models.MeterAggregationSum,
			GroupBy:       []string{"$.path", "$.method"},
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
			query: "CREATE TABLE IF NOT EXISTS `OM_METER_METER1` WITH ( KAFKA_TOPIC = 'om_meter_meter1', KEY_FORMAT = 'JSON', VALUE_FORMAT = 'JSON', PARTITIONS = 1 ) AS SELECT SUBJECT, COALESCE(EXTRACTJSONFIELD(data, '$.path'), '') AS `$.path`, SUM(CAST(EXTRACTJSONFIELD(data, '$.bytes') AS DECIMAL(12, 4))) AS VALUE FROM OM_DETECTED_EVENTS_STREAM WINDOW TUMBLING ( SIZE 1 HOUR, RETENTION 365 DAYS ) WHERE ID_COUNT = 1 AND TYPE = 'api-calls' GROUP BY SUBJECT, COALESCE(EXTRACTJSONFIELD(data, '$.path'), ''), COALESCE(EXTRACTJSONFIELD(data, '$.method'), '') EMIT CHANGES;",
			match: nil,
		},
		{
			name:  "should not match if value property differs",
			data:  data,
			query: "CREATE TABLE IF NOT EXISTS `OM_METER_METER1` WITH ( KAFKA_TOPIC = 'om_meter_meter1', KEY_FORMAT = 'JSON', VALUE_FORMAT = 'JSON', PARTITIONS = 1 ) AS SELECT SUBJECT, COALESCE(EXTRACTJSONFIELD(data, '$.path'), '') AS `$.path`, SUM(CAST(EXTRACTJSONFIELD(data, '$.invalid') AS DECIMAL(12, 4))) AS VALUE FROM OM_DETECTED_EVENTS_STREAM WINDOW TUMBLING ( SIZE 1 HOUR, RETENTION 365 DAYS ) WHERE ID_COUNT = 1 AND TYPE = 'api-calls' GROUP BY SUBJECT, COALESCE(EXTRACTJSONFIELD(data, '$.path'), ''), COALESCE(EXTRACTJSONFIELD(data, '$.method'), '') EMIT CHANGES;",
			match: fmt.Errorf("meter value property mismatch, old: $.invalid, new: $.bytes"),
		},
		{
			name:  "should not match if group by length differs",
			data:  data,
			query: "CREATE TABLE IF NOT EXISTS `OM_METER_METER1` WITH ( KAFKA_TOPIC = 'om_meter_meter1', KEY_FORMAT = 'JSON', VALUE_FORMAT = 'JSON', PARTITIONS = 1 ) AS SELECT SUBJECT, COALESCE(EXTRACTJSONFIELD(data, '$.path'), '') AS `$.path`, SUM(CAST(EXTRACTJSONFIELD(data, '$.bytes') AS DECIMAL(12, 4))) AS VALUE FROM OM_DETECTED_EVENTS_STREAM WINDOW TUMBLING ( SIZE 1 HOUR, RETENTION 365 DAYS ) WHERE ID_COUNT = 1 AND TYPE = 'api-calls' GROUP BY SUBJECT, COALESCE(EXTRACTJSONFIELD(data, '$.path'), '') EMIT CHANGES;",
			match: fmt.Errorf("meter group by length mistmatch, old: 1, new: 2"),
		},
		{
			name:  "should not match if group by differs",
			data:  data,
			query: "CREATE TABLE IF NOT EXISTS `OM_METER_METER1` WITH ( KAFKA_TOPIC = 'om_meter_meter1', KEY_FORMAT = 'JSON', VALUE_FORMAT = 'JSON', PARTITIONS = 1 ) AS SELECT SUBJECT, COALESCE(EXTRACTJSONFIELD(data, '$.path'), '') AS `$.path`, SUM(CAST(EXTRACTJSONFIELD(data, '$.bytes') AS DECIMAL(12, 4))) AS VALUE FROM OM_DETECTED_EVENTS_STREAM WINDOW TUMBLING ( SIZE 1 HOUR, RETENTION 365 DAYS ) WHERE ID_COUNT = 1 AND TYPE = 'api-calls' GROUP BY SUBJECT, COALESCE(EXTRACTJSONFIELD(data, '$.invalid'), ''), COALESCE(EXTRACTJSONFIELD(data, '$.method'), '') EMIT CHANGES;",
			match: fmt.Errorf("meter group by not found: $.path"),
		},
		{
			name:  "should not match if window size differs",
			data:  data,
			query: "CREATE TABLE IF NOT EXISTS `OM_METER_METER1` WITH ( KAFKA_TOPIC = 'om_meter_meter1', KEY_FORMAT = 'JSON', VALUE_FORMAT = 'JSON', PARTITIONS = 1 ) AS SELECT SUBJECT, COALESCE(EXTRACTJSONFIELD(data, '$.path'), '') AS `$.path`, SUM(CAST(EXTRACTJSONFIELD(data, '$.bytes') AS DECIMAL(12, 4))) AS VALUE FROM OM_DETECTED_EVENTS_STREAM WINDOW TUMBLING ( SIZE 2 HOUR, RETENTION 365 DAYS ) WHERE ID_COUNT = 1 AND TYPE = 'api-calls' GROUP BY SUBJECT, COALESCE(EXTRACTJSONFIELD(data, '$.path'), ''), COALESCE(EXTRACTJSONFIELD(data, '$.method'), '') EMIT CHANGES;",
			match: fmt.Errorf("meter window size mismatch, old: 2 HOUR, new: 1 HOUR"),
		},
		{
			name:  "should not match if window retention differs",
			data:  data,
			query: "CREATE TABLE IF NOT EXISTS `OM_METER_METER1` WITH ( KAFKA_TOPIC = 'om_meter_meter1', KEY_FORMAT = 'JSON', VALUE_FORMAT = 'JSON', PARTITIONS = 1 ) AS SELECT SUBJECT, COALESCE(EXTRACTJSONFIELD(data, '$.path'), '') AS `$.path`, SUM(CAST(EXTRACTJSONFIELD(data, '$.bytes') AS DECIMAL(12, 4))) AS VALUE FROM OM_DETECTED_EVENTS_STREAM WINDOW TUMBLING ( SIZE 1 HOUR, RETENTION 1 DAYS ) WHERE ID_COUNT = 1 AND TYPE = 'api-calls' GROUP BY SUBJECT, COALESCE(EXTRACTJSONFIELD(data, '$.path'), ''), COALESCE(EXTRACTJSONFIELD(data, '$.method'), '') EMIT CHANGES;",
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
