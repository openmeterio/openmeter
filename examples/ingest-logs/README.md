# Ingest Logs

In this example, we'll parse application logs and convert them into OpenMeter events. This method is beneficial when you want to gather usage information from systems where integrating the OpenMeter SDK or sending HTTP requests isn't feasible. Using this powerful approach, you can track usage and bill for low-level infrastructure components like message queues, logs, and more.

## Example

Here, we replicate our execution duration example from the OpenMeter [quickstart](/quickstart). To collect and transform logs, we'll utilize [Vector](https://vector.dev). In this scenario, we'll tail the logs of a Docker container (named `demologs`) and transform these log messages into OpenMeter-compatible [CloudEvents](https://cloudevents.io/). These are then ingested into OpenMeter via HTTP. The process consists of the following steps:

1. Vector tails logs from the Docker container.
2. Vector filters logs based on our criteria.
3. Vector transforms logs into the CloudEvents format.
4. Vector sends the CloudEvents to the OpenMeter ingest API.
5. OpenMeter tracks usage.

Transformations in Vector uses the [Vector Remap Language](https://vector.dev/docs/reference/vrl/) (VRL). In this example we add, delete and rename object properties with VRL.

### Run The Example

1. First, start OpenMeter. Refer to the [quickstart guide](/quickstart) for instructions.
2. Execute `docker compose up` within this example directory.
3. To query meters, use the following command: `curl 'http://localhost:8888/api/v1/meters/api_requests_duration/query?groupBy=subject'`.

Note: It's important that you run quickstart's `docker compose` first.
