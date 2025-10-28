# pylint: disable=too-many-lines
# coding=utf-8
from collections.abc import MutableMapping
import datetime
from io import IOBase
import json
from typing import Any, Callable, IO, Optional, TYPE_CHECKING, TypeVar, Union, overload

from corehttp.exceptions import (
    ClientAuthenticationError,
    HttpResponseError,
    ResourceExistsError,
    ResourceNotFoundError,
    ResourceNotModifiedError,
    StreamClosedError,
    StreamConsumedError,
    map_error,
)
from corehttp.paging import AsyncItemPaged, AsyncList
from corehttp.rest import AsyncHttpResponse, HttpRequest
from corehttp.runtime import AsyncPipelineClient
from corehttp.runtime.pipeline import PipelineResponse
from corehttp.utils import case_insensitive_dict

from ... import models as _models
from ..._utils.model_base import SdkJSONEncoder, _deserialize, _failsafe_deserialize
from ..._utils.serialization import Deserializer, Serializer
from ...operations._operations import (
    build_app_app_custom_invoicing_draft_syncronized_request,
    build_app_app_custom_invoicing_finalized_request,
    build_app_app_custom_invoicing_payment_status_request,
    build_app_app_stripe_create_checkout_session_request,
    build_app_app_stripe_update_stripe_api_key_request,
    build_app_app_stripe_webhook_request,
    build_app_apps_get_request,
    build_app_apps_list_request,
    build_app_apps_uninstall_request,
    build_app_apps_update_request,
    build_app_marketplace_authorize_o_auth2_install_request,
    build_app_marketplace_get_o_auth2_install_url_request,
    build_app_marketplace_get_request,
    build_app_marketplace_install_request,
    build_app_marketplace_install_with_api_key_request,
    build_app_marketplace_list_request,
    build_billing_customer_invoice_endpoints_create_pending_invoice_line_request,
    build_billing_customer_invoice_endpoints_simulate_invoice_request,
    build_billing_customer_overrides_delete_request,
    build_billing_customer_overrides_get_request,
    build_billing_customer_overrides_list_request,
    build_billing_customer_overrides_upsert_request,
    build_billing_invoice_endpoints_advance_action_request,
    build_billing_invoice_endpoints_approve_action_request,
    build_billing_invoice_endpoints_delete_invoice_request,
    build_billing_invoice_endpoints_get_invoice_request,
    build_billing_invoice_endpoints_recalculate_tax_action_request,
    build_billing_invoice_endpoints_retry_action_request,
    build_billing_invoice_endpoints_snapshot_quantities_action_request,
    build_billing_invoice_endpoints_update_invoice_request,
    build_billing_invoice_endpoints_void_invoice_action_request,
    build_billing_invoices_endpoints_invoice_pending_lines_action_request,
    build_billing_invoices_endpoints_list_request,
    build_billing_profiles_create_request,
    build_billing_profiles_delete_request,
    build_billing_profiles_get_request,
    build_billing_profiles_list_request,
    build_billing_profiles_update_request,
    build_customer_customers_apps_delete_app_data_request,
    build_customer_customers_apps_list_app_data_request,
    build_customer_customers_apps_upsert_app_data_request,
    build_customer_customers_create_request,
    build_customer_customers_delete_request,
    build_customer_customers_get_request,
    build_customer_customers_list_customer_subscriptions_request,
    build_customer_customers_list_request,
    build_customer_customers_stripe_create_portal_session_request,
    build_customer_customers_stripe_get_request,
    build_customer_customers_stripe_upsert_request,
    build_customer_customers_update_request,
    build_debug_metrics_request,
    build_entitlements_customer_entitlement_get_customer_entitlement_value_request,
    build_entitlements_customer_get_customer_access_request,
    build_entitlements_entitlements_get_request,
    build_entitlements_entitlements_list_request,
    build_entitlements_grants_delete_request,
    build_entitlements_grants_list_request,
    build_entitlements_subjects_create_grant_request,
    build_entitlements_subjects_delete_request,
    build_entitlements_subjects_get_entitlement_history_request,
    build_entitlements_subjects_get_entitlement_value_request,
    build_entitlements_subjects_get_grants_request,
    build_entitlements_subjects_get_request,
    build_entitlements_subjects_list_request,
    build_entitlements_subjects_override_request,
    build_entitlements_subjects_post_request,
    build_entitlements_subjects_reset_request,
    build_entitlements_v2_customer_entitlement_create_customer_entitlement_grant_request,
    build_entitlements_v2_customer_entitlement_get_customer_entitlement_history_request,
    build_entitlements_v2_customer_entitlement_get_customer_entitlement_value_request,
    build_entitlements_v2_customer_entitlement_get_grants_request,
    build_entitlements_v2_customer_entitlement_reset_customer_entitlement_request,
    build_entitlements_v2_customer_entitlements_delete_request,
    build_entitlements_v2_customer_entitlements_get_request,
    build_entitlements_v2_customer_entitlements_list_request,
    build_entitlements_v2_customer_entitlements_override_request,
    build_entitlements_v2_customer_entitlements_post_request,
    build_entitlements_v2_entitlements_get_request,
    build_entitlements_v2_entitlements_list_request,
    build_entitlements_v2_grants_list_request,
    build_events_ingest_event_request,
    build_events_ingest_events_json_request,
    build_events_ingest_events_request,
    build_events_list_request,
    build_events_v2_list_request,
    build_info_currencies_list_currencies_request,
    build_info_progresses_get_progress_request,
    build_meters_create_request,
    build_meters_delete_request,
    build_meters_get_request,
    build_meters_list_group_by_values_request,
    build_meters_list_request,
    build_meters_list_subjects_request,
    build_meters_query_csv_request,
    build_meters_query_json_request,
    build_meters_query_request,
    build_meters_update_request,
    build_notification_channels_create_request,
    build_notification_channels_delete_request,
    build_notification_channels_get_request,
    build_notification_channels_list_request,
    build_notification_channels_update_request,
    build_notification_events_get_request,
    build_notification_events_list_request,
    build_notification_rules_create_request,
    build_notification_rules_delete_request,
    build_notification_rules_get_request,
    build_notification_rules_list_request,
    build_notification_rules_test_request,
    build_notification_rules_update_request,
    build_portal_meters_query_csv_request,
    build_portal_meters_query_json_request,
    build_portal_tokens_create_request,
    build_portal_tokens_invalidate_request,
    build_portal_tokens_list_request,
    build_product_catalog_addons_archive_request,
    build_product_catalog_addons_create_request,
    build_product_catalog_addons_delete_request,
    build_product_catalog_addons_get_request,
    build_product_catalog_addons_list_request,
    build_product_catalog_addons_publish_request,
    build_product_catalog_addons_update_request,
    build_product_catalog_features_create_request,
    build_product_catalog_features_delete_request,
    build_product_catalog_features_get_request,
    build_product_catalog_features_list_request,
    build_product_catalog_plan_addons_create_request,
    build_product_catalog_plan_addons_delete_request,
    build_product_catalog_plan_addons_get_request,
    build_product_catalog_plan_addons_list_request,
    build_product_catalog_plan_addons_update_request,
    build_product_catalog_plans_archive_request,
    build_product_catalog_plans_create_request,
    build_product_catalog_plans_delete_request,
    build_product_catalog_plans_get_request,
    build_product_catalog_plans_list_request,
    build_product_catalog_plans_next_request,
    build_product_catalog_plans_publish_request,
    build_product_catalog_plans_update_request,
    build_product_catalog_subscription_addons_create_request,
    build_product_catalog_subscription_addons_get_request,
    build_product_catalog_subscription_addons_list_request,
    build_product_catalog_subscription_addons_update_request,
    build_product_catalog_subscriptions_cancel_request,
    build_product_catalog_subscriptions_change_request,
    build_product_catalog_subscriptions_create_request,
    build_product_catalog_subscriptions_delete_request,
    build_product_catalog_subscriptions_edit_request,
    build_product_catalog_subscriptions_get_expanded_request,
    build_product_catalog_subscriptions_migrate_request,
    build_product_catalog_subscriptions_restore_request,
    build_product_catalog_subscriptions_unschedule_cancelation_request,
    build_subjects_delete_request,
    build_subjects_get_request,
    build_subjects_list_request,
    build_subjects_upsert_request,
)
from .._configuration import OpenMeterClientConfiguration

if TYPE_CHECKING:
    from ... import _types
JSON = MutableMapping[str, Any]
T = TypeVar("T")
ClsType = Optional[Callable[[PipelineResponse[HttpRequest, AsyncHttpResponse], T, dict[str, Any]], Any]]
_Unset: Any = object()
List = list


class AppOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`app` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

        self.apps = AppAppsOperations(self._client, self._config, self._serialize, self._deserialize)
        self.app_stripe = AppAppStripeOperations(self._client, self._config, self._serialize, self._deserialize)
        self.marketplace = AppMarketplaceOperations(self._client, self._config, self._serialize, self._deserialize)
        self.app_custom_invoicing = AppAppCustomInvoicingOperations(
            self._client, self._config, self._serialize, self._deserialize
        )


class CustomerOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`customer` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

        self.customers_apps = CustomerCustomersAppsOperations(
            self._client, self._config, self._serialize, self._deserialize
        )
        self.customers = CustomerCustomersOperations(self._client, self._config, self._serialize, self._deserialize)
        self.customers_stripe = CustomerCustomersStripeOperations(
            self._client, self._config, self._serialize, self._deserialize
        )


class ProductCatalogOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`product_catalog` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

        self.features = ProductCatalogFeaturesOperations(self._client, self._config, self._serialize, self._deserialize)
        self.plans = ProductCatalogPlansOperations(self._client, self._config, self._serialize, self._deserialize)
        self.plan_addons = ProductCatalogPlanAddonsOperations(
            self._client, self._config, self._serialize, self._deserialize
        )
        self.addons = ProductCatalogAddonsOperations(self._client, self._config, self._serialize, self._deserialize)
        self.subscriptions = ProductCatalogSubscriptionsOperations(
            self._client, self._config, self._serialize, self._deserialize
        )
        self.subscription_addons = ProductCatalogSubscriptionAddonsOperations(
            self._client, self._config, self._serialize, self._deserialize
        )


class EntitlementsOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`entitlements` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

        self.v2 = EntitlementsV2Operations(self._client, self._config, self._serialize, self._deserialize)
        self.entitlements = EntitlementsEntitlementsOperations(
            self._client, self._config, self._serialize, self._deserialize
        )
        self.grants = EntitlementsGrantsOperations(self._client, self._config, self._serialize, self._deserialize)
        self.subjects = EntitlementsSubjectsOperations(self._client, self._config, self._serialize, self._deserialize)
        self.customer = EntitlementsCustomerOperations(self._client, self._config, self._serialize, self._deserialize)
        self.customer_entitlement = EntitlementsCustomerEntitlementOperations(
            self._client, self._config, self._serialize, self._deserialize
        )


class BillingOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`billing` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

        self.profiles = BillingProfilesOperations(self._client, self._config, self._serialize, self._deserialize)
        self.customer_overrides = BillingCustomerOverridesOperations(
            self._client, self._config, self._serialize, self._deserialize
        )
        self.invoices_endpoints = BillingInvoicesEndpointsOperations(
            self._client, self._config, self._serialize, self._deserialize
        )
        self.invoice_endpoints = BillingInvoiceEndpointsOperations(
            self._client, self._config, self._serialize, self._deserialize
        )
        self.customer_invoice_endpoints = BillingCustomerInvoiceEndpointsOperations(
            self._client, self._config, self._serialize, self._deserialize
        )


class PortalOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`portal` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

        self.tokens = PortalTokensOperations(self._client, self._config, self._serialize, self._deserialize)
        self.meters = PortalMetersOperations(self._client, self._config, self._serialize, self._deserialize)


class NotificationOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`notification` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

        self.channels = NotificationChannelsOperations(self._client, self._config, self._serialize, self._deserialize)
        self.rules = NotificationRulesOperations(self._client, self._config, self._serialize, self._deserialize)
        self.events = NotificationEventsOperations(self._client, self._config, self._serialize, self._deserialize)


class InfoOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`info` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

        self.progresses = InfoProgressesOperations(self._client, self._config, self._serialize, self._deserialize)
        self.currencies = InfoCurrenciesOperations(self._client, self._config, self._serialize, self._deserialize)


class EventsOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`events` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

    async def list(
        self,
        *,
        client_id: Optional[str] = None,
        ingested_at_from: Optional[datetime.datetime] = None,
        ingested_at_to: Optional[datetime.datetime] = None,
        id: Optional[str] = None,
        subject: Optional[str] = None,
        customer_id: Optional[List[str]] = None,
        from_parameter: Optional[datetime.datetime] = None,
        to: Optional[datetime.datetime] = None,
        limit: Optional[int] = None,
        **kwargs: Any
    ) -> List[_models.IngestedEvent]:
        """List ingested events.

        List ingested events within a time range.

        If the from query param is not provided it defaults to last 72 hours.

        :keyword client_id: Client ID
         Useful to track progress of a query. Default value is None.
        :paramtype client_id: str
        :keyword ingested_at_from: Start date-time in RFC 3339 format.

         Inclusive. Default value is None.
        :paramtype ingested_at_from: ~datetime.datetime
        :keyword ingested_at_to: End date-time in RFC 3339 format.

         Inclusive. Default value is None.
        :paramtype ingested_at_to: ~datetime.datetime
        :keyword id: The event ID.

         Accepts partial ID. Default value is None.
        :paramtype id: str
        :keyword subject: The event subject.

         Accepts partial subject. Default value is None.
        :paramtype subject: str
        :keyword customer_id: The event customer ID. Default value is None.
        :paramtype customer_id: list[str]
        :keyword from_parameter: Start date-time in RFC 3339 format.

         Inclusive. Default value is None.
        :paramtype from_parameter: ~datetime.datetime
        :keyword to: End date-time in RFC 3339 format.

         Inclusive. Default value is None.
        :paramtype to: ~datetime.datetime
        :keyword limit: Number of events to return. Default value is None.
        :paramtype limit: int
        :return: list of IngestedEvent
        :rtype: list[~openmeter._generated.models.IngestedEvent]
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[List[_models.IngestedEvent]] = kwargs.pop("cls", None)

        _request = build_events_list_request(
            client_id=client_id,
            ingested_at_from=ingested_at_from,
            ingested_at_to=ingested_at_to,
            id=id,
            subject=subject,
            customer_id=customer_id,
            from_parameter=from_parameter,
            to=to,
            limit=limit,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(List[_models.IngestedEvent], response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    @overload
    async def ingest_event(
        self, body: _models.Event, *, content_type: str = "application/cloudevents+json", **kwargs: Any
    ) -> None:
        """Ingest.

        Ingests an event or batch of events following the CloudEvents specification.

        :param body: Required.
        :type body: ~openmeter._generated.models.Event
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/cloudevents+json".
        :paramtype content_type: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def ingest_event(
        self, body: JSON, *, content_type: str = "application/cloudevents+json", **kwargs: Any
    ) -> None:
        """Ingest.

        Ingests an event or batch of events following the CloudEvents specification.

        :param body: Required.
        :type body: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/cloudevents+json".
        :paramtype content_type: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def ingest_event(
        self, body: IO[bytes], *, content_type: str = "application/cloudevents+json", **kwargs: Any
    ) -> None:
        """Ingest.

        Ingests an event or batch of events following the CloudEvents specification.

        :param body: Required.
        :type body: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/cloudevents+json".
        :paramtype content_type: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def ingest_event(self, body: Union[_models.Event, JSON, IO[bytes]], **kwargs: Any) -> None:
        """Ingest.

        Ingests an event or batch of events following the CloudEvents specification.

        :param body: Is one of the following types: Event, JSON, IO[bytes] Required.
        :type body: ~openmeter._generated.models.Event or JSON or IO[bytes]
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("content-type", None))
        cls: ClsType[None] = kwargs.pop("cls", None)

        content_type = content_type or "application/cloudevents+json"
        _content = None
        if isinstance(body, (IOBase, bytes)):
            _content = body
        else:
            _content = json.dumps(body, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_events_ingest_event_request(
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = False
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [204]:
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if cls:
            return cls(pipeline_response, None, {})  # type: ignore

    @overload
    async def ingest_events(
        self, body: List[_models.Event], *, content_type: str = "application/cloudevents-batch+json", **kwargs: Any
    ) -> None:
        """events.

        ingest_events.

        :param body: Required.
        :type body: list[~openmeter._generated.models.Event]
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/cloudevents-batch+json".
        :paramtype content_type: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def ingest_events(
        self, body: List[JSON], *, content_type: str = "application/cloudevents-batch+json", **kwargs: Any
    ) -> None:
        """events.

        ingest_events.

        :param body: Required.
        :type body: list[JSON]
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/cloudevents-batch+json".
        :paramtype content_type: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def ingest_events(
        self, body: IO[bytes], *, content_type: str = "application/cloudevents-batch+json", **kwargs: Any
    ) -> None:
        """events.

        ingest_events.

        :param body: Required.
        :type body: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/cloudevents-batch+json".
        :paramtype content_type: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def ingest_events(self, body: Union[List[_models.Event], List[JSON], IO[bytes]], **kwargs: Any) -> None:
        """events.

        ingest_events.

        :param body: Is one of the following types: [Event], [JSON], IO[bytes] Required.
        :type body: list[~openmeter._generated.models.Event] or list[JSON] or IO[bytes]
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("content-type", None))
        cls: ClsType[None] = kwargs.pop("cls", None)

        content_type = content_type or "application/cloudevents-batch+json"
        _content = None
        if isinstance(body, (IOBase, bytes)):
            _content = body
        else:
            _content = json.dumps(body, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_events_ingest_events_request(
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = False
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [204]:
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if cls:
            return cls(pipeline_response, None, {})  # type: ignore

    @overload
    async def ingest_events_json(
        self, body: _models.Event, *, content_type: str = "application/json", **kwargs: Any
    ) -> None:
        """ingest_events_json.

        :param body: Required.
        :type body: ~openmeter._generated.models.Event
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def ingest_events_json(
        self, body: List[_models.Event], *, content_type: str = "application/json", **kwargs: Any
    ) -> None:
        """ingest_events_json.

        :param body: Required.
        :type body: list[~openmeter._generated.models.Event]
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def ingest_events_json(self, body: "_types.IngestEventsBody", **kwargs: Any) -> None:
        """ingest_events_json.

        :param body: Is either a Event type or a [Event] type. Required.
        :type body: ~openmeter._generated.models.Event or list[~openmeter._generated.models.Event]
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("content-type", None))
        cls: ClsType[None] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(body, _models.Event):
            _content = json.dumps(body, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore
        elif isinstance(body, list):
            _content = json.dumps(body, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_events_ingest_events_json_request(
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = False
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [204]:
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if cls:
            return cls(pipeline_response, None, {})  # type: ignore


class EventsV2Operations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`events_v2` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

    def list(
        self,
        *,
        cursor: Optional[str] = None,
        limit: Optional[int] = None,
        client_id: Optional[str] = None,
        filter: Optional[_models.ListRequestFilter] = None,
        **kwargs: Any
    ) -> AsyncItemPaged["_models.IngestedEvent"]:
        """List ingested events.

        List ingested events with advanced filtering and cursor pagination.

        :keyword cursor: The cursor after which to start the pagination. Default value is None.
        :paramtype cursor: str
        :keyword limit: The limit of the pagination. Default value is None.
        :paramtype limit: int
        :keyword client_id: Client ID
         Useful to track progress of a query. Default value is None.
        :paramtype client_id: str
        :keyword filter: The filter for the events encoded as JSON string. Default value is None.
        :paramtype filter: ~openmeter._generated.models.ListRequestFilter
        :return: An iterator like instance of IngestedEvent
        :rtype: ~corehttp.paging.AsyncItemPaged[~openmeter._generated.models.IngestedEvent]
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[List[_models.IngestedEvent]] = kwargs.pop("cls", None)

        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        def prepare_request(next_link=None):
            if not next_link:

                _request = build_events_v2_list_request(
                    cursor=cursor,
                    limit=limit,
                    client_id=client_id,
                    filter=filter,
                    headers=_headers,
                    params=_params,
                )
                path_format_arguments = {
                    "endpoint": self._serialize.url(
                        "self._config.endpoint", self._config.endpoint, "str", skip_quote=True
                    ),
                }
                _request.url = self._client.format_url(_request.url, **path_format_arguments)

            else:
                _request = HttpRequest("GET", next_link)
                path_format_arguments = {
                    "endpoint": self._serialize.url(
                        "self._config.endpoint", self._config.endpoint, "str", skip_quote=True
                    ),
                }
                _request.url = self._client.format_url(_request.url, **path_format_arguments)

            return _request

        async def extract_data(pipeline_response):
            deserialized = pipeline_response.http_response.json()
            list_of_elem = _deserialize(List[_models.IngestedEvent], deserialized.get("items", []))
            if cls:
                list_of_elem = cls(list_of_elem)  # type: ignore
            return None, AsyncList(list_of_elem)

        async def get_next(next_link=None):
            _request = prepare_request(next_link)

            _stream = False
            pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)
            response = pipeline_response.http_response

            if response.status_code not in [200]:
                map_error(status_code=response.status_code, response=response, error_map=error_map)
                error = None
                if response.status_code == 400:
                    error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
                elif response.status_code == 401:
                    error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                    raise ClientAuthenticationError(response=response, model=error)
                if response.status_code == 403:
                    error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
                elif response.status_code == 500:
                    error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
                elif response.status_code == 503:
                    error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
                elif response.status_code == 412:
                    error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
                else:
                    error = _failsafe_deserialize(
                        _models.UnexpectedProblemResponse,
                        response,
                    )
                raise HttpResponseError(response=response, model=error)

            return pipeline_response

        return AsyncItemPaged(get_next, extract_data)


class MetersOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`meters` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

    async def list(
        self,
        *,
        page: Optional[int] = None,
        page_size: Optional[int] = None,
        order: Optional[Union[str, _models.SortOrder]] = None,
        order_by: Optional[Union[str, _models.MeterOrderBy]] = None,
        include_deleted: Optional[bool] = None,
        **kwargs: Any
    ) -> List[_models.Meter]:
        """List meters.

        List meters.

        :keyword page: Page index.

         Default is 1. Default value is None.
        :paramtype page: int
        :keyword page_size: The maximum number of items per page.

         Default is 100. Default value is None.
        :paramtype page_size: int
        :keyword order: The order direction. Known values are: "ASC" and "DESC". Default value is None.
        :paramtype order: str or ~openmeter.models.SortOrder
        :keyword order_by: The order by field. Known values are: "key", "name", "aggregation",
         "createdAt", and "updatedAt". Default value is None.
        :paramtype order_by: str or ~openmeter.models.MeterOrderBy
        :keyword include_deleted: Include deleted meters. Default value is None.
        :paramtype include_deleted: bool
        :return: list of Meter
        :rtype: list[~openmeter._generated.models.Meter]
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[List[_models.Meter]] = kwargs.pop("cls", None)

        _request = build_meters_list_request(
            page=page,
            page_size=page_size,
            order=order,
            order_by=order_by,
            include_deleted=include_deleted,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(List[_models.Meter], response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def get(self, meter_id_or_slug: str, **kwargs: Any) -> _models.Meter:
        """Get meter.

        Get a meter by ID or slug.

        :param meter_id_or_slug: Required.
        :type meter_id_or_slug: str
        :return: Meter. The Meter is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Meter
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.Meter] = kwargs.pop("cls", None)

        _request = build_meters_get_request(
            meter_id_or_slug=meter_id_or_slug,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.Meter, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    @overload
    async def create(
        self, meter: _models.MeterCreate, *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.Meter:
        """Create meter.

        Create a meter.

        :param meter: Required.
        :type meter: ~openmeter._generated.models.MeterCreate
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Meter. The Meter is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Meter
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def create(self, meter: JSON, *, content_type: str = "application/json", **kwargs: Any) -> _models.Meter:
        """Create meter.

        Create a meter.

        :param meter: Required.
        :type meter: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Meter. The Meter is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Meter
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def create(self, meter: IO[bytes], *, content_type: str = "application/json", **kwargs: Any) -> _models.Meter:
        """Create meter.

        Create a meter.

        :param meter: Required.
        :type meter: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Meter. The Meter is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Meter
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def create(self, meter: Union[_models.MeterCreate, JSON, IO[bytes]], **kwargs: Any) -> _models.Meter:
        """Create meter.

        Create a meter.

        :param meter: Is one of the following types: MeterCreate, JSON, IO[bytes] Required.
        :type meter: ~openmeter._generated.models.MeterCreate or JSON or IO[bytes]
        :return: Meter. The Meter is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Meter
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[_models.Meter] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(meter, (IOBase, bytes)):
            _content = meter
        else:
            _content = json.dumps(meter, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_meters_create_request(
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [201]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.Meter, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    @overload
    async def update(
        self,
        meter_id_or_slug: str,
        meter: _models.MeterUpdate,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> _models.Meter:
        """Update meter.

        Update a meter.

        :param meter_id_or_slug: Required.
        :type meter_id_or_slug: str
        :param meter: Required.
        :type meter: ~openmeter._generated.models.MeterUpdate
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Meter. The Meter is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Meter
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def update(
        self, meter_id_or_slug: str, meter: JSON, *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.Meter:
        """Update meter.

        Update a meter.

        :param meter_id_or_slug: Required.
        :type meter_id_or_slug: str
        :param meter: Required.
        :type meter: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Meter. The Meter is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Meter
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def update(
        self, meter_id_or_slug: str, meter: IO[bytes], *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.Meter:
        """Update meter.

        Update a meter.

        :param meter_id_or_slug: Required.
        :type meter_id_or_slug: str
        :param meter: Required.
        :type meter: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Meter. The Meter is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Meter
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def update(
        self, meter_id_or_slug: str, meter: Union[_models.MeterUpdate, JSON, IO[bytes]], **kwargs: Any
    ) -> _models.Meter:
        """Update meter.

        Update a meter.

        :param meter_id_or_slug: Required.
        :type meter_id_or_slug: str
        :param meter: Is one of the following types: MeterUpdate, JSON, IO[bytes] Required.
        :type meter: ~openmeter._generated.models.MeterUpdate or JSON or IO[bytes]
        :return: Meter. The Meter is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Meter
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[_models.Meter] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(meter, (IOBase, bytes)):
            _content = meter
        else:
            _content = json.dumps(meter, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_meters_update_request(
            meter_id_or_slug=meter_id_or_slug,
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.Meter, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def delete(self, meter_id_or_slug: str, **kwargs: Any) -> None:
        """Delete meter.

        Delete a meter.

        :param meter_id_or_slug: Required.
        :type meter_id_or_slug: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[None] = kwargs.pop("cls", None)

        _request = build_meters_delete_request(
            meter_id_or_slug=meter_id_or_slug,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = False
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [204]:
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if cls:
            return cls(pipeline_response, None, {})  # type: ignore

    async def query_json(
        self,
        meter_id_or_slug: str,
        *,
        client_id: Optional[str] = None,
        from_parameter: Optional[datetime.datetime] = None,
        to: Optional[datetime.datetime] = None,
        window_size: Optional[Union[str, _models.WindowSize]] = None,
        window_time_zone: Optional[str] = None,
        subject: Optional[List[str]] = None,
        filter_customer_id: Optional[List[str]] = None,
        filter_group_by: Optional[dict[str, str]] = None,
        advanced_meter_group_by_filters: Optional[dict[str, _models.FilterString]] = None,
        group_by: Optional[List[str]] = None,
        **kwargs: Any
    ) -> _models.MeterQueryResult:
        """Query meter.

        Query meter for usage.

        :param meter_id_or_slug: Required.
        :type meter_id_or_slug: str
        :keyword client_id: Client ID
         Useful to track progress of a query. Default value is None.
        :paramtype client_id: str
        :keyword from_parameter: Start date-time in RFC 3339 format.

         Inclusive.

         For example: ?from=2025-01-01T00%3A00%3A00.000Z. Default value is None.
        :paramtype from_parameter: ~datetime.datetime
        :keyword to: End date-time in RFC 3339 format.

         Inclusive.

         For example: ?to=2025-02-01T00%3A00%3A00.000Z. Default value is None.
        :paramtype to: ~datetime.datetime
        :keyword window_size: If not specified, a single usage aggregate will be returned for the
         entirety of the specified period for each subject and group.

         For example: ?windowSize=DAY. Known values are: "MINUTE", "HOUR", "DAY", and "MONTH". Default
         value is None.
        :paramtype window_size: str or ~openmeter.models.WindowSize
        :keyword window_time_zone: The value is the name of the time zone as defined in the IANA Time
         Zone Database (`http://www.iana.org/time-zones <http://www.iana.org/time-zones>`_).
         If not specified, the UTC timezone will be used.

         For example: ?windowTimeZone=UTC. Default value is None.
        :paramtype window_time_zone: str
        :keyword subject: Filtering by multiple subjects.

         For example: ?subject=subject-1&subject=subject-2. Default value is None.
        :paramtype subject: list[str]
        :keyword filter_customer_id: Filtering by multiple customers.

         For example: ?filterCustomerId=customer-1&filterCustomerId=customer-2. Default value is None.
        :paramtype filter_customer_id: list[str]
        :keyword filter_group_by: Simple filter for group bys with exact match.

         For example: ?filterGroupBy[vendor]=openai&filterGroupBy[model]=gpt-4-turbo. Default value is
         None.
        :paramtype filter_group_by: dict[str, str]
        :keyword advanced_meter_group_by_filters: Advanced meter group by filters. Default value is
         None.
        :paramtype advanced_meter_group_by_filters: dict[str,
         ~openmeter._generated.models.FilterString]
        :keyword group_by: If not specified a single aggregate will be returned for each subject and
         time window.
         ``subject`` is a reserved group by value.

         For example: ?groupBy=subject&groupBy=model. Default value is None.
        :paramtype group_by: list[str]
        :return: MeterQueryResult. The MeterQueryResult is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.MeterQueryResult
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.MeterQueryResult] = kwargs.pop("cls", None)

        _request = build_meters_query_json_request(
            meter_id_or_slug=meter_id_or_slug,
            client_id=client_id,
            from_parameter=from_parameter,
            to=to,
            window_size=window_size,
            window_time_zone=window_time_zone,
            subject=subject,
            filter_customer_id=filter_customer_id,
            filter_group_by=filter_group_by,
            advanced_meter_group_by_filters=advanced_meter_group_by_filters,
            group_by=group_by,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        response_headers = {}
        response_headers["content-type"] = self._deserialize("str", response.headers.get("content-type"))

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.MeterQueryResult, response.json())

        if cls:
            return cls(pipeline_response, deserialized, response_headers)  # type: ignore

        return deserialized  # type: ignore

    async def query_csv(
        self,
        meter_id_or_slug: str,
        *,
        client_id: Optional[str] = None,
        from_parameter: Optional[datetime.datetime] = None,
        to: Optional[datetime.datetime] = None,
        window_size: Optional[Union[str, _models.WindowSize]] = None,
        window_time_zone: Optional[str] = None,
        subject: Optional[List[str]] = None,
        filter_customer_id: Optional[List[str]] = None,
        filter_group_by: Optional[dict[str, str]] = None,
        advanced_meter_group_by_filters: Optional[dict[str, _models.FilterString]] = None,
        group_by: Optional[List[str]] = None,
        **kwargs: Any
    ) -> str:
        """query_csv.

        :param meter_id_or_slug: Required.
        :type meter_id_or_slug: str
        :keyword client_id: Client ID
         Useful to track progress of a query. Default value is None.
        :paramtype client_id: str
        :keyword from_parameter: Start date-time in RFC 3339 format.

         Inclusive.

         For example: ?from=2025-01-01T00%3A00%3A00.000Z. Default value is None.
        :paramtype from_parameter: ~datetime.datetime
        :keyword to: End date-time in RFC 3339 format.

         Inclusive.

         For example: ?to=2025-02-01T00%3A00%3A00.000Z. Default value is None.
        :paramtype to: ~datetime.datetime
        :keyword window_size: If not specified, a single usage aggregate will be returned for the
         entirety of the specified period for each subject and group.

         For example: ?windowSize=DAY. Known values are: "MINUTE", "HOUR", "DAY", and "MONTH". Default
         value is None.
        :paramtype window_size: str or ~openmeter.models.WindowSize
        :keyword window_time_zone: The value is the name of the time zone as defined in the IANA Time
         Zone Database (`http://www.iana.org/time-zones <http://www.iana.org/time-zones>`_).
         If not specified, the UTC timezone will be used.

         For example: ?windowTimeZone=UTC. Default value is None.
        :paramtype window_time_zone: str
        :keyword subject: Filtering by multiple subjects.

         For example: ?subject=subject-1&subject=subject-2. Default value is None.
        :paramtype subject: list[str]
        :keyword filter_customer_id: Filtering by multiple customers.

         For example: ?filterCustomerId=customer-1&filterCustomerId=customer-2. Default value is None.
        :paramtype filter_customer_id: list[str]
        :keyword filter_group_by: Simple filter for group bys with exact match.

         For example: ?filterGroupBy[vendor]=openai&filterGroupBy[model]=gpt-4-turbo. Default value is
         None.
        :paramtype filter_group_by: dict[str, str]
        :keyword advanced_meter_group_by_filters: Advanced meter group by filters. Default value is
         None.
        :paramtype advanced_meter_group_by_filters: dict[str,
         ~openmeter._generated.models.FilterString]
        :keyword group_by: If not specified a single aggregate will be returned for each subject and
         time window.
         ``subject`` is a reserved group by value.

         For example: ?groupBy=subject&groupBy=model. Default value is None.
        :paramtype group_by: list[str]
        :return: str
        :rtype: str
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[str] = kwargs.pop("cls", None)

        _request = build_meters_query_csv_request(
            meter_id_or_slug=meter_id_or_slug,
            client_id=client_id,
            from_parameter=from_parameter,
            to=to,
            window_size=window_size,
            window_time_zone=window_time_zone,
            subject=subject,
            filter_customer_id=filter_customer_id,
            filter_group_by=filter_group_by,
            advanced_meter_group_by_filters=advanced_meter_group_by_filters,
            group_by=group_by,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        response_headers = {}
        response_headers["content-type"] = self._deserialize("str", response.headers.get("content-type"))

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(str, response.text())

        if cls:
            return cls(pipeline_response, deserialized, response_headers)  # type: ignore

        return deserialized  # type: ignore

    @overload
    async def query(
        self,
        meter_id_or_slug: str,
        request: _models.MeterQueryRequest,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> _models.MeterQueryResult:
        """Query meter.

        query.

        :param meter_id_or_slug: Required.
        :type meter_id_or_slug: str
        :param request: Required.
        :type request: ~openmeter._generated.models.MeterQueryRequest
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: MeterQueryResult. The MeterQueryResult is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.MeterQueryResult
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def query(
        self, meter_id_or_slug: str, request: JSON, *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.MeterQueryResult:
        """Query meter.

        query.

        :param meter_id_or_slug: Required.
        :type meter_id_or_slug: str
        :param request: Required.
        :type request: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: MeterQueryResult. The MeterQueryResult is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.MeterQueryResult
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def query(
        self, meter_id_or_slug: str, request: IO[bytes], *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.MeterQueryResult:
        """Query meter.

        query.

        :param meter_id_or_slug: Required.
        :type meter_id_or_slug: str
        :param request: Required.
        :type request: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: MeterQueryResult. The MeterQueryResult is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.MeterQueryResult
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def query(
        self, meter_id_or_slug: str, request: Union[_models.MeterQueryRequest, JSON, IO[bytes]], **kwargs: Any
    ) -> _models.MeterQueryResult:
        """Query meter.

        query.

        :param meter_id_or_slug: Required.
        :type meter_id_or_slug: str
        :param request: Is one of the following types: MeterQueryRequest, JSON, IO[bytes] Required.
        :type request: ~openmeter._generated.models.MeterQueryRequest or JSON or IO[bytes]
        :return: MeterQueryResult. The MeterQueryResult is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.MeterQueryResult
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[_models.MeterQueryResult] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(request, (IOBase, bytes)):
            _content = request
        else:
            _content = json.dumps(request, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_meters_query_request(
            meter_id_or_slug=meter_id_or_slug,
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        response_headers = {}
        response_headers["content-type"] = self._deserialize("str", response.headers.get("content-type"))

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.MeterQueryResult, response.json())

        if cls:
            return cls(pipeline_response, deserialized, response_headers)  # type: ignore

        return deserialized  # type: ignore

    async def list_subjects(
        self,
        meter_id_or_slug: str,
        *,
        from_parameter: Optional[datetime.datetime] = None,
        to: Optional[datetime.datetime] = None,
        **kwargs: Any
    ) -> List[str]:
        """List meter subjects.

        List subjects for a meter.

        :param meter_id_or_slug: Required.
        :type meter_id_or_slug: str
        :keyword from_parameter: Start date-time in RFC 3339 format.

         Inclusive. Defaults to the beginning of time.

         For example: ?from=2025-01-01T00%3A00%3A00.000Z. Default value is None.
        :paramtype from_parameter: ~datetime.datetime
        :keyword to: End date-time in RFC 3339 format.

         Inclusive.

         For example: ?to=2025-02-01T00%3A00%3A00.000Z. Default value is None.
        :paramtype to: ~datetime.datetime
        :return: list of str
        :rtype: list[str]
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[List[str]] = kwargs.pop("cls", None)

        _request = build_meters_list_subjects_request(
            meter_id_or_slug=meter_id_or_slug,
            from_parameter=from_parameter,
            to=to,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(List[str], response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def list_group_by_values(
        self,
        meter_id_or_slug: str,
        group_by_key: str,
        *,
        from_parameter: Optional[datetime.datetime] = None,
        to: Optional[datetime.datetime] = None,
        **kwargs: Any
    ) -> List[str]:
        """List meter group by values.

        List meter group by values.

        :param meter_id_or_slug: Required.
        :type meter_id_or_slug: str
        :param group_by_key: Required.
        :type group_by_key: str
        :keyword from_parameter: Start date-time in RFC 3339 format.

         Inclusive. Defaults to 24 hours ago.

         For example: ?from=2025-01-01T00%3A00%3A00.000Z. Default value is None.
        :paramtype from_parameter: ~datetime.datetime
        :keyword to: End date-time in RFC 3339 format.

         Inclusive.

         For example: ?to=2025-02-01T00%3A00%3A00.000Z. Default value is None.
        :paramtype to: ~datetime.datetime
        :return: list of str
        :rtype: list[str]
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[List[str]] = kwargs.pop("cls", None)

        _request = build_meters_list_group_by_values_request(
            meter_id_or_slug=meter_id_or_slug,
            group_by_key=group_by_key,
            from_parameter=from_parameter,
            to=to,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(List[str], response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore


class SubjectsOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`subjects` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

    async def list(self, **kwargs: Any) -> List[_models.Subject]:
        """List subjects.

        List subjects.

        :return: list of Subject
        :rtype: list[~openmeter._generated.models.Subject]
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[List[_models.Subject]] = kwargs.pop("cls", None)

        _request = build_subjects_list_request(
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(List[_models.Subject], response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def get(self, subject_id_or_key: str, **kwargs: Any) -> _models.Subject:
        """Get subject.

        Get subject by ID or key.

        :param subject_id_or_key: Required.
        :type subject_id_or_key: str
        :return: Subject. The Subject is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Subject
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.Subject] = kwargs.pop("cls", None)

        _request = build_subjects_get_request(
            subject_id_or_key=subject_id_or_key,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.Subject, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    @overload
    async def upsert(
        self, subject: List[_models.SubjectUpsert], *, content_type: str = "application/json", **kwargs: Any
    ) -> List[_models.Subject]:
        """Upsert subject.

        Upserts a subject. Creates or updates subject.

        If the subject doesn't exist, it will be created.
        If the subject exists, it will be partially updated with the provided fields.

        :param subject: Required.
        :type subject: list[~openmeter._generated.models.SubjectUpsert]
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: list of Subject
        :rtype: list[~openmeter._generated.models.Subject]
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def upsert(
        self, subject: List[JSON], *, content_type: str = "application/json", **kwargs: Any
    ) -> List[_models.Subject]:
        """Upsert subject.

        Upserts a subject. Creates or updates subject.

        If the subject doesn't exist, it will be created.
        If the subject exists, it will be partially updated with the provided fields.

        :param subject: Required.
        :type subject: list[JSON]
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: list of Subject
        :rtype: list[~openmeter._generated.models.Subject]
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def upsert(
        self, subject: IO[bytes], *, content_type: str = "application/json", **kwargs: Any
    ) -> List[_models.Subject]:
        """Upsert subject.

        Upserts a subject. Creates or updates subject.

        If the subject doesn't exist, it will be created.
        If the subject exists, it will be partially updated with the provided fields.

        :param subject: Required.
        :type subject: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: list of Subject
        :rtype: list[~openmeter._generated.models.Subject]
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def upsert(
        self, subject: Union[List[_models.SubjectUpsert], List[JSON], IO[bytes]], **kwargs: Any
    ) -> List[_models.Subject]:
        """Upsert subject.

        Upserts a subject. Creates or updates subject.

        If the subject doesn't exist, it will be created.
        If the subject exists, it will be partially updated with the provided fields.

        :param subject: Is one of the following types: [SubjectUpsert], [JSON], IO[bytes] Required.
        :type subject: list[~openmeter._generated.models.SubjectUpsert] or list[JSON] or IO[bytes]
        :return: list of Subject
        :rtype: list[~openmeter._generated.models.Subject]
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[List[_models.Subject]] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(subject, (IOBase, bytes)):
            _content = subject
        else:
            _content = json.dumps(subject, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_subjects_upsert_request(
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(List[_models.Subject], response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def delete(self, subject_id_or_key: str, **kwargs: Any) -> None:
        """Delete subject.

        Delete subject by ID or key.

        :param subject_id_or_key: Required.
        :type subject_id_or_key: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[None] = kwargs.pop("cls", None)

        _request = build_subjects_delete_request(
            subject_id_or_key=subject_id_or_key,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = False
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [204]:
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if cls:
            return cls(pipeline_response, None, {})  # type: ignore


class DebugOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`debug` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

    async def metrics(self, **kwargs: Any) -> str:
        """Get event metrics.

        Returns debug metrics (in OpenMetrics format) like the number of ingested events since
        mindnight UTC.

        The OpenMetrics Counter(s) reset every day at midnight UTC.

        :return: str
        :rtype: str
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[str] = kwargs.pop("cls", None)

        _request = build_debug_metrics_request(
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        response_headers = {}
        response_headers["content-type"] = self._deserialize("str", response.headers.get("content-type"))

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(str, response.text())

        if cls:
            return cls(pipeline_response, deserialized, response_headers)  # type: ignore

        return deserialized  # type: ignore


class AppAppsOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`apps` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

    async def list(
        self, *, page: Optional[int] = None, page_size: Optional[int] = None, **kwargs: Any
    ) -> _models.AppPaginatedResponse:
        """List apps.

        List apps.

        :keyword page: Page index.

         Default is 1. Default value is None.
        :paramtype page: int
        :keyword page_size: The maximum number of items per page.

         Default is 100. Default value is None.
        :paramtype page_size: int
        :return: AppPaginatedResponse. The AppPaginatedResponse is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.AppPaginatedResponse
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.AppPaginatedResponse] = kwargs.pop("cls", None)

        _request = build_app_apps_list_request(
            page=page,
            page_size=page_size,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.AppPaginatedResponse, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def get(self, id: str, **kwargs: Any) -> "_types.App":
        """Get app.

        Get the app.

        :param id: Required.
        :type id: str
        :return: StripeApp or SandboxApp or CustomInvoicingApp
        :rtype: ~openmeter._generated.models.StripeApp or ~openmeter._generated.models.SandboxApp or
         ~openmeter._generated.models.CustomInvoicingApp
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType["_types.App"] = kwargs.pop("cls", None)

        _request = build_app_apps_get_request(
            id=id,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize("_types.App", response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    @overload
    async def update(
        self, id: str, app: _models.StripeAppReplaceUpdate, *, content_type: str = "application/json", **kwargs: Any
    ) -> "_types.App":
        """Update app.

        Update an app.

        :param id: Required.
        :type id: str
        :param app: Required.
        :type app: ~openmeter._generated.models.StripeAppReplaceUpdate
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: StripeApp or SandboxApp or CustomInvoicingApp
        :rtype: ~openmeter._generated.models.StripeApp or ~openmeter._generated.models.SandboxApp or
         ~openmeter._generated.models.CustomInvoicingApp
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def update(
        self, id: str, app: _models.SandboxAppReplaceUpdate, *, content_type: str = "application/json", **kwargs: Any
    ) -> "_types.App":
        """Update app.

        Update an app.

        :param id: Required.
        :type id: str
        :param app: Required.
        :type app: ~openmeter._generated.models.SandboxAppReplaceUpdate
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: StripeApp or SandboxApp or CustomInvoicingApp
        :rtype: ~openmeter._generated.models.StripeApp or ~openmeter._generated.models.SandboxApp or
         ~openmeter._generated.models.CustomInvoicingApp
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def update(
        self,
        id: str,
        app: _models.CustomInvoicingAppReplaceUpdate,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> "_types.App":
        """Update app.

        Update an app.

        :param id: Required.
        :type id: str
        :param app: Required.
        :type app: ~openmeter._generated.models.CustomInvoicingAppReplaceUpdate
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: StripeApp or SandboxApp or CustomInvoicingApp
        :rtype: ~openmeter._generated.models.StripeApp or ~openmeter._generated.models.SandboxApp or
         ~openmeter._generated.models.CustomInvoicingApp
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def update(self, id: str, app: "_types.AppReplaceUpdate", **kwargs: Any) -> "_types.App":
        """Update app.

        Update an app.

        :param id: Required.
        :type id: str
        :param app: Is one of the following types: StripeAppReplaceUpdate, SandboxAppReplaceUpdate,
         CustomInvoicingAppReplaceUpdate Required.
        :type app: ~openmeter._generated.models.StripeAppReplaceUpdate or
         ~openmeter._generated.models.SandboxAppReplaceUpdate or
         ~openmeter._generated.models.CustomInvoicingAppReplaceUpdate
        :return: StripeApp or SandboxApp or CustomInvoicingApp
        :rtype: ~openmeter._generated.models.StripeApp or ~openmeter._generated.models.SandboxApp or
         ~openmeter._generated.models.CustomInvoicingApp
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType["_types.App"] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(app, _models.StripeAppReplaceUpdate):
            _content = json.dumps(app, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore
        elif isinstance(app, _models.SandboxAppReplaceUpdate):
            _content = json.dumps(app, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore
        elif isinstance(app, _models.CustomInvoicingAppReplaceUpdate):
            _content = json.dumps(app, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_app_apps_update_request(
            id=id,
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize("_types.App", response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def uninstall(self, id: str, **kwargs: Any) -> None:
        """Uninstall app.

        Uninstall an app.

        :param id: Required.
        :type id: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[None] = kwargs.pop("cls", None)

        _request = build_app_apps_uninstall_request(
            id=id,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = False
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [204]:
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if cls:
            return cls(pipeline_response, None, {})  # type: ignore


class AppAppStripeOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`app_stripe` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

    @overload
    async def webhook(
        self, id: str, body: _models.StripeWebhookEvent, *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.StripeWebhookResponse:
        """Stripe webhook.

        Handle stripe webhooks for apps.

        :param id: Required.
        :type id: str
        :param body: Required.
        :type body: ~openmeter._generated.models.StripeWebhookEvent
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: StripeWebhookResponse. The StripeWebhookResponse is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.StripeWebhookResponse
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def webhook(
        self, id: str, body: JSON, *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.StripeWebhookResponse:
        """Stripe webhook.

        Handle stripe webhooks for apps.

        :param id: Required.
        :type id: str
        :param body: Required.
        :type body: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: StripeWebhookResponse. The StripeWebhookResponse is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.StripeWebhookResponse
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def webhook(
        self, id: str, body: IO[bytes], *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.StripeWebhookResponse:
        """Stripe webhook.

        Handle stripe webhooks for apps.

        :param id: Required.
        :type id: str
        :param body: Required.
        :type body: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: StripeWebhookResponse. The StripeWebhookResponse is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.StripeWebhookResponse
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def webhook(
        self, id: str, body: Union[_models.StripeWebhookEvent, JSON, IO[bytes]], **kwargs: Any
    ) -> _models.StripeWebhookResponse:
        """Stripe webhook.

        Handle stripe webhooks for apps.

        :param id: Required.
        :type id: str
        :param body: Is one of the following types: StripeWebhookEvent, JSON, IO[bytes] Required.
        :type body: ~openmeter._generated.models.StripeWebhookEvent or JSON or IO[bytes]
        :return: StripeWebhookResponse. The StripeWebhookResponse is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.StripeWebhookResponse
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[_models.StripeWebhookResponse] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(body, (IOBase, bytes)):
            _content = body
        else:
            _content = json.dumps(body, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_app_app_stripe_webhook_request(
            id=id,
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.StripeWebhookResponse, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    @overload
    async def update_stripe_api_key(
        self, id: str, request: _models.StripeAPIKeyInput, *, content_type: str = "application/json", **kwargs: Any
    ) -> None:
        """Update Stripe API key.

        Update the Stripe API key.

        :param id: Required.
        :type id: str
        :param request: Required.
        :type request: ~openmeter._generated.models.StripeAPIKeyInput
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def update_stripe_api_key(
        self, id: str, request: JSON, *, content_type: str = "application/json", **kwargs: Any
    ) -> None:
        """Update Stripe API key.

        Update the Stripe API key.

        :param id: Required.
        :type id: str
        :param request: Required.
        :type request: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def update_stripe_api_key(
        self, id: str, request: IO[bytes], *, content_type: str = "application/json", **kwargs: Any
    ) -> None:
        """Update Stripe API key.

        Update the Stripe API key.

        :param id: Required.
        :type id: str
        :param request: Required.
        :type request: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def update_stripe_api_key(
        self, id: str, request: Union[_models.StripeAPIKeyInput, JSON, IO[bytes]], **kwargs: Any
    ) -> None:
        """Update Stripe API key.

        Update the Stripe API key.

        :param id: Required.
        :type id: str
        :param request: Is one of the following types: StripeAPIKeyInput, JSON, IO[bytes] Required.
        :type request: ~openmeter._generated.models.StripeAPIKeyInput or JSON or IO[bytes]
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[None] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(request, (IOBase, bytes)):
            _content = request
        else:
            _content = json.dumps(request, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_app_app_stripe_update_stripe_api_key_request(
            id=id,
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = False
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [204]:
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if cls:
            return cls(pipeline_response, None, {})  # type: ignore

    @overload
    async def create_checkout_session(
        self, body: _models.CreateStripeCheckoutSessionRequest, *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.CreateStripeCheckoutSessionResult:
        """Create checkout session.

        Create checkout session.

        :param body: Required.
        :type body: ~openmeter._generated.models.CreateStripeCheckoutSessionRequest
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: CreateStripeCheckoutSessionResult. The CreateStripeCheckoutSessionResult is compatible
         with MutableMapping
        :rtype: ~openmeter._generated.models.CreateStripeCheckoutSessionResult
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def create_checkout_session(
        self, body: JSON, *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.CreateStripeCheckoutSessionResult:
        """Create checkout session.

        Create checkout session.

        :param body: Required.
        :type body: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: CreateStripeCheckoutSessionResult. The CreateStripeCheckoutSessionResult is compatible
         with MutableMapping
        :rtype: ~openmeter._generated.models.CreateStripeCheckoutSessionResult
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def create_checkout_session(
        self, body: IO[bytes], *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.CreateStripeCheckoutSessionResult:
        """Create checkout session.

        Create checkout session.

        :param body: Required.
        :type body: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: CreateStripeCheckoutSessionResult. The CreateStripeCheckoutSessionResult is compatible
         with MutableMapping
        :rtype: ~openmeter._generated.models.CreateStripeCheckoutSessionResult
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def create_checkout_session(
        self, body: Union[_models.CreateStripeCheckoutSessionRequest, JSON, IO[bytes]], **kwargs: Any
    ) -> _models.CreateStripeCheckoutSessionResult:
        """Create checkout session.

        Create checkout session.

        :param body: Is one of the following types: CreateStripeCheckoutSessionRequest, JSON, IO[bytes]
         Required.
        :type body: ~openmeter._generated.models.CreateStripeCheckoutSessionRequest or JSON or
         IO[bytes]
        :return: CreateStripeCheckoutSessionResult. The CreateStripeCheckoutSessionResult is compatible
         with MutableMapping
        :rtype: ~openmeter._generated.models.CreateStripeCheckoutSessionResult
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[_models.CreateStripeCheckoutSessionResult] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(body, (IOBase, bytes)):
            _content = body
        else:
            _content = json.dumps(body, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_app_app_stripe_create_checkout_session_request(
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [201]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.CreateStripeCheckoutSessionResult, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore


class AppMarketplaceOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`marketplace` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

    async def list(
        self, *, page: Optional[int] = None, page_size: Optional[int] = None, **kwargs: Any
    ) -> _models.MarketplaceListingPaginatedResponse:
        """List available apps.

        List available apps of the app marketplace.

        :keyword page: Page index.

         Default is 1. Default value is None.
        :paramtype page: int
        :keyword page_size: The maximum number of items per page.

         Default is 100. Default value is None.
        :paramtype page_size: int
        :return: MarketplaceListingPaginatedResponse. The MarketplaceListingPaginatedResponse is
         compatible with MutableMapping
        :rtype: ~openmeter._generated.models.MarketplaceListingPaginatedResponse
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.MarketplaceListingPaginatedResponse] = kwargs.pop("cls", None)

        _request = build_app_marketplace_list_request(
            page=page,
            page_size=page_size,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.MarketplaceListingPaginatedResponse, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def get(self, type: Union[str, _models.AppType], **kwargs: Any) -> _models.MarketplaceListing:
        """Get app details by type.

        Get a marketplace listing by type.

        :param type: Known values are: "stripe", "sandbox", and "custom_invoicing". Required.
        :type type: str or ~openmeter.models.AppType
        :return: MarketplaceListing. The MarketplaceListing is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.MarketplaceListing
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.MarketplaceListing] = kwargs.pop("cls", None)

        _request = build_app_marketplace_get_request(
            type=type,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.MarketplaceListing, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def get_o_auth2_install_url(
        self, type: Union[str, _models.AppType], **kwargs: Any
    ) -> _models.ClientAppStartResponse:
        """Get OAuth2 install URL.

        Install an app via OAuth.
        Returns a URL to start the OAuth 2.0 flow.

        :param type: Known values are: "stripe", "sandbox", and "custom_invoicing". Required.
        :type type: str or ~openmeter.models.AppType
        :return: ClientAppStartResponse. The ClientAppStartResponse is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.ClientAppStartResponse
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.ClientAppStartResponse] = kwargs.pop("cls", None)

        _request = build_app_marketplace_get_o_auth2_install_url_request(
            type=type,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.ClientAppStartResponse, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def authorize_o_auth2_install(
        self,
        type: Union[str, _models.AppType],
        *,
        state: Optional[str] = None,
        code: Optional[str] = None,
        error: Optional[Union[str, _models.OAuth2AuthorizationCodeGrantErrorType]] = None,
        error_description: Optional[str] = None,
        error_uri: Optional[str] = None,
        **kwargs: Any
    ) -> None:
        """Install app via OAuth2.

        Authorize OAuth2 code.
        Verifies the OAuth code and exchanges it for a token and refresh token.

        :param type: The type of the app to install. Known values are: "stripe", "sandbox", and
         "custom_invoicing". Required.
        :type type: str or ~openmeter.models.AppType
        :keyword state: Required if the "state" parameter was present in the client authorization
         request.
         The exact value received from the client:

         Unique, randomly generated, opaque, and non-guessable string that is sent
         when starting an authentication request and validated when processing the response. Default
         value is None.
        :paramtype state: str
        :keyword code: Authorization code which the client will later exchange for an access token.
         Required with the success response. Default value is None.
        :paramtype code: str
        :keyword error: Error code.
         Required with the error response. Known values are: "invalid_request", "unauthorized_client",
         "access_denied", "unsupported_response_type", "invalid_scope", "server_error", and
         "temporarily_unavailable". Default value is None.
        :paramtype error: str or ~openmeter.models.OAuth2AuthorizationCodeGrantErrorType
        :keyword error_description: Optional human-readable text providing additional information,
         used to assist the client developer in understanding the error that occurred. Default value is
         None.
        :paramtype error_description: str
        :keyword error_uri: Optional uri identifying a human-readable web page with
         information about the error, used to provide the client
         developer with additional information about the error. Default value is None.
        :paramtype error_uri: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[None] = kwargs.pop("cls", None)

        _request = build_app_marketplace_authorize_o_auth2_install_request(
            type=type,
            state=state,
            code=code,
            error=error,
            error_description=error_description,
            error_uri=error_uri,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = False
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [303]:
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if cls:
            return cls(pipeline_response, None, {})  # type: ignore

    @overload
    async def install_with_api_key(
        self,
        type: Union[str, _models.AppType],
        _: _models.InstallWithApiKeyRequest,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> _models.MarketplaceInstallResponse:
        """Install app via API key.

        Install an marketplace app via API Key.

        :param type: The type of the app to install. Known values are: "stripe", "sandbox", and
         "custom_invoicing". Required.
        :type type: str or ~openmeter.models.AppType
        :param _: Required.
        :type _: ~openmeter._generated.models.InstallWithApiKeyRequest
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: MarketplaceInstallResponse. The MarketplaceInstallResponse is compatible with
         MutableMapping
        :rtype: ~openmeter._generated.models.MarketplaceInstallResponse
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def install_with_api_key(
        self, type: Union[str, _models.AppType], _: JSON, *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.MarketplaceInstallResponse:
        """Install app via API key.

        Install an marketplace app via API Key.

        :param type: The type of the app to install. Known values are: "stripe", "sandbox", and
         "custom_invoicing". Required.
        :type type: str or ~openmeter.models.AppType
        :param _: Required.
        :type _: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: MarketplaceInstallResponse. The MarketplaceInstallResponse is compatible with
         MutableMapping
        :rtype: ~openmeter._generated.models.MarketplaceInstallResponse
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def install_with_api_key(
        self, type: Union[str, _models.AppType], _: IO[bytes], *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.MarketplaceInstallResponse:
        """Install app via API key.

        Install an marketplace app via API Key.

        :param type: The type of the app to install. Known values are: "stripe", "sandbox", and
         "custom_invoicing". Required.
        :type type: str or ~openmeter.models.AppType
        :param _: Required.
        :type _: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: MarketplaceInstallResponse. The MarketplaceInstallResponse is compatible with
         MutableMapping
        :rtype: ~openmeter._generated.models.MarketplaceInstallResponse
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def install_with_api_key(
        self,
        type: Union[str, _models.AppType],
        _: Union[_models.InstallWithApiKeyRequest, JSON, IO[bytes]],
        **kwargs: Any
    ) -> _models.MarketplaceInstallResponse:
        """Install app via API key.

        Install an marketplace app via API Key.

        :param type: The type of the app to install. Known values are: "stripe", "sandbox", and
         "custom_invoicing". Required.
        :type type: str or ~openmeter.models.AppType
        :param _: Is one of the following types: InstallWithApiKeyRequest, JSON, IO[bytes] Required.
        :type _: ~openmeter._generated.models.InstallWithApiKeyRequest or JSON or IO[bytes]
        :return: MarketplaceInstallResponse. The MarketplaceInstallResponse is compatible with
         MutableMapping
        :rtype: ~openmeter._generated.models.MarketplaceInstallResponse
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[_models.MarketplaceInstallResponse] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(_, (IOBase, bytes)):
            _content = _
        else:
            _content = json.dumps(_, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_app_marketplace_install_with_api_key_request(
            type=type,
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.MarketplaceInstallResponse, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    @overload
    async def install(
        self,
        type: Union[str, _models.AppType],
        _: _models.MarketplaceInstallRequestPayload,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> _models.MarketplaceInstallResponse:
        """Install app.

        Install an app from the marketplace.

        :param type: The type of the app to install. Known values are: "stripe", "sandbox", and
         "custom_invoicing". Required.
        :type type: str or ~openmeter.models.AppType
        :param _: Required.
        :type _: ~openmeter._generated.models.MarketplaceInstallRequestPayload
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: MarketplaceInstallResponse. The MarketplaceInstallResponse is compatible with
         MutableMapping
        :rtype: ~openmeter._generated.models.MarketplaceInstallResponse
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def install(
        self, type: Union[str, _models.AppType], _: JSON, *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.MarketplaceInstallResponse:
        """Install app.

        Install an app from the marketplace.

        :param type: The type of the app to install. Known values are: "stripe", "sandbox", and
         "custom_invoicing". Required.
        :type type: str or ~openmeter.models.AppType
        :param _: Required.
        :type _: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: MarketplaceInstallResponse. The MarketplaceInstallResponse is compatible with
         MutableMapping
        :rtype: ~openmeter._generated.models.MarketplaceInstallResponse
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def install(
        self, type: Union[str, _models.AppType], _: IO[bytes], *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.MarketplaceInstallResponse:
        """Install app.

        Install an app from the marketplace.

        :param type: The type of the app to install. Known values are: "stripe", "sandbox", and
         "custom_invoicing". Required.
        :type type: str or ~openmeter.models.AppType
        :param _: Required.
        :type _: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: MarketplaceInstallResponse. The MarketplaceInstallResponse is compatible with
         MutableMapping
        :rtype: ~openmeter._generated.models.MarketplaceInstallResponse
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def install(
        self,
        type: Union[str, _models.AppType],
        _: Union[_models.MarketplaceInstallRequestPayload, JSON, IO[bytes]],
        **kwargs: Any
    ) -> _models.MarketplaceInstallResponse:
        """Install app.

        Install an app from the marketplace.

        :param type: The type of the app to install. Known values are: "stripe", "sandbox", and
         "custom_invoicing". Required.
        :type type: str or ~openmeter.models.AppType
        :param _: Is one of the following types: MarketplaceInstallRequestPayload, JSON, IO[bytes]
         Required.
        :type _: ~openmeter._generated.models.MarketplaceInstallRequestPayload or JSON or IO[bytes]
        :return: MarketplaceInstallResponse. The MarketplaceInstallResponse is compatible with
         MutableMapping
        :rtype: ~openmeter._generated.models.MarketplaceInstallResponse
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[_models.MarketplaceInstallResponse] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(_, (IOBase, bytes)):
            _content = _
        else:
            _content = json.dumps(_, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_app_marketplace_install_request(
            type=type,
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.MarketplaceInstallResponse, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore


class AppAppCustomInvoicingOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`app_custom_invoicing` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

    @overload
    async def draft_syncronized(
        self,
        invoice_id: str,
        body: _models.CustomInvoicingDraftSynchronizedRequest,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> None:
        """Submit draft synchronization results.

        draft_syncronized.

        :param invoice_id: Required.
        :type invoice_id: str
        :param body: Required.
        :type body: ~openmeter._generated.models.CustomInvoicingDraftSynchronizedRequest
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def draft_syncronized(
        self, invoice_id: str, body: JSON, *, content_type: str = "application/json", **kwargs: Any
    ) -> None:
        """Submit draft synchronization results.

        draft_syncronized.

        :param invoice_id: Required.
        :type invoice_id: str
        :param body: Required.
        :type body: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def draft_syncronized(
        self, invoice_id: str, body: IO[bytes], *, content_type: str = "application/json", **kwargs: Any
    ) -> None:
        """Submit draft synchronization results.

        draft_syncronized.

        :param invoice_id: Required.
        :type invoice_id: str
        :param body: Required.
        :type body: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def draft_syncronized(
        self,
        invoice_id: str,
        body: Union[_models.CustomInvoicingDraftSynchronizedRequest, JSON, IO[bytes]],
        **kwargs: Any
    ) -> None:
        """Submit draft synchronization results.

        draft_syncronized.

        :param invoice_id: Required.
        :type invoice_id: str
        :param body: Is one of the following types: CustomInvoicingDraftSynchronizedRequest, JSON,
         IO[bytes] Required.
        :type body: ~openmeter._generated.models.CustomInvoicingDraftSynchronizedRequest or JSON or
         IO[bytes]
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[None] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(body, (IOBase, bytes)):
            _content = body
        else:
            _content = json.dumps(body, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_app_app_custom_invoicing_draft_syncronized_request(
            invoice_id=invoice_id,
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = False
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [204]:
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if cls:
            return cls(pipeline_response, None, {})  # type: ignore

    @overload
    async def finalized(
        self,
        invoice_id: str,
        body: _models.CustomInvoicingFinalizedRequest,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> None:
        """Submit issuing synchronization results.

        finalized.

        :param invoice_id: Required.
        :type invoice_id: str
        :param body: Required.
        :type body: ~openmeter._generated.models.CustomInvoicingFinalizedRequest
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def finalized(
        self, invoice_id: str, body: JSON, *, content_type: str = "application/json", **kwargs: Any
    ) -> None:
        """Submit issuing synchronization results.

        finalized.

        :param invoice_id: Required.
        :type invoice_id: str
        :param body: Required.
        :type body: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def finalized(
        self, invoice_id: str, body: IO[bytes], *, content_type: str = "application/json", **kwargs: Any
    ) -> None:
        """Submit issuing synchronization results.

        finalized.

        :param invoice_id: Required.
        :type invoice_id: str
        :param body: Required.
        :type body: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def finalized(
        self, invoice_id: str, body: Union[_models.CustomInvoicingFinalizedRequest, JSON, IO[bytes]], **kwargs: Any
    ) -> None:
        """Submit issuing synchronization results.

        finalized.

        :param invoice_id: Required.
        :type invoice_id: str
        :param body: Is one of the following types: CustomInvoicingFinalizedRequest, JSON, IO[bytes]
         Required.
        :type body: ~openmeter._generated.models.CustomInvoicingFinalizedRequest or JSON or IO[bytes]
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[None] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(body, (IOBase, bytes)):
            _content = body
        else:
            _content = json.dumps(body, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_app_app_custom_invoicing_finalized_request(
            invoice_id=invoice_id,
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = False
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [204]:
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if cls:
            return cls(pipeline_response, None, {})  # type: ignore

    @overload
    async def payment_status(
        self,
        invoice_id: str,
        body: _models.CustomInvoicingUpdatePaymentStatusRequest,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> None:
        """Update payment status.

        payment_status.

        :param invoice_id: Required.
        :type invoice_id: str
        :param body: Required.
        :type body: ~openmeter._generated.models.CustomInvoicingUpdatePaymentStatusRequest
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def payment_status(
        self, invoice_id: str, body: JSON, *, content_type: str = "application/json", **kwargs: Any
    ) -> None:
        """Update payment status.

        payment_status.

        :param invoice_id: Required.
        :type invoice_id: str
        :param body: Required.
        :type body: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def payment_status(
        self, invoice_id: str, body: IO[bytes], *, content_type: str = "application/json", **kwargs: Any
    ) -> None:
        """Update payment status.

        payment_status.

        :param invoice_id: Required.
        :type invoice_id: str
        :param body: Required.
        :type body: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def payment_status(
        self,
        invoice_id: str,
        body: Union[_models.CustomInvoicingUpdatePaymentStatusRequest, JSON, IO[bytes]],
        **kwargs: Any
    ) -> None:
        """Update payment status.

        payment_status.

        :param invoice_id: Required.
        :type invoice_id: str
        :param body: Is one of the following types: CustomInvoicingUpdatePaymentStatusRequest, JSON,
         IO[bytes] Required.
        :type body: ~openmeter._generated.models.CustomInvoicingUpdatePaymentStatusRequest or JSON or
         IO[bytes]
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[None] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(body, (IOBase, bytes)):
            _content = body
        else:
            _content = json.dumps(body, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_app_app_custom_invoicing_payment_status_request(
            invoice_id=invoice_id,
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = False
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [204]:
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if cls:
            return cls(pipeline_response, None, {})  # type: ignore


class CustomerCustomersAppsOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`customers_apps` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

    async def list_app_data(
        self,
        customer_id_or_key: "_types.ULIDOrExternalKey",
        *,
        page: Optional[int] = None,
        page_size: Optional[int] = None,
        type: Optional[Union[str, _models.AppType]] = None,
        **kwargs: Any
    ) -> _models.CustomerAppDataPaginatedResponse:
        """List customer app data.

        List customers app data.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :keyword page: Page index.

         Default is 1. Default value is None.
        :paramtype page: int
        :keyword page_size: The maximum number of items per page.

         Default is 100. Default value is None.
        :paramtype page_size: int
        :keyword type: Filter customer data by app type. Known values are: "stripe", "sandbox", and
         "custom_invoicing". Default value is None.
        :paramtype type: str or ~openmeter.models.AppType
        :return: CustomerAppDataPaginatedResponse. The CustomerAppDataPaginatedResponse is compatible
         with MutableMapping
        :rtype: ~openmeter._generated.models.CustomerAppDataPaginatedResponse
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.CustomerAppDataPaginatedResponse] = kwargs.pop("cls", None)

        _request = build_customer_customers_apps_list_app_data_request(
            customer_id_or_key=customer_id_or_key,
            page=page,
            page_size=page_size,
            type=type,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.CustomerAppDataPaginatedResponse, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    @overload
    async def upsert_app_data(
        self,
        customer_id_or_key: "_types.ULIDOrExternalKey",
        app_data: List["_types.CustomerAppData"],
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> List["_types.CustomerAppData"]:
        """Upsert customer app data.

        Upsert customer app data.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :param app_data: Required.
        :type app_data: list[~openmeter._generated.models.StripeCustomerAppData or
         ~openmeter._generated.models.SandboxCustomerAppData or
         ~openmeter._generated.models.CustomInvoicingCustomerAppData]
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: list of StripeCustomerAppData or SandboxCustomerAppData or
         CustomInvoicingCustomerAppData
        :rtype: list[~openmeter._generated.models.StripeCustomerAppData or
         ~openmeter._generated.models.SandboxCustomerAppData or
         ~openmeter._generated.models.CustomInvoicingCustomerAppData]
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def upsert_app_data(
        self,
        customer_id_or_key: "_types.ULIDOrExternalKey",
        app_data: IO[bytes],
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> List["_types.CustomerAppData"]:
        """Upsert customer app data.

        Upsert customer app data.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :param app_data: Required.
        :type app_data: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: list of StripeCustomerAppData or SandboxCustomerAppData or
         CustomInvoicingCustomerAppData
        :rtype: list[~openmeter._generated.models.StripeCustomerAppData or
         ~openmeter._generated.models.SandboxCustomerAppData or
         ~openmeter._generated.models.CustomInvoicingCustomerAppData]
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def upsert_app_data(
        self,
        customer_id_or_key: "_types.ULIDOrExternalKey",
        app_data: Union[List["_types.CustomerAppData"], IO[bytes]],
        **kwargs: Any
    ) -> List["_types.CustomerAppData"]:
        """Upsert customer app data.

        Upsert customer app data.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :param app_data: Is either a ["_types.CustomerAppData"] type or a IO[bytes] type. Required.
        :type app_data: list[~openmeter._generated.models.StripeCustomerAppData or
         ~openmeter._generated.models.SandboxCustomerAppData or
         ~openmeter._generated.models.CustomInvoicingCustomerAppData] or IO[bytes]
        :return: list of StripeCustomerAppData or SandboxCustomerAppData or
         CustomInvoicingCustomerAppData
        :rtype: list[~openmeter._generated.models.StripeCustomerAppData or
         ~openmeter._generated.models.SandboxCustomerAppData or
         ~openmeter._generated.models.CustomInvoicingCustomerAppData]
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[List["_types.CustomerAppData"]] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(app_data, (IOBase, bytes)):
            _content = app_data
        else:
            _content = json.dumps(app_data, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_customer_customers_apps_upsert_app_data_request(
            customer_id_or_key=customer_id_or_key,
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(List["_types.CustomerAppData"], response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def delete_app_data(self, customer_id_or_key: "_types.ULIDOrExternalKey", app_id: str, **kwargs: Any) -> None:
        """Delete customer app data.

        Delete customer app data.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :param app_id: Required.
        :type app_id: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[None] = kwargs.pop("cls", None)

        _request = build_customer_customers_apps_delete_app_data_request(
            customer_id_or_key=customer_id_or_key,
            app_id=app_id,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = False
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [204]:
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if cls:
            return cls(pipeline_response, None, {})  # type: ignore


class CustomerCustomersOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`customers` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

    @overload
    async def create(
        self, customer: _models.CustomerCreate, *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.Customer:
        """Create customer.

        Create a new customer.

        :param customer: Required.
        :type customer: ~openmeter._generated.models.CustomerCreate
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Customer. The Customer is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Customer
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def create(
        self, customer: JSON, *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.Customer:
        """Create customer.

        Create a new customer.

        :param customer: Required.
        :type customer: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Customer. The Customer is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Customer
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def create(
        self, customer: IO[bytes], *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.Customer:
        """Create customer.

        Create a new customer.

        :param customer: Required.
        :type customer: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Customer. The Customer is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Customer
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def create(self, customer: Union[_models.CustomerCreate, JSON, IO[bytes]], **kwargs: Any) -> _models.Customer:
        """Create customer.

        Create a new customer.

        :param customer: Is one of the following types: CustomerCreate, JSON, IO[bytes] Required.
        :type customer: ~openmeter._generated.models.CustomerCreate or JSON or IO[bytes]
        :return: Customer. The Customer is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Customer
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[_models.Customer] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(customer, (IOBase, bytes)):
            _content = customer
        else:
            _content = json.dumps(customer, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_customer_customers_create_request(
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [201]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.Customer, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def list(
        self,
        *,
        page: Optional[int] = None,
        page_size: Optional[int] = None,
        order: Optional[Union[str, _models.SortOrder]] = None,
        order_by: Optional[Union[str, _models.CustomerOrderBy]] = None,
        include_deleted: Optional[bool] = None,
        key: Optional[str] = None,
        name: Optional[str] = None,
        primary_email: Optional[str] = None,
        subject: Optional[str] = None,
        plan_key: Optional[str] = None,
        expand: Optional[List[Union[str, _models.CustomerExpand]]] = None,
        **kwargs: Any
    ) -> _models.CustomerPaginatedResponse:
        """List customers.

        List customers.

        :keyword page: Page index.

         Default is 1. Default value is None.
        :paramtype page: int
        :keyword page_size: The maximum number of items per page.

         Default is 100. Default value is None.
        :paramtype page_size: int
        :keyword order: The order direction. Known values are: "ASC" and "DESC". Default value is None.
        :paramtype order: str or ~openmeter.models.SortOrder
        :keyword order_by: The order by field. Known values are: "id", "name", and "createdAt". Default
         value is None.
        :paramtype order_by: str or ~openmeter.models.CustomerOrderBy
        :keyword include_deleted: Include deleted customers. Default value is None.
        :paramtype include_deleted: bool
        :keyword key: Filter customers by key.
         Case-insensitive partial match. Default value is None.
        :paramtype key: str
        :keyword name: Filter customers by name.
         Case-insensitive partial match. Default value is None.
        :paramtype name: str
        :keyword primary_email: Filter customers by primary email.
         Case-insensitive partial match. Default value is None.
        :paramtype primary_email: str
        :keyword subject: Filter customers by usage attribution subject.
         Case-insensitive partial match. Default value is None.
        :paramtype subject: str
        :keyword plan_key: Filter customers by the plan key of their susbcription. Default value is
         None.
        :paramtype plan_key: str
        :keyword expand: What parts of the list output to expand in listings. Default value is None.
        :paramtype expand: list[str or ~openmeter.models.CustomerExpand]
        :return: CustomerPaginatedResponse. The CustomerPaginatedResponse is compatible with
         MutableMapping
        :rtype: ~openmeter._generated.models.CustomerPaginatedResponse
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.CustomerPaginatedResponse] = kwargs.pop("cls", None)

        _request = build_customer_customers_list_request(
            page=page,
            page_size=page_size,
            order=order,
            order_by=order_by,
            include_deleted=include_deleted,
            key=key,
            name=name,
            primary_email=primary_email,
            subject=subject,
            plan_key=plan_key,
            expand=expand,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.CustomerPaginatedResponse, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def get(
        self,
        customer_id_or_key: "_types.ULIDOrExternalKey",
        *,
        expand: Optional[List[Union[str, _models.CustomerExpand]]] = None,
        **kwargs: Any
    ) -> _models.Customer:
        """Get customer.

        Get a customer by ID or key.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :keyword expand: What parts of the customer output to expand. Default value is None.
        :paramtype expand: list[str or ~openmeter.models.CustomerExpand]
        :return: Customer. The Customer is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Customer
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.Customer] = kwargs.pop("cls", None)

        _request = build_customer_customers_get_request(
            customer_id_or_key=customer_id_or_key,
            expand=expand,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.Customer, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    @overload
    async def update(
        self,
        customer_id_or_key: "_types.ULIDOrExternalKey",
        customer: _models.CustomerReplaceUpdate,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> _models.Customer:
        """Update customer.

        Update a customer by ID.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :param customer: Required.
        :type customer: ~openmeter._generated.models.CustomerReplaceUpdate
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Customer. The Customer is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Customer
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def update(
        self,
        customer_id_or_key: "_types.ULIDOrExternalKey",
        customer: JSON,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> _models.Customer:
        """Update customer.

        Update a customer by ID.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :param customer: Required.
        :type customer: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Customer. The Customer is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Customer
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def update(
        self,
        customer_id_or_key: "_types.ULIDOrExternalKey",
        customer: IO[bytes],
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> _models.Customer:
        """Update customer.

        Update a customer by ID.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :param customer: Required.
        :type customer: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Customer. The Customer is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Customer
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def update(
        self,
        customer_id_or_key: "_types.ULIDOrExternalKey",
        customer: Union[_models.CustomerReplaceUpdate, JSON, IO[bytes]],
        **kwargs: Any
    ) -> _models.Customer:
        """Update customer.

        Update a customer by ID.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :param customer: Is one of the following types: CustomerReplaceUpdate, JSON, IO[bytes]
         Required.
        :type customer: ~openmeter._generated.models.CustomerReplaceUpdate or JSON or IO[bytes]
        :return: Customer. The Customer is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Customer
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[_models.Customer] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(customer, (IOBase, bytes)):
            _content = customer
        else:
            _content = json.dumps(customer, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_customer_customers_update_request(
            customer_id_or_key=customer_id_or_key,
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.Customer, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def delete(self, customer_id_or_key: "_types.ULIDOrExternalKey", **kwargs: Any) -> None:
        """Delete customer.

        Delete a customer by ID.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[None] = kwargs.pop("cls", None)

        _request = build_customer_customers_delete_request(
            customer_id_or_key=customer_id_or_key,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = False
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [204]:
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if cls:
            return cls(pipeline_response, None, {})  # type: ignore

    async def list_customer_subscriptions(
        self,
        customer_id_or_key: "_types.ULIDOrExternalKey",
        *,
        status: Optional[List[Union[str, _models.SubscriptionStatus]]] = None,
        order: Optional[Union[str, _models.SortOrder]] = None,
        order_by: Optional[Union[str, _models.CustomerSubscriptionOrderBy]] = None,
        page: Optional[int] = None,
        page_size: Optional[int] = None,
        **kwargs: Any
    ) -> _models.SubscriptionPaginatedResponse:
        """List customer subscriptions.

        Lists all subscriptions for a customer.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :keyword status: Default value is None.
        :paramtype status: list[str or ~openmeter.models.SubscriptionStatus]
        :keyword order: The order direction. Known values are: "ASC" and "DESC". Default value is None.
        :paramtype order: str or ~openmeter.models.SortOrder
        :keyword order_by: The order by field. Known values are: "activeFrom" and "activeTo". Default
         value is None.
        :paramtype order_by: str or ~openmeter.models.CustomerSubscriptionOrderBy
        :keyword page: Page index.

         Default is 1. Default value is None.
        :paramtype page: int
        :keyword page_size: The maximum number of items per page.

         Default is 100. Default value is None.
        :paramtype page_size: int
        :return: SubscriptionPaginatedResponse. The SubscriptionPaginatedResponse is compatible with
         MutableMapping
        :rtype: ~openmeter._generated.models.SubscriptionPaginatedResponse
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.SubscriptionPaginatedResponse] = kwargs.pop("cls", None)

        _request = build_customer_customers_list_customer_subscriptions_request(
            customer_id_or_key=customer_id_or_key,
            status=status,
            order=order,
            order_by=order_by,
            page=page,
            page_size=page_size,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.SubscriptionPaginatedResponse, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore


class CustomerCustomersStripeOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`customers_stripe` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

    async def get(self, customer_id_or_key: "_types.ULIDOrExternalKey", **kwargs: Any) -> _models.StripeCustomerAppData:
        """Get customer stripe app data.

        Get stripe app data for a customer.
        Only returns data if the customer billing profile is linked to a stripe app.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :return: StripeCustomerAppData. The StripeCustomerAppData is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.StripeCustomerAppData
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.StripeCustomerAppData] = kwargs.pop("cls", None)

        _request = build_customer_customers_stripe_get_request(
            customer_id_or_key=customer_id_or_key,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.StripeCustomerAppData, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    @overload
    async def upsert(
        self,
        customer_id_or_key: "_types.ULIDOrExternalKey",
        app_data: _models.StripeCustomerAppDataBase,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> _models.StripeCustomerAppData:
        """Upsert customer stripe app data.

        Upsert stripe app data for a customer.
        Only updates data if the customer billing profile is linked to a stripe app.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :param app_data: Required.
        :type app_data: ~openmeter._generated.models.StripeCustomerAppDataBase
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: StripeCustomerAppData. The StripeCustomerAppData is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.StripeCustomerAppData
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def upsert(
        self,
        customer_id_or_key: "_types.ULIDOrExternalKey",
        app_data: JSON,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> _models.StripeCustomerAppData:
        """Upsert customer stripe app data.

        Upsert stripe app data for a customer.
        Only updates data if the customer billing profile is linked to a stripe app.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :param app_data: Required.
        :type app_data: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: StripeCustomerAppData. The StripeCustomerAppData is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.StripeCustomerAppData
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def upsert(
        self,
        customer_id_or_key: "_types.ULIDOrExternalKey",
        app_data: IO[bytes],
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> _models.StripeCustomerAppData:
        """Upsert customer stripe app data.

        Upsert stripe app data for a customer.
        Only updates data if the customer billing profile is linked to a stripe app.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :param app_data: Required.
        :type app_data: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: StripeCustomerAppData. The StripeCustomerAppData is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.StripeCustomerAppData
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def upsert(
        self,
        customer_id_or_key: "_types.ULIDOrExternalKey",
        app_data: Union[_models.StripeCustomerAppDataBase, JSON, IO[bytes]],
        **kwargs: Any
    ) -> _models.StripeCustomerAppData:
        """Upsert customer stripe app data.

        Upsert stripe app data for a customer.
        Only updates data if the customer billing profile is linked to a stripe app.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :param app_data: Is one of the following types: StripeCustomerAppDataBase, JSON, IO[bytes]
         Required.
        :type app_data: ~openmeter._generated.models.StripeCustomerAppDataBase or JSON or IO[bytes]
        :return: StripeCustomerAppData. The StripeCustomerAppData is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.StripeCustomerAppData
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[_models.StripeCustomerAppData] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(app_data, (IOBase, bytes)):
            _content = app_data
        else:
            _content = json.dumps(app_data, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_customer_customers_stripe_upsert_request(
            customer_id_or_key=customer_id_or_key,
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.StripeCustomerAppData, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    @overload
    async def create_portal_session(
        self,
        customer_id_or_key: "_types.ULIDOrExternalKey",
        params: _models.CreateStripeCustomerPortalSessionParams,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> _models.StripeCustomerPortalSession:
        """Create Stripe customer portal session.

        Create Stripe customer portal session.
        Only returns URL if the customer billing profile is linked to a stripe app and customer.

        Useful to redirect the customer to the Stripe customer portal to manage their payment methods,
        change their billing address and access their invoice history.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :param params: Required.
        :type params: ~openmeter._generated.models.CreateStripeCustomerPortalSessionParams
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: StripeCustomerPortalSession. The StripeCustomerPortalSession is compatible with
         MutableMapping
        :rtype: ~openmeter._generated.models.StripeCustomerPortalSession
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def create_portal_session(
        self,
        customer_id_or_key: "_types.ULIDOrExternalKey",
        params: JSON,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> _models.StripeCustomerPortalSession:
        """Create Stripe customer portal session.

        Create Stripe customer portal session.
        Only returns URL if the customer billing profile is linked to a stripe app and customer.

        Useful to redirect the customer to the Stripe customer portal to manage their payment methods,
        change their billing address and access their invoice history.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :param params: Required.
        :type params: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: StripeCustomerPortalSession. The StripeCustomerPortalSession is compatible with
         MutableMapping
        :rtype: ~openmeter._generated.models.StripeCustomerPortalSession
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def create_portal_session(
        self,
        customer_id_or_key: "_types.ULIDOrExternalKey",
        params: IO[bytes],
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> _models.StripeCustomerPortalSession:
        """Create Stripe customer portal session.

        Create Stripe customer portal session.
        Only returns URL if the customer billing profile is linked to a stripe app and customer.

        Useful to redirect the customer to the Stripe customer portal to manage their payment methods,
        change their billing address and access their invoice history.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :param params: Required.
        :type params: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: StripeCustomerPortalSession. The StripeCustomerPortalSession is compatible with
         MutableMapping
        :rtype: ~openmeter._generated.models.StripeCustomerPortalSession
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def create_portal_session(
        self,
        customer_id_or_key: "_types.ULIDOrExternalKey",
        params: Union[_models.CreateStripeCustomerPortalSessionParams, JSON, IO[bytes]],
        **kwargs: Any
    ) -> _models.StripeCustomerPortalSession:
        """Create Stripe customer portal session.

        Create Stripe customer portal session.
        Only returns URL if the customer billing profile is linked to a stripe app and customer.

        Useful to redirect the customer to the Stripe customer portal to manage their payment methods,
        change their billing address and access their invoice history.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :param params: Is one of the following types: CreateStripeCustomerPortalSessionParams, JSON,
         IO[bytes] Required.
        :type params: ~openmeter._generated.models.CreateStripeCustomerPortalSessionParams or JSON or
         IO[bytes]
        :return: StripeCustomerPortalSession. The StripeCustomerPortalSession is compatible with
         MutableMapping
        :rtype: ~openmeter._generated.models.StripeCustomerPortalSession
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[_models.StripeCustomerPortalSession] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(params, (IOBase, bytes)):
            _content = params
        else:
            _content = json.dumps(params, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_customer_customers_stripe_create_portal_session_request(
            customer_id_or_key=customer_id_or_key,
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [201]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.StripeCustomerPortalSession, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore


class ProductCatalogFeaturesOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`features` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

    async def list(
        self,
        *,
        meter_slug: Optional[List[str]] = None,
        include_archived: Optional[bool] = None,
        page: Optional[int] = None,
        page_size: Optional[int] = None,
        offset: Optional[int] = None,
        limit: Optional[int] = None,
        order: Optional[Union[str, _models.SortOrder]] = None,
        order_by: Optional[Union[str, _models.FeatureOrderBy]] = None,
        **kwargs: Any
    ) -> "_types.ListFeaturesResult":
        """List features.

        List features.

        :keyword meter_slug: Filter by meterSlug. Default value is None.
        :paramtype meter_slug: list[str]
        :keyword include_archived: Include archived features in response. Default value is None.
        :paramtype include_archived: bool
        :keyword page: Page index.

         Default is 1. Default value is None.
        :paramtype page: int
        :keyword page_size: The maximum number of items per page.

         Default is 100. Default value is None.
        :paramtype page_size: int
        :keyword offset: Number of items to skip.

         Default is 0. Default value is None.
        :paramtype offset: int
        :keyword limit: Number of items to return.

         Default is 100. Default value is None.
        :paramtype limit: int
        :keyword order: The order direction. Known values are: "ASC" and "DESC". Default value is None.
        :paramtype order: str or ~openmeter.models.SortOrder
        :keyword order_by: The order by field. Known values are: "id", "key", "name", "createdAt", and
         "updatedAt". Default value is None.
        :paramtype order_by: str or ~openmeter.models.FeatureOrderBy
        :return: list of Feature or FeaturePaginatedResponse
        :rtype: list[~openmeter._generated.models.Feature] or
         ~openmeter._generated.models.FeaturePaginatedResponse
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType["_types.ListFeaturesResult"] = kwargs.pop("cls", None)

        _request = build_product_catalog_features_list_request(
            meter_slug=meter_slug,
            include_archived=include_archived,
            page=page,
            page_size=page_size,
            offset=offset,
            limit=limit,
            order=order,
            order_by=order_by,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize("_types.ListFeaturesResult", response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    @overload
    async def create(
        self, feature: _models.FeatureCreateInputs, *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.Feature:
        """Create feature.

        Features are either metered or static. A feature is metered if meterSlug is provided at
        creation.
        For metered features you can pass additional filters that will be applied when calculating
        feature usage, based on the meter's groupBy fields.
        Only meters with SUM and COUNT aggregation are supported for features.
        Features cannot be updated later, only archived.

        :param feature: Required.
        :type feature: ~openmeter._generated.models.FeatureCreateInputs
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Feature. The Feature is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Feature
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def create(self, feature: JSON, *, content_type: str = "application/json", **kwargs: Any) -> _models.Feature:
        """Create feature.

        Features are either metered or static. A feature is metered if meterSlug is provided at
        creation.
        For metered features you can pass additional filters that will be applied when calculating
        feature usage, based on the meter's groupBy fields.
        Only meters with SUM and COUNT aggregation are supported for features.
        Features cannot be updated later, only archived.

        :param feature: Required.
        :type feature: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Feature. The Feature is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Feature
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def create(
        self, feature: IO[bytes], *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.Feature:
        """Create feature.

        Features are either metered or static. A feature is metered if meterSlug is provided at
        creation.
        For metered features you can pass additional filters that will be applied when calculating
        feature usage, based on the meter's groupBy fields.
        Only meters with SUM and COUNT aggregation are supported for features.
        Features cannot be updated later, only archived.

        :param feature: Required.
        :type feature: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Feature. The Feature is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Feature
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def create(
        self, feature: Union[_models.FeatureCreateInputs, JSON, IO[bytes]], **kwargs: Any
    ) -> _models.Feature:
        """Create feature.

        Features are either metered or static. A feature is metered if meterSlug is provided at
        creation.
        For metered features you can pass additional filters that will be applied when calculating
        feature usage, based on the meter's groupBy fields.
        Only meters with SUM and COUNT aggregation are supported for features.
        Features cannot be updated later, only archived.

        :param feature: Is one of the following types: FeatureCreateInputs, JSON, IO[bytes] Required.
        :type feature: ~openmeter._generated.models.FeatureCreateInputs or JSON or IO[bytes]
        :return: Feature. The Feature is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Feature
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[_models.Feature] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(feature, (IOBase, bytes)):
            _content = feature
        else:
            _content = json.dumps(feature, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_product_catalog_features_create_request(
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [201]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.Feature, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def get(self, feature_id: str, **kwargs: Any) -> _models.Feature:
        """Get feature.

        Get a feature by ID.

        :param feature_id: Required.
        :type feature_id: str
        :return: Feature. The Feature is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Feature
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.Feature] = kwargs.pop("cls", None)

        _request = build_product_catalog_features_get_request(
            feature_id=feature_id,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.Feature, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def delete(self, feature_id: str, **kwargs: Any) -> None:
        """Delete feature.

        Archive a feature by ID.

        Once a feature is archived it cannot be unarchived. If a feature is archived, new entitlements
        cannot be created for it, but archiving the feature does not affect existing entitlements.
        This means, if you want to create a new feature with the same key, and then create entitlements
        for it, the previous entitlements have to be deleted first on a per subject basis.

        :param feature_id: Required.
        :type feature_id: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[None] = kwargs.pop("cls", None)

        _request = build_product_catalog_features_delete_request(
            feature_id=feature_id,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = False
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [204]:
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if cls:
            return cls(pipeline_response, None, {})  # type: ignore


class ProductCatalogPlansOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`plans` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

    def list(
        self,
        *,
        include_deleted: Optional[bool] = None,
        id: Optional[List[str]] = None,
        key: Optional[List[str]] = None,
        key_version: Optional[dict[str, List[int]]] = None,
        status: Optional[List[Union[str, _models.PlanStatus]]] = None,
        currency: Optional[List[str]] = None,
        page: Optional[int] = None,
        page_size: Optional[int] = None,
        order: Optional[Union[str, _models.SortOrder]] = None,
        order_by: Optional[Union[str, _models.PlanOrderBy]] = None,
        **kwargs: Any
    ) -> AsyncItemPaged["_models.Plan"]:
        """List plans.

        List all plans.

        :keyword include_deleted: Include deleted plans in response.

         Usage: ``?includeDeleted=true``. Default value is None.
        :paramtype include_deleted: bool
        :keyword id: Filter by plan.id attribute. Default value is None.
        :paramtype id: list[str]
        :keyword key: Filter by plan.key attribute. Default value is None.
        :paramtype key: list[str]
        :keyword key_version: Filter by plan.key and plan.version attributes. Default value is None.
        :paramtype key_version: dict[str, list[int]]
        :keyword status: Only return plans with the given status.

         Usage:

         * `?status=active`: return only the currently active plan
         * `?status=draft`: return only the draft plan
         * `?status=archived`: return only the archived plans. Default value is None.
        :paramtype status: list[str or ~openmeter.models.PlanStatus]
        :keyword currency: Filter by plan.currency attribute. Default value is None.
        :paramtype currency: list[str]
        :keyword page: Page index.

         Default is 1. Default value is None.
        :paramtype page: int
        :keyword page_size: The maximum number of items per page.

         Default is 100. Default value is None.
        :paramtype page_size: int
        :keyword order: The order direction. Known values are: "ASC" and "DESC". Default value is None.
        :paramtype order: str or ~openmeter.models.SortOrder
        :keyword order_by: The order by field. Known values are: "id", "key", "version", "created_at",
         and "updated_at". Default value is None.
        :paramtype order_by: str or ~openmeter.models.PlanOrderBy
        :return: An iterator like instance of Plan
        :rtype: ~corehttp.paging.AsyncItemPaged[~openmeter._generated.models.Plan]
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[List[_models.Plan]] = kwargs.pop("cls", None)

        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        def prepare_request(next_link=None):
            if not next_link:

                _request = build_product_catalog_plans_list_request(
                    include_deleted=include_deleted,
                    id=id,
                    key=key,
                    key_version=key_version,
                    status=status,
                    currency=currency,
                    page=page,
                    page_size=page_size,
                    order=order,
                    order_by=order_by,
                    headers=_headers,
                    params=_params,
                )
                path_format_arguments = {
                    "endpoint": self._serialize.url(
                        "self._config.endpoint", self._config.endpoint, "str", skip_quote=True
                    ),
                }
                _request.url = self._client.format_url(_request.url, **path_format_arguments)

            else:
                _request = HttpRequest("GET", next_link)
                path_format_arguments = {
                    "endpoint": self._serialize.url(
                        "self._config.endpoint", self._config.endpoint, "str", skip_quote=True
                    ),
                }
                _request.url = self._client.format_url(_request.url, **path_format_arguments)

            return _request

        async def extract_data(pipeline_response):
            deserialized = pipeline_response.http_response.json()
            list_of_elem = _deserialize(List[_models.Plan], deserialized.get("items", []))
            if cls:
                list_of_elem = cls(list_of_elem)  # type: ignore
            return None, AsyncList(list_of_elem)

        async def get_next(next_link=None):
            _request = prepare_request(next_link)

            _stream = False
            pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)
            response = pipeline_response.http_response

            if response.status_code not in [200]:
                map_error(status_code=response.status_code, response=response, error_map=error_map)
                error = None
                if response.status_code == 400:
                    error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
                elif response.status_code == 401:
                    error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                    raise ClientAuthenticationError(response=response, model=error)
                if response.status_code == 403:
                    error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
                elif response.status_code == 500:
                    error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
                elif response.status_code == 503:
                    error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
                elif response.status_code == 412:
                    error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
                else:
                    error = _failsafe_deserialize(
                        _models.UnexpectedProblemResponse,
                        response,
                    )
                raise HttpResponseError(response=response, model=error)

            return pipeline_response

        return AsyncItemPaged(get_next, extract_data)

    @overload
    async def create(
        self, request: _models.PlanCreate, *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.Plan:
        """Create a plan.

        Create a new plan.

        :param request: Required.
        :type request: ~openmeter._generated.models.PlanCreate
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Plan. The Plan is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Plan
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def create(self, request: JSON, *, content_type: str = "application/json", **kwargs: Any) -> _models.Plan:
        """Create a plan.

        Create a new plan.

        :param request: Required.
        :type request: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Plan. The Plan is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Plan
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def create(
        self, request: IO[bytes], *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.Plan:
        """Create a plan.

        Create a new plan.

        :param request: Required.
        :type request: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Plan. The Plan is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Plan
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def create(self, request: Union[_models.PlanCreate, JSON, IO[bytes]], **kwargs: Any) -> _models.Plan:
        """Create a plan.

        Create a new plan.

        :param request: Is one of the following types: PlanCreate, JSON, IO[bytes] Required.
        :type request: ~openmeter._generated.models.PlanCreate or JSON or IO[bytes]
        :return: Plan. The Plan is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Plan
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[_models.Plan] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(request, (IOBase, bytes)):
            _content = request
        else:
            _content = json.dumps(request, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_product_catalog_plans_create_request(
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [201]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.Plan, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    @overload
    async def update(
        self, plan_id: str, body: _models.PlanReplaceUpdate, *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.Plan:
        """Update a plan.

        Update plan by id.

        :param plan_id: Required.
        :type plan_id: str
        :param body: Required.
        :type body: ~openmeter._generated.models.PlanReplaceUpdate
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Plan. The Plan is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Plan
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def update(
        self, plan_id: str, body: JSON, *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.Plan:
        """Update a plan.

        Update plan by id.

        :param plan_id: Required.
        :type plan_id: str
        :param body: Required.
        :type body: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Plan. The Plan is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Plan
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def update(
        self, plan_id: str, body: IO[bytes], *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.Plan:
        """Update a plan.

        Update plan by id.

        :param plan_id: Required.
        :type plan_id: str
        :param body: Required.
        :type body: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Plan. The Plan is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Plan
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def update(
        self, plan_id: str, body: Union[_models.PlanReplaceUpdate, JSON, IO[bytes]], **kwargs: Any
    ) -> _models.Plan:
        """Update a plan.

        Update plan by id.

        :param plan_id: Required.
        :type plan_id: str
        :param body: Is one of the following types: PlanReplaceUpdate, JSON, IO[bytes] Required.
        :type body: ~openmeter._generated.models.PlanReplaceUpdate or JSON or IO[bytes]
        :return: Plan. The Plan is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Plan
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[_models.Plan] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(body, (IOBase, bytes)):
            _content = body
        else:
            _content = json.dumps(body, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_product_catalog_plans_update_request(
            plan_id=plan_id,
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.Plan, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def get(self, plan_id: str, *, include_latest: Optional[bool] = None, **kwargs: Any) -> _models.Plan:
        """Get plan.

        Get a plan by id or key. The latest published version is returned if latter is used.

        :param plan_id: Required.
        :type plan_id: str
        :keyword include_latest: Include latest version of the Plan instead of the version in active
         state.

         Usage: ``?includeLatest=true``. Default value is None.
        :paramtype include_latest: bool
        :return: Plan. The Plan is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Plan
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.Plan] = kwargs.pop("cls", None)

        _request = build_product_catalog_plans_get_request(
            plan_id=plan_id,
            include_latest=include_latest,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.Plan, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def delete(self, plan_id: str, **kwargs: Any) -> None:
        """Delete plan.

        Soft delete plan by plan.id.

        Once a plan is deleted it cannot be undeleted.

        :param plan_id: Required.
        :type plan_id: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[None] = kwargs.pop("cls", None)

        _request = build_product_catalog_plans_delete_request(
            plan_id=plan_id,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = False
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [204]:
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if cls:
            return cls(pipeline_response, None, {})  # type: ignore

    async def publish(self, plan_id: str, **kwargs: Any) -> _models.Plan:
        """Publish plan.

        Publish a plan version.

        :param plan_id: Required.
        :type plan_id: str
        :return: Plan. The Plan is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Plan
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.Plan] = kwargs.pop("cls", None)

        _request = build_product_catalog_plans_publish_request(
            plan_id=plan_id,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.Plan, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def archive(self, plan_id: str, **kwargs: Any) -> _models.Plan:
        """Archive plan version.

        Archive a plan version.

        :param plan_id: Required.
        :type plan_id: str
        :return: Plan. The Plan is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Plan
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.Plan] = kwargs.pop("cls", None)

        _request = build_product_catalog_plans_archive_request(
            plan_id=plan_id,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.Plan, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def next(self, plan_id_or_key: str, **kwargs: Any) -> _models.Plan:
        """New draft plan.

        Create a new draft version from plan.
        It returns error if there is already a plan in draft or planId does not reference the latest
        published version.

        :param plan_id_or_key: Required.
        :type plan_id_or_key: str
        :return: Plan. The Plan is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Plan
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.Plan] = kwargs.pop("cls", None)

        _request = build_product_catalog_plans_next_request(
            plan_id_or_key=plan_id_or_key,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [201]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.Plan, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore


class ProductCatalogPlanAddonsOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`plan_addons` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

    async def list(
        self,
        plan_id: str,
        *,
        include_deleted: Optional[bool] = None,
        id: Optional[List[str]] = None,
        key: Optional[List[str]] = None,
        key_version: Optional[dict[str, List[int]]] = None,
        page: Optional[int] = None,
        page_size: Optional[int] = None,
        order: Optional[Union[str, _models.SortOrder]] = None,
        order_by: Optional[Union[str, _models.PlanAddonOrderBy]] = None,
        **kwargs: Any
    ) -> _models.PlanAddonPaginatedResponse:
        """List all available add-ons for plan.

        List all available add-ons for plan.

        :param plan_id: Required.
        :type plan_id: str
        :keyword include_deleted: Include deleted plan add-on assignments.

         Usage: ``?includeDeleted=true``. Default value is None.
        :paramtype include_deleted: bool
        :keyword id: Filter by addon.id attribute. Default value is None.
        :paramtype id: list[str]
        :keyword key: Filter by addon.key attribute. Default value is None.
        :paramtype key: list[str]
        :keyword key_version: Filter by addon.key and addon.version attributes. Default value is None.
        :paramtype key_version: dict[str, list[int]]
        :keyword page: Page index.

         Default is 1. Default value is None.
        :paramtype page: int
        :keyword page_size: The maximum number of items per page.

         Default is 100. Default value is None.
        :paramtype page_size: int
        :keyword order: The order direction. Known values are: "ASC" and "DESC". Default value is None.
        :paramtype order: str or ~openmeter.models.SortOrder
        :keyword order_by: The order by field. Known values are: "id", "key", "version", "created_at",
         and "updated_at". Default value is None.
        :paramtype order_by: str or ~openmeter.models.PlanAddonOrderBy
        :return: PlanAddonPaginatedResponse. The PlanAddonPaginatedResponse is compatible with
         MutableMapping
        :rtype: ~openmeter._generated.models.PlanAddonPaginatedResponse
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.PlanAddonPaginatedResponse] = kwargs.pop("cls", None)

        _request = build_product_catalog_plan_addons_list_request(
            plan_id=plan_id,
            include_deleted=include_deleted,
            id=id,
            key=key,
            key_version=key_version,
            page=page,
            page_size=page_size,
            order=order,
            order_by=order_by,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.PlanAddonPaginatedResponse, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    @overload
    async def create(
        self, plan_id: str, body: _models.PlanAddonCreate, *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.PlanAddon:
        """Create new add-on assignment for plan.

        Create new add-on assignment for plan.

        :param plan_id: Required.
        :type plan_id: str
        :param body: Required.
        :type body: ~openmeter._generated.models.PlanAddonCreate
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: PlanAddon. The PlanAddon is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.PlanAddon
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def create(
        self, plan_id: str, body: JSON, *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.PlanAddon:
        """Create new add-on assignment for plan.

        Create new add-on assignment for plan.

        :param plan_id: Required.
        :type plan_id: str
        :param body: Required.
        :type body: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: PlanAddon. The PlanAddon is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.PlanAddon
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def create(
        self, plan_id: str, body: IO[bytes], *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.PlanAddon:
        """Create new add-on assignment for plan.

        Create new add-on assignment for plan.

        :param plan_id: Required.
        :type plan_id: str
        :param body: Required.
        :type body: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: PlanAddon. The PlanAddon is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.PlanAddon
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def create(
        self, plan_id: str, body: Union[_models.PlanAddonCreate, JSON, IO[bytes]], **kwargs: Any
    ) -> _models.PlanAddon:
        """Create new add-on assignment for plan.

        Create new add-on assignment for plan.

        :param plan_id: Required.
        :type plan_id: str
        :param body: Is one of the following types: PlanAddonCreate, JSON, IO[bytes] Required.
        :type body: ~openmeter._generated.models.PlanAddonCreate or JSON or IO[bytes]
        :return: PlanAddon. The PlanAddon is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.PlanAddon
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[_models.PlanAddon] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(body, (IOBase, bytes)):
            _content = body
        else:
            _content = json.dumps(body, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_product_catalog_plan_addons_create_request(
            plan_id=plan_id,
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [201]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 409:
                error = _failsafe_deserialize(_models.ConflictProblemResponse, response)
                raise ResourceExistsError(response=response, model=error)
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.PlanAddon, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    @overload
    async def update(
        self,
        plan_id: str,
        plan_addon_id: str,
        body: _models.PlanAddonReplaceUpdate,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> _models.PlanAddon:
        """Update add-on assignment for plan.

        Update add-on assignment for plan.

        :param plan_id: Required.
        :type plan_id: str
        :param plan_addon_id: Required.
        :type plan_addon_id: str
        :param body: Required.
        :type body: ~openmeter._generated.models.PlanAddonReplaceUpdate
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: PlanAddon. The PlanAddon is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.PlanAddon
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def update(
        self, plan_id: str, plan_addon_id: str, body: JSON, *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.PlanAddon:
        """Update add-on assignment for plan.

        Update add-on assignment for plan.

        :param plan_id: Required.
        :type plan_id: str
        :param plan_addon_id: Required.
        :type plan_addon_id: str
        :param body: Required.
        :type body: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: PlanAddon. The PlanAddon is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.PlanAddon
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def update(
        self,
        plan_id: str,
        plan_addon_id: str,
        body: IO[bytes],
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> _models.PlanAddon:
        """Update add-on assignment for plan.

        Update add-on assignment for plan.

        :param plan_id: Required.
        :type plan_id: str
        :param plan_addon_id: Required.
        :type plan_addon_id: str
        :param body: Required.
        :type body: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: PlanAddon. The PlanAddon is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.PlanAddon
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def update(
        self,
        plan_id: str,
        plan_addon_id: str,
        body: Union[_models.PlanAddonReplaceUpdate, JSON, IO[bytes]],
        **kwargs: Any
    ) -> _models.PlanAddon:
        """Update add-on assignment for plan.

        Update add-on assignment for plan.

        :param plan_id: Required.
        :type plan_id: str
        :param plan_addon_id: Required.
        :type plan_addon_id: str
        :param body: Is one of the following types: PlanAddonReplaceUpdate, JSON, IO[bytes] Required.
        :type body: ~openmeter._generated.models.PlanAddonReplaceUpdate or JSON or IO[bytes]
        :return: PlanAddon. The PlanAddon is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.PlanAddon
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[_models.PlanAddon] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(body, (IOBase, bytes)):
            _content = body
        else:
            _content = json.dumps(body, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_product_catalog_plan_addons_update_request(
            plan_id=plan_id,
            plan_addon_id=plan_addon_id,
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.PlanAddon, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def get(self, plan_id: str, plan_addon_id: str, **kwargs: Any) -> _models.PlanAddon:
        """Get add-on assignment for plan.

        Get add-on assignment for plan by id.

        :param plan_id: Required.
        :type plan_id: str
        :param plan_addon_id: Required.
        :type plan_addon_id: str
        :return: PlanAddon. The PlanAddon is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.PlanAddon
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.PlanAddon] = kwargs.pop("cls", None)

        _request = build_product_catalog_plan_addons_get_request(
            plan_id=plan_id,
            plan_addon_id=plan_addon_id,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.PlanAddon, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def delete(self, plan_id: str, plan_addon_id: str, **kwargs: Any) -> None:
        """Delete add-on assignment for plan.

        Delete add-on assignment for plan.

        Once a plan is deleted it cannot be undeleted.

        :param plan_id: Required.
        :type plan_id: str
        :param plan_addon_id: Required.
        :type plan_addon_id: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[None] = kwargs.pop("cls", None)

        _request = build_product_catalog_plan_addons_delete_request(
            plan_id=plan_id,
            plan_addon_id=plan_addon_id,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = False
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [204]:
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if cls:
            return cls(pipeline_response, None, {})  # type: ignore


class ProductCatalogAddonsOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`addons` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

    def list(
        self,
        *,
        include_deleted: Optional[bool] = None,
        id: Optional[List[str]] = None,
        key: Optional[List[str]] = None,
        key_version: Optional[dict[str, List[int]]] = None,
        status: Optional[List[Union[str, _models.AddonStatus]]] = None,
        currency: Optional[List[str]] = None,
        page: Optional[int] = None,
        page_size: Optional[int] = None,
        order: Optional[Union[str, _models.SortOrder]] = None,
        order_by: Optional[Union[str, _models.AddonOrderBy]] = None,
        **kwargs: Any
    ) -> AsyncItemPaged["_models.Addon"]:
        """List add-ons.

        List all add-ons.

        :keyword include_deleted: Include deleted add-ons in response.

         Usage: ``?includeDeleted=true``. Default value is None.
        :paramtype include_deleted: bool
        :keyword id: Filter by addon.id attribute. Default value is None.
        :paramtype id: list[str]
        :keyword key: Filter by addon.key attribute. Default value is None.
        :paramtype key: list[str]
        :keyword key_version: Filter by addon.key and addon.version attributes. Default value is None.
        :paramtype key_version: dict[str, list[int]]
        :keyword status: Only return add-ons with the given status.

         Usage:

         * `?status=active`: return only the currently active add-ons
         * `?status=draft`: return only the draft add-ons
         * `?status=archived`: return only the archived add-ons. Default value is None.
        :paramtype status: list[str or ~openmeter.models.AddonStatus]
        :keyword currency: Filter by addon.currency attribute. Default value is None.
        :paramtype currency: list[str]
        :keyword page: Page index.

         Default is 1. Default value is None.
        :paramtype page: int
        :keyword page_size: The maximum number of items per page.

         Default is 100. Default value is None.
        :paramtype page_size: int
        :keyword order: The order direction. Known values are: "ASC" and "DESC". Default value is None.
        :paramtype order: str or ~openmeter.models.SortOrder
        :keyword order_by: The order by field. Known values are: "id", "key", "version", "created_at",
         and "updated_at". Default value is None.
        :paramtype order_by: str or ~openmeter.models.AddonOrderBy
        :return: An iterator like instance of Addon
        :rtype: ~corehttp.paging.AsyncItemPaged[~openmeter._generated.models.Addon]
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[List[_models.Addon]] = kwargs.pop("cls", None)

        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        def prepare_request(next_link=None):
            if not next_link:

                _request = build_product_catalog_addons_list_request(
                    include_deleted=include_deleted,
                    id=id,
                    key=key,
                    key_version=key_version,
                    status=status,
                    currency=currency,
                    page=page,
                    page_size=page_size,
                    order=order,
                    order_by=order_by,
                    headers=_headers,
                    params=_params,
                )
                path_format_arguments = {
                    "endpoint": self._serialize.url(
                        "self._config.endpoint", self._config.endpoint, "str", skip_quote=True
                    ),
                }
                _request.url = self._client.format_url(_request.url, **path_format_arguments)

            else:
                _request = HttpRequest("GET", next_link)
                path_format_arguments = {
                    "endpoint": self._serialize.url(
                        "self._config.endpoint", self._config.endpoint, "str", skip_quote=True
                    ),
                }
                _request.url = self._client.format_url(_request.url, **path_format_arguments)

            return _request

        async def extract_data(pipeline_response):
            deserialized = pipeline_response.http_response.json()
            list_of_elem = _deserialize(List[_models.Addon], deserialized.get("items", []))
            if cls:
                list_of_elem = cls(list_of_elem)  # type: ignore
            return None, AsyncList(list_of_elem)

        async def get_next(next_link=None):
            _request = prepare_request(next_link)

            _stream = False
            pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)
            response = pipeline_response.http_response

            if response.status_code not in [200]:
                map_error(status_code=response.status_code, response=response, error_map=error_map)
                error = None
                if response.status_code == 400:
                    error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
                elif response.status_code == 401:
                    error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                    raise ClientAuthenticationError(response=response, model=error)
                if response.status_code == 403:
                    error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
                elif response.status_code == 500:
                    error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
                elif response.status_code == 503:
                    error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
                elif response.status_code == 412:
                    error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
                else:
                    error = _failsafe_deserialize(
                        _models.UnexpectedProblemResponse,
                        response,
                    )
                raise HttpResponseError(response=response, model=error)

            return pipeline_response

        return AsyncItemPaged(get_next, extract_data)

    @overload
    async def create(
        self, request: _models.AddonCreate, *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.Addon:
        """Create an add-on.

        Create a new add-on.

        :param request: Required.
        :type request: ~openmeter._generated.models.AddonCreate
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Addon. The Addon is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Addon
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def create(self, request: JSON, *, content_type: str = "application/json", **kwargs: Any) -> _models.Addon:
        """Create an add-on.

        Create a new add-on.

        :param request: Required.
        :type request: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Addon. The Addon is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Addon
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def create(
        self, request: IO[bytes], *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.Addon:
        """Create an add-on.

        Create a new add-on.

        :param request: Required.
        :type request: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Addon. The Addon is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Addon
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def create(self, request: Union[_models.AddonCreate, JSON, IO[bytes]], **kwargs: Any) -> _models.Addon:
        """Create an add-on.

        Create a new add-on.

        :param request: Is one of the following types: AddonCreate, JSON, IO[bytes] Required.
        :type request: ~openmeter._generated.models.AddonCreate or JSON or IO[bytes]
        :return: Addon. The Addon is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Addon
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[_models.Addon] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(request, (IOBase, bytes)):
            _content = request
        else:
            _content = json.dumps(request, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_product_catalog_addons_create_request(
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [201]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.Addon, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    @overload
    async def update(
        self,
        addon_id: str,
        request: _models.AddonReplaceUpdate,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> _models.Addon:
        """Update add-on.

        Update add-on by id.

        :param addon_id: Required.
        :type addon_id: str
        :param request: Required.
        :type request: ~openmeter._generated.models.AddonReplaceUpdate
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Addon. The Addon is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Addon
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def update(
        self, addon_id: str, request: JSON, *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.Addon:
        """Update add-on.

        Update add-on by id.

        :param addon_id: Required.
        :type addon_id: str
        :param request: Required.
        :type request: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Addon. The Addon is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Addon
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def update(
        self, addon_id: str, request: IO[bytes], *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.Addon:
        """Update add-on.

        Update add-on by id.

        :param addon_id: Required.
        :type addon_id: str
        :param request: Required.
        :type request: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Addon. The Addon is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Addon
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def update(
        self, addon_id: str, request: Union[_models.AddonReplaceUpdate, JSON, IO[bytes]], **kwargs: Any
    ) -> _models.Addon:
        """Update add-on.

        Update add-on by id.

        :param addon_id: Required.
        :type addon_id: str
        :param request: Is one of the following types: AddonReplaceUpdate, JSON, IO[bytes] Required.
        :type request: ~openmeter._generated.models.AddonReplaceUpdate or JSON or IO[bytes]
        :return: Addon. The Addon is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Addon
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[_models.Addon] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(request, (IOBase, bytes)):
            _content = request
        else:
            _content = json.dumps(request, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_product_catalog_addons_update_request(
            addon_id=addon_id,
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.Addon, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def get(self, addon_id: str, *, include_latest: Optional[bool] = None, **kwargs: Any) -> _models.Addon:
        """Get add-on.

        Get add-on by id or key. The latest published version is returned if latter is used.

        :param addon_id: Required.
        :type addon_id: str
        :keyword include_latest: Include latest version of the add-on instead of the version in active
         state.

         Usage: ``?includeLatest=true``. Default value is None.
        :paramtype include_latest: bool
        :return: Addon. The Addon is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Addon
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.Addon] = kwargs.pop("cls", None)

        _request = build_product_catalog_addons_get_request(
            addon_id=addon_id,
            include_latest=include_latest,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.Addon, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def delete(self, addon_id: str, **kwargs: Any) -> None:
        """Delete add-on.

        Soft delete add-on by id.

        Once a add-on is deleted it cannot be undeleted.

        :param addon_id: Required.
        :type addon_id: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[None] = kwargs.pop("cls", None)

        _request = build_product_catalog_addons_delete_request(
            addon_id=addon_id,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = False
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [204]:
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if cls:
            return cls(pipeline_response, None, {})  # type: ignore

    async def publish(self, addon_id: str, **kwargs: Any) -> _models.Addon:
        """Publish add-on.

        Publish a add-on version.

        :param addon_id: Required.
        :type addon_id: str
        :return: Addon. The Addon is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Addon
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.Addon] = kwargs.pop("cls", None)

        _request = build_product_catalog_addons_publish_request(
            addon_id=addon_id,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.Addon, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def archive(self, addon_id: str, **kwargs: Any) -> _models.Addon:
        """Archive add-on version.

        Archive a add-on version.

        :param addon_id: Required.
        :type addon_id: str
        :return: Addon. The Addon is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Addon
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.Addon] = kwargs.pop("cls", None)

        _request = build_product_catalog_addons_archive_request(
            addon_id=addon_id,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.Addon, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore


class ProductCatalogSubscriptionsOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`subscriptions` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

    async def get_expanded(
        self, subscription_id: str, *, at: Optional[datetime.datetime] = None, **kwargs: Any
    ) -> _models.SubscriptionExpanded:
        """Get subscription.

        get_expanded.

        :param subscription_id: Required.
        :type subscription_id: str
        :keyword at: The time at which the subscription should be queried. If not provided the current
         time is used. Default value is None.
        :paramtype at: ~datetime.datetime
        :return: SubscriptionExpanded. The SubscriptionExpanded is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.SubscriptionExpanded
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.SubscriptionExpanded] = kwargs.pop("cls", None)

        _request = build_product_catalog_subscriptions_get_expanded_request(
            subscription_id=subscription_id,
            at=at,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.SubscriptionExpanded, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    @overload
    async def create(
        self, body: _models.PlanSubscriptionCreate, *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.Subscription:
        """Create subscription.

        create.

        :param body: Required.
        :type body: ~openmeter._generated.models.PlanSubscriptionCreate
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Subscription. The Subscription is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Subscription
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def create(
        self, body: _models.CustomSubscriptionCreate, *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.Subscription:
        """Create subscription.

        create.

        :param body: Required.
        :type body: ~openmeter._generated.models.CustomSubscriptionCreate
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Subscription. The Subscription is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Subscription
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def create(self, body: "_types.SubscriptionCreate", **kwargs: Any) -> _models.Subscription:
        """Create subscription.

        create.

        :param body: Is either a PlanSubscriptionCreate type or a CustomSubscriptionCreate type.
         Required.
        :type body: ~openmeter._generated.models.PlanSubscriptionCreate or
         ~openmeter._generated.models.CustomSubscriptionCreate
        :return: Subscription. The Subscription is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Subscription
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[_models.Subscription] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(body, _models.PlanSubscriptionCreate):
            _content = json.dumps(body, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore
        elif isinstance(body, _models.CustomSubscriptionCreate):
            _content = json.dumps(body, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_product_catalog_subscriptions_create_request(
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [201]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 409:
                error = _failsafe_deserialize(_models.ConflictProblemResponse, response)
                raise ResourceExistsError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.ValidationErrorProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.Subscription, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    @overload
    async def edit(
        self,
        subscription_id: str,
        body: _models.SubscriptionEdit,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> _models.Subscription:
        """Edit subscription.

        Batch processing commands for manipulating running subscriptions.
        The key format is ``/phases/{phaseKey}`` or ``/phases/{phaseKey}/items/{itemKey}``.

        :param subscription_id: Required.
        :type subscription_id: str
        :param body: Required.
        :type body: ~openmeter._generated.models.SubscriptionEdit
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Subscription. The Subscription is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Subscription
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def edit(
        self, subscription_id: str, body: JSON, *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.Subscription:
        """Edit subscription.

        Batch processing commands for manipulating running subscriptions.
        The key format is ``/phases/{phaseKey}`` or ``/phases/{phaseKey}/items/{itemKey}``.

        :param subscription_id: Required.
        :type subscription_id: str
        :param body: Required.
        :type body: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Subscription. The Subscription is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Subscription
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def edit(
        self, subscription_id: str, body: IO[bytes], *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.Subscription:
        """Edit subscription.

        Batch processing commands for manipulating running subscriptions.
        The key format is ``/phases/{phaseKey}`` or ``/phases/{phaseKey}/items/{itemKey}``.

        :param subscription_id: Required.
        :type subscription_id: str
        :param body: Required.
        :type body: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Subscription. The Subscription is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Subscription
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def edit(
        self, subscription_id: str, body: Union[_models.SubscriptionEdit, JSON, IO[bytes]], **kwargs: Any
    ) -> _models.Subscription:
        """Edit subscription.

        Batch processing commands for manipulating running subscriptions.
        The key format is ``/phases/{phaseKey}`` or ``/phases/{phaseKey}/items/{itemKey}``.

        :param subscription_id: Required.
        :type subscription_id: str
        :param body: Is one of the following types: SubscriptionEdit, JSON, IO[bytes] Required.
        :type body: ~openmeter._generated.models.SubscriptionEdit or JSON or IO[bytes]
        :return: Subscription. The Subscription is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Subscription
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[_models.Subscription] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(body, (IOBase, bytes)):
            _content = body
        else:
            _content = json.dumps(body, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_product_catalog_subscriptions_edit_request(
            subscription_id=subscription_id,
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 409:
                error = _failsafe_deserialize(_models.ConflictProblemResponse, response)
                raise ResourceExistsError(response=response, model=error)
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.ValidationErrorProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.Subscription, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    @overload
    async def change(
        self,
        subscription_id: str,
        body: _models.PlanSubscriptionChange,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> _models.SubscriptionChangeResponseBody:
        """Change subscription.

        Closes a running subscription and starts a new one according to the specification.
        Can be used for upgrades, downgrades, and plan changes.

        :param subscription_id: Required.
        :type subscription_id: str
        :param body: Required.
        :type body: ~openmeter._generated.models.PlanSubscriptionChange
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: SubscriptionChangeResponseBody. The SubscriptionChangeResponseBody is compatible with
         MutableMapping
        :rtype: ~openmeter._generated.models.SubscriptionChangeResponseBody
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def change(
        self,
        subscription_id: str,
        body: _models.CustomSubscriptionChange,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> _models.SubscriptionChangeResponseBody:
        """Change subscription.

        Closes a running subscription and starts a new one according to the specification.
        Can be used for upgrades, downgrades, and plan changes.

        :param subscription_id: Required.
        :type subscription_id: str
        :param body: Required.
        :type body: ~openmeter._generated.models.CustomSubscriptionChange
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: SubscriptionChangeResponseBody. The SubscriptionChangeResponseBody is compatible with
         MutableMapping
        :rtype: ~openmeter._generated.models.SubscriptionChangeResponseBody
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def change(
        self, subscription_id: str, body: "_types.SubscriptionChange", **kwargs: Any
    ) -> _models.SubscriptionChangeResponseBody:
        """Change subscription.

        Closes a running subscription and starts a new one according to the specification.
        Can be used for upgrades, downgrades, and plan changes.

        :param subscription_id: Required.
        :type subscription_id: str
        :param body: Is either a PlanSubscriptionChange type or a CustomSubscriptionChange type.
         Required.
        :type body: ~openmeter._generated.models.PlanSubscriptionChange or
         ~openmeter._generated.models.CustomSubscriptionChange
        :return: SubscriptionChangeResponseBody. The SubscriptionChangeResponseBody is compatible with
         MutableMapping
        :rtype: ~openmeter._generated.models.SubscriptionChangeResponseBody
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[_models.SubscriptionChangeResponseBody] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(body, _models.PlanSubscriptionChange):
            _content = json.dumps(body, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore
        elif isinstance(body, _models.CustomSubscriptionChange):
            _content = json.dumps(body, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_product_catalog_subscriptions_change_request(
            subscription_id=subscription_id,
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 409:
                error = _failsafe_deserialize(_models.ConflictProblemResponse, response)
                raise ResourceExistsError(response=response, model=error)
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.ValidationErrorProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.SubscriptionChangeResponseBody, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    @overload
    async def migrate(
        self,
        subscription_id: str,
        body: _models.MigrateRequest,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> _models.SubscriptionChangeResponseBody:
        """Migrate subscription.

        Migrates the subscripiton to the provided version of the current plan.
        If possible, the migration will be done immediately.
        If not, the migration will be scheduled to the end of the current billing period.

        :param subscription_id: Required.
        :type subscription_id: str
        :param body: Required.
        :type body: ~openmeter._generated.models.MigrateRequest
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: SubscriptionChangeResponseBody. The SubscriptionChangeResponseBody is compatible with
         MutableMapping
        :rtype: ~openmeter._generated.models.SubscriptionChangeResponseBody
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def migrate(
        self, subscription_id: str, body: JSON, *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.SubscriptionChangeResponseBody:
        """Migrate subscription.

        Migrates the subscripiton to the provided version of the current plan.
        If possible, the migration will be done immediately.
        If not, the migration will be scheduled to the end of the current billing period.

        :param subscription_id: Required.
        :type subscription_id: str
        :param body: Required.
        :type body: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: SubscriptionChangeResponseBody. The SubscriptionChangeResponseBody is compatible with
         MutableMapping
        :rtype: ~openmeter._generated.models.SubscriptionChangeResponseBody
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def migrate(
        self, subscription_id: str, body: IO[bytes], *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.SubscriptionChangeResponseBody:
        """Migrate subscription.

        Migrates the subscripiton to the provided version of the current plan.
        If possible, the migration will be done immediately.
        If not, the migration will be scheduled to the end of the current billing period.

        :param subscription_id: Required.
        :type subscription_id: str
        :param body: Required.
        :type body: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: SubscriptionChangeResponseBody. The SubscriptionChangeResponseBody is compatible with
         MutableMapping
        :rtype: ~openmeter._generated.models.SubscriptionChangeResponseBody
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def migrate(
        self, subscription_id: str, body: Union[_models.MigrateRequest, JSON, IO[bytes]], **kwargs: Any
    ) -> _models.SubscriptionChangeResponseBody:
        """Migrate subscription.

        Migrates the subscripiton to the provided version of the current plan.
        If possible, the migration will be done immediately.
        If not, the migration will be scheduled to the end of the current billing period.

        :param subscription_id: Required.
        :type subscription_id: str
        :param body: Is one of the following types: MigrateRequest, JSON, IO[bytes] Required.
        :type body: ~openmeter._generated.models.MigrateRequest or JSON or IO[bytes]
        :return: SubscriptionChangeResponseBody. The SubscriptionChangeResponseBody is compatible with
         MutableMapping
        :rtype: ~openmeter._generated.models.SubscriptionChangeResponseBody
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[_models.SubscriptionChangeResponseBody] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(body, (IOBase, bytes)):
            _content = body
        else:
            _content = json.dumps(body, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_product_catalog_subscriptions_migrate_request(
            subscription_id=subscription_id,
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 409:
                error = _failsafe_deserialize(_models.ConflictProblemResponse, response)
                raise ResourceExistsError(response=response, model=error)
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.ValidationErrorProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.SubscriptionChangeResponseBody, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def restore(self, subscription_id: str, **kwargs: Any) -> _models.Subscription:
        """Restore subscription.

        Restores a canceled subscription.
        Any subscription scheduled to start later will be deleted and this subscription will be
        continued indefinitely.

        :param subscription_id: Required.
        :type subscription_id: str
        :return: Subscription. The Subscription is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Subscription
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.Subscription] = kwargs.pop("cls", None)

        _request = build_product_catalog_subscriptions_restore_request(
            subscription_id=subscription_id,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.Subscription, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    @overload
    async def cancel(
        self,
        subscription_id: str,
        body: _models.CancelRequest,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> _models.Subscription:
        """Cancel subscription.

        Cancels the subscription.
        Will result in a scheduling conflict if there are other subscriptions scheduled to start after
        the cancellation time.

        :param subscription_id: Required.
        :type subscription_id: str
        :param body: Required.
        :type body: ~openmeter._generated.models.CancelRequest
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Subscription. The Subscription is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Subscription
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def cancel(
        self, subscription_id: str, body: JSON, *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.Subscription:
        """Cancel subscription.

        Cancels the subscription.
        Will result in a scheduling conflict if there are other subscriptions scheduled to start after
        the cancellation time.

        :param subscription_id: Required.
        :type subscription_id: str
        :param body: Required.
        :type body: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Subscription. The Subscription is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Subscription
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def cancel(
        self, subscription_id: str, body: IO[bytes], *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.Subscription:
        """Cancel subscription.

        Cancels the subscription.
        Will result in a scheduling conflict if there are other subscriptions scheduled to start after
        the cancellation time.

        :param subscription_id: Required.
        :type subscription_id: str
        :param body: Required.
        :type body: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Subscription. The Subscription is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Subscription
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def cancel(
        self, subscription_id: str, body: Union[_models.CancelRequest, JSON, IO[bytes]], **kwargs: Any
    ) -> _models.Subscription:
        """Cancel subscription.

        Cancels the subscription.
        Will result in a scheduling conflict if there are other subscriptions scheduled to start after
        the cancellation time.

        :param subscription_id: Required.
        :type subscription_id: str
        :param body: Is one of the following types: CancelRequest, JSON, IO[bytes] Required.
        :type body: ~openmeter._generated.models.CancelRequest or JSON or IO[bytes]
        :return: Subscription. The Subscription is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Subscription
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[_models.Subscription] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(body, (IOBase, bytes)):
            _content = body
        else:
            _content = json.dumps(body, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_product_catalog_subscriptions_cancel_request(
            subscription_id=subscription_id,
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 409:
                error = _failsafe_deserialize(_models.ConflictProblemResponse, response)
                raise ResourceExistsError(response=response, model=error)
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.ValidationErrorProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.Subscription, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def unschedule_cancelation(self, subscription_id: str, **kwargs: Any) -> _models.Subscription:
        """Unschedule cancelation.

        Cancels the scheduled cancelation.

        :param subscription_id: Required.
        :type subscription_id: str
        :return: Subscription. The Subscription is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Subscription
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.Subscription] = kwargs.pop("cls", None)

        _request = build_product_catalog_subscriptions_unschedule_cancelation_request(
            subscription_id=subscription_id,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 409:
                error = _failsafe_deserialize(_models.ConflictProblemResponse, response)
                raise ResourceExistsError(response=response, model=error)
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.ValidationErrorProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.Subscription, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def delete(self, subscription_id: str, **kwargs: Any) -> None:
        """Delete subscription.

        Deletes a subscription. Only scheduled subscriptions can be deleted.

        :param subscription_id: Required.
        :type subscription_id: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[None] = kwargs.pop("cls", None)

        _request = build_product_catalog_subscriptions_delete_request(
            subscription_id=subscription_id,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = False
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [204]:
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.ValidationErrorProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if cls:
            return cls(pipeline_response, None, {})  # type: ignore


class ProductCatalogSubscriptionAddonsOperations:  # pylint: disable=name-too-long
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`subscription_addons` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

    @overload
    async def create(
        self,
        subscription_id: str,
        request: _models.SubscriptionAddonCreate,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> _models.SubscriptionAddon:
        """Create subscription addon.

        Create a new subscription addon, either providing the key or the id of the addon.

        :param subscription_id: Required.
        :type subscription_id: str
        :param request: Required.
        :type request: ~openmeter._generated.models.SubscriptionAddonCreate
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: SubscriptionAddon. The SubscriptionAddon is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.SubscriptionAddon
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def create(
        self, subscription_id: str, request: JSON, *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.SubscriptionAddon:
        """Create subscription addon.

        Create a new subscription addon, either providing the key or the id of the addon.

        :param subscription_id: Required.
        :type subscription_id: str
        :param request: Required.
        :type request: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: SubscriptionAddon. The SubscriptionAddon is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.SubscriptionAddon
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def create(
        self, subscription_id: str, request: IO[bytes], *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.SubscriptionAddon:
        """Create subscription addon.

        Create a new subscription addon, either providing the key or the id of the addon.

        :param subscription_id: Required.
        :type subscription_id: str
        :param request: Required.
        :type request: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: SubscriptionAddon. The SubscriptionAddon is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.SubscriptionAddon
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def create(
        self, subscription_id: str, request: Union[_models.SubscriptionAddonCreate, JSON, IO[bytes]], **kwargs: Any
    ) -> _models.SubscriptionAddon:
        """Create subscription addon.

        Create a new subscription addon, either providing the key or the id of the addon.

        :param subscription_id: Required.
        :type subscription_id: str
        :param request: Is one of the following types: SubscriptionAddonCreate, JSON, IO[bytes]
         Required.
        :type request: ~openmeter._generated.models.SubscriptionAddonCreate or JSON or IO[bytes]
        :return: SubscriptionAddon. The SubscriptionAddon is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.SubscriptionAddon
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[_models.SubscriptionAddon] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(request, (IOBase, bytes)):
            _content = request
        else:
            _content = json.dumps(request, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_product_catalog_subscription_addons_create_request(
            subscription_id=subscription_id,
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [201]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 409:
                error = _failsafe_deserialize(_models.ConflictProblemResponse, response)
                raise ResourceExistsError(response=response, model=error)
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.SubscriptionAddon, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def list(self, subscription_id: str, **kwargs: Any) -> List[_models.SubscriptionAddon]:
        """List subscription addons.

        List all addons of a subscription. In the returned list will match to a set unique by addonId.

        :param subscription_id: Required.
        :type subscription_id: str
        :return: list of SubscriptionAddon
        :rtype: list[~openmeter._generated.models.SubscriptionAddon]
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[List[_models.SubscriptionAddon]] = kwargs.pop("cls", None)

        _request = build_product_catalog_subscription_addons_list_request(
            subscription_id=subscription_id,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(List[_models.SubscriptionAddon], response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def get(self, subscription_id: str, subscription_addon_id: str, **kwargs: Any) -> _models.SubscriptionAddon:
        """Get subscription addon.

        Get a subscription addon by id.

        :param subscription_id: Required.
        :type subscription_id: str
        :param subscription_addon_id: Required.
        :type subscription_addon_id: str
        :return: SubscriptionAddon. The SubscriptionAddon is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.SubscriptionAddon
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.SubscriptionAddon] = kwargs.pop("cls", None)

        _request = build_product_catalog_subscription_addons_get_request(
            subscription_id=subscription_id,
            subscription_addon_id=subscription_addon_id,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.SubscriptionAddon, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    @overload
    async def update(
        self,
        subscription_id: str,
        subscription_addon_id: str,
        body: _models.SubscriptionAddonUpdate,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> _models.SubscriptionAddon:
        """Update subscription addon.

        Updates a subscription addon (allows changing the quantity: purchasing more instances or
        cancelling the current instances).

        :param subscription_id: Required.
        :type subscription_id: str
        :param subscription_addon_id: Required.
        :type subscription_addon_id: str
        :param body: Required.
        :type body: ~openmeter._generated.models.SubscriptionAddonUpdate
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: SubscriptionAddon. The SubscriptionAddon is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.SubscriptionAddon
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def update(
        self,
        subscription_id: str,
        subscription_addon_id: str,
        body: JSON,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> _models.SubscriptionAddon:
        """Update subscription addon.

        Updates a subscription addon (allows changing the quantity: purchasing more instances or
        cancelling the current instances).

        :param subscription_id: Required.
        :type subscription_id: str
        :param subscription_addon_id: Required.
        :type subscription_addon_id: str
        :param body: Required.
        :type body: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: SubscriptionAddon. The SubscriptionAddon is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.SubscriptionAddon
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def update(
        self,
        subscription_id: str,
        subscription_addon_id: str,
        body: IO[bytes],
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> _models.SubscriptionAddon:
        """Update subscription addon.

        Updates a subscription addon (allows changing the quantity: purchasing more instances or
        cancelling the current instances).

        :param subscription_id: Required.
        :type subscription_id: str
        :param subscription_addon_id: Required.
        :type subscription_addon_id: str
        :param body: Required.
        :type body: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: SubscriptionAddon. The SubscriptionAddon is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.SubscriptionAddon
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def update(
        self,
        subscription_id: str,
        subscription_addon_id: str,
        body: Union[_models.SubscriptionAddonUpdate, JSON, IO[bytes]],
        **kwargs: Any
    ) -> _models.SubscriptionAddon:
        """Update subscription addon.

        Updates a subscription addon (allows changing the quantity: purchasing more instances or
        cancelling the current instances).

        :param subscription_id: Required.
        :type subscription_id: str
        :param subscription_addon_id: Required.
        :type subscription_addon_id: str
        :param body: Is one of the following types: SubscriptionAddonUpdate, JSON, IO[bytes] Required.
        :type body: ~openmeter._generated.models.SubscriptionAddonUpdate or JSON or IO[bytes]
        :return: SubscriptionAddon. The SubscriptionAddon is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.SubscriptionAddon
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[_models.SubscriptionAddon] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(body, (IOBase, bytes)):
            _content = body
        else:
            _content = json.dumps(body, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_product_catalog_subscription_addons_update_request(
            subscription_id=subscription_id,
            subscription_addon_id=subscription_addon_id,
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.SubscriptionAddon, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore


class EntitlementsV2Operations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`v2` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

        self.entitlements = EntitlementsV2EntitlementsOperations(
            self._client, self._config, self._serialize, self._deserialize
        )
        self.customer_entitlements = EntitlementsV2CustomerEntitlementsOperations(
            self._client, self._config, self._serialize, self._deserialize
        )
        self.customer_entitlement = EntitlementsV2CustomerEntitlementOperations(
            self._client, self._config, self._serialize, self._deserialize
        )
        self.grants = EntitlementsV2GrantsOperations(self._client, self._config, self._serialize, self._deserialize)


class EntitlementsEntitlementsOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`entitlements` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

    async def list(
        self,
        *,
        feature: Optional[List[str]] = None,
        subject: Optional[List[str]] = None,
        entitlement_type: Optional[List[Union[str, _models.EntitlementType]]] = None,
        exclude_inactive: Optional[bool] = None,
        page: Optional[int] = None,
        page_size: Optional[int] = None,
        offset: Optional[int] = None,
        limit: Optional[int] = None,
        order: Optional[Union[str, _models.SortOrder]] = None,
        order_by: Optional[Union[str, _models.EntitlementOrderBy]] = None,
        **kwargs: Any
    ) -> "_types.ListEntitlementsResult":
        """List all entitlements.

        List all entitlements for all the subjects and features. This endpoint is intended for
        administrative purposes only.
        To fetch the entitlements of a specific subject please use the
        /api/v1/subjects/{subjectKeyOrID}/entitlements endpoint.
        If page is provided that takes precedence and the paginated response is returned.

        :keyword feature: Filtering by multiple features.

         Usage: ``?feature=feature-1&feature=feature-2``. Default value is None.
        :paramtype feature: list[str]
        :keyword subject: Filtering by multiple subjects.

         Usage: ``?subject=customer-1&subject=customer-2``. Default value is None.
        :paramtype subject: list[str]
        :keyword entitlement_type: Filtering by multiple entitlement types.

         Usage: ``?entitlementType=metered&entitlementType=boolean``. Default value is None.
        :paramtype entitlement_type: list[str or ~openmeter.models.EntitlementType]
        :keyword exclude_inactive: Exclude inactive entitlements in the response (those scheduled for
         later or earlier). Default value is None.
        :paramtype exclude_inactive: bool
        :keyword page: Page index.

         Default is 1. Default value is None.
        :paramtype page: int
        :keyword page_size: The maximum number of items per page.

         Default is 100. Default value is None.
        :paramtype page_size: int
        :keyword offset: Number of items to skip.

         Default is 0. Default value is None.
        :paramtype offset: int
        :keyword limit: Number of items to return.

         Default is 100. Default value is None.
        :paramtype limit: int
        :keyword order: The order direction. Known values are: "ASC" and "DESC". Default value is None.
        :paramtype order: str or ~openmeter.models.SortOrder
        :keyword order_by: The order by field. Known values are: "createdAt" and "updatedAt". Default
         value is None.
        :paramtype order_by: str or ~openmeter.models.EntitlementOrderBy
        :return: list of EntitlementMetered or EntitlementStatic or EntitlementBoolean or
         EntitlementPaginatedResponse
        :rtype: list[~openmeter._generated.models.EntitlementMetered or
         ~openmeter._generated.models.EntitlementStatic or
         ~openmeter._generated.models.EntitlementBoolean] or
         ~openmeter._generated.models.EntitlementPaginatedResponse
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType["_types.ListEntitlementsResult"] = kwargs.pop("cls", None)

        _request = build_entitlements_entitlements_list_request(
            feature=feature,
            subject=subject,
            entitlement_type=entitlement_type,
            exclude_inactive=exclude_inactive,
            page=page,
            page_size=page_size,
            offset=offset,
            limit=limit,
            order=order,
            order_by=order_by,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize("_types.ListEntitlementsResult", response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def get(self, entitlement_id: str, **kwargs: Any) -> "_types.Entitlement":
        """Get entitlement by id.

        Get entitlement by id.

        :param entitlement_id: Required.
        :type entitlement_id: str
        :return: EntitlementMetered or EntitlementStatic or EntitlementBoolean
        :rtype: ~openmeter._generated.models.EntitlementMetered or
         ~openmeter._generated.models.EntitlementStatic or
         ~openmeter._generated.models.EntitlementBoolean
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType["_types.Entitlement"] = kwargs.pop("cls", None)

        _request = build_entitlements_entitlements_get_request(
            entitlement_id=entitlement_id,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize("_types.Entitlement", response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore


class EntitlementsGrantsOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`grants` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

    async def list(
        self,
        *,
        feature: Optional[List[str]] = None,
        subject: Optional[List[str]] = None,
        include_deleted: Optional[bool] = None,
        page: Optional[int] = None,
        page_size: Optional[int] = None,
        offset: Optional[int] = None,
        limit: Optional[int] = None,
        order: Optional[Union[str, _models.SortOrder]] = None,
        order_by: Optional[Union[str, _models.GrantOrderBy]] = None,
        **kwargs: Any
    ) -> Union[List[_models.EntitlementGrant], _models.GrantPaginatedResponse]:
        """List grants.

        List all grants for all the subjects and entitlements. This endpoint is intended for
        administrative purposes only.
        To fetch the grants of a specific entitlement please use the
        /api/v1/subjects/{subjectKeyOrID}/entitlements/{entitlementOrFeatureID}/grants endpoint.
        If page is provided that takes precedence and the paginated response is returned.

        :keyword feature: Filtering by multiple features.

         Usage: ``?feature=feature-1&feature=feature-2``. Default value is None.
        :paramtype feature: list[str]
        :keyword subject: Filtering by multiple subjects.

         Usage: ``?subject=customer-1&subject=customer-2``. Default value is None.
        :paramtype subject: list[str]
        :keyword include_deleted: Include deleted. Default value is None.
        :paramtype include_deleted: bool
        :keyword page: Page index.

         Default is 1. Default value is None.
        :paramtype page: int
        :keyword page_size: The maximum number of items per page.

         Default is 100. Default value is None.
        :paramtype page_size: int
        :keyword offset: Number of items to skip.

         Default is 0. Default value is None.
        :paramtype offset: int
        :keyword limit: Number of items to return.

         Default is 100. Default value is None.
        :paramtype limit: int
        :keyword order: The order direction. Known values are: "ASC" and "DESC". Default value is None.
        :paramtype order: str or ~openmeter.models.SortOrder
        :keyword order_by: The order by field. Known values are: "id", "createdAt", and "updatedAt".
         Default value is None.
        :paramtype order_by: str or ~openmeter.models.GrantOrderBy
        :return: list of EntitlementGrant or GrantPaginatedResponse
        :rtype: list[~openmeter._generated.models.EntitlementGrant] or
         ~openmeter._generated.models.GrantPaginatedResponse
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[Union[List[_models.EntitlementGrant], _models.GrantPaginatedResponse]] = kwargs.pop("cls", None)

        _request = build_entitlements_grants_list_request(
            feature=feature,
            subject=subject,
            include_deleted=include_deleted,
            page=page,
            page_size=page_size,
            offset=offset,
            limit=limit,
            order=order,
            order_by=order_by,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(
                Union[List[_models.EntitlementGrant], _models.GrantPaginatedResponse], response.json()
            )

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def delete(self, grant_id: str, **kwargs: Any) -> None:
        """Void grant.

        Voiding a grant means it is no longer valid, it doesn't take part in further balance
        calculations. Voiding a grant does not retroactively take effect, meaning any usage that has
        already been attributed to the grant will remain, but future usage cannot be burnt down from
        the grant.
        For example, if you have a single grant for your metered entitlement with an initial amount of
        100, and so far 60 usage has been metered, the grant (and the entitlement itself) would have a
        balance of 40. If you then void that grant, balance becomes 0, but the 60 previous usage will
        not be affected.

        :param grant_id: Required.
        :type grant_id: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[None] = kwargs.pop("cls", None)

        _request = build_entitlements_grants_delete_request(
            grant_id=grant_id,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = False
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [204]:
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            elif response.status_code == 409:
                error = _failsafe_deserialize(_models.ConflictProblemResponse, response)
                raise ResourceExistsError(response=response, model=error)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if cls:
            return cls(pipeline_response, None, {})  # type: ignore


class EntitlementsSubjectsOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`subjects` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

    @overload
    async def post(
        self,
        subject_id_or_key: str,
        entitlement: _models.EntitlementMeteredCreateInputs,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> "_types.Entitlement":
        """Create a subject entitlement.

        OpenMeter has three types of entitlements: metered, boolean, and static. The type property
        determines the type of entitlement. The underlying feature has to be compatible with the
        entitlement type specified in the request (e.g., a metered entitlement needs a feature
        associated with a meter).



        * Boolean entitlements define static feature access, e.g. "Can use SSO authentication".
        * Static entitlements let you pass along a configuration while granting access, e.g. "Using
        this feature with X Y settings" (passed in the config).
        * Metered entitlements have many use cases, from setting up usage-based access to implementing
        complex credit systems.  Example: The customer can use 10000 AI tokens during the usage period
        of the entitlement.

        A given subject can only have one active (non-deleted) entitlement per featureKey. If you try
        to create a new entitlement for a featureKey that already has an active entitlement, the
        request will fail with a 409 error.

        Once an entitlement is created you cannot modify it, only delete it.

        :param subject_id_or_key: Required.
        :type subject_id_or_key: str
        :param entitlement: Required.
        :type entitlement: ~openmeter._generated.models.EntitlementMeteredCreateInputs
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: EntitlementMetered or EntitlementStatic or EntitlementBoolean
        :rtype: ~openmeter._generated.models.EntitlementMetered or
         ~openmeter._generated.models.EntitlementStatic or
         ~openmeter._generated.models.EntitlementBoolean
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def post(
        self,
        subject_id_or_key: str,
        entitlement: _models.EntitlementStaticCreateInputs,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> "_types.Entitlement":
        """Create a subject entitlement.

        OpenMeter has three types of entitlements: metered, boolean, and static. The type property
        determines the type of entitlement. The underlying feature has to be compatible with the
        entitlement type specified in the request (e.g., a metered entitlement needs a feature
        associated with a meter).



        * Boolean entitlements define static feature access, e.g. "Can use SSO authentication".
        * Static entitlements let you pass along a configuration while granting access, e.g. "Using
        this feature with X Y settings" (passed in the config).
        * Metered entitlements have many use cases, from setting up usage-based access to implementing
        complex credit systems.  Example: The customer can use 10000 AI tokens during the usage period
        of the entitlement.

        A given subject can only have one active (non-deleted) entitlement per featureKey. If you try
        to create a new entitlement for a featureKey that already has an active entitlement, the
        request will fail with a 409 error.

        Once an entitlement is created you cannot modify it, only delete it.

        :param subject_id_or_key: Required.
        :type subject_id_or_key: str
        :param entitlement: Required.
        :type entitlement: ~openmeter._generated.models.EntitlementStaticCreateInputs
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: EntitlementMetered or EntitlementStatic or EntitlementBoolean
        :rtype: ~openmeter._generated.models.EntitlementMetered or
         ~openmeter._generated.models.EntitlementStatic or
         ~openmeter._generated.models.EntitlementBoolean
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def post(
        self,
        subject_id_or_key: str,
        entitlement: _models.EntitlementBooleanCreateInputs,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> "_types.Entitlement":
        """Create a subject entitlement.

        OpenMeter has three types of entitlements: metered, boolean, and static. The type property
        determines the type of entitlement. The underlying feature has to be compatible with the
        entitlement type specified in the request (e.g., a metered entitlement needs a feature
        associated with a meter).



        * Boolean entitlements define static feature access, e.g. "Can use SSO authentication".
        * Static entitlements let you pass along a configuration while granting access, e.g. "Using
        this feature with X Y settings" (passed in the config).
        * Metered entitlements have many use cases, from setting up usage-based access to implementing
        complex credit systems.  Example: The customer can use 10000 AI tokens during the usage period
        of the entitlement.

        A given subject can only have one active (non-deleted) entitlement per featureKey. If you try
        to create a new entitlement for a featureKey that already has an active entitlement, the
        request will fail with a 409 error.

        Once an entitlement is created you cannot modify it, only delete it.

        :param subject_id_or_key: Required.
        :type subject_id_or_key: str
        :param entitlement: Required.
        :type entitlement: ~openmeter._generated.models.EntitlementBooleanCreateInputs
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: EntitlementMetered or EntitlementStatic or EntitlementBoolean
        :rtype: ~openmeter._generated.models.EntitlementMetered or
         ~openmeter._generated.models.EntitlementStatic or
         ~openmeter._generated.models.EntitlementBoolean
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def post(
        self, subject_id_or_key: str, entitlement: "_types.EntitlementCreateInputs", **kwargs: Any
    ) -> "_types.Entitlement":
        """Create a subject entitlement.

        OpenMeter has three types of entitlements: metered, boolean, and static. The type property
        determines the type of entitlement. The underlying feature has to be compatible with the
        entitlement type specified in the request (e.g., a metered entitlement needs a feature
        associated with a meter).



        * Boolean entitlements define static feature access, e.g. "Can use SSO authentication".
        * Static entitlements let you pass along a configuration while granting access, e.g. "Using
        this feature with X Y settings" (passed in the config).
        * Metered entitlements have many use cases, from setting up usage-based access to implementing
        complex credit systems.  Example: The customer can use 10000 AI tokens during the usage period
        of the entitlement.

        A given subject can only have one active (non-deleted) entitlement per featureKey. If you try
        to create a new entitlement for a featureKey that already has an active entitlement, the
        request will fail with a 409 error.

        Once an entitlement is created you cannot modify it, only delete it.

        :param subject_id_or_key: Required.
        :type subject_id_or_key: str
        :param entitlement: Is one of the following types: EntitlementMeteredCreateInputs,
         EntitlementStaticCreateInputs, EntitlementBooleanCreateInputs Required.
        :type entitlement: ~openmeter._generated.models.EntitlementMeteredCreateInputs or
         ~openmeter._generated.models.EntitlementStaticCreateInputs or
         ~openmeter._generated.models.EntitlementBooleanCreateInputs
        :return: EntitlementMetered or EntitlementStatic or EntitlementBoolean
        :rtype: ~openmeter._generated.models.EntitlementMetered or
         ~openmeter._generated.models.EntitlementStatic or
         ~openmeter._generated.models.EntitlementBoolean
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType["_types.Entitlement"] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(entitlement, _models.EntitlementMeteredCreateInputs):
            _content = json.dumps(entitlement, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore
        elif isinstance(entitlement, _models.EntitlementStaticCreateInputs):
            _content = json.dumps(entitlement, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore
        elif isinstance(entitlement, _models.EntitlementBooleanCreateInputs):
            _content = json.dumps(entitlement, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_entitlements_subjects_post_request(
            subject_id_or_key=subject_id_or_key,
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [201]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            elif response.status_code == 409:
                error = _failsafe_deserialize(_models.ConflictProblemResponse, response)
                raise ResourceExistsError(response=response, model=error)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize("_types.Entitlement", response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def list(
        self, subject_id_or_key: str, *, include_deleted: Optional[bool] = None, **kwargs: Any
    ) -> List["_types.Entitlement"]:
        """List subject entitlements.

        List all entitlements for a subject. For checking entitlement access, use the /value endpoint
        instead.

        :param subject_id_or_key: Required.
        :type subject_id_or_key: str
        :keyword include_deleted: Default value is None.
        :paramtype include_deleted: bool
        :return: list of EntitlementMetered or EntitlementStatic or EntitlementBoolean
        :rtype: list[~openmeter._generated.models.EntitlementMetered or
         ~openmeter._generated.models.EntitlementStatic or
         ~openmeter._generated.models.EntitlementBoolean]
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[List["_types.Entitlement"]] = kwargs.pop("cls", None)

        _request = build_entitlements_subjects_list_request(
            subject_id_or_key=subject_id_or_key,
            include_deleted=include_deleted,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(List["_types.Entitlement"], response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def get(self, subject_id_or_key: str, entitlement_id: str, **kwargs: Any) -> "_types.Entitlement":
        """Get subject entitlement.

        Get entitlement by id. For checking entitlement access, use the /value endpoint instead.

        :param subject_id_or_key: Required.
        :type subject_id_or_key: str
        :param entitlement_id: Required.
        :type entitlement_id: str
        :return: EntitlementMetered or EntitlementStatic or EntitlementBoolean
        :rtype: ~openmeter._generated.models.EntitlementMetered or
         ~openmeter._generated.models.EntitlementStatic or
         ~openmeter._generated.models.EntitlementBoolean
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType["_types.Entitlement"] = kwargs.pop("cls", None)

        _request = build_entitlements_subjects_get_request(
            subject_id_or_key=subject_id_or_key,
            entitlement_id=entitlement_id,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            elif response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize("_types.Entitlement", response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def delete(self, subject_id_or_key: str, entitlement_id: str, **kwargs: Any) -> None:
        """Delete subject entitlement.

        Deleting an entitlement revokes access to the associated feature. As a single subject can only
        have one entitlement per featureKey, when "migrating" features you have to delete the old
        entitlements as well.
        As access and status checks can be historical queries, deleting an entitlement populates the
        deletedAt timestamp. When queried for a time before that, the entitlement is still considered
        active, you cannot have retroactive changes to access, which is important for, among other
        things, auditing.

        :param subject_id_or_key: Required.
        :type subject_id_or_key: str
        :param entitlement_id: Required.
        :type entitlement_id: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[None] = kwargs.pop("cls", None)

        _request = build_entitlements_subjects_delete_request(
            subject_id_or_key=subject_id_or_key,
            entitlement_id=entitlement_id,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = False
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [204]:
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            elif response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if cls:
            return cls(pipeline_response, None, {})  # type: ignore

    @overload
    async def override(
        self,
        subject_id_or_key: str,
        entitlement_id_or_feature_key: str,
        entitlement: _models.EntitlementMeteredCreateInputs,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> "_types.Entitlement":
        """Override subject entitlement.

        Overriding an entitlement creates a new entitlement from the provided inputs and soft deletes
        the previous entitlement for the provided subject-feature pair. If the previous entitlement is
        already deleted or otherwise doesnt exist, the override will fail.

        This endpoint is useful for upgrades, downgrades, or other changes to entitlements that require
        a new entitlement to be created with zero downtime.

        :param subject_id_or_key: Required.
        :type subject_id_or_key: str
        :param entitlement_id_or_feature_key: Required.
        :type entitlement_id_or_feature_key: str
        :param entitlement: Required.
        :type entitlement: ~openmeter._generated.models.EntitlementMeteredCreateInputs
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: EntitlementMetered or EntitlementStatic or EntitlementBoolean
        :rtype: ~openmeter._generated.models.EntitlementMetered or
         ~openmeter._generated.models.EntitlementStatic or
         ~openmeter._generated.models.EntitlementBoolean
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def override(
        self,
        subject_id_or_key: str,
        entitlement_id_or_feature_key: str,
        entitlement: _models.EntitlementStaticCreateInputs,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> "_types.Entitlement":
        """Override subject entitlement.

        Overriding an entitlement creates a new entitlement from the provided inputs and soft deletes
        the previous entitlement for the provided subject-feature pair. If the previous entitlement is
        already deleted or otherwise doesnt exist, the override will fail.

        This endpoint is useful for upgrades, downgrades, or other changes to entitlements that require
        a new entitlement to be created with zero downtime.

        :param subject_id_or_key: Required.
        :type subject_id_or_key: str
        :param entitlement_id_or_feature_key: Required.
        :type entitlement_id_or_feature_key: str
        :param entitlement: Required.
        :type entitlement: ~openmeter._generated.models.EntitlementStaticCreateInputs
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: EntitlementMetered or EntitlementStatic or EntitlementBoolean
        :rtype: ~openmeter._generated.models.EntitlementMetered or
         ~openmeter._generated.models.EntitlementStatic or
         ~openmeter._generated.models.EntitlementBoolean
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def override(
        self,
        subject_id_or_key: str,
        entitlement_id_or_feature_key: str,
        entitlement: _models.EntitlementBooleanCreateInputs,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> "_types.Entitlement":
        """Override subject entitlement.

        Overriding an entitlement creates a new entitlement from the provided inputs and soft deletes
        the previous entitlement for the provided subject-feature pair. If the previous entitlement is
        already deleted or otherwise doesnt exist, the override will fail.

        This endpoint is useful for upgrades, downgrades, or other changes to entitlements that require
        a new entitlement to be created with zero downtime.

        :param subject_id_or_key: Required.
        :type subject_id_or_key: str
        :param entitlement_id_or_feature_key: Required.
        :type entitlement_id_or_feature_key: str
        :param entitlement: Required.
        :type entitlement: ~openmeter._generated.models.EntitlementBooleanCreateInputs
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: EntitlementMetered or EntitlementStatic or EntitlementBoolean
        :rtype: ~openmeter._generated.models.EntitlementMetered or
         ~openmeter._generated.models.EntitlementStatic or
         ~openmeter._generated.models.EntitlementBoolean
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def override(
        self,
        subject_id_or_key: str,
        entitlement_id_or_feature_key: str,
        entitlement: "_types.EntitlementCreateInputs",
        **kwargs: Any
    ) -> "_types.Entitlement":
        """Override subject entitlement.

        Overriding an entitlement creates a new entitlement from the provided inputs and soft deletes
        the previous entitlement for the provided subject-feature pair. If the previous entitlement is
        already deleted or otherwise doesnt exist, the override will fail.

        This endpoint is useful for upgrades, downgrades, or other changes to entitlements that require
        a new entitlement to be created with zero downtime.

        :param subject_id_or_key: Required.
        :type subject_id_or_key: str
        :param entitlement_id_or_feature_key: Required.
        :type entitlement_id_or_feature_key: str
        :param entitlement: Is one of the following types: EntitlementMeteredCreateInputs,
         EntitlementStaticCreateInputs, EntitlementBooleanCreateInputs Required.
        :type entitlement: ~openmeter._generated.models.EntitlementMeteredCreateInputs or
         ~openmeter._generated.models.EntitlementStaticCreateInputs or
         ~openmeter._generated.models.EntitlementBooleanCreateInputs
        :return: EntitlementMetered or EntitlementStatic or EntitlementBoolean
        :rtype: ~openmeter._generated.models.EntitlementMetered or
         ~openmeter._generated.models.EntitlementStatic or
         ~openmeter._generated.models.EntitlementBoolean
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType["_types.Entitlement"] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(entitlement, _models.EntitlementMeteredCreateInputs):
            _content = json.dumps(entitlement, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore
        elif isinstance(entitlement, _models.EntitlementStaticCreateInputs):
            _content = json.dumps(entitlement, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore
        elif isinstance(entitlement, _models.EntitlementBooleanCreateInputs):
            _content = json.dumps(entitlement, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_entitlements_subjects_override_request(
            subject_id_or_key=subject_id_or_key,
            entitlement_id_or_feature_key=entitlement_id_or_feature_key,
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [201]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            elif response.status_code == 409:
                error = _failsafe_deserialize(_models.ConflictProblemResponse, response)
                raise ResourceExistsError(response=response, model=error)
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize("_types.Entitlement", response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def get_grants(
        self,
        subject_id_or_key: str,
        entitlement_id_or_feature_key: str,
        *,
        include_deleted: Optional[bool] = None,
        order_by: Optional[Union[str, _models.GrantOrderBy]] = None,
        **kwargs: Any
    ) -> List[_models.EntitlementGrant]:
        """List subject entitlement grants.

        List all grants issued for an entitlement. The entitlement can be defined either by its id or
        featureKey.

        :param subject_id_or_key: Required.
        :type subject_id_or_key: str
        :param entitlement_id_or_feature_key: Required.
        :type entitlement_id_or_feature_key: str
        :keyword include_deleted: Default value is None.
        :paramtype include_deleted: bool
        :keyword order_by: Known values are: "id", "createdAt", and "updatedAt". Default value is None.
        :paramtype order_by: str or ~openmeter.models.GrantOrderBy
        :return: list of EntitlementGrant
        :rtype: list[~openmeter._generated.models.EntitlementGrant]
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[List[_models.EntitlementGrant]] = kwargs.pop("cls", None)

        _request = build_entitlements_subjects_get_grants_request(
            subject_id_or_key=subject_id_or_key,
            entitlement_id_or_feature_key=entitlement_id_or_feature_key,
            include_deleted=include_deleted,
            order_by=order_by,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(List[_models.EntitlementGrant], response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    @overload
    async def create_grant(
        self,
        subject_id_or_key: str,
        entitlement_id_or_feature_key: str,
        grant: _models.EntitlementGrantCreateInput,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> _models.EntitlementGrant:
        """Create subject entitlement grant.

        Grants define a behavior of granting usage for a metered entitlement. They can have complicated
        recurrence and rollover rules, thanks to which you can define a wide range of access patterns
        with a single grant, in most cases you don't have to periodically create new grants. You can
        only issue grants for active metered entitlements.

        A grant defines a given amount of usage that can be consumed for the entitlement. The grant is
        in effect between its effective date and its expiration date. Specifying both is mandatory for
        new grants.

        Grants have a priority setting that determines their order of use. Lower numbers have higher
        priority, with 0 being the highest priority.

        Grants can have a recurrence setting intended to automate the manual reissuing of grants. For
        example, a daily recurrence is equal to reissuing that same grant every day (ignoring rollover
        settings).

        Rollover settings define what happens to the remaining balance of a grant at a reset.
        Balance_After_Reset = MIN(MaxRolloverAmount, MAX(Balance_Before_Reset, MinRolloverAmount))

        Grants cannot be changed once created, only deleted. This is to ensure that balance is
        deterministic regardless of when it is queried.

        :param subject_id_or_key: Required.
        :type subject_id_or_key: str
        :param entitlement_id_or_feature_key: Required.
        :type entitlement_id_or_feature_key: str
        :param grant: Required.
        :type grant: ~openmeter._generated.models.EntitlementGrantCreateInput
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: EntitlementGrant. The EntitlementGrant is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.EntitlementGrant
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def create_grant(
        self,
        subject_id_or_key: str,
        entitlement_id_or_feature_key: str,
        grant: JSON,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> _models.EntitlementGrant:
        """Create subject entitlement grant.

        Grants define a behavior of granting usage for a metered entitlement. They can have complicated
        recurrence and rollover rules, thanks to which you can define a wide range of access patterns
        with a single grant, in most cases you don't have to periodically create new grants. You can
        only issue grants for active metered entitlements.

        A grant defines a given amount of usage that can be consumed for the entitlement. The grant is
        in effect between its effective date and its expiration date. Specifying both is mandatory for
        new grants.

        Grants have a priority setting that determines their order of use. Lower numbers have higher
        priority, with 0 being the highest priority.

        Grants can have a recurrence setting intended to automate the manual reissuing of grants. For
        example, a daily recurrence is equal to reissuing that same grant every day (ignoring rollover
        settings).

        Rollover settings define what happens to the remaining balance of a grant at a reset.
        Balance_After_Reset = MIN(MaxRolloverAmount, MAX(Balance_Before_Reset, MinRolloverAmount))

        Grants cannot be changed once created, only deleted. This is to ensure that balance is
        deterministic regardless of when it is queried.

        :param subject_id_or_key: Required.
        :type subject_id_or_key: str
        :param entitlement_id_or_feature_key: Required.
        :type entitlement_id_or_feature_key: str
        :param grant: Required.
        :type grant: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: EntitlementGrant. The EntitlementGrant is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.EntitlementGrant
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def create_grant(
        self,
        subject_id_or_key: str,
        entitlement_id_or_feature_key: str,
        grant: IO[bytes],
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> _models.EntitlementGrant:
        """Create subject entitlement grant.

        Grants define a behavior of granting usage for a metered entitlement. They can have complicated
        recurrence and rollover rules, thanks to which you can define a wide range of access patterns
        with a single grant, in most cases you don't have to periodically create new grants. You can
        only issue grants for active metered entitlements.

        A grant defines a given amount of usage that can be consumed for the entitlement. The grant is
        in effect between its effective date and its expiration date. Specifying both is mandatory for
        new grants.

        Grants have a priority setting that determines their order of use. Lower numbers have higher
        priority, with 0 being the highest priority.

        Grants can have a recurrence setting intended to automate the manual reissuing of grants. For
        example, a daily recurrence is equal to reissuing that same grant every day (ignoring rollover
        settings).

        Rollover settings define what happens to the remaining balance of a grant at a reset.
        Balance_After_Reset = MIN(MaxRolloverAmount, MAX(Balance_Before_Reset, MinRolloverAmount))

        Grants cannot be changed once created, only deleted. This is to ensure that balance is
        deterministic regardless of when it is queried.

        :param subject_id_or_key: Required.
        :type subject_id_or_key: str
        :param entitlement_id_or_feature_key: Required.
        :type entitlement_id_or_feature_key: str
        :param grant: Required.
        :type grant: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: EntitlementGrant. The EntitlementGrant is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.EntitlementGrant
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def create_grant(
        self,
        subject_id_or_key: str,
        entitlement_id_or_feature_key: str,
        grant: Union[_models.EntitlementGrantCreateInput, JSON, IO[bytes]],
        **kwargs: Any
    ) -> _models.EntitlementGrant:
        """Create subject entitlement grant.

        Grants define a behavior of granting usage for a metered entitlement. They can have complicated
        recurrence and rollover rules, thanks to which you can define a wide range of access patterns
        with a single grant, in most cases you don't have to periodically create new grants. You can
        only issue grants for active metered entitlements.

        A grant defines a given amount of usage that can be consumed for the entitlement. The grant is
        in effect between its effective date and its expiration date. Specifying both is mandatory for
        new grants.

        Grants have a priority setting that determines their order of use. Lower numbers have higher
        priority, with 0 being the highest priority.

        Grants can have a recurrence setting intended to automate the manual reissuing of grants. For
        example, a daily recurrence is equal to reissuing that same grant every day (ignoring rollover
        settings).

        Rollover settings define what happens to the remaining balance of a grant at a reset.
        Balance_After_Reset = MIN(MaxRolloverAmount, MAX(Balance_Before_Reset, MinRolloverAmount))

        Grants cannot be changed once created, only deleted. This is to ensure that balance is
        deterministic regardless of when it is queried.

        :param subject_id_or_key: Required.
        :type subject_id_or_key: str
        :param entitlement_id_or_feature_key: Required.
        :type entitlement_id_or_feature_key: str
        :param grant: Is one of the following types: EntitlementGrantCreateInput, JSON, IO[bytes]
         Required.
        :type grant: ~openmeter._generated.models.EntitlementGrantCreateInput or JSON or IO[bytes]
        :return: EntitlementGrant. The EntitlementGrant is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.EntitlementGrant
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[_models.EntitlementGrant] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(grant, (IOBase, bytes)):
            _content = grant
        else:
            _content = json.dumps(grant, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_entitlements_subjects_create_grant_request(
            subject_id_or_key=subject_id_or_key,
            entitlement_id_or_feature_key=entitlement_id_or_feature_key,
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [201]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            elif response.status_code == 409:
                error = _failsafe_deserialize(_models.ConflictProblemResponse, response)
                raise ResourceExistsError(response=response, model=error)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.EntitlementGrant, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def get_entitlement_value(
        self,
        subject_id_or_key: str,
        entitlement_id_or_feature_key: str,
        *,
        time: Optional[datetime.datetime] = None,
        **kwargs: Any
    ) -> _models.EntitlementValue:
        """Get subject entitlement value.

        This endpoint should be used for access checks and enforcement. All entitlement types share the
        hasAccess property in their value response, but multiple other properties are returned based on
        the entitlement type.

        For convenience reasons, /value works with both entitlementId and featureKey.

        :param subject_id_or_key: Required.
        :type subject_id_or_key: str
        :param entitlement_id_or_feature_key: Required.
        :type entitlement_id_or_feature_key: str
        :keyword time: Default value is None.
        :paramtype time: ~datetime.datetime
        :return: EntitlementValue. The EntitlementValue is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.EntitlementValue
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.EntitlementValue] = kwargs.pop("cls", None)

        _request = build_entitlements_subjects_get_entitlement_value_request(
            subject_id_or_key=subject_id_or_key,
            entitlement_id_or_feature_key=entitlement_id_or_feature_key,
            time=time,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            elif response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.EntitlementValue, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def get_entitlement_history(
        self,
        subject_id_or_key: str,
        entitlement_id: str,
        *,
        window_size: Union[str, _models.WindowSize],
        from_parameter: Optional[datetime.datetime] = None,
        to: Optional[datetime.datetime] = None,
        window_time_zone: Optional[str] = None,
        **kwargs: Any
    ) -> _models.WindowedBalanceHistory:
        """Get subject entitlement history.

        Returns historical balance and usage data for the entitlement. The queried history can span
        accross multiple reset events.

        BurndownHistory returns a continous history of segments, where the segments are seperated by
        events that changed either the grant burndown priority or the usage period.

        WindowedHistory returns windowed usage data for the period enriched with balance information
        and the list of grants that were being burnt down in that window.

        :param subject_id_or_key: Required.
        :type subject_id_or_key: str
        :param entitlement_id: Required.
        :type entitlement_id: str
        :keyword window_size: Windowsize. Known values are: "MINUTE", "HOUR", "DAY", and "MONTH".
         Required.
        :paramtype window_size: str or ~openmeter.models.WindowSize
        :keyword from_parameter: Start of time range to query entitlement: date-time in RFC 3339
         format. Defaults to the last reset. Gets truncated to the granularity of the underlying meter.
         Default value is None.
        :paramtype from_parameter: ~datetime.datetime
        :keyword to: End of time range to query entitlement: date-time in RFC 3339 format. Defaults to
         now.
         If not now then gets truncated to the granularity of the underlying meter. Default value is
         None.
        :paramtype to: ~datetime.datetime
        :keyword window_time_zone: The timezone used when calculating the windows. Default value is
         None.
        :paramtype window_time_zone: str
        :return: WindowedBalanceHistory. The WindowedBalanceHistory is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.WindowedBalanceHistory
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.WindowedBalanceHistory] = kwargs.pop("cls", None)

        _request = build_entitlements_subjects_get_entitlement_history_request(
            subject_id_or_key=subject_id_or_key,
            entitlement_id=entitlement_id,
            window_size=window_size,
            from_parameter=from_parameter,
            to=to,
            window_time_zone=window_time_zone,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            elif response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.WindowedBalanceHistory, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    @overload
    async def reset(
        self,
        subject_id_or_key: str,
        entitlement_id: str,
        reset: _models.ResetEntitlementUsageInput,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> None:
        """Reset subject entitlement.

        Reset marks the start of a new usage period for the entitlement and initiates grant rollover.
        At the start of a period usage is zerod out and grants are rolled over based on their rollover
        settings. It would typically be synced with the subjects billing period to enforce usage based
        on their subscription.

        Usage is automatically reset for metered entitlements based on their usage period, but this
        endpoint allows to manually reset it at any time. When doing so the period anchor of the
        entitlement can be changed if needed.

        :param subject_id_or_key: Required.
        :type subject_id_or_key: str
        :param entitlement_id: Required.
        :type entitlement_id: str
        :param reset: Required.
        :type reset: ~openmeter._generated.models.ResetEntitlementUsageInput
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def reset(
        self,
        subject_id_or_key: str,
        entitlement_id: str,
        reset: JSON,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> None:
        """Reset subject entitlement.

        Reset marks the start of a new usage period for the entitlement and initiates grant rollover.
        At the start of a period usage is zerod out and grants are rolled over based on their rollover
        settings. It would typically be synced with the subjects billing period to enforce usage based
        on their subscription.

        Usage is automatically reset for metered entitlements based on their usage period, but this
        endpoint allows to manually reset it at any time. When doing so the period anchor of the
        entitlement can be changed if needed.

        :param subject_id_or_key: Required.
        :type subject_id_or_key: str
        :param entitlement_id: Required.
        :type entitlement_id: str
        :param reset: Required.
        :type reset: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def reset(
        self,
        subject_id_or_key: str,
        entitlement_id: str,
        reset: IO[bytes],
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> None:
        """Reset subject entitlement.

        Reset marks the start of a new usage period for the entitlement and initiates grant rollover.
        At the start of a period usage is zerod out and grants are rolled over based on their rollover
        settings. It would typically be synced with the subjects billing period to enforce usage based
        on their subscription.

        Usage is automatically reset for metered entitlements based on their usage period, but this
        endpoint allows to manually reset it at any time. When doing so the period anchor of the
        entitlement can be changed if needed.

        :param subject_id_or_key: Required.
        :type subject_id_or_key: str
        :param entitlement_id: Required.
        :type entitlement_id: str
        :param reset: Required.
        :type reset: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def reset(
        self,
        subject_id_or_key: str,
        entitlement_id: str,
        reset: Union[_models.ResetEntitlementUsageInput, JSON, IO[bytes]],
        **kwargs: Any
    ) -> None:
        """Reset subject entitlement.

        Reset marks the start of a new usage period for the entitlement and initiates grant rollover.
        At the start of a period usage is zerod out and grants are rolled over based on their rollover
        settings. It would typically be synced with the subjects billing period to enforce usage based
        on their subscription.

        Usage is automatically reset for metered entitlements based on their usage period, but this
        endpoint allows to manually reset it at any time. When doing so the period anchor of the
        entitlement can be changed if needed.

        :param subject_id_or_key: Required.
        :type subject_id_or_key: str
        :param entitlement_id: Required.
        :type entitlement_id: str
        :param reset: Is one of the following types: ResetEntitlementUsageInput, JSON, IO[bytes]
         Required.
        :type reset: ~openmeter._generated.models.ResetEntitlementUsageInput or JSON or IO[bytes]
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[None] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(reset, (IOBase, bytes)):
            _content = reset
        else:
            _content = json.dumps(reset, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_entitlements_subjects_reset_request(
            subject_id_or_key=subject_id_or_key,
            entitlement_id=entitlement_id,
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = False
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [204]:
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            elif response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if cls:
            return cls(pipeline_response, None, {})  # type: ignore


class EntitlementsCustomerOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`customer` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

    async def get_customer_access(
        self, customer_id_or_key: "_types.ULIDOrExternalKey", **kwargs: Any
    ) -> _models.CustomerAccess:
        """Get customer access.

        Get the overall access of a customer.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :return: CustomerAccess. The CustomerAccess is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.CustomerAccess
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.CustomerAccess] = kwargs.pop("cls", None)

        _request = build_entitlements_customer_get_customer_access_request(
            customer_id_or_key=customer_id_or_key,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            elif response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.CustomerAccess, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore


class EntitlementsCustomerEntitlementOperations:  # pylint: disable=name-too-long
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`customer_entitlement` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

    async def get_customer_entitlement_value(
        self,
        customer_id_or_key: "_types.ULIDOrExternalKey",
        feature_key: str,
        *,
        time: Optional[datetime.datetime] = None,
        **kwargs: Any
    ) -> _models.EntitlementValue:
        """Get customer entitlement value.

        Checks customer access to a given feature (by key). All entitlement types share the hasAccess
        property in their value response, but multiple other properties are returned based on the
        entitlement type.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :param feature_key: Required.
        :type feature_key: str
        :keyword time: Default value is None.
        :paramtype time: ~datetime.datetime
        :return: EntitlementValue. The EntitlementValue is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.EntitlementValue
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.EntitlementValue] = kwargs.pop("cls", None)

        _request = build_entitlements_customer_entitlement_get_customer_entitlement_value_request(
            customer_id_or_key=customer_id_or_key,
            feature_key=feature_key,
            time=time,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            elif response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.EntitlementValue, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore


class BillingProfilesOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`profiles` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

    async def list(
        self,
        *,
        include_archived: Optional[bool] = None,
        expand: Optional[List[Union[str, _models.BillingProfileExpand]]] = None,
        page: Optional[int] = None,
        page_size: Optional[int] = None,
        order: Optional[Union[str, _models.SortOrder]] = None,
        order_by: Optional[Union[str, _models.BillingProfileOrderBy]] = None,
        **kwargs: Any
    ) -> _models.BillingProfilePaginatedResponse:
        """List billing profiles.

        List all billing profiles matching the specified filters.

        The expand option can be used to include additional information (besides the billing profile)
        in the response. For example by adding the expand=apps option the apps used by the billing
        profile
        will be included in the response.

        :keyword include_archived: Default value is None.
        :paramtype include_archived: bool
        :keyword expand: Default value is None.
        :paramtype expand: list[str or ~openmeter.models.BillingProfileExpand]
        :keyword page: Page index.

         Default is 1. Default value is None.
        :paramtype page: int
        :keyword page_size: The maximum number of items per page.

         Default is 100. Default value is None.
        :paramtype page_size: int
        :keyword order: The order direction. Known values are: "ASC" and "DESC". Default value is None.
        :paramtype order: str or ~openmeter.models.SortOrder
        :keyword order_by: The order by field. Known values are: "createdAt", "updatedAt", "default",
         and "name". Default value is None.
        :paramtype order_by: str or ~openmeter.models.BillingProfileOrderBy
        :return: BillingProfilePaginatedResponse. The BillingProfilePaginatedResponse is compatible
         with MutableMapping
        :rtype: ~openmeter._generated.models.BillingProfilePaginatedResponse
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.BillingProfilePaginatedResponse] = kwargs.pop("cls", None)

        _request = build_billing_profiles_list_request(
            include_archived=include_archived,
            expand=expand,
            page=page,
            page_size=page_size,
            order=order,
            order_by=order_by,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.BillingProfilePaginatedResponse, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    @overload
    async def create(
        self, profile: _models.BillingProfileCreate, *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.BillingProfile:
        """Create a new billing profile.

        Create a new billing profile

        Billing profiles are representations of a customer's billing information. Customer overrides
        can be applied to a billing profile to customize the billing behavior for a specific customer.

        :param profile: Required.
        :type profile: ~openmeter._generated.models.BillingProfileCreate
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: BillingProfile. The BillingProfile is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.BillingProfile
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def create(
        self, profile: JSON, *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.BillingProfile:
        """Create a new billing profile.

        Create a new billing profile

        Billing profiles are representations of a customer's billing information. Customer overrides
        can be applied to a billing profile to customize the billing behavior for a specific customer.

        :param profile: Required.
        :type profile: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: BillingProfile. The BillingProfile is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.BillingProfile
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def create(
        self, profile: IO[bytes], *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.BillingProfile:
        """Create a new billing profile.

        Create a new billing profile

        Billing profiles are representations of a customer's billing information. Customer overrides
        can be applied to a billing profile to customize the billing behavior for a specific customer.

        :param profile: Required.
        :type profile: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: BillingProfile. The BillingProfile is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.BillingProfile
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def create(
        self, profile: Union[_models.BillingProfileCreate, JSON, IO[bytes]], **kwargs: Any
    ) -> _models.BillingProfile:
        """Create a new billing profile.

        Create a new billing profile

        Billing profiles are representations of a customer's billing information. Customer overrides
        can be applied to a billing profile to customize the billing behavior for a specific customer.

        :param profile: Is one of the following types: BillingProfileCreate, JSON, IO[bytes] Required.
        :type profile: ~openmeter._generated.models.BillingProfileCreate or JSON or IO[bytes]
        :return: BillingProfile. The BillingProfile is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.BillingProfile
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[_models.BillingProfile] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(profile, (IOBase, bytes)):
            _content = profile
        else:
            _content = json.dumps(profile, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_billing_profiles_create_request(
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [201]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.BillingProfile, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def delete(self, id: str, **kwargs: Any) -> None:
        """Delete a billing profile.

        Delete a billing profile by id.

        Only such billing profiles can be deleted that are:

        * not the default one
        * not pinned to any customer using customer overrides
        * only have finalized invoices.

        :param id: Required.
        :type id: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[None] = kwargs.pop("cls", None)

        _request = build_billing_profiles_delete_request(
            id=id,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = False
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [204]:
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if cls:
            return cls(pipeline_response, None, {})  # type: ignore

    async def get(
        self, id: str, *, expand: Optional[List[Union[str, _models.BillingProfileExpand]]] = None, **kwargs: Any
    ) -> _models.BillingProfile:
        """Get a billing profile.

        Get a billing profile by id.

        The expand option can be used to include additional information (besides the billing profile)
        in the response. For example by adding the expand=apps option the apps used by the billing
        profile
        will be included in the response.

        :param id: Required.
        :type id: str
        :keyword expand: Default value is None.
        :paramtype expand: list[str or ~openmeter.models.BillingProfileExpand]
        :return: BillingProfile. The BillingProfile is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.BillingProfile
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.BillingProfile] = kwargs.pop("cls", None)

        _request = build_billing_profiles_get_request(
            id=id,
            expand=expand,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.BillingProfile, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    @overload
    async def update(
        self,
        id: str,
        profile: _models.BillingProfileReplaceUpdateWithWorkflow,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> _models.BillingProfile:
        """Update a billing profile.

        Update a billing profile by id.

        The apps field cannot be updated directly, if an app change is desired a new
        profile should be created.

        :param id: Required.
        :type id: str
        :param profile: Required.
        :type profile: ~openmeter._generated.models.BillingProfileReplaceUpdateWithWorkflow
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: BillingProfile. The BillingProfile is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.BillingProfile
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def update(
        self, id: str, profile: JSON, *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.BillingProfile:
        """Update a billing profile.

        Update a billing profile by id.

        The apps field cannot be updated directly, if an app change is desired a new
        profile should be created.

        :param id: Required.
        :type id: str
        :param profile: Required.
        :type profile: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: BillingProfile. The BillingProfile is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.BillingProfile
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def update(
        self, id: str, profile: IO[bytes], *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.BillingProfile:
        """Update a billing profile.

        Update a billing profile by id.

        The apps field cannot be updated directly, if an app change is desired a new
        profile should be created.

        :param id: Required.
        :type id: str
        :param profile: Required.
        :type profile: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: BillingProfile. The BillingProfile is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.BillingProfile
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def update(
        self, id: str, profile: Union[_models.BillingProfileReplaceUpdateWithWorkflow, JSON, IO[bytes]], **kwargs: Any
    ) -> _models.BillingProfile:
        """Update a billing profile.

        Update a billing profile by id.

        The apps field cannot be updated directly, if an app change is desired a new
        profile should be created.

        :param id: Required.
        :type id: str
        :param profile: Is one of the following types: BillingProfileReplaceUpdateWithWorkflow, JSON,
         IO[bytes] Required.
        :type profile: ~openmeter._generated.models.BillingProfileReplaceUpdateWithWorkflow or JSON or
         IO[bytes]
        :return: BillingProfile. The BillingProfile is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.BillingProfile
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[_models.BillingProfile] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(profile, (IOBase, bytes)):
            _content = profile
        else:
            _content = json.dumps(profile, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_billing_profiles_update_request(
            id=id,
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.BillingProfile, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore


class BillingCustomerOverridesOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`customer_overrides` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

    async def list(
        self,
        *,
        billing_profile: Optional[List[str]] = None,
        customers_without_pinned_profile: Optional[bool] = None,
        include_all_customers: Optional[bool] = None,
        customer_id: Optional[List[str]] = None,
        customer_name: Optional[str] = None,
        customer_key: Optional[str] = None,
        customer_primary_email: Optional[str] = None,
        expand: Optional[List[Union[str, _models.BillingProfileCustomerOverrideExpand]]] = None,
        order: Optional[Union[str, _models.SortOrder]] = None,
        order_by: Optional[Union[str, _models.BillingProfileCustomerOverrideOrderBy]] = None,
        page: Optional[int] = None,
        page_size: Optional[int] = None,
        **kwargs: Any
    ) -> _models.BillingProfileCustomerOverrideWithDetailsPaginatedResponse:
        """List customer overrides.

        List customer overrides using the specified filters.

        The response will include the customer override values and the merged billing profile values.

        If the includeAllCustomers is set to true, the list contains all customers. This mode is
        useful for getting the current effective billing workflow settings for all users regardless
        if they have customer orverrides or not.

        :keyword billing_profile: Filter by billing profile. Default value is None.
        :paramtype billing_profile: list[str]
        :keyword customers_without_pinned_profile: Only return customers without pinned billing
         profiles. This implicitly sets includeAllCustomers to true. Default value is None.
        :paramtype customers_without_pinned_profile: bool
        :keyword include_all_customers: Include customers without customer overrides.

         If set to false only the customers specifically associated with a billing profile will be
         returned.

         If set to true, in case of the default billing profile, all customers will be returned.
         Default value is None.
        :paramtype include_all_customers: bool
        :keyword customer_id: Filter by customer id. Default value is None.
        :paramtype customer_id: list[str]
        :keyword customer_name: Filter by customer name. Default value is None.
        :paramtype customer_name: str
        :keyword customer_key: Filter by customer key. Default value is None.
        :paramtype customer_key: str
        :keyword customer_primary_email: Filter by customer primary email. Default value is None.
        :paramtype customer_primary_email: str
        :keyword expand: Expand the response with additional details. Default value is None.
        :paramtype expand: list[str or ~openmeter.models.BillingProfileCustomerOverrideExpand]
        :keyword order: The order direction. Known values are: "ASC" and "DESC". Default value is None.
        :paramtype order: str or ~openmeter.models.SortOrder
        :keyword order_by: The order by field. Known values are: "customerId", "customerName",
         "customerKey", "customerPrimaryEmail", and "customerCreatedAt". Default value is None.
        :paramtype order_by: str or ~openmeter.models.BillingProfileCustomerOverrideOrderBy
        :keyword page: Page index.

         Default is 1. Default value is None.
        :paramtype page: int
        :keyword page_size: The maximum number of items per page.

         Default is 100. Default value is None.
        :paramtype page_size: int
        :return: BillingProfileCustomerOverrideWithDetailsPaginatedResponse. The
         BillingProfileCustomerOverrideWithDetailsPaginatedResponse is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.BillingProfileCustomerOverrideWithDetailsPaginatedResponse
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.BillingProfileCustomerOverrideWithDetailsPaginatedResponse] = kwargs.pop("cls", None)

        _request = build_billing_customer_overrides_list_request(
            billing_profile=billing_profile,
            customers_without_pinned_profile=customers_without_pinned_profile,
            include_all_customers=include_all_customers,
            customer_id=customer_id,
            customer_name=customer_name,
            customer_key=customer_key,
            customer_primary_email=customer_primary_email,
            expand=expand,
            order=order,
            order_by=order_by,
            page=page,
            page_size=page_size,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(
                _models.BillingProfileCustomerOverrideWithDetailsPaginatedResponse, response.json()
            )

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    @overload
    async def upsert(
        self,
        customer_id: str,
        request: _models.BillingProfileCustomerOverrideCreate,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> _models.BillingProfileCustomerOverrideWithDetails:
        """Create a new or update a customer override.

        The customer override can be used to pin a given customer to a billing profile
        different from the default one.

        This can be used to test the effect of different billing profiles before making them
        the default ones or have different workflow settings for example for enterprise customers.

        :param customer_id: Required.
        :type customer_id: str
        :param request: Required.
        :type request: ~openmeter._generated.models.BillingProfileCustomerOverrideCreate
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: BillingProfileCustomerOverrideWithDetails. The
         BillingProfileCustomerOverrideWithDetails is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.BillingProfileCustomerOverrideWithDetails
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def upsert(
        self, customer_id: str, request: JSON, *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.BillingProfileCustomerOverrideWithDetails:
        """Create a new or update a customer override.

        The customer override can be used to pin a given customer to a billing profile
        different from the default one.

        This can be used to test the effect of different billing profiles before making them
        the default ones or have different workflow settings for example for enterprise customers.

        :param customer_id: Required.
        :type customer_id: str
        :param request: Required.
        :type request: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: BillingProfileCustomerOverrideWithDetails. The
         BillingProfileCustomerOverrideWithDetails is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.BillingProfileCustomerOverrideWithDetails
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def upsert(
        self, customer_id: str, request: IO[bytes], *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.BillingProfileCustomerOverrideWithDetails:
        """Create a new or update a customer override.

        The customer override can be used to pin a given customer to a billing profile
        different from the default one.

        This can be used to test the effect of different billing profiles before making them
        the default ones or have different workflow settings for example for enterprise customers.

        :param customer_id: Required.
        :type customer_id: str
        :param request: Required.
        :type request: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: BillingProfileCustomerOverrideWithDetails. The
         BillingProfileCustomerOverrideWithDetails is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.BillingProfileCustomerOverrideWithDetails
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def upsert(
        self,
        customer_id: str,
        request: Union[_models.BillingProfileCustomerOverrideCreate, JSON, IO[bytes]],
        **kwargs: Any
    ) -> _models.BillingProfileCustomerOverrideWithDetails:
        """Create a new or update a customer override.

        The customer override can be used to pin a given customer to a billing profile
        different from the default one.

        This can be used to test the effect of different billing profiles before making them
        the default ones or have different workflow settings for example for enterprise customers.

        :param customer_id: Required.
        :type customer_id: str
        :param request: Is one of the following types: BillingProfileCustomerOverrideCreate, JSON,
         IO[bytes] Required.
        :type request: ~openmeter._generated.models.BillingProfileCustomerOverrideCreate or JSON or
         IO[bytes]
        :return: BillingProfileCustomerOverrideWithDetails. The
         BillingProfileCustomerOverrideWithDetails is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.BillingProfileCustomerOverrideWithDetails
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[_models.BillingProfileCustomerOverrideWithDetails] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(request, (IOBase, bytes)):
            _content = request
        else:
            _content = json.dumps(request, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_billing_customer_overrides_upsert_request(
            customer_id=customer_id,
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.BillingProfileCustomerOverrideWithDetails, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def get(
        self,
        customer_id: str,
        *,
        expand: Optional[List[Union[str, _models.BillingProfileCustomerOverrideExpand]]] = None,
        **kwargs: Any
    ) -> _models.BillingProfileCustomerOverrideWithDetails:
        """Get a customer override.

        Get a customer override by customer id.

        The response will include the customer override values and the merged billing profile values.

        If the customer override is not found, the default billing profile's values are returned. This
        behavior
        allows for getting a merged profile regardless of the customer override existence.

        :param customer_id: Required.
        :type customer_id: str
        :keyword expand: Default value is None.
        :paramtype expand: list[str or ~openmeter.models.BillingProfileCustomerOverrideExpand]
        :return: BillingProfileCustomerOverrideWithDetails. The
         BillingProfileCustomerOverrideWithDetails is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.BillingProfileCustomerOverrideWithDetails
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.BillingProfileCustomerOverrideWithDetails] = kwargs.pop("cls", None)

        _request = build_billing_customer_overrides_get_request(
            customer_id=customer_id,
            expand=expand,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.BillingProfileCustomerOverrideWithDetails, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def delete(self, customer_id: str, **kwargs: Any) -> None:
        """Delete a customer override.

        Delete a customer override by customer id.

        This will remove the customer override and the customer will be subject to the default
        billing profile's settings again.

        :param customer_id: Required.
        :type customer_id: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[None] = kwargs.pop("cls", None)

        _request = build_billing_customer_overrides_delete_request(
            customer_id=customer_id,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = False
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [204]:
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if cls:
            return cls(pipeline_response, None, {})  # type: ignore


class BillingInvoicesEndpointsOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`invoices_endpoints` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

    @overload
    async def invoice_pending_lines_action(
        self, request: _models.InvoicePendingLinesActionInput, *, content_type: str = "application/json", **kwargs: Any
    ) -> List[_models.Invoice]:
        """Invoice a customer based on the pending line items.

        Create a new invoice from the pending line items.

        This should be only called if for some reason we need to invoice a customer outside of the
        normal billing cycle.

        When creating an invoice, the pending line items will be marked as invoiced and the invoice
        will be created with the total amount of the pending items.

        New pending line items will be created for the period between now() and the next billing
        cycle's begining date for any metered item.

        The call can return multiple invoices if the pending line items are in different currencies.

        :param request: Required.
        :type request: ~openmeter._generated.models.InvoicePendingLinesActionInput
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: list of Invoice
        :rtype: list[~openmeter._generated.models.Invoice]
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def invoice_pending_lines_action(
        self, request: JSON, *, content_type: str = "application/json", **kwargs: Any
    ) -> List[_models.Invoice]:
        """Invoice a customer based on the pending line items.

        Create a new invoice from the pending line items.

        This should be only called if for some reason we need to invoice a customer outside of the
        normal billing cycle.

        When creating an invoice, the pending line items will be marked as invoiced and the invoice
        will be created with the total amount of the pending items.

        New pending line items will be created for the period between now() and the next billing
        cycle's begining date for any metered item.

        The call can return multiple invoices if the pending line items are in different currencies.

        :param request: Required.
        :type request: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: list of Invoice
        :rtype: list[~openmeter._generated.models.Invoice]
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def invoice_pending_lines_action(
        self, request: IO[bytes], *, content_type: str = "application/json", **kwargs: Any
    ) -> List[_models.Invoice]:
        """Invoice a customer based on the pending line items.

        Create a new invoice from the pending line items.

        This should be only called if for some reason we need to invoice a customer outside of the
        normal billing cycle.

        When creating an invoice, the pending line items will be marked as invoiced and the invoice
        will be created with the total amount of the pending items.

        New pending line items will be created for the period between now() and the next billing
        cycle's begining date for any metered item.

        The call can return multiple invoices if the pending line items are in different currencies.

        :param request: Required.
        :type request: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: list of Invoice
        :rtype: list[~openmeter._generated.models.Invoice]
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def invoice_pending_lines_action(
        self, request: Union[_models.InvoicePendingLinesActionInput, JSON, IO[bytes]], **kwargs: Any
    ) -> List[_models.Invoice]:
        """Invoice a customer based on the pending line items.

        Create a new invoice from the pending line items.

        This should be only called if for some reason we need to invoice a customer outside of the
        normal billing cycle.

        When creating an invoice, the pending line items will be marked as invoiced and the invoice
        will be created with the total amount of the pending items.

        New pending line items will be created for the period between now() and the next billing
        cycle's begining date for any metered item.

        The call can return multiple invoices if the pending line items are in different currencies.

        :param request: Is one of the following types: InvoicePendingLinesActionInput, JSON, IO[bytes]
         Required.
        :type request: ~openmeter._generated.models.InvoicePendingLinesActionInput or JSON or IO[bytes]
        :return: list of Invoice
        :rtype: list[~openmeter._generated.models.Invoice]
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[List[_models.Invoice]] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(request, (IOBase, bytes)):
            _content = request
        else:
            _content = json.dumps(request, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_billing_invoices_endpoints_invoice_pending_lines_action_request(
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [201]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(List[_models.Invoice], response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def list(
        self,
        *,
        statuses: Optional[List[Union[str, _models.InvoiceStatus]]] = None,
        extended_statuses: Optional[List[str]] = None,
        issued_after: Optional[datetime.datetime] = None,
        issued_before: Optional[datetime.datetime] = None,
        period_start_after: Optional[datetime.datetime] = None,
        period_start_before: Optional[datetime.datetime] = None,
        created_after: Optional[datetime.datetime] = None,
        created_before: Optional[datetime.datetime] = None,
        expand: Optional[List[Union[str, _models.InvoiceExpand]]] = None,
        customers: Optional[List[str]] = None,
        include_deleted: Optional[bool] = None,
        page: Optional[int] = None,
        page_size: Optional[int] = None,
        order: Optional[Union[str, _models.SortOrder]] = None,
        order_by: Optional[Union[str, _models.InvoiceOrderBy]] = None,
        **kwargs: Any
    ) -> _models.InvoicePaginatedResponse:
        """List invoices.

        List invoices based on the specified filters.

        The expand option can be used to include additional information (besides the invoice header and
        totals)
        in the response. For example by adding the expand=lines option the invoice lines will be
        included in the response.

        Gathering invoices will always show the current usage calculated on the fly.

        :keyword statuses: Filter by the invoice status. Default value is None.
        :paramtype statuses: list[str or ~openmeter.models.InvoiceStatus]
        :keyword extended_statuses: Filter by invoice extended statuses. Default value is None.
        :paramtype extended_statuses: list[str]
        :keyword issued_after: Filter by invoice issued time.
         Inclusive. Default value is None.
        :paramtype issued_after: ~datetime.datetime
        :keyword issued_before: Filter by invoice issued time.
         Inclusive. Default value is None.
        :paramtype issued_before: ~datetime.datetime
        :keyword period_start_after: Filter by period start time.
         Inclusive. Default value is None.
        :paramtype period_start_after: ~datetime.datetime
        :keyword period_start_before: Filter by period start time.
         Inclusive. Default value is None.
        :paramtype period_start_before: ~datetime.datetime
        :keyword created_after: Filter by invoice created time.
         Inclusive. Default value is None.
        :paramtype created_after: ~datetime.datetime
        :keyword created_before: Filter by invoice created time.
         Inclusive. Default value is None.
        :paramtype created_before: ~datetime.datetime
        :keyword expand: What parts of the list output to expand in listings. Default value is None.
        :paramtype expand: list[str or ~openmeter.models.InvoiceExpand]
        :keyword customers: Filter by customer ID. Default value is None.
        :paramtype customers: list[str]
        :keyword include_deleted: Include deleted invoices. Default value is None.
        :paramtype include_deleted: bool
        :keyword page: Page index.

         Default is 1. Default value is None.
        :paramtype page: int
        :keyword page_size: The maximum number of items per page.

         Default is 100. Default value is None.
        :paramtype page_size: int
        :keyword order: The order direction. Known values are: "ASC" and "DESC". Default value is None.
        :paramtype order: str or ~openmeter.models.SortOrder
        :keyword order_by: The order by field. Known values are: "customer.name", "issuedAt", "status",
         "createdAt", "updatedAt", and "periodStart". Default value is None.
        :paramtype order_by: str or ~openmeter.models.InvoiceOrderBy
        :return: InvoicePaginatedResponse. The InvoicePaginatedResponse is compatible with
         MutableMapping
        :rtype: ~openmeter._generated.models.InvoicePaginatedResponse
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.InvoicePaginatedResponse] = kwargs.pop("cls", None)

        _request = build_billing_invoices_endpoints_list_request(
            statuses=statuses,
            extended_statuses=extended_statuses,
            issued_after=issued_after,
            issued_before=issued_before,
            period_start_after=period_start_after,
            period_start_before=period_start_before,
            created_after=created_after,
            created_before=created_before,
            expand=expand,
            customers=customers,
            include_deleted=include_deleted,
            page=page,
            page_size=page_size,
            order=order,
            order_by=order_by,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.InvoicePaginatedResponse, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore


class BillingInvoiceEndpointsOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`invoice_endpoints` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

    async def get_invoice(
        self,
        invoice_id: str,
        *,
        expand: Optional[List[Union[str, _models.InvoiceExpand]]] = None,
        include_deleted_lines: Optional[bool] = None,
        **kwargs: Any
    ) -> _models.Invoice:
        """Get an invoice.

        Get an invoice by ID.

        Gathering invoices will always show the current usage calculated on the fly.

        :param invoice_id: Required.
        :type invoice_id: str
        :keyword expand: Default value is None.
        :paramtype expand: list[str or ~openmeter.models.InvoiceExpand]
        :keyword include_deleted_lines: Default value is None.
        :paramtype include_deleted_lines: bool
        :return: Invoice. The Invoice is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Invoice
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.Invoice] = kwargs.pop("cls", None)

        _request = build_billing_invoice_endpoints_get_invoice_request(
            invoice_id=invoice_id,
            expand=expand,
            include_deleted_lines=include_deleted_lines,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.Invoice, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def delete_invoice(self, invoice_id: str, **kwargs: Any) -> None:
        """Delete an invoice.

        Delete an invoice

        Only invoices that are in the draft (or earlier) status can be deleted.

        Invoices that are post finalization can only be voided.

        :param invoice_id: Required.
        :type invoice_id: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[None] = kwargs.pop("cls", None)

        _request = build_billing_invoice_endpoints_delete_invoice_request(
            invoice_id=invoice_id,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = False
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [204]:
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if cls:
            return cls(pipeline_response, None, {})  # type: ignore

    @overload
    async def update_invoice(
        self,
        invoice_id: str,
        request: _models.InvoiceReplaceUpdate,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> _models.Invoice:
        """Update an invoice.

        Update an invoice

        Only invoices in draft or earlier status can be updated.

        :param invoice_id: Required.
        :type invoice_id: str
        :param request: Required.
        :type request: ~openmeter._generated.models.InvoiceReplaceUpdate
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Invoice. The Invoice is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Invoice
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def update_invoice(
        self, invoice_id: str, request: JSON, *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.Invoice:
        """Update an invoice.

        Update an invoice

        Only invoices in draft or earlier status can be updated.

        :param invoice_id: Required.
        :type invoice_id: str
        :param request: Required.
        :type request: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Invoice. The Invoice is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Invoice
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def update_invoice(
        self, invoice_id: str, request: IO[bytes], *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.Invoice:
        """Update an invoice.

        Update an invoice

        Only invoices in draft or earlier status can be updated.

        :param invoice_id: Required.
        :type invoice_id: str
        :param request: Required.
        :type request: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Invoice. The Invoice is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Invoice
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def update_invoice(
        self, invoice_id: str, request: Union[_models.InvoiceReplaceUpdate, JSON, IO[bytes]], **kwargs: Any
    ) -> _models.Invoice:
        """Update an invoice.

        Update an invoice

        Only invoices in draft or earlier status can be updated.

        :param invoice_id: Required.
        :type invoice_id: str
        :param request: Is one of the following types: InvoiceReplaceUpdate, JSON, IO[bytes] Required.
        :type request: ~openmeter._generated.models.InvoiceReplaceUpdate or JSON or IO[bytes]
        :return: Invoice. The Invoice is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Invoice
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[_models.Invoice] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(request, (IOBase, bytes)):
            _content = request
        else:
            _content = json.dumps(request, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_billing_invoice_endpoints_update_invoice_request(
            invoice_id=invoice_id,
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.Invoice, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def recalculate_tax_action(self, invoice_id: str, **kwargs: Any) -> _models.Invoice:
        """Recalculate an invoice's tax amounts.

        Recalculate an invoice's tax amounts (using the app set in the customer's billing profile)

        Note: charges might apply, depending on the tax provider.

        :param invoice_id: Required.
        :type invoice_id: str
        :return: Invoice. The Invoice is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Invoice
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.Invoice] = kwargs.pop("cls", None)

        _request = build_billing_invoice_endpoints_recalculate_tax_action_request(
            invoice_id=invoice_id,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.Invoice, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def approve_action(self, invoice_id: str, **kwargs: Any) -> _models.Invoice:
        """Send the invoice to the customer.

        Approve an invoice and start executing the payment workflow.

        This call instantly sends the invoice to the customer using the configured billing profile app.

        This call is valid in two invoice statuses:

        * `draft`: the invoice will be sent to the customer, the invluce state becomes issued
        * `manual_approval_needed`: the invoice will be sent to the customer, the invoice state becomes
        issued.

        :param invoice_id: Required.
        :type invoice_id: str
        :return: Invoice. The Invoice is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Invoice
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.Invoice] = kwargs.pop("cls", None)

        _request = build_billing_invoice_endpoints_approve_action_request(
            invoice_id=invoice_id,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.Invoice, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    @overload
    async def void_invoice_action(
        self,
        invoice_id: str,
        request: _models.VoidInvoiceActionInput,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> _models.Invoice:
        """Void an invoice.

        Void an invoice

        Only invoices that have been alread issued can be voided.

        Voiding an invoice will mark it as voided, the user can specify how to handle the voided line
        items.

        :param invoice_id: Required.
        :type invoice_id: str
        :param request: Required.
        :type request: ~openmeter._generated.models.VoidInvoiceActionInput
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Invoice. The Invoice is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Invoice
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def void_invoice_action(
        self, invoice_id: str, request: JSON, *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.Invoice:
        """Void an invoice.

        Void an invoice

        Only invoices that have been alread issued can be voided.

        Voiding an invoice will mark it as voided, the user can specify how to handle the voided line
        items.

        :param invoice_id: Required.
        :type invoice_id: str
        :param request: Required.
        :type request: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Invoice. The Invoice is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Invoice
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def void_invoice_action(
        self, invoice_id: str, request: IO[bytes], *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.Invoice:
        """Void an invoice.

        Void an invoice

        Only invoices that have been alread issued can be voided.

        Voiding an invoice will mark it as voided, the user can specify how to handle the voided line
        items.

        :param invoice_id: Required.
        :type invoice_id: str
        :param request: Required.
        :type request: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Invoice. The Invoice is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Invoice
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def void_invoice_action(
        self, invoice_id: str, request: Union[_models.VoidInvoiceActionInput, JSON, IO[bytes]], **kwargs: Any
    ) -> _models.Invoice:
        """Void an invoice.

        Void an invoice

        Only invoices that have been alread issued can be voided.

        Voiding an invoice will mark it as voided, the user can specify how to handle the voided line
        items.

        :param invoice_id: Required.
        :type invoice_id: str
        :param request: Is one of the following types: VoidInvoiceActionInput, JSON, IO[bytes]
         Required.
        :type request: ~openmeter._generated.models.VoidInvoiceActionInput or JSON or IO[bytes]
        :return: Invoice. The Invoice is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Invoice
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[_models.Invoice] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(request, (IOBase, bytes)):
            _content = request
        else:
            _content = json.dumps(request, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_billing_invoice_endpoints_void_invoice_action_request(
            invoice_id=invoice_id,
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.Invoice, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def advance_action(self, invoice_id: str, **kwargs: Any) -> _models.Invoice:
        """Advance the invoice's state to the next status.

        Advance the invoice's state to the next status.

        The call doesn't "approve the invoice", it only advances the invoice to the next status if the
        transition would be automatic.

        The action can be called when the invoice's statusDetails' actions field contain the "advance"
        action.

        :param invoice_id: Required.
        :type invoice_id: str
        :return: Invoice. The Invoice is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Invoice
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.Invoice] = kwargs.pop("cls", None)

        _request = build_billing_invoice_endpoints_advance_action_request(
            invoice_id=invoice_id,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.Invoice, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def retry_action(self, invoice_id: str, **kwargs: Any) -> _models.Invoice:
        """Retry advancing the invoice after a failed attempt.

        Retry advancing the invoice after a failed attempt.

        The action can be called when the invoice's statusDetails' actions field contain the "retry"
        action.

        :param invoice_id: Required.
        :type invoice_id: str
        :return: Invoice. The Invoice is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Invoice
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.Invoice] = kwargs.pop("cls", None)

        _request = build_billing_invoice_endpoints_retry_action_request(
            invoice_id=invoice_id,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.Invoice, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def snapshot_quantities_action(self, invoice_id: str, **kwargs: Any) -> _models.Invoice:
        """Snapshot quantities for usage based line items.

        Snapshot quantities for usage based line items.

        This call will snapshot the quantities for all usage based line items in the invoice.

        This call is only valid in ``draft.waiting_for_collection`` status, where the collection period
        can be skipped using this action.

        :param invoice_id: Required.
        :type invoice_id: str
        :return: Invoice. The Invoice is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Invoice
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.Invoice] = kwargs.pop("cls", None)

        _request = build_billing_invoice_endpoints_snapshot_quantities_action_request(
            invoice_id=invoice_id,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.Invoice, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore


class BillingCustomerInvoiceEndpointsOperations:  # pylint: disable=name-too-long
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`customer_invoice_endpoints` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

    @overload
    async def simulate_invoice(
        self,
        customer_id: str,
        request: _models.InvoiceSimulationInput,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> _models.Invoice:
        """Simulate an invoice for a customer.

        Simulate an invoice for a customer.

        This call will simulate an invoice for a customer based on the pending line items.

        The call will return the total amount of the invoice and the line items that will be included
        in the invoice.

        :param customer_id: Required.
        :type customer_id: str
        :param request: Required.
        :type request: ~openmeter._generated.models.InvoiceSimulationInput
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Invoice. The Invoice is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Invoice
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def simulate_invoice(
        self, customer_id: str, request: JSON, *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.Invoice:
        """Simulate an invoice for a customer.

        Simulate an invoice for a customer.

        This call will simulate an invoice for a customer based on the pending line items.

        The call will return the total amount of the invoice and the line items that will be included
        in the invoice.

        :param customer_id: Required.
        :type customer_id: str
        :param request: Required.
        :type request: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Invoice. The Invoice is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Invoice
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def simulate_invoice(
        self, customer_id: str, request: IO[bytes], *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.Invoice:
        """Simulate an invoice for a customer.

        Simulate an invoice for a customer.

        This call will simulate an invoice for a customer based on the pending line items.

        The call will return the total amount of the invoice and the line items that will be included
        in the invoice.

        :param customer_id: Required.
        :type customer_id: str
        :param request: Required.
        :type request: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: Invoice. The Invoice is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Invoice
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def simulate_invoice(
        self, customer_id: str, request: Union[_models.InvoiceSimulationInput, JSON, IO[bytes]], **kwargs: Any
    ) -> _models.Invoice:
        """Simulate an invoice for a customer.

        Simulate an invoice for a customer.

        This call will simulate an invoice for a customer based on the pending line items.

        The call will return the total amount of the invoice and the line items that will be included
        in the invoice.

        :param customer_id: Required.
        :type customer_id: str
        :param request: Is one of the following types: InvoiceSimulationInput, JSON, IO[bytes]
         Required.
        :type request: ~openmeter._generated.models.InvoiceSimulationInput or JSON or IO[bytes]
        :return: Invoice. The Invoice is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Invoice
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[_models.Invoice] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(request, (IOBase, bytes)):
            _content = request
        else:
            _content = json.dumps(request, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_billing_customer_invoice_endpoints_simulate_invoice_request(
            customer_id=customer_id,
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.Invoice, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    @overload
    async def create_pending_invoice_line(
        self,
        customer_id: str,
        request: _models.InvoicePendingLineCreateInput,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> _models.InvoicePendingLineCreateResponse:
        """Create pending line items.

        Create a new pending line item (charge).

        This call is used to create a new pending line item for the customer if required a new
        gathering invoice will be created.

        A new invoice will be created if:

        * there is no invoice in gathering state
        * the currency of the line item doesn't match the currency of any invoices in gathering state.

        :param customer_id: Required.
        :type customer_id: str
        :param request: Required.
        :type request: ~openmeter._generated.models.InvoicePendingLineCreateInput
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: InvoicePendingLineCreateResponse. The InvoicePendingLineCreateResponse is compatible
         with MutableMapping
        :rtype: ~openmeter._generated.models.InvoicePendingLineCreateResponse
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def create_pending_invoice_line(
        self, customer_id: str, request: JSON, *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.InvoicePendingLineCreateResponse:
        """Create pending line items.

        Create a new pending line item (charge).

        This call is used to create a new pending line item for the customer if required a new
        gathering invoice will be created.

        A new invoice will be created if:

        * there is no invoice in gathering state
        * the currency of the line item doesn't match the currency of any invoices in gathering state.

        :param customer_id: Required.
        :type customer_id: str
        :param request: Required.
        :type request: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: InvoicePendingLineCreateResponse. The InvoicePendingLineCreateResponse is compatible
         with MutableMapping
        :rtype: ~openmeter._generated.models.InvoicePendingLineCreateResponse
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def create_pending_invoice_line(
        self, customer_id: str, request: IO[bytes], *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.InvoicePendingLineCreateResponse:
        """Create pending line items.

        Create a new pending line item (charge).

        This call is used to create a new pending line item for the customer if required a new
        gathering invoice will be created.

        A new invoice will be created if:

        * there is no invoice in gathering state
        * the currency of the line item doesn't match the currency of any invoices in gathering state.

        :param customer_id: Required.
        :type customer_id: str
        :param request: Required.
        :type request: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: InvoicePendingLineCreateResponse. The InvoicePendingLineCreateResponse is compatible
         with MutableMapping
        :rtype: ~openmeter._generated.models.InvoicePendingLineCreateResponse
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def create_pending_invoice_line(
        self, customer_id: str, request: Union[_models.InvoicePendingLineCreateInput, JSON, IO[bytes]], **kwargs: Any
    ) -> _models.InvoicePendingLineCreateResponse:
        """Create pending line items.

        Create a new pending line item (charge).

        This call is used to create a new pending line item for the customer if required a new
        gathering invoice will be created.

        A new invoice will be created if:

        * there is no invoice in gathering state
        * the currency of the line item doesn't match the currency of any invoices in gathering state.

        :param customer_id: Required.
        :type customer_id: str
        :param request: Is one of the following types: InvoicePendingLineCreateInput, JSON, IO[bytes]
         Required.
        :type request: ~openmeter._generated.models.InvoicePendingLineCreateInput or JSON or IO[bytes]
        :return: InvoicePendingLineCreateResponse. The InvoicePendingLineCreateResponse is compatible
         with MutableMapping
        :rtype: ~openmeter._generated.models.InvoicePendingLineCreateResponse
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[_models.InvoicePendingLineCreateResponse] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(request, (IOBase, bytes)):
            _content = request
        else:
            _content = json.dumps(request, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_billing_customer_invoice_endpoints_create_pending_invoice_line_request(
            customer_id=customer_id,
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [201]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.InvoicePendingLineCreateResponse, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore


class PortalTokensOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`tokens` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

    @overload
    async def create(
        self, token: _models.PortalToken, *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.PortalToken:
        """Create consumer portal token.

        Create a consumer portal token.

        :param token: Required.
        :type token: ~openmeter._generated.models.PortalToken
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: PortalToken. The PortalToken is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.PortalToken
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def create(
        self, token: JSON, *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.PortalToken:
        """Create consumer portal token.

        Create a consumer portal token.

        :param token: Required.
        :type token: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: PortalToken. The PortalToken is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.PortalToken
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def create(
        self, token: IO[bytes], *, content_type: str = "application/json", **kwargs: Any
    ) -> _models.PortalToken:
        """Create consumer portal token.

        Create a consumer portal token.

        :param token: Required.
        :type token: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: PortalToken. The PortalToken is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.PortalToken
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def create(self, token: Union[_models.PortalToken, JSON, IO[bytes]], **kwargs: Any) -> _models.PortalToken:
        """Create consumer portal token.

        Create a consumer portal token.

        :param token: Is one of the following types: PortalToken, JSON, IO[bytes] Required.
        :type token: ~openmeter._generated.models.PortalToken or JSON or IO[bytes]
        :return: PortalToken. The PortalToken is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.PortalToken
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[_models.PortalToken] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(token, (IOBase, bytes)):
            _content = token
        else:
            _content = json.dumps(token, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_portal_tokens_create_request(
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.PortalToken, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def list(self, *, limit: Optional[int] = None, **kwargs: Any) -> List[_models.PortalToken]:
        """List consumer portal tokens.

        List tokens.

        :keyword limit: Default value is None.
        :paramtype limit: int
        :return: list of PortalToken
        :rtype: list[~openmeter._generated.models.PortalToken]
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[List[_models.PortalToken]] = kwargs.pop("cls", None)

        _request = build_portal_tokens_list_request(
            limit=limit,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(List[_models.PortalToken], response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    @overload
    async def invalidate(
        self,
        *,
        content_type: str = "application/json",
        id: Optional[str] = None,
        subject: Optional[str] = None,
        **kwargs: Any
    ) -> None:
        """Invalidate portal tokens.

        Invalidates consumer portal tokens by ID or subject.

        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :keyword id: Invalidate a portal token by ID. Default value is None.
        :paramtype id: str
        :keyword subject: Invalidate all portal tokens for a subject. Default value is None.
        :paramtype subject: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def invalidate(self, body: JSON, *, content_type: str = "application/json", **kwargs: Any) -> None:
        """Invalidate portal tokens.

        Invalidates consumer portal tokens by ID or subject.

        :param body: Required.
        :type body: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def invalidate(self, body: IO[bytes], *, content_type: str = "application/json", **kwargs: Any) -> None:
        """Invalidate portal tokens.

        Invalidates consumer portal tokens by ID or subject.

        :param body: Required.
        :type body: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def invalidate(
        self,
        body: Union[JSON, IO[bytes]] = _Unset,
        *,
        id: Optional[str] = None,
        subject: Optional[str] = None,
        **kwargs: Any
    ) -> None:
        """Invalidate portal tokens.

        Invalidates consumer portal tokens by ID or subject.

        :param body: Is either a JSON type or a IO[bytes] type. Required.
        :type body: JSON or IO[bytes]
        :keyword id: Invalidate a portal token by ID. Default value is None.
        :paramtype id: str
        :keyword subject: Invalidate all portal tokens for a subject. Default value is None.
        :paramtype subject: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[None] = kwargs.pop("cls", None)

        if body is _Unset:
            body = {"id": id, "subject": subject}
            body = {k: v for k, v in body.items() if v is not None}
        content_type = content_type or "application/json"
        _content = None
        if isinstance(body, (IOBase, bytes)):
            _content = body
        else:
            _content = json.dumps(body, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_portal_tokens_invalidate_request(
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = False
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [204]:
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if cls:
            return cls(pipeline_response, None, {})  # type: ignore


class PortalMetersOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`meters` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

    async def query_json(
        self,
        meter_slug: str,
        *,
        client_id: Optional[str] = None,
        from_parameter: Optional[datetime.datetime] = None,
        to: Optional[datetime.datetime] = None,
        window_size: Optional[Union[str, _models.WindowSize]] = None,
        window_time_zone: Optional[str] = None,
        filter_customer_id: Optional[List[str]] = None,
        filter_group_by: Optional[dict[str, str]] = None,
        advanced_meter_group_by_filters: Optional[dict[str, _models.FilterString]] = None,
        group_by: Optional[List[str]] = None,
        **kwargs: Any
    ) -> _models.MeterQueryResult:
        """Query meter.

        Query meter for consumer portal. This endpoint is publicly exposable to consumers.

        :param meter_slug: Required.
        :type meter_slug: str
        :keyword client_id: Client ID
         Useful to track progress of a query. Default value is None.
        :paramtype client_id: str
        :keyword from_parameter: Start date-time in RFC 3339 format.

         Inclusive.

         For example: ?from=2025-01-01T00%3A00%3A00.000Z. Default value is None.
        :paramtype from_parameter: ~datetime.datetime
        :keyword to: End date-time in RFC 3339 format.

         Inclusive.

         For example: ?to=2025-02-01T00%3A00%3A00.000Z. Default value is None.
        :paramtype to: ~datetime.datetime
        :keyword window_size: If not specified, a single usage aggregate will be returned for the
         entirety of the specified period for each subject and group.

         For example: ?windowSize=DAY. Known values are: "MINUTE", "HOUR", "DAY", and "MONTH". Default
         value is None.
        :paramtype window_size: str or ~openmeter.models.WindowSize
        :keyword window_time_zone: The value is the name of the time zone as defined in the IANA Time
         Zone Database (`http://www.iana.org/time-zones <http://www.iana.org/time-zones>`_).
         If not specified, the UTC timezone will be used.

         For example: ?windowTimeZone=UTC. Default value is None.
        :paramtype window_time_zone: str
        :keyword filter_customer_id: Filtering by multiple customers.

         For example: ?filterCustomerId=customer-1&filterCustomerId=customer-2. Default value is None.
        :paramtype filter_customer_id: list[str]
        :keyword filter_group_by: Simple filter for group bys with exact match.

         For example: ?filterGroupBy[vendor]=openai&filterGroupBy[model]=gpt-4-turbo. Default value is
         None.
        :paramtype filter_group_by: dict[str, str]
        :keyword advanced_meter_group_by_filters: Advanced meter group by filters. Default value is
         None.
        :paramtype advanced_meter_group_by_filters: dict[str,
         ~openmeter._generated.models.FilterString]
        :keyword group_by: If not specified a single aggregate will be returned for each subject and
         time window.
         ``subject`` is a reserved group by value.

         For example: ?groupBy=subject&groupBy=model. Default value is None.
        :paramtype group_by: list[str]
        :return: MeterQueryResult. The MeterQueryResult is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.MeterQueryResult
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.MeterQueryResult] = kwargs.pop("cls", None)

        _request = build_portal_meters_query_json_request(
            meter_slug=meter_slug,
            client_id=client_id,
            from_parameter=from_parameter,
            to=to,
            window_size=window_size,
            window_time_zone=window_time_zone,
            filter_customer_id=filter_customer_id,
            filter_group_by=filter_group_by,
            advanced_meter_group_by_filters=advanced_meter_group_by_filters,
            group_by=group_by,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        response_headers = {}
        response_headers["content-type"] = self._deserialize("str", response.headers.get("content-type"))

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.MeterQueryResult, response.json())

        if cls:
            return cls(pipeline_response, deserialized, response_headers)  # type: ignore

        return deserialized  # type: ignore

    async def query_csv(
        self,
        meter_slug: str,
        *,
        client_id: Optional[str] = None,
        from_parameter: Optional[datetime.datetime] = None,
        to: Optional[datetime.datetime] = None,
        window_size: Optional[Union[str, _models.WindowSize]] = None,
        window_time_zone: Optional[str] = None,
        filter_customer_id: Optional[List[str]] = None,
        filter_group_by: Optional[dict[str, str]] = None,
        advanced_meter_group_by_filters: Optional[dict[str, _models.FilterString]] = None,
        group_by: Optional[List[str]] = None,
        **kwargs: Any
    ) -> str:
        """Query meter.

        Query meter for consumer portal. This endpoint is publicly exposable to consumers.

        :param meter_slug: Required.
        :type meter_slug: str
        :keyword client_id: Client ID
         Useful to track progress of a query. Default value is None.
        :paramtype client_id: str
        :keyword from_parameter: Start date-time in RFC 3339 format.

         Inclusive.

         For example: ?from=2025-01-01T00%3A00%3A00.000Z. Default value is None.
        :paramtype from_parameter: ~datetime.datetime
        :keyword to: End date-time in RFC 3339 format.

         Inclusive.

         For example: ?to=2025-02-01T00%3A00%3A00.000Z. Default value is None.
        :paramtype to: ~datetime.datetime
        :keyword window_size: If not specified, a single usage aggregate will be returned for the
         entirety of the specified period for each subject and group.

         For example: ?windowSize=DAY. Known values are: "MINUTE", "HOUR", "DAY", and "MONTH". Default
         value is None.
        :paramtype window_size: str or ~openmeter.models.WindowSize
        :keyword window_time_zone: The value is the name of the time zone as defined in the IANA Time
         Zone Database (`http://www.iana.org/time-zones <http://www.iana.org/time-zones>`_).
         If not specified, the UTC timezone will be used.

         For example: ?windowTimeZone=UTC. Default value is None.
        :paramtype window_time_zone: str
        :keyword filter_customer_id: Filtering by multiple customers.

         For example: ?filterCustomerId=customer-1&filterCustomerId=customer-2. Default value is None.
        :paramtype filter_customer_id: list[str]
        :keyword filter_group_by: Simple filter for group bys with exact match.

         For example: ?filterGroupBy[vendor]=openai&filterGroupBy[model]=gpt-4-turbo. Default value is
         None.
        :paramtype filter_group_by: dict[str, str]
        :keyword advanced_meter_group_by_filters: Advanced meter group by filters. Default value is
         None.
        :paramtype advanced_meter_group_by_filters: dict[str,
         ~openmeter._generated.models.FilterString]
        :keyword group_by: If not specified a single aggregate will be returned for each subject and
         time window.
         ``subject`` is a reserved group by value.

         For example: ?groupBy=subject&groupBy=model. Default value is None.
        :paramtype group_by: list[str]
        :return: str
        :rtype: str
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[str] = kwargs.pop("cls", None)

        _request = build_portal_meters_query_csv_request(
            meter_slug=meter_slug,
            client_id=client_id,
            from_parameter=from_parameter,
            to=to,
            window_size=window_size,
            window_time_zone=window_time_zone,
            filter_customer_id=filter_customer_id,
            filter_group_by=filter_group_by,
            advanced_meter_group_by_filters=advanced_meter_group_by_filters,
            group_by=group_by,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        response_headers = {}
        response_headers["content-type"] = self._deserialize("str", response.headers.get("content-type"))

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(str, response.text())

        if cls:
            return cls(pipeline_response, deserialized, response_headers)  # type: ignore

        return deserialized  # type: ignore


class NotificationChannelsOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`channels` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

    async def list(
        self,
        *,
        include_deleted: Optional[bool] = None,
        include_disabled: Optional[bool] = None,
        page: Optional[int] = None,
        page_size: Optional[int] = None,
        order: Optional[Union[str, _models.SortOrder]] = None,
        order_by: Optional[Union[str, _models.NotificationChannelOrderBy]] = None,
        **kwargs: Any
    ) -> _models.NotificationChannelPaginatedResponse:
        """List notification channels.

        List all notification channels.

        :keyword include_deleted: Include deleted notification channels in response.

         Usage: ``?includeDeleted=true``. Default value is None.
        :paramtype include_deleted: bool
        :keyword include_disabled: Include disabled notification channels in response.

         Usage: ``?includeDisabled=false``. Default value is None.
        :paramtype include_disabled: bool
        :keyword page: Page index.

         Default is 1. Default value is None.
        :paramtype page: int
        :keyword page_size: The maximum number of items per page.

         Default is 100. Default value is None.
        :paramtype page_size: int
        :keyword order: The order direction. Known values are: "ASC" and "DESC". Default value is None.
        :paramtype order: str or ~openmeter.models.SortOrder
        :keyword order_by: The order by field. Known values are: "id", "type", "createdAt", and
         "updatedAt". Default value is None.
        :paramtype order_by: str or ~openmeter.models.NotificationChannelOrderBy
        :return: NotificationChannelPaginatedResponse. The NotificationChannelPaginatedResponse is
         compatible with MutableMapping
        :rtype: ~openmeter._generated.models.NotificationChannelPaginatedResponse
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.NotificationChannelPaginatedResponse] = kwargs.pop("cls", None)

        _request = build_notification_channels_list_request(
            include_deleted=include_deleted,
            include_disabled=include_disabled,
            page=page,
            page_size=page_size,
            order=order,
            order_by=order_by,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.NotificationChannelPaginatedResponse, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    @overload
    async def create(
        self,
        request: _models.NotificationChannelWebhookCreateRequest,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> "_types.NotificationChannel":
        """Create a notification channel.

        Create a new notification channel.

        :param request: Required.
        :type request: ~openmeter._generated.models.NotificationChannelWebhookCreateRequest
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: NotificationChannelWebhook
        :rtype: ~openmeter._generated.models.NotificationChannelWebhook
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def create(
        self, request: "_types.NotificationChannelCreateRequest", **kwargs: Any
    ) -> "_types.NotificationChannel":
        """Create a notification channel.

        Create a new notification channel.

        :param request: Is one of the following types: NotificationChannelWebhookCreateRequest
         Required.
        :type request: ~openmeter._generated.models.NotificationChannelWebhookCreateRequest
        :return: NotificationChannelWebhook
        :rtype: ~openmeter._generated.models.NotificationChannelWebhook
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType["_types.NotificationChannel"] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(request, _models.NotificationChannelWebhookCreateRequest):
            _content = json.dumps(request, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_notification_channels_create_request(
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [201]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize("_types.NotificationChannel", response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    @overload
    async def update(
        self,
        channel_id: str,
        request: _models.NotificationChannelWebhookCreateRequest,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> "_types.NotificationChannel":
        """Update a notification channel.

        Update notification channel.

        :param channel_id: Required.
        :type channel_id: str
        :param request: Required.
        :type request: ~openmeter._generated.models.NotificationChannelWebhookCreateRequest
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: NotificationChannelWebhook
        :rtype: ~openmeter._generated.models.NotificationChannelWebhook
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def update(
        self, channel_id: str, request: "_types.NotificationChannelCreateRequest", **kwargs: Any
    ) -> "_types.NotificationChannel":
        """Update a notification channel.

        Update notification channel.

        :param channel_id: Required.
        :type channel_id: str
        :param request: Is one of the following types: NotificationChannelWebhookCreateRequest
         Required.
        :type request: ~openmeter._generated.models.NotificationChannelWebhookCreateRequest
        :return: NotificationChannelWebhook
        :rtype: ~openmeter._generated.models.NotificationChannelWebhook
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType["_types.NotificationChannel"] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(request, _models.NotificationChannelWebhookCreateRequest):
            _content = json.dumps(request, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_notification_channels_update_request(
            channel_id=channel_id,
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize("_types.NotificationChannel", response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def get(self, channel_id: str, **kwargs: Any) -> "_types.NotificationChannel":
        """Get notification channel.

        Get a notification channel by id.

        :param channel_id: Required.
        :type channel_id: str
        :return: NotificationChannelWebhook
        :rtype: ~openmeter._generated.models.NotificationChannelWebhook
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType["_types.NotificationChannel"] = kwargs.pop("cls", None)

        _request = build_notification_channels_get_request(
            channel_id=channel_id,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize("_types.NotificationChannel", response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def delete(self, channel_id: str, **kwargs: Any) -> None:
        """Delete a notification channel.

        Soft delete notification channel by id.

        Once a notification channel is deleted it cannot be undeleted.

        :param channel_id: Required.
        :type channel_id: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[None] = kwargs.pop("cls", None)

        _request = build_notification_channels_delete_request(
            channel_id=channel_id,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = False
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [204]:
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if cls:
            return cls(pipeline_response, None, {})  # type: ignore


class NotificationRulesOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`rules` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

    async def list(
        self,
        *,
        include_deleted: Optional[bool] = None,
        include_disabled: Optional[bool] = None,
        feature: Optional[List[str]] = None,
        channel: Optional[List[str]] = None,
        page: Optional[int] = None,
        page_size: Optional[int] = None,
        order: Optional[Union[str, _models.SortOrder]] = None,
        order_by: Optional[Union[str, _models.NotificationRuleOrderBy]] = None,
        **kwargs: Any
    ) -> _models.NotificationRulePaginatedResponse:
        """List notification rules.

        List all notification rules.

        :keyword include_deleted: Include deleted notification rules in response.

         Usage: ``?includeDeleted=true``. Default value is None.
        :paramtype include_deleted: bool
        :keyword include_disabled: Include disabled notification rules in response.

         Usage: ``?includeDisabled=false``. Default value is None.
        :paramtype include_disabled: bool
        :keyword feature: Filtering by multiple feature ids/keys.

         Usage: ``?feature=feature-1&feature=feature-2``. Default value is None.
        :paramtype feature: list[str]
        :keyword channel: Filtering by multiple notifiaction channel ids.

         Usage: ``?channel=01ARZ3NDEKTSV4RRFFQ69G5FAV&channel=01J8J2Y5X4NNGQS32CF81W95E3``. Default
         value is None.
        :paramtype channel: list[str]
        :keyword page: Page index.

         Default is 1. Default value is None.
        :paramtype page: int
        :keyword page_size: The maximum number of items per page.

         Default is 100. Default value is None.
        :paramtype page_size: int
        :keyword order: The order direction. Known values are: "ASC" and "DESC". Default value is None.
        :paramtype order: str or ~openmeter.models.SortOrder
        :keyword order_by: The order by field. Known values are: "id", "type", "createdAt", and
         "updatedAt". Default value is None.
        :paramtype order_by: str or ~openmeter.models.NotificationRuleOrderBy
        :return: NotificationRulePaginatedResponse. The NotificationRulePaginatedResponse is compatible
         with MutableMapping
        :rtype: ~openmeter._generated.models.NotificationRulePaginatedResponse
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.NotificationRulePaginatedResponse] = kwargs.pop("cls", None)

        _request = build_notification_rules_list_request(
            include_deleted=include_deleted,
            include_disabled=include_disabled,
            feature=feature,
            channel=channel,
            page=page,
            page_size=page_size,
            order=order,
            order_by=order_by,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.NotificationRulePaginatedResponse, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    @overload
    async def create(
        self,
        request: _models.NotificationRuleBalanceThresholdCreateRequest,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> "_types.NotificationRule":
        """Create a notification rule.

        Create a new notification rule.

        :param request: Required.
        :type request: ~openmeter._generated.models.NotificationRuleBalanceThresholdCreateRequest
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: NotificationRuleBalanceThreshold or NotificationRuleEntitlementReset or
         NotificationRuleInvoiceCreated or NotificationRuleInvoiceUpdated
        :rtype: ~openmeter._generated.models.NotificationRuleBalanceThreshold or
         ~openmeter._generated.models.NotificationRuleEntitlementReset or
         ~openmeter._generated.models.NotificationRuleInvoiceCreated or
         ~openmeter._generated.models.NotificationRuleInvoiceUpdated
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def create(
        self,
        request: _models.NotificationRuleEntitlementResetCreateRequest,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> "_types.NotificationRule":
        """Create a notification rule.

        Create a new notification rule.

        :param request: Required.
        :type request: ~openmeter._generated.models.NotificationRuleEntitlementResetCreateRequest
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: NotificationRuleBalanceThreshold or NotificationRuleEntitlementReset or
         NotificationRuleInvoiceCreated or NotificationRuleInvoiceUpdated
        :rtype: ~openmeter._generated.models.NotificationRuleBalanceThreshold or
         ~openmeter._generated.models.NotificationRuleEntitlementReset or
         ~openmeter._generated.models.NotificationRuleInvoiceCreated or
         ~openmeter._generated.models.NotificationRuleInvoiceUpdated
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def create(
        self,
        request: _models.NotificationRuleInvoiceCreatedCreateRequest,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> "_types.NotificationRule":
        """Create a notification rule.

        Create a new notification rule.

        :param request: Required.
        :type request: ~openmeter._generated.models.NotificationRuleInvoiceCreatedCreateRequest
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: NotificationRuleBalanceThreshold or NotificationRuleEntitlementReset or
         NotificationRuleInvoiceCreated or NotificationRuleInvoiceUpdated
        :rtype: ~openmeter._generated.models.NotificationRuleBalanceThreshold or
         ~openmeter._generated.models.NotificationRuleEntitlementReset or
         ~openmeter._generated.models.NotificationRuleInvoiceCreated or
         ~openmeter._generated.models.NotificationRuleInvoiceUpdated
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def create(
        self,
        request: _models.NotificationRuleInvoiceUpdatedCreateRequest,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> "_types.NotificationRule":
        """Create a notification rule.

        Create a new notification rule.

        :param request: Required.
        :type request: ~openmeter._generated.models.NotificationRuleInvoiceUpdatedCreateRequest
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: NotificationRuleBalanceThreshold or NotificationRuleEntitlementReset or
         NotificationRuleInvoiceCreated or NotificationRuleInvoiceUpdated
        :rtype: ~openmeter._generated.models.NotificationRuleBalanceThreshold or
         ~openmeter._generated.models.NotificationRuleEntitlementReset or
         ~openmeter._generated.models.NotificationRuleInvoiceCreated or
         ~openmeter._generated.models.NotificationRuleInvoiceUpdated
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def create(self, request: "_types.NotificationRuleCreateRequest", **kwargs: Any) -> "_types.NotificationRule":
        """Create a notification rule.

        Create a new notification rule.

        :param request: Is one of the following types: NotificationRuleBalanceThresholdCreateRequest,
         NotificationRuleEntitlementResetCreateRequest, NotificationRuleInvoiceCreatedCreateRequest,
         NotificationRuleInvoiceUpdatedCreateRequest Required.
        :type request: ~openmeter._generated.models.NotificationRuleBalanceThresholdCreateRequest or
         ~openmeter._generated.models.NotificationRuleEntitlementResetCreateRequest or
         ~openmeter._generated.models.NotificationRuleInvoiceCreatedCreateRequest or
         ~openmeter._generated.models.NotificationRuleInvoiceUpdatedCreateRequest
        :return: NotificationRuleBalanceThreshold or NotificationRuleEntitlementReset or
         NotificationRuleInvoiceCreated or NotificationRuleInvoiceUpdated
        :rtype: ~openmeter._generated.models.NotificationRuleBalanceThreshold or
         ~openmeter._generated.models.NotificationRuleEntitlementReset or
         ~openmeter._generated.models.NotificationRuleInvoiceCreated or
         ~openmeter._generated.models.NotificationRuleInvoiceUpdated
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType["_types.NotificationRule"] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(request, _models.NotificationRuleBalanceThresholdCreateRequest):
            _content = json.dumps(request, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore
        elif isinstance(request, _models.NotificationRuleEntitlementResetCreateRequest):
            _content = json.dumps(request, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore
        elif isinstance(request, _models.NotificationRuleInvoiceCreatedCreateRequest):
            _content = json.dumps(request, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore
        elif isinstance(request, _models.NotificationRuleInvoiceUpdatedCreateRequest):
            _content = json.dumps(request, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_notification_rules_create_request(
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [201]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize("_types.NotificationRule", response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    @overload
    async def update(
        self,
        rule_id: str,
        request: _models.NotificationRuleBalanceThresholdCreateRequest,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> "_types.NotificationRule":
        """Update a notification rule.

        Update notification rule.

        :param rule_id: Required.
        :type rule_id: str
        :param request: Required.
        :type request: ~openmeter._generated.models.NotificationRuleBalanceThresholdCreateRequest
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: NotificationRuleBalanceThreshold or NotificationRuleEntitlementReset or
         NotificationRuleInvoiceCreated or NotificationRuleInvoiceUpdated
        :rtype: ~openmeter._generated.models.NotificationRuleBalanceThreshold or
         ~openmeter._generated.models.NotificationRuleEntitlementReset or
         ~openmeter._generated.models.NotificationRuleInvoiceCreated or
         ~openmeter._generated.models.NotificationRuleInvoiceUpdated
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def update(
        self,
        rule_id: str,
        request: _models.NotificationRuleEntitlementResetCreateRequest,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> "_types.NotificationRule":
        """Update a notification rule.

        Update notification rule.

        :param rule_id: Required.
        :type rule_id: str
        :param request: Required.
        :type request: ~openmeter._generated.models.NotificationRuleEntitlementResetCreateRequest
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: NotificationRuleBalanceThreshold or NotificationRuleEntitlementReset or
         NotificationRuleInvoiceCreated or NotificationRuleInvoiceUpdated
        :rtype: ~openmeter._generated.models.NotificationRuleBalanceThreshold or
         ~openmeter._generated.models.NotificationRuleEntitlementReset or
         ~openmeter._generated.models.NotificationRuleInvoiceCreated or
         ~openmeter._generated.models.NotificationRuleInvoiceUpdated
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def update(
        self,
        rule_id: str,
        request: _models.NotificationRuleInvoiceCreatedCreateRequest,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> "_types.NotificationRule":
        """Update a notification rule.

        Update notification rule.

        :param rule_id: Required.
        :type rule_id: str
        :param request: Required.
        :type request: ~openmeter._generated.models.NotificationRuleInvoiceCreatedCreateRequest
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: NotificationRuleBalanceThreshold or NotificationRuleEntitlementReset or
         NotificationRuleInvoiceCreated or NotificationRuleInvoiceUpdated
        :rtype: ~openmeter._generated.models.NotificationRuleBalanceThreshold or
         ~openmeter._generated.models.NotificationRuleEntitlementReset or
         ~openmeter._generated.models.NotificationRuleInvoiceCreated or
         ~openmeter._generated.models.NotificationRuleInvoiceUpdated
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def update(
        self,
        rule_id: str,
        request: _models.NotificationRuleInvoiceUpdatedCreateRequest,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> "_types.NotificationRule":
        """Update a notification rule.

        Update notification rule.

        :param rule_id: Required.
        :type rule_id: str
        :param request: Required.
        :type request: ~openmeter._generated.models.NotificationRuleInvoiceUpdatedCreateRequest
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: NotificationRuleBalanceThreshold or NotificationRuleEntitlementReset or
         NotificationRuleInvoiceCreated or NotificationRuleInvoiceUpdated
        :rtype: ~openmeter._generated.models.NotificationRuleBalanceThreshold or
         ~openmeter._generated.models.NotificationRuleEntitlementReset or
         ~openmeter._generated.models.NotificationRuleInvoiceCreated or
         ~openmeter._generated.models.NotificationRuleInvoiceUpdated
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def update(
        self, rule_id: str, request: "_types.NotificationRuleCreateRequest", **kwargs: Any
    ) -> "_types.NotificationRule":
        """Update a notification rule.

        Update notification rule.

        :param rule_id: Required.
        :type rule_id: str
        :param request: Is one of the following types: NotificationRuleBalanceThresholdCreateRequest,
         NotificationRuleEntitlementResetCreateRequest, NotificationRuleInvoiceCreatedCreateRequest,
         NotificationRuleInvoiceUpdatedCreateRequest Required.
        :type request: ~openmeter._generated.models.NotificationRuleBalanceThresholdCreateRequest or
         ~openmeter._generated.models.NotificationRuleEntitlementResetCreateRequest or
         ~openmeter._generated.models.NotificationRuleInvoiceCreatedCreateRequest or
         ~openmeter._generated.models.NotificationRuleInvoiceUpdatedCreateRequest
        :return: NotificationRuleBalanceThreshold or NotificationRuleEntitlementReset or
         NotificationRuleInvoiceCreated or NotificationRuleInvoiceUpdated
        :rtype: ~openmeter._generated.models.NotificationRuleBalanceThreshold or
         ~openmeter._generated.models.NotificationRuleEntitlementReset or
         ~openmeter._generated.models.NotificationRuleInvoiceCreated or
         ~openmeter._generated.models.NotificationRuleInvoiceUpdated
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType["_types.NotificationRule"] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(request, _models.NotificationRuleBalanceThresholdCreateRequest):
            _content = json.dumps(request, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore
        elif isinstance(request, _models.NotificationRuleEntitlementResetCreateRequest):
            _content = json.dumps(request, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore
        elif isinstance(request, _models.NotificationRuleInvoiceCreatedCreateRequest):
            _content = json.dumps(request, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore
        elif isinstance(request, _models.NotificationRuleInvoiceUpdatedCreateRequest):
            _content = json.dumps(request, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_notification_rules_update_request(
            rule_id=rule_id,
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize("_types.NotificationRule", response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def get(self, rule_id: str, **kwargs: Any) -> "_types.NotificationRule":
        """Get notification rule.

        Get a notification rule by id.

        :param rule_id: Required.
        :type rule_id: str
        :return: NotificationRuleBalanceThreshold or NotificationRuleEntitlementReset or
         NotificationRuleInvoiceCreated or NotificationRuleInvoiceUpdated
        :rtype: ~openmeter._generated.models.NotificationRuleBalanceThreshold or
         ~openmeter._generated.models.NotificationRuleEntitlementReset or
         ~openmeter._generated.models.NotificationRuleInvoiceCreated or
         ~openmeter._generated.models.NotificationRuleInvoiceUpdated
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType["_types.NotificationRule"] = kwargs.pop("cls", None)

        _request = build_notification_rules_get_request(
            rule_id=rule_id,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize("_types.NotificationRule", response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def delete(self, rule_id: str, **kwargs: Any) -> None:
        """Delete a notification rule.

        Soft delete notification rule by id.

        Once a notification rule is deleted it cannot be undeleted.

        :param rule_id: Required.
        :type rule_id: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[None] = kwargs.pop("cls", None)

        _request = build_notification_rules_delete_request(
            rule_id=rule_id,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = False
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [204]:
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if cls:
            return cls(pipeline_response, None, {})  # type: ignore

    async def test(self, rule_id: str, **kwargs: Any) -> _models.NotificationEvent:
        """Test notification rule.

        Test a notification rule by sending a test event with random data.

        :param rule_id: Required.
        :type rule_id: str
        :return: NotificationEvent. The NotificationEvent is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.NotificationEvent
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.NotificationEvent] = kwargs.pop("cls", None)

        _request = build_notification_rules_test_request(
            rule_id=rule_id,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [201]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.NotificationEvent, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore


class NotificationEventsOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`events` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

    async def list(
        self,
        *,
        from_parameter: Optional[datetime.datetime] = None,
        to: Optional[datetime.datetime] = None,
        feature: Optional[List[str]] = None,
        subject: Optional[List[str]] = None,
        rule: Optional[List[str]] = None,
        channel: Optional[List[str]] = None,
        page: Optional[int] = None,
        page_size: Optional[int] = None,
        order: Optional[Union[str, _models.SortOrder]] = None,
        order_by: Optional[Union[str, _models.NotificationEventOrderBy]] = None,
        **kwargs: Any
    ) -> _models.NotificationEventPaginatedResponse:
        """List notification events.

        List all notification events.

        :keyword from_parameter: Start date-time in RFC 3339 format.
         Inclusive. Default value is None.
        :paramtype from_parameter: ~datetime.datetime
        :keyword to: End date-time in RFC 3339 format.
         Inclusive. Default value is None.
        :paramtype to: ~datetime.datetime
        :keyword feature: Filtering by multiple feature ids or keys.

         Usage: ``?feature=feature-1&feature=feature-2``. Default value is None.
        :paramtype feature: list[str]
        :keyword subject: Filtering by multiple subject ids or keys.

         Usage: ``?subject=subject-1&subject=subject-2``. Default value is None.
        :paramtype subject: list[str]
        :keyword rule: Filtering by multiple rule ids.

         Usage: ``?rule=01J8J2XYZ2N5WBYK09EDZFBSZM&rule=01J8J4R4VZH180KRKQ63NB2VA5``. Default value is
         None.
        :paramtype rule: list[str]
        :keyword channel: Filtering by multiple channel ids.

         Usage: ``?channel=01J8J4RXH778XB056JS088PCYT&channel=01J8J4S1R1G9EVN62RG23A9M6J``. Default
         value is None.
        :paramtype channel: list[str]
        :keyword page: Page index.

         Default is 1. Default value is None.
        :paramtype page: int
        :keyword page_size: The maximum number of items per page.

         Default is 100. Default value is None.
        :paramtype page_size: int
        :keyword order: The order direction. Known values are: "ASC" and "DESC". Default value is None.
        :paramtype order: str or ~openmeter.models.SortOrder
        :keyword order_by: The order by field. Known values are: "id" and "createdAt". Default value is
         None.
        :paramtype order_by: str or ~openmeter.models.NotificationEventOrderBy
        :return: NotificationEventPaginatedResponse. The NotificationEventPaginatedResponse is
         compatible with MutableMapping
        :rtype: ~openmeter._generated.models.NotificationEventPaginatedResponse
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.NotificationEventPaginatedResponse] = kwargs.pop("cls", None)

        _request = build_notification_events_list_request(
            from_parameter=from_parameter,
            to=to,
            feature=feature,
            subject=subject,
            rule=rule,
            channel=channel,
            page=page,
            page_size=page_size,
            order=order,
            order_by=order_by,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.NotificationEventPaginatedResponse, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def get(self, event_id: str, **kwargs: Any) -> _models.NotificationEvent:
        """Get notification event.

        Get a notification event by id.

        :param event_id: Required.
        :type event_id: str
        :return: NotificationEvent. The NotificationEvent is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.NotificationEvent
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.NotificationEvent] = kwargs.pop("cls", None)

        _request = build_notification_events_get_request(
            event_id=event_id,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.NotificationEvent, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore


class InfoProgressesOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`progresses` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

    async def get_progress(self, id: str, **kwargs: Any) -> _models.Progress:
        """Get progress.

        Get progress.

        :param id: Required.
        :type id: str
        :return: Progress. The Progress is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.Progress
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.Progress] = kwargs.pop("cls", None)

        _request = build_info_progresses_get_progress_request(
            id=id,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.Progress, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore


class InfoCurrenciesOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`currencies` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

    async def list_currencies(self, **kwargs: Any) -> List[_models.Currency]:
        """List supported currencies.

        List all supported currencies.

        :return: list of Currency
        :rtype: list[~openmeter._generated.models.Currency]
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[List[_models.Currency]] = kwargs.pop("cls", None)

        _request = build_info_currencies_list_currencies_request(
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(List[_models.Currency], response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore


class EntitlementsV2EntitlementsOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`entitlements` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

    async def list(
        self,
        *,
        feature: Optional[List[str]] = None,
        customer_keys: Optional[List[str]] = None,
        customer_ids: Optional[List[str]] = None,
        entitlement_type: Optional[List[Union[str, _models.EntitlementType]]] = None,
        exclude_inactive: Optional[bool] = None,
        page: Optional[int] = None,
        page_size: Optional[int] = None,
        offset: Optional[int] = None,
        limit: Optional[int] = None,
        order: Optional[Union[str, _models.SortOrder]] = None,
        order_by: Optional[Union[str, _models.EntitlementOrderBy]] = None,
        **kwargs: Any
    ) -> _models.EntitlementV2PaginatedResponse:
        """List all entitlements.

        List all entitlements for all the customers and features. This endpoint is intended for
        administrative purposes only.
        To fetch the entitlements of a specific subject please use the
        /api/v2/customers/{customerIdOrKey}/entitlements endpoint.

        :keyword feature: Filtering by multiple features.

         Usage: ``?feature=feature-1&feature=feature-2``. Default value is None.
        :paramtype feature: list[str]
        :keyword customer_keys: Filtering by multiple customers.

         Usage: ``?customerKeys=customer-1&customerKeys=customer-3``. Default value is None.
        :paramtype customer_keys: list[str]
        :keyword customer_ids: Filtering by multiple customers.

         Usage: ``?customerIds=01K4WAQ0J99ZZ0MD75HXR112H8&customerIds=01K4WAQ0J99ZZ0MD75HXR112H9``.
         Default value is None.
        :paramtype customer_ids: list[str]
        :keyword entitlement_type: Filtering by multiple entitlement types.

         Usage: ``?entitlementType=metered&entitlementType=boolean``. Default value is None.
        :paramtype entitlement_type: list[str or ~openmeter.models.EntitlementType]
        :keyword exclude_inactive: Exclude inactive entitlements in the response (those scheduled for
         later or earlier). Default value is None.
        :paramtype exclude_inactive: bool
        :keyword page: Page index.

         Default is 1. Default value is None.
        :paramtype page: int
        :keyword page_size: The maximum number of items per page.

         Default is 100. Default value is None.
        :paramtype page_size: int
        :keyword offset: Number of items to skip.

         Default is 0. Default value is None.
        :paramtype offset: int
        :keyword limit: Number of items to return.

         Default is 100. Default value is None.
        :paramtype limit: int
        :keyword order: The order direction. Known values are: "ASC" and "DESC". Default value is None.
        :paramtype order: str or ~openmeter.models.SortOrder
        :keyword order_by: The order by field. Known values are: "createdAt" and "updatedAt". Default
         value is None.
        :paramtype order_by: str or ~openmeter.models.EntitlementOrderBy
        :return: EntitlementV2PaginatedResponse. The EntitlementV2PaginatedResponse is compatible with
         MutableMapping
        :rtype: ~openmeter._generated.models.EntitlementV2PaginatedResponse
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.EntitlementV2PaginatedResponse] = kwargs.pop("cls", None)

        _request = build_entitlements_v2_entitlements_list_request(
            feature=feature,
            customer_keys=customer_keys,
            customer_ids=customer_ids,
            entitlement_type=entitlement_type,
            exclude_inactive=exclude_inactive,
            page=page,
            page_size=page_size,
            offset=offset,
            limit=limit,
            order=order,
            order_by=order_by,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.EntitlementV2PaginatedResponse, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def get(self, entitlement_id: str, **kwargs: Any) -> "_types.EntitlementV2":
        """Get entitlement by id.

        Get entitlement by id.

        :param entitlement_id: Required.
        :type entitlement_id: str
        :return: EntitlementMeteredV2 or EntitlementStaticV2 or EntitlementBooleanV2
        :rtype: ~openmeter._generated.models.EntitlementMeteredV2 or
         ~openmeter._generated.models.EntitlementStaticV2 or
         ~openmeter._generated.models.EntitlementBooleanV2
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType["_types.EntitlementV2"] = kwargs.pop("cls", None)

        _request = build_entitlements_v2_entitlements_get_request(
            entitlement_id=entitlement_id,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize("_types.EntitlementV2", response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore


class EntitlementsV2CustomerEntitlementsOperations:  # pylint: disable=name-too-long
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`customer_entitlements` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

    @overload
    async def post(
        self,
        customer_id_or_key: "_types.ULIDOrExternalKey",
        entitlement: _models.EntitlementMeteredV2CreateInputs,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> "_types.EntitlementV2":
        """Create a customer entitlement.

        OpenMeter has three types of entitlements: metered, boolean, and static. The type property
        determines the type of entitlement. The underlying feature has to be compatible with the
        entitlement type specified in the request (e.g., a metered entitlement needs a feature
        associated with a meter).



        * Boolean entitlements define static feature access, e.g. "Can use SSO authentication".
        * Static entitlements let you pass along a configuration while granting access, e.g. "Using
        this feature with X Y settings" (passed in the config).
        * Metered entitlements have many use cases, from setting up usage-based access to implementing
        complex credit systems.  Example: The customer can use 10000 AI tokens during the usage period
        of the entitlement.

        A given customer can only have one active (non-deleted) entitlement per featureKey. If you try
        to create a new entitlement for a featureKey that already has an active entitlement, the
        request will fail with a 409 error.

        Once an entitlement is created you cannot modify it, only delete it.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :param entitlement: Required.
        :type entitlement: ~openmeter._generated.models.EntitlementMeteredV2CreateInputs
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: EntitlementMeteredV2 or EntitlementStaticV2 or EntitlementBooleanV2
        :rtype: ~openmeter._generated.models.EntitlementMeteredV2 or
         ~openmeter._generated.models.EntitlementStaticV2 or
         ~openmeter._generated.models.EntitlementBooleanV2
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def post(
        self,
        customer_id_or_key: "_types.ULIDOrExternalKey",
        entitlement: _models.EntitlementStaticCreateInputs,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> "_types.EntitlementV2":
        """Create a customer entitlement.

        OpenMeter has three types of entitlements: metered, boolean, and static. The type property
        determines the type of entitlement. The underlying feature has to be compatible with the
        entitlement type specified in the request (e.g., a metered entitlement needs a feature
        associated with a meter).



        * Boolean entitlements define static feature access, e.g. "Can use SSO authentication".
        * Static entitlements let you pass along a configuration while granting access, e.g. "Using
        this feature with X Y settings" (passed in the config).
        * Metered entitlements have many use cases, from setting up usage-based access to implementing
        complex credit systems.  Example: The customer can use 10000 AI tokens during the usage period
        of the entitlement.

        A given customer can only have one active (non-deleted) entitlement per featureKey. If you try
        to create a new entitlement for a featureKey that already has an active entitlement, the
        request will fail with a 409 error.

        Once an entitlement is created you cannot modify it, only delete it.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :param entitlement: Required.
        :type entitlement: ~openmeter._generated.models.EntitlementStaticCreateInputs
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: EntitlementMeteredV2 or EntitlementStaticV2 or EntitlementBooleanV2
        :rtype: ~openmeter._generated.models.EntitlementMeteredV2 or
         ~openmeter._generated.models.EntitlementStaticV2 or
         ~openmeter._generated.models.EntitlementBooleanV2
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def post(
        self,
        customer_id_or_key: "_types.ULIDOrExternalKey",
        entitlement: _models.EntitlementBooleanCreateInputs,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> "_types.EntitlementV2":
        """Create a customer entitlement.

        OpenMeter has three types of entitlements: metered, boolean, and static. The type property
        determines the type of entitlement. The underlying feature has to be compatible with the
        entitlement type specified in the request (e.g., a metered entitlement needs a feature
        associated with a meter).



        * Boolean entitlements define static feature access, e.g. "Can use SSO authentication".
        * Static entitlements let you pass along a configuration while granting access, e.g. "Using
        this feature with X Y settings" (passed in the config).
        * Metered entitlements have many use cases, from setting up usage-based access to implementing
        complex credit systems.  Example: The customer can use 10000 AI tokens during the usage period
        of the entitlement.

        A given customer can only have one active (non-deleted) entitlement per featureKey. If you try
        to create a new entitlement for a featureKey that already has an active entitlement, the
        request will fail with a 409 error.

        Once an entitlement is created you cannot modify it, only delete it.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :param entitlement: Required.
        :type entitlement: ~openmeter._generated.models.EntitlementBooleanCreateInputs
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: EntitlementMeteredV2 or EntitlementStaticV2 or EntitlementBooleanV2
        :rtype: ~openmeter._generated.models.EntitlementMeteredV2 or
         ~openmeter._generated.models.EntitlementStaticV2 or
         ~openmeter._generated.models.EntitlementBooleanV2
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def post(
        self,
        customer_id_or_key: "_types.ULIDOrExternalKey",
        entitlement: "_types.EntitlementV2CreateInputs",
        **kwargs: Any
    ) -> "_types.EntitlementV2":
        """Create a customer entitlement.

        OpenMeter has three types of entitlements: metered, boolean, and static. The type property
        determines the type of entitlement. The underlying feature has to be compatible with the
        entitlement type specified in the request (e.g., a metered entitlement needs a feature
        associated with a meter).



        * Boolean entitlements define static feature access, e.g. "Can use SSO authentication".
        * Static entitlements let you pass along a configuration while granting access, e.g. "Using
        this feature with X Y settings" (passed in the config).
        * Metered entitlements have many use cases, from setting up usage-based access to implementing
        complex credit systems.  Example: The customer can use 10000 AI tokens during the usage period
        of the entitlement.

        A given customer can only have one active (non-deleted) entitlement per featureKey. If you try
        to create a new entitlement for a featureKey that already has an active entitlement, the
        request will fail with a 409 error.

        Once an entitlement is created you cannot modify it, only delete it.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :param entitlement: Is one of the following types: EntitlementMeteredV2CreateInputs,
         EntitlementStaticCreateInputs, EntitlementBooleanCreateInputs Required.
        :type entitlement: ~openmeter._generated.models.EntitlementMeteredV2CreateInputs or
         ~openmeter._generated.models.EntitlementStaticCreateInputs or
         ~openmeter._generated.models.EntitlementBooleanCreateInputs
        :return: EntitlementMeteredV2 or EntitlementStaticV2 or EntitlementBooleanV2
        :rtype: ~openmeter._generated.models.EntitlementMeteredV2 or
         ~openmeter._generated.models.EntitlementStaticV2 or
         ~openmeter._generated.models.EntitlementBooleanV2
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType["_types.EntitlementV2"] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(entitlement, _models.EntitlementMeteredV2CreateInputs):
            _content = json.dumps(entitlement, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore
        elif isinstance(entitlement, _models.EntitlementStaticCreateInputs):
            _content = json.dumps(entitlement, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore
        elif isinstance(entitlement, _models.EntitlementBooleanCreateInputs):
            _content = json.dumps(entitlement, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_entitlements_v2_customer_entitlements_post_request(
            customer_id_or_key=customer_id_or_key,
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [201]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            elif response.status_code == 409:
                error = _failsafe_deserialize(_models.ConflictProblemResponse, response)
                raise ResourceExistsError(response=response, model=error)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize("_types.EntitlementV2", response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def list(
        self,
        customer_id_or_key: "_types.ULIDOrExternalKey",
        *,
        include_deleted: Optional[bool] = None,
        page: Optional[int] = None,
        page_size: Optional[int] = None,
        order: Optional[Union[str, _models.SortOrder]] = None,
        order_by: Optional[Union[str, _models.EntitlementOrderBy]] = None,
        **kwargs: Any
    ) -> _models.EntitlementV2PaginatedResponse:
        """List customer entitlements.

        List all entitlements for a customer. For checking entitlement access, use the /value endpoint
        instead.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :keyword include_deleted: Default value is None.
        :paramtype include_deleted: bool
        :keyword page: Page index.

         Default is 1. Default value is None.
        :paramtype page: int
        :keyword page_size: The maximum number of items per page.

         Default is 100. Default value is None.
        :paramtype page_size: int
        :keyword order: The order direction. Known values are: "ASC" and "DESC". Default value is None.
        :paramtype order: str or ~openmeter.models.SortOrder
        :keyword order_by: The order by field. Known values are: "createdAt" and "updatedAt". Default
         value is None.
        :paramtype order_by: str or ~openmeter.models.EntitlementOrderBy
        :return: EntitlementV2PaginatedResponse. The EntitlementV2PaginatedResponse is compatible with
         MutableMapping
        :rtype: ~openmeter._generated.models.EntitlementV2PaginatedResponse
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.EntitlementV2PaginatedResponse] = kwargs.pop("cls", None)

        _request = build_entitlements_v2_customer_entitlements_list_request(
            customer_id_or_key=customer_id_or_key,
            include_deleted=include_deleted,
            page=page,
            page_size=page_size,
            order=order,
            order_by=order_by,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.EntitlementV2PaginatedResponse, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def get(
        self, customer_id_or_key: "_types.ULIDOrExternalKey", entitlement_id_or_feature_key: str, **kwargs: Any
    ) -> "_types.EntitlementV2":
        """Get customer entitlement.

        Get entitlement by feature key. For checking entitlement access, use the /value endpoint
        instead.
        If featureKey is used, the entitlement is resolved for the current timestamp.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :param entitlement_id_or_feature_key: Required.
        :type entitlement_id_or_feature_key: str
        :return: EntitlementMeteredV2 or EntitlementStaticV2 or EntitlementBooleanV2
        :rtype: ~openmeter._generated.models.EntitlementMeteredV2 or
         ~openmeter._generated.models.EntitlementStaticV2 or
         ~openmeter._generated.models.EntitlementBooleanV2
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType["_types.EntitlementV2"] = kwargs.pop("cls", None)

        _request = build_entitlements_v2_customer_entitlements_get_request(
            customer_id_or_key=customer_id_or_key,
            entitlement_id_or_feature_key=entitlement_id_or_feature_key,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            elif response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize("_types.EntitlementV2", response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def delete(
        self, customer_id_or_key: "_types.ULIDOrExternalKey", entitlement_id_or_feature_key: str, **kwargs: Any
    ) -> None:
        """Delete customer entitlement.

        Deleting an entitlement revokes access to the associated feature. As a single customer can only
        have one entitlement per featureKey, when "migrating" features you have to delete the old
        entitlements as well.
        As access and status checks can be historical queries, deleting an entitlement populates the
        deletedAt timestamp. When queried for a time before that, the entitlement is still considered
        active, you cannot have retroactive changes to access, which is important for, among other
        things, auditing.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :param entitlement_id_or_feature_key: Required.
        :type entitlement_id_or_feature_key: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[None] = kwargs.pop("cls", None)

        _request = build_entitlements_v2_customer_entitlements_delete_request(
            customer_id_or_key=customer_id_or_key,
            entitlement_id_or_feature_key=entitlement_id_or_feature_key,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = False
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [204]:
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            elif response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if cls:
            return cls(pipeline_response, None, {})  # type: ignore

    @overload
    async def override(
        self,
        customer_id_or_key: "_types.ULIDOrExternalKey",
        entitlement_id_or_feature_key: "_types.ULIDOrExternalKey",
        entitlement: _models.EntitlementMeteredV2CreateInputs,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> "_types.EntitlementV2":
        """Override customer entitlement.

        Overriding an entitlement creates a new entitlement from the provided inputs and soft deletes
        the previous entitlement for the provided customer-feature pair. If the previous entitlement is
        already deleted or otherwise doesnt exist, the override will fail.

        This endpoint is useful for upgrades, downgrades, or other changes to entitlements that require
        a new entitlement to be created with zero downtime.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :param entitlement_id_or_feature_key: Is one of the following types: str Required.
        :type entitlement_id_or_feature_key: str or str
        :param entitlement: Required.
        :type entitlement: ~openmeter._generated.models.EntitlementMeteredV2CreateInputs
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: EntitlementMeteredV2 or EntitlementStaticV2 or EntitlementBooleanV2
        :rtype: ~openmeter._generated.models.EntitlementMeteredV2 or
         ~openmeter._generated.models.EntitlementStaticV2 or
         ~openmeter._generated.models.EntitlementBooleanV2
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def override(
        self,
        customer_id_or_key: "_types.ULIDOrExternalKey",
        entitlement_id_or_feature_key: "_types.ULIDOrExternalKey",
        entitlement: _models.EntitlementStaticCreateInputs,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> "_types.EntitlementV2":
        """Override customer entitlement.

        Overriding an entitlement creates a new entitlement from the provided inputs and soft deletes
        the previous entitlement for the provided customer-feature pair. If the previous entitlement is
        already deleted or otherwise doesnt exist, the override will fail.

        This endpoint is useful for upgrades, downgrades, or other changes to entitlements that require
        a new entitlement to be created with zero downtime.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :param entitlement_id_or_feature_key: Is one of the following types: str Required.
        :type entitlement_id_or_feature_key: str or str
        :param entitlement: Required.
        :type entitlement: ~openmeter._generated.models.EntitlementStaticCreateInputs
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: EntitlementMeteredV2 or EntitlementStaticV2 or EntitlementBooleanV2
        :rtype: ~openmeter._generated.models.EntitlementMeteredV2 or
         ~openmeter._generated.models.EntitlementStaticV2 or
         ~openmeter._generated.models.EntitlementBooleanV2
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def override(
        self,
        customer_id_or_key: "_types.ULIDOrExternalKey",
        entitlement_id_or_feature_key: "_types.ULIDOrExternalKey",
        entitlement: _models.EntitlementBooleanCreateInputs,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> "_types.EntitlementV2":
        """Override customer entitlement.

        Overriding an entitlement creates a new entitlement from the provided inputs and soft deletes
        the previous entitlement for the provided customer-feature pair. If the previous entitlement is
        already deleted or otherwise doesnt exist, the override will fail.

        This endpoint is useful for upgrades, downgrades, or other changes to entitlements that require
        a new entitlement to be created with zero downtime.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :param entitlement_id_or_feature_key: Is one of the following types: str Required.
        :type entitlement_id_or_feature_key: str or str
        :param entitlement: Required.
        :type entitlement: ~openmeter._generated.models.EntitlementBooleanCreateInputs
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: EntitlementMeteredV2 or EntitlementStaticV2 or EntitlementBooleanV2
        :rtype: ~openmeter._generated.models.EntitlementMeteredV2 or
         ~openmeter._generated.models.EntitlementStaticV2 or
         ~openmeter._generated.models.EntitlementBooleanV2
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def override(
        self,
        customer_id_or_key: "_types.ULIDOrExternalKey",
        entitlement_id_or_feature_key: "_types.ULIDOrExternalKey",
        entitlement: "_types.EntitlementV2CreateInputs",
        **kwargs: Any
    ) -> "_types.EntitlementV2":
        """Override customer entitlement.

        Overriding an entitlement creates a new entitlement from the provided inputs and soft deletes
        the previous entitlement for the provided customer-feature pair. If the previous entitlement is
        already deleted or otherwise doesnt exist, the override will fail.

        This endpoint is useful for upgrades, downgrades, or other changes to entitlements that require
        a new entitlement to be created with zero downtime.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :param entitlement_id_or_feature_key: Is one of the following types: str Required.
        :type entitlement_id_or_feature_key: str or str
        :param entitlement: Is one of the following types: EntitlementMeteredV2CreateInputs,
         EntitlementStaticCreateInputs, EntitlementBooleanCreateInputs Required.
        :type entitlement: ~openmeter._generated.models.EntitlementMeteredV2CreateInputs or
         ~openmeter._generated.models.EntitlementStaticCreateInputs or
         ~openmeter._generated.models.EntitlementBooleanCreateInputs
        :return: EntitlementMeteredV2 or EntitlementStaticV2 or EntitlementBooleanV2
        :rtype: ~openmeter._generated.models.EntitlementMeteredV2 or
         ~openmeter._generated.models.EntitlementStaticV2 or
         ~openmeter._generated.models.EntitlementBooleanV2
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType["_types.EntitlementV2"] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(entitlement, _models.EntitlementMeteredV2CreateInputs):
            _content = json.dumps(entitlement, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore
        elif isinstance(entitlement, _models.EntitlementStaticCreateInputs):
            _content = json.dumps(entitlement, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore
        elif isinstance(entitlement, _models.EntitlementBooleanCreateInputs):
            _content = json.dumps(entitlement, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_entitlements_v2_customer_entitlements_override_request(
            customer_id_or_key=customer_id_or_key,
            entitlement_id_or_feature_key=entitlement_id_or_feature_key,
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [201]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            elif response.status_code == 409:
                error = _failsafe_deserialize(_models.ConflictProblemResponse, response)
                raise ResourceExistsError(response=response, model=error)
            if response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize("_types.EntitlementV2", response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore


class EntitlementsV2CustomerEntitlementOperations:  # pylint: disable=name-too-long
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`customer_entitlement` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

    async def get_grants(
        self,
        customer_id_or_key: "_types.ULIDOrExternalKey",
        entitlement_id_or_feature_key: str,
        *,
        include_deleted: Optional[bool] = None,
        page: Optional[int] = None,
        page_size: Optional[int] = None,
        offset: Optional[int] = None,
        limit: Optional[int] = None,
        order: Optional[Union[str, _models.SortOrder]] = None,
        order_by: Optional[Union[str, _models.GrantOrderBy]] = None,
        **kwargs: Any
    ) -> _models.GrantV2PaginatedResponse:
        """List customer entitlement grants.

        List all grants issued for an entitlement. The entitlement can be defined either by its id or
        featureKey.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :param entitlement_id_or_feature_key: Required.
        :type entitlement_id_or_feature_key: str
        :keyword include_deleted: Default value is None.
        :paramtype include_deleted: bool
        :keyword page: Page index.

         Default is 1. Default value is None.
        :paramtype page: int
        :keyword page_size: The maximum number of items per page.

         Default is 100. Default value is None.
        :paramtype page_size: int
        :keyword offset: Number of items to skip.

         Default is 0. Default value is None.
        :paramtype offset: int
        :keyword limit: Number of items to return.

         Default is 100. Default value is None.
        :paramtype limit: int
        :keyword order: The order direction. Known values are: "ASC" and "DESC". Default value is None.
        :paramtype order: str or ~openmeter.models.SortOrder
        :keyword order_by: The order by field. Known values are: "id", "createdAt", and "updatedAt".
         Default value is None.
        :paramtype order_by: str or ~openmeter.models.GrantOrderBy
        :return: GrantV2PaginatedResponse. The GrantV2PaginatedResponse is compatible with
         MutableMapping
        :rtype: ~openmeter._generated.models.GrantV2PaginatedResponse
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.GrantV2PaginatedResponse] = kwargs.pop("cls", None)

        _request = build_entitlements_v2_customer_entitlement_get_grants_request(
            customer_id_or_key=customer_id_or_key,
            entitlement_id_or_feature_key=entitlement_id_or_feature_key,
            include_deleted=include_deleted,
            page=page,
            page_size=page_size,
            offset=offset,
            limit=limit,
            order=order,
            order_by=order_by,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.GrantV2PaginatedResponse, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    @overload
    async def create_customer_entitlement_grant(
        self,
        customer_id_or_key: "_types.ULIDOrExternalKey",
        entitlement_id_or_feature_key: str,
        grant: _models.EntitlementGrantCreateInputV2,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> _models.EntitlementGrantV2:
        """Create customer entitlement grant.

        Grants define a behavior of granting usage for a metered entitlement. They can have complicated
        recurrence and rollover rules, thanks to which you can define a wide range of access patterns
        with a single grant, in most cases you don't have to periodically create new grants. You can
        only issue grants for active metered entitlements.

        A grant defines a given amount of usage that can be consumed for the entitlement. The grant is
        in effect between its effective date and its expiration date. Specifying both is mandatory for
        new grants.

        Grants have a priority setting that determines their order of use. Lower numbers have higher
        priority, with 0 being the highest priority.

        Grants can have a recurrence setting intended to automate the manual reissuing of grants. For
        example, a daily recurrence is equal to reissuing that same grant every day (ignoring rollover
        settings).

        Rollover settings define what happens to the remaining balance of a grant at a reset.
        Balance_After_Reset = MIN(MaxRolloverAmount, MAX(Balance_Before_Reset, MinRolloverAmount))

        Grants cannot be changed once created, only deleted. This is to ensure that balance is
        deterministic regardless of when it is queried.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :param entitlement_id_or_feature_key: Required.
        :type entitlement_id_or_feature_key: str
        :param grant: Required.
        :type grant: ~openmeter._generated.models.EntitlementGrantCreateInputV2
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: EntitlementGrantV2. The EntitlementGrantV2 is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.EntitlementGrantV2
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def create_customer_entitlement_grant(
        self,
        customer_id_or_key: "_types.ULIDOrExternalKey",
        entitlement_id_or_feature_key: str,
        grant: JSON,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> _models.EntitlementGrantV2:
        """Create customer entitlement grant.

        Grants define a behavior of granting usage for a metered entitlement. They can have complicated
        recurrence and rollover rules, thanks to which you can define a wide range of access patterns
        with a single grant, in most cases you don't have to periodically create new grants. You can
        only issue grants for active metered entitlements.

        A grant defines a given amount of usage that can be consumed for the entitlement. The grant is
        in effect between its effective date and its expiration date. Specifying both is mandatory for
        new grants.

        Grants have a priority setting that determines their order of use. Lower numbers have higher
        priority, with 0 being the highest priority.

        Grants can have a recurrence setting intended to automate the manual reissuing of grants. For
        example, a daily recurrence is equal to reissuing that same grant every day (ignoring rollover
        settings).

        Rollover settings define what happens to the remaining balance of a grant at a reset.
        Balance_After_Reset = MIN(MaxRolloverAmount, MAX(Balance_Before_Reset, MinRolloverAmount))

        Grants cannot be changed once created, only deleted. This is to ensure that balance is
        deterministic regardless of when it is queried.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :param entitlement_id_or_feature_key: Required.
        :type entitlement_id_or_feature_key: str
        :param grant: Required.
        :type grant: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: EntitlementGrantV2. The EntitlementGrantV2 is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.EntitlementGrantV2
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def create_customer_entitlement_grant(
        self,
        customer_id_or_key: "_types.ULIDOrExternalKey",
        entitlement_id_or_feature_key: str,
        grant: IO[bytes],
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> _models.EntitlementGrantV2:
        """Create customer entitlement grant.

        Grants define a behavior of granting usage for a metered entitlement. They can have complicated
        recurrence and rollover rules, thanks to which you can define a wide range of access patterns
        with a single grant, in most cases you don't have to periodically create new grants. You can
        only issue grants for active metered entitlements.

        A grant defines a given amount of usage that can be consumed for the entitlement. The grant is
        in effect between its effective date and its expiration date. Specifying both is mandatory for
        new grants.

        Grants have a priority setting that determines their order of use. Lower numbers have higher
        priority, with 0 being the highest priority.

        Grants can have a recurrence setting intended to automate the manual reissuing of grants. For
        example, a daily recurrence is equal to reissuing that same grant every day (ignoring rollover
        settings).

        Rollover settings define what happens to the remaining balance of a grant at a reset.
        Balance_After_Reset = MIN(MaxRolloverAmount, MAX(Balance_Before_Reset, MinRolloverAmount))

        Grants cannot be changed once created, only deleted. This is to ensure that balance is
        deterministic regardless of when it is queried.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :param entitlement_id_or_feature_key: Required.
        :type entitlement_id_or_feature_key: str
        :param grant: Required.
        :type grant: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: EntitlementGrantV2. The EntitlementGrantV2 is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.EntitlementGrantV2
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def create_customer_entitlement_grant(
        self,
        customer_id_or_key: "_types.ULIDOrExternalKey",
        entitlement_id_or_feature_key: str,
        grant: Union[_models.EntitlementGrantCreateInputV2, JSON, IO[bytes]],
        **kwargs: Any
    ) -> _models.EntitlementGrantV2:
        """Create customer entitlement grant.

        Grants define a behavior of granting usage for a metered entitlement. They can have complicated
        recurrence and rollover rules, thanks to which you can define a wide range of access patterns
        with a single grant, in most cases you don't have to periodically create new grants. You can
        only issue grants for active metered entitlements.

        A grant defines a given amount of usage that can be consumed for the entitlement. The grant is
        in effect between its effective date and its expiration date. Specifying both is mandatory for
        new grants.

        Grants have a priority setting that determines their order of use. Lower numbers have higher
        priority, with 0 being the highest priority.

        Grants can have a recurrence setting intended to automate the manual reissuing of grants. For
        example, a daily recurrence is equal to reissuing that same grant every day (ignoring rollover
        settings).

        Rollover settings define what happens to the remaining balance of a grant at a reset.
        Balance_After_Reset = MIN(MaxRolloverAmount, MAX(Balance_Before_Reset, MinRolloverAmount))

        Grants cannot be changed once created, only deleted. This is to ensure that balance is
        deterministic regardless of when it is queried.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :param entitlement_id_or_feature_key: Required.
        :type entitlement_id_or_feature_key: str
        :param grant: Is one of the following types: EntitlementGrantCreateInputV2, JSON, IO[bytes]
         Required.
        :type grant: ~openmeter._generated.models.EntitlementGrantCreateInputV2 or JSON or IO[bytes]
        :return: EntitlementGrantV2. The EntitlementGrantV2 is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.EntitlementGrantV2
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[_models.EntitlementGrantV2] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(grant, (IOBase, bytes)):
            _content = grant
        else:
            _content = json.dumps(grant, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_entitlements_v2_customer_entitlement_create_customer_entitlement_grant_request(
            customer_id_or_key=customer_id_or_key,
            entitlement_id_or_feature_key=entitlement_id_or_feature_key,
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [201]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            elif response.status_code == 409:
                error = _failsafe_deserialize(_models.ConflictProblemResponse, response)
                raise ResourceExistsError(response=response, model=error)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.EntitlementGrantV2, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def get_customer_entitlement_value(
        self,
        customer_id_or_key: "_types.ULIDOrExternalKey",
        entitlement_id_or_feature_key: str,
        *,
        time: Optional[datetime.datetime] = None,
        **kwargs: Any
    ) -> _models.EntitlementValue:
        """Get customer entitlement value.

        Checks customer access to a given feature (by key). All entitlement types share the hasAccess
        property in their value response, but multiple other properties are returned based on the
        entitlement type.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :param entitlement_id_or_feature_key: Required.
        :type entitlement_id_or_feature_key: str
        :keyword time: Default value is None.
        :paramtype time: ~datetime.datetime
        :return: EntitlementValue. The EntitlementValue is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.EntitlementValue
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.EntitlementValue] = kwargs.pop("cls", None)

        _request = build_entitlements_v2_customer_entitlement_get_customer_entitlement_value_request(
            customer_id_or_key=customer_id_or_key,
            entitlement_id_or_feature_key=entitlement_id_or_feature_key,
            time=time,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            elif response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.EntitlementValue, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    async def get_customer_entitlement_history(
        self,
        customer_id_or_key: "_types.ULIDOrExternalKey",
        entitlement_id_or_feature_key: str,
        *,
        window_size: Union[str, _models.WindowSize],
        from_parameter: Optional[datetime.datetime] = None,
        to: Optional[datetime.datetime] = None,
        window_time_zone: Optional[str] = None,
        **kwargs: Any
    ) -> _models.WindowedBalanceHistory:
        """Get customer entitlement history.

        Returns historical balance and usage data for the entitlement. The queried history can span
        accross multiple reset events.

        BurndownHistory returns a continous history of segments, where the segments are seperated by
        events that changed either the grant burndown priority or the usage period.

        WindowedHistory returns windowed usage data for the period enriched with balance information
        and the list of grants that were being burnt down in that window.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :param entitlement_id_or_feature_key: Required.
        :type entitlement_id_or_feature_key: str
        :keyword window_size: Windowsize. Known values are: "MINUTE", "HOUR", "DAY", and "MONTH".
         Required.
        :paramtype window_size: str or ~openmeter.models.WindowSize
        :keyword from_parameter: Start of time range to query entitlement: date-time in RFC 3339
         format. Defaults to the last reset. Gets truncated to the granularity of the underlying meter.
         Default value is None.
        :paramtype from_parameter: ~datetime.datetime
        :keyword to: End of time range to query entitlement: date-time in RFC 3339 format. Defaults to
         now.
         If not now then gets truncated to the granularity of the underlying meter. Default value is
         None.
        :paramtype to: ~datetime.datetime
        :keyword window_time_zone: The timezone used when calculating the windows. Default value is
         None.
        :paramtype window_time_zone: str
        :return: WindowedBalanceHistory. The WindowedBalanceHistory is compatible with MutableMapping
        :rtype: ~openmeter._generated.models.WindowedBalanceHistory
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.WindowedBalanceHistory] = kwargs.pop("cls", None)

        _request = build_entitlements_v2_customer_entitlement_get_customer_entitlement_history_request(
            customer_id_or_key=customer_id_or_key,
            entitlement_id_or_feature_key=entitlement_id_or_feature_key,
            window_size=window_size,
            from_parameter=from_parameter,
            to=to,
            window_time_zone=window_time_zone,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            elif response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.WindowedBalanceHistory, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore

    @overload
    async def reset_customer_entitlement(
        self,
        customer_id_or_key: "_types.ULIDOrExternalKey",
        entitlement_id_or_feature_key: str,
        reset: _models.ResetEntitlementUsageInput,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> None:
        """Reset customer entitlement.

        Reset marks the start of a new usage period for the entitlement and initiates grant rollover.
        At the start of a period usage is zerod out and grants are rolled over based on their rollover
        settings. It would typically be synced with the customers billing period to enforce usage based
        on their subscription.

        Usage is automatically reset for metered entitlements based on their usage period, but this
        endpoint allows to manually reset it at any time. When doing so the period anchor of the
        entitlement can be changed if needed.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :param entitlement_id_or_feature_key: Required.
        :type entitlement_id_or_feature_key: str
        :param reset: Required.
        :type reset: ~openmeter._generated.models.ResetEntitlementUsageInput
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def reset_customer_entitlement(
        self,
        customer_id_or_key: "_types.ULIDOrExternalKey",
        entitlement_id_or_feature_key: str,
        reset: JSON,
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> None:
        """Reset customer entitlement.

        Reset marks the start of a new usage period for the entitlement and initiates grant rollover.
        At the start of a period usage is zerod out and grants are rolled over based on their rollover
        settings. It would typically be synced with the customers billing period to enforce usage based
        on their subscription.

        Usage is automatically reset for metered entitlements based on their usage period, but this
        endpoint allows to manually reset it at any time. When doing so the period anchor of the
        entitlement can be changed if needed.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :param entitlement_id_or_feature_key: Required.
        :type entitlement_id_or_feature_key: str
        :param reset: Required.
        :type reset: JSON
        :keyword content_type: Body Parameter content-type. Content type parameter for JSON body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    @overload
    async def reset_customer_entitlement(
        self,
        customer_id_or_key: "_types.ULIDOrExternalKey",
        entitlement_id_or_feature_key: str,
        reset: IO[bytes],
        *,
        content_type: str = "application/json",
        **kwargs: Any
    ) -> None:
        """Reset customer entitlement.

        Reset marks the start of a new usage period for the entitlement and initiates grant rollover.
        At the start of a period usage is zerod out and grants are rolled over based on their rollover
        settings. It would typically be synced with the customers billing period to enforce usage based
        on their subscription.

        Usage is automatically reset for metered entitlements based on their usage period, but this
        endpoint allows to manually reset it at any time. When doing so the period anchor of the
        entitlement can be changed if needed.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :param entitlement_id_or_feature_key: Required.
        :type entitlement_id_or_feature_key: str
        :param reset: Required.
        :type reset: IO[bytes]
        :keyword content_type: Body Parameter content-type. Content type parameter for binary body.
         Default value is "application/json".
        :paramtype content_type: str
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """

    async def reset_customer_entitlement(
        self,
        customer_id_or_key: "_types.ULIDOrExternalKey",
        entitlement_id_or_feature_key: str,
        reset: Union[_models.ResetEntitlementUsageInput, JSON, IO[bytes]],
        **kwargs: Any
    ) -> None:
        """Reset customer entitlement.

        Reset marks the start of a new usage period for the entitlement and initiates grant rollover.
        At the start of a period usage is zerod out and grants are rolled over based on their rollover
        settings. It would typically be synced with the customers billing period to enforce usage based
        on their subscription.

        Usage is automatically reset for metered entitlements based on their usage period, but this
        endpoint allows to manually reset it at any time. When doing so the period anchor of the
        entitlement can be changed if needed.

        :param customer_id_or_key: Is one of the following types: str Required.
        :type customer_id_or_key: str or str
        :param entitlement_id_or_feature_key: Required.
        :type entitlement_id_or_feature_key: str
        :param reset: Is one of the following types: ResetEntitlementUsageInput, JSON, IO[bytes]
         Required.
        :type reset: ~openmeter._generated.models.ResetEntitlementUsageInput or JSON or IO[bytes]
        :return: None
        :rtype: None
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
        _params = kwargs.pop("params", {}) or {}

        content_type: Optional[str] = kwargs.pop("content_type", _headers.pop("Content-Type", None))
        cls: ClsType[None] = kwargs.pop("cls", None)

        content_type = content_type or "application/json"
        _content = None
        if isinstance(reset, (IOBase, bytes)):
            _content = reset
        else:
            _content = json.dumps(reset, cls=SdkJSONEncoder, exclude_readonly=True)  # type: ignore

        _request = build_entitlements_v2_customer_entitlement_reset_customer_entitlement_request(
            customer_id_or_key=customer_id_or_key,
            entitlement_id_or_feature_key=entitlement_id_or_feature_key,
            content_type=content_type,
            content=_content,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = False
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [204]:
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            elif response.status_code == 404:
                error = _failsafe_deserialize(_models.NotFoundProblemResponse, response)
                raise ResourceNotFoundError(response=response, model=error)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if cls:
            return cls(pipeline_response, None, {})  # type: ignore


class EntitlementsV2GrantsOperations:
    """
    .. warning::
        **DO NOT** instantiate this class directly.

        Instead, you should access the following operations through
        :class:`~openmeter.aio.OpenMeterClient`'s
        :attr:`grants` attribute.
    """

    def __init__(self, *args, **kwargs) -> None:
        input_args = list(args)
        self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop("client")
        self._config: OpenMeterClientConfiguration = input_args.pop(0) if input_args else kwargs.pop("config")
        self._serialize: Serializer = input_args.pop(0) if input_args else kwargs.pop("serializer")
        self._deserialize: Deserializer = input_args.pop(0) if input_args else kwargs.pop("deserializer")

    async def list(
        self,
        *,
        feature: Optional[List[str]] = None,
        customer: Optional[List["_types.ULIDOrExternalKey"]] = None,
        include_deleted: Optional[bool] = None,
        page: Optional[int] = None,
        page_size: Optional[int] = None,
        offset: Optional[int] = None,
        limit: Optional[int] = None,
        order: Optional[Union[str, _models.SortOrder]] = None,
        order_by: Optional[Union[str, _models.GrantOrderBy]] = None,
        **kwargs: Any
    ) -> _models.GrantV2PaginatedResponse:
        """List grants.

        List all grants for all the customers and entitlements. This endpoint is intended for
        administrative purposes only.
        To fetch the grants of a specific entitlement please use the
        /api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/grants endpoint.
        If page is provided that takes precedence and the paginated response is returned.

        :keyword feature: Filtering by multiple features.

         Usage: ``?feature=feature-1&feature=feature-2``. Default value is None.
        :paramtype feature: list[str]
        :keyword customer: Filtering by multiple customers (either by ID or key).

         Usage: ``?customer=customer-1&customer=customer-2``. Default value is None.
        :paramtype customer: list[str or str]
        :keyword include_deleted: Include deleted. Default value is None.
        :paramtype include_deleted: bool
        :keyword page: Page index.

         Default is 1. Default value is None.
        :paramtype page: int
        :keyword page_size: The maximum number of items per page.

         Default is 100. Default value is None.
        :paramtype page_size: int
        :keyword offset: Number of items to skip.

         Default is 0. Default value is None.
        :paramtype offset: int
        :keyword limit: Number of items to return.

         Default is 100. Default value is None.
        :paramtype limit: int
        :keyword order: The order direction. Known values are: "ASC" and "DESC". Default value is None.
        :paramtype order: str or ~openmeter.models.SortOrder
        :keyword order_by: The order by field. Known values are: "id", "createdAt", and "updatedAt".
         Default value is None.
        :paramtype order_by: str or ~openmeter.models.GrantOrderBy
        :return: GrantV2PaginatedResponse. The GrantV2PaginatedResponse is compatible with
         MutableMapping
        :rtype: ~openmeter._generated.models.GrantV2PaginatedResponse
        :raises ~corehttp.exceptions.HttpResponseError:
        """
        error_map: MutableMapping = {
            404: ResourceNotFoundError,
            409: ResourceExistsError,
            304: ResourceNotModifiedError,
        }
        error_map.update(kwargs.pop("error_map", {}) or {})

        _headers = kwargs.pop("headers", {}) or {}
        _params = kwargs.pop("params", {}) or {}

        cls: ClsType[_models.GrantV2PaginatedResponse] = kwargs.pop("cls", None)

        _request = build_entitlements_v2_grants_list_request(
            feature=feature,
            customer=customer,
            include_deleted=include_deleted,
            page=page,
            page_size=page_size,
            offset=offset,
            limit=limit,
            order=order,
            order_by=order_by,
            headers=_headers,
            params=_params,
        )
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }
        _request.url = self._client.format_url(_request.url, **path_format_arguments)

        _stream = kwargs.pop("stream", False)
        pipeline_response: PipelineResponse = await self._client.pipeline.run(_request, stream=_stream, **kwargs)

        response = pipeline_response.http_response

        if response.status_code not in [200]:
            if _stream:
                try:
                    await response.read()  # Load the body in memory and close the socket
                except (StreamConsumedError, StreamClosedError):
                    pass
            map_error(status_code=response.status_code, response=response, error_map=error_map)
            error = None
            if response.status_code == 400:
                error = _failsafe_deserialize(_models.BadRequestProblemResponse, response)
            elif response.status_code == 401:
                error = _failsafe_deserialize(_models.UnauthorizedProblemResponse, response)
                raise ClientAuthenticationError(response=response, model=error)
            if response.status_code == 403:
                error = _failsafe_deserialize(_models.ForbiddenProblemResponse, response)
            elif response.status_code == 500:
                error = _failsafe_deserialize(_models.InternalServerErrorProblemResponse, response)
            elif response.status_code == 503:
                error = _failsafe_deserialize(_models.ServiceUnavailableProblemResponse, response)
            elif response.status_code == 412:
                error = _failsafe_deserialize(_models.PreconditionFailedProblemResponse, response)
            else:
                error = _failsafe_deserialize(
                    _models.UnexpectedProblemResponse,
                    response,
                )
            raise HttpResponseError(response=response, model=error)

        if _stream:
            deserialized = response.iter_bytes()
        else:
            deserialized = _deserialize(_models.GrantV2PaginatedResponse, response.json())

        if cls:
            return cls(pipeline_response, deserialized, {})  # type: ignore

        return deserialized  # type: ignore
