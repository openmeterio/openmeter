package subscription

import (
	"fmt"
)

type PatchConflictError struct {
	Msg string
}

func (e *PatchConflictError) Error() string {
	return fmt.Sprintf("patch conflict error: %s", e.Msg)
}

type PatchValidationError struct {
	Msg string
}

func (e *PatchValidationError) Error() string {
	return fmt.Sprintf("patch validation error: %s", e.Msg)
}

type PatchForbiddenError struct {
	Msg string
}

func (e *PatchForbiddenError) Error() string {
	return fmt.Sprintf("patch forbidden error: %s", e.Msg)
}

type PatchOperation string

const (
	PatchOperationAdd        PatchOperation = "add"
	PatchOperationRemove     PatchOperation = "remove"
	PatchOperationUnschedule PatchOperation = "unschedule"
	PatchOperationStretch    PatchOperation = "stretch"
)

func (o PatchOperation) Validate() error {
	switch o {
	case PatchOperationAdd, PatchOperationRemove, PatchOperationStretch, PatchOperationUnschedule:
		return nil
	default:
		return fmt.Errorf("invalid patch operation: %s", o)
	}
}

type Patch interface {
	AppliesToSpec
	Validate() error
	Op() PatchOperation
	Path() SpecPath
}

type AnyValuePatch interface {
	ValueAsAny() any
}

type ValuePatch[T any] interface {
	Patch
	Value() T
	AnyValuePatch
}

func ToApplies(p Patch, _ int) AppliesToSpec {
	return p
}
