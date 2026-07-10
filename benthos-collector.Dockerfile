FROM --platform=$BUILDPLATFORM golang:1.26.4-alpine3.23@sha256:18b460dd17542c2ba43299a633cf6ebfc1115101509531471d7cfce1019af083 AS builder

RUN apk add --update --no-cache ca-certificates make git curl

ARG TARGETPLATFORM

WORKDIR /src

ARG GOPROXY

ENV CGO_ENABLED=0

ENV GOCACHE=/go/cache
ENV GOMODCACHE=/go/pkg/mod

COPY --link go.mod go.sum ./
# go.mod replaces the nested v3 SDK module with this local path, so its
# manifests must exist before `go mod download` can resolve the module graph.
COPY --link api/v3/client/go.mod api/v3/client/go.sum ./api/v3/client/

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/go/cache \
    go mod download -x

COPY --link collector/go.mod collector/go.sum ./collector/

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/go/cache \
    go mod download -C ./collector -x

ARG VERSION

COPY --link . .

RUN chmod +x entrypoint.sh

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/go/cache \
    go build -C ./collector -ldflags "-X main.version=${VERSION}" -o /usr/local/bin/benthos ./cmd

FROM alpine:3.24.1@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b

RUN apk add --update --no-cache ca-certificates tzdata bash

SHELL ["/bin/bash", "-c"]

# This is so we can reuse presets in development
WORKDIR /etc/benthos

COPY cloudevents.spec.json /etc/benthos/

COPY collector/benthos/presets /etc/benthos/presets

COPY --link --from=builder /usr/local/bin/benthos /usr/local/bin/
COPY --link --from=builder /src/entrypoint.sh /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh", "/usr/local/bin/benthos"]
