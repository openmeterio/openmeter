from os import environ
from typing import Optional

from openmeter import Client
from openmeter.models import MeterQueryResult, FilterString
from corehttp.exceptions import HttpResponseError

ENDPOINT: str = environ.get("OPENMETER_ENDPOINT") or "https://openmeter.cloud"
token: Optional[str] = environ.get("OPENMETER_TOKEN")

client = Client(
    endpoint=ENDPOINT,
    token=token,
)


def main() -> None:
    try:
        # Query total values
        r: MeterQueryResult = client.meters.query_json(meter_id_or_slug="tokens_total")
        if r.data and len(r.data) > 0:
            print("Query total values:", r.data[0].value)
        else:
            print("Query total values: No data returned")

        # Query total values grouped by language
        r = client.meters.query_json(
            meter_id_or_slug="tokens_total",
            group_by=["model"],
        )
        print("Query total values grouped by model:")
        for row in r.data:
            print("\t", row.group_by["model"], ":", row.value)

        # Query total values for model=gpt-4o
        r = client.meters.query_json(
            meter_id_or_slug="tokens_total",
            advanced_meter_group_by_filters={"model": FilterString(eq="gpt-4o")},
        )
        if r.data and len(r.data) > 0:
            print("Query total values for model=gpt-4o:", r.data[0].value)
        else:
            print("Query total values for model=gpt-4o: No data returned")
    except HttpResponseError as e:
        print(e)


main()
