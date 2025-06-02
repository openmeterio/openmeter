export interface paths {
  '/api/v1/addons': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * List add-ons
     * @description List all add-ons.
     */
    get: operations['listAddons']
    put?: never
    /**
     * Create an add-on
     * @description Create a new add-on.
     */
    post: operations['createAddon']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/addons/{addonId}': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * Get add-on
     * @description Get add-on by id or key. The latest published version is returned if latter is used.
     */
    get: operations['getAddon']
    /**
     * Update add-on
     * @description Update add-on by id.
     */
    put: operations['updateAddon']
    post?: never
    /**
     * Delete add-on
     * @description Soft delete add-on by id.
     *
     *     Once a add-on is deleted it cannot be undeleted.
     */
    delete: operations['deleteAddon']
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/addons/{addonId}/archive': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /**
     * Archive add-on version
     * @description Archive a add-on version.
     */
    post: operations['archiveAddon']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/addons/{addonId}/publish': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /**
     * Publish add-on
     * @description Publish a add-on version.
     */
    post: operations['publishAddon']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/apps': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * List apps
     * @description List apps.
     */
    get: operations['listApps']
    put?: never
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/apps/custom-invoicing/{invoiceId}/draft/synchronized': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /** Submit draft synchronization results */
    post: operations['appCustomInvoicingDraftSynchronized']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/apps/custom-invoicing/{invoiceId}/issuing/synchronized': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /** Submit issuing synchronization results */
    post: operations['appCustomInvoicingIssuingSynchronized']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/apps/custom-invoicing/{invoiceId}/payment/status': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /** Update payment status */
    post: operations['appCustomInvoicingUpdatePaymentStatus']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/apps/{id}': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * Get app
     * @description Get the app.
     */
    get: operations['getApp']
    /**
     * Update app
     * @description Update an app.
     */
    put: operations['updateApp']
    post?: never
    /**
     * Uninstall app
     * @description Uninstall an app.
     */
    delete: operations['uninstallApp']
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/apps/{id}/stripe/api-key': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    /**
     * Update Stripe API key
     * @deprecated
     * @description Update the Stripe API key.
     */
    put: operations['updateStripeAPIKey']
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/apps/{id}/stripe/webhook': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /**
     * Stripe webhook
     * @description Handle stripe webhooks for apps.
     */
    post: operations['appStripeWebhook']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/billing/customers': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * List customer overrides
     * @description List customer overrides using the specified filters.
     *
     *     The response will include the customer override values and the merged billing profile values.
     *
     *     If the includeAllCustomers is set to true, the list contains all customers. This mode is
     *     useful for getting the current effective billing workflow settings for all users regardless
     *     if they have customer orverrides or not.
     */
    get: operations['listBillingProfileCustomerOverrides']
    put?: never
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/billing/customers/{customerId}': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * Get a customer override
     * @description Get a customer override by customer id.
     *
     *     The response will include the customer override values and the merged billing profile values.
     *
     *     If the customer override is not found, the default billing profile's values are returned. This behavior
     *     allows for getting a merged profile regardless of the customer override existence.
     */
    get: operations['getBillingProfileCustomerOverride']
    /**
     * Create a new or update a customer override
     * @description The customer override can be used to pin a given customer to a billing profile
     *     different from the default one.
     *
     *     This can be used to test the effect of different billing profiles before making them
     *     the default ones or have different workflow settings for example for enterprise customers.
     */
    put: operations['upsertBillingProfileCustomerOverride']
    post?: never
    /**
     * Delete a customer override
     * @description Delete a customer override by customer id.
     *
     *     This will remove the customer override and the customer will be subject to the default
     *     billing profile's settings again.
     */
    delete: operations['deleteBillingProfileCustomerOverride']
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/billing/customers/{customerId}/invoices/pending-lines': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /**
     * Create pending line items
     * @description Create a new pending line item (charge).
     *
     *     This call is used to create a new pending line item for the customer if required a new
     *     gathering invoice will be created.
     *
     *     A new invoice will be created if:
     *     - there is no invoice in gathering state
     *     - the currency of the line item doesn't match the currency of any invoices in gathering state
     */
    post: operations['createPendingInvoiceLine']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/billing/customers/{customerId}/invoices/simulate': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /**
     * Simulate an invoice for a customer
     * @description Simulate an invoice for a customer.
     *
     *     This call will simulate an invoice for a customer based on the pending line items.
     *
     *     The call will return the total amount of the invoice and the line items that will be included in the invoice.
     */
    post: operations['simulateInvoice']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/billing/invoices': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * List invoices
     * @description List invoices based on the specified filters.
     *
     *     The expand option can be used to include additional information (besides the invoice header and totals)
     *     in the response. For example by adding the expand=lines option the invoice lines will be included in the response.
     *
     *     Gathering invoices will always show the current usage calculated on the fly.
     */
    get: operations['listInvoices']
    put?: never
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/billing/invoices/invoice': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /**
     * Invoice a customer based on the pending line items
     * @description Create a new invoice from the pending line items.
     *
     *     This should be only called if for some reason we need to invoice a customer outside of the normal billing cycle.
     *
     *     When creating an invoice, the pending line items will be marked as invoiced and the invoice will be created with the total amount of the pending items.
     *
     *     New pending line items will be created for the period between now() and the next billing cycle's begining date for any metered item.
     *
     *     The call can return multiple invoices if the pending line items are in different currencies.
     */
    post: operations['invoicePendingLinesAction']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/billing/invoices/{invoiceId}': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * Get an invoice
     * @description Get an invoice by ID.
     *
     *     Gathering invoices will always show the current usage calculated on the fly.
     */
    get: operations['getInvoice']
    /**
     * Update an invoice
     * @description Update an invoice
     *
     *     Only invoices in draft or earlier status can be updated.
     */
    put: operations['updateInvoice']
    post?: never
    /**
     * Delete an invoice
     * @description Delete an invoice
     *
     *     Only invoices that are in the draft (or earlier) status can be deleted.
     *
     *     Invoices that are post finalization can only be voided.
     */
    delete: operations['deleteInvoice']
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/billing/invoices/{invoiceId}/advance': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /**
     * Advance the invoice's state to the next status
     * @description Advance the invoice's state to the next status.
     *
     *     The call doesn't "approve the invoice", it only advances the invoice to the next status if the transition would be automatic.
     *
     *     The action can be called when the invoice's statusDetails' actions field contain the "advance" action.
     */
    post: operations['advanceInvoiceAction']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/billing/invoices/{invoiceId}/approve': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /**
     * Send the invoice to the customer
     * @description Approve an invoice and start executing the payment workflow.
     *
     *     This call instantly sends the invoice to the customer using the configured billing profile app.
     *
     *     This call is valid in two invoice statuses:
     *     - `draft`: the invoice will be sent to the customer, the invluce state becomes issued
     *     - `manual_approval_needed`: the invoice will be sent to the customer, the invoice state becomes issued
     */
    post: operations['approveInvoiceAction']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/billing/invoices/{invoiceId}/retry': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /**
     * Retry advancing the invoice after a failed attempt.
     * @description Retry advancing the invoice after a failed attempt.
     *
     *     The action can be called when the invoice's statusDetails' actions field contain the "retry" action.
     */
    post: operations['retryInvoiceAction']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/billing/invoices/{invoiceId}/snapshot-quantities': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /**
     * Snapshot quantities for usage based line items
     * @description Snapshot quantities for usage based line items.
     *
     *     This call will snapshot the quantities for all usage based line items in the invoice.
     *
     *     This call is only valid in `draft.waiting_for_collection` status, where the collection period
     *     can be skipped using this action.
     */
    post: operations['snapshotQuantitiesInvoiceAction']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/billing/invoices/{invoiceId}/taxes/recalculate': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /**
     * Recalculate an invoice's tax amounts
     * @description Recalculate an invoice's tax amounts (using the app set in the customer's billing profile)
     *
     *     Note: charges might apply, depending on the tax provider.
     */
    post: operations['recalculateInvoiceTaxAction']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/billing/invoices/{invoiceId}/void': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /**
     * Void an invoice
     * @description Void an invoice
     *
     *     Only invoices that have been alread issued can be voided.
     *
     *     Voiding an invoice will mark it as voided, the user can specify how to handle the voided line items.
     */
    post: operations['voidInvoiceAction']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/billing/profiles': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * List billing profiles
     * @description List all billing profiles matching the specified filters.
     *
     *     The expand option can be used to include additional information (besides the billing profile)
     *     in the response. For example by adding the expand=apps option the apps used by the billing profile
     *     will be included in the response.
     */
    get: operations['listBillingProfiles']
    put?: never
    /**
     * Create a new billing profile
     * @description Create a new billing profile
     *
     *     Billing profiles are representations of a customer's billing information. Customer overrides
     *     can be applied to a billing profile to customize the billing behavior for a specific customer.
     */
    post: operations['createBillingProfile']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/billing/profiles/{id}': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * Get a billing profile
     * @description Get a billing profile by id.
     *
     *     The expand option can be used to include additional information (besides the billing profile)
     *     in the response. For example by adding the expand=apps option the apps used by the billing profile
     *     will be included in the response.
     */
    get: operations['getBillingProfile']
    /**
     * Update a billing profile
     * @description Update a billing profile by id.
     *
     *     The apps field cannot be updated directly, if an app change is desired a new
     *     profile should be created.
     */
    put: operations['updateBillingProfile']
    post?: never
    /**
     * Delete a billing profile
     * @description Delete a billing profile by id.
     *
     *     Only such billing profiles can be deleted that are:
     *     - not the default one
     *     - not pinned to any customer using customer overrides
     *     - only have finalized invoices
     */
    delete: operations['deleteBillingProfile']
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/customers': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * List customers
     * @description List customers.
     */
    get: operations['listCustomers']
    put?: never
    /**
     * Create customer
     * @description Create a new customer.
     */
    post: operations['createCustomer']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/customers/{customerIdOrKey}': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * Get customer
     * @description Get a customer by ID or key.
     */
    get: operations['getCustomer']
    /**
     * Update customer
     * @description Update a customer by ID.
     */
    put: operations['updateCustomer']
    post?: never
    /**
     * Delete customer
     * @description Delete a customer by ID.
     */
    delete: operations['deleteCustomer']
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/customers/{customerIdOrKey}/access': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * Get customer access
     * @description Get the overall access of a customer.
     */
    get: operations['getCustomerAccess']
    put?: never
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/customers/{customerIdOrKey}/apps': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * List customer app data
     * @description List customers app data.
     */
    get: operations['listCustomerAppData']
    /**
     * Upsert customer app data
     * @description Upsert customer app data.
     */
    put: operations['upsertCustomerAppData']
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/customers/{customerIdOrKey}/apps/{appId}': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    post?: never
    /**
     * Delete customer app data
     * @description Delete customer app data.
     */
    delete: operations['deleteCustomerAppData']
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/customers/{customerIdOrKey}/entitlements/{featureKey}/value': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * Get entitlement value
     * @description Checks customer access to a given feature (by key). All entitlement types share the hasAccess property in their value response, but multiple other properties are returned based on the entitlement type.
     */
    get: operations['getCustomerEntitlementValue']
    put?: never
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/customers/{customerIdOrKey}/subscriptions': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * List customer subscriptions
     * @description Lists all subscriptions for a customer.
     */
    get: operations['listCustomerSubscriptions']
    put?: never
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/debug/metrics': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * Get event metrics
     * @description Returns debug metrics (in OpenMetrics format) like the number of ingested events since mindnight UTC.
     *
     *     The OpenMetrics Counter(s) reset every day at midnight UTC.
     */
    get: operations['getDebugMetrics']
    put?: never
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/entitlements': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * List all entitlements
     * @description List all entitlements for all the subjects and features. This endpoint is intended for administrative purposes only.
     *     To fetch the entitlements of a specific subject please use the /api/v1/subjects/{subjectKeyOrID}/entitlements endpoint.
     *     If page is provided that takes precedence and the paginated response is returned.
     */
    get: operations['listEntitlements']
    put?: never
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/entitlements/{entitlementId}': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * Get entitlement by id
     * @description Get entitlement by id.
     */
    get: operations['getEntitlementById']
    put?: never
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/events': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * List ingested events
     * @description List ingested events within a time range.
     *
     *     If the from query param is not provided it defaults to last 72 hours.
     */
    get: operations['listEvents']
    put?: never
    /**
     * Ingest events
     * @description Ingests an event or batch of events following the CloudEvents specification.
     */
    post: operations['ingestEvents']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/features': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * List features
     * @description List features.
     */
    get: operations['listFeatures']
    put?: never
    /**
     * Create feature
     * @description Features are either metered or static. A feature is metered if meterSlug is provided at creation.
     *     For metered features you can pass additional filters that will be applied when calculating feature usage, based on the meter's groupBy fields.
     *     Only meters with SUM and COUNT aggregation are supported for features.
     *     Features cannot be updated later, only archived.
     */
    post: operations['createFeature']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/features/{featureId}': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * Get feature
     * @description Get a feature by ID.
     */
    get: operations['getFeature']
    put?: never
    post?: never
    /**
     * Delete feature
     * @description Archive a feature by ID.
     *
     *     Once a feature is archived it cannot be unarchived. If a feature is archived, new entitlements cannot be created for it, but archiving the feature does not affect existing entitlements.
     *     This means, if you want to create a new feature with the same key, and then create entitlements for it, the previous entitlements have to be deleted first on a per subject basis.
     */
    delete: operations['deleteFeature']
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/grants': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * List grants
     * @description List all grants for all the subjects and entitlements. This endpoint is intended for administrative purposes only.
     *     To fetch the grants of a specific entitlement please use the /api/v1/subjects/{subjectKeyOrID}/entitlements/{entitlementOrFeatureID}/grants endpoint.
     *     If page is provided that takes precedence and the paginated response is returned.
     */
    get: operations['listGrants']
    put?: never
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/grants/{grantId}': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    post?: never
    /**
     * Void grant
     * @description Voiding a grant means it is no longer valid, it doesn't take part in further balance calculations. Voiding a grant does not retroactively take effect, meaning any usage that has already been attributed to the grant will remain, but future usage cannot be burnt down from the grant.
     *     For example, if you have a single grant for your metered entitlement with an initial amount of 100, and so far 60 usage has been metered, the grant (and the entitlement itself) would have a balance of 40. If you then void that grant, balance becomes 0, but the 60 previous usage will not be affected.
     */
    delete: operations['voidGrant']
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/info/currencies': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * List supported currencies
     * @description List all supported currencies.
     */
    get: operations['listCurrencies']
    put?: never
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/info/progress/{id}': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * Get progress
     * @description Get progress
     */
    get: operations['getProgress']
    put?: never
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/marketplace/listings': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * List available apps
     * @description List available apps of the app marketplace.
     */
    get: operations['listMarketplaceListings']
    put?: never
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/marketplace/listings/{type}': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * Get app details by type
     * @description Get a marketplace listing by type.
     */
    get: operations['getMarketplaceListing']
    put?: never
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/marketplace/listings/{type}/install': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /**
     * Install app
     * @description Install an app from the marketplace.
     */
    post: operations['marketplaceAppInstall']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/marketplace/listings/{type}/install/apikey': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /**
     * Install app via API key
     * @description Install an marketplace app via API Key.
     */
    post: operations['marketplaceAppAPIKeyInstall']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/marketplace/listings/{type}/install/oauth2': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * Get OAuth2 install URL
     * @description Install an app via OAuth.
     *     Returns a URL to start the OAuth 2.0 flow.
     */
    get: operations['marketplaceOAuth2InstallGetURL']
    put?: never
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/marketplace/listings/{type}/install/oauth2/authorize': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * Install app via OAuth2
     * @description Authorize OAuth2 code.
     *     Verifies the OAuth code and exchanges it for a token and refresh token
     */
    get: operations['marketplaceOAuth2InstallAuthorize']
    put?: never
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/meters': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * List meters
     * @description List meters.
     */
    get: operations['listMeters']
    put?: never
    /**
     * Create meter
     * @description Create a meter.
     */
    post: operations['createMeter']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/meters/{meterIdOrSlug}': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * Get meter
     * @description Get a meter by ID or slug.
     */
    get: operations['getMeter']
    /**
     * Update meter
     * @description Update a meter.
     */
    put: operations['updateMeter']
    post?: never
    /**
     * Delete meter
     * @description Delete a meter.
     */
    delete: operations['deleteMeter']
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/meters/{meterIdOrSlug}/query': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * Query meter
     * @description Query meter for usage.
     */
    get: operations['queryMeter']
    put?: never
    /** Query meter */
    post: operations['queryMeterPost']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/meters/{meterIdOrSlug}/subjects': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * List meter subjects
     * @description List subjects for a meter.
     */
    get: operations['listMeterSubjects']
    put?: never
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/notification/channels': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * List notification channels
     * @description List all notification channels.
     */
    get: operations['listNotificationChannels']
    put?: never
    /**
     * Create a notification channel
     * @description Create a new notification channel.
     */
    post: operations['createNotificationChannel']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/notification/channels/{channelId}': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * Get notification channel
     * @description Get a notification channel by id.
     */
    get: operations['getNotificationChannel']
    /**
     * Update a notification channel
     * @description Update notification channel.
     */
    put: operations['updateNotificationChannel']
    post?: never
    /**
     * Delete a notification channel
     * @description Soft delete notification channel by id.
     *
     *     Once a notification channel is deleted it cannot be undeleted.
     */
    delete: operations['deleteNotificationChannel']
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/notification/events': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * List notification events
     * @description List all notification events.
     */
    get: operations['listNotificationEvents']
    put?: never
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/notification/events/{eventId}': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * Get notification event
     * @description Get a notification event by id.
     */
    get: operations['getNotificationEvent']
    put?: never
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/notification/rules': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * List notification rules
     * @description List all notification rules.
     */
    get: operations['listNotificationRules']
    put?: never
    /**
     * Create a notification rule
     * @description Create a new notification rule.
     */
    post: operations['createNotificationRule']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/notification/rules/{ruleId}': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * Get notification rule
     * @description Get a notification rule by id.
     */
    get: operations['getNotificationRule']
    /**
     * Update a notification rule
     * @description Update notification rule.
     */
    put: operations['updateNotificationRule']
    post?: never
    /**
     * Delete a notification rule
     * @description Soft delete notification rule by id.
     *
     *     Once a notification rule is deleted it cannot be undeleted.
     */
    delete: operations['deleteNotificationRule']
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/notification/rules/{ruleId}/test': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /**
     * Test notification rule
     * @description Test a notification rule by sending a test event with random data.
     */
    post: operations['testNotificationRule']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/plans': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * List plans
     * @description List all plans.
     */
    get: operations['listPlans']
    put?: never
    /**
     * Create a plan
     * @description Create a new plan.
     */
    post: operations['createPlan']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/plans/{planIdOrKey}/next': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /**
     * New draft plan
     * @deprecated
     * @description Create a new draft version from plan.
     *     It returns error if there is already a plan in draft or planId does not reference the latest published version.
     */
    post: operations['nextPlan']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/plans/{planId}': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * Get plan
     * @description Get a plan by id or key. The latest published version is returned if latter is used.
     */
    get: operations['getPlan']
    /**
     * Update a plan
     * @description Update plan by id.
     */
    put: operations['updatePlan']
    post?: never
    /**
     * Delete plan
     * @description Soft delete plan by plan.id.
     *
     *     Once a plan is deleted it cannot be undeleted.
     */
    delete: operations['deletePlan']
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/plans/{planId}/addons': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * List all available add-ons for plan
     * @description List all available add-ons for plan.
     */
    get: operations['listPlanAddons']
    put?: never
    /**
     * Create new add-on assignment for plan
     * @description Create new add-on assignment for plan.
     */
    post: operations['createPlanAddon']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/plans/{planId}/addons/{planAddonId}': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * Get add-on assignment for plan
     * @description Get add-on assignment for plan by id.
     */
    get: operations['getPlanAddon']
    /**
     * Update add-on assignment for plan
     * @description Update add-on assignment for plan.
     */
    put: operations['updatePlanAddon']
    post?: never
    /**
     * Delete add-on assignment for plan
     * @description Delete add-on assignment for plan.
     *
     *     Once a plan is deleted it cannot be undeleted.
     */
    delete: operations['deletePlanAddon']
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/plans/{planId}/archive': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /**
     * Archive plan version
     * @description Archive a plan version.
     */
    post: operations['archivePlan']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/plans/{planId}/publish': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /**
     * Publish plan
     * @description Publish a plan version.
     */
    post: operations['publishPlan']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/portal/meters/{meterSlug}/query': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * Query meter Query meter
     * @description Query meter for consumer portal. This endpoint is publicly exposable to consumers. Query meter for consumer portal. This endpoint is publicly exposable to consumers.
     */
    get: operations['queryPortalMeter']
    put?: never
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/portal/tokens': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * List consumer portal tokens
     * @description List tokens.
     */
    get: operations['listPortalTokens']
    put?: never
    /**
     * Create consumer portal token
     * @description Create a consumer portal token.
     */
    post: operations['createPortalToken']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/portal/tokens/invalidate': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /**
     * Invalidate portal tokens
     * @description Invalidates consumer portal tokens by ID or subject.
     */
    post: operations['invalidatePortalTokens']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/stripe/checkout/sessions': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /**
     * Create checkout session
     * @description Create checkout session.
     */
    post: operations['createStripeCheckoutSession']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/subjects': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * List subjects
     * @description List subjects.
     */
    get: operations['listSubjects']
    put?: never
    /**
     * Upsert subject
     * @description Upserts a subject. Creates or updates subject.
     *
     *     If the subject doesn't exist, it will be created.
     *     If the subject exists, it will be partially updated with the provided fields.
     */
    post: operations['upsertSubject']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/subjects/{subjectIdOrKey}': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * Get subject
     * @description Get subject by ID or key.
     */
    get: operations['getSubject']
    put?: never
    post?: never
    /**
     * Delete subject
     * @description Delete subject by ID or key.
     */
    delete: operations['deleteSubject']
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/subjects/{subjectIdOrKey}/entitlements': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * List entitlements
     * @description List all entitlements for a subject. For checking entitlement access, use the /value endpoint instead.
     */
    get: operations['listSubjectEntitlements']
    put?: never
    /**
     * Create an entitlement
     * @description OpenMeter has three types of entitlements: metered, boolean, and static. The type property determines the type of entitlement. The underlying feature has to be compatible with the entitlement type specified in the request (e.g., a metered entitlement needs a feature associated with a meter).
     *
     *     - Boolean entitlements define static feature access, e.g. "Can use SSO authentication".
     *     - Static entitlements let you pass along a configuration while granting access, e.g. "Using this feature with X Y settings" (passed in the config).
     *     - Metered entitlements have many use cases, from setting up usage-based access to implementing complex credit systems.  Example: The customer can use 10000 AI tokens during the usage period of the entitlement.
     *
     *     A given subject can only have one active (non-deleted) entitlement per featureKey. If you try to create a new entitlement for a featureKey that already has an active entitlement, the request will fail with a 409 error.
     *
     *     Once an entitlement is created you cannot modify it, only delete it.
     */
    post: operations['createEntitlement']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/subjects/{subjectIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/grants': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * List entitlement grants
     * @description List all grants issued for an entitlement. The entitlement can be defined either by its id or featureKey.
     */
    get: operations['listEntitlementGrants']
    put?: never
    /**
     * Create grant
     * @description Grants define a behavior of granting usage for a metered entitlement. They can have complicated recurrence and rollover rules, thanks to which you can define a wide range of access patterns with a single grant, in most cases you don't have to periodically create new grants. You can only issue grants for active metered entitlements.
     *
     *     A grant defines a given amount of usage that can be consumed for the entitlement. The grant is in effect between its effective date and its expiration date. Specifying both is mandatory for new grants.
     *
     *     Grants have a priority setting that determines their order of use. Lower numbers have higher priority, with 0 being the highest priority.
     *
     *     Grants can have a recurrence setting intended to automate the manual reissuing of grants. For example, a daily recurrence is equal to reissuing that same grant every day (ignoring rollover settings).
     *
     *     Rollover settings define what happens to the remaining balance of a grant at a reset. Balance_After_Reset = MIN(MaxRolloverAmount, MAX(Balance_Before_Reset, MinRolloverAmount))
     *
     *     Grants cannot be changed once created, only deleted. This is to ensure that balance is deterministic regardless of when it is queried.
     */
    post: operations['createGrant']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/subjects/{subjectIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/override': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    /**
     * Override entitlement
     * @description Overriding an entitlement creates a new entitlement from the provided inputs and soft deletes the previous entitlement for the provided subject-feature pair. If the previous entitlement is already deleted or otherwise doesnt exist, the override will fail.
     *
     *     This endpoint is useful for upgrades, downgrades, or other changes to entitlements that require a new entitlement to be created with zero downtime.
     */
    put: operations['overrideEntitlement']
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/subjects/{subjectIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/value': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * Get entitlement value
     * @description This endpoint should be used for access checks and enforcement. All entitlement types share the hasAccess property in their value response, but multiple other properties are returned based on the entitlement type.
     *
     *     For convenience reasons, /value works with both entitlementId and featureKey.
     */
    get: operations['getEntitlementValue']
    put?: never
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/subjects/{subjectIdOrKey}/entitlements/{entitlementId}': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * Get entitlement
     * @description Get entitlement by id. For checking entitlement access, use the /value endpoint instead.
     */
    get: operations['getEntitlement']
    put?: never
    post?: never
    /**
     * Delete entitlement
     * @description Deleting an entitlement revokes access to the associated feature. As a single subject can only have one entitlement per featureKey, when "migrating" features you have to delete the old entitlements as well.
     *     As access and status checks can be historical queries, deleting an entitlement populates the deletedAt timestamp. When queried for a time before that, the entitlement is still considered active, you cannot have retroactive changes to access, which is important for, among other things, auditing.
     */
    delete: operations['deleteEntitlement']
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/subjects/{subjectIdOrKey}/entitlements/{entitlementId}/history': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * Get entitlement history
     * @description Returns historical balance and usage data for the entitlement. The queried history can span accross multiple reset events.
     *
     *     BurndownHistory returns a continous history of segments, where the segments are seperated by events that changed either the grant burndown priority or the usage period.
     *
     *     WindowedHistory returns windowed usage data for the period enriched with balance information and the list of grants that were being burnt down in that window.
     */
    get: operations['getEntitlementHistory']
    put?: never
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/subjects/{subjectIdOrKey}/entitlements/{entitlementId}/reset': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /**
     * Reset entitlement
     * @description Reset marks the start of a new usage period for the entitlement and initiates grant rollover. At the start of a period usage is zerod out and grants are rolled over based on their rollover settings. It would typically be synced with the subjects billing period to enforce usage based on their subscription.
     *
     *     Usage is automatically reset for metered entitlements based on their usage period, but this endpoint allows to manually reset it at any time. When doing so the period anchor of the entitlement can be changed if needed.
     */
    post: operations['resetEntitlementUsage']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/subscriptions': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /** Create subscription */
    post: operations['createSubscription']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/subscriptions/{subscriptionId}': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /** Get subscription */
    get: operations['getSubscription']
    put?: never
    post?: never
    /**
     * Delete subscription
     * @description Deletes a subscription. Only scheduled subscriptions can be deleted.
     */
    delete: operations['deleteSubscription']
    options?: never
    head?: never
    /**
     * Edit subscription
     * @description Batch processing commands for manipulating running subscriptions.
     *     The key format is `/phases/{phaseKey}` or `/phases/{phaseKey}/items/{itemKey}`.
     */
    patch: operations['editSubscription']
    trace?: never
  }
  '/api/v1/subscriptions/{subscriptionId}/addons': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * List subscription addons
     * @description List all addons of a subscription. In the returned list will match to a set unique by addonId.
     */
    get: operations['listSubscriptionAddons']
    put?: never
    /**
     * Create subscription addon
     * @description Create a new subscription addon, either providing the key or the id of the addon.
     */
    post: operations['createSubscriptionAddon']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/subscriptions/{subscriptionId}/addons/{subscriptionAddonId}': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * Get subscription addon
     * @description Get a subscription addon by id.
     */
    get: operations['getSubscriptionAddon']
    put?: never
    post?: never
    delete?: never
    options?: never
    head?: never
    /**
     * Update subscription addon
     * @description Updates a subscription addon (allows changing the quantity: purchasing more instances or cancelling the current instances)
     */
    patch: operations['updateSubscriptionAddon']
    trace?: never
  }
  '/api/v1/subscriptions/{subscriptionId}/cancel': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /**
     * Cancel subscription
     * @description Cancels the subscription.
     *     Will result in a scheduling conflict if there are other subscriptions scheduled to start after the cancellation time.
     */
    post: operations['cancelSubscription']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/subscriptions/{subscriptionId}/change': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /**
     * Change subscription
     * @description Closes a running subscription and starts a new one according to the specification.
     *     Can be used for upgrades, downgrades, and plan changes.
     */
    post: operations['changeSubscription']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/subscriptions/{subscriptionId}/migrate': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /**
     * Migrate subscription
     * @description Migrates the subscripiton to the provided version of the current plan.
     *     If possible, the migration will be done immediately.
     *     If not, the migration will be scheduled to the end of the current billing period.
     */
    post: operations['migrateSubscription']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/subscriptions/{subscriptionId}/restore': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /**
     * Restore subscription
     * @description Restores a canceled subscription.
     *     Any subscription scheduled to start later will be deleted and this subscription will be continued indefinitely.
     */
    post: operations['restoreSubscription']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v1/subscriptions/{subscriptionId}/unschedule-cancelation': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /**
     * Unschedule cancelation
     * @description Cancels the scheduled cancelation.
     */
    post: operations['unscheduleCancelation']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/api/v2/events': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * List ingested events
     * @description List ingested events with advanced filtering and cursor pagination.
     */
    get: operations['listEventsV2']
    put?: never
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
}
export type webhooks = Record<string, never>
export interface components {
  schemas: {
    /** @description Add-on allows extending subscriptions with compatible plans with additional ratecards. */
    Addon: {
      /**
       * ID
       * @description A unique identifier for the resource.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      readonly id: string
      /**
       * Display name
       * @description Human-readable name for the resource. Between 1 and 256 characters.
       */
      name: string
      /**
       * Description
       * @description Optional description of the resource. Maximum 1024 characters.
       */
      description?: string
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata'] | null
      /**
       * Creation Time
       * Format: date-time
       * @description Timestamp of when the resource was created.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly createdAt: Date
      /**
       * Last Update Time
       * Format: date-time
       * @description Timestamp of when the resource was last updated.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly updatedAt: Date
      /**
       * Deletion Time
       * Format: date-time
       * @description Timestamp of when the resource was permanently deleted.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly deletedAt?: Date
      /**
       * Key
       * @description A semi-unique identifier for the resource.
       */
      key: string
      /**
       * Annotations
       * @description Set of key-value pairs managed by the system. Cannot be modified by user.
       */
      readonly annotations?: components['schemas']['Annotations']
      /**
       * Version
       * @description Version of the add-on. Incremented when the add-on is updated.
       * @default 1
       */
      readonly version: number
      /**
       * InstanceType
       * @description The instanceType of the add-ons. Can be "single" or "multiple".
       */
      instanceType: components['schemas']['AddonInstanceType']
      /**
       * Currency
       * @description The currency code of the add-on.
       * @default USD
       */
      currency: components['schemas']['CurrencyCode']
      /**
       * Effective start date
       * Format: date-time
       * @description The date and time when the add-on becomes effective. When not specified, the add-on is a draft.
       * @example 2023-01-01T01:01:01.001Z
       */
      readonly effectiveFrom?: Date
      /**
       * Effective end date
       * Format: date-time
       * @description The date and time when the add-on is no longer effective. When not specified, the add-on is effective indefinitely.
       * @example 2023-01-01T01:01:01.001Z
       */
      readonly effectiveTo?: Date
      /**
       * Status
       * @description The status of the add-on.
       *     Computed based on the effective start and end dates:
       *     - draft = no effectiveFrom
       *     - active = effectiveFrom <= now < effectiveTo
       *     - archived  = effectiveTo <= now
       */
      readonly status: components['schemas']['AddonStatus']
      /**
       * Rate cards
       * @description The rate cards of the add-on.
       */
      rateCards: components['schemas']['RateCard'][]
      /**
       * Validation errors
       * @description List of validation errors.
       */
      readonly validationErrors:
        | components['schemas']['ValidationError'][]
        | null
    }
    /** @description Resource create operation model. */
    AddonCreate: {
      /**
       * Display name
       * @description Human-readable name for the resource. Between 1 and 256 characters.
       */
      name: string
      /**
       * Description
       * @description Optional description of the resource. Maximum 1024 characters.
       */
      description?: string
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata'] | null
      /**
       * Key
       * @description A semi-unique identifier for the resource.
       */
      key: string
      /**
       * InstanceType
       * @description The instanceType of the add-ons. Can be "single" or "multiple".
       */
      instanceType: components['schemas']['AddonInstanceType']
      /**
       * Currency
       * @description The currency code of the add-on.
       * @default USD
       */
      currency: components['schemas']['CurrencyCode']
      /**
       * Rate cards
       * @description The rate cards of the add-on.
       */
      rateCards: components['schemas']['RateCard'][]
    }
    /**
     * @description The instanceType of the add-on.
     *     Single instance add-ons can be added to subscription only once while add-ons with multiple type can be added more then once.
     * @enum {string}
     */
    AddonInstanceType: 'single' | 'multiple'
    /**
     * @description Order by options for add-ons.
     * @enum {string}
     */
    AddonOrderBy: 'id' | 'key' | 'version' | 'created_at' | 'updated_at'
    /** @description Paginated response */
    AddonPaginatedResponse: {
      /**
       * @description The total number of items.
       * @example 500
       */
      totalCount: number
      /**
       * @description The page index.
       * @example 1
       */
      page: number
      /**
       * @description The maximum number of items per page.
       * @example 100
       */
      pageSize: number
      /** @description The items in the current page. */
      items: components['schemas']['Addon'][]
    }
    /** @description Resource update operation model. */
    AddonReplaceUpdate: {
      /**
       * Display name
       * @description Human-readable name for the resource. Between 1 and 256 characters.
       */
      name: string
      /**
       * Description
       * @description Optional description of the resource. Maximum 1024 characters.
       */
      description?: string
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata'] | null
      /**
       * InstanceType
       * @description The instanceType of the add-ons. Can be "single" or "multiple".
       */
      instanceType: components['schemas']['AddonInstanceType']
      /**
       * Rate cards
       * @description The rate cards of the add-on.
       */
      rateCards: components['schemas']['RateCard'][]
    }
    /**
     * @description The status of the add-on defined by the effectiveFrom and effectiveTo properties.
     * @enum {string}
     */
    AddonStatus: 'draft' | 'active' | 'archived'
    /** @description Address */
    Address: {
      /** @description Country code in [ISO 3166-1](https://www.iso.org/iso-3166-country-codes.html) alpha-2 format. */
      country?: components['schemas']['CountryCode']
      /** @description Postal code. */
      postalCode?: string
      /** @description State or province. */
      state?: string
      /** @description City. */
      city?: string
      /** @description First line of the address. */
      line1?: string
      /** @description Second line of the address. */
      line2?: string
      /** @description Phone number. */
      phoneNumber?: string
    }
    /** @description Alignment configuration for a plan or subscription. */
    Alignment: {
      /** @description Whether all Billable items and RateCards must align.
       *     Alignment means the Price's BillingCadence must align for both duration and anchor time. */
      billablesMustAlign?: boolean
    }
    /**
     * @description Set of key-value pairs managed by the system. Cannot be modified by user.
     * @example {
     *       "externalId": "019142cc-a016-796a-8113-1a942fecd26d"
     *     }
     */
    Annotations: {
      [key: string]: unknown
    }
    /** @description App.
     *     One of: stripe */
    App:
      | components['schemas']['StripeApp']
      | components['schemas']['SandboxApp']
      | components['schemas']['CustomInvoicingApp']
    /**
     * @description App capability.
     *
     *     Capabilities only exist in config so they don't extend the Resource model.
     * @example {
     *       "type": "collectPayments",
     *       "key": "stripe_collect_payment",
     *       "name": "Collect Payments",
     *       "description": "Stripe payments collects outstanding revenue with Stripe customer's default payment method."
     *     }
     */
    AppCapability: {
      /** @description The capability type. */
      type: components['schemas']['AppCapabilityType']
      /** @description Key */
      key: string
      /** @description The capability name. */
      name: string
      /** @description The capability description. */
      description: string
    }
    /**
     * @description App capability type.
     * @enum {string}
     */
    AppCapabilityType:
      | 'reportUsage'
      | 'reportEvents'
      | 'calculateTax'
      | 'invoiceCustomers'
      | 'collectPayments'
    /** @description Paginated response */
    AppPaginatedResponse: {
      /**
       * @description The total number of items.
       * @example 500
       */
      totalCount: number
      /**
       * @description The page index.
       * @example 1
       */
      page: number
      /**
       * @description The maximum number of items per page.
       * @example 100
       */
      pageSize: number
      /** @description The items in the current page. */
      items: components['schemas']['App'][]
    }
    /** @description App reference
     *
     *     Can be used as a short reference to an app if the full app object is not needed. */
    AppReference: {
      /**
       * @description The ID of the app.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      id: string
    }
    /** @description App ReplaceUpdate Model */
    AppReplaceUpdate:
      | components['schemas']['StripeAppReplaceUpdate']
      | components['schemas']['SandboxAppReplaceUpdate']
      | components['schemas']['CustomInvoicingAppReplaceUpdate']
    /**
     * @description App installed status.
     * @enum {string}
     */
    AppStatus: 'ready' | 'unauthorized'
    /**
     * @description Type of the app.
     * @enum {string}
     */
    AppType: 'stripe' | 'sandbox' | 'custom_invoicing'
    /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
    BadRequestProblemResponse: components['schemas']['UnexpectedProblemResponse']
    /** @description The balance history window. */
    BalanceHistoryWindow: {
      period: components['schemas']['Period']
      /**
       * Format: double
       * @description The total usage of the feature in the period.
       * @example 100
       */
      readonly usage: number
      /**
       * Format: double
       * @description The entitlement balance at the start of the period.
       * @example 100
       */
      readonly balanceAtStart: number
    }
    /** @description Customer specific merged profile.
     *
     *     This profile is calculated from the customer override and the billing profile it references or the default.
     *
     *     Thus this does not have any kind of resource fields, only the calculated values. */
    BillingCustomerProfile: {
      /** @description The name and contact information for the supplier this billing profile represents */
      readonly supplier: components['schemas']['BillingParty']
      /** @description The billing workflow settings for this profile */
      readonly workflow: components['schemas']['BillingWorkflow']
      /** @description The applications used by this billing profile.
       *
       *     Expand settings govern if this includes the whole app object or just the ID references. */
      readonly apps: components['schemas']['BillingProfileAppsOrReference']
    }
    /** @description A percentage discount. */
    BillingDiscountPercentage: {
      /**
       * Percentage
       * @description The percentage of the discount.
       */
      percentage: components['schemas']['Percentage']
      /**
       * @description Correlation ID for the discount.
       *
       *     This is used to link discounts across different invoices (progressive billing use case).
       *
       *     If not provided, the invoicing engine will auto-generate one. When editing an invoice line,
       *     please make sure to keep the same correlation ID of the discount or in progressive billing
       *     setups the discount amounts might be incorrect.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      correlationId?: string
    }
    /** @description The reason for the discount. */
    BillingDiscountReason:
      | components['schemas']['DiscountReasonMaximumSpend']
      | components['schemas']['DiscountReasonRatecardPercentage']
      | components['schemas']['DiscountReasonRatecardUsage']
    /** @description A usage discount. */
    BillingDiscountUsage: {
      /**
       * Usage
       * @description The quantity of the usage discount.
       *
       *     Must be positive.
       */
      quantity: components['schemas']['Numeric']
      /**
       * @description Correlation ID for the discount.
       *
       *     This is used to link discounts across different invoices (progressive billing use case).
       *
       *     If not provided, the invoicing engine will auto-generate one. When editing an invoice line,
       *     please make sure to keep the same correlation ID of the discount or in progressive billing
       *     setups the discount amounts might be incorrect.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      correlationId?: string
    }
    /** @description A discount by type. */
    BillingDiscounts: {
      /** @description The percentage discount. */
      percentage?: components['schemas']['BillingDiscountPercentage']
      /** @description The usage discount. */
      usage?: components['schemas']['BillingDiscountUsage']
    }
    /** @description Party represents a person or business entity. */
    BillingParty: {
      /** @description Unique identifier for the party (if available) */
      readonly id?: string
      /** @description Legal name or representation of the organization. */
      name?: string
      /** @description The entity's legal ID code used for tax purposes. They may have
       *     other numbers, but we're only interested in those valid for tax purposes. */
      taxId?: components['schemas']['BillingPartyTaxIdentity']
      /** @description Regular post addresses for where information should be sent if needed. */
      addresses?: components['schemas']['Address'][]
    }
    /** @description Resource update operation model. */
    BillingPartyReplaceUpdate: {
      /** @description Legal name or representation of the organization. */
      name?: string
      /** @description The entity's legal ID code used for tax purposes. They may have
       *     other numbers, but we're only interested in those valid for tax purposes. */
      taxId?: components['schemas']['BillingPartyTaxIdentity']
      /** @description Regular post addresses for where information should be sent if needed. */
      addresses?: components['schemas']['Address'][]
    }
    /** @description Identity stores the details required to identify an entity for tax purposes in a specific country. */
    BillingPartyTaxIdentity: {
      /** @description Normalized tax code shown on the original identity document. */
      code?: components['schemas']['BillingTaxIdentificationCode']
    }
    /** @description BillingProfile represents a billing profile */
    BillingProfile: {
      /**
       * ID
       * @description A unique identifier for the resource.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      readonly id: string
      /**
       * Display name
       * @description Human-readable name for the resource. Between 1 and 256 characters.
       */
      name: string
      /**
       * Description
       * @description Optional description of the resource. Maximum 1024 characters.
       */
      description?: string
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata'] | null
      /**
       * Creation Time
       * Format: date-time
       * @description Timestamp of when the resource was created.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly createdAt: Date
      /**
       * Last Update Time
       * Format: date-time
       * @description Timestamp of when the resource was last updated.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly updatedAt: Date
      /**
       * Deletion Time
       * Format: date-time
       * @description Timestamp of when the resource was permanently deleted.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly deletedAt?: Date
      /** @description The name and contact information for the supplier this billing profile represents */
      supplier: components['schemas']['BillingParty']
      /** @description The billing workflow settings for this profile */
      readonly workflow: components['schemas']['BillingWorkflow']
      /** @description The applications used by this billing profile.
       *
       *     Expand settings govern if this includes the whole app object or just the ID references. */
      readonly apps: components['schemas']['BillingProfileAppsOrReference']
      /** @description Is this the default profile? */
      default: boolean
    }
    /** @description BillingProfileAppReferences represents the references (id, type) to the apps used by a billing profile */
    BillingProfileAppReferences: {
      /** @description The tax app used for this workflow */
      readonly tax: components['schemas']['AppReference']
      /** @description The invoicing app used for this workflow */
      readonly invoicing: components['schemas']['AppReference']
      /** @description The payment app used for this workflow */
      readonly payment: components['schemas']['AppReference']
    }
    /** @description BillingProfileApps represents the applications used by a billing profile */
    BillingProfileApps: {
      /** @description The tax app used for this workflow */
      readonly tax: components['schemas']['App']
      /** @description The invoicing app used for this workflow */
      readonly invoicing: components['schemas']['App']
      /** @description The payment app used for this workflow */
      readonly payment: components['schemas']['App']
    }
    /** @description BillingProfileAppsCreate represents the input for creating a billing profile's apps */
    BillingProfileAppsCreate: {
      /** @description The tax app used for this workflow */
      tax: string
      /** @description The invoicing app used for this workflow */
      invoicing: string
      /** @description The payment app used for this workflow */
      payment: string
    }
    /** @description ProfileAppsOrReference represents the union of ProfileApps and ProfileAppReferences
     *     for a billing profile. */
    BillingProfileAppsOrReference:
      | components['schemas']['BillingProfileApps']
      | components['schemas']['BillingProfileAppReferences']
    /** @description BillingProfileCreate represents the input for creating a billing profile */
    BillingProfileCreate: {
      /**
       * Display name
       * @description Human-readable name for the resource. Between 1 and 256 characters.
       */
      name: string
      /**
       * Description
       * @description Optional description of the resource. Maximum 1024 characters.
       */
      description?: string
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata'] | null
      /** @description The name and contact information for the supplier this billing profile represents */
      supplier: components['schemas']['BillingParty']
      /** @description Is this the default profile? */
      default: boolean
      /** @description The billing workflow settings for this profile. */
      workflow: components['schemas']['BillingWorkflowCreate']
      /** @description The apps used by this billing profile. */
      apps: components['schemas']['BillingProfileAppsCreate']
    }
    /** @description Customer override values. */
    BillingProfileCustomerOverride: {
      /**
       * Creation Time
       * Format: date-time
       * @description Timestamp of when the resource was created.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly createdAt: Date
      /**
       * Last Update Time
       * Format: date-time
       * @description Timestamp of when the resource was last updated.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly updatedAt: Date
      /**
       * @description The billing profile this override is associated with.
       *
       *     If empty the default profile is looked up dynamically.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      billingProfileId?: string
      /**
       * @description The customer id this override is associated with.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      customerId: string
    }
    /** @description Payload for creating a new or updating an existing customer override. */
    BillingProfileCustomerOverrideCreate: {
      /**
       * @description The billing profile this override is associated with.
       *
       *     If not provided, the default billing profile is chosen if available.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      billingProfileId?: string
    }
    /**
     * @description CustomerOverrideExpand specifies the parts of the profile to expand.
     * @enum {string}
     */
    BillingProfileCustomerOverrideExpand: 'apps' | 'customer'
    /**
     * @description Order by options for customers.
     * @enum {string}
     */
    BillingProfileCustomerOverrideOrderBy:
      | 'customerId'
      | 'customerName'
      | 'customerKey'
      | 'customerPrimaryEmail'
      | 'customerCreatedAt'
    /** @description Customer specific workflow overrides. */
    BillingProfileCustomerOverrideWithDetails: {
      /** @description The customer override values.
       *
       *     If empty the merged values are calculated based on the default profile. */
      customerOverride?: components['schemas']['BillingProfileCustomerOverride']
      /**
       * @description The billing profile the customerProfile is associated with at the time of query.
       *
       *     customerOverride contains the explicit mapping set in the customer override object. If that is
       *     empty, then the baseBillingProfileId is the default profile.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      baseBillingProfileId: string
      /** @description Merged billing profile with the customer specific overrides. */
      customerProfile?: components['schemas']['BillingCustomerProfile']
      /** @description The customer this override belongs to. */
      customer?: components['schemas']['Customer']
    }
    /** @description Paginated response */
    BillingProfileCustomerOverrideWithDetailsPaginatedResponse: {
      /**
       * @description The total number of items.
       * @example 500
       */
      totalCount: number
      /**
       * @description The page index.
       * @example 1
       */
      page: number
      /**
       * @description The maximum number of items per page.
       * @example 100
       */
      pageSize: number
      /** @description The items in the current page. */
      items: components['schemas']['BillingProfileCustomerOverrideWithDetails'][]
    }
    /**
     * @description BillingProfileExpand details what profile fields to expand
     * @enum {string}
     */
    BillingProfileExpand: 'apps'
    /**
     * @description BillingProfileOrderBy specifies the ordering options for profiles
     * @enum {string}
     */
    BillingProfileOrderBy: 'createdAt' | 'updatedAt' | 'default' | 'name'
    /** @description Paginated response */
    BillingProfilePaginatedResponse: {
      /**
       * @description The total number of items.
       * @example 500
       */
      totalCount: number
      /**
       * @description The page index.
       * @example 1
       */
      page: number
      /**
       * @description The maximum number of items per page.
       * @example 100
       */
      pageSize: number
      /** @description The items in the current page. */
      items: components['schemas']['BillingProfile'][]
    }
    /** @description BillingProfileReplaceUpdate represents the input for updating a billing profile
     *
     *     The apps field cannot be updated directly, if an app change is desired a new
     *     profile should be created. */
    BillingProfileReplaceUpdateWithWorkflow: {
      /**
       * Display name
       * @description Human-readable name for the resource. Between 1 and 256 characters.
       */
      name: string
      /**
       * Description
       * @description Optional description of the resource. Maximum 1024 characters.
       */
      description?: string
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata'] | null
      /** @description The name and contact information for the supplier this billing profile represents */
      supplier: components['schemas']['BillingParty']
      /** @description Is this the default profile? */
      default: boolean
      /** @description The billing workflow settings for this profile. */
      workflow: components['schemas']['BillingWorkflow']
    }
    /** @description TaxIdentificationCode is a normalized tax code shown on the original identity document. */
    BillingTaxIdentificationCode: string
    /** @description BillingWorkflow represents the settings for a billing workflow. */
    BillingWorkflow: {
      /** @description The collection settings for this workflow */
      collection?: components['schemas']['BillingWorkflowCollectionSettings']
      /** @description The invoicing settings for this workflow */
      invoicing?: components['schemas']['BillingWorkflowInvoicingSettings']
      /** @description The payment settings for this workflow */
      payment?: components['schemas']['BillingWorkflowPaymentSettings']
      /** @description The tax settings for this workflow */
      tax?: components['schemas']['BillingWorkflowTaxSettings']
    }
    /** @description The alignment for collecting the pending line items into an invoice.
     *
     *     Defaults to subscription, which means that we are to create a new invoice every time the
     *     a subscription period starts (for in advance items) or ends (for in arrears items). */
    BillingWorkflowCollectionAlignment: components['schemas']['BillingWorkflowCollectionAlignmentSubscription']
    /** @description BillingWorkflowCollectionAlignmentSubscription specifies the alignment for collecting the pending line items
     *     into an invoice. */
    BillingWorkflowCollectionAlignmentSubscription: {
      /**
       * @description The type of alignment.
       * @enum {string}
       */
      type: 'subscription'
    }
    /** @description Workflow collection specifies how to collect the pending line items for an invoice */
    BillingWorkflowCollectionSettings: {
      /**
       * @description The alignment for collecting the pending line items into an invoice.
       * @default {
       *       "type": "subscription"
       *     }
       */
      alignment?: components['schemas']['BillingWorkflowCollectionAlignment']
      /**
       * Format: ISO8601
       * @description This grace period can be used to delay the collection of the pending line items specified in
       *     alignment.
       *
       *     This is useful, in case of multiple subscriptions having slightly different billing periods.
       * @default PT1H
       * @example P1D
       */
      interval?: string
    }
    /** @description Resource create operation model. */
    BillingWorkflowCreate: {
      /** @description The collection settings for this workflow */
      collection?: components['schemas']['BillingWorkflowCollectionSettings']
      /** @description The invoicing settings for this workflow */
      invoicing?: components['schemas']['BillingWorkflowInvoicingSettings']
      /** @description The payment settings for this workflow */
      payment?: components['schemas']['BillingWorkflowPaymentSettings']
      /** @description The tax settings for this workflow */
      tax?: components['schemas']['BillingWorkflowTaxSettings']
    }
    /**
     * Workflow invoice settings
     * @description BillingWorkflowInvoicingSettings represents the invoice settings for a billing workflow
     */
    BillingWorkflowInvoicingSettings: {
      /**
       * @description Whether to automatically issue the invoice after the draftPeriod has passed.
       * @default true
       */
      autoAdvance?: boolean
      /**
       * Format: ISO8601
       * @description The period for the invoice to be kept in draft status for manual reviews.
       * @default P0D
       * @example P1D
       */
      draftPeriod?: string
      /**
       * Format: ISO8601
       * @description The period after which the invoice is due.
       *     With some payment solutions it's only applicable for manual collection method.
       * @default P30D
       * @example P30D
       */
      dueAfter?: string
      /**
       * @description Should progressive billing be allowed for this workflow?
       * @default false
       */
      progressiveBilling?: boolean
      /** @description Default tax configuration to apply to the invoices. */
      defaultTaxConfig?: components['schemas']['TaxConfig']
    }
    /**
     * Workflow payment settings
     * @description BillingWorkflowPaymentSettings represents the payment settings for a billing workflow
     */
    BillingWorkflowPaymentSettings: {
      /**
       * @description The payment method for the invoice.
       * @default charge_automatically
       */
      collectionMethod?: components['schemas']['CollectionMethod']
    }
    /**
     * Workflow tax settings
     * @description BillingWorkflowTaxSettings represents the tax settings for a billing workflow
     */
    BillingWorkflowTaxSettings: {
      /**
       * @description Enable automatic tax calculation when tax is supported by the app.
       *     For example, with Stripe Invoicing when enabled, tax is calculated via Stripe Tax.
       * @default true
       */
      enabled?: boolean
      /**
       * @description Enforce tax calculation when tax is supported by the app.
       *     When enabled, OpenMeter will not allow to create an invoice without tax calculation.
       *     Enforcement is different per apps, for example, Stripe app requires customer
       *     to have a tax location when starting a paid subscription.
       * @default false
       */
      enforced?: boolean
    }
    /** @description Stripe CheckoutSession.custom_text */
    CheckoutSessionCustomTextAfterSubmitParams: {
      /** @description Custom text that should be displayed after the payment confirmation button. */
      afterSubmit?: {
        message?: string
      }
      /** @description Custom text that should be displayed alongside shipping address collection. */
      shippingAddress?: {
        message?: string
      }
      /** @description Custom text that should be displayed alongside the payment confirmation button. */
      submit?: {
        message?: string
      }
      /** @description Custom text that should be displayed in place of the default terms of service agreement text. */
      termsOfServiceAcceptance?: {
        message?: string
      }
    }
    /**
     * @description Stripe CheckoutSession.ui_mode
     * @enum {string}
     */
    CheckoutSessionUIMode: 'embedded' | 'hosted'
    /** @description Response from the client app (OpenMeter backend) to start the OAuth2 flow. */
    ClientAppStartResponse: {
      /** @description The URL to start the OAuth2 authorization code grant flow. */
      url: string
    }
    /**
     * Collection method
     * @description CollectionMethod specifies how the invoice should be collected (automatic vs manual)
     * @enum {string}
     */
    CollectionMethod: 'charge_automatically' | 'send_invoice'
    /** @description The request could not be completed due to a conflict with the current state of the target resource. */
    ConflictProblemResponse: components['schemas']['UnexpectedProblemResponse']
    /**
     * @description [ISO 3166-1](https://www.iso.org/iso-3166-country-codes.html) alpha-2 country code.
     *     Custom two-letter country codes are also supported for convenience.
     * @example US
     */
    CountryCode: string
    /** @description Create Stripe checkout session tax ID collection. */
    CreateCheckoutSessionTaxIdCollection: {
      /** @description Enable tax ID collection during checkout. Defaults to false. */
      enabled: boolean
      /** @description Describes whether a tax ID is required during checkout. Defaults to never. */
      required?: components['schemas']['CreateCheckoutSessionTaxIdCollectionRequired']
    }
    /**
     * @description Create Stripe checkout session tax ID collection required.
     * @enum {string}
     */
    CreateCheckoutSessionTaxIdCollectionRequired: 'if_supported' | 'never'
    /**
     * @description Specify whether Checkout should collect the customer’s billing address.
     * @enum {string}
     */
    CreateStripeCheckoutSessionBillingAddressCollection: 'auto' | 'required'
    /** @description Configure fields for the Checkout Session to gather active consent from customers. */
    CreateStripeCheckoutSessionConsentCollection: {
      /** @description Determines the position and visibility of the payment method reuse agreement in the UI.
       *     When set to auto, Stripe’s defaults will be used. When set to hidden, the payment method reuse agreement text will always be hidden in the UI. */
      paymentMethodReuseAgreement?: components['schemas']['CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreement']
      /** @description If set to auto, enables the collection of customer consent for promotional communications.
       *     The Checkout Session will determine whether to display an option to opt into promotional
       *     communication from the merchant depending on the customer’s locale. Only available to US merchants. */
      promotions?: components['schemas']['CreateStripeCheckoutSessionConsentCollectionPromotions']
      /** @description If set to required, it requires customers to check a terms of service checkbox before being able to pay.
       *     There must be a valid terms of service URL set in your Stripe Dashboard settings.
       *     https://dashboard.stripe.com/settings/public */
      termsOfService?: components['schemas']['CreateStripeCheckoutSessionConsentCollectionTermsOfService']
    }
    /** @description Create Stripe checkout session payment method reuse agreement. */
    CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreement: {
      position?: components['schemas']['CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPosition']
    }
    /**
     * @description Create Stripe checkout session consent collection agreement position.
     * @enum {string}
     */
    CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPosition:
      | 'auto'
      | 'hidden'
    /**
     * @description Create Stripe checkout session consent collection promotions.
     * @enum {string}
     */
    CreateStripeCheckoutSessionConsentCollectionPromotions: 'auto' | 'none'
    /**
     * @description Create Stripe checkout session consent collection terms of service.
     * @enum {string}
     */
    CreateStripeCheckoutSessionConsentCollectionTermsOfService:
      | 'none'
      | 'required'
    /** @description Controls what fields on Customer can be updated by the Checkout Session. */
    CreateStripeCheckoutSessionCustomerUpdate: {
      /** @description Describes whether Checkout saves the billing address onto customer.address.
       *     To always collect a full billing address, use billing_address_collection.
       *     Defaults to never. */
      address?: components['schemas']['CreateStripeCheckoutSessionCustomerUpdateBehavior']
      /** @description Describes whether Checkout saves the name onto customer.name.
       *     Defaults to never. */
      name?: components['schemas']['CreateStripeCheckoutSessionCustomerUpdateBehavior']
      /** @description Describes whether Checkout saves shipping information onto customer.shipping.
       *     To collect shipping information, use shipping_address_collection.
       *     Defaults to never. */
      shipping?: components['schemas']['CreateStripeCheckoutSessionCustomerUpdateBehavior']
    }
    /**
     * @description Create Stripe checkout session customer update behavior.
     * @enum {string}
     */
    CreateStripeCheckoutSessionCustomerUpdateBehavior: 'auto' | 'never'
    /**
     * @description Create Stripe checkout session redirect on completion.
     * @enum {string}
     */
    CreateStripeCheckoutSessionRedirectOnCompletion:
      | 'always'
      | 'if_required'
      | 'never'
    /**
     * @description Create Stripe checkout session request.
     * @example {
     *       "customer": {
     *         "name": "ACME, Inc.",
     *         "currency": "USD",
     *         "usageAttribution": {
     *           "subjectKeys": [
     *             "my-identifier"
     *           ]
     *         }
     *       },
     *       "options": {
     *         "currency": "USD",
     *         "successURL": "http://example.com",
     *         "billingAddressCollection": "required",
     *         "taxIdCollection": {
     *           "enabled": true,
     *           "required": "if_supported"
     *         },
     *         "customerUpdate": {
     *           "name": "auto",
     *           "address": "auto"
     *         }
     *       }
     *     }
     */
    CreateStripeCheckoutSessionRequest: {
      /**
       * @description If not provided, the default Stripe app is used if any.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      appId?: string
      /** @description Provide a customer ID or key to use an existing OpenMeter customer.
       *     or provide a customer object to create a new customer. */
      customer:
        | components['schemas']['CustomerId']
        | components['schemas']['CustomerKey']
        | components['schemas']['CustomerCreate']
      /** @description Stripe customer ID.
       *     If not provided OpenMeter creates a new Stripe customer or
       *     uses the OpenMeter customer's default Stripe customer ID. */
      stripeCustomerId?: string
      /** @description Options passed to Stripe when creating the checkout session. */
      options: components['schemas']['CreateStripeCheckoutSessionRequestOptions']
    }
    /** @description Create Stripe checkout session options
     *     See https://docs.stripe.com/api/checkout/sessions/create */
    CreateStripeCheckoutSessionRequestOptions: {
      /** @description Specify whether Checkout should collect the customer’s billing address. Defaults to auto. */
      billingAddressCollection?: components['schemas']['CreateStripeCheckoutSessionBillingAddressCollection']
      /** @description If set, Checkout displays a back button and customers will be directed to this URL if they decide to cancel payment and return to your website.
       *     This parameter is not allowed if ui_mode is embedded. */
      cancelURL?: string
      /** @description A unique string to reference the Checkout Session. This can be a customer ID, a cart ID, or similar, and can be used to reconcile the session with your internal systems. */
      clientReferenceID?: string
      /** @description Controls what fields on Customer can be updated by the Checkout Session. */
      customerUpdate?: components['schemas']['CreateStripeCheckoutSessionCustomerUpdate']
      /** @description Configure fields for the Checkout Session to gather active consent from customers. */
      consentCollection?: components['schemas']['CreateStripeCheckoutSessionConsentCollection']
      /** @description Three-letter ISO currency code, in lowercase. */
      currency?: components['schemas']['CurrencyCode']
      /** @description Display additional text for your customers using custom text. */
      customText?: components['schemas']['CheckoutSessionCustomTextAfterSubmitParams']
      /**
       * Format: int64
       * @description The Epoch time in seconds at which the Checkout Session will expire.
       *     It can be anywhere from 30 minutes to 24 hours after Checkout Session creation. By default, this value is 24 hours from creation.
       */
      expiresAt?: number
      locale?: string
      /** @description Set of key-value pairs that you can attach to an object.
       *     This can be useful for storing additional information about the object in a structured format.
       *     Individual keys can be unset by posting an empty value to them.
       *     All keys can be unset by posting an empty value to metadata. */
      metadata?: {
        [key: string]: string
      }
      /** @description The URL to redirect your customer back to after they authenticate or cancel their payment on the payment method’s app or site.
       *     This parameter is required if ui_mode is embedded and redirect-based payment methods are enabled on the session. */
      returnURL?: string
      /** @description The URL to which Stripe should send customers when payment or setup is complete.
       *     This parameter is not allowed if ui_mode is embedded.
       *     If you’d like to use information from the successful Checkout Session on your page, read the guide on customizing your success page:
       *     https://docs.stripe.com/payments/checkout/custom-success-page */
      successURL?: string
      /** @description The UI mode of the Session. Defaults to hosted. */
      uiMode?: components['schemas']['CheckoutSessionUIMode']
      /** @description A list of the types of payment methods (e.g., card) this Checkout Session can accept. */
      paymentMethodTypes?: string[]
      /** @description This parameter applies to ui_mode: embedded. Defaults to always.
       *     Learn more about the redirect behavior of embedded sessions at
       *     https://docs.stripe.com/payments/checkout/custom-success-page?payment-ui=embedded-form */
      redirectOnCompletion?: components['schemas']['CreateStripeCheckoutSessionRedirectOnCompletion']
      /** @description Controls tax ID collection during checkout. */
      taxIdCollection?: components['schemas']['CreateCheckoutSessionTaxIdCollection']
    }
    /** @description Create Stripe Checkout Session response. */
    CreateStripeCheckoutSessionResult: {
      /**
       * @description The OpenMeter customer ID.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      customerId: string
      /** @description The Stripe customer ID. */
      stripeCustomerId: string
      /** @description The checkout session ID. */
      sessionId: string
      /** @description The checkout session setup intent ID. */
      setupIntentId: string
      /** @description The client secret of the checkout session.
       *     This can be used to initialize Stripe.js for your client-side implementation. */
      clientSecret?: string
      /** @description A unique string to reference the Checkout Session.
       *     This can be a customer ID, a cart ID, or similar, and can be used to reconcile the session with your internal systems. */
      clientReferenceId?: string
      /** @description Customer's email address provided to Stripe. */
      customerEmail?: string
      /** @description Three-letter ISO currency code, in lowercase. */
      currency?: components['schemas']['CurrencyCode']
      /**
       * Format: date-time
       * @description Timestamp at which the checkout session was created.
       * @example 2023-01-01T01:01:01.001Z
       */
      createdAt: Date
      /**
       * Format: date-time
       * @description Timestamp at which the checkout session will expire.
       * @example 2023-01-01T01:01:01.001Z
       */
      expiresAt?: Date
      /** @description Set of key-value pairs attached to the checkout session. */
      metadata?: {
        [key: string]: string
      }
      /** @description The status of the checkout session. */
      status?: string
      /** @description URL to show the checkout session. */
      url?: string
      /** @description Mode
       *     Always `setup` for now. */
      mode: components['schemas']['StripeCheckoutSessionMode']
      /** @description Cancel URL. */
      cancelURL?: string
      /** @description Success URL. */
      successURL?: string
      /** @description Return URL. */
      returnURL?: string
    }
    /** @description CreditNoteOriginalInvoiceRef is used to reference the original invoice that a credit note is based on. */
    CreditNoteOriginalInvoiceRef: {
      /**
       * @description Type of the invoice.
       * @enum {string}
       */
      type: 'credit_note_original_invoice'
      /**
       * Format: date-time
       * @description IssueAt reflects the time the document was issued.
       * @example 2023-01-01T01:01:01.001Z
       */
      readonly issuedAt?: Date
      /** @description (Serial) Number of the referenced document. */
      readonly number?: components['schemas']['InvoiceNumber']
      /**
       * Format: uri
       * @description Link to the source document.
       */
      readonly url: string
    } & WithRequired<components['schemas']['InvoiceGenericDocumentRef'], 'type'>
    /** @description Currency describes a currency supported by OpenMeter. */
    Currency: {
      /** @description The currency ISO code. */
      code: components['schemas']['CurrencyCode']
      /** @description The currency name. */
      name: string
      /** @description The currency symbol. */
      symbol: string
      /**
       * Format: uint32
       * @description Subunit of the currency.
       */
      subunits: number
    }
    /**
     * @description Three-letter [ISO4217](https://www.iso.org/iso-4217-currency-codes.html) currency code.
     *     Custom three-letter currency codes are also supported for convenience.
     * @example USD
     */
    CurrencyCode: string
    /** @description Custom Invoicing app can be used for interface with any invoicing or payment system.
     *
     *     This app provides ways to manipulate invoices and payments, however the integration
     *     must rely on Notifications API to get notified about invoice changes. */
    CustomInvoicingApp: {
      /**
       * ID
       * @description A unique identifier for the resource.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      readonly id: string
      /**
       * Display name
       * @description Human-readable name for the resource. Between 1 and 256 characters.
       */
      name: string
      /**
       * Description
       * @description Optional description of the resource. Maximum 1024 characters.
       */
      description?: string
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata'] | null
      /**
       * Creation Time
       * Format: date-time
       * @description Timestamp of when the resource was created.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly createdAt: Date
      /**
       * Last Update Time
       * Format: date-time
       * @description Timestamp of when the resource was last updated.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly updatedAt: Date
      /**
       * Deletion Time
       * Format: date-time
       * @description Timestamp of when the resource was permanently deleted.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly deletedAt?: Date
      /** @description The marketplace listing that this installed app is based on. */
      readonly listing: components['schemas']['MarketplaceListing']
      /** @description Status of the app connection. */
      readonly status: components['schemas']['AppStatus']
      /** @description Default for the app type
       *     Only one app of each type can be default. */
      default: boolean
      /**
       * @description The app's type is CustomInvoicing. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'custom_invoicing'
      /** @description Enable draft.sync hook.
       *
       *     If the hook is not enabled, the invoice will be progressed to the next state automatically. */
      enableDraftSyncHook: boolean
      /** @description Enable issuing.sync hook.
       *
       *     If the hook is not enabled, the invoice will be progressed to the next state automatically. */
      enableIssuingSyncHook: boolean
    }
    /** @description Resource update operation model. */
    CustomInvoicingAppReplaceUpdate: {
      /**
       * Display name
       * @description Human-readable name for the resource. Between 1 and 256 characters.
       */
      name: string
      /**
       * Description
       * @description Optional description of the resource. Maximum 1024 characters.
       */
      description?: string
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata'] | null
      /** @description Default for the app type
       *     Only one app of each type can be default. */
      default: boolean
      /**
       * @description The app's type is CustomInvoicing. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'custom_invoicing'
      /** @description Enable draft.sync hook.
       *
       *     If the hook is not enabled, the invoice will be progressed to the next state automatically. */
      enableDraftSyncHook: boolean
      /** @description Enable issuing.sync hook.
       *
       *     If the hook is not enabled, the invoice will be progressed to the next state automatically. */
      enableIssuingSyncHook: boolean
    }
    /** @description Custom Invoicing Customer App Data. */
    CustomInvoicingCustomerAppData: {
      /** @description The installed custom invoicing app this data belongs to. */
      readonly app?: components['schemas']['CustomInvoicingApp']
      /**
       * App ID
       * @description The app ID.
       *     If not provided, it will use the global default for the app type.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      id?: string
      /**
       * @description The app name. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'custom_invoicing'
      /** @description Metadata to be used by the custom invoicing provider. */
      metadata?: components['schemas']['Metadata']
    }
    /** @description Information to finalize the draft details of an invoice. */
    CustomInvoicingDraftSynchronizedRequest: {
      /** @description The result of the synchronization. */
      invoicing?: components['schemas']['CustomInvoicingSyncResult']
    }
    /** @description Information to finalize the invoicing details of an invoice. */
    CustomInvoicingFinalizedInvoicingRequest: {
      /** @description If set the invoice's number will be set to this value. */
      invoiceNumber?: components['schemas']['InvoiceNumber']
      /**
       * Format: date-time
       * @description If set the invoice's sent to customer at will be set to this value.
       * @example 2023-01-01T01:01:01.001Z
       */
      sentToCustomerAt?: Date
    }
    /** @description Information to finalize the payment details of an invoice. */
    CustomInvoicingFinalizedPaymentRequest: {
      /** @description If set the invoice's payment external ID will be set to this value. */
      externalId?: string
    }
    /** @description Information to finalize the invoice.
     *
     *     If invoicing.invoiceNumber is not set, then a new invoice number will be generated (INV- prefix). */
    CustomInvoicingFinalizedRequest: {
      /** @description The result of the synchronization. */
      invoicing?: components['schemas']['CustomInvoicingFinalizedInvoicingRequest']
      /** @description The result of the payment synchronization. */
      payment?: components['schemas']['CustomInvoicingFinalizedPaymentRequest']
    }
    /** @description Mapping between line discounts and external IDs. */
    CustomInvoicingLineDiscountExternalIdMapping: {
      /**
       * @description The line discount ID.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      lineDiscountId: string
      /** @description The external ID (e.g. custom invoicing system's ID). */
      externalId: string
    }
    /** @description Mapping between lines and external IDs. */
    CustomInvoicingLineExternalIdMapping: {
      /**
       * @description The line ID.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      lineId: string
      /** @description The external ID (e.g. custom invoicing system's ID). */
      externalId: string
    }
    /**
     * @description Payment trigger to execute on a finalized invoice.
     * @enum {string}
     */
    CustomInvoicingPaymentTrigger:
      | 'paid'
      | 'payment_failed'
      | 'payment_uncollectible'
      | 'payment_overdue'
      | 'action_required'
      | 'void'
    /** @description Information to synchronize the invoice.
     *
     *     Can be used to store external app's IDs on the invoice or lines. */
    CustomInvoicingSyncResult: {
      /** @description If set the invoice's number will be set to this value. */
      invoiceNumber?: components['schemas']['InvoiceNumber']
      /** @description If set the invoice's invoicing external ID will be set to this value. */
      externalId?: string
      /** @description If set the invoice's line external IDs will be set to this value.
       *
       *     This can be used to reference the external system's entities in the
       *     invoice. */
      lineExternalIds?: components['schemas']['CustomInvoicingLineExternalIdMapping'][]
      /** @description If set the invoice's line discount external IDs will be set to this value.
       *
       *     This can be used to reference the external system's entities in the
       *     invoice. */
      lineDiscountExternalIds?: components['schemas']['CustomInvoicingLineDiscountExternalIdMapping'][]
    }
    /** @description Custom invoicing tax config. */
    CustomInvoicingTaxConfig: {
      /**
       * Tax code
       * @description Tax code.
       *
       *     The tax code should be interpreted by the custom invoicing provider.
       */
      code: string
    }
    /** @description Update payment status request.
     *
     *     Can be used to manipulate invoice's payment status (when custominvoicing app is being used). */
    CustomInvoicingUpdatePaymentStatusRequest: {
      /** @description The trigger to be executed on the invoice. */
      trigger: components['schemas']['CustomInvoicingPaymentTrigger']
    }
    /** @description Plan input for custom subscription creation (without key and version). */
    CustomPlanInput: {
      /**
       * Display name
       * @description Human-readable name for the resource. Between 1 and 256 characters.
       */
      name: string
      /**
       * Description
       * @description Optional description of the resource. Maximum 1024 characters.
       */
      description?: string
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata'] | null
      /** @description Alignment configuration for the plan. */
      alignment?: components['schemas']['Alignment']
      /**
       * Currency
       * @description The currency code of the plan.
       * @default USD
       */
      currency: components['schemas']['CurrencyCode']
      /**
       * Billing cadence
       * Format: duration
       * @description The default billing cadence for subscriptions using this plan.
       *     Defines how often customers are billed using ISO8601 duration format.
       *     Examples: "P1M" (monthly), "P3M" (quarterly), "P1Y" (annually).
       * @example P1M
       */
      billingCadence: string
      /**
       * Pro-rating configuration
       * @description Default pro-rating configuration for subscriptions using this plan.
       * @default {
       *       "enabled": true,
       *       "mode": "prorate_prices"
       *     }
       */
      proRatingConfig?: components['schemas']['ProRatingConfig']
      /**
       * Plan phases
       * @description The plan phase or pricing ramp allows changing a plan's rate cards over time as a subscription progresses.
       *     A phase switch occurs only at the end of a billing period, ensuring that a single subscription invoice will not include charges from different phase prices.
       */
      phases: components['schemas']['PlanPhase'][]
    }
    /** @description Change a custom subscription. */
    CustomSubscriptionChange: {
      /** @description Timing configuration for the change, when the change should take effect.
       *     For changing a subscription, the accepted values depend on the subscription configuration. */
      timing: components['schemas']['SubscriptionTiming']
      /** @description The custom plan description which defines the Subscription. */
      customPlan: components['schemas']['CustomPlanInput']
    }
    /** @description Create a custom subscription. */
    CustomSubscriptionCreate: {
      /** @description The custom plan description which defines the Subscription. */
      customPlan: components['schemas']['CustomPlanInput']
      /**
       * @description Timing configuration for the change, when the change should take effect.
       *     The default is immediate.
       * @default immediate
       */
      timing?: components['schemas']['SubscriptionTiming']
      /**
       * @description The ID of the customer. Provide either the key or ID. Has presedence over the key.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      customerId?: string
      /** @description The key of the customer. Provide either the key or ID. */
      customerKey?: string
      /**
       * Format: date-time
       * @description The billing anchor of the subscription. The provided date will be normalized according to the billing cadence to the nearest recurrence before start time. If not provided, the subscription start time will be used.
       * @example 2023-01-01T01:01:01.001Z
       */
      billingAnchor?: Date
    }
    /**
     * @description A customer object.
     * @example {
     *       "id": "01G65Z755AFWAKHE12NY0CQ9FH",
     *       "name": "ACME Inc.",
     *       "usageAttribution": {
     *         "subjectKeys": [
     *           "my_subject_key"
     *         ]
     *       },
     *       "createdAt": "2024-01-01T01:01:01.001Z",
     *       "updatedAt": "2024-01-01T01:01:01.001Z"
     *     }
     */
    Customer: {
      /**
       * ID
       * @description A unique identifier for the resource.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      readonly id: string
      /**
       * Display name
       * @description Human-readable name for the resource. Between 1 and 256 characters.
       */
      name: string
      /**
       * Description
       * @description Optional description of the resource. Maximum 1024 characters.
       */
      description?: string
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata'] | null
      /**
       * Creation Time
       * Format: date-time
       * @description Timestamp of when the resource was created.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly createdAt: Date
      /**
       * Last Update Time
       * Format: date-time
       * @description Timestamp of when the resource was last updated.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly updatedAt: Date
      /**
       * Deletion Time
       * Format: date-time
       * @description Timestamp of when the resource was permanently deleted.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly deletedAt?: Date
      /**
       * Key
       * @description An optional unique key of the customer.
       *     Useful to reference the customer in external systems.
       *     For example, your database ID.
       */
      key?: string
      /**
       * Usage Attribution
       * @description Mapping to attribute metered usage to the customer
       */
      usageAttribution: components['schemas']['CustomerUsageAttribution']
      /**
       * Primary Email
       * @description The primary email address of the customer.
       */
      primaryEmail?: string
      /**
       * Currency
       * @description Currency of the customer.
       *     Used for billing, tax and invoicing.
       */
      currency?: components['schemas']['CurrencyCode']
      /**
       * Billing Address
       * @description The billing address of the customer.
       *     Used for tax and invoicing.
       */
      billingAddress?: components['schemas']['Address']
      /**
       * Current Subscription ID
       * @description The ID of the Subscription if the customer has one.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      readonly currentSubscriptionId?: string
      /**
       * Subscriptions
       * @description The subscriptions of the customer.
       *     Only with the `subscriptions` expand option.
       */
      readonly subscriptions?: components['schemas']['Subscription'][]
    }
    /** @description CustomerAccess describes what features the customer has access to. */
    CustomerAccess: {
      /** @description Map of entitlements the customer has access to.
       *     The key is the feature key, the value is the entitlement value + the entitlement ID. */
      readonly entitlements: {
        [key: string]: components['schemas']['EntitlementValue']
      }
    }
    /** @description CustomerAppData
     *     Stores the app specific data for the customer.
     *     One of: stripe, sandbox, custom_invoicing */
    CustomerAppData:
      | components['schemas']['StripeCustomerAppData']
      | components['schemas']['SandboxCustomerAppData']
      | components['schemas']['CustomInvoicingCustomerAppData']
    /** @description CustomerAppData
     *     Stores the app specific data for the customer.
     *     One of: stripe, sandbox, custom_invoicing */
    CustomerAppDataCreateOrUpdateItem:
      | components['schemas']['StripeCustomerAppDataCreateOrUpdateItem']
      | components['schemas']['SandboxCustomerAppData']
      | components['schemas']['CustomInvoicingCustomerAppData']
    /** @description Paginated response */
    CustomerAppDataPaginatedResponse: {
      /**
       * @description The total number of items.
       * @example 500
       */
      totalCount: number
      /**
       * @description The page index.
       * @example 1
       */
      page: number
      /**
       * @description The maximum number of items per page.
       * @example 100
       */
      pageSize: number
      /** @description The items in the current page. */
      items: components['schemas']['CustomerAppData'][]
    }
    /** @description Resource create operation model. */
    CustomerCreate: {
      /**
       * Display name
       * @description Human-readable name for the resource. Between 1 and 256 characters.
       */
      name: string
      /**
       * Description
       * @description Optional description of the resource. Maximum 1024 characters.
       */
      description?: string
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata'] | null
      /**
       * Key
       * @description An optional unique key of the customer.
       *     Useful to reference the customer in external systems.
       *     For example, your database ID.
       */
      key?: string
      /**
       * Usage Attribution
       * @description Mapping to attribute metered usage to the customer
       */
      usageAttribution: components['schemas']['CustomerUsageAttribution']
      /**
       * Primary Email
       * @description The primary email address of the customer.
       */
      primaryEmail?: string
      /**
       * Currency
       * @description Currency of the customer.
       *     Used for billing, tax and invoicing.
       */
      currency?: components['schemas']['CurrencyCode']
      /**
       * Billing Address
       * @description The billing address of the customer.
       *     Used for tax and invoicing.
       */
      billingAddress?: components['schemas']['Address']
    }
    /**
     * @description CustomerExpand specifies the parts of the customer to expand in the list output.
     * @enum {string}
     */
    CustomerExpand: 'subscriptions'
    /** @description Create Stripe checkout session with customer ID. */
    CustomerId: {
      /**
       * @description ULID (Universally Unique Lexicographically Sortable Identifier).
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      id: string
    }
    /** @description Create Stripe checkout session with customer key. */
    CustomerKey: {
      key: string
    }
    /**
     * @description Order by options for customers.
     * @enum {string}
     */
    CustomerOrderBy: 'id' | 'name' | 'createdAt'
    /** @description Paginated response */
    CustomerPaginatedResponse: {
      /**
       * @description The total number of items.
       * @example 500
       */
      totalCount: number
      /**
       * @description The page index.
       * @example 1
       */
      page: number
      /**
       * @description The maximum number of items per page.
       * @example 100
       */
      pageSize: number
      /** @description The items in the current page. */
      items: components['schemas']['Customer'][]
    }
    /** @description Resource update operation model. */
    CustomerReplaceUpdate: {
      /**
       * Display name
       * @description Human-readable name for the resource. Between 1 and 256 characters.
       */
      name: string
      /**
       * Description
       * @description Optional description of the resource. Maximum 1024 characters.
       */
      description?: string
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata'] | null
      /**
       * Key
       * @description An optional unique key of the customer.
       *     Useful to reference the customer in external systems.
       *     For example, your database ID.
       */
      key?: string
      /**
       * Usage Attribution
       * @description Mapping to attribute metered usage to the customer
       */
      usageAttribution: components['schemas']['CustomerUsageAttribution']
      /**
       * Primary Email
       * @description The primary email address of the customer.
       */
      primaryEmail?: string
      /**
       * Currency
       * @description Currency of the customer.
       *     Used for billing, tax and invoicing.
       */
      currency?: components['schemas']['CurrencyCode']
      /**
       * Billing Address
       * @description The billing address of the customer.
       *     Used for tax and invoicing.
       */
      billingAddress?: components['schemas']['Address']
    }
    /** @description Mapping to attribute metered usage to the customer.
     *     One customer can have multiple subjects,
     *     but one subject can only belong to one customer. */
    CustomerUsageAttribution: {
      /**
       * SubjectKeys
       * @description The subjects that are attributed to the customer.
       */
      subjectKeys: string[]
    }
    /** @description Percentage discount. */
    DiscountPercentage: {
      /**
       * Percentage
       * @description The percentage of the discount.
       */
      percentage: components['schemas']['Percentage']
    }
    /** @description The reason for the discount is a maximum spend. */
    DiscountReasonMaximumSpend: {
      /**
       * @description discriminator enum property added by openapi-typescript
       * @enum {string}
       */
      type: 'maximum_spend'
    }
    /** @description The reason for the discount is a ratecard percentage. */
    DiscountReasonRatecardPercentage: {
      /**
       * @description discriminator enum property added by openapi-typescript
       * @enum {string}
       */
      type: 'ratecard_percentage'
      /**
       * Percentage
       * @description The percentage of the discount.
       */
      percentage: components['schemas']['Percentage']
      /**
       * @description Correlation ID for the discount.
       *
       *     This is used to link discounts across different invoices (progressive billing use case).
       *
       *     If not provided, the invoicing engine will auto-generate one. When editing an invoice line,
       *     please make sure to keep the same correlation ID of the discount or in progressive billing
       *     setups the discount amounts might be incorrect.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      correlationId?: string
    }
    /** @description The reason for the discount is a ratecard usage. */
    DiscountReasonRatecardUsage: {
      /**
       * @description discriminator enum property added by openapi-typescript
       * @enum {string}
       */
      type: 'ratecard_usage'
      /**
       * Usage
       * @description The quantity of the usage discount.
       *
       *     Must be positive.
       */
      quantity: components['schemas']['Numeric']
      /**
       * @description Correlation ID for the discount.
       *
       *     This is used to link discounts across different invoices (progressive billing use case).
       *
       *     If not provided, the invoicing engine will auto-generate one. When editing an invoice line,
       *     please make sure to keep the same correlation ID of the discount or in progressive billing
       *     setups the discount amounts might be incorrect.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      correlationId?: string
    }
    /** @description Usage discount.
     *
     *     Usage discount means that the first N items are free. From billing perspective
     *     this means that any usage on a specific feature is considered 0 until this discount
     *     is exhausted. */
    DiscountUsage: {
      /**
       * Usage
       * @description The quantity of the usage discount.
       *
       *     Must be positive.
       */
      quantity: components['schemas']['Numeric']
    }
    /** @description Discount by type on a price */
    Discounts: {
      /** @description The percentage discount. */
      percentage?: components['schemas']['DiscountPercentage']
      /** @description The usage discount. */
      usage?: components['schemas']['DiscountUsage']
    }
    /** @description Dynamic price with spend commitments. */
    DynamicPriceWithCommitments: {
      /**
       * @description The type of the price. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'dynamic'
      /**
       * The multiplier to apply to the base price to get the dynamic price
       * @description The multiplier to apply to the base price to get the dynamic price.
       *
       *     Examples:
       *     - 0.0: the price is zero
       *     - 0.5: the price is 50% of the base price
       *     - 1.0: the price is the same as the base price
       *     - 1.5: the price is 150% of the base price
       * @default 1
       */
      multiplier?: components['schemas']['Numeric']
      /**
       * Minimum amount
       * @description The customer is committed to spend at least the amount.
       */
      minimumAmount?: components['schemas']['Numeric']
      /**
       * Maximum amount
       * @description The customer is limited to spend at most the amount.
       */
      maximumAmount?: components['schemas']['Numeric']
    }
    /** @description Add a new item to a phase. */
    EditSubscriptionAddItem: {
      /**
       * @description discriminator enum property added by openapi-typescript
       * @enum {string}
       */
      op: 'add_item'
      phaseKey: string
      rateCard: components['schemas']['RateCard']
    }
    /** @description Add a new phase */
    EditSubscriptionAddPhase: {
      /**
       * @description discriminator enum property added by openapi-typescript
       * @enum {string}
       */
      op: 'add_phase'
      phase: components['schemas']['SubscriptionPhaseCreate']
    }
    /** @description Remove an item from a phase. */
    EditSubscriptionRemoveItem: {
      /**
       * @description discriminator enum property added by openapi-typescript
       * @enum {string}
       */
      op: 'remove_item'
      phaseKey: string
      itemKey: string
    }
    /** @description Remove a phase */
    EditSubscriptionRemovePhase: {
      /**
       * @description discriminator enum property added by openapi-typescript
       * @enum {string}
       */
      op: 'remove_phase'
      phaseKey: string
      shift: components['schemas']['RemovePhaseShifting']
    }
    /** @description Stretch a phase */
    EditSubscriptionStretchPhase: {
      /**
       * @description discriminator enum property added by openapi-typescript
       * @enum {string}
       */
      op: 'stretch_phase'
      phaseKey: string
      /** Format: duration */
      extendBy: string
    }
    /** @description Unschedules any edits from the current phase. */
    EditSubscriptionUnscheduleEdit: {
      /**
       * @description discriminator enum property added by openapi-typescript
       * @enum {string}
       */
      op: 'unschedule_edit'
    }
    /** @description Entitlement templates are used to define the entitlements of a plan.
     *     Features are omitted from the entitlement template, as they are defined in the rate card. */
    Entitlement:
      | components['schemas']['EntitlementMetered']
      | components['schemas']['EntitlementStatic']
      | components['schemas']['EntitlementBoolean']
    /** @description Shared fields of the entitlement templates. */
    EntitlementBaseTemplate: {
      /**
       * Creation Time
       * Format: date-time
       * @description Timestamp of when the resource was created.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly createdAt: Date
      /**
       * Last Update Time
       * Format: date-time
       * @description Timestamp of when the resource was last updated.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly updatedAt: Date
      /**
       * Deletion Time
       * Format: date-time
       * @description Timestamp of when the resource was permanently deleted.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly deletedAt?: Date
      /** @description Additional metadata for the feature. */
      metadata?: components['schemas']['Metadata']
      /**
       * Format: date-time
       * @description The cadence start of the resource.
       * @example 2023-01-01T01:01:01.001Z
       */
      activeFrom: Date
      /**
       * Format: date-time
       * @description The cadence end of the resource.
       * @example 2023-01-01T01:01:01.001Z
       */
      activeTo?: Date
      /**
       * @description The annotations of the entitlement.
       * @example {
       *       "subscription.id": "sub_123"
       *     }
       */
      readonly annotations?: components['schemas']['Annotations']
      /**
       * @description Readonly unique ULID identifier.
       * @example 01ARZ3NDEKTSV4RRFFQ69G5FAV
       */
      readonly id: string
      /**
       * Type
       * @description The type of the entitlement.
       */
      type: components['schemas']['EntitlementType']
      /**
       * @description The identifier key unique to the subject
       * @example customer-1
       */
      subjectKey: string
      /**
       * @description The feature the subject is entitled to use.
       * @example example-feature-key
       */
      featureKey: string
      /**
       * @description The feature the subject is entitled to use.
       * @example 01ARZ3NDEKTSV4RRFFQ69G5FAV
       */
      featureId: string
      /** @description The current usage period. */
      currentUsagePeriod?: components['schemas']['Period']
      /** @description The defined usage period of the entitlement */
      usagePeriod?: components['schemas']['RecurringPeriod']
    }
    /** @description Entitlement template of a boolean entitlement. */
    EntitlementBoolean: {
      /** @enum {string} */
      type: 'boolean'
      /**
       * Creation Time
       * Format: date-time
       * @description Timestamp of when the resource was created.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly createdAt: Date
      /**
       * Last Update Time
       * Format: date-time
       * @description Timestamp of when the resource was last updated.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly updatedAt: Date
      /**
       * Deletion Time
       * Format: date-time
       * @description Timestamp of when the resource was permanently deleted.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly deletedAt?: Date
      /** @description Additional metadata for the feature. */
      metadata?: components['schemas']['Metadata']
      /**
       * Format: date-time
       * @description The cadence start of the resource.
       * @example 2023-01-01T01:01:01.001Z
       */
      activeFrom: Date
      /**
       * Format: date-time
       * @description The cadence end of the resource.
       * @example 2023-01-01T01:01:01.001Z
       */
      activeTo?: Date
      /**
       * @description The annotations of the entitlement.
       * @example {
       *       "subscription.id": "sub_123"
       *     }
       */
      readonly annotations?: components['schemas']['Annotations']
      /**
       * @description Readonly unique ULID identifier.
       * @example 01ARZ3NDEKTSV4RRFFQ69G5FAV
       */
      readonly id: string
      /**
       * @description The identifier key unique to the subject
       * @example customer-1
       */
      subjectKey: string
      /**
       * @description The feature the subject is entitled to use.
       * @example example-feature-key
       */
      featureKey: string
      /**
       * @description The feature the subject is entitled to use.
       * @example 01ARZ3NDEKTSV4RRFFQ69G5FAV
       */
      featureId: string
      /** @description The current usage period. */
      currentUsagePeriod?: components['schemas']['Period']
      /** @description The defined usage period of the entitlement */
      usagePeriod?: components['schemas']['RecurringPeriod']
    } & (WithRequired<
      components['schemas']['EntitlementBaseTemplate'],
      | 'type'
      | 'createdAt'
      | 'updatedAt'
      | 'activeFrom'
      | 'id'
      | 'subjectKey'
      | 'featureKey'
      | 'featureId'
    > & {
      /**
       * @description discriminator enum property added by openapi-typescript
       * @enum {string}
       */
      type: 'boolean'
    })
    /** @description Create inputs for boolean entitlement */
    EntitlementBooleanCreateInputs: {
      /**
       * @description The feature the subject is entitled to use.
       *     Either featureKey or featureId is required.
       * @example example-feature-key
       */
      featureKey?: string
      /**
       * @description The feature the subject is entitled to use.
       *     Either featureKey or featureId is required.
       * @example 01ARZ3NDEKTSV4RRFFQ69G5FAV
       */
      featureId?: string
      /** @description Additional metadata for the feature. */
      metadata?: components['schemas']['Metadata']
      /** @description The usage period associated with the entitlement. */
      usagePeriod?: components['schemas']['RecurringPeriodCreateInput']
      /**
       * @description discriminator enum property added by openapi-typescript
       * @enum {string}
       */
      type: 'boolean'
    }
    /** @description Create inputs for entitlement */
    EntitlementCreateInputs:
      | components['schemas']['EntitlementMeteredCreateInputs']
      | components['schemas']['EntitlementStaticCreateInputs']
      | components['schemas']['EntitlementBooleanCreateInputs']
    /** @description The grant. */
    EntitlementGrant: {
      /**
       * Creation Time
       * Format: date-time
       * @description Timestamp of when the resource was created.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly createdAt: Date
      /**
       * Last Update Time
       * Format: date-time
       * @description Timestamp of when the resource was last updated.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly updatedAt: Date
      /**
       * Deletion Time
       * Format: date-time
       * @description Timestamp of when the resource was permanently deleted.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly deletedAt?: Date
      /**
       * Format: double
       * @description The amount to grant. Should be a positive number.
       * @example 100
       */
      amount: number
      /**
       * Format: uint8
       * @description The priority of the grant. Grants with higher priority are applied first.
       *     Priority is a positive decimal numbers. With lower numbers indicating higher importance.
       *     For example, a priority of 1 is more urgent than a priority of 2.
       *     When there are several grants available for the same subject, the system selects the grant with the highest priority.
       *     In cases where grants share the same priority level, the grant closest to its expiration will be used first.
       *     In the case of two grants have identical priorities and expiration dates, the system will use the grant that was created first.
       * @example 1
       */
      priority?: number
      /**
       * Format: date-time
       * @description Effective date for grants and anchor for recurring grants. Provided value will be ceiled to metering windowSize (minute).
       * @example 2023-01-01T01:01:01.001Z
       */
      effectiveAt: Date
      /** @description The grant expiration definition */
      expiration: components['schemas']['ExpirationPeriod']
      /**
       * Format: double
       * @description Grants are rolled over at reset, after which they can have a different balance compared to what they had before the reset.
       *     Balance after the reset is calculated as: Balance_After_Reset = MIN(MaxRolloverAmount, MAX(Balance_Before_Reset, MinRolloverAmount))
       * @default 0
       * @example 100
       */
      maxRolloverAmount?: number
      /**
       * Format: double
       * @description Grants are rolled over at reset, after which they can have a different balance compared to what they had before the reset.
       *     Balance after the reset is calculated as: Balance_After_Reset = MIN(MaxRolloverAmount, MAX(Balance_Before_Reset, MinRolloverAmount))
       * @default 0
       * @example 100
       */
      minRolloverAmount?: number
      /**
       * @description The grant metadata.
       * @example {
       *       "stripePaymentId": "pi_4OrAkhLvyihio9p51h9iiFnB"
       *     }
       */
      metadata?: components['schemas']['Metadata']
      /**
       * @description Readonly unique ULID identifier.
       * @example 01ARZ3NDEKTSV4RRFFQ69G5FAV
       */
      readonly id: string
      /**
       * @description The unique entitlement ULID that the grant is associated with.
       * @example 01ARZ3NDEKTSV4RRFFQ69G5FAV
       */
      readonly entitlementId: string
      /**
       * Format: date-time
       * @description The next time the grant will recurr.
       * @example 2023-01-01T01:01:01.001Z
       */
      nextRecurrence?: Date
      /**
       * Format: date-time
       * @description The time the grant expires.
       * @example 2023-01-01T01:01:01.001Z
       */
      readonly expiresAt?: Date
      /**
       * Format: date-time
       * @description The time the grant was voided.
       * @example 2023-01-01T01:01:01.001Z
       */
      voidedAt?: Date
      /** @description The recurrence period of the grant. */
      recurrence?: components['schemas']['RecurringPeriod']
    }
    /** @description The grant creation input. */
    EntitlementGrantCreateInput: {
      /**
       * Format: double
       * @description The amount to grant. Should be a positive number.
       * @example 100
       */
      amount: number
      /**
       * Format: uint8
       * @description The priority of the grant. Grants with higher priority are applied first.
       *     Priority is a positive decimal numbers. With lower numbers indicating higher importance.
       *     For example, a priority of 1 is more urgent than a priority of 2.
       *     When there are several grants available for the same subject, the system selects the grant with the highest priority.
       *     In cases where grants share the same priority level, the grant closest to its expiration will be used first.
       *     In the case of two grants have identical priorities and expiration dates, the system will use the grant that was created first.
       * @example 1
       */
      priority?: number
      /**
       * Format: date-time
       * @description Effective date for grants and anchor for recurring grants. Provided value will be ceiled to metering windowSize (minute).
       * @example 2023-01-01T01:01:01.001Z
       */
      effectiveAt: Date
      /** @description The grant expiration definition */
      expiration: components['schemas']['ExpirationPeriod']
      /**
       * Format: double
       * @description Grants are rolled over at reset, after which they can have a different balance compared to what they had before the reset.
       *     Balance after the reset is calculated as: Balance_After_Reset = MIN(MaxRolloverAmount, MAX(Balance_Before_Reset, MinRolloverAmount))
       * @default 0
       * @example 100
       */
      maxRolloverAmount?: number
      /**
       * Format: double
       * @description Grants are rolled over at reset, after which they can have a different balance compared to what they had before the reset.
       *     Balance after the reset is calculated as: Balance_After_Reset = MIN(MaxRolloverAmount, MAX(Balance_Before_Reset, MinRolloverAmount))
       * @default 0
       * @example 100
       */
      minRolloverAmount?: number
      /**
       * @description The grant metadata.
       * @example {
       *       "stripePaymentId": "pi_4OrAkhLvyihio9p51h9iiFnB"
       *     }
       */
      metadata?: components['schemas']['Metadata']
      /** @description The subject of the grant. */
      recurrence?: components['schemas']['RecurringPeriodCreateInput']
    }
    /** @description Metered entitlements are useful for many different use cases, from setting up usage based access to implementing complex credit systems.
     *     Access is determined based on feature usage using a balance calculation (the "usage allowance" provided by the issued grants is "burnt down" by the usage). */
    EntitlementMetered: {
      /**
       * @description discriminator enum property added by openapi-typescript
       * @enum {string}
       */
      type: 'metered'
      /**
       * Soft limit
       * @description If softLimit=true the subject can use the feature even if the entitlement is exhausted, hasAccess will always be true.
       * @default false
       */
      isSoftLimit?: boolean
      /**
       * @deprecated
       * @description Deprecated, ignored by the backend. Please use isSoftLimit instead; this field will be removed in the future.
       * @default false
       */
      isUnlimited?: boolean
      /**
       * Initial grant amount
       * Format: double
       * @description You can grant usage automatically alongside the entitlement, the example scenario would be creating a starting balance.
       *     If an amount is specified here, a grant will be created alongside the entitlement with the specified amount.
       *     That grant will have it's rollover settings configured in a way that after each reset operation, the balance will return the original amount specified here.
       *     Manually creating such a grant would mean having the "amount", "minRolloverAmount", and "maxRolloverAmount" fields all be the same.
       */
      issueAfterReset?: number
      /**
       * Issue grant after reset priority
       * Format: uint8
       * @description Defines the grant priority for the default grant.
       * @default 1
       */
      issueAfterResetPriority?: number
      /**
       * Preserve overage at reset
       * @description If true, the overage is preserved at reset. If false, the usage is reset to 0.
       * @default false
       */
      preserveOverageAtReset?: boolean
      /**
       * Creation Time
       * Format: date-time
       * @description Timestamp of when the resource was created.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly createdAt: Date
      /**
       * Last Update Time
       * Format: date-time
       * @description Timestamp of when the resource was last updated.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly updatedAt: Date
      /**
       * Deletion Time
       * Format: date-time
       * @description Timestamp of when the resource was permanently deleted.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly deletedAt?: Date
      /** @description Additional metadata for the feature. */
      metadata?: components['schemas']['Metadata']
      /**
       * Format: date-time
       * @description The cadence start of the resource.
       * @example 2023-01-01T01:01:01.001Z
       */
      activeFrom: Date
      /**
       * Format: date-time
       * @description The cadence end of the resource.
       * @example 2023-01-01T01:01:01.001Z
       */
      activeTo?: Date
      /**
       * @description The annotations of the entitlement.
       * @example {
       *       "subscription.id": "sub_123"
       *     }
       */
      readonly annotations?: components['schemas']['Annotations']
      /**
       * @description Readonly unique ULID identifier.
       * @example 01ARZ3NDEKTSV4RRFFQ69G5FAV
       */
      readonly id: string
      /**
       * @description The identifier key unique to the subject
       * @example customer-1
       */
      subjectKey: string
      /**
       * @description The feature the subject is entitled to use.
       * @example example-feature-key
       */
      featureKey: string
      /**
       * @description The feature the subject is entitled to use.
       * @example 01ARZ3NDEKTSV4RRFFQ69G5FAV
       */
      featureId: string
      /**
       * Format: date-time
       * @description The time the last reset happened.
       * @example 2023-01-01T01:01:01.001Z
       */
      readonly lastReset: Date
      /** @description The current usage period. */
      readonly currentUsagePeriod: components['schemas']['Period']
      /**
       * Format: date-time
       * @description The time from which usage is measured. If not specified on creation, defaults to entitlement creation time.
       * @example 2023-01-01T01:01:01.001Z
       */
      readonly measureUsageFrom: Date
      /** @description THe usage period of the entitlement. */
      readonly usagePeriod: components['schemas']['RecurringPeriod']
    }
    /** @description Create inpurs for metered entitlement */
    EntitlementMeteredCreateInputs: {
      /**
       * @description The feature the subject is entitled to use.
       *     Either featureKey or featureId is required.
       * @example example-feature-key
       */
      featureKey?: string
      /**
       * @description The feature the subject is entitled to use.
       *     Either featureKey or featureId is required.
       * @example 01ARZ3NDEKTSV4RRFFQ69G5FAV
       */
      featureId?: string
      /** @description Additional metadata for the feature. */
      metadata?: components['schemas']['Metadata']
      /**
       * @description discriminator enum property added by openapi-typescript
       * @enum {string}
       */
      type: 'metered'
      /**
       * Soft limit
       * @description If softLimit=true the subject can use the feature even if the entitlement is exhausted, hasAccess will always be true.
       * @default false
       */
      isSoftLimit?: boolean
      /**
       * @deprecated
       * @description Deprecated, ignored by the backend. Please use isSoftLimit instead; this field will be removed in the future.
       * @default false
       */
      isUnlimited?: boolean
      /** @description The usage period associated with the entitlement. */
      usagePeriod: components['schemas']['RecurringPeriodCreateInput']
      /** @description Defines the time from which usage is measured. If not specified on creation, defaults to entitlement creation time. */
      measureUsageFrom?: components['schemas']['MeasureUsageFrom']
      /**
       * Initial grant amount
       * Format: double
       * @description You can grant usage automatically alongside the entitlement, the example scenario would be creating a starting balance.
       *     If an amount is specified here, a grant will be created alongside the entitlement with the specified amount.
       *     That grant will have it's rollover settings configured in a way that after each reset operation, the balance will return the original amount specified here.
       *     Manually creating such a grant would mean having the "amount", "minRolloverAmount", and "maxRolloverAmount" fields all be the same.
       */
      issueAfterReset?: number
      /**
       * Issue grant after reset priority
       * Format: uint8
       * @description Defines the grant priority for the default grant.
       * @default 1
       */
      issueAfterResetPriority?: number
      /**
       * Preserve overage at reset
       * @description If true, the overage is preserved at reset. If false, the usage is reset to 0.
       * @default false
       */
      preserveOverageAtReset?: boolean
    }
    /**
     * @description Order by options for entitlements.
     * @enum {string}
     */
    EntitlementOrderBy: 'createdAt' | 'updatedAt'
    /** @description Paginated response */
    EntitlementPaginatedResponse: {
      /**
       * @description The total number of items.
       * @example 500
       */
      totalCount: number
      /**
       * @description The page index.
       * @example 1
       */
      page: number
      /**
       * @description The maximum number of items per page.
       * @example 100
       */
      pageSize: number
      /** @description The items in the current page. */
      items: components['schemas']['Entitlement'][]
    }
    /** @description A static entitlement. */
    EntitlementStatic: {
      /**
       * @description discriminator enum property added by openapi-typescript
       * @enum {string}
       */
      type: 'static'
      /**
       * Format: json
       * @description The JSON parsable config of the entitlement. This value is also returned when checking entitlement access and it is useful for configuring fine-grained access settings to the feature, implemented in your own system. Has to be an object.
       * @example { "integrations": ["github"] }
       */
      config: string
      /**
       * Creation Time
       * Format: date-time
       * @description Timestamp of when the resource was created.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly createdAt: Date
      /**
       * Last Update Time
       * Format: date-time
       * @description Timestamp of when the resource was last updated.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly updatedAt: Date
      /**
       * Deletion Time
       * Format: date-time
       * @description Timestamp of when the resource was permanently deleted.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly deletedAt?: Date
      /** @description Additional metadata for the feature. */
      metadata?: components['schemas']['Metadata']
      /**
       * Format: date-time
       * @description The cadence start of the resource.
       * @example 2023-01-01T01:01:01.001Z
       */
      activeFrom: Date
      /**
       * Format: date-time
       * @description The cadence end of the resource.
       * @example 2023-01-01T01:01:01.001Z
       */
      activeTo?: Date
      /**
       * @description The annotations of the entitlement.
       * @example {
       *       "subscription.id": "sub_123"
       *     }
       */
      readonly annotations?: components['schemas']['Annotations']
      /**
       * @description Readonly unique ULID identifier.
       * @example 01ARZ3NDEKTSV4RRFFQ69G5FAV
       */
      readonly id: string
      /**
       * @description The identifier key unique to the subject
       * @example customer-1
       */
      subjectKey: string
      /**
       * @description The feature the subject is entitled to use.
       * @example example-feature-key
       */
      featureKey: string
      /**
       * @description The feature the subject is entitled to use.
       * @example 01ARZ3NDEKTSV4RRFFQ69G5FAV
       */
      featureId: string
      /** @description The current usage period. */
      currentUsagePeriod?: components['schemas']['Period']
      /** @description The defined usage period of the entitlement */
      usagePeriod?: components['schemas']['RecurringPeriod']
    }
    /** @description Create inputs for static entitlement */
    EntitlementStaticCreateInputs: {
      /**
       * @description The feature the subject is entitled to use.
       *     Either featureKey or featureId is required.
       * @example example-feature-key
       */
      featureKey?: string
      /**
       * @description The feature the subject is entitled to use.
       *     Either featureKey or featureId is required.
       * @example 01ARZ3NDEKTSV4RRFFQ69G5FAV
       */
      featureId?: string
      /** @description Additional metadata for the feature. */
      metadata?: components['schemas']['Metadata']
      /** @description The usage period associated with the entitlement. */
      usagePeriod?: components['schemas']['RecurringPeriodCreateInput']
      /**
       * @description discriminator enum property added by openapi-typescript
       * @enum {string}
       */
      type: 'static'
      /**
       * Format: json
       * @description The JSON parsable config of the entitlement. This value is also returned when checking entitlement access and it is useful for configuring fine-grained access settings to the feature, implemented in your own system. Has to be an object.
       * @example { "integrations": ["github"] }
       */
      config: string
    }
    /**
     * @description Type of the entitlement.
     * @enum {string}
     */
    EntitlementType: 'metered' | 'boolean' | 'static'
    /** @description Entitlements are the core of OpenMeter access management. They define access to features for subjects. Entitlements can be metered, boolean, or static. */
    EntitlementValue: {
      /**
       * @description Whether the subject has access to the feature. Shared accross all entitlement types.
       * @example true
       */
      readonly hasAccess: boolean
      /**
       * Format: double
       * @description Only available for metered entitlements. Metered entitlements are built around a balance calculation where feature usage is deducted from the issued grants. Balance represents the remaining balance of the entitlement, it's value never turns negative.
       * @example 100
       */
      readonly balance?: number
      /**
       * Format: double
       * @description Only available for metered entitlements. Returns the total feature usage in the current period.
       * @example 50
       */
      readonly usage?: number
      /**
       * Format: double
       * @description Only available for metered entitlements. Overage represents the usage that wasn't covered by grants, e.g. if the subject had a total feature usage of 100 in the period but they were only granted 80, there would be 20 overage.
       * @example 0
       */
      readonly overage?: number
      /**
       * @description Only available for static entitlements. The JSON parsable config of the entitlement.
       * @example { key: "value" }
       */
      readonly config?: string
    }
    /**
     * @description CloudEvents Specification JSON Schema
     *
     *     Optional properties are nullable according to the CloudEvents specification:
     *     OPTIONAL not omitted attributes MAY be represented as a null JSON value.
     * @example {
     *       "id": "5c10fade-1c9e-4d6c-8275-c52c36731d3c",
     *       "source": "service-name",
     *       "specversion": "1.0",
     *       "type": "prompt",
     *       "subject": "customer-id",
     *       "time": "2023-01-01T01:01:01.001Z"
     *     }
     */
    Event: {
      /**
       * @description Identifies the event.
       * @example 5c10fade-1c9e-4d6c-8275-c52c36731d3c
       */
      id?: string
      /**
       * Format: uri-reference
       * @description Identifies the context in which an event happened.
       * @example service-name
       */
      source?: string
      /**
       * @description The version of the CloudEvents specification which the event uses.
       * @default 1.0
       * @example 1.0
       */
      specversion?: string
      /**
       * @description Contains a value describing the type of event related to the originating occurrence.
       * @example com.example.someevent
       */
      type: string
      /**
       * @description Content type of the CloudEvents data value. Only the value "application/json" is allowed over HTTP.
       * @example application/json
       * @enum {string|null}
       */
      datacontenttype?: 'application/json' | null
      /**
       * Format: uri
       * @description Identifies the schema that data adheres to.
       */
      dataschema?: string | null
      /**
       * @description Describes the subject of the event in the context of the event producer (identified by source).
       * @example customer-id
       */
      subject: string
      /**
       * Format: date-time
       * @description Timestamp of when the occurrence happened. Must adhere to RFC 3339.
       * @example 2023-01-01T01:01:01.001Z
       */
      time?: Date | null
      /** @description The event payload.
       *     Optional, if present it must be a JSON object. */
      data?: {
        [key: string]: unknown
      } | null
    }
    /**
     * @description The expiration duration enum
     * @enum {string}
     */
    ExpirationDuration: 'HOUR' | 'DAY' | 'WEEK' | 'MONTH' | 'YEAR'
    /** @description The grant expiration definition */
    ExpirationPeriod: {
      /** @description The unit of time for the expiration period. */
      duration: components['schemas']['ExpirationDuration']
      /**
       * @description The number of time units in the expiration period.
       * @example 12
       */
      count: number
    }
    /** @description Represents a feature that can be enabled or disabled for a plan.
     *     Used both for product catalog and entitlements. */
    Feature: {
      /**
       * Creation Time
       * Format: date-time
       * @description Timestamp of when the resource was created.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly createdAt: Date
      /**
       * Last Update Time
       * Format: date-time
       * @description Timestamp of when the resource was last updated.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly updatedAt: Date
      /**
       * Deletion Time
       * Format: date-time
       * @description Timestamp of when the resource was permanently deleted.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly deletedAt?: Date
      /**
       * Archival Time
       * Format: date-time
       * @description Timestamp of when the resource was archived.
       * @example 2023-01-01T01:01:01.001Z
       */
      readonly archivedAt?: Date
      /**
       * The unique key of the feature
       * @description A key is a unique string that is used to identify a resource.
       */
      key: string
      /** The human-readable name of the feature */
      name: string
      /**
       * Optional metadata
       * @example {
       *       "key": "value"
       *     }
       */
      metadata?: components['schemas']['Metadata']
      /**
       * Meter slug
       * @description A key is a unique string that is used to identify a resource.
       * @example tokens_total
       */
      meterSlug?: string
      /**
       * Meter group by filters
       * @description Optional meter group by filters.
       *     Useful if the meter scope is broader than what feature tracks.
       *     Example scenario would be a meter tracking all token use with groupBy fields for the model,
       *     then the feature could filter for model=gpt-4.
       * @example {
       *       "model": "gpt-4",
       *       "type": "input"
       *     }
       */
      meterGroupByFilters?: {
        [key: string]: string
      }
      /**
       * @description Readonly unique ULID identifier.
       * @example 01ARZ3NDEKTSV4RRFFQ69G5FAV
       */
      readonly id: string
    }
    /** @description Represents a feature that can be enabled or disabled for a plan.
     *     Used both for product catalog and entitlements. */
    FeatureCreateInputs: {
      /**
       * The unique key of the feature
       * @description A key is a unique string that is used to identify a resource.
       */
      key: string
      /** The human-readable name of the feature */
      name: string
      /**
       * Optional metadata
       * @example {
       *       "key": "value"
       *     }
       */
      metadata?: components['schemas']['Metadata']
      /**
       * Meter slug
       * @description A key is a unique string that is used to identify a resource.
       * @example tokens_total
       */
      meterSlug?: string
      /**
       * Meter group by filters
       * @description Optional meter group by filters.
       *     Useful if the meter scope is broader than what feature tracks.
       *     Example scenario would be a meter tracking all token use with groupBy fields for the model,
       *     then the feature could filter for model=gpt-4.
       * @example {
       *       "model": "gpt-4",
       *       "type": "input"
       *     }
       */
      meterGroupByFilters?: {
        [key: string]: string
      }
    }
    /** @description Limited representation of a feature resource which includes only its unique identifiers (id, key). */
    FeatureMeta: {
      /**
       * Feature Unique Identifier
       * @description Unique identifier of a feature.
       * @example 01ARZ3NDEKTSV4RRFFQ69G5FAV
       */
      id: string
      /**
       * Feature Key
       * @description The key is an immutable unique identifier of the feature used throughout the API,
       *     for example when interacting with a subject's entitlements.
       * @example gpt4_tokens
       */
      key: string
    }
    /**
     * @description Order by options for features.
     * @enum {string}
     */
    FeatureOrderBy: 'id' | 'key' | 'name' | 'createdAt' | 'updatedAt'
    /** @description Paginated response */
    FeaturePaginatedResponse: {
      /**
       * @description The total number of items.
       * @example 500
       */
      totalCount: number
      /**
       * @description The page index.
       * @example 1
       */
      page: number
      /**
       * @description The maximum number of items per page.
       * @example 100
       */
      pageSize: number
      /** @description The items in the current page. */
      items: components['schemas']['Feature'][]
    }
    /** @description A filter for a string field. */
    FilterString: {
      /** @description The field must be equal to the provided value. */
      $eq?: string | null
      /** @description The field must not be equal to the provided value. */
      $ne?: string | null
      /** @description The field must be in the provided list of values. */
      $in?: string[] | null
      /** @description The field must not be in the provided list of values. */
      $nin?: string[] | null
      /** @description The field must match the provided value. */
      $like?: string | null
      /** @description The field must not match the provided value. */
      $nlike?: string | null
      /** @description The field must match the provided value, ignoring case. */
      $ilike?: string | null
      /** @description The field must not match the provided value, ignoring case. */
      $nilike?: string | null
      /** @description The field must be greater than the provided value. */
      $gt?: string | null
      /** @description The field must be greater than or equal to the provided value. */
      $gte?: string | null
      /** @description The field must be less than the provided value. */
      $lt?: string | null
      /** @description The field must be less than or equal to the provided value. */
      $lte?: string | null
      /** @description Provide a list of filters to be combined with a logical AND. */
      $and?: components['schemas']['FilterString'][] | null
      /** @description Provide a list of filters to be combined with a logical OR. */
      $or?: components['schemas']['FilterString'][] | null
    }
    /** @description A filter for a time field. */
    FilterTime: {
      /**
       * Format: date-time
       * @description The field must be greater than the provided value.
       */
      $gt?: Date | null
      /**
       * Format: date-time
       * @description The field must be greater than or equal to the provided value.
       */
      $gte?: Date | null
      /**
       * Format: date-time
       * @description The field must be less than the provided value.
       */
      $lt?: Date | null
      /**
       * Format: date-time
       * @description The field must be less than or equal to the provided value.
       */
      $lte?: Date | null
      /** @description Provide a list of filters to be combined with a logical AND. */
      $and?: components['schemas']['FilterTime'][] | null
      /** @description Provide a list of filters to be combined with a logical OR. */
      $or?: components['schemas']['FilterTime'][] | null
    }
    /** @description Flat price. */
    FlatPrice: {
      /**
       * @description The type of the price.
       * @enum {string}
       */
      type: 'flat'
      /** @description The amount of the flat price. */
      amount: components['schemas']['Numeric']
    }
    /** @description Flat price with payment term. */
    FlatPriceWithPaymentTerm: {
      /**
       * @description The type of the price. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'flat'
      /** @description The amount of the flat price. */
      amount: components['schemas']['Numeric']
      /**
       * @description The payment term of the flat price.
       *     Defaults to in advance.
       * @default in_advance
       */
      paymentTerm?: components['schemas']['PricePaymentTerm']
    }
    /** @description The server understood the request but refuses to authorize it. */
    ForbiddenProblemResponse: components['schemas']['UnexpectedProblemResponse']
    /** @description The server, while acting as a gateway or proxy, did not receive a timely response from an upstream server it needed to access in order to complete the request. */
    GatewayTimeoutProblemResponse: components['schemas']['UnexpectedProblemResponse']
    /** @description A segment of the grant burn down history.
     *
     *     A given segment represents the usage of a grant between events that changed either the grant burn down priority order or the usag period. */
    GrantBurnDownHistorySegment: {
      /** @description The period of the segment. */
      period: components['schemas']['Period']
      /**
       * Format: double
       * @description The total usage of the grant in the period.
       * @example 100
       */
      readonly usage: number
      /**
       * Format: double
       * @description Overuse that wasn't covered by grants.
       * @example 100
       */
      readonly overage: number
      /**
       * Format: double
       * @description entitlement balance at the start of the period.
       * @example 100
       */
      readonly balanceAtStart: number
      /**
       * @description The balance breakdown of each active grant at the start of the period: GrantID: Balance
       * @example {
       *       "01G65Z755AFWAKHE12NY0CQ9FH": 100
       *     }
       */
      readonly grantBalancesAtStart: {
        [key: string]: number
      }
      /**
       * Format: double
       * @description The entitlement balance at the end of the period.
       * @example 100
       */
      readonly balanceAtEnd: number
      /**
       * @description The balance breakdown of each active grant at the end of the period: GrantID: Balance
       * @example {
       *       "01G65Z755AFWAKHE12NY0CQ9FH": 100
       *     }
       */
      readonly grantBalancesAtEnd: {
        [key: string]: number
      }
      /** @description Which grants were actually burnt down in the period and by what amount. */
      readonly grantUsages: components['schemas']['GrantUsageRecord'][]
    }
    /**
     * @description Order by options for grants.
     * @enum {string}
     */
    GrantOrderBy: 'id' | 'createdAt' | 'updatedAt'
    /** @description Paginated response */
    GrantPaginatedResponse: {
      /**
       * @description The total number of items.
       * @example 500
       */
      totalCount: number
      /**
       * @description The page index.
       * @example 1
       */
      page: number
      /**
       * @description The maximum number of items per page.
       * @example 100
       */
      pageSize: number
      /** @description The items in the current page. */
      items: components['schemas']['EntitlementGrant'][]
    }
    /** @description Usage Record */
    GrantUsageRecord: {
      /**
       * @description The id of the grant
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      grantId: string
      /**
       * Format: double
       * @description The usage in the period
       * @example 100
       */
      usage: number
    }
    /** @description IDResource is a resouce with an ID. */
    IDResource: {
      /**
       * ID
       * @description A unique identifier for the resource.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      readonly id: string
    }
    /** @description The body of the events request.
     *     Either a single event or a batch of events. */
    IngestEventsBody:
      | components['schemas']['Event']
      | components['schemas']['Event'][]
    /**
     * @description An ingested event with optional validation error.
     * @example {
     *       "event": {
     *         "id": "5c10fade-1c9e-4d6c-8275-c52c36731d3c",
     *         "source": "service-name",
     *         "specversion": "1.0",
     *         "type": "prompt",
     *         "subject": "customer-id",
     *         "time": "2023-01-01T01:01:01.001Z"
     *       },
     *       "ingestedAt": "2023-01-01T01:01:01.001Z",
     *       "storedAt": "2023-01-01T01:01:02.001Z"
     *     }
     */
    IngestedEvent: {
      /** @description The original event ingested. */
      event: components['schemas']['Event']
      /** @description The validation error if the event failed validation. */
      validationError?: string
      /**
       * Format: date-time
       * @description The date and time the event was ingested.
       * @example 2023-01-01T01:01:01.001Z
       */
      ingestedAt: Date
      /**
       * Format: date-time
       * @description The date and time the event was stored.
       * @example 2023-01-01T01:01:01.001Z
       */
      storedAt: Date
    }
    /** @description A response for cursor pagination. */
    IngestedEventCursorPaginatedResponse: {
      /** @description The items in the response. */
      items: components['schemas']['IngestedEvent'][]
      /** @description The cursor of the last item in the list. */
      nextCursor?: string
    }
    /**
     * @description Install method of the application.
     * @enum {string}
     */
    InstallMethod: 'with_oauth2' | 'with_api_key' | 'no_credentials_required'
    /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
    InternalServerErrorProblemResponse: components['schemas']['UnexpectedProblemResponse']
    /** @description Invoice represents an invoice in the system. */
    Invoice: {
      /**
       * ID
       * @description A unique identifier for the resource.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      readonly id: string
      /**
       * Description
       * @description Optional description of the resource. Maximum 1024 characters.
       */
      description?: string
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata'] | null
      /**
       * Creation Time
       * Format: date-time
       * @description Timestamp of when the resource was created.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly createdAt: Date
      /**
       * Last Update Time
       * Format: date-time
       * @description Timestamp of when the resource was last updated.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly updatedAt: Date
      /**
       * Deletion Time
       * Format: date-time
       * @description Timestamp of when the resource was permanently deleted.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly deletedAt?: Date
      /** @description Type of the invoice.
       *
       *     The type of invoice determines the purpose of the invoice and how it should be handled.
       *
       *     Supported types:
       *     - standard: A regular commercial invoice document between a supplier and customer.
       *     - credit_note: Reflects a refund either partial or complete of the preceding document. A credit note effectively *extends* the previous document. */
      readonly type: components['schemas']['InvoiceType']
      /** @description The taxable entity supplying the goods or services. */
      supplier: components['schemas']['BillingParty']
      /** @description Legal entity receiving the goods or services. */
      customer: components['schemas']['BillingParty']
      /** @description Number specifies the human readable key used to reference this Invoice.
       *
       *     The invoice number can change in the draft phases, as we are allocating temporary draft
       *     invoice numbers, but it's final as soon as the invoice gets finalized (issued state).
       *
       *     Please note that the number is (depending on the upstream settings) either unique for the
       *     whole organization or unique for the customer, or in multi (stripe) account setups unique for the
       *     account. */
      readonly number: components['schemas']['InvoiceNumber']
      /** @description Currency for all invoice line items.
       *
       *     Multi currency invoices are not supported yet. */
      currency: components['schemas']['CurrencyCode']
      /** @description Key information regarding previous invoices and potentially details as to why they were corrected. */
      readonly preceding?: components['schemas']['InvoiceDocumentRef'][]
      /** @description Summary of all the invoice totals, including taxes (calculated). */
      readonly totals: components['schemas']['InvoiceTotals']
      /** @description The status of the invoice.
       *
       *     This field only conatins a simplified status, for more detailed information use the statusDetails field. */
      readonly status: components['schemas']['InvoiceStatus']
      /** @description The details of the current invoice status. */
      readonly statusDetails: components['schemas']['InvoiceStatusDetails']
      /**
       * Format: date-time
       * @description The time the invoice was issued.
       *
       *     Depending on the status of the invoice this can mean multiple things:
       *     - draft, gathering: The time the invoice will be issued based on the workflow settings.
       *     - issued: The time the invoice was issued.
       * @example 2023-01-01T01:01:01.001Z
       */
      readonly issuedAt?: Date
      /**
       * Format: date-time
       * @description The time until the invoice is in draft status.
       *
       *     On draft invoice creation it is calculated from the workflow settings.
       *
       *     If manual approval is required, the draftUntil time is set.
       * @example 2023-01-01T01:01:01.001Z
       */
      draftUntil?: Date
      /**
       * Format: date-time
       * @description The time when the quantity snapshots on the invoice lines were taken.
       * @example 2023-01-01T01:01:01.001Z
       */
      readonly quantitySnapshotedAt?: Date
      /**
       * Format: date-time
       * @description The time when the invoice will be/has been collected.
       * @example 2023-01-01T01:01:01.001Z
       */
      readonly collectionAt?: Date
      /**
       * Format: date-time
       * @description Due time of the fulfillment of the invoice (if available).
       * @example 2023-01-01T01:01:01.001Z
       */
      readonly dueAt?: Date
      /** @description The period the invoice covers. If the invoice has no line items, it's not set. */
      period?: components['schemas']['Period']
      /**
       * Format: date-time
       * @description The time the invoice was voided.
       *
       *     If the invoice was voided, this field will be set to the time the invoice was voided.
       * @example 2023-01-01T01:01:01.001Z
       */
      readonly voidedAt?: Date
      /**
       * Format: date-time
       * @description The time the invoice was sent to customer.
       * @example 2023-01-01T01:01:01.001Z
       */
      readonly sentToCustomerAt?: Date
      /** @description The workflow associated with the invoice.
       *
       *     It is always a snapshot of the workflow settings at the time of invoice creation. The
       *     field is optional as it should be explicitly requested with expand options. */
      workflow: components['schemas']['InvoiceWorkflowSettings']
      /** @description List of invoice lines representing each of the items sold to the customer. */
      lines?: components['schemas']['InvoiceLine'][]
      /** @description Information on when, how, and to whom the invoice should be paid. */
      readonly payment?: components['schemas']['InvoicePaymentTerms']
      /** @description Validation issues reported by the invoice workflow. */
      readonly validationIssues?: components['schemas']['ValidationIssue'][]
      /** @description External IDs of the invoice in other apps such as Stripe. */
      readonly externalIds?: components['schemas']['InvoiceAppExternalIds']
    }
    /** @description InvoiceAppExternalIds contains the external IDs of the invoice in other apps such as Stripe. */
    InvoiceAppExternalIds: {
      /** @description The external ID of the invoice in the invoicing app if available. */
      readonly invoicing?: string
      /** @description The external ID of the invoice in the tax app if available. */
      readonly tax?: string
      /** @description The external ID of the invoice in the payment app if available. */
      readonly payment?: string
    }
    /** @description InvoiceAvailableActionInvoiceDetails represents the details of the invoice action for
     *     non-gathering invoices. */
    InvoiceAvailableActionDetails: {
      /** @description The state the invoice will reach if the action is activated and
       *     all intermediate steps are successful.
       *
       *     For example advancing a draft_created invoice will result in a draft_manual_approval_needed invoice. */
      readonly resultingState: string
    }
    /** @description InvoiceAvailableActionInvoiceDetails represents the details of the invoice action for
     *     gathering invoices. */
    InvoiceAvailableActionInvoiceDetails: Record<string, never>
    /** @description InvoiceAvailableActions represents the actions that can be performed on the invoice. */
    InvoiceAvailableActions: {
      /** @description Advance the invoice to the next status. */
      readonly advance?: components['schemas']['InvoiceAvailableActionDetails']
      /** @description Approve an invoice that requires manual approval. */
      readonly approve?: components['schemas']['InvoiceAvailableActionDetails']
      /** @description Delete the invoice (only non-issued invoices can be deleted). */
      readonly delete?: components['schemas']['InvoiceAvailableActionDetails']
      /** @description Retry an invoice issuing step that failed. */
      readonly retry?: components['schemas']['InvoiceAvailableActionDetails']
      /** @description Snapshot quantities for usage based line items. */
      readonly snapshotQuantities?: components['schemas']['InvoiceAvailableActionDetails']
      /** @description Void an already issued invoice. */
      readonly void?: components['schemas']['InvoiceAvailableActionDetails']
      /** @description Invoice a gathering invoice */
      readonly invoice?: components['schemas']['InvoiceAvailableActionInvoiceDetails']
    }
    /** @description InvoiceDetailedLine represents a line item that is sold to the customer as a manually added fee. */
    InvoiceDetailedLine: {
      /**
       * Display name
       * @description Human-readable name for the resource. Between 1 and 256 characters.
       */
      name: string
      /**
       * Description
       * @description Optional description of the resource. Maximum 1024 characters.
       */
      description?: string
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata'] | null
      /**
       * Creation Time
       * Format: date-time
       * @description Timestamp of when the resource was created.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly createdAt: Date
      /**
       * Last Update Time
       * Format: date-time
       * @description Timestamp of when the resource was last updated.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly updatedAt: Date
      /**
       * Deletion Time
       * Format: date-time
       * @description Timestamp of when the resource was permanently deleted.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly deletedAt?: Date
      /**
       * @description ID of the line.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      id: string
      /** @description managedBy specifies if the line is manually added via the api or managed by OpenMeter. */
      readonly managedBy: components['schemas']['InvoiceLineManagedBy']
      /** @description Status of the line.
       *
       *     External calls always create valid lines, other line types are managed by the
       *     billing engine of OpenMeter. */
      readonly status: components['schemas']['InvoiceLineStatus']
      /** @description Discounts detailes applied to this line.
       *
       *     New discounts can be added via the invoice's discounts API, to facilitate
       *     discounts that are affecting multiple lines. */
      readonly discounts?: components['schemas']['InvoiceLineDiscounts']
      /** @description The invoice this item belongs to. */
      invoice?: components['schemas']['InvoiceReference']
      /** @description The currency of this line. */
      currency: components['schemas']['CurrencyCode']
      /** @description Taxes applied to the invoice totals. */
      readonly taxes?: components['schemas']['InvoiceLineTaxItem'][]
      /**
       * @deprecated
       * @description Tax config specify the tax configuration for this line.
       */
      taxConfig?: components['schemas']['TaxConfig']
      /** @description Totals for this line. */
      readonly totals: components['schemas']['InvoiceTotals']
      /** @description Period of the line item applies to for revenue recognition pruposes.
       *
       *     Billing always treats periods as start being inclusive and end being exclusive. */
      period: components['schemas']['Period']
      /**
       * Format: date-time
       * @description The time this line item should be invoiced.
       * @example 2023-01-01T01:01:01.001Z
       */
      invoiceAt: Date
      /** @description External IDs of the invoice in other apps such as Stripe. */
      readonly externalIds?: components['schemas']['InvoiceLineAppExternalIds']
      /** @description Subscription are the references to the subscritpions that this line is related to. */
      readonly subscription?: components['schemas']['InvoiceLineSubscriptionReference']
      /**
       * @deprecated
       * @description Type of the line.
       * @enum {string}
       */
      readonly type: 'flat_fee'
      /**
       * @deprecated
       * @description Price of the item being sold.
       */
      perUnitAmount?: components['schemas']['Numeric']
      /**
       * @deprecated
       * @description Payment term of the line.
       * @default in_advance
       */
      paymentTerm?: components['schemas']['PricePaymentTerm']
      /**
       * @deprecated
       * @description Quantity of the item being sold.
       */
      quantity?: components['schemas']['Numeric']
      /** @description The rate card that is used for this line. */
      rateCard?: components['schemas']['InvoiceDetailedLineRateCard']
      /**
       * @description Category of the flat fee.
       * @default regular
       */
      readonly category?: components['schemas']['InvoiceDetailedLineCostCategory']
    }
    /**
     * @description InvoiceDetailedLineCostCategory determines if the flat fee is a regular fee due to use due to a
     *     commitment.
     * @enum {string}
     */
    InvoiceDetailedLineCostCategory: 'regular' | 'commitment'
    /** @description InvoiceDetailedLineRateCard represents the rate card (intent) for a flat fee line. */
    InvoiceDetailedLineRateCard: {
      /**
       * Tax config
       * @description The tax config of the rate card.
       *     When undefined, the tax config of the feature or the default tax config of the plan is used.
       */
      taxConfig?: components['schemas']['TaxConfig']
      /**
       * Price
       * @description The price of the rate card.
       *     When null, the feature or service is free.
       * @example {
       *       "type": "flat",
       *       "amount": "100",
       *       "paymentTerm": "in_arrears"
       *     }
       */
      price: components['schemas']['FlatPriceWithPaymentTerm'] | null
      /** @description Quantity of the item being sold.
       *
       *     Default: 1 */
      quantity?: components['schemas']['Numeric']
      /** @description The discounts that are applied to the line. */
      discounts?: components['schemas']['BillingDiscounts']
    }
    /** @description InvoiceDocumentRef is used to describe a reference to an existing document (invoice). */
    InvoiceDocumentRef: components['schemas']['CreditNoteOriginalInvoiceRef']
    /**
     * @description InvoiceDocumentRefType defines the type of document that is being referenced.
     * @enum {string}
     */
    InvoiceDocumentRefType: 'credit_note_original_invoice'
    /**
     * @description InvoiceExpand specifies the parts of the invoice to expand in the list output.
     * @enum {string}
     */
    InvoiceExpand: 'lines' | 'preceding' | 'workflow.apps'
    /**
     * InvoiceGenericDocumentRef is used to describe an existing document or a specific part of it's contents.
     * @description Omitted fields:
     *     period: Tax period in which the referred document had an effect required by some tax regimes and formats.
     *     stamps: Seals of approval from other organisations that may need to be listed.
     *     ext: 	Extensions for additional codes that may be required.
     */
    InvoiceGenericDocumentRef: {
      /** @description Type of the document referenced. */
      readonly type: components['schemas']['InvoiceDocumentRefType']
      /** @description Human readable description on why this reference is here or needs to be used. */
      readonly reason?: string
      /** @description Additional details about the document. */
      readonly description?: string
    }
    /** @description InvoiceUsageBasedLine represents a line item that is sold to the customer based on usage. */
    InvoiceLine: {
      /**
       * Display name
       * @description Human-readable name for the resource. Between 1 and 256 characters.
       */
      name: string
      /**
       * Description
       * @description Optional description of the resource. Maximum 1024 characters.
       */
      description?: string
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata'] | null
      /**
       * Creation Time
       * Format: date-time
       * @description Timestamp of when the resource was created.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly createdAt: Date
      /**
       * Last Update Time
       * Format: date-time
       * @description Timestamp of when the resource was last updated.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly updatedAt: Date
      /**
       * Deletion Time
       * Format: date-time
       * @description Timestamp of when the resource was permanently deleted.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly deletedAt?: Date
      /**
       * @description ID of the line.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      id: string
      /** @description managedBy specifies if the line is manually added via the api or managed by OpenMeter. */
      readonly managedBy: components['schemas']['InvoiceLineManagedBy']
      /** @description Status of the line.
       *
       *     External calls always create valid lines, other line types are managed by the
       *     billing engine of OpenMeter. */
      readonly status: components['schemas']['InvoiceLineStatus']
      /** @description Discounts detailes applied to this line.
       *
       *     New discounts can be added via the invoice's discounts API, to facilitate
       *     discounts that are affecting multiple lines. */
      readonly discounts?: components['schemas']['InvoiceLineDiscounts']
      /** @description The invoice this item belongs to. */
      invoice?: components['schemas']['InvoiceReference']
      /** @description The currency of this line. */
      currency: components['schemas']['CurrencyCode']
      /** @description Taxes applied to the invoice totals. */
      readonly taxes?: components['schemas']['InvoiceLineTaxItem'][]
      /**
       * @deprecated
       * @description Tax config specify the tax configuration for this line.
       */
      taxConfig?: components['schemas']['TaxConfig']
      /** @description Totals for this line. */
      readonly totals: components['schemas']['InvoiceTotals']
      /** @description Period of the line item applies to for revenue recognition pruposes.
       *
       *     Billing always treats periods as start being inclusive and end being exclusive. */
      period: components['schemas']['Period']
      /**
       * Format: date-time
       * @description The time this line item should be invoiced.
       * @example 2023-01-01T01:01:01.001Z
       */
      invoiceAt: Date
      /** @description External IDs of the invoice in other apps such as Stripe. */
      readonly externalIds?: components['schemas']['InvoiceLineAppExternalIds']
      /** @description Subscription are the references to the subscritpions that this line is related to. */
      readonly subscription?: components['schemas']['InvoiceLineSubscriptionReference']
      /**
       * @deprecated
       * @description Type of the line.
       * @enum {string}
       */
      readonly type: 'usage_based'
      /**
       * @deprecated
       * @description Price of the usage-based item being sold.
       */
      price?: components['schemas']['RateCardUsageBasedPrice']
      /**
       * @deprecated
       * @description The feature that the usage is based on.
       */
      featureKey?: string
      /** @description The lines detailing the item or service sold. */
      readonly children?: components['schemas']['InvoiceDetailedLine'][]
      /** @description The rate card that is used for this line.
       *
       *     The rate card captures the intent of the price and discounts for the usage-based item. */
      rateCard?: components['schemas']['InvoiceUsageBasedRateCard']
      /** @description The quantity of the item being sold.
       *
       *     Any usage discounts applied previously are deducted from this quantity. */
      readonly quantity?: components['schemas']['Numeric']
      /** @description The quantity of the item that has been metered for the period before any discounts were applied. */
      readonly meteredQuantity?: components['schemas']['Numeric']
      /** @description The quantity of the item used before this line's period.
       *
       *     It is non-zero in case of progressive billing, when this shows how much of the usage was already billed.
       *
       *     Any usage discounts applied previously are deducted from this quantity. */
      readonly preLinePeriodQuantity?: components['schemas']['Numeric']
      /** @description The metered quantity of the item used in before this line's period without any discounts applied.
       *
       *     It is non-zero in case of progressive billing, when this shows how much of the usage was already billed. */
      readonly meteredPreLinePeriodQuantity?: components['schemas']['Numeric']
    }
    /** @description InvoiceLineAmountDiscount represents an amount deducted from the line, and will be applied before taxes. */
    InvoiceLineAmountDiscount: {
      /**
       * Creation Time
       * Format: date-time
       * @description Timestamp of when the resource was created.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly createdAt: Date
      /**
       * Last Update Time
       * Format: date-time
       * @description Timestamp of when the resource was last updated.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly updatedAt: Date
      /**
       * Deletion Time
       * Format: date-time
       * @description Timestamp of when the resource was permanently deleted.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly deletedAt?: Date
      /**
       * @description ID of the charge or discount.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      readonly id: string
      /** @description Reason code. */
      readonly reason: components['schemas']['BillingDiscountReason']
      /** @description Text description as to why the discount was applied. */
      readonly description?: string
      /** @description External IDs of the invoice in other apps such as Stripe. */
      readonly externalIds?: components['schemas']['InvoiceLineAppExternalIds']
      /**
       * Amount in the currency of the invoice
       * @description Fixed discount amount to apply (calculated if percent present).
       */
      readonly amount: components['schemas']['Numeric']
    }
    /** @description InvoiceLineAppExternalIds contains the external IDs of the invoice in other apps such as Stripe. */
    InvoiceLineAppExternalIds: {
      /** @description The external ID of the invoice in the invoicing app if available. */
      readonly invoicing?: string
      /** @description The external ID of the invoice in the tax app if available. */
      readonly tax?: string
    }
    /** @description InvoiceLineDiscounts represents the discounts applied to the invoice line by type. */
    InvoiceLineDiscounts: {
      /** @description Amount based discounts applied to the line.
       *
       *     Amount based discounts are deduced from the total price of the line. */
      amount?: components['schemas']['InvoiceLineAmountDiscount'][]
      /** @description Usage based discounts applied to the line.
       *
       *     Usage based discounts are deduced from the usage of the line before price calculations are applied. */
      usage?: components['schemas']['InvoiceLineUsageDiscount'][]
    }
    /**
     * @description InvoiceLineManagedBy specifies who manages the line.
     * @enum {string}
     */
    InvoiceLineManagedBy: 'subscription' | 'system' | 'manual'
    /** @description InvoiceLineReplaceUpdate represents the update model for an UBP invoice line.
     *
     *     This type makes ID optional to allow for creating new lines as part of the update. */
    InvoiceLineReplaceUpdate: {
      /**
       * Display name
       * @description Human-readable name for the resource. Between 1 and 256 characters.
       */
      name: string
      /**
       * Description
       * @description Optional description of the resource. Maximum 1024 characters.
       */
      description?: string
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata'] | null
      /**
       * @deprecated
       * @description Tax config specify the tax configuration for this line.
       */
      taxConfig?: components['schemas']['TaxConfig']
      /** @description Period of the line item applies to for revenue recognition pruposes.
       *
       *     Billing always treats periods as start being inclusive and end being exclusive. */
      period: components['schemas']['Period']
      /**
       * Format: date-time
       * @description The time this line item should be invoiced.
       * @example 2023-01-01T01:01:01.001Z
       */
      invoiceAt: Date
      /**
       * @deprecated
       * @description Price of the usage-based item being sold.
       */
      price?: components['schemas']['RateCardUsageBasedPrice']
      /**
       * @deprecated
       * @description The feature that the usage is based on.
       */
      featureKey?: string
      /** @description The rate card that is used for this line.
       *
       *     The rate card captures the intent of the price and discounts for the usage-based item. */
      rateCard?: components['schemas']['InvoiceUsageBasedRateCard']
      /**
       * @description The ID of the line.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      id?: string
    }
    /**
     * @description Line status specifies the status of the line.
     * @enum {string}
     */
    InvoiceLineStatus: 'valid' | 'detail' | 'split'
    /** @description InvoiceLineSubscriptionReference contains the references to the subscription that this line is related to. */
    InvoiceLineSubscriptionReference: {
      /** @description The subscription. */
      readonly subscription: components['schemas']['IDResource']
      /** @description The phase of the subscription. */
      readonly phase: components['schemas']['IDResource']
      /** @description The item this line is related to. */
      readonly item: components['schemas']['IDResource']
    }
    /**
     * @description InvoiceLineTaxBehavior details how the tax item is applied to the base amount.
     *
     *     Inclusive means the tax is included in the base amount.
     *     Exclusive means the tax is added to the base amount.
     * @enum {string}
     */
    InvoiceLineTaxBehavior: 'inclusive' | 'exclusive'
    /** @description TaxConfig stores the configuration for a tax line relative to an invoice line. */
    InvoiceLineTaxItem: {
      /** @description Tax provider configuration. */
      readonly config?: components['schemas']['TaxConfig']
      /** @description Percent defines the percentage set manually or determined from
       *     the rate key (calculated if rate present). A nil percent implies that
       *     this tax combo is **exempt** from tax.") */
      readonly percent?: components['schemas']['Percentage']
      /** @description Some countries require an additional surcharge (calculated if rate present). */
      readonly surcharge?: components['schemas']['Numeric']
      /** @description Is the tax item inclusive or exclusive of the base amount. */
      readonly behavior?: components['schemas']['InvoiceLineTaxBehavior']
    }
    /** @description InvoiceLineUsageDiscount represents an usage-based discount applied to the line.
     *
     *     The deduction is done before the pricing algorithm is applied. */
    InvoiceLineUsageDiscount: {
      /**
       * Creation Time
       * Format: date-time
       * @description Timestamp of when the resource was created.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly createdAt: Date
      /**
       * Last Update Time
       * Format: date-time
       * @description Timestamp of when the resource was last updated.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly updatedAt: Date
      /**
       * Deletion Time
       * Format: date-time
       * @description Timestamp of when the resource was permanently deleted.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly deletedAt?: Date
      /**
       * @description ID of the charge or discount.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      readonly id: string
      /** @description Reason code. */
      readonly reason: components['schemas']['BillingDiscountReason']
      /** @description Text description as to why the discount was applied. */
      readonly description?: string
      /** @description External IDs of the invoice in other apps such as Stripe. */
      readonly externalIds?: components['schemas']['InvoiceLineAppExternalIds']
      /**
       * Usage quantity in the unit of the underlying meter
       * @description The usage to apply.
       */
      readonly quantity: components['schemas']['Numeric']
      /**
       * Usage quantity in the unit of the underlying meter
       * @description The usage discount already applied to the previous split lines.
       *
       *     Only set if progressive billing is enabled and the line is a split line.
       */
      readonly preLinePeriodQuantity?: components['schemas']['Numeric']
    }
    /**
     * @description InvoiceNumber is a unique identifier for the invoice, generated by the
     *     invoicing app.
     *
     *     The uniqueness depends on a lot of factors:
     *     - app setting (unique per app or unique per customer)
     *     - multiple app scenarios (multiple apps generating invoices with the same prefix)
     * @example INV-2024-01-01-01
     */
    InvoiceNumber: string
    /**
     * @description InvoiceOrderBy specifies the ordering options for invoice listing.
     * @enum {string}
     */
    InvoiceOrderBy:
      | 'customer.name'
      | 'issuedAt'
      | 'status'
      | 'createdAt'
      | 'updatedAt'
      | 'periodStart'
    /** @description Paginated response */
    InvoicePaginatedResponse: {
      /**
       * @description The total number of items.
       * @example 500
       */
      totalCount: number
      /**
       * @description The page index.
       * @example 1
       */
      page: number
      /**
       * @description The maximum number of items per page.
       * @example 100
       */
      pageSize: number
      /** @description The items in the current page. */
      items: components['schemas']['Invoice'][]
    }
    /** @description Payment contains details as to how the invoice should be paid. */
    InvoicePaymentTerms: {
      /** @description The terms of payment for the invoice. */
      terms?: components['schemas']['PaymentTerms']
    }
    /** @description InvoicePendingLineCreate represents the create model for an invoice line that is sold to the customer based on usage. */
    InvoicePendingLineCreate: {
      /**
       * Display name
       * @description Human-readable name for the resource. Between 1 and 256 characters.
       */
      name: string
      /**
       * Description
       * @description Optional description of the resource. Maximum 1024 characters.
       */
      description?: string
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata'] | null
      /**
       * @deprecated
       * @description Tax config specify the tax configuration for this line.
       */
      taxConfig?: components['schemas']['TaxConfig']
      /** @description Period of the line item applies to for revenue recognition pruposes.
       *
       *     Billing always treats periods as start being inclusive and end being exclusive. */
      period: components['schemas']['Period']
      /**
       * Format: date-time
       * @description The time this line item should be invoiced.
       * @example 2023-01-01T01:01:01.001Z
       */
      invoiceAt: Date
      /**
       * @deprecated
       * @description Price of the usage-based item being sold.
       */
      price?: components['schemas']['RateCardUsageBasedPrice']
      /**
       * @deprecated
       * @description The feature that the usage is based on.
       */
      featureKey?: string
      /** @description The rate card that is used for this line.
       *
       *     The rate card captures the intent of the price and discounts for the usage-based item. */
      rateCard?: components['schemas']['InvoiceUsageBasedRateCard']
    }
    /** @description InvoicePendingLineCreate represents the create model for a pending invoice line. */
    InvoicePendingLineCreateInput: {
      /** @description The currency of the lines to be created. */
      currency: components['schemas']['CurrencyCode']
      /** @description The lines to be created. */
      lines: components['schemas']['InvoicePendingLineCreate'][]
    }
    /** @description InvoicePendingLineCreateResponse represents the response from the create pending line endpoint. */
    InvoicePendingLineCreateResponse: {
      /** @description The lines that were created. */
      readonly lines: components['schemas']['InvoiceLine'][]
      /** @description The invoice containing the created lines. */
      readonly invoice: components['schemas']['Invoice']
      /** @description Whether the invoice was newly created. */
      readonly isInvoiceNew: boolean
    }
    /** @description InvoicePendingLinesActionFiltersInput specifies which lines to include in the invoice. */
    InvoicePendingLinesActionFiltersInput: {
      /** @description The pending line items to include in the invoice, if not provided:
       *     - all line items that have invoice_at < asOf will be included
       *     - [progressive billing only] all usage based line items will be included up to asOf, new
       *     usage-based line items will be staged for the rest of the billing cycle
       *
       *     All lineIDs present in the list, must exists and must be invoicable as of asOf, or the action will fail. */
      lineIds?: string[]
    }
    /** @description BillingInvoiceActionInput is the input for creating an invoice.
     *
     *     Invoice creation is always based on already pending line items created by the billingCreateLineByCustomer
     *     operation. Empty invoices are not allowed. */
    InvoicePendingLinesActionInput: {
      /** @description Filters to apply when creating the invoice. */
      filters?: components['schemas']['InvoicePendingLinesActionFiltersInput']
      /**
       * Format: date-time
       * @description The time as of which the invoice is created.
       *
       *     If not provided, the current time is used.
       * @example 2023-01-01T01:01:01.001Z
       */
      asOf?: Date
      /**
       * @description The customer ID for which to create the invoice.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      customerId: string
      /** @description Override the progressive billing setting of the customer.
       *
       *     Can be used to disable/enable progressive billing in case the business logic
       *     requires it, if not provided the billing profile's progressive billing setting will be used. */
      progressiveBillingOverride?: boolean
    }
    /** @description Reference to an invoice. */
    InvoiceReference: {
      /**
       * @description The ID of the invoice.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      readonly id: string
      /** @description The number of the invoice. */
      readonly number?: components['schemas']['InvoiceNumber']
    }
    /** @description InvoiceReplaceUpdate represents the update model for an invoice. */
    InvoiceReplaceUpdate: {
      /**
       * Description
       * @description Optional description of the resource. Maximum 1024 characters.
       */
      description?: string
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata'] | null
      /** @description The supplier of the lines included in the invoice. */
      supplier: components['schemas']['BillingPartyReplaceUpdate']
      /** @description The customer the invoice is sent to. */
      customer: components['schemas']['BillingPartyReplaceUpdate']
      /** @description The lines included in the invoice. */
      lines: components['schemas']['InvoiceLineReplaceUpdate'][]
      /** @description The workflow settings for the invoice. */
      workflow: components['schemas']['InvoiceWorkflowReplaceUpdate']
    }
    /** @description InvoiceSimulationInput is the input for simulating an invoice. */
    InvoiceSimulationInput: {
      /** @description The number of the invoice. */
      number?: components['schemas']['InvoiceNumber']
      /** @description Currency for all invoice line items.
       *
       *     Multi currency invoices are not supported yet. */
      currency: components['schemas']['CurrencyCode']
      /** @description Lines to be included in the generated invoice. */
      lines: components['schemas']['InvoiceSimulationLine'][]
    }
    /** @description InvoiceSimulationLine represents a usage-based line item that can be input to the simulation endpoint. */
    InvoiceSimulationLine: {
      /**
       * Display name
       * @description Human-readable name for the resource. Between 1 and 256 characters.
       */
      name: string
      /**
       * Description
       * @description Optional description of the resource. Maximum 1024 characters.
       */
      description?: string
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata'] | null
      /**
       * @deprecated
       * @description Tax config specify the tax configuration for this line.
       */
      taxConfig?: components['schemas']['TaxConfig']
      /** @description Period of the line item applies to for revenue recognition pruposes.
       *
       *     Billing always treats periods as start being inclusive and end being exclusive. */
      period: components['schemas']['Period']
      /**
       * Format: date-time
       * @description The time this line item should be invoiced.
       * @example 2023-01-01T01:01:01.001Z
       */
      invoiceAt: Date
      /**
       * @deprecated
       * @description Price of the usage-based item being sold.
       */
      price?: components['schemas']['RateCardUsageBasedPrice']
      /**
       * @deprecated
       * @description The feature that the usage is based on.
       */
      featureKey?: string
      /** @description The rate card that is used for this line.
       *
       *     The rate card captures the intent of the price and discounts for the usage-based item. */
      rateCard?: components['schemas']['InvoiceUsageBasedRateCard']
      /** @description The quantity of the item being sold. */
      quantity: components['schemas']['Numeric']
      /** @description The quantity of the item used before this line's period, if the line is billed progressively. */
      preLinePeriodQuantity?: components['schemas']['Numeric']
      /**
       * @description ID of the line. If not specified it will be auto-generated.
       *
       *     When discounts are specified, this must be provided, so that the discount can reference it.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      id?: string
    }
    /**
     * @description InvoiceStatus describes the status of an invoice.
     * @enum {string}
     */
    InvoiceStatus:
      | 'gathering'
      | 'draft'
      | 'issuing'
      | 'issued'
      | 'payment_processing'
      | 'overdue'
      | 'paid'
      | 'uncollectible'
      | 'voided'
    /** @description InvoiceStatusDetails represents the details of the invoice status.
     *
     *     API users are encouraged to rely on the immutable/failed/avaliableActions fields to determine
     *     the next steps of the invoice instead of the extendedStatus field. */
    InvoiceStatusDetails: {
      /** @description Is the invoice editable? */
      readonly immutable: boolean
      /** @description Is the invoice in a failed state? */
      readonly failed: boolean
      /** @description Extended status information for the invoice. */
      readonly extendedStatus: string
      /** @description The actions that can be performed on the invoice. */
      availableActions: components['schemas']['InvoiceAvailableActions']
    }
    /** @description Totals contains the summaries of all calculations for the invoice. */
    InvoiceTotals: {
      /** @description The total value of the line before taxes, discounts and commitments. */
      readonly amount: components['schemas']['Numeric']
      /** @description The amount of value of the line that are due to additional charges. */
      readonly chargesTotal: components['schemas']['Numeric']
      /** @description The amount of value of the line that are due to discounts. */
      readonly discountsTotal: components['schemas']['Numeric']
      /** @description The total amount of taxes that are included in the line. */
      readonly taxesInclusiveTotal: components['schemas']['Numeric']
      /** @description The total amount of taxes that are added on top of amount from the line. */
      readonly taxesExclusiveTotal: components['schemas']['Numeric']
      /** @description The total amount of taxes for this line. */
      readonly taxesTotal: components['schemas']['Numeric']
      /** @description The total amount value of the line after taxes, discounts and commitments. */
      readonly total: components['schemas']['Numeric']
    }
    /**
     * @description InvoiceType represents the type of invoice.
     *
     *     The type of invoice determines the purpose of the invoice and how it should be handled.
     * @enum {string}
     */
    InvoiceType: 'standard' | 'credit_note'
    /** @description InvoiceUsageBasedRateCard represents the rate card (intent) for an usage-based line. */
    InvoiceUsageBasedRateCard: {
      /**
       * Feature key
       * @description The feature the customer is entitled to use.
       */
      featureKey?: string
      /**
       * Tax config
       * @description The tax config of the rate card.
       *     When undefined, the tax config of the feature or the default tax config of the plan is used.
       */
      taxConfig?: components['schemas']['TaxConfig']
      /** @description The price of the rate card.
       *     When null, the feature or service is free. */
      price: components['schemas']['RateCardUsageBasedPrice'] | null
      /** @description The discounts that are applied to the line. */
      discounts?: components['schemas']['BillingDiscounts']
    }
    /** @description InvoiceWorkflowInvoicingSettingsReplaceUpdate represents the update model for the invoicing settings of an invoice workflow. */
    InvoiceWorkflowInvoicingSettingsReplaceUpdate: {
      /**
       * @description Whether to automatically issue the invoice after the draftPeriod has passed.
       * @default true
       */
      autoAdvance?: boolean
      /**
       * Format: ISO8601
       * @description The period for the invoice to be kept in draft status for manual reviews.
       * @default P0D
       * @example P1D
       */
      draftPeriod?: string
      /**
       * Format: ISO8601
       * @description The period after which the invoice is due.
       *     With some payment solutions it's only applicable for manual collection method.
       * @default P30D
       * @example P30D
       */
      dueAfter?: string
      /** @description Default tax configuration to apply to the invoices. */
      defaultTaxConfig?: components['schemas']['TaxConfig']
    }
    /** @description InvoiceWorkflowReplaceUpdate represents the update model for an invoice workflow.
     *
     *     Fields that are immutable a re removed from the model. This is based on InvoiceWorkflowSettings. */
    InvoiceWorkflowReplaceUpdate: {
      /** @description The workflow used for this invoice. */
      workflow: components['schemas']['InvoiceWorkflowSettingsReplaceUpdate']
    }
    /** @description InvoiceWorkflowSettings represents the workflow settings used by the invoice.
     *
     *     This is a clone of the billing profile's workflow settings at the time of invoice creation
     *     with customer overrides considered. */
    InvoiceWorkflowSettings: {
      /** @description The apps that will be used to orchestrate the invoice's workflow. */
      readonly apps?: components['schemas']['BillingProfileAppsOrReference']
      /**
       * @description sourceBillingProfileID is the billing profile on which the workflow was based on.
       *
       *     The profile is snapshotted on invoice creation, after which it can be altered independently
       *     of the profile itself.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      readonly sourceBillingProfileId: string
      /** @description The workflow details used by this invoice. */
      workflow: components['schemas']['BillingWorkflow']
    }
    /** @description Mutable workflow settings for an invoice.
     *
     *     Other fields on the invoice's workflow are not mutable, they serve as a history of the invoice's workflow
     *     at creation time. */
    InvoiceWorkflowSettingsReplaceUpdate: {
      /** @description The invoicing settings for this workflow */
      invoicing: components['schemas']['InvoiceWorkflowInvoicingSettingsReplaceUpdate']
      /** @description The payment settings for this workflow */
      payment: components['schemas']['BillingWorkflowPaymentSettings']
    }
    /** @description List entitlements result */
    ListEntitlementsResult:
      | components['schemas']['Entitlement'][]
      | components['schemas']['EntitlementPaginatedResponse']
    /** @description List features result */
    ListFeaturesResult:
      | components['schemas']['Feature'][]
      | components['schemas']['FeaturePaginatedResponse']
    /** @description Marketplace install response. */
    MarketplaceInstallResponse: {
      app: components['schemas']['App']
      /** @description Default for capabilities */
      defaultForCapabilityTypes: components['schemas']['AppCapabilityType'][]
    }
    /**
     * @description A marketplace listing.
     *     Represent an available app in the app marketplace that can be installed to the organization.
     *
     *     Marketplace apps only exist in config so they don't extend the Resource model.
     * @example {
     *       "type": "stripe",
     *       "name": "Stripe",
     *       "description": "Stripe interation allows you to collect payments with Stripe.",
     *       "capabilities": [
     *         {
     *           "type": "calculateTax",
     *           "key": "stripe_calculate_tax",
     *           "name": "Calculate Tax",
     *           "description": "Stripe Tax calculates tax portion of the invoices."
     *         },
     *         {
     *           "type": "invoiceCustomers",
     *           "key": "stripe_invoice_customers",
     *           "name": "Invoice Customers",
     *           "description": "Stripe invoices customers with due amount."
     *         },
     *         {
     *           "type": "collectPayments",
     *           "key": "stripe_collect_payments",
     *           "name": "Collect Payments",
     *           "description": "Stripe payments collects outstanding revenue with Stripe customer's default payment method."
     *         }
     *       ],
     *       "installMethods": [
     *         "with_oauth2",
     *         "with_api_key"
     *       ]
     *     }
     */
    MarketplaceListing: {
      /** @description The app's type */
      type: components['schemas']['AppType']
      /** @description The app's name. */
      name: string
      /** @description The app's description. */
      description: string
      /** @description The app's capabilities. */
      capabilities: components['schemas']['AppCapability'][]
      /** @description Install methods.
       *
       *     List of methods to install the app. */
      installMethods: components['schemas']['InstallMethod'][]
    }
    /** @description Paginated response */
    MarketplaceListingPaginatedResponse: {
      /**
       * @description The total number of items.
       * @example 500
       */
      totalCount: number
      /**
       * @description The page index.
       * @example 1
       */
      page: number
      /**
       * @description The maximum number of items per page.
       * @example 100
       */
      pageSize: number
      /** @description The items in the current page. */
      items: components['schemas']['MarketplaceListing'][]
    }
    /** @description Measure usage from */
    MeasureUsageFrom:
      | components['schemas']['MeasureUsageFromPreset']
      | components['schemas']['MeasureUsageFromTime']
    /**
     * @description Start of measurement options
     * @enum {string}
     */
    MeasureUsageFromPreset: 'CURRENT_PERIOD_START' | 'NOW'
    /**
     * Format: date-time
     * @description [RFC3339](https://tools.ietf.org/html/rfc3339) formatted date-time string in UTC.
     * @example 2023-01-01T01:01:01.001Z
     */
    MeasureUsageFromTime: Date
    /**
     * @description Set of key-value pairs.
     *     Metadata can be used to store additional information about a resource.
     * @example {
     *       "externalId": "019142cc-a016-796a-8113-1a942fecd26d"
     *     }
     */
    Metadata: {
      [key: string]: string
    }
    /**
     * @description A meter is a configuration that defines how to match and aggregate events.
     * @example {
     *       "id": "01G65Z755AFWAKHE12NY0CQ9FH",
     *       "slug": "tokens_total",
     *       "name": "Tokens Total",
     *       "description": "AI Token Usage",
     *       "aggregation": "SUM",
     *       "eventType": "prompt",
     *       "valueProperty": "$.tokens",
     *       "groupBy": {
     *         "model": "$.model",
     *         "type": "$.type"
     *       },
     *       "createdAt": "2024-01-01T01:01:01.001Z",
     *       "updatedAt": "2024-01-01T01:01:01.001Z"
     *     }
     */
    Meter: {
      /**
       * ID
       * @description A unique identifier for the resource.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      readonly id: string
      /**
       * Description
       * @description Optional description of the resource. Maximum 1024 characters.
       */
      description?: string
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata'] | null
      /**
       * Creation Time
       * Format: date-time
       * @description Timestamp of when the resource was created.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly createdAt: Date
      /**
       * Last Update Time
       * Format: date-time
       * @description Timestamp of when the resource was last updated.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly updatedAt: Date
      /**
       * Deletion Time
       * Format: date-time
       * @description Timestamp of when the resource was permanently deleted.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly deletedAt?: Date
      /**
       * Display name
       * @description Human-readable name for the resource. Between 1 and 256 characters.
       *     Defaults to the slug if not specified.
       */
      name?: string
      /**
       * @description A unique, human-readable identifier for the meter.
       *     Must consist only alphanumeric and underscore characters.
       * @example tokens_total
       */
      slug: string
      /**
       * @description The aggregation type to use for the meter.
       * @example SUM
       */
      aggregation: components['schemas']['MeterAggregation']
      /**
       * @description The event type to aggregate.
       * @example prompt
       */
      eventType: string
      /**
       * Format: date-time
       * @description The date since the meter should include events.
       *     Useful to skip old events.
       *     If not specified, all historical events are included.
       * @example 2023-01-01T01:01:01.001Z
       */
      eventFrom?: Date
      /**
       * @description JSONPath expression to extract the value from the ingested event's data property.
       *
       *     The ingested value for SUM, AVG, MIN, and MAX aggregations is a number or a string that can be parsed to a number.
       *
       *     For UNIQUE_COUNT aggregation, the ingested value must be a string. For COUNT aggregation the valueProperty is ignored.
       * @example $.tokens
       */
      valueProperty?: string
      /**
       * @description Named JSONPath expressions to extract the group by values from the event data.
       *
       *     Keys must be unique and consist only alphanumeric and underscore characters.
       * @example {
       *       "type": "$.type"
       *     }
       */
      groupBy?: {
        [key: string]: string
      }
    }
    /**
     * @description The aggregation type to use for the meter.
     * @enum {string}
     */
    MeterAggregation:
      | 'SUM'
      | 'COUNT'
      | 'UNIQUE_COUNT'
      | 'AVG'
      | 'MIN'
      | 'MAX'
      | 'LATEST'
    /**
     * @description A meter create model.
     * @example {
     *       "slug": "tokens_total",
     *       "name": "Tokens Total",
     *       "description": "AI Token Usage",
     *       "aggregation": "SUM",
     *       "eventType": "prompt",
     *       "valueProperty": "$.tokens",
     *       "groupBy": {
     *         "model": "$.model",
     *         "type": "$.type"
     *       }
     *     }
     */
    MeterCreate: {
      /**
       * Description
       * @description Optional description of the resource. Maximum 1024 characters.
       */
      description?: string
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata'] | null
      /**
       * Display name
       * @description Human-readable name for the resource. Between 1 and 256 characters.
       *     Defaults to the slug if not specified.
       */
      name?: string
      /**
       * @description A unique, human-readable identifier for the meter.
       *     Must consist only alphanumeric and underscore characters.
       * @example tokens_total
       */
      slug: string
      /**
       * @description The aggregation type to use for the meter.
       * @example SUM
       */
      aggregation: components['schemas']['MeterAggregation']
      /**
       * @description The event type to aggregate.
       * @example prompt
       */
      eventType: string
      /**
       * Format: date-time
       * @description The date since the meter should include events.
       *     Useful to skip old events.
       *     If not specified, all historical events are included.
       * @example 2023-01-01T01:01:01.001Z
       */
      eventFrom?: Date
      /**
       * @description JSONPath expression to extract the value from the ingested event's data property.
       *
       *     The ingested value for SUM, AVG, MIN, and MAX aggregations is a number or a string that can be parsed to a number.
       *
       *     For UNIQUE_COUNT aggregation, the ingested value must be a string. For COUNT aggregation the valueProperty is ignored.
       * @example $.tokens
       */
      valueProperty?: string
      /**
       * @description Named JSONPath expressions to extract the group by values from the event data.
       *
       *     Keys must be unique and consist only alphanumeric and underscore characters.
       * @example {
       *       "type": "$.type"
       *     }
       */
      groupBy?: {
        [key: string]: string
      }
    }
    /**
     * @description Order by options for meters.
     * @enum {string}
     */
    MeterOrderBy: 'key' | 'name' | 'aggregation' | 'createdAt' | 'updatedAt'
    /** @description A meter query request. */
    MeterQueryRequest: {
      /**
       * @description Client ID
       *     Useful to track progress of a query.
       * @example f74e58ed-94ce-4041-ae06-cf45420451a3
       */
      clientId?: string
      /**
       * Format: date-time
       * @description Start date-time in RFC 3339 format.
       *
       *     Inclusive.
       * @example 2023-01-01T01:01:01.001Z
       */
      from?: Date
      /**
       * Format: date-time
       * @description End date-time in RFC 3339 format.
       *
       *     Inclusive.
       * @example 2023-01-01T01:01:01.001Z
       */
      to?: Date
      /**
       * @description If not specified, a single usage aggregate will be returned for the entirety of the specified period for each subject and group.
       * @example DAY
       */
      windowSize?: components['schemas']['WindowSize']
      /**
       * @description The value is the name of the time zone as defined in the IANA Time Zone Database (http://www.iana.org/time-zones).
       *     If not specified, the UTC timezone will be used.
       * @default UTC
       * @example UTC
       */
      windowTimeZone?: string
      /**
       * @description Filtering by multiple subjects.
       * @example [
       *       "customer-1",
       *       "customer-2"
       *     ]
       */
      subject?: string[]
      /**
       * @description Simple filter for group bys with exact match.
       * @example {
       *       "model": [
       *         "gpt-4-turbo",
       *         "gpt-4o"
       *       ],
       *       "type": [
       *         "prompt"
       *       ]
       *     }
       */
      filterGroupBy?: {
        [key: string]: string[]
      }
      /**
       * @description If not specified a single aggregate will be returned for each subject and time window.
       *     `subject` is a reserved group by value.
       * @example [
       *       "model",
       *       "type"
       *     ]
       */
      groupBy?: string[]
    }
    /**
     * @description The result of a meter query.
     * @example {
     *       "from": "2023-01-01T00:00:00Z",
     *       "to": "2023-01-02T00:00:00Z",
     *       "windowSize": "DAY",
     *       "data": [
     *         {
     *           "value": 12,
     *           "windowStart": "2023-01-01T00:00:00Z",
     *           "windowEnd": "2023-01-02T00:00:00Z",
     *           "subject": "customer-1",
     *           "groupBy": {
     *             "model": "gpt-4-turbo",
     *             "type": "prompt"
     *           }
     *         }
     *       ]
     *     }
     */
    MeterQueryResult: {
      /**
       * Format: date-time
       * @description The start of the period the usage is queried from.
       *     If not specified, the usage is queried from the beginning of time.
       * @example 2023-01-01T01:01:01.001Z
       */
      from?: Date
      /**
       * Format: date-time
       * @description The end of the period the usage is queried to.
       *     If not specified, the usage is queried up to the current time.
       * @example 2023-01-01T01:01:01.001Z
       */
      to?: Date
      /** @description The window size that the usage is aggregated.
       *     If not specified, the usage is aggregated over the entire period. */
      windowSize?: components['schemas']['WindowSize']
      /** @description The usage data.
       *     If no data is available, an empty array is returned. */
      data: components['schemas']['MeterQueryRow'][]
    }
    /**
     * @description A row in the result of a meter query.
     * @example {
     *       "value": 12,
     *       "windowStart": "2023-01-01T00:00:00Z",
     *       "windowEnd": "2023-01-02T00:00:00Z",
     *       "subject": "customer-1",
     *       "groupBy": {
     *         "model": "gpt-4-turbo",
     *         "type": "prompt"
     *       }
     *     }
     */
    MeterQueryRow: {
      /**
       * Format: double
       * @description The aggregated value.
       */
      value: number
      /**
       * Format: date-time
       * @description The start of the window the value is aggregated over.
       * @example 2023-01-01T01:01:01.001Z
       */
      windowStart: Date
      /**
       * Format: date-time
       * @description The end of the window the value is aggregated over.
       * @example 2023-01-01T01:01:01.001Z
       */
      windowEnd: Date
      /** @description The subject the value is aggregated over.
       *     If not specified, the value is aggregated over all subjects. */
      subject: string | null
      /** @description The group by values the value is aggregated over. */
      groupBy: {
        [key: string]: string | null
      }
    }
    /**
     * @description A meter update model.
     *
     *     Only the properties that can be updated are included.
     *     For example, the slug and aggregation cannot be updated.
     * @example {
     *       "name": "Tokens Total",
     *       "description": "AI Token Usage",
     *       "groupBy": {
     *         "model": "$.model",
     *         "type": "$.type"
     *       }
     *     }
     */
    MeterUpdate: {
      /**
       * Description
       * @description Optional description of the resource. Maximum 1024 characters.
       */
      description?: string
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata'] | null
      /**
       * Display name
       * @description Human-readable name for the resource. Between 1 and 256 characters.
       *     Defaults to the slug if not specified.
       */
      name?: string
      /**
       * @description Named JSONPath expressions to extract the group by values from the event data.
       *
       *     Keys must be unique and consist only alphanumeric and underscore characters.
       * @example {
       *       "type": "$.type"
       *     }
       */
      groupBy?: {
        [key: string]: string
      }
    }
    /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
    NotFoundProblemResponse: components['schemas']['UnexpectedProblemResponse']
    /** @description The server does not support the functionality required to fulfill the request. */
    NotImplementedProblemResponse: components['schemas']['UnexpectedProblemResponse']
    /** @description Notification channel. */
    NotificationChannel: components['schemas']['NotificationChannelWebhook']
    /** @description Union type for requests creating new notification channel with certain type. */
    NotificationChannelCreateRequest: components['schemas']['NotificationChannelWebhookCreateRequest']
    /** @description Metadata only fields of a notification channel. */
    NotificationChannelMeta: {
      /**
       * Channel Unique Identifier
       * @description Identifies the notification channel.
       * @example 01ARZ3NDEKTSV4RRFFQ69G5FAV
       */
      readonly id: string
      /**
       * Channel Type
       * @description Notification channel type.
       */
      type: components['schemas']['NotificationChannelType']
    }
    /**
     * @description Order by options for notification channels.
     * @enum {string}
     */
    NotificationChannelOrderBy: 'id' | 'type' | 'createdAt' | 'updatedAt'
    /** @description Paginated response */
    NotificationChannelPaginatedResponse: {
      /**
       * @description The total number of items.
       * @example 500
       */
      totalCount: number
      /**
       * @description The page index.
       * @example 1
       */
      page: number
      /**
       * @description The maximum number of items per page.
       * @example 100
       */
      pageSize: number
      /** @description The items in the current page. */
      items: components['schemas']['NotificationChannel'][]
    }
    /**
     * @description Type of the notification channel.
     * @enum {string}
     */
    NotificationChannelType: 'WEBHOOK'
    /** @description Notification channel with webhook type. */
    NotificationChannelWebhook: {
      /**
       * Creation Time
       * Format: date-time
       * @description Timestamp of when the resource was created.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly createdAt: Date
      /**
       * Last Update Time
       * Format: date-time
       * @description Timestamp of when the resource was last updated.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly updatedAt: Date
      /**
       * Deletion Time
       * Format: date-time
       * @description Timestamp of when the resource was permanently deleted.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly deletedAt?: Date
      /**
       * Channel Unique Identifier
       * @description Identifies the notification channel.
       * @example 01ARZ3NDEKTSV4RRFFQ69G5FAV
       */
      readonly id: string
      /**
       * Channel Type
       * @description Notification channel type.
       * @enum {string}
       */
      type: 'WEBHOOK'
      /**
       * Channel Name
       * @description User friendly name of the channel.
       * @example customer-webhook
       */
      name: string
      /**
       * Channel Disabled
       * @description Whether the channel is disabled or not.
       * @default false
       * @example true
       */
      disabled?: boolean
      /**
       * Webhook URL
       * @description Webhook URL where the notification is sent.
       * @example https://example.com/webhook
       */
      url: string
      /**
       * Custom HTTP Headers
       * @description Custom HTTP headers sent as part of the webhook request.
       */
      customHeaders?: {
        [key: string]: string
      }
      /**
       * Signing Secret
       * @description Signing secret used for webhook request validation on the receiving end.
       *
       *     Format: `base64` encoded random bytes optionally prefixed with `whsec_`. Recommended size: 24
       * @example whsec_S6g2HLnTwd9AhHwUIMFggVS9OfoPafN8
       */
      signingSecret?: string
    }
    /** @description Request with input parameters for creating new notification channel with webhook type. */
    NotificationChannelWebhookCreateRequest: {
      /**
       * Channel Type
       * @description Notification channel type.
       * @enum {string}
       */
      type: 'WEBHOOK'
      /**
       * Channel Name
       * @description User friendly name of the channel.
       * @example customer-webhook
       */
      name: string
      /**
       * Channel Disabled
       * @description Whether the channel is disabled or not.
       * @default false
       * @example true
       */
      disabled?: boolean
      /**
       * Webhook URL
       * @description Webhook URL where the notification is sent.
       * @example https://example.com/webhook
       */
      url: string
      /**
       * Custom HTTP Headers
       * @description Custom HTTP headers sent as part of the webhook request.
       */
      customHeaders?: {
        [key: string]: string
      }
      /**
       * Signing Secret
       * @description Signing secret used for webhook request validation on the receiving end.
       *
       *     Format: `base64` encoded random bytes optionally prefixed with `whsec_`. Recommended size: 24
       * @example whsec_S6g2HLnTwd9AhHwUIMFggVS9OfoPafN8
       */
      signingSecret?: string
    }
    /** @description Type of the notification event. */
    NotificationEvent: {
      /**
       * Event Identifier
       * @description A unique identifier of the notification event.
       * @example 01J2KNP1YTXQRXHTDJ4KPR7PZ0
       */
      readonly id: string
      /**
       * Event Type
       * @description Type of the notification event.
       */
      readonly type: components['schemas']['NotificationEventType']
      /**
       * Creation Time
       * Format: date-time
       * @description Timestamp when the notification event was created in RFC 3339 format.
       * @example 2023-01-01T01:01:01.001Z
       */
      readonly createdAt: Date
      /** @description The nnotification rule which generated this event. */
      readonly rule: components['schemas']['NotificationRule']
      /**
       * Delivery Status
       * @description The delivery status of the notification event.
       */
      readonly deliveryStatus: components['schemas']['NotificationEventDeliveryStatus'][]
      /** @description Timestamp when the notification event was created in RFC 3339 format. */
      readonly payload: components['schemas']['NotificationEventPayload']
      /**
       * Annotations
       * @description Set of key-value pairs managed by the system. Cannot be modified by user.
       */
      readonly annotations?: components['schemas']['Annotations']
    }
    /** @description Payload for notification event with `entitlements.balance.threshold` type. */
    NotificationEventBalanceThresholdPayload: {
      /**
       * Notification Event Identifier
       * @description A unique identifier for the notification event the payload belongs to.
       * @example 01J2KNP1YTXQRXHTDJ4KPR7PZ0
       */
      readonly id: string
      /**
       * @description Type of the notification event. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'entitlements.balance.threshold'
      /**
       * Creation Time
       * Format: date-time
       * @description Timestamp when the notification event was created in RFC 3339 format.
       * @example 2023-01-01T01:01:01.001Z
       */
      readonly timestamp: Date
      /**
       * Payload Data
       * @description The data of the payload.
       */
      readonly data: components['schemas']['NotificationEventBalanceThresholdPayloadData']
    }
    /** @description Data of the payload for notification event with `entitlements.balance.threshold` type. */
    NotificationEventBalanceThresholdPayloadData: {
      /** Entitlement */
      readonly entitlement: components['schemas']['EntitlementMetered']
      /** Feature */
      readonly feature: components['schemas']['Feature']
      /** Subject */
      readonly subject: components['schemas']['Subject']
      /** Entitlement Value */
      readonly value: components['schemas']['EntitlementValue']
      /** Threshold */
      readonly threshold: components['schemas']['NotificationRuleBalanceThresholdValue']
    }
    /** @description The delivery status of the notification event. */
    NotificationEventDeliveryStatus: {
      /**
       * @description Delivery state of the notification event to the channel.
       * @example SUCCESS
       */
      readonly state: components['schemas']['NotificationEventDeliveryStatusState']
      /**
       * State Reason
       * @description The reason of the last deliverry state update.
       * @example Failed to dispatch event due to provider error.
       */
      readonly reason: string
      /**
       * Last Update Time
       * Format: date-time
       * @description Timestamp of when the status was last updated in RFC 3339 format.
       * @example 2023-01-01T01:01:01.001Z
       */
      readonly updatedAt: Date
      /**
       * Notification Channel
       * @description Notification channel the delivery sattus associated with.
       */
      readonly channel: components['schemas']['NotificationChannelMeta']
    }
    /**
     * Delivery State
     * @description The delivery state of the notification event to the channel.
     * @enum {string}
     */
    NotificationEventDeliveryStatusState:
      | 'SUCCESS'
      | 'FAILED'
      | 'SENDING'
      | 'PENDING'
    /** @description Base data for any payload with entitlement entitlement value. */
    NotificationEventEntitlementValuePayloadBase: {
      /** Entitlement */
      readonly entitlement: components['schemas']['EntitlementMetered']
      /** Feature */
      readonly feature: components['schemas']['Feature']
      /** Subject */
      readonly subject: components['schemas']['Subject']
      /** Entitlement Value */
      readonly value: components['schemas']['EntitlementValue']
    }
    /** @description Payload for notification event with `invoice.created` type. */
    NotificationEventInvoiceCreatedPayload: {
      /**
       * Notification Event Identifier
       * @description A unique identifier for the notification event the payload belongs to.
       * @example 01J2KNP1YTXQRXHTDJ4KPR7PZ0
       */
      readonly id: string
      /**
       * @description Type of the notification event. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'invoice.created'
      /**
       * Creation Time
       * Format: date-time
       * @description Timestamp when the notification event was created in RFC 3339 format.
       * @example 2023-01-01T01:01:01.001Z
       */
      readonly timestamp: Date
      /**
       * Payload Data
       * @description The data of the payload.
       */
      readonly data: components['schemas']['Invoice']
    }
    /** @description Payload for notification event with `invoice.updated` type. */
    NotificationEventInvoiceUpdatedPayload: {
      /**
       * Notification Event Identifier
       * @description A unique identifier for the notification event the payload belongs to.
       * @example 01J2KNP1YTXQRXHTDJ4KPR7PZ0
       */
      readonly id: string
      /**
       * @description Type of the notification event. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'invoice.updated'
      /**
       * Creation Time
       * Format: date-time
       * @description Timestamp when the notification event was created in RFC 3339 format.
       * @example 2023-01-01T01:01:01.001Z
       */
      readonly timestamp: Date
      /**
       * Payload Data
       * @description The data of the payload.
       */
      readonly data: components['schemas']['Invoice']
    }
    /**
     * @description Order by options for notification channels.
     * @enum {string}
     */
    NotificationEventOrderBy: 'id' | 'createdAt'
    /** @description Paginated response */
    NotificationEventPaginatedResponse: {
      /**
       * @description The total number of items.
       * @example 500
       */
      totalCount: number
      /**
       * @description The page index.
       * @example 1
       */
      page: number
      /**
       * @description The maximum number of items per page.
       * @example 100
       */
      pageSize: number
      /** @description The items in the current page. */
      items: components['schemas']['NotificationEvent'][]
    }
    /** @description The delivery status of the notification event. */
    NotificationEventPayload:
      | components['schemas']['NotificationEventResetPayload']
      | components['schemas']['NotificationEventBalanceThresholdPayload']
      | components['schemas']['NotificationEventInvoiceCreatedPayload']
      | components['schemas']['NotificationEventInvoiceUpdatedPayload']
    /** @description Payload for notification event with `entitlements.reset` type. */
    NotificationEventResetPayload: {
      /**
       * Notification Event Identifier
       * @description A unique identifier for the notification event the payload belongs to.
       * @example 01J2KNP1YTXQRXHTDJ4KPR7PZ0
       */
      readonly id: string
      /**
       * @description Type of the notification event. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'entitlements.reset'
      /**
       * Creation Time
       * Format: date-time
       * @description Timestamp when the notification event was created in RFC 3339 format.
       * @example 2023-01-01T01:01:01.001Z
       */
      readonly timestamp: Date
      /**
       * Payload Data
       * @description The data of the payload.
       */
      readonly data: components['schemas']['NotificationEventEntitlementValuePayloadBase']
    }
    /**
     * @description Type of the notification event.
     * @enum {string}
     */
    NotificationEventType:
      | 'entitlements.balance.threshold'
      | 'entitlements.reset'
      | 'invoice.created'
      | 'invoice.updated'
    /** @description Notification Rule. */
    NotificationRule:
      | components['schemas']['NotificationRuleBalanceThreshold']
      | components['schemas']['NotificationRuleEntitlementReset']
      | components['schemas']['NotificationRuleInvoiceCreated']
      | components['schemas']['NotificationRuleInvoiceUpdated']
    /** @description Notification rule with entitlements.balance.threshold type. */
    NotificationRuleBalanceThreshold: {
      /**
       * Creation Time
       * Format: date-time
       * @description Timestamp of when the resource was created.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly createdAt: Date
      /**
       * Last Update Time
       * Format: date-time
       * @description Timestamp of when the resource was last updated.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly updatedAt: Date
      /**
       * Deletion Time
       * Format: date-time
       * @description Timestamp of when the resource was permanently deleted.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly deletedAt?: Date
      /**
       * Rule Unique Identifier
       * @description Identifies the notification rule.
       * @example 01ARZ3NDEKTSV4RRFFQ69G5FAV
       */
      readonly id: string
      /**
       * @description Notification rule type. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'entitlements.balance.threshold'
      /**
       * Rule Name
       * @description The user friendly name of the notification rule.
       * @example Balance threshold reached
       */
      name: string
      /**
       * Rule Disabled
       * @description Whether the rule is disabled or not.
       * @default false
       * @example true
       */
      disabled?: boolean
      /**
       * Channels assigned to Rule
       * @description List of notification channels the rule applies to.
       */
      channels: components['schemas']['NotificationChannelMeta'][]
      /**
       * Entitlement Balance Thresholds
       * @description List of thresholds the rule suppose to be triggered.
       */
      thresholds: components['schemas']['NotificationRuleBalanceThresholdValue'][]
      /**
       * Features
       * @description Optional field containing list of features the rule applies to.
       */
      features?: components['schemas']['FeatureMeta'][]
    }
    /** @description Request with input parameters for creating new notification rule with entitlements.balance.threshold type. */
    NotificationRuleBalanceThresholdCreateRequest: {
      /**
       * @description Notification rule type. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'entitlements.balance.threshold'
      /**
       * Rule Name
       * @description The user friendly name of the notification rule.
       * @example Balance threshold reached
       */
      name: string
      /**
       * Rule Disabled
       * @description Whether the rule is disabled or not.
       * @default false
       * @example true
       */
      disabled?: boolean
      /**
       * Entitlement Balance Thresholds
       * @description List of thresholds the rule suppose to be triggered.
       */
      thresholds: components['schemas']['NotificationRuleBalanceThresholdValue'][]
      /**
       * Channels
       * @description List of notification channels the rule is applied to.
       */
      channels: string[]
      /**
       * Features
       * @description Optional field for defining the scope of notification by feature. It may contain features by id or key.
       */
      features?: string[]
    }
    /** @description Threshold value with multiple supported types. */
    NotificationRuleBalanceThresholdValue: {
      /**
       * Threshold Value
       * Format: double
       * @description Value of the threshold.
       * @example 100
       */
      value: number
      /**
       * @description Type of the threshold.
       * @example NUMBER
       */
      type: components['schemas']['NotificationRuleBalanceThresholdValueType']
    }
    /**
     * Notification balance threshold type
     * @description Type of the rule in the balance threshold specification.
     * @enum {string}
     */
    NotificationRuleBalanceThresholdValueType: 'PERCENT' | 'NUMBER'
    /** @description Union type for requests creating new notification rule with certain type. */
    NotificationRuleCreateRequest:
      | components['schemas']['NotificationRuleBalanceThresholdCreateRequest']
      | components['schemas']['NotificationRuleEntitlementResetCreateRequest']
      | components['schemas']['NotificationRuleInvoiceCreatedCreateRequest']
      | components['schemas']['NotificationRuleInvoiceUpdatedCreateRequest']
    /** @description Notification rule with entitlements.reset type. */
    NotificationRuleEntitlementReset: {
      /**
       * Creation Time
       * Format: date-time
       * @description Timestamp of when the resource was created.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly createdAt: Date
      /**
       * Last Update Time
       * Format: date-time
       * @description Timestamp of when the resource was last updated.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly updatedAt: Date
      /**
       * Deletion Time
       * Format: date-time
       * @description Timestamp of when the resource was permanently deleted.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly deletedAt?: Date
      /**
       * Rule Unique Identifier
       * @description Identifies the notification rule.
       * @example 01ARZ3NDEKTSV4RRFFQ69G5FAV
       */
      readonly id: string
      /**
       * @description Notification rule type. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'entitlements.reset'
      /**
       * Rule Name
       * @description The user friendly name of the notification rule.
       * @example Balance threshold reached
       */
      name: string
      /**
       * Rule Disabled
       * @description Whether the rule is disabled or not.
       * @default false
       * @example true
       */
      disabled?: boolean
      /**
       * Channels assigned to Rule
       * @description List of notification channels the rule applies to.
       */
      channels: components['schemas']['NotificationChannelMeta'][]
      /**
       * Features
       * @description Optional field containing list of features the rule applies to.
       */
      features?: components['schemas']['FeatureMeta'][]
    }
    /** @description Request with input parameters for creating new notification rule with entitlements.reset type. */
    NotificationRuleEntitlementResetCreateRequest: {
      /**
       * @description Notification rule type. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'entitlements.reset'
      /**
       * Rule Name
       * @description The user friendly name of the notification rule.
       * @example Balance threshold reached
       */
      name: string
      /**
       * Rule Disabled
       * @description Whether the rule is disabled or not.
       * @default false
       * @example true
       */
      disabled?: boolean
      /**
       * Channels
       * @description List of notification channels the rule is applied to.
       */
      channels: string[]
      /**
       * Features
       * @description Optional field for defining the scope of notification by feature. It may contain features by id or key.
       */
      features?: string[]
    }
    /** @description Notification rule with invoice.created type. */
    NotificationRuleInvoiceCreated: {
      /**
       * Creation Time
       * Format: date-time
       * @description Timestamp of when the resource was created.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly createdAt: Date
      /**
       * Last Update Time
       * Format: date-time
       * @description Timestamp of when the resource was last updated.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly updatedAt: Date
      /**
       * Deletion Time
       * Format: date-time
       * @description Timestamp of when the resource was permanently deleted.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly deletedAt?: Date
      /**
       * Rule Unique Identifier
       * @description Identifies the notification rule.
       * @example 01ARZ3NDEKTSV4RRFFQ69G5FAV
       */
      readonly id: string
      /**
       * @description Notification rule type. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'invoice.created'
      /**
       * Rule Name
       * @description The user friendly name of the notification rule.
       * @example Balance threshold reached
       */
      name: string
      /**
       * Rule Disabled
       * @description Whether the rule is disabled or not.
       * @default false
       * @example true
       */
      disabled?: boolean
      /**
       * Channels assigned to Rule
       * @description List of notification channels the rule applies to.
       */
      channels: components['schemas']['NotificationChannelMeta'][]
    }
    /** @description Request with input parameters for creating new notification rule with invoice.created type. */
    NotificationRuleInvoiceCreatedCreateRequest: {
      /**
       * @description Notification rule type. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'invoice.created'
      /**
       * Rule Name
       * @description The user friendly name of the notification rule.
       * @example Balance threshold reached
       */
      name: string
      /**
       * Rule Disabled
       * @description Whether the rule is disabled or not.
       * @default false
       * @example true
       */
      disabled?: boolean
      /**
       * Channels
       * @description List of notification channels the rule is applied to.
       */
      channels: string[]
    }
    /** @description Notification rule with invoice.updated type. */
    NotificationRuleInvoiceUpdated: {
      /**
       * Creation Time
       * Format: date-time
       * @description Timestamp of when the resource was created.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly createdAt: Date
      /**
       * Last Update Time
       * Format: date-time
       * @description Timestamp of when the resource was last updated.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly updatedAt: Date
      /**
       * Deletion Time
       * Format: date-time
       * @description Timestamp of when the resource was permanently deleted.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly deletedAt?: Date
      /**
       * Rule Unique Identifier
       * @description Identifies the notification rule.
       * @example 01ARZ3NDEKTSV4RRFFQ69G5FAV
       */
      readonly id: string
      /**
       * @description Notification rule type. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'invoice.updated'
      /**
       * Rule Name
       * @description The user friendly name of the notification rule.
       * @example Balance threshold reached
       */
      name: string
      /**
       * Rule Disabled
       * @description Whether the rule is disabled or not.
       * @default false
       * @example true
       */
      disabled?: boolean
      /**
       * Channels assigned to Rule
       * @description List of notification channels the rule applies to.
       */
      channels: components['schemas']['NotificationChannelMeta'][]
    }
    /** @description Request with input parameters for creating new notification rule with invoice.updated  type. */
    NotificationRuleInvoiceUpdatedCreateRequest: {
      /**
       * @description Notification rule type. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'invoice.updated'
      /**
       * Rule Name
       * @description The user friendly name of the notification rule.
       * @example Balance threshold reached
       */
      name: string
      /**
       * Rule Disabled
       * @description Whether the rule is disabled or not.
       * @default false
       * @example true
       */
      disabled?: boolean
      /**
       * Channels
       * @description List of notification channels the rule is applied to.
       */
      channels: string[]
    }
    /**
     * @description Order by options for notification channels.
     * @enum {string}
     */
    NotificationRuleOrderBy: 'id' | 'type' | 'createdAt' | 'updatedAt'
    /** @description Paginated response */
    NotificationRulePaginatedResponse: {
      /**
       * @description The total number of items.
       * @example 500
       */
      totalCount: number
      /**
       * @description The page index.
       * @example 1
       */
      page: number
      /**
       * @description The maximum number of items per page.
       * @example 100
       */
      pageSize: number
      /** @description The items in the current page. */
      items: components['schemas']['NotificationRule'][]
    }
    /** @description Numeric represents an arbitrary precision number. */
    Numeric: string
    /**
     * @description OAuth2 authorization code grant error types.
     * @enum {string}
     */
    OAuth2AuthorizationCodeGrantErrorType:
      | 'invalid_request'
      | 'unauthorized_client'
      | 'access_denied'
      | 'unsupported_response_type'
      | 'invalid_scope'
      | 'server_error'
      | 'temporarily_unavailable'
    /** @description Package price with spend commitments. */
    PackagePriceWithCommitments: {
      /**
       * @description The type of the price. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'package'
      /**
       * Amount
       * @description The price of one package.
       */
      amount: components['schemas']['Numeric']
      /**
       * Quantity per package
       * @description The quantity per package.
       */
      quantityPerPackage: components['schemas']['Numeric']
      /**
       * Minimum amount
       * @description The customer is committed to spend at least the amount.
       */
      minimumAmount?: components['schemas']['Numeric']
      /**
       * Maximum amount
       * @description The customer is limited to spend at most the amount.
       */
      maximumAmount?: components['schemas']['Numeric']
    }
    /** @description PaymentDueDate contains an amount that should be paid by the given date. */
    PaymentDueDate: {
      /**
       * Format: date-time
       * @description When the payment is due.
       * @example 2023-01-01T01:01:01.001Z
       */
      readonly dueAt: Date
      /** @description Other details to take into account for the due date. */
      readonly notes?: string
      /** @description How much needs to be paid by the date. */
      readonly amount: components['schemas']['Numeric']
      /** @description Percentage of the total that should be paid by the date. */
      readonly percent?: components['schemas']['Percentage']
      /** @description If different from the parent document's base currency. */
      readonly currency?: components['schemas']['CurrencyCode']
    }
    /** @description PaymentTermDueDate defines the terms for payment on a specific date. */
    PaymentTermDueDate: {
      /**
       * @description Type of terms to be applied.
       * @enum {string}
       */
      type: 'due_date'
      /** @description Text detail of the chosen payment terms. */
      readonly detail?: string
      /** @description Description of the conditions for payment. */
      readonly notes?: string
      /** @description When the payment is due. */
      readonly dueAt: components['schemas']['PaymentDueDate'][]
    }
    /** @description PaymentTermInstant defines the terms for payment on receipt of invoice. */
    PaymentTermInstant: {
      /**
       * @description Type of terms to be applied.
       * @enum {string}
       */
      type: 'instant'
      /** @description Text detail of the chosen payment terms. */
      readonly detail?: string
      /** @description Description of the conditions for payment. */
      readonly notes?: string
    }
    /** @description PaymentTerms defines the terms for payment. */
    PaymentTerms:
      | components['schemas']['PaymentTermInstant']
      | components['schemas']['PaymentTermDueDate']
    /**
     * Format: double
     * @description Numeric representation of a percentage
     *
     *     50% is represented as 50
     * @example 50
     */
    Percentage: number
    /** @description A period with a start and end time. */
    Period: {
      /**
       * Format: date-time
       * @description Period start time.
       * @example 2023-01-01T01:01:01.001Z
       */
      from: Date
      /**
       * Format: date-time
       * @description Period end time.
       * @example 2023-02-01T01:01:01.001Z
       */
      to: Date
    }
    /** @description Plans provide a template for subscriptions. */
    Plan: {
      /**
       * ID
       * @description A unique identifier for the resource.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      readonly id: string
      /**
       * Display name
       * @description Human-readable name for the resource. Between 1 and 256 characters.
       */
      name: string
      /**
       * Description
       * @description Optional description of the resource. Maximum 1024 characters.
       */
      description?: string
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata'] | null
      /**
       * Creation Time
       * Format: date-time
       * @description Timestamp of when the resource was created.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly createdAt: Date
      /**
       * Last Update Time
       * Format: date-time
       * @description Timestamp of when the resource was last updated.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly updatedAt: Date
      /**
       * Deletion Time
       * Format: date-time
       * @description Timestamp of when the resource was permanently deleted.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly deletedAt?: Date
      /**
       * Key
       * @description A semi-unique identifier for the resource.
       */
      key: string
      /** @description Alignment configuration for the plan. */
      alignment?: components['schemas']['Alignment']
      /**
       * Version
       * @description Version of the plan. Incremented when the plan is updated.
       * @default 1
       */
      readonly version: number
      /**
       * Currency
       * @description The currency code of the plan.
       * @default USD
       */
      currency: components['schemas']['CurrencyCode']
      /**
       * Billing cadence
       * Format: duration
       * @description The default billing cadence for subscriptions using this plan.
       *     Defines how often customers are billed using ISO8601 duration format.
       *     Examples: "P1M" (monthly), "P3M" (quarterly), "P1Y" (annually).
       * @example P1M
       */
      billingCadence: string
      /**
       * Pro-rating configuration
       * @description Default pro-rating configuration for subscriptions using this plan.
       * @default {
       *       "enabled": true,
       *       "mode": "prorate_prices"
       *     }
       */
      proRatingConfig?: components['schemas']['ProRatingConfig']
      /**
       * Effective start date
       * Format: date-time
       * @description The date and time when the plan becomes effective. When not specified, the plan is a draft.
       * @example 2023-01-01T01:01:01.001Z
       */
      readonly effectiveFrom?: Date
      /**
       * Effective end date
       * Format: date-time
       * @description The date and time when the plan is no longer effective. When not specified, the plan is effective indefinitely.
       * @example 2023-01-01T01:01:01.001Z
       */
      readonly effectiveTo?: Date
      /**
       * Status
       * @description The status of the plan.
       *     Computed based on the effective start and end dates:
       *     - draft = no effectiveFrom
       *     - active = effectiveFrom <= now < effectiveTo
       *     - archived / inactive = effectiveTo <= now
       *     - scheduled = now < effectiveFrom < effectiveTo
       */
      readonly status: components['schemas']['PlanStatus']
      /**
       * Plan phases
       * @description The plan phase or pricing ramp allows changing a plan's rate cards over time as a subscription progresses.
       *     A phase switch occurs only at the end of a billing period, ensuring that a single subscription invoice will not include charges from different phase prices.
       */
      phases: components['schemas']['PlanPhase'][]
      /**
       * Validation errors
       * @description List of validation errors.
       */
      readonly validationErrors:
        | components['schemas']['ValidationError'][]
        | null
    }
    /** @description The PlanAddon describes the association between a plan and add-on. */
    PlanAddon: {
      /**
       * Creation Time
       * Format: date-time
       * @description Timestamp of when the resource was created.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly createdAt: Date
      /**
       * Last Update Time
       * Format: date-time
       * @description Timestamp of when the resource was last updated.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly updatedAt: Date
      /**
       * Deletion Time
       * Format: date-time
       * @description Timestamp of when the resource was permanently deleted.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly deletedAt?: Date
      /**
       * Annotations
       * @description Set of key-value pairs managed by the system. Cannot be modified by user.
       */
      readonly annotations?: components['schemas']['Annotations']
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata']
      /**
       * Addon
       * @description Add-on object.
       */
      readonly addon: components['schemas']['Addon']
      /**
       * The plan phase from the add-on becomes purchasable
       * @description The key of the plan phase from the add-on becomes available for purchase.
       */
      fromPlanPhase: string
      /**
       * Max quantity of the add-on
       * @description The maximum number of times the add-on can be purchased for the plan.
       *     It is not applicable for add-ons with single instance type.
       */
      maxQuantity?: number
      /**
       * Validation errors
       * @description List of validation errors.
       */
      readonly validationErrors:
        | components['schemas']['ValidationError'][]
        | null
    }
    /** @description A plan add-on assignment create request. */
    PlanAddonCreate: {
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata']
      /**
       * The plan phase from the add-on becomes purchasable
       * @description The key of the plan phase from the add-on becomes available for purchase.
       */
      fromPlanPhase: string
      /**
       * Max quantity of the add-on
       * @description The maximum number of times the add-on can be purchased for the plan.
       *     It is not applicable for add-ons with single instance type.
       */
      maxQuantity?: number
      /**
       * Add-on unique identifier
       * @description The add-on unique identifier in ULID format.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      addonId: string
    }
    /**
     * @description Order by options for plan add-on assignments.
     * @enum {string}
     */
    PlanAddonOrderBy: 'id' | 'key' | 'version' | 'created_at' | 'updated_at'
    /** @description Paginated response */
    PlanAddonPaginatedResponse: {
      /**
       * @description The total number of items.
       * @example 500
       */
      totalCount: number
      /**
       * @description The page index.
       * @example 1
       */
      page: number
      /**
       * @description The maximum number of items per page.
       * @example 100
       */
      pageSize: number
      /** @description The items in the current page. */
      items: components['schemas']['PlanAddon'][]
    }
    /** @description Resource update operation model. */
    PlanAddonReplaceUpdate: {
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata']
      /**
       * The plan phase from the add-on becomes purchasable
       * @description The key of the plan phase from the add-on becomes available for purchase.
       */
      fromPlanPhase: string
      /**
       * Max quantity of the add-on
       * @description The maximum number of times the add-on can be purchased for the plan.
       *     It is not applicable for add-ons with single instance type.
       */
      maxQuantity?: number
    }
    /** @description Resource create operation model. */
    PlanCreate: {
      /**
       * Display name
       * @description Human-readable name for the resource. Between 1 and 256 characters.
       */
      name: string
      /**
       * Description
       * @description Optional description of the resource. Maximum 1024 characters.
       */
      description?: string
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata'] | null
      /**
       * Key
       * @description A semi-unique identifier for the resource.
       */
      key: string
      /** @description Alignment configuration for the plan. */
      alignment?: components['schemas']['Alignment']
      /**
       * Currency
       * @description The currency code of the plan.
       * @default USD
       */
      currency: components['schemas']['CurrencyCode']
      /**
       * Billing cadence
       * Format: duration
       * @description The default billing cadence for subscriptions using this plan.
       *     Defines how often customers are billed using ISO8601 duration format.
       *     Examples: "P1M" (monthly), "P3M" (quarterly), "P1Y" (annually).
       * @example P1M
       */
      billingCadence: string
      /**
       * Pro-rating configuration
       * @description Default pro-rating configuration for subscriptions using this plan.
       * @default {
       *       "enabled": true,
       *       "mode": "prorate_prices"
       *     }
       */
      proRatingConfig?: components['schemas']['ProRatingConfig']
      /**
       * Plan phases
       * @description The plan phase or pricing ramp allows changing a plan's rate cards over time as a subscription progresses.
       *     A phase switch occurs only at the end of a billing period, ensuring that a single subscription invoice will not include charges from different phase prices.
       */
      phases: components['schemas']['PlanPhase'][]
    }
    /**
     * @description Order by options for plans.
     * @enum {string}
     */
    PlanOrderBy: 'id' | 'key' | 'version' | 'created_at' | 'updated_at'
    /** @description Paginated response */
    PlanPaginatedResponse: {
      /**
       * @description The total number of items.
       * @example 500
       */
      totalCount: number
      /**
       * @description The page index.
       * @example 1
       */
      page: number
      /**
       * @description The maximum number of items per page.
       * @example 100
       */
      pageSize: number
      /** @description The items in the current page. */
      items: components['schemas']['Plan'][]
    }
    /** @description The plan phase or pricing ramp allows changing a plan's rate cards over time as a subscription progresses. */
    PlanPhase: {
      /**
       * Key
       * @description A semi-unique identifier for the resource.
       */
      key: string
      /**
       * Display name
       * @description Human-readable name for the resource. Between 1 and 256 characters.
       */
      name: string
      /**
       * Description
       * @description Optional description of the resource. Maximum 1024 characters.
       */
      description?: string
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata'] | null
      /**
       * Duration
       * Format: duration
       * @description The duration of the phase.
       * @example P1Y
       */
      duration: string | null
      /**
       * Rate cards
       * @description The rate cards of the plan.
       */
      rateCards: components['schemas']['RateCard'][]
    }
    /** @description References an exact plan. */
    PlanReference: {
      /**
       * @description The plan ID.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      id: string
      /** @description The plan key. */
      key: string
      /** @description The plan version. */
      version: number
    }
    /** @description References an exact plan defaulting to the current active version. */
    PlanReferenceInput: {
      /** @description The plan key. */
      key: string
      /** @description The plan version. */
      version?: number
    }
    /** @description Resource update operation model. */
    PlanReplaceUpdate: {
      /**
       * Display name
       * @description Human-readable name for the resource. Between 1 and 256 characters.
       */
      name: string
      /**
       * Description
       * @description Optional description of the resource. Maximum 1024 characters.
       */
      description?: string
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata'] | null
      /** @description Alignment configuration for the plan. */
      alignment?: components['schemas']['Alignment']
      /**
       * Billing cadence
       * Format: duration
       * @description The default billing cadence for subscriptions using this plan.
       *     Defines how often customers are billed using ISO8601 duration format.
       *     Examples: "P1M" (monthly), "P3M" (quarterly), "P1Y" (annually).
       * @example P1M
       */
      billingCadence: string
      /**
       * Pro-rating configuration
       * @description Default pro-rating configuration for subscriptions using this plan.
       * @default {
       *       "enabled": true,
       *       "mode": "prorate_prices"
       *     }
       */
      proRatingConfig?: components['schemas']['ProRatingConfig']
      /**
       * Plan phases
       * @description The plan phase or pricing ramp allows changing a plan's rate cards over time as a subscription progresses.
       *     A phase switch occurs only at the end of a billing period, ensuring that a single subscription invoice will not include charges from different phase prices.
       */
      phases: components['schemas']['PlanPhase'][]
    }
    /**
     * @description The status of a plan.
     * @enum {string}
     */
    PlanStatus: 'draft' | 'active' | 'archived' | 'scheduled'
    /** @description Change subscription based on plan. */
    PlanSubscriptionChange: {
      /** @description Timing configuration for the change, when the change should take effect.
       *     For changing a subscription, the accepted values depend on the subscription configuration. */
      timing: components['schemas']['SubscriptionTiming']
      /** @description What alignment settings the subscription should have. */
      alignment?: components['schemas']['Alignment']
      /** @description Arbitrary metadata associated with the subscription. */
      metadata?: components['schemas']['Metadata']
      /** @description The plan reference to change to. */
      plan: components['schemas']['PlanReferenceInput']
      /** @description The key of the phase to start the subscription in.
       *     If not provided, the subscription will start in the first phase of the plan. */
      startingPhase?: string
      /** @description The name of the Subscription. If not provided the plan name is used. */
      name?: string
      /** @description Description for the Subscription. */
      description?: string
    }
    /** @description Create subscription based on plan. */
    PlanSubscriptionCreate: {
      /** @description What alignment settings the subscription should have. */
      alignment?: components['schemas']['Alignment']
      /** @description Arbitrary metadata associated with the subscription. */
      metadata?: components['schemas']['Metadata']
      /** @description The plan reference to change to. */
      plan: components['schemas']['PlanReferenceInput']
      /** @description The key of the phase to start the subscription in.
       *     If not provided, the subscription will start in the first phase of the plan. */
      startingPhase?: string
      /** @description The name of the Subscription. If not provided the plan name is used. */
      name?: string
      /** @description Description for the Subscription. */
      description?: string
      /**
       * @description Timing configuration for the change, when the change should take effect.
       *     The default is immediate.
       * @default immediate
       */
      timing?: components['schemas']['SubscriptionTiming']
      /**
       * @description The ID of the customer. Provide either the key or ID. Has presedence over the key.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      customerId?: string
      /** @description The key of the customer. Provide either the key or ID. */
      customerKey?: string
      /**
       * Format: date-time
       * @description The billing anchor of the subscription. The provided date will be normalized according to the billing cadence to the nearest recurrence before start time. If not provided, the subscription start time will be used.
       * @example 2023-01-01T01:01:01.001Z
       */
      billingAnchor?: Date
    }
    /** @description A consumer portal token.
     *
     *     Validator doesn't obey required for readOnly properties
     *     See: https://github.com/stoplightio/spectral/issues/1274 */
    PortalToken: {
      /**
       * @description ULID (Universally Unique Lexicographically Sortable Identifier).
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      readonly id?: string
      /** @example customer-1 */
      subject: string
      /**
       * Format: date-time
       * @description [RFC3339](https://tools.ietf.org/html/rfc3339) formatted date-time string in UTC.
       * @example 2023-01-01T01:01:01.001Z
       */
      readonly expiresAt?: Date
      readonly expired?: boolean
      /**
       * Format: date-time
       * @description [RFC3339](https://tools.ietf.org/html/rfc3339) formatted date-time string in UTC.
       * @example 2023-01-01T01:01:01.001Z
       */
      readonly createdAt?: Date
      /**
       * @description The token is only returned at creation.
       * @example om_portal_IAnD3PpWW2A2Wr8m9jfzeHlGX8xmCXwG.y5q4S-AWqFu6qjfaFz0zQq4Ez28RsnyVwJffX5qxMvo
       */
      readonly token?: string
      /**
       * @description Optional, if defined only the specified meters will be allowed.
       * @example [
       *       "tokens_total"
       *     ]
       */
      allowedMeterSlugs?: string[]
    }
    /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
    PreconditionFailedProblemResponse: components['schemas']['UnexpectedProblemResponse']
    /**
     * @description The payment term of a flat price.
     *     One of: in_advance or in_arrears.
     * @enum {string}
     */
    PricePaymentTerm: 'in_advance' | 'in_arrears'
    /** @description A price tier.
     *     At least one price component is required in each tier. */
    PriceTier: {
      /**
       * Up to quantity
       * @description Up to and including to this quantity will be contained in the tier.
       *     If null, the tier is open-ended.
       */
      upToAmount?: components['schemas']['Numeric']
      /**
       * Flat price component
       * @description The flat price component of the tier.
       */
      flatPrice: components['schemas']['FlatPrice'] | null
      /**
       * Unit price component
       * @description The unit price component of the tier.
       */
      unitPrice: components['schemas']['UnitPrice'] | null
    }
    /** @description Configuration for pro-rating behavior. */
    ProRatingConfig: {
      /**
       * Enable pro-rating
       * @description Whether pro-rating is enabled for this plan.
       * @default true
       */
      enabled: boolean
      /**
       * Pro-rating mode
       * @description How to handle pro-rating for billing period changes.
       * @default prorate_prices
       */
      mode: components['schemas']['ProRatingMode']
    }
    /**
     * @description Pro-rating mode options for handling billing period changes.
     * @enum {string}
     */
    ProRatingMode: 'prorate_prices'
    /** @description Progress describes a progress of a task. */
    Progress: {
      /**
       * Format: uint64
       * @description Success is the number of items that succeeded
       */
      success: number
      /**
       * Format: uint64
       * @description Failed is the number of items that failed
       */
      failed: number
      /**
       * Format: uint64
       * @description The total number of items to process
       */
      total: number
      /**
       * Format: date-time
       * @description The time the progress was last updated
       * @example 2023-01-01T01:01:01.001Z
       */
      updatedAt: Date
    }
    /** @description A rate card defines the pricing and entitlement of a feature or service. */
    RateCard:
      | components['schemas']['RateCardFlatFee']
      | components['schemas']['RateCardUsageBased']
    /** @description Entitlement template of a boolean entitlement. */
    RateCardBooleanEntitlement: {
      /** @description Additional metadata for the feature. */
      metadata?: components['schemas']['Metadata']
      /**
       * @description discriminator enum property added by openapi-typescript
       * @enum {string}
       */
      type: 'boolean'
    }
    /** @description Entitlement templates are used to define the entitlements of a plan.
     *     Features are omitted from the entitlement template, as they are defined in the rate card. */
    RateCardEntitlement:
      | components['schemas']['RateCardMeteredEntitlement']
      | components['schemas']['RateCardStaticEntitlement']
      | components['schemas']['RateCardBooleanEntitlement']
    /** @description A flat fee rate card defines a one-time purchase or a recurring fee. */
    RateCardFlatFee: {
      /**
       * @description The type of the RateCard. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'flat_fee'
      /**
       * Key
       * @description A semi-unique identifier for the resource.
       */
      key: string
      /**
       * Display name
       * @description Human-readable name for the resource. Between 1 and 256 characters.
       */
      name: string
      /**
       * Description
       * @description Optional description of the resource. Maximum 1024 characters.
       */
      description?: string
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata'] | null
      /**
       * Feature key
       * @description The feature the customer is entitled to use.
       */
      featureKey?: string
      /** @description The entitlement of the rate card.
       *     Only available when featureKey is set. */
      entitlementTemplate?: components['schemas']['RateCardEntitlement']
      /**
       * Tax config
       * @description The tax config of the rate card.
       *     When undefined, the tax config of the feature or the default tax config of the plan is used.
       */
      taxConfig?: components['schemas']['TaxConfig']
      /**
       * Billing cadence
       * Format: duration
       * @description The billing cadence of the rate card.
       *     When null it means it is a one time fee.
       */
      billingCadence: string | null
      /**
       * Price
       * @description The price of the rate card.
       *     When null, the feature or service is free.
       * @example {
       *       "type": "flat",
       *       "amount": "100",
       *       "paymentTerm": "in_arrears"
       *     }
       */
      price: components['schemas']['FlatPriceWithPaymentTerm'] | null
      /**
       * Discounts
       * @description The discount of the rate card. For flat fee rate cards only percentage discounts are supported.
       *     Only available when price is set.
       */
      discounts?: components['schemas']['Discounts']
    }
    /** @description The entitlement template with a metered entitlement. */
    RateCardMeteredEntitlement: {
      /** @description Additional metadata for the feature. */
      metadata?: components['schemas']['Metadata']
      /**
       * @description discriminator enum property added by openapi-typescript
       * @enum {string}
       */
      type: 'metered'
      /**
       * Soft limit
       * @description If softLimit=true the subject can use the feature even if the entitlement is exhausted, hasAccess will always be true.
       * @default false
       */
      isSoftLimit?: boolean
      /**
       * Initial grant amount
       * Format: double
       * @description You can grant usage automatically alongside the entitlement, the example scenario would be creating a starting balance.
       *     If an amount is specified here, a grant will be created alongside the entitlement with the specified amount.
       *     That grant will have it's rollover settings configured in a way that after each reset operation, the balance will return the original amount specified here.
       *     Manually creating such a grant would mean having the "amount", "minRolloverAmount", and "maxRolloverAmount" fields all be the same.
       */
      issueAfterReset?: number
      /**
       * Issue grant after reset priority
       * Format: uint8
       * @description Defines the grant priority for the default grant.
       * @default 1
       */
      issueAfterResetPriority?: number
      /**
       * Preserve overage at reset
       * @description If true, the overage is preserved at reset. If false, the usage is reset to 0.
       * @default false
       */
      preserveOverageAtReset?: boolean
      /**
       * Usage Period
       * Format: duration
       * @description The interval of the metered entitlement.
       *     Defaults to the billing cadence of the rate card.
       */
      usagePeriod?: string
    }
    /** @description Entitlement template of a static entitlement. */
    RateCardStaticEntitlement: {
      /** @description Additional metadata for the feature. */
      metadata?: components['schemas']['Metadata']
      /**
       * @description discriminator enum property added by openapi-typescript
       * @enum {string}
       */
      type: 'static'
      /**
       * Format: json
       * @description The JSON parsable config of the entitlement. This value is also returned when checking entitlement access and it is useful for configuring fine-grained access settings to the feature, implemented in your own system. Has to be an object.
       * @example { "integrations": ["github"] }
       */
      config: string
    }
    /** @description A usage-based rate card defines a price based on usage. */
    RateCardUsageBased: {
      /**
       * @description The type of the RateCard. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'usage_based'
      /**
       * Key
       * @description A semi-unique identifier for the resource.
       */
      key: string
      /**
       * Display name
       * @description Human-readable name for the resource. Between 1 and 256 characters.
       */
      name: string
      /**
       * Description
       * @description Optional description of the resource. Maximum 1024 characters.
       */
      description?: string
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata'] | null
      /**
       * Feature key
       * @description The feature the customer is entitled to use.
       */
      featureKey?: string
      /** @description The entitlement of the rate card.
       *     Only available when featureKey is set. */
      entitlementTemplate?: components['schemas']['RateCardEntitlement']
      /**
       * Tax config
       * @description The tax config of the rate card.
       *     When undefined, the tax config of the feature or the default tax config of the plan is used.
       */
      taxConfig?: components['schemas']['TaxConfig']
      /**
       * Billing cadence
       * Format: duration
       * @description The billing cadence of the rate card.
       */
      billingCadence: string
      /** @description The price of the rate card.
       *     When null, the feature or service is free. */
      price: components['schemas']['RateCardUsageBasedPrice'] | null
      /**
       * Discounts
       * @description The discounts of the rate card.
       *
       *     Flat fee rate cards only support percentage discounts.
       */
      discounts?: components['schemas']['Discounts']
    }
    /** @description The price of the usage based rate card. */
    RateCardUsageBasedPrice:
      | components['schemas']['FlatPriceWithPaymentTerm']
      | components['schemas']['UnitPriceWithCommitments']
      | components['schemas']['TieredPriceWithCommitments']
      | components['schemas']['DynamicPriceWithCommitments']
      | components['schemas']['PackagePriceWithCommitments']
    /**
     * @description Recurring period with an interval and an anchor.
     * @example {
     *       "interval": "DAY",
     *       "intervalISO": "P1D",
     *       "anchor": "2023-01-01T01:01:01.001Z"
     *     }
     */
    RecurringPeriod: {
      /**
       * Interval
       * @description The unit of time for the interval. Heuristically maps ISO duraitons to enum values or returns the ISO duration.
       */
      interval: components['schemas']['RecurringPeriodInterval']
      /**
       * Format: duration
       * @description The unit of time for the interval in ISO8601 format.
       */
      intervalISO: string
      /**
       * Anchor time
       * Format: date-time
       * @description A date-time anchor to base the recurring period on.
       * @example 2023-01-01T01:01:01.001Z
       */
      anchor: Date
    }
    /**
     * @description Recurring period with an interval and an anchor.
     * @example {
     *       "interval": "DAY",
     *       "anchor": "2023-01-01T01:01:01.001Z"
     *     }
     */
    RecurringPeriodCreateInput: {
      /**
       * Interval
       * @description The unit of time for the interval.
       */
      interval: components['schemas']['RecurringPeriodInterval']
      /**
       * Anchor time
       * Format: date-time
       * @description A date-time anchor to base the recurring period on.
       * @example 2023-01-01T01:01:01.001Z
       */
      anchor?: Date
    }
    /** @description Period duration for the recurrence */
    RecurringPeriodInterval:
      | string
      | components['schemas']['RecurringPeriodIntervalEnum']
    /**
     * @description The unit of time for the interval.
     *     One of: `day`, `week`, `month`, or `year`.
     * @enum {string}
     */
    RecurringPeriodIntervalEnum: 'DAY' | 'WEEK' | 'MONTH' | 'YEAR'
    /**
     * @description The direction of the phase shift when a phase is removed.
     * @enum {string}
     */
    RemovePhaseShifting: 'next' | 'prev'
    /** @description Reset parameters */
    ResetEntitlementUsageInput: {
      /**
       * Format: date-time
       * @description The time at which the reset takes effect, defaults to now. The reset cannot be in the future. The provided value is truncated to the minute due to how historical meter data is stored.
       * @example 2023-01-01T01:01:01.001Z
       */
      effectiveAt?: Date
      /** @description Determines whether the usage period anchor is retained or reset to the effectiveAt time.
       *     - If true, the usage period anchor is retained.
       *     - If false, the usage period anchor is reset to the effectiveAt time. */
      retainAnchor?: boolean
      /** @description Determines whether the overage is preserved or forgiven, overriding the entitlement's default behavior.
       *     - If true, the overage is preserved.
       *     - If false, the overage is forgiven. */
      preserveOverage?: boolean
    }
    /** @description Sandbox app can be used for testing OpenMeter features.
     *
     *     The app is not creating anything in external systems, thus it is safe to use for
     *     verifying OpenMeter features. */
    SandboxApp: {
      /**
       * ID
       * @description A unique identifier for the resource.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      readonly id: string
      /**
       * Display name
       * @description Human-readable name for the resource. Between 1 and 256 characters.
       */
      name: string
      /**
       * Description
       * @description Optional description of the resource. Maximum 1024 characters.
       */
      description?: string
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata'] | null
      /**
       * Creation Time
       * Format: date-time
       * @description Timestamp of when the resource was created.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly createdAt: Date
      /**
       * Last Update Time
       * Format: date-time
       * @description Timestamp of when the resource was last updated.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly updatedAt: Date
      /**
       * Deletion Time
       * Format: date-time
       * @description Timestamp of when the resource was permanently deleted.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly deletedAt?: Date
      /** @description The marketplace listing that this installed app is based on. */
      readonly listing: components['schemas']['MarketplaceListing']
      /** @description Status of the app connection. */
      readonly status: components['schemas']['AppStatus']
      /** @description Default for the app type
       *     Only one app of each type can be default. */
      default: boolean
      /**
       * @description The app's type is Sandbox. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'sandbox'
    }
    /** @description Resource update operation model. */
    SandboxAppReplaceUpdate: {
      /**
       * Display name
       * @description Human-readable name for the resource. Between 1 and 256 characters.
       */
      name: string
      /**
       * Description
       * @description Optional description of the resource. Maximum 1024 characters.
       */
      description?: string
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata'] | null
      /** @description Default for the app type
       *     Only one app of each type can be default. */
      default: boolean
      /**
       * @description The app's type is Sandbox. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'sandbox'
    }
    /** @description Sandbox Customer App Data. */
    SandboxCustomerAppData: {
      /** @description The installed sandbox app this data belongs to. */
      readonly app?: components['schemas']['SandboxApp']
      /**
       * App ID
       * @description The app ID.
       *     If not provided, it will use the global default for the app type.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      id?: string
      /**
       * @description The app name. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'sandbox'
    }
    /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
    ServiceUnavailableProblemResponse: components['schemas']['UnexpectedProblemResponse']
    /**
     * @description The order direction.
     * @enum {string}
     */
    SortOrder: 'ASC' | 'DESC'
    /** @description The Stripe API key input.
     *     Used to authenticate with the Stripe API. */
    StripeAPIKeyInput: {
      secretAPIKey: string
    }
    /**
     * @description A installed Stripe app object.
     * @example {
     *       "id": "01G65Z755AFWAKHE12NY0CQ9FH",
     *       "type": "stripe",
     *       "name": "Stripe",
     *       "status": "ready",
     *       "default": true,
     *       "listing": {
     *         "type": "stripe",
     *         "name": "Stripe",
     *         "description": "Stripe interation allows you to collect payments with Stripe.",
     *         "capabilities": [
     *           {
     *             "type": "calculateTax",
     *             "key": "stripe_calculate_tax",
     *             "name": "Calculate Tax",
     *             "description": "Stripe Tax calculates tax portion of the invoices."
     *           },
     *           {
     *             "type": "invoiceCustomers",
     *             "key": "stripe_invoice_customers",
     *             "name": "Invoice Customers",
     *             "description": "Stripe invoices customers with due amount."
     *           },
     *           {
     *             "type": "collectPayments",
     *             "key": "stripe_collect_payments",
     *             "name": "Collect Payments",
     *             "description": "Stripe payments collects outstanding revenue with Stripe customer's default payment method."
     *           }
     *         ],
     *         "installMethods": [
     *           "with_oauth2",
     *           "with_api_key"
     *         ]
     *       },
     *       "createdAt": "2024-01-01T01:01:01.001Z",
     *       "updatedAt": "2024-01-01T01:01:01.001Z",
     *       "stripeAccountId": "acct_123456789",
     *       "livemode": true,
     *       "maskedAPIKey": "sk_live_************abc"
     *     }
     */
    StripeApp: {
      /**
       * ID
       * @description A unique identifier for the resource.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      readonly id: string
      /**
       * Display name
       * @description Human-readable name for the resource. Between 1 and 256 characters.
       */
      name: string
      /**
       * Description
       * @description Optional description of the resource. Maximum 1024 characters.
       */
      description?: string
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata'] | null
      /**
       * Creation Time
       * Format: date-time
       * @description Timestamp of when the resource was created.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly createdAt: Date
      /**
       * Last Update Time
       * Format: date-time
       * @description Timestamp of when the resource was last updated.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly updatedAt: Date
      /**
       * Deletion Time
       * Format: date-time
       * @description Timestamp of when the resource was permanently deleted.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly deletedAt?: Date
      /** @description The marketplace listing that this installed app is based on. */
      readonly listing: components['schemas']['MarketplaceListing']
      /** @description Status of the app connection. */
      readonly status: components['schemas']['AppStatus']
      /** @description Default for the app type
       *     Only one app of each type can be default. */
      default: boolean
      /**
       * @description The app's type is Stripe. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'stripe'
      /** @description The Stripe account ID. */
      readonly stripeAccountId: string
      /** @description Livemode, true if the app is in production mode. */
      readonly livemode: boolean
      /** @description The masked API key.
       *     Only shows the first 8 and last 3 characters. */
      readonly maskedAPIKey: string
    }
    /** @description Resource update operation model. */
    StripeAppReplaceUpdate: {
      /**
       * Display name
       * @description Human-readable name for the resource. Between 1 and 256 characters.
       */
      name: string
      /**
       * Description
       * @description Optional description of the resource. Maximum 1024 characters.
       */
      description?: string
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata'] | null
      /** @description Default for the app type
       *     Only one app of each type can be default. */
      default: boolean
      /**
       * @description The app's type is Stripe. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'stripe'
      /**
       * Format: password
       * @description The Stripe API key.
       */
      secretAPIKey?: string
    }
    /**
     * @description Stripe CheckoutSession.mode
     * @enum {string}
     */
    StripeCheckoutSessionMode: 'setup'
    /**
     * @description Stripe Customer App Data.
     * @example {
     *       "type": "stripe",
     *       "stripeCustomerId": "cus_xxxxxxxxxxxxxx"
     *     }
     */
    StripeCustomerAppData: {
      /**
       * App ID
       * @description The app ID.
       *     If not provided, it will use the global default for the app type.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      id?: string
      /**
       * @description The app name. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'stripe'
      /** @description The installed stripe app this data belongs to. */
      readonly app?: components['schemas']['StripeApp']
      /** @description The Stripe customer ID. */
      stripeCustomerId: string
      /** @description The Stripe default payment method ID. */
      stripeDefaultPaymentMethodId?: string
    }
    /**
     * @description Stripe Customer App Data.
     * @example {
     *       "type": "stripe",
     *       "stripeCustomerId": "cus_xxxxxxxxxxxxxx"
     *     }
     */
    StripeCustomerAppDataCreateOrUpdateItem: {
      /**
       * App ID
       * @description The app ID.
       *     If not provided, it will use the global default for the app type.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      id?: string
      /**
       * @description The app name. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'stripe'
      /** @description The Stripe customer ID. */
      stripeCustomerId: string
      /** @description The Stripe default payment method ID. */
      stripeDefaultPaymentMethodId?: string
    }
    /** @description The tax config for Stripe. */
    StripeTaxConfig: {
      /**
       * Tax code
       * @description Product tax code.
       *
       *     See: https://docs.stripe.com/tax/tax-codes
       * @example txcd_10000000
       */
      code: string
    }
    /** @description Stripe webhook event. */
    StripeWebhookEvent: {
      /** @description The event ID. */
      id: string
      /** @description The event type. */
      type: string
      /** @description Live mode. */
      livemode: boolean
      /**
       * Format: int32
       * @description The event created timestamp.
       */
      created: number
      /** @description The event data. */
      data: {
        object: unknown
      }
    }
    /** @description Stripe webhook response. */
    StripeWebhookResponse: {
      /**
       * @description ULID (Universally Unique Lexicographically Sortable Identifier).
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      namespaceId: string
      /**
       * @description ULID (Universally Unique Lexicographically Sortable Identifier).
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      appId: string
      /**
       * @description ULID (Universally Unique Lexicographically Sortable Identifier).
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      customerId?: string
      message?: string
    }
    /**
     * @description A subject is a unique identifier for a usage attribution by its key.
     *     Subjects only exist in the concept of metering.
     *     Subjects are optional to create and work as an enrichment for the subject key like displayName, metadata, etc.
     *     Subjects are useful when you are reporting usage events with your own database ID but want to enrich the subject with a human-readable name or metadata.
     *     For most use cases, a subject is equivalent to a customer.
     * @example {
     *       "id": "01G65Z755AFWAKHE12NY0CQ9FH",
     *       "key": "customer-id",
     *       "displayName": "Customer Name",
     *       "metadata": {
     *         "hubspotId": "123456"
     *       },
     *       "stripeCustomerId": "cus_JMOlctsKV8"
     *     }
     */
    Subject: {
      /**
       * @description A unique identifier for the subject.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      readonly id: string
      /**
       * @description A unique, human-readable identifier for the subject.
       *     This is typically a database ID or a customer key.
       * @example customer-db-id-123
       */
      key: string
      /**
       * @description A human-readable display name for the subject.
       * @example Customer Name
       */
      displayName?: string | null
      /**
       * @description Metadata for the subject.
       * @example {
       *       "hubspotId": "123456"
       *     }
       */
      metadata?: {
        [key: string]: unknown
      } | null
      /**
       * Format: date-time
       * @deprecated
       * @description The start of the current period for the subject.
       * @example 2023-01-01T00:00:00Z
       */
      currentPeriodStart?: Date
      /**
       * Format: date-time
       * @deprecated
       * @description The end of the current period for the subject.
       * @example 2023-02-01T00:00:00Z
       */
      currentPeriodEnd?: Date
      /**
       * @deprecated
       * @description The Stripe customer ID for the subject.
       * @example cus_JMOlctsKV8
       */
      stripeCustomerId?: string | null
    }
    /**
     * @description A subject is a unique identifier for a user or entity.
     * @example {
     *       "key": "customer-id",
     *       "displayName": "Customer Name",
     *       "metadata": {
     *         "hubspotId": "123456"
     *       },
     *       "stripeCustomerId": "cus_JMOlctsKV8"
     *     }
     */
    SubjectUpsert: {
      /**
       * @description A unique, human-readable identifier for the subject.
       *     This is typically a database ID or a customer key.
       * @example customer-db-id-123
       */
      key: string
      /**
       * @description A human-readable display name for the subject.
       * @example Customer Name
       */
      displayName?: string | null
      /**
       * @description Metadata for the subject.
       * @example {
       *       "hubspotId": "123456"
       *     }
       */
      metadata?: {
        [key: string]: unknown
      } | null
      /**
       * Format: date-time
       * @deprecated
       * @description The start of the current period for the subject.
       * @example 2023-01-01T00:00:00Z
       */
      currentPeriodStart?: Date
      /**
       * Format: date-time
       * @deprecated
       * @description The end of the current period for the subject.
       * @example 2023-02-01T00:00:00Z
       */
      currentPeriodEnd?: Date
      /**
       * @deprecated
       * @description The Stripe customer ID for the subject.
       * @example cus_JMOlctsKV8
       */
      stripeCustomerId?: string | null
    }
    /** @description Subscription is an exact subscription instance. */
    Subscription: {
      /**
       * ID
       * @description A unique identifier for the resource.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      readonly id: string
      /**
       * Display name
       * @description Human-readable name for the resource. Between 1 and 256 characters.
       */
      name: string
      /**
       * Description
       * @description Optional description of the resource. Maximum 1024 characters.
       */
      description?: string
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata'] | null
      /**
       * Creation Time
       * Format: date-time
       * @description Timestamp of when the resource was created.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly createdAt: Date
      /**
       * Last Update Time
       * Format: date-time
       * @description Timestamp of when the resource was last updated.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly updatedAt: Date
      /**
       * Deletion Time
       * Format: date-time
       * @description Timestamp of when the resource was permanently deleted.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly deletedAt?: Date
      /**
       * Format: date-time
       * @description The cadence start of the resource.
       * @example 2023-01-01T01:01:01.001Z
       */
      activeFrom: Date
      /**
       * Format: date-time
       * @description The cadence end of the resource.
       * @example 2023-01-01T01:01:01.001Z
       */
      activeTo?: Date
      /** @description Alignment configuration for the plan. */
      alignment?: components['schemas']['Alignment']
      /** @description The status of the subscription. */
      readonly status: components['schemas']['SubscriptionStatus']
      /**
       * @description The customer ID of the subscription.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      customerId: string
      /** @description The plan of the subscription. */
      plan?: components['schemas']['PlanReference']
      /**
       * Currency
       * @description The currency code of the subscription.
       *     Will be revised once we add multi currency support.
       * @default USD
       */
      currency: components['schemas']['CurrencyCode']
      /**
       * Billing cadence
       * Format: duration
       * @description The billing cadence for the subscriptions.
       *     Defines how often customers are billed using ISO8601 duration format.
       *     Examples: "P1M" (monthly), "P3M" (quarterly), "P1Y" (annually).
       * @example P1M
       */
      readonly billingCadence: string
      /**
       * Pro-rating configuration
       * @description The pro-rating configuration for the subscriptions.
       * @default {
       *       "enabled": true,
       *       "mode": "prorate_prices"
       *     }
       */
      readonly proRatingConfig?: components['schemas']['ProRatingConfig']
      /**
       * Billing anchor
       * Format: date-time
       * @description The normalizedbilling anchor of the subscription.
       * @example 2023-01-01T01:01:01.001Z
       */
      readonly billingAnchor: Date
    }
    /** @description A subscription add-on, represents concrete instances of an add-on for a given subscription. */
    SubscriptionAddon: {
      /**
       * ID
       * @description A unique identifier for the resource.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      readonly id: string
      /**
       * Display name
       * @description Human-readable name for the resource. Between 1 and 256 characters.
       */
      name: string
      /**
       * Description
       * @description Optional description of the resource. Maximum 1024 characters.
       */
      description?: string
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata'] | null
      /**
       * Creation Time
       * Format: date-time
       * @description Timestamp of when the resource was created.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly createdAt: Date
      /**
       * Last Update Time
       * Format: date-time
       * @description Timestamp of when the resource was last updated.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly updatedAt: Date
      /**
       * Deletion Time
       * Format: date-time
       * @description Timestamp of when the resource was permanently deleted.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly deletedAt?: Date
      /**
       * Format: date-time
       * @description The cadence start of the resource.
       * @example 2023-01-01T01:01:01.001Z
       */
      readonly activeFrom: Date
      /**
       * Format: date-time
       * @description The cadence end of the resource.
       * @example 2023-01-01T01:01:01.001Z
       */
      readonly activeTo?: Date
      /**
       * Addon
       * @description Partially populated add-on properties.
       */
      addon: {
        /**
         * ID
         * @description The ID of the add-on.
         * @example 01G65Z755AFWAKHE12NY0CQ9FH
         */
        id: string
        /**
         * Key
         * @description A semi-unique identifier for the resource.
         */
        readonly key: string
        /**
         * Version
         * @description The version of the Add-on which templates this instance.
         * @default 1
         */
        readonly version: number
        /**
         * InstanceType
         * @description The instance type of the add-on.
         */
        readonly instanceType: components['schemas']['AddonInstanceType']
      }
      /**
       * QuantityAt
       * Format: date-time
       * @description For which point in time the quantity was resolved to.
       * @example 2025-01-05T00:00:00Z
       */
      readonly quantityAt: Date
      /**
       * Quantity
       * @description The quantity of the add-on. Always 1 for single instance add-ons.
       * @example 1
       */
      quantity: number
      /**
       * Timeline
       * @description The timeline of the add-on. The returned periods are sorted and continuous.
       * @example [
       *       {
       *         "quantity": 1,
       *         "activeFrom": "2025-01-01T00:00:00Z",
       *         "activeTo": "2025-01-02T00:00:00Z"
       *       },
       *       {
       *         "quantity": 0,
       *         "activeFrom": "2025-01-02T00:00:00Z",
       *         "activeTo": "2025-01-03T00:00:00Z"
       *       },
       *       {
       *         "quantity": 1,
       *         "activeFrom": "2025-01-03T00:00:00Z"
       *       }
       *     ]
       */
      readonly timeline: components['schemas']['SubscriptionAddonTimelineSegment'][]
      /**
       * SubscriptionID
       * @description The ID of the subscription.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      readonly subscriptionId: string
      /**
       * Rate cards
       * @description The rate cards of the add-on.
       */
      readonly rateCards: components['schemas']['SubscriptionAddonRateCard'][]
    }
    /** @description A subscription add-on create body. */
    SubscriptionAddonCreate: {
      /**
       * Display name
       * @description Human-readable name for the resource. Between 1 and 256 characters.
       */
      name: string
      /**
       * Description
       * @description Optional description of the resource. Maximum 1024 characters.
       */
      description?: string
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata'] | null
      /**
       * Quantity
       * @description The quantity of the add-on. Always 1 for single instance add-ons.
       * @example 1
       */
      quantity: number
      /**
       * Timing
       * @description The timing of the operation. After the create or update, a new entry will be created in the timeline.
       */
      timing: components['schemas']['SubscriptionTiming']
      /**
       * Addon
       * @description The add-on to create.
       */
      addon: {
        /**
         * @description The ID of the add-on.
         * @example 01G65Z755AFWAKHE12NY0CQ9FH
         */
        id: string
      }
    }
    /** @description A rate card for a subscription add-on. */
    SubscriptionAddonRateCard: {
      /**
       * Rate card
       * @description The rate card.
       */
      rateCard: components['schemas']['RateCard']
      /**
       * Affected subscription item IDs
       * @description The IDs of the subscription items that this rate card belongs to.
       */
      readonly affectedSubscriptionItemIds: string[]
    }
    /** @description A subscription add-on event. */
    SubscriptionAddonTimelineSegment: {
      /**
       * Format: date-time
       * @description The cadence start of the resource.
       * @example 2023-01-01T01:01:01.001Z
       */
      activeFrom: Date
      /**
       * Format: date-time
       * @description The cadence end of the resource.
       * @example 2023-01-01T01:01:01.001Z
       */
      activeTo?: Date
      /**
       * Quantity
       * @description The quantity of the add-on for the given period.
       * @example 1
       */
      readonly quantity: number
    }
    /** @description Resource create or update operation model. */
    SubscriptionAddonUpdate: {
      /**
       * Display name
       * @description Human-readable name for the resource. Between 1 and 256 characters.
       */
      name?: string
      /**
       * Description
       * @description Optional description of the resource. Maximum 1024 characters.
       */
      description?: string
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata'] | null
      /**
       * Quantity
       * @description The quantity of the add-on. Always 1 for single instance add-ons.
       * @example 1
       */
      quantity?: number
      /**
       * Timing
       * @description The timing of the operation. After the create or update, a new entry will be created in the timeline.
       */
      timing?: components['schemas']['SubscriptionTiming']
    }
    /** @description Alignment details enriched with the current billing period. */
    SubscriptionAlignment: {
      /** @description Whether all Billable items and RateCards must align.
       *     Alignment means the Price's BillingCadence must align for both duration and anchor time. */
      billablesMustAlign?: boolean
      /** @description The current billing period. Only has value if the subscription is aligned and active. */
      currentAlignedBillingPeriod?: components['schemas']['Period']
    }
    /** @description Change a subscription. */
    SubscriptionChange:
      | components['schemas']['PlanSubscriptionChange']
      | components['schemas']['CustomSubscriptionChange']
    /** @description Response body for subscription change. */
    SubscriptionChangeResponseBody: {
      /**
       * Current subscription
       * @description The current subscription before the change.
       */
      current: components['schemas']['Subscription']
      /**
       * The subscription it will be changed to
       * @description The new state of the subscription after the change.
       */
      next: components['schemas']['SubscriptionExpanded']
    }
    /** @description Create a subscription. */
    SubscriptionCreate:
      | components['schemas']['PlanSubscriptionCreate']
      | components['schemas']['CustomSubscriptionCreate']
    /** @description Subscription edit input. */
    SubscriptionEdit: {
      /** @description Batch processing commands for manipulating running subscriptions.
       *     The key format is `/phases/{phaseKey}` or `/phases/{phaseKey}/items/{itemKey}`. */
      customizations: components['schemas']['SubscriptionEditOperation'][]
      /** @description Whether the billing period should be restarted.Timing configuration to allow for the changes to take effect at different times. */
      timing?: components['schemas']['SubscriptionTiming']
    }
    /** @description The operation to be performed on the subscription. */
    SubscriptionEditOperation:
      | components['schemas']['EditSubscriptionAddItem']
      | components['schemas']['EditSubscriptionRemoveItem']
      | components['schemas']['EditSubscriptionAddPhase']
      | components['schemas']['EditSubscriptionRemovePhase']
      | components['schemas']['EditSubscriptionStretchPhase']
      | components['schemas']['EditSubscriptionUnscheduleEdit']
    /** @description Expanded subscription */
    SubscriptionExpanded: {
      /**
       * ID
       * @description A unique identifier for the resource.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      readonly id: string
      /**
       * Display name
       * @description Human-readable name for the resource. Between 1 and 256 characters.
       */
      name: string
      /**
       * Description
       * @description Optional description of the resource. Maximum 1024 characters.
       */
      description?: string
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata'] | null
      /**
       * Creation Time
       * Format: date-time
       * @description Timestamp of when the resource was created.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly createdAt: Date
      /**
       * Last Update Time
       * Format: date-time
       * @description Timestamp of when the resource was last updated.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly updatedAt: Date
      /**
       * Deletion Time
       * Format: date-time
       * @description Timestamp of when the resource was permanently deleted.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly deletedAt?: Date
      /**
       * Format: date-time
       * @description The cadence start of the resource.
       * @example 2023-01-01T01:01:01.001Z
       */
      activeFrom: Date
      /**
       * Format: date-time
       * @description The cadence end of the resource.
       * @example 2023-01-01T01:01:01.001Z
       */
      activeTo?: Date
      /** @description The status of the subscription. */
      readonly status: components['schemas']['SubscriptionStatus']
      /**
       * @description The customer ID of the subscription.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      customerId: string
      /** @description The plan of the subscription. */
      plan?: components['schemas']['PlanReference']
      /**
       * Currency
       * @description The currency code of the subscription.
       *     Will be revised once we add multi currency support.
       * @default USD
       */
      currency: components['schemas']['CurrencyCode']
      /**
       * Billing cadence
       * Format: duration
       * @description The billing cadence for the subscriptions.
       *     Defines how often customers are billed using ISO8601 duration format.
       *     Examples: "P1M" (monthly), "P3M" (quarterly), "P1Y" (annually).
       * @example P1M
       */
      readonly billingCadence: string
      /**
       * Pro-rating configuration
       * @description The pro-rating configuration for the subscriptions.
       * @default {
       *       "enabled": true,
       *       "mode": "prorate_prices"
       *     }
       */
      readonly proRatingConfig?: components['schemas']['ProRatingConfig']
      /**
       * Billing anchor
       * Format: date-time
       * @description The normalizedbilling anchor of the subscription.
       * @example 2023-01-01T01:01:01.001Z
       */
      readonly billingAnchor: Date
      /** @description Alignment details enriched with the current billing period. */
      alignment?: components['schemas']['SubscriptionAlignment']
      /** @description The phases of the subscription. */
      phases: components['schemas']['SubscriptionPhaseExpanded'][]
    }
    /** @description The actual contents of the Subscription, what the user gets, what they pay, etc... */
    SubscriptionItem: {
      /**
       * ID
       * @description A unique identifier for the resource.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      readonly id: string
      /**
       * Display name
       * @description Human-readable name for the resource. Between 1 and 256 characters.
       */
      name: string
      /**
       * Description
       * @description Optional description of the resource. Maximum 1024 characters.
       */
      description?: string
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata'] | null
      /**
       * Creation Time
       * Format: date-time
       * @description Timestamp of when the resource was created.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly createdAt: Date
      /**
       * Last Update Time
       * Format: date-time
       * @description Timestamp of when the resource was last updated.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly updatedAt: Date
      /**
       * Deletion Time
       * Format: date-time
       * @description Timestamp of when the resource was permanently deleted.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly deletedAt?: Date
      /**
       * Format: date-time
       * @description The cadence start of the resource.
       * @example 2023-01-01T01:01:01.001Z
       */
      activeFrom: Date
      /**
       * Format: date-time
       * @description The cadence end of the resource.
       * @example 2023-01-01T01:01:01.001Z
       */
      activeTo?: Date
      /** @description The identifier of the RateCard.
       *     SubscriptionItem/RateCard can be identified, it has a reference:
       *
       *     1. If a Feature is associated with the SubscriptionItem, it is identified by the Feature
       *     1.1 It can be an ID reference, for an exact version of the Feature (Features can change across versions)
       *     1.2 It can be a Key reference, which always refers to the latest (active or inactive) version of a Feature
       *
       *     2. If a Feature is not associated with the SubscriptionItem, it is referenced by the Price
       *
       *     We say “referenced by the Price” regardless of how a price itself is referenced, it colloquially makes sense to say “paying the same price for the same thing”. In practice this should be derived from what's printed on the invoice line-item. */
      key: string
      /** @description The feature's key (if present). */
      featureKey?: string
      /**
       * Billing cadence
       * Format: duration
       * @description The billing cadence of the rate card.
       *     When null, the rate card is a one-time purchase.
       */
      billingCadence: string | null
      /**
       * Price
       * @description The price of the rate card.
       *     When null, the feature or service is free.
       * @example {}
       */
      price: components['schemas']['RateCardUsageBasedPrice'] | null
      /**
       * Discounts
       * @description The discounts applied to the rate card.
       */
      discounts?: components['schemas']['Discounts']
      /** @description Describes what access is gained via the SubscriptionItem */
      included?: components['schemas']['SubscriptionItemIncluded']
      /**
       * Tax config
       * @description The tax config of the Subscription Item.
       *     When undefined, the tax config of the feature or the default tax config of the plan is used.
       */
      taxConfig?: components['schemas']['TaxConfig']
    }
    /** @description Included contents like Entitlement, or the Feature. */
    SubscriptionItemIncluded: {
      /** @description The feature the customer is entitled to use. */
      feature: components['schemas']['Feature']
      /** @description The entitlement of the Subscription Item. */
      entitlement?: components['schemas']['Entitlement']
    }
    /** @description Paginated response */
    SubscriptionPaginatedResponse: {
      /**
       * @description The total number of items.
       * @example 500
       */
      totalCount: number
      /**
       * @description The page index.
       * @example 1
       */
      page: number
      /**
       * @description The maximum number of items per page.
       * @example 100
       */
      pageSize: number
      /** @description The items in the current page. */
      items: components['schemas']['Subscription'][]
    }
    /** @description Subscription phase create input. */
    SubscriptionPhaseCreate: {
      /**
       * Start after
       * Format: duration
       * @description Interval after the subscription starts to transition to the phase.
       *     When null, the phase starts immediately after the subscription starts.
       * @example P1Y
       */
      startAfter: string | null
      /**
       * Duration
       * Format: duration
       * @description The intended duration of the new phase.
       *     Duration is required when the phase will not be the last phase.
       * @example P1M
       */
      duration?: string
      /**
       * Discounts
       * @description The discounts on the plan.
       */
      discounts?: components['schemas']['Discounts']
      /** @description A locally unique identifier for the phase. */
      key: string
      /** @description The name of the phase. */
      name: string
      /** @description The description of the phase. */
      description?: string
    }
    /** @description Expanded subscription phase */
    SubscriptionPhaseExpanded: {
      /**
       * ID
       * @description A unique identifier for the resource.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      readonly id: string
      /**
       * Display name
       * @description Human-readable name for the resource. Between 1 and 256 characters.
       */
      name: string
      /**
       * Description
       * @description Optional description of the resource. Maximum 1024 characters.
       */
      description?: string
      /**
       * Metadata
       * @description Additional metadata for the resource.
       */
      metadata?: components['schemas']['Metadata'] | null
      /**
       * Creation Time
       * Format: date-time
       * @description Timestamp of when the resource was created.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly createdAt: Date
      /**
       * Last Update Time
       * Format: date-time
       * @description Timestamp of when the resource was last updated.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly updatedAt: Date
      /**
       * Deletion Time
       * Format: date-time
       * @description Timestamp of when the resource was permanently deleted.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly deletedAt?: Date
      /** @description A locally unique identifier for the resource. */
      key: string
      /**
       * Discounts
       * @description The discounts on the plan.
       */
      discounts?: components['schemas']['Discounts']
      /**
       * Format: date-time
       * @description The time from which the phase is active.
       * @example 2023-01-01T01:01:01.001Z
       */
      activeFrom: Date
      /**
       * Format: date-time
       * @description The until which the Phase is active.
       * @example 2023-01-01T01:01:01.001Z
       */
      activeTo?: Date
      /** @description The items of the phase. The structure is flattened to better conform to the Plan API.
       *     The timelines are flattened according to the following rules:
       *     - for the current phase, the `items` contains only the active item for each key
       *     - for past phases, the `items` contains only the last item for each key
       *     - for future phases, the `items` contains only the first version of the item for each key */
      items: components['schemas']['SubscriptionItem'][]
      /** @description Includes all versions of the items on each key, including all edits, scheduled changes, etc... */
      itemTimelines: {
        [key: string]: components['schemas']['SubscriptionItem'][]
      }
    }
    /**
     * @description Subscription status.
     * @enum {string}
     */
    SubscriptionStatus: 'active' | 'inactive' | 'canceled' | 'scheduled'
    /** @description Subscription edit timing defined when the changes should take effect.
     *     If the provided configuration is not supported by the subscription, an error will be returned. */
    SubscriptionTiming: components['schemas']['SubscriptionTimingEnum'] | Date
    /**
     * @description Subscription edit timing.
     *     When immediate, the requested changes take effect immediately.
     *     When nextBillingCycle, the requested changes take effect at the next billing cycle.
     * @enum {string}
     */
    SubscriptionTimingEnum: 'immediate' | 'next_billing_cycle'
    /**
     * @description Tax behavior.
     *
     *     This enum is used to specify whether tax is included in the price or excluded from the price.
     * @enum {string}
     */
    TaxBehavior: 'inclusive' | 'exclusive'
    /** @description Set of provider specific tax configs. */
    TaxConfig: {
      /**
       * Tax behavior
       * @description Tax behavior.
       *
       *     If not specified the billing profile is used to determine the tax behavior.
       *     If not specified in the billing profile, the provider's default behavior is used.
       */
      behavior?: components['schemas']['TaxBehavior']
      /**
       * Stripe tax config
       * @description Stripe tax config.
       */
      stripe?: components['schemas']['StripeTaxConfig']
      /**
       * Custom invoicing tax config
       * @description Custom invoicing tax config.
       */
      customInvoicing?: components['schemas']['CustomInvoicingTaxConfig']
    }
    /**
     * @description The mode of the tiered price.
     * @enum {string}
     */
    TieredPriceMode: 'volume' | 'graduated'
    /** @description Tiered price with spend commitments. */
    TieredPriceWithCommitments: {
      /**
       * @description The type of the price.
       *
       *     One of: flat, unit, or tiered. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'tiered'
      /**
       * Mode
       * @description Defines if the tiering mode is volume-based or graduated:
       *     - In `volume`-based tiering, the maximum quantity within a period determines the per unit price.
       *     - In `graduated` tiering, pricing can change as the quantity grows.
       */
      mode: components['schemas']['TieredPriceMode']
      /**
       * Tiers
       * @description The tiers of the tiered price.
       *     At least one price component is required in each tier.
       */
      tiers: components['schemas']['PriceTier'][]
      /**
       * Minimum amount
       * @description The customer is committed to spend at least the amount.
       */
      minimumAmount?: components['schemas']['Numeric']
      /**
       * Maximum amount
       * @description The customer is limited to spend at most the amount.
       */
      maximumAmount?: components['schemas']['Numeric']
    }
    /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
    UnauthorizedProblemResponse: components['schemas']['UnexpectedProblemResponse']
    /** @description A Problem Details object (RFC 7807).
     *     Additional properties specific to the problem type may be present. */
    UnexpectedProblemResponse: {
      /**
       * Format: uri
       * @description Type contains a URI that identifies the problem type.
       * @default about:blank
       * @example about:blank
       */
      type: string
      /**
       * @description A a short, human-readable summary of the problem type.
       * @example Bad Request
       */
      title: string
      /**
       * Format: int16
       * @description The HTTP status code generated by the origin server for this occurrence of the problem.
       * @example 400
       */
      status?: number
      /**
       * @description A human-readable explanation specific to this occurrence of the problem.
       * @example The request body must be a JSON object.
       */
      detail: string
      /**
       * Format: uri
       * @description A URI reference that identifies the specific occurrence of the problem.
       * @example urn:request:local/JMOlctsKV8-000001
       */
      instance: string
    } & {
      [key: string]: string | number
    }
    /** @description Unit price. */
    UnitPrice: {
      /**
       * @description The type of the price.
       * @enum {string}
       */
      type: 'unit'
      /** @description The amount of the unit price. */
      amount: components['schemas']['Numeric']
    }
    /** @description Unit price with spend commitments. */
    UnitPriceWithCommitments: {
      /**
       * @description The type of the price. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'unit'
      /** @description The amount of the unit price. */
      amount: components['schemas']['Numeric']
      /**
       * Minimum amount
       * @description The customer is committed to spend at least the amount.
       */
      minimumAmount?: components['schemas']['Numeric']
      /**
       * Maximum amount
       * @description The customer is limited to spend at most the amount.
       */
      maximumAmount?: components['schemas']['Numeric']
    }
    /** @description Validation errors providing details about compatibility issues between a plan and its add-on. */
    ValidationError: {
      /**
       * @description The path to the field.
       * @example addons/pro/ratecards/token/featureKey
       */
      readonly field: string
      /**
       * @description The machine readable description of the error.
       * @example invalid_feature_key
       */
      readonly code: string
      /**
       * @description The human readable description of the error.
       * @example not found feature by key
       */
      readonly message: string
      /** @description Additional attributes. */
      readonly attributes?: components['schemas']['Annotations']
    }
    /** @description ValidationIssue captures any validation issues related to the invoice.
     *
     *     Issues with severity "critical" will prevent the invoice from being issued. */
    ValidationIssue: {
      /**
       * Creation Time
       * Format: date-time
       * @description Timestamp of when the resource was created.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly createdAt: Date
      /**
       * Last Update Time
       * Format: date-time
       * @description Timestamp of when the resource was last updated.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly updatedAt: Date
      /**
       * Deletion Time
       * Format: date-time
       * @description Timestamp of when the resource was permanently deleted.
       * @example 2024-01-01T01:01:01.001Z
       */
      readonly deletedAt?: Date
      /**
       * @description ID of the charge or discount.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      readonly id: string
      /** @description The severity of the issue. */
      readonly severity: components['schemas']['ValidationIssueSeverity']
      /** @description The field that the issue is related to, if available in JSON path format. */
      readonly field?: string
      /** @description Machine indentifiable code for the issue, if available. */
      readonly code?: string
      /** @description Component reporting the issue. */
      readonly component: string
      /** @description A human-readable description of the issue. */
      readonly message: string
      /** @description Additional context for the issue. */
      readonly metadata?: components['schemas']['Metadata']
    }
    /**
     * @description ValidationIssueSeverity describes the severity of a validation issue.
     *
     *     Issues with severity "critical" will prevent the invoice from being issued.
     * @enum {string}
     */
    ValidationIssueSeverity: 'critical' | 'warning'
    /** @description InvoiceVoidAction describes how to handle the voided line items. */
    VoidInvoiceActionCreate: {
      /** @description How much of the total line items to be voided? (e.g. 100% means all charges are voided) */
      percentage: components['schemas']['Percentage']
      /** @description The action to take on the line items. */
      action: components['schemas']['VoidInvoiceLineActionCreate']
    }
    /** @description InvoiceVoidAction describes how to handle the voided line items. */
    VoidInvoiceActionCreateItem: {
      /** @description How much of the total line items to be voided? (e.g. 100% means all charges are voided) */
      percentage: components['schemas']['Percentage']
      /** @description The action to take on the line items. */
      action: components['schemas']['VoidInvoiceLineActionCreateItem']
    }
    /** @description Request to void an invoice */
    VoidInvoiceActionInput: {
      /** @description The action to take on the voided line items. */
      action: components['schemas']['VoidInvoiceActionCreate']
      /** @description The reason for voiding the invoice. */
      reason: string
      /** @description Per line item overrides for the action.
       *
       *     If not specified, the `action` will be applied to all line items. */
      overrides?:
        | components['schemas']['VoidInvoiceActionLineOverride'][]
        | null
    }
    /** @description VoidInvoiceLineOverride describes how to handle a specific line item in the invoice when voiding. */
    VoidInvoiceActionLineOverride: {
      /**
       * @description The line item ID to override.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      lineId: string
      /** @description The action to take on the line item. */
      action: components['schemas']['VoidInvoiceActionCreateItem']
    }
    /** @description VoidInvoiceLineAction describes how to handle a specific line item in the invoice when voiding. */
    VoidInvoiceLineActionCreate:
      | components['schemas']['VoidInvoiceLineDiscardAction']
      | components['schemas']['VoidInvoiceLinePendingActionCreate']
    /** @description VoidInvoiceLineAction describes how to handle a specific line item in the invoice when voiding. */
    VoidInvoiceLineActionCreateItem:
      | components['schemas']['VoidInvoiceLineDiscardAction']
      | components['schemas']['VoidInvoiceLinePendingActionCreateItem']
    /** @description VoidInvoiceLineDiscardAction describes how to handle the voidied line item in the invoice. */
    VoidInvoiceLineDiscardAction: {
      /**
       * @description The action to take on the line item. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'discard'
    }
    /** @description VoidInvoiceLinePendingAction describes how to handle the voidied line item in the invoice. */
    VoidInvoiceLinePendingActionCreate: {
      /**
       * @description The action to take on the line item. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'pending'
      /**
       * Format: date-time
       * @description The time at which the line item should be invoiced again.
       *
       *     If not provided, the line item will be re-invoiced now.
       * @example 2023-01-01T01:01:01.001Z
       */
      nextInvoiceAt?: Date
    }
    /** @description VoidInvoiceLinePendingAction describes how to handle the voidied line item in the invoice. */
    VoidInvoiceLinePendingActionCreateItem: {
      /**
       * @description The action to take on the line item. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'pending'
      /**
       * Format: date-time
       * @description The time at which the line item should be invoiced again.
       *
       *     If not provided, the line item will be re-invoiced now.
       * @example 2023-01-01T01:01:01.001Z
       */
      nextInvoiceAt?: Date
    }
    /**
     * @description Aggregation window size.
     * @enum {string}
     */
    WindowSize: 'MINUTE' | 'HOUR' | 'DAY'
    /** @description The windowed balance history. */
    WindowedBalanceHistory: {
      /** @description The windowed balance history.
       *     - It only returns rows for windows where there was usage.
       *     - The windows are inclusive at their start and exclusive at their end.
       *     - The last window may be smaller than the window size and is inclusive at both ends. */
      windowedHistory: components['schemas']['BalanceHistoryWindow'][]
      /** @description Grant burndown history. */
      burndownHistory: components['schemas']['GrantBurnDownHistorySegment'][]
    }
  }
  responses: never
  parameters: {
    /** @description The order direction. */
    'AddonOrderByOrdering.order': components['schemas']['SortOrder']
    /** @description The order by field. */
    'AddonOrderByOrdering.orderBy': components['schemas']['AddonOrderBy']
    /** @description The order direction. */
    'BillingProfileCustomerOverrideOrderByOrdering.order': components['schemas']['SortOrder']
    /** @description The order by field. */
    'BillingProfileCustomerOverrideOrderByOrdering.orderBy': components['schemas']['BillingProfileCustomerOverrideOrderBy']
    /** @description Filter by billing profile. */
    'BillingProfileListCustomerOverridesParams.billingProfile': string[]
    /** @description Filter by customer id. */
    'BillingProfileListCustomerOverridesParams.customerId': string[]
    /** @description Filter by customer key */
    'BillingProfileListCustomerOverridesParams.customerKey': string
    /** @description Filter by customer name. */
    'BillingProfileListCustomerOverridesParams.customerName': string
    /** @description Filter by customer primary email */
    'BillingProfileListCustomerOverridesParams.customerPrimaryEmail': string
    /** @description Expand the response with additional details. */
    'BillingProfileListCustomerOverridesParams.expand': components['schemas']['BillingProfileCustomerOverrideExpand'][]
    /** @description Include customers without customer overrides.
     *
     *     If set to false only the customers specifically associated with a billing profile will be returned.
     *
     *     If set to true, in case of the default billing profile, all customers will be returned. */
    'BillingProfileListCustomerOverridesParams.includeAllCustomers': boolean
    /** @description The order direction. */
    'BillingProfileOrderByOrdering.order': components['schemas']['SortOrder']
    /** @description The order by field. */
    'BillingProfileOrderByOrdering.orderBy': components['schemas']['BillingProfileOrderBy']
    /** @description The cursor after which to start the pagination. */
    'CursorPagination.cursor': string
    /** @description The limit of the pagination. */
    'CursorPagination.limit': number
    /** @description The order direction. */
    'CustomerOrderByOrdering.order': components['schemas']['SortOrder']
    /** @description The order by field. */
    'CustomerOrderByOrdering.orderBy': components['schemas']['CustomerOrderBy']
    /** @description The order direction. */
    'EntitlementOrderByOrdering.order': components['schemas']['SortOrder']
    /** @description The order by field. */
    'EntitlementOrderByOrdering.orderBy': components['schemas']['EntitlementOrderBy']
    /** @description The order direction. */
    'FeatureOrderByOrdering.order': components['schemas']['SortOrder']
    /** @description The order by field. */
    'FeatureOrderByOrdering.orderBy': components['schemas']['FeatureOrderBy']
    /** @description The order direction. */
    'GrantOrderByOrdering.order': components['schemas']['SortOrder']
    /** @description The order by field. */
    'GrantOrderByOrdering.orderBy': components['schemas']['GrantOrderBy']
    /** @description Filter by invoice created time.
     *     Inclusive. */
    'InvoiceListParams.createdAfter': Date | string
    /** @description Filter by invoice created time.
     *     Inclusive. */
    'InvoiceListParams.createdBefore': Date | string
    /** @description Filter by customer ID */
    'InvoiceListParams.customers': string[]
    /** @description What parts of the list output to expand in listings */
    'InvoiceListParams.expand': components['schemas']['InvoiceExpand'][]
    /** @description Filter by invoice extended statuses */
    'InvoiceListParams.extendedStatuses': string[]
    /** @description Include deleted invoices */
    'InvoiceListParams.includeDeleted': boolean
    /** @description Filter by invoice issued time.
     *     Inclusive. */
    'InvoiceListParams.issuedAfter': Date | string
    /** @description Filter by invoice issued time.
     *     Inclusive. */
    'InvoiceListParams.issuedBefore': Date | string
    /** @description Filter by period start time.
     *     Inclusive. */
    'InvoiceListParams.periodStartAfter': Date | string
    /** @description Filter by period start time.
     *     Inclusive. */
    'InvoiceListParams.periodStartBefore': Date | string
    /** @description Filter by the invoice status. */
    'InvoiceListParams.statuses': components['schemas']['InvoiceStatus'][]
    /** @description The order direction. */
    'InvoiceOrderByOrdering.order': components['schemas']['SortOrder']
    /** @description The order by field. */
    'InvoiceOrderByOrdering.orderBy': components['schemas']['InvoiceOrderBy']
    /** @description Number of items to return.
     *
     *     Default is 100. */
    'LimitOffset.limit': number
    /** @description Number of items to skip.
     *
     *     Default is 0. */
    'LimitOffset.offset': number
    /** @description The type of the app to install. */
    'MarketplaceApiKeyInstallRequest.type': components['schemas']['AppType']
    /** @description The type of the app to install. */
    'MarketplaceInstallRequest.type': components['schemas']['AppType']
    /** @description The type of the app to install. */
    'MarketplaceOAuth2InstallAuthorizeRequest.type': components['schemas']['AppType']
    /** @description The order direction. */
    'MeterOrderByOrdering.order': components['schemas']['SortOrder']
    /** @description The order by field. */
    'MeterOrderByOrdering.orderBy': components['schemas']['MeterOrderBy']
    /** @description Client ID
     *     Useful to track progress of a query. */
    'MeterQuery.clientId': string
    /** @description Simple filter for group bys with exact match.
     *
     *     For example: ?filterGroupBy[vendor]=openai&filterGroupBy[model]=gpt-4-turbo */
    'MeterQuery.filterGroupBy': {
      [key: string]: string
    }
    /** @description Start date-time in RFC 3339 format.
     *
     *     Inclusive.
     *
     *     For example: ?from=2025-01-01T00%3A00%3A00.000Z */
    'MeterQuery.from': Date | string
    /** @description If not specified a single aggregate will be returned for each subject and time window.
     *     `subject` is a reserved group by value.
     *
     *     For example: ?groupBy=subject&groupBy=model */
    'MeterQuery.groupBy': string[]
    /** @description Filtering by multiple subjects.
     *
     *     For example: ?subject=customer-1&subject=customer-2 */
    'MeterQuery.subject': string[]
    /** @description End date-time in RFC 3339 format.
     *
     *     Inclusive.
     *
     *     For example: ?to=2025-02-01T00%3A00%3A00.000Z */
    'MeterQuery.to': Date | string
    /** @description If not specified, a single usage aggregate will be returned for the entirety of the specified period for each subject and group.
     *
     *     For example: ?windowSize=DAY */
    'MeterQuery.windowSize': components['schemas']['WindowSize']
    /** @description The value is the name of the time zone as defined in the IANA Time Zone Database (http://www.iana.org/time-zones).
     *     If not specified, the UTC timezone will be used.
     *
     *     For example: ?windowTimeZone=UTC */
    'MeterQuery.windowTimeZone': string
    /** @description The order direction. */
    'NotificationChannelOrderByOrdering.order': components['schemas']['SortOrder']
    /** @description The order by field. */
    'NotificationChannelOrderByOrdering.orderBy': components['schemas']['NotificationChannelOrderBy']
    /** @description The order direction. */
    'NotificationEventOrderByOrdering.order': components['schemas']['SortOrder']
    /** @description The order by field. */
    'NotificationEventOrderByOrdering.orderBy': components['schemas']['NotificationEventOrderBy']
    /** @description The order direction. */
    'NotificationRuleOrderByOrdering.order': components['schemas']['SortOrder']
    /** @description The order by field. */
    'NotificationRuleOrderByOrdering.orderBy': components['schemas']['NotificationRuleOrderBy']
    /** @description Error code.
     *     Required with the error response. */
    'OAuth2AuthorizationCodeGrantErrorParams.error': components['schemas']['OAuth2AuthorizationCodeGrantErrorType']
    /** @description Optional human-readable text providing additional information,
     *     used to assist the client developer in understanding the error that occurred. */
    'OAuth2AuthorizationCodeGrantErrorParams.error_description': string
    /** @description Optional uri identifying a human-readable web page with
     *     information about the error, used to provide the client
     *     developer with additional information about the error */
    'OAuth2AuthorizationCodeGrantErrorParams.error_uri': string
    /** @description Authorization code which the client will later exchange for an access token.
     *     Required with the success response. */
    'OAuth2AuthorizationCodeGrantSuccessParams.code': string
    /** @description Required if the "state" parameter was present in the client authorization request.
     *     The exact value received from the client:
     *
     *     Unique, randomly generated, opaque, and non-guessable string that is sent
     *     when starting an authentication request and validated when processing the response. */
    'OAuth2AuthorizationCodeGrantSuccessParams.state': string
    /** @description Page index.
     *
     *     Default is 1. */
    'Pagination.page': number
    /** @description The maximum number of items per page.
     *
     *     Default is 100. */
    'Pagination.pageSize': number
    /** @description The order direction. */
    'PlanAddonOrderByOrdering.order': components['schemas']['SortOrder']
    /** @description The order by field. */
    'PlanAddonOrderByOrdering.orderBy': components['schemas']['PlanAddonOrderBy']
    /** @description The order direction. */
    'PlanOrderByOrdering.order': components['schemas']['SortOrder']
    /** @description The order by field. */
    'PlanOrderByOrdering.orderBy': components['schemas']['PlanOrderBy']
    /** @description What parts of the customer output to expand */
    queryCustomerGet: components['schemas']['CustomerExpand'][]
    /** @description What parts of the list output to expand in listings */
    'queryCustomerList.expand': components['schemas']['CustomerExpand'][]
    /** @description Include deleted customers. */
    'queryCustomerList.includeDeleted': boolean
    /** @description Filter customers by key.
     *     Case-sensitive exact match. */
    'queryCustomerList.key': string
    /** @description Filter customers by name.
     *     Case-insensitive partial match. */
    'queryCustomerList.name': string
    /** @description Filter customers by the plan key of their susbcription. */
    'queryCustomerList.planKey': string
    /** @description Filter customers by primary email.
     *     Case-insensitive partial match. */
    'queryCustomerList.primaryEmail': string
    /** @description Filter customers by usage attribution subject.
     *     Case-insensitive partial match. */
    'queryCustomerList.subject': string
    /** @description Filter customer data by app type. */
    'queryCustomerList.type': components['schemas']['AppType']
    /** @description Include deleted meters. */
    'queryMeterList.includeDeleted': boolean
  }
  requestBodies: never
  headers: never
  pathItems: never
}
export type Addon = components['schemas']['Addon']
export type AddonCreate = components['schemas']['AddonCreate']
export type AddonInstanceType = components['schemas']['AddonInstanceType']
export type AddonOrderBy = components['schemas']['AddonOrderBy']
export type AddonPaginatedResponse =
  components['schemas']['AddonPaginatedResponse']
export type AddonReplaceUpdate = components['schemas']['AddonReplaceUpdate']
export type AddonStatus = components['schemas']['AddonStatus']
export type Address = components['schemas']['Address']
export type Alignment = components['schemas']['Alignment']
export type Annotations = components['schemas']['Annotations']
export type App = components['schemas']['App']
export type AppCapability = components['schemas']['AppCapability']
export type AppCapabilityType = components['schemas']['AppCapabilityType']
export type AppPaginatedResponse = components['schemas']['AppPaginatedResponse']
export type AppReference = components['schemas']['AppReference']
export type AppReplaceUpdate = components['schemas']['AppReplaceUpdate']
export type AppStatus = components['schemas']['AppStatus']
export type AppType = components['schemas']['AppType']
export type BadRequestProblemResponse =
  components['schemas']['BadRequestProblemResponse']
export type BalanceHistoryWindow = components['schemas']['BalanceHistoryWindow']
export type BillingCustomerProfile =
  components['schemas']['BillingCustomerProfile']
export type BillingDiscountPercentage =
  components['schemas']['BillingDiscountPercentage']
export type BillingDiscountReason =
  components['schemas']['BillingDiscountReason']
export type BillingDiscountUsage = components['schemas']['BillingDiscountUsage']
export type BillingDiscounts = components['schemas']['BillingDiscounts']
export type BillingParty = components['schemas']['BillingParty']
export type BillingPartyReplaceUpdate =
  components['schemas']['BillingPartyReplaceUpdate']
export type BillingPartyTaxIdentity =
  components['schemas']['BillingPartyTaxIdentity']
export type BillingProfile = components['schemas']['BillingProfile']
export type BillingProfileAppReferences =
  components['schemas']['BillingProfileAppReferences']
export type BillingProfileApps = components['schemas']['BillingProfileApps']
export type BillingProfileAppsCreate =
  components['schemas']['BillingProfileAppsCreate']
export type BillingProfileAppsOrReference =
  components['schemas']['BillingProfileAppsOrReference']
export type BillingProfileCreate = components['schemas']['BillingProfileCreate']
export type BillingProfileCustomerOverride =
  components['schemas']['BillingProfileCustomerOverride']
export type BillingProfileCustomerOverrideCreate =
  components['schemas']['BillingProfileCustomerOverrideCreate']
export type BillingProfileCustomerOverrideExpand =
  components['schemas']['BillingProfileCustomerOverrideExpand']
export type BillingProfileCustomerOverrideOrderBy =
  components['schemas']['BillingProfileCustomerOverrideOrderBy']
export type BillingProfileCustomerOverrideWithDetails =
  components['schemas']['BillingProfileCustomerOverrideWithDetails']
export type BillingProfileCustomerOverrideWithDetailsPaginatedResponse =
  components['schemas']['BillingProfileCustomerOverrideWithDetailsPaginatedResponse']
export type BillingProfileExpand = components['schemas']['BillingProfileExpand']
export type BillingProfileOrderBy =
  components['schemas']['BillingProfileOrderBy']
export type BillingProfilePaginatedResponse =
  components['schemas']['BillingProfilePaginatedResponse']
export type BillingProfileReplaceUpdateWithWorkflow =
  components['schemas']['BillingProfileReplaceUpdateWithWorkflow']
export type BillingTaxIdentificationCode =
  components['schemas']['BillingTaxIdentificationCode']
export type BillingWorkflow = components['schemas']['BillingWorkflow']
export type BillingWorkflowCollectionAlignment =
  components['schemas']['BillingWorkflowCollectionAlignment']
export type BillingWorkflowCollectionAlignmentSubscription =
  components['schemas']['BillingWorkflowCollectionAlignmentSubscription']
export type BillingWorkflowCollectionSettings =
  components['schemas']['BillingWorkflowCollectionSettings']
export type BillingWorkflowCreate =
  components['schemas']['BillingWorkflowCreate']
export type BillingWorkflowInvoicingSettings =
  components['schemas']['BillingWorkflowInvoicingSettings']
export type BillingWorkflowPaymentSettings =
  components['schemas']['BillingWorkflowPaymentSettings']
export type BillingWorkflowTaxSettings =
  components['schemas']['BillingWorkflowTaxSettings']
export type CheckoutSessionCustomTextAfterSubmitParams =
  components['schemas']['CheckoutSessionCustomTextAfterSubmitParams']
export type CheckoutSessionUiMode =
  components['schemas']['CheckoutSessionUIMode']
export type ClientAppStartResponse =
  components['schemas']['ClientAppStartResponse']
export type CollectionMethod = components['schemas']['CollectionMethod']
export type ConflictProblemResponse =
  components['schemas']['ConflictProblemResponse']
export type CountryCode = components['schemas']['CountryCode']
export type CreateCheckoutSessionTaxIdCollection =
  components['schemas']['CreateCheckoutSessionTaxIdCollection']
export type CreateCheckoutSessionTaxIdCollectionRequired =
  components['schemas']['CreateCheckoutSessionTaxIdCollectionRequired']
export type CreateStripeCheckoutSessionBillingAddressCollection =
  components['schemas']['CreateStripeCheckoutSessionBillingAddressCollection']
export type CreateStripeCheckoutSessionConsentCollection =
  components['schemas']['CreateStripeCheckoutSessionConsentCollection']
export type CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreement =
  components['schemas']['CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreement']
export type CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPosition =
  components['schemas']['CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPosition']
export type CreateStripeCheckoutSessionConsentCollectionPromotions =
  components['schemas']['CreateStripeCheckoutSessionConsentCollectionPromotions']
export type CreateStripeCheckoutSessionConsentCollectionTermsOfService =
  components['schemas']['CreateStripeCheckoutSessionConsentCollectionTermsOfService']
export type CreateStripeCheckoutSessionCustomerUpdate =
  components['schemas']['CreateStripeCheckoutSessionCustomerUpdate']
export type CreateStripeCheckoutSessionCustomerUpdateBehavior =
  components['schemas']['CreateStripeCheckoutSessionCustomerUpdateBehavior']
export type CreateStripeCheckoutSessionRedirectOnCompletion =
  components['schemas']['CreateStripeCheckoutSessionRedirectOnCompletion']
export type CreateStripeCheckoutSessionRequest =
  components['schemas']['CreateStripeCheckoutSessionRequest']
export type CreateStripeCheckoutSessionRequestOptions =
  components['schemas']['CreateStripeCheckoutSessionRequestOptions']
export type CreateStripeCheckoutSessionResult =
  components['schemas']['CreateStripeCheckoutSessionResult']
export type CreditNoteOriginalInvoiceRef =
  components['schemas']['CreditNoteOriginalInvoiceRef']
export type Currency = components['schemas']['Currency']
export type CurrencyCode = components['schemas']['CurrencyCode']
export type CustomInvoicingApp = components['schemas']['CustomInvoicingApp']
export type CustomInvoicingAppReplaceUpdate =
  components['schemas']['CustomInvoicingAppReplaceUpdate']
export type CustomInvoicingCustomerAppData =
  components['schemas']['CustomInvoicingCustomerAppData']
export type CustomInvoicingDraftSynchronizedRequest =
  components['schemas']['CustomInvoicingDraftSynchronizedRequest']
export type CustomInvoicingFinalizedInvoicingRequest =
  components['schemas']['CustomInvoicingFinalizedInvoicingRequest']
export type CustomInvoicingFinalizedPaymentRequest =
  components['schemas']['CustomInvoicingFinalizedPaymentRequest']
export type CustomInvoicingFinalizedRequest =
  components['schemas']['CustomInvoicingFinalizedRequest']
export type CustomInvoicingLineDiscountExternalIdMapping =
  components['schemas']['CustomInvoicingLineDiscountExternalIdMapping']
export type CustomInvoicingLineExternalIdMapping =
  components['schemas']['CustomInvoicingLineExternalIdMapping']
export type CustomInvoicingPaymentTrigger =
  components['schemas']['CustomInvoicingPaymentTrigger']
export type CustomInvoicingSyncResult =
  components['schemas']['CustomInvoicingSyncResult']
export type CustomInvoicingTaxConfig =
  components['schemas']['CustomInvoicingTaxConfig']
export type CustomInvoicingUpdatePaymentStatusRequest =
  components['schemas']['CustomInvoicingUpdatePaymentStatusRequest']
export type CustomPlanInput = components['schemas']['CustomPlanInput']
export type CustomSubscriptionChange =
  components['schemas']['CustomSubscriptionChange']
export type CustomSubscriptionCreate =
  components['schemas']['CustomSubscriptionCreate']
export type Customer = components['schemas']['Customer']
export type CustomerAccess = components['schemas']['CustomerAccess']
export type CustomerAppData = components['schemas']['CustomerAppData']
export type CustomerAppDataCreateOrUpdateItem =
  components['schemas']['CustomerAppDataCreateOrUpdateItem']
export type CustomerAppDataPaginatedResponse =
  components['schemas']['CustomerAppDataPaginatedResponse']
export type CustomerCreate = components['schemas']['CustomerCreate']
export type CustomerExpand = components['schemas']['CustomerExpand']
export type CustomerId = components['schemas']['CustomerId']
export type CustomerKey = components['schemas']['CustomerKey']
export type CustomerOrderBy = components['schemas']['CustomerOrderBy']
export type CustomerPaginatedResponse =
  components['schemas']['CustomerPaginatedResponse']
export type CustomerReplaceUpdate =
  components['schemas']['CustomerReplaceUpdate']
export type CustomerUsageAttribution =
  components['schemas']['CustomerUsageAttribution']
export type DiscountPercentage = components['schemas']['DiscountPercentage']
export type DiscountReasonMaximumSpend =
  components['schemas']['DiscountReasonMaximumSpend']
export type DiscountReasonRatecardPercentage =
  components['schemas']['DiscountReasonRatecardPercentage']
export type DiscountReasonRatecardUsage =
  components['schemas']['DiscountReasonRatecardUsage']
export type DiscountUsage = components['schemas']['DiscountUsage']
export type Discounts = components['schemas']['Discounts']
export type DynamicPriceWithCommitments =
  components['schemas']['DynamicPriceWithCommitments']
export type EditSubscriptionAddItem =
  components['schemas']['EditSubscriptionAddItem']
export type EditSubscriptionAddPhase =
  components['schemas']['EditSubscriptionAddPhase']
export type EditSubscriptionRemoveItem =
  components['schemas']['EditSubscriptionRemoveItem']
export type EditSubscriptionRemovePhase =
  components['schemas']['EditSubscriptionRemovePhase']
export type EditSubscriptionStretchPhase =
  components['schemas']['EditSubscriptionStretchPhase']
export type EditSubscriptionUnscheduleEdit =
  components['schemas']['EditSubscriptionUnscheduleEdit']
export type Entitlement = components['schemas']['Entitlement']
export type EntitlementBaseTemplate =
  components['schemas']['EntitlementBaseTemplate']
export type EntitlementBoolean = components['schemas']['EntitlementBoolean']
export type EntitlementBooleanCreateInputs =
  components['schemas']['EntitlementBooleanCreateInputs']
export type EntitlementCreateInputs =
  components['schemas']['EntitlementCreateInputs']
export type EntitlementGrant = components['schemas']['EntitlementGrant']
export type EntitlementGrantCreateInput =
  components['schemas']['EntitlementGrantCreateInput']
export type EntitlementMetered = components['schemas']['EntitlementMetered']
export type EntitlementMeteredCreateInputs =
  components['schemas']['EntitlementMeteredCreateInputs']
export type EntitlementOrderBy = components['schemas']['EntitlementOrderBy']
export type EntitlementPaginatedResponse =
  components['schemas']['EntitlementPaginatedResponse']
export type EntitlementStatic = components['schemas']['EntitlementStatic']
export type EntitlementStaticCreateInputs =
  components['schemas']['EntitlementStaticCreateInputs']
export type EntitlementType = components['schemas']['EntitlementType']
export type EntitlementValue = components['schemas']['EntitlementValue']
export type Event = components['schemas']['Event']
export type ExpirationDuration = components['schemas']['ExpirationDuration']
export type ExpirationPeriod = components['schemas']['ExpirationPeriod']
export type Feature = components['schemas']['Feature']
export type FeatureCreateInputs = components['schemas']['FeatureCreateInputs']
export type FeatureMeta = components['schemas']['FeatureMeta']
export type FeatureOrderBy = components['schemas']['FeatureOrderBy']
export type FeaturePaginatedResponse =
  components['schemas']['FeaturePaginatedResponse']
export type FilterString = components['schemas']['FilterString']
export type FilterTime = components['schemas']['FilterTime']
export type FlatPrice = components['schemas']['FlatPrice']
export type FlatPriceWithPaymentTerm =
  components['schemas']['FlatPriceWithPaymentTerm']
export type ForbiddenProblemResponse =
  components['schemas']['ForbiddenProblemResponse']
export type GatewayTimeoutProblemResponse =
  components['schemas']['GatewayTimeoutProblemResponse']
export type GrantBurnDownHistorySegment =
  components['schemas']['GrantBurnDownHistorySegment']
export type GrantOrderBy = components['schemas']['GrantOrderBy']
export type GrantPaginatedResponse =
  components['schemas']['GrantPaginatedResponse']
export type GrantUsageRecord = components['schemas']['GrantUsageRecord']
export type IdResource = components['schemas']['IDResource']
export type IngestEventsBody = components['schemas']['IngestEventsBody']
export type IngestedEvent = components['schemas']['IngestedEvent']
export type IngestedEventCursorPaginatedResponse =
  components['schemas']['IngestedEventCursorPaginatedResponse']
export type InstallMethod = components['schemas']['InstallMethod']
export type InternalServerErrorProblemResponse =
  components['schemas']['InternalServerErrorProblemResponse']
export type Invoice = components['schemas']['Invoice']
export type InvoiceAppExternalIds =
  components['schemas']['InvoiceAppExternalIds']
export type InvoiceAvailableActionDetails =
  components['schemas']['InvoiceAvailableActionDetails']
export type InvoiceAvailableActionInvoiceDetails =
  components['schemas']['InvoiceAvailableActionInvoiceDetails']
export type InvoiceAvailableActions =
  components['schemas']['InvoiceAvailableActions']
export type InvoiceDetailedLine = components['schemas']['InvoiceDetailedLine']
export type InvoiceDetailedLineCostCategory =
  components['schemas']['InvoiceDetailedLineCostCategory']
export type InvoiceDetailedLineRateCard =
  components['schemas']['InvoiceDetailedLineRateCard']
export type InvoiceDocumentRef = components['schemas']['InvoiceDocumentRef']
export type InvoiceDocumentRefType =
  components['schemas']['InvoiceDocumentRefType']
export type InvoiceExpand = components['schemas']['InvoiceExpand']
export type InvoiceGenericDocumentRef =
  components['schemas']['InvoiceGenericDocumentRef']
export type InvoiceLine = components['schemas']['InvoiceLine']
export type InvoiceLineAmountDiscount =
  components['schemas']['InvoiceLineAmountDiscount']
export type InvoiceLineAppExternalIds =
  components['schemas']['InvoiceLineAppExternalIds']
export type InvoiceLineDiscounts = components['schemas']['InvoiceLineDiscounts']
export type InvoiceLineManagedBy = components['schemas']['InvoiceLineManagedBy']
export type InvoiceLineReplaceUpdate =
  components['schemas']['InvoiceLineReplaceUpdate']
export type InvoiceLineStatus = components['schemas']['InvoiceLineStatus']
export type InvoiceLineSubscriptionReference =
  components['schemas']['InvoiceLineSubscriptionReference']
export type InvoiceLineTaxBehavior =
  components['schemas']['InvoiceLineTaxBehavior']
export type InvoiceLineTaxItem = components['schemas']['InvoiceLineTaxItem']
export type InvoiceLineUsageDiscount =
  components['schemas']['InvoiceLineUsageDiscount']
export type InvoiceNumber = components['schemas']['InvoiceNumber']
export type InvoiceOrderBy = components['schemas']['InvoiceOrderBy']
export type InvoicePaginatedResponse =
  components['schemas']['InvoicePaginatedResponse']
export type InvoicePaymentTerms = components['schemas']['InvoicePaymentTerms']
export type InvoicePendingLineCreate =
  components['schemas']['InvoicePendingLineCreate']
export type InvoicePendingLineCreateInput =
  components['schemas']['InvoicePendingLineCreateInput']
export type InvoicePendingLineCreateResponse =
  components['schemas']['InvoicePendingLineCreateResponse']
export type InvoicePendingLinesActionFiltersInput =
  components['schemas']['InvoicePendingLinesActionFiltersInput']
export type InvoicePendingLinesActionInput =
  components['schemas']['InvoicePendingLinesActionInput']
export type InvoiceReference = components['schemas']['InvoiceReference']
export type InvoiceReplaceUpdate = components['schemas']['InvoiceReplaceUpdate']
export type InvoiceSimulationInput =
  components['schemas']['InvoiceSimulationInput']
export type InvoiceSimulationLine =
  components['schemas']['InvoiceSimulationLine']
export type InvoiceStatus = components['schemas']['InvoiceStatus']
export type InvoiceStatusDetails = components['schemas']['InvoiceStatusDetails']
export type InvoiceTotals = components['schemas']['InvoiceTotals']
export type InvoiceType = components['schemas']['InvoiceType']
export type InvoiceUsageBasedRateCard =
  components['schemas']['InvoiceUsageBasedRateCard']
export type InvoiceWorkflowInvoicingSettingsReplaceUpdate =
  components['schemas']['InvoiceWorkflowInvoicingSettingsReplaceUpdate']
export type InvoiceWorkflowReplaceUpdate =
  components['schemas']['InvoiceWorkflowReplaceUpdate']
export type InvoiceWorkflowSettings =
  components['schemas']['InvoiceWorkflowSettings']
export type InvoiceWorkflowSettingsReplaceUpdate =
  components['schemas']['InvoiceWorkflowSettingsReplaceUpdate']
export type ListEntitlementsResult =
  components['schemas']['ListEntitlementsResult']
export type ListFeaturesResult = components['schemas']['ListFeaturesResult']
export type MarketplaceInstallResponse =
  components['schemas']['MarketplaceInstallResponse']
export type MarketplaceListing = components['schemas']['MarketplaceListing']
export type MarketplaceListingPaginatedResponse =
  components['schemas']['MarketplaceListingPaginatedResponse']
export type MeasureUsageFrom = components['schemas']['MeasureUsageFrom']
export type MeasureUsageFromPreset =
  components['schemas']['MeasureUsageFromPreset']
export type MeasureUsageFromTime = components['schemas']['MeasureUsageFromTime']
export type Metadata = components['schemas']['Metadata']
export type Meter = components['schemas']['Meter']
export type MeterAggregation = components['schemas']['MeterAggregation']
export type MeterCreate = components['schemas']['MeterCreate']
export type MeterOrderBy = components['schemas']['MeterOrderBy']
export type MeterQueryRequest = components['schemas']['MeterQueryRequest']
export type MeterQueryResult = components['schemas']['MeterQueryResult']
export type MeterQueryRow = components['schemas']['MeterQueryRow']
export type MeterUpdate = components['schemas']['MeterUpdate']
export type NotFoundProblemResponse =
  components['schemas']['NotFoundProblemResponse']
export type NotImplementedProblemResponse =
  components['schemas']['NotImplementedProblemResponse']
export type NotificationChannel = components['schemas']['NotificationChannel']
export type NotificationChannelCreateRequest =
  components['schemas']['NotificationChannelCreateRequest']
export type NotificationChannelMeta =
  components['schemas']['NotificationChannelMeta']
export type NotificationChannelOrderBy =
  components['schemas']['NotificationChannelOrderBy']
export type NotificationChannelPaginatedResponse =
  components['schemas']['NotificationChannelPaginatedResponse']
export type NotificationChannelType =
  components['schemas']['NotificationChannelType']
export type NotificationChannelWebhook =
  components['schemas']['NotificationChannelWebhook']
export type NotificationChannelWebhookCreateRequest =
  components['schemas']['NotificationChannelWebhookCreateRequest']
export type NotificationEvent = components['schemas']['NotificationEvent']
export type NotificationEventBalanceThresholdPayload =
  components['schemas']['NotificationEventBalanceThresholdPayload']
export type NotificationEventBalanceThresholdPayloadData =
  components['schemas']['NotificationEventBalanceThresholdPayloadData']
export type NotificationEventDeliveryStatus =
  components['schemas']['NotificationEventDeliveryStatus']
export type NotificationEventDeliveryStatusState =
  components['schemas']['NotificationEventDeliveryStatusState']
export type NotificationEventEntitlementValuePayloadBase =
  components['schemas']['NotificationEventEntitlementValuePayloadBase']
export type NotificationEventInvoiceCreatedPayload =
  components['schemas']['NotificationEventInvoiceCreatedPayload']
export type NotificationEventInvoiceUpdatedPayload =
  components['schemas']['NotificationEventInvoiceUpdatedPayload']
export type NotificationEventOrderBy =
  components['schemas']['NotificationEventOrderBy']
export type NotificationEventPaginatedResponse =
  components['schemas']['NotificationEventPaginatedResponse']
export type NotificationEventPayload =
  components['schemas']['NotificationEventPayload']
export type NotificationEventResetPayload =
  components['schemas']['NotificationEventResetPayload']
export type NotificationEventType =
  components['schemas']['NotificationEventType']
export type NotificationRule = components['schemas']['NotificationRule']
export type NotificationRuleBalanceThreshold =
  components['schemas']['NotificationRuleBalanceThreshold']
export type NotificationRuleBalanceThresholdCreateRequest =
  components['schemas']['NotificationRuleBalanceThresholdCreateRequest']
export type NotificationRuleBalanceThresholdValue =
  components['schemas']['NotificationRuleBalanceThresholdValue']
export type NotificationRuleBalanceThresholdValueType =
  components['schemas']['NotificationRuleBalanceThresholdValueType']
export type NotificationRuleCreateRequest =
  components['schemas']['NotificationRuleCreateRequest']
export type NotificationRuleEntitlementReset =
  components['schemas']['NotificationRuleEntitlementReset']
export type NotificationRuleEntitlementResetCreateRequest =
  components['schemas']['NotificationRuleEntitlementResetCreateRequest']
export type NotificationRuleInvoiceCreated =
  components['schemas']['NotificationRuleInvoiceCreated']
export type NotificationRuleInvoiceCreatedCreateRequest =
  components['schemas']['NotificationRuleInvoiceCreatedCreateRequest']
export type NotificationRuleInvoiceUpdated =
  components['schemas']['NotificationRuleInvoiceUpdated']
export type NotificationRuleInvoiceUpdatedCreateRequest =
  components['schemas']['NotificationRuleInvoiceUpdatedCreateRequest']
export type NotificationRuleOrderBy =
  components['schemas']['NotificationRuleOrderBy']
export type NotificationRulePaginatedResponse =
  components['schemas']['NotificationRulePaginatedResponse']
export type Numeric = components['schemas']['Numeric']
export type OAuth2AuthorizationCodeGrantErrorType =
  components['schemas']['OAuth2AuthorizationCodeGrantErrorType']
export type PackagePriceWithCommitments =
  components['schemas']['PackagePriceWithCommitments']
export type PaymentDueDate = components['schemas']['PaymentDueDate']
export type PaymentTermDueDate = components['schemas']['PaymentTermDueDate']
export type PaymentTermInstant = components['schemas']['PaymentTermInstant']
export type PaymentTerms = components['schemas']['PaymentTerms']
export type Percentage = components['schemas']['Percentage']
export type Period = components['schemas']['Period']
export type Plan = components['schemas']['Plan']
export type PlanAddon = components['schemas']['PlanAddon']
export type PlanAddonCreate = components['schemas']['PlanAddonCreate']
export type PlanAddonOrderBy = components['schemas']['PlanAddonOrderBy']
export type PlanAddonPaginatedResponse =
  components['schemas']['PlanAddonPaginatedResponse']
export type PlanAddonReplaceUpdate =
  components['schemas']['PlanAddonReplaceUpdate']
export type PlanCreate = components['schemas']['PlanCreate']
export type PlanOrderBy = components['schemas']['PlanOrderBy']
export type PlanPaginatedResponse =
  components['schemas']['PlanPaginatedResponse']
export type PlanPhase = components['schemas']['PlanPhase']
export type PlanReference = components['schemas']['PlanReference']
export type PlanReferenceInput = components['schemas']['PlanReferenceInput']
export type PlanReplaceUpdate = components['schemas']['PlanReplaceUpdate']
export type PlanStatus = components['schemas']['PlanStatus']
export type PlanSubscriptionChange =
  components['schemas']['PlanSubscriptionChange']
export type PlanSubscriptionCreate =
  components['schemas']['PlanSubscriptionCreate']
export type PortalToken = components['schemas']['PortalToken']
export type PreconditionFailedProblemResponse =
  components['schemas']['PreconditionFailedProblemResponse']
export type PricePaymentTerm = components['schemas']['PricePaymentTerm']
export type PriceTier = components['schemas']['PriceTier']
export type ProRatingConfig = components['schemas']['ProRatingConfig']
export type ProRatingMode = components['schemas']['ProRatingMode']
export type Progress = components['schemas']['Progress']
export type RateCard = components['schemas']['RateCard']
export type RateCardBooleanEntitlement =
  components['schemas']['RateCardBooleanEntitlement']
export type RateCardEntitlement = components['schemas']['RateCardEntitlement']
export type RateCardFlatFee = components['schemas']['RateCardFlatFee']
export type RateCardMeteredEntitlement =
  components['schemas']['RateCardMeteredEntitlement']
export type RateCardStaticEntitlement =
  components['schemas']['RateCardStaticEntitlement']
export type RateCardUsageBased = components['schemas']['RateCardUsageBased']
export type RateCardUsageBasedPrice =
  components['schemas']['RateCardUsageBasedPrice']
export type RecurringPeriod = components['schemas']['RecurringPeriod']
export type RecurringPeriodCreateInput =
  components['schemas']['RecurringPeriodCreateInput']
export type RecurringPeriodInterval =
  components['schemas']['RecurringPeriodInterval']
export type RecurringPeriodIntervalEnum =
  components['schemas']['RecurringPeriodIntervalEnum']
export type RemovePhaseShifting = components['schemas']['RemovePhaseShifting']
export type ResetEntitlementUsageInput =
  components['schemas']['ResetEntitlementUsageInput']
export type SandboxApp = components['schemas']['SandboxApp']
export type SandboxAppReplaceUpdate =
  components['schemas']['SandboxAppReplaceUpdate']
export type SandboxCustomerAppData =
  components['schemas']['SandboxCustomerAppData']
export type ServiceUnavailableProblemResponse =
  components['schemas']['ServiceUnavailableProblemResponse']
export type SortOrder = components['schemas']['SortOrder']
export type StripeApiKeyInput = components['schemas']['StripeAPIKeyInput']
export type StripeApp = components['schemas']['StripeApp']
export type StripeAppReplaceUpdate =
  components['schemas']['StripeAppReplaceUpdate']
export type StripeCheckoutSessionMode =
  components['schemas']['StripeCheckoutSessionMode']
export type StripeCustomerAppData =
  components['schemas']['StripeCustomerAppData']
export type StripeCustomerAppDataCreateOrUpdateItem =
  components['schemas']['StripeCustomerAppDataCreateOrUpdateItem']
export type StripeTaxConfig = components['schemas']['StripeTaxConfig']
export type StripeWebhookEvent = components['schemas']['StripeWebhookEvent']
export type StripeWebhookResponse =
  components['schemas']['StripeWebhookResponse']
export type Subject = components['schemas']['Subject']
export type SubjectUpsert = components['schemas']['SubjectUpsert']
export type Subscription = components['schemas']['Subscription']
export type SubscriptionAddon = components['schemas']['SubscriptionAddon']
export type SubscriptionAddonCreate =
  components['schemas']['SubscriptionAddonCreate']
export type SubscriptionAddonRateCard =
  components['schemas']['SubscriptionAddonRateCard']
export type SubscriptionAddonTimelineSegment =
  components['schemas']['SubscriptionAddonTimelineSegment']
export type SubscriptionAddonUpdate =
  components['schemas']['SubscriptionAddonUpdate']
export type SubscriptionAlignment =
  components['schemas']['SubscriptionAlignment']
export type SubscriptionChange = components['schemas']['SubscriptionChange']
export type SubscriptionChangeResponseBody =
  components['schemas']['SubscriptionChangeResponseBody']
export type SubscriptionCreate = components['schemas']['SubscriptionCreate']
export type SubscriptionEdit = components['schemas']['SubscriptionEdit']
export type SubscriptionEditOperation =
  components['schemas']['SubscriptionEditOperation']
export type SubscriptionExpanded = components['schemas']['SubscriptionExpanded']
export type SubscriptionItem = components['schemas']['SubscriptionItem']
export type SubscriptionItemIncluded =
  components['schemas']['SubscriptionItemIncluded']
export type SubscriptionPaginatedResponse =
  components['schemas']['SubscriptionPaginatedResponse']
export type SubscriptionPhaseCreate =
  components['schemas']['SubscriptionPhaseCreate']
export type SubscriptionPhaseExpanded =
  components['schemas']['SubscriptionPhaseExpanded']
export type SubscriptionStatus = components['schemas']['SubscriptionStatus']
export type SubscriptionTiming = components['schemas']['SubscriptionTiming']
export type SubscriptionTimingEnum =
  components['schemas']['SubscriptionTimingEnum']
export type TaxBehavior = components['schemas']['TaxBehavior']
export type TaxConfig = components['schemas']['TaxConfig']
export type TieredPriceMode = components['schemas']['TieredPriceMode']
export type TieredPriceWithCommitments =
  components['schemas']['TieredPriceWithCommitments']
export type UnauthorizedProblemResponse =
  components['schemas']['UnauthorizedProblemResponse']
export type UnexpectedProblemResponse =
  components['schemas']['UnexpectedProblemResponse']
export type UnitPrice = components['schemas']['UnitPrice']
export type UnitPriceWithCommitments =
  components['schemas']['UnitPriceWithCommitments']
export type ValidationError = components['schemas']['ValidationError']
export type ValidationIssue = components['schemas']['ValidationIssue']
export type ValidationIssueSeverity =
  components['schemas']['ValidationIssueSeverity']
export type VoidInvoiceActionCreate =
  components['schemas']['VoidInvoiceActionCreate']
export type VoidInvoiceActionCreateItem =
  components['schemas']['VoidInvoiceActionCreateItem']
export type VoidInvoiceActionInput =
  components['schemas']['VoidInvoiceActionInput']
export type VoidInvoiceActionLineOverride =
  components['schemas']['VoidInvoiceActionLineOverride']
export type VoidInvoiceLineActionCreate =
  components['schemas']['VoidInvoiceLineActionCreate']
export type VoidInvoiceLineActionCreateItem =
  components['schemas']['VoidInvoiceLineActionCreateItem']
export type VoidInvoiceLineDiscardAction =
  components['schemas']['VoidInvoiceLineDiscardAction']
export type VoidInvoiceLinePendingActionCreate =
  components['schemas']['VoidInvoiceLinePendingActionCreate']
export type VoidInvoiceLinePendingActionCreateItem =
  components['schemas']['VoidInvoiceLinePendingActionCreateItem']
export type WindowSize = components['schemas']['WindowSize']
export type WindowedBalanceHistory =
  components['schemas']['WindowedBalanceHistory']
export type ParameterAddonOrderByOrderingOrder =
  components['parameters']['AddonOrderByOrdering.order']
export type ParameterAddonOrderByOrderingOrderBy =
  components['parameters']['AddonOrderByOrdering.orderBy']
export type ParameterBillingProfileCustomerOverrideOrderByOrderingOrder =
  components['parameters']['BillingProfileCustomerOverrideOrderByOrdering.order']
export type ParameterBillingProfileCustomerOverrideOrderByOrderingOrderBy =
  components['parameters']['BillingProfileCustomerOverrideOrderByOrdering.orderBy']
export type ParameterBillingProfileListCustomerOverridesParamsBillingProfile =
  components['parameters']['BillingProfileListCustomerOverridesParams.billingProfile']
export type ParameterBillingProfileListCustomerOverridesParamsCustomerId =
  components['parameters']['BillingProfileListCustomerOverridesParams.customerId']
export type ParameterBillingProfileListCustomerOverridesParamsCustomerKey =
  components['parameters']['BillingProfileListCustomerOverridesParams.customerKey']
export type ParameterBillingProfileListCustomerOverridesParamsCustomerName =
  components['parameters']['BillingProfileListCustomerOverridesParams.customerName']
export type ParameterBillingProfileListCustomerOverridesParamsCustomerPrimaryEmail =
  components['parameters']['BillingProfileListCustomerOverridesParams.customerPrimaryEmail']
export type ParameterBillingProfileListCustomerOverridesParamsExpand =
  components['parameters']['BillingProfileListCustomerOverridesParams.expand']
export type ParameterBillingProfileListCustomerOverridesParamsIncludeAllCustomers =
  components['parameters']['BillingProfileListCustomerOverridesParams.includeAllCustomers']
export type ParameterBillingProfileOrderByOrderingOrder =
  components['parameters']['BillingProfileOrderByOrdering.order']
export type ParameterBillingProfileOrderByOrderingOrderBy =
  components['parameters']['BillingProfileOrderByOrdering.orderBy']
export type ParameterCursorPaginationCursor =
  components['parameters']['CursorPagination.cursor']
export type ParameterCursorPaginationLimit =
  components['parameters']['CursorPagination.limit']
export type ParameterCustomerOrderByOrderingOrder =
  components['parameters']['CustomerOrderByOrdering.order']
export type ParameterCustomerOrderByOrderingOrderBy =
  components['parameters']['CustomerOrderByOrdering.orderBy']
export type ParameterEntitlementOrderByOrderingOrder =
  components['parameters']['EntitlementOrderByOrdering.order']
export type ParameterEntitlementOrderByOrderingOrderBy =
  components['parameters']['EntitlementOrderByOrdering.orderBy']
export type ParameterFeatureOrderByOrderingOrder =
  components['parameters']['FeatureOrderByOrdering.order']
export type ParameterFeatureOrderByOrderingOrderBy =
  components['parameters']['FeatureOrderByOrdering.orderBy']
export type ParameterGrantOrderByOrderingOrder =
  components['parameters']['GrantOrderByOrdering.order']
export type ParameterGrantOrderByOrderingOrderBy =
  components['parameters']['GrantOrderByOrdering.orderBy']
export type ParameterInvoiceListParamsCreatedAfter =
  components['parameters']['InvoiceListParams.createdAfter']
export type ParameterInvoiceListParamsCreatedBefore =
  components['parameters']['InvoiceListParams.createdBefore']
export type ParameterInvoiceListParamsCustomers =
  components['parameters']['InvoiceListParams.customers']
export type ParameterInvoiceListParamsExpand =
  components['parameters']['InvoiceListParams.expand']
export type ParameterInvoiceListParamsExtendedStatuses =
  components['parameters']['InvoiceListParams.extendedStatuses']
export type ParameterInvoiceListParamsIncludeDeleted =
  components['parameters']['InvoiceListParams.includeDeleted']
export type ParameterInvoiceListParamsIssuedAfter =
  components['parameters']['InvoiceListParams.issuedAfter']
export type ParameterInvoiceListParamsIssuedBefore =
  components['parameters']['InvoiceListParams.issuedBefore']
export type ParameterInvoiceListParamsPeriodStartAfter =
  components['parameters']['InvoiceListParams.periodStartAfter']
export type ParameterInvoiceListParamsPeriodStartBefore =
  components['parameters']['InvoiceListParams.periodStartBefore']
export type ParameterInvoiceListParamsStatuses =
  components['parameters']['InvoiceListParams.statuses']
export type ParameterInvoiceOrderByOrderingOrder =
  components['parameters']['InvoiceOrderByOrdering.order']
export type ParameterInvoiceOrderByOrderingOrderBy =
  components['parameters']['InvoiceOrderByOrdering.orderBy']
export type ParameterLimitOffsetLimit =
  components['parameters']['LimitOffset.limit']
export type ParameterLimitOffsetOffset =
  components['parameters']['LimitOffset.offset']
export type ParameterMarketplaceApiKeyInstallRequestType =
  components['parameters']['MarketplaceApiKeyInstallRequest.type']
export type ParameterMarketplaceInstallRequestType =
  components['parameters']['MarketplaceInstallRequest.type']
export type ParameterMarketplaceOAuth2InstallAuthorizeRequestType =
  components['parameters']['MarketplaceOAuth2InstallAuthorizeRequest.type']
export type ParameterMeterOrderByOrderingOrder =
  components['parameters']['MeterOrderByOrdering.order']
export type ParameterMeterOrderByOrderingOrderBy =
  components['parameters']['MeterOrderByOrdering.orderBy']
export type ParameterMeterQueryClientId =
  components['parameters']['MeterQuery.clientId']
export type ParameterMeterQueryFilterGroupBy =
  components['parameters']['MeterQuery.filterGroupBy']
export type ParameterMeterQueryFrom =
  components['parameters']['MeterQuery.from']
export type ParameterMeterQueryGroupBy =
  components['parameters']['MeterQuery.groupBy']
export type ParameterMeterQuerySubject =
  components['parameters']['MeterQuery.subject']
export type ParameterMeterQueryTo = components['parameters']['MeterQuery.to']
export type ParameterMeterQueryWindowSize =
  components['parameters']['MeterQuery.windowSize']
export type ParameterMeterQueryWindowTimeZone =
  components['parameters']['MeterQuery.windowTimeZone']
export type ParameterNotificationChannelOrderByOrderingOrder =
  components['parameters']['NotificationChannelOrderByOrdering.order']
export type ParameterNotificationChannelOrderByOrderingOrderBy =
  components['parameters']['NotificationChannelOrderByOrdering.orderBy']
export type ParameterNotificationEventOrderByOrderingOrder =
  components['parameters']['NotificationEventOrderByOrdering.order']
export type ParameterNotificationEventOrderByOrderingOrderBy =
  components['parameters']['NotificationEventOrderByOrdering.orderBy']
export type ParameterNotificationRuleOrderByOrderingOrder =
  components['parameters']['NotificationRuleOrderByOrdering.order']
export type ParameterNotificationRuleOrderByOrderingOrderBy =
  components['parameters']['NotificationRuleOrderByOrdering.orderBy']
export type ParameterOAuth2AuthorizationCodeGrantErrorParamsError =
  components['parameters']['OAuth2AuthorizationCodeGrantErrorParams.error']
export type ParameterOAuth2AuthorizationCodeGrantErrorParamsErrorDescription =
  components['parameters']['OAuth2AuthorizationCodeGrantErrorParams.error_description']
export type ParameterOAuth2AuthorizationCodeGrantErrorParamsErrorUri =
  components['parameters']['OAuth2AuthorizationCodeGrantErrorParams.error_uri']
export type ParameterOAuth2AuthorizationCodeGrantSuccessParamsCode =
  components['parameters']['OAuth2AuthorizationCodeGrantSuccessParams.code']
export type ParameterOAuth2AuthorizationCodeGrantSuccessParamsState =
  components['parameters']['OAuth2AuthorizationCodeGrantSuccessParams.state']
export type ParameterPaginationPage =
  components['parameters']['Pagination.page']
export type ParameterPaginationPageSize =
  components['parameters']['Pagination.pageSize']
export type ParameterPlanAddonOrderByOrderingOrder =
  components['parameters']['PlanAddonOrderByOrdering.order']
export type ParameterPlanAddonOrderByOrderingOrderBy =
  components['parameters']['PlanAddonOrderByOrdering.orderBy']
export type ParameterPlanOrderByOrderingOrder =
  components['parameters']['PlanOrderByOrdering.order']
export type ParameterPlanOrderByOrderingOrderBy =
  components['parameters']['PlanOrderByOrdering.orderBy']
export type ParameterQueryCustomerGet =
  components['parameters']['queryCustomerGet']
export type ParameterQueryCustomerListExpand =
  components['parameters']['queryCustomerList.expand']
export type ParameterQueryCustomerListIncludeDeleted =
  components['parameters']['queryCustomerList.includeDeleted']
export type ParameterQueryCustomerListKey =
  components['parameters']['queryCustomerList.key']
export type ParameterQueryCustomerListName =
  components['parameters']['queryCustomerList.name']
export type ParameterQueryCustomerListPlanKey =
  components['parameters']['queryCustomerList.planKey']
export type ParameterQueryCustomerListPrimaryEmail =
  components['parameters']['queryCustomerList.primaryEmail']
export type ParameterQueryCustomerListSubject =
  components['parameters']['queryCustomerList.subject']
export type ParameterQueryCustomerListType =
  components['parameters']['queryCustomerList.type']
export type ParameterQueryMeterListIncludeDeleted =
  components['parameters']['queryMeterList.includeDeleted']
export type $defs = Record<string, never>
export interface operations {
  listAddons: {
    parameters: {
      query?: {
        /** @description Include deleted add-ons in response.
         *
         *     Usage: `?includeDeleted=true` */
        includeDeleted?: boolean
        /** @description Filter by addon.id attribute */
        id?: string[]
        /** @description Filter by addon.key attribute */
        key?: string[]
        /** @description Filter by addon.key and addon.version attributes */
        keyVersion?: {
          [key: string]: number[]
        }
        /** @description Only return add-ons with the given status.
         *
         *     Usage:
         *     - `?status=active`: return only the currently active add-ons
         *     - `?status=draft`: return only the draft add-ons
         *     - `?status=archived`: return only the archived add-ons */
        status?: components['schemas']['AddonStatus'][]
        /** @description Filter by addon.currency attribute */
        currency?: components['schemas']['CurrencyCode'][]
        /** @description Page index.
         *
         *     Default is 1. */
        page?: components['parameters']['Pagination.page']
        /** @description The maximum number of items per page.
         *
         *     Default is 100. */
        pageSize?: components['parameters']['Pagination.pageSize']
        /** @description The order direction. */
        order?: components['parameters']['AddonOrderByOrdering.order']
        /** @description The order by field. */
        orderBy?: components['parameters']['AddonOrderByOrdering.orderBy']
      }
      header?: never
      path?: never
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['AddonPaginatedResponse']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  createAddon: {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['AddonCreate']
      }
    }
    responses: {
      /** @description The request has succeeded and a new resource has been created as a result. */
      201: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Addon']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  getAddon: {
    parameters: {
      query?: {
        /** @description Include latest version of the add-on instead of the version in active state.
         *
         *     Usage: `?includeLatest=true` */
        includeLatest?: boolean
      }
      header?: never
      path: {
        addonId: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Addon']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  updateAddon: {
    parameters: {
      query?: never
      header?: never
      path: {
        addonId: string
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['AddonReplaceUpdate']
      }
    }
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Addon']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  deleteAddon: {
    parameters: {
      query?: never
      header?: never
      path: {
        addonId: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description There is no content to send for this request, but the headers may be useful.  */
      204: {
        headers: {
          [name: string]: unknown
        }
        content?: never
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  archiveAddon: {
    parameters: {
      query?: never
      header?: never
      path: {
        addonId: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Addon']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  publishAddon: {
    parameters: {
      query?: never
      header?: never
      path: {
        addonId: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Addon']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  listApps: {
    parameters: {
      query?: {
        /** @description Page index.
         *
         *     Default is 1. */
        page?: components['parameters']['Pagination.page']
        /** @description The maximum number of items per page.
         *
         *     Default is 100. */
        pageSize?: components['parameters']['Pagination.pageSize']
      }
      header?: never
      path?: never
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['AppPaginatedResponse']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  appCustomInvoicingDraftSynchronized: {
    parameters: {
      query?: never
      header?: never
      path: {
        invoiceId: string
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['CustomInvoicingDraftSynchronizedRequest']
      }
    }
    responses: {
      /** @description There is no content to send for this request, but the headers may be useful.  */
      204: {
        headers: {
          [name: string]: unknown
        }
        content?: never
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  appCustomInvoicingIssuingSynchronized: {
    parameters: {
      query?: never
      header?: never
      path: {
        invoiceId: string
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['CustomInvoicingFinalizedRequest']
      }
    }
    responses: {
      /** @description There is no content to send for this request, but the headers may be useful.  */
      204: {
        headers: {
          [name: string]: unknown
        }
        content?: never
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  appCustomInvoicingUpdatePaymentStatus: {
    parameters: {
      query?: never
      header?: never
      path: {
        invoiceId: string
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['CustomInvoicingUpdatePaymentStatusRequest']
      }
    }
    responses: {
      /** @description There is no content to send for this request, but the headers may be useful.  */
      204: {
        headers: {
          [name: string]: unknown
        }
        content?: never
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  getApp: {
    parameters: {
      query?: never
      header?: never
      path: {
        id: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['App']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  updateApp: {
    parameters: {
      query?: never
      header?: never
      path: {
        id: string
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['AppReplaceUpdate']
      }
    }
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['App']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  uninstallApp: {
    parameters: {
      query?: never
      header?: never
      path: {
        id: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description There is no content to send for this request, but the headers may be useful.  */
      204: {
        headers: {
          [name: string]: unknown
        }
        content?: never
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  updateStripeAPIKey: {
    parameters: {
      query?: never
      header?: never
      path: {
        id: string
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['StripeAPIKeyInput']
      }
    }
    responses: {
      /** @description There is no content to send for this request, but the headers may be useful.  */
      204: {
        headers: {
          [name: string]: unknown
        }
        content?: never
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  appStripeWebhook: {
    parameters: {
      query?: never
      header?: never
      path: {
        id: string
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['StripeWebhookEvent']
      }
    }
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['StripeWebhookResponse']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  listBillingProfileCustomerOverrides: {
    parameters: {
      query?: {
        /** @description Filter by billing profile. */
        billingProfile?: components['parameters']['BillingProfileListCustomerOverridesParams.billingProfile']
        /** @description Include customers without customer overrides.
         *
         *     If set to false only the customers specifically associated with a billing profile will be returned.
         *
         *     If set to true, in case of the default billing profile, all customers will be returned. */
        includeAllCustomers?: components['parameters']['BillingProfileListCustomerOverridesParams.includeAllCustomers']
        /** @description Filter by customer id. */
        customerId?: components['parameters']['BillingProfileListCustomerOverridesParams.customerId']
        /** @description Filter by customer name. */
        customerName?: components['parameters']['BillingProfileListCustomerOverridesParams.customerName']
        /** @description Filter by customer key */
        customerKey?: components['parameters']['BillingProfileListCustomerOverridesParams.customerKey']
        /** @description Filter by customer primary email */
        customerPrimaryEmail?: components['parameters']['BillingProfileListCustomerOverridesParams.customerPrimaryEmail']
        /** @description Expand the response with additional details. */
        expand?: components['parameters']['BillingProfileListCustomerOverridesParams.expand']
        /** @description The order direction. */
        order?: components['parameters']['BillingProfileCustomerOverrideOrderByOrdering.order']
        /** @description The order by field. */
        orderBy?: components['parameters']['BillingProfileCustomerOverrideOrderByOrdering.orderBy']
        /** @description Page index.
         *
         *     Default is 1. */
        page?: components['parameters']['Pagination.page']
        /** @description The maximum number of items per page.
         *
         *     Default is 100. */
        pageSize?: components['parameters']['Pagination.pageSize']
      }
      header?: never
      path?: never
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['BillingProfileCustomerOverrideWithDetailsPaginatedResponse']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  getBillingProfileCustomerOverride: {
    parameters: {
      query?: {
        expand?: components['schemas']['BillingProfileCustomerOverrideExpand'][]
      }
      header?: never
      path: {
        customerId: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['BillingProfileCustomerOverrideWithDetails']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  upsertBillingProfileCustomerOverride: {
    parameters: {
      query?: never
      header?: never
      path: {
        customerId: string
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['BillingProfileCustomerOverrideCreate']
      }
    }
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['BillingProfileCustomerOverrideWithDetails']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  deleteBillingProfileCustomerOverride: {
    parameters: {
      query?: never
      header?: never
      path: {
        customerId: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description There is no content to send for this request, but the headers may be useful.  */
      204: {
        headers: {
          [name: string]: unknown
        }
        content?: never
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  createPendingInvoiceLine: {
    parameters: {
      query?: never
      header?: never
      path: {
        customerId: string
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['InvoicePendingLineCreateInput']
      }
    }
    responses: {
      /** @description The request has succeeded and a new resource has been created as a result. */
      201: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['InvoicePendingLineCreateResponse']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  simulateInvoice: {
    parameters: {
      query?: never
      header?: never
      path: {
        customerId: string
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['InvoiceSimulationInput']
      }
    }
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Invoice']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  listInvoices: {
    parameters: {
      query?: {
        /** @description Filter by the invoice status. */
        statuses?: components['parameters']['InvoiceListParams.statuses']
        /** @description Filter by invoice extended statuses */
        extendedStatuses?: components['parameters']['InvoiceListParams.extendedStatuses']
        /** @description Filter by invoice issued time.
         *     Inclusive. */
        issuedAfter?: components['parameters']['InvoiceListParams.issuedAfter']
        /** @description Filter by invoice issued time.
         *     Inclusive. */
        issuedBefore?: components['parameters']['InvoiceListParams.issuedBefore']
        /** @description Filter by period start time.
         *     Inclusive. */
        periodStartAfter?: components['parameters']['InvoiceListParams.periodStartAfter']
        /** @description Filter by period start time.
         *     Inclusive. */
        periodStartBefore?: components['parameters']['InvoiceListParams.periodStartBefore']
        /** @description Filter by invoice created time.
         *     Inclusive. */
        createdAfter?: components['parameters']['InvoiceListParams.createdAfter']
        /** @description Filter by invoice created time.
         *     Inclusive. */
        createdBefore?: components['parameters']['InvoiceListParams.createdBefore']
        /** @description What parts of the list output to expand in listings */
        expand?: components['parameters']['InvoiceListParams.expand']
        /** @description Filter by customer ID */
        customers?: components['parameters']['InvoiceListParams.customers']
        /** @description Include deleted invoices */
        includeDeleted?: components['parameters']['InvoiceListParams.includeDeleted']
        /** @description Page index.
         *
         *     Default is 1. */
        page?: components['parameters']['Pagination.page']
        /** @description The maximum number of items per page.
         *
         *     Default is 100. */
        pageSize?: components['parameters']['Pagination.pageSize']
        /** @description The order direction. */
        order?: components['parameters']['InvoiceOrderByOrdering.order']
        /** @description The order by field. */
        orderBy?: components['parameters']['InvoiceOrderByOrdering.orderBy']
      }
      header?: never
      path?: never
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['InvoicePaginatedResponse']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  invoicePendingLinesAction: {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['InvoicePendingLinesActionInput']
      }
    }
    responses: {
      /** @description The request has succeeded and a new resource has been created as a result. */
      201: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Invoice'][]
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  getInvoice: {
    parameters: {
      query?: {
        expand?: components['schemas']['InvoiceExpand'][]
        includeDeletedLines?: boolean
      }
      header?: never
      path: {
        invoiceId: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Invoice']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  updateInvoice: {
    parameters: {
      query?: never
      header?: never
      path: {
        invoiceId: string
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['InvoiceReplaceUpdate']
      }
    }
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Invoice']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  deleteInvoice: {
    parameters: {
      query?: never
      header?: never
      path: {
        invoiceId: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description There is no content to send for this request, but the headers may be useful.  */
      204: {
        headers: {
          [name: string]: unknown
        }
        content?: never
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  advanceInvoiceAction: {
    parameters: {
      query?: never
      header?: never
      path: {
        invoiceId: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Invoice']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  approveInvoiceAction: {
    parameters: {
      query?: never
      header?: never
      path: {
        invoiceId: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Invoice']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  retryInvoiceAction: {
    parameters: {
      query?: never
      header?: never
      path: {
        invoiceId: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Invoice']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  snapshotQuantitiesInvoiceAction: {
    parameters: {
      query?: never
      header?: never
      path: {
        invoiceId: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Invoice']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  recalculateInvoiceTaxAction: {
    parameters: {
      query?: never
      header?: never
      path: {
        invoiceId: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Invoice']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  voidInvoiceAction: {
    parameters: {
      query?: never
      header?: never
      path: {
        invoiceId: string
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['VoidInvoiceActionInput']
      }
    }
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Invoice']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  listBillingProfiles: {
    parameters: {
      query?: {
        includeArchived?: boolean
        expand?: components['schemas']['BillingProfileExpand'][]
        /** @description Page index.
         *
         *     Default is 1. */
        page?: components['parameters']['Pagination.page']
        /** @description The maximum number of items per page.
         *
         *     Default is 100. */
        pageSize?: components['parameters']['Pagination.pageSize']
        /** @description The order direction. */
        order?: components['parameters']['BillingProfileOrderByOrdering.order']
        /** @description The order by field. */
        orderBy?: components['parameters']['BillingProfileOrderByOrdering.orderBy']
      }
      header?: never
      path?: never
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['BillingProfilePaginatedResponse']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  createBillingProfile: {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['BillingProfileCreate']
      }
    }
    responses: {
      /** @description The request has succeeded and a new resource has been created as a result. */
      201: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['BillingProfile']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  getBillingProfile: {
    parameters: {
      query?: {
        expand?: components['schemas']['BillingProfileExpand'][]
      }
      header?: never
      path: {
        id: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['BillingProfile']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  updateBillingProfile: {
    parameters: {
      query?: never
      header?: never
      path: {
        id: string
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['BillingProfileReplaceUpdateWithWorkflow']
      }
    }
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['BillingProfile']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  deleteBillingProfile: {
    parameters: {
      query?: never
      header?: never
      path: {
        id: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description There is no content to send for this request, but the headers may be useful.  */
      204: {
        headers: {
          [name: string]: unknown
        }
        content?: never
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  listCustomers: {
    parameters: {
      query?: {
        /** @description Page index.
         *
         *     Default is 1. */
        page?: components['parameters']['Pagination.page']
        /** @description The maximum number of items per page.
         *
         *     Default is 100. */
        pageSize?: components['parameters']['Pagination.pageSize']
        /** @description The order direction. */
        order?: components['parameters']['CustomerOrderByOrdering.order']
        /** @description The order by field. */
        orderBy?: components['parameters']['CustomerOrderByOrdering.orderBy']
        /** @description Include deleted customers. */
        includeDeleted?: components['parameters']['queryCustomerList.includeDeleted']
        /** @description Filter customers by key.
         *     Case-sensitive exact match. */
        key?: components['parameters']['queryCustomerList.key']
        /** @description Filter customers by name.
         *     Case-insensitive partial match. */
        name?: components['parameters']['queryCustomerList.name']
        /** @description Filter customers by primary email.
         *     Case-insensitive partial match. */
        primaryEmail?: components['parameters']['queryCustomerList.primaryEmail']
        /** @description Filter customers by usage attribution subject.
         *     Case-insensitive partial match. */
        subject?: components['parameters']['queryCustomerList.subject']
        /** @description Filter customers by the plan key of their susbcription. */
        planKey?: components['parameters']['queryCustomerList.planKey']
        /** @description What parts of the list output to expand in listings */
        expand?: components['parameters']['queryCustomerList.expand']
      }
      header?: never
      path?: never
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['CustomerPaginatedResponse']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  createCustomer: {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['CustomerCreate']
      }
    }
    responses: {
      /** @description The request has succeeded and a new resource has been created as a result. */
      201: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Customer']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  getCustomer: {
    parameters: {
      query?: {
        /** @description What parts of the customer output to expand */
        expand?: components['parameters']['queryCustomerGet']
      }
      header?: never
      path: {
        customerIdOrKey: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Customer']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  updateCustomer: {
    parameters: {
      query?: never
      header?: never
      path: {
        customerIdOrKey: string
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['CustomerReplaceUpdate']
      }
    }
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Customer']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  deleteCustomer: {
    parameters: {
      query?: never
      header?: never
      path: {
        customerIdOrKey: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description There is no content to send for this request, but the headers may be useful.  */
      204: {
        headers: {
          [name: string]: unknown
        }
        content?: never
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  getCustomerAccess: {
    parameters: {
      query?: never
      header?: never
      path: {
        customerIdOrKey: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['CustomerAccess']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  listCustomerAppData: {
    parameters: {
      query?: {
        /** @description Page index.
         *
         *     Default is 1. */
        page?: components['parameters']['Pagination.page']
        /** @description The maximum number of items per page.
         *
         *     Default is 100. */
        pageSize?: components['parameters']['Pagination.pageSize']
        /** @description Filter customer data by app type. */
        type?: components['parameters']['queryCustomerList.type']
      }
      header?: never
      path: {
        customerIdOrKey: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['CustomerAppDataPaginatedResponse']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  upsertCustomerAppData: {
    parameters: {
      query?: never
      header?: never
      path: {
        customerIdOrKey: string
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['CustomerAppDataCreateOrUpdateItem'][]
      }
    }
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['CustomerAppData'][]
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  deleteCustomerAppData: {
    parameters: {
      query?: never
      header?: never
      path: {
        customerIdOrKey: string
        appId: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description There is no content to send for this request, but the headers may be useful.  */
      204: {
        headers: {
          [name: string]: unknown
        }
        content?: never
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  getCustomerEntitlementValue: {
    parameters: {
      query?: {
        time?: Date | string
      }
      header?: never
      path: {
        customerIdOrKey: string
        featureKey: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['EntitlementValue']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  listCustomerSubscriptions: {
    parameters: {
      query?: {
        /** @description Page index.
         *
         *     Default is 1. */
        page?: components['parameters']['Pagination.page']
        /** @description The maximum number of items per page.
         *
         *     Default is 100. */
        pageSize?: components['parameters']['Pagination.pageSize']
      }
      header?: never
      path: {
        customerIdOrKey: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['SubscriptionPaginatedResponse']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  getDebugMetrics: {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'text/plain': string
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  listEntitlements: {
    parameters: {
      query?: {
        /** @description Filtering by multiple features.
         *
         *     Usage: `?feature=feature-1&feature=feature-2` */
        feature?: string[]
        /** @description Filtering by multiple subjects.
         *
         *     Usage: `?subject=customer-1&subject=customer-2` */
        subject?: string[]
        /** @description Filtering by multiple entitlement types.
         *
         *     Usage: `?entitlementType=metered&entitlementType=boolean` */
        entitlementType?: components['schemas']['EntitlementType'][]
        /** @description Exclude inactive entitlements in the response (those scheduled for later or earlier) */
        excludeInactive?: boolean
        /** @description Page index.
         *
         *     Default is 1. */
        page?: components['parameters']['Pagination.page']
        /** @description The maximum number of items per page.
         *
         *     Default is 100. */
        pageSize?: components['parameters']['Pagination.pageSize']
        /** @description Number of items to skip.
         *
         *     Default is 0. */
        offset?: components['parameters']['LimitOffset.offset']
        /** @description Number of items to return.
         *
         *     Default is 100. */
        limit?: components['parameters']['LimitOffset.limit']
        /** @description The order direction. */
        order?: components['parameters']['EntitlementOrderByOrdering.order']
        /** @description The order by field. */
        orderBy?: components['parameters']['EntitlementOrderByOrdering.orderBy']
      }
      header?: never
      path?: never
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['ListEntitlementsResult']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  getEntitlementById: {
    parameters: {
      query?: never
      header?: never
      path: {
        entitlementId: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Entitlement']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  listEvents: {
    parameters: {
      query?: {
        /** @description Client ID
         *     Useful to track progress of a query. */
        clientId?: string
        /** @description Start date-time in RFC 3339 format.
         *
         *     Inclusive. */
        ingestedAtFrom?: Date | string
        /** @description End date-time in RFC 3339 format.
         *
         *     Inclusive. */
        ingestedAtTo?: Date | string
        /** @description The event ID.
         *
         *     Accepts partial ID. */
        id?: string
        /** @description The event subject.
         *
         *     Accepts partial subject. */
        subject?: string
        /** @description Start date-time in RFC 3339 format.
         *
         *     Inclusive. */
        from?: Date | string
        /** @description End date-time in RFC 3339 format.
         *
         *     Inclusive. */
        to?: Date | string
        /** @description Number of events to return. */
        limit?: number
      }
      header?: never
      path?: never
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['IngestedEvent'][]
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  ingestEvents: {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    requestBody: {
      content: {
        'application/cloudevents+json': components['schemas']['Event']
        'application/cloudevents-batch+json': components['schemas']['Event'][]
        'application/json': components['schemas']['IngestEventsBody']
      }
    }
    responses: {
      /** @description There is no content to send for this request, but the headers may be useful.  */
      204: {
        headers: {
          [name: string]: unknown
        }
        content?: never
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  listFeatures: {
    parameters: {
      query?: {
        /** @description Filter by meterSlug */
        meterSlug?: string[]
        /** @description Filter by meterGroupByFilters */
        includeArchived?: boolean
        /** @description Page index.
         *
         *     Default is 1. */
        page?: components['parameters']['Pagination.page']
        /** @description The maximum number of items per page.
         *
         *     Default is 100. */
        pageSize?: components['parameters']['Pagination.pageSize']
        /** @description Number of items to skip.
         *
         *     Default is 0. */
        offset?: components['parameters']['LimitOffset.offset']
        /** @description Number of items to return.
         *
         *     Default is 100. */
        limit?: components['parameters']['LimitOffset.limit']
        /** @description The order direction. */
        order?: components['parameters']['FeatureOrderByOrdering.order']
        /** @description The order by field. */
        orderBy?: components['parameters']['FeatureOrderByOrdering.orderBy']
      }
      header?: never
      path?: never
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['ListFeaturesResult']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  createFeature: {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['FeatureCreateInputs']
      }
    }
    responses: {
      /** @description The request has succeeded and a new resource has been created as a result. */
      201: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Feature']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  getFeature: {
    parameters: {
      query?: never
      header?: never
      path: {
        featureId: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Feature']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  deleteFeature: {
    parameters: {
      query?: never
      header?: never
      path: {
        featureId: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description There is no content to send for this request, but the headers may be useful.  */
      204: {
        headers: {
          [name: string]: unknown
        }
        content?: never
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  listGrants: {
    parameters: {
      query?: {
        /** @description Filtering by multiple features.
         *
         *     Usage: `?feature=feature-1&feature=feature-2` */
        feature?: string[]
        /** @description Filtering by multiple subjects.
         *
         *     Usage: `?subject=customer-1&subject=customer-2` */
        subject?: string[]
        /** @description Include deleted */
        includeDeleted?: boolean
        /** @description Page index.
         *
         *     Default is 1. */
        page?: components['parameters']['Pagination.page']
        /** @description The maximum number of items per page.
         *
         *     Default is 100. */
        pageSize?: components['parameters']['Pagination.pageSize']
        /** @description Number of items to skip.
         *
         *     Default is 0. */
        offset?: components['parameters']['LimitOffset.offset']
        /** @description Number of items to return.
         *
         *     Default is 100. */
        limit?: components['parameters']['LimitOffset.limit']
        /** @description The order direction. */
        order?: components['parameters']['GrantOrderByOrdering.order']
        /** @description The order by field. */
        orderBy?: components['parameters']['GrantOrderByOrdering.orderBy']
      }
      header?: never
      path?: never
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json':
            | components['schemas']['EntitlementGrant'][]
            | components['schemas']['GrantPaginatedResponse']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  voidGrant: {
    parameters: {
      query?: never
      header?: never
      path: {
        grantId: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description There is no content to send for this request, but the headers may be useful.  */
      204: {
        headers: {
          [name: string]: unknown
        }
        content?: never
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description The request could not be completed due to a conflict with the current state of the target resource. */
      409: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ConflictProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  listCurrencies: {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Currency'][]
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  getProgress: {
    parameters: {
      query?: never
      header?: never
      path: {
        id: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Progress']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  listMarketplaceListings: {
    parameters: {
      query?: {
        /** @description Page index.
         *
         *     Default is 1. */
        page?: components['parameters']['Pagination.page']
        /** @description The maximum number of items per page.
         *
         *     Default is 100. */
        pageSize?: components['parameters']['Pagination.pageSize']
      }
      header?: never
      path?: never
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['MarketplaceListingPaginatedResponse']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  getMarketplaceListing: {
    parameters: {
      query?: never
      header?: never
      path: {
        type: components['schemas']['AppType']
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['MarketplaceListing']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  marketplaceAppInstall: {
    parameters: {
      query?: never
      header?: never
      path: {
        /** @description The type of the app to install. */
        type: components['parameters']['MarketplaceInstallRequest.type']
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': {
          /** @description Name of the application to install.
           *
           *     If not set defaults to the marketplace item's description. */
          name?: string
        }
      }
    }
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['MarketplaceInstallResponse']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  marketplaceAppAPIKeyInstall: {
    parameters: {
      query?: never
      header?: never
      path: {
        /** @description The type of the app to install. */
        type: components['parameters']['MarketplaceApiKeyInstallRequest.type']
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': {
          /** @description The API key for the provider.
           *     For example, the Stripe API key. */
          apiKey: string
          /** @description Name of the application to install.
           *
           *     If not set defaults to the marketplace item's description. */
          name?: string
        }
      }
    }
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['MarketplaceInstallResponse']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  marketplaceOAuth2InstallGetURL: {
    parameters: {
      query?: never
      header?: never
      path: {
        type: components['schemas']['AppType']
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['ClientAppStartResponse']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  marketplaceOAuth2InstallAuthorize: {
    parameters: {
      query?: {
        /** @description Required if the "state" parameter was present in the client authorization request.
         *     The exact value received from the client:
         *
         *     Unique, randomly generated, opaque, and non-guessable string that is sent
         *     when starting an authentication request and validated when processing the response. */
        state?: components['parameters']['OAuth2AuthorizationCodeGrantSuccessParams.state']
        /** @description Authorization code which the client will later exchange for an access token.
         *     Required with the success response. */
        code?: components['parameters']['OAuth2AuthorizationCodeGrantSuccessParams.code']
        /** @description Error code.
         *     Required with the error response. */
        error?: components['parameters']['OAuth2AuthorizationCodeGrantErrorParams.error']
        /** @description Optional human-readable text providing additional information,
         *     used to assist the client developer in understanding the error that occurred. */
        error_description?: components['parameters']['OAuth2AuthorizationCodeGrantErrorParams.error_description']
        /** @description Optional uri identifying a human-readable web page with
         *     information about the error, used to provide the client
         *     developer with additional information about the error */
        error_uri?: components['parameters']['OAuth2AuthorizationCodeGrantErrorParams.error_uri']
      }
      header?: never
      path: {
        /** @description The type of the app to install. */
        type: components['parameters']['MarketplaceOAuth2InstallAuthorizeRequest.type']
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description Redirection */
      303: {
        headers: {
          [name: string]: unknown
        }
        content?: never
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  listMeters: {
    parameters: {
      query?: {
        /** @description Page index.
         *
         *     Default is 1. */
        page?: components['parameters']['Pagination.page']
        /** @description The maximum number of items per page.
         *
         *     Default is 100. */
        pageSize?: components['parameters']['Pagination.pageSize']
        /** @description The order direction. */
        order?: components['parameters']['MeterOrderByOrdering.order']
        /** @description The order by field. */
        orderBy?: components['parameters']['MeterOrderByOrdering.orderBy']
        /** @description Include deleted meters. */
        includeDeleted?: components['parameters']['queryMeterList.includeDeleted']
      }
      header?: never
      path?: never
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Meter'][]
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  createMeter: {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['MeterCreate']
      }
    }
    responses: {
      /** @description The request has succeeded and a new resource has been created as a result. */
      201: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Meter']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  getMeter: {
    parameters: {
      query?: never
      header?: never
      path: {
        meterIdOrSlug: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Meter']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  updateMeter: {
    parameters: {
      query?: never
      header?: never
      path: {
        meterIdOrSlug: string
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['MeterUpdate']
      }
    }
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Meter']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  deleteMeter: {
    parameters: {
      query?: never
      header?: never
      path: {
        meterIdOrSlug: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description There is no content to send for this request, but the headers may be useful.  */
      204: {
        headers: {
          [name: string]: unknown
        }
        content?: never
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  queryMeter: {
    parameters: {
      query?: {
        /** @description Client ID
         *     Useful to track progress of a query. */
        clientId?: components['parameters']['MeterQuery.clientId']
        /** @description Start date-time in RFC 3339 format.
         *
         *     Inclusive.
         *
         *     For example: ?from=2025-01-01T00%3A00%3A00.000Z */
        from?: components['parameters']['MeterQuery.from']
        /** @description End date-time in RFC 3339 format.
         *
         *     Inclusive.
         *
         *     For example: ?to=2025-02-01T00%3A00%3A00.000Z */
        to?: components['parameters']['MeterQuery.to']
        /** @description If not specified, a single usage aggregate will be returned for the entirety of the specified period for each subject and group.
         *
         *     For example: ?windowSize=DAY */
        windowSize?: components['parameters']['MeterQuery.windowSize']
        /** @description The value is the name of the time zone as defined in the IANA Time Zone Database (http://www.iana.org/time-zones).
         *     If not specified, the UTC timezone will be used.
         *
         *     For example: ?windowTimeZone=UTC */
        windowTimeZone?: components['parameters']['MeterQuery.windowTimeZone']
        /** @description Filtering by multiple subjects.
         *
         *     For example: ?subject=customer-1&subject=customer-2 */
        subject?: components['parameters']['MeterQuery.subject']
        /** @description Simple filter for group bys with exact match.
         *
         *     For example: ?filterGroupBy[vendor]=openai&filterGroupBy[model]=gpt-4-turbo */
        filterGroupBy?: components['parameters']['MeterQuery.filterGroupBy']
        /** @description If not specified a single aggregate will be returned for each subject and time window.
         *     `subject` is a reserved group by value.
         *
         *     For example: ?groupBy=subject&groupBy=model */
        groupBy?: components['parameters']['MeterQuery.groupBy']
      }
      header?: never
      path: {
        meterIdOrSlug: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['MeterQueryResult']
          'text/csv': string
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  queryMeterPost: {
    parameters: {
      query?: never
      header?: never
      path: {
        meterIdOrSlug: string
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['MeterQueryRequest']
      }
    }
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['MeterQueryResult']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  listMeterSubjects: {
    parameters: {
      query?: never
      header?: never
      path: {
        meterIdOrSlug: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': string[]
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  listNotificationChannels: {
    parameters: {
      query?: {
        /** @description Include deleted notification channels in response.
         *
         *     Usage: `?includeDeleted=true` */
        includeDeleted?: boolean
        /** @description Include disabled notification channels in response.
         *
         *     Usage: `?includeDisabled=false` */
        includeDisabled?: boolean
        /** @description Page index.
         *
         *     Default is 1. */
        page?: components['parameters']['Pagination.page']
        /** @description The maximum number of items per page.
         *
         *     Default is 100. */
        pageSize?: components['parameters']['Pagination.pageSize']
        /** @description The order direction. */
        order?: components['parameters']['NotificationChannelOrderByOrdering.order']
        /** @description The order by field. */
        orderBy?: components['parameters']['NotificationChannelOrderByOrdering.orderBy']
      }
      header?: never
      path?: never
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['NotificationChannelPaginatedResponse']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  createNotificationChannel: {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['NotificationChannelCreateRequest']
      }
    }
    responses: {
      /** @description The request has succeeded and a new resource has been created as a result. */
      201: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['NotificationChannel']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  getNotificationChannel: {
    parameters: {
      query?: never
      header?: never
      path: {
        channelId: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['NotificationChannel']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  updateNotificationChannel: {
    parameters: {
      query?: never
      header?: never
      path: {
        channelId: string
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['NotificationChannelCreateRequest']
      }
    }
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['NotificationChannel']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  deleteNotificationChannel: {
    parameters: {
      query?: never
      header?: never
      path: {
        channelId: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description There is no content to send for this request, but the headers may be useful.  */
      204: {
        headers: {
          [name: string]: unknown
        }
        content?: never
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  listNotificationEvents: {
    parameters: {
      query?: {
        /** @description Start date-time in RFC 3339 format.
         *     Inclusive. */
        from?: Date | string
        /** @description End date-time in RFC 3339 format.
         *     Inclusive. */
        to?: Date | string
        /** @description Filtering by multiple feature ids or keys.
         *
         *     Usage: `?feature=feature-1&feature=feature-2` */
        feature?: string[]
        /** @description Filtering by multiple subject ids or keys.
         *
         *     Usage: `?subject=subject-1&subject=subject-2` */
        subject?: string[]
        /** @description Filtering by multiple rule ids.
         *
         *     Usage: `?rule=01J8J2XYZ2N5WBYK09EDZFBSZM&rule=01J8J4R4VZH180KRKQ63NB2VA5` */
        rule?: string[]
        /** @description Filtering by multiple channel ids.
         *
         *     Usage: `?channel=01J8J4RXH778XB056JS088PCYT&channel=01J8J4S1R1G9EVN62RG23A9M6J` */
        channel?: string[]
        /** @description Page index.
         *
         *     Default is 1. */
        page?: components['parameters']['Pagination.page']
        /** @description The maximum number of items per page.
         *
         *     Default is 100. */
        pageSize?: components['parameters']['Pagination.pageSize']
        /** @description The order direction. */
        order?: components['parameters']['NotificationEventOrderByOrdering.order']
        /** @description The order by field. */
        orderBy?: components['parameters']['NotificationEventOrderByOrdering.orderBy']
      }
      header?: never
      path?: never
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['NotificationEventPaginatedResponse']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  getNotificationEvent: {
    parameters: {
      query?: never
      header?: never
      path: {
        eventId: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['NotificationEvent']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  listNotificationRules: {
    parameters: {
      query?: {
        /** @description Include deleted notification rules in response.
         *
         *     Usage: `?includeDeleted=true` */
        includeDeleted?: boolean
        /** @description Include disabled notification rules in response.
         *
         *     Usage: `?includeDisabled=false` */
        includeDisabled?: boolean
        /** @description Filtering by multiple feature ids/keys.
         *
         *     Usage: `?feature=feature-1&feature=feature-2` */
        feature?: string[]
        /** @description Filtering by multiple notifiaction channel ids.
         *
         *     Usage: `?channel=01ARZ3NDEKTSV4RRFFQ69G5FAV&channel=01J8J2Y5X4NNGQS32CF81W95E3` */
        channel?: string[]
        /** @description Page index.
         *
         *     Default is 1. */
        page?: components['parameters']['Pagination.page']
        /** @description The maximum number of items per page.
         *
         *     Default is 100. */
        pageSize?: components['parameters']['Pagination.pageSize']
        /** @description The order direction. */
        order?: components['parameters']['NotificationRuleOrderByOrdering.order']
        /** @description The order by field. */
        orderBy?: components['parameters']['NotificationRuleOrderByOrdering.orderBy']
      }
      header?: never
      path?: never
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['NotificationRulePaginatedResponse']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  createNotificationRule: {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['NotificationRuleCreateRequest']
      }
    }
    responses: {
      /** @description The request has succeeded and a new resource has been created as a result. */
      201: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['NotificationRule']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  getNotificationRule: {
    parameters: {
      query?: never
      header?: never
      path: {
        ruleId: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['NotificationRule']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  updateNotificationRule: {
    parameters: {
      query?: never
      header?: never
      path: {
        ruleId: string
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['NotificationRuleCreateRequest']
      }
    }
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['NotificationRule']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  deleteNotificationRule: {
    parameters: {
      query?: never
      header?: never
      path: {
        ruleId: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description There is no content to send for this request, but the headers may be useful.  */
      204: {
        headers: {
          [name: string]: unknown
        }
        content?: never
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  testNotificationRule: {
    parameters: {
      query?: never
      header?: never
      path: {
        ruleId: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded and a new resource has been created as a result. */
      201: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['NotificationEvent']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  listPlans: {
    parameters: {
      query?: {
        /** @description Include deleted plans in response.
         *
         *     Usage: `?includeDeleted=true` */
        includeDeleted?: boolean
        /** @description Filter by plan.id attribute */
        id?: string[]
        /** @description Filter by plan.key attribute */
        key?: string[]
        /** @description Filter by plan.key and plan.version attributes */
        keyVersion?: {
          [key: string]: number[]
        }
        /** @description Only return plans with the given status.
         *
         *     Usage:
         *     - `?status=active`: return only the currently active plan
         *     - `?status=draft`: return only the draft plan
         *     - `?status=archived`: return only the archived plans */
        status?: components['schemas']['PlanStatus'][]
        /** @description Filter by plan.currency attribute */
        currency?: components['schemas']['CurrencyCode'][]
        /** @description Page index.
         *
         *     Default is 1. */
        page?: components['parameters']['Pagination.page']
        /** @description The maximum number of items per page.
         *
         *     Default is 100. */
        pageSize?: components['parameters']['Pagination.pageSize']
        /** @description The order direction. */
        order?: components['parameters']['PlanOrderByOrdering.order']
        /** @description The order by field. */
        orderBy?: components['parameters']['PlanOrderByOrdering.orderBy']
      }
      header?: never
      path?: never
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['PlanPaginatedResponse']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  createPlan: {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['PlanCreate']
      }
    }
    responses: {
      /** @description The request has succeeded and a new resource has been created as a result. */
      201: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Plan']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  nextPlan: {
    parameters: {
      query?: never
      header?: never
      path: {
        planIdOrKey: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded and a new resource has been created as a result. */
      201: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Plan']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  getPlan: {
    parameters: {
      query?: {
        /** @description Include latest version of the Plan instead of the version in active state.
         *
         *     Usage: `?includeLatest=true` */
        includeLatest?: boolean
      }
      header?: never
      path: {
        planId: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Plan']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  updatePlan: {
    parameters: {
      query?: never
      header?: never
      path: {
        planId: string
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['PlanReplaceUpdate']
      }
    }
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Plan']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  deletePlan: {
    parameters: {
      query?: never
      header?: never
      path: {
        planId: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description There is no content to send for this request, but the headers may be useful.  */
      204: {
        headers: {
          [name: string]: unknown
        }
        content?: never
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  listPlanAddons: {
    parameters: {
      query?: {
        /** @description Include deleted plan add-on assignments.
         *
         *     Usage: `?includeDeleted=true` */
        includeDeleted?: boolean
        /** @description Filter by addon.id attribute. */
        id?: string[]
        /** @description Filter by addon.key attribute. */
        key?: string[]
        /** @description Filter by addon.key and addon.version attributes. */
        keyVersion?: {
          [key: string]: number[]
        }
        /** @description Page index.
         *
         *     Default is 1. */
        page?: components['parameters']['Pagination.page']
        /** @description The maximum number of items per page.
         *
         *     Default is 100. */
        pageSize?: components['parameters']['Pagination.pageSize']
        /** @description The order direction. */
        order?: components['parameters']['PlanAddonOrderByOrdering.order']
        /** @description The order by field. */
        orderBy?: components['parameters']['PlanAddonOrderByOrdering.orderBy']
      }
      header?: never
      path: {
        planId: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['PlanAddonPaginatedResponse']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  createPlanAddon: {
    parameters: {
      query?: never
      header?: never
      path: {
        planId: string
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['PlanAddonCreate']
      }
    }
    responses: {
      /** @description The request has succeeded and a new resource has been created as a result. */
      201: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['PlanAddon']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description The request could not be completed due to a conflict with the current state of the target resource. */
      409: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ConflictProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  getPlanAddon: {
    parameters: {
      query?: never
      header?: never
      path: {
        planId: string
        planAddonId: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['PlanAddon']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  updatePlanAddon: {
    parameters: {
      query?: never
      header?: never
      path: {
        planId: string
        planAddonId: string
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['PlanAddonReplaceUpdate']
      }
    }
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['PlanAddon']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  deletePlanAddon: {
    parameters: {
      query?: never
      header?: never
      path: {
        planId: string
        planAddonId: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description There is no content to send for this request, but the headers may be useful.  */
      204: {
        headers: {
          [name: string]: unknown
        }
        content?: never
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  archivePlan: {
    parameters: {
      query?: never
      header?: never
      path: {
        planId: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Plan']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  publishPlan: {
    parameters: {
      query?: never
      header?: never
      path: {
        planId: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Plan']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  queryPortalMeter: {
    parameters: {
      query?: {
        /** @description Client ID
         *     Useful to track progress of a query. */
        clientId?: components['parameters']['MeterQuery.clientId']
        /** @description Start date-time in RFC 3339 format.
         *
         *     Inclusive.
         *
         *     For example: ?from=2025-01-01T00%3A00%3A00.000Z */
        from?: components['parameters']['MeterQuery.from']
        /** @description End date-time in RFC 3339 format.
         *
         *     Inclusive.
         *
         *     For example: ?to=2025-02-01T00%3A00%3A00.000Z */
        to?: components['parameters']['MeterQuery.to']
        /** @description If not specified, a single usage aggregate will be returned for the entirety of the specified period for each subject and group.
         *
         *     For example: ?windowSize=DAY */
        windowSize?: components['parameters']['MeterQuery.windowSize']
        /** @description The value is the name of the time zone as defined in the IANA Time Zone Database (http://www.iana.org/time-zones).
         *     If not specified, the UTC timezone will be used.
         *
         *     For example: ?windowTimeZone=UTC */
        windowTimeZone?: components['parameters']['MeterQuery.windowTimeZone']
        /** @description Simple filter for group bys with exact match.
         *
         *     For example: ?filterGroupBy[vendor]=openai&filterGroupBy[model]=gpt-4-turbo */
        filterGroupBy?: components['parameters']['MeterQuery.filterGroupBy']
        /** @description If not specified a single aggregate will be returned for each subject and time window.
         *     `subject` is a reserved group by value.
         *
         *     For example: ?groupBy=subject&groupBy=model */
        groupBy?: components['parameters']['MeterQuery.groupBy']
      }
      header?: never
      path: {
        meterSlug: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['MeterQueryResult']
          'text/csv': string
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  listPortalTokens: {
    parameters: {
      query?: {
        limit?: number
      }
      header?: never
      path?: never
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['PortalToken'][]
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  createPortalToken: {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['PortalToken']
      }
    }
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['PortalToken']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  invalidatePortalTokens: {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': {
          /** @description Invalidate a portal token by ID. */
          id?: string
          /** @description Invalidate all portal tokens for a subject. */
          subject?: string
        }
      }
    }
    responses: {
      /** @description There is no content to send for this request, but the headers may be useful.  */
      204: {
        headers: {
          [name: string]: unknown
        }
        content?: never
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  createStripeCheckoutSession: {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['CreateStripeCheckoutSessionRequest']
      }
    }
    responses: {
      /** @description The request has succeeded and a new resource has been created as a result. */
      201: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['CreateStripeCheckoutSessionResult']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  listSubjects: {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Subject'][]
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  upsertSubject: {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['SubjectUpsert'][]
      }
    }
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Subject'][]
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  getSubject: {
    parameters: {
      query?: never
      header?: never
      path: {
        subjectIdOrKey: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Subject']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  deleteSubject: {
    parameters: {
      query?: never
      header?: never
      path: {
        subjectIdOrKey: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description There is no content to send for this request, but the headers may be useful.  */
      204: {
        headers: {
          [name: string]: unknown
        }
        content?: never
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  listSubjectEntitlements: {
    parameters: {
      query?: {
        includeDeleted?: boolean
      }
      header?: never
      path: {
        subjectIdOrKey: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Entitlement'][]
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  createEntitlement: {
    parameters: {
      query?: never
      header?: never
      path: {
        subjectIdOrKey: string
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['EntitlementCreateInputs']
      }
    }
    responses: {
      /** @description The request has succeeded and a new resource has been created as a result. */
      201: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Entitlement']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The request could not be completed due to a conflict with the current state of the target resource. */
      409: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ConflictProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  listEntitlementGrants: {
    parameters: {
      query?: {
        includeDeleted?: boolean
        orderBy?: components['schemas']['GrantOrderBy']
      }
      header?: never
      path: {
        subjectIdOrKey: string
        entitlementIdOrFeatureKey: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['EntitlementGrant'][]
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  createGrant: {
    parameters: {
      query?: never
      header?: never
      path: {
        subjectIdOrKey: string
        entitlementIdOrFeatureKey: string
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['EntitlementGrantCreateInput']
      }
    }
    responses: {
      /** @description The request has succeeded and a new resource has been created as a result. */
      201: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['EntitlementGrant']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The request could not be completed due to a conflict with the current state of the target resource. */
      409: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ConflictProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  overrideEntitlement: {
    parameters: {
      query?: never
      header?: never
      path: {
        subjectIdOrKey: string
        entitlementIdOrFeatureKey: string
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['EntitlementCreateInputs']
      }
    }
    responses: {
      /** @description The request has succeeded and a new resource has been created as a result. */
      201: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Entitlement']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description The request could not be completed due to a conflict with the current state of the target resource. */
      409: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ConflictProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  getEntitlementValue: {
    parameters: {
      query?: {
        time?: Date | string
      }
      header?: never
      path: {
        subjectIdOrKey: string
        entitlementIdOrFeatureKey: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['EntitlementValue']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  getEntitlement: {
    parameters: {
      query?: never
      header?: never
      path: {
        subjectIdOrKey: string
        entitlementId: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Entitlement']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  deleteEntitlement: {
    parameters: {
      query?: never
      header?: never
      path: {
        subjectIdOrKey: string
        entitlementId: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description There is no content to send for this request, but the headers may be useful.  */
      204: {
        headers: {
          [name: string]: unknown
        }
        content?: never
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  getEntitlementHistory: {
    parameters: {
      query: {
        /** @description Start of time range to query entitlement: date-time in RFC 3339 format. Defaults to the last reset. Gets truncated to the granularity of the underlying meter. */
        from?: Date | string
        /** @description End of time range to query entitlement: date-time in RFC 3339 format. Defaults to now.
         *     If not now then gets truncated to the granularity of the underlying meter. */
        to?: Date | string
        /** @description Windowsize */
        windowSize: components['schemas']['WindowSize']
        /** @description The timezone used when calculating the windows. */
        windowTimeZone?: string
      }
      header?: never
      path: {
        subjectIdOrKey: string
        entitlementId: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['WindowedBalanceHistory']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  resetEntitlementUsage: {
    parameters: {
      query?: never
      header?: never
      path: {
        subjectIdOrKey: string
        entitlementId: string
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['ResetEntitlementUsageInput']
      }
    }
    responses: {
      /** @description There is no content to send for this request, but the headers may be useful.  */
      204: {
        headers: {
          [name: string]: unknown
        }
        content?: never
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  createSubscription: {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['SubscriptionCreate']
      }
    }
    responses: {
      /** @description The request has succeeded and a new resource has been created as a result. */
      201: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Subscription']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The request could not be completed due to a conflict with the current state of the target resource. */
      409: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ConflictProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  getSubscription: {
    parameters: {
      query?: {
        /** @description The time at which the subscription should be queried. If not provided the current time is used. */
        at?: Date | string
      }
      header?: never
      path: {
        subscriptionId: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['SubscriptionExpanded']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  deleteSubscription: {
    parameters: {
      query?: never
      header?: never
      path: {
        subscriptionId: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description There is no content to send for this request, but the headers may be useful.  */
      204: {
        headers: {
          [name: string]: unknown
        }
        content?: never
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  editSubscription: {
    parameters: {
      query?: never
      header?: never
      path: {
        subscriptionId: string
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['SubscriptionEdit']
      }
    }
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Subscription']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description The request could not be completed due to a conflict with the current state of the target resource. */
      409: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ConflictProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  listSubscriptionAddons: {
    parameters: {
      query?: never
      header?: never
      path: {
        subscriptionId: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['SubscriptionAddon'][]
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  createSubscriptionAddon: {
    parameters: {
      query?: never
      header?: never
      path: {
        subscriptionId: string
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['SubscriptionAddonCreate']
      }
    }
    responses: {
      /** @description The request has succeeded and a new resource has been created as a result. */
      201: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['SubscriptionAddon']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description The request could not be completed due to a conflict with the current state of the target resource. */
      409: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ConflictProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  getSubscriptionAddon: {
    parameters: {
      query?: never
      header?: never
      path: {
        subscriptionId: string
        subscriptionAddonId: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['SubscriptionAddon']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  updateSubscriptionAddon: {
    parameters: {
      query?: never
      header?: never
      path: {
        subscriptionId: string
        subscriptionAddonId: string
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['SubscriptionAddonUpdate']
      }
    }
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['SubscriptionAddon']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  cancelSubscription: {
    parameters: {
      query?: never
      header?: never
      path: {
        subscriptionId: string
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': {
          /** @description If not provided the subscription is canceled immediately. */
          timing?: components['schemas']['SubscriptionTiming']
        }
      }
    }
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Subscription']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description The request could not be completed due to a conflict with the current state of the target resource. */
      409: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ConflictProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  changeSubscription: {
    parameters: {
      query?: never
      header?: never
      path: {
        subscriptionId: string
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['SubscriptionChange']
      }
    }
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['SubscriptionChangeResponseBody']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description The request could not be completed due to a conflict with the current state of the target resource. */
      409: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ConflictProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  migrateSubscription: {
    parameters: {
      query?: never
      header?: never
      path: {
        subscriptionId: string
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': {
          /**
           * @description Timing configuration for the migration, when the migration should take effect.
           *     If not supported by the subscription, 400 will be returned.
           * @default immediate
           */
          timing?: components['schemas']['SubscriptionTiming']
          /** @description The version of the plan to migrate to.
           *     If not provided, the subscription will migrate to the latest version of the current plan. */
          targetVersion?: number
          /** @description The key of the phase to start the subscription in.
           *     If not provided, the subscription will start in the first phase of the plan. */
          startingPhase?: string
        }
      }
    }
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['SubscriptionChangeResponseBody']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description The request could not be completed due to a conflict with the current state of the target resource. */
      409: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ConflictProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  restoreSubscription: {
    parameters: {
      query?: never
      header?: never
      path: {
        subscriptionId: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Subscription']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  unscheduleCancelation: {
    parameters: {
      query?: never
      header?: never
      path: {
        subscriptionId: string
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Subscription']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description The origin server did not find a current representation for the target resource or is not willing to disclose that one exists. */
      404: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['NotFoundProblemResponse']
        }
      }
      /** @description The request could not be completed due to a conflict with the current state of the target resource. */
      409: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ConflictProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
  listEventsV2: {
    parameters: {
      query?: {
        /** @description The cursor after which to start the pagination. */
        cursor?: components['parameters']['CursorPagination.cursor']
        /** @description The limit of the pagination. */
        limit?: components['parameters']['CursorPagination.limit']
        /** @description Client ID
         *     Useful to track progress of a query. */
        clientId?: string
        /** @description The filter for the events encoded as JSON string. */
        filter?: string
      }
      header?: never
      path?: never
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['IngestedEventCursorPaginatedResponse']
        }
      }
      /** @description The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). */
      400: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['BadRequestProblemResponse']
        }
      }
      /** @description The request has not been applied because it lacks valid authentication credentials for the target resource. */
      401: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnauthorizedProblemResponse']
        }
      }
      /** @description The server understood the request but refuses to authorize it. */
      403: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ForbiddenProblemResponse']
        }
      }
      /** @description One or more conditions given in the request header fields evaluated to false when tested on the server. */
      412: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['PreconditionFailedProblemResponse']
        }
      }
      /** @description The server encountered an unexpected condition that prevented it from fulfilling the request. */
      500: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['InternalServerErrorProblemResponse']
        }
      }
      /** @description The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay. */
      503: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['ServiceUnavailableProblemResponse']
        }
      }
      /** @description An unexpected error response. */
      default: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/problem+json': components['schemas']['UnexpectedProblemResponse']
        }
      }
    }
  }
}
type WithRequired<T, K extends keyof T> = T & {
  [P in K]-?: T[P]
}
