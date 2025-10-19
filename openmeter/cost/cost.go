package cost

import (
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/pkg/currencyx"
)

// InvoiceLineCost InvoiceLineCost is the cost of a feature for an invoice.
type InvoiceLineCost struct {
	// Cost The total cost of the feature for the invoice.
	Cost alpacadecimal.Decimal `json:"cost"`

	// CostPerUnit The cost per unit of the feature for the invoice.
	CostPerUnit alpacadecimal.Decimal `json:"costPerUnit"`

	// Currency Currency of the cost.
	Currency currencyx.Code `json:"currency"`

	// From The start of the period the value is aggregated over.
	From time.Time `json:"from"`

	// InternalCost The total internal cost of the feature for the invoice.
	InternalCost *alpacadecimal.Decimal `json:"internalCost,omitempty"`

	// InternalCostPerUnit The internal cost per unit of the feature for the invoice.
	InternalCostPerUnit *alpacadecimal.Decimal `json:"internalCostPerUnit,omitempty"`

	// Margin The margin of the feature for the invoice.
	// Calculated as cost - internalCost.
	Margin *alpacadecimal.Decimal `json:"margin,omitempty"`

	// MarginRate The margin rate of the feature for the invoice.
	// Calculated as 1 - (internal cost / cost).
	MarginRate *alpacadecimal.Decimal `json:"marginRate,omitempty"`

	// Rows The rows of the feature cost for the invoice.
	Rows []InvoiceLineCostRow `json:"rows"`

	// To The end of the period the value is aggregated over.
	To time.Time `json:"to"`

	// Usage The total usage of the feature for the invoice.
	Usage alpacadecimal.Decimal `json:"usage"`
}

// InvoiceLineCostRow InvoiceLineCostRow is a row of the feature cost for the invoice.
type InvoiceLineCostRow struct {
	// Cost The cost of the feature for the invoice.
	Cost alpacadecimal.Decimal `json:"cost"`

	// CostPerUnit The cost per unit of the feature for the invoice.
	CostPerUnit alpacadecimal.Decimal `json:"costPerUnit"`

	// GroupBy The group by values the value is aggregated over.
	GroupBy map[string]*string `json:"groupBy"`

	// InternalCost The internal cost of the feature for the invoice.
	InternalCost *alpacadecimal.Decimal `json:"internalCost,omitempty"`

	// InternalCostPerUnit The internal cost per unit of the feature for the invoice.
	InternalCostPerUnit *alpacadecimal.Decimal `json:"internalCostPerUnit,omitempty"`

	// Margin The margin of the feature for the invoice.
	// Calculated as cost - internalCost.
	Margin *alpacadecimal.Decimal `json:"margin,omitempty"`

	// MarginRate The margin rate of the feature for the invoice.
	// Calculated as 1 - (internal cost / cost).
	MarginRate *alpacadecimal.Decimal `json:"marginRate,omitempty"`

	// Usage The usage of the feature for the invoice.
	Usage alpacadecimal.Decimal `json:"usage"`

	// WindowEnd The end of the window the value is aggregated over.
	WindowEnd time.Time `json:"windowEnd"`

	// WindowStart The start of the window the value is aggregated over.
	WindowStart time.Time `json:"windowStart"`
}
