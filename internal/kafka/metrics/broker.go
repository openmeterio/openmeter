package metrics

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/internal/kafka/metrics/stats"
)

type BrokerMetrics struct {
	// Broker source (learned, configured, internal, logical)
	Source metric.Int64Gauge
	// Broker state (INIT, DOWN, CONNECT, AUTH, APIVERSION_QUERY, AUTH_HANDSHAKE, UP, UPDATE)
	State metric.Int64Gauge
	// Time since last broker state change (microseconds)
	StateAge metric.Int64Gauge
	// Number of requests awaiting transmission to broker
	RequestsAwaitingTransmission metric.Int64Gauge
	// Number of messages awaiting transmission to broker
	MessagesAwaitingTransmission metric.Int64Gauge
	// Number of requests in-flight to broker awaiting response
	InflightRequestsAwaitingResponse metric.Int64Gauge
	// Number of messages in-flight to broker awaiting response
	InflightMessagesAwaitingResponse metric.Int64Gauge
	// Total number of requests sent
	RequestsSent metric.Int64Counter
	// Total number of bytes sent
	RequestBytesSent metric.Int64Counter
	// Total number of transmission errors
	RequestErrors metric.Int64Counter
	// Total number of request retries
	RequestRetries metric.Int64Counter
	// Microseconds since last socket send (or -1 if no sends yet for current connection).
	LastSocketSend metric.Int64Counter
	// Total number of requests timed out
	RequestTimeouts metric.Int64Counter
	// Total number of responses received
	ResponsesReceived metric.Int64Counter
	// Total number of bytes received
	ResponseBytesReceived metric.Int64Counter
	// Total number of receive errors
	ResponseErrors metric.Int64Counter
	// Microseconds since last socket receive (or -1 if no receives yet for current connection).
	LastSocketReceive metric.Int64Counter
	// Number of connection attempts, including successful and failed, and name resolution failures.
	Connects metric.Int64Counter
	// Number of disconnects (triggered by broker, network, load-balancer, etc.).
	Disconnects metric.Int64Counter

	// Smallest value
	LatencyMin metric.Int64Gauge
	// Largest value
	LatencyMax metric.Int64Gauge
	// Average value
	LatencyAvg metric.Int64Gauge
	// Sum of values
	LatencySum metric.Int64Gauge
	// Standard deviation (based on histogram)
	LatencyStdDev metric.Int64Gauge
	// 50th percentile
	LatencyP50 metric.Int64Gauge
	// 75th percentile
	LatencyP75 metric.Int64Gauge
	// 90th percentile
	LatencyP90 metric.Int64Gauge
	// 95th percentile
	LatencyP95 metric.Int64Gauge
	// 99th percentile
	LatencyP99 metric.Int64Gauge
	// 99.99th percentile
	LatencyP9999 metric.Int64Gauge

	// Smallest value
	ThrottleMin metric.Int64Gauge
	// Largest value
	ThrottleMax metric.Int64Gauge
	// Average value
	ThrottleAvg metric.Int64Gauge
	// Sum of values
	ThrottleSum metric.Int64Gauge
	// Standard deviation (based on histogram)
	ThrottleStdDev metric.Int64Gauge
	// Memory size of Hdr Histogram
	ThrottleHdrSize metric.Int64Gauge
	// 50th percentile
	ThrottleP50 metric.Int64Gauge
	// 75th percentile
	ThrottleP75 metric.Int64Gauge
	// 90th percentile
	ThrottleP90 metric.Int64Gauge
	// 95th percentile
	ThrottleP95 metric.Int64Gauge
	// 99th percentile
	ThrottleP99 metric.Int64Gauge
	// 99.99th percentile
	ThrottleP9999 metric.Int64Gauge
}

