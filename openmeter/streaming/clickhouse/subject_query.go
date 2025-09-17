package clickhouse

import (
	"fmt"
	"time"

	"github.com/huandu/go-sqlbuilder"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
)

type listSubjectsQuery struct {
	Database        string
	EventsTableName string
	Namespace       string
	Meter           *meterpkg.Meter
	From            *time.Time
	To              *time.Time
	Search          *string
}

func (d listSubjectsQuery) toSQL() (string, []interface{}) {
	tableName := getTableName(d.Database, d.EventsTableName)

	sb := sqlbuilder.ClickHouse.NewSelectBuilder()
	sb.Select("DISTINCT subject")
	sb.Where(sb.Equal("namespace", d.Namespace))

	// If we have a meter, we add the type filter
	if d.Meter != nil {
		sb.Where(sb.Equal("type", d.Meter.EventType))
	}

	sb.From(tableName)
	sb.OrderBy("subject")

	if d.From != nil {
		sb.Where(sb.GreaterEqualThan("time", d.From.Unix()))
	}

	if d.To != nil {
		sb.Where(sb.LessThan("time", d.To.Unix()))
	}

	if d.Search != nil && *d.Search != "" {
		sb.Where(fmt.Sprintf("positionCaseInsensitive(subject, %s) > 0", sb.Var(*d.Search)))
	}

	sql, args := sb.Build()
	return sql, args
}
