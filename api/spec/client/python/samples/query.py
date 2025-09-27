from os import environ
from typing import Optional

from openmeter import OpenMeterCloudClient
from openmeter.models import MeterQueryResult
from corehttp.exceptions import HttpResponseError
from corehttp.credentials import ServiceKeyCredential

ENDPOINT: str = environ.get("OPENMETER_ENDPOINT") or "http://localhost:8888"
token: Optional[str] = environ.get("OPENMETER_TOKEN")

credential = ServiceKeyCredential(token)

client = OpenMeterCloudClient(
    endpoint=ENDPOINT,
    credential=credential,
)


def main() -> None:
    try:
        # Query total values
        r: MeterQueryResult = client.meters.query_json(meter_id_or_slug="tokens_total")
        print("Query total values:", r.data[0].value)

        # Query total values grouped by language
        r = client.meters.query_json(
            meter_id_or_slug="tokens_total",
            group_by=["language"],
        )
        print("Query total values grouped by language:")
        for row in r.data:
            print(row.group_by["language"], ":", row.value)

        # Query total values for language=en
        r = client.meters.query_json(
            meter_id_or_slug="tokens_total",
            filter_group_by={"language": ["en"]},
        )
        print("Query total values for language=en:", r.data[0].value)
    except HttpResponseError as e:
        print(e)


main()
