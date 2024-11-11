package planentity

import (
	"github.com/samber/lo"

	productcatalogmodel "github.com/openmeterio/openmeter/openmeter/productcatalog/model"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Plan struct {
	models.NamespacedID
	models.ManagedModel

	productcatalogmodel.Plan

	Phases []Phase `json:"phases"`
}

type NewPlanConfig struct {
	models.NamespacedID
	models.ManagedModel

	Plan   productcatalogmodel.PlanGeneric
	Phases []Phase
}

func NewPlan(conf NewPlanConfig) Plan {
	planModel := productcatalogmodel.Plan{
		PlanGeneric: conf.Plan,
		Phases:      lo.Map(conf.Phases, func(phase Phase, _ int) productcatalogmodel.Phase { return phase.Phase }),
	}
	return Plan{
		NamespacedID: conf.NamespacedID,
		ManagedModel: conf.ManagedModel,
		Plan:         planModel,
		Phases:       conf.Phases,
	}
}
