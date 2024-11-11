FROM --platform=$BUILDPLATFORM tonistiigi/xx:1.5.0@sha256:0c6a569797744e45955f39d4f7538ac344bfb7ebf0a54006a0a4297b153ccf0f AS xx

FROM --platform=$BUILDPLATFORM golang:1.23.3-alpine3.20@sha256:09742590377387b931261cbeb72ce56da1b0d750a27379f7385245b2b058b63a AS builder

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

# Build balance-worker binary
RUN go build -ldflags "-linkmode external -extldflags \"-static\" -X main.version=${VERSION}" -tags musl -o /usr/local/bin/openmeter-balance-worker ./cmd/balance-worker
RUN xx-verify /usr/local/bin/openmeter-balance-worker

# Build balance-worker binary
RUN go build -ldflags "-linkmode external -extldflags \"-static\" -X main.version=${VERSION}" -tags musl -o /usr/local/bin/openmeter-notification-service ./cmd/notification-service
RUN xx-verify /usr/local/bin/openmeter-notification-service


FROM alpine:3.20.3@sha256:beefdbd8a1da6d2915566fde36db9db0b524eb737fc57cd1367effd16dc0d06d

RUN apk add --update --no-cache ca-certificates tzdata bash

SHELL ["/bin/bash", "-c"]

COPY --from=builder /usr/local/bin/openmeter /usr/local/bin/
COPY --from=builder /usr/local/bin/openmeter-sink-worker /usr/local/bin/
COPY --from=builder /usr/local/bin/openmeter-balance-worker /usr/local/bin/
COPY --from=builder /usr/local/bin/openmeter-notification-service /usr/local/bin/
COPY --from=builder /usr/local/src/openmeter/go.* /usr/local/src/openmeter/

CMD openmeter
