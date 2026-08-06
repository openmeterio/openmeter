"""List and query meters with the synchronous v3 client."""

import os

from openmeter import Client, MeterQueryRequest


def main() -> None:
    """Run the synchronous example using environment-based configuration."""

    base_url = os.getenv("OPENMETER_BASE_URL", "http://127.0.0.1:8888/api/v3")
    token = os.getenv("OPENMETER_TOKEN")

    with Client(base_url, token=token) as client:
        for meter in client.meters.list_all():
            print(meter.key, meter.aggregation)

        meter_id = os.getenv("OPENMETER_METER_ID")
        if meter_id:
            result = client.meters.query(meter_id, MeterQueryRequest())
            print(result.data)


if __name__ == "__main__":
    main()
