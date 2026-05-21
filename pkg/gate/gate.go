package gate

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/google/wire"
)

var FeatureGateNoopSet = wire.NewSet(
	NewNoopFeatureGate,
)

type Org interface {
	FGContext

	ID() *uuid.UUID
	PortalID() *uuid.UUID
	OrgName() *string
	FeatureSet() *string
	Tier() *string
}

type FGContext interface {
	Key() string
	Kind() string
	Anonymous() bool
	GetCustomAttributes() map[string]any
	AddCustomAttribute(name string, value any)
}

type FeatureGate interface {
	EvaluateBool(flag string, defaultValue bool) (bool, error)
	EvaluateInt(flag string, defaultValue int) (int, error)
	EvaluateFloat64(flag string, defaultValue float64) (float64, error)
	EvaluateString(flag string, defaultValue string) (string, error)
	EvaluateJSON(flag string, defaultValue json.RawMessage) (json.RawMessage, error)

	WithOrg(org Org) (FeatureGate, error)
	WithFFContext(custom ...FGContext) (FeatureGate, error)
}

func NewNoopFeatureGate() FeatureGate {
	return NoopFeatureGate{}
}

type NoopFeatureGate struct{}

func (n NoopFeatureGate) EvaluateBool(string, bool) (bool, error) {
	return true, nil
}

func (n NoopFeatureGate) EvaluateInt(string, int) (int, error) {
	return 0, nil
}

func (n NoopFeatureGate) EvaluateFloat64(string, float64) (float64, error) {
	return 0, nil
}

func (n NoopFeatureGate) EvaluateString(string, string) (string, error) {
	return "", nil
}

func (n NoopFeatureGate) EvaluateJSON(string, json.RawMessage) (json.RawMessage, error) {
	return json.RawMessage(`{}`), nil
}

func (n NoopFeatureGate) WithFFContext(custom ...FGContext) (FeatureGate, error) {
	return NoopFeatureGate{}, nil
}

func (n NoopFeatureGate) WithOrg(org Org) (FeatureGate, error) {
	return n.WithFFContext(org)
}
