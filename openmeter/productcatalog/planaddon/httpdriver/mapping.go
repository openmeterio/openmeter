package httpdriver

import (
	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	"github.com/openmeterio/openmeter/pkg/models"
)

func FromPlanAddon(a planaddon.PlanAddon) (api.PlanAddon, error) {
	resp := api.PlanAddon{
		Addon: struct {
			Id           string                `json:"id"`
			InstanceType api.AddonInstanceType `json:"instanceType"`
			Key          string                `json:"key"`
			Version      int                   `json:"version"`
		}{
			Id:           a.Addon.ID,
			InstanceType: api.AddonInstanceType(a.Addon.InstanceType),
			Key:          a.Addon.Key,
			Version:      a.Addon.Version,
		},
		FromPlanPhase: a.PlanAddonConfig.FromPlanPhase,
		MaxQuantity:   a.PlanAddonConfig.MaxQuantity,
		CreatedAt:     a.CreatedAt,
		DeletedAt:     a.DeletedAt,
		UpdatedAt:     a.UpdatedAt,
		Annotations:   FromAnnotations(a.Annotations),
	}

	return resp, nil
}

func FromAnnotations(annotations models.Annotations) *api.Annotations {
	var result api.Annotations

	if len(annotations) > 0 {
		result = make(api.Annotations)

		for k, v := range annotations {
			result[k] = v
		}
	}

	return &result
}

func AsCreatePlanAddonRequest(a api.PlanAddonCreate, namespace string, planID string) (CreatePlanAddonRequest, error) {
	req := CreatePlanAddonRequest{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},
		Metadata:      nil, //FIXME:
		PlanID:        planID,
		AddonID:       a.Addon.Id,
		FromPlanPhase: "",
		MaxQuantity:   nil,
	}

	return req, nil
}

func AsUpdatePlanAddonRequest(a api.PlanAddonUpdate, namespace string, planID string, addonID string) (UpdatePlanAddonRequest, error) {
	req := UpdatePlanAddonRequest{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},
		Metadata:      nil, // FIXME:
		PlanID:        planID,
		AddonID:       addonID,
		FromPlanPhase: &a.FromPlanPhase,
		MaxQuantity:   a.MaxQuantity,
	}

	return req, nil
}
