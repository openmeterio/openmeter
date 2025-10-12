# coding=utf-8

from copy import deepcopy
from typing import Any
from typing_extensions import Self

from corehttp.rest import HttpRequest, HttpResponse
from corehttp.runtime import PipelineClient, policies

from ._configuration import OpenMeterClientConfiguration
from ._utils.serialization import Deserializer, Serializer
from .operations import (
    AppOperations,
    BillingOperations,
    CustomerOperations,
    DebugOperations,
    EntitlementsOperations,
    EventsOperations,
    EventsV2Operations,
    ExportsOperations,
    InfoOperations,
    MetersOperations,
    NotificationOperations,
    PortalOperations,
    ProductCatalogOperations,
    SubjectsOperations,
)


class OpenMeterClient:  # pylint: disable=client-accepts-api-version-keyword,too-many-instance-attributes
    """OpenMeter is a cloud native usage metering service.
    The OpenMeter API allows you to ingest events, query meter usage, and manage resources.

    :ivar app: AppOperations operations
    :vartype app: openmeter.operations.AppOperations
    :ivar customer: CustomerOperations operations
    :vartype customer: openmeter.operations.CustomerOperations
    :ivar product_catalog: ProductCatalogOperations operations
    :vartype product_catalog: openmeter.operations.ProductCatalogOperations
    :ivar entitlements: EntitlementsOperations operations
    :vartype entitlements: openmeter.operations.EntitlementsOperations
    :ivar billing: BillingOperations operations
    :vartype billing: openmeter.operations.BillingOperations
    :ivar portal: PortalOperations operations
    :vartype portal: openmeter.operations.PortalOperations
    :ivar notification: NotificationOperations operations
    :vartype notification: openmeter.operations.NotificationOperations
    :ivar info: InfoOperations operations
    :vartype info: openmeter.operations.InfoOperations
    :ivar exports: ExportsOperations operations
    :vartype exports: openmeter.operations.ExportsOperations
    :ivar events: EventsOperations operations
    :vartype events: openmeter.operations.EventsOperations
    :ivar events_v2: EventsV2Operations operations
    :vartype events_v2: openmeter.operations.EventsV2Operations
    :ivar meters: MetersOperations operations
    :vartype meters: openmeter.operations.MetersOperations
    :ivar subjects: SubjectsOperations operations
    :vartype subjects: openmeter.operations.SubjectsOperations
    :ivar debug: DebugOperations operations
    :vartype debug: openmeter.operations.DebugOperations
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
        self._client: PipelineClient = PipelineClient(endpoint=_endpoint, policies=_policies, **kwargs)

        self._serialize = Serializer()
        self._deserialize = Deserializer()
        self._serialize.client_side_validation = False
        self.app = AppOperations(self._client, self._config, self._serialize, self._deserialize)
        self.customer = CustomerOperations(self._client, self._config, self._serialize, self._deserialize)
        self.product_catalog = ProductCatalogOperations(self._client, self._config, self._serialize, self._deserialize)
        self.entitlements = EntitlementsOperations(self._client, self._config, self._serialize, self._deserialize)
        self.billing = BillingOperations(self._client, self._config, self._serialize, self._deserialize)
        self.portal = PortalOperations(self._client, self._config, self._serialize, self._deserialize)
        self.notification = NotificationOperations(self._client, self._config, self._serialize, self._deserialize)
        self.info = InfoOperations(self._client, self._config, self._serialize, self._deserialize)
        self.exports = ExportsOperations(self._client, self._config, self._serialize, self._deserialize)
        self.events = EventsOperations(self._client, self._config, self._serialize, self._deserialize)
        self.events_v2 = EventsV2Operations(self._client, self._config, self._serialize, self._deserialize)
        self.meters = MetersOperations(self._client, self._config, self._serialize, self._deserialize)
        self.subjects = SubjectsOperations(self._client, self._config, self._serialize, self._deserialize)
        self.debug = DebugOperations(self._client, self._config, self._serialize, self._deserialize)

    def send_request(self, request: HttpRequest, *, stream: bool = False, **kwargs: Any) -> HttpResponse:
        """Runs the network request through the client's chained policies.

        >>> from corehttp.rest import HttpRequest
        >>> request = HttpRequest("GET", "https://www.example.org/")
        <HttpRequest [GET], url: 'https://www.example.org/'>
        >>> response = client.send_request(request)
        <HttpResponse: 200 OK>

        For more information on this code flow, see https://aka.ms/azsdk/dpcodegen/python/send_request

        :param request: The network request you want to make. Required.
        :type request: ~corehttp.rest.HttpRequest
        :keyword bool stream: Whether the response payload will be streamed. Defaults to False.
        :return: The response of your network call. Does not do error handling on your response.
        :rtype: ~corehttp.rest.HttpResponse
        """

        request_copy = deepcopy(request)
        path_format_arguments = {
            "endpoint": self._serialize.url("self._config.endpoint", self._config.endpoint, "str", skip_quote=True),
        }

        request_copy.url = self._client.format_url(request_copy.url, **path_format_arguments)
        return self._client.send_request(request_copy, stream=stream, **kwargs)  # type: ignore

    def close(self) -> None:
        self._client.close()

    def __enter__(self) -> Self:
        self._client.__enter__()
        return self

    def __exit__(self, *exc_details: Any) -> None:
        self._client.__exit__(*exc_details)
