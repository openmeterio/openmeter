package governance

import (
	"context"
	"fmt"
	"net/http"
	"sort"

	"github.com/oapi-codegen/nullable"
	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	customershandler "github.com/openmeterio/openmeter/api/v3/handlers/customers"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	pagination "github.com/openmeterio/openmeter/pkg/pagination/v2"
)

const (
	defaultPageSize = 100
	maxPageSize     = 100
)

type (
	QueryGovernanceAccessParams   = api.QueryGovernanceAccessParams
	QueryGovernanceAccessResponse = api.GovernanceQueryResponse
	QueryGovernanceAccessHandler  = httptransport.HandlerWithArgs[queryGovernanceAccessRequest, QueryGovernanceAccessResponse, QueryGovernanceAccessParams]
)

type queryGovernanceAccessRequest struct {
	Namespace      string
	CustomerKeys   []string
	FeatureKeys    []string // nil means all features
	IncludeCredits bool
	PageSize       int
	Cursor         *pagination.Cursor
}

func (h *handler) QueryGovernanceAccess() QueryGovernanceAccessHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params QueryGovernanceAccessParams) (queryGovernanceAccessRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return queryGovernanceAccessRequest{}, err
			}

			var body api.GovernanceQueryRequest
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return queryGovernanceAccessRequest{}, err
			}

			pageSize := defaultPageSize
			var cursor *pagination.Cursor

			if params.Page != nil {
				if params.Page.Size != nil {
					pageSize = *params.Page.Size
					if pageSize < 1 || pageSize > maxPageSize {
						return queryGovernanceAccessRequest{}, apierrors.NewBadRequestError(ctx,
							fmt.Errorf("page[size] must be between 1 and %d", maxPageSize),
							apierrors.InvalidParameters{{
								Field:  "page[size]",
								Reason: fmt.Sprintf("must be between 1 and %d", maxPageSize),
								Source: apierrors.InvalidParamSourceQuery,
							}},
						)
					}
				}

				if params.Page.After != nil {
					decoded, err := pagination.DecodeCursor(*params.Page.After)
					if err != nil {
						return queryGovernanceAccessRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{{
							Field:  "page[after]",
							Reason: err.Error(),
							Source: apierrors.InvalidParamSourceQuery,
						}})
					}
					cursor = decoded
				}
			}

			req := queryGovernanceAccessRequest{
				Namespace:      ns,
				CustomerKeys:   body.Customer.Keys,
				IncludeCredits: lo.FromPtrOr(body.IncludeCredits, false),
				PageSize:       pageSize,
				Cursor:         cursor,
			}

			if body.Feature != nil {
				req.FeatureKeys = body.Feature.Keys
			}

			return req, nil
		},
		func(ctx context.Context, req queryGovernanceAccessRequest) (QueryGovernanceAccessResponse, error) {
			return h.processGovernanceQuery(ctx, req)
		},
		commonhttp.JSONResponseEncoderWithStatus[QueryGovernanceAccessResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("query-governance-access"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}

// resolvedCustomer groups matched input keys for a single customer.
type resolvedCustomer struct {
	Customer customer.Customer
	Matched  []string
}

func (h *handler) processGovernanceQuery(ctx context.Context, req queryGovernanceAccessRequest) (QueryGovernanceAccessResponse, error) {
	// Resolve each input key to a customer; deduplicate by customer ID.
	customerMap := make(map[string]*resolvedCustomer)
	var queryErrors []api.GovernanceQueryError

	for _, key := range req.CustomerKeys {
		cus, err := h.customerService.GetCustomerByUsageAttribution(ctx, customer.GetCustomerByUsageAttributionInput{
			Namespace: req.Namespace,
			Key:       key,
		})
		if err != nil {
			if models.IsGenericNotFoundError(err) {
				queryErrors = append(queryErrors, api.GovernanceQueryError{
					Customer: lo.ToPtr(key),
					Code:     api.GovernanceQueryErrorCodeCustomerNotFound,
					Message:  "customer not found",
				})
				continue
			}
			return QueryGovernanceAccessResponse{}, fmt.Errorf("resolve customer key %q: %w", key, err)
		}

		if rc, ok := customerMap[cus.ID]; ok {
			rc.Matched = append(rc.Matched, key)
		} else {
			customerMap[cus.ID] = &resolvedCustomer{
				Customer: *cus,
				Matched:  []string{key},
			}
		}
	}

	// Sort by (CreatedAt, ID) for stable cursor pagination.
	customers := lo.Values(customerMap)
	sort.Slice(customers, func(i, j int) bool {
		ti := customers[i].Customer.CreatedAt
		tj := customers[j].Customer.CreatedAt
		if !ti.Equal(tj) {
			return ti.Before(tj)
		}
		return customers[i].Customer.ID < customers[j].Customer.ID
	})

	// Apply cursor: skip everything at or before the cursor position.
	if req.Cursor != nil {
		afterCursor := *req.Cursor
		start := len(customers) // default: nothing left if cursor is beyond all items
		for i, rc := range customers {
			c := pagination.NewCursor(rc.Customer.CreatedAt, rc.Customer.ID)
			if c.Time.After(afterCursor.Time) || (c.Time.Equal(afterCursor.Time) && c.ID > afterCursor.ID) {
				start = i
				break
			}
		}
		customers = customers[start:]
	}

	// Apply page size.
	hasMore := len(customers) > req.PageSize
	if hasMore {
		customers = customers[:req.PageSize]
	}

	// Compute feature access for each paged customer.
	now := clock.Now()
	results := make([]api.GovernanceQueryResult, 0, len(customers))

	for _, rc := range customers {
		access, err := h.entitlementService.GetAccess(ctx, req.Namespace, rc.Customer.ID)
		if err != nil {
			return QueryGovernanceAccessResponse{}, fmt.Errorf("get access for customer %s: %w", rc.Customer.ID, err)
		}

		featureAccess, err := h.buildFeatureAccess(ctx, req.Namespace, req.FeatureKeys, access)
		if err != nil {
			return QueryGovernanceAccessResponse{}, fmt.Errorf("build feature access for customer %s: %w", rc.Customer.ID, err)
		}

		results = append(results, api.GovernanceQueryResult{
			Matched:   rc.Matched,
			Customer:  customershandler.ToAPIBillingCustomer(rc.Customer),
			Features:  featureAccess,
			UpdatedAt: now,
		})
	}

	return QueryGovernanceAccessResponse{
		Data:   results,
		Errors: queryErrors,
		Meta:   buildCursorMeta(customers, req.PageSize, hasMore),
	}, nil
}

// buildFeatureAccess returns the feature access map for a single customer.
// If featureKeys is non-empty, only those keys are evaluated; otherwise all entitlements are returned.
func (h *handler) buildFeatureAccess(ctx context.Context, ns string, featureKeys []string, access entitlement.Access) (map[string]api.GovernanceFeatureAccess, error) {
	result := make(map[string]api.GovernanceFeatureAccess)

	if len(featureKeys) == 0 {
		for key, ev := range access.Entitlements {
			result[key] = mapEntitlementToAccess(ev.Value)
		}
		return result, nil
	}

	for _, key := range featureKeys {
		ev, ok := access.Entitlements[key]
		if !ok {
			access, err := h.resolveAbsentFeature(ctx, ns, key)
			if err != nil {
				return nil, err
			}
			result[key] = access
			continue
		}
		result[key] = mapEntitlementToAccess(ev.Value)
	}

	return result, nil
}

// resolveAbsentFeature determines the reason a requested feature key is absent from GetAccess results:
// either the feature doesn't exist in the org (FeatureNotFound) or the customer has no entitlement for it (FeatureUnavailable).
func (h *handler) resolveAbsentFeature(ctx context.Context, ns, featureKey string) (api.GovernanceFeatureAccess, error) {
	_, err := h.featureConnector.GetFeature(ctx, ns, featureKey, feature.IncludeArchivedFeatureFalse)
	if err != nil {
		if models.IsGenericNotFoundError(err) {
			return api.GovernanceFeatureAccess{
				HasAccess: false,
				Reason: &api.GovernanceFeatureAccessReason{
					Code:    api.GovernanceFeatureAccessReasonCodeFeatureNotFound,
					Message: fmt.Sprintf("feature %q not found", featureKey),
				},
			}, nil
		}
		return api.GovernanceFeatureAccess{}, fmt.Errorf("get feature %q: %w", featureKey, err)
	}

	return api.GovernanceFeatureAccess{
		HasAccess: false,
		Reason: &api.GovernanceFeatureAccessReason{
			Code:    api.GovernanceFeatureAccessReasonCodeFeatureUnavailable,
			Message: fmt.Sprintf("feature %q is not available for this customer", featureKey),
		},
	}, nil
}

func buildCursorMeta(customers []*resolvedCustomer, pageSize int, hasMore bool) api.CursorMeta {
	meta := api.CursorMeta{
		Page: api.CursorMetaPage{
			Next:     nullable.NewNullNullable[string](),
			Previous: nullable.NewNullNullable[string](),
			Size:     float32(pageSize),
		},
	}

	if len(customers) > 0 {
		first := customers[0]
		last := customers[len(customers)-1]
		firstCursor := pagination.NewCursor(first.Customer.CreatedAt, first.Customer.ID)
		lastCursor := pagination.NewCursor(last.Customer.CreatedAt, last.Customer.ID)
		meta.Page.First = lo.ToPtr(firstCursor.Encode())
		meta.Page.Last = lo.ToPtr(lastCursor.Encode())
		if hasMore {
			meta.Page.Next = nullable.NewNullableWithValue(lastCursor.Encode())
		}
	}

	return meta
}
