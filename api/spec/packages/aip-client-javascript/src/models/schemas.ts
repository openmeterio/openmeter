import { z } from 'zod'

export const labels = z
  .record(z.string(), z.string())

  .describe(
    'Labels store metadata of an entity that can be used for filtering an entity list or for searching across entity types. Keys must be of length 1-63 characters, and cannot start with "kong", "konnect", "mesh", "kic", or "\\_".',
  )

export const currencyCode = z
  .string()
  .min(3)
  .max(3)
  .regex(new RegExp('^[A-Z]{3}$'))

  .describe(
    'Three-letter [ISO4217](https://www.iso.org/iso-4217-currency-codes.html) currency code. Custom three-letter currency codes are also supported for convenience.',
  )

export const numeric = z
  .string()
  .regex(new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$'))
  .describe('Numeric represents an arbitrary precision number.')

export const cursorPaginationQueryPage = z
  .object({
    size: z
      .number()
      .int()
      .optional()
      .describe('The number of items to include per page.'),
    after: z
      .string()
      .optional()

      .describe(
        'Request the next page of data, starting with the item after this parameter.',
      ),
    before: z
      .string()
      .optional()

      .describe(
        'Request the previous page of data, starting with the item before this parameter.',
      ),
  })
  .describe('Determines which page of the collection to retrieve.')

export const stringFieldFilter = z
  .union([
    z.string(),
    z.object({
      eq: z
        .string()
        .optional()
        .describe('Value strictly equals the given string value.'),
      neq: z
        .string()
        .optional()
        .describe('Value does not equal the given string value.'),
      contains: z
        .string()
        .optional()
        .describe('Value contains the given string value (fuzzy match).'),
      ocontains: z
        .array(z.string())
        .optional()

        .describe(
          'Returns entities that fuzzy-match any of the comma-delimited phrases in the filter string.',
        ),
      oeq: z
        .array(z.string())
        .optional()

        .describe(
          'Returns entities that exact match any of the comma-delimited phrases in the filter string.',
        ),
      gt: z
        .string()
        .optional()

        .describe(
          'Value is greater than the given string value (lexicographic compare).',
        ),
      gte: z
        .string()
        .optional()

        .describe(
          'Value is greater than or equal to the given string value (lexicographic compare).',
        ),
      lt: z
        .string()
        .optional()

        .describe(
          'Value is less than the given string value (lexicographic compare).',
        ),
      lte: z
        .string()
        .optional()

        .describe(
          'Value is less than or equal to the given string value (lexicographic compare).',
        ),
      exists: z
        .boolean()
        .optional()

        .describe(
          'When true, the field must be present (non-null); when false, the field must be absent (null).',
        ),
    }),
  ])

  .describe(
    'Filters on the given string field value by either exact or fuzzy match. All properties are optional; provide exactly one to specify the comparison.',
  )

export const ulid = z
  .string()
  .regex(new RegExp('^[0-7][0-9A-HJKMNP-TV-Z]{25}$'))
  .describe('ULID (Universally Unique Lexicographically Sortable Identifier).')

export const dateTime = z
  .string()
  .datetime()

  .describe(
    '[RFC3339](https://tools.ietf.org/html/rfc3339) formatted date-time string in UTC.',
  )

export const sortQuery = z
  .object({
    by: z.string().describe('The attribute to sort by.'),
    order: z
      .union([z.literal('asc'), z.literal('desc')])
      .optional()
      .default('asc')
      .describe('The sort order. `asc` for ascending, `desc` for descending.'),
  })

  .describe(
    'Sort query. The `asc` suffix is optional as the default sort order is ascending. The `desc` suffix is used to specify a descending order.',
  )

export const ingestedEventValidationError = z
  .object({
    code: z.string().describe('The machine readable code of the error.'),
    message: z
      .string()
      .describe('The human readable description of the error.'),
    attributes: z
      .record(z.string(), z.unknown())
      .optional()
      .describe('Additional attributes.'),
  })
  .describe('Event validation errors.')

export const cursorMetaPage = z
  .object({
    first: z.string().optional().describe('URI to the first page.'),
    last: z.string().optional().describe('URI to the last page.'),
    next: z.string().optional().describe('URI to the next page.'),
    previous: z.string().optional().describe('URI to the previous page.'),
    size: z.number().int().optional().describe('Requested page size.'),
  })
  .describe('Cursor pagination metadata.')

export const invalidRules = z
  .enum([
    'required',
    'is_array',
    'is_base64',
    'is_boolean',
    'is_date_time',
    'is_integer',
    'is_null',
    'is_number',
    'is_object',
    'is_string',
    'is_uuid',
    'is_fqdn',
    'is_arn',
    'unknown_property',
    'missing_reference',
    'is_label',
    'matches_regex',
    'invalid',
    'is_supported_network_availability_zone_list',
    'is_supported_network_cidr_block',
    'is_supported_provider_region',
    'type',
  ])
  .describe('The validation rule that a parameter failed.')

export const invalidParameterMinimumRule = z
  .enum([
    'min_length',
    'min_digits',
    'min_lowercase',
    'min_uppercase',
    'min_symbols',
    'min_items',
    'min',
  ])
  .describe('Minimum-length (or minimum-value) validation rules.')

export const invalidParameterMaximumRule = z
  .enum(['max_length', 'max_items', 'max'])
  .describe('Maximum-length (or maximum-value) validation rules.')

export const invalidParameterChoiceRule = z
  .enum(['enum'])
  .describe('The enum validation rule.')

export const invalidParameterDependentRule = z
  .enum(['dependent_fields'])
  .describe('The dependent-fields validation rule.')

export const baseError = z
  .intersection(
    z.object({
      type: z
        .string()
        .default('about:blank')
        .describe('Type contains a URI that identifies the problem type.'),
      status: z
        .number()
        .int()

        .describe(
          'The HTTP status code generated by the origin server for this occurrence of the problem.',
        ),
      title: z
        .string()
        .describe('A a short, human-readable summary of the problem type.'),
      detail: z
        .string()

        .describe(
          'A human-readable explanation specific to this occurrence of the problem.',
        ),
      instance: z
        .string()

        .describe(
          'A URI reference that identifies the specific occurrence of the problem.',
        ),
    }),
    z.record(z.string(), z.unknown()),
  )
  .describe('Standard error response.')

export const resourceKey = z
  .string()
  .min(1)
  .max(64)
  .regex(new RegExp('^[a-z0-9]+(?:_[a-z0-9]+)*$'))
  .describe('A key is a unique string that is used to identify a resource.')

export const meterAggregation = z
  .union([
    z.literal('sum'),
    z.literal('count'),
    z.literal('unique_count'),
    z.literal('avg'),
    z.literal('min'),
    z.literal('max'),
    z.literal('latest'),
  ])
  .describe('The aggregation type to use for the meter.')

export const pageMeta = z
  .object({
    number: z.number().int().describe('Page number.'),
    size: z.number().int().describe('Page size.'),
    total: z
      .number()
      .int()
      .describe('Total number of items in the collection.'),
  })
  .describe('Pagination information.')

export const meterQueryGranularity = z
  .union([
    z.literal('PT1M'),
    z.literal('PT1H'),
    z.literal('P1D'),
    z.literal('P1M'),
  ])

  .describe(
    'The granularity of the time grouping. Time durations are specified in ISO 8601 format.',
  )

export const queryFilterString = z
  .object({
    eq: z
      .string()
      .optional()
      .describe('The attribute equals the provided value.'),
    neq: z
      .string()
      .optional()
      .describe('The attribute does not equal the provided value.'),
    in: z
      .array(z.string())
      .min(1)
      .max(100)
      .optional()
      .describe('The attribute is one of the provided values.'),
    nin: z
      .array(z.string())
      .min(1)
      .max(100)
      .optional()
      .describe('The attribute is not one of the provided values.'),
    contains: z
      .string()
      .optional()
      .describe('The attribute contains the provided value.'),
    ncontains: z
      .string()
      .optional()
      .describe('The attribute does not contain the provided value.'),
    get and() {
      return z
        .array(queryFilterString)
        .min(1)
        .max(10)
        .optional()
        .describe('Combines the provided filters with a logical AND.')
    },
    get or() {
      return z
        .array(queryFilterString)
        .min(1)
        .max(10)
        .optional()
        .describe('Combines the provided filters with a logical OR.')
    },
  })

  .describe(
    'A query filter for a string attribute. Operators are mutually exclusive, only one operator is allowed at a time.',
  )

export const externalResourceKey = z
  .string()
  .min(1)
  .max(256)

  .describe(
    'ExternalResourceKey is a unique string that is used to identify a resource in an external system.',
  )

export const usageAttributionSubjectKey = z
  .string()
  .min(1)
  .describe('Subject key.')

export const countryCode = z
  .string()
  .min(2)
  .max(2)
  .regex(new RegExp('^[A-Z]{2}$'))

  .describe(
    '[ISO 3166-1](https://www.iso.org/iso-3166-country-codes.html) alpha-2 country code. Custom two-letter country codes are also supported for convenience.',
  )

export const appStripeCreateCheckoutSessionBillingAddressCollection = z
  .enum(['auto', 'required'])

  .describe(
    "Controls whether Checkout collects the customer's billing address.",
  )

export const appStripeCreateCheckoutSessionCustomerUpdateBehavior = z
  .enum(['auto', 'never'])
  .describe('Behavior for updating customer fields from checkout session.')

export const appStripeCreateCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPosition =
  z
    .enum(['auto', 'hidden'])
    .describe('Position of payment method reuse agreement in the UI.')

export const appStripeCreateCheckoutSessionConsentCollectionPromotions = z
  .enum(['auto', 'none'])
  .describe('Promotional communication consent collection setting.')

export const appStripeCreateCheckoutSessionConsentCollectionTermsOfService = z
  .enum(['none', 'required'])
  .describe('Terms of service acceptance requirement.')

export const appStripeCheckoutSessionCustomTextParams = z
  .object({
    after_submit: z
      .object({
        message: z
          .string()
          .max(1200)
          .optional()
          .describe('The custom message text (max 1200 characters).'),
      })
      .optional()
      .describe('Text displayed after the payment confirmation button.'),
    shipping_address: z
      .object({
        message: z
          .string()
          .max(1200)
          .optional()
          .describe('The custom message text (max 1200 characters).'),
      })
      .optional()
      .describe('Text displayed alongside shipping address collection.'),
    submit: z
      .object({
        message: z
          .string()
          .max(1200)
          .optional()
          .describe('The custom message text (max 1200 characters).'),
      })
      .optional()
      .describe('Text displayed alongside the payment confirmation button.'),
    terms_of_service_acceptance: z
      .object({
        message: z
          .string()
          .max(1200)
          .optional()
          .describe('The custom message text (max 1200 characters).'),
      })
      .optional()
      .describe('Text replacing the default terms of service agreement text.'),
  })
  .describe('Custom text displayed at various stages of the checkout flow.')

export const appStripeCheckoutSessionUiMode = z
  .enum(['embedded', 'hosted'])
  .describe('Checkout Session UI mode.')

export const appStripeCreateCheckoutSessionRedirectOnCompletion = z
  .enum(['always', 'if_required', 'never'])
  .describe('Redirect behavior for embedded checkout sessions.')

export const appStripeCreateCheckoutSessionTaxIdCollectionRequired = z
  .enum(['if_supported', 'never'])
  .describe('Tax ID collection requirement level.')

export const appStripeCheckoutSessionMode = z
  .enum(['setup'])

  .describe(
    'Stripe Checkout Session mode. Determines the primary purpose of the checkout session.',
  )

export const appStripeCreateCustomerPortalSessionOptions = z
  .object({
    configuration_id: z
      .string()
      .optional()

      .describe(
        'The ID of an existing [Stripe configuration](https://docs.stripe.com/api/customer_portal/configurations) to use for this session, describing its functionality and features. If not specified, the session uses the default configuration.',
      ),
    locale: z
      .string()
      .optional()

      .describe(
        "The IETF [language tag](https://docs.stripe.com/api/customer_portal/sessions/create#create_portal_session-locale) of the locale customer portal is displayed in. If blank or `auto`, the customer's preferred_locales or browser's locale is used.",
      ),
    return_url: z
      .string()
      .optional()

      .describe(
        'The [URL to redirect](https://docs.stripe.com/api/customer_portal/sessions/create#create_portal_session-return_url) the customer to after they have completed their requested actions.',
      ),
  })
  .describe('Request to create a Stripe Customer Portal Session.')

export const entitlementType = z
  .enum(['metered', 'static', 'boolean'])
  .describe('The type of the entitlement.')

export const createLabels = z
  .record(z.string(), z.string())

  .describe(
    'Labels store metadata of an entity that can be used for filtering an entity list or for searching across entity types. Keys must be of length 1-63 characters, and cannot start with "kong", "konnect", "mesh", "kic", or "\\_".',
  )

export const creditFundingMethod = z
  .enum(['none', 'invoice', 'external'])

  .describe(
    'The funding method describes how the grant is funded. - `none`: No funding workflow applies, for example promotional grants - `invoice`: The grant is funded by an in-system invoice flow - `external`: The grant is funded outside the system (e.g., wire transfer, external invoice, or manual reconciliation)',
  )

export const creditAvailabilityPolicy = z
  .enum(['on_creation'])

  .describe(
    'When credits become available for consumption. - `on_creation`: Credits are available as soon as the grant is created. - `on_authorization`: Credits are available once the payment is authorized. - `on_settlement`: Credits are available once the payment is settled.',
  )

export const taxBehavior = z
  .enum(['inclusive', 'exclusive'])

  .describe(
    'Tax behavior. This enum is used to specify whether tax is included in the price or excluded from the price.',
  )

export const iso8601Duration = z
  .string()

  .regex(
    new RegExp(
      '^P(?:\\d+(?:\\.\\d+)?Y)?(?:\\d+(?:\\.\\d+)?M)?(?:\\d+(?:\\.\\d+)?W)?(?:\\d+(?:\\.\\d+)?D)?(?:T(?:\\d+(?:\\.\\d+)?H)?(?:\\d+(?:\\.\\d+)?M)?(?:\\d+(?:\\.\\d+)?S)?)?$',
    ),
  )

  .describe(
    '[ISO 8601 Duration](https://docs.digi.com/resources/documentation/digidocs/90001488-13/reference/r_iso_8601_duration_format.htm) string.',
  )

export const creditPurchasePaymentSettlementStatus = z
  .enum(['pending', 'authorized', 'settled'])

  .describe(
    'Credit purchase payment settlement status. - `pending`: Payment has been initiated and is not yet authorized. - `authorized`: Payment has been authorized. - `settled`: Payment has been settled.',
  )

export const creditGrantStatus = z
  .enum(['pending', 'active', 'expired', 'voided'])

  .describe(
    'Credit grant lifecycle status. - `pending`: The credit block has been created but is not yet valid. (`effective_at` is in the future or availability_policy is not met) - `active`: The credit block is currently valid and eligible for consumption. (`effective_at` is in the past, `expires_at` is in the future and availability_policy is met) - `expired`: The credit block expired with remaining unused balance, `expires_at` time has passed. - `voided`: The credit block was voided. Remaining balance is forfeited.',
  )

export const stringFieldFilterExact = z
  .union([
    z.string(),
    z.object({
      eq: z
        .string()
        .optional()
        .describe('Value strictly equals the given string value.'),
      oeq: z
        .array(z.string())
        .optional()

        .describe(
          'Returns entities that exact match any of the comma-delimited phrases in the filter string.',
        ),
      neq: z
        .string()
        .optional()
        .describe('Value does not equal the given string value.'),
    }),
  ])

  .describe(
    'Filters on the given string field value by exact match. All properties are optional; provide exactly one to specify the comparison.',
  )

export const creditTransactionType = z
  .enum(['funded', 'consumed', 'expired'])

  .describe(
    'The type of the credit transaction. - `funded`: Credit granted and available for consumption. - `consumed`: Credit consumed by usage or fees. - `expired`: Credit removed because it expired before being used.',
  )

export const chargesExpand = z
  .enum(['real_time_usage'])

  .describe(
    "Expands for customer charges. Values: - `real_time_usage`: The charge's real-time usage.",
  )

export const resourceManagedBy = z
  .enum(['manual', 'system', 'subscription'])

  .describe(
    'Identifies which system manages a resource. Values: - `manual`: The resource is managed manually (overridden by our API users). - `system`: The resource is managed by the system. - `subscription`: The resource is managed by the subscription.',
  )

export const chargeStatus = z
  .enum(['created', 'active', 'final', 'deleted'])

  .describe(
    'Lifecycle status of a charge. Values: - `created`: The charge has been created but is not active yet. - `active`: The charge is active. - `final`: The charge is fully finalized and no further changes are expected. - `deleted`: The charge has been deleted.',
  )

export const settlementMode = z
  .enum(['credit_then_invoice', 'credit_only'])

  .describe(
    'Settlement mode for billing. Values: - `credit_then_invoice`: Credits are applied first, then any remainder is invoiced. - `credit_only`: Usage is settled exclusively against credits.',
  )

export const taxConfigStripe = z
  .object({
    code: z
      .string()
      .regex(new RegExp('^txcd_\\d{8}$'))
      .describe('Product [tax code](https://docs.stripe.com/tax/tax-codes).'),
  })
  .describe('The tax config for Stripe.')

export const taxConfigExternalInvoicing = z
  .object({
    code: z
      .string()
      .max(64)

      .describe(
        'The tax code should be interpreted by the external invoicing provider.',
      ),
  })
  .describe('External invoicing tax config.')

export const pricePaymentTerm = z
  .union([z.literal('in_advance'), z.literal('in_arrears')])
  .describe('The payment term of a flat price.')

export const flatFeeDiscounts = z
  .object({
    percentage: z
      .number()
      .nonnegative()
      .lte(100)
      .optional()
      .describe('Percentage discount applied to the price (0–100).'),
  })

  .describe(
    'Discounts applicable to flat fee charges. This is the same as `ProductCatalog.Discounts` but without the `usage` field, which is not applicable to flat fee charges.',
  )

export const rateCardProrationMode = z
  .enum(['no_proration', 'prorate_prices'])

  .describe(
    'The proration mode of the rate card. Values: - `no_proration`: No proration. - `prorate_prices`: Prorate the price based on the time remaining in the billing period.',
  )

export const priceFree = z
  .object({
    type: z.literal('free').describe('The type of the price.'),
  })
  .describe('Free price.')

export const subscriptionStatus = z
  .enum(['active', 'inactive', 'canceled', 'scheduled'])
  .describe('Subscription status.')

export const subscriptionEditTimingEnum = z
  .enum(['immediate', 'next_billing_cycle'])

  .describe(
    'Subscription edit timing. When immediate, the requested changes take effect immediately. When next_billing_cycle, the requested changes take effect at the next billing cycle.',
  )

export const appType = z
  .enum(['sandbox', 'stripe', 'external_invoicing'])
  .describe('The type of the app.')

export const appStatus = z
  .enum(['ready', 'unauthorized'])
  .describe('Connection status of an installed app.')

export const taxIdentificationCode = z
  .string()
  .min(1)
  .max(32)

  .describe(
    'Tax identifier code is a normalized tax code shown on the original identity document.',
  )

export const workflowCollectionAlignmentSubscription = z
  .object({
    type: z.literal('subscription').describe('The type of alignment.'),
  })

  .describe(
    'BillingWorkflowCollectionAlignmentSubscription specifies the alignment for collecting the pending line items into an invoice.',
  )

export const workflowInvoicingSubscriptionEndProrationMode = z
  .enum(['bill_full_period', 'bill_actual_period'])
  .describe('Billing workflow subscription end proration mode.')

export const workflowPaymentChargeAutomaticallySettings = z
  .object({
    collection_method: z
      .literal('charge_automatically')
      .describe('The collection method for the invoice.'),
  })

  .describe(
    'Payment settings for a billing workflow when the collection method is charge automatically.',
  )

export const workflowPaymentSendInvoiceSettings = z
  .object({
    collection_method: z
      .literal('send_invoice')
      .describe('The collection method for the invoice.'),
    due_after: z
      .string()
      .optional()
      .default('P30D')

      .describe(
        "The period after which the invoice is due. With some payment solutions it's only applicable for manual collection method.",
      ),
  })

  .describe(
    'Payment settings for a billing workflow when the collection method is send invoice.',
  )

export const currencyType = z
  .enum(['fiat', 'custom'])

  .describe(
    'Currency type for custom currencies. It should be a unique code but not conflicting with any existing standard currency codes.',
  )

export const currencyCodeCustom = z
  .string()
  .min(3)
  .max(24)

  .describe(
    'Custom currency code. It should be a unique code but not conflicting with any existing fiat currency codes.',
  )

export const featureLlmTokenType = z
  .enum([
    'input',
    'output',
    'cache_read',
    'cache_write',
    'reasoning',
    'request',
    'response',
  ])
  .describe('Token type for LLM cost lookup.')

export const llmCostProvider = z
  .object({
    id: z
      .string()
      .describe('Identifier of the provider, e.g., "openai", "anthropic".'),
    name: z
      .string()
      .describe('Name of the provider, e.g., "OpenAI", "Anthropic".'),
  })
  .describe('LLM Provider')

export const llmCostModel = z
  .object({
    id: z
      .string()

      .describe('Identifier of the model, e.g., "gpt-4", "claude-3-5-sonnet".'),
    name: z
      .string()
      .describe('Name of the model, e.g., "GPT-4", "Claude 3.5 Sonnet".'),
  })
  .describe('LLM Model')

export const llmCostPriceSource = z
  .enum(['manual', 'system'])
  .describe('Identifies where an LLM cost price came from.')

export const planStatus = z
  .enum(['draft', 'active', 'archived', 'scheduled'])

  .describe(
    'The status of a plan. - `draft`: The plan has not yet been published and can be edited. - `active`: The plan is published and can be used in subscriptions. - `archived`: The plan is no longer available for use. - `scheduled`: The plan is scheduled to be published at a future date.',
  )

export const unitConfigOperation = z
  .enum(['divide', 'multiply'])

  .describe(
    'The arithmetic operation used to convert raw metered units into billing units. - `divide`: Divide the metered quantity by the conversion factor (e.g., bytes ÷ 1e9 = GB). - `multiply`: Multiply the metered quantity by the conversion factor (e.g., cost × 1.2 = cost + 20% margin).',
  )

export const unitConfigRoundingMode = z
  .enum(['ceiling', 'floor', 'half_up', 'none'])

  .describe(
    'The rounding mode applied to the converted quantity for invoicing. Rounding is applied only to the invoiced quantity. Entitlement balance checks use the precise decimal value after conversion. - `ceiling`: Round up to the next integer (typical for package-style billing). - `floor`: Round down to the previous integer. - `half_up`: Round to the nearest integer, with 0.5 rounding up. - `none`: No rounding; the converted value is used as-is.',
  )

export const rateCardStaticEntitlement = z
  .object({
    type: z.literal('static').describe('The type of the entitlement template.'),
    config: z
      .unknown()

      .describe(
        'The entitlement config as a JSON object. Returned when checking entitlement access; useful for configuring fine-grained access settings implemented in your own system.',
      ),
  })
  .describe('The entitlement template of a static entitlement.')

export const rateCardBooleanEntitlement = z
  .object({
    type: z
      .literal('boolean')
      .describe('The type of the entitlement template.'),
  })
  .describe('The entitlement template of a boolean entitlement.')

export const productCatalogValidationError = z
  .object({
    code: z.string().describe('Machine-readable error code.'),
    message: z.string().describe('Human-readable description of the error.'),
    attributes: z
      .record(z.string(), z.unknown())
      .optional()
      .describe('Additional structured context.'),
    field: z.string().describe('The path to the field.'),
  })
  .describe('Validation errors providing detailed description of the issue.')

export const addonInstanceType = z
  .enum(['single', 'multiple'])

  .describe(
    'The instanceType of the add-on. - `single`: Can be added to a subscription only once. - `multiple`: Can be added to a subscription more than once.',
  )

export const addonStatus = z
  .enum(['draft', 'active', 'archived'])

  .describe(
    'The status of the add-on defined by the `effective_from` and `effective_to` properties. - `draft`: The add-on has not yet been published and can be edited. - `active`: The add-on is published and available for use. - `archived`: The add-on is no longer available for use.',
  )

export const governanceQueryRequestCustomers = z
  .object({
    keys: z
      .array(z.string())
      .min(1)
      .max(100)

      .describe(
        'Each entry can be a customer `key` or a usage-attribution subject `key`. Identifiers that cannot be resolved to a customer are reported in the response `errors` array.',
      ),
  })
  .describe('List of customer identifiers to evaluate access for.')

export const governanceQueryRequestFeatures = z
  .object({
    keys: z
      .array(z.string())
      .min(1)
      .max(100)
      .describe('List of feature keys to evaluate access for.'),
  })

  .describe(
    'Optional list of feature keys to evaluate access for. If omitted, all features available in the organization are returned. Providing this list is recommended to reduce the response size and the load on the backend services.',
  )

export const governanceFeatureAccessReasonCode = z
  .enum([
    'unknown',
    'usage_limit_reached',
    'feature_unavailable',
    'feature_not_found',
    'no_credit_available',
  ])
  .describe('Machine-readable reason code for denied feature access.')

export const governanceQueryErrorCode = z
  .enum(['unknown', 'customer_not_found'])
  .describe('Error code for a governance query failure.')

export const queryFilterInteger = z
  .object({
    eq: z
      .number()
      .int()
      .optional()
      .describe('The attribute equals the provided value.'),
    neq: z
      .number()
      .int()
      .optional()
      .describe('The attribute does not equal the provided value.'),
    in: z
      .array(z.number().int())
      .min(1)
      .max(100)
      .optional()
      .describe('The attribute is one of the provided values.'),
    nin: z
      .array(z.number().int())
      .min(1)
      .max(100)
      .optional()
      .describe('The attribute is not one of the provided values.'),
    gt: z
      .number()
      .int()
      .optional()
      .describe('The attribute is greater than the provided value.'),
    gte: z
      .number()
      .int()
      .optional()

      .describe(
        'The attribute is greater than or equal to the provided value.',
      ),
    lt: z
      .number()
      .int()
      .optional()
      .describe('The attribute is less than the provided value.'),
    lte: z
      .number()
      .int()
      .optional()
      .describe('The attribute is less than or equal to the provided value.'),
    get and() {
      return z
        .array(queryFilterInteger)
        .min(1)
        .max(10)
        .optional()
        .describe('Combines the provided filters with a logical AND.')
    },
    get or() {
      return z
        .array(queryFilterInteger)
        .min(1)
        .max(10)
        .optional()
        .describe('Combines the provided filters with a logical OR.')
    },
  })

  .describe(
    'A query filter for an integer attribute. Operators are mutually exclusive, only one operator is allowed at a time.',
  )

export const queryFilterFloat = z
  .object({
    gt: z
      .number()
      .optional()
      .describe('The attribute is greater than the provided value.'),
    gte: z
      .number()
      .optional()

      .describe(
        'The attribute is greater than or equal to the provided value.',
      ),
    lt: z
      .number()
      .optional()
      .describe('The attribute is less than the provided value.'),
    lte: z
      .number()
      .optional()
      .describe('The attribute is less than or equal to the provided value.'),
    get and() {
      return z
        .array(queryFilterFloat)
        .min(1)
        .max(10)
        .optional()
        .describe('Combines the provided filters with a logical AND.')
    },
    get or() {
      return z
        .array(queryFilterFloat)
        .min(1)
        .max(10)
        .optional()
        .describe('Combines the provided filters with a logical OR.')
    },
  })

  .describe(
    'A query filter for a float attribute. Operators are mutually exclusive, only one operator is allowed at a time.',
  )

export const queryFilterBoolean = z
  .object({
    eq: z
      .boolean()
      .optional()
      .describe('The attribute equals the provided value.'),
  })

  .describe(
    'A query filter for a boolean attribute. Operators are mutually exclusive, only one operator is allowed at a time.',
  )

export const pagePaginationQuery = z
  .object({
    page: z
      .object({
        size: z
          .number()
          .int()
          .optional()
          .describe('The number of items to include per page.'),
        number: z.number().int().optional().describe('The page number.'),
      })
      .optional()
      .describe('Determines which page of the collection to retrieve.'),
  })
  .describe('Page pagination query.')

export const publicLabels = z
  .record(z.string(), z.string())

  .describe(
    'Public labels store information about an entity that can be used for filtering a list of objects.',
  )

export const booleanFieldFilter = z
  .union([
    z.boolean(),
    z.object({
      eq: z
        .boolean()
        .describe('Value strictly equals the given boolean value.'),
    }),
  ])
  .describe('Filter by a boolean value (true/false).')

export const numericFieldFilter = z
  .union([
    z.number(),
    z.object({
      eq: z
        .number()
        .optional()
        .describe('Value strictly equals the given numeric value.'),
      neq: z
        .number()
        .optional()
        .describe('Value does not equal the given numeric value.'),
      oeq: z
        .array(z.number())
        .optional()

        .describe(
          'Returns entities that match any of the comma-delimited numeric values.',
        ),
      lt: z
        .number()
        .optional()
        .describe('Value is less than the given numeric value.'),
      lte: z
        .number()
        .optional()
        .describe('Value is less than or equal to the given numeric value.'),
      gt: z
        .number()
        .optional()
        .describe('Value is greater than the given numeric value.'),
      gte: z
        .number()
        .optional()
        .describe('Value is greater than or equal to the given numeric value.'),
    }),
  ])

  .describe(
    'Filter by a numeric value. All properties are optional; provide exactly one to specify the comparison.',
  )

export const chargeType = z
  .enum(['flat_fee', 'usage_based'])

  .describe(
    'Type of a charge. Values: - `flat_fee`: A fixed-amount charge. - `usage_based`: A usage-priced charge.',
  )

export const invoiceNumber = z
  .string()
  .min(1)
  .max(256)

  .describe(
    'InvoiceNumber is a unique identifier for the invoice, generated by the invoicing app. The uniqueness depends on a lot of factors: - app setting (unique per app or unique per customer) - multiple app scenarios (multiple apps generating invoices with the same prefix)',
  )

export const collectionAlignment = z
  .enum(['subscription', 'anchored'])

  .describe(
    'BillingCollectionAlignment specifies when the pending line items should be collected into an invoice.',
  )

export const collectionMethod = z
  .enum(['charge_automatically', 'send_invoice'])

  .describe(
    'Collection method specifies how the invoice should be collected (automatic or manual).',
  )

export const priceType = z
  .enum(['free', 'flat', 'unit', 'graduated', 'volume'])

  .describe(
    "The type of the price. - `free`: No charge, the rate card is included at no cost. - `flat`: A fixed amount charged once per billing period, regardless of usage. - `unit`: A fixed rate charged per billing unit consumed. - `graduated`: Tiered pricing where each tier's rate applies only to usage within that tier. - `volume`: Tiered pricing where the rate for the highest tier reached applies to all units in the period.",
  )

export const featureUnitCostType = z
  .enum(['llm', 'manual'])
  .describe('The type of unit cost.')

export const systemAccountAccessToken = z
  .object({
    type: z.literal('http').describe('Http authentication'),
    scheme: z.literal('Bearer').describe('bearer auth scheme'),
  })

  .describe(
    'The system account access token is meant for automations and integrations that are not directly associated with a human identity.',
  )

export const personalAccessToken = z
  .object({
    type: z.literal('http').describe('Http authentication'),
    scheme: z.literal('Bearer').describe('bearer auth scheme'),
  })

  .describe(
    'The personal access token is meant to be used as an alternative to basic-auth when accessing Konnect via APIs.',
  )

export const konnectAccessToken = z
  .object({
    type: z.literal('http').describe('Http authentication'),
    scheme: z.literal('Bearer').describe('bearer auth scheme'),
  })

  .describe(
    'The Konnect access token is meant to be used by the Konnect dashboard and the decK CLI authenticate with.',
  )

export const updateMeterRequest = z
  .object({
    name: z
      .string()
      .min(1)
      .max(256)
      .optional()
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    dimensions: z
      .record(z.string(), z.string())
      .optional()

      .describe(
        'Named JSONPath expressions to extract the group by values from the event data. Keys must be unique and consist only alphanumeric and underscore characters.',
      ),
  })
  .describe('Meter update request.')

export const appCustomerDataStripe = z
  .object({
    customer_id: z.string().optional().describe('The Stripe customer ID used.'),
    default_payment_method_id: z
      .string()
      .optional()
      .describe('The Stripe default payment method ID.'),
    labels: labels.optional(),
  })
  .describe('Stripe customer data.')

export const appCustomerDataExternalInvoicing = z
  .object({
    labels: labels.optional(),
  })
  .describe('External invoicing customer data.')

export const billingCurrencyCode = z
  .union([currencyCode])
  .describe('Fiat or custom currency code.')

export const createCurrencyCode = z
  .union([currencyCode])
  .describe('Fiat or custom currency code.')

export const listCostBasesParamsFilter = z
  .object({
    fiat_code: currencyCode.optional(),
  })
  .describe('Filter options for listing cost bases.')

export const currencyAmount = z
  .object({
    amount: numeric,
    currency: currencyCode,
  })
  .describe('Monetary amount in a specific currency.')

export const priceFlat = z
  .object({
    type: z.literal('flat').describe('The type of the price.'),
    amount: numeric,
  })
  .describe('Flat price.')

export const priceUnit = z
  .object({
    type: z.literal('unit').describe('The type of the price.'),
    amount: numeric,
  })

  .describe(
    'Unit price. Charges a fixed rate per billing unit. When UnitConfig is present on the rate card, billing units are the converted quantities (e.g. GB instead of bytes).',
  )

export const rateCardDiscounts = z
  .object({
    percentage: z
      .number()
      .nonnegative()
      .lte(100)
      .optional()
      .describe('Percentage discount applied to the price (0–100).'),
    usage: numeric.optional(),
  })
  .describe('Discount configuration for a rate card.')

export const totals = z
  .object({
    amount: numeric,
    taxes_total: numeric,
    taxes_inclusive_total: numeric,
    taxes_exclusive_total: numeric,
    charges_total: numeric,
    discounts_total: numeric,
    credits_total: numeric,
    total: numeric,
  })

  .describe(
    'Totals contains the summaries of all calculations for a billing resource.',
  )

export const featureManualUnitCost = z
  .object({
    type: z
      .literal('manual')
      .describe('The type discriminator for manual unit cost.'),
    amount: numeric,
  })
  .describe('A fixed per-unit cost amount.')

export const featureLlmUnitCostPricing = z
  .object({
    input_per_token: numeric,
    output_per_token: numeric,
    cache_read_per_token: numeric.optional(),
    reasoning_per_token: numeric.optional(),
    cache_write_per_token: numeric.optional(),
  })
  .describe('Resolved per-token pricing from the LLM cost database.')

export const llmCostModelPricing = z
  .object({
    input_per_token: numeric,
    output_per_token: numeric,
    cache_read_per_token: numeric.optional(),
    cache_write_per_token: numeric.optional(),
    reasoning_per_token: numeric.optional(),
  })
  .describe('Token pricing for an LLM model, denominated per token.')

export const spendCommitments = z
  .object({
    minimum_amount: numeric.optional(),
    maximum_amount: numeric.optional(),
  })

  .describe(
    'Spend commitments for a rate card. The customer is committed to spend at least the minimum amount and at most the maximum amount.',
  )

export const queryFilterNumeric = z
  .object({
    gt: numeric.optional(),
    gte: numeric.optional(),
    lt: numeric.optional(),
    lte: numeric.optional(),
    get and() {
      return z
        .array(queryFilterNumeric)
        .min(1)
        .max(10)
        .optional()
        .describe('Combines the provided filters with a logical AND.')
    },
    get or() {
      return z
        .array(queryFilterNumeric)
        .min(1)
        .max(10)
        .optional()
        .describe('Combines the provided filters with a logical OR.')
    },
  })

  .describe(
    'A query filter for a numeric attribute. Operators are mutually exclusive, only one operator is allowed at a time.',
  )

export const cursorPaginationQuery = z
  .object({
    page: cursorPaginationQueryPage.optional(),
  })
  .describe('Cursor page query.')

export const listMetersParamsFilter = z
  .object({
    key: stringFieldFilter.optional(),
    name: stringFieldFilter.optional(),
  })
  .describe('Filter options for listing meters.')

export const listLlmCostPricesParamsFilter = z
  .object({
    provider: stringFieldFilter.optional(),
    model_id: stringFieldFilter.optional(),
    model_name: stringFieldFilter.optional(),
    currency: stringFieldFilter.optional(),
    source: stringFieldFilter.optional(),
  })
  .describe('Filter options for listing LLM cost prices.')

export const labelsFieldFilter = z
  .record(z.string(), stringFieldFilter)

  .describe(
    "Filters on the resource's `labels` field. The schema is a map keyed by the label name; each value is a `StringFieldFilter`. Both deepObject forms are accepted: `filter[labels][key]=value` (nested) and `filter[labels.key]=value` (dot-notation).",
  )

export const ulidFieldFilter = z
  .union([
    ulid,
    z.object({
      eq: ulid.optional(),
      oeq: z
        .array(ulid)
        .optional()

        .describe(
          'Returns entities that exact match any of the comma-delimited ULIDs in the filter string.',
        ),
      neq: ulid.optional(),
    }),
  ])

  .describe(
    'Filters on the given ULID field value by exact match. All properties are optional; provide exactly one to specify the comparison.',
  )

export const customerReference = z
  .object({
    id: ulid,
  })
  .describe('Customer reference.')

export const profileReference = z
  .object({
    id: ulid,
  })
  .describe('Billing profile reference.')

export const createResourceReference = z
  .object({
    id: ulid,
  })
  .describe('TaxCode reference.')

export const taxCodeReference = z
  .object({
    id: ulid,
  })
  .describe('TaxCode reference.')

export const creditGrantInvoiceReference = z
  .object({
    id: ulid.optional(),
    line: z
      .object({
        id: ulid,
      })
      .optional()
      .describe('Identifier of the invoice line associated with the grant.'),
  })
  .describe('Invoice references for the grant.')

export const billingCustomerReference = z
  .object({
    id: ulid,
  })
  .describe('Customer reference.')

export const subscriptionReference = z
  .object({
    id: ulid,
    phase: z
      .object({
        id: ulid,
        item: z
          .object({
            id: ulid,
          })
          .describe('The item of the phase.'),
      })
      .describe('The phase of the subscription.'),
  })

  .describe(
    'Subscription reference represents a reference to the specific subscription item this entity represents.',
  )

export const addonReference = z
  .object({
    id: ulid,
  })
  .describe('Addon reference.')

export const appReference = z
  .object({
    id: ulid,
  })
  .describe('App reference.')

export const currencyFiat = z
  .object({
    id: ulid,
    type: z.literal('fiat').describe('The type of the currency.'),
    name: z
      .string()
      .min(1)
      .max(256)

      .describe(
        'The name of the currency. It should be a human-readable string that represents the name of the currency, such as "US Dollar" or "Euro".',
      ),
    description: z
      .string()
      .min(1)
      .max(256)
      .optional()
      .describe('Description of the currency.'),
    symbol: z
      .string()
      .min(1)
      .optional()

      .describe(
        'The symbol of the currency. It should be a string that represents the symbol of the currency, such as "$" for US Dollar or "€" for Euro.',
      ),
    code: currencyCode,
  })
  .describe('Currency describes a currency supported by the billing system.')

export const featureReference = z
  .object({
    id: ulid,
  })
  .describe('Feature reference.')

export const dateTimeFieldFilter = z
  .union([
    dateTime,
    z.object({
      eq: dateTime.optional(),
      lt: dateTime.optional(),
      lte: dateTime.optional(),
      gt: dateTime.optional(),
      gte: dateTime.optional(),
    }),
  ])

  .describe(
    'Filters on the given datetime (RFC-3339) field value. All properties are optional; provide exactly one to specify the comparison.',
  )

export const event = z
  .object({
    id: z.string().min(1).describe('Identifies the event.'),
    source: z
      .string()
      .min(1)
      .describe('Identifies the context in which an event happened.'),
    specversion: z
      .string()
      .min(1)
      .default('1.0')

      .describe(
        'The version of the CloudEvents specification which the event uses.',
      ),
    type: z
      .string()
      .min(1)

      .describe(
        'Contains a value describing the type of event related to the originating occurrence.',
      ),
    datacontenttype: z
      .union([z.literal('application/json'), z.null()])
      .optional()

      .describe(
        'Content type of the CloudEvents data value. Only the value "application/json" is allowed over HTTP.',
      ),
    dataschema: z
      .union([z.string(), z.null()])
      .optional()
      .describe('Identifies the schema that data adheres to.'),
    subject: z
      .string()
      .min(1)

      .describe(
        'Describes the subject of the event in the context of the event producer (identified by source).',
      ),
    time: z
      .union([dateTime, z.null()])
      .optional()

      .describe(
        'Timestamp of when the occurrence happened. Must adhere to RFC 3339.',
      ),
    data: z
      .union([z.record(z.string(), z.unknown()), z.null()])
      .optional()

      .describe(
        'The event payload. Optional, if present it must be a JSON object.',
      ),
  })
  .describe('Metering event following the CloudEvents specification.')

export const meterQueryRow = z
  .object({
    value: numeric,
    from: dateTime,
    to: dateTime,
    dimensions: z
      .record(z.string(), z.string())

      .describe(
        'The dimensions the value is aggregated over. `subject` and `customer_id` are reserved dimensions.',
      ),
  })
  .describe('A row in the result of a meter query.')

export const appStripeCreateCustomerPortalSessionResult = z
  .object({
    id: z
      .string()

      .describe(
        'The ID of the customer portal session. See: https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-id',
      ),
    stripe_customer_id: z.string().describe('The ID of the stripe customer.'),
    configuration_id: z
      .string()

      .describe(
        'Configuration used to customize the customer portal. See: https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-configuration',
      ),
    livemode: z
      .boolean()

      .describe(
        'Livemode. See: https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-livemode',
      ),
    created_at: dateTime,
    return_url: z
      .string()

      .describe(
        'Return URL. See: https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-return_url',
      ),
    locale: z
      .string()

      .describe(
        'The IETF language tag of the locale customer portal is displayed in. See: https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-locale',
      ),
    url: z
      .string()

      .describe(
        'The URL to redirect the customer to after they have completed their requested actions.',
      ),
  })

  .describe(
    'Result of creating a [Stripe Customer Portal Session](https://docs.stripe.com/api/customer_portal/sessions/object). Contains all the information needed to redirect the customer to the Stripe Customer Portal.',
  )

export const closedPeriod = z
  .object({
    from: dateTime,
    to: dateTime,
  })

  .describe(
    'A period with defined start and end dates. The period is always inclusive at the start and exclusive at the end.',
  )

export const costBasis = z
  .object({
    id: ulid,
    fiat_code: currencyCode,
    rate: numeric,
    effective_from: dateTime.optional(),
    created_at: dateTime,
  })
  .describe('Describes currency basis supported by billing system.')

export const createCostBasisRequest = z
  .object({
    fiat_code: currencyCode,
    rate: numeric,
    effective_from: dateTime.optional(),
  })
  .describe('CostBasis create request.')

export const featureCostQueryRow = z
  .object({
    usage: numeric,
    cost: z
      .union([numeric, z.null()])

      .describe(
        'The computed cost amount (usage × unit cost). Null when pricing is not available for the given combination of dimensions.',
      ),
    currency: currencyCode,
    detail: z
      .string()
      .optional()

      .describe(
        'Detail message when cost amount is null, explaining why the cost could not be resolved.',
      ),
    from: dateTime,
    to: dateTime,
    dimensions: z
      .record(z.string(), z.string())

      .describe(
        'The dimensions the value is aggregated over. `subject` and `customer_id` are reserved dimensions.',
      ),
  })
  .describe('A row in the result of a feature cost query.')

export const resource = z
  .object({
    id: ulid,
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    created_at: dateTime,
    updated_at: dateTime,
    deleted_at: dateTime.optional(),
  })
  .describe('Represents common fields of resources.')

export const resourceImmutable = z
  .object({
    id: ulid,
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    created_at: dateTime,
  })
  .describe('Represents common fields of immutable resources.')

export const queryFilterDateTime = z
  .object({
    gt: dateTime.optional(),
    gte: dateTime.optional(),
    lt: dateTime.optional(),
    lte: dateTime.optional(),
    get and() {
      return z
        .array(queryFilterDateTime)
        .min(1)
        .max(10)
        .optional()
        .describe('Combines the provided filters with a logical AND.')
    },
    get or() {
      return z
        .array(queryFilterDateTime)
        .min(1)
        .max(10)
        .optional()
        .describe('Combines the provided filters with a logical OR.')
    },
  })

  .describe(
    'A query filter for a time attribute. Operators are mutually exclusive, only one operator is allowed at a time.',
  )

export const cursorMeta = z
  .object({
    page: cursorMetaPage,
  })
  .describe('Cursor pagination metadata.')

export const invalidParameterStandard = z
  .object({
    field: z.string().describe('The name of the field that failed validation.'),
    rule: invalidRules.optional(),
    source: z
      .string()
      .optional()

      .describe(
        'The part of the request the field came from (e.g. `body`, `query`).',
      ),
    reason: z
      .string()

      .describe(
        'A human readable explanation of why the field failed validation.',
      ),
  })
  .describe('A parameter that failed a standard validation rule.')

export const invalidParameterMinimumLength = z
  .object({
    field: z.string().describe('The name of the field that failed validation.'),
    rule: invalidParameterMinimumRule,
    minimum: z.number().int().describe('The minimum allowed value or length.'),
    source: z
      .string()
      .optional()

      .describe(
        'The part of the request the field came from (e.g. `body`, `query`).',
      ),
    reason: z
      .string()

      .describe(
        'A human readable explanation of why the field failed validation.',
      ),
  })

  .describe(
    'A parameter that failed a minimum-length (or minimum-value) validation rule.',
  )

export const invalidParameterMaximumLength = z
  .object({
    field: z.string().describe('The name of the field that failed validation.'),
    rule: invalidParameterMaximumRule,
    maximum: z.number().int().describe('The maximum allowed value or length.'),
    source: z
      .string()
      .optional()

      .describe(
        'The part of the request the field came from (e.g. `body`, `query`).',
      ),
    reason: z
      .string()

      .describe(
        'A human readable explanation of why the field failed validation.',
      ),
  })

  .describe(
    'A parameter that failed a maximum-length (or maximum-value) validation rule.',
  )

export const invalidParameterChoiceItem = z
  .object({
    field: z.string().describe('The name of the field that failed validation.'),
    rule: invalidParameterChoiceRule,
    reason: z
      .string()

      .describe(
        'A human readable explanation of why the field failed validation.',
      ),
    choices: z
      .array(z.unknown())
      .min(1)
      .describe('The allowed choices for the field.'),
    source: z
      .string()
      .optional()

      .describe(
        'The part of the request the field came from (e.g. `body`, `query`).',
      ),
  })
  .describe('A parameter whose value was not one of the allowed choices.')

export const invalidParameterDependentItem = z
  .object({
    field: z.string().describe('The name of the field that failed validation.'),
    rule: invalidParameterDependentRule,
    reason: z
      .string()

      .describe(
        'A human readable explanation of why the field failed validation.',
      ),
    dependents: z
      .array(z.unknown())
      .describe('The fields that this field depends on.'),
    source: z
      .string()
      .optional()

      .describe(
        'The part of the request the field came from (e.g. `body`, `query`).',
      ),
  })
  .describe('A parameter that failed a dependent-fields validation rule.')

export const unauthorized = baseError.describe('Unauthorized.')

export const forbidden = baseError.describe('Forbidden.')

export const notFound = baseError.describe('Not Found.')

export const gone = baseError.describe('Gone.')

export const conflict = baseError.describe('Conflict.')

export const payloadTooLarge = baseError.describe('Payload Too Large.')

export const unsupportedMediaType = baseError.describe(
  'Unsupported Media Type.',
)

export const unprocessableContent = baseError.describe('Unprocessable Content.')

export const tooManyRequests = baseError.describe('Too Many Requests.')

export const internal = baseError.describe('Internal Server Error.')

export const notImplemented = baseError.describe('Not Implemented.')

export const notAvailable = baseError.describe('Not Available.')

export const createCreditGrantFilters = z
  .object({
    features: z
      .array(resourceKey)
      .optional()

      .describe(
        'Limit the credit grant to specific features. If no features are specified, the credit grant can be used for any feature.',
      ),
  })
  .describe('Filters for the credit grant.')

export const creditGrantFilters = z
  .object({
    features: z
      .array(resourceKey)
      .optional()

      .describe(
        'Limit the credit grant to specific features. If no features are specified, the credit grant can be used for any feature.',
      ),
  })
  .describe('Filters for the credit grant.')

export const upsertPlanAddonRequest = z
  .object({
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    from_plan_phase: resourceKey,
    max_quantity: z
      .number()
      .int()
      .gte(1)
      .optional()

      .describe(
        'The maximum number of times the add-on can be purchased for the plan. For single-instance add-ons this field must be omitted. For multi-instance add-ons when omitted, unlimited quantity can be purchased.',
      ),
  })
  .describe('PlanAddon upsert request.')

export const resourceWithKey = z
  .object({
    id: ulid,
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    created_at: dateTime,
    updated_at: dateTime,
    deleted_at: dateTime.optional(),
    key: resourceKey,
  })
  .describe('Represents common fields of resources with a key.')

export const ulidOrResourceKey = z
  .union([ulid, resourceKey])
  .describe('ULID ID or Resource Key.')

export const createMeterRequest = z
  .object({
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    key: resourceKey,
    aggregation: meterAggregation,
    event_type: z
      .string()
      .min(1)
      .describe('The event type to include in the aggregation.'),
    events_from: dateTime.optional(),
    value_property: z
      .string()
      .min(1)
      .optional()

      .describe(
        "JSONPath expression to extract the value from the ingested event's data property. The ingested value for sum, avg, min, and max aggregations is a number or a string that can be parsed to a number. For unique_count aggregation, the ingested value must be a string. For count aggregation the value_property is ignored.",
      ),
    dimensions: z
      .record(z.string(), z.string())
      .optional()

      .describe(
        'Named JSONPath expressions to extract the group by values from the event data. Keys must be unique and consist only alphanumeric and underscore characters.',
      ),
  })
  .describe('Meter create request.')

export const meter = z
  .object({
    id: ulid,
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    created_at: dateTime,
    updated_at: dateTime,
    deleted_at: dateTime.optional(),
    key: resourceKey,
    aggregation: meterAggregation,
    event_type: z
      .string()
      .min(1)
      .describe('The event type to include in the aggregation.'),
    events_from: dateTime.optional(),
    value_property: z
      .string()
      .min(1)
      .optional()

      .describe(
        "JSONPath expression to extract the value from the ingested event's data property. The ingested value for sum, avg, min, and max aggregations is a number or a string that can be parsed to a number. For unique_count aggregation, the ingested value must be a string. For count aggregation the value_property is ignored.",
      ),
    dimensions: z
      .record(z.string(), z.string())
      .optional()

      .describe(
        'Named JSONPath expressions to extract the group by values from the event data. Keys must be unique and consist only alphanumeric and underscore characters.',
      ),
  })

  .describe(
    'A meter is a configuration that defines how to match and aggregate events.',
  )

export const paginatedMeta = z
  .object({
    page: pageMeta,
  })
  .describe('Pagination metadata.')

export const queryFilterStringMapItem = z
  .object({
    exists: z.boolean().optional().describe('The attribute exists.'),
    eq: z
      .string()
      .optional()
      .describe('The attribute equals the provided value.'),
    neq: z
      .string()
      .optional()
      .describe('The attribute does not equal the provided value.'),
    in: z
      .array(z.string())
      .min(1)
      .max(100)
      .optional()
      .describe('The attribute is one of the provided values.'),
    nin: z
      .array(z.string())
      .min(1)
      .max(100)
      .optional()
      .describe('The attribute is not one of the provided values.'),
    contains: z
      .string()
      .optional()
      .describe('The attribute contains the provided value.'),
    ncontains: z
      .string()
      .optional()
      .describe('The attribute does not contain the provided value.'),
    and: z
      .array(queryFilterString)
      .min(1)
      .max(10)
      .optional()
      .describe('Combines the provided filters with a logical AND.'),
    or: z
      .array(queryFilterString)
      .min(1)
      .max(10)
      .optional()
      .describe('Combines the provided filters with a logical OR.'),
  })

  .describe(
    'A query filter for an item in a string map attribute. Operators are mutually exclusive, only one operator is allowed at a time.',
  )

export const ulidOrExternalResourceKey = z
  .union([ulid, externalResourceKey])
  .describe('ULID ID or External Resource Key.')

export const customerKeyReference = z
  .object({
    key: externalResourceKey,
  })
  .describe('Customer reference by external key.')

export const customerUsageAttribution = z
  .object({
    subject_keys: z
      .array(usageAttributionSubjectKey)

      .describe(
        'The subjects that are attributed to the customer. Can be empty when no usage event subjects are associated with the customer.',
      ),
  })

  .describe(
    'Mapping to attribute metered usage to the customer. One customer can have zero or more subjects, but one subject can only belong to one customer.',
  )

export const billingAddress = z
  .object({
    country: countryCode.optional(),
    postal_code: z.string().optional().describe('Postal code.'),
    state: z.string().optional().describe('State or province.'),
    city: z.string().optional().describe('City.'),
    line1: z.string().optional().describe('First line of the address.'),
    line2: z.string().optional().describe('Second line of the address.'),
    phone_number: z.string().optional().describe('Phone number.'),
  })
  .describe('Address')

export const address = z
  .object({
    country: countryCode.optional(),
    postal_code: z.string().optional().describe('Postal code.'),
    state: z.string().optional().describe('State or province.'),
    city: z.string().optional().describe('City.'),
    line1: z.string().optional().describe('First line of the address.'),
    line2: z.string().optional().describe('Second line of the address.'),
    phone_number: z.string().optional().describe('Phone number.'),
  })
  .describe('Address')

export const appStripeCreateCheckoutSessionCustomerUpdate = z
  .object({
    address: appStripeCreateCheckoutSessionCustomerUpdateBehavior
      .optional()
      .default('never'),
    name: appStripeCreateCheckoutSessionCustomerUpdateBehavior
      .optional()
      .default('never'),
    shipping: appStripeCreateCheckoutSessionCustomerUpdateBehavior
      .optional()
      .default('never'),
  })

  .describe(
    'Controls which customer fields can be updated by the checkout session.',
  )

export const appStripeCreateCheckoutSessionConsentCollectionPaymentMethodReuseAgreement =
  z
    .object({
      position:
        appStripeCreateCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPosition.optional(),
    })
    .describe('Payment method reuse agreement configuration.')

export const appStripeCreateCheckoutSessionTaxIdCollection = z
  .object({
    enabled: z
      .boolean()
      .optional()
      .default(false)
      .describe('Enable tax ID collection during checkout. Defaults to false.'),
    required: appStripeCreateCheckoutSessionTaxIdCollectionRequired
      .optional()
      .default('never'),
  })
  .describe('Tax ID collection configuration for checkout sessions.')

export const appStripeCreateCheckoutSessionResult = z
  .object({
    customer_id: ulid,
    stripe_customer_id: z.string().describe('The Stripe customer ID.'),
    session_id: z.string().describe('The Stripe checkout session ID.'),
    setup_intent_id: z
      .string()

      .describe(
        'The setup intent ID created for collecting the payment method.',
      ),
    client_secret: z
      .string()
      .optional()

      .describe(
        'Client secret for initializing Stripe.js on the client side. Required for embedded checkout sessions. See: https://docs.stripe.com/payments/checkout/custom-success-page',
      ),
    client_reference_id: z
      .string()
      .optional()

      .describe(
        'The client reference ID provided in the request. Useful for reconciling the session with your internal systems.',
      ),
    customer_email: z
      .string()
      .optional()
      .describe("Customer's email address if provided to Stripe."),
    currency: currencyCode.optional(),
    created_at: dateTime,
    expires_at: dateTime.optional(),
    metadata: z
      .record(z.string(), z.string())
      .optional()
      .describe('Metadata attached to the checkout session.'),
    status: z
      .string()
      .optional()

      .describe(
        'The status of the checkout session. See: https://docs.stripe.com/api/checkout/sessions/object#checkout_session_object-status',
      ),
    url: z
      .string()
      .optional()

      .describe(
        'URL to redirect customers to the checkout page (for hosted mode).',
      ),
    mode: appStripeCheckoutSessionMode,
    cancel_url: z
      .string()
      .optional()

      .describe(
        'The cancel URL where customers are redirected if they cancel.',
      ),
    success_url: z
      .string()
      .optional()

      .describe(
        'The success URL where customers are redirected after completion.',
      ),
    return_url: z
      .string()
      .optional()
      .describe('The return URL for embedded sessions after authentication.'),
  })

  .describe(
    'Result of creating a Stripe Checkout Session. Contains all the information needed to redirect customers to the checkout or initialize an embedded checkout flow.',
  )

export const customerStripeCreateCustomerPortalSessionRequest = z
  .object({
    stripe_options: appStripeCreateCustomerPortalSessionOptions,
  })

  .describe(
    'Request to create a Stripe Customer Portal Session for the customer. Useful to redirect the customer to the Stripe Customer Portal to manage their payment methods, change their billing address and access their invoice history. Only returns URL if the customer billing profile is linked to a stripe app and customer.',
  )

export const entitlementAccessResult = z
  .object({
    type: entitlementType,
    feature_key: resourceKey,
    has_access: z
      .boolean()

      .describe(
        'Whether the customer has access to the feature. Always true for `boolean` and `static` entitlements. Depends on balance for `metered` entitlements.',
      ),
    config: z
      .string()
      .optional()

      .describe(
        'Only available for static entitlements. Config is the JSON parsable configuration of the entitlement. Useful to describe per customer configuration.',
      ),
  })
  .describe('Entitlement access result.')

export const createCreditGrantPurchase = z
  .object({
    currency: currencyCode,
    per_unit_cost_basis: numeric.optional().default('1.0'),
    availability_policy: creditAvailabilityPolicy
      .optional()
      .default('on_creation'),
  })
  .describe('Purchase and payment terms of the grant.')

export const recurringPeriod = z
  .object({
    anchor: dateTime,
    interval: iso8601Duration,
  })
  .describe('Recurring period with an anchor and an interval.')

export const rateCardMeteredEntitlement = z
  .object({
    type: z
      .literal('metered')
      .describe('The type of the entitlement template.'),
    is_soft_limit: z
      .boolean()
      .optional()
      .default(false)

      .describe(
        'If soft limit is true, the subject can use the feature even if the entitlement is exhausted; access remains granted.',
      ),
    limit: z
      .number()
      .nonnegative()
      .optional()

      .describe(
        "The amount of usage granted each usage period, in the feature's unit. Usage is counted against this allowance and the balance resets every usage period. When `is_soft_limit` is true the subject keeps access after the limit is reached; otherwise access is denied once the allowance is exhausted.",
      ),
    usage_period: iso8601Duration.optional(),
  })
  .describe('The entitlement template of a metered entitlement.')

export const creditGrantPurchase = z
  .object({
    currency: currencyCode,
    per_unit_cost_basis: numeric.optional().default('1.0'),
    amount: numeric,
    availability_policy: creditAvailabilityPolicy
      .optional()
      .default('on_creation'),
    settlement_status: creditPurchasePaymentSettlementStatus.optional(),
  })
  .describe('Purchase and payment terms of the grant.')

export const updateCreditGrantExternalSettlementRequest = z
  .object({
    status: creditPurchasePaymentSettlementStatus,
  })

  .describe(
    'Request body for updating the external payment settlement status of a credit grant.',
  )

export const listCreditGrantsParamsFilter = z
  .object({
    status: creditGrantStatus.optional(),
    currency: currencyCode.optional(),
  })
  .describe('Filter options for listing credit grants.')

export const getCreditBalanceParamsFilter = z
  .object({
    currency: stringFieldFilterExact.optional(),
    feature_key: stringFieldFilter.optional(),
  })
  .describe('Filter options for getting a credit balance.')

export const listChargesParamsFilter = z
  .object({
    status: stringFieldFilterExact.optional(),
  })
  .describe('Filter options for listing charges.')

export const listPlansParamsFilter = z
  .object({
    key: stringFieldFilter.optional(),
    name: stringFieldFilter.optional(),
    status: stringFieldFilterExact.optional(),
    currency: stringFieldFilterExact.optional(),
  })
  .describe('Filter options for listing plans.')

export const subscriptionCreate = z
  .object({
    labels: labels.optional(),
    settlement_mode: settlementMode.optional(),
    customer: z
      .object({
        id: ulid.optional(),
        key: externalResourceKey.optional(),
      })
      .describe('The customer to create the subscription for.'),
    plan: z
      .object({
        id: ulid.optional(),
        key: resourceKey.optional(),
        version: z
          .number()
          .int()
          .optional()

          .describe(
            'The plan version of the subscription, if any. If not provided, the latest version of the plan will be used.',
          ),
      })
      .describe('The plan reference of the subscription.'),
    billing_anchor: dateTime.optional(),
  })
  .describe('Subscription create request.')

export const rateCardProrationConfiguration = z
  .object({
    mode: rateCardProrationMode,
  })
  .describe('The proration configuration of the rate card.')

export const subscription = z
  .object({
    id: ulid,
    labels: labels.optional(),
    created_at: dateTime,
    updated_at: dateTime,
    deleted_at: dateTime.optional(),
    customer_id: ulid,
    plan_id: ulid.optional(),
    billing_anchor: dateTime,
    status: subscriptionStatus,
    settlement_mode: settlementMode.optional(),
  })
  .describe('Subscription.')

export const subscriptionEditTiming = z
  .union([subscriptionEditTimingEnum, dateTime])

  .describe(
    'Subscription edit timing defined when the changes should take effect. If the provided configuration is not supported by the subscription, an error will be returned.',
  )

export const appCatalogItem = z
  .object({
    type: appType,
    name: z.string().describe('Name of the app.'),
    description: z.string().describe('Description of the app.'),
  })

  .describe(
    'Available apps for billing integrations to connect with third-party services. Apps can have various capabilities like syncing data from or to external systems, integrating with third-party services for tax calculation, delivery of invoices, collection of payments, etc.',
  )

export const taxCodeAppMapping = z
  .object({
    app_type: appType,
    tax_code: z.string().describe('Tax code.'),
  })
  .describe('Mapping of app types to tax codes.')

export const partyTaxIdentity = z
  .object({
    code: taxIdentificationCode.optional(),
  })

  .describe(
    'Identity stores the details required to identify an entity for tax purposes in a specific country.',
  )

export const workflowInvoicingSettings = z
  .object({
    auto_advance: z
      .boolean()
      .optional()
      .default(true)

      .describe(
        'Whether to automatically issue the invoice after the draftPeriod has passed.',
      ),
    draft_period: z
      .string()
      .optional()
      .default('P0D')

      .describe(
        'The period for the invoice to be kept in draft status for manual reviews.',
      ),
    progressive_billing: z
      .boolean()
      .optional()
      .default(true)
      .describe('Should progressive billing be allowed for this workflow?'),
    subscription_end_proration_mode:
      workflowInvoicingSubscriptionEndProrationMode
        .optional()
        .default('bill_actual_period'),
  })
  .describe('Invoice settings for a billing workflow.')

export const workflowPaymentSettings = z
  .discriminatedUnion('collection_method', [
    workflowPaymentChargeAutomaticallySettings,
    workflowPaymentSendInvoiceSettings,
  ])
  .describe('Payment settings for a billing workflow.')

export const listCurrenciesParamsFilter = z
  .object({
    type: currencyType.optional(),
    code: stringFieldFilter.optional(),
  })
  .describe('Filter options for listing currencies.')

export const currencyCustom = z
  .object({
    id: ulid,
    type: z.literal('custom').describe('The type of the currency.'),
    name: z
      .string()
      .min(1)
      .max(256)

      .describe(
        'The name of the currency. It should be a human-readable string that represents the name of the currency, such as "US Dollar" or "Euro".',
      ),
    description: z
      .string()
      .min(1)
      .max(256)
      .optional()
      .describe('Description of the currency.'),
    symbol: z
      .string()
      .min(1)
      .optional()

      .describe(
        'The symbol of the currency. It should be a string that represents the symbol of the currency, such as "$" for US Dollar or "€" for Euro.',
      ),
    code: currencyCodeCustom,
    created_at: dateTime,
  })
  .describe('Describes custom currency.')

export const createCurrencyCustomRequest = z
  .object({
    name: z
      .string()
      .min(1)
      .max(256)

      .describe(
        'The name of the currency. It should be a human-readable string that represents the name of the currency, such as "US Dollar" or "Euro".',
      ),
    description: z
      .string()
      .min(1)
      .max(256)
      .optional()
      .describe('Description of the currency.'),
    symbol: z
      .string()
      .min(1)
      .optional()

      .describe(
        'The symbol of the currency. It should be a string that represents the symbol of the currency, such as "$" for US Dollar or "€" for Euro.',
      ),
    code: currencyCodeCustom,
  })
  .describe('CurrencyCustom create request.')

export const unitConfig = z
  .object({
    operation: unitConfigOperation,
    conversion_factor: numeric,
    rounding: unitConfigRoundingMode.optional().default('none'),
    precision: z
      .number()
      .int()
      .optional()
      .default(0)

      .describe(
        'The number of decimal places to retain after rounding. Only meaningful when rounding is not "none". Defaults to 0 (round to whole numbers).',
      ),
    display_unit: z
      .string()
      .optional()

      .describe(
        'A human-readable label for the converted unit shown on invoices and in the customer portal (e.g., "GB", "hours", "M tokens"). Optional. When omitted, no unit label is rendered.',
      ),
  })

  .describe(
    'Unit conversion configuration. Transforms raw metered quantities into billing-ready units before pricing and entitlement evaluation. Applied at the rate card level so the same feature can be billed in different units across plans. Examples: - Meter bytes, bill GB: operation=divide, conversionFactor=1e9, rounding=ceiling, displayUnit="GB" - Meter seconds, bill hours: operation=divide, conversionFactor=3600, rounding=ceiling, displayUnit="hours" - Cost + 20% margin: operation=multiply, conversionFactor=1.2 - Bill per million tokens: operation=divide, conversionFactor=1e6, rounding=ceiling, displayUnit="M" v1 equivalents: - DynamicPrice(multiplier): operation=multiply, conversionFactor=multiplier + UnitPrice(amount=1) - PackagePrice(amount, quantityPerPkg): operation=divide, conversionFactor=quantityPerPkg, rounding=ceiling + UnitPrice(amount)',
  )

export const governanceQueryRequest = z
  .object({
    include_credits: z
      .boolean()
      .optional()
      .default(false)

      .describe(
        'Whether to include credit balance availability for each resolved customer. When true, each feature evaluation includes credit balance checks. Defaults to `false`.',
      ),
    customer: governanceQueryRequestCustomers,
    feature: governanceQueryRequestFeatures.optional(),
  })
  .describe('Query to evaluate feature access for a list of customers.')

export const governanceFeatureAccessReason = z
  .object({
    code: governanceFeatureAccessReasonCode,
    message: z.string().describe('Human-readable description of the error.'),
    attributes: z
      .record(z.string(), z.unknown())
      .optional()
      .describe('Additional structured context.'),
  })
  .describe('Reason a feature is not accessible to a customer.')

export const governanceQueryError = z
  .object({
    code: governanceQueryErrorCode,
    message: z.string().describe('Human-readable description of the error.'),
    attributes: z
      .record(z.string(), z.unknown())
      .optional()
      .describe('Additional structured context.'),
    customer: z
      .string()
      .optional()

      .describe(
        'The customer identifier from the request that produced this error.',
      ),
  })

  .describe(
    'Query error within a partially successful governance query response.',
  )

export const appCustomerData = z
  .object({
    stripe: appCustomerDataStripe.optional(),
    external_invoicing: appCustomerDataExternalInvoicing.optional(),
  })
  .describe('App customer data.')

export const upsertAppCustomerDataRequest = z
  .object({
    stripe: appCustomerDataStripe.optional(),
    external_invoicing: appCustomerDataExternalInvoicing.optional(),
  })
  .describe('AppCustomerData upsert request.')

export const creditAdjustment = z
  .object({
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    currency: billingCurrencyCode,
    amount: numeric,
  })

  .describe(
    "A credit adjustment can be used to make manual adjustments to a customer's credit balance. Supported use-cases: - Usage correction",
  )

export const creditBalance = z
  .object({
    currency: billingCurrencyCode,
    pending: numeric,
    available: numeric,
  })
  .describe('The credit balance by currency.')

export const createCreditAdjustmentRequest = z
  .object({
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    currency: billingCurrencyCode,
    amount: numeric,
  })
  .describe('CreditAdjustment create request.')

export const listCreditTransactionsParamsFilter = z
  .object({
    type: creditTransactionType.optional(),
    currency: billingCurrencyCode.optional(),
    feature_key: stringFieldFilter.optional(),
  })
  .describe('Filter options for listing credit transactions.')

export const creditTransaction = z
  .object({
    id: ulid,
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    created_at: dateTime,
    booked_at: dateTime,
    type: creditTransactionType,
    currency: billingCurrencyCode,
    amount: numeric,
    available_balance: z
      .object({
        before: numeric,
        after: numeric,
      })
      .describe('The available balance before and after the transaction.'),
  })

  .describe(
    "A credit transaction represents a single credit movement on the customer's balance. Credit transactions are immutable.",
  )

export const priceTier = z
  .object({
    up_to_amount: numeric.optional(),
    flat_price: priceFlat.optional(),
    unit_price: priceUnit.optional(),
  })

  .describe(
    'A price tier used in graduated and volume pricing. At least one price component (flat_price or unit_price) must be set. When UnitConfig is present on the rate card, up_to_amount is expressed in converted billing units.',
  )

export const chargeTotals = z
  .object({
    booked: totals,
    realtime: totals.optional(),
  })

  .describe(
    'The totals of a change. RealTime is only expanded when the `real_time_usage` expand is used.',
  )

export const featureLlmUnitCost = z
  .object({
    type: z
      .literal('llm')
      .describe('The type discriminator for LLM unit cost.'),
    provider_property: z
      .string()
      .optional()

      .describe(
        'Meter group-by property that holds the LLM provider. Use this when the meter has a group-by dimension for provider. Mutually exclusive with `provider`.',
      ),
    provider: z
      .string()
      .optional()

      .describe(
        'Static LLM provider value (e.g., "openai", "anthropic"). Use this when the feature tracks a single provider. Mutually exclusive with `provider_property`.',
      ),
    model_property: z
      .string()
      .optional()

      .describe(
        'Meter group-by property that holds the model ID. Use this when the meter has a group-by dimension for model. Mutually exclusive with `model`.',
      ),
    model: z
      .string()
      .optional()

      .describe(
        'Static model ID value (e.g., "gpt-4", "claude-3-5-sonnet"). Use this when the feature tracks a single model. Mutually exclusive with `model_property`.',
      ),
    token_type_property: z
      .string()
      .optional()

      .describe(
        'Meter group-by property that holds the token type. Use this when the meter has a group-by dimension for token type. Mutually exclusive with `token_type`.',
      ),
    token_type: featureLlmTokenType.optional(),
    pricing: featureLlmUnitCostPricing.optional(),
  })

  .describe(
    'LLM cost lookup configuration. Each dimension (provider, model, token type) can be specified as either a static value or a meter group-by property name (mutually exclusive).',
  )

export const llmCostPrice = z
  .object({
    id: ulid,
    provider: llmCostProvider,
    model: llmCostModel,
    pricing: llmCostModelPricing,
    currency: currencyCode,
    source: llmCostPriceSource,
    effective_from: dateTime,
    effective_to: dateTime.optional(),
    created_at: dateTime,
    updated_at: dateTime,
  })

  .describe(
    'An LLM cost price record, representing the cost per token for a specific model from a specific provider.',
  )

export const llmCostOverrideCreate = z
  .object({
    provider: z.string().describe('Provider/vendor of the model.'),
    model_id: z.string().describe('Canonical model identifier.'),
    model_name: z.string().optional().describe('Human-readable model name.'),
    pricing: llmCostModelPricing,
    currency: currencyCode,
    effective_from: dateTime,
    effective_to: dateTime.optional(),
  })

  .describe(
    'Input for creating a per-namespace price override. Unique per provider, model and currency. If an override already exists for the given provider, model and currency, it will be updated. If an override does not exist, it will be created.',
  )

export const listCustomersParamsFilter = z
  .object({
    key: stringFieldFilter.optional(),
    name: stringFieldFilter.optional(),
    primary_email: stringFieldFilter.optional(),
    usage_attribution_subject_key: stringFieldFilter.optional(),
    plan_key: stringFieldFilter.optional(),
    billing_profile_id: ulidFieldFilter.optional(),
  })
  .describe('Filter options for listing customers.')

export const listSubscriptionsParamsFilter = z
  .object({
    id: ulidFieldFilter.optional(),
    customer_id: ulidFieldFilter.optional(),
    status: stringFieldFilterExact.optional(),
    plan_id: ulidFieldFilter.optional(),
    plan_key: stringFieldFilterExact.optional(),
  })
  .describe('Filter options for listing subscriptions.')

export const listAppsParamsFilter = z
  .object({
    id: ulidFieldFilter.optional(),
    name: stringFieldFilter.optional(),
    type: stringFieldFilterExact.optional(),
    status: stringFieldFilterExact.optional(),
  })
  .describe('Filter options for listing apps.')

export const listFeatureParamsFilter = z
  .object({
    meter_id: ulidFieldFilter.optional(),
    key: stringFieldFilter.optional(),
    name: stringFieldFilter.optional(),
  })
  .describe('Filter options for listing features.')

export const listAddonsParamsFilter = z
  .object({
    id: ulidFieldFilter.optional(),
    key: stringFieldFilter.optional(),
    name: stringFieldFilter.optional(),
    status: stringFieldFilterExact.optional(),
    currency: stringFieldFilterExact.optional(),
  })
  .describe('Filter options for listing add-ons.')

export const createCreditGrantTaxConfig = z
  .object({
    behavior: taxBehavior.optional(),
    tax_code: createResourceReference.optional(),
  })

  .describe(
    'Tax configuration for a credit grant. Tax configuration should be provided to ensure correct revenue recognition, including for externally funded grants.',
  )

export const creditGrantTaxConfig = z
  .object({
    behavior: taxBehavior.optional(),
    tax_code: taxCodeReference.optional(),
  })

  .describe(
    'Tax configuration for a credit grant. Tax configuration should be provided to ensure correct revenue recognition, including for externally funded grants.',
  )

export const taxConfig = z
  .object({
    behavior: taxBehavior.optional(),
    stripe: taxConfigStripe.optional(),
    external_invoicing: taxConfigExternalInvoicing.optional(),
    tax_code_id: ulid.optional(),
    tax_code: taxCodeReference.optional(),
  })
  .describe('Set of provider specific tax configs.')

export const rateCardTaxConfig = z
  .object({
    behavior: taxBehavior.optional(),
    code: taxCodeReference,
  })
  .describe('The tax config of the rate card.')

export const organizationDefaultTaxCodes = z
  .object({
    invoicing_tax_code: taxCodeReference,
    credit_grant_tax_code: taxCodeReference,
    created_at: dateTime,
    updated_at: dateTime,
  })

  .describe(
    'Organization-level default tax code references. Stores the default tax codes applied to specific billing contexts for this organization. Provisioned automatically when the organization is created.',
  )

export const updateOrganizationDefaultTaxCodesRequest = z
  .object({
    invoicing_tax_code: taxCodeReference.optional(),
    credit_grant_tax_code: taxCodeReference.optional(),
  })
  .describe('OrganizationDefaultTaxCodes update request.')

export const subscriptionAddon = z
  .object({
    id: ulid,
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    created_at: dateTime,
    updated_at: dateTime,
    deleted_at: dateTime.optional(),
    addon: addonReference,
    quantity: z
      .number()
      .int()
      .gte(1)

      .describe(
        'The quantity of the add-on. Always 1 for single instance add-ons.',
      ),
    quantity_at: dateTime,
    active_from: dateTime,
    active_to: dateTime.optional(),
  })
  .describe('Addon purchased with a subscription.')

export const planAddon = z
  .object({
    id: ulid,
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    created_at: dateTime,
    updated_at: dateTime,
    deleted_at: dateTime.optional(),
    addon: addonReference,
    from_plan_phase: resourceKey,
    max_quantity: z
      .number()
      .int()
      .gte(1)
      .optional()

      .describe(
        'The maximum number of times the add-on can be purchased for the plan. For single-instance add-ons this field must be omitted. For multi-instance add-ons when omitted, unlimited quantity can be purchased.',
      ),
    validation_errors: z
      .array(productCatalogValidationError)
      .optional()
      .describe('List of validation errors.'),
  })

  .describe(
    'PlanAddon represents an association between a plan and an add-on, controlling which add-ons are available for purchase within a plan.',
  )

export const createPlanAddonRequest = z
  .object({
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    addon: addonReference,
    from_plan_phase: resourceKey,
    max_quantity: z
      .number()
      .int()
      .gte(1)
      .optional()

      .describe(
        'The maximum number of times the add-on can be purchased for the plan. For single-instance add-ons this field must be omitted. For multi-instance add-ons when omitted, unlimited quantity can be purchased.',
      ),
  })
  .describe('PlanAddon create request.')

export const profileAppReferences = z
  .object({
    tax: appReference,
    invoicing: appReference,
    payment: appReference,
  })
  .describe('References to the applications used by a billing profile.')

export const listEventsParamsFilter = z
  .object({
    id: stringFieldFilter.optional(),
    source: stringFieldFilter.optional(),
    subject: stringFieldFilter.optional(),
    type: stringFieldFilter.optional(),
    customer_id: ulidFieldFilter.optional(),
    time: dateTimeFieldFilter.optional(),
    ingested_at: dateTimeFieldFilter.optional(),
    stored_at: dateTimeFieldFilter.optional(),
  })
  .describe('Filter options for listing ingested events.')

export const resourceFilters = z
  .object({
    name: stringFieldFilter.optional(),
    labels: labelsFieldFilter.optional(),
    public_labels: labelsFieldFilter.optional(),
    created_at: dateTimeFieldFilter.optional(),
    updated_at: dateTimeFieldFilter.optional(),
    deleted_at: dateTimeFieldFilter.optional(),
  })
  .describe('Resource filters.')

export const fieldFilters = z
  .object({
    boolean: booleanFieldFilter.optional(),
    numeric: numericFieldFilter.optional(),
    string: stringFieldFilter.optional(),
    string_exact: stringFieldFilterExact.optional(),
    ulid: ulidFieldFilter.optional(),
    datetime: dateTimeFieldFilter.optional(),
    labels: labelsFieldFilter.optional(),
  })
  .describe('Field filters with all supported types.')

export const ingestedEvent = z
  .object({
    event: event,
    customer: customerReference.optional(),
    ingested_at: dateTime,
    stored_at: dateTime,
    validation_errors: z
      .array(ingestedEventValidationError)
      .optional()
      .describe('The validation errors of the ingested event.'),
  })
  .describe('An ingested metering event with ingestion metadata.')

export const meterQueryResult = z
  .object({
    from: dateTime.optional(),
    to: dateTime.optional(),
    data: z
      .array(meterQueryRow)

      .describe(
        'The usage data. If no data is available, an empty array is returned.',
      ),
  })
  .describe('Meter query result.')

export const featureCostQueryResult = z
  .object({
    from: dateTime.optional(),
    to: dateTime.optional(),
    data: z.array(featureCostQueryRow).describe('The cost data rows.'),
  })
  .describe('Result of a feature cost query.')

export const invalidParameter = z
  .union([
    invalidParameterStandard,
    invalidParameterMinimumLength,
    invalidParameterMaximumLength,
    invalidParameterChoiceItem,
    invalidParameterDependentItem,
  ])
  .describe('A parameter that failed validation.')

export const meterPagePaginatedResponse = z
  .object({
    data: z.array(meter),
    meta: paginatedMeta,
  })
  .describe('Page paginated response.')

export const costBasisPagePaginatedResponse = z
  .object({
    data: z.array(costBasis),
    meta: paginatedMeta,
  })
  .describe('Page paginated response.')

export const meterQueryFilters = z
  .object({
    dimensions: z
      .record(z.string(), queryFilterStringMapItem)
      .optional()

      .describe(
        'Filters to apply to the dimensions of the query. For `subject` and `customer_id` only equals ("eq", "in") comparisons are supported.',
      ),
  })
  .describe('Filters to apply to a meter query.')

export const featureMeterReference = z
  .object({
    id: ulid,
    filters: z
      .record(z.string(), queryFilterStringMapItem)
      .optional()
      .describe('Filters to apply to the dimensions of the meter.'),
  })
  .describe('Reference to a meter associated with a feature.')

export const createCustomerRequest = z
  .object({
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    key: externalResourceKey,
    usage_attribution: customerUsageAttribution.optional(),
    primary_email: z
      .string()
      .optional()
      .describe('The primary email address of the customer.'),
    currency: currencyCode.optional(),
    billing_address: billingAddress.optional(),
  })
  .describe('Customer create request.')

export const customer = z
  .object({
    id: ulid,
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    created_at: dateTime,
    updated_at: dateTime,
    deleted_at: dateTime.optional(),
    key: externalResourceKey,
    usage_attribution: customerUsageAttribution.optional(),
    primary_email: z
      .string()
      .optional()
      .describe('The primary email address of the customer.'),
    currency: currencyCode.optional(),
    billing_address: billingAddress.optional(),
  })

  .describe(
    'Customers can be individuals or organizations that can subscribe to plans and have access to features.',
  )

export const upsertCustomerRequest = z
  .object({
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    usage_attribution: customerUsageAttribution.optional(),
    primary_email: z
      .string()
      .optional()
      .describe('The primary email address of the customer.'),
    currency: currencyCode.optional(),
    billing_address: billingAddress.optional(),
  })
  .describe('Customer upsert request.')

export const partyAddresses = z
  .object({
    billing_address: address,
  })
  .describe('A collection of addresses for the party.')

export const appStripeCreateCheckoutSessionConsentCollection = z
  .object({
    payment_method_reuse_agreement:
      appStripeCreateCheckoutSessionConsentCollectionPaymentMethodReuseAgreement.optional(),
    promotions:
      appStripeCreateCheckoutSessionConsentCollectionPromotions.optional(),
    terms_of_service:
      appStripeCreateCheckoutSessionConsentCollectionTermsOfService.optional(),
  })
  .describe('Checkout Session consent collection configuration.')

export const listCustomerEntitlementAccessResponseData = z
  .object({
    data: z
      .array(entitlementAccessResult)
      .describe('The list of entitlement access results.'),
  })
  .describe('List customer entitlement access response data.')

export const workflowCollectionAlignmentAnchored = z
  .object({
    type: z.literal('anchored').describe('The type of alignment.'),
    recurring_period: recurringPeriod,
  })

  .describe(
    'BillingWorkflowCollectionAlignmentAnchored specifies the alignment for collecting the pending line items into an invoice.',
  )

export const rateCardEntitlement = z
  .discriminatedUnion('type', [
    rateCardMeteredEntitlement,
    rateCardStaticEntitlement,
    rateCardBooleanEntitlement,
  ])

  .describe(
    'Entitlement template configured on a rate card. The feature is taken from the rate card itself, so it is omitted here.',
  )

export const subscriptionPagePaginatedResponse = z
  .object({
    data: z.array(subscription),
    meta: paginatedMeta,
  })
  .describe('Page paginated response.')

export const subscriptionChangeResponse = z
  .object({
    current: subscription,
    next: subscription,
  })
  .describe('Response for changing a subscription.')

export const subscriptionCancel = z
  .object({
    timing: subscriptionEditTiming.optional().default('immediate'),
  })
  .describe('Request for canceling a subscription.')

export const subscriptionChange = z
  .object({
    labels: labels.optional(),
    settlement_mode: settlementMode.optional(),
    customer: z
      .object({
        id: ulid.optional(),
        key: externalResourceKey.optional(),
      })
      .describe('The customer to create the subscription for.'),
    plan: z
      .object({
        id: ulid.optional(),
        key: resourceKey.optional(),
        version: z
          .number()
          .int()
          .optional()

          .describe(
            'The plan version of the subscription, if any. If not provided, the latest version of the plan will be used.',
          ),
      })
      .describe('The plan reference of the subscription.'),
    billing_anchor: dateTime.optional(),
    timing: subscriptionEditTiming,
  })
  .describe('Request for changing a subscription.')

export const appStripe = z
  .object({
    id: ulid,
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    created_at: dateTime,
    updated_at: dateTime,
    deleted_at: dateTime.optional(),
    type: z.literal('stripe').describe('The app type.'),
    definition: appCatalogItem,
    status: appStatus,
    account_id: z
      .string()

      .describe(
        'The Stripe account ID associated with the connected Stripe account.',
      ),
    livemode: z
      .boolean()

      .describe(
        'Indicates whether the app is connected to a live Stripe account.',
      ),
    masked_api_key: z
      .string()

      .describe(
        'The masked Stripe API key that only exposes the first and last few characters.',
      ),
    secret_api_key: z
      .string()
      .optional()
      .describe('The Stripe secret API key used to authenticate API requests.'),
  })
  .describe('Stripe app.')

export const appSandbox = z
  .object({
    id: ulid,
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    created_at: dateTime,
    updated_at: dateTime,
    deleted_at: dateTime.optional(),
    type: z.literal('sandbox').describe('The app type.'),
    definition: appCatalogItem,
    status: appStatus,
  })
  .describe('Sandbox app can be used for testing billing features.')

export const appExternalInvoicing = z
  .object({
    id: ulid,
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    created_at: dateTime,
    updated_at: dateTime,
    deleted_at: dateTime.optional(),
    type: z.literal('external_invoicing').describe('The app type.'),
    definition: appCatalogItem,
    status: appStatus,
    enable_draft_sync_hook: z
      .boolean()

      .describe(
        'Enable draft synchronization hook. When enabled, invoices will pause at the draft state and wait for the integration to call the draft synchronized endpoint before progressing to the issuing state. This allows the external system to validate and prepare the invoice data. When disabled, invoices automatically progress through the draft state based on the configured workflow timing.',
      ),
    enable_issuing_sync_hook: z
      .boolean()

      .describe(
        'Enable issuing synchronization hook. When enabled, invoices will pause at the issuing state and wait for the integration to call the issuing synchronized endpoint before progressing to the issued state. This ensures the external invoicing system has successfully created and finalized the invoice before it is marked as issued. When disabled, invoices automatically progress through the issuing state and are immediately marked as issued.',
      ),
  })

  .describe(
    'External Invoicing app enables integration with third-party invoicing or payment system. The app supports a bi-directional synchronization pattern where OpenMeter Billing manages the invoice lifecycle while the external system handles invoice presentation and payment collection. Integration workflow: 1. The billing system creates invoices and transitions them through lifecycle states (draft → issuing → issued) 2. The integration receives webhook notifications about invoice state changes 3. The integration calls back to provide external system IDs and metadata 4. The integration reports payment events back via the payment status API State synchronization is controlled by hooks that pause invoice progression until the external system confirms synchronization via API callbacks.',
  )

export const createTaxCodeRequest = z
  .object({
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    key: resourceKey,
    app_mappings: z
      .array(taxCodeAppMapping)
      .describe('Mapping of app types to tax codes.'),
  })
  .describe('TaxCode create request.')

export const taxCode = z
  .object({
    id: ulid,
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    created_at: dateTime,
    updated_at: dateTime,
    deleted_at: dateTime.optional(),
    key: resourceKey,
    app_mappings: z
      .array(taxCodeAppMapping)
      .describe('Mapping of app types to tax codes.'),
  })
  .describe('Tax codes by provider.')

export const upsertTaxCodeRequest = z
  .object({
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    app_mappings: z
      .array(taxCodeAppMapping)
      .describe('Mapping of app types to tax codes.'),
  })
  .describe('TaxCode upsert request.')

export const currency = z
  .discriminatedUnion('type', [currencyFiat, currencyCustom])
  .describe('Fiat or custom currency.')

export const invoiceUsageQuantityDetail = z
  .object({
    raw_quantity: numeric,
    converted_quantity: numeric,
    invoiced_quantity: numeric,
    display_unit: z
      .string()
      .optional()

      .describe('The display unit label (e.g., "GB", "hours", "M tokens").'),
    applied_unit_config: unitConfig,
  })

  .describe(
    'Usage quantity details on an invoice line item when UnitConfig is in effect. Provides the full audit trail from raw meter output to the invoiced amount.',
  )

export const governanceFeatureAccess = z
  .object({
    has_access: z
      .boolean()

      .describe(
        'Whether the customer currently has access to the feature. `true` for boolean and static entitlements that are available, and for metered entitlements with remaining balance. `false` when the feature is unavailable, the usage limit has been reached, or (when applicable) credits have been exhausted.',
      ),
    reason: governanceFeatureAccessReason.optional(),
  })
  .describe('Access status for a single feature.')

export const customerData = z
  .object({
    billing_profile: profileReference.optional(),
    app_data: appCustomerData.optional(),
  })
  .describe('Billing customer data.')

export const upsertCustomerBillingDataRequest = z
  .object({
    billing_profile: profileReference.optional(),
    app_data: appCustomerData.optional(),
  })
  .describe('CustomerBillingData upsert request.')

export const creditBalances = z
  .object({
    retrieved_at: dateTime,
    balances: z.array(creditBalance).describe('The balances by currencies.'),
  })
  .describe('The balances of the credits of a customer.')

export const creditTransactionPaginatedResponse = z
  .object({
    data: z.array(creditTransaction),
    meta: cursorMeta,
  })
  .describe('Cursor paginated response.')

export const priceGraduated = z
  .object({
    type: z.literal('graduated').describe('The type of the price.'),
    tiers: z
      .array(priceTier)
      .min(1)

      .describe(
        'The tiers of the graduated price. At least one tier is required.',
      ),
  })

  .describe(
    "Graduated tiered price. Each tier's rate applies only to the usage within that tier. Pricing can change as cumulative usage crosses tier boundaries. When UnitConfig is present on the rate card, tier boundaries (up_to_amount) are expressed in converted billing units.",
  )

export const priceVolume = z
  .object({
    type: z.literal('volume').describe('The type of the price.'),
    tiers: z
      .array(priceTier)
      .min(1)

      .describe(
        'The tiers of the volume price. At least one tier is required.',
      ),
  })

  .describe(
    'Volume tiered price. The maximum quantity within a period determines the per-unit price for all units in that period. When UnitConfig is present on the rate card, tier boundaries (up_to_amount) are expressed in converted billing units.',
  )

export const featureUnitCost = z
  .discriminatedUnion('type', [featureManualUnitCost, featureLlmUnitCost])

  .describe(
    'Per-unit cost configuration for a feature. Either a fixed manual amount or a dynamic LLM cost lookup.',
  )

export const pricePagePaginatedResponse = z
  .object({
    data: z.array(llmCostPrice),
    meta: paginatedMeta,
  })
  .describe('Page paginated response.')

export const createCreditGrantRequest = z
  .object({
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: createLabels.optional(),
    funding_method: creditFundingMethod,
    currency: createCurrencyCode,
    amount: numeric,
    purchase: createCreditGrantPurchase.optional(),
    tax_config: createCreditGrantTaxConfig.optional(),
    filters: createCreditGrantFilters.optional(),
    priority: z
      .number()
      .int()
      .gte(1)
      .lte(1000)
      .optional()
      .default(10)

      .describe(
        'Draw-down priority of the grant. Lower values have higher priority.',
      ),
    expires_after: iso8601Duration.optional(),
  })
  .describe('CreditGrant create request.')

export const creditGrant = z
  .object({
    id: ulid,
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    created_at: dateTime,
    updated_at: dateTime,
    deleted_at: dateTime.optional(),
    funding_method: creditFundingMethod,
    currency: billingCurrencyCode,
    amount: numeric,
    purchase: creditGrantPurchase.optional(),
    tax_config: creditGrantTaxConfig.optional(),
    invoice: creditGrantInvoiceReference.optional(),
    filters: creditGrantFilters.optional(),
    priority: z
      .number()
      .int()
      .gte(1)
      .lte(1000)
      .optional()
      .default(10)

      .describe(
        'Draw-down priority of the grant. Lower values have higher priority.',
      ),
    expires_after: iso8601Duration.optional(),
    expires_at: dateTime.optional(),
    voided_at: dateTime.optional(),
    status: creditGrantStatus,
  })

  .describe(
    'A credit grant allocates credits to a customer. Credits are drawn down against charges according to the settlement mode configured on the rate card.',
  )

export const createFlatFeeChargeRequest = z
  .object({
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    type: z.literal('flat_fee').describe('The type of the charge.'),
    currency: currencyCode,
    invoice_at: dateTime,
    service_period: closedPeriod,
    unique_reference_id: z
      .string()
      .optional()
      .describe('Unique reference ID of the charge.'),
    settlement_mode: settlementMode,
    tax_config: taxConfig.optional(),
    payment_term: pricePaymentTerm,
    discounts: flatFeeDiscounts.optional(),
    feature_key: z
      .string()
      .optional()
      .describe('The feature associated with the charge, when applicable.'),
    proration_configuration: rateCardProrationConfiguration,
    amount_before_proration: currencyAmount,
    full_service_period: closedPeriod.optional(),
    billing_period: closedPeriod.optional(),
  })
  .describe('Flat fee charge create request.')

export const workflowTaxSettings = z
  .object({
    enabled: z
      .boolean()
      .optional()
      .default(true)

      .describe(
        'Enable automatic tax calculation when tax is supported by the app. For example, with Stripe Invoicing when enabled, tax is calculated via Stripe Tax.',
      ),
    enforced: z
      .boolean()
      .optional()
      .default(false)

      .describe(
        'Enforce tax calculation when tax is supported by the app. When enabled, the billing system will not allow to create an invoice without tax calculation. Enforcement is different per apps, for example, Stripe app requires customer to have a tax location when starting a paid subscription.',
      ),
    default_tax_config: taxConfig.optional(),
  })
  .describe('Tax settings for a billing workflow.')

export const subscriptionAddonPagePaginatedResponse = z
  .object({
    data: z.array(subscriptionAddon),
    meta: paginatedMeta,
  })
  .describe('Page paginated response.')

export const planAddonPagePaginatedResponse = z
  .object({
    data: z.array(planAddon),
    meta: paginatedMeta,
  })
  .describe('Page paginated response.')

export const ingestedEventPaginatedResponse = z
  .object({
    data: z.array(ingestedEvent),
    meta: cursorMeta,
  })
  .describe('Cursor paginated response.')

export const invalidParameters = z
  .array(invalidParameter)
  .min(1)
  .describe('The list of parameters that failed validation.')

export const meterQueryRequest = z
  .object({
    from: dateTime.optional(),
    to: dateTime.optional(),
    granularity: meterQueryGranularity.optional(),
    time_zone: z
      .string()
      .optional()
      .default('UTC')

      .describe(
        'The value is the name of the time zone as defined in the IANA Time Zone Database (http://www.iana.org/time-zones). The time zone is used to determine the start and end of the time buckets. If not specified, the UTC timezone will be used.',
      ),
    group_by_dimensions: z
      .array(z.string())
      .max(100)
      .optional()
      .describe('The dimensions to group the results by.'),
    filters: meterQueryFilters.optional(),
  })
  .describe('A meter query request.')

export const customerPagePaginatedResponse = z
  .object({
    data: z.array(customer),
    meta: paginatedMeta,
  })
  .describe('Page paginated response.')

export const party = z
  .object({
    id: z.string().optional().describe('Unique identifier for the party.'),
    key: externalResourceKey.optional(),
    name: z
      .string()
      .optional()
      .describe('Legal name or representation of the party.'),
    tax_id: partyTaxIdentity.optional(),
    addresses: partyAddresses.optional(),
  })
  .describe('Party represents a person or business entity.')

export const appStripeCreateCheckoutSessionRequestOptions = z
  .object({
    billing_address_collection:
      appStripeCreateCheckoutSessionBillingAddressCollection
        .optional()
        .default('auto'),
    cancel_url: z
      .string()
      .optional()

      .describe(
        'URL to redirect customers who cancel the checkout session. Not allowed when ui_mode is "embedded".',
      ),
    client_reference_id: z
      .string()
      .optional()

      .describe(
        'Unique reference string for reconciling sessions with internal systems. Can be a customer ID, cart ID, or any other identifier.',
      ),
    customer_update: appStripeCreateCheckoutSessionCustomerUpdate.optional(),
    consent_collection:
      appStripeCreateCheckoutSessionConsentCollection.optional(),
    currency: currencyCode.optional(),
    custom_text: appStripeCheckoutSessionCustomTextParams.optional(),
    expires_at: z.coerce
      .bigint()
      .gte(-9223372036854775808n)
      .lte(9223372036854775807n)
      .optional()

      .describe(
        'Unix timestamp when the checkout session expires. Can be 30 minutes to 24 hours from creation. Defaults to 24 hours.',
      ),
    locale: z
      .string()
      .optional()

      .describe(
        'IETF language tag for the checkout UI locale. If blank or "auto", uses the browser\'s locale. Example: "en", "fr", "de".',
      ),
    metadata: z
      .record(z.string(), z.string())
      .optional()

      .describe(
        'Set of key-value pairs to attach to the checkout session. Useful for storing additional structured information.',
      ),
    return_url: z
      .string()
      .optional()

      .describe(
        'Return URL for embedded checkout sessions after payment authentication. Required if ui_mode is "embedded" and redirect-based payment methods are enabled.',
      ),
    success_url: z
      .string()
      .optional()

      .describe(
        'Success URL to redirect customers after completing payment or setup. Not allowed when ui_mode is "embedded". See: https://docs.stripe.com/payments/checkout/custom-success-page',
      ),
    ui_mode: appStripeCheckoutSessionUiMode.optional().default('hosted'),
    payment_method_types: z
      .array(z.string())
      .optional()

      .describe(
        'List of payment method types to enable (e.g., "card", "us_bank_account"). If not specified, Stripe enables all relevant payment methods.',
      ),
    redirect_on_completion:
      appStripeCreateCheckoutSessionRedirectOnCompletion.optional(),
    tax_id_collection: appStripeCreateCheckoutSessionTaxIdCollection.optional(),
  })

  .describe(
    "Configuration options for creating a Stripe Checkout Session. Based on Stripe's [Checkout Session API parameters](https://docs.stripe.com/api/checkout/sessions/create).",
  )

export const workflowCollectionAlignment = z
  .discriminatedUnion('type', [
    workflowCollectionAlignmentSubscription,
    workflowCollectionAlignmentAnchored,
  ])

  .describe(
    'The alignment for collecting the pending line items into an invoice. Defaults to subscription, which means that we are to create a new invoice every time the a subscription period starts (for in advance items) or ends (for in arrears items).',
  )

export const app = z
  .discriminatedUnion('type', [appStripe, appSandbox, appExternalInvoicing])
  .describe('Installed application.')

export const taxCodePagePaginatedResponse = z
  .object({
    data: z.array(taxCode),
    meta: paginatedMeta,
  })
  .describe('Page paginated response.')

export const currencyPagePaginatedResponse = z
  .object({
    data: z.array(currency),
    meta: paginatedMeta,
  })
  .describe('Page paginated response.')

export const governanceQueryResult = z
  .object({
    matched: z
      .array(z.string())

      .describe(
        'The list of identifiers from the request that resolved to this customer. Each entry is either the customer `key` or one of its usage-attribution subject `key`s. Duplicate or aliased identifiers that resolve to the same customer collapse to a single result entry, with every requested identifier listed here.',
      ),
    customer: customer,
    features: z
      .record(z.string(), governanceFeatureAccess)

      .describe(
        'Map of features with their access status. Map keys are the feature keys requested in `feature.keys`, or every feature `key` available in the organization when the feature filter was omitted.',
      ),
    updated_at: dateTime,
  })
  .describe('Access evaluation result for a single resolved customer.')

export const price = z
  .discriminatedUnion('type', [
    priceFree,
    priceFlat,
    priceUnit,
    priceGraduated,
    priceVolume,
  ])
  .describe('Price.')

export const priceUsageBased = z
  .discriminatedUnion('type', [priceUnit, priceGraduated, priceVolume])

  .describe(
    'Usage-based price types that can appear on a usage-based rate card. When UnitConfig is present on the rate card, these price types operate on billing units (i.e. post-conversion quantities), not raw metered units.',
  )

export const feature = z
  .object({
    id: ulid,
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    created_at: dateTime,
    updated_at: dateTime,
    deleted_at: dateTime.optional(),
    key: resourceKey,
    meter: featureMeterReference.optional(),
    unit_cost: featureUnitCost.optional(),
  })
  .describe('A capability or billable dimension offered by a provider.')

export const createFeatureRequest = z
  .object({
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    key: resourceKey,
    meter: featureMeterReference.optional(),
    unit_cost: featureUnitCost.optional(),
  })
  .describe('Feature create request.')

export const updateFeatureRequest = z
  .object({
    unit_cost: z
      .union([featureUnitCost, z.null()])
      .optional()

      .describe(
        'Optional per-unit cost configuration. Use "manual" for a fixed per-unit cost, or "llm" to look up cost from the LLM cost database based on meter group-by properties. Set to `null` to clear the existing unit cost; omit to leave it unchanged.',
      ),
  })

  .describe(
    'Request body for updating a feature. Currently only the unit_cost field can be updated.',
  )

export const creditGrantPagePaginatedResponse = z
  .object({
    data: z.array(creditGrant),
    meta: paginatedMeta,
  })
  .describe('Page paginated response.')

export const badRequest = z
  .intersection(
    baseError,
    z.object({
      invalid_parameters: invalidParameters,
    }),
  )
  .describe('Bad Request.')

export const customerStripeCreateCheckoutSessionRequest = z
  .object({
    stripe_options: appStripeCreateCheckoutSessionRequestOptions,
  })

  .describe(
    'Request to create a Stripe Checkout Session for the customer. Checkout Sessions are used to collect payment method information from customers in a secure, Stripe-hosted interface. This integration uses setup mode to collect payment methods that can be charged later for subscription billing.',
  )

export const workflowCollectionSettings = z
  .object({
    alignment: workflowCollectionAlignment.optional().default({
      type: 'subscription',
    }),
    interval: z
      .string()
      .optional()
      .default('PT1H')

      .describe(
        'This grace period can be used to delay the collection of the pending line items specified in alignment. This is useful, in case of multiple subscriptions having slightly different billing periods.',
      ),
  })

  .describe(
    'Workflow collection specifies how to collect the pending line items for an invoice.',
  )

export const appPagePaginatedResponse = z
  .object({
    data: z.array(app),
    meta: paginatedMeta,
  })
  .describe('Page paginated response.')

export const profileApps = z
  .object({
    tax: app,
    invoicing: app,
    payment: app,
  })
  .describe('Applications used by a billing profile.')

export const governanceQueryResponse = z
  .object({
    data: z
      .array(governanceQueryResult)
      .describe('Access evaluation results, one entry per resolved customer.'),
    errors: z
      .array(governanceQueryError)
      .describe('Partial errors encountered while processing the request.'),
    meta: cursorMeta,
  })
  .describe('Response of the governance query.')

export const flatFeeCharge = z
  .object({
    id: ulid,
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    created_at: dateTime,
    updated_at: dateTime,
    deleted_at: dateTime.optional(),
    type: z.literal('flat_fee').describe('The type of the charge.'),
    customer: billingCustomerReference,
    managed_by: resourceManagedBy,
    subscription: subscriptionReference.optional(),
    currency: currencyCode,
    status: chargeStatus,
    invoice_at: dateTime,
    service_period: closedPeriod,
    full_service_period: closedPeriod,
    billing_period: closedPeriod,
    advance_after: dateTime.optional(),
    unique_reference_id: z
      .string()
      .optional()
      .describe('Unique reference ID of the charge.'),
    settlement_mode: settlementMode,
    tax_config: taxConfig.optional(),
    payment_term: pricePaymentTerm,
    discounts: flatFeeDiscounts.optional(),
    feature_key: z
      .string()
      .optional()
      .describe('The feature associated with the charge, when applicable.'),
    proration_configuration: rateCardProrationConfiguration,
    amount_before_proration: currencyAmount,
    amount_after_proration: currencyAmount,
    price: price,
  })
  .describe('A flat fee charge for a customer.')

export const usageBasedCharge = z
  .object({
    id: ulid,
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    created_at: dateTime,
    updated_at: dateTime,
    deleted_at: dateTime.optional(),
    type: z.literal('usage_based').describe('The type of the charge.'),
    customer: billingCustomerReference,
    managed_by: resourceManagedBy,
    subscription: subscriptionReference.optional(),
    currency: currencyCode,
    status: chargeStatus,
    invoice_at: dateTime,
    service_period: closedPeriod,
    full_service_period: closedPeriod,
    billing_period: closedPeriod,
    advance_after: dateTime.optional(),
    unique_reference_id: z
      .string()
      .optional()
      .describe('Unique reference ID of the charge.'),
    settlement_mode: settlementMode,
    tax_config: taxConfig.optional(),
    discounts: rateCardDiscounts.optional(),
    feature_key: z.string().describe('The feature associated with the charge.'),
    totals: chargeTotals,
    price: price,
  })
  .describe('A usage-based charge for a customer.')

export const createUsageBasedChargeRequest = z
  .object({
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    type: z.literal('usage_based').describe('The type of the charge.'),
    currency: currencyCode,
    invoice_at: dateTime,
    service_period: closedPeriod,
    unique_reference_id: z
      .string()
      .optional()
      .describe('Unique reference ID of the charge.'),
    settlement_mode: settlementMode,
    tax_config: taxConfig.optional(),
    discounts: rateCardDiscounts.optional(),
    feature_key: z.string().describe('The feature associated with the charge.'),
    price: price,
    full_service_period: closedPeriod.optional(),
    billing_period: closedPeriod.optional(),
  })
  .describe('Usage-based charge create request.')

export const rateCard = z
  .object({
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    key: resourceKey,
    feature: featureReference.optional(),
    billing_cadence: iso8601Duration.optional(),
    price: price,
    unit_config: unitConfig.optional(),
    payment_term: pricePaymentTerm.optional().default('in_arrears'),
    commitments: spendCommitments.optional(),
    discounts: rateCardDiscounts.optional(),
    tax_config: rateCardTaxConfig.optional(),
    entitlement: rateCardEntitlement.optional(),
  })

  .describe(
    'A rate card defines the pricing and entitlement of a feature or service.',
  )

export const featurePagePaginatedResponse = z
  .object({
    data: z.array(feature),
    meta: paginatedMeta,
  })
  .describe('Page paginated response.')

export const workflow = z
  .object({
    collection: workflowCollectionSettings.optional(),
    invoicing: workflowInvoicingSettings.optional(),
    payment: workflowPaymentSettings.optional(),
    tax: workflowTaxSettings.optional(),
  })
  .describe('Billing workflow settings.')

export const charge = z
  .discriminatedUnion('type', [flatFeeCharge, usageBasedCharge])
  .describe('Customer charge.')

export const createChargeRequest = z
  .discriminatedUnion('type', [
    createFlatFeeChargeRequest,
    createUsageBasedChargeRequest,
  ])
  .describe('Customer charge.')

export const planPhase = z
  .object({
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    key: resourceKey,
    duration: iso8601Duration.optional(),
    rate_cards: z.array(rateCard).describe('The rate cards of the plan.'),
  })

  .describe(
    "The plan phase or pricing ramp allows changing a plan's rate cards over time as a subscription progresses.",
  )

export const addon = z
  .object({
    id: ulid,
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    created_at: dateTime,
    updated_at: dateTime,
    deleted_at: dateTime.optional(),
    key: resourceKey,
    version: z
      .number()
      .int()
      .gte(1)
      .default(1)

      .describe(
        'Version of the add-on. Incremented when the add-on is updated.',
      ),
    instance_type: addonInstanceType,
    currency: billingCurrencyCode,
    effective_from: dateTime.optional(),
    effective_to: dateTime.optional(),
    status: addonStatus,
    rate_cards: z.array(rateCard).describe('The rate cards of the add-on.'),
    validation_errors: z
      .array(productCatalogValidationError)
      .optional()
      .describe('List of validation errors.'),
  })

  .describe(
    'Add-on allows extending subscriptions with compatible plans with additional ratecards.',
  )

export const createAddonRequest = z
  .object({
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    key: resourceKey,
    instance_type: addonInstanceType,
    currency: billingCurrencyCode,
    rate_cards: z.array(rateCard).describe('The rate cards of the add-on.'),
  })
  .describe('Addon create request.')

export const upsertAddonRequest = z
  .object({
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    instance_type: addonInstanceType,
    rate_cards: z.array(rateCard).describe('The rate cards of the add-on.'),
  })
  .describe('Addon upsert request.')

export const profile = z
  .object({
    id: ulid,
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    created_at: dateTime,
    updated_at: dateTime,
    deleted_at: dateTime.optional(),
    supplier: party,
    workflow: workflow,
    apps: profileAppReferences,
    default: z.boolean().describe('Whether this is the default profile.'),
  })

  .describe(
    'Billing profiles contain the settings for billing and controls invoice generation.',
  )

export const createBillingProfileRequest = z
  .object({
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    supplier: party,
    workflow: workflow,
    apps: profileAppReferences,
    default: z.boolean().describe('Whether this is the default profile.'),
  })
  .describe('BillingProfile create request.')

export const upsertBillingProfileRequest = z
  .object({
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    supplier: party,
    workflow: workflow,
    default: z.boolean().describe('Whether this is the default profile.'),
  })
  .describe('BillingProfile upsert request.')

export const chargePagePaginatedResponse = z
  .object({
    data: z.array(charge),
    meta: paginatedMeta,
  })
  .describe('Page paginated response.')

export const plan = z
  .object({
    id: ulid,
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    created_at: dateTime,
    updated_at: dateTime,
    deleted_at: dateTime.optional(),
    key: resourceKey,
    version: z
      .number()
      .int()
      .gte(1)
      .default(1)

      .describe(
        'Plans are versioned to allow you to make changes without affecting running subscriptions.',
      ),
    currency: currencyCode,
    billing_cadence: iso8601Duration,
    pro_rating_enabled: z
      .boolean()
      .optional()
      .default(true)
      .describe('Whether pro-rating is enabled for this plan.'),
    effective_from: dateTime.optional(),
    effective_to: dateTime.optional(),
    status: planStatus,
    phases: z
      .array(planPhase)
      .min(1)

      .describe(
        'The plan phases define the pricing ramp for a subscription. A phase switch occurs only at the end of a billing period. At least one phase is required.',
      ),
    settlement_mode: settlementMode.optional().default('credit_then_invoice'),
    validation_errors: z
      .array(productCatalogValidationError)
      .optional()

      .describe(
        'List of validation errors in `draft` state that prevent the plan from being published.',
      ),
  })
  .describe('Plans provide a template for subscriptions.')

export const createPlanRequest = z
  .object({
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    key: resourceKey,
    currency: currencyCode,
    billing_cadence: iso8601Duration,
    pro_rating_enabled: z
      .boolean()
      .optional()
      .default(true)
      .describe('Whether pro-rating is enabled for this plan.'),
    phases: z
      .array(planPhase)
      .min(1)

      .describe(
        'The plan phases define the pricing ramp for a subscription. A phase switch occurs only at the end of a billing period. At least one phase is required.',
      ),
  })
  .describe('Plan create request.')

export const upsertPlanRequest = z
  .object({
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    pro_rating_enabled: z
      .boolean()
      .optional()
      .default(true)
      .describe('Whether pro-rating is enabled for this plan.'),
    phases: z
      .array(planPhase)
      .min(1)

      .describe(
        'The plan phases define the pricing ramp for a subscription. A phase switch occurs only at the end of a billing period. At least one phase is required.',
      ),
  })
  .describe('Plan upsert request.')

export const addonPagePaginatedResponse = z
  .object({
    data: z.array(addon),
    meta: paginatedMeta,
  })
  .describe('Page paginated response.')

export const profilePagePaginatedResponse = z
  .object({
    data: z.array(profile),
    meta: paginatedMeta,
  })
  .describe('Page paginated response.')

export const planPagePaginatedResponse = z
  .object({
    data: z.array(plan),
    meta: paginatedMeta,
  })
  .describe('Page paginated response.')

export const listMeteringEventsQueryParams = z.object({
  page: cursorPaginationQueryPage.optional(),
  filter: listEventsParamsFilter.optional(),
  sort: sortQuery.optional(),
})

export const listMeteringEventsResponse = z.object({
  data: z.array(ingestedEvent),
  meta: cursorMeta,
})

export const ingestMeteringEventsBody = event

export const createMeterBody = createMeterRequest

export const createMeterResponse = meter

export const getMeterPathParams = z.object({
  meterId: ulid,
})

export const getMeterResponse = meter

export const listMetersQueryParams = z.object({
  page: z
    .object({
      size: z.coerce
        .number()
        .int()
        .optional()
        .describe('The number of items to include per page.'),
      number: z.coerce.number().int().optional().describe('The page number.'),
    })
    .optional()
    .describe('Determines which page of the collection to retrieve.'),
  sort: sortQuery.optional(),
  filter: listMetersParamsFilter.optional(),
})

export const listMetersResponse = z.object({
  data: z.array(meter),
  meta: paginatedMeta,
})

export const updateMeterPathParams = z.object({
  meterId: ulid,
})

export const updateMeterBody = updateMeterRequest

export const updateMeterResponse = meter

export const deleteMeterPathParams = z.object({
  meterId: ulid,
})

export const queryMeterPathParams = z.object({
  meterId: ulid,
})

export const queryMeterBody = meterQueryRequest

export const queryMeterResponse = meterQueryResult

export const queryMeterCsvPathParams = z.object({
  meterId: ulid,
})

export const queryMeterCsvResponse = z.string()

export const createCustomerBody = createCustomerRequest

export const createCustomerResponse = customer

export const getCustomerPathParams = z.object({
  customerId: ulid,
})

export const getCustomerResponse = customer

export const listCustomersQueryParams = z.object({
  page: z
    .object({
      size: z.coerce
        .number()
        .int()
        .optional()
        .describe('The number of items to include per page.'),
      number: z.coerce.number().int().optional().describe('The page number.'),
    })
    .optional()
    .describe('Determines which page of the collection to retrieve.'),
  sort: sortQuery.optional(),
  filter: listCustomersParamsFilter.optional(),
})

export const listCustomersResponse = z.object({
  data: z.array(customer),
  meta: paginatedMeta,
})

export const upsertCustomerPathParams = z.object({
  customerId: ulid,
})

export const upsertCustomerBody = upsertCustomerRequest

export const upsertCustomerResponse = customer

export const deleteCustomerPathParams = z.object({
  customerId: ulid,
})

export const getCustomerBillingPathParams = z.object({
  customerId: ulid,
})

export const getCustomerBillingResponse = customerData

export const updateCustomerBillingPathParams = z.object({
  customerId: ulid,
})

export const updateCustomerBillingBody = upsertCustomerBillingDataRequest

export const updateCustomerBillingResponse = customerData

export const updateCustomerBillingAppDataPathParams = z.object({
  customerId: ulid,
})

export const updateCustomerBillingAppDataBody = upsertAppCustomerDataRequest

export const updateCustomerBillingAppDataResponse = appCustomerData

export const createCustomerStripeCheckoutSessionPathParams = z.object({
  customerId: ulid,
})

export const createCustomerStripeCheckoutSessionBody =
  customerStripeCreateCheckoutSessionRequest

export const createCustomerStripeCheckoutSessionResponse =
  appStripeCreateCheckoutSessionResult

export const createCustomerStripePortalSessionPathParams = z.object({
  customerId: ulid,
})

export const createCustomerStripePortalSessionBody =
  customerStripeCreateCustomerPortalSessionRequest

export const createCustomerStripePortalSessionResponse =
  appStripeCreateCustomerPortalSessionResult

export const listCustomerEntitlementAccessPathParams = z.object({
  customerId: ulid,
})

export const listCustomerEntitlementAccessResponse =
  listCustomerEntitlementAccessResponseData

export const createCreditGrantPathParams = z.object({
  customerId: ulid,
})

export const createCreditGrantBody = createCreditGrantRequest

export const createCreditGrantResponse = creditGrant

export const getCreditGrantPathParams = z.object({
  customerId: ulid,
  creditGrantId: ulid,
})

export const getCreditGrantResponse = creditGrant

export const listCreditGrantsPathParams = z.object({
  customerId: ulid,
})

export const listCreditGrantsQueryParams = z.object({
  page: z
    .object({
      size: z.coerce
        .number()
        .int()
        .optional()
        .describe('The number of items to include per page.'),
      number: z.coerce.number().int().optional().describe('The page number.'),
    })
    .optional()
    .describe('Determines which page of the collection to retrieve.'),
  filter: listCreditGrantsParamsFilter.optional(),
})

export const listCreditGrantsResponse = z.object({
  data: z.array(creditGrant),
  meta: paginatedMeta,
})

export const getCustomerCreditBalancePathParams = z.object({
  customerId: ulid,
})

export const getCustomerCreditBalanceQueryParams = z.object({
  timestamp: dateTime.optional(),
  filter: getCreditBalanceParamsFilter.optional(),
})

export const getCustomerCreditBalanceResponse = creditBalances

export const createCreditAdjustmentPathParams = z.object({
  customerId: ulid,
})

export const createCreditAdjustmentBody = createCreditAdjustmentRequest

export const createCreditAdjustmentResponse = creditAdjustment

export const updateCreditGrantExternalSettlementPathParams = z.object({
  customerId: ulid,
  creditGrantId: ulid,
})

export const updateCreditGrantExternalSettlementBody =
  updateCreditGrantExternalSettlementRequest

export const updateCreditGrantExternalSettlementResponse = creditGrant

export const listCreditTransactionsPathParams = z.object({
  customerId: ulid,
})

export const listCreditTransactionsQueryParams = z.object({
  page: cursorPaginationQueryPage.optional(),
  filter: listCreditTransactionsParamsFilter.optional(),
})

export const listCreditTransactionsResponse = z.object({
  data: z.array(creditTransaction),
  meta: cursorMeta,
})

export const listCustomerChargesPathParams = z.object({
  customerId: ulid,
})

export const listCustomerChargesQueryParams = z.object({
  page: z
    .object({
      size: z.coerce
        .number()
        .int()
        .optional()
        .describe('The number of items to include per page.'),
      number: z.coerce.number().int().optional().describe('The page number.'),
    })
    .optional()
    .describe('Determines which page of the collection to retrieve.'),
  sort: sortQuery.optional(),
  filter: listChargesParamsFilter.optional(),
  expand: z
    .array(chargesExpand)
    .optional()

    .describe(
      "Expand full objects for referenced entities. Supported values are: - `real_time_usage`: Expand the charge's real-time usage.",
    ),
})

export const listCustomerChargesResponse = z.object({
  data: z.array(charge),
  meta: paginatedMeta,
})

export const createCustomerChargesPathParams = z.object({
  customerId: ulid,
})

export const createCustomerChargesBody = createChargeRequest

export const createCustomerChargesResponse = charge

export const createSubscriptionBody = subscriptionCreate

export const createSubscriptionResponse = subscription

export const listSubscriptionsQueryParams = z.object({
  page: z
    .object({
      size: z.coerce
        .number()
        .int()
        .optional()
        .describe('The number of items to include per page.'),
      number: z.coerce.number().int().optional().describe('The page number.'),
    })
    .optional()
    .describe('Determines which page of the collection to retrieve.'),
  sort: sortQuery.optional(),
  filter: listSubscriptionsParamsFilter.optional(),
})

export const listSubscriptionsResponse = z.object({
  data: z.array(subscription),
  meta: paginatedMeta,
})

export const getSubscriptionPathParams = z.object({
  subscriptionId: ulid,
})

export const getSubscriptionResponse = subscription

export const cancelSubscriptionPathParams = z.object({
  subscriptionId: ulid,
})

export const cancelSubscriptionBody = subscriptionCancel

export const cancelSubscriptionResponse = subscription

export const unscheduleCancelationPathParams = z.object({
  subscriptionId: ulid,
})

export const unscheduleCancelationResponse = subscription

export const changeSubscriptionPathParams = z.object({
  subscriptionId: ulid,
})

export const changeSubscriptionBody = subscriptionChange

export const changeSubscriptionResponse = subscriptionChangeResponse

export const listSubscriptionAddonsPathParams = z.object({
  subscriptionId: ulid,
})

export const listSubscriptionAddonsQueryParams = z.object({
  page: z
    .object({
      size: z.coerce
        .number()
        .int()
        .optional()
        .describe('The number of items to include per page.'),
      number: z.coerce.number().int().optional().describe('The page number.'),
    })
    .optional()
    .describe('Determines which page of the collection to retrieve.'),
  sort: sortQuery.optional(),
})

export const listSubscriptionAddonsResponse = z.object({
  data: z.array(subscriptionAddon),
  meta: paginatedMeta,
})

export const getSubscriptionAddonPathParams = z.object({
  subscriptionId: ulid,
  subscriptionAddonId: ulid,
})

export const getSubscriptionAddonResponse = subscriptionAddon

export const listAppsQueryParams = z.object({
  page: z
    .object({
      size: z.coerce
        .number()
        .int()
        .optional()
        .describe('The number of items to include per page.'),
      number: z.coerce.number().int().optional().describe('The page number.'),
    })
    .optional()
    .describe('Determines which page of the collection to retrieve.'),
  sort: sortQuery.optional(),
  filter: listAppsParamsFilter.optional(),
})

export const listAppsResponse = z.object({
  data: z.array(app),
  meta: paginatedMeta,
})

export const getAppPathParams = z.object({
  appId: ulid,
})

export const getAppResponse = app

export const listBillingProfilesQueryParams = z.object({
  page: z
    .object({
      size: z.coerce
        .number()
        .int()
        .optional()
        .describe('The number of items to include per page.'),
      number: z.coerce.number().int().optional().describe('The page number.'),
    })
    .optional()
    .describe('Determines which page of the collection to retrieve.'),
})

export const listBillingProfilesResponse = z.object({
  data: z.array(profile),
  meta: paginatedMeta,
})

export const createBillingProfileBody = createBillingProfileRequest

export const createBillingProfileResponse = profile

export const getBillingProfilePathParams = z.object({
  id: ulid,
})

export const getBillingProfileResponse = profile

export const updateBillingProfilePathParams = z.object({
  id: ulid,
})

export const updateBillingProfileBody = upsertBillingProfileRequest

export const updateBillingProfileResponse = profile

export const deleteBillingProfilePathParams = z.object({
  id: ulid,
})

export const createTaxCodeBody = createTaxCodeRequest

export const createTaxCodeResponse = taxCode

export const getTaxCodePathParams = z.object({
  taxCodeId: ulid,
})

export const getTaxCodeResponse = taxCode

export const listTaxCodesQueryParams = z.object({
  page: z
    .object({
      size: z.coerce
        .number()
        .int()
        .optional()
        .describe('The number of items to include per page.'),
      number: z.coerce.number().int().optional().describe('The page number.'),
    })
    .optional()
    .describe('Determines which page of the collection to retrieve.'),
  include_deleted: z.coerce
    .boolean()
    .optional()
    .describe('Include deleted tax codes in the response.'),
})

export const listTaxCodesResponse = z.object({
  data: z.array(taxCode),
  meta: paginatedMeta,
})

export const upsertTaxCodePathParams = z.object({
  taxCodeId: ulid,
})

export const upsertTaxCodeBody = upsertTaxCodeRequest

export const upsertTaxCodeResponse = taxCode

export const deleteTaxCodePathParams = z.object({
  taxCodeId: ulid,
})

export const listCurrenciesQueryParams = z.object({
  page: z
    .object({
      size: z.coerce
        .number()
        .int()
        .optional()
        .describe('The number of items to include per page.'),
      number: z.coerce.number().int().optional().describe('The page number.'),
    })
    .optional()
    .describe('Determines which page of the collection to retrieve.'),
  sort: sortQuery.optional(),
  filter: listCurrenciesParamsFilter.optional(),
})

export const listCurrenciesResponse = z.object({
  data: z.array(currency),
  meta: paginatedMeta,
})

export const createCustomCurrencyBody = createCurrencyCustomRequest

export const createCustomCurrencyResponse = currencyCustom

export const listCostBasesPathParams = z.object({
  currencyId: ulid,
})

export const listCostBasesQueryParams = z.object({
  filter: listCostBasesParamsFilter.optional(),
  page: z
    .object({
      size: z.coerce
        .number()
        .int()
        .optional()
        .describe('The number of items to include per page.'),
      number: z.coerce.number().int().optional().describe('The page number.'),
    })
    .optional()
    .describe('Determines which page of the collection to retrieve.'),
})

export const listCostBasesResponse = z.object({
  data: z.array(costBasis),
  meta: paginatedMeta,
})

export const createCostBasisPathParams = z.object({
  currencyId: ulid,
})

export const createCostBasisBody = createCostBasisRequest

export const createCostBasisResponse = costBasis

export const listFeaturesQueryParams = z.object({
  page: z
    .object({
      size: z.coerce
        .number()
        .int()
        .optional()
        .describe('The number of items to include per page.'),
      number: z.coerce.number().int().optional().describe('The page number.'),
    })
    .optional()
    .describe('Determines which page of the collection to retrieve.'),
  sort: sortQuery.optional(),
  filter: listFeatureParamsFilter.optional(),
})

export const listFeaturesResponse = z.object({
  data: z.array(feature),
  meta: paginatedMeta,
})

export const createFeatureBody = createFeatureRequest

export const createFeatureResponse = feature

export const getFeaturePathParams = z.object({
  featureId: ulid,
})

export const getFeatureResponse = feature

export const updateFeaturePathParams = z.object({
  featureId: ulid,
})

export const updateFeatureBody = updateFeatureRequest

export const updateFeatureResponse = feature

export const deleteFeaturePathParams = z.object({
  featureId: ulid,
})

export const queryFeatureCostPathParams = z.object({
  featureId: ulid,
})

export const queryFeatureCostBody = meterQueryRequest

export const queryFeatureCostResponse = featureCostQueryResult

export const listLlmCostPricesQueryParams = z.object({
  filter: listLlmCostPricesParamsFilter.optional(),
  sort: sortQuery.optional(),
  page: z
    .object({
      size: z.coerce
        .number()
        .int()
        .optional()
        .describe('The number of items to include per page.'),
      number: z.coerce.number().int().optional().describe('The page number.'),
    })
    .optional()
    .describe('Determines which page of the collection to retrieve.'),
})

export const listLlmCostPricesResponse = z.object({
  data: z.array(llmCostPrice),
  meta: paginatedMeta,
})

export const getLlmCostPricePathParams = z.object({
  priceId: ulid,
})

export const getLlmCostPriceResponse = llmCostPrice

export const listLlmCostOverridesQueryParams = z.object({
  filter: listLlmCostPricesParamsFilter.optional(),
  page: z
    .object({
      size: z.coerce
        .number()
        .int()
        .optional()
        .describe('The number of items to include per page.'),
      number: z.coerce.number().int().optional().describe('The page number.'),
    })
    .optional()
    .describe('Determines which page of the collection to retrieve.'),
})

export const listLlmCostOverridesResponse = z.object({
  data: z.array(llmCostPrice),
  meta: paginatedMeta,
})

export const createLlmCostOverrideBody = llmCostOverrideCreate

export const createLlmCostOverrideResponse = llmCostPrice

export const deleteLlmCostOverridePathParams = z.object({
  priceId: ulid,
})

export const listPlansQueryParams = z.object({
  page: z
    .object({
      size: z.coerce
        .number()
        .int()
        .optional()
        .describe('The number of items to include per page.'),
      number: z.coerce.number().int().optional().describe('The page number.'),
    })
    .optional()
    .describe('Determines which page of the collection to retrieve.'),
  sort: sortQuery.optional(),
  filter: listPlansParamsFilter.optional(),
})

export const listPlansResponse = z.object({
  data: z.array(plan),
  meta: paginatedMeta,
})

export const createPlanBody = createPlanRequest

export const createPlanResponse = plan

export const updatePlanPathParams = z.object({
  planId: ulid,
})

export const updatePlanBody = upsertPlanRequest

export const updatePlanResponse = plan

export const getPlanPathParams = z.object({
  planId: ulid,
})

export const getPlanResponse = plan

export const deletePlanPathParams = z.object({
  planId: ulid,
})

export const archivePlanPathParams = z.object({
  planId: ulid,
})

export const archivePlanResponse = plan

export const publishPlanPathParams = z.object({
  planId: ulid,
})

export const publishPlanResponse = plan

export const listAddonsQueryParams = z.object({
  page: z
    .object({
      size: z.coerce
        .number()
        .int()
        .optional()
        .describe('The number of items to include per page.'),
      number: z.coerce.number().int().optional().describe('The page number.'),
    })
    .optional()
    .describe('Determines which page of the collection to retrieve.'),
  sort: sortQuery.optional(),
  filter: listAddonsParamsFilter.optional(),
})

export const listAddonsResponse = z.object({
  data: z.array(addon),
  meta: paginatedMeta,
})

export const createAddonBody = createAddonRequest

export const createAddonResponse = addon

export const updateAddonPathParams = z.object({
  addonId: ulid,
})

export const updateAddonBody = upsertAddonRequest

export const updateAddonResponse = addon

export const getAddonPathParams = z.object({
  addonId: ulid,
})

export const getAddonResponse = addon

export const deleteAddonPathParams = z.object({
  addonId: ulid,
})

export const archiveAddonPathParams = z.object({
  addonId: ulid,
})

export const archiveAddonResponse = addon

export const publishAddonPathParams = z.object({
  addonId: ulid,
})

export const publishAddonResponse = addon

export const listPlanAddonsPathParams = z.object({
  planId: ulid,
})

export const listPlanAddonsQueryParams = z.object({
  page: z
    .object({
      size: z.coerce
        .number()
        .int()
        .optional()
        .describe('The number of items to include per page.'),
      number: z.coerce.number().int().optional().describe('The page number.'),
    })
    .optional()
    .describe('Determines which page of the collection to retrieve.'),
})

export const listPlanAddonsResponse = z.object({
  data: z.array(planAddon),
  meta: paginatedMeta,
})

export const createPlanAddonPathParams = z.object({
  planId: ulid,
})

export const createPlanAddonBody = createPlanAddonRequest

export const createPlanAddonResponse = planAddon

export const getPlanAddonPathParams = z.object({
  planId: ulid,
  planAddonId: ulid,
})

export const getPlanAddonResponse = planAddon

export const updatePlanAddonPathParams = z.object({
  planId: ulid,
  planAddonId: ulid,
})

export const updatePlanAddonBody = upsertPlanAddonRequest

export const updatePlanAddonResponse = planAddon

export const deletePlanAddonPathParams = z.object({
  planId: ulid,
  planAddonId: ulid,
})

export const getOrganizationDefaultTaxCodesResponse =
  organizationDefaultTaxCodes

export const updateOrganizationDefaultTaxCodesBody =
  updateOrganizationDefaultTaxCodesRequest

export const updateOrganizationDefaultTaxCodesResponse =
  organizationDefaultTaxCodes

export const queryGovernanceAccessQueryParams = z.object({
  page: cursorPaginationQueryPage.optional(),
})

export const queryGovernanceAccessBody = governanceQueryRequest

export const queryGovernanceAccessResponse = governanceQueryResponse
