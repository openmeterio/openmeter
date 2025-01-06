package subscription

import (
	"fmt"

	"github.com/openmeterio/openmeter/pkg/errorsx"
)

type ForbiddenError struct {
	Msg string
}

func (e *ForbiddenError) Error() string {
	return e.Msg
}

func (e *ForbiddenError) Traits() []errorsx.Trait {
	return []errorsx.Trait{errorsx.Forbidden}
}

var _ errorsx.ErrorWithTraits = &ForbiddenError{}

type NotFoundError struct {
	ID         string
	CustomerID string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("subscription %s not found for customer %s", e.ID, e.CustomerID)
}

func (e *NotFoundError) Traits() []errorsx.Trait {
	return []errorsx.Trait{errorsx.NotFound}
}

var _ errorsx.ErrorWithTraits = &NotFoundError{}

type PhaseNotFoundError struct {
	ID string
}

func (e *PhaseNotFoundError) Error() string {
	return fmt.Sprintf("subscription phase %s not found", e.ID)
}

func (e *PhaseNotFoundError) Traits() []errorsx.Trait {
	return []errorsx.Trait{errorsx.NotFound}
}

type ItemNotFoundError struct {
	ID string
}

func (e *ItemNotFoundError) Error() string {
	return fmt.Sprintf("subscription item %s not found", e.ID)
}

func (e *ItemNotFoundError) Traits() []errorsx.Trait {
	return []errorsx.Trait{errorsx.NotFound}
}
