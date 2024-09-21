// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pgdriver

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/XSAM/otelsql"
	"github.com/jackc/pgx/v5/pgxpool"
	pgxstdlib "github.com/jackc/pgx/v5/stdlib"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/pkg/pgxpoolobserver"
)

type Option interface {
	apply(*options)
}

type optionFunc func(c *options)

func (fn optionFunc) apply(c *options) {
	fn(c)
}

func WithTracerProvider(p trace.TracerProvider) Option {
	return optionFunc(func(o *options) {
		o.otelOptions = append(o.otelOptions, otelsql.WithTracerProvider(p))
	})
}

func WithMeterProvider(p metric.MeterProvider) Option {
	return optionFunc(func(o *options) {
		o.otelOptions = append(o.otelOptions, otelsql.WithMeterProvider(p))
	})
}

func WithMetricMeter(m metric.Meter) Option {
	return optionFunc(func(o *options) {
		o.metricMeter = m
	})
}

type options struct {
	connConfig  *pgxpool.Config
	otelOptions []otelsql.Option
	metricMeter metric.Meter
}

type Driver struct {
	pool *pgxpool.Pool
	db   *sql.DB
}

func (d *Driver) DB() *sql.DB {
	return d.db
}

func (d *Driver) Close() error {
	d.pool.Close()

	return nil
}

func NewPostgresDriver(ctx context.Context, url string, opts ...Option) (*Driver, error) {
	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse postgres url: %w", err)
	}

	o := &options{
		connConfig: config,
		otelOptions: []otelsql.Option{
			otelsql.WithAttributes(
				semconv.DBSystemPostgreSQL,
			),
		},
	}

	for _, opt := range opts {
		opt.apply(o)
	}

	pool, err := pgxpool.NewWithConfig(ctx, o.connConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres pool: %w", err)
	}

	if o.metricMeter != nil {
		if err := pgxpoolobserver.ObservePoolMetrics(o.metricMeter, pool); err != nil {
			return nil, err
		}
	}

	db := otelsql.OpenDB(pgxstdlib.GetPoolConnector(pool), o.otelOptions...)

	// Set maximum idle connections to 0 as connections are managed from pgx.Pool.
	// See: https://github.com/jackc/pgx/blob/v5.6.0/stdlib/sql.go#L204-L208
	db.SetMaxIdleConns(0)

	return &Driver{
		pool: pool,
		db:   db,
	}, nil
}
