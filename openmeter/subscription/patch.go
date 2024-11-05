package subscription

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/datex"
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
	At() time.Time
}

type ValuePatch[T any] interface {
	Patch
	Value() T
	AnyValuePatch
}

func ToApplies(p Patch, _ int) Applies {
	return p
}

func SetAt(at time.Time, p Patch) (Patch, error) {
	switch v := p.(type) {
	case PatchAddItem:
		v.AppliedAt = at
		return v, nil
	case PatchRemoveItem:
		v.AppliedAt = at
		return v, nil
	case PatchAddPhase:
		v.AppliedAt = at
		return v, nil
	case PatchRemovePhase:
		v.AppliedAt = at
		return v, nil
	case PatchExtendPhase:
		v.AppliedAt = at
		return v, nil
	}

	return nil, fmt.Errorf("unsupported patch type when setting applied at: %T", p)
}

type exec struct {
	Applies
	exec func() error
}

func (e exec) Exec() error {
	return e.exec()
}

type PatchAddItem struct {
	PhaseKey    string
	ItemKey     string
	CreateInput SubscriptionItemSpec
	AppliedAt   time.Time
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

func (a PatchAddItem) At() time.Time {
	return a.AppliedAt
}

func (a PatchAddItem) ValueAsAny() any {
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
		currentPhase, exists := spec.GetCurrentPhaseAt(a.At())
		if !exists {
			// either all phases are in the past or in the future
			// if all phases are in the future then any addition is possible
			// if all phases are in the past then no addition is possible
			//
			// If all phases are in the past then the selected one is also in the past
			if st, _ := phase.StartAfter.AddTo(spec.ActiveFrom); st.Before(a.At()) {
				return &PatchForbiddenError{Msg: fmt.Sprintf("cannot add item to phase %s which starts before current phase", a.PhaseKey)}
			}
		} else {
			pST, _ := phase.StartAfter.AddTo(spec.ActiveFrom)
			cPST, _ := currentPhase.StartAfter.AddTo(spec.ActiveFrom)
			if pST.Before(cPST) {
				return &PatchForbiddenError{Msg: fmt.Sprintf("cannot add item to phase %s which starts before current phase", a.PhaseKey)}
			}
		}
	}

	phase.Items[a.ItemKey] = &a.CreateInput
	return nil
}

type PatchRemoveItem struct {
	PhaseKey  string
	ItemKey   string
	AppliedAt time.Time
}

func (r PatchRemoveItem) Op() PatchOperation {
	return PatchOperationRemove
}

func (r PatchRemoveItem) Path() PatchPath {
	return NewItemPath(r.PhaseKey, r.ItemKey)
}

