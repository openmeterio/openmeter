// Copyright 2022 The OpenMeter Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package common

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/wire"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/pkg/framework/clickhouseotel"
)

var ClickHouse = wire.NewSet(
	NewClickHouse,
)

func NewClickHouse(ctx context.Context, conf config.ClickHouseAggregationConfiguration, tracer trace.Tracer, meter metric.Meter, logger *slog.Logger) (clickhouse.Conn, func(), error) {
	noopClose := func() {}

	closers := []func(){}

	conn, err := clickhouse.Open(conf.GetClientOptions())
	if err != nil {
		return nil, noopClose, fmt.Errorf("failed to initialize clickhouse client: %w", err)
	}

	closers = append(closers, func() {
		if err := conn.Close(); err != nil {
			logger.Error("failed to close clickhouse client", "error", err)
		}
	})

	if conf.Tracing {
		conn, err = clickhouseotel.NewClickHouseTracer(clickhouseotel.ClickHouseTracerConfig{
			Conn:   conn,
			Tracer: tracer,
		})
		if err != nil {
			return nil, noopClose, fmt.Errorf("failed to initialize clickhouse tracer: %w", err)
		}
	}

	if conf.PoolMetrics.Enabled {
		connPoolMetrics, err := clickhouseotel.NewConnPoolMetrics(clickhouseotel.ConnPoolMetricsConfig{
			Conn:         conn,
			Meter:        meter,
			PollInterval: conf.PoolMetrics.PollInterval,
			Logger:       logger,
		})
		if err != nil {
			return nil, noopClose, fmt.Errorf("failed to initialize clickhouse pool metrics: %w", err)
		}

		if err := connPoolMetrics.Start(ctx); err != nil {
			return nil, noopClose, fmt.Errorf("failed to start clickhouse pool metrics: %w", err)
		}

		closers = append(closers, func() {
			if err := connPoolMetrics.Shutdown(); err != nil {
				logger.Error("failed to shutdown clickhouse pool metrics", "error", err)
			}
		})
	}

	return conn, func() {
		for _, closer := range closers {
			closer()
		}
	}, nil
}
