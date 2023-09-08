# Deploy All-In-One OpenMeter to a *local* Kubernetes cluster

The Helm Chart is only suited for development environments.

## Prerequisites

- [docker](https://www.docker.com/)
- [kind](https://kind.sigs.k8s.io/)
- [helm](https://helm.sh/)

## 1. Setup local cluster

```sh
kind create cluster --config ./kind.yaml
```

## 2. Install OpenMeter via Helm

Then, we're able to install OpenMeter and its dependencies via Helm to the local cluster.

```sh
helm upgrade --install --dependency-update openmeter ./charts/openmeter
```

Once the `openmeter` pod is ready, we can use `port-forward` to access the API.

```sh
kubectl port-forward svc/openmeter 8888
```

See the available values in `./charts/openmeter/values.yaml`.
