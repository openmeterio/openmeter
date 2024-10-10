package customerentity

import (
	"fmt"
	"strings"

	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
)

// NotFoundError represents an error when a resource is not found
var _ error = (*NotFoundError)(nil)

type NotFoundError struct {
	CustomerID
}

func (e NotFoundError) Error() string {
	return fmt.Sprintf("resource with id %s not found in %s namespace", e.ID, e.Namespace)
}

// ValidationError represents an error when a validation fails
var _ error = (*ValidationError)(nil)

type ValidationError genericError

func (e ValidationError) Error() string {
	return e.Err.Error()
}

func (e ValidationError) Unwrap() error {
	return e.Err
}

// UpdateAfterDeleteError represents an error when an update is attempted after a delete
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

// CustomerAppError represents an error when an app is not found
var _ error = (*CustomerAppError)(nil)

type CustomerAppError struct {
	AppID   appentitybase.AppID
	AppType appentitybase.AppType
	Err     error
}

func (e CustomerAppError) Error() string {
	return fmt.Sprintf("app %s type with id %s in namespace %s: %s", e.AppType, e.AppID.ID, e.AppID.Namespace, e.Err.Error())
}

// genericError represents a generic error
type genericError struct {
	Err error
}
