# Kubernetes Pod Execution Time

This example demonstrates metering execution time of Pods running in Kubernetes.

## Table of Contents <!-- omit from toc -->

- [Prerequisites](#prerequisites)
- [Preparations](#preparations)
- [Deploy the example](#deploy-the-example)
- [Checking events](#checking-events)
- [Cleanup](#cleanup)
- [Advanced configuration](#advanced-configuration)
- [Production use](#production-use)

## Prerequisites

Any local (or remote if that's what's available for you) Kubernetes cluster will do.

We will use [kind](https://kind.sigs.k8s.io/) in this example.

Additional tools you are going to need:

- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [helm](https://helm.sh/docs/intro/install/)

## Preparations

Create a new Kubernetes cluster using `kind`:

```shell
kind create cluster
```

> [!TIP]
> Alternatively, set up your `kubectl` context to point to an existing cluster.

Deploy the test Pods to the cluster:

```shell
kubectl apply -f https://raw.githubusercontent.com/openmeterio/openmeter/main/examples/collectors/kubernetes-pod-exec-time/seed/pod.yaml
```

<details><summary><i>Running locally</i></summary><br>

```shell
kubectl apply -f seed/pod.yaml
```
</details>

[<kbd> <br> Create a meter <br> </kbd>](https://openmeter.cloud/meters/create?meter=%7B%22slug%22%3A%22pod_execution_time%22%2C%22eventType%22%3A%22kube-pod-exec-time%22%2C%22valueProperty%22%3A%22%24.duration_seconds%22%2C%22aggregation%22%3A%22SUM%22%2C%22windowSize%22%3A%22MINUTE%22%2C%22groupBy%22%3A%5B%7B%22name%22%3A%22pod_name%22%7D%2C%7B%22name%22%3A%22pod_namespace%22%7D%5D%7D&utm_source=github&utm_medium=link&utm_content=collectors)
using the button or [manually](https://openmeter.cloud/meters/create) with the following details:

- Event type: `kube-pod-exec-time`
- Aggregation: `SUM`
- Value property: `$.duration_seconds`
- Group by (optional):
  - `pod_namespace`: `$.pod_namespace`
  - `pod_name`: `$.pod_name`

<details><summary><i>Configuration for self-hosted OpenMeter</i></summary><br>

```yaml
# ...

meters:
  - slug: pod_execution_time
    eventType: kube-pod-exec-time
    aggregation: SUM
    valueProperty: $.duration_seconds
    groupBy:
      pod_namespace: $.pod_namespace
      pod_name: $.pod_name
```
</details>

> [!TIP]
> Read more about creating a meters in the [documentation](https://openmeter.io/docs/getting-started/meters).

## Deploy the example

Deploy Benthos to your cluster:

```shell
helm install --devel --namespace benthos-collector --create-namespace --set preset=kubernetes-pod-exec-time --set openmeter.url=<OPENMETER_URL> --set openmeter.token=<OPENMETER_TOKEN> benthos-collector oci://ghcr.io/openmeterio/helm-charts/benthos-collector
```

<details><summary><i>Running locally</i></summary><br>

```shell
helm install --devel --namespace benthos-collector --create-namespace --set preset=kubernetes-pod-exec-time --set openmeter.url=$OPENMETER_URL --set openmeter.token=$OPENMETER_TOKEN benthos-collector ../../../deploy/charts/benthos-collector
```
</details>

> [!NOTE]
> If you use OpenMeter Cloud, you can omit the `openmeter.url` parameter.

## Checking events

Read more in the collector examples [README](../README.md#Checking-events-in-OpenMeter).

## Cleanup

Uninstall Benthos from the cluster:

```shell
helm delete --namespace benthos-collector benthos-collector
```

Remove the sample Pods from the cluster:

```shell
kubectl delete -f https://raw.githubusercontent.com/openmeterio/openmeter/main/examples/collectors/kubernetes-pod-exec-time/seed/pod.yaml
```

<details><summary><i>Running locally</i></summary><br>

```shell
kubectl delete -f seed/pod.yaml
```
</details>

Delete the cluster:

```shell
kind delete cluster
```

## Advanced configuration

This example uses a custom Benthos plugin called `kubernetes_resources` (included in this project) to periodically scrape the Kubernetes API for active pods.

The entire pipeline can be found in [this file](/collector/benthos/presets/kubernetes-pod-exec-time/config.yaml).

Check out the configuration file and the [Benthos documentation](https://www.benthos.dev/docs/about) for more details.

## Production use

We are actively working on improving the documentation and the examples.
In the meantime, feel free to contact us [in email](https://us10.list-manage.com/contact-form?u=c7d6a96403a0e5e19032ee885&form_id=fe04a7fc4851f8547cfee56763850e95) or [on Discord](https://discord.gg/nYH3ZQ3Xzq).

We are more than happy to help you set up OpenMeter in your production environment.
