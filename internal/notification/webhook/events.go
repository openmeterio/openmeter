package webhook

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

const (
	EntitlementsEventGroupName              = "entitlements"
	EntitlementsBalanceThresholdType        = "entitlements.balance.threshold"
	EntitlementsBalanceThresholdDescription = "Notification event for entitlements balance threshold violations"
)

var NotificationEventTypes = []EventType{
	EntitlementsBalanceThresholdEventType,
}

var EntitlementsBalanceThresholdEventType = EventType{
	Name:        EntitlementsBalanceThresholdType,
	Description: EntitlementsBalanceThresholdDescription,
	GroupName:   EntitlementsEventGroupName,
	Schemas:     EntitlementsBalanceThresholdSchemas,
}

//go:embed schemas/entitlements-balance-threshold.v1.json
var entitlementsBalanceThresholdV1SchemaJSON []byte

var EntitlementsBalanceThresholdSchemas = func() map[string]map[string]interface{} {
	var schemaV1 map[string]interface{}

	if err := json.Unmarshal(entitlementsBalanceThresholdV1SchemaJSON, &schemaV1); err != nil {
		panic(fmt.Errorf("failed to unmarshal entitlements balance threshold schema: %w", err))
	}

	return map[string]map[string]interface{}{
		"1": schemaV1,
	}
}()
