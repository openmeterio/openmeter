package subscriptionhandler

import (
	"fmt"
	"testing"
	"time"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/datex"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const NotSet = ""

type PhaseIteratorTestSuite struct {
	*require.Assertions
	suite.Suite
}

func TestPhaseIterator(t *testing.T) {
	suite.Run(t, new(PhaseIteratorTestSuite))
}

func (s *PhaseIteratorTestSuite) SetupSuite() {
	s.Assertions = require.New(s.T())
}

func (s *PhaseIteratorTestSuite) newSIWWithBillingCadence(itemKey string, cadence string) subscription.SubscriptionItemView {
	return subscription.SubscriptionItemView{
		Spec: subscription.SubscriptionItemSpec{
			CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
				CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
					ItemKey: itemKey,
					RateCard: subscription.RateCard{
						Price:          productcatalog.NewPriceFrom(productcatalog.FlatPrice{}),
						BillingCadence: lo.ToPtr(datex.MustParse(s.T(), cadence)),
					},
				},
			},
		},
	}
}

func (s *PhaseIteratorTestSuite) newSIWWithBillingCadenceActiveFromTo(itemKey string, cadence string, activeFrom string, activeTo string) subscription.SubscriptionItemView {
	out := s.newSIWWithBillingCadence(itemKey, cadence)

	// TODO: validate if subscription fills this properly
	if activeFrom != "" {
		out.SubscriptionItem.ActiveFrom = lo.Must(time.Parse(time.RFC3339, activeFrom))
	}

	if activeTo != "" {
		out.SubscriptionItem.ActiveTo = lo.ToPtr(lo.Must(time.Parse(time.RFC3339, activeTo)))
	}

	return out
}

type expectedIterations struct {
	Start time.Time
	End   time.Time
	Key   string
}

func (s *PhaseIteratorTestSuite) mustParseTime(t string) time.Time {
	return lo.Must(time.Parse(time.RFC3339, t))
}

