package subscription

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

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
	PatchOperationAdd    PatchOperation = "add"
	PatchOperationRemove PatchOperation = "remove"
	PatchOperationExtend PatchOperation = "extend"
)

func (o PatchOperation) Validate() error {
	switch o {
	case PatchOperationAdd, PatchOperationRemove, PatchOperationExtend:
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
	Op() PatchOperation
	Path() PatchPath
}

type ValuePatch[T any] interface {
	Patch
	Value() T
}

func ToApplies(p Patch, _ int) Applies {
	return p
}

type PatchAddItem struct {
	PhaseKey    string
	ItemKey     string
	CreateInput SubscriptionItemSpec
}

func (a PatchAddItem) Op() PatchOperation {
	return PatchOperationAdd
}

func (a PatchAddItem) Path() PatchPath {
	return NewItemPath(a.PhaseKey, a.ItemKey)
}

func (a PatchAddItem) Value() SubscriptionItemSpec {
	return a.CreateInput
}

var _ ValuePatch[SubscriptionItemSpec] = PatchAddItem{}

func (a PatchAddItem) ApplyTo(spec *SubscriptionSpec, actx ApplyContext) error {
	phase, ok := spec.Phases[a.PhaseKey]
	if !ok {
		return &PatchValidationError{Msg: fmt.Sprintf("phase %s not found", a.PhaseKey)}
	}

	if _, exists := phase.Items[a.ItemKey]; exists {
		return &PatchConflictError{Msg: fmt.Sprintf("item %s already exists in phase %s", a.ItemKey, a.PhaseKey)}
	}

	// Checks we need:
	// 1. You cannot add items to previous phases
	if actx.Operation == SpecOperationEdit {
		currentPhase, exists := spec.GetCurrentPhaseAt(actx.CurrentTime)
		if !exists {
			// either all phases are in the past or in the future
			// if all phases are in the future then any addition is possible
			// if all phases are in the past then no addition is possible
			//
			// If all phases are in the past then the selected one is also in the past
			if spec.ActiveFrom.Add(phase.StartAfter).Before(actx.CurrentTime) {
				return &PatchForbiddenError{Msg: fmt.Sprintf("cannot add item to phase %s which starts before current phase", a.PhaseKey)}
			}
		} else {
			if phase.StartAfter < currentPhase.StartAfter {
				return &PatchForbiddenError{Msg: fmt.Sprintf("cannot add item to phase %s which starts before current phase", a.PhaseKey)}
			}
		}
	}

	phase.Items[a.ItemKey] = &a.CreateInput
	return nil
}

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

func (r PatchRemoveItem) ApplyTo(spec *SubscriptionSpec, actx ApplyContext) error {
	phase, ok := spec.Phases[r.PhaseKey]
	if !ok {
		return &PatchValidationError{Msg: fmt.Sprintf("phase %s not found", r.PhaseKey)}
	}

	if _, exists := phase.Items[r.ItemKey]; !exists {
		return &PatchConflictError{Msg: fmt.Sprintf("item %s already exists in phase %s", r.ItemKey, r.PhaseKey)}
	}

	// Checks we need:
	// 1. You cannot remove items from previous phases
	if actx.Operation == SpecOperationEdit {
		currentPhase, exists := spec.GetCurrentPhaseAt(actx.CurrentTime)
		if !exists {
			// either all phases are in the past or in the future
			// if all phases are in the future then any addition is possible
			// if all phases are in the past then no addition is possible
			//
			// If all phases are in the past then the selected one is also in the past
			if spec.ActiveFrom.Add(phase.StartAfter).Before(actx.CurrentTime) {
				return &PatchForbiddenError{Msg: fmt.Sprintf("cannot remove item from phase %s which starts before current phase", r.PhaseKey)}
			}
		} else {
			if phase.StartAfter < currentPhase.StartAfter {
				return &PatchForbiddenError{Msg: fmt.Sprintf("cannot remove item from phase %s which starts before current phase", r.PhaseKey)}
			}
		}
	}

	delete(phase.Items, r.ItemKey)
	return nil
}

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

func (a PatchAddPhase) ApplyTo(spec *SubscriptionSpec, actx ApplyContext) error {
	if _, exists := spec.Phases[a.PhaseKey]; exists {
		return fmt.Errorf("phase %s already exists", a.PhaseKey)
	}

	// TODO: Adding phases needs to be better defined.
	// Should it delay all subsequent phases or just cut the line shortening the adjacent phases?
	// Can it be any time in the future or should it be after the last phase?
	// For now lets go with a strict approach where adding a phase can only be the last phase.
	// Checks we need:
	// 1. You can only add a phase in edits
	if actx.Operation != SpecOperationEdit {
		return &PatchForbiddenError{Msg: "you can only add a phase in edit"}
	}
	// 2. You can only add a phase as the last phase
	sortedPhases := spec.GetSortedPhases()
	lastPhase := sortedPhases[len(sortedPhases)-1]
	if lastPhase.StartAfter >= a.Value().StartAfter {
		return &PatchForbiddenError{Msg: "cannot add phase before the last phase"}
	}
	// 3. You can only add a phase before the subscription ends
	if spec.ActiveTo != nil && !spec.ActiveFrom.Add(a.Value().StartAfter).Before(*spec.ActiveTo) {
		return &PatchForbiddenError{Msg: "cannot add phase after the subscription ends"}
	}

	spec.Phases[a.PhaseKey] = &SubscriptionPhaseSpec{
		CreateSubscriptionPhaseInput: a.CreateInput,
	}
	return nil
}

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

func (r PatchRemovePhase) ApplyTo(spec *SubscriptionSpec, actx ApplyContext) error {
	phase, exists := spec.Phases[r.PhaseKey]
	if !exists {
		return fmt.Errorf("phase %s not found", r.PhaseKey)
	}

	// Checks we need:
	// 1. You can only remove a phase in edit
	if actx.Operation != SpecOperationEdit {
		return &PatchForbiddenError{Msg: "you can only remove a phase in edit"}
	}
	// 2. You can only remove future phases
	if !spec.ActiveFrom.Add(phase.StartAfter).After(actx.CurrentTime) {
		return &PatchForbiddenError{Msg: "cannot remove already started phase"}
	}

	delete(spec.Phases, r.PhaseKey)
	return nil
}

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

// FIXME:
// We can't store duration as time.Duration in the spec if it's provided as ISO8601.
// The reason is ISO8601 => time.Duration would yield different results for the start time of each phase we're extending.
//
// Either
// 1. We can store time.Duration as it gets transalted for the targe phase, then convert back and forth and reapply for all phases (quite hacky)
// 2. We store it as ISO string
//
// Furthermore, we should check if go Date normalization behaves as expected (e.g. shifts in day of the month values when extending by months)
func (e PatchExtendPhase) ApplyTo(spec *SubscriptionSpec, actx ApplyContext) error {
	phase, ok := spec.Phases[e.PhaseKey]
	if !ok {
		return fmt.Errorf("phase %s not found", e.PhaseKey)
	}

	sortedPhases := spec.GetSortedPhases()

	// Checks we need:
	// 1. You can only extend a phase in edit?
	if actx.Operation != SpecOperationEdit {
		return &PatchForbiddenError{Msg: "you can only extend a phase in edit"}
	}
	// 2. You cannot extend past phases, only current or future ones
	current, exists := spec.GetCurrentPhaseAt(actx.CurrentTime)
	if exists {
		if phase.StartAfter < current.StartAfter {
			return &PatchForbiddenError{Msg: "cannot extend past phase"}
		}
	} else {
		// If current phase doesn't exist then all phases are either in the past or in the future
		// If they're all in the past then the by checking any we can see if it should fail or not
		if spec.ActiveFrom.Add(phase.StartAfter).Before(actx.CurrentTime) {
			return &PatchForbiddenError{Msg: "cannot extend past phase"}
		}
	}

	reachedTargetPhase := false
	for i, p := range sortedPhases {
		if p.PhaseKey == e.PhaseKey {
			reachedTargetPhase = true
		}

		if reachedTargetPhase {
			sortedPhases[i].StartAfter += e.Duration
		}
	}

	return nil
}
