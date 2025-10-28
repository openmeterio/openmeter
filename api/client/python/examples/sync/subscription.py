from os import environ
from typing import Optional

from openmeter import Client
from openmeter.models import (
    PlanSubscriptionCreate,
    PlanReferenceInput,
    SubscriptionStatus,
)
from corehttp.exceptions import HttpResponseError

ENDPOINT: str = environ.get("OPENMETER_ENDPOINT") or "https://openmeter.cloud"
token: Optional[str] = environ.get("OPENMETER_TOKEN")
customer_key: str = environ.get("OPENMETER_CUSTOMER_KEY") or "acme-corp-1"
plan_key: str = environ.get("OPENMETER_PLAN_KEY") or "free"

client = Client(
    endpoint=ENDPOINT,
    token=token,
)


def main() -> None:
    try:
        # Create a subscription for the customer using the free plan
        print(f"Creating subscription for customer '{customer_key}' with plan '{plan_key}'...")

        subscription_create = PlanSubscriptionCreate(
            customer_key=customer_key,
            plan=PlanReferenceInput(
                key=plan_key,
            ),
            name="Free Plan Subscription",
            description="Subscription to the free plan for Acme Corporation",
            metadata={
                "source": "example",
                "environment": "development",
            },
        )

        subscription = client.product_catalog.subscriptions.create(subscription_create)
        print(f"Subscription created successfully with ID: {subscription.id}")
        print(f"Subscription name: {subscription.name}")
        print(f"Subscription status: {subscription.status}")
        print(f"Customer ID: {subscription.customer_id}")
        print(f"Active from: {subscription.active_from}")
        print(f"Active to: {subscription.active_to}")
        print(f"Currency: {subscription.currency}")
        print(f"Billing cadence: {subscription.billing_cadence}")

        # Retrieve the subscription to verify
        retrieved_subscription = client.product_catalog.subscriptions.get_expanded(subscription.id)
        print(f"\nRetrieved subscription: {retrieved_subscription.name}")
        print(f"Status: {retrieved_subscription.status}")
        if retrieved_subscription.plan:
            print(f"Plan key: {retrieved_subscription.plan.key}")
            print(f"Plan version: {retrieved_subscription.plan.version}")

        # List subscriptions for the customer
        print(f"\nListing subscriptions for customer '{customer_key}'...")
        subscriptions_response = client.customer.customers.list_customer_subscriptions(
            customer_key, status=[SubscriptionStatus.ACTIVE]
        )
        for sub in subscriptions_response.items_property:
            print(f"\t{sub.name} (ID: {sub.id}, Status: {sub.status})")

    except HttpResponseError as e:
        print(f"Error: {e}")


main()
