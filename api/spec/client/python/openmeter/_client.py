# coding=utf-8

from copy import deepcopy
from typing import Any
from typing_extensions import Self

from corehttp.credentials import ServiceKeyCredential
from corehttp.rest import HttpRequest, HttpResponse
from corehttp.runtime import PipelineClient, policies

from ._configuration import OpenMeterCloudClientConfiguration
from ._utils.serialization import Deserializer, Serializer
from .operations import (
    DebugOperations,
    EventsOperations,
    EventsV2Operations,
    MetersOperations,
    OpenMeterCloudOperations,
)


class OpenMeterCloudClient:  # pylint: disable=client-accepts-api-version-keyword
    """OpenMeter is a cloud native usage metering service.
    The OpenMeter API allows you to ingest events, query meter usage, and manage resources.

    :ivar open_meter_cloud: OpenMeterCloudOperations operations
    :vartype open_meter_cloud: openmeter.operations.OpenMeterCloudOperations
    :ivar events: EventsOperations operations
    :vartype events: openmeter.operations.EventsOperations
    :ivar events_v2: EventsV2Operations operations
    :vartype events_v2: openmeter.operations.EventsV2Operations
    :ivar meters: MetersOperations operations
    :vartype meters: openmeter.operations.MetersOperations
    :ivar debug: DebugOperations operations
    :vartype debug: openmeter.operations.DebugOperations
    :param credential: Credential used to authenticate requests to the service. Required.
    :type credential: ~corehttp.credentials.ServiceKeyCredential
    :keyword endpoint: Service host. Default value is "https://openmeter.cloud".
    :paramtype endpoint: str
    """

    def __init__(
        self, credential: ServiceKeyCredential, *, endpoint: str = "https://openmeter.cloud", **kwargs: Any
    ) -> None:
        _endpoint = "{endpoint}"
        self._config = OpenMeterCloudClientConfiguration(credential=credential, endpoint=endpoint, **kwargs)

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
        self.open_meter_cloud = OpenMeterCloudOperations(self._client, self._config, self._serialize, self._deserialize)
        self.events = EventsOperations(self._client, self._config, self._serialize, self._deserialize)
        self.events_v2 = EventsV2Operations(self._client, self._config, self._serialize, self._deserialize)
        self.meters = MetersOperations(self._client, self._config, self._serialize, self._deserialize)
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
