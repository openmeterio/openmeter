// Code generated by ent, DO NOT EDIT.

package db

import (
	"time"

	"github.com/alpacahq/alpacadecimal"
	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datex"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timezone"
)

func (u *AppUpdate) SetOrClearMetadata(value *map[string]string) *AppUpdate {
	if value == nil {
		return u.ClearMetadata()
	}
	return u.SetMetadata(*value)
}

func (u *AppUpdateOne) SetOrClearMetadata(value *map[string]string) *AppUpdateOne {
	if value == nil {
		return u.ClearMetadata()
	}
	return u.SetMetadata(*value)
}

func (u *AppUpdate) SetOrClearDeletedAt(value *time.Time) *AppUpdate {
	if value == nil {
		return u.ClearDeletedAt()
	}
	return u.SetDeletedAt(*value)
}

func (u *AppUpdateOne) SetOrClearDeletedAt(value *time.Time) *AppUpdateOne {
	if value == nil {
		return u.ClearDeletedAt()
	}
	return u.SetDeletedAt(*value)
}

func (u *AppUpdate) SetOrClearDescription(value *string) *AppUpdate {
	if value == nil {
		return u.ClearDescription()
	}
	return u.SetDescription(*value)
}

func (u *AppUpdateOne) SetOrClearDescription(value *string) *AppUpdateOne {
	if value == nil {
		return u.ClearDescription()
	}
	return u.SetDescription(*value)
}

func (u *AppCustomerUpdate) SetOrClearDeletedAt(value *time.Time) *AppCustomerUpdate {
	if value == nil {
		return u.ClearDeletedAt()
	}
	return u.SetDeletedAt(*value)
}

func (u *AppCustomerUpdateOne) SetOrClearDeletedAt(value *time.Time) *AppCustomerUpdateOne {
	if value == nil {
		return u.ClearDeletedAt()
	}
	return u.SetDeletedAt(*value)
}

func (u *AppStripeUpdate) SetOrClearDeletedAt(value *time.Time) *AppStripeUpdate {
	if value == nil {
		return u.ClearDeletedAt()
	}
	return u.SetDeletedAt(*value)
}

func (u *AppStripeUpdateOne) SetOrClearDeletedAt(value *time.Time) *AppStripeUpdateOne {
	if value == nil {
		return u.ClearDeletedAt()
	}
	return u.SetDeletedAt(*value)
}

func (u *AppStripeCustomerUpdate) SetOrClearDeletedAt(value *time.Time) *AppStripeCustomerUpdate {
	if value == nil {
		return u.ClearDeletedAt()
	}
	return u.SetDeletedAt(*value)
}

func (u *AppStripeCustomerUpdateOne) SetOrClearDeletedAt(value *time.Time) *AppStripeCustomerUpdateOne {
	if value == nil {
		return u.ClearDeletedAt()
	}
	return u.SetDeletedAt(*value)
}

func (u *AppStripeCustomerUpdate) SetOrClearStripeDefaultPaymentMethodID(value *string) *AppStripeCustomerUpdate {
	if value == nil {
		return u.ClearStripeDefaultPaymentMethodID()
	}
	return u.SetStripeDefaultPaymentMethodID(*value)
}

func (u *AppStripeCustomerUpdateOne) SetOrClearStripeDefaultPaymentMethodID(value *string) *AppStripeCustomerUpdateOne {
	if value == nil {
		return u.ClearStripeDefaultPaymentMethodID()
	}
	return u.SetStripeDefaultPaymentMethodID(*value)
}

func (u *BalanceSnapshotUpdate) SetOrClearDeletedAt(value *time.Time) *BalanceSnapshotUpdate {
	if value == nil {
		return u.ClearDeletedAt()
	}
	return u.SetDeletedAt(*value)
}

func (u *BalanceSnapshotUpdateOne) SetOrClearDeletedAt(value *time.Time) *BalanceSnapshotUpdateOne {
	if value == nil {
		return u.ClearDeletedAt()
	}
	return u.SetDeletedAt(*value)
}

func (u *BillingCustomerOverrideUpdate) SetOrClearDeletedAt(value *time.Time) *BillingCustomerOverrideUpdate {
	if value == nil {
		return u.ClearDeletedAt()
	}
	return u.SetDeletedAt(*value)
}

func (u *BillingCustomerOverrideUpdateOne) SetOrClearDeletedAt(value *time.Time) *BillingCustomerOverrideUpdateOne {
	if value == nil {
		return u.ClearDeletedAt()
	}
	return u.SetDeletedAt(*value)
}

