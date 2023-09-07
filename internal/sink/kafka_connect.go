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

// ConfigureConnector configures a connector.
func (k *KafkaConnect) ConfigureConnector(ctx context.Context, name string, config map[string]string) error {
	req := kafkaconnect.CreateConnectorRequest{
		Name:   &name,
		Config: &config,
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
