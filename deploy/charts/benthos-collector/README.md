# benthos-collector

![type: application](https://img.shields.io/badge/type-application-informational?style=flat-square)  [![artifact hub](https://img.shields.io/badge/artifact%20hub-benthos--collector-informational?style=flat-square)](https://artifacthub.io/packages/helm/openmeter/benthos-collector)

A Benthos-based collector for OpenMeter

**Homepage:** <https://openmeter.io>

## TL;DR;

```bash
helm install --generate-name --wait oci://ghcr.io/openmeterio/helm-charts/benthos-collector
```

to install a specific version:

```bash
helm install --generate-name --wait oci://ghcr.io/openmeterio/helm-charts/benthos-collector --version $VERSION
```

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| image.repository | string | `"ghcr.io/openmeterio/benthos-collector"` | Name of the image repository to pull the container image from. |
| image.pullPolicy | string | `"IfNotPresent"` | [Image pull policy](https://kubernetes.io/docs/concepts/containers/images/#updating-images) for updating already existing images on a node. |
| image.tag | string | `""` | Image tag override for the default value (chart appVersion). |
| openmeter.url | string | `"https://openmeter.cloud"` | OpenMeter API URL |
| openmeter.token | string | `""` | OpenMeter token |
| config | object | `{}` | Benthos configuration Takes precedence over `configFile` and `preset`. |
| configFile | string | `""` | Use an existing config file mounted via `volumes` and `volumeMounts`. Takes precedence over `preset`. |
| preset | string | `""` | Use one of the predefined presets. Note: Read the documentation for the specific preset (example) to learn about configuration via env vars. |
| service | object | `{"annotations":{},"enabled":false,"port":80,"type":"ClusterIP"}` | Service configuration |
| service.enabled | bool | `false` | Specifies whether a service should be created |
| service.type | string | `"ClusterIP"` | Service type |
| service.port | int | `80` | Service port |
| service.annotations | object | `{}` | Annotations to add to the service |
| imagePullSecrets | list | `[]` | Reference to one or more secrets to be used when [pulling images](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/#create-a-pod-that-uses-your-secret) (from private registries). |
| nameOverride | string | `""` | A name in place of the chart name for `app:` labels. |
| fullnameOverride | string | `""` | A name to substitute for the full names of resources. |
| serviceAccount.create | bool | `true` | Specifies whether a service account should be created. |
| serviceAccount.automount | bool | `true` | Automatically mount a ServiceAccount's API credentials? |
| serviceAccount.annotations | object | `{}` | Annotations to add to the service account. |
| serviceAccount.name | string | `""` | The name of the service account to use. If not set and create is true, a name is generated using the fullname template |
| leaderElection.enabled | bool | `false` | Specifies whether leader election should be enabled. |
| leaderElection.lease.duration | string | `"10s"` | Duration of the lease. |
| leaderElection.lease.renewDeadline | string | `"5s"` | Renew deadline of the lease. |
| leaderElection.lease.retryPeriod | string | `"2s"` | Retry period of the lease. |
| rbac.create | bool | `true` | Specifies whether RBAC resources should be created. If disabled, the operator is responsible for creating the necessary resources based on the templates. |
| podAnnotations | object | `{}` | Annotations to be added to pods. |
| podLabels | object | `{}` | Labels to be added to pods. |
| podSecurityContext | object | `{}` | Pod [security context](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#set-the-security-context-for-a-pod). See the [API reference](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#security-context) for details. |
| securityContext | object | `{}` | Container [security context](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#set-the-security-context-for-a-container). See the [API reference](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#security-context-1) for details. |
| resources | object | No requests or limits. | Container resource [requests and limits](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/). See the [API reference](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#resources) for details. |
| storage | object | `{"accessModes":["ReadWriteOnce"],"annotations":{},"enabled":false,"labels":{},"mountPath":"/data","selector":{},"size":"1Gi","storageClass":""}` | Configuration for the PersistentVolumeClaim, which controls the storage for the Collector. |
| storage.enabled | bool | `false` | Enable a PersistentVolumeClaim for the StatefulSet. |
| storage.annotations | object | `{}` | Annotations to add to the PersistentVolumeClaim. |
| storage.labels | object | `{}` | Labels to add to the PersistentVolumeClaim. |
| storage.selector | object | `{}` | Selector for the PersistentVolumeClaim. |
| storage.accessModes | list | `["ReadWriteOnce"]` | Access modes for the PersistentVolumeClaim. |
| storage.size | string | `"1Gi"` | Size of the PersistentVolumeClaim. |
| storage.mountPath | string | `"/data"` | Mount path for the PersistentVolumeClaim. |
| storage.storageClass | string | `""` | Storage class for the PersistentVolumeClaim. |
| volumes | list | `[]` | Additional volumes on the output State definition. |
| volumeMounts | list | `[]` | Additional volumeMounts on the output State definition. |
| envFrom | list | `[]` | Additional environment variables mounted from [secrets](https://kubernetes.io/docs/concepts/configuration/secret/#using-secrets-as-environment-variables) or [config maps](https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/#configure-all-key-value-pairs-in-a-configmap-as-container-environment-variables). See the [API reference](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#environment-variables) for details. |
| env | object | `{}` | Additional environment variables passed directly to containers. See the [API reference](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#environment-variables) for details. |
| nodeSelector | object | `{}` | [Node selector](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#nodeselector) configuration. |
| tolerations | list | `[]` | [Tolerations](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/) for node taints. See the [API reference](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#scheduling) for details. |
| affinity | object | `{}` | [Affinity](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#affinity-and-anti-affinity) configuration. See the [API reference](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#scheduling) for details. |
| caRootCertificates | object | `{}` | List of CA Root certificates to inject into pods at runtime. See [values.yaml](values.yaml) |
