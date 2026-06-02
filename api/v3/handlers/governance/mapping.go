package governance

import (
	"github.com/oapi-codegen/nullable"
	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	customershandler "github.com/openmeterio/openmeter/api/v3/handlers/customers"
	"github.com/openmeterio/openmeter/openmeter/governance"
)

// ToAPIGovernanceQueryResponse maps a domain QueryResult to the API response.
func ToAPIGovernanceQueryResponse(res governance.QueryResult, pageSize int) api.GovernanceQueryResponse {
	data := make([]api.GovernanceQueryResult, 0, len(res.Customers))
	for _, c := range res.Customers {
		features := make(map[string]api.GovernanceFeatureAccess, len(c.Features))
		for key, fa := range c.Features {
			features[key] = toAPIFeatureAccess(fa)
		}

		data = append(data, api.GovernanceQueryResult{
			Matched:   c.Matched,
			Customer:  customershandler.ToAPIBillingCustomer(c.Customer),
			Features:  features,
			UpdatedAt: c.UpdatedAt,
		})
	}

	errs := make([]api.GovernanceQueryError, 0, len(res.Errors))
	for _, e := range res.Errors {
		errs = append(errs, api.GovernanceQueryError{
			Customer: lo.ToPtr(e.CustomerKey),
			Code:     toAPIQueryErrorCode(e.Code),
			Message:  e.Message,
		})
	}

	return api.GovernanceQueryResponse{
		Data:   data,
		Errors: errs,
		Meta:   toAPICursorMeta(res, pageSize),
	}
}

func toAPIFeatureAccess(fa governance.FeatureAccess) api.GovernanceFeatureAccess {
	out := api.GovernanceFeatureAccess{HasAccess: fa.HasAccess}
	if fa.Reason != nil {
		out.Reason = &api.GovernanceFeatureAccessReason{
			Code:    toAPIReasonCode(fa.Reason.Code),
			Message: fa.Reason.Message,
		}
	}
	return out
}

func toAPIReasonCode(code governance.ReasonCode) api.GovernanceFeatureAccessReasonCode {
	switch code {
	case governance.ReasonUsageLimitReached:
		return api.GovernanceFeatureAccessReasonCodeUsageLimitReached
	case governance.ReasonFeatureUnavailable:
		return api.GovernanceFeatureAccessReasonCodeFeatureUnavailable
	case governance.ReasonFeatureNotFound:
		return api.GovernanceFeatureAccessReasonCodeFeatureNotFound
	case governance.ReasonNoCreditAvailable:
		return api.GovernanceFeatureAccessReasonCodeNoCreditAvailable
	default:
		return api.GovernanceFeatureAccessReasonCodeUnknown
	}
}

func toAPIQueryErrorCode(code governance.QueryErrorCode) api.GovernanceQueryErrorCode {
	switch code {
	case governance.QueryErrorCustomerNotFound:
		return api.GovernanceQueryErrorCodeCustomerNotFound
	default:
		return api.GovernanceQueryErrorCodeUnknown
	}
}

// toAPICursorMeta builds cursor pagination metadata from the domain result.
func toAPICursorMeta(res governance.QueryResult, pageSize int) api.CursorMeta {
	meta := api.CursorMeta{
		Page: api.CursorMetaPage{
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
