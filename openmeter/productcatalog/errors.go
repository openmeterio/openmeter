package productcatalog

import "github.com/openmeterio/openmeter/pkg/models"

// PlanAddon errors

var (
	ErrPlanAddonIncompatibleStatus = models.NewValidationIssue(
		"plan_addon_incompatible_status",
		"plan status is incompatible with the addon status",
		models.WithPath("status"),
		models.WithCriticalSeverity(),
	)

	ErrPlanAddonMaxQuantityMustBeSet = models.NewValidationIssue(
		"plan_addon_max_quantity_must_be_set",
		"maximum quantity must be set to positive number for add-on with multiple instance type",
		models.WithPath("maxQuantity"),
		models.WithCriticalSeverity(),
	)

	ErrPlanAddonMaxQuantityMustNotBeSet = models.NewValidationIssue(
		"plan_addon_max_quantity_must_not_be_set",
		"maximum quantity must not be set for add-on with single instance type",
		models.WithPath("maxQuantity"),
		models.WithCriticalSeverity(),
	)

	ErrPlanAddonCurrencyMismatch = models.NewValidationIssue(
		"plan_addon_currency_mismatch",
		"currency of the plan and addon must match",
		models.WithPath("currency"),
		models.WithCriticalSeverity(),
	)

	ErrPlanAddonUnknownPlanPhaseKey = models.NewValidationIssue(
		"plan_addon_unknown_plan_phase_key",
		"add-on must define valid/existing plan phase key from which the add-on is available for purchase",
		models.WithPath("fromPlanPhase"),
		models.WithCriticalSeverity(),
	)
)

// RateCard errors

var (
	ErrRateCardKeyMismatch = models.NewValidationIssue(
		"rate_card_key_mismatch",
		"key must match",
		models.WithPath("key"),
		models.WithCriticalSeverity(),
	)

	ErrRateCardPriceTypeMismatch = models.NewValidationIssue(
		"rate_card_price_type_mismatch",
		"price type must match",
		models.WithPath("price", "type"),
		models.WithCriticalSeverity(),
	)

	ErrRateCardPricePaymentTermMismatch = models.NewValidationIssue(
		"rate_card_price_payment_term_mismatch",
		"price payment term must match",
		models.WithPath("price", "paymentTerm"),
		models.WithCriticalSeverity(),
	)

	ErrRateCardOnlyFlatPriceAllowed = models.NewValidationIssue(
		"rate_card_only_flat_price_allowed",
		"only flat price is allowed",
		models.WithPath("price", "type"),
		models.WithCriticalSeverity(),
	)

	ErrRateCardFeatureIDMismatch = models.NewValidationIssue(
		"rate_card_feature_id_mismatch",
		"feature identifiers id must match",
		models.WithPath("featureId"),
		models.WithCriticalSeverity(),
	)

	ErrRateCardFeatureKeyMismatch = models.NewValidationIssue(
		"rate_card_feature_key_mismatch",
		"feature key must match",
		models.WithPath("featureKey"),
		models.WithCriticalSeverity(),
	)

	ErrRateCardBillingCadenceMismatch = models.NewValidationIssue(
		"rate_card_billing_cadence_mismatch",
		"billing cadence must match",
		models.WithPath("billingCadence"),
		models.WithCriticalSeverity(),
	)

	ErrRateCardEntitlementTemplateTypeMismatch = models.NewValidationIssue(
		"rate_card_entitlement_template_type_mismatch",
		"entitlement template type must match",
		models.WithPath("entitlementTemplate", "type"),
		models.WithCriticalSeverity(),
	)

	ErrRateCardStaticEntitlementTemplateNotAllowed = models.NewValidationIssue(
		"rate_card_static_entitlement_template_not_allowed",
		"static entitlement template is not allowed",
		models.WithPath("entitlementTemplate", "type"),
	)

	ErrRateCardMeteredEntitlementTemplateUsagePeriodMismatch = models.NewValidationIssue(
		"rate_card_metered_entitlement_template_usage_period_mismatch",
		"usage period for metered entitlement template must match",
		models.WithPath("entitlementTemplate", "usagePeriod"),
	)

	ErrRateCardPercentageDiscountNotAllowed = models.NewValidationIssue(
		"rate_card_percentage_discount_not_allowed",
		"percentage discount is not allowed",
		models.WithPath("discounts", "percentage"),
	)
)
