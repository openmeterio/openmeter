# Event Ingestion

OpenMeter ingests usage data through events using the [CloudEvents](https://cloudevents.io/) specification incubated under CNCF.

A usage event can be anything you need to track accurately over time for billing or analytics purposes.
For example, a CI/CD product can include active or parallel jobs, build minutes, network traffic, storage used, or other product-related actions.

## Event Format

As we mentioned in the intro, OpenMeter ingests events using the CloudEvents specification from CNCF.
This event format is abstracted away in many of the cloud infrastructure integrations and SDKs from the end-user.

Here is an example of a CloudEvent describing the execution duration of a serverless application:

```json
{
  "specversion": "1.0",
  "type": "api-calls",
  "id": "00001",
  "time": "2023-01-01T00:00:00.001Z",
  "source": "service-0",
  "subject": "customer-1",
  "data": {
    "duration": "12",
    "path": "/hello"
  }
}
```

OpenMeter currently supports the [JSON format](https://github.com/cloudevents/spec/blob/main/cloudevents/formats/json-format.md) of CloudEvents, with plans to extend support for Protobuf and other formats.
The `data` property can contain any valid JSON object, and when configuring meters using [JsonPath](https://github.com/json-path/JsonPath), individual values can be extracted.

CloudEvents is adopted by many Cloud Native solutions, making it effortless to extract usage data from infrastructure solutions.
SDKs are available for common programming languages, simplifying the creation, validation, and reporting of usage events to OpenMeter.

## Event Processing

OpenMeter continuously processes usage events, allowing you to update meters in real-time. Once an event is ingested, OpenMeter aggregates the data based on your defined meters.
For example, you can define a meter called "Parallel jobs", and OpenMeter will aggregate the maximum number of jobs by each customer over a given time period.

Using an example, let’s dive into how OpenMeter’s event processing works.
Imagine you want to track serverless execution duration by endpoint and you defined the following meter:

```yaml
meters:
  - slug: m1
    type: api-calls
    valueProperty: $.duration
    aggregation: SUM
    groupBy:
      path: $.path
```

The meter config above tells OpenMeter to expect CloudEvents with `type=api-calls` where the usage value is stored in `data.duration` and we need to sum them by `data.path`.
OpenMeter will track the usage value for every time window when at least one event was reported and tracks it for every `subject` and `groupBy` permutation.

Note that `$.path` is a [JsonPath](https://github.com/json-path/JsonPath) expression to access the `data.path` property, providing powerful capabilities to extract values from nested data properties.

For example, when you send your first event:

```json
{
  "specversion": "1.0",
  "type": "api-calls",
  "id": "00001",
  "time": "2023-01-01T00:00:00.001Z",
  "source": "service-0",
  "subject": "customer-1",
  "data": {
    "duration": "10",
    "path": "/hello"
  }
}
```

OpenMeter will track the usage value for the time window and customer as:

```sh
windowstart   = "2023-01-01T00:00"
windowend     = "2023-01-01T00:01"
subject       = "customer-1"
value         = 10
path          = "/hello"
```

When sending the second event (with a different `id` and `duration` value):

```json
{
  "specversion": "1.0",
  "type": "api-calls",
  "id": "00002",
  "time": "2023-01-01T00:00:00.001Z",
  "source": "service-0",
  "subject": "customer-1",
  "data": {
    "duration": "20",
    "path": "/hello"
  }
}
```

OpenMeter will increase sum of the duration for the two events for the same time window (`windowstart`, `windowend`), `$.path` and `subject`:

```sh
windowstart   = "2023-01-01T00:00"
windowend     = "2023-01-01T00:01"
subject       = "customer-1"
value         = 30
path          = "/hello"
```

## Event Deduplication

CloudEvents are unique by `id` and `source`, see CloudEvent's [specification](https://github.com/cloudevents/spec/blob/main/cloudevents/spec.md):

> Producers MUST ensure that `source` + `id` is unique for each distinct event. If a duplicate event is re-sent (e.g. due to a network error) it MAY have the same id. Consumers MAY assume that Events with identical source and id are duplicates.

OpenMeter deduplicates events by id and source, by default, for 32 days. This ensures that if multiple events with the same id and source are sent, they will be processed only once.
This is useful when you want to retry or replay events in your infrastructure.
