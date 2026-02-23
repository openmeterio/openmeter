package adapter

import (
	"errors"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
)

func mapTaxCodeFromEntity(entity *db.TaxCode) (taxcode.TaxCode, error) {
	if entity == nil {
		return taxcode.TaxCode{}, errors.New("entity is required")
	}

	return taxcode.TaxCode{
		NamespacedID: models.NamespacedID{
			Namespace: entity.Namespace,
			ID:        entity.ID,
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: entity.CreatedAt,
			UpdatedAt: entity.UpdatedAt,
			DeletedAt: entity.DeletedAt,
		},
		Key:         entity.Key,
		Name:        entity.Name,
		Description: entity.Description,
		AppMappings: lo.FromPtr(entity.AppMappings),
		Metadata:    models.NewMetadata(entity.Metadata),
	}, nil
}
