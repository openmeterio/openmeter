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

var (
	ErrNamespaceRequired = errors.New("namespace is required")
	ErrIDRequired        = errors.New("id is required")
	ErrDuplicateID       = errors.New("duplicate id")
	ErrNotFound          = errors.New("not found")
)

func InIDOrder[T InIDOrderAccessor](namespace string, targetOrderIDs []string, results []T) ([]T, error) {
	return inIDOrder(namespace, targetOrderIDs, results, false)
}

// InIDOrderSkipMissing orders results like InIDOrder but drops target IDs with
// no matching entity instead of failing. Callers use it when the target ID set
// comes from an earlier snapshot and entities may have been deleted since.
func InIDOrderSkipMissing[T InIDOrderAccessor](namespace string, targetOrderIDs []string, results []T) ([]T, error) {
	return inIDOrder(namespace, targetOrderIDs, results, true)
}

func inIDOrder[T InIDOrderAccessor](namespace string, targetOrderIDs []string, results []T, skipMissing bool) ([]T, error) {
	// Input validation (let's make sure that namespace/id is set for all entities)
	if namespace == "" {
		return nil, ErrNamespaceRequired
	}

	for _, id := range targetOrderIDs {
		if id == "" {
			return nil, ErrIDRequired
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

	// Check for duplicate results
	for id, entities := range entitiesByID {
		if len(entities) > 1 {
			return nil, fmt.Errorf("%w [id=%s, count=%d]", ErrDuplicateID, id, len(entities))
		}
	}

	// We allow for more entities being present in the results set, as we are not filtering for namespace for the query to allow
	// multi-namespace listing as needed.
	var errs []error
	out := make([]T, 0, len(targetOrderIDs))
	for _, id := range targetOrderIDs {
		entities, ok := entitiesByID[models.NamespacedID{Namespace: namespace, ID: id}]
		if !ok {
			if !skipMissing {
				errs = append(errs, fmt.Errorf("%w [id=%s]", ErrNotFound, id))
			}
			continue
		}

		out = append(out, entities...)
	}

	if len(errs) > 0 {
		return nil, models.NewGenericNotFoundError(errors.Join(errs...))
	}

	return out, nil
}
