package customerscredits

import (
	"testing"

	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
)

func TestFromAPICustomerCreditFeatureFilter(t *testing.T) {
	tests := []struct {
		name    string
		filter  *apiv3models.StringFieldFilter
		want    mo.Option[creditpurchase.FeatureFilters]
		wantErr bool
	}{
		{
			name: "omitted filter returns all",
			want: customerbalance.AllFeatureFilter(),
		},
		{
			name:   "eq returns feature filter",
			filter: &apiv3models.StringFieldFilter{Eq: lo.ToPtr("feature-a")},
			want:   customerbalance.NewFeatureFilter([]string{"feature-a"}),
		},
		{
			name:   "single oeq returns feature filter",
			filter: &apiv3models.StringFieldFilter{Oeq: []string{"feature-a"}},
			want:   customerbalance.NewFeatureFilter([]string{"feature-a"}),
		},
		{
			name:    "multiple features are rejected",
			filter:  &apiv3models.StringFieldFilter{Oeq: []string{"feature-a", "feature-b"}},
			wantErr: true,
		},
		{
			name:   "exists false returns unrestricted",
			filter: &apiv3models.StringFieldFilter{Exists: lo.ToPtr(false)},
			want:   customerbalance.NewUnrestrictedFeatureFilter(),
		},
		{
			name:    "exists true is rejected",
			filter:  &apiv3models.StringFieldFilter{Exists: lo.ToPtr(true)},
			wantErr: true,
		},
		{
			name:    "contains is rejected",
			filter:  &apiv3models.StringFieldFilter{Contains: lo.ToPtr("feature")},
			wantErr: true,
		},
		{
			name:    "neq is rejected",
			filter:  &apiv3models.StringFieldFilter{Neq: lo.ToPtr("feature-a")},
			wantErr: true,
		},
		{
			name:    "ocontains is rejected",
			filter:  &apiv3models.StringFieldFilter{Ocontains: []string{"feature"}},
			wantErr: true,
		},
		{
			name:    "gt is rejected",
			filter:  &apiv3models.StringFieldFilter{Gt: lo.ToPtr("feature-a")},
			wantErr: true,
		},
		{
			name:    "gte is rejected",
			filter:  &apiv3models.StringFieldFilter{Gte: lo.ToPtr("feature-a")},
			wantErr: true,
		},
		{
			name:    "lt is rejected",
			filter:  &apiv3models.StringFieldFilter{Lt: lo.ToPtr("feature-a")},
			wantErr: true,
		},
		{
			name:    "lte is rejected",
			filter:  &apiv3models.StringFieldFilter{Lte: lo.ToPtr("feature-a")},
			wantErr: true,
		},
		{
			name:    "empty feature is rejected",
			filter:  &apiv3models.StringFieldFilter{Eq: lo.ToPtr("")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fromAPICustomerCreditFeatureFilter(tt.filter)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
