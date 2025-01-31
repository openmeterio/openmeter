package progressmanager

import (
	"fmt"
)

var _ error = (*NotFoundError)(nil)

type NotFoundError struct {
	ID     string
	Entity string
}

func (e NotFoundError) Error() string {
	return fmt.Sprintf("not found: %s %s", e.Entity, e.ID)
}
