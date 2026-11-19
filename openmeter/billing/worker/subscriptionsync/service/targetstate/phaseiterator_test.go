package targetstate

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

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
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

func TestFlatFeeInvoiceAtUsesDefaultPaymentTerm(t *testing.T) {
	billingStart := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	serviceEnd := time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC)
	billingEnd := time.Date(2026, time.February, 5, 0, 0, 0, 0, time.UTC)

	for _, tc := range []struct {
		name        string
		paymentTerm productcatalog.PaymentTermType
		want        time.Time
	}{
		{name: "omitted term", want: billingStart},
		{name: "explicit default", paymentTerm: productcatalog.InAdvancePaymentTerm, want: billingStart},
		{name: "in arrears", paymentTerm: productcatalog.InArrearsPaymentTerm, want: billingEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			item := SubscriptionItemWithPeriods{
				ServicePeriod: timeutil.ClosedPeriod{From: billingStart, To: serviceEnd},
				BillingPeriod: timeutil.ClosedPeriod{From: billingStart, To: billingEnd},
			}
			item.Spec.RateCard = &productcatalog.FlatFeeRateCard{
				RateCardMeta: productcatalog.RateCardMeta{
					Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount: alpacadecimal.NewFromInt(10), PaymentTerm: tc.paymentTerm,
					}),
				},
			}

			require.Equal(t, tc.want, item.GetInvoiceAt())
		})
	}
}

