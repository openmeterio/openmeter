package billingworkersubscription

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/datex"
	"github.com/openmeterio/openmeter/pkg/models"
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

type expectedIterations struct {
	Start           time.Time
	End             time.Time
	Key             string
	NonTruncatedEnd time.Time
}

type subscriptionItemViewMock struct {
	Key     string
	Cadence string

	ActiveFrom string
	ActiveTo   string

	Type productcatalog.PriceType
}

const NoPriceType = productcatalog.PriceType("NoPrice")

func (s *PhaseIteratorTestSuite) mustParseTime(t string) time.Time {
	return lo.Must(time.Parse(time.RFC3339, t))
}

func (s *PhaseIteratorTestSuite) TestPhaseIterator() {
	tcs := []struct {
		name        string
		items       []subscriptionItemViewMock
		end         time.Time
		expected    []expectedIterations
		phaseEnd    *time.Time
		expectError bool
	}{
		{
			name:     "empty",
			items:    []subscriptionItemViewMock{},
			end:      s.mustParseTime("2021-01-01T00:00:00Z"),
			expected: []expectedIterations{},
		},
		{
			name: "sanity",
			items: []subscriptionItemViewMock{
				{
					Key:     "item-key",
					Cadence: "P1D",
				},
			},
			end: s.mustParseTime("2021-01-03T00:00:00Z"),
			expected: []expectedIterations{
				{
					Start: s.mustParseTime("2021-01-01T00:00:00Z"),
					End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					Key:   "subID/phase-test/item-key/v[0]/period[0]",
				},
				{
					Start: s.mustParseTime("2021-01-02T00:00:00Z"),
					End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					Key:   "subID/phase-test/item-key/v[0]/period[1]",
				},
			},
		},
		{
			name: "sanity-non-billable-filtering",
			items: []subscriptionItemViewMock{
				{
					Key:     "item-key",
					Cadence: "P1D",
				},
				{
					Key:  "item-key-no-price",
					Type: NoPriceType,
				},
			},
			end: s.mustParseTime("2021-01-03T00:00:00Z"),
			expected: []expectedIterations{
				{
					Start: s.mustParseTime("2021-01-01T00:00:00Z"),
					End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					Key:   "subID/phase-test/item-key/v[0]/period[0]",
				},
				{
					Start: s.mustParseTime("2021-01-02T00:00:00Z"),
					End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					Key:   "subID/phase-test/item-key/v[0]/period[1]",
				},
			},
		},
		{
			name: "sanity-phase-end",
			items: []subscriptionItemViewMock{
				{
					Key:     "item-key",
					Cadence: "P1D",
				},
			},
			end: s.mustParseTime("2021-01-03T00:00:00Z"),
			expected: []expectedIterations{
				{
					Start: s.mustParseTime("2021-01-01T00:00:00Z"),
					End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					Key:   "subID/phase-test/item-key/v[0]/period[0]",
				},
				{
					Start:           s.mustParseTime("2021-01-02T00:00:00Z"),
					End:             s.mustParseTime("2021-01-02T15:00:00Z"),
					Key:             "subID/phase-test/item-key/v[0]/period[1]",
					NonTruncatedEnd: s.mustParseTime("2021-01-03T00:00:00Z"),
				},
			},
			phaseEnd: lo.ToPtr(s.mustParseTime("2021-01-02T15:00:00Z")),
		},
		{
			name: "different cadence",
			items: []subscriptionItemViewMock{
				{
					Key:     "item-key-1d",
					Cadence: "P1D",
				},
				{
					Key:     "item-key-2d",
					Cadence: "P2D",
				},
			},
			end: s.mustParseTime("2021-01-04T00:00:00Z"),
			expected: []expectedIterations{
				{
					Start: s.mustParseTime("2021-01-01T00:00:00Z"),
					End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					Key:   "subID/phase-test/item-key-1d/v[0]/period[0]",
				},
				{
					Start: s.mustParseTime("2021-01-02T00:00:00Z"),
					End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					Key:   "subID/phase-test/item-key-1d/v[0]/period[1]",
				},
				{
					Start: s.mustParseTime("2021-01-03T00:00:00Z"),
					End:   s.mustParseTime("2021-01-04T00:00:00Z"),
					Key:   "subID/phase-test/item-key-1d/v[0]/period[2]",
				},
				{
					Start: s.mustParseTime("2021-01-01T00:00:00Z"),
					End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					Key:   "subID/phase-test/item-key-2d/v[0]/period[0]",
				},
				{
					Start: s.mustParseTime("2021-01-03T00:00:00Z"),
					End:   s.mustParseTime("2021-01-05T00:00:00Z"),
					Key:   "subID/phase-test/item-key-2d/v[0]/period[1]",
				},
			},
		},
		{
			// Note: this happens on subscription updates, but the active to/from is always disjunct
			name: "active-from-to-matching-period",
			items: []subscriptionItemViewMock{
				{
					Key:      "item-key",
					Cadence:  "P1D",
					ActiveTo: "2021-01-02T00:00:00Z",
				},
				{
					Key:        "item-key",
					Cadence:    "P1D",
					ActiveFrom: "2021-01-02T00:00:00Z",
				},
			},
			end: s.mustParseTime("2021-01-03T00:00:00Z"),
			expected: []expectedIterations{
				{
					Start: s.mustParseTime("2021-01-01T00:00:00Z"),
					End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					Key:   "subID/phase-test/item-key/v[0]/period[0]",
				},
				{
					Start: s.mustParseTime("2021-01-02T00:00:00Z"),
					End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					Key:   "subID/phase-test/item-key/v[1]/period[0]",
				},
			},
		},
		{
			name: "active-from-to-missmatching-period",
			items: []subscriptionItemViewMock{
				{
					Key:      "item-key",
					Cadence:  "P1D",
					ActiveTo: "2021-01-02T20:00:00Z",
				},
				{
					Key:        "item-key",
					Cadence:    "P1D",
					ActiveFrom: "2021-01-02T20:00:00Z",
				},
			},
			end: s.mustParseTime("2021-01-03T00:00:00Z"),
			expected: []expectedIterations{
				{
					Start: s.mustParseTime("2021-01-01T00:00:00Z"),
					End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					Key:   "subID/phase-test/item-key/v[0]/period[0]",
				},
				{
					Start:           s.mustParseTime("2021-01-02T00:00:00Z"),
					End:             s.mustParseTime("2021-01-02T20:00:00Z"),
					Key:             "subID/phase-test/item-key/v[0]/period[1]",
					NonTruncatedEnd: s.mustParseTime("2021-01-03T00:00:00Z"),
				},
				{
					Start: s.mustParseTime("2021-01-02T20:00:00Z"),
					End:   s.mustParseTime("2021-01-03T20:00:00Z"),
					Key:   "subID/phase-test/item-key/v[1]/period[0]",
				},
			},
		},
		{
			name: "ubp-time truncated",
			items: []subscriptionItemViewMock{
				{
					Key:      "item-key",
					Cadence:  "P1D",
					ActiveTo: "2021-01-02T20:00:02Z",
					Type:     productcatalog.UnitPriceType,
				},
				{
					Key:        "item-key",
					Cadence:    "P1D",
					ActiveFrom: "2021-01-02T20:00:02Z",
					ActiveTo:   "2021-01-02T20:00:03Z",
					Type:       productcatalog.UnitPriceType,
				},
				{
					Key:        "item-key",
					Cadence:    "P1D",
					ActiveFrom: "2021-01-02T20:00:03Z",
					ActiveTo:   "2021-01-02T20:00:04Z",
					Type:       productcatalog.UnitPriceType,
				},
				{
					Key:        "item-key",
					Cadence:    "P1D",
					ActiveFrom: "2021-01-02T20:00:04Z",
					Type:       productcatalog.UnitPriceType,
				},
			},
			end: s.mustParseTime("2021-01-03T00:00:00Z"),
			expected: []expectedIterations{
				{
					Start: s.mustParseTime("2021-01-01T00:00:00Z"),
					End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					Key:   "subID/phase-test/item-key/v[0]/period[0]",
				},
				{
					Start:           s.mustParseTime("2021-01-02T00:00:00Z"),
					End:             s.mustParseTime("2021-01-02T20:00:00Z"),
					Key:             "subID/phase-test/item-key/v[0]/period[1]",
					NonTruncatedEnd: s.mustParseTime("2021-01-03T00:00:00Z"),
				},
				{
					Start: s.mustParseTime("2021-01-02T20:00:00Z"),
					End:   s.mustParseTime("2021-01-03T20:00:00Z"),
					Key:   "subID/phase-test/item-key/v[3]/period[0]",
				},
			},
		},
		{
			name: "flat-fee recurring",
			items: []subscriptionItemViewMock{
				{
					Key:     "item-key",
					Cadence: "P1D",
					Type:    productcatalog.FlatPriceType,
				},
			},
			end: s.mustParseTime("2021-01-03T00:00:00Z"),
			expected: []expectedIterations{
				{
					Start: s.mustParseTime("2021-01-01T00:00:00Z"),
					End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					Key:   "subID/phase-test/item-key/v[0]/period[0]",
				},
				{
					Start: s.mustParseTime("2021-01-02T00:00:00Z"),
					End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					Key:   "subID/phase-test/item-key/v[0]/period[1]",
				},
			},
		},
		{
			name: "flat-fee one-time",
			items: []subscriptionItemViewMock{
				{
					Key:  "item-key",
					Type: productcatalog.FlatPriceType,
				},
			},
			end:      s.mustParseTime("2021-01-03T00:00:00Z"),
			phaseEnd: lo.ToPtr(s.mustParseTime("2021-01-05T00:00:00Z")),
			expected: []expectedIterations{
				{
					Start: s.mustParseTime("2021-01-01T00:00:00Z"),
					End:   s.mustParseTime("2021-01-05T00:00:00Z"),
					Key:   "subID/phase-test/item-key/v[0]",
				},
			},
		},
		{
			name: "flat-fee recurring, edited",
			items: []subscriptionItemViewMock{
				{
					Key:      "item-key",
					Type:     productcatalog.FlatPriceType,
					Cadence:  "P1D",
					ActiveTo: "2021-01-02T20:00:00Z",
				},
				{
					Key:        "item-key",
					Type:       productcatalog.FlatPriceType,
					Cadence:    "P1D",
					ActiveFrom: "2021-01-02T20:00:00Z",
				},
			},
			end: s.mustParseTime("2021-01-03T00:00:00Z"),
			expected: []expectedIterations{
				{
					Start: s.mustParseTime("2021-01-01T00:00:00Z"),
					End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					Key:   "subID/phase-test/item-key/v[0]/period[0]",
				},
				{
					Start:           s.mustParseTime("2021-01-02T00:00:00Z"),
					End:             s.mustParseTime("2021-01-02T20:00:00Z"),
					Key:             "subID/phase-test/item-key/v[0]/period[1]",
					NonTruncatedEnd: s.mustParseTime("2021-01-03T00:00:00Z"),
				},
				{
					Start: s.mustParseTime("2021-01-02T20:00:00Z"),
					End:   s.mustParseTime("2021-01-03T20:00:00Z"),
					Key:   "subID/phase-test/item-key/v[1]/period[0]",
				},
			},
		},
		{
			name: "flat-fee one-time, no phase end",
			items: []subscriptionItemViewMock{
				{
					Key:  "item-key",
					Type: productcatalog.FlatPriceType,
				},
			},
			expectError: false,
		},
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
				sitem := subscription.SubscriptionItemView{}

				sitem.Spec.ItemKey = item.Key
				switch item.Type {
				case productcatalog.UnitPriceType:
					sitem.Spec.RateCard.Price = productcatalog.NewPriceFrom(productcatalog.UnitPrice{})
				case productcatalog.FlatPriceType:
					sitem.Spec.RateCard.Price = productcatalog.NewPriceFrom(productcatalog.FlatPrice{})
				case NoPriceType:
					sitem.Spec.RateCard.Price = nil
				default:
					sitem.Spec.RateCard.Price = productcatalog.NewPriceFrom(productcatalog.UnitPrice{})
				}

				if item.Cadence != "" {
					sitem.Spec.RateCard.BillingCadence = lo.ToPtr(datex.MustParse(s.T(), item.Cadence))
				}

				if item.ActiveFrom != "" {
					sitem.SubscriptionItem.ActiveFrom = lo.Must(time.Parse(time.RFC3339, item.ActiveFrom))
				}

				if item.ActiveTo != "" {
					sitem.SubscriptionItem.ActiveTo = lo.ToPtr(lo.Must(time.Parse(time.RFC3339, item.ActiveTo)))
				}

				if sitem.SubscriptionItem.ActiveFrom.IsZero() {
					sitem.SubscriptionItem.ActiveFrom = phase.SubscriptionPhase.ActiveFrom
				}

				phase.ItemsByKey[sitem.Spec.ItemKey] = append(phase.ItemsByKey[sitem.Spec.ItemKey], sitem)
			}

			subs := subscription.SubscriptionView{
				Subscription: subscription.Subscription{
					NamespacedID: models.NamespacedID{
						ID: "subID",
					},
				},
				Phases: []subscription.SubscriptionPhaseView{phase},
			}

			if tc.phaseEnd != nil {
				subs.Phases = append(subs.Phases, subscription.SubscriptionPhaseView{
					SubscriptionPhase: subscription.SubscriptionPhase{
						ActiveFrom: *tc.phaseEnd,
					},
				})
			}

			it, err := NewPhaseIterator(
				subs,
				phase.SubscriptionPhase.Key,
			)
			s.NoError(err)

			out, err := it.Generate(tc.end)
			if tc.expectError {
				s.Error(err)
				return
			} else {
				s.NoError(err)
			}

			outAsExpect := make([]expectedIterations, 0, len(out))
			for i, item := range out {
				// For now we never truncate the start, so we can just codify this
				s.Equal(item.Period.Start, item.NonTruncatedPeriod.Start)

				nonTruncatedEnd := time.Time{}
				if !item.NonTruncatedPeriod.End.Equal(item.Period.End) {
					nonTruncatedEnd = item.NonTruncatedPeriod.End
				}

				outAsExpect = append(outAsExpect, expectedIterations{
					Start:           item.Period.Start,
					End:             item.Period.End,
					Key:             item.UniqueID,
					NonTruncatedEnd: nonTruncatedEnd,
				})

				s.T().Logf("out[%d]: [%s..%s] %s (non-truncated: %s)\n", i, item.Period.Start, item.Period.End, item.UniqueID, nonTruncatedEnd)
			}

			for i, item := range tc.expected {
				s.T().Logf("expected[%d]: [%s..%s] %s (non-truncated: %s)\n", i, item.Start, item.End, item.Key, item.NonTruncatedEnd)
			}

			s.ElementsMatch(tc.expected, outAsExpect)
		})
	}
}
