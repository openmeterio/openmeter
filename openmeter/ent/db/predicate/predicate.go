// Code generated by ent, DO NOT EDIT.

package predicate

import (
	"entgo.io/ent/dialect/sql"
)

// App is the predicate function for app builders.
type App func(*sql.Selector)

// AppCustomer is the predicate function for appcustomer builders.
type AppCustomer func(*sql.Selector)

// AppStripe is the predicate function for appstripe builders.
type AppStripe func(*sql.Selector)

// AppStripeCustomer is the predicate function for appstripecustomer builders.
type AppStripeCustomer func(*sql.Selector)

// BalanceSnapshot is the predicate function for balancesnapshot builders.
type BalanceSnapshot func(*sql.Selector)

// BillingCustomerOverride is the predicate function for billingcustomeroverride builders.
type BillingCustomerOverride func(*sql.Selector)

// BillingInvoice is the predicate function for billinginvoice builders.
type BillingInvoice func(*sql.Selector)

// BillingInvoiceLine is the predicate function for billinginvoiceline builders.
type BillingInvoiceLine func(*sql.Selector)

// BillingInvoiceManualLineConfig is the predicate function for billinginvoicemanuallineconfig builders.
type BillingInvoiceManualLineConfig func(*sql.Selector)

// BillingInvoiceValidationIssue is the predicate function for billinginvoicevalidationissue builders.
type BillingInvoiceValidationIssue func(*sql.Selector)

// BillingProfile is the predicate function for billingprofile builders.
type BillingProfile func(*sql.Selector)

// BillingWorkflowConfig is the predicate function for billingworkflowconfig builders.
type BillingWorkflowConfig func(*sql.Selector)

// Customer is the predicate function for customer builders.
type Customer func(*sql.Selector)

// CustomerSubjects is the predicate function for customersubjects builders.
type CustomerSubjects func(*sql.Selector)

// Entitlement is the predicate function for entitlement builders.
type Entitlement func(*sql.Selector)

// Feature is the predicate function for feature builders.
type Feature func(*sql.Selector)

// Grant is the predicate function for dbgrant builders.
type Grant func(*sql.Selector)

// NotificationChannel is the predicate function for notificationchannel builders.
type NotificationChannel func(*sql.Selector)

// NotificationChannelOrErr calls the predicate only if the error is not nit.
func NotificationChannelOrErr(p NotificationChannel, err error) NotificationChannel {
	return func(s *sql.Selector) {
		if err != nil {
			s.AddError(err)
			return
		}
		p(s)
	}
}

// NotificationEvent is the predicate function for notificationevent builders.
type NotificationEvent func(*sql.Selector)

// NotificationEventOrErr calls the predicate only if the error is not nit.
func NotificationEventOrErr(p NotificationEvent, err error) NotificationEvent {
	return func(s *sql.Selector) {
		if err != nil {
			s.AddError(err)
			return
		}
		p(s)
	}
}

// NotificationEventDeliveryStatus is the predicate function for notificationeventdeliverystatus builders.
type NotificationEventDeliveryStatus func(*sql.Selector)

// NotificationRule is the predicate function for notificationrule builders.
type NotificationRule func(*sql.Selector)

// NotificationRuleOrErr calls the predicate only if the error is not nit.
func NotificationRuleOrErr(p NotificationRule, err error) NotificationRule {
	return func(s *sql.Selector) {
		if err != nil {
			s.AddError(err)
			return
		}
		p(s)
	}
}

// Plan is the predicate function for dbplan builders.
type Plan func(*sql.Selector)

// PlanPhase is the predicate function for planphase builders.
type PlanPhase func(*sql.Selector)

// PlanPhaseOrErr calls the predicate only if the error is not nit.
func PlanPhaseOrErr(p PlanPhase, err error) PlanPhase {
	return func(s *sql.Selector) {
		if err != nil {
			s.AddError(err)
			return
		}
		p(s)
	}
}

// PlanRateCard is the predicate function for planratecard builders.
type PlanRateCard func(*sql.Selector)

// PlanRateCardOrErr calls the predicate only if the error is not nit.
func PlanRateCardOrErr(p PlanRateCard, err error) PlanRateCard {
	return func(s *sql.Selector) {
		if err != nil {
			s.AddError(err)
			return
		}
		p(s)
	}
}

// Subscription is the predicate function for subscription builders.
type Subscription func(*sql.Selector)

// SubscriptionPatch is the predicate function for subscriptionpatch builders.
type SubscriptionPatch func(*sql.Selector)

// SubscriptionPatchValueAddItem is the predicate function for subscriptionpatchvalueadditem builders.
type SubscriptionPatchValueAddItem func(*sql.Selector)

// SubscriptionPatchValueAddPhase is the predicate function for subscriptionpatchvalueaddphase builders.
type SubscriptionPatchValueAddPhase func(*sql.Selector)

// SubscriptionPatchValueExtendPhase is the predicate function for subscriptionpatchvalueextendphase builders.
type SubscriptionPatchValueExtendPhase func(*sql.Selector)

// UsageReset is the predicate function for usagereset builders.
type UsageReset func(*sql.Selector)
