# coding=utf-8

from typing import Any, Optional
from typing_extensions import Self

from corehttp.credentials import ServiceKeyCredential
from corehttp.runtime import policies

from ._generated._client import OpenMeterClient


class Client(OpenMeterClient):
    def __init__(
        self,
        endpoint: str = "https://openmeter.cloud",
        token: Optional[str] = None,
        **kwargs: Any,
    ) -> None:
        if token and not kwargs.get("authentication_policy"):
            credential = ServiceKeyCredential(token)
            kwargs["authentication_policy"] = policies.ServiceKeyCredentialPolicy(
                credential, "Authorization", prefix="Bearer"
            )

        super().__init__(endpoint=endpoint, **kwargs)

    def __enter__(self) -> Self:
        return super().__enter__()

    def __exit__(self, *exc_details: Any) -> None:
        return super().__exit__(*exc_details)
