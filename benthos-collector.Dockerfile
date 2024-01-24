FROM --platform=$BUILDPLATFORM golang:1.21.6-alpine3.18@sha256:3bd447580bc0df66bada3d8e38f37ca85faf66d6a0e37f0ccba287eaf5962757 AS builder

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

FROM alpine:3.19.0@sha256:51b67269f354137895d43f3b3d810bfacd3945438e94dc5ac55fdac340352f48

RUN apk add --update --no-cache ca-certificates tzdata bash

SHELL ["/bin/bash", "-c"]

# This is so we can reuse presets in development
WORKDIR /etc/benthos

COPY cloudevents.spec.json /etc/benthos/

COPY collector/benthos/presets /etc/benthos/presets

COPY --from=builder /usr/local/bin/benthos /usr/local/bin/

CMD benthos
