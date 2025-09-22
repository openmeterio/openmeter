from os import environ
import asyncio

from openmeter import OpenMeterCloudClient
from corehttp.exceptions import HttpResponseError
from corehttp.credentials import ServiceKeyCredential

ENDPOINT = environ.get("OPENMETER_ENDPOINT") or "http://localhost:8888"
token = environ.get("OPENMETER_TOKEN")

credential = ServiceKeyCredential(token)

client = OpenMeterCloudClient(
    endpoint=ENDPOINT,
    credential=credential,
)


async def main():
    async with client as c:
        try:
            r = await c.query_meter(meter_id_or_slug="tokens_total")
            print("Query total values:\n\n", r)
            r = await c.query_meter(
                meter_id_or_slug="tokens_total",
                group_by=["method"],
            )
            print("\n\n---\n\nQuery total values grouped by method:\n\n", r)
            r = await c.query_meter(
                meter_id_or_slug="tokens_total",
                filter_group_by={"language": "en"},
            )
            print("\n\n---\n\nQuery total values for language=en:\n\n", r)
        except HttpResponseError as e:
            print(e)


asyncio.run(main())
