# coding=utf-8
# pylint: disable=wrong-import-position

from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from ._patch import *  # pylint: disable=unused-wildcard-import

from ._operations import PortalOperations  # type: ignore
from ._operations import AppsOperations  # type: ignore
from ._operations import AppStripeOperations  # type: ignore
from ._operations import CustomerAppsOperations  # type: ignore
from ._operations import CustomersOperations  # type: ignore
from ._operations import FeaturesOperations  # type: ignore
from ._operations import PlansOperations  # type: ignore
from ._operations import PlanAddonsOperations  # type: ignore
from ._operations import AddonsOperations  # type: ignore
from ._operations import SubscriptionsOperations  # type: ignore
from ._operations import SubscriptionAddonsOperations  # type: ignore
from ._operations import EntitlementsOperations  # type: ignore
from ._operations import GrantsOperations  # type: ignore
from ._operations import SubjectsOperations  # type: ignore
from ._operations import CustomerOperations  # type: ignore
from ._operations import CustomerEntitlementOperations  # type: ignore
from ._operations import CustomerStripeOperations  # type: ignore
from ._operations import MarketplaceOperations  # type: ignore
from ._operations import AppCustomInvoicingOperations  # type: ignore
from ._operations import EventsOperations  # type: ignore
from ._operations import EventsV2Operations  # type: ignore
from ._operations import MetersOperations  # type: ignore
from ._operations import SubjectsOperations  # type: ignore
from ._operations import DebugOperations  # type: ignore
from ._operations import NotificationChannelsOperations  # type: ignore
from ._operations import NotificationRulesOperations  # type: ignore
from ._operations import NotificationEventsOperations  # type: ignore
from ._operations import EntitlementsV2Operations  # type: ignore
from ._operations import CustomerEntitlementsV2Operations  # type: ignore
from ._operations import CustomerEntitlementV2Operations  # type: ignore
from ._operations import GrantsV2Operations  # type: ignore
from ._operations import BillingProfilesOperations  # type: ignore
from ._operations import CustomerOverridesOperations  # type: ignore
from ._operations import InvoicesOperations  # type: ignore
from ._operations import InvoiceOperations  # type: ignore
from ._operations import CustomerInvoiceOperations  # type: ignore
from ._operations import ProgressOperations  # type: ignore
from ._operations import CurrenciesOperations  # type: ignore

from ._patch import __all__ as _patch_all
from ._patch import *
from ._patch import patch_sdk as _patch_sdk

__all__ = [
    "PortalOperations",
    "AppsOperations",
    "AppStripeOperations",
    "CustomerAppsOperations",
    "CustomersOperations",
    "FeaturesOperations",
    "PlansOperations",
    "PlanAddonsOperations",
    "AddonsOperations",
    "SubscriptionsOperations",
    "SubscriptionAddonsOperations",
    "EntitlementsOperations",
    "GrantsOperations",
    "SubjectsOperations",
    "CustomerOperations",
    "CustomerEntitlementOperations",
    "CustomerStripeOperations",
    "MarketplaceOperations",
    "AppCustomInvoicingOperations",
    "EventsOperations",
    "EventsV2Operations",
    "MetersOperations",
    "SubjectsOperations",
    "DebugOperations",
    "NotificationChannelsOperations",
    "NotificationRulesOperations",
    "NotificationEventsOperations",
    "EntitlementsV2Operations",
    "CustomerEntitlementsV2Operations",
    "CustomerEntitlementV2Operations",
    "GrantsV2Operations",
    "BillingProfilesOperations",
    "CustomerOverridesOperations",
    "InvoicesOperations",
    "InvoiceOperations",
    "CustomerInvoiceOperations",
    "ProgressOperations",
    "CurrenciesOperations",
]
__all__.extend([p for p in _patch_all if p not in __all__])  # pyright: ignore
_patch_sdk()
