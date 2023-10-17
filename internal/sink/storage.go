package sink

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/huandu/go-sqlbuilder"

	"github.com/openmeterio/openmeter/internal/ingest/kafkaingest/serializer"
	"github.com/openmeterio/openmeter/internal/streaming/clickhouse_connector"
)

var codeRegexp = regexp.MustCompile(`code: (0-9]+)`)

type Storage interface {
	BatchInsert(ctx context.Context, namespace string, events []*serializer.CloudEventsKafkaPayload) error
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
