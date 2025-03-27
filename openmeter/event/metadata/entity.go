package metadata

import (
	"errors"
	"fmt"
	"time"
)

type EntityMutationType string

const (
	EntityMutationTypeCreate EntityMutationType = "create"
	EntityMutationTypeUpdate EntityMutationType = "update"
	EntityMutationTypeDelete EntityMutationType = "delete"
)

type EventEntityMutationPayload[T any] struct {
	Entity       string             // entity type "customer"
	MutationType EntityMutationType // mutation type "create"
	New          *T                 // only applicable for create and update
	Previous     *T                 // only applicable for update and delete
}

func (e EventEntityMutationPayload[T]) Validate() error {
	var errs []error

	if e.Entity == "" {
		errs = append(errs, fmt.Errorf("entity is required"))
	}

	if e.MutationType == "" {
		errs = append(errs, fmt.Errorf("mutation type is required"))
	}

	if e.MutationType == EntityMutationTypeCreate {
		if e.New == nil {
			errs = append(errs, fmt.Errorf("new is required"))
		}
	}

	if e.MutationType == EntityMutationTypeUpdate {
		if e.New == nil {
			errs = append(errs, fmt.Errorf("new is required"))
		}

		if e.Previous == nil {
			errs = append(errs, fmt.Errorf("previous is required"))
		}
	}

	if e.MutationType == EntityMutationTypeDelete {
		if e.Previous == nil {
			errs = append(errs, fmt.Errorf("previous is required"))
		}

		if e.New != nil {
			errs = append(errs, fmt.Errorf("new is not allowed for delete"))
		}
	}

	return errors.Join(errs...)
}

func GetMutationEventID(mutationType EntityMutationType, id string, t time.Time) string {
	return fmt.Sprintf("%s-%s-%s", id, mutationType, t.Format(time.RFC3339))
}
