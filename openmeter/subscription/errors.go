package subscription

import "fmt"

type NotFoundError struct {
	ID         string
	CustomerID string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("subscription %s not found for customer %s", e.ID, e.CustomerID)
}
