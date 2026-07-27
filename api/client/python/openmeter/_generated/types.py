# pylint: disable=too-many-lines
# coding=utf-8

from typing import Any, Literal, Optional, TYPE_CHECKING, Union
from typing_extensions import Required, TypedDict

from .models._enums import (
    AppType,
    BillingCollectionAlignment,
    EditOp,
    EntitlementType,
    FeatureUnitCostType,
    NotificationChannelType,
    NotificationEventType,
    PriceType,
    RateCardType,
    VoidInvoiceLineActionType,
)

if TYPE_CHECKING:
    from . import _unions
    from .models import (
        AddonInstanceType,
        AppCapabilityType,
        AppStatus,
        AppType,
        BillingSettlementMode,
        BillingWorkflowInvoicingSubscriptionEndProrationMode,
        CheckoutSessionUIMode,
        CollectionMethod,
        CreateCheckoutSessionTaxIdCollectionRequired,
        CreateStripeCheckoutSessionBillingAddressCollection,
        CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPosition,
        CreateStripeCheckoutSessionConsentCollectionPromotions,
        CreateStripeCheckoutSessionConsentCollectionTermsOfService,
        CreateStripeCheckoutSessionCustomerUpdateBehavior,
        CreateStripeCheckoutSessionRedirectOnCompletion,
        CustomInvoicingPaymentTrigger,
        ExpirationDuration,
        InstallMethod,
        MeterAggregation,
        NotificationRuleBalanceThresholdValueType,
        PricePaymentTerm,
        ProRatingMode,
        RemovePhaseShifting,
        TaxBehavior,
        TieredPriceMode,
        WindowSize,
    )


class AddonCreate(TypedDict, total=False):
    """Resource create operation model.

    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: "Metadata"
    :ivar key: Key. Required.
    :vartype key: str
    :ivar instance_type: InstanceType. Required. Known values are: "single" and "multiple".
    :vartype instance_type: Union[str, "AddonInstanceType"]
    :ivar currency: Currency. Required.
    :vartype currency: str
    :ivar rate_cards: Rate cards. Required.
    :vartype rate_cards: list["_unions.RateCard"]
    """

    name: Required[str]
    """Display name. Required."""
    description: str
    """Description."""
    metadata: Optional["Metadata"]
    """Metadata."""
    key: Required[str]
    """Key. Required."""
    instanceType: Required[Union[str, "AddonInstanceType"]]
    """InstanceType. Required. Known values are: \"single\" and \"multiple\"."""
    currency: Required[str]
    """Currency. Required."""
    rateCards: Required[list["_unions.RateCard"]]
    """Rate cards. Required."""


class AddonReplaceUpdate(TypedDict, total=False):
    """Resource update operation model.

    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: "Metadata"
    :ivar instance_type: InstanceType. Required. Known values are: "single" and "multiple".
    :vartype instance_type: Union[str, "AddonInstanceType"]
    :ivar rate_cards: Rate cards. Required.
    :vartype rate_cards: list["_unions.RateCard"]
    """

    name: Required[str]
    """Display name. Required."""
    description: str
    """Description."""
    metadata: Optional["Metadata"]
    """Metadata."""
    instanceType: Required[Union[str, "AddonInstanceType"]]
    """InstanceType. Required. Known values are: \"single\" and \"multiple\"."""
    rateCards: Required[list["_unions.RateCard"]]
    """Rate cards. Required."""


class Address(TypedDict, total=False):
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

    country: str
    """Country code in `ISO 3166-1 <https://www.iso.org/iso-3166-country-codes.html>`_ alpha-2 format."""
    postalCode: str
    """Postal code."""
    state: str
    """State or province."""
    city: str
    """City."""
    line1: str
    """First line of the address."""
    line2: str
    """Second line of the address."""
    phoneNumber: str
    """Phone number."""


class Alignment(TypedDict, total=False):
    """Alignment configuration for a plan or subscription.

    :ivar billables_must_align: Whether all Billable items and RateCards must align. Alignment
     means the Price's BillingCadence must align for both duration and anchor time.
    :vartype billables_must_align: bool
    """

    billablesMustAlign: bool
    """Whether all Billable items and RateCards must align. Alignment means the Price's BillingCadence
     must align for both duration and anchor time."""


class Annotations(TypedDict, total=False):
    """Set of key-value pairs managed by the system. Cannot be modified by user."""


class AppCapability(TypedDict, total=False):
    """App capability.

    Capabilities only exist in config so they don't extend the Resource model.

    :ivar type: The capability type. Required. Known values are: "reportUsage", "reportEvents",
     "calculateTax", "invoiceCustomers", and "collectPayments".
    :vartype type: Union[str, "AppCapabilityType"]
    :ivar key: Key. Required.
    :vartype key: str
    :ivar name: The capability name. Required.
    :vartype name: str
    :ivar description: The capability description. Required.
    :vartype description: str
    """

    type: Required[Union[str, "AppCapabilityType"]]
    """The capability type. Required. Known values are: \"reportUsage\", \"reportEvents\",
     \"calculateTax\", \"invoiceCustomers\", and \"collectPayments\"."""
    key: Required[str]
    """Key. Required."""
    name: Required[str]
    """The capability name. Required."""
    description: Required[str]
    """The capability description. Required."""


class BillingDiscountPercentage(TypedDict, total=False):
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

    percentage: Required[float]
    """Percentage. Required."""
    correlationId: str
    """Correlation ID for the discount.
     
     This is used to link discounts across different invoices (progressive billing use case).
     
     If not provided, the invoicing engine will auto-generate one. When editing an invoice line,
     please make sure to keep the same correlation ID of the discount or in progressive billing
     setups the discount amounts might be incorrect."""


class BillingDiscounts(TypedDict, total=False):
    """A discount by type.

    :ivar percentage: The percentage discount.
    :vartype percentage: "BillingDiscountPercentage"
    :ivar usage: The usage discount.
    :vartype usage: "BillingDiscountUsage"
    """

    percentage: "BillingDiscountPercentage"
    """The percentage discount."""
    usage: "BillingDiscountUsage"
    """The usage discount."""


class BillingDiscountUsage(TypedDict, total=False):
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

    quantity: Required[str]
    """Usage. Required."""
    correlationId: str
    """Correlation ID for the discount.
     
     This is used to link discounts across different invoices (progressive billing use case).
     
     If not provided, the invoicing engine will auto-generate one. When editing an invoice line,
     please make sure to keep the same correlation ID of the discount or in progressive billing
     setups the discount amounts might be incorrect."""


class BillingParty(TypedDict, total=False):
    """Party represents a person or business entity.

    :ivar id: Unique identifier for the party (if available).
    :vartype id: str
    :ivar key: Key.
    :vartype key: str
    :ivar name: Legal name or representation of the organization.
    :vartype name: str
    :ivar tax_id: The entity's legal ID code used for tax purposes. They may have other numbers,
     but we're only interested in those valid for tax purposes.
    :vartype tax_id: "BillingPartyTaxIdentity"
    :ivar addresses: Regular post addresses for where information should be sent if needed.
    :vartype addresses: list["Address"]
    """

    id: str
    """Unique identifier for the party (if available)."""
    key: str
    """Key."""
    name: str
    """Legal name or representation of the organization."""
    taxId: "BillingPartyTaxIdentity"
    """The entity's legal ID code used for tax purposes. They may have other numbers, but we're only
     interested in those valid for tax purposes."""
    addresses: list["Address"]
    """Regular post addresses for where information should be sent if needed."""


class BillingPartyReplaceUpdate(TypedDict, total=False):
    """Resource update operation model.

    :ivar key: Key.
    :vartype key: str
    :ivar name: Legal name or representation of the organization.
    :vartype name: str
    :ivar tax_id: The entity's legal ID code used for tax purposes. They may have other numbers,
     but we're only interested in those valid for tax purposes.
    :vartype tax_id: "BillingPartyTaxIdentity"
    :ivar addresses: Regular post addresses for where information should be sent if needed.
    :vartype addresses: list["Address"]
    """

    key: str
    """Key."""
    name: str
    """Legal name or representation of the organization."""
    taxId: "BillingPartyTaxIdentity"
    """The entity's legal ID code used for tax purposes. They may have other numbers, but we're only
     interested in those valid for tax purposes."""
    addresses: list["Address"]
    """Regular post addresses for where information should be sent if needed."""


class BillingPartyTaxIdentity(TypedDict, total=False):
    """Identity stores the details required to identify an entity for tax purposes in a specific
    country.

    :ivar code: Normalized tax code shown on the original identity document.
    :vartype code: str
    """

    code: str
    """Normalized tax code shown on the original identity document."""


class BillingProfileAppsCreate(TypedDict, total=False):
    """BillingProfileAppsCreate represents the input for creating a billing profile's apps.

    :ivar tax: The tax app used for this workflow. Required.
    :vartype tax: str
    :ivar invoicing: The invoicing app used for this workflow. Required.
    :vartype invoicing: str
    :ivar payment: The payment app used for this workflow. Required.
    :vartype payment: str
    """

    tax: Required[str]
    """The tax app used for this workflow. Required."""
    invoicing: Required[str]
    """The invoicing app used for this workflow. Required."""
    payment: Required[str]
    """The payment app used for this workflow. Required."""


class BillingProfileCreate(TypedDict, total=False):
    """BillingProfileCreate represents the input for creating a billing profile.

    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: "Metadata"
    :ivar supplier: The name and contact information for the supplier this billing profile
     represents. Required.
    :vartype supplier: "BillingParty"
    :ivar default: Is this the default profile?. Required.
    :vartype default: bool
    :ivar workflow: The billing workflow settings for this profile. Required.
    :vartype workflow: "BillingWorkflowCreate"
    :ivar apps: The apps used by this billing profile. Required.
    :vartype apps: "BillingProfileAppsCreate"
    """

    name: Required[str]
    """Display name. Required."""
    description: str
    """Description."""
    metadata: Optional["Metadata"]
    """Metadata."""
    supplier: Required["BillingParty"]
    """The name and contact information for the supplier this billing profile represents. Required."""
    default: Required[bool]
    """Is this the default profile?. Required."""
    workflow: Required["BillingWorkflowCreate"]
    """The billing workflow settings for this profile. Required."""
    apps: Required["BillingProfileAppsCreate"]
    """The apps used by this billing profile. Required."""


class BillingProfileCustomerOverrideCreate(TypedDict, total=False):
    """Payload for creating a new or updating an existing customer override.

    :ivar billing_profile_id: The billing profile this override is associated with.

     If not provided, the default billing profile is chosen if available.
    :vartype billing_profile_id: str
    """

    billingProfileId: str
    """The billing profile this override is associated with.
     
     If not provided, the default billing profile is chosen if available."""


class BillingProfileReplaceUpdateWithWorkflow(TypedDict, total=False):
    """BillingProfileReplaceUpdate represents the input for updating a billing profile

    The apps field cannot be updated directly, if an app change is desired a new
    profile should be created.

    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: "Metadata"
    :ivar supplier: The name and contact information for the supplier this billing profile
     represents. Required.
    :vartype supplier: "BillingParty"
    :ivar default: Is this the default profile?. Required.
    :vartype default: bool
    :ivar workflow: The billing workflow settings for this profile. Required.
    :vartype workflow: "BillingWorkflow"
    """

    name: Required[str]
    """Display name. Required."""
    description: str
    """Description."""
    metadata: Optional["Metadata"]
    """Metadata."""
    supplier: Required["BillingParty"]
    """The name and contact information for the supplier this billing profile represents. Required."""
    default: Required[bool]
    """Is this the default profile?. Required."""
    workflow: Required["BillingWorkflow"]
    """The billing workflow settings for this profile. Required."""


class BillingWorkflow(TypedDict, total=False):
    """BillingWorkflow represents the settings for a billing workflow.

    :ivar collection: The collection settings for this workflow.
    :vartype collection: "BillingWorkflowCollectionSettings"
    :ivar invoicing: The invoicing settings for this workflow.
    :vartype invoicing: "BillingWorkflowInvoicingSettings"
    :ivar payment: The payment settings for this workflow.
    :vartype payment: "BillingWorkflowPaymentSettings"
    :ivar tax: The tax settings for this workflow.
    :vartype tax: "BillingWorkflowTaxSettings"
    """

    collection: "BillingWorkflowCollectionSettings"
    """The collection settings for this workflow."""
    invoicing: "BillingWorkflowInvoicingSettings"
    """The invoicing settings for this workflow."""
    payment: "BillingWorkflowPaymentSettings"
    """The payment settings for this workflow."""
    tax: "BillingWorkflowTaxSettings"
    """The tax settings for this workflow."""


class BillingWorkflowCollectionAlignmentAnchored(TypedDict, total=False):  # pylint: disable=name-too-long
    """BillingWorkflowCollectionAlignmentAnchored specifies the alignment for collecting the pending
    line items into an invoice.

    :ivar type: The type of alignment. Required. Align the collection to the anchor time and
     cadence.
    :vartype type: Literal[BillingCollectionAlignment.ANCHORED]
    :ivar recurring_period: The recurring period for the alignment. Required.
    :vartype recurring_period: "RecurringPeriodV2"
    """

    type: Required[Literal[BillingCollectionAlignment.ANCHORED]]
    """The type of alignment. Required. Align the collection to the anchor time and cadence."""
    recurringPeriod: Required["RecurringPeriodV2"]
    """The recurring period for the alignment. Required."""


class BillingWorkflowCollectionAlignmentSubscription(TypedDict, total=False):  # pylint: disable=name-too-long
    """BillingWorkflowCollectionAlignmentSubscription specifies the alignment for collecting the
    pending line items into an invoice.

    :ivar type: The type of alignment. Required. Align the collection to the start of the
     subscription period.
    :vartype type: Literal[BillingCollectionAlignment.SUBSCRIPTION]
    """

    type: Required[Literal[BillingCollectionAlignment.SUBSCRIPTION]]
    """The type of alignment. Required. Align the collection to the start of the subscription period."""


class BillingWorkflowCollectionSettings(TypedDict, total=False):
    """Workflow collection specifies how to collect the pending line items for an invoice.

    :ivar alignment: The alignment for collecting the pending line items into an invoice. Is either
     a BillingWorkflowCollectionAlignmentSubscription type or a
     BillingWorkflowCollectionAlignmentAnchored type.
    :vartype alignment: "_unions.BillingWorkflowCollectionAlignment"
    :ivar interval: This grace period can be used to delay the collection of the pending line items
     specified in
     alignment.

     This is useful, in case of multiple subscriptions having slightly different billing periods.
    :vartype interval: str
    """

    alignment: "_unions.BillingWorkflowCollectionAlignment"
    """The alignment for collecting the pending line items into an invoice. Is either a
     BillingWorkflowCollectionAlignmentSubscription type or a
     BillingWorkflowCollectionAlignmentAnchored type."""
    interval: str
    """This grace period can be used to delay the collection of the pending line items specified in
     alignment.
     
     This is useful, in case of multiple subscriptions having slightly different billing periods."""


class BillingWorkflowCreate(TypedDict, total=False):
    """Resource create operation model.

    :ivar collection: The collection settings for this workflow.
    :vartype collection: "BillingWorkflowCollectionSettings"
    :ivar invoicing: The invoicing settings for this workflow.
    :vartype invoicing: "BillingWorkflowInvoicingSettings"
    :ivar payment: The payment settings for this workflow.
    :vartype payment: "BillingWorkflowPaymentSettings"
    :ivar tax: The tax settings for this workflow.
    :vartype tax: "BillingWorkflowTaxSettings"
    """

    collection: "BillingWorkflowCollectionSettings"
    """The collection settings for this workflow."""
    invoicing: "BillingWorkflowInvoicingSettings"
    """The invoicing settings for this workflow."""
    payment: "BillingWorkflowPaymentSettings"
    """The payment settings for this workflow."""
    tax: "BillingWorkflowTaxSettings"
    """The tax settings for this workflow."""


class BillingWorkflowInvoicingSettings(TypedDict, total=False):
    """Workflow invoice settings.

    :ivar auto_advance: Whether to automatically issue the invoice after the draftPeriod has
     passed.
    :vartype auto_advance: bool
    :ivar draft_period: The period for the invoice to be kept in draft status for manual reviews.
    :vartype draft_period: str
    :ivar due_after: The period after which the invoice is due. With some payment solutions it's
     only applicable for manual collection method.
    :vartype due_after: str
    :ivar progressive_billing: Should progressive billing be allowed for this workflow?.
    :vartype progressive_billing: bool
    :ivar subscription_end_proration_mode: Controls how subscription-ending shortened service
     periods are billed. Known values are: "bill_full_period" and "bill_actual_period".
    :vartype subscription_end_proration_mode: Union[str,
     "BillingWorkflowInvoicingSubscriptionEndProrationMode"]
    :ivar default_tax_config: Default tax configuration to apply to the invoices.

     Setting a tax code (``stripe.code`` / ``taxCodeId``) on a profile's default tax config is
     deprecated and can no longer be added or changed: the organization default tax code is
     used instead. Existing tax-code values may still be removed, and ``behavior`` remains
     fully supported.
    :vartype default_tax_config: "TaxConfig"
    """

    autoAdvance: bool
    """Whether to automatically issue the invoice after the draftPeriod has passed."""
    draftPeriod: str
    """The period for the invoice to be kept in draft status for manual reviews."""
    dueAfter: str
    """The period after which the invoice is due. With some payment solutions it's only applicable for
     manual collection method."""
    progressiveBilling: bool
    """Should progressive billing be allowed for this workflow?."""
    subscriptionEndProrationMode: Union[str, "BillingWorkflowInvoicingSubscriptionEndProrationMode"]
    """Controls how subscription-ending shortened service periods are billed. Known values are:
     \"bill_full_period\" and \"bill_actual_period\"."""
    defaultTaxConfig: "TaxConfig"
    """Default tax configuration to apply to the invoices.
     
     Setting a tax code (``stripe.code`` / ``taxCodeId``) on a profile's default tax config is
     deprecated and can no longer be added or changed: the organization default tax code is
     used instead. Existing tax-code values may still be removed, and ``behavior`` remains
     fully supported."""


class BillingWorkflowPaymentSettings(TypedDict, total=False):
    """Workflow payment settings.

    :ivar collection_method: The payment method for the invoice. Known values are:
     "charge_automatically" and "send_invoice".
    :vartype collection_method: Union[str, "CollectionMethod"]
    """

    collectionMethod: Union[str, "CollectionMethod"]
    """The payment method for the invoice. Known values are: \"charge_automatically\" and
     \"send_invoice\"."""


class BillingWorkflowTaxSettings(TypedDict, total=False):
    """Workflow tax settings.

    :ivar enabled: Enable automatic tax calculation when tax is supported by the app. For example,
     with Stripe Invoicing when enabled, tax is calculated via Stripe Tax.
    :vartype enabled: bool
    :ivar enforced: Enforce tax calculation when tax is supported by the app. When enabled,
     OpenMeter will not allow to create an invoice without tax calculation. Enforcement is different
     per apps, for example, Stripe app requires customer to have a tax location when starting a paid
     subscription.
    :vartype enforced: bool
    """

    enabled: bool
    """Enable automatic tax calculation when tax is supported by the app. For example, with Stripe
     Invoicing when enabled, tax is calculated via Stripe Tax."""
    enforced: bool
    """Enforce tax calculation when tax is supported by the app. When enabled, OpenMeter will not
     allow to create an invoice without tax calculation. Enforcement is different per apps, for
     example, Stripe app requires customer to have a tax location when starting a paid subscription."""


class CancelRequest(TypedDict, total=False):
    """CancelRequest.

    :ivar timing: If not provided the subscription is canceled immediately. Is either a Union[str,
     "_models.SubscriptionTimingEnum"] type or a datetime.datetime type.
    :vartype timing: "_unions.SubscriptionTiming"
    """

    timing: "_unions.SubscriptionTiming"
    """If not provided the subscription is canceled immediately. Is either a Union[str,
     \"_models.SubscriptionTimingEnum\"] type or a datetime.datetime type."""


