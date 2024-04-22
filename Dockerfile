FROM --platform=$BUILDPLATFORM tonistiigi/xx:1.4.0@sha256:0cd3f05c72d6c9b038eb135f91376ee1169ef3a330d34e418e65e2a5c2e9c0d4 AS xx

FROM --platform=$BUILDPLATFORM golang:1.22.2-alpine3.18@sha256:d995eb689a0c123590a3d34c65f57f3a118bda3fa26f92da5e089ae7d8fd81a0 AS builder

COPY --from=xx / /

RUN apk add --update --no-cache ca-certificates make git curl clang lld

ARG TARGETPLATFORM

RUN xx-apk --update --no-cache add musl-dev gcc

RUN xx-go --wrap

WORKDIR /usr/local/src/openmeter

ARG GOPROXY

ENV CGO_ENABLED=1

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION

# See https://github.com/confluentinc/confluent-kafka-go#librdkafka
# See https://github.com/confluentinc/confluent-kafka-go#static-builds-on-linux
# Build server binary (default)
RUN go build -ldflags "-linkmode external -extldflags \"-static\" -X main.version=${VERSION}" -tags musl -o /usr/local/bin/openmeter ./cmd/server
RUN xx-verify /usr/local/bin/openmeter

# Build sink-worker binary
RUN go build -ldflags "-linkmode external -extldflags \"-static\" -X main.version=${VERSION}" -tags musl -o /usr/local/bin/openmeter-sink-worker ./cmd/sink-worker
RUN xx-verify /usr/local/bin/openmeter-sink-worker

FROM gcr.io/distroless/base-debian11:latest@sha256:280852156756ea3f39f9e774a30346f2e756244e1f432aea3061c4ac85d90a66 AS distroless

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=builder /usr/local/bin/openmeter /usr/local/bin/
COPY --from=builder /usr/local/bin/openmeter-sink-worker /usr/local/bin/
COPY --from=builder /usr/local/src/openmeter/go.* /usr/local/src/openmeter/

CMD openmeter

FROM redhat/ubi8-micro:8.9-15@sha256:d0b7ee00c983167b3d77c7df2b89bb51f47cbc6de356bc3d0e59a07df910f5b4 AS ubi8

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=builder /usr/local/bin/openmeter /usr/local/bin/
COPY --from=builder /usr/local/bin/openmeter-sink-worker /usr/local/bin/
COPY --from=builder /usr/local/src/openmeter/go.* /usr/local/src/openmeter/

CMD openmeter

FROM alpine:3.19.1@sha256:c5b1261d6d3e43071626931fc004f70149baeba2c8ec672bd4f27761f8e1ad6b AS alpine

RUN apk add --update --no-cache ca-certificates tzdata bash

SHELL ["/bin/bash", "-c"]

COPY --from=builder /usr/local/bin/openmeter /usr/local/bin/
COPY --from=builder /usr/local/bin/openmeter-sink-worker /usr/local/bin/
COPY --from=builder /usr/local/src/openmeter/go.* /usr/local/src/openmeter/

CMD openmeter
