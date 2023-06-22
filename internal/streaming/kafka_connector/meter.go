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
	"regexp"
	"strings"
	"time"

	"github.com/thmeitz/ksqldb-go"
	"golang.org/x/exp/slog"

	. "github.com/openmeterio/openmeter/internal/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
)

func GetTableQuery(data meterTableQueryData) (string, error) {
	return templateQuery(meterTableQueryTemplate, data)
}

func GetTableDescribeQuery(meter *models.Meter) (string, error) {
	return templateQuery(meterTableDescribeQueryTemplate, meterTableDescribeQueryData{
		Meter: meter,
	})
}

func GetTableValuesQuery(meter *models.Meter, params *GetValuesParams) (string, error) {
	return templateQuery(meterValuesTemplate, meterValuesData{
		Meter:           meter,
		GetValuesParams: params,
	})
}

func NewMeterValues(header ksqldb.Header, payload ksqldb.Payload) ([]*models.MeterValue, error) {
	values := make([]*models.MeterValue, 0, len(payload))

	for _, row := range payload {
		value := &models.MeterValue{
			GroupBy: make(map[string]string),
		}
		for idx, column := range header.Columns {
			if column.Name == "WINDOWSTART" {
				value.WindowStart = time.UnixMilli(int64(row[idx].(float64))).UTC()
			} else if column.Name == "WINDOWEND" {
				value.WindowEnd = time.UnixMilli(int64(row[idx].(float64))).UTC()
			} else if column.Name == "SUBJECT" {
				value.Subject = row[idx].(string)
			} else if column.Name == "VALUE" {
				value.Value = row[idx].(float64)
			} else {
				value.GroupBy[column.Name] = fmt.Sprintf("%s", row[idx])
			}
		}

		values = append(values, value)
	}

	return values, nil
}

func MeterQueryAssert(query string, data meterTableQueryData) error {
	// support json path returning a single value
	// syntax: https://github.com/json-path/JsonPath
	jsonRe := regexp.MustCompile(`([$][._\[\]0-9a-zA-Z]+)+`)
	groupBy := jsonRe.FindAllString(getStringInBetweenTwoString(query, "GROUP BY", "EMIT CHANGES"), -1)
	valueProperty := jsonRe.FindString(getStringInBetweenTwoString(query, string(data.Meter.Aggregation), "VALUE"))
	windowSizeRe := regexp.MustCompile(`SIZE (?:\d+ [DAY|HOUR|MINUTE|SECOND|MILLISECOND]{1,})`)
	windowSize := windowSizeRe.FindString(query)
	// Go doesn't support \K to reset match after SIZE in regex so we trim it out
	windowSize = strings.TrimPrefix(windowSize, "SIZE ")
	windowRetentionRe := regexp.MustCompile(`RETENTION (?:\d+ [DAY|HOUR]{1,})`)
	windowRetention := windowRetentionRe.FindString(query)
	// Go doesn't support \K to reset match after RETENTION in regex so we trim it out
	windowRetention = strings.TrimPrefix(windowRetention, "RETENTION ")

	slog.Debug("ksqlDB meter assert", "query", query)

	if valueProperty != data.Meter.ValueProperty {
		return fmt.Errorf("meter value property mismatch, old: %s, new: %s", valueProperty, data.Meter.ValueProperty)
	}
	if len(groupBy) != len(data.Meter.GroupBy) {
		return fmt.Errorf("meter group by length mistmatch, old: %d, new: %d", len(groupBy), len(data.Meter.GroupBy))
	}

	for _, g1 := range data.Meter.GroupBy {
		contains := false
		for _, g2 := range groupBy {
			if g1 == g2 {
				contains = true
			}
		}
		if !contains {
			return fmt.Errorf("meter group by not found: %s", g1)
		}
	}

	// We trim tailing S in case config would be in plural DAY vs DAYS
	if fmt.Sprintf("1 %s", data.WindowSize) != strings.TrimRight(windowSize, "S") {
		return fmt.Errorf("meter window size mismatch, old: %s, new: 1 %s", windowSize, data.WindowSize)
	}
	if windowRetention != strings.TrimRight(data.WindowRetention, "S") {
		return fmt.Errorf("meter window retention mismatch, old: %s, new: %s", windowRetention, data.WindowRetention)
	}

	return nil
}
