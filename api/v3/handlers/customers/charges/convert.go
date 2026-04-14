package charges

import (
	"fmt"
	"strings"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/labels"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingcharges "github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// ConvertMetadataToLabels converts domain metadata to API labels.
var ConvertMetadataToLabels = labels.FromMetadata[models.Metadata]

// convertFlatFeeChargeToAPI maps a flatfee.Charge to the API representation.
func convertFlatFeeChargeToAPI(source flatfee.Charge) api.BillingFlatFeeCharge {
	var out api.BillingFlatFeeCharge
	out.AmountAfterProration = ConvertDecimalToCurrencyAmount(source.ChargeBase.State.AmountAfterProration)
	out.BillingPeriod = ConvertClosedPeriodToAPI(source.ChargeBase.Intent.Intent.BillingPeriod)
	out.CreatedAt = TimePtrFromTime(source.ChargeBase.ManagedResource.ManagedModel.CreatedAt)
	out.Currency = ConvertCurrencyCodeToAPI(source.ChargeBase.Intent.Intent.Currency)
	out.Customer = ConvertCustomerIDToReference(source.ChargeBase.Intent.Intent.CustomerID)
	out.DeletedAt = source.ChargeBase.ManagedResource.ManagedModel.DeletedAt
	out.Description = source.ChargeBase.Intent.Intent.Description
	out.FeatureKey = lo.ToPtr(source.ChargeBase.Intent.FeatureKey)
	out.FullServicePeriod = ConvertClosedPeriodToAPI(source.ChargeBase.Intent.Intent.FullServicePeriod)
	out.Id = source.ChargeBase.ManagedResource.ID
	out.InvoiceAt = source.ChargeBase.Intent.InvoiceAt
	out.Labels = ConvertMetadataToLabels(source.ChargeBase.Intent.Intent.Metadata)
	out.ManagedBy = ConvertManagedByToAPI(source.ChargeBase.Intent.Intent.ManagedBy)
	out.Name = source.ChargeBase.Intent.Intent.Name
	out.PaymentTerm = ConvertPaymentTermToAPI(source.ChargeBase.Intent.PaymentTerm)
	out.Price = ConvertDecimalToCurrencyAmount(source.ChargeBase.Intent.AmountBeforeProration)
	out.ProrationConfiguration = ConvertProRatingConfigToAPI(source.ChargeBase.Intent.ProRating)
	out.ServicePeriod = ConvertClosedPeriodToAPI(source.ChargeBase.Intent.Intent.ServicePeriod)
	out.SettlementMode = ConvertSettlementModeToAPI(source.ChargeBase.Intent.SettlementMode)
	out.Subscription = subscriptionRefPtrToAPI(source.ChargeBase.Intent.Intent.Subscription)
	out.UniqueReferenceId = source.ChargeBase.Intent.Intent.UniqueReferenceID
	out.UpdatedAt = TimePtrFromTime(source.ChargeBase.ManagedResource.ManagedModel.UpdatedAt)
	return out
}

// convertUsageBasedChargeToAPI maps a usagebased.Charge to the API representation.
func convertUsageBasedChargeToAPI(source usagebased.Charge) api.BillingUsageBasedCharge {
	var out api.BillingUsageBasedCharge
	out.BillingPeriod = ConvertClosedPeriodToAPI(source.ChargeBase.Intent.Intent.BillingPeriod)
	out.CreatedAt = TimePtrFromTime(source.ChargeBase.ManagedResource.ManagedModel.CreatedAt)
	out.Currency = ConvertCurrencyCodeToAPI(source.ChargeBase.Intent.Intent.Currency)
	out.Customer = ConvertCustomerIDToReference(source.ChargeBase.Intent.Intent.CustomerID)
	out.DeletedAt = source.ChargeBase.ManagedResource.ManagedModel.DeletedAt
	out.Description = source.ChargeBase.Intent.Intent.Description
	out.FeatureKey = source.ChargeBase.Intent.FeatureKey
	out.FullServicePeriod = ConvertClosedPeriodToAPI(source.ChargeBase.Intent.Intent.FullServicePeriod)
	out.Id = source.ChargeBase.ManagedResource.ID
	out.InvoiceAt = source.ChargeBase.Intent.InvoiceAt
	out.Labels = ConvertMetadataToLabels(source.ChargeBase.Intent.Intent.Metadata)
	out.ManagedBy = ConvertManagedByToAPI(source.ChargeBase.Intent.Intent.ManagedBy)
	out.Name = source.ChargeBase.Intent.Intent.Name
	out.ServicePeriod = ConvertClosedPeriodToAPI(source.ChargeBase.Intent.Intent.ServicePeriod)
	out.SettlementMode = ConvertSettlementModeToAPI(source.ChargeBase.Intent.SettlementMode)
	out.Status = ConvertUsageBasedStatusToAPI(source.ChargeBase.Status)
	out.Subscription = subscriptionRefPtrToAPI(source.ChargeBase.Intent.Intent.Subscription)
	out.UniqueReferenceId = source.ChargeBase.Intent.Intent.UniqueReferenceID
	out.UpdatedAt = TimePtrFromTime(source.ChargeBase.ManagedResource.ManagedModel.UpdatedAt)
	return out
}

// subscriptionRefPtrToAPI converts a nullable SubscriptionReference pointer to the API type.
func subscriptionRefPtrToAPI(source *meta.SubscriptionReference) *api.BillingSubscriptionReference {
	if source == nil {
		return nil
	}
	ref := ConvertSubscriptionRefToAPI(*source)
	return &ref
}

// convertChargeToAPI dispatches on charge type and maps to the API union type.
func convertChargeToAPI(charge billingcharges.Charge) (api.BillingCharge, error) {
	var out api.BillingCharge

	switch charge.Type() {
	case meta.ChargeTypeFlatFee:
		ff, err := charge.AsFlatFeeCharge()
		if err != nil {
			return out, fmt.Errorf("converting flat fee charge: %w", err)
		}
		apiFF := convertFlatFeeChargeToAPI(ff)
		apiFF.Type = api.BillingFlatFeeChargeTypeFlatFee
		apiFF.Status = ConvertChargeStatusToAPI(meta.ChargeStatus(ff.Status))
		apiFF.AdvanceAfter = TimePtrFromOptional(ff.State.AdvanceAfter)
		apiFF.Discounts = convertFlatFeeDiscounts(ff.Intent.PercentageDiscounts)
		if err := out.FromBillingFlatFeeCharge(apiFF); err != nil {
			return out, fmt.Errorf("setting flat fee charge union: %w", err)
		}

	case meta.ChargeTypeUsageBased:
		ub, err := charge.AsUsageBasedCharge()
		if err != nil {
			return out, fmt.Errorf("converting usage based charge: %w", err)
		}
		apiUB := convertUsageBasedChargeToAPI(ub)
		apiUB.Type = api.BillingUsageBasedChargeTypeUsageBased
		apiUB.AdvanceAfter = TimePtrFromOptional(ub.State.AdvanceAfter)
		apiUB.Price = api.CurrencyAmount{Amount: "0"} // TODO: map complex productcatalog.Price type
		apiUB.Totals = convertUsageBasedChargeTotals(ub)
		apiUB.Discounts = convertUsageBasedDiscounts(ub.Intent.Discounts)
		if err := out.FromBillingUsageBasedCharge(apiUB); err != nil {
			return out, fmt.Errorf("setting usage based charge union: %w", err)
		}

	case meta.ChargeTypeCreditPurchase:
		// Credit purchases are excluded at the query level (ChargeTypes filter) and
		// should never reach this path. Return an error as a defensive measure.
		return out, fmt.Errorf("credit purchase charges are not supported in the charges API")

	default:
		return out, fmt.Errorf("unsupported charge type: %s", charge.Type())
	}

	return out, nil
}

// convertUsageBasedChargeTotals returns aggregated booked and realtime totals.
// TODO: implement proper totals aggregation from realization runs.
func convertUsageBasedChargeTotals(_ usagebased.Charge) api.BillingChargeTotals {
	return api.BillingChargeTotals{
		Booked: api.BillingTotals{
			Amount:              "0",
			ChargesTotal:        "0",
			CreditsTotal:        "0",
			DiscountsTotal:      "0",
			TaxesExclusiveTotal: "0",
		},
	}
}

// convertFlatFeeDiscounts maps the optional percentage discount to the anonymous API struct.
func convertFlatFeeDiscounts(pd *productcatalog.PercentageDiscount) *struct {
	Percentage *float32 `json:"percentage,omitempty"`
} {
	if pd == nil {
		return nil
	}
	pct := float32(pd.Percentage.InexactFloat64())
	return &struct {
		Percentage *float32 `json:"percentage,omitempty"`
	}{Percentage: &pct}
}

// convertUsageBasedDiscounts maps usage-based discounts to the API type.
func convertUsageBasedDiscounts(d productcatalog.Discounts) *api.BillingRateCardDiscounts {
	if d.Percentage == nil && d.Usage == nil {
		return nil
	}
	result := &api.BillingRateCardDiscounts{}
	if d.Percentage != nil {
		pct := float32(d.Percentage.Percentage.InexactFloat64())
		result.Percentage = &pct
	}
	if d.Usage != nil {
		s := d.Usage.Quantity.String()
		result.Usage = &s
	}
	return result
}

// ConvertUsageBasedStatusToAPI maps usage-based substates to their top-level API status.
// For example, "active.final_realization.started" maps to "active".
func ConvertUsageBasedStatusToAPI(status usagebased.Status) api.BillingChargeStatus {
	s := string(status)
	if idx := strings.IndexByte(s, '.'); idx >= 0 {
		s = s[:idx]
	}
	return api.BillingChargeStatus(s)
}

// ConvertClosedPeriodToAPI maps a domain ClosedPeriod to the API type.
func ConvertClosedPeriodToAPI(p timeutil.ClosedPeriod) api.ClosedPeriod {
	return api.ClosedPeriod{From: p.From, To: p.To}
}

// ConvertDecimalToCurrencyAmount wraps a decimal amount in a CurrencyAmount.
func ConvertDecimalToCurrencyAmount(d alpacadecimal.Decimal) api.CurrencyAmount {
	return api.CurrencyAmount{Amount: d.String()}
}

// ConvertCustomerIDToReference builds a BillingCustomerReference from a customer ID string.
func ConvertCustomerIDToReference(id string) api.BillingCustomerReference {
	return api.BillingCustomerReference{Id: id}
}

// ConvertProRatingConfigToAPI maps a ProRatingConfig to the API proration configuration.
func ConvertProRatingConfigToAPI(c productcatalog.ProRatingConfig) api.BillingRateCardProrationConfiguration {
	return api.BillingRateCardProrationConfiguration{
		Mode: api.BillingRateCardProrationMode(c.Mode),
	}
}

// ConvertSubscriptionRefToAPI maps a SubscriptionReference to the API type.
func ConvertSubscriptionRefToAPI(ref meta.SubscriptionReference) api.BillingSubscriptionReference {
	return api.BillingSubscriptionReference{
		Id: ref.SubscriptionID,
		Phase: struct {
			Id   api.ULID `json:"id"`
			Item struct {
				Id api.ULID `json:"id"`
			} `json:"item"`
		}{
			Id: ref.PhaseID,
			Item: struct {
				Id api.ULID `json:"id"`
			}{Id: ref.ItemID},
		},
	}
}

// ConvertChargeStatusToAPI casts a meta.ChargeStatus to api.BillingChargeStatus.
func ConvertChargeStatusToAPI(s meta.ChargeStatus) api.BillingChargeStatus {
	return api.BillingChargeStatus(s)
}

// ConvertSettlementModeToAPI casts a SettlementMode to its API equivalent.
func ConvertSettlementModeToAPI(s productcatalog.SettlementMode) api.BillingSettlementMode {
	return api.BillingSettlementMode(s)
}

// ConvertPaymentTermToAPI casts a PaymentTermType to its API equivalent.
func ConvertPaymentTermToAPI(pt productcatalog.PaymentTermType) api.BillingPricePaymentTerm {
	return api.BillingPricePaymentTerm(pt)
}

// ConvertManagedByToAPI casts an InvoiceLineManagedBy to its API equivalent.
func ConvertManagedByToAPI(mb billing.InvoiceLineManagedBy) api.ResourceManagedBy {
	return api.ResourceManagedBy(mb)
}

// ConvertCurrencyCodeToAPI casts a currencyx.Code to an API CurrencyCode.
func ConvertCurrencyCodeToAPI(c currencyx.Code) api.CurrencyCode {
	return api.CurrencyCode(c)
}

// TimePtrFromTime wraps a time value in a pointer, returning nil for zero times.
func TimePtrFromTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

// TimePtrFromOptional returns nil if the pointer is nil or points to a zero time.
func TimePtrFromOptional(t *time.Time) *time.Time {
	if t == nil || t.IsZero() {
		return nil
	}
	return t
}

// convertAPIChargeStatus maps an API status string to its domain equivalent.
func convertAPIChargeStatus(s string) (meta.ChargeStatus, error) {
	switch api.BillingChargeStatus(s) {
	case api.BillingChargeStatusCreated:
		return meta.ChargeStatusCreated, nil
	case api.BillingChargeStatusActive:
		return meta.ChargeStatusActive, nil
	case api.BillingChargeStatusFinal:
		return meta.ChargeStatusFinal, nil
	case api.BillingChargeStatusDeleted:
		return meta.ChargeStatusDeleted, nil
	default:
		return "", fmt.Errorf("unsupported charge status: %q", s)
	}
}
