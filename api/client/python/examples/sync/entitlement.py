from os import environ
from typing import Optional

from openmeter import Client
from corehttp.exceptions import HttpResponseError

ENDPOINT: str = environ.get("OPENMETER_ENDPOINT") or "https://openmeter.cloud"
token: Optional[str] = environ.get("OPENMETER_TOKEN")
customer_key: str = environ.get("OPENMETER_CUSTOMER_KEY") or "acme-corp-1"
feature_key: str = environ.get("OPENMETER_FEATURE_KEY") or "api_access"

client = Client(
    endpoint=ENDPOINT,
    token=token,
)


def main() -> None:
    try:
        # Check customer access to a specific feature
        print(f"Checking access for customer '{customer_key}' to feature '{feature_key}'...")

        entitlement_value = client.customer_entitlement.get_customer_entitlement_value(customer_key, feature_key)

        print(f"\nEntitlement Value:")
        print(f"Has Access: {entitlement_value.has_access}")

        # For metered entitlements, additional properties are available
        if entitlement_value.balance is not None:
            print(f"Balance: {entitlement_value.balance}")
        if entitlement_value.usage is not None:
            print(f"Usage: {entitlement_value.usage}")
        if entitlement_value.overage is not None:
            print(f"Overage: {entitlement_value.overage}")

        # For static entitlements, config is available
        if entitlement_value.config is not None:
            print(f"Config: {entitlement_value.config}")

        # List customer entitlements and demonstrate type-specific handling
        print(f"\nListing all entitlements for customer '{customer_key}'...")
        entitlements_response = client.customer_entitlements_v2.list(customer_key)

        print(f"\nEntitlements by Type:")
        for entitlement in entitlements_response.items_property:
            # Note: Due to a deserialization issue in the SDK, items come back as dicts
            # Access fields using dict syntax or .get()
            print(f"\n  Feature: {entitlement.get('featureKey')}")
            print(f"  ID: {entitlement.get('id')}")

            # Handle different entitlement types using discriminator
            entitlement_type = entitlement.get("type")
            if entitlement_type == "metered":
                # Metered entitlement
                print(f"  Type: Metered")
                print(f"  Soft Limit: {entitlement.get('isSoftLimit')}")
                if entitlement.get("issueAfterReset") is not None:
                    print(f"  Issue After Reset: {entitlement.get('issueAfterReset')}")
            elif entitlement_type == "static":
                # Static entitlement
                print(f"  Type: Static")
                if entitlement.get("config") is not None:
                    print(f"  Config: {entitlement.get('config')}")
            elif entitlement_type == "boolean":
                # Boolean entitlement
                print(f"  Type: Boolean")

        # Get overall customer access to all features
        print(f"\nGetting overall access for customer '{customer_key}'...")
        customer_access = client.customer.get_customer_access(customer_key)

        print(f"\nCustomer Access Summary:")
        print(f"Total entitlements: {len(customer_access.entitlements)}")
        for feature, value in customer_access.entitlements.items():
            access_status = "✓" if value.has_access else "✗"
            print(f"  {access_status} {feature}: has_access={value.has_access}")
            if value.balance is not None:
                print(f"    Balance: {value.balance}")
            if value.usage is not None:
                print(f"    Usage: {value.usage}")

    except HttpResponseError as e:
        print(f"Error: {e}")


main()
