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

func (c *ClickHouseTracer) PrepareBatch(ctx context.Context, query string, opts ...driver.PrepareBatchOption) (driver.Batch, error) {
	ctx, span := c.Tracer.Start(ctx, "clickhouse.PrepareBatch", trace.WithAttributes(
		attribute.String("query", query),
	))

	batch, err := c.Conn.PrepareBatch(ctx, query, opts...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.End()

		return batch, err
	}

	return &tracedBatch{Batch: batch, span: span}, nil
}

// tracedBatch records Append/Send errors on the PrepareBatch span and ends the span
// when the batch is finalized. Ending an already-ended span is a no-op, so the
// Send/Abort/Close finalizers do not need to coordinate.
type tracedBatch struct {
	driver.Batch

	span trace.Span
}

func (b *tracedBatch) Append(v ...any) error {
	err := b.Batch.Append(v...)
	if err != nil {
		b.span.RecordError(err)
		b.span.SetStatus(codes.Error, err.Error())
	}

	return err
}

func (b *tracedBatch) AppendStruct(v any) error {
	err := b.Batch.AppendStruct(v)
	if err != nil {
		b.span.RecordError(err)
		b.span.SetStatus(codes.Error, err.Error())
	}

	return err
}

func (b *tracedBatch) Send() error {
	b.span.SetAttributes(attribute.Int("rows", b.Batch.Rows()))

	err := b.Batch.Send()
	if err != nil {
		b.span.RecordError(err)
		b.span.SetStatus(codes.Error, err.Error())
	}
	b.span.End()

	return err
}

func (b *tracedBatch) Abort() error {
	err := b.Batch.Abort()
	b.span.End()

	return err
}

func (b *tracedBatch) Close() error {
	err := b.Batch.Close()
	b.span.End()

	return err
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
