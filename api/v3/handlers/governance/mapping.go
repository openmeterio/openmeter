package governance

import (
	"github.com/oapi-codegen/nullable"
	"github.com/samber/lo"

	apiv3 "github.com/openmeterio/openmeter/api/v3"
	customershandler "github.com/openmeterio/openmeter/api/v3/handlers/customers"
	"github.com/openmeterio/openmeter/openmeter/governance"
)

// ToAPIGovernanceQueryResponse maps a domain QueryResult to the API response.
func ToAPIGovernanceQueryResponse(res governance.QueryResult, pageSize int) apiv3.GovernanceQueryResponse {
	data := make([]apiv3.GovernanceQueryResult, 0, len(res.Customers))

	for _, c := range res.Customers {
		features := make(map[string]apiv3.GovernanceFeatureAccess, len(c.Features))
		for key, fa := range c.Features {
			features[key] = toAPIFeatureAccess(fa)
		}

		data = append(data, apiv3.GovernanceQueryResult{
			Matched:   c.Matched,
			Customer:  customershandler.ToAPIBillingCustomer(c.Customer),
			Features:  features,
			UpdatedAt: c.UpdatedAt,
		})
	}

	errs := make([]apiv3.GovernanceQueryError, 0, len(res.Errors))

	for _, e := range res.Errors {
		errs = append(errs, apiv3.GovernanceQueryError{
			Customer: lo.ToPtr(e.CustomerKey),
			Code:     toAPIQueryErrorCode(e.Code),
			Message:  e.Message,
		})
	}

	return apiv3.GovernanceQueryResponse{
		Data:   data,
		Errors: errs,
		Meta:   toAPICursorMeta(res, pageSize),
	}
}

func toAPIFeatureAccess(fa governance.FeatureAccess) apiv3.GovernanceFeatureAccess {
	out := apiv3.GovernanceFeatureAccess{HasAccess: fa.HasAccess}

	if fa.Reason != nil {
		out.Reason = &apiv3.GovernanceFeatureAccessReason{
			Code:    toAPIReasonCode(fa.Reason.Code),
			Message: fa.Reason.Message,
		}
	}

	return out
}

func toAPIReasonCode(code governance.ReasonCode) apiv3.GovernanceFeatureAccessReasonCode {
	switch code {
	case governance.ReasonUsageLimitReached:
		return apiv3.GovernanceFeatureAccessReasonCodeUsageLimitReached
	case governance.ReasonFeatureUnavailable:
		return apiv3.GovernanceFeatureAccessReasonCodeFeatureUnavailable
	case governance.ReasonFeatureNotFound:
		return apiv3.GovernanceFeatureAccessReasonCodeFeatureNotFound
	case governance.ReasonNoCreditAvailable:
		return apiv3.GovernanceFeatureAccessReasonCodeNoCreditAvailable
	default:
		return apiv3.GovernanceFeatureAccessReasonCodeUnknown
	}
}

func toAPIQueryErrorCode(code governance.QueryErrorCode) apiv3.GovernanceQueryErrorCode {
	switch code {
	case governance.QueryErrorCustomerNotFound:
		return apiv3.GovernanceQueryErrorCodeCustomerNotFound
	default:
		return apiv3.GovernanceQueryErrorCodeUnknown
	}
}

// toAPICursorMeta builds cursor pagination metadata from the domain result.
func toAPICursorMeta(res governance.QueryResult, pageSize int) apiv3.CursorMeta {
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
