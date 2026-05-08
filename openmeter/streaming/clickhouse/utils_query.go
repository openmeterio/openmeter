package clickhouse

import "github.com/huandu/go-sqlbuilder"

type validateJsonPathQuery struct {
	jsonPath string
}

// See: https://github.com/huandu/go-sqlbuilder#freestyle-builder
func (d validateJsonPathQuery) toSQL() (string, []interface{}) {
	return sqlbuilder.Buildf("SELECT JSON_VALUE('{}', %v)", d.jsonPath).
		BuildWithFlavor(sqlbuilder.ClickHouse)
}
