"""List meters with the asynchronous v3 client."""

import asyncio
import os

from openmeter import AsyncClient


async def main() -> None:
    """Run the asynchronous example using environment-based configuration."""

    base_url = os.getenv("OPENMETER_BASE_URL", "http://127.0.0.1:8888/api/v3")
    token = os.getenv("OPENMETER_TOKEN")

    async with AsyncClient(base_url, token=token) as client:
        async for meter in client.meters.list_all():
            print(meter.key, meter.aggregation)


if __name__ == "__main__":
    asyncio.run(main())
