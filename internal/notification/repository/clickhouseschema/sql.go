package clickhouseschema

const (
	ColumnNamespace = "namespace"

	ColumnCreatedAt  = "created_at"
	ColumnType       = "type"
	ColumnID         = "id"
	ColumnPayload    = "payload"
	ColumnRule       = "rule"
	ColumnSubjectID  = "subject_id"
	ColumnSubjectKey = "subject_key"
	ColumnFeatureID  = "feature_id"
	ColumnFeatureKey = "feature_key"

	ColumnTimestamp = "timestamp"
	ColumnEventID   = "event_id"
	ColumnChannelID = "channel_id"
	ColumnState     = "state"

	ColumnTotalCount = "total_count"
)

type SQLQuery struct {
	SQL  string
	Args []interface{}
}

func SQLQueryFromBuilder(builder QueryBuilder) SQLQuery {
	sql, args := builder.Build()

	return SQLQuery{
		SQL:  sql,
		Args: args,
	}
}

type QueryBuilder interface {
	Build() (sql string, args []interface{})
}
