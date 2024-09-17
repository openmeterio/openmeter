package kafka

import (
	"regexp"
	"slices"
	"strings"

	"go.opentelemetry.io/otel/attribute"

	"github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka/metrics"
)

var (
	forBrokerMetricRegex = regexp.MustCompile("(.*)-for-broker-(.*)")
	forTopicMetricRegex  = regexp.MustCompile("(.*)-for-topic-(.*)")

	ignoreMetrics = []string{
		"batch-size", // we have batch-size-for-topic

		"consumer-batch-size",   // we have batch-size-for-topic
		"consumer-fetch-rate",   // we have for topic metric
		"incoming-byte-rate",    // we have for broker metric
		"outgoing-byte-rate",    // we have for broker metric
		"record-send-rate",      // we have for broker metric
		"request-latency-in-ms", // we have for broker metric
		"request-size",          // we have for broker metric
		"request-rate-total",    // we have for broker metric
		"records-per-request",   // we have for topic metric
		"requests-in-flight",    // we have for broker metric
		"response-rate",         // we have for broker metric
		"response-size",         // we have for broker metric
	}

	ingorePrefixes = []string{
		"protocol-requests-rate", // too low level, we don't need it for now
		"compression-",           // don't care
	}
)

func SaramaMetricRenamer(role string) metrics.TransformMetricsNameToOtel {
	return func(name string) metrics.TransformedMetric {
		res := metrics.TransformedMetric{
			Name: "sarama." + name,
		}

		if slices.Contains(ignoreMetrics, name) {
			res.Drop = true
			return res
		}

		for _, prefix := range ingorePrefixes {
			if strings.HasPrefix(name, prefix) {
				res.Drop = true
				return res
			}
		}

		attributes := []attribute.KeyValue{
			attribute.String("role", role),
		}

		if matches := forBrokerMetricRegex.FindStringSubmatch(name); len(matches) == 3 {
			res.Name = "sarama." + matches[1] + "_for_broker"

			attributes = append(attributes, attribute.String("broker_id", matches[2]))

			res.Attributes = attribute.NewSet(attributes...)
			return res
		}

		if matches := forTopicMetricRegex.FindStringSubmatch(name); len(matches) == 3 {
			res.Name = "sarama." + matches[1] + "_for_topic"

			attributes = append(attributes, attribute.String("topic", matches[2]))

			res.Attributes = attribute.NewSet(attributes...)
			return res
		}

		res.Attributes = attribute.NewSet(attributes...)
		return res
	}
}