class CheckoutSessionCustomTextAfterSubmitParams(TypedDict, total=False):  # pylint: disable=name-too-long
    """Stripe CheckoutSession.custom_text.

    :ivar after_submit: Custom text that should be displayed after the payment confirmation button.
    :vartype after_submit: "CheckoutSessionCustomTextParamsAfterSubmit"
    :ivar shipping_address: Custom text that should be displayed alongside shipping address
     collection.
    :vartype shipping_address: "CheckoutSessionCustomTextParamsShippingAddress"
    :ivar submit: Custom text that should be displayed alongside the payment confirmation button.
    :vartype submit: "CheckoutSessionCustomTextParamsSubmit"
    :ivar terms_of_service_acceptance: Custom text that should be displayed in place of the default
     terms of service agreement text.
    :vartype terms_of_service_acceptance: "CheckoutSessionCustomTextParamsTermsOfServiceAcceptance"
    """

    afterSubmit: "CheckoutSessionCustomTextParamsAfterSubmit"
    """Custom text that should be displayed after the payment confirmation button."""
    shippingAddress: "CheckoutSessionCustomTextParamsShippingAddress"
    """Custom text that should be displayed alongside shipping address collection."""
    submit: "CheckoutSessionCustomTextParamsSubmit"
    """Custom text that should be displayed alongside the payment confirmation button."""
    termsOfServiceAcceptance: "CheckoutSessionCustomTextParamsTermsOfServiceAcceptance"
    """Custom text that should be displayed in place of the default terms of service agreement text."""


class CheckoutSessionCustomTextParamsAfterSubmit(TypedDict, total=False):  # pylint: disable=name-too-long
    """CheckoutSessionCustomTextParamsAfterSubmit.

    :ivar message:
    :vartype message: str
    """

    message: str


class CheckoutSessionCustomTextParamsShippingAddress(TypedDict, total=False):  # pylint: disable=name-too-long
    """CheckoutSessionCustomTextParamsShippingAddress.

    :ivar message:
    :vartype message: str
    """

    message: str


class CheckoutSessionCustomTextParamsSubmit(TypedDict, total=False):
    """CheckoutSessionCustomTextParamsSubmit.

    :ivar message:
    :vartype message: str
    """

    message: str


class CheckoutSessionCustomTextParamsTermsOfServiceAcceptance(TypedDict, total=False):  # pylint: disable=name-too-long
    """CheckoutSessionCustomTextParamsTermsOfServiceAcceptance.

    :ivar message:
    :vartype message: str
    """

    message: str


class CreateCheckoutSessionTaxIdCollection(TypedDict, total=False):
    """Create Stripe checkout session tax ID collection.

    :ivar enabled: Enable tax ID collection during checkout. Defaults to false. Required.
    :vartype enabled: bool
    :ivar required: Describes whether a tax ID is required during checkout. Defaults to never.
     Known values are: "if_supported" and "never".
    :vartype required: Union[str, "CreateCheckoutSessionTaxIdCollectionRequired"]
    """

    enabled: Required[bool]
    """Enable tax ID collection during checkout. Defaults to false. Required."""
    required: Union[str, "CreateCheckoutSessionTaxIdCollectionRequired"]
    """Describes whether a tax ID is required during checkout. Defaults to never. Known values are:
     \"if_supported\" and \"never\"."""


class CreateStripeCheckoutSessionConsentCollection(TypedDict, total=False):  # pylint: disable=name-too-long
    """Configure fields for the Checkout Session to gather active consent from customers.

    :ivar payment_method_reuse_agreement: Determines the position and visibility of the payment
     method reuse agreement in the UI. When set to auto, Stripe’s defaults will be used. When set to
     hidden, the payment method reuse agreement text will always be hidden in the UI.
    :vartype payment_method_reuse_agreement:
     "CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreement"
    :ivar promotions: If set to auto, enables the collection of customer consent for promotional
     communications. The Checkout Session will determine whether to display an option to opt into
     promotional communication from the merchant depending on the customer’s locale. Only available
     to US merchants. Known values are: "auto" and "none".
    :vartype promotions: Union[str, "CreateStripeCheckoutSessionConsentCollectionPromotions"]
    :ivar terms_of_service: If set to required, it requires customers to check a terms of service
     checkbox before being able to pay. There must be a valid terms of service URL set in your
     Stripe Dashboard settings. `https://dashboard.stripe.com/settings/public
     <https://dashboard.stripe.com/settings/public>`_. Known values are: "none" and "required".
    :vartype terms_of_service: Union[str,
     "CreateStripeCheckoutSessionConsentCollectionTermsOfService"]
    """

    paymentMethodReuseAgreement: "CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreement"
    """Determines the position and visibility of the payment method reuse agreement in the UI. When
     set to auto, Stripe’s defaults will be used. When set to hidden, the payment method reuse
     agreement text will always be hidden in the UI."""
    promotions: Union[str, "CreateStripeCheckoutSessionConsentCollectionPromotions"]
    """If set to auto, enables the collection of customer consent for promotional communications. The
     Checkout Session will determine whether to display an option to opt into promotional
     communication from the merchant depending on the customer’s locale. Only available to US
     merchants. Known values are: \"auto\" and \"none\"."""
    termsOfService: Union[str, "CreateStripeCheckoutSessionConsentCollectionTermsOfService"]
    """If set to required, it requires customers to check a terms of service checkbox before being
     able to pay. There must be a valid terms of service URL set in your Stripe Dashboard settings.
     `https://dashboard.stripe.com/settings/public <https://dashboard.stripe.com/settings/public>`_.
     Known values are: \"none\" and \"required\"."""


class CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreement(
    TypedDict, total=False
):  # pylint: disable=name-too-long
    """Create Stripe checkout session payment method reuse agreement.

    :ivar position: Known values are: "auto" and "hidden".
    :vartype position: Union[str,
     "CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPosition"]
    """

    position: Union[str, "CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPosition"]
    """Known values are: \"auto\" and \"hidden\"."""


class CreateStripeCheckoutSessionCustomerUpdate(TypedDict, total=False):  # pylint: disable=name-too-long
    """Controls what fields on Customer can be updated by the Checkout Session.

    :ivar address: Describes whether Checkout saves the billing address onto customer.address. To
     always collect a full billing address, use billing_address_collection. Defaults to never. Known
     values are: "auto" and "never".
    :vartype address: Union[str, "CreateStripeCheckoutSessionCustomerUpdateBehavior"]
    :ivar name: Describes whether Checkout saves the name onto customer.name. Defaults to never.
     Known values are: "auto" and "never".
    :vartype name: Union[str, "CreateStripeCheckoutSessionCustomerUpdateBehavior"]
    :ivar shipping: Describes whether Checkout saves shipping information onto customer.shipping.
     To collect shipping information, use shipping_address_collection. Defaults to never. Known
     values are: "auto" and "never".
    :vartype shipping: Union[str, "CreateStripeCheckoutSessionCustomerUpdateBehavior"]
    """

    address: Union[str, "CreateStripeCheckoutSessionCustomerUpdateBehavior"]
    """Describes whether Checkout saves the billing address onto customer.address. To always collect a
     full billing address, use billing_address_collection. Defaults to never. Known values are:
     \"auto\" and \"never\"."""
    name: Union[str, "CreateStripeCheckoutSessionCustomerUpdateBehavior"]
    """Describes whether Checkout saves the name onto customer.name. Defaults to never. Known values
     are: \"auto\" and \"never\"."""
    shipping: Union[str, "CreateStripeCheckoutSessionCustomerUpdateBehavior"]
    """Describes whether Checkout saves shipping information onto customer.shipping. To collect
     shipping information, use shipping_address_collection. Defaults to never. Known values are:
     \"auto\" and \"never\"."""


class CreateStripeCheckoutSessionRequest(TypedDict, total=False):
    """Create Stripe checkout session request.

    :ivar app_id: If not provided, the default Stripe app is used if any.
    :vartype app_id: str
    :ivar customer: Provide a customer ID or key to use an existing OpenMeter customer. or provide
     a customer object to create a new customer. Required. Is one of the following types:
     CustomerId, CustomerKey, CustomerCreate
    :vartype customer: Union["CustomerId", "CustomerKey", "CustomerCreate"]
    :ivar stripe_customer_id: Stripe customer ID. If not provided OpenMeter creates a new Stripe
     customer or uses the OpenMeter customer's default Stripe customer ID.
    :vartype stripe_customer_id: str
    :ivar options: Options passed to Stripe when creating the checkout session. Required.
    :vartype options: "CreateStripeCheckoutSessionRequestOptions"
    """

    appId: str
    """If not provided, the default Stripe app is used if any."""
    customer: Required[Union["CustomerId", "CustomerKey", "CustomerCreate"]]
    """Provide a customer ID or key to use an existing OpenMeter customer. or provide a customer
     object to create a new customer. Required. Is one of the following types: CustomerId,
     CustomerKey, CustomerCreate"""
    stripeCustomerId: str
    """Stripe customer ID. If not provided OpenMeter creates a new Stripe customer or uses the
     OpenMeter customer's default Stripe customer ID."""
    options: Required["CreateStripeCheckoutSessionRequestOptions"]
    """Options passed to Stripe when creating the checkout session. Required."""


class CreateStripeCheckoutSessionRequestOptions(TypedDict, total=False):  # pylint: disable=name-too-long
    """Create Stripe checkout session options See
    `https://docs.stripe.com/api/checkout/sessions/create
    <https://docs.stripe.com/api/checkout/sessions/create>`_.

    :ivar billing_address_collection: Specify whether Checkout should collect the customer’s
     billing address. Defaults to auto. Known values are: "auto" and "required".
    :vartype billing_address_collection: Union[str,
     "CreateStripeCheckoutSessionBillingAddressCollection"]
    :ivar cancel_url: If set, Checkout displays a back button and customers will be directed to
     this URL if they decide to cancel payment and return to your website. This parameter is not
     allowed if ui_mode is embedded.
    :vartype cancel_url: str
    :ivar client_reference_id: A unique string to reference the Checkout Session. This can be a
     customer ID, a cart ID, or similar, and can be used to reconcile the session with your internal
     systems.
    :vartype client_reference_id: str
    :ivar customer_update: Controls what fields on Customer can be updated by the Checkout Session.
    :vartype customer_update: "CreateStripeCheckoutSessionCustomerUpdate"
    :ivar consent_collection: Configure fields for the Checkout Session to gather active consent
     from customers.
    :vartype consent_collection: "CreateStripeCheckoutSessionConsentCollection"
    :ivar currency: Three-letter ISO currency code, in lowercase.
    :vartype currency: str
    :ivar custom_text: Display additional text for your customers using custom text.
    :vartype custom_text: "CheckoutSessionCustomTextAfterSubmitParams"
    :ivar expires_at: The Epoch time in seconds at which the Checkout Session will expire. It can
     be anywhere from 30 minutes to 24 hours after Checkout Session creation. By default, this value
     is 24 hours from creation.
    :vartype expires_at: int
    :ivar locale:
    :vartype locale: str
    :ivar metadata: Set of key-value pairs that you can attach to an object. This can be useful for
     storing additional information about the object in a structured format. Individual keys can be
     unset by posting an empty value to them. All keys can be unset by posting an empty value to
     metadata.
    :vartype metadata: dict[str, str]
    :ivar return_url: The URL to redirect your customer back to after they authenticate or cancel
     their payment on the payment method’s app or site. This parameter is required if ui_mode is
     embedded and redirect-based payment methods are enabled on the session.
    :vartype return_url: str
    :ivar success_url: The URL to which Stripe should send customers when payment or setup is
     complete. This parameter is not allowed if ui_mode is embedded. If you’d like to use
     information from the successful Checkout Session on your page, read the guide on customizing
     your success page: `https://docs.stripe.com/payments/checkout/custom-success-page
     <https://docs.stripe.com/payments/checkout/custom-success-page>`_.
    :vartype success_url: str
    :ivar ui_mode: The UI mode of the Session. Defaults to hosted. Known values are: "embedded" and
     "hosted".
    :vartype ui_mode: Union[str, "CheckoutSessionUIMode"]
    :ivar payment_method_types: A list of the types of payment methods (e.g., card) this Checkout
     Session can accept.
    :vartype payment_method_types: list[str]
    :ivar redirect_on_completion: This parameter applies to ui_mode: embedded. Defaults to always.
     Learn more about the redirect behavior of embedded sessions at
     `https://docs.stripe.com/payments/checkout/custom-success-page?payment-ui=embedded-form
     <https://docs.stripe.com/payments/checkout/custom-success-page?payment-ui=embedded-form>`_.
     Known values are: "always", "if_required", and "never".
    :vartype redirect_on_completion: Union[str, "CreateStripeCheckoutSessionRedirectOnCompletion"]
    :ivar tax_id_collection: Controls tax ID collection during checkout.
    :vartype tax_id_collection: "CreateCheckoutSessionTaxIdCollection"
    """

    billingAddressCollection: Union[str, "CreateStripeCheckoutSessionBillingAddressCollection"]
    """Specify whether Checkout should collect the customer’s billing address. Defaults to auto. Known
     values are: \"auto\" and \"required\"."""
    cancelURL: str
    """If set, Checkout displays a back button and customers will be directed to this URL if they
     decide to cancel payment and return to your website. This parameter is not allowed if ui_mode
     is embedded."""
    clientReferenceID: str
    """A unique string to reference the Checkout Session. This can be a customer ID, a cart ID, or
     similar, and can be used to reconcile the session with your internal systems."""
    customerUpdate: "CreateStripeCheckoutSessionCustomerUpdate"
    """Controls what fields on Customer can be updated by the Checkout Session."""
    consentCollection: "CreateStripeCheckoutSessionConsentCollection"
    """Configure fields for the Checkout Session to gather active consent from customers."""
    currency: str
    """Three-letter ISO currency code, in lowercase."""
    customText: "CheckoutSessionCustomTextAfterSubmitParams"
    """Display additional text for your customers using custom text."""
    expiresAt: int
    """The Epoch time in seconds at which the Checkout Session will expire. It can be anywhere from 30
     minutes to 24 hours after Checkout Session creation. By default, this value is 24 hours from
     creation."""
    locale: str
    metadata: dict[str, str]
    """Set of key-value pairs that you can attach to an object. This can be useful for storing
     additional information about the object in a structured format. Individual keys can be unset by
     posting an empty value to them. All keys can be unset by posting an empty value to metadata."""
    returnURL: str
    """The URL to redirect your customer back to after they authenticate or cancel their payment on
     the payment method’s app or site. This parameter is required if ui_mode is embedded and
     redirect-based payment methods are enabled on the session."""
    successURL: str
    """The URL to which Stripe should send customers when payment or setup is complete. This parameter
     is not allowed if ui_mode is embedded. If you’d like to use information from the successful
     Checkout Session on your page, read the guide on customizing your success page:
     `https://docs.stripe.com/payments/checkout/custom-success-page
     <https://docs.stripe.com/payments/checkout/custom-success-page>`_."""
    uiMode: Union[str, "CheckoutSessionUIMode"]
    """The UI mode of the Session. Defaults to hosted. Known values are: \"embedded\" and \"hosted\"."""
    paymentMethodTypes: list[str]
    """A list of the types of payment methods (e.g., card) this Checkout Session can accept."""
    redirectOnCompletion: Union[str, "CreateStripeCheckoutSessionRedirectOnCompletion"]
    """This parameter applies to ui_mode: embedded. Defaults to always. Learn more about the redirect
     behavior of embedded sessions at
     `https://docs.stripe.com/payments/checkout/custom-success-page?payment-ui=embedded-form
     <https://docs.stripe.com/payments/checkout/custom-success-page?payment-ui=embedded-form>`_.
     Known values are: \"always\", \"if_required\", and \"never\"."""
    taxIdCollection: "CreateCheckoutSessionTaxIdCollection"
    """Controls tax ID collection during checkout."""


class CreateStripeCustomerPortalSessionParams(TypedDict, total=False):
    """Stripe customer portal request params.

    :ivar configuration_id: Configuration.
    :vartype configuration_id: str
    :ivar locale: Locale.
    :vartype locale: str
    :ivar return_url: ReturnUrl.
    :vartype return_url: str
    """

    configurationId: str
    """Configuration."""
    locale: str
    """Locale."""
    returnUrl: str
    """ReturnUrl."""


class CustomerCreate(TypedDict, total=False):
    """Resource create operation model.

    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: "Metadata"
    :ivar key: Key.
    :vartype key: str
    :ivar usage_attribution: Usage Attribution.
    :vartype usage_attribution: "CustomerUsageAttribution"
    :ivar primary_email: Primary Email.
    :vartype primary_email: str
    :ivar currency: Currency.
    :vartype currency: str
    :ivar billing_address: Billing Address.
    :vartype billing_address: "Address"
    """

    name: Required[str]
    """Display name. Required."""
    description: str
    """Description."""
    metadata: Optional["Metadata"]
    """Metadata."""
    key: str
    """Key."""
    usageAttribution: "CustomerUsageAttribution"
    """Usage Attribution."""
    primaryEmail: str
    """Primary Email."""
    currency: str
    """Currency."""
    billingAddress: "Address"
    """Billing Address."""


class CustomerId(TypedDict, total=False):
    """Create Stripe checkout session with customer ID.

    :ivar id: Required.
    :vartype id: str
    """

    id: Required[str]
    """Required."""


class CustomerKey(TypedDict, total=False):
    """Create Stripe checkout session with customer key.

    :ivar key: Required.
    :vartype key: str
    """

    key: Required[str]
    """Required."""


class CustomerReplaceUpdate(TypedDict, total=False):
    """Resource update operation model.

    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: "Metadata"
    :ivar key: Key.
    :vartype key: str
    :ivar usage_attribution: Usage Attribution.
    :vartype usage_attribution: "CustomerUsageAttribution"
    :ivar primary_email: Primary Email.
    :vartype primary_email: str
    :ivar currency: Currency.
    :vartype currency: str
    :ivar billing_address: Billing Address.
    :vartype billing_address: "Address"
    """

    name: Required[str]
    """Display name. Required."""
    description: str
    """Description."""
    metadata: Optional["Metadata"]
    """Metadata."""
    key: str
    """Key."""
    usageAttribution: "CustomerUsageAttribution"
    """Usage Attribution."""
    primaryEmail: str
    """Primary Email."""
    currency: str
    """Currency."""
    billingAddress: "Address"
    """Billing Address."""


class CustomerUsageAttribution(TypedDict, total=False):
    """Mapping to attribute metered usage to the customer. One customer can have zero or more
    subjects, but one subject can only belong to one customer.

    :ivar subject_keys: SubjectKeys. Required.
    :vartype subject_keys: list[str]
    """

    subjectKeys: Required[list[str]]
    """SubjectKeys. Required."""


