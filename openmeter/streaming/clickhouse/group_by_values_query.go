package clickhouse

import (
	"fmt"
	"time"

	"github.com/huandu/go-sqlbuilder"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
)

type listGroupByValuesQuery struct {
	Database        string
	EventsTableName string
	Namespace       string
	Meter           meterpkg.Meter
	GroupByKey      string
	From            *time.Time
	To              *time.Time
	Search          *string
}

func (d listGroupByValuesQuery) toSQL() (string, []interface{}) {
	tableName := getTableName(d.Database, d.EventsTableName)

	// Helper function to get the column
	getColumn := columnFactory(d.EventsTableName)

	groupByJSONPath := d.Meter.GroupBy[d.GroupByKey]

	sb := sqlbuilder.ClickHouse.NewSelectBuilder()
	sb.Select(sb.As(fmt.Sprintf("DISTINCT JSON_VALUE(%s, %s)", getColumn("data"), sb.Var(groupByJSONPath)), "group_by_values"))
	sb.Where(sb.Equal("namespace", d.Namespace), sb.Equal("type", d.Meter.EventType))

	sb.From(tableName)
	sb.OrderBy("group_by_values")

	if d.From != nil {
		sb.Where(sb.GreaterEqualThan("time", d.From.Unix()))
	}

	if d.To != nil {
		sb.Where(sb.LessThan("time", d.To.Unix()))
	}

	if d.Search != nil && *d.Search != "" {
		sb.Where(fmt.Sprintf("positionCaseInsensitive(group_by_values, %s) > 0", sb.Var(*d.Search)))
	}

	sql, args := sb.Build()
	return sql, args
}
