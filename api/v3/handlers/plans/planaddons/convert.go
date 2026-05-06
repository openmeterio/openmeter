package planaddons

import (
	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/labels"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	"github.com/openmeterio/openmeter/pkg/models"
)

func ToAPIPlanAddon(a planaddon.PlanAddon) (api.PlanAddon, error) {
	validationIssues, _ := a.AsProductCatalogPlanAddon().ValidationErrors()

	return api.PlanAddon{
		Id:               a.ID,
		Addon:            api.AddonReferenceItem{Id: a.Addon.ID},
		FromPlanPhase:    a.PlanAddonConfig.FromPlanPhase,
		MaxQuantity:      a.PlanAddonConfig.MaxQuantity,
		CreatedAt:        a.CreatedAt,
		UpdatedAt:        a.UpdatedAt,
		DeletedAt:        a.DeletedAt,
		Labels:           labels.FromMetadata(a.Metadata),
		ValidationErrors: ToAPIProductCatalogValidationErrors(validationIssues),
	}, nil
}

func ToAPIProductCatalogValidationErrors(issues models.ValidationIssues) *[]api.ProductCatalogValidationError {
	if len(issues) == 0 {
		return nil
	}

	result := make([]api.ProductCatalogValidationError, 0, len(issues))
	for _, issue := range issues {
		result = append(result, api.ProductCatalogValidationError{
			Code:    string(issue.Code()),
			Field:   issue.Field().JSONPath(),
			Message: issue.Message(),
		})
	}

	return &result
}
