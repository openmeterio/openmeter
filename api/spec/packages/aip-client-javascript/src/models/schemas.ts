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
    afterSubmit: z
      .object({
        message: z
          .string()
          .max(1200)
          .optional()
          .describe('The custom message text (max 1200 characters).'),
      })
      .optional()
      .describe('Text displayed after the payment confirmation button.'),
    shippingAddress: z
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
    termsOfServiceAcceptance: z
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
    configurationId: z
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
    returnUrl: z
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

export const lifecycleController = z
  .enum(['system', 'manual'])

  .describe(
    'Identifies whether a resource lifecycle is controlled by OpenMeter or manually overridden by the API user. Values: - `system`: The resource lifecycle is controlled by OpenMeter. - `manual`: The resource lifecycle was manually overridden by the API user.',
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

export const chargeFlatFeeDiscounts = z
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
    collectionMethod: z
      .literal('charge_automatically')
      .describe('The collection method for the invoice.'),
  })

  .describe(
    'Payment settings for a billing workflow when the collection method is charge automatically.',
  )

export const workflowPaymentSendInvoiceSettings = z
  .object({
    collectionMethod: z
      .literal('send_invoice')
      .describe('The collection method for the invoice.'),
    dueAfter: z
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

export const invoiceNumber = z
  .string()
  .min(1)
  .max(256)

  .describe(
    'InvoiceNumber is a unique identifier for the invoice, generated by the invoicing app. The uniqueness depends on a lot of factors: - app setting (unique per app or unique per customer) - multiple app scenarios (multiple apps generating invoices with the same prefix)',
  )

export const invoiceValidationIssueSeverity = z
  .enum(['critical', 'warning'])
  .describe('Severity level of an invoice validation issue.')

export const invoiceExternalReferences = z
  .object({
    invoicingId: z
      .string()
      .optional()

      .describe(
        'The ID assigned by the external invoicing app (e.g. Stripe invoice ID).',
      ),
    paymentId: z
      .string()
      .optional()

      .describe(
        'The ID assigned by the external payment app (e.g. Stripe payment intent ID).',
      ),
  })

  .describe(
    'External identifiers assigned to an invoice by third-party systems.',
  )

export const invoiceStandardStatus = z
  .enum([
    'draft',
    'issuing',
    'issued',
    'payment_processing',
    'overdue',
    'paid',
    'uncollectible',
    'voided',
  ])
  .describe('Lifecycle status of a standard invoice.')

export const invoiceAvailableActionDetails = z
  .object({
    resultingState: z
      .string()

      .describe(
        'The extended status the invoice will transition to after performing this action.',
      ),
  })

  .describe(
    'Details about an available invoice action including the resulting state.',
  )

export const invoiceWorkflowInvoicingSettings = z
  .object({
    autoAdvance: z
      .boolean()
      .optional()
      .default(true)

      .describe(
        'Whether to automatically issue the invoice after the draft_period has passed.',
      ),
    draftPeriod: z
      .string()
      .optional()
      .default('P0D')

      .describe(
        'The period for the invoice to be kept in draft status for manual reviews.',
      ),
  })

  .describe(
    'Invoice-level invoicing settings. A subset of BillingWorkflowInvoicingSettings limited to fields that are meaningful per-invoice. progressive_billing is omitted as it is a gather-time / profile-level decision.',
  )

export const invoiceDiscountReason = z
  .enum(['maximum_spend', 'ratecard_percentage', 'ratecard_usage'])
  .describe('The reason a discount was applied to an invoice line.')

export const invoiceLineExternalReferences = z
  .object({
    invoicingId: z
      .string()
      .optional()
      .describe('The ID assigned by the external invoicing app.'),
  })

  .describe(
    'External identifiers for an invoice line item assigned by third-party systems.',
  )

export const invoiceDetailedLineCostCategory = z
  .enum(['regular', 'commitment'])
  .describe('Cost category of a detailed invoice line item.')

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

export const invoiceType = z
  .enum(['standard'])
  .describe('The type of a billing invoice.')

export const invoiceLineType = z
  .enum(['standard_line'])
  .describe('Line item type discriminator.')

export const priceType = z
  .enum(['free', 'flat', 'unit', 'graduated', 'volume'])

  .describe(
    "The type of the price. - `free`: No charge, the rate card is included at no cost. - `flat`: A fixed amount charged once per billing period, regardless of usage. - `unit`: A fixed rate charged per billing unit consumed. - `graduated`: Tiered pricing where each tier's rate applies only to usage within that tier. - `volume`: Tiered pricing where the rate for the highest tier reached applies to all units in the period.",
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
    customerId: z.string().optional().describe('The Stripe customer ID used.'),
    defaultPaymentMethodId: z
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
    fiatCode: currencyCode.optional(),
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
    taxesTotal: numeric,
    taxesInclusiveTotal: numeric,
    taxesExclusiveTotal: numeric,
    chargesTotal: numeric,
    discountsTotal: numeric,
    creditsTotal: numeric,
    total: numeric,
  })

  .describe(
    'Totals contains the summaries of all calculations for a billing resource.',
  )

export const spendCommitments = z
  .object({
    minimumAmount: numeric.optional(),
    maximumAmount: numeric.optional(),
  })

  .describe(
    'Spend commitments for a rate card. The customer is committed to spend at least the minimum amount and at most the maximum amount.',
  )

export const invoiceLineCreditsApplied = z
  .object({
    amount: numeric,
    description: z
      .string()
      .optional()

      .describe(
        'Optional human-readable description of the credit allocation.',
      ),
  })
  .describe('A credit allocation applied to an invoice line item.')

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
    inputPerToken: numeric,
    outputPerToken: numeric,
    cacheReadPerToken: numeric.optional(),
    reasoningPerToken: numeric.optional(),
    cacheWritePerToken: numeric.optional(),
  })
  .describe('Resolved per-token pricing from the LLM cost database.')

export const llmCostModelPricing = z
  .object({
    inputPerToken: numeric,
    outputPerToken: numeric,
    cacheReadPerToken: numeric.optional(),
    cacheWritePerToken: numeric.optional(),
    reasoningPerToken: numeric.optional(),
  })
  .describe('Token pricing for an LLM model, denominated per token.')

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
    modelId: stringFieldFilter.optional(),
    modelName: stringFieldFilter.optional(),
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

export const featureReference = z
  .object({
    id: ulid,
  })
  .describe('Feature reference.')

export const appReference = z
  .object({
    id: ulid,
  })
  .describe('App reference.')

export const chargeReference = z
  .object({
    id: ulid,
  })
  .describe('Reference to a charge associated with an invoice line.')

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
    stripeCustomerId: z.string().describe('The ID of the stripe customer.'),
    configurationId: z
      .string()

      .describe(
        'Configuration used to customize the customer portal. See: https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-configuration',
      ),
    livemode: z
      .boolean()

      .describe(
        'Livemode. See: https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-livemode',
      ),
    createdAt: dateTime,
    returnUrl: z
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

export const subscriptionAddonTimelineSegment = z
  .object({
    activeFrom: dateTime,
    activeTo: dateTime.optional(),
    quantity: z
      .number()
      .int()
      .nonnegative()
      .describe('The quantity of the add-on for the given period.'),
  })
  .describe('A subscription add-on event.')

export const costBasis = z
  .object({
    id: ulid,
    fiatCode: currencyCode,
    rate: numeric,
    effectiveFrom: dateTime.optional(),
    createdAt: dateTime,
  })
  .describe('Describes currency basis supported by billing system.')

export const createCostBasisRequest = z
  .object({
    fiatCode: currencyCode,
    rate: numeric,
    effectiveFrom: dateTime.optional(),
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
    createdAt: dateTime,
    updatedAt: dateTime,
    deletedAt: dateTime.optional(),
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
    createdAt: dateTime,
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
    fromPlanPhase: resourceKey,
    maxQuantity: z
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
    createdAt: dateTime,
    updatedAt: dateTime,
    deletedAt: dateTime.optional(),
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
    eventType: z
      .string()
      .min(1)
      .describe('The event type to include in the aggregation.'),
    eventsFrom: dateTime.optional(),
    valueProperty: z
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
    createdAt: dateTime,
    updatedAt: dateTime,
    deletedAt: dateTime.optional(),
    key: resourceKey,
    aggregation: meterAggregation,
    eventType: z
      .string()
      .min(1)
      .describe('The event type to include in the aggregation.'),
    eventsFrom: dateTime.optional(),
    valueProperty: z
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
    subjectKeys: z
      .array(usageAttributionSubjectKey)

      .describe(
        'The subjects that are attributed to the customer. Can be empty when no usage event subjects are associated with the customer.',
      ),
  })

  .describe(
    'Mapping to attribute metered usage to the customer. One customer can have zero or more subjects, but one subject can only belong to one customer.',
  )

export const address = z
  .object({
    country: countryCode.optional(),
    postalCode: z.string().optional().describe('Postal code.'),
    state: z.string().optional().describe('State or province.'),
    city: z.string().optional().describe('City.'),
    line1: z.string().optional().describe('First line of the address.'),
    line2: z.string().optional().describe('Second line of the address.'),
    phoneNumber: z.string().optional().describe('Phone number.'),
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
    customerId: ulid,
    stripeCustomerId: z.string().describe('The Stripe customer ID.'),
    sessionId: z.string().describe('The Stripe checkout session ID.'),
    setupIntentId: z
      .string()

      .describe(
        'The setup intent ID created for collecting the payment method.',
      ),
    clientSecret: z
      .string()
      .optional()

      .describe(
        'Client secret for initializing Stripe.js on the client side. Required for embedded checkout sessions. See: https://docs.stripe.com/payments/checkout/custom-success-page',
      ),
    clientReferenceId: z
      .string()
      .optional()

      .describe(
        'The client reference ID provided in the request. Useful for reconciling the session with your internal systems.',
      ),
    customerEmail: z
      .string()
      .optional()
      .describe("Customer's email address if provided to Stripe."),
    currency: currencyCode.optional(),
    createdAt: dateTime,
    expiresAt: dateTime.optional(),
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
    cancelUrl: z
      .string()
      .optional()

      .describe(
        'The cancel URL where customers are redirected if they cancel.',
      ),
    successUrl: z
      .string()
      .optional()

      .describe(
        'The success URL where customers are redirected after completion.',
      ),
    returnUrl: z
      .string()
      .optional()
      .describe('The return URL for embedded sessions after authentication.'),
  })

  .describe(
    'Result of creating a Stripe Checkout Session. Contains all the information needed to redirect customers to the checkout or initialize an embedded checkout flow.',
  )

export const customerStripeCreateCustomerPortalSessionRequest = z
  .object({
    stripeOptions: appStripeCreateCustomerPortalSessionOptions,
  })

  .describe(
    'Request to create a Stripe Customer Portal Session for the customer. Useful to redirect the customer to the Stripe Customer Portal to manage their payment methods, change their billing address and access their invoice history. Only returns URL if the customer billing profile is linked to a stripe app and customer.',
  )

export const entitlementAccessResult = z
  .object({
    type: entitlementType,
    featureKey: resourceKey,
    hasAccess: z
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
    perUnitCostBasis: numeric.optional().default('1.0'),
    availabilityPolicy: creditAvailabilityPolicy
      .optional()
      .default('on_creation'),
  })
  .describe('Purchase and payment terms of the grant.')

export const rateCardMeteredEntitlement = z
  .object({
    type: z
      .literal('metered')
      .describe('The type of the entitlement template.'),
    isSoftLimit: z
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
    usagePeriod: iso8601Duration.optional(),
  })
  .describe('The entitlement template of a metered entitlement.')

export const recurringPeriod = z
  .object({
    anchor: dateTime,
    interval: iso8601Duration,
  })
  .describe('Recurring period with an anchor and an interval.')

export const creditGrantPurchase = z
  .object({
    currency: currencyCode,
    perUnitCostBasis: numeric.optional().default('1.0'),
    amount: numeric,
    availabilityPolicy: creditAvailabilityPolicy
      .optional()
      .default('on_creation'),
    settlementStatus: creditPurchasePaymentSettlementStatus.optional(),
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
    key: stringFieldFilter.optional(),
  })
  .describe('Filter options for listing credit grants.')

export const getCreditBalanceParamsFilter = z
  .object({
    currency: stringFieldFilterExact.optional(),
    featureKey: stringFieldFilter.optional(),
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
    settlementMode: settlementMode.optional(),
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
    billingAnchor: dateTime.optional(),
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
    createdAt: dateTime,
    updatedAt: dateTime,
    deletedAt: dateTime.optional(),
    customerId: ulid,
    planId: ulid.optional(),
    billingAnchor: dateTime,
    status: subscriptionStatus,
    settlementMode: settlementMode.optional(),
  })
  .describe('Subscription.')

export const subscriptionEditTiming = z
  .union([subscriptionEditTimingEnum, dateTime])

  .describe(
    'Subscription edit timing defined when the changes should take effect. If the provided configuration is not supported by the subscription, an error will be returned.',
  )

export const unitConfig = z
  .object({
    operation: unitConfigOperation,
    conversionFactor: numeric,
    rounding: unitConfigRoundingMode.optional().default('none'),
    precision: z
      .number()
      .int()
      .optional()
      .default(0)

      .describe(
        'The number of decimal places to retain after rounding. Only meaningful when rounding is not "none". Defaults to 0 (round to whole numbers).',
      ),
    displayUnit: z
      .string()
      .optional()

      .describe(
        'A human-readable label for the converted unit shown on invoices and in the customer portal (e.g., "GB", "hours", "M tokens"). Optional. When omitted, no unit label is rendered.',
      ),
  })

  .describe(
    'Unit conversion configuration. Transforms raw metered quantities into billing-ready units before pricing and entitlement evaluation. Applied at the rate card level so the same feature can be billed in different units across plans. Examples: - Meter bytes, bill GB: operation=divide, conversionFactor=1e9, rounding=ceiling, displayUnit="GB" - Meter seconds, bill hours: operation=divide, conversionFactor=3600, rounding=ceiling, displayUnit="hours" - Cost + 20% margin: operation=multiply, conversionFactor=1.2 - Bill per million tokens: operation=divide, conversionFactor=1e6, rounding=ceiling, displayUnit="M" v1 equivalents: - DynamicPrice(multiplier): operation=multiply, conversionFactor=multiplier + UnitPrice(amount=1) - PackagePrice(amount, quantityPerPkg): operation=divide, conversionFactor=quantityPerPkg, rounding=ceiling + UnitPrice(amount)',
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
    appType: appType,
    taxCode: z.string().describe('Tax code.'),
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
    autoAdvance: z
      .boolean()
      .optional()
      .default(true)

      .describe(
        'Whether to automatically issue the invoice after the draftPeriod has passed.',
      ),
    draftPeriod: z
      .string()
      .optional()
      .default('P0D')

      .describe(
        'The period for the invoice to be kept in draft status for manual reviews.',
      ),
    progressiveBilling: z
      .boolean()
      .optional()
      .default(true)
      .describe('Should progressive billing be allowed for this workflow?'),
    subscriptionEndProrationMode: workflowInvoicingSubscriptionEndProrationMode
      .optional()
      .default('bill_actual_period'),
  })
  .describe('Invoice settings for a billing workflow.')

export const workflowPaymentSettings = z
  .discriminatedUnion('collectionMethod', [
    workflowPaymentChargeAutomaticallySettings,
    workflowPaymentSendInvoiceSettings,
  ])
  .describe('Payment settings for a billing workflow.')

export const invoiceValidationIssue = z
  .object({
    code: z.string().describe('Machine-readable error code.'),
    message: z.string().describe('Human-readable description of the error.'),
    attributes: z
      .record(z.string(), z.unknown())
      .optional()
      .describe('Additional structured context.'),
    severity: invoiceValidationIssueSeverity,
    field: z
      .string()
      .optional()

      .describe(
        'JSON path to the field that caused this validation issue, if applicable. For example: `lines/0/rate_card/price`.',
      ),
  })

  .describe(
    'A validation issue found during invoice processing. Converges on the same structure used by plan and subscription validation errors: a machine-readable `code`, a human-readable `message`, optional structured `attributes`, plus a `severity` and optional `field` path.',
  )

export const invoiceAvailableActions = z
  .object({
    advance: invoiceAvailableActionDetails.optional(),
    approve: invoiceAvailableActionDetails.optional(),
    delete: invoiceAvailableActionDetails.optional(),
    retry: invoiceAvailableActionDetails.optional(),
    snapshotQuantities: invoiceAvailableActionDetails.optional(),
  })

  .describe(
    'The set of state-transition actions available for an invoice in its current status. A field is present only when that action is permitted from the current state.',
  )

export const invoiceLineAmountDiscount = z
  .object({
    id: ulid,
    reason: invoiceDiscountReason,
    description: z
      .string()
      .optional()
      .describe('Optional human-readable description of the discount.'),
    externalReferences: invoiceLineExternalReferences.optional(),
    amount: numeric,
  })
  .describe('A monetary amount discount applied to an invoice line item.')

export const invoiceLineUsageDiscount = z
  .object({
    id: ulid,
    reason: invoiceDiscountReason,
    description: z
      .string()
      .optional()
      .describe('Optional human-readable description of the discount.'),
    externalReferences: invoiceLineExternalReferences.optional(),
    quantity: numeric,
  })
  .describe('A usage quantity discount applied to an invoice line item.')

export const invoiceLineBaseDiscount = z
  .object({
    id: ulid,
    reason: invoiceDiscountReason,
    description: z
      .string()
      .optional()
      .describe('Optional human-readable description of the discount.'),
    externalReferences: invoiceLineExternalReferences.optional(),
  })
  .describe('Base fields shared by all invoice line item discounts.')

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
    createdAt: dateTime,
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

export const governanceQueryRequest = z
  .object({
    includeCredits: z
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
    externalInvoicing: appCustomerDataExternalInvoicing.optional(),
  })
  .describe('App customer data.')

export const upsertAppCustomerDataRequest = z
  .object({
    stripe: appCustomerDataStripe.optional(),
    externalInvoicing: appCustomerDataExternalInvoicing.optional(),
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
  })

  .describe(
    "A credit adjustment can be used to make manual adjustments to a customer's credit balance. Supported use-cases: - Usage correction",
  )

export const creditBalance = z
  .object({
    currency: billingCurrencyCode,
    live: numeric,
    settled: numeric,
    pending: numeric,
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
    featureKey: stringFieldFilter.optional(),
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
    createdAt: dateTime,
    bookedAt: dateTime,
    type: creditTransactionType,
    currency: billingCurrencyCode,
    amount: numeric,
    availableBalance: z
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
    upToAmount: numeric.optional(),
    flatPrice: priceFlat.optional(),
    unitPrice: priceUnit.optional(),
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
    providerProperty: z
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
    modelProperty: z
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
    tokenTypeProperty: z
      .string()
      .optional()

      .describe(
        'Meter group-by property that holds the token type. Use this when the meter has a group-by dimension for token type. Mutually exclusive with `token_type`.',
      ),
    tokenType: featureLlmTokenType.optional(),
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
    effectiveFrom: dateTime,
    effectiveTo: dateTime.optional(),
    createdAt: dateTime,
    updatedAt: dateTime,
  })

  .describe(
    'An LLM cost price record, representing the cost per token for a specific model from a specific provider.',
  )

export const llmCostOverrideCreate = z
  .object({
    provider: z.string().describe('Provider/vendor of the model.'),
    modelId: z.string().describe('Canonical model identifier.'),
    modelName: z.string().optional().describe('Human-readable model name.'),
    pricing: llmCostModelPricing,
    currency: currencyCode,
    effectiveFrom: dateTime,
    effectiveTo: dateTime.optional(),
  })

  .describe(
    'Input for creating a per-namespace price override. Unique per provider, model and currency. If an override already exists for the given provider, model and currency, it will be updated. If an override does not exist, it will be created.',
  )

export const listCustomersParamsFilter = z
  .object({
    key: stringFieldFilter.optional(),
    name: stringFieldFilter.optional(),
    primaryEmail: stringFieldFilter.optional(),
    usageAttributionSubjectKey: stringFieldFilter.optional(),
    planKey: stringFieldFilter.optional(),
    billingProfileId: ulidFieldFilter.optional(),
  })
  .describe('Filter options for listing customers.')

export const listSubscriptionsParamsFilter = z
  .object({
    id: ulidFieldFilter.optional(),
    customerId: ulidFieldFilter.optional(),
    status: stringFieldFilterExact.optional(),
    planId: ulidFieldFilter.optional(),
    planKey: stringFieldFilterExact.optional(),
  })
  .describe('Filter options for listing subscriptions.')

export const listFeatureParamsFilter = z
  .object({
    meterId: ulidFieldFilter.optional(),
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
    taxCode: createResourceReference.optional(),
  })

  .describe(
    'Tax configuration for a credit grant. Tax configuration should be provided to ensure correct revenue recognition, including for externally funded grants.',
  )

export const creditGrantTaxConfig = z
  .object({
    behavior: taxBehavior.optional(),
    taxCode: taxCodeReference.optional(),
  })

  .describe(
    'Tax configuration for a credit grant. Tax configuration should be provided to ensure correct revenue recognition, including for externally funded grants.',
  )

export const taxConfig = z
  .object({
    behavior: taxBehavior.optional(),
    stripe: taxConfigStripe.optional(),
    externalInvoicing: taxConfigExternalInvoicing.optional(),
    taxCodeId: ulid.optional(),
    taxCode: taxCodeReference.optional(),
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
    invoicingTaxCode: taxCodeReference,
    creditGrantTaxCode: taxCodeReference,
    createdAt: dateTime,
    updatedAt: dateTime,
  })

  .describe(
    'Organization-level default tax code references. Stores the default tax codes applied to specific billing contexts for this organization. Provisioned automatically when the organization is created.',
  )

export const updateOrganizationDefaultTaxCodesRequest = z
  .object({
    invoicingTaxCode: taxCodeReference.optional(),
    creditGrantTaxCode: taxCodeReference.optional(),
  })
  .describe('OrganizationDefaultTaxCodes update request.')

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
    createdAt: dateTime,
    updatedAt: dateTime,
    deletedAt: dateTime.optional(),
    addon: addonReference,
    fromPlanPhase: resourceKey,
    maxQuantity: z
      .number()
      .int()
      .gte(1)
      .optional()

      .describe(
        'The maximum number of times the add-on can be purchased for the plan. For single-instance add-ons this field must be omitted. For multi-instance add-ons when omitted, unlimited quantity can be purchased.',
      ),
    validationErrors: z
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
    fromPlanPhase: resourceKey,
    maxQuantity: z
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

