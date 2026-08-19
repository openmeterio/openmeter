package charges

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// The service-period window is half-open ([from, to)): ServicePeriodFrom
// supports only gte, ServicePeriodTo supports only lt, and both filters are
// optional and validated independently.
func TestListChargesInputValidateServicePeriodFilters(t *testing.T) {
	at := time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC)

	validInput := ListChargesInput{
		Page:      pagination.NewPage(1, 10),
		Namespace: "namespace",
	}

	tests := []struct {
		name    string
		mutate  func(*ListChargesInput)
		wantErr string
	}{
		{
			name: "both filters omitted",
		},
		{
			name: "from with gte only",
			mutate: func(input *ListChargesInput) {
				input.ServicePeriodFrom = &filter.FilterTime{Gte: lo.ToPtr(at)}
			},
		},
		{
			name: "to with lt only",
			mutate: func(input *ListChargesInput) {
				input.ServicePeriodTo = &filter.FilterTime{Lt: lo.ToPtr(at)}
			},
		},
		{
			name: "combined window",
			mutate: func(input *ListChargesInput) {
				input.ServicePeriodFrom = &filter.FilterTime{Gte: lo.ToPtr(at)}
				input.ServicePeriodTo = &filter.FilterTime{Lt: lo.ToPtr(at.AddDate(0, 1, 0))}
			},
		},
		{
			name: "from without any operator",
			mutate: func(input *ListChargesInput) {
				input.ServicePeriodFrom = &filter.FilterTime{}
			},
			wantErr: "service period from filter supports only the gte operator",
		},
		{
			name: "from with an unsupported operator",
			mutate: func(input *ListChargesInput) {
				input.ServicePeriodFrom = &filter.FilterTime{Lte: lo.ToPtr(at)}
			},
			wantErr: "service period from filter supports only the gte operator",
		},
		{
			name: "from with an unsupported operator next to gte",
			mutate: func(input *ListChargesInput) {
				input.ServicePeriodFrom = &filter.FilterTime{Gte: lo.ToPtr(at), Eq: lo.ToPtr(at)}
			},
			wantErr: "service period from filter: validation error: filter is invalid: multiple operators are set",
		},
		{
			name: "from with a composite operator",
			mutate: func(input *ListChargesInput) {
				input.ServicePeriodFrom = &filter.FilterTime{And: &[]filter.FilterTime{{Gte: lo.ToPtr(at)}}}
			},
			wantErr: "service period from filter supports only the gte operator",
		},
		{
			name: "to without any operator",
			mutate: func(input *ListChargesInput) {
				input.ServicePeriodTo = &filter.FilterTime{}
			},
			wantErr: "service period to filter supports only the lt operator",
		},
		{
			name: "to with an unsupported operator",
			mutate: func(input *ListChargesInput) {
				input.ServicePeriodTo = &filter.FilterTime{Lte: lo.ToPtr(at)}
			},
			wantErr: "service period to filter supports only the lt operator",
		},
		{
			name: "to with an unsupported operator next to lt",
			mutate: func(input *ListChargesInput) {
				input.ServicePeriodTo = &filter.FilterTime{Lt: lo.ToPtr(at), Exists: lo.ToPtr(true)}
			},
			wantErr: "service period to filter: validation error: filter is invalid: multiple operators are set",
		},
		{
			name: "to with a composite operator",
			mutate: func(input *ListChargesInput) {
				input.ServicePeriodTo = &filter.FilterTime{Or: &[]filter.FilterTime{{Lt: lo.ToPtr(at)}}}
			},
			wantErr: "service period to filter supports only the lt operator",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := validInput
			if tc.mutate != nil {
				tc.mutate(&input)
			}

			err := input.Validate()
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