class CustomInvoicingApp(TypedDict, total=False):
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
    :vartype metadata: "Metadata"
    :ivar created_at: Creation Time. Required.
    :vartype created_at: str
    :ivar updated_at: Last Update Time. Required.
    :vartype updated_at: str
    :ivar deleted_at: Deletion Time.
    :vartype deleted_at: str
    :ivar listing: The marketplace listing that this installed app is based on. Required.
    :vartype listing: "MarketplaceListing"
    :ivar status: Status of the app connection. Required. Known values are: "ready" and
     "unauthorized".
    :vartype status: Union[str, "AppStatus"]
    :ivar type: The app's type is CustomInvoicing. Required. CUSTOM_INVOICING.
    :vartype type: Literal[AppType.CUSTOM_INVOICING]
    :ivar enable_draft_sync_hook: Enable draft.sync hook.

     If the hook is not enabled, the invoice will be progressed to the next state automatically.
     Required.
    :vartype enable_draft_sync_hook: bool
    :ivar enable_issuing_sync_hook: Enable issuing.sync hook.

     If the hook is not enabled, the invoice will be progressed to the next state automatically.
     Required.
    :vartype enable_issuing_sync_hook: bool
    """

    id: Required[str]
    """ID. Required."""
    name: Required[str]
    """Display name. Required."""
    description: str
    """Description."""
    metadata: Optional["Metadata"]
    """Metadata."""
    createdAt: Required[str]
    """Creation Time. Required."""
    updatedAt: Required[str]
    """Last Update Time. Required."""
    deletedAt: str
    """Deletion Time."""
    listing: Required["MarketplaceListing"]
    """The marketplace listing that this installed app is based on. Required."""
    status: Required[Union[str, "AppStatus"]]
    """Status of the app connection. Required. Known values are: \"ready\" and \"unauthorized\"."""
    type: Required[Literal[AppType.CUSTOM_INVOICING]]
    """The app's type is CustomInvoicing. Required. CUSTOM_INVOICING."""
    enableDraftSyncHook: Required[bool]
    """Enable draft.sync hook.
     
     If the hook is not enabled, the invoice will be progressed to the next state automatically.
     Required."""
    enableIssuingSyncHook: Required[bool]
    """Enable issuing.sync hook.
     
     If the hook is not enabled, the invoice will be progressed to the next state automatically.
     Required."""


class CustomInvoicingAppReplaceUpdate(TypedDict, total=False):
    """Resource update operation model.

    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: "Metadata"
    :ivar type: The app's type is CustomInvoicing. Required. CUSTOM_INVOICING.
    :vartype type: Literal[AppType.CUSTOM_INVOICING]
    :ivar enable_draft_sync_hook: Enable draft.sync hook.

     If the hook is not enabled, the invoice will be progressed to the next state automatically.
     Required.
    :vartype enable_draft_sync_hook: bool
    :ivar enable_issuing_sync_hook: Enable issuing.sync hook.

     If the hook is not enabled, the invoice will be progressed to the next state automatically.
     Required.
    :vartype enable_issuing_sync_hook: bool
    """

    name: Required[str]
    """Display name. Required."""
    description: str
    """Description."""
    metadata: Optional["Metadata"]
    """Metadata."""
    type: Required[Literal[AppType.CUSTOM_INVOICING]]
    """The app's type is CustomInvoicing. Required. CUSTOM_INVOICING."""
    enableDraftSyncHook: Required[bool]
    """Enable draft.sync hook.
     
     If the hook is not enabled, the invoice will be progressed to the next state automatically.
     Required."""
    enableIssuingSyncHook: Required[bool]
    """Enable issuing.sync hook.
     
     If the hook is not enabled, the invoice will be progressed to the next state automatically.
     Required."""


class CustomInvoicingCustomerAppData(TypedDict, total=False):
    """Custom Invoicing Customer App Data.

    :ivar app: The installed custom invoicing app this data belongs to.
    :vartype app: "CustomInvoicingApp"
    :ivar id: App ID.
    :vartype id: str
    :ivar type: App Type. Required. CUSTOM_INVOICING.
    :vartype type: Literal[AppType.CUSTOM_INVOICING]
    :ivar metadata: Metadata to be used by the custom invoicing provider.
    :vartype metadata: "Metadata"
    """

    app: "CustomInvoicingApp"
    """The installed custom invoicing app this data belongs to."""
    id: str
    """App ID."""
    type: Required[Literal[AppType.CUSTOM_INVOICING]]
    """App Type. Required. CUSTOM_INVOICING."""
    metadata: "Metadata"
    """Metadata to be used by the custom invoicing provider."""


class CustomInvoicingDraftSynchronizedRequest(TypedDict, total=False):
    """Information to finalize the draft details of an invoice.

    :ivar invoicing: The result of the synchronization.
    :vartype invoicing: "CustomInvoicingSyncResult"
    """

    invoicing: "CustomInvoicingSyncResult"
    """The result of the synchronization."""


class CustomInvoicingFinalizedInvoicingRequest(TypedDict, total=False):
    """Information to finalize the invoicing details of an invoice.

    :ivar invoice_number: If set the invoice's number will be set to this value.
    :vartype invoice_number: str
    :ivar sent_to_customer_at: If set the invoice's sent to customer at will be set to this value.
    :vartype sent_to_customer_at: str
    """

    invoiceNumber: str
    """If set the invoice's number will be set to this value."""
    sentToCustomerAt: str
    """If set the invoice's sent to customer at will be set to this value."""


class CustomInvoicingFinalizedPaymentRequest(TypedDict, total=False):
    """Information to finalize the payment details of an invoice.

    :ivar external_id: If set the invoice's payment external ID will be set to this value.
    :vartype external_id: str
    """

    externalId: str
    """If set the invoice's payment external ID will be set to this value."""


class CustomInvoicingFinalizedRequest(TypedDict, total=False):
    """Information to finalize the invoice.

    If invoicing.invoiceNumber is not set, then a new invoice number will be generated (INV-
    prefix).

    :ivar invoicing: The result of the synchronization.
    :vartype invoicing: "CustomInvoicingFinalizedInvoicingRequest"
    :ivar payment: The result of the payment synchronization.
    :vartype payment: "CustomInvoicingFinalizedPaymentRequest"
    """

    invoicing: "CustomInvoicingFinalizedInvoicingRequest"
    """The result of the synchronization."""
    payment: "CustomInvoicingFinalizedPaymentRequest"
    """The result of the payment synchronization."""


class CustomInvoicingLineDiscountExternalIdMapping(TypedDict, total=False):  # pylint: disable=name-too-long
    """Mapping between line discounts and external IDs.

    :ivar line_discount_id: The line discount ID. Required.
    :vartype line_discount_id: str
    :ivar external_id: The external ID (e.g. custom invoicing system's ID). Required.
    :vartype external_id: str
    """

    lineDiscountId: Required[str]
    """The line discount ID. Required."""
    externalId: Required[str]
    """The external ID (e.g. custom invoicing system's ID). Required."""


class CustomInvoicingLineExternalIdMapping(TypedDict, total=False):
    """Mapping between lines and external IDs.

    :ivar line_id: The line ID. Required.
    :vartype line_id: str
    :ivar external_id: The external ID (e.g. custom invoicing system's ID). Required.
    :vartype external_id: str
    """

    lineId: Required[str]
    """The line ID. Required."""
    externalId: Required[str]
    """The external ID (e.g. custom invoicing system's ID). Required."""


class CustomInvoicingSyncResult(TypedDict, total=False):
    """Information to synchronize the invoice.

    Can be used to store external app's IDs on the invoice or lines.

    :ivar invoice_number: If set the invoice's number will be set to this value.
    :vartype invoice_number: str
    :ivar external_id: If set the invoice's invoicing external ID will be set to this value.
    :vartype external_id: str
    :ivar line_external_ids: If set the invoice's line external IDs will be set to this value.

     This can be used to reference the external system's entities in the
     invoice.
    :vartype line_external_ids: list["CustomInvoicingLineExternalIdMapping"]
    :ivar line_discount_external_ids: If set the invoice's line discount external IDs will be set
     to this value.

     This can be used to reference the external system's entities in the
     invoice.
    :vartype line_discount_external_ids: list["CustomInvoicingLineDiscountExternalIdMapping"]
    """

    invoiceNumber: str
    """If set the invoice's number will be set to this value."""
    externalId: str
    """If set the invoice's invoicing external ID will be set to this value."""
    lineExternalIds: list["CustomInvoicingLineExternalIdMapping"]
    """If set the invoice's line external IDs will be set to this value.
     
     This can be used to reference the external system's entities in the
     invoice."""
    lineDiscountExternalIds: list["CustomInvoicingLineDiscountExternalIdMapping"]
    """If set the invoice's line discount external IDs will be set to this value.
     
     This can be used to reference the external system's entities in the
     invoice."""


class CustomInvoicingTaxConfig(TypedDict, total=False):
    """Custom invoicing tax config.

    :ivar code: Tax code. Required.
    :vartype code: str
    """

    code: Required[str]
    """Tax code. Required."""


class CustomInvoicingUpdatePaymentStatusRequest(TypedDict, total=False):  # pylint: disable=name-too-long
    """Update payment status request.

    Can be used to manipulate invoice's payment status (when custominvoicing app is being used).

    :ivar trigger: The trigger to be executed on the invoice. Required. Known values are: "paid",
     "payment_failed", "payment_uncollectible", "payment_overdue", "action_required", and "void".
    :vartype trigger: Union[str, "CustomInvoicingPaymentTrigger"]
    """

    trigger: Required[Union[str, "CustomInvoicingPaymentTrigger"]]
    """The trigger to be executed on the invoice. Required. Known values are: \"paid\",
     \"payment_failed\", \"payment_uncollectible\", \"payment_overdue\", \"action_required\", and
     \"void\"."""


class OmitPropertiesResourceCreateModel(TypedDict, total=False):
    """The template for omitting properties.

    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: "Metadata"
    :ivar alignment: Alignment configuration for the plan.
    :vartype alignment: "Alignment"
    :ivar currency: Currency. Required.
    :vartype currency: str
    :ivar billing_cadence: Billing cadence. Required.
    :vartype billing_cadence: str
    :ivar pro_rating_config: Pro-rating configuration.
    :vartype pro_rating_config: "ProRatingConfig"
    :ivar settlement_mode: Settlement mode. Known values are: "credit_then_invoice" and
     "credit_only".
    :vartype settlement_mode: Union[str, "BillingSettlementMode"]
    :ivar phases: Plan phases. Required.
    :vartype phases: list["PlanPhase"]
    """

    name: Required[str]
    """Display name. Required."""
    description: str
    """Description."""
    metadata: Optional["Metadata"]
    """Metadata."""
    alignment: "Alignment"
    """Alignment configuration for the plan."""
    currency: Required[str]
    """Currency. Required."""
    billingCadence: Required[str]
    """Billing cadence. Required."""
    proRatingConfig: "ProRatingConfig"
    """Pro-rating configuration."""
    settlementMode: Union[str, "BillingSettlementMode"]
    """Settlement mode. Known values are: \"credit_then_invoice\" and \"credit_only\"."""
    phases: Required[list["PlanPhase"]]
    """Plan phases. Required."""


class CustomPlanInput(OmitPropertiesResourceCreateModel):
    """Plan input for custom subscription creation (without key and version).

    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: "Metadata"
    :ivar alignment: Alignment configuration for the plan.
    :vartype alignment: "Alignment"
    :ivar currency: Currency. Required.
    :vartype currency: str
    :ivar billing_cadence: Billing cadence. Required.
    :vartype billing_cadence: str
    :ivar pro_rating_config: Pro-rating configuration.
    :vartype pro_rating_config: "ProRatingConfig"
    :ivar settlement_mode: Settlement mode. Known values are: "credit_then_invoice" and
     "credit_only".
    :vartype settlement_mode: Union[str, "BillingSettlementMode"]
    :ivar phases: Plan phases. Required.
    :vartype phases: list["PlanPhase"]
    """


class CustomSubscriptionChange(TypedDict, total=False):
    """Change a custom subscription.

    :ivar timing: Timing configuration for the change, when the change should take effect. For
     changing a subscription, the accepted values depend on the subscription configuration.
     Required. Is either a Union[str, "_models.SubscriptionTimingEnum"] type or a datetime.datetime
     type.
    :vartype timing: "_unions.SubscriptionTiming"
    :ivar billing_anchor: The billing anchor of the subscription. The provided date will be
     normalized according to the billing cadence to the nearest recurrence before start time. If not
     provided, the previous subscription billing anchor will be used.
    :vartype billing_anchor: str
    :ivar custom_plan: The custom plan description which defines the Subscription. Required.
    :vartype custom_plan: "CustomPlanInput"
    """

    timing: Required["_unions.SubscriptionTiming"]
    """Timing configuration for the change, when the change should take effect. For changing a
     subscription, the accepted values depend on the subscription configuration. Required. Is either
     a Union[str, \"_models.SubscriptionTimingEnum\"] type or a datetime.datetime type."""
    billingAnchor: str
    """The billing anchor of the subscription. The provided date will be normalized according to the
     billing cadence to the nearest recurrence before start time. If not provided, the previous
     subscription billing anchor will be used."""
    customPlan: Required["CustomPlanInput"]
    """The custom plan description which defines the Subscription. Required."""


class CustomSubscriptionCreate(TypedDict, total=False):
    """Create custom.

    :ivar custom_plan: The custom plan description which defines the Subscription. Required.
    :vartype custom_plan: "CustomPlanInput"
    :ivar timing: Timing configuration for the change, when the change should take effect. The
     default is immediate. Is either a Union[str, "_models.SubscriptionTimingEnum"] type or a
     datetime.datetime type.
    :vartype timing: "_unions.SubscriptionTiming"
    :ivar customer_id: The ID of the customer. Provide either the key or ID. Has presedence over
     the key.
    :vartype customer_id: str
    :ivar customer_key: The key of the customer. Provide either the key or ID.
    :vartype customer_key: str
    :ivar billing_anchor: The billing anchor of the subscription. The provided date will be
     normalized according to the billing cadence to the nearest recurrence before start time. If not
     provided, the subscription start time will be used.
    :vartype billing_anchor: str
    """

    customPlan: Required["CustomPlanInput"]
    """The custom plan description which defines the Subscription. Required."""
    timing: "_unions.SubscriptionTiming"
    """Timing configuration for the change, when the change should take effect. The default is
     immediate. Is either a Union[str, \"_models.SubscriptionTimingEnum\"] type or a
     datetime.datetime type."""
    customerId: str
    """The ID of the customer. Provide either the key or ID. Has presedence over the key."""
    customerKey: str
    """The key of the customer. Provide either the key or ID."""
    billingAnchor: str
    """The billing anchor of the subscription. The provided date will be normalized according to the
     billing cadence to the nearest recurrence before start time. If not provided, the subscription
     start time will be used."""


class DiscountPercentage(TypedDict, total=False):
    """Percentage discount.

    :ivar percentage: Percentage. Required.
    :vartype percentage: float
    """

    percentage: Required[float]
    """Percentage. Required."""


class Discounts(TypedDict, total=False):
    """Discount by type on a price.

    :ivar percentage: The percentage discount.
    :vartype percentage: "DiscountPercentage"
    :ivar usage: The usage discount.
    :vartype usage: "DiscountUsage"
    """

    percentage: "DiscountPercentage"
    """The percentage discount."""
    usage: "DiscountUsage"
    """The usage discount."""


class DiscountUsage(TypedDict, total=False):
    """Usage discount.

    Usage discount means that the first N items are free. From billing perspective
    this means that any usage on a specific feature is considered 0 until this discount
    is exhausted.

    :ivar quantity: Usage. Required.
    :vartype quantity: str
    """

    quantity: Required[str]
    """Usage. Required."""


class DynamicPriceWithCommitments(TypedDict, total=False):
    """Dynamic price with spend commitments.

    :ivar type: The type of the price. Required. DYNAMIC.
    :vartype type: Literal[PriceType.DYNAMIC]
    :ivar multiplier: The multiplier to apply to the base price to get the dynamic price.
    :vartype multiplier: str
    :ivar minimum_amount: Minimum amount.
    :vartype minimum_amount: str
    :ivar maximum_amount: Maximum amount.
    :vartype maximum_amount: str
    """

    type: Required[Literal[PriceType.DYNAMIC]]
    """The type of the price. Required. DYNAMIC."""
    multiplier: str
    """The multiplier to apply to the base price to get the dynamic price."""
    minimumAmount: str
    """Minimum amount."""
    maximumAmount: str
    """Maximum amount."""


class EditSubscriptionAddItem(TypedDict, total=False):
    """Add a new item to a phase.

    :ivar op: Required. ADD_ITEM.
    :vartype op: Literal[EditOp.ADD_ITEM]
    :ivar phase_key: Required.
    :vartype phase_key: str
    :ivar rate_card: Required. Is either a RateCardFlatFee type or a RateCardUsageBased type.
    :vartype rate_card: "_unions.RateCard"
    """

    op: Required[Literal[EditOp.ADD_ITEM]]
    """Required. ADD_ITEM."""
    phaseKey: Required[str]
    """Required."""
    rateCard: Required["_unions.RateCard"]
    """Required. Is either a RateCardFlatFee type or a RateCardUsageBased type."""


class EditSubscriptionAddPhase(TypedDict, total=False):
    """Add a new phase.

    :ivar op: Required. ADD_PHASE.
    :vartype op: Literal[EditOp.ADD_PHASE]
    :ivar phase: Required.
    :vartype phase: "SubscriptionPhaseCreate"
    """

    op: Required[Literal[EditOp.ADD_PHASE]]
    """Required. ADD_PHASE."""
    phase: Required["SubscriptionPhaseCreate"]
    """Required."""


class EditSubscriptionRemoveItem(TypedDict, total=False):
    """Remove an item from a phase.

    :ivar op: Required. REMOVE_ITEM.
    :vartype op: Literal[EditOp.REMOVE_ITEM]
    :ivar phase_key: Required.
    :vartype phase_key: str
    :ivar item_key: Required.
    :vartype item_key: str
    """

    op: Required[Literal[EditOp.REMOVE_ITEM]]
    """Required. REMOVE_ITEM."""
    phaseKey: Required[str]
    """Required."""
    itemKey: Required[str]
    """Required."""


class EditSubscriptionRemovePhase(TypedDict, total=False):
    """Remove a phase.

    :ivar op: Required. REMOVE_PHASE.
    :vartype op: Literal[EditOp.REMOVE_PHASE]
    :ivar phase_key: Required.
    :vartype phase_key: str
    :ivar shift: Required. Known values are: "next" and "prev".
    :vartype shift: Union[str, "RemovePhaseShifting"]
    """

    op: Required[Literal[EditOp.REMOVE_PHASE]]
    """Required. REMOVE_PHASE."""
    phaseKey: Required[str]
    """Required."""
    shift: Required[Union[str, "RemovePhaseShifting"]]
    """Required. Known values are: \"next\" and \"prev\"."""


class EditSubscriptionStretchPhase(TypedDict, total=False):
    """Stretch a phase.

    :ivar op: Required. STRETCH_PHASE.
    :vartype op: Literal[EditOp.STRETCH_PHASE]
    :ivar phase_key: Required.
    :vartype phase_key: str
    :ivar extend_by: Required.
    :vartype extend_by: str
    """

    op: Required[Literal[EditOp.STRETCH_PHASE]]
    """Required. STRETCH_PHASE."""
    phaseKey: Required[str]
    """Required."""
    extendBy: Required[str]
    """Required."""


class EditSubscriptionUnscheduleEdit(TypedDict, total=False):
    """Unschedules any edits from the current phase.

    :ivar op: Required. UNSCHEDULE_EDIT.
    :vartype op: Literal[EditOp.UNSCHEDULE_EDIT]
    """

    op: Required[Literal[EditOp.UNSCHEDULE_EDIT]]
    """Required. UNSCHEDULE_EDIT."""


class EntitlementBooleanCreateInputs(TypedDict, total=False):
    """Create inputs for boolean entitlement.

    :ivar feature_key: The feature the subject is entitled to use. Either featureKey or featureId
     is required.
    :vartype feature_key: str
    :ivar feature_id: The feature the subject is entitled to use. Either featureKey or featureId is
     required.
    :vartype feature_id: str
    :ivar metadata: Additional metadata for the feature.
    :vartype metadata: "Metadata"
    :ivar usage_period: The usage period associated with the entitlement.
    :vartype usage_period: "RecurringPeriodCreateInput"
    :ivar type: Required. BOOLEAN.
    :vartype type: Literal[EntitlementType.BOOLEAN]
    """

    featureKey: str
    """The feature the subject is entitled to use. Either featureKey or featureId is required."""
    featureId: str
    """The feature the subject is entitled to use. Either featureKey or featureId is required."""
    metadata: "Metadata"
    """Additional metadata for the feature."""
    usagePeriod: "RecurringPeriodCreateInput"
    """The usage period associated with the entitlement."""
    type: Required[Literal[EntitlementType.BOOLEAN]]
    """Required. BOOLEAN."""