export const invoiceWorkflowAppsReferences = z
  .object({
    tax: appReference,
    invoicing: appReference,
    payment: appReference,
  })

  .describe(
    'BillingInvoiceWorkflowAppsReferences represents the references (id) to the apps used by a billing profile',
  )

export const listEventsParamsFilter = z
  .object({
    id: stringFieldFilter.optional(),
    source: stringFieldFilter.optional(),
    subject: stringFieldFilter.optional(),
    type: stringFieldFilter.optional(),
    customerId: ulidFieldFilter.optional(),
    time: dateTimeFieldFilter.optional(),
    ingestedAt: dateTimeFieldFilter.optional(),
    storedAt: dateTimeFieldFilter.optional(),
  })
  .describe('Filter options for listing ingested events.')

export const resourceFilters = z
  .object({
    name: stringFieldFilter.optional(),
    labels: labelsFieldFilter.optional(),
    publicLabels: labelsFieldFilter.optional(),
    createdAt: dateTimeFieldFilter.optional(),
    updatedAt: dateTimeFieldFilter.optional(),
    deletedAt: dateTimeFieldFilter.optional(),
  })
  .describe('Resource filters.')

export const fieldFilters = z
  .object({
    boolean: booleanFieldFilter.optional(),
    numeric: numericFieldFilter.optional(),
    string: stringFieldFilter.optional(),
    stringExact: stringFieldFilterExact.optional(),
    ulid: ulidFieldFilter.optional(),
    datetime: dateTimeFieldFilter.optional(),
    labels: labelsFieldFilter.optional(),
  })
  .describe('Field filters with all supported types.')

export const ingestedEvent = z
  .object({
    event: event,
    customer: customerReference.optional(),
    ingestedAt: dateTime,
    storedAt: dateTime,
    validationErrors: z
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
    usageAttribution: customerUsageAttribution.optional(),
    primaryEmail: z
      .string()
      .optional()
      .describe('The primary email address of the customer.'),
    currency: currencyCode.optional(),
    billingAddress: address.optional(),
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
    createdAt: dateTime,
    updatedAt: dateTime,
    deletedAt: dateTime.optional(),
    key: externalResourceKey,
    usageAttribution: customerUsageAttribution.optional(),
    primaryEmail: z
      .string()
      .optional()
      .describe('The primary email address of the customer.'),
    currency: currencyCode.optional(),
    billingAddress: address.optional(),
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
    usageAttribution: customerUsageAttribution.optional(),
    primaryEmail: z
      .string()
      .optional()
      .describe('The primary email address of the customer.'),
    currency: currencyCode.optional(),
    billingAddress: address.optional(),
  })
  .describe('Customer upsert request.')

export const partyAddresses = z
  .object({
    billingAddress: address,
  })
  .describe('A collection of addresses for the party.')

export const invoiceCustomer = z
  .object({
    id: ulid,
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    usageAttribution: customerUsageAttribution.optional(),
    billingAddress: address.optional(),
    key: externalResourceKey.optional(),
  })

  .describe(
    "Snapshot of the customer's information at the time the invoice was issued.",
  )

export const appStripeCreateCheckoutSessionConsentCollection = z
  .object({
    paymentMethodReuseAgreement:
      appStripeCreateCheckoutSessionConsentCollectionPaymentMethodReuseAgreement.optional(),
    promotions:
      appStripeCreateCheckoutSessionConsentCollectionPromotions.optional(),
    termsOfService:
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

export const rateCardEntitlement = z
  .discriminatedUnion('type', [
    rateCardMeteredEntitlement,
    rateCardStaticEntitlement,
    rateCardBooleanEntitlement,
  ])

  .describe(
    'Entitlement template configured on a rate card. The feature is taken from the rate card itself, so it is omitted here.',
  )

export const workflowCollectionAlignmentAnchored = z
  .object({
    type: z.literal('anchored').describe('The type of alignment.'),
    recurringPeriod: recurringPeriod,
  })

  .describe(
    'BillingWorkflowCollectionAlignmentAnchored specifies the alignment for collecting the pending line items into an invoice.',
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
    settlementMode: settlementMode.optional(),
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
    billingAnchor: dateTime.optional(),
    timing: subscriptionEditTiming,
  })
  .describe('Request for changing a subscription.')

export const createSubscriptionAddonRequest = z
  .object({
    labels: labels.optional(),
    addon: addonReference,
    quantity: z
      .number()
      .int()
      .gte(1)

      .describe(
        'The quantity of the add-on. Always 1 for single instance add-ons.',
      ),
    timing: subscriptionEditTiming,
  })
  .describe('SubscriptionAddon create request.')

export const invoiceUsageQuantityDetail = z
  .object({
    rawQuantity: numeric,
    convertedQuantity: numeric,
    invoicedQuantity: numeric,
    displayUnit: z
      .string()
      .optional()

      .describe('The display unit label (e.g., "GB", "hours", "M tokens").'),
    appliedUnitConfig: unitConfig,
  })

  .describe(
    'Usage quantity details on an invoice line item when UnitConfig is in effect. Provides the full audit trail from raw meter output to the invoiced amount.',
  )

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
    createdAt: dateTime,
    updatedAt: dateTime,
    deletedAt: dateTime.optional(),
    type: z.literal('stripe').describe('The app type.'),
    definition: appCatalogItem,
    status: appStatus,
    accountId: z
      .string()

      .describe(
        'The Stripe account ID associated with the connected Stripe account.',
      ),
    livemode: z
      .boolean()

      .describe(
        'Indicates whether the app is connected to a live Stripe account.',
      ),
    maskedApiKey: z
      .string()

      .describe(
        'The masked Stripe API key that only exposes the first and last few characters.',
      ),
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
    createdAt: dateTime,
    updatedAt: dateTime,
    deletedAt: dateTime.optional(),
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
    createdAt: dateTime,
    updatedAt: dateTime,
    deletedAt: dateTime.optional(),
    type: z.literal('external_invoicing').describe('The app type.'),
    definition: appCatalogItem,
    status: appStatus,
    enableDraftSyncHook: z
      .boolean()

      .describe(
        'Enable draft synchronization hook. When enabled, invoices will pause at the draft state and wait for the integration to call the draft synchronized endpoint before progressing to the issuing state. This allows the external system to validate and prepare the invoice data. When disabled, invoices automatically progress through the draft state based on the configured workflow timing.',
      ),
    enableIssuingSyncHook: z
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
    appMappings: z
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
    createdAt: dateTime,
    updatedAt: dateTime,
    deletedAt: dateTime.optional(),
    key: resourceKey,
    appMappings: z
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
    appMappings: z
      .array(taxCodeAppMapping)
      .describe('Mapping of app types to tax codes.'),
  })
  .describe('TaxCode upsert request.')

export const invoiceWorkflow = z
  .object({
    invoicing: invoiceWorkflowInvoicingSettings.optional(),
    payment: workflowPaymentSettings.optional(),
  })

  .describe(
    'Invoice-level snapshot of the workflow configuration. Contains only the settings that are meaningful for an already-created invoice: invoicing behaviour and payment settings. Collection alignment and tax policy are gather-time / profile-wide concerns and are not included.',
  )

export const invoiceStatusDetails = z
  .object({
    immutable: z
      .boolean()

      .describe(
        'Whether the invoice is immutable (i.e. cannot be modified or deleted).',
      ),
    failed: z.boolean().describe('Whether the invoice is in a failed state.'),
    extendedStatus: z
      .string()

      .describe(
        'Fine-grained internal status string providing additional workflow detail beyond the top-level status enum.',
      ),
    availableActions: invoiceAvailableActions,
  })
  .describe('Detailed status information for a standard invoice.')

export const invoiceLineDiscounts = z
  .object({
    amount: z
      .array(invoiceLineAmountDiscount)
      .optional()

      .describe(
        'Monetary amount discounts (e.g. from maximum spend commitments).',
      ),
    usage: z
      .array(invoiceLineUsageDiscount)
      .optional()
      .describe('Usage quantity discounts (e.g. free tier usage allowances).'),
  })
  .describe('Discounts applied to an invoice line item.')

export const currency = z
  .discriminatedUnion('type', [currencyFiat, currencyCustom])
  .describe('Fiat or custom currency.')

export const governanceFeatureAccess = z
  .object({
    hasAccess: z
      .boolean()

      .describe(
        'Whether the customer currently has access to the feature. `true` for boolean and static entitlements that are available, and for metered entitlements with remaining balance. `false` when the feature is unavailable, the usage limit has been reached, or (when applicable) credits have been exhausted.',
      ),
    reason: governanceFeatureAccessReason.optional(),
  })
  .describe('Access status for a single feature.')

export const customerData = z
  .object({
    billingProfile: profileReference.optional(),
    appData: appCustomerData.optional(),
  })
  .describe('Billing customer data.')

export const upsertCustomerBillingDataRequest = z
  .object({
    billingProfile: profileReference.optional(),
    appData: appCustomerData.optional(),
  })
  .describe('CustomerBillingData upsert request.')

export const creditBalances = z
  .object({
    retrievedAt: dateTime,
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
    fundingMethod: creditFundingMethod,
    currency: createCurrencyCode,
    amount: numeric,
    purchase: createCreditGrantPurchase.optional(),
    taxConfig: createCreditGrantTaxConfig.optional(),
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
    effectiveAt: dateTime.optional(),
    expiresAfter: iso8601Duration.optional(),
    key: externalResourceKey.optional(),
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
    createdAt: dateTime,
    updatedAt: dateTime,
    deletedAt: dateTime.optional(),
    fundingMethod: creditFundingMethod,
    currency: billingCurrencyCode,
    amount: numeric,
    purchase: creditGrantPurchase.optional(),
    taxConfig: creditGrantTaxConfig.optional(),
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
    effectiveAt: dateTime.optional(),
    key: externalResourceKey.optional(),
    expiresAt: dateTime.optional(),
    voidedAt: dateTime.optional(),
    status: creditGrantStatus,
  })

  .describe(
    'A credit grant allocates credits to a customer. Credits are drawn down against charges according to the settlement mode configured on the rate card.',
  )

export const createChargeFlatFeeRequest = z
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
    invoiceAt: dateTime,
    servicePeriod: closedPeriod,
    uniqueReferenceId: z
      .string()
      .optional()
      .describe('Unique reference ID of the charge.'),
    settlementMode: settlementMode,
    taxConfig: taxConfig.optional(),
    paymentTerm: pricePaymentTerm,
    discounts: chargeFlatFeeDiscounts.optional(),
    featureKey: z
      .string()
      .optional()
      .describe('The feature associated with the charge, when applicable.'),
    prorationConfiguration: rateCardProrationConfiguration,
    amountBeforeProration: currencyAmount,
    fullServicePeriod: closedPeriod.optional(),
    billingPeriod: closedPeriod.optional(),
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
    defaultTaxConfig: taxConfig.optional(),
  })
  .describe('Tax settings for a billing workflow.')

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
    timeZone: z
      .string()
      .optional()
      .default('UTC')

      .describe(
        'The value is the name of the time zone as defined in the IANA Time Zone Database (http://www.iana.org/time-zones). The time zone is used to determine the start and end of the time buckets. If not specified, the UTC timezone will be used.',
      ),
    groupByDimensions: z
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
    taxId: partyTaxIdentity.optional(),
    addresses: partyAddresses.optional(),
  })
  .describe('Party represents a person or business entity.')

export const supplier = z
  .object({
    id: z.string().optional().describe('Unique identifier for the party.'),
    name: z
      .string()
      .optional()
      .describe('Legal name or representation of the party.'),
    taxId: partyTaxIdentity.optional(),
    addresses: partyAddresses.optional(),
  })

  .describe(
    "Snapshot of the supplier's information at the time the invoice was issued. Structurally a read-only subset of `BillingParty` (the type configured on the billing profile), so the snapshot stays aligned with the source. `key` is omitted because it is not part of the snapshotted supplier data.",
  )

export const appStripeCreateCheckoutSessionRequestOptions = z
  .object({
    billingAddressCollection:
      appStripeCreateCheckoutSessionBillingAddressCollection
        .optional()
        .default('auto'),
    cancelUrl: z
      .string()
      .optional()

      .describe(
        'URL to redirect customers who cancel the checkout session. Not allowed when ui_mode is "embedded".',
      ),
    clientReferenceId: z
      .string()
      .optional()

      .describe(
        'Unique reference string for reconciling sessions with internal systems. Can be a customer ID, cart ID, or any other identifier.',
      ),
    customerUpdate: appStripeCreateCheckoutSessionCustomerUpdate.optional(),
    consentCollection:
      appStripeCreateCheckoutSessionConsentCollection.optional(),
    currency: currencyCode.optional(),
    customText: appStripeCheckoutSessionCustomTextParams.optional(),
    expiresAt: z.coerce
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
    returnUrl: z
      .string()
      .optional()

      .describe(
        'Return URL for embedded checkout sessions after payment authentication. Required if ui_mode is "embedded" and redirect-based payment methods are enabled.',
      ),
    successUrl: z
      .string()
      .optional()

      .describe(
        'Success URL to redirect customers after completing payment or setup. Not allowed when ui_mode is "embedded". See: https://docs.stripe.com/payments/checkout/custom-success-page',
      ),
    uiMode: appStripeCheckoutSessionUiMode.optional().default('hosted'),
    paymentMethodTypes: z
      .array(z.string())
      .optional()

      .describe(
        'List of payment method types to enable (e.g., "card", "us_bank_account"). If not specified, Stripe enables all relevant payment methods.',
      ),
    redirectOnCompletion:
      appStripeCreateCheckoutSessionRedirectOnCompletion.optional(),
    taxIdCollection: appStripeCreateCheckoutSessionTaxIdCollection.optional(),
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

export const invoiceWorkflowSettings = z
  .object({
    apps: invoiceWorkflowAppsReferences.optional(),
    sourceBillingProfile: profileReference,
    workflow: invoiceWorkflow,
  })

  .describe(
    'Snapshot of the billing workflow configuration captured at invoice creation.',
  )

export const invoiceDetailedLine = z
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
    createdAt: dateTime,
    updatedAt: dateTime,
    deletedAt: dateTime.optional(),
    servicePeriod: closedPeriod,
    totals: totals,
    category: invoiceDetailedLineCostCategory.default('regular'),
    discounts: invoiceLineDiscounts.optional(),
    creditsApplied: z
      .array(invoiceLineCreditsApplied)
      .optional()
      .describe('Credit applied to this detailed line.'),
    externalReferences: invoiceLineExternalReferences.optional(),
    quantity: numeric,
    unitPrice: numeric,
  })

  .describe(
    'A detailed (child) sub-line belonging to a parent invoice line. Detailed lines represent the individual flat-fee components that make up a usage-based parent line after quantity snapshotting.',
  )

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
    updatedAt: dateTime,
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
    createdAt: dateTime,
    updatedAt: dateTime,
    deletedAt: dateTime.optional(),
    key: resourceKey,
    meter: featureMeterReference.optional(),
    unitCost: featureUnitCost.optional(),
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
    unitCost: featureUnitCost.optional(),
  })
  .describe('Feature create request.')

export const updateFeatureRequest = z
  .object({
    unitCost: z
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
      invalidParameters: invalidParameters,
    }),
  )
  .describe('Bad Request.')

export const invoiceBase = z
  .object({
    id: ulid,
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    createdAt: dateTime,
    updatedAt: dateTime,
    deletedAt: dateTime.optional(),
    number: invoiceNumber,
    currency: currencyCode,
    supplier: supplier,
    customer: invoiceCustomer,
    totals: totals,
    servicePeriod: closedPeriod,
    validationIssues: z
      .array(invoiceValidationIssue)
      .optional()

      .describe(
        'Validation issues found during invoice processing. Present only when there are one or more validation findings. An empty list is omitted.',
      ),
    externalReferences: invoiceExternalReferences.optional(),
  })

  .describe(
    'Base fields shared by all invoice types. Spread this model into each concrete invoice variant.',
  )

export const customerStripeCreateCheckoutSessionRequest = z
  .object({
    stripeOptions: appStripeCreateCheckoutSessionRequestOptions,
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

export const chargeFlatFee = z
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
    createdAt: dateTime,
    updatedAt: dateTime,
    deletedAt: dateTime.optional(),
    type: z.literal('flat_fee').describe('The type of the charge.'),
    customer: billingCustomerReference,
    lifecycleController: lifecycleController,
    subscription: subscriptionReference.optional(),
    currency: currencyCode,
    status: chargeStatus,
    invoiceAt: dateTime,
    servicePeriod: closedPeriod,
    fullServicePeriod: closedPeriod,
    billingPeriod: closedPeriod,
    advanceAfter: dateTime.optional(),
    uniqueReferenceId: z
      .string()
      .optional()
      .describe('Unique reference ID of the charge.'),
    settlementMode: settlementMode,
    taxConfig: taxConfig.optional(),
    paymentTerm: pricePaymentTerm,
    discounts: chargeFlatFeeDiscounts.optional(),
    featureKey: z
      .string()
      .optional()
      .describe('The feature associated with the charge, when applicable.'),
    prorationConfiguration: rateCardProrationConfiguration,
    amountAfterProration: currencyAmount,
    price: price,
  })
  .describe('A flat fee charge for a customer.')

export const chargeUsageBased = z
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
    createdAt: dateTime,
    updatedAt: dateTime,
    deletedAt: dateTime.optional(),
    type: z.literal('usage_based').describe('The type of the charge.'),
    customer: billingCustomerReference,
    lifecycleController: lifecycleController,
    subscription: subscriptionReference.optional(),
    currency: currencyCode,
    status: chargeStatus,
    invoiceAt: dateTime,
    servicePeriod: closedPeriod,
    fullServicePeriod: closedPeriod,
    billingPeriod: closedPeriod,
    advanceAfter: dateTime.optional(),
    uniqueReferenceId: z
      .string()
      .optional()
      .describe('Unique reference ID of the charge.'),
    settlementMode: settlementMode,
    taxConfig: taxConfig.optional(),
    discounts: rateCardDiscounts.optional(),
    featureKey: z.string().describe('The feature associated with the charge.'),
    totals: chargeTotals,
    price: price,
  })
  .describe('A usage-based charge for a customer.')

export const createChargeUsageBasedRequest = z
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
    invoiceAt: dateTime,
    servicePeriod: closedPeriod,
    uniqueReferenceId: z
      .string()
      .optional()
      .describe('Unique reference ID of the charge.'),
    settlementMode: settlementMode,
    taxConfig: taxConfig.optional(),
    discounts: rateCardDiscounts.optional(),
    featureKey: z.string().describe('The feature associated with the charge.'),
    price: price,
    fullServicePeriod: closedPeriod.optional(),
    billingPeriod: closedPeriod.optional(),
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
    billingCadence: iso8601Duration.optional(),
    price: price,
    unitConfig: unitConfig.optional(),
    paymentTerm: pricePaymentTerm.optional().default('in_arrears'),
    commitments: spendCommitments.optional(),
    discounts: rateCardDiscounts.optional(),
    taxConfig: rateCardTaxConfig.optional(),
    entitlement: rateCardEntitlement.optional(),
  })

  .describe(
    'A rate card defines the pricing and entitlement of a feature or service.',
  )

export const invoiceLineRateCard = z
  .object({
    price: price,
    taxConfig: rateCardTaxConfig.optional(),
    featureKey: resourceKey.optional(),
    discounts: rateCardDiscounts.optional(),
  })
  .describe('Rate card configuration snapshot for a usage-based invoice line.')

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
  .discriminatedUnion('type', [chargeFlatFee, chargeUsageBased])
  .describe('Customer charge.')

export const createChargeRequest = z
  .discriminatedUnion('type', [
    createChargeFlatFeeRequest,
    createChargeUsageBasedRequest,
  ])
  .describe('Customer charge.')

export const subscriptionAddonRateCard = z
  .object({
    rateCard: rateCard,
    affectedSubscriptionItemIds: z
      .array(ulid)

      .describe(
        'The IDs of the subscription items that this rate card belongs to.',
      ),
  })
  .describe('A rate card for a subscription add-on.')

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
    rateCards: z.array(rateCard).describe('The rate cards of the plan.'),
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
    createdAt: dateTime,
    updatedAt: dateTime,
    deletedAt: dateTime.optional(),
    key: resourceKey,
    version: z
      .number()
      .int()
      .gte(1)
      .default(1)

      .describe(
        'Version of the add-on. Incremented when the add-on is updated.',
      ),
    instanceType: addonInstanceType,
    currency: billingCurrencyCode,
    effectiveFrom: dateTime.optional(),
    effectiveTo: dateTime.optional(),
    status: addonStatus,
    rateCards: z.array(rateCard).describe('The rate cards of the add-on.'),
    validationErrors: z
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
    instanceType: addonInstanceType,
    currency: billingCurrencyCode,
    rateCards: z.array(rateCard).describe('The rate cards of the add-on.'),
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
    instanceType: addonInstanceType,
    rateCards: z.array(rateCard).describe('The rate cards of the add-on.'),
  })
  .describe('Addon upsert request.')

export const invoiceStandardLine = z
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
    createdAt: dateTime,
    updatedAt: dateTime,
    deletedAt: dateTime.optional(),
    type: z
      .literal('standard_line')
      .describe('The type of charge this line item represents.'),
    lifecycleController: lifecycleController,
    servicePeriod: closedPeriod,
    totals: totals,
    discounts: invoiceLineDiscounts.optional(),
    creditsApplied: z
      .array(invoiceLineCreditsApplied)
      .optional()
      .describe('Credit applied to this line item.'),
    externalReferences: invoiceLineExternalReferences.optional(),
    subscription: subscriptionReference.optional(),
    rateCard: invoiceLineRateCard,
    detailedLines: z
      .array(invoiceDetailedLine)

      .describe(
        'Detailed sub-lines that this line has been broken down into. Present when line has individual details.',
      ),
    charge: chargeReference.optional(),
  })

  .describe(
    'A top-level line item on an invoice. Each line represents a single charge, typically associated with a rate card from a subscription. Detailed (child) lines are nested under `detailed_lines` when present.',
  )

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
    createdAt: dateTime,
    updatedAt: dateTime,
    deletedAt: dateTime.optional(),
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

