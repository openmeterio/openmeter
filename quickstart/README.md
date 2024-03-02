# Quickstart

## Prerequisites

- Docker (with Compose)
- curl
- jq

Clone the repository:

```sh
git clone git@github.com:openmeterio/openmeter.git
cd openmeter/quickstart
```

## 1. Launch OpenMeter

Launch OpenMeter and its dependencies via:

```sh
docker compose up -d
```

## 2. Ingest usage event(s)

Ingest usage events in [CloudEvents](https://cloudevents.io/) format:

```sh
curl -X POST http://localhost:8888/api/v1/events \
-H 'Content-Type: application/cloudevents+json' \
--data-raw '
{
  "specversion" : "1.0",
  "type": "request",
  "id": "00001",
  "time": "2023-01-01T00:00:00.001Z",
  "source": "service-0",
  "subject": "customer-1",
  "data": {
    "method": "GET",
    "route": "/hello"
  }
}
'
```

Note how ID is different:

```sh
curl -X POST http://localhost:8888/api/v1/events \
-H 'Content-Type: application/cloudevents+json' \
--data-raw '
{
  "specversion" : "1.0",
  "type": "request",
  "id": "00002",
  "time": "2023-01-01T00:00:00.001Z",
  "source": "service-0",
  "subject": "customer-1",
  "data": {
    "method": "GET",
    "route": "/hello"
  }
}
'
```

Note how ID and time are different:

```sh
curl -X POST http://localhost:8888/api/v1/events \
-H 'Content-Type: application/cloudevents+json' \
--data-raw '
{
  "specversion" : "1.0",
  "type": "request",
  "id": "00003",
  "time": "2023-01-02T00:00:00.001Z",
  "source": "service-0",
  "subject": "customer-1",
  "data": {
    "method": "GET",
    "route": "/hello"
  }
}
'
```

## 3. Query Usage

Query the usage hourly:

```sh
curl 'http://localhost:8888/api/v1/meters/api_request/query?windowSize=HOUR&groupBy=method&groupBy=route' | jq
```

```json
{
  "windowSize": "HOUR",
  "data": [
    {
      "value": 2,
      "windowStart": "2023-01-01T00:00:00Z",
      "windowEnd": "2023-01-01T01:00:00Z",
      "subject": null,
      "groupBy": {
        "method": "GET",
        "route": "/hello"
      }
    },
    {
      "value": 1,
      "windowStart": "2023-01-02T00:00:00Z",
      "windowEnd": "2023-01-02T01:00:00Z",
      "subject": null,
      "groupBy": {
        "method": "GET",
        "route": "/hello"
      }
    }
  ]
}
```

Query the total usage for `customer-1`:

```sh
curl 'http://localhost:8888/api/v1/meters/api_request/query?subject=customer-1' | jq
```

```json
{
  "data": [
    {
      "value": 3,
      "windowStart": "2023-01-01T00:00:00Z",
      "windowEnd": "2023-01-02T00:01:00Z",
      "subject": "customer-1",
      "groupBy": {}
    }
  ]
}
```

## 4. Configure additional meter(s) _(optional)_

In this example we will meter LLM token usage, groupped by AI model and prompt type.
You can think about it how OpenAI [charges](https://openai.com/pricing) by tokens for ChatGPT.

Configure how OpenMeter should process your usage events in this new `token_usage` meter.

```yaml
# ...

meters:
  # Sample meter to count LLM Token Usage
  - slug: token_usage
    description: AI Token Usage
    eventType: prompt               # Filter events by type
    aggregation: SUM
    valueProperty: $.tokens         # JSONPath to parse usage value
    groupBy:
      model: $.model                # AI model used: gpt4-turbo, etc.
      type: $.type                  # Prompt type: input, output, system

```

## Cleanup

Once you are done, stop any running instances:

```sh
docker compose down -v
```