class EntitlementGrantCreateInput(TypedDict, total=False):
    """The grant creation input.

    :ivar amount: The amount to grant. Should be a positive number. Required.
    :vartype amount: float
    :ivar priority: The priority of the grant. Grants with higher priority are applied first.
     Priority is a positive decimal numbers. With lower numbers indicating higher importance. For
     example, a priority of 1 is more urgent than a priority of 2. When there are several grants
     available for the same subject, the system selects the grant with the highest priority. In
     cases where grants share the same priority level, the grant closest to its expiration will be
     used first. In the case of two grants have identical priorities and expiration dates, the
     system will use the grant that was created first.
    :vartype priority: int
    :ivar effective_at: Effective date for grants and anchor for recurring grants. Provided value
     will be ceiled to metering windowSize (minute). Required.
    :vartype effective_at: str
    :ivar expiration: The grant expiration definition. Required.
    :vartype expiration: "ExpirationPeriod"
    :ivar max_rollover_amount: Grants are rolled over at reset, after which they can have a
     different balance compared to what they had before the reset. Balance after the reset is
     calculated as: Balance_After_Reset = MIN(MaxRolloverAmount, MAX(Balance_Before_Reset,
     MinRolloverAmount)).
    :vartype max_rollover_amount: float
    :ivar min_rollover_amount: Grants are rolled over at reset, after which they can have a
     different balance compared to what they had before the reset. Balance after the reset is
     calculated as: Balance_After_Reset = MIN(MaxRolloverAmount, MAX(Balance_Before_Reset,
     MinRolloverAmount)).
    :vartype min_rollover_amount: float
    :ivar metadata: The grant metadata.
    :vartype metadata: "Metadata"
    :ivar recurrence: The subject of the grant.
    :vartype recurrence: "RecurringPeriodCreateInput"
    """

    amount: Required[float]
    """The amount to grant. Should be a positive number. Required."""
    priority: int
    """The priority of the grant. Grants with higher priority are applied first. Priority is a
     positive decimal numbers. With lower numbers indicating higher importance. For example, a
     priority of 1 is more urgent than a priority of 2. When there are several grants available for
     the same subject, the system selects the grant with the highest priority. In cases where grants
     share the same priority level, the grant closest to its expiration will be used first. In the
     case of two grants have identical priorities and expiration dates, the system will use the
     grant that was created first."""
    effectiveAt: Required[str]
    """Effective date for grants and anchor for recurring grants. Provided value will be ceiled to
     metering windowSize (minute). Required."""
    expiration: Required["ExpirationPeriod"]
    """The grant expiration definition. Required."""
    maxRolloverAmount: float
    """Grants are rolled over at reset, after which they can have a different balance compared to what
     they had before the reset. Balance after the reset is calculated as: Balance_After_Reset =
     MIN(MaxRolloverAmount, MAX(Balance_Before_Reset, MinRolloverAmount))."""
    minRolloverAmount: float
    """Grants are rolled over at reset, after which they can have a different balance compared to what
     they had before the reset. Balance after the reset is calculated as: Balance_After_Reset =
     MIN(MaxRolloverAmount, MAX(Balance_Before_Reset, MinRolloverAmount))."""
    metadata: "Metadata"
    """The grant metadata."""
    recurrence: "RecurringPeriodCreateInput"
    """The subject of the grant."""


class EntitlementGrantCreateInputV2(TypedDict, total=False):
    """The grant creation input.

    :ivar amount: The amount to grant. Should be a positive number. Required.
    :vartype amount: float
    :ivar priority: The priority of the grant. Grants with higher priority are applied first.
     Priority is a positive decimal numbers. With lower numbers indicating higher importance. For
     example, a priority of 1 is more urgent than a priority of 2. When there are several grants
     available for the same subject, the system selects the grant with the highest priority. In
     cases where grants share the same priority level, the grant closest to its expiration will be
     used first. In the case of two grants have identical priorities and expiration dates, the
     system will use the grant that was created first.
    :vartype priority: int
    :ivar effective_at: Effective date for grants and anchor for recurring grants. Provided value
     will be ceiled to metering windowSize (minute). Required.
    :vartype effective_at: str
    :ivar min_rollover_amount: Grants are rolled over at reset, after which they can have a
     different balance compared to what they had before the reset. Balance after the reset is
     calculated as: Balance_After_Reset = MIN(MaxRolloverAmount, MAX(Balance_Before_Reset,
     MinRolloverAmount)).
    :vartype min_rollover_amount: float
    :ivar metadata: The grant metadata.
    :vartype metadata: "Metadata"
    :ivar recurrence: The subject of the grant.
    :vartype recurrence: "RecurringPeriodCreateInput"
    :ivar max_rollover_amount: Grants are rolled over at reset, after which they can have a
     different balance compared to what they had before the reset. The default value equals grant
     amount. Balance after the reset is calculated as: Balance_After_Reset = MIN(MaxRolloverAmount,
     MAX(Balance_Before_Reset, MinRolloverAmount)).
    :vartype max_rollover_amount: float
    :ivar expiration: The grant expiration definition. If no expiration is provided, the grant can
     be active indefinitely.
    :vartype expiration: "ExpirationPeriod"
    :ivar annotations: Grant annotations.
    :vartype annotations: "Annotations"
    """

    amount: Required[float]
    """The amount to grant. Should be a positive number. Required."""
    priority: int
    """The priority of the grant. Grants with higher priority are applied first. Priority is a
     positive decimal numbers. With lower numbers indicating higher importance. For example, a
     priority of 1 is more urgent than a priority of 2. When there are several grants available for
     the same subject, the system selects the grant with the highest priority. In cases where grants
     share the same priority level, the grant closest to its expiration will be used first. In the
     case of two grants have identical priorities and expiration dates, the system will use the
     grant that was created first."""
    effectiveAt: Required[str]
    """Effective date for grants and anchor for recurring grants. Provided value will be ceiled to
     metering windowSize (minute). Required."""
    minRolloverAmount: float
    """Grants are rolled over at reset, after which they can have a different balance compared to what
     they had before the reset. Balance after the reset is calculated as: Balance_After_Reset =
     MIN(MaxRolloverAmount, MAX(Balance_Before_Reset, MinRolloverAmount))."""
    metadata: "Metadata"
    """The grant metadata."""
    recurrence: "RecurringPeriodCreateInput"
    """The subject of the grant."""
    maxRolloverAmount: float
    """Grants are rolled over at reset, after which they can have a different balance compared to what
     they had before the reset. The default value equals grant amount. Balance after the reset is
     calculated as: Balance_After_Reset = MIN(MaxRolloverAmount, MAX(Balance_Before_Reset,
     MinRolloverAmount))."""
    expiration: "ExpirationPeriod"
    """The grant expiration definition. If no expiration is provided, the grant can be active
     indefinitely."""
    annotations: "Annotations"
    """Grant annotations."""


class EntitlementMeteredCreateInputs(TypedDict, total=False):
    """Create inpurs for metered entitlement.

    :ivar feature_key: The feature the subject is entitled to use. Either featureKey or featureId
     is required.
    :vartype feature_key: str
    :ivar feature_id: The feature the subject is entitled to use. Either featureKey or featureId is
     required.
    :vartype feature_id: str
    :ivar metadata: Additional metadata for the feature.
    :vartype metadata: "Metadata"
    :ivar type: Required. METERED.
    :vartype type: Literal[EntitlementType.METERED]
    :ivar is_soft_limit: Soft limit.
    :vartype is_soft_limit: bool
    :ivar is_unlimited: Deprecated, ignored by the backend. Please use isSoftLimit instead; this
     field will be removed in the future.
    :vartype is_unlimited: bool
    :ivar usage_period: The usage period associated with the entitlement. Required.
    :vartype usage_period: "RecurringPeriodCreateInput"
    :ivar measure_usage_from: Defines the time from which usage is measured. If not specified on
     creation, defaults to entitlement creation time. Is either a Union[str,
     "_models.MeasureUsageFromPreset"] type or a datetime.datetime type.
    :vartype measure_usage_from: "_unions.MeasureUsageFrom"
    :ivar issue_after_reset: Initial grant amount.
    :vartype issue_after_reset: float
    :ivar issue_after_reset_priority: Issue grant after reset priority.
    :vartype issue_after_reset_priority: int
    :ivar preserve_overage_at_reset: Preserve overage at reset.
    :vartype preserve_overage_at_reset: bool
    """

    featureKey: str
    """The feature the subject is entitled to use. Either featureKey or featureId is required."""
    featureId: str
    """The feature the subject is entitled to use. Either featureKey or featureId is required."""
    metadata: "Metadata"
    """Additional metadata for the feature."""
    type: Required[Literal[EntitlementType.METERED]]
    """Required. METERED."""
    isSoftLimit: bool
    """Soft limit."""
    isUnlimited: bool
    """Deprecated, ignored by the backend. Please use isSoftLimit instead; this field will be removed
     in the future."""
    usagePeriod: Required["RecurringPeriodCreateInput"]
    """The usage period associated with the entitlement. Required."""
    measureUsageFrom: "_unions.MeasureUsageFrom"
    """Defines the time from which usage is measured. If not specified on creation, defaults to
     entitlement creation time. Is either a Union[str, \"_models.MeasureUsageFromPreset\"] type or a
     datetime.datetime type."""
    issueAfterReset: float
    """Initial grant amount."""
    issueAfterResetPriority: int
    """Issue grant after reset priority."""
    preserveOverageAtReset: bool
    """Preserve overage at reset."""


class EntitlementMeteredV2CreateInputs(TypedDict, total=False):
    """Create inputs for metered entitlement.

    :ivar feature_key: The feature the subject is entitled to use. Either featureKey or featureId
     is required.
    :vartype feature_key: str
    :ivar feature_id: The feature the subject is entitled to use. Either featureKey or featureId is
     required.
    :vartype feature_id: str
    :ivar metadata: Additional metadata for the feature.
    :vartype metadata: "Metadata"
    :ivar type: Required. METERED.
    :vartype type: Literal[EntitlementType.METERED]
    :ivar is_soft_limit: Soft limit.
    :vartype is_soft_limit: bool
    :ivar usage_period: The usage period associated with the entitlement. Required.
    :vartype usage_period: "RecurringPeriodCreateInput"
    :ivar measure_usage_from: Defines the time from which usage is measured. If not specified on
     creation, defaults to entitlement creation time. Is either a Union[str,
     "_models.MeasureUsageFromPreset"] type or a datetime.datetime type.
    :vartype measure_usage_from: "_unions.MeasureUsageFrom"
    :ivar preserve_overage_at_reset: Preserve overage at reset.
    :vartype preserve_overage_at_reset: bool
    :ivar issue_after_reset: Initial grant amount.
    :vartype issue_after_reset: float
    :ivar issue_after_reset_priority: Issue grant after reset priority.
    :vartype issue_after_reset_priority: int
    :ivar issue: Issue after reset.
    :vartype issue: "IssueAfterReset"
    :ivar grants: Grants.
    :vartype grants: list["EntitlementGrantCreateInputV2"]
    """

    featureKey: str
    """The feature the subject is entitled to use. Either featureKey or featureId is required."""
    featureId: str
    """The feature the subject is entitled to use. Either featureKey or featureId is required."""
    metadata: "Metadata"
    """Additional metadata for the feature."""
    type: Required[Literal[EntitlementType.METERED]]
    """Required. METERED."""
    isSoftLimit: bool
    """Soft limit."""
    usagePeriod: Required["RecurringPeriodCreateInput"]
    """The usage period associated with the entitlement. Required."""
    measureUsageFrom: "_unions.MeasureUsageFrom"
    """Defines the time from which usage is measured. If not specified on creation, defaults to
     entitlement creation time. Is either a Union[str, \"_models.MeasureUsageFromPreset\"] type or a
     datetime.datetime type."""
    preserveOverageAtReset: bool
    """Preserve overage at reset."""
    issueAfterReset: float
    """Initial grant amount."""
    issueAfterResetPriority: int
    """Issue grant after reset priority."""
    issue: "IssueAfterReset"
    """Issue after reset."""
    grants: list["EntitlementGrantCreateInputV2"]
    """Grants."""


class EntitlementStaticCreateInputs(TypedDict, total=False):
    """Create inputs for static entitlement.

    :ivar feature_key: The feature the subject is entitled to use. Either featureKey or featureId
     is required.
    :vartype feature_key: str
    :ivar feature_id: The feature the subject is entitled to use. Either featureKey or featureId is
     required.
    :vartype feature_id: str
    :ivar metadata: Additional metadata for the feature.
    :vartype metadata: "Metadata"
    :ivar usage_period: The usage period associated with the entitlement.
    :vartype usage_period: "RecurringPeriodCreateInput"
    :ivar type: Required. STATIC.
    :vartype type: Literal[EntitlementType.STATIC]
    :ivar config: The JSON parsable config of the entitlement. This value is also returned when
     checking entitlement access and it is useful for configuring fine-grained access settings to
     the feature, implemented in your own system. Has to be an object. Required.
    :vartype config: str
    """

    featureKey: str
    """The feature the subject is entitled to use. Either featureKey or featureId is required."""
    featureId: str
    """The feature the subject is entitled to use. Either featureKey or featureId is required."""
    metadata: "Metadata"
    """Additional metadata for the feature."""
    usagePeriod: "RecurringPeriodCreateInput"
    """The usage period associated with the entitlement."""
    type: Required[Literal[EntitlementType.STATIC]]
    """Required. STATIC."""
    config: Required[str]
    """The JSON parsable config of the entitlement. This value is also returned when checking
     entitlement access and it is useful for configuring fine-grained access settings to the
     feature, implemented in your own system. Has to be an object. Required."""


class Event(TypedDict, total=False):
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
    :vartype datacontenttype: Literal["application/json"]
    :ivar dataschema: Identifies the schema that data adheres to.
    :vartype dataschema: str
    :ivar subject: Describes the subject of the event in the context of the event producer
     (identified by source). Required.
    :vartype subject: str
    :ivar time: Timestamp of when the occurrence happened. Must adhere to RFC 3339.
    :vartype time: str
    :ivar data: The event payload. Optional, if present it must be a JSON object.
    :vartype data: dict[str, Any]
    """

    id: Required[str]
    """Identifies the event. Required."""
    source: Required[str]
    """Identifies the context in which an event happened. Required."""
    specversion: Required[str]
    """The version of the CloudEvents specification which the event uses. Required."""
    type: Required[str]
    """Contains a value describing the type of event related to the originating occurrence. Required."""
    datacontenttype: Optional[Literal["application/json"]]
    """Content type of the CloudEvents data value. Only the value \"application/json\" is allowed over
     HTTP. Default value is \"application/json\"."""
    dataschema: Optional[str]
    """Identifies the schema that data adheres to."""
    subject: Required[str]
    """Describes the subject of the event in the context of the event producer (identified by source).
     Required."""
    time: Optional[str]
    """Timestamp of when the occurrence happened. Must adhere to RFC 3339."""
    data: Optional[dict[str, Any]]
    """The event payload. Optional, if present it must be a JSON object."""


class ExpirationPeriod(TypedDict, total=False):
    """The grant expiration definition.

    :ivar duration: The unit of time for the expiration period. Required. Known values are: "HOUR",
     "DAY", "WEEK", "MONTH", and "YEAR".
    :vartype duration: Union[str, "ExpirationDuration"]
    :ivar count: The number of time units in the expiration period. Required.
    :vartype count: int
    """

    duration: Required[Union[str, "ExpirationDuration"]]
    """The unit of time for the expiration period. Required. Known values are: \"HOUR\", \"DAY\",
     \"WEEK\", \"MONTH\", and \"YEAR\"."""
    count: Required[int]
    """The number of time units in the expiration period. Required."""


class FeatureCreateInputs(TypedDict, total=False):
    """Represents a feature that can be enabled or disabled for a plan. Used both for product catalog
    and entitlements.

    :ivar key: The unique key of the feature. Required.
    :vartype key: str
    :ivar name: The human-readable name of the feature. Required.
    :vartype name: str
    :ivar metadata: Optional metadata.
    :vartype metadata: "Metadata"
    :ivar meter_slug: Meter slug.
    :vartype meter_slug: str
    :ivar meter_group_by_filters: Meter group by filters.
    :vartype meter_group_by_filters: dict[str, str]
    :ivar advanced_meter_group_by_filters: Advanced meter group by filters.
    :vartype advanced_meter_group_by_filters: dict[str, "FilterString"]
    :ivar unit_cost: Unit cost. Is either a FeatureManualUnitCost type or a FeatureLLMUnitCost
     type.
    :vartype unit_cost: "_unions.FeatureUnitCost"
    """

    key: Required[str]
    """The unique key of the feature. Required."""
    name: Required[str]
    """The human-readable name of the feature. Required."""
    metadata: "Metadata"
    """Optional metadata."""
    meterSlug: str
    """Meter slug."""
    meterGroupByFilters: dict[str, str]
    """Meter group by filters."""
    advancedMeterGroupByFilters: dict[str, "FilterString"]
    """Advanced meter group by filters."""
    unitCost: "_unions.FeatureUnitCost"
    """Unit cost. Is either a FeatureManualUnitCost type or a FeatureLLMUnitCost type."""


class FeatureLLMUnitCost(TypedDict, total=False):
    """LLM cost lookup configuration. Maps meter group-by dimensions to LLM cost database fields.

    :ivar type: Required. LLM.
    :vartype type: Literal[FeatureUnitCostType.LLM]
    :ivar provider_property: Provider property.
    :vartype provider_property: str
    :ivar provider: Provider.
    :vartype provider: str
    :ivar model_property: Model property.
    :vartype model_property: str
    :ivar model: Model.
    :vartype model: str
    :ivar token_type_property: Token type property.
    :vartype token_type_property: str
    :ivar token_type: Token type.
    :vartype token_type: str
    :ivar pricing: Resolved pricing.
    :vartype pricing: "FeatureLLMUnitCostPricing"
    """

    type: Required[Literal[FeatureUnitCostType.LLM]]
    """Required. LLM."""
    providerProperty: str
    """Provider property."""
    provider: str
    """Provider."""
    modelProperty: str
    """Model property."""
    model: str
    """Model."""
    tokenTypeProperty: str
    """Token type property."""
    tokenType: str
    """Token type."""
    pricing: "FeatureLLMUnitCostPricing"
    """Resolved pricing."""


class FeatureLLMUnitCostPricing(TypedDict, total=False):
    """Resolved per-token pricing from the LLM cost database.

    :ivar input_per_token: Input per token. Required.
    :vartype input_per_token: str
    :ivar output_per_token: Output per token. Required.
    :vartype output_per_token: str
    :ivar cache_read_per_token: Cache read per token.
    :vartype cache_read_per_token: str
    :ivar reasoning_per_token: Reasoning per token.
    :vartype reasoning_per_token: str
    :ivar cache_write_per_token: Cache write per token.
    :vartype cache_write_per_token: str
    """

    inputPerToken: Required[str]
    """Input per token. Required."""
    outputPerToken: Required[str]
    """Output per token. Required."""
    cacheReadPerToken: str
    """Cache read per token."""
    reasoningPerToken: str
    """Reasoning per token."""
    cacheWritePerToken: str
    """Cache write per token."""


class FeatureManualUnitCost(TypedDict, total=False):
    """A fixed per-unit cost amount.

    :ivar type: Required. MANUAL.
    :vartype type: Literal[FeatureUnitCostType.MANUAL]
    :ivar amount: Fixed per-unit cost amount in USD. Required.
    :vartype amount: str
    """

    type: Required[Literal[FeatureUnitCostType.MANUAL]]
    """Required. MANUAL."""
    amount: Required[str]
    """Fixed per-unit cost amount in USD. Required."""


