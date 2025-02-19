package billingworkersubscription

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/isodate"
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
	Start             time.Time
	End               time.Time
	Key               string
	NonTruncatedStart time.Time
	NonTruncatedEnd   time.Time
}

type subscriptionItemViewMock struct {
	Key     string
	Cadence string

	StartAfter *isodate.Period
	EndAfter   *isodate.Period

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
		expectedErr error
		phaseEnd    *time.Time
		alignedSub  bool
	}{
		{
			name:     "empty",
			items:    []subscriptionItemViewMock{},
			end:      s.mustParseTime("2021-01-01T00:00:00Z"),
			expected: []expectedIterations{},
		},
		{
			name:       "empty2",
			items:      []subscriptionItemViewMock{},
			alignedSub: true,
			end:        s.mustParseTime("2021-01-01T00:00:00Z"),
			expected:   []expectedIterations{},
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
					EndAfter: lo.ToPtr(isodate.MustParse(s.T(), "P1D")),
				},
				{
					Key:        "item-key",
					Cadence:    "P1D",
					StartAfter: lo.ToPtr(isodate.MustParse(s.T(), "P1D")),
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
					EndAfter: lo.ToPtr(isodate.MustParse(s.T(), "P1DT20H")),
				},
				{
					Key:        "item-key",
					Cadence:    "P1D",
					StartAfter: lo.ToPtr(isodate.MustParse(s.T(), "P1DT20H")),
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
					EndAfter: lo.ToPtr(isodate.MustParse(s.T(), "P1DT20H2S")),
					Type:     productcatalog.UnitPriceType,
				},
				{
					Key:        "item-key",
					Cadence:    "P1D",
					StartAfter: lo.ToPtr(isodate.MustParse(s.T(), "P1DT20H2S")),
					EndAfter:   lo.ToPtr(isodate.MustParse(s.T(), "P1DT20H3S")),
					Type:       productcatalog.UnitPriceType,
				},
				{
					Key:        "item-key",
					Cadence:    "P1D",
					StartAfter: lo.ToPtr(isodate.MustParse(s.T(), "P1DT20H3S")),
					EndAfter:   lo.ToPtr(isodate.MustParse(s.T(), "P1DT20H4S")),
					Type:       productcatalog.UnitPriceType,
				},
				{
					Key:        "item-key",
					Cadence:    "P1D",
					StartAfter: lo.ToPtr(isodate.MustParse(s.T(), "P1DT20H4S")),
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
					EndAfter: lo.ToPtr(isodate.MustParse(s.T(), "P1DT20H")),
				},
				{
					Key:        "item-key",
					Type:       productcatalog.FlatPriceType,
					Cadence:    "P1D",
					StartAfter: lo.ToPtr(isodate.MustParse(s.T(), "P1DT20H")),
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
			name: "aligned flat-fee recurring",
			items: []subscriptionItemViewMock{
				{
					Key:      "item-key",
					Type:     productcatalog.FlatPriceType,
					Cadence:  "P1D",
					EndAfter: lo.ToPtr(isodate.MustParse(s.T(), "P1DT20H")),
				},
				{
					Key:        "item-key",
					Type:       productcatalog.FlatPriceType,
					Cadence:    "P1D",
					StartAfter: lo.ToPtr(isodate.MustParse(s.T(), "P1DT20H")),
				},
			},
			end:        s.mustParseTime("2021-01-03T00:00:00Z"),
			alignedSub: true,
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
					Start:             s.mustParseTime("2021-01-02T20:00:00Z"),
					End:               s.mustParseTime("2021-01-03T00:00:00Z"),
					Key:               "subID/phase-test/item-key/v[1]/period[0]",
					NonTruncatedStart: s.mustParseTime("2021-01-02T00:00:00Z"),
				},
			},
		},
		{
			name: "aligned one-time without cadence",
			items: []subscriptionItemViewMock{
				{
					Key:  "item-key",
					Type: productcatalog.FlatPriceType,
				},
			},
			end:        s.mustParseTime("2021-01-03T00:00:00Z"),
			alignedSub: true,
			expected: []expectedIterations{
				{
					Start: s.mustParseTime("2021-01-01T00:00:00Z"),
					End:   s.mustParseTime("2021-01-01T00:00:00Z"),
					Key:   "subID/phase-test/item-key/v[0]",
				},
			},
		},
		{
			name: "aligned one-time with cadence",
			items: []subscriptionItemViewMock{
				{
					Key:  "item-key",
					Type: productcatalog.FlatPriceType,
				},
				{
					Key:     "item-key2",
					Type:    productcatalog.FlatPriceType,
					Cadence: "P1D",
				},
			},
			end:        s.mustParseTime("2021-01-02T12:00:00Z"),
			alignedSub: true,
			expected: []expectedIterations{
				{
					Start:           s.mustParseTime("2021-01-01T00:00:00Z"),
					End:             s.mustParseTime("2021-01-01T00:00:00Z"),
					NonTruncatedEnd: s.mustParseTime("2021-01-02T00:00:00Z"),
					Key:             "subID/phase-test/item-key/v[0]",
				},
				{
					Start: s.mustParseTime("2021-01-01T00:00:00Z"),
					End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					Key:   "subID/phase-test/item-key2/v[0]/period[0]",
				},
				{
					Start: s.mustParseTime("2021-01-02T00:00:00Z"),
					End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					Key:   "subID/phase-test/item-key2/v[0]/period[1]",
				},
			},
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
				Spec: subscription.SubscriptionPhaseSpec{
					CreateSubscriptionPhasePlanInput: subscription.CreateSubscriptionPhasePlanInput{
						PhaseKey: "phase-test",
					},
					ItemsByKey: map[string][]*subscription.SubscriptionItemSpec{},
				},
			}

			for _, item := range tc.items {
				spec := subscription.SubscriptionItemSpec{
					CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
						CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
							ItemKey:  item.Key,
							PhaseKey: "phase-test",
						},
					},
				}
				view := subscription.SubscriptionItemView{}

				var pp *productcatalog.Price

				switch item.Type {
				case productcatalog.UnitPriceType:
					pp = productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromInt(1),
					})
				case productcatalog.FlatPriceType:
					pp = productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      alpacadecimal.NewFromInt(1),
						PaymentTerm: productcatalog.InAdvancePaymentTerm,
					})
				case NoPriceType:
					pp = nil
				default:
					pp = productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromInt(1),
					})
				}

				spec.RateCard.Price = pp
				view.SubscriptionItem.RateCard.Price = pp

				if item.Cadence != "" {
					bc := lo.ToPtr(isodate.MustParse(s.T(), item.Cadence))

					spec.RateCard.BillingCadence = bc
					view.SubscriptionItem.RateCard.BillingCadence = bc
				}

				if item.StartAfter != nil {
					af, _ := item.StartAfter.AddTo(phase.SubscriptionPhase.ActiveFrom)
					view.SubscriptionItem.ActiveFrom = af
					spec.ActiveFromOverrideRelativeToPhaseStart = item.StartAfter
				}

				if item.EndAfter != nil {
					at, _ := item.EndAfter.AddTo(phase.SubscriptionPhase.ActiveFrom)
					view.SubscriptionItem.ActiveTo = &at
					spec.ActiveToOverrideRelativeToPhaseStart = item.EndAfter
				}

				if view.SubscriptionItem.ActiveFrom.IsZero() {
					view.SubscriptionItem.ActiveFrom = phase.SubscriptionPhase.ActiveFrom
				}

				view.Spec = spec

				phase.ItemsByKey[view.Spec.ItemKey] = append(phase.ItemsByKey[view.Spec.ItemKey], view)
				phase.Spec.ItemsByKey[view.Spec.ItemKey] = append(phase.Spec.ItemsByKey[view.Spec.ItemKey], &spec)
			}

			subs := subscription.SubscriptionView{
				Subscription: subscription.Subscription{
					NamespacedID: models.NamespacedID{
						ID: "subID",
					},
				},
				Spec: subscription.SubscriptionSpec{
					CreateSubscriptionCustomerInput: subscription.CreateSubscriptionCustomerInput{
						ActiveFrom: phase.SubscriptionPhase.ActiveFrom,
					},
					Phases: map[string]*subscription.SubscriptionPhaseSpec{
						phase.SubscriptionPhase.Key: &phase.Spec,
					},
				},
				Phases: []subscription.SubscriptionPhaseView{phase},
			}

			if tc.alignedSub {
				subs.Subscription.BillablesMustAlign = true
				subs.Spec.BillablesMustAlign = true
			}

			if tc.phaseEnd != nil {
				subs.Spec.ActiveTo = tc.phaseEnd
				subs.Subscription.ActiveTo = tc.phaseEnd
				// Item activity is butched here
			}

			it, err := NewPhaseIterator(
				subs,
				phase.SubscriptionPhase.Key,
			)
			s.NoError(err)

			out, err := it.Generate(tc.end)

			if tc.expectedErr != nil {
				s.EqualError(err, tc.expectedErr.Error())
				return
			}

			s.NoError(err)

			outAsExpect := make([]expectedIterations, 0, len(out))
			for i, item := range out {
				nonTruncatedEnd := time.Time{}
				if !item.NonTruncatedPeriod.End.Equal(item.Period.End) {
					nonTruncatedEnd = item.NonTruncatedPeriod.End
				}

				nonTruncatedStart := time.Time{}
				if !item.NonTruncatedPeriod.Start.Equal(item.Period.Start) {
					nonTruncatedStart = item.NonTruncatedPeriod.Start
				}

				outAsExpect = append(outAsExpect, expectedIterations{
					Start:             item.Period.Start,
					End:               item.Period.End,
					Key:               item.UniqueID,
					NonTruncatedEnd:   nonTruncatedEnd,
					NonTruncatedStart: nonTruncatedStart,
				})

				s.T().Logf("out[%d]: [%s..%s] %s (non-truncated: %s..%s)\n", i, item.Period.Start, item.Period.End, item.UniqueID, nonTruncatedStart, nonTruncatedEnd)
			}

			for i, item := range tc.expected {
				s.T().Logf("expected[%d]: [%s..%s] %s (non-truncated: %s..%s)\n", i, item.Start, item.End, item.Key, item.NonTruncatedStart, item.NonTruncatedEnd)
			}

			s.ElementsMatch(tc.expected, outAsExpect)
		})
	}
}
