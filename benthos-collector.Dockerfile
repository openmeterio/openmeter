FROM --platform=$BUILDPLATFORM golang:1.25.5-alpine3.23@sha256:ac09a5f469f307e5da71e766b0bd59c9c49ea460a528cc3e6686513d64a6f1fb AS builder

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

FROM alpine:3.23.4@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11

RUN apk add --update --no-cache ca-certificates tzdata bash

SHELL ["/bin/bash", "-c"]

# This is so we can reuse presets in development
WORKDIR /etc/benthos

COPY cloudevents.spec.json /etc/benthos/

COPY collector/benthos/presets /etc/benthos/presets

COPY --link --from=builder /usr/local/bin/benthos /usr/local/bin/
COPY --link --from=builder /src/entrypoint.sh /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh", "/usr/local/bin/benthos"]
