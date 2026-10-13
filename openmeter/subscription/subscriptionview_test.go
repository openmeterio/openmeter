package subscription_test

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestSubscriptionViewLegacyCadenceKeepsOtherValidationErrors(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name        string
		rateCardKey string
		featureID   *string
		wantError   string
	}{
		{name: "short cadence only", rateCardKey: "fee"},
		{name: "short cadence and key mismatch", rateCardKey: "other", wantError: "rate card key must match feature key"},
		{name: "feature identity mismatch", rateCardKey: "fee", featureID: lo.ToPtr("other-feature"), wantError: "id mismatch between reference and feature"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// given: a weekly subscription with a persisted hourly item and a resolvable feature
			sub := subscription.Subscription{
				NamespacedID:  models.NamespacedID{Namespace: "default", ID: "subscription"},
				CadencedModel: models.CadencedModel{ActiveFrom: start},
				CustomerId:    "customer", InvoiceCurrency: "USD", BillingAnchor: start,
				BillingCadence: datetime.MustParseDuration(t, "P1W"),
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
			}
			phase := subscription.SubscriptionPhase{
				NamespacedID: models.NamespacedID{Namespace: sub.Namespace, ID: "phase"},
				ActiveFrom:   start, SubscriptionID: sub.ID, Key: "default",
			}
			item := subscription.SubscriptionItem{
				NamespacedID:   models.NamespacedID{Namespace: sub.Namespace, ID: "item"},
				CadencedModel:  models.CadencedModel{ActiveFrom: start},
				SubscriptionId: sub.ID, PhaseId: phase.ID, Key: "fee",
				RateCard: &productcatalog.FlatFeeRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key: tc.rateCardKey, Name: "Fee",
						Feature: productcatalog.NewFeatureReference(tc.featureID, lo.ToPtr("fee")),
					},
					BillingCadence: lo.ToPtr(datetime.MustParseDuration(t, "PT1H")),
				},
			}
			feat := feature.Feature{
				Namespace: sub.Namespace, ID: "feature", Key: "fee", Name: "Fee",
				CreatedAt: start, UpdatedAt: start,
			}

			// when: view loading resolves the item's feature
			view, err := subscription.NewSubscriptionView(sub, customer.Customer{}, []subscription.SubscriptionPhase{phase},
				[]subscription.SubscriptionItem{item}, nil, nil, []feature.Feature{feat})

			// then: only the legacy cadence error is tolerated on reads
			if tc.wantError != "" {
				require.ErrorContains(t, err, tc.wantError)
				require.ErrorContains(t, err, "failed to resolve feature reference")

				return
			}

			require.NoError(t, err)
			require.Equal(t, "PT1H", view.Phases[0].ItemsByKey["fee"][0].Spec.RateCard.GetBillingCadence().String())
			require.ErrorIs(t, view.Spec.Validate(), productcatalog.ErrRateCardBillingCadenceTooShort)
		})
	}
}

func TestSubscriptionViewPreservesIntendedStartsAfterCancellation(t *testing.T) {
	// given: cancellation clips two differently scheduled revisions to the same empty interval
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	sub := subscription.Subscription{
		NamespacedID:  models.NamespacedID{Namespace: "default", ID: "subscription"},
		CadencedModel: models.CadencedModel{ActiveFrom: start, ActiveTo: &end},
		Name:          "Subscription", CustomerId: "customer", InvoiceCurrency: "USD",
		BillingAnchor: start, BillingCadence: datetime.MustParseDuration(t, "P1M"),
		SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
	}
	phase := subscription.SubscriptionPhase{
		NamespacedID: models.NamespacedID{Namespace: sub.Namespace, ID: "phase"},
		ActiveFrom:   start, SubscriptionID: sub.ID, Key: "default", Name: "Default phase",
	}
	items := []subscription.SubscriptionItem{
		{
			NamespacedID:                         models.NamespacedID{Namespace: sub.Namespace, ID: "original"},
			CadencedModel:                        models.CadencedModel{ActiveFrom: start, ActiveTo: &end},
			ActiveToOverrideRelativeToPhaseStart: lo.ToPtr(datetime.MustParseDuration(t, "PT2H")),
		},
		{
			NamespacedID:                           models.NamespacedID{Namespace: sub.Namespace, ID: "first-future"},
			CadencedModel:                          models.CadencedModel{ActiveFrom: end, ActiveTo: &end},
			ActiveFromOverrideRelativeToPhaseStart: lo.ToPtr(datetime.MustParseDuration(t, "PT2H")),
			ActiveToOverrideRelativeToPhaseStart:   lo.ToPtr(datetime.MustParseDuration(t, "PT3H")),
			Annotations:                            models.Annotations{subscription.AnnotationEditUniqueKey: "01J00000000000000000000002"},
		},
		{
			NamespacedID:                           models.NamespacedID{Namespace: sub.Namespace, ID: "second-future"},
			CadencedModel:                          models.CadencedModel{ActiveFrom: end, ActiveTo: &end},
			ActiveFromOverrideRelativeToPhaseStart: lo.ToPtr(datetime.MustParseDuration(t, "PT3H")),
			Annotations:                            models.Annotations{subscription.AnnotationEditUniqueKey: "01J00000000000000000000001"},
		},
	}
	for i := range items {
		items[i].SubscriptionId = sub.ID
		items[i].PhaseId = phase.ID
		items[i].Key = "fee"
		items[i].Name = items[i].ID
		items[i].RateCard = &productcatalog.FlatFeeRateCard{RateCardMeta: productcatalog.RateCardMeta{
			Key: "fee", Name: items[i].Name,
		}}
	}

	// when: the view receives shuffled rows whose patch IDs oppose intended start order
	view, err := subscription.NewSubscriptionView(sub, customer.Customer{}, []subscription.SubscriptionPhase{phase},
		[]subscription.SubscriptionItem{items[2], items[0], items[1]}, nil, nil, nil)
	require.NoError(t, err)

	// then: the history still follows the intended schedule rather than clipped starts or patch IDs
	history := view.Phases[0].ItemsByKey["fee"]
	require.Equal(t, []string{"original", "first-future", "second-future"}, lo.Map(history, func(item subscription.SubscriptionItemView, _ int) string {
		return item.SubscriptionItem.ID
	}))
	for i, item := range view.Spec.Phases[phase.Key].ItemsByKey["fee"] {
		require.Equal(t, items[i].ActiveFromOverrideRelativeToPhaseStart, item.ActiveFromOverrideRelativeToPhaseStart)
	}
}
