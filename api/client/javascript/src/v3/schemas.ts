export interface paths {
  '/openmeter/addons': {
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
    get: operations['list-addons']
    put?: never
    /**
     * Create add-on
     * @description Create a new add-on.
     */
    post: operations['create-addon']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/addons/{addonId}': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * Get add-on
     * @description Get add-on by id.
     */
    get: operations['get-addon']
    /**
     * Update add-on
     * @description Update an add-on by id.
     */
    put: operations['update-addon']
    post?: never
    /**
     * Soft delete add-on
     * @description Soft delete add-on by id.
     */
    delete: operations['delete-addon']
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/addons/{addonId}/archive': {
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
     * @description Archive an add-on version.
     */
    post: operations['archive-addon']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/addons/{addonId}/publish': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /**
     * Publish add-on version
     * @description Publish an add-on version.
     */
    post: operations['publish-addon']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/apps': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * List apps
     * @description List installed apps.
     */
    get: operations['list-apps']
    put?: never
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/apps/{appId}': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * Get app
     * @description Get an installed app.
     */
    get: operations['get-app']
    put?: never
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/currencies': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * List currencies
     * @description List currencies supported by the billing system.
     */
    get: operations['list-currencies']
    put?: never
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/currencies/custom': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /**
     * Create custom currency
     * @description Create a custom currency. This operation allows defining your own custom
     *     currency for billing purposes.
     */
    post: operations['create-custom-currency']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/currencies/custom/{currencyId}/cost-bases': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * List cost bases
     * @description List cost bases for a currency. For custom currencies, there can be multiple
     *     cost bases with different `effective_from` dates.
     */
    get: operations['list-cost-bases']
    put?: never
    /**
     * Create cost basis
     * @description Create a cost basis for a currency.
     */
    post: operations['create-cost-basis']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/customers': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /** List customers */
    get: operations['list-customers']
    put?: never
    /** Create customer */
    post: operations['create-customer']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/customers/{customerId}': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /** Get customer */
    get: operations['get-customer']
    /** Upsert customer */
    put: operations['upsert-customer']
    post?: never
    /** Delete customer */
    delete: operations['delete-customer']
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/customers/{customerId}/billing': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /** Get customer billing data */
    get: operations['get-customer-billing']
    /** Update customer billing data */
    put: operations['update-customer-billing']
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/customers/{customerId}/billing/app-data': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    /** Update customer billing app data */
    put: operations['update-customer-billing-app-data']
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/customers/{customerId}/billing/stripe/checkout-sessions': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /**
     * Create Stripe Checkout Session
     * @description Create a [Stripe Checkout Session](https://docs.stripe.com/payments/checkout)
     *     for the customer.
     *
     *     Creates a Checkout Session for collecting payment method information from
     *     customers. The session operates in "setup" mode, which collects payment details
     *     without charging the customer immediately. The collected payment method can be
     *     used for future subscription billing.
     *
     *     For hosted checkout sessions, redirect customers to the returned URL. For
     *     embedded sessions, use the client_secret to initialize Stripe.js in your
     *     application.
     */
    post: operations['create-customer-stripe-checkout-session']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/customers/{customerId}/billing/stripe/portal-sessions': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /**
     * Create Stripe customer portal session
     * @description Create Stripe Customer Portal Session.
     *
     *     Useful to redirect the customer to the Stripe Customer Portal to manage their
     *     payment methods, change their billing address and access their invoice history.
     *     Only returns URL if the customer billing profile is linked to a stripe app and
     *     customer.
     */
    post: operations['create-customer-stripe-portal-session']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/customers/{customerId}/charges': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * List customer charges
     * @description List customer charges.
     *
     *     Returns the customer's charges that are represented as either flat fee or
     *     usage-based charges.
     */
    get: operations['list-customer-charges']
    put?: never
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/customers/{customerId}/credits/adjustments': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /**
     * Create a credit adjustment
     * @description A credit adjustment can be used to make manual adjustments to a customer's
     *     credit balance.
     *
     *     Supported use-cases:
     *
     *     - Usage correction
     */
    post: operations['create-credit-adjustment']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/customers/{customerId}/credits/balance': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * Get a customer's credit balance
     * @description Get a credit balance.
     */
    get: operations['get-customer-credit-balance']
    put?: never
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/customers/{customerId}/credits/grants': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * List credit grants
     * @description List credit grants.
     */
    get: operations['list-credit-grants']
    put?: never
    /**
     * Create a new credit grant
     * @description Create a new credit grant. A credit grant represents an allocation of prepaid
     *     credits to a customer.
     */
    post: operations['create-credit-grant']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/customers/{customerId}/credits/grants/{creditGrantId}': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * Get a credit grant
     * @description Get a credit grant.
     */
    get: operations['get-credit-grant']
    put?: never
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/customers/{customerId}/credits/grants/{creditGrantId}/settlement/external': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /**
     * Update credit grant external settlement status
     * @description Update the payment settlement status of an externally funded credit grant.
     *
     *     Use this endpoint to synchronize the payment state of an external payment with
     *     the system so that revenue recognition and credit availability work as expected.
     */
    post: operations['update-credit-grant-external-settlement']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/customers/{customerId}/credits/transactions': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * List credit transactions
     * @description List credit transactions for a customer.
     *
     *     Returns an immutable, chronological record of credit movements: funded credits
     *     and consumed credits. Transactions are returned in reverse chronological order
     *     by default.
     */
    get: operations['list-credit-transactions']
    put?: never
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/customers/{customerId}/entitlement-access': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /** List customer entitlement access */
    get: operations['list-customer-entitlement-access']
    put?: never
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/defaults/tax-codes': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /** Get organization default tax codes */
    get: operations['get-organization-default-tax-codes']
    /** Update organization default tax codes */
    put: operations['update-organization-default-tax-codes']
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/events': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * List metering events
     * @description List ingested events.
     */
    get: operations['list-metering-events']
    put?: never
    /**
     * Ingest metering events
     * @description Ingests an event or batch of events following the CloudEvents specification.
     */
    post: operations['ingest-metering-events']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/features': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * List features
     * @description List all features.
     */
    get: operations['list-features']
    put?: never
    /**
     * Create feature
     * @description Create a feature.
     */
    post: operations['create-feature']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/features/{featureId}': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * Get feature
     * @description Get a feature by id.
     */
    get: operations['get-feature']
    put?: never
    post?: never
    /**
     * Delete feature
     * @description Delete a feature by id.
     */
    delete: operations['delete-feature']
    options?: never
    head?: never
    /**
     * Update feature
     * @description Update a feature by id. Currently only the unit_cost field can be updated.
     */
    patch: operations['update-feature']
    trace?: never
  }
  '/openmeter/features/{featureId}/cost/query': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /**
     * Query feature cost
     * @description Query the cost of a feature.
     */
    post: operations['query-feature-cost']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/governance/query': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /**
     * Query governance access
     * @description Query feature access for a list of customers.
     *
     *     The endpoint resolves each provided identifier to a customer and returns the
     *     access status for the requested features, plus optional credit balance
     *     availability.
     *
     *     _Designed to be called on a fixed refresh interval and the query response is
     *     intended to be cached._
     */
    post: operations['query-governance-access']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/llm-cost/overrides': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * List LLM cost overrides
     * @description List per-namespace price overrides.
     */
    get: operations['list-llm-cost-overrides']
    put?: never
    /**
     * Create LLM cost override
     * @description Create a per-namespace price override.
     */
    post: operations['create-llm-cost-override']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/llm-cost/overrides/{priceId}': {
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
     * Delete LLM cost override
     * @description Delete a per-namespace price override.
     */
    delete: operations['delete-llm-cost-override']
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/llm-cost/prices': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * List LLM cost prices
     * @description List global LLM cost prices. Returns prices with overrides applied if any.
     */
    get: operations['list-llm-cost-prices']
    put?: never
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/llm-cost/prices/{priceId}': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * Get LLM cost price
     * @description Get a specific LLM cost price by ID. Returns the price with overrides applied if
     *     any.
     */
    get: operations['get-llm-cost-price']
    put?: never
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/meters': {
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
    get: operations['list-meters']
    put?: never
    /**
     * Create meter
     * @description Create a meter.
     */
    post: operations['create-meter']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/meters/{meterId}': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * Get meter
     * @description Get a meter by ID.
     */
    get: operations['get-meter']
    /**
     * Update meter
     * @description Update a meter.
     */
    put: operations['update-meter']
    post?: never
    /**
     * Delete meter
     * @description Delete a meter.
     */
    delete: operations['delete-meter']
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/meters/{meterId}/query': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /**
     * Query meter
     * @description Query a meter for usage.
     *
     *     Set `Accept: application/json` (the default) to get a structured JSON response.
     *     Set `Accept: text/csv` to download the same data as a CSV file suitable for
     *     spreadsheets. The CSV columns, in order, are:
     *
     *     `from, to, [subject,] [customer_id, customer_key, customer_name,] <dimensions...>, value`
     *
     *     The `subject` column is emitted only when `subject` is in the query's
     *     `group_by_dimensions`. The three `customer_*` columns are emitted together only
     *     when `customer_id` is in the query's `group_by_dimensions`.
     */
    post: operations['query-meter']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/plans': {
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
    get: operations['list-plans']
    put?: never
    /**
     * Create plan
     * @description Create a new plan.
     */
    post: operations['create-plan']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/plans/{planId}': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * Get plan
     * @description Get a plan by id.
     */
    get: operations['get-plan']
    /**
     * Update plan
     * @description Update a plan by id.
     */
    put: operations['update-plan']
    post?: never
    /**
     * Delete plan
     * @description Delete a plan by id.
     */
    delete: operations['delete-plan']
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/plans/{planId}/addons': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * List add-ons for plan
     * @description List add-ons associated with a plan.
     */
    get: operations['list-plan-addons']
    put?: never
    /**
     * Add add-on to plan
     * @description Add an add-on to a plan.
     */
    post: operations['create-plan-addon']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/plans/{planId}/addons/{planAddonId}': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * Get add-on association for plan
     * @description Get an add-on association for a plan.
     */
    get: operations['get-plan-addon']
    /**
     * Update add-on association for plan
     * @description Update an add-on association for a plan.
     */
    put: operations['update-plan-addon']
    post?: never
    /**
     * Remove add-on from plan
     * @description Remove an add-on from a plan.
     */
    delete: operations['delete-plan-addon']
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/plans/{planId}/archive': {
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
    post: operations['archive-plan']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/plans/{planId}/publish': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /**
     * Publish plan version
     * @description Publish a plan version.
     */
    post: operations['publish-plan']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/profiles': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * List billing profiles
     * @description List billing profiles.
     */
    get: operations['list-billing-profiles']
    put?: never
    /**
     * Create a new billing profile
     * @description Create a new billing profile.
     *
     *     Billing profiles contain the settings for billing and controls invoice
     *     generation. An organization can have multiple billing profiles defined. A
     *     billing profile is linked to a specific app. This association is established
     *     during the billing profile's creation and remains immutable.
     */
    post: operations['create-billing-profile']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/profiles/{id}': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * Get a billing profile
     * @description Get a billing profile.
     */
    get: operations['get-billing-profile']
    /**
     * Update a billing profile
     * @description Update a billing profile.
     */
    put: operations['update-billing-profile']
    post?: never
    /**
     * Delete a billing profile
     * @description Delete a billing profile.
     *
     *     Only such billing profiles can be deleted that are:
     *
     *     - not the default profile
     *     - not pinned to any customer using customer overrides
     *     - only have finalized invoices
     */
    delete: operations['delete-billing-profile']
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/subscriptions': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /** List subscriptions */
    get: operations['list-subscriptions']
    put?: never
    /** Create subscription */
    post: operations['create-subscription']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/subscriptions/{subscriptionId}': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /** Get subscription */
    get: operations['get-subscription']
    put?: never
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/subscriptions/{subscriptionId}/addons': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * List subscription addons
     * @description List the addons of a subscription.
     */
    get: operations['list-subscription-addons']
    put?: never
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/subscriptions/{subscriptionId}/addons/{subscriptionAddonId}': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /**
     * Get add-on association for subscription
     * @description Get an add-on association for a subscription.
     */
    get: operations['get-subscription-addon']
    put?: never
    post?: never
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/subscriptions/{subscriptionId}/cancel': {
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
     * @description Cancels the subscription. Will result in a scheduling conflict if there are
     *     other subscriptions scheduled to start after the cancelation time.
     */
    post: operations['cancel-subscription']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/subscriptions/{subscriptionId}/change': {
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
     * @description Closes a running subscription and starts a new one according to the
     *     specification. Can be used for upgrades, downgrades, and plan changes.
     */
    post: operations['change-subscription']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/subscriptions/{subscriptionId}/unschedule-cancelation': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    get?: never
    put?: never
    /**
     * Unschedule subscription cancelation
     * @description Unschedules the subscription cancelation.
     */
    post: operations['unschedule-cancelation']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/tax-codes': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /** List tax codes */
    get: operations['list-tax-codes']
    put?: never
    /** Create tax code */
    post: operations['create-tax-code']
    delete?: never
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
  '/openmeter/tax-codes/{taxCodeId}': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    /** Get tax code */
    get: operations['get-tax-code']
    /** Upsert tax code */
    put: operations['upsert-tax-code']
    post?: never
    /** Delete tax code */
    delete: operations['delete-tax-code']
    options?: never
    head?: never
    patch?: never
    trace?: never
  }
}
export type webhooks = Record<string, never>
export interface components {
  schemas: {
    /**
     * @description Add-on allows extending subscriptions with compatible plans with additional
     *     ratecards.
     */
    Addon: {
      readonly id: components['schemas']['ULID']
      /**
       * @description Display name of the resource.
       *
       *     Between 1 and 256 characters.
       */
      name: string
      /**
       * @description Optional description of the resource.
       *
       *     Maximum 1024 characters.
       */
      description?: string
      labels?: components['schemas']['Labels']
      /** @description An ISO-8601 timestamp representation of entity creation date. */
      readonly created_at: components['schemas']['DateTime']
      /** @description An ISO-8601 timestamp representation of entity last update date. */
      readonly updated_at: components['schemas']['DateTime']
      /** @description An ISO-8601 timestamp representation of entity deletion date. */
      readonly deleted_at?: components['schemas']['DateTime']
      /**
       * Key
       * @description A key is a semi-unique string that is used to identify the add-on. It is used to
       *     reference the latest `active` version of the add-on and is unique with the
       *     version number.
       */
      key: components['schemas']['ResourceKey']
      /**
       * Version
       * @description Version of the add-on. Incremented when the add-on is updated.
       * @default 1
       */
      readonly version: number
      /**
       * The InstanceType of the add-ons. Can be "single" or "multiple".
       * @description The InstanceType of the add-ons. Can be "single" or "multiple".
       */
      instance_type: components['schemas']['AddonInstanceType']
      /**
       * Currency
       * @description The currency code of the add-on.
       */
      currency: components['schemas']['BillingCurrencyCode']
      /**
       * Effective start date
       * @description The date and time when the add-on becomes effective. When not specified, the
       *     add-on is a draft.
       */
      readonly effective_from?: components['schemas']['DateTime']
      /**
       * Effective end date
       * @description The date and time when the add-on is no longer effective. When not specified,
       *     the add-on is effective indefinitely.
       */
      readonly effective_to?: components['schemas']['DateTime']
      /**
       * Status
       * @description The status of the add-on. Computed based on the effective start and end dates:
       *
       *     - `draft`: `effective_from` is not set.
       *     - `active`: `effective_from <= now` and (`effective_to` is not set or
       *     `now < effective_to`).
       *     - `archived`: `effective_to <= now`.
       */
      readonly status: components['schemas']['AddonStatus']
      /**
       * Rate cards
       * @description The rate cards of the add-on.
       */
      rate_cards: components['schemas']['BillingRateCard'][]
      /**
       * Validation errors
       * @description List of validation errors.
       */
      readonly validation_errors?: components['schemas']['ProductCatalogValidationError'][]
    }
    /**
     * @description The instanceType of the add-on.
     *
     *     - `single`: Can be added to a subscription only once.
     *     - `multiple`: Can be added to a subscription more than once.
     * @enum {string}
     */
    AddonInstanceType: 'single' | 'multiple'
    /** @description Page paginated response. */
    AddonPagePaginatedResponse: {
      data: components['schemas']['Addon'][]
      meta: components['schemas']['PaginatedMeta']
    }
    /** @description Addon reference. */
    AddonReference: {
      id: components['schemas']['ULID']
    }
    /** @description Addon reference. */
    AddonReferenceItem: {
      id: components['schemas']['ULID']
    }
    /**
     * @description The status of the add-on defined by the `effective_from` and `effective_to`
     *     properties.
     *
     *     - `draft`: The add-on has not yet been published and can be edited.
     *     - `active`: The add-on is published and available for use.
     *     - `archived`: The add-on is no longer available for use.
     * @enum {string}
     */
    AddonStatus: 'draft' | 'active' | 'archived'
    /** @description Address */
    Address: {
      /**
       * Country
       * @description Country code in [ISO 3166-1](https://www.iso.org/iso-3166-country-codes.html)
       *     alpha-2 format.
       */
      country?: components['schemas']['CountryCode']
      /**
       * Postal Code
       * @description Postal code.
       */
      postal_code?: string
      /**
       * State
       * @description State or province.
       */
      state?: string
      /**
       * City
       * @description City.
       */
      city?: string
      /**
       * Line 1
       * @description First line of the address.
       */
      line1?: string
      /**
       * Line 2
       * @description Second line of the address.
       */
      line2?: string
      /**
       * Phone Number
       * @description Phone number.
       */
      phone_number?: string
    }
    /** @description Page paginated response. */
    AppPagePaginatedResponse: {
      data: components['schemas']['BillingApp'][]
      meta: components['schemas']['PaginatedMeta']
    }
    /** @description Address */
    BillingAddress: {
      /**
       * Country
       * @description Country code in [ISO 3166-1](https://www.iso.org/iso-3166-country-codes.html)
       *     alpha-2 format.
       */
      country?: components['schemas']['CountryCode']
      /**
       * Postal Code
       * @description Postal code.
       */
      postal_code?: string
      /**
       * State
       * @description State or province.
       */
      state?: string
      /**
       * City
       * @description City.
       */
      city?: string
      /**
       * Line 1
       * @description First line of the address.
       */
      line1?: string
      /**
       * Line 2
       * @description Second line of the address.
       */
      line2?: string
      /**
       * Phone Number
       * @description Phone number.
       */
      phone_number?: string
    }
    /** @description Installed application. */
    BillingApp:
      | components['schemas']['BillingAppStripe']
      | components['schemas']['BillingAppSandbox']
      | components['schemas']['BillingAppExternalInvoicing']
    /**
     * @description Available apps for billing integrations to connect with third-party services.
     *     Apps can have various capabilities like syncing data from or to external
     *     systems, integrating with third-party services for tax calculation, delivery of
     *     invoices, collection of payments, etc.
     * @example {
     *       "type": "stripe",
     *       "name": "Stripe",
     *       "description": "Stripe integration allows you to collect payments with Stripe."
     *     }
     */
    BillingAppCatalogItem: {
      /** @description Type of the app. */
      readonly type: components['schemas']['BillingAppType']
      /** @description Name of the app. */
      readonly name: string
      /** @description Description of the app. */
      readonly description: string
    }
    /** @description App customer data. */
    BillingAppCustomerData: {
      /**
       * Stripe
       * @description Used if the customer has a linked Stripe app.
       */
      stripe?: components['schemas']['BillingAppCustomerDataStripe']
      /**
       * External invoicing
       * @description Used if the customer has a linked external invoicing app.
       */
      external_invoicing?: components['schemas']['BillingAppCustomerDataExternalInvoicing']
    }
    /** @description External invoicing customer data. */
    BillingAppCustomerDataExternalInvoicing: {
      /**
       * Labels
       * @description Labels for this external invoicing integration on the customer.
       */
      labels?: components['schemas']['Labels']
    }
    /** @description Stripe customer data. */
    BillingAppCustomerDataStripe: {
      /**
       * Stripe customer ID
       * @description The Stripe customer ID used.
       * @example cus_1234567890
       */
      customer_id?: string
      /**
       * Stripe default payment method ID
       * @description The Stripe default payment method ID.
       * @example pm_1234567890
       */
      default_payment_method_id?: string
      /**
       * Labels
       * @description Labels for this Stripe integration on the customer.
       */
      labels?: components['schemas']['Labels']
    }
    /**
     * @description External Invoicing app enables integration with third-party invoicing or payment
     *     system.
     *
     *     The app supports a bi-directional synchronization pattern where OpenMeter
     *     Billing manages the invoice lifecycle while the external system handles invoice
     *     presentation and payment collection.
     *
     *     Integration workflow:
     *
     *     1. The billing system creates invoices and transitions them through lifecycle
     *     states (draft → issuing → issued)
     *     2. The integration receives webhook notifications about invoice state changes
     *     3. The integration calls back to provide external system IDs and metadata
     *     4. The integration reports payment events back via the payment status API
     *
     *     State synchronization is controlled by hooks that pause invoice progression
     *     until the external system confirms synchronization via API callbacks.
     */
    BillingAppExternalInvoicing: {
      readonly id: components['schemas']['ULID']
      /**
       * @description Display name of the resource.
       *
       *     Between 1 and 256 characters.
       */
      name: string
      /**
       * @description Optional description of the resource.
       *
       *     Maximum 1024 characters.
       */
      description?: string
      labels?: components['schemas']['Labels']
      /** @description An ISO-8601 timestamp representation of entity creation date. */
      readonly created_at: components['schemas']['DateTime']
      /** @description An ISO-8601 timestamp representation of entity last update date. */
      readonly updated_at: components['schemas']['DateTime']
      /** @description An ISO-8601 timestamp representation of entity deletion date. */
      readonly deleted_at?: components['schemas']['DateTime']
      /**
       * @description The app type. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'external_invoicing'
      /** @description The app catalog definition that this installed app is based on. */
      readonly definition: components['schemas']['BillingAppCatalogItem']
      /** @description Status of the app connection. */
      readonly status: components['schemas']['BillingAppStatus']
      /**
       * @description Enable draft synchronization hook.
       *
       *     When enabled, invoices will pause at the draft state and wait for the
       *     integration to call the draft synchronized endpoint before progressing to the
       *     issuing state. This allows the external system to validate and prepare the
       *     invoice data.
       *
       *     When disabled, invoices automatically progress through the draft state based on
       *     the configured workflow timing.
       */
      enable_draft_sync_hook: boolean
      /**
       * @description Enable issuing synchronization hook.
       *
       *     When enabled, invoices will pause at the issuing state and wait for the
       *     integration to call the issuing synchronized endpoint before progressing to the
       *     issued state. This ensures the external invoicing system has successfully
       *     created and finalized the invoice before it is marked as issued.
       *
       *     When disabled, invoices automatically progress through the issuing state and are
       *     immediately marked as issued.
       */
      enable_issuing_sync_hook: boolean
    }
    /** @description App reference. */
    BillingAppReference: {
      /** @description The ID of the app. */
      id: components['schemas']['ULID']
    }
    /** @description Sandbox app can be used for testing billing features. */
    BillingAppSandbox: {
      readonly id: components['schemas']['ULID']
      /**
       * @description Display name of the resource.
       *
       *     Between 1 and 256 characters.
       */
      name: string
      /**
       * @description Optional description of the resource.
       *
       *     Maximum 1024 characters.
       */
      description?: string
      labels?: components['schemas']['Labels']
      /** @description An ISO-8601 timestamp representation of entity creation date. */
      readonly created_at: components['schemas']['DateTime']
      /** @description An ISO-8601 timestamp representation of entity last update date. */
      readonly updated_at: components['schemas']['DateTime']
      /** @description An ISO-8601 timestamp representation of entity deletion date. */
      readonly deleted_at?: components['schemas']['DateTime']
      /**
       * @description The app type. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'sandbox'
      /** @description The app catalog definition that this installed app is based on. */
      readonly definition: components['schemas']['BillingAppCatalogItem']
      /** @description Status of the app connection. */
      readonly status: components['schemas']['BillingAppStatus']
    }
    /**
     * @description Connection status of an installed app.
     * @enum {string}
     */
    BillingAppStatus: 'ready' | 'unauthorized'
    /** @description Stripe app. */
    BillingAppStripe: {
      readonly id: components['schemas']['ULID']
      /**
       * @description Display name of the resource.
       *
       *     Between 1 and 256 characters.
       */
      name: string
      /**
       * @description Optional description of the resource.
       *
       *     Maximum 1024 characters.
       */
      description?: string
      labels?: components['schemas']['Labels']
      /** @description An ISO-8601 timestamp representation of entity creation date. */
      readonly created_at: components['schemas']['DateTime']
      /** @description An ISO-8601 timestamp representation of entity last update date. */
      readonly updated_at: components['schemas']['DateTime']
      /** @description An ISO-8601 timestamp representation of entity deletion date. */
      readonly deleted_at?: components['schemas']['DateTime']
      /**
       * @description The app type. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'stripe'
      /** @description The app catalog definition that this installed app is based on. */
      readonly definition: components['schemas']['BillingAppCatalogItem']
      /** @description Status of the app connection. */
      readonly status: components['schemas']['BillingAppStatus']
      /** @description The Stripe account ID associated with the connected Stripe account. */
      readonly account_id: string
      /** @description Indicates whether the app is connected to a live Stripe account. */
      readonly livemode: boolean
      /** @description The masked Stripe API key that only exposes the first and last few characters. */
      readonly masked_api_key: string
    }
    /** @description Custom text displayed at various stages of the checkout flow. */
    BillingAppStripeCheckoutSessionCustomTextParams: {
      /** @description Text displayed after the payment confirmation button. */
      after_submit?: {
        /** @description The custom message text (max 1200 characters). */
        message?: string
      }
      /** @description Text displayed alongside shipping address collection. */
      shipping_address?: {
        /** @description The custom message text (max 1200 characters). */
        message?: string
      }
      /** @description Text displayed alongside the payment confirmation button. */
      submit?: {
        /** @description The custom message text (max 1200 characters). */
        message?: string
      }
      /** @description Text replacing the default terms of service agreement text. */
      terms_of_service_acceptance?: {
        /** @description The custom message text (max 1200 characters). */
        message?: string
      }
    }
    /**
     * @description Stripe Checkout Session mode.
     *
     *     Determines the primary purpose of the checkout session.
     * @enum {string}
     */
    BillingAppStripeCheckoutSessionMode: 'setup'
    /**
     * @description Checkout Session UI mode.
     * @enum {string}
     */
    BillingAppStripeCheckoutSessionUIMode: 'embedded' | 'hosted'
    /**
     * @description Controls whether Checkout collects the customer's billing address.
     * @enum {string}
     */
    BillingAppStripeCreateCheckoutSessionBillingAddressCollection:
      | 'auto'
      | 'required'
    /** @description Checkout Session consent collection configuration. */
    BillingAppStripeCreateCheckoutSessionConsentCollection: {
      /** @description Controls the visibility of payment method reuse agreement. */
      payment_method_reuse_agreement?: components['schemas']['BillingAppStripeCreateCheckoutSessionConsentCollectionPaymentMethodReuseAgreement']
      /**
       * @description Enables collection of promotional communication consent.
       *
       *     Only available to US merchants. When set to "auto", Checkout determines whether
       *     to show the option based on the customer's locale.
       */
      promotions?: components['schemas']['BillingAppStripeCreateCheckoutSessionConsentCollectionPromotions']
      /**
       * @description Requires customers to accept terms of service before payment.
       *
       *     Requires a valid terms of service URL in your Stripe Dashboard settings.
       */
      terms_of_service?: components['schemas']['BillingAppStripeCreateCheckoutSessionConsentCollectionTermsOfService']
    }
    /** @description Payment method reuse agreement configuration. */
    BillingAppStripeCreateCheckoutSessionConsentCollectionPaymentMethodReuseAgreement: {
      /** @description Position and visibility of the payment method reuse agreement. */
      position?: components['schemas']['BillingAppStripeCreateCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPosition']
    }
    /**
     * @description Position of payment method reuse agreement in the UI.
     * @enum {string}
     */
    BillingAppStripeCreateCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPosition:
      | 'auto'
      | 'hidden'
    /**
     * @description Promotional communication consent collection setting.
     * @enum {string}
     */
    BillingAppStripeCreateCheckoutSessionConsentCollectionPromotions:
      | 'auto'
      | 'none'
    /**
     * @description Terms of service acceptance requirement.
     * @enum {string}
     */
    BillingAppStripeCreateCheckoutSessionConsentCollectionTermsOfService:
      | 'none'
      | 'required'
    /** @description Controls which customer fields can be updated by the checkout session. */
    BillingAppStripeCreateCheckoutSessionCustomerUpdate: {
      /**
       * @description Whether to save the billing address to customer.address.
       *
       *     Defaults to "never".
       * @default never
       */
      address?: components['schemas']['BillingAppStripeCreateCheckoutSessionCustomerUpdateBehavior']
      /**
       * @description Whether to save the customer name to customer.name.
       *
       *     Defaults to "never".
       * @default never
       */
      name?: components['schemas']['BillingAppStripeCreateCheckoutSessionCustomerUpdateBehavior']
      /**
       * @description Whether to save shipping information to customer.shipping.
       *
       *     Defaults to "never".
       * @default never
       */
      shipping?: components['schemas']['BillingAppStripeCreateCheckoutSessionCustomerUpdateBehavior']
    }
    /**
     * @description Behavior for updating customer fields from checkout session.
     * @enum {string}
     */
    BillingAppStripeCreateCheckoutSessionCustomerUpdateBehavior:
      | 'auto'
      | 'never'
    /**
     * @description Redirect behavior for embedded checkout sessions.
     * @enum {string}
     */
    BillingAppStripeCreateCheckoutSessionRedirectOnCompletion:
      | 'always'
      | 'if_required'
      | 'never'
    /**
     * @description Configuration options for creating a Stripe Checkout Session.
     *
     *     Based on Stripe's
     *     [Checkout Session API parameters](https://docs.stripe.com/api/checkout/sessions/create).
     */
    BillingAppStripeCreateCheckoutSessionRequestOptions: {
      /**
       * @description Whether to collect the customer's billing address.
       *
       *     Defaults to auto, which only collects the address when necessary for tax
       *     calculation.
       * @default auto
       */
      billing_address_collection?: components['schemas']['BillingAppStripeCreateCheckoutSessionBillingAddressCollection']
      /**
       * @description URL to redirect customers who cancel the checkout session.
       *
       *     Not allowed when ui_mode is "embedded".
       */
      cancel_url?: string
      /**
       * @description Unique reference string for reconciling sessions with internal systems.
       *
       *     Can be a customer ID, cart ID, or any other identifier.
       */
      client_reference_id?: string
      /** @description Controls which customer fields can be updated by the checkout session. */
      customer_update?: components['schemas']['BillingAppStripeCreateCheckoutSessionCustomerUpdate']
      /** @description Configuration for collecting customer consent during checkout. */
      consent_collection?: components['schemas']['BillingAppStripeCreateCheckoutSessionConsentCollection']
      /**
       * @description Three-letter ISO 4217 currency code in uppercase.
       *
       *     Required for payment mode sessions. Optional for setup mode sessions.
       */
      currency?: components['schemas']['CurrencyCode']
      /** @description Custom text to display during checkout at various stages. */
      custom_text?: components['schemas']['BillingAppStripeCheckoutSessionCustomTextParams']
      /**
       * Format: int64
       * @description Unix timestamp when the checkout session expires.
       *
       *     Can be 30 minutes to 24 hours from creation. Defaults to 24 hours.
       */
      expires_at?: number
      /**
       * @description IETF language tag for the checkout UI locale.
       *
       *     If blank or "auto", uses the browser's locale. Example: "en", "fr", "de".
       */
      locale?: string
      /**
       * @description Set of key-value pairs to attach to the checkout session.
       *
       *     Useful for storing additional structured information.
       */
      metadata?: {
        [key: string]: string
      }
      /**
       * @description Return URL for embedded checkout sessions after payment authentication.
       *
       *     Required if ui_mode is "embedded" and redirect-based payment methods are
       *     enabled.
       */
      return_url?: string
      /**
       * @description Success URL to redirect customers after completing payment or setup.
       *
       *     Not allowed when ui_mode is "embedded". See:
       *     https://docs.stripe.com/payments/checkout/custom-success-page
       */
      success_url?: string
      /**
       * @description The UI mode for the checkout session.
       *
       *     "hosted" displays a Stripe-hosted page. "embedded" integrates directly into your
       *     app. Defaults to "hosted".
       * @default hosted
       */
      ui_mode?: components['schemas']['BillingAppStripeCheckoutSessionUIMode']
      /**
       * @description List of payment method types to enable (e.g., "card", "us_bank_account").
       *
       *     If not specified, Stripe enables all relevant payment methods.
       */
      payment_method_types?: string[]
      /**
       * @description Redirect behavior for embedded checkout sessions.
       *
       *     Controls when to redirect users after completion. See:
       *     https://docs.stripe.com/payments/checkout/custom-success-page?payment-ui=embedded-form
       */
      redirect_on_completion?: components['schemas']['BillingAppStripeCreateCheckoutSessionRedirectOnCompletion']
      /** @description Configuration for collecting tax IDs during checkout. */
      tax_id_collection?: components['schemas']['BillingAppStripeCreateCheckoutSessionTaxIdCollection']
    }
    /**
     * @description Result of creating a Stripe Checkout Session.
     *
     *     Contains all the information needed to redirect customers to the checkout or
     *     initialize an embedded checkout flow.
     */
    BillingAppStripeCreateCheckoutSessionResult: {
      /** @description The customer ID in the billing system. */
      customer_id: components['schemas']['ULID']
      /** @description The Stripe customer ID. */
      stripe_customer_id: string
      /** @description The Stripe checkout session ID. */
      session_id: string
      /** @description The setup intent ID created for collecting the payment method. */
      setup_intent_id: string
      /**
       * @description Client secret for initializing Stripe.js on the client side.
       *
       *     Required for embedded checkout sessions. See:
       *     https://docs.stripe.com/payments/checkout/custom-success-page
       */
      client_secret?: string
      /**
       * @description The client reference ID provided in the request.
       *
       *     Useful for reconciling the session with your internal systems.
       */
      client_reference_id?: string
      /** @description Customer's email address if provided to Stripe. */
      customer_email?: string
      /** @description Currency code for the checkout session. */
      currency?: components['schemas']['CurrencyCode']
      /** @description Timestamp when the checkout session was created. */
      created_at: components['schemas']['DateTime']
      /** @description Timestamp when the checkout session will expire. */
      expires_at?: components['schemas']['DateTime']
      /** @description Metadata attached to the checkout session. */
      metadata?: {
        [key: string]: string
      }
      /**
       * @description The status of the checkout session.
       *
       *     See:
       *     https://docs.stripe.com/api/checkout/sessions/object#checkout_session_object-status
       */
      status?: string
      /** @description URL to redirect customers to the checkout page (for hosted mode). */
      url?: string
      /**
       * @description Mode of the checkout session.
       *
       *     Currently only "setup" mode is supported for collecting payment methods.
       */
      mode: components['schemas']['BillingAppStripeCheckoutSessionMode']
      /** @description The cancel URL where customers are redirected if they cancel. */
      cancel_url?: string
      /** @description The success URL where customers are redirected after completion. */
      success_url?: string
      /** @description The return URL for embedded sessions after authentication. */
      return_url?: string
    }
    /** @description Tax ID collection configuration for checkout sessions. */
    BillingAppStripeCreateCheckoutSessionTaxIdCollection: {
      /**
       * @description Enable tax ID collection during checkout.
       *
       *     Defaults to false.
       * @default false
       */
      enabled?: boolean
      /**
       * @description Whether tax ID collection is required.
       *
       *     Defaults to "never".
       * @default never
       */
      required?: components['schemas']['BillingAppStripeCreateCheckoutSessionTaxIdCollectionRequired']
    }
    /**
     * @description Tax ID collection requirement level.
     * @enum {string}
     */
    BillingAppStripeCreateCheckoutSessionTaxIdCollectionRequired:
      | 'if_supported'
      | 'never'
    /** @description Request to create a Stripe Customer Portal Session. */
    BillingAppStripeCreateCustomerPortalSessionOptions: {
      /**
       * @description The ID of an existing
       *     [Stripe configuration](https://docs.stripe.com/api/customer_portal/configurations)
       *     to use for this session, describing its functionality and features. If not
       *     specified, the session uses the default configuration.
       */
      configuration_id?: string
      /**
       * @description The IETF
       *     [language tag](https://docs.stripe.com/api/customer_portal/sessions/create#create_portal_session-locale)
       *     of the locale customer portal is displayed in. If blank or `auto`, the
       *     customer's preferred_locales or browser's locale is used.
       */
      locale?: string
      /**
       * @description The
       *     [URL to redirect](https://docs.stripe.com/api/customer_portal/sessions/create#create_portal_session-return_url)
       *     the customer to after they have completed their requested actions.
       */
      return_url?: string
    }
    /**
     * @description Result of creating a
     *     [Stripe Customer Portal Session](https://docs.stripe.com/api/customer_portal/sessions/object).
     *
     *     Contains all the information needed to redirect the customer to the Stripe
     *     Customer Portal.
     */
    BillingAppStripeCreateCustomerPortalSessionResult: {
      /**
       * @description The ID of the customer portal session.
       *
       *     See:
       *     https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-id
       */
      id: string
      /** @description The ID of the stripe customer. */
      stripe_customer_id: string
      /**
       * @description Configuration used to customize the customer portal.
       *
       *     See:
       *     https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-configuration
       */
      configuration_id: string
      /**
       * @description Livemode.
       *
       *     See:
       *     https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-livemode
       */
      livemode: boolean
      /**
       * @description Created at.
       *
       *     See:
       *     https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-created
       */
      created_at: components['schemas']['DateTime']
      /**
       * @description Return URL.
       *
       *     See:
       *     https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-return_url
       */
      return_url: string
      /**
       * @description The IETF language tag of the locale customer portal is displayed in.
       *
       *     See:
       *     https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-locale
       */
      locale: string
      /**
       * @description The URL to redirect the customer to after they have completed their requested
       *     actions.
       */
      url: string
    }
    /**
     * @description The type of the app.
     * @enum {string}
     */
    BillingAppType: 'sandbox' | 'stripe' | 'external_invoicing'
    /**
     * Customer charge
     * @description Customer charge.
     */
    BillingCharge:
      | components['schemas']['BillingFlatFeeCharge']
      | components['schemas']['BillingUsageBasedCharge']
    /**
     * Charge status
     * @description Lifecycle status of a charge.
     *
     *     Values:
     *
     *     - `created`: The charge has been created but is not active yet.
     *     - `active`: The charge is active.
     *     - `final`: The charge is fully finalized and no further changes are expected.
     *     - `deleted`: The charge has been deleted.
     * @enum {string}
     */
    BillingChargeStatus: 'created' | 'active' | 'final' | 'deleted'
    /**
     * @description The totals of a change.
     *
     *     RealTime is only expanded when the `real_time_usage` expand is used.
     */
    BillingChargeTotals: {
      /**
       * Booked
       * @description The amount of the charge already booked to the internal accounting system.
       */
      readonly booked: components['schemas']['BillingTotals']
      /**
       * Realtime totals
       * @description The realtime amount of the charge.
       *
       *     Requires the `realtime_usage` expand.
       */
      readonly realtime?: components['schemas']['BillingTotals']
    }
    /**
     * Customer charge expands
     * @description Expands for customer charges.
     *
     *     Values:
     *
     *     - `real_time_usage`: The charge's real-time usage.
     * @enum {string}
     */
    BillingChargesExpand: 'real_time_usage'
    /** @description Describes currency basis supported by billing system. */
    BillingCostBasis: {
      readonly id: components['schemas']['ULID']
      /** @description The fiat currency code for the cost basis. */
      fiat_code: components['schemas']['CurrencyCode']
      /** @description The cost rate for the currency. */
      rate: components['schemas']['Numeric']
      /**
       * @description An ISO-8601 timestamp representation of the date from which the cost basis is
       *     effective. If not provided, it will be effective immediately and will be set to
       *     `now` by the system.
       */
      effective_from?: components['schemas']['DateTime']
      /** @description An ISO-8601 timestamp representation of entity creation date. */
      readonly created_at: components['schemas']['DateTime']
    }
    /**
     * Credit adjustment
     * @description A credit adjustment can be used to make manual adjustments to a customer's
     *     credit balance.
     *
     *     Supported use-cases:
     *
     *     - Usage correction
     */
    BillingCreditAdjustment: {
      /**
       * @description Display name of the resource.
       *
       *     Between 1 and 256 characters.
       */
      name: string
      /**
       * @description Optional description of the resource.
       *
       *     Maximum 1024 characters.
       */
      description?: string
      labels?: components['schemas']['Labels']
    }
    /**
     * Credit availability policy
     * @description When credits become available for consumption.
     *
     *     - `on_creation`: Credits are available as soon as the grant is created.
     *     - `on_authorization`: Credits are available once the payment is authorized.
     *     - `on_settlement`: Credits are available once the payment is settled.
     * @enum {string}
     */
    BillingCreditAvailabilityPolicy: 'on_creation'
    /**
     * Credit balances
     * @description The balances of the credits of a customer.
     */
    BillingCreditBalances: {
      /** @description The timestamp of the balance retrieval. */
      readonly retrieved_at: components['schemas']['DateTime']
      /** @description The balances by currencies. */
      readonly balances: components['schemas']['CreditBalance'][]
    }
    /**
     * Credit funding method
     * @description The funding method describes how the grant is funded.
     *
     *     - `none`: No funding workflow applies, for example promotional grants
     *     - `invoice`: The grant is funded by an in-system invoice flow
     *     - `external`: The grant is funded outside the system (e.g., wire transfer,
     *     external invoice, or manual reconciliation)
     * @enum {string}
     */
    BillingCreditFundingMethod: 'none' | 'invoice' | 'external'
    /**
     * Credit grant
     * @description A credit grant allocates credits to a customer.
     *
     *     Credits are drawn down against charges according to the settlement mode
     *     configured on the rate card.
     */
    BillingCreditGrant: {
      readonly id: components['schemas']['ULID']
      /**
       * @description Display name of the resource.
       *
       *     Between 1 and 256 characters.
       */
      name: string
      /**
       * @description Optional description of the resource.
       *
       *     Maximum 1024 characters.
       */
      description?: string
      labels?: components['schemas']['Labels']
      /** @description An ISO-8601 timestamp representation of entity creation date. */
      readonly created_at: components['schemas']['DateTime']
      /** @description An ISO-8601 timestamp representation of entity last update date. */
      readonly updated_at: components['schemas']['DateTime']
      /** @description An ISO-8601 timestamp representation of entity deletion date. */
      readonly deleted_at?: components['schemas']['DateTime']
      /** @description Funding method of the grant. */
      funding_method: components['schemas']['BillingCreditFundingMethod']
      /** @description The currency of the granted credits. */
      currency: components['schemas']['BillingCurrencyCode']
      /** @description Granted credit amount. */
      amount: components['schemas']['Numeric']
      /** @description Present when a funding workflow applies (funding_method is not `none`). */
      purchase?: components['schemas']['BillingCreditGrantPurchase']
      /**
       * @description Tax configuration for the grant.
       *
       *     For `invoice` and `external` funding methods, tax configuration should be
       *     provided to ensure correct revenue recognition. When not provided, the default
       *     credit grant tax code is applied, if that's not set the global default taxcode
       *     is used.
       */
      tax_config?: components['schemas']['BillingCreditGrantTaxConfig']
      /** @description Available when `funding_method` is `invoice`. */
      readonly invoice?: components['schemas']['BillingCreditGrantInvoiceReference']
      filters?: components['schemas']['BillingCreditGrantFilters']
      /**
       * Format: int16
       * @description Draw-down priority of the grant. Lower values have higher priority.
       * @default 10
       */
      priority?: number
      /**
       * @description The timestamp when the credit grant expires.
       *
       *     Calculated from the grant effective time and `expires_after` if provided.
       */
      readonly expires_at?: components['schemas']['DateTime']
      /** @description Timestamp when the grant was voided. */
      readonly voided_at?: components['schemas']['DateTime']
      /** @description Current lifecycle status of the grant. */
      readonly status: components['schemas']['BillingCreditGrantStatus']
    }
    /** @description Filters for the credit grant. */
    BillingCreditGrantFilters: {
      /**
       * @description Limit the credit grant to specific features. If no features are specified, the
       *     credit grant can be used for any feature.
       * @example [
       *       "input_tokens",
       *       "output_tokens"
       *     ]
       */
      features?: components['schemas']['ResourceKey'][]
    }
    /** @description Invoice references for the grant. */
    BillingCreditGrantInvoiceReference: {
      /** @description Identifier of the invoice associated with the grant. */
      readonly id?: components['schemas']['ULID']
      /** @description Identifier of the invoice line associated with the grant. */
      readonly line?: {
        id: components['schemas']['ULID']
      }
    }
    /** @description Purchase and payment terms of the grant. */
    BillingCreditGrantPurchase: {
      /** @description Currency of the purchase amount. */
      currency: components['schemas']['CurrencyCode']
      /**
       * @description Cost basis per credit unit used to calculate the purchase amount.
       *
       *     If `per_unit_cost_basis` is 0.50 and credit amount is $100.00, the total charge
       *     is $50.00. The value must be greater than 0. If the cost basis is 0, use
       *     `funding_method=none` instead.
       *
       *     Defaults to 1.0.
       * @default 1.0
       */
      per_unit_cost_basis?: components['schemas']['Numeric']
      /** @description The purchase amount. Calculated from `per_unit_cost_basis` and credit `amount`. */
      readonly amount: components['schemas']['Numeric']
      /**
       * @description Controls when credits become available for consumption.
       *
       *     Defaults to `on_creation`.
       * @default on_creation
       */
      availability_policy?: components['schemas']['BillingCreditAvailabilityPolicy']
      /** @description Current payment settlement status. */
      readonly settlement_status?: components['schemas']['BillingCreditPurchasePaymentSettlementStatus']
    }
    /**
     * Credit grant lifecycle status
     * @description Credit grant lifecycle status.
     *
     *     - `pending`: The credit block has been created but is not yet valid.
     *     (`effective_at` is in the future or availability_policy is not met)
     *     - `active`: The credit block is currently valid and eligible for consumption.
     *     (`effective_at` is in the past, `expires_at` is in the future and
     *     availability_policy is met)
     *     - `expired`: The credit block expired with remaining unused balance,
     *     `expires_at` time has passed.
     *     - `voided`: The credit block was voided. Remaining balance is forfeited.
     * @enum {string}
     */
    BillingCreditGrantStatus: 'pending' | 'active' | 'expired' | 'voided'
    /**
     * Tax configuration for a credit grant
     * @description Tax configuration for a credit grant.
     *
     *     Tax configuration should be provided to ensure correct revenue recognition,
     *     including for externally funded grants.
     */
    BillingCreditGrantTaxConfig: {
      /** @description Tax behavior applied to the invoice line item. */
      behavior?: components['schemas']['BillingTaxBehavior']
      /** @description Tax code applied to the invoice line item. */
      tax_code?: components['schemas']['TaxCodeReference']
    }
    /**
     * Credit purchase payment settlement status
     * @description Credit purchase payment settlement status.
     *
     *     - `pending`: Payment has been initiated and is not yet authorized.
     *     - `authorized`: Payment has been authorized.
     *     - `settled`: Payment has been settled.
     * @enum {string}
     */
    BillingCreditPurchasePaymentSettlementStatus:
      | 'pending'
      | 'authorized'
      | 'settled'
    /**
     * Credit transaction
     * @description A credit transaction represents a single credit movement on the customer's
     *     balance.
     *
     *     Credit transactions are immutable.
     */
    BillingCreditTransaction: {
      readonly id: components['schemas']['ULID']
      /**
       * @description Display name of the resource.
       *
       *     Between 1 and 256 characters.
       */
      name: string
      /**
       * @description Optional description of the resource.
       *
       *     Maximum 1024 characters.
       */
      description?: string
      labels?: components['schemas']['Labels']
      /** @description An ISO-8601 timestamp representation of entity creation date. */
      readonly created_at: components['schemas']['DateTime']
      /** @description The date and time the transaction was booked. */
      readonly booked_at: components['schemas']['DateTime']
      /** @description The type of credit transaction. */
      readonly type: components['schemas']['BillingCreditTransactionType']
      /** @description Currency of the balance affected by the transaction. */
      readonly currency: components['schemas']['BillingCurrencyCode']
      /**
       * @description Signed amount of the credit movement. Positive values add balance, negative
       *     values reduce balance.
       */
      readonly amount: components['schemas']['Numeric']
      /** @description The available balance before and after the transaction. */
      readonly available_balance: {
        before: components['schemas']['Numeric']
        after: components['schemas']['Numeric']
      }
    }
    /**
     * @description The type of the credit transaction.
     *
     *     - `funded`: Credit granted and available for consumption.
     *     - `consumed`: Credit consumed by usage or fees.
     *     - `expired`: Credit removed because it expired before being used.
     * @enum {string}
     */
    BillingCreditTransactionType: 'funded' | 'consumed' | 'expired'
    /** @description Fiat or custom currency. */
    BillingCurrency:
      | components['schemas']['BillingCurrencyFiat']
      | components['schemas']['BillingCurrencyCustom']
    /** @description Fiat or custom currency code. */
    BillingCurrencyCode: string & components['schemas']['CurrencyCode']
    /**
     * @description Custom currency code. It should be a unique code but not conflicting with any
     *     existing fiat currency codes.
     */
    BillingCurrencyCodeCustom: string
    /** @description Describes custom currency. */
    BillingCurrencyCustom: {
      readonly id: components['schemas']['ULID']
      /**
       * @description The type of the currency. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'custom'
      /**
       * @description The name of the currency. It should be a human-readable string that represents
       *     the name of the currency, such as "US Dollar" or "Euro".
       */
      name: string
      /** @description Description of the currency. */
      description?: string
      /**
       * @description The symbol of the currency. It should be a string that represents the symbol of
       *     the currency, such as "$" for US Dollar or "€" for Euro.
       */
      symbol?: string
      code: components['schemas']['BillingCurrencyCodeCustom']
      /** @description An ISO-8601 timestamp representation of the custom currency creation date. */
      readonly created_at: components['schemas']['DateTime']
    }
    /** @description Currency describes a currency supported by the billing system. */
    BillingCurrencyFiat: {
      readonly id: components['schemas']['ULID']
      /**
       * @description The type of the currency. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'fiat'
      /**
       * @description The name of the currency. It should be a human-readable string that represents
       *     the name of the currency, such as "US Dollar" or "Euro".
       */
      name: string
      /** @description Description of the currency. */
      description?: string
      /**
       * @description The symbol of the currency. It should be a string that represents the symbol of
       *     the currency, such as "$" for US Dollar or "€" for Euro.
       */
      symbol?: string
      readonly code: components['schemas']['CurrencyCode']
    }
    /**
     * @description Currency type for custom currencies. It should be a unique code but not
     *     conflicting with any existing standard currency codes.
     * @enum {string}
     */
    BillingCurrencyType: 'fiat' | 'custom'
    /**
     * @description Customers can be individuals or organizations that can subscribe to plans and
     *     have access to features.
     */
    BillingCustomer: {
      readonly id: components['schemas']['ULID']
      /**
       * @description Display name of the resource.
       *
       *     Between 1 and 256 characters.
       */
      name: string
      /**
       * @description Optional description of the resource.
       *
       *     Maximum 1024 characters.
       */
      description?: string
      labels?: components['schemas']['Labels']
      /** @description An ISO-8601 timestamp representation of entity creation date. */
      readonly created_at: components['schemas']['DateTime']
      /** @description An ISO-8601 timestamp representation of entity last update date. */
      readonly updated_at: components['schemas']['DateTime']
      /** @description An ISO-8601 timestamp representation of entity deletion date. */
      readonly deleted_at?: components['schemas']['DateTime']
      key: components['schemas']['ExternalResourceKey']
      /**
       * Usage Attribution
       * @description Mapping to attribute metered usage to the customer by the event subject.
       */
      usage_attribution?: components['schemas']['BillingCustomerUsageAttribution']
      /**
       * Primary Email
       * @description The primary email address of the customer.
       */
      primary_email?: string
      /**
       * Currency
       * @description Currency of the customer. Used for billing, tax and invoicing.
       */
      currency?: components['schemas']['CurrencyCode']
      /**
       * Billing Address
       * @description The billing address of the customer. Used for tax and invoicing.
       */
      billing_address?: components['schemas']['BillingAddress']
    }
    /** @description Billing customer data. */
    BillingCustomerData: {
      /**
       * Billing profile
       * @description The billing profile for the customer.
       *
       *     If not provided, the default billing profile will be used.
       */
      billing_profile?: components['schemas']['BillingProfileReference']
      /**
       * App customer data
       * @description App customer data.
       */
      app_data?: components['schemas']['BillingAppCustomerData']
    }
    /** @description Customer reference. */
    BillingCustomerReference: {
      /** @description The ID of the customer. */
      id: components['schemas']['ULID']
    }
    /**
     * @description Request to create a Stripe Checkout Session for the customer.
     *
     *     Checkout Sessions are used to collect payment method information from customers
     *     in a secure, Stripe-hosted interface. This integration uses setup mode to
     *     collect payment methods that can be charged later for subscription billing.
     */
    BillingCustomerStripeCreateCheckoutSessionRequest: {
      /**
       * @description Options for configuring the Stripe Checkout Session.
       *
       *     These options are passed directly to Stripe's
       *     [checkout session creation API](https://docs.stripe.com/api/checkout/sessions/create).
       */
      stripe_options: components['schemas']['BillingAppStripeCreateCheckoutSessionRequestOptions']
    }
    /**
     * @description Request to create a Stripe Customer Portal Session for the customer.
     *
     *     Useful to redirect the customer to the Stripe Customer Portal to manage their
     *     payment methods, change their billing address and access their invoice history.
     *     Only returns URL if the customer billing profile is linked to a stripe app and
     *     customer.
     */
    BillingCustomerStripeCreateCustomerPortalSessionRequest: {
      /** @description Options for configuring the Stripe Customer Portal Session. */
      stripe_options: components['schemas']['BillingAppStripeCreateCustomerPortalSessionOptions']
    }
    /**
     * @description Mapping to attribute metered usage to the customer. One customer can have zero
     *     or more subjects, but one subject can only belong to one customer.
     */
    BillingCustomerUsageAttribution: {
      /**
       * Subject Keys
       * @description The subjects that are attributed to the customer. Can be empty when no usage
       *     event subjects are associated with the customer.
       */
      subject_keys: components['schemas']['UsageAttributionSubjectKey'][]
    }
    /** @description Entitlement access result. */
    BillingEntitlementAccessResult: {
      /**
       * @description The type of the entitlement.
       * @example static
       */
      readonly type: components['schemas']['BillingEntitlementType']
      /**
       * @description The feature key of the entitlement.
       * @example available_models
       */
      readonly feature_key: components['schemas']['ResourceKey']
      /**
       * @description Whether the customer has access to the feature. Always true for `boolean` and
       *     `static` entitlements. Depends on balance for `metered` entitlements.
       * @example true
       */
      readonly has_access: boolean
      /**
       * @description Only available for static entitlements. Config is the JSON parsable
       *     configuration of the entitlement. Useful to describe per customer configuration.
       * @example { "availableModels": ["gpt-5", "gpt-4o"] }
       */
      readonly config?: string
    }
    /**
     * @description The type of the entitlement.
     * @enum {string}
     */
    BillingEntitlementType: 'metered' | 'static' | 'boolean'
    /**
     * @description Token type for LLM cost lookup.
     * @enum {string}
     */
    BillingFeatureLLMTokenType:
      | 'input'
      | 'output'
      | 'cache_read'
      | 'cache_write'
      | 'reasoning'
      | 'request'
      | 'response'
    /**
     * @description LLM cost lookup configuration. Each dimension (provider, model, token type) can
     *     be specified as either a static value or a meter group-by property name
     *     (mutually exclusive).
     */
    BillingFeatureLLMUnitCost: {
      /**
       * @description The type discriminator for LLM unit cost. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'llm'
      /**
       * Provider property
       * @description Meter group-by property that holds the LLM provider. Use this when the meter has
       *     a group-by dimension for provider. Mutually exclusive with `provider`.
       */
      provider_property?: string
      /**
       * Provider
       * @description Static LLM provider value (e.g., "openai", "anthropic"). Use this when the
       *     feature tracks a single provider. Mutually exclusive with `provider_property`.
       */
      provider?: string
      /**
       * Model property
       * @description Meter group-by property that holds the model ID. Use this when the meter has a
       *     group-by dimension for model. Mutually exclusive with `model`.
       */
      model_property?: string
      /**
       * Model
       * @description Static model ID value (e.g., "gpt-4", "claude-3-5-sonnet"). Use this when the
       *     feature tracks a single model. Mutually exclusive with `model_property`.
       */
      model?: string
      /**
       * Token type property
       * @description Meter group-by property that holds the token type. Use this when the meter has a
       *     group-by dimension for token type. Mutually exclusive with `token_type`.
       */
      token_type_property?: string
      /**
       * Token type
       * @description Static token type value. Use this when the feature tracks a single token type
       *     (e.g., only input tokens). `request` is an alias for `input`, `response` is an
       *     alias for `output`. Mutually exclusive with `token_type_property`.
       */
      token_type?: components['schemas']['BillingFeatureLLMTokenType']
      /**
       * Resolved pricing
       * @description Resolved per-token pricing from the LLM cost database. Populated in responses
       *     when the provider and model can be determined, either from static values or from
       *     meter group-by filters with exact matches.
       */
      readonly pricing?: components['schemas']['BillingFeatureLLMUnitCostPricing']
    }
    /** @description Resolved per-token pricing from the LLM cost database. */
    BillingFeatureLLMUnitCostPricing: {
      /**
       * Input per token
       * @description Cost per input token in USD.
       */
      input_per_token: components['schemas']['Numeric']
      /**
       * Output per token
       * @description Cost per output token in USD.
       */
      output_per_token: components['schemas']['Numeric']
      /**
       * Cache read per token
       * @description Cost per cache read token in USD.
       */
      cache_read_per_token?: components['schemas']['Numeric']
      /**
       * Reasoning per token
       * @description Cost per reasoning token in USD.
       */
      reasoning_per_token?: components['schemas']['Numeric']
      /**
       * Cache write per token
       * @description Cost per cache write token in USD.
       */
      cache_write_per_token?: components['schemas']['Numeric']
    }
    /** @description A fixed per-unit cost amount. */
    BillingFeatureManualUnitCost: {
      /**
       * @description The type discriminator for manual unit cost. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'manual'
      /** @description Fixed per-unit cost amount in USD. */
      amount: components['schemas']['Numeric']
    }
    /**
     * @description Per-unit cost configuration for a feature. Either a fixed manual amount or a
     *     dynamic LLM cost lookup.
     */
    BillingFeatureUnitCost:
      | components['schemas']['BillingFeatureManualUnitCost']
      | components['schemas']['BillingFeatureLLMUnitCost']
    /**
     * Flat fee charge
     * @description A flat fee charge for a customer.
     */
    BillingFlatFeeCharge: {
      readonly id: components['schemas']['ULID']
      /**
       * @description Display name of the resource.
       *
       *     Between 1 and 256 characters.
       */
      name: string
      /**
       * @description Optional description of the resource.
       *
       *     Maximum 1024 characters.
       */
      description?: string
      labels?: components['schemas']['Labels']
      /** @description An ISO-8601 timestamp representation of entity creation date. */
      readonly created_at: components['schemas']['DateTime']
      /** @description An ISO-8601 timestamp representation of entity last update date. */
      readonly updated_at: components['schemas']['DateTime']
      /** @description An ISO-8601 timestamp representation of entity deletion date. */
      readonly deleted_at?: components['schemas']['DateTime']
      /**
       * @description The type of the charge. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'flat_fee'
      /**
       * Customer
       * @description The customer owning the charge.
       */
      readonly customer: components['schemas']['BillingCustomerReference']
      /**
       * Managed by
       * @description The charge is managed by the following entity.
       */
      readonly managed_by: components['schemas']['ResourceManagedBy']
      /**
       * Subscription
       * @description The subscription that originated the charge, when the charge was created from a
       *     subscription item.
       */
      readonly subscription?: components['schemas']['BillingSubscriptionReference']
      /**
       * Currency
       * @description The currency of the charge.
       */
      readonly currency: components['schemas']['CurrencyCode']
      /**
       * Status
       * @description The lifecycle status of the charge.
       */
      readonly status: components['schemas']['BillingChargeStatus']
      /**
       * Invoice at
       * @description The timestamp when the charge is intended to be invoiced.
       */
      readonly invoice_at: components['schemas']['DateTime']
      /**
       * Service period
       * @description The effective service period covered by the charge.
       */
      readonly service_period: components['schemas']['ClosedPeriod']
      /**
       * Full service period
       * @description The full, unprorated service period of the charge.
       */
      readonly full_service_period: components['schemas']['ClosedPeriod']
      /**
       * Billing period
       * @description The billing period the charge belongs to.
       */
      readonly billing_period: components['schemas']['ClosedPeriod']
      /**
       * Advance after
       * @description The earliest time when the charge should be advanced again by background
       *     processing.
       */
      readonly advance_after?: components['schemas']['DateTime']
      /**
       * Price
       * @description The price of the charge.
       */
      readonly price: components['schemas']['BillingPrice']
      /**
       * Unique reference ID
       * @description Unique reference ID of the charge.
       */
      readonly unique_reference_id?: string
      /**
       * Settlement mode
       * @description Settlement mode of the charge.
       */
      readonly settlement_mode: components['schemas']['BillingSettlementMode']
      /**
       * Tax configuration
       * @description Tax configuration of the charge.
       */
      readonly tax_config?: components['schemas']['BillingTaxConfig']
      /**
       * Payment term
       * @description Payment term of the flat fee charge.
       */
      payment_term: components['schemas']['BillingPricePaymentTerm']
      /**
       * Discounts
       * @description The discounts applied to the charge.
       */
      discounts?: components['schemas']['BillingFlatFeeDiscounts']
      /**
       * Feature key
       * @description The feature associated with the charge, when applicable.
       */
      feature_key?: string
      /**
       * Proration configuration
       * @description The proration configuration of the charge.
       */
      proration_configuration: components['schemas']['BillingRateCardProrationConfiguration']
      /**
       * Amount after proration
       * @description The amount after proration of the charge.
       */
      readonly amount_after_proration: components['schemas']['CurrencyAmount']
    }
    /**
     * Flat fee charge discounts
     * @description Discounts applicable to flat fee charges.
     *
     *     This is the same as `ProductCatalog.Discounts` but without the `usage` field,
     *     which is not applicable to flat fee charges.
     */
    BillingFlatFeeDiscounts: {
      /** @description Percentage discount applied to the price (0–100). */
      percentage?: number
    }
    /** @description Party represents a person or business entity. */
    BillingParty: {
      /** @description Unique identifier for the party. */
      readonly id?: string
      /** @description An optional unique key of the party. */
      key?: components['schemas']['ExternalResourceKey']
      /** @description Legal name or representation of the party. */
      name?: string
      /**
       * @description The entity's legal identification used for tax purposes. They may have other
       *     numbers, but we're only interested in those valid for tax purposes.
       */
      tax_id?: components['schemas']['BillingPartyTaxIdentity']
      /** @description Address for where information should be sent if needed. */
      addresses?: components['schemas']['BillingPartyAddresses']
    }
    /** @description A collection of addresses for the party. */
    BillingPartyAddresses: {
      /** @description Billing address. */
      billing_address: components['schemas']['Address']
    }
    /**
     * @description Identity stores the details required to identify an entity for tax purposes in a
     *     specific country.
     */
    BillingPartyTaxIdentity: {
      /** @description Normalized tax identification code shown on the original identity document. */
      code?: components['schemas']['BillingTaxIdentificationCode']
    }
    /** @description Plans provide a template for subscriptions. */
    BillingPlan: {
      readonly id: components['schemas']['ULID']
      /**
       * @description Display name of the resource.
       *
       *     Between 1 and 256 characters.
       */
      name: string
      /**
       * @description Optional description of the resource.
       *
       *     Maximum 1024 characters.
       */
      description?: string
      labels?: components['schemas']['Labels']
      /** @description An ISO-8601 timestamp representation of entity creation date. */
      readonly created_at: components['schemas']['DateTime']
      /** @description An ISO-8601 timestamp representation of entity last update date. */
      readonly updated_at: components['schemas']['DateTime']
      /** @description An ISO-8601 timestamp representation of entity deletion date. */
      readonly deleted_at?: components['schemas']['DateTime']
      /**
       * Key
       * @description A key is a semi-unique string that is used to identify the plan. It is used to
       *     reference the latest `active` version of the plan and is unique with the version
       *     number.
       */
      key: components['schemas']['ResourceKey']
      /**
       * Version
       * @description Plans are versioned to allow you to make changes without affecting running
       *     subscriptions.
       * @default 1
       */
      readonly version: number
      /**
       * Currency
       * @description The currency code of the plan.
       */
      currency: components['schemas']['CurrencyCode']
      /**
       * Billing cadence
       * @description The billing cadence for subscriptions using this plan.
       */
      billing_cadence: components['schemas']['ISO8601Duration']
      /**
       * Pro-rating enabled
       * @description Whether pro-rating is enabled for this plan.
       * @default true
       */
      pro_rating_enabled?: boolean
      /**
       * Effective start date
       * @description The date and time when the plan becomes `active`. When not specified, the plan
       *     is in `draft` status.
       */
      readonly effective_from?: components['schemas']['DateTime']
      /**
       * Effective end date
       * @description A scheduled date and time when the plan becomes `archived`. When not specified,
       *     the plan is in `active` status indefinitely.
       */
      readonly effective_to?: components['schemas']['DateTime']
      /**
       * Status
       * @description The status of the plan. Computed based on the effective start and end dates:
       *
       *     - `draft`: `effective_from` is not set.
       *     - `scheduled`: `now < effective_from`.
       *     - `active`: `effective_from <= now` and (`effective_to` is not set or
       *     `now < effective_to`).
       *     - `archived`: `effective_to <= now`.
       */
      readonly status: components['schemas']['BillingPlanStatus']
      /**
       * Plan phases
       * @description The plan phases define the pricing ramp for a subscription. A phase switch
       *     occurs only at the end of a billing period. At least one phase is required.
       */
      phases: components['schemas']['BillingPlanPhase'][]
      /**
       * Validation errors
       * @description List of validation errors in `draft` state that prevent the plan from being
       *     published.
       */
      readonly validation_errors?: components['schemas']['ProductCatalogValidationError'][]
    }
    /**
     * @description The plan phase or pricing ramp allows changing a plan's rate cards over time as
     *     a subscription progresses.
     */
    BillingPlanPhase: {
      /**
       * @description Display name of the resource.
       *
       *     Between 1 and 256 characters.
       */
      name: string
      /**
       * @description Optional description of the resource.
       *
       *     Maximum 1024 characters.
       */
      description?: string
      labels?: components['schemas']['Labels']
      key: components['schemas']['ResourceKey']
      /**
       * Duration
       * @description The duration of the phase. When not specified, the phase runs indefinitely. Only
       *     the last phase may omit the duration.
       */
      duration?: components['schemas']['ISO8601Duration']
      /**
       * Rate cards
       * @description The rate cards of the plan.
       */
      rate_cards: components['schemas']['BillingRateCard'][]
    }
    /**
     * @description The status of a plan.
     *
     *     - `draft`: The plan has not yet been published and can be edited.
     *     - `active`: The plan is published and can be used in subscriptions.
     *     - `archived`: The plan is no longer available for use.
     *     - `scheduled`: The plan is scheduled to be published at a future date.
     * @enum {string}
     */
    BillingPlanStatus: 'draft' | 'active' | 'archived' | 'scheduled'
    /** @description Price. */
    BillingPrice:
      | components['schemas']['BillingPriceFree']
      | components['schemas']['BillingPriceFlat']
      | components['schemas']['BillingPriceUnit']
      | components['schemas']['BillingPriceGraduated']
      | components['schemas']['BillingPriceVolume']
    /** @description Flat price. */
    BillingPriceFlat: {
      /**
       * @description The type of the price. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'flat'
      /**
       * Amount
       * @description The amount of the flat price.
       */
      amount: components['schemas']['Numeric']
    }
    /** @description Free price. */
    BillingPriceFree: {
      /**
       * @description The type of the price. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'free'
    }
    /**
     * @description Graduated tiered price.
     *
     *     Each tier's rate applies only to the usage within that tier. Pricing can change
     *     as cumulative usage crosses tier boundaries.
     *
     *     When UnitConfig is present on the rate card, tier boundaries (up_to_amount) are
     *     expressed in converted billing units.
     */
    BillingPriceGraduated: {
      /**
       * @description The type of the price. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'graduated'
      /**
       * Tiers
       * @description The tiers of the graduated price. At least one tier is required.
       */
      tiers: components['schemas']['BillingPriceTier'][]
    }
    /**
     * @description The payment term of a flat price.
     * @enum {string}
     */
    BillingPricePaymentTerm: 'in_advance' | 'in_arrears'
    /**
     * @description A price tier used in graduated and volume pricing.
     *
     *     At least one price component (flat_price or unit_price) must be set. When
     *     UnitConfig is present on the rate card, up_to_amount is expressed in converted
     *     billing units.
     */
    BillingPriceTier: {
      /**
       * Up to quantity
       * @description Up to and including this quantity will be contained in the tier. If undefined,
       *     the tier is open-ended (the last tier).
       */
      up_to_amount?: components['schemas']['Numeric']
      /**
       * Flat price component
       * @description The flat price component of the tier. Charged once when the tier is entered.
       */
      flat_price?: components['schemas']['BillingPriceFlat']
      /**
       * Unit price component
       * @description The unit price component of the tier. Charged per billing unit within the tier.
       */
      unit_price?: components['schemas']['BillingPriceUnit']
    }
    /**
     * @description Unit price.
     *
     *     Charges a fixed rate per billing unit. When UnitConfig is present on the rate
     *     card, billing units are the converted quantities (e.g. GB instead of bytes).
     */
    BillingPriceUnit: {
      /**
       * @description The type of the price. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'unit'
      /**
       * Amount
       * @description The amount of the unit price.
       */
      amount: components['schemas']['Numeric']
    }
    /**
     * @description Volume tiered price.
     *
     *     The maximum quantity within a period determines the per-unit price for all units
     *     in that period.
     *
     *     When UnitConfig is present on the rate card, tier boundaries (up_to_amount) are
     *     expressed in converted billing units.
     */
    BillingPriceVolume: {
      /**
       * @description The type of the price. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'volume'
      /**
       * Tiers
       * @description The tiers of the volume price. At least one tier is required.
       */
      tiers: components['schemas']['BillingPriceTier'][]
    }
    /**
     * @description Billing profiles contain the settings for billing and controls invoice
     *     generation.
     */
    BillingProfile: {
      readonly id: components['schemas']['ULID']
      /**
       * @description Display name of the resource.
       *
       *     Between 1 and 256 characters.
       */
      name: string
      /**
       * @description Optional description of the resource.
       *
       *     Maximum 1024 characters.
       */
      description?: string
      labels?: components['schemas']['Labels']
      /** @description An ISO-8601 timestamp representation of entity creation date. */
      readonly created_at: components['schemas']['DateTime']
      /** @description An ISO-8601 timestamp representation of entity last update date. */
      readonly updated_at: components['schemas']['DateTime']
      /** @description An ISO-8601 timestamp representation of entity deletion date. */
      readonly deleted_at?: components['schemas']['DateTime']
      /**
       * @description The name and contact information for the supplier this billing profile
       *     represents
       */
      supplier: components['schemas']['BillingParty']
      /** @description The billing workflow settings for this profile */
      workflow: components['schemas']['BillingWorkflow']
      /** @description The applications used by this billing profile. */
      apps: components['schemas']['BillingProfileAppReferences']
      /** @description Whether this is the default profile. */
      default: boolean
    }
    /** @description References to the applications used by a billing profile. */
    BillingProfileAppReferences: {
      /** @description The tax app used for this workflow. */
      tax: components['schemas']['BillingAppReference']
      /** @description The invoicing app used for this workflow. */
      invoicing: components['schemas']['BillingAppReference']
      /** @description The payment app used for this workflow. */
      payment: components['schemas']['BillingAppReference']
    }
    /** @description Page paginated response. */
    BillingProfilePagePaginatedResponse: {
      data: components['schemas']['BillingProfile'][]
      meta: components['schemas']['PaginatedMeta']
    }
    /** @description Billing profile reference. */
    BillingProfileReference: {
      /** @description The ID of the billing profile. */
      id: components['schemas']['ULID']
    }
    /** @description A rate card defines the pricing and entitlement of a feature or service. */
    BillingRateCard: {
      /**
       * @description Display name of the resource.
       *
       *     Between 1 and 256 characters.
       */
      name: string
      /**
       * @description Optional description of the resource.
       *
       *     Maximum 1024 characters.
       */
      description?: string
      labels?: components['schemas']['Labels']
      key: components['schemas']['ResourceKey']
      /**
       * Feature reference
       * @description The feature associated with the rate card.
       */
      feature?: components['schemas']['FeatureReferenceItem']
      /**
       * Billing cadence
       * @description The billing cadence of the rate card. When null, the charge is one-time
       *     (non-recurring). Only valid for flat prices.
       */
      billing_cadence?: components['schemas']['ISO8601Duration']
      /**
       * Price
       * @description The price of the rate card.
       */
      price: components['schemas']['BillingPrice']
      /**
       * Payment term
       * @description The payment term of the rate card. In advance payment term can only be used for
       *     flat prices.
       * @default in_arrears
       */
      payment_term?: components['schemas']['BillingPricePaymentTerm']
      /**
       * Commitments
       * @description Spend commitments for this rate card. Only applicable to usage-based prices
       *     (unit, graduated, volume).
       */
      commitments?: components['schemas']['BillingSpendCommitments']
      /**
       * Discounts
       * @description The discounts of the rate card.
       */
      discounts?: components['schemas']['BillingRateCardDiscounts']
      /**
       * Tax config
       * @description The tax config of the rate card.
       */
      tax_config?: components['schemas']['BillingRateCardTaxConfig']
    }
    /** @description Discount configuration for a rate card. */
    BillingRateCardDiscounts: {
      /** @description Percentage discount applied to the price (0–100). */
      percentage?: number
      /**
       * @description Number of usage units granted free before billing starts. Only applies to
       *     usage-based lines (not flat fees). Usage is treated as zero until this amount is
       *     exhausted.
       */
      usage?: components['schemas']['Numeric']
    }
    /** @description The proration configuration of the rate card. */
    BillingRateCardProrationConfiguration: {
      /**
       * Proration mode
       * @description The proration mode of the rate card.
       */
      mode: components['schemas']['BillingRateCardProrationMode']
    }
    /**
     * @description The proration mode of the rate card.
     *
     *     Values:
     *
     *     - `no_proration`: No proration.
     *     - `prorate_prices`: Prorate the price based on the time remaining in the billing
     *     period.
     * @enum {string}
     */
    BillingRateCardProrationMode: 'no_proration' | 'prorate_prices'
    /** @description The tax config of the rate card. */
    BillingRateCardTaxConfig: {
      behavior?: components['schemas']['BillingTaxBehavior']
      code: components['schemas']['TaxCodeReferenceItem']
    }
    /**
     * Settlement mode
     * @description Settlement mode for billing.
     *
     *     Values:
     *
     *     - `credit_then_invoice`: Credits are applied first, then any remainder is
     *     invoiced.
     *     - `credit_only`: Usage is settled exclusively against credits.
     * @enum {string}
     */
    BillingSettlementMode: 'credit_then_invoice' | 'credit_only'
    /**
     * @description Spend commitments for a rate card. The customer is committed to spend at least
     *     the minimum amount and at most the maximum amount.
     */
    BillingSpendCommitments: {
      /**
       * Minimum amount
       * @description The customer is committed to spend at least the amount.
       */
      minimum_amount?: components['schemas']['Numeric']
      /**
       * Maximum amount
       * @description The customer is limited to spend at most the amount.
       */
      maximum_amount?: components['schemas']['Numeric']
    }
    /** @description Subscription. */
    BillingSubscription: {
      readonly id: components['schemas']['ULID']
      labels?: components['schemas']['Labels']
      /** @description An ISO-8601 timestamp representation of entity creation date. */
      readonly created_at: components['schemas']['DateTime']
      /** @description An ISO-8601 timestamp representation of entity last update date. */
      readonly updated_at: components['schemas']['DateTime']
      /** @description An ISO-8601 timestamp representation of entity deletion date. */
      readonly deleted_at?: components['schemas']['DateTime']
      /**
       * Customer ID
       * @description The customer ID of the subscription.
       */
      readonly customer_id: components['schemas']['ULID']
      /**
       * Plan ID
       * @description The plan ID of the subscription. Set if subscription is created from a plan.
       */
      readonly plan_id?: components['schemas']['ULID']
      /**
       * Billing anchor
       * @description A billing anchor is the fixed point in time that determines the subscription's
       *     recurring billing cycle. It affects when charges occur and how prorations are
       *     calculated. Common anchors:
       *
       *     - Calendar month (1st of each month): `2025-01-01T00:00:00Z`
       *     - Subscription anniversary (day customer signed up)
       *     - Custom date (customer-specified day)
       */
      readonly billing_anchor: components['schemas']['DateTime']
      /**
       * Status
       * @description The status of the subscription.
       */
      readonly status: components['schemas']['BillingSubscriptionStatus']
    }
    /** @description Request for canceling a subscription. */
    BillingSubscriptionCancel: {
      /**
       * @description If not provided the subscription is canceled immediately.
       * @default immediate
       */
      timing?: components['schemas']['BillingSubscriptionEditTiming']
    }
    /** @description Request for changing a subscription. */
    BillingSubscriptionChange: {
      labels?: components['schemas']['Labels']
      /** @description The customer to create the subscription for. */
      customer: {
        /**
         * Customer ID
         * @description The ID of the customer to create the subscription for.
         *
         *     Either customer ID or customer key must be provided. If both are provided, the
         *     ID will be used.
         */
        id?: components['schemas']['ULID']
        /**
         * Customer Key
         * @description The key of the customer to create the subscription for.
         *
         *     Either customer ID or customer key must be provided. If both are provided, the
         *     ID will be used.
         */
        key?: components['schemas']['ExternalResourceKey']
      }
      /** @description The plan reference of the subscription. */
      plan: {
        /**
         * Plan ID
         * @description The plan ID of the subscription. Set if subscription is created from a plan.
         *
         *     ID or Key of the plan is required if creating a subscription from a plan. If
         *     both are provided, the ID will be used.
         */
        id?: components['schemas']['ULID']
        /**
         * Plan Key
         * @description The plan Key of the subscription, if any. Set if subscription is created from a
         *     plan.
         *
         *     ID or Key of the plan is required if creating a subscription from a plan. If
         *     both are provided, the ID will be used.
         */
        key?: components['schemas']['ResourceKey']
        /**
         * Plan Version
         * @description The plan version of the subscription, if any. If not provided, the latest
         *     version of the plan will be used.
         */
        version?: number
      }
      /**
       * Billing anchor
       * @description A billing anchor is the fixed point in time that determines the subscription's
       *     recurring billing cycle. It affects when charges occur and how prorations are
       *     calculated. Common anchors:
       *
       *     - Calendar month (1st of each month): `2025-01-01T00:00:00Z`
       *     - Subscription anniversary (day customer signed up)
       *     - Custom date (customer-specified day)
       *
       *     If not provided, the subscription will be created with the subscription's
       *     creation time as the billing anchor.
       */
      billing_anchor?: components['schemas']['DateTime']
      /**
       * @description Timing configuration for the change, when the change should take effect. For
       *     changing a subscription, the accepted values depend on the subscription
       *     configuration.
       */
      timing: components['schemas']['BillingSubscriptionEditTiming']
    }
    /** @description Response for changing a subscription. */
    BillingSubscriptionChangeResponse: {
      /** @description The current subscription before the change. */
      current: components['schemas']['BillingSubscription']
      /** @description The new state of the subscription after the change. */
      next: components['schemas']['BillingSubscription']
    }
    /** @description Subscription create request. */
    BillingSubscriptionCreate: {
      labels?: components['schemas']['Labels']
      /** @description The customer to create the subscription for. */
      customer: {
        /**
         * Customer ID
         * @description The ID of the customer to create the subscription for.
         *
         *     Either customer ID or customer key must be provided. If both are provided, the
         *     ID will be used.
         */
        id?: components['schemas']['ULID']
        /**
         * Customer Key
         * @description The key of the customer to create the subscription for.
         *
         *     Either customer ID or customer key must be provided. If both are provided, the
         *     ID will be used.
         */
        key?: components['schemas']['ExternalResourceKey']
      }
      /** @description The plan reference of the subscription. */
      plan: {
        /**
         * Plan ID
         * @description The plan ID of the subscription. Set if subscription is created from a plan.
         *
         *     ID or Key of the plan is required if creating a subscription from a plan. If
         *     both are provided, the ID will be used.
         */
        id?: components['schemas']['ULID']
        /**
         * Plan Key
         * @description The plan Key of the subscription, if any. Set if subscription is created from a
         *     plan.
         *
         *     ID or Key of the plan is required if creating a subscription from a plan. If
         *     both are provided, the ID will be used.
         */
        key?: components['schemas']['ResourceKey']
        /**
         * Plan Version
         * @description The plan version of the subscription, if any. If not provided, the latest
         *     version of the plan will be used.
         */
        version?: number
      }
      /**
       * Billing anchor
       * @description A billing anchor is the fixed point in time that determines the subscription's
       *     recurring billing cycle. It affects when charges occur and how prorations are
       *     calculated. Common anchors:
       *
       *     - Calendar month (1st of each month): `2025-01-01T00:00:00Z`
       *     - Subscription anniversary (day customer signed up)
       *     - Custom date (customer-specified day)
       *
       *     If not provided, the subscription will be created with the subscription's
       *     creation time as the billing anchor.
       */
      billing_anchor?: components['schemas']['DateTime']
    }
    /**
     * @description Subscription edit timing defined when the changes should take effect. If the
     *     provided configuration is not supported by the subscription, an error will be
     *     returned.
     * @example immediate
     */
    BillingSubscriptionEditTiming:
      | components['schemas']['BillingSubscriptionEditTimingEnum']
      | components['schemas']['DateTime']
    /**
     * @description Subscription edit timing. When immediate, the requested changes take effect
     *     immediately. When next_billing_cycle, the requested changes take effect at the
     *     next billing cycle.
     * @enum {string}
     */
    BillingSubscriptionEditTimingEnum: 'immediate' | 'next_billing_cycle'
    /**
     * @description Subscription reference represents a reference to the specific subscription item
     *     this entity represents.
     */
    BillingSubscriptionReference: {
      /**
       * Subscription ID
       * @description The ID of the subscription.
       */
      readonly id: components['schemas']['ULID']
      /**
       * Phase ID
       * @description The phase of the subscription.
       */
      readonly phase: {
        /**
         * Phase ID
         * @description The ID of the phase.
         */
        readonly id: components['schemas']['ULID']
        /**
         * Item ID
         * @description The item of the phase.
         */
        readonly item: {
          /**
           * Item ID
           * @description The ID of the item.
           */
          readonly id: components['schemas']['ULID']
        }
      }
    }
    /**
     * @description Subscription status.
     * @enum {string}
     */
    BillingSubscriptionStatus: 'active' | 'inactive' | 'canceled' | 'scheduled'
    /**
     * @description Tax behavior.
     *
     *     This enum is used to specify whether tax is included in the price or excluded
     *     from the price.
     * @enum {string}
     */
    BillingTaxBehavior: 'inclusive' | 'exclusive'
    /** @description Tax codes by provider. */
    BillingTaxCode: {
      readonly id: components['schemas']['ULID']
      /**
       * @description Display name of the resource.
       *
       *     Between 1 and 256 characters.
       */
      name: string
      /**
       * @description Optional description of the resource.
       *
       *     Maximum 1024 characters.
       */
      description?: string
      labels?: components['schemas']['Labels']
      /** @description An ISO-8601 timestamp representation of entity creation date. */
      readonly created_at: components['schemas']['DateTime']
      /** @description An ISO-8601 timestamp representation of entity last update date. */
      readonly updated_at: components['schemas']['DateTime']
      /** @description An ISO-8601 timestamp representation of entity deletion date. */
      readonly deleted_at?: components['schemas']['DateTime']
      key: components['schemas']['ResourceKey']
      /**
       * App type to tax code mappings
       * @description Mapping of app types to tax codes.
       */
      app_mappings: components['schemas']['BillingTaxCodeAppMapping'][]
    }
    /** @description Mapping of app types to tax codes. */
    BillingTaxCodeAppMapping: {
      /**
       * App type
       * @description The app type that the tax code is associated with.
       */
      app_type: components['schemas']['BillingAppType']
      /**
       * Tax code
       * @description Tax code.
       */
      tax_code: string
    }
    /** @description Set of provider specific tax configs. */
    BillingTaxConfig: {
      /**
       * Tax behavior
       * @description Tax behavior.
       *
       *     If not specified the billing profile is used to determine the tax behavior. If
       *     not specified in the billing profile, the provider's default behavior is used.
       */
      behavior?: components['schemas']['BillingTaxBehavior']
      /**
       * Stripe tax config
       * @deprecated
       * @description Stripe tax config.
       */
      stripe?: components['schemas']['BillingTaxConfigStripe']
      /**
       * External invoicing tax config
       * @deprecated
       * @description External invoicing tax config.
       */
      external_invoicing?: components['schemas']['BillingTaxConfigExternalInvoicing']
      /**
       * Tax code ID
       * @deprecated
       * @description Tax code ID.
       */
      tax_code_id?: components['schemas']['ULID']
      /**
       * Tax code
       * @description Tax code reference.
       *
       *     When both `tax_code` and `tax_code_id` are provided, `tax_code` takes
       *     precedence. When `stripe.code` is also provided, `tax_code` still wins and
       *     `stripe.code` is ignored.
       */
      tax_code?: components['schemas']['TaxCodeReference']
    }
    /** @description External invoicing tax config. */
    BillingTaxConfigExternalInvoicing: {
      /**
       * Tax code
       * @description The tax code should be interpreted by the external invoicing provider.
       */
      code: string
    }
    /** @description The tax config for Stripe. */
    BillingTaxConfigStripe: {
      /**
       * Tax code
       * @description Product [tax code](https://docs.stripe.com/tax/tax-codes).
       * @example txcd_10000000
       */
      code: string
    }
    /**
     * @description Tax identifier code is a normalized tax code shown on the original identity
     *     document.
     */
    BillingTaxIdentificationCode: string
    /** @description Totals contains the summaries of all calculations for a billing resource. */
    BillingTotals: {
      /**
       * Amount
       * @description The total value of the resource before taxes, discounts and commitments.
       */
      readonly amount: components['schemas']['Numeric']
      /**
       * Taxes total
       * @description The total tax amount applied to the resource.
       */
      readonly taxes_total: components['schemas']['Numeric']
      /**
       * Inclusive taxes total
       * @description The total tax amount already included in the resource amount.
       */
      readonly taxes_inclusive_total: components['schemas']['Numeric']
      /**
       * Exclusive taxes total
       * @description The total tax amount added on top of the resource amount.
       */
      readonly taxes_exclusive_total: components['schemas']['Numeric']
      /**
       * Charges total
       * @description The total amount contributed by additional charges.
       */
      readonly charges_total: components['schemas']['Numeric']
      /**
       * Discounts total
       * @description The total amount deducted through discounts.
       */
      readonly discounts_total: components['schemas']['Numeric']
      /**
       * Credits total
       * @description The total amount deducted through credits before taxes are applied.
       */
      readonly credits_total: components['schemas']['Numeric']
      /**
       * Total
       * @description The final total value of the resource after taxes, discounts and commitments.
       */
      readonly total: components['schemas']['Numeric']
    }
    /**
     * Usage-based charge
     * @description A usage-based charge for a customer.
     */
    BillingUsageBasedCharge: {
      readonly id: components['schemas']['ULID']
      /**
       * @description Display name of the resource.
       *
       *     Between 1 and 256 characters.
       */
      name: string
      /**
       * @description Optional description of the resource.
       *
       *     Maximum 1024 characters.
       */
      description?: string
      labels?: components['schemas']['Labels']
      /** @description An ISO-8601 timestamp representation of entity creation date. */
      readonly created_at: components['schemas']['DateTime']
      /** @description An ISO-8601 timestamp representation of entity last update date. */
      readonly updated_at: components['schemas']['DateTime']
      /** @description An ISO-8601 timestamp representation of entity deletion date. */
      readonly deleted_at?: components['schemas']['DateTime']
      /**
       * @description The type of the charge. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'usage_based'
      /**
       * Customer
       * @description The customer owning the charge.
       */
      readonly customer: components['schemas']['BillingCustomerReference']
      /**
       * Managed by
       * @description The charge is managed by the following entity.
       */
      readonly managed_by: components['schemas']['ResourceManagedBy']
      /**
       * Subscription
       * @description The subscription that originated the charge, when the charge was created from a
       *     subscription item.
       */
      readonly subscription?: components['schemas']['BillingSubscriptionReference']
      /**
       * Currency
       * @description The currency of the charge.
       */
      readonly currency: components['schemas']['CurrencyCode']
      /**
       * Status
       * @description The lifecycle status of the charge.
       */
      readonly status: components['schemas']['BillingChargeStatus']
      /**
       * Invoice at
       * @description The timestamp when the charge is intended to be invoiced.
       */
      readonly invoice_at: components['schemas']['DateTime']
      /**
       * Service period
       * @description The effective service period covered by the charge.
       */
      readonly service_period: components['schemas']['ClosedPeriod']
      /**
       * Full service period
       * @description The full, unprorated service period of the charge.
       */
      readonly full_service_period: components['schemas']['ClosedPeriod']
      /**
       * Billing period
       * @description The billing period the charge belongs to.
       */
      readonly billing_period: components['schemas']['ClosedPeriod']
      /**
       * Advance after
       * @description The earliest time when the charge should be advanced again by background
       *     processing.
       */
      readonly advance_after?: components['schemas']['DateTime']
      /**
       * Price
       * @description The price of the charge.
       */
      readonly price: components['schemas']['BillingPrice']
      /**
       * Unique reference ID
       * @description Unique reference ID of the charge.
       */
      readonly unique_reference_id?: string
      /**
       * Settlement mode
       * @description Settlement mode of the charge.
       */
      readonly settlement_mode: components['schemas']['BillingSettlementMode']
      /**
       * Tax configuration
       * @description Tax configuration of the charge.
       */
      readonly tax_config?: components['schemas']['BillingTaxConfig']
      /**
       * Discounts
       * @description Discounts applied to the usage-based charge.
       */
      discounts?: components['schemas']['BillingRateCardDiscounts']
      /**
       * Feature key
       * @description The feature associated with the charge.
       */
      feature_key: string
      /**
       * Totals for the charge
       * @description Aggregated booked and realtime totals for the charge.
       */
      readonly totals: components['schemas']['BillingChargeTotals']
    }
    /** @description Billing workflow settings. */
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
    /**
     * @description The alignment for collecting the pending line items into an invoice.
     *
     *     Defaults to subscription, which means that we are to create a new invoice every
     *     time the a subscription period starts (for in advance items) or ends (for in
     *     arrears items).
     */
    BillingWorkflowCollectionAlignment:
      | components['schemas']['BillingWorkflowCollectionAlignmentSubscription']
      | components['schemas']['BillingWorkflowCollectionAlignmentAnchored']
    /**
     * @description BillingWorkflowCollectionAlignmentAnchored specifies the alignment for
     *     collecting the pending line items into an invoice.
     */
    BillingWorkflowCollectionAlignmentAnchored: {
      /**
       * @description The type of alignment. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'anchored'
      /** @description The recurring period for the alignment. */
      recurring_period: components['schemas']['RecurringPeriod']
    }
    /**
     * @description BillingWorkflowCollectionAlignmentSubscription specifies the alignment for
     *     collecting the pending line items into an invoice.
     */
    BillingWorkflowCollectionAlignmentSubscription: {
      /**
       * @description The type of alignment. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      type: 'subscription'
    }
    /**
     * Workflow collection settings
     * @description Workflow collection specifies how to collect the pending line items for an
     *     invoice.
     */
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
       * @description This grace period can be used to delay the collection of the pending line items
       *     specified in alignment.
       *
       *     This is useful, in case of multiple subscriptions having slightly different
       *     billing periods.
       * @default PT1H
       * @example P1D
       */
      interval?: string
    }
    /**
     * Workflow invoice settings
     * @description Invoice settings for a billing workflow.
     */
    BillingWorkflowInvoicingSettings: {
      /**
       * @description Whether to automatically issue the invoice after the draftPeriod has passed.
       * @default true
       */
      auto_advance?: boolean
      /**
       * Format: ISO8601
       * @description The period for the invoice to be kept in draft status for manual reviews.
       * @default P0D
       * @example P1D
       */
      draft_period?: string
      /**
       * @description Should progressive billing be allowed for this workflow?
       * @default true
       */
      progressive_billing?: boolean
    }
    /**
     * @description Payment settings for a billing workflow when the collection method is charge
     *     automatically.
     */
    BillingWorkflowPaymentChargeAutomaticallySettings: {
      /**
       * @description The collection method for the invoice. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      collection_method: 'charge_automatically'
    }
    /**
     * @description Payment settings for a billing workflow when the collection method is send
     *     invoice.
     */
    BillingWorkflowPaymentSendInvoiceSettings: {
      /**
       * @description The collection method for the invoice. (enum property replaced by openapi-typescript)
       * @enum {string}
       */
      collection_method: 'send_invoice'
      /**
       * Format: ISO8601
       * @description The period after which the invoice is due. With some payment solutions it's only
       *     applicable for manual collection method.
       * @default P30D
       * @example P30D
       */
      due_after?: string
    }
    /** @description Payment settings for a billing workflow. */
    BillingWorkflowPaymentSettings:
      | components['schemas']['BillingWorkflowPaymentChargeAutomaticallySettings']
      | components['schemas']['BillingWorkflowPaymentSendInvoiceSettings']
    /**
     * Workflow tax settings
     * @description Tax settings for a billing workflow.
     */
    BillingWorkflowTaxSettings: {
      /**
       * @description Enable automatic tax calculation when tax is supported by the app. For example,
       *     with Stripe Invoicing when enabled, tax is calculated via Stripe Tax.
       * @default true
       */
      enabled?: boolean
      /**
       * @description Enforce tax calculation when tax is supported by the app. When enabled, the
       *     billing system will not allow to create an invoice without tax calculation.
       *     Enforcement is different per apps, for example, Stripe app requires customer to
       *     have a tax location when starting a paid subscription.
       * @default false
       */
      enforced?: boolean
      /** @description Default tax configuration to apply to the invoices for line items. */
      default_tax_config?: components['schemas']['BillingTaxConfig']
    }
    /** @description Page paginated response. */
    ChargePagePaginatedResponse: {
      data: components['schemas']['BillingCharge'][]
      meta: components['schemas']['PaginatedMeta']
    }
    /**
     * @description A period with defined start and end dates.
     *
     *     The period is always inclusive at the start and exclusive at the end.
     */
    ClosedPeriod: {
      /**
       * Start
       * @description The start of the period.
       *
       *     The period is inclusive at the start.
       * @example 2023-01-01T01:01:01.001Z
       */
      from: components['schemas']['DateTime']
      /**
       * End
       * @description The end of the period.
       *
       *     The period is exclusive at the end.
       * @example 2023-01-01T01:01:01.001Z
       */
      to: components['schemas']['DateTime']
    }
    /** @description Page paginated response. */
    CostBasisPagePaginatedResponse: {
      data: components['schemas']['BillingCostBasis'][]
      meta: components['schemas']['PaginatedMeta']
    }
    /**
     * @description [ISO 3166-1](https://www.iso.org/iso-3166-country-codes.html) alpha-2 country
     *     code. Custom two-letter country codes are also supported for convenience.
     * @example US
     */
    CountryCode: string
    /** @description Addon create request. */
    CreateAddonRequest: {
      /**
       * @description Display name of the resource.
       *
       *     Between 1 and 256 characters.
       */
      name: string
      /**
       * @description Optional description of the resource.
       *
       *     Maximum 1024 characters.
       */
      description?: string
      labels?: components['schemas']['Labels']
      /**
       * Key
       * @description A key is a semi-unique string that is used to identify the add-on. It is used to
       *     reference the latest `active` version of the add-on and is unique with the
       *     version number.
       */
      key: components['schemas']['ResourceKey']
      /**
       * The InstanceType of the add-ons. Can be "single" or "multiple".
       * @description The InstanceType of the add-ons. Can be "single" or "multiple".
       */
      instance_type: components['schemas']['AddonInstanceType']
      /**
       * Currency
       * @description The currency code of the add-on.
       */
      currency: components['schemas']['BillingCurrencyCode']
      /**
       * Rate cards
       * @description The rate cards of the add-on.
       */
      rate_cards: components['schemas']['BillingRateCard'][]
    }
    /** @description BillingProfile create request. */
    CreateBillingProfileRequest: {
      /**
       * @description Display name of the resource.
       *
       *     Between 1 and 256 characters.
       */
      name: string
      /**
       * @description Optional description of the resource.
       *
       *     Maximum 1024 characters.
       */
      description?: string
      labels?: components['schemas']['Labels']
      /**
       * @description The name and contact information for the supplier this billing profile
       *     represents
       */
      supplier: components['schemas']['BillingParty']
      /** @description The billing workflow settings for this profile */
      workflow: components['schemas']['BillingWorkflow']
      /** @description The applications used by this billing profile. */
      apps: components['schemas']['BillingProfileAppReferences']
      /** @description Whether this is the default profile. */
      default: boolean
    }
    /** @description CostBasis create request. */
    CreateCostBasisRequest: {
      /** @description The fiat currency code for the cost basis. */
      fiat_code: components['schemas']['CurrencyCode']
      /** @description The cost rate for the currency. */
      rate: components['schemas']['Numeric']
      /**
       * @description An ISO-8601 timestamp representation of the date from which the cost basis is
       *     effective. If not provided, it will be effective immediately and will be set to
       *     `now` by the system.
       */
      effective_from?: components['schemas']['DateTime']
    }
    /** @description CreditAdjustment create request. */
    CreateCreditAdjustmentRequest: {
      /**
       * @description Display name of the resource.
       *
       *     Between 1 and 256 characters.
       */
      name: string
      /**
       * @description Optional description of the resource.
       *
       *     Maximum 1024 characters.
       */
      description?: string
      labels?: components['schemas']['Labels']
      /** @description The currency of the granted credits. */
      currency: components['schemas']['BillingCurrencyCode']
      /** @description Granted credit amount. */
      amount: components['schemas']['Numeric']
    }
    /** @description Filters for the credit grant. */
    CreateCreditGrantFilters: {
      /**
       * @description Limit the credit grant to specific features. If no features are specified, the
       *     credit grant can be used for any feature.
       * @example [
       *       "input_tokens",
       *       "output_tokens"
       *     ]
       */
      features?: components['schemas']['ResourceKey'][]
    }
    /** @description Purchase and payment terms of the grant. */
    CreateCreditGrantPurchase: {
      /** @description Currency of the purchase amount. */
      currency: components['schemas']['CurrencyCode']
      /**
       * @description Cost basis per credit unit used to calculate the purchase amount.
       *
       *     If `per_unit_cost_basis` is 0.50 and credit amount is $100.00, the total charge
       *     is $50.00. The value must be greater than 0. If the cost basis is 0, use
       *     `funding_method=none` instead.
       *
       *     Defaults to 1.0.
       * @default 1.0
       */
      per_unit_cost_basis?: components['schemas']['Numeric']
      /**
       * @description Controls when credits become available for consumption.
       *
       *     Defaults to `on_creation`.
       * @default on_creation
       */
      availability_policy?: components['schemas']['BillingCreditAvailabilityPolicy']
    }
    /** @description CreditGrant create request. */
    CreateCreditGrantRequest: {
      /**
       * @description Display name of the resource.
       *
       *     Between 1 and 256 characters.
       */
      name: string
      /**
       * @description Optional description of the resource.
       *
       *     Maximum 1024 characters.
       */
      description?: string
      labels?: components['schemas']['Labels']
      /** @description Funding method of the grant. */
      funding_method: components['schemas']['BillingCreditFundingMethod']
      /** @description The currency of the granted credits. */
      currency: components['schemas']['CreateCurrencyCode']
      /** @description Granted credit amount. */
      amount: components['schemas']['Numeric']
      /** @description Present when a funding workflow applies (funding_method is not `none`). */
      purchase?: components['schemas']['CreateCreditGrantPurchase']
      /**
       * @description Tax configuration for the grant.
       *
       *     For `invoice` and `external` funding methods, tax configuration should be
       *     provided to ensure correct revenue recognition. When not provided, the default
       *     credit grant tax code is applied, if that's not set the global default taxcode
       *     is used.
       */
      tax_config?: components['schemas']['CreateCreditGrantTaxConfig']
      filters?: components['schemas']['CreateCreditGrantFilters']
      /**
       * Format: int16
       * @description Draw-down priority of the grant. Lower values have higher priority.
       * @default 10
       */
      priority?: number
      /**
       * @description The duration after which the credit grant expires.
       *
       *     Defaults to never expiring.
       */
      expires_after?: components['schemas']['ISO8601Duration']
    }
    /**
     * Tax configuration for a credit grant
     * @description Tax configuration for a credit grant.
     *
     *     Tax configuration should be provided to ensure correct revenue recognition,
     *     including for externally funded grants.
     */
    CreateCreditGrantTaxConfig: {
      /** @description Tax behavior applied to the invoice line item. */
      behavior?: components['schemas']['BillingTaxBehavior']
      /** @description Tax code applied to the invoice line item. */
      tax_code?: components['schemas']['CreateResourceReference']
    }
    /** @description Fiat or custom currency code. */
    CreateCurrencyCode: string & components['schemas']['CurrencyCode']
    /** @description CurrencyCustom create request. */
    CreateCurrencyCustomRequest: {
      /**
       * @description The name of the currency. It should be a human-readable string that represents
       *     the name of the currency, such as "US Dollar" or "Euro".
       */
      name: string
      /** @description Description of the currency. */
      description?: string
      /**
       * @description The symbol of the currency. It should be a string that represents the symbol of
       *     the currency, such as "$" for US Dollar or "€" for Euro.
       */
      symbol?: string
      code: components['schemas']['BillingCurrencyCodeCustom']
    }
    /** @description Customer create request. */
    CreateCustomerRequest: {
      /**
       * @description Display name of the resource.
       *
       *     Between 1 and 256 characters.
       */
      name: string
      /**
       * @description Optional description of the resource.
       *
       *     Maximum 1024 characters.
       */
      description?: string
      labels?: components['schemas']['Labels']
      key: components['schemas']['ExternalResourceKey']
      /**
       * Usage Attribution
       * @description Mapping to attribute metered usage to the customer by the event subject.
       */
      usage_attribution?: components['schemas']['BillingCustomerUsageAttribution']
      /**
       * Primary Email
       * @description The primary email address of the customer.
       */
      primary_email?: string
      /**
       * Currency
       * @description Currency of the customer. Used for billing, tax and invoicing.
       */
      currency?: components['schemas']['CurrencyCode']
      /**
       * Billing Address
       * @description The billing address of the customer. Used for tax and invoicing.
       */
      billing_address?: components['schemas']['BillingAddress']
    }
    /** @description Feature create request. */
    CreateFeatureRequest: {
      /**
       * @description Display name of the resource.
       *
       *     Between 1 and 256 characters.
       */
      name: string
      /**
       * @description Optional description of the resource.
       *
       *     Maximum 1024 characters.
       */
      description?: string
      labels?: components['schemas']['Labels']
      key: components['schemas']['ResourceKey']
      /**
       * Meter reference
       * @description The meter that the feature is associated with and based on which usage is
       *     calculated. If not specified, the feature is static.
       */
      meter?: components['schemas']['FeatureMeterReference']
      /**
       * Unit cost
       * @description Optional per-unit cost configuration. Use "manual" for a fixed per-unit cost, or
       *     "llm" to look up cost from the LLM cost database based on meter group-by
       *     properties.
       */
      unit_cost?: components['schemas']['BillingFeatureUnitCost']
    }
    /** @description Meter create request. */
    CreateMeterRequest: {
      /**
       * @description Display name of the resource.
       *
       *     Between 1 and 256 characters.
       */
      name: string
      /**
       * @description Optional description of the resource.
       *
       *     Maximum 1024 characters.
       */
      description?: string
      labels?: components['schemas']['Labels']
      key: components['schemas']['ResourceKey']
      /** @description The aggregation type to use for the meter. */
      aggregation: components['schemas']['MeterAggregation']
      /**
       * @description The event type to include in the aggregation.
       * @example prompt
       */
      event_type: string
      /**
       * @description The date since the meter should include events. Useful to skip old events. If
       *     not specified, all historical events are included.
       */
      events_from?: components['schemas']['DateTime']
      /**
       * @description JSONPath expression to extract the value from the ingested event's data
       *     property.
       *
       *     The ingested value for sum, avg, min, and max aggregations is a number or a
       *     string that can be parsed to a number.
       *
       *     For unique_count aggregation, the ingested value must be a string. For count
       *     aggregation the value_property is ignored.
       * @example $.tokens
       */
      value_property?: string
      /**
       * @description Named JSONPath expressions to extract the group by values from the event data.
       *
       *     Keys must be unique and consist only alphanumeric and underscore characters.
       * @example {
       *       "type": "$.type"
       *     }
       */
      dimensions?: {
        [key: string]: string
      }
    }
    /** @description PlanAddon create request. */
    CreatePlanAddonRequest: {
      /**
       * @description Display name of the resource.
       *
       *     Between 1 and 256 characters.
       */
      name: string
      /**
       * @description Optional description of the resource.
       *
       *     Maximum 1024 characters.
       */
      description?: string
      labels?: components['schemas']['Labels']
      /**
       * Add-on
       * @description The add-on associated with the plan.
       */
      addon: components['schemas']['AddonReference']
      /**
       * From plan phase
       * @description The key of the plan phase from which the add-on becomes available for purchase.
       */
      from_plan_phase: components['schemas']['ResourceKey']
      /**
       * Max quantity
       * @description The maximum number of times the add-on can be purchased for the plan. For
       *     single-instance add-ons this field must be omitted. For multi-instance add-ons
       *     when omitted, unlimited quantity can be purchased.
       */
      max_quantity?: number
    }
    /** @description Plan create request. */
    CreatePlanRequest: {
      /**
       * @description Display name of the resource.
       *
       *     Between 1 and 256 characters.
       */
      name: string
      /**
       * @description Optional description of the resource.
       *
       *     Maximum 1024 characters.
       */
      description?: string
      labels?: components['schemas']['Labels']
      /**
       * Key
       * @description A key is a semi-unique string that is used to identify the plan. It is used to
       *     reference the latest `active` version of the plan and is unique with the version
       *     number.
       */
      key: components['schemas']['ResourceKey']
      /**
       * Currency
       * @description The currency code of the plan.
       */
      currency: components['schemas']['CurrencyCode']
      /**
       * Billing cadence
       * @description The billing cadence for subscriptions using this plan.
       */
      billing_cadence: components['schemas']['ISO8601Duration']
      /**
       * Pro-rating enabled
       * @description Whether pro-rating is enabled for this plan.
       * @default true
       */
      pro_rating_enabled?: boolean
      /**
       * Plan phases
       * @description The plan phases define the pricing ramp for a subscription. A phase switch
       *     occurs only at the end of a billing period. At least one phase is required.
       */
      phases: components['schemas']['BillingPlanPhase'][]
    }
    /** @description TaxCode reference. */
    CreateResourceReference: {
      id: components['schemas']['ULID']
    }
    /** @description TaxCode create request. */
    CreateTaxCodeRequest: {
      /**
       * @description Display name of the resource.
       *
       *     Between 1 and 256 characters.
       */
      name: string
      /**
       * @description Optional description of the resource.
       *
       *     Maximum 1024 characters.
       */
      description?: string
      labels?: components['schemas']['Labels']
      key: components['schemas']['ResourceKey']
      /**
       * App type to tax code mappings
       * @description Mapping of app types to tax codes.
       */
      app_mappings: components['schemas']['BillingTaxCodeAppMapping'][]
    }
    /**
     * Credit balance
     * @description The credit balance by currency.
     */
    CreditBalance: {
      readonly currency: components['schemas']['BillingCurrencyCode']
      /**
       * @description Credits that have been granted but cannot yet be consumed. Includes grants
       *     awaiting payment clearance or with a future effective date.
       * @example 200.00
       */
      readonly pending: components['schemas']['Numeric']
      /**
       * @description Credits that can be consumed right now. Derived from cleared grants after
       *     applying eligibility and restriction rules.
       * @example 150.00
       */
      readonly available: components['schemas']['Numeric']
    }
    /** @description Page paginated response. */
    CreditGrantPagePaginatedResponse: {
      data: components['schemas']['BillingCreditGrant'][]
      meta: components['schemas']['PaginatedMeta']
    }
    /** @description Cursor paginated response. */
    CreditTransactionPaginatedResponse: {
      data: components['schemas']['BillingCreditTransaction'][]
      meta: components['schemas']['CursorMeta']
    }
    /** @description Monetary amount in a specific currency. */
    CurrencyAmount: {
      amount: components['schemas']['Numeric']
      currency: components['schemas']['CurrencyCode']
    }
    /**
     * @description Three-letter [ISO4217](https://www.iso.org/iso-4217-currency-codes.html)
     *     currency code. Custom three-letter currency codes are also supported for
     *     convenience.
     * @example USD
     */
    CurrencyCode: string
    /** @description Page paginated response. */
    CurrencyPagePaginatedResponse: {
      data: components['schemas']['BillingCurrency'][]
      meta: components['schemas']['PaginatedMeta']
    }
    /** @description Determines which page of the collection to retrieve. */
    CursorPaginationQueryPage: {
      /** @description The number of items to include per page. */
      size?: number
      /** @description Request the next page of data, starting with the item after this parameter. */
      after?: string
      /** @description Request the previous page of data, starting with the item before this parameter. */
      before?: string
    }
    /** @description Page paginated response. */
    CustomerPagePaginatedResponse: {
      data: components['schemas']['BillingCustomer'][]
      meta: components['schemas']['PaginatedMeta']
    }
    /** @description Customer reference. */
    CustomerReference: {
      id: components['schemas']['ULID']
    }
    /**
     * RFC3339 Date-Time
     * Format: date-time
     * @description [RFC3339](https://tools.ietf.org/html/rfc3339) formatted date-time string in
     *     UTC.
     * @example 2023-01-01T01:01:01.001Z
     */
    DateTime: Date
    /**
     * DateTime Field Filter
     * @description Filters on the given datetime (RFC-3339) field value. All properties are
     *     optional; provide exactly one to specify the comparison.
     */
    DateTimeFieldFilter:
      | components['schemas']['DateTime']
      | {
          /** @description Value strictly equals given RFC-3339 formatted timestamp in UTC. */
          eq?: components['schemas']['DateTime']
          /** @description Value is less than the given RFC-3339 formatted timestamp in UTC. */
          lt?: components['schemas']['DateTime']
          /** @description Value is less than or equal to the given RFC-3339 formatted timestamp in UTC. */
          lte?: components['schemas']['DateTime']
          /** @description Value is greater than the given RFC-3339 formatted timestamp in UTC. */
          gt?: components['schemas']['DateTime']
          /** @description Value is greater than or equal to the given RFC-3339 formatted timestamp in UTC. */
          gte?: components['schemas']['DateTime']
        }
    /**
     * External Resource Key
     * @description ExternalResourceKey is a unique string that is used to identify a resource in an
     *     external system.
     * @example 019ae40f-4258-7f15-9491-842f42a7d6ac
     */
    ExternalResourceKey: string
    /** @description A capability or billable dimension offered by a provider. */
    Feature: {
      readonly id: components['schemas']['ULID']
      /**
       * @description Display name of the resource.
       *
       *     Between 1 and 256 characters.
       */
      name: string
      /**
       * @description Optional description of the resource.
       *
       *     Maximum 1024 characters.
       */
      description?: string
      labels?: components['schemas']['Labels']
      /** @description An ISO-8601 timestamp representation of entity creation date. */
      readonly created_at: components['schemas']['DateTime']
      /** @description An ISO-8601 timestamp representation of entity last update date. */
      readonly updated_at: components['schemas']['DateTime']
      /** @description An ISO-8601 timestamp representation of entity deletion date. */
      readonly deleted_at?: components['schemas']['DateTime']
      key: components['schemas']['ResourceKey']
      /**
       * Meter reference
       * @description The meter that the feature is associated with and based on which usage is
       *     calculated. If not specified, the feature is static.
       */
      meter?: components['schemas']['FeatureMeterReference']
      /**
       * Unit cost
       * @description Optional per-unit cost configuration. Use "manual" for a fixed per-unit cost, or
       *     "llm" to look up cost from the LLM cost database based on meter group-by
       *     properties.
       */
      unit_cost?: components['schemas']['BillingFeatureUnitCost']
    }
    /** @description Result of a feature cost query. */
    FeatureCostQueryResult: {
      /** @description Start of the queried period. */
      from?: components['schemas']['DateTime']
      /** @description End of the queried period. */
      to?: components['schemas']['DateTime']
      /** @description The cost data rows. */
      data: components['schemas']['FeatureCostQueryRow'][]
    }
    /** @description A row in the result of a feature cost query. */
    FeatureCostQueryRow: {
      /** @description The metered usage value for the period. */
      usage: components['schemas']['Numeric']
      /**
       * @description The computed cost amount (usage × unit cost). Null when pricing is not available
       *     for the given combination of dimensions.
       */
      cost: (string & components['schemas']['Numeric']) | null
      /** @description The currency code of the cost amount. */
      currency: components['schemas']['CurrencyCode']
      /**
       * @description Detail message when cost amount is null, explaining why the cost could not be
       *     resolved.
       */
      detail?: string
      /** @description The start of the time bucket the value is aggregated over. */
      from: components['schemas']['DateTime']
      /** @description The end of the time bucket the value is aggregated over. */
      to: components['schemas']['DateTime']
      /**
       * @description The dimensions the value is aggregated over. `subject` and `customer_id` are
       *     reserved dimensions.
       */
      dimensions: {
        [key: string]: string
      }
    }
    /** @description Reference to a meter associated with a feature. */
    FeatureMeterReference: {
      /**
       * Meter ID
       * @description The ID of the meter to associate with this feature.
       */
      id: components['schemas']['ULID']
      /**
       * Meter dimensions filters
       * @description Filters to apply to the dimensions of the meter.
       */
      filters?: {
        [key: string]: components['schemas']['QueryFilterStringMapItem']
      }
    }
    /** @description Page paginated response. */
    FeaturePagePaginatedResponse: {
      data: components['schemas']['Feature'][]
      meta: components['schemas']['PaginatedMeta']
    }
    /** @description Feature reference. */
    FeatureReferenceItem: {
      id: components['schemas']['ULID']
    }
    /** @description Filter options for getting a credit balance. */
    GetCreditBalanceParamsFilter: {
      /** @description Filter credit balance by currency. */
      currency?: components['schemas']['StringFieldFilterExact']
    }
    /** @description Access status for a single feature. */
    GovernanceFeatureAccess: {
      /**
       * Has access
       * @description Whether the customer currently has access to the feature.
       *
       *     `true` for boolean and static entitlements that are available, and for metered
       *     entitlements with remaining balance. `false` when the feature is unavailable,
       *     the usage limit has been reached, or (when applicable) credits have been
       *     exhausted.
       */
      readonly has_access: boolean
      /**
       * Reason
       * @description Optional reason when the customer does not have access to the feature. Populated
       *     when `has_access` is `false`.
       */
      readonly reason?: components['schemas']['GovernanceFeatureAccessReason']
    }
    /** @description Reason a feature is not accessible to a customer. */
    GovernanceFeatureAccessReason: {
      /**
       * Code
       * @description Machine-readable error code.
       */
      readonly code: components['schemas']['GovernanceFeatureAccessReasonCode']
      /**
       * Message
       * @description Human-readable description of the error.
       */
      readonly message: string
      /**
       * Attributes
       * @description Additional structured context.
       */
      readonly attributes?: {
        [key: string]: unknown
      }
    }
    /**
     * @description Machine-readable reason code for denied feature access.
     * @enum {string}
     */
    GovernanceFeatureAccessReasonCode:
      | 'unknown'
      | 'usage_limit_reached'
      | 'feature_unavailable'
      | 'feature_not_found'
      | 'no_credit_available'
    /** @description Query error within a partially successful governance query response. */
    GovernanceQueryError: {
      /**
       * Code
       * @description Machine-readable error code.
       */
      readonly code: components['schemas']['GovernanceQueryErrorCode']
      /**
       * Message
       * @description Human-readable description of the error.
       */
      readonly message: string
      /**
       * Attributes
       * @description Additional structured context.
       */
      readonly attributes?: {
        [key: string]: unknown
      }
      /**
       * Customer identifier
       * @description The customer identifier from the request that produced this error.
       */
      readonly customer?: string
    }
    /**
     * @description Error code for a governance query failure.
     * @enum {string}
     */
    GovernanceQueryErrorCode: 'unknown' | 'customer_not_found'
    /** @description Query to evaluate feature access for a list of customers. */
    GovernanceQueryRequest: {
      /**
       * Include credits
       * @description Whether to include credit balance availability for each resolved customer. When
       *     true, each feature evaluation includes credit balance checks.
       *
       *     Defaults to `false`.
       * @default false
       */
      include_credits?: boolean
      /** Customer */
      customer: components['schemas']['GovernanceQueryRequestCustomers']
      /** Feature */
      feature?: components['schemas']['GovernanceQueryRequestFeatures']
    }
    /** @description List of customer identifiers to evaluate access for. */
    GovernanceQueryRequestCustomers: {
      /**
       * Customer keys and usage-attribution subjects
       * @description Each entry can be a customer `key` or a usage-attribution subject `key`.
       *     Identifiers that cannot be resolved to a customer are reported in the response
       *     `errors` array.
       */
      keys: string[]
    }
    /**
     * @description Optional list of feature keys to evaluate access for. If omitted, all features
     *     available in the organization are returned. Providing this list is recommended
     *     to reduce the response size and the load on the backend services.
     */
    GovernanceQueryRequestFeatures: {
      /**
       * Feature Keys
       * @description List of feature keys to evaluate access for.
       */
      keys: string[]
    }
    /** @description Response of the governance query. */
    GovernanceQueryResponse: {
      /**
       * Data
       * @description Access evaluation results, one entry per resolved customer.
       */
      readonly data: components['schemas']['GovernanceQueryResult'][]
      /**
       * Errors
       * @description Partial errors encountered while processing the request.
       */
      readonly errors: components['schemas']['GovernanceQueryError'][]
      /**
       * Meta
       * @description Pagination metadata. The endpoint may return a partial response if the full
       *     response would exceed server-side limits.
       */
      readonly meta: components['schemas']['CursorMeta']
    }
    /** @description Access evaluation result for a single resolved customer. */
    GovernanceQueryResult: {
      /**
       * Matched identifiers
       * @description The list of identifiers from the request that resolved to this customer. Each
       *     entry is either the customer `key` or one of its usage-attribution subject
       *     `key`s.
       *
       *     Duplicate or aliased identifiers that resolve to the same customer collapse to a
       *     single result entry, with every requested identifier listed here.
       */
      readonly matched: string[]
      /**
       * Customer
       * @description The customer the matched identifiers resolved to.
       */
      readonly customer: components['schemas']['BillingCustomer']
      /**
       * Features
       * @description Map of features with their access status.
       *
       *     Map keys are the feature keys requested in `feature.keys`, or every feature
       *     `key` available in the organization when the feature filter was omitted.
       */
      readonly features: {
        [key: string]: components['schemas']['GovernanceFeatureAccess']
      }
      /**
       * Updated at
       * @description Timestamp of the most recent change to the customer's access state reflected in
       *     this result.
       */
      readonly updated_at: components['schemas']['DateTime']
    }
    /**
     * ISO 8601 Duration
     * Format: ISO8601
     * @description [ISO 8601 Duration](https://docs.digi.com/resources/documentation/digidocs/90001488-13/reference/r_iso_8601_duration_format.htm)
     *     string.
     * @example P1Y
     */
    ISO8601Duration: string
    /** @description Cursor paginated response. */
    IngestedEventPaginatedResponse: {
      data: components['schemas']['MeteringIngestedEvent'][]
      meta: components['schemas']['CursorMeta']
    }
    /** @description LLM Model */
    LLMCostModel: {
      /** @description Identifier of the model, e.g., "gpt-4", "claude-3-5-sonnet". */
      id: string
      /** @description Name of the model, e.g., "GPT-4", "Claude 3.5 Sonnet". */
      name: string
    }
    /** @description Token pricing for an LLM model, denominated per token. */
    LLMCostModelPricing: {
      /** @description Input price per token (USD). */
      input_per_token: components['schemas']['Numeric']
      /** @description Output price per token (USD). */
      output_per_token: components['schemas']['Numeric']
      /** @description Cache read price per token (USD). */
      cache_read_per_token?: components['schemas']['Numeric']
      /** @description Cache write price per token (USD). */
      cache_write_per_token?: components['schemas']['Numeric']
      /** @description Reasoning output price per token (USD). */
      reasoning_per_token?: components['schemas']['Numeric']
    }
    /**
     * @description Input for creating a per-namespace price override. Unique per provider, model
     *     and currency. If an override already exists for the given provider, model and
     *     currency, it will be updated. If an override does not exist, it will be created.
     */
    LLMCostOverrideCreate: {
      /** @description Provider/vendor of the model. */
      provider: string
      /** @description Canonical model identifier. */
      model_id: string
      /** @description Human-readable model name. */
      model_name?: string
      /** @description Token pricing data. */
      pricing: components['schemas']['LLMCostModelPricing']
      /** @description Currency code. */
      currency: components['schemas']['CurrencyCode']
      /** @description When this override becomes effective. */
      effective_from: components['schemas']['DateTime']
      /** @description When this override expires. */
      effective_to?: components['schemas']['DateTime']
    }
    /**
     * @description An LLM cost price record, representing the cost per token for a specific model
     *     from a specific provider.
     */
    LLMCostPrice: {
      /** @description Unique identifier. */
      readonly id: components['schemas']['ULID']
      /** @description Provider of the model. */
      readonly provider: components['schemas']['LLMCostProvider']
      /** @description The model. */
      readonly model: components['schemas']['LLMCostModel']
      /** @description Token pricing data. */
      readonly pricing: components['schemas']['LLMCostModelPricing']
      /** @description Currency code (currently always "USD"). */
      readonly currency: components['schemas']['CurrencyCode']
      /** @description Where this price came from. */
      readonly source: components['schemas']['LLMCostPriceSource']
      /** @description When this price becomes effective. */
      readonly effective_from: components['schemas']['DateTime']
      /** @description When this price expires. Omitted when the price is currently effective. */
      readonly effective_to?: components['schemas']['DateTime']
      /** @description Creation timestamp. */
      readonly created_at: components['schemas']['DateTime']
      /** @description Last update timestamp. */
      readonly updated_at: components['schemas']['DateTime']
    }
    /**
     * @description Identifies where an LLM cost price came from.
     * @enum {string}
     */
    LLMCostPriceSource: 'manual' | 'system'
    /** @description LLM Provider */
    LLMCostProvider: {
      /** @description Identifier of the provider, e.g., "openai", "anthropic". */
      id: string
      /** @description Name of the provider, e.g., "OpenAI", "Anthropic". */
      name: string
    }
    /** @description Filter options for listing add-ons. */
    ListAddonsParamsFilter: {
      id?: components['schemas']['ULIDFieldFilter']
      key?: components['schemas']['StringFieldFilter']
      name?: components['schemas']['StringFieldFilter']
      status?: components['schemas']['StringFieldFilterExact']
      currency?: components['schemas']['StringFieldFilterExact']
    }
    /** @description Filter options for listing charges. */
    ListChargesParamsFilter: {
      /**
       * @description Filter charges by status.
       *
       *     Supported statuses are:
       *
       *     - `created`
       *     - `active`
       *     - `final`
       *     - `deleted`
       *
       *     If omitted, all statuses are returned except for `deleted`.
       */
      status?: components['schemas']['StringFieldFilterExact']
    }
    /** @description Filter options for listing cost bases. */
    ListCostBasesParamsFilter: {
      /** @description Filter cost bases by fiat currency code. */
      fiat_code?: components['schemas']['CurrencyCode']
    }
    /** @description Filter options for listing credit grants. */
    ListCreditGrantsParamsFilter: {
      /** @description Filter credit grants by status. */
      status?: components['schemas']['BillingCreditGrantStatus']
      /** @description Filter credit grants by currency. */
      currency?: components['schemas']['CurrencyCode']
    }
    /** @description Filter options for listing credit transactions. */
    ListCreditTransactionsParamsFilter: {
      /** @description Filter credit transactions by type. */
      type?: components['schemas']['BillingCreditTransactionType']
      /** @description Filter credit transactions by currency. */
      currency?: components['schemas']['BillingCurrencyCode']
    }
    /** @description Filter options for listing currencies. */
    ListCurrenciesParamsFilter: {
      type?: components['schemas']['BillingCurrencyType']
      code?: components['schemas']['StringFieldFilter']
    }
    /** @description List customer entitlement access response data. */
    ListCustomerEntitlementAccessResponseData: {
      /** @description The list of entitlement access results. */
      readonly data: components['schemas']['BillingEntitlementAccessResult'][]
    }
    /** @description Filter options for listing customers. */
    ListCustomersParamsFilter: {
      key?: components['schemas']['StringFieldFilter']
      name?: components['schemas']['StringFieldFilter']
      primary_email?: components['schemas']['StringFieldFilter']
      usage_attribution_subject_key?: components['schemas']['StringFieldFilter']
      plan_key?: components['schemas']['StringFieldFilter']
      billing_profile_id?: components['schemas']['ULIDFieldFilter']
    }
    /** @description Filter options for listing ingested events. */
    ListEventsParamsFilter: {
      /** @description Filter events by ID. */
      id?: components['schemas']['StringFieldFilter']
      /** @description Filter events by source. */
      source?: components['schemas']['StringFieldFilter']
      /** @description Filter events by subject. */
      subject?: components['schemas']['StringFieldFilter']
      /** @description Filter events by type. */
      type?: components['schemas']['StringFieldFilter']
      /** @description Filter events by the associated customer ID. */
      customer_id?: components['schemas']['ULIDFieldFilter']
      /** @description Filter events by event time. */
      time?: components['schemas']['DateTimeFieldFilter']
      /** @description Filter events by the time the event was ingested. */
      ingested_at?: components['schemas']['DateTimeFieldFilter']
      /** @description Filter events by the time the event was stored. */
      stored_at?: components['schemas']['DateTimeFieldFilter']
    }
    /** @description Filter options for listing features. */
    ListFeatureParamsFilter: {
      meter_id?: components['schemas']['ULIDFieldFilter']
      key?: components['schemas']['StringFieldFilter']
      name?: components['schemas']['StringFieldFilter']
    }
    /** @description Filter options for listing LLM cost prices. */
    ListLLMCostPricesParamsFilter: {
      /** @description Filter by provider. e.g. ?filter[provider][eq]=openai */
      provider?: components['schemas']['StringFieldFilter']
      /** @description Filter by model ID. e.g. ?filter[model_id][eq]=gpt-4 */
      model_id?: components['schemas']['StringFieldFilter']
      /** @description Filter by model name. e.g. ?filter[model_name][contains]=gpt */
      model_name?: components['schemas']['StringFieldFilter']
      /** @description Filter by currency code. e.g. ?filter[currency][eq]=USD */
      currency?: components['schemas']['StringFieldFilter']
      /** @description Filter by source. e.g. ?filter[source][eq]=system */
      source?: components['schemas']['StringFieldFilter']
    }
    /** @description Filter options for listing meters. */
    ListMetersParamsFilter: {
      /** @description Filter meters by key. */
      key?: components['schemas']['StringFieldFilter']
      /** @description Filter meters by name. */
      name?: components['schemas']['StringFieldFilter']
    }
    /** @description Filter options for listing plans. */
    ListPlansParamsFilter: {
      key?: components['schemas']['StringFieldFilter']
      name?: components['schemas']['StringFieldFilter']
      status?: components['schemas']['StringFieldFilterExact']
      currency?: components['schemas']['StringFieldFilterExact']
    }
    /** @description Filter options for listing subscriptions. */
    ListSubscriptionsParamsFilter: {
      id?: components['schemas']['ULIDFieldFilter']
      customer_id?: components['schemas']['ULIDFieldFilter']
      status?: components['schemas']['StringFieldFilterExact']
      plan_id?: components['schemas']['ULIDFieldFilter']
      plan_key?: components['schemas']['StringFieldFilterExact']
    }
    /**
     * @description A meter is a configuration that defines how to match and aggregate events.
     * @example {
     *       "id": "01G65Z755AFWAKHE12NY0CQ9FH",
     *       "key": "tokens_total",
     *       "name": "Tokens Total",
     *       "description": "AI Token Usage",
     *       "aggregation": "sum",
     *       "event_type": "prompt",
     *       "value_property": "$.tokens",
     *       "dimensions": {
     *         "model": "$.model",
     *         "type": "$.type"
     *       },
     *       "created_at": "2024-01-01T01:01:01.001Z",
     *       "updated_at": "2024-01-01T01:01:01.001Z"
     *     }
     */
    Meter: {
      readonly id: components['schemas']['ULID']
      /**
       * @description Display name of the resource.
       *
       *     Between 1 and 256 characters.
       */
      name: string
      /**
       * @description Optional description of the resource.
       *
       *     Maximum 1024 characters.
       */
      description?: string
      labels?: components['schemas']['Labels']
      /** @description An ISO-8601 timestamp representation of entity creation date. */
      readonly created_at: components['schemas']['DateTime']
      /** @description An ISO-8601 timestamp representation of entity last update date. */
      readonly updated_at: components['schemas']['DateTime']
      /** @description An ISO-8601 timestamp representation of entity deletion date. */
      readonly deleted_at?: components['schemas']['DateTime']
      key: components['schemas']['ResourceKey']
      /** @description The aggregation type to use for the meter. */
      aggregation: components['schemas']['MeterAggregation']
      /**
       * @description The event type to include in the aggregation.
       * @example prompt
       */
      event_type: string
      /**
       * @description The date since the meter should include events. Useful to skip old events. If
       *     not specified, all historical events are included.
       */
      events_from?: components['schemas']['DateTime']
      /**
       * @description JSONPath expression to extract the value from the ingested event's data
       *     property.
       *
       *     The ingested value for sum, avg, min, and max aggregations is a number or a
       *     string that can be parsed to a number.
       *
       *     For unique_count aggregation, the ingested value must be a string. For count
       *     aggregation the value_property is ignored.
       * @example $.tokens
       */
      value_property?: string
      /**
       * @description Named JSONPath expressions to extract the group by values from the event data.
       *
       *     Keys must be unique and consist only alphanumeric and underscore characters.
       * @example {
       *       "type": "$.type"
       *     }
       */
      dimensions?: {
        [key: string]: string
      }
    }
    /**
     * @description The aggregation type to use for the meter.
     * @enum {string}
     */
    MeterAggregation:
      | 'sum'
      | 'count'
      | 'unique_count'
      | 'avg'
      | 'min'
      | 'max'
      | 'latest'
    /** @description Page paginated response. */
    MeterPagePaginatedResponse: {
      data: components['schemas']['Meter'][]
      meta: components['schemas']['PaginatedMeta']
    }
    /** @description Filters to apply to a meter query. */
    MeterQueryFilters: {
      /**
       * @description Filters to apply to the dimensions of the query. For `subject` and `customer_id`
       *     only equals ("eq", "in") comparisons are supported.
       */
      dimensions?: {
        [key: string]: components['schemas']['QueryFilterStringMapItem']
      }
    }
    /**
     * @description The granularity of the time grouping. Time durations are specified in ISO 8601
     *     format.
     * @enum {string}
     */
    MeterQueryGranularity: 'PT1M' | 'PT1H' | 'P1D' | 'P1M'
    /**
     * @description A meter query request.
     * @example {
     *       "from": "2023-01-01T00:00:00Z",
     *       "to": "2023-01-02T00:00:00Z",
     *       "granularity": "P1D",
     *       "time_zone": "UTC"
     *     }
     */
    MeterQueryRequest: {
      /** @description The start of the period the usage is queried from. */
      from?: components['schemas']['DateTime']
      /** @description The end of the period the usage is queried to. */
      to?: components['schemas']['DateTime']
      /**
       * @description The size of the time buckets to group the usage into. If not specified, the
       *     usage is aggregated over the entire period.
       */
      granularity?: components['schemas']['MeterQueryGranularity']
      /**
       * @description The value is the name of the time zone as defined in the IANA Time Zone Database
       *     (http://www.iana.org/time-zones). The time zone is used to determine the start
       *     and end of the time buckets. If not specified, the UTC timezone will be used.
       * @default UTC
       */
      time_zone?: string
      /**
       * @description The dimensions to group the results by.
       * @example [
       *       "model",
       *       "type"
       *     ]
       */
      group_by_dimensions?: string[]
      /** @description Filters to apply to the query. */
      filters?: components['schemas']['MeterQueryFilters']
    }
    /**
     * @description Meter query result.
     * @example {
     *       "from": "2023-01-01T00:00:00Z",
     *       "to": "2023-01-02T00:00:00Z",
     *       "data": [
     *         {
     *           "value": "12.3456",
     *           "from": "2023-01-01T00:00:00Z",
     *           "to": "2023-01-02T00:00:00Z",
     *           "dimensions": {
     *             "customer_id": "01G65Z755AFWAKHE12NY0CQ9FH",
     *             "model": "gpt-4-turbo",
     *             "type": "input"
     *           }
     *         }
     *       ]
     *     }
     */
    MeterQueryResult: {
      /** @description The start of the period the usage is queried from. */
      from?: components['schemas']['DateTime']
      /** @description The end of the period the usage is queried to. */
      to?: components['schemas']['DateTime']
      /** @description The usage data. If no data is available, an empty array is returned. */
      data: components['schemas']['MeterQueryRow'][]
    }
    /**
     * @description A row in the result of a meter query.
     * @example {
     *       "value": "12.3456",
     *       "from": "2023-01-01T00:00:00Z",
     *       "to": "2023-01-02T00:00:00Z",
     *       "dimensions": {
     *         "customer_id": "01G65Z755AFWAKHE12NY0CQ9FH",
     *         "model": "gpt-4-turbo",
     *         "type": "input"
     *       }
     *     }
     */
    MeterQueryRow: {
      /** @description The aggregated value. */
      value: components['schemas']['Numeric']
      /** @description The start of the time bucket the value is aggregated over. */
      from: components['schemas']['DateTime']
      /** @description The end of the time bucket the value is aggregated over. */
      to: components['schemas']['DateTime']
      /**
       * @description The dimensions the value is aggregated over. `subject` and `customer_id` are
       *     reserved dimensions.
       */
      dimensions: {
        [key: string]: string
      }
    }
    /**
     * Metering Event
     * @description Metering event following the CloudEvents specification.
     * @example {
     *       "specversion": "1.0",
     *       "id": "5c10fade-1c9e-4d6c-8275-c52c36731d3c",
     *       "source": "service-name",
     *       "type": "prompt",
     *       "subject": "customer-id",
     *       "time": "2023-01-01T01:01:01.001Z",
     *       "data": {
     *         "prompt": "Hello, world!",
     *         "tokens": 100,
     *         "model": "gpt-4o",
     *         "type": "input"
     *       }
     *     }
     */
    MeteringEvent: {
      /**
       * @description Identifies the event.
       * @example 5c10fade-1c9e-4d6c-8275-c52c36731d3c
       */
      id: string
      /**
       * Format: uri-reference
       * @description Identifies the context in which an event happened.
       * @example service-name
       */
      source: string
      /**
       * @description The version of the CloudEvents specification which the event uses.
       * @default 1.0
       * @example 1.0
       */
      specversion: string
      /**
       * @description Contains a value describing the type of event related to the originating
       *     occurrence.
       * @example com.example.someevent
       */
      type: string
      /**
       * @description Content type of the CloudEvents data value. Only the value "application/json" is
       *     allowed over HTTP.
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
       * @description Describes the subject of the event in the context of the event producer
       *     (identified by source).
       * @example customer-id
       */
      subject: string
      /**
       * @description Timestamp of when the occurrence happened. Must adhere to RFC 3339.
       * @example 2023-01-01T01:01:01.001Z
       */
      time?: (string & components['schemas']['DateTime']) | null
      /** @description The event payload. Optional, if present it must be a JSON object. */
      data?: {
        [key: string]: unknown
      } | null
    }
    /**
     * Ingested Event
     * @description An ingested metering event with ingestion metadata.
     * @example {
     *       "event": {
     *         "id": "5c10fade-1c9e-4d6c-8275-c52c36731d3c",
     *         "source": "service-name",
     *         "specversion": "1.0",
     *         "type": "prompt",
     *         "subject": "customer_key",
     *         "time": "2023-01-01T01:01:01.001Z"
     *       },
     *       "customer": {
     *         "id": "01G65Z755AFWAKHE12NY0CQ9FH"
     *       },
     *       "ingested_at": "2023-01-01T01:01:01.001Z",
     *       "stored_at": "2023-01-01T01:01:02.001Z"
     *     }
     */
    MeteringIngestedEvent: {
      /** @description The original event ingested. */
      event: components['schemas']['MeteringEvent']
      /** @description The customer if the event is associated with a customer. */
      customer?: components['schemas']['CustomerReference']
      /** @description The date and time the event was ingested and its processing started. */
      ingested_at: components['schemas']['DateTime']
      /** @description The date and time the event was stored in the database. */
      stored_at: components['schemas']['DateTime']
      /** @description The validation errors of the ingested event. */
      validation_errors?: components['schemas']['MeteringIngestedEventValidationError'][]
    }
    /** @description Event validation errors. */
    MeteringIngestedEventValidationError: {
      /**
       * Code
       * @description The machine readable code of the error.
       */
      readonly code: string
      /**
       * Message
       * @description The human readable description of the error.
       */
      readonly message: string
      /**
       * Attributes
       * @description Additional attributes.
       */
      readonly attributes?: {
        [key: string]: unknown
      }
    }
    /** @description Numeric represents an arbitrary precision number. */
    Numeric: string
    /**
     * @description Organization-level default tax code references.
     *
     *     Stores the default tax codes applied to specific billing contexts for this
     *     organization. Provisioned automatically when the organization is created.
     */
    OrganizationDefaultTaxCodes: {
      /**
       * Invoicing tax code
       * @description Default tax code for invoicing.
       */
      invoicing_tax_code: components['schemas']['TaxCodeReference']
      /**
       * Credit grant tax code
       * @description Default tax code for credit grants.
       */
      credit_grant_tax_code: components['schemas']['TaxCodeReference']
      /** @description Timestamp of creation. */
      readonly created_at: components['schemas']['DateTime']
      /** @description Timestamp of last update. */
      readonly updated_at: components['schemas']['DateTime']
    }
    /**
     * @description PlanAddon represents an association between a plan and an add-on, controlling
     *     which add-ons are available for purchase within a plan.
     */
    PlanAddon: {
      readonly id: components['schemas']['ULID']
      /**
       * @description Display name of the resource.
       *
       *     Between 1 and 256 characters.
       */
      name: string
      /**
       * @description Optional description of the resource.
       *
       *     Maximum 1024 characters.
       */
      description?: string
      labels?: components['schemas']['Labels']
      /** @description An ISO-8601 timestamp representation of entity creation date. */
      readonly created_at: components['schemas']['DateTime']
      /** @description An ISO-8601 timestamp representation of entity last update date. */
      readonly updated_at: components['schemas']['DateTime']
      /** @description An ISO-8601 timestamp representation of entity deletion date. */
      readonly deleted_at?: components['schemas']['DateTime']
      /**
       * Add-on
       * @description The add-on associated with the plan.
       */
      addon: components['schemas']['AddonReferenceItem']
      /**
       * From plan phase
       * @description The key of the plan phase from which the add-on becomes available for purchase.
       */
      from_plan_phase: components['schemas']['ResourceKey']
      /**
       * Max quantity
       * @description The maximum number of times the add-on can be purchased for the plan. For
       *     single-instance add-ons this field must be omitted. For multi-instance add-ons
       *     when omitted, unlimited quantity can be purchased.
       */
      max_quantity?: number
      /**
       * Validation errors
       * @description List of validation errors.
       */
      readonly validation_errors?: components['schemas']['ProductCatalogValidationError'][]
    }
    /** @description Page paginated response. */
    PlanAddonPagePaginatedResponse: {
      data: components['schemas']['PlanAddon'][]
      meta: components['schemas']['PaginatedMeta']
    }
    /** @description Page paginated response. */
    PlanPagePaginatedResponse: {
      data: components['schemas']['BillingPlan'][]
      meta: components['schemas']['PaginatedMeta']
    }
    /** @description Page paginated response. */
    PricePagePaginatedResponse: {
      data: components['schemas']['LLMCostPrice'][]
      meta: components['schemas']['PaginatedMeta']
    }
    /** @description Validation errors providing detailed description of the issue. */
    ProductCatalogValidationError: {
      /**
       * Code
       * @description Machine-readable error code.
       */
      readonly code: string
      /**
       * Message
       * @description Human-readable description of the error.
       */
      readonly message: string
      /**
       * Attributes
       * @description Additional structured context.
       */
      readonly attributes?: {
        [key: string]: unknown
      }
      /**
       * Field
       * @description The path to the field.
       * @example addons/pro/ratecards/token/featureKey
       */
      readonly field: string
    }
    /**
     * Query String Filter
     * @description A query filter for a string attribute. Operators are mutually exclusive, only
     *     one operator is allowed at a time.
     */
    QueryFilterString: {
      /** @description The attribute equals the provided value. */
      eq?: string
      /** @description The attribute does not equal the provided value. */
      neq?: string
      /** @description The attribute is one of the provided values. */
      in?: string[]
      /** @description The attribute is not one of the provided values. */
      nin?: string[]
      /** @description The attribute contains the provided value. */
      contains?: string
      /** @description The attribute does not contain the provided value. */
      ncontains?: string
      /** @description Combines the provided filters with a logical AND. */
      and?: components['schemas']['QueryFilterString'][]
      /** @description Combines the provided filters with a logical OR. */
      or?: components['schemas']['QueryFilterString'][]
    }
    /**
     * Query String Map Item Filter
     * @description A query filter for an item in a string map attribute. Operators are mutually
     *     exclusive, only one operator is allowed at a time.
     */
    QueryFilterStringMapItem: {
      /** @description The attribute exists. */
      exists?: boolean
      /** @description The attribute equals the provided value. */
      eq?: string
      /** @description The attribute does not equal the provided value. */
      neq?: string
      /** @description The attribute is one of the provided values. */
      in?: string[]
      /** @description The attribute is not one of the provided values. */
      nin?: string[]
      /** @description The attribute contains the provided value. */
      contains?: string
      /** @description The attribute does not contain the provided value. */
      ncontains?: string
      /** @description Combines the provided filters with a logical AND. */
      and?: components['schemas']['QueryFilterString'][]
      /** @description Combines the provided filters with a logical OR. */
      or?: components['schemas']['QueryFilterString'][]
    }
    /** @description Recurring period with an anchor and an interval. */
    RecurringPeriod: {
      /**
       * Anchor time
       * @description A date-time anchor to base the recurring period on.
       * @example 2023-01-01T01:01:01.001Z
       */
      anchor: components['schemas']['DateTime']
      /**
       * Interval in ISO 8601 duration format
       * @description The interval duration in ISO 8601 format.
       * @example P1M
       */
      interval: components['schemas']['ISO8601Duration']
    }
    /**
     * Resource Key
     * @description A key is a unique string that is used to identify a resource.
     * @example resource_key
     */
    ResourceKey: string
    /**
     * Resource managed by
     * @description Identifies which system manages a resource.
     *
     *     Values:
     *
     *     - `manual`: The resource is managed manually (overridden by our API users).
     *     - `system`: The resource is managed by the system.
     *     - `subscription`: The resource is managed by the subscription.
     * @enum {string}
     */
    ResourceManagedBy: 'manual' | 'system' | 'subscription'
    /**
     * @description Filters on the given string field value by either exact or fuzzy match. All
     *     properties are optional; provide exactly one to specify the comparison.
     */
    StringFieldFilter:
      | string
      | {
          /** @description Value strictly equals the given string value. */
          eq?: string
          /** @description Value does not equal the given string value. */
          neq?: string
          /** @description Value contains the given string value (fuzzy match). */
          contains?: string
          /**
           * Format: ArrayEncoding.commaDelimited
           * @description Returns entities that fuzzy-match any of the comma-delimited phrases in the
           *     filter string.
           */
          ocontains?: string
          /**
           * Format: ArrayEncoding.commaDelimited
           * @description Returns entities that exact match any of the comma-delimited phrases in the
           *     filter string.
           */
          oeq?: string
          /** @description Value is greater than the given string value (lexicographic compare). */
          gt?: string
          /**
           * @description Value is greater than or equal to the given string value (lexicographic
           *     compare).
           */
          gte?: string
          /** @description Value is less than the given string value (lexicographic compare). */
          lt?: string
          /** @description Value is less than or equal to the given string value (lexicographic compare). */
          lte?: string
          /**
           * @description When true, the field must be present (non-null); when false, the field must be
           *     absent (null).
           */
          exists?: boolean
        }
    /**
     * String Field Filter Exact
     * @description Filters on the given string field value by exact match. All properties are
     *     optional; provide exactly one to specify the comparison.
     */
    StringFieldFilterExact:
      | string
      | {
          /** @description Value strictly equals the given string value. */
          eq?: string
          /**
           * Format: ArrayEncoding.commaDelimited
           * @description Returns entities that exact match any of the comma-delimited phrases in the
           *     filter string.
           */
          oeq?: string
          /** @description Value does not equal the given string value. */
          neq?: string
        }
    /** @description Addon purchased with a subscription. */
    SubscriptionAddon: {
      readonly id: components['schemas']['ULID']
      /**
       * @description Display name of the resource.
       *
       *     Between 1 and 256 characters.
       */
      name: string
      /**
       * @description Optional description of the resource.
       *
       *     Maximum 1024 characters.
       */
      description?: string
      labels?: components['schemas']['Labels']
      /** @description An ISO-8601 timestamp representation of entity creation date. */
      readonly created_at: components['schemas']['DateTime']
      /** @description An ISO-8601 timestamp representation of entity last update date. */
      readonly updated_at: components['schemas']['DateTime']
      /** @description An ISO-8601 timestamp representation of entity deletion date. */
      readonly deleted_at?: components['schemas']['DateTime']
      /**
       * Add-on
       * @description The add-on associated with the subscription.
       */
      addon: components['schemas']['AddonReferenceItem']
      /**
       * Quantity
       * @description The quantity of the add-on. Always 1 for single instance add-ons.
       */
      quantity: number
      /**
       * @description An ISO-8601 timestamp representation of which point in time the quantity was
       *     resolved to.
       */
      readonly quantity_at: components['schemas']['DateTime']
      /** @description An ISO-8601 timestamp representation of the cadence start of the resource. */
      readonly active_from: components['schemas']['DateTime']
      /** @description An ISO-8601 timestamp representation of the cadence end of the resource. */
      readonly active_to?: components['schemas']['DateTime']
    }
    /** @description Page paginated response. */
    SubscriptionAddonPagePaginatedResponse: {
      data: components['schemas']['SubscriptionAddon'][]
      meta: components['schemas']['PaginatedMeta']
    }
    /** @description Page paginated response. */
    SubscriptionPagePaginatedResponse: {
      data: components['schemas']['BillingSubscription'][]
      meta: components['schemas']['PaginatedMeta']
    }
    /** @description Page paginated response. */
    TaxCodePagePaginatedResponse: {
      data: components['schemas']['BillingTaxCode'][]
      meta: components['schemas']['PaginatedMeta']
    }
    /** @description TaxCode reference. */
    TaxCodeReference: {
      id: components['schemas']['ULID']
    }
    /** @description TaxCode reference. */
    TaxCodeReferenceItem: {
      id: components['schemas']['ULID']
    }
    /**
     * ULID
     * @description ULID (Universally Unique Lexicographically Sortable Identifier).
     * @example 01G65Z755AFWAKHE12NY0CQ9FH
     */
    ULID: string
    /**
     * ULID Field Filter
     * @description Filters on the given ULID field value by exact match. All properties are
     *     optional; provide exactly one to specify the comparison.
     */
    ULIDFieldFilter:
      | components['schemas']['ULID']
      | {
          /** @description Value strictly equals the given ULID value. */
          eq?: components['schemas']['ULID']
          /**
           * Format: ArrayEncoding.commaDelimited
           * @description Returns entities that exact match any of the comma-delimited ULIDs in the filter
           *     string.
           */
          oeq?: string
          /** @description Value does not equal the given ULID value. */
          neq?: components['schemas']['ULID']
        }
    /**
     * @description Request body for updating the external payment settlement status of a credit
     *     grant.
     */
    UpdateCreditGrantExternalSettlementRequest: {
      /** @description The new payment settlement status. */
      status: components['schemas']['BillingCreditPurchasePaymentSettlementStatus']
    }
    /**
     * @description Request body for updating a feature. Currently only the unit_cost field can be
     *     updated.
     */
    UpdateFeatureRequest: {
      /**
       * Unit cost
       * @description Optional per-unit cost configuration. Use "manual" for a fixed per-unit cost, or
       *     "llm" to look up cost from the LLM cost database based on meter group-by
       *     properties. Set to `null` to clear the existing unit cost; omit to leave it
       *     unchanged.
       */
      unit_cost?: components['schemas']['BillingFeatureUnitCost'] | null
    }
    /** @description Meter update request. */
    UpdateMeterRequest: {
      /**
       * @description Display name of the resource.
       *
       *     Between 1 and 256 characters.
       */
      name?: string
      /**
       * @description Optional description of the resource.
       *
       *     Maximum 1024 characters.
       */
      description?: string
      labels?: components['schemas']['Labels']
      /**
       * @description Named JSONPath expressions to extract the group by values from the event data.
       *
       *     Keys must be unique and consist only alphanumeric and underscore characters.
       * @example {
       *       "type": "$.type"
       *     }
       */
      dimensions?: {
        [key: string]: string
      }
    }
    /** @description OrganizationDefaultTaxCodes update request. */
    UpdateOrganizationDefaultTaxCodesRequest: {
      /**
       * Invoicing tax code
       * @description Default tax code for invoicing.
       */
      invoicing_tax_code?: components['schemas']['TaxCodeReference']
      /**
       * Credit grant tax code
       * @description Default tax code for credit grants.
       */
      credit_grant_tax_code?: components['schemas']['TaxCodeReference']
    }
    /** @description Addon upsert request. */
    UpsertAddonRequest: {
      /**
       * @description Display name of the resource.
       *
       *     Between 1 and 256 characters.
       */
      name: string
      /**
       * @description Optional description of the resource.
       *
       *     Maximum 1024 characters.
       */
      description?: string
      labels?: components['schemas']['Labels']
      /**
       * The InstanceType of the add-ons. Can be "single" or "multiple".
       * @description The InstanceType of the add-ons. Can be "single" or "multiple".
       */
      instance_type: components['schemas']['AddonInstanceType']
      /**
       * Rate cards
       * @description The rate cards of the add-on.
       */
      rate_cards: components['schemas']['BillingRateCard'][]
    }
    /** @description AppCustomerData upsert request. */
    UpsertAppCustomerDataRequest: {
      /**
       * Stripe
       * @description Used if the customer has a linked Stripe app.
       */
      stripe?: components['schemas']['BillingAppCustomerDataStripe']
      /**
       * External invoicing
       * @description Used if the customer has a linked external invoicing app.
       */
      external_invoicing?: components['schemas']['BillingAppCustomerDataExternalInvoicing']
    }
    /** @description BillingProfile upsert request. */
    UpsertBillingProfileRequest: {
      /**
       * @description Display name of the resource.
       *
       *     Between 1 and 256 characters.
       */
      name: string
      /**
       * @description Optional description of the resource.
       *
       *     Maximum 1024 characters.
       */
      description?: string
      labels?: components['schemas']['Labels']
      /**
       * @description The name and contact information for the supplier this billing profile
       *     represents
       */
      supplier: components['schemas']['BillingParty']
      /** @description The billing workflow settings for this profile */
      workflow: components['schemas']['BillingWorkflow']
      /** @description Whether this is the default profile. */
      default: boolean
    }
    /** @description CustomerBillingData upsert request. */
    UpsertCustomerBillingDataRequest: {
      /**
       * Billing profile
       * @description The billing profile for the customer.
       *
       *     If not provided, the default billing profile will be used.
       */
      billing_profile?: components['schemas']['BillingProfileReference']
      /**
       * App customer data
       * @description App customer data.
       */
      app_data?: components['schemas']['BillingAppCustomerData']
    }
    /** @description Customer upsert request. */
    UpsertCustomerRequest: {
      /**
       * @description Display name of the resource.
       *
       *     Between 1 and 256 characters.
       */
      name: string
      /**
       * @description Optional description of the resource.
       *
       *     Maximum 1024 characters.
       */
      description?: string
      labels?: components['schemas']['Labels']
      /**
       * Usage Attribution
       * @description Mapping to attribute metered usage to the customer by the event subject.
       */
      usage_attribution?: components['schemas']['BillingCustomerUsageAttribution']
      /**
       * Primary Email
       * @description The primary email address of the customer.
       */
      primary_email?: string
      /**
       * Currency
       * @description Currency of the customer. Used for billing, tax and invoicing.
       */
      currency?: components['schemas']['CurrencyCode']
      /**
       * Billing Address
       * @description The billing address of the customer. Used for tax and invoicing.
       */
      billing_address?: components['schemas']['BillingAddress']
    }
    /** @description PlanAddon upsert request. */
    UpsertPlanAddonRequest: {
      /**
       * @description Display name of the resource.
       *
       *     Between 1 and 256 characters.
       */
      name: string
      /**
       * @description Optional description of the resource.
       *
       *     Maximum 1024 characters.
       */
      description?: string
      labels?: components['schemas']['Labels']
      /**
       * From plan phase
       * @description The key of the plan phase from which the add-on becomes available for purchase.
       */
      from_plan_phase: components['schemas']['ResourceKey']
      /**
       * Max quantity
       * @description The maximum number of times the add-on can be purchased for the plan. For
       *     single-instance add-ons this field must be omitted. For multi-instance add-ons
       *     when omitted, unlimited quantity can be purchased.
       */
      max_quantity?: number
    }
    /** @description Plan upsert request. */
    UpsertPlanRequest: {
      /**
       * @description Display name of the resource.
       *
       *     Between 1 and 256 characters.
       */
      name: string
      /**
       * @description Optional description of the resource.
       *
       *     Maximum 1024 characters.
       */
      description?: string
      labels?: components['schemas']['Labels']
      /**
       * Pro-rating enabled
       * @description Whether pro-rating is enabled for this plan.
       * @default true
       */
      pro_rating_enabled?: boolean
      /**
       * Plan phases
       * @description The plan phases define the pricing ramp for a subscription. A phase switch
       *     occurs only at the end of a billing period. At least one phase is required.
       */
      phases: components['schemas']['BillingPlanPhase'][]
    }
    /** @description TaxCode upsert request. */
    UpsertTaxCodeRequest: {
      /**
       * @description Display name of the resource.
       *
       *     Between 1 and 256 characters.
       */
      name: string
      /**
       * @description Optional description of the resource.
       *
       *     Maximum 1024 characters.
       */
      description?: string
      labels?: components['schemas']['Labels']
      /**
       * App type to tax code mappings
       * @description Mapping of app types to tax codes.
       */
      app_mappings: components['schemas']['BillingTaxCodeAppMapping'][]
    }
    /** @description Subject key. */
    UsageAttributionSubjectKey: string
    /**
     * SortQuery
     * @description The `asc` suffix is optional as the default sort order is ascending.
     *     The `desc` suffix is used to specify a descending order.
     *     Multiple sort attributes may be provided via a comma separated list.
     *     JSONPath notation may be used to specify a sub-attribute (eg: 'foo.bar desc').
     * @example created_at desc
     */
    SortQuery: string
    /**
     * Labels
     * @description Labels store metadata of an entity that can be used for filtering an entity list or for searching across entity types.
     *
     *     Keys must be of length 1-63 characters, and cannot start with "kong", "konnect", "mesh", "kic", or "_".
     * @example {
     *       "env": "test"
     *     }
     */
    Labels: {
      [key: string]: string
    }
    /** @description Contains pagination query parameters and the total number of objects returned. */
    PageMeta: {
      /** @example 1 */
      number: number
      /** @example 10 */
      size: number
      /** @example 100 */
      total: number
    }
    /**
     * PaginatedMeta
     * @description returns the pagination information
     */
    PaginatedMeta: {
      page: components['schemas']['PageMeta']
    }
    /**
     * Error
     * @description standard error
     */
    BaseError: {
      /**
       * @description The HTTP status code of the error. Useful when passing the response
       *     body to child properties in a frontend UI. Must be returned as an integer.
       */
      readonly status: number
      /**
       * @description A short, human-readable summary of the problem. It should not
       *     change between occurences of a problem, except for localization.
       *     Should be provided as "Sentence case" for direct use in the UI.
       */
      readonly title: string
      /** @description The error type. */
      readonly type?: string
      /**
       * @description Used to return the correlation ID back to the user, in the format
       *     kong:trace:<correlation_id>. This helps us find the relevant logs
       *     when a customer reports an issue.
       */
      readonly instance: string
      /**
       * @description A human readable explanation specific to this occurence of the problem.
       *     This field may contain request/entity data to help the user understand
       *     what went wrong. Enclose variable values in square brackets. Should be
       *     provided as "Sentence case" for direct use in the UI.
       */
      readonly detail: string
    }
    /**
     * @description invalid parameters rules
     * @enum {string|null}
     */
    InvalidRules:
      | 'required'
      | 'is_array'
      | 'is_base64'
      | 'is_boolean'
      | 'is_date_time'
      | 'is_integer'
      | 'is_null'
      | 'is_number'
      | 'is_object'
      | 'is_string'
      | 'is_uuid'
      | 'is_fqdn'
      | 'is_arn'
      | 'unknown_property'
      | 'missing_reference'
      | 'is_label'
      | 'matches_regex'
      | 'invalid'
      | 'is_supported_network_availability_zone_list'
      | 'is_supported_network_cidr_block'
      | 'is_supported_provider_region'
      | 'type'
      | null
    InvalidParameterStandard: {
      /** @example name */
      readonly field: string
      rule?: components['schemas']['InvalidRules']
      /** @example body */
      source?: string
      /** @example is a required field */
      readonly reason: string
    }
    InvalidParameterMinimumLength: {
      /** @example name */
      readonly field: string
      /**
       * @description invalid parameters rules
       * @enum {string}
       */
      readonly rule:
        | 'min_length'
        | 'min_digits'
        | 'min_lowercase'
        | 'min_uppercase'
        | 'min_symbols'
        | 'min_items'
        | 'min'
      /** @example 8 */
      minimum: number
      /** @example body */
      source?: string
      /** @example must have at least 8 characters */
      readonly reason: string
    }
    InvalidParameterMaximumLength: {
      /** @example name */
      readonly field: string
      /**
       * @description invalid parameters rules
       * @enum {string}
       */
      readonly rule: 'max_length' | 'max_items' | 'max'
      /** @example 8 */
      maximum: number
      /** @example body */
      source?: string
      /** @example must not have more than 8 characters */
      readonly reason: string
    }
    InvalidParameterChoiceItem: {
      /** @example name */
      readonly field: string
      /**
       * @description invalid parameters rules
       * @enum {string}
       */
      readonly rule: 'enum'
      /** @example is a required field */
      readonly reason: string
      readonly choices: unknown[]
      /** @example body */
      source?: string
    }
    InvalidParameterDependentItem: {
      /** @example name */
      readonly field: string
      /**
       * @description invalid parameters rules
       * @enum {string|null}
       */
      readonly rule: 'dependent_fields' | null
      /** @example is a required field */
      readonly reason: string
      readonly dependents: unknown[] | null
      /** @example body */
      source?: string
    }
    /** @description invalid parameters */
    InvalidParameters: (
      | components['schemas']['InvalidParameterStandard']
      | components['schemas']['InvalidParameterMinimumLength']
      | components['schemas']['InvalidParameterMaximumLength']
      | components['schemas']['InvalidParameterChoiceItem']
      | components['schemas']['InvalidParameterDependentItem']
    )[]
    BadRequestError: components['schemas']['BaseError'] & {
      invalid_parameters: components['schemas']['InvalidParameters']
    }
    UnauthorizedError: components['schemas']['BaseError'] & {
      /** @example 401 */
      status?: unknown
      /** @example Unauthorized */
      title?: unknown
      /** @example https://httpstatuses.com/401 */
      type?: unknown
      /** @example kong:trace:1234567890 */
      instance?: unknown
      /** @example Invalid credentials */
      detail?: unknown
    }
    ForbiddenError: components['schemas']['BaseError'] & {
      /** @example 403 */
      status?: unknown
      /** @example Forbidden */
      title?: unknown
      /** @example https://httpstatuses.com/403 */
      type?: unknown
      /** @example kong:trace:1234567890 */
      instance?: unknown
      /** @example Forbidden */
      detail?: unknown
    }
    NotFoundError: components['schemas']['BaseError'] & {
      /** @example 404 */
      status?: unknown
      /** @example Not Found */
      title?: unknown
      /** @example https://httpstatuses.com/404 */
      type?: unknown
      /** @example kong:trace:1234567890 */
      instance?: unknown
      /** @example Not found */
      detail?: unknown
    }
    GoneError: components['schemas']['BaseError'] & {
      /** @example 410 */
      status?: unknown
      /** @example Gone */
      title?: unknown
      /** @example https://httpstatuses.com/410 */
      type?: unknown
      /** @example kong:trace:1234567890 */
      instance?: unknown
      /** @example Gone */
      detail?: unknown
    }
    CursorMetaPage: {
      /**
       * Format: path
       * @description URI to the first page
       */
      first?: string
      /**
       * Format: path
       * @description URI to the last page
       */
      last?: string
      /**
       * Format: path
       * @description URI to the next page
       */
      next: string | null
      /**
       * Format: path
       * @description URI to the previous page
       */
      previous: string | null
      /**
       * @description Requested page size
       * @example 10
       */
      size: number
    }
    /** @description Pagination metadata. */
    CursorMeta: {
      page: components['schemas']['CursorMetaPage']
    }
    ConflictError: components['schemas']['BaseError'] & {
      /** @example 409 */
      status?: unknown
      /** @example Conflict */
      title?: unknown
      /** @example https://httpstatuses.com/409 */
      type?: unknown
      /** @example kong:trace:1234567890 */
      instance?: unknown
      /** @example Conflict */
      detail?: unknown
    }
  }
  responses: {
    /** @description Bad Request */
    BadRequest: {
      headers: {
        [name: string]: unknown
      }
      content: {
        'application/problem+json': components['schemas']['BadRequestError']
      }
    }
    /** @description Unauthorized */
    Unauthorized: {
      headers: {
        [name: string]: unknown
      }
      content: {
        'application/problem+json': components['schemas']['UnauthorizedError']
      }
    }
    /** @description Forbidden */
    Forbidden: {
      headers: {
        [name: string]: unknown
      }
      content: {
        'application/problem+json': components['schemas']['ForbiddenError']
      }
    }
    /** @description Not Found */
    NotFound: {
      headers: {
        [name: string]: unknown
      }
      content: {
        'application/problem+json': components['schemas']['NotFoundError']
      }
    }
    /** @description Gone */
    Gone: {
      headers: {
        [name: string]: unknown
      }
      content: {
        'application/problem+json': components['schemas']['GoneError']
      }
    }
    /** @description Conflict */
    Conflict: {
      headers: {
        [name: string]: unknown
      }
      content: {
        'application/problem+json': components['schemas']['ConflictError']
      }
    }
  }
  parameters: {
    CursorPaginationQuery: components['schemas']['CursorPaginationQueryPage']
    /** @description Determines which page of the collection to retrieve. */
    PagePaginationQuery: {
      /** @description The number of items to include per page. */
      size?: number
      /** @description The page number. */
      number?: number
    }
  }
  requestBodies: never
  headers: never
  pathItems: never
}
export type Addon = components['schemas']['Addon']
export type AddonInstanceType = components['schemas']['AddonInstanceType']
export type AddonPagePaginatedResponse =
  components['schemas']['AddonPagePaginatedResponse']
export type AddonReference = components['schemas']['AddonReference']
export type AddonReferenceItem = components['schemas']['AddonReferenceItem']
export type AddonStatus = components['schemas']['AddonStatus']
export type Address = components['schemas']['Address']
export type AppPagePaginatedResponse =
  components['schemas']['AppPagePaginatedResponse']
export type BillingAddress = components['schemas']['BillingAddress']
export type BillingApp = components['schemas']['BillingApp']
export type BillingAppCatalogItem =
  components['schemas']['BillingAppCatalogItem']
export type BillingAppCustomerData =
  components['schemas']['BillingAppCustomerData']
export type BillingAppCustomerDataExternalInvoicing =
  components['schemas']['BillingAppCustomerDataExternalInvoicing']
export type BillingAppCustomerDataStripe =
  components['schemas']['BillingAppCustomerDataStripe']
export type BillingAppExternalInvoicing =
  components['schemas']['BillingAppExternalInvoicing']
export type BillingAppReference = components['schemas']['BillingAppReference']
export type BillingAppSandbox = components['schemas']['BillingAppSandbox']
export type BillingAppStatus = components['schemas']['BillingAppStatus']
export type BillingAppStripe = components['schemas']['BillingAppStripe']
export type BillingAppStripeCheckoutSessionCustomTextParams =
  components['schemas']['BillingAppStripeCheckoutSessionCustomTextParams']
export type BillingAppStripeCheckoutSessionMode =
  components['schemas']['BillingAppStripeCheckoutSessionMode']
export type BillingAppStripeCheckoutSessionUiMode =
  components['schemas']['BillingAppStripeCheckoutSessionUIMode']
export type BillingAppStripeCreateCheckoutSessionBillingAddressCollection =
  components['schemas']['BillingAppStripeCreateCheckoutSessionBillingAddressCollection']
export type BillingAppStripeCreateCheckoutSessionConsentCollection =
  components['schemas']['BillingAppStripeCreateCheckoutSessionConsentCollection']
export type BillingAppStripeCreateCheckoutSessionConsentCollectionPaymentMethodReuseAgreement =
  components['schemas']['BillingAppStripeCreateCheckoutSessionConsentCollectionPaymentMethodReuseAgreement']
export type BillingAppStripeCreateCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPosition =
  components['schemas']['BillingAppStripeCreateCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPosition']
export type BillingAppStripeCreateCheckoutSessionConsentCollectionPromotions =
  components['schemas']['BillingAppStripeCreateCheckoutSessionConsentCollectionPromotions']
export type BillingAppStripeCreateCheckoutSessionConsentCollectionTermsOfService =
  components['schemas']['BillingAppStripeCreateCheckoutSessionConsentCollectionTermsOfService']
export type BillingAppStripeCreateCheckoutSessionCustomerUpdate =
  components['schemas']['BillingAppStripeCreateCheckoutSessionCustomerUpdate']
export type BillingAppStripeCreateCheckoutSessionCustomerUpdateBehavior =
  components['schemas']['BillingAppStripeCreateCheckoutSessionCustomerUpdateBehavior']
export type BillingAppStripeCreateCheckoutSessionRedirectOnCompletion =
  components['schemas']['BillingAppStripeCreateCheckoutSessionRedirectOnCompletion']
export type BillingAppStripeCreateCheckoutSessionRequestOptions =
  components['schemas']['BillingAppStripeCreateCheckoutSessionRequestOptions']
export type BillingAppStripeCreateCheckoutSessionResult =
  components['schemas']['BillingAppStripeCreateCheckoutSessionResult']
export type BillingAppStripeCreateCheckoutSessionTaxIdCollection =
  components['schemas']['BillingAppStripeCreateCheckoutSessionTaxIdCollection']
export type BillingAppStripeCreateCheckoutSessionTaxIdCollectionRequired =
  components['schemas']['BillingAppStripeCreateCheckoutSessionTaxIdCollectionRequired']
export type BillingAppStripeCreateCustomerPortalSessionOptions =
  components['schemas']['BillingAppStripeCreateCustomerPortalSessionOptions']
export type BillingAppStripeCreateCustomerPortalSessionResult =
  components['schemas']['BillingAppStripeCreateCustomerPortalSessionResult']
export type BillingAppType = components['schemas']['BillingAppType']
export type BillingCharge = components['schemas']['BillingCharge']
export type BillingChargeStatus = components['schemas']['BillingChargeStatus']
export type BillingChargeTotals = components['schemas']['BillingChargeTotals']
export type BillingChargesExpand = components['schemas']['BillingChargesExpand']
export type BillingCostBasis = components['schemas']['BillingCostBasis']
export type BillingCreditAdjustment =
  components['schemas']['BillingCreditAdjustment']
export type BillingCreditAvailabilityPolicy =
  components['schemas']['BillingCreditAvailabilityPolicy']
export type BillingCreditBalances =
  components['schemas']['BillingCreditBalances']
export type BillingCreditFundingMethod =
  components['schemas']['BillingCreditFundingMethod']
export type BillingCreditGrant = components['schemas']['BillingCreditGrant']
export type BillingCreditGrantFilters =
  components['schemas']['BillingCreditGrantFilters']
export type BillingCreditGrantInvoiceReference =
  components['schemas']['BillingCreditGrantInvoiceReference']
export type BillingCreditGrantPurchase =
  components['schemas']['BillingCreditGrantPurchase']
export type BillingCreditGrantStatus =
  components['schemas']['BillingCreditGrantStatus']
export type BillingCreditGrantTaxConfig =
  components['schemas']['BillingCreditGrantTaxConfig']
export type BillingCreditPurchasePaymentSettlementStatus =
  components['schemas']['BillingCreditPurchasePaymentSettlementStatus']
export type BillingCreditTransaction =
  components['schemas']['BillingCreditTransaction']
export type BillingCreditTransactionType =
  components['schemas']['BillingCreditTransactionType']
export type BillingCurrency = components['schemas']['BillingCurrency']
export type BillingCurrencyCode = components['schemas']['BillingCurrencyCode']
export type BillingCurrencyCodeCustom =
  components['schemas']['BillingCurrencyCodeCustom']
export type BillingCurrencyCustom =
  components['schemas']['BillingCurrencyCustom']
export type BillingCurrencyFiat = components['schemas']['BillingCurrencyFiat']
export type BillingCurrencyType = components['schemas']['BillingCurrencyType']
export type BillingCustomer = components['schemas']['BillingCustomer']
export type BillingCustomerData = components['schemas']['BillingCustomerData']
export type BillingCustomerReference =
  components['schemas']['BillingCustomerReference']
export type BillingCustomerStripeCreateCheckoutSessionRequest =
  components['schemas']['BillingCustomerStripeCreateCheckoutSessionRequest']
export type BillingCustomerStripeCreateCustomerPortalSessionRequest =
  components['schemas']['BillingCustomerStripeCreateCustomerPortalSessionRequest']
export type BillingCustomerUsageAttribution =
  components['schemas']['BillingCustomerUsageAttribution']
export type BillingEntitlementAccessResult =
  components['schemas']['BillingEntitlementAccessResult']
export type BillingEntitlementType =
  components['schemas']['BillingEntitlementType']
export type BillingFeatureLlmTokenType =
  components['schemas']['BillingFeatureLLMTokenType']
export type BillingFeatureLlmUnitCost =
  components['schemas']['BillingFeatureLLMUnitCost']
export type BillingFeatureLlmUnitCostPricing =
  components['schemas']['BillingFeatureLLMUnitCostPricing']
export type BillingFeatureManualUnitCost =
  components['schemas']['BillingFeatureManualUnitCost']
export type BillingFeatureUnitCost =
  components['schemas']['BillingFeatureUnitCost']
export type BillingFlatFeeCharge = components['schemas']['BillingFlatFeeCharge']
export type BillingFlatFeeDiscounts =
  components['schemas']['BillingFlatFeeDiscounts']
export type BillingParty = components['schemas']['BillingParty']
export type BillingPartyAddresses =
  components['schemas']['BillingPartyAddresses']
export type BillingPartyTaxIdentity =
  components['schemas']['BillingPartyTaxIdentity']
export type BillingPlan = components['schemas']['BillingPlan']
export type BillingPlanPhase = components['schemas']['BillingPlanPhase']
export type BillingPlanStatus = components['schemas']['BillingPlanStatus']
export type BillingPrice = components['schemas']['BillingPrice']
export type BillingPriceFlat = components['schemas']['BillingPriceFlat']
export type BillingPriceFree = components['schemas']['BillingPriceFree']
export type BillingPriceGraduated =
  components['schemas']['BillingPriceGraduated']
export type BillingPricePaymentTerm =
  components['schemas']['BillingPricePaymentTerm']
export type BillingPriceTier = components['schemas']['BillingPriceTier']
export type BillingPriceUnit = components['schemas']['BillingPriceUnit']
export type BillingPriceVolume = components['schemas']['BillingPriceVolume']
export type BillingProfile = components['schemas']['BillingProfile']
export type BillingProfileAppReferences =
  components['schemas']['BillingProfileAppReferences']
export type BillingProfilePagePaginatedResponse =
  components['schemas']['BillingProfilePagePaginatedResponse']
export type BillingProfileReference =
  components['schemas']['BillingProfileReference']
export type BillingRateCard = components['schemas']['BillingRateCard']
export type BillingRateCardDiscounts =
  components['schemas']['BillingRateCardDiscounts']
export type BillingRateCardProrationConfiguration =
  components['schemas']['BillingRateCardProrationConfiguration']
export type BillingRateCardProrationMode =
  components['schemas']['BillingRateCardProrationMode']
export type BillingRateCardTaxConfig =
  components['schemas']['BillingRateCardTaxConfig']
export type BillingSettlementMode =
  components['schemas']['BillingSettlementMode']
export type BillingSpendCommitments =
  components['schemas']['BillingSpendCommitments']
export type BillingSubscription = components['schemas']['BillingSubscription']
export type BillingSubscriptionCancel =
  components['schemas']['BillingSubscriptionCancel']
export type BillingSubscriptionChange =
  components['schemas']['BillingSubscriptionChange']
export type BillingSubscriptionChangeResponse =
  components['schemas']['BillingSubscriptionChangeResponse']
export type BillingSubscriptionCreate =
  components['schemas']['BillingSubscriptionCreate']
export type BillingSubscriptionEditTiming =
  components['schemas']['BillingSubscriptionEditTiming']
export type BillingSubscriptionEditTimingEnum =
  components['schemas']['BillingSubscriptionEditTimingEnum']
export type BillingSubscriptionReference =
  components['schemas']['BillingSubscriptionReference']
export type BillingSubscriptionStatus =
  components['schemas']['BillingSubscriptionStatus']
export type BillingTaxBehavior = components['schemas']['BillingTaxBehavior']
export type BillingTaxCode = components['schemas']['BillingTaxCode']
export type BillingTaxCodeAppMapping =
  components['schemas']['BillingTaxCodeAppMapping']
export type BillingTaxConfig = components['schemas']['BillingTaxConfig']
export type BillingTaxConfigExternalInvoicing =
  components['schemas']['BillingTaxConfigExternalInvoicing']
export type BillingTaxConfigStripe =
  components['schemas']['BillingTaxConfigStripe']
export type BillingTaxIdentificationCode =
  components['schemas']['BillingTaxIdentificationCode']
export type BillingTotals = components['schemas']['BillingTotals']
export type BillingUsageBasedCharge =
  components['schemas']['BillingUsageBasedCharge']
export type BillingWorkflow = components['schemas']['BillingWorkflow']
export type BillingWorkflowCollectionAlignment =
  components['schemas']['BillingWorkflowCollectionAlignment']
export type BillingWorkflowCollectionAlignmentAnchored =
  components['schemas']['BillingWorkflowCollectionAlignmentAnchored']
export type BillingWorkflowCollectionAlignmentSubscription =
  components['schemas']['BillingWorkflowCollectionAlignmentSubscription']
export type BillingWorkflowCollectionSettings =
  components['schemas']['BillingWorkflowCollectionSettings']
export type BillingWorkflowInvoicingSettings =
  components['schemas']['BillingWorkflowInvoicingSettings']
export type BillingWorkflowPaymentChargeAutomaticallySettings =
  components['schemas']['BillingWorkflowPaymentChargeAutomaticallySettings']
export type BillingWorkflowPaymentSendInvoiceSettings =
  components['schemas']['BillingWorkflowPaymentSendInvoiceSettings']
export type BillingWorkflowPaymentSettings =
  components['schemas']['BillingWorkflowPaymentSettings']
export type BillingWorkflowTaxSettings =
  components['schemas']['BillingWorkflowTaxSettings']
export type ChargePagePaginatedResponse =
  components['schemas']['ChargePagePaginatedResponse']
export type ClosedPeriod = components['schemas']['ClosedPeriod']
export type CostBasisPagePaginatedResponse =
  components['schemas']['CostBasisPagePaginatedResponse']
export type CountryCode = components['schemas']['CountryCode']
export type CreateAddonRequest = components['schemas']['CreateAddonRequest']
export type CreateBillingProfileRequest =
  components['schemas']['CreateBillingProfileRequest']
export type CreateCostBasisRequest =
  components['schemas']['CreateCostBasisRequest']
export type CreateCreditAdjustmentRequest =
  components['schemas']['CreateCreditAdjustmentRequest']
export type CreateCreditGrantFilters =
  components['schemas']['CreateCreditGrantFilters']
export type CreateCreditGrantPurchase =
  components['schemas']['CreateCreditGrantPurchase']
export type CreateCreditGrantRequest =
  components['schemas']['CreateCreditGrantRequest']
export type CreateCreditGrantTaxConfig =
  components['schemas']['CreateCreditGrantTaxConfig']
export type CreateCurrencyCode = components['schemas']['CreateCurrencyCode']
export type CreateCurrencyCustomRequest =
  components['schemas']['CreateCurrencyCustomRequest']
export type CreateCustomerRequest =
  components['schemas']['CreateCustomerRequest']
export type CreateFeatureRequest = components['schemas']['CreateFeatureRequest']
export type CreateMeterRequest = components['schemas']['CreateMeterRequest']
export type CreatePlanAddonRequest =
  components['schemas']['CreatePlanAddonRequest']
export type CreatePlanRequest = components['schemas']['CreatePlanRequest']
export type CreateResourceReference =
  components['schemas']['CreateResourceReference']
export type CreateTaxCodeRequest = components['schemas']['CreateTaxCodeRequest']
export type CreditBalance = components['schemas']['CreditBalance']
export type CreditGrantPagePaginatedResponse =
  components['schemas']['CreditGrantPagePaginatedResponse']
export type CreditTransactionPaginatedResponse =
  components['schemas']['CreditTransactionPaginatedResponse']
export type CurrencyAmount = components['schemas']['CurrencyAmount']
export type CurrencyCode = components['schemas']['CurrencyCode']
export type CurrencyPagePaginatedResponse =
  components['schemas']['CurrencyPagePaginatedResponse']
export type CursorPaginationQueryPage =
  components['schemas']['CursorPaginationQueryPage']
export type CustomerPagePaginatedResponse =
  components['schemas']['CustomerPagePaginatedResponse']
export type CustomerReference = components['schemas']['CustomerReference']
export type DateTime = components['schemas']['DateTime']
export type DateTimeFieldFilter = components['schemas']['DateTimeFieldFilter']
export type ExternalResourceKey = components['schemas']['ExternalResourceKey']
export type Feature = components['schemas']['Feature']
export type FeatureCostQueryResult =
  components['schemas']['FeatureCostQueryResult']
export type FeatureCostQueryRow = components['schemas']['FeatureCostQueryRow']
export type FeatureMeterReference =
  components['schemas']['FeatureMeterReference']
export type FeaturePagePaginatedResponse =
  components['schemas']['FeaturePagePaginatedResponse']
export type FeatureReferenceItem = components['schemas']['FeatureReferenceItem']
export type GetCreditBalanceParamsFilter =
  components['schemas']['GetCreditBalanceParamsFilter']
export type GovernanceFeatureAccess =
  components['schemas']['GovernanceFeatureAccess']
export type GovernanceFeatureAccessReason =
  components['schemas']['GovernanceFeatureAccessReason']
export type GovernanceFeatureAccessReasonCode =
  components['schemas']['GovernanceFeatureAccessReasonCode']
export type GovernanceQueryError = components['schemas']['GovernanceQueryError']
export type GovernanceQueryErrorCode =
  components['schemas']['GovernanceQueryErrorCode']
export type GovernanceQueryRequest =
  components['schemas']['GovernanceQueryRequest']
export type GovernanceQueryRequestCustomers =
  components['schemas']['GovernanceQueryRequestCustomers']
export type GovernanceQueryRequestFeatures =
  components['schemas']['GovernanceQueryRequestFeatures']
export type GovernanceQueryResponse =
  components['schemas']['GovernanceQueryResponse']
export type GovernanceQueryResult =
  components['schemas']['GovernanceQueryResult']
export type Iso8601Duration = components['schemas']['ISO8601Duration']
export type IngestedEventPaginatedResponse =
  components['schemas']['IngestedEventPaginatedResponse']
export type LlmCostModel = components['schemas']['LLMCostModel']
export type LlmCostModelPricing = components['schemas']['LLMCostModelPricing']
export type LlmCostOverrideCreate =
  components['schemas']['LLMCostOverrideCreate']
export type LlmCostPrice = components['schemas']['LLMCostPrice']
export type LlmCostPriceSource = components['schemas']['LLMCostPriceSource']
export type LlmCostProvider = components['schemas']['LLMCostProvider']
export type ListAddonsParamsFilter =
  components['schemas']['ListAddonsParamsFilter']
export type ListChargesParamsFilter =
  components['schemas']['ListChargesParamsFilter']
export type ListCostBasesParamsFilter =
  components['schemas']['ListCostBasesParamsFilter']
export type ListCreditGrantsParamsFilter =
  components['schemas']['ListCreditGrantsParamsFilter']
export type ListCreditTransactionsParamsFilter =
  components['schemas']['ListCreditTransactionsParamsFilter']
export type ListCurrenciesParamsFilter =
  components['schemas']['ListCurrenciesParamsFilter']
export type ListCustomerEntitlementAccessResponseData =
  components['schemas']['ListCustomerEntitlementAccessResponseData']
export type ListCustomersParamsFilter =
  components['schemas']['ListCustomersParamsFilter']
export type ListEventsParamsFilter =
  components['schemas']['ListEventsParamsFilter']
export type ListFeatureParamsFilter =
  components['schemas']['ListFeatureParamsFilter']
export type ListLlmCostPricesParamsFilter =
  components['schemas']['ListLLMCostPricesParamsFilter']
export type ListMetersParamsFilter =
  components['schemas']['ListMetersParamsFilter']
export type ListPlansParamsFilter =
  components['schemas']['ListPlansParamsFilter']
export type ListSubscriptionsParamsFilter =
  components['schemas']['ListSubscriptionsParamsFilter']
export type Meter = components['schemas']['Meter']
export type MeterAggregation = components['schemas']['MeterAggregation']
export type MeterPagePaginatedResponse =
  components['schemas']['MeterPagePaginatedResponse']
export type MeterQueryFilters = components['schemas']['MeterQueryFilters']
export type MeterQueryGranularity =
  components['schemas']['MeterQueryGranularity']
export type MeterQueryRequest = components['schemas']['MeterQueryRequest']
export type MeterQueryResult = components['schemas']['MeterQueryResult']
export type MeterQueryRow = components['schemas']['MeterQueryRow']
export type MeteringEvent = components['schemas']['MeteringEvent']
export type MeteringIngestedEvent =
  components['schemas']['MeteringIngestedEvent']
export type MeteringIngestedEventValidationError =
  components['schemas']['MeteringIngestedEventValidationError']
export type Numeric = components['schemas']['Numeric']
export type OrganizationDefaultTaxCodes =
  components['schemas']['OrganizationDefaultTaxCodes']
export type PlanAddon = components['schemas']['PlanAddon']
export type PlanAddonPagePaginatedResponse =
  components['schemas']['PlanAddonPagePaginatedResponse']
export type PlanPagePaginatedResponse =
  components['schemas']['PlanPagePaginatedResponse']
export type PricePagePaginatedResponse =
  components['schemas']['PricePagePaginatedResponse']
export type ProductCatalogValidationError =
  components['schemas']['ProductCatalogValidationError']
export type QueryFilterString = components['schemas']['QueryFilterString']
export type QueryFilterStringMapItem =
  components['schemas']['QueryFilterStringMapItem']
export type RecurringPeriod = components['schemas']['RecurringPeriod']
export type ResourceKey = components['schemas']['ResourceKey']
export type ResourceManagedBy = components['schemas']['ResourceManagedBy']
export type StringFieldFilter = components['schemas']['StringFieldFilter']
export type StringFieldFilterExact =
  components['schemas']['StringFieldFilterExact']
export type SubscriptionAddon = components['schemas']['SubscriptionAddon']
export type SubscriptionAddonPagePaginatedResponse =
  components['schemas']['SubscriptionAddonPagePaginatedResponse']
export type SubscriptionPagePaginatedResponse =
  components['schemas']['SubscriptionPagePaginatedResponse']
export type TaxCodePagePaginatedResponse =
  components['schemas']['TaxCodePagePaginatedResponse']
export type TaxCodeReference = components['schemas']['TaxCodeReference']
export type TaxCodeReferenceItem = components['schemas']['TaxCodeReferenceItem']
export type Ulid = components['schemas']['ULID']
export type UlidFieldFilter = components['schemas']['ULIDFieldFilter']
export type UpdateCreditGrantExternalSettlementRequest =
  components['schemas']['UpdateCreditGrantExternalSettlementRequest']
export type UpdateFeatureRequest = components['schemas']['UpdateFeatureRequest']
export type UpdateMeterRequest = components['schemas']['UpdateMeterRequest']
export type UpdateOrganizationDefaultTaxCodesRequest =
  components['schemas']['UpdateOrganizationDefaultTaxCodesRequest']
export type UpsertAddonRequest = components['schemas']['UpsertAddonRequest']
export type UpsertAppCustomerDataRequest =
  components['schemas']['UpsertAppCustomerDataRequest']
export type UpsertBillingProfileRequest =
  components['schemas']['UpsertBillingProfileRequest']
export type UpsertCustomerBillingDataRequest =
  components['schemas']['UpsertCustomerBillingDataRequest']
export type UpsertCustomerRequest =
  components['schemas']['UpsertCustomerRequest']
export type UpsertPlanAddonRequest =
  components['schemas']['UpsertPlanAddonRequest']
export type UpsertPlanRequest = components['schemas']['UpsertPlanRequest']
export type UpsertTaxCodeRequest = components['schemas']['UpsertTaxCodeRequest']
export type UsageAttributionSubjectKey =
  components['schemas']['UsageAttributionSubjectKey']
export type SortQuery = components['schemas']['SortQuery']
export type Labels = components['schemas']['Labels']
export type PageMeta = components['schemas']['PageMeta']
export type PaginatedMeta = components['schemas']['PaginatedMeta']
export type BaseError = components['schemas']['BaseError']
export type InvalidRules = components['schemas']['InvalidRules']
export type InvalidParameterStandard =
  components['schemas']['InvalidParameterStandard']
export type InvalidParameterMinimumLength =
  components['schemas']['InvalidParameterMinimumLength']
export type InvalidParameterMaximumLength =
  components['schemas']['InvalidParameterMaximumLength']
export type InvalidParameterChoiceItem =
  components['schemas']['InvalidParameterChoiceItem']
export type InvalidParameterDependentItem =
  components['schemas']['InvalidParameterDependentItem']
export type InvalidParameters = components['schemas']['InvalidParameters']
export type BadRequestError = components['schemas']['BadRequestError']
export type UnauthorizedError = components['schemas']['UnauthorizedError']
export type ForbiddenError = components['schemas']['ForbiddenError']
export type NotFoundError = components['schemas']['NotFoundError']
export type GoneError = components['schemas']['GoneError']
export type CursorMetaPage = components['schemas']['CursorMetaPage']
export type CursorMeta = components['schemas']['CursorMeta']
export type ConflictError = components['schemas']['ConflictError']
export type ResponseBadRequest = components['responses']['BadRequest']
export type ResponseUnauthorized = components['responses']['Unauthorized']
export type ResponseForbidden = components['responses']['Forbidden']
export type ResponseNotFound = components['responses']['NotFound']
export type ResponseGone = components['responses']['Gone']
export type ResponseConflict = components['responses']['Conflict']
export type ParameterCursorPaginationQuery =
  components['parameters']['CursorPaginationQuery']
export type ParameterPagePaginationQuery =
  components['parameters']['PagePaginationQuery']
export type $defs = Record<string, never>
export interface operations {
  'list-addons': {
    parameters: {
      query?: {
        /** @description Determines which page of the collection to retrieve. */
        page?: components['parameters']['PagePaginationQuery']
        /**
         * @description Sort add-ons returned in the response. Supported sort attributes are:
         *
         *     - `id`
         *     - `key`
         *     - `name`
         *     - `created_at` (default)
         *     - `updated_at`
         *
         *     The `asc` suffix is optional as the default sort order is ascending. The `desc`
         *     suffix is used to specify a descending order.
         */
        sort?: components['schemas']['SortQuery']
        /** @description Filter add-ons returned in the response. */
        filter?: components['schemas']['ListAddonsParamsFilter']
      }
      header?: never
      path?: never
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description Page paginated response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['AddonPagePaginatedResponse']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
    }
  }
  'create-addon': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['CreateAddonRequest']
      }
    }
    responses: {
      /** @description Addon created response. */
      201: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Addon']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
    }
  }
  'get-addon': {
    parameters: {
      query?: never
      header?: never
      path: {
        addonId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description Addon response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Addon']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
      410: components['responses']['Gone']
    }
  }
  'update-addon': {
    parameters: {
      query?: never
      header?: never
      path: {
        addonId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['UpsertAddonRequest']
      }
    }
    responses: {
      /** @description Addon upsert response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Addon']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
      410: components['responses']['Gone']
    }
  }
  'delete-addon': {
    parameters: {
      query?: never
      header?: never
      path: {
        addonId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description Deleted response. */
      204: {
        headers: {
          [name: string]: unknown
        }
        content?: never
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'archive-addon': {
    parameters: {
      query?: never
      header?: never
      path: {
        addonId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description Addon updated response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Addon']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'publish-addon': {
    parameters: {
      query?: never
      header?: never
      path: {
        addonId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description Addon updated response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Addon']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'list-apps': {
    parameters: {
      query?: {
        /** @description Determines which page of the collection to retrieve. */
        page?: components['parameters']['PagePaginationQuery']
      }
      header?: never
      path?: never
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description Page paginated response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['AppPagePaginatedResponse']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
    }
  }
  'get-app': {
    parameters: {
      query?: never
      header?: never
      path: {
        appId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description App response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['BillingApp']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'list-currencies': {
    parameters: {
      query?: {
        /** @description Determines which page of the collection to retrieve. */
        page?: components['parameters']['PagePaginationQuery']
        /**
         * @description Sort currencies returned in the response. Supported sort attributes are:
         *
         *     - `code` (default)
         *     - `name`
         *
         *     The `asc` suffix is optional as the default sort order is ascending. The `desc`
         *     suffix is used to specify a descending order.
         */
        sort?: components['schemas']['SortQuery']
        /**
         * @description Filter currencies returned in the response.
         *
         *     To filter currencies by type add the following query param: filter[type]=custom
         */
        filter?: components['schemas']['ListCurrenciesParamsFilter']
      }
      header?: never
      path?: never
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description Page paginated response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['CurrencyPagePaginatedResponse']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
    }
  }
  'create-custom-currency': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['CreateCurrencyCustomRequest']
      }
    }
    responses: {
      /** @description CurrencyCustom created response. */
      201: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['BillingCurrencyCustom']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
    }
  }
  'list-cost-bases': {
    parameters: {
      query?: {
        /**
         * @description Filter cost bases returned in the response.
         *
         *     To filter cost bases by fiat currency code add the following query param:
         *     filter[fiat_code]=USD
         */
        filter?: components['schemas']['ListCostBasesParamsFilter']
        /** @description Determines which page of the collection to retrieve. */
        page?: components['parameters']['PagePaginationQuery']
      }
      header?: never
      path: {
        currencyId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description Page paginated response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['CostBasisPagePaginatedResponse']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'create-cost-basis': {
    parameters: {
      query?: never
      header?: never
      path: {
        currencyId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['CreateCostBasisRequest']
      }
    }
    responses: {
      /** @description CostBasis created response. */
      201: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['BillingCostBasis']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
    }
  }
  'list-customers': {
    parameters: {
      query?: {
        /** @description Determines which page of the collection to retrieve. */
        page?: components['parameters']['PagePaginationQuery']
        /**
         * @description Sort customers returned in the response. Supported sort attributes are:
         *
         *     - `id`
         *     - `name` (default)
         *     - `created_at`
         *
         *     The `asc` suffix is optional as the default sort order is ascending. The `desc`
         *     suffix is used to specify a descending order.
         */
        sort?: components['schemas']['SortQuery']
        /**
         * @description Filter customers returned in the response.
         *
         *     To filter customers by key add the following query param: filter[key]=my-db-id
         */
        filter?: components['schemas']['ListCustomersParamsFilter']
      }
      header?: never
      path?: never
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description Page paginated response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['CustomerPagePaginatedResponse']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
    }
  }
  'create-customer': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['CreateCustomerRequest']
      }
    }
    responses: {
      /** @description Customer created response. */
      201: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['BillingCustomer']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
    }
  }
  'get-customer': {
    parameters: {
      query?: never
      header?: never
      path: {
        customerId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description Customer response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['BillingCustomer']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'upsert-customer': {
    parameters: {
      query?: never
      header?: never
      path: {
        customerId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['UpsertCustomerRequest']
      }
    }
    responses: {
      /** @description Customer upsert response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['BillingCustomer']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
      410: components['responses']['Gone']
    }
  }
  'delete-customer': {
    parameters: {
      query?: never
      header?: never
      path: {
        customerId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description Deleted response. */
      204: {
        headers: {
          [name: string]: unknown
        }
        content?: never
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'get-customer-billing': {
    parameters: {
      query?: never
      header?: never
      path: {
        customerId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description CustomerBillingData response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['BillingCustomerData']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'update-customer-billing': {
    parameters: {
      query?: never
      header?: never
      path: {
        customerId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['UpsertCustomerBillingDataRequest']
      }
    }
    responses: {
      /** @description CustomerBillingData upsert response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['BillingCustomerData']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
      410: components['responses']['Gone']
    }
  }
  'update-customer-billing-app-data': {
    parameters: {
      query?: never
      header?: never
      path: {
        customerId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['UpsertAppCustomerDataRequest']
      }
    }
    responses: {
      /** @description AppCustomerData upsert response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['BillingAppCustomerData']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
      410: components['responses']['Gone']
    }
  }
  'create-customer-stripe-checkout-session': {
    parameters: {
      query?: never
      header?: never
      path: {
        customerId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['BillingCustomerStripeCreateCheckoutSessionRequest']
      }
    }
    responses: {
      /** @description CreateStripeCheckoutSessionResult created response. */
      201: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['BillingAppStripeCreateCheckoutSessionResult']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
      410: components['responses']['Gone']
    }
  }
  'create-customer-stripe-portal-session': {
    parameters: {
      query?: never
      header?: never
      path: {
        customerId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['BillingCustomerStripeCreateCustomerPortalSessionRequest']
      }
    }
    responses: {
      /** @description CreateStripeCustomerPortalSessionResult created response. */
      201: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['BillingAppStripeCreateCustomerPortalSessionResult']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
      410: components['responses']['Gone']
    }
  }
  'list-customer-charges': {
    parameters: {
      query?: {
        /** @description Determines which page of the collection to retrieve. */
        page?: components['parameters']['PagePaginationQuery']
        /**
         * @description Sort charges returned in the response.
         *
         *     Supported sort attributes are:
         *
         *     - `id`
         *     - `created_at`
         *     - `service_period.from`
         *     - `billing_period.from`
         */
        sort?: components['schemas']['SortQuery']
        /**
         * @description Filter charges.
         *
         *     To filter charges by status add the following query param:
         *     `filter[status][oeq]=created,active`
         */
        filter?: components['schemas']['ListChargesParamsFilter']
        /**
         * @description Expand full objects for referenced entities.
         *
         *     Supported values are:
         *
         *     - `real_time_usage`: Expand the charge's real-time usage.
         */
        expand?: components['schemas']['BillingChargesExpand'][]
      }
      header?: never
      path: {
        customerId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description Page paginated response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['ChargePagePaginatedResponse']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'create-credit-adjustment': {
    parameters: {
      query?: never
      header?: never
      path: {
        customerId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['CreateCreditAdjustmentRequest']
      }
    }
    responses: {
      /** @description CreditAdjustment created response. */
      201: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['BillingCreditAdjustment']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'get-customer-credit-balance': {
    parameters: {
      query?: {
        /**
         * @description Return the credit balance as of this timestamp.
         *
         *     Defaults to the current time.
         */
        timestamp?: components['schemas']['DateTime']
        filter?: components['schemas']['GetCreditBalanceParamsFilter']
      }
      header?: never
      path: {
        customerId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description CreditBalances response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['BillingCreditBalances']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'list-credit-grants': {
    parameters: {
      query?: {
        /** @description Determines which page of the collection to retrieve. */
        page?: components['parameters']['PagePaginationQuery']
        /** @description Filter credit grants returned in the response. */
        filter?: components['schemas']['ListCreditGrantsParamsFilter']
      }
      header?: never
      path: {
        customerId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description Page paginated response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['CreditGrantPagePaginatedResponse']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'create-credit-grant': {
    parameters: {
      query?: never
      header?: never
      path: {
        customerId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['CreateCreditGrantRequest']
      }
    }
    responses: {
      /** @description CreditGrant created response. */
      201: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['BillingCreditGrant']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'get-credit-grant': {
    parameters: {
      query?: never
      header?: never
      path: {
        customerId: components['schemas']['ULID']
        creditGrantId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description CreditGrant response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['BillingCreditGrant']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'update-credit-grant-external-settlement': {
    parameters: {
      query?: never
      header?: never
      path: {
        customerId: components['schemas']['ULID']
        creditGrantId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['UpdateCreditGrantExternalSettlementRequest']
      }
    }
    responses: {
      /** @description CreditGrant updated response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['BillingCreditGrant']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'list-credit-transactions': {
    parameters: {
      query?: {
        page?: components['parameters']['CursorPaginationQuery']
        /** @description Filter credit transactions returned in the response. */
        filter?: components['schemas']['ListCreditTransactionsParamsFilter']
      }
      header?: never
      path: {
        customerId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description Cursor paginated response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['CreditTransactionPaginatedResponse']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'list-customer-entitlement-access': {
    parameters: {
      query?: never
      header?: never
      path: {
        customerId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description List the customer's active features and their access. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['ListCustomerEntitlementAccessResponseData']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'get-organization-default-tax-codes': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description OrganizationDefaultTaxCodes response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['OrganizationDefaultTaxCodes']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'update-organization-default-tax-codes': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['UpdateOrganizationDefaultTaxCodesRequest']
      }
    }
    responses: {
      /** @description OrganizationDefaultTaxCodes upsert response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['OrganizationDefaultTaxCodes']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'list-metering-events': {
    parameters: {
      query?: {
        page?: components['parameters']['CursorPaginationQuery']
        /**
         * @description Filter events returned in the response.
         *
         *     To filter events by subject add the following query param:
         *     filter[subject][eq]=customer-1
         */
        filter?: components['schemas']['ListEventsParamsFilter']
        /**
         * @description Sort events returned in the response. Supported sort attributes are:
         *
         *     - `time` (default)
         *     - `ingested_at`
         *     - `stored_at`
         *
         *     When omitted, events are sorted by `time desc` (most recent first). When a sort
         *     field is provided without a suffix, it sorts descending. Append the `asc` suffix
         *     to sort ascending, or the `desc` suffix to sort descending.
         */
        sort?: components['schemas']['SortQuery']
      }
      header?: never
      path?: never
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description Cursor paginated response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['IngestedEventPaginatedResponse']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
    }
  }
  'ingest-metering-events': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    requestBody: {
      content: {
        'application/cloudevents+json': components['schemas']['MeteringEvent']
        'application/cloudevents-batch+json': components['schemas']['MeteringEvent'][]
        'application/json':
          | components['schemas']['MeteringEvent']
          | components['schemas']['MeteringEvent'][]
      }
    }
    responses: {
      /** @description The events have been ingested and are being processed asynchronously. */
      202: {
        headers: {
          [name: string]: unknown
        }
        content?: never
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
    }
  }
  'list-features': {
    parameters: {
      query?: {
        /** @description Determines which page of the collection to retrieve. */
        page?: components['parameters']['PagePaginationQuery']
        /**
         * @description Sort features returned in the response. Supported sort attributes are:
         *
         *     - `key`
         *     - `name`
         *     - `created_at` (default)
         *     - `updated_at`
         *
         *     The `asc` suffix is optional as the default sort order is ascending. The `desc`
         *     suffix is used to specify a descending order.
         */
        sort?: components['schemas']['SortQuery']
        /**
         * @description Filter features returned in the response.
         *
         *     To filter features by meter_id add the following query param:
         *     filter[meter_id][oeq]=<id>
         */
        filter?: components['schemas']['ListFeatureParamsFilter']
      }
      header?: never
      path?: never
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description Page paginated response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['FeaturePagePaginatedResponse']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
    }
  }
  'create-feature': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['CreateFeatureRequest']
      }
    }
    responses: {
      /** @description Feature created response. */
      201: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Feature']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
    }
  }
  'get-feature': {
    parameters: {
      query?: never
      header?: never
      path: {
        featureId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description Feature response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Feature']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
      410: components['responses']['Gone']
    }
  }
  'delete-feature': {
    parameters: {
      query?: never
      header?: never
      path: {
        featureId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description Deleted response. */
      204: {
        headers: {
          [name: string]: unknown
        }
        content?: never
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'update-feature': {
    parameters: {
      query?: never
      header?: never
      path: {
        featureId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['UpdateFeatureRequest']
      }
    }
    responses: {
      /** @description Feature updated response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Feature']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'query-feature-cost': {
    parameters: {
      query?: never
      header?: never
      path: {
        featureId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody?: {
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
          'application/json': components['schemas']['FeatureCostQueryResult']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'query-governance-access': {
    parameters: {
      query?: {
        page?: components['parameters']['CursorPaginationQuery']
      }
      header?: never
      path?: never
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['GovernanceQueryRequest']
      }
    }
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['GovernanceQueryResponse']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
    }
  }
  'list-llm-cost-overrides': {
    parameters: {
      query?: {
        filter?: components['schemas']['ListLLMCostPricesParamsFilter']
        /** @description Determines which page of the collection to retrieve. */
        page?: components['parameters']['PagePaginationQuery']
      }
      header?: never
      path?: never
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description Page paginated response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['PricePagePaginatedResponse']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
    }
  }
  'create-llm-cost-override': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['LLMCostOverrideCreate']
      }
    }
    responses: {
      /** @description Price created response. */
      201: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['LLMCostPrice']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
    }
  }
  'delete-llm-cost-override': {
    parameters: {
      query?: never
      header?: never
      path: {
        priceId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description Deleted response. */
      204: {
        headers: {
          [name: string]: unknown
        }
        content?: never
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'list-llm-cost-prices': {
    parameters: {
      query?: {
        /** @description Filter prices. */
        filter?: components['schemas']['ListLLMCostPricesParamsFilter']
        /**
         * @description Sort prices returned in the response. Supported sort attributes are:
         *
         *     - `id`
         *     - `provider.id`
         *     - `model.id` (default)
         *     - `effective_from`
         *     - `effective_to`
         *
         *     The `asc` suffix is optional as the default sort order is ascending. The `desc`
         *     suffix is used to specify a descending order.
         */
        sort?: components['schemas']['SortQuery']
        /** @description Determines which page of the collection to retrieve. */
        page?: components['parameters']['PagePaginationQuery']
      }
      header?: never
      path?: never
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description Page paginated response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['PricePagePaginatedResponse']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
    }
  }
  'get-llm-cost-price': {
    parameters: {
      query?: never
      header?: never
      path: {
        priceId: components['schemas']['ULID']
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
          'application/json': components['schemas']['LLMCostPrice']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'list-meters': {
    parameters: {
      query?: {
        /** @description Determines which page of the collection to retrieve. */
        page?: components['parameters']['PagePaginationQuery']
        /**
         * @description Sort meters returned in the response. Supported sort attributes are:
         *
         *     - `key`
         *     - `name`
         *     - `aggregation`
         *     - `createdAt` (default)
         *     - `updatedAt`
         *
         *     The `asc` suffix is optional as the default sort order is ascending. The `desc`
         *     suffix is used to specify a descending order.
         */
        sort?: components['schemas']['SortQuery']
        /**
         * @description Filter meters returned in the response.
         *
         *     To filter meters by key add the following query param: filter[key]=my-meter-key
         */
        filter?: components['schemas']['ListMetersParamsFilter']
      }
      header?: never
      path?: never
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description Page paginated response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['MeterPagePaginatedResponse']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
    }
  }
  'create-meter': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['CreateMeterRequest']
      }
    }
    responses: {
      /** @description Meter created response. */
      201: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Meter']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
    }
  }
  'get-meter': {
    parameters: {
      query?: never
      header?: never
      path: {
        meterId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description Meter response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Meter']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'update-meter': {
    parameters: {
      query?: never
      header?: never
      path: {
        meterId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['UpdateMeterRequest']
      }
    }
    responses: {
      /** @description Meter updated response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['Meter']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'delete-meter': {
    parameters: {
      query?: never
      header?: never
      path: {
        meterId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description Deleted response. */
      204: {
        headers: {
          [name: string]: unknown
        }
        content?: never
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'query-meter': {
    parameters: {
      query?: never
      header?: never
      path: {
        meterId: components['schemas']['ULID']
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
          'text/csv': string
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'list-plans': {
    parameters: {
      query?: {
        /** @description Determines which page of the collection to retrieve. */
        page?: components['parameters']['PagePaginationQuery']
        /**
         * @description Sort plans returned in the response. Supported sort attributes are:
         *
         *     - `id`
         *     - `key`
         *     - `version`
         *     - `created_at` (default)
         *     - `updated_at`
         */
        sort?: components['schemas']['SortQuery']
        /** @description Filter plans returned in the response. */
        filter?: components['schemas']['ListPlansParamsFilter']
      }
      header?: never
      path?: never
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description Page paginated response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['PlanPagePaginatedResponse']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
    }
  }
  'create-plan': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['CreatePlanRequest']
      }
    }
    responses: {
      /** @description Plan created response. */
      201: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['BillingPlan']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
    }
  }
  'get-plan': {
    parameters: {
      query?: never
      header?: never
      path: {
        planId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description Plan response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['BillingPlan']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
      410: components['responses']['Gone']
    }
  }
  'update-plan': {
    parameters: {
      query?: never
      header?: never
      path: {
        planId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['UpsertPlanRequest']
      }
    }
    responses: {
      /** @description Plan upsert response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['BillingPlan']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
      410: components['responses']['Gone']
    }
  }
  'delete-plan': {
    parameters: {
      query?: never
      header?: never
      path: {
        planId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description Deleted response. */
      204: {
        headers: {
          [name: string]: unknown
        }
        content?: never
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'list-plan-addons': {
    parameters: {
      query?: {
        /** @description Determines which page of the collection to retrieve. */
        page?: components['parameters']['PagePaginationQuery']
      }
      header?: never
      path: {
        planId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description Page paginated response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['PlanAddonPagePaginatedResponse']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'create-plan-addon': {
    parameters: {
      query?: never
      header?: never
      path: {
        planId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['CreatePlanAddonRequest']
      }
    }
    responses: {
      /** @description PlanAddon created response. */
      201: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['PlanAddon']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'get-plan-addon': {
    parameters: {
      query?: never
      header?: never
      path: {
        planId: components['schemas']['ULID']
        planAddonId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description PlanAddon response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['PlanAddon']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'update-plan-addon': {
    parameters: {
      query?: never
      header?: never
      path: {
        planId: components['schemas']['ULID']
        planAddonId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['UpsertPlanAddonRequest']
      }
    }
    responses: {
      /** @description PlanAddon upsert response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['PlanAddon']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'delete-plan-addon': {
    parameters: {
      query?: never
      header?: never
      path: {
        planId: components['schemas']['ULID']
        planAddonId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description Deleted response. */
      204: {
        headers: {
          [name: string]: unknown
        }
        content?: never
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'archive-plan': {
    parameters: {
      query?: never
      header?: never
      path: {
        planId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description Plan updated response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['BillingPlan']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'publish-plan': {
    parameters: {
      query?: never
      header?: never
      path: {
        planId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description Plan updated response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['BillingPlan']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'list-billing-profiles': {
    parameters: {
      query?: {
        /** @description Determines which page of the collection to retrieve. */
        page?: components['parameters']['PagePaginationQuery']
      }
      header?: never
      path?: never
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description Page paginated response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['BillingProfilePagePaginatedResponse']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
    }
  }
  'create-billing-profile': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['CreateBillingProfileRequest']
      }
    }
    responses: {
      /** @description BillingProfile created response. */
      201: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['BillingProfile']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
    }
  }
  'get-billing-profile': {
    parameters: {
      query?: never
      header?: never
      path: {
        id: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description BillingProfile response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['BillingProfile']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'update-billing-profile': {
    parameters: {
      query?: never
      header?: never
      path: {
        id: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['UpsertBillingProfileRequest']
      }
    }
    responses: {
      /** @description BillingProfile updated response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['BillingProfile']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'delete-billing-profile': {
    parameters: {
      query?: never
      header?: never
      path: {
        id: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description Deleted response. */
      204: {
        headers: {
          [name: string]: unknown
        }
        content?: never
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'list-subscriptions': {
    parameters: {
      query?: {
        /** @description Determines which page of the collection to retrieve. */
        page?: components['parameters']['PagePaginationQuery']
        /**
         * @description Sort subscriptions returned in the response. Supported sort attributes are:
         *
         *     - `id`
         *     - `active_from` (default)
         *     - `active_to`
         *
         *     The `asc` suffix is optional as the default sort order is ascending. The `desc`
         *     suffix is used to specify a descending order.
         */
        sort?: components['schemas']['SortQuery']
        /** @description Filter subscriptions. */
        filter?: components['schemas']['ListSubscriptionsParamsFilter']
      }
      header?: never
      path?: never
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description Page paginated response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['SubscriptionPagePaginatedResponse']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'create-subscription': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['BillingSubscriptionCreate']
      }
    }
    responses: {
      /** @description Subscription created response. */
      201: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['BillingSubscription']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
      409: components['responses']['Conflict']
    }
  }
  'get-subscription': {
    parameters: {
      query?: never
      header?: never
      path: {
        subscriptionId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description Subscription response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['BillingSubscription']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'list-subscription-addons': {
    parameters: {
      query?: {
        /** @description Determines which page of the collection to retrieve. */
        page?: components['parameters']['PagePaginationQuery']
        sort?: components['schemas']['SortQuery']
      }
      header?: never
      path: {
        subscriptionId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description Page paginated response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['SubscriptionAddonPagePaginatedResponse']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'get-subscription-addon': {
    parameters: {
      query?: never
      header?: never
      path: {
        subscriptionId: components['schemas']['ULID']
        subscriptionAddonId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description SubscriptionAddon response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['SubscriptionAddon']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'cancel-subscription': {
    parameters: {
      query?: never
      header?: never
      path: {
        subscriptionId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['BillingSubscriptionCancel']
      }
    }
    responses: {
      /** @description Subscription updated response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['BillingSubscription']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
      409: components['responses']['Conflict']
    }
  }
  'change-subscription': {
    parameters: {
      query?: never
      header?: never
      path: {
        subscriptionId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['BillingSubscriptionChange']
      }
    }
    responses: {
      /** @description The request has succeeded. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['BillingSubscriptionChangeResponse']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
      409: components['responses']['Conflict']
    }
  }
  'unschedule-cancelation': {
    parameters: {
      query?: never
      header?: never
      path: {
        subscriptionId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description Subscription updated response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['BillingSubscription']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
      409: components['responses']['Conflict']
    }
  }
  'list-tax-codes': {
    parameters: {
      query?: {
        /** @description Determines which page of the collection to retrieve. */
        page?: components['parameters']['PagePaginationQuery']
        /** @description Include deleted tax codes in the response. */
        include_deleted?: boolean
      }
      header?: never
      path?: never
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description Page paginated response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['TaxCodePagePaginatedResponse']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
    }
  }
  'create-tax-code': {
    parameters: {
      query?: never
      header?: never
      path?: never
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['CreateTaxCodeRequest']
      }
    }
    responses: {
      /** @description TaxCode created response. */
      201: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['BillingTaxCode']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
    }
  }
  'get-tax-code': {
    parameters: {
      query?: never
      header?: never
      path: {
        taxCodeId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description TaxCode response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['BillingTaxCode']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
  'upsert-tax-code': {
    parameters: {
      query?: never
      header?: never
      path: {
        taxCodeId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody: {
      content: {
        'application/json': components['schemas']['UpsertTaxCodeRequest']
      }
    }
    responses: {
      /** @description TaxCode upsert response. */
      200: {
        headers: {
          [name: string]: unknown
        }
        content: {
          'application/json': components['schemas']['BillingTaxCode']
        }
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
      410: components['responses']['Gone']
    }
  }
  'delete-tax-code': {
    parameters: {
      query?: never
      header?: never
      path: {
        taxCodeId: components['schemas']['ULID']
      }
      cookie?: never
    }
    requestBody?: never
    responses: {
      /** @description Deleted response. */
      204: {
        headers: {
          [name: string]: unknown
        }
        content?: never
      }
      400: components['responses']['BadRequest']
      401: components['responses']['Unauthorized']
      403: components['responses']['Forbidden']
      404: components['responses']['NotFound']
    }
  }
}
