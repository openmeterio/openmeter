package patch

import (
	"encoding/json"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/isodate"
)

// wPatch is used to serialize patches
type wPatch struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value any    `json:"value,omitempty"`
}

// Serialization of patches

func asWPatch(val subscription.Patch) *wPatch {
	p := &wPatch{
		Op:   string(val.Op()),
		Path: string(val.Path()),
	}

	if v, ok := val.(subscription.AnyValuePatch); ok {
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

func (p PatchStretchPhase) MarshalJSON() ([]byte, error) {
	return json.Marshal(asWPatch(p))
}

type rPatch struct {
	Op    string          `json:"op"`
	Path  string          `json:"path"`
	Value json.RawMessage `json:"value,omitempty"`
}

// TODO: patch serialization currently is only ever needed for parsing API requests
// The internal patch types (these) don't properly match up with the API types so they're explicitly mapped in the httpdriver package
// In conclusion, this serialization is redundant

// Deserialization of patches
func Deserialize(b []byte) (any, error) {
	p := &rPatch{}
	if err := json.Unmarshal(b, p); err != nil {
		return nil, err
	}

	pPath := subscription.PatchPath(p.Path)
	if err := pPath.Validate(); err != nil {
		return nil, fmt.Errorf("invalid patch path: %w", err)
	}

	pOp := subscription.PatchOperation(p.Op)
	if err := pOp.Validate(); err != nil {
		return nil, fmt.Errorf("invalid patch operation: %w", err)
	}

	if pPath.Type() == subscription.PatchPathTypePhase && pOp == subscription.PatchOperationAdd {
		var val *subscription.CreateSubscriptionPhaseInput

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
	} else if pPath.Type() == subscription.PatchPathTypePhase && pOp == subscription.PatchOperationRemove {
		var val *subscription.RemoveSubscriptionPhaseInput

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
	} else if pPath.Type() == subscription.PatchPathTypePhase && pOp == subscription.PatchOperationStretch {
		var val *isodate.Period

		if err := json.Unmarshal(p.Value, &val); err != nil {
			return nil, fmt.Errorf("failed to unmarshal patch value: %w", err)
		}

		if val == nil {
			return nil, fmt.Errorf("patch value is nil")
		}

		return &PatchStretchPhase{
			PhaseKey: pPath.PhaseKey(),
			Duration: *val,
		}, nil
	} else if pPath.Type() == subscription.PatchPathTypeItem && pOp == subscription.PatchOperationAdd {
		var val *subscription.SubscriptionItemSpec

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
	} else if pPath.Type() == subscription.PatchPathTypeItem && pOp == subscription.PatchOperationRemove {
		return &PatchRemoveItem{
			PhaseKey: pPath.PhaseKey(),
			ItemKey:  pPath.ItemKey(),
		}, nil
	}

	return nil, fmt.Errorf("unsupported patch operation: %s %s", pPath.Type(), pOp)
}
