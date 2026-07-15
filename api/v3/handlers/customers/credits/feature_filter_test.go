package customerscredits

import (
	"testing"

	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
)

func TestFromAPICustomerCreditFeatureFilter(t *testing.T) {
	tests := []struct {
		name    string
		filter  *api.StringFieldFilter
		want    mo.Option[creditpurchase.FeatureFilters]
		wantErr bool
	}{
		{
			name: "omitted filter returns all",
			want: customerbalance.AllFeatureFilter(),
		},
		{
			name:   "eq returns feature filter",
			filter: &api.StringFieldFilter{Eq: lo.ToPtr("feature-a")},
			want:   customerbalance.NewFeatureFilter([]string{"feature-a"}),
		},
		{
			name:   "single oeq returns feature filter",
			filter: &api.StringFieldFilter{Oeq: []string{"feature-a"}},
			want:   customerbalance.NewFeatureFilter([]string{"feature-a"}),
		},
		{
			name:    "multiple features are rejected",
			filter:  &api.StringFieldFilter{Oeq: []string{"feature-a", "feature-b"}},
			wantErr: true,
		},
		{
			name:   "exists false returns unrestricted",
			filter: &api.StringFieldFilter{Exists: lo.ToPtr(false)},
			want:   customerbalance.NewUnrestrictedFeatureFilter(),
		},
		{
			name:    "exists true is rejected",
			filter:  &api.StringFieldFilter{Exists: lo.ToPtr(true)},
			wantErr: true,
		},
		{
			name:    "contains is rejected",
			filter:  &api.StringFieldFilter{Contains: lo.ToPtr("feature")},
			wantErr: true,
		},
		{
			name:    "neq is rejected",
			filter:  &api.StringFieldFilter{Neq: lo.ToPtr("feature-a")},
			wantErr: true,
		},
		{
			name:    "ocontains is rejected",
			filter:  &api.StringFieldFilter{Ocontains: []string{"feature"}},
			wantErr: true,
		},
		{
			name:    "gt is rejected",
			filter:  &api.StringFieldFilter{Gt: lo.ToPtr("feature-a")},
			wantErr: true,
		},
		{
			name:    "gte is rejected",
			filter:  &api.StringFieldFilter{Gte: lo.ToPtr("feature-a")},
			wantErr: true,
		},
		{
			name:    "lt is rejected",
			filter:  &api.StringFieldFilter{Lt: lo.ToPtr("feature-a")},
			wantErr: true,
		},
		{
			name:    "lte is rejected",
			filter:  &api.StringFieldFilter{Lte: lo.ToPtr("feature-a")},
			wantErr: true,
		},
		{
			name:    "empty feature is rejected",
			filter:  &api.StringFieldFilter{Eq: lo.ToPtr("")},
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

func TestFromAPICreditGrantFeatureKeyFilter(t *testing.T) {
	tests := []struct {
		name    string
		filter  *api.StringFieldFilter
		want    *creditpurchase.FeatureKeyFilter
		wantErr bool
	}{
		{
			name: "omitted filter returns nil",
		},
		{
			name:   "eq returns a single key",
			filter: &api.StringFieldFilter{Eq: lo.ToPtr("feature-a")},
			want:   &creditpurchase.FeatureKeyFilter{In: []string{"feature-a"}},
		},
		{
			name:   "oeq returns multiple keys",
			filter: &api.StringFieldFilter{Oeq: []string{"feature-a", "feature-b"}},
			want:   &creditpurchase.FeatureKeyFilter{In: []string{"feature-a", "feature-b"}},
		},
		{
			name:   "eq and oeq combine",
			filter: &api.StringFieldFilter{Eq: lo.ToPtr("feature-a"), Oeq: []string{"feature-b"}},
			want:   &creditpurchase.FeatureKeyFilter{In: []string{"feature-a", "feature-b"}},
		},
		{
			name:   "exists false selects unrestricted grants",
			filter: &api.StringFieldFilter{Exists: lo.ToPtr(false)},
			want:   &creditpurchase.FeatureKeyFilter{Exists: lo.ToPtr(false)},
		},
		{
			name:   "exists true selects restricted grants",
			filter: &api.StringFieldFilter{Exists: lo.ToPtr(true)},
			want:   &creditpurchase.FeatureKeyFilter{Exists: lo.ToPtr(true)},
		},
		{
			name:    "neq is rejected",
			filter:  &api.StringFieldFilter{Neq: lo.ToPtr("feature-a")},
			wantErr: true,
		},
		{
			name:    "contains is rejected",
			filter:  &api.StringFieldFilter{Contains: lo.ToPtr("feature")},
			wantErr: true,
		},
		{
			name:    "ocontains is rejected",
			filter:  &api.StringFieldFilter{Ocontains: []string{"feature"}},
			wantErr: true,
		},
		{
			name:    "gt is rejected",
			filter:  &api.StringFieldFilter{Gt: lo.ToPtr("feature-a")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fromAPICreditGrantFeatureKeyFilter(tt.filter)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
