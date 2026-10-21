package rating

import (
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type PriceAccessor interface {
	GetPrice() *productcatalog.Price
	GetServicePeriod() timeutil.ClosedPeriod
}

type GatheringLineAccessor interface {
	PriceAccessor
	GetSplitLineGroupID() *string
	GetInvoiceAt() time.Time
	GetID() string
}

type StandardLineAccessor interface {
	PriceAccessor

	// TODO: Drop the error return once billing has migrated to the new currency engine
	// and no longer needs to construct fiat currency calculators from stored codes.
	// GetCurrencyCalculator returns the currency calculator of the line.
	GetCurrencyCalculator() (currencyx.Currency, error)
	// GetMeteredUsage returns the metered usage of the line for the current service period
	GetMeteredQuantity() (*alpacadecimal.Decimal, error)
	// GetCreditsApplied returns the list of credits to be applied for the line
	GetCreditsApplied() billing.CreditsApplied
	// GetName returns the name of the line
	GetName() string
	// GetRateCardDiscounts returns the rate card discounts for the line
	GetRateCardDiscounts() billing.Discounts
	// GetUnitConfig returns the optional unit conversion to apply to the raw metered
	// quantity before pricing. Nil means no conversion (rating is unchanged).
	GetUnitConfig() *productcatalog.UnitConfig
	// GetStandardLineDiscounts returns the standard line discounts for the line
	GetStandardLineDiscounts() billing.StandardLineDiscounts
}

type LegacyProgressivelyBillableLineAccessor interface {
	StandardLineAccessor

	// GetMeteredPreLinePeriodQuantity returns raw usage preceding the line's service period.
	GetMeteredPreLinePeriodQuantity() (*alpacadecimal.Decimal, error)
	IsProgressivelyBilled() bool
	// GetProgressivelyBilledServicePeriod returns the full service period, including
	// earlier splits, or the line's service period when it is not progressively billed.
	GetProgressivelyBilledServicePeriod() (timeutil.ClosedPeriod, error)
	// GetPreviouslyBilledAmount returns the amount billed before the current line.
	GetPreviouslyBilledAmount() (alpacadecimal.Decimal, error)
}
