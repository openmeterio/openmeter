package stdinvoicesanity

import (
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/models"
)

// validateCreditTotals checks that each active detail's CreditsTotal matches
// its allocations, rounding each allocation before adding them. Apps send the
// allocations separately, so the stored total must agree with what they send.
// No allocations means a zero total. Deleted parents and details are skipped.
// Other checks cover realization IDs; this check does not look up ledger ownership
// or prevent one realization from being split across details.
func validateCreditTotals(invoice billing.StandardInvoice) error {
	currency, err := invoice.Currency.AsFiatCurrency()
	if err != nil {
		// validateCurrency already reports this; avoid repeating the same issue.
		return nil
	}

	var errs []error
	for lineIdx, line := range invoice.Lines.OrEmpty() {
		if line == nil || line.DeletedAt != nil {
			continue
		}

		for detailIdx, detail := range line.DetailedLines {
			if detail.DeletedAt != nil {
				continue
			}

			expected := detail.CreditsApplied.SumAmount(currency)
			if expected.Equal(detail.Totals.CreditsTotal) {
				continue
			}

			errs = append(errs, billing.ValidationWithFieldPrefix(
				fmt.Sprintf("lines/%d/detailedLines/%d/totals/creditsTotal", lineIdx, detailIdx),
				billing.ValidationWithAttributes(models.Annotations{
					"invoice_id":       invoice.ID,
					"line_id":          line.ID,
					"detailed_line_id": detail.ID,
					"expected":         expected.String(),
					"actual":           detail.Totals.CreditsTotal.String(),
				}, ErrCreditTotalMismatch),
			))
		}
	}

	return errors.Join(errs...)
}

// validateCreditConservation checks that each parent's credits reach its active
// details with the same amount for each realization ID. One realization can be
// split across details, but losing, duplicating or changing its ID would change
// which credits the app applies. Round each entry before summing, as billing does.
func validateCreditConservation(invoice billing.StandardInvoice) error {
	currency, err := invoice.Currency.AsFiatCurrency()
	if err != nil {
		return nil
	}

	var errs []error
	for lineIdx, line := range invoice.Lines.OrEmpty() {
		if line == nil || line.DeletedAt != nil {
			continue
		}

		expected := map[string]alpacadecimal.Decimal{}
		actual := map[string]alpacadecimal.Decimal{}
		for _, credit := range line.CreditsApplied {
			expected[credit.CreditRealizationID] = expected[credit.CreditRealizationID].Add(currency.RoundToPrecision(credit.Amount))
		}

		for _, detail := range line.DetailedLines {
			if detail.DeletedAt != nil {
				continue
			}

			for _, credit := range detail.CreditsApplied {
				actual[credit.CreditRealizationID] = actual[credit.CreditRealizationID].Add(currency.RoundToPrecision(credit.Amount))
			}
		}

		ids := map[string]struct{}{}
		for id := range expected {
			ids[id] = struct{}{}
		}

		for id := range actual {
			ids[id] = struct{}{}
		}

		for _, id := range slices.Sorted(maps.Keys(ids)) {
			if expected[id].Equal(actual[id]) {
				continue
			}

			errs = append(errs, billing.ValidationWithFieldPrefix(fmt.Sprintf("lines/%d/creditsApplied", lineIdx),
				billing.ValidationWithAttributes(models.Annotations{
					"invoice_id":            invoice.ID,
					"line_id":               line.ID,
					"credit_realization_id": id,
					"expected":              expected[id].String(),
					"actual":                actual[id].String(),
				}, ErrCreditConservationMismatch)))
		}
	}

	return errors.Join(errs...)
}

// validateCreditEntries requires positive credit amounts and realization IDs
// on active parents and details. A negative credit adds a charge, and a missing
// ID leaves us unable to match the allocation to its realization. This check
// does not look up whether that realization belongs to the invoice's account.
func validateCreditEntries(invoice billing.StandardInvoice) error {
	var errs []error
	for lineIdx, line := range invoice.Lines.OrEmpty() {
		if line == nil || line.DeletedAt != nil {
			continue
		}

		lineErrs := []error{validateAppliedCredits(line.CreditsApplied)}
		for detailIdx, detail := range line.DetailedLines {
			if detail.DeletedAt != nil {
				continue
			}

			lineErrs = append(lineErrs, billing.ValidationWithFieldPrefix(fmt.Sprintf("detailedLines/%d", detailIdx), billing.ValidationWithAttributes(models.Annotations{"detailed_line_id": detail.ID}, validateAppliedCredits(detail.CreditsApplied))))
		}

		errs = append(errs, billing.ValidationWithFieldPrefix(fmt.Sprintf("lines/%d", lineIdx), billing.ValidationWithAttributes(models.Annotations{"line_id": line.ID}, errors.Join(lineErrs...))))
	}

	return billing.ValidationWithAttributes(models.Annotations{"invoice_id": invoice.ID}, errors.Join(errs...))
}

func validateAppliedCredits(credits billing.CreditsApplied) error {
	var errs []error
	for idx, credit := range credits {
		if !credit.Amount.IsPositive() {
			errs = append(errs, billing.ValidationWithFieldPrefix(fmt.Sprintf("creditsApplied/%d/amount", idx), billing.ValidationWithAttributes(models.Annotations{"actual": credit.Amount.String()}, ErrInvalidCreditAmount)))
		}

		if credit.CreditRealizationID == "" {
			errs = append(errs, billing.ValidationWithFieldPrefix(fmt.Sprintf("creditsApplied/%d/creditRealizationID", idx), ErrMissingCreditRealizationID))
		}
	}

	return errors.Join(errs...)
}

// validateCreditAllocationLimits checks that allocated credits fit within a
// positive detail balance: amount plus charges minus discounts. Credits cannot
// pay tax. Use the allocations themselves because CreditsTotal may be wrong.
// Negative correction details are valid as long as they have no allocations.
func validateCreditAllocationLimits(invoice billing.StandardInvoice) error {
	currency, err := invoice.Currency.AsFiatCurrency()
	if err != nil {
		return nil
	}

	var errs []error
	for lineIdx, line := range invoice.Lines.OrEmpty() {
		if line == nil || line.DeletedAt != nil {
			continue
		}

		for detailIdx, detail := range line.DetailedLines {
			if detail.DeletedAt != nil || len(detail.CreditsApplied) == 0 {
				continue
			}

			available := detail.Totals.Amount.Add(detail.Totals.ChargesTotal).Sub(detail.Totals.DiscountsTotal)
			applied := detail.CreditsApplied.SumAmount(currency)
			if available.IsPositive() && !applied.GreaterThan(available) {
				continue
			}

			errs = append(errs, billing.ValidationWithFieldPrefix(fmt.Sprintf("lines/%d/detailedLines/%d/creditsApplied", lineIdx, detailIdx), billing.ValidationWithAttributes(models.Annotations{
				"invoice_id":       invoice.ID,
				"line_id":          line.ID,
				"detailed_line_id": detail.ID,
				"expected":         available.String(),
				"actual":           applied.String(),
			}, ErrCreditAllocationLimit)))
		}
	}

	return errors.Join(errs...)
}
