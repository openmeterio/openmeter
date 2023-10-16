package sink

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/huandu/go-sqlbuilder"

	"github.com/openmeterio/openmeter/internal/ingest/kafkaingest/serializer"
	"github.com/openmeterio/openmeter/internal/streaming/clickhouse_connector"
	"github.com/openmeterio/openmeter/pkg/models"
)

var codeRegexp = regexp.MustCompile(`code: (0-9]+)`)
var jsonValueRegexp = regexp.MustCompile(`JSON_VALUE\(data, '(?P<path>[$._a-z]+)'\)(?:, 'Float64'\)\))? AS (?P<name>[a-z_]+)`)
var eventTypeRegexp = regexp.MustCompile(`(?m:^WHERE [a-zA-z0-z_.]+_events\.type = '(?P<type>[a-zA-z0-9_-]+)'$)`)
var namespaceRegexp = regexp.MustCompile(`(?m:^FROM (?P<namespace>[a-zA-z0-9_.]+)_events$)`)

type Storage interface {
	BatchInsert(ctx context.Context, namespace string, events []*serializer.CloudEventsKafkaPayload) error
	GetMeters(ctx context.Context) ([]*models.Meter, error)
}

type ClickHouseStorageConfig struct {
	ClickHouse clickhouse.Conn
	Database   string
}

func NewClickhouseStorage(config ClickHouseStorageConfig) *ClickHouseStorage {
	return &ClickHouseStorage{
		config: config,
	}
}

type ClickHouseStorage struct {
	config ClickHouseStorageConfig
}

func (c *ClickHouseStorage) BatchInsert(ctx context.Context, namespace string, events []*serializer.CloudEventsKafkaPayload) error {
	query := insertEventsQuery{
		Database:        c.config.Database,
		EventsTableName: clickhouse_connector.GetEventsTableName(namespace),
		Events:          events,
	}
	sql, args, err := query.toSQL()
	if err != nil {
		return err
	}

	err = c.config.ClickHouse.Exec(ctx, sql, args...)
	if err != nil {
		code := getCode(err)

		// Although we validate events before we sink them to storage
		// we can still get a bad request error if for example namespace gots dropped in the meantime or meter modified
		// We probably want to handle in ClickHouse (https://github.com/ClickHouse/ClickHouse/blob/master/src/Common/ErrorCodes.cpp)
		// Couple of code we want to deadletter instead of retry:
		// 6 CANNOT_PARSE_TEXT
		// 8 THERE_IS_NO_COLUMN
		// 16 NO_SUCH_COLUMN_IN_TABLE
		// 25 CANNOT_PARSE_ESCAPE_SEQUENCE
		// 26 CANNOT_PARSE_QUOTED_STRING
		// 27 CANNOT_PARSE_INPUT_ASSERTION_FAILED
		// 38 CANNOT_PARSE_DATE
		// 41 CANNOT_PARSE_DATETIME
		// 53 TYPE_MISMATCH
		// 60 UNKNOWN_TABLE
		// 69 ARGUMENT_OUT_OF_BOUND
		// 70 CANNOT_CONVERT_TYPE
		// 72 CANNOT_PARSE_NUMBER
		// 85 FORMAT_IS_NOT_SUITABLE_FOR_INPUT
		// 131 TOO_LARGE_STRING_SIZE
		// 158 TOO_MANY_ROWS
		// 201 QUOTA_EXCEEDED
		// 246 CORRUPTED_DATA
		switch code {
		case 6, 8, 16, 25, 26, 27, 38, 41, 53, 60, 69, 70, 72, 85, 131, 158, 201, 246:
			return NewProcessingError(fmt.Sprintf("insert events malformed event: %s", err), DEADLETTER)
		}

		return err
	}

	return nil
}

func (c *ClickHouseStorage) GetMeters(ctx context.Context) ([]*models.Meter, error) {
	views, err := c.getViews(ctx)
	if err != nil {
		return nil, err
	}

	meters := []*models.Meter{}
	for _, view := range views {
		// Skip tables like .inner_*
		if !strings.HasPrefix(view, "om_") {
			continue
		}

		// Skip namespace event tables
		if strings.HasSuffix(view, "_events") {
			continue
		}

		meter, err := c.getMeter(ctx, c.config.Database, view)
		if err != nil {
			return nil, err
		}
		meters = append(meters, meter)
	}

	return meters, nil
}

