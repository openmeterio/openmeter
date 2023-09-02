// Package kafkaconnect implements a simple client for the Kafka Connect REST interface.
package kafkaconnect

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
)

// Client is a Kafka Connect client using the [REST interface].
//
// [REST interface]: https://docs.confluent.io/platform/current/connect/references/restapi.html
type Client struct {
	url string

	httpClient *http.Client
	logger     *slog.Logger

	initOnce sync.Once
}

// NewClient returns a new [Client].
func NewClient(url string) *Client {
	client := &Client{
		url: url,
	}

	client.init()

	return client
}

// Option configures a [Client] using the functional options paradigm
// popularized by Rob Pike and Dave Cheney.
//
// If you're unfamiliar with this style, see:
// - https://commandcenter.blogspot.com/2014/01/self-referential-functions-and-design.html
// - https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis.
// - https://sagikazarmark.hu/blog/functional-options-on-steroids/
type Option interface {
	apply(*Client)
}

type optionFunc func(*Client)

func (fn optionFunc) apply(c *Client) {
	fn(c)
}

// HTTPClient configures an [http.Client] in [Client].
func HTTPClient(httpClient *http.Client) Option {
	return optionFunc(func(c *Client) {
		c.httpClient = httpClient
	})
}

// Logger configures an [slog.Logger] in [Client].
func Logger(logger *slog.Logger) Option {
	return optionFunc(func(c *Client) {
		c.logger = logger
	})
}

func (c *Client) init() {
	c.initOnce.Do(func() {
		if c.httpClient == nil {
			c.httpClient = http.DefaultClient
		}

		if c.logger == nil {
			c.logger = slog.Default()
		}
	})
}

// Connector is a connector in Kafka Connect.
type Connector struct {
	Name   string            `json:"name"`
	Config map[string]string `json:"config"`
}

// CreateConnector creates a new connector in Kafka Connect.
func (c *Client) CreateConnector(ctx context.Context, connector Connector) error {
	endpoint := fmt.Sprintf("%s/connectors", c.url)

	requestBody, err := json.Marshal(connector)
	if err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("unable to parse body: %w", err)
	}

	// TODO: do this check one level higher.
	if response.StatusCode == http.StatusCreated {
		c.logger.Debug("connector created", slog.String("name", connector.Name))

		return nil
	}

	if response.StatusCode == http.StatusConflict {
		c.logger.Debug("connector may already exist or rebalance is in progress", slog.String("name", connector.Name))

		return nil
	}

	// TODO: only log error not the whole body
	c.logger.Error("unexpected status code at connector create", "status_code", response.StatusCode, "body", string(responseBody))

	return fmt.Errorf("unexpected status code at connector create: %d", response.StatusCode)
}
