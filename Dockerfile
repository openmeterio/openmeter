FROM --platform=$BUILDPLATFORM tonistiigi/xx:1.6.1@sha256:923441d7c25f1e2eb5789f82d987693c47b8ed987c4ab3b075d6ed2b5d6779a3 AS xx

FROM --platform=$BUILDPLATFORM golang:1.24.6-alpine3.21@sha256:50f8a10a46c0c26b5b816a80314f1999196c44c3e3571f41026b061339c29db6 AS builder

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

COPY --link . .

RUN chmod +x entrypoint.sh

# See https://github.com/confluentinc/confluent-kafka-go#librdkafka
# See https://github.com/confluentinc/confluent-kafka-go#static-builds-on-linux
# Build server binary (default)
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/go/cache \
    xx-go build -ldflags "-linkmode external -extldflags \"-static\" -X main.version=${VERSION}" -tags musl -o /usr/local/bin/openmeter ./cmd/server

RUN xx-verify /usr/local/bin/openmeter

# Build sink-worker binary
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/go/cache \
    xx-go build -ldflags "-linkmode external -extldflags \"-static\" -X main.version=${VERSION}" -tags musl -o /usr/local/bin/openmeter-sink-worker ./cmd/sink-worker

RUN xx-verify /usr/local/bin/openmeter-sink-worker

# Build balance-worker binary
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/go/cache \
    xx-go build -ldflags "-linkmode external -extldflags \"-static\" -X main.version=${VERSION}" -tags musl -o /usr/local/bin/openmeter-balance-worker ./cmd/balance-worker

RUN xx-verify /usr/local/bin/openmeter-balance-worker

# Build notification-service binary
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/go/cache \
    xx-go build -ldflags "-linkmode external -extldflags \"-static\" -X main.version=${VERSION}" -tags musl -o /usr/local/bin/openmeter-notification-service ./cmd/notification-service

RUN xx-verify /usr/local/bin/openmeter-notification-service

# Build billing-worker binary
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/go/cache \
    xx-go build -ldflags "-linkmode external -extldflags \"-static\" -X main.version=${VERSION}" -tags musl -o /usr/local/bin/openmeter-billing-worker ./cmd/billing-worker

RUN xx-verify /usr/local/bin/openmeter-billing-worker

# Build periodic jobs binary
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/go/cache \
    xx-go build -ldflags "-linkmode external -extldflags \"-static\" -X main.version=${VERSION}" -tags musl -o /usr/local/bin/openmeter-jobs ./cmd/jobs

RUN xx-verify /usr/local/bin/openmeter-jobs

FROM alpine:3.22.1@sha256:4bcff63911fcb4448bd4fdacec207030997caf25e9bea4045fa6c8c44de311d1

RUN apk add --update --no-cache ca-certificates tzdata bash

SHELL ["/bin/bash", "-c"]

COPY --link --from=builder /usr/local/bin/openmeter /usr/local/bin/
COPY --link --from=builder /usr/local/bin/openmeter-sink-worker /usr/local/bin/
COPY --link --from=builder /usr/local/bin/openmeter-balance-worker /usr/local/bin/
COPY --link --from=builder /usr/local/bin/openmeter-notification-service /usr/local/bin/
COPY --link --from=builder /usr/local/bin/openmeter-billing-worker /usr/local/bin/
COPY --link --from=builder /usr/local/bin/openmeter-jobs /usr/local/bin/
COPY --link --from=builder /src/go.* /usr/local/src/openmeter/
COPY --link --from=builder /src/entrypoint.sh /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]

CMD openmeter
