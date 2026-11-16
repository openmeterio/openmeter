package planaddons

import (
	"github.com/openmeterio/openmeter/api/v3/labels"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	"github.com/openmeterio/openmeter/pkg/models"
)

func ToAPIPlanAddon(a planaddon.PlanAddon) (apiv3models.PlanAddon, error) {
	validationIssues, _ := a.AsProductCatalogPlanAddon().ValidationErrors()

	return apiv3models.PlanAddon{
		Id:               a.ID,
		Addon:            apiv3models.AddonReference{Id: a.Addon.ID},
		FromPlanPhase:    a.PlanAddonConfig.FromPlanPhase,
		MaxQuantity:      a.PlanAddonConfig.MaxQuantity,
		CreatedAt:        a.CreatedAt,
		UpdatedAt:        a.UpdatedAt,
		DeletedAt:        a.DeletedAt,
		Labels:           labels.FromMetadata(a.Metadata),
		ValidationErrors: ToAPIProductCatalogValidationErrors(validationIssues),
	}, nil
}

func ToAPIProductCatalogValidationErrors(issues models.ValidationIssues) *[]apiv3models.ProductCatalogValidationError {
	if len(issues) == 0 {
		return nil
	}

	result := make([]apiv3models.ProductCatalogValidationError, 0, len(issues))
	for _, issue := range issues {
		result = append(result, apiv3models.ProductCatalogValidationError{
			Code:    string(issue.Code()),
			Field:   issue.Field().JSONPath(),
			Message: issue.Message(),
		})
	}

	return &result
}
