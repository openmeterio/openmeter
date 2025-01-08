package subscription

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/samber/lo"
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
	PatchOperationAdd     PatchOperation = "add"
	PatchOperationRemove  PatchOperation = "remove"
	PatchOperationStretch PatchOperation = "stretch"
)

func (o PatchOperation) Validate() error {
	switch o {
	case PatchOperationAdd, PatchOperationRemove, PatchOperationStretch:
		return nil
	default:
		return fmt.Errorf("invalid patch operation: %s", o)
	}
}

type PatchPath string

const (
	phasePathPrefix = "phases"
	itemPathPrefix  = "items"
)

type PatchPathType string

const (
	PatchPathTypePhase PatchPathType = "phase"
	PatchPathTypeItem  PatchPathType = "item"
)

// Lets implement JSON Unmarshaler for Path
func (p *PatchPath) UnmarshalJSON(data []byte) error {
	if err := PatchPath(data).Validate(); err != nil {
		return fmt.Errorf("path validation failed: %s", err)
	}

	*p = PatchPath(data)
	return nil
}

// Lets implement JSON Marshaler for Path
func (p PatchPath) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, p)), nil
}

func (p PatchPath) seg() []string {
	// For a properly formatted path the first segment is empty
	return strings.Split(string(p), "/")[1:]
}

// Checks whether p is a parent of other where parent means all segments of p are present and in order in other
func (p PatchPath) IsParentOf(other PatchPath) bool {
	segments := p.seg()
	otherSegments := other.seg()

	if len(otherSegments) < len(segments) {
		return false
	}

	for i, s := range segments {
		if otherSegments[i] != s {
			return false
		}
	}

	return true
}

// Lets implement validation for Path
func (p PatchPath) Validate() error {
	strVal := string(p)

	if !strings.HasPrefix(strVal, "/") {
		return &PatchValidationError{
			Msg: fmt.Sprintf("invalid path: %s, should start with /", strVal),
		}
	}

	segments := p.seg()
	if len(segments) != 2 && len(segments) != 4 {
		return &PatchValidationError{
			Msg: fmt.Sprintf("invalid path: %s, should have 2 or 4 segments, has %d", strVal, len(segments)),
		}
	}

	if segments[0] != phasePathPrefix {
		return &PatchValidationError{Msg: fmt.Sprintf("invalid path: %s, first segment should be %s", strVal, phasePathPrefix)}
	}

	if len(segments) == 4 && segments[2] != itemPathPrefix {
		return &PatchValidationError{Msg: fmt.Sprintf("invalid path: %s, third segment should be %s", strVal, itemPathPrefix)}
	}

	if lo.SomeBy(segments, func(s string) bool { return s == "" }) {
		return &PatchValidationError{Msg: fmt.Sprintf("invalid path: %s, segments should not be empty", strVal)}
	}

	return nil
}

func (p PatchPath) Type() PatchPathType {
	if len(p.seg()) == 4 {
		return PatchPathTypeItem
	}

	return PatchPathTypePhase
}

func (p PatchPath) PhaseKey() string {
	return p.seg()[1]
}

func (p PatchPath) ItemKey() string {
	if p.Type() != PatchPathTypeItem {
		return ""
	}

	return p.seg()[3]
}

func NewPhasePath(phaseKey string) PatchPath {
	return PatchPath(fmt.Sprintf("/%s/%s", phasePathPrefix, phaseKey))
}

func NewItemPath(phaseKey, itemKey string) PatchPath {
	return PatchPath(fmt.Sprintf("/%s/%s/%s/%s", phasePathPrefix, phaseKey, itemPathPrefix, itemKey))
}

type Patch interface {
	json.Marshaler
	Applies
	Validate() error
	Op() PatchOperation
	Path() PatchPath
}

type AnyValuePatch interface {
	ValueAsAny() any
}

type ValuePatch[T any] interface {
	Patch
	Value() T
	AnyValuePatch
}

func ToApplies(p Patch, _ int) Applies {
	return p
}
