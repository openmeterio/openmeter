package price

import "github.com/openmeterio/openmeter/pkg/models"

type Repository interface {
	models.CadencedResourceRepo[Price]

	GetByID(id string) (*Price, error)
	Get(subscriptionId string, phaseKey string, itemKey string) (*Price, error)
	GetForSubscription(subscriptionId string) ([]Price, error)
	Create(input CreateInput) (*Price, error)
}
