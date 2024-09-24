package schema

// Table of ID prefixes for each entity type.
// Please make sure you are picking a unique prefix for each entity type.
// Recommendation:
// - at least 3 characters of the entity type name.
// - max length is 8 characters (field size limit)

const (
	// Billing
	IDPrefixBillingProfile     = "bi_p_"
	IDPrefixBillingInvoice     = "bi_i_"
	IDPrefixBillingInvoiceItem = "bi_ii_"
	IDPrefixWorkflowConfig     = "bi_wc_"

	// Customer
	IDPrefixCustomer = "cus_"

	// Entitlements
	IDPrefixEntitlement           = "en_"
	IDPrefixEntitlementUsageReset = "en_ur_"

	// Credits
	IDPrefixGrant = "cr_g_"

	// Notifications
	IDPrefixNotificationChannel             = "no_c_"
	IDPrefixNotificationRule                = "no_r_"
	IDPrefixNotificationEvent               = "no_e_"
	IDPrefixNotificationEventDeliveryStatus = "no_ed_"

	// Product catalog
	IDPrefixFeature = "pr_f_"
)
