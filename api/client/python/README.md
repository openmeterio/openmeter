# OpenMeter Python SDK

[https://pypi.org/project/openmeter](On PyPI)

## Prerequisites

Python version: >= 3.9

## Install

> The Python SDK is in preview mode.

```sh
pip install openmeter
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

## Publish

Update version number in `pyproject.toml`.
Run the following commands:

```sh
poetry config pypi-token.pypi {your_pypi_api_token}
poetry publish --build
```