func (m *BrokerMetrics) Add(ctx context.Context, stats *stats.BrokerStats, attrs ...attribute.KeyValue) {
	attrs = append(attrs, []attribute.KeyValue{
		attribute.String("node_name", stats.NodeName),
		attribute.Int64("node_id", stats.NodeID),
	}...)

	m.Source.Record(ctx, stats.Source.Int64(), metric.WithAttributes(attrs...))
	m.State.Record(ctx, stats.State.Int64(), metric.WithAttributes(attrs...))
	m.StateAge.Record(ctx, stats.StateAge, metric.WithAttributes(attrs...))
	m.RequestsAwaitingTransmission.Record(ctx, stats.RequestsAwaitingTransmission, metric.WithAttributes(attrs...))
	m.MessagesAwaitingTransmission.Record(ctx, stats.MessagesAwaitingTransmission, metric.WithAttributes(attrs...))
	m.InflightRequestsAwaitingResponse.Record(ctx, stats.InflightRequestsAwaitingResponse, metric.WithAttributes(attrs...))
	m.InflightMessagesAwaitingResponse.Record(ctx, stats.InflightMessagesAwaitingResponse, metric.WithAttributes(attrs...))
	m.RequestsSent.Add(ctx, stats.RequestsSent, metric.WithAttributes(attrs...))
	m.RequestBytesSent.Add(ctx, stats.RequestBytesSent, metric.WithAttributes(attrs...))
	m.RequestErrors.Add(ctx, stats.RequestErrors, metric.WithAttributes(attrs...))
	m.RequestRetries.Add(ctx, stats.RequestRetries, metric.WithAttributes(attrs...))
	m.LastSocketSend.Add(ctx, stats.LastSocketSend, metric.WithAttributes(attrs...))
	m.RequestTimeouts.Add(ctx, stats.RequestTimeouts, metric.WithAttributes(attrs...))
	m.ResponsesReceived.Add(ctx, stats.ResponsesReceived, metric.WithAttributes(attrs...))
	m.ResponseBytesReceived.Add(ctx, stats.ResponseBytesReceived, metric.WithAttributes(attrs...))
	m.ResponseErrors.Add(ctx, stats.ResponseErrors, metric.WithAttributes(attrs...))
	m.LastSocketReceive.Add(ctx, stats.LastSocketReceive, metric.WithAttributes(attrs...))
	m.Connects.Add(ctx, stats.Connects, metric.WithAttributes(attrs...))
	m.Disconnects.Add(ctx, stats.Disconnects, metric.WithAttributes(attrs...))

	m.LatencyMin.Record(ctx, stats.Latency.Min, metric.WithAttributes(attrs...))
	m.LatencyMax.Record(ctx, stats.Latency.Max, metric.WithAttributes(attrs...))
	m.LatencyAvg.Record(ctx, stats.Latency.Avg, metric.WithAttributes(attrs...))
	m.LatencySum.Record(ctx, stats.Latency.Sum, metric.WithAttributes(attrs...))
	m.LatencyStdDev.Record(ctx, stats.Latency.StdDev, metric.WithAttributes(attrs...))
	m.LatencyP50.Record(ctx, stats.Latency.P50, metric.WithAttributes(attrs...))
	m.LatencyP75.Record(ctx, stats.Latency.P75, metric.WithAttributes(attrs...))
	m.LatencyP90.Record(ctx, stats.Latency.P90, metric.WithAttributes(attrs...))
	m.LatencyP95.Record(ctx, stats.Latency.P95, metric.WithAttributes(attrs...))
	m.LatencyP99.Record(ctx, stats.Latency.P99, metric.WithAttributes(attrs...))
	m.LatencyP9999.Record(ctx, stats.Latency.P9999, metric.WithAttributes(attrs...))

	m.ThrottleMin.Record(ctx, stats.Throttle.Min, metric.WithAttributes(attrs...))
	m.ThrottleMax.Record(ctx, stats.Throttle.Max, metric.WithAttributes(attrs...))
	m.ThrottleAvg.Record(ctx, stats.Throttle.Avg, metric.WithAttributes(attrs...))
	m.ThrottleSum.Record(ctx, stats.Throttle.Sum, metric.WithAttributes(attrs...))
	m.ThrottleStdDev.Record(ctx, stats.Throttle.StdDev, metric.WithAttributes(attrs...))
	m.ThrottleP50.Record(ctx, stats.Throttle.P50, metric.WithAttributes(attrs...))
	m.ThrottleP75.Record(ctx, stats.Throttle.P75, metric.WithAttributes(attrs...))
	m.ThrottleP90.Record(ctx, stats.Throttle.P90, metric.WithAttributes(attrs...))
	m.ThrottleP95.Record(ctx, stats.Throttle.P95, metric.WithAttributes(attrs...))
	m.ThrottleP99.Record(ctx, stats.Throttle.P99, metric.WithAttributes(attrs...))
	m.ThrottleP9999.Record(ctx, stats.Throttle.P9999, metric.WithAttributes(attrs...))
}

