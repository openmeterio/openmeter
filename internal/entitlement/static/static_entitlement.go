package staticentitlement

import (
	"encoding/json"

	"github.com/openmeterio/openmeter/internal/entitlement"
)

type Entitlement struct {
	entitlement.GenericProperties

	Config *string `json:"config,omitempty"`
}

// Attempts to parse the JSON string in `Config` to map[string]interface{}
// and returns it's value
func (e *Entitlement) ParseConfig() (map[string]interface{}, error) {
	if e.Config == nil {
		return nil, nil
	}

	result := make(map[string]interface{})
	json.Unmarshal([]byte(*e.Config), &result)
	return result, nil
}

func ParseFromGenericEntitlement(model *entitlement.Entitlement) (*Entitlement, error) {
	if model.EntitlementType != entitlement.EntitlementTypeStatic {
		return nil, &entitlement.WrongTypeError{Expected: entitlement.EntitlementTypeStatic, Actual: model.EntitlementType}
	}

	return &Entitlement{
		GenericProperties: model.GenericProperties,
		Config:            model.Config,
	}, nil
}
