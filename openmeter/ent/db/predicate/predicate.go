// Code generated by ent, DO NOT EDIT.

package predicate

import (
	"entgo.io/ent/dialect/sql"
)

// Addon is the predicate function for addon builders.
type Addon func(*sql.Selector)

// AddonOrErr calls the predicate only if the error is not nit.
func AddonOrErr(p Addon, err error) Addon {
	return func(s *sql.Selector) {
		if err != nil {
			s.AddError(err)
			return
		}
		p(s)
	}
}

// AddonRateCard is the predicate function for addonratecard builders.
type AddonRateCard func(*sql.Selector)

// AddonRateCardOrErr calls the predicate only if the error is not nit.
func AddonRateCardOrErr(p AddonRateCard, err error) AddonRateCard {
	return func(s *sql.Selector) {
		if err != nil {
			s.AddError(err)
			return
		}
		p(s)
	}
}

// App is the predicate function for dbapp builders.
type App func(*sql.Selector)

// AppCustomer is the predicate function for appcustomer builders.
type AppCustomer func(*sql.Selector)

// AppStripe is the predicate function for appstripe builders.
type AppStripe func(*sql.Selector)

// AppStripeCustomer is the predicate function for appstripecustomer builders.
type AppStripeCustomer func(*sql.Selector)

// BalanceSnapshot is the predicate function for balancesnapshot builders.
type BalanceSnapshot func(*sql.Selector)

// BillingCustomerLock is the predicate function for billingcustomerlock builders.
type BillingCustomerLock func(*sql.Selector)

// BillingCustomerOverride is the predicate function for billingcustomeroverride builders.
type BillingCustomerOverride func(*sql.Selector)

// BillingInvoice is the predicate function for billinginvoice builders.
type BillingInvoice func(*sql.Selector)

// BillingInvoiceDiscount is the predicate function for billinginvoicediscount builders.
type BillingInvoiceDiscount func(*sql.Selector)

// BillingInvoiceFlatFeeLineConfig is the predicate function for billinginvoiceflatfeelineconfig builders.
type BillingInvoiceFlatFeeLineConfig func(*sql.Selector)

// BillingInvoiceLine is the predicate function for billinginvoiceline builders.
type BillingInvoiceLine func(*sql.Selector)

// BillingInvoiceLineDiscount is the predicate function for billinginvoicelinediscount builders.
type BillingInvoiceLineDiscount func(*sql.Selector)

// BillingInvoiceUsageBasedLineConfig is the predicate function for billinginvoiceusagebasedlineconfig builders.
type BillingInvoiceUsageBasedLineConfig func(*sql.Selector)

// BillingInvoiceUsageBasedLineConfigOrErr calls the predicate only if the error is not nit.
func BillingInvoiceUsageBasedLineConfigOrErr(p BillingInvoiceUsageBasedLineConfig, err error) BillingInvoiceUsageBasedLineConfig {
	return func(s *sql.Selector) {
		if err != nil {
			s.AddError(err)
			return
		}
		p(s)
	}
}

// BillingInvoiceValidationIssue is the predicate function for billinginvoicevalidationissue builders.
type BillingInvoiceValidationIssue func(*sql.Selector)

// BillingProfile is the predicate function for billingprofile builders.
type BillingProfile func(*sql.Selector)

// BillingSequenceNumbers is the predicate function for billingsequencenumbers builders.
type BillingSequenceNumbers func(*sql.Selector)

// BillingWorkflowConfig is the predicate function for billingworkflowconfig builders.
type BillingWorkflowConfig func(*sql.Selector)

// Customer is the predicate function for customer builders.
type Customer func(*sql.Selector)

// CustomerSubjects is the predicate function for customersubjects builders.
type CustomerSubjects func(*sql.Selector)

// Entitlement is the predicate function for entitlement builders.
type Entitlement func(*sql.Selector)

// EntitlementOrErr calls the predicate only if the error is not nit.
func EntitlementOrErr(p Entitlement, err error) Entitlement {
	return func(s *sql.Selector) {
		if err != nil {
			s.AddError(err)
			return
		}
		p(s)
	}
}

// Feature is the predicate function for feature builders.
type Feature func(*sql.Selector)

// Grant is the predicate function for dbgrant builders.
type Grant func(*sql.Selector)

// Meter is the predicate function for dbmeter builders.
type Meter func(*sql.Selector)

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

// Plan is the predicate function for plan builders.
type Plan func(*sql.Selector)

// PlanPhase is the predicate function for planphase builders.
type PlanPhase func(*sql.Selector)

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

// SubscriptionItem is the predicate function for subscriptionitem builders.
type SubscriptionItem func(*sql.Selector)

// SubscriptionItemOrErr calls the predicate only if the error is not nit.
func SubscriptionItemOrErr(p SubscriptionItem, err error) SubscriptionItem {
	return func(s *sql.Selector) {
		if err != nil {
			s.AddError(err)
			return
		}
		p(s)
	}
}

// SubscriptionPhase is the predicate function for subscriptionphase builders.
type SubscriptionPhase func(*sql.Selector)

// UsageReset is the predicate function for usagereset builders.
type UsageReset func(*sql.Selector)
