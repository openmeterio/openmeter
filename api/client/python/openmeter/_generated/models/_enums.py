# coding=utf-8

from enum import Enum
from corehttp.utils import CaseInsensitiveEnumMeta


class AddonInstanceType(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """The instanceType of the add-on.
    Single instance add-ons can be added to subscription only once while add-ons with multiple type
    can be added more then once.
    """

    SINGLE = "single"
    MULTIPLE = "multiple"


class AddonOrderBy(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """Order by options for add-ons."""

    ID = "id"
    KEY = "key"
    VERSION = "version"
    CREATED_AT = "created_at"
    UPDATED_AT = "updated_at"


class AddonStatus(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """The status of the add-on defined by the effectiveFrom and effectiveTo properties."""

    DRAFT = "draft"
    ACTIVE = "active"
    ARCHIVED = "archived"


class AppCapabilityType(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """App capability type."""

    REPORT_USAGE = "reportUsage"
    """The app can report aggregated usage."""
    REPORT_EVENTS = "reportEvents"
    """The app can report raw events."""
    CALCULATE_TAX = "calculateTax"
    """The app can calculate tax."""
    INVOICE_CUSTOMERS = "invoiceCustomers"
    """The app can invoice customers."""
    COLLECT_PAYMENTS = "collectPayments"
    """The app can collect payments."""


class AppStatus(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """App installed status."""

    READY = "ready"
    """The app is ready to be used."""
    UNAUTHORIZED = "unauthorized"
    """The app is unauthorized.
    This usually happens when the app's credentials are revoked or expired.
    To resolve this, the user must re-authorize the app."""


class AppType(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """Type of the app."""

    STRIPE = "stripe"
    SANDBOX = "sandbox"
    CUSTOM_INVOICING = "custom_invoicing"


class BillingCollectionAlignment(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """Collection alignment."""

    SUBSCRIPTION = "subscription"
    """Align the collection to the start of the subscription period."""
    ANCHORED = "anchored"
    """Align the collection to the anchor time and cadence."""


class BillingProfileCustomerOverrideExpand(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """CustomerOverrideExpand specifies the parts of the profile to expand."""

    APPS = "apps"
    CUSTOMER = "customer"


class BillingProfileCustomerOverrideOrderBy(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """Order by options for customers."""

    CUSTOMER_ID = "customerId"
    CUSTOMER_NAME = "customerName"
    CUSTOMER_KEY = "customerKey"
    CUSTOMER_PRIMARY_EMAIL = "customerPrimaryEmail"
    CUSTOMER_CREATED_AT = "customerCreatedAt"


class BillingProfileExpand(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """BillingProfileExpand details what profile fields to expand."""

    APPS = "apps"


class BillingProfileOrderBy(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """BillingProfileOrderBy specifies the ordering options for profiles."""

    CREATED_AT = "createdAt"
    UPDATED_AT = "updatedAt"
    DEFAULT = "default"
    NAME = "name"


class CheckoutSessionUIMode(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """Stripe CheckoutSession.ui_mode."""

    EMBEDDED = "embedded"
    HOSTED = "hosted"


class CollectionMethod(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """Collection method."""

    CHARGE_AUTOMATICALLY = "charge_automatically"
    SEND_INVOICE = "send_invoice"


class CreateCheckoutSessionTaxIdCollectionRequired(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """Create Stripe checkout session tax ID collection required."""

    IF_SUPPORTED = "if_supported"
    """A tax ID will be required if collection is supported for the selected billing address country.
    See: `https://docs.stripe.com/tax/checkout/tax-ids#supported-types
    <https://docs.stripe.com/tax/checkout/tax-ids#supported-types>`_"""
    NEVER = "never"
    """Tax ID collection is never required."""


class CreateStripeCheckoutSessionBillingAddressCollection(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """Specify whether Checkout should collect the customer’s billing address."""

    AUTO = "auto"
    """Checkout will only collect the billing address when necessary.
    When using automatic_tax, Checkout will collect the minimum number of fields required for tax
    calculation."""
    REQUIRED = "required"
    """Checkout will always collect the customer’s billing address."""


class CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPosition(
    str, Enum, metaclass=CaseInsensitiveEnumMeta
):
    """Create Stripe checkout session consent collection agreement position."""

    AUTO = "auto"
    """Uses Stripe defaults to determine the visibility and position of the payment method reuse
    agreement."""
    HIDDEN = "hidden"
    """Hides the payment method reuse agreement."""


class CreateStripeCheckoutSessionConsentCollectionPromotions(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """Create Stripe checkout session consent collection promotions."""

    AUTO = "auto"
    """Enable the collection of customer consent for promotional communications.
    The Checkout Session will determine whether to display an option to opt into promotional
    communication from the merchant depending on if a customer is provided,
    and if that customer has consented to receiving promotional communications from the merchant in
    the past."""
    NONE = "none"
    """Checkout will not collect customer consent for promotional communications."""


class CreateStripeCheckoutSessionConsentCollectionTermsOfService(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """Create Stripe checkout session consent collection terms of service."""

    NONE = "none"
    """Does not display checkbox for the terms of service agreement."""
    REQUIRED = "required"
    """Displays a checkbox for the terms of service agreement which requires customer to check before
    being able to pay."""


class CreateStripeCheckoutSessionCustomerUpdateBehavior(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """Create Stripe checkout session customer update behavior."""

    AUTO = "auto"
    """Checkout will automatically determine whether to update the provided Customer object using
    details from the session."""
    NEVER = "never"
    """Checkout will never update the provided Customer object."""


class CreateStripeCheckoutSessionRedirectOnCompletion(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """Create Stripe checkout session redirect on completion."""

    ALWAYS = "always"
    """The Session will always redirect to the return_url after successful confirmation."""
    IF_REQUIRED = "if_required"
    """The Session will only redirect to the return_url after a redirect-based payment method is used."""
    NEVER = "never"
    """The Session will never redirect to the return_url, and redirect-based payment methods will be
    disabled."""


class CustomerExpand(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """CustomerExpand specifies the parts of the customer to expand in the list output."""

    SUBSCRIPTIONS = "subscriptions"


class CustomerOrderBy(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """Order by options for customers."""

    ID = "id"
    NAME = "name"
    CREATED_AT = "createdAt"


class CustomerSubscriptionOrderBy(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """Order by options for customer subscriptions."""

    ACTIVE_FROM = "activeFrom"
    ACTIVE_TO = "activeTo"


class CustomInvoicingPaymentTrigger(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """Payment trigger to execute on a finalized invoice."""

    PAID = "paid"
    PAYMENT_FAILED = "payment_failed"
    PAYMENT_UNCOLLECTIBLE = "payment_uncollectible"
    PAYMENT_OVERDUE = "payment_overdue"
    ACTION_REQUIRED = "action_required"
    VOID = "void"


class DiscountReasonType(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """The type of the discount reason."""

    MAXIMUM_SPEND = "maximum_spend"
    RATECARD_PERCENTAGE = "ratecard_percentage"
    RATECARD_USAGE = "ratecard_usage"


class EditOp(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """Enum listing the different operation types."""

    ADD_ITEM = "add_item"
    REMOVE_ITEM = "remove_item"
    UNSCHEDULE_EDIT = "unschedule_edit"
    ADD_PHASE = "add_phase"
    REMOVE_PHASE = "remove_phase"
    STRETCH_PHASE = "stretch_phase"


class EntitlementOrderBy(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """Order by options for entitlements."""

    CREATED_AT = "createdAt"
    UPDATED_AT = "updatedAt"


class EntitlementType(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """Type of the entitlement."""

    METERED = "metered"
    BOOLEAN = "boolean"
    STATIC = "static"


class ExpirationDuration(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """The expiration duration enum."""

    HOUR = "HOUR"
    DAY = "DAY"
    WEEK = "WEEK"
    MONTH = "MONTH"
    YEAR = "YEAR"


class FeatureOrderBy(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """Order by options for features."""

    ID = "id"
    KEY = "key"
    NAME = "name"
    CREATED_AT = "createdAt"
    UPDATED_AT = "updatedAt"


class GrantOrderBy(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """Order by options for grants."""

    ID = "id"
    CREATED_AT = "createdAt"
    UPDATED_AT = "updatedAt"


class InstallMethod(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """Install method of the application."""

    WITH_O_AUTH2 = "with_oauth2"
    WITH_API_KEY = "with_api_key"
    NO_CREDENTIALS_REQUIRED = "no_credentials_required"


class InvoiceDetailedLineCostCategory(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """InvoiceDetailedLineCostCategory determines if the flat fee is a regular fee due to use due to a
    commitment.
    """

    REGULAR = "regular"
    """The fee is a regular fee due to usage."""
    COMMITMENT = "commitment"
    """The fee is a fee due to a commitment (e.g. minimum spend)."""


class InvoiceDocumentRefType(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """InvoiceDocumentRefType defines the type of document that is being referenced."""

    CREDIT_NOTE_ORIGINAL_INVOICE = "credit_note_original_invoice"


class InvoiceExpand(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """InvoiceExpand specifies the parts of the invoice to expand in the list output."""

    LINES = "lines"
    PRECEDING = "preceding"
    WORKFLOW_APPS = "workflow.apps"


class InvoiceLineManagedBy(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """InvoiceLineManagedBy specifies who manages the line."""

    SUBSCRIPTION = "subscription"
    """The line is managed by the susbcription engine of
    
    If there are any changes to the subscription the line will be updated accordingly."""
    SYSTEM = "system"
    """The line is managed by the billing system of the
    
    The line is immutable."""
    MANUAL = "manual"
    """The line is managed via our API.
    
    If the line is coming from a subscription we will not update the line if the subscription
    changes.
    
    The only exception is that the period and invoiceAt fields will be updated in case of
    progressively billed
    usage-based lines to maintain the coherence of the line structure. Any other fields edited will
    be kept as is."""


class InvoiceLineStatus(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """Line status specifies the status of the line."""

    VALID = "valid"
    """The line is valid and can be used in the invoice."""
    DETAILED = "detailed"
    """The line is a detail line which is used to detail the individual
    charges and discounts of a valid line."""
    SPLIT = "split"
    """The line has been split into multiple valid lines due to progressive
    billing."""


class InvoiceLineTaxBehavior(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """InvoiceLineTaxBehavior details how the tax item is applied to the base amount.

    Inclusive means the tax is included in the base amount.
    Exclusive means the tax is added to the base amount.
    """

    INCLUSIVE = "inclusive"
    """Tax is included in the base amount."""
    EXCLUSIVE = "exclusive"
    """Tax is added to the base amount."""


class InvoiceLineTypes(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """LineTypes represents the different types of lines that can be used in an invoice."""

    FLAT_FEE = "flat_fee"
    USAGE_BASED = "usage_based"


class InvoiceOrderBy(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """InvoiceOrderBy specifies the ordering options for invoice listing."""

    CUSTOMER_NAME = "customer.name"
    ISSUED_AT = "issuedAt"
    STATUS = "status"
    CREATED_AT = "createdAt"
    UPDATED_AT = "updatedAt"
    PERIOD_START = "periodStart"


class InvoiceStatus(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """InvoiceStatus describes the status of an invoice."""

    GATHERING = "gathering"
    """The list of line items for the next invoice is being gathered."""
    DRAFT = "draft"
    """The invoice is in draft status."""
    ISSUING = "issuing"
    """The invoice is in the process of being issued."""
    ISSUED = "issued"
    """The invoice has been issued to the customer."""
    PAYMENT_PROCESSING = "payment_processing"
    """The payment for the invoice is being processed."""
    OVERDUE = "overdue"
    """The invoice's payment is overdue."""
    PAID = "paid"
    """The invoice has been paid."""
    UNCOLLECTIBLE = "uncollectible"
    """The invoice has been marked uncollectible."""
    VOIDED = "voided"
    """The invoice has been voided."""


class InvoiceType(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """InvoiceType represents the type of invoice.

    The type of invoice determines the purpose of the invoice and how it should be handled.
    """

    STANDARD = "standard"
    """A regular commercial invoice document between a supplier and customer."""
    CREDIT_NOTE = "credit_note"
    """Reflects a refund either partial or complete of the preceding document. A credit note
    effectively *extends* the previous document."""


class MeasureUsageFromPreset(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """Start of measurement options."""

    CURRENT_PERIOD_START = "CURRENT_PERIOD_START"
    NOW = "NOW"


class MeterAggregation(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """The aggregation type to use for the meter."""

    SUM = "SUM"
    COUNT = "COUNT"
    UNIQUE_COUNT = "UNIQUE_COUNT"
    AVG = "AVG"
    MIN = "MIN"
    MAX = "MAX"
    LATEST = "LATEST"


class MeterOrderBy(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """Order by options for meters."""

    KEY = "key"
    NAME = "name"
    AGGREGATION = "aggregation"
    CREATED_AT = "createdAt"
    UPDATED_AT = "updatedAt"


class NotificationChannelOrderBy(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """Order by options for notification channels."""

    ID = "id"
    TYPE = "type"
    CREATED_AT = "createdAt"
    UPDATED_AT = "updatedAt"


class NotificationChannelType(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """Type of the notification channel."""

    WEBHOOK = "WEBHOOK"


class NotificationEventDeliveryStatusState(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """Delivery State."""

    SUCCESS = "SUCCESS"
    FAILED = "FAILED"
    SENDING = "SENDING"
    PENDING = "PENDING"
    RESENDING = "RESENDING"


class NotificationEventOrderBy(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """Order by options for notification channels."""

    ID = "id"
    CREATED_AT = "createdAt"


class NotificationEventType(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """Type of the notification event."""

    ENTITLEMENTS_BALANCE_THRESHOLD = "entitlements.balance.threshold"
    ENTITLEMENTS_RESET = "entitlements.reset"
    INVOICE_CREATED = "invoice.created"
    INVOICE_UPDATED = "invoice.updated"


class NotificationRuleBalanceThresholdValueType(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """Notification balance threshold type."""

    PERCENT = "PERCENT"
    NUMBER = "NUMBER"
    BALANCE_VALUE = "balance_value"
    USAGE_PERCENTAGE = "usage_percentage"
    USAGE_VALUE = "usage_value"


class NotificationRuleOrderBy(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """Order by options for notification channels."""

    ID = "id"
    TYPE = "type"
    CREATED_AT = "createdAt"
    UPDATED_AT = "updatedAt"


class OAuth2AuthorizationCodeGrantErrorType(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """OAuth2 authorization code grant error types."""

    INVALID_REQUEST = "invalid_request"
    """The request is missing a required parameter, includes an invalid parameter value,
    includes a parameter more than once, or is otherwise malformed."""
    UNAUTHORIZED_CLIENT = "unauthorized_client"
    """The client is not authorized to request an authorization code using this method."""
    ACCESS_DENIED = "access_denied"
    """The resource owner or authorization server denied the request."""
    UNSUPPORTED_RESPONSE_TYPE = "unsupported_response_type"
    """The authorization server does not support obtaining an authorization code using this method."""
    INVALID_SCOPE = "invalid_scope"
    """The requested scope is invalid, unknown, or malformed."""
    SERVER_ERROR = "server_error"
    """The authorization server encountered an unexpected condition that prevented it from fulfilling
    the request."""
    TEMPORARILY_UNAVAILABLE = "temporarily_unavailable"
    """The authorization server is currently unable to handle the request due to a temporary
    overloading or maintenance of the server."""


class PaymentTermType(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """PaymentTermType defines the type of terms to be applied."""

    DUE_DATE = "due_date"
    """Due on a specific date."""
    INSTANT = "instant"
    """On receipt of invoice"""


class PlanAddonOrderBy(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """Order by options for plan add-on assignments."""

    ID = "id"
    KEY = "key"
    VERSION = "version"
    CREATED_AT = "created_at"
    UPDATED_AT = "updated_at"


class PlanOrderBy(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """Order by options for plans."""

    ID = "id"
    KEY = "key"
    VERSION = "version"
    CREATED_AT = "created_at"
    UPDATED_AT = "updated_at"


class PlanStatus(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """The status of a plan."""

    DRAFT = "draft"
    ACTIVE = "active"
    ARCHIVED = "archived"
    SCHEDULED = "scheduled"


class PricePaymentTerm(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """The payment term of a flat price.
    One of: in_advance or in_arrears.
    """

    IN_ADVANCE = "in_advance"
    """If in_advance, the rate card will be invoiced in the previous billing cycle."""
    IN_ARREARS = "in_arrears"
    """If in_arrears, the rate card will be invoiced in the current billing cycle."""


class PriceType(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """The type of the price."""

    FLAT = "flat"
    UNIT = "unit"
    TIERED = "tiered"
    DYNAMIC = "dynamic"
    PACKAGE = "package"


class ProRatingMode(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """Pro-rating mode options for handling billing period changes."""

    PRORATE_PRICES = "prorate_prices"
    """Calculate pro-rated charges based on time remaining in billing period."""


class RateCardType(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """The type of the rate card."""

    FLAT_FEE = "flat_fee"
    USAGE_BASED = "usage_based"


class RecurringPeriodIntervalEnum(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """The unit of time for the interval.
    One of: ``day``, ``week``, ``month``, or ``year``.
    """

    DAY = "DAY"
    WEEK = "WEEK"
    MONTH = "MONTH"
    YEAR = "YEAR"


class RemovePhaseShifting(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """The direction of the phase shift when a phase is removed."""

    NEXT = "next"
    """Shifts all subsequent phases to start sooner by the deleted phase's length"""
    PREV = "prev"
    """Extends the previous phase to end later by the deleted phase's length"""


class SortOrder(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """The order direction."""

    ASC = "ASC"
    DESC = "DESC"


class StripeCheckoutSessionMode(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """Stripe CheckoutSession.mode."""

    SETUP = "setup"


class SubscriptionStatus(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """Subscription status."""

    ACTIVE = "active"
    INACTIVE = "inactive"
    CANCELED = "canceled"
    SCHEDULED = "scheduled"


class SubscriptionTimingEnum(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """Subscription edit timing.
    When immediate, the requested changes take effect immediately.
    When nextBillingCycle, the requested changes take effect at the next billing cycle.
    """

    IMMEDIATE = "immediate"
    NEXT_BILLING_CYCLE = "next_billing_cycle"


class TaxBehavior(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """Tax behavior.

    This enum is used to specify whether tax is included in the price or excluded from the price.
    """

    INCLUSIVE = "inclusive"
    """Tax is included in the price."""
    EXCLUSIVE = "exclusive"
    """Tax is excluded from the price."""


class TieredPriceMode(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """The mode of the tiered price."""

    VOLUME = "volume"
    GRADUATED = "graduated"


class ValidationIssueSeverity(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """ValidationIssueSeverity describes the severity of a validation issue.

    Issues with severity "critical" will prevent the invoice from being issued.
    """

    CRITICAL = "critical"
    WARNING = "warning"


class VoidInvoiceLineActionType(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """VoidInvoiceLineActionType describes how to handle the voidied line item in the invoice."""

    DISCARD = "discard"
    """The line items will never be charged for again"""
    PENDING = "pending"
    """Queue the line items into the pending state, they will be included in the next invoice. (We
    want to generate an invoice right now)"""


class WindowSize(str, Enum, metaclass=CaseInsensitiveEnumMeta):
    """Aggregation window size."""

    MINUTE = "MINUTE"
    HOUR = "HOUR"
    DAY = "DAY"
    MONTH = "MONTH"
