package streamingretry

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	chproto "github.com/ClickHouse/ch-go/proto"
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

type Config struct {
	DownstreamConnector streaming.Connector
	Logger              *slog.Logger
	RetryWaitDuration   time.Duration
	MaxTries            int
}

func (c Config) Validate() error {
	var errs []error

	if c.DownstreamConnector == nil {
		errs = append(errs, errors.New("downstream connector is required"))
	}

	if c.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}

	if c.RetryWaitDuration <= 0 {
		errs = append(errs, errors.New("retry wait duration must be greater than 0"))
	}

	if c.MaxTries <= 1 {
		errs = append(errs, errors.New("max retries must be greater than or equal to 1"))
	}

	return errors.Join(errs...)
}

type Connector struct {
	downstreamConnector streaming.Connector
	logger              *slog.Logger

	maxTries          int
	retryWaitDuration time.Duration
}

var _ streaming.Connector = (*Connector)(nil)

func New(config Config) (*Connector, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Connector{
		downstreamConnector: config.DownstreamConnector,
		maxTries:            config.MaxTries,
		retryWaitDuration:   config.RetryWaitDuration,
		logger:              config.Logger,
	}, nil
}

func (c *Connector) BatchInsert(ctx context.Context, events []streaming.RawEvent) error {
	return c.downstreamConnector.BatchInsert(ctx, events)
}

func (c *Connector) CountEvents(ctx context.Context, namespace string, params streaming.CountEventsParams) ([]streaming.CountEventRow, error) {
	return retry(ctx, c, func() ([]streaming.CountEventRow, error) {
		return c.downstreamConnector.CountEvents(ctx, namespace, params)
	})
}

func (c *Connector) ListEvents(ctx context.Context, namespace string, params streaming.ListEventsParams) ([]streaming.RawEvent, error) {
	return retry(ctx, c, func() ([]streaming.RawEvent, error) {
		return c.downstreamConnector.ListEvents(ctx, namespace, params)
	})
}

func (c *Connector) ListEventsV2(ctx context.Context, params streaming.ListEventsV2Params) ([]streaming.RawEvent, error) {
	return retry(ctx, c, func() ([]streaming.RawEvent, error) {
		return c.downstreamConnector.ListEventsV2(ctx, params)
	})
}

func (c *Connector) ListSubjects(ctx context.Context, params streaming.ListSubjectsParams) ([]string, error) {
	return retry(ctx, c, func() ([]string, error) {
		return c.downstreamConnector.ListSubjects(ctx, params)
	})
}

func (c *Connector) ListGroupByValues(ctx context.Context, params streaming.ListGroupByValuesParams) ([]string, error) {
	return retry(ctx, c, func() ([]string, error) {
		return c.downstreamConnector.ListGroupByValues(ctx, params)
	})
}

func (c *Connector) QueryMeter(ctx context.Context, namespace string, m meter.Meter, params streaming.QueryParams) ([]meter.MeterQueryRow, error) {
	return retry(ctx, c, func() ([]meter.MeterQueryRow, error) {
		return c.downstreamConnector.QueryMeter(ctx, namespace, m, params)
	})
}

func (c *Connector) ValidateJSONPath(ctx context.Context, jsonPath string) (bool, error) {
	return c.downstreamConnector.ValidateJSONPath(ctx, jsonPath)
}

func (c *Connector) CreateNamespace(ctx context.Context, namespace string) error {
	return c.downstreamConnector.CreateNamespace(ctx, namespace)
}

func (c *Connector) DeleteNamespace(ctx context.Context, namespace string) error {
	return c.downstreamConnector.DeleteNamespace(ctx, namespace)
}

func retry[T any](ctx context.Context, c *Connector, fn func() (T, error)) (T, error) {
	var empty T
	var err error

	for i := 0; i < c.maxTries; i++ {
		var res T

		res, err = fn()
		if err == nil {
			return res, nil
		}

		// Retirable errors
		if isErrorRetirable(ctx, c, err) {
			time.Sleep(c.retryWaitDuration)
			continue
		}

		// Everything else is permanent.
		return empty, err
	}

	return empty, fmt.Errorf("maximum retries exceeded: %w", err)
}

func isErrorRetirable(ctx context.Context, c *Connector, err error) bool {
	// Connection pool seems to be neglecting the pings in the connection pool, so we need to retry on EOFs to
	// compensate for clickhouse restarts due to updates.
	if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
		return true
	}

	// If the connection pool is full, we can retry, hoping for a free connection.
	if errors.Is(err, clickhouse.ErrAcquireConnTimeout) {
		c.logger.WarnContext(ctx, "clickhouse acquire connection timeout, connection pool is full", "error", err)
		return true
	}

	chException, ok := lo.ErrorsAs[*clickhouse.Exception](err)
	if ok {
		// During upscale/downscale of the cluster CH might return this error, so let's retry.
		if chException.Code == int32(chproto.ErrAllConnectionTriesFailed) {
			return true
		}
	}

	return false
}
