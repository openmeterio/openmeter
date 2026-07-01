package billingservice

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestLimitGatheringLinesForInvoice(t *testing.T) {
	line := func(id, from, to string) gatheringLineWithBillablePeriod {
		gatheringLine := billing.GatheringLine{}
		gatheringLine.ID = id
		gatheringLine.ServicePeriod = timeutil.ClosedPeriod{
			From: mustTime(t, from),
			To:   mustTime(t, to),
		}

		return gatheringLineWithBillablePeriod{
			Line:           gatheringLine,
			BillablePeriod: gatheringLine.ServicePeriod,
		}
	}

	lines := []gatheringLineWithBillablePeriod{
		line("later", "2025-03-01T00:00:00Z", "2025-04-01T00:00:00Z"),
		line("tie-b", "2025-01-01T00:00:00Z", "2025-02-01T00:00:00Z"),
		line("earliest", "2024-12-01T00:00:00Z", "2025-01-01T00:00:00Z"),
		line("tie-a", "2025-01-01T00:00:00Z", "2025-02-01T00:00:00Z"),
	}

	t.Run("zero keeps all lines without reordering", func(t *testing.T) {
		got := limitGatheringLinesForInvoice(lines, 0)

		require.Equal(t, []gatheringLineWithBillablePeriod(lines), got)
	})

	t.Run("positive limit keeps earliest service periods", func(t *testing.T) {
		got := limitGatheringLinesForInvoice(lines, 3)

		require.Equal(t, []string{"earliest", "tie-a", "tie-b"}, gatheringLineIDsForLimitTest(got))
	})
}

func TestResolvePendingLineCollectionCutoff(t *testing.T) {
	asOf := mustTime(t, "2025-06-15T12:00:00Z")
	anchor := mustTime(t, "2025-06-01T00:00:00Z")
	monthly := datetime.MustParseDuration(t, "P1M")

	tests := []struct {
		name       string
		opts       []billing.InvoicePendingLinesOption
		collection billing.CollectionConfig
		asOf       time.Time
		want       time.Time
		wantErr    string
	}{
		{
			name: "bypassing alignment returns as of unchanged for subscription alignment",
			opts: []billing.InvoicePendingLinesOption{
				billing.WithBypassCollectionAlignment(),
			},
			collection: billing.CollectionConfig{
				Alignment: billing.AlignmentKindSubscription,
				Interval:  datetime.MustParseDuration(t, "PT1H"),
			},
			asOf: asOf,
			want: asOf,
		},
		{
			name: "bypassing alignment returns as of unchanged for anchored alignment",
			opts: []billing.InvoicePendingLinesOption{
				billing.WithBypassCollectionAlignment(),
			},
			collection: billing.CollectionConfig{
				Alignment: billing.AlignmentKindAnchored,
				AnchoredAlignmentDetail: &billing.AnchoredAlignmentDetail{
					Interval: monthly,
					Anchor:   anchor,
				},
				Interval: datetime.MustParseDuration(t, "PT1H"),
			},
			asOf: asOf,
			want: asOf,
		},
		{
			name: "subscription alignment returns as of unchanged",
			collection: billing.CollectionConfig{
				Alignment: billing.AlignmentKindSubscription,
				Interval:  datetime.MustParseDuration(t, "PT1H"),
			},
			asOf: asOf,
			want: asOf,
		},
		{
			name: "anchored alignment returns previous anchor before as of",
			collection: billing.CollectionConfig{
				Alignment: billing.AlignmentKindAnchored,
				AnchoredAlignmentDetail: &billing.AnchoredAlignmentDetail{
					Interval: monthly,
					Anchor:   anchor,
				},
				Interval: datetime.MustParseDuration(t, "PT1H"),
			},
			asOf: mustTime(t, "2025-06-15T12:00:00Z"),
			want: mustTime(t, "2025-06-01T00:00:00Z"),
		},
		{
			name: "anchored alignment returns exact anchor when as of lands on anchor",
			collection: billing.CollectionConfig{
				Alignment: billing.AlignmentKindAnchored,
				AnchoredAlignmentDetail: &billing.AnchoredAlignmentDetail{
					Interval: monthly,
					Anchor:   anchor,
				},
				Interval: datetime.MustParseDuration(t, "PT1H"),
			},
			asOf: mustTime(t, "2025-07-01T00:00:00Z"),
			want: mustTime(t, "2025-07-01T00:00:00Z"),
		},
		{
			name: "anchored alignment can walk backwards from future anchor",
			collection: billing.CollectionConfig{
				Alignment: billing.AlignmentKindAnchored,
				AnchoredAlignmentDetail: &billing.AnchoredAlignmentDetail{
					Interval: monthly,
					Anchor:   mustTime(t, "2025-08-01T00:00:00Z"),
				},
				Interval: datetime.MustParseDuration(t, "PT1H"),
			},
			asOf: mustTime(t, "2025-06-15T12:00:00Z"),
			want: mustTime(t, "2025-06-01T00:00:00Z"),
		},
		{
			name: "anchored alignment errors without detail",
			collection: billing.CollectionConfig{
				Alignment: billing.AlignmentKindAnchored,
				Interval:  datetime.MustParseDuration(t, "PT1H"),
			},
			asOf:    asOf,
			wantErr: "anchored alignment detail is required",
		},
		{
			name: "anchored alignment errors for invalid recurrence interval",
			collection: billing.CollectionConfig{
				Alignment: billing.AlignmentKindAnchored,
				AnchoredAlignmentDetail: &billing.AnchoredAlignmentDetail{
					Interval: datetime.ISODuration{},
					Anchor:   anchor,
				},
				Interval: datetime.MustParseDuration(t, "PT1H"),
			},
			asOf:    asOf,
			wantErr: "creating anchored alignment recurrence",
		},
		{
			name: "errors for unsupported alignment",
			collection: billing.CollectionConfig{
				Alignment: billing.AlignmentKind("invalid"),
				Interval:  datetime.MustParseDuration(t, "PT1H"),
			},
			asOf:    asOf,
			wantErr: "unsupported collection alignment: invalid",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := resolvePendingLineCollectionCutoff(billing.NewInvoicePendingLinesOptions(test.opts...), test.collection, test.asOf)

			if test.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.wantErr)
				require.True(t, got.IsZero())
				return
			}

			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}
}

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()

	parsed, err := time.Parse(time.RFC3339, value)
	require.NoError(t, err)

	return parsed
}

func gatheringLineIDsForLimitTest(lines []gatheringLineWithBillablePeriod) []string {
	ids := make([]string, 0, len(lines))
	for _, line := range lines {
		ids = append(ids, line.Line.ID)
	}

	return ids
}
