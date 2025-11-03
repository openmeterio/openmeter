package cogs

import (
	"fmt"

	"go.opentelemetry.io/otel/metric"
)

const (
	MetricNameComputeInProcessExecutionMs = "cogs.compute.in_process_execution"
	MetricNamePostgresRoundTripMs         = "cogs.postgres.round_trip"
	MetricNamePostgresPingRoundTripMs     = "cogs.postgres.ping_round_trip"
	MetricNameClickHouseRoundTripMs       = "cogs.clickhouse.round_trip"
	MetricNameRedisRoundTripMs            = "cogs.redis.round_trip"
	MetricNameKafkaProducedMsgCount       = "cogs.kafka.produced_msg_count"
	MetricNameKafkaConsumedMsgCount       = "cogs.kafka.consumed_msg_count"
)

type Metrics struct {
	ComputeInProcessExecutionMs metric.Int64Histogram

	// We COULD add service specific meaningful implementations (e.g. PgStatActivity based metrics for Postgres).
	// It's GOOD ENOUGH for now to estimate resource-cost by round-trip time of the request.
	// We can deduct estimated network time from the total round-trip on a per environment basis in the analytics dashboards (e.g. ~8ms for PG in AWS us-east-1)
	PostgresRoundTripMs     metric.Int64Histogram
	PostgresPingRoundTripMs metric.Int64Histogram
	ClickHouseRoundTripMs   metric.Int64Histogram
	RedisRoundTripMs        metric.Int64Histogram

	KafkaProducedMsgCount metric.Int64Counter
	KafkaConsumedMsgCount metric.Int64Counter
}

func NewMetrics(meter metric.Meter) (*Metrics, error) {
	computeInProcessExecutionMs, err := meter.Int64Histogram(
		MetricNameComputeInProcessExecutionMs,
		metric.WithDescription("Compute in process execution time in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create compute in process execution ms metric: %w", err)
	}

	postgresRoundTripMs, err := meter.Int64Histogram(
		MetricNamePostgresRoundTripMs,
		metric.WithDescription("Postgres round trip time in milliseconds. When estimating resource cost, deduct estimated network time from the total round-trip in the analytics."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres round trip ms metric: %w", err)
	}

	postgresPingRoundTripMs, err := meter.Int64Histogram(
		MetricNamePostgresPingRoundTripMs,
		metric.WithDescription("Postgres ping round trip time in milliseconds. When estimating resource cost, deduct estimated network time from the total round-trip in the analytics."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres ping round trip ms metric: %w", err)
	}

	clickhouseRoundTripMs, err := meter.Int64Histogram(
		MetricNameClickHouseRoundTripMs,
		metric.WithDescription("Clickhouse round trip time in milliseconds. When estimating resource cost, deduct estimated network time from the total round-trip on a per environment basis in the analytics."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create clickhouse round trip ms metric: %w", err)
	}

	redisRoundTripMs, err := meter.Int64Histogram(
		MetricNameRedisRoundTripMs,
		metric.WithDescription("Redis round trip time in milliseconds. When estimating resource cost, deduct estimated network time from the total round-trip on a per environment basis in the analytics."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create redis round trip ms metric: %w", err)
	}

	kafkaProducedMsgCount, err := meter.Int64Counter(
		MetricNameKafkaProducedMsgCount,
		metric.WithDescription("Kafka produced message count"),
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka produced msg count metric: %w", err)
	}

	kafkaConsumedMsgCount, err := meter.Int64Counter(
		MetricNameKafkaConsumedMsgCount,
		metric.WithDescription("Kafka consumed message count"),
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka consumed msg count metric: %w", err)
	}

	return &Metrics{
		ComputeInProcessExecutionMs: computeInProcessExecutionMs,
		PostgresRoundTripMs:         postgresRoundTripMs,
		PostgresPingRoundTripMs:     postgresPingRoundTripMs,
		ClickHouseRoundTripMs:       clickhouseRoundTripMs,
		RedisRoundTripMs:            redisRoundTripMs,
		KafkaProducedMsgCount:       kafkaProducedMsgCount,
		KafkaConsumedMsgCount:       kafkaConsumedMsgCount,
	}, nil
}
