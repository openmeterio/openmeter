FROM --platform=$BUILDPLATFORM tonistiigi/xx:1.2.1@sha256:8879a398dedf0aadaacfbd332b29ff2f84bc39ae6d4e9c0a1109db27ac5ba012 AS xx

FROM --platform=$BUILDPLATFORM golang:1.20.5-alpine3.18@sha256:b036c52b3bcc8e4e31be19a7a902bb9897b2bf18028f40fd306a9778bab5771c AS builder

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

# See https://github.com/confluentinc/confluent-kafka-go#librdkafka
# See https://github.com/confluentinc/confluent-kafka-go#static-builds-on-linux
RUN go build -ldflags '-linkmode external -extldflags "-static"' -tags musl -o /usr/local/bin/openmeter .
RUN xx-verify /usr/local/bin/openmeter

FROM gcr.io/distroless/base-debian11:latest@sha256:73deaaf6a207c1a33850257ba74e0f196bc418636cada9943a03d7abea980d6d AS distroless

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=builder /usr/local/bin/openmeter /usr/local/bin/
COPY --from=builder /usr/local/src/openmeter/go.* /usr/local/src/openmeter/

CMD openmeter serve

FROM redhat/ubi8-micro:8.8@sha256:c743e8d6f673f8287a07e3590cbf65dfa7c5c21bb81df6dbd4d9a2fcf21173cd AS ubi8

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=builder /usr/local/bin/openmeter /usr/local/bin/
COPY --from=builder /usr/local/src/openmeter/go.* /usr/local/src/openmeter/

CMD openmeter serve

FROM alpine:3.18.2@sha256:82d1e9d7ed48a7523bdebc18cf6290bdb97b82302a8a9c27d4fe885949ea94d1 AS alpine

RUN apk add --update --no-cache ca-certificates tzdata bash

SHELL ["/bin/bash", "-c"]

COPY --from=builder /usr/local/bin/openmeter /usr/local/bin/
COPY --from=builder /usr/local/src/openmeter/go.* /usr/local/src/openmeter/

CMD openmeter serve
