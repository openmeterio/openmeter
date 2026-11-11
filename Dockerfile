FROM --platform=$BUILDPLATFORM tonistiigi/xx:1.10.0@sha256:40fe21f44bb8cddf1eb1226db190d0a984118f62ee02a91b77d99448d28264a6 AS xx

FROM --platform=$BUILDPLATFORM golang:1.27.3-alpine3.23@sha256:5d5113301ffcac84e2bfb20c42648803fecaea6789c89bb9b66884e18cbb30a7 AS builder

COPY --link --from=xx / /

RUN xx-apk add --update --no-cache ca-certificates make git curl clang lld

ARG TARGETPLATFORM

RUN xx-apk --update --no-cache add musl-dev gcc

WORKDIR /src

ARG GOPROXY

ENV CGO_ENABLED=1

ENV GOCACHE=/go/cache
ENV GOMODCACHE=/go/pkg/mod

COPY --link go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/go/cache \
    xx-go mod download -x

ARG VERSION
ARG COMMIT_SHA

COPY --link . .

# See https://github.com/confluentinc/confluent-kafka-go#librdkafka
# See https://github.com/confluentinc/confluent-kafka-go#static-builds-on-linux
# Build server binary (default)
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/go/cache \
    xx-go build -ldflags "-linkmode external -extldflags \"-static\" -X main.version=${VERSION} -X main.revision=${COMMIT_SHA}" -tags musl -o /usr/local/bin/openmeter ./cmd/server

RUN xx-verify /usr/local/bin/openmeter

# Build sink-worker binary
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/go/cache \
    xx-go build -ldflags "-linkmode external -extldflags \"-static\" -X main.version=${VERSION} -X main.revision=${COMMIT_SHA}" -tags musl -o /usr/local/bin/openmeter-sink-worker ./cmd/sink-worker

RUN xx-verify /usr/local/bin/openmeter-sink-worker

# Build balance-worker binary
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/go/cache \
    xx-go build -ldflags "-linkmode external -extldflags \"-static\" -X main.version=${VERSION} -X main.revision=${COMMIT_SHA}" -tags musl -o /usr/local/bin/openmeter-balance-worker ./cmd/balance-worker

RUN xx-verify /usr/local/bin/openmeter-balance-worker

# Build notification-service binary
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/go/cache \
    xx-go build -ldflags "-linkmode external -extldflags \"-static\" -X main.version=${VERSION} -X main.revision=${COMMIT_SHA}" -tags musl -o /usr/local/bin/openmeter-notification-service ./cmd/notification-service

RUN xx-verify /usr/local/bin/openmeter-notification-service

# Build billing-worker binary
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/go/cache \
    xx-go build -ldflags "-linkmode external -extldflags \"-static\" -X main.version=${VERSION} -X main.revision=${COMMIT_SHA}" -tags musl -o /usr/local/bin/openmeter-billing-worker ./cmd/billing-worker

RUN xx-verify /usr/local/bin/openmeter-billing-worker

# Build periodic jobs binary
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/go/cache \
    xx-go build -ldflags "-linkmode external -extldflags \"-static\" -X github.com/openmeterio/openmeter/cmd/jobs/internal.version=${VERSION} -X github.com/openmeterio/openmeter/cmd/jobs/internal.revision=${COMMIT_SHA}" -tags musl -o /usr/local/bin/openmeter-jobs ./cmd/jobs

RUN xx-verify /usr/local/bin/openmeter-jobs

FROM alpine:3.24.3@sha256:be2656061b536c1883c4034790e21d0a09c5aaaf51968b89368a3c0371ea2c9a

RUN apk add --update --no-cache ca-certificates tzdata bash \
    && addgroup -S -g 10001 openmeter \
    && adduser -S -D -H -u 10001 -G openmeter openmeter \
    && mkdir -p /var/run/openmeter-ca \
    && chown openmeter:openmeter /var/run/openmeter-ca

SHELL ["/bin/bash", "-c"]

COPY --link --from=builder /usr/local/bin/openmeter /usr/local/bin/
COPY --link --from=builder /usr/local/bin/openmeter-sink-worker /usr/local/bin/
COPY --link --from=builder /usr/local/bin/openmeter-balance-worker /usr/local/bin/
COPY --link --from=builder /usr/local/bin/openmeter-notification-service /usr/local/bin/
COPY --link --from=builder /usr/local/bin/openmeter-billing-worker /usr/local/bin/
COPY --link --from=builder /usr/local/bin/openmeter-jobs /usr/local/bin/
COPY --link --from=builder /src/go.* /usr/local/src/openmeter/
COPY --link --chmod=755 prepare-ca-bundle.sh /usr/local/bin/prepare-ca-bundle

USER 10001:10001

CMD ["openmeter"]