func (r PatchRemoveItem) At() time.Time {
	return r.AppliedAt
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
		currentPhase, exists := spec.GetCurrentPhaseAt(r.At())
		if !exists {
			// either all phases are in the past or in the future
			// if all phases are in the future then any addition is possible
			// if all phases are in the past then no addition is possible
			//
			// If all phases are in the past then the selected one is also in the past
			if st, _ := phase.StartAfter.AddTo(spec.ActiveFrom); st.Before(r.At()) {
				return &PatchForbiddenError{Msg: fmt.Sprintf("cannot remove item from phase %s which starts before current phase", r.PhaseKey)}
			}
		} else {
			pST, _ := phase.StartAfter.AddTo(spec.ActiveFrom)
			cPST, _ := currentPhase.StartAfter.AddTo(spec.ActiveFrom)
			if pST.Before(cPST) {
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
	AppliedAt   time.Time
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

func (a PatchAddPhase) ValueAsAny() any {
	return a.CreateInput
}

func (r PatchAddPhase) At() time.Time {
	return r.AppliedAt
}

var _ ValuePatch[CreateSubscriptionPhaseInput] = PatchAddPhase{}

func (a PatchAddPhase) ApplyTo(spec *SubscriptionSpec, actx ApplyContext) error {
	if _, exists := spec.Phases[a.PhaseKey]; exists {
		return &PatchConflictError{Msg: fmt.Sprintf("phase %s already exists", a.PhaseKey)}
	}

	// Checks we need:
	// 1. You can only add a phase in edits
	if actx.Operation != SpecOperationEdit {
		return &PatchForbiddenError{Msg: "you can only add a phase in edit"}
	}

	vST, _ := a.Value().StartAfter.AddTo(spec.ActiveFrom)

	// 3. You can only add a phase before the subscription ends
	if spec.ActiveTo != nil && !vST.Before(*spec.ActiveTo) {
		return &PatchForbiddenError{Msg: "cannot add phase after the subscription ends"}
	}

	// Let's apply the patch

	// Let's get all later phases & make sure their start times is aligned based on the new phase's duration:
	// 1. The very next phase should start based on the new phase's duration
	// 2. All other phases should preserve their relative start times i.e. spacing
	// To achieve this, we determine the difference between the next already scheduled phase's start and the duration, then add that difference to all later phases. Note that this difference is signed.

	sortedPhases := spec.GetSortedPhases()
	var diff datex.Period

	for i := range sortedPhases {
		p := sortedPhases[i]
		// We use !.Before() cause we might insert the phase at the same time another one starts
		if v, _ := p.StartAfter.AddTo(spec.ActiveFrom); !v.Before(vST) && diff.IsZero() {
			tillNextPhase, err := p.StartAfter.Subtract(a.Value().StartAfter)
			if err != nil {
				return fmt.Errorf("failed to calculate difference between phases: %w", err)
			}
			diff, err = a.Value().Duration.Subtract(tillNextPhase)
			if err != nil {
				return fmt.Errorf("failed to calculate difference between phases: %w", err)
			}
		}

		// Once we've reached the next phase lets increment the StartAfter by diff
		if !diff.IsZero() {
			sa, err := p.StartAfter.Add(diff)
			if err != nil {
				return fmt.Errorf("failed to adjust phase %s start time: %w", p.PhaseKey, err)
			}
			sortedPhases[i].StartAfter = sa
		}
	}

	// And then let's add the new phase
	spec.Phases[a.PhaseKey] = &SubscriptionPhaseSpec{
		CreateSubscriptionPhaseInput: a.CreateInput,
		Items:                        make(map[string]*SubscriptionItemSpec),
	}

	return nil
}

type PatchRemovePhase struct {
	PhaseKey    string
	RemoveInput RemoveSubscriptionPhaseInput
	AppliedAt   time.Time
}

func (r PatchRemovePhase) Op() PatchOperation {
	return PatchOperationRemove
}

func (r PatchRemovePhase) Path() PatchPath {
	return NewPhasePath(r.PhaseKey)
}

func (r PatchRemovePhase) At() time.Time {
	return r.AppliedAt
}

func (r PatchRemovePhase) Value() RemoveSubscriptionPhaseInput {
	return r.RemoveInput
}

func (r PatchRemovePhase) ValueAsAny() any {
	return r.RemoveInput
}

var _ ValuePatch[RemoveSubscriptionPhaseInput] = PatchRemovePhase{}

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
	if st, _ := phase.StartAfter.AddTo(spec.ActiveFrom); !st.After(r.At()) {
		return &PatchForbiddenError{Msg: "cannot remove already started phase"}
	}

	// And lets honor the shift behavior.
	switch r.RemoveInput.Shift {
	case RemoveSubscriptionPhaseShiftNext:
		// Let's find all subsequent phases and shift them back by the duration of the original phase
		sortedPhases := spec.GetSortedPhases()

		// We have to calculate what to shift by. Note that phase.Duration is misleading, as though it's part of the creation input, it cannot be trusted as it's only present for customizations and edits.
		deletedPhaseStart, _ := phase.StartAfter.AddTo(spec.ActiveFrom)
		var nextPhaseStartAfter datex.Period
		for _, p := range spec.GetSortedPhases() {
			if v, _ := p.StartAfter.AddTo(spec.ActiveFrom); v.After(deletedPhaseStart) {
				nextPhaseStartAfter = p.StartAfter
				break
			}
		}

		if nextPhaseStartAfter.IsZero() {
			// If there is no next phase then we don't need to shift anything
			break
		}

		shift, err := nextPhaseStartAfter.Subtract(phase.StartAfter)
		if err != nil {
			return fmt.Errorf("failed to calculate shift: %w", err)
		}

		reachedTargetPhase := false

		for i, p := range sortedPhases {
			if v, _ := p.StartAfter.AddTo(spec.ActiveFrom); v.After(deletedPhaseStart) {
				reachedTargetPhase = true
			}

			if reachedTargetPhase {
				sa, err := p.StartAfter.Subtract(shift)
				if err != nil {
					return fmt.Errorf("failed to shift phase %s: %w", p.PhaseKey, err)
				}
				sortedPhases[i].StartAfter = sa
			}
		}
		//nolint:gosimple
		break
	case RemoveSubscriptionPhaseShiftPrev:
		// We leave everything as is, the previous phase will fill up the gap
		break
	default:
		return &PatchValidationError{Msg: fmt.Sprintf("invalid shift behavior: %T", r.RemoveInput.Shift)}
	}

	// Then let's remove the phase
	delete(spec.Phases, r.PhaseKey)

	return nil
}

type PatchExtendPhase struct {
	PhaseKey  string
	Duration  datex.Period
	AppliedAt time.Time
}

func (e PatchExtendPhase) Op() PatchOperation {
	return PatchOperationExtend
}

func (e PatchExtendPhase) Path() PatchPath {
	return NewPhasePath(e.PhaseKey)
}

func (r PatchExtendPhase) At() time.Time {
	return r.AppliedAt
}

func (e PatchExtendPhase) Value() datex.Period {
	return e.Duration
}

func (e PatchExtendPhase) ValueAsAny() any {
	return e.Duration
}

var _ ValuePatch[datex.Period] = PatchExtendPhase{}

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
	pST, _ := phase.StartAfter.AddTo(spec.ActiveFrom)
	// 2. You cannot extend past phases, only current or future ones
	current, exists := spec.GetCurrentPhaseAt(e.At())
	if exists {
		cPST, _ := current.StartAfter.AddTo(spec.ActiveFrom)

		if pST.Before(cPST) {
			return &PatchForbiddenError{Msg: "cannot extend past phase"}
		}
	} else {
		// If current phase doesn't exist then all phases are either in the past or in the future
		// If they're all in the past then the by checking any we can see if it should fail or not
		if pST.Before(e.At()) {
			return &PatchForbiddenError{Msg: "cannot extend past phase"}
		}
	}

	reachedTargetPhase := false
	for i, p := range sortedPhases {
		if p.PhaseKey == e.PhaseKey {
			reachedTargetPhase = true
		}

		if reachedTargetPhase {
			// Adding durtions in the semantic way (using ISO8601 format)
			sa, err := p.StartAfter.Add(e.Duration)
			if err != nil {
				return &PatchValidationError{Msg: fmt.Sprintf("failed to extend phase %s: %s", p.PhaseKey, err)}
			}
			sortedPhases[i].StartAfter = sa
		}
	}

	return nil
}
