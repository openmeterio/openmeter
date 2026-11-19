package charges

import (
	"errors"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	billingcharges "github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

func assertBadRequestField(t *testing.T, err error, field string) {
	t.Helper()

	var apiErr *apierrors.BaseAPIError
	require.True(t, errors.As(err, &apiErr), "expected *apierrors.BaseAPIError, got %T", err)
	require.Len(t, apiErr.InvalidParameters, 1)
	require.Equal(t, field, apiErr.InvalidParameters[0].Field)
}

func TestFromAPIListChargesParams(t *testing.T) {
	ctx := t.Context()

	t.Run("defaults apply without query parameters", func(t *testing.T) {
		req, err := fromAPIListChargesParams(ctx, "ns", nil, nil, nil)
		require.NoError(t, err)

		require.Equal(t, "ns", req.Namespace)
		require.Equal(t, 1, req.Page.PageNumber)
		require.Equal(t, 20, req.Page.PageSize)
		require.Empty(t, req.CustomerIDs)
		require.Nil(t, req.CustomerID)
		require.Equal(t, []meta.ChargeType{meta.ChargeTypeFlatFee, meta.ChargeTypeUsageBased}, req.ChargeTypes)
		require.Empty(t, req.OrderBy)
		require.False(t, req.Expands.Has(meta.ExpandCustomer))
	})

	t.Run("page size above the cap is rejected", func(t *testing.T) {
		_, err := fromAPIListChargesParams(ctx, "ns", &apiv3models.PagePaginationQuery{Size: lo.ToPtr(maxListChargesPageSize + 1)}, nil, nil)
		assertBadRequestField(t, err, "page[size]")
	})

	t.Run("sort maps to the service order", func(t *testing.T) {
		sort := apiv3models.SortQuery("service_period.from desc")
		req, err := fromAPIListChargesParams(ctx, "ns", nil, &sort, nil)
		require.NoError(t, err)

		require.Equal(t, "service_period.from", req.OrderBy)
		require.Equal(t, sortx.OrderDesc, req.Order)
	})

	t.Run("unsupported sort field is rejected", func(t *testing.T) {
		sort := apiv3models.SortQuery("name")
		_, err := fromAPIListChargesParams(ctx, "ns", nil, &sort, nil)
		require.Error(t, err)
	})

	t.Run("expands map to domain expands", func(t *testing.T) {
		req, err := fromAPIListChargesParams(ctx, "ns", nil, nil, &[]apiv3models.BillingChargesExpand{apiv3models.BillingChargesExpandCustomer, apiv3models.BillingChargesExpandRealizationTotals})
		require.NoError(t, err)

		require.True(t, req.Expands.Has(meta.ExpandCustomer))
		require.True(t, req.Expands.Has(meta.ExpandRealizationTotals))
		require.False(t, req.Expands.Has(meta.ExpandFeature))
	})

	t.Run("unknown expand is rejected", func(t *testing.T) {
		_, err := fromAPIListChargesParams(ctx, "ns", nil, nil, &[]apiv3models.BillingChargesExpand{"invoice"})
		assertBadRequestField(t, err, "expand")
	})
}

func TestFromAPIListChargesParamsFilter(t *testing.T) {
	ctx := t.Context()
	const customerID = "01ARZ3NDEKTSV4RRFFQ69G5FAV"

	t.Run("customer id supports the full operator set", func(t *testing.T) {
		var req billingcharges.ListCustomerChargesInput
		err := fromAPIListChargesParamsFilter(ctx, &apiv3models.ListChargesParamsFilter{
			CustomerId: &apiv3models.ULIDFieldFilter{Eq: lo.ToPtr(customerID)},
		}, &req)
		require.NoError(t, err)
		require.Equal(t, customerID, lo.FromPtr(req.CustomerID.Eq))

		req = billingcharges.ListCustomerChargesInput{}
		err = fromAPIListChargesParamsFilter(ctx, &apiv3models.ListChargesParamsFilter{
			CustomerId: &apiv3models.ULIDFieldFilter{Oeq: []string{customerID}},
		}, &req)
		require.NoError(t, err)
		require.Equal(t, []string{customerID}, lo.FromPtr(req.CustomerID.In))

		req = billingcharges.ListCustomerChargesInput{}
		err = fromAPIListChargesParamsFilter(ctx, &apiv3models.ListChargesParamsFilter{
			CustomerId: &apiv3models.ULIDFieldFilter{Neq: lo.ToPtr(customerID)},
		}, &req)
		require.NoError(t, err)
		require.Equal(t, customerID, lo.FromPtr(req.CustomerID.Ne))
	})

	t.Run("subscription id maps to the subscription filter", func(t *testing.T) {
		var req billingcharges.ListCustomerChargesInput
		subscriptionIDs := []string{"01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW"}
		require.NoError(t, fromAPIListChargesParamsFilter(ctx, &apiv3models.ListChargesParamsFilter{
			SubscriptionId: &apiv3models.ULIDFieldFilter{Oeq: subscriptionIDs},
		}, &req))
		require.NotNil(t, req.SubscriptionID)
		require.Equal(t, subscriptionIDs, lo.FromPtr(req.SubscriptionID.In))
	})

	t.Run("selecting deleted status lifts the deleted guard", func(t *testing.T) {
		deleted := string(meta.ChargeStatusDeleted)

		var req billingcharges.ListCustomerChargesInput
		require.NoError(t, fromAPIListChargesParamsFilter(ctx, &apiv3models.ListChargesParamsFilter{
			Status: &apiv3models.StringFieldFilterExact{Eq: lo.ToPtr(deleted)},
		}, &req))
		require.True(t, req.IncludeDeleted)

		req = billingcharges.ListCustomerChargesInput{}
		require.NoError(t, fromAPIListChargesParamsFilter(ctx, &apiv3models.ListChargesParamsFilter{
			Status: &apiv3models.StringFieldFilterExact{Oeq: []string{"active", deleted}},
		}, &req))
		require.True(t, req.IncludeDeleted)

		req = billingcharges.ListCustomerChargesInput{}
		require.NoError(t, fromAPIListChargesParamsFilter(ctx, &apiv3models.ListChargesParamsFilter{
			Status: &apiv3models.StringFieldFilterExact{Neq: lo.ToPtr(deleted)},
		}, &req))
		require.False(t, req.IncludeDeleted)
		require.NotNil(t, req.Status)
	})

	t.Run("unknown status values are rejected", func(t *testing.T) {
		var req billingcharges.ListCustomerChargesInput
		err := fromAPIListChargesParamsFilter(ctx, &apiv3models.ListChargesParamsFilter{
			Status: &apiv3models.StringFieldFilterExact{Oeq: []string{"active", "unknown"}},
		}, &req)
		assertBadRequestField(t, err, "filter[status]")

		err = fromAPIListChargesParamsFilter(ctx, &apiv3models.ListChargesParamsFilter{
			Status: &apiv3models.StringFieldFilterExact{Neq: lo.ToPtr("bogus")},
		}, &req)
		assertBadRequestField(t, err, "filter[status]")
	})

	t.Run("feature and service period filters map onto the request", func(t *testing.T) {
		var req billingcharges.ListCustomerChargesInput
		require.NoError(t, fromAPIListChargesParamsFilter(ctx, &apiv3models.ListChargesParamsFilter{
			FeatureId:  &apiv3models.ULIDFieldFilter{Eq: lo.ToPtr(customerID)},
			FeatureKey: &apiv3models.StringFieldFilterExact{Eq: lo.ToPtr("api_requests")},
		}, &req))
		require.Equal(t, customerID, lo.FromPtr(req.FeatureID.Eq))
		require.Equal(t, "api_requests", lo.FromPtr(req.FeatureKey.Eq))
		require.Nil(t, req.ServicePeriodFrom)
		require.Nil(t, req.ServicePeriodTo)
	})
}
