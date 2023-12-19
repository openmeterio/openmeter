# openmeter

![Version: 1.0.0-beta.32](https://img.shields.io/badge/Version-1.0.0--beta.32-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: v1.0.0-beta.32](https://img.shields.io/badge/AppVersion-v1.0.0--beta.32-informational?style=flat-square)

Usage Metering for AI, DevOps, and Billing. Built for engineers to collect and aggregate millions of events in real-time.

**Homepage:** <https://openmeter.io>

## Source Code

* <https://github.com/openmeterio/openmeter>

## Requirements

| Repository | Name | Version |
|------------|------|---------|
| https://docs.altinity.com/clickhouse-operator/ | altinity-clickhouse-operator | 0.22.1 |
| https://strimzi.io/charts/ | strimzi-kafka-operator | 0.38.0 |

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
| kafka.enabled | bool | `true` | Specifies whether Kafka (using the [Kafka Operator](https://github.com/strimzi/strimzi-kafka-operator)) should be installed. **Not recommended for production environments.** |
| kafka.operator.install | bool | `true` | Specifies whether [Kafka Operator](https://github.com/strimzi/strimzi-kafka-operator) should be installed. **Not recommended for production environments.** |
| clickhouse.enabled | bool | `true` | Specifies whether Clickhouse (using the [Clickhouse Operator](https://github.com/Altinity/clickhouse-operator)) should be installed. **Not recommended for production environments.** |
| clickhouse.operator.install | bool | `true` | Specifies whether [Clickhouse Operator](https://github.com/Altinity/clickhouse-operator) should be installed. **Not recommended for production environments.** |
| api.replicaCount | int | `1` | Number of API replicas (pods) to launch. |
| sinkWorker.replicaCount | int | `1` | Number of Sink Worker replicas (pods) to launch. |

----------------------------------------------
Autogenerated from chart metadata using [helm-docs v1.11.2](https://github.com/norwoodj/helm-docs/releases/v1.11.2)
