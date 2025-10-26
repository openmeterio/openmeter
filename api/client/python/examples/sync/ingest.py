from os import environ
from typing import Optional
import datetime
import uuid

from openmeter import Client
from openmeter.models import Event
from corehttp.exceptions import HttpResponseError

ENDPOINT: str = environ.get("OPENMETER_ENDPOINT") or "https://openmeter.cloud"
token: Optional[str] = environ.get("OPENMETER_TOKEN")


client = Client(
    endpoint=ENDPOINT,
    token=token,
)


def main() -> None:
    try:
        # Create a CloudEvents event
        event = Event(
            id=str(uuid.uuid4()),
            source="my-app",
            specversion="1.0",
            type="prompt",
            subject="customer-1",
            time=datetime.datetime.now(datetime.timezone.utc),
            data={
                "tokens": 100,
                "model": "gpt-4o",
                "type": "input",
            },
        )

        # Ingest the event
        client.events.ingest_event(event)
        print("Event ingested successfully")
    except HttpResponseError as e:
        print(f"Error ingesting event: {e}")


main()
