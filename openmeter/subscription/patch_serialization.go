package subscription

import (
	"encoding/json"
	"fmt"
	"time"
)

type AnyPatch struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value any    `json:"value,omitempty"`
}

// Serialization of patches

func serialize(val Patch) ([]byte, error) {
	p := &AnyPatch{
		Op:   string(val.Op()),
		Path: string(val.Path()),
	}

	return json.Marshal(p)
}

func serializeVal[T any](val ValuePatch[T]) ([]byte, error) {
	p := &AnyPatch{
		Op:    string(val.Op()),
		Path:  string(val.Path()),
		Value: val.Value(),
	}

	return json.Marshal(p)
}

func (p PatchAddItem) MarshalJSON() ([]byte, error) {
	return serializeVal(p)
}

func (p PatchRemoveItem) MarshalJSON() ([]byte, error) {
	return serialize(p)
}

func (p PatchAddPhase) MarshalJSON() ([]byte, error) {
	return serializeVal(p)
}

func (p PatchRemovePhase) MarshalJSON() ([]byte, error) {
	return serialize(p)
}

func (p PatchExtendPhase) MarshalJSON() ([]byte, error) {
	return serializeVal(p)
}

type rPatch struct {
	Op    string          `json:"op"`
	Path  string          `json:"path"`
	Value json.RawMessage `json:"value,omitempty"`
}

// Deserialization of patches
func Deserialize(b []byte) (any, error) {
	p := &rPatch{}
	if err := json.Unmarshal(b, p); err != nil {
		return nil, err
	}

	pPath := PatchPath(p.Path)
	if err := pPath.Validate(); err != nil {
		return nil, fmt.Errorf("invalid patch path: %w", err)
	}

	pOp := PatchOperation(p.Op)
	if err := pOp.Validate(); err != nil {
		return nil, fmt.Errorf("invalid patch operation: %w", err)
	}

	if pPath.Type() == PatchPathTypePhase && pOp == PatchOperationAdd {
		var val *CreateSubscriptionPhaseInput

		if err := json.Unmarshal(p.Value, &val); err != nil {
			return nil, fmt.Errorf("failed to unmarshal patch value: %w", err)
		}

		if val == nil {
			return nil, fmt.Errorf("patch value is nil")
		}

		return &PatchAddPhase{
			PhaseKey:    pPath.PhaseKey(),
			CreateInput: *val,
		}, nil
	} else if pPath.Type() == PatchPathTypePhase && pOp == PatchOperationRemove {
		return &PatchRemovePhase{
			PhaseKey: pPath.PhaseKey(),
		}, nil
	} else if pPath.Type() == PatchPathTypePhase && pOp == PatchOperationExtend {
		var val *time.Duration

		if err := json.Unmarshal(p.Value, &val); err != nil {
			return nil, fmt.Errorf("failed to unmarshal patch value: %w", err)
		}

		if val == nil {
			return nil, fmt.Errorf("patch value is nil")
		}

		return &PatchExtendPhase{
			PhaseKey: pPath.PhaseKey(),
			Duration: *val,
		}, nil
	} else if pPath.Type() == PatchPathTypeItem && pOp == PatchOperationAdd {
		var val *SubscriptionItemSpec

		if err := json.Unmarshal(p.Value, &val); err != nil {
			return nil, fmt.Errorf("failed to unmarshal patch value: %w", err)
		}

		if val == nil {
			return nil, fmt.Errorf("patch value is nil")
		}

		return &PatchAddItem{
			PhaseKey:    pPath.PhaseKey(),
			ItemKey:     pPath.ItemKey(),
			CreateInput: *val,
		}, nil
	} else if pPath.Type() == PatchPathTypeItem && pOp == PatchOperationRemove {
		return &PatchRemoveItem{
			PhaseKey: pPath.PhaseKey(),
			ItemKey:  pPath.ItemKey(),
		}, nil
	}

	return nil, fmt.Errorf("unsupported patch operation: %s %s", pPath.Type(), pOp)
}
