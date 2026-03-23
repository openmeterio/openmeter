FROM --platform=$BUILDPLATFORM public.ecr.aws/t3d5i9m2/konnect-public-base-go:1.25@sha256:11f068689a53c7fa4ac75ebbac4c1778989cd2655f587414d2bc772353404728 AS builder

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

FROM public.ecr.aws/t3d5i9m2/konnect-public-base-alpine:3.21@sha256:69921597598341cd859428a4cc6a46ed62bcfd609bfb62fee7c5d7533397c71c

USER root

RUN apk add --update --no-cache ca-certificates tzdata bash

USER kong

SHELL ["/bin/bash", "-c"]

# This is so we can reuse presets in development
WORKDIR /etc/benthos

COPY cloudevents.spec.json /etc/benthos/

COPY collector/benthos/presets /etc/benthos/presets

COPY --link --from=builder /usr/local/bin/benthos /usr/local/bin/
COPY --link --from=builder /src/entrypoint.sh /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh", "/usr/local/bin/benthos"]
