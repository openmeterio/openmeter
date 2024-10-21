package clickhouse_connector_parse

import "github.com/openmeterio/openmeter/pkg/models"

type MeterView struct {
	Slug        string
	Aggregation models.MeterAggregation
	GroupBy     []string
}
