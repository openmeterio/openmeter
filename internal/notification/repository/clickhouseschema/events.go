package clickhouseschema

import (
	"fmt"
	"time"

	"github.com/huandu/go-sqlbuilder"

	"github.com/openmeterio/openmeter/internal/notification"
)

// EntityTotals is an interface for supporting WITH TOTALS for db entities stored in Clickhouse.
// It must return an instance of the entity and a list of pointers to the object fields in instance
// base on the column names provided in columns input parameter.
// It must return an error if unknown column name is provided in columns parameter.
//
//	type Entity struct {
//		Field1 string `ch:"field1"`
//		Field2 string `ch:"field2"`
//
//		TotalCount string `ch:"total_count"`
//	}
//
//	var columns = []string{"field1", "field2"}
//
//	func (e Entity) Totals(columns []string) (*Entity, []any, error) {
//		entity := Entity{}
//		fields := make([]any, 0, len(columns))
//
//		for _, column := range columns {
//			var field any
//
//			switch column {
//			case "field1":
//				field = &entity.Field1
//			case "field2":
//				field = &entity.Field2
//			case "total_count":
//				field = &entity.TotalCount
//			default:
//				return nil, nil, fmt.Errorf("unknown column type: %s", column)
//			}
//
//			fields = append(fields, field)
//		}
//
//		return &entity, fields, nil
//	}
type EntityTotals[T any] interface {
	Totals(columns []string) (*T, []any, error)
}

var _ EntityTotals[EventDBEntity] = (*EventDBEntity)(nil)

type EventDBEntity struct {
	Namespace string `db:"namespace" ch:"namespace"`
	// CreatedAt Timestamp when the notification event was created.
	CreatedAt time.Time `db:"created_at" ch:"created_at"`
	// Type of the notification Event (e.g. entitlements.balance.threshold)
	Type string `db:"type" ch:"type"`
	// ID is the unique identifier for Event.
	ID string `db:"id" ch:"id"`
	// Payload is the actual payload sent to Channel as part of the notification Event.
	Payload string `db:"payload" ch:"payload"`
	// Rule defines the notification Rule that generated this Event.
	Rule string `db:"rule" ch:"rule"`

	// FeatureID computed field which is extracted from Payload field (in JSON format).
	FeatureID string `db:"-" ch:"feature_id"`
	// FeatureID computed field which is extracted from Payload field (in JSON format).
	FeatureKey string `db:"-" ch:"feature_key"`
	// FeatureID computed field which is  extracted from Payload field (in JSON format).
	SubjectID string `db:"-" ch:"subject_id"`
	// FeatureID computed field which is  extracted from Payload field (in JSON format).
	SubjectKey string `db:"-" ch:"subject_key"`

	TotalCount uint64 `db:"-" ch:"total_count"`
}

func (e EventDBEntity) Columns() []string {
	return []string{ColumnCreatedAt, ColumnNamespace, ColumnType, ColumnID, ColumnPayload, ColumnRule}
}

func (e EventDBEntity) Totals(columns []string) (*EventDBEntity, []any, error) {
	entity := EventDBEntity{}
	fields := make([]any, 0, len(columns))

	for _, column := range columns {
		var field any

		switch column {
		case ColumnID:
			field = &entity.ID
		case ColumnNamespace:
			field = &entity.Namespace
		case ColumnCreatedAt:
			field = &entity.CreatedAt
		case ColumnType:
			field = &entity.Type
		case ColumnPayload:
			field = &entity.Payload
		case ColumnRule:
			field = &entity.Rule
		case ColumnSubjectID:
			field = &entity.SubjectID
		case ColumnSubjectKey:
			field = &entity.SubjectKey
		case ColumnFeatureID:
			field = &entity.FeatureID
		case ColumnFeatureKey:
			field = &entity.FeatureKey
		case ColumnTotalCount:
			field = &entity.TotalCount
		default:
			return nil, nil, fmt.Errorf("unknown column type: %s", column)
		}

		fields = append(fields, field)
	}

	return &entity, fields, nil
}

func (e EventDBEntity) Values(columns []string) ([]interface{}, error) {
	values := make([]any, 0, len(columns))

	for _, column := range columns {
		var value interface{}

		switch column {
		case ColumnID:
			value = e.ID
		case ColumnNamespace:
			value = e.Namespace
		case ColumnCreatedAt:
			value = e.CreatedAt
		case ColumnType:
			value = e.Type
		case ColumnPayload:
			value = e.Payload
		case ColumnRule:
			value = e.Rule
		default:
			return nil, fmt.Errorf("unknown column type: %s", column)
		}

		values = append(values, value)
	}

	return values, nil
}

type EventsTable struct {
	tableStruct  *sqlbuilder.Struct
	queryBuilder sqlbuilder.Flavor
	databaseName string
	tableName    string
}

func (t EventsTable) Name() string {
	return fmt.Sprintf("%s.%s", t.databaseName, t.tableName)
}

