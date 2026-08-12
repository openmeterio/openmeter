package entitlementaccess

import (
	"github.com/oapi-codegen/nullable"
	"github.com/samber/lo"

	apiv3 "github.com/openmeterio/openmeter/api/v3"
	customershandler "github.com/openmeterio/openmeter/api/v3/handlers/customers"
	"github.com/openmeterio/openmeter/openmeter/entitlementaccess"
)

// ToAPIEntitlementAccessQueryResponse maps a domain QueryResult to the API response.
func ToAPIEntitlementAccessQueryResponse(res entitlementaccess.QueryResult, pageSize int) apiv3.EntitlementAccessQueryResponse {
	data := make([]apiv3.EntitlementAccessQueryResult, 0, len(res.Customers))

	for _, c := range res.Customers {
		features := make(map[string]apiv3.EntitlementFeatureAccess, len(c.Features))
		for key, fa := range c.Features {
			features[key] = toAPIFeatureAccess(fa)
		}

		data = append(data, apiv3.EntitlementAccessQueryResult{
			Matched:   c.Matched,
			Customer:  customershandler.ToAPIBillingCustomer(c.Customer),
			Features:  features,
			UpdatedAt: c.UpdatedAt,
		})
	}

	errs := make([]apiv3.EntitlementAccessQueryError, 0, len(res.Errors))

	for _, e := range res.Errors {
		errs = append(errs, apiv3.EntitlementAccessQueryError{
			Customer: lo.ToPtr(e.CustomerKey),
			Code:     toAPIQueryErrorCode(e.Code),
			Message:  e.Message,
		})
	}

	return apiv3.EntitlementAccessQueryResponse{
		Data:   data,
		Errors: errs,
		Meta:   toAPICursorMeta(res, pageSize),
	}
}

func toAPIFeatureAccess(fa entitlementaccess.FeatureAccess) apiv3.EntitlementFeatureAccess {
	out := apiv3.EntitlementFeatureAccess{HasAccess: fa.HasAccess}

	if fa.Reason != nil {
		out.Reason = &apiv3.EntitlementFeatureAccessReason{
			Code:    toAPIReasonCode(fa.Reason.Code),
			Message: fa.Reason.Message,
		}
	}

	return out
}

func toAPIReasonCode(code entitlementaccess.ReasonCode) apiv3.EntitlementFeatureAccessReasonCode {
	switch code {
	case entitlementaccess.ReasonCodeUsageLimitReached:
		return apiv3.EntitlementFeatureAccessReasonCodeUsageLimitReached
	case entitlementaccess.ReasonCodeFeatureUnavailable:
		return apiv3.EntitlementFeatureAccessReasonCodeFeatureUnavailable
	case entitlementaccess.ReasonCodeFeatureNotFound:
		return apiv3.EntitlementFeatureAccessReasonCodeFeatureNotFound
	case entitlementaccess.ReasonCodeNoCreditAvailable:
		return apiv3.EntitlementFeatureAccessReasonCodeNoCreditAvailable
	default:
		return apiv3.EntitlementFeatureAccessReasonCodeUnknown
	}
}

func toAPIQueryErrorCode(code entitlementaccess.QueryErrorCode) apiv3.EntitlementAccessQueryErrorCode {
	switch code {
	case entitlementaccess.QueryErrorCustomerNotFound:
		return apiv3.EntitlementAccessQueryErrorCodeCustomerNotFound
	default:
		return apiv3.EntitlementAccessQueryErrorCodeUnknown
	}
}

// toAPICursorMeta builds cursor pagination metadata from the domain result.
func toAPICursorMeta(res entitlementaccess.QueryResult, pageSize int) apiv3.CursorMeta {
	meta := apiv3.CursorMeta{
		Page: apiv3.CursorMetaPage{
			Next:     nullable.NewNullNullable[string](),
			Previous: nullable.NewNullNullable[string](),
			Size:     float32(pageSize),
		},
	}

	if res.First != nil {
		meta.Page.First = lo.ToPtr(res.First.Encode())
		if res.HasPrev {
			meta.Page.Previous = nullable.NewNullableWithValue(res.First.Encode())
		}
	}

	if res.Last != nil {
		meta.Page.Last = lo.ToPtr(res.Last.Encode())
		if res.HasNext {
			meta.Page.Next = nullable.NewNullableWithValue(res.Last.Encode())
		}
	}

	return meta
}
