package http

import (
	"github.com/samber/lo"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/pkg/models"
)

func FromMetadata(metadata models.Metadata) *apiv1models.Metadata {
	return lo.ToPtr(apiv1models.Metadata(metadata))
}

func AsMetadata(metadata *apiv1models.Metadata) models.Metadata {
	if metadata == nil {
		return nil
	}

	return models.Metadata(*metadata)
}
