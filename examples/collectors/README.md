# Collector examples

The following examples demonstrate how to ingest events from various sources into OpenMeter using collectors.

The examples use the custom Benthos distribution in this repository.

- [Database](database/)
- [Generate](generate/) random events
- [HTTP server](http-server/) (forwarding events to OpenMeter)
- [Kubernetes Pod execution time](kubernetes-pod-exec-time/)
- [OpenTelemetry Logs](otel-log/)

## Prerequisites

These examples require a running OpenMeter instance. If you don't have one, you can [sign up for free](https://openmeter.cloud/sign-up).

If you are using OpenMeter Cloud, [grab a new API token](https://openmeter.cloud/ingest).

### Create a meter

In order to see data in OpenMeter, you need to create a meter first.

In OpenMeter Cloud, go to the [Meters](https://openmeter.cloud/meters) page and click the **Create meter** button in the right upper corner.
Fill in the details of the meter as instructed by the specific example and click **Create**.

> [!TIP]
> You can start ingesting events without creating a meter first, but you won't be able to query data.
> You can inspect the ingested events in the [Event debugger](https://openmeter.cloud/ingest/debug).

In a self-hosted OpenMeter instance you can create a meter in the configuration file:

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
```

## Checking events in OpenMeter

Once you start ingesting events, you can check them in the [Event debugger](https://openmeter.cloud/ingest/debug) or in the [Query](https://openmeter.cloud/query) page.

If you self-host OpenMeter, you can use the [REST API](https://openmeter.io/docs/getting-started/rest-api) to query data.

## Production use

We are actively working on improving the documentation and the examples.
In the meantime, feel free to contact us [in email](https://us10.list-manage.com/contact-form?u=c7d6a96403a0e5e19032ee885&form_id=fe04a7fc4851f8547cfee56763850e95) or [on Discord](https://discord.gg/nYH3ZQ3Xzq).

We are more than happy to help you set up OpenMeter in your production environment.