FilterIDExact = TypedDict(
    "FilterIDExact",
    {
        "$in": Optional[list[str]],
    },
    total=False,
)
FilterIDExact.__doc__ = """A filter for a ID (ULID) field allowing only equality or inclusion.

:ivar in_property: The field must be in the provided list of values.
:vartype in_property: list[str]
"""


FilterString = TypedDict(
    "FilterString",
    {
        "$eq": Optional[str],
        "$ne": Optional[str],
        "$in": Optional[list[str]],
        "$nin": Optional[list[str]],
        "$like": Optional[str],
        "$nlike": Optional[str],
        "$ilike": Optional[str],
        "$nilike": Optional[str],
        "$gt": Optional[str],
        "$gte": Optional[str],
        "$lt": Optional[str],
        "$lte": Optional[str],
        "$and": Optional[list["FilterString"]],
        "$or": Optional[list["FilterString"]],
    },
    total=False,
)
FilterString.__doc__ = """A filter for a string field.

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
:vartype and_property: list["FilterString"]
:ivar or_property: Provide a list of filters to be combined with a logical OR.
:vartype or_property: list["FilterString"]
"""


FilterTime = TypedDict(
    "FilterTime",
    {
        "$gt": Optional[str],
        "$gte": Optional[str],
        "$lt": Optional[str],
        "$lte": Optional[str],
        "$and": Optional[list["FilterTime"]],
        "$or": Optional[list["FilterTime"]],
    },
    total=False,
)
FilterTime.__doc__ = """A filter for a time field.

:ivar gt: The field must be greater than the provided value.
:vartype gt: str
:ivar gte: The field must be greater than or equal to the provided value.
:vartype gte: str
:ivar lt: The field must be less than the provided value.
:vartype lt: str
:ivar lte: The field must be less than or equal to the provided value.
:vartype lte: str
:ivar and_property: Provide a list of filters to be combined with a logical AND.
:vartype and_property: list["FilterTime"]
:ivar or_property: Provide a list of filters to be combined with a logical OR.
:vartype or_property: list["FilterTime"]
"""


class FlatPrice(TypedDict, total=False):
    """Flat price.

    :ivar type: The type of the price. Required. FLAT.
    :vartype type: Literal[PriceType.FLAT]
    :ivar amount: The amount of the flat price. Required.
    :vartype amount: str
    """

    type: Required[Literal[PriceType.FLAT]]
    """The type of the price. Required. FLAT."""
    amount: Required[str]
    """The amount of the flat price. Required."""


class FlatPriceWithPaymentTerm(TypedDict, total=False):
    """Flat price with payment term.

    :ivar type: The type of the price. Required. FLAT.
    :vartype type: Literal[PriceType.FLAT]
    :ivar amount: The amount of the flat price. Required.
    :vartype amount: str
    :ivar payment_term: The payment term of the flat price. Defaults to in advance. Known values
     are: "in_advance" and "in_arrears".
    :vartype payment_term: Union[str, "PricePaymentTerm"]
    """

    type: Required[Literal[PriceType.FLAT]]
    """The type of the price. Required. FLAT."""
    amount: Required[str]
    """The amount of the flat price. Required."""
    paymentTerm: Union[str, "PricePaymentTerm"]
    """The payment term of the flat price. Defaults to in advance. Known values are: \"in_advance\"
     and \"in_arrears\"."""


class InstallWithApiKeyRequest(TypedDict, total=False):
    """InstallWithApiKeyRequest.

    :ivar name: Name of the application to install.

     If name is not provided defaults to the marketplace listing's name.
    :vartype name: str
    :ivar create_billing_profile: If true, a billing profile will be created for the app. The
     Stripe app will be also set as the default billing profile if the current default is a Sandbox
     app.
    :vartype create_billing_profile: bool
    :ivar api_key: The API key for the provider. For example, the Stripe API key. Required.
    :vartype api_key: str
    """

    name: str
    """Name of the application to install.
     
     If name is not provided defaults to the marketplace listing's name."""
    createBillingProfile: bool
    """If true, a billing profile will be created for the app. The Stripe app will be also set as the
     default billing profile if the current default is a Sandbox app."""
    apiKey: Required[str]
    """The API key for the provider. For example, the Stripe API key. Required."""


class InvoiceLineReplaceUpdate(TypedDict, total=False):
    """InvoiceLineReplaceUpdate represents the update model for an UBP invoice line.

    This type makes ID optional to allow for creating new lines as part of the update.

    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: "Metadata"
    :ivar tax_config: Tax config specify the tax configuration for this line.
    :vartype tax_config: "TaxConfig"
    :ivar period: Period of the line item applies to for revenue recognition pruposes.

     Billing always treats periods as start being inclusive and end being exclusive. Required.
    :vartype period: "Period"
    :ivar invoice_at: The time this line item should be invoiced. Required.
    :vartype invoice_at: str
    :ivar price: Price of the usage-based item being sold. Is one of the following types:
     FlatPriceWithPaymentTerm, UnitPriceWithCommitments, TieredPriceWithCommitments,
     DynamicPriceWithCommitments, PackagePriceWithCommitments
    :vartype price: "_unions.RateCardUsageBasedPrice"
    :ivar feature_key: The feature that the usage is based on.
    :vartype feature_key: str
    :ivar rate_card: The rate card that is used for this line.

     The rate card captures the intent of the price and discounts for the usage-based item.
    :vartype rate_card: "InvoiceUsageBasedRateCard"
    :ivar id: The ID of the line.
    :vartype id: str
    """

    name: Required[str]
    """Display name. Required."""
    description: str
    """Description."""
    metadata: Optional["Metadata"]
    """Metadata."""
    taxConfig: "TaxConfig"
    """Tax config specify the tax configuration for this line."""
    period: Required["Period"]
    """Period of the line item applies to for revenue recognition pruposes.
     
     Billing always treats periods as start being inclusive and end being exclusive. Required."""
    invoiceAt: Required[str]
    """The time this line item should be invoiced. Required."""
    price: "_unions.RateCardUsageBasedPrice"
    """Price of the usage-based item being sold. Is one of the following types:
     FlatPriceWithPaymentTerm, UnitPriceWithCommitments, TieredPriceWithCommitments,
     DynamicPriceWithCommitments, PackagePriceWithCommitments"""
    featureKey: str
    """The feature that the usage is based on."""
    rateCard: "InvoiceUsageBasedRateCard"
    """The rate card that is used for this line.
     
     The rate card captures the intent of the price and discounts for the usage-based item."""
    id: str
    """The ID of the line."""


class InvoicePendingLineCreate(TypedDict, total=False):
    """InvoicePendingLineCreate represents the create model for an invoice line that is sold to the
    customer based on usage.

    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: "Metadata"
    :ivar tax_config: Tax config specify the tax configuration for this line.
    :vartype tax_config: "TaxConfig"
    :ivar period: Period of the line item applies to for revenue recognition pruposes.

     Billing always treats periods as start being inclusive and end being exclusive. Required.
    :vartype period: "Period"
    :ivar invoice_at: The time this line item should be invoiced. Required.
    :vartype invoice_at: str
    :ivar price: Price of the usage-based item being sold. Is one of the following types:
     FlatPriceWithPaymentTerm, UnitPriceWithCommitments, TieredPriceWithCommitments,
     DynamicPriceWithCommitments, PackagePriceWithCommitments
    :vartype price: "_unions.RateCardUsageBasedPrice"
    :ivar feature_key: The feature that the usage is based on.
    :vartype feature_key: str
    :ivar rate_card: The rate card that is used for this line.

     The rate card captures the intent of the price and discounts for the usage-based item.
    :vartype rate_card: "InvoiceUsageBasedRateCard"
    """

    name: Required[str]
    """Display name. Required."""
    description: str
    """Description."""
    metadata: Optional["Metadata"]
    """Metadata."""
    taxConfig: "TaxConfig"
    """Tax config specify the tax configuration for this line."""
    period: Required["Period"]
    """Period of the line item applies to for revenue recognition pruposes.
     
     Billing always treats periods as start being inclusive and end being exclusive. Required."""
    invoiceAt: Required[str]
    """The time this line item should be invoiced. Required."""
    price: "_unions.RateCardUsageBasedPrice"
    """Price of the usage-based item being sold. Is one of the following types:
     FlatPriceWithPaymentTerm, UnitPriceWithCommitments, TieredPriceWithCommitments,
     DynamicPriceWithCommitments, PackagePriceWithCommitments"""
    featureKey: str
    """The feature that the usage is based on."""
    rateCard: "InvoiceUsageBasedRateCard"
    """The rate card that is used for this line.
     
     The rate card captures the intent of the price and discounts for the usage-based item."""


class InvoicePendingLineCreateInput(TypedDict, total=False):
    """InvoicePendingLineCreate represents the create model for a pending invoice line.

    :ivar currency: The currency of the lines to be created. Required.
    :vartype currency: str
    :ivar lines: The lines to be created. Required.
    :vartype lines: list["InvoicePendingLineCreate"]
    """

    currency: Required[str]
    """The currency of the lines to be created. Required."""
    lines: Required[list["InvoicePendingLineCreate"]]
    """The lines to be created. Required."""


class InvoicePendingLinesActionFiltersInput(TypedDict, total=False):
    """InvoicePendingLinesActionFiltersInput specifies which lines to include in the invoice.

    :ivar line_ids: The pending line items to include in the invoice, if not provided:

     * all line items that have invoice_at < asOf will be included
     * [progressive billing only] all usage based line items will be included up to asOf, new
     usage-based line items will be staged for the rest of the billing cycle

     All lineIDs present in the list, must exists and must be invoicable as of asOf, or the
     action will fail.
    :vartype line_ids: list[str]
    """

    lineIds: list[str]
    """The pending line items to include in the invoice, if not provided:
 
      * all line items that have invoice_at < asOf will be included
      * [progressive billing only] all usage based line items will be included up to asOf, new
      usage-based line items will be staged for the rest of the billing cycle
 
      All lineIDs present in the list, must exists and must be invoicable as of asOf, or the
      action will fail."""


class InvoicePendingLinesActionInput(TypedDict, total=False):
    """BillingInvoiceActionInput is the input for creating an invoice.

    Invoice creation is always based on already pending line items created by the
    billingCreateLineByCustomer
    operation. Empty invoices are not allowed.

    :ivar filters: Filters to apply when creating the invoice.
    :vartype filters: "InvoicePendingLinesActionFiltersInput"
    :ivar as_of: The time as of which the invoice is created.

     If not provided, the current time is used.
    :vartype as_of: str
    :ivar customer_id: The customer ID for which to create the invoice. Required.
    :vartype customer_id: str
    :ivar progressive_billing_override: Override the progressive billing setting of the customer.

     Can be used to disable/enable progressive billing in case the business logic
     requires it, if not provided the billing profile's progressive billing setting will be used.
    :vartype progressive_billing_override: bool
    """

    filters: "InvoicePendingLinesActionFiltersInput"
    """Filters to apply when creating the invoice."""
    asOf: str
    """The time as of which the invoice is created.
     
     If not provided, the current time is used."""
    customerId: Required[str]
    """The customer ID for which to create the invoice. Required."""
    progressiveBillingOverride: bool
    """Override the progressive billing setting of the customer.
     
     Can be used to disable/enable progressive billing in case the business logic
     requires it, if not provided the billing profile's progressive billing setting will be used."""


class InvoiceReplaceUpdate(TypedDict, total=False):
    """InvoiceReplaceUpdate represents the update model for an invoice.

    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: "Metadata"
    :ivar supplier: The supplier of the lines included in the invoice. Required.
    :vartype supplier: "BillingPartyReplaceUpdate"
    :ivar customer: The customer the invoice is sent to. Required.
    :vartype customer: "BillingPartyReplaceUpdate"
    :ivar lines: The lines included in the invoice. Required.
    :vartype lines: list["InvoiceLineReplaceUpdate"]
    :ivar workflow: The workflow settings for the invoice. Required.
    :vartype workflow: "InvoiceWorkflowReplaceUpdate"
    """

    description: str
    """Description."""
    metadata: Optional["Metadata"]
    """Metadata."""
    supplier: Required["BillingPartyReplaceUpdate"]
    """The supplier of the lines included in the invoice. Required."""
    customer: Required["BillingPartyReplaceUpdate"]
    """The customer the invoice is sent to. Required."""
    lines: Required[list["InvoiceLineReplaceUpdate"]]
    """The lines included in the invoice. Required."""
    workflow: Required["InvoiceWorkflowReplaceUpdate"]
    """The workflow settings for the invoice. Required."""


class InvoiceSimulationInput(TypedDict, total=False):
    """InvoiceSimulationInput is the input for simulating an invoice.

    :ivar number: The number of the invoice.
    :vartype number: str
    :ivar currency: Currency for all invoice line items.

     Multi currency invoices are not supported yet. Required.
    :vartype currency: str
    :ivar lines: Lines to be included in the generated invoice. Required.
    :vartype lines: list["InvoiceSimulationLine"]
    """

    number: str
    """The number of the invoice."""
    currency: Required[str]
    """Currency for all invoice line items.
     
     Multi currency invoices are not supported yet. Required."""
    lines: Required[list["InvoiceSimulationLine"]]
    """Lines to be included in the generated invoice. Required."""


class InvoiceSimulationLine(TypedDict, total=False):
    """InvoiceSimulationLine represents a usage-based line item that can be input to the simulation
    endpoint.

    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: "Metadata"
    :ivar tax_config: Tax config specify the tax configuration for this line.
    :vartype tax_config: "TaxConfig"
    :ivar period: Period of the line item applies to for revenue recognition pruposes.

     Billing always treats periods as start being inclusive and end being exclusive. Required.
    :vartype period: "Period"
    :ivar invoice_at: The time this line item should be invoiced. Required.
    :vartype invoice_at: str
    :ivar price: Price of the usage-based item being sold. Is one of the following types:
     FlatPriceWithPaymentTerm, UnitPriceWithCommitments, TieredPriceWithCommitments,
     DynamicPriceWithCommitments, PackagePriceWithCommitments
    :vartype price: "_unions.RateCardUsageBasedPrice"
    :ivar feature_key: The feature that the usage is based on.
    :vartype feature_key: str
    :ivar rate_card: The rate card that is used for this line.

     The rate card captures the intent of the price and discounts for the usage-based item.
    :vartype rate_card: "InvoiceUsageBasedRateCard"
    :ivar quantity: The quantity of the item being sold. Required.
    :vartype quantity: str
    :ivar pre_line_period_quantity: The quantity of the item used before this line's period, if the
     line is billed progressively.
    :vartype pre_line_period_quantity: str
    :ivar id: ID of the line. If not specified it will be auto-generated.

     When discounts are specified, this must be provided, so that the discount can reference it.
    :vartype id: str
    """

    name: Required[str]
    """Display name. Required."""
    description: str
    """Description."""
    metadata: Optional["Metadata"]
    """Metadata."""
    taxConfig: "TaxConfig"
    """Tax config specify the tax configuration for this line."""
    period: Required["Period"]
    """Period of the line item applies to for revenue recognition pruposes.
     
     Billing always treats periods as start being inclusive and end being exclusive. Required."""
    invoiceAt: Required[str]
    """The time this line item should be invoiced. Required."""
    price: "_unions.RateCardUsageBasedPrice"
    """Price of the usage-based item being sold. Is one of the following types:
     FlatPriceWithPaymentTerm, UnitPriceWithCommitments, TieredPriceWithCommitments,
     DynamicPriceWithCommitments, PackagePriceWithCommitments"""
    featureKey: str
    """The feature that the usage is based on."""
    rateCard: "InvoiceUsageBasedRateCard"
    """The rate card that is used for this line.
     
     The rate card captures the intent of the price and discounts for the usage-based item."""
    quantity: Required[str]
    """The quantity of the item being sold. Required."""
    preLinePeriodQuantity: str
    """The quantity of the item used before this line's period, if the line is billed progressively."""
    id: str
    """ID of the line. If not specified it will be auto-generated.
     
     When discounts are specified, this must be provided, so that the discount can reference it."""


class InvoiceUsageBasedRateCard(TypedDict, total=False):
    """InvoiceUsageBasedRateCard represents the rate card (intent) for an usage-based line.

    :ivar feature_key: Feature key.
    :vartype feature_key: str
    :ivar tax_config: Tax config.
    :vartype tax_config: "TaxConfig"
    :ivar price: The price of the rate card. When null, the feature or service is free. Required.
     Is one of the following types: FlatPriceWithPaymentTerm, UnitPriceWithCommitments,
     TieredPriceWithCommitments, DynamicPriceWithCommitments, PackagePriceWithCommitments
    :vartype price: "_unions.RateCardUsageBasedPrice"
    :ivar discounts: The discounts that are applied to the line.
    :vartype discounts: "BillingDiscounts"
    """

    featureKey: str
    """Feature key."""
    taxConfig: "TaxConfig"
    """Tax config."""
    price: Required[Optional["_unions.RateCardUsageBasedPrice"]]
    """The price of the rate card. When null, the feature or service is free. Required. Is one of the
     following types: FlatPriceWithPaymentTerm, UnitPriceWithCommitments,
     TieredPriceWithCommitments, DynamicPriceWithCommitments, PackagePriceWithCommitments"""
    discounts: "BillingDiscounts"
    """The discounts that are applied to the line."""


class InvoiceWorkflowInvoicingSettingsReplaceUpdate(TypedDict, total=False):  # pylint: disable=name-too-long
    """InvoiceWorkflowInvoicingSettingsReplaceUpdate represents the update model for the invoicing
    settings of an invoice workflow.

    :ivar auto_advance: Whether to automatically issue the invoice after the draftPeriod has
     passed.
    :vartype auto_advance: bool
    :ivar draft_period: The period for the invoice to be kept in draft status for manual reviews.
    :vartype draft_period: str
    :ivar due_after: The period after which the invoice is due. With some payment solutions it's
     only applicable for manual collection method.
    :vartype due_after: str
    :ivar subscription_end_proration_mode: Controls how subscription-ending shortened service
     periods are billed. Known values are: "bill_full_period" and "bill_actual_period".
    :vartype subscription_end_proration_mode: Union[str,
     "BillingWorkflowInvoicingSubscriptionEndProrationMode"]
    :ivar default_tax_config: Default tax configuration to apply to the invoices.

     Setting a tax code (``stripe.code`` / ``taxCodeId``) on a profile's default tax config is
     deprecated and can no longer be added or changed: the organization default tax code is
     used instead. Existing tax-code values may still be removed, and ``behavior`` remains
     fully supported.
    :vartype default_tax_config: "TaxConfig"
    """

    autoAdvance: bool
    """Whether to automatically issue the invoice after the draftPeriod has passed."""
    draftPeriod: str
    """The period for the invoice to be kept in draft status for manual reviews."""
    dueAfter: str
    """The period after which the invoice is due. With some payment solutions it's only applicable for
     manual collection method."""
    subscriptionEndProrationMode: Union[str, "BillingWorkflowInvoicingSubscriptionEndProrationMode"]
    """Controls how subscription-ending shortened service periods are billed. Known values are:
     \"bill_full_period\" and \"bill_actual_period\"."""
    defaultTaxConfig: "TaxConfig"
    """Default tax configuration to apply to the invoices.
     
     Setting a tax code (``stripe.code`` / ``taxCodeId``) on a profile's default tax config is
     deprecated and can no longer be added or changed: the organization default tax code is
     used instead. Existing tax-code values may still be removed, and ``behavior`` remains
     fully supported."""


class InvoiceWorkflowReplaceUpdate(TypedDict, total=False):
    """InvoiceWorkflowReplaceUpdate represents the update model for an invoice workflow.

    Fields that are immutable a re removed from the model. This is based on
    InvoiceWorkflowSettings.

    :ivar workflow: The workflow used for this invoice. Required.
    :vartype workflow: "InvoiceWorkflowSettingsReplaceUpdate"
    """

    workflow: Required["InvoiceWorkflowSettingsReplaceUpdate"]
    """The workflow used for this invoice. Required."""