func TestStartBillingAt(t *testing.T) {
	day := func(year int, month time.Month, day int) time.Time {
		return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	}

	period := func(from, to time.Time) timeutil.ClosedPeriod {
		return timeutil.ClosedPeriod{From: from, To: to}
	}

	monthly := datetime.MustParseDuration(t, "P1M")

	// A subscription billed monthly on the first of the month.
	december := period(day(2026, time.December, 1), day(2027, time.January, 1))
	insideDecember := day(2026, time.December, 17)

	type expected struct {
		billed        bool
		servicePeriod timeutil.ClosedPeriod
		billingPeriod timeutil.ClosedPeriod
		invoiceAt     time.Time
	}

	for _, tc := range []struct {
		name string
		// rateCardCadence is nil for a one-time fee.
		rateCardCadence   *datetime.ISODuration
		servicePeriod     timeutil.ClosedPeriod
		fullServicePeriod timeutil.ClosedPeriod
		billingPeriod     timeutil.ClosedPeriod
		billingStartsAt   time.Time
		expected          expected
	}{
		{
			name:              "service starting at the billing start is billed as is",
			rateCardCadence:   &monthly,
			servicePeriod:     december,
			fullServicePeriod: december,
			billingPeriod:     december,
			billingStartsAt:   day(2026, time.December, 1),
			expected:          expected{billed: true, servicePeriod: december, billingPeriod: december, invoiceAt: december.From},
		},
		{
			name:              "service ending at the billing start is not billed",
			rateCardCadence:   &monthly,
			servicePeriod:     period(day(2026, time.November, 1), day(2026, time.December, 1)),
			fullServicePeriod: period(day(2026, time.November, 1), day(2026, time.December, 1)),
			billingPeriod:     period(day(2026, time.November, 1), day(2026, time.December, 1)),
			billingStartsAt:   day(2026, time.December, 1),
		},
		{
			name:              "service spanning the billing start is billed from there",
			rateCardCadence:   &monthly,
			servicePeriod:     december,
			fullServicePeriod: december,
			billingPeriod:     december,
			billingStartsAt:   insideDecember,
			expected: expected{
				billed:        true,
				servicePeriod: period(insideDecember, december.To),
				billingPeriod: period(insideDecember, december.To),
				invoiceAt:     insideDecember,
			},
		},
		{
			// A yearly fee on monthly billing: its own billing period is the month it started in.
			name:              "a longer rate card cadence moves to the billing period of the billing start",
			rateCardCadence:   lo.ToPtr(datetime.MustParseDuration(t, "P1Y")),
			servicePeriod:     period(day(2026, time.November, 1), day(2027, time.January, 1)),
			fullServicePeriod: period(day(2026, time.January, 1), day(2027, time.January, 1)),
			billingPeriod:     period(day(2026, time.November, 1), day(2026, time.December, 1)),
			billingStartsAt:   day(2026, time.December, 1),
			expected: expected{
				billed:        true,
				servicePeriod: december,
				billingPeriod: december,
				invoiceAt:     december.From,
			},
		},
		{
			// A daily fee on monthly billing, in the month where billing starts.
			name:              "later service in the first billing period is invoiced no earlier than the billing start",
			rateCardCadence:   lo.ToPtr(datetime.MustParseDuration(t, "P1D")),
			servicePeriod:     period(day(2026, time.December, 20), day(2026, time.December, 21)),
			fullServicePeriod: period(day(2026, time.December, 20), day(2026, time.December, 21)),
			billingPeriod:     december,
			billingStartsAt:   insideDecember,
			expected: expected{
				billed:        true,
				servicePeriod: period(day(2026, time.December, 20), day(2026, time.December, 21)),
				billingPeriod: period(insideDecember, december.To),
				invoiceAt:     insideDecember,
			},
		},
		{
			name:              "a one-time fee that started earlier is not billed",
			servicePeriod:     period(day(2026, time.November, 1), day(2027, time.March, 1)),
			fullServicePeriod: period(day(2026, time.November, 1), day(2027, time.March, 1)),
			billingPeriod:     period(day(2026, time.November, 1), day(2026, time.December, 1)),
			billingStartsAt:   day(2026, time.December, 1),
		},
		{
			name:              "a one-time fee at the billing start is billed",
			servicePeriod:     period(day(2026, time.December, 1), day(2026, time.December, 1)),
			fullServicePeriod: period(day(2026, time.December, 1), day(2026, time.December, 1)),
			billingPeriod:     december,
			billingStartsAt:   day(2026, time.December, 1),
			expected: expected{
				billed:        true,
				servicePeriod: period(day(2026, time.December, 1), day(2026, time.December, 1)),
				billingPeriod: december,
				invoiceAt:     december.From,
			},
		},
		{
			// Item periods have 1s resolution, so the billing start is compared at that resolution too.
			name:              "a billing start with sub-second precision",
			rateCardCadence:   &monthly,
			servicePeriod:     december,
			fullServicePeriod: december,
			billingPeriod:     december,
			billingStartsAt:   december.From.Add(500 * time.Millisecond),
			expected:          expected{billed: true, servicePeriod: december, billingPeriod: december, invoiceAt: december.From},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			item := SubscriptionItemWithPeriods{
				UniqueID:          "subID/phase/item/v[0]/period[1]",
				PeriodIndex:       1,
				ServicePeriod:     tc.servicePeriod,
				FullServicePeriod: tc.fullServicePeriod,
				BillingPeriod:     tc.billingPeriod,
			}
			item.Spec.RateCard = &productcatalog.FlatFeeRateCard{
				RateCardMeta: productcatalog.RateCardMeta{
					Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount: alpacadecimal.NewFromInt(10), PaymentTerm: productcatalog.InAdvancePaymentTerm,
					}),
				},
				BillingCadence: tc.rateCardCadence,
			}

			got, billed := item.startBillingAt(tc.billingStartsAt, december)

			require.Equal(t, tc.expected.billed, billed)
			if !billed {
				return
			}

			require.Equal(t, tc.expected.servicePeriod, got.ServicePeriod)
			require.Equal(t, tc.expected.billingPeriod, got.BillingPeriod)
			require.Equal(t, tc.expected.invoiceAt, got.GetInvoiceAt())

			// The full service period is what proration is measured against, and the identity
			// must stay stable across syncs.
			require.Equal(t, tc.fullServicePeriod, got.FullServicePeriod)
			require.Equal(t, item.UniqueID, got.UniqueID)
			require.Equal(t, item.PeriodIndex, got.PeriodIndex)
		})
	}
}

