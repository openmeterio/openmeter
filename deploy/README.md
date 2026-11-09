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

Once the `openmeter-api` pod is ready, we can use `port-forward` to access the API.
This might take a few minutes.

```shell
kubectl port-forward svc/openmeter-api 8888:80
```

## Custom root certificates

Both Helm charts accept PEM-encoded roots through `caRootCertificates`. For example, add
`--set-file caRootCertificates.company=company-root.crt` to your Helm command.
On pod startup, a non-root init container combines those roots with the ones shipped in
the application image. Updating the value rolls the affected pods.

For plain Docker or Compose, replace any custom CA directory mount with a read-only
file mount at `/etc/ssl/certs/ca-certificates.crt`. Prepare the bundle from the same
image tag you deploy and rebuild it when the image or your roots change.

```sh
docker run --rm --entrypoint cat "$IMAGE" /etc/ssl/certs/ca-certificates.crt > combined.pem
printf '\n' >> combined.pem
cat company-root.crt >> combined.pem
```

Use `./combined.pem:/etc/ssl/certs/ca-certificates.crt:ro` in Compose. Give UID/GID
10001 access to any writable data or log directories you bind-mount.
