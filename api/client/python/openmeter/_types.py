# coding=utf-8

import datetime
from typing import TYPE_CHECKING, Union

if TYPE_CHECKING:
    from . import models as _models
App = Union["_models.StripeApp", "_models.SandboxApp", "_models.CustomInvoicingApp"]
App = Union["_models.StripeApp", "_models.SandboxApp", "_models.CustomInvoicingApp"]
CustomerAppData = Union[
    "_models.StripeCustomerAppData",
    "_models.SandboxCustomerAppData",
    "_models.CustomInvoicingCustomerAppData",
]
RateCardEntitlement = Union[
    "_models.RateCardMeteredEntitlement",
    "_models.RateCardStaticEntitlement",
    "_models.RateCardBooleanEntitlement",
]
RateCardUsageBasedPrice = Union[
    "_models.FlatPriceWithPaymentTerm",
    "_models.UnitPriceWithCommitments",
    "_models.TieredPriceWithCommitments",
    "_models.DynamicPriceWithCommitments",
    "_models.PackagePriceWithCommitments",
]
RateCard = Union["_models.RateCardFlatFee", "_models.RateCardUsageBased"]
RecurringPeriodInterval = Union[str, str, "_models.RecurringPeriodIntervalEnum"]
Entitlement = Union[
    "_models.EntitlementMetered",
    "_models.EntitlementStatic",
    "_models.EntitlementBoolean",
]
SubscriptionTiming = Union[str, "_models.SubscriptionTimingEnum", datetime.datetime]
SubscriptionEditOperation = Union[
    "_models.EditSubscriptionAddItem",
    "_models.EditSubscriptionRemoveItem",
    "_models.EditSubscriptionAddPhase",
    "_models.EditSubscriptionRemovePhase",
    "_models.EditSubscriptionStretchPhase",
    "_models.EditSubscriptionUnscheduleEdit",
]
EntitlementV2 = Union[
    "_models.EntitlementMeteredV2",
    "_models.EntitlementStaticV2",
    "_models.EntitlementBooleanV2",
]
MeasureUsageFrom = Union[str, "_models.MeasureUsageFromPreset", datetime.datetime]
BillingWorkflowCollectionAlignment = (
    "_models.BillingWorkflowCollectionAlignmentSubscription"
)
BillingProfileAppsOrReference = Union[
    "_models.BillingProfileApps", "_models.BillingProfileAppReferences"
]
InvoiceDocumentRef = "_models.CreditNoteOriginalInvoiceRef"
BillingDiscountReason = Union[
    "_models.DiscountReasonMaximumSpend",
    "_models.DiscountReasonRatecardPercentage",
    "_models.DiscountReasonRatecardUsage",
]
PaymentTerms = Union["_models.PaymentTermInstant", "_models.PaymentTermDueDate"]
VoidInvoiceLineAction = Union[
    "_models.VoidInvoiceLineDiscardAction", "_models.VoidInvoiceLinePendingAction"
]
NotificationChannel = "_models.NotificationChannelWebhook"
NotificationRule = Union[
    "_models.NotificationRuleBalanceThreshold",
    "_models.NotificationRuleEntitlementReset",
    "_models.NotificationRuleInvoiceCreated",
    "_models.NotificationRuleInvoiceUpdated",
]
NotificationEventPayload = Union[
    "_models.NotificationEventResetPayload",
    "_models.NotificationEventBalanceThresholdPayload",
    "_models.NotificationEventInvoiceCreatedPayload",
    "_models.NotificationEventInvoiceUpdatedPayload",
]
AppReplaceUpdate = Union[
    "_models.StripeAppReplaceUpdate",
    "_models.SandboxAppReplaceUpdate",
    "_models.CustomInvoicingAppReplaceUpdate",
]
ULIDOrExternalKey = str
ListFeaturesResult = Union[list["_models.Feature"], "_models.FeaturePaginatedResponse"]
SubscriptionCreate = Union[
    "_models.PlanSubscriptionCreate", "_models.CustomSubscriptionCreate"
]
SubscriptionChange = Union[
    "_models.PlanSubscriptionChange", "_models.CustomSubscriptionChange"
]
EntitlementV2CreateInputs = Union[
    "_models.EntitlementMeteredV2CreateInputs",
    "_models.EntitlementStaticCreateInputs",
    "_models.EntitlementBooleanCreateInputs",
]
ListEntitlementsResult = Union[
    list["_types.Entitlement"], "_models.EntitlementPaginatedResponse"
]
EntitlementCreateInputs = Union[
    "_models.EntitlementMeteredCreateInputs",
    "_models.EntitlementStaticCreateInputs",
    "_models.EntitlementBooleanCreateInputs",
]
NotificationChannelCreateRequest = "_models.NotificationChannelWebhookCreateRequest"
NotificationRuleCreateRequest = Union[
    "_models.NotificationRuleBalanceThresholdCreateRequest",
    "_models.NotificationRuleEntitlementResetCreateRequest",
    "_models.NotificationRuleInvoiceCreatedCreateRequest",
    "_models.NotificationRuleInvoiceUpdatedCreateRequest",
]
IngestEventsBody = Union["_models.Event", list["_models.Event"]]
