FROM golang:1.20

WORKDIR /usr/src/app

COPY go.* ./
COPY examples/ingest-kubernetes-pod-time-go/go.* ./examples/ingest-kubernetes-pod-time-go/
RUN cd examples/ingest-kubernetes-pod-time-go && go mod download && go mod verify

COPY . .

WORKDIR /usr/src/app/examples/ingest-kubernetes-pod-time-go

RUN go build -v -o /usr/local/bin/app ./...

CMD ["app"]
