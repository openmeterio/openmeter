package plan

import (
	"errors"
	"fmt"
)

var _ error = (*NotFoundError)(nil)

type NotFoundError struct {
	Namespace string
	ID        string
	Key       string
	Version   int
}

func (e *NotFoundError) Error() string {
	var m string

	if e.Namespace != "" {
		m += fmt.Sprintf(" namespace=%s", e.Namespace)
	}

	if e.ID != "" {
		m += fmt.Sprintf(" id=%s", e.ID)
	}

	if e.Key != "" {
		m += fmt.Sprintf(" key=%s", e.Key)
	}

	if e.Version != 0 {
		m += fmt.Sprintf(" version=%d", e.Version)
	}

	if len(m) > 0 {
		return fmt.Sprintf("plan not found. [%s]", m[1:])
	}

	return "plan not found"
}

func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	var e *NotFoundError

	return errors.As(err, &e)
}
