# OpenMeter Collector

OpenMeter Collector is a configurable, production-ready data pipeline for **usage metering**. It helps you **collect, transform, buffer, and reliably deliver** usage events into OpenMeter, especially in distributed and network-unreliable environments.

Learn more in the docs:

* Overview: [https://openmeter.io/docs/collectors](https://openmeter.io/docs/collectors)
* Quickstart: [https://openmeter.io/docs/collectors/quickstart](https://openmeter.io/docs/collectors/quickstart)
* How it works: [https://openmeter.io/docs/collectors/how-it-works](https://openmeter.io/docs/collectors/how-it-works)

---

## Capabilities

* **Multiple ingestion sources**: HTTP/event ingestion and a growing set of presets and integrations

  * Kubernetes: [https://openmeter.io/docs/collectors/kubernetes](https://openmeter.io/docs/collectors/kubernetes)
  * Prometheus: [https://openmeter.io/docs/collectors/prometheus](https://openmeter.io/docs/collectors/prometheus)
  * OpenTelemetry: [https://openmeter.io/docs/collectors/otel](https://openmeter.io/docs/collectors/otel)
  * ClickHouse: [https://openmeter.io/docs/collectors/clickhouse](https://openmeter.io/docs/collectors/clickhouse)
  * Postgres: [https://openmeter.io/docs/collectors/postgres](https://openmeter.io/docs/collectors/postgres)
  * S3: [https://openmeter.io/docs/collectors/s3](https://openmeter.io/docs/collectors/s3)
  * NVIDIA Run:ai: [https://openmeter.io/docs/collectors/nvidia-run-ai](https://openmeter.io/docs/collectors/nvidia-run-ai)

* **Reliable delivery with buffering**: disk-backed buffering for network resilience and backpressure handling
  [https://openmeter.io/docs/collectors/buffer](https://openmeter.io/docs/collectors/buffer)

* **High availability patterns**: deploy and operate collectors in HA setups
  [https://openmeter.io/docs/collectors/high-availability](https://openmeter.io/docs/collectors/high-availability)

* **Built-in observability**: metrics and operational visibility for pipelines
  [https://openmeter.io/docs/collectors/observability](https://openmeter.io/docs/collectors/observability)

---

## Architecture

```text
+-------------------+   +-------------------+   +-------------------+
|  App / SDK Events  |   |  Infra Signals     |   |  Data Stores       |
|  (HTTP, webhooks)  |   |  (K8s/Prom/OTEL)   |   |  (PG/CH/S3/...)    |
+---------+---------+   +---------+---------+   +---------+---------+
          \                   |                     /
           \                  |                    /
            \                 |                   /
             v                v                  v
        +------------------------------------------------+
        |                OpenMeter Collector              |
        |------------------------------------------------|
        |  Ingest -> Validate/Transform -> Batch/Retry    |
        |                 |                               |
        |                 v                               |
        |        Disk Buffer (durable replay)             |
        +-----------------+------------------------------+
                          |
                          v
              +------------------------------+
              |          OpenMeter           |
              |   metering • usage • billing |
              +------------------------------+

```

See detailed architecture and concepts: [https://openmeter.io/docs/collectors/how-it-works](https://openmeter.io/docs/collectors/how-it-works)

---

## Getting Started

### 1. Choose a deployment model

* Kubernetes (Helm): [https://openmeter.io/docs/collectors/kubernetes](https://openmeter.io/docs/collectors/kubernetes)
* Other environments and presets: [https://openmeter.io/docs/collectors/quickstart](https://openmeter.io/docs/collectors/quickstart)

### 2. Install the Collector

Follow the official quickstart for step-by-step installation and configuration examples:
[https://openmeter.io/docs/collectors/quickstart](https://openmeter.io/docs/collectors/quickstart)

### 3. Send usage events

Configure your SDKs or usage producers to send events to the Collector endpoint.

### 4. Run in production

* Buffering & delivery guarantees: [https://openmeter.io/docs/collectors/buffer](https://openmeter.io/docs/collectors/buffer)
* High availability: [https://openmeter.io/docs/collectors/high-availability](https://openmeter.io/docs/collectors/high-availability)
* Observability & metrics: [https://openmeter.io/docs/collectors/observability](https://openmeter.io/docs/collectors/observability)

---

## Repository

Collector source code lives in the main OpenMeter repository:
[https://github.com/openmeterio/openmeter/tree/main/collector](https://github.com/openmeterio/openmeter/tree/main/collector)
