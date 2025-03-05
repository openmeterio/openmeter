package raw_events

import (
	"fmt"

	"github.com/huandu/go-sqlbuilder"
)

type validateJsonPathQuery struct {
	jsonPath string
}

func (d validateJsonPathQuery) toSQL() (string, []interface{}, error) {
	sb := sqlbuilder.ClickHouse.NewSelectBuilder()
	sb.Select(fmt.Sprintf("JSON_VALUE('{}', '%s')", sqlbuilder.Escape(d.jsonPath)))

	sql, args := sb.Build()

	return sql, args, nil
}
