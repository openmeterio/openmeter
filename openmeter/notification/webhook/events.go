package webhook

// FIXME: add JSON schema for events
// FIXME: refactor notifications to keep these in one place
var NotificationEventTypes = []EventType{
	EventTypeEntitlementsBalanceThreshold,
	EventTypeEntitlementsReset,
	EventTypeInvoiceCreated,
	EventTypeInvoiceUpdated,
}

const (
	EntitlementsEventGroupName = "entitlements"

	EntitlementsBalanceThresholdType        = "entitlements.balance.threshold"
	EntitlementsBalanceThresholdDescription = "Notification event for entitlements balance threshold violations"

	EntitlementResetType        = "entitlements.reset"
	EntitlementResetDescription = "Notification event for entitlement reset events."
)

var EventTypeEntitlementsBalanceThreshold = EventType{
	Name:        EntitlementsBalanceThresholdType,
	Description: EntitlementsBalanceThresholdDescription,
	GroupName:   EntitlementsEventGroupName,
}

var EventTypeEntitlementsReset = EventType{
	Name:        EntitlementResetType,
	Description: EntitlementResetDescription,
	GroupName:   EntitlementsEventGroupName,
}

const (
	InvoiceEventGroupName = "invoice"

	InvoiceCreatedType        = "invoice.created"
	InvoiceCreatedDescription = "Notification event for new invoice created."

	InvoiceUpdatedType        = "invoice.updated"
	InvoiceUpdatedDescription = "Notification event for new invoice updated."
)

var EventTypeInvoiceCreated = EventType{
	Name:        InvoiceCreatedType,
	Description: InvoiceCreatedDescription,
	GroupName:   InvoiceEventGroupName,
}

var EventTypeInvoiceUpdated = EventType{
	Name:        InvoiceUpdatedType,
	Description: InvoiceUpdatedDescription,
	GroupName:   InvoiceEventGroupName,
}
