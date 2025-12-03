package productcatalog

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/models"
)

// PlanAddon errors

const ErrCodePlanAddonIncompatibleStatus models.ErrorCode = "plan_addon_incompatible_status"

var ErrPlanAddonIncompatibleStatus = models.NewValidationIssue(
	ErrCodePlanAddonIncompatibleStatus,
	"plan status is incompatible with the addon status",
	models.WithFieldString("status"),
	models.WithWarningSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodePlanAddonMaxQuantityMustBeSet models.ErrorCode = "plan_addon_max_quantity_must_be_set"

var ErrPlanAddonMaxQuantityMustBeSet = models.NewValidationIssue(
	ErrCodePlanAddonMaxQuantityMustBeSet,
	"maximum quantity must be set to positive number for add-on with multiple instance type",
	models.WithFieldString("maxQuantity"),
	models.WithWarningSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodePlanAddonMaxQuantityMustNotBeSet models.ErrorCode = "plan_addon_max_quantity_must_not_be_set"

var ErrPlanAddonMaxQuantityMustNotBeSet = models.NewValidationIssue(
	ErrCodePlanAddonMaxQuantityMustNotBeSet,
	"maximum quantity must not be set for add-on with single instance type",
	models.WithFieldString("maxQuantity"),
	models.WithWarningSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodePlanAddonCurrencyMismatch models.ErrorCode = "plan_addon_currency_mismatch"

var ErrPlanAddonCurrencyMismatch = models.NewValidationIssue(
	ErrCodePlanAddonCurrencyMismatch,
	"currency of the plan and addon must match",
	models.WithFieldString("currency"),
	models.WithWarningSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodePlanAddonUnknownPlanPhaseKey models.ErrorCode = "plan_addon_unknown_plan_phase_key"

var ErrPlanAddonUnknownPlanPhaseKey = models.NewValidationIssue(
	ErrCodePlanAddonUnknownPlanPhaseKey,
	"add-on must define valid/existing plan phase key from which the add-on is available for purchase",
	models.WithFieldString("fromPlanPhase"),
	models.WithWarningSeverity(),
)

// RateCard errors

const ErrCodeRateCardKeyMismatch models.ErrorCode = "rate_card_key_mismatch"

var ErrRateCardKeyMismatch = models.NewValidationIssue(
	ErrCodeRateCardKeyMismatch,
	"key must match",
	models.WithFieldString("key"),
	models.WithWarningSeverity(),
)

const ErrCodeRateCardPriceTypeMismatch models.ErrorCode = "rate_card_price_type_mismatch"

var ErrRateCardPriceTypeMismatch = models.NewValidationIssue(
	ErrCodeRateCardPriceTypeMismatch,
	"price type must match",
	models.WithFieldString("price"),
	models.WithWarningSeverity(),
)

const ErrCodeRateCardPricePaymentTermMismatch models.ErrorCode = "rate_card_price_payment_term_mismatch"

var ErrRateCardPricePaymentTermMismatch = models.NewValidationIssue(
	ErrCodeRateCardPricePaymentTermMismatch,
	"price payment term must match",
	models.WithFieldString("price"),
	models.WithWarningSeverity(),
)

const ErrCodeRateCardOnlyFlatPriceAllowed models.ErrorCode = "rate_card_only_flat_price_allowed"

var ErrRateCardOnlyFlatPriceAllowed = models.NewValidationIssue(
	ErrCodeRateCardOnlyFlatPriceAllowed,
	"only flat price is allowed",
	models.WithFieldString("price"),
	models.WithWarningSeverity(),
)

const ErrCodeRateCardFeatureNotFound models.ErrorCode = "rate_card_feature_not_found"

var ErrRateCardFeatureNotFound = models.NewValidationIssue(
	ErrCodeRateCardFeatureNotFound,
	"feature not found",
	models.WithFieldString("featureKey"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeRateCardFeatureArchived models.ErrorCode = "rate_card_feature_archived"

var ErrRateCardFeatureArchived = models.NewValidationIssue(
	ErrCodeRateCardFeatureArchived,
	"feature archived",
	models.WithFieldString("featureKey"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeRateCardFeatureMismatch models.ErrorCode = "rate_card_feature_mismatch"

var ErrRateCardFeatureMismatch = models.NewValidationIssue(
	ErrCodeRateCardFeatureMismatch,
	"feature id and key must reference the same feature",
	models.WithFieldString("featureKey"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeRateCardFeatureIDMismatch models.ErrorCode = "rate_card_feature_id_mismatch"

var ErrRateCardFeatureIDMismatch = models.NewValidationIssue(
	ErrCodeRateCardFeatureIDMismatch,
	"feature identifiers id must match",
	models.WithFieldString("featureId"),
	models.WithWarningSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeRateCardFeatureKeyMismatch models.ErrorCode = "rate_card_feature_key_mismatch"

var ErrRateCardFeatureKeyMismatch = models.NewValidationIssue(
	ErrCodeRateCardFeatureKeyMismatch,
	"feature key must match",
	models.WithFieldString("featureKey"),
	models.WithWarningSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeRateCardBillingCadenceMismatch models.ErrorCode = "rate_card_billing_cadence_mismatch"

var ErrRateCardBillingCadenceMismatch = models.NewValidationIssue(
	ErrCodeRateCardBillingCadenceMismatch,
	"billing cadence must match",
	models.WithFieldString("billingCadence"),
	models.WithWarningSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeRateCardEntitlementTemplateTypeMismatch models.ErrorCode = "rate_card_entitlement_template_type_mismatch"

var ErrRateCardEntitlementTemplateTypeMismatch = models.NewValidationIssue(
	ErrCodeRateCardEntitlementTemplateTypeMismatch,
	"entitlement template type must match",
	models.WithFieldString("type"),
	models.WithWarningSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeRateCardStaticEntitlementTemplateNotAllowed models.ErrorCode = "rate_card_static_entitlement_template_not_allowed"

var ErrRateCardStaticEntitlementTemplateNotAllowed = models.NewValidationIssue(
	ErrCodeRateCardStaticEntitlementTemplateNotAllowed,
	"static entitlement template is not allowed",
	models.WithFieldString("type"),
	models.WithWarningSeverity(),
)

const ErrCodeRateCardMeteredEntitlementTemplateUsagePeriodMismatch models.ErrorCode = "rate_card_metered_entitlement_template_usage_period_mismatch"

var ErrRateCardMeteredEntitlementTemplateUsagePeriodMismatch = models.NewValidationIssue(
	ErrCodeRateCardMeteredEntitlementTemplateUsagePeriodMismatch,
	"usage period for metered entitlement template must match",
	models.WithFieldString("usagePeriod"),
	models.WithWarningSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeRateCardPercentageDiscountNotAllowed models.ErrorCode = "rate_card_percentage_discount_not_allowed"

var ErrRateCardPercentageDiscountNotAllowed = models.NewValidationIssue(
	ErrCodeRateCardPercentageDiscountNotAllowed,
	"percentage discount is not allowed",
	models.WithFieldString("percentage"),
	models.WithWarningSeverity(),
)

const ErrCodeRateCardDuplicatedKey models.ErrorCode = "rate_card_duplicated_key"

var ErrRateCardDuplicatedKey = models.NewValidationIssue(
	ErrCodeRateCardDuplicatedKey,
	"duplicated key",
	models.WithFieldString("key"),
	models.WithCriticalSeverity(),
)

const ErrCodeRateCardEntitlementTemplateWithNoFeature models.ErrorCode = "entitlement_template_with_no_feature"

var ErrRateCardEntitlementTemplateWithNoFeature = models.NewValidationIssue(
	ErrCodeRateCardEntitlementTemplateWithNoFeature,
	"entitlement template requires feature to be associated with",
	models.WithFieldString("featureKey"),
	models.WithWarningSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeEffectivePeriodFromAfterTo models.ErrorCode = "effective_period_from_after_to"

var ErrEffectivePeriodFromAfterTo = models.NewValidationIssue(
	ErrCodeEffectivePeriodFromAfterTo,
	"effectiveFrom is after effectiveTo",
	models.WithFieldString("effectiveFrom"),
	models.WithWarningSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeEffectivePeriodFromNotSet models.ErrorCode = "effective_period_from_not_set"

var ErrEffectivePeriodFromNotSet = models.NewValidationIssue(
	ErrCodeEffectivePeriodFromNotSet,
	"effectiveFrom is must be provided if effectiveTo is set",
	models.WithFieldString("effectiveFrom"),
	models.WithWarningSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeCurrencyInvalid models.ErrorCode = "currency_invalid"

var ErrCurrencyInvalid = models.NewValidationIssue(
	ErrCodeCurrencyInvalid,
	"currency is invalid",
	models.WithFieldString("currency"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeEntitlementTemplateInvalidIssueAfterResetWithPriority models.ErrorCode = "entitlement_template_invalid_issue_after_reset_with_priority"

var ErrEntitlementTemplateInvalidIssueAfterResetWithPriority = models.NewValidationIssue(
	ErrCodeEntitlementTemplateInvalidIssueAfterResetWithPriority,
	"invalid entitlement template as issue after reset is required if issue after reset priority is set",
	models.WithFieldString("issueAfterReset"),
	models.WithWarningSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeEntitlementTemplateNegativeUsagePeriod models.ErrorCode = "entitlement_template_negative_usage_period"

var ErrEntitlementTemplateNegativeUsagePeriod = models.NewValidationIssue(
	ErrCodeEntitlementTemplateNegativeUsagePeriod,
	"usage period must be positive",
	models.WithFieldString("usagePeriod"),
	models.WithWarningSeverity(),
)

const ErrCodeEntitlementTemplateUsagePeriodLessThenAnHour models.ErrorCode = "entitlement_template_usage_period_less_then_an_hour"

var ErrEntitlementTemplateUsagePeriodLessThenAnHour = models.NewValidationIssue(
	ErrCodeEntitlementTemplateUsagePeriodLessThenAnHour,
	"usage period must be at least 1 hour",
	models.WithFieldString("usagePeriod"),
	models.WithWarningSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeEntitlementTemplateInvalidJSONConfig models.ErrorCode = "entitlement_template_invalid_json_config"

var ErrEntitlementTemplateInvalidJSONConfig = models.NewValidationIssue(
	ErrCodeEntitlementTemplateInvalidJSONConfig,
	"invalid JSON in static entitlement config",
	models.WithFieldString("entitlementTemplate", "config"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeRateCardKeyFeatureKeyMismatch models.ErrorCode = "rate_card_key_feature_key_mismatch"

var ErrRateCardKeyFeatureKeyMismatch = models.NewValidationIssue(
	ErrCodeRateCardKeyFeatureKeyMismatch,
	"rate card key must match feature key",
	models.WithFieldString("key"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodePercentageDiscountInvalidValue models.ErrorCode = "percentage_discount_invalid_value"

var ErrPercentageDiscountInvalidValue = models.NewValidationIssue(
	ErrCodePercentageDiscountInvalidValue,
	"percentage must be between 0 and 100",
	models.WithFieldString("percentage"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeUsageDiscountNegativeQuantity models.ErrorCode = "usage_discount_negative_quantity"

var ErrUsageDiscountNegativeQuantity = models.NewValidationIssue(
	ErrCodeUsageDiscountNegativeQuantity,
	"usage must be greater than 0",
	models.WithFieldString("quantity"),
	models.WithWarningSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeUsageDiscountWithFlatPrice models.ErrorCode = "usage_discount_with_flat_price"

var ErrUsageDiscountWithFlatPrice = models.NewValidationIssue(
	ErrCodeUsageDiscountWithFlatPrice,
	"usage discount is not supported for flat price",
	models.WithWarningSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeBillingCadenceInvalidValue models.ErrorCode = "billing_cadence_invalid_value"

var ErrBillingCadenceInvalidValue = models.NewValidationIssue(
	ErrCodeBillingCadenceInvalidValue,
	"billing cadence must be positive and 1 hour long duration at least",
	models.WithFieldString("billingCadence"),
	models.WithWarningSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeRateCardMultipleBillingCadence models.ErrorCode = "rate_card_multiple_billing_cadence"

var ErrRateCardMultipleBillingCadence = models.NewValidationIssue(
	ErrCodeRateCardMultipleBillingCadence,
	"ratecards with prices must have the exact same billing cadence",
	models.WithFieldString("billingCadence"),
	models.WithWarningSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeRateCardBillingCadenceUnaligned models.ErrorCode = "rate_card_billing_cadence_unaligned"

var ErrRateCardBillingCadenceUnaligned = models.NewValidationIssue(
	ErrCodeRateCardBillingCadenceUnaligned,
	"ratecards with prices must have compatible billing cadence",
	models.WithFieldString("billingCadence"),
	models.WithWarningSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

// Addon errors

const ErrCodeAddonKeyEmpty models.ErrorCode = "addon_key_empty"

var ErrAddonKeyEmpty = models.NewValidationIssue(
	ErrCodeAddonKeyEmpty,
	"key must not be empty",
	models.WithFieldString("key"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeAddonNameEmpty models.ErrorCode = "addon_name_empty"

var ErrAddonNameEmpty = models.NewValidationIssue(
	ErrCodeAddonNameEmpty,
	"name must not be empty",
	models.WithFieldString("name"),
	models.WithWarningSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeAddonInvalidInstanceType models.ErrorCode = "addon_invalid_instance_type"

var ErrAddonInvalidInstanceType = models.NewValidationIssue(
	ErrCodeAddonInvalidInstanceType,
	"invalid instance type",
	models.WithFieldString("instanceType"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeAddonInvalidStatus models.ErrorCode = "addon_invalid_status"

var ErrAddonInvalidStatus = models.NewValidationIssue(
	ErrCodeAddonInvalidStatus,
	"invalid status",
	models.WithFieldString("status"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeAddonInvalidStatusForPublish models.ErrorCode = "addon_invalid_status_for_publish"

var ErrAddonInvalidStatusForPublish = models.NewValidationIssue(
	ErrCodeAddonInvalidStatusForPublish,
	"only draft add-ons can be published",
	models.WithFieldString("status"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeAddonInvalidPriceForMultiInstance models.ErrorCode = "addon_invalid_ratecard_price_for_multi_instance"

var ErrAddonInvalidPriceForMultiInstance = models.NewValidationIssue(
	ErrCodeAddonInvalidPriceForMultiInstance,
	"only free or flat price ratecards are allowed for add-on with multiple instance type",
	models.WithFieldString("price"),
	models.WithWarningSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeAddonHasNoRateCards models.ErrorCode = "addon_has_no_rate_cards"

var ErrAddonHasNoRateCards = models.NewValidationIssue(
	ErrCodeAddonHasNoRateCards,
	"add-on must have at least one rate card",
	models.WithFieldString("rateCards"),
	models.WithWarningSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

// Generic errors

const ErrCodeResourceKeyEmpty models.ErrorCode = "resource_key_empty"

var ErrResourceKeyEmpty = models.NewValidationIssue(
	ErrCodeResourceKeyEmpty,
	"key must not be empty",
	models.WithFieldString("key"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeResourceNameEmpty models.ErrorCode = "resource_name_empty"

var ErrResourceNameEmpty = models.NewValidationIssue(
	ErrCodeResourceNameEmpty,
	"name must not be empty",
	models.WithFieldString("name"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeNamespaceEmpty models.ErrorCode = "resource_namespace_empty"

var ErrNamespaceEmpty = models.NewValidationIssue(
	ErrCodeNamespaceEmpty,
	"namespace must not be empty",
	models.WithFieldString("namespace"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeIDEmpty models.ErrorCode = "resource_id_empty"

var ErrIDEmpty = models.NewValidationIssue(
	ErrCodeIDEmpty,
	"id must not be empty",
	models.WithFieldString("id"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

// Alignment errors

const ErrCodeDeprecatedUnalignedSubscription models.ErrorCode = "deprecated_unaligned_subscription"

var ErrDeprecatedUnalignedSubscription = models.NewValidationIssue(
	ErrCodeDeprecatedUnalignedSubscription,
	"unaligned subscriptions are being deprecated",
	models.WithWarningSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

// Plan errors

var ErrPlanBillingCadenceAllowedValues = []datetime.ISODurationString{
	"P1W",
	"P2W",
	"P4W",
	"P1M",
	"P3M",
	"P6M",
	"P12M",
	"P1Y",
}

const ErrCodePlanBillingCadenceInvalid models.ErrorCode = "plan_billing_cadence_invalid"

var ErrPlanBillingCadenceInvalid = models.NewValidationIssue(
	ErrCodePlanBillingCadenceInvalid,
	fmt.Sprintf("billing cadence must be one of the following: %s", strings.Join(lo.Map(ErrPlanBillingCadenceAllowedValues, func(v datetime.ISODurationString, _ int) string { return v.String() }), ", ")),
	models.WithFieldString("billingCadence"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodePlanPhaseWithNegativeDuration models.ErrorCode = "plan_phase_with_negative_duration"

var ErrPlanPhaseWithNegativeDuration = models.NewValidationIssue(
	ErrCodePlanPhaseWithNegativeDuration,
	"duration must be positive",
	models.WithFieldString("duration"),
	models.WithWarningSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodePlanPhaseDurationLessThenAnHour models.ErrorCode = "plan_phase_duration_less_then_an_hour"

var ErrPlanPhaseDurationLessThenAnHour = models.NewValidationIssue(
	ErrCodePlanPhaseDurationLessThenAnHour,
	"duration must be at least 1 hour",
	models.WithFieldString("duration"),
	models.WithWarningSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodePlanPhaseDuplicatedKey models.ErrorCode = "plan_phase_duplicated_key"

var ErrPlanPhaseDuplicatedKey = models.NewValidationIssue(
	ErrCodePlanPhaseDuplicatedKey,
	"duplicated key",
	models.WithFieldString("key"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodePlanInvalidStatus models.ErrorCode = "plan_invalid_status"

var ErrPlanInvalidStatus = models.NewValidationIssue(
	ErrCodePlanInvalidStatus,
	"invalid status",
	models.WithFieldString("status"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodePlanWithNoPhases models.ErrorCode = "plan_with_no_phases"

var ErrPlanWithNoPhases = models.NewValidationIssue(
	ErrCodePlanWithNoPhases,
	"plan must have at least one phase",
	models.WithFieldString("phases"),
	models.WithWarningSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodePlanHasNonLastPhaseWithNoDuration models.ErrorCode = "plan_has_non_last_phase_with_no_duration"

var ErrPlanHasNonLastPhaseWithNoDuration = models.NewValidationIssue(
	ErrCodePlanHasNonLastPhaseWithNoDuration,
	"duration must be set for plan phase if it is not the last one",
	models.WithFieldString("duration"),
	models.WithWarningSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodePlanHasLastPhaseWithDuration models.ErrorCode = "plan_has_last_phase_with_duration"

var ErrPlanHasLastPhaseWithDuration = models.NewValidationIssue(
	ErrCodePlanHasLastPhaseWithDuration,
	"duration must not be set for the last plan phase",
	models.WithFieldString("duration"),
	models.WithWarningSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodePlanPhaseHasNoRateCards models.ErrorCode = "plan_phase_has_no_rate_cards"

var ErrPlanPhaseHasNoRateCards = models.NewValidationIssue(
	ErrCodePlanPhaseHasNoRateCards,
	"plan phase must have at least one rate card",
	models.WithFieldString("rateCards"),
	models.WithWarningSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodePlanHasIncompatibleAddon models.ErrorCode = "plan_has_incompatible_addon"

var ErrPlanHasIncompatibleAddon = models.NewValidationIssue(
	ErrCodePlanHasIncompatibleAddon,
	"plan has incompatible add-on assignment",
	models.WithWarningSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodePlanBillingCadenceNotCompatible models.ErrorCode = "plan_billing_cadence_not_compatible"

var ErrPlanBillingCadenceNotCompatible = models.NewValidationIssue(
	ErrCodePlanBillingCadenceNotCompatible,
	"plan billing cadence is not compatible with rate card billing cadence",
	models.WithFieldString("billingCadence"),
	models.WithWarningSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)
