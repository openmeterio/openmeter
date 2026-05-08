package periodpreserving

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
)

type generatedUniqueReferenceIDGenerator struct{}

func (generatedUniqueReferenceIDGenerator) CurrentOnly(line usagebased.DetailedLine) (string, error) {
	return line.ChildUniqueReferenceID, nil
}

func (generatedUniqueReferenceIDGenerator) MatchedDelta(current, _ usagebased.DetailedLine) (string, error) {
	return current.ChildUniqueReferenceID, nil
}

func (generatedUniqueReferenceIDGenerator) PreviousOnlyReversal(line usagebased.DetailedLine) (string, error) {
	return line.ChildUniqueReferenceID, nil
}

type bookedCorrectionUniqueReferenceIDGenerator struct{}

func (bookedCorrectionUniqueReferenceIDGenerator) CurrentOnly(line usagebased.DetailedLine) (string, error) {
	return line.ChildUniqueReferenceID, nil
}

func (bookedCorrectionUniqueReferenceIDGenerator) MatchedDelta(current, _ usagebased.DetailedLine) (string, error) {
	return current.ChildUniqueReferenceID, nil
}

func (bookedCorrectionUniqueReferenceIDGenerator) PreviousOnlyReversal(line usagebased.DetailedLine) (string, error) {
	if line.ID == "" {
		return "", fmt.Errorf("detailed line id is required")
	}

	return fmt.Sprintf("%s#correction:detailed_line_id=%s", line.PricerReferenceID, line.ID), nil
}
