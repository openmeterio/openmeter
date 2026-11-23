package entitlementaccess

import (
	"github.com/samber/lo"

	customershandler "github.com/openmeterio/openmeter/api/v3/handlers/customers"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/entitlementaccess"
)

// ToAPIEntitlementAccessQueryResponse maps a domain QueryResult to the API response.
func ToAPIEntitlementAccessQueryResponse(res entitlementaccess.QueryResult, pageSize int) apiv3models.EntitlementAccessQueryResponse {
	data := make([]apiv3models.EntitlementAccessQueryResult, 0, len(res.Customers))

	for _, c := range res.Customers {
		features := make(map[string]apiv3models.EntitlementFeatureAccess, len(c.Features))
		for key, fa := range c.Features {
			features[key] = toAPIFeatureAccess(fa)
		}

		data = append(data, apiv3models.EntitlementAccessQueryResult{
			Matched:   c.Matched,
			Customer:  customershandler.ToAPIBillingCustomer(c.Customer),
			Features:  features,
			UpdatedAt: c.UpdatedAt,
		})
	}

	errs := make([]apiv3models.EntitlementAccessQueryError, 0, len(res.Errors))

	for _, e := range res.Errors {
		errs = append(errs, apiv3models.EntitlementAccessQueryError{
			Customer: lo.ToPtr(e.CustomerKey),
			Code:     toAPIQueryErrorCode(e.Code),
			Message:  e.Message,
		})
	}

	return apiv3models.EntitlementAccessQueryResponse{
		Data:   data,
		Errors: errs,
		Meta:   toAPICursorMeta(res, pageSize),
	}
}

func toAPIFeatureAccess(fa entitlementaccess.FeatureAccess) apiv3models.EntitlementFeatureAccess {
	out := apiv3models.EntitlementFeatureAccess{HasAccess: fa.HasAccess}

	if fa.Reason != nil {
		out.Reason = &apiv3models.EntitlementFeatureAccessReason{
			Code:    toAPIReasonCode(fa.Reason.Code),
			Message: fa.Reason.Message,
		}
	}

	return out
}

func toAPIReasonCode(code entitlementaccess.ReasonCode) apiv3models.EntitlementFeatureAccessReasonCode {
	switch code {
	case entitlementaccess.ReasonCodeUsageLimitReached:
		return apiv3models.EntitlementFeatureAccessReasonCodeUsageLimitReached
	case entitlementaccess.ReasonCodeFeatureUnavailable:
		return apiv3models.EntitlementFeatureAccessReasonCodeFeatureUnavailable
	case entitlementaccess.ReasonCodeFeatureNotFound:
		return apiv3models.EntitlementFeatureAccessReasonCodeFeatureNotFound
	case entitlementaccess.ReasonCodeNoCreditAvailable:
		return apiv3models.EntitlementFeatureAccessReasonCodeNoCreditAvailable
	default:
		return apiv3models.EntitlementFeatureAccessReasonCodeUnknown
	}
}

func toAPIQueryErrorCode(code entitlementaccess.QueryErrorCode) apiv3models.EntitlementAccessQueryErrorCode {
	switch code {
	case entitlementaccess.QueryErrorCustomerNotFound:
		return apiv3models.EntitlementAccessQueryErrorCodeCustomerNotFound
	default:
		return apiv3models.EntitlementAccessQueryErrorCodeUnknown
	}
}

// toAPICursorMeta builds cursor pagination metadata from the domain result.
func toAPICursorMeta(res entitlementaccess.QueryResult, pageSize int) apiv3models.CursorMeta {
	edges := response.CursorPageEdges{
		First: res.First.EncodePtr(),
		Last:  res.Last.EncodePtr(),
	}

	if res.HasPrev {
		edges.Previous = edges.First
	}

	if res.HasNext {
		edges.Next = edges.Last
	}

	return response.NewCursorMeta(pageSize, edges)
}
