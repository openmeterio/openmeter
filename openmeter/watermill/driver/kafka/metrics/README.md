# go-metrics to OpenTelemetry adapter

Sarama (any most probably other projects) rely on the [http://github.com/rcrowley/go-metrics](go-metrics) package, numerous connectors exist
for the package, however there seem to be no opentelemetry one.

Given the package only supports periodic scraping, it's better to wrap the metric types of go-metrics so that we can send the raw events to OpenTelemetry. The existing event interface of go-metrics is quite limited compared to OpenTelemetry esp considering the usage in Sarama lib, so right now:

- Context is context.Background() for OpenTelemetry calls
- Errors for metric registration is only logged
