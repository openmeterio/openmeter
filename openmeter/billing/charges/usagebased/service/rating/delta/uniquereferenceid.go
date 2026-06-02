package delta

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
)

type uniqueReferenceIDGenerator struct{}

func (uniqueReferenceIDGenerator) CurrentOnly(line usagebased.DetailedLine) (string, error) {
	return line.ChildUniqueReferenceID, nil
}

func (uniqueReferenceIDGenerator) MatchedDelta(current, _ usagebased.DetailedLine) (string, error) {
	return current.ChildUniqueReferenceID, nil
}

func (uniqueReferenceIDGenerator) PreviousOnlyReversal(line usagebased.DetailedLine) (string, error) {
	if line.ID == "" {
		return "", fmt.Errorf("detailed line id is required")
	}

	return fmt.Sprintf("%s#correction:detailed_line_id=%s", line.PricerReferenceID, line.ID), nil
}
