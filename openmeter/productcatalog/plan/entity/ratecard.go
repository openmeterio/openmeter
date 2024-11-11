package planentity

import (
	productcatalogmodel "github.com/openmeterio/openmeter/openmeter/productcatalog/model"
	"github.com/openmeterio/openmeter/pkg/models"
)

type RateCard struct {
	models.NamespacedID
	models.ManagedModel

	productcatalogmodel.RateCard
}
