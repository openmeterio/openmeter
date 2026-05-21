package featuregate

import (
	"encoding/json"

	"github.com/google/uuid"
)

type Org interface {
	Context

	ID() *uuid.UUID
	PortalID() *uuid.UUID
	OrgName() *string
	FeatureSet() *string
	Tier() *string
}

type Context interface {
	Key() string
	Kind() string
	Anonymous() bool
	GetCustomAttributes() map[string]any
	AddCustomAttribute(name string, value any)
}

type Gate interface {
	EvaluateBool(flag string, defaultValue bool) (bool, error)
	EvaluateInt(flag string, defaultValue int) (int, error)
	EvaluateFloat64(flag string, defaultValue float64) (float64, error)
	EvaluateString(flag string, defaultValue string) (string, error)
	EvaluateJSON(flag string, defaultValue json.RawMessage) (json.RawMessage, error)

	WithOrg(org Org) (Gate, error)
	WithFFContext(custom ...Context) (Gate, error)
}

func NewNoop() Gate {
	return Noop{}
}

type Noop struct{}

func (n Noop) EvaluateBool(string, bool) (bool, error) {
	return true, nil
}

func (n Noop) EvaluateInt(string, int) (int, error) {
	return 0, nil
}

func (n Noop) EvaluateFloat64(string, float64) (float64, error) {
	return 0, nil
}

func (n Noop) EvaluateString(string, string) (string, error) {
	return "", nil
}

func (n Noop) EvaluateJSON(string, json.RawMessage) (json.RawMessage, error) {
	return json.RawMessage(`{}`), nil
}

func (n Noop) WithFFContext(custom ...Context) (Gate, error) {
	return Noop{}, nil
}

func (n Noop) WithOrg(org Org) (Gate, error) {
	return n.WithFFContext(org)
}
