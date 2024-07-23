package clickhouseschema

import (
	"fmt"
	"time"

	"github.com/huandu/go-sqlbuilder"

	"github.com/openmeterio/openmeter/internal/notification"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

var _ EntityTotals[DeliveryStatusDBEntity] = (*DeliveryStatusDBEntity)(nil)

type DeliveryStatusDBEntity struct {
	Namespace string `db:"namespace" ch:"namespace"`
	// Timestamp defines the time the delivery status report was created.
	Timestamp time.Time `db:"timestamp" ch:"timestamp"`
	// ID is the unique identifier for Event.
	EventID string `db:"event_id" ch:"event_id"`
	// Rule defines the notification Rule that generated this Event.
	ChannelID string `db:"channel_id" ch:"channel_id"`
	// State defines the state of teh Event delivery
	State string `db:"state" ch:"state"`

	TotalCount uint64 `db:"-" ch:"total_count"`
}

func (e DeliveryStatusDBEntity) Columns() []string {
	return []string{ColumnNamespace, ColumnTimestamp, ColumnEventID, ColumnChannelID, ColumnState}
}

func (e DeliveryStatusDBEntity) Totals(columns []string) (*DeliveryStatusDBEntity, []any, error) {
	entity := DeliveryStatusDBEntity{}
	fields := make([]any, 0, len(columns))

	for _, column := range columns {
		var field any

		switch column {
		case ColumnNamespace:
			field = &entity.Namespace
		case ColumnTimestamp:
			field = &entity.Timestamp
		case ColumnEventID:
			field = &entity.EventID
		case ColumnChannelID:
			field = &entity.ChannelID
		case ColumnState:
			field = &entity.State
		case ColumnTotalCount:
			field = &entity.TotalCount
		default:
			return nil, nil, fmt.Errorf("unknown column type: %s", column)
		}

		fields = append(fields, field)
	}

	return &entity, fields, nil
}

func (e DeliveryStatusDBEntity) Values(columns []string) ([]interface{}, error) {
	values := make([]any, 0, len(columns))

	for _, column := range columns {
		var value interface{}

		switch column {
		case ColumnNamespace:
			value = e.Namespace
		case ColumnTimestamp:
			value = e.Timestamp
		case ColumnEventID:
			value = e.EventID
		case ColumnChannelID:
			value = e.ChannelID
		case ColumnState:
			value = e.State
		case ColumnTotalCount:
			value = e.TotalCount
		default:
			return nil, fmt.Errorf("unknown column type: %s", column)
		}

		values = append(values, value)
	}

	return values, nil
}

type DeliveryStatusTable struct {
	tableStruct  *sqlbuilder.Struct
	queryBuilder sqlbuilder.Flavor
	databaseName string
	tableName    string
}

func (t DeliveryStatusTable) Name() string {
	return fmt.Sprintf("%s.%s", t.databaseName, t.tableName)
}

func (t DeliveryStatusTable) CreateTable() string {
	table := t.queryBuilder.NewCreateTableBuilder().
		CreateTable(t.tableName).
		IfNotExists().
		Define(ColumnTimestamp, "DateTime").
		Define(ColumnNamespace, "String").
		Define(ColumnState, "LowCardinality(String)").
		Define(ColumnEventID, "String").
		Define(ColumnChannelID, "String").
		SQL("ENGINE = MergeTree").
		SQL(fmt.Sprintf("PARTITION BY toYYYYMMDD(%s)", ColumnTimestamp)).
		SQL(fmt.Sprintf("ORDER BY (%s, %s, %s, %s)", ColumnTimestamp, ColumnNamespace, ColumnState, ColumnEventID))

	sql, _ := table.Build()

	return sql
}

func (t DeliveryStatusTable) ListDeliveryStatus(params notification.ListEventsDeliveryStatusInput) SQLQuery {
	//
	//	Inner Query
	//

	innerQuery := t.tableStruct.SelectFrom(t.tableName)

	var namespaces []interface{}
	if len(params.Namespaces) > 0 {
		namespaces = make([]interface{}, 0, len(params.Namespaces))
		for _, namespace := range params.Namespaces {
			namespaces = append(namespaces, namespace)
		}
	}

	if namespaces != nil {
		innerQuery = innerQuery.Where(innerQuery.In(ColumnNamespace, namespaces...))
	}

	if len(params.EventIDs) > 0 {
		eventIDs := slicesx.Map(params.EventIDs, func(id string) interface{} {
			return id
		})

		innerQuery = innerQuery.Where(innerQuery.In(ColumnEventID, eventIDs...))
	}

	if !params.From.IsZero() {
		innerQuery = innerQuery.Where(innerQuery.GE(ColumnCreatedAt, params.From.Truncate(time.Second).UTC()))
	}

	if !params.To.IsZero() {
		innerQuery = innerQuery.Where(innerQuery.LTE(ColumnCreatedAt, params.To.Truncate(time.Second).UTC()))
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
		query = query.Limit(params.Page.Limit())

		if params.PageNumber > 0 {
			query = query.Offset(params.Page.Offset())
		}
	}

	pagedQuerySQL, _ := query.Build()

	return SQLQuery{
		SQL:  pagedQuerySQL,
		Args: innerQueryArgs,
	}
}

func (t DeliveryStatusTable) GetDeliveryStatus(params notification.GetEventDeliveryStatusInput) (SQLQuery, error) {
	createdAt, err := notification.GetCreatedAtFromEventID(params.EventID)
	if err != nil {
		return SQLQuery{}, err
	}

	query := t.tableStruct.For(t.queryBuilder).SelectFrom(t.tableName)

	query = query.Where(query.Equal(ColumnNamespace, params.Namespace)).
		Where(query.GE(ColumnCreatedAt, createdAt.Truncate(time.Second).UTC())).
		Where(query.Equal(ColumnEventID, params.EventID))

	return SQLQueryFromBuilder(query), nil
}

func (t DeliveryStatusTable) CreateDeliveryStatus(params notification.CreateEventDeliveryStatusInput) (SQLQuery, *DeliveryStatusDBEntity, error) {
	entity, err := CreateEventDeliveryStatusInputToDBEntity(params)
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

func NewDeliveryStatusTable(database, table string) *DeliveryStatusTable {
	return &DeliveryStatusTable{
		tableStruct:  sqlbuilder.NewStruct(new(DeliveryStatusDBEntity)).For(sqlbuilder.ClickHouse),
		queryBuilder: sqlbuilder.ClickHouse,
		databaseName: database,
		tableName:    table,
	}
}
