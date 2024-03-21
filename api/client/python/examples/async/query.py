from os import environ
import asyncio

from openmeter.aio import Client
from azure.core.exceptions import HttpResponseError

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
            r = await c.query_meter(meter_id_or_slug="api_requests_total")
            print("Query total values:\n\n", r)
            r = await c.query_meter(
                meter_id_or_slug="api_requests_total",
                group_by=["method"],
            )
            print("\n\n---\n\nQuery total values grouped by method:\n\n", r)
            r = await c.query_meter(
                meter_id_or_slug="api_requests_total",
                filter_group_by={"method": "GET"},
            )
            print("\n\n---\n\nQuery total values for GET method:\n\n", r)
        except HttpResponseError as e:
            print(e)


asyncio.run(main())
