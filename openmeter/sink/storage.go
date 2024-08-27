package sink

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/huandu/go-sqlbuilder"

	sinkmodels "github.com/openmeterio/openmeter/openmeter/sink/models"
	"github.com/openmeterio/openmeter/openmeter/streaming/clickhouse_connector"
)

type Storage interface {
	BatchInsert(ctx context.Context, messages []sinkmodels.SinkMessage) error
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

func (c *ClickHouseStorage) BatchInsert(ctx context.Context, messages []sinkmodels.SinkMessage) error {
	query := InsertEventsQuery{
		Database: c.config.Database,
		Messages: messages,
	}
	sql, args, err := query.ToSQL()
	if err != nil {
		return err
	}

	err = c.config.ClickHouse.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("failed to batch insert events: %w", err)
	}

	return nil
}

type InsertEventsQuery struct {
	Database string
	Messages []sinkmodels.SinkMessage
}

func (q InsertEventsQuery) ToSQL() (string, []interface{}, error) {
	tableName := clickhouse_connector.GetEventsTableName(q.Database)

	query := sqlbuilder.ClickHouse.NewInsertBuilder()
	query.InsertInto(tableName)
	query.Cols("namespace", "validation_error", "id", "type", "source", "subject", "time", "data", "ingested_at", "stored_at")

	for _, message := range q.Messages {
		var eventErr string
		if message.Status.Error != nil {
			eventErr = message.Status.Error.Error()
		}

		storedAt := time.Now()
		ingestedAt := storedAt

		if message.KafkaMessage != nil {
			for _, header := range message.KafkaMessage.Headers {
				// Parse ingested_at header
				if header.Key == "ingested_at" {
					var err error

					ingestedAt, err = time.Parse(time.RFC3339, string(header.Value))
					if err != nil {
						eventErr = fmt.Sprintf("failed to parse ingested_at header: %s", err)
					}
				}
			}
		}

		query.Values(
			message.Namespace,
			eventErr,
			message.Serialized.Id,
			message.Serialized.Type,
			message.Serialized.Source,
			message.Serialized.Subject,
			message.Serialized.Time,
			message.Serialized.Data,
			ingestedAt,
			storedAt,
		)
	}

	sql, args := query.Build()
	return sql, args, nil
}
