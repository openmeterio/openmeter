package customerscredits

import (
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"net/http"
	"net/url"
	"testing"

	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/filters"
	"github.com/openmeterio/openmeter/openmeter/ledger"
)

func TestCustomerCreditPlanFilterQuery(t *testing.T) {
	for _, tc := range []struct {
		query   string
		want    mo.Option[*ledger.PlanFilter]
		invalid bool
	}{
		{query: ""},
		{query: "filter[plan_key]=pro", want: mo.Some(&ledger.PlanFilter{Key: "pro"})},
		{query: "filter[plan_key][oeq]=pro", want: mo.Some(&ledger.PlanFilter{Key: "pro"})},
		{query: "filter[plan_key]=pro&filter[plan_version]=2", want: mo.Some(&ledger.PlanFilter{Key: "pro", Version: &ledger.VersionFilter{Eq: lo.ToPtr(2)}})},
		{query: "filter[plan_key][exists]=false", want: mo.Some[*ledger.PlanFilter](nil)},
		{query: "filter[plan_key][exists]=true", invalid: true},
		{query: "filter[plan_key][neq]=pro", invalid: true},
		{query: "filter[plan_key][contains]=pro", invalid: true},
		{query: "filter[plan_key][oeq]=pro,enterprise", invalid: true},
		{query: "filter[plan_key][eq]=", invalid: true},
		{query: "filter[plan_key]=pro&filter[plan_version][gte]=2", invalid: true},
		{query: "filter[plan_key]=pro&filter[plan_version]=0", invalid: true},
		{query: "filter[plan_key]=pro&filter[plan_version]=-1", invalid: true},
		{query: "filter[plan_key]=pro&filter[plan_version]=1.5", invalid: true},
		{query: "filter[plan_key]=pro&filter[plan_version]=2147483648", invalid: true},
		{query: "filter[plan_version]=2", invalid: true},
		{query: "filter[plan_key][exists]=false&filter[plan_version]=2", invalid: true},
	} {
		t.Run(tc.query, func(t *testing.T) {
			values, err := url.ParseQuery(tc.query)
			require.NoError(t, err)
			for _, parse := range []func() (mo.Option[*ledger.PlanFilter], error){
				func() (mo.Option[*ledger.PlanFilter], error) {
					var params api.GetCreditBalanceParamsFilter
					if err := filters.Parse(values, &params); err != nil {
						return mo.None[*ledger.PlanFilter](), err
					}
					return fromAPICustomerCreditPlanFilter(t.Context(), params.PlanKey, params.PlanVersion)
				},
				func() (mo.Option[*ledger.PlanFilter], error) {
					var params api.ListCreditTransactionsParamsFilter
					if err := filters.Parse(values, &params); err != nil {
						return mo.None[*ledger.PlanFilter](), err
					}
					return fromAPICustomerCreditPlanFilter(t.Context(), params.PlanKey, params.PlanVersion)
				},
			} {
				got, err := parse()
				if tc.invalid {
					require.Error(t, err)
					continue
				}
				require.NoError(t, err)
				require.Equal(t, tc.want, got)
			}
		})
	}
}

func TestCustomerCreditPlanFilterErrorField(t *testing.T) {
	for _, tc := range []struct {
		name    string
		key     *api.StringFieldFilter
		version *api.NumericFieldFilter
		field   string
	}{
		{name: "key", key: &api.StringFieldFilter{Contains: lo.ToPtr("pro")}, field: "filter[plan_key]"},
		{name: "empty key", key: &api.StringFieldFilter{Eq: lo.ToPtr("")}, field: "filter[plan_key]"},
		{name: "version", key: &api.StringFieldFilter{Eq: lo.ToPtr("pro")}, version: &api.NumericFieldFilter{Eq: lo.ToPtr(0.0)}, field: "filter[plan_version]"},
		{name: "version without key", version: &api.NumericFieldFilter{Eq: lo.ToPtr(2.0)}, field: "filter[plan_version]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := fromAPICustomerCreditPlanFilter(t.Context(), tc.key, tc.version)
			var apiErr *apierrors.BaseAPIError
			require.ErrorAs(t, err, &apiErr)
			require.Equal(t, http.StatusBadRequest, apiErr.Status)
			require.Len(t, apiErr.InvalidParameters, 1)
			require.Equal(t, tc.field, apiErr.InvalidParameters[0].Field)
			require.Equal(t, apierrors.InvalidParamSourceQuery, apiErr.InvalidParameters[0].Source)
		})
	}
}
