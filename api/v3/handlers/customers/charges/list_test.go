package charges

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/api/v3/filters"
	"github.com/openmeterio/openmeter/pkg/filter"
)

// The service-period window is half-open — [from, to) — so the wire contract
// only supports gte on service_period_from and lt on service_period_to; both
// filters are optional and validated independently of each other.
func TestConvertAPIServicePeriodFilters(t *testing.T) {
	at := time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC)

	t.Run("service_period_from", func(t *testing.T) {
		tests := []struct {
			name        string
			in          *filters.FilterDateTime
			expected    *filter.FilterTime
			expectedErr string
		}{
			{name: "nil filter is skipped", in: nil},
			{name: "empty filter is skipped", in: &filters.FilterDateTime{}},
			{
				name:     "gte is supported",
				in:       &filters.FilterDateTime{Gte: lo.ToPtr(at)},
				expected: &filter.FilterTime{Gte: lo.ToPtr(at)},
			},
			{
				name:        "eq is rejected",
				in:          &filters.FilterDateTime{Eq: lo.ToPtr(at)},
				expectedErr: "supports only the gte operator",
			},
			{
				name:        "gt is rejected",
				in:          &filters.FilterDateTime{Gt: lo.ToPtr(at)},
				expectedErr: "supports only the gte operator",
			},
			{
				name:        "lt is rejected",
				in:          &filters.FilterDateTime{Lt: lo.ToPtr(at)},
				expectedErr: "supports only the gte operator",
			},
			{
				name:        "lte is rejected",
				in:          &filters.FilterDateTime{Lte: lo.ToPtr(at)},
				expectedErr: "supports only the gte operator",
			},
			{
				name:        "an unsupported operator next to gte is still rejected",
				in:          &filters.FilterDateTime{Gte: lo.ToPtr(at), Lte: lo.ToPtr(at)},
				expectedErr: "supports only the gte operator",
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				out, err := filters.FromAPIFilterDateTime(tc.in)
				if tc.expectedErr != "" {
					require.ErrorContains(t, err, tc.expectedErr)
					return
				}

				require.NoError(t, err)
				assert.Equal(t, tc.expected, out)
			})
		}
	})

	t.Run("service_period_to", func(t *testing.T) {
		tests := []struct {
			name        string
			in          *filters.FilterDateTime
			expected    *filter.FilterTime
			expectedErr string
		}{
			{name: "nil filter is skipped", in: nil},
			{name: "empty filter is skipped", in: &filters.FilterDateTime{}},
			{
				name:     "lt is supported",
				in:       &filters.FilterDateTime{Lt: lo.ToPtr(at)},
				expected: &filter.FilterTime{Lt: lo.ToPtr(at)},
			},
			{
				name:        "eq is rejected",
				in:          &filters.FilterDateTime{Eq: lo.ToPtr(at)},
				expectedErr: "supports only the lt operator",
			},
			{
				name:        "gt is rejected",
				in:          &filters.FilterDateTime{Gt: lo.ToPtr(at)},
				expectedErr: "supports only the lt operator",
			},
			{
				name:        "gte is rejected",
				in:          &filters.FilterDateTime{Gte: lo.ToPtr(at)},
				expectedErr: "supports only the lt operator",
			},
			{
				name:        "lte is rejected",
				in:          &filters.FilterDateTime{Lte: lo.ToPtr(at)},
				expectedErr: "supports only the lt operator",
			},
			{
				name:        "an unsupported operator next to lt is still rejected",
				in:          &filters.FilterDateTime{Lt: lo.ToPtr(at), Gte: lo.ToPtr(at)},
				expectedErr: "supports only the lt operator",
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				out, err := filters.FromAPIFilterDateTime(tc.in)
				if tc.expectedErr != "" {
					require.ErrorContains(t, err, tc.expectedErr)
					return
				}

				require.NoError(t, err)
				assert.Equal(t, tc.expected, out)
			})
		}
	})
}
