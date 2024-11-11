# openmeter

![type: application](https://img.shields.io/badge/type-application-informational?style=flat-square)  [![artifact hub](https://img.shields.io/badge/artifact%20hub-openmeter-informational?style=flat-square)](https://artifacthub.io/packages/helm/openmeter/openmeter)

Usage Metering for AI, DevOps, and Billing. Built for engineers to collect and aggregate millions of events in real-time.

**Homepage:** <https://openmeter.io>

## Requirements

| Repository | Name | Version |
|------------|------|---------|
| https://docs.altinity.com/clickhouse-operator/ | altinity-clickhouse-operator | 0.23.3 |
| oci://registry-1.docker.io/bitnamicharts | kafka | 30.1.8 |
| oci://registry-1.docker.io/bitnamicharts | postgresql | 16.1.2 |
| oci://registry-1.docker.io/bitnamicharts | redis | 20.2.1 |

## TL;DR;

```bash
helm install --generate-name --wait oci://ghcr.io/openmeterio/helm-charts/openmeter
```

to install a specific version:

```bash
helm install --generate-name --wait oci://ghcr.io/openmeterio/helm-charts/openmeter --version $VERSION
```

## Getting started

OpenMeter needs meters to be defined in configuration to process events:

```yaml
config:
  meters:
    - slug: m1
      description: API calls
      eventType: api-calls
      valueProperty: $.duration_ms
      aggregation: SUM
      groupBy:
        method: $.method
        path: $.path
```

See [values.example.yaml](values.example.yaml) for more details.

> Values defined in `config` will get overwritten by the values calculated from chart values!

## Running OpenMeter in production

This Helm chart comes with a default setups for Kafka, ClickHouse, Postgres, Redis and Svix.

**It is highly recommended to use your own dependencies in production.**

You can disable installing the above dependencies to bring your own:

```yaml
svix:
  enabled: false

redis:
  enabled: false

postgres:
  enabled: false

kafka:
  enabled: false

clickhouse:
  enabled: false
```

In this case, you need to provide the connection details in `config`:

```yaml
config:
    ingest:
      kafka:
        broker: KAFKA_ADDRESS

    aggregation:
      clickhouse:
        address: CLICKHOUSE_ADDRESS
        username: default
        password: ""
        database: default

    postgres:
      url: postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable
      autoMigrate: migration

    svix:
      apiKey: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdml4LXNlcnZlciIsInN1YiI6Im9yZ18yM3JiOFlkR3FNVDBxSXpwZ0d3ZFhmSGlyTXUiLCJleHAiOjE4OTM0NTYwMDAsIm5iZiI6MTcwNDA2NzIwMCwiaWF0IjoxNzIzNTUzMTQ0fQ.JVOFgHymisTD-Zw_p03qD4iUXXXw-VwABda2Q3f1wfs
      serverURL: http://127.0.0.1:8071/
      debug: true
```

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| image.repository | string | `"ghcr.io/openmeterio/openmeter"` | Name of the image repository to pull the container image from. |
| image.pullPolicy | string | `"IfNotPresent"` | [Image pull policy](https://kubernetes.io/docs/concepts/containers/images/#updating-images) for updating already existing images on a node. |
| image.tag | string | `""` | Image tag override for the default value (chart appVersion). |
| imagePullSecrets | list | `[]` | Reference to one or more secrets to be used when [pulling images](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/#create-a-pod-that-uses-your-secret) (from private registries). |
| nameOverride | string | `""` | A name in place of the chart name for `app:` labels. |
| fullnameOverride | string | `""` | A name to substitute for the full names of resources. |
| serviceAccount.create | bool | `true` | Specifies whether a service account should be created. |
| serviceAccount.automount | bool | `true` | Automatically mount a ServiceAccount's API credentials? |
| serviceAccount.annotations | object | `{}` | Annotations to add to the service account. |
| serviceAccount.name | string | `""` | The name of the service account to use. If not set and create is true, a name is generated using the fullname template |
| podAnnotations | object | `{}` | Annotations to be added to pods. |
| podLabels | object | `{}` | Labels to be added to pods. |
| podSecurityContext | object | `{}` | Pod [security context](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#set-the-security-context-for-a-pod). See the [API reference](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#security-context) for details. |
| securityContext | object | `{}` | Container [security context](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#set-the-security-context-for-a-container). See the [API reference](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#security-context-1) for details. |
| service.type | string | `"ClusterIP"` | Kubernetes [service type](https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types). |
| service.port | int | `80` | Service port. |
| ingress.enabled | bool | `false` | Enable [ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/). |
| ingress.className | string | `""` | Ingress [class name](https://kubernetes.io/docs/concepts/services-networking/ingress/#ingress-class). |
| ingress.annotations | object | `{}` | Annotations to be added to the ingress. |
| ingress.hosts | list | See [values.yaml](values.yaml). | Ingress host configuration. |
| ingress.tls | list | See [values.yaml](values.yaml). | Ingress TLS configuration. |
| resources | object | No requests or limits. | Container resource [requests and limits](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/). See the [API reference](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#resources) for details. |
| nodeSelector | object | `{}` | [Node selector](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#nodeselector) configuration. |
| tolerations | list | `[]` | [Tolerations](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/) for node taints. See the [API reference](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#scheduling) for details. |
| affinity | object | `{}` | [Affinity](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#affinity-and-anti-affinity) configuration. See the [API reference](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#scheduling) for details. |
| config | object | `{}` | OpenMeter configuration |
| init | object | `{"busybox":{"tag":"1.37.0"}}` | Defines parameters for the InitContainers |
| kafka.enabled | bool | `true` | Specifies whether Kafka (using the [Bitnami Kafka](oci://registry-1.docker.io/bitnamicharts)) should be installed. **Not recommended for production environments.** |
| kafka.listeners.client.name | string | `"plain"` |  |
| kafka.listeners.client.containerPort | int | `9092` |  |
| kafka.listeners.client.protocol | string | `"PLAINTEXT"` |  |
| kafka.listeners.client.sslClientAuth | string | `""` |  |
| kafka.listeners.controller.name | string | `"CONTROLLER"` |  |
| kafka.listeners.controller.containerPort | int | `9093` |  |
| kafka.listeners.controller.protocol | string | `"SASL_PLAINTEXT"` |  |
| kafka.listeners.controller.sslClientAuth | string | `""` |  |
| clickhouse.enabled | bool | `true` | Specifies whether Clickhouse (using the [Clickhouse Operator](https://github.com/Altinity/clickhouse-operator)) should be installed. **Not recommended for production environments.** |
| clickhouse.operator.install | bool | `true` | Specifies whether [Clickhouse Operator](https://github.com/Altinity/clickhouse-operator) should be installed. **Not recommended for production environments.** |
| svix.enabled | bool | `true` |  |
| svix.signingSecret | string | `"CeRc6WK8KjzRXrKkd9YFnSWcNyqLSIY8JwiaCeRc6WK4UkM"` | Specifies the JWT secret SVIX uses for authentication. For details, [see](https://github.com/svix/svix-webhooks?tab=readme-ov-file#authentication). **It is recommended to change this before deployment.** |
| svix.signedJwt | string | `"eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJzdml4LXNlcnZlciIsImlhdCI6MTczMDk4NzU0MywiZXhwIjoyMjM1OTA5MDAwLCJhdWQiOiIiLCJzdWIiOiJvcmdfMjNyYjhZZEdxTVQwcUl6cGdHd2RYZkhpck11In0.J90SzVuNwecyCtWAQlOoWplJaK4rnIb3rCWXrHQPqJY"` | Specifies the JWT OpenMeter uses to authenticate with Svix. This is an example Token intended for development purposes. You should create your own using the instructions in [svix documentation](https://github.com/svix/svix-webhooks?tab=readme-ov-file#authentication). |
| svix.replicaCount | int | `1` | Number of replicas (pods) to launch. |
| svix.image.repository | string | `"docker.io/svix/svix-server"` | Name of the image repository to pull the container image from. |
| svix.image.pullPolicy | string | `"IfNotPresent"` | [Image pull policy](https://kubernetes.io/docs/concepts/containers/images/#updating-images) for updating already existing images on a node. |
| svix.image.tag | string | `"v1.37"` | Image tag to use |
| svix.service.type | string | `"ClusterIP"` | Kubernetes [service type](https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types). |
| svix.service.port | int | `80` | Service port. |
| svix.resources | object | No requests or limits. | Container resource [requests and limits](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/). See the [API reference](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#resources) for details. |
| redis.enabled | bool | `true` | Specifies whether Redis (using the [Bitnami Redis](oci://registry-1.docker.io/bitnamicharts/redis) chart) should be installed. **Not recommended for production environments.** All further values can be configured in the `redis` section. |
| redis.auth.enabled | bool | `false` |  |
| redis.nameOverride | string | `"redis"` |  |
| postgresql.enabled | bool | `true` | Specifies whether Postgres (using the [Bitnami Chart](https://github.com/bitnami/charts/tree/main/bitnami/postgresql) should be installed. **Not recommended for production environments.** All further values can be configured in the `postgres` section. |
| postgresql.nameOverride | string | `"postgres"` |  |
| postgresql.primary.initdb.scripts."setup.sql" | string | `"CREATE USER application WITH PASSWORD 'application';\nCREATE DATABASE application;\nGRANT ALL PRIVILEGES ON DATABASE application TO application;\nALTER DATABASE application OWNER TO application;\nCREATE USER svix WITH PASSWORD 'svix';\nCREATE DATABASE svix;\nGRANT ALL PRIVILEGES ON DATABASE svix TO svix;\nALTER DATABASE svix OWNER TO svix;\n"` |  |
| api.replicaCount | int | `1` | Number of API replicas (pods) to launch. |
| balanceWorker.replicaCount | int | `1` | Number of Balance Worker replicas (pods) to launch. |
| notificationService.replicaCount | int | `1` | Number of Notification Service replicas (pods) to launch. |
| sinkWorker.replicaCount | int | `1` | Number of Sink Worker replicas (pods) to launch. |
