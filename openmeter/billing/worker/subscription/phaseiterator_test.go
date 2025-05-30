package billingworkersubscription

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
	ServicePeriod     billing.Period
	FullServicePeriod billing.Period
	BillingPeriod     billing.Period
	Key               string
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
		name                  string
		items                 []subscriptionItemViewMock
		end                   time.Time
		expected              []expectedIterations
		expectedErr           error
		phaseEnd              *time.Time
		subscriptionEnd       *time.Time
		alignedBillingCadence isodate.Period
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
			alignedBillingCadence: isodate.MustParse(s.T(), "P1M"),
			end:                   s.mustParseTime("2021-01-01T00:00:00Z"),
			expected:              []expectedIterations{},
		},
		//
		// Non-Aligned Subscription Tests
		//
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
			end: s.mustParseTime("2021-01-03T00:00:00Z"),
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
			end: s.mustParseTime("2021-01-03T00:00:00Z"),
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
			end: s.mustParseTime("2021-01-04T00:00:00Z"),
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
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key-2d/v[0]/period[0]",
				},
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-03T00:00:00Z"),
						End:   s.mustParseTime("2021-01-05T00:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-03T00:00:00Z"),
						End:   s.mustParseTime("2021-01-05T00:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-03T00:00:00Z"),
						End:   s.mustParseTime("2021-01-05T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key-2d/v[0]/period[1]",
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
						End:   s.mustParseTime("2021-01-03T20:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T20:00:00Z"),
						End:   s.mustParseTime("2021-01-03T20:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T20:00:00Z"),
						End:   s.mustParseTime("2021-01-03T20:00:00Z"),
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
					// billing period will follow the otherwise cadence
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T00:00:00Z"),
						End:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[1]",
				},
				// 0 length service period items are dropped
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T20:00:00Z"),
						End:   s.mustParseTime("2021-01-03T20:00:00Z"),
					},
					// We only truncate the service period to the meter resolution
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T20:00:04Z"),
						End:   s.mustParseTime("2021-01-03T20:00:04Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T20:00:04Z"),
						End:   s.mustParseTime("2021-01-03T20:00:04Z"),
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
			end: s.mustParseTime("2021-01-03T00:00:00Z"),
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
			end: s.mustParseTime("2021-01-03T00:00:00Z"),
			// If PhaseEnd is defined, we that should be the end of the periods for the one-time item
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
						End:   s.mustParseTime("2021-01-03T20:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T20:00:00Z"),
						End:   s.mustParseTime("2021-01-03T20:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-02T20:00:00Z"),
						End:   s.mustParseTime("2021-01-03T20:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[1]/period[0]",
				},
				{
					ServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-03T20:00:00Z"),
						End:   s.mustParseTime("2021-01-04T20:00:00Z"),
					},
					FullServicePeriod: billing.Period{
						Start: s.mustParseTime("2021-01-03T20:00:00Z"),
						End:   s.mustParseTime("2021-01-04T20:00:00Z"),
					},
					BillingPeriod: billing.Period{
						Start: s.mustParseTime("2021-01-03T20:00:00Z"),
						End:   s.mustParseTime("2021-01-04T20:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[1]/period[1]",
				},
			},
		},
		//
		// Aligned Subscription Tests
		//
		{
			name: "aligned flat-fee recurring when billing cadence is same as service cadence",
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
			end:                   s.mustParseTime("2021-01-03T00:00:00Z"),
			alignedBillingCadence: isodate.MustParse(s.T(), "P1D"),
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
				// Given invoiceAt should be >= end, we have an extra in advance item
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
			alignedBillingCadence: isodate.MustParse(s.T(), "P1M"),
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
			alignedBillingCadence: isodate.MustParse(s.T(), "P1M"),
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
			name: "aligned flat fee recurring with billing cadence different than service cadence",
			items: []subscriptionItemViewMock{
				{
					Key:     "item-key",
					Type:    productcatalog.FlatPriceType,
					Cadence: "P1D",
				},
			},
			end:                   s.mustParseTime("2021-01-02T12:00:00Z"),
			alignedBillingCadence: isodate.MustParse(s.T(), "P3D"),
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
					StartAfter: lo.ToPtr(isodate.MustParse(s.T(), "P30D")),
				},
			},
			end:                   s.mustParseTime("2021-01-03T00:00:00Z"),
			alignedBillingCadence: isodate.MustParse(s.T(), "P1D"),
			expected:              []expectedIterations{},
		},
		{
			name:                  "aligned subscription item crosses subs cancellation date (also phase end date)",
			subscriptionEnd:       lo.ToPtr(s.mustParseTime("2021-01-03T00:00:00Z")),
			end:                   s.mustParseTime("2021-01-03T00:00:00Z"),
			alignedBillingCadence: isodate.MustParse(s.T(), "P1M"),
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
				var rc productcatalog.RateCard
				var bc *isodate.Period

				if item.Cadence != "" {
					bc = lo.ToPtr(isodate.MustParse(s.T(), item.Cadence))
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

			if tc.subscriptionEnd != nil {
				subs.Subscription.ActiveTo = tc.subscriptionEnd
				subs.Spec.ActiveTo = tc.subscriptionEnd
			}

			if !tc.alignedBillingCadence.IsZero() {
				subs.Subscription.BillablesMustAlign = true
				subs.Spec.BillablesMustAlign = true
				subs.Spec.BillingCadence = tc.alignedBillingCadence
				subs.Subscription.BillingCadence = tc.alignedBillingCadence
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

				s.T().Logf("out[%d]: [%s..%s] %s (full-service: %s..%s) (billing: %s..%s)\n", i, item.ServicePeriod.Start, item.ServicePeriod.End, item.UniqueID, item.FullServicePeriod.Start, item.FullServicePeriod.End, item.BillingPeriod.Start, item.BillingPeriod.End)
			}

			for i, item := range tc.expected {
				s.T().Logf("expected[%d]: [%s..%s] %s (full-service: %s..%s) (billing: %s..%s)\n", i, item.ServicePeriod.Start, item.ServicePeriod.End, item.Key, item.FullServicePeriod.Start, item.FullServicePeriod.End, item.BillingPeriod.Start, item.BillingPeriod.End)
			}

			s.ElementsMatch(tc.expected, outAsExpect)
		})
	}
}
