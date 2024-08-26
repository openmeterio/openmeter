package webhook

import (
	_ "embed"
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
}

// TODO(chrisgacsal): add JSON Schema for entitlements.balance.threshold event type
