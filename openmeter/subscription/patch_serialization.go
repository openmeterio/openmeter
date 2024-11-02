package subscription

import (
	"encoding/json"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/datex"
)

type AnyValuePatch interface {
	ValueAsAny() any
}

// wPatch is used to serialize patches
type wPatch struct {
	Op    string `json:"operation"`
	Path  string `json:"path"`
	Value any    `json:"value,omitempty"`
}

// Serialization of patches

func asWPatch(val Patch) *wPatch {
	p := &wPatch{
		Op:   string(val.Op()),
		Path: string(val.Path()),
	}

	if v, ok := val.(AnyValuePatch); ok {
		p.Value = v.ValueAsAny()
	}

	return p
}

func (p PatchAddItem) MarshalJSON() ([]byte, error) {
	return json.Marshal(asWPatch(p))
}

func (p PatchRemoveItem) MarshalJSON() ([]byte, error) {
	return json.Marshal(asWPatch(p))
}

func (p PatchAddPhase) MarshalJSON() ([]byte, error) {
	return json.Marshal(asWPatch(p))
}

func (p PatchRemovePhase) MarshalJSON() ([]byte, error) {
	return json.Marshal(asWPatch(p))
}

func (p PatchExtendPhase) MarshalJSON() ([]byte, error) {
	return json.Marshal(asWPatch(p))
}

type rPatch struct {
	Op    string          `json:"operation"`
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
		var val *RemoveSubscriptionPhaseInput

		if err := json.Unmarshal(p.Value, &val); err != nil {
			return nil, fmt.Errorf("failed to unmarshal patch value: %w", err)
		}

		if val == nil {
			return nil, fmt.Errorf("patch value is nil")
		}
		return &PatchRemovePhase{
			PhaseKey:    pPath.PhaseKey(),
			RemoveInput: *val,
		}, nil
	} else if pPath.Type() == PatchPathTypePhase && pOp == PatchOperationExtend {
		var val *datex.Period

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
