package clickhouseotel

import (
	"context"
	"errors"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type ClickHouseTracer struct {
	clickhouse.Conn

	Tracer trace.Tracer
}

type ClickHouseTracerConfig struct {
	Tracer trace.Tracer
	Conn   clickhouse.Conn
}

func (c ClickHouseTracerConfig) Validate() error {
	var errs []error

	if c.Tracer == nil {
		errs = append(errs, errors.New("tracer is required"))
	}

	if c.Conn == nil {
		errs = append(errs, errors.New("conn is required"))
	}

	return errors.Join(errs...)
}

var _ clickhouse.Conn = (*ClickHouseTracer)(nil)

func NewClickHouseTracer(cfg ClickHouseTracerConfig) (clickhouse.Conn, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &ClickHouseTracer{
		Conn:   cfg.Conn,
		Tracer: cfg.Tracer,
	}, nil
}

func anyToStrings(args ...any) []string {
	return lo.Map(args, func(arg any, _ int) string {
		return fmt.Sprintf("%v", arg)
	})
}

func (c *ClickHouseTracer) Query(ctx context.Context, query string, args ...any) (rows driver.Rows, err error) {
	ctx, span := c.Tracer.Start(ctx, "clickhouse.Query", trace.WithAttributes(
		attribute.String("query", query),
		attribute.StringSlice("args", anyToStrings(args...)),
	))
	defer span.End()

	rows, err = c.Conn.Query(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return rows, err
}

func (c *ClickHouseTracer) QueryRow(ctx context.Context, query string, args ...any) driver.Row {
	ctx, span := c.Tracer.Start(ctx, "clickhouse.QueryRow", trace.WithAttributes(
		attribute.String("query", query),
		attribute.StringSlice("args", anyToStrings(args...)),
	))
	defer span.End()

	row := c.Conn.QueryRow(ctx, query, args...)
	if row != nil && row.Err() != nil {
		err := row.Err()
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return row
}

func (c *ClickHouseTracer) Exec(ctx context.Context, query string, args ...any) error {
	ctx, span := c.Tracer.Start(ctx, "clickhouse.Exec", trace.WithAttributes(
		attribute.String("query", query),
		attribute.StringSlice("args", anyToStrings(args...)),
	))
	defer span.End()

	err := c.Conn.Exec(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return err
	}

	return nil
}

func (c *ClickHouseTracer) AsyncInsert(ctx context.Context, query string, wait bool, args ...any) error {
	ctx, span := c.Tracer.Start(ctx, "clickhouse.AsyncInsert", trace.WithAttributes(
		attribute.String("query", query),
		attribute.Bool("wait", wait),
		attribute.StringSlice("args", anyToStrings(args...)),
	))
	defer span.End()

	err := c.Conn.AsyncInsert(ctx, query, wait, args...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}