export const subscriptionAddon = z
  .object({
    id: ulid,
    labels: labels.optional(),
    createdAt: dateTime,
    updatedAt: dateTime,
    deletedAt: dateTime.optional(),
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
    addon: addonReference,
    quantity: z
      .number()
      .int()
      .gte(1)

      .describe(
        'The quantity of the add-on. Always 1 for single instance add-ons.',
      ),
    quantityAt: dateTime,
    activeFrom: dateTime,
    activeTo: dateTime.optional(),
    timeline: z
      .array(subscriptionAddonTimelineSegment)

      .describe(
        'The timeline of the add-on. The returned periods are sorted and continuous.',
      ),
    rateCards: z
      .array(subscriptionAddonRateCard)
      .describe('The rate cards of the add-on.'),
  })
  .describe('Addon purchased with a subscription.')

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
    createdAt: dateTime,
    updatedAt: dateTime,
    deletedAt: dateTime.optional(),
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
    billingCadence: iso8601Duration,
    proRatingEnabled: z
      .boolean()
      .optional()
      .default(true)
      .describe('Whether pro-rating is enabled for this plan.'),
    effectiveFrom: dateTime.optional(),
    effectiveTo: dateTime.optional(),
    status: planStatus,
    phases: z
      .array(planPhase)
      .min(1)

      .describe(
        'The plan phases define the pricing ramp for a subscription. A phase switch occurs only at the end of a billing period. At least one phase is required.',
      ),
    settlementMode: settlementMode.optional().default('credit_then_invoice'),
    validationErrors: z
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
    billingCadence: iso8601Duration,
    proRatingEnabled: z
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
    proRatingEnabled: z
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

export const invoiceLine = z
  .discriminatedUnion('type', [invoiceStandardLine])

  .describe(
    'A top-level line item on an invoice. Each line represents a single charge, typically associated with a rate card from a subscription. Detailed (child) lines are nested under `detailed_lines` when present.',
  )

export const profilePagePaginatedResponse = z
  .object({
    data: z.array(profile),
    meta: paginatedMeta,
  })
  .describe('Page paginated response.')

export const subscriptionAddonPagePaginatedResponse = z
  .object({
    data: z.array(subscriptionAddon),
    meta: paginatedMeta,
  })
  .describe('Page paginated response.')

export const planPagePaginatedResponse = z
  .object({
    data: z.array(plan),
    meta: paginatedMeta,
  })
  .describe('Page paginated response.')

export const invoiceStandard = z
  .object({
    id: ulid,
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labels.optional(),
    createdAt: dateTime,
    updatedAt: dateTime,
    deletedAt: dateTime.optional(),
    number: invoiceNumber,
    currency: currencyCode,
    supplier: supplier,
    customer: invoiceCustomer,
    totals: totals,
    servicePeriod: closedPeriod,
    validationIssues: z
      .array(invoiceValidationIssue)
      .optional()

      .describe(
        'Validation issues found during invoice processing. Present only when there are one or more validation findings. An empty list is omitted.',
      ),
    externalReferences: invoiceExternalReferences.optional(),
    type: z
      .literal('standard')
      .describe('Discriminator field identifying this as a standard invoice.'),
    status: invoiceStandardStatus,
    statusDetails: invoiceStatusDetails,
    issuedAt: dateTime.optional(),
    draftUntil: dateTime.optional(),
    quantitySnapshottedAt: dateTime.optional(),
    collectionAt: dateTime.optional(),
    dueAt: dateTime.optional(),
    sentToCustomerAt: dateTime.optional(),
    workflow: invoiceWorkflowSettings,
    lines: z
      .array(invoiceLine)
      .optional()

      .describe(
        'Line items on this invoice. Always returned on single-resource GET; omitted on list endpoints unless explicitly expanded.',
      ),
  })
  .describe('A standard invoice for charges owed by the customer.')

export const invoice = z
  .discriminatedUnion('type', [invoiceStandard])

  .describe(
    'An invoice issued to a customer. The `type` field determines the concrete variant: - `standard`: a standard invoice for charges owed.',
  )

export const listMeteringEventsQueryParams = z.object({
  page: cursorPaginationQueryPage.optional(),
  filter: listEventsParamsFilter.optional(),
  sort: sortQuery.optional(),
})

export const listMeteringEventsResponse = z.object({
  data: z.array(ingestedEvent),
  meta: cursorMeta,
})

export const ingestMeteringEventsBody = z.union([event, z.array(event)])

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

export const queryMeterCsvBody = meterQueryRequest

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

export const createSubscriptionAddonPathParams = z.object({
  subscriptionId: ulid,
})

export const createSubscriptionAddonBody = createSubscriptionAddonRequest

export const createSubscriptionAddonResponse = subscriptionAddon

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

export const getInvoicePathParams = z.object({
  invoiceId: ulid,
})

export const getInvoiceResponse = invoice

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
  includeDeleted: z.coerce
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

export const labelsWire = z
  .record(z.string(), z.string())

  .describe(
    'Labels store metadata of an entity that can be used for filtering an entity list or for searching across entity types. Keys must be of length 1-63 characters, and cannot start with "kong", "konnect", "mesh", "kic", or "\\_".',
  )

export const currencyCodeWire = z
  .string()
  .min(3)
  .max(3)
  .regex(new RegExp('^[A-Z]{3}$'))

  .describe(
    'Three-letter [ISO4217](https://www.iso.org/iso-4217-currency-codes.html) currency code. Custom three-letter currency codes are also supported for convenience.',
  )

