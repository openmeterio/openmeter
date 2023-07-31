package sink

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/exp/slog"
)

type KafkaConnect struct {
	config KafkaConnectConfig
}

type KafkaConnectConfig struct {
	URL        string
	HttpClient *http.Client
	Logger     *slog.Logger
}

type Connector struct {
	Name   string            `json:"name"`
	Config map[string]string `json:"config"`
}

func NewKafkaConnect(config KafkaConnectConfig) (KafkaConnect, error) {
	if config.HttpClient == nil {
		config.HttpClient = http.DefaultClient
	}

	return KafkaConnect{
		config: config,
	}, nil
}

func (k *KafkaConnect) CreateConnector(ctx context.Context, connector Connector) error {
	endpoint := fmt.Sprintf("%s/connectors", k.config.URL)

	jsonData, err := json.Marshal(connector)
	if err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	response, err := k.config.HttpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("unable to parse body: %w", err)
	}

	if response.StatusCode == 201 || response.StatusCode == 409 {
		return nil
	}

	// TODO: only log error not the whole body
	k.config.Logger.Error("unexpected status code at connector create", "status_code", response.StatusCode, "body", string(body))
	return fmt.Errorf("unexpected status code at connector create: %d", response.StatusCode)
}
