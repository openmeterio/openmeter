package customerentity

import (
	"fmt"
	"strings"
)

var _ error = (*NotFoundError)(nil)

type NotFoundError struct {
	CustomerID
}

func (e NotFoundError) Error() string {
	return fmt.Sprintf("resource with id %s not found in %s namespace", e.ID, e.Namespace)
}

type genericError struct {
	Err error
}

var _ error = (*ValidationError)(nil)

type ValidationError genericError

func (e ValidationError) Error() string {
	return e.Err.Error()
}

func (e ValidationError) Unwrap() error {
	return e.Err
}

var _ error = (*UpdateAfterDeleteError)(nil)

type UpdateAfterDeleteError genericError

func (e UpdateAfterDeleteError) Error() string {
	return e.Err.Error()
}

func (e UpdateAfterDeleteError) Unwrap() error {
	return e.Err
}

// SubjectKeyConflictError represents an error when a subject key is already associated with a customer
type SubjectKeyConflictError struct {
	Namespace   string   `json:"namespace"`
	SubjectKeys []string `json:"subjectKeys"`
}

func (e SubjectKeyConflictError) Error() string {
	return fmt.Sprintf("one or multiple subject keys of [%s] are already associated with an different customer in the namespace %s", strings.Join(e.SubjectKeys, ", "), e.Namespace)
}