export const numericWire = z
  .string()
  .regex(new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$'))
  .describe('Numeric represents an arbitrary precision number.')

export const cursorPaginationQueryPageWire = z
  .strictObject({
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

export const stringFieldFilterWire = z
  .union([
    z.string(),
    z.strictObject({
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

export const ulidWire = z
  .string()
  .regex(new RegExp('^[0-7][0-9A-HJKMNP-TV-Z]{25}$'))
  .describe('ULID (Universally Unique Lexicographically Sortable Identifier).')

export const dateTimeWire = z
  .string()
  .datetime()

  .describe(
    '[RFC3339](https://tools.ietf.org/html/rfc3339) formatted date-time string in UTC.',
  )

export const sortQueryWire = z
  .strictObject({
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

export const ingestedEventValidationErrorWire = z
  .strictObject({
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

export const cursorMetaPageWire = z
  .strictObject({
    first: z.string().optional().describe('URI to the first page.'),
    last: z.string().optional().describe('URI to the last page.'),
    next: z.string().optional().describe('URI to the next page.'),
    previous: z.string().optional().describe('URI to the previous page.'),
    size: z.number().int().optional().describe('Requested page size.'),
  })
  .describe('Cursor pagination metadata.')

export const invalidRulesWire = z
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

export const invalidParameterMinimumRuleWire = z
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

export const invalidParameterMaximumRuleWire = z
  .enum(['max_length', 'max_items', 'max'])
  .describe('Maximum-length (or maximum-value) validation rules.')

export const invalidParameterChoiceRuleWire = z
  .enum(['enum'])
  .describe('The enum validation rule.')

export const invalidParameterDependentRuleWire = z
  .enum(['dependent_fields'])
  .describe('The dependent-fields validation rule.')

export const baseErrorWire = z
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

export const resourceKeyWire = z
  .string()
  .min(1)
  .max(64)
  .regex(new RegExp('^[a-z0-9]+(?:_[a-z0-9]+)*$'))
  .describe('A key is a unique string that is used to identify a resource.')

export const meterAggregationWire = z
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

export const pageMetaWire = z
  .strictObject({
    number: z.number().int().describe('Page number.'),
    size: z.number().int().describe('Page size.'),
    total: z
      .number()
      .int()
      .describe('Total number of items in the collection.'),
  })
  .describe('Pagination information.')

export const meterQueryGranularityWire = z
  .union([
    z.literal('PT1M'),
    z.literal('PT1H'),
    z.literal('P1D'),
    z.literal('P1M'),
  ])

  .describe(
    'The granularity of the time grouping. Time durations are specified in ISO 8601 format.',
  )

export const queryFilterStringWire = z
  .strictObject({
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
        .array(queryFilterStringWire)
        .min(1)
        .max(10)
        .optional()
        .describe('Combines the provided filters with a logical AND.')
    },
    get or() {
      return z
        .array(queryFilterStringWire)
        .min(1)
        .max(10)
        .optional()
        .describe('Combines the provided filters with a logical OR.')
    },
  })

  .describe(
    'A query filter for a string attribute. Operators are mutually exclusive, only one operator is allowed at a time.',
  )

export const externalResourceKeyWire = z
  .string()
  .min(1)
  .max(256)

  .describe(
    'ExternalResourceKey is a unique string that is used to identify a resource in an external system.',
  )

export const usageAttributionSubjectKeyWire = z
  .string()
  .min(1)
  .describe('Subject key.')

export const countryCodeWire = z
  .string()
  .min(2)
  .max(2)
  .regex(new RegExp('^[A-Z]{2}$'))

  .describe(
    '[ISO 3166-1](https://www.iso.org/iso-3166-country-codes.html) alpha-2 country code. Custom two-letter country codes are also supported for convenience.',
  )

export const appStripeCreateCheckoutSessionBillingAddressCollectionWire = z
  .enum(['auto', 'required'])

  .describe(
    "Controls whether Checkout collects the customer's billing address.",
  )

export const appStripeCreateCheckoutSessionCustomerUpdateBehaviorWire = z
  .enum(['auto', 'never'])
  .describe('Behavior for updating customer fields from checkout session.')

export const appStripeCreateCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPositionWire =
  z
    .enum(['auto', 'hidden'])
    .describe('Position of payment method reuse agreement in the UI.')

export const appStripeCreateCheckoutSessionConsentCollectionPromotionsWire = z
  .enum(['auto', 'none'])
  .describe('Promotional communication consent collection setting.')

export const appStripeCreateCheckoutSessionConsentCollectionTermsOfServiceWire =
  z
    .enum(['none', 'required'])
    .describe('Terms of service acceptance requirement.')

export const appStripeCheckoutSessionCustomTextParamsWire = z
  .strictObject({
    after_submit: z
      .strictObject({
        message: z
          .string()
          .max(1200)
          .optional()
          .describe('The custom message text (max 1200 characters).'),
      })
      .optional()
      .describe('Text displayed after the payment confirmation button.'),
    shipping_address: z
      .strictObject({
        message: z
          .string()
          .max(1200)
          .optional()
          .describe('The custom message text (max 1200 characters).'),
      })
      .optional()
      .describe('Text displayed alongside shipping address collection.'),
    submit: z
      .strictObject({
        message: z
          .string()
          .max(1200)
          .optional()
          .describe('The custom message text (max 1200 characters).'),
      })
      .optional()
      .describe('Text displayed alongside the payment confirmation button.'),
    terms_of_service_acceptance: z
      .strictObject({
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

export const appStripeCheckoutSessionUiModeWire = z
  .enum(['embedded', 'hosted'])
  .describe('Checkout Session UI mode.')

export const appStripeCreateCheckoutSessionRedirectOnCompletionWire = z
  .enum(['always', 'if_required', 'never'])
  .describe('Redirect behavior for embedded checkout sessions.')

export const appStripeCreateCheckoutSessionTaxIdCollectionRequiredWire = z
  .enum(['if_supported', 'never'])
  .describe('Tax ID collection requirement level.')

export const appStripeCheckoutSessionModeWire = z
  .enum(['setup'])

  .describe(
    'Stripe Checkout Session mode. Determines the primary purpose of the checkout session.',
  )

export const appStripeCreateCustomerPortalSessionOptionsWire = z
  .strictObject({
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

export const entitlementTypeWire = z
  .enum(['metered', 'static', 'boolean'])
  .describe('The type of the entitlement.')

export const createLabelsWire = z
  .record(z.string(), z.string())

  .describe(
    'Labels store metadata of an entity that can be used for filtering an entity list or for searching across entity types. Keys must be of length 1-63 characters, and cannot start with "kong", "konnect", "mesh", "kic", or "\\_".',
  )

export const creditFundingMethodWire = z
  .enum(['none', 'invoice', 'external'])

  .describe(
    'The funding method describes how the grant is funded. - `none`: No funding workflow applies, for example promotional grants - `invoice`: The grant is funded by an in-system invoice flow - `external`: The grant is funded outside the system (e.g., wire transfer, external invoice, or manual reconciliation)',
  )

export const creditAvailabilityPolicyWire = z
  .enum(['on_creation'])

  .describe(
    'When credits become available for consumption. - `on_creation`: Credits are available as soon as the grant is created. - `on_authorization`: Credits are available once the payment is authorized. - `on_settlement`: Credits are available once the payment is settled.',
  )

export const taxBehaviorWire = z
  .enum(['inclusive', 'exclusive'])

  .describe(
    'Tax behavior. This enum is used to specify whether tax is included in the price or excluded from the price.',
  )

export const iso8601DurationWire = z
  .string()

  .regex(
    new RegExp(
      '^P(?:\\d+(?:\\.\\d+)?Y)?(?:\\d+(?:\\.\\d+)?M)?(?:\\d+(?:\\.\\d+)?W)?(?:\\d+(?:\\.\\d+)?D)?(?:T(?:\\d+(?:\\.\\d+)?H)?(?:\\d+(?:\\.\\d+)?M)?(?:\\d+(?:\\.\\d+)?S)?)?$',
    ),
  )

  .describe(
    '[ISO 8601 Duration](https://docs.digi.com/resources/documentation/digidocs/90001488-13/reference/r_iso_8601_duration_format.htm) string.',
  )

export const creditPurchasePaymentSettlementStatusWire = z
  .enum(['pending', 'authorized', 'settled'])

  .describe(
    'Credit purchase payment settlement status. - `pending`: Payment has been initiated and is not yet authorized. - `authorized`: Payment has been authorized. - `settled`: Payment has been settled.',
  )

export const creditGrantStatusWire = z
  .enum(['pending', 'active', 'expired', 'voided'])

  .describe(
    'Credit grant lifecycle status. - `pending`: The credit block has been created but is not yet valid. (`effective_at` is in the future or availability_policy is not met) - `active`: The credit block is currently valid and eligible for consumption. (`effective_at` is in the past, `expires_at` is in the future and availability_policy is met) - `expired`: The credit block expired with remaining unused balance, `expires_at` time has passed. - `voided`: The credit block was voided. Remaining balance is forfeited.',
  )

export const stringFieldFilterExactWire = z
  .union([
    z.string(),
    z.strictObject({
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

export const creditTransactionTypeWire = z
  .enum(['funded', 'consumed', 'expired'])

  .describe(
    'The type of the credit transaction. - `funded`: Credit granted and available for consumption. - `consumed`: Credit consumed by usage or fees. - `expired`: Credit removed because it expired before being used.',
  )

export const chargesExpandWire = z
  .enum(['real_time_usage'])

  .describe(
    "Expands for customer charges. Values: - `real_time_usage`: The charge's real-time usage.",
  )

export const lifecycleControllerWire = z
  .enum(['system', 'manual'])

  .describe(
    'Identifies whether a resource lifecycle is controlled by OpenMeter or manually overridden by the API user. Values: - `system`: The resource lifecycle is controlled by OpenMeter. - `manual`: The resource lifecycle was manually overridden by the API user.',
  )

export const chargeStatusWire = z
  .enum(['created', 'active', 'final', 'deleted'])

  .describe(
    'Lifecycle status of a charge. Values: - `created`: The charge has been created but is not active yet. - `active`: The charge is active. - `final`: The charge is fully finalized and no further changes are expected. - `deleted`: The charge has been deleted.',
  )

export const settlementModeWire = z
  .enum(['credit_then_invoice', 'credit_only'])

  .describe(
    'Settlement mode for billing. Values: - `credit_then_invoice`: Credits are applied first, then any remainder is invoiced. - `credit_only`: Usage is settled exclusively against credits.',
  )

export const taxConfigStripeWire = z
  .strictObject({
    code: z
      .string()
      .regex(new RegExp('^txcd_\\d{8}$'))
      .describe('Product [tax code](https://docs.stripe.com/tax/tax-codes).'),
  })
  .describe('The tax config for Stripe.')

export const taxConfigExternalInvoicingWire = z
  .strictObject({
    code: z
      .string()
      .max(64)

      .describe(
        'The tax code should be interpreted by the external invoicing provider.',
      ),
  })
  .describe('External invoicing tax config.')

export const pricePaymentTermWire = z
  .union([z.literal('in_advance'), z.literal('in_arrears')])
  .describe('The payment term of a flat price.')

export const chargeFlatFeeDiscountsWire = z
  .strictObject({
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

export const rateCardProrationModeWire = z
  .enum(['no_proration', 'prorate_prices'])

  .describe(
    'The proration mode of the rate card. Values: - `no_proration`: No proration. - `prorate_prices`: Prorate the price based on the time remaining in the billing period.',
  )

export const priceFreeWire = z
  .strictObject({
    type: z.literal('free').describe('The type of the price.'),
  })
  .describe('Free price.')

export const subscriptionStatusWire = z
  .enum(['active', 'inactive', 'canceled', 'scheduled'])
  .describe('Subscription status.')

export const subscriptionEditTimingEnumWire = z
  .enum(['immediate', 'next_billing_cycle'])

  .describe(
    'Subscription edit timing. When immediate, the requested changes take effect immediately. When next_billing_cycle, the requested changes take effect at the next billing cycle.',
  )

export const unitConfigOperationWire = z
  .enum(['divide', 'multiply'])

  .describe(
    'The arithmetic operation used to convert raw metered units into billing units. - `divide`: Divide the metered quantity by the conversion factor (e.g., bytes ÷ 1e9 = GB). - `multiply`: Multiply the metered quantity by the conversion factor (e.g., cost × 1.2 = cost + 20% margin).',
  )

export const unitConfigRoundingModeWire = z
  .enum(['ceiling', 'floor', 'half_up', 'none'])

  .describe(
    'The rounding mode applied to the converted quantity for invoicing. Rounding is applied only to the invoiced quantity. Entitlement balance checks use the precise decimal value after conversion. - `ceiling`: Round up to the next integer (typical for package-style billing). - `floor`: Round down to the previous integer. - `half_up`: Round to the nearest integer, with 0.5 rounding up. - `none`: No rounding; the converted value is used as-is.',
  )

export const rateCardStaticEntitlementWire = z
  .strictObject({
    type: z.literal('static').describe('The type of the entitlement template.'),
    config: z
      .unknown()

      .describe(
        'The entitlement config as a JSON object. Returned when checking entitlement access; useful for configuring fine-grained access settings implemented in your own system.',
      ),
  })
  .describe('The entitlement template of a static entitlement.')

export const rateCardBooleanEntitlementWire = z
  .strictObject({
    type: z
      .literal('boolean')
      .describe('The type of the entitlement template.'),
  })
  .describe('The entitlement template of a boolean entitlement.')

export const appTypeWire = z
  .enum(['sandbox', 'stripe', 'external_invoicing'])
  .describe('The type of the app.')

export const appStatusWire = z
  .enum(['ready', 'unauthorized'])
  .describe('Connection status of an installed app.')

export const taxIdentificationCodeWire = z
  .string()
  .min(1)
  .max(32)

  .describe(
    'Tax identifier code is a normalized tax code shown on the original identity document.',
  )

export const workflowCollectionAlignmentSubscriptionWire = z
  .strictObject({
    type: z.literal('subscription').describe('The type of alignment.'),
  })

  .describe(
    'BillingWorkflowCollectionAlignmentSubscription specifies the alignment for collecting the pending line items into an invoice.',
  )

export const workflowInvoicingSubscriptionEndProrationModeWire = z
  .enum(['bill_full_period', 'bill_actual_period'])
  .describe('Billing workflow subscription end proration mode.')

export const workflowPaymentChargeAutomaticallySettingsWire = z
  .strictObject({
    collection_method: z
      .literal('charge_automatically')
      .describe('The collection method for the invoice.'),
  })

  .describe(
    'Payment settings for a billing workflow when the collection method is charge automatically.',
  )

export const workflowPaymentSendInvoiceSettingsWire = z
  .strictObject({
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

export const invoiceNumberWire = z
  .string()
  .min(1)
  .max(256)

  .describe(
    'InvoiceNumber is a unique identifier for the invoice, generated by the invoicing app. The uniqueness depends on a lot of factors: - app setting (unique per app or unique per customer) - multiple app scenarios (multiple apps generating invoices with the same prefix)',
  )

export const invoiceValidationIssueSeverityWire = z
  .enum(['critical', 'warning'])
  .describe('Severity level of an invoice validation issue.')

export const invoiceExternalReferencesWire = z
  .strictObject({
    invoicing_id: z
      .string()
      .optional()

      .describe(
        'The ID assigned by the external invoicing app (e.g. Stripe invoice ID).',
      ),
    payment_id: z
      .string()
      .optional()

      .describe(
        'The ID assigned by the external payment app (e.g. Stripe payment intent ID).',
      ),
  })

  .describe(
    'External identifiers assigned to an invoice by third-party systems.',
  )

export const invoiceStandardStatusWire = z
  .enum([
    'draft',
    'issuing',
    'issued',
    'payment_processing',
    'overdue',
    'paid',
    'uncollectible',
    'voided',
  ])
  .describe('Lifecycle status of a standard invoice.')

export const invoiceAvailableActionDetailsWire = z
  .strictObject({
    resulting_state: z
      .string()

      .describe(
        'The extended status the invoice will transition to after performing this action.',
      ),
  })

  .describe(
    'Details about an available invoice action including the resulting state.',
  )

export const invoiceWorkflowInvoicingSettingsWire = z
  .strictObject({
    auto_advance: z
      .boolean()
      .optional()
      .default(true)

      .describe(
        'Whether to automatically issue the invoice after the draft_period has passed.',
      ),
    draft_period: z
      .string()
      .optional()
      .default('P0D')

      .describe(
        'The period for the invoice to be kept in draft status for manual reviews.',
      ),
  })

  .describe(
    'Invoice-level invoicing settings. A subset of BillingWorkflowInvoicingSettings limited to fields that are meaningful per-invoice. progressive_billing is omitted as it is a gather-time / profile-level decision.',
  )

export const invoiceDiscountReasonWire = z
  .enum(['maximum_spend', 'ratecard_percentage', 'ratecard_usage'])
  .describe('The reason a discount was applied to an invoice line.')

export const invoiceLineExternalReferencesWire = z
  .strictObject({
    invoicing_id: z
      .string()
      .optional()
      .describe('The ID assigned by the external invoicing app.'),
  })

  .describe(
    'External identifiers for an invoice line item assigned by third-party systems.',
  )

export const invoiceDetailedLineCostCategoryWire = z
  .enum(['regular', 'commitment'])
  .describe('Cost category of a detailed invoice line item.')

export const currencyTypeWire = z
  .enum(['fiat', 'custom'])

  .describe(
    'Currency type for custom currencies. It should be a unique code but not conflicting with any existing standard currency codes.',
  )

export const currencyCodeCustomWire = z
  .string()
  .min(3)
  .max(24)

  .describe(
    'Custom currency code. It should be a unique code but not conflicting with any existing fiat currency codes.',
  )

export const featureLlmTokenTypeWire = z
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

export const llmCostProviderWire = z
  .strictObject({
    id: z
      .string()
      .describe('Identifier of the provider, e.g., "openai", "anthropic".'),
    name: z
      .string()
      .describe('Name of the provider, e.g., "OpenAI", "Anthropic".'),
  })
  .describe('LLM Provider')

export const llmCostModelWire = z
  .strictObject({
    id: z
      .string()

      .describe('Identifier of the model, e.g., "gpt-4", "claude-3-5-sonnet".'),
    name: z
      .string()
      .describe('Name of the model, e.g., "GPT-4", "Claude 3.5 Sonnet".'),
  })
  .describe('LLM Model')

export const llmCostPriceSourceWire = z
  .enum(['manual', 'system'])
  .describe('Identifies where an LLM cost price came from.')

export const planStatusWire = z
  .enum(['draft', 'active', 'archived', 'scheduled'])

  .describe(
    'The status of a plan. - `draft`: The plan has not yet been published and can be edited. - `active`: The plan is published and can be used in subscriptions. - `archived`: The plan is no longer available for use. - `scheduled`: The plan is scheduled to be published at a future date.',
  )

export const productCatalogValidationErrorWire = z
  .strictObject({
    code: z.string().describe('Machine-readable error code.'),
    message: z.string().describe('Human-readable description of the error.'),
    attributes: z
      .record(z.string(), z.unknown())
      .optional()
      .describe('Additional structured context.'),
    field: z.string().describe('The path to the field.'),
  })
  .describe('Validation errors providing detailed description of the issue.')

export const addonInstanceTypeWire = z
  .enum(['single', 'multiple'])

  .describe(
    'The instanceType of the add-on. - `single`: Can be added to a subscription only once. - `multiple`: Can be added to a subscription more than once.',
  )

export const addonStatusWire = z
  .enum(['draft', 'active', 'archived'])

  .describe(
    'The status of the add-on defined by the `effective_from` and `effective_to` properties. - `draft`: The add-on has not yet been published and can be edited. - `active`: The add-on is published and available for use. - `archived`: The add-on is no longer available for use.',
  )

export const governanceQueryRequestCustomersWire = z
  .strictObject({
    keys: z
      .array(z.string())
      .min(1)
      .max(100)

      .describe(
        'Each entry can be a customer `key` or a usage-attribution subject `key`. Identifiers that cannot be resolved to a customer are reported in the response `errors` array.',
      ),
  })
  .describe('List of customer identifiers to evaluate access for.')

export const governanceQueryRequestFeaturesWire = z
  .strictObject({
    keys: z
      .array(z.string())
      .min(1)
      .max(100)
      .describe('List of feature keys to evaluate access for.'),
  })

  .describe(
    'Optional list of feature keys to evaluate access for. If omitted, all features available in the organization are returned. Providing this list is recommended to reduce the response size and the load on the backend services.',
  )

export const governanceFeatureAccessReasonCodeWire = z
  .enum([
    'unknown',
    'usage_limit_reached',
    'feature_unavailable',
    'feature_not_found',
    'no_credit_available',
  ])
  .describe('Machine-readable reason code for denied feature access.')

export const governanceQueryErrorCodeWire = z
  .enum(['unknown', 'customer_not_found'])
  .describe('Error code for a governance query failure.')

export const queryFilterIntegerWire = z
  .strictObject({
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
        .array(queryFilterIntegerWire)
        .min(1)
        .max(10)
        .optional()
        .describe('Combines the provided filters with a logical AND.')
    },
    get or() {
      return z
        .array(queryFilterIntegerWire)
        .min(1)
        .max(10)
        .optional()
        .describe('Combines the provided filters with a logical OR.')
    },
  })

  .describe(
    'A query filter for an integer attribute. Operators are mutually exclusive, only one operator is allowed at a time.',
  )

export const queryFilterFloatWire = z
  .strictObject({
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
        .array(queryFilterFloatWire)
        .min(1)
        .max(10)
        .optional()
        .describe('Combines the provided filters with a logical AND.')
    },
    get or() {
      return z
        .array(queryFilterFloatWire)
        .min(1)
        .max(10)
        .optional()
        .describe('Combines the provided filters with a logical OR.')
    },
  })

  .describe(
    'A query filter for a float attribute. Operators are mutually exclusive, only one operator is allowed at a time.',
  )

export const queryFilterBooleanWire = z
  .strictObject({
    eq: z
      .boolean()
      .optional()
      .describe('The attribute equals the provided value.'),
  })

  .describe(
    'A query filter for a boolean attribute. Operators are mutually exclusive, only one operator is allowed at a time.',
  )

export const pagePaginationQueryWire = z
  .strictObject({
    page: z
      .strictObject({
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

export const publicLabelsWire = z
  .record(z.string(), z.string())

  .describe(
    'Public labels store information about an entity that can be used for filtering a list of objects.',
  )

export const booleanFieldFilterWire = z
  .union([
    z.boolean(),
    z.strictObject({
      eq: z
        .boolean()
        .describe('Value strictly equals the given boolean value.'),
    }),
  ])
  .describe('Filter by a boolean value (true/false).')

export const numericFieldFilterWire = z
  .union([
    z.number(),
    z.strictObject({
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

export const chargeTypeWire = z
  .enum(['flat_fee', 'usage_based'])

  .describe(
    'Type of a charge. Values: - `flat_fee`: A fixed-amount charge. - `usage_based`: A usage-priced charge.',
  )

export const invoiceTypeWire = z
  .enum(['standard'])
  .describe('The type of a billing invoice.')

export const invoiceLineTypeWire = z
  .enum(['standard_line'])
  .describe('Line item type discriminator.')

export const priceTypeWire = z
  .enum(['free', 'flat', 'unit', 'graduated', 'volume'])

  .describe(
    "The type of the price. - `free`: No charge, the rate card is included at no cost. - `flat`: A fixed amount charged once per billing period, regardless of usage. - `unit`: A fixed rate charged per billing unit consumed. - `graduated`: Tiered pricing where each tier's rate applies only to usage within that tier. - `volume`: Tiered pricing where the rate for the highest tier reached applies to all units in the period.",
  )

export const collectionAlignmentWire = z
  .enum(['subscription', 'anchored'])

  .describe(
    'BillingCollectionAlignment specifies when the pending line items should be collected into an invoice.',
  )

export const collectionMethodWire = z
  .enum(['charge_automatically', 'send_invoice'])

  .describe(
    'Collection method specifies how the invoice should be collected (automatic or manual).',
  )

export const featureUnitCostTypeWire = z
  .enum(['llm', 'manual'])
  .describe('The type of unit cost.')

export const systemAccountAccessTokenWire = z
  .strictObject({
    type: z.literal('http').describe('Http authentication'),
    scheme: z.literal('Bearer').describe('bearer auth scheme'),
  })

  .describe(
    'The system account access token is meant for automations and integrations that are not directly associated with a human identity.',
  )

export const personalAccessTokenWire = z
  .strictObject({
    type: z.literal('http').describe('Http authentication'),
    scheme: z.literal('Bearer').describe('bearer auth scheme'),
  })

  .describe(
    'The personal access token is meant to be used as an alternative to basic-auth when accessing Konnect via APIs.',
  )

export const konnectAccessTokenWire = z
  .strictObject({
    type: z.literal('http').describe('Http authentication'),
    scheme: z.literal('Bearer').describe('bearer auth scheme'),
  })

  .describe(
    'The Konnect access token is meant to be used by the Konnect dashboard and the decK CLI authenticate with.',
  )

export const updateMeterRequestWire = z
  .strictObject({
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
    labels: labelsWire.optional(),
    dimensions: z
      .record(z.string(), z.string())
      .optional()

      .describe(
        'Named JSONPath expressions to extract the group by values from the event data. Keys must be unique and consist only alphanumeric and underscore characters.',
      ),
  })
  .describe('Meter update request.')

export const appCustomerDataStripeWire = z
  .strictObject({
    customer_id: z.string().optional().describe('The Stripe customer ID used.'),
    default_payment_method_id: z
      .string()
      .optional()
      .describe('The Stripe default payment method ID.'),
    labels: labelsWire.optional(),
  })
  .describe('Stripe customer data.')

export const appCustomerDataExternalInvoicingWire = z
  .strictObject({
    labels: labelsWire.optional(),
  })
  .describe('External invoicing customer data.')

export const billingCurrencyCodeWire = z
  .union([currencyCodeWire])
  .describe('Fiat or custom currency code.')

export const createCurrencyCodeWire = z
  .union([currencyCodeWire])
  .describe('Fiat or custom currency code.')

export const listCostBasesParamsFilterWire = z
  .strictObject({
    fiat_code: currencyCodeWire.optional(),
  })
  .describe('Filter options for listing cost bases.')

export const currencyAmountWire = z
  .strictObject({
    amount: numericWire,
    currency: currencyCodeWire,
  })
  .describe('Monetary amount in a specific currency.')

export const priceFlatWire = z
  .strictObject({
    type: z.literal('flat').describe('The type of the price.'),
    amount: numericWire,
  })
  .describe('Flat price.')

export const priceUnitWire = z
  .strictObject({
    type: z.literal('unit').describe('The type of the price.'),
    amount: numericWire,
  })

  .describe(
    'Unit price. Charges a fixed rate per billing unit. When UnitConfig is present on the rate card, billing units are the converted quantities (e.g. GB instead of bytes).',
  )

export const rateCardDiscountsWire = z
  .strictObject({
    percentage: z
      .number()
      .nonnegative()
      .lte(100)
      .optional()
      .describe('Percentage discount applied to the price (0–100).'),
    usage: numericWire.optional(),
  })
  .describe('Discount configuration for a rate card.')

export const totalsWire = z
  .strictObject({
    amount: numericWire,
    taxes_total: numericWire,
    taxes_inclusive_total: numericWire,
    taxes_exclusive_total: numericWire,
    charges_total: numericWire,
    discounts_total: numericWire,
    credits_total: numericWire,
    total: numericWire,
  })

  .describe(
    'Totals contains the summaries of all calculations for a billing resource.',
  )

export const spendCommitmentsWire = z
  .strictObject({
    minimum_amount: numericWire.optional(),
    maximum_amount: numericWire.optional(),
  })

  .describe(
    'Spend commitments for a rate card. The customer is committed to spend at least the minimum amount and at most the maximum amount.',
  )

export const invoiceLineCreditsAppliedWire = z
  .strictObject({
    amount: numericWire,
    description: z
      .string()
      .optional()

      .describe(
        'Optional human-readable description of the credit allocation.',
      ),
  })
  .describe('A credit allocation applied to an invoice line item.')

export const featureManualUnitCostWire = z
  .strictObject({
    type: z
      .literal('manual')
      .describe('The type discriminator for manual unit cost.'),
    amount: numericWire,
  })
  .describe('A fixed per-unit cost amount.')

export const featureLlmUnitCostPricingWire = z
  .strictObject({
    input_per_token: numericWire,
    output_per_token: numericWire,
    cache_read_per_token: numericWire.optional(),
    reasoning_per_token: numericWire.optional(),
    cache_write_per_token: numericWire.optional(),
  })
  .describe('Resolved per-token pricing from the LLM cost database.')

export const llmCostModelPricingWire = z
  .strictObject({
    input_per_token: numericWire,
    output_per_token: numericWire,
    cache_read_per_token: numericWire.optional(),
    cache_write_per_token: numericWire.optional(),
    reasoning_per_token: numericWire.optional(),
  })
  .describe('Token pricing for an LLM model, denominated per token.')

export const queryFilterNumericWire = z
  .strictObject({
    gt: numericWire.optional(),
    gte: numericWire.optional(),
    lt: numericWire.optional(),
    lte: numericWire.optional(),
    get and() {
      return z
        .array(queryFilterNumericWire)
        .min(1)
        .max(10)
        .optional()
        .describe('Combines the provided filters with a logical AND.')
    },
    get or() {
      return z
        .array(queryFilterNumericWire)
        .min(1)
        .max(10)
        .optional()
        .describe('Combines the provided filters with a logical OR.')
    },
  })

  .describe(
    'A query filter for a numeric attribute. Operators are mutually exclusive, only one operator is allowed at a time.',
  )

export const cursorPaginationQueryWire = z
  .strictObject({
    page: cursorPaginationQueryPageWire.optional(),
  })
  .describe('Cursor page query.')

export const listMetersParamsFilterWire = z
  .strictObject({
    key: stringFieldFilterWire.optional(),
    name: stringFieldFilterWire.optional(),
  })
  .describe('Filter options for listing meters.')

export const listLlmCostPricesParamsFilterWire = z
  .strictObject({
    provider: stringFieldFilterWire.optional(),
    model_id: stringFieldFilterWire.optional(),
    model_name: stringFieldFilterWire.optional(),
    currency: stringFieldFilterWire.optional(),
    source: stringFieldFilterWire.optional(),
  })
  .describe('Filter options for listing LLM cost prices.')

export const labelsFieldFilterWire = z
  .record(z.string(), stringFieldFilterWire)

  .describe(
    "Filters on the resource's `labels` field. The schema is a map keyed by the label name; each value is a `StringFieldFilter`. Both deepObject forms are accepted: `filter[labels][key]=value` (nested) and `filter[labels.key]=value` (dot-notation).",
  )

export const ulidFieldFilterWire = z
  .union([
    ulidWire,
    z.strictObject({
      eq: ulidWire.optional(),
      oeq: z
        .array(ulidWire)
        .optional()

        .describe(
          'Returns entities that exact match any of the comma-delimited ULIDs in the filter string.',
        ),
      neq: ulidWire.optional(),
    }),
  ])

  .describe(
    'Filters on the given ULID field value by exact match. All properties are optional; provide exactly one to specify the comparison.',
  )

export const customerReferenceWire = z
  .strictObject({
    id: ulidWire,
  })
  .describe('Customer reference.')

export const profileReferenceWire = z
  .strictObject({
    id: ulidWire,
  })
  .describe('Billing profile reference.')

export const createResourceReferenceWire = z
  .strictObject({
    id: ulidWire,
  })
  .describe('TaxCode reference.')

export const taxCodeReferenceWire = z
  .strictObject({
    id: ulidWire,
  })
  .describe('TaxCode reference.')

export const creditGrantInvoiceReferenceWire = z
  .strictObject({
    id: ulidWire.optional(),
    line: z
      .strictObject({
        id: ulidWire,
      })
      .optional()
      .describe('Identifier of the invoice line associated with the grant.'),
  })
  .describe('Invoice references for the grant.')

export const billingCustomerReferenceWire = z
  .strictObject({
    id: ulidWire,
  })
  .describe('Customer reference.')

export const subscriptionReferenceWire = z
  .strictObject({
    id: ulidWire,
    phase: z
      .strictObject({
        id: ulidWire,
        item: z
          .strictObject({
            id: ulidWire,
          })
          .describe('The item of the phase.'),
      })
      .describe('The phase of the subscription.'),
  })

  .describe(
    'Subscription reference represents a reference to the specific subscription item this entity represents.',
  )

export const addonReferenceWire = z
  .strictObject({
    id: ulidWire,
  })
  .describe('Addon reference.')

export const featureReferenceWire = z
  .strictObject({
    id: ulidWire,
  })
  .describe('Feature reference.')

export const appReferenceWire = z
  .strictObject({
    id: ulidWire,
  })
  .describe('App reference.')

export const chargeReferenceWire = z
  .strictObject({
    id: ulidWire,
  })
  .describe('Reference to a charge associated with an invoice line.')

export const currencyFiatWire = z
  .strictObject({
    id: ulidWire,
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
    code: currencyCodeWire,
  })
  .describe('Currency describes a currency supported by the billing system.')

export const dateTimeFieldFilterWire = z
  .union([
    dateTimeWire,
    z.strictObject({
      eq: dateTimeWire.optional(),
      lt: dateTimeWire.optional(),
      lte: dateTimeWire.optional(),
      gt: dateTimeWire.optional(),
      gte: dateTimeWire.optional(),
    }),
  ])

  .describe(
    'Filters on the given datetime (RFC-3339) field value. All properties are optional; provide exactly one to specify the comparison.',
  )

export const eventWire = z
  .strictObject({
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
      .union([dateTimeWire, z.null()])
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

export const meterQueryRowWire = z
  .strictObject({
    value: numericWire,
    from: dateTimeWire,
    to: dateTimeWire,
    dimensions: z
      .record(z.string(), z.string())

      .describe(
        'The dimensions the value is aggregated over. `subject` and `customer_id` are reserved dimensions.',
      ),
  })
  .describe('A row in the result of a meter query.')

export const appStripeCreateCustomerPortalSessionResultWire = z
  .strictObject({
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
    created_at: dateTimeWire,
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

export const closedPeriodWire = z
  .strictObject({
    from: dateTimeWire,
    to: dateTimeWire,
  })

  .describe(
    'A period with defined start and end dates. The period is always inclusive at the start and exclusive at the end.',
  )

export const subscriptionAddonTimelineSegmentWire = z
  .strictObject({
    active_from: dateTimeWire,
    active_to: dateTimeWire.optional(),
    quantity: z
      .number()
      .int()
      .nonnegative()
      .describe('The quantity of the add-on for the given period.'),
  })
  .describe('A subscription add-on event.')

export const costBasisWire = z
  .strictObject({
    id: ulidWire,
    fiat_code: currencyCodeWire,
    rate: numericWire,
    effective_from: dateTimeWire.optional(),
    created_at: dateTimeWire,
  })
  .describe('Describes currency basis supported by billing system.')

export const createCostBasisRequestWire = z
  .strictObject({
    fiat_code: currencyCodeWire,
    rate: numericWire,
    effective_from: dateTimeWire.optional(),
  })
  .describe('CostBasis create request.')

export const featureCostQueryRowWire = z
  .strictObject({
    usage: numericWire,
    cost: z
      .union([numericWire, z.null()])

      .describe(
        'The computed cost amount (usage × unit cost). Null when pricing is not available for the given combination of dimensions.',
      ),
    currency: currencyCodeWire,
    detail: z
      .string()
      .optional()

      .describe(
        'Detail message when cost amount is null, explaining why the cost could not be resolved.',
      ),
    from: dateTimeWire,
    to: dateTimeWire,
    dimensions: z
      .record(z.string(), z.string())

      .describe(
        'The dimensions the value is aggregated over. `subject` and `customer_id` are reserved dimensions.',
      ),
  })
  .describe('A row in the result of a feature cost query.')

export const resourceWire = z
  .strictObject({
    id: ulidWire,
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
    labels: labelsWire.optional(),
    created_at: dateTimeWire,
    updated_at: dateTimeWire,
    deleted_at: dateTimeWire.optional(),
  })
  .describe('Represents common fields of resources.')

export const resourceImmutableWire = z
  .strictObject({
    id: ulidWire,
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
    labels: labelsWire.optional(),
    created_at: dateTimeWire,
  })
  .describe('Represents common fields of immutable resources.')

export const queryFilterDateTimeWire = z
  .strictObject({
    gt: dateTimeWire.optional(),
    gte: dateTimeWire.optional(),
    lt: dateTimeWire.optional(),
    lte: dateTimeWire.optional(),
    get and() {
      return z
        .array(queryFilterDateTimeWire)
        .min(1)
        .max(10)
        .optional()
        .describe('Combines the provided filters with a logical AND.')
    },
    get or() {
      return z
        .array(queryFilterDateTimeWire)
        .min(1)
        .max(10)
        .optional()
        .describe('Combines the provided filters with a logical OR.')
    },
  })

  .describe(
    'A query filter for a time attribute. Operators are mutually exclusive, only one operator is allowed at a time.',
  )

export const cursorMetaWire = z
  .strictObject({
    page: cursorMetaPageWire,
  })
  .describe('Cursor pagination metadata.')

export const invalidParameterStandardWire = z
  .strictObject({
    field: z.string().describe('The name of the field that failed validation.'),
    rule: invalidRulesWire.optional(),
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

export const invalidParameterMinimumLengthWire = z
  .strictObject({
    field: z.string().describe('The name of the field that failed validation.'),
    rule: invalidParameterMinimumRuleWire,
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

export const invalidParameterMaximumLengthWire = z
  .strictObject({
    field: z.string().describe('The name of the field that failed validation.'),
    rule: invalidParameterMaximumRuleWire,
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

export const invalidParameterChoiceItemWire = z
  .strictObject({
    field: z.string().describe('The name of the field that failed validation.'),
    rule: invalidParameterChoiceRuleWire,
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

export const invalidParameterDependentItemWire = z
  .strictObject({
    field: z.string().describe('The name of the field that failed validation.'),
    rule: invalidParameterDependentRuleWire,
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

export const unauthorizedWire = baseErrorWire.describe('Unauthorized.')

export const forbiddenWire = baseErrorWire.describe('Forbidden.')

export const notFoundWire = baseErrorWire.describe('Not Found.')

export const goneWire = baseErrorWire.describe('Gone.')

export const conflictWire = baseErrorWire.describe('Conflict.')

export const payloadTooLargeWire = baseErrorWire.describe('Payload Too Large.')

export const unsupportedMediaTypeWire = baseErrorWire.describe(
  'Unsupported Media Type.',
)

export const unprocessableContentWire = baseErrorWire.describe(
  'Unprocessable Content.',
)

export const tooManyRequestsWire = baseErrorWire.describe('Too Many Requests.')

export const internalWire = baseErrorWire.describe('Internal Server Error.')

export const notImplementedWire = baseErrorWire.describe('Not Implemented.')

export const notAvailableWire = baseErrorWire.describe('Not Available.')

export const createCreditGrantFiltersWire = z
  .strictObject({
    features: z
      .array(resourceKeyWire)
      .optional()

      .describe(
        'Limit the credit grant to specific features. If no features are specified, the credit grant can be used for any feature.',
      ),
  })
  .describe('Filters for the credit grant.')

export const creditGrantFiltersWire = z
  .strictObject({
    features: z
      .array(resourceKeyWire)
      .optional()

      .describe(
        'Limit the credit grant to specific features. If no features are specified, the credit grant can be used for any feature.',
      ),
  })
  .describe('Filters for the credit grant.')

export const upsertPlanAddonRequestWire = z
  .strictObject({
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
    labels: labelsWire.optional(),
    from_plan_phase: resourceKeyWire,
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

export const resourceWithKeyWire = z
  .strictObject({
    id: ulidWire,
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
    labels: labelsWire.optional(),
    created_at: dateTimeWire,
    updated_at: dateTimeWire,
    deleted_at: dateTimeWire.optional(),
    key: resourceKeyWire,
  })
  .describe('Represents common fields of resources with a key.')

export const ulidOrResourceKeyWire = z
  .union([ulidWire, resourceKeyWire])
  .describe('ULID ID or Resource Key.')

export const createMeterRequestWire = z
  .strictObject({
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
    labels: labelsWire.optional(),
    key: resourceKeyWire,
    aggregation: meterAggregationWire,
    event_type: z
      .string()
      .min(1)
      .describe('The event type to include in the aggregation.'),
    events_from: dateTimeWire.optional(),
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

export const meterWire = z
  .strictObject({
    id: ulidWire,
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
    labels: labelsWire.optional(),
    created_at: dateTimeWire,
    updated_at: dateTimeWire,
    deleted_at: dateTimeWire.optional(),
    key: resourceKeyWire,
    aggregation: meterAggregationWire,
    event_type: z
      .string()
      .min(1)
      .describe('The event type to include in the aggregation.'),
    events_from: dateTimeWire.optional(),
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

export const paginatedMetaWire = z
  .strictObject({
    page: pageMetaWire,
  })
  .describe('Pagination metadata.')

export const queryFilterStringMapItemWire = z
  .strictObject({
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
      .array(queryFilterStringWire)
      .min(1)
      .max(10)
      .optional()
      .describe('Combines the provided filters with a logical AND.'),
    or: z
      .array(queryFilterStringWire)
      .min(1)
      .max(10)
      .optional()
      .describe('Combines the provided filters with a logical OR.'),
  })

  .describe(
    'A query filter for an item in a string map attribute. Operators are mutually exclusive, only one operator is allowed at a time.',
  )

export const ulidOrExternalResourceKeyWire = z
  .union([ulidWire, externalResourceKeyWire])
  .describe('ULID ID or External Resource Key.')

export const customerKeyReferenceWire = z
  .strictObject({
    key: externalResourceKeyWire,
  })
  .describe('Customer reference by external key.')

export const customerUsageAttributionWire = z
  .strictObject({
    subject_keys: z
      .array(usageAttributionSubjectKeyWire)

      .describe(
        'The subjects that are attributed to the customer. Can be empty when no usage event subjects are associated with the customer.',
      ),
  })

  .describe(
    'Mapping to attribute metered usage to the customer. One customer can have zero or more subjects, but one subject can only belong to one customer.',
  )

export const addressWire = z
  .strictObject({
    country: countryCodeWire.optional(),
    postal_code: z.string().optional().describe('Postal code.'),
    state: z.string().optional().describe('State or province.'),
    city: z.string().optional().describe('City.'),
    line1: z.string().optional().describe('First line of the address.'),
    line2: z.string().optional().describe('Second line of the address.'),
    phone_number: z.string().optional().describe('Phone number.'),
  })
  .describe('Address')

export const appStripeCreateCheckoutSessionCustomerUpdateWire = z
  .strictObject({
    address: appStripeCreateCheckoutSessionCustomerUpdateBehaviorWire
      .optional()
      .default('never'),
    name: appStripeCreateCheckoutSessionCustomerUpdateBehaviorWire
      .optional()
      .default('never'),
    shipping: appStripeCreateCheckoutSessionCustomerUpdateBehaviorWire
      .optional()
      .default('never'),
  })

  .describe(
    'Controls which customer fields can be updated by the checkout session.',
  )

export const appStripeCreateCheckoutSessionConsentCollectionPaymentMethodReuseAgreementWire =
  z
    .strictObject({
      position:
        appStripeCreateCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPositionWire.optional(),
    })
    .describe('Payment method reuse agreement configuration.')

export const appStripeCreateCheckoutSessionTaxIdCollectionWire = z
  .strictObject({
    enabled: z
      .boolean()
      .optional()
      .default(false)
      .describe('Enable tax ID collection during checkout. Defaults to false.'),
    required: appStripeCreateCheckoutSessionTaxIdCollectionRequiredWire
      .optional()
      .default('never'),
  })
  .describe('Tax ID collection configuration for checkout sessions.')

export const appStripeCreateCheckoutSessionResultWire = z
  .strictObject({
    customer_id: ulidWire,
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
    currency: currencyCodeWire.optional(),
    created_at: dateTimeWire,
    expires_at: dateTimeWire.optional(),
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
    mode: appStripeCheckoutSessionModeWire,
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

export const customerStripeCreateCustomerPortalSessionRequestWire = z
  .strictObject({
    stripe_options: appStripeCreateCustomerPortalSessionOptionsWire,
  })

  .describe(
    'Request to create a Stripe Customer Portal Session for the customer. Useful to redirect the customer to the Stripe Customer Portal to manage their payment methods, change their billing address and access their invoice history. Only returns URL if the customer billing profile is linked to a stripe app and customer.',
  )

export const entitlementAccessResultWire = z
  .strictObject({
    type: entitlementTypeWire,
    feature_key: resourceKeyWire,
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

export const createCreditGrantPurchaseWire = z
  .strictObject({
    currency: currencyCodeWire,
    per_unit_cost_basis: numericWire.optional().default('1.0'),
    availability_policy: creditAvailabilityPolicyWire
      .optional()
      .default('on_creation'),
  })
  .describe('Purchase and payment terms of the grant.')

export const rateCardMeteredEntitlementWire = z
  .strictObject({
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
    usage_period: iso8601DurationWire.optional(),
  })
  .describe('The entitlement template of a metered entitlement.')

export const recurringPeriodWire = z
  .strictObject({
    anchor: dateTimeWire,
    interval: iso8601DurationWire,
  })
  .describe('Recurring period with an anchor and an interval.')

export const creditGrantPurchaseWire = z
  .strictObject({
    currency: currencyCodeWire,
    per_unit_cost_basis: numericWire.optional().default('1.0'),
    amount: numericWire,
    availability_policy: creditAvailabilityPolicyWire
      .optional()
      .default('on_creation'),
    settlement_status: creditPurchasePaymentSettlementStatusWire.optional(),
  })
  .describe('Purchase and payment terms of the grant.')

export const updateCreditGrantExternalSettlementRequestWire = z
  .strictObject({
    status: creditPurchasePaymentSettlementStatusWire,
  })

  .describe(
    'Request body for updating the external payment settlement status of a credit grant.',
  )

export const listCreditGrantsParamsFilterWire = z
  .strictObject({
    status: creditGrantStatusWire.optional(),
    currency: currencyCodeWire.optional(),
    key: stringFieldFilterWire.optional(),
  })
  .describe('Filter options for listing credit grants.')

export const getCreditBalanceParamsFilterWire = z
  .strictObject({
    currency: stringFieldFilterExactWire.optional(),
    feature_key: stringFieldFilterWire.optional(),
  })
  .describe('Filter options for getting a credit balance.')

export const listChargesParamsFilterWire = z
  .strictObject({
    status: stringFieldFilterExactWire.optional(),
  })
  .describe('Filter options for listing charges.')

export const listPlansParamsFilterWire = z
  .strictObject({
    key: stringFieldFilterWire.optional(),
    name: stringFieldFilterWire.optional(),
    status: stringFieldFilterExactWire.optional(),
    currency: stringFieldFilterExactWire.optional(),
  })
  .describe('Filter options for listing plans.')

export const subscriptionCreateWire = z
  .strictObject({
    labels: labelsWire.optional(),
    settlement_mode: settlementModeWire.optional(),
    customer: z
      .strictObject({
        id: ulidWire.optional(),
        key: externalResourceKeyWire.optional(),
      })
      .describe('The customer to create the subscription for.'),
    plan: z
      .strictObject({
        id: ulidWire.optional(),
        key: resourceKeyWire.optional(),
        version: z
          .number()
          .int()
          .optional()

          .describe(
            'The plan version of the subscription, if any. If not provided, the latest version of the plan will be used.',
          ),
      })
      .describe('The plan reference of the subscription.'),
    billing_anchor: dateTimeWire.optional(),
  })
  .describe('Subscription create request.')

export const rateCardProrationConfigurationWire = z
  .strictObject({
    mode: rateCardProrationModeWire,
  })
  .describe('The proration configuration of the rate card.')

export const subscriptionWire = z
  .strictObject({
    id: ulidWire,
    labels: labelsWire.optional(),
    created_at: dateTimeWire,
    updated_at: dateTimeWire,
    deleted_at: dateTimeWire.optional(),
    customer_id: ulidWire,
    plan_id: ulidWire.optional(),
    billing_anchor: dateTimeWire,
    status: subscriptionStatusWire,
    settlement_mode: settlementModeWire.optional(),
  })
  .describe('Subscription.')

export const subscriptionEditTimingWire = z
  .union([subscriptionEditTimingEnumWire, dateTimeWire])

  .describe(
    'Subscription edit timing defined when the changes should take effect. If the provided configuration is not supported by the subscription, an error will be returned.',
  )

export const unitConfigWire = z
  .strictObject({
    operation: unitConfigOperationWire,
    conversion_factor: numericWire,
    rounding: unitConfigRoundingModeWire.optional().default('none'),
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

export const appCatalogItemWire = z
  .strictObject({
    type: appTypeWire,
    name: z.string().describe('Name of the app.'),
    description: z.string().describe('Description of the app.'),
  })

  .describe(
    'Available apps for billing integrations to connect with third-party services. Apps can have various capabilities like syncing data from or to external systems, integrating with third-party services for tax calculation, delivery of invoices, collection of payments, etc.',
  )

export const taxCodeAppMappingWire = z
  .strictObject({
    app_type: appTypeWire,
    tax_code: z.string().describe('Tax code.'),
  })
  .describe('Mapping of app types to tax codes.')

export const partyTaxIdentityWire = z
  .strictObject({
    code: taxIdentificationCodeWire.optional(),
  })

  .describe(
    'Identity stores the details required to identify an entity for tax purposes in a specific country.',
  )

export const workflowInvoicingSettingsWire = z
  .strictObject({
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
      workflowInvoicingSubscriptionEndProrationModeWire
        .optional()
        .default('bill_actual_period'),
  })
  .describe('Invoice settings for a billing workflow.')

export const workflowPaymentSettingsWire = z
  .discriminatedUnion('collection_method', [
    workflowPaymentChargeAutomaticallySettingsWire,
    workflowPaymentSendInvoiceSettingsWire,
  ])
  .describe('Payment settings for a billing workflow.')

export const invoiceValidationIssueWire = z
  .strictObject({
    code: z.string().describe('Machine-readable error code.'),
    message: z.string().describe('Human-readable description of the error.'),
    attributes: z
      .record(z.string(), z.unknown())
      .optional()
      .describe('Additional structured context.'),
    severity: invoiceValidationIssueSeverityWire,
    field: z
      .string()
      .optional()

      .describe(
        'JSON path to the field that caused this validation issue, if applicable. For example: `lines/0/rate_card/price`.',
      ),
  })

  .describe(
    'A validation issue found during invoice processing. Converges on the same structure used by plan and subscription validation errors: a machine-readable `code`, a human-readable `message`, optional structured `attributes`, plus a `severity` and optional `field` path.',
  )

export const invoiceAvailableActionsWire = z
  .strictObject({
    advance: invoiceAvailableActionDetailsWire.optional(),
    approve: invoiceAvailableActionDetailsWire.optional(),
    delete: invoiceAvailableActionDetailsWire.optional(),
    retry: invoiceAvailableActionDetailsWire.optional(),
    snapshot_quantities: invoiceAvailableActionDetailsWire.optional(),
  })

  .describe(
    'The set of state-transition actions available for an invoice in its current status. A field is present only when that action is permitted from the current state.',
  )

export const invoiceLineAmountDiscountWire = z
  .strictObject({
    id: ulidWire,
    reason: invoiceDiscountReasonWire,
    description: z
      .string()
      .optional()
      .describe('Optional human-readable description of the discount.'),
    external_references: invoiceLineExternalReferencesWire.optional(),
    amount: numericWire,
  })
  .describe('A monetary amount discount applied to an invoice line item.')

export const invoiceLineUsageDiscountWire = z
  .strictObject({
    id: ulidWire,
    reason: invoiceDiscountReasonWire,
    description: z
      .string()
      .optional()
      .describe('Optional human-readable description of the discount.'),
    external_references: invoiceLineExternalReferencesWire.optional(),
    quantity: numericWire,
  })
  .describe('A usage quantity discount applied to an invoice line item.')

export const invoiceLineBaseDiscountWire = z
  .strictObject({
    id: ulidWire,
    reason: invoiceDiscountReasonWire,
    description: z
      .string()
      .optional()
      .describe('Optional human-readable description of the discount.'),
    external_references: invoiceLineExternalReferencesWire.optional(),
  })
  .describe('Base fields shared by all invoice line item discounts.')

export const listCurrenciesParamsFilterWire = z
  .strictObject({
    type: currencyTypeWire.optional(),
    code: stringFieldFilterWire.optional(),
  })
  .describe('Filter options for listing currencies.')

export const currencyCustomWire = z
  .strictObject({
    id: ulidWire,
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
    code: currencyCodeCustomWire,
    created_at: dateTimeWire,
  })
  .describe('Describes custom currency.')

export const createCurrencyCustomRequestWire = z
  .strictObject({
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
    code: currencyCodeCustomWire,
  })
  .describe('CurrencyCustom create request.')

export const governanceQueryRequestWire = z
  .strictObject({
    include_credits: z
      .boolean()
      .optional()
      .default(false)

      .describe(
        'Whether to include credit balance availability for each resolved customer. When true, each feature evaluation includes credit balance checks. Defaults to `false`.',
      ),
    customer: governanceQueryRequestCustomersWire,
    feature: governanceQueryRequestFeaturesWire.optional(),
  })
  .describe('Query to evaluate feature access for a list of customers.')

export const governanceFeatureAccessReasonWire = z
  .strictObject({
    code: governanceFeatureAccessReasonCodeWire,
    message: z.string().describe('Human-readable description of the error.'),
    attributes: z
      .record(z.string(), z.unknown())
      .optional()
      .describe('Additional structured context.'),
  })
  .describe('Reason a feature is not accessible to a customer.')

export const governanceQueryErrorWire = z
  .strictObject({
    code: governanceQueryErrorCodeWire,
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

export const appCustomerDataWire = z
  .strictObject({
    stripe: appCustomerDataStripeWire.optional(),
    external_invoicing: appCustomerDataExternalInvoicingWire.optional(),
  })
  .describe('App customer data.')

export const upsertAppCustomerDataRequestWire = z
  .strictObject({
    stripe: appCustomerDataStripeWire.optional(),
    external_invoicing: appCustomerDataExternalInvoicingWire.optional(),
  })
  .describe('AppCustomerData upsert request.')

export const creditAdjustmentWire = z
  .strictObject({
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
    labels: labelsWire.optional(),
  })

  .describe(
    "A credit adjustment can be used to make manual adjustments to a customer's credit balance. Supported use-cases: - Usage correction",
  )

export const creditBalanceWire = z
  .strictObject({
    currency: billingCurrencyCodeWire,
    live: numericWire,
    settled: numericWire,
    pending: numericWire,
  })
  .describe('The credit balance by currency.')

export const createCreditAdjustmentRequestWire = z
  .strictObject({
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
    labels: labelsWire.optional(),
    currency: billingCurrencyCodeWire,
    amount: numericWire,
  })
  .describe('CreditAdjustment create request.')

export const listCreditTransactionsParamsFilterWire = z
  .strictObject({
    type: creditTransactionTypeWire.optional(),
    currency: billingCurrencyCodeWire.optional(),
    feature_key: stringFieldFilterWire.optional(),
  })
  .describe('Filter options for listing credit transactions.')

export const creditTransactionWire = z
  .strictObject({
    id: ulidWire,
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
    labels: labelsWire.optional(),
    created_at: dateTimeWire,
    booked_at: dateTimeWire,
    type: creditTransactionTypeWire,
    currency: billingCurrencyCodeWire,
    amount: numericWire,
    available_balance: z
      .strictObject({
        before: numericWire,
        after: numericWire,
      })
      .describe('The available balance before and after the transaction.'),
  })

  .describe(
    "A credit transaction represents a single credit movement on the customer's balance. Credit transactions are immutable.",
  )

export const priceTierWire = z
  .strictObject({
    up_to_amount: numericWire.optional(),
    flat_price: priceFlatWire.optional(),
    unit_price: priceUnitWire.optional(),
  })

  .describe(
    'A price tier used in graduated and volume pricing. At least one price component (flat_price or unit_price) must be set. When UnitConfig is present on the rate card, up_to_amount is expressed in converted billing units.',
  )

export const chargeTotalsWire = z
  .strictObject({
    booked: totalsWire,
    realtime: totalsWire.optional(),
  })

  .describe(
    'The totals of a change. RealTime is only expanded when the `real_time_usage` expand is used.',
  )

export const featureLlmUnitCostWire = z
  .strictObject({
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
    token_type: featureLlmTokenTypeWire.optional(),
    pricing: featureLlmUnitCostPricingWire.optional(),
  })

  .describe(
    'LLM cost lookup configuration. Each dimension (provider, model, token type) can be specified as either a static value or a meter group-by property name (mutually exclusive).',
  )

export const llmCostPriceWire = z
  .strictObject({
    id: ulidWire,
    provider: llmCostProviderWire,
    model: llmCostModelWire,
    pricing: llmCostModelPricingWire,
    currency: currencyCodeWire,
    source: llmCostPriceSourceWire,
    effective_from: dateTimeWire,
    effective_to: dateTimeWire.optional(),
    created_at: dateTimeWire,
    updated_at: dateTimeWire,
  })

  .describe(
    'An LLM cost price record, representing the cost per token for a specific model from a specific provider.',
  )

export const llmCostOverrideCreateWire = z
  .strictObject({
    provider: z.string().describe('Provider/vendor of the model.'),
    model_id: z.string().describe('Canonical model identifier.'),
    model_name: z.string().optional().describe('Human-readable model name.'),
    pricing: llmCostModelPricingWire,
    currency: currencyCodeWire,
    effective_from: dateTimeWire,
    effective_to: dateTimeWire.optional(),
  })

  .describe(
    'Input for creating a per-namespace price override. Unique per provider, model and currency. If an override already exists for the given provider, model and currency, it will be updated. If an override does not exist, it will be created.',
  )

export const listCustomersParamsFilterWire = z
  .strictObject({
    key: stringFieldFilterWire.optional(),
    name: stringFieldFilterWire.optional(),
    primary_email: stringFieldFilterWire.optional(),
    usage_attribution_subject_key: stringFieldFilterWire.optional(),
    plan_key: stringFieldFilterWire.optional(),
    billing_profile_id: ulidFieldFilterWire.optional(),
  })
  .describe('Filter options for listing customers.')

export const listSubscriptionsParamsFilterWire = z
  .strictObject({
    id: ulidFieldFilterWire.optional(),
    customer_id: ulidFieldFilterWire.optional(),
    status: stringFieldFilterExactWire.optional(),
    plan_id: ulidFieldFilterWire.optional(),
    plan_key: stringFieldFilterExactWire.optional(),
  })
  .describe('Filter options for listing subscriptions.')

export const listFeatureParamsFilterWire = z
  .strictObject({
    meter_id: ulidFieldFilterWire.optional(),
    key: stringFieldFilterWire.optional(),
    name: stringFieldFilterWire.optional(),
  })
  .describe('Filter options for listing features.')

export const listAddonsParamsFilterWire = z
  .strictObject({
    id: ulidFieldFilterWire.optional(),
    key: stringFieldFilterWire.optional(),
    name: stringFieldFilterWire.optional(),
    status: stringFieldFilterExactWire.optional(),
    currency: stringFieldFilterExactWire.optional(),
  })
  .describe('Filter options for listing add-ons.')

export const createCreditGrantTaxConfigWire = z
  .strictObject({
    behavior: taxBehaviorWire.optional(),
    tax_code: createResourceReferenceWire.optional(),
  })

  .describe(
    'Tax configuration for a credit grant. Tax configuration should be provided to ensure correct revenue recognition, including for externally funded grants.',
  )

export const creditGrantTaxConfigWire = z
  .strictObject({
    behavior: taxBehaviorWire.optional(),
    tax_code: taxCodeReferenceWire.optional(),
  })

  .describe(
    'Tax configuration for a credit grant. Tax configuration should be provided to ensure correct revenue recognition, including for externally funded grants.',
  )

export const taxConfigWire = z
  .strictObject({
    behavior: taxBehaviorWire.optional(),
    stripe: taxConfigStripeWire.optional(),
    external_invoicing: taxConfigExternalInvoicingWire.optional(),
    tax_code_id: ulidWire.optional(),
    tax_code: taxCodeReferenceWire.optional(),
  })
  .describe('Set of provider specific tax configs.')

export const rateCardTaxConfigWire = z
  .strictObject({
    behavior: taxBehaviorWire.optional(),
    code: taxCodeReferenceWire,
  })
  .describe('The tax config of the rate card.')

export const organizationDefaultTaxCodesWire = z
  .strictObject({
    invoicing_tax_code: taxCodeReferenceWire,
    credit_grant_tax_code: taxCodeReferenceWire,
    created_at: dateTimeWire,
    updated_at: dateTimeWire,
  })

  .describe(
    'Organization-level default tax code references. Stores the default tax codes applied to specific billing contexts for this organization. Provisioned automatically when the organization is created.',
  )

export const updateOrganizationDefaultTaxCodesRequestWire = z
  .strictObject({
    invoicing_tax_code: taxCodeReferenceWire.optional(),
    credit_grant_tax_code: taxCodeReferenceWire.optional(),
  })
  .describe('OrganizationDefaultTaxCodes update request.')

export const planAddonWire = z
  .strictObject({
    id: ulidWire,
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
    labels: labelsWire.optional(),
    created_at: dateTimeWire,
    updated_at: dateTimeWire,
    deleted_at: dateTimeWire.optional(),
    addon: addonReferenceWire,
    from_plan_phase: resourceKeyWire,
    max_quantity: z
      .number()
      .int()
      .gte(1)
      .optional()

      .describe(
        'The maximum number of times the add-on can be purchased for the plan. For single-instance add-ons this field must be omitted. For multi-instance add-ons when omitted, unlimited quantity can be purchased.',
      ),
    validation_errors: z
      .array(productCatalogValidationErrorWire)
      .optional()
      .describe('List of validation errors.'),
  })

  .describe(
    'PlanAddon represents an association between a plan and an add-on, controlling which add-ons are available for purchase within a plan.',
  )

export const createPlanAddonRequestWire = z
  .strictObject({
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
    labels: labelsWire.optional(),
    addon: addonReferenceWire,
    from_plan_phase: resourceKeyWire,
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

export const profileAppReferencesWire = z
  .strictObject({
    tax: appReferenceWire,
    invoicing: appReferenceWire,
    payment: appReferenceWire,
  })
  .describe('References to the applications used by a billing profile.')

export const invoiceWorkflowAppsReferencesWire = z
  .strictObject({
    tax: appReferenceWire,
    invoicing: appReferenceWire,
    payment: appReferenceWire,
  })

  .describe(
    'BillingInvoiceWorkflowAppsReferences represents the references (id) to the apps used by a billing profile',
  )

export const listEventsParamsFilterWire = z
  .strictObject({
    id: stringFieldFilterWire.optional(),
    source: stringFieldFilterWire.optional(),
    subject: stringFieldFilterWire.optional(),
    type: stringFieldFilterWire.optional(),
    customer_id: ulidFieldFilterWire.optional(),
    time: dateTimeFieldFilterWire.optional(),
    ingested_at: dateTimeFieldFilterWire.optional(),
    stored_at: dateTimeFieldFilterWire.optional(),
  })
  .describe('Filter options for listing ingested events.')

export const resourceFiltersWire = z
  .strictObject({
    name: stringFieldFilterWire.optional(),
    labels: labelsFieldFilterWire.optional(),
    public_labels: labelsFieldFilterWire.optional(),
    created_at: dateTimeFieldFilterWire.optional(),
    updated_at: dateTimeFieldFilterWire.optional(),
    deleted_at: dateTimeFieldFilterWire.optional(),
  })
  .describe('Resource filters.')

export const fieldFiltersWire = z
  .strictObject({
    boolean: booleanFieldFilterWire.optional(),
    numeric: numericFieldFilterWire.optional(),
    string: stringFieldFilterWire.optional(),
    string_exact: stringFieldFilterExactWire.optional(),
    ulid: ulidFieldFilterWire.optional(),
    datetime: dateTimeFieldFilterWire.optional(),
    labels: labelsFieldFilterWire.optional(),
  })
  .describe('Field filters with all supported types.')

export const ingestedEventWire = z
  .strictObject({
    event: eventWire,
    customer: customerReferenceWire.optional(),
    ingested_at: dateTimeWire,
    stored_at: dateTimeWire,
    validation_errors: z
      .array(ingestedEventValidationErrorWire)
      .optional()
      .describe('The validation errors of the ingested event.'),
  })
  .describe('An ingested metering event with ingestion metadata.')

export const meterQueryResultWire = z
  .strictObject({
    from: dateTimeWire.optional(),
    to: dateTimeWire.optional(),
    data: z
      .array(meterQueryRowWire)

      .describe(
        'The usage data. If no data is available, an empty array is returned.',
      ),
  })
  .describe('Meter query result.')

export const featureCostQueryResultWire = z
  .strictObject({
    from: dateTimeWire.optional(),
    to: dateTimeWire.optional(),
    data: z.array(featureCostQueryRowWire).describe('The cost data rows.'),
  })
  .describe('Result of a feature cost query.')

export const invalidParameterWire = z
  .union([
    invalidParameterStandardWire,
    invalidParameterMinimumLengthWire,
    invalidParameterMaximumLengthWire,
    invalidParameterChoiceItemWire,
    invalidParameterDependentItemWire,
  ])
  .describe('A parameter that failed validation.')

export const meterPagePaginatedResponseWire = z
  .strictObject({
    data: z.array(meterWire),
    meta: paginatedMetaWire,
  })
  .describe('Page paginated response.')

export const costBasisPagePaginatedResponseWire = z
  .strictObject({
    data: z.array(costBasisWire),
    meta: paginatedMetaWire,
  })
  .describe('Page paginated response.')

export const meterQueryFiltersWire = z
  .strictObject({
    dimensions: z
      .record(z.string(), queryFilterStringMapItemWire)
      .optional()

      .describe(
        'Filters to apply to the dimensions of the query. For `subject` and `customer_id` only equals ("eq", "in") comparisons are supported.',
      ),
  })
  .describe('Filters to apply to a meter query.')

export const featureMeterReferenceWire = z
  .strictObject({
    id: ulidWire,
    filters: z
      .record(z.string(), queryFilterStringMapItemWire)
      .optional()
      .describe('Filters to apply to the dimensions of the meter.'),
  })
  .describe('Reference to a meter associated with a feature.')

export const createCustomerRequestWire = z
  .strictObject({
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
    labels: labelsWire.optional(),
    key: externalResourceKeyWire,
    usage_attribution: customerUsageAttributionWire.optional(),
    primary_email: z
      .string()
      .optional()
      .describe('The primary email address of the customer.'),
    currency: currencyCodeWire.optional(),
    billing_address: addressWire.optional(),
  })
  .describe('Customer create request.')

export const customerWire = z
  .strictObject({
    id: ulidWire,
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
    labels: labelsWire.optional(),
    created_at: dateTimeWire,
    updated_at: dateTimeWire,
    deleted_at: dateTimeWire.optional(),
    key: externalResourceKeyWire,
    usage_attribution: customerUsageAttributionWire.optional(),
    primary_email: z
      .string()
      .optional()
      .describe('The primary email address of the customer.'),
    currency: currencyCodeWire.optional(),
    billing_address: addressWire.optional(),
  })

  .describe(
    'Customers can be individuals or organizations that can subscribe to plans and have access to features.',
  )

export const upsertCustomerRequestWire = z
  .strictObject({
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
    labels: labelsWire.optional(),
    usage_attribution: customerUsageAttributionWire.optional(),
    primary_email: z
      .string()
      .optional()
      .describe('The primary email address of the customer.'),
    currency: currencyCodeWire.optional(),
    billing_address: addressWire.optional(),
  })
  .describe('Customer upsert request.')

export const partyAddressesWire = z
  .strictObject({
    billing_address: addressWire,
  })
  .describe('A collection of addresses for the party.')

export const invoiceCustomerWire = z
  .strictObject({
    id: ulidWire,
    name: z
      .string()
      .min(1)
      .max(256)
      .describe('Display name of the resource. Between 1 and 256 characters.'),
    usage_attribution: customerUsageAttributionWire.optional(),
    billing_address: addressWire.optional(),
    key: externalResourceKeyWire.optional(),
  })

  .describe(
    "Snapshot of the customer's information at the time the invoice was issued.",
  )

export const appStripeCreateCheckoutSessionConsentCollectionWire = z
  .strictObject({
    payment_method_reuse_agreement:
      appStripeCreateCheckoutSessionConsentCollectionPaymentMethodReuseAgreementWire.optional(),
    promotions:
      appStripeCreateCheckoutSessionConsentCollectionPromotionsWire.optional(),
    terms_of_service:
      appStripeCreateCheckoutSessionConsentCollectionTermsOfServiceWire.optional(),
  })
  .describe('Checkout Session consent collection configuration.')

export const listCustomerEntitlementAccessResponseDataWire = z
  .strictObject({
    data: z
      .array(entitlementAccessResultWire)
      .describe('The list of entitlement access results.'),
  })
  .describe('List customer entitlement access response data.')

export const rateCardEntitlementWire = z
  .discriminatedUnion('type', [
    rateCardMeteredEntitlementWire,
    rateCardStaticEntitlementWire,
    rateCardBooleanEntitlementWire,
  ])

  .describe(
    'Entitlement template configured on a rate card. The feature is taken from the rate card itself, so it is omitted here.',
  )

export const workflowCollectionAlignmentAnchoredWire = z
  .strictObject({
    type: z.literal('anchored').describe('The type of alignment.'),
    recurring_period: recurringPeriodWire,
  })

  .describe(
    'BillingWorkflowCollectionAlignmentAnchored specifies the alignment for collecting the pending line items into an invoice.',
  )

export const subscriptionPagePaginatedResponseWire = z
  .strictObject({
    data: z.array(subscriptionWire),
    meta: paginatedMetaWire,
  })
  .describe('Page paginated response.')

export const subscriptionChangeResponseWire = z
  .strictObject({
    current: subscriptionWire,
    next: subscriptionWire,
  })
  .describe('Response for changing a subscription.')

export const subscriptionCancelWire = z
  .strictObject({
    timing: subscriptionEditTimingWire.optional().default('immediate'),
  })
  .describe('Request for canceling a subscription.')

export const subscriptionChangeWire = z
  .strictObject({
    labels: labelsWire.optional(),
    settlement_mode: settlementModeWire.optional(),
    customer: z
      .strictObject({
        id: ulidWire.optional(),
        key: externalResourceKeyWire.optional(),
      })
      .describe('The customer to create the subscription for.'),
    plan: z
      .strictObject({
        id: ulidWire.optional(),
        key: resourceKeyWire.optional(),
        version: z
          .number()
          .int()
          .optional()

          .describe(
            'The plan version of the subscription, if any. If not provided, the latest version of the plan will be used.',
          ),
      })
      .describe('The plan reference of the subscription.'),
    billing_anchor: dateTimeWire.optional(),
    timing: subscriptionEditTimingWire,
  })
  .describe('Request for changing a subscription.')

export const createSubscriptionAddonRequestWire = z
  .strictObject({
    labels: labelsWire.optional(),
    addon: addonReferenceWire,
    quantity: z
      .number()
      .int()
      .gte(1)

      .describe(
        'The quantity of the add-on. Always 1 for single instance add-ons.',
      ),
    timing: subscriptionEditTimingWire,
  })
  .describe('SubscriptionAddon create request.')

export const invoiceUsageQuantityDetailWire = z
  .strictObject({
    raw_quantity: numericWire,
    converted_quantity: numericWire,
    invoiced_quantity: numericWire,
    display_unit: z
      .string()
      .optional()

      .describe('The display unit label (e.g., "GB", "hours", "M tokens").'),
    applied_unit_config: unitConfigWire,
  })

  .describe(
    'Usage quantity details on an invoice line item when UnitConfig is in effect. Provides the full audit trail from raw meter output to the invoiced amount.',
  )

export const appStripeWire = z
  .strictObject({
    id: ulidWire,
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
    labels: labelsWire.optional(),
    created_at: dateTimeWire,
    updated_at: dateTimeWire,
    deleted_at: dateTimeWire.optional(),
    type: z.literal('stripe').describe('The app type.'),
    definition: appCatalogItemWire,
    status: appStatusWire,
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
  })
  .describe('Stripe app.')

export const appSandboxWire = z
  .strictObject({
    id: ulidWire,
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
    labels: labelsWire.optional(),
    created_at: dateTimeWire,
    updated_at: dateTimeWire,
    deleted_at: dateTimeWire.optional(),
    type: z.literal('sandbox').describe('The app type.'),
    definition: appCatalogItemWire,
    status: appStatusWire,
  })
  .describe('Sandbox app can be used for testing billing features.')

export const appExternalInvoicingWire = z
  .strictObject({
    id: ulidWire,
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
    labels: labelsWire.optional(),
    created_at: dateTimeWire,
    updated_at: dateTimeWire,
    deleted_at: dateTimeWire.optional(),
    type: z.literal('external_invoicing').describe('The app type.'),
    definition: appCatalogItemWire,
    status: appStatusWire,
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

export const createTaxCodeRequestWire = z
  .strictObject({
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
    labels: labelsWire.optional(),
    key: resourceKeyWire,
    app_mappings: z
      .array(taxCodeAppMappingWire)
      .describe('Mapping of app types to tax codes.'),
  })
  .describe('TaxCode create request.')

export const taxCodeWire = z
  .strictObject({
    id: ulidWire,
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
    labels: labelsWire.optional(),
    created_at: dateTimeWire,
    updated_at: dateTimeWire,
    deleted_at: dateTimeWire.optional(),
    key: resourceKeyWire,
    app_mappings: z
      .array(taxCodeAppMappingWire)
      .describe('Mapping of app types to tax codes.'),
  })
  .describe('Tax codes by provider.')

export const upsertTaxCodeRequestWire = z
  .strictObject({
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
    labels: labelsWire.optional(),
    app_mappings: z
      .array(taxCodeAppMappingWire)
      .describe('Mapping of app types to tax codes.'),
  })
  .describe('TaxCode upsert request.')

export const invoiceWorkflowWire = z
  .strictObject({
    invoicing: invoiceWorkflowInvoicingSettingsWire.optional(),
    payment: workflowPaymentSettingsWire.optional(),
  })

  .describe(
    'Invoice-level snapshot of the workflow configuration. Contains only the settings that are meaningful for an already-created invoice: invoicing behaviour and payment settings. Collection alignment and tax policy are gather-time / profile-wide concerns and are not included.',
  )

export const invoiceStatusDetailsWire = z
  .strictObject({
    immutable: z
      .boolean()

      .describe(
        'Whether the invoice is immutable (i.e. cannot be modified or deleted).',
      ),
    failed: z.boolean().describe('Whether the invoice is in a failed state.'),
    extended_status: z
      .string()

      .describe(
        'Fine-grained internal status string providing additional workflow detail beyond the top-level status enum.',
      ),
    available_actions: invoiceAvailableActionsWire,
  })
  .describe('Detailed status information for a standard invoice.')

export const invoiceLineDiscountsWire = z
  .strictObject({
    amount: z
      .array(invoiceLineAmountDiscountWire)
      .optional()

      .describe(
        'Monetary amount discounts (e.g. from maximum spend commitments).',
      ),
    usage: z
      .array(invoiceLineUsageDiscountWire)
      .optional()
      .describe('Usage quantity discounts (e.g. free tier usage allowances).'),
  })
  .describe('Discounts applied to an invoice line item.')

export const currencyWire = z
  .discriminatedUnion('type', [currencyFiatWire, currencyCustomWire])
  .describe('Fiat or custom currency.')

export const governanceFeatureAccessWire = z
  .strictObject({
    has_access: z
      .boolean()

      .describe(
        'Whether the customer currently has access to the feature. `true` for boolean and static entitlements that are available, and for metered entitlements with remaining balance. `false` when the feature is unavailable, the usage limit has been reached, or (when applicable) credits have been exhausted.',
      ),
    reason: governanceFeatureAccessReasonWire.optional(),
  })
  .describe('Access status for a single feature.')

export const customerDataWire = z
  .strictObject({
    billing_profile: profileReferenceWire.optional(),
    app_data: appCustomerDataWire.optional(),
  })
  .describe('Billing customer data.')

export const upsertCustomerBillingDataRequestWire = z
  .strictObject({
    billing_profile: profileReferenceWire.optional(),
    app_data: appCustomerDataWire.optional(),
  })
  .describe('CustomerBillingData upsert request.')

export const creditBalancesWire = z
  .strictObject({
    retrieved_at: dateTimeWire,
    balances: z
      .array(creditBalanceWire)
      .describe('The balances by currencies.'),
  })
  .describe('The balances of the credits of a customer.')

export const creditTransactionPaginatedResponseWire = z
  .strictObject({
    data: z.array(creditTransactionWire),
    meta: cursorMetaWire,
  })
  .describe('Cursor paginated response.')

export const priceGraduatedWire = z
  .strictObject({
    type: z.literal('graduated').describe('The type of the price.'),
    tiers: z
      .array(priceTierWire)
      .min(1)

      .describe(
        'The tiers of the graduated price. At least one tier is required.',
      ),
  })

  .describe(
    "Graduated tiered price. Each tier's rate applies only to the usage within that tier. Pricing can change as cumulative usage crosses tier boundaries. When UnitConfig is present on the rate card, tier boundaries (up_to_amount) are expressed in converted billing units.",
  )

export const priceVolumeWire = z
  .strictObject({
    type: z.literal('volume').describe('The type of the price.'),
    tiers: z
      .array(priceTierWire)
      .min(1)

      .describe(
        'The tiers of the volume price. At least one tier is required.',
      ),
  })

  .describe(
    'Volume tiered price. The maximum quantity within a period determines the per-unit price for all units in that period. When UnitConfig is present on the rate card, tier boundaries (up_to_amount) are expressed in converted billing units.',
  )

export const featureUnitCostWire = z
  .discriminatedUnion('type', [
    featureManualUnitCostWire,
    featureLlmUnitCostWire,
  ])

  .describe(
    'Per-unit cost configuration for a feature. Either a fixed manual amount or a dynamic LLM cost lookup.',
  )

export const pricePagePaginatedResponseWire = z
  .strictObject({
    data: z.array(llmCostPriceWire),
    meta: paginatedMetaWire,
  })
  .describe('Page paginated response.')

export const createCreditGrantRequestWire = z
  .strictObject({
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
    labels: createLabelsWire.optional(),
    funding_method: creditFundingMethodWire,
    currency: createCurrencyCodeWire,
    amount: numericWire,
    purchase: createCreditGrantPurchaseWire.optional(),
    tax_config: createCreditGrantTaxConfigWire.optional(),
    filters: createCreditGrantFiltersWire.optional(),
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
    effective_at: dateTimeWire.optional(),
    expires_after: iso8601DurationWire.optional(),
    key: externalResourceKeyWire.optional(),
  })
  .describe('CreditGrant create request.')

export const creditGrantWire = z
  .strictObject({
    id: ulidWire,
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
    labels: labelsWire.optional(),
    created_at: dateTimeWire,
    updated_at: dateTimeWire,
    deleted_at: dateTimeWire.optional(),
    funding_method: creditFundingMethodWire,
    currency: billingCurrencyCodeWire,
    amount: numericWire,
    purchase: creditGrantPurchaseWire.optional(),
    tax_config: creditGrantTaxConfigWire.optional(),
    invoice: creditGrantInvoiceReferenceWire.optional(),
    filters: creditGrantFiltersWire.optional(),
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
    effective_at: dateTimeWire.optional(),
    key: externalResourceKeyWire.optional(),
    expires_at: dateTimeWire.optional(),
    voided_at: dateTimeWire.optional(),
    status: creditGrantStatusWire,
  })

  .describe(
    'A credit grant allocates credits to a customer. Credits are drawn down against charges according to the settlement mode configured on the rate card.',
  )

export const createChargeFlatFeeRequestWire = z
  .strictObject({
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
    labels: labelsWire.optional(),
    type: z.literal('flat_fee').describe('The type of the charge.'),
    currency: currencyCodeWire,
    invoice_at: dateTimeWire,
    service_period: closedPeriodWire,
    unique_reference_id: z
      .string()
      .optional()
      .describe('Unique reference ID of the charge.'),
    settlement_mode: settlementModeWire,
    tax_config: taxConfigWire.optional(),
    payment_term: pricePaymentTermWire,
    discounts: chargeFlatFeeDiscountsWire.optional(),
    feature_key: z
      .string()
      .optional()
      .describe('The feature associated with the charge, when applicable.'),
    proration_configuration: rateCardProrationConfigurationWire,
    amount_before_proration: currencyAmountWire,
    full_service_period: closedPeriodWire.optional(),
    billing_period: closedPeriodWire.optional(),
  })
  .describe('Flat fee charge create request.')

export const workflowTaxSettingsWire = z
  .strictObject({
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
    default_tax_config: taxConfigWire.optional(),
  })
  .describe('Tax settings for a billing workflow.')

export const planAddonPagePaginatedResponseWire = z
  .strictObject({
    data: z.array(planAddonWire),
    meta: paginatedMetaWire,
  })
  .describe('Page paginated response.')

export const ingestedEventPaginatedResponseWire = z
  .strictObject({
    data: z.array(ingestedEventWire),
    meta: cursorMetaWire,
  })
  .describe('Cursor paginated response.')

export const invalidParametersWire = z
  .array(invalidParameterWire)
  .min(1)
  .describe('The list of parameters that failed validation.')

export const meterQueryRequestWire = z
  .strictObject({
    from: dateTimeWire.optional(),
    to: dateTimeWire.optional(),
    granularity: meterQueryGranularityWire.optional(),
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
    filters: meterQueryFiltersWire.optional(),
  })
  .describe('A meter query request.')

export const customerPagePaginatedResponseWire = z
  .strictObject({
    data: z.array(customerWire),
    meta: paginatedMetaWire,
  })
  .describe('Page paginated response.')

export const partyWire = z
  .strictObject({
    id: z.string().optional().describe('Unique identifier for the party.'),
    key: externalResourceKeyWire.optional(),
    name: z
      .string()
      .optional()
      .describe('Legal name or representation of the party.'),
    tax_id: partyTaxIdentityWire.optional(),
    addresses: partyAddressesWire.optional(),
  })
  .describe('Party represents a person or business entity.')

export const supplierWire = z
  .strictObject({
    id: z.string().optional().describe('Unique identifier for the party.'),
    name: z
      .string()
      .optional()
      .describe('Legal name or representation of the party.'),
    tax_id: partyTaxIdentityWire.optional(),
    addresses: partyAddressesWire.optional(),
  })

  .describe(
    "Snapshot of the supplier's information at the time the invoice was issued. Structurally a read-only subset of `BillingParty` (the type configured on the billing profile), so the snapshot stays aligned with the source. `key` is omitted because it is not part of the snapshotted supplier data.",
  )

export const appStripeCreateCheckoutSessionRequestOptionsWire = z
  .strictObject({
    billing_address_collection:
      appStripeCreateCheckoutSessionBillingAddressCollectionWire
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
    customer_update:
      appStripeCreateCheckoutSessionCustomerUpdateWire.optional(),
    consent_collection:
      appStripeCreateCheckoutSessionConsentCollectionWire.optional(),
    currency: currencyCodeWire.optional(),
    custom_text: appStripeCheckoutSessionCustomTextParamsWire.optional(),
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
    ui_mode: appStripeCheckoutSessionUiModeWire.optional().default('hosted'),
    payment_method_types: z
      .array(z.string())
      .optional()

      .describe(
        'List of payment method types to enable (e.g., "card", "us_bank_account"). If not specified, Stripe enables all relevant payment methods.',
      ),
    redirect_on_completion:
      appStripeCreateCheckoutSessionRedirectOnCompletionWire.optional(),
    tax_id_collection:
      appStripeCreateCheckoutSessionTaxIdCollectionWire.optional(),
  })

  .describe(
    "Configuration options for creating a Stripe Checkout Session. Based on Stripe's [Checkout Session API parameters](https://docs.stripe.com/api/checkout/sessions/create).",
  )

export const workflowCollectionAlignmentWire = z
  .discriminatedUnion('type', [
    workflowCollectionAlignmentSubscriptionWire,
    workflowCollectionAlignmentAnchoredWire,
  ])

  .describe(
    'The alignment for collecting the pending line items into an invoice. Defaults to subscription, which means that we are to create a new invoice every time the a subscription period starts (for in advance items) or ends (for in arrears items).',
  )

export const appWire = z
  .discriminatedUnion('type', [
    appStripeWire,
    appSandboxWire,
    appExternalInvoicingWire,
  ])
  .describe('Installed application.')

export const taxCodePagePaginatedResponseWire = z
  .strictObject({
    data: z.array(taxCodeWire),
    meta: paginatedMetaWire,
  })
  .describe('Page paginated response.')

export const invoiceWorkflowSettingsWire = z
  .strictObject({
    apps: invoiceWorkflowAppsReferencesWire.optional(),
    source_billing_profile: profileReferenceWire,
    workflow: invoiceWorkflowWire,
  })

  .describe(
    'Snapshot of the billing workflow configuration captured at invoice creation.',
  )

export const invoiceDetailedLineWire = z
  .strictObject({
    id: ulidWire,
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
    labels: labelsWire.optional(),
    created_at: dateTimeWire,
    updated_at: dateTimeWire,
    deleted_at: dateTimeWire.optional(),
    service_period: closedPeriodWire,
    totals: totalsWire,
    category: invoiceDetailedLineCostCategoryWire.default('regular'),
    discounts: invoiceLineDiscountsWire.optional(),
    credits_applied: z
      .array(invoiceLineCreditsAppliedWire)
      .optional()
      .describe('Credit applied to this detailed line.'),
    external_references: invoiceLineExternalReferencesWire.optional(),
    quantity: numericWire,
    unit_price: numericWire,
  })

  .describe(
    'A detailed (child) sub-line belonging to a parent invoice line. Detailed lines represent the individual flat-fee components that make up a usage-based parent line after quantity snapshotting.',
  )

export const currencyPagePaginatedResponseWire = z
  .strictObject({
    data: z.array(currencyWire),
    meta: paginatedMetaWire,
  })
  .describe('Page paginated response.')

export const governanceQueryResultWire = z
  .strictObject({
    matched: z
      .array(z.string())

      .describe(
        'The list of identifiers from the request that resolved to this customer. Each entry is either the customer `key` or one of its usage-attribution subject `key`s. Duplicate or aliased identifiers that resolve to the same customer collapse to a single result entry, with every requested identifier listed here.',
      ),
    customer: customerWire,
    features: z
      .record(z.string(), governanceFeatureAccessWire)

      .describe(
        'Map of features with their access status. Map keys are the feature keys requested in `feature.keys`, or every feature `key` available in the organization when the feature filter was omitted.',
      ),
    updated_at: dateTimeWire,
  })
  .describe('Access evaluation result for a single resolved customer.')

export const priceWire = z
  .discriminatedUnion('type', [
    priceFreeWire,
    priceFlatWire,
    priceUnitWire,
    priceGraduatedWire,
    priceVolumeWire,
  ])
  .describe('Price.')

export const priceUsageBasedWire = z
  .discriminatedUnion('type', [
    priceUnitWire,
    priceGraduatedWire,
    priceVolumeWire,
  ])

  .describe(
    'Usage-based price types that can appear on a usage-based rate card. When UnitConfig is present on the rate card, these price types operate on billing units (i.e. post-conversion quantities), not raw metered units.',
  )

export const featureWire = z
  .strictObject({
    id: ulidWire,
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
    labels: labelsWire.optional(),
    created_at: dateTimeWire,
    updated_at: dateTimeWire,
    deleted_at: dateTimeWire.optional(),
    key: resourceKeyWire,
    meter: featureMeterReferenceWire.optional(),
    unit_cost: featureUnitCostWire.optional(),
  })
  .describe('A capability or billable dimension offered by a provider.')

export const createFeatureRequestWire = z
  .strictObject({
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
    labels: labelsWire.optional(),
    key: resourceKeyWire,
    meter: featureMeterReferenceWire.optional(),
    unit_cost: featureUnitCostWire.optional(),
  })
  .describe('Feature create request.')

export const updateFeatureRequestWire = z
  .strictObject({
    unit_cost: z
      .union([featureUnitCostWire, z.null()])
      .optional()

      .describe(
        'Optional per-unit cost configuration. Use "manual" for a fixed per-unit cost, or "llm" to look up cost from the LLM cost database based on meter group-by properties. Set to `null` to clear the existing unit cost; omit to leave it unchanged.',
      ),
  })

  .describe(
    'Request body for updating a feature. Currently only the unit_cost field can be updated.',
  )

export const creditGrantPagePaginatedResponseWire = z
  .strictObject({
    data: z.array(creditGrantWire),
    meta: paginatedMetaWire,
  })
  .describe('Page paginated response.')

export const badRequestWire = z
  .intersection(
    baseErrorWire,
    z.strictObject({
      invalid_parameters: invalidParametersWire,
    }),
  )
  .describe('Bad Request.')

export const invoiceBaseWire = z
  .strictObject({
    id: ulidWire,
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labelsWire.optional(),
    created_at: dateTimeWire,
    updated_at: dateTimeWire,
    deleted_at: dateTimeWire.optional(),
    number: invoiceNumberWire,
    currency: currencyCodeWire,
    supplier: supplierWire,
    customer: invoiceCustomerWire,
    totals: totalsWire,
    service_period: closedPeriodWire,
    validation_issues: z
      .array(invoiceValidationIssueWire)
      .optional()

      .describe(
        'Validation issues found during invoice processing. Present only when there are one or more validation findings. An empty list is omitted.',
      ),
    external_references: invoiceExternalReferencesWire.optional(),
  })

  .describe(
    'Base fields shared by all invoice types. Spread this model into each concrete invoice variant.',
  )

export const customerStripeCreateCheckoutSessionRequestWire = z
  .strictObject({
    stripe_options: appStripeCreateCheckoutSessionRequestOptionsWire,
  })

  .describe(
    'Request to create a Stripe Checkout Session for the customer. Checkout Sessions are used to collect payment method information from customers in a secure, Stripe-hosted interface. This integration uses setup mode to collect payment methods that can be charged later for subscription billing.',
  )

export const workflowCollectionSettingsWire = z
  .strictObject({
    alignment: workflowCollectionAlignmentWire.optional().default({
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

export const appPagePaginatedResponseWire = z
  .strictObject({
    data: z.array(appWire),
    meta: paginatedMetaWire,
  })
  .describe('Page paginated response.')

export const profileAppsWire = z
  .strictObject({
    tax: appWire,
    invoicing: appWire,
    payment: appWire,
  })
  .describe('Applications used by a billing profile.')

export const governanceQueryResponseWire = z
  .strictObject({
    data: z
      .array(governanceQueryResultWire)
      .describe('Access evaluation results, one entry per resolved customer.'),
    errors: z
      .array(governanceQueryErrorWire)
      .describe('Partial errors encountered while processing the request.'),
    meta: cursorMetaWire,
  })
  .describe('Response of the governance query.')

export const chargeFlatFeeWire = z
  .strictObject({
    id: ulidWire,
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
    labels: labelsWire.optional(),
    created_at: dateTimeWire,
    updated_at: dateTimeWire,
    deleted_at: dateTimeWire.optional(),
    type: z.literal('flat_fee').describe('The type of the charge.'),
    customer: billingCustomerReferenceWire,
    lifecycle_controller: lifecycleControllerWire,
    subscription: subscriptionReferenceWire.optional(),
    currency: currencyCodeWire,
    status: chargeStatusWire,
    invoice_at: dateTimeWire,
    service_period: closedPeriodWire,
    full_service_period: closedPeriodWire,
    billing_period: closedPeriodWire,
    advance_after: dateTimeWire.optional(),
    unique_reference_id: z
      .string()
      .optional()
      .describe('Unique reference ID of the charge.'),
    settlement_mode: settlementModeWire,
    tax_config: taxConfigWire.optional(),
    payment_term: pricePaymentTermWire,
    discounts: chargeFlatFeeDiscountsWire.optional(),
    feature_key: z
      .string()
      .optional()
      .describe('The feature associated with the charge, when applicable.'),
    proration_configuration: rateCardProrationConfigurationWire,
    amount_after_proration: currencyAmountWire,
    price: priceWire,
  })
  .describe('A flat fee charge for a customer.')

export const chargeUsageBasedWire = z
  .strictObject({
    id: ulidWire,
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
    labels: labelsWire.optional(),
    created_at: dateTimeWire,
    updated_at: dateTimeWire,
    deleted_at: dateTimeWire.optional(),
    type: z.literal('usage_based').describe('The type of the charge.'),
    customer: billingCustomerReferenceWire,
    lifecycle_controller: lifecycleControllerWire,
    subscription: subscriptionReferenceWire.optional(),
    currency: currencyCodeWire,
    status: chargeStatusWire,
    invoice_at: dateTimeWire,
    service_period: closedPeriodWire,
    full_service_period: closedPeriodWire,
    billing_period: closedPeriodWire,
    advance_after: dateTimeWire.optional(),
    unique_reference_id: z
      .string()
      .optional()
      .describe('Unique reference ID of the charge.'),
    settlement_mode: settlementModeWire,
    tax_config: taxConfigWire.optional(),
    discounts: rateCardDiscountsWire.optional(),
    feature_key: z.string().describe('The feature associated with the charge.'),
    totals: chargeTotalsWire,
    price: priceWire,
  })
  .describe('A usage-based charge for a customer.')

export const createChargeUsageBasedRequestWire = z
  .strictObject({
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
    labels: labelsWire.optional(),
    type: z.literal('usage_based').describe('The type of the charge.'),
    currency: currencyCodeWire,
    invoice_at: dateTimeWire,
    service_period: closedPeriodWire,
    unique_reference_id: z
      .string()
      .optional()
      .describe('Unique reference ID of the charge.'),
    settlement_mode: settlementModeWire,
    tax_config: taxConfigWire.optional(),
    discounts: rateCardDiscountsWire.optional(),
    feature_key: z.string().describe('The feature associated with the charge.'),
    price: priceWire,
    full_service_period: closedPeriodWire.optional(),
    billing_period: closedPeriodWire.optional(),
  })
  .describe('Usage-based charge create request.')

export const rateCardWire = z
  .strictObject({
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
    labels: labelsWire.optional(),
    key: resourceKeyWire,
    feature: featureReferenceWire.optional(),
    billing_cadence: iso8601DurationWire.optional(),
    price: priceWire,
    unit_config: unitConfigWire.optional(),
    payment_term: pricePaymentTermWire.optional().default('in_arrears'),
    commitments: spendCommitmentsWire.optional(),
    discounts: rateCardDiscountsWire.optional(),
    tax_config: rateCardTaxConfigWire.optional(),
    entitlement: rateCardEntitlementWire.optional(),
  })

  .describe(
    'A rate card defines the pricing and entitlement of a feature or service.',
  )

export const invoiceLineRateCardWire = z
  .strictObject({
    price: priceWire,
    tax_config: rateCardTaxConfigWire.optional(),
    feature_key: resourceKeyWire.optional(),
    discounts: rateCardDiscountsWire.optional(),
  })
  .describe('Rate card configuration snapshot for a usage-based invoice line.')

export const featurePagePaginatedResponseWire = z
  .strictObject({
    data: z.array(featureWire),
    meta: paginatedMetaWire,
  })
  .describe('Page paginated response.')

export const workflowWire = z
  .strictObject({
    collection: workflowCollectionSettingsWire.optional(),
    invoicing: workflowInvoicingSettingsWire.optional(),
    payment: workflowPaymentSettingsWire.optional(),
    tax: workflowTaxSettingsWire.optional(),
  })
  .describe('Billing workflow settings.')

export const chargeWire = z
  .discriminatedUnion('type', [chargeFlatFeeWire, chargeUsageBasedWire])
  .describe('Customer charge.')

export const createChargeRequestWire = z
  .discriminatedUnion('type', [
    createChargeFlatFeeRequestWire,
    createChargeUsageBasedRequestWire,
  ])
  .describe('Customer charge.')

export const subscriptionAddonRateCardWire = z
  .strictObject({
    rate_card: rateCardWire,
    affected_subscription_item_ids: z
      .array(ulidWire)

      .describe(
        'The IDs of the subscription items that this rate card belongs to.',
      ),
  })
  .describe('A rate card for a subscription add-on.')

export const planPhaseWire = z
  .strictObject({
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
    labels: labelsWire.optional(),
    key: resourceKeyWire,
    duration: iso8601DurationWire.optional(),
    rate_cards: z.array(rateCardWire).describe('The rate cards of the plan.'),
  })

  .describe(
    "The plan phase or pricing ramp allows changing a plan's rate cards over time as a subscription progresses.",
  )

export const addonWire = z
  .strictObject({
    id: ulidWire,
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
    labels: labelsWire.optional(),
    created_at: dateTimeWire,
    updated_at: dateTimeWire,
    deleted_at: dateTimeWire.optional(),
    key: resourceKeyWire,
    version: z
      .number()
      .int()
      .gte(1)
      .default(1)

      .describe(
        'Version of the add-on. Incremented when the add-on is updated.',
      ),
    instance_type: addonInstanceTypeWire,
    currency: billingCurrencyCodeWire,
    effective_from: dateTimeWire.optional(),
    effective_to: dateTimeWire.optional(),
    status: addonStatusWire,
    rate_cards: z.array(rateCardWire).describe('The rate cards of the add-on.'),
    validation_errors: z
      .array(productCatalogValidationErrorWire)
      .optional()
      .describe('List of validation errors.'),
  })

  .describe(
    'Add-on allows extending subscriptions with compatible plans with additional ratecards.',
  )

export const createAddonRequestWire = z
  .strictObject({
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
    labels: labelsWire.optional(),
    key: resourceKeyWire,
    instance_type: addonInstanceTypeWire,
    currency: billingCurrencyCodeWire,
    rate_cards: z.array(rateCardWire).describe('The rate cards of the add-on.'),
  })
  .describe('Addon create request.')

export const upsertAddonRequestWire = z
  .strictObject({
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
    labels: labelsWire.optional(),
    instance_type: addonInstanceTypeWire,
    rate_cards: z.array(rateCardWire).describe('The rate cards of the add-on.'),
  })
  .describe('Addon upsert request.')

export const invoiceStandardLineWire = z
  .strictObject({
    id: ulidWire,
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
    labels: labelsWire.optional(),
    created_at: dateTimeWire,
    updated_at: dateTimeWire,
    deleted_at: dateTimeWire.optional(),
    type: z
      .literal('standard_line')
      .describe('The type of charge this line item represents.'),
    lifecycle_controller: lifecycleControllerWire,
    service_period: closedPeriodWire,
    totals: totalsWire,
    discounts: invoiceLineDiscountsWire.optional(),
    credits_applied: z
      .array(invoiceLineCreditsAppliedWire)
      .optional()
      .describe('Credit applied to this line item.'),
    external_references: invoiceLineExternalReferencesWire.optional(),
    subscription: subscriptionReferenceWire.optional(),
    rate_card: invoiceLineRateCardWire,
    detailed_lines: z
      .array(invoiceDetailedLineWire)

      .describe(
        'Detailed sub-lines that this line has been broken down into. Present when line has individual details.',
      ),
    charge: chargeReferenceWire.optional(),
  })

  .describe(
    'A top-level line item on an invoice. Each line represents a single charge, typically associated with a rate card from a subscription. Detailed (child) lines are nested under `detailed_lines` when present.',
  )

export const profileWire = z
  .strictObject({
    id: ulidWire,
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
    labels: labelsWire.optional(),
    created_at: dateTimeWire,
    updated_at: dateTimeWire,
    deleted_at: dateTimeWire.optional(),
    supplier: partyWire,
    workflow: workflowWire,
    apps: profileAppReferencesWire,
    default: z.boolean().describe('Whether this is the default profile.'),
  })

  .describe(
    'Billing profiles contain the settings for billing and controls invoice generation.',
  )

export const createBillingProfileRequestWire = z
  .strictObject({
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
    labels: labelsWire.optional(),
    supplier: partyWire,
    workflow: workflowWire,
    apps: profileAppReferencesWire,
    default: z.boolean().describe('Whether this is the default profile.'),
  })
  .describe('BillingProfile create request.')

export const upsertBillingProfileRequestWire = z
  .strictObject({
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
    labels: labelsWire.optional(),
    supplier: partyWire,
    workflow: workflowWire,
    default: z.boolean().describe('Whether this is the default profile.'),
  })
  .describe('BillingProfile upsert request.')

export const chargePagePaginatedResponseWire = z
  .strictObject({
    data: z.array(chargeWire),
    meta: paginatedMetaWire,
  })
  .describe('Page paginated response.')

export const subscriptionAddonWire = z
  .strictObject({
    id: ulidWire,
    labels: labelsWire.optional(),
    created_at: dateTimeWire,
    updated_at: dateTimeWire,
    deleted_at: dateTimeWire.optional(),
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
    addon: addonReferenceWire,
    quantity: z
      .number()
      .int()
      .gte(1)

      .describe(
        'The quantity of the add-on. Always 1 for single instance add-ons.',
      ),
    quantity_at: dateTimeWire,
    active_from: dateTimeWire,
    active_to: dateTimeWire.optional(),
    timeline: z
      .array(subscriptionAddonTimelineSegmentWire)

      .describe(
        'The timeline of the add-on. The returned periods are sorted and continuous.',
      ),
    rate_cards: z
      .array(subscriptionAddonRateCardWire)
      .describe('The rate cards of the add-on.'),
  })
  .describe('Addon purchased with a subscription.')

export const planWire = z
  .strictObject({
    id: ulidWire,
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
    labels: labelsWire.optional(),
    created_at: dateTimeWire,
    updated_at: dateTimeWire,
    deleted_at: dateTimeWire.optional(),
    key: resourceKeyWire,
    version: z
      .number()
      .int()
      .gte(1)
      .default(1)

      .describe(
        'Plans are versioned to allow you to make changes without affecting running subscriptions.',
      ),
    currency: currencyCodeWire,
    billing_cadence: iso8601DurationWire,
    pro_rating_enabled: z
      .boolean()
      .optional()
      .default(true)
      .describe('Whether pro-rating is enabled for this plan.'),
    effective_from: dateTimeWire.optional(),
    effective_to: dateTimeWire.optional(),
    status: planStatusWire,
    phases: z
      .array(planPhaseWire)
      .min(1)

      .describe(
        'The plan phases define the pricing ramp for a subscription. A phase switch occurs only at the end of a billing period. At least one phase is required.',
      ),
    settlement_mode: settlementModeWire
      .optional()
      .default('credit_then_invoice'),
    validation_errors: z
      .array(productCatalogValidationErrorWire)
      .optional()

      .describe(
        'List of validation errors in `draft` state that prevent the plan from being published.',
      ),
  })
  .describe('Plans provide a template for subscriptions.')

export const createPlanRequestWire = z
  .strictObject({
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
    labels: labelsWire.optional(),
    key: resourceKeyWire,
    currency: currencyCodeWire,
    billing_cadence: iso8601DurationWire,
    pro_rating_enabled: z
      .boolean()
      .optional()
      .default(true)
      .describe('Whether pro-rating is enabled for this plan.'),
    phases: z
      .array(planPhaseWire)
      .min(1)

      .describe(
        'The plan phases define the pricing ramp for a subscription. A phase switch occurs only at the end of a billing period. At least one phase is required.',
      ),
  })
  .describe('Plan create request.')

export const upsertPlanRequestWire = z
  .strictObject({
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
    labels: labelsWire.optional(),
    pro_rating_enabled: z
      .boolean()
      .optional()
      .default(true)
      .describe('Whether pro-rating is enabled for this plan.'),
    phases: z
      .array(planPhaseWire)
      .min(1)

      .describe(
        'The plan phases define the pricing ramp for a subscription. A phase switch occurs only at the end of a billing period. At least one phase is required.',
      ),
  })
  .describe('Plan upsert request.')

export const addonPagePaginatedResponseWire = z
  .strictObject({
    data: z.array(addonWire),
    meta: paginatedMetaWire,
  })
  .describe('Page paginated response.')

export const invoiceLineWire = z
  .discriminatedUnion('type', [invoiceStandardLineWire])

  .describe(
    'A top-level line item on an invoice. Each line represents a single charge, typically associated with a rate card from a subscription. Detailed (child) lines are nested under `detailed_lines` when present.',
  )

export const profilePagePaginatedResponseWire = z
  .strictObject({
    data: z.array(profileWire),
    meta: paginatedMetaWire,
  })
  .describe('Page paginated response.')

export const subscriptionAddonPagePaginatedResponseWire = z
  .strictObject({
    data: z.array(subscriptionAddonWire),
    meta: paginatedMetaWire,
  })
  .describe('Page paginated response.')

export const planPagePaginatedResponseWire = z
  .strictObject({
    data: z.array(planWire),
    meta: paginatedMetaWire,
  })
  .describe('Page paginated response.')

export const invoiceStandardWire = z
  .strictObject({
    id: ulidWire,
    description: z
      .string()
      .max(1024)
      .optional()

      .describe(
        'Optional description of the resource. Maximum 1024 characters.',
      ),
    labels: labelsWire.optional(),
    created_at: dateTimeWire,
    updated_at: dateTimeWire,
    deleted_at: dateTimeWire.optional(),
    number: invoiceNumberWire,
    currency: currencyCodeWire,
    supplier: supplierWire,
    customer: invoiceCustomerWire,
    totals: totalsWire,
    service_period: closedPeriodWire,
    validation_issues: z
      .array(invoiceValidationIssueWire)
      .optional()

      .describe(
        'Validation issues found during invoice processing. Present only when there are one or more validation findings. An empty list is omitted.',
      ),
    external_references: invoiceExternalReferencesWire.optional(),
    type: z
      .literal('standard')
      .describe('Discriminator field identifying this as a standard invoice.'),
    status: invoiceStandardStatusWire,
    status_details: invoiceStatusDetailsWire,
    issued_at: dateTimeWire.optional(),
    draft_until: dateTimeWire.optional(),
    quantity_snapshotted_at: dateTimeWire.optional(),
    collection_at: dateTimeWire.optional(),
    due_at: dateTimeWire.optional(),
    sent_to_customer_at: dateTimeWire.optional(),
    workflow: invoiceWorkflowSettingsWire,
    lines: z
      .array(invoiceLineWire)
      .optional()

      .describe(
        'Line items on this invoice. Always returned on single-resource GET; omitted on list endpoints unless explicitly expanded.',
      ),
  })
  .describe('A standard invoice for charges owed by the customer.')

export const invoiceWire = z
  .discriminatedUnion('type', [invoiceStandardWire])

  .describe(
    'An invoice issued to a customer. The `type` field determines the concrete variant: - `standard`: a standard invoice for charges owed.',
  )

export const listMeteringEventsQueryParamsWire = z.object({
  page: cursorPaginationQueryPageWire.optional(),
  filter: listEventsParamsFilterWire.optional(),
  sort: sortQueryWire.optional(),
})

export const listMeteringEventsResponseWire = z.strictObject({
  data: z.array(ingestedEventWire),
  meta: cursorMetaWire,
})

export const ingestMeteringEventsBodyWire = z.union([
  eventWire,
  z.array(eventWire),
])

export const createMeterBodyWire = createMeterRequestWire

export const createMeterResponseWire = meterWire

export const getMeterPathParamsWire = z.object({
  meterId: ulidWire,
})

export const getMeterResponseWire = meterWire

export const listMetersQueryParamsWire = z.object({
  page: z
    .strictObject({
      size: z.coerce
        .number()
        .int()
        .optional()
        .describe('The number of items to include per page.'),
      number: z.coerce.number().int().optional().describe('The page number.'),
    })
    .optional()
    .describe('Determines which page of the collection to retrieve.'),
  sort: sortQueryWire.optional(),
  filter: listMetersParamsFilterWire.optional(),
})

export const listMetersResponseWire = z.strictObject({
  data: z.array(meterWire),
  meta: paginatedMetaWire,
})

export const updateMeterPathParamsWire = z.object({
  meterId: ulidWire,
})

export const updateMeterBodyWire = updateMeterRequestWire

export const updateMeterResponseWire = meterWire

export const deleteMeterPathParamsWire = z.object({
  meterId: ulidWire,
})

export const queryMeterPathParamsWire = z.object({
  meterId: ulidWire,
})

export const queryMeterBodyWire = meterQueryRequestWire

export const queryMeterResponseWire = meterQueryResultWire

export const queryMeterCsvPathParamsWire = z.object({
  meterId: ulidWire,
})

export const queryMeterCsvBodyWire = meterQueryRequestWire

export const queryMeterCsvResponseWire = z.string()

export const createCustomerBodyWire = createCustomerRequestWire

export const createCustomerResponseWire = customerWire

export const getCustomerPathParamsWire = z.object({
  customerId: ulidWire,
})

export const getCustomerResponseWire = customerWire

export const listCustomersQueryParamsWire = z.object({
  page: z
    .strictObject({
      size: z.coerce
        .number()
        .int()
        .optional()
        .describe('The number of items to include per page.'),
      number: z.coerce.number().int().optional().describe('The page number.'),
    })
    .optional()
    .describe('Determines which page of the collection to retrieve.'),
  sort: sortQueryWire.optional(),
  filter: listCustomersParamsFilterWire.optional(),
})

export const listCustomersResponseWire = z.strictObject({
  data: z.array(customerWire),
  meta: paginatedMetaWire,
})

export const upsertCustomerPathParamsWire = z.object({
  customerId: ulidWire,
})

export const upsertCustomerBodyWire = upsertCustomerRequestWire

export const upsertCustomerResponseWire = customerWire

export const deleteCustomerPathParamsWire = z.object({
  customerId: ulidWire,
})

export const getCustomerBillingPathParamsWire = z.object({
  customerId: ulidWire,
})

export const getCustomerBillingResponseWire = customerDataWire

export const updateCustomerBillingPathParamsWire = z.object({
  customerId: ulidWire,
})

export const updateCustomerBillingBodyWire =
  upsertCustomerBillingDataRequestWire

export const updateCustomerBillingResponseWire = customerDataWire

export const updateCustomerBillingAppDataPathParamsWire = z.object({
  customerId: ulidWire,
})

export const updateCustomerBillingAppDataBodyWire =
  upsertAppCustomerDataRequestWire

export const updateCustomerBillingAppDataResponseWire = appCustomerDataWire

export const createCustomerStripeCheckoutSessionPathParamsWire = z.object({
  customerId: ulidWire,
})

export const createCustomerStripeCheckoutSessionBodyWire =
  customerStripeCreateCheckoutSessionRequestWire

export const createCustomerStripeCheckoutSessionResponseWire =
  appStripeCreateCheckoutSessionResultWire

export const createCustomerStripePortalSessionPathParamsWire = z.object({
  customerId: ulidWire,
})

export const createCustomerStripePortalSessionBodyWire =
  customerStripeCreateCustomerPortalSessionRequestWire

export const createCustomerStripePortalSessionResponseWire =
  appStripeCreateCustomerPortalSessionResultWire

export const listCustomerEntitlementAccessPathParamsWire = z.object({
  customerId: ulidWire,
})

export const listCustomerEntitlementAccessResponseWire =
  listCustomerEntitlementAccessResponseDataWire

export const createCreditGrantPathParamsWire = z.object({
  customerId: ulidWire,
})

export const createCreditGrantBodyWire = createCreditGrantRequestWire

export const createCreditGrantResponseWire = creditGrantWire

export const getCreditGrantPathParamsWire = z.object({
  customerId: ulidWire,
  creditGrantId: ulidWire,
})

export const getCreditGrantResponseWire = creditGrantWire

export const listCreditGrantsPathParamsWire = z.object({
  customerId: ulidWire,
})

export const listCreditGrantsQueryParamsWire = z.object({
  page: z
    .strictObject({
      size: z.coerce
        .number()
        .int()
        .optional()
        .describe('The number of items to include per page.'),
      number: z.coerce.number().int().optional().describe('The page number.'),
    })
    .optional()
    .describe('Determines which page of the collection to retrieve.'),
  filter: listCreditGrantsParamsFilterWire.optional(),
})

export const listCreditGrantsResponseWire = z.strictObject({
  data: z.array(creditGrantWire),
  meta: paginatedMetaWire,
})

export const getCustomerCreditBalancePathParamsWire = z.object({
  customerId: ulidWire,
})

export const getCustomerCreditBalanceQueryParamsWire = z.object({
  timestamp: dateTimeWire.optional(),
  filter: getCreditBalanceParamsFilterWire.optional(),
})

export const getCustomerCreditBalanceResponseWire = creditBalancesWire

export const createCreditAdjustmentPathParamsWire = z.object({
  customerId: ulidWire,
})

export const createCreditAdjustmentBodyWire = createCreditAdjustmentRequestWire

export const createCreditAdjustmentResponseWire = creditAdjustmentWire

export const updateCreditGrantExternalSettlementPathParamsWire = z.object({
  customerId: ulidWire,
  creditGrantId: ulidWire,
})

export const updateCreditGrantExternalSettlementBodyWire =
  updateCreditGrantExternalSettlementRequestWire

export const updateCreditGrantExternalSettlementResponseWire = creditGrantWire

export const listCreditTransactionsPathParamsWire = z.object({
  customerId: ulidWire,
})

export const listCreditTransactionsQueryParamsWire = z.object({
  page: cursorPaginationQueryPageWire.optional(),
  filter: listCreditTransactionsParamsFilterWire.optional(),
})

export const listCreditTransactionsResponseWire = z.strictObject({
  data: z.array(creditTransactionWire),
  meta: cursorMetaWire,
})

export const listCustomerChargesPathParamsWire = z.object({
  customerId: ulidWire,
})

export const listCustomerChargesQueryParamsWire = z.object({
  page: z
    .strictObject({
      size: z.coerce
        .number()
        .int()
        .optional()
        .describe('The number of items to include per page.'),
      number: z.coerce.number().int().optional().describe('The page number.'),
    })
    .optional()
    .describe('Determines which page of the collection to retrieve.'),
  sort: sortQueryWire.optional(),
  filter: listChargesParamsFilterWire.optional(),
  expand: z
    .array(chargesExpandWire)
    .optional()

    .describe(
      "Expand full objects for referenced entities. Supported values are: - `real_time_usage`: Expand the charge's real-time usage.",
    ),
})

export const listCustomerChargesResponseWire = z.strictObject({
  data: z.array(chargeWire),
  meta: paginatedMetaWire,
})

export const createCustomerChargesPathParamsWire = z.object({
  customerId: ulidWire,
})

export const createCustomerChargesBodyWire = createChargeRequestWire

export const createCustomerChargesResponseWire = chargeWire

export const createSubscriptionBodyWire = subscriptionCreateWire

export const createSubscriptionResponseWire = subscriptionWire

export const listSubscriptionsQueryParamsWire = z.object({
  page: z
    .strictObject({
      size: z.coerce
        .number()
        .int()
        .optional()
        .describe('The number of items to include per page.'),
      number: z.coerce.number().int().optional().describe('The page number.'),
    })
    .optional()
    .describe('Determines which page of the collection to retrieve.'),
  sort: sortQueryWire.optional(),
  filter: listSubscriptionsParamsFilterWire.optional(),
})

export const listSubscriptionsResponseWire = z.strictObject({
  data: z.array(subscriptionWire),
  meta: paginatedMetaWire,
})

export const getSubscriptionPathParamsWire = z.object({
  subscriptionId: ulidWire,
})

export const getSubscriptionResponseWire = subscriptionWire

export const cancelSubscriptionPathParamsWire = z.object({
  subscriptionId: ulidWire,
})

export const cancelSubscriptionBodyWire = subscriptionCancelWire

export const cancelSubscriptionResponseWire = subscriptionWire

export const unscheduleCancelationPathParamsWire = z.object({
  subscriptionId: ulidWire,
})

export const unscheduleCancelationResponseWire = subscriptionWire

export const changeSubscriptionPathParamsWire = z.object({
  subscriptionId: ulidWire,
})

export const changeSubscriptionBodyWire = subscriptionChangeWire

export const changeSubscriptionResponseWire = subscriptionChangeResponseWire

export const createSubscriptionAddonPathParamsWire = z.object({
  subscriptionId: ulidWire,
})

export const createSubscriptionAddonBodyWire =
  createSubscriptionAddonRequestWire

export const createSubscriptionAddonResponseWire = subscriptionAddonWire

export const listSubscriptionAddonsPathParamsWire = z.object({
  subscriptionId: ulidWire,
})

export const listSubscriptionAddonsQueryParamsWire = z.object({
  page: z
    .strictObject({
      size: z.coerce
        .number()
        .int()
        .optional()
        .describe('The number of items to include per page.'),
      number: z.coerce.number().int().optional().describe('The page number.'),
    })
    .optional()
    .describe('Determines which page of the collection to retrieve.'),
  sort: sortQueryWire.optional(),
})

export const listSubscriptionAddonsResponseWire = z.strictObject({
  data: z.array(subscriptionAddonWire),
  meta: paginatedMetaWire,
})

export const getSubscriptionAddonPathParamsWire = z.object({
  subscriptionId: ulidWire,
  subscriptionAddonId: ulidWire,
})

export const getSubscriptionAddonResponseWire = subscriptionAddonWire

export const listAppsQueryParamsWire = z.object({
  page: z
    .strictObject({
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

export const listAppsResponseWire = z.strictObject({
  data: z.array(appWire),
  meta: paginatedMetaWire,
})

export const getAppPathParamsWire = z.object({
  appId: ulidWire,
})

export const getAppResponseWire = appWire

export const listBillingProfilesQueryParamsWire = z.object({
  page: z
    .strictObject({
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

export const listBillingProfilesResponseWire = z.strictObject({
  data: z.array(profileWire),
  meta: paginatedMetaWire,
})

export const createBillingProfileBodyWire = createBillingProfileRequestWire

export const createBillingProfileResponseWire = profileWire

export const getBillingProfilePathParamsWire = z.object({
  id: ulidWire,
})

export const getBillingProfileResponseWire = profileWire

export const updateBillingProfilePathParamsWire = z.object({
  id: ulidWire,
})

export const updateBillingProfileBodyWire = upsertBillingProfileRequestWire

export const updateBillingProfileResponseWire = profileWire

export const deleteBillingProfilePathParamsWire = z.object({
  id: ulidWire,
})

export const getInvoicePathParamsWire = z.object({
  invoiceId: ulidWire,
})

export const getInvoiceResponseWire = invoiceWire

export const createTaxCodeBodyWire = createTaxCodeRequestWire

export const createTaxCodeResponseWire = taxCodeWire

export const getTaxCodePathParamsWire = z.object({
  taxCodeId: ulidWire,
})

export const getTaxCodeResponseWire = taxCodeWire

export const listTaxCodesQueryParamsWire = z.object({
  page: z
    .strictObject({
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

export const listTaxCodesResponseWire = z.strictObject({
  data: z.array(taxCodeWire),
  meta: paginatedMetaWire,
})

export const upsertTaxCodePathParamsWire = z.object({
  taxCodeId: ulidWire,
})

export const upsertTaxCodeBodyWire = upsertTaxCodeRequestWire

export const upsertTaxCodeResponseWire = taxCodeWire

export const deleteTaxCodePathParamsWire = z.object({
  taxCodeId: ulidWire,
})

export const listCurrenciesQueryParamsWire = z.object({
  page: z
    .strictObject({
      size: z.coerce
        .number()
        .int()
        .optional()
        .describe('The number of items to include per page.'),
      number: z.coerce.number().int().optional().describe('The page number.'),
    })
    .optional()
    .describe('Determines which page of the collection to retrieve.'),
  sort: sortQueryWire.optional(),
  filter: listCurrenciesParamsFilterWire.optional(),
})

export const listCurrenciesResponseWire = z.strictObject({
  data: z.array(currencyWire),
  meta: paginatedMetaWire,
})

export const createCustomCurrencyBodyWire = createCurrencyCustomRequestWire

export const createCustomCurrencyResponseWire = currencyCustomWire

export const listCostBasesPathParamsWire = z.object({
  currencyId: ulidWire,
})

export const listCostBasesQueryParamsWire = z.object({
  filter: listCostBasesParamsFilterWire.optional(),
  page: z
    .strictObject({
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

export const listCostBasesResponseWire = z.strictObject({
  data: z.array(costBasisWire),
  meta: paginatedMetaWire,
})

export const createCostBasisPathParamsWire = z.object({
  currencyId: ulidWire,
})

export const createCostBasisBodyWire = createCostBasisRequestWire

export const createCostBasisResponseWire = costBasisWire

export const listFeaturesQueryParamsWire = z.object({
  page: z
    .strictObject({
      size: z.coerce
        .number()
        .int()
        .optional()
        .describe('The number of items to include per page.'),
      number: z.coerce.number().int().optional().describe('The page number.'),
    })
    .optional()
    .describe('Determines which page of the collection to retrieve.'),
  sort: sortQueryWire.optional(),
  filter: listFeatureParamsFilterWire.optional(),
})

export const listFeaturesResponseWire = z.strictObject({
  data: z.array(featureWire),
  meta: paginatedMetaWire,
})

export const createFeatureBodyWire = createFeatureRequestWire

export const createFeatureResponseWire = featureWire

export const getFeaturePathParamsWire = z.object({
  featureId: ulidWire,
})

export const getFeatureResponseWire = featureWire

export const updateFeaturePathParamsWire = z.object({
  featureId: ulidWire,
})

export const updateFeatureBodyWire = updateFeatureRequestWire

export const updateFeatureResponseWire = featureWire

export const deleteFeaturePathParamsWire = z.object({
  featureId: ulidWire,
})

export const queryFeatureCostPathParamsWire = z.object({
  featureId: ulidWire,
})

export const queryFeatureCostBodyWire = meterQueryRequestWire

export const queryFeatureCostResponseWire = featureCostQueryResultWire

export const listLlmCostPricesQueryParamsWire = z.object({
  filter: listLlmCostPricesParamsFilterWire.optional(),
  sort: sortQueryWire.optional(),
  page: z
    .strictObject({
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

export const listLlmCostPricesResponseWire = z.strictObject({
  data: z.array(llmCostPriceWire),
  meta: paginatedMetaWire,
})

export const getLlmCostPricePathParamsWire = z.object({
  priceId: ulidWire,
})

export const getLlmCostPriceResponseWire = llmCostPriceWire

export const listLlmCostOverridesQueryParamsWire = z.object({
  filter: listLlmCostPricesParamsFilterWire.optional(),
  page: z
    .strictObject({
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

export const listLlmCostOverridesResponseWire = z.strictObject({
  data: z.array(llmCostPriceWire),
  meta: paginatedMetaWire,
})

export const createLlmCostOverrideBodyWire = llmCostOverrideCreateWire

export const createLlmCostOverrideResponseWire = llmCostPriceWire

export const deleteLlmCostOverridePathParamsWire = z.object({
  priceId: ulidWire,
})

export const listPlansQueryParamsWire = z.object({
  page: z
    .strictObject({
      size: z.coerce
        .number()
        .int()
        .optional()
        .describe('The number of items to include per page.'),
      number: z.coerce.number().int().optional().describe('The page number.'),
    })
    .optional()
    .describe('Determines which page of the collection to retrieve.'),
  sort: sortQueryWire.optional(),
  filter: listPlansParamsFilterWire.optional(),
})

export const listPlansResponseWire = z.strictObject({
  data: z.array(planWire),
  meta: paginatedMetaWire,
})

export const createPlanBodyWire = createPlanRequestWire

export const createPlanResponseWire = planWire

export const updatePlanPathParamsWire = z.object({
  planId: ulidWire,
})

export const updatePlanBodyWire = upsertPlanRequestWire

export const updatePlanResponseWire = planWire

export const getPlanPathParamsWire = z.object({
  planId: ulidWire,
})

export const getPlanResponseWire = planWire

export const deletePlanPathParamsWire = z.object({
  planId: ulidWire,
})

export const archivePlanPathParamsWire = z.object({
  planId: ulidWire,
})

export const archivePlanResponseWire = planWire

export const publishPlanPathParamsWire = z.object({
  planId: ulidWire,
})

export const publishPlanResponseWire = planWire

export const listAddonsQueryParamsWire = z.object({
  page: z
    .strictObject({
      size: z.coerce
        .number()
        .int()
        .optional()
        .describe('The number of items to include per page.'),
      number: z.coerce.number().int().optional().describe('The page number.'),
    })
    .optional()
    .describe('Determines which page of the collection to retrieve.'),
  sort: sortQueryWire.optional(),
  filter: listAddonsParamsFilterWire.optional(),
})

export const listAddonsResponseWire = z.strictObject({
  data: z.array(addonWire),
  meta: paginatedMetaWire,
})

export const createAddonBodyWire = createAddonRequestWire

export const createAddonResponseWire = addonWire

export const updateAddonPathParamsWire = z.object({
  addonId: ulidWire,
})

export const updateAddonBodyWire = upsertAddonRequestWire

export const updateAddonResponseWire = addonWire

export const getAddonPathParamsWire = z.object({
  addonId: ulidWire,
})

export const getAddonResponseWire = addonWire

export const deleteAddonPathParamsWire = z.object({
  addonId: ulidWire,
})

export const archiveAddonPathParamsWire = z.object({
  addonId: ulidWire,
})

export const archiveAddonResponseWire = addonWire

export const publishAddonPathParamsWire = z.object({
  addonId: ulidWire,
})

export const publishAddonResponseWire = addonWire

export const listPlanAddonsPathParamsWire = z.object({
  planId: ulidWire,
})

export const listPlanAddonsQueryParamsWire = z.object({
  page: z
    .strictObject({
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

export const listPlanAddonsResponseWire = z.strictObject({
  data: z.array(planAddonWire),
  meta: paginatedMetaWire,
})

export const createPlanAddonPathParamsWire = z.object({
  planId: ulidWire,
})

export const createPlanAddonBodyWire = createPlanAddonRequestWire

export const createPlanAddonResponseWire = planAddonWire

export const getPlanAddonPathParamsWire = z.object({
  planId: ulidWire,
  planAddonId: ulidWire,
})

export const getPlanAddonResponseWire = planAddonWire

export const updatePlanAddonPathParamsWire = z.object({
  planId: ulidWire,
  planAddonId: ulidWire,
})

export const updatePlanAddonBodyWire = upsertPlanAddonRequestWire

export const updatePlanAddonResponseWire = planAddonWire

export const deletePlanAddonPathParamsWire = z.object({
  planId: ulidWire,
  planAddonId: ulidWire,
})

export const getOrganizationDefaultTaxCodesResponseWire =
  organizationDefaultTaxCodesWire

export const updateOrganizationDefaultTaxCodesBodyWire =
  updateOrganizationDefaultTaxCodesRequestWire

export const updateOrganizationDefaultTaxCodesResponseWire =
  organizationDefaultTaxCodesWire

export const queryGovernanceAccessQueryParamsWire = z.object({
  page: cursorPaginationQueryPageWire.optional(),
})

export const queryGovernanceAccessBodyWire = governanceQueryRequestWire

export const queryGovernanceAccessResponseWire = governanceQueryResponseWire
