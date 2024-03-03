# OpenTelemetry Log

This example demonstrates ingesting logs into OpenMeter using the OTLP protocol.

## Table of Contents <!-- omit from toc -->

- [Prerequisites](#prerequisites)
- [Launch the example](#launch-the-example)
- [Checking events](#checking-events)
- [Cleanup](#cleanup)
- [Advanced configuration](#advanced-configuration)
- [Production use](#production-use)

## Prerequisites

This example uses [Docker](https://docker.com) and [Docker Compose](https://docs.docker.com/compose/), but you are free to run the components in any other way.

Check out this repository if you want to run the example locally:

```shell
git clone https://github.com/openmeterio/openmeter.git
cd openmeter/examples/collectors/otel-log
```

Create a new `.env` file and add the details of your OpenMeter instance:

```shell
cp .env.dist .env
# edit .env and fill in the details
```

> [!TIP]
> Tweak other options in the `.env` file to change the behavior of the example.

[<kbd> <br> Create a meter <br> </kbd>](https://openmeter.cloud/meters/create?meter=%7B%22slug%22%3A%22api_calls%22%2C%22eventType%22%3A%22api-calls%22%2C%22valueProperty%22%3A%22%24.duration_ms%22%2C%22aggregation%22%3A%22SUM%22%2C%22windowSize%22%3A%22MINUTE%22%2C%22groupBy%22%3A%5B%7B%22name%22%3A%22method%22%7D%2C%7B%22name%22%3A%22path%22%7D%2C%7B%22name%22%3A%22region%22%7D%2C%7B%22name%22%3A%22zone%22%7D%5D%7D&utm_source=github&utm_medium=link&utm_content=collectors)
using the button or [manually](https://openmeter.cloud/meters/create) with the following details:

- Event type: `api-calls`
- Aggregation: `SUM`
- Value property: `$.duration_ms`
- Group by (optional):
  - `method`: `$.method`
  - `path`: `$.path`
  - `region`: `$.region`
  - `zone`: `$.zone`

<details><summary><i>Configuration for self-hosted OpenMeter</i></summary><br>

```yaml
# ...

meters:
  - slug: api_calls
    eventType: api-calls
    aggregation: SUM
    valueProperty: $.duration_ms
    groupBy:
      method: $.method
      path: $.path
      region: $.region
      zone: $.zone
```
</details>

> [!TIP]
> Read more about creating a meters in the [documentation](https://openmeter.io/docs/getting-started/meters).

## Launch the example

Launch the example (event collector and seeder):

```shell
docker compose up -d
```

## Checking events

Read more in the collector examples [README](../README.md#Checking-events-in-OpenMeter).

## Cleanup

Stop containers:

```shell
docker compose down -v
```

## Advanced configuration

Check out the configuration files and the [Benthos documentation](https://www.benthos.dev/docs/about) for more details.

## Production use

We are actively working on improving the documentation and the examples.
In the meantime, feel free to contact us [in email](https://us10.list-manage.com/contact-form?u=c7d6a96403a0e5e19032ee885&form_id=fe04a7fc4851f8547cfee56763850e95) or [on Discord](https://discord.gg/nYH3ZQ3Xzq).

We are more than happy to help you set up OpenMeter in your production environment.
