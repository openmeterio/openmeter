//go:generate go run github.com/jmattheis/goverter/cmd/goverter gen ./

package addons

import (
	"fmt"

	apiv3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/labels"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/pkg/models"
)

// goverter:variables
// goverter:skipCopySameType
// goverter:output:file ./convert.gen.go
// goverter:useZeroValueOnPointerInconsistency
// goverter:useUnderlyingTypeMethods
// goverter:matchIgnoreCase
// goverter:enum:unknown @error
// goverter:extend ConvertMetadataAnnotationsToLabels
// goverter:extend ConvertLabelsToMetadata
// goverter:extend ConvertAddonStatus
var (
	// goverter:map ManagedModel.CreatedAt CreatedAt
	// goverter:map AddonMeta.Currency Currency
	// goverter:map ManagedModel.DeletedAt DeletedAt
	// goverter:map AddonMeta.Description Description
	// goverter:map AddonMeta.EffectivePeriod.EffectiveFrom EffectiveFrom
	// goverter:map AddonMeta.EffectivePeriod.EffectiveTo EffectiveTo
	// goverter:map NamespacedID.ID Id
	// goverter:map AddonMeta.InstanceType InstanceType
	// goverter:map AddonMeta.Key Key
	// goverter:map . Labels | ConvertMetadataAnnotationsToLabels
	// goverter:map AddonMeta.Name Name
	// FIXME: AddonMeta.RateCards
	// goverter:ignore RateCards
	// goverter:map . Status | ConvertAddonStatus
	// goverter:map ManagedModel.UpdatedAt UpdatedAt
	// goverter:map . ValidationErrors | ConvertToValidationErrors
	// goverter:map AddonMeta.Version Version
	ConvertFromAddon func(addon addon.Addon) (apiv3.Addon, error)

	// goverter:context namespace
	// goverter:map NamespacedModel | NamespaceModelFromContext
	// goverter:map . Addon | convertToCreateAddonRequestAddon
	// goverter:ignore inputOptions
	ConvertToCreateAddonRequest func(namespace string, request apiv3.CreateAddonRequest) (addon.CreateAddonInput, error)

	// goverter:map . AddonMeta | convertToCreateAddonRequestAddonMeta
	// FIXME: AddonMeta.RateCards
	// goverter:ignore RateCards
	convertToCreateAddonRequestAddon func(request apiv3.CreateAddonRequest) (productcatalog.Addon, error)

	// goverter:map Labels Metadata | ConvertLabelsToMetadata
	// goverter:ignore EffectivePeriod
	// goverter:ignore Version
	// goverter:ignore Annotations
	convertToCreateAddonRequestAddonMeta func(request apiv3.CreateAddonRequest) (productcatalog.AddonMeta, error)
)

// goverter:context namespace
func NamespaceModelFromContext(namespace string) models.NamespacedModel {
	return models.NamespacedModel{
		Namespace: namespace,
	}
}

var ConvertLabelsToMetadata = labels.ToMetadata

func ConvertMetadataAnnotationsToLabels(source addon.Addon) *apiv3.Labels {
	return labels.FromMetadataAnnotations(source.Metadata, source.Annotations)
}

func ConvertAddonStatus(source addon.Addon) (apiv3.AddonStatus, error) {
	switch source.Status() {
	case productcatalog.AddonStatusDraft:
		return apiv3.AddonStatusDraft, nil
	case productcatalog.AddonStatusActive:
		return apiv3.AddonStatusActive, nil
	case productcatalog.AddonStatusArchived:
		return apiv3.AddonStatusArchived, nil
	default:
		return "", fmt.Errorf("invalid add-on status: %s", source.Status())
	}
}

func FromValidationAttributes(attrs models.Attributes) *map[string]interface{} {
	if len(attrs) == 0 {
		return nil
	}

	out := attrs.AsStringMap()

	if len(out) == 0 {
		return nil
	}

	return &out
}

func ConvertToValidationErrors(source addon.Addon) (*[]apiv3.ProductCatalogValidationError, error) {
	issues, err := source.AsProductCatalogAddon().ValidationErrors()
	if err != nil {
		return nil, err
	}

	if len(issues) == 0 {
		return nil, nil
	}

	var result []apiv3.ProductCatalogValidationError

	for _, issue := range issues {
		result = append(result, apiv3.ProductCatalogValidationError{
			Message:    issue.Message(),
			Field:      issue.Field().JSONPath(),
			Code:       string(issue.Code()),
			Attributes: FromValidationAttributes(issue.Attributes()),
		})
	}

	return &result, nil
}