class InvoiceWorkflowSettingsReplaceUpdate(TypedDict, total=False):
    """Mutable workflow settings for an invoice.

    Other fields on the invoice's workflow are not mutable, they serve as a history of the
    invoice's workflow
    at creation time.

    :ivar invoicing: The invoicing settings for this workflow. Required.
    :vartype invoicing: "InvoiceWorkflowInvoicingSettingsReplaceUpdate"
    :ivar payment: The payment settings for this workflow. Required.
    :vartype payment: "BillingWorkflowPaymentSettings"
    """

    invoicing: Required["InvoiceWorkflowInvoicingSettingsReplaceUpdate"]
    """The invoicing settings for this workflow. Required."""
    payment: Required["BillingWorkflowPaymentSettings"]
    """The payment settings for this workflow. Required."""


class IssueAfterReset(TypedDict, total=False):
    """Issue after reset.

    :ivar amount: Initial grant amount. Required.
    :vartype amount: float
    :ivar priority: Issue grant after reset priority.
    :vartype priority: int
    """

    amount: Required[float]
    """Initial grant amount. Required."""
    priority: int
    """Issue grant after reset priority."""


class ListRequestFilter(TypedDict, total=False):
    """ListRequestFilter.

    :ivar id:
    :vartype id: "FilterString"
    :ivar source:
    :vartype source: "FilterString"
    :ivar subject:
    :vartype subject: "FilterString"
    :ivar customer_id:
    :vartype customer_id: "FilterIDExact"
    :ivar type:
    :vartype type: "FilterString"
    :ivar time:
    :vartype time: "FilterTime"
    :ivar ingested_at:
    :vartype ingested_at: "FilterTime"
    """

    id: "FilterString"
    source: "FilterString"
    subject: "FilterString"
    customerId: "FilterIDExact"
    type: "FilterString"
    time: "FilterTime"
    ingestedAt: "FilterTime"


class MarketplaceInstallRequestPayload(TypedDict, total=False):
    """Marketplace install request payload.

    :ivar name: Name of the application to install.

     If name is not provided defaults to the marketplace listing's name.
    :vartype name: str
    :ivar create_billing_profile: If true, a billing profile will be created for the app. The
     Stripe app will be also set as the default billing profile if the current default is a Sandbox
     app.
    :vartype create_billing_profile: bool
    """

    name: str
    """Name of the application to install.
     
     If name is not provided defaults to the marketplace listing's name."""
    createBillingProfile: bool
    """If true, a billing profile will be created for the app. The Stripe app will be also set as the
     default billing profile if the current default is a Sandbox app."""


class MarketplaceListing(TypedDict, total=False):
    """A marketplace listing.
    Represent an available app in the app marketplace that can be installed to the organization.

    Marketplace apps only exist in config so they don't extend the Resource model.

    :ivar type: The app's type. Required. Known values are: "stripe", "sandbox", and
     "custom_invoicing".
    :vartype type: Union[str, "AppType"]
    :ivar name: The app's name. Required.
    :vartype name: str
    :ivar description: The app's description. Required.
    :vartype description: str
    :ivar capabilities: The app's capabilities. Required.
    :vartype capabilities: list["AppCapability"]
    :ivar install_methods: Install methods.

     List of methods to install the app. Required.
    :vartype install_methods: list[Union[str, "InstallMethod"]]
    """

    type: Required[Union[str, "AppType"]]
    """The app's type. Required. Known values are: \"stripe\", \"sandbox\", and \"custom_invoicing\"."""
    name: Required[str]
    """The app's name. Required."""
    description: Required[str]
    """The app's description. Required."""
    capabilities: Required[list["AppCapability"]]
    """The app's capabilities. Required."""
    installMethods: Required[list[Union[str, "InstallMethod"]]]
    """Install methods.
     
     List of methods to install the app. Required."""


class Metadata(TypedDict, total=False):
    """Set of key-value pairs. Metadata can be used to store additional information about a resource."""


class MeterCreate(TypedDict, total=False):
    """A meter create model.

    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: "Metadata"
    :ivar name: Display name.
    :vartype name: str
    :ivar slug: A unique, human-readable identifier for the meter. Must consist only alphanumeric
     and underscore characters. Required.
    :vartype slug: str
    :ivar aggregation: The aggregation type to use for the meter. Required. Known values are:
     "SUM", "COUNT", "UNIQUE_COUNT", "AVG", "MIN", "MAX", and "LATEST".
    :vartype aggregation: Union[str, "MeterAggregation"]
    :ivar event_type: The event type to aggregate. Required.
    :vartype event_type: str
    :ivar event_from: The date since the meter should include events. Useful to skip old events. If
     not specified, all historical events are included.
    :vartype event_from: str
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

    description: str
    """Description."""
    metadata: Optional["Metadata"]
    """Metadata."""
    name: str
    """Display name."""
    slug: Required[str]
    """A unique, human-readable identifier for the meter. Must consist only alphanumeric and
     underscore characters. Required."""
    aggregation: Required[Union[str, "MeterAggregation"]]
    """The aggregation type to use for the meter. Required. Known values are: \"SUM\", \"COUNT\",
     \"UNIQUE_COUNT\", \"AVG\", \"MIN\", \"MAX\", and \"LATEST\"."""
    eventType: Required[str]
    """The event type to aggregate. Required."""
    eventFrom: str
    """The date since the meter should include events. Useful to skip old events. If not specified,
     all historical events are included."""
    valueProperty: str
    """JSONPath expression to extract the value from the ingested event's data property.
     
     The ingested value for SUM, AVG, MIN, and MAX aggregations is a number or a string that can be
     parsed to a number.
     
     For UNIQUE_COUNT aggregation, the ingested value must be a string. For COUNT aggregation the
     valueProperty is ignored."""
    groupBy: dict[str, str]
    """Named JSONPath expressions to extract the group by values from the event data.
     
     Keys must be unique and consist only alphanumeric and underscore characters."""


MeterQueryRequest = TypedDict(
    "MeterQueryRequest",
    {
        "clientId": str,
        "from": str,
        "to": str,
        "windowSize": Union[str, "WindowSize"],
        "windowTimeZone": str,
        "subject": list[str],
        "filterCustomerId": list[str],
        "filterGroupBy": dict[str, list[str]],
        "advancedMeterGroupByFilters": dict[str, "FilterString"],
        "groupBy": list[str],
    },
    total=False,
)
MeterQueryRequest.__doc__ = """A meter query request.

:ivar client_id: Client ID Useful to track progress of a query.
:vartype client_id: str
:ivar from_property: Start date-time in RFC 3339 format.
 
 Inclusive.
:vartype from_property: str
:ivar to: End date-time in RFC 3339 format.
 
 Inclusive.
:vartype to: str
:ivar window_size: If not specified, a single usage aggregate will be returned for the entirety
 of the specified period for each subject and group. Known values are: "MINUTE", "HOUR", "DAY",
 and "MONTH".
:vartype window_size: Union[str, "WindowSize"]
:ivar window_time_zone: The value is the name of the time zone as defined in the IANA Time Zone
 Database (`http://www.iana.org/time-zones <http://www.iana.org/time-zones>`_). If not
 specified, the UTC timezone will be used.
:vartype window_time_zone: str
:ivar subject: Filtering by multiple subjects.
:vartype subject: list[str]
:ivar filter_customer_id: Filtering by multiple customers.
:vartype filter_customer_id: list[str]
:ivar filter_group_by: Simple filter for group bys with exact match.
:vartype filter_group_by: dict[str, list[str]]
:ivar advanced_meter_group_by_filters: Optional advanced meter group by filters. You can use
 this to filter for values of the meter groupBy fields.
:vartype advanced_meter_group_by_filters: dict[str, "FilterString"]
:ivar group_by: If not specified a single aggregate will be returned for each subject and time
 window. ``subject`` is a reserved group by value.
:vartype group_by: list[str]
"""


class MeterUpdate(TypedDict, total=False):
    """A meter update model.

    Only the properties that can be updated are included.
    For example, the slug and aggregation cannot be updated.

    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: "Metadata"
    :ivar name: Display name.
    :vartype name: str
    :ivar group_by: Named JSONPath expressions to extract the group by values from the event data.

     Keys must be unique and consist only alphanumeric and underscore characters.
    :vartype group_by: dict[str, str]
    """

    description: str
    """Description."""
    metadata: Optional["Metadata"]
    """Metadata."""
    name: str
    """Display name."""
    groupBy: dict[str, str]
    """Named JSONPath expressions to extract the group by values from the event data.
     
     Keys must be unique and consist only alphanumeric and underscore characters."""


class MigrateRequest(TypedDict, total=False):
    """MigrateRequest.

    :ivar timing: Timing configuration for the migration, when the migration should take effect. If
     not supported by the subscription, 400 will be returned. Is either a Union[str,
     "_models.SubscriptionTimingEnum"] type or a datetime.datetime type.
    :vartype timing: "_unions.SubscriptionTiming"
    :ivar target_version: The version of the plan to migrate to. If not provided, the subscription
     will migrate to the latest version of the current plan.
    :vartype target_version: int
    :ivar starting_phase: The key of the phase to start the subscription in. If not provided, the
     subscription will start in the first phase of the plan.
    :vartype starting_phase: str
    :ivar billing_anchor: The billing anchor of the subscription. The provided date will be
     normalized according to the billing cadence to the nearest recurrence before start time. If not
     provided, the previous subscription billing anchor will be used.
    :vartype billing_anchor: str
    """

    timing: "_unions.SubscriptionTiming"
    """Timing configuration for the migration, when the migration should take effect. If not supported
     by the subscription, 400 will be returned. Is either a Union[str,
     \"_models.SubscriptionTimingEnum\"] type or a datetime.datetime type."""
    targetVersion: int
    """The version of the plan to migrate to. If not provided, the subscription will migrate to the
     latest version of the current plan."""
    startingPhase: str
    """The key of the phase to start the subscription in. If not provided, the subscription will start
     in the first phase of the plan."""
    billingAnchor: str
    """The billing anchor of the subscription. The provided date will be normalized according to the
     billing cadence to the nearest recurrence before start time. If not provided, the previous
     subscription billing anchor will be used."""


class NotificationChannelWebhookCreateRequest(TypedDict, total=False):
    """Request with input parameters for creating new notification channel with webhook type.

    :ivar type: Channel Type. Required. WEBHOOK.
    :vartype type: Literal[NotificationChannelType.WEBHOOK]
    :ivar name: Channel Name. Required.
    :vartype name: str
    :ivar disabled: Channel Disabled.
    :vartype disabled: bool
    :ivar metadata: Metadata.
    :vartype metadata: "Metadata"
    :ivar url: Webhook URL. Required.
    :vartype url: str
    :ivar custom_headers: Custom HTTP Headers.
    :vartype custom_headers: dict[str, str]
    :ivar signing_secret: Signing Secret.
    :vartype signing_secret: str
    """

    type: Required[Literal[NotificationChannelType.WEBHOOK]]
    """Channel Type. Required. WEBHOOK."""
    name: Required[str]
    """Channel Name. Required."""
    disabled: bool
    """Channel Disabled."""
    metadata: Optional["Metadata"]
    """Metadata."""
    url: Required[str]
    """Webhook URL. Required."""
    customHeaders: dict[str, str]
    """Custom HTTP Headers."""
    signingSecret: str
    """Signing Secret."""


class NotificationEventResendRequest(TypedDict, total=False):
    """A notification event that will be re-sent.

    :ivar channels: Channels.
    :vartype channels: list[str]
    """

    channels: list[str]
    """Channels."""


class NotificationRuleBalanceThresholdCreateRequest(TypedDict, total=False):  # pylint: disable=name-too-long
    """Request with input parameters for creating new notification rule with
    entitlements.balance.threshold type.

    :ivar type: Rule Type. Required. ENTITLEMENTS_BALANCE_THRESHOLD.
    :vartype type: Literal[NotificationEventType.ENTITLEMENTS_BALANCE_THRESHOLD]
    :ivar name: Rule Name. Required.
    :vartype name: str
    :ivar disabled: Rule Disabled.
    :vartype disabled: bool
    :ivar metadata: Metadata.
    :vartype metadata: "Metadata"
    :ivar thresholds: Entitlement Balance Thresholds. Required.
    :vartype thresholds: list["NotificationRuleBalanceThresholdValue"]
    :ivar channels: Channels. Required.
    :vartype channels: list[str]
    :ivar features: Features.
    :vartype features: list[str]
    """

    type: Required[Literal[NotificationEventType.ENTITLEMENTS_BALANCE_THRESHOLD]]
    """Rule Type. Required. ENTITLEMENTS_BALANCE_THRESHOLD."""
    name: Required[str]
    """Rule Name. Required."""
    disabled: bool
    """Rule Disabled."""
    metadata: Optional["Metadata"]
    """Metadata."""
    thresholds: Required[list["NotificationRuleBalanceThresholdValue"]]
    """Entitlement Balance Thresholds. Required."""
    channels: Required[list[str]]
    """Channels. Required."""
    features: list[str]
    """Features."""


class NotificationRuleBalanceThresholdValue(TypedDict, total=False):
    """Threshold value with multiple supported types.

    :ivar value: Threshold Value. Required.
    :vartype value: float
    :ivar type: Type of the threshold. Required. Known values are: "PERCENT", "NUMBER",
     "balance_value", "usage_percentage", and "usage_value".
    :vartype type: Union[str, "NotificationRuleBalanceThresholdValueType"]
    """

    value: Required[float]
    """Threshold Value. Required."""
    type: Required[Union[str, "NotificationRuleBalanceThresholdValueType"]]
    """Type of the threshold. Required. Known values are: \"PERCENT\", \"NUMBER\", \"balance_value\",
     \"usage_percentage\", and \"usage_value\"."""


class NotificationRuleEntitlementResetCreateRequest(TypedDict, total=False):  # pylint: disable=name-too-long
    """Request with input parameters for creating new notification rule with entitlements.reset type.

    :ivar type: Rule Type. Required. ENTITLEMENTS_RESET.
    :vartype type: Literal[NotificationEventType.ENTITLEMENTS_RESET]
    :ivar name: Rule Name. Required.
    :vartype name: str
    :ivar disabled: Rule Disabled.
    :vartype disabled: bool
    :ivar metadata: Metadata.
    :vartype metadata: "Metadata"
    :ivar channels: Channels. Required.
    :vartype channels: list[str]
    :ivar features: Features.
    :vartype features: list[str]
    """

    type: Required[Literal[NotificationEventType.ENTITLEMENTS_RESET]]
    """Rule Type. Required. ENTITLEMENTS_RESET."""
    name: Required[str]
    """Rule Name. Required."""
    disabled: bool
    """Rule Disabled."""
    metadata: Optional["Metadata"]
    """Metadata."""
    channels: Required[list[str]]
    """Channels. Required."""
    features: list[str]
    """Features."""


class NotificationRuleInvoiceCreatedCreateRequest(TypedDict, total=False):  # pylint: disable=name-too-long
    """Request with input parameters for creating new notification rule with invoice.created type.

    :ivar type: Rule Type. Required. INVOICE_CREATED.
    :vartype type: Literal[NotificationEventType.INVOICE_CREATED]
    :ivar name: Rule Name. Required.
    :vartype name: str
    :ivar disabled: Rule Disabled.
    :vartype disabled: bool
    :ivar metadata: Metadata.
    :vartype metadata: "Metadata"
    :ivar channels: Channels. Required.
    :vartype channels: list[str]
    """

    type: Required[Literal[NotificationEventType.INVOICE_CREATED]]
    """Rule Type. Required. INVOICE_CREATED."""
    name: Required[str]
    """Rule Name. Required."""
    disabled: bool
    """Rule Disabled."""
    metadata: Optional["Metadata"]
    """Metadata."""
    channels: Required[list[str]]
    """Channels. Required."""


class NotificationRuleInvoiceUpdatedCreateRequest(TypedDict, total=False):  # pylint: disable=name-too-long
    """Request with input parameters for creating new notification rule with invoice.updated  type.

    :ivar type: Rule Type. Required. INVOICE_UPDATED.
    :vartype type: Literal[NotificationEventType.INVOICE_UPDATED]
    :ivar name: Rule Name. Required.
    :vartype name: str
    :ivar disabled: Rule Disabled.
    :vartype disabled: bool
    :ivar metadata: Metadata.
    :vartype metadata: "Metadata"
    :ivar channels: Channels. Required.
    :vartype channels: list[str]
    """

    type: Required[Literal[NotificationEventType.INVOICE_UPDATED]]
    """Rule Type. Required. INVOICE_UPDATED."""
    name: Required[str]
    """Rule Name. Required."""
    disabled: bool
    """Rule Disabled."""
    metadata: Optional["Metadata"]
    """Metadata."""
    channels: Required[list[str]]
    """Channels. Required."""


class PackagePriceWithCommitments(TypedDict, total=False):
    """Package price with spend commitments.

    :ivar type: The type of the price. Required. PACKAGE.
    :vartype type: Literal[PriceType.PACKAGE]
    :ivar amount: Amount. Required.
    :vartype amount: str
    :ivar quantity_per_package: Quantity per package. Required.
    :vartype quantity_per_package: str
    :ivar minimum_amount: Minimum amount.
    :vartype minimum_amount: str
    :ivar maximum_amount: Maximum amount.
    :vartype maximum_amount: str
    """

    type: Required[Literal[PriceType.PACKAGE]]
    """The type of the price. Required. PACKAGE."""
    amount: Required[str]
    """Amount. Required."""
    quantityPerPackage: Required[str]
    """Quantity per package. Required."""
    minimumAmount: str
    """Minimum amount."""
    maximumAmount: str
    """Maximum amount."""


Period = TypedDict(
    "Period",
    {
        "from": Required[str],
        "to": Required[str],
    },
    total=False,
)
Period.__doc__ = """A period with a start and end time.

:ivar from_property: Period start time. Required.
:vartype from_property: str
:ivar to: Period end time. Required.
:vartype to: str
"""


class PlanAddonCreate(TypedDict, total=False):
    """A plan add-on assignment create request.

    :ivar metadata: Metadata.
    :vartype metadata: "Metadata"
    :ivar from_plan_phase: The plan phase from the add-on becomes purchasable. Required.
    :vartype from_plan_phase: str
    :ivar max_quantity: Max quantity of the add-on.
    :vartype max_quantity: int
    :ivar addon_id: Add-on unique identifier. Required.
    :vartype addon_id: str
    """

    metadata: "Metadata"
    """Metadata."""
    fromPlanPhase: Required[str]
    """The plan phase from the add-on becomes purchasable. Required."""
    maxQuantity: int
    """Max quantity of the add-on."""
    addonId: Required[str]
    """Add-on unique identifier. Required."""


class PlanAddonReplaceUpdate(TypedDict, total=False):
    """Resource update operation model.

    :ivar metadata: Metadata.
    :vartype metadata: "Metadata"
    :ivar from_plan_phase: The plan phase from the add-on becomes purchasable. Required.
    :vartype from_plan_phase: str
    :ivar max_quantity: Max quantity of the add-on.
    :vartype max_quantity: int
    """

    metadata: "Metadata"
    """Metadata."""
    fromPlanPhase: Required[str]
    """The plan phase from the add-on becomes purchasable. Required."""
    maxQuantity: int
    """Max quantity of the add-on."""


