# ------------------------------------
# Copyright (c) Microsoft Corporation.
# Licensed under the MIT License.
# ------------------------------------
import datetime
from typing import Any, Dict, List, Optional
from azure.core.rest import HttpRequest
from ._operations import _SERIALIZER
from . import _operations

from azure.core.utils import case_insensitive_dict

"""Customize generated code here.

Follow our quickstart for examples: https://aka.ms/azsdk/python/dpcodegen/python/customize
"""
from typing import List

__all__: List[str] = []  # Add all objects you want publicly available to users at this package level


def patch_sdk():
    """Do not remove from this file.

    `patch_sdk` is a last resort escape hatch that allows you to do customizations
    you can't accomplish using the techniques described in
    https://aka.ms/azsdk/python/dpcodegen/python/customize
    """
    _operations.build_query_meter_request = build_query_meter_request
    _operations.build_query_portal_meter_request = build_query_portal_meter_request


def build_query_meter_request(
    meter_id_or_slug: str,
    *,
    from_parameter: Optional[datetime.datetime] = None,
    to: Optional[datetime.datetime] = None,
    window_size: Optional[str] = None,
    window_time_zone: str = "UTC",
    subject: Optional[List[str]] = None,
    filter_group_by: Optional[Dict[str, str]] = None,
    group_by: Optional[List[str]] = None,
    **kwargs: Any,
) -> HttpRequest:
    _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
    _params = case_insensitive_dict(kwargs.pop("params", {}) or {})

    accept = _headers.pop("Accept", "application/json, text/csv, application/problem+json")

    # Construct URL
    _url = "/api/v1/meters/{meterIdOrSlug}/query"
    path_format_arguments = {
        "meterIdOrSlug": _SERIALIZER.url("meter_id_or_slug", meter_id_or_slug, "str"),
    }

    _url: str = _url.format(**path_format_arguments)  # type: ignore

    # Construct parameters
    if from_parameter is not None:
        _params["from"] = _SERIALIZER.query("from_parameter", from_parameter, "iso-8601")
    if to is not None:
        _params["to"] = _SERIALIZER.query("to", to, "iso-8601")
    if window_size is not None:
        _params["windowSize"] = _SERIALIZER.query("window_size", window_size, "str")
    if window_time_zone is not None:
        _params["windowTimeZone"] = _SERIALIZER.query("window_time_zone", window_time_zone, "str")
    if subject is not None:
        _params["subject"] = _SERIALIZER.query("subject", subject, "[str]")
    if filter_group_by is not None:
        for key in filter_group_by:
            _params["filterGroupBy[{}]".format(key)] = _SERIALIZER.query("filter_group_by", filter_group_by[key], "str")
    if group_by is not None:
        _params["groupBy"] = _SERIALIZER.query("group_by", group_by, "[str]")

    # Construct headers
    _headers["Accept"] = _SERIALIZER.header("accept", accept, "str")

    return HttpRequest(method="GET", url=_url, params=_params, headers=_headers, **kwargs)


def build_query_portal_meter_request(
    meter_slug: str,
    *,
    from_parameter: Optional[datetime.datetime] = None,
    to: Optional[datetime.datetime] = None,
    window_size: Optional[str] = None,
    window_time_zone: str = "UTC",
    filter_group_by: Optional[Dict[str, str]] = None,
    group_by: Optional[List[str]] = None,
    **kwargs: Any,
) -> HttpRequest:
    _headers = case_insensitive_dict(kwargs.pop("headers", {}) or {})
    _params = case_insensitive_dict(kwargs.pop("params", {}) or {})

    accept = _headers.pop("Accept", "application/json, text/csv, application/problem+json")

    # Construct URL
    _url = "/api/v1/portal/meters/{meterSlug}/query"
    path_format_arguments = {
        "meterSlug": _SERIALIZER.url("meter_slug", meter_slug, "str"),
    }

    _url: str = _url.format(**path_format_arguments)  # type: ignore

    # Construct parameters
    if from_parameter is not None:
        _params["from"] = _SERIALIZER.query("from_parameter", from_parameter, "iso-8601")
    if to is not None:
        _params["to"] = _SERIALIZER.query("to", to, "iso-8601")
    if window_size is not None:
        _params["windowSize"] = _SERIALIZER.query("window_size", window_size, "str")
    if window_time_zone is not None:
        _params["windowTimeZone"] = _SERIALIZER.query("window_time_zone", window_time_zone, "str")
    if filter_group_by is not None:
        for key in filter_group_by:
            _params["filterGroupBy[{}]".format(key)] = _SERIALIZER.query("filter_group_by", filter_group_by[key], "str")
    if group_by is not None:
        _params["groupBy"] = _SERIALIZER.query("group_by", group_by, "[str]")

    # Construct headers
    _headers["Accept"] = _SERIALIZER.header("accept", accept, "str")

    return HttpRequest(method="GET", url=_url, params=_params, headers=_headers, **kwargs)
