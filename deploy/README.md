# Deploy OpenMeter to a local Kubernetes cluster

## 1. Setup Kind cluster

```sh
kind create cluster --config ./kind.yaml
```

An optional step is to install an [Ingress controller](https://kind.sigs.k8s.io/docs/user/ingress/):

```sh
# NGINX
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml

# Wait for the controller to start
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
```

## 2. Install OpenMeter via Helm

Unfortunately, the Confluent Helm charts aren't kept up to date.
The [registry](https://confluentinc.github.io/cp-helm-charts/) itself was not updated since 2020.

A temporary workaround is to clone the [Confluent charts repository](https://github.com/confluentinc/cp-helm-charts) to `/deploy/charts/cp-helm-charts`. It was added as a submodule for now.

```sh
git submodule update --init --recursive
```

Then, we're able to install OpenMeter and its dependencies via Helm to the local cluster.

```sh
helm upgrade --install --dependency-update openmeter ./charts/openmeter
```

Once the `openmeter` pod is ready, we can use `port-forward` to access the API.

```sh
kubectl port-forward svc/openmeter 8888
```

See the available values in `./charts/openmeter/values.yaml`.