func (t EventsTable) CreateTable() SQLQuery {
	query := t.queryBuilder.NewCreateTableBuilder().
		CreateTable(t.tableName).
		IfNotExists().
		Define(ColumnCreatedAt, "DateTime").
		Define(ColumnNamespace, "String").
		Define(ColumnType, "LowCardinality(String)").
		Define(ColumnID, "String").
		Define(ColumnPayload, "String").
		Define(ColumnRule, "String").
		SQL("ENGINE = MergeTree").
		SQL(fmt.Sprintf("PARTITION BY toYYYYMM(%s)", ColumnCreatedAt)).
		SQL(fmt.Sprintf("ORDER BY (%s, %s, %s, %s)", ColumnCreatedAt, ColumnNamespace, ColumnType, ColumnID))

	return SQLQueryFromBuilder(query)
}

func (t EventsTable) ListEvents(params notification.ListEventsInput) SQLQuery {
	//
	//	Inner Query
	//

	innerQuery := t.tableStruct.SelectFrom(t.tableName)

	var features []interface{}
	if len(params.FeatureFilter) > 0 {
		innerQuery = innerQuery.SQL(innerQuery.As(", JSON_VALUE(payload, '$.data.feature.id')", ColumnFeatureID)).
			SQL(innerQuery.As(", JSON_VALUE(payload, '$.data.feature.key')", ColumnFeatureKey))

		features = make([]interface{}, 0, len(params.FeatureFilter))
		for _, feature := range params.FeatureFilter {
			features = append(features, feature)
		}
	}

	var subjects []interface{}
	if len(params.SubjectFilter) > 0 {
		innerQuery = innerQuery.SQL(innerQuery.As(", JSON_VALUE(payload, '$.data.subject.id')", ColumnSubjectID)).
			SQL(innerQuery.As(", JSON_VALUE(payload, '$.data.subject.key')", ColumnSubjectKey))

		subjects = make([]interface{}, 0, len(params.SubjectFilter))
		for _, subject := range params.SubjectFilter {
			subjects = append(subjects, subject)
		}
	}

	var namespaces []interface{}
	if len(params.Namespaces) > 0 {
		namespaces = make([]interface{}, 0, len(params.Namespaces))
		for _, namespace := range params.Namespaces {
			namespaces = append(namespaces, namespace)
		}
	}

	if features != nil {
		innerQuery = innerQuery.Where(
			innerQuery.Or(
				innerQuery.In(ColumnFeatureID, features...),
				innerQuery.In(ColumnFeatureKey, features...),
			),
		)
	}

	if subjects != nil {
		innerQuery = innerQuery.Where(
			innerQuery.Or(
				innerQuery.In(ColumnSubjectID, subjects...),
				innerQuery.In(ColumnSubjectKey, subjects...),
			),
		)
	}

	if namespaces != nil {
		innerQuery = innerQuery.Where(innerQuery.In(ColumnNamespace, namespaces...))
	}

	if !params.From.IsZero() {
		innerQuery = innerQuery.Where(innerQuery.GE(ColumnCreatedAt, params.From))
	}

	if !params.To.IsZero() {
		innerQuery = innerQuery.Where(innerQuery.LTE(ColumnCreatedAt, params.To))
	}

	innerQuerySQL, innerQueryArgs := innerQuery.Build()

	//
	//	Paged Query
	//
	// SELECT *, COUNT(*) as total_count FROM (<INNER_QUERY>) GROUP BY ALL WITH TOTALS [LIMIT X OFFSET Y]

	query := t.queryBuilder.NewSelectBuilder()

	query = query.Select("*", query.As("COUNT(*)", "total_count")).
		From(fmt.Sprintf("(%s)", innerQuerySQL)).
		GroupBy("ALL").SQL("WITH TOTALS")

	if !params.Page.IsZero() {
		query = query.Limit(params.PageSize)

		if params.PageNumber > 0 {
			query = query.Offset(params.PageSize * (params.PageNumber - 1))
		}
	}

	pagedQuerySQL, _ := query.Build()

	return SQLQuery{
		SQL:  pagedQuerySQL,
		Args: innerQueryArgs,
	}
}

func (t EventsTable) GetEvent(params notification.GetEventInput) (SQLQuery, error) {
	createdAt, err := notification.GetCreatedAtFromEventID(params.ID)
	if err != nil {
		return SQLQuery{}, err
	}

	query := t.tableStruct.For(t.queryBuilder).SelectFrom(t.tableName)

	query = query.Where(query.Equal(ColumnNamespace, params.Namespace)).
		Where(query.Equal(ColumnCreatedAt, createdAt.Truncate(time.Second).UTC())).
		Where(query.Equal(ColumnID, params.ID))

	return SQLQueryFromBuilder(query), nil
}

func (t EventsTable) CreateEvent(params notification.CreateEventInput) (SQLQuery, *EventDBEntity, error) {
	entity, err := CreateEventInputToDBEntity(params)
	if err != nil {
		return SQLQuery{}, nil, fmt.Errorf("failed to create event entity: %w", err)
	}

	columns := entity.Columns()

	values, err := entity.Values(columns)
	if err != nil {
		return SQLQuery{}, nil, fmt.Errorf("failed to get values from event entity: %w", err)
	}

	query := t.tableStruct.InsertInto(t.tableName).
		Cols(columns...).
		Values(values...)

	return SQLQueryFromBuilder(query), entity, nil
}

func NewEventsTable(database, table string) *EventsTable {
	return &EventsTable{
		tableStruct:  sqlbuilder.NewStruct(new(EventDBEntity)).For(sqlbuilder.ClickHouse),
		queryBuilder: sqlbuilder.ClickHouse,
		databaseName: database,
		tableName:    table,
	}
}
