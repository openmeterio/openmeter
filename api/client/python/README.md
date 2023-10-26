# OpenMeter Python SDK

## Prerequisites

Python version: >= 3.9

## Install

> The Python SDK is in preview mode.

```sh
pip install -e "git+https://github.com/openmeterio/openmeter.git@main#egg=openmeter&subdirectory=api/client/python"
```

## Quickstart

The client can be initialized with `openmeter.Client()`:

```python
from os import environ
from openmeter import Client

ENDPOINT = environ.get("OPENMETER_ENDPOINT") or "http://localhost:8888"

# it's recommended to also set the Accept header at the client level
client = Client(
    endpoint=ENDPOINT,
    headers={"Accept": "application/json"},
)
```

**Async** client can be initialized by importing the `Client` from `openmeter.aio`.

Ingest events:

```python
from cloudevents.http import CloudEvent
from cloudevents.conversion import to_dict

event = CloudEvent(
    attributes={
        "type": "tokens",
        "source": "openmeter-python",
        "subject": "user-id",
    },
    data={
        "prompt_tokens": 5,
        "completion_tokens": 10,
        "total_tokens": 15,
        "model": "gpt-3.5-turbo",
    },
)

resp = client.ingest_events(to_dict(event))
```
