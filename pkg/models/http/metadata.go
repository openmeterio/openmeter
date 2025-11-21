package http

import (
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/pkg/models"
)

func FromMetadata(metadata models.Metadata) *api.Metadata {
	return lo.ToPtr((api.Metadata)(metadata))
}

func AsMetadata(metadata *api.Metadata) models.Metadata {
	if metadata == nil {
		return nil
	}

	return (models.Metadata)(*metadata)
}
