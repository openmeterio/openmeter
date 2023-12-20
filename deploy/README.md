# Deploy OpenMeter

## Deploy OpenMeter to a local Kubernetes cluster

## Prerequisites

- [docker](https://www.docker.com/)
- [kind](https://kind.sigs.k8s.io/)
- [helm](https://helm.sh/)

## 1. Check out this repository

```shell
git clone git@github.com:openmeterio/openmeter.git
cd openmeter/deploy
```

## 2. Setup local cluster

```shell
kind create cluster --config ./kind.yaml
```

## 3. Install OpenMeter via Helm

```shell
helm upgrade --install --dependency-update -f ./charts/openmeter/values.example.yaml openmeter ./charts/openmeter
```

Once the `openmeter` pod is ready, we can use `port-forward` to access the API.

```shell
kubectl port-forward svc/openmeter-api 8888:80
```
