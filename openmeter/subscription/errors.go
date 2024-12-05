package subscription

import "fmt"

type ForbiddenError struct {
	Msg string
}

func (e *ForbiddenError) Error() string {
	return e.Msg
}

type NotFoundError struct {
	ID         string
	CustomerID string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("subscription %s not found for customer %s", e.ID, e.CustomerID)
}

type PhaseNotFoundError struct {
	ID string
}

func (e *PhaseNotFoundError) Error() string {
	return fmt.Sprintf("subscription phase %s not found", e.ID)
}

type ItemNotFoundError struct {
	ID string
}

func (e *ItemNotFoundError) Error() string {
	return fmt.Sprintf("subscription item %s not found", e.ID)
}
