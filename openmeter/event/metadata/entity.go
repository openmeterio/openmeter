package metadata

import (
	"errors"
	"fmt"
	"time"
)

// EntityMutationType is the type of mutation that occurred on an entity
type EntityMutationType string

const (
	EntityMutationTypeCreate EntityMutationType = "create"
	EntityMutationTypeUpdate EntityMutationType = "update"
	EntityMutationTypeDelete EntityMutationType = "delete"
)

// EventEntityMutationPayload is a payload for an entity mutation event
type EventEntityMutationPayload[T any] struct {
	Entity       string             `json:"entity"`       // entity type "customer"
	MutationType EntityMutationType `json:"mutationType"` // mutation type "create"
	New          *T                 `json:"new"`          // only applicable for create and update
	Previous     *T                 `json:"previous"`     // only applicable for update and delete
}

// Validate validates the event entity mutation payload
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

// GetMutationEventID returns a unique identifier for a mutation event
func GetMutationEventID(mutationType EntityMutationType, id string, t time.Time) string {
	return fmt.Sprintf("%s-%s-%s", id, mutationType, t.Format(time.RFC3339))
}
