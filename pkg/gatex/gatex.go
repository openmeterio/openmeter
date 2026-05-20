package gatex

import (
	"encoding/json"
)

type FeatureGate interface {
	EvaluateBool(flag string, defaultValue bool) (bool, error)
	EvaluateInt(flag string, defaultValue int) (int, error)
	EvaluateFloat64(flag string, defaultValue float64) (float64, error)
	EvaluateString(flag string, defaultValue string) (string, error)
	EvaluateJSON(flag string, defaultValue json.RawMessage) (json.RawMessage, error)
}

var instance FeatureGate = NoopFeatureGate{}

func GetDefault() FeatureGate {
	if instance == nil {
		instance = NoopFeatureGate{}
	}
	return instance
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
