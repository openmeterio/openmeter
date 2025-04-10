package billingservice

import (
	"fmt"

	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

func (s *Service) generateDiscountCorrelationIDs(discounts billing.Discounts) (billing.Discounts, error) {
	if discounts.Percentage != nil {
		corrID, err := s.generateCorrelationID(discounts.Percentage.CorrelationID)
		if err != nil {
			return billing.Discounts{}, err
		}
		discounts.Percentage.CorrelationID = corrID
	}

	if discounts.Usage != nil {
		corrID, err := s.generateCorrelationID(discounts.Usage.CorrelationID)
		if err != nil {
			return billing.Discounts{}, err
		}
		discounts.Usage.CorrelationID = corrID
	}

	return discounts, nil
}

func (s *Service) generateCorrelationID(correlationID string) (string, error) {
	if correlationID == "" {
		return ulid.Make().String(), nil
	}

	_, err := ulid.Parse(correlationID)
	if err != nil {
		return "", fmt.Errorf("invalid correlation ID: %w", err)
	}

	return correlationID, nil
}
