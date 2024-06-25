package staticentitlement

import (
	"encoding/json"
	"fmt"

	"github.com/openmeterio/openmeter/internal/entitlement"
)

type Entitlement struct {
	entitlement.GenericProperties

	Config string `json:"config,omitempty"`
}

// Attempts to parse the JSON string in `Config` to map[string]interface{}
// and returns it's value
func (e *Entitlement) ParseConfig() (map[string]interface{}, error) {
	if len(e.Config) == 0 {
		return nil, fmt.Errorf("Config is empty")
	}

	result := make(map[string]interface{})
	json.Unmarshal([]byte(e.Config), &result)
	return result, nil
}

func ParseFromGenericEntitlement(model *entitlement.Entitlement) (*Entitlement, error) {
	if model.EntitlementType != entitlement.EntitlementTypeStatic {
		return nil, &entitlement.WrongTypeError{Expected: entitlement.EntitlementTypeStatic, Actual: model.EntitlementType}
	}

	if model.Config == nil {
		return nil, &entitlement.InvalidValueError{Type: model.EntitlementType, Message: "Config is required"}
	}

	return &Entitlement{
		GenericProperties: model.GenericProperties,
		Config:            *model.Config,
	}, nil
}
