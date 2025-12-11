# pylint: disable=too-many-lines
# coding=utf-8
# pylint: disable=useless-super-delegation

import datetime
from typing import Any, Literal, Mapping, Optional, TYPE_CHECKING, Union, overload

from .._utils.model_base import Model as _Model, rest_field
from ._enums import (
    AppType,
    BillingCollectionAlignment,
    DiscountReasonType,
    EditOp,
    EntitlementType,
    InvoiceDocumentRefType,
    InvoiceLineTypes,
    NotificationChannelType,
    NotificationEventType,
    PaymentTermType,
    PriceType,
    RateCardType,
    VoidInvoiceLineActionType,
)

if TYPE_CHECKING:
    from .. import _types, models as _models


class Addon(_Model):
    """Add-on allows extending subscriptions with compatible plans with additional ratecards.

    :ivar id: ID. Required.
    :vartype id: str
    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar created_at: Creation Time. Required.
    :vartype created_at: ~datetime.datetime
    :ivar updated_at: Last Update Time. Required.
    :vartype updated_at: ~datetime.datetime
    :ivar deleted_at: Deletion Time.
    :vartype deleted_at: ~datetime.datetime
    :ivar key: Key. Required.
    :vartype key: str
    :ivar annotations: Annotations.
    :vartype annotations: ~openmeter._generated.models.Annotations
    :ivar version: Version. Required.
    :vartype version: int
    :ivar instance_type: InstanceType. Required. Known values are: "single" and "multiple".
    :vartype instance_type: str or ~openmeter.models.AddonInstanceType
    :ivar currency: Currency. Required.
    :vartype currency: str
    :ivar effective_from: Effective start date.
    :vartype effective_from: ~datetime.datetime
    :ivar effective_to: Effective end date.
    :vartype effective_to: ~datetime.datetime
    :ivar status: Status. Required. Known values are: "draft", "active", and "archived".
    :vartype status: str or ~openmeter.models.AddonStatus
    :ivar rate_cards: Rate cards. Required.
    :vartype rate_cards: list[~openmeter._generated.models.RateCardFlatFee or
     ~openmeter._generated.models.RateCardUsageBased]
    :ivar validation_errors: Validation errors. Required.
    :vartype validation_errors: list[~openmeter._generated.models.ValidationError]
    """

    id: str = rest_field(visibility=["read"])
    """ID. Required."""
    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Display name. Required."""
    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    created_at: datetime.datetime = rest_field(name="createdAt", visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    updated_at: datetime.datetime = rest_field(name="updatedAt", visibility=["read"], format="rfc3339")
    """Last Update Time. Required."""
    deleted_at: Optional[datetime.datetime] = rest_field(name="deletedAt", visibility=["read"], format="rfc3339")
    """Deletion Time."""
    key: str = rest_field(visibility=["read", "create"])
    """Key. Required."""
    annotations: Optional["_models.Annotations"] = rest_field(visibility=["read"])
    """Annotations."""
    version: int = rest_field(visibility=["read"])
    """Version. Required."""
    instance_type: Union[str, "_models.AddonInstanceType"] = rest_field(
        name="instanceType", visibility=["read", "create", "update"]
    )
    """InstanceType. Required. Known values are: \"single\" and \"multiple\"."""
    currency: str = rest_field(visibility=["read", "create"])
    """Currency. Required."""
    effective_from: Optional[datetime.datetime] = rest_field(
        name="effectiveFrom", visibility=["read"], format="rfc3339"
    )
    """Effective start date."""
    effective_to: Optional[datetime.datetime] = rest_field(name="effectiveTo", visibility=["read"], format="rfc3339")
    """Effective end date."""
    status: Union[str, "_models.AddonStatus"] = rest_field(visibility=["read"])
    """Status. Required. Known values are: \"draft\", \"active\", and \"archived\"."""
    rate_cards: list["_types.RateCard"] = rest_field(name="rateCards", visibility=["read", "create", "update"])
    """Rate cards. Required."""
    validation_errors: list["_models.ValidationError"] = rest_field(name="validationErrors", visibility=["read"])
    """Validation errors. Required."""

    @overload
    def __init__(
        self,
        *,
        name: str,
        key: str,
        instance_type: Union[str, "_models.AddonInstanceType"],
        currency: str,
        rate_cards: list["_types.RateCard"],
        description: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class AddonCreate(_Model):
    """Resource create operation model.

    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar key: Key. Required.
    :vartype key: str
    :ivar instance_type: InstanceType. Required. Known values are: "single" and "multiple".
    :vartype instance_type: str or ~openmeter.models.AddonInstanceType
    :ivar currency: Currency. Required.
    :vartype currency: str
    :ivar rate_cards: Rate cards. Required.
    :vartype rate_cards: list[~openmeter._generated.models.RateCardFlatFee or
     ~openmeter._generated.models.RateCardUsageBased]
    """

    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Display name. Required."""
    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    key: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Key. Required."""
    instance_type: Union[str, "_models.AddonInstanceType"] = rest_field(
        name="instanceType", visibility=["read", "create", "update", "delete", "query"]
    )
    """InstanceType. Required. Known values are: \"single\" and \"multiple\"."""
    currency: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Currency. Required."""
    rate_cards: list["_types.RateCard"] = rest_field(
        name="rateCards", visibility=["read", "create", "update", "delete", "query"]
    )
    """Rate cards. Required."""

    @overload
    def __init__(
        self,
        *,
        name: str,
        key: str,
        instance_type: Union[str, "_models.AddonInstanceType"],
        currency: str,
        rate_cards: list["_types.RateCard"],
        description: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class AddonReplaceUpdate(_Model):
    """Resource update operation model.

    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar instance_type: InstanceType. Required. Known values are: "single" and "multiple".
    :vartype instance_type: str or ~openmeter.models.AddonInstanceType
    :ivar rate_cards: Rate cards. Required.
    :vartype rate_cards: list[~openmeter._generated.models.RateCardFlatFee or
     ~openmeter._generated.models.RateCardUsageBased]
    """

    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Display name. Required."""
    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    instance_type: Union[str, "_models.AddonInstanceType"] = rest_field(
        name="instanceType", visibility=["read", "create", "update"]
    )
    """InstanceType. Required. Known values are: \"single\" and \"multiple\"."""
    rate_cards: list["_types.RateCard"] = rest_field(name="rateCards", visibility=["read", "create", "update"])
    """Rate cards. Required."""

    @overload
    def __init__(
        self,
        *,
        name: str,
        instance_type: Union[str, "_models.AddonInstanceType"],
        rate_cards: list["_types.RateCard"],
        description: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class Address(_Model):
    """Address.

    :ivar country: Country code in `ISO 3166-1 <https://www.iso.org/iso-3166-country-codes.html>`_
     alpha-2 format.
    :vartype country: str
    :ivar postal_code: Postal code.
    :vartype postal_code: str
    :ivar state: State or province.
    :vartype state: str
    :ivar city: City.
    :vartype city: str
    :ivar line1: First line of the address.
    :vartype line1: str
    :ivar line2: Second line of the address.
    :vartype line2: str
    :ivar phone_number: Phone number.
    :vartype phone_number: str
    """

    country: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Country code in `ISO 3166-1 <https://www.iso.org/iso-3166-country-codes.html>`_ alpha-2 format."""
    postal_code: Optional[str] = rest_field(
        name="postalCode", visibility=["read", "create", "update", "delete", "query"]
    )
    """Postal code."""
    state: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """State or province."""
    city: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """City."""
    line1: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """First line of the address."""
    line2: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Second line of the address."""
    phone_number: Optional[str] = rest_field(
        name="phoneNumber", visibility=["read", "create", "update", "delete", "query"]
    )
    """Phone number."""

    @overload
    def __init__(
        self,
        *,
        country: Optional[str] = None,
        postal_code: Optional[str] = None,
        state: Optional[str] = None,
        city: Optional[str] = None,
        line1: Optional[str] = None,
        line2: Optional[str] = None,
        phone_number: Optional[str] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class Alignment(_Model):
    """Alignment configuration for a plan or subscription.

    :ivar billables_must_align: Whether all Billable items and RateCards must align.
     Alignment means the Price's BillingCadence must align for both duration and anchor time.
    :vartype billables_must_align: bool
    """

    billables_must_align: Optional[bool] = rest_field(
        name="billablesMustAlign", visibility=["read", "create", "update"]
    )
    """Whether all Billable items and RateCards must align.
     Alignment means the Price's BillingCadence must align for both duration and anchor time."""

    @overload
    def __init__(
        self,
        *,
        billables_must_align: Optional[bool] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class Annotations(_Model):
    """Set of key-value pairs managed by the system. Cannot be modified by user."""


class AppCapability(_Model):
    """App capability.

    Capabilities only exist in config so they don't extend the Resource model.

    :ivar type: The capability type. Required. Known values are: "reportUsage", "reportEvents",
     "calculateTax", "invoiceCustomers", and "collectPayments".
    :vartype type: str or ~openmeter.models.AppCapabilityType
    :ivar key: Key. Required.
    :vartype key: str
    :ivar name: The capability name. Required.
    :vartype name: str
    :ivar description: The capability description. Required.
    :vartype description: str
    """

    type: Union[str, "_models.AppCapabilityType"] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """The capability type. Required. Known values are: \"reportUsage\", \"reportEvents\",
     \"calculateTax\", \"invoiceCustomers\", and \"collectPayments\"."""
    key: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Key. Required."""
    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The capability name. Required."""
    description: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The capability description. Required."""

    @overload
    def __init__(
        self,
        *,
        type: Union[str, "_models.AppCapabilityType"],
        key: str,
        name: str,
        description: str,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class AppPaginatedResponse(_Model):
    """Paginated response.

    :ivar total_count: The total number of items. Required.
    :vartype total_count: int
    :ivar page: The page index. Required.
    :vartype page: int
    :ivar page_size: The maximum number of items per page. Required.
    :vartype page_size: int
    :ivar items_property: The items in the current page. Required.
    :vartype items_property: list[~openmeter._generated.models.StripeApp or
     ~openmeter._generated.models.SandboxApp or ~openmeter._generated.models.CustomInvoicingApp]
    """

    total_count: int = rest_field(name="totalCount", visibility=["read", "create", "update", "delete", "query"])
    """The total number of items. Required."""
    page: int = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The page index. Required."""
    page_size: int = rest_field(name="pageSize", visibility=["read", "create", "update", "delete", "query"])
    """The maximum number of items per page. Required."""
    items_property: list["_types.App"] = rest_field(
        name="items", visibility=["read", "create", "update", "delete", "query"]
    )
    """The items in the current page. Required."""

    @overload
    def __init__(
        self,
        *,
        total_count: int,
        page: int,
        page_size: int,
        items_property: list["_types.App"],
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class AppReference(_Model):
    """App reference

    Can be used as a short reference to an app if the full app object is not needed.

    :ivar id: The ID of the app. Required.
    :vartype id: str
    """

    id: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The ID of the app. Required."""

    @overload
    def __init__(
        self,
        *,
        id: str,  # pylint: disable=redefined-builtin
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class UnexpectedProblemResponse(_Model):
    """A Problem Details object (RFC 7807).
    Additional properties specific to the problem type may be present.

    :ivar type: Type contains a URI that identifies the problem type. Required.
    :vartype type: str
    :ivar title: A a short, human-readable summary of the problem type. Required.
    :vartype title: str
    :ivar status: The HTTP status code generated by the origin server for this occurrence of the
     problem.
    :vartype status: int
    :ivar detail: A human-readable explanation specific to this occurrence of the problem.
     Required.
    :vartype detail: str
    :ivar instance: A URI reference that identifies the specific occurrence of the problem.
     Required.
    :vartype instance: str
    :ivar extensions: Additional properties specific to the problem type may be present.
    :vartype extensions: dict[str, any]
    """

    type: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Type contains a URI that identifies the problem type. Required."""
    title: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """A a short, human-readable summary of the problem type. Required."""
    status: Optional[int] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The HTTP status code generated by the origin server for this occurrence of the problem."""
    detail: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """A human-readable explanation specific to this occurrence of the problem. Required."""
    instance: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """A URI reference that identifies the specific occurrence of the problem. Required."""
    extensions: Optional[dict[str, Any]] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Additional properties specific to the problem type may be present."""

    @overload
    def __init__(
        self,
        *,
        type: str,
        title: str,
        detail: str,
        instance: str,
        status: Optional[int] = None,
        extensions: Optional[dict[str, Any]] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class BadRequestProblemResponse(UnexpectedProblemResponse):
    """The server cannot or will not process the request due to something that is perceived to be a
    client error (e.g., malformed request syntax, invalid request message framing, or deceptive
    request routing).

    :ivar type: Type contains a URI that identifies the problem type. Required.
    :vartype type: str
    :ivar title: A a short, human-readable summary of the problem type. Required.
    :vartype title: str
    :ivar status: The HTTP status code generated by the origin server for this occurrence of the
     problem.
    :vartype status: int
    :ivar detail: A human-readable explanation specific to this occurrence of the problem.
     Required.
    :vartype detail: str
    :ivar instance: A URI reference that identifies the specific occurrence of the problem.
     Required.
    :vartype instance: str
    :ivar extensions: Additional properties specific to the problem type may be present.
    :vartype extensions: dict[str, any]
    """

    @overload
    def __init__(
        self,
        *,
        type: str,
        title: str,
        detail: str,
        instance: str,
        status: Optional[int] = None,
        extensions: Optional[dict[str, Any]] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class BalanceHistoryWindow(_Model):
    """The balance history window.

    :ivar period: Required.
    :vartype period: ~openmeter._generated.models.Period
    :ivar usage: The total usage of the feature in the period. Required.
    :vartype usage: float
    :ivar balance_at_start: The entitlement balance at the start of the period. Required.
    :vartype balance_at_start: float
    """

    period: "_models.Period" = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Required."""
    usage: float = rest_field(visibility=["read"])
    """The total usage of the feature in the period. Required."""
    balance_at_start: float = rest_field(name="balanceAtStart", visibility=["read"])
    """The entitlement balance at the start of the period. Required."""

    @overload
    def __init__(
        self,
        *,
        period: "_models.Period",
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class BillingCustomerProfile(_Model):
    """Customer specific merged profile.

    This profile is calculated from the customer override and the billing profile it references or
    the default.

    Thus this does not have any kind of resource fields, only the calculated values.

    :ivar supplier: The name and contact information for the supplier this billing profile
     represents. Required.
    :vartype supplier: ~openmeter._generated.models.BillingParty
    :ivar workflow: The billing workflow settings for this profile. Required.
    :vartype workflow: ~openmeter._generated.models.BillingWorkflow
    :ivar apps: The applications used by this billing profile.

     Expand settings govern if this includes the whole app object or just the ID references.
     Required. Is either a BillingProfileApps type or a BillingProfileAppReferences type.
    :vartype apps: ~openmeter._generated.models.BillingProfileApps or
     ~openmeter._generated.models.BillingProfileAppReferences
    """

    supplier: "_models.BillingParty" = rest_field(visibility=["read"])
    """The name and contact information for the supplier this billing profile represents. Required."""
    workflow: "_models.BillingWorkflow" = rest_field(visibility=["read"])
    """The billing workflow settings for this profile. Required."""
    apps: "_types.BillingProfileAppsOrReference" = rest_field(visibility=["read"])
    """The applications used by this billing profile.
     
     Expand settings govern if this includes the whole app object or just the ID references.
     Required. Is either a BillingProfileApps type or a BillingProfileAppReferences type."""


class BillingDiscountPercentage(_Model):
    """A percentage discount.

    :ivar percentage: Percentage. Required.
    :vartype percentage: float
    :ivar correlation_id: Correlation ID for the discount.

     This is used to link discounts across different invoices (progressive billing use case).

     If not provided, the invoicing engine will auto-generate one. When editing an invoice line,
     please make sure to keep the same correlation ID of the discount or in progressive billing
     setups the discount amounts might be incorrect.
    :vartype correlation_id: str
    """

    percentage: float = rest_field(visibility=["read", "create", "update"])
    """Percentage. Required."""
    correlation_id: Optional[str] = rest_field(
        name="correlationId", visibility=["read", "create", "update", "delete", "query"]
    )
    """Correlation ID for the discount.
     
     This is used to link discounts across different invoices (progressive billing use case).
     
     If not provided, the invoicing engine will auto-generate one. When editing an invoice line,
     please make sure to keep the same correlation ID of the discount or in progressive billing
     setups the discount amounts might be incorrect."""

    @overload
    def __init__(
        self,
        *,
        percentage: float,
        correlation_id: Optional[str] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class BillingDiscounts(_Model):
    """A discount by type.

    :ivar percentage: The percentage discount.
    :vartype percentage: ~openmeter._generated.models.BillingDiscountPercentage
    :ivar usage: The usage discount.
    :vartype usage: ~openmeter._generated.models.BillingDiscountUsage
    """

    percentage: Optional["_models.BillingDiscountPercentage"] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """The percentage discount."""
    usage: Optional["_models.BillingDiscountUsage"] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """The usage discount."""

    @overload
    def __init__(
        self,
        *,
        percentage: Optional["_models.BillingDiscountPercentage"] = None,
        usage: Optional["_models.BillingDiscountUsage"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class BillingDiscountUsage(_Model):
    """A usage discount.

    :ivar quantity: Usage. Required.
    :vartype quantity: str
    :ivar correlation_id: Correlation ID for the discount.

     This is used to link discounts across different invoices (progressive billing use case).

     If not provided, the invoicing engine will auto-generate one. When editing an invoice line,
     please make sure to keep the same correlation ID of the discount or in progressive billing
     setups the discount amounts might be incorrect.
    :vartype correlation_id: str
    """

    quantity: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Usage. Required."""
    correlation_id: Optional[str] = rest_field(
        name="correlationId", visibility=["read", "create", "update", "delete", "query"]
    )
    """Correlation ID for the discount.
     
     This is used to link discounts across different invoices (progressive billing use case).
     
     If not provided, the invoicing engine will auto-generate one. When editing an invoice line,
     please make sure to keep the same correlation ID of the discount or in progressive billing
     setups the discount amounts might be incorrect."""

    @overload
    def __init__(
        self,
        *,
        quantity: str,
        correlation_id: Optional[str] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class BillingInvoiceCustomerExtendedDetails(_Model):
    """BillingInvoiceCustomerExtendedDetails is a collection of fields that are used to extend the
    billing party details for invoices.

    These fields contain the OpenMeter specific details for the customer, that are not strictly
    required for the invoice itself.

    :ivar id: Unique identifier for the party (if available).
    :vartype id: str
    :ivar key: Key.
    :vartype key: str
    :ivar name: Legal name or representation of the organization.
    :vartype name: str
    :ivar tax_id: The entity's legal ID code used for tax purposes. They may have
     other numbers, but we're only interested in those valid for tax purposes.
    :vartype tax_id: ~openmeter._generated.models.BillingPartyTaxIdentity
    :ivar addresses: Regular post addresses for where information should be sent if needed.
    :vartype addresses: list[~openmeter._generated.models.Address]
    :ivar usage_attribution: Usage Attribution. Required.
    :vartype usage_attribution: ~openmeter._generated.models.CustomerUsageAttribution
    """

    id: Optional[str] = rest_field(visibility=["read"])
    """Unique identifier for the party (if available)."""
    key: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Key."""
    name: Optional[str] = rest_field(visibility=["read", "create", "update"])
    """Legal name or representation of the organization."""
    tax_id: Optional["_models.BillingPartyTaxIdentity"] = rest_field(
        name="taxId", visibility=["read", "create", "update"]
    )
    """The entity's legal ID code used for tax purposes. They may have
     other numbers, but we're only interested in those valid for tax purposes."""
    addresses: Optional[list["_models.Address"]] = rest_field(visibility=["read", "create", "update"])
    """Regular post addresses for where information should be sent if needed."""
    usage_attribution: "_models.CustomerUsageAttribution" = rest_field(
        name="usageAttribution", visibility=["read", "create", "update", "delete", "query"]
    )
    """Usage Attribution. Required."""

    @overload
    def __init__(
        self,
        *,
        usage_attribution: "_models.CustomerUsageAttribution",
        key: Optional[str] = None,
        name: Optional[str] = None,
        tax_id: Optional["_models.BillingPartyTaxIdentity"] = None,
        addresses: Optional[list["_models.Address"]] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class BillingParty(_Model):
    """Party represents a person or business entity.

    :ivar id: Unique identifier for the party (if available).
    :vartype id: str
    :ivar key: Key.
    :vartype key: str
    :ivar name: Legal name or representation of the organization.
    :vartype name: str
    :ivar tax_id: The entity's legal ID code used for tax purposes. They may have
     other numbers, but we're only interested in those valid for tax purposes.
    :vartype tax_id: ~openmeter._generated.models.BillingPartyTaxIdentity
    :ivar addresses: Regular post addresses for where information should be sent if needed.
    :vartype addresses: list[~openmeter._generated.models.Address]
    """

    id: Optional[str] = rest_field(visibility=["read"])
    """Unique identifier for the party (if available)."""
    key: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Key."""
    name: Optional[str] = rest_field(visibility=["read", "create", "update"])
    """Legal name or representation of the organization."""
    tax_id: Optional["_models.BillingPartyTaxIdentity"] = rest_field(
        name="taxId", visibility=["read", "create", "update"]
    )
    """The entity's legal ID code used for tax purposes. They may have
     other numbers, but we're only interested in those valid for tax purposes."""
    addresses: Optional[list["_models.Address"]] = rest_field(visibility=["read", "create", "update"])
    """Regular post addresses for where information should be sent if needed."""

    @overload
    def __init__(
        self,
        *,
        key: Optional[str] = None,
        name: Optional[str] = None,
        tax_id: Optional["_models.BillingPartyTaxIdentity"] = None,
        addresses: Optional[list["_models.Address"]] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class BillingPartyReplaceUpdate(_Model):
    """Resource update operation model.

    :ivar key: Key.
    :vartype key: str
    :ivar name: Legal name or representation of the organization.
    :vartype name: str
    :ivar tax_id: The entity's legal ID code used for tax purposes. They may have
     other numbers, but we're only interested in those valid for tax purposes.
    :vartype tax_id: ~openmeter._generated.models.BillingPartyTaxIdentity
    :ivar addresses: Regular post addresses for where information should be sent if needed.
    :vartype addresses: list[~openmeter._generated.models.Address]
    """

    key: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Key."""
    name: Optional[str] = rest_field(visibility=["read", "create", "update"])
    """Legal name or representation of the organization."""
    tax_id: Optional["_models.BillingPartyTaxIdentity"] = rest_field(
        name="taxId", visibility=["read", "create", "update"]
    )
    """The entity's legal ID code used for tax purposes. They may have
     other numbers, but we're only interested in those valid for tax purposes."""
    addresses: Optional[list["_models.Address"]] = rest_field(visibility=["read", "create", "update"])
    """Regular post addresses for where information should be sent if needed."""

    @overload
    def __init__(
        self,
        *,
        key: Optional[str] = None,
        name: Optional[str] = None,
        tax_id: Optional["_models.BillingPartyTaxIdentity"] = None,
        addresses: Optional[list["_models.Address"]] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class BillingPartyTaxIdentity(_Model):
    """Identity stores the details required to identify an entity for tax purposes in a specific
    country.

    :ivar code: Normalized tax code shown on the original identity document.
    :vartype code: str
    """

    code: Optional[str] = rest_field(visibility=["read", "create", "update"])
    """Normalized tax code shown on the original identity document."""

    @overload
    def __init__(
        self,
        *,
        code: Optional[str] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class BillingProfile(_Model):
    """BillingProfile represents a billing profile.

    :ivar id: ID. Required.
    :vartype id: str
    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar created_at: Creation Time. Required.
    :vartype created_at: ~datetime.datetime
    :ivar updated_at: Last Update Time. Required.
    :vartype updated_at: ~datetime.datetime
    :ivar deleted_at: Deletion Time.
    :vartype deleted_at: ~datetime.datetime
    :ivar supplier: The name and contact information for the supplier this billing profile
     represents. Required.
    :vartype supplier: ~openmeter._generated.models.BillingParty
    :ivar workflow: The billing workflow settings for this profile. Required.
    :vartype workflow: ~openmeter._generated.models.BillingWorkflow
    :ivar apps: The applications used by this billing profile.

     Expand settings govern if this includes the whole app object or just the ID references.
     Required. Is either a BillingProfileApps type or a BillingProfileAppReferences type.
    :vartype apps: ~openmeter._generated.models.BillingProfileApps or
     ~openmeter._generated.models.BillingProfileAppReferences
    :ivar default: Is this the default profile?. Required.
    :vartype default: bool
    """

    id: str = rest_field(visibility=["read"])
    """ID. Required."""
    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Display name. Required."""
    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    created_at: datetime.datetime = rest_field(name="createdAt", visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    updated_at: datetime.datetime = rest_field(name="updatedAt", visibility=["read"], format="rfc3339")
    """Last Update Time. Required."""
    deleted_at: Optional[datetime.datetime] = rest_field(name="deletedAt", visibility=["read"], format="rfc3339")
    """Deletion Time."""
    supplier: "_models.BillingParty" = rest_field(visibility=["read", "create", "update"])
    """The name and contact information for the supplier this billing profile represents. Required."""
    workflow: "_models.BillingWorkflow" = rest_field(visibility=["read"])
    """The billing workflow settings for this profile. Required."""
    apps: "_types.BillingProfileAppsOrReference" = rest_field(visibility=["read"])
    """The applications used by this billing profile.
     
     Expand settings govern if this includes the whole app object or just the ID references.
     Required. Is either a BillingProfileApps type or a BillingProfileAppReferences type."""
    default: bool = rest_field(visibility=["read", "create", "update"])
    """Is this the default profile?. Required."""

    @overload
    def __init__(
        self,
        *,
        name: str,
        supplier: "_models.BillingParty",
        default: bool,
        description: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class BillingProfileAppReferences(_Model):
    """BillingProfileAppReferences represents the references (id, type) to the apps used by a billing
    profile.

    :ivar tax: The tax app used for this workflow. Required.
    :vartype tax: ~openmeter._generated.models.AppReference
    :ivar invoicing: The invoicing app used for this workflow. Required.
    :vartype invoicing: ~openmeter._generated.models.AppReference
    :ivar payment: The payment app used for this workflow. Required.
    :vartype payment: ~openmeter._generated.models.AppReference
    """

    tax: "_models.AppReference" = rest_field(visibility=["read"])
    """The tax app used for this workflow. Required."""
    invoicing: "_models.AppReference" = rest_field(visibility=["read"])
    """The invoicing app used for this workflow. Required."""
    payment: "_models.AppReference" = rest_field(visibility=["read"])
    """The payment app used for this workflow. Required."""


class BillingProfileApps(_Model):
    """BillingProfileApps represents the applications used by a billing profile.

    :ivar tax: The tax app used for this workflow. Required. Is one of the following types:
     StripeApp, SandboxApp, CustomInvoicingApp
    :vartype tax: ~openmeter._generated.models.StripeApp or ~openmeter._generated.models.SandboxApp
     or ~openmeter._generated.models.CustomInvoicingApp
    :ivar invoicing: The invoicing app used for this workflow. Required. Is one of the following
     types: StripeApp, SandboxApp, CustomInvoicingApp
    :vartype invoicing: ~openmeter._generated.models.StripeApp or
     ~openmeter._generated.models.SandboxApp or ~openmeter._generated.models.CustomInvoicingApp
    :ivar payment: The payment app used for this workflow. Required. Is one of the following types:
     StripeApp, SandboxApp, CustomInvoicingApp
    :vartype payment: ~openmeter._generated.models.StripeApp or
     ~openmeter._generated.models.SandboxApp or ~openmeter._generated.models.CustomInvoicingApp
    """

    tax: "_types.App" = rest_field(visibility=["read"])
    """The tax app used for this workflow. Required. Is one of the following types: StripeApp,
     SandboxApp, CustomInvoicingApp"""
    invoicing: "_types.App" = rest_field(visibility=["read"])
    """The invoicing app used for this workflow. Required. Is one of the following types: StripeApp,
     SandboxApp, CustomInvoicingApp"""
    payment: "_types.App" = rest_field(visibility=["read"])
    """The payment app used for this workflow. Required. Is one of the following types: StripeApp,
     SandboxApp, CustomInvoicingApp"""


class BillingProfileAppsCreate(_Model):
    """BillingProfileAppsCreate represents the input for creating a billing profile's apps.

    :ivar tax: The tax app used for this workflow. Required.
    :vartype tax: str
    :ivar invoicing: The invoicing app used for this workflow. Required.
    :vartype invoicing: str
    :ivar payment: The payment app used for this workflow. Required.
    :vartype payment: str
    """

    tax: str = rest_field(visibility=["create"])
    """The tax app used for this workflow. Required."""
    invoicing: str = rest_field(visibility=["create"])
    """The invoicing app used for this workflow. Required."""
    payment: str = rest_field(visibility=["create"])
    """The payment app used for this workflow. Required."""

    @overload
    def __init__(
        self,
        *,
        tax: str,
        invoicing: str,
        payment: str,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class BillingProfileCreate(_Model):
    """BillingProfileCreate represents the input for creating a billing profile.

    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar supplier: The name and contact information for the supplier this billing profile
     represents. Required.
    :vartype supplier: ~openmeter._generated.models.BillingParty
    :ivar default: Is this the default profile?. Required.
    :vartype default: bool
    :ivar workflow: The billing workflow settings for this profile. Required.
    :vartype workflow: ~openmeter._generated.models.BillingWorkflowCreate
    :ivar apps: The apps used by this billing profile. Required.
    :vartype apps: ~openmeter._generated.models.BillingProfileAppsCreate
    """

    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Display name. Required."""
    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    supplier: "_models.BillingParty" = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The name and contact information for the supplier this billing profile represents. Required."""
    default: bool = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Is this the default profile?. Required."""
    workflow: "_models.BillingWorkflowCreate" = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The billing workflow settings for this profile. Required."""
    apps: "_models.BillingProfileAppsCreate" = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The apps used by this billing profile. Required."""

    @overload
    def __init__(
        self,
        *,
        name: str,
        supplier: "_models.BillingParty",
        default: bool,
        workflow: "_models.BillingWorkflowCreate",
        apps: "_models.BillingProfileAppsCreate",
        description: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class BillingProfileCustomerOverride(_Model):
    """Customer override values.

    :ivar created_at: Creation Time. Required.
    :vartype created_at: ~datetime.datetime
    :ivar updated_at: Last Update Time. Required.
    :vartype updated_at: ~datetime.datetime
    :ivar billing_profile_id: The billing profile this override is associated with.

     If empty the default profile is looked up dynamically.
    :vartype billing_profile_id: str
    :ivar customer_id: The customer id this override is associated with. Required.
    :vartype customer_id: str
    """

    created_at: datetime.datetime = rest_field(name="createdAt", visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    updated_at: datetime.datetime = rest_field(name="updatedAt", visibility=["read"], format="rfc3339")
    """Last Update Time. Required."""
    billing_profile_id: Optional[str] = rest_field(
        name="billingProfileId", visibility=["read", "create", "update", "delete", "query"]
    )
    """The billing profile this override is associated with.
     
     If empty the default profile is looked up dynamically."""
    customer_id: str = rest_field(name="customerId", visibility=["read", "create", "update", "delete", "query"])
    """The customer id this override is associated with. Required."""

    @overload
    def __init__(
        self,
        *,
        customer_id: str,
        billing_profile_id: Optional[str] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class BillingProfileCustomerOverrideCreate(_Model):
    """Payload for creating a new or updating an existing customer override.

    :ivar billing_profile_id: The billing profile this override is associated with.

     If not provided, the default billing profile is chosen if available.
    :vartype billing_profile_id: str
    """

    billing_profile_id: Optional[str] = rest_field(
        name="billingProfileId", visibility=["read", "create", "update", "delete", "query"]
    )
    """The billing profile this override is associated with.
     
     If not provided, the default billing profile is chosen if available."""

    @overload
    def __init__(
        self,
        *,
        billing_profile_id: Optional[str] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class BillingProfileCustomerOverrideWithDetails(_Model):  # pylint: disable=name-too-long
    """Customer specific workflow overrides.

    :ivar customer_override: The customer override values.

     If empty the merged values are calculated based on the default profile.
    :vartype customer_override: ~openmeter._generated.models.BillingProfileCustomerOverride
    :ivar base_billing_profile_id: The billing profile the customerProfile is associated with at
     the time of query.

     customerOverride contains the explicit mapping set in the customer override object. If that is
     empty, then the baseBillingProfileId is the default profile. Required.
    :vartype base_billing_profile_id: str
    :ivar customer_profile: Merged billing profile with the customer specific overrides.
    :vartype customer_profile: ~openmeter._generated.models.BillingCustomerProfile
    :ivar customer: The customer this override belongs to.
    :vartype customer: ~openmeter._generated.models.Customer
    """

    customer_override: Optional["_models.BillingProfileCustomerOverride"] = rest_field(
        name="customerOverride", visibility=["read", "create", "update", "delete", "query"]
    )
    """The customer override values.
     
     If empty the merged values are calculated based on the default profile."""
    base_billing_profile_id: str = rest_field(
        name="baseBillingProfileId", visibility=["read", "create", "update", "delete", "query"]
    )
    """The billing profile the customerProfile is associated with at the time of query.
     
     customerOverride contains the explicit mapping set in the customer override object. If that is
     empty, then the baseBillingProfileId is the default profile. Required."""
    customer_profile: Optional["_models.BillingCustomerProfile"] = rest_field(
        name="customerProfile", visibility=["read", "create", "update", "delete", "query"]
    )
    """Merged billing profile with the customer specific overrides."""
    customer: Optional["_models.Customer"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The customer this override belongs to."""

    @overload
    def __init__(
        self,
        *,
        base_billing_profile_id: str,
        customer_override: Optional["_models.BillingProfileCustomerOverride"] = None,
        customer_profile: Optional["_models.BillingCustomerProfile"] = None,
        customer: Optional["_models.Customer"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class BillingProfileCustomerOverrideWithDetailsPaginatedResponse(_Model):  # pylint: disable=name-too-long
    """Paginated response.

    :ivar total_count: The total number of items. Required.
    :vartype total_count: int
    :ivar page: The page index. Required.
    :vartype page: int
    :ivar page_size: The maximum number of items per page. Required.
    :vartype page_size: int
    :ivar items_property: The items in the current page. Required.
    :vartype items_property:
     list[~openmeter._generated.models.BillingProfileCustomerOverrideWithDetails]
    """

    total_count: int = rest_field(name="totalCount", visibility=["read", "create", "update", "delete", "query"])
    """The total number of items. Required."""
    page: int = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The page index. Required."""
    page_size: int = rest_field(name="pageSize", visibility=["read", "create", "update", "delete", "query"])
    """The maximum number of items per page. Required."""
    items_property: list["_models.BillingProfileCustomerOverrideWithDetails"] = rest_field(
        name="items", visibility=["read", "create", "update", "delete", "query"]
    )
    """The items in the current page. Required."""

    @overload
    def __init__(
        self,
        *,
        total_count: int,
        page: int,
        page_size: int,
        items_property: list["_models.BillingProfileCustomerOverrideWithDetails"],
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class BillingProfilePaginatedResponse(_Model):
    """Paginated response.

    :ivar total_count: The total number of items. Required.
    :vartype total_count: int
    :ivar page: The page index. Required.
    :vartype page: int
    :ivar page_size: The maximum number of items per page. Required.
    :vartype page_size: int
    :ivar items_property: The items in the current page. Required.
    :vartype items_property: list[~openmeter._generated.models.BillingProfile]
    """

    total_count: int = rest_field(name="totalCount", visibility=["read", "create", "update", "delete", "query"])
    """The total number of items. Required."""
    page: int = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The page index. Required."""
    page_size: int = rest_field(name="pageSize", visibility=["read", "create", "update", "delete", "query"])
    """The maximum number of items per page. Required."""
    items_property: list["_models.BillingProfile"] = rest_field(
        name="items", visibility=["read", "create", "update", "delete", "query"]
    )
    """The items in the current page. Required."""

    @overload
    def __init__(
        self,
        *,
        total_count: int,
        page: int,
        page_size: int,
        items_property: list["_models.BillingProfile"],
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class BillingProfileReplaceUpdateWithWorkflow(_Model):
    """BillingProfileReplaceUpdate represents the input for updating a billing profile

    The apps field cannot be updated directly, if an app change is desired a new
    profile should be created.

    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar supplier: The name and contact information for the supplier this billing profile
     represents. Required.
    :vartype supplier: ~openmeter._generated.models.BillingParty
    :ivar default: Is this the default profile?. Required.
    :vartype default: bool
    :ivar workflow: The billing workflow settings for this profile. Required.
    :vartype workflow: ~openmeter._generated.models.BillingWorkflow
    """

    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Display name. Required."""
    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    supplier: "_models.BillingParty" = rest_field(visibility=["read", "create", "update"])
    """The name and contact information for the supplier this billing profile represents. Required."""
    default: bool = rest_field(visibility=["read", "create", "update"])
    """Is this the default profile?. Required."""
    workflow: "_models.BillingWorkflow" = rest_field(visibility=["update"])
    """The billing workflow settings for this profile. Required."""

    @overload
    def __init__(
        self,
        *,
        name: str,
        supplier: "_models.BillingParty",
        default: bool,
        workflow: "_models.BillingWorkflow",
        description: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class BillingWorkflow(_Model):
    """BillingWorkflow represents the settings for a billing workflow.

    :ivar collection: The collection settings for this workflow.
    :vartype collection: ~openmeter._generated.models.BillingWorkflowCollectionSettings
    :ivar invoicing: The invoicing settings for this workflow.
    :vartype invoicing: ~openmeter._generated.models.BillingWorkflowInvoicingSettings
    :ivar payment: The payment settings for this workflow.
    :vartype payment: ~openmeter._generated.models.BillingWorkflowPaymentSettings
    :ivar tax: The tax settings for this workflow.
    :vartype tax: ~openmeter._generated.models.BillingWorkflowTaxSettings
    """

    collection: Optional["_models.BillingWorkflowCollectionSettings"] = rest_field(
        visibility=["read", "create", "update"]
    )
    """The collection settings for this workflow."""
    invoicing: Optional["_models.BillingWorkflowInvoicingSettings"] = rest_field(
        visibility=["read", "create", "update"]
    )
    """The invoicing settings for this workflow."""
    payment: Optional["_models.BillingWorkflowPaymentSettings"] = rest_field(visibility=["read", "create", "update"])
    """The payment settings for this workflow."""
    tax: Optional["_models.BillingWorkflowTaxSettings"] = rest_field(visibility=["read", "create", "update"])
    """The tax settings for this workflow."""

    @overload
    def __init__(
        self,
        *,
        collection: Optional["_models.BillingWorkflowCollectionSettings"] = None,
        invoicing: Optional["_models.BillingWorkflowInvoicingSettings"] = None,
        payment: Optional["_models.BillingWorkflowPaymentSettings"] = None,
        tax: Optional["_models.BillingWorkflowTaxSettings"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class BillingWorkflowCollectionAlignmentAnchored(_Model):  # pylint: disable=name-too-long
    """BillingWorkflowCollectionAlignmentAnchored specifies the alignment for collecting the pending
    line items
    into an invoice.

    :ivar type: The type of alignment. Required. Align the collection to the anchor time and
     cadence.
    :vartype type: str or ~openmeter._generated.models.ANCHORED
    :ivar recurring_period: The recurring period for the alignment. Required.
    :vartype recurring_period: ~openmeter._generated.models.RecurringPeriodV2
    """

    type: Literal[BillingCollectionAlignment.ANCHORED] = rest_field(visibility=["read", "create", "update"])
    """The type of alignment. Required. Align the collection to the anchor time and cadence."""
    recurring_period: "_models.RecurringPeriodV2" = rest_field(
        name="recurringPeriod", visibility=["read", "create", "update"]
    )
    """The recurring period for the alignment. Required."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[BillingCollectionAlignment.ANCHORED],
        recurring_period: "_models.RecurringPeriodV2",
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class BillingWorkflowCollectionAlignmentSubscription(_Model):  # pylint: disable=name-too-long
    """BillingWorkflowCollectionAlignmentSubscription specifies the alignment for collecting the
    pending line items
    into an invoice.

    :ivar type: The type of alignment. Required. Align the collection to the start of the
     subscription period.
    :vartype type: str or ~openmeter._generated.models.SUBSCRIPTION
    """

    type: Literal[BillingCollectionAlignment.SUBSCRIPTION] = rest_field(visibility=["read", "create", "update"])
    """The type of alignment. Required. Align the collection to the start of the subscription period."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[BillingCollectionAlignment.SUBSCRIPTION],
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class BillingWorkflowCollectionSettings(_Model):
    """Workflow collection specifies how to collect the pending line items for an invoice.

    :ivar alignment: The alignment for collecting the pending line items into an invoice. Is either
     a BillingWorkflowCollectionAlignmentSubscription type or a
     BillingWorkflowCollectionAlignmentAnchored type.
    :vartype alignment: ~openmeter._generated.models.BillingWorkflowCollectionAlignmentSubscription
     or ~openmeter._generated.models.BillingWorkflowCollectionAlignmentAnchored
    :ivar interval: This grace period can be used to delay the collection of the pending line items
     specified in
     alignment.

     This is useful, in case of multiple subscriptions having slightly different billing periods.
    :vartype interval: str
    """

    alignment: Optional["_types.BillingWorkflowCollectionAlignment"] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """The alignment for collecting the pending line items into an invoice. Is either a
     BillingWorkflowCollectionAlignmentSubscription type or a
     BillingWorkflowCollectionAlignmentAnchored type."""
    interval: Optional[str] = rest_field(visibility=["read", "create", "update"])
    """This grace period can be used to delay the collection of the pending line items specified in
     alignment.
     
     This is useful, in case of multiple subscriptions having slightly different billing periods."""

    @overload
    def __init__(
        self,
        *,
        alignment: Optional["_types.BillingWorkflowCollectionAlignment"] = None,
        interval: Optional[str] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class BillingWorkflowCreate(_Model):
    """Resource create operation model.

    :ivar collection: The collection settings for this workflow.
    :vartype collection: ~openmeter._generated.models.BillingWorkflowCollectionSettings
    :ivar invoicing: The invoicing settings for this workflow.
    :vartype invoicing: ~openmeter._generated.models.BillingWorkflowInvoicingSettings
    :ivar payment: The payment settings for this workflow.
    :vartype payment: ~openmeter._generated.models.BillingWorkflowPaymentSettings
    :ivar tax: The tax settings for this workflow.
    :vartype tax: ~openmeter._generated.models.BillingWorkflowTaxSettings
    """

    collection: Optional["_models.BillingWorkflowCollectionSettings"] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """The collection settings for this workflow."""
    invoicing: Optional["_models.BillingWorkflowInvoicingSettings"] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """The invoicing settings for this workflow."""
    payment: Optional["_models.BillingWorkflowPaymentSettings"] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """The payment settings for this workflow."""
    tax: Optional["_models.BillingWorkflowTaxSettings"] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """The tax settings for this workflow."""

    @overload
    def __init__(
        self,
        *,
        collection: Optional["_models.BillingWorkflowCollectionSettings"] = None,
        invoicing: Optional["_models.BillingWorkflowInvoicingSettings"] = None,
        payment: Optional["_models.BillingWorkflowPaymentSettings"] = None,
        tax: Optional["_models.BillingWorkflowTaxSettings"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class BillingWorkflowInvoicingSettings(_Model):
    """Workflow invoice settings.

    :ivar auto_advance: Whether to automatically issue the invoice after the draftPeriod has
     passed.
    :vartype auto_advance: bool
    :ivar draft_period: The period for the invoice to be kept in draft status for manual reviews.
    :vartype draft_period: str
    :ivar due_after: The period after which the invoice is due.
     With some payment solutions it's only applicable for manual collection method.
    :vartype due_after: str
    :ivar progressive_billing: Should progressive billing be allowed for this workflow?.
    :vartype progressive_billing: bool
    :ivar default_tax_config: Default tax configuration to apply to the invoices.
    :vartype default_tax_config: ~openmeter._generated.models.TaxConfig
    """

    auto_advance: Optional[bool] = rest_field(name="autoAdvance", visibility=["read", "create", "update"])
    """Whether to automatically issue the invoice after the draftPeriod has passed."""
    draft_period: Optional[str] = rest_field(name="draftPeriod", visibility=["read", "create", "update"])
    """The period for the invoice to be kept in draft status for manual reviews."""
    due_after: Optional[str] = rest_field(name="dueAfter", visibility=["read", "create", "update"])
    """The period after which the invoice is due.
     With some payment solutions it's only applicable for manual collection method."""
    progressive_billing: Optional[bool] = rest_field(name="progressiveBilling", visibility=["read", "create", "update"])
    """Should progressive billing be allowed for this workflow?."""
    default_tax_config: Optional["_models.TaxConfig"] = rest_field(
        name="defaultTaxConfig", visibility=["read", "create", "update"]
    )
    """Default tax configuration to apply to the invoices."""

    @overload
    def __init__(
        self,
        *,
        auto_advance: Optional[bool] = None,
        draft_period: Optional[str] = None,
        due_after: Optional[str] = None,
        progressive_billing: Optional[bool] = None,
        default_tax_config: Optional["_models.TaxConfig"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class BillingWorkflowPaymentSettings(_Model):
    """Workflow payment settings.

    :ivar collection_method: The payment method for the invoice. Known values are:
     "charge_automatically" and "send_invoice".
    :vartype collection_method: str or ~openmeter.models.CollectionMethod
    """

    collection_method: Optional[Union[str, "_models.CollectionMethod"]] = rest_field(
        name="collectionMethod", visibility=["read", "create", "update"]
    )
    """The payment method for the invoice. Known values are: \"charge_automatically\" and
     \"send_invoice\"."""

    @overload
    def __init__(
        self,
        *,
        collection_method: Optional[Union[str, "_models.CollectionMethod"]] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class BillingWorkflowTaxSettings(_Model):
    """Workflow tax settings.

    :ivar enabled: Enable automatic tax calculation when tax is supported by the app.
     For example, with Stripe Invoicing when enabled, tax is calculated via Stripe Tax.
    :vartype enabled: bool
    :ivar enforced: Enforce tax calculation when tax is supported by the app.
     When enabled, OpenMeter will not allow to create an invoice without tax calculation.
     Enforcement is different per apps, for example, Stripe app requires customer
     to have a tax location when starting a paid subscription.
    :vartype enforced: bool
    """

    enabled: Optional[bool] = rest_field(visibility=["read", "create", "update"])
    """Enable automatic tax calculation when tax is supported by the app.
     For example, with Stripe Invoicing when enabled, tax is calculated via Stripe Tax."""
    enforced: Optional[bool] = rest_field(visibility=["read", "create", "update"])
    """Enforce tax calculation when tax is supported by the app.
     When enabled, OpenMeter will not allow to create an invoice without tax calculation.
     Enforcement is different per apps, for example, Stripe app requires customer
     to have a tax location when starting a paid subscription."""

    @overload
    def __init__(
        self,
        *,
        enabled: Optional[bool] = None,
        enforced: Optional[bool] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class CancelRequest(_Model):
    """CancelRequest.

    :ivar timing: If not provided the subscription is canceled immediately. Is either a Union[str,
     "_models.SubscriptionTimingEnum"] type or a datetime.datetime type.
    :vartype timing: str or ~openmeter.models.SubscriptionTimingEnum or ~datetime.datetime
    """

    timing: Optional["_types.SubscriptionTiming"] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """If not provided the subscription is canceled immediately. Is either a Union[str,
     \"_models.SubscriptionTimingEnum\"] type or a datetime.datetime type."""

    @overload
    def __init__(
        self,
        *,
        timing: Optional["_types.SubscriptionTiming"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class CheckoutSessionCustomTextAfterSubmitParams(_Model):  # pylint: disable=name-too-long
    """Stripe CheckoutSession.custom_text.

    :ivar after_submit: Custom text that should be displayed after the payment confirmation button.
    :vartype after_submit: ~openmeter._generated.models.CheckoutSessionCustomTextParamsAfterSubmit
    :ivar shipping_address: Custom text that should be displayed alongside shipping address
     collection.
    :vartype shipping_address:
     ~openmeter._generated.models.CheckoutSessionCustomTextParamsShippingAddress
    :ivar submit: Custom text that should be displayed alongside the payment confirmation button.
    :vartype submit: ~openmeter._generated.models.CheckoutSessionCustomTextParamsSubmit
    :ivar terms_of_service_acceptance: Custom text that should be displayed in place of the default
     terms of service agreement text.
    :vartype terms_of_service_acceptance:
     ~openmeter._generated.models.CheckoutSessionCustomTextParamsTermsOfServiceAcceptance
    """

    after_submit: Optional["_models.CheckoutSessionCustomTextParamsAfterSubmit"] = rest_field(
        name="afterSubmit", visibility=["read", "create", "update", "delete", "query"]
    )
    """Custom text that should be displayed after the payment confirmation button."""
    shipping_address: Optional["_models.CheckoutSessionCustomTextParamsShippingAddress"] = rest_field(
        name="shippingAddress", visibility=["read", "create", "update", "delete", "query"]
    )
    """Custom text that should be displayed alongside shipping address collection."""
    submit: Optional["_models.CheckoutSessionCustomTextParamsSubmit"] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """Custom text that should be displayed alongside the payment confirmation button."""
    terms_of_service_acceptance: Optional["_models.CheckoutSessionCustomTextParamsTermsOfServiceAcceptance"] = (
        rest_field(name="termsOfServiceAcceptance", visibility=["read", "create", "update", "delete", "query"])
    )
    """Custom text that should be displayed in place of the default terms of service agreement text."""

    @overload
    def __init__(
        self,
        *,
        after_submit: Optional["_models.CheckoutSessionCustomTextParamsAfterSubmit"] = None,
        shipping_address: Optional["_models.CheckoutSessionCustomTextParamsShippingAddress"] = None,
        submit: Optional["_models.CheckoutSessionCustomTextParamsSubmit"] = None,
        terms_of_service_acceptance: Optional["_models.CheckoutSessionCustomTextParamsTermsOfServiceAcceptance"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class CheckoutSessionCustomTextParamsAfterSubmit(_Model):  # pylint: disable=name-too-long
    """CheckoutSessionCustomTextParamsAfterSubmit.

    :ivar message:
    :vartype message: str
    """

    message: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])

    @overload
    def __init__(
        self,
        *,
        message: Optional[str] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class CheckoutSessionCustomTextParamsShippingAddress(_Model):  # pylint: disable=name-too-long
    """CheckoutSessionCustomTextParamsShippingAddress.

    :ivar message:
    :vartype message: str
    """

    message: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])

    @overload
    def __init__(
        self,
        *,
        message: Optional[str] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class CheckoutSessionCustomTextParamsSubmit(_Model):
    """CheckoutSessionCustomTextParamsSubmit.

    :ivar message:
    :vartype message: str
    """

    message: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])

    @overload
    def __init__(
        self,
        *,
        message: Optional[str] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class CheckoutSessionCustomTextParamsTermsOfServiceAcceptance(_Model):  # pylint: disable=name-too-long
    """CheckoutSessionCustomTextParamsTermsOfServiceAcceptance.

    :ivar message:
    :vartype message: str
    """

    message: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])

    @overload
    def __init__(
        self,
        *,
        message: Optional[str] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class ClientAppStartResponse(_Model):
    """Response from the client app (OpenMeter backend) to start the OAuth2 flow.

    :ivar url: The URL to start the OAuth2 authorization code grant flow. Required.
    :vartype url: str
    """

    url: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The URL to start the OAuth2 authorization code grant flow. Required."""

    @overload
    def __init__(
        self,
        *,
        url: str,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class ConflictProblemResponse(UnexpectedProblemResponse):
    """The request could not be completed due to a conflict with the current state of the target
    resource.

    :ivar type: Type contains a URI that identifies the problem type. Required.
    :vartype type: str
    :ivar title: A a short, human-readable summary of the problem type. Required.
    :vartype title: str
    :ivar status: The HTTP status code generated by the origin server for this occurrence of the
     problem.
    :vartype status: int
    :ivar detail: A human-readable explanation specific to this occurrence of the problem.
     Required.
    :vartype detail: str
    :ivar instance: A URI reference that identifies the specific occurrence of the problem.
     Required.
    :vartype instance: str
    :ivar extensions: Additional properties specific to the problem type may be present.
    :vartype extensions: dict[str, any]
    """

    @overload
    def __init__(
        self,
        *,
        type: str,
        title: str,
        detail: str,
        instance: str,
        status: Optional[int] = None,
        extensions: Optional[dict[str, Any]] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class CreateCheckoutSessionTaxIdCollection(_Model):
    """Create Stripe checkout session tax ID collection.

    :ivar enabled: Enable tax ID collection during checkout. Defaults to false. Required.
    :vartype enabled: bool
    :ivar required: Describes whether a tax ID is required during checkout. Defaults to never.
     Known values are: "if_supported" and "never".
    :vartype required: str or ~openmeter.models.CreateCheckoutSessionTaxIdCollectionRequired
    """

    enabled: bool = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Enable tax ID collection during checkout. Defaults to false. Required."""
    required: Optional[Union[str, "_models.CreateCheckoutSessionTaxIdCollectionRequired"]] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """Describes whether a tax ID is required during checkout. Defaults to never. Known values are:
     \"if_supported\" and \"never\"."""

    @overload
    def __init__(
        self,
        *,
        enabled: bool,
        required: Optional[Union[str, "_models.CreateCheckoutSessionTaxIdCollectionRequired"]] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class CreateResponseExtensions(_Model):
    """CreateResponseExtensions.

    :ivar validation_errors: Required.
    :vartype validation_errors: list[~openmeter._generated.models.ErrorExtension]
    """

    validation_errors: list["_models.ErrorExtension"] = rest_field(
        name="validationErrors", visibility=["read", "create", "update", "delete", "query"]
    )
    """Required."""

    @overload
    def __init__(
        self,
        *,
        validation_errors: list["_models.ErrorExtension"],
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class CreateStripeCheckoutSessionConsentCollection(_Model):  # pylint: disable=name-too-long
    """Configure fields for the Checkout Session to gather active consent from customers.

    :ivar payment_method_reuse_agreement: Determines the position and visibility of the payment
     method reuse agreement in the UI.
     When set to auto, Stripe’s defaults will be used. When set to hidden, the payment method reuse
     agreement text will always be hidden in the UI.
    :vartype payment_method_reuse_agreement:
     ~openmeter._generated.models.CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreement
    :ivar promotions: If set to auto, enables the collection of customer consent for promotional
     communications.
     The Checkout Session will determine whether to display an option to opt into promotional
     communication from the merchant depending on the customer’s locale. Only available to US
     merchants. Known values are: "auto" and "none".
    :vartype promotions: str or
     ~openmeter.models.CreateStripeCheckoutSessionConsentCollectionPromotions
    :ivar terms_of_service: If set to required, it requires customers to check a terms of service
     checkbox before being able to pay.
     There must be a valid terms of service URL set in your Stripe Dashboard settings.
     `https://dashboard.stripe.com/settings/public <https://dashboard.stripe.com/settings/public>`_.
     Known values are: "none" and "required".
    :vartype terms_of_service: str or
     ~openmeter.models.CreateStripeCheckoutSessionConsentCollectionTermsOfService
    """

    payment_method_reuse_agreement: Optional[
        "_models.CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreement"
    ] = rest_field(name="paymentMethodReuseAgreement", visibility=["read", "create", "update", "delete", "query"])
    """Determines the position and visibility of the payment method reuse agreement in the UI.
     When set to auto, Stripe’s defaults will be used. When set to hidden, the payment method reuse
     agreement text will always be hidden in the UI."""
    promotions: Optional[Union[str, "_models.CreateStripeCheckoutSessionConsentCollectionPromotions"]] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """If set to auto, enables the collection of customer consent for promotional communications.
     The Checkout Session will determine whether to display an option to opt into promotional
     communication from the merchant depending on the customer’s locale. Only available to US
     merchants. Known values are: \"auto\" and \"none\"."""
    terms_of_service: Optional[Union[str, "_models.CreateStripeCheckoutSessionConsentCollectionTermsOfService"]] = (
        rest_field(name="termsOfService", visibility=["read", "create", "update", "delete", "query"])
    )
    """If set to required, it requires customers to check a terms of service checkbox before being
     able to pay.
     There must be a valid terms of service URL set in your Stripe Dashboard settings.
     `https://dashboard.stripe.com/settings/public <https://dashboard.stripe.com/settings/public>`_.
     Known values are: \"none\" and \"required\"."""

    @overload
    def __init__(
        self,
        *,
        payment_method_reuse_agreement: Optional[
            "_models.CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreement"
        ] = None,
        promotions: Optional[Union[str, "_models.CreateStripeCheckoutSessionConsentCollectionPromotions"]] = None,
        terms_of_service: Optional[
            Union[str, "_models.CreateStripeCheckoutSessionConsentCollectionTermsOfService"]
        ] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreement(_Model):  # pylint: disable=name-too-long
    """Create Stripe checkout session payment method reuse agreement.

    :ivar position: Known values are: "auto" and "hidden".
    :vartype position: str or
     ~openmeter.models.CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPosition
    """

    position: Optional[
        Union[str, "_models.CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPosition"]
    ] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Known values are: \"auto\" and \"hidden\"."""

    @overload
    def __init__(
        self,
        *,
        position: Optional[
            Union[str, "_models.CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPosition"]
        ] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class CreateStripeCheckoutSessionCustomerUpdate(_Model):  # pylint: disable=name-too-long
    """Controls what fields on Customer can be updated by the Checkout Session.

    :ivar address: Describes whether Checkout saves the billing address onto customer.address.
     To always collect a full billing address, use billing_address_collection.
     Defaults to never. Known values are: "auto" and "never".
    :vartype address: str or ~openmeter.models.CreateStripeCheckoutSessionCustomerUpdateBehavior
    :ivar name: Describes whether Checkout saves the name onto customer.name.
     Defaults to never. Known values are: "auto" and "never".
    :vartype name: str or ~openmeter.models.CreateStripeCheckoutSessionCustomerUpdateBehavior
    :ivar shipping: Describes whether Checkout saves shipping information onto customer.shipping.
     To collect shipping information, use shipping_address_collection.
     Defaults to never. Known values are: "auto" and "never".
    :vartype shipping: str or ~openmeter.models.CreateStripeCheckoutSessionCustomerUpdateBehavior
    """

    address: Optional[Union[str, "_models.CreateStripeCheckoutSessionCustomerUpdateBehavior"]] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """Describes whether Checkout saves the billing address onto customer.address.
     To always collect a full billing address, use billing_address_collection.
     Defaults to never. Known values are: \"auto\" and \"never\"."""
    name: Optional[Union[str, "_models.CreateStripeCheckoutSessionCustomerUpdateBehavior"]] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """Describes whether Checkout saves the name onto customer.name.
     Defaults to never. Known values are: \"auto\" and \"never\"."""
    shipping: Optional[Union[str, "_models.CreateStripeCheckoutSessionCustomerUpdateBehavior"]] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """Describes whether Checkout saves shipping information onto customer.shipping.
     To collect shipping information, use shipping_address_collection.
     Defaults to never. Known values are: \"auto\" and \"never\"."""

    @overload
    def __init__(
        self,
        *,
        address: Optional[Union[str, "_models.CreateStripeCheckoutSessionCustomerUpdateBehavior"]] = None,
        name: Optional[Union[str, "_models.CreateStripeCheckoutSessionCustomerUpdateBehavior"]] = None,
        shipping: Optional[Union[str, "_models.CreateStripeCheckoutSessionCustomerUpdateBehavior"]] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class CreateStripeCheckoutSessionRequest(_Model):
    """Create Stripe checkout session request.

    :ivar app_id: If not provided, the default Stripe app is used if any.
    :vartype app_id: str
    :ivar customer: Provide a customer ID or key to use an existing OpenMeter customer.
     or provide a customer object to create a new customer. Required. Is one of the following types:
     CustomerId, CustomerKey, CustomerCreate
    :vartype customer: ~openmeter._generated.models.CustomerId or
     ~openmeter._generated.models.CustomerKey or ~openmeter._generated.models.CustomerCreate
    :ivar stripe_customer_id: Stripe customer ID.
     If not provided OpenMeter creates a new Stripe customer or
     uses the OpenMeter customer's default Stripe customer ID.
    :vartype stripe_customer_id: str
    :ivar options: Options passed to Stripe when creating the checkout session. Required.
    :vartype options: ~openmeter._generated.models.CreateStripeCheckoutSessionRequestOptions
    """

    app_id: Optional[str] = rest_field(name="appId", visibility=["read", "create", "update", "delete", "query"])
    """If not provided, the default Stripe app is used if any."""
    customer: Union["_models.CustomerId", "_models.CustomerKey", "_models.CustomerCreate"] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """Provide a customer ID or key to use an existing OpenMeter customer.
     or provide a customer object to create a new customer. Required. Is one of the following types:
     CustomerId, CustomerKey, CustomerCreate"""
    stripe_customer_id: Optional[str] = rest_field(
        name="stripeCustomerId", visibility=["read", "create", "update", "delete", "query"]
    )
    """Stripe customer ID.
     If not provided OpenMeter creates a new Stripe customer or
     uses the OpenMeter customer's default Stripe customer ID."""
    options: "_models.CreateStripeCheckoutSessionRequestOptions" = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """Options passed to Stripe when creating the checkout session. Required."""

    @overload
    def __init__(
        self,
        *,
        customer: Union["_models.CustomerId", "_models.CustomerKey", "_models.CustomerCreate"],
        options: "_models.CreateStripeCheckoutSessionRequestOptions",
        app_id: Optional[str] = None,
        stripe_customer_id: Optional[str] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class CreateStripeCheckoutSessionRequestOptions(_Model):  # pylint: disable=name-too-long
    """Create Stripe checkout session options
    See `https://docs.stripe.com/api/checkout/sessions/create
    <https://docs.stripe.com/api/checkout/sessions/create>`_.

    :ivar billing_address_collection: Specify whether Checkout should collect the customer’s
     billing address. Defaults to auto. Known values are: "auto" and "required".
    :vartype billing_address_collection: str or
     ~openmeter.models.CreateStripeCheckoutSessionBillingAddressCollection
    :ivar cancel_url: If set, Checkout displays a back button and customers will be directed to
     this URL if they decide to cancel payment and return to your website.
     This parameter is not allowed if ui_mode is embedded.
    :vartype cancel_url: str
    :ivar client_reference_id: A unique string to reference the Checkout Session. This can be a
     customer ID, a cart ID, or similar, and can be used to reconcile the session with your internal
     systems.
    :vartype client_reference_id: str
    :ivar customer_update: Controls what fields on Customer can be updated by the Checkout Session.
    :vartype customer_update:
     ~openmeter._generated.models.CreateStripeCheckoutSessionCustomerUpdate
    :ivar consent_collection: Configure fields for the Checkout Session to gather active consent
     from customers.
    :vartype consent_collection:
     ~openmeter._generated.models.CreateStripeCheckoutSessionConsentCollection
    :ivar currency: Three-letter ISO currency code, in lowercase.
    :vartype currency: str
    :ivar custom_text: Display additional text for your customers using custom text.
    :vartype custom_text: ~openmeter._generated.models.CheckoutSessionCustomTextAfterSubmitParams
    :ivar expires_at: The Epoch time in seconds at which the Checkout Session will expire.
     It can be anywhere from 30 minutes to 24 hours after Checkout Session creation. By default,
     this value is 24 hours from creation.
    :vartype expires_at: int
    :ivar locale:
    :vartype locale: str
    :ivar metadata: Set of key-value pairs that you can attach to an object.
     This can be useful for storing additional information about the object in a structured format.
     Individual keys can be unset by posting an empty value to them.
     All keys can be unset by posting an empty value to metadata.
    :vartype metadata: dict[str, str]
    :ivar return_url: The URL to redirect your customer back to after they authenticate or cancel
     their payment on the payment method’s app or site.
     This parameter is required if ui_mode is embedded and redirect-based payment methods are
     enabled on the session.
    :vartype return_url: str
    :ivar success_url: The URL to which Stripe should send customers when payment or setup is
     complete.
     This parameter is not allowed if ui_mode is embedded.
     If you’d like to use information from the successful Checkout Session on your page, read the
     guide on customizing your success page:
     `https://docs.stripe.com/payments/checkout/custom-success-page
     <https://docs.stripe.com/payments/checkout/custom-success-page>`_.
    :vartype success_url: str
    :ivar ui_mode: The UI mode of the Session. Defaults to hosted. Known values are: "embedded" and
     "hosted".
    :vartype ui_mode: str or ~openmeter.models.CheckoutSessionUIMode
    :ivar payment_method_types: A list of the types of payment methods (e.g., card) this Checkout
     Session can accept.
    :vartype payment_method_types: list[str]
    :ivar redirect_on_completion: This parameter applies to ui_mode: embedded. Defaults to always.
     Learn more about the redirect behavior of embedded sessions at
     `https://docs.stripe.com/payments/checkout/custom-success-page?payment-ui=embedded-form
     <https://docs.stripe.com/payments/checkout/custom-success-page?payment-ui=embedded-form>`_.
     Known values are: "always", "if_required", and "never".
    :vartype redirect_on_completion: str or
     ~openmeter.models.CreateStripeCheckoutSessionRedirectOnCompletion
    :ivar tax_id_collection: Controls tax ID collection during checkout.
    :vartype tax_id_collection: ~openmeter._generated.models.CreateCheckoutSessionTaxIdCollection
    """

    billing_address_collection: Optional[Union[str, "_models.CreateStripeCheckoutSessionBillingAddressCollection"]] = (
        rest_field(name="billingAddressCollection", visibility=["read", "create", "update", "delete", "query"])
    )
    """Specify whether Checkout should collect the customer’s billing address. Defaults to auto. Known
     values are: \"auto\" and \"required\"."""
    cancel_url: Optional[str] = rest_field(name="cancelURL", visibility=["read", "create", "update", "delete", "query"])
    """If set, Checkout displays a back button and customers will be directed to this URL if they
     decide to cancel payment and return to your website.
     This parameter is not allowed if ui_mode is embedded."""
    client_reference_id: Optional[str] = rest_field(
        name="clientReferenceID", visibility=["read", "create", "update", "delete", "query"]
    )
    """A unique string to reference the Checkout Session. This can be a customer ID, a cart ID, or
     similar, and can be used to reconcile the session with your internal systems."""
    customer_update: Optional["_models.CreateStripeCheckoutSessionCustomerUpdate"] = rest_field(
        name="customerUpdate", visibility=["read", "create", "update", "delete", "query"]
    )
    """Controls what fields on Customer can be updated by the Checkout Session."""
    consent_collection: Optional["_models.CreateStripeCheckoutSessionConsentCollection"] = rest_field(
        name="consentCollection", visibility=["read", "create", "update", "delete", "query"]
    )
    """Configure fields for the Checkout Session to gather active consent from customers."""
    currency: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Three-letter ISO currency code, in lowercase."""
    custom_text: Optional["_models.CheckoutSessionCustomTextAfterSubmitParams"] = rest_field(
        name="customText", visibility=["read", "create", "update", "delete", "query"]
    )
    """Display additional text for your customers using custom text."""
    expires_at: Optional[int] = rest_field(name="expiresAt", visibility=["read", "create", "update", "delete", "query"])
    """The Epoch time in seconds at which the Checkout Session will expire.
     It can be anywhere from 30 minutes to 24 hours after Checkout Session creation. By default,
     this value is 24 hours from creation."""
    locale: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    metadata: Optional[dict[str, str]] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Set of key-value pairs that you can attach to an object.
     This can be useful for storing additional information about the object in a structured format.
     Individual keys can be unset by posting an empty value to them.
     All keys can be unset by posting an empty value to metadata."""
    return_url: Optional[str] = rest_field(name="returnURL", visibility=["read", "create", "update", "delete", "query"])
    """The URL to redirect your customer back to after they authenticate or cancel their payment on
     the payment method’s app or site.
     This parameter is required if ui_mode is embedded and redirect-based payment methods are
     enabled on the session."""
    success_url: Optional[str] = rest_field(
        name="successURL", visibility=["read", "create", "update", "delete", "query"]
    )
    """The URL to which Stripe should send customers when payment or setup is complete.
     This parameter is not allowed if ui_mode is embedded.
     If you’d like to use information from the successful Checkout Session on your page, read the
     guide on customizing your success page:
     `https://docs.stripe.com/payments/checkout/custom-success-page
     <https://docs.stripe.com/payments/checkout/custom-success-page>`_."""
    ui_mode: Optional[Union[str, "_models.CheckoutSessionUIMode"]] = rest_field(
        name="uiMode", visibility=["read", "create", "update", "delete", "query"]
    )
    """The UI mode of the Session. Defaults to hosted. Known values are: \"embedded\" and \"hosted\"."""
    payment_method_types: Optional[list[str]] = rest_field(
        name="paymentMethodTypes", visibility=["read", "create", "update", "delete", "query"]
    )
    """A list of the types of payment methods (e.g., card) this Checkout Session can accept."""
    redirect_on_completion: Optional[Union[str, "_models.CreateStripeCheckoutSessionRedirectOnCompletion"]] = (
        rest_field(name="redirectOnCompletion", visibility=["read", "create", "update", "delete", "query"])
    )
    """This parameter applies to ui_mode: embedded. Defaults to always.
     Learn more about the redirect behavior of embedded sessions at
     `https://docs.stripe.com/payments/checkout/custom-success-page?payment-ui=embedded-form
     <https://docs.stripe.com/payments/checkout/custom-success-page?payment-ui=embedded-form>`_.
     Known values are: \"always\", \"if_required\", and \"never\"."""
    tax_id_collection: Optional["_models.CreateCheckoutSessionTaxIdCollection"] = rest_field(
        name="taxIdCollection", visibility=["read", "create", "update", "delete", "query"]
    )
    """Controls tax ID collection during checkout."""

    @overload
    def __init__(
        self,
        *,
        billing_address_collection: Optional[
            Union[str, "_models.CreateStripeCheckoutSessionBillingAddressCollection"]
        ] = None,
        cancel_url: Optional[str] = None,
        client_reference_id: Optional[str] = None,
        customer_update: Optional["_models.CreateStripeCheckoutSessionCustomerUpdate"] = None,
        consent_collection: Optional["_models.CreateStripeCheckoutSessionConsentCollection"] = None,
        currency: Optional[str] = None,
        custom_text: Optional["_models.CheckoutSessionCustomTextAfterSubmitParams"] = None,
        expires_at: Optional[int] = None,
        locale: Optional[str] = None,
        metadata: Optional[dict[str, str]] = None,
        return_url: Optional[str] = None,
        success_url: Optional[str] = None,
        ui_mode: Optional[Union[str, "_models.CheckoutSessionUIMode"]] = None,
        payment_method_types: Optional[list[str]] = None,
        redirect_on_completion: Optional[Union[str, "_models.CreateStripeCheckoutSessionRedirectOnCompletion"]] = None,
        tax_id_collection: Optional["_models.CreateCheckoutSessionTaxIdCollection"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class CreateStripeCheckoutSessionResult(_Model):
    """Create Stripe Checkout Session response.

    :ivar customer_id: The OpenMeter customer ID. Required.
    :vartype customer_id: str
    :ivar stripe_customer_id: The Stripe customer ID. Required.
    :vartype stripe_customer_id: str
    :ivar session_id: The checkout session ID. Required.
    :vartype session_id: str
    :ivar setup_intent_id: The checkout session setup intent ID. Required.
    :vartype setup_intent_id: str
    :ivar client_secret: The client secret of the checkout session.
     This can be used to initialize Stripe.js for your client-side implementation.
    :vartype client_secret: str
    :ivar client_reference_id: A unique string to reference the Checkout Session.
     This can be a customer ID, a cart ID, or similar, and can be used to reconcile the session with
     your internal systems.
    :vartype client_reference_id: str
    :ivar customer_email: Customer's email address provided to Stripe.
    :vartype customer_email: str
    :ivar currency: Three-letter ISO currency code, in lowercase.
    :vartype currency: str
    :ivar created_at: Timestamp at which the checkout session was created. Required.
    :vartype created_at: ~datetime.datetime
    :ivar expires_at: Timestamp at which the checkout session will expire.
    :vartype expires_at: ~datetime.datetime
    :ivar metadata: Set of key-value pairs attached to the checkout session.
    :vartype metadata: dict[str, str]
    :ivar status: The status of the checkout session.
    :vartype status: str
    :ivar url: URL to show the checkout session.
    :vartype url: str
    :ivar mode: Mode
     Always ``setup`` for now. Required. "setup"
    :vartype mode: str or ~openmeter.models.StripeCheckoutSessionMode
    :ivar cancel_url: Cancel URL.
    :vartype cancel_url: str
    :ivar success_url: Success URL.
    :vartype success_url: str
    :ivar return_url: Return URL.
    :vartype return_url: str
    """

    customer_id: str = rest_field(name="customerId", visibility=["read", "create", "update", "delete", "query"])
    """The OpenMeter customer ID. Required."""
    stripe_customer_id: str = rest_field(
        name="stripeCustomerId", visibility=["read", "create", "update", "delete", "query"]
    )
    """The Stripe customer ID. Required."""
    session_id: str = rest_field(name="sessionId", visibility=["read", "create", "update", "delete", "query"])
    """The checkout session ID. Required."""
    setup_intent_id: str = rest_field(name="setupIntentId", visibility=["read", "create", "update", "delete", "query"])
    """The checkout session setup intent ID. Required."""
    client_secret: Optional[str] = rest_field(
        name="clientSecret", visibility=["read", "create", "update", "delete", "query"]
    )
    """The client secret of the checkout session.
     This can be used to initialize Stripe.js for your client-side implementation."""
    client_reference_id: Optional[str] = rest_field(
        name="clientReferenceId", visibility=["read", "create", "update", "delete", "query"]
    )
    """A unique string to reference the Checkout Session.
     This can be a customer ID, a cart ID, or similar, and can be used to reconcile the session with
     your internal systems."""
    customer_email: Optional[str] = rest_field(
        name="customerEmail", visibility=["read", "create", "update", "delete", "query"]
    )
    """Customer's email address provided to Stripe."""
    currency: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Three-letter ISO currency code, in lowercase."""
    created_at: datetime.datetime = rest_field(
        name="createdAt", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """Timestamp at which the checkout session was created. Required."""
    expires_at: Optional[datetime.datetime] = rest_field(
        name="expiresAt", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """Timestamp at which the checkout session will expire."""
    metadata: Optional[dict[str, str]] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Set of key-value pairs attached to the checkout session."""
    status: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The status of the checkout session."""
    url: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """URL to show the checkout session."""
    mode: Union[str, "_models.StripeCheckoutSessionMode"] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """Mode
     Always ``setup`` for now. Required. \"setup\""""
    cancel_url: Optional[str] = rest_field(name="cancelURL", visibility=["read", "create", "update", "delete", "query"])
    """Cancel URL."""
    success_url: Optional[str] = rest_field(
        name="successURL", visibility=["read", "create", "update", "delete", "query"]
    )
    """Success URL."""
    return_url: Optional[str] = rest_field(name="returnURL", visibility=["read", "create", "update", "delete", "query"])
    """Return URL."""

    @overload
    def __init__(
        self,
        *,
        customer_id: str,
        stripe_customer_id: str,
        session_id: str,
        setup_intent_id: str,
        created_at: datetime.datetime,
        mode: Union[str, "_models.StripeCheckoutSessionMode"],
        client_secret: Optional[str] = None,
        client_reference_id: Optional[str] = None,
        customer_email: Optional[str] = None,
        currency: Optional[str] = None,
        expires_at: Optional[datetime.datetime] = None,
        metadata: Optional[dict[str, str]] = None,
        status: Optional[str] = None,
        url: Optional[str] = None,
        cancel_url: Optional[str] = None,
        success_url: Optional[str] = None,
        return_url: Optional[str] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class CreateStripeCustomerPortalSessionParams(_Model):
    """Stripe customer portal request params.

    :ivar configuration_id: Configuration.
    :vartype configuration_id: str
    :ivar locale: Locale.
    :vartype locale: str
    :ivar return_url: ReturnUrl.
    :vartype return_url: str
    """

    configuration_id: Optional[str] = rest_field(
        name="configurationId", visibility=["read", "create", "update", "delete", "query"]
    )
    """Configuration."""
    locale: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Locale."""
    return_url: Optional[str] = rest_field(name="returnUrl", visibility=["read", "create", "update", "delete", "query"])
    """ReturnUrl."""

    @overload
    def __init__(
        self,
        *,
        configuration_id: Optional[str] = None,
        locale: Optional[str] = None,
        return_url: Optional[str] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class InvoiceGenericDocumentRef(_Model):
    """InvoiceGenericDocumentRef is used to describe an existing document or a specific part of it's
    contents.

    :ivar type: Type of the document referenced. Required. "credit_note_original_invoice"
    :vartype type: str or ~openmeter.models.InvoiceDocumentRefType
    :ivar reason: Human readable description on why this reference is here or needs to be used.
    :vartype reason: str
    :ivar description: Additional details about the document.
    :vartype description: str
    """

    type: Union[str, "_models.InvoiceDocumentRefType"] = rest_field(visibility=["read"])
    """Type of the document referenced. Required. \"credit_note_original_invoice\""""
    reason: Optional[str] = rest_field(visibility=["read"])
    """Human readable description on why this reference is here or needs to be used."""
    description: Optional[str] = rest_field(visibility=["read"])
    """Additional details about the document."""


class CreditNoteOriginalInvoiceRef(InvoiceGenericDocumentRef):
    """CreditNoteOriginalInvoiceRef is used to reference the original invoice that a credit note is
    based on.

    :ivar reason: Human readable description on why this reference is here or needs to be used.
    :vartype reason: str
    :ivar description: Additional details about the document.
    :vartype description: str
    :ivar type: Type of the invoice. Required.
    :vartype type: str or ~openmeter._generated.models.CREDIT_NOTE_ORIGINAL_INVOICE
    :ivar issued_at: IssueAt reflects the time the document was issued.
    :vartype issued_at: ~datetime.datetime
    :ivar number: (Serial) Number of the referenced document.
    :vartype number: str
    :ivar url: Link to the source document. Required.
    :vartype url: str
    """

    type: Literal[InvoiceDocumentRefType.CREDIT_NOTE_ORIGINAL_INVOICE] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """Type of the invoice. Required."""
    issued_at: Optional[datetime.datetime] = rest_field(name="issuedAt", visibility=["read"], format="rfc3339")
    """IssueAt reflects the time the document was issued."""
    number: Optional[str] = rest_field(visibility=["read"])
    """(Serial) Number of the referenced document."""
    url: str = rest_field(visibility=["read"])
    """Link to the source document. Required."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[InvoiceDocumentRefType.CREDIT_NOTE_ORIGINAL_INVOICE],
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class Currency(_Model):
    """Currency describes a currency supported by OpenMeter.

    :ivar code: The currency ISO code. Required.
    :vartype code: str
    :ivar name: The currency name. Required.
    :vartype name: str
    :ivar symbol: The currency symbol. Required.
    :vartype symbol: str
    :ivar subunits: Subunit of the currency. Required.
    :vartype subunits: int
    """

    code: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The currency ISO code. Required."""
    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The currency name. Required."""
    symbol: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The currency symbol. Required."""
    subunits: int = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Subunit of the currency. Required."""

    @overload
    def __init__(
        self,
        *,
        code: str,
        name: str,
        symbol: str,
        subunits: int,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class Customer(_Model):
    """A customer object.

    :ivar id: ID. Required.
    :vartype id: str
    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar created_at: Creation Time. Required.
    :vartype created_at: ~datetime.datetime
    :ivar updated_at: Last Update Time. Required.
    :vartype updated_at: ~datetime.datetime
    :ivar deleted_at: Deletion Time.
    :vartype deleted_at: ~datetime.datetime
    :ivar key: Key.
    :vartype key: str
    :ivar usage_attribution: Usage Attribution.
    :vartype usage_attribution: ~openmeter._generated.models.CustomerUsageAttribution
    :ivar primary_email: Primary Email.
    :vartype primary_email: str
    :ivar currency: Currency.
    :vartype currency: str
    :ivar billing_address: Billing Address.
    :vartype billing_address: ~openmeter._generated.models.Address
    :ivar current_subscription_id: Current Subscription ID.
    :vartype current_subscription_id: str
    :ivar subscriptions: Subscriptions.
    :vartype subscriptions: list[~openmeter._generated.models.Subscription]
    :ivar annotations: Annotations.
    :vartype annotations: ~openmeter._generated.models.Annotations
    """

    id: str = rest_field(visibility=["read"])
    """ID. Required."""
    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Display name. Required."""
    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    created_at: datetime.datetime = rest_field(name="createdAt", visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    updated_at: datetime.datetime = rest_field(name="updatedAt", visibility=["read"], format="rfc3339")
    """Last Update Time. Required."""
    deleted_at: Optional[datetime.datetime] = rest_field(name="deletedAt", visibility=["read"], format="rfc3339")
    """Deletion Time."""
    key: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Key."""
    usage_attribution: Optional["_models.CustomerUsageAttribution"] = rest_field(
        name="usageAttribution", visibility=["read", "create", "update", "delete", "query"]
    )
    """Usage Attribution."""
    primary_email: Optional[str] = rest_field(
        name="primaryEmail", visibility=["read", "create", "update", "delete", "query"]
    )
    """Primary Email."""
    currency: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Currency."""
    billing_address: Optional["_models.Address"] = rest_field(
        name="billingAddress", visibility=["read", "create", "update", "delete", "query"]
    )
    """Billing Address."""
    current_subscription_id: Optional[str] = rest_field(name="currentSubscriptionId", visibility=["read"])
    """Current Subscription ID."""
    subscriptions: Optional[list["_models.Subscription"]] = rest_field(visibility=["read"])
    """Subscriptions."""
    annotations: Optional["_models.Annotations"] = rest_field(visibility=["read"])
    """Annotations."""

    @overload
    def __init__(
        self,
        *,
        name: str,
        description: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
        key: Optional[str] = None,
        usage_attribution: Optional["_models.CustomerUsageAttribution"] = None,
        primary_email: Optional[str] = None,
        currency: Optional[str] = None,
        billing_address: Optional["_models.Address"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class CustomerAccess(_Model):
    """CustomerAccess describes what features the customer has access to.

    :ivar entitlements: Map of entitlements the customer has access to.
     The key is the feature key, the value is the entitlement value + the entitlement ID. Required.
    :vartype entitlements: dict[str, ~openmeter._generated.models.EntitlementValue]
    """

    entitlements: dict[str, "_models.EntitlementValue"] = rest_field(visibility=["read"])
    """Map of entitlements the customer has access to.
     The key is the feature key, the value is the entitlement value + the entitlement ID. Required."""


class CustomerAppDataPaginatedResponse(_Model):
    """Paginated response.

    :ivar total_count: The total number of items. Required.
    :vartype total_count: int
    :ivar page: The page index. Required.
    :vartype page: int
    :ivar page_size: The maximum number of items per page. Required.
    :vartype page_size: int
    :ivar items_property: The items in the current page. Required.
    :vartype items_property: list[~openmeter._generated.models.StripeCustomerAppData or
     ~openmeter._generated.models.SandboxCustomerAppData or
     ~openmeter._generated.models.CustomInvoicingCustomerAppData]
    """

    total_count: int = rest_field(name="totalCount", visibility=["read", "create", "update", "delete", "query"])
    """The total number of items. Required."""
    page: int = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The page index. Required."""
    page_size: int = rest_field(name="pageSize", visibility=["read", "create", "update", "delete", "query"])
    """The maximum number of items per page. Required."""
    items_property: list["_types.CustomerAppData"] = rest_field(
        name="items", visibility=["read", "create", "update", "delete", "query"]
    )
    """The items in the current page. Required."""

    @overload
    def __init__(
        self,
        *,
        total_count: int,
        page: int,
        page_size: int,
        items_property: list["_types.CustomerAppData"],
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class CustomerCreate(_Model):
    """Resource create operation model.

    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar key: Key.
    :vartype key: str
    :ivar usage_attribution: Usage Attribution.
    :vartype usage_attribution: ~openmeter._generated.models.CustomerUsageAttribution
    :ivar primary_email: Primary Email.
    :vartype primary_email: str
    :ivar currency: Currency.
    :vartype currency: str
    :ivar billing_address: Billing Address.
    :vartype billing_address: ~openmeter._generated.models.Address
    """

    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Display name. Required."""
    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    key: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Key."""
    usage_attribution: Optional["_models.CustomerUsageAttribution"] = rest_field(
        name="usageAttribution", visibility=["read", "create", "update", "delete", "query"]
    )
    """Usage Attribution."""
    primary_email: Optional[str] = rest_field(
        name="primaryEmail", visibility=["read", "create", "update", "delete", "query"]
    )
    """Primary Email."""
    currency: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Currency."""
    billing_address: Optional["_models.Address"] = rest_field(
        name="billingAddress", visibility=["read", "create", "update", "delete", "query"]
    )
    """Billing Address."""

    @overload
    def __init__(
        self,
        *,
        name: str,
        description: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
        key: Optional[str] = None,
        usage_attribution: Optional["_models.CustomerUsageAttribution"] = None,
        primary_email: Optional[str] = None,
        currency: Optional[str] = None,
        billing_address: Optional["_models.Address"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class CustomerId(_Model):
    """Create Stripe checkout session with customer ID.

    :ivar id: Required.
    :vartype id: str
    """

    id: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Required."""

    @overload
    def __init__(
        self,
        *,
        id: str,  # pylint: disable=redefined-builtin
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class CustomerKey(_Model):
    """Create Stripe checkout session with customer key.

    :ivar key: Required.
    :vartype key: str
    """

    key: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Required."""

    @overload
    def __init__(
        self,
        *,
        key: str,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class CustomerPaginatedResponse(_Model):
    """Paginated response.

    :ivar total_count: The total number of items. Required.
    :vartype total_count: int
    :ivar page: The page index. Required.
    :vartype page: int
    :ivar page_size: The maximum number of items per page. Required.
    :vartype page_size: int
    :ivar items_property: The items in the current page. Required.
    :vartype items_property: list[~openmeter._generated.models.Customer]
    """

    total_count: int = rest_field(name="totalCount", visibility=["read", "create", "update", "delete", "query"])
    """The total number of items. Required."""
    page: int = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The page index. Required."""
    page_size: int = rest_field(name="pageSize", visibility=["read", "create", "update", "delete", "query"])
    """The maximum number of items per page. Required."""
    items_property: list["_models.Customer"] = rest_field(
        name="items", visibility=["read", "create", "update", "delete", "query"]
    )
    """The items in the current page. Required."""

    @overload
    def __init__(
        self,
        *,
        total_count: int,
        page: int,
        page_size: int,
        items_property: list["_models.Customer"],
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class CustomerReplaceUpdate(_Model):
    """Resource update operation model.

    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar key: Key.
    :vartype key: str
    :ivar usage_attribution: Usage Attribution.
    :vartype usage_attribution: ~openmeter._generated.models.CustomerUsageAttribution
    :ivar primary_email: Primary Email.
    :vartype primary_email: str
    :ivar currency: Currency.
    :vartype currency: str
    :ivar billing_address: Billing Address.
    :vartype billing_address: ~openmeter._generated.models.Address
    """

    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Display name. Required."""
    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    key: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Key."""
    usage_attribution: Optional["_models.CustomerUsageAttribution"] = rest_field(
        name="usageAttribution", visibility=["read", "create", "update", "delete", "query"]
    )
    """Usage Attribution."""
    primary_email: Optional[str] = rest_field(
        name="primaryEmail", visibility=["read", "create", "update", "delete", "query"]
    )
    """Primary Email."""
    currency: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Currency."""
    billing_address: Optional["_models.Address"] = rest_field(
        name="billingAddress", visibility=["read", "create", "update", "delete", "query"]
    )
    """Billing Address."""

    @overload
    def __init__(
        self,
        *,
        name: str,
        description: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
        key: Optional[str] = None,
        usage_attribution: Optional["_models.CustomerUsageAttribution"] = None,
        primary_email: Optional[str] = None,
        currency: Optional[str] = None,
        billing_address: Optional["_models.Address"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class CustomerUsageAttribution(_Model):
    """Mapping to attribute metered usage to the customer.
    One customer can have zero or more subjects,
    but one subject can only belong to one customer.

    :ivar subject_keys: SubjectKeys. Required.
    :vartype subject_keys: list[str]
    """

    subject_keys: list[str] = rest_field(name="subjectKeys", visibility=["read", "create", "update", "delete", "query"])
    """SubjectKeys. Required."""

    @overload
    def __init__(
        self,
        *,
        subject_keys: list[str],
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class CustomInvoicingApp(_Model):
    """Custom Invoicing app can be used for interface with any invoicing or payment system.

    This app provides ways to manipulate invoices and payments, however the integration
    must rely on Notifications API to get notified about invoice changes.

    :ivar id: ID. Required.
    :vartype id: str
    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar created_at: Creation Time. Required.
    :vartype created_at: ~datetime.datetime
    :ivar updated_at: Last Update Time. Required.
    :vartype updated_at: ~datetime.datetime
    :ivar deleted_at: Deletion Time.
    :vartype deleted_at: ~datetime.datetime
    :ivar listing: The marketplace listing that this installed app is based on. Required.
    :vartype listing: ~openmeter._generated.models.MarketplaceListing
    :ivar status: Status of the app connection. Required. Known values are: "ready" and
     "unauthorized".
    :vartype status: str or ~openmeter.models.AppStatus
    :ivar type: The app's type is CustomInvoicing. Required.
    :vartype type: str or ~openmeter._generated.models.CUSTOM_INVOICING
    :ivar enable_draft_sync_hook: Enable draft.sync hook.

     If the hook is not enabled, the invoice will be progressed to the next state automatically.
     Required.
    :vartype enable_draft_sync_hook: bool
    :ivar enable_issuing_sync_hook: Enable issuing.sync hook.

     If the hook is not enabled, the invoice will be progressed to the next state automatically.
     Required.
    :vartype enable_issuing_sync_hook: bool
    """

    id: str = rest_field(visibility=["read"])
    """ID. Required."""
    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Display name. Required."""
    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    created_at: datetime.datetime = rest_field(name="createdAt", visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    updated_at: datetime.datetime = rest_field(name="updatedAt", visibility=["read"], format="rfc3339")
    """Last Update Time. Required."""
    deleted_at: Optional[datetime.datetime] = rest_field(name="deletedAt", visibility=["read"], format="rfc3339")
    """Deletion Time."""
    listing: "_models.MarketplaceListing" = rest_field(visibility=["read"])
    """The marketplace listing that this installed app is based on. Required."""
    status: Union[str, "_models.AppStatus"] = rest_field(visibility=["read"])
    """Status of the app connection. Required. Known values are: \"ready\" and \"unauthorized\"."""
    type: Literal[AppType.CUSTOM_INVOICING] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The app's type is CustomInvoicing. Required."""
    enable_draft_sync_hook: bool = rest_field(
        name="enableDraftSyncHook", visibility=["read", "create", "update", "delete", "query"]
    )
    """Enable draft.sync hook.
     
     If the hook is not enabled, the invoice will be progressed to the next state automatically.
     Required."""
    enable_issuing_sync_hook: bool = rest_field(
        name="enableIssuingSyncHook", visibility=["read", "create", "update", "delete", "query"]
    )
    """Enable issuing.sync hook.
     
     If the hook is not enabled, the invoice will be progressed to the next state automatically.
     Required."""

    @overload
    def __init__(
        self,
        *,
        name: str,
        type: Literal[AppType.CUSTOM_INVOICING],
        enable_draft_sync_hook: bool,
        enable_issuing_sync_hook: bool,
        description: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class CustomInvoicingAppReplaceUpdate(_Model):
    """Resource update operation model.

    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar type: The app's type is CustomInvoicing. Required.
    :vartype type: str or ~openmeter._generated.models.CUSTOM_INVOICING
    :ivar enable_draft_sync_hook: Enable draft.sync hook.

     If the hook is not enabled, the invoice will be progressed to the next state automatically.
     Required.
    :vartype enable_draft_sync_hook: bool
    :ivar enable_issuing_sync_hook: Enable issuing.sync hook.

     If the hook is not enabled, the invoice will be progressed to the next state automatically.
     Required.
    :vartype enable_issuing_sync_hook: bool
    """

    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Display name. Required."""
    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    type: Literal[AppType.CUSTOM_INVOICING] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The app's type is CustomInvoicing. Required."""
    enable_draft_sync_hook: bool = rest_field(
        name="enableDraftSyncHook", visibility=["read", "create", "update", "delete", "query"]
    )
    """Enable draft.sync hook.
     
     If the hook is not enabled, the invoice will be progressed to the next state automatically.
     Required."""
    enable_issuing_sync_hook: bool = rest_field(
        name="enableIssuingSyncHook", visibility=["read", "create", "update", "delete", "query"]
    )
    """Enable issuing.sync hook.
     
     If the hook is not enabled, the invoice will be progressed to the next state automatically.
     Required."""

    @overload
    def __init__(
        self,
        *,
        name: str,
        type: Literal[AppType.CUSTOM_INVOICING],
        enable_draft_sync_hook: bool,
        enable_issuing_sync_hook: bool,
        description: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class CustomInvoicingCustomerAppData(_Model):
    """Custom Invoicing Customer App Data.

    :ivar app: The installed custom invoicing app this data belongs to.
    :vartype app: ~openmeter._generated.models.CustomInvoicingApp
    :ivar id: App ID.
    :vartype id: str
    :ivar type: App Type. Required.
    :vartype type: str or ~openmeter._generated.models.CUSTOM_INVOICING
    :ivar metadata: Metadata to be used by the custom invoicing provider.
    :vartype metadata: ~openmeter._generated.models.Metadata
    """

    app: Optional["_models.CustomInvoicingApp"] = rest_field(visibility=["read"])
    """The installed custom invoicing app this data belongs to."""
    id: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """App ID."""
    type: Literal[AppType.CUSTOM_INVOICING] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """App Type. Required."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata to be used by the custom invoicing provider."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[AppType.CUSTOM_INVOICING],
        id: Optional[str] = None,  # pylint: disable=redefined-builtin
        metadata: Optional["_models.Metadata"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class CustomInvoicingDraftSynchronizedRequest(_Model):
    """Information to finalize the draft details of an invoice.

    :ivar invoicing: The result of the synchronization.
    :vartype invoicing: ~openmeter._generated.models.CustomInvoicingSyncResult
    """

    invoicing: Optional["_models.CustomInvoicingSyncResult"] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """The result of the synchronization."""

    @overload
    def __init__(
        self,
        *,
        invoicing: Optional["_models.CustomInvoicingSyncResult"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class CustomInvoicingFinalizedInvoicingRequest(_Model):
    """Information to finalize the invoicing details of an invoice.

    :ivar invoice_number: If set the invoice's number will be set to this value.
    :vartype invoice_number: str
    :ivar sent_to_customer_at: If set the invoice's sent to customer at will be set to this value.
    :vartype sent_to_customer_at: ~datetime.datetime
    """

    invoice_number: Optional[str] = rest_field(
        name="invoiceNumber", visibility=["read", "create", "update", "delete", "query"]
    )
    """If set the invoice's number will be set to this value."""
    sent_to_customer_at: Optional[datetime.datetime] = rest_field(
        name="sentToCustomerAt", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """If set the invoice's sent to customer at will be set to this value."""

    @overload
    def __init__(
        self,
        *,
        invoice_number: Optional[str] = None,
        sent_to_customer_at: Optional[datetime.datetime] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class CustomInvoicingFinalizedPaymentRequest(_Model):
    """Information to finalize the payment details of an invoice.

    :ivar external_id: If set the invoice's payment external ID will be set to this value.
    :vartype external_id: str
    """

    external_id: Optional[str] = rest_field(
        name="externalId", visibility=["read", "create", "update", "delete", "query"]
    )
    """If set the invoice's payment external ID will be set to this value."""

    @overload
    def __init__(
        self,
        *,
        external_id: Optional[str] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class CustomInvoicingFinalizedRequest(_Model):
    """Information to finalize the invoice.

    If invoicing.invoiceNumber is not set, then a new invoice number will be generated (INV-
    prefix).

    :ivar invoicing: The result of the synchronization.
    :vartype invoicing: ~openmeter._generated.models.CustomInvoicingFinalizedInvoicingRequest
    :ivar payment: The result of the payment synchronization.
    :vartype payment: ~openmeter._generated.models.CustomInvoicingFinalizedPaymentRequest
    """

    invoicing: Optional["_models.CustomInvoicingFinalizedInvoicingRequest"] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """The result of the synchronization."""
    payment: Optional["_models.CustomInvoicingFinalizedPaymentRequest"] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """The result of the payment synchronization."""

    @overload
    def __init__(
        self,
        *,
        invoicing: Optional["_models.CustomInvoicingFinalizedInvoicingRequest"] = None,
        payment: Optional["_models.CustomInvoicingFinalizedPaymentRequest"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class CustomInvoicingLineDiscountExternalIdMapping(_Model):  # pylint: disable=name-too-long
    """Mapping between line discounts and external IDs.

    :ivar line_discount_id: The line discount ID. Required.
    :vartype line_discount_id: str
    :ivar external_id: The external ID (e.g. custom invoicing system's ID). Required.
    :vartype external_id: str
    """

    line_discount_id: str = rest_field(
        name="lineDiscountId", visibility=["read", "create", "update", "delete", "query"]
    )
    """The line discount ID. Required."""
    external_id: str = rest_field(name="externalId", visibility=["read", "create", "update", "delete", "query"])
    """The external ID (e.g. custom invoicing system's ID). Required."""

    @overload
    def __init__(
        self,
        *,
        line_discount_id: str,
        external_id: str,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class CustomInvoicingLineExternalIdMapping(_Model):
    """Mapping between lines and external IDs.

    :ivar line_id: The line ID. Required.
    :vartype line_id: str
    :ivar external_id: The external ID (e.g. custom invoicing system's ID). Required.
    :vartype external_id: str
    """

    line_id: str = rest_field(name="lineId", visibility=["read", "create", "update", "delete", "query"])
    """The line ID. Required."""
    external_id: str = rest_field(name="externalId", visibility=["read", "create", "update", "delete", "query"])
    """The external ID (e.g. custom invoicing system's ID). Required."""

    @overload
    def __init__(
        self,
        *,
        line_id: str,
        external_id: str,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class CustomInvoicingSyncResult(_Model):
    """Information to synchronize the invoice.

    Can be used to store external app's IDs on the invoice or lines.

    :ivar invoice_number: If set the invoice's number will be set to this value.
    :vartype invoice_number: str
    :ivar external_id: If set the invoice's invoicing external ID will be set to this value.
    :vartype external_id: str
    :ivar line_external_ids: If set the invoice's line external IDs will be set to this value.

     This can be used to reference the external system's entities in the
     invoice.
    :vartype line_external_ids:
     list[~openmeter._generated.models.CustomInvoicingLineExternalIdMapping]
    :ivar line_discount_external_ids: If set the invoice's line discount external IDs will be set
     to this value.

     This can be used to reference the external system's entities in the
     invoice.
    :vartype line_discount_external_ids:
     list[~openmeter._generated.models.CustomInvoicingLineDiscountExternalIdMapping]
    """

    invoice_number: Optional[str] = rest_field(
        name="invoiceNumber", visibility=["read", "create", "update", "delete", "query"]
    )
    """If set the invoice's number will be set to this value."""
    external_id: Optional[str] = rest_field(
        name="externalId", visibility=["read", "create", "update", "delete", "query"]
    )
    """If set the invoice's invoicing external ID will be set to this value."""
    line_external_ids: Optional[list["_models.CustomInvoicingLineExternalIdMapping"]] = rest_field(
        name="lineExternalIds", visibility=["read", "create", "update", "delete", "query"]
    )
    """If set the invoice's line external IDs will be set to this value.
     
     This can be used to reference the external system's entities in the
     invoice."""
    line_discount_external_ids: Optional[list["_models.CustomInvoicingLineDiscountExternalIdMapping"]] = rest_field(
        name="lineDiscountExternalIds", visibility=["read", "create", "update", "delete", "query"]
    )
    """If set the invoice's line discount external IDs will be set to this value.
     
     This can be used to reference the external system's entities in the
     invoice."""

    @overload
    def __init__(
        self,
        *,
        invoice_number: Optional[str] = None,
        external_id: Optional[str] = None,
        line_external_ids: Optional[list["_models.CustomInvoicingLineExternalIdMapping"]] = None,
        line_discount_external_ids: Optional[list["_models.CustomInvoicingLineDiscountExternalIdMapping"]] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class CustomInvoicingTaxConfig(_Model):
    """Custom invoicing tax config.

    :ivar code: Tax code. Required.
    :vartype code: str
    """

    code: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Tax code. Required."""

    @overload
    def __init__(
        self,
        *,
        code: str,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class CustomInvoicingUpdatePaymentStatusRequest(_Model):  # pylint: disable=name-too-long
    """Update payment status request.

    Can be used to manipulate invoice's payment status (when custominvoicing app is being used).

    :ivar trigger: The trigger to be executed on the invoice. Required. Known values are: "paid",
     "payment_failed", "payment_uncollectible", "payment_overdue", "action_required", and "void".
    :vartype trigger: str or ~openmeter.models.CustomInvoicingPaymentTrigger
    """

    trigger: Union[str, "_models.CustomInvoicingPaymentTrigger"] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """The trigger to be executed on the invoice. Required. Known values are: \"paid\",
     \"payment_failed\", \"payment_uncollectible\", \"payment_overdue\", \"action_required\", and
     \"void\"."""

    @overload
    def __init__(
        self,
        *,
        trigger: Union[str, "_models.CustomInvoicingPaymentTrigger"],
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class OmitPropertiesResourceCreateModel(_Model):
    """The template for omitting properties.

    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar alignment: Alignment configuration for the plan.
    :vartype alignment: ~openmeter._generated.models.Alignment
    :ivar currency: Currency. Required.
    :vartype currency: str
    :ivar billing_cadence: Billing cadence. Required.
    :vartype billing_cadence: ~datetime.timedelta
    :ivar pro_rating_config: Pro-rating configuration.
    :vartype pro_rating_config: ~openmeter._generated.models.ProRatingConfig
    :ivar phases: Plan phases. Required.
    :vartype phases: list[~openmeter._generated.models.PlanPhase]
    """

    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Display name. Required."""
    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    alignment: Optional["_models.Alignment"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Alignment configuration for the plan."""
    currency: str = rest_field(visibility=["read", "create"])
    """Currency. Required."""
    billing_cadence: datetime.timedelta = rest_field(name="billingCadence", visibility=["read", "create", "update"])
    """Billing cadence. Required."""
    pro_rating_config: Optional["_models.ProRatingConfig"] = rest_field(
        name="proRatingConfig", visibility=["read", "create", "update"]
    )
    """Pro-rating configuration."""
    phases: list["_models.PlanPhase"] = rest_field(visibility=["read", "create", "update"])
    """Plan phases. Required."""

    @overload
    def __init__(
        self,
        *,
        name: str,
        currency: str,
        billing_cadence: datetime.timedelta,
        phases: list["_models.PlanPhase"],
        description: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
        alignment: Optional["_models.Alignment"] = None,
        pro_rating_config: Optional["_models.ProRatingConfig"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class CustomPlanInput(OmitPropertiesResourceCreateModel):
    """Plan input for custom subscription creation (without key and version).

    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar alignment: Alignment configuration for the plan.
    :vartype alignment: ~openmeter._generated.models.Alignment
    :ivar currency: Currency. Required.
    :vartype currency: str
    :ivar billing_cadence: Billing cadence. Required.
    :vartype billing_cadence: ~datetime.timedelta
    :ivar pro_rating_config: Pro-rating configuration.
    :vartype pro_rating_config: ~openmeter._generated.models.ProRatingConfig
    :ivar phases: Plan phases. Required.
    :vartype phases: list[~openmeter._generated.models.PlanPhase]
    """

    @overload
    def __init__(
        self,
        *,
        name: str,
        currency: str,
        billing_cadence: datetime.timedelta,
        phases: list["_models.PlanPhase"],
        description: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
        alignment: Optional["_models.Alignment"] = None,
        pro_rating_config: Optional["_models.ProRatingConfig"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class CustomSubscriptionChange(_Model):
    """Change a custom subscription.

    :ivar timing: Timing configuration for the change, when the change should take effect.
     For changing a subscription, the accepted values depend on the subscription configuration.
     Required. Is either a Union[str, "_models.SubscriptionTimingEnum"] type or a datetime.datetime
     type.
    :vartype timing: str or ~openmeter.models.SubscriptionTimingEnum or ~datetime.datetime
    :ivar billing_anchor: The billing anchor of the subscription. The provided date will be
     normalized according to the billing cadence to the nearest recurrence before start time. If not
     provided, the previous subscription billing anchor will be used.
    :vartype billing_anchor: ~datetime.datetime
    :ivar custom_plan: The custom plan description which defines the Subscription. Required.
    :vartype custom_plan: ~openmeter._generated.models.CustomPlanInput
    """

    timing: "_types.SubscriptionTiming" = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Timing configuration for the change, when the change should take effect.
     For changing a subscription, the accepted values depend on the subscription configuration.
     Required. Is either a Union[str, \"_models.SubscriptionTimingEnum\"] type or a
     datetime.datetime type."""
    billing_anchor: Optional[datetime.datetime] = rest_field(
        name="billingAnchor", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The billing anchor of the subscription. The provided date will be normalized according to the
     billing cadence to the nearest recurrence before start time. If not provided, the previous
     subscription billing anchor will be used."""
    custom_plan: "_models.CustomPlanInput" = rest_field(
        name="customPlan", visibility=["read", "create", "update", "delete", "query"]
    )
    """The custom plan description which defines the Subscription. Required."""

    @overload
    def __init__(
        self,
        *,
        timing: "_types.SubscriptionTiming",
        custom_plan: "_models.CustomPlanInput",
        billing_anchor: Optional[datetime.datetime] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class CustomSubscriptionCreate(_Model):
    """Create custom.

    :ivar custom_plan: The custom plan description which defines the Subscription. Required.
    :vartype custom_plan: ~openmeter._generated.models.CustomPlanInput
    :ivar timing: Timing configuration for the change, when the change should take effect.
     The default is immediate. Is either a Union[str, "_models.SubscriptionTimingEnum"] type or a
     datetime.datetime type.
    :vartype timing: str or ~openmeter.models.SubscriptionTimingEnum or ~datetime.datetime
    :ivar customer_id: The ID of the customer. Provide either the key or ID. Has presedence over
     the key.
    :vartype customer_id: str
    :ivar customer_key: The key of the customer. Provide either the key or ID.
    :vartype customer_key: str
    :ivar billing_anchor: The billing anchor of the subscription. The provided date will be
     normalized according to the billing cadence to the nearest recurrence before start time. If not
     provided, the subscription start time will be used.
    :vartype billing_anchor: ~datetime.datetime
    """

    custom_plan: "_models.CustomPlanInput" = rest_field(
        name="customPlan", visibility=["read", "create", "update", "delete", "query"]
    )
    """The custom plan description which defines the Subscription. Required."""
    timing: Optional["_types.SubscriptionTiming"] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """Timing configuration for the change, when the change should take effect.
     The default is immediate. Is either a Union[str, \"_models.SubscriptionTimingEnum\"] type or a
     datetime.datetime type."""
    customer_id: Optional[str] = rest_field(
        name="customerId", visibility=["read", "create", "update", "delete", "query"]
    )
    """The ID of the customer. Provide either the key or ID. Has presedence over the key."""
    customer_key: Optional[str] = rest_field(
        name="customerKey", visibility=["read", "create", "update", "delete", "query"]
    )
    """The key of the customer. Provide either the key or ID."""
    billing_anchor: Optional[datetime.datetime] = rest_field(
        name="billingAnchor", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The billing anchor of the subscription. The provided date will be normalized according to the
     billing cadence to the nearest recurrence before start time. If not provided, the subscription
     start time will be used."""

    @overload
    def __init__(
        self,
        *,
        custom_plan: "_models.CustomPlanInput",
        timing: Optional["_types.SubscriptionTiming"] = None,
        customer_id: Optional[str] = None,
        customer_key: Optional[str] = None,
        billing_anchor: Optional[datetime.datetime] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class DiscountPercentage(_Model):
    """Percentage discount.

    :ivar percentage: Percentage. Required.
    :vartype percentage: float
    """

    percentage: float = rest_field(visibility=["read", "create", "update"])
    """Percentage. Required."""

    @overload
    def __init__(
        self,
        *,
        percentage: float,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class DiscountReasonMaximumSpend(_Model):
    """The reason for the discount is a maximum spend.

    :ivar type: Required.
    :vartype type: str or ~openmeter._generated.models.MAXIMUM_SPEND
    """

    type: Literal[DiscountReasonType.MAXIMUM_SPEND] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """Required."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[DiscountReasonType.MAXIMUM_SPEND],
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class DiscountReasonRatecardPercentage(_Model):
    """The reason for the discount is a ratecard percentage.

    :ivar type: Required.
    :vartype type: str or ~openmeter._generated.models.RATECARD_PERCENTAGE
    :ivar percentage: Percentage. Required.
    :vartype percentage: float
    :ivar correlation_id: Correlation ID for the discount.

     This is used to link discounts across different invoices (progressive billing use case).

     If not provided, the invoicing engine will auto-generate one. When editing an invoice line,
     please make sure to keep the same correlation ID of the discount or in progressive billing
     setups the discount amounts might be incorrect.
    :vartype correlation_id: str
    """

    type: Literal[DiscountReasonType.RATECARD_PERCENTAGE] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """Required."""
    percentage: float = rest_field(visibility=["read", "create", "update"])
    """Percentage. Required."""
    correlation_id: Optional[str] = rest_field(
        name="correlationId", visibility=["read", "create", "update", "delete", "query"]
    )
    """Correlation ID for the discount.
     
     This is used to link discounts across different invoices (progressive billing use case).
     
     If not provided, the invoicing engine will auto-generate one. When editing an invoice line,
     please make sure to keep the same correlation ID of the discount or in progressive billing
     setups the discount amounts might be incorrect."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[DiscountReasonType.RATECARD_PERCENTAGE],
        percentage: float,
        correlation_id: Optional[str] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class DiscountReasonRatecardUsage(_Model):
    """The reason for the discount is a ratecard usage.

    :ivar type: Required.
    :vartype type: str or ~openmeter._generated.models.RATECARD_USAGE
    :ivar quantity: Usage. Required.
    :vartype quantity: str
    :ivar correlation_id: Correlation ID for the discount.

     This is used to link discounts across different invoices (progressive billing use case).

     If not provided, the invoicing engine will auto-generate one. When editing an invoice line,
     please make sure to keep the same correlation ID of the discount or in progressive billing
     setups the discount amounts might be incorrect.
    :vartype correlation_id: str
    """

    type: Literal[DiscountReasonType.RATECARD_USAGE] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """Required."""
    quantity: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Usage. Required."""
    correlation_id: Optional[str] = rest_field(
        name="correlationId", visibility=["read", "create", "update", "delete", "query"]
    )
    """Correlation ID for the discount.
     
     This is used to link discounts across different invoices (progressive billing use case).
     
     If not provided, the invoicing engine will auto-generate one. When editing an invoice line,
     please make sure to keep the same correlation ID of the discount or in progressive billing
     setups the discount amounts might be incorrect."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[DiscountReasonType.RATECARD_USAGE],
        quantity: str,
        correlation_id: Optional[str] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class Discounts(_Model):
    """Discount by type on a price.

    :ivar percentage: The percentage discount.
    :vartype percentage: ~openmeter._generated.models.DiscountPercentage
    :ivar usage: The usage discount.
    :vartype usage: ~openmeter._generated.models.DiscountUsage
    """

    percentage: Optional["_models.DiscountPercentage"] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """The percentage discount."""
    usage: Optional["_models.DiscountUsage"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The usage discount."""

    @overload
    def __init__(
        self,
        *,
        percentage: Optional["_models.DiscountPercentage"] = None,
        usage: Optional["_models.DiscountUsage"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class DiscountUsage(_Model):
    """Usage discount.

    Usage discount means that the first N items are free. From billing perspective
    this means that any usage on a specific feature is considered 0 until this discount
    is exhausted.

    :ivar quantity: Usage. Required.
    :vartype quantity: str
    """

    quantity: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Usage. Required."""

    @overload
    def __init__(
        self,
        *,
        quantity: str,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class DynamicPriceWithCommitments(_Model):
    """Dynamic price with spend commitments.

    :ivar type: The type of the price. Required.
    :vartype type: str or ~openmeter._generated.models.DYNAMIC
    :ivar multiplier: The multiplier to apply to the base price to get the dynamic price.
    :vartype multiplier: str
    :ivar minimum_amount: Minimum amount.
    :vartype minimum_amount: str
    :ivar maximum_amount: Maximum amount.
    :vartype maximum_amount: str
    """

    type: Literal[PriceType.DYNAMIC] = rest_field(visibility=["read", "create", "update"])
    """The type of the price. Required."""
    multiplier: Optional[str] = rest_field(visibility=["read", "create", "update"])
    """The multiplier to apply to the base price to get the dynamic price."""
    minimum_amount: Optional[str] = rest_field(name="minimumAmount", visibility=["read", "create", "update"])
    """Minimum amount."""
    maximum_amount: Optional[str] = rest_field(name="maximumAmount", visibility=["read", "create", "update"])
    """Maximum amount."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[PriceType.DYNAMIC],
        multiplier: Optional[str] = None,
        minimum_amount: Optional[str] = None,
        maximum_amount: Optional[str] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class EditSubscriptionAddItem(_Model):
    """Add a new item to a phase.

    :ivar op: Required.
    :vartype op: str or ~openmeter._generated.models.ADD_ITEM
    :ivar phase_key: Required.
    :vartype phase_key: str
    :ivar rate_card: Required. Is either a RateCardFlatFee type or a RateCardUsageBased type.
    :vartype rate_card: ~openmeter._generated.models.RateCardFlatFee or
     ~openmeter._generated.models.RateCardUsageBased
    """

    op: Literal[EditOp.ADD_ITEM] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Required."""
    phase_key: str = rest_field(name="phaseKey", visibility=["read", "create", "update", "delete", "query"])
    """Required."""
    rate_card: "_types.RateCard" = rest_field(
        name="rateCard", visibility=["read", "create", "update", "delete", "query"]
    )
    """Required. Is either a RateCardFlatFee type or a RateCardUsageBased type."""

    @overload
    def __init__(
        self,
        *,
        op: Literal[EditOp.ADD_ITEM],
        phase_key: str,
        rate_card: "_types.RateCard",
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class EditSubscriptionAddPhase(_Model):
    """Add a new phase.

    :ivar op: Required.
    :vartype op: str or ~openmeter._generated.models.ADD_PHASE
    :ivar phase: Required.
    :vartype phase: ~openmeter._generated.models.SubscriptionPhaseCreate
    """

    op: Literal[EditOp.ADD_PHASE] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Required."""
    phase: "_models.SubscriptionPhaseCreate" = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Required."""

    @overload
    def __init__(
        self,
        *,
        op: Literal[EditOp.ADD_PHASE],
        phase: "_models.SubscriptionPhaseCreate",
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class EditSubscriptionRemoveItem(_Model):
    """Remove an item from a phase.

    :ivar op: Required.
    :vartype op: str or ~openmeter._generated.models.REMOVE_ITEM
    :ivar phase_key: Required.
    :vartype phase_key: str
    :ivar item_key: Required.
    :vartype item_key: str
    """

    op: Literal[EditOp.REMOVE_ITEM] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Required."""
    phase_key: str = rest_field(name="phaseKey", visibility=["read", "create", "update", "delete", "query"])
    """Required."""
    item_key: str = rest_field(name="itemKey", visibility=["read", "create", "update", "delete", "query"])
    """Required."""

    @overload
    def __init__(
        self,
        *,
        op: Literal[EditOp.REMOVE_ITEM],
        phase_key: str,
        item_key: str,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class EditSubscriptionRemovePhase(_Model):
    """Remove a phase.

    :ivar op: Required.
    :vartype op: str or ~openmeter._generated.models.REMOVE_PHASE
    :ivar phase_key: Required.
    :vartype phase_key: str
    :ivar shift: Required. Known values are: "next" and "prev".
    :vartype shift: str or ~openmeter.models.RemovePhaseShifting
    """

    op: Literal[EditOp.REMOVE_PHASE] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Required."""
    phase_key: str = rest_field(name="phaseKey", visibility=["read", "create", "update", "delete", "query"])
    """Required."""
    shift: Union[str, "_models.RemovePhaseShifting"] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """Required. Known values are: \"next\" and \"prev\"."""

    @overload
    def __init__(
        self,
        *,
        op: Literal[EditOp.REMOVE_PHASE],
        phase_key: str,
        shift: Union[str, "_models.RemovePhaseShifting"],
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class EditSubscriptionStretchPhase(_Model):
    """Stretch a phase.

    :ivar op: Required.
    :vartype op: str or ~openmeter._generated.models.STRETCH_PHASE
    :ivar phase_key: Required.
    :vartype phase_key: str
    :ivar extend_by: Required.
    :vartype extend_by: ~datetime.timedelta
    """

    op: Literal[EditOp.STRETCH_PHASE] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Required."""
    phase_key: str = rest_field(name="phaseKey", visibility=["read", "create", "update", "delete", "query"])
    """Required."""
    extend_by: datetime.timedelta = rest_field(
        name="extendBy", visibility=["read", "create", "update", "delete", "query"]
    )
    """Required."""

    @overload
    def __init__(
        self,
        *,
        op: Literal[EditOp.STRETCH_PHASE],
        phase_key: str,
        extend_by: datetime.timedelta,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class EditSubscriptionUnscheduleEdit(_Model):
    """Unschedules any edits from the current phase.

    :ivar op: Required.
    :vartype op: str or ~openmeter._generated.models.UNSCHEDULE_EDIT
    """

    op: Literal[EditOp.UNSCHEDULE_EDIT] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Required."""

    @overload
    def __init__(
        self,
        *,
        op: Literal[EditOp.UNSCHEDULE_EDIT],
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class EntitlementBoolean(_Model):
    """Entitlement template of a boolean entitlement.

    :ivar type: Required.
    :vartype type: str or ~openmeter._generated.models.BOOLEAN
    :ivar created_at: Creation Time. Required.
    :vartype created_at: ~datetime.datetime
    :ivar updated_at: Last Update Time. Required.
    :vartype updated_at: ~datetime.datetime
    :ivar deleted_at: Deletion Time.
    :vartype deleted_at: ~datetime.datetime
    :ivar metadata: Additional metadata for the feature.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar active_from: The cadence start of the resource. Required.
    :vartype active_from: ~datetime.datetime
    :ivar active_to: The cadence end of the resource.
    :vartype active_to: ~datetime.datetime
    :ivar annotations: The annotations of the entitlement.
    :vartype annotations: ~openmeter._generated.models.Annotations
    :ivar id: Readonly unique ULID identifier. Required.
    :vartype id: str
    :ivar subject_key: The identifier key unique to the subject.
     NOTE: Subjects are being deprecated, please use the new customer APIs. Required.
    :vartype subject_key: str
    :ivar feature_key: The feature the subject is entitled to use. Required.
    :vartype feature_key: str
    :ivar feature_id: The feature the subject is entitled to use. Required.
    :vartype feature_id: str
    :ivar current_usage_period: The current usage period.
    :vartype current_usage_period: ~openmeter._generated.models.Period
    :ivar usage_period: The defined usage period of the entitlement.
    :vartype usage_period: ~openmeter._generated.models.RecurringPeriod
    """

    type: Literal[EntitlementType.BOOLEAN] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Required."""
    created_at: datetime.datetime = rest_field(name="createdAt", visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    updated_at: datetime.datetime = rest_field(name="updatedAt", visibility=["read"], format="rfc3339")
    """Last Update Time. Required."""
    deleted_at: Optional[datetime.datetime] = rest_field(name="deletedAt", visibility=["read"], format="rfc3339")
    """Deletion Time."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Additional metadata for the feature."""
    active_from: datetime.datetime = rest_field(
        name="activeFrom", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The cadence start of the resource. Required."""
    active_to: Optional[datetime.datetime] = rest_field(
        name="activeTo", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The cadence end of the resource."""
    annotations: Optional["_models.Annotations"] = rest_field(visibility=["read"])
    """The annotations of the entitlement."""
    id: str = rest_field(visibility=["read"])
    """Readonly unique ULID identifier. Required."""
    subject_key: str = rest_field(name="subjectKey", visibility=["read", "create", "update", "delete", "query"])
    """The identifier key unique to the subject.
     NOTE: Subjects are being deprecated, please use the new customer APIs. Required."""
    feature_key: str = rest_field(name="featureKey", visibility=["read", "create", "update", "delete", "query"])
    """The feature the subject is entitled to use. Required."""
    feature_id: str = rest_field(name="featureId", visibility=["read", "create", "update", "delete", "query"])
    """The feature the subject is entitled to use. Required."""
    current_usage_period: Optional["_models.Period"] = rest_field(
        name="currentUsagePeriod", visibility=["read", "create", "update", "delete", "query"]
    )
    """The current usage period."""
    usage_period: Optional["_models.RecurringPeriod"] = rest_field(
        name="usagePeriod", visibility=["read", "create", "update", "delete", "query"]
    )
    """The defined usage period of the entitlement."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[EntitlementType.BOOLEAN],
        active_from: datetime.datetime,
        subject_key: str,
        feature_key: str,
        feature_id: str,
        metadata: Optional["_models.Metadata"] = None,
        active_to: Optional[datetime.datetime] = None,
        current_usage_period: Optional["_models.Period"] = None,
        usage_period: Optional["_models.RecurringPeriod"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class EntitlementBooleanCreateInputs(_Model):
    """Create inputs for boolean entitlement.

    :ivar feature_key: The feature the subject is entitled to use.
     Either featureKey or featureId is required.
    :vartype feature_key: str
    :ivar feature_id: The feature the subject is entitled to use.
     Either featureKey or featureId is required.
    :vartype feature_id: str
    :ivar metadata: Additional metadata for the feature.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar usage_period: The usage period associated with the entitlement.
    :vartype usage_period: ~openmeter._generated.models.RecurringPeriodCreateInput
    :ivar type: Required.
    :vartype type: str or ~openmeter._generated.models.BOOLEAN
    """

    feature_key: Optional[str] = rest_field(
        name="featureKey", visibility=["read", "create", "update", "delete", "query"]
    )
    """The feature the subject is entitled to use.
     Either featureKey or featureId is required."""
    feature_id: Optional[str] = rest_field(name="featureId", visibility=["read", "create", "update", "delete", "query"])
    """The feature the subject is entitled to use.
     Either featureKey or featureId is required."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Additional metadata for the feature."""
    usage_period: Optional["_models.RecurringPeriodCreateInput"] = rest_field(
        name="usagePeriod", visibility=["read", "create", "update", "delete", "query"]
    )
    """The usage period associated with the entitlement."""
    type: Literal[EntitlementType.BOOLEAN] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Required."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[EntitlementType.BOOLEAN],
        feature_key: Optional[str] = None,
        feature_id: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
        usage_period: Optional["_models.RecurringPeriodCreateInput"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class EntitlementBooleanV2(_Model):
    """Entitlement template of a boolean entitlement.

    :ivar type: Required.
    :vartype type: str or ~openmeter._generated.models.BOOLEAN
    :ivar created_at: Creation Time. Required.
    :vartype created_at: ~datetime.datetime
    :ivar updated_at: Last Update Time. Required.
    :vartype updated_at: ~datetime.datetime
    :ivar deleted_at: Deletion Time.
    :vartype deleted_at: ~datetime.datetime
    :ivar metadata: Additional metadata for the feature.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar active_from: The cadence start of the resource. Required.
    :vartype active_from: ~datetime.datetime
    :ivar active_to: The cadence end of the resource.
    :vartype active_to: ~datetime.datetime
    :ivar annotations: The annotations of the entitlement.
    :vartype annotations: ~openmeter._generated.models.Annotations
    :ivar id: Readonly unique ULID identifier. Required.
    :vartype id: str
    :ivar feature_key: The feature the subject is entitled to use. Required.
    :vartype feature_key: str
    :ivar feature_id: The feature the subject is entitled to use. Required.
    :vartype feature_id: str
    :ivar current_usage_period: The current usage period.
    :vartype current_usage_period: ~openmeter._generated.models.Period
    :ivar usage_period: The defined usage period of the entitlement.
    :vartype usage_period: ~openmeter._generated.models.RecurringPeriod
    :ivar customer_key: The identifier key unique to the customer.
    :vartype customer_key: str
    :ivar customer_id: The identifier unique to the customer. Required.
    :vartype customer_id: str
    """

    type: Literal[EntitlementType.BOOLEAN] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Required."""
    created_at: datetime.datetime = rest_field(name="createdAt", visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    updated_at: datetime.datetime = rest_field(name="updatedAt", visibility=["read"], format="rfc3339")
    """Last Update Time. Required."""
    deleted_at: Optional[datetime.datetime] = rest_field(name="deletedAt", visibility=["read"], format="rfc3339")
    """Deletion Time."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Additional metadata for the feature."""
    active_from: datetime.datetime = rest_field(
        name="activeFrom", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The cadence start of the resource. Required."""
    active_to: Optional[datetime.datetime] = rest_field(
        name="activeTo", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The cadence end of the resource."""
    annotations: Optional["_models.Annotations"] = rest_field(visibility=["read"])
    """The annotations of the entitlement."""
    id: str = rest_field(visibility=["read"])
    """Readonly unique ULID identifier. Required."""
    feature_key: str = rest_field(name="featureKey", visibility=["read", "create", "update", "delete", "query"])
    """The feature the subject is entitled to use. Required."""
    feature_id: str = rest_field(name="featureId", visibility=["read", "create", "update", "delete", "query"])
    """The feature the subject is entitled to use. Required."""
    current_usage_period: Optional["_models.Period"] = rest_field(
        name="currentUsagePeriod", visibility=["read", "create", "update", "delete", "query"]
    )
    """The current usage period."""
    usage_period: Optional["_models.RecurringPeriod"] = rest_field(
        name="usagePeriod", visibility=["read", "create", "update", "delete", "query"]
    )
    """The defined usage period of the entitlement."""
    customer_key: Optional[str] = rest_field(
        name="customerKey", visibility=["read", "create", "update", "delete", "query"]
    )
    """The identifier key unique to the customer."""
    customer_id: str = rest_field(name="customerId", visibility=["read", "create", "update", "delete", "query"])
    """The identifier unique to the customer. Required."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[EntitlementType.BOOLEAN],
        active_from: datetime.datetime,
        feature_key: str,
        feature_id: str,
        customer_id: str,
        metadata: Optional["_models.Metadata"] = None,
        active_to: Optional[datetime.datetime] = None,
        current_usage_period: Optional["_models.Period"] = None,
        usage_period: Optional["_models.RecurringPeriod"] = None,
        customer_key: Optional[str] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class EntitlementGrant(_Model):
    """The grant.

    :ivar created_at: Creation Time. Required.
    :vartype created_at: ~datetime.datetime
    :ivar updated_at: Last Update Time. Required.
    :vartype updated_at: ~datetime.datetime
    :ivar deleted_at: Deletion Time.
    :vartype deleted_at: ~datetime.datetime
    :ivar amount: The amount to grant. Should be a positive number. Required.
    :vartype amount: float
    :ivar priority: The priority of the grant. Grants with higher priority are applied first.
     Priority is a positive decimal numbers. With lower numbers indicating higher importance.
     For example, a priority of 1 is more urgent than a priority of 2.
     When there are several grants available for the same subject, the system selects the grant with
     the highest priority.
     In cases where grants share the same priority level, the grant closest to its expiration will
     be used first.
     In the case of two grants have identical priorities and expiration dates, the system will use
     the grant that was created first.
    :vartype priority: int
    :ivar effective_at: Effective date for grants and anchor for recurring grants. Provided value
     will be ceiled to metering windowSize (minute). Required.
    :vartype effective_at: ~datetime.datetime
    :ivar expiration: The grant expiration definition. Required.
    :vartype expiration: ~openmeter._generated.models.ExpirationPeriod
    :ivar max_rollover_amount: Grants are rolled over at reset, after which they can have a
     different balance compared to what they had before the reset.
     Balance after the reset is calculated as: Balance_After_Reset = MIN(MaxRolloverAmount,
     MAX(Balance_Before_Reset, MinRolloverAmount)).
    :vartype max_rollover_amount: float
    :ivar min_rollover_amount: Grants are rolled over at reset, after which they can have a
     different balance compared to what they had before the reset.
     Balance after the reset is calculated as: Balance_After_Reset = MIN(MaxRolloverAmount,
     MAX(Balance_Before_Reset, MinRolloverAmount)).
    :vartype min_rollover_amount: float
    :ivar metadata: The grant metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar id: Readonly unique ULID identifier. Required.
    :vartype id: str
    :ivar entitlement_id: The unique entitlement ULID that the grant is associated with. Required.
    :vartype entitlement_id: str
    :ivar next_recurrence: The next time the grant will recurr.
    :vartype next_recurrence: ~datetime.datetime
    :ivar expires_at: The time the grant expires.
    :vartype expires_at: ~datetime.datetime
    :ivar voided_at: The time the grant was voided.
    :vartype voided_at: ~datetime.datetime
    :ivar recurrence: The recurrence period of the grant.
    :vartype recurrence: ~openmeter._generated.models.RecurringPeriod
    :ivar annotations: Grant annotations.
    :vartype annotations: ~openmeter._generated.models.Annotations
    """

    created_at: datetime.datetime = rest_field(name="createdAt", visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    updated_at: datetime.datetime = rest_field(name="updatedAt", visibility=["read"], format="rfc3339")
    """Last Update Time. Required."""
    deleted_at: Optional[datetime.datetime] = rest_field(name="deletedAt", visibility=["read"], format="rfc3339")
    """Deletion Time."""
    amount: float = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The amount to grant. Should be a positive number. Required."""
    priority: Optional[int] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The priority of the grant. Grants with higher priority are applied first.
     Priority is a positive decimal numbers. With lower numbers indicating higher importance.
     For example, a priority of 1 is more urgent than a priority of 2.
     When there are several grants available for the same subject, the system selects the grant with
     the highest priority.
     In cases where grants share the same priority level, the grant closest to its expiration will
     be used first.
     In the case of two grants have identical priorities and expiration dates, the system will use
     the grant that was created first."""
    effective_at: datetime.datetime = rest_field(
        name="effectiveAt", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """Effective date for grants and anchor for recurring grants. Provided value will be ceiled to
     metering windowSize (minute). Required."""
    expiration: "_models.ExpirationPeriod" = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The grant expiration definition. Required."""
    max_rollover_amount: Optional[float] = rest_field(
        name="maxRolloverAmount", visibility=["read", "create", "update", "delete", "query"]
    )
    """Grants are rolled over at reset, after which they can have a different balance compared to what
     they had before the reset.
     Balance after the reset is calculated as: Balance_After_Reset = MIN(MaxRolloverAmount,
     MAX(Balance_Before_Reset, MinRolloverAmount))."""
    min_rollover_amount: Optional[float] = rest_field(
        name="minRolloverAmount", visibility=["read", "create", "update", "delete", "query"]
    )
    """Grants are rolled over at reset, after which they can have a different balance compared to what
     they had before the reset.
     Balance after the reset is calculated as: Balance_After_Reset = MIN(MaxRolloverAmount,
     MAX(Balance_Before_Reset, MinRolloverAmount))."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The grant metadata."""
    id: str = rest_field(visibility=["read"])
    """Readonly unique ULID identifier. Required."""
    entitlement_id: str = rest_field(name="entitlementId", visibility=["read"])
    """The unique entitlement ULID that the grant is associated with. Required."""
    next_recurrence: Optional[datetime.datetime] = rest_field(
        name="nextRecurrence", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The next time the grant will recurr."""
    expires_at: Optional[datetime.datetime] = rest_field(name="expiresAt", visibility=["read"], format="rfc3339")
    """The time the grant expires."""
    voided_at: Optional[datetime.datetime] = rest_field(
        name="voidedAt", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The time the grant was voided."""
    recurrence: Optional["_models.RecurringPeriod"] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """The recurrence period of the grant."""
    annotations: Optional["_models.Annotations"] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """Grant annotations."""

    @overload
    def __init__(
        self,
        *,
        amount: float,
        effective_at: datetime.datetime,
        expiration: "_models.ExpirationPeriod",
        priority: Optional[int] = None,
        max_rollover_amount: Optional[float] = None,
        min_rollover_amount: Optional[float] = None,
        metadata: Optional["_models.Metadata"] = None,
        next_recurrence: Optional[datetime.datetime] = None,
        voided_at: Optional[datetime.datetime] = None,
        recurrence: Optional["_models.RecurringPeriod"] = None,
        annotations: Optional["_models.Annotations"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class EntitlementGrantCreateInput(_Model):
    """The grant creation input.

    :ivar amount: The amount to grant. Should be a positive number. Required.
    :vartype amount: float
    :ivar priority: The priority of the grant. Grants with higher priority are applied first.
     Priority is a positive decimal numbers. With lower numbers indicating higher importance.
     For example, a priority of 1 is more urgent than a priority of 2.
     When there are several grants available for the same subject, the system selects the grant with
     the highest priority.
     In cases where grants share the same priority level, the grant closest to its expiration will
     be used first.
     In the case of two grants have identical priorities and expiration dates, the system will use
     the grant that was created first.
    :vartype priority: int
    :ivar effective_at: Effective date for grants and anchor for recurring grants. Provided value
     will be ceiled to metering windowSize (minute). Required.
    :vartype effective_at: ~datetime.datetime
    :ivar expiration: The grant expiration definition. Required.
    :vartype expiration: ~openmeter._generated.models.ExpirationPeriod
    :ivar max_rollover_amount: Grants are rolled over at reset, after which they can have a
     different balance compared to what they had before the reset.
     Balance after the reset is calculated as: Balance_After_Reset = MIN(MaxRolloverAmount,
     MAX(Balance_Before_Reset, MinRolloverAmount)).
    :vartype max_rollover_amount: float
    :ivar min_rollover_amount: Grants are rolled over at reset, after which they can have a
     different balance compared to what they had before the reset.
     Balance after the reset is calculated as: Balance_After_Reset = MIN(MaxRolloverAmount,
     MAX(Balance_Before_Reset, MinRolloverAmount)).
    :vartype min_rollover_amount: float
    :ivar metadata: The grant metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar recurrence: The subject of the grant.
    :vartype recurrence: ~openmeter._generated.models.RecurringPeriodCreateInput
    """

    amount: float = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The amount to grant. Should be a positive number. Required."""
    priority: Optional[int] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The priority of the grant. Grants with higher priority are applied first.
     Priority is a positive decimal numbers. With lower numbers indicating higher importance.
     For example, a priority of 1 is more urgent than a priority of 2.
     When there are several grants available for the same subject, the system selects the grant with
     the highest priority.
     In cases where grants share the same priority level, the grant closest to its expiration will
     be used first.
     In the case of two grants have identical priorities and expiration dates, the system will use
     the grant that was created first."""
    effective_at: datetime.datetime = rest_field(
        name="effectiveAt", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """Effective date for grants and anchor for recurring grants. Provided value will be ceiled to
     metering windowSize (minute). Required."""
    expiration: "_models.ExpirationPeriod" = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The grant expiration definition. Required."""
    max_rollover_amount: Optional[float] = rest_field(
        name="maxRolloverAmount", visibility=["read", "create", "update", "delete", "query"]
    )
    """Grants are rolled over at reset, after which they can have a different balance compared to what
     they had before the reset.
     Balance after the reset is calculated as: Balance_After_Reset = MIN(MaxRolloverAmount,
     MAX(Balance_Before_Reset, MinRolloverAmount))."""
    min_rollover_amount: Optional[float] = rest_field(
        name="minRolloverAmount", visibility=["read", "create", "update", "delete", "query"]
    )
    """Grants are rolled over at reset, after which they can have a different balance compared to what
     they had before the reset.
     Balance after the reset is calculated as: Balance_After_Reset = MIN(MaxRolloverAmount,
     MAX(Balance_Before_Reset, MinRolloverAmount))."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The grant metadata."""
    recurrence: Optional["_models.RecurringPeriodCreateInput"] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """The subject of the grant."""

    @overload
    def __init__(
        self,
        *,
        amount: float,
        effective_at: datetime.datetime,
        expiration: "_models.ExpirationPeriod",
        priority: Optional[int] = None,
        max_rollover_amount: Optional[float] = None,
        min_rollover_amount: Optional[float] = None,
        metadata: Optional["_models.Metadata"] = None,
        recurrence: Optional["_models.RecurringPeriodCreateInput"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class EntitlementGrantCreateInputV2(_Model):
    """The grant creation input.

    :ivar amount: The amount to grant. Should be a positive number. Required.
    :vartype amount: float
    :ivar priority: The priority of the grant. Grants with higher priority are applied first.
     Priority is a positive decimal numbers. With lower numbers indicating higher importance.
     For example, a priority of 1 is more urgent than a priority of 2.
     When there are several grants available for the same subject, the system selects the grant with
     the highest priority.
     In cases where grants share the same priority level, the grant closest to its expiration will
     be used first.
     In the case of two grants have identical priorities and expiration dates, the system will use
     the grant that was created first.
    :vartype priority: int
    :ivar effective_at: Effective date for grants and anchor for recurring grants. Provided value
     will be ceiled to metering windowSize (minute). Required.
    :vartype effective_at: ~datetime.datetime
    :ivar min_rollover_amount: Grants are rolled over at reset, after which they can have a
     different balance compared to what they had before the reset.
     Balance after the reset is calculated as: Balance_After_Reset = MIN(MaxRolloverAmount,
     MAX(Balance_Before_Reset, MinRolloverAmount)).
    :vartype min_rollover_amount: float
    :ivar metadata: The grant metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar recurrence: The subject of the grant.
    :vartype recurrence: ~openmeter._generated.models.RecurringPeriodCreateInput
    :ivar max_rollover_amount: Grants are rolled over at reset, after which they can have a
     different balance compared to what they had before the reset. The default value equals grant
     amount.
     Balance after the reset is calculated as: Balance_After_Reset = MIN(MaxRolloverAmount,
     MAX(Balance_Before_Reset, MinRolloverAmount)).
    :vartype max_rollover_amount: float
    :ivar expiration: The grant expiration definition. If no expiration is provided, the grant can
     be active indefinitely.
    :vartype expiration: ~openmeter._generated.models.ExpirationPeriod
    :ivar annotations: Grant annotations.
    :vartype annotations: ~openmeter._generated.models.Annotations
    """

    amount: float = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The amount to grant. Should be a positive number. Required."""
    priority: Optional[int] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The priority of the grant. Grants with higher priority are applied first.
     Priority is a positive decimal numbers. With lower numbers indicating higher importance.
     For example, a priority of 1 is more urgent than a priority of 2.
     When there are several grants available for the same subject, the system selects the grant with
     the highest priority.
     In cases where grants share the same priority level, the grant closest to its expiration will
     be used first.
     In the case of two grants have identical priorities and expiration dates, the system will use
     the grant that was created first."""
    effective_at: datetime.datetime = rest_field(
        name="effectiveAt", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """Effective date for grants and anchor for recurring grants. Provided value will be ceiled to
     metering windowSize (minute). Required."""
    min_rollover_amount: Optional[float] = rest_field(
        name="minRolloverAmount", visibility=["read", "create", "update", "delete", "query"]
    )
    """Grants are rolled over at reset, after which they can have a different balance compared to what
     they had before the reset.
     Balance after the reset is calculated as: Balance_After_Reset = MIN(MaxRolloverAmount,
     MAX(Balance_Before_Reset, MinRolloverAmount))."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The grant metadata."""
    recurrence: Optional["_models.RecurringPeriodCreateInput"] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """The subject of the grant."""
    max_rollover_amount: Optional[float] = rest_field(
        name="maxRolloverAmount", visibility=["read", "create", "update", "delete", "query"]
    )
    """Grants are rolled over at reset, after which they can have a different balance compared to what
     they had before the reset. The default value equals grant amount.
     Balance after the reset is calculated as: Balance_After_Reset = MIN(MaxRolloverAmount,
     MAX(Balance_Before_Reset, MinRolloverAmount))."""
    expiration: Optional["_models.ExpirationPeriod"] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """The grant expiration definition. If no expiration is provided, the grant can be active
     indefinitely."""
    annotations: Optional["_models.Annotations"] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """Grant annotations."""

    @overload
    def __init__(
        self,
        *,
        amount: float,
        effective_at: datetime.datetime,
        priority: Optional[int] = None,
        min_rollover_amount: Optional[float] = None,
        metadata: Optional["_models.Metadata"] = None,
        recurrence: Optional["_models.RecurringPeriodCreateInput"] = None,
        max_rollover_amount: Optional[float] = None,
        expiration: Optional["_models.ExpirationPeriod"] = None,
        annotations: Optional["_models.Annotations"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class EntitlementGrantV2(_Model):
    """The grant.

    :ivar created_at: Creation Time. Required.
    :vartype created_at: ~datetime.datetime
    :ivar updated_at: Last Update Time. Required.
    :vartype updated_at: ~datetime.datetime
    :ivar deleted_at: Deletion Time.
    :vartype deleted_at: ~datetime.datetime
    :ivar amount: The amount to grant. Should be a positive number. Required.
    :vartype amount: float
    :ivar priority: The priority of the grant. Grants with higher priority are applied first.
     Priority is a positive decimal numbers. With lower numbers indicating higher importance.
     For example, a priority of 1 is more urgent than a priority of 2.
     When there are several grants available for the same subject, the system selects the grant with
     the highest priority.
     In cases where grants share the same priority level, the grant closest to its expiration will
     be used first.
     In the case of two grants have identical priorities and expiration dates, the system will use
     the grant that was created first.
    :vartype priority: int
    :ivar effective_at: Effective date for grants and anchor for recurring grants. Provided value
     will be ceiled to metering windowSize (minute). Required.
    :vartype effective_at: ~datetime.datetime
    :ivar min_rollover_amount: Grants are rolled over at reset, after which they can have a
     different balance compared to what they had before the reset.
     Balance after the reset is calculated as: Balance_After_Reset = MIN(MaxRolloverAmount,
     MAX(Balance_Before_Reset, MinRolloverAmount)).
    :vartype min_rollover_amount: float
    :ivar metadata: The grant metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar max_rollover_amount: Grants are rolled over at reset, after which they can have a
     different balance compared to what they had before the reset. The default value equals grant
     amount.
     Balance after the reset is calculated as: Balance_After_Reset = MIN(MaxRolloverAmount,
     MAX(Balance_Before_Reset, MinRolloverAmount)).
    :vartype max_rollover_amount: float
    :ivar expiration: The grant expiration definition. If no expiration is provided, the grant can
     be active indefinitely.
    :vartype expiration: ~openmeter._generated.models.ExpirationPeriod
    :ivar annotations: Grant annotations.
    :vartype annotations: ~openmeter._generated.models.Annotations
    :ivar id: Readonly unique ULID identifier. Required.
    :vartype id: str
    :ivar entitlement_id: The unique entitlement ULID that the grant is associated with. Required.
    :vartype entitlement_id: str
    :ivar next_recurrence: The next time the grant will recurr.
    :vartype next_recurrence: ~datetime.datetime
    :ivar expires_at: The time the grant expires.
    :vartype expires_at: ~datetime.datetime
    :ivar voided_at: The time the grant was voided.
    :vartype voided_at: ~datetime.datetime
    :ivar recurrence: The recurrence period of the grant.
    :vartype recurrence: ~openmeter._generated.models.RecurringPeriod
    """

    created_at: datetime.datetime = rest_field(name="createdAt", visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    updated_at: datetime.datetime = rest_field(name="updatedAt", visibility=["read"], format="rfc3339")
    """Last Update Time. Required."""
    deleted_at: Optional[datetime.datetime] = rest_field(name="deletedAt", visibility=["read"], format="rfc3339")
    """Deletion Time."""
    amount: float = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The amount to grant. Should be a positive number. Required."""
    priority: Optional[int] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The priority of the grant. Grants with higher priority are applied first.
     Priority is a positive decimal numbers. With lower numbers indicating higher importance.
     For example, a priority of 1 is more urgent than a priority of 2.
     When there are several grants available for the same subject, the system selects the grant with
     the highest priority.
     In cases where grants share the same priority level, the grant closest to its expiration will
     be used first.
     In the case of two grants have identical priorities and expiration dates, the system will use
     the grant that was created first."""
    effective_at: datetime.datetime = rest_field(
        name="effectiveAt", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """Effective date for grants and anchor for recurring grants. Provided value will be ceiled to
     metering windowSize (minute). Required."""
    min_rollover_amount: Optional[float] = rest_field(
        name="minRolloverAmount", visibility=["read", "create", "update", "delete", "query"]
    )
    """Grants are rolled over at reset, after which they can have a different balance compared to what
     they had before the reset.
     Balance after the reset is calculated as: Balance_After_Reset = MIN(MaxRolloverAmount,
     MAX(Balance_Before_Reset, MinRolloverAmount))."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The grant metadata."""
    max_rollover_amount: Optional[float] = rest_field(
        name="maxRolloverAmount", visibility=["read", "create", "update", "delete", "query"]
    )
    """Grants are rolled over at reset, after which they can have a different balance compared to what
     they had before the reset. The default value equals grant amount.
     Balance after the reset is calculated as: Balance_After_Reset = MIN(MaxRolloverAmount,
     MAX(Balance_Before_Reset, MinRolloverAmount))."""
    expiration: Optional["_models.ExpirationPeriod"] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """The grant expiration definition. If no expiration is provided, the grant can be active
     indefinitely."""
    annotations: Optional["_models.Annotations"] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """Grant annotations."""
    id: str = rest_field(visibility=["read"])
    """Readonly unique ULID identifier. Required."""
    entitlement_id: str = rest_field(name="entitlementId", visibility=["read"])
    """The unique entitlement ULID that the grant is associated with. Required."""
    next_recurrence: Optional[datetime.datetime] = rest_field(
        name="nextRecurrence", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The next time the grant will recurr."""
    expires_at: Optional[datetime.datetime] = rest_field(name="expiresAt", visibility=["read"], format="rfc3339")
    """The time the grant expires."""
    voided_at: Optional[datetime.datetime] = rest_field(
        name="voidedAt", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The time the grant was voided."""
    recurrence: Optional["_models.RecurringPeriod"] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """The recurrence period of the grant."""

    @overload
    def __init__(
        self,
        *,
        amount: float,
        effective_at: datetime.datetime,
        priority: Optional[int] = None,
        min_rollover_amount: Optional[float] = None,
        metadata: Optional["_models.Metadata"] = None,
        max_rollover_amount: Optional[float] = None,
        expiration: Optional["_models.ExpirationPeriod"] = None,
        annotations: Optional["_models.Annotations"] = None,
        next_recurrence: Optional[datetime.datetime] = None,
        voided_at: Optional[datetime.datetime] = None,
        recurrence: Optional["_models.RecurringPeriod"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class EntitlementMetered(_Model):
    """Metered entitlements are useful for many different use cases, from setting up usage based
    access to implementing complex credit systems.
    Access is determined based on feature usage using a balance calculation (the "usage allowance"
    provided by the issued grants is "burnt down" by the usage).

    :ivar type: Required.
    :vartype type: str or ~openmeter._generated.models.METERED
    :ivar is_soft_limit: Soft limit.
    :vartype is_soft_limit: bool
    :ivar is_unlimited: Deprecated, ignored by the backend. Please use isSoftLimit instead; this
     field will be removed in the future.
    :vartype is_unlimited: bool
    :ivar issue_after_reset: Initial grant amount.
    :vartype issue_after_reset: float
    :ivar issue_after_reset_priority: Issue grant after reset priority.
    :vartype issue_after_reset_priority: int
    :ivar preserve_overage_at_reset: Preserve overage at reset.
    :vartype preserve_overage_at_reset: bool
    :ivar created_at: Creation Time. Required.
    :vartype created_at: ~datetime.datetime
    :ivar updated_at: Last Update Time. Required.
    :vartype updated_at: ~datetime.datetime
    :ivar deleted_at: Deletion Time.
    :vartype deleted_at: ~datetime.datetime
    :ivar metadata: Additional metadata for the feature.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar active_from: The cadence start of the resource. Required.
    :vartype active_from: ~datetime.datetime
    :ivar active_to: The cadence end of the resource.
    :vartype active_to: ~datetime.datetime
    :ivar annotations: The annotations of the entitlement.
    :vartype annotations: ~openmeter._generated.models.Annotations
    :ivar id: Readonly unique ULID identifier. Required.
    :vartype id: str
    :ivar subject_key: The identifier key unique to the subject.
     NOTE: Subjects are being deprecated, please use the new customer APIs. Required.
    :vartype subject_key: str
    :ivar feature_key: The feature the subject is entitled to use. Required.
    :vartype feature_key: str
    :ivar feature_id: The feature the subject is entitled to use. Required.
    :vartype feature_id: str
    :ivar last_reset: The time the last reset happened. Required.
    :vartype last_reset: ~datetime.datetime
    :ivar current_usage_period: The current usage period. Required.
    :vartype current_usage_period: ~openmeter._generated.models.Period
    :ivar measure_usage_from: The time from which usage is measured. If not specified on creation,
     defaults to entitlement creation time. Required.
    :vartype measure_usage_from: ~datetime.datetime
    :ivar usage_period: THe usage period of the entitlement. Required.
    :vartype usage_period: ~openmeter._generated.models.RecurringPeriod
    """

    type: Literal[EntitlementType.METERED] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Required."""
    is_soft_limit: Optional[bool] = rest_field(
        name="isSoftLimit", visibility=["read", "create", "update", "delete", "query"]
    )
    """Soft limit."""
    is_unlimited: Optional[bool] = rest_field(
        name="isUnlimited", visibility=["read", "create", "update", "delete", "query"]
    )
    """Deprecated, ignored by the backend. Please use isSoftLimit instead; this field will be removed
     in the future."""
    issue_after_reset: Optional[float] = rest_field(
        name="issueAfterReset", visibility=["read", "create", "update", "delete", "query"]
    )
    """Initial grant amount."""
    issue_after_reset_priority: Optional[int] = rest_field(
        name="issueAfterResetPriority", visibility=["read", "create", "update", "delete", "query"]
    )
    """Issue grant after reset priority."""
    preserve_overage_at_reset: Optional[bool] = rest_field(
        name="preserveOverageAtReset", visibility=["read", "create", "update", "delete", "query"]
    )
    """Preserve overage at reset."""
    created_at: datetime.datetime = rest_field(name="createdAt", visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    updated_at: datetime.datetime = rest_field(name="updatedAt", visibility=["read"], format="rfc3339")
    """Last Update Time. Required."""
    deleted_at: Optional[datetime.datetime] = rest_field(name="deletedAt", visibility=["read"], format="rfc3339")
    """Deletion Time."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Additional metadata for the feature."""
    active_from: datetime.datetime = rest_field(
        name="activeFrom", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The cadence start of the resource. Required."""
    active_to: Optional[datetime.datetime] = rest_field(
        name="activeTo", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The cadence end of the resource."""
    annotations: Optional["_models.Annotations"] = rest_field(visibility=["read"])
    """The annotations of the entitlement."""
    id: str = rest_field(visibility=["read"])
    """Readonly unique ULID identifier. Required."""
    subject_key: str = rest_field(name="subjectKey", visibility=["read", "create", "update", "delete", "query"])
    """The identifier key unique to the subject.
     NOTE: Subjects are being deprecated, please use the new customer APIs. Required."""
    feature_key: str = rest_field(name="featureKey", visibility=["read", "create", "update", "delete", "query"])
    """The feature the subject is entitled to use. Required."""
    feature_id: str = rest_field(name="featureId", visibility=["read", "create", "update", "delete", "query"])
    """The feature the subject is entitled to use. Required."""
    last_reset: datetime.datetime = rest_field(name="lastReset", visibility=["read"], format="rfc3339")
    """The time the last reset happened. Required."""
    current_usage_period: "_models.Period" = rest_field(name="currentUsagePeriod", visibility=["read"])
    """The current usage period. Required."""
    measure_usage_from: datetime.datetime = rest_field(name="measureUsageFrom", visibility=["read"], format="rfc3339")
    """The time from which usage is measured. If not specified on creation, defaults to entitlement
     creation time. Required."""
    usage_period: "_models.RecurringPeriod" = rest_field(name="usagePeriod", visibility=["read"])
    """THe usage period of the entitlement. Required."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[EntitlementType.METERED],
        active_from: datetime.datetime,
        subject_key: str,
        feature_key: str,
        feature_id: str,
        is_soft_limit: Optional[bool] = None,
        is_unlimited: Optional[bool] = None,
        issue_after_reset: Optional[float] = None,
        issue_after_reset_priority: Optional[int] = None,
        preserve_overage_at_reset: Optional[bool] = None,
        metadata: Optional["_models.Metadata"] = None,
        active_to: Optional[datetime.datetime] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class EntitlementMeteredCreateInputs(_Model):
    """Create inpurs for metered entitlement.

    :ivar feature_key: The feature the subject is entitled to use.
     Either featureKey or featureId is required.
    :vartype feature_key: str
    :ivar feature_id: The feature the subject is entitled to use.
     Either featureKey or featureId is required.
    :vartype feature_id: str
    :ivar metadata: Additional metadata for the feature.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar type: Required.
    :vartype type: str or ~openmeter._generated.models.METERED
    :ivar is_soft_limit: Soft limit.
    :vartype is_soft_limit: bool
    :ivar is_unlimited: Deprecated, ignored by the backend. Please use isSoftLimit instead; this
     field will be removed in the future.
    :vartype is_unlimited: bool
    :ivar usage_period: The usage period associated with the entitlement. Required.
    :vartype usage_period: ~openmeter._generated.models.RecurringPeriodCreateInput
    :ivar measure_usage_from: Defines the time from which usage is measured. If not specified on
     creation, defaults to entitlement creation time. Is either a Union[str,
     "_models.MeasureUsageFromPreset"] type or a datetime.datetime type.
    :vartype measure_usage_from: str or ~openmeter.models.MeasureUsageFromPreset or
     ~datetime.datetime
    :ivar issue_after_reset: Initial grant amount.
    :vartype issue_after_reset: float
    :ivar issue_after_reset_priority: Issue grant after reset priority.
    :vartype issue_after_reset_priority: int
    :ivar preserve_overage_at_reset: Preserve overage at reset.
    :vartype preserve_overage_at_reset: bool
    """

    feature_key: Optional[str] = rest_field(
        name="featureKey", visibility=["read", "create", "update", "delete", "query"]
    )
    """The feature the subject is entitled to use.
     Either featureKey or featureId is required."""
    feature_id: Optional[str] = rest_field(name="featureId", visibility=["read", "create", "update", "delete", "query"])
    """The feature the subject is entitled to use.
     Either featureKey or featureId is required."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Additional metadata for the feature."""
    type: Literal[EntitlementType.METERED] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Required."""
    is_soft_limit: Optional[bool] = rest_field(
        name="isSoftLimit", visibility=["read", "create", "update", "delete", "query"]
    )
    """Soft limit."""
    is_unlimited: Optional[bool] = rest_field(
        name="isUnlimited", visibility=["read", "create", "update", "delete", "query"]
    )
    """Deprecated, ignored by the backend. Please use isSoftLimit instead; this field will be removed
     in the future."""
    usage_period: "_models.RecurringPeriodCreateInput" = rest_field(
        name="usagePeriod", visibility=["read", "create", "update", "delete", "query"]
    )
    """The usage period associated with the entitlement. Required."""
    measure_usage_from: Optional["_types.MeasureUsageFrom"] = rest_field(
        name="measureUsageFrom", visibility=["read", "create", "update", "delete", "query"]
    )
    """Defines the time from which usage is measured. If not specified on creation, defaults to
     entitlement creation time. Is either a Union[str, \"_models.MeasureUsageFromPreset\"] type or a
     datetime.datetime type."""
    issue_after_reset: Optional[float] = rest_field(
        name="issueAfterReset", visibility=["read", "create", "update", "delete", "query"]
    )
    """Initial grant amount."""
    issue_after_reset_priority: Optional[int] = rest_field(
        name="issueAfterResetPriority", visibility=["read", "create", "update", "delete", "query"]
    )
    """Issue grant after reset priority."""
    preserve_overage_at_reset: Optional[bool] = rest_field(
        name="preserveOverageAtReset", visibility=["read", "create", "update", "delete", "query"]
    )
    """Preserve overage at reset."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[EntitlementType.METERED],
        usage_period: "_models.RecurringPeriodCreateInput",
        feature_key: Optional[str] = None,
        feature_id: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
        is_soft_limit: Optional[bool] = None,
        is_unlimited: Optional[bool] = None,
        measure_usage_from: Optional["_types.MeasureUsageFrom"] = None,
        issue_after_reset: Optional[float] = None,
        issue_after_reset_priority: Optional[int] = None,
        preserve_overage_at_reset: Optional[bool] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class EntitlementMeteredV2(_Model):
    """Metered entitlements are useful for many different use cases, from setting up usage based
    access to implementing complex credit systems.
    Access is determined based on feature usage using a balance calculation (the "usage allowance"
    provided by the issued grants is "burnt down" by the usage).

    :ivar type: Required.
    :vartype type: str or ~openmeter._generated.models.METERED
    :ivar is_soft_limit: Soft limit.
    :vartype is_soft_limit: bool
    :ivar preserve_overage_at_reset: Preserve overage at reset.
    :vartype preserve_overage_at_reset: bool
    :ivar issue_after_reset: Initial grant amount.
    :vartype issue_after_reset: float
    :ivar issue_after_reset_priority: Issue grant after reset priority.
    :vartype issue_after_reset_priority: int
    :ivar issue: Issue after reset.
    :vartype issue: ~openmeter._generated.models.IssueAfterReset
    :ivar created_at: Creation Time. Required.
    :vartype created_at: ~datetime.datetime
    :ivar updated_at: Last Update Time. Required.
    :vartype updated_at: ~datetime.datetime
    :ivar deleted_at: Deletion Time.
    :vartype deleted_at: ~datetime.datetime
    :ivar metadata: Additional metadata for the feature.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar active_from: The cadence start of the resource. Required.
    :vartype active_from: ~datetime.datetime
    :ivar active_to: The cadence end of the resource.
    :vartype active_to: ~datetime.datetime
    :ivar annotations: The annotations of the entitlement.
    :vartype annotations: ~openmeter._generated.models.Annotations
    :ivar id: Readonly unique ULID identifier. Required.
    :vartype id: str
    :ivar feature_key: The feature the subject is entitled to use. Required.
    :vartype feature_key: str
    :ivar feature_id: The feature the subject is entitled to use. Required.
    :vartype feature_id: str
    :ivar last_reset: The time the last reset happened. Required.
    :vartype last_reset: ~datetime.datetime
    :ivar current_usage_period: The current usage period. Required.
    :vartype current_usage_period: ~openmeter._generated.models.Period
    :ivar measure_usage_from: The time from which usage is measured. If not specified on creation,
     defaults to entitlement creation time. Required.
    :vartype measure_usage_from: ~datetime.datetime
    :ivar usage_period: THe usage period of the entitlement. Required.
    :vartype usage_period: ~openmeter._generated.models.RecurringPeriod
    :ivar customer_key: The identifier key unique to the customer.
    :vartype customer_key: str
    :ivar customer_id: The identifier unique to the customer. Required.
    :vartype customer_id: str
    """

    type: Literal[EntitlementType.METERED] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Required."""
    is_soft_limit: Optional[bool] = rest_field(
        name="isSoftLimit", visibility=["read", "create", "update", "delete", "query"]
    )
    """Soft limit."""
    preserve_overage_at_reset: Optional[bool] = rest_field(
        name="preserveOverageAtReset", visibility=["read", "create", "update", "delete", "query"]
    )
    """Preserve overage at reset."""
    issue_after_reset: Optional[float] = rest_field(
        name="issueAfterReset", visibility=["read", "create", "update", "delete", "query"]
    )
    """Initial grant amount."""
    issue_after_reset_priority: Optional[int] = rest_field(
        name="issueAfterResetPriority", visibility=["read", "create", "update", "delete", "query"]
    )
    """Issue grant after reset priority."""
    issue: Optional["_models.IssueAfterReset"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Issue after reset."""
    created_at: datetime.datetime = rest_field(name="createdAt", visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    updated_at: datetime.datetime = rest_field(name="updatedAt", visibility=["read"], format="rfc3339")
    """Last Update Time. Required."""
    deleted_at: Optional[datetime.datetime] = rest_field(name="deletedAt", visibility=["read"], format="rfc3339")
    """Deletion Time."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Additional metadata for the feature."""
    active_from: datetime.datetime = rest_field(
        name="activeFrom", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The cadence start of the resource. Required."""
    active_to: Optional[datetime.datetime] = rest_field(
        name="activeTo", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The cadence end of the resource."""
    annotations: Optional["_models.Annotations"] = rest_field(visibility=["read"])
    """The annotations of the entitlement."""
    id: str = rest_field(visibility=["read"])
    """Readonly unique ULID identifier. Required."""
    feature_key: str = rest_field(name="featureKey", visibility=["read", "create", "update", "delete", "query"])
    """The feature the subject is entitled to use. Required."""
    feature_id: str = rest_field(name="featureId", visibility=["read", "create", "update", "delete", "query"])
    """The feature the subject is entitled to use. Required."""
    last_reset: datetime.datetime = rest_field(name="lastReset", visibility=["read"], format="rfc3339")
    """The time the last reset happened. Required."""
    current_usage_period: "_models.Period" = rest_field(name="currentUsagePeriod", visibility=["read"])
    """The current usage period. Required."""
    measure_usage_from: datetime.datetime = rest_field(name="measureUsageFrom", visibility=["read"], format="rfc3339")
    """The time from which usage is measured. If not specified on creation, defaults to entitlement
     creation time. Required."""
    usage_period: "_models.RecurringPeriod" = rest_field(name="usagePeriod", visibility=["read"])
    """THe usage period of the entitlement. Required."""
    customer_key: Optional[str] = rest_field(
        name="customerKey", visibility=["read", "create", "update", "delete", "query"]
    )
    """The identifier key unique to the customer."""
    customer_id: str = rest_field(name="customerId", visibility=["read", "create", "update", "delete", "query"])
    """The identifier unique to the customer. Required."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[EntitlementType.METERED],
        active_from: datetime.datetime,
        feature_key: str,
        feature_id: str,
        customer_id: str,
        is_soft_limit: Optional[bool] = None,
        preserve_overage_at_reset: Optional[bool] = None,
        issue_after_reset: Optional[float] = None,
        issue_after_reset_priority: Optional[int] = None,
        issue: Optional["_models.IssueAfterReset"] = None,
        metadata: Optional["_models.Metadata"] = None,
        active_to: Optional[datetime.datetime] = None,
        customer_key: Optional[str] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class EntitlementMeteredV2CreateInputs(_Model):
    """Create inputs for metered entitlement.

    :ivar feature_key: The feature the subject is entitled to use.
     Either featureKey or featureId is required.
    :vartype feature_key: str
    :ivar feature_id: The feature the subject is entitled to use.
     Either featureKey or featureId is required.
    :vartype feature_id: str
    :ivar metadata: Additional metadata for the feature.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar type: Required.
    :vartype type: str or ~openmeter._generated.models.METERED
    :ivar is_soft_limit: Soft limit.
    :vartype is_soft_limit: bool
    :ivar usage_period: The usage period associated with the entitlement. Required.
    :vartype usage_period: ~openmeter._generated.models.RecurringPeriodCreateInput
    :ivar measure_usage_from: Defines the time from which usage is measured. If not specified on
     creation, defaults to entitlement creation time. Is either a Union[str,
     "_models.MeasureUsageFromPreset"] type or a datetime.datetime type.
    :vartype measure_usage_from: str or ~openmeter.models.MeasureUsageFromPreset or
     ~datetime.datetime
    :ivar preserve_overage_at_reset: Preserve overage at reset.
    :vartype preserve_overage_at_reset: bool
    :ivar issue_after_reset: Initial grant amount.
    :vartype issue_after_reset: float
    :ivar issue_after_reset_priority: Issue grant after reset priority.
    :vartype issue_after_reset_priority: int
    :ivar issue: Issue after reset.
    :vartype issue: ~openmeter._generated.models.IssueAfterReset
    :ivar grants: Grants.
    :vartype grants: list[~openmeter._generated.models.EntitlementGrantCreateInputV2]
    """

    feature_key: Optional[str] = rest_field(
        name="featureKey", visibility=["read", "create", "update", "delete", "query"]
    )
    """The feature the subject is entitled to use.
     Either featureKey or featureId is required."""
    feature_id: Optional[str] = rest_field(name="featureId", visibility=["read", "create", "update", "delete", "query"])
    """The feature the subject is entitled to use.
     Either featureKey or featureId is required."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Additional metadata for the feature."""
    type: Literal[EntitlementType.METERED] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Required."""
    is_soft_limit: Optional[bool] = rest_field(
        name="isSoftLimit", visibility=["read", "create", "update", "delete", "query"]
    )
    """Soft limit."""
    usage_period: "_models.RecurringPeriodCreateInput" = rest_field(
        name="usagePeriod", visibility=["read", "create", "update", "delete", "query"]
    )
    """The usage period associated with the entitlement. Required."""
    measure_usage_from: Optional["_types.MeasureUsageFrom"] = rest_field(
        name="measureUsageFrom", visibility=["read", "create", "update", "delete", "query"]
    )
    """Defines the time from which usage is measured. If not specified on creation, defaults to
     entitlement creation time. Is either a Union[str, \"_models.MeasureUsageFromPreset\"] type or a
     datetime.datetime type."""
    preserve_overage_at_reset: Optional[bool] = rest_field(
        name="preserveOverageAtReset", visibility=["read", "create", "update", "delete", "query"]
    )
    """Preserve overage at reset."""
    issue_after_reset: Optional[float] = rest_field(
        name="issueAfterReset", visibility=["read", "create", "update", "delete", "query"]
    )
    """Initial grant amount."""
    issue_after_reset_priority: Optional[int] = rest_field(
        name="issueAfterResetPriority", visibility=["read", "create", "update", "delete", "query"]
    )
    """Issue grant after reset priority."""
    issue: Optional["_models.IssueAfterReset"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Issue after reset."""
    grants: Optional[list["_models.EntitlementGrantCreateInputV2"]] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """Grants."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[EntitlementType.METERED],
        usage_period: "_models.RecurringPeriodCreateInput",
        feature_key: Optional[str] = None,
        feature_id: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
        is_soft_limit: Optional[bool] = None,
        measure_usage_from: Optional["_types.MeasureUsageFrom"] = None,
        preserve_overage_at_reset: Optional[bool] = None,
        issue_after_reset: Optional[float] = None,
        issue_after_reset_priority: Optional[int] = None,
        issue: Optional["_models.IssueAfterReset"] = None,
        grants: Optional[list["_models.EntitlementGrantCreateInputV2"]] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class EntitlementPaginatedResponse(_Model):
    """Paginated response.

    :ivar total_count: The total number of items. Required.
    :vartype total_count: int
    :ivar page: The page index. Required.
    :vartype page: int
    :ivar page_size: The maximum number of items per page. Required.
    :vartype page_size: int
    :ivar items_property: The items in the current page. Required.
    :vartype items_property: list[~openmeter._generated.models.EntitlementMetered or
     ~openmeter._generated.models.EntitlementStatic or
     ~openmeter._generated.models.EntitlementBoolean]
    """

    total_count: int = rest_field(name="totalCount", visibility=["read", "create", "update", "delete", "query"])
    """The total number of items. Required."""
    page: int = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The page index. Required."""
    page_size: int = rest_field(name="pageSize", visibility=["read", "create", "update", "delete", "query"])
    """The maximum number of items per page. Required."""
    items_property: list["_types.Entitlement"] = rest_field(
        name="items", visibility=["read", "create", "update", "delete", "query"]
    )
    """The items in the current page. Required."""

    @overload
    def __init__(
        self,
        *,
        total_count: int,
        page: int,
        page_size: int,
        items_property: list["_types.Entitlement"],
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class EntitlementStatic(_Model):
    """A static entitlement.

    :ivar type: Required.
    :vartype type: str or ~openmeter._generated.models.STATIC
    :ivar config: The JSON parsable config of the entitlement. This value is also returned when
     checking entitlement access and it is useful for configuring fine-grained access settings to
     the feature, implemented in your own system. Has to be an object. Required.
    :vartype config: str
    :ivar created_at: Creation Time. Required.
    :vartype created_at: ~datetime.datetime
    :ivar updated_at: Last Update Time. Required.
    :vartype updated_at: ~datetime.datetime
    :ivar deleted_at: Deletion Time.
    :vartype deleted_at: ~datetime.datetime
    :ivar metadata: Additional metadata for the feature.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar active_from: The cadence start of the resource. Required.
    :vartype active_from: ~datetime.datetime
    :ivar active_to: The cadence end of the resource.
    :vartype active_to: ~datetime.datetime
    :ivar annotations: The annotations of the entitlement.
    :vartype annotations: ~openmeter._generated.models.Annotations
    :ivar id: Readonly unique ULID identifier. Required.
    :vartype id: str
    :ivar subject_key: The identifier key unique to the subject.
     NOTE: Subjects are being deprecated, please use the new customer APIs. Required.
    :vartype subject_key: str
    :ivar feature_key: The feature the subject is entitled to use. Required.
    :vartype feature_key: str
    :ivar feature_id: The feature the subject is entitled to use. Required.
    :vartype feature_id: str
    :ivar current_usage_period: The current usage period.
    :vartype current_usage_period: ~openmeter._generated.models.Period
    :ivar usage_period: The defined usage period of the entitlement.
    :vartype usage_period: ~openmeter._generated.models.RecurringPeriod
    """

    type: Literal[EntitlementType.STATIC] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Required."""
    config: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The JSON parsable config of the entitlement. This value is also returned when checking
     entitlement access and it is useful for configuring fine-grained access settings to the
     feature, implemented in your own system. Has to be an object. Required."""
    created_at: datetime.datetime = rest_field(name="createdAt", visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    updated_at: datetime.datetime = rest_field(name="updatedAt", visibility=["read"], format="rfc3339")
    """Last Update Time. Required."""
    deleted_at: Optional[datetime.datetime] = rest_field(name="deletedAt", visibility=["read"], format="rfc3339")
    """Deletion Time."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Additional metadata for the feature."""
    active_from: datetime.datetime = rest_field(
        name="activeFrom", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The cadence start of the resource. Required."""
    active_to: Optional[datetime.datetime] = rest_field(
        name="activeTo", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The cadence end of the resource."""
    annotations: Optional["_models.Annotations"] = rest_field(visibility=["read"])
    """The annotations of the entitlement."""
    id: str = rest_field(visibility=["read"])
    """Readonly unique ULID identifier. Required."""
    subject_key: str = rest_field(name="subjectKey", visibility=["read", "create", "update", "delete", "query"])
    """The identifier key unique to the subject.
     NOTE: Subjects are being deprecated, please use the new customer APIs. Required."""
    feature_key: str = rest_field(name="featureKey", visibility=["read", "create", "update", "delete", "query"])
    """The feature the subject is entitled to use. Required."""
    feature_id: str = rest_field(name="featureId", visibility=["read", "create", "update", "delete", "query"])
    """The feature the subject is entitled to use. Required."""
    current_usage_period: Optional["_models.Period"] = rest_field(
        name="currentUsagePeriod", visibility=["read", "create", "update", "delete", "query"]
    )
    """The current usage period."""
    usage_period: Optional["_models.RecurringPeriod"] = rest_field(
        name="usagePeriod", visibility=["read", "create", "update", "delete", "query"]
    )
    """The defined usage period of the entitlement."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[EntitlementType.STATIC],
        config: str,
        active_from: datetime.datetime,
        subject_key: str,
        feature_key: str,
        feature_id: str,
        metadata: Optional["_models.Metadata"] = None,
        active_to: Optional[datetime.datetime] = None,
        current_usage_period: Optional["_models.Period"] = None,
        usage_period: Optional["_models.RecurringPeriod"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class EntitlementStaticCreateInputs(_Model):
    """Create inputs for static entitlement.

    :ivar feature_key: The feature the subject is entitled to use.
     Either featureKey or featureId is required.
    :vartype feature_key: str
    :ivar feature_id: The feature the subject is entitled to use.
     Either featureKey or featureId is required.
    :vartype feature_id: str
    :ivar metadata: Additional metadata for the feature.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar usage_period: The usage period associated with the entitlement.
    :vartype usage_period: ~openmeter._generated.models.RecurringPeriodCreateInput
    :ivar type: Required.
    :vartype type: str or ~openmeter._generated.models.STATIC
    :ivar config: The JSON parsable config of the entitlement. This value is also returned when
     checking entitlement access and it is useful for configuring fine-grained access settings to
     the feature, implemented in your own system. Has to be an object. Required.
    :vartype config: str
    """

    feature_key: Optional[str] = rest_field(
        name="featureKey", visibility=["read", "create", "update", "delete", "query"]
    )
    """The feature the subject is entitled to use.
     Either featureKey or featureId is required."""
    feature_id: Optional[str] = rest_field(name="featureId", visibility=["read", "create", "update", "delete", "query"])
    """The feature the subject is entitled to use.
     Either featureKey or featureId is required."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Additional metadata for the feature."""
    usage_period: Optional["_models.RecurringPeriodCreateInput"] = rest_field(
        name="usagePeriod", visibility=["read", "create", "update", "delete", "query"]
    )
    """The usage period associated with the entitlement."""
    type: Literal[EntitlementType.STATIC] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Required."""
    config: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The JSON parsable config of the entitlement. This value is also returned when checking
     entitlement access and it is useful for configuring fine-grained access settings to the
     feature, implemented in your own system. Has to be an object. Required."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[EntitlementType.STATIC],
        config: str,
        feature_key: Optional[str] = None,
        feature_id: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
        usage_period: Optional["_models.RecurringPeriodCreateInput"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class EntitlementStaticV2(_Model):
    """A static entitlement.

    :ivar type: Required.
    :vartype type: str or ~openmeter._generated.models.STATIC
    :ivar config: The JSON parsable config of the entitlement. This value is also returned when
     checking entitlement access and it is useful for configuring fine-grained access settings to
     the feature, implemented in your own system. Has to be an object. Required.
    :vartype config: str
    :ivar created_at: Creation Time. Required.
    :vartype created_at: ~datetime.datetime
    :ivar updated_at: Last Update Time. Required.
    :vartype updated_at: ~datetime.datetime
    :ivar deleted_at: Deletion Time.
    :vartype deleted_at: ~datetime.datetime
    :ivar metadata: Additional metadata for the feature.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar active_from: The cadence start of the resource. Required.
    :vartype active_from: ~datetime.datetime
    :ivar active_to: The cadence end of the resource.
    :vartype active_to: ~datetime.datetime
    :ivar annotations: The annotations of the entitlement.
    :vartype annotations: ~openmeter._generated.models.Annotations
    :ivar id: Readonly unique ULID identifier. Required.
    :vartype id: str
    :ivar feature_key: The feature the subject is entitled to use. Required.
    :vartype feature_key: str
    :ivar feature_id: The feature the subject is entitled to use. Required.
    :vartype feature_id: str
    :ivar current_usage_period: The current usage period.
    :vartype current_usage_period: ~openmeter._generated.models.Period
    :ivar usage_period: The defined usage period of the entitlement.
    :vartype usage_period: ~openmeter._generated.models.RecurringPeriod
    :ivar customer_key: The identifier key unique to the customer.
    :vartype customer_key: str
    :ivar customer_id: The identifier unique to the customer. Required.
    :vartype customer_id: str
    """

    type: Literal[EntitlementType.STATIC] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Required."""
    config: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The JSON parsable config of the entitlement. This value is also returned when checking
     entitlement access and it is useful for configuring fine-grained access settings to the
     feature, implemented in your own system. Has to be an object. Required."""
    created_at: datetime.datetime = rest_field(name="createdAt", visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    updated_at: datetime.datetime = rest_field(name="updatedAt", visibility=["read"], format="rfc3339")
    """Last Update Time. Required."""
    deleted_at: Optional[datetime.datetime] = rest_field(name="deletedAt", visibility=["read"], format="rfc3339")
    """Deletion Time."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Additional metadata for the feature."""
    active_from: datetime.datetime = rest_field(
        name="activeFrom", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The cadence start of the resource. Required."""
    active_to: Optional[datetime.datetime] = rest_field(
        name="activeTo", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The cadence end of the resource."""
    annotations: Optional["_models.Annotations"] = rest_field(visibility=["read"])
    """The annotations of the entitlement."""
    id: str = rest_field(visibility=["read"])
    """Readonly unique ULID identifier. Required."""
    feature_key: str = rest_field(name="featureKey", visibility=["read", "create", "update", "delete", "query"])
    """The feature the subject is entitled to use. Required."""
    feature_id: str = rest_field(name="featureId", visibility=["read", "create", "update", "delete", "query"])
    """The feature the subject is entitled to use. Required."""
    current_usage_period: Optional["_models.Period"] = rest_field(
        name="currentUsagePeriod", visibility=["read", "create", "update", "delete", "query"]
    )
    """The current usage period."""
    usage_period: Optional["_models.RecurringPeriod"] = rest_field(
        name="usagePeriod", visibility=["read", "create", "update", "delete", "query"]
    )
    """The defined usage period of the entitlement."""
    customer_key: Optional[str] = rest_field(
        name="customerKey", visibility=["read", "create", "update", "delete", "query"]
    )
    """The identifier key unique to the customer."""
    customer_id: str = rest_field(name="customerId", visibility=["read", "create", "update", "delete", "query"])
    """The identifier unique to the customer. Required."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[EntitlementType.STATIC],
        config: str,
        active_from: datetime.datetime,
        feature_key: str,
        feature_id: str,
        customer_id: str,
        metadata: Optional["_models.Metadata"] = None,
        active_to: Optional[datetime.datetime] = None,
        current_usage_period: Optional["_models.Period"] = None,
        usage_period: Optional["_models.RecurringPeriod"] = None,
        customer_key: Optional[str] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class EntitlementV2PaginatedResponse(_Model):
    """Paginated response.

    :ivar total_count: The total number of items. Required.
    :vartype total_count: int
    :ivar page: The page index. Required.
    :vartype page: int
    :ivar page_size: The maximum number of items per page. Required.
    :vartype page_size: int
    :ivar items_property: The items in the current page. Required.
    :vartype items_property: list[~openmeter._generated.models.EntitlementMeteredV2 or
     ~openmeter._generated.models.EntitlementStaticV2 or
     ~openmeter._generated.models.EntitlementBooleanV2]
    """

    total_count: int = rest_field(name="totalCount", visibility=["read", "create", "update", "delete", "query"])
    """The total number of items. Required."""
    page: int = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The page index. Required."""
    page_size: int = rest_field(name="pageSize", visibility=["read", "create", "update", "delete", "query"])
    """The maximum number of items per page. Required."""
    items_property: list["_types.EntitlementV2"] = rest_field(
        name="items", visibility=["read", "create", "update", "delete", "query"]
    )
    """The items in the current page. Required."""

    @overload
    def __init__(
        self,
        *,
        total_count: int,
        page: int,
        page_size: int,
        items_property: list["_types.EntitlementV2"],
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class EntitlementValue(_Model):
    """Entitlements are the core of OpenMeter access management. They define access to features for
    subjects. Entitlements can be metered, boolean, or static.

    :ivar has_access: Whether the subject has access to the feature. Shared accross all entitlement
     types. Required.
    :vartype has_access: bool
    :ivar balance: Only available for metered entitlements. Metered entitlements are built around a
     balance calculation where feature usage is deducted from the issued grants. Balance represents
     the remaining balance of the entitlement, it's value never turns negative.
    :vartype balance: float
    :ivar usage: Only available for metered entitlements. Returns the total feature usage in the
     current period.
    :vartype usage: float
    :ivar overage: Only available for metered entitlements. Overage represents the usage that
     wasn't covered by grants, e.g. if the subject had a total feature usage of 100 in the period
     but they were only granted 80, there would be 20 overage.
    :vartype overage: float
    :ivar total_available_grant_amount: The summed value of all grant amounts that are active at
     the time of the query.
    :vartype total_available_grant_amount: float
    :ivar config: Only available for static entitlements. The JSON parsable config of the
     entitlement.
    :vartype config: str
    """

    has_access: bool = rest_field(name="hasAccess", visibility=["read"])
    """Whether the subject has access to the feature. Shared accross all entitlement types. Required."""
    balance: Optional[float] = rest_field(visibility=["read"])
    """Only available for metered entitlements. Metered entitlements are built around a balance
     calculation where feature usage is deducted from the issued grants. Balance represents the
     remaining balance of the entitlement, it's value never turns negative."""
    usage: Optional[float] = rest_field(visibility=["read"])
    """Only available for metered entitlements. Returns the total feature usage in the current period."""
    overage: Optional[float] = rest_field(visibility=["read"])
    """Only available for metered entitlements. Overage represents the usage that wasn't covered by
     grants, e.g. if the subject had a total feature usage of 100 in the period but they were only
     granted 80, there would be 20 overage."""
    total_available_grant_amount: Optional[float] = rest_field(name="totalAvailableGrantAmount", visibility=["read"])
    """The summed value of all grant amounts that are active at the time of the query."""
    config: Optional[str] = rest_field(visibility=["read"])
    """Only available for static entitlements. The JSON parsable config of the entitlement."""


class ErrorExtension(_Model):
    """Generic ErrorExtension as part of HTTPProblem.Extensions.[StatusCode].

    :ivar field: The path to the field. Required.
    :vartype field: str
    :ivar code: The machine readable description of the error. Required.
    :vartype code: str
    :ivar message: The human readable description of the error. Required.
    :vartype message: str
    """

    field: str = rest_field(visibility=["read"])
    """The path to the field. Required."""
    code: str = rest_field(visibility=["read"])
    """The machine readable description of the error. Required."""
    message: str = rest_field(visibility=["read"])
    """The human readable description of the error. Required."""


class Event(_Model):
    """CloudEvents Specification JSON Schema

    Optional properties are nullable according to the CloudEvents specification:
    OPTIONAL not omitted attributes MAY be represented as a null JSON value.

    :ivar id: Identifies the event. Required.
    :vartype id: str
    :ivar source: Identifies the context in which an event happened. Required.
    :vartype source: str
    :ivar specversion: The version of the CloudEvents specification which the event uses. Required.
    :vartype specversion: str
    :ivar type: Contains a value describing the type of event related to the originating
     occurrence. Required.
    :vartype type: str
    :ivar datacontenttype: Content type of the CloudEvents data value. Only the value
     "application/json" is allowed over HTTP. Default value is "application/json".
    :vartype datacontenttype: str
    :ivar dataschema: Identifies the schema that data adheres to.
    :vartype dataschema: str
    :ivar subject: Describes the subject of the event in the context of the event producer
     (identified by source). Required.
    :vartype subject: str
    :ivar time: Timestamp of when the occurrence happened. Must adhere to RFC 3339.
    :vartype time: ~datetime.datetime
    :ivar data: The event payload.
     Optional, if present it must be a JSON object.
    :vartype data: dict[str, any]
    """

    id: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Identifies the event. Required."""
    source: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Identifies the context in which an event happened. Required."""
    specversion: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The version of the CloudEvents specification which the event uses. Required."""
    type: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Contains a value describing the type of event related to the originating occurrence. Required."""
    datacontenttype: Optional[Literal["application/json"]] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """Content type of the CloudEvents data value. Only the value \"application/json\" is allowed over
     HTTP. Default value is \"application/json\"."""
    dataschema: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Identifies the schema that data adheres to."""
    subject: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Describes the subject of the event in the context of the event producer (identified by source).
     Required."""
    time: Optional[datetime.datetime] = rest_field(
        visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """Timestamp of when the occurrence happened. Must adhere to RFC 3339."""
    data: Optional[dict[str, Any]] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The event payload.
     Optional, if present it must be a JSON object."""

    @overload
    def __init__(
        self,
        *,
        id: str,  # pylint: disable=redefined-builtin
        source: str,
        specversion: str,
        type: str,
        subject: str,
        datacontenttype: Optional[Literal["application/json"]] = None,
        dataschema: Optional[str] = None,
        time: Optional[datetime.datetime] = None,
        data: Optional[dict[str, Any]] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class EventDeliveryAttemptResponse(_Model):
    """The response of the event delivery attempt.

    :ivar status_code: Status Code.
    :vartype status_code: int
    :ivar body: Response Body. Required.
    :vartype body: str
    :ivar duration_ms: Response Duration. Required.
    :vartype duration_ms: int
    :ivar url: URL.
    :vartype url: str
    """

    status_code: Optional[int] = rest_field(name="statusCode", visibility=["read"])
    """Status Code."""
    body: str = rest_field(visibility=["read"])
    """Response Body. Required."""
    duration_ms: int = rest_field(name="durationMs", visibility=["read"])
    """Response Duration. Required."""
    url: Optional[str] = rest_field(visibility=["read"])
    """URL."""


class ExpirationPeriod(_Model):
    """The grant expiration definition.

    :ivar duration: The unit of time for the expiration period. Required. Known values are: "HOUR",
     "DAY", "WEEK", "MONTH", and "YEAR".
    :vartype duration: str or ~openmeter.models.ExpirationDuration
    :ivar count: The number of time units in the expiration period. Required.
    :vartype count: int
    """

    duration: Union[str, "_models.ExpirationDuration"] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """The unit of time for the expiration period. Required. Known values are: \"HOUR\", \"DAY\",
     \"WEEK\", \"MONTH\", and \"YEAR\"."""
    count: int = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The number of time units in the expiration period. Required."""

    @overload
    def __init__(
        self,
        *,
        duration: Union[str, "_models.ExpirationDuration"],
        count: int,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class Feature(_Model):
    """Represents a feature that can be enabled or disabled for a plan.
    Used both for product catalog and entitlements.

    :ivar created_at: Creation Time. Required.
    :vartype created_at: ~datetime.datetime
    :ivar updated_at: Last Update Time. Required.
    :vartype updated_at: ~datetime.datetime
    :ivar deleted_at: Deletion Time.
    :vartype deleted_at: ~datetime.datetime
    :ivar archived_at: Archival Time.
    :vartype archived_at: ~datetime.datetime
    :ivar key: The unique key of the feature. Required.
    :vartype key: str
    :ivar name: The human-readable name of the feature. Required.
    :vartype name: str
    :ivar metadata: Optional metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar meter_slug: Meter slug.
    :vartype meter_slug: str
    :ivar meter_group_by_filters: Meter group by filters.
    :vartype meter_group_by_filters: dict[str, str]
    :ivar advanced_meter_group_by_filters: Advanced meter group by filters.
    :vartype advanced_meter_group_by_filters: dict[str, ~openmeter._generated.models.FilterString]
    :ivar id: Readonly unique ULID identifier. Required.
    :vartype id: str
    """

    created_at: datetime.datetime = rest_field(name="createdAt", visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    updated_at: datetime.datetime = rest_field(name="updatedAt", visibility=["read"], format="rfc3339")
    """Last Update Time. Required."""
    deleted_at: Optional[datetime.datetime] = rest_field(name="deletedAt", visibility=["read"], format="rfc3339")
    """Deletion Time."""
    archived_at: Optional[datetime.datetime] = rest_field(name="archivedAt", visibility=["read"], format="rfc3339")
    """Archival Time."""
    key: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The unique key of the feature. Required."""
    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The human-readable name of the feature. Required."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Optional metadata."""
    meter_slug: Optional[str] = rest_field(name="meterSlug", visibility=["read", "create", "update", "delete", "query"])
    """Meter slug."""
    meter_group_by_filters: Optional[dict[str, str]] = rest_field(
        name="meterGroupByFilters", visibility=["read", "create", "update", "delete", "query"]
    )
    """Meter group by filters."""
    advanced_meter_group_by_filters: Optional[dict[str, "_models.FilterString"]] = rest_field(
        name="advancedMeterGroupByFilters", visibility=["read", "create", "update", "delete", "query"]
    )
    """Advanced meter group by filters."""
    id: str = rest_field(visibility=["read"])
    """Readonly unique ULID identifier. Required."""

    @overload
    def __init__(
        self,
        *,
        key: str,
        name: str,
        metadata: Optional["_models.Metadata"] = None,
        meter_slug: Optional[str] = None,
        meter_group_by_filters: Optional[dict[str, str]] = None,
        advanced_meter_group_by_filters: Optional[dict[str, "_models.FilterString"]] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class FeatureCreateInputs(_Model):
    """Represents a feature that can be enabled or disabled for a plan.
    Used both for product catalog and entitlements.

    :ivar key: The unique key of the feature. Required.
    :vartype key: str
    :ivar name: The human-readable name of the feature. Required.
    :vartype name: str
    :ivar metadata: Optional metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar meter_slug: Meter slug.
    :vartype meter_slug: str
    :ivar meter_group_by_filters: Meter group by filters.
    :vartype meter_group_by_filters: dict[str, str]
    :ivar advanced_meter_group_by_filters: Advanced meter group by filters.
    :vartype advanced_meter_group_by_filters: dict[str, ~openmeter._generated.models.FilterString]
    """

    key: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The unique key of the feature. Required."""
    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The human-readable name of the feature. Required."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Optional metadata."""
    meter_slug: Optional[str] = rest_field(name="meterSlug", visibility=["read", "create", "update", "delete", "query"])
    """Meter slug."""
    meter_group_by_filters: Optional[dict[str, str]] = rest_field(
        name="meterGroupByFilters", visibility=["read", "create", "update", "delete", "query"]
    )
    """Meter group by filters."""
    advanced_meter_group_by_filters: Optional[dict[str, "_models.FilterString"]] = rest_field(
        name="advancedMeterGroupByFilters", visibility=["read", "create", "update", "delete", "query"]
    )
    """Advanced meter group by filters."""

    @overload
    def __init__(
        self,
        *,
        key: str,
        name: str,
        metadata: Optional["_models.Metadata"] = None,
        meter_slug: Optional[str] = None,
        meter_group_by_filters: Optional[dict[str, str]] = None,
        advanced_meter_group_by_filters: Optional[dict[str, "_models.FilterString"]] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class FeatureMeta(_Model):
    """Limited representation of a feature resource which includes only its unique identifiers (id,
    key).

    :ivar id: Feature Unique Identifier. Required.
    :vartype id: str
    :ivar key: Feature Key. Required.
    :vartype key: str
    """

    id: str = rest_field(visibility=["read", "create", "update"])
    """Feature Unique Identifier. Required."""
    key: str = rest_field(visibility=["read", "create", "update"])
    """Feature Key. Required."""

    @overload
    def __init__(
        self,
        *,
        id: str,  # pylint: disable=redefined-builtin
        key: str,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class FeaturePaginatedResponse(_Model):
    """Paginated response.

    :ivar total_count: The total number of items. Required.
    :vartype total_count: int
    :ivar page: The page index. Required.
    :vartype page: int
    :ivar page_size: The maximum number of items per page. Required.
    :vartype page_size: int
    :ivar items_property: The items in the current page. Required.
    :vartype items_property: list[~openmeter._generated.models.Feature]
    """

    total_count: int = rest_field(name="totalCount", visibility=["read", "create", "update", "delete", "query"])
    """The total number of items. Required."""
    page: int = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The page index. Required."""
    page_size: int = rest_field(name="pageSize", visibility=["read", "create", "update", "delete", "query"])
    """The maximum number of items per page. Required."""
    items_property: list["_models.Feature"] = rest_field(
        name="items", visibility=["read", "create", "update", "delete", "query"]
    )
    """The items in the current page. Required."""

    @overload
    def __init__(
        self,
        *,
        total_count: int,
        page: int,
        page_size: int,
        items_property: list["_models.Feature"],
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class FilterIDExact(_Model):
    """A filter for a ID (ULID) field allowing only equality or inclusion.

    :ivar in_property: The field must be in the provided list of values.
    :vartype in_property: list[str]
    """

    in_property: Optional[list[str]] = rest_field(
        name="$in", visibility=["read", "create", "update", "delete", "query"]
    )
    """The field must be in the provided list of values."""

    @overload
    def __init__(
        self,
        *,
        in_property: Optional[list[str]] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class FilterString(_Model):
    """A filter for a string field.

    :ivar eq: The field must be equal to the provided value.
    :vartype eq: str
    :ivar ne: The field must not be equal to the provided value.
    :vartype ne: str
    :ivar in_property: The field must be in the provided list of values.
    :vartype in_property: list[str]
    :ivar nin: The field must not be in the provided list of values.
    :vartype nin: list[str]
    :ivar like: The field must match the provided value.
    :vartype like: str
    :ivar nlike: The field must not match the provided value.
    :vartype nlike: str
    :ivar ilike: The field must match the provided value, ignoring case.
    :vartype ilike: str
    :ivar nilike: The field must not match the provided value, ignoring case.
    :vartype nilike: str
    :ivar gt: The field must be greater than the provided value.
    :vartype gt: str
    :ivar gte: The field must be greater than or equal to the provided value.
    :vartype gte: str
    :ivar lt: The field must be less than the provided value.
    :vartype lt: str
    :ivar lte: The field must be less than or equal to the provided value.
    :vartype lte: str
    :ivar and_property: Provide a list of filters to be combined with a logical AND.
    :vartype and_property: list[~openmeter._generated.models.FilterString]
    :ivar or_property: Provide a list of filters to be combined with a logical OR.
    :vartype or_property: list[~openmeter._generated.models.FilterString]
    """

    eq: Optional[str] = rest_field(name="$eq", visibility=["read", "create", "update", "delete", "query"])
    """The field must be equal to the provided value."""
    ne: Optional[str] = rest_field(name="$ne", visibility=["read", "create", "update", "delete", "query"])
    """The field must not be equal to the provided value."""
    in_property: Optional[list[str]] = rest_field(
        name="$in", visibility=["read", "create", "update", "delete", "query"]
    )
    """The field must be in the provided list of values."""
    nin: Optional[list[str]] = rest_field(name="$nin", visibility=["read", "create", "update", "delete", "query"])
    """The field must not be in the provided list of values."""
    like: Optional[str] = rest_field(name="$like", visibility=["read", "create", "update", "delete", "query"])
    """The field must match the provided value."""
    nlike: Optional[str] = rest_field(name="$nlike", visibility=["read", "create", "update", "delete", "query"])
    """The field must not match the provided value."""
    ilike: Optional[str] = rest_field(name="$ilike", visibility=["read", "create", "update", "delete", "query"])
    """The field must match the provided value, ignoring case."""
    nilike: Optional[str] = rest_field(name="$nilike", visibility=["read", "create", "update", "delete", "query"])
    """The field must not match the provided value, ignoring case."""
    gt: Optional[str] = rest_field(name="$gt", visibility=["read", "create", "update", "delete", "query"])
    """The field must be greater than the provided value."""
    gte: Optional[str] = rest_field(name="$gte", visibility=["read", "create", "update", "delete", "query"])
    """The field must be greater than or equal to the provided value."""
    lt: Optional[str] = rest_field(name="$lt", visibility=["read", "create", "update", "delete", "query"])
    """The field must be less than the provided value."""
    lte: Optional[str] = rest_field(name="$lte", visibility=["read", "create", "update", "delete", "query"])
    """The field must be less than or equal to the provided value."""
    and_property: Optional[list["_models.FilterString"]] = rest_field(
        name="$and", visibility=["read", "create", "update", "delete", "query"]
    )
    """Provide a list of filters to be combined with a logical AND."""
    or_property: Optional[list["_models.FilterString"]] = rest_field(
        name="$or", visibility=["read", "create", "update", "delete", "query"]
    )
    """Provide a list of filters to be combined with a logical OR."""

    @overload
    def __init__(
        self,
        *,
        eq: Optional[str] = None,
        ne: Optional[str] = None,
        in_property: Optional[list[str]] = None,
        nin: Optional[list[str]] = None,
        like: Optional[str] = None,
        nlike: Optional[str] = None,
        ilike: Optional[str] = None,
        nilike: Optional[str] = None,
        gt: Optional[str] = None,
        gte: Optional[str] = None,
        lt: Optional[str] = None,
        lte: Optional[str] = None,
        and_property: Optional[list["_models.FilterString"]] = None,
        or_property: Optional[list["_models.FilterString"]] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class FilterTime(_Model):
    """A filter for a time field.

    :ivar gt: The field must be greater than the provided value.
    :vartype gt: ~datetime.datetime
    :ivar gte: The field must be greater than or equal to the provided value.
    :vartype gte: ~datetime.datetime
    :ivar lt: The field must be less than the provided value.
    :vartype lt: ~datetime.datetime
    :ivar lte: The field must be less than or equal to the provided value.
    :vartype lte: ~datetime.datetime
    :ivar and_property: Provide a list of filters to be combined with a logical AND.
    :vartype and_property: list[~openmeter._generated.models.FilterTime]
    :ivar or_property: Provide a list of filters to be combined with a logical OR.
    :vartype or_property: list[~openmeter._generated.models.FilterTime]
    """

    gt: Optional[datetime.datetime] = rest_field(
        name="$gt", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The field must be greater than the provided value."""
    gte: Optional[datetime.datetime] = rest_field(
        name="$gte", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The field must be greater than or equal to the provided value."""
    lt: Optional[datetime.datetime] = rest_field(
        name="$lt", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The field must be less than the provided value."""
    lte: Optional[datetime.datetime] = rest_field(
        name="$lte", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The field must be less than or equal to the provided value."""
    and_property: Optional[list["_models.FilterTime"]] = rest_field(
        name="$and", visibility=["read", "create", "update", "delete", "query"]
    )
    """Provide a list of filters to be combined with a logical AND."""
    or_property: Optional[list["_models.FilterTime"]] = rest_field(
        name="$or", visibility=["read", "create", "update", "delete", "query"]
    )
    """Provide a list of filters to be combined with a logical OR."""

    @overload
    def __init__(
        self,
        *,
        gt: Optional[datetime.datetime] = None,
        gte: Optional[datetime.datetime] = None,
        lt: Optional[datetime.datetime] = None,
        lte: Optional[datetime.datetime] = None,
        and_property: Optional[list["_models.FilterTime"]] = None,
        or_property: Optional[list["_models.FilterTime"]] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class FlatPrice(_Model):
    """Flat price.

    :ivar type: The type of the price. Required.
    :vartype type: str or ~openmeter._generated.models.FLAT
    :ivar amount: The amount of the flat price. Required.
    :vartype amount: str
    """

    type: Literal[PriceType.FLAT] = rest_field(visibility=["read", "create", "update"])
    """The type of the price. Required."""
    amount: str = rest_field(visibility=["read", "create", "update"])
    """The amount of the flat price. Required."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[PriceType.FLAT],
        amount: str,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class FlatPriceWithPaymentTerm(_Model):
    """Flat price with payment term.

    :ivar type: The type of the price. Required.
    :vartype type: str or ~openmeter._generated.models.FLAT
    :ivar amount: The amount of the flat price. Required.
    :vartype amount: str
    :ivar payment_term: The payment term of the flat price.
     Defaults to in advance. Known values are: "in_advance" and "in_arrears".
    :vartype payment_term: str or ~openmeter.models.PricePaymentTerm
    """

    type: Literal[PriceType.FLAT] = rest_field(visibility=["read", "create", "update"])
    """The type of the price. Required."""
    amount: str = rest_field(visibility=["read", "create", "update"])
    """The amount of the flat price. Required."""
    payment_term: Optional[Union[str, "_models.PricePaymentTerm"]] = rest_field(
        name="paymentTerm", visibility=["read", "create", "update"]
    )
    """The payment term of the flat price.
     Defaults to in advance. Known values are: \"in_advance\" and \"in_arrears\"."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[PriceType.FLAT],
        amount: str,
        payment_term: Optional[Union[str, "_models.PricePaymentTerm"]] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class ForbiddenProblemResponse(UnexpectedProblemResponse):
    """The server understood the request but refuses to authorize it.

    :ivar type: Type contains a URI that identifies the problem type. Required.
    :vartype type: str
    :ivar title: A a short, human-readable summary of the problem type. Required.
    :vartype title: str
    :ivar status: The HTTP status code generated by the origin server for this occurrence of the
     problem.
    :vartype status: int
    :ivar detail: A human-readable explanation specific to this occurrence of the problem.
     Required.
    :vartype detail: str
    :ivar instance: A URI reference that identifies the specific occurrence of the problem.
     Required.
    :vartype instance: str
    :ivar extensions: Additional properties specific to the problem type may be present.
    :vartype extensions: dict[str, any]
    """

    @overload
    def __init__(
        self,
        *,
        type: str,
        title: str,
        detail: str,
        instance: str,
        status: Optional[int] = None,
        extensions: Optional[dict[str, Any]] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class GrantBurnDownHistorySegment(_Model):
    """A segment of the grant burn down history.

    A given segment represents the usage of a grant between events that changed either the grant
    burn down priority order or the usag period.

    :ivar period: The period of the segment. Required.
    :vartype period: ~openmeter._generated.models.Period
    :ivar usage: The total usage of the grant in the period. Required.
    :vartype usage: float
    :ivar overage: Overuse that wasn't covered by grants. Required.
    :vartype overage: float
    :ivar balance_at_start: entitlement balance at the start of the period. Required.
    :vartype balance_at_start: float
    :ivar grant_balances_at_start: The balance breakdown of each active grant at the start of the
     period: GrantID: Balance. Required.
    :vartype grant_balances_at_start: dict[str, float]
    :ivar balance_at_end: The entitlement balance at the end of the period. Required.
    :vartype balance_at_end: float
    :ivar grant_balances_at_end: The balance breakdown of each active grant at the end of the
     period: GrantID: Balance. Required.
    :vartype grant_balances_at_end: dict[str, float]
    :ivar grant_usages: Which grants were actually burnt down in the period and by what amount.
     Required.
    :vartype grant_usages: list[~openmeter._generated.models.GrantUsageRecord]
    """

    period: "_models.Period" = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The period of the segment. Required."""
    usage: float = rest_field(visibility=["read"])
    """The total usage of the grant in the period. Required."""
    overage: float = rest_field(visibility=["read"])
    """Overuse that wasn't covered by grants. Required."""
    balance_at_start: float = rest_field(name="balanceAtStart", visibility=["read"])
    """entitlement balance at the start of the period. Required."""
    grant_balances_at_start: dict[str, float] = rest_field(name="grantBalancesAtStart", visibility=["read"])
    """The balance breakdown of each active grant at the start of the period: GrantID: Balance.
     Required."""
    balance_at_end: float = rest_field(name="balanceAtEnd", visibility=["read"])
    """The entitlement balance at the end of the period. Required."""
    grant_balances_at_end: dict[str, float] = rest_field(name="grantBalancesAtEnd", visibility=["read"])
    """The balance breakdown of each active grant at the end of the period: GrantID: Balance.
     Required."""
    grant_usages: list["_models.GrantUsageRecord"] = rest_field(name="grantUsages", visibility=["read"])
    """Which grants were actually burnt down in the period and by what amount. Required."""

    @overload
    def __init__(
        self,
        *,
        period: "_models.Period",
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class GrantPaginatedResponse(_Model):
    """Paginated response.

    :ivar total_count: The total number of items. Required.
    :vartype total_count: int
    :ivar page: The page index. Required.
    :vartype page: int
    :ivar page_size: The maximum number of items per page. Required.
    :vartype page_size: int
    :ivar items_property: The items in the current page. Required.
    :vartype items_property: list[~openmeter._generated.models.EntitlementGrant]
    """

    total_count: int = rest_field(name="totalCount", visibility=["read", "create", "update", "delete", "query"])
    """The total number of items. Required."""
    page: int = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The page index. Required."""
    page_size: int = rest_field(name="pageSize", visibility=["read", "create", "update", "delete", "query"])
    """The maximum number of items per page. Required."""
    items_property: list["_models.EntitlementGrant"] = rest_field(
        name="items", visibility=["read", "create", "update", "delete", "query"]
    )
    """The items in the current page. Required."""

    @overload
    def __init__(
        self,
        *,
        total_count: int,
        page: int,
        page_size: int,
        items_property: list["_models.EntitlementGrant"],
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class GrantUsageRecord(_Model):
    """Usage Record.

    :ivar grant_id: The id of the grant. Required.
    :vartype grant_id: str
    :ivar usage: The usage in the period. Required.
    :vartype usage: float
    """

    grant_id: str = rest_field(name="grantId", visibility=["read", "create", "update", "delete", "query"])
    """The id of the grant. Required."""
    usage: float = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The usage in the period. Required."""

    @overload
    def __init__(
        self,
        *,
        grant_id: str,
        usage: float,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class GrantV2PaginatedResponse(_Model):
    """Paginated response.

    :ivar total_count: The total number of items. Required.
    :vartype total_count: int
    :ivar page: The page index. Required.
    :vartype page: int
    :ivar page_size: The maximum number of items per page. Required.
    :vartype page_size: int
    :ivar items_property: The items in the current page. Required.
    :vartype items_property: list[~openmeter._generated.models.EntitlementGrantV2]
    """

    total_count: int = rest_field(name="totalCount", visibility=["read", "create", "update", "delete", "query"])
    """The total number of items. Required."""
    page: int = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The page index. Required."""
    page_size: int = rest_field(name="pageSize", visibility=["read", "create", "update", "delete", "query"])
    """The maximum number of items per page. Required."""
    items_property: list["_models.EntitlementGrantV2"] = rest_field(
        name="items", visibility=["read", "create", "update", "delete", "query"]
    )
    """The items in the current page. Required."""

    @overload
    def __init__(
        self,
        *,
        total_count: int,
        page: int,
        page_size: int,
        items_property: list["_models.EntitlementGrantV2"],
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class IDResource(_Model):
    """IDResource is a resouce with an ID.

    :ivar id: ID. Required.
    :vartype id: str
    """

    id: str = rest_field(visibility=["read"])
    """ID. Required."""


class IngestedEvent(_Model):
    """An ingested event with optional validation error.

    :ivar event: The original event ingested. Required.
    :vartype event: ~openmeter._generated.models.Event
    :ivar customer_id: The customer ID if the event is associated with a customer.
    :vartype customer_id: str
    :ivar validation_error: The validation error if the event failed validation.
    :vartype validation_error: str
    :ivar ingested_at: The date and time the event was ingested. Required.
    :vartype ingested_at: ~datetime.datetime
    :ivar stored_at: The date and time the event was stored. Required.
    :vartype stored_at: ~datetime.datetime
    """

    event: "_models.Event" = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The original event ingested. Required."""
    customer_id: Optional[str] = rest_field(
        name="customerId", visibility=["read", "create", "update", "delete", "query"]
    )
    """The customer ID if the event is associated with a customer."""
    validation_error: Optional[str] = rest_field(
        name="validationError", visibility=["read", "create", "update", "delete", "query"]
    )
    """The validation error if the event failed validation."""
    ingested_at: datetime.datetime = rest_field(
        name="ingestedAt", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The date and time the event was ingested. Required."""
    stored_at: datetime.datetime = rest_field(
        name="storedAt", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The date and time the event was stored. Required."""

    @overload
    def __init__(
        self,
        *,
        event: "_models.Event",
        ingested_at: datetime.datetime,
        stored_at: datetime.datetime,
        customer_id: Optional[str] = None,
        validation_error: Optional[str] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class InstallWithApiKeyRequest(_Model):
    """InstallWithApiKeyRequest.

    :ivar name: Name of the application to install.

     If name is not provided defaults to the marketplace listing's name.
    :vartype name: str
    :ivar create_billing_profile: If true, a billing profile will be created for the app.
     The Stripe app will be also set as the default billing profile if the current default is a
     Sandbox app.
    :vartype create_billing_profile: bool
    :ivar api_key: The API key for the provider.
     For example, the Stripe API key. Required.
    :vartype api_key: str
    """

    name: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Name of the application to install.
     
     If name is not provided defaults to the marketplace listing's name."""
    create_billing_profile: Optional[bool] = rest_field(
        name="createBillingProfile", visibility=["read", "create", "update", "delete", "query"]
    )
    """If true, a billing profile will be created for the app.
     The Stripe app will be also set as the default billing profile if the current default is a
     Sandbox app."""
    api_key: str = rest_field(name="apiKey", visibility=["read", "create", "update", "delete", "query"])
    """The API key for the provider.
     For example, the Stripe API key. Required."""

    @overload
    def __init__(
        self,
        *,
        api_key: str,
        name: Optional[str] = None,
        create_billing_profile: Optional[bool] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class InternalServerErrorProblemResponse(UnexpectedProblemResponse):
    """The server encountered an unexpected condition that prevented it from fulfilling the request.

    :ivar type: Type contains a URI that identifies the problem type. Required.
    :vartype type: str
    :ivar title: A a short, human-readable summary of the problem type. Required.
    :vartype title: str
    :ivar status: The HTTP status code generated by the origin server for this occurrence of the
     problem.
    :vartype status: int
    :ivar detail: A human-readable explanation specific to this occurrence of the problem.
     Required.
    :vartype detail: str
    :ivar instance: A URI reference that identifies the specific occurrence of the problem.
     Required.
    :vartype instance: str
    :ivar extensions: Additional properties specific to the problem type may be present.
    :vartype extensions: dict[str, any]
    """

    @overload
    def __init__(
        self,
        *,
        type: str,
        title: str,
        detail: str,
        instance: str,
        status: Optional[int] = None,
        extensions: Optional[dict[str, Any]] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class Invoice(_Model):
    """Invoice represents an invoice in the system.

        :ivar id: ID. Required.
        :vartype id: str
        :ivar description: Description.
        :vartype description: str
        :ivar metadata: Metadata.
        :vartype metadata: ~openmeter._generated.models.Metadata
        :ivar created_at: Creation Time. Required.
        :vartype created_at: ~datetime.datetime
        :ivar updated_at: Last Update Time. Required.
        :vartype updated_at: ~datetime.datetime
        :ivar deleted_at: Deletion Time.
        :vartype deleted_at: ~datetime.datetime
        :ivar type: Type of the invoice.

    The type of invoice determines the purpose of the invoice and how it should be handled.

    Supported types:

         * standard: A regular commercial invoice document between a supplier and customer.
         * credit_note: Reflects a refund either partial or complete of the preceding document. A
           credit note effectively *extends* the previous document. Required. Known values are: "standard"
           and "credit_note".
        :vartype type: str or ~openmeter.models.InvoiceType
        :ivar supplier: The taxable entity supplying the goods or services. Required.
        :vartype supplier: ~openmeter._generated.models.BillingParty
        :ivar customer: Legal entity receiving the goods or services. Required.
        :vartype customer: ~openmeter._generated.models.BillingInvoiceCustomerExtendedDetails
        :ivar number: Number specifies the human readable key used to reference this Invoice.

         The invoice number can change in the draft phases, as we are allocating temporary draft
         invoice numbers, but it's final as soon as the invoice gets finalized (issued state).

         Please note that the number is (depending on the upstream settings) either unique for the
         whole organization or unique for the customer, or in multi (stripe) account setups unique for
         the
         account. Required.
        :vartype number: str
        :ivar currency: Currency for all invoice line items.

         Multi currency invoices are not supported yet. Required.
        :vartype currency: str
        :ivar preceding: Key information regarding previous invoices and potentially details as to why
         they were corrected.
        :vartype preceding: list[~openmeter._generated.models.CreditNoteOriginalInvoiceRef]
        :ivar totals: Summary of all the invoice totals, including taxes (calculated). Required.
        :vartype totals: ~openmeter._generated.models.InvoiceTotals
        :ivar status: The status of the invoice.

         This field only conatins a simplified status, for more detailed information use the
         statusDetails field. Required. Known values are: "gathering", "draft", "issuing", "issued",
         "payment_processing", "overdue", "paid", "uncollectible", and "voided".
        :vartype status: str or ~openmeter.models.InvoiceStatus
        :ivar status_details: The details of the current invoice status. Required.
        :vartype status_details: ~openmeter._generated.models.InvoiceStatusDetails
        :ivar issued_at: The time the invoice was issued.

    Depending on the status of the invoice this can mean multiple things:

         * draft, gathering: The time the invoice will be issued based on the workflow settings.
         * issued: The time the invoice was issued.
        :vartype issued_at: ~datetime.datetime
        :ivar draft_until: The time until the invoice is in draft status.

         On draft invoice creation it is calculated from the workflow settings.

         If manual approval is required, the draftUntil time is set.
        :vartype draft_until: ~datetime.datetime
        :ivar quantity_snapshoted_at: The time when the quantity snapshots on the invoice lines were
         taken.
        :vartype quantity_snapshoted_at: ~datetime.datetime
        :ivar collection_at: The time when the invoice will be/has been collected.
        :vartype collection_at: ~datetime.datetime
        :ivar due_at: Due time of the fulfillment of the invoice (if available).
        :vartype due_at: ~datetime.datetime
        :ivar period: The period the invoice covers. If the invoice has no line items, it's not set.
        :vartype period: ~openmeter._generated.models.Period
        :ivar voided_at: The time the invoice was voided.

         If the invoice was voided, this field will be set to the time the invoice was voided.
        :vartype voided_at: ~datetime.datetime
        :ivar sent_to_customer_at: The time the invoice was sent to customer.
        :vartype sent_to_customer_at: ~datetime.datetime
        :ivar workflow: The workflow associated with the invoice.

         It is always a snapshot of the workflow settings at the time of invoice creation. The
         field is optional as it should be explicitly requested with expand options. Required.
        :vartype workflow: ~openmeter._generated.models.InvoiceWorkflowSettings
        :ivar lines: List of invoice lines representing each of the items sold to the customer.
        :vartype lines: list[~openmeter._generated.models.InvoiceLine]
        :ivar payment: Information on when, how, and to whom the invoice should be paid.
        :vartype payment: ~openmeter._generated.models.InvoicePaymentTerms
        :ivar validation_issues: Validation issues reported by the invoice workflow.
        :vartype validation_issues: list[~openmeter._generated.models.ValidationIssue]
        :ivar external_ids: External IDs of the invoice in other apps such as Stripe.
        :vartype external_ids: ~openmeter._generated.models.InvoiceAppExternalIds
    """

    id: str = rest_field(visibility=["read"])
    """ID. Required."""
    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    created_at: datetime.datetime = rest_field(name="createdAt", visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    updated_at: datetime.datetime = rest_field(name="updatedAt", visibility=["read"], format="rfc3339")
    """Last Update Time. Required."""
    deleted_at: Optional[datetime.datetime] = rest_field(name="deletedAt", visibility=["read"], format="rfc3339")
    """Deletion Time."""
    type: Union[str, "_models.InvoiceType"] = rest_field(visibility=["read"])
    """Type of the invoice.
 
 The type of invoice determines the purpose of the invoice and how it should be handled.
 
 Supported types:
 
      * standard: A regular commercial invoice document between a supplier and customer.
      * credit_note: Reflects a refund either partial or complete of the preceding document. A
        credit note effectively *extends* the previous document. Required. Known values are:
        \"standard\" and \"credit_note\"."""
    supplier: "_models.BillingParty" = rest_field(visibility=["read", "create", "update"])
    """The taxable entity supplying the goods or services. Required."""
    customer: "_models.BillingInvoiceCustomerExtendedDetails" = rest_field(visibility=["read", "create", "update"])
    """Legal entity receiving the goods or services. Required."""
    number: str = rest_field(visibility=["read"])
    """Number specifies the human readable key used to reference this Invoice.
     
     The invoice number can change in the draft phases, as we are allocating temporary draft
     invoice numbers, but it's final as soon as the invoice gets finalized (issued state).
     
     Please note that the number is (depending on the upstream settings) either unique for the
     whole organization or unique for the customer, or in multi (stripe) account setups unique for
     the
     account. Required."""
    currency: str = rest_field(visibility=["read", "create"])
    """Currency for all invoice line items.
     
     Multi currency invoices are not supported yet. Required."""
    preceding: Optional[list["_types.InvoiceDocumentRef"]] = rest_field(visibility=["read"])
    """Key information regarding previous invoices and potentially details as to why they were
     corrected."""
    totals: "_models.InvoiceTotals" = rest_field(visibility=["read"])
    """Summary of all the invoice totals, including taxes (calculated). Required."""
    status: Union[str, "_models.InvoiceStatus"] = rest_field(visibility=["read"])
    """The status of the invoice.
     
     This field only conatins a simplified status, for more detailed information use the
     statusDetails field. Required. Known values are: \"gathering\", \"draft\", \"issuing\",
     \"issued\", \"payment_processing\", \"overdue\", \"paid\", \"uncollectible\", and \"voided\"."""
    status_details: "_models.InvoiceStatusDetails" = rest_field(name="statusDetails", visibility=["read"])
    """The details of the current invoice status. Required."""
    issued_at: Optional[datetime.datetime] = rest_field(name="issuedAt", visibility=["read"], format="rfc3339")
    """The time the invoice was issued.
 
 Depending on the status of the invoice this can mean multiple things:
 
      * draft, gathering: The time the invoice will be issued based on the workflow settings.
      * issued: The time the invoice was issued."""
    draft_until: Optional[datetime.datetime] = rest_field(
        name="draftUntil", visibility=["read", "update"], format="rfc3339"
    )
    """The time until the invoice is in draft status.
     
     On draft invoice creation it is calculated from the workflow settings.
     
     If manual approval is required, the draftUntil time is set."""
    quantity_snapshoted_at: Optional[datetime.datetime] = rest_field(
        name="quantitySnapshotedAt", visibility=["read"], format="rfc3339"
    )
    """The time when the quantity snapshots on the invoice lines were taken."""
    collection_at: Optional[datetime.datetime] = rest_field(name="collectionAt", visibility=["read"], format="rfc3339")
    """The time when the invoice will be/has been collected."""
    due_at: Optional[datetime.datetime] = rest_field(name="dueAt", visibility=["read"], format="rfc3339")
    """Due time of the fulfillment of the invoice (if available)."""
    period: Optional["_models.Period"] = rest_field(visibility=["read", "create"])
    """The period the invoice covers. If the invoice has no line items, it's not set."""
    voided_at: Optional[datetime.datetime] = rest_field(name="voidedAt", visibility=["read"], format="rfc3339")
    """The time the invoice was voided.
     
     If the invoice was voided, this field will be set to the time the invoice was voided."""
    sent_to_customer_at: Optional[datetime.datetime] = rest_field(
        name="sentToCustomerAt", visibility=["read"], format="rfc3339"
    )
    """The time the invoice was sent to customer."""
    workflow: "_models.InvoiceWorkflowSettings" = rest_field(visibility=["read", "create", "update"])
    """The workflow associated with the invoice.
     
     It is always a snapshot of the workflow settings at the time of invoice creation. The
     field is optional as it should be explicitly requested with expand options. Required."""
    lines: Optional[list["_models.InvoiceLine"]] = rest_field(visibility=["read", "update"])
    """List of invoice lines representing each of the items sold to the customer."""
    payment: Optional["_models.InvoicePaymentTerms"] = rest_field(visibility=["read"])
    """Information on when, how, and to whom the invoice should be paid."""
    validation_issues: Optional[list["_models.ValidationIssue"]] = rest_field(
        name="validationIssues", visibility=["read"]
    )
    """Validation issues reported by the invoice workflow."""
    external_ids: Optional["_models.InvoiceAppExternalIds"] = rest_field(name="externalIds", visibility=["read"])
    """External IDs of the invoice in other apps such as Stripe."""

    @overload
    def __init__(  # pylint: disable=too-many-locals
        self,
        *,
        supplier: "_models.BillingParty",
        customer: "_models.BillingInvoiceCustomerExtendedDetails",
        currency: str,
        workflow: "_models.InvoiceWorkflowSettings",
        description: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
        draft_until: Optional[datetime.datetime] = None,
        period: Optional["_models.Period"] = None,
        lines: Optional[list["_models.InvoiceLine"]] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class InvoiceAppExternalIds(_Model):
    """InvoiceAppExternalIds contains the external IDs of the invoice in other apps such as Stripe.

    :ivar invoicing: The external ID of the invoice in the invoicing app if available.
    :vartype invoicing: str
    :ivar tax: The external ID of the invoice in the tax app if available.
    :vartype tax: str
    :ivar payment: The external ID of the invoice in the payment app if available.
    :vartype payment: str
    """

    invoicing: Optional[str] = rest_field(visibility=["read"])
    """The external ID of the invoice in the invoicing app if available."""
    tax: Optional[str] = rest_field(visibility=["read"])
    """The external ID of the invoice in the tax app if available."""
    payment: Optional[str] = rest_field(visibility=["read"])
    """The external ID of the invoice in the payment app if available."""


class InvoiceAvailableActionDetails(_Model):
    """InvoiceAvailableActionInvoiceDetails represents the details of the invoice action for
    non-gathering invoices.

    :ivar resulting_state: The state the invoice will reach if the action is activated and
     all intermediate steps are successful.

     For example advancing a draft_created invoice will result in a draft_manual_approval_needed
     invoice. Required.
    :vartype resulting_state: str
    """

    resulting_state: str = rest_field(name="resultingState", visibility=["read"])
    """The state the invoice will reach if the action is activated and
     all intermediate steps are successful.
     
     For example advancing a draft_created invoice will result in a draft_manual_approval_needed
     invoice. Required."""


class InvoiceAvailableActionInvoiceDetails(_Model):
    """InvoiceAvailableActionInvoiceDetails represents the details of the invoice action for
    gathering invoices.

    """


class InvoiceAvailableActions(_Model):
    """InvoiceAvailableActions represents the actions that can be performed on the invoice.

    :ivar advance: Advance the invoice to the next status.
    :vartype advance: ~openmeter._generated.models.InvoiceAvailableActionDetails
    :ivar approve: Approve an invoice that requires manual approval.
    :vartype approve: ~openmeter._generated.models.InvoiceAvailableActionDetails
    :ivar delete: Delete the invoice (only non-issued invoices can be deleted).
    :vartype delete: ~openmeter._generated.models.InvoiceAvailableActionDetails
    :ivar retry: Retry an invoice issuing step that failed.
    :vartype retry: ~openmeter._generated.models.InvoiceAvailableActionDetails
    :ivar snapshot_quantities: Snapshot quantities for usage based line items.
    :vartype snapshot_quantities: ~openmeter._generated.models.InvoiceAvailableActionDetails
    :ivar void: Void an already issued invoice.
    :vartype void: ~openmeter._generated.models.InvoiceAvailableActionDetails
    :ivar invoice: Invoice a gathering invoice.
    :vartype invoice: ~openmeter._generated.models.InvoiceAvailableActionInvoiceDetails
    """

    advance: Optional["_models.InvoiceAvailableActionDetails"] = rest_field(visibility=["read"])
    """Advance the invoice to the next status."""
    approve: Optional["_models.InvoiceAvailableActionDetails"] = rest_field(visibility=["read"])
    """Approve an invoice that requires manual approval."""
    delete: Optional["_models.InvoiceAvailableActionDetails"] = rest_field(visibility=["read"])
    """Delete the invoice (only non-issued invoices can be deleted)."""
    retry: Optional["_models.InvoiceAvailableActionDetails"] = rest_field(visibility=["read"])
    """Retry an invoice issuing step that failed."""
    snapshot_quantities: Optional["_models.InvoiceAvailableActionDetails"] = rest_field(
        name="snapshotQuantities", visibility=["read"]
    )
    """Snapshot quantities for usage based line items."""
    void: Optional["_models.InvoiceAvailableActionDetails"] = rest_field(visibility=["read"])
    """Void an already issued invoice."""
    invoice: Optional["_models.InvoiceAvailableActionInvoiceDetails"] = rest_field(visibility=["read"])
    """Invoice a gathering invoice."""


class InvoiceDetailedLine(_Model):
    """InvoiceDetailedLine represents a line item that is sold to the customer as a manually added
    fee.

    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar created_at: Creation Time. Required.
    :vartype created_at: ~datetime.datetime
    :ivar updated_at: Last Update Time. Required.
    :vartype updated_at: ~datetime.datetime
    :ivar deleted_at: Deletion Time.
    :vartype deleted_at: ~datetime.datetime
    :ivar id: ID of the line. Required.
    :vartype id: str
    :ivar managed_by: managedBy specifies if the line is manually added via the api or managed by
     OpenMeter. Required. Known values are: "subscription", "system", and "manual".
    :vartype managed_by: str or ~openmeter.models.InvoiceLineManagedBy
    :ivar status: Status of the line.

     External calls always create valid lines, other line types are managed by the
     billing engine of OpenMeter. Required. Known values are: "valid", "detailed", and "split".
    :vartype status: str or ~openmeter.models.InvoiceLineStatus
    :ivar discounts: Discounts detailes applied to this line.

     New discounts can be added via the invoice's discounts API, to facilitate
     discounts that are affecting multiple lines.
    :vartype discounts: ~openmeter._generated.models.InvoiceLineDiscounts
    :ivar invoice: The invoice this item belongs to.
    :vartype invoice: ~openmeter._generated.models.InvoiceReference
    :ivar currency: The currency of this line. Required.
    :vartype currency: str
    :ivar taxes: Taxes applied to the invoice totals.
    :vartype taxes: list[~openmeter._generated.models.InvoiceLineTaxItem]
    :ivar tax_config: Tax config specify the tax configuration for this line.
    :vartype tax_config: ~openmeter._generated.models.TaxConfig
    :ivar totals: Totals for this line. Required.
    :vartype totals: ~openmeter._generated.models.InvoiceTotals
    :ivar period: Period of the line item applies to for revenue recognition pruposes.

     Billing always treats periods as start being inclusive and end being exclusive. Required.
    :vartype period: ~openmeter._generated.models.Period
    :ivar external_ids: External IDs of the invoice in other apps such as Stripe.
    :vartype external_ids: ~openmeter._generated.models.InvoiceLineAppExternalIds
    :ivar subscription: Subscription are the references to the subscritpions that this line is
     related to.
    :vartype subscription: ~openmeter._generated.models.InvoiceLineSubscriptionReference
    :ivar invoice_at: The time this line item should be invoiced. Required.
    :vartype invoice_at: ~datetime.datetime
    :ivar type: Type of the line. Required.
    :vartype type: str or ~openmeter._generated.models.FLAT_FEE
    :ivar per_unit_amount: Price of the item being sold.
    :vartype per_unit_amount: str
    :ivar payment_term: Payment term of the line. Known values are: "in_advance" and "in_arrears".
    :vartype payment_term: str or ~openmeter.models.PricePaymentTerm
    :ivar quantity: Quantity of the item being sold.
    :vartype quantity: str
    :ivar rate_card: The rate card that is used for this line.
    :vartype rate_card: ~openmeter._generated.models.InvoiceDetailedLineRateCard
    :ivar category: Category of the flat fee. Known values are: "regular" and "commitment".
    :vartype category: str or ~openmeter.models.InvoiceDetailedLineCostCategory
    """

    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Display name. Required."""
    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    created_at: datetime.datetime = rest_field(name="createdAt", visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    updated_at: datetime.datetime = rest_field(name="updatedAt", visibility=["read"], format="rfc3339")
    """Last Update Time. Required."""
    deleted_at: Optional[datetime.datetime] = rest_field(name="deletedAt", visibility=["read"], format="rfc3339")
    """Deletion Time."""
    id: str = rest_field(visibility=["read", "update"])
    """ID of the line. Required."""
    managed_by: Union[str, "_models.InvoiceLineManagedBy"] = rest_field(name="managedBy", visibility=["read"])
    """managedBy specifies if the line is manually added via the api or managed by OpenMeter.
     Required. Known values are: \"subscription\", \"system\", and \"manual\"."""
    status: Union[str, "_models.InvoiceLineStatus"] = rest_field(visibility=["read"])
    """Status of the line.
     
     External calls always create valid lines, other line types are managed by the
     billing engine of OpenMeter. Required. Known values are: \"valid\", \"detailed\", and
     \"split\"."""
    discounts: Optional["_models.InvoiceLineDiscounts"] = rest_field(visibility=["read"])
    """Discounts detailes applied to this line.
     
     New discounts can be added via the invoice's discounts API, to facilitate
     discounts that are affecting multiple lines."""
    invoice: Optional["_models.InvoiceReference"] = rest_field(visibility=["read", "create"])
    """The invoice this item belongs to."""
    currency: str = rest_field(visibility=["read", "create"])
    """The currency of this line. Required."""
    taxes: Optional[list["_models.InvoiceLineTaxItem"]] = rest_field(visibility=["read"])
    """Taxes applied to the invoice totals."""
    tax_config: Optional["_models.TaxConfig"] = rest_field(name="taxConfig", visibility=["read", "create", "update"])
    """Tax config specify the tax configuration for this line."""
    totals: "_models.InvoiceTotals" = rest_field(visibility=["read"])
    """Totals for this line. Required."""
    period: "_models.Period" = rest_field(visibility=["read", "create", "update"])
    """Period of the line item applies to for revenue recognition pruposes.
     
     Billing always treats periods as start being inclusive and end being exclusive. Required."""
    external_ids: Optional["_models.InvoiceLineAppExternalIds"] = rest_field(name="externalIds", visibility=["read"])
    """External IDs of the invoice in other apps such as Stripe."""
    subscription: Optional["_models.InvoiceLineSubscriptionReference"] = rest_field(visibility=["read"])
    """Subscription are the references to the subscritpions that this line is related to."""
    invoice_at: datetime.datetime = rest_field(
        name="invoiceAt", visibility=["read", "create", "update"], format="rfc3339"
    )
    """The time this line item should be invoiced. Required."""
    type: Literal[InvoiceLineTypes.FLAT_FEE] = rest_field(visibility=["read"])
    """Type of the line. Required."""
    per_unit_amount: Optional[str] = rest_field(name="perUnitAmount", visibility=["read", "create", "update"])
    """Price of the item being sold."""
    payment_term: Optional[Union[str, "_models.PricePaymentTerm"]] = rest_field(
        name="paymentTerm", visibility=["read", "create", "update"]
    )
    """Payment term of the line. Known values are: \"in_advance\" and \"in_arrears\"."""
    quantity: Optional[str] = rest_field(visibility=["read", "create", "update"])
    """Quantity of the item being sold."""
    rate_card: Optional["_models.InvoiceDetailedLineRateCard"] = rest_field(
        name="rateCard", visibility=["read", "create", "update"]
    )
    """The rate card that is used for this line."""
    category: Optional[Union[str, "_models.InvoiceDetailedLineCostCategory"]] = rest_field(visibility=["read"])
    """Category of the flat fee. Known values are: \"regular\" and \"commitment\"."""

    @overload
    def __init__(  # pylint: disable=too-many-locals
        self,
        *,
        name: str,
        id: str,  # pylint: disable=redefined-builtin
        currency: str,
        period: "_models.Period",
        invoice_at: datetime.datetime,
        description: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
        invoice: Optional["_models.InvoiceReference"] = None,
        tax_config: Optional["_models.TaxConfig"] = None,
        per_unit_amount: Optional[str] = None,
        payment_term: Optional[Union[str, "_models.PricePaymentTerm"]] = None,
        quantity: Optional[str] = None,
        rate_card: Optional["_models.InvoiceDetailedLineRateCard"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class InvoiceDetailedLineRateCard(_Model):
    """InvoiceDetailedLineRateCard represents the rate card (intent) for a flat fee line.

    :ivar tax_config: Tax config.
    :vartype tax_config: ~openmeter._generated.models.TaxConfig
    :ivar price: Price. Required.
    :vartype price: ~openmeter._generated.models.FlatPriceWithPaymentTerm
    :ivar quantity: Quantity of the item being sold.

     Default: 1.
    :vartype quantity: str
    :ivar discounts: The discounts that are applied to the line.
    :vartype discounts: ~openmeter._generated.models.BillingDiscounts
    """

    tax_config: Optional["_models.TaxConfig"] = rest_field(name="taxConfig", visibility=["read", "create", "update"])
    """Tax config."""
    price: "_models.FlatPriceWithPaymentTerm" = rest_field(visibility=["read", "create", "update"])
    """Price. Required."""
    quantity: Optional[str] = rest_field(visibility=["read", "create", "update"])
    """Quantity of the item being sold.
     
     Default: 1."""
    discounts: Optional["_models.BillingDiscounts"] = rest_field(visibility=["read", "create", "update"])
    """The discounts that are applied to the line."""

    @overload
    def __init__(
        self,
        *,
        price: "_models.FlatPriceWithPaymentTerm",
        tax_config: Optional["_models.TaxConfig"] = None,
        quantity: Optional[str] = None,
        discounts: Optional["_models.BillingDiscounts"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class InvoiceLine(_Model):
    """InvoiceUsageBasedLine represents a line item that is sold to the customer based on usage.

    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar created_at: Creation Time. Required.
    :vartype created_at: ~datetime.datetime
    :ivar updated_at: Last Update Time. Required.
    :vartype updated_at: ~datetime.datetime
    :ivar deleted_at: Deletion Time.
    :vartype deleted_at: ~datetime.datetime
    :ivar id: ID of the line. Required.
    :vartype id: str
    :ivar managed_by: managedBy specifies if the line is manually added via the api or managed by
     OpenMeter. Required. Known values are: "subscription", "system", and "manual".
    :vartype managed_by: str or ~openmeter.models.InvoiceLineManagedBy
    :ivar status: Status of the line.

     External calls always create valid lines, other line types are managed by the
     billing engine of OpenMeter. Required. Known values are: "valid", "detailed", and "split".
    :vartype status: str or ~openmeter.models.InvoiceLineStatus
    :ivar discounts: Discounts detailes applied to this line.

     New discounts can be added via the invoice's discounts API, to facilitate
     discounts that are affecting multiple lines.
    :vartype discounts: ~openmeter._generated.models.InvoiceLineDiscounts
    :ivar invoice: The invoice this item belongs to.
    :vartype invoice: ~openmeter._generated.models.InvoiceReference
    :ivar currency: The currency of this line. Required.
    :vartype currency: str
    :ivar taxes: Taxes applied to the invoice totals.
    :vartype taxes: list[~openmeter._generated.models.InvoiceLineTaxItem]
    :ivar tax_config: Tax config specify the tax configuration for this line.
    :vartype tax_config: ~openmeter._generated.models.TaxConfig
    :ivar totals: Totals for this line. Required.
    :vartype totals: ~openmeter._generated.models.InvoiceTotals
    :ivar period: Period of the line item applies to for revenue recognition pruposes.

     Billing always treats periods as start being inclusive and end being exclusive. Required.
    :vartype period: ~openmeter._generated.models.Period
    :ivar invoice_at: The time this line item should be invoiced. Required.
    :vartype invoice_at: ~datetime.datetime
    :ivar external_ids: External IDs of the invoice in other apps such as Stripe.
    :vartype external_ids: ~openmeter._generated.models.InvoiceLineAppExternalIds
    :ivar subscription: Subscription are the references to the subscritpions that this line is
     related to.
    :vartype subscription: ~openmeter._generated.models.InvoiceLineSubscriptionReference
    :ivar type: Type of the line. Required.
    :vartype type: str or ~openmeter._generated.models.USAGE_BASED
    :ivar price: Price of the usage-based item being sold. Is one of the following types:
     FlatPriceWithPaymentTerm, UnitPriceWithCommitments, TieredPriceWithCommitments,
     DynamicPriceWithCommitments, PackagePriceWithCommitments
    :vartype price: ~openmeter._generated.models.FlatPriceWithPaymentTerm or
     ~openmeter._generated.models.UnitPriceWithCommitments or
     ~openmeter._generated.models.TieredPriceWithCommitments or
     ~openmeter._generated.models.DynamicPriceWithCommitments or
     ~openmeter._generated.models.PackagePriceWithCommitments
    :ivar feature_key: The feature that the usage is based on.
    :vartype feature_key: str
    :ivar children: The lines detailing the item or service sold.
    :vartype children: list[~openmeter._generated.models.InvoiceDetailedLine]
    :ivar rate_card: The rate card that is used for this line.

     The rate card captures the intent of the price and discounts for the usage-based item.
    :vartype rate_card: ~openmeter._generated.models.InvoiceUsageBasedRateCard
    :ivar quantity: The quantity of the item being sold.

     Any usage discounts applied previously are deducted from this quantity.
    :vartype quantity: str
    :ivar metered_quantity: The quantity of the item that has been metered for the period before
     any discounts were applied.
    :vartype metered_quantity: str
    :ivar pre_line_period_quantity: The quantity of the item used before this line's period.

     It is non-zero in case of progressive billing, when this shows how much of the usage was
     already billed.

     Any usage discounts applied previously are deducted from this quantity.
    :vartype pre_line_period_quantity: str
    :ivar metered_pre_line_period_quantity: The metered quantity of the item used in before this
     line's period without any discounts applied.

     It is non-zero in case of progressive billing, when this shows how much of the usage was
     already billed.
    :vartype metered_pre_line_period_quantity: str
    """

    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Display name. Required."""
    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    created_at: datetime.datetime = rest_field(name="createdAt", visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    updated_at: datetime.datetime = rest_field(name="updatedAt", visibility=["read"], format="rfc3339")
    """Last Update Time. Required."""
    deleted_at: Optional[datetime.datetime] = rest_field(name="deletedAt", visibility=["read"], format="rfc3339")
    """Deletion Time."""
    id: str = rest_field(visibility=["read", "update"])
    """ID of the line. Required."""
    managed_by: Union[str, "_models.InvoiceLineManagedBy"] = rest_field(name="managedBy", visibility=["read"])
    """managedBy specifies if the line is manually added via the api or managed by OpenMeter.
     Required. Known values are: \"subscription\", \"system\", and \"manual\"."""
    status: Union[str, "_models.InvoiceLineStatus"] = rest_field(visibility=["read"])
    """Status of the line.
     
     External calls always create valid lines, other line types are managed by the
     billing engine of OpenMeter. Required. Known values are: \"valid\", \"detailed\", and
     \"split\"."""
    discounts: Optional["_models.InvoiceLineDiscounts"] = rest_field(visibility=["read"])
    """Discounts detailes applied to this line.
     
     New discounts can be added via the invoice's discounts API, to facilitate
     discounts that are affecting multiple lines."""
    invoice: Optional["_models.InvoiceReference"] = rest_field(visibility=["read", "create"])
    """The invoice this item belongs to."""
    currency: str = rest_field(visibility=["read", "create"])
    """The currency of this line. Required."""
    taxes: Optional[list["_models.InvoiceLineTaxItem"]] = rest_field(visibility=["read"])
    """Taxes applied to the invoice totals."""
    tax_config: Optional["_models.TaxConfig"] = rest_field(name="taxConfig", visibility=["read", "create", "update"])
    """Tax config specify the tax configuration for this line."""
    totals: "_models.InvoiceTotals" = rest_field(visibility=["read"])
    """Totals for this line. Required."""
    period: "_models.Period" = rest_field(visibility=["read", "create", "update"])
    """Period of the line item applies to for revenue recognition pruposes.
     
     Billing always treats periods as start being inclusive and end being exclusive. Required."""
    invoice_at: datetime.datetime = rest_field(
        name="invoiceAt", visibility=["read", "create", "update"], format="rfc3339"
    )
    """The time this line item should be invoiced. Required."""
    external_ids: Optional["_models.InvoiceLineAppExternalIds"] = rest_field(name="externalIds", visibility=["read"])
    """External IDs of the invoice in other apps such as Stripe."""
    subscription: Optional["_models.InvoiceLineSubscriptionReference"] = rest_field(visibility=["read"])
    """Subscription are the references to the subscritpions that this line is related to."""
    type: Literal[InvoiceLineTypes.USAGE_BASED] = rest_field(visibility=["read"])
    """Type of the line. Required."""
    price: Optional["_types.RateCardUsageBasedPrice"] = rest_field(visibility=["read", "create", "update"])
    """Price of the usage-based item being sold. Is one of the following types:
     FlatPriceWithPaymentTerm, UnitPriceWithCommitments, TieredPriceWithCommitments,
     DynamicPriceWithCommitments, PackagePriceWithCommitments"""
    feature_key: Optional[str] = rest_field(name="featureKey", visibility=["read", "create", "update"])
    """The feature that the usage is based on."""
    children: Optional[list["_models.InvoiceDetailedLine"]] = rest_field(visibility=["read"])
    """The lines detailing the item or service sold."""
    rate_card: Optional["_models.InvoiceUsageBasedRateCard"] = rest_field(
        name="rateCard", visibility=["read", "create", "update"]
    )
    """The rate card that is used for this line.
     
     The rate card captures the intent of the price and discounts for the usage-based item."""
    quantity: Optional[str] = rest_field(visibility=["read"])
    """The quantity of the item being sold.
     
     Any usage discounts applied previously are deducted from this quantity."""
    metered_quantity: Optional[str] = rest_field(name="meteredQuantity", visibility=["read"])
    """The quantity of the item that has been metered for the period before any discounts were
     applied."""
    pre_line_period_quantity: Optional[str] = rest_field(name="preLinePeriodQuantity", visibility=["read"])
    """The quantity of the item used before this line's period.
     
     It is non-zero in case of progressive billing, when this shows how much of the usage was
     already billed.
     
     Any usage discounts applied previously are deducted from this quantity."""
    metered_pre_line_period_quantity: Optional[str] = rest_field(
        name="meteredPreLinePeriodQuantity", visibility=["read"]
    )
    """The metered quantity of the item used in before this line's period without any discounts
     applied.
     
     It is non-zero in case of progressive billing, when this shows how much of the usage was
     already billed."""

    @overload
    def __init__(  # pylint: disable=too-many-locals
        self,
        *,
        name: str,
        id: str,  # pylint: disable=redefined-builtin
        currency: str,
        period: "_models.Period",
        invoice_at: datetime.datetime,
        description: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
        invoice: Optional["_models.InvoiceReference"] = None,
        tax_config: Optional["_models.TaxConfig"] = None,
        price: Optional["_types.RateCardUsageBasedPrice"] = None,
        feature_key: Optional[str] = None,
        rate_card: Optional["_models.InvoiceUsageBasedRateCard"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class InvoiceLineAmountDiscount(_Model):
    """InvoiceLineAmountDiscount represents an amount deducted from the line, and will be applied
    before taxes.

    :ivar created_at: Creation Time. Required.
    :vartype created_at: ~datetime.datetime
    :ivar updated_at: Last Update Time. Required.
    :vartype updated_at: ~datetime.datetime
    :ivar deleted_at: Deletion Time.
    :vartype deleted_at: ~datetime.datetime
    :ivar id: ID of the charge or discount. Required.
    :vartype id: str
    :ivar reason: Reason code. Required. Is one of the following types: DiscountReasonMaximumSpend,
     DiscountReasonRatecardPercentage, DiscountReasonRatecardUsage
    :vartype reason: ~openmeter._generated.models.DiscountReasonMaximumSpend or
     ~openmeter._generated.models.DiscountReasonRatecardPercentage or
     ~openmeter._generated.models.DiscountReasonRatecardUsage
    :ivar description: Text description as to why the discount was applied.
    :vartype description: str
    :ivar external_ids: External IDs of the invoice in other apps such as Stripe.
    :vartype external_ids: ~openmeter._generated.models.InvoiceLineAppExternalIds
    :ivar amount: Amount in the currency of the invoice. Required.
    :vartype amount: str
    """

    created_at: datetime.datetime = rest_field(name="createdAt", visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    updated_at: datetime.datetime = rest_field(name="updatedAt", visibility=["read"], format="rfc3339")
    """Last Update Time. Required."""
    deleted_at: Optional[datetime.datetime] = rest_field(name="deletedAt", visibility=["read"], format="rfc3339")
    """Deletion Time."""
    id: str = rest_field(visibility=["read"])
    """ID of the charge or discount. Required."""
    reason: "_types.BillingDiscountReason" = rest_field(visibility=["read"])
    """Reason code. Required. Is one of the following types: DiscountReasonMaximumSpend,
     DiscountReasonRatecardPercentage, DiscountReasonRatecardUsage"""
    description: Optional[str] = rest_field(visibility=["read"])
    """Text description as to why the discount was applied."""
    external_ids: Optional["_models.InvoiceLineAppExternalIds"] = rest_field(name="externalIds", visibility=["read"])
    """External IDs of the invoice in other apps such as Stripe."""
    amount: str = rest_field(visibility=["read"])
    """Amount in the currency of the invoice. Required."""


class InvoiceLineAppExternalIds(_Model):
    """InvoiceLineAppExternalIds contains the external IDs of the invoice in other apps such as
    Stripe.

    :ivar invoicing: The external ID of the invoice in the invoicing app if available.
    :vartype invoicing: str
    :ivar tax: The external ID of the invoice in the tax app if available.
    :vartype tax: str
    """

    invoicing: Optional[str] = rest_field(visibility=["read"])
    """The external ID of the invoice in the invoicing app if available."""
    tax: Optional[str] = rest_field(visibility=["read"])
    """The external ID of the invoice in the tax app if available."""


class InvoiceLineDiscounts(_Model):
    """InvoiceLineDiscounts represents the discounts applied to the invoice line by type.

    :ivar amount: Amount based discounts applied to the line.

     Amount based discounts are deduced from the total price of the line.
    :vartype amount: list[~openmeter._generated.models.InvoiceLineAmountDiscount]
    :ivar usage: Usage based discounts applied to the line.

     Usage based discounts are deduced from the usage of the line before price calculations are
     applied.
    :vartype usage: list[~openmeter._generated.models.InvoiceLineUsageDiscount]
    """

    amount: Optional[list["_models.InvoiceLineAmountDiscount"]] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """Amount based discounts applied to the line.
     
     Amount based discounts are deduced from the total price of the line."""
    usage: Optional[list["_models.InvoiceLineUsageDiscount"]] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """Usage based discounts applied to the line.
     
     Usage based discounts are deduced from the usage of the line before price calculations are
     applied."""

    @overload
    def __init__(
        self,
        *,
        amount: Optional[list["_models.InvoiceLineAmountDiscount"]] = None,
        usage: Optional[list["_models.InvoiceLineUsageDiscount"]] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class InvoiceLineReplaceUpdate(_Model):
    """InvoiceLineReplaceUpdate represents the update model for an UBP invoice line.

    This type makes ID optional to allow for creating new lines as part of the update.

    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar tax_config: Tax config specify the tax configuration for this line.
    :vartype tax_config: ~openmeter._generated.models.TaxConfig
    :ivar period: Period of the line item applies to for revenue recognition pruposes.

     Billing always treats periods as start being inclusive and end being exclusive. Required.
    :vartype period: ~openmeter._generated.models.Period
    :ivar invoice_at: The time this line item should be invoiced. Required.
    :vartype invoice_at: ~datetime.datetime
    :ivar price: Price of the usage-based item being sold. Is one of the following types:
     FlatPriceWithPaymentTerm, UnitPriceWithCommitments, TieredPriceWithCommitments,
     DynamicPriceWithCommitments, PackagePriceWithCommitments
    :vartype price: ~openmeter._generated.models.FlatPriceWithPaymentTerm or
     ~openmeter._generated.models.UnitPriceWithCommitments or
     ~openmeter._generated.models.TieredPriceWithCommitments or
     ~openmeter._generated.models.DynamicPriceWithCommitments or
     ~openmeter._generated.models.PackagePriceWithCommitments
    :ivar feature_key: The feature that the usage is based on.
    :vartype feature_key: str
    :ivar rate_card: The rate card that is used for this line.

     The rate card captures the intent of the price and discounts for the usage-based item.
    :vartype rate_card: ~openmeter._generated.models.InvoiceUsageBasedRateCard
    :ivar id: The ID of the line.
    :vartype id: str
    """

    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Display name. Required."""
    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    tax_config: Optional["_models.TaxConfig"] = rest_field(name="taxConfig", visibility=["read", "create", "update"])
    """Tax config specify the tax configuration for this line."""
    period: "_models.Period" = rest_field(visibility=["read", "create", "update"])
    """Period of the line item applies to for revenue recognition pruposes.
     
     Billing always treats periods as start being inclusive and end being exclusive. Required."""
    invoice_at: datetime.datetime = rest_field(
        name="invoiceAt", visibility=["read", "create", "update"], format="rfc3339"
    )
    """The time this line item should be invoiced. Required."""
    price: Optional["_types.RateCardUsageBasedPrice"] = rest_field(visibility=["read", "create", "update"])
    """Price of the usage-based item being sold. Is one of the following types:
     FlatPriceWithPaymentTerm, UnitPriceWithCommitments, TieredPriceWithCommitments,
     DynamicPriceWithCommitments, PackagePriceWithCommitments"""
    feature_key: Optional[str] = rest_field(name="featureKey", visibility=["read", "create", "update"])
    """The feature that the usage is based on."""
    rate_card: Optional["_models.InvoiceUsageBasedRateCard"] = rest_field(
        name="rateCard", visibility=["read", "create", "update"]
    )
    """The rate card that is used for this line.
     
     The rate card captures the intent of the price and discounts for the usage-based item."""
    id: Optional[str] = rest_field(visibility=["update"])
    """The ID of the line."""

    @overload
    def __init__(
        self,
        *,
        name: str,
        period: "_models.Period",
        invoice_at: datetime.datetime,
        description: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
        tax_config: Optional["_models.TaxConfig"] = None,
        price: Optional["_types.RateCardUsageBasedPrice"] = None,
        feature_key: Optional[str] = None,
        rate_card: Optional["_models.InvoiceUsageBasedRateCard"] = None,
        id: Optional[str] = None,  # pylint: disable=redefined-builtin
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class InvoiceLineSubscriptionReference(_Model):
    """InvoiceLineSubscriptionReference contains the references to the subscription that this line is
        related to.

        :ivar subscription: The subscription. Required.
        :vartype subscription: ~openmeter._generated.models.IDResource
        :ivar phase: The phase of the subscription. Required.
        :vartype phase: ~openmeter._generated.models.IDResource
        :ivar item: The item this line is related to. Required.
        :vartype item: ~openmeter._generated.models.IDResource
        :ivar billing_period: The billing period of the subscription. In case the subscription item's
        billing period is different
    from the subscription's billing period, this field will contain the billing period of the
        subscription itself.

    For example, in case of:

         * A monthly billed subscription anchored to 2025-01-01
         * A subscription item billed daily

    An example line would have the period of 2025-01-02 to 2025-01-03 as the item is billed daily,
        but the subscription's billing period
    will be 2025-01-01 to 2025-01-31. Required.
        :vartype billing_period: ~openmeter._generated.models.Period
    """

    subscription: "_models.IDResource" = rest_field(visibility=["read"])
    """The subscription. Required."""
    phase: "_models.IDResource" = rest_field(visibility=["read"])
    """The phase of the subscription. Required."""
    item: "_models.IDResource" = rest_field(visibility=["read"])
    """The item this line is related to. Required."""
    billing_period: "_models.Period" = rest_field(name="billingPeriod", visibility=["read"])
    """The billing period of the subscription. In case the subscription item's billing period is
     different
 from the subscription's billing period, this field will contain the billing period of the
     subscription itself.
 
 For example, in case of:
 
      * A monthly billed subscription anchored to 2025-01-01
      * A subscription item billed daily
 
 An example line would have the period of 2025-01-02 to 2025-01-03 as the item is billed daily,
     but the subscription's billing period
 will be 2025-01-01 to 2025-01-31. Required."""


class InvoiceLineTaxItem(_Model):
    """TaxConfig stores the configuration for a tax line relative to an invoice line.

    :ivar config: Tax provider configuration.
    :vartype config: ~openmeter._generated.models.TaxConfig
    :ivar percent: Percent defines the percentage set manually or determined from
     the rate key (calculated if rate present). A nil percent implies that
     this tax combo is **exempt** from tax.").
    :vartype percent: float
    :ivar surcharge: Some countries require an additional surcharge (calculated if rate present).
    :vartype surcharge: str
    :ivar behavior: Is the tax item inclusive or exclusive of the base amount. Known values are:
     "inclusive" and "exclusive".
    :vartype behavior: str or ~openmeter.models.InvoiceLineTaxBehavior
    """

    config: Optional["_models.TaxConfig"] = rest_field(visibility=["read"])
    """Tax provider configuration."""
    percent: Optional[float] = rest_field(visibility=["read"])
    """Percent defines the percentage set manually or determined from
     the rate key (calculated if rate present). A nil percent implies that
     this tax combo is **exempt** from tax.\")."""
    surcharge: Optional[str] = rest_field(visibility=["read"])
    """Some countries require an additional surcharge (calculated if rate present)."""
    behavior: Optional[Union[str, "_models.InvoiceLineTaxBehavior"]] = rest_field(visibility=["read"])
    """Is the tax item inclusive or exclusive of the base amount. Known values are: \"inclusive\" and
     \"exclusive\"."""


class InvoiceLineUsageDiscount(_Model):
    """InvoiceLineUsageDiscount represents an usage-based discount applied to the line.

    The deduction is done before the pricing algorithm is applied.

    :ivar created_at: Creation Time. Required.
    :vartype created_at: ~datetime.datetime
    :ivar updated_at: Last Update Time. Required.
    :vartype updated_at: ~datetime.datetime
    :ivar deleted_at: Deletion Time.
    :vartype deleted_at: ~datetime.datetime
    :ivar id: ID of the charge or discount. Required.
    :vartype id: str
    :ivar reason: Reason code. Required. Is one of the following types: DiscountReasonMaximumSpend,
     DiscountReasonRatecardPercentage, DiscountReasonRatecardUsage
    :vartype reason: ~openmeter._generated.models.DiscountReasonMaximumSpend or
     ~openmeter._generated.models.DiscountReasonRatecardPercentage or
     ~openmeter._generated.models.DiscountReasonRatecardUsage
    :ivar description: Text description as to why the discount was applied.
    :vartype description: str
    :ivar external_ids: External IDs of the invoice in other apps such as Stripe.
    :vartype external_ids: ~openmeter._generated.models.InvoiceLineAppExternalIds
    :ivar quantity: Usage quantity in the unit of the underlying meter. Required.
    :vartype quantity: str
    :ivar pre_line_period_quantity: Usage quantity in the unit of the underlying meter.
    :vartype pre_line_period_quantity: str
    """

    created_at: datetime.datetime = rest_field(name="createdAt", visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    updated_at: datetime.datetime = rest_field(name="updatedAt", visibility=["read"], format="rfc3339")
    """Last Update Time. Required."""
    deleted_at: Optional[datetime.datetime] = rest_field(name="deletedAt", visibility=["read"], format="rfc3339")
    """Deletion Time."""
    id: str = rest_field(visibility=["read"])
    """ID of the charge or discount. Required."""
    reason: "_types.BillingDiscountReason" = rest_field(visibility=["read"])
    """Reason code. Required. Is one of the following types: DiscountReasonMaximumSpend,
     DiscountReasonRatecardPercentage, DiscountReasonRatecardUsage"""
    description: Optional[str] = rest_field(visibility=["read"])
    """Text description as to why the discount was applied."""
    external_ids: Optional["_models.InvoiceLineAppExternalIds"] = rest_field(name="externalIds", visibility=["read"])
    """External IDs of the invoice in other apps such as Stripe."""
    quantity: str = rest_field(visibility=["read"])
    """Usage quantity in the unit of the underlying meter. Required."""
    pre_line_period_quantity: Optional[str] = rest_field(name="preLinePeriodQuantity", visibility=["read"])
    """Usage quantity in the unit of the underlying meter."""


class InvoicePaginatedResponse(_Model):
    """Paginated response.

    :ivar total_count: The total number of items. Required.
    :vartype total_count: int
    :ivar page: The page index. Required.
    :vartype page: int
    :ivar page_size: The maximum number of items per page. Required.
    :vartype page_size: int
    :ivar items_property: The items in the current page. Required.
    :vartype items_property: list[~openmeter._generated.models.Invoice]
    """

    total_count: int = rest_field(name="totalCount", visibility=["read", "create", "update", "delete", "query"])
    """The total number of items. Required."""
    page: int = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The page index. Required."""
    page_size: int = rest_field(name="pageSize", visibility=["read", "create", "update", "delete", "query"])
    """The maximum number of items per page. Required."""
    items_property: list["_models.Invoice"] = rest_field(
        name="items", visibility=["read", "create", "update", "delete", "query"]
    )
    """The items in the current page. Required."""

    @overload
    def __init__(
        self,
        *,
        total_count: int,
        page: int,
        page_size: int,
        items_property: list["_models.Invoice"],
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class InvoicePaymentTerms(_Model):
    """Payment contains details as to how the invoice should be paid.

    :ivar terms: The terms of payment for the invoice. Is either a PaymentTermInstant type or a
     PaymentTermDueDate type.
    :vartype terms: ~openmeter._generated.models.PaymentTermInstant or
     ~openmeter._generated.models.PaymentTermDueDate
    """

    terms: Optional["_types.PaymentTerms"] = rest_field(visibility=["read", "create", "update"])
    """The terms of payment for the invoice. Is either a PaymentTermInstant type or a
     PaymentTermDueDate type."""

    @overload
    def __init__(
        self,
        *,
        terms: Optional["_types.PaymentTerms"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class InvoicePendingLineCreate(_Model):
    """InvoicePendingLineCreate represents the create model for an invoice line that is sold to the
    customer based on usage.

    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar tax_config: Tax config specify the tax configuration for this line.
    :vartype tax_config: ~openmeter._generated.models.TaxConfig
    :ivar period: Period of the line item applies to for revenue recognition pruposes.

     Billing always treats periods as start being inclusive and end being exclusive. Required.
    :vartype period: ~openmeter._generated.models.Period
    :ivar invoice_at: The time this line item should be invoiced. Required.
    :vartype invoice_at: ~datetime.datetime
    :ivar price: Price of the usage-based item being sold. Is one of the following types:
     FlatPriceWithPaymentTerm, UnitPriceWithCommitments, TieredPriceWithCommitments,
     DynamicPriceWithCommitments, PackagePriceWithCommitments
    :vartype price: ~openmeter._generated.models.FlatPriceWithPaymentTerm or
     ~openmeter._generated.models.UnitPriceWithCommitments or
     ~openmeter._generated.models.TieredPriceWithCommitments or
     ~openmeter._generated.models.DynamicPriceWithCommitments or
     ~openmeter._generated.models.PackagePriceWithCommitments
    :ivar feature_key: The feature that the usage is based on.
    :vartype feature_key: str
    :ivar rate_card: The rate card that is used for this line.

     The rate card captures the intent of the price and discounts for the usage-based item.
    :vartype rate_card: ~openmeter._generated.models.InvoiceUsageBasedRateCard
    """

    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Display name. Required."""
    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    tax_config: Optional["_models.TaxConfig"] = rest_field(name="taxConfig", visibility=["read", "create", "update"])
    """Tax config specify the tax configuration for this line."""
    period: "_models.Period" = rest_field(visibility=["read", "create", "update"])
    """Period of the line item applies to for revenue recognition pruposes.
     
     Billing always treats periods as start being inclusive and end being exclusive. Required."""
    invoice_at: datetime.datetime = rest_field(
        name="invoiceAt", visibility=["read", "create", "update"], format="rfc3339"
    )
    """The time this line item should be invoiced. Required."""
    price: Optional["_types.RateCardUsageBasedPrice"] = rest_field(visibility=["read", "create", "update"])
    """Price of the usage-based item being sold. Is one of the following types:
     FlatPriceWithPaymentTerm, UnitPriceWithCommitments, TieredPriceWithCommitments,
     DynamicPriceWithCommitments, PackagePriceWithCommitments"""
    feature_key: Optional[str] = rest_field(name="featureKey", visibility=["read", "create", "update"])
    """The feature that the usage is based on."""
    rate_card: Optional["_models.InvoiceUsageBasedRateCard"] = rest_field(
        name="rateCard", visibility=["read", "create", "update"]
    )
    """The rate card that is used for this line.
     
     The rate card captures the intent of the price and discounts for the usage-based item."""

    @overload
    def __init__(
        self,
        *,
        name: str,
        period: "_models.Period",
        invoice_at: datetime.datetime,
        description: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
        tax_config: Optional["_models.TaxConfig"] = None,
        price: Optional["_types.RateCardUsageBasedPrice"] = None,
        feature_key: Optional[str] = None,
        rate_card: Optional["_models.InvoiceUsageBasedRateCard"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class InvoicePendingLineCreateInput(_Model):
    """InvoicePendingLineCreate represents the create model for a pending invoice line.

    :ivar currency: The currency of the lines to be created. Required.
    :vartype currency: str
    :ivar lines: The lines to be created. Required.
    :vartype lines: list[~openmeter._generated.models.InvoicePendingLineCreate]
    """

    currency: str = rest_field(visibility=["create"])
    """The currency of the lines to be created. Required."""
    lines: list["_models.InvoicePendingLineCreate"] = rest_field(visibility=["create"])
    """The lines to be created. Required."""

    @overload
    def __init__(
        self,
        *,
        currency: str,
        lines: list["_models.InvoicePendingLineCreate"],
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class InvoicePendingLineCreateResponse(_Model):
    """InvoicePendingLineCreateResponse represents the response from the create pending line endpoint.

    :ivar lines: The lines that were created. Required.
    :vartype lines: list[~openmeter._generated.models.InvoiceLine]
    :ivar invoice: The invoice containing the created lines. Required.
    :vartype invoice: ~openmeter._generated.models.Invoice
    :ivar is_invoice_new: Whether the invoice was newly created. Required.
    :vartype is_invoice_new: bool
    """

    lines: list["_models.InvoiceLine"] = rest_field(visibility=["read"])
    """The lines that were created. Required."""
    invoice: "_models.Invoice" = rest_field(visibility=["read"])
    """The invoice containing the created lines. Required."""
    is_invoice_new: bool = rest_field(name="isInvoiceNew", visibility=["read"])
    """Whether the invoice was newly created. Required."""


class InvoicePendingLinesActionFiltersInput(_Model):
    """InvoicePendingLinesActionFiltersInput specifies which lines to include in the invoice.

        :ivar line_ids: The pending line items to include in the invoice, if not provided:

         * all line items that have invoice_at < asOf will be included
         * [progressive billing only] all usage based line items will be included up to asOf, new
    usage-based line items will be staged for the rest of the billing cycle

    All lineIDs present in the list, must exists and must be invoicable as of asOf, or the action
        will fail.
        :vartype line_ids: list[str]
    """

    line_ids: Optional[list[str]] = rest_field(name="lineIds", visibility=["create"])
    """The pending line items to include in the invoice, if not provided:
 
      * all line items that have invoice_at < asOf will be included
      * [progressive billing only] all usage based line items will be included up to asOf, new
 usage-based line items will be staged for the rest of the billing cycle
 
 All lineIDs present in the list, must exists and must be invoicable as of asOf, or the action
     will fail."""

    @overload
    def __init__(
        self,
        *,
        line_ids: Optional[list[str]] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class InvoicePendingLinesActionInput(_Model):
    """BillingInvoiceActionInput is the input for creating an invoice.

    Invoice creation is always based on already pending line items created by the
    billingCreateLineByCustomer
    operation. Empty invoices are not allowed.

    :ivar filters: Filters to apply when creating the invoice.
    :vartype filters: ~openmeter._generated.models.InvoicePendingLinesActionFiltersInput
    :ivar as_of: The time as of which the invoice is created.

     If not provided, the current time is used.
    :vartype as_of: ~datetime.datetime
    :ivar customer_id: The customer ID for which to create the invoice. Required.
    :vartype customer_id: str
    :ivar progressive_billing_override: Override the progressive billing setting of the customer.

     Can be used to disable/enable progressive billing in case the business logic
     requires it, if not provided the billing profile's progressive billing setting will be used.
    :vartype progressive_billing_override: bool
    """

    filters: Optional["_models.InvoicePendingLinesActionFiltersInput"] = rest_field(visibility=["create"])
    """Filters to apply when creating the invoice."""
    as_of: Optional[datetime.datetime] = rest_field(name="asOf", visibility=["create"], format="rfc3339")
    """The time as of which the invoice is created.
     
     If not provided, the current time is used."""
    customer_id: str = rest_field(name="customerId", visibility=["create"])
    """The customer ID for which to create the invoice. Required."""
    progressive_billing_override: Optional[bool] = rest_field(name="progressiveBillingOverride", visibility=["create"])
    """Override the progressive billing setting of the customer.
     
     Can be used to disable/enable progressive billing in case the business logic
     requires it, if not provided the billing profile's progressive billing setting will be used."""

    @overload
    def __init__(
        self,
        *,
        customer_id: str,
        filters: Optional["_models.InvoicePendingLinesActionFiltersInput"] = None,
        as_of: Optional[datetime.datetime] = None,
        progressive_billing_override: Optional[bool] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class InvoiceReference(_Model):
    """Reference to an invoice.

    :ivar id: The ID of the invoice. Required.
    :vartype id: str
    :ivar number: The number of the invoice.
    :vartype number: str
    """

    id: str = rest_field(visibility=["read"])
    """The ID of the invoice. Required."""
    number: Optional[str] = rest_field(visibility=["read"])
    """The number of the invoice."""


class InvoiceReplaceUpdate(_Model):
    """InvoiceReplaceUpdate represents the update model for an invoice.

    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar supplier: The supplier of the lines included in the invoice. Required.
    :vartype supplier: ~openmeter._generated.models.BillingPartyReplaceUpdate
    :ivar customer: The customer the invoice is sent to. Required.
    :vartype customer: ~openmeter._generated.models.BillingPartyReplaceUpdate
    :ivar lines: The lines included in the invoice. Required.
    :vartype lines: list[~openmeter._generated.models.InvoiceLineReplaceUpdate]
    :ivar workflow: The workflow settings for the invoice. Required.
    :vartype workflow: ~openmeter._generated.models.InvoiceWorkflowReplaceUpdate
    """

    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    supplier: "_models.BillingPartyReplaceUpdate" = rest_field(visibility=["update"])
    """The supplier of the lines included in the invoice. Required."""
    customer: "_models.BillingPartyReplaceUpdate" = rest_field(visibility=["update"])
    """The customer the invoice is sent to. Required."""
    lines: list["_models.InvoiceLineReplaceUpdate"] = rest_field(visibility=["update"])
    """The lines included in the invoice. Required."""
    workflow: "_models.InvoiceWorkflowReplaceUpdate" = rest_field(visibility=["update"])
    """The workflow settings for the invoice. Required."""

    @overload
    def __init__(
        self,
        *,
        supplier: "_models.BillingPartyReplaceUpdate",
        customer: "_models.BillingPartyReplaceUpdate",
        lines: list["_models.InvoiceLineReplaceUpdate"],
        workflow: "_models.InvoiceWorkflowReplaceUpdate",
        description: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class InvoiceSimulationInput(_Model):
    """InvoiceSimulationInput is the input for simulating an invoice.

    :ivar number: The number of the invoice.
    :vartype number: str
    :ivar currency: Currency for all invoice line items.

     Multi currency invoices are not supported yet. Required.
    :vartype currency: str
    :ivar lines: Lines to be included in the generated invoice. Required.
    :vartype lines: list[~openmeter._generated.models.InvoiceSimulationLine]
    """

    number: Optional[str] = rest_field(visibility=["create"])
    """The number of the invoice."""
    currency: str = rest_field(visibility=["create"])
    """Currency for all invoice line items.
     
     Multi currency invoices are not supported yet. Required."""
    lines: list["_models.InvoiceSimulationLine"] = rest_field(visibility=["create"])
    """Lines to be included in the generated invoice. Required."""

    @overload
    def __init__(
        self,
        *,
        currency: str,
        lines: list["_models.InvoiceSimulationLine"],
        number: Optional[str] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class InvoiceSimulationLine(_Model):
    """InvoiceSimulationLine represents a usage-based line item that can be input to the simulation
    endpoint.

    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar tax_config: Tax config specify the tax configuration for this line.
    :vartype tax_config: ~openmeter._generated.models.TaxConfig
    :ivar period: Period of the line item applies to for revenue recognition pruposes.

     Billing always treats periods as start being inclusive and end being exclusive. Required.
    :vartype period: ~openmeter._generated.models.Period
    :ivar invoice_at: The time this line item should be invoiced. Required.
    :vartype invoice_at: ~datetime.datetime
    :ivar price: Price of the usage-based item being sold. Is one of the following types:
     FlatPriceWithPaymentTerm, UnitPriceWithCommitments, TieredPriceWithCommitments,
     DynamicPriceWithCommitments, PackagePriceWithCommitments
    :vartype price: ~openmeter._generated.models.FlatPriceWithPaymentTerm or
     ~openmeter._generated.models.UnitPriceWithCommitments or
     ~openmeter._generated.models.TieredPriceWithCommitments or
     ~openmeter._generated.models.DynamicPriceWithCommitments or
     ~openmeter._generated.models.PackagePriceWithCommitments
    :ivar feature_key: The feature that the usage is based on.
    :vartype feature_key: str
    :ivar rate_card: The rate card that is used for this line.

     The rate card captures the intent of the price and discounts for the usage-based item.
    :vartype rate_card: ~openmeter._generated.models.InvoiceUsageBasedRateCard
    :ivar quantity: The quantity of the item being sold. Required.
    :vartype quantity: str
    :ivar pre_line_period_quantity: The quantity of the item used before this line's period, if the
     line is billed progressively.
    :vartype pre_line_period_quantity: str
    :ivar id: ID of the line. If not specified it will be auto-generated.

     When discounts are specified, this must be provided, so that the discount can reference it.
    :vartype id: str
    """

    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Display name. Required."""
    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    tax_config: Optional["_models.TaxConfig"] = rest_field(name="taxConfig", visibility=["read", "create", "update"])
    """Tax config specify the tax configuration for this line."""
    period: "_models.Period" = rest_field(visibility=["read", "create", "update"])
    """Period of the line item applies to for revenue recognition pruposes.
     
     Billing always treats periods as start being inclusive and end being exclusive. Required."""
    invoice_at: datetime.datetime = rest_field(
        name="invoiceAt", visibility=["read", "create", "update"], format="rfc3339"
    )
    """The time this line item should be invoiced. Required."""
    price: Optional["_types.RateCardUsageBasedPrice"] = rest_field(visibility=["read", "create", "update"])
    """Price of the usage-based item being sold. Is one of the following types:
     FlatPriceWithPaymentTerm, UnitPriceWithCommitments, TieredPriceWithCommitments,
     DynamicPriceWithCommitments, PackagePriceWithCommitments"""
    feature_key: Optional[str] = rest_field(name="featureKey", visibility=["read", "create", "update"])
    """The feature that the usage is based on."""
    rate_card: Optional["_models.InvoiceUsageBasedRateCard"] = rest_field(
        name="rateCard", visibility=["read", "create", "update"]
    )
    """The rate card that is used for this line.
     
     The rate card captures the intent of the price and discounts for the usage-based item."""
    quantity: str = rest_field(visibility=["create"])
    """The quantity of the item being sold. Required."""
    pre_line_period_quantity: Optional[str] = rest_field(name="preLinePeriodQuantity", visibility=["create"])
    """The quantity of the item used before this line's period, if the line is billed progressively."""
    id: Optional[str] = rest_field(visibility=["create"])
    """ID of the line. If not specified it will be auto-generated.
     
     When discounts are specified, this must be provided, so that the discount can reference it."""

    @overload
    def __init__(
        self,
        *,
        name: str,
        period: "_models.Period",
        invoice_at: datetime.datetime,
        quantity: str,
        description: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
        tax_config: Optional["_models.TaxConfig"] = None,
        price: Optional["_types.RateCardUsageBasedPrice"] = None,
        feature_key: Optional[str] = None,
        rate_card: Optional["_models.InvoiceUsageBasedRateCard"] = None,
        pre_line_period_quantity: Optional[str] = None,
        id: Optional[str] = None,  # pylint: disable=redefined-builtin
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class InvoiceStatusDetails(_Model):
    """InvoiceStatusDetails represents the details of the invoice status.

    API users are encouraged to rely on the immutable/failed/avaliableActions fields to determine
    the next steps of the invoice instead of the extendedStatus field.

    :ivar immutable: Is the invoice editable?. Required.
    :vartype immutable: bool
    :ivar failed: Is the invoice in a failed state?. Required.
    :vartype failed: bool
    :ivar extended_status: Extended status information for the invoice. Required.
    :vartype extended_status: str
    :ivar available_actions: The actions that can be performed on the invoice. Required.
    :vartype available_actions: ~openmeter._generated.models.InvoiceAvailableActions
    """

    immutable: bool = rest_field(visibility=["read"])
    """Is the invoice editable?. Required."""
    failed: bool = rest_field(visibility=["read"])
    """Is the invoice in a failed state?. Required."""
    extended_status: str = rest_field(name="extendedStatus", visibility=["read"])
    """Extended status information for the invoice. Required."""
    available_actions: "_models.InvoiceAvailableActions" = rest_field(
        name="availableActions", visibility=["read", "create", "update", "delete", "query"]
    )
    """The actions that can be performed on the invoice. Required."""

    @overload
    def __init__(
        self,
        *,
        available_actions: "_models.InvoiceAvailableActions",
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class InvoiceTotals(_Model):
    """Totals contains the summaries of all calculations for the invoice.

    :ivar amount: The total value of the line before taxes, discounts and commitments. Required.
    :vartype amount: str
    :ivar charges_total: The amount of value of the line that are due to additional charges.
     Required.
    :vartype charges_total: str
    :ivar discounts_total: The amount of value of the line that are due to discounts. Required.
    :vartype discounts_total: str
    :ivar taxes_inclusive_total: The total amount of taxes that are included in the line. Required.
    :vartype taxes_inclusive_total: str
    :ivar taxes_exclusive_total: The total amount of taxes that are added on top of amount from the
     line. Required.
    :vartype taxes_exclusive_total: str
    :ivar taxes_total: The total amount of taxes for this line. Required.
    :vartype taxes_total: str
    :ivar total: The total amount value of the line after taxes, discounts and commitments.
     Required.
    :vartype total: str
    """

    amount: str = rest_field(visibility=["read"])
    """The total value of the line before taxes, discounts and commitments. Required."""
    charges_total: str = rest_field(name="chargesTotal", visibility=["read"])
    """The amount of value of the line that are due to additional charges. Required."""
    discounts_total: str = rest_field(name="discountsTotal", visibility=["read"])
    """The amount of value of the line that are due to discounts. Required."""
    taxes_inclusive_total: str = rest_field(name="taxesInclusiveTotal", visibility=["read"])
    """The total amount of taxes that are included in the line. Required."""
    taxes_exclusive_total: str = rest_field(name="taxesExclusiveTotal", visibility=["read"])
    """The total amount of taxes that are added on top of amount from the line. Required."""
    taxes_total: str = rest_field(name="taxesTotal", visibility=["read"])
    """The total amount of taxes for this line. Required."""
    total: str = rest_field(visibility=["read"])
    """The total amount value of the line after taxes, discounts and commitments. Required."""


class InvoiceUsageBasedRateCard(_Model):
    """InvoiceUsageBasedRateCard represents the rate card (intent) for an usage-based line.

    :ivar feature_key: Feature key.
    :vartype feature_key: str
    :ivar tax_config: Tax config.
    :vartype tax_config: ~openmeter._generated.models.TaxConfig
    :ivar price: The price of the rate card.
     When null, the feature or service is free. Required. Is one of the following types:
     FlatPriceWithPaymentTerm, UnitPriceWithCommitments, TieredPriceWithCommitments,
     DynamicPriceWithCommitments, PackagePriceWithCommitments
    :vartype price: ~openmeter._generated.models.FlatPriceWithPaymentTerm or
     ~openmeter._generated.models.UnitPriceWithCommitments or
     ~openmeter._generated.models.TieredPriceWithCommitments or
     ~openmeter._generated.models.DynamicPriceWithCommitments or
     ~openmeter._generated.models.PackagePriceWithCommitments
    :ivar discounts: The discounts that are applied to the line.
    :vartype discounts: ~openmeter._generated.models.BillingDiscounts
    """

    feature_key: Optional[str] = rest_field(name="featureKey", visibility=["read", "create", "update"])
    """Feature key."""
    tax_config: Optional["_models.TaxConfig"] = rest_field(name="taxConfig", visibility=["read", "create", "update"])
    """Tax config."""
    price: "_types.RateCardUsageBasedPrice" = rest_field(visibility=["read", "create", "update"])
    """The price of the rate card.
     When null, the feature or service is free. Required. Is one of the following types:
     FlatPriceWithPaymentTerm, UnitPriceWithCommitments, TieredPriceWithCommitments,
     DynamicPriceWithCommitments, PackagePriceWithCommitments"""
    discounts: Optional["_models.BillingDiscounts"] = rest_field(visibility=["read", "create", "update"])
    """The discounts that are applied to the line."""

    @overload
    def __init__(
        self,
        *,
        price: "_types.RateCardUsageBasedPrice",
        feature_key: Optional[str] = None,
        tax_config: Optional["_models.TaxConfig"] = None,
        discounts: Optional["_models.BillingDiscounts"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class InvoiceWorkflowInvoicingSettingsReplaceUpdate(_Model):  # pylint: disable=name-too-long
    """InvoiceWorkflowInvoicingSettingsReplaceUpdate represents the update model for the invoicing
    settings of an invoice workflow.

    :ivar auto_advance: Whether to automatically issue the invoice after the draftPeriod has
     passed.
    :vartype auto_advance: bool
    :ivar draft_period: The period for the invoice to be kept in draft status for manual reviews.
    :vartype draft_period: str
    :ivar due_after: The period after which the invoice is due.
     With some payment solutions it's only applicable for manual collection method.
    :vartype due_after: str
    :ivar default_tax_config: Default tax configuration to apply to the invoices.
    :vartype default_tax_config: ~openmeter._generated.models.TaxConfig
    """

    auto_advance: Optional[bool] = rest_field(name="autoAdvance", visibility=["read", "create", "update"])
    """Whether to automatically issue the invoice after the draftPeriod has passed."""
    draft_period: Optional[str] = rest_field(name="draftPeriod", visibility=["read", "create", "update"])
    """The period for the invoice to be kept in draft status for manual reviews."""
    due_after: Optional[str] = rest_field(name="dueAfter", visibility=["read", "create", "update"])
    """The period after which the invoice is due.
     With some payment solutions it's only applicable for manual collection method."""
    default_tax_config: Optional["_models.TaxConfig"] = rest_field(
        name="defaultTaxConfig", visibility=["read", "create", "update"]
    )
    """Default tax configuration to apply to the invoices."""

    @overload
    def __init__(
        self,
        *,
        auto_advance: Optional[bool] = None,
        draft_period: Optional[str] = None,
        due_after: Optional[str] = None,
        default_tax_config: Optional["_models.TaxConfig"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class InvoiceWorkflowReplaceUpdate(_Model):
    """InvoiceWorkflowReplaceUpdate represents the update model for an invoice workflow.

    Fields that are immutable a re removed from the model. This is based on
    InvoiceWorkflowSettings.

    :ivar workflow: The workflow used for this invoice. Required.
    :vartype workflow: ~openmeter._generated.models.InvoiceWorkflowSettingsReplaceUpdate
    """

    workflow: "_models.InvoiceWorkflowSettingsReplaceUpdate" = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """The workflow used for this invoice. Required."""

    @overload
    def __init__(
        self,
        *,
        workflow: "_models.InvoiceWorkflowSettingsReplaceUpdate",
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class InvoiceWorkflowSettings(_Model):
    """InvoiceWorkflowSettings represents the workflow settings used by the invoice.

    This is a clone of the billing profile's workflow settings at the time of invoice creation
    with customer overrides considered.

    :ivar apps: The apps that will be used to orchestrate the invoice's workflow. Is either a
     BillingProfileApps type or a BillingProfileAppReferences type.
    :vartype apps: ~openmeter._generated.models.BillingProfileApps or
     ~openmeter._generated.models.BillingProfileAppReferences
    :ivar source_billing_profile_id: sourceBillingProfileID is the billing profile on which the
     workflow was based on.

     The profile is snapshotted on invoice creation, after which it can be altered independently
     of the profile itself. Required.
    :vartype source_billing_profile_id: str
    :ivar workflow: The workflow details used by this invoice. Required.
    :vartype workflow: ~openmeter._generated.models.BillingWorkflow
    """

    apps: Optional["_types.BillingProfileAppsOrReference"] = rest_field(visibility=["read"])
    """The apps that will be used to orchestrate the invoice's workflow. Is either a
     BillingProfileApps type or a BillingProfileAppReferences type."""
    source_billing_profile_id: str = rest_field(name="sourceBillingProfileId", visibility=["read"])
    """sourceBillingProfileID is the billing profile on which the workflow was based on.
     
     The profile is snapshotted on invoice creation, after which it can be altered independently
     of the profile itself. Required."""
    workflow: "_models.BillingWorkflow" = rest_field(visibility=["read", "create", "update"])
    """The workflow details used by this invoice. Required."""

    @overload
    def __init__(
        self,
        *,
        workflow: "_models.BillingWorkflow",
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class InvoiceWorkflowSettingsReplaceUpdate(_Model):
    """Mutable workflow settings for an invoice.

    Other fields on the invoice's workflow are not mutable, they serve as a history of the
    invoice's workflow
    at creation time.

    :ivar invoicing: The invoicing settings for this workflow. Required.
    :vartype invoicing: ~openmeter._generated.models.InvoiceWorkflowInvoicingSettingsReplaceUpdate
    :ivar payment: The payment settings for this workflow. Required.
    :vartype payment: ~openmeter._generated.models.BillingWorkflowPaymentSettings
    """

    invoicing: "_models.InvoiceWorkflowInvoicingSettingsReplaceUpdate" = rest_field(visibility=["update"])
    """The invoicing settings for this workflow. Required."""
    payment: "_models.BillingWorkflowPaymentSettings" = rest_field(visibility=["update"])
    """The payment settings for this workflow. Required."""

    @overload
    def __init__(
        self,
        *,
        invoicing: "_models.InvoiceWorkflowInvoicingSettingsReplaceUpdate",
        payment: "_models.BillingWorkflowPaymentSettings",
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class IssueAfterReset(_Model):
    """Issue after reset.

    :ivar amount: Initial grant amount. Required.
    :vartype amount: float
    :ivar priority: Issue grant after reset priority.
    :vartype priority: int
    """

    amount: float = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Initial grant amount. Required."""
    priority: Optional[int] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Issue grant after reset priority."""

    @overload
    def __init__(
        self,
        *,
        amount: float,
        priority: Optional[int] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class ListRequestFilter(_Model):
    """ListRequestFilter.

    :ivar id:
    :vartype id: ~openmeter._generated.models.FilterString
    :ivar source:
    :vartype source: ~openmeter._generated.models.FilterString
    :ivar subject:
    :vartype subject: ~openmeter._generated.models.FilterString
    :ivar customer_id:
    :vartype customer_id: ~openmeter._generated.models.FilterIDExact
    :ivar type:
    :vartype type: ~openmeter._generated.models.FilterString
    :ivar time:
    :vartype time: ~openmeter._generated.models.FilterTime
    :ivar ingested_at:
    :vartype ingested_at: ~openmeter._generated.models.FilterTime
    """

    id: Optional["_models.FilterString"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    source: Optional["_models.FilterString"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    subject: Optional["_models.FilterString"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    customer_id: Optional["_models.FilterIDExact"] = rest_field(
        name="customerId", visibility=["read", "create", "update", "delete", "query"]
    )
    type: Optional["_models.FilterString"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    time: Optional["_models.FilterTime"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    ingested_at: Optional["_models.FilterTime"] = rest_field(
        name="ingestedAt", visibility=["read", "create", "update", "delete", "query"]
    )

    @overload
    def __init__(
        self,
        *,
        id: Optional["_models.FilterString"] = None,  # pylint: disable=redefined-builtin
        source: Optional["_models.FilterString"] = None,
        subject: Optional["_models.FilterString"] = None,
        customer_id: Optional["_models.FilterIDExact"] = None,
        type: Optional["_models.FilterString"] = None,
        time: Optional["_models.FilterTime"] = None,
        ingested_at: Optional["_models.FilterTime"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class MarketplaceInstallRequestPayload(_Model):
    """Marketplace install request payload.

    :ivar name: Name of the application to install.

     If name is not provided defaults to the marketplace listing's name.
    :vartype name: str
    :ivar create_billing_profile: If true, a billing profile will be created for the app.
     The Stripe app will be also set as the default billing profile if the current default is a
     Sandbox app.
    :vartype create_billing_profile: bool
    """

    name: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Name of the application to install.
     
     If name is not provided defaults to the marketplace listing's name."""
    create_billing_profile: Optional[bool] = rest_field(
        name="createBillingProfile", visibility=["read", "create", "update", "delete", "query"]
    )
    """If true, a billing profile will be created for the app.
     The Stripe app will be also set as the default billing profile if the current default is a
     Sandbox app."""

    @overload
    def __init__(
        self,
        *,
        name: Optional[str] = None,
        create_billing_profile: Optional[bool] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class MarketplaceInstallResponse(_Model):
    """Marketplace install response.

    :ivar app: Required. Is one of the following types: StripeApp, SandboxApp, CustomInvoicingApp
    :vartype app: ~openmeter._generated.models.StripeApp or ~openmeter._generated.models.SandboxApp
     or ~openmeter._generated.models.CustomInvoicingApp
    :ivar default_for_capability_types: Default for capabilities. Required.
    :vartype default_for_capability_types: list[str or ~openmeter.models.AppCapabilityType]
    """

    app: "_types.App" = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Required. Is one of the following types: StripeApp, SandboxApp, CustomInvoicingApp"""
    default_for_capability_types: list[Union[str, "_models.AppCapabilityType"]] = rest_field(
        name="defaultForCapabilityTypes", visibility=["read", "create", "update", "delete", "query"]
    )
    """Default for capabilities. Required."""

    @overload
    def __init__(
        self,
        *,
        app: "_types.App",
        default_for_capability_types: list[Union[str, "_models.AppCapabilityType"]],
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class MarketplaceListing(_Model):
    """A marketplace listing.
    Represent an available app in the app marketplace that can be installed to the organization.

    Marketplace apps only exist in config so they don't extend the Resource model.

    :ivar type: The app's type. Required. Known values are: "stripe", "sandbox", and
     "custom_invoicing".
    :vartype type: str or ~openmeter.models.AppType
    :ivar name: The app's name. Required.
    :vartype name: str
    :ivar description: The app's description. Required.
    :vartype description: str
    :ivar capabilities: The app's capabilities. Required.
    :vartype capabilities: list[~openmeter._generated.models.AppCapability]
    :ivar install_methods: Install methods.

     List of methods to install the app. Required.
    :vartype install_methods: list[str or ~openmeter.models.InstallMethod]
    """

    type: Union[str, "_models.AppType"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The app's type. Required. Known values are: \"stripe\", \"sandbox\", and \"custom_invoicing\"."""
    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The app's name. Required."""
    description: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The app's description. Required."""
    capabilities: list["_models.AppCapability"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The app's capabilities. Required."""
    install_methods: list[Union[str, "_models.InstallMethod"]] = rest_field(
        name="installMethods", visibility=["read", "create", "update", "delete", "query"]
    )
    """Install methods.
     
     List of methods to install the app. Required."""

    @overload
    def __init__(
        self,
        *,
        type: Union[str, "_models.AppType"],
        name: str,
        description: str,
        capabilities: list["_models.AppCapability"],
        install_methods: list[Union[str, "_models.InstallMethod"]],
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class MarketplaceListingPaginatedResponse(_Model):
    """Paginated response.

    :ivar total_count: The total number of items. Required.
    :vartype total_count: int
    :ivar page: The page index. Required.
    :vartype page: int
    :ivar page_size: The maximum number of items per page. Required.
    :vartype page_size: int
    :ivar items_property: The items in the current page. Required.
    :vartype items_property: list[~openmeter._generated.models.MarketplaceListing]
    """

    total_count: int = rest_field(name="totalCount", visibility=["read", "create", "update", "delete", "query"])
    """The total number of items. Required."""
    page: int = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The page index. Required."""
    page_size: int = rest_field(name="pageSize", visibility=["read", "create", "update", "delete", "query"])
    """The maximum number of items per page. Required."""
    items_property: list["_models.MarketplaceListing"] = rest_field(
        name="items", visibility=["read", "create", "update", "delete", "query"]
    )
    """The items in the current page. Required."""

    @overload
    def __init__(
        self,
        *,
        total_count: int,
        page: int,
        page_size: int,
        items_property: list["_models.MarketplaceListing"],
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class Metadata(_Model):
    """Set of key-value pairs.
    Metadata can be used to store additional information about a resource.

    """


class Meter(_Model):
    """A meter is a configuration that defines how to match and aggregate events.

    :ivar id: ID. Required.
    :vartype id: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar created_at: Creation Time. Required.
    :vartype created_at: ~datetime.datetime
    :ivar updated_at: Last Update Time. Required.
    :vartype updated_at: ~datetime.datetime
    :ivar deleted_at: Deletion Time.
    :vartype deleted_at: ~datetime.datetime
    :ivar name: Display name.
    :vartype name: str
    :ivar slug: A unique, human-readable identifier for the meter.
     Must consist only alphanumeric and underscore characters. Required.
    :vartype slug: str
    :ivar aggregation: The aggregation type to use for the meter. Required. Known values are:
     "SUM", "COUNT", "UNIQUE_COUNT", "AVG", "MIN", "MAX", and "LATEST".
    :vartype aggregation: str or ~openmeter.models.MeterAggregation
    :ivar event_type: The event type to aggregate. Required.
    :vartype event_type: str
    :ivar event_from: The date since the meter should include events.
     Useful to skip old events.
     If not specified, all historical events are included.
    :vartype event_from: ~datetime.datetime
    :ivar value_property: JSONPath expression to extract the value from the ingested event's data
     property.

     The ingested value for SUM, AVG, MIN, and MAX aggregations is a number or a string that can be
     parsed to a number.

     For UNIQUE_COUNT aggregation, the ingested value must be a string. For COUNT aggregation the
     valueProperty is ignored.
    :vartype value_property: str
    :ivar group_by: Named JSONPath expressions to extract the group by values from the event data.

     Keys must be unique and consist only alphanumeric and underscore characters.
    :vartype group_by: dict[str, str]
    :ivar annotations: Annotations.
    :vartype annotations: ~openmeter._generated.models.Annotations
    """

    id: str = rest_field(visibility=["read"])
    """ID. Required."""
    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    created_at: datetime.datetime = rest_field(name="createdAt", visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    updated_at: datetime.datetime = rest_field(name="updatedAt", visibility=["read"], format="rfc3339")
    """Last Update Time. Required."""
    deleted_at: Optional[datetime.datetime] = rest_field(name="deletedAt", visibility=["read"], format="rfc3339")
    """Deletion Time."""
    name: Optional[str] = rest_field(visibility=["read", "create", "update"])
    """Display name."""
    slug: str = rest_field(visibility=["read", "create"])
    """A unique, human-readable identifier for the meter.
     Must consist only alphanumeric and underscore characters. Required."""
    aggregation: Union[str, "_models.MeterAggregation"] = rest_field(visibility=["read", "create"])
    """The aggregation type to use for the meter. Required. Known values are: \"SUM\", \"COUNT\",
     \"UNIQUE_COUNT\", \"AVG\", \"MIN\", \"MAX\", and \"LATEST\"."""
    event_type: str = rest_field(name="eventType", visibility=["read", "create"])
    """The event type to aggregate. Required."""
    event_from: Optional[datetime.datetime] = rest_field(
        name="eventFrom", visibility=["read", "create"], format="rfc3339"
    )
    """The date since the meter should include events.
     Useful to skip old events.
     If not specified, all historical events are included."""
    value_property: Optional[str] = rest_field(name="valueProperty", visibility=["read", "create"])
    """JSONPath expression to extract the value from the ingested event's data property.
     
     The ingested value for SUM, AVG, MIN, and MAX aggregations is a number or a string that can be
     parsed to a number.
     
     For UNIQUE_COUNT aggregation, the ingested value must be a string. For COUNT aggregation the
     valueProperty is ignored."""
    group_by: Optional[dict[str, str]] = rest_field(name="groupBy", visibility=["read", "create", "update"])
    """Named JSONPath expressions to extract the group by values from the event data.
     
     Keys must be unique and consist only alphanumeric and underscore characters."""
    annotations: Optional["_models.Annotations"] = rest_field(visibility=["read"])
    """Annotations."""

    @overload
    def __init__(
        self,
        *,
        slug: str,
        aggregation: Union[str, "_models.MeterAggregation"],
        event_type: str,
        description: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
        name: Optional[str] = None,
        event_from: Optional[datetime.datetime] = None,
        value_property: Optional[str] = None,
        group_by: Optional[dict[str, str]] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class MeterCreate(_Model):
    """A meter create model.

    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar name: Display name.
    :vartype name: str
    :ivar slug: A unique, human-readable identifier for the meter.
     Must consist only alphanumeric and underscore characters. Required.
    :vartype slug: str
    :ivar aggregation: The aggregation type to use for the meter. Required. Known values are:
     "SUM", "COUNT", "UNIQUE_COUNT", "AVG", "MIN", "MAX", and "LATEST".
    :vartype aggregation: str or ~openmeter.models.MeterAggregation
    :ivar event_type: The event type to aggregate. Required.
    :vartype event_type: str
    :ivar event_from: The date since the meter should include events.
     Useful to skip old events.
     If not specified, all historical events are included.
    :vartype event_from: ~datetime.datetime
    :ivar value_property: JSONPath expression to extract the value from the ingested event's data
     property.

     The ingested value for SUM, AVG, MIN, and MAX aggregations is a number or a string that can be
     parsed to a number.

     For UNIQUE_COUNT aggregation, the ingested value must be a string. For COUNT aggregation the
     valueProperty is ignored.
    :vartype value_property: str
    :ivar group_by: Named JSONPath expressions to extract the group by values from the event data.

     Keys must be unique and consist only alphanumeric and underscore characters.
    :vartype group_by: dict[str, str]
    """

    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    name: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Display name."""
    slug: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """A unique, human-readable identifier for the meter.
     Must consist only alphanumeric and underscore characters. Required."""
    aggregation: Union[str, "_models.MeterAggregation"] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """The aggregation type to use for the meter. Required. Known values are: \"SUM\", \"COUNT\",
     \"UNIQUE_COUNT\", \"AVG\", \"MIN\", \"MAX\", and \"LATEST\"."""
    event_type: str = rest_field(name="eventType", visibility=["read", "create", "update", "delete", "query"])
    """The event type to aggregate. Required."""
    event_from: Optional[datetime.datetime] = rest_field(
        name="eventFrom", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The date since the meter should include events.
     Useful to skip old events.
     If not specified, all historical events are included."""
    value_property: Optional[str] = rest_field(
        name="valueProperty", visibility=["read", "create", "update", "delete", "query"]
    )
    """JSONPath expression to extract the value from the ingested event's data property.
     
     The ingested value for SUM, AVG, MIN, and MAX aggregations is a number or a string that can be
     parsed to a number.
     
     For UNIQUE_COUNT aggregation, the ingested value must be a string. For COUNT aggregation the
     valueProperty is ignored."""
    group_by: Optional[dict[str, str]] = rest_field(
        name="groupBy", visibility=["read", "create", "update", "delete", "query"]
    )
    """Named JSONPath expressions to extract the group by values from the event data.
     
     Keys must be unique and consist only alphanumeric and underscore characters."""

    @overload
    def __init__(
        self,
        *,
        slug: str,
        aggregation: Union[str, "_models.MeterAggregation"],
        event_type: str,
        description: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
        name: Optional[str] = None,
        event_from: Optional[datetime.datetime] = None,
        value_property: Optional[str] = None,
        group_by: Optional[dict[str, str]] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class MeterQueryRequest(_Model):
    """A meter query request.

    :ivar client_id: Client ID
     Useful to track progress of a query.
    :vartype client_id: str
    :ivar from_property: Start date-time in RFC 3339 format.

     Inclusive.
    :vartype from_property: ~datetime.datetime
    :ivar to: End date-time in RFC 3339 format.

     Inclusive.
    :vartype to: ~datetime.datetime
    :ivar window_size: If not specified, a single usage aggregate will be returned for the entirety
     of the specified period for each subject and group. Known values are: "MINUTE", "HOUR", "DAY",
     and "MONTH".
    :vartype window_size: str or ~openmeter.models.WindowSize
    :ivar window_time_zone: The value is the name of the time zone as defined in the IANA Time Zone
     Database (`http://www.iana.org/time-zones <http://www.iana.org/time-zones>`_).
     If not specified, the UTC timezone will be used.
    :vartype window_time_zone: str
    :ivar subject: Filtering by multiple subjects.
    :vartype subject: list[str]
    :ivar filter_customer_id: Filtering by multiple customers.
    :vartype filter_customer_id: list[str]
    :ivar filter_group_by: Simple filter for group bys with exact match.
    :vartype filter_group_by: dict[str, list[str]]
    :ivar advanced_meter_group_by_filters: Optional advanced meter group by filters.
     You can use this to filter for values of the meter groupBy fields.
    :vartype advanced_meter_group_by_filters: dict[str, ~openmeter._generated.models.FilterString]
    :ivar group_by: If not specified a single aggregate will be returned for each subject and time
     window.
     ``subject`` is a reserved group by value.
    :vartype group_by: list[str]
    """

    client_id: Optional[str] = rest_field(name="clientId", visibility=["read", "create", "update", "delete", "query"])
    """Client ID
     Useful to track progress of a query."""
    from_property: Optional[datetime.datetime] = rest_field(
        name="from", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """Start date-time in RFC 3339 format.
     
     Inclusive."""
    to: Optional[datetime.datetime] = rest_field(
        visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """End date-time in RFC 3339 format.
     
     Inclusive."""
    window_size: Optional[Union[str, "_models.WindowSize"]] = rest_field(
        name="windowSize", visibility=["read", "create", "update", "delete", "query"]
    )
    """If not specified, a single usage aggregate will be returned for the entirety of the specified
     period for each subject and group. Known values are: \"MINUTE\", \"HOUR\", \"DAY\", and
     \"MONTH\"."""
    window_time_zone: Optional[str] = rest_field(
        name="windowTimeZone", visibility=["read", "create", "update", "delete", "query"]
    )
    """The value is the name of the time zone as defined in the IANA Time Zone Database
     (`http://www.iana.org/time-zones <http://www.iana.org/time-zones>`_).
     If not specified, the UTC timezone will be used."""
    subject: Optional[list[str]] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Filtering by multiple subjects."""
    filter_customer_id: Optional[list[str]] = rest_field(
        name="filterCustomerId", visibility=["read", "create", "update", "delete", "query"]
    )
    """Filtering by multiple customers."""
    filter_group_by: Optional[dict[str, list[str]]] = rest_field(
        name="filterGroupBy", visibility=["read", "create", "update", "delete", "query"]
    )
    """Simple filter for group bys with exact match."""
    advanced_meter_group_by_filters: Optional[dict[str, "_models.FilterString"]] = rest_field(
        name="advancedMeterGroupByFilters", visibility=["read", "create", "update", "delete", "query"]
    )
    """Optional advanced meter group by filters.
     You can use this to filter for values of the meter groupBy fields."""
    group_by: Optional[list[str]] = rest_field(
        name="groupBy", visibility=["read", "create", "update", "delete", "query"]
    )
    """If not specified a single aggregate will be returned for each subject and time window.
     ``subject`` is a reserved group by value."""

    @overload
    def __init__(
        self,
        *,
        client_id: Optional[str] = None,
        from_property: Optional[datetime.datetime] = None,
        to: Optional[datetime.datetime] = None,
        window_size: Optional[Union[str, "_models.WindowSize"]] = None,
        window_time_zone: Optional[str] = None,
        subject: Optional[list[str]] = None,
        filter_customer_id: Optional[list[str]] = None,
        filter_group_by: Optional[dict[str, list[str]]] = None,
        advanced_meter_group_by_filters: Optional[dict[str, "_models.FilterString"]] = None,
        group_by: Optional[list[str]] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class MeterQueryResult(_Model):
    """The result of a meter query.

    :ivar from_property: The start of the period the usage is queried from.
     If not specified, the usage is queried from the beginning of time.
    :vartype from_property: ~datetime.datetime
    :ivar to: The end of the period the usage is queried to.
     If not specified, the usage is queried up to the current time.
    :vartype to: ~datetime.datetime
    :ivar window_size: The window size that the usage is aggregated.
     If not specified, the usage is aggregated over the entire period. Known values are: "MINUTE",
     "HOUR", "DAY", and "MONTH".
    :vartype window_size: str or ~openmeter.models.WindowSize
    :ivar data: The usage data.
     If no data is available, an empty array is returned. Required.
    :vartype data: list[~openmeter._generated.models.MeterQueryRow]
    """

    from_property: Optional[datetime.datetime] = rest_field(
        name="from", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The start of the period the usage is queried from.
     If not specified, the usage is queried from the beginning of time."""
    to: Optional[datetime.datetime] = rest_field(
        visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The end of the period the usage is queried to.
     If not specified, the usage is queried up to the current time."""
    window_size: Optional[Union[str, "_models.WindowSize"]] = rest_field(
        name="windowSize", visibility=["read", "create", "update", "delete", "query"]
    )
    """The window size that the usage is aggregated.
     If not specified, the usage is aggregated over the entire period. Known values are: \"MINUTE\",
     \"HOUR\", \"DAY\", and \"MONTH\"."""
    data: list["_models.MeterQueryRow"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The usage data.
     If no data is available, an empty array is returned. Required."""

    @overload
    def __init__(
        self,
        *,
        data: list["_models.MeterQueryRow"],
        from_property: Optional[datetime.datetime] = None,
        to: Optional[datetime.datetime] = None,
        window_size: Optional[Union[str, "_models.WindowSize"]] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class MeterQueryRow(_Model):
    """A row in the result of a meter query.

    :ivar value: The aggregated value. Required.
    :vartype value: float
    :ivar window_start: The start of the window the value is aggregated over. Required.
    :vartype window_start: ~datetime.datetime
    :ivar window_end: The end of the window the value is aggregated over. Required.
    :vartype window_end: ~datetime.datetime
    :ivar subject: The subject the value is aggregated over.
     If not specified, the value is aggregated over all subjects. Required.
    :vartype subject: str
    :ivar customer_id: The customer ID the value is aggregated over.
    :vartype customer_id: str
    :ivar group_by: The group by values the value is aggregated over. Required.
    :vartype group_by: dict[str, str]
    """

    value: float = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The aggregated value. Required."""
    window_start: datetime.datetime = rest_field(
        name="windowStart", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The start of the window the value is aggregated over. Required."""
    window_end: datetime.datetime = rest_field(
        name="windowEnd", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The end of the window the value is aggregated over. Required."""
    subject: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The subject the value is aggregated over.
     If not specified, the value is aggregated over all subjects. Required."""
    customer_id: Optional[str] = rest_field(
        name="customerId", visibility=["read", "create", "update", "delete", "query"]
    )
    """The customer ID the value is aggregated over."""
    group_by: dict[str, str] = rest_field(name="groupBy", visibility=["read", "create", "update", "delete", "query"])
    """The group by values the value is aggregated over. Required."""

    @overload
    def __init__(
        self,
        *,
        value: float,
        window_start: datetime.datetime,
        window_end: datetime.datetime,
        subject: str,
        group_by: dict[str, str],
        customer_id: Optional[str] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class MeterUpdate(_Model):
    """A meter update model.

    Only the properties that can be updated are included.
    For example, the slug and aggregation cannot be updated.

    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar name: Display name.
    :vartype name: str
    :ivar group_by: Named JSONPath expressions to extract the group by values from the event data.

     Keys must be unique and consist only alphanumeric and underscore characters.
    :vartype group_by: dict[str, str]
    """

    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    name: Optional[str] = rest_field(visibility=["read", "create", "update"])
    """Display name."""
    group_by: Optional[dict[str, str]] = rest_field(name="groupBy", visibility=["read", "create", "update"])
    """Named JSONPath expressions to extract the group by values from the event data.
     
     Keys must be unique and consist only alphanumeric and underscore characters."""

    @overload
    def __init__(
        self,
        *,
        description: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
        name: Optional[str] = None,
        group_by: Optional[dict[str, str]] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class MigrateRequest(_Model):
    """MigrateRequest.

    :ivar timing: Timing configuration for the migration, when the migration should take effect.
     If not supported by the subscription, 400 will be returned. Is either a Union[str,
     "_models.SubscriptionTimingEnum"] type or a datetime.datetime type.
    :vartype timing: str or ~openmeter.models.SubscriptionTimingEnum or ~datetime.datetime
    :ivar target_version: The version of the plan to migrate to.
     If not provided, the subscription will migrate to the latest version of the current plan.
    :vartype target_version: int
    :ivar starting_phase: The key of the phase to start the subscription in.
     If not provided, the subscription will start in the first phase of the plan.
    :vartype starting_phase: str
    :ivar billing_anchor: The billing anchor of the subscription. The provided date will be
     normalized according to the billing cadence to the nearest recurrence before start time. If not
     provided, the previous subscription billing anchor will be used.
    :vartype billing_anchor: ~datetime.datetime
    """

    timing: Optional["_types.SubscriptionTiming"] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """Timing configuration for the migration, when the migration should take effect.
     If not supported by the subscription, 400 will be returned. Is either a Union[str,
     \"_models.SubscriptionTimingEnum\"] type or a datetime.datetime type."""
    target_version: Optional[int] = rest_field(
        name="targetVersion", visibility=["read", "create", "update", "delete", "query"]
    )
    """The version of the plan to migrate to.
     If not provided, the subscription will migrate to the latest version of the current plan."""
    starting_phase: Optional[str] = rest_field(
        name="startingPhase", visibility=["read", "create", "update", "delete", "query"]
    )
    """The key of the phase to start the subscription in.
     If not provided, the subscription will start in the first phase of the plan."""
    billing_anchor: Optional[datetime.datetime] = rest_field(
        name="billingAnchor", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The billing anchor of the subscription. The provided date will be normalized according to the
     billing cadence to the nearest recurrence before start time. If not provided, the previous
     subscription billing anchor will be used."""

    @overload
    def __init__(
        self,
        *,
        timing: Optional["_types.SubscriptionTiming"] = None,
        target_version: Optional[int] = None,
        starting_phase: Optional[str] = None,
        billing_anchor: Optional[datetime.datetime] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class NotFoundProblemResponse(UnexpectedProblemResponse):
    """The origin server did not find a current representation for the target resource or is not
    willing to disclose that one exists.

    :ivar type: Type contains a URI that identifies the problem type. Required.
    :vartype type: str
    :ivar title: A a short, human-readable summary of the problem type. Required.
    :vartype title: str
    :ivar status: The HTTP status code generated by the origin server for this occurrence of the
     problem.
    :vartype status: int
    :ivar detail: A human-readable explanation specific to this occurrence of the problem.
     Required.
    :vartype detail: str
    :ivar instance: A URI reference that identifies the specific occurrence of the problem.
     Required.
    :vartype instance: str
    :ivar extensions: Additional properties specific to the problem type may be present.
    :vartype extensions: dict[str, any]
    """

    @overload
    def __init__(
        self,
        *,
        type: str,
        title: str,
        detail: str,
        instance: str,
        status: Optional[int] = None,
        extensions: Optional[dict[str, Any]] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class NotificationChannelMeta(_Model):
    """Metadata only fields of a notification channel.

    :ivar id: Channel Unique Identifier. Required.
    :vartype id: str
    :ivar type: Channel Type. Required. "WEBHOOK"
    :vartype type: str or ~openmeter.models.NotificationChannelType
    """

    id: str = rest_field(visibility=["read"])
    """Channel Unique Identifier. Required."""
    type: Union[str, "_models.NotificationChannelType"] = rest_field(visibility=["read", "create"])
    """Channel Type. Required. \"WEBHOOK\""""

    @overload
    def __init__(
        self,
        *,
        type: Union[str, "_models.NotificationChannelType"],
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class NotificationChannelPaginatedResponse(_Model):
    """Paginated response.

    :ivar total_count: The total number of items. Required.
    :vartype total_count: int
    :ivar page: The page index. Required.
    :vartype page: int
    :ivar page_size: The maximum number of items per page. Required.
    :vartype page_size: int
    :ivar items_property: The items in the current page. Required.
    :vartype items_property: list[~openmeter._generated.models.NotificationChannelWebhook]
    """

    total_count: int = rest_field(name="totalCount", visibility=["read", "create", "update", "delete", "query"])
    """The total number of items. Required."""
    page: int = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The page index. Required."""
    page_size: int = rest_field(name="pageSize", visibility=["read", "create", "update", "delete", "query"])
    """The maximum number of items per page. Required."""
    items_property: list["_types.NotificationChannel"] = rest_field(
        name="items", visibility=["read", "create", "update", "delete", "query"]
    )
    """The items in the current page. Required."""

    @overload
    def __init__(
        self,
        *,
        total_count: int,
        page: int,
        page_size: int,
        items_property: list["_types.NotificationChannel"],
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class NotificationChannelWebhook(_Model):
    """Notification channel with webhook type.

    :ivar created_at: Creation Time. Required.
    :vartype created_at: ~datetime.datetime
    :ivar updated_at: Last Update Time. Required.
    :vartype updated_at: ~datetime.datetime
    :ivar deleted_at: Deletion Time.
    :vartype deleted_at: ~datetime.datetime
    :ivar id: Channel Unique Identifier. Required.
    :vartype id: str
    :ivar type: Channel Type. Required.
    :vartype type: str or ~openmeter._generated.models.WEBHOOK
    :ivar name: Channel Name. Required.
    :vartype name: str
    :ivar disabled: Channel Disabled.
    :vartype disabled: bool
    :ivar annotations: Annotations.
    :vartype annotations: ~openmeter._generated.models.Annotations
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar url: Webhook URL. Required.
    :vartype url: str
    :ivar custom_headers: Custom HTTP Headers.
    :vartype custom_headers: dict[str, str]
    :ivar signing_secret: Signing Secret.
    :vartype signing_secret: str
    """

    created_at: datetime.datetime = rest_field(name="createdAt", visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    updated_at: datetime.datetime = rest_field(name="updatedAt", visibility=["read"], format="rfc3339")
    """Last Update Time. Required."""
    deleted_at: Optional[datetime.datetime] = rest_field(name="deletedAt", visibility=["read"], format="rfc3339")
    """Deletion Time."""
    id: str = rest_field(visibility=["read"])
    """Channel Unique Identifier. Required."""
    type: Literal[NotificationChannelType.WEBHOOK] = rest_field(visibility=["read", "create"])
    """Channel Type. Required."""
    name: str = rest_field(visibility=["read", "create", "update"])
    """Channel Name. Required."""
    disabled: Optional[bool] = rest_field(visibility=["read", "create", "update"])
    """Channel Disabled."""
    annotations: Optional["_models.Annotations"] = rest_field(visibility=["read"])
    """Annotations."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update"])
    """Metadata."""
    url: str = rest_field(visibility=["read", "create", "update"])
    """Webhook URL. Required."""
    custom_headers: Optional[dict[str, str]] = rest_field(name="customHeaders", visibility=["read", "create", "update"])
    """Custom HTTP Headers."""
    signing_secret: Optional[str] = rest_field(name="signingSecret", visibility=["read", "create", "update"])
    """Signing Secret."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[NotificationChannelType.WEBHOOK],
        name: str,
        url: str,
        disabled: Optional[bool] = None,
        metadata: Optional["_models.Metadata"] = None,
        custom_headers: Optional[dict[str, str]] = None,
        signing_secret: Optional[str] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class NotificationChannelWebhookCreateRequest(_Model):
    """Request with input parameters for creating new notification channel with webhook type.

    :ivar type: Channel Type. Required.
    :vartype type: str or ~openmeter._generated.models.WEBHOOK
    :ivar name: Channel Name. Required.
    :vartype name: str
    :ivar disabled: Channel Disabled.
    :vartype disabled: bool
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar url: Webhook URL. Required.
    :vartype url: str
    :ivar custom_headers: Custom HTTP Headers.
    :vartype custom_headers: dict[str, str]
    :ivar signing_secret: Signing Secret.
    :vartype signing_secret: str
    """

    type: Literal[NotificationChannelType.WEBHOOK] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """Channel Type. Required."""
    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Channel Name. Required."""
    disabled: Optional[bool] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Channel Disabled."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    url: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Webhook URL. Required."""
    custom_headers: Optional[dict[str, str]] = rest_field(
        name="customHeaders", visibility=["read", "create", "update", "delete", "query"]
    )
    """Custom HTTP Headers."""
    signing_secret: Optional[str] = rest_field(
        name="signingSecret", visibility=["read", "create", "update", "delete", "query"]
    )
    """Signing Secret."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[NotificationChannelType.WEBHOOK],
        name: str,
        url: str,
        disabled: Optional[bool] = None,
        metadata: Optional["_models.Metadata"] = None,
        custom_headers: Optional[dict[str, str]] = None,
        signing_secret: Optional[str] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class NotificationEvent(_Model):
    """Type of the notification event.

    :ivar id: Event Identifier. Required.
    :vartype id: str
    :ivar type: Event Type. Required. Known values are: "entitlements.balance.threshold",
     "entitlements.reset", "invoice.created", and "invoice.updated".
    :vartype type: str or ~openmeter.models.NotificationEventType
    :ivar created_at: Creation Time. Required.
    :vartype created_at: ~datetime.datetime
    :ivar rule: The nnotification rule which generated this event. Required. Is one of the
     following types: NotificationRuleBalanceThreshold, NotificationRuleEntitlementReset,
     NotificationRuleInvoiceCreated, NotificationRuleInvoiceUpdated
    :vartype rule: ~openmeter._generated.models.NotificationRuleBalanceThreshold or
     ~openmeter._generated.models.NotificationRuleEntitlementReset or
     ~openmeter._generated.models.NotificationRuleInvoiceCreated or
     ~openmeter._generated.models.NotificationRuleInvoiceUpdated
    :ivar delivery_status: Delivery Status. Required.
    :vartype delivery_status: list[~openmeter._generated.models.NotificationEventDeliveryStatus]
    :ivar payload: Timestamp when the notification event was created in RFC 3339 format. Required.
     Is one of the following types: NotificationEventResetPayload,
     NotificationEventBalanceThresholdPayload, NotificationEventInvoiceCreatedPayload,
     NotificationEventInvoiceUpdatedPayload
    :vartype payload: ~openmeter._generated.models.NotificationEventResetPayload or
     ~openmeter._generated.models.NotificationEventBalanceThresholdPayload or
     ~openmeter._generated.models.NotificationEventInvoiceCreatedPayload or
     ~openmeter._generated.models.NotificationEventInvoiceUpdatedPayload
    :ivar annotations: Annotations.
    :vartype annotations: ~openmeter._generated.models.Annotations
    """

    id: str = rest_field(visibility=["read"])
    """Event Identifier. Required."""
    type: Union[str, "_models.NotificationEventType"] = rest_field(visibility=["read"])
    """Event Type. Required. Known values are: \"entitlements.balance.threshold\",
     \"entitlements.reset\", \"invoice.created\", and \"invoice.updated\"."""
    created_at: datetime.datetime = rest_field(name="createdAt", visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    rule: "_types.NotificationRule" = rest_field(visibility=["read"])
    """The nnotification rule which generated this event. Required. Is one of the following types:
     NotificationRuleBalanceThreshold, NotificationRuleEntitlementReset,
     NotificationRuleInvoiceCreated, NotificationRuleInvoiceUpdated"""
    delivery_status: list["_models.NotificationEventDeliveryStatus"] = rest_field(
        name="deliveryStatus", visibility=["read"]
    )
    """Delivery Status. Required."""
    payload: "_types.NotificationEventPayload" = rest_field(visibility=["read"])
    """Timestamp when the notification event was created in RFC 3339 format. Required. Is one of the
     following types: NotificationEventResetPayload, NotificationEventBalanceThresholdPayload,
     NotificationEventInvoiceCreatedPayload, NotificationEventInvoiceUpdatedPayload"""
    annotations: Optional["_models.Annotations"] = rest_field(visibility=["read"])
    """Annotations."""


class NotificationEventBalanceThresholdPayload(_Model):
    """Payload for notification event with ``entitlements.balance.threshold`` type.

    :ivar id: Notification Event Identifier. Required.
    :vartype id: str
    :ivar type: Notification Event Type. Required.
    :vartype type: str or ~openmeter._generated.models.ENTITLEMENTS_BALANCE_THRESHOLD
    :ivar timestamp: Creation Time. Required.
    :vartype timestamp: ~datetime.datetime
    :ivar data: Payload Data. Required.
    :vartype data: ~openmeter._generated.models.NotificationEventBalanceThresholdPayloadData
    """

    id: str = rest_field(visibility=["read"])
    """Notification Event Identifier. Required."""
    type: Literal[NotificationEventType.ENTITLEMENTS_BALANCE_THRESHOLD] = rest_field(visibility=["read"])
    """Notification Event Type. Required."""
    timestamp: datetime.datetime = rest_field(visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    data: "_models.NotificationEventBalanceThresholdPayloadData" = rest_field(visibility=["read"])
    """Payload Data. Required."""


class NotificationEventBalanceThresholdPayloadData(_Model):  # pylint: disable=name-too-long
    """Data of the payload for notification event with ``entitlements.balance.threshold`` type.

    :ivar entitlement: Entitlement. Required.
    :vartype entitlement: ~openmeter._generated.models.EntitlementMetered
    :ivar feature: Feature. Required.
    :vartype feature: ~openmeter._generated.models.Feature
    :ivar subject: Subject. Required.
    :vartype subject: ~openmeter._generated.models.Subject
    :ivar value: Entitlement Value. Required.
    :vartype value: ~openmeter._generated.models.EntitlementValue
    :ivar customer: Customer.
    :vartype customer: ~openmeter._generated.models.Customer
    :ivar threshold: Threshold. Required.
    :vartype threshold: ~openmeter._generated.models.NotificationRuleBalanceThresholdValue
    """

    entitlement: "_models.EntitlementMetered" = rest_field(visibility=["read"])
    """Entitlement. Required."""
    feature: "_models.Feature" = rest_field(visibility=["read"])
    """Feature. Required."""
    subject: "_models.Subject" = rest_field(visibility=["read"])
    """Subject. Required."""
    value: "_models.EntitlementValue" = rest_field(visibility=["read"])
    """Entitlement Value. Required."""
    customer: Optional["_models.Customer"] = rest_field(visibility=["read"])
    """Customer."""
    threshold: "_models.NotificationRuleBalanceThresholdValue" = rest_field(visibility=["read"])
    """Threshold. Required."""


class NotificationEventDeliveryAttempt(_Model):
    """The delivery attempt of the notification event.

    :ivar state: State of teh delivery attempt. Required. Known values are: "SUCCESS", "FAILED",
     "SENDING", "PENDING", and "RESENDING".
    :vartype state: str or ~openmeter.models.NotificationEventDeliveryStatusState
    :ivar response: Response returned by the notification event recipient. Required.
    :vartype response: ~openmeter._generated.models.EventDeliveryAttemptResponse
    :ivar timestamp: Timestamp of the delivery attempt. Required.
    :vartype timestamp: ~datetime.datetime
    """

    state: Union[str, "_models.NotificationEventDeliveryStatusState"] = rest_field(visibility=["read"])
    """State of teh delivery attempt. Required. Known values are: \"SUCCESS\", \"FAILED\",
     \"SENDING\", \"PENDING\", and \"RESENDING\"."""
    response: "_models.EventDeliveryAttemptResponse" = rest_field(visibility=["read"])
    """Response returned by the notification event recipient. Required."""
    timestamp: datetime.datetime = rest_field(visibility=["read"], format="rfc3339")
    """Timestamp of the delivery attempt. Required."""


class NotificationEventDeliveryStatus(_Model):
    """The delivery status of the notification event.

    :ivar state: Delivery state of the notification event to the channel. Required. Known values
     are: "SUCCESS", "FAILED", "SENDING", "PENDING", and "RESENDING".
    :vartype state: str or ~openmeter.models.NotificationEventDeliveryStatusState
    :ivar reason: State Reason. Required.
    :vartype reason: str
    :ivar updated_at: Last Update Time. Required.
    :vartype updated_at: ~datetime.datetime
    :ivar channel: Notification Channel. Required.
    :vartype channel: ~openmeter._generated.models.NotificationChannelMeta
    :ivar annotations: Annotations.
    :vartype annotations: ~openmeter._generated.models.Annotations
    :ivar next_attempt: Timestamp of the next delivery attempt.
    :vartype next_attempt: ~datetime.datetime
    :ivar attempts: Delivery Attempts. Required.
    :vartype attempts: list[~openmeter._generated.models.NotificationEventDeliveryAttempt]
    """

    state: Union[str, "_models.NotificationEventDeliveryStatusState"] = rest_field(visibility=["read"])
    """Delivery state of the notification event to the channel. Required. Known values are:
     \"SUCCESS\", \"FAILED\", \"SENDING\", \"PENDING\", and \"RESENDING\"."""
    reason: str = rest_field(visibility=["read"])
    """State Reason. Required."""
    updated_at: datetime.datetime = rest_field(name="updatedAt", visibility=["read"], format="rfc3339")
    """Last Update Time. Required."""
    channel: "_models.NotificationChannelMeta" = rest_field(visibility=["read"])
    """Notification Channel. Required."""
    annotations: Optional["_models.Annotations"] = rest_field(visibility=["read"])
    """Annotations."""
    next_attempt: Optional[datetime.datetime] = rest_field(name="nextAttempt", visibility=["read"], format="rfc3339")
    """Timestamp of the next delivery attempt."""
    attempts: list["_models.NotificationEventDeliveryAttempt"] = rest_field(visibility=["read"])
    """Delivery Attempts. Required."""


class NotificationEventEntitlementValuePayloadBase(_Model):  # pylint: disable=name-too-long
    """Base data for any payload with entitlement entitlement value.

    :ivar entitlement: Entitlement. Required.
    :vartype entitlement: ~openmeter._generated.models.EntitlementMetered
    :ivar feature: Feature. Required.
    :vartype feature: ~openmeter._generated.models.Feature
    :ivar subject: Subject. Required.
    :vartype subject: ~openmeter._generated.models.Subject
    :ivar value: Entitlement Value. Required.
    :vartype value: ~openmeter._generated.models.EntitlementValue
    :ivar customer: Customer.
    :vartype customer: ~openmeter._generated.models.Customer
    """

    entitlement: "_models.EntitlementMetered" = rest_field(visibility=["read"])
    """Entitlement. Required."""
    feature: "_models.Feature" = rest_field(visibility=["read"])
    """Feature. Required."""
    subject: "_models.Subject" = rest_field(visibility=["read"])
    """Subject. Required."""
    value: "_models.EntitlementValue" = rest_field(visibility=["read"])
    """Entitlement Value. Required."""
    customer: Optional["_models.Customer"] = rest_field(visibility=["read"])
    """Customer."""


class NotificationEventInvoiceCreatedPayload(_Model):
    """Payload for notification event with ``invoice.created`` type.

    :ivar id: Notification Event Identifier. Required.
    :vartype id: str
    :ivar type: Notification Event Type. Required.
    :vartype type: str or ~openmeter._generated.models.INVOICE_CREATED
    :ivar timestamp: Creation Time. Required.
    :vartype timestamp: ~datetime.datetime
    :ivar data: Payload Data. Required.
    :vartype data: ~openmeter._generated.models.Invoice
    """

    id: str = rest_field(visibility=["read"])
    """Notification Event Identifier. Required."""
    type: Literal[NotificationEventType.INVOICE_CREATED] = rest_field(visibility=["read"])
    """Notification Event Type. Required."""
    timestamp: datetime.datetime = rest_field(visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    data: "_models.Invoice" = rest_field(visibility=["read"])
    """Payload Data. Required."""


class NotificationEventInvoiceUpdatedPayload(_Model):
    """Payload for notification event with ``invoice.updated`` type.

    :ivar id: Notification Event Identifier. Required.
    :vartype id: str
    :ivar type: Notification Event Type. Required.
    :vartype type: str or ~openmeter._generated.models.INVOICE_UPDATED
    :ivar timestamp: Creation Time. Required.
    :vartype timestamp: ~datetime.datetime
    :ivar data: Payload Data. Required.
    :vartype data: ~openmeter._generated.models.Invoice
    """

    id: str = rest_field(visibility=["read"])
    """Notification Event Identifier. Required."""
    type: Literal[NotificationEventType.INVOICE_UPDATED] = rest_field(visibility=["read"])
    """Notification Event Type. Required."""
    timestamp: datetime.datetime = rest_field(visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    data: "_models.Invoice" = rest_field(visibility=["read"])
    """Payload Data. Required."""


class NotificationEventPaginatedResponse(_Model):
    """Paginated response.

    :ivar total_count: The total number of items. Required.
    :vartype total_count: int
    :ivar page: The page index. Required.
    :vartype page: int
    :ivar page_size: The maximum number of items per page. Required.
    :vartype page_size: int
    :ivar items_property: The items in the current page. Required.
    :vartype items_property: list[~openmeter._generated.models.NotificationEvent]
    """

    total_count: int = rest_field(name="totalCount", visibility=["read", "create", "update", "delete", "query"])
    """The total number of items. Required."""
    page: int = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The page index. Required."""
    page_size: int = rest_field(name="pageSize", visibility=["read", "create", "update", "delete", "query"])
    """The maximum number of items per page. Required."""
    items_property: list["_models.NotificationEvent"] = rest_field(
        name="items", visibility=["read", "create", "update", "delete", "query"]
    )
    """The items in the current page. Required."""

    @overload
    def __init__(
        self,
        *,
        total_count: int,
        page: int,
        page_size: int,
        items_property: list["_models.NotificationEvent"],
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class NotificationEventResendRequest(_Model):
    """A notification event that will be re-sent.

    :ivar channels: Channels.
    :vartype channels: list[str]
    """

    channels: Optional[list[str]] = rest_field(visibility=["create"])
    """Channels."""

    @overload
    def __init__(
        self,
        *,
        channels: Optional[list[str]] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class NotificationEventResetPayload(_Model):
    """Payload for notification event with ``entitlements.reset`` type.

    :ivar id: Notification Event Identifier. Required.
    :vartype id: str
    :ivar type: Notification Event Type. Required.
    :vartype type: str or ~openmeter._generated.models.ENTITLEMENTS_RESET
    :ivar timestamp: Creation Time. Required.
    :vartype timestamp: ~datetime.datetime
    :ivar data: Payload Data. Required.
    :vartype data: ~openmeter._generated.models.NotificationEventEntitlementValuePayloadBase
    """

    id: str = rest_field(visibility=["read"])
    """Notification Event Identifier. Required."""
    type: Literal[NotificationEventType.ENTITLEMENTS_RESET] = rest_field(visibility=["read"])
    """Notification Event Type. Required."""
    timestamp: datetime.datetime = rest_field(visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    data: "_models.NotificationEventEntitlementValuePayloadBase" = rest_field(visibility=["read"])
    """Payload Data. Required."""


class NotificationRuleBalanceThreshold(_Model):
    """Notification rule with entitlements.balance.threshold type.

    :ivar created_at: Creation Time. Required.
    :vartype created_at: ~datetime.datetime
    :ivar updated_at: Last Update Time. Required.
    :vartype updated_at: ~datetime.datetime
    :ivar deleted_at: Deletion Time.
    :vartype deleted_at: ~datetime.datetime
    :ivar id: Rule Unique Identifier. Required.
    :vartype id: str
    :ivar type: Rule Type. Required.
    :vartype type: str or ~openmeter._generated.models.ENTITLEMENTS_BALANCE_THRESHOLD
    :ivar name: Rule Name. Required.
    :vartype name: str
    :ivar disabled: Rule Disabled.
    :vartype disabled: bool
    :ivar channels: Channels assigned to Rule. Required.
    :vartype channels: list[~openmeter._generated.models.NotificationChannelMeta]
    :ivar annotations: Annotations.
    :vartype annotations: ~openmeter._generated.models.Annotations
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar thresholds: Entitlement Balance Thresholds. Required.
    :vartype thresholds: list[~openmeter._generated.models.NotificationRuleBalanceThresholdValue]
    :ivar features: Features.
    :vartype features: list[~openmeter._generated.models.FeatureMeta]
    """

    created_at: datetime.datetime = rest_field(name="createdAt", visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    updated_at: datetime.datetime = rest_field(name="updatedAt", visibility=["read"], format="rfc3339")
    """Last Update Time. Required."""
    deleted_at: Optional[datetime.datetime] = rest_field(name="deletedAt", visibility=["read"], format="rfc3339")
    """Deletion Time."""
    id: str = rest_field(visibility=["read"])
    """Rule Unique Identifier. Required."""
    type: Literal[NotificationEventType.ENTITLEMENTS_BALANCE_THRESHOLD] = rest_field(
        visibility=["read", "create", "update"]
    )
    """Rule Type. Required."""
    name: str = rest_field(visibility=["read", "create", "update"])
    """Rule Name. Required."""
    disabled: Optional[bool] = rest_field(visibility=["read", "create", "update"])
    """Rule Disabled."""
    channels: list["_models.NotificationChannelMeta"] = rest_field(visibility=["read", "create", "update"])
    """Channels assigned to Rule. Required."""
    annotations: Optional["_models.Annotations"] = rest_field(visibility=["read"])
    """Annotations."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update"])
    """Metadata."""
    thresholds: list["_models.NotificationRuleBalanceThresholdValue"] = rest_field(
        visibility=["read", "create", "update"]
    )
    """Entitlement Balance Thresholds. Required."""
    features: Optional[list["_models.FeatureMeta"]] = rest_field(visibility=["read", "create", "update"])
    """Features."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[NotificationEventType.ENTITLEMENTS_BALANCE_THRESHOLD],
        name: str,
        channels: list["_models.NotificationChannelMeta"],
        thresholds: list["_models.NotificationRuleBalanceThresholdValue"],
        disabled: Optional[bool] = None,
        metadata: Optional["_models.Metadata"] = None,
        features: Optional[list["_models.FeatureMeta"]] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class NotificationRuleBalanceThresholdCreateRequest(_Model):  # pylint: disable=name-too-long
    """Request with input parameters for creating new notification rule with
    entitlements.balance.threshold type.

    :ivar type: Rule Type. Required.
    :vartype type: str or ~openmeter._generated.models.ENTITLEMENTS_BALANCE_THRESHOLD
    :ivar name: Rule Name. Required.
    :vartype name: str
    :ivar disabled: Rule Disabled.
    :vartype disabled: bool
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar thresholds: Entitlement Balance Thresholds. Required.
    :vartype thresholds: list[~openmeter._generated.models.NotificationRuleBalanceThresholdValue]
    :ivar channels: Channels. Required.
    :vartype channels: list[str]
    :ivar features: Features.
    :vartype features: list[str]
    """

    type: Literal[NotificationEventType.ENTITLEMENTS_BALANCE_THRESHOLD] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """Rule Type. Required."""
    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Rule Name. Required."""
    disabled: Optional[bool] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Rule Disabled."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    thresholds: list["_models.NotificationRuleBalanceThresholdValue"] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """Entitlement Balance Thresholds. Required."""
    channels: list[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Channels. Required."""
    features: Optional[list[str]] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Features."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[NotificationEventType.ENTITLEMENTS_BALANCE_THRESHOLD],
        name: str,
        thresholds: list["_models.NotificationRuleBalanceThresholdValue"],
        channels: list[str],
        disabled: Optional[bool] = None,
        metadata: Optional["_models.Metadata"] = None,
        features: Optional[list[str]] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class NotificationRuleBalanceThresholdValue(_Model):
    """Threshold value with multiple supported types.

    :ivar value: Threshold Value. Required.
    :vartype value: float
    :ivar type: Type of the threshold. Required. Known values are: "PERCENT", "NUMBER",
     "balance_value", "usage_percentage", and "usage_value".
    :vartype type: str or ~openmeter.models.NotificationRuleBalanceThresholdValueType
    """

    value: float = rest_field(visibility=["read", "create", "update"])
    """Threshold Value. Required."""
    type: Union[str, "_models.NotificationRuleBalanceThresholdValueType"] = rest_field(
        visibility=["read", "create", "update"]
    )
    """Type of the threshold. Required. Known values are: \"PERCENT\", \"NUMBER\", \"balance_value\",
     \"usage_percentage\", and \"usage_value\"."""

    @overload
    def __init__(
        self,
        *,
        value: float,
        type: Union[str, "_models.NotificationRuleBalanceThresholdValueType"],
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class NotificationRuleEntitlementReset(_Model):
    """Notification rule with entitlements.reset type.

    :ivar created_at: Creation Time. Required.
    :vartype created_at: ~datetime.datetime
    :ivar updated_at: Last Update Time. Required.
    :vartype updated_at: ~datetime.datetime
    :ivar deleted_at: Deletion Time.
    :vartype deleted_at: ~datetime.datetime
    :ivar id: Rule Unique Identifier. Required.
    :vartype id: str
    :ivar type: Rule Type. Required.
    :vartype type: str or ~openmeter._generated.models.ENTITLEMENTS_RESET
    :ivar name: Rule Name. Required.
    :vartype name: str
    :ivar disabled: Rule Disabled.
    :vartype disabled: bool
    :ivar channels: Channels assigned to Rule. Required.
    :vartype channels: list[~openmeter._generated.models.NotificationChannelMeta]
    :ivar annotations: Annotations.
    :vartype annotations: ~openmeter._generated.models.Annotations
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar features: Features.
    :vartype features: list[~openmeter._generated.models.FeatureMeta]
    """

    created_at: datetime.datetime = rest_field(name="createdAt", visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    updated_at: datetime.datetime = rest_field(name="updatedAt", visibility=["read"], format="rfc3339")
    """Last Update Time. Required."""
    deleted_at: Optional[datetime.datetime] = rest_field(name="deletedAt", visibility=["read"], format="rfc3339")
    """Deletion Time."""
    id: str = rest_field(visibility=["read"])
    """Rule Unique Identifier. Required."""
    type: Literal[NotificationEventType.ENTITLEMENTS_RESET] = rest_field(visibility=["read", "create", "update"])
    """Rule Type. Required."""
    name: str = rest_field(visibility=["read", "create", "update"])
    """Rule Name. Required."""
    disabled: Optional[bool] = rest_field(visibility=["read", "create", "update"])
    """Rule Disabled."""
    channels: list["_models.NotificationChannelMeta"] = rest_field(visibility=["read", "create", "update"])
    """Channels assigned to Rule. Required."""
    annotations: Optional["_models.Annotations"] = rest_field(visibility=["read"])
    """Annotations."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update"])
    """Metadata."""
    features: Optional[list["_models.FeatureMeta"]] = rest_field(visibility=["read", "create", "update"])
    """Features."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[NotificationEventType.ENTITLEMENTS_RESET],
        name: str,
        channels: list["_models.NotificationChannelMeta"],
        disabled: Optional[bool] = None,
        metadata: Optional["_models.Metadata"] = None,
        features: Optional[list["_models.FeatureMeta"]] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class NotificationRuleEntitlementResetCreateRequest(_Model):  # pylint: disable=name-too-long
    """Request with input parameters for creating new notification rule with entitlements.reset type.

    :ivar type: Rule Type. Required.
    :vartype type: str or ~openmeter._generated.models.ENTITLEMENTS_RESET
    :ivar name: Rule Name. Required.
    :vartype name: str
    :ivar disabled: Rule Disabled.
    :vartype disabled: bool
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar channels: Channels. Required.
    :vartype channels: list[str]
    :ivar features: Features.
    :vartype features: list[str]
    """

    type: Literal[NotificationEventType.ENTITLEMENTS_RESET] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """Rule Type. Required."""
    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Rule Name. Required."""
    disabled: Optional[bool] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Rule Disabled."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    channels: list[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Channels. Required."""
    features: Optional[list[str]] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Features."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[NotificationEventType.ENTITLEMENTS_RESET],
        name: str,
        channels: list[str],
        disabled: Optional[bool] = None,
        metadata: Optional["_models.Metadata"] = None,
        features: Optional[list[str]] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class NotificationRuleInvoiceCreated(_Model):
    """Notification rule with invoice.created type.

    :ivar created_at: Creation Time. Required.
    :vartype created_at: ~datetime.datetime
    :ivar updated_at: Last Update Time. Required.
    :vartype updated_at: ~datetime.datetime
    :ivar deleted_at: Deletion Time.
    :vartype deleted_at: ~datetime.datetime
    :ivar id: Rule Unique Identifier. Required.
    :vartype id: str
    :ivar type: Rule Type. Required.
    :vartype type: str or ~openmeter._generated.models.INVOICE_CREATED
    :ivar name: Rule Name. Required.
    :vartype name: str
    :ivar disabled: Rule Disabled.
    :vartype disabled: bool
    :ivar channels: Channels assigned to Rule. Required.
    :vartype channels: list[~openmeter._generated.models.NotificationChannelMeta]
    :ivar annotations: Annotations.
    :vartype annotations: ~openmeter._generated.models.Annotations
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    """

    created_at: datetime.datetime = rest_field(name="createdAt", visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    updated_at: datetime.datetime = rest_field(name="updatedAt", visibility=["read"], format="rfc3339")
    """Last Update Time. Required."""
    deleted_at: Optional[datetime.datetime] = rest_field(name="deletedAt", visibility=["read"], format="rfc3339")
    """Deletion Time."""
    id: str = rest_field(visibility=["read"])
    """Rule Unique Identifier. Required."""
    type: Literal[NotificationEventType.INVOICE_CREATED] = rest_field(visibility=["read", "create", "update"])
    """Rule Type. Required."""
    name: str = rest_field(visibility=["read", "create", "update"])
    """Rule Name. Required."""
    disabled: Optional[bool] = rest_field(visibility=["read", "create", "update"])
    """Rule Disabled."""
    channels: list["_models.NotificationChannelMeta"] = rest_field(visibility=["read", "create", "update"])
    """Channels assigned to Rule. Required."""
    annotations: Optional["_models.Annotations"] = rest_field(visibility=["read"])
    """Annotations."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update"])
    """Metadata."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[NotificationEventType.INVOICE_CREATED],
        name: str,
        channels: list["_models.NotificationChannelMeta"],
        disabled: Optional[bool] = None,
        metadata: Optional["_models.Metadata"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class NotificationRuleInvoiceCreatedCreateRequest(_Model):  # pylint: disable=name-too-long
    """Request with input parameters for creating new notification rule with invoice.created type.

    :ivar type: Rule Type. Required.
    :vartype type: str or ~openmeter._generated.models.INVOICE_CREATED
    :ivar name: Rule Name. Required.
    :vartype name: str
    :ivar disabled: Rule Disabled.
    :vartype disabled: bool
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar channels: Channels. Required.
    :vartype channels: list[str]
    """

    type: Literal[NotificationEventType.INVOICE_CREATED] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """Rule Type. Required."""
    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Rule Name. Required."""
    disabled: Optional[bool] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Rule Disabled."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    channels: list[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Channels. Required."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[NotificationEventType.INVOICE_CREATED],
        name: str,
        channels: list[str],
        disabled: Optional[bool] = None,
        metadata: Optional["_models.Metadata"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class NotificationRuleInvoiceUpdated(_Model):
    """Notification rule with invoice.updated type.

    :ivar created_at: Creation Time. Required.
    :vartype created_at: ~datetime.datetime
    :ivar updated_at: Last Update Time. Required.
    :vartype updated_at: ~datetime.datetime
    :ivar deleted_at: Deletion Time.
    :vartype deleted_at: ~datetime.datetime
    :ivar id: Rule Unique Identifier. Required.
    :vartype id: str
    :ivar type: Rule Type. Required.
    :vartype type: str or ~openmeter._generated.models.INVOICE_UPDATED
    :ivar name: Rule Name. Required.
    :vartype name: str
    :ivar disabled: Rule Disabled.
    :vartype disabled: bool
    :ivar channels: Channels assigned to Rule. Required.
    :vartype channels: list[~openmeter._generated.models.NotificationChannelMeta]
    :ivar annotations: Annotations.
    :vartype annotations: ~openmeter._generated.models.Annotations
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    """

    created_at: datetime.datetime = rest_field(name="createdAt", visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    updated_at: datetime.datetime = rest_field(name="updatedAt", visibility=["read"], format="rfc3339")
    """Last Update Time. Required."""
    deleted_at: Optional[datetime.datetime] = rest_field(name="deletedAt", visibility=["read"], format="rfc3339")
    """Deletion Time."""
    id: str = rest_field(visibility=["read"])
    """Rule Unique Identifier. Required."""
    type: Literal[NotificationEventType.INVOICE_UPDATED] = rest_field(visibility=["read", "create", "update"])
    """Rule Type. Required."""
    name: str = rest_field(visibility=["read", "create", "update"])
    """Rule Name. Required."""
    disabled: Optional[bool] = rest_field(visibility=["read", "create", "update"])
    """Rule Disabled."""
    channels: list["_models.NotificationChannelMeta"] = rest_field(visibility=["read", "create", "update"])
    """Channels assigned to Rule. Required."""
    annotations: Optional["_models.Annotations"] = rest_field(visibility=["read"])
    """Annotations."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update"])
    """Metadata."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[NotificationEventType.INVOICE_UPDATED],
        name: str,
        channels: list["_models.NotificationChannelMeta"],
        disabled: Optional[bool] = None,
        metadata: Optional["_models.Metadata"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class NotificationRuleInvoiceUpdatedCreateRequest(_Model):  # pylint: disable=name-too-long
    """Request with input parameters for creating new notification rule with invoice.updated  type.

    :ivar type: Rule Type. Required.
    :vartype type: str or ~openmeter._generated.models.INVOICE_UPDATED
    :ivar name: Rule Name. Required.
    :vartype name: str
    :ivar disabled: Rule Disabled.
    :vartype disabled: bool
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar channels: Channels. Required.
    :vartype channels: list[str]
    """

    type: Literal[NotificationEventType.INVOICE_UPDATED] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """Rule Type. Required."""
    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Rule Name. Required."""
    disabled: Optional[bool] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Rule Disabled."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    channels: list[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Channels. Required."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[NotificationEventType.INVOICE_UPDATED],
        name: str,
        channels: list[str],
        disabled: Optional[bool] = None,
        metadata: Optional["_models.Metadata"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class NotificationRulePaginatedResponse(_Model):
    """Paginated response.

    :ivar total_count: The total number of items. Required.
    :vartype total_count: int
    :ivar page: The page index. Required.
    :vartype page: int
    :ivar page_size: The maximum number of items per page. Required.
    :vartype page_size: int
    :ivar items_property: The items in the current page. Required.
    :vartype items_property: list[~openmeter._generated.models.NotificationRuleBalanceThreshold or
     ~openmeter._generated.models.NotificationRuleEntitlementReset or
     ~openmeter._generated.models.NotificationRuleInvoiceCreated or
     ~openmeter._generated.models.NotificationRuleInvoiceUpdated]
    """

    total_count: int = rest_field(name="totalCount", visibility=["read", "create", "update", "delete", "query"])
    """The total number of items. Required."""
    page: int = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The page index. Required."""
    page_size: int = rest_field(name="pageSize", visibility=["read", "create", "update", "delete", "query"])
    """The maximum number of items per page. Required."""
    items_property: list["_types.NotificationRule"] = rest_field(
        name="items", visibility=["read", "create", "update", "delete", "query"]
    )
    """The items in the current page. Required."""

    @overload
    def __init__(
        self,
        *,
        total_count: int,
        page: int,
        page_size: int,
        items_property: list["_types.NotificationRule"],
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class PackagePriceWithCommitments(_Model):
    """Package price with spend commitments.

    :ivar type: The type of the price. Required.
    :vartype type: str or ~openmeter._generated.models.PACKAGE
    :ivar amount: Amount. Required.
    :vartype amount: str
    :ivar quantity_per_package: Quantity per package. Required.
    :vartype quantity_per_package: str
    :ivar minimum_amount: Minimum amount.
    :vartype minimum_amount: str
    :ivar maximum_amount: Maximum amount.
    :vartype maximum_amount: str
    """

    type: Literal[PriceType.PACKAGE] = rest_field(visibility=["read", "create", "update"])
    """The type of the price. Required."""
    amount: str = rest_field(visibility=["read", "create", "update"])
    """Amount. Required."""
    quantity_per_package: str = rest_field(name="quantityPerPackage", visibility=["read", "create", "update"])
    """Quantity per package. Required."""
    minimum_amount: Optional[str] = rest_field(name="minimumAmount", visibility=["read", "create", "update"])
    """Minimum amount."""
    maximum_amount: Optional[str] = rest_field(name="maximumAmount", visibility=["read", "create", "update"])
    """Maximum amount."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[PriceType.PACKAGE],
        amount: str,
        quantity_per_package: str,
        minimum_amount: Optional[str] = None,
        maximum_amount: Optional[str] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class PaymentDueDate(_Model):
    """PaymentDueDate contains an amount that should be paid by the given date.

    :ivar due_at: When the payment is due. Required.
    :vartype due_at: ~datetime.datetime
    :ivar notes: Other details to take into account for the due date.
    :vartype notes: str
    :ivar amount: How much needs to be paid by the date. Required.
    :vartype amount: str
    :ivar percent: Percentage of the total that should be paid by the date.
    :vartype percent: float
    :ivar currency: If different from the parent document's base currency.
    :vartype currency: str
    """

    due_at: datetime.datetime = rest_field(name="dueAt", visibility=["read"], format="rfc3339")
    """When the payment is due. Required."""
    notes: Optional[str] = rest_field(visibility=["read"])
    """Other details to take into account for the due date."""
    amount: str = rest_field(visibility=["read"])
    """How much needs to be paid by the date. Required."""
    percent: Optional[float] = rest_field(visibility=["read"])
    """Percentage of the total that should be paid by the date."""
    currency: Optional[str] = rest_field(visibility=["read"])
    """If different from the parent document's base currency."""


class PaymentTermDueDate(_Model):
    """PaymentTermDueDate defines the terms for payment on a specific date.

    :ivar type: Type of terms to be applied. Required. Due on a specific date.
    :vartype type: str or ~openmeter._generated.models.DUE_DATE
    :ivar detail: Text detail of the chosen payment terms.
    :vartype detail: str
    :ivar notes: Description of the conditions for payment.
    :vartype notes: str
    :ivar due_at: When the payment is due. Required.
    :vartype due_at: list[~openmeter._generated.models.PaymentDueDate]
    """

    type: Literal[PaymentTermType.DUE_DATE] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Type of terms to be applied. Required. Due on a specific date."""
    detail: Optional[str] = rest_field(visibility=["read"])
    """Text detail of the chosen payment terms."""
    notes: Optional[str] = rest_field(visibility=["read"])
    """Description of the conditions for payment."""
    due_at: list["_models.PaymentDueDate"] = rest_field(name="dueAt", visibility=["read"])
    """When the payment is due. Required."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[PaymentTermType.DUE_DATE],
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class PaymentTermInstant(_Model):
    """PaymentTermInstant defines the terms for payment on receipt of invoice.

    :ivar type: Type of terms to be applied. Required. On receipt of invoice
    :vartype type: str or ~openmeter._generated.models.INSTANT
    :ivar detail: Text detail of the chosen payment terms.
    :vartype detail: str
    :ivar notes: Description of the conditions for payment.
    :vartype notes: str
    """

    type: Literal[PaymentTermType.INSTANT] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Type of terms to be applied. Required. On receipt of invoice"""
    detail: Optional[str] = rest_field(visibility=["read"])
    """Text detail of the chosen payment terms."""
    notes: Optional[str] = rest_field(visibility=["read"])
    """Description of the conditions for payment."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[PaymentTermType.INSTANT],
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class Period(_Model):
    """A period with a start and end time.

    :ivar from_property: Period start time. Required.
    :vartype from_property: ~datetime.datetime
    :ivar to: Period end time. Required.
    :vartype to: ~datetime.datetime
    """

    from_property: datetime.datetime = rest_field(
        name="from", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """Period start time. Required."""
    to: datetime.datetime = rest_field(visibility=["read", "create", "update", "delete", "query"], format="rfc3339")
    """Period end time. Required."""

    @overload
    def __init__(
        self,
        *,
        from_property: datetime.datetime,
        to: datetime.datetime,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class Plan(_Model):
    """Plans provide a template for subscriptions.

    :ivar id: ID. Required.
    :vartype id: str
    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar created_at: Creation Time. Required.
    :vartype created_at: ~datetime.datetime
    :ivar updated_at: Last Update Time. Required.
    :vartype updated_at: ~datetime.datetime
    :ivar deleted_at: Deletion Time.
    :vartype deleted_at: ~datetime.datetime
    :ivar key: Key. Required.
    :vartype key: str
    :ivar alignment: Alignment configuration for the plan.
    :vartype alignment: ~openmeter._generated.models.Alignment
    :ivar version: Version. Required.
    :vartype version: int
    :ivar currency: Currency. Required.
    :vartype currency: str
    :ivar billing_cadence: Billing cadence. Required.
    :vartype billing_cadence: ~datetime.timedelta
    :ivar pro_rating_config: Pro-rating configuration.
    :vartype pro_rating_config: ~openmeter._generated.models.ProRatingConfig
    :ivar effective_from: Effective start date.
    :vartype effective_from: ~datetime.datetime
    :ivar effective_to: Effective end date.
    :vartype effective_to: ~datetime.datetime
    :ivar status: Status. Required. Known values are: "draft", "active", "archived", and
     "scheduled".
    :vartype status: str or ~openmeter.models.PlanStatus
    :ivar phases: Plan phases. Required.
    :vartype phases: list[~openmeter._generated.models.PlanPhase]
    :ivar validation_errors: Validation errors. Required.
    :vartype validation_errors: list[~openmeter._generated.models.ValidationError]
    """

    id: str = rest_field(visibility=["read"])
    """ID. Required."""
    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Display name. Required."""
    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    created_at: datetime.datetime = rest_field(name="createdAt", visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    updated_at: datetime.datetime = rest_field(name="updatedAt", visibility=["read"], format="rfc3339")
    """Last Update Time. Required."""
    deleted_at: Optional[datetime.datetime] = rest_field(name="deletedAt", visibility=["read"], format="rfc3339")
    """Deletion Time."""
    key: str = rest_field(visibility=["read", "create"])
    """Key. Required."""
    alignment: Optional["_models.Alignment"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Alignment configuration for the plan."""
    version: int = rest_field(visibility=["read"])
    """Version. Required."""
    currency: str = rest_field(visibility=["read", "create"])
    """Currency. Required."""
    billing_cadence: datetime.timedelta = rest_field(name="billingCadence", visibility=["read", "create", "update"])
    """Billing cadence. Required."""
    pro_rating_config: Optional["_models.ProRatingConfig"] = rest_field(
        name="proRatingConfig", visibility=["read", "create", "update"]
    )
    """Pro-rating configuration."""
    effective_from: Optional[datetime.datetime] = rest_field(
        name="effectiveFrom", visibility=["read"], format="rfc3339"
    )
    """Effective start date."""
    effective_to: Optional[datetime.datetime] = rest_field(name="effectiveTo", visibility=["read"], format="rfc3339")
    """Effective end date."""
    status: Union[str, "_models.PlanStatus"] = rest_field(visibility=["read"])
    """Status. Required. Known values are: \"draft\", \"active\", \"archived\", and \"scheduled\"."""
    phases: list["_models.PlanPhase"] = rest_field(visibility=["read", "create", "update"])
    """Plan phases. Required."""
    validation_errors: list["_models.ValidationError"] = rest_field(name="validationErrors", visibility=["read"])
    """Validation errors. Required."""

    @overload
    def __init__(
        self,
        *,
        name: str,
        key: str,
        currency: str,
        billing_cadence: datetime.timedelta,
        phases: list["_models.PlanPhase"],
        description: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
        alignment: Optional["_models.Alignment"] = None,
        pro_rating_config: Optional["_models.ProRatingConfig"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class PlanAddon(_Model):
    """The PlanAddon describes the association between a plan and add-on.

    :ivar created_at: Creation Time. Required.
    :vartype created_at: ~datetime.datetime
    :ivar updated_at: Last Update Time. Required.
    :vartype updated_at: ~datetime.datetime
    :ivar deleted_at: Deletion Time.
    :vartype deleted_at: ~datetime.datetime
    :ivar annotations: Annotations.
    :vartype annotations: ~openmeter._generated.models.Annotations
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar addon: Addon. Required.
    :vartype addon: ~openmeter._generated.models.Addon
    :ivar from_plan_phase: The plan phase from the add-on becomes purchasable. Required.
    :vartype from_plan_phase: str
    :ivar max_quantity: Max quantity of the add-on.
    :vartype max_quantity: int
    :ivar validation_errors: Validation errors. Required.
    :vartype validation_errors: list[~openmeter._generated.models.ValidationError]
    """

    created_at: datetime.datetime = rest_field(name="createdAt", visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    updated_at: datetime.datetime = rest_field(name="updatedAt", visibility=["read"], format="rfc3339")
    """Last Update Time. Required."""
    deleted_at: Optional[datetime.datetime] = rest_field(name="deletedAt", visibility=["read"], format="rfc3339")
    """Deletion Time."""
    annotations: Optional["_models.Annotations"] = rest_field(visibility=["read"])
    """Annotations."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update"])
    """Metadata."""
    addon: "_models.Addon" = rest_field(visibility=["read"])
    """Addon. Required."""
    from_plan_phase: str = rest_field(name="fromPlanPhase", visibility=["read", "create", "update"])
    """The plan phase from the add-on becomes purchasable. Required."""
    max_quantity: Optional[int] = rest_field(name="maxQuantity", visibility=["read", "create", "update"])
    """Max quantity of the add-on."""
    validation_errors: list["_models.ValidationError"] = rest_field(name="validationErrors", visibility=["read"])
    """Validation errors. Required."""

    @overload
    def __init__(
        self,
        *,
        from_plan_phase: str,
        metadata: Optional["_models.Metadata"] = None,
        max_quantity: Optional[int] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class PlanAddonCreate(_Model):
    """A plan add-on assignment create request.

    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar from_plan_phase: The plan phase from the add-on becomes purchasable. Required.
    :vartype from_plan_phase: str
    :ivar max_quantity: Max quantity of the add-on.
    :vartype max_quantity: int
    :ivar addon_id: Add-on unique identifier. Required.
    :vartype addon_id: str
    """

    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    from_plan_phase: str = rest_field(name="fromPlanPhase", visibility=["read", "create", "update", "delete", "query"])
    """The plan phase from the add-on becomes purchasable. Required."""
    max_quantity: Optional[int] = rest_field(
        name="maxQuantity", visibility=["read", "create", "update", "delete", "query"]
    )
    """Max quantity of the add-on."""
    addon_id: str = rest_field(name="addonId", visibility=["read", "create", "update", "delete", "query"])
    """Add-on unique identifier. Required."""

    @overload
    def __init__(
        self,
        *,
        from_plan_phase: str,
        addon_id: str,
        metadata: Optional["_models.Metadata"] = None,
        max_quantity: Optional[int] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class PlanAddonPaginatedResponse(_Model):
    """Paginated response.

    :ivar total_count: The total number of items. Required.
    :vartype total_count: int
    :ivar page: The page index. Required.
    :vartype page: int
    :ivar page_size: The maximum number of items per page. Required.
    :vartype page_size: int
    :ivar items_property: The items in the current page. Required.
    :vartype items_property: list[~openmeter._generated.models.PlanAddon]
    """

    total_count: int = rest_field(name="totalCount", visibility=["read", "create", "update", "delete", "query"])
    """The total number of items. Required."""
    page: int = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The page index. Required."""
    page_size: int = rest_field(name="pageSize", visibility=["read", "create", "update", "delete", "query"])
    """The maximum number of items per page. Required."""
    items_property: list["_models.PlanAddon"] = rest_field(
        name="items", visibility=["read", "create", "update", "delete", "query"]
    )
    """The items in the current page. Required."""

    @overload
    def __init__(
        self,
        *,
        total_count: int,
        page: int,
        page_size: int,
        items_property: list["_models.PlanAddon"],
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class PlanAddonReplaceUpdate(_Model):
    """Resource update operation model.

    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar from_plan_phase: The plan phase from the add-on becomes purchasable. Required.
    :vartype from_plan_phase: str
    :ivar max_quantity: Max quantity of the add-on.
    :vartype max_quantity: int
    """

    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update"])
    """Metadata."""
    from_plan_phase: str = rest_field(name="fromPlanPhase", visibility=["read", "create", "update"])
    """The plan phase from the add-on becomes purchasable. Required."""
    max_quantity: Optional[int] = rest_field(name="maxQuantity", visibility=["read", "create", "update"])
    """Max quantity of the add-on."""

    @overload
    def __init__(
        self,
        *,
        from_plan_phase: str,
        metadata: Optional["_models.Metadata"] = None,
        max_quantity: Optional[int] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class PlanCreate(_Model):
    """Resource create operation model.

    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar key: Key. Required.
    :vartype key: str
    :ivar alignment: Alignment configuration for the plan.
    :vartype alignment: ~openmeter._generated.models.Alignment
    :ivar currency: Currency. Required.
    :vartype currency: str
    :ivar billing_cadence: Billing cadence. Required.
    :vartype billing_cadence: ~datetime.timedelta
    :ivar pro_rating_config: Pro-rating configuration.
    :vartype pro_rating_config: ~openmeter._generated.models.ProRatingConfig
    :ivar phases: Plan phases. Required.
    :vartype phases: list[~openmeter._generated.models.PlanPhase]
    """

    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Display name. Required."""
    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    key: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Key. Required."""
    alignment: Optional["_models.Alignment"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Alignment configuration for the plan."""
    currency: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Currency. Required."""
    billing_cadence: datetime.timedelta = rest_field(
        name="billingCadence", visibility=["read", "create", "update", "delete", "query"]
    )
    """Billing cadence. Required."""
    pro_rating_config: Optional["_models.ProRatingConfig"] = rest_field(
        name="proRatingConfig", visibility=["read", "create", "update", "delete", "query"]
    )
    """Pro-rating configuration."""
    phases: list["_models.PlanPhase"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Plan phases. Required."""

    @overload
    def __init__(
        self,
        *,
        name: str,
        key: str,
        currency: str,
        billing_cadence: datetime.timedelta,
        phases: list["_models.PlanPhase"],
        description: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
        alignment: Optional["_models.Alignment"] = None,
        pro_rating_config: Optional["_models.ProRatingConfig"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class PlanPhase(_Model):
    """The plan phase or pricing ramp allows changing a plan's rate cards over time as a subscription
    progresses.

    :ivar key: Key. Required.
    :vartype key: str
    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar duration: Duration. Required.
    :vartype duration: ~datetime.timedelta
    :ivar rate_cards: Rate cards. Required.
    :vartype rate_cards: list[~openmeter._generated.models.RateCardFlatFee or
     ~openmeter._generated.models.RateCardUsageBased]
    """

    key: str = rest_field(visibility=["read", "create"])
    """Key. Required."""
    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Display name. Required."""
    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    duration: datetime.timedelta = rest_field(visibility=["read", "create", "update"])
    """Duration. Required."""
    rate_cards: list["_types.RateCard"] = rest_field(name="rateCards", visibility=["read", "create", "update"])
    """Rate cards. Required."""

    @overload
    def __init__(
        self,
        *,
        key: str,
        name: str,
        duration: datetime.timedelta,
        rate_cards: list["_types.RateCard"],
        description: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class PlanReference(_Model):
    """References an exact plan.

    :ivar id: The plan ID. Required.
    :vartype id: str
    :ivar key: The plan key. Required.
    :vartype key: str
    :ivar version: The plan version. Required.
    :vartype version: int
    """

    id: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The plan ID. Required."""
    key: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The plan key. Required."""
    version: int = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The plan version. Required."""

    @overload
    def __init__(
        self,
        *,
        id: str,  # pylint: disable=redefined-builtin
        key: str,
        version: int,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class PlanReferenceInput(_Model):
    """References an exact plan defaulting to the current active version.

    :ivar key: The plan key. Required.
    :vartype key: str
    :ivar version: The plan version.
    :vartype version: int
    """

    key: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The plan key. Required."""
    version: Optional[int] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The plan version."""

    @overload
    def __init__(
        self,
        *,
        key: str,
        version: Optional[int] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class PlanReplaceUpdate(_Model):
    """Resource update operation model.

    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar alignment: Alignment configuration for the plan.
    :vartype alignment: ~openmeter._generated.models.Alignment
    :ivar billing_cadence: Billing cadence. Required.
    :vartype billing_cadence: ~datetime.timedelta
    :ivar pro_rating_config: Pro-rating configuration.
    :vartype pro_rating_config: ~openmeter._generated.models.ProRatingConfig
    :ivar phases: Plan phases. Required.
    :vartype phases: list[~openmeter._generated.models.PlanPhase]
    """

    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Display name. Required."""
    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    alignment: Optional["_models.Alignment"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Alignment configuration for the plan."""
    billing_cadence: datetime.timedelta = rest_field(name="billingCadence", visibility=["read", "create", "update"])
    """Billing cadence. Required."""
    pro_rating_config: Optional["_models.ProRatingConfig"] = rest_field(
        name="proRatingConfig", visibility=["read", "create", "update"]
    )
    """Pro-rating configuration."""
    phases: list["_models.PlanPhase"] = rest_field(visibility=["read", "create", "update"])
    """Plan phases. Required."""

    @overload
    def __init__(
        self,
        *,
        name: str,
        billing_cadence: datetime.timedelta,
        phases: list["_models.PlanPhase"],
        description: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
        alignment: Optional["_models.Alignment"] = None,
        pro_rating_config: Optional["_models.ProRatingConfig"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class PlanSubscriptionChange(_Model):
    """Change subscription based on plan.

    :ivar timing: Timing configuration for the change, when the change should take effect.
     For changing a subscription, the accepted values depend on the subscription configuration.
     Required. Is either a Union[str, "_models.SubscriptionTimingEnum"] type or a datetime.datetime
     type.
    :vartype timing: str or ~openmeter.models.SubscriptionTimingEnum or ~datetime.datetime
    :ivar alignment: What alignment settings the subscription should have.
    :vartype alignment: ~openmeter._generated.models.Alignment
    :ivar metadata: Arbitrary metadata associated with the subscription.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar plan: The plan reference to change to. Required.
    :vartype plan: ~openmeter._generated.models.PlanReferenceInput
    :ivar starting_phase: The key of the phase to start the subscription in.
     If not provided, the subscription will start in the first phase of the plan.
    :vartype starting_phase: str
    :ivar name: The name of the Subscription. If not provided the plan name is used.
    :vartype name: str
    :ivar description: Description for the Subscription.
    :vartype description: str
    :ivar billing_anchor: The billing anchor of the subscription. The provided date will be
     normalized according to the billing cadence to the nearest recurrence before start time. If not
     provided, the previous subscription billing anchor will be used.
    :vartype billing_anchor: ~datetime.datetime
    """

    timing: "_types.SubscriptionTiming" = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Timing configuration for the change, when the change should take effect.
     For changing a subscription, the accepted values depend on the subscription configuration.
     Required. Is either a Union[str, \"_models.SubscriptionTimingEnum\"] type or a
     datetime.datetime type."""
    alignment: Optional["_models.Alignment"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """What alignment settings the subscription should have."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Arbitrary metadata associated with the subscription."""
    plan: "_models.PlanReferenceInput" = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The plan reference to change to. Required."""
    starting_phase: Optional[str] = rest_field(
        name="startingPhase", visibility=["read", "create", "update", "delete", "query"]
    )
    """The key of the phase to start the subscription in.
     If not provided, the subscription will start in the first phase of the plan."""
    name: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The name of the Subscription. If not provided the plan name is used."""
    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description for the Subscription."""
    billing_anchor: Optional[datetime.datetime] = rest_field(
        name="billingAnchor", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The billing anchor of the subscription. The provided date will be normalized according to the
     billing cadence to the nearest recurrence before start time. If not provided, the previous
     subscription billing anchor will be used."""

    @overload
    def __init__(
        self,
        *,
        timing: "_types.SubscriptionTiming",
        plan: "_models.PlanReferenceInput",
        alignment: Optional["_models.Alignment"] = None,
        metadata: Optional["_models.Metadata"] = None,
        starting_phase: Optional[str] = None,
        name: Optional[str] = None,
        description: Optional[str] = None,
        billing_anchor: Optional[datetime.datetime] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class PlanSubscriptionCreate(_Model):
    """Create from plan.

    :ivar alignment: What alignment settings the subscription should have.
    :vartype alignment: ~openmeter._generated.models.Alignment
    :ivar metadata: Arbitrary metadata associated with the subscription.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar plan: The plan reference to change to. Required.
    :vartype plan: ~openmeter._generated.models.PlanReferenceInput
    :ivar starting_phase: The key of the phase to start the subscription in.
     If not provided, the subscription will start in the first phase of the plan.
    :vartype starting_phase: str
    :ivar name: The name of the Subscription. If not provided the plan name is used.
    :vartype name: str
    :ivar description: Description for the Subscription.
    :vartype description: str
    :ivar timing: Timing configuration for the change, when the change should take effect.
     The default is immediate. Is either a Union[str, "_models.SubscriptionTimingEnum"] type or a
     datetime.datetime type.
    :vartype timing: str or ~openmeter.models.SubscriptionTimingEnum or ~datetime.datetime
    :ivar customer_id: The ID of the customer. Provide either the key or ID. Has presedence over
     the key.
    :vartype customer_id: str
    :ivar customer_key: The key of the customer. Provide either the key or ID.
    :vartype customer_key: str
    :ivar billing_anchor: The billing anchor of the subscription. The provided date will be
     normalized according to the billing cadence to the nearest recurrence before start time. If not
     provided, the subscription start time will be used.
    :vartype billing_anchor: ~datetime.datetime
    """

    alignment: Optional["_models.Alignment"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """What alignment settings the subscription should have."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Arbitrary metadata associated with the subscription."""
    plan: "_models.PlanReferenceInput" = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The plan reference to change to. Required."""
    starting_phase: Optional[str] = rest_field(
        name="startingPhase", visibility=["read", "create", "update", "delete", "query"]
    )
    """The key of the phase to start the subscription in.
     If not provided, the subscription will start in the first phase of the plan."""
    name: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The name of the Subscription. If not provided the plan name is used."""
    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description for the Subscription."""
    timing: Optional["_types.SubscriptionTiming"] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """Timing configuration for the change, when the change should take effect.
     The default is immediate. Is either a Union[str, \"_models.SubscriptionTimingEnum\"] type or a
     datetime.datetime type."""
    customer_id: Optional[str] = rest_field(
        name="customerId", visibility=["read", "create", "update", "delete", "query"]
    )
    """The ID of the customer. Provide either the key or ID. Has presedence over the key."""
    customer_key: Optional[str] = rest_field(
        name="customerKey", visibility=["read", "create", "update", "delete", "query"]
    )
    """The key of the customer. Provide either the key or ID."""
    billing_anchor: Optional[datetime.datetime] = rest_field(
        name="billingAnchor", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The billing anchor of the subscription. The provided date will be normalized according to the
     billing cadence to the nearest recurrence before start time. If not provided, the subscription
     start time will be used."""

    @overload
    def __init__(
        self,
        *,
        plan: "_models.PlanReferenceInput",
        alignment: Optional["_models.Alignment"] = None,
        metadata: Optional["_models.Metadata"] = None,
        starting_phase: Optional[str] = None,
        name: Optional[str] = None,
        description: Optional[str] = None,
        timing: Optional["_types.SubscriptionTiming"] = None,
        customer_id: Optional[str] = None,
        customer_key: Optional[str] = None,
        billing_anchor: Optional[datetime.datetime] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class PortalToken(_Model):
    """A consumer portal token.

    Validator doesn't obey required for readOnly properties
    See: `https://github.com/stoplightio/spectral/issues/1274
    <https://github.com/stoplightio/spectral/issues/1274>`_.

    :ivar id:
    :vartype id: str
    :ivar subject: Required.
    :vartype subject: str
    :ivar expires_at:
    :vartype expires_at: ~datetime.datetime
    :ivar expired:
    :vartype expired: bool
    :ivar created_at:
    :vartype created_at: ~datetime.datetime
    :ivar token: The token is only returned at creation.
    :vartype token: str
    :ivar allowed_meter_slugs: Optional, if defined only the specified meters will be allowed.
    :vartype allowed_meter_slugs: list[str]
    """

    id: Optional[str] = rest_field(visibility=["read"])
    subject: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Required."""
    expires_at: Optional[datetime.datetime] = rest_field(name="expiresAt", visibility=["read"], format="rfc3339")
    expired: Optional[bool] = rest_field(visibility=["read"])
    created_at: Optional[datetime.datetime] = rest_field(name="createdAt", visibility=["read"], format="rfc3339")
    token: Optional[str] = rest_field(visibility=["read"])
    """The token is only returned at creation."""
    allowed_meter_slugs: Optional[list[str]] = rest_field(
        name="allowedMeterSlugs", visibility=["read", "create", "update", "delete", "query"]
    )
    """Optional, if defined only the specified meters will be allowed."""

    @overload
    def __init__(
        self,
        *,
        subject: str,
        allowed_meter_slugs: Optional[list[str]] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class PreconditionFailedProblemResponse(UnexpectedProblemResponse):
    """One or more conditions given in the request header fields evaluated to false when tested on the
    server.

    :ivar type: Type contains a URI that identifies the problem type. Required.
    :vartype type: str
    :ivar title: A a short, human-readable summary of the problem type. Required.
    :vartype title: str
    :ivar status: The HTTP status code generated by the origin server for this occurrence of the
     problem.
    :vartype status: int
    :ivar detail: A human-readable explanation specific to this occurrence of the problem.
     Required.
    :vartype detail: str
    :ivar instance: A URI reference that identifies the specific occurrence of the problem.
     Required.
    :vartype instance: str
    :ivar extensions: Additional properties specific to the problem type may be present.
    :vartype extensions: dict[str, any]
    """

    @overload
    def __init__(
        self,
        *,
        type: str,
        title: str,
        detail: str,
        instance: str,
        status: Optional[int] = None,
        extensions: Optional[dict[str, Any]] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class PriceTier(_Model):
    """A price tier.
    At least one price component is required in each tier.

    :ivar up_to_amount: Up to quantity.
    :vartype up_to_amount: str
    :ivar flat_price: Flat price component. Required.
    :vartype flat_price: ~openmeter._generated.models.FlatPrice
    :ivar unit_price: Unit price component. Required.
    :vartype unit_price: ~openmeter._generated.models.UnitPrice
    """

    up_to_amount: Optional[str] = rest_field(name="upToAmount", visibility=["read", "create", "update"])
    """Up to quantity."""
    flat_price: "_models.FlatPrice" = rest_field(name="flatPrice", visibility=["read", "create", "update"])
    """Flat price component. Required."""
    unit_price: "_models.UnitPrice" = rest_field(name="unitPrice", visibility=["read", "create", "update"])
    """Unit price component. Required."""

    @overload
    def __init__(
        self,
        *,
        flat_price: "_models.FlatPrice",
        unit_price: "_models.UnitPrice",
        up_to_amount: Optional[str] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class Progress(_Model):
    """Progress describes a progress of a task.

    :ivar success: Success is the number of items that succeeded. Required.
    :vartype success: int
    :ivar failed: Failed is the number of items that failed. Required.
    :vartype failed: int
    :ivar total: The total number of items to process. Required.
    :vartype total: int
    :ivar updated_at: The time the progress was last updated. Required.
    :vartype updated_at: ~datetime.datetime
    """

    success: int = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Success is the number of items that succeeded. Required."""
    failed: int = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Failed is the number of items that failed. Required."""
    total: int = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The total number of items to process. Required."""
    updated_at: datetime.datetime = rest_field(
        name="updatedAt", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The time the progress was last updated. Required."""

    @overload
    def __init__(
        self,
        *,
        success: int,
        failed: int,
        total: int,
        updated_at: datetime.datetime,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class ProRatingConfig(_Model):
    """Configuration for pro-rating behavior.

    :ivar enabled: Enable pro-rating. Required.
    :vartype enabled: bool
    :ivar mode: Pro-rating mode. Required. "prorate_prices"
    :vartype mode: str or ~openmeter.models.ProRatingMode
    """

    enabled: bool = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Enable pro-rating. Required."""
    mode: Union[str, "_models.ProRatingMode"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Pro-rating mode. Required. \"prorate_prices\""""

    @overload
    def __init__(
        self,
        *,
        enabled: bool,
        mode: Union[str, "_models.ProRatingMode"],
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class RateCardBooleanEntitlement(_Model):
    """Entitlement template of a boolean entitlement.

    :ivar metadata: Additional metadata for the feature.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar type: Required.
    :vartype type: str or ~openmeter._generated.models.BOOLEAN
    """

    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Additional metadata for the feature."""
    type: Literal[EntitlementType.BOOLEAN] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Required."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[EntitlementType.BOOLEAN],
        metadata: Optional["_models.Metadata"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class RateCardFlatFee(_Model):
    """A flat fee rate card defines a one-time purchase or a recurring fee.

    :ivar type: RateCard type. Required.
    :vartype type: str or ~openmeter._generated.models.FLAT_FEE
    :ivar key: Key. Required.
    :vartype key: str
    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar feature_key: Feature key.
    :vartype feature_key: str
    :ivar entitlement_template: The entitlement of the rate card.
     Only available when featureKey is set. Is one of the following types:
     RateCardMeteredEntitlement, RateCardStaticEntitlement, RateCardBooleanEntitlement
    :vartype entitlement_template: ~openmeter._generated.models.RateCardMeteredEntitlement or
     ~openmeter._generated.models.RateCardStaticEntitlement or
     ~openmeter._generated.models.RateCardBooleanEntitlement
    :ivar tax_config: Tax config.
    :vartype tax_config: ~openmeter._generated.models.TaxConfig
    :ivar billing_cadence: Billing cadence. Required.
    :vartype billing_cadence: ~datetime.timedelta
    :ivar price: Price. Required.
    :vartype price: ~openmeter._generated.models.FlatPriceWithPaymentTerm
    :ivar discounts: Discounts.
    :vartype discounts: ~openmeter._generated.models.Discounts
    """

    type: Literal[RateCardType.FLAT_FEE] = rest_field(visibility=["read", "create", "update"])
    """RateCard type. Required."""
    key: str = rest_field(visibility=["read", "create"])
    """Key. Required."""
    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Display name. Required."""
    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    feature_key: Optional[str] = rest_field(name="featureKey", visibility=["read", "create", "update"])
    """Feature key."""
    entitlement_template: Optional["_types.RateCardEntitlement"] = rest_field(
        name="entitlementTemplate", visibility=["read", "create", "update"]
    )
    """The entitlement of the rate card.
     Only available when featureKey is set. Is one of the following types:
     RateCardMeteredEntitlement, RateCardStaticEntitlement, RateCardBooleanEntitlement"""
    tax_config: Optional["_models.TaxConfig"] = rest_field(name="taxConfig", visibility=["read", "create", "update"])
    """Tax config."""
    billing_cadence: datetime.timedelta = rest_field(name="billingCadence", visibility=["read", "create", "update"])
    """Billing cadence. Required."""
    price: "_models.FlatPriceWithPaymentTerm" = rest_field(visibility=["read", "create", "update"])
    """Price. Required."""
    discounts: Optional["_models.Discounts"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Discounts."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[RateCardType.FLAT_FEE],
        key: str,
        name: str,
        billing_cadence: datetime.timedelta,
        price: "_models.FlatPriceWithPaymentTerm",
        description: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
        feature_key: Optional[str] = None,
        entitlement_template: Optional["_types.RateCardEntitlement"] = None,
        tax_config: Optional["_models.TaxConfig"] = None,
        discounts: Optional["_models.Discounts"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class RateCardMeteredEntitlement(_Model):
    """The entitlement template with a metered entitlement.

    :ivar metadata: Additional metadata for the feature.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar type: Required.
    :vartype type: str or ~openmeter._generated.models.METERED
    :ivar is_soft_limit: Soft limit.
    :vartype is_soft_limit: bool
    :ivar issue_after_reset: Initial grant amount.
    :vartype issue_after_reset: float
    :ivar issue_after_reset_priority: Issue grant after reset priority.
    :vartype issue_after_reset_priority: int
    :ivar preserve_overage_at_reset: Preserve overage at reset.
    :vartype preserve_overage_at_reset: bool
    :ivar usage_period: Usage Period.
    :vartype usage_period: ~datetime.timedelta
    """

    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Additional metadata for the feature."""
    type: Literal[EntitlementType.METERED] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Required."""
    is_soft_limit: Optional[bool] = rest_field(
        name="isSoftLimit", visibility=["read", "create", "update", "delete", "query"]
    )
    """Soft limit."""
    issue_after_reset: Optional[float] = rest_field(
        name="issueAfterReset", visibility=["read", "create", "update", "delete", "query"]
    )
    """Initial grant amount."""
    issue_after_reset_priority: Optional[int] = rest_field(
        name="issueAfterResetPriority", visibility=["read", "create", "update", "delete", "query"]
    )
    """Issue grant after reset priority."""
    preserve_overage_at_reset: Optional[bool] = rest_field(
        name="preserveOverageAtReset", visibility=["read", "create", "update", "delete", "query"]
    )
    """Preserve overage at reset."""
    usage_period: Optional[datetime.timedelta] = rest_field(name="usagePeriod", visibility=["read", "create", "update"])
    """Usage Period."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[EntitlementType.METERED],
        metadata: Optional["_models.Metadata"] = None,
        is_soft_limit: Optional[bool] = None,
        issue_after_reset: Optional[float] = None,
        issue_after_reset_priority: Optional[int] = None,
        preserve_overage_at_reset: Optional[bool] = None,
        usage_period: Optional[datetime.timedelta] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class RateCardStaticEntitlement(_Model):
    """Entitlement template of a static entitlement.

    :ivar metadata: Additional metadata for the feature.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar type: Required.
    :vartype type: str or ~openmeter._generated.models.STATIC
    :ivar config: The JSON parsable config of the entitlement. This value is also returned when
     checking entitlement access and it is useful for configuring fine-grained access settings to
     the feature, implemented in your own system. Has to be an object. Required.
    :vartype config: str
    """

    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Additional metadata for the feature."""
    type: Literal[EntitlementType.STATIC] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Required."""
    config: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The JSON parsable config of the entitlement. This value is also returned when checking
     entitlement access and it is useful for configuring fine-grained access settings to the
     feature, implemented in your own system. Has to be an object. Required."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[EntitlementType.STATIC],
        config: str,
        metadata: Optional["_models.Metadata"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class RateCardUsageBased(_Model):
    """A usage-based rate card defines a price based on usage.

    :ivar type: RateCard type. Required.
    :vartype type: str or ~openmeter._generated.models.USAGE_BASED
    :ivar key: Key. Required.
    :vartype key: str
    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar feature_key: Feature key.
    :vartype feature_key: str
    :ivar entitlement_template: The entitlement of the rate card.
     Only available when featureKey is set. Is one of the following types:
     RateCardMeteredEntitlement, RateCardStaticEntitlement, RateCardBooleanEntitlement
    :vartype entitlement_template: ~openmeter._generated.models.RateCardMeteredEntitlement or
     ~openmeter._generated.models.RateCardStaticEntitlement or
     ~openmeter._generated.models.RateCardBooleanEntitlement
    :ivar tax_config: Tax config.
    :vartype tax_config: ~openmeter._generated.models.TaxConfig
    :ivar billing_cadence: Billing cadence. Required.
    :vartype billing_cadence: ~datetime.timedelta
    :ivar price: The price of the rate card.
     When null, the feature or service is free. Required. Is one of the following types:
     FlatPriceWithPaymentTerm, UnitPriceWithCommitments, TieredPriceWithCommitments,
     DynamicPriceWithCommitments, PackagePriceWithCommitments
    :vartype price: ~openmeter._generated.models.FlatPriceWithPaymentTerm or
     ~openmeter._generated.models.UnitPriceWithCommitments or
     ~openmeter._generated.models.TieredPriceWithCommitments or
     ~openmeter._generated.models.DynamicPriceWithCommitments or
     ~openmeter._generated.models.PackagePriceWithCommitments
    :ivar discounts: Discounts.
    :vartype discounts: ~openmeter._generated.models.Discounts
    """

    type: Literal[RateCardType.USAGE_BASED] = rest_field(visibility=["read", "create", "update"])
    """RateCard type. Required."""
    key: str = rest_field(visibility=["read", "create"])
    """Key. Required."""
    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Display name. Required."""
    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    feature_key: Optional[str] = rest_field(name="featureKey", visibility=["read", "create", "update"])
    """Feature key."""
    entitlement_template: Optional["_types.RateCardEntitlement"] = rest_field(
        name="entitlementTemplate", visibility=["read", "create", "update"]
    )
    """The entitlement of the rate card.
     Only available when featureKey is set. Is one of the following types:
     RateCardMeteredEntitlement, RateCardStaticEntitlement, RateCardBooleanEntitlement"""
    tax_config: Optional["_models.TaxConfig"] = rest_field(name="taxConfig", visibility=["read", "create", "update"])
    """Tax config."""
    billing_cadence: datetime.timedelta = rest_field(name="billingCadence", visibility=["read", "create", "update"])
    """Billing cadence. Required."""
    price: "_types.RateCardUsageBasedPrice" = rest_field(visibility=["read", "create", "update"])
    """The price of the rate card.
     When null, the feature or service is free. Required. Is one of the following types:
     FlatPriceWithPaymentTerm, UnitPriceWithCommitments, TieredPriceWithCommitments,
     DynamicPriceWithCommitments, PackagePriceWithCommitments"""
    discounts: Optional["_models.Discounts"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Discounts."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[RateCardType.USAGE_BASED],
        key: str,
        name: str,
        billing_cadence: datetime.timedelta,
        price: "_types.RateCardUsageBasedPrice",
        description: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
        feature_key: Optional[str] = None,
        entitlement_template: Optional["_types.RateCardEntitlement"] = None,
        tax_config: Optional["_models.TaxConfig"] = None,
        discounts: Optional["_models.Discounts"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class RecurringPeriod(_Model):
    """Recurring period with an interval and an anchor.

    :ivar interval: Interval. Required. Is either a str type or a Union[str,
     "_models.RecurringPeriodIntervalEnum"] type.
    :vartype interval: str or str or ~openmeter.models.RecurringPeriodIntervalEnum
    :ivar anchor: Anchor time. Required.
    :vartype anchor: ~datetime.datetime
    :ivar interval_iso: The unit of time for the interval in ISO8601 format. Required.
    :vartype interval_iso: ~datetime.timedelta
    """

    interval: "_types.RecurringPeriodInterval" = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Interval. Required. Is either a str type or a Union[str,
     \"_models.RecurringPeriodIntervalEnum\"] type."""
    anchor: datetime.datetime = rest_field(visibility=["read", "create", "update", "delete", "query"], format="rfc3339")
    """Anchor time. Required."""
    interval_iso: datetime.timedelta = rest_field(
        name="intervalISO", visibility=["read", "create", "update", "delete", "query"]
    )
    """The unit of time for the interval in ISO8601 format. Required."""

    @overload
    def __init__(
        self,
        *,
        interval: "_types.RecurringPeriodInterval",
        anchor: datetime.datetime,
        interval_iso: datetime.timedelta,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class RecurringPeriodCreateInput(_Model):
    """Recurring period with an interval and an anchor.

    :ivar interval: Interval. Required. Is either a str type or a Union[str,
     "_models.RecurringPeriodIntervalEnum"] type.
    :vartype interval: str or str or ~openmeter.models.RecurringPeriodIntervalEnum
    :ivar anchor: Anchor time.
    :vartype anchor: ~datetime.datetime
    """

    interval: "_types.RecurringPeriodInterval" = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Interval. Required. Is either a str type or a Union[str,
     \"_models.RecurringPeriodIntervalEnum\"] type."""
    anchor: Optional[datetime.datetime] = rest_field(
        visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """Anchor time."""

    @overload
    def __init__(
        self,
        *,
        interval: "_types.RecurringPeriodInterval",
        anchor: Optional[datetime.datetime] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class RecurringPeriodV2(_Model):
    """Recurring period with an interval and an anchor.

    :ivar interval: Interval. Required. Is either a str type or a Union[str,
     "_models.RecurringPeriodIntervalEnum"] type.
    :vartype interval: str or str or ~openmeter.models.RecurringPeriodIntervalEnum
    :ivar anchor: Anchor time. Required.
    :vartype anchor: ~datetime.datetime
    """

    interval: "_types.RecurringPeriodInterval" = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Interval. Required. Is either a str type or a Union[str,
     \"_models.RecurringPeriodIntervalEnum\"] type."""
    anchor: datetime.datetime = rest_field(visibility=["read", "create", "update", "delete", "query"], format="rfc3339")
    """Anchor time. Required."""

    @overload
    def __init__(
        self,
        *,
        interval: "_types.RecurringPeriodInterval",
        anchor: datetime.datetime,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class ResetEntitlementUsageInput(_Model):
    """Reset parameters.

    :ivar effective_at: The time at which the reset takes effect, defaults to now. The reset cannot
     be in the future. The provided value is truncated to the minute due to how historical meter
     data is stored.
    :vartype effective_at: ~datetime.datetime
    :ivar retain_anchor: Determines whether the usage period anchor is retained or reset to the
    effectiveAt time.

     * If true, the usage period anchor is retained.
     * If false, the usage period anchor is reset to the effectiveAt time.
    :vartype retain_anchor: bool
    :ivar preserve_overage: Determines whether the overage is preserved or forgiven, overriding the
    entitlement's default behavior.

     * If true, the overage is preserved.
     * If false, the overage is forgiven.
    :vartype preserve_overage: bool
    """

    effective_at: Optional[datetime.datetime] = rest_field(
        name="effectiveAt", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The time at which the reset takes effect, defaults to now. The reset cannot be in the future.
     The provided value is truncated to the minute due to how historical meter data is stored."""
    retain_anchor: Optional[bool] = rest_field(
        name="retainAnchor", visibility=["read", "create", "update", "delete", "query"]
    )
    """Determines whether the usage period anchor is retained or reset to the effectiveAt time.
 
      * If true, the usage period anchor is retained.
      * If false, the usage period anchor is reset to the effectiveAt time."""
    preserve_overage: Optional[bool] = rest_field(
        name="preserveOverage", visibility=["read", "create", "update", "delete", "query"]
    )
    """Determines whether the overage is preserved or forgiven, overriding the entitlement's default
     behavior.
 
      * If true, the overage is preserved.
      * If false, the overage is forgiven."""

    @overload
    def __init__(
        self,
        *,
        effective_at: Optional[datetime.datetime] = None,
        retain_anchor: Optional[bool] = None,
        preserve_overage: Optional[bool] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class SandboxApp(_Model):
    """Sandbox app can be used for testing OpenMeter features.

    The app is not creating anything in external systems, thus it is safe to use for
    verifying OpenMeter features.

    :ivar id: ID. Required.
    :vartype id: str
    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar created_at: Creation Time. Required.
    :vartype created_at: ~datetime.datetime
    :ivar updated_at: Last Update Time. Required.
    :vartype updated_at: ~datetime.datetime
    :ivar deleted_at: Deletion Time.
    :vartype deleted_at: ~datetime.datetime
    :ivar listing: The marketplace listing that this installed app is based on. Required.
    :vartype listing: ~openmeter._generated.models.MarketplaceListing
    :ivar status: Status of the app connection. Required. Known values are: "ready" and
     "unauthorized".
    :vartype status: str or ~openmeter.models.AppStatus
    :ivar type: The app's type is Sandbox. Required.
    :vartype type: str or ~openmeter._generated.models.SANDBOX
    """

    id: str = rest_field(visibility=["read"])
    """ID. Required."""
    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Display name. Required."""
    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    created_at: datetime.datetime = rest_field(name="createdAt", visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    updated_at: datetime.datetime = rest_field(name="updatedAt", visibility=["read"], format="rfc3339")
    """Last Update Time. Required."""
    deleted_at: Optional[datetime.datetime] = rest_field(name="deletedAt", visibility=["read"], format="rfc3339")
    """Deletion Time."""
    listing: "_models.MarketplaceListing" = rest_field(visibility=["read"])
    """The marketplace listing that this installed app is based on. Required."""
    status: Union[str, "_models.AppStatus"] = rest_field(visibility=["read"])
    """Status of the app connection. Required. Known values are: \"ready\" and \"unauthorized\"."""
    type: Literal[AppType.SANDBOX] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The app's type is Sandbox. Required."""

    @overload
    def __init__(
        self,
        *,
        name: str,
        type: Literal[AppType.SANDBOX],
        description: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class SandboxAppReplaceUpdate(_Model):
    """Resource update operation model.

    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar type: The app's type is Sandbox. Required.
    :vartype type: str or ~openmeter._generated.models.SANDBOX
    """

    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Display name. Required."""
    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    type: Literal[AppType.SANDBOX] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The app's type is Sandbox. Required."""

    @overload
    def __init__(
        self,
        *,
        name: str,
        type: Literal[AppType.SANDBOX],
        description: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class SandboxCustomerAppData(_Model):
    """Sandbox Customer App Data.

    :ivar app: The installed sandbox app this data belongs to.
    :vartype app: ~openmeter._generated.models.SandboxApp
    :ivar id: App ID.
    :vartype id: str
    :ivar type: App Type. Required.
    :vartype type: str or ~openmeter._generated.models.SANDBOX
    """

    app: Optional["_models.SandboxApp"] = rest_field(visibility=["read"])
    """The installed sandbox app this data belongs to."""
    id: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """App ID."""
    type: Literal[AppType.SANDBOX] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """App Type. Required."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[AppType.SANDBOX],
        id: Optional[str] = None,  # pylint: disable=redefined-builtin
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class ServiceUnavailableProblemResponse(UnexpectedProblemResponse):
    """The server is currently unable to handle the request due to a temporary overload or scheduled
    maintenance, which will likely be alleviated after some delay.

    :ivar type: Type contains a URI that identifies the problem type. Required.
    :vartype type: str
    :ivar title: A a short, human-readable summary of the problem type. Required.
    :vartype title: str
    :ivar status: The HTTP status code generated by the origin server for this occurrence of the
     problem.
    :vartype status: int
    :ivar detail: A human-readable explanation specific to this occurrence of the problem.
     Required.
    :vartype detail: str
    :ivar instance: A URI reference that identifies the specific occurrence of the problem.
     Required.
    :vartype instance: str
    :ivar extensions: Additional properties specific to the problem type may be present.
    :vartype extensions: dict[str, any]
    """

    @overload
    def __init__(
        self,
        *,
        type: str,
        title: str,
        detail: str,
        instance: str,
        status: Optional[int] = None,
        extensions: Optional[dict[str, Any]] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class StripeAPIKeyInput(_Model):
    """The Stripe API key input.
    Used to authenticate with the Stripe API.

    :ivar secret_api_key: Required.
    :vartype secret_api_key: str
    """

    secret_api_key: str = rest_field(name="secretAPIKey", visibility=["read", "create", "update", "delete", "query"])
    """Required."""

    @overload
    def __init__(
        self,
        *,
        secret_api_key: str,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class StripeApp(_Model):
    """A installed Stripe app object.

    :ivar id: ID. Required.
    :vartype id: str
    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar created_at: Creation Time. Required.
    :vartype created_at: ~datetime.datetime
    :ivar updated_at: Last Update Time. Required.
    :vartype updated_at: ~datetime.datetime
    :ivar deleted_at: Deletion Time.
    :vartype deleted_at: ~datetime.datetime
    :ivar listing: The marketplace listing that this installed app is based on. Required.
    :vartype listing: ~openmeter._generated.models.MarketplaceListing
    :ivar status: Status of the app connection. Required. Known values are: "ready" and
     "unauthorized".
    :vartype status: str or ~openmeter.models.AppStatus
    :ivar type: The app's type is Stripe. Required.
    :vartype type: str or ~openmeter._generated.models.STRIPE
    :ivar stripe_account_id: The Stripe account ID. Required.
    :vartype stripe_account_id: str
    :ivar livemode: Livemode, true if the app is in production mode. Required.
    :vartype livemode: bool
    :ivar masked_api_key: The masked API key.
     Only shows the first 8 and last 3 characters. Required.
    :vartype masked_api_key: str
    :ivar secret_api_key: The Stripe API key.
    :vartype secret_api_key: str
    """

    id: str = rest_field(visibility=["read"])
    """ID. Required."""
    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Display name. Required."""
    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    created_at: datetime.datetime = rest_field(name="createdAt", visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    updated_at: datetime.datetime = rest_field(name="updatedAt", visibility=["read"], format="rfc3339")
    """Last Update Time. Required."""
    deleted_at: Optional[datetime.datetime] = rest_field(name="deletedAt", visibility=["read"], format="rfc3339")
    """Deletion Time."""
    listing: "_models.MarketplaceListing" = rest_field(visibility=["read"])
    """The marketplace listing that this installed app is based on. Required."""
    status: Union[str, "_models.AppStatus"] = rest_field(visibility=["read"])
    """Status of the app connection. Required. Known values are: \"ready\" and \"unauthorized\"."""
    type: Literal[AppType.STRIPE] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The app's type is Stripe. Required."""
    stripe_account_id: str = rest_field(name="stripeAccountId", visibility=["read"])
    """The Stripe account ID. Required."""
    livemode: bool = rest_field(visibility=["read"])
    """Livemode, true if the app is in production mode. Required."""
    masked_api_key: str = rest_field(name="maskedAPIKey", visibility=["read"])
    """The masked API key.
     Only shows the first 8 and last 3 characters. Required."""
    secret_api_key: Optional[str] = rest_field(name="secretAPIKey", visibility=["create", "update"])
    """The Stripe API key."""

    @overload
    def __init__(
        self,
        *,
        name: str,
        type: Literal[AppType.STRIPE],
        description: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
        secret_api_key: Optional[str] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class StripeAppReplaceUpdate(_Model):
    """Resource update operation model.

    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar type: The app's type is Stripe. Required.
    :vartype type: str or ~openmeter._generated.models.STRIPE
    :ivar secret_api_key: The Stripe API key.
    :vartype secret_api_key: str
    """

    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Display name. Required."""
    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    type: Literal[AppType.STRIPE] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The app's type is Stripe. Required."""
    secret_api_key: Optional[str] = rest_field(name="secretAPIKey", visibility=["create", "update"])
    """The Stripe API key."""

    @overload
    def __init__(
        self,
        *,
        name: str,
        type: Literal[AppType.STRIPE],
        description: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
        secret_api_key: Optional[str] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class StripeCustomerAppData(_Model):
    """Stripe Customer App Data.

    :ivar id: App ID.
    :vartype id: str
    :ivar type: App Type. Required.
    :vartype type: str or ~openmeter._generated.models.STRIPE
    :ivar stripe_customer_id: The Stripe customer ID. Required.
    :vartype stripe_customer_id: str
    :ivar stripe_default_payment_method_id: The Stripe default payment method ID.
    :vartype stripe_default_payment_method_id: str
    :ivar app: The installed stripe app this data belongs to.
    :vartype app: ~openmeter._generated.models.StripeApp
    """

    id: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """App ID."""
    type: Literal[AppType.STRIPE] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """App Type. Required."""
    stripe_customer_id: str = rest_field(
        name="stripeCustomerId", visibility=["read", "create", "update", "delete", "query"]
    )
    """The Stripe customer ID. Required."""
    stripe_default_payment_method_id: Optional[str] = rest_field(
        name="stripeDefaultPaymentMethodId", visibility=["read", "create", "update", "delete", "query"]
    )
    """The Stripe default payment method ID."""
    app: Optional["_models.StripeApp"] = rest_field(visibility=["read"])
    """The installed stripe app this data belongs to."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[AppType.STRIPE],
        stripe_customer_id: str,
        id: Optional[str] = None,  # pylint: disable=redefined-builtin
        stripe_default_payment_method_id: Optional[str] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class StripeCustomerAppDataBase(_Model):
    """Stripe Customer App Data Base.

    :ivar stripe_customer_id: The Stripe customer ID. Required.
    :vartype stripe_customer_id: str
    :ivar stripe_default_payment_method_id: The Stripe default payment method ID.
    :vartype stripe_default_payment_method_id: str
    """

    stripe_customer_id: str = rest_field(
        name="stripeCustomerId", visibility=["read", "create", "update", "delete", "query"]
    )
    """The Stripe customer ID. Required."""
    stripe_default_payment_method_id: Optional[str] = rest_field(
        name="stripeDefaultPaymentMethodId", visibility=["read", "create", "update", "delete", "query"]
    )
    """The Stripe default payment method ID."""

    @overload
    def __init__(
        self,
        *,
        stripe_customer_id: str,
        stripe_default_payment_method_id: Optional[str] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class StripeCustomerPortalSession(_Model):
    """Stripe customer portal session.

    See: `https://docs.stripe.com/api/customer_portal/sessions/object
    <https://docs.stripe.com/api/customer_portal/sessions/object>`_.

    :ivar id: The ID of the customer portal session.

     See: `https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-id
     <https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-id>`_.
     Required.
    :vartype id: str
    :ivar stripe_customer_id: The ID of the stripe customer. Required.
    :vartype stripe_customer_id: str
    :ivar configuration_id: Configuration used to customize the customer portal.

     See:
     `https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-configuration
     <https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-configuration>`_.
     Required.
    :vartype configuration_id: str
    :ivar livemode: Livemode.

     See:
     `https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-livemode
     <https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-livemode>`_.
     Required.
    :vartype livemode: bool
    :ivar created_at: Created at.

     See: `https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-created
     <https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-created>`_.
     Required.
    :vartype created_at: ~datetime.datetime
    :ivar return_url: Return URL.

     See:
     `https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-return_url
     <https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-return_url>`_.
     Required.
    :vartype return_url: str
    :ivar locale: Status.
       /**
     The IETF language tag of the locale customer portal is displayed in.

     See: `https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-locale
     <https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-locale>`_.
     Required.
    :vartype locale: str
    :ivar url: /**
     The ID of the customer.The URL to redirect the customer to after they have completed
     their requested actions. Required.
    :vartype url: str
    """

    id: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The ID of the customer portal session.
     
     See: `https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-id
     <https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-id>`_.
     Required."""
    stripe_customer_id: str = rest_field(
        name="stripeCustomerId", visibility=["read", "create", "update", "delete", "query"]
    )
    """The ID of the stripe customer. Required."""
    configuration_id: str = rest_field(
        name="configurationId", visibility=["read", "create", "update", "delete", "query"]
    )
    """Configuration used to customize the customer portal.
     
     See:
     `https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-configuration
     <https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-configuration>`_.
     Required."""
    livemode: bool = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Livemode.
     
     See:
     `https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-livemode
     <https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-livemode>`_.
     Required."""
    created_at: datetime.datetime = rest_field(
        name="createdAt", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """Created at.
     
     See: `https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-created
     <https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-created>`_.
     Required."""
    return_url: str = rest_field(name="returnUrl", visibility=["read", "create", "update", "delete", "query"])
    """Return URL.
     
     See:
     `https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-return_url
     <https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-return_url>`_.
     Required."""
    locale: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Status.
       /**
     The IETF language tag of the locale customer portal is displayed in.
     
     See: `https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-locale
     <https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-locale>`_.
     Required."""
    url: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """/**
     The ID of the customer.The URL to redirect the customer to after they have completed
     their requested actions. Required."""

    @overload
    def __init__(
        self,
        *,
        id: str,  # pylint: disable=redefined-builtin
        stripe_customer_id: str,
        configuration_id: str,
        livemode: bool,
        created_at: datetime.datetime,
        return_url: str,
        locale: str,
        url: str,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class StripeTaxConfig(_Model):
    """The tax config for Stripe.

    :ivar code: Tax code. Required.
    :vartype code: str
    """

    code: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Tax code. Required."""

    @overload
    def __init__(
        self,
        *,
        code: str,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class StripeWebhookEvent(_Model):
    """Stripe webhook event.

    :ivar id: The event ID. Required.
    :vartype id: str
    :ivar type: The event type. Required.
    :vartype type: str
    :ivar livemode: Live mode. Required.
    :vartype livemode: bool
    :ivar created: The event created timestamp. Required.
    :vartype created: int
    :ivar data: The event data. Required.
    :vartype data: ~openmeter._generated.models.StripeWebhookEventData
    """

    id: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The event ID. Required."""
    type: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The event type. Required."""
    livemode: bool = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Live mode. Required."""
    created: int = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The event created timestamp. Required."""
    data: "_models.StripeWebhookEventData" = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The event data. Required."""

    @overload
    def __init__(
        self,
        *,
        id: str,  # pylint: disable=redefined-builtin
        type: str,
        livemode: bool,
        created: int,
        data: "_models.StripeWebhookEventData",
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class StripeWebhookEventData(_Model):
    """StripeWebhookEventData.

    :ivar object: Required.
    :vartype object: any
    """

    object: Any = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Required."""

    @overload
    def __init__(
        self,
        *,
        object: Any,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class StripeWebhookResponse(_Model):
    """Stripe webhook response.

    :ivar namespace_id: Required.
    :vartype namespace_id: str
    :ivar app_id: Required.
    :vartype app_id: str
    :ivar customer_id:
    :vartype customer_id: str
    :ivar message:
    :vartype message: str
    """

    namespace_id: str = rest_field(name="namespaceId", visibility=["read", "create", "update", "delete", "query"])
    """Required."""
    app_id: str = rest_field(name="appId", visibility=["read", "create", "update", "delete", "query"])
    """Required."""
    customer_id: Optional[str] = rest_field(
        name="customerId", visibility=["read", "create", "update", "delete", "query"]
    )
    message: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])

    @overload
    def __init__(
        self,
        *,
        namespace_id: str,
        app_id: str,
        customer_id: Optional[str] = None,
        message: Optional[str] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class Subject(_Model):
    """A subject is a unique identifier for a usage attribution by its key.
    Subjects only exist in the concept of metering.
    Subjects are optional to create and work as an enrichment for the subject key like displayName,
    metadata, etc.
    Subjects are useful when you are reporting usage events with your own database ID but want to
    enrich the subject with a human-readable name or metadata.
    For most use cases, a subject is equivalent to a customer.

    ⚠️ **Deprecated**: Subjects as managable entities are being depracated, use customers with
    subject key usage attribution instead.

    :ivar created_at: Creation Time. Required.
    :vartype created_at: ~datetime.datetime
    :ivar updated_at: Last Update Time. Required.
    :vartype updated_at: ~datetime.datetime
    :ivar deleted_at: Deletion Time.
    :vartype deleted_at: ~datetime.datetime
    :ivar id: A unique identifier for the subject. Required.
    :vartype id: str
    :ivar key: A unique, human-readable identifier for the subject.
     This is typically a database ID or a customer key. Required.
    :vartype key: str
    :ivar display_name: A human-readable display name for the subject.
    :vartype display_name: str
    :ivar metadata: Metadata for the subject.
    :vartype metadata: dict[str, any]
    :ivar current_period_start: The start of the current period for the subject.
    :vartype current_period_start: ~datetime.datetime
    :ivar current_period_end: The end of the current period for the subject.
    :vartype current_period_end: ~datetime.datetime
    :ivar stripe_customer_id: The Stripe customer ID for the subject.
    :vartype stripe_customer_id: str
    """

    created_at: datetime.datetime = rest_field(name="createdAt", visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    updated_at: datetime.datetime = rest_field(name="updatedAt", visibility=["read"], format="rfc3339")
    """Last Update Time. Required."""
    deleted_at: Optional[datetime.datetime] = rest_field(name="deletedAt", visibility=["read"], format="rfc3339")
    """Deletion Time."""
    id: str = rest_field(visibility=["read"])
    """A unique identifier for the subject. Required."""
    key: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """A unique, human-readable identifier for the subject.
     This is typically a database ID or a customer key. Required."""
    display_name: Optional[str] = rest_field(
        name="displayName", visibility=["read", "create", "update", "delete", "query"]
    )
    """A human-readable display name for the subject."""
    metadata: Optional[dict[str, Any]] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata for the subject."""
    current_period_start: Optional[datetime.datetime] = rest_field(
        name="currentPeriodStart", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The start of the current period for the subject."""
    current_period_end: Optional[datetime.datetime] = rest_field(
        name="currentPeriodEnd", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The end of the current period for the subject."""
    stripe_customer_id: Optional[str] = rest_field(
        name="stripeCustomerId", visibility=["read", "create", "update", "delete", "query"]
    )
    """The Stripe customer ID for the subject."""

    @overload
    def __init__(
        self,
        *,
        key: str,
        display_name: Optional[str] = None,
        metadata: Optional[dict[str, Any]] = None,
        current_period_start: Optional[datetime.datetime] = None,
        current_period_end: Optional[datetime.datetime] = None,
        stripe_customer_id: Optional[str] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class SubjectUpsert(_Model):
    """A subject is a unique identifier for a user or entity.

    ⚠️ **Deprecated**: Subjects as managable entities are being depracated, use customers with
    subject key usage attribution instead.

    :ivar key: A unique, human-readable identifier for the subject.
     This is typically a database ID or a customer key. Required.
    :vartype key: str
    :ivar display_name: A human-readable display name for the subject.
    :vartype display_name: str
    :ivar metadata: Metadata for the subject.
    :vartype metadata: dict[str, any]
    :ivar current_period_start: The start of the current period for the subject.
    :vartype current_period_start: ~datetime.datetime
    :ivar current_period_end: The end of the current period for the subject.
    :vartype current_period_end: ~datetime.datetime
    :ivar stripe_customer_id: The Stripe customer ID for the subject.
    :vartype stripe_customer_id: str
    """

    key: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """A unique, human-readable identifier for the subject.
     This is typically a database ID or a customer key. Required."""
    display_name: Optional[str] = rest_field(
        name="displayName", visibility=["read", "create", "update", "delete", "query"]
    )
    """A human-readable display name for the subject."""
    metadata: Optional[dict[str, Any]] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata for the subject."""
    current_period_start: Optional[datetime.datetime] = rest_field(
        name="currentPeriodStart", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The start of the current period for the subject."""
    current_period_end: Optional[datetime.datetime] = rest_field(
        name="currentPeriodEnd", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The end of the current period for the subject."""
    stripe_customer_id: Optional[str] = rest_field(
        name="stripeCustomerId", visibility=["read", "create", "update", "delete", "query"]
    )
    """The Stripe customer ID for the subject."""

    @overload
    def __init__(
        self,
        *,
        key: str,
        display_name: Optional[str] = None,
        metadata: Optional[dict[str, Any]] = None,
        current_period_start: Optional[datetime.datetime] = None,
        current_period_end: Optional[datetime.datetime] = None,
        stripe_customer_id: Optional[str] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class Subscription(_Model):
    """Subscription is an exact subscription instance.

    :ivar id: ID. Required.
    :vartype id: str
    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar created_at: Creation Time. Required.
    :vartype created_at: ~datetime.datetime
    :ivar updated_at: Last Update Time. Required.
    :vartype updated_at: ~datetime.datetime
    :ivar deleted_at: Deletion Time.
    :vartype deleted_at: ~datetime.datetime
    :ivar active_from: The cadence start of the resource. Required.
    :vartype active_from: ~datetime.datetime
    :ivar active_to: The cadence end of the resource.
    :vartype active_to: ~datetime.datetime
    :ivar annotations: Annotations.
    :vartype annotations: ~openmeter._generated.models.Annotations
    :ivar alignment: Alignment configuration for the plan.
    :vartype alignment: ~openmeter._generated.models.Alignment
    :ivar status: The status of the subscription. Required. Known values are: "active", "inactive",
     "canceled", and "scheduled".
    :vartype status: str or ~openmeter.models.SubscriptionStatus
    :ivar customer_id: The customer ID of the subscription. Required.
    :vartype customer_id: str
    :ivar plan: The plan of the subscription.
    :vartype plan: ~openmeter._generated.models.PlanReference
    :ivar currency: Currency. Required.
    :vartype currency: str
    :ivar billing_cadence: Billing cadence. Required.
    :vartype billing_cadence: ~datetime.timedelta
    :ivar pro_rating_config: Pro-rating configuration.
    :vartype pro_rating_config: ~openmeter._generated.models.ProRatingConfig
    :ivar billing_anchor: Billing anchor. Required.
    :vartype billing_anchor: ~datetime.datetime
    """

    id: str = rest_field(visibility=["read"])
    """ID. Required."""
    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Display name. Required."""
    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    created_at: datetime.datetime = rest_field(name="createdAt", visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    updated_at: datetime.datetime = rest_field(name="updatedAt", visibility=["read"], format="rfc3339")
    """Last Update Time. Required."""
    deleted_at: Optional[datetime.datetime] = rest_field(name="deletedAt", visibility=["read"], format="rfc3339")
    """Deletion Time."""
    active_from: datetime.datetime = rest_field(
        name="activeFrom", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The cadence start of the resource. Required."""
    active_to: Optional[datetime.datetime] = rest_field(
        name="activeTo", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The cadence end of the resource."""
    annotations: Optional["_models.Annotations"] = rest_field(visibility=["read"])
    """Annotations."""
    alignment: Optional["_models.Alignment"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Alignment configuration for the plan."""
    status: Union[str, "_models.SubscriptionStatus"] = rest_field(visibility=["read"])
    """The status of the subscription. Required. Known values are: \"active\", \"inactive\",
     \"canceled\", and \"scheduled\"."""
    customer_id: str = rest_field(name="customerId", visibility=["read", "create", "update", "delete", "query"])
    """The customer ID of the subscription. Required."""
    plan: Optional["_models.PlanReference"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The plan of the subscription."""
    currency: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Currency. Required."""
    billing_cadence: datetime.timedelta = rest_field(name="billingCadence", visibility=["read"])
    """Billing cadence. Required."""
    pro_rating_config: Optional["_models.ProRatingConfig"] = rest_field(name="proRatingConfig", visibility=["read"])
    """Pro-rating configuration."""
    billing_anchor: datetime.datetime = rest_field(name="billingAnchor", visibility=["read"], format="rfc3339")
    """Billing anchor. Required."""

    @overload
    def __init__(
        self,
        *,
        name: str,
        active_from: datetime.datetime,
        customer_id: str,
        currency: str,
        description: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
        active_to: Optional[datetime.datetime] = None,
        alignment: Optional["_models.Alignment"] = None,
        plan: Optional["_models.PlanReference"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class SubscriptionAddon(_Model):
    """A subscription add-on, represents concrete instances of an add-on for a given subscription.

    :ivar id: ID. Required.
    :vartype id: str
    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar created_at: Creation Time. Required.
    :vartype created_at: ~datetime.datetime
    :ivar updated_at: Last Update Time. Required.
    :vartype updated_at: ~datetime.datetime
    :ivar deleted_at: Deletion Time.
    :vartype deleted_at: ~datetime.datetime
    :ivar active_from: The cadence start of the resource. Required.
    :vartype active_from: ~datetime.datetime
    :ivar active_to: The cadence end of the resource.
    :vartype active_to: ~datetime.datetime
    :ivar addon: Addon. Required.
    :vartype addon: ~openmeter._generated.models.SubscriptionAddonAddon
    :ivar quantity_at: QuantityAt. Required.
    :vartype quantity_at: ~datetime.datetime
    :ivar quantity: Quantity. Required.
    :vartype quantity: int
    :ivar timing: Timing. Required. Is either a Union[str, "_models.SubscriptionTimingEnum"] type
     or a datetime.datetime type.
    :vartype timing: str or ~openmeter.models.SubscriptionTimingEnum or ~datetime.datetime
    :ivar timeline: Timeline. Required.
    :vartype timeline: list[~openmeter._generated.models.SubscriptionAddonTimelineSegment]
    :ivar subscription_id: SubscriptionID. Required.
    :vartype subscription_id: str
    :ivar rate_cards: Rate cards. Required.
    :vartype rate_cards: list[~openmeter._generated.models.SubscriptionAddonRateCard]
    """

    id: str = rest_field(visibility=["read"])
    """ID. Required."""
    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Display name. Required."""
    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    created_at: datetime.datetime = rest_field(name="createdAt", visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    updated_at: datetime.datetime = rest_field(name="updatedAt", visibility=["read"], format="rfc3339")
    """Last Update Time. Required."""
    deleted_at: Optional[datetime.datetime] = rest_field(name="deletedAt", visibility=["read"], format="rfc3339")
    """Deletion Time."""
    active_from: datetime.datetime = rest_field(name="activeFrom", visibility=["read"], format="rfc3339")
    """The cadence start of the resource. Required."""
    active_to: Optional[datetime.datetime] = rest_field(name="activeTo", visibility=["read"], format="rfc3339")
    """The cadence end of the resource."""
    addon: "_models.SubscriptionAddonAddon" = rest_field(visibility=["read", "create"])
    """Addon. Required."""
    quantity_at: datetime.datetime = rest_field(name="quantityAt", visibility=["read"], format="rfc3339")
    """QuantityAt. Required."""
    quantity: int = rest_field(visibility=["read", "create", "update"])
    """Quantity. Required."""
    timing: "_types.SubscriptionTiming" = rest_field(visibility=["create", "update"])
    """Timing. Required. Is either a Union[str, \"_models.SubscriptionTimingEnum\"] type or a
     datetime.datetime type."""
    timeline: list["_models.SubscriptionAddonTimelineSegment"] = rest_field(visibility=["read"])
    """Timeline. Required."""
    subscription_id: str = rest_field(name="subscriptionId", visibility=["read"])
    """SubscriptionID. Required."""
    rate_cards: list["_models.SubscriptionAddonRateCard"] = rest_field(name="rateCards", visibility=["read"])
    """Rate cards. Required."""

    @overload
    def __init__(
        self,
        *,
        name: str,
        addon: "_models.SubscriptionAddonAddon",
        quantity: int,
        timing: "_types.SubscriptionTiming",
        description: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class SubscriptionAddonAddon(_Model):
    """SubscriptionAddonAddon.

    :ivar id: ID. Required.
    :vartype id: str
    :ivar key: Key. Required.
    :vartype key: str
    :ivar version: Version. Required.
    :vartype version: int
    :ivar instance_type: InstanceType. Required. Known values are: "single" and "multiple".
    :vartype instance_type: str or ~openmeter.models.AddonInstanceType
    """

    id: str = rest_field(visibility=["read", "create"])
    """ID. Required."""
    key: str = rest_field(visibility=["read"])
    """Key. Required."""
    version: int = rest_field(visibility=["read"])
    """Version. Required."""
    instance_type: Union[str, "_models.AddonInstanceType"] = rest_field(name="instanceType", visibility=["read"])
    """InstanceType. Required. Known values are: \"single\" and \"multiple\"."""

    @overload
    def __init__(
        self,
        *,
        id: str,  # pylint: disable=redefined-builtin
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class SubscriptionAddonCreate(_Model):
    """A subscription add-on create body.

    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar quantity: Quantity. Required.
    :vartype quantity: int
    :ivar timing: Timing. Required. Is either a Union[str, "_models.SubscriptionTimingEnum"] type
     or a datetime.datetime type.
    :vartype timing: str or ~openmeter.models.SubscriptionTimingEnum or ~datetime.datetime
    :ivar addon: Addon. Required.
    :vartype addon: ~openmeter._generated.models.SubscriptionAddonCreateAddon
    """

    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Display name. Required."""
    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    quantity: int = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Quantity. Required."""
    timing: "_types.SubscriptionTiming" = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Timing. Required. Is either a Union[str, \"_models.SubscriptionTimingEnum\"] type or a
     datetime.datetime type."""
    addon: "_models.SubscriptionAddonCreateAddon" = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """Addon. Required."""

    @overload
    def __init__(
        self,
        *,
        name: str,
        quantity: int,
        timing: "_types.SubscriptionTiming",
        addon: "_models.SubscriptionAddonCreateAddon",
        description: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class SubscriptionAddonCreateAddon(_Model):
    """SubscriptionAddonCreateAddon.

    :ivar id: The ID of the add-on. Required.
    :vartype id: str
    """

    id: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The ID of the add-on. Required."""

    @overload
    def __init__(
        self,
        *,
        id: str,  # pylint: disable=redefined-builtin
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class SubscriptionAddonRateCard(_Model):
    """A rate card for a subscription add-on.

    :ivar rate_card: Rate card. Required. Is either a RateCardFlatFee type or a RateCardUsageBased
     type.
    :vartype rate_card: ~openmeter._generated.models.RateCardFlatFee or
     ~openmeter._generated.models.RateCardUsageBased
    :ivar affected_subscription_item_ids: Affected subscription item IDs. Required.
    :vartype affected_subscription_item_ids: list[str]
    """

    rate_card: "_types.RateCard" = rest_field(
        name="rateCard", visibility=["read", "create", "update", "delete", "query"]
    )
    """Rate card. Required. Is either a RateCardFlatFee type or a RateCardUsageBased type."""
    affected_subscription_item_ids: list[str] = rest_field(name="affectedSubscriptionItemIds", visibility=["read"])
    """Affected subscription item IDs. Required."""

    @overload
    def __init__(
        self,
        *,
        rate_card: "_types.RateCard",
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class SubscriptionAddonTimelineSegment(_Model):
    """A subscription add-on event.

    :ivar active_from: The cadence start of the resource. Required.
    :vartype active_from: ~datetime.datetime
    :ivar active_to: The cadence end of the resource.
    :vartype active_to: ~datetime.datetime
    :ivar quantity: Quantity. Required.
    :vartype quantity: int
    """

    active_from: datetime.datetime = rest_field(
        name="activeFrom", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The cadence start of the resource. Required."""
    active_to: Optional[datetime.datetime] = rest_field(
        name="activeTo", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The cadence end of the resource."""
    quantity: int = rest_field(visibility=["read"])
    """Quantity. Required."""

    @overload
    def __init__(
        self,
        *,
        active_from: datetime.datetime,
        active_to: Optional[datetime.datetime] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class SubscriptionAddonUpdate(_Model):
    """Resource create or update operation model.

    :ivar name: Display name.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar quantity: Quantity.
    :vartype quantity: int
    :ivar timing: Timing. Is either a Union[str, "_models.SubscriptionTimingEnum"] type or a
     datetime.datetime type.
    :vartype timing: str or ~openmeter.models.SubscriptionTimingEnum or ~datetime.datetime
    """

    name: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Display name."""
    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    quantity: Optional[int] = rest_field(visibility=["read", "create", "update"])
    """Quantity."""
    timing: Optional["_types.SubscriptionTiming"] = rest_field(visibility=["create", "update"])
    """Timing. Is either a Union[str, \"_models.SubscriptionTimingEnum\"] type or a datetime.datetime
     type."""

    @overload
    def __init__(
        self,
        *,
        name: Optional[str] = None,
        description: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
        quantity: Optional[int] = None,
        timing: Optional["_types.SubscriptionTiming"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class SubscriptionAlignment(_Model):
    """Alignment details enriched with the current billing period.

    :ivar billables_must_align: Whether all Billable items and RateCards must align.
     Alignment means the Price's BillingCadence must align for both duration and anchor time.
    :vartype billables_must_align: bool
    :ivar current_aligned_billing_period: The current billing period. Only has value if the
     subscription is aligned and active.
    :vartype current_aligned_billing_period: ~openmeter._generated.models.Period
    """

    billables_must_align: Optional[bool] = rest_field(
        name="billablesMustAlign", visibility=["read", "create", "update"]
    )
    """Whether all Billable items and RateCards must align.
     Alignment means the Price's BillingCadence must align for both duration and anchor time."""
    current_aligned_billing_period: Optional["_models.Period"] = rest_field(
        name="currentAlignedBillingPeriod", visibility=["read", "create", "update", "delete", "query"]
    )
    """The current billing period. Only has value if the subscription is aligned and active."""

    @overload
    def __init__(
        self,
        *,
        billables_must_align: Optional[bool] = None,
        current_aligned_billing_period: Optional["_models.Period"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class SubscriptionBadRequestErrorResponse(_Model):
    """The server cannot or will not process the request due to something that is perceived to be a
    client error (e.g., malformed request syntax, invalid request message framing, or deceptive
    request routing). Variants with ErrorExtensions specific to subscriptions.

    :ivar type: Type contains a URI that identifies the problem type. Required.
    :vartype type: str
    :ivar title: A a short, human-readable summary of the problem type. Required.
    :vartype title: str
    :ivar status: The HTTP status code generated by the origin server for this occurrence of the
     problem.
    :vartype status: int
    :ivar detail: A human-readable explanation specific to this occurrence of the problem.
     Required.
    :vartype detail: str
    :ivar instance: A URI reference that identifies the specific occurrence of the problem.
     Required.
    :vartype instance: str
    :ivar extensions: Additional properties specific to the problem type may be present. Is one of
     the following types: CreateResponseExtensions
    :vartype extensions: ~openmeter._generated.models.CreateResponseExtensions
    """

    type: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Type contains a URI that identifies the problem type. Required."""
    title: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """A a short, human-readable summary of the problem type. Required."""
    status: Optional[int] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The HTTP status code generated by the origin server for this occurrence of the problem."""
    detail: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """A human-readable explanation specific to this occurrence of the problem. Required."""
    instance: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """A URI reference that identifies the specific occurrence of the problem. Required."""
    extensions: Optional["_types.SubscriptionErrorExtensions"] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """Additional properties specific to the problem type may be present. Is one of the following
     types: CreateResponseExtensions"""

    @overload
    def __init__(
        self,
        *,
        type: str,
        title: str,
        detail: str,
        instance: str,
        status: Optional[int] = None,
        extensions: Optional["_types.SubscriptionErrorExtensions"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class SubscriptionChangeResponseBody(_Model):
    """Response body for subscription change.

    :ivar current: Current subscription. Required.
    :vartype current: ~openmeter._generated.models.Subscription
    :ivar next: The subscription it will be changed to. Required.
    :vartype next: ~openmeter._generated.models.SubscriptionExpanded
    """

    current: "_models.Subscription" = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Current subscription. Required."""
    next: "_models.SubscriptionExpanded" = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The subscription it will be changed to. Required."""

    @overload
    def __init__(
        self,
        *,
        current: "_models.Subscription",
        next: "_models.SubscriptionExpanded",
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class SubscriptionConflictErrorResponse(_Model):
    """The request could not be completed due to a conflict with the current state of the target
    resource.
    Variants with ErrorExtensions specific to subscriptions.

    :ivar type: Type contains a URI that identifies the problem type. Required.
    :vartype type: str
    :ivar title: A a short, human-readable summary of the problem type. Required.
    :vartype title: str
    :ivar status: The HTTP status code generated by the origin server for this occurrence of the
     problem.
    :vartype status: int
    :ivar detail: A human-readable explanation specific to this occurrence of the problem.
     Required.
    :vartype detail: str
    :ivar instance: A URI reference that identifies the specific occurrence of the problem.
     Required.
    :vartype instance: str
    :ivar extensions: Additional properties specific to the problem type may be present. Is one of
     the following types: CreateResponseExtensions
    :vartype extensions: ~openmeter._generated.models.CreateResponseExtensions
    """

    type: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Type contains a URI that identifies the problem type. Required."""
    title: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """A a short, human-readable summary of the problem type. Required."""
    status: Optional[int] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The HTTP status code generated by the origin server for this occurrence of the problem."""
    detail: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """A human-readable explanation specific to this occurrence of the problem. Required."""
    instance: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """A URI reference that identifies the specific occurrence of the problem. Required."""
    extensions: Optional["_types.SubscriptionErrorExtensions"] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """Additional properties specific to the problem type may be present. Is one of the following
     types: CreateResponseExtensions"""

    @overload
    def __init__(
        self,
        *,
        type: str,
        title: str,
        detail: str,
        instance: str,
        status: Optional[int] = None,
        extensions: Optional["_types.SubscriptionErrorExtensions"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class SubscriptionEdit(_Model):
    """Subscription edit input.

    :ivar customizations: Batch processing commands for manipulating running subscriptions.
     The key format is ``/phases/{phaseKey}`` or ``/phases/{phaseKey}/items/{itemKey}``. Required.
    :vartype customizations: list[~openmeter._generated.models.EditSubscriptionAddItem or
     ~openmeter._generated.models.EditSubscriptionRemoveItem or
     ~openmeter._generated.models.EditSubscriptionAddPhase or
     ~openmeter._generated.models.EditSubscriptionRemovePhase or
     ~openmeter._generated.models.EditSubscriptionStretchPhase or
     ~openmeter._generated.models.EditSubscriptionUnscheduleEdit]
    :ivar timing: Whether the billing period should be restarted.Timing configuration to allow for
     the changes to take effect at different times. Is either a Union[str,
     "_models.SubscriptionTimingEnum"] type or a datetime.datetime type.
    :vartype timing: str or ~openmeter.models.SubscriptionTimingEnum or ~datetime.datetime
    """

    customizations: list["_types.SubscriptionEditOperation"] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """Batch processing commands for manipulating running subscriptions.
     The key format is ``/phases/{phaseKey}`` or ``/phases/{phaseKey}/items/{itemKey}``. Required."""
    timing: Optional["_types.SubscriptionTiming"] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """Whether the billing period should be restarted.Timing configuration to allow for the changes to
     take effect at different times. Is either a Union[str, \"_models.SubscriptionTimingEnum\"] type
     or a datetime.datetime type."""

    @overload
    def __init__(
        self,
        *,
        customizations: list["_types.SubscriptionEditOperation"],
        timing: Optional["_types.SubscriptionTiming"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class SubscriptionExpanded(_Model):
    """Expanded subscription.

    :ivar id: ID. Required.
    :vartype id: str
    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar created_at: Creation Time. Required.
    :vartype created_at: ~datetime.datetime
    :ivar updated_at: Last Update Time. Required.
    :vartype updated_at: ~datetime.datetime
    :ivar deleted_at: Deletion Time.
    :vartype deleted_at: ~datetime.datetime
    :ivar active_from: The cadence start of the resource. Required.
    :vartype active_from: ~datetime.datetime
    :ivar active_to: The cadence end of the resource.
    :vartype active_to: ~datetime.datetime
    :ivar annotations: Annotations.
    :vartype annotations: ~openmeter._generated.models.Annotations
    :ivar status: The status of the subscription. Required. Known values are: "active", "inactive",
     "canceled", and "scheduled".
    :vartype status: str or ~openmeter.models.SubscriptionStatus
    :ivar customer_id: The customer ID of the subscription. Required.
    :vartype customer_id: str
    :ivar plan: The plan of the subscription.
    :vartype plan: ~openmeter._generated.models.PlanReference
    :ivar currency: Currency. Required.
    :vartype currency: str
    :ivar billing_cadence: Billing cadence. Required.
    :vartype billing_cadence: ~datetime.timedelta
    :ivar pro_rating_config: Pro-rating configuration.
    :vartype pro_rating_config: ~openmeter._generated.models.ProRatingConfig
    :ivar billing_anchor: Billing anchor. Required.
    :vartype billing_anchor: ~datetime.datetime
    :ivar alignment: Alignment details enriched with the current billing period.
    :vartype alignment: ~openmeter._generated.models.SubscriptionAlignment
    :ivar phases: The phases of the subscription. Required.
    :vartype phases: list[~openmeter._generated.models.SubscriptionPhaseExpanded]
    """

    id: str = rest_field(visibility=["read"])
    """ID. Required."""
    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Display name. Required."""
    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    created_at: datetime.datetime = rest_field(name="createdAt", visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    updated_at: datetime.datetime = rest_field(name="updatedAt", visibility=["read"], format="rfc3339")
    """Last Update Time. Required."""
    deleted_at: Optional[datetime.datetime] = rest_field(name="deletedAt", visibility=["read"], format="rfc3339")
    """Deletion Time."""
    active_from: datetime.datetime = rest_field(
        name="activeFrom", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The cadence start of the resource. Required."""
    active_to: Optional[datetime.datetime] = rest_field(
        name="activeTo", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The cadence end of the resource."""
    annotations: Optional["_models.Annotations"] = rest_field(visibility=["read"])
    """Annotations."""
    status: Union[str, "_models.SubscriptionStatus"] = rest_field(visibility=["read"])
    """The status of the subscription. Required. Known values are: \"active\", \"inactive\",
     \"canceled\", and \"scheduled\"."""
    customer_id: str = rest_field(name="customerId", visibility=["read", "create", "update", "delete", "query"])
    """The customer ID of the subscription. Required."""
    plan: Optional["_models.PlanReference"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The plan of the subscription."""
    currency: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Currency. Required."""
    billing_cadence: datetime.timedelta = rest_field(name="billingCadence", visibility=["read"])
    """Billing cadence. Required."""
    pro_rating_config: Optional["_models.ProRatingConfig"] = rest_field(name="proRatingConfig", visibility=["read"])
    """Pro-rating configuration."""
    billing_anchor: datetime.datetime = rest_field(name="billingAnchor", visibility=["read"], format="rfc3339")
    """Billing anchor. Required."""
    alignment: Optional["_models.SubscriptionAlignment"] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """Alignment details enriched with the current billing period."""
    phases: list["_models.SubscriptionPhaseExpanded"] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """The phases of the subscription. Required."""

    @overload
    def __init__(
        self,
        *,
        name: str,
        active_from: datetime.datetime,
        customer_id: str,
        currency: str,
        phases: list["_models.SubscriptionPhaseExpanded"],
        description: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
        active_to: Optional[datetime.datetime] = None,
        plan: Optional["_models.PlanReference"] = None,
        alignment: Optional["_models.SubscriptionAlignment"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class SubscriptionItem(_Model):
    """The actual contents of the Subscription, what the user gets, what they pay, etc...

    :ivar id: ID. Required.
    :vartype id: str
    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: ~openmeter._generated.models.Metadata
    :ivar created_at: Creation Time. Required.
    :vartype created_at: ~datetime.datetime
    :ivar updated_at: Last Update Time. Required.
    :vartype updated_at: ~datetime.datetime
    :ivar deleted_at: Deletion Time.
    :vartype deleted_at: ~datetime.datetime
    :ivar active_from: The cadence start of the resource. Required.
    :vartype active_from: ~datetime.datetime
    :ivar active_to: The cadence end of the resource.
    :vartype active_to: ~datetime.datetime
    :ivar key: The identifier of the RateCard.
     SubscriptionItem/RateCard can be identified, it has a reference:



     1. If a Feature is associated with the SubscriptionItem, it is identified by the Feature
     1.1 It can be an ID reference, for an exact version of the Feature (Features can change across
     versions)
     1.2 It can be a Key reference, which always refers to the latest (active or inactive) version
     of a Feature

     2. If a Feature is not associated with the SubscriptionItem, it is referenced by the Price

     We say "referenced by the Price" regardless of how a price itself is referenced, it
     colloquially makes sense to say "paying the same price for the same thing". In practice this
     should be derived from what's printed on the invoice line-item. Required.
    :vartype key: str
    :ivar feature_key: The feature's key (if present).
    :vartype feature_key: str
    :ivar billing_cadence: Billing cadence. Required.
    :vartype billing_cadence: ~datetime.timedelta
    :ivar price: Price. Required. Is one of the following types: FlatPriceWithPaymentTerm,
     UnitPriceWithCommitments, TieredPriceWithCommitments, DynamicPriceWithCommitments,
     PackagePriceWithCommitments
    :vartype price: ~openmeter._generated.models.FlatPriceWithPaymentTerm or
     ~openmeter._generated.models.UnitPriceWithCommitments or
     ~openmeter._generated.models.TieredPriceWithCommitments or
     ~openmeter._generated.models.DynamicPriceWithCommitments or
     ~openmeter._generated.models.PackagePriceWithCommitments
    :ivar discounts: Discounts.
    :vartype discounts: ~openmeter._generated.models.Discounts
    :ivar included: Describes what access is gained via the SubscriptionItem.
    :vartype included: ~openmeter._generated.models.SubscriptionItemIncluded
    :ivar tax_config: Tax config.
    :vartype tax_config: ~openmeter._generated.models.TaxConfig
    """

    id: str = rest_field(visibility=["read"])
    """ID. Required."""
    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Display name. Required."""
    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    created_at: datetime.datetime = rest_field(name="createdAt", visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    updated_at: datetime.datetime = rest_field(name="updatedAt", visibility=["read"], format="rfc3339")
    """Last Update Time. Required."""
    deleted_at: Optional[datetime.datetime] = rest_field(name="deletedAt", visibility=["read"], format="rfc3339")
    """Deletion Time."""
    active_from: datetime.datetime = rest_field(
        name="activeFrom", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The cadence start of the resource. Required."""
    active_to: Optional[datetime.datetime] = rest_field(
        name="activeTo", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The cadence end of the resource."""
    key: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The identifier of the RateCard.
     SubscriptionItem/RateCard can be identified, it has a reference:
     
     
     
     1. If a Feature is associated with the SubscriptionItem, it is identified by the Feature
     1.1 It can be an ID reference, for an exact version of the Feature (Features can change across
     versions)
     1.2 It can be a Key reference, which always refers to the latest (active or inactive) version
     of a Feature
     
     2. If a Feature is not associated with the SubscriptionItem, it is referenced by the Price
     
     We say \"referenced by the Price\" regardless of how a price itself is referenced, it
     colloquially makes sense to say \"paying the same price for the same thing\". In practice this
     should be derived from what's printed on the invoice line-item. Required."""
    feature_key: Optional[str] = rest_field(
        name="featureKey", visibility=["read", "create", "update", "delete", "query"]
    )
    """The feature's key (if present)."""
    billing_cadence: datetime.timedelta = rest_field(
        name="billingCadence", visibility=["read", "create", "update", "delete", "query"]
    )
    """Billing cadence. Required."""
    price: "_types.RateCardUsageBasedPrice" = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Price. Required. Is one of the following types: FlatPriceWithPaymentTerm,
     UnitPriceWithCommitments, TieredPriceWithCommitments, DynamicPriceWithCommitments,
     PackagePriceWithCommitments"""
    discounts: Optional["_models.Discounts"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Discounts."""
    included: Optional["_models.SubscriptionItemIncluded"] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """Describes what access is gained via the SubscriptionItem."""
    tax_config: Optional["_models.TaxConfig"] = rest_field(name="taxConfig", visibility=["read", "create", "update"])
    """Tax config."""

    @overload
    def __init__(
        self,
        *,
        name: str,
        active_from: datetime.datetime,
        key: str,
        billing_cadence: datetime.timedelta,
        price: "_types.RateCardUsageBasedPrice",
        description: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
        active_to: Optional[datetime.datetime] = None,
        feature_key: Optional[str] = None,
        discounts: Optional["_models.Discounts"] = None,
        included: Optional["_models.SubscriptionItemIncluded"] = None,
        tax_config: Optional["_models.TaxConfig"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class SubscriptionItemIncluded(_Model):
    """Included contents like Entitlement, or the Feature.

    :ivar feature: The feature the customer is entitled to use. Required.
    :vartype feature: ~openmeter._generated.models.Feature
    :ivar entitlement: The entitlement of the Subscription Item. Is one of the following types:
     EntitlementMetered, EntitlementStatic, EntitlementBoolean
    :vartype entitlement: ~openmeter._generated.models.EntitlementMetered or
     ~openmeter._generated.models.EntitlementStatic or
     ~openmeter._generated.models.EntitlementBoolean
    """

    feature: "_models.Feature" = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The feature the customer is entitled to use. Required."""
    entitlement: Optional["_types.Entitlement"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The entitlement of the Subscription Item. Is one of the following types: EntitlementMetered,
     EntitlementStatic, EntitlementBoolean"""

    @overload
    def __init__(
        self,
        *,
        feature: "_models.Feature",
        entitlement: Optional["_types.Entitlement"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class SubscriptionPaginatedResponse(_Model):
    """Paginated response.

    :ivar total_count: The total number of items. Required.
    :vartype total_count: int
    :ivar page: The page index. Required.
    :vartype page: int
    :ivar page_size: The maximum number of items per page. Required.
    :vartype page_size: int
    :ivar items_property: The items in the current page. Required.
    :vartype items_property: list[~openmeter._generated.models.Subscription]
    """

    total_count: int = rest_field(name="totalCount", visibility=["read", "create", "update", "delete", "query"])
    """The total number of items. Required."""
    page: int = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The page index. Required."""
    page_size: int = rest_field(name="pageSize", visibility=["read", "create", "update", "delete", "query"])
    """The maximum number of items per page. Required."""
    items_property: list["_models.Subscription"] = rest_field(
        name="items", visibility=["read", "create", "update", "delete", "query"]
    )
    """The items in the current page. Required."""

    @overload
    def __init__(
        self,
        *,
        total_count: int,
        page: int,
        page_size: int,
        items_property: list["_models.Subscription"],
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class SubscriptionPhaseCreate(_Model):
    """Subscription phase create input.

    :ivar start_after: Start after. Required.
    :vartype start_after: ~datetime.timedelta
    :ivar duration: Duration.
    :vartype duration: ~datetime.timedelta
    :ivar discounts: Discounts.
    :vartype discounts: ~openmeter._generated.models.Discounts
    :ivar key: A locally unique identifier for the phase. Required.
    :vartype key: str
    :ivar name: The name of the phase. Required.
    :vartype name: str
    :ivar description: The description of the phase.
    :vartype description: str
    """

    start_after: datetime.timedelta = rest_field(
        name="startAfter", visibility=["read", "create", "update", "delete", "query"]
    )
    """Start after. Required."""
    duration: Optional[datetime.timedelta] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Duration."""
    discounts: Optional["_models.Discounts"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Discounts."""
    key: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """A locally unique identifier for the phase. Required."""
    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The name of the phase. Required."""
    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The description of the phase."""

    @overload
    def __init__(
        self,
        *,
        start_after: datetime.timedelta,
        key: str,
        name: str,
        duration: Optional[datetime.timedelta] = None,
        discounts: Optional["_models.Discounts"] = None,
        description: Optional[str] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class SubscriptionPhaseExpanded(_Model):
    """Expanded subscription phase.

        :ivar id: ID. Required.
        :vartype id: str
        :ivar name: Display name. Required.
        :vartype name: str
        :ivar description: Description.
        :vartype description: str
        :ivar metadata: Metadata.
        :vartype metadata: ~openmeter._generated.models.Metadata
        :ivar created_at: Creation Time. Required.
        :vartype created_at: ~datetime.datetime
        :ivar updated_at: Last Update Time. Required.
        :vartype updated_at: ~datetime.datetime
        :ivar deleted_at: Deletion Time.
        :vartype deleted_at: ~datetime.datetime
        :ivar key: A locally unique identifier for the resource. Required.
        :vartype key: str
        :ivar discounts: Discounts.
        :vartype discounts: ~openmeter._generated.models.Discounts
        :ivar active_from: The time from which the phase is active. Required.
        :vartype active_from: ~datetime.datetime
        :ivar active_to: The until which the Phase is active.
        :vartype active_to: ~datetime.datetime
        :ivar items_property: The items of the phase. The structure is flattened to better conform to
        the Plan API.
    The timelines are flattened according to the following rules:

         * for the current phase, the `items` contains only the active item for each key
         * for past phases, the `items` contains only the last item for each key
         * for future phases, the `items` contains only the first version of the item for each key.
           Required.
        :vartype items_property: list[~openmeter._generated.models.SubscriptionItem]
        :ivar item_timelines: Includes all versions of the items on each key, including all edits,
         scheduled changes, etc... Required.
        :vartype item_timelines: dict[str, list[~openmeter._generated.models.SubscriptionItem]]
    """

    id: str = rest_field(visibility=["read"])
    """ID. Required."""
    name: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Display name. Required."""
    description: Optional[str] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Description."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Metadata."""
    created_at: datetime.datetime = rest_field(name="createdAt", visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    updated_at: datetime.datetime = rest_field(name="updatedAt", visibility=["read"], format="rfc3339")
    """Last Update Time. Required."""
    deleted_at: Optional[datetime.datetime] = rest_field(name="deletedAt", visibility=["read"], format="rfc3339")
    """Deletion Time."""
    key: str = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """A locally unique identifier for the resource. Required."""
    discounts: Optional["_models.Discounts"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Discounts."""
    active_from: datetime.datetime = rest_field(
        name="activeFrom", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The time from which the phase is active. Required."""
    active_to: Optional[datetime.datetime] = rest_field(
        name="activeTo", visibility=["read", "create", "update", "delete", "query"], format="rfc3339"
    )
    """The until which the Phase is active."""
    items_property: list["_models.SubscriptionItem"] = rest_field(
        name="items", visibility=["read", "create", "update", "delete", "query"]
    )
    """The items of the phase. The structure is flattened to better conform to the Plan API.
 The timelines are flattened according to the following rules:
 
      * for the current phase, the `items` contains only the active item for each key
      * for past phases, the `items` contains only the last item for each key
      * for future phases, the `items` contains only the first version of the item for each key.
        Required."""
    item_timelines: dict[str, list["_models.SubscriptionItem"]] = rest_field(
        name="itemTimelines", visibility=["read", "create", "update", "delete", "query"]
    )
    """Includes all versions of the items on each key, including all edits, scheduled changes, etc...
     Required."""

    @overload
    def __init__(
        self,
        *,
        name: str,
        key: str,
        active_from: datetime.datetime,
        items_property: list["_models.SubscriptionItem"],
        item_timelines: dict[str, list["_models.SubscriptionItem"]],
        description: Optional[str] = None,
        metadata: Optional["_models.Metadata"] = None,
        discounts: Optional["_models.Discounts"] = None,
        active_to: Optional[datetime.datetime] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class TaxConfig(_Model):
    """Set of provider specific tax configs.

    :ivar behavior: Tax behavior. Known values are: "inclusive" and "exclusive".
    :vartype behavior: str or ~openmeter.models.TaxBehavior
    :ivar stripe: Stripe tax config.
    :vartype stripe: ~openmeter._generated.models.StripeTaxConfig
    :ivar custom_invoicing: Custom invoicing tax config.
    :vartype custom_invoicing: ~openmeter._generated.models.CustomInvoicingTaxConfig
    """

    behavior: Optional[Union[str, "_models.TaxBehavior"]] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """Tax behavior. Known values are: \"inclusive\" and \"exclusive\"."""
    stripe: Optional["_models.StripeTaxConfig"] = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """Stripe tax config."""
    custom_invoicing: Optional["_models.CustomInvoicingTaxConfig"] = rest_field(
        name="customInvoicing", visibility=["read", "create", "update", "delete", "query"]
    )
    """Custom invoicing tax config."""

    @overload
    def __init__(
        self,
        *,
        behavior: Optional[Union[str, "_models.TaxBehavior"]] = None,
        stripe: Optional["_models.StripeTaxConfig"] = None,
        custom_invoicing: Optional["_models.CustomInvoicingTaxConfig"] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class TieredPriceWithCommitments(_Model):
    """Tiered price with spend commitments.

    :ivar type: The type of the price.

     One of: flat, unit, or tiered. Required.
    :vartype type: str or ~openmeter._generated.models.TIERED
    :ivar mode: Mode. Required. Known values are: "volume" and "graduated".
    :vartype mode: str or ~openmeter.models.TieredPriceMode
    :ivar tiers: Tiers. Required.
    :vartype tiers: list[~openmeter._generated.models.PriceTier]
    :ivar minimum_amount: Minimum amount.
    :vartype minimum_amount: str
    :ivar maximum_amount: Maximum amount.
    :vartype maximum_amount: str
    """

    type: Literal[PriceType.TIERED] = rest_field(visibility=["read", "create", "update"])
    """The type of the price.
     
     One of: flat, unit, or tiered. Required."""
    mode: Union[str, "_models.TieredPriceMode"] = rest_field(visibility=["read", "create", "update"])
    """Mode. Required. Known values are: \"volume\" and \"graduated\"."""
    tiers: list["_models.PriceTier"] = rest_field(visibility=["read", "create", "update"])
    """Tiers. Required."""
    minimum_amount: Optional[str] = rest_field(name="minimumAmount", visibility=["read", "create", "update"])
    """Minimum amount."""
    maximum_amount: Optional[str] = rest_field(name="maximumAmount", visibility=["read", "create", "update"])
    """Maximum amount."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[PriceType.TIERED],
        mode: Union[str, "_models.TieredPriceMode"],
        tiers: list["_models.PriceTier"],
        minimum_amount: Optional[str] = None,
        maximum_amount: Optional[str] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class UnauthorizedProblemResponse(UnexpectedProblemResponse):
    """The request has not been applied because it lacks valid authentication credentials for the
    target resource.

    :ivar type: Type contains a URI that identifies the problem type. Required.
    :vartype type: str
    :ivar title: A a short, human-readable summary of the problem type. Required.
    :vartype title: str
    :ivar status: The HTTP status code generated by the origin server for this occurrence of the
     problem.
    :vartype status: int
    :ivar detail: A human-readable explanation specific to this occurrence of the problem.
     Required.
    :vartype detail: str
    :ivar instance: A URI reference that identifies the specific occurrence of the problem.
     Required.
    :vartype instance: str
    :ivar extensions: Additional properties specific to the problem type may be present.
    :vartype extensions: dict[str, any]
    """

    @overload
    def __init__(
        self,
        *,
        type: str,
        title: str,
        detail: str,
        instance: str,
        status: Optional[int] = None,
        extensions: Optional[dict[str, Any]] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class UnitPrice(_Model):
    """Unit price.

    :ivar type: The type of the price. Required.
    :vartype type: str or ~openmeter._generated.models.UNIT
    :ivar amount: The amount of the unit price. Required.
    :vartype amount: str
    """

    type: Literal[PriceType.UNIT] = rest_field(visibility=["read", "create", "update"])
    """The type of the price. Required."""
    amount: str = rest_field(visibility=["read", "create", "update"])
    """The amount of the unit price. Required."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[PriceType.UNIT],
        amount: str,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class UnitPriceWithCommitments(_Model):
    """Unit price with spend commitments.

    :ivar type: The type of the price. Required.
    :vartype type: str or ~openmeter._generated.models.UNIT
    :ivar amount: The amount of the unit price. Required.
    :vartype amount: str
    :ivar minimum_amount: Minimum amount.
    :vartype minimum_amount: str
    :ivar maximum_amount: Maximum amount.
    :vartype maximum_amount: str
    """

    type: Literal[PriceType.UNIT] = rest_field(visibility=["read", "create", "update"])
    """The type of the price. Required."""
    amount: str = rest_field(visibility=["read", "create", "update"])
    """The amount of the unit price. Required."""
    minimum_amount: Optional[str] = rest_field(name="minimumAmount", visibility=["read", "create", "update"])
    """Minimum amount."""
    maximum_amount: Optional[str] = rest_field(name="maximumAmount", visibility=["read", "create", "update"])
    """Maximum amount."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[PriceType.UNIT],
        amount: str,
        minimum_amount: Optional[str] = None,
        maximum_amount: Optional[str] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class ValidationError(_Model):
    """Validation errors providing detailed description of the issue.

    :ivar field: The path to the field. Required.
    :vartype field: str
    :ivar code: The machine readable description of the error. Required.
    :vartype code: str
    :ivar message: The human readable description of the error. Required.
    :vartype message: str
    :ivar attributes: Additional attributes.
    :vartype attributes: ~openmeter._generated.models.Annotations
    """

    field: str = rest_field(visibility=["read"])
    """The path to the field. Required."""
    code: str = rest_field(visibility=["read"])
    """The machine readable description of the error. Required."""
    message: str = rest_field(visibility=["read"])
    """The human readable description of the error. Required."""
    attributes: Optional["_models.Annotations"] = rest_field(visibility=["read"])
    """Additional attributes."""


class ValidationIssue(_Model):
    """ValidationIssue captures any validation issues related to the invoice.

    Issues with severity "critical" will prevent the invoice from being issued.

    :ivar created_at: Creation Time. Required.
    :vartype created_at: ~datetime.datetime
    :ivar updated_at: Last Update Time. Required.
    :vartype updated_at: ~datetime.datetime
    :ivar deleted_at: Deletion Time.
    :vartype deleted_at: ~datetime.datetime
    :ivar id: ID of the charge or discount. Required.
    :vartype id: str
    :ivar severity: The severity of the issue. Required. Known values are: "critical" and
     "warning".
    :vartype severity: str or ~openmeter.models.ValidationIssueSeverity
    :ivar field: The field that the issue is related to, if available in JSON path format.
    :vartype field: str
    :ivar code: Machine indentifiable code for the issue, if available.
    :vartype code: str
    :ivar component: Component reporting the issue. Required.
    :vartype component: str
    :ivar message: A human-readable description of the issue. Required.
    :vartype message: str
    :ivar metadata: Additional context for the issue.
    :vartype metadata: ~openmeter._generated.models.Metadata
    """

    created_at: datetime.datetime = rest_field(name="createdAt", visibility=["read"], format="rfc3339")
    """Creation Time. Required."""
    updated_at: datetime.datetime = rest_field(name="updatedAt", visibility=["read"], format="rfc3339")
    """Last Update Time. Required."""
    deleted_at: Optional[datetime.datetime] = rest_field(name="deletedAt", visibility=["read"], format="rfc3339")
    """Deletion Time."""
    id: str = rest_field(visibility=["read"])
    """ID of the charge or discount. Required."""
    severity: Union[str, "_models.ValidationIssueSeverity"] = rest_field(visibility=["read"])
    """The severity of the issue. Required. Known values are: \"critical\" and \"warning\"."""
    field: Optional[str] = rest_field(visibility=["read"])
    """The field that the issue is related to, if available in JSON path format."""
    code: Optional[str] = rest_field(visibility=["read"])
    """Machine indentifiable code for the issue, if available."""
    component: str = rest_field(visibility=["read"])
    """Component reporting the issue. Required."""
    message: str = rest_field(visibility=["read"])
    """A human-readable description of the issue. Required."""
    metadata: Optional["_models.Metadata"] = rest_field(visibility=["read"])
    """Additional context for the issue."""


class VoidInvoiceAction(_Model):
    """InvoiceVoidAction describes how to handle the voided line items.

    :ivar percentage: How much of the total line items to be voided? (e.g. 100% means all charges
     are voided). Required.
    :vartype percentage: float
    :ivar action: The action to take on the line items. Required. Is either a
     VoidInvoiceLineDiscardAction type or a VoidInvoiceLinePendingAction type.
    :vartype action: ~openmeter._generated.models.VoidInvoiceLineDiscardAction or
     ~openmeter._generated.models.VoidInvoiceLinePendingAction
    """

    percentage: float = rest_field(visibility=["create"])
    """How much of the total line items to be voided? (e.g. 100% means all charges are voided).
     Required."""
    action: "_types.VoidInvoiceLineAction" = rest_field(visibility=["read", "create", "update", "delete", "query"])
    """The action to take on the line items. Required. Is either a VoidInvoiceLineDiscardAction type
     or a VoidInvoiceLinePendingAction type."""

    @overload
    def __init__(
        self,
        *,
        percentage: float,
        action: "_types.VoidInvoiceLineAction",
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class VoidInvoiceActionInput(_Model):
    """Request to void an invoice.

    :ivar action: The action to take on the voided line items. Required.
    :vartype action: ~openmeter._generated.models.VoidInvoiceAction
    :ivar reason: The reason for voiding the invoice. Required.
    :vartype reason: str
    :ivar overrides: Per line item overrides for the action.

     If not specified, the ``action`` will be applied to all line items.
    :vartype overrides: list[~openmeter._generated.models.VoidInvoiceActionLineOverride]
    """

    action: "_models.VoidInvoiceAction" = rest_field(visibility=["create"])
    """The action to take on the voided line items. Required."""
    reason: str = rest_field(visibility=["create"])
    """The reason for voiding the invoice. Required."""
    overrides: Optional[list["_models.VoidInvoiceActionLineOverride"]] = rest_field(visibility=["create"])
    """Per line item overrides for the action.
     
     If not specified, the ``action`` will be applied to all line items."""

    @overload
    def __init__(
        self,
        *,
        action: "_models.VoidInvoiceAction",
        reason: str,
        overrides: Optional[list["_models.VoidInvoiceActionLineOverride"]] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class VoidInvoiceActionLineOverride(_Model):
    """VoidInvoiceLineOverride describes how to handle a specific line item in the invoice when
    voiding.

    :ivar line_id: The line item ID to override. Required.
    :vartype line_id: str
    :ivar action: The action to take on the line item. Required.
    :vartype action: ~openmeter._generated.models.VoidInvoiceAction
    """

    line_id: str = rest_field(name="lineId", visibility=["create"])
    """The line item ID to override. Required."""
    action: "_models.VoidInvoiceAction" = rest_field(visibility=["create"])
    """The action to take on the line item. Required."""

    @overload
    def __init__(
        self,
        *,
        line_id: str,
        action: "_models.VoidInvoiceAction",
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class VoidInvoiceLineDiscardAction(_Model):
    """VoidInvoiceLineDiscardAction describes how to handle the voidied line item in the invoice.

    :ivar type: The action to take on the line item. Required. The line items will never be charged
     for again
    :vartype type: str or ~openmeter._generated.models.DISCARD
    """

    type: Literal[VoidInvoiceLineActionType.DISCARD] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """The action to take on the line item. Required. The line items will never be charged for again"""

    @overload
    def __init__(
        self,
        *,
        type: Literal[VoidInvoiceLineActionType.DISCARD],
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class VoidInvoiceLinePendingAction(_Model):
    """VoidInvoiceLinePendingAction describes how to handle the voidied line item in the invoice.

    :ivar type: The action to take on the line item. Required. Queue the line items into the
     pending state, they will be included in the next invoice. (We want to generate an invoice right
     now)
    :vartype type: str or ~openmeter._generated.models.PENDING
    :ivar next_invoice_at: The time at which the line item should be invoiced again.

     If not provided, the line item will be re-invoiced now.
    :vartype next_invoice_at: ~datetime.datetime
    """

    type: Literal[VoidInvoiceLineActionType.PENDING] = rest_field(
        visibility=["read", "create", "update", "delete", "query"]
    )
    """The action to take on the line item. Required. Queue the line items into the pending state,
     they will be included in the next invoice. (We want to generate an invoice right now)"""
    next_invoice_at: Optional[datetime.datetime] = rest_field(
        name="nextInvoiceAt", visibility=["create"], format="rfc3339"
    )
    """The time at which the line item should be invoiced again.
     
     If not provided, the line item will be re-invoiced now."""

    @overload
    def __init__(
        self,
        *,
        type: Literal[VoidInvoiceLineActionType.PENDING],
        next_invoice_at: Optional[datetime.datetime] = None,
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)


class WindowedBalanceHistory(_Model):
    """The windowed balance history.

    :ivar windowed_history: The windowed balance history.

     * It only returns rows for windows where there was usage.
     * The windows are inclusive at their start and exclusive at their end.
     * The last window may be smaller than the window size and is inclusive at both ends.
       Required.
    :vartype windowed_history: list[~openmeter._generated.models.BalanceHistoryWindow]
    :ivar burndown_history: Grant burndown history. Required.
    :vartype burndown_history: list[~openmeter._generated.models.GrantBurnDownHistorySegment]
    """

    windowed_history: list["_models.BalanceHistoryWindow"] = rest_field(
        name="windowedHistory", visibility=["read", "create", "update", "delete", "query"]
    )
    """The windowed balance history.
 
      * It only returns rows for windows where there was usage.
      * The windows are inclusive at their start and exclusive at their end.
      * The last window may be smaller than the window size and is inclusive at both ends.
        Required."""
    burndown_history: list["_models.GrantBurnDownHistorySegment"] = rest_field(
        name="burndownHistory", visibility=["read", "create", "update", "delete", "query"]
    )
    """Grant burndown history. Required."""

    @overload
    def __init__(
        self,
        *,
        windowed_history: list["_models.BalanceHistoryWindow"],
        burndown_history: list["_models.GrantBurnDownHistorySegment"],
    ) -> None: ...

    @overload
    def __init__(self, mapping: Mapping[str, Any]) -> None:
        """
        :param mapping: raw JSON to initialize the model.
        :type mapping: Mapping[str, Any]
        """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)
