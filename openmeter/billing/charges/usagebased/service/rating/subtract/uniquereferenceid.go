package subtract

import (
	"fmt"
	"testing"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
)

// UniqueReferenceIDGenerator generates ChildUniqueReferenceID values for lines
// emitted by subtraction.
type UniqueReferenceIDGenerator interface {
	// CurrentOnly returns the ChildUniqueReferenceID for a current detailed line
	// that had no matching previously billed detailed line.
	CurrentOnly(line usagebased.DetailedLine) (string, error)

	// MatchedDelta returns the ChildUniqueReferenceID for the delta between a
	// current detailed line and its matching previously billed detailed line.
	MatchedDelta(current, previous usagebased.DetailedLine) (string, error)

	// PreviousOnlyReversal returns the ChildUniqueReferenceID for a reversal of
	// a previously billed detailed line that had no matching current detailed line.
	PreviousOnlyReversal(line usagebased.DetailedLine) (string, error)
}

func NewMockUniqueReferenceIDGenerator(_ testing.TB) UniqueReferenceIDGenerator {
	return &mockUniqueReferenceIDGenerator{}
}

type mockUniqueReferenceIDGenerator struct{}

func (mockUniqueReferenceIDGenerator) CurrentOnly(line usagebased.DetailedLine) (string, error) {
	return line.ChildUniqueReferenceID, nil
}

func (mockUniqueReferenceIDGenerator) MatchedDelta(current, _ usagebased.DetailedLine) (string, error) {
	return current.ChildUniqueReferenceID, nil
}

func (mockUniqueReferenceIDGenerator) PreviousOnlyReversal(line usagebased.DetailedLine) (string, error) {
	return fmt.Sprintf(
		"%s#reversal:category=%s:payment_term=%s:per_unit_amount=%s",
		line.ChildUniqueReferenceID,
		line.Category,
		line.PaymentTerm,
		line.PerUnitAmount.String(),
	), nil
}
