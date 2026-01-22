package invoicecalc

import (
	"fmt"

	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

func UpsertDiscountCorrelationIDs(invoice *billing.StandardInvoice) error {
	lines := invoice.Lines.OrEmpty()
	for _, line := range lines {
		updatedDiscounts, err := ensureDiscountCorrelationIDs(line.RateCardDiscounts)
		if err != nil {
			return err
		}

		line.RateCardDiscounts = updatedDiscounts
	}

	return nil
}

func ensureDiscountCorrelationIDs(discounts billing.Discounts) (billing.Discounts, error) {
	if discounts.Percentage != nil {
		corrID, err := generateDiscountCorrelationID(discounts.Percentage.CorrelationID)
		if err != nil {
			return billing.Discounts{}, err
		}
		discounts.Percentage.CorrelationID = corrID
	}

	if discounts.Usage != nil {
		corrID, err := generateDiscountCorrelationID(discounts.Usage.CorrelationID)
		if err != nil {
			return billing.Discounts{}, err
		}
		discounts.Usage.CorrelationID = corrID
	}

	return discounts, nil
}

func generateDiscountCorrelationID(correlationID string) (string, error) {
	if correlationID == "" {
		return ulid.Make().String(), nil
	}

	_, err := ulid.Parse(correlationID)
	if err != nil {
		return "", fmt.Errorf("invalid correlation ID: %w", err)
	}

	return correlationID, nil
}
