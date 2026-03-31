package streamingretry

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"time"

	chproto "github.com/ClickHouse/ch-go/proto"
	"github.com/ClickHouse/clickhouse-go/v2"
	retryLib "github.com/avast/retry-go/v4"
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
		errs = append(errs, errors.New("max tries must be greater than 1"))
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
	return retryLib.DoWithData(fn,
		retryLib.Context(ctx),
		retryLib.Attempts(uint(c.maxTries)),
		retryLib.LastErrorOnly(true),
		retryLib.DelayType(retryLib.CombineDelay(
			retryLib.BackOffDelay,
			retryLib.RandomDelay,
		)),
		retryLib.Delay(c.retryWaitDuration),
		retryLib.OnRetry(func(n uint, err error) {
			c.logger.WarnContext(ctx, "operation failed, retrying",
				"attempt", n+1,
				"max_attempts", c.maxTries,
				"error", err,
			)
		}),
		retryLib.RetryIf(func(err error) bool {
			// Connection pool seems to be neglecting the pings in the connection pool, so we need to retry on EOFs to
			// compensate for clickhouse restarts due to updates.
			if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
				return true
			}

			// If the connection pool is full, we can retry, hoping for a free connection.
			if errors.Is(err, clickhouse.ErrAcquireConnTimeout) {
				return true
			}

			chException, ok := lo.ErrorsAs[*clickhouse.Exception](err)
			if ok {
				// During upscale/downscale of the cluster, CH might return this error, so let's retry.
				if chException.Code == int32(chproto.ErrAllConnectionTriesFailed) {
					return true
				}
			}

			return false
		}),
	)
}
