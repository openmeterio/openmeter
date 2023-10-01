from os import environ
import asyncio

from openmeter.aio import Client
from azure.core.exceptions import HttpResponseError
from cloudevents.http import CloudEvent
from cloudevents.conversion import to_dict

ENDPOINT = environ.get("OPENMETER_ENDPOINT") or "http://localhost:8888"
token = environ.get("OPENMETER_TOKEN")

headers = {"Accept": "application/json"}
if token and token != "":
    headers["Authorization"] = f"Bearer {token}"

client = Client(
    endpoint=ENDPOINT,
    headers=headers,
)

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


async def main():
    async with client as c:
        try:
            await c.ingest_events(to_dict(event))
        except HttpResponseError as e:
            print(e)


asyncio.run(main())
