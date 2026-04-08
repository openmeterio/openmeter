//go:generate go run github.com/jmattheis/goverter/cmd/goverter gen ./
package charges

import (
	"fmt"
	"strings"
	"time"

	"github.com/alpacahq/alpacadecimal"

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

// goverter:variables
// goverter:skipCopySameType
// goverter:output:file ./convert.gen.go
// goverter:useZeroValueOnPointerInconsistency
// goverter:useUnderlyingTypeMethods
// goverter:matchIgnoreCase
// goverter:extend ConvertMetadataToLabels
// goverter:extend TimePtrFromTime
// goverter:extend ConvertClosedPeriodToAPI
// goverter:extend ConvertDecimalToCurrencyAmount
// goverter:extend ConvertCustomerIDToReference
// goverter:extend ConvertProRatingConfigToAPI
// goverter:extend ConvertSubscriptionRefToAPI
// goverter:extend ConvertFeatureKeyToPtr
// goverter:extend ConvertUsageBasedStatusToAPI
// goverter:extend ConvertChargeStatusToAPI
// goverter:extend ConvertSettlementModeToAPI
// goverter:extend ConvertPaymentTermToAPI
// goverter:extend ConvertManagedByToAPI
// goverter:extend ConvertCurrencyCodeToAPI
var (

	// goverter:ignore Type
	// goverter:ignore Discounts
	// goverter:ignore AdvanceAfter
	// Fields inside meta.Intent (embedded within flatfee.Intent) need Intent.Intent.* paths.
	// Fields in models.ManagedModel need ManagedResource.ManagedModel.* paths.
	// goverter:map ManagedResource.ID Id
	// goverter:map ManagedResource.ManagedModel.CreatedAt CreatedAt
	// goverter:map ManagedResource.ManagedModel.UpdatedAt UpdatedAt
	// goverter:map ManagedResource.ManagedModel.DeletedAt DeletedAt
	// goverter:map Intent.Intent.Name Name
	// goverter:map Intent.Intent.Description Description
	// goverter:map Intent.Intent.Metadata Labels
	// goverter:map Intent.Intent.CustomerID Customer
	// goverter:map Intent.Intent.Currency Currency
	// goverter:map Intent.Intent.ManagedBy ManagedBy
	// goverter:map Intent.InvoiceAt InvoiceAt
	// goverter:map Intent.SettlementMode SettlementMode
	// goverter:map Intent.PaymentTerm PaymentTerm
	// goverter:map Intent.Intent.ServicePeriod ServicePeriod
	// goverter:map Intent.Intent.FullServicePeriod FullServicePeriod
	// goverter:map Intent.Intent.BillingPeriod BillingPeriod
	// goverter:map Intent.Intent.UniqueReferenceID UniqueReferenceId
	// goverter:map Intent.AmountBeforeProration Price
	// goverter:map State.AmountAfterProration AmountAfterProration
	// goverter:map Intent.ProRating ProrationConfiguration
	// goverter:map Intent.FeatureKey FeatureKey
	// goverter:map Intent.Intent.Subscription Subscription
	ConvertFlatFeeChargeToAPI func(flatfee.Charge) (api.BillingFlatFeeCharge, error)

	// goverter:ignore Type
	// goverter:ignore Discounts
	// goverter:ignore Totals
	// goverter:ignore Price
	// goverter:ignore AdvanceAfter
	// usagebased.Charge embeds ChargeBase anonymously; goverter requires the full path.
	// Fields inside meta.Intent (embedded in usagebased.Intent) need ChargeBase.Intent.Intent.* paths.
	// goverter:map ChargeBase.ManagedResource.ID Id
	// goverter:map ChargeBase.ManagedResource.ManagedModel.CreatedAt CreatedAt
	// goverter:map ChargeBase.ManagedResource.ManagedModel.UpdatedAt UpdatedAt
	// goverter:map ChargeBase.ManagedResource.ManagedModel.DeletedAt DeletedAt
	// goverter:map ChargeBase.Status Status
	// goverter:map ChargeBase.Intent.Intent.Name Name
	// goverter:map ChargeBase.Intent.Intent.Description Description
	// goverter:map ChargeBase.Intent.Intent.Metadata Labels
	// goverter:map ChargeBase.Intent.Intent.CustomerID Customer
	// goverter:map ChargeBase.Intent.Intent.Currency Currency
	// goverter:map ChargeBase.Intent.Intent.ManagedBy ManagedBy
	// goverter:map ChargeBase.Intent.InvoiceAt InvoiceAt
	// goverter:map ChargeBase.Intent.SettlementMode SettlementMode
	// goverter:map ChargeBase.Intent.Intent.ServicePeriod ServicePeriod
	// goverter:map ChargeBase.Intent.Intent.FullServicePeriod FullServicePeriod
	// goverter:map ChargeBase.Intent.Intent.BillingPeriod BillingPeriod
	// goverter:map ChargeBase.Intent.Intent.UniqueReferenceID UniqueReferenceId
	// goverter:map ChargeBase.Intent.FeatureKey FeatureKey
	// goverter:map ChargeBase.Intent.Intent.Subscription Subscription
	ConvertUsageBasedChargeToAPI func(usagebased.Charge) (api.BillingUsageBasedCharge, error)
)

// convertChargeToAPI dispatches on charge type, delegates field-by-field mapping to the
// generated converters, then sets the fields goverter cannot produce (Type constant,
// anonymous-struct Discounts, Totals stub, Price placeholder).
func convertChargeToAPI(charge billingcharges.Charge) (api.BillingCharge, error) {
	var out api.BillingCharge

	switch charge.Type() {
	case meta.ChargeTypeFlatFee:
		ff, err := charge.AsFlatFeeCharge()
		if err != nil {
			return out, fmt.Errorf("converting flat fee charge: %w", err)
		}
		apiFF, err := ConvertFlatFeeChargeToAPI(ff)
		if err != nil {
			return out, fmt.Errorf("converting flat fee charge fields: %w", err)
		}
		apiFF.Type = api.BillingFlatFeeChargeTypeFlatFee
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
		apiUB, err := ConvertUsageBasedChargeToAPI(ub)
		if err != nil {
			return out, fmt.Errorf("converting usage based charge fields: %w", err)
		}
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
// Goverter cannot generate code for anonymous struct target types.
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
// Goverter cannot generate this due to the conditional multi-field construction logic.
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

// --- Extend functions used by goverter ---

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
// Goverter handles the *SubscriptionReference → *BillingSubscriptionReference nil check
// automatically because this extend function operates on the dereferenced value.
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

// ConvertFeatureKeyToPtr converts a feature key string to a pointer, returning nil when empty.
// This prevents goverter's useZeroValueOnPointerInconsistency from creating non-nil pointers
// for empty feature key strings on flat fee charges.
func ConvertFeatureKeyToPtr(key string) *string {
	if key == "" {
		return nil
	}
	return &key
}

// ConvertChargeStatusToAPI casts a meta.ChargeStatus to api.BillingChargeStatus.
// Hand-written: goverter's enum-name matching requires identical const names across packages,
// but meta uses ChargeStatusActive while api uses BillingChargeStatusActive.
func ConvertChargeStatusToAPI(s meta.ChargeStatus) api.BillingChargeStatus {
	return api.BillingChargeStatus(s)
}

// ConvertSettlementModeToAPI casts a SettlementMode to its API equivalent.
func ConvertSettlementModeToAPI(s productcatalog.SettlementMode) api.BillingSettlementMode {
	return api.BillingSettlementMode(s)
}

// ConvertPaymentTermToAPI casts a PaymentTermType to its API equivalent.
// Also kept hand-written because productcatalog.DefaultPaymentTerm has no matching
// const name in api.BillingPricePaymentTerm, so @ignore would emit an empty string.
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
// Used by goverter to convert CreatedAt/UpdatedAt (time.Time) to *time.Time.
func TimePtrFromTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

// TimePtrFromOptional returns nil if the pointer is nil or points to a zero time, otherwise
// returns the pointer unchanged. Used for AdvanceAfter fields that live inside nested State
// structs that goverter cannot navigate with its map-plus-converter syntax.
func TimePtrFromOptional(t *time.Time) *time.Time {
	if t == nil || t.IsZero() {
		return nil
	}
	return t
}
