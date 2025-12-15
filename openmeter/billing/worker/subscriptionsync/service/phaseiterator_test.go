package service

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/datetime"
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
	ServicePeriod     billing.Period
	FullServicePeriod billing.Period
	BillingPeriod     billing.Period
	Key               string
}

type subscriptionItemViewMock struct {
	Key     string
	Cadence string

	StartAfter *datetime.ISODuration
	EndAfter   *datetime.ISODuration

	Type productcatalog.PriceType
}

const NoPriceType = productcatalog.PriceType("NoPrice")

func (s *PhaseIteratorTestSuite) mustParseTime(t string) time.Time {
	return lo.Must(time.Parse(time.RFC3339, t))
}

func (s *PhaseIteratorTestSuite) TestPhaseIterator() {
	tcs := []struct {
		name                                     string
		items                                    []subscriptionItemViewMock
		end                                      time.Time
		expected                                 []expectedIterations
		expectedErr                              error
		phaseEnd                                 *time.Time
		subscriptionEnd                          *time.Time
		alignedBillingCadence                    datetime.ISODuration
		billingAnchorRelativeToSubscriptionStart datetime.ISODuration // remember to use a negative otherwise test will fail
	}{
		{
			name:     "unaligned empty",
			items:    []subscriptionItemViewMock{},
			end:      s.mustParseTime("2021-01-01T00:00:00Z"),
			expected: []expectedIterations{},
		},
		{
			name:                  "aligned empty",
			items:                 []subscriptionItemViewMock{},
			alignedBillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
			end:                   s.mustParseTime("2021-01-01T00:00:00Z"),
			expected:              []expectedIterations{},
		},
		{
			name: "sanity",
			items: []subscriptionItemViewMock{
				{
					Key:     "item-key",
					Cadence: "P1D",
				},
			},
			end:                   s.mustParseTime("2021-01-03T00:00:00Z"),
			alignedBillingCadence: datetime.MustParseDuration(s.T(), "P1D"),
			expected: []expectedIterations{
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[0]",
				},
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[1]",
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
			end:                   s.mustParseTime("2021-01-03T00:00:00Z"),
			alignedBillingCadence: datetime.MustParseDuration(s.T(), "P1D"),
			expected: []expectedIterations{
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[0]",
				},
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[1]",
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
			end:                   s.mustParseTime("2021-01-03T00:00:00Z"),
			alignedBillingCadence: datetime.MustParseDuration(s.T(), "P1D"),
			expected: []expectedIterations{
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[0]",
				},
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-02T15:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-02T15:00:00Z"), // billing period can never reach over phases
					},
					Key: "subID/phase-test/item-key/v[0]/period[1]",
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
			end:                   s.mustParseTime("2021-01-04T00:00:00Z"),
			alignedBillingCadence: datetime.MustParseDuration(s.T(), "P1D"),
			expected: []expectedIterations{
				// 1d cadence
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key-1d/v[0]/period[0]",
				},
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key-1d/v[0]/period[1]",
				},
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-03T00:00:00Z"),
						End:   s.mustParseTime("2021-01-04T00:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-03T00:00:00Z"),
						End:   s.mustParseTime("2021-01-04T00:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-03T00:00:00Z"),
						End:   s.mustParseTime("2021-01-04T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key-1d/v[0]/period[2]",
				},
				// 2d cadence
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key-2d/v[0]/period[0]",
				},
			},
		},
		{
			// Note: this happens on subscription updates, but the active to/from is always disjunct
			name: "new-version-split-aligned-with-regular-cadence",
			items: []subscriptionItemViewMock{
				{
					Key:      "item-key",
					Cadence:  "P1D",
					EndAfter: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1D")),
				},
				{
					Key:        "item-key",
					Cadence:    "P1D",
					StartAfter: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1D")),
				},
			},
			alignedBillingCadence: datetime.MustParseDuration(s.T(), "P1D"),
			end:                   s.mustParseTime("2021-01-03T00:00:00Z"),
			expected: []expectedIterations{
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[0]",
				},
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[1]/period[0]",
				},
			},
		},
		{
			name: "active-from-to-missmatching-period",
			items: []subscriptionItemViewMock{
				{
					Key:      "item-key",
					Cadence:  "P1D",
					EndAfter: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1DT20H")),
				},
				{
					Key:        "item-key",
					Cadence:    "P1D",
					StartAfter: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1DT20H")),
				},
			},
			alignedBillingCadence: datetime.MustParseDuration(s.T(), "P1D"),
			end:                   s.mustParseTime("2021-01-03T00:00:00Z"),
			expected: []expectedIterations{
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[0]",
				},
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-02T20:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[1]",
				},
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T20:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[1]/period[0]",
				},
			},
		},
		{
			name: "ubp-time truncated",
			items: []subscriptionItemViewMock{
				{
					Key:      "item-key",
					Cadence:  "P1D",
					EndAfter: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1DT20H2S")),
					Type:     productcatalog.UnitPriceType,
				},
				{
					Key:        "item-key",
					Cadence:    "P1D",
					StartAfter: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1DT20H2S")), // empty period
					EndAfter:   lo.ToPtr(datetime.MustParseDuration(s.T(), "P1DT20H2S")),
					Type:       productcatalog.UnitPriceType,
				},
				{
					Key:        "item-key",
					Cadence:    "P1D",
					StartAfter: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1DT20H2S")),
					EndAfter:   lo.ToPtr(datetime.MustParseDuration(s.T(), "P1DT20H4S")),
					Type:       productcatalog.UnitPriceType,
				},
				{
					Key:        "item-key",
					Cadence:    "P1D",
					StartAfter: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1DT20H4S")),
					Type:       productcatalog.UnitPriceType,
				},
			},
			alignedBillingCadence: datetime.MustParseDuration(s.T(), "P1D"),
			end:                   s.mustParseTime("2021-01-03T00:00:00Z"),
			expected: []expectedIterations{
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[0]",
				},
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-02T20:00:02Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					// billing period will follow the otherwise cadence
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[1]",
				},
				// 0 length service period v1 is dropped
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T20:00:02Z"),
						End:   s.mustParseTime("2021-01-02T20:00:04Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					// billing period will follow the otherwise cadence
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[2]/period[0]",
				},
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T20:00:04Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[3]/period[0]",
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
			alignedBillingCadence: datetime.MustParseDuration(s.T(), "P1D"),
			end:                   s.mustParseTime("2021-01-03T00:00:00Z"),
			expected: []expectedIterations{
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[0]",
				},
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[1]",
				},
				{
					// Given end is >= invoice_at only at this point
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-03T00:00:00Z"),
						End:   s.mustParseTime("2021-01-04T00:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-03T00:00:00Z"),
						End:   s.mustParseTime("2021-01-04T00:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-03T00:00:00Z"),
						End:   s.mustParseTime("2021-01-04T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[2]",
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
			alignedBillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
			end:                   s.mustParseTime("2021-01-03T00:00:00Z"),
			// If PhaseEnd is defined, and before the otherwise end of the billing cadence, that should be the end of the periods for the one-time item
			phaseEnd: lo.ToPtr(s.mustParseTime("2021-01-05T00:00:00Z")),
			expected: []expectedIterations{
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-05T00:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-05T00:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-05T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]",
				},
			},
		},
		{
			name: "aligned in-advance and in-arrears recurring when billing cadence is same as service cadence",
			items: []subscriptionItemViewMock{
				// In Advance
				{
					Key:      "item-key",
					Type:     productcatalog.FlatPriceType,
					Cadence:  "P1D",
					EndAfter: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1DT20H")),
				},
				{
					Key:        "item-key",
					Type:       productcatalog.FlatPriceType,
					Cadence:    "P1D",
					StartAfter: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1DT20H")),
				},
				// In Arrears
				{
					Key:     "arrears-key",
					Type:    productcatalog.UnitPriceType,
					Cadence: "P1D",
				},
			},
			end:                   s.mustParseTime("2021-01-03T00:00:00Z"),
			alignedBillingCadence: datetime.MustParseDuration(s.T(), "P1D"),
			expected: []expectedIterations{
				// In Advance
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[0]",
				},
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-02T20:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[1]",
				},
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T20:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[1]/period[0]",
				},
				// As we want to generate all in advance items invoicable by end time, we get an extra in advance item
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-03T00:00:00Z"),
						End:   s.mustParseTime("2021-01-04T00:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-03T00:00:00Z"),
						End:   s.mustParseTime("2021-01-04T00:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-03T00:00:00Z"),
						End:   s.mustParseTime("2021-01-04T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[1]/period[1]",
				},
				// In Arrears
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					Key: "subID/phase-test/arrears-key/v[0]/period[0]",
				},
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					Key: "subID/phase-test/arrears-key/v[0]/period[1]",
				},
			},
		},
		{
			name: "aligned one-time no phase end",
			items: []subscriptionItemViewMock{
				{
					Key:  "item-key",
					Type: productcatalog.FlatPriceType,
				},
			},
			end:                   s.mustParseTime("2021-01-03T00:00:00Z"),
			alignedBillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
			expected: []expectedIterations{
				{
					// If there is no phase end, the service period will be an instant
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-01T00:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-01T00:00:00Z"),
					},
					// If there is no foreseeable end to the phase, we'll bill after a single iteration
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-02-01T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]",
				},
			},
		},
		{
			name: "aligned one-time with phase end",
			items: []subscriptionItemViewMock{
				{
					Key:  "item-key",
					Type: productcatalog.FlatPriceType,
				},
			},
			end:                   s.mustParseTime("2021-01-03T00:00:00Z"),
			phaseEnd:              lo.ToPtr(s.mustParseTime("2021-02-05T00:00:00Z")),
			alignedBillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
			expected: []expectedIterations{
				{
					// If there is a foreseeable phase end, the service period will account for the entire phase
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-02-05T00:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-02-05T00:00:00Z"),
					},
					// Billing period still follows the aligned cadence
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-02-01T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]",
				},
			},
		},
		{
			name: "aligned in-advance and in-arrears recurring with billing cadence different than service cadence",
			items: []subscriptionItemViewMock{
				{
					Key:     "item-key",
					Type:    productcatalog.FlatPriceType,
					Cadence: "P1D",
				},
				{
					Key:     "arrears-key",
					Type:    productcatalog.UnitPriceType,
					Cadence: "P1D",
				},
			},
			end:                   s.mustParseTime("2021-01-04T00:00:00Z"),
			alignedBillingCadence: datetime.MustParseDuration(s.T(), "P3D"),
			expected: []expectedIterations{
				// In Advance
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-04T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[0]",
				},
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-04T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[1]",
				},
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-03T00:00:00Z"),
						End:   s.mustParseTime("2021-01-04T00:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-03T00:00:00Z"),
						End:   s.mustParseTime("2021-01-04T00:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-04T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[2]",
				},
				// We will also generate for the next billing period the in advance items (as their invoice at will be next start = current end = iteration end)
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-04T00:00:00Z"),
						End:   s.mustParseTime("2021-01-05T00:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-04T00:00:00Z"),
						End:   s.mustParseTime("2021-01-05T00:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-04T00:00:00Z"),
						End:   s.mustParseTime("2021-01-07T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[3]",
				},
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-05T00:00:00Z"),
						End:   s.mustParseTime("2021-01-06T00:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-05T00:00:00Z"),
						End:   s.mustParseTime("2021-01-06T00:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-04T00:00:00Z"),
						End:   s.mustParseTime("2021-01-07T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[4]",
				},
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-06T00:00:00Z"),
						End:   s.mustParseTime("2021-01-07T00:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-06T00:00:00Z"),
						End:   s.mustParseTime("2021-01-07T00:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-04T00:00:00Z"),
						End:   s.mustParseTime("2021-01-07T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[5]",
				},
				// In Arrears
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-04T00:00:00Z"),
					},
					Key: "subID/phase-test/arrears-key/v[0]/period[0]",
				},
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-04T00:00:00Z"),
					},
					Key: "subID/phase-test/arrears-key/v[0]/period[1]",
				},
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-03T00:00:00Z"),
						End:   s.mustParseTime("2021-01-04T00:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-03T00:00:00Z"),
						End:   s.mustParseTime("2021-01-04T00:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-04T00:00:00Z"),
					},
					Key: "subID/phase-test/arrears-key/v[0]/period[2]",
				},
			},
		},
		{
			name:            "aligned subscription item is outside of subscription",
			subscriptionEnd: lo.ToPtr(s.mustParseTime("2021-01-03T00:00:00Z")),
			items: []subscriptionItemViewMock{
				{
					Key:        "item-key",
					Type:       productcatalog.FlatPriceType,
					Cadence:    "P1D",
					StartAfter: lo.ToPtr(datetime.MustParseDuration(s.T(), "P30D")),
				},
			},
			end:                   s.mustParseTime("2021-01-03T00:00:00Z"),
			alignedBillingCadence: datetime.MustParseDuration(s.T(), "P1D"),
			expected:              []expectedIterations{},
		},
		{
			name:                  "aligned subscription item crosses subs cancellation date (also phase end date)",
			subscriptionEnd:       lo.ToPtr(s.mustParseTime("2021-01-03T00:00:00Z")),
			end:                   s.mustParseTime("2021-01-03T00:00:00Z"),
			alignedBillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
			items: []subscriptionItemViewMock{
				{
					Key:     "item-key",
					Type:    productcatalog.FlatPriceType,
					Cadence: "P1M",
				},
			},
			expected: []expectedIterations{
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						// The full service period wasn't served, this will still be a month long
						End: s.mustParseTime("2021-02-01T00:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						// If the subscription ends, of course we can bill
						// Also, as otherwise, cadence cannot reach cross phase boundaries, which the subscription end is
						End: s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[0]",
				},
			},
		},
		//
		// Aligned Subscription Tests with Billing Anchor
		//
		{
			name: "aligned flat-fee recurring when billing cadence is same as service cadence",
			items: []subscriptionItemViewMock{
				{
					Key:      "item-key",
					Type:     productcatalog.FlatPriceType,
					Cadence:  "P1D",
					EndAfter: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1DT20H")),
				},
				{
					Key:        "item-key",
					Type:       productcatalog.FlatPriceType,
					Cadence:    "P1D",
					StartAfter: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1DT20H")),
				},
			},
			end:                                      s.mustParseTime("2021-01-03T00:00:00Z"),
			alignedBillingCadence:                    datetime.MustParseDuration(s.T(), "P1D"),
			billingAnchorRelativeToSubscriptionStart: datetime.MustParseDuration(s.T(), "-PT1H"),
			// We expect the full service periods to be shifted
			expected: []expectedIterations{
				{
					// Service Periods will be aligned to the billing anchor
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-01T23:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2020-12-31T23:00:00Z"),
						End:   s.mustParseTime("2021-01-01T23:00:00Z"),
					},
					BillingPeriod: billing.Period{
						// BillingPeriod cannot fall outside of subscription active period (and phase active period) so start gets truncated
						Start: s.mustParseTime("2021-01-01T00:00:00Z"),
						End:   s.mustParseTime("2021-01-01T23:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[0]",
				},
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T23:00:00Z"),
						// Item ends so does service period
						End: s.mustParseTime("2021-01-02T20:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T23:00:00Z"),
						End:   s.mustParseTime("2021-01-02T23:00:00Z"),
					},
					// Billing period will be the next whole day
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T23:00:00Z"),
						End:   s.mustParseTime("2021-01-02T23:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[1]",
				},
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T20:00:00Z"),
						End:   s.mustParseTime("2021-01-02T23:00:00Z"),
					},
					// Item was changed during period so the full service period will be the same as that of the previous version
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T23:00:00Z"),
						End:   s.mustParseTime("2021-01-02T23:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-01T23:00:00Z"),
						End:   s.mustParseTime("2021-01-02T23:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[1]/period[0]",
				},
				// Given invoiceAt should be >= end, we have an extra in advance item
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T23:00:00Z"),
						End:   s.mustParseTime("2021-01-03T23:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T23:00:00Z"),
						End:   s.mustParseTime("2021-01-03T23:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T23:00:00Z"),
						End:   s.mustParseTime("2021-01-03T23:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[1]/period[1]",
				},
			},
		},
	}

	for _, tc := range tcs {
		s.Run(tc.name, func() {
			subscriptionStart := s.mustParseTime("2021-01-01T00:00:00.1234Z")

			phase := subscription.SubscriptionPhaseView{
				SubscriptionPhase: subscription.SubscriptionPhase{
					ActiveFrom: subscriptionStart,
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
				var rc productcatalog.RateCard
				var bc *datetime.ISODuration

				if item.Cadence != "" {
					bc = lo.ToPtr(datetime.MustParseDuration(s.T(), item.Cadence))
				}

				switch item.Type {
				case productcatalog.UnitPriceType:
					pp = productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromInt(1),
					})
					rc = &productcatalog.UsageBasedRateCard{
						RateCardMeta: productcatalog.RateCardMeta{
							Price: pp,
						},
						BillingCadence: *bc,
					}
				case productcatalog.FlatPriceType:
					pp = productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      alpacadecimal.NewFromInt(1),
						PaymentTerm: productcatalog.InAdvancePaymentTerm,
					})
					rc = &productcatalog.FlatFeeRateCard{
						RateCardMeta: productcatalog.RateCardMeta{
							Price: pp,
						},
						BillingCadence: bc,
					}
				case NoPriceType:
					pp = nil
					rc = &productcatalog.FlatFeeRateCard{
						BillingCadence: bc,
					}
				default:
					pp = productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromInt(1),
					})
					rc = &productcatalog.FlatFeeRateCard{
						RateCardMeta: productcatalog.RateCardMeta{
							Price: pp,
						},
						BillingCadence: bc,
					}
				}

				spec.RateCard = rc
				view.SubscriptionItem.RateCard = rc

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

				if tc.phaseEnd != nil {
					if view.SubscriptionItem.ActiveTo != nil && tc.phaseEnd.Before(*view.SubscriptionItem.ActiveTo) {
						view.SubscriptionItem.ActiveTo = tc.phaseEnd
					} else {
						view.SubscriptionItem.ActiveTo = tc.phaseEnd
					}
				}

				if tc.subscriptionEnd != nil {
					if view.SubscriptionItem.ActiveTo != nil && tc.subscriptionEnd.Before(*view.SubscriptionItem.ActiveTo) {
						view.SubscriptionItem.ActiveTo = tc.subscriptionEnd
					} else {
						view.SubscriptionItem.ActiveTo = tc.subscriptionEnd
					}
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
					BillingAnchor: subscriptionStart,
					NamespacedID: models.NamespacedID{
						ID: "subID",
					},
				},
				Spec: subscription.SubscriptionSpec{
					CreateSubscriptionCustomerInput: subscription.CreateSubscriptionCustomerInput{
						ActiveFrom:    phase.SubscriptionPhase.ActiveFrom,
						BillingAnchor: subscriptionStart,
					},
					Phases: map[string]*subscription.SubscriptionPhaseSpec{
						phase.SubscriptionPhase.Key: &phase.Spec,
					},
				},
				Phases: []subscription.SubscriptionPhaseView{phase},
			}

			if tc.subscriptionEnd != nil {
				subs.Subscription.ActiveTo = tc.subscriptionEnd
				subs.Spec.ActiveTo = tc.subscriptionEnd
			}

			if tc.billingAnchorRelativeToSubscriptionStart.Sign() == 1 {
				require.Fail(s.T(), "billing anchor relative to subscription start must be negative in test-case")
			}

			if !tc.alignedBillingCadence.IsZero() {
				subs.Spec.BillingCadence = tc.alignedBillingCadence
				subs.Subscription.BillingCadence = tc.alignedBillingCadence
				anchorTime, ok := tc.billingAnchorRelativeToSubscriptionStart.AddTo(subscriptionStart)
				s.True(ok)

				subs.Subscription.BillingAnchor = anchorTime
				subs.Spec.BillingAnchor = anchorTime
			}

			if tc.phaseEnd != nil {
				subs.Spec.ActiveTo = tc.phaseEnd
				subs.Subscription.ActiveTo = tc.phaseEnd
			}

			it, err := NewPhaseIterator(
				slog.Default(),
				noop.NewTracerProvider().Tracer("test"),
				subs,
				phase.SubscriptionPhase.Key,
			)
			s.NoError(err)

			out, err := it.Generate(context.Background(), tc.end)

			if tc.expectedErr != nil {
				s.EqualError(err, tc.expectedErr.Error())
				return
			}

			s.NoError(err)

			outAsExpect := make([]expectedIterations, 0, len(out))
			for i, item := range out {
				outAsExpect = append(outAsExpect, expectedIterations{
					Key:               item.UniqueID,
					ServicePeriod:     item.ServicePeriod,
					FullServicePeriod: item.FullServicePeriod,
					BillingPeriod:     item.BillingPeriod,
				})

				s.T().Logf("out[%d]: %s \nService Period: [%s..%s] \nFull Service Period: [%s..%s] \nBilling Period: [%s..%s]\n", i, item.UniqueID, item.ServicePeriod.Start, item.ServicePeriod.End, item.FullServicePeriod.Start, item.FullServicePeriod.End, item.BillingPeriod.Start, item.BillingPeriod.End)
			}

			for i, item := range tc.expected {
				s.T().Logf("expected[%d]: %s \nService Period: [%s..%s] \nFull Service Period: [%s..%s] \nBilling Period: [%s..%s]\n", i, item.Key, item.ServicePeriod.Start, item.ServicePeriod.End, item.FullServicePeriod.Start, item.FullServicePeriod.End, item.BillingPeriod.Start, item.BillingPeriod.End)
			}

			s.ElementsMatch(tc.expected, outAsExpect)
		})
	}
}