class PlanCreate(TypedDict, total=False):
    """Resource create operation model.

    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: "Metadata"
    :ivar key: Key. Required.
    :vartype key: str
    :ivar alignment: Alignment configuration for the plan.
    :vartype alignment: "Alignment"
    :ivar currency: Currency. Required.
    :vartype currency: str
    :ivar billing_cadence: Billing cadence. Required.
    :vartype billing_cadence: str
    :ivar pro_rating_config: Pro-rating configuration.
    :vartype pro_rating_config: "ProRatingConfig"
    :ivar settlement_mode: Settlement mode. Known values are: "credit_then_invoice" and
     "credit_only".
    :vartype settlement_mode: Union[str, "BillingSettlementMode"]
    :ivar phases: Plan phases. Required.
    :vartype phases: list["PlanPhase"]
    """

    name: Required[str]
    """Display name. Required."""
    description: str
    """Description."""
    metadata: Optional["Metadata"]
    """Metadata."""
    key: Required[str]
    """Key. Required."""
    alignment: "Alignment"
    """Alignment configuration for the plan."""
    currency: Required[str]
    """Currency. Required."""
    billingCadence: Required[str]
    """Billing cadence. Required."""
    proRatingConfig: "ProRatingConfig"
    """Pro-rating configuration."""
    settlementMode: Union[str, "BillingSettlementMode"]
    """Settlement mode. Known values are: \"credit_then_invoice\" and \"credit_only\"."""
    phases: Required[list["PlanPhase"]]
    """Plan phases. Required."""


class PlanPhase(TypedDict, total=False):
    """The plan phase or pricing ramp allows changing a plan's rate cards over time as a subscription
    progresses.

    :ivar key: Key. Required.
    :vartype key: str
    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: "Metadata"
    :ivar duration: Duration. Required.
    :vartype duration: str
    :ivar rate_cards: Rate cards. Required.
    :vartype rate_cards: list["_unions.RateCard"]
    """

    key: Required[str]
    """Key. Required."""
    name: Required[str]
    """Display name. Required."""
    description: str
    """Description."""
    metadata: Optional["Metadata"]
    """Metadata."""
    duration: Required[Optional[str]]
    """Duration. Required."""
    rateCards: Required[list["_unions.RateCard"]]
    """Rate cards. Required."""


class PlanReferenceInput(TypedDict, total=False):
    """References an exact plan defaulting to the current active version.

    :ivar key: The plan key. Required.
    :vartype key: str
    :ivar version: The plan version.
    :vartype version: int
    """

    key: Required[str]
    """The plan key. Required."""
    version: int
    """The plan version."""


class PlanReplaceUpdate(TypedDict, total=False):
    """Resource update operation model.

    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: "Metadata"
    :ivar alignment: Alignment configuration for the plan.
    :vartype alignment: "Alignment"
    :ivar billing_cadence: Billing cadence. Required.
    :vartype billing_cadence: str
    :ivar pro_rating_config: Pro-rating configuration.
    :vartype pro_rating_config: "ProRatingConfig"
    :ivar settlement_mode: Settlement mode. Known values are: "credit_then_invoice" and
     "credit_only".
    :vartype settlement_mode: Union[str, "BillingSettlementMode"]
    :ivar phases: Plan phases. Required.
    :vartype phases: list["PlanPhase"]
    """

    name: Required[str]
    """Display name. Required."""
    description: str
    """Description."""
    metadata: Optional["Metadata"]
    """Metadata."""
    alignment: "Alignment"
    """Alignment configuration for the plan."""
    billingCadence: Required[str]
    """Billing cadence. Required."""
    proRatingConfig: "ProRatingConfig"
    """Pro-rating configuration."""
    settlementMode: Union[str, "BillingSettlementMode"]
    """Settlement mode. Known values are: \"credit_then_invoice\" and \"credit_only\"."""
    phases: Required[list["PlanPhase"]]
    """Plan phases. Required."""


class PlanSubscriptionChange(TypedDict, total=False):
    """Change subscription based on plan.

    :ivar timing: Timing configuration for the change, when the change should take effect. For
     changing a subscription, the accepted values depend on the subscription configuration.
     Required. Is either a Union[str, "_models.SubscriptionTimingEnum"] type or a datetime.datetime
     type.
    :vartype timing: "_unions.SubscriptionTiming"
    :ivar alignment: What alignment settings the subscription should have.
    :vartype alignment: "Alignment"
    :ivar metadata: Arbitrary metadata associated with the subscription.
    :vartype metadata: "Metadata"
    :ivar plan: The plan reference to change to. Required.
    :vartype plan: "PlanReferenceInput"
    :ivar starting_phase: The key of the phase to start the subscription in. If not provided, the
     subscription will start in the first phase of the plan.
    :vartype starting_phase: str
    :ivar name: The name of the Subscription. If not provided the plan name is used.
    :vartype name: str
    :ivar description: Description for the Subscription.
    :vartype description: str
    :ivar billing_anchor: The billing anchor of the subscription. The provided date will be
     normalized according to the billing cadence to the nearest recurrence before start time. If not
     provided, the previous subscription billing anchor will be used.
    :vartype billing_anchor: str
    :ivar settlement_mode: The settlement mode of the subscription. Known values are:
     "credit_then_invoice" and "credit_only".
    :vartype settlement_mode: Union[str, "BillingSettlementMode"]
    """

    timing: Required["_unions.SubscriptionTiming"]
    """Timing configuration for the change, when the change should take effect. For changing a
     subscription, the accepted values depend on the subscription configuration. Required. Is either
     a Union[str, \"_models.SubscriptionTimingEnum\"] type or a datetime.datetime type."""
    alignment: "Alignment"
    """What alignment settings the subscription should have."""
    metadata: "Metadata"
    """Arbitrary metadata associated with the subscription."""
    plan: Required["PlanReferenceInput"]
    """The plan reference to change to. Required."""
    startingPhase: str
    """The key of the phase to start the subscription in. If not provided, the subscription will start
     in the first phase of the plan."""
    name: str
    """The name of the Subscription. If not provided the plan name is used."""
    description: str
    """Description for the Subscription."""
    billingAnchor: str
    """The billing anchor of the subscription. The provided date will be normalized according to the
     billing cadence to the nearest recurrence before start time. If not provided, the previous
     subscription billing anchor will be used."""
    settlementMode: Union[str, "BillingSettlementMode"]
    """The settlement mode of the subscription. Known values are: \"credit_then_invoice\" and
     \"credit_only\"."""


class PlanSubscriptionCreate(TypedDict, total=False):
    """Create from plan.

    :ivar alignment: What alignment settings the subscription should have.
    :vartype alignment: "Alignment"
    :ivar metadata: Arbitrary metadata associated with the subscription.
    :vartype metadata: "Metadata"
    :ivar plan: The plan reference to change to. Required.
    :vartype plan: "PlanReferenceInput"
    :ivar starting_phase: The key of the phase to start the subscription in. If not provided, the
     subscription will start in the first phase of the plan.
    :vartype starting_phase: str
    :ivar name: The name of the Subscription. If not provided the plan name is used.
    :vartype name: str
    :ivar description: Description for the Subscription.
    :vartype description: str
    :ivar settlement_mode: The settlement mode of the subscription. Known values are:
     "credit_then_invoice" and "credit_only".
    :vartype settlement_mode: Union[str, "BillingSettlementMode"]
    :ivar timing: Timing configuration for the change, when the change should take effect. The
     default is immediate. Is either a Union[str, "_models.SubscriptionTimingEnum"] type or a
     datetime.datetime type.
    :vartype timing: "_unions.SubscriptionTiming"
    :ivar customer_id: The ID of the customer. Provide either the key or ID. Has presedence over
     the key.
    :vartype customer_id: str
    :ivar customer_key: The key of the customer. Provide either the key or ID.
    :vartype customer_key: str
    :ivar billing_anchor: The billing anchor of the subscription. The provided date will be
     normalized according to the billing cadence to the nearest recurrence before start time. If not
     provided, the subscription start time will be used.
    :vartype billing_anchor: str
    """

    alignment: "Alignment"
    """What alignment settings the subscription should have."""
    metadata: "Metadata"
    """Arbitrary metadata associated with the subscription."""
    plan: Required["PlanReferenceInput"]
    """The plan reference to change to. Required."""
    startingPhase: str
    """The key of the phase to start the subscription in. If not provided, the subscription will start
     in the first phase of the plan."""
    name: str
    """The name of the Subscription. If not provided the plan name is used."""
    description: str
    """Description for the Subscription."""
    settlementMode: Union[str, "BillingSettlementMode"]
    """The settlement mode of the subscription. Known values are: \"credit_then_invoice\" and
     \"credit_only\"."""
    timing: "_unions.SubscriptionTiming"
    """Timing configuration for the change, when the change should take effect. The default is
     immediate. Is either a Union[str, \"_models.SubscriptionTimingEnum\"] type or a
     datetime.datetime type."""
    customerId: str
    """The ID of the customer. Provide either the key or ID. Has presedence over the key."""
    customerKey: str
    """The key of the customer. Provide either the key or ID."""
    billingAnchor: str
    """The billing anchor of the subscription. The provided date will be normalized according to the
     billing cadence to the nearest recurrence before start time. If not provided, the subscription
     start time will be used."""


class PortalToken(TypedDict, total=False):
    """A consumer portal token.

    Validator doesn't obey required for readOnly properties
    See: `https://github.com/stoplightio/spectral/issues/1274
    <https://github.com/stoplightio/spectral/issues/1274>`_.

    :ivar id:
    :vartype id: str
    :ivar subject: Required.
    :vartype subject: str
    :ivar expires_at:
    :vartype expires_at: str
    :ivar expired:
    :vartype expired: bool
    :ivar created_at:
    :vartype created_at: str
    :ivar token: The token is only returned at creation.
    :vartype token: str
    :ivar allowed_meter_slugs: Optional, if defined only the specified meters will be allowed.
    :vartype allowed_meter_slugs: list[str]
    """

    id: str
    subject: Required[str]
    """Required."""
    expiresAt: str
    expired: bool
    createdAt: str
    token: str
    """The token is only returned at creation."""
    allowedMeterSlugs: list[str]
    """Optional, if defined only the specified meters will be allowed."""


class PriceTier(TypedDict, total=False):
    """A price tier. At least one price component is required in each tier.

    :ivar up_to_amount: Up to quantity.
    :vartype up_to_amount: str
    :ivar flat_price: Flat price component. Required.
    :vartype flat_price: "FlatPrice"
    :ivar unit_price: Unit price component. Required.
    :vartype unit_price: "UnitPrice"
    """

    upToAmount: str
    """Up to quantity."""
    flatPrice: Required[Optional["FlatPrice"]]
    """Flat price component. Required."""
    unitPrice: Required[Optional["UnitPrice"]]
    """Unit price component. Required."""


class ProRatingConfig(TypedDict, total=False):
    """Configuration for pro-rating behavior.

    :ivar enabled: Enable pro-rating. Required.
    :vartype enabled: bool
    :ivar mode: Pro-rating mode. Required. "prorate_prices"
    :vartype mode: Union[str, "ProRatingMode"]
    """

    enabled: Required[bool]
    """Enable pro-rating. Required."""
    mode: Required[Union[str, "ProRatingMode"]]
    """Pro-rating mode. Required. \"prorate_prices\""""


class RateCardBooleanEntitlement(TypedDict, total=False):
    """Entitlement template of a boolean entitlement.

    :ivar metadata: Additional metadata for the feature.
    :vartype metadata: "Metadata"
    :ivar type: Required. BOOLEAN.
    :vartype type: Literal[EntitlementType.BOOLEAN]
    """

    metadata: "Metadata"
    """Additional metadata for the feature."""
    type: Required[Literal[EntitlementType.BOOLEAN]]
    """Required. BOOLEAN."""


class RateCardFlatFee(TypedDict, total=False):
    """A flat fee rate card defines a one-time purchase or a recurring fee.

    :ivar type: RateCard type. Required. FLAT_FEE.
    :vartype type: Literal[RateCardType.FLAT_FEE]
    :ivar key: Key. Required.
    :vartype key: str
    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: "Metadata"
    :ivar feature_key: Feature key.
    :vartype feature_key: str
    :ivar entitlement_template: The entitlement of the rate card. Only available when featureKey is
     set. Is one of the following types: RateCardMeteredEntitlement, RateCardStaticEntitlement,
     RateCardBooleanEntitlement
    :vartype entitlement_template: "_unions.RateCardEntitlement"
    :ivar tax_config: Tax config.
    :vartype tax_config: "TaxConfig"
    :ivar billing_cadence: Billing cadence. Required.
    :vartype billing_cadence: str
    :ivar price: Price. Required.
    :vartype price: "FlatPriceWithPaymentTerm"
    :ivar discounts: Discounts.
    :vartype discounts: "Discounts"
    """

    type: Required[Literal[RateCardType.FLAT_FEE]]
    """RateCard type. Required. FLAT_FEE."""
    key: Required[str]
    """Key. Required."""
    name: Required[str]
    """Display name. Required."""
    description: str
    """Description."""
    metadata: Optional["Metadata"]
    """Metadata."""
    featureKey: str
    """Feature key."""
    entitlementTemplate: "_unions.RateCardEntitlement"
    """The entitlement of the rate card. Only available when featureKey is set. Is one of the
     following types: RateCardMeteredEntitlement, RateCardStaticEntitlement,
     RateCardBooleanEntitlement"""
    taxConfig: "TaxConfig"
    """Tax config."""
    billingCadence: Required[Optional[str]]
    """Billing cadence. Required."""
    price: Required[Optional["FlatPriceWithPaymentTerm"]]
    """Price. Required."""
    discounts: "Discounts"
    """Discounts."""


class RateCardMeteredEntitlement(TypedDict, total=False):
    """The entitlement template with a metered entitlement.

    :ivar metadata: Additional metadata for the feature.
    :vartype metadata: "Metadata"
    :ivar type: Required. METERED.
    :vartype type: Literal[EntitlementType.METERED]
    :ivar is_soft_limit: Soft limit.
    :vartype is_soft_limit: bool
    :ivar issue_after_reset: Initial grant amount.
    :vartype issue_after_reset: float
    :ivar issue_after_reset_priority: Issue grant after reset priority.
    :vartype issue_after_reset_priority: int
    :ivar preserve_overage_at_reset: Preserve overage at reset.
    :vartype preserve_overage_at_reset: bool
    :ivar usage_period: Usage Period.
    :vartype usage_period: str
    """

    metadata: "Metadata"
    """Additional metadata for the feature."""
    type: Required[Literal[EntitlementType.METERED]]
    """Required. METERED."""
    isSoftLimit: bool
    """Soft limit."""
    issueAfterReset: float
    """Initial grant amount."""
    issueAfterResetPriority: int
    """Issue grant after reset priority."""
    preserveOverageAtReset: bool
    """Preserve overage at reset."""
    usagePeriod: str
    """Usage Period."""


class RateCardStaticEntitlement(TypedDict, total=False):
    """Entitlement template of a static entitlement.

    :ivar metadata: Additional metadata for the feature.
    :vartype metadata: "Metadata"
    :ivar type: Required. STATIC.
    :vartype type: Literal[EntitlementType.STATIC]
    :ivar config: The JSON parsable config of the entitlement. This value is also returned when
     checking entitlement access and it is useful for configuring fine-grained access settings to
     the feature, implemented in your own system. Has to be an object. Required.
    :vartype config: str
    """

    metadata: "Metadata"
    """Additional metadata for the feature."""
    type: Required[Literal[EntitlementType.STATIC]]
    """Required. STATIC."""
    config: Required[str]
    """The JSON parsable config of the entitlement. This value is also returned when checking
     entitlement access and it is useful for configuring fine-grained access settings to the
     feature, implemented in your own system. Has to be an object. Required."""


class RateCardUsageBased(TypedDict, total=False):
    """A usage-based rate card defines a price based on usage.

    :ivar type: RateCard type. Required. USAGE_BASED.
    :vartype type: Literal[RateCardType.USAGE_BASED]
    :ivar key: Key. Required.
    :vartype key: str
    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: "Metadata"
    :ivar feature_key: Feature key.
    :vartype feature_key: str
    :ivar entitlement_template: The entitlement of the rate card. Only available when featureKey is
     set. Is one of the following types: RateCardMeteredEntitlement, RateCardStaticEntitlement,
     RateCardBooleanEntitlement
    :vartype entitlement_template: "_unions.RateCardEntitlement"
    :ivar tax_config: Tax config.
    :vartype tax_config: "TaxConfig"
    :ivar billing_cadence: Billing cadence. Required.
    :vartype billing_cadence: str
    :ivar price: The price of the rate card. When null, the feature or service is free. Required.
     Is one of the following types: FlatPriceWithPaymentTerm, UnitPriceWithCommitments,
     TieredPriceWithCommitments, DynamicPriceWithCommitments, PackagePriceWithCommitments
    :vartype price: "_unions.RateCardUsageBasedPrice"
    :ivar discounts: Discounts.
    :vartype discounts: "Discounts"
    """

    type: Required[Literal[RateCardType.USAGE_BASED]]
    """RateCard type. Required. USAGE_BASED."""
    key: Required[str]
    """Key. Required."""
    name: Required[str]
    """Display name. Required."""
    description: str
    """Description."""
    metadata: Optional["Metadata"]
    """Metadata."""
    featureKey: str
    """Feature key."""
    entitlementTemplate: "_unions.RateCardEntitlement"
    """The entitlement of the rate card. Only available when featureKey is set. Is one of the
     following types: RateCardMeteredEntitlement, RateCardStaticEntitlement,
     RateCardBooleanEntitlement"""
    taxConfig: "TaxConfig"
    """Tax config."""
    billingCadence: Required[str]
    """Billing cadence. Required."""
    price: Required[Optional["_unions.RateCardUsageBasedPrice"]]
    """The price of the rate card. When null, the feature or service is free. Required. Is one of the
     following types: FlatPriceWithPaymentTerm, UnitPriceWithCommitments,
     TieredPriceWithCommitments, DynamicPriceWithCommitments, PackagePriceWithCommitments"""
    discounts: "Discounts"
    """Discounts."""


class RecurringPeriodCreateInput(TypedDict, total=False):
    """Recurring period with an interval and an anchor.

    :ivar interval: Interval. Required. Is either a str type or a Union[str,
     "_models.RecurringPeriodIntervalEnum"] type.
    :vartype interval: "_unions.RecurringPeriodInterval"
    :ivar anchor: Anchor time.
    :vartype anchor: str
    """

    interval: Required["_unions.RecurringPeriodInterval"]
    """Interval. Required. Is either a str type or a Union[str,
     \"_models.RecurringPeriodIntervalEnum\"] type."""
    anchor: str
    """Anchor time."""


class RecurringPeriodV2(TypedDict, total=False):
    """Recurring period with an interval and an anchor.

    :ivar interval: Interval. Required. Is either a str type or a Union[str,
     "_models.RecurringPeriodIntervalEnum"] type.
    :vartype interval: "_unions.RecurringPeriodInterval"
    :ivar anchor: Anchor time. Required.
    :vartype anchor: str
    """

    interval: Required["_unions.RecurringPeriodInterval"]
    """Interval. Required. Is either a str type or a Union[str,
     \"_models.RecurringPeriodIntervalEnum\"] type."""
    anchor: Required[str]
    """Anchor time. Required."""


