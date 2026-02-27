package billing

import (
	"errors"

	"github.com/alpacahq/alpacadecimal"
)

type Totals struct {
	// Amount is the total amount value of the line before taxes, discounts and commitments
	Amount alpacadecimal.Decimal `json:"amount"`
	// ChargesTotal is the amount of value of the line that are due to additional charges
	ChargesTotal alpacadecimal.Decimal `json:"chargesTotal"`
	// DiscountsTotal is the amount of value of the line that are due to discounts
	DiscountsTotal alpacadecimal.Decimal `json:"discountsTotal"`

	// TaxesInclusiveTotal is the total amount of taxes that are included in the line
	TaxesInclusiveTotal alpacadecimal.Decimal `json:"taxesInclusiveTotal"`
	// TaxesExclusiveTotal is the total amount of taxes that are excluded from the line
	TaxesExclusiveTotal alpacadecimal.Decimal `json:"taxesExclusiveTotal"`
	// TaxesTotal is the total amount of taxes that are included in the line
	TaxesTotal alpacadecimal.Decimal `json:"taxesTotal"`

	// CreditsTotal is the total amount of credits that are applied to the line (credits are pre-tax)
	CreditsTotal alpacadecimal.Decimal `json:"creditsTotal"`

	// Total is the total amount value of the line after taxes, discounts and commitments
	Total alpacadecimal.Decimal `json:"total"`
}

func (t Totals) Validate() error {
	if t.Amount.IsNegative() {
		return errors.New("amount is negative")
	}

	if t.ChargesTotal.IsNegative() {
		return errors.New("charges total is negative")
	}

	if t.DiscountsTotal.IsNegative() {
		return errors.New("discounts total is negative")
	}

	if t.TaxesInclusiveTotal.IsNegative() {
		return errors.New("taxes inclusive total is negative")
	}

	if t.TaxesExclusiveTotal.IsNegative() {
		return errors.New("taxes exclusive total is negative")
	}

	if t.TaxesTotal.IsNegative() {
		return errors.New("taxes total is negative")
	}

	if t.Total.IsNegative() {
		return errors.New("total is negative")
	}

	if t.CreditsTotal.IsNegative() {
		return errors.New("credits total is negative")
	}

	return nil
}

func (t Totals) Add(others ...Totals) Totals {
	res := t

	for _, other := range others {
		res.Amount = res.Amount.Add(other.Amount)
		res.ChargesTotal = res.ChargesTotal.Add(other.ChargesTotal)
		res.DiscountsTotal = res.DiscountsTotal.Add(other.DiscountsTotal)
		res.TaxesInclusiveTotal = res.TaxesInclusiveTotal.Add(other.TaxesInclusiveTotal)
		res.TaxesExclusiveTotal = res.TaxesExclusiveTotal.Add(other.TaxesExclusiveTotal)
		res.TaxesTotal = res.TaxesTotal.Add(other.TaxesTotal)
		res.CreditsTotal = res.CreditsTotal.Add(other.CreditsTotal)
		res.Total = res.Total.Add(other.Total)
	}

	return res
}

func (t Totals) CalculateTotal() alpacadecimal.Decimal {
	return alpacadecimal.Sum(
		t.Amount,
		t.ChargesTotal,
		t.TaxesExclusiveTotal,
		t.DiscountsTotal.Neg(),
		t.CreditsTotal.Neg())
}
