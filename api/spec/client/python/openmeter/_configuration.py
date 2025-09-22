# coding=utf-8

from typing import Any

from corehttp.credentials import ServiceKeyCredential
from corehttp.runtime import policies

from ._version import VERSION


class OpenMeterCloudClientConfiguration:
    """Configuration for OpenMeterCloudClient.

    Note that all parameters used to create this instance are saved as instance
    attributes.

    :param credential: Credential used to authenticate requests to the service. Required.
    :type credential: ~corehttp.credentials.ServiceKeyCredential
    :param endpoint: Service host. Default value is "https://openmeter.cloud".
    :type endpoint: str
    """

    def __init__(
        self, credential: ServiceKeyCredential, endpoint: str = "https://openmeter.cloud", **kwargs: Any
    ) -> None:
        if credential is None:
            raise ValueError("Parameter 'credential' must not be None.")

        self.credential = credential
        self.endpoint = endpoint
        kwargs.setdefault("sdk_moniker", "openmeter/{}".format(VERSION))
        self.polling_interval = kwargs.get("polling_interval", 30)
        self._configure(**kwargs)

    def _configure(self, **kwargs: Any) -> None:
        self.user_agent_policy = kwargs.get("user_agent_policy") or policies.UserAgentPolicy(**kwargs)
        self.headers_policy = kwargs.get("headers_policy") or policies.HeadersPolicy(**kwargs)
        self.proxy_policy = kwargs.get("proxy_policy") or policies.ProxyPolicy(**kwargs)
        self.logging_policy = kwargs.get("logging_policy") or policies.NetworkTraceLoggingPolicy(**kwargs)
        self.retry_policy = kwargs.get("retry_policy") or policies.RetryPolicy(**kwargs)
        self.authentication_policy = kwargs.get("authentication_policy")
        if self.credential and not self.authentication_policy:
            self.authentication_policy = policies.ServiceKeyCredentialPolicy(
                self.credential, "Authorization", prefix="Bearer", **kwargs
            )
