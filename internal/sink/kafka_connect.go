package sink

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type KafkaConnect struct {
	config *KafkaConnectConfig
}

type KafkaConnectConfig struct {
	Address string
}

type Connector struct {
	Name   string            `json:"name"`
	Config map[string]string `json:"config"`
}

func NewKafkaConnect(config *KafkaConnectConfig) (*KafkaConnect, error) {
	return &KafkaConnect{
		config: config,
	}, nil
}

func (k *KafkaConnect) CreateConnector(ctx context.Context, connector *Connector) error {
	endpoint := fmt.Sprintf("%s/connectors", k.config.Address)

	jsonData, err := json.Marshal(connector)
	if err != nil {
		return err
	}

	request, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	request = request.WithContext(ctx)
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