class ResetEntitlementUsageInput(TypedDict, total=False):
    """Reset parameters.

    :ivar effective_at: The time at which the reset takes effect, defaults to now. The reset cannot
     be in the future. The provided value is truncated to the minute due to how historical meter
     data is stored.
    :vartype effective_at: str
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

    effectiveAt: str
    """The time at which the reset takes effect, defaults to now. The reset cannot be in the future.
     The provided value is truncated to the minute due to how historical meter data is stored."""
    retainAnchor: bool
    """Determines whether the usage period anchor is retained or reset to the effectiveAt time.
 
      * If true, the usage period anchor is retained.
      * If false, the usage period anchor is reset to the effectiveAt time."""
    preserveOverage: bool
    """Determines whether the overage is preserved or forgiven, overriding the entitlement's default
      behavior.
 
      * If true, the overage is preserved.
      * If false, the overage is forgiven."""


class SandboxApp(TypedDict, total=False):
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
    :vartype metadata: "Metadata"
    :ivar created_at: Creation Time. Required.
    :vartype created_at: str
    :ivar updated_at: Last Update Time. Required.
    :vartype updated_at: str
    :ivar deleted_at: Deletion Time.
    :vartype deleted_at: str
    :ivar listing: The marketplace listing that this installed app is based on. Required.
    :vartype listing: "MarketplaceListing"
    :ivar status: Status of the app connection. Required. Known values are: "ready" and
     "unauthorized".
    :vartype status: Union[str, "AppStatus"]
    :ivar type: The app's type is Sandbox. Required. SANDBOX.
    :vartype type: Literal[AppType.SANDBOX]
    """

    id: Required[str]
    """ID. Required."""
    name: Required[str]
    """Display name. Required."""
    description: str
    """Description."""
    metadata: Optional["Metadata"]
    """Metadata."""
    createdAt: Required[str]
    """Creation Time. Required."""
    updatedAt: Required[str]
    """Last Update Time. Required."""
    deletedAt: str
    """Deletion Time."""
    listing: Required["MarketplaceListing"]
    """The marketplace listing that this installed app is based on. Required."""
    status: Required[Union[str, "AppStatus"]]
    """Status of the app connection. Required. Known values are: \"ready\" and \"unauthorized\"."""
    type: Required[Literal[AppType.SANDBOX]]
    """The app's type is Sandbox. Required. SANDBOX."""


class SandboxAppReplaceUpdate(TypedDict, total=False):
    """Resource update operation model.

    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: "Metadata"
    :ivar type: The app's type is Sandbox. Required. SANDBOX.
    :vartype type: Literal[AppType.SANDBOX]
    """

    name: Required[str]
    """Display name. Required."""
    description: str
    """Description."""
    metadata: Optional["Metadata"]
    """Metadata."""
    type: Required[Literal[AppType.SANDBOX]]
    """The app's type is Sandbox. Required. SANDBOX."""


class SandboxCustomerAppData(TypedDict, total=False):
    """Sandbox Customer App Data.

    :ivar app: The installed sandbox app this data belongs to.
    :vartype app: "SandboxApp"
    :ivar id: App ID.
    :vartype id: str
    :ivar type: App Type. Required. SANDBOX.
    :vartype type: Literal[AppType.SANDBOX]
    """

    app: "SandboxApp"
    """The installed sandbox app this data belongs to."""
    id: str
    """App ID."""
    type: Required[Literal[AppType.SANDBOX]]
    """App Type. Required. SANDBOX."""


class StripeAPIKeyInput(TypedDict, total=False):
    """The Stripe API key input. Used to authenticate with the Stripe API.

    :ivar secret_api_key: Required.
    :vartype secret_api_key: str
    """

    secretAPIKey: Required[str]
    """Required."""


class StripeApp(TypedDict, total=False):
    """A installed Stripe app object.

    :ivar id: ID. Required.
    :vartype id: str
    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: "Metadata"
    :ivar created_at: Creation Time. Required.
    :vartype created_at: str
    :ivar updated_at: Last Update Time. Required.
    :vartype updated_at: str
    :ivar deleted_at: Deletion Time.
    :vartype deleted_at: str
    :ivar listing: The marketplace listing that this installed app is based on. Required.
    :vartype listing: "MarketplaceListing"
    :ivar status: Status of the app connection. Required. Known values are: "ready" and
     "unauthorized".
    :vartype status: Union[str, "AppStatus"]
    :ivar type: The app's type is Stripe. Required. STRIPE.
    :vartype type: Literal[AppType.STRIPE]
    :ivar stripe_account_id: The Stripe account ID. Required.
    :vartype stripe_account_id: str
    :ivar livemode: Livemode, true if the app is in production mode. Required.
    :vartype livemode: bool
    :ivar masked_api_key: The masked API key. Only shows the first 8 and last 3 characters.
     Required.
    :vartype masked_api_key: str
    :ivar secret_api_key: The Stripe API key.
    :vartype secret_api_key: str
    """

    id: Required[str]
    """ID. Required."""
    name: Required[str]
    """Display name. Required."""
    description: str
    """Description."""
    metadata: Optional["Metadata"]
    """Metadata."""
    createdAt: Required[str]
    """Creation Time. Required."""
    updatedAt: Required[str]
    """Last Update Time. Required."""
    deletedAt: str
    """Deletion Time."""
    listing: Required["MarketplaceListing"]
    """The marketplace listing that this installed app is based on. Required."""
    status: Required[Union[str, "AppStatus"]]
    """Status of the app connection. Required. Known values are: \"ready\" and \"unauthorized\"."""
    type: Required[Literal[AppType.STRIPE]]
    """The app's type is Stripe. Required. STRIPE."""
    stripeAccountId: Required[str]
    """The Stripe account ID. Required."""
    livemode: Required[bool]
    """Livemode, true if the app is in production mode. Required."""
    maskedAPIKey: Required[str]
    """The masked API key. Only shows the first 8 and last 3 characters. Required."""
    secretAPIKey: str
    """The Stripe API key."""


class StripeAppReplaceUpdate(TypedDict, total=False):
    """Resource update operation model.

    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: "Metadata"
    :ivar type: The app's type is Stripe. Required. STRIPE.
    :vartype type: Literal[AppType.STRIPE]
    :ivar secret_api_key: The Stripe API key.
    :vartype secret_api_key: str
    """

    name: Required[str]
    """Display name. Required."""
    description: str
    """Description."""
    metadata: Optional["Metadata"]
    """Metadata."""
    type: Required[Literal[AppType.STRIPE]]
    """The app's type is Stripe. Required. STRIPE."""
    secretAPIKey: str
    """The Stripe API key."""


class StripeCustomerAppData(TypedDict, total=False):
    """Stripe Customer App Data.

    :ivar id: App ID.
    :vartype id: str
    :ivar type: App Type. Required. STRIPE.
    :vartype type: Literal[AppType.STRIPE]
    :ivar stripe_customer_id: The Stripe customer ID. Required.
    :vartype stripe_customer_id: str
    :ivar stripe_default_payment_method_id: The Stripe default payment method ID.
    :vartype stripe_default_payment_method_id: str
    :ivar app: The installed stripe app this data belongs to.
    :vartype app: "StripeApp"
    """

    id: str
    """App ID."""
    type: Required[Literal[AppType.STRIPE]]
    """App Type. Required. STRIPE."""
    stripeCustomerId: Required[str]
    """The Stripe customer ID. Required."""
    stripeDefaultPaymentMethodId: str
    """The Stripe default payment method ID."""
    app: "StripeApp"
    """The installed stripe app this data belongs to."""


class StripeCustomerAppDataBase(TypedDict, total=False):
    """Stripe Customer App Data Base.

    :ivar stripe_customer_id: The Stripe customer ID. Required.
    :vartype stripe_customer_id: str
    :ivar stripe_default_payment_method_id: The Stripe default payment method ID.
    :vartype stripe_default_payment_method_id: str
    """

    stripeCustomerId: Required[str]
    """The Stripe customer ID. Required."""
    stripeDefaultPaymentMethodId: str
    """The Stripe default payment method ID."""


class StripeTaxConfig(TypedDict, total=False):
    """The tax config for Stripe.

    :ivar code: Tax code. Required.
    :vartype code: str
    """

    code: Required[str]
    """Tax code. Required."""


class StripeWebhookEvent(TypedDict, total=False):
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
    :vartype data: "StripeWebhookEventData"
    """

    id: Required[str]
    """The event ID. Required."""
    type: Required[str]
    """The event type. Required."""
    livemode: Required[bool]
    """Live mode. Required."""
    created: Required[int]
    """The event created timestamp. Required."""
    data: Required["StripeWebhookEventData"]
    """The event data. Required."""


class StripeWebhookEventData(TypedDict, total=False):
    """StripeWebhookEventData.

    :ivar object: Required.
    :vartype object: Any
    """

    object: Required[Any]
    """Required."""


class SubjectUpsert(TypedDict, total=False):
    """A subject is a unique identifier for a user or entity.

    ⚠️ **Deprecated**: Subjects as managable entities are being depracated, use customers with
    subject key usage attribution instead.

    :ivar key: A unique, human-readable identifier for the subject. This is typically a database ID
     or a customer key. Required.
    :vartype key: str
    :ivar display_name: A human-readable display name for the subject.
    :vartype display_name: str
    :ivar metadata: Metadata for the subject.
    :vartype metadata: dict[str, Any]
    :ivar current_period_start: The start of the current period for the subject.
    :vartype current_period_start: str
    :ivar current_period_end: The end of the current period for the subject.
    :vartype current_period_end: str
    :ivar stripe_customer_id: The Stripe customer ID for the subject.
    :vartype stripe_customer_id: str
    """

    key: Required[str]
    """A unique, human-readable identifier for the subject. This is typically a database ID or a
     customer key. Required."""
    displayName: Optional[str]
    """A human-readable display name for the subject."""
    metadata: Optional[dict[str, Any]]
    """Metadata for the subject."""
    currentPeriodStart: str
    """The start of the current period for the subject."""
    currentPeriodEnd: str
    """The end of the current period for the subject."""
    stripeCustomerId: Optional[str]
    """The Stripe customer ID for the subject."""


class SubscriptionAddonCreate(TypedDict, total=False):
    """A subscription add-on create body.

    :ivar name: Display name. Required.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: "Metadata"
    :ivar quantity: Quantity. Required.
    :vartype quantity: int
    :ivar timing: Timing. Required. Is either a Union[str, "_models.SubscriptionTimingEnum"] type
     or a datetime.datetime type.
    :vartype timing: "_unions.SubscriptionTiming"
    :ivar addon: Addon. Required.
    :vartype addon: "SubscriptionAddonCreateAddon"
    """

    name: Required[str]
    """Display name. Required."""
    description: str
    """Description."""
    metadata: Optional["Metadata"]
    """Metadata."""
    quantity: Required[int]
    """Quantity. Required."""
    timing: Required["_unions.SubscriptionTiming"]
    """Timing. Required. Is either a Union[str, \"_models.SubscriptionTimingEnum\"] type or a
     datetime.datetime type."""
    addon: Required["SubscriptionAddonCreateAddon"]
    """Addon. Required."""


class SubscriptionAddonCreateAddon(TypedDict, total=False):
    """SubscriptionAddonCreateAddon.

    :ivar id: The ID of the add-on. Required.
    :vartype id: str
    """

    id: Required[str]
    """The ID of the add-on. Required."""


class SubscriptionAddonUpdate(TypedDict, total=False):
    """Resource create or update operation model.

    :ivar name: Display name.
    :vartype name: str
    :ivar description: Description.
    :vartype description: str
    :ivar metadata: Metadata.
    :vartype metadata: "Metadata"
    :ivar quantity: Quantity.
    :vartype quantity: int
    :ivar timing: Timing. Is either a Union[str, "_models.SubscriptionTimingEnum"] type or a
     datetime.datetime type.
    :vartype timing: "_unions.SubscriptionTiming"
    """

    name: str
    """Display name."""
    description: str
    """Description."""
    metadata: Optional["Metadata"]
    """Metadata."""
    quantity: int
    """Quantity."""
    timing: "_unions.SubscriptionTiming"
    """Timing. Is either a Union[str, \"_models.SubscriptionTimingEnum\"] type or a datetime.datetime
     type."""


class SubscriptionEdit(TypedDict, total=False):
    """Subscription edit input.

    :ivar customizations: Batch processing commands for manipulating running subscriptions. The key
     format is ``/phases/{phaseKey}`` or ``/phases/{phaseKey}/items/{itemKey}``. Required.
    :vartype customizations: list["_unions.SubscriptionEditOperation"]
    :ivar timing: Whether the billing period should be restarted.Timing configuration to allow for
     the changes to take effect at different times. Is either a Union[str,
     "_models.SubscriptionTimingEnum"] type or a datetime.datetime type.
    :vartype timing: "_unions.SubscriptionTiming"
    """

    customizations: Required[list["_unions.SubscriptionEditOperation"]]
    """Batch processing commands for manipulating running subscriptions. The key format is
     ``/phases/{phaseKey}`` or ``/phases/{phaseKey}/items/{itemKey}``. Required."""
    timing: "_unions.SubscriptionTiming"
    """Whether the billing period should be restarted.Timing configuration to allow for the changes to
     take effect at different times. Is either a Union[str, \"_models.SubscriptionTimingEnum\"] type
     or a datetime.datetime type."""


class SubscriptionPhaseCreate(TypedDict, total=False):
    """Subscription phase create input.

    :ivar start_after: Start after. Required.
    :vartype start_after: str
    :ivar duration: Duration.
    :vartype duration: str
    :ivar discounts: Discounts.
    :vartype discounts: "Discounts"
    :ivar key: A locally unique identifier for the phase. Required.
    :vartype key: str
    :ivar name: The name of the phase. Required.
    :vartype name: str
    :ivar description: The description of the phase.
    :vartype description: str
    """

    startAfter: Required[Optional[str]]
    """Start after. Required."""
    duration: str
    """Duration."""
    discounts: "Discounts"
    """Discounts."""
    key: Required[str]
    """A locally unique identifier for the phase. Required."""
    name: Required[str]
    """The name of the phase. Required."""
    description: str
    """The description of the phase."""


class TaxConfig(TypedDict, total=False):
    """Set of provider specific tax configs.

    :ivar behavior: Tax behavior. Known values are: "inclusive" and "exclusive".
    :vartype behavior: Union[str, "TaxBehavior"]
    :ivar stripe: Stripe tax config.
    :vartype stripe: "StripeTaxConfig"
    :ivar custom_invoicing: Custom invoicing tax config.
    :vartype custom_invoicing: "CustomInvoicingTaxConfig"
    :ivar tax_code_id: Tax code ID.
    :vartype tax_code_id: str
    """

    behavior: Union[str, "TaxBehavior"]
    """Tax behavior. Known values are: \"inclusive\" and \"exclusive\"."""
    stripe: "StripeTaxConfig"
    """Stripe tax config."""
    customInvoicing: "CustomInvoicingTaxConfig"
    """Custom invoicing tax config."""
    taxCodeId: str
    """Tax code ID."""


class TieredPriceWithCommitments(TypedDict, total=False):
    """Tiered price with spend commitments.

    :ivar type: The type of the price.

     One of: flat, unit, or tiered. Required. TIERED.
    :vartype type: Literal[PriceType.TIERED]
    :ivar mode: Mode. Required. Known values are: "volume" and "graduated".
    :vartype mode: Union[str, "TieredPriceMode"]
    :ivar tiers: Tiers. Required.
    :vartype tiers: list["PriceTier"]
    :ivar minimum_amount: Minimum amount.
    :vartype minimum_amount: str
    :ivar maximum_amount: Maximum amount.
    :vartype maximum_amount: str
    """

    type: Required[Literal[PriceType.TIERED]]
    """The type of the price.
     
     One of: flat, unit, or tiered. Required. TIERED."""
    mode: Required[Union[str, "TieredPriceMode"]]
    """Mode. Required. Known values are: \"volume\" and \"graduated\"."""
    tiers: Required[list["PriceTier"]]
    """Tiers. Required."""
    minimumAmount: str
    """Minimum amount."""
    maximumAmount: str
    """Maximum amount."""


class UnitPrice(TypedDict, total=False):
    """Unit price.

    :ivar type: The type of the price. Required. UNIT.
    :vartype type: Literal[PriceType.UNIT]
    :ivar amount: The amount of the unit price. Required.
    :vartype amount: str
    """

    type: Required[Literal[PriceType.UNIT]]
    """The type of the price. Required. UNIT."""
    amount: Required[str]
    """The amount of the unit price. Required."""


class UnitPriceWithCommitments(TypedDict, total=False):
    """Unit price with spend commitments.

    :ivar type: The type of the price. Required. UNIT.
    :vartype type: Literal[PriceType.UNIT]
    :ivar amount: The amount of the unit price. Required.
    :vartype amount: str
    :ivar minimum_amount: Minimum amount.
    :vartype minimum_amount: str
    :ivar maximum_amount: Maximum amount.
    :vartype maximum_amount: str
    """

    type: Required[Literal[PriceType.UNIT]]
    """The type of the price. Required. UNIT."""
    amount: Required[str]
    """The amount of the unit price. Required."""
    minimumAmount: str
    """Minimum amount."""
    maximumAmount: str
    """Maximum amount."""


class VoidInvoiceAction(TypedDict, total=False):
    """InvoiceVoidAction describes how to handle the voided line items.

    :ivar percentage: How much of the total line items to be voided? (e.g. 100% means all charges
     are voided). Required.
    :vartype percentage: float
    :ivar action: The action to take on the line items. Required. Is either a
     VoidInvoiceLineDiscardAction type or a VoidInvoiceLinePendingAction type.
    :vartype action: "_unions.VoidInvoiceLineAction"
    """

    percentage: Required[float]
    """How much of the total line items to be voided? (e.g. 100% means all charges are voided).
     Required."""
    action: Required["_unions.VoidInvoiceLineAction"]
    """The action to take on the line items. Required. Is either a VoidInvoiceLineDiscardAction type
     or a VoidInvoiceLinePendingAction type."""


class VoidInvoiceActionInput(TypedDict, total=False):
    """Request to void an invoice.

    :ivar action: The action to take on the voided line items. Required.
    :vartype action: "VoidInvoiceAction"
    :ivar reason: The reason for voiding the invoice. Required.
    :vartype reason: str
    :ivar overrides: Per line item overrides for the action.

     If not specified, the ``action`` will be applied to all line items.
    :vartype overrides: list["VoidInvoiceActionLineOverride"]
    """

    action: Required["VoidInvoiceAction"]
    """The action to take on the voided line items. Required."""
    reason: Required[str]
    """The reason for voiding the invoice. Required."""
    overrides: Optional[list["VoidInvoiceActionLineOverride"]]
    """Per line item overrides for the action.
     
     If not specified, the ``action`` will be applied to all line items."""


class VoidInvoiceActionLineOverride(TypedDict, total=False):
    """VoidInvoiceLineOverride describes how to handle a specific line item in the invoice when
    voiding.

    :ivar line_id: The line item ID to override. Required.
    :vartype line_id: str
    :ivar action: The action to take on the line item. Required.
    :vartype action: "VoidInvoiceAction"
    """

    lineId: Required[str]
    """The line item ID to override. Required."""
    action: Required["VoidInvoiceAction"]
    """The action to take on the line item. Required."""


class VoidInvoiceLineDiscardAction(TypedDict, total=False):
    """VoidInvoiceLineDiscardAction describes how to handle the voidied line item in the invoice.

    :ivar type: The action to take on the line item. Required. The line items will never be charged
     for again.
    :vartype type: Literal[VoidInvoiceLineActionType.DISCARD]
    """

    type: Required[Literal[VoidInvoiceLineActionType.DISCARD]]
    """The action to take on the line item. Required. The line items will never be charged for again."""


class VoidInvoiceLinePendingAction(TypedDict, total=False):
    """VoidInvoiceLinePendingAction describes how to handle the voidied line item in the invoice.

    :ivar type: The action to take on the line item. Required. Queue the line items into the
     pending state, they will be included in the next invoice. (We want to generate an invoice right
     now).
    :vartype type: Literal[VoidInvoiceLineActionType.PENDING]
    :ivar next_invoice_at: The time at which the line item should be invoiced again.

     If not provided, the line item will be re-invoiced now.
    :vartype next_invoice_at: str
    """

    type: Required[Literal[VoidInvoiceLineActionType.PENDING]]
    """The action to take on the line item. Required. Queue the line items into the pending state,
     they will be included in the next invoice. (We want to generate an invoice right now)."""
    nextInvoiceAt: str
    """The time at which the line item should be invoiced again.
     
     If not provided, the line item will be re-invoiced now."""


class InvalidateRequest(TypedDict, total=False):
    """InvalidateRequest.

    :ivar id: Invalidate a portal token by ID.
    :vartype id: str
    :ivar subject: Invalidate all portal tokens for a subject.
    :vartype subject: str
    """

    id: str
    """Invalidate a portal token by ID."""
    subject: str
    """Invalidate all portal tokens for a subject."""