func (u *BillingCustomerOverrideUpdate) SetOrClearBillingProfileID(value *string) *BillingCustomerOverrideUpdate {
	if value == nil {
		return u.ClearBillingProfileID()
	}
	return u.SetBillingProfileID(*value)
}

func (u *BillingCustomerOverrideUpdateOne) SetOrClearBillingProfileID(value *string) *BillingCustomerOverrideUpdateOne {
	if value == nil {
		return u.ClearBillingProfileID()
	}
	return u.SetBillingProfileID(*value)
}

func (u *BillingCustomerOverrideUpdate) SetOrClearCollectionAlignment(value *billingentity.AlignmentKind) *BillingCustomerOverrideUpdate {
	if value == nil {
		return u.ClearCollectionAlignment()
	}
	return u.SetCollectionAlignment(*value)
}

func (u *BillingCustomerOverrideUpdateOne) SetOrClearCollectionAlignment(value *billingentity.AlignmentKind) *BillingCustomerOverrideUpdateOne {
	if value == nil {
		return u.ClearCollectionAlignment()
	}
	return u.SetCollectionAlignment(*value)
}

func (u *BillingCustomerOverrideUpdate) SetOrClearItemCollectionPeriod(value *datex.ISOString) *BillingCustomerOverrideUpdate {
	if value == nil {
		return u.ClearItemCollectionPeriod()
	}
	return u.SetItemCollectionPeriod(*value)
}

func (u *BillingCustomerOverrideUpdateOne) SetOrClearItemCollectionPeriod(value *datex.ISOString) *BillingCustomerOverrideUpdateOne {
	if value == nil {
		return u.ClearItemCollectionPeriod()
	}
	return u.SetItemCollectionPeriod(*value)
}

func (u *BillingCustomerOverrideUpdate) SetOrClearInvoiceAutoAdvance(value *bool) *BillingCustomerOverrideUpdate {
	if value == nil {
		return u.ClearInvoiceAutoAdvance()
	}
	return u.SetInvoiceAutoAdvance(*value)
}

func (u *BillingCustomerOverrideUpdateOne) SetOrClearInvoiceAutoAdvance(value *bool) *BillingCustomerOverrideUpdateOne {
	if value == nil {
		return u.ClearInvoiceAutoAdvance()
	}
	return u.SetInvoiceAutoAdvance(*value)
}

func (u *BillingCustomerOverrideUpdate) SetOrClearInvoiceDraftPeriod(value *datex.ISOString) *BillingCustomerOverrideUpdate {
	if value == nil {
		return u.ClearInvoiceDraftPeriod()
	}
	return u.SetInvoiceDraftPeriod(*value)
}

func (u *BillingCustomerOverrideUpdateOne) SetOrClearInvoiceDraftPeriod(value *datex.ISOString) *BillingCustomerOverrideUpdateOne {
	if value == nil {
		return u.ClearInvoiceDraftPeriod()
	}
	return u.SetInvoiceDraftPeriod(*value)
}

func (u *BillingCustomerOverrideUpdate) SetOrClearInvoiceDueAfter(value *datex.ISOString) *BillingCustomerOverrideUpdate {
	if value == nil {
		return u.ClearInvoiceDueAfter()
	}
	return u.SetInvoiceDueAfter(*value)
}

func (u *BillingCustomerOverrideUpdateOne) SetOrClearInvoiceDueAfter(value *datex.ISOString) *BillingCustomerOverrideUpdateOne {
	if value == nil {
		return u.ClearInvoiceDueAfter()
	}
	return u.SetInvoiceDueAfter(*value)
}

func (u *BillingCustomerOverrideUpdate) SetOrClearInvoiceCollectionMethod(value *billingentity.CollectionMethod) *BillingCustomerOverrideUpdate {
	if value == nil {
		return u.ClearInvoiceCollectionMethod()
	}
	return u.SetInvoiceCollectionMethod(*value)
}

func (u *BillingCustomerOverrideUpdateOne) SetOrClearInvoiceCollectionMethod(value *billingentity.CollectionMethod) *BillingCustomerOverrideUpdateOne {
	if value == nil {
		return u.ClearInvoiceCollectionMethod()
	}
	return u.SetInvoiceCollectionMethod(*value)
}

