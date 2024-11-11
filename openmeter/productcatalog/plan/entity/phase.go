package planentity

import (
	"github.com/samber/lo"

	productcatalogmodel "github.com/openmeterio/openmeter/openmeter/productcatalog/model"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Phase struct {
	models.NamespacedID
	models.ManagedModel

	productcatalogmodel.Phase

	RateCards []RateCard `json:"rateCards"`
}

type NewPhaseConfig struct {
	models.NamespacedID
	models.ManagedModel

	productcatalogmodel.PhaseGeneric
	RateCards []RateCard                     `json:"rateCards"`
	Discounts []productcatalogmodel.Discount `json:"discounts"`
}

func NewPhase(conf NewPhaseConfig) Phase {
	phaseModel := productcatalogmodel.Phase{
		PhaseGeneric: conf.PhaseGeneric,
		RateCards:    lo.Map(conf.RateCards, func(rateCard RateCard, _ int) productcatalogmodel.RateCard { return rateCard.RateCard }),
		Discounts:    conf.Discounts,
	}

	return Phase{
		NamespacedID: conf.NamespacedID,
		ManagedModel: conf.ManagedModel,
		Phase:        phaseModel,
		RateCards:    conf.RateCards,
	}
}
