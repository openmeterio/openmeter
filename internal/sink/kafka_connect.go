package sink

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/openmeterio/openmeter/pkg/kafkaconnect"
)

// KafkaConnect configures a connector.
type KafkaConnect struct {
	Client *kafkaconnect.Client
	Logger *slog.Logger
}

// Config configures a ClickHouse connector.
type Config struct {
	DeadLetterQueueTopicName         string
	DeadLetterQueueReplicationFactor int
	DeadLetterQueueContextHeaders    bool
	Database                         string
	Hostname                         string
	Port                             int
	SSL                              bool
	Username                         string
	Password                         string
}

// ConfigureConnector configures a connector.
func (k *KafkaConnect) ConfigureConnector(ctx context.Context, config Config) error {
	name := "clickhouse"

	req := kafkaconnect.CreateConnectorRequest{
		Name: &name,
		Config: &map[string]string{
			"connector.class":                   "com.clickhouse.kafka.connect.ClickHouseSinkConnector",
			"database":                          config.Database,
			"errors.retry.timeout":              "30",
			"hostname":                          config.Hostname,
			"port":                              fmt.Sprint(config.Port),
			"ssl":                               fmt.Sprint(config.SSL),
			"username":                          config.Username,
			"password":                          config.Password,
			"key.converter":                     "org.apache.kafka.connect.storage.StringConverter",
			"value.converter":                   "org.apache.kafka.connect.json.JsonConverter",
			"value.converter.schemas.enable":    "false",
			"schemas.enable":                    "false",
			"topics.regex":                      "^om_[A-Za-z0-9]+(?:_[A-Za-z0-9]+)*_events$",
			"errors.tolerance":                  "all",
			"errors.deadletterqueue.topic.name": config.DeadLetterQueueTopicName,
			"errors.deadletterqueue.topic.replication.factor": fmt.Sprint(config.DeadLetterQueueReplicationFactor),
			"errors.deadletterqueue.context.headers.enable":   fmt.Sprint(config.DeadLetterQueueContextHeaders),
		},
	}

	resp, err := k.Client.CreateConnector(ctx, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		k.Logger.Debug("connector created", slog.String("name", name))

		return nil
	}

	if resp.StatusCode == http.StatusConflict {
		k.Logger.Debug("connector already exists or rebalancing is in progress", slog.String("name", name))

		return nil
	}

	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)

		// TODO: only log error not the whole body
		k.Logger.Error("unexpected status code at connector create", "status_code", resp.StatusCode, "body", string(body))

		return fmt.Errorf("unexpected status code at connector create: %d", resp.StatusCode)
	}

	return nil
}
