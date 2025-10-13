FROM --platform=$BUILDPLATFORM golang:1.25.2-alpine3.21@sha256:0ae17b3ad9583fcc9c2b195d12f2aa5dd1c18380d3827bd1a81c6e52aded353c AS builder

RUN apk add --update --no-cache ca-certificates make git curl

ARG TARGETPLATFORM

WORKDIR /src

ARG GOPROXY

ENV CGO_ENABLED=0

ENV GOCACHE=/go/cache
ENV GOMODCACHE=/go/pkg/mod

COPY --link go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/go/cache \
    go mod download -x

ARG VERSION

COPY --link . .

RUN chmod +x entrypoint.sh

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/go/cache \
    go build -ldflags "-X main.version=${VERSION}" -o /usr/local/bin/benthos ./cmd/benthos-collector

FROM alpine:3.22.1@sha256:4bcff63911fcb4448bd4fdacec207030997caf25e9bea4045fa6c8c44de311d1

RUN apk add --update --no-cache ca-certificates tzdata bash

SHELL ["/bin/bash", "-c"]

# This is so we can reuse presets in development
WORKDIR /etc/benthos

COPY cloudevents.spec.json /etc/benthos/

COPY collector/benthos/presets /etc/benthos/presets

COPY --link --from=builder /usr/local/bin/benthos /usr/local/bin/
COPY --link --from=builder /src/entrypoint.sh /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh", "/usr/local/bin/benthos"]
