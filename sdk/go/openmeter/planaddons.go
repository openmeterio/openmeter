package openmeter

import (
	"context"
	"iter"
	"net/http"
	"net/url"
	"time"
)

const plansBasePath = "/openmeter/plans"

// AddonReference identifies an add-on by its ULID.
type AddonReference struct {
	ID string `json:"id"`
}

// ProductCatalogValidationError describes a single product-catalog validation
// problem reported against a plan-addon association.
type ProductCatalogValidationError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	// Field is the path to the offending field, e.g.
	// "addons/pro/ratecards/token/featureKey".
	Field string `json:"field"`
	// Attributes carries additional structured context for the error.
	Attributes map[string]any `json:"attributes,omitempty"`
}

// PlanAddon is an association between a plan and an add-on, controlling which
// add-ons are available for purchase within a plan.
type PlanAddon struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description *string           `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	// Addon is the add-on associated with the plan.
	Addon AddonReference `json:"addon"`
	// FromPlanPhase is the key of the plan phase from which the add-on becomes
	// available for purchase.
	FromPlanPhase string `json:"from_plan_phase"`
	// MaxQuantity is the maximum number of times the add-on can be purchased for
	// the plan. It is omitted for single-instance add-ons; when omitted for
	// multi-instance add-ons, unlimited quantity can be purchased.
	MaxQuantity *int `json:"max_quantity,omitempty"`
	// ValidationErrors lists problems that make this plan-addon association
	// invalid. It is populated by the server and read-only.
	ValidationErrors []ProductCatalogValidationError `json:"validation_errors,omitempty"`
	CreatedAt        time.Time                       `json:"created_at"`
	UpdatedAt        time.Time                       `json:"updated_at"`
	DeletedAt        *time.Time                      `json:"deleted_at,omitempty"`
}

// PlanAddonPagePaginatedResponse is a page of plan-addons plus pagination metadata.
type PlanAddonPagePaginatedResponse struct {
	Data []PlanAddon   `json:"data"`
	Meta PaginatedMeta `json:"meta"`
}

// CreatePlanAddonRequest is the body for associating an add-on with a plan.
type CreatePlanAddonRequest struct {
	Name string `json:"name"`
	// Addon references the add-on to associate with the plan.
	Addon AddonReference `json:"addon"`
	// FromPlanPhase is the key of the plan phase from which the add-on becomes
	// available for purchase.
	FromPlanPhase string            `json:"from_plan_phase"`
	Description   *string           `json:"description,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
	MaxQuantity   *int              `json:"max_quantity,omitempty"`
}

// UpsertPlanAddonRequest is the body for updating a plan-addon association. The
// associated add-on cannot be changed, so it carries no Addon field.
type UpsertPlanAddonRequest struct {
	Name string `json:"name"`
	// FromPlanPhase is the key of the plan phase from which the add-on becomes
	// available for purchase.
	FromPlanPhase string            `json:"from_plan_phase"`
	Description   *string           `json:"description,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
	MaxQuantity   *int              `json:"max_quantity,omitempty"`
}

// PlanAddonsService groups operations on the add-ons associated with a plan,
// nested under /plans/{planId}/addons. Access it via Client.PlanAddons; every
// operation takes the parent plan ID as its first argument.
type PlanAddonsService struct {
	client *Client
}

// PlanAddonListParams are the optional query parameters for listing a plan's
// add-ons. The zero value lists the first default page.
type PlanAddonListParams struct {
	Page *PageParams
}

func (p PlanAddonListParams) values() url.Values {
	q := url.Values{}

	addPageParams(q, p.Page)

	return q
}

// planAddonsPath builds /openmeter/plans/{planID}/addons, guarding an empty
// plan ID via resourcePath.
func planAddonsPath(planID string) (string, error) {
	base, err := resourcePath(plansBasePath, planID)
	if err != nil {
		return "", err
	}

	return base + "/addons", nil
}

// planAddonPath builds /openmeter/plans/{planID}/addons/{planAddonID}, guarding
// both IDs via resourcePath.
func planAddonPath(planID, planAddonID string) (string, error) {
	base, err := planAddonsPath(planID)
	if err != nil {
		return "", err
	}

	return resourcePath(base, planAddonID)
}

// List returns a page of the add-ons associated with a plan.
func (s *PlanAddonsService) List(ctx context.Context, planID string, params PlanAddonListParams) (*PlanAddonPagePaginatedResponse, error) {
	path, err := planAddonsPath(planID)
	if err != nil {
		return nil, err
	}

	req, err := s.client.newRequest(ctx, http.MethodGet, path, params.values(), nil, contentTypeJSON)
	if err != nil {
		return nil, err
	}

	var out PlanAddonPagePaginatedResponse
	if err := s.client.doJSON(req, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

// ListAll returns an iterator over every add-on associated with a plan,
// transparently fetching successive pages. Range over it with Go 1.23+
// range-over-func; see MetersService.ListAll for the iteration contract.
func (s *PlanAddonsService) ListAll(ctx context.Context, planID string, params PlanAddonListParams) iter.Seq2[PlanAddon, error] {
	return paginate(params.Page, func(page, size int) ([]PlanAddon, int, error) {
		pageParams := params
		pageParams.Page = &PageParams{Size: Int(size), Number: Int(page)}

		resp, err := s.List(ctx, planID, pageParams)
		if err != nil {
			return nil, 0, err
		}

		return resp.Data, resp.Meta.Page.Total, nil
	})
}

// Create associates an add-on with a plan and returns the created association
// (HTTP 201).
func (s *PlanAddonsService) Create(ctx context.Context, planID string, request CreatePlanAddonRequest) (*PlanAddon, error) {
	path, err := planAddonsPath(planID)
	if err != nil {
		return nil, err
	}

	req, err := s.client.newRequest(ctx, http.MethodPost, path, nil, request, contentTypeJSON)
	if err != nil {
		return nil, err
	}

	var out PlanAddon
	if err := s.client.doJSON(req, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

// Get retrieves a single plan-addon association by its ID.
func (s *PlanAddonsService) Get(ctx context.Context, planID, planAddonID string) (*PlanAddon, error) {
	path, err := planAddonPath(planID, planAddonID)
	if err != nil {
		return nil, err
	}

	req, err := s.client.newRequest(ctx, http.MethodGet, path, nil, nil, contentTypeJSON)
	if err != nil {
		return nil, err
	}

	var out PlanAddon
	if err := s.client.doJSON(req, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

// Update replaces a plan-addon association by ID and returns the updated
// association.
func (s *PlanAddonsService) Update(ctx context.Context, planID, planAddonID string, request UpsertPlanAddonRequest) (*PlanAddon, error) {
	path, err := planAddonPath(planID, planAddonID)
	if err != nil {
		return nil, err
	}

	req, err := s.client.newRequest(ctx, http.MethodPut, path, nil, request, contentTypeJSON)
	if err != nil {
		return nil, err
	}

	var out PlanAddon
	if err := s.client.doJSON(req, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

// Delete removes a plan-addon association by ID. It returns nil on success
// (HTTP 204 No Content).
func (s *PlanAddonsService) Delete(ctx context.Context, planID, planAddonID string) error {
	path, err := planAddonPath(planID, planAddonID)
	if err != nil {
		return err
	}

	req, err := s.client.newRequest(ctx, http.MethodDelete, path, nil, nil, contentTypeJSON)
	if err != nil {
		return err
	}

	_, err = s.client.doRaw(req)
	return err
}
