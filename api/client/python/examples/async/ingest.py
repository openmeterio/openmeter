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


async def main():
    async with client as c:
        try:
            event = CloudEvent(
                attributes={
                    "type": "request",
                    "source": "openmeter-python",
                    "subject": "user-id",
                },
                data={"method": "GET", "route": "/hello"},
            )
            await c.ingest_events(to_dict(event))
            event = CloudEvent(
                attributes={
                    "type": "request",
                    "source": "openmeter-python",
                    "subject": "user-id",
                },
                data={"method": "POST", "route": "/hello"},
            )
            await c.ingest_events(to_dict(event))
        except HttpResponseError as e:
            print(e)


asyncio.run(main())
