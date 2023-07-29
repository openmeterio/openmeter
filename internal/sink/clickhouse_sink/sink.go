package clickhouse_sink

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/internal/streaming/clickhouse_connector"
)

type ClickHouseSink struct {
	config *ClickHouseSinkConfig
}

type ClickHouseSinkConfig struct {
	KafkaConnectAddress string
}

type connector struct {
	Name   string            `json:"name"`
	Config map[string]string `json:"config"`
}

func NewClickHouseSink(config *ClickHouseSinkConfig) (*ClickHouseSink, error) {
	return &ClickHouseSink{
		config: config,
	}, nil
}

func (k *ClickHouseSink) CreateNamespace(ctx context.Context, namespace string) error {
	endpoint := fmt.Sprintf("%s/connectors", k.config.KafkaConnectAddress)

	connector := connector{
		Name: "clickhouse",
		Config: map[string]string{
			"connector.class":                "com.clickhouse.kafka.connect.ClickHouseSinkConnector",
			"database":                       "default",
			"errors.retry.timeout":           "30",
			"hostname":                       "clickhouse",
			"port":                           "8123",
			"ssl":                            "false",
			"username":                       "default",
			"password":                       "",
			"key.converter":                  "org.apache.kafka.connect.storage.StringConverter",
			"value.converter":                "org.apache.kafka.connect.json.JsonConverter",
			"value.converter.schemas.enable": "false",
			"schemas.enable":                 "false",
			"topics":                         clickhouse_connector.GetEventsTableName(namespace),
		},
	}

	jsonData, err := json.Marshal(connector)
	if err != nil {
		return err
	}

	request, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode == 201 || response.StatusCode == 409 {
		return nil
	}

	return fmt.Errorf("unexpected status code at connector create: %d", response.StatusCode)
}