func (u *BillingInvoiceUpdate) SetOrClearDeletedAt(value *time.Time) *BillingInvoiceUpdate {
	if value == nil {
		return u.ClearDeletedAt()
	}
	return u.SetDeletedAt(*value)
}

func (u *BillingInvoiceUpdateOne) SetOrClearDeletedAt(value *time.Time) *BillingInvoiceUpdateOne {
	if value == nil {
		return u.ClearDeletedAt()
	}
	return u.SetDeletedAt(*value)
}

func (u *BillingInvoiceUpdate) SetOrClearMetadata(value *map[string]string) *BillingInvoiceUpdate {
	if value == nil {
		return u.ClearMetadata()
	}
	return u.SetMetadata(*value)
}

func (u *BillingInvoiceUpdateOne) SetOrClearMetadata(value *map[string]string) *BillingInvoiceUpdateOne {
	if value == nil {
		return u.ClearMetadata()
	}
	return u.SetMetadata(*value)
}

func (u *BillingInvoiceUpdate) SetOrClearSeries(value *string) *BillingInvoiceUpdate {
	if value == nil {
		return u.ClearSeries()
	}
	return u.SetSeries(*value)
}

func (u *BillingInvoiceUpdateOne) SetOrClearSeries(value *string) *BillingInvoiceUpdateOne {
	if value == nil {
		return u.ClearSeries()
	}
	return u.SetSeries(*value)
}

func (u *BillingInvoiceUpdate) SetOrClearCode(value *string) *BillingInvoiceUpdate {
	if value == nil {
		return u.ClearCode()
	}
	return u.SetCode(*value)
}

func (u *BillingInvoiceUpdateOne) SetOrClearCode(value *string) *BillingInvoiceUpdateOne {
	if value == nil {
		return u.ClearCode()
	}
	return u.SetCode(*value)
}

func (u *BillingInvoiceUpdate) SetOrClearVoidedAt(value *time.Time) *BillingInvoiceUpdate {
	if value == nil {
		return u.ClearVoidedAt()
	}
	return u.SetVoidedAt(*value)
}

func (u *BillingInvoiceUpdateOne) SetOrClearVoidedAt(value *time.Time) *BillingInvoiceUpdateOne {
	if value == nil {
		return u.ClearVoidedAt()
	}
	return u.SetVoidedAt(*value)
}

func (u *BillingInvoiceItemUpdate) SetOrClearDeletedAt(value *time.Time) *BillingInvoiceItemUpdate {
	if value == nil {
		return u.ClearDeletedAt()
	}
	return u.SetDeletedAt(*value)
}

func (u *BillingInvoiceItemUpdateOne) SetOrClearDeletedAt(value *time.Time) *BillingInvoiceItemUpdateOne {
	if value == nil {
		return u.ClearDeletedAt()
	}
	return u.SetDeletedAt(*value)
}

func (u *BillingInvoiceItemUpdate) SetOrClearMetadata(value *map[string]string) *BillingInvoiceItemUpdate {
	if value == nil {
		return u.ClearMetadata()
	}
	return u.SetMetadata(*value)
}

func (u *BillingInvoiceItemUpdateOne) SetOrClearMetadata(value *map[string]string) *BillingInvoiceItemUpdateOne {
	if value == nil {
		return u.ClearMetadata()
	}
	return u.SetMetadata(*value)
}

func (u *BillingInvoiceItemUpdate) SetOrClearInvoiceID(value *string) *BillingInvoiceItemUpdate {
	if value == nil {
		return u.ClearInvoiceID()
	}
	return u.SetInvoiceID(*value)
}

func (u *BillingInvoiceItemUpdateOne) SetOrClearInvoiceID(value *string) *BillingInvoiceItemUpdateOne {
	if value == nil {
		return u.ClearInvoiceID()
	}
	return u.SetInvoiceID(*value)
}

func (u *BillingInvoiceItemUpdate) SetOrClearQuantity(value *alpacadecimal.Decimal) *BillingInvoiceItemUpdate {
	if value == nil {
		return u.ClearQuantity()
	}
	return u.SetQuantity(*value)
}

func (u *BillingInvoiceItemUpdateOne) SetOrClearQuantity(value *alpacadecimal.Decimal) *BillingInvoiceItemUpdateOne {
	if value == nil {
		return u.ClearQuantity()
	}
	return u.SetQuantity(*value)
}

func (u *BillingProfileUpdate) SetOrClearMetadata(value *map[string]string) *BillingProfileUpdate {
	if value == nil {
		return u.ClearMetadata()
	}
	return u.SetMetadata(*value)
}

