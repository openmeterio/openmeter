package ksqldb_connector

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/thmeitz/ksqldb-go"
	"golang.org/x/exp/slog"

	"github.com/openmeterio/openmeter/internal/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
)

// support json path returning a single value
// syntax: https://github.com/json-path/JsonPath
var jsonPathRe = regexp.MustCompile(`([$][._\[\]0-9a-zA-Z]+)+`)
var aggregationRe = regexp.MustCompile(`(COUNT|MIN|MAX|SUM|DISTINCT_COUNT)\(`)

// TODO: support ` character and replace .{1} with `
var groupByRe = regexp.MustCompile(`AS_VALUE\(COALESCE\(EXTRACTJSONFIELD\([A-Za-z_0-1.]*` + "`" + `data` + "`" + `, '([$][._\[\]0-9a-zA-Z]+)'\), ''\)\) ?A?S? ` + "`" + `([A-Za-z0-9_-]+)` + "`" + `,?`)
var windowSizeRe = regexp.MustCompile(`SIZE (?:\d+ [DAY|HOUR|MINUTE|SECOND|MILLISECOND]{1,})`)
var windowRetentionRe = regexp.MustCompile(`RETENTION (?:\d+ [DAY|HOUR]{1,})`)

type MeterTable struct {
	Aggregation     models.MeterAggregation
	WindowSize      models.WindowSize
	WindowRetention string
	ValueProperty   string
	GroupBy         map[string]string
}

func GetTableQuery(data meterTableQueryData) (string, error) {
	return streaming.TemplateQuery(meterTableQueryTemplate, data)
}

func GetTableDescribeQuery(namespace string, meterSlug string) (string, error) {
	return streaming.TemplateQuery(meterTableDescribeQueryTemplate, meterTableDescribeQueryData{
		Namespace: namespace,
		Slug:      meterSlug,
	})
}

func DeleteTableQuery(data deleteMeterTableQueryData) (string, error) {
	return streaming.TemplateQuery(deleteMeterTableQueryTemplate, data)
}

func GetTableValuesQuery(namespace string, meterSlug string, groupBy []string, params *streaming.QueryParams) (string, error) {
	return streaming.TemplateQuery(meterValuesTemplate, meterValuesData{
		Namespace:   namespace,
		Slug:        meterSlug,
		GroupBy:     groupBy,
		QueryParams: params,
	})
}

func NewMeterValues(header ksqldb.Header, payload ksqldb.Payload) ([]*models.MeterValue, error) {
	values := make([]*models.MeterValue, 0, len(payload))

	for _, row := range payload {
		value := &models.MeterValue{
			GroupBy: make(map[string]string),
		}
		for idx, column := range header.Columns {
			// KSQL returns them in uppsercase even if we use lowercase in the query
			if column.Name == "windowstart" {
				value.WindowStart = time.UnixMilli(int64(row[idx].(float64))).UTC()
			} else if column.Name == "windowend" {
				value.WindowEnd = time.UnixMilli(int64(row[idx].(float64))).UTC()
			} else if column.Name == "subject" {
				value.Subject = row[idx].(string)
			} else if column.Name == "value" {
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
	table, err := ParseMeterTable(query)
	if err != nil {
		return err
	}

	slog.Debug("ksqlDB meter assert", "query", query)

	if table.ValueProperty != data.Meter.ValueProperty {
		return fmt.Errorf("meter value property mismatch, old: %s, new: %s", table.ValueProperty, data.Meter.ValueProperty)
	}
	if len(table.GroupBy) != len(data.Meter.GroupBy) {
		return fmt.Errorf("meter group by length mistmatch, old: %d, new: %d", len(table.GroupBy), len(data.Meter.GroupBy))
	}

	for _, g1 := range data.Meter.GroupBy {
		contains := false
		for _, g2 := range table.GroupBy {
			if g1 == g2 {
				contains = true
			}
		}
		if !contains {
			return fmt.Errorf("meter group by not found: %s", g1)
		}
	}

	// We trim tailing S in case config would be in plural DAY vs DAYS
	if fmt.Sprintf("1 %s", data.WindowSize) != strings.TrimRight(string(table.WindowSize), "S") {
		return fmt.Errorf("meter window size mismatch, old: %s, new: 1 %s", table.WindowSize, data.WindowSize)
	}
	if table.WindowRetention != strings.TrimRight(data.WindowRetention, "S") {
		return fmt.Errorf("meter window retention mismatch, old: %s, new: %s", table.WindowRetention, data.WindowRetention)
	}

	return nil
}

func ParseMeterTable(query string) (*MeterTable, error) {
	groupByTmp := groupByRe.FindAllStringSubmatch(query, -1)
	groupBy := map[string]string{}
	for _, row := range groupByTmp {
		key := strings.TrimSuffix(row[2], "`")
		groupBy[key] = row[1]
	}

	aggregation := aggregationRe.FindStringSubmatch(query)
	valueProperty := jsonPathRe.FindString(getStringInBetweenTwoString(query, aggregation[1], "value"))
	windowSize := windowSizeRe.FindString(query)
	// Go doesn't support \K to reset match after SIZE in regex so we trim it out
	windowSize = strings.TrimPrefix(windowSize, "SIZE ")
	windowRetention := windowRetentionRe.FindString(query)
	// Go doesn't support \K to reset match after RETENTION in regex so we trim it out
	windowRetention = strings.TrimPrefix(windowRetention, "RETENTION ")

	if len(aggregation) < 2 {
		return nil, fmt.Errorf("regex does not find aggregation: %s", aggregation)
	}

	agg := models.MeterAggregation("")
	if ok := agg.IsValid(aggregation[1]); !ok {
		return nil, fmt.Errorf("invalid aggregation: %s", agg)
	}

	meterTable := &MeterTable{
		Aggregation:     models.MeterAggregation(aggregation[1]),
		WindowSize:      models.WindowSize(windowSize),
		WindowRetention: windowRetention,
		GroupBy:         groupBy,
		ValueProperty:   valueProperty,
	}

	return meterTable, nil
}
