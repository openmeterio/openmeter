#!/bin/bash

curl -X POST http://localhost:8888/api/v1alpha2/events \
    -H "Expect:" \
    -H 'Content-Type: application/cloudevents+json' \
    --data-raw '
{
  "specversion" : "1.0",
  "type": "api-calls",
  "id": "00001",
  "time": "2023-01-01T00:00:00.001Z",
  "source": "service-0",
  "subject": "customer-1",
  "data": {
    "duration_ms": "1",
    "method": "GET",
    "path": "/hello"
  }
}
'

curl -X POST http://localhost:8888/api/v1alpha2/events \
    -H "Expect:" \
    -H 'Content-Type: application/cloudevents+json' \
    --data-raw '
{
  "specversion" : "1.0",
  "type": "api-calls",
  "id": "00002",
  "time": "2023-01-01T00:00:00.001Z",
  "source": "service-0",
  "subject": "customer-1",
  "data": {
    "duration_ms": "1",
    "method": "GET",
    "path": "/hello"
  }
}
'

curl -X POST http://localhost:8888/api/v1alpha2/events \
    -H "Expect:" \
    -H 'Content-Type: application/cloudevents+json' \
    --data-raw '
{
  "specversion" : "1.0",
  "type": "api-calls",
  "id": "00003",
  "time": "2023-01-02T00:00:00.001Z",
  "source": "service-0",
  "subject": "customer-1",
  "data": {
    "duration_ms": "1",
    "method": "GET",
    "path": "/hello"
  }
}
'