func (u *BillingProfileUpdateOne) SetOrClearMetadata(value *map[string]string) *BillingProfileUpdateOne {
	if value == nil {
		return u.ClearMetadata()
	}
	return u.SetMetadata(*value)
}

func (u *BillingProfileUpdate) SetOrClearDeletedAt(value *time.Time) *BillingProfileUpdate {
	if value == nil {
		return u.ClearDeletedAt()
	}
	return u.SetDeletedAt(*value)
}

func (u *BillingProfileUpdateOne) SetOrClearDeletedAt(value *time.Time) *BillingProfileUpdateOne {
	if value == nil {
		return u.ClearDeletedAt()
	}
	return u.SetDeletedAt(*value)
}

func (u *BillingProfileUpdate) SetOrClearDescription(value *string) *BillingProfileUpdate {
	if value == nil {
		return u.ClearDescription()
	}
	return u.SetDescription(*value)
}

func (u *BillingProfileUpdateOne) SetOrClearDescription(value *string) *BillingProfileUpdateOne {
	if value == nil {
		return u.ClearDescription()
	}
	return u.SetDescription(*value)
}

func (u *BillingProfileUpdate) SetOrClearSupplierAddressCountry(value *models.CountryCode) *BillingProfileUpdate {
	if value == nil {
		return u.ClearSupplierAddressCountry()
	}
	return u.SetSupplierAddressCountry(*value)
}

func (u *BillingProfileUpdateOne) SetOrClearSupplierAddressCountry(value *models.CountryCode) *BillingProfileUpdateOne {
	if value == nil {
		return u.ClearSupplierAddressCountry()
	}
	return u.SetSupplierAddressCountry(*value)
}

func (u *BillingProfileUpdate) SetOrClearSupplierAddressPostalCode(value *string) *BillingProfileUpdate {
	if value == nil {
		return u.ClearSupplierAddressPostalCode()
	}
	return u.SetSupplierAddressPostalCode(*value)
}

func (u *BillingProfileUpdateOne) SetOrClearSupplierAddressPostalCode(value *string) *BillingProfileUpdateOne {
	if value == nil {
		return u.ClearSupplierAddressPostalCode()
	}
	return u.SetSupplierAddressPostalCode(*value)
}

func (u *BillingProfileUpdate) SetOrClearSupplierAddressState(value *string) *BillingProfileUpdate {
	if value == nil {
		return u.ClearSupplierAddressState()
	}
	return u.SetSupplierAddressState(*value)
}

func (u *BillingProfileUpdateOne) SetOrClearSupplierAddressState(value *string) *BillingProfileUpdateOne {
	if value == nil {
		return u.ClearSupplierAddressState()
	}
	return u.SetSupplierAddressState(*value)
}

func (u *BillingProfileUpdate) SetOrClearSupplierAddressCity(value *string) *BillingProfileUpdate {
	if value == nil {
		return u.ClearSupplierAddressCity()
	}
	return u.SetSupplierAddressCity(*value)
}

func (u *BillingProfileUpdateOne) SetOrClearSupplierAddressCity(value *string) *BillingProfileUpdateOne {
	if value == nil {
		return u.ClearSupplierAddressCity()
	}
	return u.SetSupplierAddressCity(*value)
}

func (u *BillingProfileUpdate) SetOrClearSupplierAddressLine1(value *string) *BillingProfileUpdate {
	if value == nil {
		return u.ClearSupplierAddressLine1()
	}
	return u.SetSupplierAddressLine1(*value)
}

func (u *BillingProfileUpdateOne) SetOrClearSupplierAddressLine1(value *string) *BillingProfileUpdateOne {
	if value == nil {
		return u.ClearSupplierAddressLine1()
	}
	return u.SetSupplierAddressLine1(*value)
}

func (u *BillingProfileUpdate) SetOrClearSupplierAddressLine2(value *string) *BillingProfileUpdate {
	if value == nil {
		return u.ClearSupplierAddressLine2()
	}
	return u.SetSupplierAddressLine2(*value)
}

func (u *BillingProfileUpdateOne) SetOrClearSupplierAddressLine2(value *string) *BillingProfileUpdateOne {
	if value == nil {
		return u.ClearSupplierAddressLine2()
	}
	return u.SetSupplierAddressLine2(*value)
}

