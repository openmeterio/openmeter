FROM --platform=$BUILDPLATFORM golang:1.22.5-alpine3.19@sha256:0642d4f809abf039440540de1f0e83502401686e3946ed8e7398a1d94648aa6d AS builder

RUN apk add --update --no-cache ca-certificates make git curl

ARG TARGETPLATFORM

WORKDIR /usr/local/src/openmeter

ARG GOPROXY

ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION

RUN go build -ldflags "-X main.version=${VERSION}" -o /usr/local/bin/benthos ./cmd/benthos-collector

FROM alpine:3.20.2@sha256:0a4eaa0eecf5f8c050e5bba433f58c052be7587ee8af3e8b3910ef9ab5fbe9f5

RUN apk add --update --no-cache ca-certificates tzdata bash

SHELL ["/bin/bash", "-c"]

# This is so we can reuse presets in development
WORKDIR /etc/benthos

COPY cloudevents.spec.json /etc/benthos/

COPY collector/benthos/presets /etc/benthos/presets

COPY --from=builder /usr/local/bin/benthos /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/benthos"]