type expectedIterations struct {
	ServicePeriod     timeutil.ClosedPeriod
	FullServicePeriod timeutil.ClosedPeriod
	BillingPeriod     timeutil.ClosedPeriod
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
		customerDeletedAt                        *time.Time
		alignedBillingCadence                    datetime.ISODuration
		billingAnchorRelativeToSubscriptionStart datetime.ISODuration // remember to use a negative otherwise test will fail
	}{
		{
			name: "customer deletion truncates recurring flat fee service",
			items: []subscriptionItemViewMock{{
				Key:     "item-key",
				Cadence: "P1M",
				Type:    productcatalog.FlatPriceType,
			}},
			end:                   s.mustParseTime("2021-01-20T00:00:00Z"),
			customerDeletedAt:     lo.ToPtr(s.mustParseTime("2021-01-15T00:00:00Z")),
			alignedBillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
			expected: []expectedIterations{{
				Key:               "subID/phase-test/item-key/v[0]/period[0]",
				ServicePeriod:     timeutil.ClosedPeriod{From: s.mustParseTime("2021-01-01T00:00:00Z"), To: s.mustParseTime("2021-01-15T00:00:00Z")},
				FullServicePeriod: timeutil.ClosedPeriod{From: s.mustParseTime("2021-01-01T00:00:00Z"), To: s.mustParseTime("2021-02-01T00:00:00Z")},
				BillingPeriod:     timeutil.ClosedPeriod{From: s.mustParseTime("2021-01-01T00:00:00Z"), To: s.mustParseTime("2021-01-15T00:00:00Z")},
			}},
		},
		{
			name: "customer deletion truncates recurring usage service",
			items: []subscriptionItemViewMock{{
				Key:     "item-key",
				Cadence: "P1M",
				Type:    productcatalog.UnitPriceType,
			}},
			end:                   s.mustParseTime("2021-01-20T00:00:00Z"),
			customerDeletedAt:     lo.ToPtr(s.mustParseTime("2021-01-15T00:00:00Z")),
			alignedBillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
			expected: []expectedIterations{{
				Key:               "subID/phase-test/item-key/v[0]/period[0]",
				ServicePeriod:     timeutil.ClosedPeriod{From: s.mustParseTime("2021-01-01T00:00:00Z"), To: s.mustParseTime("2021-01-15T00:00:00Z")},
				FullServicePeriod: timeutil.ClosedPeriod{From: s.mustParseTime("2021-01-01T00:00:00Z"), To: s.mustParseTime("2021-02-01T00:00:00Z")},
				BillingPeriod:     timeutil.ClosedPeriod{From: s.mustParseTime("2021-01-01T00:00:00Z"), To: s.mustParseTime("2021-01-15T00:00:00Z")},
			}},
		},
		{
			name: "item ending before customer deletion keeps its service end",
			items: []subscriptionItemViewMock{{
				Key:      "item-key",
				Cadence:  "P1M",
				Type:     productcatalog.FlatPriceType,
				EndAfter: lo.ToPtr(datetime.MustParseDuration(s.T(), "P9D")),
			}},
			end:                   s.mustParseTime("2021-01-20T00:00:00Z"),
			customerDeletedAt:     lo.ToPtr(s.mustParseTime("2021-01-15T00:00:00Z")),
			alignedBillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
			expected: []expectedIterations{{
				Key:               "subID/phase-test/item-key/v[0]/period[0]",
				ServicePeriod:     timeutil.ClosedPeriod{From: s.mustParseTime("2021-01-01T00:00:00Z"), To: s.mustParseTime("2021-01-10T00:00:00Z")},
				FullServicePeriod: timeutil.ClosedPeriod{From: s.mustParseTime("2021-01-01T00:00:00Z"), To: s.mustParseTime("2021-02-01T00:00:00Z")},
				BillingPeriod:     timeutil.ClosedPeriod{From: s.mustParseTime("2021-01-01T00:00:00Z"), To: s.mustParseTime("2021-01-15T00:00:00Z")},
			}},
		},
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
					ServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					FullServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					BillingPeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[0]",
				},
				{
					ServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					FullServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					BillingPeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
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
					ServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					FullServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					BillingPeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[0]",
				},
				{
					ServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					FullServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					BillingPeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
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
					ServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					FullServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					BillingPeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[0]",
				},
				{
					ServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-02T15:00:00Z"),
					},
					FullServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					BillingPeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-02T15:00:00Z"), // billing period can never reach over phases
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
					ServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					FullServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					BillingPeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key-1d/v[0]/period[0]",
				},
				{
					ServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					FullServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					BillingPeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key-1d/v[0]/period[1]",
				},
				{
					ServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-03T00:00:00Z"),
						To:   s.mustParseTime("2021-01-04T00:00:00Z"),
					},
					FullServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-03T00:00:00Z"),
						To:   s.mustParseTime("2021-01-04T00:00:00Z"),
					},
					BillingPeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-03T00:00:00Z"),
						To:   s.mustParseTime("2021-01-04T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key-1d/v[0]/period[2]",
				},
				// 2d cadence
				{
					ServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					FullServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					BillingPeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-02T00:00:00Z"),
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
					ServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					FullServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					BillingPeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[0]",
				},
				{
					ServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					FullServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					BillingPeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
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
					ServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					FullServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					BillingPeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[0]",
				},
				{
					ServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-02T20:00:00Z"),
					},
					FullServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					BillingPeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[1]",
				},
				{
					ServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T20:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					FullServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					BillingPeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
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
					ServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					FullServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					BillingPeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[0]",
				},
				{
					ServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-02T20:00:02Z"),
					},
					FullServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					// billing period will follow the otherwise cadence
					BillingPeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[1]",
				},
				// 0 length service period v1 is dropped
				{
					ServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T20:00:02Z"),
						To:   s.mustParseTime("2021-01-02T20:00:04Z"),
					},
					FullServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					// billing period will follow the otherwise cadence
					BillingPeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[2]/period[0]",
				},
				{
					ServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T20:00:04Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					FullServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					BillingPeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
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
					ServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					FullServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					BillingPeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[0]",
				},
				{
					ServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					FullServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					BillingPeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[1]",
				},
				// The period starting at end is invoiceable at end, but its service belongs to the next iteration.
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
					ServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-05T00:00:00Z"),
					},
					FullServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-05T00:00:00Z"),
					},
					BillingPeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-05T00:00:00Z"),
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
					ServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					FullServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					BillingPeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[0]",
				},
				{
					ServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-02T20:00:00Z"),
					},
					FullServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					BillingPeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[1]",
				},
				{
					ServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T20:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					FullServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					BillingPeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[1]/period[0]",
				},
				// The in advance item starting at end is invoiceable at end, but its service belongs to the next iteration.
				// In Arrears
				{
					ServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					FullServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					BillingPeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					Key: "subID/phase-test/arrears-key/v[0]/period[0]",
				},
				{
					ServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					FullServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					BillingPeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
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
					ServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-01T00:00:00Z"),
					},
					FullServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-01T00:00:00Z"),
					},
					// If there is no foreseeable end to the phase, we'll bill after a single iteration
					BillingPeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-02-01T00:00:00Z"),
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
					ServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-02-05T00:00:00Z"),
					},
					FullServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-02-05T00:00:00Z"),
					},
					// Billing period still follows the aligned cadence
					BillingPeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-02-01T00:00:00Z"),
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
					ServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					FullServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					BillingPeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-04T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[0]",
				},
				{
					ServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					FullServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					BillingPeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-04T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[1]",
				},
				{
					ServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-03T00:00:00Z"),
						To:   s.mustParseTime("2021-01-04T00:00:00Z"),
					},
					FullServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-03T00:00:00Z"),
						To:   s.mustParseTime("2021-01-04T00:00:00Z"),
					},
					BillingPeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-04T00:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[2]",
				},
				// The next billing period's in advance items are invoiceable at end, but their service belongs to the next iteration.
				// In Arrears
				{
					ServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					FullServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-02T00:00:00Z"),
					},
					BillingPeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-04T00:00:00Z"),
					},
					Key: "subID/phase-test/arrears-key/v[0]/period[0]",
				},
				{
					ServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					FullServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T00:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					BillingPeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-04T00:00:00Z"),
					},
					Key: "subID/phase-test/arrears-key/v[0]/period[1]",
				},
				{
					ServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-03T00:00:00Z"),
						To:   s.mustParseTime("2021-01-04T00:00:00Z"),
					},
					FullServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-03T00:00:00Z"),
						To:   s.mustParseTime("2021-01-04T00:00:00Z"),
					},
					BillingPeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-04T00:00:00Z"),
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
					ServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-03T00:00:00Z"),
					},
					FullServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						// The full service period wasn't served, this will still be a month long
						To: s.mustParseTime("2021-02-01T00:00:00Z"),
					},
					BillingPeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						// If the subscription ends, of course we can bill
						// Also, as otherwise, cadence cannot reach cross phase boundaries, which the subscription end is
						To: s.mustParseTime("2021-01-03T00:00:00Z"),
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
					ServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-01T23:00:00Z"),
					},
					FullServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2020-12-31T23:00:00Z"),
						To:   s.mustParseTime("2021-01-01T23:00:00Z"),
					},
					BillingPeriod: timeutil.ClosedPeriod{
						// BillingPeriod cannot fall outside of subscription active period (and phase active period) so start gets truncated
						From: s.mustParseTime("2021-01-01T00:00:00Z"),
						To:   s.mustParseTime("2021-01-01T23:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[0]",
				},
				{
					ServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T23:00:00Z"),
						// Item ends so does service period
						To: s.mustParseTime("2021-01-02T20:00:00Z"),
					},
					FullServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T23:00:00Z"),
						To:   s.mustParseTime("2021-01-02T23:00:00Z"),
					},
					// Billing period will be the next whole day
					BillingPeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T23:00:00Z"),
						To:   s.mustParseTime("2021-01-02T23:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[0]/period[1]",
				},
				{
					ServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T20:00:00Z"),
						To:   s.mustParseTime("2021-01-02T23:00:00Z"),
					},
					// Item was changed during period so the full service period will be the same as that of the previous version
					FullServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T23:00:00Z"),
						To:   s.mustParseTime("2021-01-02T23:00:00Z"),
					},
					BillingPeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-01T23:00:00Z"),
						To:   s.mustParseTime("2021-01-02T23:00:00Z"),
					},
					Key: "subID/phase-test/item-key/v[1]/period[0]",
				},
				// Given invoiceAt should be >= end, we have an extra in advance item
				{
					ServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T23:00:00Z"),
						To:   s.mustParseTime("2021-01-03T23:00:00Z"),
					},
					FullServicePeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T23:00:00Z"),
						To:   s.mustParseTime("2021-01-03T23:00:00Z"),
					},
					BillingPeriod: timeutil.ClosedPeriod{
						From: s.mustParseTime("2021-01-02T23:00:00Z"),
						To:   s.mustParseTime("2021-01-03T23:00:00Z"),
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

			if tc.customerDeletedAt != nil {
				// The target builder caps subscription dates for customer deletion;
				// the stored item dates still describe the original subscription.
				subs = withActiveTo(subs, *tc.customerDeletedAt)
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

				s.T().Logf("out[%d]: %s \nService Period: [%s..%s] \nFull Service Period: [%s..%s] \nBilling Period: [%s..%s]\n", i, item.UniqueID, item.ServicePeriod.From, item.ServicePeriod.To, item.FullServicePeriod.From, item.FullServicePeriod.To, item.BillingPeriod.From, item.BillingPeriod.To)
			}

			for i, item := range tc.expected {
				s.T().Logf("expected[%d]: %s \nService Period: [%s..%s] \nFull Service Period: [%s..%s] \nBilling Period: [%s..%s]\n", i, item.Key, item.ServicePeriod.From, item.ServicePeriod.To, item.FullServicePeriod.From, item.FullServicePeriod.To, item.BillingPeriod.From, item.BillingPeriod.To)
			}

			s.ElementsMatch(tc.expected, outAsExpect)
		})
	}
}