func (u *BillingProfileUpdate) SetOrClearSupplierAddressPhoneNumber(value *string) *BillingProfileUpdate {
	if value == nil {
		return u.ClearSupplierAddressPhoneNumber()
	}
	return u.SetSupplierAddressPhoneNumber(*value)
}

func (u *BillingProfileUpdateOne) SetOrClearSupplierAddressPhoneNumber(value *string) *BillingProfileUpdateOne {
	if value == nil {
		return u.ClearSupplierAddressPhoneNumber()
	}
	return u.SetSupplierAddressPhoneNumber(*value)
}

func (u *BillingProfileUpdate) SetOrClearSupplierTaxCode(value *string) *BillingProfileUpdate {
	if value == nil {
		return u.ClearSupplierTaxCode()
	}
	return u.SetSupplierTaxCode(*value)
}

func (u *BillingProfileUpdateOne) SetOrClearSupplierTaxCode(value *string) *BillingProfileUpdateOne {
	if value == nil {
		return u.ClearSupplierTaxCode()
	}
	return u.SetSupplierTaxCode(*value)
}

func (u *BillingWorkflowConfigUpdate) SetOrClearDeletedAt(value *time.Time) *BillingWorkflowConfigUpdate {
	if value == nil {
		return u.ClearDeletedAt()
	}
	return u.SetDeletedAt(*value)
}

func (u *BillingWorkflowConfigUpdateOne) SetOrClearDeletedAt(value *time.Time) *BillingWorkflowConfigUpdateOne {
	if value == nil {
		return u.ClearDeletedAt()
	}
	return u.SetDeletedAt(*value)
}

func (u *CustomerUpdate) SetOrClearMetadata(value *map[string]string) *CustomerUpdate {
	if value == nil {
		return u.ClearMetadata()
	}
	return u.SetMetadata(*value)
}

func (u *CustomerUpdateOne) SetOrClearMetadata(value *map[string]string) *CustomerUpdateOne {
	if value == nil {
		return u.ClearMetadata()
	}
	return u.SetMetadata(*value)
}

func (u *CustomerUpdate) SetOrClearDeletedAt(value *time.Time) *CustomerUpdate {
	if value == nil {
		return u.ClearDeletedAt()
	}
	return u.SetDeletedAt(*value)
}

func (u *CustomerUpdateOne) SetOrClearDeletedAt(value *time.Time) *CustomerUpdateOne {
	if value == nil {
		return u.ClearDeletedAt()
	}
	return u.SetDeletedAt(*value)
}

func (u *CustomerUpdate) SetOrClearDescription(value *string) *CustomerUpdate {
	if value == nil {
		return u.ClearDescription()
	}
	return u.SetDescription(*value)
}

func (u *CustomerUpdateOne) SetOrClearDescription(value *string) *CustomerUpdateOne {
	if value == nil {
		return u.ClearDescription()
	}
	return u.SetDescription(*value)
}

func (u *CustomerUpdate) SetOrClearBillingAddressCountry(value *models.CountryCode) *CustomerUpdate {
	if value == nil {
		return u.ClearBillingAddressCountry()
	}
	return u.SetBillingAddressCountry(*value)
}

func (u *CustomerUpdateOne) SetOrClearBillingAddressCountry(value *models.CountryCode) *CustomerUpdateOne {
	if value == nil {
		return u.ClearBillingAddressCountry()
	}
	return u.SetBillingAddressCountry(*value)
}

func (u *CustomerUpdate) SetOrClearBillingAddressPostalCode(value *string) *CustomerUpdate {
	if value == nil {
		return u.ClearBillingAddressPostalCode()
	}
	return u.SetBillingAddressPostalCode(*value)
}

func (u *CustomerUpdateOne) SetOrClearBillingAddressPostalCode(value *string) *CustomerUpdateOne {
	if value == nil {
		return u.ClearBillingAddressPostalCode()
	}
	return u.SetBillingAddressPostalCode(*value)
}

func (u *CustomerUpdate) SetOrClearBillingAddressState(value *string) *CustomerUpdate {
	if value == nil {
		return u.ClearBillingAddressState()
	}
	return u.SetBillingAddressState(*value)
}

func (u *CustomerUpdateOne) SetOrClearBillingAddressState(value *string) *CustomerUpdateOne {
	if value == nil {
		return u.ClearBillingAddressState()
	}
	return u.SetBillingAddressState(*value)
}

