package featuregate

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/samber/lo"
)

const (
	FlagMeteringPrepaidCredits = "metering-prepaid-credits"
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

type NamespaceOrg string

var _ Org = NamespaceOrg("")

func (n NamespaceOrg) AddCustomAttribute(name string, value any) {}

func (n NamespaceOrg) Anonymous() bool {
	return false
}

func (n NamespaceOrg) FeatureSet() *string {
	return nil
}

func (n NamespaceOrg) GetCustomAttributes() map[string]any {
	return nil
}

func (n NamespaceOrg) ID() *uuid.UUID {
	return lo.ToPtr(uuid.NewSHA1(uuid.NameSpaceURL, []byte(n)))
}

func (n NamespaceOrg) Key() string {
	return string(n)
}

func (n NamespaceOrg) Kind() string {
	return "namespace"
}

func (n NamespaceOrg) OrgName() *string {
	return lo.ToPtr(string(n))
}

func (n NamespaceOrg) PortalID() *uuid.UUID {
	return n.ID()
}

func (n NamespaceOrg) Tier() *string {
	return nil
}
