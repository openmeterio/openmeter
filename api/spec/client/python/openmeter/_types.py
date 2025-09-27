# coding=utf-8

import datetime
from typing import TYPE_CHECKING, Union

if TYPE_CHECKING:
    from . import models as _models
App = Union["_models.StripeApp", "_models.SandboxApp", "_models.CustomInvoicingApp"]
NotificationChannel = "_models.NotificationChannelWebhook"
NotificationRule = Union[
    "_models.NotificationRuleBalanceThreshold",
    "_models.NotificationRuleEntitlementReset",
    "_models.NotificationRuleInvoiceCreated",
    "_models.NotificationRuleInvoiceUpdated",
]
RecurringPeriodInterval = Union[str, str, "_models.RecurringPeriodIntervalEnum"]
InvoiceDocumentRef = "_models.CreditNoteOriginalInvoiceRef"
BillingProfileAppsOrReference = Union["_models.BillingProfileApps", "_models.BillingProfileAppReferences"]
BillingWorkflowCollectionAlignment = "_models.BillingWorkflowCollectionAlignmentSubscription"
BillingDiscountReason = Union[
    "_models.DiscountReasonMaximumSpend",
    "_models.DiscountReasonRatecardPercentage",
    "_models.DiscountReasonRatecardUsage",
]
RateCardUsageBasedPrice = Union[
    "_models.FlatPriceWithPaymentTerm",
    "_models.UnitPriceWithCommitments",
    "_models.TieredPriceWithCommitments",
    "_models.DynamicPriceWithCommitments",
    "_models.PackagePriceWithCommitments",
]
PaymentTerms = Union["_models.PaymentTermInstant", "_models.PaymentTermDueDate"]
NotificationEventPayload = Union[
    "_models.NotificationEventResetPayload",
    "_models.NotificationEventBalanceThresholdPayload",
    "_models.NotificationEventInvoiceCreatedPayload",
    "_models.NotificationEventInvoiceUpdatedPayload",
]
MeasureUsageFrom = Union[str, "_models.MeasureUsageFromPreset", datetime.datetime]
EntitlementV2 = Union["_models.EntitlementMeteredV2", "_models.EntitlementStaticV2", "_models.EntitlementBooleanV2"]
Entitlement = Union["_models.EntitlementMetered", "_models.EntitlementStatic", "_models.EntitlementBoolean"]
CustomerAppData = Union[
    "_models.StripeCustomerAppData", "_models.SandboxCustomerAppData", "_models.CustomInvoicingCustomerAppData"
]
VoidInvoiceLineAction = Union["_models.VoidInvoiceLineDiscardAction", "_models.VoidInvoiceLinePendingAction"]
RateCardEntitlement = Union[
    "_models.RateCardMeteredEntitlement", "_models.RateCardStaticEntitlement", "_models.RateCardBooleanEntitlement"
]
RateCard = Union["_models.RateCardFlatFee", "_models.RateCardUsageBased"]
SubscriptionTiming = Union[str, "_models.SubscriptionTimingEnum", datetime.datetime]
SubscriptionEditOperation = Union[
    "_models.EditSubscriptionAddItem",
    "_models.EditSubscriptionRemoveItem",
    "_models.EditSubscriptionAddPhase",
    "_models.EditSubscriptionRemovePhase",
    "_models.EditSubscriptionStretchPhase",
    "_models.EditSubscriptionUnscheduleEdit",
]
AppReplaceUpdate = Union[
    "_models.StripeAppReplaceUpdate", "_models.SandboxAppReplaceUpdate", "_models.CustomInvoicingAppReplaceUpdate"
]
NotificationChannelCreateRequest = "_models.NotificationChannelWebhookCreateRequest"
NotificationRuleCreateRequest = Union[
    "_models.NotificationRuleBalanceThresholdCreateRequest",
    "_models.NotificationRuleEntitlementResetCreateRequest",
    "_models.NotificationRuleInvoiceCreatedCreateRequest",
    "_models.NotificationRuleInvoiceUpdatedCreateRequest",
]
ULIDOrExternalKey = str
EntitlementCreateInputs = Union[
    "_models.EntitlementMeteredCreateInputs",
    "_models.EntitlementStaticCreateInputs",
    "_models.EntitlementBooleanCreateInputs",
]
ListEntitlementsResult = Union[list["_types.Entitlement"], "_models.EntitlementPaginatedResponse"]
ListFeaturesResult = Union[list["_models.Feature"], "_models.FeaturePaginatedResponse"]
SubscriptionCreate = Union["_models.PlanSubscriptionCreate", "_models.CustomSubscriptionCreate"]
SubscriptionChange = Union["_models.PlanSubscriptionChange", "_models.CustomSubscriptionChange"]
IngestEventsBody = Union["_models.Event", list["_models.Event"]]
