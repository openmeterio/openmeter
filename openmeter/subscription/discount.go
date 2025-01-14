package subscription

import (
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Discount struct {
	models.NamespacedID
	models.CadencedModel
	productcatalog.Discount

	PhaseID        string `json:"phaseId"`
	SubscriptionID string `json:"subscriptionId"`
}

func (d Discount) Validate() error {
	if err := d.Discount.Validate(); err != nil {
		return err
	}

	return nil
}