func (c *ClickHouseStorage) getMeter(ctx context.Context, database string, view string) (*models.Meter, error) {
	// Get create view query
	sql := showCreateViewQuery{Database: c.config.Database, View: view}.toSQL()
	row := c.config.ClickHouse.QueryRow(ctx, sql)
	err := row.Err()
	if err != nil {
		return nil, fmt.Errorf("failed to get view: %w", err)
	}
	var createSql string
	if err = row.Scan(&createSql); err != nil {
		return nil, err
	}

	// TODO: parse out window size and aggregation
	meter := &models.Meter{
		GroupBy: map[string]string{},
	}

	// Parse namespace
	match := namespaceRegexp.FindStringSubmatch(createSql)
	if len(match) != 2 {
		return nil, fmt.Errorf("failed to parse namespace from view: %s", view)
	}
	meter.Namespace = match[1][len(database)+4:]

	// Parse event type
	match = eventTypeRegexp.FindStringSubmatch(createSql)
	if len(match) != 2 {
		return nil, fmt.Errorf("failed to parse event type from view: %s", view)
	}
	meter.EventType = match[1]

	// Parse JSON Values
	lines := strings.Split(createSql, "\n")
	for _, line := range lines {
		match := jsonValueRegexp.FindStringSubmatch(line)
		if len(match) == 0 {
			continue
		}
		paramsMap := make(map[string]string)
		for i, name := range jsonValueRegexp.SubexpNames() {
			if i > 0 && i <= len(match) {
				if i > 0 && i <= len(match) {
					paramsMap[name] = match[i]
				}
			}
		}
		if paramsMap["name"] == "value" {
			meter.ValueProperty = paramsMap["path"]
		} else {
			meter.GroupBy[paramsMap["name"]] = paramsMap["path"]
		}
	}

	return meter, nil
}

func (c *ClickHouseStorage) getViews(ctx context.Context) ([]string, error) {
	sql := showTablesQuery{Database: c.config.Database}.toSQL()
	rows, err := c.config.ClickHouse.Query(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("failed to get views: %w", err)
	}
	var views []string
	for rows.Next() {
		var view string
		if err = rows.Scan(&view); err != nil {
			return nil, err
		}
		views = append(views, view)
	}

	return views, nil
}

type showCreateViewQuery struct {
	Database string
	View     string
}

func (q showCreateViewQuery) toSQL() string {
	sql := fmt.Sprintf("show create view %s.%s", sqlbuilder.Escape(q.Database), sqlbuilder.Escape(q.View))
	return sql
}

type showTablesQuery struct {
	Database string
}

func (q showTablesQuery) toSQL() string {
	sql := fmt.Sprintf("show tables from %s", sqlbuilder.Escape(q.Database))
	return sql
}

type insertEventsQuery struct {
	Database        string
	EventsTableName string
	Events          []*serializer.CloudEventsKafkaPayload
}

func (q insertEventsQuery) toSQL() (string, []interface{}, error) {
	tableName := fmt.Sprintf("%s.%s", sqlbuilder.Escape(q.Database), sqlbuilder.Escape(q.EventsTableName))

	query := sqlbuilder.ClickHouse.NewInsertBuilder()
	query.InsertInto(tableName)
	query.Cols("id", "type", "source", "subject", "time", "data")

	for _, event := range q.Events {
		query.Values(event.Id, event.Type, event.Source, event.Subject, event.Time, event.Data)
	}

	sql, args := query.Build()
	return sql, args, nil
}

func getCode(err error) int {
	tmp := codeRegexp.FindStringSubmatch(err.Error())
	if len(tmp) != 2 || tmp[1] == "" {
		return 0
	}

	code, err := strconv.Atoi(tmp[1])
	if err != nil {
		return 0
	}

	return code
}
