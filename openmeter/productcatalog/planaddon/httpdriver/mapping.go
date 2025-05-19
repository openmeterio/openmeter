package httpdriver

import (
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	"github.com/openmeterio/openmeter/pkg/models"
)

func FromPlanAddon(a planaddon.PlanAddon) (api.PlanAddon, error) {
	validationIssues, _ := a.AsProductCatalogPlanAddon().ValidationErrors()

	resp := api.PlanAddon{
		Addon: struct {
			Id           string                `json:"id"`
			InstanceType api.AddonInstanceType `json:"instanceType"`
			Key          string                `json:"key"`
			Name         string                `json:"name"`
			Version      int                   `json:"version"`
		}{
			Id:           a.Addon.ID,
			InstanceType: api.AddonInstanceType(a.Addon.InstanceType),
			Key:          a.Addon.Key,
			Name:         a.Addon.Name,
			Version:      a.Addon.Version,
		},
		FromPlanPhase:    a.PlanAddonConfig.FromPlanPhase,
		MaxQuantity:      a.PlanAddonConfig.MaxQuantity,
		CreatedAt:        a.CreatedAt,
		DeletedAt:        a.DeletedAt,
		UpdatedAt:        a.UpdatedAt,
		Annotations:      FromAnnotations(a.Annotations),
		Metadata:         FromMetadata(a.Metadata),
		ValidationErrors: FromValidationErrors(validationIssues),
	}

	return resp, nil
}

func FromValidationErrors(issues models.ValidationIssues) *[]api.ValidationError {
	if len(issues) == 0 {
		return nil
	}

	var result []api.ValidationError

	for _, issue := range issues {
		result = append(result, api.ValidationError{
			Message: issue.Message,
			Field:   issue.Path,
			Code:    string(issue.Code),
		})
	}

	return &result
}

func FromAnnotations(annotations models.Annotations) *api.Annotations {
	if len(annotations) == 0 {
		return nil
	}

	var result api.Annotations

	if len(annotations) > 0 {
		result = make(api.Annotations)

		for k, v := range annotations {
			result[k] = v
		}
	}

	return &result
}

func FromMetadata(metadata models.Metadata) *api.Metadata {
	if len(metadata) == 0 {
		return nil
	}

	var result api.Metadata

	if len(metadata) > 0 {
		result = make(api.Metadata)

		for k, v := range metadata {
			result[k] = v
		}
	}

	return &result
}

func AsCreatePlanAddonRequest(a api.PlanAddonCreate, namespace string, planID string) (CreatePlanAddonRequest, error) {
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
		AddonID:       a.Addon.Id,
		FromPlanPhase: a.FromPlanPhase,
		MaxQuantity:   a.MaxQuantity,
	}

	return req, nil
}

func AsUpdatePlanAddonRequest(a api.PlanAddonReplaceUpdate, namespace string, planID string, addonID string) (UpdatePlanAddonRequest, error) {
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

func AsMetadata(metadata api.Metadata) models.Metadata {
	var result models.Metadata

	if len(metadata) > 0 {
		result = make(models.Metadata)

		for k, v := range metadata {
			result[k] = v
		}
	}

	return result
}
