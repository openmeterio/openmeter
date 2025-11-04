# coding=utf-8

from copy import deepcopy
from typing import Any, Awaitable
from typing_extensions import Self

from corehttp.rest import AsyncHttpResponse, HttpRequest
from corehttp.runtime import AsyncPipelineClient, policies

from .._utils.serialization import Deserializer, Serializer
from ._configuration import OpenMeterClientConfiguration
from .operations import (
    AddonsOperations,
    AppCustomInvoicingOperations,
    AppStripeOperations,
    AppsOperations,
    BillingProfilesOperations,
    CurrenciesOperations,
    CustomerAppsOperations,
    CustomerEntitlementOperations,
    CustomerEntitlementV2Operations,
    CustomerEntitlementsV2Operations,
    CustomerInvoiceOperations,
    CustomerOperations,
    CustomerOverridesOperations,
    CustomerStripeOperations,
    CustomersOperations,
    DebugOperations,
    EntitlementsOperations,
    EntitlementsV2Operations,
    EventsOperations,
    EventsV2Operations,
    FeaturesOperations,
    GrantsOperations,
    GrantsV2Operations,
    InvoiceOperations,
    InvoicesOperations,
    MarketplaceOperations,
    MetersOperations,
    NotificationChannelsOperations,
    NotificationEventsOperations,
    NotificationRulesOperations,
    PlanAddonsOperations,
    PlansOperations,
    PortalOperations,
    ProgressOperations,
    SubjectsOperations,
    SubscriptionAddonsOperations,
    SubscriptionsOperations,
)


