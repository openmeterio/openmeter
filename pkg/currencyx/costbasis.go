package currencyx

import (
	"time"

	"github.com/alpacahq/alpacadecimal"
)

// CostBasis defines the exchange rate from a custom currency to a fiat currency
// over a specific time period. It is used to convert custom currency amounts
// (e.g. credits, tokens) into monetary values for billing and invoicing.
type CostBasis struct {
	// FiatCode is the target fiat currency code (e.g. USD, EUR) that the rate converts to.
	FiatCode Code `json:"fiat_code"`
	// Rate is the exchange rate: one unit of the custom currency equals this many units of FiatCode.
	Rate alpacadecimal.Decimal `json:"rate"`
	// EffectiveFrom is the start of the period during which this rate applies (inclusive).
	EffectiveFrom time.Time `json:"effective_from"`
	// EffectiveTo is the end of the period during which this rate applies (exclusive).
	// Nil means the rate is open-ended and currently active.
	EffectiveTo *time.Time `json:"effective_to,omitempty"`
}
