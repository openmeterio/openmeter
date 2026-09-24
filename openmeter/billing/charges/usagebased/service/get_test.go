package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
)

func TestWithRealtimeValidationIssuesReplacesOnlyRatingIssues(t *testing.T) {
	storedRating := billing.ValidationIssue{Component: billing.ValidationComponentBillingRating, Code: "stored"}
	other := billing.ValidationIssue{Component: billing.ValidationComponentProductCatalog, Code: "other"}
	liveRating := billing.ValidationIssue{Component: billing.ValidationComponentBillingRating, Code: "live"}
	charge := usagebased.Charge{ChargeBase: usagebased.ChargeBase{
		ValidationIssues: billing.ValidationIssues{storedRating, other},
	}}

	withWarnings := withRealtimeValidationIssues(charge, billing.ValidationIssues{liveRating})
	require.Equal(t, billing.ValidationIssues{other, liveRating}, withWarnings.ValidationIssues)

	withoutWarnings := withRealtimeValidationIssues(charge, nil)
	require.Equal(t, billing.ValidationIssues{other}, withoutWarnings.ValidationIssues)

	require.Equal(t, billing.ValidationIssues{storedRating, other}, charge.ValidationIssues)
}