func (u *CustomerUpdate) SetOrClearBillingAddressCity(value *string) *CustomerUpdate {
	if value == nil {
		return u.ClearBillingAddressCity()
	}
	return u.SetBillingAddressCity(*value)
}

func (u *CustomerUpdateOne) SetOrClearBillingAddressCity(value *string) *CustomerUpdateOne {
	if value == nil {
		return u.ClearBillingAddressCity()
	}
	return u.SetBillingAddressCity(*value)
}

func (u *CustomerUpdate) SetOrClearBillingAddressLine1(value *string) *CustomerUpdate {
	if value == nil {
		return u.ClearBillingAddressLine1()
	}
	return u.SetBillingAddressLine1(*value)
}

func (u *CustomerUpdateOne) SetOrClearBillingAddressLine1(value *string) *CustomerUpdateOne {
	if value == nil {
		return u.ClearBillingAddressLine1()
	}
	return u.SetBillingAddressLine1(*value)
}

func (u *CustomerUpdate) SetOrClearBillingAddressLine2(value *string) *CustomerUpdate {
	if value == nil {
		return u.ClearBillingAddressLine2()
	}
	return u.SetBillingAddressLine2(*value)
}

func (u *CustomerUpdateOne) SetOrClearBillingAddressLine2(value *string) *CustomerUpdateOne {
	if value == nil {
		return u.ClearBillingAddressLine2()
	}
	return u.SetBillingAddressLine2(*value)
}

func (u *CustomerUpdate) SetOrClearBillingAddressPhoneNumber(value *string) *CustomerUpdate {
	if value == nil {
		return u.ClearBillingAddressPhoneNumber()
	}
	return u.SetBillingAddressPhoneNumber(*value)
}

func (u *CustomerUpdateOne) SetOrClearBillingAddressPhoneNumber(value *string) *CustomerUpdateOne {
	if value == nil {
		return u.ClearBillingAddressPhoneNumber()
	}
	return u.SetBillingAddressPhoneNumber(*value)
}

func (u *CustomerUpdate) SetOrClearPrimaryEmail(value *string) *CustomerUpdate {
	if value == nil {
		return u.ClearPrimaryEmail()
	}
	return u.SetPrimaryEmail(*value)
}

func (u *CustomerUpdateOne) SetOrClearPrimaryEmail(value *string) *CustomerUpdateOne {
	if value == nil {
		return u.ClearPrimaryEmail()
	}
	return u.SetPrimaryEmail(*value)
}

func (u *CustomerUpdate) SetOrClearTimezone(value *timezone.Timezone) *CustomerUpdate {
	if value == nil {
		return u.ClearTimezone()
	}
	return u.SetTimezone(*value)
}

func (u *CustomerUpdateOne) SetOrClearTimezone(value *timezone.Timezone) *CustomerUpdateOne {
	if value == nil {
		return u.ClearTimezone()
	}
	return u.SetTimezone(*value)
}

func (u *CustomerUpdate) SetOrClearCurrency(value *currencyx.Code) *CustomerUpdate {
	if value == nil {
		return u.ClearCurrency()
	}
	return u.SetCurrency(*value)
}

func (u *CustomerUpdateOne) SetOrClearCurrency(value *currencyx.Code) *CustomerUpdateOne {
	if value == nil {
		return u.ClearCurrency()
	}
	return u.SetCurrency(*value)
}

func (u *EntitlementUpdate) SetOrClearMetadata(value *map[string]string) *EntitlementUpdate {
	if value == nil {
		return u.ClearMetadata()
	}
	return u.SetMetadata(*value)
}

func (u *EntitlementUpdateOne) SetOrClearMetadata(value *map[string]string) *EntitlementUpdateOne {
	if value == nil {
		return u.ClearMetadata()
	}
	return u.SetMetadata(*value)
}

func (u *EntitlementUpdate) SetOrClearDeletedAt(value *time.Time) *EntitlementUpdate {
	if value == nil {
		return u.ClearDeletedAt()
	}
	return u.SetDeletedAt(*value)
}

func (u *EntitlementUpdateOne) SetOrClearDeletedAt(value *time.Time) *EntitlementUpdateOne {
	if value == nil {
		return u.ClearDeletedAt()
	}
	return u.SetDeletedAt(*value)
}

func (u *EntitlementUpdate) SetOrClearActiveTo(value *time.Time) *EntitlementUpdate {
	if value == nil {
		return u.ClearActiveTo()
	}
	return u.SetActiveTo(*value)
}

