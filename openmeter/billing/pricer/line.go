package pricer

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
	GetFeatureKey() string
}

type GatheringLineAccessor interface {
	PriceAccessor
	GetSplitLineGroupID() *string
	GetInvoiceAt() time.Time
	GetID() string
}

type StandardLineAccessor interface {
	PriceAccessor

	// GetCurrency returns the currency of the line
	GetCurrency() currencyx.Code
	// GetMeteredUsage returns the metered usage of the line for the current service period
	GetMeteredQuantity() (*alpacadecimal.Decimal, error)
	// GetMeteredPreLinePeriodUsage returns the metered usage of the line for the previous service period
	GetMeteredPreLinePeriodQuantity() (*alpacadecimal.Decimal, error)
	// GetCreditsApplied returns the list of credits to be applied for the line
	GetCreditsApplied() billing.CreditsApplied
	// GetName returns the name of the line
	GetName() string
	// GetRateCardDiscounts returns the rate card discounts for the line
	GetRateCardDiscounts() billing.Discounts
	// GetStandardLineDiscounts returns the standard line discounts for the line
	GetStandardLineDiscounts() billing.StandardLineDiscounts

	// Progressive billing related information
	// IsProgressivelyBilled returns true if the line is progressively billed
	IsProgressivelyBilled() bool
	// GetProgressivelyBilledServicePeriod returns the full service period of the line (if the line is progressively billed, or matches the service period)
	GetProgressivelyBilledServicePeriod() (timeutil.ClosedPeriod, error)

	// GetPreviouslyBilledAmount returns the amount that has already been billed for the line before the current line
	GetPreviouslyBilledAmount() (alpacadecimal.Decimal, error)
}