func NewBrokerMetrics(meter metric.Meter) (*BrokerMetrics, error) {
	var err error
	m := &BrokerMetrics{}

	m.Source, err = meter.Int64Gauge(
		"kafka.broker.source",
		metric.WithDescription("Broker source: [Unknown(-1), Learned(0), Configured(1), Internal(2), Logical(3)]"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.state: %w", err)
	}

	m.State, err = meter.Int64Gauge(
		"kafka.broker.state",
		metric.WithDescription("Broker state: [Unknown(-1), Init(0), Down(1), Connect(2), Auth(3), ApiVersionQuery(4), AuthHandshake(5), Up(6), Update(7)]"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.state: %w", err)
	}

	m.StateAge, err = meter.Int64Gauge(
		"kafka.broker.state_age",
		metric.WithDescription("Time since last broker state change (microseconds)"),
		metric.WithUnit("{microsecond}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.state_age: %w", err)
	}

	m.RequestsAwaitingTransmission, err = meter.Int64Gauge(
		"kafka.broker.request_awaiting_send",
		metric.WithDescription("Number of requests awaiting transmission to broker"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.request_awaiting_send: %w", err)
	}

	m.MessagesAwaitingTransmission, err = meter.Int64Gauge(
		"kafka.broker.message_awaiting_send",
		metric.WithDescription("Number of messages awaiting transmission to broker"),
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.message_awaiting_send: %w", err)
	}

	m.InflightRequestsAwaitingResponse, err = meter.Int64Gauge(
		"kafka.broker.inflight_requests_awaiting_response",
		metric.WithDescription("Number of requests in-flight to broker awaiting response"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.inflight_requests_awaiting_response: %w", err)
	}

	m.InflightMessagesAwaitingResponse, err = meter.Int64Gauge(
		"kafka.broker.inflight_messages_awaiting_response",
		metric.WithDescription("Number of messages in-flight to broker awaiting response"),
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.inflight_messages_awaiting_response: %w", err)
	}

	m.RequestsSent, err = meter.Int64Counter(
		"kafka.broker.request_sent",
		metric.WithDescription("Total number of requests sent"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.request_sent: %w", err)
	}

	m.RequestBytesSent, err = meter.Int64Counter(
		"kafka.broker.request_bytes_sent",
		metric.WithDescription("Total number of bytes sent"),
		metric.WithUnit("{byte}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.request_bytes_sent: %w", err)
	}

	m.RequestErrors, err = meter.Int64Counter(
		"kafka.broker.request_errors",
		metric.WithDescription("Total number of transmission errors"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.request_errors: %w", err)
	}

	m.RequestRetries, err = meter.Int64Counter(
		"kafka.broker.request_retries",
		metric.WithDescription("Total number of request retries"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.request_retries: %w", err)
	}

	m.LastSocketSend, err = meter.Int64Counter(
		"kafka.broker.last_socket_send",
		metric.WithDescription("Microseconds since last socket send (or -1 if no sends yet for current connection)"),
		metric.WithUnit("{microsecond}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.last_socket_send: %w", err)
	}

	m.RequestTimeouts, err = meter.Int64Counter(
		"kafka.broker.request_timeouts",
		metric.WithDescription("Total number of requests timed out"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.request_retries: %w", err)
	}

	m.ResponsesReceived, err = meter.Int64Counter(
		"kafka.broker.responses_received",
		metric.WithDescription("Total number of responses received"),
		metric.WithUnit("{response}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.responses_received: %w", err)
	}

	m.ResponseBytesReceived, err = meter.Int64Counter(
		"kafka.broker.responses_bytes_received",
		metric.WithDescription("Total number of bytes received"),
		metric.WithUnit("{byte}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.responses_bytes_received: %w", err)
	}

	m.ResponseErrors, err = meter.Int64Counter(
		"kafka.broker.responses_errors",
		metric.WithDescription("Total number of receive errors"),
		metric.WithUnit("{event}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.responses_errors: %w", err)
	}

	m.LastSocketReceive, err = meter.Int64Counter(
		"kafka.broker.last_socket_receive",
		metric.WithDescription("Microseconds since last socket receive (or -1 if no receives yet for current connection)"),
		metric.WithUnit("{microsecond}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.last_socket_receive: %w", err)
	}

	m.Connects, err = meter.Int64Counter(
		"kafka.broker.connects",
		metric.WithDescription("Number of connection attempts, including successful and failed, and name resolution failures"),
		metric.WithUnit("{event}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.connects: %w", err)
	}

	m.Disconnects, err = meter.Int64Counter(
		"kafka.broker.disconnects",
		metric.WithDescription("Number of disconnects (triggered by broker, network, load-balancer, etc.)"),
		metric.WithUnit("{event}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.disconnects: %w", err)
	}

	m.LatencyMin, err = meter.Int64Gauge(
		"kafka.broker.latency_min",
		metric.WithUnit("{microsecond}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.latency_min: %w", err)
	}

	m.LatencyMax, err = meter.Int64Gauge(
		"kafka.broker.latency_max",
		metric.WithUnit("{microsecond}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.latency_max: %w", err)
	}

	m.LatencyAvg, err = meter.Int64Gauge(
		"kafka.broker.latency_avg",
		metric.WithUnit("{microsecond}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.latency_avg: %w", err)
	}

	m.LatencySum, err = meter.Int64Gauge(
		"kafka.broker.latency_sum",
		metric.WithUnit("{microsecond}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.latency_sum: %w", err)
	}

	m.LatencyStdDev, err = meter.Int64Gauge(
		"kafka.broker.latency_stddev",
		metric.WithDescription("Standard deviation (based on histogram)"),
		metric.WithUnit("{microsecond}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.latency_stddev: %w", err)
	}

	m.LatencyP50, err = meter.Int64Gauge(
		"kafka.broker.latency_p50",
		metric.WithUnit("{microsecond}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.latency_p50: %w", err)
	}

	m.LatencyP75, err = meter.Int64Gauge(
		"kafka.broker.latency_p75",
		metric.WithUnit("{microsecond}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.latency_p75: %w", err)
	}

	m.LatencyP90, err = meter.Int64Gauge(
		"kafka.broker.latency_p90",
		metric.WithUnit("{microsecond}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.latency_p90: %w", err)
	}

	m.LatencyP95, err = meter.Int64Gauge(
		"kafka.broker.latency_p95",
		metric.WithUnit("{microsecond}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.latency_p95: %w", err)
	}

	m.LatencyP99, err = meter.Int64Gauge(
		"kafka.broker.latency_p99",
		metric.WithUnit("{microsecond}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.latency_p99: %w", err)
	}

	m.LatencyP9999, err = meter.Int64Gauge(
		"kafka.broker.latency_p9999",
		metric.WithUnit("{microsecond}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.latency_p9999: %w", err)
	}

	m.ThrottleMin, err = meter.Int64Gauge(
		"kafka.broker.throttle_min",
		metric.WithUnit("{microsecond}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.throttle_min: %w", err)
	}

	m.ThrottleMax, err = meter.Int64Gauge(
		"kafka.broker.throttle_max",
		metric.WithUnit("{microsecond}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.throttle_max: %w", err)
	}

	m.ThrottleAvg, err = meter.Int64Gauge(
		"kafka.broker.throttle_avg",
		metric.WithUnit("{microsecond}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.throttle_avg: %w", err)
	}

	m.ThrottleSum, err = meter.Int64Gauge(
		"kafka.broker.throttle_sum",
		metric.WithUnit("{microsecond}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.throttle_sum: %w", err)
	}

	m.ThrottleStdDev, err = meter.Int64Gauge(
		"kafka.broker.throttle_stddev",
		metric.WithDescription("Standard deviation (based on histogram)"),
		metric.WithUnit("{microsecond}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.throttle_stddev: %w", err)
	}

	m.ThrottleP50, err = meter.Int64Gauge(
		"kafka.broker.throttle_p50",
		metric.WithUnit("{microsecond}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.throttle_p50: %w", err)
	}

	m.ThrottleP75, err = meter.Int64Gauge(
		"kafka.broker.throttle_p75",
		metric.WithUnit("{microsecond}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.throttle_p75: %w", err)
	}

	m.ThrottleP90, err = meter.Int64Gauge(
		"kafka.broker.throttle_p90",
		metric.WithUnit("{microsecond}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.throttle_p90: %w", err)
	}

	m.ThrottleP95, err = meter.Int64Gauge(
		"kafka.broker.throttle_p95",
		metric.WithUnit("{microsecond}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.throttle_p95: %w", err)
	}

	m.ThrottleP99, err = meter.Int64Gauge(
		"kafka.broker.throttle_p99",
		metric.WithUnit("{microsecond}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.throttle_p99: %w", err)
	}

	m.ThrottleP9999, err = meter.Int64Gauge(
		"kafka.broker.throttle_p9999",
		metric.WithUnit("{microsecond}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.broker.throttle_p9999: %w", err)
	}

	return m, nil
}