func (u *EntitlementUpdateOne) SetOrClearActiveTo(value *time.Time) *EntitlementUpdateOne {
	if value == nil {
		return u.ClearActiveTo()
	}
	return u.SetActiveTo(*value)
}

func (u *EntitlementUpdate) SetOrClearConfig(value *[]uint8) *EntitlementUpdate {
	if value == nil {
		return u.ClearConfig()
	}
	return u.SetConfig(*value)
}

func (u *EntitlementUpdateOne) SetOrClearConfig(value *[]uint8) *EntitlementUpdateOne {
	if value == nil {
		return u.ClearConfig()
	}
	return u.SetConfig(*value)
}

func (u *EntitlementUpdate) SetOrClearUsagePeriodAnchor(value *time.Time) *EntitlementUpdate {
	if value == nil {
		return u.ClearUsagePeriodAnchor()
	}
	return u.SetUsagePeriodAnchor(*value)
}

func (u *EntitlementUpdateOne) SetOrClearUsagePeriodAnchor(value *time.Time) *EntitlementUpdateOne {
	if value == nil {
		return u.ClearUsagePeriodAnchor()
	}
	return u.SetUsagePeriodAnchor(*value)
}

func (u *EntitlementUpdate) SetOrClearCurrentUsagePeriodStart(value *time.Time) *EntitlementUpdate {
	if value == nil {
		return u.ClearCurrentUsagePeriodStart()
	}
	return u.SetCurrentUsagePeriodStart(*value)
}

func (u *EntitlementUpdateOne) SetOrClearCurrentUsagePeriodStart(value *time.Time) *EntitlementUpdateOne {
	if value == nil {
		return u.ClearCurrentUsagePeriodStart()
	}
	return u.SetCurrentUsagePeriodStart(*value)
}

func (u *EntitlementUpdate) SetOrClearCurrentUsagePeriodEnd(value *time.Time) *EntitlementUpdate {
	if value == nil {
		return u.ClearCurrentUsagePeriodEnd()
	}
	return u.SetCurrentUsagePeriodEnd(*value)
}

func (u *EntitlementUpdateOne) SetOrClearCurrentUsagePeriodEnd(value *time.Time) *EntitlementUpdateOne {
	if value == nil {
		return u.ClearCurrentUsagePeriodEnd()
	}
	return u.SetCurrentUsagePeriodEnd(*value)
}

func (u *FeatureUpdate) SetOrClearDeletedAt(value *time.Time) *FeatureUpdate {
	if value == nil {
		return u.ClearDeletedAt()
	}
	return u.SetDeletedAt(*value)
}

func (u *FeatureUpdateOne) SetOrClearDeletedAt(value *time.Time) *FeatureUpdateOne {
	if value == nil {
		return u.ClearDeletedAt()
	}
	return u.SetDeletedAt(*value)
}

func (u *FeatureUpdate) SetOrClearMetadata(value *map[string]string) *FeatureUpdate {
	if value == nil {
		return u.ClearMetadata()
	}
	return u.SetMetadata(*value)
}

func (u *FeatureUpdateOne) SetOrClearMetadata(value *map[string]string) *FeatureUpdateOne {
	if value == nil {
		return u.ClearMetadata()
	}
	return u.SetMetadata(*value)
}

func (u *FeatureUpdate) SetOrClearMeterGroupByFilters(value *map[string]string) *FeatureUpdate {
	if value == nil {
		return u.ClearMeterGroupByFilters()
	}
	return u.SetMeterGroupByFilters(*value)
}

func (u *FeatureUpdateOne) SetOrClearMeterGroupByFilters(value *map[string]string) *FeatureUpdateOne {
	if value == nil {
		return u.ClearMeterGroupByFilters()
	}
	return u.SetMeterGroupByFilters(*value)
}

func (u *FeatureUpdate) SetOrClearArchivedAt(value *time.Time) *FeatureUpdate {
	if value == nil {
		return u.ClearArchivedAt()
	}
	return u.SetArchivedAt(*value)
}

func (u *FeatureUpdateOne) SetOrClearArchivedAt(value *time.Time) *FeatureUpdateOne {
	if value == nil {
		return u.ClearArchivedAt()
	}
	return u.SetArchivedAt(*value)
}

func (u *GrantUpdate) SetOrClearMetadata(value *map[string]string) *GrantUpdate {
	if value == nil {
		return u.ClearMetadata()
	}
	return u.SetMetadata(*value)
}

