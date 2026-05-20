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

var instance FeatureGate = noopFeatureGate{}

func GetDefault() FeatureGate {
	if instance == nil {
		instance = noopFeatureGate{}
	}
	return instance
}

type noopFeatureGate struct{}

func (n noopFeatureGate) EvaluateBool(string, bool) (bool, error) {
	return true, nil
}

func (n noopFeatureGate) EvaluateInt(string, int) (int, error) {
	return 0, nil
}

func (n noopFeatureGate) EvaluateFloat64(string, float64) (float64, error) {
	return 0, nil
}

func (n noopFeatureGate) EvaluateString(string, string) (string, error) {
	return "", nil
}

func (n noopFeatureGate) EvaluateJSON(string, json.RawMessage) (json.RawMessage, error) {
	return json.RawMessage(`{}`), nil
}
