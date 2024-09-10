package subscription

import (
	"fmt"
	"strings"
	"time"

	"github.com/samber/lo"
)

type PatchOperation string

const (
	PatchOperationAdd    PatchOperation = "add"
	PatchOperationRemove PatchOperation = "remove"
	PatchOperationExtend PatchOperation = "extend"
)

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

// Lets implement validation for Path
func (p PatchPath) Validate() error {
	strVal := string(p)

	if !strings.HasPrefix(strVal, "/") {
		return fmt.Errorf("invalid path: %s, should start with /", strVal)
	}

	segments := p.seg()
	if len(segments) != 2 || len(segments) != 4 {
		return fmt.Errorf("invalid path: %s, should have 2 or 4 segments, has %d", strVal, len(segments))
	}

	if segments[0] != phasePathPrefix {
		return fmt.Errorf("invalid path: %s, first segment should be %s", strVal, phasePathPrefix)
	}

	if len(segments) == 4 && segments[2] != itemPathPrefix {
		return fmt.Errorf("invalid path: %s, third segment should be %s", strVal, itemPathPrefix)
	}

	if lo.SomeBy(segments, func(s string) bool { return s == "" }) {
		return fmt.Errorf("invalid path: %s, segments should not be empty", strVal)
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
	Op() PatchOperation
	Path() PatchPath
}

type ValuePatch[T any] interface {
	Patch
	Value() T
}

type PatchAddItem struct {
	PhaseKey    string
	ItemKey     string
	CreateInput CreateSubscriptionItemInput
}

func (a PatchAddItem) Op() PatchOperation {
	return PatchOperationAdd
}

func (a PatchAddItem) Path() PatchPath {
	return NewItemPath(a.PhaseKey, a.ItemKey)
}

func (a PatchAddItem) Value() CreateSubscriptionItemInput {
	return a.CreateInput
}

var _ ValuePatch[CreateSubscriptionItemInput] = PatchAddItem{}

type PatchRemoveItem struct {
	PhaseKey string
	ItemKey  string
}

func (r PatchRemoveItem) Op() PatchOperation {
	return PatchOperationRemove
}

func (r PatchRemoveItem) Path() PatchPath {
	return NewItemPath(r.PhaseKey, r.ItemKey)
}

var _ Patch = PatchRemoveItem{}

type PatchAddPhase struct {
	PhaseKey    string
	CreateInput CreateSubscriptionPhaseInput
}

func (a PatchAddPhase) Op() PatchOperation {
	return PatchOperationAdd
}

func (a PatchAddPhase) Path() PatchPath {
	return NewPhasePath(a.PhaseKey)
}

func (a PatchAddPhase) Value() CreateSubscriptionPhaseInput {
	return a.CreateInput
}

var _ ValuePatch[CreateSubscriptionPhaseInput] = PatchAddPhase{}

type PatchRemovePhase struct {
	PhaseKey string
}

func (r PatchRemovePhase) Op() PatchOperation {
	return PatchOperationRemove
}

func (r PatchRemovePhase) Path() PatchPath {
	return NewPhasePath(r.PhaseKey)
}

var _ Patch = PatchRemovePhase{}

type PatchExtendPhase struct {
	PhaseKey string
	Duration time.Duration
}

func (e PatchExtendPhase) Op() PatchOperation {
	return PatchOperationExtend
}

func (e PatchExtendPhase) Path() PatchPath {
	return NewPhasePath(e.PhaseKey)
}

func (e PatchExtendPhase) Value() time.Duration {
	return e.Duration
}

var _ ValuePatch[time.Duration] = PatchExtendPhase{}
