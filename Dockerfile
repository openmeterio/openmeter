FROM --platform=$BUILDPLATFORM tonistiigi/xx:1.3.0@sha256:904fe94f236d36d65aeb5a2462f88f2c537b8360475f6342e7599194f291fb7e AS xx

FROM --platform=$BUILDPLATFORM golang:1.21.4-alpine3.18@sha256:110b07af87238fbdc5f1df52b00927cf58ce3de358eeeb1854f10a8b5e5e1411 AS builder

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

FROM gcr.io/distroless/base-debian11:latest@sha256:b31a6e02605827e77b7ebb82a0ac9669ec51091edd62c2c076175e05556f4ab9 AS distroless

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=builder /usr/local/bin/openmeter /usr/local/bin/
COPY --from=builder /usr/local/bin/openmeter-sink-worker /usr/local/bin/
COPY --from=builder /usr/local/src/openmeter/go.* /usr/local/src/openmeter/

CMD openmeter

FROM redhat/ubi8-micro:8.9-4@sha256:ed850fafd97a7144268f29d58c39c253a2d7b583605b2d66814400505b6b0063 AS ubi8

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=builder /usr/local/bin/openmeter /usr/local/bin/
COPY --from=builder /usr/local/bin/openmeter-sink-worker /usr/local/bin/
COPY --from=builder /usr/local/src/openmeter/go.* /usr/local/src/openmeter/

CMD openmeter

FROM alpine:3.18.4@sha256:eece025e432126ce23f223450a0326fbebde39cdf496a85d8c016293fc851978 AS alpine

RUN apk add --update --no-cache ca-certificates tzdata bash

SHELL ["/bin/bash", "-c"]

COPY --from=builder /usr/local/bin/openmeter /usr/local/bin/
COPY --from=builder /usr/local/bin/openmeter-sink-worker /usr/local/bin/
COPY --from=builder /usr/local/src/openmeter/go.* /usr/local/src/openmeter/

CMD openmeter
