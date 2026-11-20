package eventtypes

import (
	"bytes"
	"encoding/json"
	"fmt"

	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	v4 "github.com/openmeterio/openmeter/openmeter/notification/v4"
)

func ToAPIBillingNotificationEventTypeSummary(d v4.EventTypeDefinition) apiv3models.BillingNotificationEventTypeSummary {
	out := apiv3models.BillingNotificationEventTypeSummary{
		Type:        d.EventType.String(),
		Description: d.Description,
	}

	if d.DeprecatedBy != nil {
		deprecatedBy := d.DeprecatedBy.String()
		out.DeprecatedBy = &deprecatedBy
	}

	return out
}

func ToAPIBillingNotificationEventTypeDefinition(d v4.EventTypeDefinition) (apiv3models.BillingNotificationEventTypeDefinition, error) {
	summary := ToAPIBillingNotificationEventTypeSummary(d)

	payload, err := toJSONSchemaObject(d.PayloadJSONSchema)
	if err != nil {
		return apiv3models.BillingNotificationEventTypeDefinition{}, fmt.Errorf("payload JSON schema of %s: %w", d.EventType, err)
	}

	out := apiv3models.BillingNotificationEventTypeDefinition{
		Type:              summary.Type,
		Description:       summary.Description,
		DeprecatedBy:      summary.DeprecatedBy,
		PayloadJsonSchema: payload,
	}

	if d.Rule != nil {
		ruleConfig, err := toJSONSchemaObject(d.Rule.JSONSchema)
		if err != nil {
			return apiv3models.BillingNotificationEventTypeDefinition{}, fmt.Errorf("rule config JSON schema of %s: %w", d.EventType, err)
		}

		out.RuleConfigJsonSchema = &ruleConfig
	}

	return out, nil
}

func toJSONSchemaObject(schema json.RawMessage) (map[string]interface{}, error) {
	decoder := json.NewDecoder(bytes.NewReader(schema))
	// Keep numbers as written, float64 would change large or precise values.
	decoder.UseNumber()

	var out map[string]interface{}
	if err := decoder.Decode(&out); err != nil {
		return nil, err
	}

	return out, nil
}
