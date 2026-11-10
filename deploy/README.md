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

Upgrade the charts and images together. The init container uses the application image
to read its Alpine roots and run the bundle helper.

For Compose, both the [server quickstart](../quickstart/docker-compose.ca.yaml) and
[collector quickstart](../collector/quickstart/docker-compose.ca.yaml) include an optional
override. From the corresponding quickstart directory, run

```sh
export CA_CERTIFICATES_DIR=/absolute/path/to/company-certs
docker compose -f docker-compose.yaml -f docker-compose.ca.yaml up -d
```

A one-shot service prepares the bundle in a named volume. The application services
wait for it to succeed and mount only the bundle file, read-only. Recreate the
helper and consumers with `up -d --force-recreate` after changing the certificates.
The collector override also applies the server override to its included OpenMeter stack.
Set `OPENMETER_IMAGE` and `COLLECTOR_IMAGE` to pin the application and helper to the
same release. The overrides require Compose and Engine support for volume file subpaths.

For plain Docker, bind-mount a combined PEM file read-only at
`/etc/ssl/certs/ca-certificates.crt`. Rebuild it from the deployed image when its
public roots or your custom roots change. Give UID/GID 10001 access to writable
data and log directories. The collector HTTP preset writes SQLite to its data volume (ephemeral
by default, persistent with `storage.enabled`). Plain Docker uses
`/var/lib/collector/buffer.db`. An explicit `BUFFER_PATH` takes precedence.
Drain or migrate an existing SQLite buffer in the old container working directory
before upgrading.
