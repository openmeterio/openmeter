from os import environ
from typing import Optional
import asyncio

from openmeter.aio import Client
from openmeter.models import (
    CustomerCreate,
    CustomerReplaceUpdate,
    CustomerUsageAttribution,
    Metadata,
)
from corehttp.exceptions import HttpResponseError

ENDPOINT: str = environ.get("OPENMETER_ENDPOINT") or "https://openmeter.cloud"
token: Optional[str] = environ.get("OPENMETER_TOKEN")
customer_key: str = environ.get("OPENMETER_CUSTOMER_KEY") or "acme-corp-1"
subject_key: str = environ.get("OPENMETER_SUBJECT_KEY") or "acme-user-1"


async def main() -> None:
    async with Client(
        endpoint=ENDPOINT,
        token=token,
    ) as client:
        try:
            # Create a customer
            customer_create = CustomerCreate(
                name="Acme Corporation",
                usage_attribution=CustomerUsageAttribution(subject_keys=[subject_key]),
                description="A demo customer for testing",
                metadata=Metadata(
                    {
                        "industry": "technology",
                    }
                ),
                key=customer_key,
                primary_email="contact@acme-corp.example.com",
                currency="EUR",
            )

            created_customer = await client.customers.create(customer_create)
            print(f"Customer created successfully with ID: {created_customer.id}")
            print(f"Customer name: {created_customer.name}")
            print(f"Customer key: {created_customer.key}")

            # Get the customer by ID or key
            customer = await client.customers.get(created_customer.id)
            print(f"\nRetrieved customer: {customer.name}")
            print(f"Primary email: {customer.primary_email}")
            print(f"Currency: {customer.currency}")

            # Update the customer
            customer_update = CustomerReplaceUpdate(
                name="Acme Corporation Ltd.",
                usage_attribution=CustomerUsageAttribution(subject_keys=[subject_key]),
                description="Updated demo customer",
                metadata=Metadata(
                    {
                        "industry": "technology",
                    }
                ),
                key=customer_key,
                primary_email="info@acme-corp.example.com",
                currency="USD",
            )

            updated_customer = await client.customers.update(created_customer.id, customer_update)
            print(f"\nCustomer updated successfully")
            print(f"Updated name: {updated_customer.name}")
            print(f"Updated email: {updated_customer.primary_email}")
            print(f"Updated currency: {updated_customer.currency}")

        except HttpResponseError as e:
            print(f"Error: {e}")


asyncio.run(main())
