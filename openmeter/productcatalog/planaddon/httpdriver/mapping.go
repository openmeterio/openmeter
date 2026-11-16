package httpdriver

import (
	"fmt"

	"github.com/samber/lo"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	addonhttp "github.com/openmeterio/openmeter/openmeter/productcatalog/addon/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/http"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
)

// FromPlanAddon maps a plan add-on assignment to its v1 shape. taxCodes must hold every Tax Code
// referenced by the add-on's rate cards: a missing entry is an error.
func FromPlanAddon(a planaddon.PlanAddon, taxCodes map[string]*taxcode.TaxCode) (apiv1models.PlanAddon, error) {
	validationIssues, _ := a.AsProductCatalogPlanAddon().ValidationErrors()

	apiAddon, err := addonhttp.FromAddon(a.Addon, taxCodes)
	if err != nil {
		return apiv1models.PlanAddon{}, fmt.Errorf("failed to cast add-on [namespace=%s id=%s key=%s]: %w",
			a.Addon.Namespace, a.Addon.ID, a.Addon.Key, err)
	}

	resp := apiv1models.PlanAddon{
		Addon:            apiAddon,
		FromPlanPhase:    a.PlanAddonConfig.FromPlanPhase,
		MaxQuantity:      a.PlanAddonConfig.MaxQuantity,
		CreatedAt:        a.CreatedAt,
		DeletedAt:        a.DeletedAt,
		UpdatedAt:        a.UpdatedAt,
		Annotations:      http.FromAnnotations(a.Annotations),
		Metadata:         http.FromMetadata(a.Metadata),
		ValidationErrors: http.FromValidationErrors(validationIssues),
	}

	return resp, nil
}

func AsCreatePlanAddonRequest(a apiv1models.PlanAddonCreate, namespace string, planID string) (CreatePlanAddonRequest, error) {
	var metadata models.Metadata

	if a.Metadata != nil {
		metadata = AsMetadata(*a.Metadata)
	}

	req := CreatePlanAddonRequest{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},
		Metadata:      metadata,
		PlanID:        planID,
		AddonID:       a.AddonId,
		FromPlanPhase: a.FromPlanPhase,
		MaxQuantity:   a.MaxQuantity,
	}

	return req, nil
}

func AsUpdatePlanAddonRequest(a apiv1models.PlanAddonReplaceUpdate, namespace string, planID string, addonID string) (UpdatePlanAddonRequest, error) {
	var metadata *models.Metadata

	if a.Metadata != nil {
		metadata = lo.ToPtr(AsMetadata(*a.Metadata))
	}

	req := UpdatePlanAddonRequest{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},
		Metadata:      metadata,
		PlanID:        planID,
		AddonID:       addonID,
		FromPlanPhase: &a.FromPlanPhase,
		MaxQuantity:   a.MaxQuantity,
	}

	return req, nil
}

func AsMetadata(metadata apiv1models.Metadata) models.Metadata {
	var result models.Metadata

	if len(metadata) > 0 {
		result = make(models.Metadata)

		for k, v := range metadata {
			result[k] = v
		}
	}

	return result
}