func (s *PhaseIteratorTestSuite) TestPhaseIterator() {
	tcs := []struct {
		name     string
		items    []subscription.SubscriptionItemView
		end      time.Time
		expected []expectedIterations
	}{
		{
			name:     "empty",
			items:    []subscription.SubscriptionItemView{},
			end:      s.mustParseTime("2021-01-01T00:00:00Z"),
			expected: []expectedIterations{},
		},
		{
			name: "sanity",
			items: []subscription.SubscriptionItemView{
				s.newSIWWithBillingCadence("item-key", "P1D"),
			},
			end: s.mustParseTime("2021-01-03T00:00:00Z"),
			expected: []expectedIterations{
				{
					Start: s.mustParseTime("2021-01-01T00:00:00Z"),
					End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					Key:   "subID/phase-test/item-key/period[0]",
				},
				{
					Start: s.mustParseTime("2021-01-02T00:00:00Z"),
					End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					Key:   "subID/phase-test/item-key/period[1]",
				},
			},
		},
		{
			name: "different cadence",
			items: []subscription.SubscriptionItemView{
				s.newSIWWithBillingCadence("item-key-1d", "P1D"),
				s.newSIWWithBillingCadence("item-key-2d", "P2D"),
			},
			end: s.mustParseTime("2021-01-04T00:00:00Z"),
			expected: []expectedIterations{
				{
					Start: s.mustParseTime("2021-01-01T00:00:00Z"),
					End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					Key:   "subID/phase-test/item-key-1d/period[0]",
				},
				{
					Start: s.mustParseTime("2021-01-02T00:00:00Z"),
					End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					Key:   "subID/phase-test/item-key-1d/period[1]",
				},
				{
					Start: s.mustParseTime("2021-01-03T00:00:00Z"),
					End:   s.mustParseTime("2021-01-04T00:00:00Z"),
					Key:   "subID/phase-test/item-key-1d/period[2]",
				},
				{
					Start: s.mustParseTime("2021-01-01T00:00:00Z"),
					End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					Key:   "subID/phase-test/item-key-2d/period[0]",
				},
				{
					Start: s.mustParseTime("2021-01-03T00:00:00Z"),
					End:   s.mustParseTime("2021-01-05T00:00:00Z"),
					Key:   "subID/phase-test/item-key-2d/period[1]",
				},
			},
		},
		{
			// Note: this happens on subscription updates, but the active to/from is always disjunct
			name: "active-from-to-matching-period",
			items: []subscription.SubscriptionItemView{
				s.newSIWWithBillingCadenceActiveFromTo("item-key", "P1D", NotSet, "2021-01-02T00:00:00Z"),
				s.newSIWWithBillingCadenceActiveFromTo("item-key", "P1D", "2021-01-02T00:00:00Z", NotSet),
			},
			end: s.mustParseTime("2021-01-03T00:00:00Z"),
			expected: []expectedIterations{
				{
					Start: s.mustParseTime("2021-01-01T00:00:00Z"),
					End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					Key:   "subID/phase-test/item-key/period[0]",
				},
				{
					Start: s.mustParseTime("2021-01-02T00:00:00Z"),
					End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					Key:   "subID/phase-test/item-key/period[1]",
				},
			},
		},
		{
			// Note: this happens on subscription updates, but the active to/from is always disjunct
			name: "active-from-to-matching-period",
			items: []subscription.SubscriptionItemView{
				s.newSIWWithBillingCadenceActiveFromTo("item-key", "P1D", NotSet, "2021-01-02T20:00:00Z"),
				s.newSIWWithBillingCadenceActiveFromTo("item-key", "P1D", "2021-01-02T20:00:00Z", NotSet),
			},
			end: s.mustParseTime("2021-01-03T00:00:00Z"),
			expected: []expectedIterations{
				{
					Start: s.mustParseTime("2021-01-01T00:00:00Z"),
					End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					Key:   "subID/phase-test/item-key/period[0]",
				},
				{
					Start: s.mustParseTime("2021-01-02T00:00:00Z"),
					End:   s.mustParseTime("2021-01-02T20:00:00Z"),
					Key:   "subID/phase-test/item-key/period[1]",
				},
				{
					Start: s.mustParseTime("2021-01-02T20:00:00Z"),
					End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					Key:   "subID/phase-test/item-key/period[2]",
				},
			},
		},
		// TODO: let's add flat fee tests
		// - flat fee with cadence (recurring)
		// - flat fee without cadence (one-time, in arrears) => only if we have phase end set
		// - flat fee without cadence (one-time, in advance) => ok
	}

	for _, tc := range tcs {
		s.Run(tc.name, func() {
			phase := subscription.SubscriptionPhaseView{
				SubscriptionPhase: subscription.SubscriptionPhase{
					ActiveFrom: lo.Must(time.Parse(time.RFC3339, "2021-01-01T00:00:00Z")),
					Key:        "phase-test",
				},
				ItemsByKey: map[string][]subscription.SubscriptionItemView{},
			}

			for _, item := range tc.items {
				if item.SubscriptionItem.ActiveFrom.IsZero() {
					item.SubscriptionItem.ActiveFrom = phase.SubscriptionPhase.ActiveFrom
				}

				phase.ItemsByKey[item.Spec.ItemKey] = append(phase.ItemsByKey[item.Spec.ItemKey], item)
			}

			it := NewPhaseIterator(
				phase,
				subscription.SubscriptionView{
					Subscription: subscription.Subscription{
						NamespacedID: models.NamespacedID{
							ID: "subID",
						},
					},
				},
				tc.end,
			)

			out := make([]rateCardWithPeriod, 0, 10)
			for item := range it.Seq() {
				out = append(out, item)
			}

			outAsExpect := make([]expectedIterations, 0, len(out))
			for i, item := range out {
				outAsExpect = append(outAsExpect, expectedIterations{
					Start: item.Period.Start,
					End:   item.Period.End,
					Key:   item.UniqueID,
				})

				// TODO: remove prints

				fmt.Printf("out[%d]: [%s..%s] %s\n", i, item.Period.Start, item.Period.End, item.UniqueID)
			}

			for i, item := range tc.expected {
				fmt.Printf("expected[%d]: [%s..%s] %s\n", i, item.Start, item.End, item.Key)
			}

			s.ElementsMatch(tc.expected, outAsExpect)
		})
	}
}