func (u *GrantUpdateOne) SetOrClearMetadata(value *map[string]string) *GrantUpdateOne {
	if value == nil {
		return u.ClearMetadata()
	}
	return u.SetMetadata(*value)
}

func (u *GrantUpdate) SetOrClearDeletedAt(value *time.Time) *GrantUpdate {
	if value == nil {
		return u.ClearDeletedAt()
	}
	return u.SetDeletedAt(*value)
}

func (u *GrantUpdateOne) SetOrClearDeletedAt(value *time.Time) *GrantUpdateOne {
	if value == nil {
		return u.ClearDeletedAt()
	}
	return u.SetDeletedAt(*value)
}

func (u *GrantUpdate) SetOrClearVoidedAt(value *time.Time) *GrantUpdate {
	if value == nil {
		return u.ClearVoidedAt()
	}
	return u.SetVoidedAt(*value)
}

func (u *GrantUpdateOne) SetOrClearVoidedAt(value *time.Time) *GrantUpdateOne {
	if value == nil {
		return u.ClearVoidedAt()
	}
	return u.SetVoidedAt(*value)
}

func (u *NotificationChannelUpdate) SetOrClearDeletedAt(value *time.Time) *NotificationChannelUpdate {
	if value == nil {
		return u.ClearDeletedAt()
	}
	return u.SetDeletedAt(*value)
}

func (u *NotificationChannelUpdateOne) SetOrClearDeletedAt(value *time.Time) *NotificationChannelUpdateOne {
	if value == nil {
		return u.ClearDeletedAt()
	}
	return u.SetDeletedAt(*value)
}

func (u *NotificationChannelUpdate) SetOrClearDisabled(value *bool) *NotificationChannelUpdate {
	if value == nil {
		return u.ClearDisabled()
	}
	return u.SetDisabled(*value)
}

func (u *NotificationChannelUpdateOne) SetOrClearDisabled(value *bool) *NotificationChannelUpdateOne {
	if value == nil {
		return u.ClearDisabled()
	}
	return u.SetDisabled(*value)
}

func (u *NotificationEventUpdate) SetOrClearAnnotations(value *map[string]interface{}) *NotificationEventUpdate {
	if value == nil {
		return u.ClearAnnotations()
	}
	return u.SetAnnotations(*value)
}

func (u *NotificationEventUpdateOne) SetOrClearAnnotations(value *map[string]interface{}) *NotificationEventUpdateOne {
	if value == nil {
		return u.ClearAnnotations()
	}
	return u.SetAnnotations(*value)
}

func (u *NotificationEventDeliveryStatusUpdate) SetOrClearReason(value *string) *NotificationEventDeliveryStatusUpdate {
	if value == nil {
		return u.ClearReason()
	}
	return u.SetReason(*value)
}

func (u *NotificationEventDeliveryStatusUpdateOne) SetOrClearReason(value *string) *NotificationEventDeliveryStatusUpdateOne {
	if value == nil {
		return u.ClearReason()
	}
	return u.SetReason(*value)
}

func (u *NotificationRuleUpdate) SetOrClearDeletedAt(value *time.Time) *NotificationRuleUpdate {
	if value == nil {
		return u.ClearDeletedAt()
	}
	return u.SetDeletedAt(*value)
}

func (u *NotificationRuleUpdateOne) SetOrClearDeletedAt(value *time.Time) *NotificationRuleUpdateOne {
	if value == nil {
		return u.ClearDeletedAt()
	}
	return u.SetDeletedAt(*value)
}

func (u *NotificationRuleUpdate) SetOrClearDisabled(value *bool) *NotificationRuleUpdate {
	if value == nil {
		return u.ClearDisabled()
	}
	return u.SetDisabled(*value)
}

func (u *NotificationRuleUpdateOne) SetOrClearDisabled(value *bool) *NotificationRuleUpdateOne {
	if value == nil {
		return u.ClearDisabled()
	}
	return u.SetDisabled(*value)
}

func (u *UsageResetUpdate) SetOrClearDeletedAt(value *time.Time) *UsageResetUpdate {
	if value == nil {
		return u.ClearDeletedAt()
	}
	return u.SetDeletedAt(*value)
}

func (u *UsageResetUpdateOne) SetOrClearDeletedAt(value *time.Time) *UsageResetUpdateOne {
	if value == nil {
		return u.ClearDeletedAt()
	}
	return u.SetDeletedAt(*value)
}
