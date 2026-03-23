package entutils

import (
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/models"
)

type InIDOrderAccessor interface {
	IDMixinGetter
	NamespaceMixinGetter
}

func InIDOrder[T InIDOrderAccessor](targetOrderIDs []models.NamespacedID, results []T) ([]T, error) {
	// Input validation (let's make sure that namespace/id is set for all entities)
	for _, id := range targetOrderIDs {
		if err := id.Validate(); err != nil {
			return nil, err
		}
	}

	for _, result := range results {
		namespacedID := models.NamespacedID{
			Namespace: result.GetNamespace(),
			ID:        result.GetID(),
		}
		if err := namespacedID.Validate(); err != nil {
			return nil, err
		}
	}

	// Logic implementation
	if len(targetOrderIDs) == 0 && len(results) == 0 {
		return results, nil
	}

	entitiesByID := lo.GroupBy(results, func(result T) models.NamespacedID {
		return models.NamespacedID{
			Namespace: result.GetNamespace(),
			ID:        result.GetID(),
		}
	})

	// We allow for more entities being present in the results set, as we are not filtering for namespace for the query to allow
	// multi-namespace listing as needed.
	var errs []error
	for _, id := range targetOrderIDs {
		entities, ok := entitiesByID[id]
		if !ok {
			errs = append(errs, fmt.Errorf("not found [id=%s]", id))
			continue
		}

		results = append(results, entities...)
	}

	if len(errs) > 0 {
		return nil, models.NewGenericNotFoundError(errors.Join(errs...))
	}

	return results, nil
}