class OpenMeterClient:  # pylint: disable=client-accepts-api-version-keyword,too-many-instance-attributes
    """OpenMeter is a cloud native usage metering service.
    The OpenMeter API allows you to ingest events, query meter usage, and manage resources.

    :ivar portal: PortalOperations operations
    :vartype portal: openmeter.aio.operations.PortalOperations
    :ivar apps: AppsOperations operations
    :vartype apps: openmeter.aio.operations.AppsOperations
    :ivar app_stripe: AppStripeOperations operations
    :vartype app_stripe: openmeter.aio.operations.AppStripeOperations
    :ivar customer_apps: CustomerAppsOperations operations
    :vartype customer_apps: openmeter.aio.operations.CustomerAppsOperations
    :ivar customers: CustomersOperations operations
    :vartype customers: openmeter.aio.operations.CustomersOperations
    :ivar features: FeaturesOperations operations
    :vartype features: openmeter.aio.operations.FeaturesOperations
    :ivar plans: PlansOperations operations
    :vartype plans: openmeter.aio.operations.PlansOperations
    :ivar plan_addons: PlanAddonsOperations operations
    :vartype plan_addons: openmeter.aio.operations.PlanAddonsOperations
    :ivar addons: AddonsOperations operations
    :vartype addons: openmeter.aio.operations.AddonsOperations
    :ivar subscriptions: SubscriptionsOperations operations
    :vartype subscriptions: openmeter.aio.operations.SubscriptionsOperations
    :ivar subscription_addons: SubscriptionAddonsOperations operations
    :vartype subscription_addons: openmeter.aio.operations.SubscriptionAddonsOperations
    :ivar entitlements: EntitlementsOperations operations
    :vartype entitlements: openmeter.aio.operations.EntitlementsOperations
    :ivar grants: GrantsOperations operations
    :vartype grants: openmeter.aio.operations.GrantsOperations
    :ivar subjects: SubjectsOperations operations
    :vartype subjects: openmeter.aio.operations.SubjectsOperations
    :ivar customer: CustomerOperations operations
    :vartype customer: openmeter.aio.operations.CustomerOperations
    :ivar customer_entitlement: CustomerEntitlementOperations operations
    :vartype customer_entitlement: openmeter.aio.operations.CustomerEntitlementOperations
    :ivar customer_stripe: CustomerStripeOperations operations
    :vartype customer_stripe: openmeter.aio.operations.CustomerStripeOperations
    :ivar marketplace: MarketplaceOperations operations
    :vartype marketplace: openmeter.aio.operations.MarketplaceOperations
    :ivar app_custom_invoicing: AppCustomInvoicingOperations operations
    :vartype app_custom_invoicing: openmeter.aio.operations.AppCustomInvoicingOperations
    :ivar events: EventsOperations operations
    :vartype events: openmeter.aio.operations.EventsOperations
    :ivar events_v2: EventsV2Operations operations
    :vartype events_v2: openmeter.aio.operations.EventsV2Operations
    :ivar meters: MetersOperations operations
    :vartype meters: openmeter.aio.operations.MetersOperations
    :ivar subjects: SubjectsOperations operations
    :vartype subjects: openmeter.aio.operations.SubjectsOperations
    :ivar debug: DebugOperations operations
    :vartype debug: openmeter.aio.operations.DebugOperations
    :ivar notification_channels: NotificationChannelsOperations operations
    :vartype notification_channels: openmeter.aio.operations.NotificationChannelsOperations
    :ivar notification_rules: NotificationRulesOperations operations
    :vartype notification_rules: openmeter.aio.operations.NotificationRulesOperations
    :ivar notification_events: NotificationEventsOperations operations
    :vartype notification_events: openmeter.aio.operations.NotificationEventsOperations
    :ivar entitlements_v2: EntitlementsV2Operations operations
    :vartype entitlements_v2: openmeter.aio.operations.EntitlementsV2Operations
    :ivar customer_entitlements_v2: CustomerEntitlementsV2Operations operations
    :vartype customer_entitlements_v2: openmeter.aio.operations.CustomerEntitlementsV2Operations
    :ivar customer_entitlement_v2: CustomerEntitlementV2Operations operations
    :vartype customer_entitlement_v2: openmeter.aio.operations.CustomerEntitlementV2Operations
    :ivar grants_v2: GrantsV2Operations operations
    :vartype grants_v2: openmeter.aio.operations.GrantsV2Operations
    :ivar billing_profiles: BillingProfilesOperations operations
    :vartype billing_profiles: openmeter.aio.operations.BillingProfilesOperations
    :ivar customer_overrides: CustomerOverridesOperations operations
    :vartype customer_overrides: openmeter.aio.operations.CustomerOverridesOperations
    :ivar invoices: InvoicesOperations operations
    :vartype invoices: openmeter.aio.operations.InvoicesOperations
    :ivar invoice: InvoiceOperations operations
    :vartype invoice: openmeter.aio.operations.InvoiceOperations
    :ivar customer_invoice: CustomerInvoiceOperations operations
    :vartype customer_invoice: openmeter.aio.operations.CustomerInvoiceOperations
    :ivar progress: ProgressOperations operations
    :vartype progress: openmeter.aio.operations.ProgressOperations
    :ivar currencies: CurrenciesOperations operations
    :vartype currencies: openmeter.aio.operations.CurrenciesOperations
    :keyword endpoint: Service host. Default value is "https://127.0.0.1".
    :paramtype endpoint: str
    """

    def __init__(  # pylint: disable=missing-client-constructor-parameter-credential
        self, *, endpoint: str = "https://127.0.0.1", **kwargs: Any
    ) -> None:
        _endpoint = "{endpoint}"
        self._config = OpenMeterClientConfiguration(endpoint=endpoint, **kwargs)

        _policies = kwargs.pop("policies", None)
        if _policies is None:
            _policies = [
                self._config.headers_policy,
                self._config.user_agent_policy,
                self._config.proxy_policy,
                policies.ContentDecodePolicy(**kwargs),
                self._config.retry_policy,
                self._config.authentication_policy,
                self._config.logging_policy,
            ]
        self._client: AsyncPipelineClient = AsyncPipelineClient(endpoint=_endpoint, policies=_policies, **kwargs)

        self._serialize = Serializer()
        self._deserialize = Deserializer()
        self._serialize.client_side_validation = False
        self.portal = PortalOperations(self._client, self._config, self._serialize, self._deserialize)
        self.apps = AppsOperations(self._client, self._config, self._serialize, self._deserialize)
        self.app_stripe = AppStripeOperations(self._client, self._config, self._serialize, self._deserialize)
        self.customer_apps = CustomerAppsOperations(self._client, self._config, self._serialize, self._deserialize)
        self.customers = CustomersOperations(self._client, self._config, self._serialize, self._deserialize)
        self.features = FeaturesOperations(self._client, self._config, self._serialize, self._deserialize)
        self.plans = PlansOperations(self._client, self._config, self._serialize, self._deserialize)
        self.plan_addons = PlanAddonsOperations(self._client, self._config, self._serialize, self._deserialize)
        self.addons = AddonsOperations(self._client, self._config, self._serialize, self._deserialize)
        self.subscriptions = SubscriptionsOperations(self._client, self._config, self._serialize, self._deserialize)
        self.subscription_addons = SubscriptionAddonsOperations(
            self._client, self._config, self._serialize, self._deserialize
        )
        self.entitlements = EntitlementsOperations(self._client, self._config, self._serialize, self._deserialize)
        self.grants = GrantsOperations(self._client, self._config, self._serialize, self._deserialize)
        self.subjects = SubjectsOperations(self._client, self._config, self._serialize, self._deserialize)
        self.customer = CustomerOperations(self._client, self._config, self._serialize, self._deserialize)
        self.customer_entitlement = CustomerEntitlementOperations(
            self._client, self._config, self._serialize, self._deserialize
        )
        self.customer_stripe = CustomerStripeOperations(self._client, self._config, self._serialize, self._deserialize)
        self.marketplace = MarketplaceOperations(self._client, self._config, self._serialize, self._deserialize)
        self.app_custom_invoicing = AppCustomInvoicingOperations(
            self._client, self._config, self._serialize, self._deserialize
        )
        self.events = EventsOperations(self._client, self._config, self._serialize, self._deserialize)
        self.events_v2 = EventsV2Operations(self._client, self._config, self._serialize, self._deserialize)
        self.meters = MetersOperations(self._client, self._config, self._serialize, self._deserialize)
        self.subjects = SubjectsOperations(self._client, self._config, self._serialize, self._deserialize)
        self.debug = DebugOperations(self._client, self._config, self._serialize, self._deserialize)
        self.notification_channels = NotificationChannelsOperations(
            self._client, self._config, self._serialize, self._deserialize
        )
        self.notification_rules = NotificationRulesOperations(
            self._client, self._config, self._serialize, self._deserialize
        )
        self.notification_events = NotificationEventsOperations(
            self._client, self._config, self._serialize, self._deserialize
        )
        self.entitlements_v2 = EntitlementsV2Operations(self._client, self._config, self._serialize, self._deserialize)
        self.customer_entitlements_v2 = CustomerEntitlementsV2Operations(
            self._client, self._config, self._serialize, self._deserialize
        )
        self.customer_entitlement_v2 = CustomerEntitlementV2Operations(
            self._client, self._config, self._serialize, self._deserialize
        )
        self.grants_v2 = GrantsV2Operations(self._client, self._config, self._serialize, self._deserialize)
        self.billing_profiles = BillingProfilesOperations(
            self._client, self._config, self._serialize, self._deserialize
        )
        self.customer_overrides = CustomerOverridesOperations(
            self._client, self._config, self._serialize, self._deserialize
        )
        self.invoices = InvoicesOperations(self._client, self._config, self._serialize, self._deserialize)
        self.invoice = InvoiceOperations(self._client, self._config, self._serialize, self._deserialize)
        self.customer_invoice = CustomerInvoiceOperations(
            self._client, self._config, self._serialize, self._deserialize
        )
        self.progress = ProgressOperations(self._client, self._config, self._serialize, self._deserialize)
        self.currencies = CurrenciesOperations(self._client, self._config, self._serialize, self._deserialize)

    def send_request(
        self, request: HttpRequest, *, stream: bool = False, **kwargs: Any
    ) -> Awaitable[AsyncHttpResponse]:
        """Runs the network request through the client's chained policies.

        >>> from corehttp.rest import HttpRequest
        >>> request = HttpRequest("GET", "https://www.example.org/")
        <HttpRequest [GET], url: 'https://www.example.org/'>
        >>> response = await client.send_request(request)
        <AsyncHttpResponse: 200 OK>

        For more information on this code flow, see https://aka.ms/azsdk/dpcodegen/python/send_request

        :param request: The network request you want to make. Required.
        :type request: ~corehttp.rest.HttpRequest
        :keyword bool stream: Whether the response payload will be streamed. Defaults to False.
        :return: The response of your network call. Does not do error handling on your response.
        :rtype: ~corehttp.rest.AsyncHttpResponse
        """

        request_copy = deepcopy(request)
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }

        request_copy.url = self._client.format_url(request_copy.url, **path_format_arguments)
        return self._client.send_request(request_copy, stream=stream, **kwargs)  # type: ignore

    async def close(self) -> None:
        await self._client.close()

    async def __aenter__(self) -> Self:
        await self._client.__aenter__()
        return self

    async def __aexit__(self, *exc_details: Any) -> None:
        await self._client.__aexit__(*exc_details)
